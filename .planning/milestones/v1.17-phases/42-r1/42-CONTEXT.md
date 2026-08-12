# Phase 42: 资产对账观测底座 (R1) - Context

**Gathered:** 2026-06-27
**Status:** Ready for planning

<domain>
## Phase Boundary

建立 v1.17 资产对账的**观测底座**——物化视图 + 主表 + 6 个 Statistics COUNT 端点 + 字典/config/workorder 分类 seed + 异常列表 admin 页（只读）+ 简化 dashboard + Cron 调度（走 sys_job 表）。**不含**告警分发、例外规则 CRUD、前端整合（R2-R4 范围）。R1 的写操作边界为 migration 初始化 + Cron 触发的 Layer 3 检测写入 `sys_data_reconciliation`，**不上"标记已解决" UI**。

**已锁定的高层决策**（沿用 v0.3 / ROADMAP / REUSE-AUDIT，不重复展开）：
- 策略：Observe-only + 告警驱动人工修复
- 菜单归属：资产管理 / 数据质量
- 权限命名：`asset:reconciliation:*`（`list` / `export` / `dashboard` / `exception:*`）
- API 前缀：`/asset/reconciliation/*`
- 跨模块调用走 service 层（无权限时降级隐藏）
- Owner 合并：运维 + 资产 + 权限 同一人（无双签）

**与 v0.3 草稿的差异（v0.5 调整，必须遵守）：**
- 实际资产表名是 `ops_asset`（不是 `sys_asset`），字段名是 `MachineIP` / `MAC1` / `MAC2` / `UserID`（不是 `responsible_user_id`）
- AD 字段来自 `sys_user_ad_attrs` 表（不是 v0.3 假设的 `ad.managed_by_dn` 直读）
- R1 不做多 IP 解析链（asset.ip → workstation.ip → network_device.ip），只用 `ops_asset.machine_ip` 单值映射
- R1 不做 CIDR 例外匹配（IP 段规则是 R3 范围）

</domain>

<decisions>
## Implementation Decisions

### 物化视图刷新策略 (Area 1)

- **D-01:** **5min 定时 CONCURRENTLY 刷新**（`REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized`）
  - 与 `sys_config:asset.reconciliation.view.refresh_interval=5m` 对齐
  - 数据延迟 0-5min，PG 允许读（`asset_id` 已有 unique index，CONCURRENTLY 可用）
  - 不引入 trigger-based 增量（高频 MAC 变更下 trigger rebuild 反而慢）

- **D-02:** **失败仅 logrus 日志，下周期重试**（不写 cache 告警位、不写 SysNotice，避免与 R2 告警通路重复设计）
  - 配合应用启动后立即 refresh 一次（避免冷启 0-5min 数据为空）

- **D-03:** **R1 IP 字段最小化**：物化视图 `asset_ip` = `ops_asset.machine_ip`（单值直取）
  - **不**加新列、**不**做多 IP 解析链、**不**做 CIDR 例外匹配（R3 范围）
  - `sys_data_reconciliation.asset_ip INET` 仍保留作审计

### Dashboard 形态与位置 (Area 2)

- **D-04:** **父路由 `/asset/reconciliation` 302 → `/asset/reconciliation/dashboard`**
  - 父路由本身不渲染内容，仅作菜单 click target

- **D-05:** **Dashboard 与异常列表双向打通**（点击图表扇区/柱条 → 跳异常列表页并预填筛选）
  - 例：饼图 Type C 扇区 → `/asset/reconciliation/exceptions?type=C`
  - 异常列表读取 URL query string 作为初始筛选条件

- **D-06:** **5 KPI 卡片选型（不含 0-100 健康度总分）**
  1. 全量资产数（`SELECT COUNT(*) FROM ops_asset WHERE deleted_at IS NULL`）
  2. 未解决异常数（`SELECT COUNT(*) FROM sys_data_reconciliation WHERE resolved_at IS NULL AND deleted_at IS NULL`）
  3. critical 级未解决数（同上 + `severity = 'critical'`）
  4. 7d 新增异常数（`detected_at >= NOW() - INTERVAL '7 days' AND deleted_at IS NULL`）
  5. Top1 冲突类型及计数（按 `conflict_type` GROUP BY 排序取第一）
  - **严禁**用 `list.length`（`stat-cards-from-list-length-capped-at-100` 项目记忆），必须独立 COUNT 端点

### Layer 3 引擎 R1 边界 (Area 3)

- **D-07:** **R1 同步做完整 Layer 3 引擎**（A-F 分类 SQL 规则 + cron 写 `sys_data_reconciliation`）
  - R1 不告警、不转工单，但有真实异常数据可看
  - 满足 ROADMAP success criteria 7 "异常列表 admin 页面可正确展示 Type A-F 分布"

