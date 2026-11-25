# Phase 158 SUMMARY — CLEANUP 收尾真相化（CLEANUP-01~03）

| 项 | 值 |
|---|---|
| **Phase** | 158（W5 波次，v1.56 里程碑第 7 个 phase） |
| **执行日期** | 2026-11-25 |
| **需求映射** | CLEANUP-01 / CLEANUP-02 / CLEANUP-03（`.planning/REQUIREMENTS.md:95-97`） |
| **前置依赖** | Phase 153（`b69c8784`）/ 155（`70405142`）/ 157（`51dca251`）**均已合入**，开工前 `git log --oneline -10` 已确认 |
| **状态口径** | `code-complete · verification-pending` —— **未 SHIPPED**（依 `.planning/notes/SHIPPED-DEFINITION.md` §2/§4：缺真实环境 UAT 证据只允许此状态） |
| **分支 / 提交** | `main`｜见本 phase 提交（commit sha 记于交付回报） |

---

## 0. 一句话结论

三条 REQ 全部**代码侧落地且有本机实证**：假数据/假配置清零（读写真值可验）、
`sys_rpa_schedules` 从「零服务零 API」补成端到端闭环并**本机真跑到点触发**、
flow 组死代码 2885 行整体删除并完成文档真相化。
**但真实环境（生产 PG + 现场真机 VDI + 真实 Worker 集群）验收证据未落盘，一律登记为 verification-pending（见 §6）。**

---

## 1. CLEANUP-01 假数据与假配置清零 —— ✅ PASS

### 1.1 批量执行报告：真落库 + 真统计

| 项 | 处置 |
|---|---|
| 假数据源 | `internal/services/rpa/excel_service.go:270` `GetBatchExecutionReport` **硬编码返回** `total=100 / success=95 / failed=5 / skipped=0` |
| 存根 | `CreateBatchExecutionReport:219`（构造内存 report 后直接丢弃）、`UpdateBatchItemReport:254`（空壳，注释写「实际实现需要读取缓存」） |
| 新表 | `internal/models/rpa/batch_report.go` → `sys_rpa_batch_item_reports`（一行 = 一个批量项：execution_id / task_id / item_index / status / input_data / output_data / error / screenshot / start_time / end_time / duration） |
| 写侧 | `CreateBatchExecutionReport` 批量落行（初始 `pending`）；`UpdateBatchItemReport` 按 `(execution_id, item_index)` 更新状态与产出、算 duration；**更新不存在的项返回明确错误**（不静默吞掉） |
| 读侧 | `GetBatchExecutionReport` 按项聚合：total=全部项数、success/failed/skipped 按 status 计数、pending 单列；**无数据 → total=0 + 明细空** |
| 建表 | migration 257（`HasTable` 守卫，双方言幂等）+ `internal/core/db/database.go` sqlite 分支 AutoMigrate 注册 + `scripts/dbprovision/main.go` 注册（PG 新部署） |

**验收（本机跑通）**：`internal/services/rpa/cleanup158_test.go`
- `TestCleanup01_BatchReport_NoData_ReturnsEmptyStats`：无数据 → `total=0 / success=0 / failed=0`、明细空、summary.total=0（**不是常量 100/95/5**）
- `TestCleanup01_BatchReport_ThreeItems_ReturnsRealCounts`：3 条数据 → `total=3 / success=2 / failed=1`，明细逐项状态/输出/错误/截图可读，summary 与明细一致
- `TestCleanup01_BatchReport_UpdateUnknownItem_ReturnsError`：未落库项更新报错

### 1.2 自动扩缩容配置：真持久化到 sys_config

| 项 | 处置 |
|---|---|
| 假配置源 | `worker_handler.go` `GetAutoScaleConfig` **恒返回硬编码默认值**；`UpdateAutoScaleConfig` **丢弃请求体**只回一句「已更新」 |
| 落库 | `sys_config.config_key = sys.rpa.autoscale.config`，值 = 配置 JSON；读-改-写（存在则 update、不存在则 create），**重复写不产生多行** |
| 兜底 | 键不存在时 GET 返回默认值（首次使用兜底；一旦写回，读写即一致 —— 不是假数据） |
| 重启语义 | 配置在 DB 内 → **重启后仍生效**（本机以真实 sqlite 文件库 + 真实 AutoMigrate 验证） |

