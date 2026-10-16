# RPA 分布式系统实现计划

> **文档日期**：2026-08-12（刷新）
> **实现状态**：设计层 `Playwright` 已被 **`rod`** 替代（rod 为 Go 原生 CDP 客户端，无需 Node.js 依赖且并发更可控），脚本编辑器仍基于 Playwright Python 语法。最新数据格式契约详见 [`RPA-数据格式规范.md`](RPA-数据格式规范.md)。

## 一、项目概述

为 xingran-go-backend 项目添加 RPA（机器人流程自动化）功能，支持：
- **混合 AI 模式**：传统选择器优先 + AI Agent 智能降级
- AI 辅助脚本生成（自然语言描述 → Playwright 脚本）
- AI Agent 自适应执行（选择器失效时智能修复）
- 可视化脚本编辑器
- 分布式执行（Docker Worker 节点）
- 实时进度监控

## 二、架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                   宿主机 - xingran-backend                        │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  Core Services                                            │  │
│  │  ├── TaskService (任务 CRUD + 执行)                       │  │
│  │  ├── WorkerService (Worker 管理 + 任务分发)               │  │
│  │  ├── ExecutionService (执行记录 + 进度)                   │  │
│  │  ├── AIService (OpenAI 兼容 API)                          │  │
│  │  └── ScalingService (自动扩缩容)                           │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              ↓                                  │
│                    Redis (10.62.10.34)                         │
│  - rpa:task:pending (任务队列)                                 │
│  - rpa:task:progress:{executionId} (进度流)                    │
│  - rpa:workers:online (在线集合)                               │
└─────────────────────────────────────────────────────────────────┘
                              ↑ Docker Network
┌─────────────────────────────────────────────────────────────────┐
│                  RPA Worker 容器 (Docker Compose)               │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ Worker-1     │  │ Worker-2     │  │ Worker-N     │         │
│  │              │  │              │  │              │         │
│  │ - Go 进程    │  │ - Go 进程    │  │ - Go 进程    │         │
│  │ - Playwright │  │ - Playwright │  │ - Playwright │         │
│  │ - Chromium   │  │ - Chromium   │  │ - Chromium   │         │
│  │              │  │              │  │              │         │
│  │ 消费任务     │  │ 消费任务     │  │ 消费任务     │         │
│  │ 上报进度     │  │ 上报进度     │  │ 上报进度     │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                    前端 (React + Ant Design)                   │
│  - AI 脚本编辑器                                                │
│  - 任务管理页面                                                │
│  - 执行监控页面（实时进度）                                     │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 混合 AI 执行模式