- **D-08:** **物理链路反向推导路径**：`ops_asset.mac1 → sys_port_mac → sys_info_point → sys_workstation_info_point → sys_workstation.user_id`
  - **mac1 优先，mac2 备选**（`COALESCE` 语义，避免物化视图单资产 2 行）
  - mac1 是大多数资产的有线 MAC，mac2 是无线 MAC 补充
  - 不做 mac1+mac2 并行 UNION（避免 Layer 3 分类歧义）

- **D-09:** **Type A 仅作 dashboard 统计，不进 `sys_data_reconciliation`**
  - 避免异常表膨胀（健康资产无需审计记录）
  - 物理链路 / 责任人 / AD 三路一致时，仅在物化视图存在，不写 reconciliation 主表
  - Type B-F 全部写 `sys_data_reconciliation`（含 severity=low 全部保留）

- **D-10:** **Cron 走 sys_job 表（`api/v1/scheduler` 现有页面管理）**
  - 在 `sys_job` 表新增 4 个 job records：MV 刷新 / Layer 3 检测 / 静默期到期重检测 / 临时例外清理
  - 不引入 `internal/scheduler/reconciliation_tasks.go`（保留 R1 无新增 cron 文件）
  - 调度可由运维在 UI 改 cron 表达式，无需发版

- **D-11:** **唯一性约束靠 `unique index`**（`uniq_recon_asset_type_open(asset_id, conflict_type) WHERE resolved_at IS NULL AND deleted_at IS NULL`）
  - Layer 3 检测 cron 写入时 catch unique violation 错误静默处理（已存在，未解决，跳过）
  - 不做先 SELECT 再 INSERT（TOCTOU 风险）
  - 不做 UPSERT（conflict_type 维度已 unique index 兜底）

### 健康度评分函数 (Area 4)

- **D-12:** **R1 不做 HealthScore 函数**
  - 5 KPI 不含 0-100 健康度总分（D-06 已锁定）
  - `internal/services/asset/reconciliation_health.go` R1 不创建
  - R4 工位详情页 HealthCard 需要时再补（与 R4 一起落地）

### 测试策略 (Area 5)

- **D-13:** **物化视图 + Layer 3 测试**：`sqlmock` 验证 SQL 语句 + 小型 test DB 集成测试验证 ETL
  - 不引入 testcontainers-go（项目无此依赖，避免新依赖）
  - 集成测试需 dev DB（CI 阶段再决定是否引入 docker）

- **D-14:** **6 个 Statistics 端点**：单元测试 sqlmock + e2e 集成（混合）
  - 单元测试用 sqlmock 验证每个端点 COUNT/GROUP BY SQL 正确性
  - 至少 1-2 个端点走 e2e（避免单元 sqlmock 与真 SQL 漂移）
  - 验证不走 `list.length` 路径（直接读取 `regression_test.go` 锁定的 11 关键词 + 25 OperType 不相关，但需自验证 SQL 不含 LENGTH）

- **D-15:** **异常列表 admin 页**：依赖 dev DB seed + 手工 UAT
  - 不引入 Vitest + React Testing Library（项目未用过，避免新依赖）
  - UAT 走 ROADMAP success criteria 7 全部 5 KPI 数字 + 3 个图表 + 列表分页 + 筛选

### operlog 与 R1 写操作边界 (Area 6)

- **D-16:** **operlog module 只定义 1 个**：`ModuleReconciliation = "资产对账"`
  - R2-R4 再加 `ModuleReconciliationExceptionRule` / `AutoWorkorder` / `Export`
  - R1 不会引入 4 个 module 常量占位（避免 R2 改动时还需对齐）

- **D-17:** **R1 全部写操作都走 operlog.Record**
  - ① Layer 3 检测 cron 触发（`OperTypeSync`）
  - ② `sys_data_reconciliation` 写入/更新（`OperTypeCreate` / `OperTypeUpdate`）
  - ③ 静默期到期重检测（`OperTypeSync`，R2 才有，但接口先就位）
  - ④ sys_job 表新增 4 个 cron 记录（`OperTypeCreate`）
  - 不需要 `RecordWithBody`（R1 写操作请求体无敏感字段）

- **D-18:** **R1 异常列表不上"标记已解决"按钮**（只读）
  - R1 success criteria 7 说的是"展示"不是"可操作"
  - R2 接入告警时再上"标记已解决" UI（与 WebSocket 推送同时落地）

### Claude's Discretion

- D-01~D-18 由用户与 Claude 共同决策，无 "you decide" 兜底项
- 未讨论的 R1 细节（如异常列表列顺序、TrendChart 时间窗口默认 7d vs 30d、TopUnresolved limit=10、Statistics 端点缓存策略）由 Claude 在 plan-phase 时根据 R1 范围与性能基线自行决定，并在 PLAN.md 中明确

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents (researcher / planner) MUST read these before planning or implementing.**

