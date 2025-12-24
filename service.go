// Package main provides Windows service integration for background device monitoring and automatic repair.
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

const serviceName = "GPDTouchFix"

type gpdTouchService struct {
	cfg          *Config
	logger       *Logger
	stats        *StatsManager
	notifier     *Notifier
	poller       *WakeEventPoller
	powerMonitor *PowerMonitor
}

func (s *gpdTouchService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue | svc.AcceptPowerEvent
	changes <- svc.Status{State: svc.StartPending}

	// 设置事件日志
	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return
	}
	defer elog.Close()

	elog.Info(1, "服务已启动")
	s.logger.InfoTag(TagService, "服务已启动")

	// 检查是否是 Modern Standby 系统
	sleepState := GetSystemSleepState()
	s.logger.InfoTag(TagService, "系统睡眠状态: %s", sleepState)
	elog.Info(1, fmt.Sprintf("系统睡眠状态: %s", sleepState))

	// 如果是 Modern Standby，启动轮询器作为备用检测方案
	if IsModernStandbySupported() {
		s.logger.InfoTag(TagService, "检测到 Modern Standby，启动设备状态轮询")
		s.startPolling(elog)

		// 同时启动电源监控器来监听显示器状态变化
		s.powerMonitor = NewPowerMonitor(func() {
			s.logger.InfoTag(TagService, "检测到显示器唤醒事件")
			if s.poller != nil {
				s.poller.Resume()
			}
		})
		s.logger.InfoTag(TagService, "电源监控器已启动，监听显示器状态变化")
	}

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// 主循环
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				elog.Info(1, "服务正在停止")
				s.logger.InfoTag(TagService, "服务正在停止")
				// 停止轮询器
				if s.poller != nil {
					s.poller.Stop()
				}
				// 停止电源监控器
				if s.powerMonitor != nil {
					// PowerMonitor 没有显式停止方法，它会在程序退出时自动清理
					s.logger.InfoTag(TagService, "电源监控器已停止")
				}
				changes <- svc.Status{State: svc.StopPending}
				return
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			case svc.PowerEvent:
				// 电源事件处理
				s.handlePowerEvent(elog, c.EventType)
			default:
				elog.Error(1, fmt.Sprintf("未处理的服务命令: %v", c.Cmd))
			}
		}
	}
}

