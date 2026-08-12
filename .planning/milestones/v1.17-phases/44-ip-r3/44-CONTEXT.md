# Phase 44: 置信度评分 + IP 段例外 (R3) - Context

**Gathered:** 2026-06-28
**Status:** Ready for planning

<domain>
## Phase Boundary

将 v1.17 资产对账的**告警通路接入 IP 段例外规则引擎**——CIDR 例外规则 CRUD（admin 管理页）+ Layer 3.5 例外过滤拦截（嵌入 Phase 42 DetectLayer3 循环）+ 命中测试工具（EXCEPTION-04）+ Excel 导入导出例外规则 + 过期例外自动停用 cron + 降噪效果量化验证。**目标：告警量比 R2 末期下降 ≥60%**（ROADMAP success criteria 8）。

**不含**：工位详情页整合 HealthCard/Drawer（R4/Phase 45）、半自动修复（R5/Phase 46）、钉钉/邮件告警通道。

**命名澄清（重要）**：Phase 44 标题"置信度评分"已在 **Phase 42 (RECON-03, `confidence_score` 字段 + Plan 42-02 Layer 3 引擎)** 落地。R3 真正交付的是 **IP 段例外规则引擎 + 降噪**（与 strategy.md §10 路线图 R3 定义一致）。Planner 不要在 R3 重复实现评分函数。

**已锁定的高层决策**（沿用 v0.2-v0.3 / Phase 42-43 / ROADMAP，不重复展开）：
- 策略：Observe-only；命中例外**仍写** `sys_data_reconciliation` + 标记 `exception_rule_id` + `applied_actions`（D4 审计要求）
- 多规则 actions 取并集（D5）；支持临时例外 `expires_at`（D6）
- 5 actions 语义固定：`no_alert` / `no_notice` / `no_workorder` / `skip_severity` / `silence`
- 例外**不影响** `confidence_score` / `raw_snapshot`（按真实数据计算/冻结）
- 权限命名 `asset:reconciliation:exception:*`（list/create/update/delete/test）；API 路由 `/asset/reconciliation/exception-rule/*`
- 表结构已建（`migration_168`，但**缺 GiST 索引 + CHECK 约束**，R3 必补）
- `default_expiry_days = 30`（`sys_config` seed，INFRA-02 已就位）
- Excel 复用 `ExcelConfig` 模式（reuse-audit P3 已给骨架）
- cron 走 `sys_job` 表 + 单 taskType `"reconciliation"` 分发；`cleanupExpiredExceptions` case 已 placeholder（reconciliation_tasks.go）
- operlog 强制约定；module 常量 R3 新增 `ModuleReconciliationExceptionRule = "资产对账-例外规则"`（Phase 42 D-16 锁定 R2-R4 补）

**与 Phase 43 R2 的边界差异**：
- R3 新增：Layer 3.5 例外过滤（DetectLayer3 循环内插入点 `reconciliation_detection.go:262` 前）+ 例外规则 CRUD admin 页 + 命中测试端点 + Excel 导入导出 + 过期清理 cron 真实实现 + 降噪基线/对比端点
- R3 不改动：R2 的转单 cron（仅 SQL 加 `applied_actions` 过滤条件）/ WS 推送 / SysNotice / resolve API（这些读 `applied_actions` 决定是否执行）
- R3 依赖 R2 已有：`applied_actions TEXT[]` 字段已建（reconciliation.go:50）、转单 cron 已注册、7d 静默期/24h 节流 guard 已在位

</domain>

<decisions>
## Implementation Decisions

### Layer 3.5 拦截语义（Area 1）

- **D-R3-A1-01（silence 记录形态）**：`silence` action 命中**仍写** `sys_data_reconciliation`（带 `exception_rule_id` + `applied_actions=[silence]`），但**异常列表默认过滤掉 silence 记录**，仅 operlog/审计可查。
  - 统一 D4 审计要求（事后可溯源被静默的资产）+ 符合 strategy §4.2 "仅审计可查"语义。
  - 解决 strategy silence="不记录" 与 D4="命中仍记录" 的字面冲突——silence 不影响"写表"，只影响"是否在异常列表展示 + 是否走告警/工单通路"。
  - 异常列表是否提供"显示已静默"开关 → 留 planner discretion。

