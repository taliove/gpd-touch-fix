# 重启 GPDTouchFix 服务
# 需要管理员权限运行

Write-Host "正在停止服务..." -ForegroundColor Yellow
Stop-Service -Name "GPDTouchFix" -ErrorAction SilentlyContinue

Write-Host "等待服务停止..." -ForegroundColor Yellow
Start-Sleep -Seconds 2

Write-Host "正在启动服务..." -ForegroundColor Yellow
Start-Service -Name "GPDTouchFix"

Write-Host "服务已重启！" -ForegroundColor Green
Write-Host ""
Write-Host "查看服务状态:" -ForegroundColor Cyan
Get-Service -Name "GPDTouchFix" | Format-Table -AutoSize
