# Phase 45: 工位详情整合 + 资产详情摘要 (R4) - Context

**Gathered:** 2026-06-28
**Status:** Ready for planning

<domain>
## Phase Boundary

把 v1.17 资产对账 R1–R3 已建好的引擎产出（检测、统计、例外、降噪）**嵌入现有前端 UI**：

- **工位侧**：在 `pages/operations/workstations` 现有列表的 expand 展开区顶部，新增 `HealthCard`（5 KPI + 趋势 mini chart + 得分）；在 `components/operations/WorkstationDeviceTable` 的 AD 设备子表 + 资产设备子表加"对账健康"列（行内徽标）。
- **资产侧**：在 `pages/operations/assets` 列表行内加 `HealthBadge` 列，点击弹 `ReconciliationDrawer`（与工位同组件复用）。
- **跨模块**：无 `asset:reconciliation:list` 权限时**静默隐藏**（不 403、不占位），双侧（工位 + 资产）都降级。
- **抽屉三 Tab**：冲突摘要 / 历史变更（resolved 记录）/ 例外规则（命中该资产 IP 的当前生效例外）。
- **后端**：新建 `POST /asset/reconciliation/by-workstation` 聚合 API（含 healthScore + 工位下资产徽标 list），service 层直接调用 + CacheProvider 缓存（TTL 5min 与 MV 刷新一致）；R2 转单/resolve 后主动 invalidate。
- **不含**：新建对账后端引擎（R1–R3 已完成）、半自动修复（R5/Phase 46）、新建 `/ops/workstation/:id` 或 `/asset/card/:id` 详情页路由（采用现有展开/列表 + 行内抽屉方案）。

</domain>

<decisions>
## Implementation Decisions

### 整合落点（Area 1）

- **D-A1-01（工位落点）**：`HealthCard` 放置于 `pages/operations/workstations` CardView/FloorPlanView 现有 expand 展开区顶部（紧邻 `WorkstationDeviceTable` 之前）。**不**新建 `/ops/workstation/:id` 详情页路由。理由：复用现有 list+expand 模式、与 strategy §6.1 "工位→子表" 一致、scope 小风险低。
- **D-A1-02（资产落点）**：在 `pages/operations/assets` 列表行加 `HealthBadge` 列，点击弹 `ReconciliationDrawer`（与工位共用同一抽屉组件、三 Tab 同契约）。**不**新建 `/asset/card/:id` 详情页。理由：与 D-A1-01 对称、复用抽屉组件、SC4 "资产详情页顶部摘要" 语义由列表行内抽屉承担（drawer 顶部即摘要块）。
- **D-A1-03（权限降级范围）**：双侧都降级——无 `asset:reconciliation:list` 时，工位展开区 HealthCard + 设备子表徽标**静默隐藏**（不 403、不占位）；无 `asset:reconciliation:list` 时，资产列表 HealthBadge 列同样静默隐藏。实现模式：`WorkstationHandler.GetByID` + `WorkstationHandler` expand 内 + `AssetHandler.List` 内统一调用 `hasReconciliationPerm`，通过 `ReconciliationVisible` 标志位控制 UI 渲染。理由：与 strategy §7.4 + cross-module-permission.md §2.3 锁定决策一致；避免读写路径碎片化（参 `xingran-perm-namespace-split-readonly-page`）。

### 健康度得分口径 + 时间窗口（Area 2）

- **D-A2-01（得分范围）**：`HealthScore.assets` 范围 = 该工位 sys_workstation_info_point → info_point → port_mac → MAC 关联的所有 sys_asset 资产（走 R1 `reconciliation_normalized` 物化视图的 `workstation_id` 列）。**不**仅按 `w.user_id` 关联 ops_asset.UserID（会漏掉同工位其他设备）。
- **D-A2-02（时间窗口）**：`HealthCard` 默认**固定本周**（最近 7 天检出异常）。**不**做今日/本周/本月切换。理由：与 strategy §6.2 mockup "[本周]" 字面一致；与 R2 24h 节流 + R3 7d 静默期语义对齐；趋势 mini chart 单独取历史趋势数据补足长期视角。
- **D-A2-03（得分公式）**：`score = clamp(round((1 - 异常资产数/总资产数) × 100), 0, 100)`，简单比。**不**用 config seed 的 `health.score_weights = {normal:1.0, drift:0.5, conflict:0.0, nodata:0.7}` 加权。理由：与 ROADMAP SC1 mockup "得分: 78/100" 字面一致；权重公式中 conflict=0 会让单个冲突资产把 score 拉到 0 产生假报警，违反"得分 ≥0 含义直观"。
- **D-A2-04（行内徽标粒度）**：`HealthBadge` 用 6 类色点（A/B/C/D/E/F），色映射复用 `useDict("asset_reconciliation_conflict_type")` 的 list_class（success/warning/error/default/processing）。**不**用文字徽标（列宽不够），**不**用二态（丢类型信息）。点击 → 抽屉冲突摘要 Tab 自动定位。

