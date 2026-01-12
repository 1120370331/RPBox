@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

echo.
echo ========================================
echo   RPBox .memories 速查工具
echo ========================================
echo.

:menu
echo [1] 查看模块索引
echo [2] 列出所有模块
echo [3] 搜索关键词
echo [4] 打开模板目录
echo [0] 退出
echo.
set /p choice=请选择:

if "%choice%"=="1" goto index
if "%choice%"=="2" goto list
if "%choice%"=="3" goto search
if "%choice%"=="4" goto templates
if "%choice%"=="0" goto end
goto menu

:index
echo.
type "%~dp0..\modules\INDEX.md"
echo.
pause
goto menu

:list
echo.
echo 模块列表:
dir /b /ad "%~dp0..\modules" 2>nul
echo.
pause
goto menu

:search
set /p keyword=输入关键词:
echo.
findstr /s /i /n "%keyword%" "%~dp0..\*.md"
echo.
pause
goto menu

:templates
explorer "%~dp0..\templates"
goto menu

:end
endlocal
