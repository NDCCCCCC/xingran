# Phase 60: 软删恢复安全 - Research

**Researched:** 2026-10-20
**Domain:** Go 1.24 / GORM / PostgreSQL + SQLite / Gin / AD LDAP 用户同步
**Confidence:** HIGH（仓库代码事实）/ MEDIUM（Context7 外部文档与框架语义）

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### 校验判定（SAFE-01 核心）

- **D-01: 校验仅在存量 `ad_dn` 非空时生效；空 DN 放行恢复。** 软删行 `ad_dn` 为空/NULL = 从未绑定 AD 的 local 用户，不存在「陈旧 DN 指向他人」的接管形态；这类用户恢复行为与今天完全一致（`applyRestoreSoftDeleted` 强制转 AD）。
  — **Reversibility:** reversible — 收紧只需在校验入口加一个空值分支。

- **D-02: DN 一致性判定用 `NormalizeDN` 归一化比较**（`internal/services/addomain/dn_util.go`，单一真实源），与 Phase 58 D-02 / Phase 59 D-10 既定口径一致；不做裸字符串相等。

- **D-03: 拒绝时行保持软删状态、绝不清空存量 `ad_dn`。** 这是**关键安全不变量**：D-01 规定空 DN 放行，若拒绝路径顺手清了存量 DN，则该用户二次登录即变「空 DN → 放行」绕过校验——整道闸门失效。存量 DN 是审计证据与后续人工处置的依据。

#### 登录路径呈现

- **D-04: 新增专用 `SyncErrorReason` 原因码**（命名如 `restore_refused` / `soft_delete_mismatch`，planner 定）+ `auth.go syncErrorReasonMessage()` 专用文案（如「该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员」）。`ad_authenticator.go:132` 既有「同步失败不崩认证、按原因如实反馈」管道自然承载。用户可见文案**不含 DN 内部细节**（防信息泄露），完整细节进审计 WARN。
  — 价值：运维看登录报错即可区分「安全拦截」与「AD 基础设施故障」（admin_bind/user_search 等），不必翻日志。

#### 审计通道

- **D-05: 纯 applogger 结构化 WARN**，字段含 username / userID / 存量 ad_dn / AD 当前 DN / 路径（login|batch）。与 Phase 59 GUARD-01 同风格；service 层自包含，不引入 service→handler 上报通道、不进 sys_oper_log、不建新表。两路径共用同一个日志产出点。

#### 批量路径记账

- **D-06: 被拒项计入 `result.Skipped` + `Errors` 明细**（error 文本标明「AD 身份不一致拒绝恢复」）。`BatchSyncResult.Skipped` 字段已存在但从未被递增过（`user_sync_service.go:696` 只打印）——本 phase 首次赋予真实语义。拒绝在校验点**短路跳过，不抛 error**：抛 error 会触发 `runInBatchedTx` 整批降级逐个重试，把策略拒绝伪装成 DB 故障。

#### 共用逻辑形态（SAFE-02）

- **D-07: 校验做成两路径共用的单一入口**（分类命中 catSoftDeleted 之后、恢复写库之前调用），登录与批量路径**不得各自实现**比对逻辑。具体挂接点（`restoreSoftDeletedUser` 入口 vs `applyRestoreSoftDeleted` 入口 vs 批量 plan 阶段）归 planner，但两路径必须收敛到同一个函数。
  — **Reversibility:** costly — SAFE-02 的本体就是「行为一致」；一旦两路径各自实现，漂移即失守，后补统一需同时改两处调用点并重写全部行为用例。

### Claude's Discretion

- 校验函数的签名/命名/文件位置（`user_sync_service.go` 包内私有）
- 批量路径校验时机：阶段 2 plan 时短路（推荐——跳过无谓的密码哈希）vs 阶段 5c 写前
- 拒绝的内部信号形态（sentinel error / 分类标记 / 返回布尔）
- 专用原因码具体字符串值与中文文案措辞
- WARN 日志具体格式（GUARD-01 :435-439 的字段风格可参照）
- 测试基建复用（`user_sync_classify_test.go` / `user_sync_service_test.go` / `user_sync_service_phase59_test.go` 的 sqlite fixture）
- **预计零迁移**：本 phase 无 schema 变更、无新菜单/权限点；若研究发现确需迁移，编号从 migration_219 起（215=队列表、216=sys_config 旋钮、217/218=Phase 59 视图与权限载体，均已占用）

### Deferred Ideas (OUT OF SCOPE)

- **拒绝后的管理员处置端点**（查看被拒明细 / 手动恢复 / 清理软删行）— 新能力，独立 phase；当前运维处置靠审计 WARN + DBA 手工
- **`catLocalSameName → bindLocalUserToAD` 绑定路径的同类身份校验** — ROADMAP SC 未要求，风险形态不同（活跃 local 用户主动绑定）
- **对账视图增加「软删恢复被拒」条目** — `v_dept_ou_mismatch` 是 dept↔OU 专属口径，扩展属 GUARD-02 后续 phase
- **auth.go AD 登录不查 user.Status** — 57 Deferred 遗留独立 bug，与 SAFE 无关
- **HYGIENE-03 死代码删除**（`user_ou_service.go` 整文件 / `GetUserDeptByADOU` / `BatchMoveUsersToNewOU`）— Phase 62 范围；本 phase **不得动**，它们是 F-22 对照语义的最后载体

### Reviewed Todos (not folded)

- `operlog-exclude-paths.md` — 「operlog.exclude_paths 配置驱动白名单」：匹配度 0.4，关键词误匹配（notes/phase），与软删恢复无关
- `v1.17-reconciliation-decisions.md` — 「v1.17 资产对账决策追踪」：匹配度 0.2，资产对账领域，无关
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SAFE-01 | `catSoftDeleted` 恢复前校验 AD 当前 DN 与存量 `ad_dn` 一致，不一致拒绝恢复 + 安全审计日志 | 软删行 `AdDn` 已由 `indexExistingUsers` 预查询携带（零额外查询）；`NormalizeDN` 可直接复用；登录路径在 `restoreSoftDeletedUser` 入口短路、批量路径在阶段 2 过滤，均可做到拒绝前不写库；`applogger.Warnf` + `SetTestBuffer` 可断言审计字段 |
| SAFE-02 | 登录路径与 batch-sync 共用分类逻辑与校验，行为一致 | `classifyUser` + `indexExistingUsers` 已两路径共用（现状基础）；新增一个包内私有校验入口，由 `restoreSoftDeletedUser` 与 `BatchSyncADUsers` 各调用一次，比对与日志集中在同一函数；一致性用例可对同一 fixture 跑两条入口断言相同结果 |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

| 约束 | 对本 phase 的含义 |
|------|------------------|
| Handler-Service 分层强制 | 校验、拒绝、审计日志全部落在 `internal/services/system`；handler 不新增逻辑 |
| 操作日志约定（`operlog.Record`） | 本 phase 无新增写端点；D-05 明确拒绝事件**不进** `sys_oper_log`，走纯 applogger WARN，属于对该约定的显式范围豁免 |
| `response.Success` / `response.Error` 包装 | 登录失败沿用 `auth.go` 既有 `response.Error(...)`；批量端点沿用既有 handler 返回结构，不改 |
| 修改 Go 文件后必须 `go build ./...` | 每个 plan 收尾执行；本 phase 预计改 3~4 个生产文件 |
| 范围约束（Scope Constrainment） | 只改软删恢复路径；不碰 `catActiveAD` 更新、不碰 `bindLocalUserToAD`、不碰 Phase 62 死代码 |
| 代码层修复、禁 DB TRIGGER | 校验在 Go service 层完成，无 schema/trigger 变更 |
| 提交前询问用户 | 研究阶段不提交；执行阶段遵循既有 GSD 流程 |
| 统计/分页约定 | 本 phase 无新增分页端点 |

