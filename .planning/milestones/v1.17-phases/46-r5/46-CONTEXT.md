# Phase 46: 半自动修复（可选） (R5) - Context

**Gathered:** 2026-07-03
**Status:** Ready for planning

<domain>
## Phase Boundary

**首次破例允许修复回写**——高置信度（默认 ≥0.9，可由 sys_config 配置）的 Type B（无主）异常生成"修复建议"，经人工确认后**仅**写入 `ops_asset.user_id` 一个字段；同时支持 7d 窗口内一键回滚 + 误修复率 <1% 监控 + 完整 operlog 审计链。完成 v1.17 milestone close。

**核心 scope**：
- 仅修 **Type B**（无主）；Type C/D/E/F 不进 R5
- 修复字段**仅 `ops_asset.user_id`**（不写 dept_id，不写 nowuser_name/deptname 等其他字段）
- 修复字段**仅来源于物理链路推导**（R1 RECON-02: `port_mac → info_point → workstation → user_id`），不采用 AD managed_by 作为建议源
- 必须经人工接受才落库（不接受"自动应用"语义）
- 触发器：`confidence_score ≥ threshold AND conflict_type='B' AND workorder_id IS NULL AND resolved_at IS NULL AND deleted_at IS NULL`

**显式不做**：
- 不修 Type D（未上线）— 修复需补充 machine_uptime/machine_ip，反验复杂、误修复率高
- 不修 dept_id / NowUserName / DeptName 等其他字段（最小变更、最大化回滚能力）
- 不使用 AD managed_by 作为建议源（参 [[ad-update-no-such-object-vs-lockout]] 已知不可靠）
- 不触发 DetectLayer3 同步重检（下次 cron 周期自然检出 Type A；走 R2 7d 静默期）
- 不联动 workorder 自动关闭（workorder 由运维在独立模块闭环）
- 不做批量接受（仅单条；防误操作）
- 不做自动覆盖业务表的"全字段修复"（observe-only 原则在 R5 局部破例，不泛化）
- 不引入 DB TRIGGER（参 [[user-prefers-code-fixes-no-db-triggers]]，走 Go service 层）

</domain>

<decisions>
## Implementation Decisions

### Area A — 修复字段范围与置信度门槛

- **D-A1（修复字段范围）**：仅修复 `ops_asset.user_id`，**不写 dept_id / NowUserName / DeptName**。理由：物理链路已锁定责任人 user_id（来自 port_mac → info_point → workstation → user_id 推导链，R1 RECON-02），dept_id 由 user_id JOIN sys_user 推导，最小变更 → 回滚粒度最小、误修复面最小。
- **D-A2（Type D 不进 R5）**：Type D（未上线）**不进**修复范围。理由：修复需补 machine_uptime / machine_ip，反验逻辑复杂、误修复高。UI 在 R5 提示"Type D 不适用 R5 自动修复"。
- **D-A3（置信度门槛可配置）**：默认 `0.9`，**新增 `sys_config: asset.reconciliation.fix.confidence_threshold`** 动态可调（参与 INFRA-02 字典/config seed）。理由：与 ROADMAP SC1 对齐 + 留运维根据实际误修复率现场调参余地。
- **D-A4（触发器限定）**：建议生成条件 = `confidence_score ≥ threshold AND conflict_type='B' AND workorder_id IS NULL AND deleted_at IS NULL AND resolved_at IS NULL`。理由：R2 已转单 B 类（workorder_id 非空）排除，避免重复创建 workorder + 重复提示。

### Area B — 建议-确认-应用-回滚数据模型

- **D-B1（独立表）**：新建 `sys_reconciliation_fix_suggestion` 表，与 `sys_data_reconciliation` 1:N 关系（异常可生成多轮建议）。**不**扩展 `sys_data_reconciliation` 加字段。理由：状态机独立、可追踪多轮建议、审阅审计链完整、不侵占 R1-R4 既有 reconciliation 表。
- **D-B2（状态机 6 状态）**：`fix_status` 字段 6 值：
  - `pending` — 已生成，等待人工确认
  - `accepted` — 运维点击接受，未落库
  - `rejected` — 运维点击拒绝，不落库
  - `applied` — 已成功写入 `ops_asset.user_id`
  - `rolled_back` — 已回滚 user_id
  - `failed` — 应用失败（写库异常、回写异常）