### 架构与策略（v0.3 / v0.5）
- `.planning/notes/asset-reconciliation-strategy.md` — v0.3 架构 + v0.4 复用审计 + v0.5 字段名调整（**必读**）
- `.planning/seeds/asset-reconciliation-v1.17.md` — v1.17 阶段种子 + R1 启动门槛
- `.planning/todos/pending/v1.17-reconciliation-decisions.md` — D1-D18 决策点状态 + T1-T30 待办（含 18 项 v0.3 锁定 + 7 项 v0.5 待 R1 启动时细化 + 7 项 R1 必补 + 6 项 T10-T18 实施期）

### 复用与跨模块（关键约束）
- `.planning/notes/260627-reconciliation-reuse-audit.md` — F1-F7 必补项 + P1-P4 部分复用项（**R1 plan 需逐条对应**）
- `.planning/notes/260627-cross-module-permission.md` — ops/workstation ↔ asset/reconciliation 跨模块权限边界（D-11 跨模块调用参考）
- `.planning/notes/260627-port-coverage-audit.md` — 端口采集覆盖率审计模板（T1 前置条件）

### Roadmap 与 Requirements
- `.planning/ROADMAP.md` Phase 42 段 — 6 plans 拆分（42-01 ~ 42-06）+ 10 条 success criteria
- `.planning/REQUIREMENTS.md` v1.17 — RECON-01~07 / MONITOR-01 / INFRA-01~05 / AUDIT-01~02 R1 范围

### 项目级 CLAUDE.md（强约束）
- `CLAUDE.md` "操作日志记录约定 (operlog convention) — 强制" — 11 关键词 + 25 OperType 常量
- `CLAUDE.md` "Status Value Convention" — 0=启用 1=停用
- `CLAUDE.md` "Cache Key Prefix Handling" — Redis `xingran:` 前缀处理

