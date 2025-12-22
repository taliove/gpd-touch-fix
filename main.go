package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	// 检查是否以服务模式运行
	if isService() {
		if err := runService(); err != nil {
			log.Fatalf("服务运行失败: %v", err)
		}
		return
	}

	// 定义命令行参数
	instanceID := flag.String("instance", "", "I2C HID 设备的 InstanceId")
	configPath := flag.String("config", "", "配置文件路径（默认为可执行文件同目录的 config.json）")
	waitSeconds := flag.Int("wait", 2, "禁用后等待的秒数")
	checkOnly := flag.Bool("check", false, "仅检查设备状态，不执行重置")
	saveConfig := flag.Bool("save-config", false, "保存当前参数到配置文件")

	// 新增功能
	setup := flag.Bool("setup", false, "运行安装向导（自动检测设备并配置）")
	scanDevices := flag.Bool("scan", false, "扫描并列出所有 I2C HID 设备")
	version := flag.Bool("version", false, "显示版本信息")

	// 服务管理命令
	install := flag.Bool("install", false, "安装为 Windows 服务")
	uninstall := flag.Bool("uninstall", false, "卸载 Windows 服务")
	start := flag.Bool("start", false, "启动 Windows 服务")
	stop := flag.Bool("stop", false, "停止 Windows 服务")

	// 新增状态和日志命令
	showStatus := flag.Bool("status", false, "显示服务状态和统计信息")
	showLog := flag.Bool("show-log", false, "显示服务日志")
	showStats := flag.Bool("stats", false, "显示统计信息")
	logLines := flag.Int("lines", 20, "显示日志行数（与 -show-log 配合使用）")

	// 通知控制命令
	enableNotify := flag.Bool("enable-notification", false, "启用 Windows 通知")
	disableNotify := flag.Bool("disable-notification", false, "禁用 Windows 通知")

	flag.Parse()

	// 显示版本信息
	if *version {
		fmt.Println(GetVersionInfo().String())
		return
	}

	// 扫描设备
	if *scanDevices {
		runScanDevices()
		return
	}

	// 安装向导
	if *setup {
		runSetupWizard()
		return
	}

	// 显示状态
	if *showStatus {
		runShowStatus()
		return
	}

	// 显示日志
	if *showLog {
		runShowLog(*logLines)
		return
	}

	// 显示统计
	if *showStats {
		runShowStats()
		return
	}

	// 通知控制
	if *enableNotify {
		runSetNotification(true)
		return
	}
	if *disableNotify {
		runSetNotification(false)
		return
	}

	// 处理服务管理命令
	if *install {
		if err := installService(); err != nil {
			log.Fatalf("安装服务失败: %v", err)
		}
		return
	}
	if *uninstall {
		if err := uninstallService(); err != nil {
			log.Fatalf("卸载服务失败: %v", err)
		}
		return
	}
	if *start {
		if err := startService(); err != nil {
			log.Fatalf("启动服务失败: %v", err)
		}
		return
	}
	if *stop {
		if err := stopService(); err != nil {
			log.Fatalf("停止服务失败: %v", err)
		}
		return
	}

	// 确定配置文件路径
	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = GetConfigPath()
	}

	// 尝试从配置文件加载
	var cfg *Config
	if _, statErr := os.Stat(cfgPath); statErr == nil {
		var loadErr error
		cfg, loadErr = LoadConfig(cfgPath)
		if loadErr != nil {
			log.Printf("警告: 加载配置文件失败: %v", loadErr)
			cfg = DefaultConfig()
		} else {
			log.Printf("已从配置文件加载: %s", cfgPath)
		}
	} else {
		cfg = DefaultConfig()
	}

	// 命令行参数覆盖配置文件
	if *instanceID != "" {
		cfg.DeviceInstanceID = *instanceID
	}
	if *waitSeconds != 2 {
		cfg.WaitSeconds = *waitSeconds
	}

	// 保存配置模式
	if *saveConfig {
		if err := cfg.Validate(); err != nil {
			log.Fatalf("配置验证失败: %v", err)
		}
		if err := cfg.SaveConfig(cfgPath); err != nil {
			log.Fatalf("保存配置失败: %v", err)
		}
		log.Printf("配置已保存到: %s", cfgPath)
		return
	}

	// 验证配置 - 如果设备 ID 为空，尝试自动检测
	if cfg.DeviceInstanceID == "" {
		log.Println("未配置设备，尝试自动检测...")

		detector := NewDetector()
		bestMatch, _, err := detector.DetectBestMatch()
		if err != nil {
			log.Printf("自动检测失败: %v", err)
			log.Fatalf("请使用 -setup 运行安装向导，或使用 -instance 参数手动指定设备 InstanceId")
		}

		cfg.DeviceInstanceID = bestMatch.InstanceID
		cfg.DeviceName = bestMatch.FriendlyName
		log.Printf("自动检测到设备: %s", bestMatch.FriendlyName)
		log.Printf("设备 ID: %s", bestMatch.InstanceID)

		// 提示用户保存配置
		log.Println("提示: 运行 -save-config 可保存此配置")
	}

	// 创建设备管理器
	dm := NewDeviceManager(cfg.DeviceInstanceID)

	// 仅检查模式
	if *checkOnly {
		status, err := dm.GetStatus()
		if err != nil {
			log.Fatalf("检查设备状态失败: %v", err)
		}
		fmt.Printf("设备状态: %s\n", status)
		return
	}

	// 检查管理员权限
	if !IsAdmin() {
		log.Println("警告: 未以管理员身份运行！")
		log.Println("设备操作需要管理员权限。请右键程序 → '以管理员身份运行'")
		os.Exit(1)
	}

	// 执行设备重置
	log.Println("GPD 触屏恢复工具")
	log.Println("==================")
	waitDuration := time.Duration(cfg.WaitSeconds) * time.Second
	if err := dm.Reset(waitDuration); err != nil {
		log.Fatalf("设备重置失败: %v", err)
	}

	log.Println("==================")
	log.Println("触屏设备已成功重置！")
}