- **D-B3（1 对多版本化）**：`sys_reconciliation_fix_suggestion.exception_id` 为索引 FK（**不**设唯一约束），一对多关系。旧记录加 `superseded_at` 字段标记不为当前生效建议。
- **D-B4（并发控制 = 乐观锁 + 部分唯一索引）**：API 层以事务 + `WHERE fix_status='pending'` 作条件 UPDATE，`affected_rows=1` 才视为成功接受。DB 层贴**部分唯一索引**：
  ```sql
  CREATE UNIQUE INDEX uniq_fix_suggestion_pending_per_exception
    ON sys_reconciliation_fix_suggestion (exception_id)
    WHERE fix_status = 'pending' AND superseded_at IS NULL AND deleted_at IS NULL;
  ```
  与 R1 `uniq_recon_asset_type_open` 同模式（部分唯一 + WHERE 状态条件），DB 层绝对拦截重叠接受。

### Area C — 回滚机制 + 误修复监控 + 缓存/静默期联动

- **D-C1（回滚粒度 = 仅恢复 user_id）**：因 D-A1 锁定仅修复 user_id，故回滚仅恢复 `ops_asset.user_id` 为 `pre_fix_user_id`（在 `applied` 时持久化在 suggestion 表）。
- **D-C2（回滚窗口期 = 固定 7d）**：suggestion 表加 `rollback_window_until = applied_at + INTERVAL '7 day'`。超过窗口期后 UI 不显示"回滚"按钮（DB 仍允许强制回滚，但前端禁用）。理由：与 R2 7d 静默期语义一致；运维需在 7d 内发现+回滚误修复。
- **D-C3（回滚强写 operlog）**：rollback 动作强写 operlog，使用 `operlog.OperTypeReset=11`（已存于 25 OperType 常量集，语义"密码/密钥重置" — 接近"恢复到原值"）。记录 rollback 前 user_id / 后 user_id / rollback_reason / suggestion_id。理由：与 CLAUDE.md "操作日志记录约定 强制" 一致。
- **D-C4（修复后启用 7d 静默期 + 缓存失效）**：
  - applied 后同 `(asset_id, conflict_type)` **自动进入 R2 7d 静默期**（复用 `sys_data_reconciliation.last_resolved_at` MV 扩展机制，**不**新建异常）
  - 主动调用 `invalidate_workstation_health(asset_id)`（R4 D-A4-04 已建 helper）使工位详情页缓存立即失效
  - 不触发 DetectLayer3 同步重检（下个 cron 周期自然检出 Type A，进入 auto-resolve 路径）
- **D-C5（误修复率监控 = 回滚/应用比率 + 7d 滑动窗口）**：
  - 新增端点 `GET /asset/reconciliation/fix-suggestion/stats` 返回 7d 滑动窗口统计：`applied / rolled_back / rejected / pending / failed` 计数
  - 误修复率 = `rolled_back / applied`（applied=0 时返回 0 不告警）
  - 超过 `sys_config: asset.reconciliation.fix.mis_fix_threshold`（默认 0.01）→ SysNotice 告警（"资产对账误修复率超阈"）
  - 与 R3 降噪基线对比端点同模式

### Area D — 人工确认 UI 形态 + 建议展示密度

