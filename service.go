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
	cfg      *Config
	logger   *Logger
	stats    *StatsManager
	notifier *Notifier
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
	// 如果是从睡眠恢复 (ResumeAutomatic = 18, ResumeSuspend = 7)
	if eventType != 18 && eventType != 7 {
		return
	}

	eventName := "ResumeAutomatic"
	if eventType == 7 {
		eventName = "ResumeSuspend"
	}

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
	} else {
		s.logger.InfoTag(TagCheck, "修复后设备状态: %s", finalStatus)
	}

	s.logger.InfoTag(TagSuccess, "触屏设备修复成功")
	elog.Info(1, "触屏设备修复成功")

	// 记录成功
	s.stats.RecordReset(true, "修复成功")

	// 发送成功通知
	s.notifier.NotifyResumeResult(true, false, deviceName, nil)
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