- **D-R3-A1-02（例外过滤执行位置）**：例外过滤**集中在 DetectLayer3 循环内（Layer 3.5）**，一次性匹配后写 `exception_rule_id` + `applied_actions` 到 `sys_data_reconciliation`。
  - 下游通路（R2 WS 推送 / SysNotice / 转单 cron）**读 `applied_actions` 决定是否执行**，不各自重查例外表。
  - 转单 cron（`createWorkorderCritical`/`createWorkorderHigh`）SQL 加条件：`AND 'no_workorder' != ANY(applied_actions)`。
  - 单一真相源，例外匹配只做一次，语义一致。

- **D-R3-A1-03（例外匹配性能架构）**：DetectLayer3 循环**前预加载所有 active 例外规则到内存**，每条资产用 Go `net.ParseCIDR` + `ipNet.Contains` 匹配（参考 `internal/middleware/apikey.go:126` 现成模式）。
  - **GiST 索引留给命中测试工具的单点查询**（输入 IP 返回命中规则），不用于批量检测循环。
  - 循环内零 DB 查询，性能稳定（active 规则十几条，资产几千条）。
  - 例外规则变更后，下个 cron 周期重新加载（无缓存陈旧问题）。

### 多规则合并语义（Area 2）

- **D-R3-A2-01（severity_override 多规则冲突）**：多条规则命中同一 (asset, conflict_type) 且各带不同 `severity_override` 时，**取最低（最宽松）严重级**。
  - 例：规则A override=low + 规则B override=medium → 最终 `low`。
  - 降噪最大化，与 D5 "多规则取并集"精神一致。
  - 单规则命中时直接用其 override。

- **D-R3-A2-02（skip_severity 语义）**：`skip_severity` = **当前 severity 降一级**（critical→high→medium→low，low 不再降）。
  - 仍记录、仍走通路，但按降级后 severity 处理（影响 SLA / 是否触发 critical 即时通知）。
  - 与 severity_override 协作：**先 skip_severity 降级，再 severity_override 覆盖**（取更宽，即与 A2-01 取最低一致）。
  - 明确 strategy §4.2 "跳过当前告警级别仍记录但不升级" 的模糊表述。

- **D-R3-A2-03（合并效果可视化）**：命中测试工具 + 规则详情页采用**命中规则列表 + 顶部合并结果卡片**形态。
  - 命中规则列表：每条显示 name / IP范围 / actions / override / scope / expires / reason。
  - 顶部合并结果卡片：最终 actions 并集 + 最终 severity + 是否 silence。
  - 满足 ROADMAP success criteria 4 "多条规则取并集（合并效果可视化）"。

### 作用域与匹配逻辑（Area 3）

- **D-R3-A3-01（ScopeType 维度 + IP 协作）**：**沿用现有代码 `global/dept/user`** 维度（不改 schema，不回退到 strategy v0.3 的 `building/floor`）。
  - `global` 规则：仅 IP CIDR 匹配即生效。
  - `dept`/`user` 规则：需「IP CIDR 命中 **AND** 资产责任人 user_id ∈ 该 dept/user」**双条件**才生效。
  - Layer 3.5 可评估双条件（资产在 MV 里有 `physical_user_id` / `responsible_user_id`）。
  - `dept` scope 是否递归子部门 → 留 planner discretion（参照 `sys_dept` ancestors 递归模式）。
  - 与"责任人"维度对齐，"某部门所有资产豁免"场景自然。

- **D-R3-A3-02（空 conflict_types 语义）**：`conflict_types` 为空数组/null 时，该规则**匹配全部 B-F 冲突类型**（A 不入主表，天然排除）。
  - 简化配置（办公网段"全类型豁免"是常见场景）。
  - planner 在 service 层 enforce 此语义（字段无 DEFAULT，reconciliation.go:85）。