// handlePowerEvent 处理电源事件
func (s *gpdTouchService) handlePowerEvent(elog *eventlog.Log, eventType uint32) {
	// 记录所有电源事件，方便调试
	eventName := s.getPowerEventName(eventType)
	s.logger.InfoTag(TagService, "收到电源事件: %s (类型: %d/0x%X)", eventName, eventType, eventType)

	// 如果有电源监控器，让它也处理这个事件
	if s.powerMonitor != nil {
		// 注意：这里假设 eventData 为 0，因为 Windows 服务 API 传递的是简化的事件
		// PowerMonitor 会处理传统电源事件和显示器状态变化
		if s.powerMonitor.HandlePowerBroadcast(eventType, 0) {
			// PowerMonitor 已处理，返回
			return
		}
	}

	// 处理唤醒相关事件:
	// - ResumeAutomatic (18): 自动恢复
	// - ResumeSuspend (7): 从挂起恢复
	// - OEM事件 (10): 可能是特定硬件的唤醒信号
	isResumeEvent := (eventType == 18 || eventType == 7)
	isOemEvent := (eventType == 10)

	if !isResumeEvent && !isOemEvent {
		return
	}

	// 对于OEM事件，如果有轮询器，触发它立即检测而不是自己处理
	if isOemEvent {
		if s.poller != nil {
			// 恢复轮询器（如果被暂停），并重置其重试状态
			s.poller.Resume()
			s.poller.ResetRetryState()
			s.logger.InfoTag(TagService, "收到OEM事件，已触发设备状态检测")
		}
		return
	}

	// 确认的唤醒事件，执行完整修复流程
	elog.Info(1, fmt.Sprintf("检测到系统从睡眠恢复 (事件: %s)", eventName))
	s.logger.InfoTag(TagResume, "系统从睡眠唤醒 (事件类型: %s)", eventName)

	// 记录唤醒事件
	s.stats.RecordResume()

	// 等待系统稳定
	delaySeconds := s.cfg.ResumeDelaySeconds
	if delaySeconds <= 0 {
		delaySeconds = 3
	}
	s.logger.InfoTag(TagResume, "等待系统稳定 (%d 秒)...", delaySeconds)
	time.Sleep(time.Duration(delaySeconds) * time.Second)

	// 创建设备管理器
	dm := NewDeviceManager(s.cfg.DeviceInstanceID)
	deviceName := s.cfg.DeviceName
	if deviceName == "" {
		deviceName = s.cfg.DeviceInstanceID
	}

	// 检查设备状态（如果启用了先检查再修复）
	if s.cfg.CheckBeforeReset {
		status, err := dm.GetStatus()
		if err != nil {
			s.logger.ErrorTag(TagCheck, "获取设备状态失败: %v", err)
			elog.Error(1, fmt.Sprintf("获取设备状态失败: %v", err))
			// 获取状态失败，尝试修复
		} else {
			s.logger.InfoTag(TagCheck, "设备状态: %s", status)

			// 如果状态正常，跳过修复
			if strings.EqualFold(status, "OK") {
				s.logger.InfoTag(TagSkip, "设备状态正常，无需修复")
				elog.Info(1, "设备状态正常，跳过修复")

				// 记录跳过
				s.stats.RecordSkip()

				// 发送通知（如果启用且记录所有事件）
				if s.cfg.LogAllEvents {
					s.notifier.NotifyResumeResult(false, true, deviceName, nil)
				}

				return
			}

			s.logger.WarningTag(TagCheck, "设备状态异常 (%s)，需要修复", status)
		}
	}

	// 执行设备重置
	s.logger.InfoTag(TagReset, "开始修复设备: %s", deviceName)
	elog.Info(1, fmt.Sprintf("开始修复设备: %s", deviceName))

	waitDuration := time.Duration(s.cfg.WaitSeconds) * time.Second
	if err := dm.Reset(waitDuration); err != nil {
		s.logger.ErrorTag(TagFail, "设备修复失败: %v", err)
		elog.Error(1, fmt.Sprintf("设备重置失败: %v", err))

		// 记录失败
		s.stats.RecordReset(false, fmt.Sprintf("失败: %v", err))

		// 发送失败通知
		s.notifier.NotifyResumeResult(false, false, deviceName, err)
		return
	}

	// 验证修复结果
	finalStatus, err := dm.GetStatus()
	if err != nil {
		s.logger.WarningTag(TagCheck, "无法验证修复结果: %v", err)
		s.stats.RecordReset(false, "无法验证修复结果")
		return
	}

	s.logger.InfoTag(TagCheck, "修复后设备状态: %s", finalStatus)

	// 根据实际状态判断是否成功
	if strings.EqualFold(finalStatus, "OK") {
		s.logger.InfoTag(TagSuccess, "触屏设备修复成功")
		elog.Info(1, "触屏设备修复成功")
		s.stats.RecordReset(true, "修复成功")
		s.notifier.NotifyResumeResult(true, false, deviceName, nil)
	} else {
		s.logger.WarningTag(TagFail, "修复后设备仍处于异常状态: %s", finalStatus)
		elog.Warning(1, fmt.Sprintf("修复后设备仍处于异常状态: %s", finalStatus))
		s.stats.RecordReset(false, fmt.Sprintf("修复后状态: %s", finalStatus))
		s.notifier.NotifyResumeResult(false, false, deviceName, fmt.Errorf("设备状态: %s", finalStatus))
	}
}

// startPolling 启动设备状态轮询（用于 Modern Standby 系统）
func (s *gpdTouchService) startPolling(elog *eventlog.Log) {
	// 配置轮询器参数
	pollerCfg := &PollerConfig{
		BaseRetryInterval: time.Duration(s.cfg.RetryIntervalSecs) * time.Second,
		MaxRetryInterval:  time.Duration(s.cfg.MaxRetryInterval) * time.Second,
		MaxRetryCount:     s.cfg.MaxRetryCount,
	}
	if pollerCfg.BaseRetryInterval <= 0 {
		pollerCfg.BaseRetryInterval = 60 * time.Second
	}
	if pollerCfg.MaxRetryInterval <= 0 {
		pollerCfg.MaxRetryInterval = 10 * time.Minute
	}

	s.poller = NewWakeEventPoller(s.cfg.DeviceInstanceID, func() bool {
		return s.handlePolledWake(elog)
	}, s.logger, pollerCfg)
	s.poller.Start()
	s.logger.InfoTag(TagService, "设备状态轮询已启动 (间隔: 10秒)")
	elog.Info(1, "Modern Standby 设备状态轮询已启动")
}