## Summary

Phase 60 是**单点安全闸门 + 双路径收敛**，不需要新表、不需要迁移、不需要前端改动。现状已把登录路径（`SyncADUser → SyncUserFromAD → restoreSoftDeletedUser`）与批量路径（`BatchSyncADUsers`）收敛到同一套 `classifyUser` / `indexExistingUsers` / `applyRestoreSoftDeleted` 上，因此 SAFE-02 的改造面很小：新增一个包内私有的「软删恢复身份校验」函数，两条路径各调用一次即可。

关键代码事实（均已读源确认）：

- 软删行的 `ad_dn` 已经在预查询结果里。`indexExistingUsers` 用 `db.Unscoped().Where("username IN ? AND deleted_at IS NOT NULL").Find(&deleted)` 加载软删行，`models.User.AdDn` 是普通字段，取用零成本。
- `NormalizeDN` 是纯字符串函数，永不失败，可直接用于比较；空输入返回空串，因此 D-01 的「空 DN 放行」可以统一写成 `NormalizeDN(stored) == ""`，同时覆盖 nil、空串与纯空白。
- 登录路径的错误会穿过 `SyncUserFromAD` 的 `fmt.Errorf("...: %w", err)` 与 `SyncADUser`，最终到达 `ad_authenticator.go` 的错误分支；该分支目前把所有错误一律映射为 `SyncErrorReason: "user_sync"`。要用专用原因码，必须让错误携带可识别信号（推荐在 `security` 包声明 sentinel，`system` 用 `%w` 包装，`ad_authenticator` 用 `errors.Is` 判定）。
- 批量路径**不能**把拒绝做成 `runInBatchedTx` 里的 error：该函数在任何 error 时会把整批降级为逐个独立事务，并把该项计入 `Failed`。D-06 要求策略拒绝计入 `Skipped`，所以在阶段 2 分类后、阶段 3 哈希前就把被拒项从 `plansByCat[catSoftDeleted]` 里过滤掉，是最贴合 D-06 且最省算力的位置（被拒用户不再做 60 万次 PBKDF2）。
- 拒绝路径必须**零写入**。登录路径在 `restoreSoftDeletedUser` 开头短路并返回 error，事务回滚时本来也没有写操作；批量路径在进入任何写事务之前过滤。两条路径都不会触碰 `ad_dn`，D-03 的安全不变量可用「拒绝后重新读库断言 `deleted_at` 仍非空且 `ad_dn` 逐字不变」锁定。

**Primary recommendation:**

1. 在 `internal/core/security/authenticator.go` 新增 `ErrSoftDeleteIdentityMismatch` sentinel 与 `SyncReasonSoftDeleteMismatch = "soft_delete_mismatch"` 常量。
2. 在 `UserSyncService` 新增包内私有 `checkSoftDeletedRestore(user *models.User, currentUserDN, path string) bool`：空存量 DN 放行；否则 `addomain.NormalizeDN` 比较；不一致时写一条 `[SECURITY]` WARN（含 username / userID / 存量 DN / 当前 DN / path）并返回 false。**日志只在这里产出**。
3. 登录路径：`restoreSoftDeletedUser` 最前面调用该函数；false 时返回 `fmt.Errorf("%w: username=%s", security.ErrSoftDeleteIdentityMismatch, ...)`，**在密码哈希与任何写库之前**短路。
4. 批量路径：阶段 2 分类完成后立即遍历 `plansByCat[catSoftDeleted]`，被拒项 `result.Skipped++` 并 append 一条 DN-free 的 `SyncError`，只把允许项放回切片；随后阶段 3 哈希、阶段 4 部门解析、阶段 5c 恢复都自然只处理允许项。
5. `ad_authenticator.go` 把 error→reason 映射抽成纯函数（或内联加 `errors.Is` 分支）：sentinel → `"soft_delete_mismatch"`，其余 → `"user_sync"`；`auth.go syncErrorReasonMessage()` 新增该原因码 → 「该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员」。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 软删恢复身份校验（SAFE-01） | API / Backend（`system.UserSyncService`） | Database（`sys_user` 预查询快照） | 分类与恢复写库本来就在该 service；校验必须贴着 `catSoftDeleted` 分支，不能上移 handler |
| DN 归一化比较 | Backend / Service（`addomain.NormalizeDN`） | — | Phase 58 已确立的单一真实源，禁止在调用点手写比较 |
| 安全审计日志 | Backend / Logging（`applogger.Warnf`） | — | D-05 锁定纯 applogger，不进 `sys_oper_log`、不建表 |
| 登录失败原因呈现 | Backend / API（`security.ADAuthenticator` + `api/v1/auth.go`） | — | sentinel → `SyncErrorReason` → 中文文案的既有管道 |
| 批量拒绝记账 | Backend / API（`BatchSyncResult`） | Browser（结果面板展示） | `Skipped` / `Errors` 已随响应返回，前端类型已存在，无需改动 |
| 被拒用户行状态保持 | Database / Storage | — | 不写库即保持软删；`ad_dn` 原值即审计证据 |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.24.5（`go.mod` 声明 1.24.0） | 语言运行时 | 项目既有 toolchain，`go version` 实测 1.24.5 `[VERIFIED: go version]` |
| GORM | v1.30.5 | 事务、软删、`Unscoped` 恢复 | 项目唯一 ORM；`gorm.io/gorm v1.30.5` `[VERIFIED: go.mod:34]` |
| `github.com/glebarez/sqlite` | v1.11.0 | 行为测试内存库 | 既有 fixture 使用；`[VERIFIED: go.mod:12]` |
| `github.com/stretchr/testify` | v1.11.1 | 断言 | 既有测试统一使用；`[VERIFIED: go.mod:27]` |
| `github.com/sirupsen/logrus`（经 `pkg/logger`） | v1.9.3 | 结构化 WARN 审计 | D-05 指定；`[VERIFIED: go.mod:25]` |
| `addomain.NormalizeDN` | 仓库内 | DN 归一化比较 | D-02 单一真实源；`[VERIFIED: internal/services/addomain/dn_util.go:36-56]` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `errors`（stdlib） | Go 1.24 | sentinel 包装与 `errors.Is` | 登录路径原因码识别 |
| `strings`（stdlib） | Go 1.24 | 空白 DN 判定（可被 `NormalizeDN` 覆盖） | 仅在需要显式判断时使用 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `NormalizeDN` 字符串归一化 | `go-ldap` 的 `ParseDN(...).EqualFold(...)` | `ParseDN` 遇未转义/非法 DN 会返回错误，而仓库存量 DN 可能来自历史未转义写入；`NormalizeDN` 永不失败且已是 Phase 58/59 契约。D-02 已锁定，**不采用** `[CITED: context7 /go-ldap/ldap]` |
| `security` 包 sentinel + `errors.Is` | `security` 包定义 `SyncReasoner` 接口 + `errors.As` | 接口更可扩展，但本 phase 只有一个原因码，sentinel 改动更小、与既有 `ErrUserNotFound` 等风格一致 |
| 批量阶段 2 过滤 | 批量阶段 5c 写前校验 | 阶段 5c 抛 error 会触发 `runInBatchedTx` 降级，违背 D-06；若在闭包内「吞掉」则 `Success` 会被错误递增。阶段 2 过滤同时省掉被拒用户的 PBKDF2 |

