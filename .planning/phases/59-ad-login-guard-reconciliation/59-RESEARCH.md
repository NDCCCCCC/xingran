# Phase 59: 登录回写守卫与部门对账视图 - Research

**Researched:** 2026-10-19
**Domain:** Go 1.24 / GORM / PostgreSQL + SQLite / Gin / Ant Design React
**Confidence:** HIGH（代码事实）/ MEDIUM（SQL 方言细节与权限落地点）

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### 守卫判定（GUARD-01）

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

#### 对账视图（GUARD-02）

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

#### 端点与前端

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

### Deferred Ideas (OUT OF SCOPE)

- **对账数据的自动修复**（一键把用户搬到映射 OU / 重建软删部门的映射）— 新能力，属独立 phase。
- **`sys_user.ad_config_id` 用户级归属** — Phase 58 已记 Deferred，本 phase 不重提。
- **登录路径不查 `user.Status`** — Phase 57 遗留独立 bug，候选后续 phase。
- **H-03 退避窗口矛盾**（业务失败路径 15min vs D-05 声明 ~61min）— Phase 58 DEFERRED，等真实熔断数据。

### Reviewed Todos (not folded)

- `operlog-exclude-paths.md` — 「operlog.exclude_paths 配置驱动白名单」：匹配度 0.6 但纯属关键词误匹配（pending/notes），与 dept↔OU 对账无关。
- `v1.17-reconciliation-decisions.md` — 「v1.17 资产对账」：匹配度 0.4，是**资产**对账而非**部门**对账，领域不同。
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GUARD-01 | AD 登录写 dept 前，若该用户存在 pending/failed 的 dept 队列行则跳过覆盖 + WARN | 插入点已定位（`user_sync_service.go:426`）；活跃谓词单一事实源已确认（`ADPushQueueActivePredicate`）；`idx_ad_push_queue_entity_id` 已由 migration_215 建好；WARN 可通过 `applogger.SetTestBuffer` 断言 |
| GUARD-02 | `v_dept_ou_mismatch` 普通 SQL VIEW + 只读端点，展示 dept↔OU 不一致及"映射指向软删部门"条目 | 四类口径的 SQL 骨架、双方言差异、Go/SQL 一致性测试方案、端点分页/鉴权/权限落地路径均已核实 |
</phase_requirements>

## Summary

Phase 59 是两条相对独立的工作流：

1. **GUARD-01 是单点行为守卫**。`SyncADUser` 在第 426 行用 `s.db.WithContext(ctx).Model(user).Update("dept_id", deptID)` 无条件把 AD 解析出的部门写回本地；守卫只需在这条 `Update` 之前查一次 `ad_push_queue` 是否存在 `entity_type='user' AND action_type='update_user' AND status IN ('pending','failed')` 的行。`migration_215` 已经为这个查询建了 `idx_ad_push_queue_entity_id`，查询成本是单索引点查。无队列行时仍走**同一个** `Update` 调用，不引入平行分支。

2. **GUARD-02 是「数据库视图 + 只读端点 + 前端 Tab」三段式**。关键难点不在端点，而在视图的**双方言可移植性**与**与 Go `MatchADConfigByDN` 的口径一致性**。已确认：PG 与 sqlite 都能表达 `WITH`、`SUBSTR`、`LENGTH`、`||`、关联子查询与 `LIMIT 1`（仓库内 `sqlite_reconciliation_views.go` 已有等价先例）；真正的方言分叉只有两处——PG 用 `CREATE OR REPLACE VIEW`，sqlite 必须 `DROP VIEW IF EXISTS` + `CREATE VIEW`；布尔列 `sync_enabled` 在 PG 是 boolean、在 sqlite 是 INTEGER，必须用 `NOT sync_enabled` 而不是 `= 0`。此外，UNION ALL 各分支的 ID 列类型不一致（PG `uuid` vs `varchar`）会直接报错，必须在每个分支显式 `CAST(... AS TEXT)`。

**Primary recommendation:**

- GUARD-01：在 `UserSyncService` 增加一个私有查询方法（返回队列行供 WARN 使用），在 `else if deptID != ""` 的 `Update` 之前插入「查询失败 → WARN + 跳过」「命中 → WARN + 跳过」「未命中 → 原 `Update`」三分支。谓词用 `Where(models.ADPushQueueActivePredicate)`，不改写常量。
- GUARD-02：migration 217 用「共享 SQL body + 双方言 wrapper」创建 `v_dept_ou_mismatch`；服务层 `Table("v_dept_ou_mismatch")` 分页查询；端点注册在现有 `adDomain` 组下的 `/reconciliation/mismatch`；前端在 `configs/index.tsx` 外包一层 `Tabs`，新增 `ReconciliationTab`。
- **必须在规划期解决的两个决策缺口**：① 权限点 `ops:ad:config:reconciliation:list` 若没有 `sys_menu` 行，前后端都无法识别（后端权限 SQL JOIN `sys_menu.perms`，前端 `permissions` 数组同样来自 `sys_menu`）——D-17 的「不新增菜单 seed」需要明确解释为「不新增可导航菜单/路由」，否则必须新增一条 `menu_type='F'` 按钮权限行；② D-14 字面路径 `/system/ad-config/...` 与真实挂载 `/ad-domain` 不一致，实际可落地路径是 `/api/v1/ad-domain/reconciliation/mismatch`。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 登录回写守卫判定（GUARD-01） | API / Backend（`system.UserSyncService`） | Database（`ad_push_queue` 只读查询） | 守卫是登录同步路径上的业务判断，必须在 `Update` 调用点就近执行；排队状态是数据库事实 |
| 活跃队列信号存储 | Database / Storage | — | `ad_push_queue` + `uq_ad_push_queue_active` 部分唯一索引是单一事实源 |
| dept↔OU 不一致计算（GUARD-02） | Database / Storage（普通 SQL VIEW） | — | D-12/D-13 锁定视图内实时计算；服务层只做过滤/分页 |
| 对账查询鉴权与分页 | API / Backend | Middleware（JWT + `RequirePermissions`） | 沿用 `adDomain` 路由组的鉴权链；分页用 `query.NormalizePagination` + `response.Page` |
| 对账结果展示 | Browser / Client | — | 只读 Tab，无写操作、无自动修复 |
| 权限点授予 | Database / Storage（`sys_menu` + `sys_role_menu`） | — | 权限字符串必须作为 `sys_menu.perms` 存在，才能被角色勾选和 `/system/my-menus/permissions` 返回 |

## Standard Stack

本 phase **不引入任何新依赖**。全部能力均由既有栈完成。

### Core（既有，不新增）
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.24.5（`go version` 实测） | 后端 | 项目既定 |
| GORM | v1.30.5 | 视图查询 / 队列行查询 | 项目唯一 ORM；`Table("v_...")` 可直接读视图 |
| Gin | v1.10.0 | 新端点 | 项目唯一 HTTP 框架 |
| glebarez/sqlite | 项目既有测试依赖 | 视图/守卫的 sqlite 测试 | 现有测试基座全部使用它 |
| antd | 6.6.4（UI-SPEC 记录） | Tabs / Table / Form / Select | 项目唯一组件库 |
| vitest | 项目既有 | 前端测试 | `npm run test` |

### 需要额外使用的仓库内既有设施
| 设施 | 位置 | 用途 |
|------|------|------|
| `models.ADPushQueueActivePredicate` | `internal/models/ad_push_queue.go:53` | 守卫查询谓词，唯一事实源 |
| `NormalizeDN` | `internal/services/addomain/dn_util.go:36` | 第一类口径的 Go 语义参照 |
| `MatchADConfigByDN` | `internal/services/addomain/ad_config_resolver.go:39` | 第四类口径的 Go 语义参照 |
| `query.NormalizePagination` | `pkg/query/pagination.go:43` | 分页钳制 |
| `response.Page` | `pkg/response/response.go:123` | `PageData` 响应 |
| `middleware.RequirePermissions` | `pkg/middleware/permission.go:277` | 新权限点 |
| `applogger.SetTestBuffer` | `pkg/logger/logger.go:225` | 断言 WARN 日志字段 |