**验收（本机跑通）**：`internal/api/v1/rpa/cleanup158_autoscale_test.go`
- `TestCleanup01_AutoScaleConfig_NotConfigured_ReturnsDefaults`：未配置 → 返回兜底默认值且**不凭空造 sys_config 行**
- `TestCleanup01_AutoScaleConfig_WriteThenRead_Persisted`：POST 写入值 → GET 原样读回 + DB 里有行 + 二次写不新增行

### 1.3 Redis 连接泄漏：删除每实例 redis.NewClient

| 项 | 处置 |
|---|---|
| 泄漏源 | `worker_handler.go:42` `NewWorkerHandler` **每次构造都 `redis.NewClient`**（PoolSize=10）且从不 `Close`；路由每次装配泄漏一个连接池 |
| 处置 | 删除 `WorkerHandler.redisClient` 字段与整段 `redis.NewClient` 构造；`publishScaleCommand` 改走 `h.core.Cache`（新增 `rpa.PublishScaleCommand`，与取消/人工干预同款「从 cache 断言底层客户端」路径） |
| 降级语义 | cache 未注入 / 不支持 Redis 操作时返回**可读错误**（`cache 类型不支持 Redis Pub/Sub`），由调用方决定降级；不静默吞掉 |

---

## 2. CLEANUP-02 定时调度闭环 —— ✅ PASS（本机真跑）

### 2.1 交付物

| 件 | 位置 | 说明 |
|---|---|---|
| 调度服务 | `internal/services/rpa/schedule_service.go`（新） | CRUD + cron 到期计算（5 位 / 6 位表达式）+ 立即触发 + `last_run_time`/`next_run_time`/`run_count` 回写 + 启停摘 job + `SyncEnabledJobs`（启动同步） |
| HTTP 层 | `internal/api/v1/rpa/schedule_handler.go`（新） | 7 端点：列表 / 详情 / 新增 / 编辑 / 删除 / 立即触发 / 启停；写操作记 `operlog`（模块「RPA定时调度」） |
| 路由注册 | `internal/api/v1/rpa/rpa_router.go` | **仅在 `SetupRPARouter` 尾部追加一行** `SetupScheduleRouter(r.Group("/schedules"), services, core)`（归属协议第 2 条：只追加，不改既有行；`:22-52` 零改动） |
| 调度接线 | `internal/scheduler/rpa_schedule_tasks.go`（新） | 注册 `rpa_schedule` 任务类型；每个启用调度 → 一条 cron job（`sys_job.invoke_target = rpa_schedule:{scheduleID}`，remark 记 `rpa_schedule_id=` 作幂等键）；位数归一（5 位 → 前置 `0 `） |
| core 装配 | `internal/core/core.go` | 在 `initSchedulerAndTasks` 内接线（**不随 `RPA.Enabled` 门控** —— RPA API 路由是无条件注册的，若调度被门控会出现「能建调度但永不触发」空档） |
| 权限码 | `pkg/permission/config.go` + `xingran-react-frontend/src/constants/permissionCodes.ts` + migration 257 | `ops:rpa:schedule:list/add/edit/remove` 四方（后端常量 / 前端 TS / sys_menu F 型 / 路由挂载）一致；写端点逐动作 strict 单码 |
| 迁移 | `internal/core/db/migrations/migration_257_rpa_batch_report_and_schedule_perm.go` | 建 `sys_rpa_batch_item_reports` + 补建 `sys_rpa_schedules` + 播种 4 个 F 型权限码；`HasTable` 守卫 + `runAndRecord` 版本记录 → **双方言幂等**；`registry.go` 补登记 `{257, ...}`；`database.go` **PG 分支 + sqlite 分支两处注册** |

### 2.2 触发语义（与 REQUIREMENTS 对齐）

- **共用同一执行入口**：到点与手动触发都走 `TaskService.Execute`（同一状态机、同一条 Redis Stream 通道）
- **`trigger_type=schedule`**：`ExecuteTaskRequest` 新增可选 `TriggerType`（空 = `manual`，存量调用方零改动）；调度触发时写 `trigger_type=schedule` / `triggered_by=scheduler`
- **时间回写**：每次触发后回写调度行 `last_run_time` / `next_run_time` / `run_count`
- **停用即摘 job**：停用/删除 → 对应 cron job 从调度器摘除（`sys_job` 行软删）

### 2.3 全链路真跑验证记录（本机，非模拟）

