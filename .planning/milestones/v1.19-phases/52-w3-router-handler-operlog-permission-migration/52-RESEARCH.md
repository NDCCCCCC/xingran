# Phase 52: W3 — Router/Handler/Operlog/Permission/Migration - Research

**Researched:** 2026-07-07
**Domain:** Go/Gin HTTP wiring + GORM migration + operlog audit + RBAC permission gating + menu seed
**Confidence:** HIGH (all 3 novel integration points directly verified against source; all CONTEXT-cited analogs re-read with line numbers; 2 critical corrections to CONTEXT surfaced)

---

## Summary

Phase 52 把 Phase 51 已稳定的 `PortWriteService`（6 方法 + 28 mock 测试绿色）暴露成 HTTP：6 个写端点 + operlog 审计 + `sys_port_write_audit` 未脱敏真相源表 + `network:port:write` 权限隔离 + 菜单 seed。本 phase 不改 Phase 51 service 任何签名（零侵入）。

研究直接核对了所有 CONTEXT 引用的源文件（operlog.go / regression_test.go / oper_log_service.go / models/log.go / models/base.go / models/menu.go / models/device_port_status.go / network_router.go / port_router.go / permission/config.go / middleware/permission.go / migration_195 / migration_200 / migration_201 / database.go / 053_fix_menu_paths_unified.sql），结果：**绝大多数 CONTEXT 引用准确**，但发现 **2 个会直接影响 planner 的关键偏差** 和 **1 个 D-13 机制需重新评估的执行风险**：

1. **migration 注册机制偏差（D-14）**：CONTEXT 说"注册到 database.go AutoMigrate + migration list"——实际上经 260704-ne5 重构后，**所有 `MigrateNNN` 函数都不再启动期自动调用**（database.go:296 明确注释）。启动期只调用 `Migrate175` + `Migrate176`（MV 重建）。Phase 52 必须二选一：(a) 把 `&models.PortWriteAudit{}` 加入 `d.DB.Migrator().AutoMigrate(...)` 列表让 GORM 建表，migration_202 只做菜单 seed + 索引 + helper 授权；(b) 在 `AutoMigrate()` postgres 分支显式添加 `migrations.Migrate202PortWriteAudit(d.DB)` 调用（与 175/176 同位置）。
2. **RequirePermissions 签名偏差（D-09）**：CONTEXT 与 CLAUDE.md 都写成 `RequirePermissions([]string)`——实际签名是 `RequirePermissions(permissions []string, core *core.Core)`（middleware/permission.go:200），**两个参数**。planner 写代码必须用 2-arg 形式。
3. **D-13 WithOperID 执行风险**：`operLogService.RecordAsync` 是真正 fire-and-forget（`go func() { db.Create(operLog) }()`），且在函数内部构造 `OperLog` struct 时**从不设置 ID 字段**（oper_log_service.go:46-64）。`BaseTimeLine.BeforeCreate` 钩子虽然保留预设 ID，但 RecordAsync 路径根本无法接收预设 ID——要么改 `Recorder` 接口签名（破坏 regression_test.go:294 的 mockRecorder），要么改 RecordAsync 实现读预设 operID。**D-13 推荐的"兜底"路径（先写 audit 拿 audit_ids，再用 `WithOperParam({"audit_ids":[...]})` 把 audit ids 嵌 operlog）风险显著低于 WithOperID 改造**。详见 §1.3。

**Primary recommendation:** 采用 2-wave 切分（与 ROADMAP 一致）：Wave 1 = NetworkPortWrite 常量 + port_write_router.go + port_write_handler.go + operlog.Record 调用 + cache_keys.go；Wave 2 = PortWriteAudit model + database.go AutoMigrate 注册 + migration_202_port_write_audit.go（菜单 seed + helper + 复合索引）+ menu_grant_helpers.go。D-13 audit↔operlog 关联**走兜底路径**（audit_ids 嵌 operlog oper_param），不改 operlog 包接口，保 regression_test.go 绿色。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions (D-01 .. D-16，逐字摘自 52-CONTEXT.md)

- **D-01**: `sys_port_write_audit` 12 列定义（id / device_id / port_id / action / before_value JSONB / after_value JSONB / command_sent TEXT / device_response TEXT / status varchar(16) / failure_reason TEXT / operator varchar(50) / oper_log_id UUID / created_at）；索引 `(device_id, port_id, created_at)` 复合 + `(created_at)` 单列；表名锁定 `sys_port_write_audit`（单数）。
- **D-02**: before_value 由 handler 预 SELECT 快照（不改 Phase 51 service）；SQL = `SELECT admin_status, dot1x_enabled, description, device_id, interface_name FROM sys_device_port_status WHERE id = ?`；端口行不存在时 before_value = `{}`。
- **D-03**: after_value 由 handler 同步填目标态（shutdown→`{"admin_status":"down"}` 等）；NoOp/skipped 路径 after_value = before_value。
- **D-04**: audit 写入归属 = handler 同步，`response.Success` 之前；NoOp/skipped 路径也写 audit（status=skipped, command_sent="", device_response="无需操作"）；失败路径也写 audit（status=failed, failure_reason=err.Error()）。
- **D-05**: batch = N 条 audit（每端口 1 条）+ 1 条 operlog 汇总（OperTypeBatch=16，oper_param 含 action/batch_size/succeeded_count/failed_count/skipped_count/device_id）。
- **D-06**: "端口配置" 菜单 menu_type='F' 按钮权限，path='write'，perms='network:port:write'；F 型不生成前端路由。
- **D-07**: 父菜单实际名 = "端口状态"（非 ROADMAP 写的 "端口管理"），实存证据 archive/053_fix_menu_paths_unified.sql:184-186；operlog module 字符串仍用 `端口管理`。
- **D-08**: 新建 `menu_grant_helpers.go::GrantNewMenuToRolesHavingParent(db, parentMenuName, newMenuID)`，幂等 SQL `INSERT INTO sys_role_menu SELECT rm.role_id, '<newMenuID>'::uuid FROM sys_role_menu rm JOIN sys_menu m ON rm.menu_id = m.id WHERE m.menu_name = '<parentMenuName>' ON CONFLICT DO NOTHING`。
- **D-09**: 子组 `/network/ports/write/*` + 组级鉴权 + kebab 命名（shutdown / undo-shutdown / description / dot1x-enable / dot1x-disable / batch）。
- **D-10**: `cache_keys.go` 仅定义常量 `CacheKeyPortWriteResult = "port:write:result:%s"` + `CacheKeyPortWriteBatch = "port:write:batch:%s"`，service/handler 都不写入。
- **D-11**: batch 端点同步阻塞（复用 Phase 51 BatchWritePorts 语义，detached 30min ctx 已在 Phase 51 落地）。
- **D-12**: BATCH-05 进度反馈推到 v1.19.x。
- **D-13**: audit↔operlog 关联 = `WithOperID(id string) RecordOption`；handler 预生成 operID，operlog.Record + audit 行用同一 operID；**`oper_log_id` 列可空**（防 operlog async 写入失败时 audit 仍能落库）。**[RESEARCH 注：此机制存在执行风险，详见 §1.3]**
- **D-14**: 迁移编号 = `migration_202_port_write_audit.go`；最新已存在 migration_201（Phase 48）。**[RESEARCH 注：注册机制偏差，详见 §1.2]**
- **D-15**: OperType 映射：shutdown/undo_shutdown/dot1x_enable/dot1x_disable → OperTypeStatus(=10)；description → OperTypeUpdate(=2)；batch → OperTypeBatch(=16)；operlog module = `端口管理`。
- **D-16**: `pkg/permission/config.go` 新增 `NetworkPortWrite PermissionCode = "network:port:write"`（在 NetworkPortQuery 旁）。

### Claude's Discretion（CONTEXT 已授权的裁量区）

