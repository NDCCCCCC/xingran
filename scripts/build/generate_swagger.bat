@echo off
REM generate_swagger.bat
REM Swagger文档生成脚本（Windows）

setlocal enabledelayedexpansion

REM 设置颜色
set "GREEN=[92m"
set "RED=[91m"
set "YELLOW=[93m"
set "NC=[0m"

echo %GREEN%======================================%NC%
echo %GREEN%  Swagger 文档生成脚本%NC%
echo %GREEN%======================================%NC%
echo.

REM 切换到项目根目录
cd /d "%~dp0.."

REM 检查swag是否安装
where swag >nul 2>nul
if %errorlevel% neq 0 (
    echo %YELLOW%swag 未安装，正在安装...%NC%
    go install github.com/swaggo/swag/cmd/swag@latest

    if %errorlevel% neq 0 (
        echo %RED%swag 安装失败%NC%
        pause
        exit /b 1
    )

    echo %GREEN%swag 安装成功%NC%
    echo.

    REM 刷新环境变量
    refreshenv >nul 2>nul
)

REM 显示swag版本
for /f "tokens=*" %%i in ('swag --version 2^>^&1') do set SWAG_VERSION=%%i
echo %GREEN%当前版本: !SWAG_VERSION!%NC%
echo.

REM 生成文档
echo %YELLOW%正在生成Swagger文档...%NC%
swag init -g cmd/main.go -o docs --parseInternal --parseDependency --parseInternal

if %errorlevel% equ 0 (
    echo.
    echo %GREEN%======================================%NC%
    echo %GREEN%  文档生成成功！%NC%
    echo %GREEN%======================================%NC%
    echo.
    echo 生成的文件:
    echo   - docs\swagger.json
    echo   - docs\swagger.yaml
    echo   - docs\docs.go
    echo.
    echo 访问地址: %YELLOW%http://localhost:8080/swagger/index.html%NC%
    echo.
) else (
    echo.
    echo %RED%======================================%NC%
    echo %RED%  文档生成失败！%NC%
    echo %RED%======================================%NC%
    echo.
    echo 请检查:
    echo   1. 是否在项目根目录
    echo   2. Handler的Swagger注解是否正确
    echo   3. 是否有编译错误
    echo.
    pause
    exit /b 1
)

pause
