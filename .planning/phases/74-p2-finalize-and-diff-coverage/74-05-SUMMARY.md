---
phase: 74
plan: 05
subsystem: rpa-services-coverage
tags: [coverage, service-tests, sqlite, regression, p2-finalize]
dependency_graph:
  requires: [phase-73-service-patterns, phase-72-p0-core-supplement]
  provides: [rpa-services-test-suite, worker-register-pk-regression]
  affects: [internal/services/rpa]
tech_stack:
  added: []
  patterns:
    - glebarez/go-sqlite RegisterScalarFunction 注入 PG 兼容函数 (gen_random_uuid/NOW) 跑原生 SQL 路径
    - 接口嵌入 nil 兜底 + 覆写单个方法的 cache/DockerClient/MetricsService 假实现
    - OpenAI 兼容 API httptest 服务器驱动 AIClient/AIService/ErrorAnalyzer
    - excelize 内存构造 + multipart 包装喂 ParseExcelFile
key-files:
  created:
    - internal/services/rpa/flow_control_test.go
    - internal/services/rpa/data_mapper_test.go
    - internal/services/rpa/worker_service_test.go
    - internal/services/rpa/credential_service_test.go
    - internal/services/rpa/task_execution_service_test.go
    - internal/services/rpa/metrics_scaling_docker_test.go
    - internal/services/rpa/ai_selector_excel_test.go
  modified: []
decisions:
  - id: D-12-STRICT
    summary: 仅 7 个 *_test.go 入库, go.mod/go.sum 零改动 (glebarez/go-sqlite 以 indirect 身份直接导入)
  - id: D-15-P2-FLOOR
    summary: services/rpa 1.1% → 86.1% (≥70%)
  - id: PK-NULL-REGRESSION
    summary: f0d0a1f Worker Register 23502 修复以行为级回归锁定
metrics:
  completed_date: 2026-08-22
  baseline_coverage: 1.1
  final_coverage: 86.1
  coverage_delta: 85.0
  test_files_added: 7
---

# Phase 74 Plan 05: RPA Services Tests Summary

**One-liner:** `internal/services/rpa`（1865 stmts，Phase 73 交接时 1.1%）经 7 个测试文件推至 **86.1%**，含 f0d0a1f（Worker Register NOT NULL 23502）的行为级回归锁定。

## PK NULL 回归测试（Plan Task 2 强制项）

`TestWorkerService_Register_PKNotNullRegression`（worker_service_test.go）：

- sqlite 表定义 `id TEXT PRIMARY KEY NOT NULL`；
- 通过 `glebarez/go-sqlite` 的 `RegisterScalarFunction` 注入 `gen_random_uuid()` 与 `NOW()` 标量函数（依赖已在 go.mod indirect 列表，直接导入不改 go.mod），使 Register 的原生 `INSERT ... ON CONFLICT ... RETURNING *` 在 sqlite 完整执行；
- 若有人把 `id` 从 INSERT 列清单移除（回退 f0d0a1f），sqlite 将收到 NULL 主键 → 约束违规 → 无行落库 → 测试失败；
- 同时覆盖 ON CONFLICT DO UPDATE 路径（同 worker_id 二次注册不产生重复行、字段被更新）。
- 已知测试环境差异：`RETURNING *` 把 TEXT `capabilities` 扫进 `json.RawMessage` 在 sqlite 报 Scan 错（PG jsonb 正常）——INSERT 已成功落库，回归断言以 DB 行的 id 为准。

## Files Created（7 files）

| File | Covers |
|------|--------|
| `flow_control_test.go` | 表达式求值器全操作符、ExecuteCondition/ExecuteLoop 四种循环、SimpleConditionEvaluator、utils.go 纯函数 |
| `data_mapper_test.go` | 15 种 Transform、JSONPath、9 种聚合、MapData 严格/宽松模式、lookup/aggregate 规则、toMap 反射 |
| `worker_service_test.go` | Register 回归、Heartbeat、Progress（截图落盘+WS 发布+failed 路径）、List/GetByID/GetAvailable/Offline/CheckOfflineWorkers |
| `credential_service_test.go` | 凭证 CRUD+加密、List 四种过滤、执行凭证优先级、会话生命周期、登录追踪 |
| `task_execution_service_test.go` | 任务 CRUD/Execute 三条发布分支（DirectRedisXAdd/getClient nil/不支持类型）、凭证会话兜底、执行记录全生命周期、错误处理 6 策略+重试退避+降级 |
| `metrics_scaling_docker_test.go` | MetricsService 全查询、MetricsCollector、扩缩容决策数学、ValidateScalingConfig、真实 DockerClient HTTP 全端点、MockDockerClient、SyncWorkersWithContainers |
| `ai_selector_excel_test.go` | AIClient 六种错误路径、AIService 生成/优化/决策、ErrorAnalyzer 本地分类 7 类+AI 降级、选择器学习全流程、Excel 解析+人工干预、ServiceGroup 装配 |

## Documented Quirks（D-12 — 不修业务码）

1. **task_service.List 的 Name 过滤用 `task_name` 列名，但 Task 模型列是 `name`**（gorm column:name）→ 带 Name 过滤的列表查询在任何库都会报 no such column。
2. **publishTaskToRedis 凭证兜底路径对 nil map 赋值 panic**：`ExecuteTaskRequest.InputParams` 为 nil 且命中凭证时 `message.Variables["__credentials"] = ...` 直接 panic（测试以非 nil InputParams 规避）。
3. **ScoreSelector 从不设置 SelectorStats.SuccessRate**（恒 0），且 raw INSERT 的时间戳在 sqlite 回读解析近零 → 得分只剩 usage 项；与 GetBestSelector/GetSelectorAlternatives（用 calculateSelectorStats 正确设 rate）行为不一致。
4. **SanitizeLogMessage 用 strings.ReplaceAll 做"正则"**：pattern 字面量 `password=[^\s]*` 永远不会命中真实日志，脱敏实际无效。
5. **RetryPolicy.MaxDelay 零值钳制**：calculateDelay 的 `if delay > MaxDelay` 在零值时把任何 InitialDelay 钳为 0，使退避等待失效（并造成 select 竞态）。
6. **RedisCache 分支类型断言永不命中**：redisCacheInterface（未导出 getClient）在包外类型上不满足，实际只有 MultiLevelCache 分支可达 —— 用同包测试类型才覆盖到 nil-client 错误路径。
7. **UpdateTaskRequest.Status=0 无法重新启用任务**（`if req.Status > 0` 才更新）。

## Constraints Honored

- D-12 STRICT: 仅 `*_test.go`（commit diff 验证，go.mod/go.sum 零改动）
- D-15: 86.1% ≥ 70%
- f0d0a1f 回归锁定（Plan Task 2）
- No STATE.md/ROADMAP.md updates（orchestrator-owned）；no push
