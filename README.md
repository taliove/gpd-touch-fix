# GPD 触屏恢复器

> 🚀 全自动解决 GPD 设备睡眠唤醒后触控失效问题

##问题描述

GPD 设备的 I2C HID 触控在睡眠唤醒后常常出现异常（设备管理器中显示感叹号⚠），触摸功能失效。手动在设备管理器中禁用该设备后再重新启用可以恢复正常。

本工具提供了**完全自动化**的解决方案：自动识别硬件、一键修复、后台服务守护。

## ✨ 功能特性

- 🔍 **智能硬件识别** - 自动扫描并识别 I2C HID 触控设备，无需手动查找
- 🎯 **一键修复** - 简单运行即可修复触控问题
- � **智能检测** - 唤醒后先检查状态，正常则跳过修复，异常才执行
- 🤖 **安装向导** - 交互式引导完成全部配置（检测→测试→通知→服务）
- 🛡️ **Windows 服务** - 后台守护，睡眠唤醒后自动修复
- 🔔 **Windows 通知** - 修复结果实时推送，可自由开关
- 📊 **统计面板** - 今日/本周/本月/累计修复统计
- 📝 **日志系统** - 详细日志记录，方便故障排查
- ⚙️ **配置管理** - 支持多设备、自动检测、备选设备
- ✅ **完整测试** - 20+ 单元测试保证代码质量
- 🎨 **彩色终端** - 友好的交互界面和进度提示

## 🚀 快速开始

### 方式一：安装向导（推荐）

以**管理员身份**运行 PowerShell，执行：

```powershell
# 1. 下载或编译程序
.\build.ps1

# 2. 运行安装向导
.\bin\gpd-touch-fix.exe --setup
```

向导会自动完成：
1. **检测设备** - 扫描并智能识别触控设备
2. **测试修复** - 验证修复功能是否正常
3. **安装服务** - 配置后台自动化

**就这么简单！** 🎉

### 方式二：手动配置

```powershell
# 扫描设备
.\gpd-touch-fix.exe -scan

# 一键修复（使用自动检测）
.\gpd-touch-fix.exe

# 手动指定设备并保存配置
.\gpd-touch-fix.exe -instance "ACPI\VEN_INT&DEV_0B45&..." -save-config

# 安装为服务
.\gpd-touch-fix.exe -install
.\gpd-touch-fix.exe -start
```

## 📖 使用指南

### 命令行参数

```
基础功能:
  -setup            运行安装向导（推荐首次使用）
  -scan             扫描并列出所有 I2C HID 设备
  -version          显示版本信息
  -status           显示服务状态和统计信息
  -stats            显示统计信息
  -show-log         显示服务日志
  -lines int        显示日志行数（与 -show-log 配合，默认 20）

设备操作:
  -instance string  指定设备 InstanceId
  -check            仅检查设备状态
  -wait int         禁用后等待秒数（默认 2）

配置管理:
  -config string    配置文件路径
  -save-config      保存当前参数到配置

通知控制:
  -enable-notification   启用 Windows 通知
  -disable-notification  禁用 Windows 通知

服务管理:
  -install          安装 Windows 服务
  -uninstall        卸载服务
  -start            启动服务
  -stop             停止服务
```

### 配置文件

配置文件会自动生成在程序同目录的 `config.json`：

```json
{
  "device_instance_id": "ACPI\\VEN_INT&DEV_0B45\\...",
  "device_name": "HID-compliant touch screen",
  "wait_seconds": 2,
  "backup_devices": [
    "ACPI\\VEN_SYNA&DEV_7813\\..."
  ],
  "auto_detect": true,
  "log_level": "INFO",
  "log_dir": "",
  "check_before_reset": true,
  "resume_delay_seconds": 3,
  "log_all_events": true,
  "enable_notification": true,
  "max_log_days": 30
}
```

**字段说明**：
- `device_instance_id` - 主设备实例 ID
- `device_name` - 设备友好名称（仅用于显示）
- `wait_seconds` - 禁用后等待时间
- `backup_devices` - 备选设备列表
- `auto_detect` - 启用时自动检测设备
- `log_level` - 日志级别（DEBUG/INFO/WARNING/ERROR）
- `log_dir` - 日志文件目录
- `check_before_reset` - **智能检测**: 修复前先检查设备状态，正常则跳过
- `resume_delay_seconds` - 睡眠唤醒后等待秒数
- `log_all_events` - 记录所有事件（包括跳过的）
- `enable_notification` - 启用 Windows 通知
- `max_log_days` - 日志保留天数

### 工作原理

#### 自动检测算法

程序通过 PowerShell 扫描系统设备，根据以下规则评分：

| 特征 | 分数 |
|------|------|
| I2C HID 设备 | +10 |
| 触控设备（touch/digitizer） | +20 |
| 错误状态（带感叹号） | +30 |
| HID 类设备 | +5 |

