# Phase 52: W3 — Router/Handler/Operlog/Permission/Migration - Context

**Gathered:** 2026-07-07
**Status:** Ready for planning
**Source:** v1.19 STATE.md 锁定决策 + Phase 51 CONTEXT D-10..D-18 (service 契约已落地) + REQUIREMENTS.md AUDIT/PERM/INFRA/CONV/PORT/BATCH 段 + ROADMAP Phase 52 段 + 代码 scout (operlog/permission/network_router/migration_195/OperLog 模型)

<domain>
## Phase Boundary

Phase 51 PortWriteService 契约已稳定(6 方法 + 28 mock 测试绿色)。本 phase 把 service 暴露成 HTTP:6 个写端点 + operlog 审计 + `sys_port_write_audit` 未脱敏真相源表 + `network:port:write` 权限隔离 + 菜单 seed。本 phase 不改 Phase 51 service 任何签名(零侵入)。

**In scope**:
- `internal/api/v1/network/port_write_router.go` 新建:`/network/ports/write` 子组 + 6 端点(5 单端口 + 1 batch),组级 `RequirePermissions([network:port:write])`
- `internal/api/v1/network/port_write_handler.go` 新建:6 handler 方法,每个调对应 Phase 51 service 方法,`response.Success` 前写 operlog + N 条 audit
- `pkg/permission/config.go` 新增 `NetworkPortWrite = "network:port:write"` 常量
- `internal/core/db/migrations/migration_202_port_write_audit.go` 新建:`sys_port_write_audit` 表(12 列含 oper_log_id FK)+ "端口配置" 菜单 seed + helper 调用
- `internal/core/db/migrations/menu_grant_helpers.go` 新建:`GrantNewMenuToRolesHavingParent(db, parentMenuName, newMenuID)` 幂等授权 helper
- `internal/services/portcollection/cache_keys.go` 新建:仅定义 `CacheKeyPortWriteResult` / `CacheKeyPortWriteBatch` 常量(MVP 不写入)
- `internal/utils/operlog/operlog.go` 增 `WithOperID(id string)` RecordOption(支持 audit↔operlog 精准 FK)
- `internal/api/v1/network/network_router.go` 改:注册 `SetupPortWriteRouter`

**Out of scope**:
- 前端 BulkWriteDrawer / Modal / API wrappers / 列表页按钮 — Phase 53
- 真机 SSH e2e / 真机 UAT — Phase 54
- BATCH-05 进度反馈(WebSocket/SSE)— v1.19.x(需重构 Phase 51 batch 同步契约)
- `sys_port_write_audit` 详情查看 UI — v1.19.x+
- audit 表数据 backfill(新表无历史)
- service 层任何签名变更(Phase 51 已锁)

</domain>

<decisions>
## Implementation Decisions

### 审计表 schema(AUDIT-03 / INFRA-01)

- **D-01: `sys_port_write_audit` 12 列定义**
  | 列 | 类型 | 说明 |
  |---|---|---|
  | `id` | UUID PK | `gen_random_uuid()` |
  | `device_id` | UUID NOT NULL | FK→sys_network_device.id |
  | `port_id` | UUID NOT NULL | FK→sys_device_port_status.id |
  | `action` | varchar(32) NOT NULL | shutdown/undo_shutdown/description/dot1x_enable/dot1x_disable(单端口)/batch(汇总) |
  | `before_value` | JSONB | `{admin_status, dot1x_enabled, description}` 快照 |
  | `after_value` | JSONB | 目标态快照(同结构) |
  | `command_sent` | TEXT | 未脱敏命令串(D-15 真相源) |
  | `device_response` | TEXT | 设备原始响应(`% Error:...` / OK) |
  | `status` | varchar(16) NOT NULL | `succeeded` / `failed` / `skipped` 枚举 |
  | `failure_reason` | TEXT | 失败原因(transport_error/device_rejected 详情),可空 |
  | `operator` | varchar(50) | 操作人 username |
  | `oper_log_id` | UUID | FK→sys_oper_log.id(可空,见 D-13) |
  | `created_at` | timestamp | 默认 now() |
  - 索引:`(device_id, port_id, created_at)` 复合 + `(created_at)` 单列(REQUIREMENTS INFRA-01 + 本 phase D 确认两索引)
  - 表名锁定:`sys_port_write_audit`(单数,与 `sys_device_port_status` 风格一致)