```
┌─────────────────────────────────────────────────────────────────┐
│                   混合 AI 执行模式                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  正常流程: 传统选择器 (快速、低成本)                            │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐        │
│  │ 生成脚本    │ → → │ 执行选择器    │ → → │ 任务完成     │        │
│  │ (含选择器)   │     │ (毫秒级)      │     │             │        │
│  └─────────────┘     └─────────────┘     └─────────────┘        │
│         ↓ 失败                                                  │
│  AI Agent 降级: 智能分析修复 (慢、但自适应)                       │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  AI Agent 执行循环                                        │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │ 1. 截取当前页面状态 (截图 + HTML)                      │  │  │
│  │  │ 2. 发送给 VLM: "选择器 #submit 失效，如何找到提交按钮？" │  │  │
│  │  │ 3. VLM 分析: "看到按钮在页面中央，点击坐标 (x,y)"     │  │  │
│  │  │ 4. 执行操作: click (x, y)                              │  │  │
│  │  │ 5. 如果找到更好选择器，更新脚本                        │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  │  持续执行直到任务完成                                      │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.3 宿主机与 Worker 通信方式

| 通信方向 | 方式 | 说明 |
|---------|------|------|
| **主服务 → Worker** | Redis Streams | 任务分发：写入 `rpa:task:pending` |
| **Worker → 主服务** | HTTP API | 注册、心跳、结果上报 |
| **Worker → 主服务** | Redis Streams | 执行进度：写入 `rpa:task:progress:{id}` |
| **主服务 → 前端** | WebSocket (扩展 NoticeHub) | 实时进度推送 |
| **Worker → AI API** | HTTP API | AI Agent 调用（多模态 VLM） |

### 2.4 通信流程图

```
任务执行流程：
1. 前端 → 主服务 API: POST /api/v1/rpa/tasks/{id}/execute
2. 主服务 → Redis: XADD rpa:task:pending {task, executionId}
3. Worker → Redis: XREADGROUP rpa:task:pending (阻塞读取)
4. Worker → 主服务 API: POST /api/v1/rpa/workers/{id}/progress (心跳+进度)
5. Worker → Redis: XADD rpa:task:progress:{executionId} {step, log, screenshot}
6. 主服务 → 前端: WebSocket 推送进度消息
7. Worker → 主服务 API: POST /api/v1/rpa/executions/{id}/complete (结果上报)
8. 主服务 → 前端: WebSocket 推送完成消息
```

## 三、数据库设计

已有迁移文件：`internal/core/db/migrations/102_add_rpa_tables.sql`

包含表：
- `sys_rpa_tasks` - 任务定义
- `sys_rpa_workers` - Worker 节点
- `sys_rpa_executions` - 执行记录
- `sys_rpa_schedules` - 定时调度
- `sys_rpa_variables` - 变量管理
- `sys_rpa_templates` - 脚本模板

## 四、后端实现

### 4.1 目录结构

```
internal/
├── api/v1/rpa/
│   ├── rpa_router.go           # 路由注册
│   ├── task_handler.go          # 任务 API
│   ├── worker_handler.go        # Worker API
│   ├── execution_handler.go     # 执行记录 API
│   └── requests/                # 请求 DTO
├── services/rpa/
│   ├── task_service.go          # 任务服务（已有）
│   ├── worker_service.go        # Worker 服务（已有）
│   ├── execution_service.go     # 执行记录服务
│   ├── ai_service.go            # AI 辅助服务
│   └── scaling_service.go       # 自动扩缩容服务
└── websocket/
    └── notice_hub.go            # 扩展支持 RPA 消息
```

### 4.2 核心 API 端点

#### 任务管理
```
POST   /api/v1/rpa/tasks              创建任务
POST   /api/v1/rpa/tasks/list         列表查询
POST   /api/v1/rpa/tasks/{id}          详情
POST   /api/v1/rpa/tasks/{id}/update   更新
POST   /api/v1/rpa/tasks/{id}/delete   删除
POST   /api/v1/rpa/tasks/{id}/execute  执行任务
```

#### Worker 管理
```
POST   /api/v1/rpa/workers/register        注册 Worker
POST   /api/v1/rpa/workers/{id}/heartbeat  心跳上报
POST   /api/v1/rpa/workers/{id}/progress   进度上报
POST   /api/v1/rpa/workers/list            列表查询
```

#### 执行记录
```
POST   /api/v1/rpa/executions/list        列表查询
POST   /api/v1/rpa/executions/{id}         详情
POST   /api/v1/rpa/executions/{id}/cancel  取消执行
POST   /api/v1/rpa/executions/{id}/logs    日志查询
```

#### AI 辅助
```
POST   /api/v1/rpa/ai/generate            生成脚本（自然语言 → 脚本）
POST   /api/v1/rpa/ai/optimize            优化脚本
POST   /api/v1/rpa/ai/explain             解释脚本

# AI Agent 专用接口
POST   /api/v1/rpa/ai/decide             AI 决策下一步动作（降级时调用）
POST   /api/v1/rpa/ai/analyze-failure     分析失败原因并修复
POST   /api/v1/rpa/ai/capture-state      捕获页面状态（截图+HTML）
```

### 4.3 AI 服务设计（混合模式）

#### 4.3.1 服务架构

```go
// internal/services/rpa/ai_service.go

type AIService interface {
    // 脚本生成
    GenerateScriptFromDescription(ctx context.Context, desc string) (*Script, error)

    // 脚本优化
    OptimizeScript(ctx context.Context, script *Script) (*Script, error)

    // AI Agent 核心
    DecideNextAction(ctx context.Context, req *AgentDecisionRequest) (*AgentAction, error)
    AnalyzeFailure(ctx context.Context, req *FailureAnalysisRequest) (*FixAction, error)
}

