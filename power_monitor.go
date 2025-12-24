// Package main provides power state monitoring for Modern Standby (S0 Low Power Idle).
// This is necessary because traditional power events may not be triggered on Modern Standby systems.
package main

import (
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// GUID_CONSOLE_DISPLAY_STATE - 控制台显示状态变化
// {6fe69556-704a-47a0-8f24-c28d936fda47}
var GUID_CONSOLE_DISPLAY_STATE = windows.GUID{
	Data1: 0x6fe69556,
	Data2: 0x704a,
	Data3: 0x47a0,
	Data4: [8]byte{0x8f, 0x24, 0xc2, 0x8d, 0x93, 0x6f, 0xda, 0x47},
}

// GUID_MONITOR_POWER_ON - 显示器电源状态
// {02731015-4510-4526-99e6-e5a17ebd1aea}
var GUID_MONITOR_POWER_ON = windows.GUID{
	Data1: 0x02731015,
	Data2: 0x4510,
	Data3: 0x4526,
	Data4: [8]byte{0x99, 0xe6, 0xe5, 0xa1, 0x7e, 0xbd, 0x1a, 0xea},
}

// GUID_SYSTEM_AWAYMODE - 离开模式状态
// {98a7f580-01f7-48aa-9c0f-44352c29e5c0}
var GUID_SYSTEM_AWAYMODE = windows.GUID{
	Data1: 0x98a7f580,
	Data2: 0x01f7,
	Data3: 0x48aa,
	Data4: [8]byte{0x9c, 0x0f, 0x44, 0x35, 0x2c, 0x29, 0xe5, 0xc0},
}

const (
	// 显示器状态值
	DisplayStateOff    = 0 // 显示器关闭
	DisplayStateDimmed = 1 // 显示器调暗
	DisplayStateOn     = 2 // 显示器开启
)

const (
	// Power setting notification registration type
	DEVICE_NOTIFY_SERVICE_HANDLE = 1
)

var (
	powrprof                               = windows.NewLazySystemDLL("powrprof.dll")
	procPowerSettingRegisterNotification   = powrprof.NewProc("PowerSettingRegisterNotification")
	procPowerSettingUnregisterNotification = powrprof.NewProc("PowerSettingUnregisterNotification")
	procPowerReadACValue                   = powrprof.NewProc("PowerReadACValue")

	user32                                 = windows.NewLazySystemDLL("user32.dll")
	procRegisterPowerSettingNotification   = user32.NewProc("RegisterPowerSettingNotification")
	procUnregisterPowerSettingNotification = user32.NewProc("UnregisterPowerSettingNotification")
	procGetLastInputInfo                   = user32.NewProc("GetLastInputInfo")

	kernel32Power      = windows.NewLazySystemDLL("kernel32.dll")
	procGetTickCount64 = kernel32Power.NewProc("GetTickCount64")
)

// PowerBroadcastSetting 结构体用于解析电源广播设置
type PowerBroadcastSetting struct {
	PowerSetting windows.GUID
	DataLength   uint32
	Data         [1]byte // 实际数据长度由 DataLength 决定
}

// PowerMonitor 监控电源状态变化
type PowerMonitor struct {
	callback         func()        // 唤醒时的回调函数
	lastDisplayState int           // 上次显示器状态
	lastResumeTime   time.Time     // 上次唤醒时间
	mu               sync.Mutex    // 保护并发访问
	minInterval      time.Duration // 最小触发间隔，防止重复触发
}

// NewPowerMonitor 创建电源监控器
func NewPowerMonitor(callback func()) *PowerMonitor {
	return &PowerMonitor{
		callback:         callback,
		lastDisplayState: DisplayStateOn, // 初始假设显示器开启
		minInterval:      5 * time.Second,
	}
}

// HandlePowerBroadcast 处理电源广播消息
// 返回 true 表示这是一个唤醒事件
func (pm *PowerMonitor) HandlePowerBroadcast(eventType uint32, eventData uintptr) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// PBT_POWERSETTINGCHANGE = 0x8013
	if eventType != 0x8013 {
		// 处理传统电源事件
		switch eventType {
		case 7, 18: // PBT_APMRESUMESUSPEND = 7, PBT_APMRESUMEAUTOMATIC = 18
			return pm.triggerResume("传统电源事件")
		}
		return false
	}

	// 解析 PowerBroadcastSetting
	if eventData == 0 {
		return false
	}

	pbs := (*PowerBroadcastSetting)(unsafe.Pointer(eventData))

	// 检查是否是显示器状态变化
	if pbs.PowerSetting == GUID_CONSOLE_DISPLAY_STATE || pbs.PowerSetting == GUID_MONITOR_POWER_ON {
		newState := int(pbs.Data[0])
		oldState := pm.lastDisplayState
		pm.lastDisplayState = newState

		// 显示器从关闭变为开启 = 系统唤醒
		if oldState == DisplayStateOff && newState == DisplayStateOn {
			return pm.triggerResume("显示器开启")
		}
	}

	return false
}