- **D-R3-A3-03（命中测试输入 + dept/user 评估）**：命中测试工具输入 **IP/CIDR（必填）+ 可选 user_id/dept_id**。
  - 不填 user/dept 时：`dept`/`user` scope 规则标记"需指定 user/dept 才能评估"，仅 `global` 规则参与合并。
  - 覆盖"单 IP 快速测试" + "精确责任人评估"两种场景（EXCEPTION-04）。

### 降噪验证 + 工具（Area 4）

- **D-R3-A4-01（≥60% 降噪验证方法）**：**基线快照 + 对比端点**。
  - 运维在 R3 例外规则生效**前**手动触发"记录基线"操作，把当前异常总数 / 工单总数 / critical 数按时间窗口快照存 `sys_config`（JSON）。
  - R3 例外生效后：dashboard 加"降噪效果"卡片 + 新增对比端点返回"基线 vs 当前"下降百分比。
  - 量化验证 ROADMAP success criteria 8（告警量比 R2 末期下降 ≥60%），避免肉眼对比主观判定。

- **D-R3-A4-02（Excel 字段映射）**：**逗号分隔 + 名称→UUID 匹配**。
  - 列：`name` / `ip_range`(CIDR文本) / `conflict_types`(逗号分隔如 `B,C,D`) / `exception_actions`(逗号分隔如 `no_alert,no_notice`) / `severity_override` / `scope_type` / `scope_name`(部门/用户名称→匹配UUID) / `expires_at`(日期) / `reason`。
  - 复用 building 导入的"名称→UUID"解析模式（参考 xingran-excel-import 系列项目记忆）。
  - reuse-audit P3 已给 `reconciliationExceptionRule` ExcelConfig 骨架，planner 按本映射细化 Columns。

- **D-R3-A4-03（过期清理行为）**：**软停用 + 保留外键**。
  - 到期 cron（`cleanupExpiredExceptions`）`UPDATE is_active=1`（停用），**不删记录**。
  - 历史 `sys_data_reconciliation.exception_rule_id` 仍指向有效（虽停用）记录，**审计链不断**（满足 AUDIT-02 溯源）。
  - 停用规则在 admin 列表标灰，可重新启用 / 改期。

### Claude's Discretion

下列 R3 细节由 planner/researcher 在 plan-phase 自决（均属技术实现或已在上游文档锁定）：
- **GiST 索引定义**：`CREATE INDEX ... USING gist (ip_range inet_ops) WHERE is_active=0 AND deleted_at IS NULL`（参考 strategy §5.2）+ CHECK 约束（chk_actions / chk_severity_override）实现方式（GORM tag vs SQL migration，参照项目记忆 `xingran-gorm-sql-constraint-naming-conflict`）。
- **dept scope 递归子部门**：参照 `sys_dept` ancestors 递归模式定。
- **"显示已静默"开关**：异常列表是否提供切换显示 silence 记录。
- **CRUD admin 页表单布局**：CIDR 输入 + 冲突类型多选 + actions 多选 + scope 三选 + 有效期 DatePicker 的字段组织。
- **命中测试端点路径**：`POST /asset/reconciliation/exception-rule/test`（queryKeys.matchTest 已注册）。
- **operlog module 常量**：`ModuleReconciliationExceptionRule = "资产对账-例外规则"`（Phase 42 D-16 锁定）。
- **cache 策略**：`CacheKeyReconciliationExceptionRuleList` 等 helper 已在 Phase 42 定义（INFRA-04），R3 service 层接入。
- **IPv4/IPv6 支持**：`net.ParseCIDR` 原生支持双栈，无需额外处理。
- **降噪基线快照存储**：`sys_config` JSON vs 独立表的取舍。
- **降级后 severity 的 SLA 联动**：与 R2 D-A2-03 SLA 分级配合。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents (researcher / planner) MUST read these before planning or implementing.**