- **D-D1（独立页面）**：新建独立页面 `/asset/reconciliation/fix-suggestion`，显示全 6 状态建议（默认筛选 `pending`）。**不**复用 R4 ReconciliationDrawer。理由：与 ROADMAP SC2 "一键接受/拒绝/修改建议" 独立交互面需求匹配；R4 抽屉以冲突摘要为主，添 Tab 与原契约冲突。
- **D-D2（紧凑行 + 点击展开详情）**：列表默认列 = `asset_code / 现 ops_asset.user_id / 建议 user_id / confidence_score / conflict_type / created_at / fix_status`。点击行 → 弹 antd Drawer 显示 raw_snapshot 三路信息（physical / declared / ad）与冲突原因（reason）。**不**默认全字段展开（带宽、屏幕空间）。
- **D-D3（仅单条接受）**：**不**提供批量接受按钮（即使按 dept_id 或 user_id 聚合）。理由：与 R5 "需人工确认" 语义一致；批量点击易忽略个体差异 → 误修复率上升。
- **D-D4（默认排序 + 部门/状态筛选）**：列表默认 `confidence_score DESC + created_at DESC`。筛选维度：`responsible_dept_id`（JOIN ops_asset）+ `fix_status`。复用 Phase 13 `BaseListRequest + ApplySort` 白名单（防 MaxPageSize=100 钳制，参 [[stat-cards-from-list-length-capped-at-100]]）。

### Claude's Discretion

下列实现细节由 planner/researcher 在 plan-phase 自决：

- 修复建议生成的触发时机（DetectLayer3 同步生成 vs 列表查询时 lazy 生成 vs 独立 cron）
- `sys_reconciliation_fix_suggestion` 表的精确字段列表（除已锁定 status / exception_id / suggested_user_id / pre_fix_user_id / confidence_score / reason / created_by / created_at / accepted_at / accepted_by / applied_at / rolled_back_at / rollback_reason / rollback_window_until / superseded_at 之外，可加版本号、IP 等上下文）
- 拒绝建议时是否必填 `rejection_reason`（建议必填；运维拒因是审计必要项）
- 修改建议功能的 UI 形态（弹窗编辑 vs 跳转编辑页）
- sys_reconciliation_fix_suggestion 是否新建独立 migration 或合并到 `migration_NNN_reconciliation_*` 系列
- 检测 raw_snapshot 是否需补字段记录"修复前完整 ops_asset 状态"（D-C1 锁定仅恢复 user_id，但留扩展空间）
- 修复建议是否在 R4 ReconciliationDrawer"冲突摘要"Tab 加跳转链接到 `/asset/reconciliation/fix-suggestion?exception_id=xxx`
- INFRA-02 中 `asset.reconciliation.fix.confidence_threshold` / `mis_fix_threshold` 的默认值 + config seed 命名

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents (researcher / planner) MUST read these before planning or implementing.**

### 前序 R1-R4 上下文（必读，R5 直接构建于其上）
- `.planning/phases/42-r1/42-CONTEXT.md` — R1 全部 18 决策：sys_data_reconciliation schema、partial uniqueIndex `uniq_recon_asset_type_open`、DetectLayer3 循环结构、cron sys_job 模式、operlog 边界、RECON-02 物理链路推导链
- `.planning/phases/43-r2/43-CONTEXT.md` — R2 转单 cron + workorder_id 字段 + 7d 静默期 + 24h 节流 + WS/SysNotice + resolve API（D-C4 静默期复用点）
- `.planning/phases/44-ip-r3/44-CONTEXT.md` — R3 例外引擎 + Layer 3.5 + applied_actions 写入语义（D-C5 监控模式参考）
- `.planning/phases/45-r4/45-CONTEXT.md` — R4 HealthCard/Drawer 集成 + D-A4-04 `invalidate_workstation_health(wsID)` helper（D-C4 缓存失效复用点）

### 已闭合项（不重实施）
- R1 信息点 port_id 漂移修复（`migration_188_fix_info_points_port_id_drift.go`）— 已闭合
- R4 port_status FK（`migration_183_add_port_status_device_fk.go`）— 已闭合
- Phase 47 R2/R3 UPSERT + R5 parseRuijiePortSecurityLine canonical MAC 校验（`migration_194`）— 已闭合
- Phase 47 R5 数据清理 migration — 已闭合