// handlePolledWake 处理轮询检测到的唤醒/设备错误事件
// 返回 true 表示修复成功，false 表示失败
func (s *gpdTouchService) handlePolledWake(elog *eventlog.Log) bool {
	s.logger.InfoTag(TagResume, "轮询检测到设备异常，开始修复")
	elog.Info(1, "轮询检测到设备异常，开始修复")

	// 记录唤醒事件
	s.stats.RecordResume()

	// 等待系统稳定
	delaySeconds := s.cfg.ResumeDelaySeconds
	if delaySeconds <= 0 {
		delaySeconds = 3
	}
	s.logger.InfoTag(TagResume, "等待系统稳定 (%d 秒)...", delaySeconds)
	time.Sleep(time.Duration(delaySeconds) * time.Second)

	// 创建设备管理器
	dm := NewDeviceManager(s.cfg.DeviceInstanceID)
	deviceName := s.cfg.DeviceName
	if deviceName == "" {
		deviceName = s.cfg.DeviceInstanceID
	}

	// 再次检查状态，可能在等待期间已经恢复
	status, err := dm.GetStatus()
	if err == nil && strings.EqualFold(status, "OK") {
		s.logger.InfoTag(TagSkip, "等待后设备状态已恢复正常，跳过修复")
		elog.Info(1, "等待后设备状态已恢复正常，跳过修复")
		s.stats.RecordSkip()
		return true // 设备已正常，视为成功
	}

	// 执行设备重置
	s.logger.InfoTag(TagReset, "开始修复设备: %s", deviceName)
	elog.Info(1, fmt.Sprintf("开始修复设备: %s", deviceName))

	waitDuration := time.Duration(s.cfg.WaitSeconds) * time.Second
	if err := dm.Reset(waitDuration); err != nil {
		s.logger.ErrorTag(TagFail, "设备修复失败: %v", err)
		elog.Error(1, fmt.Sprintf("设备重置失败: %v", err))
		s.stats.RecordReset(false, fmt.Sprintf("失败: %v", err))
		s.notifier.NotifyResumeResult(false, false, deviceName, err)
		return false
	}

	// 验证修复结果
	finalStatus, err := dm.GetStatus()
	if err != nil {
		s.logger.WarningTag(TagCheck, "无法验证修复结果: %v", err)
		// 无法验证，不确定是否成功，视为失败
		s.stats.RecordReset(false, "无法验证修复结果")
		return false
	}

	s.logger.InfoTag(TagCheck, "修复后设备状态: %s", finalStatus)

	// 根据实际状态判断是否成功
	if strings.EqualFold(finalStatus, "OK") {
		s.logger.InfoTag(TagSuccess, "触屏设备修复成功")
		elog.Info(1, "触屏设备修复成功")
		s.stats.RecordReset(true, "修复成功")
		s.notifier.NotifyResumeResult(true, false, deviceName, nil)
		return true
	}

	// 修复后设备仍处于错误状态
	s.logger.WarningTag(TagFail, "修复后设备仍处于异常状态: %s", finalStatus)
	elog.Warning(1, fmt.Sprintf("修复后设备仍处于异常状态: %s", finalStatus))
	s.stats.RecordReset(false, fmt.Sprintf("修复后状态: %s", finalStatus))
	s.notifier.NotifyResumeResult(false, false, deviceName, fmt.Errorf("设备状态: %s", finalStatus))
	return false
}

