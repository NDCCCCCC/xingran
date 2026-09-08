# Phase 104: handler层架构收敛（wire契约+样板去重）- Context

**Gathered:** 2026-09-08
**Status:** Ready for planning

<domain>
## Phase Boundary

v1.31 技术债清偿的 handler 层架构收敛——错误响应 wire 契约全仓统一（WIRE-01）+ operations 14 个同构 CRUD handler 样板收敛（HANDLER-01）+ monitor 双日志 handler 五方法去重（HANDLER-02）。无新业务能力，pure 架构重构。

成功标准（ROADMAP SC-1..4）：
1. wire 契约统一：本地 helper 与 pkg 合并为单一权威；operations 14 handler 全量切换；全仓错误响应口径一致
2. operations handler 收敛：14 个同构 CRUD handler 样板收敛；server_room List/Statistics 漂移修复
3. monitor 双 handler 去重：oper_log vs login_log 五方法复制去重
4. 响应契约行为变更附回归测试；七 gate 不倒退

</domain>

<decisions>
## Implementation Decisions

### WIRE-01：错误响应契约统一

- **D-104-1: 方向 = C（重写统一权威）**
  - `pkg/response/handler_helpers.go` 重写 `HandleJSONBinding` + `HandleServiceError` 为统一权威
  - 单一错误载体：`apperrors.AppError`（唯一）；`response.BusinessError` 彻底删除
  - 理由：最佳实践——正确分层（HTTP status 传输语义 / body code 业务细节）；`apperrors` 已原生支持自定义 HTTP status（`NewWithHTTPStatus`/`WrapWithHTTPStatus`）；BusinessError 是 V130 应急造的平行小体系（与 errors 生态割裂），本身就是债

- **D-104-2: BusinessError 迁移与删除**
  - 3 个构造点（`config_restore_task_service.go:76,93,112`）→ `apperrors.WrapWithHTTPStatus(对应 Code, 409, msg)`
  - `pkg/response/business_error.go` 同 commit 删除
  - `config_restore_task_service_97_03_test.go:48,97` 两处类型断言升级为 `*apperrors.AppError` + `HTTPStatus` 检查

- **D-104-3: binding 错误 message 策略**
  - 透传 `err.Error()` 到 message（含字段名/gin validator 消息，如「user_name: 不能为空」）
  - 理由：UX 好；gin validator 输出无服务器内部信息；现状泛泛「参数错误」四字让用户无法定位字段

- **D-104-4: Response.code 语义**
  - `code` = 业务码（1001/1500/1009...）
  - `HTTPStatus` = 语义派生（400/409/500）
  - `Response.data` 始终是业务数据（无 bizCode 征用；成功/失败 schema 统一）
  - 理由：OWASP 对齐；分层正确（网关/监控可见 HTTP status，业务逻辑可见业务码）

- **D-104-5: 统一 helper message 策略**
  - binding 错误：透传 `err.Error()`
  - apperrors 业务错误：业务文案
  - 裸内部错误：`operation + "失败"`（不泄 err.Error() 到前端）；err detail 进服务端日志（request_id 关联）

### HANDLER-01：operations 14 handler 样板收敛

- **D-104-6: 收敛机制 = 四步管线小函数族**
  - `bind → service → err → operlog → success` 五步抽出小函数族
  - 各 handler 文件内部改调小函数，不新建文件
  - 理由：Go 高阶函数是处理「同构流程 + 局部差异」的标准做法；泛型不适合（本相 service 接口差异大：List 参数各异、entity-specific 方法多）

- **D-104-7: 文件结构保留**
  - 14 个 handler 文件保留；swagger 注释留各文件
  - 个性化方法（Statistics/Geocode/SearchOptions）不变动
  - 测试文件：mock service 接口 + httptest 断言（已有）不动，只在 handler 层改小函数调用点

### HANDLER-02：monitor 双 handler 去重

- **D-104-8: 收敛机制 = 泛型 MonitorLogHandler[T]**
  - 5 同构方法（List/GetByID/Delete/BatchDelete/GetByUsername）→ 泛型 struct
  - 路由注册时用第二个模块名区分（OperLogService/LoginLogService）
  - 理由：monitor 域 service 接口高度同构（5 方法签名完全一致，仅 entity 类型不同），泛型**真正适用**

- **D-104-9: Clean 方法各文件保留**
  - `OperLog.Clean`：同步审计事务 + post-clean 验证（200行注释 chicken-and-egg 防御），独立保留
  - `LoginLog.Clean`：简单 async operlog.Record，独立保留
  - `LoginLog.UnlockUser`：Phase 107 TODO-03 占位，本相不动

### WIRE-01 回归守护

- **D-104-10: httptest 契约测试**
  - 断言 `(HTTPStatus, code, message)` 三元组
  - 覆盖 4 类路径：binding 错误 / bindError / ServiceErr / BusinessError → apperrors
  - operations/monitor 既有的 200+ handler 测试升级 code 断言值（1001→400 / 1500→500）

### 继承的锁定决策（不再讨论）

- **v1.31 D-01**: 台账 12 组全做；不留兼容壳
- **v1.31 D-02**: 行为变更附回归测试；七 gate（go build / go test / 后端 coverage ≥78.33 / 前端 45 dirs / lint / type-check / diff coverage）全程不倒退
- **v1.31 D-04**: captcha-background 1=启用语义禁改
- **Phase 102/103**: caller audit 清单形态（constructor 签名变化时 grep 全部调用点核对清单）
- **CLAUDE.md §Operlog Convention**: 收敛后写端点 `operlog.Record` 调用点逐一核对