### 架构与策略
- `.planning/notes/asset-reconciliation-strategy.md` — v0.3 架构 + v0.4 复用审计 + v0.5 字段名调整；§14 R5 阈值（confidence ≥0.9）
- `.planning/notes/260627-reconciliation-reuse-audit.md` — F1-F7 必补项 + P1-P4 部分复用项
- `.planning/notes/260627-cross-module-permission.md` — 跨模块权限边界（R5 走 service 层调用，无需 HTTP）
- `.planning/seeds/asset-reconciliation-v1.17.md` — v1.17 阶段种子
- `.planning/todos/pending/v1.17-reconciliation-decisions.md` — v1.17 决策点追踪（R5 相关项已被本 CONTEXT 锁定）

### Roadmap 与 Requirements
- `.planning/ROADMAP.md` Phase 46 段 — 2 plans（46-01: 建议生成 + UI；46-02: 回滚 + 误修复监控）+ 7 条 success criteria
- `.planning/REQUIREMENTS.md` v1.17 — RECON/AUDIT/INFRA 系列（R5 不新增 requirement，沿用 RECON-01~07 + AUDIT-01/02 + INFRA-02）

### 项目级 CLAUDE.md（强约束）
- `CLAUDE.md` "操作日志记录约定 (operlog convention) — 强制" — 11 关键词 + 25 OperType 常量（含 OperTypeUpdate=2, OperTypeReset=11, OperTypeApprove=22, OperTypeReject=23）
- `CLAUDE.md` "Status Value Convention" — 0=启用 1=停用（fix_status 6 值不在此惯例，自定义字符串枚举）
- `CLAUDE.md` "Migration 编写模板" — 编号递增 + AutoMigrate 注册
- `CLAUDE.md` "Cache Key Prefix Handling" — Redis `xingran:` 前缀处理
- `CLAUDE.md` "API Response Format" — code:0 成功、1001/4001 等错误码
- `CLAUDE.md` "前端 useEffect Dependencies" — R5 新增 React 组件必须稳定 deps
- `CLAUDE.md` "Pagination Convention" — `current / pageSize` 复用

### 项目记忆（规划时必查，已在 MEMORY.md）
- `stat-cards-from-list-length-capped-at-100` — D-D4 排序/筛选必须用专用端点，不依赖 list.length
- `xingran-server-side-sort-infra` — Phase 13 BaseListRequest + ApplySort 白名单（D-D4 复用）
- `xingran-perm-namespace-split-readonly-page` — R5 新增权限 `asset:reconciliation:fix:*`（list/accept/reject/rollback），命名空间与 R1 一致
- `user-prefers-code-fixes-no-db-triggers` — 禁 DB TRIGGER（修复回走 Go service 层 + 部分唯一索引）
- `xingran-info-point-port-id-varchar` — ops_asset.user_id / dept_id 是 varchar size:64（任何 re-point/UPDATE SQL 不能 `?::uuid` 强转）
- `ad-update-no-such-object-vs-lockout` — 不采用 AD managed_by 作建议源（D-A1 锁定物理链路）
- `workstation-ad-device-managedby-vs-description` — 物理链路 user_id 推导不依赖 AD managed_by
- `xingran-migrations-no-sql-autoloader` — 部分唯一索引 + INFRA-02 config seed 必须用 `migration_NNN_*.go` 显式调用 + AutoMigrate
- `xingran-gorm-sql-constraint-naming-conflict` — uniqueIndex 命名 `uni_*_*`（D-B4 部分唯一索引沿用此规则）
- `migration-sql-name-must-match-model` — UPSERT/SET 列名以实际 DB schema 为准
- `GORM AutoMigrate 被 PG 物化视图阻塞` — sys_reconciliation_fix_suggestion 建表需错开 reconciliation_normalized MV 重构窗口
- `pg-any-null-three-valued-logic-trap` — applied_actions 数组类字段 ANY() 过滤必加 IS NULL OR 兜底（若新代码有类似过滤）