**Installation:** 无。

## Package Legitimacy Audit

本 phase **不安装任何外部包**，无 npm/PyPI/crates 新增依赖。无需运行 Package Legitimacy Gate。

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| — | — | — | — | — | — | 无新增依赖 |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                    ┌──────────────────────────────────────────────┐
AD 登录 (LDAP bind)  │  security.ADAuthenticator                     │
                    │    └─ SyncADUser(ctx, adUserInfo, roleID)     │
                    └───────────────┬──────────────────────────────┘
                                    ▼
              ┌────────────────────────────────────────────────────┐
              │ system.UserSyncService.SyncADUser                  │
              │  1) SyncUserFromAD —— 事务内写 sys_user 基本属性     │
              │  2) resolveDeptFromOU —— sys_dept_ou_mapping 查询   │
              │  3) GUARD-01（新增）                                │
              │       EXISTS ad_push_queue(                         │
              │         entity_id=userID,                          │
              │         entity_type='user',                        │
              │         action_type='update_user',                 │
              │         status IN ('pending','failed'))            │
              │     ├─ 命中 / 查询失败 → WARN + 跳过 dept_id 更新    │
              │     └─ 未命中        → 原 Update("dept_id", deptID)│
              └───────────────┬────────────────────────────────────┘
                              ▼
                 ┌──────────────────────────┐
                 │ sys_user.dept_id（本地编辑）│
                 └──────────────────────────┘

对账链路：
前端「对账」Tab
  └─ getDeptOUMismatchList()
       └─ POST /api/v1/ad-domain/reconciliation/mismatch
            └─ JWTAuth → RequirePermissions(ops:ad:config:reconciliation:list)
                 └─ ReconciliationService.ListDeptOUMismatch
                      └─ SELECT ... FROM v_dept_ou_mismatch
                           ├─ sys_user          (ad_ou_dn / dept_id / username)
                           ├─ sys_dept          (dept_name / deleted_at)
                           ├─ sys_dept_ou_mapping (dept_id / ad_config_id / ou_dn / sync_enabled)
                           └─ sys_ad_config     (config_name / base_dn / status)
```

### Recommended Project Structure

```
internal/
├── models/
│   └── ad_push_queue.go                     # 不改；复用 ADPushQueueActivePredicate
├── services/
│   ├── system/
│   │   ├── user_sync_service.go             # GUARD-01（唯一生产改动点 ~:426）
│   │   └── user_sync_service_phase59_test.go# 守卫行为 + WARN 断言
│   └── addomain/
│       ├── dept_ou_mismatch_service.go      # 【新】对账查询 service（Table("v_...")）
│       └── dept_ou_mismatch_service_test.go # 【新】分页/筛选/一致性
├── api/v1/system/
│   ├── ad_reconciliation_handler.go         # 【新】只读 handler（构造 service）
│   ├── ad_reconciliation_handler_test.go    # 【新】绑定错误/分页默认值
│   └── ad_domain_router.go                  # 注册 /reconciliation/mismatch
└── core/db/
    ├── migrations/
    │   ├── migration_217_dept_ou_mismatch_view.go       # 【新】双方言 VIEW
    │   └── migration_217_dept_ou_mismatch_view_test.go  # 【新】sqlite 建视图 + 四类口径
    ├── database.go                          # 三条启动路径注册 + AutoMigrate 前 DROP
    └── sqlite_reconciliation_views.go       # AutoMigrate 前 DROP 列表（sqlite）

ruoyi-react-frontend/src/
├── lib/adDomainApi.ts                       # 新增 getDeptOUMismatchList
├── lib/adDomainApi.test.ts                  # 新增路径/默认分页断言
└── pages/ad-domain/configs/
    ├── index.tsx                            # 外包 Tabs（配置列表 / 对账）
    ├── ReconciliationTab.tsx                # 【新】只读表格 + 筛选
    └── __tests__/ReconciliationTab.test.tsx # 【新】权限门/空态/错误态
