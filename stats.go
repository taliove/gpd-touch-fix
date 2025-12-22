// Package main provides statistics tracking and reporting for device repairs.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventType 事件类型
type EventType string

const (
	EventResume  EventType = "RESUME"  // 系统唤醒
	EventCheck   EventType = "CHECK"   // 状态检查
	EventReset   EventType = "RESET"   // 设备重置
	EventSkip    EventType = "SKIP"    // 跳过修复
	EventSuccess EventType = "SUCCESS" // 修复成功
	EventFail    EventType = "FAIL"    // 修复失败
)

// EventRecord 事件记录
type EventRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	Type         EventType `json:"type"`
	DeviceStatus string    `json:"device_status,omitempty"`
	Message      string    `json:"message"`
	Success      bool      `json:"success"`
}

// Stats 统计数据
type Stats struct {
	TotalResumeEvents int `json:"total_resume_events"` // 总唤醒次数
	TotalResets       int `json:"total_resets"`        // 总修复次数
	TotalSkips        int `json:"total_skips"`         // 总跳过次数
	TotalFailures     int `json:"total_failures"`      // 总失败次数

	TodayResets   int `json:"today_resets"`   // 今日修复次数
	TodaySkips    int `json:"today_skips"`    // 今日跳过次数
	TodayFailures int `json:"today_failures"` // 今日失败次数

	WeekResets   int `json:"week_resets"`   // 本周修复次数
	WeekSkips    int `json:"week_skips"`    // 本周跳过次数
	WeekFailures int `json:"week_failures"` // 本周失败次数

	MonthResets   int `json:"month_resets"`   // 本月修复次数
	MonthSkips    int `json:"month_skips"`    // 本月跳过次数
	MonthFailures int `json:"month_failures"` // 本月失败次数

	LastResetTime   *time.Time `json:"last_reset_time,omitempty"`   // 上次修复时间
	LastResumeTime  *time.Time `json:"last_resume_time,omitempty"`  // 上次唤醒时间
	LastEventTime   *time.Time `json:"last_event_time,omitempty"`   // 上次事件时间
	LastResetResult string     `json:"last_reset_result,omitempty"` // 上次修复结果

	// 内部使用
	LastStatDate string `json:"last_stat_date"` // 上次统计日期，用于重置计数器
}

// StatsManager 统计管理器
type StatsManager struct {
	stats    *Stats
	statsDir string
	mu       sync.Mutex
}

// NewStatsManager 创建统计管理器
func NewStatsManager(statsDir string) *StatsManager {
	if statsDir == "" {
		exe, _ := os.Executable()
		statsDir = filepath.Dir(exe)
	}

	sm := &StatsManager{
		stats:    &Stats{},
		statsDir: statsDir,
	}

	// 尝试加载已有统计
	sm.load()

	return sm
}

// getStatsFilePath 获取统计文件路径
func (sm *StatsManager) getStatsFilePath() string {
	return filepath.Join(sm.statsDir, "stats.json")
}

// load 加载统计数据
func (sm *StatsManager) load() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	data, err := os.ReadFile(sm.getStatsFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			sm.stats = &Stats{LastStatDate: time.Now().Format("2006-01-02")}
			return nil
		}
		return err
	}

	var stats Stats
	if err := json.Unmarshal(data, &stats); err != nil {
		return err
	}

	sm.stats = &stats
	sm.checkDateRollover()

	return nil
}

// save 保存统计数据
func (sm *StatsManager) save() error {
	data, err := json.MarshalIndent(sm.stats, "", "  ")
	if err != nil {
		return err
	}

	// 确保目录存在
	if err := os.MkdirAll(sm.statsDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(sm.getStatsFilePath(), data, 0644)
}

// checkDateRollover 检查日期变化，重置计数器
func (sm *StatsManager) checkDateRollover() {
	now := time.Now()
	today := now.Format("2006-01-02")

	if sm.stats.LastStatDate == "" {
		sm.stats.LastStatDate = today
		return
	}

	lastDate, err := time.Parse("2006-01-02", sm.stats.LastStatDate)
	if err != nil {
		sm.stats.LastStatDate = today
		return
	}

	// 检查是否是新的一天
	if today != sm.stats.LastStatDate {
		// 重置今日计数
		sm.stats.TodayResets = 0
		sm.stats.TodaySkips = 0
		sm.stats.TodayFailures = 0

		// 检查是否是新的一周（周一开始）
		_, lastWeek := lastDate.ISOWeek()
		_, currentWeek := now.ISOWeek()
		if lastWeek != currentWeek || lastDate.Year() != now.Year() {
			sm.stats.WeekResets = 0
			sm.stats.WeekSkips = 0
			sm.stats.WeekFailures = 0
		}

		// 检查是否是新的一月
		if lastDate.Month() != now.Month() || lastDate.Year() != now.Year() {
			sm.stats.MonthResets = 0
			sm.stats.MonthSkips = 0
			sm.stats.MonthFailures = 0
		}

		sm.stats.LastStatDate = today
	}
}

// RecordResume 记录唤醒事件
func (sm *StatsManager) RecordResume() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.checkDateRollover()

	now := time.Now()
	sm.stats.TotalResumeEvents++
	sm.stats.LastResumeTime = &now

	sm.save()
}

