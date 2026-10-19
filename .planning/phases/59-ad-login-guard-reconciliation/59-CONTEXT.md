# Phase 59: 登录回写守卫与部门对账视图 - Context

**Gathered:** 2026-10-19
**Status:** Ready for planning

<domain>
## Phase Boundary

AD 登录回写 `dept_id` 前，若该用户存在活跃的 `update_user` 队列行则**跳过覆盖并 WARN**（用 Phase 57/58 的 outbox 信号替代被三方审计否决的时间戳仲裁）；同时提供 `v_dept_ou_mismatch` 普通 SQL VIEW + 只读分页端点，暴露 dept↔OU 不一致（含映射指向软删部门）。

**不含：** 登录路径不查 `user.Status` 的独立 bug（57 遗留）、`sys_user.ad_config_id` 用户级归属（58 Deferred）、MV/物化视图（ROADMAP 明令禁用）、对账数据的**自动修复**能力（本 phase 只做可观测）。

</domain>

<decisions>
## Implementation Decisions

### 守卫判定（GUARD-01）

- **D-01: 只拦 `dept_id`。** ROADMAP SC1 字面范围即「本地 dept 不被 AD 旧值覆盖」。`nickname`/`email`/`phone` 不纳入守卫——它们本就走队列推送，且 Phase 58 已删除全部直推路径，扩大拦截面会无谓拉长 `syncUserAttributes` 的改动范围。
  — **Reversibility:** reversible — 若日后要扩到四字段，只需在守卫判定处加字段集合。

- **D-02: 队列查询复用 `ADPushQueueActivePredicate` 常量**，拼成 `entity_type='user' AND action_type='update_user' AND <ADPushQueueActivePredicate>`。
  — **Reversibility:** one-way — 该常量是 `migration_215` 部分唯一索引 DDL 与 `adPushOnConflictClause` 的**单一事实源**，模型注释明写「谓词文本不得改写、不得加表名限定」；改写会同时破坏 ON CONFLICT 推断与排水器拾取。

- **D-03: 只查 EXISTS（`LIMIT 1`）**，不关心有几行活跃队列行。守卫需要的是布尔信号，多读无益。

- **D-04: 插入位置 = `internal/services/system/user_sync_service.go` 仅包住 `Update("dept_id", deptID)` 调用**（当前约 :426）。`resolveDeptFromOU`、`getDefaultDeptID`、`catLocalSameName` 分支全部不动。
  — 明确**不**把守卫提到 `SyncADUser` 入口或 `resolveDeptFromOU` 之前：那会让日志无法区分「没解析出部门」与「被守卫拦下」。

- **D-05: WARN 带全量诊断字段** —— `userID` / `username` / 本地 `dept_id` / AD 解析出的 `deptID` / 队列行 `id` / `action_type` / `attempts`。目标是运维看一条日志就能判断「本来要改成什么」与「为什么拦」。

- **D-06: 日志用 `Warnf`**（与 ROADMAP SC1 字面一致），不加固定前缀。

- **D-07: 守卫查队列失败（DB 瞬时故障）时，跳过 `dept_id` 覆盖并记 WARN。** 保守优先：宁可 DB 故障期间不同步部门，也绝不在故障窗口里回滚用户的本地编辑——后者正是本 phase 要防的事。
  — **Reversibility:** reversible — 降级策略是单点分支，改回「按现状覆盖」只动一处。

- **D-08: 不误伤正常同步的保证方式** —— 无活跃队列行时走**与现在完全相同的代码路径**（同一个 `Update` 调用），只多一次 EXISTS 查询。测试必须锁定「无队列行 → dept 照常更新」，且断言更新后的值等于 AD 解析值。
  — 明确**不**封装成独立的 `tryUpdateDept` 平行分支：两条路径会漂移。

### 对账视图（GUARD-02）

- **D-09: 单视图 `v_dept_ou_mismatch`，UNION ALL 四类口径**，每行带 `mismatch_type` 区分；部门级问题（软删/停用/跨配置）的 `user_id` 为 NULL。
  — **Reversibility:** costly — 视图一旦被端点与前端 Tab 依赖，拆分或改列名需同步改三处（VIEW DDL / 端点 SELECT / 前端列定义）。

- **D-10: 四类「不一致」口径**
  1. `user_ou_mismatch` — 用户 `ad_ou_dn` 与所属部门的映射 `ou_dn` 不相等（**归一化比较**）
  2. `dept_soft_deleted` — 映射行的 `dept_id` 指向 `sys_dept.deleted_at IS NOT NULL` 的部门
  3. `mapping_disabled` — 映射 `sync_enabled=false` 但该部门仍有用户
  4. `config_unrouted` — 用户 `ad_ou_dn` 无法归属任何 `status=enabled` 配置的 `base_dn` 后缀