```

### Pattern 1: 队列信号守卫（GUARD-01）

**What:** 在唯一的 `dept_id` 写回点前查询活跃队列行；任何命中或查询失败都不写。

**When to use:** 仅 `category != catLocalSameName`、`OUDN != ""`、`ouMapper != nil`、`deptID != ""` 的分支。

**Example（与仓库现有代码逐字对齐的骨架）：**

```go
// 插入点：internal/services/system/user_sync_service.go:419-430 的 else-if 链内
} else if deptID == "" {
    applogger.Infof("[SyncADUser] 无OU映射，保留原部门: userID=%s, deptID=%v", user.ID, user.DeptID)
} else {
    // GUARD-01: D-02 复用单一事实源常量，不改写、不加表名限定
    var qrow models.ADPushQueue
    qerr := s.db.WithContext(ctx).
        Where("entity_id = ? AND entity_type = ? AND action_type = ?",
            user.ID, models.ADPushEntityUser, models.ADPushActionUpdateUser).
        Where(models.ADPushQueueActivePredicate).
        Take(&qrow).Error

    switch {
    case qerr != nil && !errors.Is(qerr, gorm.ErrRecordNotFound):
        // D-07：查询失败 → 保守跳过并 WARN
        applogger.Warnf("部门回写守卫查询失败，跳过 dept_id 更新: userID=%s, username=%s, localDeptID=%s, adDeptID=%s, err=%v",
            user.ID, user.Username, deptValue(user.DeptID), deptID, qerr)
    case qerr == nil:
        // D-05：全量诊断字段
        applogger.Warnf("部门回写被活跃队列守卫拦截: userID=%s, username=%s, localDeptID=%s, adDeptID=%s, queueID=%s, actionType=%s, attempts=%d",
            user.ID, user.Username, deptValue(user.DeptID), deptID, qrow.ID, qrow.ActionType, qrow.Attempts)
    default:
        // D-08：无队列行 → 与现状完全相同的 Update 调用
        if err := s.db.WithContext(ctx).Model(user).Update("dept_id", deptID).Error; err != nil {
            applogger.Warnf("更新用户部门失败: %v", err)
        } else {
            applogger.Infof("[SyncADUser] 成功设置用户部门: userID=%s, deptID=%s", user.ID, deptID)
        }
    }
}
```

`deptValue` 是建议的小 helper：`user.DeptID` 是 `*string`，**直接 `%v` 会打印指针地址**（现有 `:418`/`:425` 的日志有此隐患），D-05 要求可读的本地 dept 值。

**测试锁定点：** `pending` / `failed` → 拦截；`pushed` / `dead` → 不拦截（D-02 谓词字面）；`disable` 动作 → 不拦截；无行 → 更新；查询失败（DROP 表模拟）→ 跳过。

### Pattern 2: 双方言普通 VIEW 迁移（GUARD-02）

**What:** 用一份共享的 `SELECT` body + 两个方言 wrapper 建视图；在 `AutoMigrate` 之前 DROP，之后重建。

**verified facts:**

- sqlite **没有** `CREATE OR REPLACE VIEW`——仓库注释逐字写明：`DROP VIEW IF EXISTS + CREATE VIEW(sqlite 无 CREATE OR REPLACE VIEW;` [VERIFIED: internal/core/db/sqlite_reconciliation_views.go:30]
- PG 普通 VIEW 也会阻塞 GORM AutoMigrate 的 `ALTER COLUMN TYPE`，历史错误为 `cannot alter type of a column used by a view or rule` → SQLSTATE 0A000 [VERIFIED: internal/core/db/database.go:439-448]
- sqlite 的 `ALTER TABLE` 走 `__temp` + `RENAME` 路径，视图引用会让 RENAME 失败；所以 sqlite 分支也在 AutoMigrate 前 DROP 视图 [VERIFIED: internal/core/db/database.go:818-832]
- sqlite 视图内可以使用 `WITH`、`SUBSTR`、`LIKE`、`||` [VERIFIED: internal/core/db/sqlite_reconciliation_views.go:174-220 与 :56-65]
- PG 视图用 `CREATE OR REPLACE VIEW` [VERIFIED: internal/core/db/migrations/migration_175_reconciliation_physical_link.go:73,122]

**建议的迁移结构：**

```go
func Migrate217DeptOUMismatchView(db *gorm.DB) error {
    body := buildDeptOUMismatchViewSelect() // 共享 SQL body（无方言分叉）
    if isPostgreSQL(db) {
        return db.Exec("CREATE OR REPLACE VIEW v_dept_ou_mismatch AS " + body).Error
    }
    // sqlite: 瞬时元数据操作，先 DROP 保证定义随代码版本刷新
    if err := db.Exec("DROP VIEW IF EXISTS v_dept_ou_mismatch").Error; err != nil {
        return err
    }
    return db.Exec("CREATE VIEW v_dept_ou_mismatch AS " + body).Error
}
```

**注册位置（三处，缺一不可，参照 Phase 57 F-03 盲区）：**

1. `database.go` PG 分支（advisory-lock 块内、`Migrate216` 之后）
2. `database.go` sqlite 分支
3. `database.go` `BootstrapMissingTables()`（`SKIP_AUTOMIGRATE=true` 旁路）

**AutoMigrate 前置 DROP：**

- PG：`dropDependentMaterializedViews()` 当前是 noop；新增一行定向 `DROP VIEW IF EXISTS v_dept_ou_mismatch`（该视图无 MV 依赖，可安全 DROP，不需要 CASCADE）。
- sqlite：在 `dropSQLiteReconciliationViews()`（`sqlite_reconciliation_views.go:113`）中追加同一行，或新增同名 helper 并在 `AutoMigrate()` 的 sqlite 分支调用。

**Why:** 该视图引用 `sys_user` / `sys_dept` / `sys_dept_ou_mapping` / `sys_ad_config`。只要未来任一模型 tag 触发列类型变更，不预先 DROP 就会在启动期 FATA（PG）或 RENAME 失败（sqlite）。这是本 phase 最容易被漏掉的运维风险。

### Pattern 3: 只读分页端点

**What:** 照抄 `ADLoginHistoryHandler` 的只读形态：handler 内部构造 service、`response.Page` 返回、不调用 `operlog.Record`。

**Example:**

```go
type ADReconciliationHandler struct {
    svc addomainServices.DeptOUMismatchService
}

func NewADReconciliationHandler(db *gorm.DB) *ADReconciliationHandler {
    return &ADReconciliationHandler{svc: addomainServices.NewDeptOUMismatchService(db)}
}

func (h *ADReconciliationHandler) ListDeptOUMismatch(c *gin.Context) {
    var req addomainServices.DeptOUMismatchQuery
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
        return
    }
    req.Current, req.PageSize = query.NormalizePagination(req.Current, req.PageSize)
    list, total, err := h.svc.ListDeptOUMismatch(c.Request.Context(), &req)
    if err != nil {
        response.Error(c, err)
        return
    }
    response.Page(c, list, total, req.Current, req.PageSize)
}
```

**路由注册（`ad_domain_router.go`）：**

```go
recon := r.Group("/reconciliation")
recon.Use(middleware.RequirePermissions([]string{"ops:ad:config:reconciliation:list"}, core))
{
    recon.POST("/mismatch", NewADReconciliationHandler(core.GetDB()).ListDeptOUMismatch)
}
```

`RequirePermissions` 是「任一权限」语义 [VERIFIED: pkg/middleware/permission.go:292-298]；`adDomain` 组已统一挂 `JWTAuth` 与 `OperLogMiddleware` [VERIFIED: internal/api/router.go:519-524]，只读 handler 不设置 `oper_log_title` 即不会写操作日志 [VERIFIED: pkg/middleware/oper_log.go:86-89]。

### Anti-Patterns to Avoid

- **改写 `ADPushQueueActivePredicate` 或给它加表名限定**：会同时破坏 migration_215 的部分唯一索引推断与 `ON CONFLICT TargetExpr`，PG 直接 42P10。D-02 红线。
- **把守卫包成 `tryUpdateDept` 平行分支**：两条路径会漂移（D-08 禁止）。
- **用时间戳仲裁 / 比较新旧值**：PROJECT.md 已判死刑。
- **用 MATERIALIZED VIEW**：ROADMAP 明令禁用；MV 刷新僵尸与 AutoMigrate 阻塞均有历史事故。
- **在视图里用 `ILIKE` / `REGEXP_REPLACE` / `::` 强转**：sqlite 不可用。用 `LOWER(...)` + `CAST(... AS TEXT)` + `SUBSTR`。
- **UNION ALL 各分支 ID 列不 CAST**：PG 会报 `UNION types uuid and character varying cannot be matched`。
- **分页查询不带确定性 ORDER BY**：视图无固有顺序，翻页会重复/漏行。
- **前端直接用 `axios`**：必须走 `import { post } from "@/lib/api"`。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 活跃队列判定 | 新写一段 `status = 'pending' OR status = 'failed'` | `models.ADPushQueueActivePredicate` | 已在 migration_215 索引 DDL 与 ON CONFLICT 中共用；漂移即 42P10 |
| DN 归一化（Go 侧） | 在调用点手写 `strings.ToLower` + `TrimSpace` | `addomain.NormalizeDN` | 值侧 ToLower + 每段 trim 的语义已有 Phase 58 契约与测试 |
| 最长后缀配置匹配（Go 侧） | 裸 `strings.HasSuffix` | `addomain.MatchADConfigByDN` | 逗号边界、空 base、等长歧义均已处理 |
| 分页参数 | 手写 `if current <= 0` | `query.NormalizePagination` | 全仓唯一分页归一化核心，上限钳制一致 |
| 分页响应 | 手写 `gin.H{"list":..., "total":...}` | `response.Page` | 统一 `code/message/data/timestamp/request_id` |
| 权限校验 | handler 内自己查角色 | `middleware.RequirePermissions` | 含超管短路、30s 结果缓存与菜单继承语义 |
| DN 拼接 / 转义 | 手写 RFC4514 转义 | `addomain.BuildOUDN` / `ldap.EscapeDN` | 特殊字符转义已由 HYGIENE-02 收敛 |

## Runtime State Inventory

> 本 phase 不是重命名/重构，但有**新增数据库对象**与**权限种子**两类运行时状态；以下为部署相关清单。

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | 无数据迁移：GUARD-01 只读 `ad_push_queue` 并阻止写 `sys_user.dept_id`；GUARD-02 只读四张表 | 无 |
| 新增 DB 对象 | `v_dept_ou_mismatch` 普通 VIEW（migration_217 每次启动重建） | 迁移注册到 PG / sqlite / `BootstrapMissingTables` 三条路径；AutoMigrate 前先 DROP |
| 权限/菜单注册 | `ops:ad:config:reconciliation:list` 必须作为 `sys_menu.perms` 行存在，才会被 `GetUserPermissions` 返回、也才能被 `sys_role_menu` 授予 | **待决策**：新增 `menu_type='F'` 按钮行（推荐）或写入运维手册要求管理员手建 |
| 角色授权 | 已有 `AD域控配置` 菜单的角色不会自动获得新权限点 | 迁移中按 `sys_role_menu` 父菜单精确授权（参照 memory `migration-grant-new-menu-precision-helper`），或在部署清单中列明 |
| Live service config | 无外部服务配置变更 | 无 |
| OS-registered state | 无 | 无 |
| Secrets/env vars | 无 | 无 |
| Build artifacts | 前端 `ruoyi-react-frontend` 的 `dist` 在部分部署形态下会被 Go 二进制嵌入（仓库存在 `internal/server/ruoyi-react-frontend/dist` 与 `//go:embed` 构建标签） | 前端改动后需 `npm run build`；若使用嵌入式部署，必须重新编译 Go 二进制并替换 |

