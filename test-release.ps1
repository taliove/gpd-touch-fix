# 本地测试脚本 - GoReleaser
# 用于本地测试 GoReleaser 构建，不会真正发布

param(
    [switch]$Install,
    [switch]$Check,
    [switch]$Build,
    [switch]$Clean
)

$ErrorActionPreference = "Stop"

function Write-Success { Write-Host "[OK] $args" -ForegroundColor Green }
function Write-Info { Write-Host "[INFO] $args" -ForegroundColor Cyan }
function Write-Err { Write-Host "[ERROR] $args" -ForegroundColor Red }

# 检查 GoReleaser 是否已安装
function Test-GoReleaser {
    return $null -ne (Get-Command goreleaser -ErrorAction SilentlyContinue)
}

# 安装 GoReleaser
if ($Install) {
    Write-Info "检查 GoReleaser 安装状态..."
    
    if (Test-GoReleaser) {
        Write-Success "GoReleaser 已安装"
        goreleaser --version
    } else {
        Write-Info "GoReleaser 未安装，开始安装..."
        
        # 尝试使用 winget
        if (Get-Command winget -ErrorAction SilentlyContinue) {
            Write-Info "使用 winget 安装 GoReleaser..."
            winget install goreleaser.goreleaser
        }
        # 尝试使用 scoop
        elseif (Get-Command scoop -ErrorAction SilentlyContinue) {
            Write-Info "使用 scoop 安装 GoReleaser..."
            scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
            scoop install goreleaser
        }
        # 使用 Go install
        elseif (Get-Command go -ErrorAction SilentlyContinue) {
            Write-Info "使用 Go install 安装 GoReleaser..."
            go install github.com/goreleaser/goreleaser/v2@latest
        }
        else {
            Write-Err "无法自动安装 GoReleaser"
            Write-Info "请手动安装：https://goreleaser.com/install/"
            exit 1
        }
        
        if (Test-GoReleaser) {
            Write-Success "GoReleaser 安装成功！"
            goreleaser --version
        } else {
            Write-Err "安装失败，请手动安装"
            exit 1
        }
    }
    
    if (!$Check -and !$Build) { exit 0 }
}

# 检查 GoReleaser 是否可用
if (-not (Test-GoReleaser)) {
    Write-Err "GoReleaser 未安装"
    Write-Info "运行以下命令安装："
    Write-Host "  .\test-release.ps1 -Install" -ForegroundColor Cyan
    exit 1
}

# 清理 dist 目录
if ($Clean) {
    Write-Info "清理 dist 目录..."
    if (Test-Path dist) {
        Remove-Item -Recurse -Force dist
        Write-Success "清理完成"
    }
    if (!$Check -and !$Build) { exit 0 }
}

# 检查配置
if ($Check) {
    Write-Info "检查 GoReleaser 配置..."
    goreleaser check
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "配置文件验证通过！"
    } else {
        Write-Err "配置文件存在问题"
        exit 1
    }
    
    if (!$Build) { exit 0 }
}

# 本地构建测试
if ($Build) {
    Write-Info "开始本地构建测试（snapshot 模式）..."
    Write-Info "这不会创建 GitHub Release"
    Write-Host ""
    
    # 确保测试通过
    Write-Info "运行单元测试..."
    go test ./...
    
    if ($LASTEXITCODE -ne 0) {
        Write-Err "测试失败，中止构建"
        exit 1
    }
    
    Write-Success "测试通过"
    Write-Host ""
    
    # 执行构建
    Write-Info "执行 GoReleaser 构建..."
    goreleaser release --snapshot --clean
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "构建成功！"
        Write-Host ""
        Write-Info "构建产物位于 dist/ 目录："
        Get-ChildItem dist -Filter "*.zip" | ForEach-Object {
            $size = [math]::Round($_.Length / 1MB, 2)
            Write-Host "  - $($_.Name) ($size MB)" -ForegroundColor Green
        }
        Write-Host ""
        Write-Info "测试构建产物："
        Write-Host "  cd dist" -ForegroundColor Cyan
        Write-Host "  Expand-Archive *.zip -DestinationPath test" -ForegroundColor Cyan
        Write-Host "  .\test\gpd-touch-fix.exe -version" -ForegroundColor Cyan
    } else {
        Write-Err "构建失败"
        exit 1
    }
    
    exit 0
}

# 如果没有指定参数，显示帮助
Write-Info "GoReleaser 本地测试工具"
Write-Host ""
Write-Host "用法:" -ForegroundColor Yellow
Write-Host "  .\test-release.ps1 -Install    # 安装 GoReleaser"
Write-Host "  .\test-release.ps1 -Check      # 检查配置文件"
Write-Host "  .\test-release.ps1 -Build      # 本地测试构建"
Write-Host "  .\test-release.ps1 -Clean      # 清理 dist 目录"
Write-Host ""
Write-Host "组合使用:" -ForegroundColor Yellow
Write-Host "  .\test-release.ps1 -Check -Build    # 先检查后构建"
Write-Host "  .\test-release.ps1 -Clean -Build    # 先清理后构建"
Write-Host ""
Write-Info "提示：发布正式版本请参考 docs/GORELEASER.md"
