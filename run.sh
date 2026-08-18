#!/bin/bash
# run.sh — 启动 xingran-backend 开发服务器(Linux/Mac)
#
# 解决问题 (log-errors-fix-20260818 P2-15): 旧进程未清理导致
#   "listen tcp :9000: bind: address already in use"
#
# 流程:
#   1. 查找并杀掉占用 9000 端口的进程(xingran-backend / go run)
#   2. 等待 1 秒确保端口释放
#   3. go run ./cmd/main.go
#
# 用法:
#   ./run.sh                  # 默认启动
#   SKIP_PORT_CLEANUP=1 ./run.sh   # 跳过端口清理(快速启动时使用)

set -e

# === 端口清理(可关闭)===
if [ "${SKIP_PORT_CLEANUP}" != "1" ]; then
    echo "[run] 清理 :9000 端口上的旧进程..."

    # lsof 在 Linux/Mac 上可用;若 lsof 不存在则用 fuser / ss 兜底
    if command -v lsof >/dev/null 2>&1; then
        PIDS=$(lsof -t -i:9000 2>/dev/null || true)
    elif command -v fuser >/dev/null 2>&1; then
        PIDS=$(fuser 9000/tcp 2>/dev/null || true)
    elif command -v ss >/dev/null 2>&1; then
        PIDS=$(ss -tlnp 'sport = :9000' 2>/dev/null | grep -oP 'pid=\K[0-9]+' || true)
    else
        PIDS=""
    fi

    if [ -n "$PIDS" ]; then
        for PID in $PIDS; do
            echo "[run] 发现占用进程 PID=$PID,尝试结束..."
            if kill -9 "$PID" 2>/dev/null; then
                echo "[run] 已结束 PID=$PID"
            else
                echo "[run] 结束 PID=$PID 失败(可能权限不足或已退出)"
            fi
        done
    fi

    # 兜底: pkill xingran-backend(避免遗漏 netstat/ss 未列出的子进程)
    pkill -9 -f xingran-backend 2>/dev/null || true

    # 等待端口释放
    sleep 1
fi

# === 启动服务 ===
echo "[run] 启动 xingran-backend 开发服务器..."
cd "$(dirname "$0")"
exec go run ./cmd/main.go