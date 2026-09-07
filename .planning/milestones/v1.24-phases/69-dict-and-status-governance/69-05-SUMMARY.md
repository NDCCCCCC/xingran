---
phase: 69-dict-and-status-governance
plan: 05
subsystem: backend-status-governance
tags: [DICT-01, status-constants, literal-ratchet, CASE-WHEN, models]
requires:
  - phase: 69-04
    provides: "批 3 状态常量治理与 85 常量锁值基线"
provides:
  - "批 4 17 个基线文件、46 个守护命中全部迁移，白名单收缩为 F 簇 1 条"
  - "B 簇 WorkOrder/Execution/Discovery/CASE WHEN 状态机改用对应 models 常量"
  - "ServerStatus 与 NotificationConfigStatus 状态真相源及锁值覆盖"
  - "工单、值班、监控、操作日志、通知、任务与通知配置散点状态常量化"
affects: [69-08, 内部状态常量治理, 后续服务端状态分支]
tech-stack:
  added: []
  patterns:
    - "CASE WHEN 聚合使用 fmt.Sprintf + int(models.XxxStatus)"
    - "普通 raw SQL 条件优先参数化，模型字段保持现有 int/typed 边界"
    - "status_constants_test.go AST 双向锁值 + watched family ratchet"
key-files:
  created: []
  modified:
    - internal/models/monitor.go
    - internal/models/notification_config.go
    - internal/models/status_constants_test.go
    - internal/services/config_execution_service.go
    - internal/services/device_discovery_service.go
    - internal/services/command_dispatch_service.go
    - internal/services/duty_pool_service.go
    - internal/services/workorder/base.go
    - internal/services/monitor/server_service.go
    - internal/services/asset/fix_suggestion_monitor.go
    - internal/services/operations/geocoding_service.go
    - internal/api/v1/system/notice_handler.go
    - internal/api/v1/monitor/oper_log_handler.go
    - scripts/check-status-literals.sh
key-decisions:
  - "device_discovery_service.go 复用 models.DiscoveryStatus 全族，不套 ExecutionStatus，保持发现任务 0-4 状态机语义"
  - "config_execution_service.go 与 command_dispatch_service.go 的统计 SQL 使用 Sprintf 注入 models.ExecutionStatus 值，AS 别名保持不变"
  - "geocoding_service.go 百度响应码保留原字面量并加 F 簇注释，属于明确外部契约而非内部状态值"
patterns-established:
  - "DICT-01 批 4：17 文件、46 个基线命中清零，终态白名单仅 external contract 1 条"
requirements-completed: [DICT-01]
metrics:
  duration: 23min
  completed: 2026-08-19
---

# Phase 69 Plan 05: DICT-01 批 4 状态语义单一真相源 Summary

**DICT-01 最后一批状态治理完成：17 个白名单文件、46 个守护命中全部迁移，Execution/Discovery/WorkOrder 等状态机与散点配置改用 models 常量，守护仅保留百度 API 外部契约 1 条。**

## Performance

- **Duration:** 23 min
- **Started:** 2026-08-19T09:36:37Z
- **Completed:** 2026-08-19T09:59:20Z
- **Tasks:** 1
- **Files modified:** 22
- **Task commit:** `bc00d9c` — `refactor(69-05): replace status literals and finalize guard`

## Accomplishments

- 清零 `workorder`、`duty_pool`、`device_discovery`、`config_execution`、`command_dispatch`、监控、资产及 API 散点命中，状态机与启停/成败语义分别引用实体常量。
- 将 `server_service.go` 的 `ServerInfo.Status` 建立为 `ServerStatus` 真相源，并为邮件/API 通知配置建立共享 `NotificationConfigStatus`。
- 扩展 AST 锁值覆盖：既有 `DiscoveryStatus` 0-4 五个常量也纳入 `watchedStatusPrefixes`，加上本批两个新二元状态族，锁值数量从 85 增至 94。
- 将 `check-status-literals.sh` 收缩为仅 `geocoding_service.go=1`，F 簇代码行为保持不变。

## Commit 对照表

| Task | 内容 | Commit | 文件数 | 替换处数 | 说明 |
|---|---|---|---:|---:|---|
| T1 | 批 4 全模块与散点迁移、F 簇收口、白名单终态 | `bc00d9c` | 22 | 46 个基线命中 + reconciliation 模板 4 处展示文案 | `refactor(69-05): replace status literals and finalize guard` |

**守护脚本基线前后快照：**
- 批 4 前：`17 文件 / 46 命中`。
- 批 4 后：`1 文件 / 1 命中`。
- 终态唯一条目：`internal/services/operations/geocoding_service.go=1`。

