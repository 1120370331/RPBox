@echo off
chcp 65001 >nul
echo ========================================
echo   RPBox 开发环境启动脚本
echo ========================================
echo.

:menu
echo 请选择要启动的服务:
echo   1. 启动客户端 (Tauri + Vue)
echo   2. 启动服务端 (Go)
echo   3. 同时启动全部
echo   4. 安装依赖
echo   5. 退出
echo.
set /p choice=请输入选项 (1-5):

if "%choice%"=="1" goto client
if "%choice%"=="2" goto server
if "%choice%"=="3" goto all
if "%choice%"=="4" goto install
if "%choice%"=="5" exit

echo 无效选项，请重新选择
goto menu

:client
echo.
echo [启动客户端...]
cd client
start cmd /k "npm run tauri dev"
cd ..
goto menu

:server
echo.
echo [启动服务端...]
cd server
start cmd /k "go run cmd/server/main.go"
cd ..
goto menu

:all
echo.
echo [启动全部服务...]
cd client
start cmd /k "npm run tauri dev"
cd ../server
start cmd /k "go run cmd/server/main.go"
cd ..
echo 已启动全部服务
goto menu

:install
echo.
echo [安装依赖...]
cd client
call npm install
cd ..
cd server
call go mod tidy
cd ..
echo 依赖安装完成
goto menu