### 抽屉历史变更 Tab 数据源（Area 3）

- **D-A3-01（历史变更数据源）**：`ReconciliationTimeline` 数据 = 该资产所有 `resolved_at IS NOT NULL` 的 `sys_data_reconciliation` 记录，按 `resolved_at` 倒序。**不**用 sys_oper_log（粒度不匹配），**不**混 open + resolved。理由：与现有唯一索引防风暴语义一致（同一 asset+type 只会有一条 open）；`resolved_at + resolution_note + raw_snapshot` 字段齐全；满足 AUDIT-02 审计要求。
- **D-A3-02（Timeline 内容）**：纯文本 timeline，**不**展开 raw_snapshot。每行显示：冲突类型（A-F 色点）+ 检出时间（detected_at）+ 解决时间（resolved_at）+ 解决人（resolved_by username，JOIN sys_user）+ 解决说明（resolution_note）。raw_snapshot 留作后台溯源入口（不在抽屉内展开，避免占空间）。
- **D-A3-03（例外规则 Tab 数据源）**：当前生效中且 IP CIDR 命中该资产 IP（或工位 IP）的 `sys_reconciliation_exception` 列表。每条显示：规则名 + IP 范围 + actions + severity_override + scope + 有效期 + reason。**不**含已停用历史（R3 A4-03 软停用保留记录但 R4 抽屉不展示；admin 例外规则管理页可看）。

### 后端聚合 API 形态 + 实时性（Area 4）

- **D-A4-01（API 路由形态）**：`POST /asset/reconciliation/by-workstation`，请求体 `{workstationId: "uuid", window: "7d"}`。**不**用 GET `/:ws_id`（与项目 CLAUDE.md "POST /list 模式" 不一致；查询参数放 query string 不便）。理由：与项目约定一致；body 易拓展（如未来加 window/filters）。
- **D-A4-02（API 响应结构）**：
  ```go
  type ByWorkstationResponse struct {
      Workstation WorkstationBrief  `json:"workstation"`        // 工位基础信息
      HealthScore HealthScore       `json:"healthScore"`        // {total, normal, drift, conflict, nodata, exceptionHit, score, trend}
      Assets      []AssetHealthItem `json:"assets"`             // 行内徽标数据
      Visible     bool              `json:"visible"`            // 权限降级标志
  }
  
  type HealthScore struct {
      Total        int          `json:"total"`
      Normal       int          `json:"normal"`
      Drift        int          `json:"drift"`
      Conflict     int          `json:"conflict"`
      NoData       int          `json:"noData"`
      ExceptionHit int          `json:"exceptionHit"`
      Score        int          `json:"score"`        // 0-100, 简单比
      Trend        []TrendPoint `json:"trend"`        // mini chart 数据
  }
  
  type AssetHealthItem struct {
      AssetID         string   `json:"assetId"`
      AssetCode       string   `json:"assetCode"`
      ConflictType    string   `json:"conflictType"`    // A-F or "" (健康)
      Severity        string   `json:"severity"`        // low/medium/high/critical or ""
      ExceptionRuleID *string  `json:"exceptionRuleId,omitempty"`
      AppliedActions  []string `json:"appliedActions,omitempty"`
      ConfidenceScore float64  `json:"confidenceScore"`
  }
  ```
  一次拿完顶部卡片 + 资产子表徽标 + 详情跳转锚点。**不**仅返 healthScore 让前端另查（避免 N+1，与 ROADMAP SC7 一致）。
