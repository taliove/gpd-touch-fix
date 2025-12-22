# GPD Touch Fix - Build Script
# Usage: .\build.ps1 [-Clean] [-Test] [-Release] [-Version "1.0.0"]

param(
    [switch]$Clean,
    [switch]$Test,
    [switch]$Release,
    [string]$Version = "1.0.0"
)

$ErrorActionPreference = "Stop"

# Project info
$ProjectName = "gpd-touch-fix"
$OutputDir = "bin"
$DistDir = "dist"

# Color output functions
function Write-Success { Write-Host "[OK] $args" -ForegroundColor Green }
function Write-Info { Write-Host "[INFO] $args" -ForegroundColor Cyan }
function Write-Err { Write-Host "[ERROR] $args" -ForegroundColor Red }

# Clean
if ($Clean) {
    Write-Info "Cleaning build directories..."
    if (Test-Path $OutputDir) { Remove-Item -Recurse -Force $OutputDir }
    if (Test-Path $DistDir) { Remove-Item -Recurse -Force $DistDir }
    Write-Success "Clean completed"
    if (!$Test -and !$Release) { exit 0 }
}

# Create output directory
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

# Get build info
$GitCommit = "unknown"
if (Get-Command git -ErrorAction SilentlyContinue) {
    try {
        $GitCommit = (git rev-parse --short HEAD 2>$null) | Out-String
        $GitCommit = $GitCommit.Trim()
        if ([string]::IsNullOrWhiteSpace($GitCommit)) {
            $GitCommit = "unknown"
        }
    } catch {}
}

$BuildTime = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
$GoVersion = (go version).Split(" ")[2]

Write-Info "Build Info:"
Write-Info "  Version: $Version"
Write-Info "  Commit: $GitCommit"
Write-Info "  Time: $BuildTime"
Write-Info "  Go: $GoVersion"
Write-Host ""

# Run tests
if ($Test) {
    Write-Info "Running unit tests..."
    go test -v ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Err "Tests failed"
        exit 1
    }
    Write-Success "All tests passed"
    Write-Host ""
}

# Build ldflags
$LdFlags = @(
    "-s",
    "-w",
    "-X main.Version=$Version",
    "-X main.GitCommit=$GitCommit",
    "-X main.BuildTime=$BuildTime"
) -join " "

# Development build
if (!$Release) {
    Write-Info "Building development version..."
    $OutputFile = Join-Path $OutputDir "$ProjectName.exe"
    
    go build -ldflags="$LdFlags" -o $OutputFile
    
    if ($LASTEXITCODE -ne 0) {
        Write-Err "Build failed"
        exit 1
    }
    
    $FileSize = (Get-Item $OutputFile).Length / 1MB
    Write-Success "Build completed: $OutputFile ($([math]::Round($FileSize, 2)) MB)"
    
    # Show version info
    Write-Host ""
    & $OutputFile -version
    
    exit 0
}

# Release build
Write-Info "Building release versions..."
New-Item -ItemType Directory -Force -Path $DistDir | Out-Null

$Platforms = @(
    @{ GOOS="windows"; GOARCH="amd64"; Suffix="windows-x64" },
    @{ GOOS="windows"; GOARCH="386"; Suffix="windows-x86" }
)

foreach ($Platform in $Platforms) {
    $env:GOOS = $Platform.GOOS
    $env:GOARCH = $Platform.GOARCH
    $Suffix = $Platform.Suffix
    
    Write-Info "Building $Suffix..."
    
    $ReleaseDir = Join-Path $DistDir "$ProjectName-$Version-$Suffix"
    New-Item -ItemType Directory -Force -Path $ReleaseDir | Out-Null
    
    $OutputFile = Join-Path $ReleaseDir "$ProjectName.exe"
    
    go build -ldflags="$LdFlags" -o $OutputFile
    
    if ($LASTEXITCODE -ne 0) {
        Write-Err "Build $Suffix failed"
        exit 1
    }
    
    # Copy config example and docs
    Copy-Item "config.example.json" $ReleaseDir
    Copy-Item "README.md" $ReleaseDir
    
    # Create ZIP package
    $ZipFile = "$ReleaseDir.zip"
    Compress-Archive -Path $ReleaseDir -DestinationPath $ZipFile -Force
    
    $FileSize = (Get-Item $OutputFile).Length / 1MB
    $ZipSize = (Get-Item $ZipFile).Length / 1MB
    
    Write-Success "$Suffix build completed (EXE: $([math]::Round($FileSize, 2)) MB, ZIP: $([math]::Round($ZipSize, 2)) MB)"
}

Write-Host ""
Write-Success "All release versions built successfully!"
Write-Info "Output directory: $DistDir"
Write-Host ""

# List generated files
Get-ChildItem $DistDir -Filter "*.zip" | ForEach-Object {
    Write-Info "  - $($_.Name)"
}