环境：真 sqlite 文件库（`db.NewDatabase` + 真实 `AutoMigrate` + 真迁移链）+ **本机 Redis 5.0.14.1**（`D:\Program Files\Redis5.0.14.1\redis-server.exe`，DSH 托管终端会话前台运行，6379）+ 真 cron 引擎（robfig/cron，秒级表达式 `0 * * * * *` = 每分钟第 0 秒）。

用例：`internal/scheduler/rpa_schedule_e2e_test.go` → `TestRPASchedule_E2E_TriggersRealExecution`

```
[2026-11-25 11:44:52] [rpa_schedule] 任务处理器已注册（InvokeTarget=rpa_schedule）
[2026-11-25 11:44:52] [rpa_schedule] 已注册调度任务: RPA调度:Phase158 每分钟验收调度
                        (schedule=2c6bce70-6048-48a3-88d8-34fd5547803c, cron=0 * * * * *)
[2026-11-25 11:45:00] 执行任务: RPA调度:Phase158 每分钟验收调度,
                        目标: rpa_schedule:2c6bce70-6048-48a3-88d8-34fd5547803c
[2026-11-25 11:45:00] [rpa_schedule] 调度已触发
                        (schedule=2c6bce70-..., execution=1801662a-6808-45f2-af4c-0e776ef97567)
--- PASS: TestRPASchedule_E2E_TriggersRealExecution (8.85s)
PASS
```

断言逐条通过：
1. `sys_job` 里有且仅有 1 条 `rpa_schedule:{scheduleID}` 的 job（启动即注册，非只在内存）
2. **到点真触发**：`sys_rpa_executions` 出现 `trigger_type='schedule'` 的执行记录（`execution=1801662a-…`）
3. 调度行回写：`run_count ≥ 1`、`last_run_time` 非空、`next_run_time` 非空
4. 停用即摘 job：停用后 `sys_job` 无该调动的未删行

> Redis 不可达时该用例 `t.Skip`（零新增依赖约束下不引入 miniredis，与 Phase 153 条件化范式一致）；
> 本次**本机 Redis 真实可达，故为真跑而非 skip**。

### 2.4 其它验收（服务层）

`internal/services/rpa/cleanup158_test.go`
- `TestCleanup02_Schedule_CRUD_Lifecycle`：非法 cron 被拒 → 创建 → 查询 → 列表 → 改 cron（job 同步更新）→ 停用（job 被摘）→ 删除（再摘一次，幂等）
- `TestCleanup02_Schedule_ComputeNextRunTime`：5 位 / 6 位表达式到期计算 + 非法 cron + 非法时区均返回可读错误
- `TestCleanup02_Schedule_TriggerNow_RejectsDisabled`：停用态拒绝触发

---

## 3. CLEANUP-03 死代码删除与文档真相化 —— ✅ PASS

### 3.1 删除清单（7 文件 / 2885 行 + 1 测试文件 184 行纯删除）

| 文件 | 行数 | 说明 |
|---|---|---|
| `internal/api/v1/rpa/flow_handler.go` | 287 | flow 组 7 端点 HTTP 层（路由已于 2026-11-11 下线，实现残留） |
| `internal/api/v1/rpa/flow_handler_test.go` | 455 | 仅测上者 |
| `internal/services/rpa/data_mapper.go` | 644 | 数据映射服务（零非测试调用方） |
| `internal/services/rpa/data_mapper_test.go` | 396 | 仅测上者 |
| `internal/services/rpa/error_handling.go` | 396 | 错误处理服务（零非测试调用方） |
| `internal/services/rpa/flow_control.go` | 422 | 流程控制服务（零非测试调用方） |
| `internal/services/rpa/flow_control_test.go` | 285 | 仅测上者 |
| **合计** | **2885** | |
| `internal/services/rpa/task_execution_service_test.go` | 602 → 418（-184，0 新增） | 删 4 个仅测被删簇的用例：`TestErrorHandlingService_Strategies` / `_Retry` / `_CalculateDelay` / `_Fallback` + 专属 helper `notTimeoutErr` + 其专属 import；**该文件其余 7 个测试与 `newTaskTestDB` 等 helper 完整保留** |

> 连带发现：`TestContainsHelper` 测的 `contains` helper 定义在被删的 `error_handling.go:388`，
> 不删则编译失败 → 判定为「仅属该簇的专属测试」，一并删除。

### 3.2 grep 取证（删前，零非测试调用方）

