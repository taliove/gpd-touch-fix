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

	user32                                 = windows.NewLazySystemDLL("user32.dll")
	procRegisterPowerSettingNotification   = user32.NewProc("RegisterPowerSettingNotification")
	procUnregisterPowerSettingNotification = user32.NewProc("UnregisterPowerSettingNotification")
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

// WaitForWakeEvent 等待唤醒事件（轮询模式，作为备用方案）
// 这是一个简单的轮询实现，当其他方法不起作用时使用
type WakeEventPoller struct {
	callback       func()
	stopChan       chan struct{}
	pollInterval   time.Duration
	retryInterval  time.Duration // 修复失败后的重试间隔
	lastStatus     string
	lastRepairTime time.Time // 上次修复时间
	deviceID       string
	logger         *Logger
}

// NewWakeEventPoller 创建唤醒事件轮询器
func NewWakeEventPoller(deviceID string, callback func(), logger *Logger) *WakeEventPoller {
	return &WakeEventPoller{
		callback:      callback,
		stopChan:      make(chan struct{}),
		pollInterval:  10 * time.Second,
		retryInterval: 60 * time.Second, // 修复后至少等60秒再重试
		deviceID:      deviceID,
		logger:        logger,
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

// poll 轮询设备状态
func (p *WakeEventPoller) poll() {
	dm := NewDeviceManager(p.deviceID)

	// 获取初始状态，如果已经是 Error 则立即尝试修复
	status, err := dm.GetStatus()
	if err == nil {
		p.lastStatus = status
		if status != "OK" {
			p.logger.InfoTag(TagResume, "服务启动时检测到设备异常状态: %s，将尝试修复", status)
			if p.callback != nil {
				p.callback()
				p.lastRepairTime = time.Now()
			}
		} else {
			p.logger.InfoTag(TagCheck, "设备初始状态正常: %s", status)
		}
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			status, err := dm.GetStatus()
			if err != nil {
				continue
			}

			shouldRepair := false
			reason := ""

			// 情况1：状态从 OK 变为 Error（刚从睡眠恢复导致）
			if status != "OK" && p.lastStatus == "OK" {
				shouldRepair = true
				reason = "状态变化"
			}

			// 情况2：设备一直是 Error 状态，且距离上次修复已经足够久
			if status != "OK" && p.lastStatus != "OK" {
				if time.Since(p.lastRepairTime) > p.retryInterval {
					shouldRepair = true
					reason = "持续异常"
				}
			}

			if shouldRepair {
				p.logger.InfoTag(TagResume, "检测到设备状态异常 (轮询-%s): %s -> %s", reason, p.lastStatus, status)
				if p.callback != nil {
					p.callback()
					p.lastRepairTime = time.Now()
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
