# RPA Worker 选择器上报明细契约（TIDY-03 :256 方案 A）

> 维护人：qualguard ｜ 2026-11-01 ｜ 消费方：`SelectorLearner.LearnFromExecution`
> 实现位置：`internal/services/rpa/selector_learner.go`（selectorEventEnvelope）

## 1. 数据通道

**零 migration**：执行记录 `sys_rpa_executions.logs`（TEXT，自由文本）中允许嵌入
一行合法 JSON。后端解析按**行**切分：含 `selectorEvents` 键的 JSON 行被消费，
其余行（普通日志文本）静默跳过——结构化明细与普通日志共存安全。

## 2. JSON 结构

```json
{"selectorEvents":[
  {
    "pageUrl":     "http://target/login",     // 必填：页面 URL
    "elementId":   "btn-submit",              // 必填：元素标识
    "selector":    "#submit",                 // 必填：实际使用的选择器
    "selectorType":"css",                     // 可选：css|xpath|text|aria|data-testid
    "success":     true,                      // 必填：本次命中结果
    "durationMs":  120,                       // success 事件有意义：查找耗时（毫秒）
    "errorType":   "not_found",               // failed 事件：timeout|not_found|multiple_matches
    "errorMessage":"element not found"        // failed 事件：原始错误
  }
]}
```

## 3. 后端处理语义

| 事件 | 路由 | 落库 |
|---|---|---|
| `success:true` | `RecordSuccess` | `selector_success_records` upsert（计数+1、平均时长、缓存失效） |
| `success:false` | `RecordFailure` | `selector_failure_records`（FailureCount 累加；**≥3 触发 `selector_update_required` WebSocket 广播**） |

- 缺 pageUrl/elementId/selector 任一必填项的事件：跳过 + WARN（不中断整批）
- 单事件落库失败：WARN + 继续；全部失败时返回首个错误
- `consumed=0` 且无错误：返回明确错误「未解析出任何选择器明细」（调用方降级，不静默吞）

## 4. worker 侧适配要点

1. 执行过程中每次选择器定位（成功或失败）累积一个事件对象；
2. 执行结束时把事件数组序列化为一行 JSON，**追加**写入执行记录的日志上报体
   （worker 现有日志上报通道即可，无需新端点）；
3. 上报频率：一次执行一条信封行（事件全量随执行收尾一次性上报，不做逐条实时）。

## 5. 与 :389 的闭环

失败事件累计 `FailureCount >= 3` 时，`markSelectorForUpdate` 经 NoticeHub
向全体在线客户端广播 `selector_update_required`（Title:「RPA 选择器需要更新」），
管理端 RPA AI 面板据此提示重新学习——学习与通知在同一入口汇合，无第二真相。