// AgentDecisionRequest AI Agent 决策请求
type AgentDecisionRequest struct {
    TaskDescription    string   // 当前任务目标
    CurrentStep        int      // 当前步骤
    FailedSelector     string   // 失败的选择器
    ScreenshotBase64    string   // 页面截图 (base64)
    HTMLSnippet         string   // HTML 片段
    AvailableSelectors  []string // 已尝试的选择器列表
}

// AgentAction AI Agent 返回的动作
type AgentAction struct {
    Type           string   // click, fill, wait, etc.
    Selector       string   // 推荐的选择器（如果有）
    Coordinates    []int    // 坐标 [x, y]（作为降级）
    Reasoning      string   // 推理过程
    Confidence     float64 // 置信度 0-1
    SuggestedFix   string   // 建议的修复说明
}
```

#### 4.3.2 AI Agent Prompt 设计

```
System Prompt:
你是一个 RPA 执行助手，帮助解决页面操作问题。

用户提供了:
1. 任务目标: {task_description}
2. 当前步骤: {current_step}
3. 失败的选择器: {failed_selector}
4. 页面截图: [截图]
5. HTML 片段: {html_snippet}

请分析页面并提供:
1. 推荐的选择器（如果可能）
2. 坐标作为降级方案
3. 详细的推理过程
4. 置信度评估

返回 JSON 格式。
```

#### 4.3.3 支持的模型（用户自配置）

```yaml
# configs/config.yaml
rpa:
  ai:
    # 脚本生成模型（文本模型）
    generator:
      enabled: true
      api_key: ${GENATOR_API_KEY}
      base_url: ${GENERATOR_BASE_URL}  # OpenAI 兼容
      model: gpt-4o-mini                 # 或 claude-3-5-sonnet-20241022

    # AI Agent 模型（视觉模型，必需）
    agent:
      enabled: true
      api_key: ${AGENT_API_KEY}
      base_url: ${AGENT_BASE_URL}    # 支持 Vision 的模型
      model: gpt-4o                    # 或 claude-3-5-sonnet-20241022, gemini-2.0-flash
      max_tokens: 4000
```

**支持的 Vision 模型：**
| 模型 | 视觉能力 | 推荐度 |
|------|---------|--------|
| GPT-4o / GPT-4o-mini | ✅ 原生多模态 | ⭐⭐⭐⭐⭐ |
| Claude 3.5 Sonnet | ✅ 原生多模态 | ⭐⭐⭐⭐⭐ |
| Gemini 2.0 Flash | ✅ 原生多模态 | ⭐⭐⭐⭐ |
| GPT-4V (旧) | ✅ 视觉能力 | ⭐⭐⭐ |

### 4.4 Worker 混合执行引擎

```go
// rpa-worker/src/executor/HybridExecutor.ts

type HybridExecutor struct {
    selectorExecutor *SelectorExecutor  // 传统选择器执行
    agentExecutor    *AgentExecutor     // AI Agent 执行
    maxRetries       int                  // 最大重试次数
}

func (e *HybridExecutor) Execute(ctx context.Context, script *Script) (*ExecutionResult, error) {
    // 第一轮：使用传统选择器
    result, err := e.selectorExecutor.Execute(ctx, script)
    if err == nil {
        return result, nil
    }

    // 记录失败
    log.Printf("Selector execution failed: %v", err)

    // 降级到 AI Agent
    if e.maxRetries > 0 {
        log.Printf("Falling back to AI Agent...")
        return e.agentExecutor.ExecuteWithRetry(ctx, script, e.maxRetries)
    }

    return nil, err
}
```

### 4.5 AI Agent 执行流程

```typescript
// rpa-worker/src/executor/AgentExecutor.ts