## Common Pitfalls

### Pitfall 1: 视图的 UNION ALL 类型不一致（PG 专属，sqlite 测不出来）
**What goes wrong:** 第一类/第二类/第三类的 `dept_id`、`ad_config_id`、`user_id` 来自 `uuid` 列，第四类的 `dept_id`/`user_id` 来自 `sys_user` 的 `varchar`/`uuid`。PG 要求 UNION 各分支同类型，否则 `UNION types uuid and character varying cannot be matched`。
**Why it happens:** sqlite 全是 TEXT，本地测试永远绿；PG 启动/查询时才炸。
**How to avoid:** 视图**每个分支**的 `dept_id` / `ad_config_id` / `user_id` 一律 `CAST(x AS TEXT)`，`dept_deleted_at` 一律 `CAST(x AS TEXT)`。
**Warning signs:** sqlite 测试全绿但目标环境 endpoint 500，错误信息含 `UNION types`。

### Pitfall 2: 布尔列跨方言比较
**What goes wrong:** `WHERE sync_enabled = 0` 在 PG 报 `operator does not exist: boolean = integer`。
**Why it happens:** GORM `bool` 在 PG 是 `boolean`，在 sqlite 是 `INTEGER`。
**How to avoid:** 用 `WHERE NOT mp.sync_enabled`（PG 与 sqlite 都成立）。
**Warning signs:** sqlite 绿、PG 视图创建或查询时报 42883。

### Pitfall 3: 视图阻塞 AutoMigrate
**What goes wrong:** 下一次启动 GORM 对 `sys_user` / `sys_dept` 发 `ALTER COLUMN TYPE`，因视图引用而 0A000 FATA（PG）；sqlite 走 RENAME 路径时报 `error in view ... no such table`。
**Why it happens:** `AutoMigrate` 先于 migrations 执行，视图在两次启动之间一直存在。
**How to avoid:** AutoMigrate 前定向 `DROP VIEW IF EXISTS v_dept_ou_mismatch`（PG 与 sqlite 各一处），migration_217 在 AutoMigrate 后重建。
**Warning signs:** 启动日志在「所有表迁移成功」之前 FATA；错误含 `cannot alter type of a column used by a view or rule` 或 `error in view`。

### Pitfall 4: sqlite 没有 `CREATE OR REPLACE VIEW`
**What goes wrong:** 迁移在 sqlite 上语法错误。
**Why it happens:** sqlite 只支持 `CREATE VIEW` / `DROP VIEW`。
**How to avoid:** 方言 wrapper：PG `CREATE OR REPLACE VIEW`；sqlite `DROP VIEW IF EXISTS` + `CREATE VIEW`。
**Warning signs:** `near "REPLACE": syntax error`。

### Pitfall 5: 视图里的 LIKE 通配符污染 DN 匹配
**What goes wrong:** 用 `LIKE '%,' || base` 做后缀匹配时，DN 值里的 `_` / `%` 会被当成通配符，产生错误路由。
**Why it happens:** OU 名称常含下划线（`OU=IT_Support`）。
**How to avoid:** 用 `SUBSTR(norm_dn, LENGTH(norm_dn) - LENGTH(norm_base)) = ',' || norm_base` 配合 `LENGTH(norm_dn) > LENGTH(norm_base)`，避免 LIKE。关键字筛选若用 LIKE，需转义 `%`/`_` 并加 `ESCAPE`。
**Warning signs:** 对账结果出现不应出现的 `user_ou_mismatch` / `config_unrouted` 消失。

### Pitfall 6: `NormalizeDN` 的 SQL 等价式不完整
**What goes wrong:** 只做 `LOWER` 不 trim，会把 `OU=IT , DC=corp` 与 `OU=IT,DC=corp` 判为不等，产生假不一致。
**Why it happens:** Go `NormalizeDN` 对每段做 `TrimSpace`（包括 `=` 两侧）。
**How to avoid:** 用 Go helper 生成 SQL 归一化表达式（嵌套 `REPLACE` 折叠空格 + `TRIM` 分段 trim + `LOWER`），并用表驱动测试把 SQL 表达式与 `NormalizeDN` 逐输入比对。sqlite 没有 `REGEXP_REPLACE`，别用 regex。
**Warning signs:** 大小写相同但空格不同的 OU 被报为 mismatch。

### Pitfall 7: 守卫 WARN 打印指针地址
**What goes wrong:** `user.DeptID` 是 `*string`，`%v` 输出 `0xc000...`。
**Why it happens:** 现有 `:418`/`:425` 的日志已犯此错。
**How to avoid:** 用显式 deref helper（nil → `<nil>`）。
**Warning signs:** 日志里看不到本地部门 UUID。

### Pitfall 8: 权限点"存在"但不可授予
**What goes wrong:** 后端 `RequirePermissions` 对超管短路，看起来"能用"；但前端 `permissions.includes("ops:ad:config:reconciliation:list")` 恒 false → Tab 永远显示 403；普通角色也无法被授予。
**Why it happens:** 后端权限检查与前端权限数组都直接来自 `sys_menu.perms`（`checkUserPermission` JOIN `sys_menu`；`GetUserPermissions` `Pluck("perms")`）。没有菜单行就没有任何来源。
**How to avoid:** 迁移中新增 `menu_type='F'` 按钮权限行（父菜单为 `AD域控配置`），并给已拥有父菜单的角色精确授权。把 D-17 的「不新增菜单 seed」解释为「不新增可导航菜单/路由」，而不是「不新增权限载体」。
**Warning signs:** 管理员登录后 Tab 显示「无对账查看权限」；`SELECT * FROM sys_menu WHERE perms='ops:ad:config:reconciliation:list'` 为空。

### Pitfall 9: 端点路径契约与实际挂载不一致
**What goes wrong:** D-14 字面 `/system/ad-config/reconciliation/mismatch`；真实路由组是 `r.Group("/ad-domain")` [VERIFIED: internal/api/router.go:519]，前端 baseURL 为 `/api/v1` [VERIFIED: ruoyi-react-frontend/src/lib/api.ts:55]。
**Why it happens:** CONTEXT 的记录沿用了旧命名。
**How to avoid:** 统一为 `/api/v1/ad-domain/reconciliation/mismatch`，前端函数写 `/ad-domain/reconciliation/mismatch`；在 plan/UAT 中显式记录该对齐。
**Warning signs:** 前端 404，后端路由表里只有 `/ad-domain/...`。

### Pitfall 10: 分页无 ORDER BY
**What goes wrong:** 翻页结果重复或漏行。
**Why it happens:** 视图/UNION 无固有顺序。
**How to avoid:** 固定 `ORDER BY mismatch_type, dept_name, username, dept_id, user_id`（全部已 CAST 为文本）。
**Warning signs:** 不同页出现同一行。

