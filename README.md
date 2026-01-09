# GPD 触屏自动修复工具

>  专为 GPD 设备设计，自动修复睡眠唤醒后触屏失灵的问题

##  这是什么？

这是一个小工具，可以**自动修复** GPD 设备睡眠唤醒后触屏失灵的问题。

###  为什么会有这个项目？

作为一名 GPD Pocket P4 用户，我发现即使使用了官方最新的 EC 固件和 BIOS，电脑从睡眠中唤醒后，**触摸屏经常无法使用**。每次都要手动去设备管理器禁用再启用触摸屏设备才能恢复。

在 Reddit 上看到其他 GPD 用户也遇到同样的问题。既然这是重复性操作，**为什么不让电脑自动完成呢？** 于是就有了这个项目。

现在，工具会在后台默默守护，每次睡眠唤醒时自动检查并修复触摸屏，**你完全不需要做任何操作**！

##  快速开始

### 第 1 步：下载程序

1. 到 [Releases](../../releases) 页面下载最新版本
2. 解压到任意文件夹

### 第 2 步：运行安装向导

1. **右键点击** `gpd-touch-fix.exe`  选择**"以管理员身份运行"**
2. 执行命令：
   ```powershell
   .\gpd-touch-fix.exe --setup
   ```
3. 按照向导提示操作（检测设备  测试修复  安装服务）

### 第 3 步：完成

就这样！从此以后触摸屏会自动修复，不再需要手动操作。

##  常见问题

**Q: 提示"拒绝访问"怎么办？**  
A: 右键程序选择"以管理员身份运行"

**Q: 如何知道它在工作？**  
A: 让电脑睡眠后唤醒，触摸屏应该能正常工作。查看日志：`.\gpd-touch-fix.exe -show-log`

**Q: 如何卸载？**  
A: 以管理员身份运行：
```powershell
.\gpd-touch-fix.exe -stop
.\gpd-touch-fix.exe -uninstall
```
然后删除文件夹即可。

**Q: 支持哪些设备？**  
A: 理论上支持所有使用 I2C HID 触摸屏的 GPD 设备（Pocket、Win 系列等）

##  其他命令

```powershell
# 查看日志
.\gpd-touch-fix.exe -show-log

# 查看统计
.\gpd-touch-fix.exe -stats

# 查看服务状态
.\gpd-touch-fix.exe -status

# 扫描设备
.\gpd-touch-fix.exe -scan

# 手动修复
.\gpd-touch-fix.exe
```

更多命令和配置选项请查看 `.\gpd-touch-fix.exe -h`

##  系统要求

- Windows 10/11
- 管理员权限

##  贡献

欢迎提交 Issue 和 Pull Request！

### 开发者快速入门

```powershell
# 克隆仓库
git clone https://github.com/gpd-touch/gpd-touch-fix.git
cd gpd-touch-fix

# 运行完整检查（lint + test + build）
make all

# 或单独运行
make lint      # 代码检查
make test      # 运行测试
make build     # 构建可执行文件
```

更多信息请查看：
- [贡献指南](CONTRIBUTING.md) - 如何参与贡献
- [GoReleaser 使用文档](docs/GORELEASER.md) - 如何发布新版本（维护者）

##  许可证

MIT License - 详见 [LICENSE](LICENSE)

---

**如果这个工具帮到你了，请给个 Star！**
