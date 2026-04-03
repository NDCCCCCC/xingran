#!/bin/bash
# RPA Worker 本地构建和部署脚本
# 用法: ./rebuild-local.sh

set -e

echo "========================================="
echo "RPA Worker 本地构建和部署"
echo "========================================="

# 配置 Go 代理（解决国内网络访问问题）
export GOPROXY=https://goproxy.cn,direct

echo "使用 Go 代理: $GOPROXY"

# 进入 rpa-worker 目录
cd "$(dirname "$0")/.."

echo ""
echo "步骤 1: 构建 Go 二进制文件..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rpa-worker ./cmd/main.go
echo "✓ 构建完成"

echo ""
echo "步骤 2: 构建 Docker 镜像..."
docker build -f deployments/Dockerfile.local -t rpa-worker:latest .
echo "✓ 镜像构建完成"

echo ""
echo "步骤 3: 重启容器..."
cd deployments
docker-compose -f docker-compose.local.yml down
docker-compose -f docker-compose.local.yml up -d
echo "✓ 容器重启完成"

echo ""
echo "========================================="
echo "部署完成！"
echo ""
echo "查看日志:"
echo "  docker-compose -f docker-compose.local.yml logs -f"
echo ""
echo "查看容器状态:"
echo "  docker-compose -f docker-compose.local.yml ps"
echo "========================================="