class AgentExecutor {
    async executeWithRetry(script: Script, maxRetries: number): Promise<ExecutionResult> {
        let currentState = await this.captureState(page);
        let stepIndex = 0;

        while (stepIndex < script.actions.length && maxRetries > 0) {
            const action = script.actions[stepIndex];

            // 尝试执行
            try {
                await this.executeAction(page, action);
                stepIndex++;
            } catch (error) {
                // 执行失败，请求 AI Agent 帮助
                const aiAction = await this.requestAIHelp({
                    task: script.description,
                    currentStep: stepIndex,
                    failedAction: action,
                    currentState: currentState,
                    error: error.message
                });

                if (aiAction.confidence > 0.7) {
                    // 置信度足够，执行 AI 推荐的操作
                    await this.executeAction(page, aiAction);

                    // 如果 AI 提供了更好的选择器，更新脚本
                    if (aiAction.selector && aiAction.selector !== action.selector) {
                        script.actions[stepIndex].selector = aiAction.selector;
                        this.recordFix(action.selector, aiAction.selector);
                    }

                    stepIndex++;
                } else {
                    // 置信度不够，重试
                    maxRetries--;
                    if (maxRetries <= 0) {
                        throw new Error("AI Agent unable to resolve: " + error.message);
                    }
                }
            }

            // 更新状态
            currentState = await this.captureState(page);
        }

        return { success: true };
    }

    async requestAIHelp(request: AgentRequest): Promise<AgentAction> {
        // 调用后端 AI API
        const response = await fetch(`${this.apiBaseURL}/api/v1/rpa/ai/decide`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(request)
        });

        return response.json();
    }
}
```

### 4.6 扩展 NoticeHub 支持 RPA

```go
// internal/websocket/notice_hub.go 扩展

type RPAProgressMessage struct {
    Type        string `json:"type"` // rpa_progress
    ExecutionID string `json:"executionId"`
    TaskID      string `json:"taskId"`
    TaskName    string `json:"taskName"`
    Step        int    `json:"step"`
    Total       int    `json:"total"`
    Message     string `json:"message"`
    Status      string `json:"status"`
    Timestamp   int64  `json:"timestamp"`
}

// 添加 RPA 相关消息类型
const (
    MessageTypeRPAProgress   = "rpa_progress"
    MessageTypeRPACompleted  = "rpa_completed"
    MessageTypeRPAFailed     = "rpa_failed"
)
```

### 4.5 自动扩缩容服务

```go
// internal/services/rpa/scaling_service.go

type ScalingService struct {
    redis      *redis.Client
    dockerClient *DockerClient
    minWorkers int32  // 最小 2
    maxWorkers int32  // 最大 10
}

// 监控队列并扩缩容
func (s *ScalingService) MonitorAndScale(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            queueLen, _ := s.redis.XLen(ctx, "rpa:task:pending").Result()

            if queueLen > 20 {
                s.scaleUp()    // docker-compose up -d --scale rpa-worker=+1
            } else if queueLen < 5 {
                s.scaleDown()  // docker-compose stop rpa-worker-N
            }
        }
    }
}
```

## 五、Worker 实现（Docker）

### 5.1 Worker 代码结构

```
rpa-worker/
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── src/
│   ├── main.go                   # 入口
│   ├── worker/
│   │   └── RPAWorker.ts          # Worker 主类
│   ├── executor/
│   │   ├── ExecutionEngine.ts    # 执行引擎
│   │   ├── ActionExecutor.ts     # 动作执行器
│   │   └── actions/              # 具体动作
│   ├── communication/
│   │   ├── RedisClient.ts        # Redis 连接
│   │   ├── APIClient.ts          # HTTP API 客户端
│   │   └── ProgressReporter.ts   # 进度上报
│   └── types/
│       └── task.ts               # 类型定义
└── scripts/
    └── install.sh                # 安装脚本
```

### 5.2 Dockerfile

```dockerfile
FROM mcr.microsoft.com/playwright:v1.48.0-jammy

# 安装 Go
RUN apt-get update && apt-get install -y \
    curl \
    && curl -fsSL https://go.dev/dl/go1.21.0.linux-amd64.tar.gz | tar -C /usr/local -xzv

WORKDIR /app

# 复制依赖
COPY go.* ./
RUN go mod download

# 复制源码
COPY ./

# 构建
RUN CGO_ENABLED=0 go build -o rpa-worker ./src/main.go

# 创建目录
RUN mkdir -p /app/downloads /app/screenshots

EXPOSE 3000

CMD ["./rpa-worker"]
```

### 5.3 docker-compose.yml

```yaml
version: '3.8'

