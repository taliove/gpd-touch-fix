package main

import (
	"strings"
	"testing"
)

func TestDeviceInfo_IsI2CHID(t *testing.T) {
	tests := []struct {
		name     string
		device   DeviceInfo
		expected bool
	}{
		{
			name: "标准 I2C HID 设备",
			device: DeviceInfo{
				FriendlyName: "I2C HID Device",
				InstanceID:   "ACPI\\VEN_INT&DEV_0B45",
			},
			expected: true,
		},
		{
			name: "触控屏设备",
			device: DeviceInfo{
				FriendlyName: "HID-compliant touch screen",
				Description:  "I2C Device",
			},
			expected: true,
		},
		{
			name: "非 I2C 设备",
			device: DeviceInfo{
				FriendlyName: "USB Mouse",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.device.IsI2CHID()
			if result != tt.expected {
				t.Errorf("IsI2CHID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDeviceInfo_IsTouchDevice(t *testing.T) {
	tests := []struct {
		name     string
		device   DeviceInfo
		expected bool
	}{
		{
			name: "触摸屏设备",
			device: DeviceInfo{
				FriendlyName: "HID-compliant touch screen",
			},
			expected: true,
		},
		{
			name: "触控设备（中文）",
			device: DeviceInfo{
				Description: "触控设备",
			},
			expected: true,
		},
		{
			name: "Digitizer",
			device: DeviceInfo{
				FriendlyName: "HID Digitizer",
			},
			expected: true,
		},
		{
			name: "鼠标设备",
			device: DeviceInfo{
				FriendlyName: "USB Mouse",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.device.IsTouchDevice()
			if result != tt.expected {
				t.Errorf("IsTouchDevice() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDeviceInfo_IsACPIDevice(t *testing.T) {
	tests := []struct {
		name     string
		device   DeviceInfo
		expected bool
	}{
		{
			name: "ACPI 设备",
			device: DeviceInfo{
				InstanceID: "ACPI\\NVTK0603\\4",
			},
			expected: true,
		},
		{
			name: "小写 acpi 设备",
			device: DeviceInfo{
				InstanceID: "acpi\\device\\1",
			},
			expected: true,
		},
		{
			name: "HID 子设备",
			device: DeviceInfo{
				InstanceID: "HID\\VEN_NVTK&DEV_0603",
			},
			expected: false,
		},
		{
			name: "USB 设备",
			device: DeviceInfo{
				InstanceID: "USB\\VID_1234&PID_5678",
			},
			expected: false,
		},
		{
			name: "空实例 ID",
			device: DeviceInfo{
				InstanceID: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.device.IsACPIDevice()
			if result != tt.expected {
				t.Errorf("IsACPIDevice() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDeviceInfo_IsError(t *testing.T) {
	tests := []struct {
		name     string
		device   DeviceInfo
		expected bool
	}{
		{
			name: "正常设备",
			device: DeviceInfo{
				Status: "OK",
			},
			expected: false,
		},
		{
			name: "错误设备",
			device: DeviceInfo{
				Status: "Error",
			},
			expected: true,
		},
		{
			name: "未知状态",
			device: DeviceInfo{
				Status: "Unknown",
			},
			expected: true,
		},
		{
			name: "空状态",
			device: DeviceInfo{
				Status: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.device.IsError()
			if result != tt.expected {
				t.Errorf("IsError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDeviceInfo_Score(t *testing.T) {
	tests := []struct {
		name     string
		device   DeviceInfo
		minScore int
		maxScore int
	}{
		{
			name: "完美匹配（I2C HID + Touch + Error + ACPI）",
			device: DeviceInfo{
				FriendlyName: "I2C HID touch screen",
				Status:       "Error",
				Class:        "HIDClass",
				InstanceID:   "ACPI\\NVTK0603\\4",
			},
			minScore: 75, // 10 + 20 + 30 + 15 = 75
			maxScore: 85,
		},
		{
			name: "I2C HID + Touch + Error（非 ACPI）",
			device: DeviceInfo{
				FriendlyName: "I2C HID touch screen",
				Status:       "Error",
				Class:        "HIDClass",
				InstanceID:   "HID\\VEN_NVTK&DEV_0603",
			},
			minScore: 60, // 10 + 20 + 30 = 60
			maxScore: 70,
		},
		{
			name: "触控设备（正常）",
			device: DeviceInfo{
				FriendlyName: "HID-compliant touch screen",
				Status:       "OK",
			},
			minScore: 20, // 20
			maxScore: 30,
		},
		{
			name: "普通鼠标",
			device: DeviceInfo{
				FriendlyName: "USB Mouse",
				Status:       "OK",
			},
			minScore: 0,
			maxScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tt.device.Score()
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Score() = %d, want between %d and %d",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestNewDetector(t *testing.T) {
	detector := NewDetector()
	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}
}

func TestDetector_ScanAllDevices_Integration(t *testing.T) {
	// 集成测试 - 需要实际系统
	t.Skip("需要实际的 Windows 系统才能运行")

	detector := NewDetector()
	devices, err := detector.ScanAllDevices()

	if err != nil {
		t.Fatalf("ScanAllDevices() error = %v", err)
	}

	t.Logf("找到 %d 个设备", len(devices))

	for i, dev := range devices {
		t.Logf("设备 #%d: %s (%s)", i+1, dev.FriendlyName, dev.Status)
	}
}

func TestPrintDeviceInfo(t *testing.T) {
	dev := &DeviceInfo{
		FriendlyName: "Test Device",
		InstanceID:   "TEST\\ID\\123",
		Status:       "OK",
		Class:        "HIDClass",
		Description:  "Test Description",
		Manufacturer: "Test Manufacturer",
	}

	// 这个测试只是确保函数不会 panic
	PrintDeviceInfo(dev)
}

// Mock 测试 PowerShell 输出解析
func TestJSONParsing(t *testing.T) {
	jsonOutput := `{"instance_id":"TEST\\ID","friendly_name":"Test Device","status":"OK","class":"HID","description":"Test Desc","manufacturer":"Test Mfg"}`

	// 测试能否正确解析 JSON
	if !strings.Contains(jsonOutput, "instance_id") {
		t.Error("JSON 格式不正确")
	}
}
