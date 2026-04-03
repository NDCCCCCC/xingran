#!/bin/sh
set -e

# 等待 Redis 就绪
echo "等待 Redis 连接..."
while ! nc -z ${REDIS_ADDR:-localhost:6379}; do
    sleep 1
done
echo "Redis 已就绪"

# 等待后端 API 就绪
echo "等待后端 API 连接..."
while ! nc -z ${BACKEND_HOST:-localhost} ${BACKEND_PORT:-9000}; do
    sleep 1
done
echo "后端 API 已就绪"

# 启动 Worker
exec /app/rpa-worker -config /app/configs/config.yaml "$@"