- **D-11: 视图列集** —— `dept_id` / `dept_name` / `dept_deleted_at` / `ad_config_id` / `config_name` / `mapped_ou_dn` / `user_id` / `username` / `user_ad_ou_dn` / `mismatch_type` / `mismatch_detail`。

- **D-12: 第四类（`config_unrouted`）在视图内用纯 SQL 实现**，与其余三类同在 UNION ALL 里。最长后缀匹配逻辑因此在 Go（`MatchADConfigByDN`）与 SQL 各有一份。
  — **必须由测试锁定两者口径一致**：这是本 phase 唯一一处「同一规则两份实现」的地方，是未来漂移的高风险点。
  — **Reversibility:** costly — 若日后改为端点层 Go 补查，视图、分页、前端三处都要改。

- **D-13: 普通 SQL VIEW，实时计算，禁 MV。** 不使用后端缓存、不加 TTL。

### 端点与前端

- **D-14: 端点 `POST /system/ad-config/reconciliation/mismatch`**，挂在现有 `ad-config` 路由组下（与配置/账号池同组）。
  — **Reversibility:** costly — 路径是前端与外部调用方可见的契约。

- **D-15: 权限点 `ops:ad:config:reconciliation:list`**，与现有 `ops:ad:config:account:list` 同构（三级命名 + `list`）。

- **D-16: POST 分页 + 筛选** —— `current`/`pageSize` + 可选 `mismatchType`/`deptId`/`keyword`，返回 `PageData`。与全仓分页惯例一致。

- **D-17: 前端嵌入现有 AD 配置页新增「对账」Tab**，不新建独立页面、**不新增菜单 seed**。Tab 内用 `hasPerm` 判断是否渲染内容。
  — 后端端点独立鉴权（D-15）；**需提醒管理员为角色授予新权限点**，否则勾选 AD 配置菜单后仍看不到 Tab 内容。这是本 phase 部署时的一个已知操作项，不是代码缺陷。

### Claude's Discretion

- 守卫查询的具体封装位置（`UserSyncService` 方法 vs 包内私有函数）
- 四类口径的 `mismatch_detail` 文本格式
- VIEW DDL 中归一化比较的具体 SQL 写法（大小写/空格处理需与 `NormalizeDN` 的 Go 口径对齐，但不必共享代码）
- 前端 Tab 的表格列宽、筛选控件形态
- migration 编号（**Phase 58 已用到 migration_216，本 phase 从 217 起**）

### Folded Todos

（无匹配 todos 折叠进本 phase——`operlog-exclude-paths` 与 `v1.17-reconciliation-decisions` 两条匹配项均与本 phase 的 dept↔OU 对账无关，属误匹配）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与路线图
- `.planning/ROADMAP.md` — Phase 59 条目：Goal、SC1~SC4、Notes「禁用 MV」、Depends on Phase 57
- `.planning/REQUIREMENTS.md:17-18` — GUARD-01 / GUARD-02 原文
- `.planning/PROJECT.md` — v1.21 关键决策段（时间戳仲裁死刑 / 禁新旧值比较 / 禁 sys_job）

### 守卫改造对象
- `internal/services/system/user_sync_service.go:400-435` — `SyncADUser` 的部门解析与回写段（D-04 的插入点，当前 `Update("dept_id", deptID)` 在约 :426）
- `internal/services/system/user_sync_service.go:375-380` — `getDefaultDeptID`
- `internal/services/system/user_sync_service.go:787-790` — `resolveDeptFromOU` 语义注释

### 队列信号源（守卫的判定依据）
- `internal/models/ad_push_queue.go:50-57` — `ADPushQueueActivePredicate` **单一事实源常量**（D-02 红线：不得改写）
- `internal/models/ad_push_queue.go:59-90` — `ADPushQueue` 模型字段
- `internal/core/db/migrations/migration_215_*.go` — 部分唯一索引 DDL（谓词的另一处消费方）
- `internal/services/addomain/ad_push_queue_service.go` — 入队语义与合并子句（理解「活跃行」何时产生/消失）

### 对账视图数据源
- `internal/models/dept_ou_mapping.go` — `sys_dept_ou_mapping` 字段（`dept_id` / `ad_config_id` / `ou_dn` / `sync_enabled` / 唯一键 `(dept_id, ad_config_id)`）
- `internal/models/user.go:20,34-35` — `DeptID` / `AdDn` / `AdOuDn`
- `internal/models/dept.go` — `BaseModel`（含 `DeletedAt`，第三类口径依赖）
- `internal/services/addomain/ad_config_resolver.go` — `MatchADConfigByDN`（第四类口径的 Go 版实现，SQL 版必须与之口径一致）

