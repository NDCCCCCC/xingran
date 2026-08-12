# RPA Worker 自动扩缩容功能说明

## 功能概述

RPA Worker 现已支持动态自动扩缩容功能，可以在运行时调整并发任务数量，无需重启 Worker。

## 架构设计

### 通信流程

```
┌─────────────────────────────────────────────────────────────────┐
│                    扩缩容指令流程                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────┐    Redis Pub/Sub     ┌─────────────────────┐  │
│  │   后端 API   │ ===================> │   RPA Worker       │  │
│  │             │                      │                     │  │
│  │  /scale-up   │  Channel:            │  scaleCommandListener│  │
│  │  /scale-down │  - worker:scale:all  │                     │  │
│  │  /scale-all   │  - worker:scale:{id} │  processScaleCommands│  │
│  └─────────────┘                      └─────────────────────┘  │
│         │                                       │             │
│         v                                       v             │
│  ┌─────────────┐                      ┌─────────────────────┐  │
│  │  发布指令    │                      │  动态调整并发数     │  │
│  └─────────────┘                      └─────────────────────┘  │
│                                             │             │
│                                             v             │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              maxConcurrency 动态变化                     │    │
│  │                                                          │    │
│  │   初始值: 5        ─────▶         10 (扩容)               │    │
│  │                    │                                      │    │
│  │   任务积压时 ──────▶         20 (进一步扩容)            │    │
│  │                    │                                      │    │
│  │   空闲时     ─────▶         3  (缩容)                 │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Worker 端实现

### 新增字段

```go
type Worker struct {
    // ... 其他字段 ...

    // Dynamic scaling fields
    maxConcurrency      int                    // 当前最大并发数（动态）
    maxConcurrencyMu    sync.RWMutex           // 保护并发数读写
    scaleCommandChan    chan scaleCommandWrapper // 接收扩缩容指令
}
```

### 关键方法

| 方法 | 说明 |
|------|------|
| `scaleCommandListener()` | 订阅 Redis Pub/Sub 频道接收扩缩容指令 |
| `processScaleCommands()` | 处理扩缩容指令并执行 |
| `scaleUp(targetConcurrency)` | 增加最大并发数 |
| `scaleDown(targetConcurrency)` | 减少最大并发数 |
| `getMaxConcurrency()` | 线程安全地获取当前最大并发数 |

### 扩缩容限制

- **最小并发数**: 1
- **最大并发数**: 50
- **缩容保护**: 当前任务数超过目标值时拒绝缩容

## 后端 API

### 扩容单个 Worker

```bash
POST /api/v1/rpa/workers/{worker_id}/scale-up
Content-Type: application/json

{
  "concurrency": 10,
  "reason": "任务积压严重"
}
```

### 缩容单个 Worker

```bash
POST /api/v1/rpa/workers/{worker_id}/scale-down
Content-Type: application/json

{
  "concurrency": 3,
  "reason": "夜间低负载"
}
```

### 批量扩缩容所有 Worker

```bash
POST /api/v1/rpa/workers/scale-all
Content-Type: application/json

{
  "direction": "up",
  "concurrency": 10,
  "reason": "系统负载增加"
}
```

### 自动扩缩容配置

```bash
# 获取配置
GET /api/v1/rpa/workers/autoscale/config

# 更新配置
POST /api/v1/rpa/workers/autoscale/config
Content-Type: application/json

{
  "enabled": true,
  "scale_up_threshold": 10,
  "scale_down_threshold": 5,
  "min_concurrency": 1,
  "max_concurrency": 20,
  "check_interval": 30
}
```

## Redis Pub/Sub 频道

| 频道 | 说明 |
|------|------|
| `worker:scale:all` | 向所有 Worker 广播的扩缩容指令 |
| `worker:scale:{worker_id}` | 向指定 Worker 发送的扩缩容指令 |
| `worker:scale:ack:{command_id}` | Worker 执行结果确认 |

## 扩缩容指令格式

```json
{
  "commandId": "550e8400-e29b-41d4-a716-446655440000",
  "workerId": "worker-12345678",
  "direction": "up",
  "concurrency": 10,
  "reason": "任务队列积压",
  "timestamp": 1709289600
}
```

## Worker 行为

### 扩容行为 (Scale Up)

1. 验证目标并发数 > 当前并发数
2. 应用最大限制 (50)
3. 更新 `maxConcurrency` 值
4. 记录扩容事件日志
5. 向后端重新注册新容量

### 缩容行为 (Scale Down)

1. 验证目标并发数 < 当前并发数
2. 应用最小限制 (1)
3. 检查当前运行任务数
4. 如果当前任务 > 目标值，拒绝缩容
5. 更新 `maxConcurrency` 值
6. 记录缩容事件日志
7. 向后端重新注册新容量

### 任务消费行为

```go
// 动态获取当前最大并发数
maxConcurrency := w.getMaxConcurrency()

// 检查是否可以接收新任务
if w.currentTasks < maxConcurrency {
    // 接收任务
}
```

## 使用示例

### 前端调用示例

```typescript
import { post } from '@/lib/api';

// 扩容 Worker
async function scaleUpWorker(workerId: string, concurrency: number) {
  await post(`/rpa/workers/${workerId}/scale-up`, {
    concurrency,
    reason: '用户手动扩容'
  });
}

// 缩容 Worker
async function scaleDownWorker(workerId: string, concurrency: number) {
  await post(`/rpa/workers/${workerId}/scale-down`, {
    concurrency,
    reason: '夜间低负载缩容'
  });
}

// 批量扩容
async function scaleAllWorkers(direction: 'up' | 'down', concurrency: number) {
  await post('/rpa/workers/scale-all', {
    direction,
    concurrency,
    reason: '系统负载变化'
  });
}
```

## 日志输出

Worker 在执行扩缩容时会输出详细日志：

```
INFO  received scale command  direction=up concurrency=10 reason=任务积压
INFO  scaled up  from=5 to=10  reason=任务积压
INFO  worker registered successfully  max_concurrency=10
```

## 注意事项

1. **并发数范围**: 1-50
2. **缩容保护**: 当前运行任务数超过目标值时会拒绝缩容
3. **确认机制**: 执行成功后会发送 ACK 确认
4. **注册更新**: 扩缩容后会向后端重新注册新容量
5. **线程安全**: 所有并发数变更都通过互斥锁保护
