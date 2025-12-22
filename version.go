// Package main provides version information and build metadata.
package main

import (
	"fmt"
	"runtime"
	"time"
)

var (
	// 这些变量会在编译时通过 -ldflags 注入
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
)

// VersionInfo 版本信息
type VersionInfo struct {
	Version   string
	GitCommit string
	BuildTime string
	GoVersion string
	OS        string
	Arch      string
}

// GetVersionInfo 获取版本信息
func GetVersionInfo() *VersionInfo {
	return &VersionInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String 返回版本信息字符串
func (v *VersionInfo) String() string {
	buildTime := v.BuildTime
	if buildTime != "unknown" {
		if t, err := time.Parse(time.RFC3339, buildTime); err == nil {
			buildTime = t.Format("2006-01-02 15:04:05")
		}
	}

	return fmt.Sprintf(`GPD Touch Fix %s
  Git Commit:  %s
  Build Time:  %s
  Go Version:  %s
  Platform:    %s/%s`,
		v.Version,
		v.GitCommit,
		buildTime,
		v.GoVersion,
		v.OS,
		v.Arch)
}

// ShortVersion 返回简短版本
func (v *VersionInfo) ShortVersion() string {
	return fmt.Sprintf("v%s", v.Version)
}