**Installation:**

```bash
# 无新增依赖。本 phase 只使用仓库已有包与标准库。
```

**Version verification:** `go.mod` 已确认上述版本；`go version go1.24.5 windows/amd64` 实测通过。本 phase 不安装任何外部包。

## Package Legitimacy Audit

**N/A** — 本 phase 不引入任何新的第三方包，不修改 `go.mod` / `go.sum`。Package Legitimacy Gate 的所有检查项均无对象。

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
AD 用户登录 (bind 成功)
  │
  ▼
ADAuthenticator.Authenticate
  │  searchUserInfo → ADUserInfo{UserDN, ...}
  ▼
UserSyncService.SyncADUser
  │
  ▼
SyncUserFromAD ── tx.Begin
  │
  ├─ indexExistingUsers ──► 活跃行 + 软删行 (含 ad_dn)
  │
  ├─ classifyUser ──► catActiveAD / catLocalSameName / catSoftDeleted / catNew
  │
  └─ catSoftDeleted
        │
        ▼
   checkSoftDeletedRestore(path="login")   ◄── 共享校验入口（唯一日志点）
        │
        ├─ 存量 ad_dn 空 ─────────────────────► 允许 (D-01)
        ├─ NormalizeDN(stored)==NormalizeDN(current) ─► 允许
        └─ 不一致 ──► WARN[SECURITY] + 返回 false
                        │
                        ▼
              return ErrSoftDeleteIdentityMismatch
                        │  (tx rollback，零写入；ad_dn 原值保留)
                        ▼
              SyncUserFromAD %w 包装 → SyncADUser → ADAuthenticator
                        │
                        ▼
              errors.Is → SyncErrorReason="soft_delete_mismatch"
                        │
                        ▼
              auth.go syncErrorReasonMessage → 用户可见中文提示


批量同步入口 (POST /ad-domain/users/batch-sync)
  │
  ▼
BatchSyncADUsers
  │
  ├─ 阶段1 indexExistingUsers（同一个预查询）
  ├─ 阶段2 classifyUser（同一个分类函数）
  │     │
  │     └─ catSoftDeleted 逐项
  │           │
  │           ▼
  │      checkSoftDeletedRestore(path="batch")   ◄── 同一共享函数
  │           │
  │           ├─ 允许 → 放回 plansByCat[catSoftDeleted]
  │           └─ 拒绝 → result.Skipped++ / result.Errors += DN-free 明细
  │                     （绝不进入 runInBatchedTx，故不会触发降级）
  │
  ├─ 阶段3 仅对允许项并发哈希
  ├─ 阶段4 部门预解析
  └─ 阶段5c 仅对允许项恢复写库
```

### Recommended Project Structure

```text
internal/
├── core/security/
│   ├── authenticator.go          # 新增 ErrSoftDeleteIdentityMismatch + SyncReasonSoftDeleteMismatch
│   └── ad_authenticator.go       # error → SyncErrorReason 映射（新增 errors.Is 分支/纯函数）
├── api/v1/
│   └── auth.go                   # syncErrorReasonMessage 新增中文文案
└── services/system/
    ├── user_sync_service.go      # 共享校验函数 + 两处调用点
    ├── user_sync_service_phase60_test.go   # 新增：SAFE-01/SAFE-02 行为用例
    └── ...（不改 user_ou_service.go，Phase 62 范围）

ruoyi-react-frontend/             # 零改动：BatchSyncResult.skipped 与登录响应管道已存在
```

### Pattern 1: 两路径共用的校验入口（SAFE-02）

**What:** 一个包内私有函数同时完成「空 DN 放行判定 / NormalizeDN 比较 / 拒绝审计 WARN」，返回布尔。两条路径只在「拒绝后的动作」上不同（登录返回 sentinel error，批量计入 Skipped）。

**When to use:** 任何需要在登录与批量两条入口上保持同一安全策略的场景。

**Example:**

```go
// internal/services/system/user_sync_service.go

// checkSoftDeletedRestore 软删用户恢复身份校验（SAFE-01 / SAFE-02 共用入口）。
// 返回 true 表示允许恢复；false 表示拒绝（本函数已写审计 WARN）。
//
// 调用约定：
//   - 登录路径：restoreSoftDeletedUser 入口，拒绝时返回 security.ErrSoftDeleteIdentityMismatch；
//   - 批量路径：BatchSyncADUsers 阶段 2 分类后，拒绝时计入 result.Skipped，
//     不得抛 error（否则 runInBatchedTx 会把策略拒绝降级成 DB 故障）。
//
// 安全不变量（D-03）：本函数只读不写。拒绝时绝不清空 user.AdDn。
func (s *UserSyncService) checkSoftDeletedRestore(user *models.User, currentUserDN, path string) bool {
	storedDN := ""
	if user.AdDn != nil {
		storedDN = *user.AdDn
	}
	normalizedStored := addomain.NormalizeDN(storedDN)

	// D-01: 存量 ad_dn 为空（NULL / 空串 / 纯空白）→ 无身份可比对，放行。
	if normalizedStored == "" {
		return true
	}

	// D-02: 归一化比较（属性名与属性值都大小写不敏感）。
	normalizedCurrent := addomain.NormalizeDN(currentUserDN)
	if normalizedStored == normalizedCurrent {
		return true
	}

	// D-05: 两路径共用的唯一审计日志产出点。用户可见文案不含 DN；
	// 完整 DN 细节只进 WARN。
	applogger.Warnf(
		"[SECURITY] 软删用户恢复被拒绝: AD 身份不一致, path=%s, userID=%s, username=%s, storedAdDN=%q, currentAdDN=%q, normalizedStored=%q, normalizedCurrent=%q",
		path, user.ID, user.Username, storedDN, currentUserDN,
		normalizedStored, normalizedCurrent,
	)
	return false
}
```

### Pattern 2: sentinel error → 专用 SyncErrorReason（D-04）

**What:** `system` 包用 `%w` 包装 `security` 包声明的 sentinel；`security` 包在错误分支用 `errors.Is` 识别并写入专用原因码。

**Why:** `security` 不能反向 import `system`（`system` 已 import `security`，会形成循环）；sentinel 放在 `security` 是唯一无环方向。`SyncUserFromAD` 的 `fmt.Errorf("同步AD用户失败: %w", err)` 不会破坏 `errors.Is`。

**覆盖范围：** hybrid 认证模式同样承载该原因码——`HybridAuthenticator` 在 AD 分支成功后原样返回 `adResult`（仅强制 `NeedsSync=true`），不会丢弃 `SyncErrorReason` `[VERIFIED: internal/core/security/hybrid_authenticator.go:40-48]`。

**Example:**

```go
// internal/core/security/authenticator.go —— 与既有 ErrUserNotFound 等并列
var (
	// ... 既有错误 ...

	// ErrSoftDeleteIdentityMismatch 软删用户恢复被安全策略拒绝：
	// AD 当前 DN 与 sys_user.ad_dn 存量值不一致（SAFE-01）。
	ErrSoftDeleteIdentityMismatch = errors.New("软删用户恢复被拒绝: AD 身份不一致")
)

