@echo off
REM RPA Worker 本地构建和部署脚本 (Windows)
REM 用法: rebuild-local.bat

echo =========================================
echo RPA Worker 本地构建和部署
echo =========================================
echo.

REM 设置环境变量
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0

echo 步骤 1: 构建 Go 二进制文件...
go build -o rpa-worker ./cmd/main.go
if errorlevel 1 (
    echo 构建失败！
    exit /b 1
)
echo 构建完成
echo.

echo 步骤 2: 构建 Docker 镜像...
docker build -f deployments/Dockerfile.local -t rpa-worker:latest .
if errorlevel 1 (
    echo 镜像构建失败！
    exit /b 1
)
echo 镜像构建完成
echo.

echo 步骤 3: 重启容器...
cd deployments
docker-compose -f docker-compose.local.yml down
docker-compose -f docker-compose.local.yml up -d
echo 容器重启完成
echo.

echo =========================================
echo 部署完成！
echo.
echo 查看日志:
echo   docker-compose -f docker-compose.local.yml logs -f
echo.
echo 查看容器状态:
echo   docker-compose -f docker-compose.local.yml ps
echo =========================================
