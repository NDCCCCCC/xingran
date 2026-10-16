# RPA 数据格式规范

本文档定义了 RPA 系统中前后端和 Worker 之间传输数据的统一格式，避免格式不匹配问题。

## 数据流概览

```
前端 → 后端 → Worker
     ↓       ↓
   存储层   Redis
```

## 数据格式定义

### 1. 前端 → 后端（创建/更新任务）

**格式**: `ScriptAction[]`

```typescript
interface ScriptAction {
  type: string;           // 动作类型: navigate, click, fill, select, wait, etc.
  selector: string;       // CSS 选择器
  value: string;          // 填充值（URL/输入内容/选项）
  attributes: {           // 其他属性
    description?: string; // 动作描述
    duration?: number;    // 等待时长（毫秒，仅 wait 动作）
    [key: string]: any;
  };
  timeout?: number;       // 超时时间（毫秒）
  retry?: number;         // 重试次数
}
```

**示例**:
```json
[
  {
    "type": "navigate",
    "value": "https://www.baidu.com",
    "attributes": {
      "description": "打开百度首页"
    },
    "timeout": 30000,
    "retry": 0
  },
  {
    "type": "fill",
    "selector": "#kw",
    "value": "RPA测试",
    "attributes": {
      "description": "输入搜索关键词"
    },
    "timeout": 30000,
    "retry": 0
  },
  {
    "type": "wait",
    "attributes": {
      "description": "等待页面加载",
      "duration": 2000
    },
    "timeout": 30000,
    "retry": 0
  }
]
```

### 2. 后端存储（数据库）

**格式**: 与前端相同，使用 `ScriptAction` 存储为 JSONB

### 3. 后端 → Worker（执行任务）

**格式**: `WorkerAction[]` (与 Worker 的 Action 结构体一致)

```go
type WorkerAction struct {
    ID          string                 `json:"id"`
    Type        string                 `json:"type"`
    Description string                 `json:"description"`      // 顶层字段
    Selector    string                 `json:"selector,omitempty"`
    Params      map[string]interface{} `json:"params,omitempty"`  // 参数都在这里
    Timeout     int                    `json:"timeout,omitempty"`
    Retry       int                    `json:"retry,omitempty"`
    Value       string                 `json:"value,omitempty"`
    AIAssisted  bool                   `json:"aiAssisted,omitempty"`
}
```

**转换规则**:

| 字段 | ScriptAction (前端/后端) | WorkerAction (Worker) |
|------|-------------------------|----------------------|
| 描述 | `attributes.description` | `description` (顶层) |
| URL (navigate) | `value` | `params.url` |
| 填充值 (fill/select) | `value` | `params.value` |
| 等待时长 (wait) | `attributes.duration` | `params.duration` |
| 其他属性 | `attributes.*` | `params.*` |

**示例**:
```json
[
  {
    "id": "action_0",
    "type": "navigate",
    "description": "打开百度首页",
    "selector": "",
    "params": {
      "url": "https://www.baidu.com"
    },
    "timeout": 30000,
    "retry": 0
  },
  {
    "id": "action_1",
    "type": "fill",
    "description": "输入搜索关键词",
    "selector": "#kw",
    "params": {
      "value": "RPA测试"
    },
    "timeout": 30000,
    "retry": 0
  },
  {
    "id": "action_2",
    "type": "wait",
    "description": "等待页面加载",
    "params": {
      "duration": 2000
    },
    "timeout": 30000,
    "retry": 0
  }
]
```

## 关键要点

1. **前端发送 ScriptAction 格式** - 使用 `attributes` 存储扩展属性
2. **后端进行转换** - 在 `convertToWorkerAction` 函数中转换格式
3. **Worker 接收 WorkerAction 格式** - 描述在顶层，参数在 `params` 中
4. **navigate 动作特殊处理** - URL 必须在 `params.url` 中
5. **wait 动作特殊处理** - 时长必须在 `params.duration` 中

## 实现位置

| 组件 | 文件 | 职责 |
|------|------|------|
| 前端编辑 | `EditModal.tsx` | 构建 ScriptAction 格式 |
| 后端转换 | `task_service.go` | `convertToWorkerAction` 函数 |
| 后端发送 | `task_service.go` | `publishTaskToRedis` 函数 |
| Worker 映射 | `engine.go` | `mapToAction` 函数（已简化） |
| Worker 浏览器 | `chrome_page.go` | 创建浏览器页面 |

## 修改记录

- **2025-01-10**: 统一数据格式，在发送前转换，避免 Worker 端复杂处理
  - 添加 `WorkerAction` 结构体
  - 添加 `convertToWorkerAction` 转换函数
  - 简化 Worker 的 `mapToAction` 函数

- **2025-01-10**: 修复 rod API 误用导致的 ERR_UNKNOWN_URL_SCHEME 错误
  - 问题: `browser.MustPage(url)` 会尝试导航到调试 URL
  - 修复: 改为 `browser.MustPage()` 创建空白页面
  - 文件: `chrome_page.go`

## 常见问题

### ERR_UNKNOWN_URL_SCHEME 错误

**原因**: 使用了错误的 rod API：
```go
// ❌ 错误 - 会尝试导航到调试 URL
page := browser.MustPage(url)

// ✅ 正确 - 创建空白页面
page := browser.MustPage()
```

**调试 URL 格式**: `ws://localhost:XXXX` - 这是 Chrome DevTools Protocol 的 WebSocket 地址，不是 HTTP URL。
