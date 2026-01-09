package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewStatsManager(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStatsManager(tmpDir)

	if sm == nil {
		t.Fatal("NewStatsManager() returned nil")
	}

	if sm.statsDir != tmpDir {
		t.Errorf("statsDir = %q, want %q", sm.statsDir, tmpDir)
	}

	if sm.stats == nil {
		t.Error("stats should not be nil")
	}
}

func TestStatsManager_RecordResume(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStatsManager(tmpDir)

	initialCount := sm.stats.TotalResumeEvents

	sm.RecordResume()

	if sm.stats.TotalResumeEvents != initialCount+1 {
		t.Errorf("TotalResumeEvents = %d, want %d", sm.stats.TotalResumeEvents, initialCount+1)
	}

	if sm.stats.LastResumeTime == nil {
		t.Error("LastResumeTime should not be nil after RecordResume()")
	}
}

func TestStatsManager_RecordReset(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStatsManager(tmpDir)

	tests := []struct {
		name    string
		success bool
		result  string
	}{
		{"success", true, "修复成功"},
		{"failure", false, "修复失败"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialResets := sm.stats.TotalResets
			initialFailures := sm.stats.TotalFailures

			sm.RecordReset(tt.success, tt.result)

			if tt.success {
				if sm.stats.TotalResets != initialResets+1 {
					t.Errorf("TotalResets = %d, want %d", sm.stats.TotalResets, initialResets+1)
				}
			} else {
				if sm.stats.TotalFailures != initialFailures+1 {
					t.Errorf("TotalFailures = %d, want %d", sm.stats.TotalFailures, initialFailures+1)
				}
			}

			if sm.stats.LastResetResult != tt.result {
				t.Errorf("LastResetResult = %q, want %q", sm.stats.LastResetResult, tt.result)
			}
		})
	}
}

func TestStatsManager_RecordSkip(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStatsManager(tmpDir)

	initialSkips := sm.stats.TotalSkips

	sm.RecordSkip()

	if sm.stats.TotalSkips != initialSkips+1 {
		t.Errorf("TotalSkips = %d, want %d", sm.stats.TotalSkips, initialSkips+1)
	}

	if sm.stats.LastResetResult != "状态正常，已跳过" {
		t.Errorf("LastResetResult = %q, want %q", sm.stats.LastResetResult, "状态正常，已跳过")
	}
}

func TestStatsManager_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStatsManager(tmpDir)

	sm.RecordResume()
	sm.RecordReset(true, "成功")
	sm.RecordSkip()

	stats := sm.GetStats()

	if stats.TotalResumeEvents < 1 {
		t.Error("GetStats() should return stats with at least 1 resume event")
	}

	if stats.TotalResets < 1 {
		t.Error("GetStats() should return stats with at least 1 reset")
	}

	if stats.TotalSkips < 1 {
		t.Error("GetStats() should return stats with at least 1 skip")
	}
}

func TestStatsManager_FormatStats(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStatsManager(tmpDir)

	sm.RecordResume()
	sm.RecordReset(true, "成功")

	formatted := sm.FormatStats()

	if formatted == "" {
		t.Error("FormatStats() should not return empty string")
	}

	// 检查是否包含统计信息的标题
	if !strings.Contains(formatted, "统计信息") {
		t.Error("FormatStats() should contain '统计信息'")
	}

	// 检查是否包含今日/本周/本月等时间段
	if !strings.Contains(formatted, "今日") {
		t.Error("FormatStats() should contain '今日'")
	}
}

func TestStatsManager_FormatStatsSimple(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStatsManager(tmpDir)

	formatted := sm.FormatStatsSimple()

	if formatted == "" {
		t.Error("FormatStatsSimple() should not return empty string")
	}

	if !strings.Contains(formatted, "今日") {
		t.Error("FormatStatsSimple() should contain '今日'")
	}

	if !strings.Contains(formatted, "累计") {
		t.Error("FormatStatsSimple() should contain '累计'")
	}
}

func TestStatsManager_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStatsManager(tmpDir)

	// 记录一些事件
	sm.RecordResume()
	sm.RecordReset(true, "测试")
	sm.RecordSkip()

	originalResumeCount := sm.stats.TotalResumeEvents
	originalResetCount := sm.stats.TotalResets
	originalSkipCount := sm.stats.TotalSkips

	// 创建新的管理器，应该加载之前保存的数据
	sm2 := NewStatsManager(tmpDir)

	if sm2.stats.TotalResumeEvents != originalResumeCount {
		t.Errorf("Loaded TotalResumeEvents = %d, want %d", sm2.stats.TotalResumeEvents, originalResumeCount)
	}

	if sm2.stats.TotalResets != originalResetCount {
		t.Errorf("Loaded TotalResets = %d, want %d", sm2.stats.TotalResets, originalResetCount)
	}

	if sm2.stats.TotalSkips != originalSkipCount {
		t.Errorf("Loaded TotalSkips = %d, want %d", sm2.stats.TotalSkips, originalSkipCount)
	}
}

func TestStatsManager_StatsFileCreated(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStatsManager(tmpDir)

	sm.RecordResume()

	statsFile := filepath.Join(tmpDir, "stats.json")
	if _, err := os.Stat(statsFile); os.IsNotExist(err) {
		t.Error("stats.json file should be created after RecordResume()")
	}
}