**最高分设备**会被推荐为目标设备。

#### 修复流程

1. **禁用设备** - 调用 `Disable-PnpDevice`
2. **等待稳定** - 默认等待 2 秒，让系统释放驱动资源
3. **启用设备** - 调用 `Enable-PnpDevice`
4. **验证结果** - 查询设备状态确认成功

#### 服务模式

Windows 服务监听电源事件（PowerEvent ID 18 = ResumeAutomatic，ID 7 = ResumeSuspend），在检测到系统从睡眠恢复后：

1. 等待系统稳定（默认 3 秒，可配置）
2. **智能检测**：检查设备状态
   - 状态正常 → 跳过修复，记录日志
   - 状态异常 → 执行修复
3. 执行设备重置（如需要）
4. 验证修复结果
5. 发送 Windows 通知（如已启用）
6. 更新统计数据

### 📊 日志与统计

#### 查看日志

```powershell
# 查看最近 20 行日志
.\gpd-touch-fix.exe -show-log

# 查看最近 50 行日志
.\gpd-touch-fix.exe -show-log -lines 50
```

日志示例：
```
2025-12-20 10:30:15.123 [INFO ] [RESUME ] 系统从睡眠唤醒
2025-12-20 10:30:18.456 [INFO ] [CHECK  ] 设备状态: OK
2025-12-20 10:30:18.457 [INFO ] [SKIP   ] 设备状态正常，无需修复

2025-12-20 14:22:33.789 [INFO ] [RESUME ] 系统从睡眠唤醒
2025-12-20 14:22:36.012 [WARN ] [CHECK  ] 设备状态异常 (Error)，需要修复
2025-12-20 14:22:36.015 [INFO ] [RESET  ] 开始修复设备
2025-12-20 14:22:40.890 [INFO ] [SUCCESS] 触屏设备修复成功
```

#### 查看统计

```powershell
.\gpd-touch-fix.exe -stats
```

输出示例：
```
╔══════════════════════════════════════════╗
║           📊 统计信息                     ║
╠══════════════════════════════════════════╣
║ 📅 今日                                  ║
║    修复: 2    跳过: 5    失败: 0         ║
║ 📆 本周                                  ║
║    修复: 8    跳过: 20   失败: 1         ║
╠══════════════════════════════════════════╣
║ 📈 累计                                  ║
║    唤醒: 156                             ║
║    修复: 45                              ║
║    跳过: 108                             ║
╚══════════════════════════════════════════╝
```

### 🔔 Windows 通知

启用通知后，每次睡眠唤醒会收到修复结果的通知：
- ✅ **触屏已修复** - 设备修复成功
- ℹ️ **触屏状态正常** - 设备正常，无需修复
- ❌ **触屏修复失败** - 修复过程出错

```powershell
# 启用通知
.\gpd-touch-fix.exe -enable-notification

# 禁用通知
.\gpd-touch-fix.exe -disable-notification
```

## 🛠️ 开发与构建

### 项目结构

```
gpd-touch/
├── main.go              # 主程序入口
├── device.go            # 设备操作（禁用/启用/重置）
├── detector.go          # 🆕 自动检测 I2C HID 设备
├── cli.go               # 🆕 交互式命令行界面
├── config.go            # 配置管理（支持多设备）
├── service.go           # Windows 服务实现
├── logger.go            # 🆕 日志系统
├── version.go           # 🆕 版本信息
├── *_test.go            # 单元测试
├── build.ps1            # 🆕 Windows 构建脚本
├── config.example.json  # 配置示例
├── go.mod
└── README.md
```

### 构建

```powershell
# 开发构建
.\build.ps1

# 带测试的构建
.\build.ps1 -Test

# 发布构建（生成 32/64 位版本 + ZIP 包）
.\build.ps1 -Release -Version "1.0.0"

# 清理
.\build.ps1 -Clean
```

**构建产物**：
- `bin/` - 开发版本
- `dist/` - 发布版本（含 ZIP 包）

### 运行测试

```powershell
go test -v              # 运行所有测试
go test -cover          # 测试覆盖率
go test -run TestXXX    # 运行特定测试
```

**测试覆盖**：
- 设备检测逻辑（评分算法、设备分类）
- PowerShell 命令格式
- 配置文件读写
- 参数转义
- 设备 InstanceId 处理

## 📊 使用示例

### 示例 1: 首次安装（向导模式）