- **D-A4-03（实时性策略）**：打开工位/资产详情页时**调一次 API + 缓存**（`CacheKeyReconciliationHealthByWorkstation`，TTL 5 分钟与 R1 MV 刷新一致）。**不**用 WS 实时增量。理由：工位详情页是低频访问页面（点开特定工位才看），5 分钟 TTL 足够用户感知；SC5 ≤200ms 用 service 层直接调用 + 缓存已达成；不重复造 WS 通道。
- **D-A4-04（缓存主动失效）**：R2 转单（`createWorkorderCritical`/`createWorkorderHigh`）+ R2 resolve API (`POST /asset/reconciliation/exception/:id/resolve`) 完成后，调用 `invalidate_workstation_health(wsID)` 删除 `CacheKeyReconciliationHealthByWorkstation` 缓存。理由：业务闭环——修复后用户重看页面立即看到变化；R3 7d 静默期生效期间不重 invalidate（避免资源重撞）。

### Claude's Discretion

下列 R4 实现细节由 planner/researcher 在 plan-phase 自决（属技术实现或已在上游文档锁定）：

- HealthCard 顶部 KPI 卡片具体排版与图标（参考 strategy §6.2 mockup）。
- 趋势 mini chart 用 echarts-for-react（与 R1 Dashboard 一致）。
- ReconciliationTimeline 时间格式（dayjs vs Intl）。
- 例外规则命中查询的 IP 解析顺序（asset.ip → workstation.ip → network_device.ip via port → unknown，与 strategy §4.4 一致）。
- `invalidate_workstation_health` 在 resolve API 中的调用位置（success path 末尾、operlog.Record 之前）。
- WS 推送**不**新增，但 R2 已有 `notice_hub.go` 通道在 R4 仍供 dashboard 使用（不扩展到工位页）。
- 工位侧跨模块 service 层注入：在 `WorkstationHandler` 构造函数加 `reconciliationService asset.ReconciliationService`（注：`WorkstationHandler` 在 `internal/api/v1/operations/workstation_handler.go`，需新增 asset service 注入；参 cross-module-permission.md §2.3）。

### Reviewed Todos (not folded)

`cross_reference_todos` 命中 2 项（与 Phase 44 审阅结论一致，R4 同样不折叠）：

- `v1.17-reconciliation-decisions.md`（v1.17 决策点追踪与待办清单，`resolves_phase: 42`）—— R4 相关项（T27 HealthScore 计算函数、T4 跨模块权限文档）已被本 CONTEXT D-A2-03 + D-A1-03 锁定；T4 文档已写为 `260627-cross-module-permission.md`。不重复跟踪。
- `operlog-exclude-paths.md`（operlog.exclude_paths 配置驱动白名单，解决 RPA 心跳日志污染）—— Phase 35 范围，与 R4 无关。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents (researcher / planner) MUST read these before planning or implementing.**

### 前序上下文（必读，R4 直接构建于其上）
- `.planning/phases/42-r1/42-CONTEXT.md` — R1 全部 18 个决策（D-01~D-18），含 sys_data_reconciliation schema、reconciliation_normalized 物化视图、cron sys_job 模式、operlog 边界
- `.planning/phases/43-r2/43-CONTEXT.md` — R2 转单 cron + WS/SysNotice + resolve API + 7d 静默期/24h 节流 guard（R4 invalidate 触发点 + N+1 后端聚合的基础）
- `.planning/phases/44-ip-r3/44-CONTEXT.md` — R3 例外引擎 + admin 例外规则 CRUD + 命中测试工具 + 降噪基线对比（R4 例外规则 Tab 数据源 + 跨模块权限边界）
- `.planning/phases/45-r4/45-DISCUSS-CHECKPOINT.json` — 本阶段讨论检查点（决策原始数据）

### 架构与策略
- `.planning/notes/asset-reconciliation-strategy.md` — v0.3 架构 + v0.4 复用审计；§6 工位详情页前端整合（HealthCard mockup + 组件结构 §6.4）、§7.5 菜单 seed 注意事项、§13 决策点追踪
- `.planning/notes/260627-reconciliation-reuse-audit.md` — F1-F7 必补项 + P1-P4 部分复用；F5 HealthScore 计算函数 = R4 实施；P3 Excel 导入配置不属 R4
- `.planning/notes/260627-cross-module-permission.md` — R4 跨模块权限边界核心文档；§2.3 Handler 内权限降级模式（hasReconciliationPerm + ReconciliationVisible flag）；§3 菜单权限矩阵
- `.planning/notes/260627-port-coverage-audit.md` — R1 启动门槛审计（已达成，R4 不重复）
- `.planning/seeds/asset-reconciliation-v1.17.md` — v1.17 阶段种子（5 阶段 R1-R5 路线）
- `.planning/todos/pending/v1.17-reconciliation-decisions.md` — v1.17 决策点追踪（R4 相关项已被本 CONTEXT 锁定）