### Phase 42-43 前序上下文（必读）
- `.planning/phases/42-r1/42-CONTEXT.md` — R1 全部 18 个决策（D-01~D-18），含例外表 schema、Layer 3 引擎、cron sys_job 模式、operlog 边界
- `.planning/phases/43-r2/43-CONTEXT.md` — R2 转单 cron、WS/SysNotice、resolve API、7d 静默期/24h 节流 guard（R3 转单 cron SQL 改造的基础）
- `.planning/phases/42-r1/42-VERIFICATION.md` — R1 验收（确认表/MV/cron 已落地）

### 架构与策略
- `.planning/notes/asset-reconciliation-strategy.md` — v0.3 架构 + v0.4 复用审计；§4 IP 段例外规则体系（actions 语义/关键原则）、§5.2 例外表 schema（含 GiST + CHECK 定义）、§13 决策点追踪
- `.planning/notes/260627-reconciliation-reuse-audit.md` — F1-F7 必补项 + P1-P4 部分复用；**P3（§3.3）Excel 导入配置骨架** + **§4.1 F1 cache key helper** + **§4.7 F7 cron 注册**
- `.planning/seeds/asset-reconciliation-v1.17.md` — v1.17 阶段种子 + R3 启动门槛
- `.planning/todos/pending/v1.17-reconciliation-decisions.md` — D18（临时例外默认有效期 30 天）+ T29（P3 Excel 导入配置，R3 实施）

### Roadmap 与 Requirements
- `.planning/ROADMAP.md` Phase 44 段 — 2 plans 拆分（44-01/02）+ 10 条 success criteria
- `.planning/REQUIREMENTS.md` v1.17 — EXCEPTION-01~04 R3 范围

### 项目级 CLAUDE.md（强约束）
- `CLAUDE.md` "操作日志记录约定 (operlog convention) — 强制" — 11 关键词 + 25 OperType 常量
- `CLAUDE.md` "Status Value Convention" — 0=启用 1=停用（例外表 `is_active` 用此）
- `CLAUDE.md` "Cache Key Prefix Handling" — Redis `xingran:` 前缀处理
- `CLAUDE.md` "Excel 导入" 系列 — ExcelConfig / 按列位置匹配 / UpsertKey 需 DBField / 路由冲突规避

### 项目记忆（规划时必查，已在 MEMORY.md）
- `stat-cards-from-list-length-capped-at-100` — Statistics 必须用 COUNT 端点（降噪对比端点同样）
- `xingran-excel-import-column-position-matching` — Excel 按列位置取值，Columns 顺序须与 Excel 列一一对应
- `xingran-excel-import-upsertkey-needs-dbfield` — UpsertKey 列必须配 DBField
- `xingran-excel-import-route-conflict` — router.go 不预注册 `/asset/reconciliation/*`，由 Setup*Router 自管
- `xingran-migrations-no-sql-autoloader` — GiST 索引/CHECK 约束必须用 `migration_NNN_*.go` 显式调用 + AutoMigrate
- `xingran-gorm-sql-constraint-naming-conflict` — uniqueIndex 命名 `uni_*_*`，CHECK 约束命名规范
- `migration-sql-name-must-match-model` — migration SQL 字段名以实际 DB schema 为准
- `ad-operation-prefix-failover-source` — 例外规则设计借鉴 AD 故障风暴教训，必须节流（降噪动机）
- `xingran-perm-namespace-split-readonly-page` — 跨模块权限边界声明（R4 工位页用，R3 admin 页是模块内）

