package main

import (
	"testing"
)

func TestNewNotifier(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewNotifier(tt.enabled)

			if n == nil {
				t.Fatal("NewNotifier() returned nil")
			}

			if n.enabled != tt.enabled {
				t.Errorf("enabled = %v, want %v", n.enabled, tt.enabled)
			}

			if n.appName == "" {
				t.Error("appName should not be empty")
			}
		})
	}
}

func TestNotifier_SetEnabled(t *testing.T) {
	n := NewNotifier(false)

	n.SetEnabled(true)
	if !n.IsEnabled() {
		t.Error("After SetEnabled(true), IsEnabled() should return true")
	}

	n.SetEnabled(false)
	if n.IsEnabled() {
		t.Error("After SetEnabled(false), IsEnabled() should return false")
	}
}

func TestNotifier_IsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		want    bool
	}{
		{"enabled", true, true},
		{"disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewNotifier(tt.enabled)

			if got := n.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotifier_Send_Disabled(t *testing.T) {
	n := NewNotifier(false)

	// 禁用时发送应该直接返回 nil，不执行 PowerShell
	err := n.Send(NotifyInfo, "Test", "Message")
	if err != nil {
		t.Errorf("Send() with disabled notifier should return nil, got %v", err)
	}
}

func TestNotifier_SendMethods_Disabled(t *testing.T) {
	n := NewNotifier(false)

	tests := []struct {
		name   string
		method func(string, string) error
	}{
		{"SendSuccess", n.SendSuccess},
		{"SendError", n.SendError},
		{"SendInfo", n.SendInfo},
		{"SendWarning", n.SendWarning},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.method("Test", "Message")
			if err != nil {
				t.Errorf("%s() with disabled notifier should return nil, got %v", tt.name, err)
			}
		})
	}
}

func TestNotifier_NotifyResumeResult_Disabled(t *testing.T) {
	n := NewNotifier(false)

	// 禁用时不应该有任何错误（不执行任何操作）
	n.NotifyResumeResult(true, false, "TestDevice", nil)
	n.NotifyResumeResult(false, true, "TestDevice", nil)
	n.NotifyResumeResult(false, false, "TestDevice", nil)
}

func TestEscapeForPowerShell(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no special chars", "hello world", "hello world"},
		{"single quote", "it's", "it''s"},
		{"backtick", "hello`world", "hello``world"},
		{"dollar sign", "price: $100", "price: `$100"},
		{"multiple special", "it's $100 `test`", "it''s `$100 ``test``"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeForPowerShell(tt.input)
			if result != tt.expected {
				t.Errorf("escapeForPowerShell(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