命令：`git grep -n -E "NewFlowHandler|NewDataMapperService|NewErrorHandlingService|NewFlowControlService|FlowControlService|ErrorHandlingService|DataMapperService" -- "*.go"`

| 符号 | 唯一非测试调用方 | 结论 |
|---|---|---|
| `NewFlowHandler` | `flow_handler_test.go:161`（测试） | 零非测试调用方 → 删 |
| `NewDataMapperService` | `data_mapper_test.go` ×5（测试） | 零非测试调用方 → 删 |
| `NewErrorHandlingService` | `task_execution_service_test.go` ×3（测试） | 零非测试调用方 → 删 |
| `NewFlowControlService` | `flow_control_test.go` ×4（测试） | 零非测试调用方 → 删 |
| 3 个 interface | 仅在死代码簇内互相引用 | 无外部消费者 → 删 |

附加取证：4 个待删文件中另有 36 个导出符号（`MappingType` / `TransformFunction` / `DataMappingRule` /
`DataMappingConfig` / `ErrorHandlingConfig` / `RetryPolicy` / `ErrorHandleRequest` / `RetryRequest` /
`FallbackRequest` / `ConditionAction` / `LoopAction` / `ExpressionEvaluator` / `SimpleConditionEvaluator` 等），
外部引用**只出现在**待删测试内；包内私有符号（`toFloat64` / `toMap` / `extractValue` / `getField` /
`setNestedValue` / `intPtr` / `contains` / `findInString` / `toSlice` / `varPlaceholderRegex`）无其它文件使用。

**删后复验**：全仓 1828 个 `.go` 文件（含未跟踪）**零残留命中**；
`go build ./...` 绿、`go test ./internal/services/rpa/ ./internal/api/v1/rpa/` 绿。

### 3.3 豁免项

**无。** 7 个文件全部删除，无任何被删文件仍被非测试代码引用；未触碰任何豁免文件。

### 3.4 文档更新清单

| 文件 | 处置 |
|---|---|
| `docs/RPA系统完成状态报告.md` | **整体重写**（原 2025-02-25 版仍描述已下线的 flow 组 7 端点并声称「100% 完成 / 可正常启动运行」，与代码严重脱节）。新版按代码事实重列：64 条已注册路由（ai 11 / credentials 7 / executions 19 / **schedules 7** / tasks 8 / workers 12）、24 个服务文件、11 个 API 文件、v1.56 七个 Phase 新增能力与落地明细、flow 组删除说明、**verification-pending 逐条登记表**；整体状态标 `code-complete · verification-pending`（**未**标 SHIPPED） |
| `CHANGELOG.md` | `[Unreleased] — v1.56 候选` 段补：里程碑状态口径声明 + **v1.56 七个 Phase 交付总览**（152/153/154/155/156/157/158）+ CLEANUP-01 专项（假数据/假配置/连接泄漏）+ CLEANUP-02 专项（调度闭环 + 4 权限码 + 本机真跑）+ Changed 段 CLEANUP-03 专项（删除清单与 grep 证据 + 文档真相化 + 路由注册表 746→753） |
| `scripts/route-registry.json` | `node scripts/scan-routes.cjs` 重生成（新增 `/rpa/schedules/*` 7 条；总路由 746 → 753） |
| `xingran-react-frontend/src/constants/permissionCodes.ts` | 新增 `rpaSchedule` 区块 4 码（后端常量 / TS / sys_menu / 路由四向零漂移） |
| `internal/core/db/migrations/registry.go` | 按维护约定补登记 `{257, "rpa_batch_report_and_schedule_perm"}` |
| `internal/core/db/database.go` | migration 257 **两处注册**（PG 分支 + sqlite 分支）+ sqlite AutoMigrate 补注册 `BatchItemReport` / `Schedule` 两模型（并更正「schedules 未注册」的过时注释） |
| `scripts/dbprovision/main.go` | 新部署建表清单补 `&rpamodels.BatchItemReport{}` |
| `pkg/permission/config.go` | 新增 4 个权限码常量（写端点逐动作 strict 单码） |

---

## 4. 闸门验证结果（本机）