### 现有代码参考（实施时查）
- `internal/models/reconciliation.go:22-60` — SysDataReconciliation 字段（exception_id 来自此表 id + workorder_id 字段）
- `internal/models/asset.go:86-87` — ops_asset.UserID / DeptID varchar size:64（D-A1 仅写 UserID）
- `internal/services/asset/reconciliation_detection.go:DetectLayer3` — Layer 3 循环（D-A4 触发条件查询此函数结果）
- `internal/services/asset/cache_keys.go:GetReconciliationHealthByWorkstationKey` + `invalidate_workstation_health(wsID)` — D-C4 缓存失效复用
- `internal/services/asset/reconciliation_exception_matcher.go:MatchException()` — 可参考（不直接复用，R5 不涉及例外规则）
- `internal/services/workorder/base.go:Create(ctx, req, submitterID)` — 已存在（D-A4 不联动 workorder 自动关闭，故仅参考）
- `internal/services/system/config_service.go:GetByKey` — D-A3/D-C5 sys_config 读取模式
- `internal/api/v1/asset/reconciliation_router.go` — R1 路由注册位置（R5 新增 `/fix-suggestion/*` 路由）
- `internal/api/v1/asset/reconciliation_handler.go:14-19` — `ModuleReconciliation = "资产对账"`（R5 新增 `ModuleReconciliationFixSuggestion = "资产对账-修复建议"`）
- `internal/services/asset/reconciliation_statistics.go` — R1 6 个 Statistics 端点（D-C5 fix-suggestion/stats 端点参考同模式）
- `internal/websocket/notice_hub.go` — 可选：D-C5 SysNotice 写入
- `internal/core/db/migrations/migration_168_reconciliation_tables.go` — partial uniqueIndex `uniq_recon_asset_type_open` 原点（D-B4 复用同模式）
- `internal/core/db/migrations/migration_170_fix_asset_list_menu_path.go` — migration 模板参考
- `internal/services/operations/excel_service.go` + `excel_config.go` — D-C5 stats 端点可选支持 Excel 导出（非必需）
- `xingran-react-frontend/src/lib/queryKeys.ts` — INFRA-05 已注册 reconciliation.*；R5 新增 `fixSuggestion.{list,detail,stats}`
- `xingran-react-frontend/src/lib/opsApi.ts` — reconciliationApi factory 模式（R5 新增 fixSuggestionApi 参考）
- `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` — R1 异常列表页（D-D1/D-D2 UI 模式参考：紧凑行 + Drawer 详情）
- `xingran-react-frontend/src/hooks/useDict.ts` — 字典 hook（D-D4 筛选器复用 `asset_reconciliation_conflict_type`）
- `internal/utils/operlog/regression_test.go` — OperType 常量集 25 个（D-C3 用 OperTypeReset=11 已锁定）

</canonical_refs>


## Existing Code Insights

### Reusable Assets
- `internal/services/asset/cache_keys.go:invalidate_workstation_health(wsID)` — R4 D-A4-04 已建 helper，D-C4 复用
- `internal/services/asset/reconciliation_statistics.go` — Statistics 端点模式（Summary / ByConflictType / HealthTrend），D-C5 fix-suggestion/stats 复用
- `internal/services/system/config_service.go:GetByKey(key)` — D-A3 / D-C5 sys_config 读取
- `internal/core/db/migrations/migration_168_reconciliation_tables.go:uniq_recon_asset_type_open` — partial uniqueIndex 原点，D-B4 uniq_fix_suggestion_pending_per_exception 复用同模式
- `internal/utils/operlog` — D-C3 强 operlog 集成路径，OperTypeReset=11 已就绪
- `internal/services/asset/reconciliation_service.go:ListExceptions` + `GetByID` — D-D1 列表 + 详情 handler 参考
- `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` — R1 异常列表页（D-D1/D-D2 UI 模式）
- `xingran-react-frontend/src/hooks/useDict.ts` — 字典 hook（D-D4 筛选器复用）
- `xingran-react-frontend/src/components/reconciliation/` — R4 新建组件目录（D-D2 Drawer 复用同 antd Modal/Drawer 模式）

