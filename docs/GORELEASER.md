# 开发者指南 - 使用 GoReleaser 发布

本项目使用 [GoReleaser](https://goreleaser.com/) 实现自动化发布流程。

## 🚀 快速开始

### 本地开发构建

日常开发时，使用标准的 Go 命令：

```powershell
# 快速构建（开发测试）
go build -o bin/gpd-touch-fix.exe

# 运行程序
.\bin\gpd-touch-fix.exe -version

# 运行测试
go test ./...
```

### 测试 GoReleaser 配置

在推送 tag 前，建议先在本地测试 GoReleaser 配置：

#### 1. 安装 GoReleaser

**方式一：使用 Scoop**
```powershell
scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
scoop install goreleaser
```

**方式二：使用 winget**
```powershell
winget install goreleaser.goreleaser
```

**方式三：使用 Go**
```powershell
go install github.com/goreleaser/goreleaser/v2@latest
```

#### 2. 本地测试构建

```powershell
# 验证配置文件是否正确
goreleaser check

# 本地构建（不发布）- 使用 snapshot 模式
goreleaser release --snapshot --clean

# 查看构建结果
ls dist/
```

构建产物在 `dist/` 目录：
- `gpd-touch-fix-{version}-windows-x64.zip` - 64位版本
- `gpd-touch-fix-{version}-windows-x86.zip` - 32位版本
- `checksums.txt` - SHA256 校验和

## 📦 发布新版本

### 前置条件

✅ 确保你有：
- GitHub 仓库的 push 权限
- 所有测试通过
- CHANGELOG.md 已更新

### 发布步骤

#### 1. 更新 CHANGELOG

编辑 [CHANGELOG.md](../CHANGELOG.md)，在顶部添加新版本的更改记录：

```markdown
## [1.1.0] - 2025-12-22

### 新功能
- 添加了 XXX 功能

### Bug 修复
- 修复了 YYY 问题

### 改进
- 优化了 ZZZ 性能
```

#### 2. 创建 Git tag

```powershell
# 确保在 main 分支
git checkout main
git pull

# 创建带注释的 tag（版本号必须以 v 开头）
git tag -a v1.1.0 -m "Release version 1.1.0"

# 查看 tag
git tag -l

# 推送 tag 到远程仓库
git push origin v1.1.0
```

#### 3. 自动化流程启动

推送 tag 后，GitHub Actions 会自动：

1. ✅ 检出代码（包含完整 Git 历史）
2. ✅ 设置 Go 1.24 环境
3. ✅ 运行所有单元测试（`go test -v ./...`）
4. ✅ 使用 GoReleaser 构建多平台版本
5. ✅ 生成 ZIP 归档包
6. ✅ 计算 SHA256 校验和
7. ✅ 自动生成 Changelog
8. ✅ 创建 GitHub Release
9. ✅ 上传所有构建产物

#### 4. 验证发布

1. 访问 [Releases 页面](https://github.com/gpd-touch/gpd-touch-fix/releases)
2. 确认新版本已创建
3. 检查归档文件（x64 和 x86）
4. 验证 Changelog 内容
5. 下载并测试二进制文件

## 🔧 GoReleaser 配置说明

配置文件：[.goreleaser.yml](../.goreleaser.yml)

### 关键配置项

#### 构建配置
```yaml
builds:
  - ldflags:
      - -s -w                           # 减小二进制体积
      - -X main.Version={{.Version}}    # 注入版本号
      - -X main.GitCommit={{.ShortCommit}}  # 注入 Git 提交
      - -X main.BuildTime={{.Date}}     # 注入构建时间
    goos: [windows]
    goarch: [amd64, "386"]              # 支持 64 位和 32 位
```

#### 归档配置
```yaml
archives:
  - format: zip                         # Windows 使用 ZIP 格式
    files:
      - config.example.json             # 包含配置示例
      - README.md                       # 包含使用文档
      - LICENSE                         # 包含许可证
      - CHANGELOG.md                    # 包含更新日志
```

#### Changelog 配置
```yaml
changelog:
  use: github                           # 使用 GitHub API
  groups:
    - title: 🚀 新功能
      regexp: '^.*?feat.*'
    - title: 🐛 Bug 修复
      regexp: '^.*?fix.*'
    - title: 📝 文档更新
      regexp: '^.*?docs.*'
```

## 📋 版本号规范

遵循 [语义化版本 2.0.0](https://semver.org/lang/zh-CN/)：

```
v主版本号.次版本号.修订号

例如：v1.2.3
```

- **主版本号（Major）**：不兼容的 API 修改
  - 例：`v1.0.0` → `v2.0.0`
- **次版本号（Minor）**：向下兼容的功能性新增
  - 例：`v1.0.0` → `v1.1.0`
- **修订号（Patch）**：向下兼容的问题修正
  - 例：`v1.0.0` → `v1.0.1`

## ⚠️ 常见问题

### Q: 如何删除错误的 tag？

```powershell
# 删除本地 tag
git tag -d v1.0.0

# 删除远程 tag
git push origin :refs/tags/v1.0.0
```

### Q: GitHub Actions 构建失败怎么办？

1. 查看 [Actions 页面](https://github.com/gpd-touch/gpd-touch-fix/actions)
2. 点击失败的工作流查看日志
3. 常见原因：
   - 测试未通过
   - 配置文件语法错误
   - 权限不足

### Q: 如何修改已发布的 Release？

```powershell
# 删除远程 tag
git push origin :refs/tags/v1.0.0

# 在 GitHub 上手动删除 Release

# 重新创建 tag 并推送
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

### Q: 本地测试构建失败？

```powershell
# 确保 Go 环境正确
go version  # 应显示 1.24+

# 清理并重试
go clean -cache
goreleaser release --snapshot --clean --verbose
```

## 🔍 调试技巧

### 查看详细构建日志

```powershell
# 本地构建时显示详细日志
goreleaser release --snapshot --clean --debug

# 或查看 GitHub Actions 日志
# 访问 https://github.com/{owner}/{repo}/actions
```

### 验证配置文件

```powershell
# 检查 .goreleaser.yml 语法
goreleaser check

# 查看将要执行的构建配置
goreleaser build --snapshot --single-target
```

## 📚 更多资源

- [GoReleaser 官方文档](https://goreleaser.com/)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [语义化版本规范](https://semver.org/lang/zh-CN/)
- [项目贡献指南](../CONTRIBUTING.md)

## 💡 最佳实践

1. **发布前务必测试**
   - 本地运行 `goreleaser release --snapshot --clean`
   - 测试生成的二进制文件

2. **编写清晰的 Changelog**
   - 遵循提交信息规范（feat/fix/docs）
   - 在 CHANGELOG.md 中补充详细说明

3. **使用语义化版本**
   - 主版本号：破坏性变更
   - 次版本号：新功能
   - 修订号：Bug 修复

4. **保持 tag 整洁**
   - 不要随意推送 tag
   - 发布前仔细检查版本号

5. **监控构建状态**
   - 推送 tag 后查看 GitHub Actions
   - 确保构建成功完成
