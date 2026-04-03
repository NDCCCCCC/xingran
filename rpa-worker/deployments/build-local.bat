@echo off
REM RPA Worker 本地构建 + Docker 打包脚本 (Windows)

setlocal enabledelayedexpansion

echo ======================================
echo RPA Worker 本地构建脚本
echo ======================================

REM 获取脚本目录
set SCRIPT_DIR=%~dp0
set PROJECT_ROOT=%SCRIPT_DIR%..

REM 1. 检查 Go 环境
echo [INFO] 检查 Go 环境...
where go >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go 未安装，请先安装 Go 1.21+
    exit /b 1
)

for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i
echo [INFO] Go 版本: %GO_VERSION%

REM 2. 配置 Go 代理
echo [INFO] 配置 Go 代理...
set GOPROXY=https://goproxy.cn,https://goproxy.io,direct
set GOSUMDB=off
set CGO_ENABLED=0

REM 3. 进入项目目录
cd /d "%PROJECT_ROOT%"

REM 4. 下载依赖
echo [INFO] 下载 Go 依赖...
go mod download
if errorlevel 1 (
    echo [ERROR] 依赖下载失败
    exit /b 1
)

REM 5. 本地构建
echo [INFO] 开始本地构建...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o rpa-worker.exe ./cmd/main.go
if errorlevel 1 (
    echo [ERROR] 构建失败
    exit /b 1
)

if not exist "rpa-worker.exe" (
    echo [ERROR] 构建失败：找不到输出文件
    exit /b 1
)

echo [INFO] 构建成功

REM 6. 重命名为 Linux 可执行文件
move /Y rpa-worker.exe rpa-worker

REM 7. 构建 Docker 镜像
cd /d "%SCRIPT_DIR%"
echo [INFO] 构建 Docker 镜像...
docker build -f Dockerfile.local -t rpa-worker:latest ..
if errorlevel 1 (
    echo [ERROR] Docker 镜像构建失败
    exit /b 1
)

echo.
echo [INFO] 完成！
echo.
echo 启动服务: docker-compose -f docker-compose.local.yml up -d
echo 查看日志: docker-compose -f docker-compose.local.yml logs -f

pause
