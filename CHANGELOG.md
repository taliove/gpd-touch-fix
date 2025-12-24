# Changelog

所有对本项目的重要变化都记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/)，项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 规范。

## [Unreleased]

### Added

### Changed

### Fixed

### Removed

## [1.0.0] - 2025-12-22

### Added
- Initial project setup and structure
- Device detection and management functionality
- Windows service integration for automatic recovery
- Statistics tracking and reporting
- Windows toast notification system
- Configuration management with JSON support
- Comprehensive logging system
- CLI interface with setup wizard
- Unit tests with 20+ test cases
- 🔍 **Smart Hardware Detection** - Automatically scan and identify I2C HID touch devices
- 🎯 **One-Click Fix** - Simple execution to fix touchpad issues
- 🤖 **Intelligent Detection** - Check device status after wake-up, skip fix if normal
- 🛡️ **Windows Service** - Background service to auto-fix after sleep/wake
- 🔔 **Windows Notifications** - Real-time fix notifications with toggle control
- 📊 **Statistics Dashboard** - Track daily/weekly/monthly/cumulative fixes
- 📝 **Logging System** - Detailed logging for troubleshooting
- ⚙️ **Configuration Management** - Support multiple devices with auto-detection and fallback
- ✅ **Complete Testing** - 20+ unit tests ensuring code quality
- 🎨 **Colored Terminal** - User-friendly interactive interface

### Changed

### Fixed

### Removed

## [0.1.0] - 2024-12-01

### Added
- Project initialization
- Basic device detection

---

### 说明

- **Added** - 新增的功能
- **Changed** - 现有功能的变化
- **Fixed** - 修复的bug
- **Removed** - 移除的功能
- **Deprecated** - 即将移除的功能

### 格式示例

```
### Added
- 新增功能描述 (#123)
- 另一个新增项

### Fixed
- 修复的bug描述 (#456)

### Breaking Changes
- 不兼容的改动描述
```

### 发布指南

发布新版本时：
1. 更新 [Unreleased] 部分到新版本号和日期
2. 创建新的 [Unreleased] 部分
3. 更新版本号在 `build.ps1` 中
4. 创建Git标签：`git tag -a v1.0.0 -m "Release version 1.0.0"`