### Roadmap 与 Requirements
- `.planning/ROADMAP.md` Phase 45 段 — 2 plans 拆分（45-01/02）+ 10 条 success criteria
- `.planning/REQUIREMENTS.md` v1.17 — INTEGRATE-01~03 R4 范围

### 项目级 CLAUDE.md（强约束）
- `CLAUDE.md` "操作日志记录约定 (operlog convention) — 强制" — 11 关键词 + 25 OperType 常量
- `CLAUDE.md` "Status Value Convention" — 0=启用 1=停用
- `CLAUDE.md` "Cache Key Prefix Handling" — Redis `xingran:` 前缀处理
- `CLAUDE.md` "API Response Format" — code:0 成功、1001/4001 等错误码
- `CLAUDE.md` "前端 useEffect Dependencies" — R4 新增 React 组件必须稳定 deps

### 项目记忆（规划时必查，已在 MEMORY.md）
- `xingran-perm-namespace-split-readonly-page` — 读写权限命名空间割裂致 403；R4 跨模块权限边界声明必须显式
- `stat-cards-from-list-length-capped-at-100` — HealthCard KPI 数字必须用 COUNT 端点，不用 list.length
- `migration-sql-name-must-match-model` — R4 不新增 menu/migration（无需）
- `xingran-gorm-sql-constraint-naming-conflict` — R4 不新增 uniqueIndex（无需）
- `xingran-excel-import-route-conflict` — R4 不涉及 Excel（无需）
- `workstation-ad-device-managedby-vs-description` — 工位侧 AD 设备关联不走 managed_by
- `ad-update-no-such-object-vs-lockout` — R4 例外规则 Tab 不暴露 DN，遵循此原则

### 现有代码参考（实施时查）
- `internal/services/asset/reconciliation_service.go:130-133` — `ListExceptions` + `GetByID`（R4 新增 GetByWorkstation + GetByAsset 方法）
- `internal/services/asset/cache_keys.go:43-44, 82-85` — `CacheKeyReconciliationHealthByWorkstation` 已预留 + helper `GetReconciliationHealthByWorkstationKey`
- `internal/services/asset/reconciliation_statistics.go` — R1 6 个 Statistics 端点（R4 HealthScore 复用 Statistics 模式 + COUNT）
- `internal/services/asset/reconciliation_exception_matcher.go` — R3 例外命中匹配函数（`MatchException(assetIP, responsibleUserID, conflictType)`），R4 例外规则 Tab 复用
- `internal/models/reconciliation.go:26-60` — `SysDataReconciliation` 字段（resolved_at + resolution_note + raw_snapshot，D-A3-01/02 数据源）
- `internal/api/v1/asset/reconciliation_router.go` — R1 路由注册位置（R4 新增 by-workstation 路由）
- `internal/api/v1/asset/reconciliation_handler.go:14-19` — `ModuleReconciliation = "资产对账"` + `ModuleReconciliationExceptionRule` 常量（D-A4-04 invalidate operlog 复用）
- `internal/scheduler/reconciliation_tasks.go` — R3 cron 注册位置（R4 不新增）
- `internal/websocket/notice_hub.go` — R2 WS 通道（R4 不复用，详见 D-A4-03）
- `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx` — 设备子表组件（D-A1-01 集成点 + D-A2-04 徽标列加在此组件）
- `xingran-react-frontend/src/pages/operations/workstations/views/CardView.tsx` — 工位列表视图（D-A1-01 expand 展开区定位）
- `xingran-react-frontend/src/pages/operations/assets/index.tsx` — 资产列表（D-A1-02 HealthBadge 列加在此处）
- `xingran-react-frontend/src/lib/opsApi.ts` — opsApi factory 模式（R4 新增 reconciliationApi 模块参考）
- `xingran-react-frontend/src/lib/queryKeys.ts` — queryKeys registry（R3 已注册 reconciliation.*，R4 复用 `workstationHealth` + `assetHealth` key）
- `xingran-react-frontend/src/hooks/useDict.ts` — `useDict("asset_reconciliation_conflict_type")` for badge color

</canonical_refs>

<existing_code>
## Existing Code Insights