| 闸门 | 命令 | 结果 |
|---|---|---|
| 后端编译 | `go build ./...` | ✅ PASS（无输出） |
| 后端全仓测试 | `go test ./...` | ⚠️ 2 个**基线既有失败**（非本 phase 引入，见下），其余全绿 |
| RPA 相关包 | `go test ./internal/api/v1/rpa/ ./internal/services/rpa/ ./internal/scheduler/` | ✅ PASS（24.6s / 0.4s / 21.2s） |
| 前端类型检查 | `npx tsc --noEmit -p tsconfig.app.json` | ✅ PASS（无报错） |
| 路由注册表门禁 | `go test ./pkg/permission/` | ✅ PASS（coverage + freshness + strict 写组三守卫全绿） |
| 路由注册表重生成 | `node scripts/scan-routes.cjs` | ✅ 总路由 753 / 写路由 338 / deferred 151 |
| 调度到点真跑 | `go test ./internal/scheduler/ -run TestRPASchedule_E2E` | ✅ PASS（**真实 Redis + 真实 cron + 真实 sqlite**） |

### 4.1 基线既有失败（已用基线 worktree 复现，非本 phase 引入）

| 失败用例 | 基线复现结论 |
|---|---|
| `internal/services/vdi` `TestAuditService_RecordOperation` | ✅ 基线（HEAD `63dfbc14`，独立 worktree）同样失败：`An error is expected but got nil` |
| `tests/e2e` `TestE2EAllEndpointsLogged` | ✅ 基线同样失败：`operlog_e2e_verify.sh failed: exit status 127`（bash 路径不可用，环境问题） |

> 处置：本 phase **不修**（超出 CLEANUP-01~03 范围，属他人领地/环境缺口），
> 已在 §6 登记为 verification-pending 既有项。

---

## 5. 越界检查（归属协议逐条）

| 协议条款 | 要求 | 实际 | 结论 |
|---|---|---|---|
| 1 | `rpa_router.go`：只在 `SetupRPARouter` 尾部追加 `SetupScheduleRouter`，`:22-52` 不回改 | 尾部追加 1 行调用 + 更新 flow 组**注释**（说明实现已删）；`:22-52`（两处 `NewServiceGroup`）**零改动** | ✅ 合规 |
| 2 | `excel_service.go`：158 独占 | 全面改造（本 phase 独占文件） | ✅ 合规 |
| 3 | `scaling_service.go` / `docker_client.go` / 根 `internal/config`：157 已改，不碰 | **零改动** | ✅ 合规 |
| 4 | `worker_handler.go`：158 独占 | 全面改造（本 phase 独占文件） | ✅ 合规 |
| 5 | 前端 `ExecutionDetailModal.tsx` / `columns.tsx`：153/155 地盘，不碰 | **零改动**（前端只改 `src/constants/permissionCodes.ts`） | ✅ 合规 |
| 硬约束 | 删代码前必须 grep 取证零非测试调用方 | 已执行并留存证据（§3.2） | ✅ 合规 |
| 硬约束 | 删后 `go build ./...` + `go test ./...` 双验证 | 已执行（§4，含基线对比） | ✅ 合规 |
| 硬约束 | migration 双方言幂等 + `database.go` 两处注册 | `HasTable` 守卫 + `runAndRecord`；PG / sqlite 两分支均注册 | ✅ 合规 |
| 硬约束 | 零新增依赖 | 未新增任何依赖（cron/redis/go-redis 均为既有依赖） | ✅ 合规 |
| 硬约束 | 构建产物只落 `out/` | 未产生构建产物（`go build ./...` 不落盘） | ✅ 合规 |
| 硬约束 | 绝不把「代码绿」等同于「已验收」 | 全文按 `code-complete · verification-pending` 口径书写，§6 逐条登记待办 | ✅ 合规 |

**额外写面说明**（本 phase 必需、非协议禁区）：`internal/core/core.go`（调度注册，用户任务明确要求）、
`internal/core/db/database.go` + `migrations/registry.go` + `migration_257`（用户任务明确要求）、
`pkg/permission/config.go`（用户任务明确要求权限码）、`internal/services/rpa/{types,task_service}.go`
（`trigger_type=schedule` 透传所必需，最小改动：仅加 1 个可选字段 + 2 处赋值 + 1 个私有参数）、
`internal/scheduler/rpa_schedule_tasks.go`（调度接线，用户任务明确要求「接入 internal/scheduler」）。

---

## 6. verification-pending 项清单（**未验收，不得当已完成**）

> 依 `.planning/notes/SHIPPED-DEFINITION.md` §2：以下条目未执行 / 证据未落盘 / 前置授权未具备，
> 一律登记为待办，**不预置勾选、不折算为已验收**。