// triggerResume 触发唤醒事件
func (pm *PowerMonitor) triggerResume(reason string) bool {
	now := time.Now()

	// 防止短时间内重复触发
	if now.Sub(pm.lastResumeTime) < pm.minInterval {
		return false
	}

	pm.lastResumeTime = now

	if pm.callback != nil {
		go pm.callback()
	}

	return true
}

// SimulateDisplayOff 模拟显示器关闭（用于测试）
func (pm *PowerMonitor) SimulateDisplayOff() {
	pm.mu.Lock()
	pm.lastDisplayState = DisplayStateOff
	pm.mu.Unlock()
}

// PowerSettingGUID 返回需要监控的电源设置 GUID
func GetPowerSettingGUIDs() []windows.GUID {
	return []windows.GUID{
		GUID_CONSOLE_DISPLAY_STATE,
		GUID_MONITOR_POWER_ON,
	}
}

// RegisterPowerNotification 注册电源通知（服务模式）
// serviceHandle 应该是服务状态句柄
func RegisterPowerNotification(serviceHandle windows.Handle, guid *windows.GUID) (windows.Handle, error) {
	var regHandle windows.Handle

	ret, _, err := procPowerSettingRegisterNotification.Call(
		uintptr(unsafe.Pointer(guid)),
		uintptr(DEVICE_NOTIFY_SERVICE_HANDLE),
		uintptr(serviceHandle),
		uintptr(unsafe.Pointer(&regHandle)),
	)

	if ret != 0 {
		return 0, err
	}

	return regHandle, nil
}

// UnregisterPowerNotification 取消电源通知注册
func UnregisterPowerNotification(handle windows.Handle) error {
	ret, _, err := procPowerSettingUnregisterNotification.Call(uintptr(handle))
	if ret != 0 {
		return err
	}
	return nil
}

// GetPowerSettingGUID 根据 GUID 返回名称（用于日志）
func GetPowerSettingName(guid windows.GUID) string {
	switch guid {
	case GUID_CONSOLE_DISPLAY_STATE:
		return "CONSOLE_DISPLAY_STATE"
	case GUID_MONITOR_POWER_ON:
		return "MONITOR_POWER_ON"
	case GUID_SYSTEM_AWAYMODE:
		return "SYSTEM_AWAYMODE"
	default:
		return "UNKNOWN"
	}
}

// WakeEventPoller 设备状态轮询器（备用方案）
// 用于 Modern Standby 系统中，当电源事件不可靠时作为补充检测
type WakeEventPoller struct {
	callback          func() bool // 修复回调，返回是否成功
	stopChan          chan struct{}
	pauseChan         chan bool // 暂停/恢复控制
	pollInterval      time.Duration
	baseRetryInterval time.Duration // 基础重试间隔
	maxRetryInterval  time.Duration // 最大重试间隔（退避上限）
	maxRetryCount     int           // 最大连续失败次数（0=无限制）
	lastStatus        string
	pendingRepair     bool          // 设备在睡眠/长空闲期间变异常，等待唤醒后再修复
	lastRepairTime    time.Time     // 上次修复时间
	consecutiveFails  int           // 连续失败次数
	currentInterval   time.Duration // 当前重试间隔（退避用）
	deviceID          string
	logger            *Logger
	paused            bool // 是否暂停
	mu                sync.Mutex
}

