# GPD Touch Fix - Makefile
# 使用方法: make [target]
# Windows 用户需要安装 GNU Make (通过 Git Bash, MinGW, 或 choco install make)

.PHONY: all build test lint coverage clean help

# 默认目标：lint + test + build
all: lint test build

# 构建可执行文件
build:
	@echo "Building..."
	go build -o bin/gpd-touch-fix.exe -ldflags="-s -w"
	@echo "Build complete: bin/gpd-touch-fix.exe"

# 运行所有测试
test:
	@echo "Running tests..."
	go test -v ./...

# 运行 lint 检查
lint:
	@echo "Running linter..."
	golangci-lint run

# 生成测试覆盖率报告
coverage:
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# 显示覆盖率百分比
coverage-percent:
	@go test -cover ./... | grep -E "coverage:" || echo "No coverage data"

# 格式化代码
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 代码静态检查 (go vet)
vet:
	@echo "Running go vet..."
	go vet ./...

# 清理构建产物
clean:
	@echo "Cleaning..."
	@if exist bin rmdir /s /q bin 2>nul || rm -rf bin
	@if exist dist rmdir /s /q dist 2>nul || rm -rf dist
	@if exist coverage.out del coverage.out 2>nul || rm -f coverage.out
	@if exist coverage.html del coverage.html 2>nul || rm -f coverage.html
	@echo "Clean complete"

# 本地发布测试 (使用 GoReleaser snapshot)
release-test:
	@echo "Testing release build..."
	goreleaser check
	goreleaser release --snapshot --clean

# 安装开发依赖
deps:
	@echo "Installing dependencies..."
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 显示帮助信息
help:
	@echo "Available targets:"
	@echo "  all              - Run lint, test, and build (default)"
	@echo "  build            - Build the executable"
	@echo "  test             - Run all tests"
	@echo "  lint             - Run golangci-lint"
	@echo "  coverage         - Generate HTML coverage report"
	@echo "  coverage-percent - Show coverage percentage"
	@echo "  fmt              - Format code with go fmt"
	@echo "  vet              - Run go vet"
	@echo "  clean            - Remove build artifacts"
	@echo "  release-test     - Test GoReleaser build locally"
	@echo "  deps             - Install development dependencies"
	@echo "  help             - Show this help message"