services:
  rpa-worker:
    build: .
    image: xingran-rpa-worker:latest
    deploy:
      replicas: 2  # 默认 2 个副本
    restart: unless-stopped
    environment:
      - WORKER_ID=${WORKER_ID:-worker-1}
      - WORKER_NAME=${WORKER_NAME:-RPA Worker 1}
      - REDIS_ADDR=10.62.10.34:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - API_BASE_URL=http://10.62.10.33:9000
      - API_TOKEN=${API_TOKEN}
      - MAX_CONCURRENCY=3
      - HEADLESS=true
    volumes:
      - ./downloads:/app/downloads
      - ./screenshots:/app/screenshots
    networks:
      - xingran-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/health"]
      interval: 30s
      timeout: 10s
      retries: 3

networks:
  xingran-network:
    external: true
```

### 5.4 Worker 核心逻辑

```go
// src/main.go

func main() {
    // 1. 注册到主服务
    registerWorker()

    // 2. 启动心跳协程
    go heartbeatLoop()

    // 3. 启动任务消费循环
    for {
        task := readFromRedis()
        result := execute(task)
        reportResult(result)
    }
}

// 从 Redis Streams 读取任务
func readFromRedis() *Task {
    result := redis.XReadGroup(ctx, &redis.XReadGroupArgs{
        Group:    "rpa-workers",
        Consumer: workerID,
        Streams:  []string{"rpa:task:pending", ">"},
        Count:    1,
        Block:    5000, // 阻塞 5 秒
    })
    // ...
}
```

## 六、前端实现

### 6.1 目录结构

```
src/pages/operations/rpa/
├── index.tsx                      # RPA 管理入口（Tab 切换）
├── tasks/
│   ├── index.tsx                  # 任务管理页面
│   ├── columns.tsx                # 表格列定义
│   ├── modals/
│   │   ├── TaskEditModal.tsx      # 编辑弹窗
│   │   └── AIScriptEditor.tsx     # AI 脚本编辑器
│   └── hooks/useTasks.ts
├── executions/
│   ├── index.tsx                  # 执行记录页面
│   ├── columns.tsx
│   ├── ExecutionDetailModal.tsx   # 详情弹窗（实时日志）
│   └── hooks/useRealtimeLogs.ts   # 实时日志 Hook
└── types.ts
```

### 6.2 AI 脚本编辑器

```
功能：
1. 自然语言输入框（描述需求）
2. AI 生成按钮
3. 可视化动作列表
4. 动作参数编辑
5. 脚本预览（JSON）
6. 测试运行按钮
```

### 6.3 扩展 WebSocket Hook

```typescript
// src/hooks/useWebSocket.ts 扩展

interface RPAProgressMessage {
  type: 'rpa_progress' | 'rpa_completed' | 'rpa_failed'
  executionId: string
  taskId: string
  step: number
  total: number
  message: string
  status: string
}

// 在现有 WebSocket 消息处理中添加 RPA 消息类型
```

## 七、配置文件

### 7.1 后端配置 (configs/config.yaml)

```yaml
# RPA 配置
rpa:
  enabled: true

  # AI 配置（混合模式）
  ai:
    # 脚本生成模型（文本模型）
    generator:
      enabled: true
      api_key: ${RPA_AI_GENERATOR_KEY}
      base_url: ${RPA_AI_GENERATOR_URL}    # OpenAI 兼容 API
      model: gpt-4o-mini                   # 推荐: 快速便宜
      max_tokens: 4000

    # AI Agent 模型（视觉模型，必需）
    agent:
      enabled: true
      api_key: ${RPA_AI_AGENT_KEY}
      base_url: ${RPA_AI_AGENT_URL}       # 支持 Vision 的模型
      model: gpt-4o                       # 推荐: gpt-4o, claude-3-5-sonnet-20241022
      max_tokens: 8000
      # 备选模型: claude-3-5-sonnet-20241022, gemini-2.0-flash

    # 降级策略
    fallback:
      max_retries: 3                      # 选择器失败后重试次数
      enable_agent: true                  # 启用 AI Agent 降级
      timeout: 30s                        # AI 请求超时

  # Worker 配置
  worker:
    min_workers: 2
    max_workers: 10
    heartbeat_interval: 30s
    task_timeout: 300s

  # 存储配置
  storage:
    downloads_dir: ./uploads/rpa/downloads
    screenshots_dir: ./uploads/rpa/screenshots
    max_retained_days: 7
