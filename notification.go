package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// NotificationType 通知类型
type NotificationType int

const (
	NotifyInfo NotificationType = iota
	NotifySuccess
	NotifyWarning
	NotifyError
)

// Notifier Windows 通知管理器
type Notifier struct {
	enabled bool
	appName string
}

// NewNotifier 创建通知管理器
func NewNotifier(enabled bool) *Notifier {
	return &Notifier{
		enabled: enabled,
		appName: "GPD 触屏修复工具",
	}
}

// SetEnabled 设置是否启用通知
func (n *Notifier) SetEnabled(enabled bool) {
	n.enabled = enabled
}

// IsEnabled 检查是否启用通知
func (n *Notifier) IsEnabled() bool {
	return n.enabled
}

// Send 发送通知
func (n *Notifier) Send(notifyType NotificationType, title, message string) error {
	if !n.enabled {
		return nil
	}

	return n.sendToast(notifyType, title, message)
}

// SendSuccess 发送成功通知
func (n *Notifier) SendSuccess(title, message string) error {
	return n.Send(NotifySuccess, title, message)
}

// SendError 发送错误通知
func (n *Notifier) SendError(title, message string) error {
	return n.Send(NotifyError, title, message)
}

// SendInfo 发送信息通知
func (n *Notifier) SendInfo(title, message string) error {
	return n.Send(NotifyInfo, title, message)
}

// SendWarning 发送警告通知
func (n *Notifier) SendWarning(title, message string) error {
	return n.Send(NotifyWarning, title, message)
}

// sendToast 使用 PowerShell 发送 Windows Toast 通知
func (n *Notifier) sendToast(notifyType NotificationType, title, message string) error {
	// 根据类型选择图标
	iconHint := ""
	switch notifyType {
	case NotifySuccess:
		iconHint = "ms-winsoundevent:Notification.Default"
	case NotifyError:
		iconHint = "ms-winsoundevent:Notification.Looping.Alarm"
	case NotifyWarning:
		iconHint = "ms-winsoundevent:Notification.Looping.Alarm2"
	default:
		iconHint = "ms-winsoundevent:Notification.Default"
	}

	// 使用 BurntToast 模块（如果可用）或者回退到基础通知
	script := fmt.Sprintf(`
$ErrorActionPreference = 'SilentlyContinue'

# 尝试使用 BurntToast 模块
if (Get-Module -ListAvailable -Name BurntToast) {
    Import-Module BurntToast
    New-BurntToastNotification -Text '%s', '%s' -AppLogo $null
} else {
    # 回退到基础 Windows 通知
    [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
    [Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
    
    $template = @"
<toast>
    <visual>
        <binding template="ToastText02">
            <text id="1">%s</text>
            <text id="2">%s</text>
        </binding>
    </visual>
    <audio src="%s"/>
</toast>
"@
    
    $xml = New-Object Windows.Data.Xml.Dom.XmlDocument
    $xml.LoadXml($template)
    
    $toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
    $notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('%s')
    $notifier.Show($toast)
}
`,
		escapeForPowerShell(title),
		escapeForPowerShell(message),
		escapeForPowerShell(title),
		escapeForPowerShell(message),
		iconHint,
		escapeForPowerShell(n.appName),
	)

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 通知失败不应影响主程序，只记录警告
		return fmt.Errorf("发送通知失败: %w, 输出: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

// escapeForPowerShell 转义 PowerShell 字符串中的特殊字符
func escapeForPowerShell(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	s = strings.ReplaceAll(s, "`", "``")
	s = strings.ReplaceAll(s, "$", "`$")
	return s
}

// NotifyResumeResult 通知睡眠唤醒结果
func (n *Notifier) NotifyResumeResult(fixed bool, skipped bool, deviceName string, err error) {
	if !n.enabled {
		return
	}

	var title, message string

	if err != nil {
		title = "触屏修复失败"
		message = fmt.Sprintf("设备: %s\n错误: %v", deviceName, err)
		n.SendError(title, message)
	} else if skipped {
		title = "触屏状态正常"
		message = fmt.Sprintf("设备 %s 状态正常，无需修复", deviceName)
		n.SendInfo(title, message)
	} else if fixed {
		title = "触屏已修复"
		message = fmt.Sprintf("设备 %s 已成功修复", deviceName)
		n.SendSuccess(title, message)
	}
}