// SyncReasonSoftDeleteMismatch 是 AuthResult.SyncErrorReason 的专用原因码，
// 由 api/v1/auth.go syncErrorReasonMessage 映射为对用户的提示。
const SyncReasonSoftDeleteMismatch = "soft_delete_mismatch"
```

```go
// internal/core/security/ad_authenticator.go —— :131-141 错误分支
syncedUser, err := a.userSyncer.SyncADUser(ctx, adUserInfo, defaultRoleID)
if err != nil {
	applogger.Errorf("[ADAuth] 同步AD用户失败: username=%s, error=%v", adUserInfo.Username, err)
	return &AuthResult{
		User:            nil,
		AuthSource:      "ad",
		ADUserInfo:      adUserInfo,
		NeedsSync:       true,
		SyncErrorReason: syncErrorReasonFor(err),
	}, nil
}
```

```go
// internal/core/security/ad_authenticator.go —— 新增纯函数，便于单测
// syncErrorReasonFor 把同步错误映射为 AuthResult.SyncErrorReason。
// 默认 user_sync（既有语义）；安全策略拒绝使用专用原因码，
// 让运维在登录报错上即可区分「安全拦截」与「AD 基础设施故障」。
func syncErrorReasonFor(err error) string {
	if errors.Is(err, ErrSoftDeleteIdentityMismatch) {
		return SyncReasonSoftDeleteMismatch
	}
	return "user_sync"
}
```

```go
// internal/api/v1/auth.go —— syncErrorReasonMessage
case security.SyncReasonSoftDeleteMismatch:
	// DN 细节绝不进用户文案（D-04）。
	return "该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员"
```

### Pattern 3: 批量短路记账，绕开降级管道（D-06）

**What:** 在阶段 2 分类完成后、阶段 3 哈希之前过滤被拒项；被拒项计入 `Skipped` + `Errors`，从 `catSoftDeleted` 工作队列中移除。

**Why:** `runInBatchedTx` 的契约是「某批事务失败则降级为该批逐个独立事务，失败项计入 `result.Failed`」。策略拒绝若以 error 形式进入，会被记账为 `Failed` 并触发无意义重试。

**Example:**

```go
// internal/services/system/user_sync_service.go —— 阶段 2 之后、阶段 3 之前

