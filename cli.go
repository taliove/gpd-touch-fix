// Package main provides command-line interface utilities and interactive dialogs.
// It handles colored output, user input, device selection, and progress indicators.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ANSI 颜色代码（Windows 10+ 支持）
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

var (
	kernel32                        = syscall.NewLazyDLL("kernel32.dll")
	procGetStdHandle                = kernel32.NewProc("GetStdHandle")
	procSetConsoleMode              = kernel32.NewProc("SetConsoleMode")
	procGetConsoleMode              = kernel32.NewProc("GetConsoleMode")
	enableVirtualTerminalProcessing = uint32(0x0004)
	stdOutputHandle                 = uint32(0xFFFFFFF5) // STD_OUTPUT_HANDLE
)

// CLI 交互式命令行界面
type CLI struct {
	reader       *bufio.Reader
	colorEnabled bool
}

// NewCLI 创建 CLI
func NewCLI() *CLI {
	cli := &CLI{
		reader:       bufio.NewReader(os.Stdin),
		colorEnabled: enableWindowsANSI(),
	}
	return cli
}

// enableWindowsANSI 启用 Windows 终端 ANSI 颜色支持
func enableWindowsANSI() bool {
	handle, _, _ := procGetStdHandle.Call(uintptr(stdOutputHandle))
	var mode uint32
	procGetConsoleMode.Call(handle, uintptr(unsafe.Pointer(&mode)))
	mode |= enableVirtualTerminalProcessing
	ret, _, _ := procSetConsoleMode.Call(handle, uintptr(mode))
	return ret != 0
}

// colorize 给文本添加颜色
func (c *CLI) colorize(color, text string) string {
	if c.colorEnabled {
		return color + text + ColorReset
	}
	return text
}

// PrintSuccess 打印成功消息
func (c *CLI) PrintSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(c.colorize(ColorGreen, "✓ "+msg))
}

// PrintError 打印错误消息
func (c *CLI) PrintError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(c.colorize(ColorRed, "✗ "+msg))
}

// PrintWarning 打印警告消息
func (c *CLI) PrintWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(c.colorize(ColorYellow, "⚠ "+msg))
}

// PrintInfo 打印信息消息
func (c *CLI) PrintInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(c.colorize(ColorCyan, "ℹ "+msg))
}

// PrintTitle 打印标题
func (c *CLI) PrintTitle(title string) {
	fmt.Println()
	fmt.Println(c.colorize(ColorBold+ColorBlue, "=== "+title+" ==="))
	fmt.Println()
}

// PrintProgress 打印进度消息
func (c *CLI) PrintProgress(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Print(c.colorize(ColorCyan, "► "+msg))
}

// AskYesNo 询问是/否问题
func (c *CLI) AskYesNo(question string, defaultYes bool) bool {
	prompt := " [Y/n]: "
	if !defaultYes {
		prompt = " [y/N]: "
	}

	fmt.Print(c.colorize(ColorYellow, "? "+question) + prompt)

	input, err := c.reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes
	}

	return input == "y" || input == "yes"
}

// AskInput 询问输入
func (c *CLI) AskInput(question string, defaultValue string) string {
	prompt := ": "
	if defaultValue != "" {
		prompt = fmt.Sprintf(" [%s]: ", defaultValue)
	}

	fmt.Print(c.colorize(ColorYellow, "? "+question) + prompt)

	input, err := c.reader.ReadString('\n')
	if err != nil || strings.TrimSpace(input) == "" {
		return defaultValue
	}

	return strings.TrimSpace(input)
}

// SelectDevice 交互式选择设备
func (c *CLI) SelectDevice(devices []*DeviceInfo, bestMatch *DeviceInfo) (*DeviceInfo, error) {
	if len(devices) == 0 {
		return nil, fmt.Errorf("没有可选择的设备")
	}

	// 只有一个设备的情况
	if len(devices) == 1 {
		dev := devices[0]
		fmt.Println()
		fmt.Println(c.colorize(ColorBold, "检测到 1 个设备:"))
		fmt.Println()

		deviceName := dev.FriendlyName
		if deviceName == "" {
			deviceName = dev.Description
		}

		statusIcon := "○"
		statusColor := ColorWhite
		if dev.IsError() {
			statusIcon = "⚠"
			statusColor = ColorRed
		} else if dev.Status == "OK" {
			statusIcon = "✓"
			statusColor = ColorGreen
		}

		fmt.Printf("  %s %s\n", c.colorize(statusColor, statusIcon), deviceName)
		if dev.Status != "" {
			fmt.Printf("  状态: %s\n", c.colorize(statusColor, dev.Status))
		}
		fmt.Printf("  设备ID: %s\n", dev.InstanceID)
		fmt.Println()

		// 单设备也需要确认
		if c.AskYesNo("是否使用此设备", true) {
			return dev, nil
		}
		return nil, fmt.Errorf("用户取消选择")
	}

	fmt.Println()
	fmt.Println(c.colorize(ColorBold, "检测到以下设备:"))
	fmt.Println()

	for i, dev := range devices {
		prefix := fmt.Sprintf("  [%d] ", i+1)

		// 状态图标
		statusIcon := "○"
		statusColor := ColorWhite
		if dev.IsError() {
			statusIcon = "⚠"
			statusColor = ColorRed
		} else if dev.Status == "OK" {
			statusIcon = "✓"
			statusColor = ColorGreen
		}

		// 标记推荐设备
		recommended := ""
		if bestMatch != nil && dev.InstanceID == bestMatch.InstanceID {
			recommended = c.colorize(ColorGreen+ColorBold, " [推荐]")
		}

		// 设备名称
		deviceName := dev.FriendlyName
		if deviceName == "" {
			deviceName = dev.Description
		}

		fmt.Printf("%s%s %s%s\n",
			prefix,
			c.colorize(statusColor, statusIcon),
			deviceName,
			recommended)

		// 显示状态
		if dev.Status != "" {
			fmt.Printf("      状态: %s\n", c.colorize(statusColor, dev.Status))
		}

		// 显示匹配度
		score := dev.Score()
		if score > 0 {
			fmt.Printf("      匹配度: %d 分\n", score)
		}

		// 显示设备 ID（截断显示）
		instanceID := dev.InstanceID
		if len(instanceID) > 50 {
			instanceID = instanceID[:47] + "..."
		}
		fmt.Printf("      ID: %s\n", c.colorize(ColorWhite, instanceID))
	}

	fmt.Println()

	// 默认选择推荐设备
	defaultChoice := 1
	if bestMatch != nil {
		for i, dev := range devices {
			if dev.InstanceID == bestMatch.InstanceID {
				defaultChoice = i + 1
				break
			}
		}
	}

	// 询问用户选择
	for {
		input := c.AskInput(
			fmt.Sprintf("请选择设备 (1-%d)", len(devices)),
			strconv.Itoa(defaultChoice))

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(devices) {
			c.PrintError("无效的选择，请输入 1-%d", len(devices))
			continue
		}

		return devices[choice-1], nil
	}
}