## 语义簇映射台账

| 文件 | 簇 | 所用常量 / 决策 |
|---|---|---|
| `internal/services/workorder/base.go` | B 工单状态机 | `WorkOrderStatusPending/Processing/Completed/Closed`；4 个 CASE WHEN |
| `internal/services/workorder/reconciliation_template.go` | A 启停 | 模板展示文案经 `UserStatusDisabled/Enabled`、`AssetStatusStopped` 派生 |
| `internal/services/workorder/assignment.go` | A 值班状态 | `DutyStatusNormal`，raw SQL 使用 `?` 参数 |
| `internal/services/duty_pool_service.go` | A 值班池启停 | `DutyPoolStatusEnabled/Disabled`；2 个 CASE WHEN |
| `internal/services/device_discovery_service.go` | B 发现任务状态机 | `DiscoveryStatusPending/Running/Success/Failed`；4 个 CASE WHEN |
| `internal/services/config_execution_service.go` | B 配置执行状态机 | `ExecutionStatusPending/Running/Success/Failed`；4 个 CASE WHEN |
| `internal/services/command_dispatch_service.go` | B 命令执行状态机 | `ExecutionStatusPending/Running/Success/Failed`；4 个 CASE WHEN |
| `internal/services/monitor/server_service.go` | A 服务器健康 | `ServerStatusNormal`；补齐 `ServerInfo.Status` typed 真相源 |
| `internal/services/asset/fix_suggestion_monitor.go` | C 成败 | `OperLogStatusSuccess` |
| `internal/api/v1/monitor/oper_log_handler.go` | C 成败 | `OperLogStatusSuccess` |
| `internal/api/v1/system/notice_handler.go` | A 定时任务 | 两处 `JobStatusNormal` |
| `internal/api/v1/scheduler/job_handler.go` | A 定时任务 | `JobStatusPause` |
| `internal/services/oper_log_service.go` | C 成败 | `OperLogStatusSuccess/Failure`，保留 `models.OperLog.Status int` 边界 |
| `internal/services/api_endpoint_service.go` | A 菜单启停 | `MenuStatusNormal`，raw SQL 参数化 |
| `internal/services/api_sender_service.go` | A 通知配置启停 | `NotificationConfigStatusNormal` |
| `internal/services/email_sender_service.go` | A 通知配置启停 | `NotificationConfigStatusNormal` |
| `internal/services/notification_config_service.go` | A 通知配置启停 | `NotificationConfigStatusNormal`，默认邮件查询参数化 |

## F 簇豁免

- `internal/services/operations/geocoding_service.go:332` 保留 `baiduResp.Status != 0`，其语义来自百度地图 API 文档而非 `internal/models`。
- 同处增加唯一注释：`F 簇：百度地图 API 返回码契约，不迁移到 models 常量（见 scripts/check-status-literals.sh 白名单）`。
- 白名单最终仍为 `geocoding_service.go=1`；没有新增网络、认证、文件访问或 schema 信任边界。

## 验证记录

- `go build ./...`（主工作树） — PASS。
- `go test ./internal/models/ -run TestStatusConstants -v` — PASS；`TestStatusConstantsStability` 与 14 个关键族子测试全绿。
- `go test ./internal/services/workorder/ ./internal/services/ ./internal/services/monitor/ ./internal/services/asset/ ./internal/api/v1/system/ ./internal/api/v1/monitor/ ./internal/api/v1/scheduler/` — PASS。
- `bash scripts/check-status-literals.sh` — PASS。
- `bash scripts/check-status-literals.sh --baseline` — 恰 1 行。
- 验收抽样：`config_execution_service.go`、`command_dispatch_service.go` 的 `AS pending/running/success/failed` 与 `workorder/base.go` 的四个 AS 别名保持不变。
- B 簇源码差异无 `Enabled/Stopped` 等误套常量。
- 干净 worktree（`bc00d9c`）执行 `go build ./...` — PASS。
- 干净 worktree 执行 `go test ./...` — 所有普通包全绿；既有 `tests/integration` 中 3 个登录加密测试失败：`TestPublicKeyEndpoint` 2 处（404/响应解码）、`TestResponseHeaders` 1 处（响应 Content-Type 非 JSON）、`TestRequestMethodValidation` 1 处（GET 404）。本 plan 未修改认证路由或该测试文件，按范围约束不扩展修复。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - 缺失关键真相源] 补齐 ServerStatus 与 NotificationConfigStatus**
- **Found during:** T1 修改服务器监控状态及邮件/API 配置状态时
- **Issue:** `ServerInfo.Status` 与 `EmailConfig/APINotificationConfig.Status` 仅有 int 字段和裸 0/1，没有实体常量族
- **Fix:** 在 `models` 新增 `ServerStatusNormal/Abnormal` 与 `NotificationConfigStatusNormal/Stopped`，消费方使用常量并注册 AST 锁值
- **Files modified:** `internal/models/monitor.go`, `internal/models/notification_config.go`, 状态消费文件, `internal/models/status_constants_test.go`
- **Verification:** `TestStatusConstantsStability`、受影响包测试、干净 worktree `go build ./...` 全绿
- **Committed in:** `bc00d9c`