- PortWriteAudit model GORM tag 细节（type/jsonb/uuid/index）
- handler 取 operator 沿用 `utils.GetUsername(c)`
- 单端口 handler 怎么拿 device_id/interface_name 拼 before_value：D-02 预 SELECT 已含
- audit 表不加 oper_log_id 之外的其他 FK
- migration_202 用 GORM AutoMigrate 还是手写 SQL：倾向手写 CREATE TABLE IF NOT EXISTS（migration_201 风格）
- 批量 audit 事务粒度：每条独立 INSERT（不用大事务，失败 applogger 告警不阻塞响应）
- 单端口 handler 的 request body struct：`PortWriteRequest{PortID string; Description string; Reason string}`（description 仅 description 方法用；Reason 是 UI-02 操作原因，后端仅记录不校验）

### Deferred Ideas (OUT OF SCOPE)

- BATCH-05 批量进度反馈（WebSocket/SSE）— v1.19.x
- `sys_port_write_audit` 详情查看 UI — v1.19.x+
- audit 表 TTL / 归档策略 — v1.19.x+
- 跨设备批量（batch 内多 device）— Phase 51 D-17 已锁 ErrMixedDevices 拒绝
- 写命令前设备可达性预检（FUTURE-07）— v1.19.x+
- operlog `WithOperID` 是否同步进入 RecordBackground（cron 路径）— 本 phase 仅 Record（HTTP 路径）需要
</user_constraints>

<phase_requirements>
## Phase Requirements (来自 REQUIREMENTS.md，本 phase 直接负责 19 项)

| ID | Description | Research Support |
|----|-------------|------------------|
| AUDIT-01 | success path 末尾、response.Success 之前调 operlog.Record | §2.4 fix_suggestion_handler.go:132/177/214/276 实战参考；module 常量定义模式 |
| AUDIT-02 | oper_param 含 device_id/port_id/action/description/operator/result_status | §1.1 WithOperParam + §3.4 JSON 结构；handler 拼 JSON 串传 WithOperParam |
| AUDIT-03 | sys_port_write_audit 表 before/after/command_sent/device_response/status/failure_reason | §1.2 迁移机制 + §2.5 migration_201 手写 SQL 风格 + §3.5 sentinel→HTTP 翻译 |
| PERM-01 | pkg/permission/config.go 新增 NetworkPortWrite 常量 | §2.1 config.go:186 旁加；不加 GetRoutePermissions() map（详见 §3.1） |
| PERM-02 | 6 端点加 RequirePermissions(["network:port:write"]) | §2.2 + §1.4 实际 2-arg 签名 RequirePermissions(perm []string, core *core.Core) |
| PERM-03 | migration seed "端口配置" 子菜单 + GrantNewMenuToRolesHavingParent 精准授权 | §2.3 migration_195 参考 + D-08 helper；父菜单名 "端口状态"（D-07） |
| INFRA-01 | sys_port_write_audit 表 + (device_id, port_id, created_at) 索引 | §1.2 迁移机制 + §3.6 GORM tag 注意事项 |
| INFRA-02 | 注册 /network/ports/write 路由 | §2.2 network_router.go:213 后插 SetupPortWriteRouter(ports, core) |
| INFRA-03 | cache_keys.go 定义 CacheKeyPortWriteResult / CacheKeyPortWriteBatch | §3.7（仅定义不写入，D-10 锁定） |
| CONV-01 | shutdown/undo_shutdown → OperTypeStatus(10) | operlog.go:52 常量已存在直接用 |
| CONV-02 | description → OperTypeUpdate(2) | operlog.go:38 常量已存在直接用 |
| CONV-03 | dot1x enable/disable → OperTypeStatus(10) | operlog.go:52 常量已存在直接用 |
| CONV-04 | batch → OperTypeBatch(16) | operlog.go:58 常量已存在直接用 |
| PORT-01 | shutdown 端点 | §2.2 路由 + §2.6 handler 调 svc.Shutdown |
| PORT-02 | undo-shutdown 端点 | §2.2 路由 + §2.6 handler 调 svc.UndoShutdown |
| PORT-03 | description 端点 | §2.2 路由 + §2.6 handler 调 svc.SetDescription（带 desc 参数） |
| PORT-04 | dot1x-enable 端点 | §2.2 路由 + §2.6 handler 调 svc.EnableDot1x |
| PORT-05 | dot1x-disable 端点 | §2.2 路由 + §2.6 handler 调 svc.DisableDot1x |
| BATCH-01 | batch 端点（同设备多端口同操作） | §2.2 路由 + §2.6 handler 调 svc.BatchWritePorts + §3.4 batch audit N 条 + 1 operlog |
</phase_requirements>

---

## 1. 执行风险点验证（3 个真正新的集成点）

### 1.1 D-13 / D-08 第 1 部分：`operlog.WithOperID` 的实际可行性 ⚠️ 关键

**验证结论：D-13 推荐的 `WithOperID` 机制对当前 operlog 包是 *非侵入式 option*（不破坏 regression_test.go），但对 `operLogService.RecordAsync` 实现是 *侵入式改造*（要么改 Recorder 接口签名破坏 mock，要么改实现层偷读预设 ID）。**

**[VERIFIED: internal/utils/operlog/operlog.go:215]** `func Record(c *gin.Context, operLogSvc Recorder, db *gorm.DB, module string, operType int, opts ...RecordOption)` — 5 fixed params + variadic RecordOption，签名稳定。

**[VERIFIED: internal/utils/operlog/operlog.go:182-184]** `type RecordOption func(*recordConfig)` — 现有 4 个 option：`WithOperParam` / `WithStatus` / `WithErrorMsg` / `WithJsonResult`（注：CLAUDE.md/CONTEXT 提到的 `WithJsonResult` 实际是 `recordConfig.jsonResult *string` 字段，但 *没有导出的 WithJsonResult option 函数* —— 仅 WithOperParam/WithStatus/WithErrorMsg 三个）。

**[VERIFIED: internal/utils/operlog/operlog.go:174-180]** `recordConfig` struct 有 5 个字段（operParam / jsonResult / errorMsg / status / costTime）—— **加 `operID string` 字段 + `WithOperID(id string) RecordOption` 是纯加法**，不破坏回归锁。

**[VERIFIED: internal/utils/operlog/regression_test.go:113-123]** `TestOperTypeCountEquals25` 锁的是 *OperType 常量数 = 25*，加 RecordOption 不动常量 → 不破坏。`TestRecordSignatureStable` 锁的是 *Record 函数 5 fixed + variadic* —— 加 option 不动函数签名 → 不破坏。`TestFilterSensitiveParamsKeywordsStable` 锁的是 sensitiveKeys 关键词集 —— 与 option 无关。

**[VERIFIED: internal/services/oper_log_service.go:42-73]** **执行风险根源**：`RecordAsync` 在函数内部 `operLog := &models.OperLog{...}`（line 46-64）构造 struct，**ID 字段未赋值**，然后 `go func() { db.Create(operLog) }()`。要让预设 operID 真正落到 DB 行，必须二选一：

- **路径 A（推荐 planner 拒绝）**：改 `Recorder` 接口签名加 operID 参数 → 破坏 regression_test.go:294 `excludedMockRecorder.RecordAsync` 的 13 参数实现 → planner 必须同步改 mock + regression_test.go 的 excludedMockRecorder → 触碰 Phase 34 锁定结构。
- **路径 B（可行但侵入 services 层）**：在 operlog 包内加一个 *同步直写* 的 helper（如 `RecordSync(c, svc, db, module, operType, opts...) (operID string, err error)`），handler 预生成 operID 后调 RecordSync 拿到 operID，再写 audit。但这违背 `Record` 当前 fire-and-forget 语义，新增了同步路径。
- **路径 C（D-13 兜底，强烈推荐）**：**不改 operlog 包**。handler 先 INSERT audit 行（拿 audit_ids），再调 `operlog.Record(c, svc, db, "端口管理", operType, operlog.WithOperParam(auditJSON))`，其中 `auditJSON` 含 `{"audit_ids":["...","..."], "device_id":"...", "port_id":"...", ...}`。UI-04 "查看审计日志" 跳转靠 *operlog → audit* 反向（前端打开 sys_oper_log 详情时，从 oper_param 解析 audit_ids 调 audit 接口）。`audit.oper_log_id` 列保留为 nullable，Phase 53 前端联动时若需要 audit→operlog 正向跳转，再用一个后台 reconciliation 任务回填（扫描最近 N 分钟 audit + operlog 按 (operator, module, time-window) 匹配）。