// SAFE-01/SAFE-02: 在哈希与任何写库之前过滤被拒项。
// 拒绝是策略行为不是故障：不进入 runInBatchedTx、不抛 error、计入 Skipped。
if items := plansByCat[catSoftDeleted]; len(items) > 0 {
	allowed := make([]*batchPlanItem, 0, len(items))
	for _, it := range items {
		if !s.checkSoftDeletedRestore(it.user, it.adUser.UserDN, "batch") {
			result.Skipped++
			// 该错误文本会随响应返回前端，因此不含 DN（DN 细节只在 WARN）。
			result.Errors = append(result.Errors, SyncError{
				Username: it.adUser.Username,
				Error:    "AD 身份不一致，拒绝恢复（需管理员核对 ad_dn）",
			})
			continue
		}
		allowed = append(allowed, it)
	}
	plansByCat[catSoftDeleted] = allowed
}
```

### Anti-Patterns to Avoid

- **把校验放进 `applyRestoreSoftDeleted` 并返回 error** — 该函数同时被批量阶段 5c 的 `runInBatchedTx` 闭包调用；error 会触发整批降级，把策略拒绝变成 DB 故障并计入 `Failed`。校验必须前置到阶段 2。
- **拒绝时顺手清空 `ad_dn`** — D-01 空 DN 放行，清空后二次登录即可绕过校验，整道闸门失效（D-03 明确禁止）。
- **用裸字符串比较 DN** — AD 回写的属性名/值大小写常与存量不一致；Phase 58 已因此踩过重复 `ModifyDN` 的坑。必须走 `NormalizeDN`。
- **用 `user.AdDn != nil` 判空** — 纯空白字符串仍是非 nil；统一用 `NormalizeDN(stored) == ""` 判定。
- **登录路径返回普通 `fmt.Errorf`** — `ad_authenticator` 会把它归类为 `user_sync`，安全拦截与基础设施故障不可区分。
- **把 DN 写进用户可见文案或 `SyncError.Error`** — 信息泄露；D-04/D-05 要求 DN 只进 WARN。
- **在批量路径把被拒项留在 `plansByCat` 里** — `runInBatchedTx` 会把它们计入 `Success`，破坏 `Success+Failed+Skipped==Total`。
- **改动 `user_ou_service.go`** — 它是 Phase 62 HYGIENE-03 的删除对象，且是 F-22 语义的对照载体；本 phase 只可参考其 `[SECURITY]` WARN 风格。
- **在 `classifyUser` 内做校验与日志** — 该函数是纯函数（既有 `user_sync_classify_test.go` 无 DB/日志依赖），加入副作用会破坏测试基座与分层；校验放在分类之后。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| DN 归一化比较 | 自写 `strings.EqualFold` / 手写 split+lower | `addomain.NormalizeDN` | Phase 58/59 既定契约；处理 attr 名与值双侧大小写、段内空格；纯函数可单测 |
| 错误原因传播 | 新的 service→handler 上报通道 / 全局 map | `fmt.Errorf("%w")` + `errors.Is` + 既有 `SyncErrorReason` | 既有管道已存在且被 `auth.go` 消费；零架构成本 |
| 软删行读取 | 额外 `Unscoped().First` 查询 | `indexExistingUsers` 预查询结果中的 `user.AdDn` | 已加载全字段（`Find(&deleted)`），零额外查询 |
| 批量结果展示 | 新响应字段 / 前端改动 | 既有 `BatchSyncResult.Skipped` / `Errors` | 字段与前端类型均存在（`adDomainApi.ts:676-686`），只差递增 |
| 审计落库 | `sys_oper_log` / 新安全事件表 | `applogger.Warnf` | D-05 明确锁定；避免跨层上报与建表 |

**Key insight:** 本 phase 的复杂度不在实现，而在**放置位置**：校验必须在「分类之后、任何写库/哈希之前」，且批量路径必须绕开既有的批次降级管道。选错位置会导致安全语义正确但运维语义错误（拒绝被记成 DB 故障）或性能白白损失（被拒用户仍做 60 万次 PBKDF2）。

## Runtime State Inventory

> 本 phase 无 schema/rename/migration，但会改变**既有软删行的运行期行为**，因此按类目核对运行时影响。

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `sys_user` 中 `deleted_at IS NOT NULL` 且 `ad_dn` 非空的历史行，下次 AD 登录/批量同步将首次接受 DN 校验 | 无数据迁移；行为变更（预期）。存量 `ad_dn` 一律保留，不批量清洗 |
| Live service config | 无。校验不含任何 `sys_config` 开关；无外部服务注册 | 无 |
| OS-registered state | 无。不涉及 Task Scheduler / pm2 / systemd | 无 |
| Secrets/env vars | 无新增；不读取任何 env | 无 |
| Build artifacts | 无。仅 Go 源码改动，`go build ./...` 重新生成二进制 | 部署时重新编译 |
| 部署影响 | 存量 soft-deleted 用户的 `ad_dn` 若因 AD 侧改名/移动而陈旧，将被拒绝恢复；这是 SAFE-01 的预期后果 | 运维须知：拒绝明细只见于 WARN 日志；管理员需手动核对/清理（处置端点已 Deferred） |

**Nothing found in category:** Live service config / OS-registered state / Secrets 均为 none（已逐项确认）。

## Common Pitfalls

### Pitfall 1: 在 `runInBatchedTx` 闭包内抛哨兵 error
**What goes wrong:** 批量同步日志出现「批次事务失败，降级逐个处理」，每个被拒用户被重试一次并计入 `Failed`，`Skipped` 始终为 0。
**Why it happens:** `runInBatchedTx` 把任何 error 当作事务失败；driven by `[VERIFIED: internal/services/system/user_sync_service.go:744-768]` 的降级逻辑。
**How to avoid:** 在阶段 2 过滤，拒绝项不进入 `runInBatchedTx`；闭包内只处理允许项。
**Warning signs:** `result.Failed` 增加、`result.Skipped == 0`、日志出现「降级逐个处理」。

### Pitfall 2: 拒绝路径清空 `ad_dn`（二次登录绕过）
**What goes wrong:** 第一次登录被拒后 `ad_dn` 被清空；第二次登录命中 D-01「空 DN 放行」，用户被恢复，闸门失效。
**Why it happens:** 为了「清理陈旧数据」在拒绝分支顺手 `Update("ad_dn", nil)`。
**How to avoid:** 校验函数只读；拒绝分支零写库。测试断言拒绝后 `ad_dn` 逐字不变。
**Warning signs:** 拒绝后 `SELECT ad_dn` 为 NULL；第二次登录同一用户成功。

### Pitfall 3: 用 `user.AdDn != nil` 判断「存量 DN 非空」
**What goes wrong:** 值 `"   "` 或 `""` 的非 nil 指针被当作「有 DN」，进入比较后与当前 DN 不等 → 误拒。
**Why it happens:** 只判指针不判内容。
**How to avoid:** 统一 `addomain.NormalizeDN(stored) == ""` 判空（空串、纯空白都返回 `""`）`[VERIFIED: internal/services/addomain/dn_util.go:36-40]`。
**Warning signs:** 从未绑定 AD 的软删 local 用户被拒。

### Pitfall 4: DN 裸比较导致大小写误拒
**What goes wrong:** AD 当前 DN 为 `CN=Zhang,OU=IT,DC=corp,DC=com`，存量是 `cn=zhang,ou=it,dc=corp,dc=com` → 字符串不等 → 合法用户被拒。
**Why it happens:** 直接 `*user.AdDn == adUser.UserDN`。
**How to avoid:** `NormalizeDN` 双侧归一化后比较（D-02）。
**Warning signs:** 日志中 `storedAdDN` 与 `currentAdDN` 仅大小写不同但被拒。

### Pitfall 5: 批量先哈希后校验
**What goes wrong:** 被拒用户仍执行 60 万次 PBKDF2，批量同步耗时无谓增加。
**Why it happens:** 校验放在阶段 5c。
**How to avoid:** 阶段 2 过滤（本 research 推荐）。允许/拒绝测试也更快：拒绝用例完全不触发哈希。
**Warning signs:** 全为被拒项的批量同步仍耗时数秒。

### Pitfall 6: 用户可见文案泄露 DN
**What goes wrong:** 前端/登录错误页展示完整 DN，暴露内部目录结构。
**Why it happens:** 直接把 `err.Error()` 返回给用户。
**How to avoid:** `auth.go` 只按原因码返回固定中文文案；`SyncError.Error` 同样 DN-free；DN 细节只进 WARN。
**Warning signs:** 响应 message 含 `CN=` / `OU=` / `DC=`。

### Pitfall 7: 原因码字符串两处漂移
**What goes wrong:** `ad_authenticator` 写 `"soft_delete_mismatch"`，`auth.go` 映射另一个拼写 → 用户看到默认文案。
**Why it happens:** 两处各自写字面量。
**How to avoid:** 在 `security` 包定义 `SyncReasonSoftDeleteMismatch` 常量，`ad_authenticator` 与 `auth.go` 都引用常量（`auth.go` 已 import `security`）。
**Warning signs:** 登录提示落到 default 分支「AD 用户信息同步失败…」。

### Pitfall 8: 批量测试 fixture 触发 nil mapper
**What goes wrong:** `BatchSyncADUsers` 阶段 4 对非空 `OuDn` 调用 `resolveDeptFromOU`，若 `ouMapper == nil` 直接 nil 解引用 panic。
**Why it happens:** 现有实现假设生产注入真实 mapper `[VERIFIED: internal/services/system/user_sync_service.go:808-817]`。
**How to avoid:** 聚焦 SAFE 的批量用例把 `ADUserInfoForSync.OuDn` 置空即可跳过部门解析；若要测部门回写，复用 `phase59GuardFixture` 的映射表 + 真实 `DeptOUmapper`。
**Warning signs:** 测试 panic 在 `FindDeptByOUDN`。

### Pitfall 9: 测试辅助函数重名
**What goes wrong:** 同包新增测试文件重复定义 `captureLogs`、`stringPtr`、`strPtrSyncTest` 等 → 编译失败。
**Why it happens:** Phase 59/既有测试已占用这些名字。
**How to avoid:** 直接复用 `captureLogs`（`user_sync_service_phase59_test.go:143-148`）与 `setupTestDBForSync`；新 helper 用 `phase60` 前缀。
**Warning signs:** `redeclared in this block`。

### Pitfall 10: 误改 Phase 62 死代码
**What goes wrong:** 为「统一逻辑」去改 `user_ou_service.go` / `HandleUserLoginAD`，破坏 HYGIENE-03 的 F-22 对照语义。
**Why it happens:** 该文件里也有一段软删拒绝逻辑。
**How to avoid:** 只把它当作 WARN 风格参考；本 phase 不编辑该文件。
**Warning signs:** diff 中出现 `internal/services/addomain/user_ou_service.go`。

## Code Examples

### 登录路径：校验前置，零写入短路

```go
// internal/services/system/user_sync_service.go —— restoreSoftDeletedUser 开头
//
// SAFE-01/SAFE-02: 在密码哈希与任何写库之前做身份校验。
// 拒绝时返回 sentinel，事务内无写入，GORM Transaction 回滚后行保持软删。
func (s *UserSyncService) restoreSoftDeletedUser(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync, defaultRoleID string) error {
	if !s.checkSoftDeletedRestore(user, adUser.UserDN, "login") {
		return fmt.Errorf("%w: username=%s", security.ErrSoftDeleteIdentityMismatch, adUser.Username)
	}

	// ... 既有逻辑不变（HashPassword → applyRestoreSoftDeleted → assignRole）...
}
```

来源：现有 `restoreSoftDeletedUser` 结构 `[VERIFIED: internal/services/system/user_sync_service.go:272-292]`；事务回滚语义 `[CITED: context7 /go-gorm/gorm — transactions]`。

### 纯函数单测骨架（校验判定）

```go
// internal/services/system/user_sync_service_phase60_test.go