- **D-02: before_value 捕获 = handler 预 SELECT 快照(不改 Phase 51)**
  - handler 在调 Phase 51 service 方法**之前**,`SELECT admin_status, dot1x_enabled, description, device_id, interface_name FROM sys_device_port_status WHERE id = ?` 取当前态
  - 拼 `before_value = {"admin_status":"up","dot1x_enabled":false,"description":"旧描述"}`
  - Phase 51 service 契约零侵入(D-15 锁定 service 不拥有审计)
  - port DB 行不存在(端口"消失"):before_value = null/`{}`,仍写 audit(status 依 service 返回)

- **D-03: after_value = 目标态快照(handler 同步填,不等 Enqueue 采集)**
  - shutdown → `{"admin_status":"down"}`;undo_shutdown → `{"admin_status":"up"}`;dot1x_enable → `{"dot1x_enabled":true}`;dot1x_disable → `{"dot1x_enabled":false}`;description → `{"description":"新描述"}`
  - 设备已接受命令 = 目标达成;Enqueue 1-2s 后采集只刷新 `sys_device_port_status`,不回填 audit
  - NoOp/skipped 路径:after_value = before_value(无变化)

- **D-04: audit 写入归属 + 时机 = handler 同步,`response.Success` 之前**
  - 遵循 CLAUDE.md operlog 强制约定:写操作 success path 末尾、`response.Success` 之前
  - audit 行同样在 handler 同步写(不走 goroutine),保证 HTTP 响应返回时 audit 已落库
  - **NoOp/skipped 路径也写 audit**(Phase 51 D-15 锁定):status=`skipped`,command_sent="",device_response="无需操作"
  - 失败路径(transport_error/device_rejected)也写 audit:status=`failed`,failure_reason=err.Error()

- **D-05: batch audit = 每端口 1 条;operlog = 1 条汇总**
  - batch handler 遍历 `BatchResult.Succeeded/Failed/Skipped`,每个端口写 1 条 audit 行(共 N 条,N = 尝试过的端口数,不含 fail-fast 未尝试的)
  - operlog 写 1 条汇总:`OperTypeBatch`(=16,CONV-04),`oper_param` JSON 含 `{action, batch_size, succeeded_count, failed_count, skipped_count, device_id}`
  - audit↔operlog 关联:batch 的 N 条 audit 行 `oper_log_id` 全部指向同一条汇总 operlog(见 D-13 机制)

### 菜单 seed + 权限授权(PERM-03)

- **D-06: "端口配置" 菜单 = `menu_type='F'` 按钮权限**
  - `menu_name='端口配置'`,`parent_id` = "端口状态"菜单的 menu_id,**path='write'**,**perms='network:port:write'**,menu_type='F'
  - F 型不生成前端路由(routeGenerator 跳过),按钮靠 perms 字符串 gating — 符合 UI-01(写操作走现有端口列表页按钮 + Modal/Drawer,无独立页面)
  - 后端 API 路由组 `/network/ports/write/*` 与前端菜单 path 是独立概念,不混淆

- **D-07: 父菜单实际名 = "端口状态"(非 ROADMAP 写的 "端口管理")**
  - 实际 DB:`menu_name='端口状态'`,path='network/ports',component='network/ports/index'(scout 确认,见 archive/053_fix_menu_paths_unified.sql:185)
  - ROADMAP 的 "端口管理" 是笔误,以实际 DB 为准
  - operlog module 字符串仍用 `端口管理`(AUDIT-01 锁定,与父菜单名解耦 — module 只是 sys_oper_log.title 显示串)

- **D-08: 创建 `menu_grant_helpers.go::GrantNewMenuToRolesHavingParent`**
  - 新建 `internal/core/db/migrations/menu_grant_helpers.go`,提供:
    ```go
    func GrantNewMenuToRolesHavingParent(db *gorm.DB, parentMenuName string, newMenuID string) error
    ```
  - 实现(幂等):`INSERT INTO sys_role_menu SELECT rm.role_id, '<newMenuID>'::uuid FROM sys_role_menu rm JOIN sys_menu m ON rm.menu_id = m.id WHERE m.menu_name = '<parentMenuName>' ON CONFLICT DO NOTHING`
  - 只波及父已关联角色(antd 父子联动陷阱根治,memory `migration-grant-new-menu-precision-helper`)
  - `migration_202` seed 完菜单后调一行:`migrations.GrantNewMenuToRolesHavingParent(db, "端口状态", newMenuID)`
  - admin 走超管旁路自动可见,非 admin 角色由父菜单关联精准继承