```

### 7.2 Worker 配置 (.env)

```bash
WORKER_ID=worker-1
WORKER_NAME=RPA Worker 1
REDIS_ADDR=10.62.10.34:6379
REDIS_PASSWORD=your_password
API_BASE_URL=http://10.62.10.33:9000
API_TOKEN=your_api_token
MAX_CONCURRENCY=3
HEADLESS=true
```

## 八、部署流程

### 8.1 数据库迁移

```bash
# 迁移文件已存在，启动主服务时自动执行
go run cmd/main.go
```

### 8.2 启动 Worker

```bash
cd rpa-worker

# 构建镜像
docker build -t xingran-rpa-worker:latest .

# 启动 2 个 Worker
docker-compose up -d --scale rpa-worker=2

# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 8.3 管理命令

```bash
# 扩容到 5 个 Worker
docker-compose up -d --scale rpa-worker=5

# 缩容到 2 个 Worker
docker-compose up -d --scale rpa-worker=2

# 停止所有
docker-compose down
```

## 九、关键文件清单

### 新增后端文件

| 文件 | 说明 |
|------|------|
| `internal/api/v1/rpa/rpa_router.go` | 路由注册 |
| `internal/api/v1/rpa/task_handler.go` | 任务处理器 |
| `internal/api/v1/rpa/worker_handler.go` | Worker 处理器 |
| `internal/api/v1/rpa/execution_handler.go` | 执行记录处理器 |
| `internal/services/rpa/ai_service.go` | AI 服务 |
| `internal/services/rpa/scaling_service.go` | 扩缩容服务 |
| `internal/services/rpa/execution_service.go` | 执行记录服务 |

### 修改的现有文件

| 文件 | 修改内容 |
|------|----------|
| `internal/api/router.go` | 添加 RPA 路由组 |
| `internal/core/core.go` | 添加 RPA 服务初始化 |
| `internal/websocket/notice_hub.go` | 扩展 RPA 消息类型 |

### 新增前端文件

| 文件 | 说明 |
|------|------|
| `src/pages/operations/rpa/index.tsx` | RPA 管理入口 |
| `src/pages/operations/rpa/tasks/index.tsx` | 任务管理页面 |
| `src/pages/operations/rpa/tasks/modals/AIScriptEditor.tsx` | AI 脚本编辑器 |
| `src/pages/operations/rpa/executions/index.tsx` | 执行记录页面 |
| `src/lib/rpaApi.ts` | RPA API 客户端 |
| `src/types/rpa.ts` | RPA 类型定义 |

### Worker 文件

| 文件 | 说明 |
|------|------|
| `rpa-worker/Dockerfile` | Docker 镜像 |
| `rpa-worker/docker-compose.yml` | 编排配置 |
| `rpa-worker/src/main.go` | Worker 入口 |
| `rpa-worker/src/worker/RPAWorker.ts` | Worker 主类 |

## 十、验证测试

### 10.1 功能测试

1. **AI 脚本生成**
   - 输入描述 → 生成脚本
   - 查看生成的动作序列
   - 保存任务

2. **任务执行**
   - 手动执行任务
   - 查看 Worker 列表状态
   - 查看执行进度（实时）

3. **Worker 管理**
   - 查看在线 Worker
   - Docker Compose 扩缩容
   - 查看心跳状态

4. **执行记录**
   - 查看历史记录
   - 查看执行日志
   - 查看截图

### 10.2 集成测试

```bash
# 1. 启动主服务
go run cmd/main.go

# 2. 启动 Worker
cd rpa-worker && docker-compose up -d

# 3. 前端访问
# http://localhost:4000/operations/rpa

# 4. 创建并执行任务
# 验证 Worker 消费任务
# 验证进度推送
```

## 十一、分阶段实施

### 第一阶段（MVP）- 混合模式基础

**目标**: 实现传统选择器 + AI Agent 降级的基础框架

1. 后端基础 API
   - 任务 CRUD、执行记录、Worker 管理
   - AI 服务（脚本生成、Agent 决策）
   - 扩展 NoticeHub 支持 RPA 消息

2. 前端基础页面
   - 任务管理列表、AI 脚本编辑器
   - 执行记录列表

3. Worker Docker
   - 混合执行引擎（选择器优先，Agent 降级）
   - Redis 任务消费、HTTP 心跳

### 第二阶段 - AI Agent 增强