func TestPhase60_CheckSoftDeletedRestore_Predicate(t *testing.T) {
	tests := []struct {
		name      string
		storedDN  *string
		currentDN string
		want      bool
	}{
		{"nil 存量 DN 放行", nil, "CN=u,DC=corp,DC=com", true},
		{"空串存量 DN 放行", strPtrSyncTest(""), "CN=u,DC=corp,DC=com", true},
		{"纯空白存量 DN 放行", strPtrSyncTest("   "), "CN=u,DC=corp,DC=com", true},
		{"完全一致允许", strPtrSyncTest("CN=u,DC=corp,DC=com"), "CN=u,DC=corp,DC=com", true},
		{"大小写不同允许", strPtrSyncTest("cn=U,dc=CORP,dc=COM"), "CN=u,DC=corp,DC=com", true},
		{"段内空格允许", strPtrSyncTest("CN=u, DC=corp, DC=com"), "CN=u,DC=corp,DC=com", true},
		{"不同 DN 拒绝", strPtrSyncTest("CN=other,DC=corp,DC=com"), "CN=u,DC=corp,DC=com", false},
		{"存量非空但当前为空拒绝", strPtrSyncTest("CN=u,DC=corp,DC=com"), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &UserSyncService{}
			got := svc.checkSoftDeletedRestore(
				&models.User{Username: "u", AdDn: tt.storedDN}, tt.currentDN, "login")
			assert.Equal(t, tt.want, got)
		})
	}
}
```

说明：`strPtrSyncTest` 是 `user_sync_classify_test.go:13` 已定义的辅助函数，可直接复用；`checkSoftDeletedRestore` 不依赖 `s` 的字段，空 service 即可调用（若 planner 改为自由函数，则直接调用自由函数）。

### 拒绝后状态不变量断言

```go
// 拒绝后必须：deleted_at 仍非空 + ad_dn 逐字不变（D-03 关键不变量）。
var after models.User
require.NoError(t, db.Unscoped().First(&after, "id = ?", f.userID).Error)
assert.True(t, after.DeletedAt.Valid, "拒绝恢复时用户必须保持软删")
require.NotNil(t, after.AdDn)
assert.Equal(t, f.storedDN, *after.AdDn, "拒绝路径绝不清空/改写存量 ad_dn")
```

来源：`models.User.AdDn *string` `[VERIFIED: internal/models/user.go:34]`；`BaseModel.DeletedAt gorm.DeletedAt` `[VERIFIED: internal/models/base.go:15]`。

### 批量路径：Skipped 记账与总量守恒

```go
result, err := svc.BatchSyncADUsers(ctx, users, "")
require.NoError(t, err)
assert.Equal(t, 1, result.Skipped, "DN 不一致的软删用户计入 Skipped")
assert.Equal(t, 0, result.Failed, "策略拒绝不是失败")
require.Len(t, result.Errors, 1)
assert.Contains(t, result.Errors[0].Error, "AD 身份不一致")
assert.NotContains(t, result.Errors[0].Error, "CN=", "返回前端的错误文本不得含 DN")
assert.Equal(t, result.Total, result.Success+result.Failed+result.Skipped)
```

来源：`BatchSyncResult` 字段 `[VERIFIED: internal/services/system/user_sync_service.go:473-480]`；handler 透传 `[VERIFIED: internal/api/v1/system/ad_domain_user_sync_handler.go:144-150]`；前端类型 `[VERIFIED: ruoyi-react-frontend/src/lib/adDomainApi.ts:675-686]`。

### 原因码映射测试

```go
// internal/core/security/ad_authenticator_phase60_test.go
func TestPhase60_SyncErrorReasonFor(t *testing.T) {
	assert.Equal(t, "user_sync", syncErrorReasonFor(errors.New("db down")))
	assert.Equal(t, SyncReasonSoftDeleteMismatch,
		syncErrorReasonFor(fmt.Errorf("wrap: %w", ErrSoftDeleteIdentityMismatch)))
}
```

```go
// internal/api/v1/auth_80_03_test.go —— 既有表驱动的期望值需要新增一行
{security.SyncReasonSoftDeleteMismatch, "该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员"},
```

来源：既有表驱动 `[VERIFIED: internal/api/v1/auth_80_03_test.go:623-640]`。

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| F-22：软删用户再次从 AD 登录时直接拒绝并 WARN，不自动恢复（`HandleUserLoginAD`） | 恢复但先校验 AD 身份 DN；一致才恢复，不一致拒绝 | v1.21 Phase 60 | 恢复能力保留，接管风险被 DN 校验挡住；旧路径已是死代码（Phase 62 删除） |
| `catSoftDeleted` 无条件恢复并覆盖 `ad_dn` | 校验通过才恢复 | v1.21 Phase 60 | 陈旧/被冒用的 AD 身份不能复活历史角色 |
| `BatchSyncResult.Skipped` 只打印不递增 | 策略拒绝首次写入真实语义 | v1.21 Phase 60 | 批量结果面板可区分「跳过」与「失败」 |
| 所有同步错误统一 `user_sync` | 安全拒绝使用专用原因码 | v1.21 Phase 60 | 运维可区分安全拦截与 AD 基础设施故障 |

**Deprecated/outdated:**
- `internal/services/addomain/user_ou_service.go` 的软删拒绝分支 — Phase 62 HYGIENE-03 删除对象；本 phase 只参考其 `[SECURITY]` WARN 风格，不修改。
- `restoreDeletedUserWithADInfo` — 同属死代码，不在本 phase 范围。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | 两条路径的 `currentUserDN` 在生产中通常非空（登录来自 LDAP `entry.DN`，批量来自 `sys_ad_user.user_dn`） | Summary / Pitfalls | 若 `sys_ad_user.user_dn` 存在空值，合法用户会被拒绝；属于安全侧失败（fail-closed），但需运维核对数据质量。`[ASSUMED]` |
| A2 | 前端结果面板会直接呈现 `skipped` 字段 | 批量记账 | 若前端未渲染该数字，用户看不到跳过数；但响应体已包含，后端可观测性不受影响。类型定义已验证，渲染路径未逐行确认。`[ASSUMED]` |
| A3 | 登录拒绝沿用 `response.ErrServerError`（HTTP 500 / code 500）不会破坏前端登录错误处理 | Open Questions | 若前端按 HTTP 状态码做特殊分支，语义上 500 与「安全 403」不同；但既有 `User==nil` 分支全部走该包装，保持一致性风险最低。`[ASSUMED]` |
| A4 | 批量路径阶段 2 的分类快照与阶段 5c 写库之间存在 TOCTOU 窗口可接受 | Architecture / Pitfalls | 管理员并发修改 `ad_dn` 的窗口极窄；批量同步本身是低频人工操作。若要求强一致，需在 5c 事务内重读并重新判定，复杂度上升。`[ASSUMED]` |
| A5 | `NormalizeDN` 对 multi-valued RDN（`CN=x+OU=y`）不做属性重排归一化；AD 用户 DN 通常不含该形态 | Pitfalls / 校验判定 | 若真实域控返回属性顺序不同的 multi-valued RDN，会 fail-closed 误拒；D-02 已锁定 `NormalizeDN`，本 phase 不扩展其语义。`[ASSUMED]` |
| A6 | 双侧 DN 的转义表示一致（都来自 LDAP，或都经过同一 `extractRDNFromDN` 回写） | 校验判定 | 若一侧是 `\,`、另一侧是 `\2C` 等不同转义，会 fail-closed 误拒；安全侧可接受，但需运维手工核对。`[ASSUMED]` |

**If this table is empty:** 不适用——上表 6 项需要 planner/用户在执行前确认或接受。

## Open Questions (RESOLVED)

> 全部 5 项已由 60-01 / 60-02 计划采纳推荐方案，无遗留待决项。

1. **拒绝时的 HTTP 响应码**
   - What we know: 现有 `authResult.User == nil` 分支统一用 `response.Error(c, response.ErrServerError, syncErrorReasonMessage(...))` `[VERIFIED: internal/api/v1/auth.go:211-216]`。
   - What's unclear: 安全拦截语义上更适合 `ErrForbidden`（403），但改变状态码会影响前端拦截器/日志分类。
   - Recommendation: 保持 `ErrServerError` + 专用文案（最小改动、与既有分支一致）；如用户要求 403，再单独加分支。
   - **Status: RESOLVED** — 60-01 Task 1 明确保持 `ErrServerError`，不新增 403 分支。

2. **原因码字符串命名**
   - What we know: CONTEXT 给出 `restore_refused` / `soft_delete_mismatch` 两个候选。
   - What's unclear: 无外部契约依赖。
   - Recommendation: `soft_delete_mismatch`——精确描述条件，便于 grep 与日志检索；若未来出现「其他恢复拒绝原因」（如目标行已被并发恢复），再引入更泛的 `restore_refused`。
   - **Status: RESOLVED** — 60-01 采用常量 `SyncReasonSoftDeleteMismatch = "soft_delete_mismatch"`。

3. **批量校验时机：阶段 2 vs 阶段 5c**
   - What we know: 阶段 2 过滤可跳过哈希、天然绕开 `runInBatchedTx`；阶段 5c 数据更新鲜但需要改造降级管道才能正确计入 `Skipped`。
   - What's unclear: 是否需要为极窄 TOCTOU 窗口付出复杂度。
   - Recommendation: 阶段 2（CONTEXT 已倾向推荐）；把 TOCTOU 作为已接受风险写入 plan 的 Assumptions。
   - **Status: RESOLVED** — 60-02 Task 1 在阶段 2 过滤并计入 `Skipped`；TOCTOU 已写入 60-02 Assumptions A4。

4. **`SyncADUser` 对策略拒绝的日志级别**
   - What we know: 当前 `SyncADUser` 在 `SyncUserFromAD` 返回 error 时用 `applogger.Errorf` `[VERIFIED: internal/services/system/user_sync_service.go:408-411]`。
   - What's unclear: 策略拒绝是否应降级为 WARN，避免与真实故障混淆。
   - Recommendation: 在 `SyncADUser` 内对 `errors.Is(err, security.ErrSoftDeleteIdentityMismatch)` 使用 `Warnf`，其余保持 `Errorf`；纯日志改动，低风险。
   - **Status: RESOLVED** — 60-01 Task 1 步骤 6 采纳；`ad_authenticator.go` 侧同样按 sentinel 分级。

5. **是否需要同步更新 `BatchSyncADUsers` 的汇总日志**
   - What we know: 完成日志已打印 `跳过=%d` `[VERIFIED: internal/services/system/user_sync_service.go:696-697]`，此前恒为 0。
   - What's unclear: 无。
   - Recommendation: 不改格式；`Skipped` 自本 phase 起才有真实值，日志立即获得可观测性。
   - **Status: RESOLVED** — 60-02 Task 1 步骤 3 明确不改阶段 6 汇总日志格式。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | 编译与测试 | ✓ | go1.24.5 windows/amd64 | — |
| `github.com/glebarez/sqlite` | 行为测试内存库 | ✓ | v1.11.0（`go.mod`） | 无（既有 fixture 已依赖） |
| `testify` | 断言 | ✓ | v1.11.1（`go.mod`） | 标准库 `testing` |
| LDAP/AD 服务 | 仅真实登录 UAT | ✗（单元测试不需要） | — | 单元测试直接调用 `SyncADUser` / `BatchSyncADUsers`，不经过 LDAP |
| PostgreSQL | 生产验证 | 未探测 | — | sqlite 覆盖行为；PG 差异仅涉及 `deleted_at` / 事务，GORM 抽象一致 |
| `gsd-tools.cjs` | 规划/状态工具 | ✓ | `C:/Users/CPIC/.claude/gsd-core/bin/gsd-tools.cjs` | 手工编辑 STATE/ROADMAP（Phase 58 先例） |

**Missing dependencies with no fallback:** 无（本 phase 单元测试不依赖外部服务）。

**Missing dependencies with fallback:** 真实 LDAP/UAT 依赖——留待人工 UAT；自动化测试通过 service 层直接构造 `ADUserInfo`。

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` + `testify` v1.11.1 + `glebarez/sqlite` v1.11.0 |
| Config file | none — 标准 `go test` |
| Quick run command | `go test ./internal/services/system/ -run 'TestPhase60' -count=1` |
| Full suite command | `go test ./internal/services/system/ ./internal/core/security/ ./internal/api/v1/ -count=1` |