### 项目记忆（来自 MEMORY.md，规划时必查）
- `stat-cards-from-list-length-capped-at-100` — Statistics 必须用 COUNT 端点
- `xingran-server-side-sort-infra` — `BaseListRequest` + `ApplySort` 白名单已就位
- `xingran-migrations-no-sql-autoloader` — migrations/*.sql 不会被自动加载，必须用 `migration_NNN_*.go` 函数显式调用
- `xingran-gorm-sql-constraint-naming-conflict` — GORM `uniqueIndex` 期望 `uni_*_*`，SQL inline UNIQUE 用 PG 自动名
- `migration-sql-name-must-match-model` — 字段名以实际 DB schema 为准（v0.5 已用 `MachineIP` / `MAC1` / `MAC2` / `UserID`）
- `ops 菜单 seed perms 与路由命名不一致` — 菜单 seed perms 用单数+连字符，路由对齐
- `GORM migration tag 不阻止 INSERT` — `gorm:"-:migration"` 只设 IgnoreMigration
- `workstation-ad-device-managedby-vs-description` — AD 反查不走 managed_by，用 MAC + 信息点 + 工位链路
- `Excel 导入路由冲突陷阱` — router.go 不预注册 `/asset/reconciliation/*`，由各 `Setup*Router` 自管
- `xingran-perm-namespace-split-readonly-page` — 跨模块调用显式权限声明 + `RequirePermissionsWithQuery`

### 现有代码参考（实施时查）
- `internal/services/operations/asset_service.go` — asset 模块现有模式（handler-service）
- `internal/services/workorder/base.go` — `workorder.BaseService.Create()` 模式（R2 引入）
- `internal/api/v1/scheduler/` — sys_job 现有 job_cron_util + job_utils.go（D-10 cron 走 sys_job 模式）
- `internal/services/system/config_service.go` — `ConfigService.GetByKey()` 模式（D-02 sys_config seed 读取）
- `internal/services/system/dict_cache_impl.go` — 字典缓存实现
- `internal/services/operations/asset_statistics_test.go` — Statistics 测试模式
- `internal/services/cache_keys.go` — CacheKey helper 函数现有模式

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/services/system/config_service.go:23-34` `ConfigService` 接口 + `GetByKey()` — 8 个 config seed 读取
- `internal/services/system/dict_cache_impl.go` — 字典缓存 4 个字典 seed
- `internal/services/operations/asset_service.go` — asset 模块现有 CRUD 模式（handler-service 双层架构）
- `internal/services/operations/asset_statistics_test.go` — Statistics sqlmock + 集成测试模式
- `internal/services/operations/pagination_helper.go:17` `MaxPageSize = 10000` — 异常列表分页（注意 Statistics 端点不走 list.length）
- `internal/services/workorder/base.go` `workorder.BaseService.Create()` — R2 引入，R1 不调用
- `internal/api/v1/scheduler/job_cron_util.go` + `job_utils.go` — sys_job 表 cron 注册与执行（D-10 复用）
- `internal/core/db/migrations/` — migration_NNN_*.go 模板（D-21 R1 必加 migration 文件）

### Established Patterns
- **Handler-Service Pattern**：interface + 私有 impl + 构造函数（`internal/api/v1/system/user_handler.go` 范例）
- **Cache Key Helper**：常量 + `GetXxxKey()` 函数（`internal/services/cache_keys.go` 范例）
- **Statistics 专用 COUNT 端点**：6 个端点独立 `POST /asset/reconciliation/statistics/*`
- **operlog 强制约定**：所有写操作 success path 末尾 `operlog.Record(...)` 之前调用（CLAUDE.md 强约束）
- **sys_job 表 + api/v1/scheduler UI 调度**：避免硬编码 cron 表达式
- **跨模块 service 层调用**：`workstationService` 注入 `reconciliationService`，Handler 内权限降级（D-11 + 跨模块权限声明）

### Integration Points
- `internal/api/router.go` — 注册 `SetupReconciliationRouter(r, core)` + `SetupReconciliationExceptionRouter(r, core)`（D-21 F2 路由注册）
- `internal/core/db/migrations/migration_NNN_reconciliation_*.go` — 表 + 物化视图 + 字典 seed + config seed + workorder 分类 seed（D-21 F1/F3 必加）
- `internal/services/asset/cache_keys.go` — 8 个 CacheKey 常量 + helper（D-24 F5 必加）
- `src/lib/queryKeys.ts` — `queryKeys.reconciliation.{all, dashboard, exceptionList, ...}` 9 个 key（D-25 F6 必加）
- `src/pages/asset/reconciliation/{index.tsx, dashboard/index.tsx, exceptions/index.tsx}` — R1 父路由 + dashboard + 异常列表（exception-rules 子菜单 R1 不创建）

</code_context>

<specifics>
## Specific Ideas

- **5 KPI 卡片顺序**（按 ROADMAP 5 KPI 选型 D-06）：全量资产数 → 未解决异常数 → critical 数 → 7d 新增 → Top1 冲突类型
- **Dashboard 3 图表布局**：饼图（按 Type A-F）+ 柱状图（按 severity 4 级）+ 趋势图（健康度 7d/30d 切换，R1 默认 7d）
- **异常列表列**：detected_at / conflict_type / severity / asset_code / asset_ip / physical_username / responsible_username / exception_rule_id（命中 R3 例外才显示）/ operlog_btn（R1 显示"查看日志"但不开可操作）
- **TrendChart 时间窗口**：默认 7d，可选 30d / 90d（与 ROADMAP INFRA-02 `health.score_weights` 配合，R1 不依赖）
- **双向跳转 URL 模式**：`/asset/reconciliation/exceptions?type=C&severity=critical&from=2026-06-20&to=2026-06-27`
- **cron 走 sys_job 表后**：R1 UI 调度周期由运维改，研发不动代码（dev 阶段 cron 表达式参考 `@every 5m` MV 刷新 + `@every 6m` Layer 3 检测 + `0 2 * * *` 静默期回收 + `0 3 * * *` 过期例外清理）

</specifics>

<deferred>
## Deferred Ideas

下列决策**显式推后**到后续 R2-R5 阶段，R1 不实现：

- **R2（Phase 43）** — critical/high 自动转工单 + WebSocket 推送 + SysNotice + 6 类工单模板 + 修复回写 7d 静默期 + 24h 节流 + 异常标记已解决 UI
- **R3（Phase 44）** — IP 段例外规则引擎（CIDR GiST 索引）+ 例外规则 CRUD admin 页 + 命中测试工具 + Excel 导入导出例外规则 + 临时例外到期 cron
- **R4（Phase 45）** — 工位详情页整合 HealthCard + HealthBadge + ReconciliationDrawer 组件 + 资产详情摘要块 + HealthScore 函数（届时实现）+ 跨模块调用 N+1 优化
- **R5（Phase 46, 可选）** — 高置信度修复建议（confidence ≥0.9）+ 人工确认 UI + 一键回滚

下列**v0.3 决策点明确推到对应 phase**，R1 不需要细化：
- D13 告警分发范围（R2）
- D14 R5 半自动修复阈值（R5）
- D18 临时例外默认有效期（R3）

**Claude's Discretion 范围**（R1 plan-phase 自决）：
- 异常列表列顺序与默认排序（`detected_at DESC` 优先）
- TopUnresolved limit = 10（与 ROADMAP success criteria 7 一致）
- 6 个 Statistics 端点缓存策略（R1 不缓存，每次 COUNT 走真表）
- HealthTrend 端点时间窗口默认 7d
- Dashboard 趋势图 x 轴粒度（按天 vs 按小时）

</deferred>

---

*Phase: 42-资产对账观测底座 (R1)*
*Context gathered: 2026-06-27*