// PollerConfig 轮询器配置
type PollerConfig struct {
	BaseRetryInterval time.Duration
	MaxRetryInterval  time.Duration
	MaxRetryCount     int
}

// NewWakeEventPoller 创建唤醒事件轮询器
func NewWakeEventPoller(deviceID string, callback func() bool, logger *Logger, cfg *PollerConfig) *WakeEventPoller {
	baseInterval := 60 * time.Second
	maxInterval := 10 * time.Minute
	maxRetry := 10

	if cfg != nil {
		if cfg.BaseRetryInterval > 0 {
			baseInterval = cfg.BaseRetryInterval
		}
		if cfg.MaxRetryInterval > 0 {
			maxInterval = cfg.MaxRetryInterval
		}
		if cfg.MaxRetryCount >= 0 {
			maxRetry = cfg.MaxRetryCount
		}
	}

	return &WakeEventPoller{
		callback:          callback,
		stopChan:          make(chan struct{}),
		pauseChan:         make(chan bool, 1),
		pollInterval:      10 * time.Second,
		baseRetryInterval: baseInterval,
		maxRetryInterval:  maxInterval,
		maxRetryCount:     maxRetry,
		currentInterval:   baseInterval,
		deviceID:          deviceID,
		logger:            logger,
	}
}

// Start 开始轮询
func (p *WakeEventPoller) Start() {
	go p.poll()
}

// Stop 停止轮询
func (p *WakeEventPoller) Stop() {
	close(p.stopChan)
}

// Pause 暂停轮询（系统睡眠时调用）
func (p *WakeEventPoller) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.paused {
		p.paused = true
		select {
		case p.pauseChan <- true:
		default:
		}
		p.logger.InfoTag(TagService, "设备状态轮询已暂停")
	}
}

// Resume 恢复轮询（系统唤醒时调用）
func (p *WakeEventPoller) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.paused {
		p.paused = false
		select {
		case p.pauseChan <- false:
		default:
		}
		p.logger.InfoTag(TagService, "设备状态轮询已恢复")

		// 如果有待处理的修复（在睡眠期间设备变异常），立即检查并处理
		if p.pendingRepair {
			p.logger.InfoTag(TagResume, "检测到待唤醒修复标记，立即检查设备状态")
			// 异步处理，避免阻塞Resume调用
			go func() {
				// 短暂等待系统稳定
				time.Sleep(2 * time.Second)

				dm := NewDeviceManager(p.deviceID)
				status, err := dm.GetStatus()
				if err == nil && status != "OK" {
					p.logger.InfoTag(TagResume, "唤醒后设备仍异常 (状态: %s)，立即执行修复", status)
					if p.callback != nil {
						success := p.callback()
						if success {
							p.pendingRepair = false
							p.ResetRetryState()
						}
					}
				} else if err == nil && status == "OK" {
					p.logger.InfoTag(TagResume, "唤醒后设备状态正常，清除待修复标记")
					p.pendingRepair = false
				}
			}()
		}
	}
}

// IsPaused 检查是否已暂停
func (p *WakeEventPoller) IsPaused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.paused
}

// ResetRetryState 重置重试状态（修复成功后调用）
func (p *WakeEventPoller) ResetRetryState() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.consecutiveFails = 0
	p.currentInterval = p.baseRetryInterval
}

// GetConsecutiveFails 获取连续失败次数
func (p *WakeEventPoller) GetConsecutiveFails() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.consecutiveFails
}