### Pitfall 11: 守卫的竞态窗口与 `dead` 行
**What goes wrong:** ① 检查通过后、`Update` 之前恰好入队，仍可能覆盖一次；② `dead` 行不在活跃谓词内，守卫不会拦截，而 58-07 明确说 dead 是「系统认为推过了、AD 侧其实没有」的异常信号。
**Why it happens:** D-03 只做单次 EXISTS；D-02 锁定 pending/failed。
**How to avoid:** 两项都按 locked decisions 接受，并在 plan/文档中显式记录：竞态窗口是保守设计的一侧（误拦多于漏拦）；dead 行的可观测性本 phase **不做**（58-07 的建议与 D-02/D-03 冲突，留给后续 phase 或对账视图扩展）。
**Warning signs:** 若复审时有人要求"把 dead 也算进去"，需回到 CONTEXT 决策层，不在实现层私改。

## Code Examples

### 1) 守卫查询（唯一允许的写法）

```go
// Source: internal/models/ad_push_queue.go:53 + 既有 GORM 用法
err := s.db.WithContext(ctx).
    Where("entity_id = ? AND entity_type = ? AND action_type = ?",
        user.ID, models.ADPushEntityUser, models.ADPushActionUpdateUser).
    Where(models.ADPushQueueActivePredicate).
    Take(&row).Error
```

`models.ADPushQueueActivePredicate` 的逐字值：

```go
const ADPushQueueActivePredicate = "status IN ('pending', 'failed')"
```

[VERIFIED: internal/models/ad_push_queue.go:53]

### 2) 共享视图 body 的四类分支（骨架，列顺序必须与 D-11 一致）

```sql
-- 归一化表达式（由 Go helper 生成，示意）：
-- N(x) = LOWER(REPLACE(...TRIM(x)...))  -- 折叠空格 + 段 trim + lower
WITH cfg AS (
    SELECT id, config_name, base_dn,
           N(base_dn) AS nb, LENGTH(N(base_dn)) AS nbl
    FROM sys_ad_config
    WHERE status = 0 AND deleted_at IS NULL AND TRIM(COALESCE(base_dn, '')) <> ''
),
usr AS (
    SELECT id, username, dept_id, ad_ou_dn,
           N(ad_ou_dn) AS nu, LENGTH(N(ad_ou_dn)) AS nul
    FROM sys_user
    WHERE deleted_at IS NULL AND dept_id IS NOT NULL
      AND ad_ou_dn IS NOT NULL AND ad_ou_dn <> ''
),
mp AS (
    SELECT dept_id, ad_config_id, ou_dn, sync_enabled, N(ou_dn) AS nm
    FROM sys_dept_ou_mapping
),
mt AS (   -- 用户 OU 命中的配置（逗号边界 + 最长后缀）
    SELECT u.id AS user_id, c.id AS config_id, c.nbl
    FROM usr u JOIN cfg c
      ON u.nul > c.nbl
     AND SUBSTR(u.nu, u.nul - c.nbl) = ',' || c.nb
),
best AS (
    SELECT user_id, MAX(nbl) AS nbl FROM mt GROUP BY user_id
),
best_cnt AS (
    SELECT m.user_id, COUNT(*) AS cnt
    FROM mt m JOIN best b ON m.user_id = b.user_id AND m.nbl = b.nbl
    GROUP BY m.user_id
),
routed AS (
    SELECT b.user_id, m.config_id
    FROM best b
    JOIN best_cnt bc ON bc.user_id = b.user_id AND bc.cnt = 1
    JOIN mt m ON m.user_id = b.user_id AND m.nbl = b.nbl
)
-- ① user_ou_mismatch：路由配置下该部门的映射 OU ≠ 用户 OU
SELECT CAST(u.dept_id AS TEXT), d.dept_name, CAST(d.deleted_at AS TEXT),
       CAST(c.id AS TEXT), c.config_name, mp.ou_dn,
       CAST(u.id AS TEXT), u.username, u.ad_ou_dn,
       'user_ou_mismatch',
       '用户 OU「' || u.ad_ou_dn || '」≠ 部门映射 OU「' || mp.ou_dn || '」'
FROM routed r
JOIN usr u ON u.id = r.user_id
JOIN cfg c ON c.id = r.config_id
JOIN mp ON CAST(mp.dept_id AS TEXT) = u.dept_id
        AND CAST(mp.ad_config_id AS TEXT) = CAST(r.config_id AS TEXT)
LEFT JOIN sys_dept d ON CAST(d.id AS TEXT) = u.dept_id
WHERE mp.nm <> u.nu

UNION ALL
-- ② dept_soft_deleted：映射指向的部门已软删（部门级，user_id NULL）
SELECT CAST(mp.dept_id AS TEXT), d.dept_name, CAST(d.deleted_at AS TEXT),
       CAST(mp.ad_config_id AS TEXT), c.config_name, mp.ou_dn,
       NULL, NULL, NULL,
       'dept_soft_deleted',
       '部门「' || d.dept_name || '」已软删（' || CAST(d.deleted_at AS TEXT) || '），映射仍指向该部门'
FROM mp
JOIN sys_dept d ON d.id = mp.dept_id AND d.deleted_at IS NOT NULL
LEFT JOIN sys_ad_config c ON c.id = mp.ad_config_id

UNION ALL
-- ③ mapping_disabled：映射停用但部门仍有活跃用户（部门级）
SELECT CAST(mp.dept_id AS TEXT), d.dept_name, CAST(d.deleted_at AS TEXT),
       CAST(mp.ad_config_id AS TEXT), c.config_name, mp.ou_dn,
       NULL, NULL, NULL,
       'mapping_disabled',
       '部门「' || d.dept_name || '」的映射已停用，但仍有用户归属'
FROM mp
JOIN sys_dept d ON d.id = mp.dept_id
LEFT JOIN sys_ad_config c ON c.id = mp.ad_config_id
WHERE NOT mp.sync_enabled
  AND EXISTS (
      SELECT 1 FROM sys_user u2
      WHERE CAST(u2.dept_id AS TEXT) = CAST(mp.dept_id AS TEXT)
        AND u2.deleted_at IS NULL
  )

UNION ALL
-- ④ config_unrouted：用户 OU 无法归属任何启用配置（用户级）
SELECT CAST(u.dept_id AS TEXT), d.dept_name, CAST(d.deleted_at AS TEXT),
       NULL, NULL, NULL,
       CAST(u.id AS TEXT), u.username, u.ad_ou_dn,
       'config_unrouted',
       '用户 OU「' || u.ad_ou_dn || '」不匹配任何启用配置的 base_dn'
FROM usr u
LEFT JOIN sys_dept d ON CAST(d.id AS TEXT) = u.dept_id
WHERE NOT EXISTS (SELECT 1 FROM routed r WHERE r.user_id = u.id)
```

**注意：**

- `dept_id` / `ad_config_id` / `user_id` / `dept_deleted_at` 在**每个分支**都 CAST 为 TEXT（Pitfall 1）。
- `NOT mp.sync_enabled`（Pitfall 2）。
- `usr` CTE 只纳入 `dept_id` 非空的活跃用户——本视图的领域是「部门↔OU」对账；无部门用户不属于 D-10 的任何一类，需在 plan 中显式记录该边界。
- 分支 ② 与 ③ 可能对同一条映射行各出一行；这是 UNION ALL 的预期语义，需在 plan 中记录（若要求互斥，可在 ③ 增加 `AND d.deleted_at IS NULL`）。
- `routed` CTE 用 `cnt = 1` 复刻 `MatchADConfigByDN` 的「等长并列 → 歧义 → error」语义；这使 SQL 与 Go 在歧义输入上同样把用户判为 `config_unrouted`。若不复刻，一致性测试会在「同域重复配置」用例上失败。
- `user_ou_mismatch` 只覆盖「有映射但不等」；**没有映射**的用户不进入任何一类（D-10 未列该口径）。这是已知可观测性缺口，见 Open Questions。

