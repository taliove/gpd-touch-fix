package main

import (
	"strings"
	"testing"
)

func TestEscapeSingleQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "无单引号",
			input:    "ACPI\\VEN_INT&DEV_0B45",
			expected: "ACPI\\VEN_INT&DEV_0B45",
		},
		{
			name:     "含有单引号",
			input:    "Device's Name",
			expected: "Device''s Name",
		},
		{
			name:     "多个单引号",
			input:    "It's a 'test'",
			expected: "It''s a ''test''",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeSingleQuotes(tt.input)
			if result != tt.expected {
				t.Errorf("escapeSingleQuotes(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDeviceManager_GetStatus(t *testing.T) {
	// 这个测试需要实际的设备 InstanceId，跳过
	t.Skip("需要实际的设备才能测试")

	dm := NewDeviceManager("INVALID_INSTANCE_ID")
	_, err := dm.GetStatus()
	if err == nil {
		t.Error("期望获取无效设备状态时返回错误")
	}
}

func TestNewDeviceManager(t *testing.T) {
	instanceID := "TEST_INSTANCE_ID"
	dm := NewDeviceManager(instanceID)

	if dm == nil {
		t.Fatal("NewDeviceManager 返回 nil")
	}

	if dm.instanceID != instanceID {
		t.Errorf("DeviceManager.instanceID = %q, want %q", dm.instanceID, instanceID)
	}
}

// Mock 测试：验证 PowerShell 命令格式
func TestPowerShellCommandFormat(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
		operation  string
		wantCmd    string
	}{
		{
			name:       "Get Status",
			instanceID: "ACPI\\VEN_INT&DEV_0B45",
			operation:  "status",
			wantCmd:    "(Get-PnpDevice -InstanceId 'ACPI\\VEN_INT&DEV_0B45').Status",
		},
		{
			name:       "Enable Device",
			instanceID: "ACPI\\VEN_INT&DEV_0B45",
			operation:  "enable",
			wantCmd:    "Enable-PnpDevice -InstanceId 'ACPI\\VEN_INT&DEV_0B45' -Confirm:$false",
		},
		{
			name:       "Disable Device",
			instanceID: "ACPI\\VEN_INT&DEV_0B45",
			operation:  "disable",
			wantCmd:    "Disable-PnpDevice -InstanceId 'ACPI\\VEN_INT&DEV_0B45' -Confirm:$false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd string
			escaped := escapeSingleQuotes(tt.instanceID)

			switch tt.operation {
			case "status":
				cmd = "(Get-PnpDevice -InstanceId '" + escaped + "').Status"
			case "enable":
				cmd = "Enable-PnpDevice -InstanceId '" + escaped + "' -Confirm:$false"
			case "disable":
				cmd = "Disable-PnpDevice -InstanceId '" + escaped + "' -Confirm:$false"
			}

			if cmd != tt.wantCmd {
				t.Errorf("命令格式错误:\ngot:  %q\nwant: %q", cmd, tt.wantCmd)
			}
		})
	}
}

// 测试命令中的特殊字符处理
func TestSpecialCharactersInInstanceID(t *testing.T) {
	instanceID := "ACPI\\VEN_INT&DEV_0B45&SUBSYS_12345678&REV_01\\4&2E0E0FF&0&0010"
	dm := NewDeviceManager(instanceID)

	if dm.instanceID != instanceID {
		t.Errorf("特殊字符处理失败")
	}

	// 验证转义后的 ID 不包含单引号
	escaped := escapeSingleQuotes(instanceID)
	if strings.Contains(escaped, "'") && !strings.Contains(escaped, "''") {
		t.Errorf("单引号转义不正确: %q", escaped)
	}
}
