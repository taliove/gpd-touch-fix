# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GPD Touch Fix is a Windows utility that automatically repairs touchscreen issues on GPD devices after sleep/wake cycles. It runs as a Windows service, monitoring power events and automatically disabling/re-enabling the I2C HID touch device when problems are detected.

## Build Commands

Use the Makefile for all development tasks:

```powershell
# Run lint, tests, and build
make all

# Individual commands
make build      # Build executable
make test       # Run tests
make lint       # Run golangci-lint
make coverage   # Generate coverage report
make clean      # Clean build artifacts

# View all available commands
make help
```

Manual commands (use Makefile instead when possible):

```powershell
# Build locally
go build -o bin/gpd-touch-fix.exe -ldflags="-s -w"

# Run tests
go test -v ./...

# Run golangci-lint
golangci-lint run

# Test GoReleaser build (without publishing)
goreleaser release --snapshot --clean
```

## Development Rules

### Before Committing

1. **Lint must pass**: Run `make lint` and fix all issues
2. **Tests must pass**: Run `make test`
3. **Format code**: Run `go fmt ./...`

### Code Quality

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- All new features should have corresponding unit tests
- Target test coverage: 60%+
- Use golangci-lint configuration in `.golangci.yml`

### Lint Configuration

The project uses golangci-lint with configuration in `.golangci.yml`. Key points:

- Enabled linters: govet, errcheck, staticcheck, gosimple, unused, ineffassign
- Windows-specific code has certain exclusions (e.g., `elog` error returns)
- Run `golangci-lint run` before committing

## Architecture

The codebase is a single Go module with flat structure (all files in root package `main`).

### Core Components

- **main.go** - Entry point and CLI command routing. Handles both interactive CLI mode and Windows service mode detection via `isService()`.

- **service.go** - Windows service implementation using `golang.org/x/sys/windows/svc`. The `gpdTouchService` struct implements the service handler and responds to power events (sleep/wake). Key function: `handlePowerEvent()` processes wake events and triggers device repair.

- **device.go** - Device management via PowerShell commands. `DeviceManager` handles disable/enable/reset operations on PnP devices using `Disable-PnpDevice` and `Enable-PnpDevice`.

- **detector.go** - Scans for I2C HID devices using PowerShell `Get-PnpDevice`. `DeviceInfo.Score()` calculates device match probability to auto-select the touchscreen.

- **power_monitor.go** - Modern Standby (S0 Low Power Idle) support. Contains `PowerMonitor` for display state changes and `WakeEventPoller` for polling-based wake detection when traditional power events are unreliable.

- **config.go** - JSON configuration management. Config stored in `config.json` next to executable.

- **cli.go** - Interactive CLI with colored output, device selection dialogs, and UAC elevation helpers (`RunElevated`, `EnsureAdmin`).

### Key Design Patterns

1. **Dual Detection Strategy**: Traditional power events (`PBT_APMRESUMESUSPEND`, `PBT_APMRESUMEAUTOMATIC`) plus polling fallback for Modern Standby systems where events may not fire reliably.

2. **Smart Repair**: Checks device status before repair (`CheckBeforeReset` config) to avoid unnecessary disable/enable cycles.

3. **Exponential Backoff**: Failed repairs use increasing retry intervals (`baseRetryInterval` -> `maxRetryInterval`).

4. **PowerShell Integration**: All device operations use PowerShell cmdlets via `runPowerShell()` which sets UTF-8 encoding and has a 90-second timeout.

## Testing

The project targets Windows 10/11 with administrator privileges required for device operations.

```powershell
# Run all tests
make test

# Run a single test
go test -run TestFunctionName -v

# Run tests with coverage
make coverage
```

## CI/CD

### GitHub Actions Workflows

- **ci.yml**: Runs on push/PR - lint, test, build
- **release.yml**: Runs on tag push - lint, test, build, release

### CI/CD Notes (Important)

1. **Windows-only project**: This is a Windows-specific project. All CI jobs (lint, test, build) must run on `windows-latest` to avoid cross-compilation issues.

2. **YAML duplicate keys**: When editing `.golangci.yml`, ensure no duplicate top-level keys (e.g., don't define `issues:` twice). YAML parsers will fail with duplicate key errors.

3. **Test output and GitHub Actions**: GitHub Actions interprets lines starting with `Error:` as error annotations. When writing tests that log error-level messages:
   - Use `InitLoggerWithOptions(dir, level, false)` to disable console output in tests
   - This prevents test log output from being misinterpreted as CI errors

4. **golangci-lint on Linux for Windows code**: If running lint on ubuntu-latest for Windows code, you must set `GOOS: windows` as an environment variable. However, it's simpler to just run on `windows-latest`.

### Release Process

Releases are automated via GitHub Actions when a version tag is pushed:

```powershell
git tag -a v1.x.x -m "Release v1.x.x"
git push origin v1.x.x
```

## Commit Message Convention

Use conventional commit prefixes: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`
