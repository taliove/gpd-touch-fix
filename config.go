package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 配置结构
type Config struct {
	DeviceInstanceID string   `json:"device_instance_id"`
	DeviceName       string   `json:"device_name,omitempty"` // 设备友好名称
	WaitSeconds      int      `json:"wait_seconds"`
	BackupDevices    []string `json:"backup_devices,omitempty"` // 备选设备列表
	AutoDetect       bool     `json:"auto_detect,omitempty"`    // 是否自动检测
	LogLevel         string   `json:"log_level,omitempty"`      // 日志级别
	LogDir           string   `json:"log_dir,omitempty"`        // 日志目录

	// 智能检测配置
	CheckBeforeReset   bool `json:"check_before_reset,omitempty"`   // 修复前先检查状态
	ResumeDelaySeconds int  `json:"resume_delay_seconds,omitempty"` // 唤醒后等待秒数
	LogAllEvents       bool `json:"log_all_events,omitempty"`       // 记录所有事件（包括跳过的）

	// 通知配置
	EnableNotification bool `json:"enable_notification,omitempty"` // 启用 Windows 通知

	// 日志管理
	MaxLogDays int `json:"max_log_days,omitempty"` // 日志保留天数
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		DeviceInstanceID:   "",
		DeviceName:         "",
		WaitSeconds:        2,
		BackupDevices:      []string{},
		AutoDetect:         true,
		LogLevel:           "INFO",
		LogDir:             "",
		CheckBeforeReset:   true, // 默认先检查再修复
		ResumeDelaySeconds: 3,    // 默认等待3秒
		LogAllEvents:       true, // 默认记录所有事件
		EnableNotification: true, // 默认启用通知
		MaxLogDays:         30,   // 默认保留30天
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &cfg, nil
}

// SaveConfig 保存配置到文件
func (c *Config) SaveConfig(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 如果启用自动检测，可以没有设备ID
	if !c.AutoDetect && c.DeviceInstanceID == "" {
		return fmt.Errorf("device_instance_id 不能为空（或启用 auto_detect）")
	}
	if c.WaitSeconds < 0 {
		return fmt.Errorf("wait_seconds 必须为非负数")
	}
	return nil
}

// ValidateDevice 验证设备是否仍然存在
func (c *Config) ValidateDevice() error {
	if c.DeviceInstanceID == "" {
		return fmt.Errorf("未配置设备")
	}

	dm := NewDeviceManager(c.DeviceInstanceID)
	_, err := dm.GetStatus()
	return err
}

// SetDevice 设置主设备
func (c *Config) SetDevice(dev *DeviceInfo) {
	c.DeviceInstanceID = dev.InstanceID
	c.DeviceName = dev.FriendlyName
}

// AddBackupDevice 添加备选设备
func (c *Config) AddBackupDevice(instanceID string) {
	// 避免重复
	for _, id := range c.BackupDevices {
		if id == instanceID {
			return
		}
	}
	c.BackupDevices = append(c.BackupDevices, instanceID)
}

// GetConfigPath 获取默认配置文件路径
func GetConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	dir := filepath.Dir(exe)
	return filepath.Join(dir, "config.json")
}