### DN 归一化口径
- `internal/services/addomain/dn_util.go` — `NormalizeDN`（attr 名与值双侧 ToLower + 去空格，**仅用于比较**）；`BuildOUDN`（RFC4514 转义）
- `internal/services/addomain/ldap_client.go:429+` — `extractRDNFromDN`（RFC4514 正确的 RDN 切分，Phase 58 quick 任务刚修）

### 先例
- `internal/api/v1/system/ad_domain_router.go:24-60` — 权限组模式（`ops:ad:config:*`）
- `internal/core/db/migrations/migration_199/212/215` — 双方言（PG/sqlite）迁移先例
- `.planning/phases/58-compare-and-move/58-CONTEXT.md` — D-02 配置路由契约（最长后缀唯一匹配）

### Phase 58 遗留（影响本 phase 的前提）
- `.planning/phases/58-compare-and-move/58-GOAL-VERIFICATION.md` — ACHIEVED WITH CONDITIONS；9 项人工 UAT 未跑；H-03 DEFERRED
- `.planning/phases/58-compare-and-move/58-HUMAN-UAT.md` — 部署前置：目标环境须先跑过部门同步

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `ADPushQueueActivePredicate` 常量 — 守卫查询直接拼接，零重复定义
- `MatchADConfigByDN`（`ad_config_resolver.go`）— 第四类口径的 Go 参考实现
- `NormalizeDN`（`dn_util.go`）— 第一类口径的归一化语义来源
- `ad_domain_router.go` 的权限组模式 — 端点权限注册照抄
- 既有 `PageData` / `BaseListRequest` 分页基建 — 端点直接复用

### Established Patterns
- Handler-Service 分层：对账端点 handler 在 `api/v1/system/`，查询逻辑在 service 层
- 双方言迁移：VIEW DDL 需同时覆盖 PG 与 sqlite（sqlite 的 VIEW 语法基本兼容，但 `ILIKE` 等 PG 专有语法不可用）
- 权限命名空间 `ops:ad:config:*` — 新权限点必须同构
- operlog 约定：只读端点**不需要** `operlog.Record`（写操作才需要）

### Integration Points
- `user_sync_service.go:426` — 守卫插入点（唯一生产改动点）
- `ad_push_queue` 表 — 守卫的查询源（只读，不写）
- `sys_dept_ou_mapping` + `sys_user` + `sys_dept` + `sys_ad_config` — 视图的四表 JOIN 源
- `ad_domain_router.go` — 端点注册点
- 前端 AD 配置页 — Tab 嵌入点

</code_context>

<specifics>
## Specific Ideas

- **部署操作项（非代码）**：管理员需为相关角色授予 `ops:ad:config:reconciliation:list`，否则 Tab 内容不渲染。应写入 UAT / 部署清单。
- **两份实现的一致性**：第四类口径在 Go 与 SQL 各有一份最长后缀匹配，必须用测试锁定两者在边界输入上结论一致（父子域共享后缀 `DC=na,DC=corp,DC=com` 是 Phase 58 已确立的关键反例）。
- 用户对 Phase 58 的复审节奏表示认可（0 BLOCKER 后仍修了 4 个 HIGH + 3 个 MISSED），本 phase 预期沿用同样的「执行 → 复审 → 反向验证」闭环。

</specifics>

<deferred>
## Deferred Ideas

- **对账数据的自动修复**（一键把用户搬到映射 OU / 重建软删部门的映射）— 新能力，属独立 phase。
- **`sys_user.ad_config_id` 用户级归属** — Phase 58 已记 Deferred，本 phase 不重提。
- **登录路径不查 `user.Status`** — Phase 57 遗留独立 bug，候选后续 phase。
- **H-03 退避窗口矛盾**（业务失败路径 15min vs D-05 声明 ~61min）— Phase 58 DEFERRED，等真实熔断数据。

### Reviewed Todos (not folded)
- `operlog-exclude-paths.md` — 「operlog.exclude_paths 配置驱动白名单」：匹配度 0.6 但纯属关键词误匹配（pending/notes），与 dept↔OU 对账无关。
- `v1.17-reconciliation-decisions.md` — 「v1.17 资产对账」：匹配度 0.4，是**资产**对账而非**部门**对账，领域不同。

</deferred>

---

*Phase: 59-登录回写守卫与部门对账视图*
*Context gathered: 2026-10-19*
