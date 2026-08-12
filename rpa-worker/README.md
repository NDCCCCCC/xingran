# RPA Worker

RPA (Robotic Process Automation) Worker 是 XingRan-Next 系统的分布式任务执行器。

## 功能特性

- **分布式任务消费**: 基于 Redis Streams 的任务队列
- **浏览器自动化**: 支持 Chromium 浏览器自动化操作
- **并发控制**: 可配置的最大并发任务数
- **自动扩缩容**: 支持运行时动态调整并发数，无需重启
- **优雅关闭**: 支持任务完成后安全关闭
- **进度上报**: 实时上报任务执行进度
- **心跳机制**: 定时向后端发送心跳
- **错误重试**: 支持动作级别的错误重试
- **变量替换**: 支持 `${variable}` 格式的变量替换

## 目录结构

```
rpa-worker/
├── cmd/
│   └── main.go              # 入口文件
├── internal/
│   ├── browser/             # 浏览器管理
│   │   ├── pool.go          # 浏览器池
│   │   └── page_manager.go  # 页面管理器
│   ├── communication/       # 通信层
│   │   ├── api_client.go    # HTTP API 客户端
│   │   ├── redis_client.go  # Redis 客户端
│   │   └── progress_reporter.go # 进度上报
│   ├── config/              # 配置管理
│   │   └── config.go
│   ├── executor/            # 执行引擎
│   │   └── engine.go
│   ├── logger/              # 日志
│   │   └── logger.go
│   ├── types/               # 类型定义
│   │   └── types.go
│   └── worker/              # Worker 主逻辑
│       └── worker.go
├── configs/
│   └── config.yaml          # 配置文件
├── deployments/             # 部署文件
├── go.mod
└── README.md
```

## 快速开始

### 1. 安装依赖

```bash
cd rpa-worker
go mod download
```

### 2. 配置

编辑 `configs/config.yaml`：

```yaml
worker:
  name: "rpa-worker-1"
  max_concurrency: 5

backend:
  base_url: "http://localhost:9000/api/v1"

redis:
  addr: "localhost:6379"
```

### 3. 构建

```bash
go build -o rpa-worker ./cmd/main.go
```

### 4. 运行

```bash
./rpa-worker -config configs/config.yaml
```

## Docker 部署

### 构建镜像

```bash
docker build -t rpa-worker:latest -f deployments/Dockerfile .
```

### 运行容器

```bash
docker run -d \
  --name rpa-worker \
  -p 8080:8080 \
  -v $(pwd)/configs/config.yaml:/app/configs/config.yaml \
  rpa-worker:latest
```

### 使用 Docker Compose

```bash
docker-compose -f deployments/docker-compose.yml up -d
```

## 配置说明

### Worker 配置

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| worker.id | Worker ID | 自动生成 UUID |
| worker.name | Worker 名称 | rpa-worker-1 |
| worker.max_concurrency | 最大并发数 | 5 |
| worker.version | Worker 版本 | 1.0.0 |

### Backend 配置

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| backend.base_url | 后端 API 地址 | http://localhost:9000/api/v1 |
| backend.api_token | API Token | 空 |
| backend.timeout | 请求超时 | 30s |

### Redis 配置

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| redis.addr | Redis 地址 | localhost:6379 |
| redis.password | Redis 密码 | 空 |
| redis.stream_tasks | 任务流名称 | rpa:tasks |

## 支持的动作

| 动作类型 | 说明 | 参数 |
|----------|------|------|
| navigate | 导航到 URL | url: 目标 URL |
| click | 点击元素 | selector: CSS 选择器 |
| fill | 填写表单 | selector, value: 选择器和值 |
| select | 选择下拉 | selector, value: 选择器和选项 |
| wait | 等待时间 | duration: 毫秒 |
| screenshot | 截图 | 无 |
| wait_for | 等待元素 | selector, timeout: 选择器和超时 |

## 变量替换

任务脚本支持 `${variable}` 格式的变量替换：

```json
{
  "type": "fill",
  "selector": "#username",
  "params": {
    "value": "${username}"
  }
}
```

## 开发

### 运行测试

```bash
go test ./...
```

### 代码格式化

```bash
go fmt ./...
goimports -w .
```

### 静态检查

```bash
golangci-lint run
```

## 自动扩缩容功能

### 概述

RPA Worker 支持运行时动态调整并发任务数，无需重启服务。通过 Redis Pub/Sub 接收后端下发的扩缩容指令。

### 工作原理

```
后端 API → Redis Pub/Sub → Worker → 动态调整 maxConcurrency
```

### 扩缩容 API

#### 扩容 Worker

```bash
POST /api/v1/rpa/workers/{worker_id}/scale-up
{
  "concurrency": 10,
  "reason": "任务积压"
}
```

#### 缩容 Worker

```bash
POST /api/v1/rpa/workers/{worker_id}/scale-down
{
  "concurrency": 3,
  "reason": "低负载"
}
```

#### 批量扩缩容

```bash
POST /api/v1/rpa/workers/scale-all
{
  "direction": "up",
  "concurrency": 10,
  "reason": "系统负载增加"
}
```

### 扩缩容限制

| 参数 | 值 | 说明 |
|------|-----|------|
| 最小并发数 | 1 | 不能低于 1 |
| 最大并发数 | 50 | 硬编码上限 |
| 缩容保护 | - | 当前任务数 > 目标值时拒绝 |

### Worker 日志

```
INFO  received scale command  direction=up concurrency=10
INFO  scaled up  from=5 to=10
INFO  worker registered  max_concurrency=10
```

### 详见文档

完整的自动扩缩容实现说明请参考: [docs/auto-scaling.md](docs/auto-scaling.md)

## 许可证

Copyright (c) 2024 XingRan-Next