| # | 条目 | 类别 | 阻塞原因 / 解锁路径 |
|---|---|---|---|
| 1 | 批量报告真值在**生产 PG** 上验证（migration 257 建表 + 读写一致性 + 与 `sys_rpa_executions` 的关联） | 验证类 | 本环境无生产 PG 写授权；解锁后按 `notes/261108-uat-writeback-policy.md` 回写 |
| 2 | 扩缩容配置在**生产 PG + 重启后**仍生效（`sys.rpa.autoscale.config` 真实落库 + 重启读取） | 验证类 | 同上 |
| 3 | 定时调度在**真实 Worker 集群**上的到点触发（含 worker 领取 Redis Stream 消息 → 执行推进） | 验证类 | 本环境无真机 VDI / Worker 集群 |
| 4 | 调度触发任务的**端到端执行结果**（本次本机只验到「生成 `trigger_type=schedule` 的执行记录并下发 Redis Stream」，未验 Worker 侧消费与终态） | 验证类 | 同上 |
| 5 | `ops:rpa:schedule:*` 四码的**角色授权实测**（migration 257 播种后，非超管持码角色能否正常 CRUD 调度） | 验证类 | 需真实角色/菜单授权环境 |
| 6 | 调度触发在**多实例**下的重复触发收敛（现有调度引擎无跨实例互斥，参考 `db_backup_tasks.go` 既例） | 能力边界 | 已登记，非本 phase 阻断项 |
| 7 | 既有基线失败 `internal/services/vdi` `TestAuditService_RecordOperation` | 既有 | 非本 phase 引入；由 vdi 域 owner 处置 |
| 8 | 既有基线失败 `tests/e2e` `TestE2EAllEndpointsLogged`（exit 127，bash 路径不可用） | 环境 | 由 e2e 脚本 owner 处置 |
| 9 | 前端 RPA **调度管理页面**（列表/新建/启停 UI） | 未实现 | 本 phase 只交付后端闭环 + 权限码；前端页面未立项（前端零 router 改动归属协议） |

---

## 7. 交付物索引

| 类 | 路径 |
|---|---|
| 新增（服务层） | `internal/services/rpa/schedule_service.go` |
| 新增（模型） | `internal/models/rpa/batch_report.go` |
| 新增（HTTP 层） | `internal/api/v1/rpa/schedule_handler.go` |
| 新增（调度层） | `internal/scheduler/rpa_schedule_tasks.go` |
| 新增（迁移） | `internal/core/db/migrations/migration_257_rpa_batch_report_and_schedule_perm.go` |
| 新增（测试） | `internal/services/rpa/cleanup158_test.go`｜`internal/api/v1/rpa/cleanup158_autoscale_test.go`｜`internal/scheduler/rpa_schedule_e2e_test.go` |
| 改造 | `internal/services/rpa/{excel_service,redis_publish,task_service,types}.go`｜`internal/api/v1/rpa/{worker_handler,rpa_router}.go`｜`internal/core/core.go`｜`internal/core/db/database.go`｜`internal/core/db/migrations/registry.go`｜`pkg/permission/config.go`｜`scripts/dbprovision/main.go`｜`xingran-react-frontend/src/constants/permissionCodes.ts` |
| 删除 | 7 文件 / 2885 行（§3.1） |
| 文档 | `docs/RPA系统完成状态报告.md`（重写）｜`CHANGELOG.md`（v1.56 7 phase + CLEANUP 专项）｜`scripts/route-registry.json`（重生成） |
| 本报告 | `.planning/phases/158-cleanup/158-SUMMARY.md` |

---

## 8. 结论

- **CLEANUP-01**：✅ PASS（假数据与假配置均已改为真读写，本机用例覆盖「无数据」与「有 3 条数据」两种情形）
- **CLEANUP-02**：✅ PASS（`sys_rpa_schedules` 端到端闭环，**本机真跑到点触发**并留下 `trigger_type=schedule` 执行记录）
- **CLEANUP-03**：✅ PASS（死代码 2885 行删除且 grep 取证零非测试调用方、无豁免；RPA 状态报告重写 + CHANGELOG 补全 7 phase）
- **整体里程碑状态**：`code-complete · verification-pending` —— **不标 SHIPPED**；§6 共 9 项验证待办待清偿。
