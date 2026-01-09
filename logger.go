// Package main provides logging functionality for application events and debugging.
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
)

// EventTag 事件标签（用于结构化日志）
type EventTag string

const (
	TagResume  EventTag = "RESUME"  // 系统唤醒
	TagCheck   EventTag = "CHECK"   // 状态检查
	TagReset   EventTag = "RESET"   // 设备重置
	TagSkip    EventTag = "SKIP"    // 跳过修复
	TagSuccess EventTag = "SUCCESS" // 修复成功
	TagFail    EventTag = "FAIL"    // 修复失败
	TagService EventTag = "SERVICE" // 服务状态
	TagConfig  EventTag = "CONFIG"  // 配置相关
)

// Logger 日志记录器
type Logger struct {
	level      LogLevel
	logDir     string
	file       *os.File
	mu         sync.Mutex
	stdLogger  *log.Logger
	fileLogger *log.Logger
}

var (
	globalLogger *Logger
	loggerOnce   sync.Once
)

// resetLoggerForTesting 仅用于测试，重置全局 logger 状态
// 注意：此函数不是线程安全的，仅在测试中使用
func resetLoggerForTesting() {
	if globalLogger != nil {
		_ = globalLogger.Close()
		globalLogger = nil
	}
	loggerOnce = sync.Once{}
}

// InitLogger 初始化全局日志记录器
func InitLogger(logDir string, level LogLevel) error {
	return InitLoggerWithOptions(logDir, level, true)
}

// InitLoggerWithOptions 初始化全局日志记录器（带选项）
func InitLoggerWithOptions(logDir string, level LogLevel, enableConsole bool) error {
	var err error
	loggerOnce.Do(func() {
		globalLogger = &Logger{
			level:  level,
			logDir: logDir,
		}

		// 创建日志目录
		if logDir != "" {
			if err = os.MkdirAll(logDir, 0o755); err != nil {
				return
			}

			// 创建日志文件
			logFile := filepath.Join(logDir, fmt.Sprintf("gpd-touch_%s.log",
				time.Now().Format("2006-01-02")))

			globalLogger.file, err = os.OpenFile(logFile,
				os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return
			}

			// 设置文件日志器（不带前缀，我们自己格式化）
			globalLogger.fileLogger = log.New(globalLogger.file, "", 0)
		}

		// 设置标准日志器（可选）
		if enableConsole {
			globalLogger.stdLogger = log.New(os.Stdout, "", 0)
		}
	})

	return err
}

// GetLogger 获取全局日志记录器
func GetLogger() *Logger {
	if globalLogger == nil {
		_ = InitLogger("", INFO)
	}
	return globalLogger
}

// Close 关闭日志文件
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// log 内部日志方法
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	levelStr := l.levelString(level)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// 格式化输出
	formattedMsg := fmt.Sprintf("%s [%-5s] %s", timestamp, levelStr, msg)

	// 控制台输出
	if l.stdLogger != nil {
		l.stdLogger.Println(formattedMsg)
	}

	// 文件输出
	if l.fileLogger != nil {
		l.fileLogger.Println(formattedMsg)
	}
}

// logWithTag 带事件标签的日志方法
func (l *Logger) logWithTag(level LogLevel, tag EventTag, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	levelStr := l.levelString(level)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// 格式化输出：时间戳 [级别] [标签] 消息
	formattedMsg := fmt.Sprintf("%s [%-5s] [%-7s] %s", timestamp, levelStr, tag, msg)

	// 控制台输出
	if l.stdLogger != nil {
		l.stdLogger.Println(formattedMsg)
	}

	// 文件输出
	if l.fileLogger != nil {
		l.fileLogger.Println(formattedMsg)
	}
}

// levelString 返回日志级别字符串
func (l *Logger) levelString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warning 警告日志
func (l *Logger) Warning(format string, args ...interface{}) {
	l.log(WARNING, format, args...)
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// 带标签的日志方法

// InfoTag 带标签的信息日志
func (l *Logger) InfoTag(tag EventTag, format string, args ...interface{}) {
	l.logWithTag(INFO, tag, format, args...)
}

// WarningTag 带标签的警告日志
func (l *Logger) WarningTag(tag EventTag, format string, args ...interface{}) {
	l.logWithTag(WARNING, tag, format, args...)
}

// ErrorTag 带标签的错误日志
func (l *Logger) ErrorTag(tag EventTag, format string, args ...interface{}) {
	l.logWithTag(ERROR, tag, format, args...)
}

// LogToFile 写入特定内容到日志文件
func (l *Logger) LogToFile(content string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return fmt.Errorf("日志文件未初始化")
	}

	_, err := io.WriteString(l.file, content+"\n")
	return err
}

// GetLogDir 获取日志目录
func GetLogDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Join(filepath.Dir(exe), "logs")
}

// GetCurrentLogFile 获取当前日志文件路径
func GetCurrentLogFile() string {
	return filepath.Join(GetLogDir(), fmt.Sprintf("gpd-touch_%s.log", time.Now().Format("2006-01-02")))
}

// ReadLogLines 读取日志文件的最后 N 行
func ReadLogLines(n int) ([]string, error) {
	logDir := GetLogDir()

	// 获取所有日志文件
	files, err := filepath.Glob(filepath.Join(logDir, "gpd-touch_*.log"))
	if err != nil {
		return nil, fmt.Errorf("查找日志文件失败: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("未找到日志文件")
	}

	// 按文件名排序（日期格式保证了正确的顺序）
	sort.Strings(files)

	// 从最新的文件开始读取
	var lines []string
	for i := len(files) - 1; i >= 0 && len(lines) < n; i-- {
		fileLines, err := readFileLines(files[i])
		if err != nil {
			continue
		}

		// 将新行插入到开头
		lines = append(fileLines, lines...)
	}

	// 只返回最后 N 行
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	return lines, nil
}

// readFileLines 读取文件所有行
func readFileLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// FormatLogForDisplay 格式化日志行用于显示
func FormatLogForDisplay(lines []string) string {
	if len(lines) == 0 {
		return "No log records found\n"
	}

	var sb strings.Builder
	sb.WriteString("========== SERVICE LOG ==========\n\n")

	for _, line := range lines {
		// Add prefix based on log level
		prefix := "  "
		if strings.Contains(line, "[ERROR]") || strings.Contains(line, "[FAIL]") {
			prefix = "! "
		} else if strings.Contains(line, "[WARN]") {
			prefix = "W "
		} else if strings.Contains(line, "[SUCCESS]") {
			prefix = "+ "
		} else if strings.Contains(line, "[RESUME]") {
			prefix = "R "
		} else if strings.Contains(line, "[RESET]") {
			prefix = "* "
		} else if strings.Contains(line, "[SKIP]") {
			prefix = "- "
		} else if strings.Contains(line, "[CHECK]") {
			prefix = "? "
		}

		sb.WriteString(fmt.Sprintf("%s%s\n", prefix, line))
	}

	sb.WriteString("\n=================================\n")
	return sb.String()
}

// CleanOldLogs 清理过期日志文件
func CleanOldLogs(maxDays int) error {
	if maxDays <= 0 {
		return nil
	}

	logDir := GetLogDir()
	files, err := filepath.Glob(filepath.Join(logDir, "gpd-touch_*.log"))
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -maxDays)

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			_ = os.Remove(file)
		}
	}

	return nil
}
