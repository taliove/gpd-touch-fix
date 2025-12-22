package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DeviceInstanceID != "" {
		t.Errorf("DefaultConfig().DeviceInstanceID = %q, want empty string", cfg.DeviceInstanceID)
	}

	if cfg.WaitSeconds != 2 {
		t.Errorf("DefaultConfig().WaitSeconds = %d, want 2", cfg.WaitSeconds)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "有效配置",
			config: Config{
				DeviceInstanceID: "ACPI\\VEN_INT&DEV_0B45",
				WaitSeconds:      2,
			},
			wantError: false,
		},
		{
			name: "空设备ID",
			config: Config{
				DeviceInstanceID: "",
				WaitSeconds:      2,
			},
			wantError: true,
		},
		{
			name: "负等待时间",
			config: Config{
				DeviceInstanceID: "ACPI\\VEN_INT&DEV_0B45",
				WaitSeconds:      -1,
			},
			wantError: true,
		},
		{
			name: "零等待时间",
			config: Config{
				DeviceInstanceID: "ACPI\\VEN_INT&DEV_0B45",
				WaitSeconds:      0,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Config.Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")

	// 创建测试配置
	original := &Config{
		DeviceInstanceID: "ACPI\\VEN_INT&DEV_0B45&SUBSYS_12345678",
		WaitSeconds:      3,
	}

	// 保存配置
	if err := original.SaveConfig(configPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("配置文件未创建")
	}

	// 加载配置
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// 验证内容
	if loaded.DeviceInstanceID != original.DeviceInstanceID {
		t.Errorf("DeviceInstanceID = %q, want %q", loaded.DeviceInstanceID, original.DeviceInstanceID)
	}
	if loaded.WaitSeconds != original.WaitSeconds {
		t.Errorf("WaitSeconds = %d, want %d", loaded.WaitSeconds, original.WaitSeconds)
	}
}

func TestLoadConfigNotExist(t *testing.T) {
	_, err := LoadConfig("nonexistent_config.json")
	if err == nil {
		t.Error("LoadConfig() 应该在文件不存在时返回错误")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	// 写入无效的 JSON
	if err := os.WriteFile(configPath, []byte("invalid json content"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 尝试加载
	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig() 应该在 JSON 无效时返回错误")
	}
}

func TestConfigJSONFormat(t *testing.T) {
	cfg := &Config{
		DeviceInstanceID: "TEST_ID",
		WaitSeconds:      5,
	}

	// 序列化
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// 验证 JSON 格式
	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	if decoded.DeviceInstanceID != cfg.DeviceInstanceID {
		t.Errorf("JSON 往返后 DeviceInstanceID 不匹配")
	}
	if decoded.WaitSeconds != cfg.WaitSeconds {
		t.Errorf("JSON 往返后 WaitSeconds 不匹配")
	}
}

func TestSaveConfigCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.json")

	cfg := DefaultConfig()
	cfg.DeviceInstanceID = "TEST_ID"

	if err := cfg.SaveConfig(configPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// 验证目录和文件都被创建
	if _, err := os.Stat(filepath.Dir(configPath)); os.IsNotExist(err) {
		t.Error("SaveConfig() 应该创建目录")
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("SaveConfig() 应该创建文件")
	}
}