1. AI 功能完善
   - 脚本优化、错误分析、智能修复
   - 选择器自动学习和更新

2. 实时进度
   - WebSocket 进度推送
   - 日志流式显示、截图关联

3. 自动扩缩容
   - 监控队列长度、Docker Compose 扩缩容

### 第三阶段 - 高级功能

1. 浏览器插件录制（可选）
2. 可视化拖拽编辑器（可选）
3. 高级功能：条件分支、循环、子流程

## 十二、AI Agent 成本与性能分析

### 12.1 Token 消耗估算

#### 单次 AI Agent 调用

```
输入（降级时）:
- 任务描述: ~200 tokens
- 当前步骤: ~100 tokens
- 失败信息: ~200 tokens
- 页面截图 (base64): ~1000 tokens (压缩后)
- HTML 片段: ~2000 tokens
- 对话历史: ~500 tokens
────────────────────────────────────
输入总计: ~4000 tokens

输出:
- 推荐动作: ~200 tokens
- 推理过程: ~500 tokens
- 坐标/选择器: ~100 tokens
────────────────────────────────────
输出总计: ~800 tokens

单次调用: ~4800 tokens
```

#### 典型场景的 Token 消耗

| 场景 | 选择器成功率 | AI 调用次数 | Token 消耗 | 成本 (GPT-4o) |
|------|-------------|-----------|-----------|---------------|
| 稳定页面 | 95%+ | 0.5 次/任务 | 2.4K | $0.0001 |
| 一般页面 | 80% | 2 次/任务 | 9.6K | $0.0003 |
| 动态页面 | 50% | 5 次/任务 | 24K | $0.0008 |
| 复杂页面 | 20% | 15 次/任务 | 72K | $0.0025 |

**成本计算（GPT-4o 定价）**：
- 输入: $2.50 / 1M tokens
- 输出: $10.00 / 1M tokens
- 单次调用: 约 $0.0001 - $0.0003

**结论**: 即使是最复杂的场景，单次执行的 AI 成本也不到 $0.01

### 12.2 性能对比

| 执行模式 | 速度 | 可靠性 | 成本 | 适用场景 |
|---------|------|--------|------|----------|
| 纯选择器 | 毫秒级 | 高 | ~$0 | 稳定页面、高频任务 |
| 混合模式 | 秒级 | 中 | $0.0001-$0.01 | 通用场景 |
| 纯 AI Agent | 5-10秒/步 | 中 | $0.10-$1.00 | 探索性任务 |

### 12.3 模型选择建议

| 模型 | 视觉能力 | 速度 | 成本 | 推荐场景 |
|------|---------|------|------|----------|
| **GPT-4o-mini** | ✅ | 快 (1-2s) | 低 | **推荐**（性价比最高）|
| **Claude 3.5 Sonnet** | ✅ | 中 (1-3s) | 中 | 复杂推理 |
| **Gemini 2.0 Flash** | ✅ | 快 (0.5-1s) | 低 | 快速响应 |
| **GPT-4o** | ✅ | 快 (1-2s) | 中 | 高精度需求 |

### 12.4 混合模式的优势总结

```
┌─────────────────────────────────────────────────────────────────┐
│                   混合模式价值分析                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  成本效益:                                                       │
│  - 稳定场景: 仅使用选择器，零 AI 成本                            │
│  - 一般场景: 1-2 次 AI 降级，$0.0003/次                            │
│  - 复杂场景: 5 次 AI 降级，$0.0025/次                              │
│                                                                 │
│  对比纯 AI Agent:                                               │
│  - 每步都需要 AI: 20 步 × $0.02 = $0.40/次                         │
│  - 混合模式: 最差情况 $0.0025/次                                  │
│                                                                 │
│  性能对比:                                                       │
│  - 选择器执行: 10ms/步                                           │
│  - AI 降级: 2s/步（仅失败时）                                     │
│  - 平均提升: 仅增加 5-10% 总时间（仅失败步骤）                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 十三、依赖项

| 依赖 | 版本 | 用途 |
|------|------|------|
| Playwright | v1.48 | 浏览器自动化 |
| Redis Streams | - | 任务队列 |
| OpenAI API | - | AI 辅助 |
| Docker Compose | v2+ | 容器编排 |
| WebSocket | - | 实时通信 |