### 3) 服务层分页与筛选

```go
// 查询入参（D-16）
type DeptOUMismatchQuery struct {
    Current      int    `json:"current"`
    PageSize     int    `json:"pageSize"`
    MismatchType string `json:"mismatchType,omitempty"` // 四值白名单
    DeptID       string `json:"deptId,omitempty"`
    Keyword      string `json:"keyword,omitempty"`
}

// 行出参（列名对齐 D-11 / UI-SPEC；ID 与时间统一用字符串，规避视图列类型跨方言差异）
type DeptOUMismatchRow struct {
    DeptID         *string `json:"deptId"`
    DeptName       *string `json:"deptName"`
    DeptDeletedAt  *string `json:"deptDeletedAt"`
    ADConfigID     *string `json:"adConfigId"`
    ConfigName     *string `json:"configName"`
    MappedOUDN     *string `json:"mappedOuDn"`
    UserID         *string `json:"userId"`
    Username       *string `json:"username"`
    UserADOUDN     *string `json:"userAdOuDn"`
    MismatchType   string  `json:"mismatchType"`
    MismatchDetail string  `json:"mismatchDetail"`
}
```

```go
// Source: 照抄 internal/services/addomain/login_history_query.go 的只读分页模式
rows := make([]DeptOUMismatchRow, 0) // 空结果必须序列化为 [] 而非 null
q := s.db.WithContext(ctx).Table("v_dept_ou_mismatch")
if req.MismatchType != "" {
    // 建议白名单校验，非法值直接忽略或 400
    q = q.Where("mismatch_type = ?", req.MismatchType)
}
if req.DeptID != "" {
    q = q.Where("dept_id = ?", req.DeptID)
}
if kw := strings.TrimSpace(req.Keyword); kw != "" {
    // escapeLike 是本 phase 需新增的小 helper（转义 \ % _），不是仓库既有函数
    like := "%" + escapeLike(strings.ToLower(kw)) + "%"
    q = q.Where(`LOWER(COALESCE(username,'')) LIKE ? ESCAPE '\'
              OR LOWER(COALESCE(dept_name,'')) LIKE ? ESCAPE '\'
              OR LOWER(COALESCE(mapped_ou_dn,'')) LIKE ? ESCAPE '\'
              OR LOWER(COALESCE(user_ad_ou_dn,'')) LIKE ? ESCAPE '\'`,
        like, like, like, like)
}
if err := q.Count(&total).Error; err != nil { return nil, 0, err }
err := q.Order("mismatch_type, dept_name, username, dept_id, user_id").
    Offset((req.Current - 1) * req.PageSize).Limit(req.PageSize).
    Scan(&rows).Error