// runScanDevices 扫描并列出设备
func runScanDevices() {
	cli := NewCLI()
	cli.PrintTitle("扫描 I2C HID 设备")

	detector := NewDetector()
	cli.PrintProgress("正在扫描设备...")

	devices, err := detector.DetectI2CHIDDevices()
	if err != nil {
		cli.PrintError("扫描失败: %v", err)
		return
	}

	fmt.Println() // 换行

	if len(devices) == 0 {
		cli.PrintWarning("未找到任何 I2C HID 设备")
		return
	}

	cli.PrintInfo("找到 %d 个设备", len(devices))
	cli.PrintDeviceList(devices)
}

// runSetupWizard 运行安装向导
func runSetupWizard() {
	cli := NewCLI()

	cli.PrintTitle("GPD 触屏修复工具 - 安装向导")
	fmt.Println("欢迎使用！本向导将帮助您：")
	fmt.Println("  1. 自动检测触控设备")
	fmt.Println("  2. 测试修复功能")
	fmt.Println("  3. 配置通知选项")
	fmt.Println("  4. 安装自动化服务")
	fmt.Println()

	if !cli.AskYesNo("是否继续", true) {
		cli.PrintInfo("已取消")
		return
	}

	// 步骤1: 检测设备
	cli.PrintTitle("步骤 1/4: 检测设备")
	cli.PrintProgress("正在扫描 I2C HID 设备...")

	detector := NewDetector()
	bestMatch, candidates, err := detector.DetectBestMatch()

	fmt.Println() // 换行

	if err != nil {
		cli.PrintError("检测失败: %v", err)
		cli.PrintInfo("您可以稍后使用 -instance 参数手动指定设备")
		return
	}

	cli.PrintSuccess("找到 %d 个候选设备", len(candidates))

	// 让用户选择设备
	selectedDevice, err := cli.SelectDevice(candidates, bestMatch)
	if err != nil {
		cli.PrintError("选择设备失败: %v", err)
		return
	}

	cli.PrintSuccess("已选择: %s", selectedDevice.FriendlyName)
	fmt.Println()

	// 步骤2: 测试修复
	cli.PrintTitle("步骤 2/4: 测试修复")

	if !cli.AskYesNo("是否测试修复此设备", true) {
		cli.PrintWarning("跳过测试，直接保存配置")
	} else {
		cli.PrintInfo("正在测试设备修复...")

		dm := NewDeviceManager(selectedDevice.InstanceID)
		waitDuration := 2 * time.Second

		if err := dm.Reset(waitDuration); err != nil {
			cli.PrintError("修复失败: %v", err)
			cli.PrintWarning("您仍然可以保存配置，但请检查设备 ID 是否正确")
		} else {
			cli.PrintSuccess("修复成功！触控设备应该已经恢复正常")
		}
		fmt.Println()
	}

	// 保存配置
	cli.PrintInfo("正在保存配置...")

	cfg := DefaultConfig()
	cfg.SetDevice(selectedDevice)

	// 添加其他候选设备作为备选
	for _, dev := range candidates {
		if dev.InstanceID != selectedDevice.InstanceID {
			cfg.AddBackupDevice(dev.InstanceID)
		}
	}

	cfgPath := GetConfigPath()
	if err := cfg.SaveConfig(cfgPath); err != nil {
		cli.PrintError("保存配置失败: %v", err)
		return
	}

	cli.PrintSuccess("配置已保存到: %s", cfgPath)
	fmt.Println()

	// 步骤3: 通知设置
	cli.PrintTitle("步骤 3/4: 通知设置")
	fmt.Println("启用通知后，每次睡眠唤醒时会收到修复结果的 Windows 通知。")
	fmt.Println()

	enableNotification := cli.AskYesNo("是否启用 Windows 通知", true)
	cfg.EnableNotification = enableNotification

	if enableNotification {
		cli.PrintSuccess("已启用通知")
	} else {
		cli.PrintInfo("已禁用通知（可通过 -enable-notification 命令重新启用）")
	}

	// 更新配置
	if err := cfg.SaveConfig(cfgPath); err != nil {
		cli.PrintError("保存配置失败: %v", err)
		return
	}
	fmt.Println()

	// 步骤4: 安装服务
	cli.PrintTitle("步骤 4/4: 安装自动化服务")
	fmt.Println("将程序安装为 Windows 服务后，每次睡眠唤醒都会自动修复触控。")
	fmt.Println()

	if !cli.AskYesNo("是否安装为 Windows 服务", true) {
		cli.PrintInfo("已跳过服务安装")
		cli.PrintInfo("您可以稍后运行以下命令安装服务:")
		fmt.Printf("  %s -install\n", os.Args[0])
		fmt.Printf("  %s -start\n", os.Args[0])
		fmt.Println()
		cli.PrintSuccess("配置完成！")
		return
	}

	// 安装服务
	cli.PrintProgress("正在安装服务...")
	if err := installService(); err != nil {
		fmt.Println() // 换行
		cli.PrintError("安装服务失败: %v", err)
		cli.PrintWarning("请以管理员身份运行本程序")
		return
	}
	fmt.Println() // 换行
	cli.PrintSuccess("服务安装成功")

	// 启动服务
	if cli.AskYesNo("是否立即启动服务", true) {
		cli.PrintProgress("正在启动服务...")
		if err := startService(); err != nil {
			fmt.Println() // 换行
			cli.PrintError("启动服务失败: %v", err)
			return
		}
		fmt.Println() // 换行
		cli.PrintSuccess("服务已启动")
	}

	fmt.Println()
	cli.PrintSuccess("安装完成！")
	fmt.Println()
	fmt.Println("现在您可以:")
	fmt.Println("  • 合上屏幕测试睡眠唤醒后触控是否自动恢复")
	fmt.Println("  • 运行 'gpd-touch-fix -status' 查看服务状态和统计")
	fmt.Println("  • 运行 'gpd-touch-fix -show-log' 查看服务日志")
	fmt.Println()
}