### 路由结构(INFRA-02)

- **D-09: 子组 `/network/ports/write/*` + 组级鉴权 + kebab 命名**
  - 在现有 `ports := r.Group("/ports")` 下建 `write := ports.Group("/write")`
  - `write.Use(middleware.RequirePermissions([]string{string(permission.NetworkPortWrite)}))` 一处挂组级中间件
  - 6 端点 kebab 命名(与现有 `/list` `/collect` `/batch-delete` 同风格):
    - `POST /network/ports/write/shutdown`
    - `POST /network/ports/write/undo-shutdown`
    - `POST /network/ports/write/description`
    - `POST /network/ports/write/dot1x-enable`
    - `POST /network/ports/write/dot1x-disable`
    - `POST /network/ports/write/batch`
  - `network_router.go` 在 `SetupPortRouter` 后调 `SetupPortWriteRouter(ports, core)`

### 缓存 + 进度(INFRA-03 / BATCH-05)

- **D-10: `cache_keys.go` 仅定义常量,service 不写入(YAGNI MVP)**
  - 新建 `internal/services/portcollection/cache_keys.go`:
    ```go
    const CacheKeyPortWriteResult = "port:write:result:%s"      // %s = port_id
    const CacheKeyPortWriteBatch  = "port:write:batch:%s"       // %s = batch_id
    ```
  - MVP 阶段 Phase 51 service / Phase 52 handler 都不写入这两个 key(INFRA-03 字面满足"定义常量",不引入未消费的缓存写入)
  - Phase 53+ 前端若需"最近一次写结果"缓存再接入 CacheProvider

- **D-11: batch 端点同步阻塞(复用 Phase 51 BatchWritePorts 语义)**
  - handler 调 `service.BatchWritePorts(ctx, req, operator)` 阻塞到完(detached 30min ctx 已在 Phase 51 D-12 落地)
  - 返回最终 `BatchResult` JSON 给前端;前端用 loading spinner(Phase 53)
  - 运维并发低,5min 阻塞可接受(gin 不限单连接时长)

- **D-12: BATCH-05 进度反馈推到 v1.19.x**
  - 需 WebSocket(`internal/websocket/` 已存在)或 SSE 重构,会动 Phase 51 batch 同步契约 + 28 测试
  - v1.19 MVP 不做,记入 STATE.md deferred;UI-05 禁用刷新按钮已防 Enqueue 竞态

### audit↔operlog 关联(UI-04)

- **D-13: `audit.oper_log_id` FK → `sys_oper_log.id`;机制 = `WithOperID` RecordOption**
  - **约束**:`operlog.Record` 当前是 async(`operLogSvc.RecordAsync` fire-and-forget),不返回 oper_id
  - **关键发现**:`OperLog` embed `BaseTimeLine`,`BeforeCreate` 钩子 `if b.ID == "" { b.ID = uuid.New() }` — **预设 ID 会被保留**
  - **推荐机制**:
    1. 新增 `operlog.WithOperID(id string) RecordOption`(小增量,不破坏 25 常量回归锁)
    2. handler 预生成 `operID := uuid.New().String()`
    3. `operlog.Record(c, svc, db, "端口管理", operType, operlog.WithOperID(operID), operlog.WithOperParam(...))`
    4. 同一 `operID` 写入 audit 行 `oper_log_id` 列
  - **兜底(若不想动 operlog)**:handler 先写 audit 拿 audit_ids,再 `operlog.Record(..., WithOperParam({"audit_ids":[...]}))` 把 audit ids 嵌 operlog;UI-04 反向跳(operlog→audit)。次选。
  - `oper_log_id` 列**可空**(NULL allowed):防 operlog async 写入失败时 audit 仍能落库

### 迁移编号 + OperType + 权限常量

- **D-14: 迁移编号 = `migration_202_port_write_audit.go`**
  - 最新已存在 migration_201(Phase 48 component columns),本 phase 用 202
  - 注册到 `database.go` AutoMigrate + migration list
  - 内容:建表 SQL(或 GORM AutoMigrate `&models.PortWriteAudit{}`)+ 菜单 seed + `GrantNewMenuToRolesHavingParent` 调用