### Claude's Discretion

- 小函数命名（`BindAndCall` / `CreateBind` / `ListBind` 等）
- 泛型 `MonitorLogHandler[T]` 构造方命名（`NewOperLogHandler` / `NewLoginLogHandler`）
- `config_restore_task_service.go` 三个 BusinessError 构造点对应的 apperrors.ErrorCode 值选择（需与业务语义匹配）
- `operations/handler.go` 新文件位置（`internal/api/v1/operations/` 下）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与目标

- `.planning/notes/260907-audit-fix-tech-debt-findings.md` — F-11（wire 契约分叉）/ F-12（handler 样板复制）台账原文
- `.planning/REQUIREMENTS.md` §WIRE-01 · HANDLER-01 · HANDLER-02 — 3 条 requirement 原文
- `.planning/ROADMAP.md` §Phase 104 — Goal / SC-1..4 / Notes（login_log_handler.go 由 Phase 107 TODO-03 决策解锁）

### 既有契约与约定

- `pkg/response/handler_helpers.go` — pkg helper 现状（含 BusinessError 特判；重写目标文件）
- `internal/api/v1/operations/base_handler.go` — operations 本地 helper 现状（删除目标）
- `pkg/errors/errors.go:78,100` — `NewWithHTTPStatus` / `WrapWithHTTPStatus`（BusinessError 迁移目标）
- `pkg/errors/codes.go` — apperrors.ErrorCode 体系（255 行；BusinessError 迁后删 business_error.go）
- `pkg/response/business_error.go` — 待删除文件（BusinessError 类型定义）
- `.planning/phases/102-mechanical-constants/102-CONTEXT.md` — Phase 102 caller audit 形态先例

### 代码（迁移目标与参照）

- `internal/api/v1/operations/{wall,server_room,floor,door,building,room_device,floor_plan_text,dedicated_line,location_alias,infopoint,workstation,workstation_device,asset,asset_component}_handler.go` — 14 handler 迁移文件（3109 行）
- `internal/api/v1/monitor/oper_log_handler.go` — OperLog handler（待泛型化；Clean 方法保留）
- `internal/api/v1/monitor/login_log_handler.go` — LoginLog handler（待泛型化；Clean + UnlockUser 保留）
- `internal/services/monitor/oper_log_service.go` — OperLogService 接口（泛型 T 上界）
- `internal/services/monitor/login_log_service.go` — LoginLogService 接口（泛型 T 上界）
- `internal/services/config_restore_task_service.go:76,93,112` — 3 个 BusinessError 构造点（待迁移）
- `internal/services/config_restore_task_service_97_03_test.go:48,97` — BusinessError 类型断言测试（待升级）

### 测试

- `internal/api/v1/operations/building_handler_test.go` — 既有 handler 测试（code 断言待升级）
- `internal/api/v1/operations/asset_handler_test.go` — 既有 handler 测试（code 断言待升级）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `apperrors.NewWithHTTPStatus(code, httpStatus, msg)` / `WrapWithHTTPStatus(err, code, httpStatus, msg)` — BusinessError 迁移目标路径，已存在
- `response.HandleIDParam` / `response.HandleGetByID` — pkg 层已有非 wire 辅助函数（不在本次 wire 收敛范围）
- monitor OperLogService / LoginLogService — 5 方法签名完全一致，泛型 T 上界可共用同一接口约束

### Established Patterns

- 行为等价重构 + 回归测试锁（v1.29/30/31 各 phase 一贯模式）
- D-01 不留兼容壳：废弃代码同 commit 删除（BusinessError + operations/base_handler.go）
- 既有 handler 测试断言 HTTP status 层面（400/500）+ `"code":0`，未深度锁 wire code 值——本相升级后语义更严格

### Integration Points

- 6 个包（operations/duty/monitor/network/scheduler/workorder）统一调用同一 helper，全仓 wire 口径收敛
- `cmd/main.go` 无 operations handler 构造方（operations 用 router 层闭包注入 core），无需改 wiring
- `login_log_handler.go:188-194` UnlockUser（Phase 107 TODO-03）本相去重不动，保持原样

</code_context>

<specifics>
## Specific Ideas

- `operations/base_handler.go` 本地 `handleJSONBinding` / `handleServiceError` → 删除（helper 合并后）
- server_room `Statistics:37` 手写 `http.StatusInternalServerError, err.Error()`（泄 detail）和 `List:101-103` 手写 `InternalServerErrorWithMsg("查询失败")`（丢 detail）两处漂移随 handler 收敛一并修复
- BusinessError 迁 `apperrors.WrapWithHTTPStatus` 时需选对应 Code：config_restore_task_service 的 3 个场景（设备不匹配→400 / 活跃任务冲突→409 / 恢复失败→500）需匹配 apperrors 既有 Code 或新增

</specifics>

<deferred>
## Deferred Ideas

- **UnlockUser 实现**（login_log_handler.go:188）：Phase 107 TODO-03 占位，本相去重不动它——等 Phase 107 决策实现或删除
- **泛型 handler 扩展**：若未来更多日志类 handler 加入（sys_log 等），`MonitorLogHandler[T]` 可复用——这是泛型架构的副产品，不在本相 scope
- **前端按业务码分支**：当前 api.ts 拦截器只依赖 `code === 0`；若未来需按 1009（记录已存在）等码做类型化分支（表单字段标红等），契约已就绪（code=业务码），前端逻辑需单独迭代

</deferred>

---

*Phase: 104-handler-wire*
*Context gathered: 2026-09-08*
