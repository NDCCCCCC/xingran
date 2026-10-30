@echo off
setlocal enabledelayedexpansion

echo ============================================
echo 编译后端服务（Linux目标）
echo ============================================

REM 设置编译环境变量
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0

echo 编译目标: Linux amd64
echo 开始编译...

go build -o xingran-backend ./cmd/main.go

if %ERRORLEVEL% NEQ 0 (
    echo 编译失败！错误代码: %ERRORLEVEL%
    pause
    exit /b 1
)

echo 编译成功！
echo.

REM 检查生成的文件
if exist xingran-backend (
    echo 生成的文件: xingran-backend
    for %%A in (xingran-backend) do (
        set size=%%~zA
        set /if sizeMB=!size!/1048576
    )
    echo 文件大小: !sizeMB! MB
    echo.
    echo ============================================
    echo 部署说明
    echo ============================================
    echo 后端服务运行在远程服务器: 10.62.10.33:9000
    echo.
    echo 需要执行以下步骤：
    echo 1. 将 xingran-backend 上传到服务器
    echo 2. 停止当前运行的后端服务
    echo 3. 替换二进制文件
    echo 4. 重新启动后端服务
    echo ============================================
) else (
    echo 错误：未找到生成的文件 xingran-backend
)

pause