func runService() error {
	// 加载配置
	cfgPath := GetConfigPath()
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 初始化日志
	logDir := cfg.LogDir
	if logDir == "" {
		logDir = GetLogDir()
	}
	if err := InitLogger(logDir, INFO); err != nil {
		log.Printf("警告: 初始化日志失败: %v", err)
	}
	logger := GetLogger()

	// 清理过期日志
	if cfg.MaxLogDays > 0 {
		CleanOldLogs(cfg.MaxLogDays)
	}

	// 初始化统计
	stats := NewStatsManager(GetStatsDir())

	// 初始化通知
	notifier := NewNotifier(cfg.EnableNotification)

	// 运行服务
	return svc.Run(serviceName, &gpdTouchService{
		cfg:      cfg,
		logger:   logger,
		stats:    stats,
		notifier: notifier,
	})
}

func installService() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	// 直接调用 sc.exe，避免 PowerShell 参数解析问题
	binPath := fmt.Sprintf(`"%s" -service`, exePath)
	cmd := exec.Command("sc.exe", "create", serviceName,
		"binPath=", binPath,
		"start=", "auto",
		"DisplayName=", "GPD Touch Fix Service")

	log.Printf("执行命令: sc.exe create %s binPath= %q start= auto DisplayName= \"GPD Touch Fix Service\"", serviceName, binPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("安装服务失败: %s", strings.TrimSpace(string(output)))
	}

	log.Println("服务安装成功")

	// 创建事件日志源
	err = eventlog.InstallAsEventCreate(serviceName, eventlog.Info|eventlog.Warning|eventlog.Error)
	if err != nil {
		log.Printf("警告: 创建事件日志源失败: %v", err)
	}

	return nil
}

func uninstallService() error {
	// 停止服务
	stopCmd := fmt.Sprintf(`sc.exe stop "%s"`, serviceName)
	runPowerShell(stopCmd) // 忽略错误，服务可能未运行

	// 删除服务
	deleteCmd := fmt.Sprintf(`sc.exe delete "%s"`, serviceName)
	output, err := runPowerShell(deleteCmd)
	if err != nil {
		return fmt.Errorf("删除服务失败: %w\n输出: %s", err, output)
	}

	log.Println("服务已卸载")

	// 删除事件日志源
	err = eventlog.Remove(serviceName)
	if err != nil {
		log.Printf("警告: 删除事件日志源失败: %v", err)
	}

	return nil
}

func startService() error {
	cmd := fmt.Sprintf(`sc.exe start "%s"`, serviceName)
	output, err := runPowerShell(cmd)
	if err != nil {
		return fmt.Errorf("启动服务失败: %w\n输出: %s", err, output)
	}
	log.Println("服务已启动")
	return nil
}

func stopService() error {
	cmd := fmt.Sprintf(`sc.exe stop "%s"`, serviceName)
	output, err := runPowerShell(cmd)
	if err != nil {
		return fmt.Errorf("停止服务失败: %w\n输出: %s", err, output)
	}
	log.Println("服务已停止")
	return nil
}

func isService() bool {
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("无法确定会话类型: %v", err)
	}
	return !isIntSess
}

// GetConfigPath 在服务模式下返回可执行文件目录的配置路径
func GetServiceConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	dir := filepath.Dir(exe)
	return filepath.Join(dir, "config.json")
}

// getPowerEventName 返回电源事件的可读名称
func (s *gpdTouchService) getPowerEventName(eventType uint32) string {
	// Windows 电源事件类型
	// https://docs.microsoft.com/en-us/windows/win32/power/power-management-events
	switch eventType {
	case 4: // PBT_APMPOWERSTATUSCHANGE
		return "电源状态改变"
	case 5: // PBT_APMRESUMEAUTOMATIC (旧版)
		return "自动恢复(旧版)"
	case 6: // PBT_APMRESUMECRITICAL (已废弃)
		return "关键恢复(已废弃)"
	case 7: // PBT_APMRESUMESUSPEND
		return "从挂起恢复(盖子打开/电源按钮)"
	case 9: // PBT_APMSUSPEND
		return "系统挂起(盖子关闭/睡眠)"
	case 10: // PBT_APMOEMEVENT
		return "OEM事件"
	case 18: // PBT_APMRESUMEAUTOMATIC
		return "自动恢复(唤醒)"
	case 0x8013: // PBT_POWERSETTINGCHANGE
		return "电源设置变化(显示器/盖子状态)"
	default:
		return fmt.Sprintf("未知事件(%d)", eventType)
	}
}
