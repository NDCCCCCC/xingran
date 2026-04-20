---
phase: 04-widget-data-fetching
plan: 02
status: complete
completed: 2026-04-20
requirements:
  - WDG-01c
---

# Summary: 数据转换管道

## What Was Built

使用 JSONata 表达式引擎实现了数据转换功能，支持表达式求值、聚合函数、排序和限制。

## Key Files Created/Modified

| File | Purpose |
|------|---------|
| `internal/transform/jsonata_evaluator.go` | JSONata 表达式求值引擎 |
| `internal/transform/jsonata_evaluator_test.go` | 单元测试 |
| `internal/services/system/widget_data_fetcher.go` | 集成数据转换 |
| `go.mod` | 添加 JSONata-go 依赖 |

## Implementation Details

### 1. JSONataEvaluator 结构

- `maxTime`: 最大执行时间（1秒），防止无限循环
- `maxDepth`: 最大递归深度（100），防止栈溢出

### 2. 核心方法

- **Evaluate**: 执行 JSONata 表达式求值
- **ApplyAggregate**: 应用聚合函数（sum、avg、count、min、max）
- **ApplyOrderBy**: 应用排序（升序/降序）
- **ApplyLimit**: 应用结果数量限制
- **Transform**: 完整转换管道（expression → aggregate → orderBy → limit）

### 3. 安全限制

- 表达式执行超时限制（1秒）
- 递归深度限制（100层）
- 防止表达式注入攻击

### 4. WidgetDataFetcher 集成

- API 数据源获取数据后自动应用转换配置
- 转换失败返回错误，不缓存无效数据

## Test Results

```
=== RUN   TestJSONataEvaluator_Evaluate
--- PASS: TestJSONataEvaluator_Evaluate (0.00s)
=== RUN   TestJSONataEvaluator_Aggregate
--- PASS: TestJSONataEvaluator_Aggregate (0.00s)
=== RUN   TestJSONataEvaluator_OrderBy
--- PASS: TestJSONataEvaluator_OrderBy (0.00s)
=== RUN   TestJSONataEvaluator_Limit
--- PASS: TestJSONataEvaluator_Limit (0.00s)
=== RUN   TestJSONataEvaluator_Transform
--- PASS: TestJSONataEvaluator_Transform (0.00s)
PASS
```

## Success Criteria Met

- [x] 表达式 `data.items[0].value` 可以正确求值
- [x] 聚合函数 sum、avg、count、min、max 正常工作
- [x] 排序功能支持升序和降序
- [x] 限制功能可以限制结果数量
- [x] 表达式执行有安全限制（maxTime、maxDepth）
- [x] 转换管道按顺序执行：表达式 → 聚合 → 排序 → 限制

## Next Steps

Wave 3 将实现批量获取和缓存（使用 errgroup 并行获取）。
