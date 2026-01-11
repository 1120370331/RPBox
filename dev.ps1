# RPBox 开发启动脚本
param(
    [string]$Action = "menu"
)

function Show-Menu {
    Write-Host "`n======== RPBox Dev ========" -ForegroundColor Cyan
    Write-Host "1. 启动客户端"
    Write-Host "2. 启动服务端"
    Write-Host "3. 启动全部"
    Write-Host "4. 安装依赖"
    Write-Host "5. 退出"
    Write-Host "============================"
}

function Start-Client {
    Write-Host "启动客户端..." -ForegroundColor Green
    Start-Process pwsh -ArgumentList "-NoExit", "-Command", "cd client; npm run tauri dev"
}

function Start-Server {
    Write-Host "启动服务端..." -ForegroundColor Green
    Start-Process pwsh -ArgumentList "-NoExit", "-Command", "cd server; go run cmd/server/main.go"
}

while ($true) {
    Show-Menu
    $choice = Read-Host "选择"

    switch ($choice) {
        "1" { Start-Client }
        "2" { Start-Server }
        "3" { Start-Client; Start-Server }
        "4" {
            Set-Location client; npm install
            Set-Location ../server; go mod tidy
            Set-Location ..
        }
        "5" { exit }
    }
}