```

### 4) Go / SQL 一致性测试（D-12 的核心交付）

```go
// 同一组配置：父子域、逗号边界、空 base、停用、软删、等长歧义
configs := loadEnabledConfigsFromDB(t, db) // status=enabled 且 deleted_at IS NULL
for _, tc := range cases {
    cfg, err := addomain.MatchADConfigByDN(tc.adOuDn, configs)
    goRouted := err == nil
    sqlRouted := viewHasConfigUnroutedRow(t, db, tc.userID) == false
    assert.Equal(t, goRouted, sqlRouted, "case=%s", tc.name)
    if goRouted {
        assert.Equal(t, cfg.ID, viewMatchedConfigID(t, db, tc.userID))
    }
}
```

必测用例（CONTEXT 点名父子域共享后缀）：

| 用例 | 配置 | 用户 OU | 期望 |
|------|------|---------|------|
| 最长后缀 | `DC=corp,DC=com` + `DC=na,DC=corp,DC=com` | `...,DC=na,DC=corp,DC=com` | 命中子域，非 unrouted |
| 逗号边界 | `DC=corp,DC=com` | `...,DC=xcorp,DC=com` | unrouted |
| 无匹配 | `DC=corp,DC=com` | `...,DC=other,DC=com` | unrouted |
| 空 base | `''` + `DC=corp,DC=com` | `...,DC=corp,DC=com` | 命中非空 base |
| 停用配置 | 仅 disabled 的 `DC=corp,DC=com` | `...,DC=corp,DC=com` | unrouted |
| 软删配置 | 仅 `deleted_at` 非空的 enabled 配置 | `...,DC=corp,DC=com` | unrouted |
| 等长歧义 | 两条归一后相同的 enabled base | `...,DC=corp,DC=com` | Go error / SQL unrouted |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 时间戳仲裁（`updated_at` vs `ad_synced_at`） | 队列信号守卫（本 phase） | v1.21 三份审计后 | GORM 自动刷 `updated_at` 使仲裁恒失效，已判死刑 |
| 物化视图对账 | 普通 SQL VIEW 实时计算 | v1.21 方案期 | 规避 MV 刷新僵尸与 AutoMigrate 0A000 |
| 全量 Outbox / FieldRule 引擎 | outbox-lite + 单点信号 | Phase 57-58 | 过度设计被否 |

**Deprecated/outdated:**

- `user_ou_service.go` 等旧 OU 直推路径：Phase 62 清理，本 phase 不碰。
- `FindOUDNByDeptID`（无 config 维度）：仍保留，但本 phase 的对账视图**必须**按 `(dept_id, ad_config_id)` 对齐，参照 `FindOUDNByDeptIDForConfig`。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | 生产库存在 `menu_name='AD域控配置'` 且 `menu_type='C'` 的菜单行（migration_163 依赖它挂按钮权限） | Pitfall 8 / Runtime State | 若不存在，按钮权限无父可挂，需改为挂在 `AD域管理` 下或由管理员手建 |
| A2 | `sys_dept_ou_mapping.dept_id` 在生产 PG 为 `uuid`，`sys_user.dept_id` 为 `varchar(64)`（依模型 tag 由 GORM 建表） | Pitfall 1 | 若类型不同，CAST 仍安全；但若 `sys_dept_ou_mapping.ad_config_id` 与 `sys_ad_config.id` 类型不一致，需要额外 CAST |
| A3 | sqlite 支持 `CAST(x AS TEXT)`、`SUBSTR`、`LENGTH`、`||`、`NOT <int>`（仓库已验证 `SUBSTR`/`LIKE`/`||`/`WITH`；`CAST` 为标准 SQL） | Pattern 2 / Pitfall 1-2 | 若 sqlite `CAST` 行为差异，需用直接等值（sqlite 全 TEXT）并保留 PG 分支 CAST |
| A4 | 对账视图的数据量（用户/映射/配置）在实时计算下可接受 | Architecture | 若数据量大，查询变慢；可加 `sys_user(dept_id)` 部分索引或后续改为快照 |
| A5 | 部署形态可能需要重新编译 Go 二进制以嵌入前端 dist | Runtime State | 若部署是独立静态站点，则只需发布 dist；plan 的部署任务需按实际形态写 |
| A6 | `sys_ad_config.sync_enabled` 在 PG 为 `boolean`、sqlite 为 `INTEGER` | Pitfall 2 | 若 PG 实际是 varchar/int，`NOT` 仍可用（int），但 varchar 不行；需核对生产 DDL |

**如果该表为空**：不适用——上表 6 条均需在 plan 或 UAT 中显式确认，其中 A1/A5 影响功能可用性。

## Open Questions

1. **(RESOLVED) 权限点如何落地（最高优先级）**
   - Resolution: 采纳 Recommendation —— migration_218 新增 `menu_type='F'` 按钮权限行（不可导航、不产生路由），只授予已关联『AD域控配置』父菜单的角色。落点 59-03 task 3。
   - What we know: 后端与前端权限都从 `sys_menu.perms` 读取；没有菜单行 = 无法授予、前端 Tab 恒 403（即使后端超管短路）。
   - What's unclear: D-17 的「不新增菜单 seed」是否允许新增一条 `menu_type='F'` 按钮权限行（不可导航、不产生路由）。
   - Recommendation: 允许并新增；把它理解为「不新增菜单/路由」，与 migration_163 的账号池按钮权限同构。若用户坚持字面禁止，则部署清单必须包含「管理员在菜单管理中手建按钮权限并授权」。

2. **(RESOLVED) D-14 路径字面值与真实挂载冲突**
   - Resolution: 采纳 Recommendation —— 最终路径 `/api/v1/ad-domain/reconciliation/mismatch`，不加 `/system/ad-config` 别名。落点 59-03 task 2 / 59-04 objective。
   - What we know: 路由组是 `/ad-domain`，前端 baseURL 为 `/api/v1`。
   - What's unclear: 是否要在 router 额外挂一个 `/system/ad-config` 别名。
   - Recommendation: 不加别名，统一 `/api/v1/ad-domain/reconciliation/mismatch`；别名只会制造双契约。

3. **(RESOLVED) `config_unrouted` 对歧义配置的判定**
   - Resolution: 采纳 Recommendation —— 等长并列算 unrouted（SQL 用 `best_cnt.cnt = 1` 复刻 Go），一致性测试锁定。落点 59-02 task 2 / 59-03 task 1。
   - What we know: Go `MatchADConfigByDN` 在等长并列时返回 error；D-10 只写「无法归属」。
   - What's unclear: 等长并列（同域重复配置）是否算 unrouted。
   - Recommendation: 算（SQL 用 `best_cnt.cnt = 1` 复刻 Go），并在 `mismatch_detail` 保持通用文案；一致性测试锁定。

4. **(RESOLVED) 「部门无映射」的用户是否应可观测**
   - Resolution: 采纳 Recommendation —— 本 phase 不扩口径；记入 59-02 objective 的 Known Gap 与部署/UAT 说明，不实现第五类。
   - What we know: D-10 四类口径不包含「无映射」；Phase 58 的 `unrecoverable:no-ou-mapping` 队列行也不在本视图。该缺口有两种形态：① 部门在任何配置下都没有映射；② 部门在「该用户 OU 所路由到的配置」下没有映射（跨配置映射缺口）。
   - What's unclear: 这是否是有意的可观测性缺口。
   - Recommendation: 本 phase 不扩口径（锁定决策），在 plan 的「Known Gaps」记录；若用户要求，再新增第五类 `mapping_missing`（按 `(dept_id, routed_config_id)` 判空）。

5. **(RESOLVED) `dead` 行是否纳入守卫/对账**
   - Resolution: 采纳 Recommendation —— 不扩展（locked decisions D-02/D-03 优先）；59-01 的守卫命中集合锁定 pending/failed + update_user。
   - What we know: D-02/D-03 锁定 pending/failed；58-07 建议守卫单独读 dead。
   - What's unclear: 是否在本 phase 扩展。
   - Recommendation: 不扩展（locked decisions 优先）；作为后续 phase 候选。

6. **(RESOLVED) `dept_deleted_at` 的 API 类型**
   - Resolution: 采纳 Recommendation —— 视图 `CAST(... AS TEXT)`，DTO 用 `*string`。落点 59-02 task 2 / 59-03 task 1。
   - What we know: sqlite 视图列无强类型，扫到 `*time.Time` 可能在驱动层失败；PG 返回 `time.Time`。
   - What's unclear: 前端是否需要真实时间格式。
   - Recommendation: 视图输出 `CAST(... AS TEXT)`，DTO 用 `*string`；前端只做非空判断 + 展示。若需要统一格式，在 service 层解析为 `time.Time` 后格式化。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | 后端构建/测试 | ✓ | 1.24.5 | — |
| Node.js | 前端构建/测试 | ✓ | 24.19.0 | — |
| npm | 前端依赖/脚本 | ✓ | 11.17.0 | — |
| PostgreSQL | 生产视图 DDL / 真实迁移 | 未探测（`pg_isready` 不在 PATH） | — | 必须在目标环境执行一次启动迁移验证 |
| SQLite（glebarez） | 单测 | ✓（项目依赖） | 既有版本 | — |
| gsd-tools | 规划/状态命令 | ✓（graphify 未启用） | — | graphify disabled，无影响 |

**Missing dependencies with no fallback:** PostgreSQL 连通性未在本机验证；视图的 PG 方言分支只能在目标环境或 CI 的 PG 实例上做一次真实启动验证（sqlite 测试无法覆盖 uuid/varchar UNION 与 boolean 差异）。

**Missing dependencies with fallback:** 无。

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` + `github.com/stretchr/testify`；sqlite in-memory（`glebarez/sqlite`）；前端 vitest 4（jsdom） |
| Config file | `ruoyi-react-frontend/vitest.config.ts`；Go 无集中配置 |
| Quick run command | `go test ./internal/services/system/ ./internal/services/addomain/ ./internal/core/db/migrations/ ./internal/api/v1/system/ -run 'TestPhase59_' -count=1` |
| Full suite command | `go test ./...` + `cd ruoyi-react-frontend && npm run test -- --run` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GUARD-01 | pending `update_user` 行 → 本地 dept 不被 AD 值覆盖 | unit | `go test ./internal/services/system/ -run TestPhase59_SyncADUser_ActiveQueueSkipsDept -count=1` | ❌ Wave 0 |
| GUARD-01 | failed 行同样拦截 | unit | 同上 `..._FailedQueueAlsoGuards` | ❌ Wave 0 |
| GUARD-01 | 无队列行 → dept 更新为 AD 解析值（不误伤） | unit | `..._NoQueueUpdatesDept` | ❌ Wave 0 |
| GUARD-01 | `pushed` / `dead` / `disable` 行不拦截（谓词字面） | unit | `..._TerminalRowsDoNotGuard` | ❌ Wave 0 |
| GUARD-01 | 队列查询失败（DROP 表）→ 跳过 + WARN | unit | `..._QueueQueryErrorSkipsDeptAndWarns` | ❌ Wave 0 |
| GUARD-01 | WARN 含 userID/username/localDept/adDept/queueID/actionType/attempts | unit（`SetTestBuffer`） | `..._WarnFields` | ❌ Wave 0 |
| GUARD-02 | migration 217 在 sqlite 建视图、可重复执行 | migration | `go test ./internal/core/db/migrations/ -run TestPhase59_Migration217 -count=1` | ❌ Wave 0 |
| GUARD-02 | 四类口径各返回预期行 | unit（sqlite 视图查询） | `go test ./internal/services/addomain/ -run TestPhase59_View_ -count=1` | ❌ Wave 0 |
| GUARD-02 | Go `MatchADConfigByDN` 与 SQL 口径一致（含父子域/逗号边界/歧义） | unit | `..._Consistency` | ❌ Wave 0 |
| GUARD-02 | 分页/筛选（mismatchType/deptId/keyword） | unit | `go test ./internal/services/addomain/ -run TestPhase59_List -count=1` | ❌ Wave 0 |
| GUARD-02 | handler 绑定错误 → 400；默认分页 | unit（httptest） | `go test ./internal/api/v1/system/ -run TestPhase59_ReconHandler -count=1` | ❌ Wave 0 |
| GUARD-02 | 登录鉴权 + 权限点 | integration / UAT | 无 token → 401；无权限 → 403；有权限 → 200（目标环境） | ❌ manual |
| GUARD-02 | 前端 API 路径与默认分页 | unit（vitest） | `npx vitest run src/lib/adDomainApi.test.ts` | ✅ 扩展 |
| GUARD-02 | Tab 无权限 → 403 且不发请求；有权限 → 表格/空态/错误态 | component | `npx vitest run src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** 上述 quick run command + `go build ./...`
- **Per wave merge:** `go test ./...` + `npm run type-check` + `npx vitest run`（至少 ad-domain 相关文件）
- **Phase gate:** 全量 `go test ./...` 绿；前端 `npm run test -- --run` 绿；目标环境启动一次并确认 `v_dept_ou_mismatch` 可查询

### Wave 0 Gaps

- [ ] `internal/services/system/user_sync_service_phase59_test.go` — 守卫行为 + WARN 字段（需要本地 `CREATE TABLE sys_dept_ou_mapping` fixture；`setupUserServiceTestDB` 已含 `sys_user`/`sys_dept`/`sys_ad_config`/`ad_push_queue`）
- [ ] `internal/core/db/migrations/migration_217_dept_ou_mismatch_view_test.go` — sqlite 建视图 + 幂等 + 四类口径
- [ ] `internal/services/addomain/dept_ou_mismatch_service_test.go` — 视图查询/分页/筛选 + Go/SQL 一致性
- [ ] `internal/api/v1/system/ad_reconciliation_handler_test.go` — 绑定错误/默认分页
- [ ] `ruoyi-react-frontend/src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx` — 权限门/空态/错误态
- [ ] 无框架安装缺口（Go/sqlite/vitest 均已就位）

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no（继承） | `adDomain` 组已挂 `JWTAuth` |
| V3 Session Management | no（继承） | JWT access/refresh 既有机制 |
| V4 Access Control | yes | `middleware.RequirePermissions([]string{"ops:ad:config:reconciliation:list"}, core)`；前端 `permissions.includes(...)` 只用于 UI 渲染，不能替代后端 |
| V5 Input Validation | yes | GORM 绑定参数；`mismatchType` 白名单；`keyword` 的 LIKE 元字符转义；`pageSize` 钳制 |
| V6 Cryptography | no | 本 phase 无加密逻辑 |

### Known Threat Patterns for Go + GORM + SQL View

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| SQL injection via `keyword` / `deptId` | Tampering | GORM `Where(..., ?)` 绑定参数；禁止字符串拼接用户输入；`mismatchType` 用四值白名单 |
| Broken access control（只靠前端隐藏 Tab） | Elevation of Privilege | 后端 `RequirePermissions` 独立鉴权（D-15）；无权限返回 403 |
| Information disclosure（DN / 用户名） | Information Disclosure | 端点要求登录 + 权限；前端只在有权限时发请求；不在公共页面暴露 |
| DoS via unbounded view scan | Denial of Service | `query.NormalizePagination` 钳制（默认 10，上限 200）；前端页大小 10/20/50/100；可选 `sys_user(dept_id)` 索引 |
| 权限缓存 30s 陈旧窗口 | Elevation of Privilege（低） | 既有 `mwPermCacheTTL = 30 * time.Second`，本 phase 不引入新风险；授权变更后最长 30s 生效 |
| 视图权限（DB 层） | Information Disclosure | 视图由应用 DB 账号创建；不额外授权给其他 DB 角色 |

## Sources

### Primary (HIGH confidence)
- `internal/models/ad_push_queue.go:17,25,53` — `ADPushEntityUser` / `ADPushActionUpdateUser` / `ADPushQueueActivePredicate`（逐字读取）
- `internal/services/system/user_sync_service.go:388-448,785-800` — `SyncADUser` 与插入点、`resolveDeptFromOU`
- `internal/services/system/user_sync_service.go:240-259` — `updateExistingUser`（确认不写 dept）
- `internal/services/addomain/dn_util.go:36-56` — `NormalizeDN` 逐字语义
- `internal/services/addomain/ad_config_resolver.go:39-81` — `MatchADConfigByDN` 逐字算法
- `internal/services/addomain/ad_push_queue_drain_service.go:196-206,480-500` — 配置加载与归一化比较先例
- `internal/api/router.go:519-524` — `/ad-domain` 挂载与中间件
- `internal/api/v1/system/ad_domain_router.go:26-78` — 配置组与账号池权限组模式
- `internal/api/v1/system/ad_login_history_handler.go:13-48` — 只读 handler 先例
- `internal/services/addomain/login_history_query.go:14-78` — 只读分页 service 先例
- `internal/services/system/menu_service.go:509-560` — 权限来自 `sys_menu.perms`
- `pkg/middleware/permission.go:145-151,277-303` — 权限 SQL 与「任一权限」语义
- `pkg/middleware/oper_log.go:66-96` — 只读 handler 不设 title 即不记录
- `internal/core/db/migrations/migration_215_create_ad_push_queue.go:30-99` — 队列表 DDL 与三条索引
- `internal/core/db/migrations/migration_175_reconciliation_physical_link.go:34-190` — 双方言视图先例（PG）
- `internal/core/db/sqlite_reconciliation_views.go:1-45,133-230` — sqlite 视图先例与方言翻译规则
- `internal/core/db/database.go:439-490,760-840,940-1000,1250-1345` — AutoMigrate 顺序、DROP 前置、三条注册路径
- `internal/models/ad_domain.go:16-45` — `ADConfigStatusEnabled = 0`、`BaseDN`、`status`
- `internal/models/dept_ou_mapping.go:12-41` — 映射字段与唯一键（无 `DeletedAt`）
- `internal/models/user.go:20,34-35` — `DeptID` / `AdDn` / `AdOuDn`
- `internal/models/base.go:10-19` — `BaseModel`（`DeletedAt`）
- `pkg/constants/pagination.go:8-20` — `DefaultPageSize=10`、`MaxPageSize=200`、`MaxListPageSize=100`
- `ruoyi-react-frontend/src/lib/api.ts:55` — `baseURL = /api/v1`
- `ruoyi-react-frontend/src/pages/ad-domain/configs/index.tsx` — Tab 嵌入点
- `.planning/phases/59-ad-login-guard-reconciliation/59-CONTEXT.md` / `59-UI-SPEC.md` — 锁定决策与 UI 契约

### Secondary (MEDIUM confidence)
- `.planning/phases/58-compare-and-move/58-07-SUMMARY.md:239-258` — dead 行对守卫的输入建议（与 D-02/D-03 冲突，见 Open Questions）
- `.planning/phases/57-ad-outbox-lite/57-RESEARCH.md` / `57-VERIFICATION.md` — 入队前置条件与 email/phone 覆盖行为
- `.planning/REQUIREMENTS.md` / `.planning/ROADMAP.md` — GUARD-01/02 原文与 SC

### Tertiary (LOW confidence)
- 外部 SQL 方言文档（sqlite.org / postgresql.org）在本会话无法访问（网络被拦截）；方言结论主要来自仓库内既有实现与训练知识，未逐条对照官方文档。凡依赖方言的结论已在 Assumptions Log 标注 A3/A6。

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — 无新依赖；全部版本与设施在仓库内逐一核实
- Architecture: HIGH — 守卫插入点、路由挂载、权限来源、迁移注册路径均已逐文件读取
- SQL view / dialect: MEDIUM — sqlite 先例充分，但 PG `uuid`/`varchar` UNION 与 `CAST AS TEXT` 未在本机 PG 实例上实测
- Permission seed: MEDIUM — 权限必须落在 `sys_menu` 已由代码证实；D-17 的「不新增菜单 seed」如何解释需用户确认
- Pitfalls: HIGH — 多数有仓库内历史事故注释与既有测试先例

**Research date:** 2026-10-19
**Valid until:** 30 天（代码结构稳定；若 Phase 58 之后又改动 `user_sync_service.go` 或路由挂载，需重新核对行号）