// PrintDeviceList 打印设备列表
func (c *CLI) PrintDeviceList(devices []*DeviceInfo) {
	if len(devices) == 0 {
		c.PrintWarning("未找到任何设备")
		return
	}

	for i, dev := range devices {
		fmt.Println()
		fmt.Printf("设备 #%d\n", i+1)
		fmt.Println(strings.Repeat("-", 50))

		statusIcon := "○"
		statusColor := ColorWhite
		if dev.IsError() {
			statusIcon = "⚠"
			statusColor = ColorRed
		} else if dev.Status == "OK" {
			statusIcon = "✓"
			statusColor = ColorGreen
		}

		fmt.Printf("名称: %s\n", dev.FriendlyName)
		fmt.Printf("状态: %s %s\n", c.colorize(statusColor, statusIcon), dev.Status)
		fmt.Printf("实例ID: %s\n", dev.InstanceID)

		if dev.Class != "" {
			fmt.Printf("类别: %s\n", dev.Class)
		}
		if dev.Manufacturer != "" {
			fmt.Printf("制造商: %s\n", dev.Manufacturer)
		}

		score := dev.Score()
		if score > 0 {
			fmt.Printf("匹配度: %d 分\n", score)
		}
	}
	fmt.Println()
}

// ShowProgress 显示进度动画
func (c *CLI) ShowProgress(msg string, done chan bool) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	fmt.Print(msg + " ")
	for {
		select {
		case <-done:
			fmt.Print("\r" + strings.Repeat(" ", len(msg)+10) + "\r")
			return
		default:
			fmt.Printf("\r%s %s", msg, c.colorize(ColorCyan, frames[i%len(frames)]))
			i++
			// time.Sleep(100 * time.Millisecond) // 需要导入time包
		}
	}
}

// IsAdmin 检查当前进程是否以管理员身份运行
func IsAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

// CheckAdminAndWarn 检查管理员权限并给出警告
func (c *CLI) CheckAdminAndWarn() bool {
	if !IsAdmin() {
		c.PrintWarning("未以管理员身份运行！")
		c.PrintInfo("设备操作需要管理员权限。")
		c.PrintInfo("请右键程序 → '以管理员身份运行'")
		fmt.Println()
		return false
	}
	return true
}

// RunElevated 以管理员权限重新运行当前程序
// 返回 true 表示已成功请求提升（当前进程应退出）
// 返回 false 表示提升失败或用户取消
func RunElevated() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}

	// 构建参数字符串（跳过程序名）
	args := strings.Join(os.Args[1:], " ")

	// 使用 ShellExecute 以 "runas" 方式运行（触发 UAC 提示）
	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	argsPtr, _ := syscall.UTF16PtrFromString(args)
	cwdPtr, _ := syscall.UTF16PtrFromString("")

	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecute := shell32.NewProc("ShellExecuteW")

	ret, _, _ := shellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(exePtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		uintptr(unsafe.Pointer(cwdPtr)),
		1, // SW_SHOWNORMAL
	)

	// ShellExecute 返回值 > 32 表示成功
	return ret > 32
}

// EnsureAdmin 确保以管理员权限运行，如果不是则尝试提升
// 返回 true 表示当前已是管理员或已请求提升（当前进程应退出）
// 返回 false 表示提升失败
func EnsureAdmin(cli *CLI) bool {
	if IsAdmin() {
		return false // 已是管理员，不需要退出
	}

	cli.PrintWarning("需要管理员权限才能执行此操作")
	fmt.Println()

	if cli.AskYesNo("是否自动以管理员身份重新运行", true) {
		cli.PrintInfo("正在请求管理员权限...")
		if RunElevated() {
			cli.PrintSuccess("已在新窗口中以管理员身份启动程序")
			cli.PrintInfo("请在新窗口中继续操作")
			return true // 请求提升成功，当前进程应退出
		}
		cli.PrintError("提升权限失败，可能是用户取消了 UAC 提示")
		cli.PrintInfo("请右键程序 → '以管理员身份运行'")
		return false
	}

	cli.PrintInfo("请右键程序 → '以管理员身份运行'")
	return false
}
