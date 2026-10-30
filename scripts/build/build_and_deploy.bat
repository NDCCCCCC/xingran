@echo off
REM ============================================
REM 后端编译和部署脚本
REM ============================================

echo ============================================
echo 编译后端服务
echo ============================================

REM 设置编译目标
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0

REM 编译
go build -o xingran-backend-linux ./cmd/main.go

if %ERRORLEVEL% NEQ 0 (
    echo 编译失败！
    pause
    exit /b 1
)

echo ============================================
echo 编译成功！
echo ============================================
echo 输出文件: xingran-backend-linux
echo.
echo ============================================
echo 部署说明
echo ============================================
echo 1. 将 xingran-backend-linux 上传到 10.62.10.33 服务器
echo 2. 停止当前运行的后端服务
echo 3. 替换二进制文件
echo 4. 重新启动后端服务
echo.
echo 示例部署命令:
echo   scp xingran-backend-linux user@10.62.10.33:/path/to/backend/
echo   ssh user@10.62.10.33 "systemctl restart xingran-backend"
echo ============================================
pause