```powershell
PS C:\> .\gpd-touch-fix.exe -setup

=== GPD 触屏修复工具 - 安装向导 ===

欢迎使用！本向导将帮助您：
  1. 自动检测触控设备
  2. 测试修复功能
  3. 配置自动化服务

? 是否继续 [Y/n]: y

=== 步骤 1/3: 检测设备 ===

► 正在扫描 I2C HID 设备...

✓ 找到 2 个候选设备

检测到以下设备:

  [1] ✓ HID-compliant device
      状态: OK
      匹配度: 15 分

  [2] ⚠ HID-compliant touch screen [推荐]
      状态: Error
      匹配度: 65 分

? 请选择设备 (1-2) [2]: 2

✓ 已选择: HID-compliant touch screen

=== 步骤 2/3: 测试修复 ===

? 是否测试修复此设备 [Y/n]: y

ℹ 正在测试设备修复...
正在禁用设备: ACPI\VEN_INT&DEV_0B45&...
设备已禁用
等待 2s...
正在启用设备: ACPI\VEN_INT&DEV_0B45&...
设备已启用

✓ 修复成功！触控设备应该已经恢复正常

ℹ 正在保存配置...
✓ 配置已保存到: C:\gpd-touch\config.json

=== 步骤 3/3: 安装自动化服务 ===

将程序安装为 Windows 服务后，每次睡眠唤醒都会自动修复触控。

? 是否安装为 Windows 服务 [Y/n]: y

► 正在安装服务...
✓ 服务安装成功

? 是否立即启动服务 [Y/n]: y

► 正在启动服务...
✓ 服务已启动

✓ 安装完成！

现在您可以:
  • 合上屏幕测试睡眠唤醒后触控是否自动恢复
  • 运行 'sc query GPDTouchFix' 检查服务状态
  • 在事件查看器中查看服务日志
```

### 示例 2: 扫描设备

```powershell
PS C:\> .\gpd-touch-fix.exe -scan

=== 扫描 I2C HID 设备 ===

► 正在扫描设备...

ℹ 找到 3 个设备

设备 #1
--------------------------------------------------
名称: I2C HID Device
状态: ✓ OK
实例ID: ACPI\VEN_INT&DEV_0B45&SUBSYS_12345678&...
类别: HIDClass
制造商: Intel
匹配度: 10 分

设备 #2
--------------------------------------------------
名称: HID-compliant touch screen
状态: ⚠ Error
实例ID: ACPI\VEN_SYNA&DEV_7813&...
类别: HIDClass
匹配度: 65 分

...
```

### 示例 3: 手动修复

```powershell
PS C:\> .\gpd-touch-fix.exe

GPD 触屏恢复工具
==================
2025/12/20 17:30:45 开始重置设备...
2025/12/20 17:30:45 初始状态: Error
2025/12/20 17:30:45 正在禁用设备: ACPI\VEN_INT&DEV_0B45&...
2025/12/20 17:30:45 设备已禁用
2025/12/20 17:30:45 等待 2s...
2025/12/20 17:30:47 正在启用设备: ACPI\VEN_INT&DEV_0B45&...
2025/12/20 17:30:48 设备已启用
2025/12/20 17:30:48 最终状态: OK
2025/12/20 17:30:48 设备重置完成
==================
触屏设备已成功重置！
```

## 🔧 故障排除

### 权限问题

**错误**：`Access is denied`

**解决**：必须以管理员身份运行。右键 → "以管理员身份运行"

### 未找到设备

**错误**：`未找到任何候选设备`

**解决**：
1. 使用 `-scan` 命令查看所有设备
2. 在设备管理器中确认触控设备存在
3. 手动使用 `-instance` 参数指定设备 ID

### 服务未触发

**现象**：服务已安装但睡眠唤醒后不自动修复

**检查**：
1. 服务状态：`sc query GPDTouchFix`
2. 事件日志：事件查看器 → Windows 日志 → 应用程序 → 查找 `GPDTouchFix`
3. 配置文件：确认 `config.json` 中设备 ID 正确

### 自动检测失败

**错误**：`检测失败` 或找不到匹配设备

**解决**：
1. 使用 `-scan` 查看所有候选设备
2. 手动选择正确的设备并用 `-instance` 指定
3. 检查是否有权限访问设备信息

## 📋 系统要求

- Windows 10/11（推荐 Windows 10 1809+以获得更好的ANSI颜色支持）
- PowerShell 5.0+（系统自带）
- 管理员权限

## 🎯 最佳实践

1. **首次使用**：运行 `--setup` 向导，让程序自动配置一切
2. **测试环境**：安装服务前先用 `-instance` 手动测试修复功能
3. **多设备**：如果有多个候选设备，都可以添加到 `backup_devices` 中
4. **日志记录**：设置 `log_dir` 可以持久化日志到文件
5. **版本管理**：使用 `-version` 查看当前版本，方便报告问题

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

**开发流程**：
1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

**代码规范**：
- 使用 `go fmt` 格式化代码
- 为新功能编写单元测试
- 更新 README 文档

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

感谢所有 GPD 设备用户的反馈和建议！

---

**⭐ 如果这个工具帮助到您，请给个 Star！**
