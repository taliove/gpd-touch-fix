package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitLogger(t *testing.T) {
	tmpDir := t.TempDir()

	// 重置全局 logger
	resetLoggerForTesting()

	err := InitLoggerWithOptions(tmpDir, INFO, false)
	if err != nil {
		t.Fatalf("InitLogger() error = %v", err)
	}

	// 验证日志目录创建
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Log directory should be created")
	}

	// 验证日志文件创建
	files, _ := filepath.Glob(filepath.Join(tmpDir, "gpd-touch_*.log"))
	if len(files) == 0 {
		t.Error("Log file should be created")
	}

	// 清理
	if globalLogger != nil {
		globalLogger.Close()
	}
}

func TestLogger_LogLevels(t *testing.T) {
	tmpDir := t.TempDir()

	// 重置全局 logger
	resetLoggerForTesting()

	err := InitLoggerWithOptions(tmpDir, DEBUG, false)
	if err != nil {
		t.Fatalf("InitLogger() error = %v", err)
	}
	defer func() {
		if globalLogger != nil {
			globalLogger.Close()
		}
	}()

	logger := GetLogger()

	// 测试不同级别的日志
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warning("warning message")
	logger.Error("error message")

	// 验证日志文件有内容
	files, _ := filepath.Glob(filepath.Join(tmpDir, "gpd-touch_*.log"))
	if len(files) == 0 {
		t.Fatal("Log file should exist")
	}

	content, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "debug message") {
		t.Error("Log should contain debug message")
	}
	if !strings.Contains(logContent, "info message") {
		t.Error("Log should contain info message")
	}
}

func TestLogger_LogWithTag(t *testing.T) {
	tmpDir := t.TempDir()

	// 重置全局 logger
	resetLoggerForTesting()

	err := InitLoggerWithOptions(tmpDir, INFO, false)
	if err != nil {
		t.Fatalf("InitLogger() error = %v", err)
	}
	defer func() {
		if globalLogger != nil {
			globalLogger.Close()
		}
	}()

	logger := GetLogger()

	logger.InfoTag(TagResume, "system resumed")
	logger.WarningTag(TagFail, "repair failed")
	logger.ErrorTag(TagReset, "reset error")

	files, _ := filepath.Glob(filepath.Join(tmpDir, "gpd-touch_*.log"))
	if len(files) == 0 {
		t.Fatal("Log file should exist")
	}

	content, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, string(TagResume)) {
		t.Error("Log should contain RESUME tag")
	}
}

func TestLogger_SetLevel(t *testing.T) {
	tmpDir := t.TempDir()

	// 重置全局 logger
	resetLoggerForTesting()

	err := InitLoggerWithOptions(tmpDir, ERROR, false)
	if err != nil {
		t.Fatalf("InitLogger() error = %v", err)
	}
	defer func() {
		if globalLogger != nil {
			globalLogger.Close()
		}
	}()

	logger := GetLogger()

	// 只有 ERROR 级别以上才会记录
	logger.Debug("debug - should not appear")
	logger.Info("info - should not appear")
	logger.Error("error - should appear")

	files, _ := filepath.Glob(filepath.Join(tmpDir, "gpd-touch_*.log"))
	if len(files) == 0 {
		t.Fatal("Log file should exist")
	}

	content, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if strings.Contains(logContent, "debug - should not appear") {
		t.Error("Debug message should not appear with ERROR level")
	}
	if strings.Contains(logContent, "info - should not appear") {
		t.Error("Info message should not appear with ERROR level")
	}
	if !strings.Contains(logContent, "error - should appear") {
		t.Error("Error message should appear with ERROR level")
	}
}

func TestGetLogger_Singleton(t *testing.T) {
	logger1 := GetLogger()
	logger2 := GetLogger()

	if logger1 != logger2 {
		t.Error("GetLogger() should return the same instance")
	}
}

func TestFormatLogForDisplay(t *testing.T) {
	lines := []string{
		"2025-01-01 12:00:00.000 [INFO ] Normal log",
		"2025-01-01 12:00:01.000 [ERROR] Error log",
		"2025-01-01 12:00:02.000 [WARN ] Warning log",
		"2025-01-01 12:00:03.000 [INFO ] [SUCCESS] Success log",
	}

	formatted := FormatLogForDisplay(lines)

	if !strings.Contains(formatted, "SERVICE LOG") {
		t.Error("Formatted log should contain header")
	}

	if !strings.Contains(formatted, "! ") {
		t.Error("Error log should have '!' prefix")
	}

	if !strings.Contains(formatted, "+ ") {
		t.Error("Success log should have '+' prefix")
	}
}

func TestFormatLogForDisplay_Empty(t *testing.T) {
	formatted := FormatLogForDisplay([]string{})

	if !strings.Contains(formatted, "No log records") {
		t.Error("Empty log should show 'No log records' message")
	}
}

func TestGetLogDir(t *testing.T) {
	dir := GetLogDir()

	if dir == "" {
		t.Error("GetLogDir() should not return empty string")
	}

	if !strings.Contains(dir, "logs") {
		t.Error("GetLogDir() should return path containing 'logs'")
	}
}

func TestGetCurrentLogFile(t *testing.T) {
	file := GetCurrentLogFile()

	if file == "" {
		t.Error("GetCurrentLogFile() should not return empty string")
	}

	if !strings.Contains(file, "gpd-touch_") {
		t.Error("GetCurrentLogFile() should return path containing 'gpd-touch_'")
	}

	if !strings.HasSuffix(file, ".log") {
		t.Error("GetCurrentLogFile() should return path ending with '.log'")
	}
}
