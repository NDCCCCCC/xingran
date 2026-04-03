#!/bin/bash
# RPA Worker 本地构建 + Docker 打包脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}RPA Worker 本地构建脚本${NC}"
echo -e "${GREEN}======================================${NC}"

# 项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# 1. 检查 Go 环境
info "检查 Go 环境..."
if ! command -v go &> /dev/null; then
    error "Go 未安装，请先安装 Go 1.21+"
fi
GO_VERSION=$(go version | awk '{print $3}')
info "Go 版本: $GO_VERSION"

# 2. 配置 Go 代理
info "配置 Go 代理..."
export GOPROXY=https://goproxy.cn,https://goproxy.io,direct
export GOSUMDB=off
export CGO_ENABLED=0

# 3. 进入项目目录
cd "$PROJECT_ROOT"

# 4. 下载依赖
info "下载 Go 依赖..."
go mod download

# 5. 本地构建
info "开始本地构建..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o rpa-worker ./cmd/main.go

if [ ! -f "rpa-worker" ]; then
    error "构建失败：找不到输出文件"
fi

info "构建成功: $(ls -lh rpa-worker | awk '{print $5}')"

# 6. 构建 Docker 镜像
info "构建 Docker 镜像..."
cd "$SCRIPT_DIR"
docker build -f Dockerfile.local -t rpa-worker:latest ..

# 7. 清理二进制文件（可选）
# rm -f "$PROJECT_ROOT/rpa-worker"

info "完成！"
echo ""
info "启动服务: docker-compose -f docker-compose.local.yml up -d"
info "查看日志: docker-compose -f docker-compose.local.yml logs -f"
