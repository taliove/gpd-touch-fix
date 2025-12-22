# Contributing to GPD Touch Fix

感谢你对 GPD Touch Fix 项目的关注！以下是如何参与贡献的指南。

## 报告Bug

在提交bug报告前，请检查 [Issues](https://github.com/gpd-touch/gpd-touch-fix/issues) 是否已有相关报告。

提交bug时，请包含以下信息：
- **清晰的描述** - 简明扼要地说明bug的性质
- **复现步骤** - 尽可能详细地描述如何重现问题
- **实际行为** - 描述实际发生了什么
- **预期行为** - 描述应该发生什么
- **系统环境** - 操作系统、Go版本等信息
- **日志** - 相关的日志输出（如果适用）

## 提议新功能

在提出新功能前，请先查看 [Issues](https://github.com/gpd-touch/gpd-touch-fix/issues) 和 [Discussions](https://github.com/gpd-touch/gpd-touch-fix/discussions)。

新功能建议应包含：
- **清晰的动机** - 说明为什么需要这个功能
- **使用案例** - 描述如何使用这个功能
- **可能的实现方案** - 简述如何实现（可选）

## 提交Pull Request

1. **Fork本仓库**
2. **创建分支**：`git checkout -b feature/your-feature-name`
3. **提交改动**：`git commit -am 'Add some feature'`
4. **推送分支**：`git push origin feature/your-feature-name`
5. **创建Pull Request**

### Pull Request要求

- 一个PR应专注于单一功能或修复
- PR标题清晰明了
- PR描述包含为什么做这个改动
- 通过所有测试（`go test ./...`）
- 代码格式规范（`go fmt ./...`）
- 添加适当的测试用例

## 代码规范

### Golang规范

- 遵循 [Effective Go](https://golang.org/doc/effective_go) 的规范
- 使用 `go fmt` 格式化代码
- 使用 `go vet` 检查代码
- 函数和类型需要添加注释说明
- 包含单元测试（目标覆盖率80%+）

### 注释规范

- 包级注释：`// Package xxx ...`
- 导出类型/函数：`// TypeName ...` 或 `// FuncName ...`
- 复杂逻辑：添加清晰的中文或英文注释

### 提交信息规范

使用清晰、简洁的提交信息：

```
feat: add new feature
fix: fix some bug
docs: update documentation
test: add unit tests
refactor: refactor code structure
chore: update dependencies
```

## 开发流程

### 环境要求

- Go 1.21 或更新版本
- Windows 10 或更新版本
- PowerShell 5.1

### 编译和测试

```powershell
# 编译
.\build.ps1

# 测试
go test ./...

# 覆盖率
go test -cover ./...
```

## 行为准则

本项目采纳 Contributor Covenant 行为准则。参与本项目，即表示你同意遵守此准则。

我们致力于为所有贡献者提供积极、包容的环境，无论其背景或身份如何。

不接受的行为包括：
- 性骚扰或歧视性语言
- 人身攻击或侮辱
- 骚扰或霸凌他人
- 发布他人私人信息

## 获取帮助

- 在 [Discussions](https://github.com/gpd-touch/gpd-touch-fix/discussions) 中提问
- 查阅 [README.md](../README.md) 和文档
- 提交 [Issue](https://github.com/gpd-touch/gpd-touch-fix/issues)

感谢你的贡献！🎉