### Established Patterns
- **Handler-Service Pattern**：interface + 私有 impl + 构造函数（CLAUDE.md 范例，R5 `FixSuggestionService` 复用）
- **GORM 部分唯一索引**：`(column) WHERE condition` 模式（D-B4 与 R1 uniq_recon_asset_type_open 同形）
- **operlog.Record 强制约定**：D-A4 accept/reject + D-C3 rollback + D-C5 stats 告警均走 operlog
- **Cache Key Helper**：D-C4 invalidate_xxx_key 函数模式
- **统计专用 COUNT 端点**：D-C5 /stats 走 COUNT，不依赖 list.length（防 MaxPageSize 钳制）
- **R2 7d 静默期自动生效**：D-C4 不需新建逻辑，DetectLayer3 内置 WHERE 条件已覆盖
- **R4 缓存失效主动调用**：D-C4 复用 `invalidate_workstation_health(asset_id)` helper
- **migration 编号递增**：已有 195（Phase 47 R5 数据清理），接 196/197（推荐拆 2：建议表 + 部分唯一索引/CHECK 约束）
- **React Query useEffect 依赖**：R5 新增 React Query hooks 必须稳定 deps（CLAUDE.md 强制）

### Integration Points
- `internal/core/db/migrations/migration_NNN_create_fix_suggestion_table.go`（新建）— sys_reconciliation_fix_suggestion 表 DDL
- `internal/core/db/migrations/migration_NNN_reconciliation_fix_gini.go`（新建）— D-B4 部分唯一索引 + 可选 CHECK 约束
- `internal/core/db/migrations/migration_NNN_reconciliation_fix_config_seeds.go`（新建，可选合并）— D-A3 / D-C5 sys_config seed
- `internal/services/asset/fix_suggestion_service.go`（新建）— 业务逻辑（生成建议 + 接受 + 拒绝 + 应用 + 回滚 + stats）
- `internal/api/v1/asset/fix_suggestion_handler.go`（新建）— HTTP handler
- `internal/api/v1/asset/fix_suggestion_router.go`（新建）— 路由注册（POST `/fix-suggestion/list` `/fix-suggestion/:id/accept` `/fix-suggestion/:id/reject` `/fix-suggestion/:id/rollback` `/fix-suggestion/stats`）
- `internal/services/asset/fix_suggestion_generator.go`（新建，可选独立）— D-A4 触发器（DetectLayer3 同步生成 vs 独立 cron vs lazy）
- `xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/index.tsx`（新建）— D-D1 独立页面
- `xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx`（新建）— D-D2 详情 Drawer
- `xingran-react-frontend/src/lib/queryKeys.ts` — 新增 `reconciliation.fixSuggestion.{list,detail,stats}`
- `xingran-react-frontend/src/lib/opsApi.ts` — 新增 `fixSuggestionApi` 模块
- `internal/api/v1/asset/reconciliation_handler.go` — 加 `ModuleReconciliationFixSuggestion` 常量

</code_context>

<specifics>
## Specific Ideas

- **D-B4 部分唯一索引 SQL 模板**（与 R1 uniq_recon_asset_type_open 同形）：
  ```sql
  CREATE UNIQUE INDEX uniq_fix_suggestion_pending_per_exception
    ON sys_reconciliation_fix_suggestion (exception_id)
    WHERE fix_status = 'pending' AND superseded_at IS NULL AND deleted_at IS NULL;
  ```
- **D-D1 页面顶部 KPI**（参考 R1 dashboard 顶部 5 KPI 卡片）：
  - 待处理建议数（pending 计数）
  - 7d 应用数（applied in 7d）
  - 7d 回滚数（rolled_back in 7d）
  - 7d 误修复率（rolled_back / applied）
  - 7d 拒绝数（rejected in 7d）
- **D-D2 Drawer 三 Tab**（参考 R4 ReconciliationDrawer 三 Tab 契约）：
  - 冲突摘要：raw_snapshot 三路物理/声明/AD 数据 + signal 标志位
  - 修复详情：当前 ops_asset.user_id vs 建议 user_id vs applied 后 user_id（时间轴）
  - 历史变更：该异常的 sys_reconciliation_fix_suggestion 全状态记录（rejected/accepted/applied/rolled_back）+ reason