- **D-15: OperType 映射(已锁 STATE.md CONV-01..04,本 phase 落地)**
  - shutdown / undo_shutdown → `operlog.OperTypeStatus`(=10)
  - description → `operlog.OperTypeUpdate`(=2)
  - dot1x_enable / dot1x_disable → `operlog.OperTypeStatus`(=10)
  - batch(汇总)→ `operlog.OperTypeBatch`(=16)
  - operlog module 字符串 = `"端口管理"`(AUDIT-01)

- **D-16: `NetworkPortWrite` 权限常量**
  - `pkg/permission/config.go` 新增:`NetworkPortWrite PermissionCode = "network:port:write"`(在 `NetworkPortQuery` 旁)
  - 归入 Network 权限组
  - handler 引用 `permission.NetworkPortWrite`,不硬编码字符串

### Claude's Discretion

- **PortWriteAudit model 字段**:GORM tag 细节(`gorm:"type:jsonb"` / `gorm:"type:uuid"` / index tag)由 planner 按 `migration-sql-name-must-match-model` memory 惯例推导
- **handler 取 operator 方式**:沿用 `utils.GetUsername(c)`(CLAUDE.md 惯例)
- **单端口 handler 怎么拿 device_id/interface_name 拼 before_value**:D-02 的预 SELECT 已含这些字段,handler 直接用
- **audit 表是否加 `oper_log_id` 之外的其他 FK**:不加(避免迁移复杂度,device_id/port_id 软关联足够,query 时 JOIN)
- **`migration_202` 用 GORM AutoMigrate 还是手写 SQL**:倾向手写 `CREATE TABLE IF NOT EXISTS`(与 migration_201 风格一致,精确控制 JSONB 类型 + 索引),model 同步加 `gorm:"-:migration"` 防双重 ALTER(memory `gorm-automigrate-blocked-by-matview` 教训)
- **批量 handler 遍历 BatchResult 写 N 条 audit 时的事务**:每条 audit 独立 INSERT(失败不阻塞响应,applogger 告警);不用大事务(N 可能 50)
- **单端口 handler 的 request body struct**:5 个单端口方法各一个 `PortWriteRequest{PortID string; Description string(仅 description 用); Reason string(UI-02 操作原因,后端仅记录不校验)}`

</decisions>

<canonical_refs>
## Canonical References

**下游 agent (planner / researcher) 必须先读这些。**

