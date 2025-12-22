# 📋 GoReleaser 快速参考

## 日常开发

```powershell
# 编译测试
go build -o bin/gpd-touch-fix.exe

# 运行测试
go test ./...

# 运行程序
.\bin\gpd-touch-fix.exe -version
```

## 发布新版本

```powershell
# 1. 更新 CHANGELOG.md
# 添加新版本的变更说明

# 2. 创建 tag
git tag -a v1.1.0 -m "Release v1.1.0"

# 3. 推送 tag（触发自动发布）
git push origin v1.1.0

# 4. 查看构建状态
# https://github.com/gpd-touch/gpd-touch-fix/actions
```

## 本地测试

```powershell
# 安装 GoReleaser
.\test-release.ps1 -Install

# 验证配置
.\test-release.ps1 -Check

# 测试构建（不发布）
.\test-release.ps1 -Build

# 查看构建产物
ls dist\*.zip
```

## 常用命令

```powershell
# 检查配置
goreleaser check

# 本地构建（snapshot 模式）
goreleaser release --snapshot --clean

# 查看版本
goreleaser --version

# 清理构建产物
Remove-Item dist -Recurse -Force
```

## 版本号规范

```
v主版本.次版本.修订号

v1.0.0 → v2.0.0  # 不兼容的变更
v1.0.0 → v1.1.0  # 新功能
v1.0.0 → v1.0.1  # Bug 修复
```

## 提交信息规范

```
feat: 新功能
fix: Bug 修复
docs: 文档更新
test: 测试相关
refactor: 代码重构
chore: 其他改动
```

## 删除 Tag

```powershell
# 删除本地 tag
git tag -d v1.0.0

# 删除远程 tag
git push origin :refs/tags/v1.0.0

# 删除 GitHub Release（需手动在网页上删除）
# https://github.com/gpd-touch/gpd-touch-fix/releases
```

## 文档链接

- 📖 [完整文档](docs/GORELEASER.md)
- 🤝 [贡献指南](CONTRIBUTING.md)
- 📦 [Releases](https://github.com/gpd-touch/gpd-touch-fix/releases)
- 🔧 [GoReleaser 官方文档](https://goreleaser.com/)
