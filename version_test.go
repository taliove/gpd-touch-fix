package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()

	if info == nil {
		t.Fatal("GetVersionInfo() returned nil")
	}

	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, runtime.Version())
	}

	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}

	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
}

func TestVersionInfo_String(t *testing.T) {
	info := GetVersionInfo()
	str := info.String()

	if !strings.Contains(str, "GPD Touch Fix") {
		t.Error("String() should contain 'GPD Touch Fix'")
	}

	if !strings.Contains(str, info.Version) {
		t.Errorf("String() should contain version %q", info.Version)
	}

	if !strings.Contains(str, info.GoVersion) {
		t.Errorf("String() should contain Go version %q", info.GoVersion)
	}

	if !strings.Contains(str, info.OS) {
		t.Errorf("String() should contain OS %q", info.OS)
	}
}

func TestVersionInfo_ShortVersion(t *testing.T) {
	info := GetVersionInfo()
	short := info.ShortVersion()

	if !strings.HasPrefix(short, "v") {
		t.Errorf("ShortVersion() = %q, should start with 'v'", short)
	}

	if !strings.Contains(short, info.Version) {
		t.Errorf("ShortVersion() = %q, should contain %q", short, info.Version)
	}
}

func TestVersionInfo_StringWithBuildTime(t *testing.T) {
	// 测试有效的 RFC3339 格式时间
	original := BuildTime
	defer func() { BuildTime = original }()

	BuildTime = "2025-01-01T12:00:00Z"
	info := GetVersionInfo()
	str := info.String()

	// 应该格式化为可读格式
	if !strings.Contains(str, "2025-01-01 12:00:00") {
		t.Error("String() should format RFC3339 time to readable format")
	}
}