### v1.19 锁定决策(本 phase 直接消费)
- `.planning/PROJECT.md` §"Current Milestone: v1.19" — 5 条 init 决策 + OperType 映射 + 权限隔离 + sys_port_write_audit 真相源
- `.planning/REQUIREMENTS.md` AUDIT-01/02/03/04 + PERM-01/02/03 + INFRA-01/02/03 + CONV-01..04 + PORT-01..05 + BATCH-01 段
- `.planning/ROADMAP.md` Phase 52 段 — 8 条 Success Criteria(注意 Success Criteria #3 写的 "端口管理" module 已 D-07 确认,#6 写的 "端口管理" 父菜单实为 "端口状态" 已 D-07 纠正)
- `.planning/STATE.md` §"Critical Pitfalls → Mitigation Map" — Pitfall #2/#4(本 phase audit 表落地)

### Phase 51 落地契约(本 phase 直接消费,零侵入)
- `.planning/phases/51-w2-portwriteservice-batch-orchestrator-mock-tests/51-CONTEXT.md` — D-10..D-18 service 契约(D-15 service 不调 operlog,handler 拥有审计)
- `.planning/phases/51-w2-portwriteservice-batch-orchestrator-mock-tests/51-01-SUMMARY.md` — PortWriteService 6 方法 + PortResult/BatchResult struct 形状
- `internal/services/portwrite/port_write_service.go` — `PortWriteService` interface + `PortResult{PortID,Action,Status,NoOp,CurrentState,Error,CommandSent}` + `BatchWriteRequest{DeviceID,Action,PortIDs,Description}` + `BatchResult{Succeeded,Failed,Skipped []PortResult}` + factory `NewPortWriteService(db, *device.DeviceExecutor, *services.DeviceInfoCollectionService)`
- `internal/services/portwrite/port_write_service.go` sentinel errors:`ErrBatchTooLarge` / `ErrEmptyBatch` / `ErrMixedDevices` / `ErrPortNotFound` / `ErrDeviceNotFound`(handler 翻译 HTTP 400/404/422)

### operlog 约定(强制 — CLAUDE.md)
- `CLAUDE.md` §"操作日志记录约定 (operlog convention)" — `operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "模块名", operType)` 调用模式 + 25 个 OperType 常量表 + 敏感端点用 `RecordWithBody`
- `internal/utils/operlog/operlog.go:215` — `func Record(c, operLogSvc, db, module, operType, opts ...RecordOptions)`(本 phase D-13 增 `WithOperID`)
- `internal/utils/operlog/operlog.go:190` — `WithOperParam` / `WithJsonResult` / `WithErrorMsg` / `WithStatus` 等现有 RecordOption(本 phase WithOperID 同模式)
- `internal/utils/operlog/regression_test.go` — 25 常量值 + 11 敏感关键词回归锁(本 phase WithOperID 是 additive,不破坏)
- `internal/api/v1/asset/reconciliation_handler.go` — 最近 operlog.Record handler 参考实现(Module 常量定义模式)
- `internal/api/v1/asset/fix_suggestion_handler.go` — operlog.Record + RecordWithBody 实战参考

### 权限中间件 + 路由注册
- `pkg/middleware/permission.go` — `RequirePermissions([]string)` + `RequirePermissionsWithQuery` 实现
- `pkg/permission/config.go:186` — `NetworkPortQuery PermissionCode = "network:port:query"`(本 phase D-16 在其旁加 `NetworkPortWrite`)
- `pkg/permission/config.go:197` — `NetworkPort PermissionCode = "network:port"`(父权限,参考)
- `internal/api/v1/network/network_router.go:205-220` — 现有 `/network/ports` 组 + `RequirePermissionsWithQuery` 模式(本 phase D-09 在其下加 `/write` 子组)
- `internal/api/v1/network/port_router.go` — 现有端口查询路由结构(kebab 命名参考:`/list` `/collect` `/collect-all` `/batch-delete`)

### 迁移 + 菜单 seed 模式
- `internal/core/db/migrations/migration_195_reconciliation_exception_rules_menu.go` — 菜单 seed + count-then-insert 幂等模式(本 phase D-06/D-08 参考,但 195 没授权,本 phase 用 helper 补授权)
- `internal/core/db/migrations/migration_201_phase48_component_columns.go` — 最新 migration 风格(手写 SQL + GORM model 同步)
- `internal/core/db/migrations/archive/053_fix_menu_paths_unified.sql:185` — "端口状态" 菜单实存证据(`menu_name='端口状态'` path=`network/ports` component=`network/ports/index`)
- memory `.claude/projects/.../migration-grant-new-menu-precision-helper.md` — helper 抽取建议 + antd 父子联动陷阱根因
- `internal/core/db/migrations/migration_169_reconciliation_dashboard_menu.go`(若存在)或 migration_195 — 父菜单查找 + 同级参照模式

### 数据模型
- `internal/models/log.go:6` — `OperLog` struct(embed `BaseTimeLine`)
- `internal/models/base.go:30-44` — `BaseTimeLine{ID,CreatedAt,UpdatedAt}` + `BeforeCreate` 钩子(`if ID=="" {ID=uuid.New()}` — D-13 关键:预设 ID 保留)
- `internal/models/device_port_status.go:31` — `DevicePortStatus{ID,DeviceID,InterfaceName,AdminStatus,Description,Dot1xEnabled,...}`(D-02 before_value 预 SELECT 源)
- `internal/services/oper_log_service.go:43` — `RecordAsync` 实现(async,不返回 oper_id — D-13 约束来源)

### 服务接口 + DI
- `internal/core/core.go` — `Core` struct 含 `OperLogService` / `DeviceExecutor` / `DeviceInfoCollectionService` / `DB` / `Cache`(handler DI 来源)
- `internal/services/operations/building_service.go` — 标准 handler-service-DI 模式参考

### response 包装 + request binding(CLAUDE.md)
- `pkg/response/` — `response.Success(c, data)` / `response.Error(c, code, msg)`
- CLAUDE.md §API Response Format — `{code:0, message, data, timestamp, request_id}`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `operlog.Record(c, svc, db, module, operType, opts...)` — 本 phase 6 handler 每个调一次;D-13 增 `WithOperID` option 后可拿 oper_id
- `middleware.RequirePermissions([]string)` — 组级鉴权,D-09 一处挂子组
- `permission.NetworkPortQuery`(config.go:186) — D-16 在其旁加 `NetworkPortWrite`
- `operlog.OperTypeStatus`(=10)/ `OperTypeUpdate`(=2)/ `OperTypeBatch`(=16) — D-15 直接用
- Phase 51 `PortWriteService` 全部 6 方法 — handler 直接调,零侵入
- `BaseTimeLine.BeforeCreate` 钩子 — D-13 oper_id 预设保留机制
- `utils.GetUsername(c)` / `utils.GetClientIP(c)` — operlog 已用,handler operator 来源

### Established Patterns
- **菜单 seed + helper 授权**:migration_195 count-then-insert + 本 phase 新增 helper(D-08)— antd 父子联动陷阱根治
- **handler 同步写 operlog + audit**:`response.Success` 前完成所有审计落库(CLAUDE.md operlog 强制约定)
- **手写 SQL migration + 同步 GORM model**:migration_201 风格,JSONB 类型精确控制
- **组级 RequirePermissions**:network_router.go:79/100/122 现有模式(credential/template/command 组)
- **sentinel error → HTTP 码翻译**:service 返 `ErrBatchTooLarge` → handler `response.Error(c, 400, ...)`

### Integration Points
- `internal/api/v1/network/network_router.go` — `SetupPortRouter` 后插 `SetupPortWriteRouter(ports, core)`
- `internal/core/db/migrations/database.go` — migration_202 注册到 AutoMigrate list + 迁移函数 map
- `pkg/permission/config.go` — `NetworkPortWrite` 常量加入(可能需同步 permission 注册表 / 前端权限定义,planner 确认)
- `internal/models/` — 新增 `PortWriteAudit` model(或并入现有 port 模型文件,planner 裁量)

</code_context>

<specifics>
## Specific Ideas

- **UI-04 "查看审计日志"跳转**:靠 `audit.oper_log_id` 精准跳 `sys_oper_log` 详情(D-13),不靠 timestamp 模糊匹配
- **NoOp/skipped 也审计**:Phase 51 D-15 锁定,本 phase handler 必须为 skipped 结果写 audit 行(status=skipped,command_sent=""),否则审计有缺口
- **before_value JSONB 结构固定**:`{"admin_status":"up|down","dot1x_enabled":bool,"description":"..."}` 三字段,description action 时 description 字段有意义,其他 action 时仍带当前描述(便于完整快照)
- **敏感字段脱敏**:operlog 会对含 password/secret/token 等关键词脱敏;`sys_port_write_audit.command_sent` 是未脱敏真相源(D-15),handler 不对 command_sent 脱敏 — 但 description 字段若含敏感词不触发(非敏感端点用 Record 非 RecordWithBody)
- **批量 audit 事务粒度**:每条独立 INSERT,不用大事务(N≤50,失败 applogger 告警不阻塞响应)

</specifics>

<deferred>
## Deferred Ideas

- **BATCH-05 批量进度反馈(WebSocket/SSE)** — v1.19.x:需重构 Phase 51 batch 同步契约为流式,动 28 测试;本 phase batch 端点同步阻塞 + 前端 spinner
- **`sys_port_write_audit` 详情查看 UI** — v1.19.x+:audit 表后端就绪,前端查看页后续补
- **audit 表 TTL / 归档策略** — v1.19.x+:audit 行会增长,后续加 cron 归档(参考 mac_history 归档模式)
- **跨设备批量(batch 内多 device)** — Phase 51 D-17 已锁 `ErrMixedDevices` 拒绝;未来若支持跨设备批量需重构 batch 编排
- **写命令前设备可达性预检(FUTURE-07)** — v1.19.x+:写前 1s ping,本 phase 不做
- **operlog `WithOperID` 是否同步进入 RecordBackground(cron 路径)** — 本 phase 仅 Record(HTTP 路径)需要;RecordBackground 若未来需要同机制再扩

### Reviewed Todos (not folded)
None — cross_reference_todos 未发现匹配本 phase 的 pending todo。

</deferred>

---

*Phase: 52-w3-router-handler-operlog-permission-migration*
*Context gathered: 2026-07-07*