// runShowStatus 显示服务状态和统计信息
func runShowStatus() {
	cli := NewCLI()
	cli.PrintTitle("GPD 触屏修复工具 - 状态")

	// 检查服务状态
	serviceStatus := getServiceStatus()
	fmt.Printf("服务状态: %s\n", serviceStatus)

	// 加载配置
	cfgPath := GetConfigPath()
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		cli.PrintWarning("未找到配置文件")
	} else {
		fmt.Printf("配置文件: %s\n", cfgPath)
		fmt.Printf("监控设备: %s\n", cfg.DeviceName)

		// 检查设备当前状态
		if cfg.DeviceInstanceID != "" {
			dm := NewDeviceManager(cfg.DeviceInstanceID)
			status, err := dm.GetStatus()
			if err != nil {
				fmt.Printf("设备状态: ❌ 无法获取 (%v)\n", err)
			} else if status == "OK" {
				fmt.Printf("设备状态: ✅ %s\n", status)
			} else {
				fmt.Printf("设备状态: ⚠️ %s\n", status)
			}
		}

		// 通知状态
		if cfg.EnableNotification {
			fmt.Println("通知状态: ✅ 已启用")
		} else {
			fmt.Println("通知状态: ❌ 已禁用")
		}
	}

	fmt.Println()

	// 显示统计
	stats := NewStatsManager(GetStatsDir())
	fmt.Print(stats.FormatStats())
}

