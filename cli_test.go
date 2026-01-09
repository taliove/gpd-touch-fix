package main

import (
	"strings"
	"testing"
)

func TestNewCLI(t *testing.T) {
	cli := NewCLI()

	if cli == nil {
		t.Fatal("NewCLI() returned nil")
	}

	if cli.reader == nil {
		t.Error("reader should not be nil")
	}
}

func TestCLI_colorize(t *testing.T) {
	cli := NewCLI()

	tests := []struct {
		name  string
		color string
		text  string
	}{
		{"green", ColorGreen, "success"},
		{"red", ColorRed, "error"},
		{"yellow", ColorYellow, "warning"},
		{"cyan", ColorCyan, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cli.colorize(tt.color, tt.text)

			// 应该包含原始文本
			if !strings.Contains(result, tt.text) {
				t.Errorf("colorize(%q, %q) should contain %q", tt.color, tt.text, tt.text)
			}

			// 如果颜色启用，应该包含颜色代码
			if cli.colorEnabled {
				if !strings.Contains(result, tt.color) {
					t.Errorf("colorize() should contain color code when color is enabled")
				}
				if !strings.Contains(result, ColorReset) {
					t.Errorf("colorize() should contain reset code when color is enabled")
				}
			}
		})
	}
}

func TestCLI_colorize_Disabled(t *testing.T) {
	cli := &CLI{
		colorEnabled: false,
	}

	result := cli.colorize(ColorGreen, "test")

	// 颜色禁用时应该只返回原始文本
	if result != "test" {
		t.Errorf("colorize() with disabled color = %q, want %q", result, "test")
	}
}

func TestColorConstants(t *testing.T) {
	colors := map[string]string{
		"Reset":  ColorReset,
		"Red":    ColorRed,
		"Green":  ColorGreen,
		"Yellow": ColorYellow,
		"Blue":   ColorBlue,
		"Purple": ColorPurple,
		"Cyan":   ColorCyan,
		"White":  ColorWhite,
		"Bold":   ColorBold,
	}

	for name, color := range colors {
		if color == "" {
			t.Errorf("Color%s should not be empty", name)
		}

		// ANSI 颜色代码应该以 ESC[ 开头
		if !strings.HasPrefix(color, "\033[") {
			t.Errorf("Color%s should start with ESC[", name)
		}
	}
}

func TestPrintDeviceList_Empty(t *testing.T) {
	cli := NewCLI()

	// 空设备列表不应该 panic
	cli.PrintDeviceList([]*DeviceInfo{})
}

func TestPrintDeviceList_WithDevices(t *testing.T) {
	cli := NewCLI()

	devices := []*DeviceInfo{
		{
			InstanceID:   "ACPI\\TEST123",
			FriendlyName: "Test Device",
			Status:       "OK",
			Class:        "HID",
		},
		{
			InstanceID:   "ACPI\\TEST456",
			FriendlyName: "Error Device",
			Status:       "Error",
			Class:        "HID",
		},
	}

	// 不应该 panic
	cli.PrintDeviceList(devices)
}

func TestSelectDevice_EmptyList(t *testing.T) {
	cli := NewCLI()

	_, err := cli.SelectDevice([]*DeviceInfo{}, nil)
	if err == nil {
		t.Error("SelectDevice() with empty list should return error")
	}
}

func TestShowProgress_NoPanic(t *testing.T) {
	cli := NewCLI()
	done := make(chan bool, 1)

	// 立即发送完成信号
	done <- true

	// 不应该 panic
	cli.ShowProgress("Testing...", done)
}