// incrementFails 增加失败计数并更新退避间隔
func (p *WakeEventPoller) incrementFails() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.consecutiveFails++

	// 检查是否超过最大重试次数
	if p.maxRetryCount > 0 && p.consecutiveFails >= p.maxRetryCount {
		p.logger.WarningTag(TagFail, "连续失败 %d 次，已达到最大重试次数，停止自动修复", p.consecutiveFails)
		return false // 返回 false 表示应该停止重试
	}

	// 指数退避：每次失败后间隔翻倍，但不超过最大值
	p.currentInterval = p.currentInterval * 2
	if p.currentInterval > p.maxRetryInterval {
		p.currentInterval = p.maxRetryInterval
	}

	p.logger.InfoTag(TagService, "连续失败 %d 次，下次重试间隔: %v", p.consecutiveFails, p.currentInterval)
	return true
}

// getCurrentInterval 获取当前重试间隔
func (p *WakeEventPoller) getCurrentInterval() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentInterval
}

// poll 轮询设备状态
func (p *WakeEventPoller) poll() {
	dm := NewDeviceManager(p.deviceID)

	// 获取初始状态
	status, err := dm.GetStatus()
	if err == nil {
		p.lastStatus = status
		if status != "OK" {
			p.logger.InfoTag(TagResume, "服务启动时检测到设备异常状态: %s，将尝试修复", status)
			if p.callback != nil {
				success := p.callback()
				p.lastRepairTime = time.Now()
				if success {
					p.ResetRetryState()
				} else {
					p.incrementFails()
				}
			}
		} else {
			p.logger.InfoTag(TagCheck, "设备初始状态正常: %s", status)
		}
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	exceededMaxRetry := false // 是否已超过最大重试次数

	for {
		select {
		case <-p.stopChan:
			return

		case paused := <-p.pauseChan:
			// 处理暂停/恢复
			if paused {
				// 等待恢复信号
				for {
					select {
					case <-p.stopChan:
						return
					case resumed := <-p.pauseChan:
						if !resumed {
							break // 跳出内层 for 循环
						}
					}
				}
			}
			// 恢复后继续下一次循环
			continue

		case <-ticker.C:
			// 如果已暂停，跳过本次轮询
			if p.IsPaused() {
				continue
			}

			// 如果已超过最大重试次数，只检查状态不再触发修复
			if exceededMaxRetry {
				status, err := dm.GetStatus()
				if err == nil && status == "OK" {
					p.logger.InfoTag(TagCheck, "设备状态已恢复正常: %s", status)
					p.ResetRetryState()
					exceededMaxRetry = false
					p.lastStatus = status
				}
				continue
			}

			status, err := dm.GetStatus()
			if err != nil {
				continue
			}

			// 状态恢复正常，清理“待唤醒修复”标记
			if status == "OK" {
				p.pendingRepair = false
			}

			shouldRepair := false
			reason := ""

			idleSec := -1
			if idleMs, err := GetIdleTime(); err == nil {
				idleSec = int(idleMs / 1000)
			}

			// 情况1：状态从 OK 变为 Error
			// 关键：如果系统已经长时间无输入（合盖/睡眠很常见），不要在睡眠期间修复；标记为待唤醒修复。
			if status != "OK" && p.lastStatus == "OK" {
				if idleSec < 0 || idleSec >= 300 {
					p.logger.InfoTag(TagCheck, "检测到设备状态变化: %s -> %s，但系统长时间空闲（%d秒），标记待唤醒修复", p.lastStatus, status, idleSec)
					p.pendingRepair = true
					p.lastStatus = status
					continue
				}
				p.logger.InfoTag(TagCheck, "检测到设备状态变化: %s -> %s，且系统活跃（空闲%d秒），立即修复", p.lastStatus, status, idleSec)
				shouldRepair = true
				reason = "状态变化"
				p.pendingRepair = false
				p.ResetRetryState()
			}

			// 情况2：设备一直是 Error 状态，且距离上次修复已经足够久
			if status != "OK" && p.lastStatus != "OK" {
				// 如果之前在长空闲/睡眠期间变异常，这里优先等待“唤醒后活跃”再立刻补修复
				if p.pendingRepair {
					if idleSec >= 0 && idleSec <= 60 {
						shouldRepair = true
						reason = "唤醒后补修复"
						p.pendingRepair = false
						p.ResetRetryState()
					} else {
						continue
					}
				} else {
					// 常规持续异常处理：仅在系统活跃时重试，避免睡眠中不断重试
					if idleSec >= 0 && idleSec > 60 {
						continue
					}
				}

				interval := p.getCurrentInterval()
				if time.Since(p.lastRepairTime) > interval {
					shouldRepair = true
					reason = "持续异常"
				}
			}

			if shouldRepair {
				p.logger.InfoTag(TagResume, "检测到设备状态异常 (轮询-%s): %s -> %s", reason, p.lastStatus, status)
				if p.callback != nil {
					success := p.callback()
					p.lastRepairTime = time.Now()
					if success {
						p.ResetRetryState()
					} else {
						if !p.incrementFails() {
							exceededMaxRetry = true
						}
					}
				}
			}

			p.lastStatus = status
		}
	}
}

// IsModernStandbySupported 检查系统是否使用 Modern Standby
func IsModernStandbySupported() bool {
	// 首先尝试注册表方法
	key, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\Power`,
		registry.QUERY_VALUE)
	if err == nil {
		defer key.Close()
		// CsEnabled = 1 表示 Connected Standby/Modern Standby 已启用
		val, _, err := key.GetIntegerValue("CsEnabled")
		if err == nil && val == 1 {
			return true
		}
	}

	// 备用方法：使用 powercfg 命令检测
	// 检查是否支持 S0 Low Power Idle（Modern Standby 的特征）
	output, err := runPowerShell("powercfg /availablesleepstates")
	if err == nil {
		// 检查输出中是否包含 S0 低电量待机的标志
		lower := strings.ToLower(output)
		if strings.Contains(lower, "s0 low power idle") ||
			strings.Contains(lower, "s0 低电量待机") ||
			strings.Contains(lower, "standby (s0") {
			return true
		}
	}

	return false
}

// GetSystemSleepState 获取系统睡眠状态信息
func GetSystemSleepState() string {
	if IsModernStandbySupported() {
		return "Modern Standby (S0 Low Power Idle)"
	}
	return "Traditional Sleep (S3)"
}

// LASTINPUTINFO 结构体用于 GetLastInputInfo
type LASTINPUTINFO struct {
	cbSize uint32
	dwTime uint32
}

// GetIdleTime 获取系统空闲时间（毫秒）
// 返回自上次用户输入（键盘/鼠标）以来的毫秒数
func GetIdleTime() (uint32, error) {
	var lii LASTINPUTINFO
	lii.cbSize = uint32(unsafe.Sizeof(lii))

	ret, _, err := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&lii)))
	if ret == 0 {
		return 0, err
	}

	// 获取当前 tick count
	tickCount, _, _ := procGetTickCount64.Call()

	// 计算空闲时间
	idleTime := uint32(tickCount) - lii.dwTime
	return idleTime, nil
}

// IsSystemLikelyAsleep 判断系统是否可能处于睡眠/休眠状态
// 在 Modern Standby 模式下，系统技术上仍在运行，但用户无交互
// 通过检测空闲时间来判断
func IsSystemLikelyAsleep(minIdleMinutes int) bool {
	if minIdleMinutes <= 0 {
		minIdleMinutes = 5 // 默认5分钟无操作视为睡眠
	}

	idleMs, err := GetIdleTime()
	if err != nil {
		return false // 无法获取，假设未睡眠
	}

	// 转换为分钟
	idleMinutes := int(idleMs / 1000 / 60)
	return idleMinutes >= minIdleMinutes
}

// IsUserSessionLocked 检查用户会话是否被锁定
// 注意：这在服务模式下可能不完全可靠
func IsUserSessionLocked() bool {
	// 通过检查空闲时间来间接判断
	// 如果空闲超过1分钟，很可能是锁屏或睡眠状态
	idleMs, err := GetIdleTime()
	if err != nil {
		return false
	}
	return idleMs > 60*1000 // 1分钟无输入
}