### Reusable Assets
- `internal/services/asset/cache_keys.go:GetReconciliationHealthByWorkstationKey(wsID)` — R4 缓存键 helper 已就位
- `internal/services/asset/reconciliation_statistics.go` — `Summary` / `ByConflictType` / `BySeverity` / `HealthTrend` 接口签名可参考，R4 `HealthScoreCalculator` 复用该模式
- `internal/services/asset/reconciliation_exception_matcher.go:MatchException()` — 例外规则命中函数，R4 抽屉例外规则 Tab 复用
- `internal/middleware/apikey.go:net.ParseCIDR + ipNet.Contains` — CIDR 匹配核心（工位 IP 解析 + 例外规则匹配复用）
- `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx` — 设备子表组件，三段折叠（手动/AD/资产），R4 徽标列加在 AD/资产两张子表
- `xingran-react-frontend/src/hooks/useDict.ts` — 字典 hook，D-A2-04 徽标色点复用
- `pkg/response` + `response.Success/Error` — Handler 响应包装（D-A1-03 ReconciliationVisible 字段注入响应）

### Established Patterns
- **Handler-Service 模式**：interface + 私有 impl + 构造函数（R4 `HealthScoreCalculator` + `ReconciliationAggregationService` 复用）
- **Cache Key Helper**：常量 + `GetXxxKey()` 函数（R4 复用 `cache_keys.go` 模式）
- **统计专用 COUNT 端点**：`SELECT COUNT(*)` 或聚合查询，不允许 `list.length`（D-A2-03 健康度数字必须遵守）
- **operlog.Record 强制约定**：R2 转单 + R3 resolve + R4 invalidate 主动失效调用后均需 operlog（CLAUDE.md 强约束）
- **跨模块 service 层调用**：service 注入而非 HTTP 调用（D-A1-03，`WorkstationHandler` 直接注入 `asset.ReconciliationService`）
- **Excel 导入路由冲突规避**：R4 不涉及 Excel，沿用 project memory `xingran-excel-import-route-conflict`
- **React Query useEffect 依赖**：R4 新增 React Query hooks 必须稳定 deps（CLAUDE.md useEffect Dependencies 强制）

### Integration Points
- `internal/services/asset/reconciliation_service.go` — 新增 `GetByWorkstation(ctx, wsID, window)` 方法（POST by-workstation handler 调用）
- `internal/services/asset/cache_keys.go` — 新增 `invalidate_workstation_health(wsID)` helper
- `internal/api/v1/asset/reconciliation_router.go` — 新增 `POST /by-workstation` 路由
- `internal/api/v1/operations/workstation_handler.go` — `WorkstationHandler` 构造函数注入 `asset.ReconciliationService`，`GetByID` 内调用 `hasReconciliationPerm`
- `internal/scheduler/reconciliation_tasks.go`（R3 已建）— R2 resolve API 不在此文件，在 `internal/api/v1/asset/reconciliation_exception_handler.go`，R4 invalidate 在 resolve handler success path 调用
- `xingran-react-frontend/src/pages/operations/workstations/views/CardView.tsx` — expand 顶部塞 HealthCard
- `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx` — `createColumns(canEdit)` 函数加 HealthBadge 列
- `xingran-react-frontend/src/pages/operations/assets/index.tsx` — 列表 columns 加 HealthBadge
- `xingran-react-frontend/src/components/reconciliation/` — 新建 R4 组件目录（HealthCard / HealthBadge / ReconciliationDrawer / ReconciliationTimeline / ExceptionMatchList + hooks，参 strategy §6.4）
- `xingran-react-frontend/src/lib/queryKeys.ts` — 复用 `workstationHealth` + `assetHealth` key（R3 INFRA-05 已注册）

</existing_code>

<specifics>
## Specific Ideas