**Planner 强烈建议**：采用路径 C（兜底）。理由：
1. 零侵入 operlog 包 + 零侵入 Recorder 接口 + regression_test.go 保持绿色。
2. UI-04 是 Phase 53 前端任务，本 phase 后端只需保证 audit 行 + operlog 行都写了；关联精度由 Phase 53 决定（用 oper_param.audit_ids 反向跳或预留 reconciliation 都可）。
3. D-13 锁定的"`oper_log_id` 列可空"已为路径 C 留好余地。

**如果 planner 仍想走 D-13 主路径（WithOperID）**：必须在 §1.3 的子任务中显式包含 (a) operlog 包加 `operID` 字段 + `WithOperID` option；(b) 改 `operLogService.RecordAsync` 实现从某处（如 context.Context value、或新增第 14 个参数）读预设 operID；(c) 同步更新 regression_test.go 的 `excludedMockRecorder.RecordAsync` 13 参数签名；(d) 加新测试覆盖 WithOperID 路径。**任务量 ≈ 兜底路径的 3-4 倍，且触及 Phase 34 锁定结构，风险显著**。

### 1.2 D-14：migration_202 在 database.go 的实际注册路径 ⚠️ 关键偏差

**验证结论：CONTEXT D-14 写的"注册到 database.go AutoMigrate + migration list"与现实不符。当前架构无 migration list，且大部分 MigrateNNN 函数启动期不调用。**

**[VERIFIED: internal/core/db/database.go:296]** 明确注释：**"所有 migrations.MigrateNNN 函数定义仍保留在 internal/core/db/migrations/, 仅不再启动期调用。如需重放某次迁移，手动跑对应函数(d.DB) 即可。"**（260704-ne5 重构后的状态）

**[VERIFIED: internal/core/db/database.go:298-421]** `AutoMigrate()` 方法实际启动期调用：
1. `cleanupOldConstraints()` + `dropDependentMaterializedViews()`（postgres only）
2. `d.DB.Migrator().AutoMigrate(&models.X{}, ...)` — **GORM 模型驱动的建表/加列**（line 308-391，~60 个 model）
3. `auditConstraintNaming()`
4. **仅 2 个显式 migration 函数调用**：`Migrate175ReconciliationPhysicalLink` + `Migrate176ReconciliationPhysicalMV`（line 412-417，仅 MV 重建）