// RecordReset 记录修复事件
func (sm *StatsManager) RecordReset(success bool, result string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.checkDateRollover()

	now := time.Now()
	sm.stats.LastResetTime = &now
	sm.stats.LastEventTime = &now
	sm.stats.LastResetResult = result

	if success {
		sm.stats.TotalResets++
		sm.stats.TodayResets++
		sm.stats.WeekResets++
		sm.stats.MonthResets++
	} else {
		sm.stats.TotalFailures++
		sm.stats.TodayFailures++
		sm.stats.WeekFailures++
		sm.stats.MonthFailures++
	}

	sm.save()
}

// RecordSkip 记录跳过事件
func (sm *StatsManager) RecordSkip() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.checkDateRollover()

	now := time.Now()
	sm.stats.LastEventTime = &now
	sm.stats.LastResetResult = "状态正常，已跳过"

	sm.stats.TotalSkips++
	sm.stats.TodaySkips++
	sm.stats.WeekSkips++
	sm.stats.MonthSkips++

	sm.save()
}

// GetStats 获取统计数据副本
func (sm *StatsManager) GetStats() Stats {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.checkDateRollover()

	return *sm.stats
}

// FormatStats 格式化统计数据为人类可读格式
func (sm *StatsManager) FormatStats() string {
	stats := sm.GetStats()

	var result string
	result += "╔══════════════════════════════════════════╗\n"
	result += "║           📊 统计信息                     ║\n"
	result += "╠══════════════════════════════════════════╣\n"

	// 今日统计
	result += fmt.Sprintf("║ 📅 今日                                  ║\n")
	result += fmt.Sprintf("║    修复: %-3d  跳过: %-3d  失败: %-3d       ║\n",
		stats.TodayResets, stats.TodaySkips, stats.TodayFailures)

	// 本周统计
	result += fmt.Sprintf("║ 📆 本周                                  ║\n")
	result += fmt.Sprintf("║    修复: %-3d  跳过: %-3d  失败: %-3d       ║\n",
		stats.WeekResets, stats.WeekSkips, stats.WeekFailures)

	// 本月统计
	result += fmt.Sprintf("║ 🗓️  本月                                  ║\n")
	result += fmt.Sprintf("║    修复: %-3d  跳过: %-3d  失败: %-3d       ║\n",
		stats.MonthResets, stats.MonthSkips, stats.MonthFailures)

	// 累计统计
	result += "╠══════════════════════════════════════════╣\n"
	result += fmt.Sprintf("║ 📈 累计                                  ║\n")
	result += fmt.Sprintf("║    唤醒: %-5d                            ║\n", stats.TotalResumeEvents)
	result += fmt.Sprintf("║    修复: %-5d                            ║\n", stats.TotalResets)
	result += fmt.Sprintf("║    跳过: %-5d                            ║\n", stats.TotalSkips)
	result += fmt.Sprintf("║    失败: %-5d                            ║\n", stats.TotalFailures)

	// 最近事件
	result += "╠══════════════════════════════════════════╣\n"
	result += "║ 🕐 最近事件                              ║\n"

	if stats.LastResumeTime != nil {
		result += fmt.Sprintf("║    上次唤醒: %s   ║\n", stats.LastResumeTime.Format("2006-01-02 15:04:05"))
	} else {
		result += "║    上次唤醒: 无记录                      ║\n"
	}

	if stats.LastResetTime != nil {
		result += fmt.Sprintf("║    上次修复: %s   ║\n", stats.LastResetTime.Format("2006-01-02 15:04:05"))
	} else {
		result += "║    上次修复: 无记录                      ║\n"
	}

	if stats.LastResetResult != "" {
		// 截断结果字符串以适应宽度
		result := stats.LastResetResult
		if len(result) > 28 {
			result = result[:25] + "..."
		}
		result = fmt.Sprintf("║    结果: %-32s ║\n", result)
	}

	result += "╚══════════════════════════════════════════╝\n"

	return result
}

// FormatStatsSimple 格式化简洁统计信息
func (sm *StatsManager) FormatStatsSimple() string {
	stats := sm.GetStats()

	return fmt.Sprintf("今日: 修复%d/跳过%d/失败%d | 累计: 修复%d/跳过%d",
		stats.TodayResets, stats.TodaySkips, stats.TodayFailures,
		stats.TotalResets, stats.TotalSkips)
}

// GetStatsDir 获取统计目录
func GetStatsDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}
