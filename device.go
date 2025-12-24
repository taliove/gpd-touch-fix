// Package main provides device management functionality for disabling and enabling devices.
package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// DeviceManager 管理设备操作
type DeviceManager struct {
	instanceID string
}

// NewDeviceManager 创建设备管理器
func NewDeviceManager(instanceID string) *DeviceManager {
	return &DeviceManager{instanceID: instanceID}
}

// GetStatus 获取设备当前状态
func (dm *DeviceManager) GetStatus() (string, error) {
	script := fmt.Sprintf("(Get-PnpDevice -InstanceId '%s').Status", escapeSingleQuotes(dm.instanceID))
	output, err := runPowerShell(script)
	if err != nil {
		return "", fmt.Errorf("获取设备状态失败: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// Disable 禁用设备
func (dm *DeviceManager) Disable() error {
	log.Printf("正在禁用设备: %s", dm.instanceID)
	script := fmt.Sprintf("Disable-PnpDevice -InstanceId '%s' -Confirm:$false", escapeSingleQuotes(dm.instanceID))
	_, err := runPowerShell(script)
	if err != nil {
		// 检查是否是权限问题
		if strings.Contains(err.Error(), "0x80041001") || strings.Contains(err.Error(), "常规故障") {
			return fmt.Errorf("禁用设备失败（需要管理员权限）: %w\n\n请以管理员身份运行此程序", err)
		}
		return fmt.Errorf("禁用设备失败: %w", err)
	}
	log.Println("设备已禁用")
	return nil
}

// Enable 启用设备
func (dm *DeviceManager) Enable() error {
	log.Printf("正在启用设备: %s", dm.instanceID)
	script := fmt.Sprintf("Enable-PnpDevice -InstanceId '%s' -Confirm:$false", escapeSingleQuotes(dm.instanceID))
	_, err := runPowerShell(script)
	if err != nil {
		return fmt.Errorf("启用设备失败: %w", err)
	}
	log.Println("设备已启用")
	return nil
}

// Reset 重置设备（禁用后再启用）
func (dm *DeviceManager) Reset(waitDuration time.Duration) error {
	log.Println("开始重置设备...")

	// 获取初始状态
	initialStatus, err := dm.GetStatus()
	if err != nil {
		log.Printf("警告: 无法获取初始状态: %v", err)
	} else {
		log.Printf("初始状态: %s", initialStatus)
	}

	// 禁用设备
	if err := dm.Disable(); err != nil {
		return err
	}

	// 等待
	log.Printf("等待 %v...", waitDuration)
	time.Sleep(waitDuration)

	// 启用设备
	if err := dm.Enable(); err != nil {
		return err
	}

	// 验证最终状态
	finalStatus, err := dm.GetStatus()
	if err != nil {
		log.Printf("警告: 无法获取最终状态: %v", err)
	} else {
		log.Printf("最终状态: %s", finalStatus)
	}

	log.Println("设备重置完成")
	return nil
}

// runPowerShell 执行 PowerShell 命令
func runPowerShell(body string) (string, error) {
	full := fmt.Sprintf("[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; $ErrorActionPreference='Stop'; %s", body)

	// 防止 Disable/Enable-PnpDevice 偶发卡死导致服务长时间阻塞
	const timeout = 90 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", full)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("PowerShell 执行超时(%v): %s", timeout, strings.TrimSpace(string(out)))
	}
	if err != nil {
		return "", fmt.Errorf("PowerShell 执行失败: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// escapeSingleQuotes 转义单引号
func escapeSingleQuotes(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