- **D-C5 stats 端点响应结构**：
  ```go
  type FixSuggestionStatsResponse struct {
      WindowDays         int     `json:"windowDays"`         // 默认 7
      Pending            int     `json:"pending"`
      Accepted           int     `json:"accepted"`
      Rejected           int     `json:"rejected"`
      Applied            int     `json:"applied"`
      RolledBack         int     `json:"rolledBack"`
      Failed             int     `json:"failed"`
      MisFixRate         float64 `json:"misFixRate"`         // rolledBack / applied, applied=0 时 0
      Threshold          float64 `json:"threshold"`          // 当前 sys_config 阈值
      ThresholdBreached  bool    `json:"thresholdBreached"`  // misFixRate > threshold
      TrendSeries        []TrendPoint `json:"trendSeries"`   // 7d 滑动趋势（可选）
  }
  ```
- **operlog 字段示例（D-C3 rollback）**：
  ```
  Module:    资产对账-修复建议
  OperType:  OperTypeReset=11 (恢复到原值)
  Title:     "回滚修复建议 #{suggestion_id}: asset {asset_code} user_id {pre} -> {post}"
  Body:      rollback_reason + raw_snapshot diff
  ```
- **INFRA-02 config seed 新增**：
  - `asset.reconciliation.fix.confidence_threshold` = 0.9 (float)
  - `asset.reconciliation.fix.mis_fix_threshold` = 0.01 (float)
  - `asset.reconciliation.fix.rollback_window_days` = 7 (int, 与 D-C2 互锁)
  - `asset.reconciliation.fix.enabled` = 1 (int, 0=暂停生成建议，便于紧急熔断)
- **权限命名**：`asset:reconciliation:fix:*` 系列
  - `asset:reconciliation:fix:list` — 查看列表
  - `asset:reconciliation:fix:accept` — 接受
  - `asset:reconciliation:fix:reject` — 拒绝
  - `asset:reconciliation:fix:rollback` — 回滚
  - `asset:reconciliation:fix:stats` — 查看统计

</specifics>

<deferred>
## Deferred Ideas

下列决策**显式推后**到后续 phase，R5 不实现：

### 显式不做（不允许 scope creep）
- Type D / Type C / Type E / Type F 修复 — 仅 Type B（无主）进 R5
- 修复 dept_id / NowUserName / DeptName / machine_uptime / machine_ip — 仅 user_id
- 批量接受 — 仅单条
- 自动触发 DetectLayer3 同步重检 — 下个 cron 周期自然检出 Type A
- 联动 workorder 自动关闭 — workorder 模块独立闭环
- DB TRIGGER 路线 — 走 Go service 层（参 [[user-prefers-code-fixes-no-db-triggers]]）
- AD managed_by 作为修复建议源 — 仅物理链路（R1 RECON-02）
- "全字段修复"自动覆盖 — observe-only 原则在 R5 局部破例，不泛化

### 后续 phase 候选（如有需要）
- v1.18 R5+: Type C / Type F 等其他类型的修复（待 Type B 误修复率经验积累后启动）
- v1.18 R5+: 修改建议功能 UI（编辑 suggested_user_id 后再 accept）
- v1.18 R5+: 修复建议的 Excel 导出（参 R3 例外规则 Excel 导入导出模式）
- v1.18 R5+: 自动接受（高 confidence 且 severity=critical 且部门白名单）— 待 R5 误修复率经验

### Reviewed Todos (not folded)
`cross_reference_todos` 命中（与 R2-R4 审阅结论一致）：
- `v1.17-reconciliation-decisions.md`（v1.17 决策点追踪与待办清单，`resolves_phase: 42`）—— R5 相关项已被本 CONTEXT D-A1~D4 / D-B1~D4 / D-C1~D5 / D-D1~D4 锁定；T1-T30 中 R5 专项已被本 phase 覆盖，不重复跟踪
- `operlog-exclude-paths.md`（operlog.exclude_paths 配置驱动白名单，解决 RPA 心跳日志污染）— Phase 35 范围，与 R5 无关

</deferred>

---

*Phase: 46-半自动修复（可选） (R5)*
*Context gathered: 2026-07-03*