**2. [Rule 2 - 锁值覆盖缺口] 补登记 DiscoveryStatus 全族**
- **Found during:** 核对 B 簇 0-4 状态机与锁值测试时
- **Issue:** `DiscoveryStatus` 已有完整 0-4 族，但未进入 watched/expected 锁值表
- **Fix:** 将 `DiscoveryStatusPending/Running/Success/Failed/Cancelled` 同时加入前缀与期望 map
- **Files modified:** `internal/models/status_constants_test.go`
- **Verification:** `TestStatusConstantsStability` PASS，锁值 85→94
- **Committed in:** `bc00d9c`

**3. [Rule 3 - 清单口径] 纳入基线中的 7 个散点文件**
- **Found during:** T1 action 1 执行 `bash scripts/check-status-literals.sh --baseline`
- **Issue:** 主文件清单仅列 10 个服务/handler，基线实际还有 `api_endpoint_service.go`、`api_sender_service.go`、`email_sender_service.go`、`notification_config_service.go`、`oper_log_service.go`、`scheduler/job_handler.go`、`workorder/assignment.go`
- **Fix:** 按 plan 的“散点全部纳入”授权迁移这 7 处，并以菜单、通知配置、操作日志、任务、排班状态映射
- **Files modified:** 上述 7 个散点文件
- **Verification:** 基线 17 文件/46 命中降为 1 文件/1 命中，守护退出码 0
- **Committed in:** `bc00d9c`

---

**Total deviations:** 3 auto-fixed（2 个缺失关键功能 + 1 个阻塞/清单修正）。**  
**Impact on plan:** 未改变业务逻辑；补齐状态真相源并保证 ratchet 真正反映全部基线，没有扩大到无关模块。

## Issues Encountered

- 干净 worktree 全仓测试仍受 3 个既有 `tests/integration` 登录加密用例失败影响；本 plan 的 22 个修改文件与认证/测试目录无交集，因此作为 `Deferred Issues` 记录，不在本批越界修改。
- `commitlint` 首次因计划标题超过 100 字符拒绝提交；未改代码，将任务提交压缩为等义短标题后通过。
- 隔离测试 worktree 在验证完成后已使用 `git worktree remove --force` 删除。

## Known Stubs

- `internal/services/device_discovery_service.go:662`：`GetDiscoveryResults` 仍以 TODO 声明未来从临时表/缓存获取结果并返回空切片；这是既有独立能力缺口，不影响本批状态常量化。
- `internal/services/workorder/base.go:270`：预期解决时间仍留有 `placeholder for future implementation` 注释；既有能力未在本批扩展。

## Deferred Issues

| Item | Evidence | Disposition |
|---|---|---|
| 登录加密集成测试 3 项失败 | `tests/integration/login_encryption_test.go`：public-key/GET 返回 404，响应头为 `text/plain` | 记录为 Phase 68 认证路由/中间件范围；不在 DICT-01 常量治理中修复 |
| 发现结果临时表/缓存 | `device_discovery_service.go:653-663` 返回空列表 | 独立功能补全，未来计划处理 |

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- DICT-01 后端替换面已完成，守护白名单达到仅 F 簇 1 条的终态。
- Phase 69 仅余 Plan 08（DICT-04 文档指针化），可继续执行；登录加密集成测试遗留项需由其对应认证范围计划/Phase 68 合并后复验。
- 主工作树 settings/default_theme 与 Phase 70 草图/未提交改动未被本 plan 修改或暂存。

---

*Phase: 69-dict-and-status-governance*  
*Completed: 2026-08-19*

## Self-Check: PASSED

- FOUND: `.planning/phases/69-dict-and-status-governance/69-05-SUMMARY.md`
- FOUND: task commit `bc00d9c` in `git log --all`
- FOUND: 14 个关键实现/锁值/守护文件全部存在
- FOUND: 临时干净 worktree 已在全仓验证后删除