// runShowLog 显示服务日志
func runShowLog(lines int) {
	cli := NewCLI()
	cli.PrintTitle("GPD 触屏修复工具 - 服务日志")

	logLines, err := ReadLogLines(lines)
	if err != nil {
		cli.PrintWarning("读取日志失败: %v", err)
		cli.PrintInfo("日志目录: %s", GetLogDir())
		return
	}

	fmt.Print(FormatLogForDisplay(logLines))
	fmt.Printf("\n显示最近 %d 行日志，日志目录: %s\n", len(logLines), GetLogDir())
}

// runShowStats 显示统计信息
func runShowStats() {
	cli := NewCLI()
	cli.PrintTitle("GPD 触屏修复工具 - 统计信息")

	stats := NewStatsManager(GetStatsDir())
	fmt.Print(stats.FormatStats())
}

// runSetNotification 设置通知开关
func runSetNotification(enable bool) {
	cli := NewCLI()

	cfgPath := GetConfigPath()
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		cfg = DefaultConfig()
	}

	cfg.EnableNotification = enable

	if err := cfg.SaveConfig(cfgPath); err != nil {
		cli.PrintError("保存配置失败: %v", err)
		return
	}

	if enable {
		cli.PrintSuccess("已启用 Windows 通知")
	} else {
		cli.PrintSuccess("已禁用 Windows 通知")
	}

	cli.PrintInfo("重启服务后生效: gpd-touch-fix -stop && gpd-touch-fix -start")
}

// getServiceStatus 获取服务状态
func getServiceStatus() string {
	output, err := runPowerShell(`$svc = Get-Service -Name "GPDTouchFix" -ErrorAction SilentlyContinue; if ($svc) { $svc.Status } else { "NotInstalled" }`)
	if err != nil {
		return "❓ 未知"
	}

	switch output {
	case "Running":
		return "✅ 运行中"
	case "Stopped":
		return "⏹️ 已停止"
	case "NotInstalled":
		return "❌ 未安装"
	default:
		return fmt.Sprintf("❓ %s", output)
	}
}
