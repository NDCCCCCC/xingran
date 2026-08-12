# RPA Worker 部署指南

## 概述

RPA Worker 是一个基于 Go 的自动化执行引擎，使用 Chrome 浏览器进行 Web 自动化操作。

## 功能特性

- ✅ 支持12种动作类型（导航、点击、填写、选择、等待、截图等）
- ✅ 循环处理（批量操作）
- ✅ 人工干预（验证码处理）
- ✅ 自动登录（凭证管理）
- ✅ 动态扩缩容
- ✅ 实时进度上报
- ✅ 健康检查和监控

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                        RPA Worker                           │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐   │
│  │ Worker Core │  │ Task Engine  │  │  Browser Pool   │   │
│  └─────────────┘  └──────────────┘  └─────────────────┘   │
│         │                │                    │             │
│         ▼                ▼                    ▼             │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐   │
│  │   Redis     │  │  Backend API │  │  Chrome Browser │   │
│  └─────────────┘  └──────────────┘  └─────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 快速开始

### 使用 Docker Compose（推荐）

1. **准备环境**

```bash
cd rpa-worker/deployments

# 复制环境变量模板
cp .env.example .env

# 编辑 .env 文件，配置必要参数
```

> **注意**: Dockerfile已配置国内Go代理 (`GOPROXY=https://goproxy.cn,direct`)，如果您的网络环境需要使用其他代理，可以通过环境变量 `GOPROXY` 覆盖。

2. **配置环境变量** (.env文件)

```bash
# Worker 配置
WORKER_ID=worker-1
WORKER_NAME=rpa-worker-1
MAX_CONCURRENCY=5

# 后端 API 配置
BACKEND_URL=http://host.docker.internal:9000/api/v1

# 日志配置
LOG_LEVEL=info
```

3. **启动服务**

```bash
docker-compose up -d

# 查看日志
docker-compose logs -f rpa-worker
```

4. **检查状态**

```bash
# 健康检查
curl http://localhost:8080/health

# 查看指标
curl http://localhost:8080/metrics
```

5. **停止服务**

```bash
docker-compose down
```

### 本地构建（推荐用于网络受限环境）

如果在 Docker 内构建时遇到网络问题（如 GitHub 访问受限），可以使用本地构建方式：

1. **在服务器上构建二进制**

```bash
cd rpa-worker

# 配置 Go 代理
export GOPROXY=https://goproxy.cn,https://goproxy.io,direct
export GOSUMDB=off

# 下载依赖
go mod download

# 本地构建
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o rpa-worker ./cmd/main.go
```

2. **使用本地构建脚本**

```bash
cd deployments

# Linux/Mac
chmod +x build-local.sh
./build-local.sh

# Windows
build-local.bat
```

3. **使用本地构建的 Docker Compose**

```bash
cd deployments
docker-compose -f docker-compose.local.yml up -d
```

### 手动构建

1. **构建镜像**

```bash
cd rpa-worker
docker build -f deployments/Dockerfile -t rpa-worker:latest .
```

2. **运行容器**

```bash
docker run -d \
  --name rpa-worker \
  -p 8080:8080 \
  -e WORKER_ID=worker-1 \
  -e BACKEND_URL=http://host.docker.internal:9000/api/v1 \
  -v $(pwd)/configs/config.yaml:/app/configs/config.yaml:ro \
  rpa-worker:latest
```

## 配置说明

### config.yaml 配置项

```yaml
worker:
  id: ""                    # Worker ID（留空自动生成）
  name: "rpa-worker-1"      # Worker 名称
  max_concurrency: 5        # 最大并发数
  task_timeout: 10m         # 任务超时时间

backend:
  base_url: "http://localhost:9000/api/v1"
  heartbeat_interval: 30s   # 心跳间隔

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

browser:
  headless: true            # 无头模式
  timeout: 30s
  max_pages: 10             # 浏览器池最大页面数
  viewport_width: 1920
  viewport_height: 1080
  chrome_path: "/usr/bin/chromium-browser"
  chrome_flags:
    - "--no-sandbox"
    - "--headless=new"

logging:
  level: "info"
  format: "json"
  output: "stdout"
```

## 健康检查端点

| 端点 | 描述 |
|------|------|
| `/health` | 健康检查（返回状态和统计信息） |
| `/ready` | 就绪检查（返回是否可接收任务） |
| `/metrics` | Prometheus格式的指标 |

## 日志和监控

### 查看日志

```bash
# Docker Compose
docker-compose logs -f rpa-worker

# Docker
docker logs -f rpa-worker
```

### 日志格式

```json
{
  "level": "info",
  "msg": "任务执行成功",
  "execution_id": "xxx",
  "duration": "5.2s",
  "timestamp": "2025-02-26T10:00:00Z"
}
```

## 故障排查

### Worker 无法连接到后端

1. 检查 `BACKEND_URL` 配置
2. 如果在本地运行，使用 `host.docker.internal`
3. 检查网络连接

```bash
docker exec rpa-worker ping host.docker.internal
```

### Chrome 启动失败

1. 检查共享内存配置
2. 确保 Docker 容器有足够的资源

```yaml
shm_size: 2gb
deploy:
  resources:
    limits:
      memory: 2G
```

### Redis 连接失败

1. 检查 Redis 服务状态
2. 验证 `REDIS_ADDR` 配置

```bash
docker exec rpa-worker redis-cli -h redis ping
```

## 性能优化

### 调整并发数

根据服务器资源调整 `MAX_CONCURRENCY`：

- 2核2G: 并发数 2-3
- 4核4G: 并发数 5-8
- 8核8G: 并发数 10-15

### 浏览器池配置

```yaml
browser:
  max_pages: 10  # 根据并发数调整
```

## 安全建议

1. 使用环境变量存储敏感信息
2. 不要在日志中记录敏感数据
3. 定期更新 Chrome 版本
4. 限制网络访问

## 许可证

MIT License