实测基线：`go test ./internal/services/system/ -run 'TestClassifyUser|TestPhase59_SyncADUser_TerminalRowsDoNotGuard' -count=1` → `ok … 2.903s` `[VERIFIED: 本次会话执行]`。

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SAFE-01 | 存量 DN 与 AD 当前 DN 一致 → 登录路径恢复成功 | unit | `go test ./internal/services/system/ -run TestPhase60_Login_MatchingDN_Restores -count=1` | ❌ Wave 0 |
| SAFE-01 | 存量 DN 为空/nil/空白 → 放行（D-01） | unit | `go test ./internal/services/system/ -run TestPhase60_CheckSoftDeletedRestore_Predicate -count=1` | ❌ Wave 0 |
| SAFE-01 | DN 不一致 → 登录路径拒绝，行保持软删 | unit | `go test ./internal/services/system/ -run TestPhase60_Login_MismatchedDN_Refuses -count=1` | ❌ Wave 0 |
| SAFE-01 | 拒绝路径不清空存量 `ad_dn`（D-03） | unit | 同上（状态不变量断言） | ❌ Wave 0 |
| SAFE-01 | 拒绝写安全审计 WARN（字段完整） | unit | 同上（`captureLogs` 断言） | ❌ Wave 0 |
| SAFE-01 | 批量路径 DN 不一致 → `Skipped++` / `Errors` / 不降级 | unit | `go test ./internal/services/system/ -run TestPhase60_Batch_MismatchedDN_Skipped -count=1` | ❌ Wave 0 |
| SAFE-01 | 批量路径 DN 一致 → 恢复成功 | unit | `go test ./internal/services/system/ -run TestPhase60_Batch_MatchingDN_Restores -count=1` | ❌ Wave 0 |
| SAFE-02 | 同一 fixture 两路径判定与 DB 结果一致 | unit | `go test ./internal/services/system/ -run TestPhase60_PathConsistency -count=1` | ❌ Wave 0 |
| SAFE-02 | 登录路径 sentinel → 专用原因码 | unit | `go test ./internal/core/security/ -run TestPhase60_SyncErrorReasonFor -count=1` | ❌ Wave 0 |
| SAFE-02 | 原因码 → 中文文案 | unit | `go test ./internal/api/v1/ -run TestAuth8003_SyncErrorReasonMessage -count=1` | ✅ 扩展现有表 |

### 静态证伪守卫（可选，沿用 Phase 58/59 风格）

SAFE-02 的「单一入口」可用一条 grep 断言锁定：`internal/services/system/user_sync_service.go` 中 `NormalizeDN` 的生产调用只出现在共享校验函数内（期望计数 = 1），且 `BatchSyncADUsers` / `restoreSoftDeletedUser` 体内不出现 `NormalizeDN(`：

