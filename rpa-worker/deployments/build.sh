#!/bin/bash
# RPA Worker 构建和部署脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}RPA Worker 构建和部署脚本${NC}"
echo -e "${GREEN}======================================${NC}"

# 函数：打印信息
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# 检查Docker是否安装
check_docker() {
    info "检查Docker是否安装..."
    if ! command -v docker &> /dev/null; then
        error "Docker未安装，请先安装Docker"
    fi
    info "Docker已安装: $(docker --version)"
}

# 检查Docker Compose是否安装
check_docker_compose() {
    info "检查Docker Compose是否安装..."
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        error "Docker Compose未安装，请先安装Docker Compose"
    fi
    info "Docker Compose已安装"
}

# 下载依赖
download_deps() {
    info "下载Go依赖..."
    cd "$PROJECT_ROOT"
    export GOPROXY=https://goproxy.cn,direct
    export GOSUMDB=off
    go mod download
    go mod tidy
    info "依赖下载完成"
}

# 构建Docker镜像
build_image() {
    info "构建Docker镜像..."
    cd "$PROJECT_ROOT"
    docker build -f deployments/Dockerfile -t rpa-worker:latest .
    info "Docker镜像构建完成"
}

# 推送镜像（可选）
push_image() {
    local registry=$1
    if [ -z "$registry" ]; then
        warn "未指定镜像仓库，跳过推送"
        return
    fi

    info "推送镜像到 $registry"
    docker tag rpa-worker:latest "$registry/rpa-worker:latest"
    docker push "$registry/rpa-worker:latest"
}

# 启动服务
start_services() {
    info "启动服务..."

    cd "$SCRIPT_DIR"

    # 检查.env文件
    if [ ! -f ".env" ]; then
        warn ".env文件不存在，从.env.example复制..."
        cp .env.example .env
        warn "请编辑.env文件配置后重新运行"
        return
    fi

    # 使用docker compose或docker-compose
    if docker compose version &> /dev/null; then
        docker compose up -d
    else
        docker-compose up -d
    fi

    info "服务已启动"
}

# 停止服务
stop_services() {
    info "停止服务..."

    cd "$SCRIPT_DIR"

    if docker compose version &> /dev/null; then
        docker compose down
    else
        docker-compose down
    fi

    info "服务已停止"
}

# 查看日志
view_logs() {
    cd "$SCRIPT_DIR"

    if docker compose version &> /dev/null; then
        docker compose logs -f rpa-worker
    else
        docker-compose logs -f rpa-worker
    fi
}

# 健康检查
health_check() {
    info "执行健康检查..."

    # 等待服务启动
    sleep 3

    if curl -s http://localhost:8080/health > /dev/null; then
        info "健康检查通过"
        curl -s http://localhost:8080/health | jq '.'
    else
        error "健康检查失败"
    fi
}

# 显示状态
show_status() {
    cd "$SCRIPT_DIR"

    if docker compose version &> /dev/null; then
        docker compose ps
    else
        docker-compose ps
    fi
}

# 清理资源
cleanup() {
    info "清理未使用的Docker资源..."
    docker system prune -f
    info "清理完成"
}

# 完整部署流程
deploy() {
    info "开始完整部署流程..."

    check_docker
    check_docker_compose
    download_deps
    build_image
    stop_services
    start_services
    sleep 5
    health_check

    info "部署完成！"
    echo ""
    info "健康检查: curl http://localhost:8080/health"
    info "查看日志: $0 logs"
    info "查看状态: $0 status"
}

# 显示使用帮助
show_help() {
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  deploy    - 完整部署（默认）"
    echo "  build     - 仅构建Docker镜像"
    echo "  start     - 启动服务"
    echo "  stop      - 停止服务"
    echo "  restart   - 重启服务"
    echo "  logs      - 查看日志"
    echo "  status    - 查看状态"
    echo "  health    - 健康检查"
    echo "  clean     - 清理Docker资源"
    echo "  push [registry] - 推送镜像到仓库"
    echo "  help      - 显示此帮助信息"
}

# 主程序
main() {
    local command=${1:-deploy}

    case "$command" in
        deploy)
            deploy
            ;;
        build)
            check_docker
            download_deps
            build_image
            ;;
        start)
            start_services
            ;;
        stop)
            stop_services
            ;;
        restart)
            stop_services
            start_services
            ;;
        logs)
            view_logs
            ;;
        status)
            show_status
            ;;
        health)
            health_check
            ;;
        clean)
            check_docker
            cleanup
            ;;
        push)
            push_image "$2"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            error "未知命令: $command"
            show_help
            ;;
    esac
}

# 执行主程序
main "$@"