### 现有代码参考（实施时查）
- `internal/models/reconciliation.go:67-113` — `SysReconciliationException` 模型（字段已建，**缺 GiST + CHECK**）
- `internal/models/reconciliation.go:46-50` — `SysDataReconciliation.ExceptionRuleID` + `AppliedActions` 字段（R3 写入点）
- `internal/services/asset/reconciliation_detection.go:191-332` — `DetectLayer3` 循环；**Layer 3.5 插入点在 :262 前**（24h 节流 guard 之后、INSERT 之前）
- `internal/scheduler/reconciliation_tasks.go:42-102` — 单 taskType `"reconciliation"` + `params["param"]` 分发；`cleanupExpiredExceptions` case 已 placeholder（R3 真实实现）
- `internal/middleware/apikey.go:126-132` — `net.ParseCIDR` + `ipNet.Contains` 现成 CIDR 匹配模式
- `internal/services/operations/excel_config.go:50-297` — `ExcelConfigs` map（新增 `reconciliationExceptionRule` entityType）
- `internal/api/v1/asset/reconciliation_handler.go:14-19` — `ModuleReconciliation = "资产对账"`（R3 加 `ModuleReconciliationExceptionRule`）
- `internal/api/v1/asset/reconciliation_exception_router.go:21-22` — 已有 `/exception-rule/list` + `/exception-rule/:id`（R3 加 create/update/delete/test）
- `internal/api/v1/asset/reconciliation_statistics.go:77-85,455-478` — `ExceptionRuleStats` 端点已实现，R3 接入数据后自动生效
- `internal/services/asset/cache_keys.go` — `CacheKeyReconciliationExceptionRuleList` 等 helper 已定义（INFRA-04）
- `src/lib/queryKeys.ts` — `queryKeys.reconciliation.{ruleList, ruleDetail, matchTest}` 已注册（INFRA-05）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/middleware/apikey.go:126` `net.ParseCIDR` + `ipNet.Contains` — CIDR 匹配核心，D-R3-A1-03 预加载内存匹配直接复用
- `internal/services/asset/reconciliation_detection.go:DetectLayer3` — Layer 3.5 插入点已明确（:262 前），R3 在此加例外匹配 + 写 applied_actions
- `internal/scheduler/reconciliation_tasks.go:RegisterReconciliationTasks` — `cleanupExpiredExceptions` case 已 placeholder，R3 填真实逻辑（软停用）
- `internal/services/operations/excel_config.go:ExcelConfigs` — 新增 `reconciliationExceptionRule` 条目（P3 骨架）
- `internal/api/v1/asset/reconciliation_statistics.go:ExceptionRuleStats` — 例外生效统计端点已就绪，等 R3 数据
- `internal/services/asset/cache_keys.go` — 例外规则 cache key helper 已就位
- `src/lib/queryKeys.ts:reconciliation.{ruleList,ruleDetail,matchTest}` — 前端 queryKey 已注册
- `internal/services/system/config_service.go:GetByKey` — 降噪基线快照存 `sys_config` 的读写（D-R3-A4-01）

### Established Patterns
- **Handler-Service Pattern**：interface + 私有 impl + 构造函数（例外规则 CRUD 复用）
- **单 taskType cron 分发**：`params["param"]` switch case（D-R3-A1-02/A4-03 复用，不新增 scheduler 文件）
- **operlog.Record 强制约定**：例外规则 CRUD success path 末尾调用（CLAUDE.md 强约束）
- **Cache Key Helper**：常量 + `GetXxxKey()` 函数（cache_keys.go 模式）
- **Excel 名称→UUID 解析**：building 导入的 `scope_name` → dept_id/user_id 匹配（D-R3-A4-02）
- **GiST 索引 + partial index**：`WHERE is_active=0 AND deleted_at IS NULL`（参考 strategy §5.2）

### Integration Points
- `internal/services/asset/reconciliation_detection.go:262` — Layer 3.5 例外过滤插入点
- `internal/scheduler/reconciliation_tasks.go:cleanupExpiredExceptions` case — 过期软停用 cron
- `internal/api/v1/asset/reconciliation_exception_router.go` — 加 `/exception-rule/{create,update,delete,test}` 路由
- `internal/core/db/migrations/migration_NNN_reconciliation_exception_gist.go`（新建）— GiST 索引 + CHECK 约束
- `internal/services/operations/excel_config.go` — 加 `reconciliationExceptionRule` ExcelConfig
- `internal/api/v1/asset/reconciliation_handler.go` — 加 `ModuleReconciliationExceptionRule` 常量 + 降噪对比端点
- `xingran-react-frontend/src/pages/asset/reconciliation/exception-rules/index.tsx` — 例外规则 CRUD admin 页（strategy §7.1 菜单结构）
- `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` — 异常列表加 silence 默认过滤（D-R3-A1-01）
- 转单 cron SQL（`createWorkorderCritical`/`High`）— 加 `'no_workorder' != ANY(applied_actions)` 条件（D-R3-A1-02）

</code_context>

<specifics>
## Specific Ideas

- **silence 语义统一**（D-R3-A1-01）：silence = "写表但列表隐藏 + 全通路静默"，不是"完全不写"。异常列表 SQL 默认 `WHERE NOT ('silence' = ANY(applied_actions))`。
- **降级链**（D-R3-A2-02）：`原始severity --skip_severity--> 降一级 --severity_override--> 取更宽`。例：critical + skip_severity → high；再加 override=low → low。
- **合并算法**（D-R3-A2-01/02/03）：多规则命中时，`final_actions = UNION(各规则 actions)`；`final_severity = MIN(skip降级后的 severity, 各 override)`；`is_silence = 'silence' ∈ final_actions`。
- **命中测试交互**（D-R3-A3-03）：输入框 IP/CIDR（必填）+ 可选 user/dept 下拉；结果区顶部合并卡片 + 下方命中规则列表；dept/user 规则在未指定 user/dept 时显示"需指定"徽标。
- **降噪基线机制**（D-R3-A4-01）：admin 页一个"记录当前为基线"按钮 → POST 存 sys_config JSON；dashboard "降噪效果"卡片读基线 + 当前 ExceptionRuleStats 算下降%。
- **dept/user scope 双条件**（D-R3-A3-01）：匹配函数签名 `MatchException(assetIP string, responsibleUserID string, conflictType string) → (ruleID, actions, override, isSilence)`。
- **Excel scope_name 解析**（D-R3-A4-02）：`scope_type=dept` 时 scope_name 按部门名查 `sys_dept`；`=user` 时按用户名查 `sys_user`；`=global` 时 scope_name 留空。

</specifics>

<deferred>
## Deferred Ideas

下列决策**显式推后**到后续 R4-R5 阶段，R3 不实现：

- **R4（Phase 45）** — 工位详情页 HealthCard/HealthBadge/ReconciliationDrawer 组件、资产详情摘要块、HealthScore 函数（0-100）、跨模块调用 N+1 优化、抽屉"申请例外"按钮预填 IP/类型
- **R5（Phase 46, 可选）** — 高置信度修复建议（confidence ≥0.9）、人工确认 UI、一键回滚、误修复监控
- **R3 显式不做**：
  - 钉钉/邮件告警通道（D13 v0.3 锁定，下个 phase）
  - 例外规则的版本历史/审计回溯（软停用已保留记录，足够 R3）
  - 例外规则批量启用/停用（单条 CRUD 已够 R3 验证）
  - 例外规则导入预览/dry-run（复用 Excel 导入标准流程，不特殊化）

### Reviewed Todos (not folded)

`cross_reference_todos` 命中 2 项（均 score 0.4，低分），审阅后**不折叠**进 R3 scope：

- `operlog-exclude-paths.md`（operlog.exclude_paths 配置驱动白名单，解决 RPA 心跳日志污染）— **与 R3 无关**：R3 的 operlog 是例外规则 CRUD 正常记录，不涉及 exclude_paths 白名单机制。属 Phase 35 范围。
- `v1.17-reconciliation-decisions.md`（v1.17 决策点追踪）— **R3 相关项已在上游文档锁定**：D18（临时例外默认有效期 30 天，INFRA-02 config seed 已就位）+ T29（P3 Excel 导入配置，本 CONTEXT D-R3-A4-02 已细化）。该 todo `resolves_phase: 42`（R1），R3 不重复跟踪。

</deferred>

---

*Phase: 44-置信度评分 + IP 段例外 (R3)*
*Context gathered: 2026-06-28*