```bash
# 期望输出 1（共享校验函数内部的比较）
rg -n "NormalizeDN\(" internal/services/system/user_sync_service.go | wc -l
# 期望输出 0（两路径不得各自实现）
rg -n "func \(s \*UserSyncService\) (BatchSyncADUsers|restoreSoftDeletedUser)[\s\S]*NormalizeDN\(" internal/services/system/user_sync_service.go
```

注意：第二条 grep 的跨函数匹配需用 `-U`（multiline）且范围可能过宽；更可靠的做法是依赖行为一致性用例，静态断言仅作辅助。

### Sampling Rate

- **Per task commit:** `go test ./internal/services/system/ -run 'TestPhase60' -count=1`
- **Per wave merge:** `go test ./internal/services/system/ ./internal/core/security/ ./internal/api/v1/ -count=1`
- **Phase gate:** `go build ./... && go test ./...`（全绿后才进入 `/gsd-verify-work`）

### Wave 0 Gaps

- [ ] `internal/services/system/user_sync_service_phase60_test.go` — 软删行 fixture + 共享校验纯函数用例 + 登录/批量行为用例 + 两路径一致性用例（复用 `setupTestDBForSync` 与 `captureLogs`）
- [ ] `internal/core/security/ad_authenticator_phase60_test.go` — `syncErrorReasonFor` 纯函数用例
- [ ] `internal/api/v1/auth_80_03_test.go` — 现有原因码表新增 `soft_delete_mismatch` 期望行
- [ ] 无框架安装缺口（Go 测试基建已存在）

## Security Domain

> `workflow.nyquist_validation` 为 true；安全域必填。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | AD bind 成功后再做本地身份一致性校验；拒绝时不签发 JWT（`authResult.User == nil` 分支直接返回） |
| V3 Session Management | no | 不改 token/session |
| V4 Access Control | yes | 防止「删除用户的全部角色/权限被另一个 AD 身份接管」——DN 校验是授权对象绑定的前置条件 |
| V5 Input Validation | yes | 双侧 DN 经 `NormalizeDN` 归一化；DN 来源为 AD/DB，非用户直接输入 |
| V6 Cryptography | no | 无新增加密；沿用既有 PBKDF2 密码哈希（本 phase 仅在拒绝时避免计算） |

### Known Threat Patterns for Go + GORM + AD sync

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| AD 同名账号复活软删用户，继承历史角色（账号接管） | Elevation of Privilege | 存量 `ad_dn` 与 AD 当前 DN 归一化比较；不一致拒绝恢复；行保持软删 |
| 通过清空 `ad_dn` 绕过校验（二次登录变空 DN） | Tampering | 拒绝路径零写库；测试锁定 `ad_dn` 不变（D-03） |
| DN 细节泄露到用户可见响应 | Information Disclosure | 用户文案仅固定中文；DN 只进 WARN；`SyncError.Error` DN-free |
| 策略拒绝被伪装为 DB 故障（运维误判/重试放大） | Repudiation / DoS | 批量不抛 error、计入 `Skipped`；登录使用专用原因码 |
| 恶意构造超长/畸形 DN 触发解析开销 | DoS | `NormalizeDN` 为线性 split/trim/toLower，无解析器与递归，永不失败/panic `[VERIFIED: internal/services/addomain/dn_util.go:12-56]` |
| 并发恢复竞态导致校验被绕过 | Tampering | 登录路径的读与恢复在同一事务内；批量路径接受窄 TOCTOU 窗口（人工低频操作），并把最终写入值统一为当前 AD DN |

## Sources

### Primary (HIGH confidence)

- `internal/services/system/user_sync_service.go:79-115` — `SyncUserFromAD` 事务闭包与 `%w` 包装
- `internal/services/system/user_sync_service.go:129-143` — `classifyUser` 纯函数与分类优先级
- `internal/services/system/user_sync_service.go:152-192` — `indexExistingUsers` 预查询（软删行含 `AdDn`，零额外查询）
- `internal/services/system/user_sync_service.go:272-327` — `restoreSoftDeletedUser` / `applyRestoreSoftDeleted`
- `internal/services/system/user_sync_service.go:389-465` — `SyncADUser`（登录路径实现 `security.UserSyncer`）
- `internal/services/system/user_sync_service.go:473-480` — `BatchSyncResult`（`Skipped` 字段）
- `internal/services/system/user_sync_service.go:505-699` — `BatchSyncADUsers` 阶段 2/3/5c
- `internal/services/system/user_sync_service.go:732-770` — `runInBatchedTx` 批次降级语义
- `internal/services/addomain/dn_util.go:12-56` — `NormalizeDN` 归一化口径与空输入返回
- `internal/core/security/authenticator.go:28-37` — `AuthResult.SyncErrorReason`
- `internal/core/security/authenticator.go:64-70` — `UserSyncer` 接口
- `internal/core/security/authenticator.go:84-91` — 既有 sentinel 错误风格
- `internal/core/security/ad_authenticator.go:125-161` — 同步失败不崩认证 + `user_sync` 分支
- `internal/api/v1/auth.go:204-216` — `authResult.User == nil` 的如实反馈
- `internal/api/v1/auth.go:649-668` — `syncErrorReasonMessage` 原因码映射
- `internal/models/user.go:31-36` — `AdDn *string`（`column:ad_dn`）
- `internal/models/base.go:11-19` — `DeletedAt gorm.DeletedAt`
- `internal/services/addomain/user_ou_service.go:26-49` — F-22 对照语义与 `[SECURITY]` WARN 风格（只读参考，不修改）
- `internal/services/system/user_sync_service_test.go:16-165` — `setupTestDBForSync` sqlite fixture（`sys_user` 含 `ad_dn` / `deleted_at`）
- `internal/services/system/user_sync_service_phase59_test.go:143-148` — `captureLogs` 可复用
- `internal/api/v1/auth_80_03_test.go:623-640` — 原因码表驱动测试
- `internal/api/v1/system/ad_domain_user_sync_handler.go:101-150` — 批量端点结果透传
- `ruoyi-react-frontend/src/lib/adDomainApi.ts:676-686` — 前端 `BatchSyncResult.skipped` 类型
- `internal/core/core.go:903-911` — `UserSyncService` DI 注入

### Secondary (MEDIUM confidence)

- Context7 `/go-gorm/gorm` — `Unscoped` 查询软删行、事务回滚语义
- Context7 `/go-ldap/ldap` — `ParseDN` / `EqualFold` 语义（用于说明为何仍选 `NormalizeDN`）
- WebSearch（verified, MEDIUM）— Go `fmt.Errorf("%w")` + `errors.Is` 跨层错误识别

### Tertiary (LOW confidence)

- 无。本 phase 的所有关键结论均有仓库源码或框架官方文档支撑。

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — 无新依赖；版本来自 `go.mod` 与 `go version` 实测
- Architecture: HIGH — 两路径调用点、预查询字段、降级管道均逐行读源确认
- Pitfalls: HIGH — 基于 `runInBatchedTx` 实际降级逻辑与 `NormalizeDN` 实际实现；TOCTOU 风险评估为 MEDIUM
- External framework semantics: MEDIUM — Context7 文档确认，但实现以仓库既有契约为准

**Research date:** 2026-10-20
**Valid until:** 2026-11-19（30 天；纯仓库内代码改造，无外部 API 漂移风险）
