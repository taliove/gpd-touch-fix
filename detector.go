package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// DeviceInfo 设备信息
type DeviceInfo struct {
	InstanceID   string `json:"instance_id"`
	FriendlyName string `json:"friendly_name"`
	Status       string `json:"status"`
	Class        string `json:"class"`
	Description  string `json:"description"`
	Manufacturer string `json:"manufacturer"`
}

// IsError 判断设备是否处于错误状态
func (d *DeviceInfo) IsError() bool {
	return d.Status != "" && !strings.EqualFold(d.Status, "OK")
}

// IsI2CHID 判断是否是 I2C HID 设备
func (d *DeviceInfo) IsI2CHID() bool {
	lower := strings.ToLower(d.FriendlyName + d.Description + d.InstanceID)
	return strings.Contains(lower, "i2c") && strings.Contains(lower, "hid")
}

// IsTouchDevice 判断是否是触控设备
func (d *DeviceInfo) IsTouchDevice() bool {
	lower := strings.ToLower(d.FriendlyName + d.Description)
	keywords := []string{"touch", "触控", "触摸", "digitizer"}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// IsACPIDevice 判断是否是 ACPI 级别的设备（更可靠的禁用目标）
func (d *DeviceInfo) IsACPIDevice() bool {
	return strings.HasPrefix(strings.ToUpper(d.InstanceID), "ACPI\\")
}

// Score 计算设备匹配度分数（越高越可能是目标设备）
func (d *DeviceInfo) Score() int {
	score := 0

	// I2C HID 设备 +10
	if d.IsI2CHID() {
		score += 10
	}

	// 触控设备 +20
	if d.IsTouchDevice() {
		score += 20
	}

	// 有错误状态 +30（优先修复有问题的）
	if d.IsError() {
		score += 30
	}

	// HID 类设备 +5
	if strings.Contains(strings.ToLower(d.Class), "hid") {
		score += 5
	}

	// ACPI 设备 +15（优先选择 ACPI 级别设备，更可靠）
	if d.IsACPIDevice() {
		score += 15
	}

	return score
}

// Detector 设备检测器
type Detector struct{}

// NewDetector 创建设备检测器
func NewDetector() *Detector {
	return &Detector{}
}

// ScanAllDevices 扫描所有 PnP 设备
func (dt *Detector) ScanAllDevices() ([]*DeviceInfo, error) {
	script := `
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Get-PnpDevice | Where-Object { 
	($_.InstanceId -like '*I2C*' -and $_.InstanceId -like '*HID*') -or 
	($_.FriendlyName -like '*I2C*' -and $_.FriendlyName -like '*HID*')
} | ForEach-Object {
	$obj = @{
		instance_id = $_.InstanceId
		friendly_name = $_.FriendlyName
		status = $_.Status
		class = $_.Class
		description = if ($_.Description) { $_.Description } else { '' }
		manufacturer = if ($_.Manufacturer) { $_.Manufacturer } else { '' }
	}
	$obj | ConvertTo-Json -Compress
}
`
	output, err := runPowerShell(script)
	if err != nil {
		return nil, fmt.Errorf("扫描设备失败: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return []*DeviceInfo{}, nil
	}

	// 解析 JSON 输出
	lines := strings.Split(output, "\n")
	devices := make([]*DeviceInfo, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var device DeviceInfo
		if err := json.Unmarshal([]byte(line), &device); err != nil {
			log.Printf("警告: 解析设备信息失败: %v, 行: %s", err, line)
			continue
		}
		devices = append(devices, &device)
	}

	return devices, nil
}

// DetectI2CHIDDevices 检测所有 I2C HID 设备
func (dt *Detector) DetectI2CHIDDevices() ([]*DeviceInfo, error) {
	allDevices, err := dt.ScanAllDevices()
	if err != nil {
		return nil, err
	}

	i2cDevices := make([]*DeviceInfo, 0)
	for _, dev := range allDevices {
		if dev.IsI2CHID() {
			i2cDevices = append(i2cDevices, dev)
		}
	}

	return i2cDevices, nil
}

// DetectTouchDevices 检测所有触控设备
func (dt *Detector) DetectTouchDevices() ([]*DeviceInfo, error) {
	allDevices, err := dt.ScanAllDevices()
	if err != nil {
		return nil, err
	}

	touchDevices := make([]*DeviceInfo, 0)
	for _, dev := range allDevices {
		if dev.IsTouchDevice() {
			touchDevices = append(touchDevices, dev)
		}
	}

	return touchDevices, nil
}

// DetectErrorDevices 检测处于错误状态的设备
func (dt *Detector) DetectErrorDevices() ([]*DeviceInfo, error) {
	allDevices, err := dt.ScanAllDevices()
	if err != nil {
		return nil, err
	}

	errorDevices := make([]*DeviceInfo, 0)
	for _, dev := range allDevices {
		if dev.IsError() {
			errorDevices = append(errorDevices, dev)
		}
	}

	return errorDevices, nil
}

// DetectBestMatch 自动检测最可能的目标设备
func (dt *Detector) DetectBestMatch() (*DeviceInfo, []*DeviceInfo, error) {
	allDevices, err := dt.ScanAllDevices()
	if err != nil {
		return nil, nil, err
	}

	if len(allDevices) == 0 {
		return nil, nil, fmt.Errorf("未找到任何候选设备")
	}

	// 按分数排序
	candidates := make([]*DeviceInfo, 0)
	var bestDevice *DeviceInfo
	bestScore := -1

	for _, dev := range allDevices {
		score := dev.Score()
		if score > 0 {
			candidates = append(candidates, dev)
			if score > bestScore {
				bestScore = score
				bestDevice = dev
			}
		}
	}

	if bestDevice == nil {
		return nil, candidates, fmt.Errorf("未找到匹配的 I2C HID 触控设备")
	}

	return bestDevice, candidates, nil
}

// PrintDeviceInfo 打印设备信息（用于调试）
func PrintDeviceInfo(dev *DeviceInfo) {
	fmt.Printf("设备名称: %s\n", dev.FriendlyName)
	fmt.Printf("实例 ID: %s\n", dev.InstanceID)
	fmt.Printf("状态: %s", dev.Status)
	if dev.IsError() {
		fmt.Print(" ⚠")
	} else {
		fmt.Print(" ✓")
	}
	fmt.Println()
	if dev.Class != "" {
		fmt.Printf("类别: %s\n", dev.Class)
	}
	if dev.Description != "" {
		fmt.Printf("描述: %s\n", dev.Description)
	}
	if dev.Manufacturer != "" {
		fmt.Printf("制造商: %s\n", dev.Manufacturer)
	}
	fmt.Printf("匹配度: %d 分\n", dev.Score())
}