- **整合落点对称性**：工位 = 现有 expand + 抽屉；资产 = 列表行 + 抽屉。两边都用同一个 `ReconciliationDrawer` 组件 + 三 Tab 同契约（冲突摘要 / 历史变更 / 例外规则）。
- **抽屉"申请例外"按钮预填**：从抽屉点击 → 跳转 `/asset/reconciliation/exception-rules/new` 并预填 `assetIP` + `conflictType`（通过 query string 或 React Router state）。R3 exception-rules 页面新增页（create 页）需支持 query 预填。
- **缓存主动失效触发点**：D-A4-04 列出 R2 转单 + resolve 两处。这两处原本就走 operlog，invalidate 与 operlog.Record 调用顺序：**invalidate → operlog.Record → response.Success**（避免先返响应再失效导致用户重看仍命中旧缓存）。
- **权限降级 flag 命名**：用 `visible: true/false` 而非 `ReconciliationVisible`（API 响应 snake_case 与 camelCase 风格统一为 lowercase 简写）。
- **健康度得分展示**：HealthCard 顶部展示得分数字（如 "78" / "92"）+ 颜色（≥80 绿 / 60-79 黄 / <60 红），与现有 antd Statistic 组件复用。
- **趋势 mini chart**：用 echarts-for-react Sparkline / line 图（仅显示得分变化 7 天），不展示完整坐标轴（节省空间）。
- **徽标点击交互**：点击 HealthBadge → drawer.open=true + selectedAssetId=record.assetId（自动跳到抽屉摘要 Tab 该资产段）。
- **跨模块注入示例**（cross-module-permission.md §2.3 + CLAUDE.md Handler-Service 模式）：
  ```go
  // internal/api/v1/operations/workstation_handler.go
  type WorkstationHandler struct {
      workstationService     operations.WorkstationService
      reconciliationService  asset.ReconciliationService  // 🆕 R4 注入
      core                   *core.Core
  }
  
  func (h *WorkstationHandler) GetByID(c *gin.Context) {
      // ... 现有逻辑
      ws, err := h.workstationService.GetByID(ctx, id)
      // 🆕 R4 跨模块调用 + 权限降级
      if h.hasReconciliationPerm(c) {
          if recon, err := h.reconciliationService.GetByWorkstation(ctx, id, "7d"); err == nil {
              ws.Reconciliation = recon
              ws.ReconciliationVisible = true
          } else {
              ws.ReconciliationVisible = false
          }
      } else {
          ws.ReconciliationVisible = false
      }
      response.Success(c, ws)
  }
  
  func (h *WorkstationHandler) hasReconciliationPerm(c *gin.Context) bool {
      userID, _ := c.Get("userID")
      return h.core.PermissionService.UserHasPerm(userID, "asset:reconciliation:list")
  }
  ```
- **ROADMAP SC1 mockup "5 正常 / 1 漂移 / 0 冲突 / 2 无数据"** 对应 HealthScore 字段：normal=5, drift=1, conflict=0, nodata=2。SC1 "趋势 mini chart" 对应 HealthScore.Trend []TrendPoint。

</specifics>

<deferred>
## Deferred Ideas

下列决策**显式推后**到后续 R5 / 后续 phase，R4 不实现：

- **R5（Phase 46，可选）** — 半自动修复（高置信度建议修复 + 人工确认 UI + 一键回滚 + 误修复监控）。R4 健康度 UI 是 R5 修复建议的基础。
- **R4 显式不做**：
  - 钉钉/邮件告警通道（D13 v0.3 锁定，下个 phase）
  - WS 推送增量到工位详情页（D-A4-03 锁定为打开拉一次 + 缓存，5min TTL）
  - 抽屉内展开 raw_snapshot JSONB（D-A3-02 锁定纯文本 timeline）
  - 工位/资产详情页路由新建（D-A1-01/02 锁定为现有 expand + 列表 + 抽屉）
  - 健康度得分公式用权重加权（D-A2-03 锁定简单比）
  - 例外规则批量启用/停用、版本历史/审计回溯（R3 也不做，R4 不重复）
  - 工位设备子表加新列以外的功能（如设备编辑、对账修复按钮直改）—— R4 范围仅限"展示对账状态"

### Reviewed Todos (not folded)

`cross_reference_todos` 命中 2 项（与 Phase 44 审阅结论一致）：

- `v1.17-reconciliation-decisions.md`（v1.17 决策点追踪与待办清单，`resolves_phase: 42`）—— R4 相关项（T27 HealthScore + T4 跨模块权限文档）已被本 CONTEXT D-A2-03 + D-A1-03 锁定；T4 文档已写为 `260627-cross-module-permission.md`。不重复跟踪。
- `operlog-exclude-paths.md`（operlog.exclude_paths 配置驱动白名单，解决 RPA 心跳日志污染）—— Phase 35 范围，与 R4 无关。

---

*Phase: 45-工位详情整合 + 资产详情摘要 (R4)*
*Context gathered: 2026-06-28*