**[VERIFIED: glob internal/core/db/migrations/*.go]** 最新 migration 编号 = **201**（migration_201_phase48_component_columns.go），CONTEXT D-14 准确。Phase 52 用 **migration_202** 编号无冲突。

**Planner 必须二选一**（推荐路径 A）：

- **路径 A（推荐）：PortWriteAudit 加入 GORM AutoMigrate 列表 + migration_202 仅做菜单/索引/helper**
  - 在 database.go:308-391 的 `d.DB.Migrator().AutoMigrate(...)` 调用中加一行 `&models.PortWriteAudit{}`
  - GORM 按 model tag 自动建表 + 列；JSONB / UUID 类型由 model tag 控制
  - migration_202_port_write_audit.go 仅包含：菜单 seed（count-then-insert）+ GrantNewMenuToRolesHavingParent 调用 + 复合索引 SQL（CREATE INDEX IF NOT EXISTS，因 GORM composite-index tag 命名不可控）
  - migration_202 也需在 database.go AutoMigrate() postgres 分支显式调用（仿 175/176），否则菜单 seed 不会跑
  - **优点**：JSONB 列类型由 GORM 自动处理（`gorm:"type:jsonb"`）；表存在性由 GORM 幂等保证
  - **缺点**：planner 必须保证 model tag `gorm:"->;-:migration"` 不被错误使用（详见 §3.6）

- **路径 B：纯手写 SQL（migration_201 风格）**
  - 不动 GORM AutoMigrate 列表，PortWriteAudit model 加 `gorm:"-:migration"` tag
  - migration_202 用 `CREATE TABLE IF NOT EXISTS sys_port_write_audit (...)` + `CREATE INDEX IF NOT EXISTS ...` + 菜单 seed + helper
  - migration_202 必须在 database.go AutoMigrate() postgres 分支显式调用（仿 175/176）
  - **优点**：精确控制 SQL（JSONB 类型 + 复合索引命名 + 字段顺序）
  - **缺点**：SQLite 分支需另写（migration_201:45-51 的 `if !isPostgreSQL(db)` 分支模式），增加任务量

**子任务清单（路径 A）**：
1. `internal/models/port_write_audit.go` 新建 PortWriteAudit model（参考 §3.6 GORM tag 注意事项）
2. `internal/core/db/database.go` AutoMigrate 列表加 `&models.PortWriteAudit{}`
3. `internal/core/db/database.go` AutoMigrate() postgres 分支加 `migrations.Migrate202PortWriteAudit(d.DB)` 调用（非阻断，参考 line 412-417 错误处理）
4. `internal/core/db/migrations/migration_202_port_write_audit.go` 实现：复合索引 SQL + 菜单 seed + GrantNewMenuToRolesHavingParent 调用
5. `internal/core/db/migrations/menu_grant_helpers.go` 实现 helper

### 1.3 gorm:"-:migration" tag 与手写 SQL 的交互

**[CITED: project memory gorm-migration-tag-does-not-block-insert + gorm-automigrate-blocked-by-matview]** 两条 memory 已记录的坑：

1. **`gorm:"-:migration"` 不阻止 INSERT**：只设 `IgnoreMigration=true`，但 `Creatable` 仍 true → `db.Create()` 把列塞进 INSERT → DB 不存在该列时报 SQLSTATE 42703。**只读字段必须用 `gorm:"->;-:migration"`**（Creatable=false, Updatable=false, Readable=true, DBName 保留供 JOIN Scan 映射）。
2. **GORM AutoMigrate 被 PG 物化视图阻塞**：MV 引用业务表列，AutoMigrate 每次启动都 ALTER TYPE → PG SQLSTATE 0A000。已由 `dropDependentMaterializedViews()` 兜底。

**对 Phase 53 PortWriteAudit 的影响**：
- PortWriteAudit 是 *新表*，无 MV 引用，无 0A000 风险
- 若走路径 A（GORM AutoMigrate 建表），所有列由 model tag 推导，**不需要 `gorm:"-:migration"`** —— 该 tag 是给"DB 已有列但 model 不应触发 ALTER"场景用的，新表用不上
- 若走路径 B（手写 SQL 建表 + model 加 `gorm:"-:migration"`），planner 必须确认 model 没有任何会被 GORM 写入的"虚假列"（参考 §3.6 列表）。PortWriteAudit 12 列都是真实 DB 列，路径 B 的 `gorm:"-:migration"` 整表级生效，安全

**[VERIFIED: internal/core/db/migrations/migration_201_phase48_component_columns.go:55-70]** migration_201 风格：`ALTER TABLE ops_asset ADD COLUMN IF NOT EXISTS xxx TYPE`，幂等。

**Planner 建议**：走路径 A（GORM AutoMigrate 建表），migration_202 只做 GORM 做不好的事（菜单 seed + 复合索引命名 + helper 授权）。这是最低风险路径。

---

## 2. 锁定决策的代码实证（CONTEXT 引用对照）

### 2.1 D-16 NetworkPortWrite 常量位置 ✅ 准确

**[VERIFIED: pkg/permission/config.go:186]** `NetworkPortQuery PermissionCode = "network:port:query"` — 行号准确。
**[VERIFIED: pkg/permission/config.go:197]** `NetworkPort PermissionCode = "network:port"` — 行号准确。

D-16 在 line 186 后加 `NetworkPortWrite PermissionCode = "network:port:write"` 是正确插入点。

### 2.2 network_router.go ports 组结构 ✅ 准确

**[VERIFIED: internal/api/v1/network/network_router.go:206-214]** 现有 ports 组结构：
```go
ports := r.Group("/ports")
ports.Use(middleware.RequirePermissionsWithQuery([]string{
    "network:port:query",
}, middleware.OpsSelectorReadPerms, core))
{
    SetupPortRouter(ports, core, exportHandler)
}
```

D-09 在 `SetupPortRouter(ports, core, exportHandler)` *之后* 加 `SetupPortWriteRouter(ports, core)` 是正确插入点（line 213 后）。注意 exportHandler 参数对 PortWriteRouter 不需要。

**[VERIFIED: internal/api/v1/network/port_router.go:8-19]** SetupPortRouter 签名 `(r *gin.RouterGroup, core *core.Core, exportHandler *NetworkExportHandler)` —— kebab 命名 `/list` `/collect` `/collect-all` `/batch-delete` 风格确认。Phase 52 的 6 个 kebab 端点与之一致。

### 2.3 migration_195 菜单 seed 模式 ✅ 准确

**[VERIFIED: internal/core/db/migrations/migration_195_reconciliation_exception_rules_menu.go:40-92]** count-then-insert + 通过同级 dashboard 菜单反查 parent_id 模式确认。**关键差异**：migration_195 line 38-39 明确"不 INSERT sys_role_menu（谁也不给）"，Phase 52 D-08 helper 是对此的修复。

**[VERIFIED: internal/core/db/migrations/migration_200_fix_suggestion_config_seeds.go:94-211]** menu_type='F' 按钮权限 seed 模式确认（line 190-201）：
```go
buttonMenu := &models.Menu{
    MenuName: btn.name,
    ParentID: &btnParent.ID,
    Path:     &emptyPath,         // "" 空路径
    MenuType: models.MenuTypeButton, // 'F'
    Visible:  models.VisibleHidden,  // 0 隐藏
    Status:   models.MenuStatusNormal,
    Perms:    &perms,
    Icon:     &icon,               // "#"
    OrderNum: btn.orderNum,
    Remark:   btn.remark,
}
```
Phase 52 D-06 "端口配置" 按钮菜单应完全照此模式（含 Path="" / Icon="#" / Visible=0）。**注意**：D-06 写的 path='write' 与 migration_200 模式（emptyPath=""）不一致 —— planner 应澄清：F 型菜单 path 字段对前端路由生成无影响（routeGenerator 跳过 F 型），填 "" 或 "write" 都可；推荐沿用 migration_200 的 "" 与项目惯例对齐，D-06 的 "write" 可保留为 perms 的语义提示但 path 字段填空。

### 2.4 fix_suggestion_handler.go operlog 实战参考 ✅ 准确

**[VERIFIED: internal/api/v1/asset/fix_suggestion_handler.go:132,177,214,276]** 4 处 `operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationFixSuggestion, operlog.OperTypeXxx)` 调用，每处之后紧跟 `response.Success(c, ...)`。Module 常量定义模式（fix_suggestion_handler.go 同包 line 35）：
```go
const ModuleReconciliationFixSuggestion = "资产对账-修复建议"
```

Phase 52 在 port_write_handler.go 同样定义 `const ModulePortWrite = "端口管理"`（AUDIT-01 锁定的 module 字符串，与父菜单名 "端口状态" 解耦）。

### 2.5 migration_201 手写 SQL 风格 ✅ 准确

**[VERIFIED: internal/core/db/migrations/migration_201_phase48_component_columns.go:41-141]** 手写 SQL 模式：`ADD COLUMN IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS` + DO $$ 块 + count-then-insert 字典 seed。SQLite 分支 `if !isPostgreSQL(db)` 走 AutoMigrate 简化路径。

### 2.6 Phase 51 PortWriteService 接口 ✅ 准确

**[VERIFIED: internal/services/portwrite/port_write_service.go:75-104]** 6 方法 interface + factory 签名确认：
```go
type PortWriteService interface {
    Shutdown(ctx context.Context, portID string, operator string) (*PortResult, error)
    UndoShutdown(ctx context.Context, portID string, operator string) (*PortResult, error)
    SetDescription(ctx context.Context, portID string, desc string, operator string) (*PortResult, error)
    EnableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error)
    DisableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error)
    BatchWritePorts(ctx context.Context, req BatchWriteRequest, operator string) (*BatchResult, error)
}
func NewPortWriteService(db *gorm.DB, deviceExecutor *device.DeviceExecutor, collectionSvc *services.DeviceInfoCollectionService) PortWriteService
```

**[VERIFIED: internal/services/portwrite/port_write_service.go:20-26]** 5 个 sentinel error：`ErrBatchTooLarge` / `ErrEmptyBatch` / `ErrMixedDevices` / `ErrPortNotFound` / `ErrDeviceNotFound`。

**[VERIFIED: internal/services/portwrite/port_write_service.go:33-50]** PortResult struct 形状：
```go
type PortResult struct {
    PortID       string `json:"portId"`
    Action       Action `json:"action"`
    Status       string `json:"status"` // "succeeded" | "failed" | "skipped"
    NoOp         bool   `json:"noOp"`
    CurrentState string `json:"currentState,omitempty"`
    Error        string `json:"error,omitempty"`
    CommandSent  string `json:"commandSent,omitempty"` // 未脱敏（audit 真相源）
}
type BatchWriteRequest struct {
    DeviceID    string   `json:"deviceId"`
    Action      Action   `json:"action"`
    PortIDs     []string `json:"portIds"`
    Description string   `json:"description,omitempty"`
}
```
注：`PortResult` 没有 `DeviceResponse` 字段 —— D-01 audit 表的 `device_response` 列，handler 需从 `PortResult.Error`（失败时）或推断（成功时 "OK"）填，**不能直接从 service 拿**。Phase 51 service 不暴露原始设备响应文本（parseConfigError 已分类），只暴露 Error 字符串。**planner 注意**：D-01 device_response 列填值策略需在 handler 中明确（建议：成功→"OK"，failed→result.Error，skipped→"无需操作"）。

### 2.7 DevicePortStatus 模型字段 ✅ 准确

**[VERIFIED: internal/models/device_port_status.go:31-57]** DevicePortStatus 含 D-02 预 SELECT 所需全部字段：
- `ID string` (uuid)
- `DeviceID string` (uuid)
- `InterfaceName string`
- `AdminStatus string` — D-03 after_value "admin_status" 来源
- `Description string` — D-03 after_value "description" 来源
- `Dot1xEnabled bool` — D-03 after_value "dot1x_enabled" 来源

D-02 的预 SELECT SQL 可直接 `db.First(&port, "id = ?", portID)`，无需手写 SQL。

### 2.8 D-07 父菜单名 "端口状态" ✅ 准确

**[VERIFIED: internal/core/db/migrations/archive/053_fix_menu_paths_unified.sql:184-186]** `UPDATE sys_menu SET component = 'network/ports/index' WHERE menu_name = '端口状态' AND path = 'network/ports';` —— 父菜单名 "端口状态"、path 'network/ports'、component 'network/ports/index' 全部确认。

ROADMAP Phase 52 Success Criteria #6 写的 "端口管理" 父菜单是笔误，D-07 纠正为 "端口状态" 准确。D-08 helper 调用应写 `GrantNewMenuToRolesHavingParent(db, "端口状态", newMenuID)`（不是 "端口管理"）。

### 2.9 Core DI 表面 ✅ 准确

**[VERIFIED: internal/core/core_services.go:17-29]** CoreServices 暴露 handler DI 所需全部字段：
- `DeviceExecutor *device.DeviceExecutor` (line 20)
- `DeviceInfoCollectionService *services.DeviceInfoCollectionService` (line 22)
- `OperLogService services.OperLogService` (line 29)

**[VERIFIED: internal/core/core.go:434]** `c.OperLogService = services.NewOperLogService()` 初始化。

**[VERIFIED: internal/core/core.go:276,284]** DeviceExecutor / DeviceInfoCollectionService 初始化。

handler 通过 `h.core.GetDB()` 拿 *gorm.DB，`h.core.OperLogService` 拿 operlog service，`core.DeviceExecutor` + `core.DeviceInfoCollectionService` 用于 NewPortWriteService 工厂。

---

## 3. CONTEXT 未覆盖的雷区

### 3.1 Permission 常量注册同步（PERM-01 隐含任务）

**[VERIFIED: pkg/permission/config.go:200-264]** `GetRoutePermissions()` 函数返回 `[]RoutePermission` —— 一个 *路由 → 权限* 的查找表。**但它只覆盖 system 模块（user/role/menu/dept/post/workstation）**，network 模块的所有路由（devices/credentials/templates/command 等）**都没在 GetRoutePermissions 里**。

**含义**：D-16 在 config.go:186 加 `NetworkPortWrite` 常量是充分的，**不需要**也**不应该**往 GetRoutePermissions 里加 6 个新映射。network 模块的路由鉴权全靠 router setup 时的 `middleware.RequirePermissions(...)` 显式挂载（network_router.go 多处），不依赖 GetRoutePermissions。planner 不要画蛇添足。

**前端权限定义同步**：Phase 53 前端任务，本 phase 不涉及。前端通过 sys_menu.perms 字段（D-06 seed 的 'network:port:write'）动态获取权限，不需要前端硬编码权限常量。

### 3.2 sys_menu schema 必填列（D-06 menu seed 隐含任务）

**[VERIFIED: internal/models/menu.go:47-68]** Menu struct 字段：
- `MenuName string` (gorm:"size:50;not null") — **必填**
- `ParentID *string` (nullable)
- `OrderNum int` (default:0)
- `Path *string` (nullable)
- `Component *string` (nullable)
- `MenuType MenuType` (default:'M') — **F 型按钮填 'F'**
- `Visible VisibleType` (default:1，即 VisibleShow) — **F 型按 migration_200 模式应填 VisibleHidden=0**
- `Status MenuStatus` (default:0，即 Normal)
- `Perms *string` (nullable) — **本 phase 关键字段**
- `Icon *string` (nullable) — **F 型按惯例填 "#"**
- `Remark string`
- `Meta *MenuMeta` (JSONB)

**重要发现**：**Go sys_menu 模型没有 `is_frame` / `is_cache` 单独列**（CONTEXT additional_context 提到的潜在 landmine）。这些字段在 Java 版 XingRan 是单独列，Go 版用 `Meta` JSONB 统一管理（参考 memory `xingran-menu-no-java-fields`）。planner 不要在 menu seed 里加 `IsFrame` / `IsCache` 字段（会编译失败）。

**D-06 menu seed 字段建议**（参照 migration_200 button 模式）：
```go
menu := &models.Menu{
    MenuName: "端口配置",
    ParentID: &parentMenuID,           // 通过 menu_name='端口状态' 反查
    OrderNum: 100,                     // F 型按钮，靠后
    Path:     &emptyPath,              // "" (D-06 写的 "write" 可保留但建议空)
    Component: nil,                    // F 型无 component
    MenuType:  models.MenuTypeButton,  // 'F'
    Visible:   models.VisibleHidden,   // 0 隐藏
    Status:    models.MenuStatusNormal,
    Perms:     &perms,                 // "network:port:write"
    Icon:      &icon,                  // "#"
    Remark:    "Phase 52: 端口写操作按钮权限（5 单端口 + 1 batch）",
}
```

### 3.3 5 个 sentinel error → HTTP 码翻译表（隐含任务）

**[VERIFIED: internal/services/portwrite/port_write_service.go:20-26 + batch_orchestrator.go]** + **[CITED: Phase 51 51-01-SUMMARY.md:197]** "transport → 503, device_rejected → 422"。完整翻译表：

| Sentinel Error | HTTP Code | response.Error msg | 触发条件 |
|----------------|-----------|-------------------|----------|
| `ErrBatchTooLarge` | 400 | "批量端口数超过上限 50" | PortIDs 长度 > 50 |
| `ErrEmptyBatch` | 400 | "批量端口列表为空" | PortIDs 长度 == 0 |
| `ErrMixedDevices` | 400 | "批量端口必须属于同一设备" | PortIDs 跨设备（D-17 锁定拒绝） |
| `ErrPortNotFound` | 404 | "端口不存在" | portID 在 sys_device_port_status 找不到（仅 batch 入口校验路径） |
| `ErrDeviceNotFound` | 404 | "设备不存在" | port 行存在但 device_id 为空，或 device 行不存在 |
| service 内 `fmt.Errorf("query port: %w", err)` | 500 | "查询端口失败" | DB 查询异常 |
| service 内 `fmt.Errorf("query device: %w", err)` | 500 | "查询设备失败" | DB 查询异常 |
| PortResult.Status == "failed" + 非 sentinel err | 200 + 业务结果 | （不调 response.Error） | transport_error / device_rejected — **service 已正常返回 PortResult，handler 写 audit(status=failed) 后正常 response.Success(PortResult)** |

**关键区分**：sentinel error（入口校验失败）→ response.Error；PortResult.Status=failed（SSH 执行失败）→ response.Success(返回 PortResult) + audit(写 failed 行)。这两种"失败"的 HTTP 处理路径不同，planner 必须在 handler 中明确分支。

### 3.4 PortResult → audit 行映射 + batch audit 写入策略

D-04 锁定 NoOp/skipped 也写 audit。映射表：

| PortResult.Status | audit.status | audit.command_sent | audit.device_response | audit.failure_reason |
|-------------------|--------------|--------------------|-----------------------|----------------------|
| `succeeded` | `succeeded` | result.CommandSent | "OK" | NULL |
| `failed` | `failed` | result.CommandSent | result.Error（含 transport_error/device_rejected 详情） | result.Error |
| `skipped` (NoOp) | `skipped` | "" (空) | "无需操作" | NULL |

**batch handler 的 N 条 audit 写入**（D-05 + CONTEXT specifics line 174 "每条独立 INSERT"）：
```go
// 伪代码
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModulePortWrite, operlog.OperTypeBatch,
    operlog.WithOperParam(batchSummaryJSON))  // {action, batch_size, succeeded_count, failed_count, skipped_count, device_id, audit_ids:[...]}

for _, pr := range append(append(result.Succeeded, result.Failed...), result.Skipped...) {
    auditRow := buildAuditRowFromPortResult(pr, beforeSnapshot[pr.PortID], operIDorNil)
    if err := h.core.GetDB().Create(auditRow).Error; err != nil {
        applogger.Warnf("port_write audit insert failed portID=%s: %v", pr.PortID, err)
        // 不阻塞响应，继续写下一条
    }
}
response.Success(c, result)
```

### 3.5 audit oper_log_id 列填充策略（与 §1.1 路径 C 配套）

走 §1.1 路径 C（兜底，不改 operlog 包）时，`audit.oper_log_id` 列的处理：
- **本 phase 不填**（NULL）—— D-13 已锁定"可空"
- Phase 53 前端 UI-04 跳转靠 operlog → audit 反向（解析 oper_param.audit_ids）
- 若 planner 后续决定补"audit → operlog 正向跳转"，可加后台 reconciliation（Phase 53+ 任务，本 phase 不做）

### 3.6 PortWriteAudit model GORM tag 注意事项

**12 列 model 推荐定义**（路径 A：靠 GORM AutoMigrate 建表）：
```go
type PortWriteAudit struct {
    ID            string          `gorm:"type:uuid;primary_key" json:"id"`
    DeviceID      string          `gorm:"type:uuid;not null;index:idx_port_write_audit_device_port_created,priority:1" json:"deviceId"`
    PortID        string          `gorm:"type:uuid;not null;index:idx_port_write_audit_device_port_created,priority:2" json:"portId"`
    Action        string          `gorm:"size:32;not null" json:"action"`
    BeforeValue   json.RawMessage `gorm:"type:jsonb" json:"beforeValue"`
    AfterValue    json.RawMessage `gorm:"type:jsonb" json:"afterValue"`
    CommandSent   string          `gorm:"type:text" json:"commandSent"`
    DeviceResponse string         `gorm:"type:text" json:"deviceResponse"`
    Status        string          `gorm:"size:16;not null" json:"status"`
    FailureReason *string         `gorm:"type:text" json:"failureReason,omitempty"`
    Operator      string          `gorm:"size:50" json:"operator"`
    OperLogID     *string         `gorm:"type:uuid" json:"operLogId,omitempty"`
    CreatedAt     time.Time       `gorm:"not null;index:idx_port_write_audit_created" json:"createdAt"`
}
func (PortWriteAudit) TableName() string { return "sys_port_write_audit" }
```

**注意事项**：
1. `BeforeValue` / `AfterValue` 用 `json.RawMessage` 而非 `map[string]interface{}` —— 避免每次 audit 写入都做 JSON marshal/unmarshal
2. 复合索引 `(device_id, port_id, created_at)` 用 GORM composite index tag；命名 `idx_port_write_audit_device_port_created` 显式控制（避免 GORM 自动命名与 PG `_key` 后缀冲突，参考 database.go:423-475 auditConstraintNaming）
3. 单列索引 `(created_at)` 用单独 index tag `idx_port_write_audit_created`
4. `FailureReason` / `OperLogID` 是 nullable → 用 `*string` 指针类型
5. **不需要** `UpdatedAt` —— audit 表 append-only，无 update 路径
6. **不需要** `BaseTimeLine` embed —— BaseTimeLine 含 UpdatedAt，audit 不需要
7. `BeforeCreate` 钩子生成 UUID（仿 DevicePortStatus pattern）：
   ```go
   func (a *PortWriteAudit) BeforeCreate(tx *gorm.DB) error {
       if a.ID == "" { a.ID = uuid.New().String() }
       return nil
   }
   ```

### 3.7 cache_keys.go 内容（INFRA-03）

**[VERIFIED: CLAUDE.md Cache Key Helpers section]** cache key 常量惯例：包级 const + `Get<Key>(args...)` helper 函数。

D-10 锁定只定义常量不写入。文件 `internal/services/portcollection/cache_keys.go` 内容（最简）：
```go
package portcollection

const (
    CacheKeyPortWriteResult = "port:write:result:%s"  // %s = port_id
    CacheKeyPortWriteBatch  = "port:write:batch:%s"   // %s = batch_id
)

// 占位 helper（Phase 53+ 接入 CacheProvider 时启用）
// func GetPortWriteResultKey(portID string) string { return fmt.Sprintf(CacheKeyPortWriteResult, portID) }
// func GetPortWriteBatchKey(batchID string) string  { return fmt.Sprintf(CacheKeyPortWriteBatch, batchID) }
```

planner 可选：是否同步定义 helper 函数（即使本 phase 不调用）。建议定义，Phase 53 直接用，避免反复修改。

### 3.8 SQLite 兼容（migration_201 模式）

**[VERIFIED: migration_201:45-51]** migration_201 在 SQLite 分支用 AutoMigrate，跳过 PG-only 的 partial unique index。Phase 52 migration_202 若走路径 A（GORM AutoMigrate 建表），SQLite 自动兼容；若走路径 B（手写 SQL），planner 必须加 SQLite 分支（JSONB 在 SQLite 是 TEXT，CREATE INDEX IF NOT EXISTS 两边都支持）。

测试若用 SQLite in-memory（参考 Phase 51 port_write_service_test.go），audit 表自动通过 AutoMigrate 建。**但**: 复合索引在 SQLite 与 PG 行为略不同（partial index 语法不同），audit 表无 partial index 需求，影响可忽略。

---

## 4. Validation Architecture（Nyquist Dimension 8）

### 4.1 测试基础设施现状

| Property | Value |
|----------|-------|
| Framework | Go 内置 testing + testify（assert/mock）+ Phase 51 已用的 sqlite in-memory |
| Config file | 无独立 config（Go 标准 *_test.go 同包） |
| Quick run command | `go test ./internal/api/v1/network/... -count=1` |
| Full suite command | `go test ./... -count=1` |
| operlog 回归锁 | `go test ./internal/utils/operlog/... -count=1 -v` |
| Phase 51 service 回归 | `go test ./internal/services/portwrite/... -count=1 -v` |

### 4.2 Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| PERM-01 | NetworkPortWrite 常量定义 | unit（编译时） | `go build ./pkg/permission/...` | ❌ Wave 1 |
| PERM-02 | 6 端点组级 RequirePermissions 挂载 | source-grep 断言 | 见 §4.3 验证脚本 | ❌ Wave 1 |
| INFRA-02 | SetupPortWriteRouter 注册 + 6 端点路由可解析 | unit（gin.Engine 路由树断言） | 见 §4.3 | ❌ Wave 1 |
| AUDIT-01 | operlog.Record 在 response.Success 之前调用 | source-grep 断言 + handler unit test | 见 §4.3 | ❌ Wave 1 |
| CONV-01..04 | OperType 映射正确 | source-grep 断言 | 见 §4.3 | ❌ Wave 1 |
| PORT-01..05 | 5 单端口 handler 调对应 service 方法 | handler unit test（mock service） | `go test ./internal/api/v1/network/... -run TestPortWriteHandler -v` | ❌ Wave 1 |
| BATCH-01 | batch handler 调 BatchWritePorts + 写 N audit | handler unit test | `go test ./internal/api/v1/network/... -run TestPortWriteBatchHandler -v` | ❌ Wave 1 |
| INFRA-01 | sys_port_write_audit 表 + 索引存在 | schema introspection | `db.Migrator().HasTable("sys_port_write_audit")` + 列检查 | ❌ Wave 2 |
| AUDIT-03 | audit 行写入（succeeded/failed/skipped 三态） | handler unit test 断言 audit 行 | 同 PORT-01..05 测试覆盖 | ❌ Wave 2 |
| PERM-03 | sys_menu seed 行存在 + sys_role_menu grant 行存在 | migration unit test | migration_202 跑后 `SELECT COUNT(*) FROM sys_menu WHERE menu_name='端口配置'` + `SELECT COUNT(*) FROM sys_role_menu WHERE menu_id=...` | ❌ Wave 2 |

### 4.3 推荐验证脚本（per task commit / per wave merge）

**Wave 1 commit 后必跑**：
```bash
# 1. operlog 回归锁 intact
go test ./internal/utils/operlog/... -count=1 -v

# 2. Phase 51 service 无回归
go test ./internal/services/portwrite/... -count=1 -v

# 3. 全包编译
go build ./...

# 4. permission 常量编译
go build ./pkg/permission/...

# 5.（可选）handler 单测（若 Wave 1 写了）
go test ./internal/api/v1/network/... -count=1 -v
```

**Wave 2 commit 后必跑**（叠加 Wave 1 全部）：
```bash
# 6. migration_202 在 SQLite in-memory 跑通
go test ./internal/core/db/migrations/... -run TestMigrate202 -v

# 7.（如可能）PG 集成：表存在 + 索引存在 + 菜单 seed + 角色授权
# 这一步需要 PG 连接，本 phase 可标记 manual-only，由 Phase 54 UAT 覆盖
```

**Source-grep 断言（可在 *_test.go 中编码）**：
```go
// port_write_router_test.go
func TestSetupPortWriteRouter_GroupPermission(t *testing.T) {
    // 跑一次 SetupPortWriteRouter，断言 /network/ports/write/* 6 个端点都在 group 里
    // 且 group handlers 含 RequirePermissions 中间件
}

// port_write_handler_test.go
func TestPortWriteHandler_OperlogBeforeResponse(t *testing.T) {
    // 用 mock OperLogService，调 handler，断言 RecordAsync 被调 + 调用顺序在 response 写出前
}
```

### 4.4 Phase Gate（per-wave merge + verify-work）

- **Per task commit**: §4.3 第 1-5 项
- **Per wave merge**: §4.3 全部 7 项
- **Phase gate (`/gsd:verify-work`)**: 全部 7 项 + `go vet ./...` exit 0 + `go build ./...` exit 0

### 4.5 Wave 0 Gaps

- `internal/api/v1/network/port_write_router_test.go` — 覆盖 PERM-02 / INFRA-02
- `internal/api/v1/network/port_write_handler_test.go` — 覆盖 PORT-01..05 / BATCH-01 / AUDIT-01/02/03
- `internal/core/db/migrations/migration_202_port_write_audit_test.go` — 覆盖 INFRA-01 / PERM-03
- `internal/core/db/migrations/menu_grant_helpers_test.go` — 覆盖 D-08 helper 幂等性
- Framework 已就位（testify/mock），无 install gap

---

## 5. Recommended Plan Split（确认或修正 ROADMAP）

### 5.1 ROADMAP 原始 2-wave 切分

ROADMAP.md Phase 52 段：
- Wave 1: NetworkPortWrite 常量 + port_write_router.go + port_write_handler.go + operlog.Record 调用
- Wave 2: migration_202 + 菜单 seed + helper + cache_keys

### 5.2 研究后的切分建议（与 ROADMAP 一致，细化任务）

**Wave 1（52-01-PLAN.md）—— HTTP wiring + operlog + 权限常量**
1. `pkg/permission/config.go` 加 `NetworkPortWrite` 常量（D-16）
2. `internal/services/portcollection/cache_keys.go` 定义 2 个常量（INFRA-03 / D-10）— *可放 Wave 1 或 Wave 2，建议 Wave 1 因极简*
3. `internal/api/v1/network/port_write_handler.go` 新建：6 handler 方法 + ModulePortWrite 常量 + before_value 预 SELECT（D-02）+ after_value 填充（D-03）+ audit 行写入（D-04）+ sentinel→HTTP 翻译（§3.3）+ operlog.Record 调用（AUDIT-01）
4. `internal/api/v1/network/port_write_router.go` 新建：`/write` 子组 + 组级 `RequirePermissions([]string{string(permission.NetworkPortWrite)}, core)`（注意 2-arg）+ 6 kebab 端点（D-09）
5. `internal/api/v1/network/network_router.go` 改：line 213 后插 `SetupPortWriteRouter(ports, core)`（D-09）
6. handler/router 单元测试（§4.5 Wave 0 Gaps）
7. §4.3 Wave 1 验证脚本全绿

**Wave 1 隐含决策**（planner 必须明确）：
- D-13 audit↔operlog 关联走哪条路径？**研究强烈推荐路径 C（兜底，不改 operlog 包）**，否则 Wave 1 任务量翻倍且触及 Phase 34 锁定区
- audit 表尚不存在（Wave 2 才建），Wave 1 handler 调 `db.Create(&PortWriteAudit{...})` 会失败 → **Wave 1 handler 测试必须 mock DB 或用 sqlite in-memory + 手动 AutoMigrate PortWriteAudit model**（即使 Wave 2 才正式注册）

**Wave 2（52-02-PLAN.md）—— 迁移 + 菜单 seed + helper**
1. `internal/models/port_write_audit.go` 新建 PortWriteAudit model（§3.6）
2. `internal/core/db/database.go` AutoMigrate 列表加 `&models.PortWriteAudit{}`（§1.2 路径 A）
3. `internal/core/db/database.go` AutoMigrate() postgres 分支加 `migrations.Migrate202PortWriteAudit(d.DB)` 显式调用（§1.2）
4. `internal/core/db/migrations/menu_grant_helpers.go` 新建：`GrantNewMenuToRolesHavingParent` 幂等函数（D-08）
5. `internal/core/db/migrations/migration_202_port_write_audit.go` 新建：复合索引 SQL + 菜单 seed（count-then-insert，父菜单名 "端口状态"，§2.3 + §3.2）+ helper 调用
6. migration 单元测试 + helper 单元测试（§4.5）
7. §4.3 Wave 2 验证脚本全绿

### 5.3 是否需要切 3-wave？

**否**。Wave 1 任务 1-2 极简（常数定义），任务 3-5 是同一组文件强耦合（handler/router 同包），不宜再切。Wave 2 任务 1-3 强耦合（model + AutoMigrate 注册），任务 4-5 强耦合（helper + migration 调 helper）。2-wave 是最优粒度。

### 5.4 Wave 顺序硬约束

**Wave 1 必须先于 Wave 2**：handler 引用 `models.PortWriteAudit`（Wave 2 任务 1 才定义），所以 **Wave 1 任务 3 写 handler 时，PortWriteAudit model 必须已定义**。

两种解决路径：
- **路径 X（推荐）**：把"PortWriteAudit model 定义"从 Wave 2 提到 Wave 1 任务 3 之前（即 Wave 1 任务 1.5）。model 定义本身 5 分钟，不引入迁移逻辑。Wave 2 仅做"AutoMigrate 注册 + migration_202 + helper"。
- **路径 Y**：Wave 1 handler 用 interface{} 或 map[string]interface{} 临时写 audit，Wave 2 再改回 PortWriteAudit model。**不推荐**——重复修改、易引入 bug。

**planner 强烈建议路径 X**：Wave 1 任务清单调整为：
1. NetworkPortWrite 常量
1.5 PortWriteAudit model 定义（仅 model 文件，不动 database.go）
2. cache_keys.go 常量
3. port_write_handler.go
4. port_write_router.go
5. network_router.go 改
6. 测试

---

## 6. 假设与待澄清问题

### Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | D-13 走路径 C（兜底，不改 operlog 包）是更优选择 | §1.1 | 若 planner 坚持 WithOperID 主路径，Wave 1 任务量翻 3-4 倍且触及 regression_test.go 锁定区 |
| A2 | migration_202 走路径 A（GORM AutoMigrate 建表 + migration 仅做 seed/index/helper） | §1.2 | 若走路径 B（纯手写 SQL），需多写 SQLite 分支 + 整表 `gorm:"-:migration"` tag |
| A3 | Wave 1 应提前定义 PortWriteAudit model（路径 X） | §5.4 | 若不提前，Wave 1 handler 无法引用 model，需临时 interface{} |
| A4 | `WithJsonResult` option 不存在（仅 recordConfig 字段） | §1.1 | CLAUDE.md/CONTEXT 写了 WithJsonResult 但实际只有 3 个 option 函数；不影响本 phase |
| A5 | `device_response` audit 列填值策略：成功→"OK"，failed→result.Error，skipped→"无需操作" | §2.6 / §3.4 | Phase 51 service 不暴露原始设备响应文本，本策略是合理推断；若需真实响应文本需改 Phase 51 service（违反零侵入） |
| A6 | F 型菜单 path 字段填 ""（不是 D-06 写的 "write"） | §3.2 | 不影响功能（F 型 path 不参与路由生成）；与 migration_200 惯例对齐 |
| A7 | sys_menu Go 模型无 is_frame/is_cache 列（用 Meta JSONB） | §3.2 | 从 Java 版复制 SQL 会 SQLSTATE 42703（memory `xingran-menu-no-java-fields` 已记录） |

### Open Questions

1. **D-13 audit↔operlog 关联机制**（最关键）
   - What we know: CONTEXT D-13 推荐 WithOperID 主路径 + 兜底路径
   - What's unclear: planner 是否接受研究强烈推荐的路径 C（兜底）
   - Recommendation: **走路径 C**。理由 §1.1 详述。若 planner 仍坚持主路径，请显式在 plan 中列出 operlog 包改造的 4 个子任务（option + 实现层 + mock + 测试）

2. **migration_202 走路径 A 还是 B**
   - What we know: 两条路径都可行
   - Recommendation: **路径 A**（GORM AutoMigrate 建表 + migration 仅做 GORM 做不好的事）。最低风险

3. **Wave 1 是否提前定义 PortWriteAudit model（路径 X vs Y）**
   - Recommendation: **路径 X**（提前定义 model）。避免 Wave 1 handler 用临时类型

4. **device_response 列填值策略**（§2.6 / A5）
   - What we know: Phase 51 service 不暴露原始响应文本
   - Recommendation: 接受 A5 推断策略；若需真实响应，记入 deferred（Phase 51 service 改造，违反零侵入原则）

---

## 7. 环境可用性

无新外部依赖。本 phase 仅消费 Go 标准库 + 已有依赖（gin / gorm / uuid / testify）。Phase 51 service + Phase 50 vendor templates 已 shipped。

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | 全 phase | ✓ | 1.24 (CLAUDE.md) | — |
| gin | router | ✓ | v1.10.0 | — |
| gorm | model + AutoMigrate | ✓ | v1.30.5 | — |
| google/uuid | PortWriteAudit.ID / operID | ✓ | v1.6.0 | — |
| testify | handler/router 单测 | ✓ | Phase 51 已用 | — |
| sqlite (modernc.org/sqlite) | in-memory 测试 | ✓ | v1.40.1 | — |

**Missing dependencies with no fallback**: 无
**Missing dependencies with fallback**: 无

---

## 8. Security Domain

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | JWT 由现有 auth 中间件处理，本 phase 不动 |
| V3 Session Management | no | 同上 |
| V4 Access Control | yes | RequirePermissions([network:port:write], core) 组级 RBAC（PERM-02） |
| V5 Input Validation | yes | PortID UUID 格式校验 + Description 长度 ≤80（Phase 50 RenderCommand 入口校验）+ batch size ≤50（Phase 51 D-17） |
| V6 Cryptography | no | 不涉及密码学 |

**Known Threat Patterns**:

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 越权写端口（无 network:port:write 权限用户调写端点） | Elevation of Privilege | RequirePermissions 组级中间件（PERM-02） |
| 跨设备批量（绕过 D-17 单设备约束） | Tampering | Phase 51 service 入口 ErrMixedDevices 校验（已 shipped） |
| audit 篡改（修改 sys_port_write_audit 行） | Repudiation | audit 表 append-only（无 UPDATE 路径，model 无 Updatable 字段） + operlog 同步双写 |
| 敏感字段泄漏到 operlog | Information Disclosure | operlog 11 关键词自动脱敏（regression_test.go 锁定）；audit.command_sent 是未脱敏真相源（D-15）；本 phase 端点非敏感端点（无 password/secret/key 字段）→ 用 Record 不用 RecordWithBody |
| SSH 命令注入（description 字段注入恶意字符） | Injection | Phase 50 RenderCommand 已做模板化（不拼字符串）；Description ≤80 字符校验在 service 入口 |

---

## 9. Sources

### Primary (HIGH confidence)
- `internal/utils/operlog/operlog.go` — Record 签名 + RecordOption 模式 + 4 个 option 函数（实际只有 WithOperParam/WithStatus/WithErrorMsg，WithJsonResult 不存在）
- `internal/utils/operlog/regression_test.go` — 25 常量锁 + 6 参数签名锁 + 18 关键词锁
- `internal/services/oper_log_service.go:42-73` — RecordAsync 实现（async fire-and-forget，struct 内部构造无 ID 字段）
- `internal/models/log.go:6-25` — OperLog struct embed BaseTimeLine
- `internal/models/base.go:30-42` — BaseTimeLine.BeforeCreate 钩子（预设 ID 保留）
- `internal/models/menu.go:47-68` — Menu schema（无 is_frame/is_cache 单独列，用 Meta JSONB）
- `internal/models/device_port_status.go:31-57` — D-02 预 SELECT 字段源
- `pkg/permission/config.go:186,197,200-264` — NetworkPortQuery 位置 + NetworkPort 父权限 + GetRoutePermissions 仅覆盖 system 模块
- `pkg/middleware/permission.go:200` — RequirePermissions(perm []string, core *core.Core) 实际 2-arg 签名
- `internal/api/v1/network/network_router.go:206-214` — ports 组 + SetupPortRouter 调用点
- `internal/api/v1/network/port_router.go:8-19` — kebab 路由命名参考
- `internal/core/db/database.go:296,308-391,412-417` — migration 不自动调用 + AutoMigrate model 列表 + 仅 175/176 显式调用
- `internal/core/db/migrations/migration_195_reconciliation_exception_rules_menu.go:38-39,67-88` — 菜单 seed + 不授权模式
- `internal/core/db/migrations/migration_200_fix_suggestion_config_seeds.go:94-211` — menu_type='F' button seed 完整模式
- `internal/core/db/migrations/migration_201_phase48_component_columns.go:41-141` — 手写 SQL migration 风格
- `internal/core/db/migrations/archive/053_fix_menu_paths_unified.sql:184-186` — "端口状态" 菜单实存
- `internal/services/portwrite/port_write_service.go:20-104` — 5 sentinel + PortResult/BatchWriteRequest + 6 方法 interface + factory
- `internal/core/core_services.go:17-29` — Core DI 表面
- `internal/core/core.go:276,284,434` — DeviceExecutor/DeviceInfoCollectionService/OperLogService 初始化
- `internal/api/v1/asset/fix_suggestion_handler.go:132,177,214,276` — operlog.Record 实战调用模式
- `.planning/phases/51-w2-portwriteservice-batch-orchestrator-mock-tests/51-01-SUMMARY.md:197` — Phase 51 service 推荐的 sentinel→HTTP 翻译

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` — 19 项 phase 需求（AUDIT/PERM/INFRA/CONV/PORT/BATCH 段）
- `.planning/ROADMAP.md` Phase 52 段 — 8 条 Success Criteria（含 D-07 已纠正的 "端口管理" → "端口状态" 笔误）
- project memory `migration-grant-new-menu-precision-helper.md` — D-08 helper 设计依据
- project memory `gorm-migration-tag-does-not-block-insert.md` — gorm:"-:migration" 不阻止 INSERT 教训
- project memory `xingran-menu-no-java-fields.md` — Go sys_menu 无 is_frame/is_cache 列教训

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 全部依赖已在项目中（gin/gorm/uuid/testify），无新依赖
- Architecture: HIGH — 6 handler/router/operlog/migration 模式都有项目内最近参考实现（fix_suggestion_handler / migration_200 / migration_201）
- Pitfalls: HIGH — 3 个关键偏差（migration 注册 / RequirePermissions 签名 / WithOperID 风险）全部直接核对源码确认
- D-13 机制选择: MEDIUM — 推荐路径 C（兜底）有充分理由，但最终决定权在 planner，若选主路径风险翻 3-4 倍

**Research date:** 2026-07-07
**Valid until:** 2026-08-06（30 天，稳定 Go/Gin/GORM 架构，无 fast-moving 依赖）

## RESEARCH COMPLETE

**Phase:** 52 - W3: Router/Handler/Operlog/Permission/Migration
**Confidence:** HIGH

### Key Findings

- **3 个关键修正** surfaced：(1) migration_202 不会自动跑——必须在 database.go 显式调用或加入 AutoMigrate 列表；(2) RequirePermissions 实际是 2-arg 签名（含 core），CONTEXT/CLAUDE.md 漏写；(3) D-13 WithOperID 主路径风险高（RecordAsync 内部构造 struct 不接收预设 ID），推荐走兜底路径 C
- **2 个推荐路径**：(A) PortWriteAudit 走 GORM AutoMigrate 建表 + migration_202 仅做 seed/index/helper；(X) Wave 1 提前定义 PortWriteAudit model，避免 handler 临时类型
- **Phase 51 service 零侵入确认**：6 方法签名 + 5 sentinel + PortResult 形状全部就绪，handler 直接消费
- **菜单 seed 完整模式确认**：父菜单名 "端口状态"（不是 "端口管理"），F 型按钮 Visible=0/Path=""/Icon="#"，sys_menu Go 模型无 is_frame/is_cache 列
- **sentinel→HTTP 翻译表 + PortResult→audit 行映射表** 已就绪，覆盖 succeeded/failed/skipped 三态

### File Created

`D:\code\ClaudeCode\xingran-go-backend\.planning\phases\52-w3-router-handler-operlog-permission-migration\52-RESEARCH.md`

### Confidence Assessment

| Area | Level | Reason |
|------|-------|--------|
| Standard Stack | HIGH | 全部依赖已在项目中，无新依赖 |
| Architecture | HIGH | 6 模式都有项目内最近参考实现（fix_suggestion_handler / migration_200 / migration_201） |
| Pitfalls | HIGH | 3 关键偏差直接核对源码确认（migration 注册 / RequirePermissions 签名 / WithOperID 风险） |
| D-13 机制 | MEDIUM | 推荐路径 C，最终决定权在 planner |

### Open Questions

1. D-13 audit↔operlog 走路径 C（兜底，不改 operlog 包）还是主路径 WithOperID（改 operlog + Recorder 接口 + regression_test.go）？
2. migration_202 走路径 A（GORM AutoMigrate 建表）还是路径 B（纯手写 SQL）？
3. Wave 1 是否提前定义 PortWriteAudit model（路径 X）？

### Ready for Planning

Research complete. Planner 可基于本 RESEARCH.md 创建 52-01-PLAN.md（Wave 1）+ 52-02-PLAN.md（Wave 2）。**强烈建议 planner 在 plan 开头显式回答 3 个 Open Questions**（默认采用研究推荐的 C / A / X 路径组合）。
