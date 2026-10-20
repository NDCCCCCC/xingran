# Phase 61: 影子表完整性（删除清理 + 组成员事务化） - Research

**Researched:** 2026-10-20
**Domain:** Go 后端 AD 域同步（`internal/services/addomain/`）— 影子表软删清理、组成员事务化、singleflight 互斥
**Confidence:** HIGH（全部关键结论来自本会话对源码的直接 Read，无外部依赖）

## Summary

本 phase 的三个需求全部落在 `internal/services/addomain/` 包内，**预计零 DDL、零新依赖、零新 API 端点**。核心发现：

1. **两道闸门已有完整先例可直接复用**：`GroupSyncService.handleDeletedGroups`（`group_sync_service.go:347-405`）已实现 LDAP 0 条拒绝（:351-354）与拟删 >50% 拒绝（:378-384，判式 `len(deletedDNs)*2 > len(staleGroups)`，严格大于——恰好 50% 放行），并有 4 个测试锁定语义（`group_sync_threshold_test.go:111-246`）。SHADOW-01 对 `sys_ad_user/ou/computer` 的清理应逐字复用该判式与分档（拒绝 = WARN + return nil 非阻断），不发明新阈值。
2. **SHADOW-02 的目标函数是 `SyncService.syncGroupMembers`（`sync.go:616-642`）**：现状是 `Unscoped()` 硬删旧成员 → 非 `memberDNs` 空则批量 `Create`，两步**不在事务内**——Create 半途失败会永久丢成员，且外部查询在两步之间能看到空成员窗口。修复 = Delete+Create 包 `s.db.WithContext(ctx).Transaction(...)`。该函数有 **3 个调用方**（`sync.go:374` 全量同步、`group_sync_service.go:160` 单组同步、`group_sync_service.go:336` 增量组同步），事务化后三处同时受益。
3. **最大的隐藏陷阱是「软删复活撞唯一约束」**：影子表的生产唯一约束**不含 `deleted_at`**（`sys_ad_user` 的 `uk_ad_user_dn UNIQUE(ad_config_id, user_dn)` 等，见 legacy 016 DDL :73/:111/:162；`sys_ad_computer` 模型 tag `uni_sys_ad_computer_config_name(ad_config_id, computer_name)`）。SHADOW-01 落地后，一条被软删的行若同 DN 在 AD 端复现，`syncUsers`/`syncOUs`/`syncComputers` 的默认作用域查询查不到它 → 走 Create 分支 → 撞唯一约束：OU 是裸 `Create`（`batchCreateOUs` 无 OnConflict，`sync.go:268-276`）会**让整轮同步失败**；User/Computer 是 OnConflict DoUpdates 会**更新到软删行上（永久不可见）**。清理逻辑必须配套「复活」路径（Go 层 Unscoped 恢复，符合用户禁 PG TRIGGER 偏好），且 `handleDeletedGroups` 对 `sys_ad_group` 已存在同款潜伏问题，建议一并覆盖。

**Primary recommendation:** 在 `syncDataInternal` 成功路径末尾（四类同步全部成功后）新增按类别独立的清理方法（复用 `handleDeletedGroups` 两道闸门 + 仅删 `deleted_at IS NULL` 存量），同时给三类 upsert 补「软删行复活」处理，`syncGroupMembers` 包 Transaction；不新增 migration（无 schema 变更）、不新增端点、不动 Phase 62 边界文件。

<user_constraints>

## User Constraints (from CONTEXT.md)

> 本 phase 无 `/gsd-discuss-phase` 产物（`61-CONTEXT.md` 不存在）。以下为 orchestrator 移交的既定口径 + 项目记忆中的用户硬约束，规划时按 LOCKED 处理。

### Locked Decisions（来自 orchestrator notes / REQUIREMENTS / 项目记忆）

- **需求即范围**：SHADOW-01、SHADOW-02、SHADOW-03 三项（REQUIREMENTS.md:27-31），Success Criteria 以 ROADMAP.md:222-227 四条为准。
- **migration 编号**：里程碑从 215 起顺延，当前仓库最高为 **migration_218**（`migration_218_ad_reconciliation_permission.go`，Phase 59 落地）[VERIFIED: internal/core/db/migrations/ 目录清单]。本 phase 若需迁移用 **219**；但研究结论是**预计不需要迁移**（软删列已存在，见 Architecture Patterns）。
- **禁用 PG TRIGGER**：数据格式/一致性修复走 Go 代码层（service 入口 + GORM hooks），不走 PG-level TRIGGER（项目记忆 user-prefers-code-fixes-no-db-triggers；REQUIREMENTS.md Out of Scope 表亦明确 "DB TRIGGER……trigger 是错误的层"）。
- **Phase 62 边界（本 phase 不得越界）**：`user_ou_service.go` 整文件、`GetUserDeptByADOU`、`BatchMoveUsersToNewOU`、`dept_ou_mapper.go:31` 过期注释是 HYGIENE-03（Phase 62）的删除对象，本 phase 不顺手删除。
- **与 QUEUE/SAFE 相互独立可并行**；phase 内 **SHADOW-03 依赖 SHADOW-01 先落地**（ROADMAP.md:220）——wave 排序必须 Wave 1 = SHADOW-01（含复活路径），后续 wave 才能挂载清理入口。
- **测试基线**：SQLite（glebarez 纯 Go 驱动）内存库 + 手动 CREATE TABLE fixture；`go test ./...` 全绿为门。参考 Phase 57 HYGIENE-06 补表做法。

### Claude's Discretion（研究建议区域）

- 清理结果的可见性载体（applogger / `SyncResult` 增字段 / `sys_ad_sync_log` 加列）——研究推荐前两者，避免 DDL。
- 「LDAP 返回 0 条」闸门按**类别**解释还是按**整轮**解释——研究推荐按类别（见 Open Questions Q1）。
- 复活路径的覆盖面（仅本 phase 三表 vs 连同 `sys_ad_group` 潜伏问题一起修）——研究推荐一并覆盖（见 Open Questions Q3）。

### Deferred Ideas (OUT OF SCOPE)

- HYGIENE-03/04/05（Phase 62）：死代码删除、嵌套组 member DN 标记/过滤、同名兄弟部门 OU DN 冲突告警。
- FUTURE-ADSYNC-01 uSNChanged 增量同步；FUTURE-ADSYNC-02 真 FK 约束；FUTURE-ADSYNC-03 一键按 AD 校正 dept。
- 时间戳仲裁、完整 Outbox、FieldRule 引擎、MQ（里程碑级 Out of Scope）。

</user_constraints>

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SHADOW-01 | 全量同步末尾对 AD 端已消失的 `sys_ad_user/ou/computer` 软删除，复用两道闸门（LDAP 0 条拒绝 / 拟删 >50% 拒绝） | 闸门先例逐字可复用：`group_sync_service.go:351-354`（0 条）+ `:378-384`（`*2 >` 50%）；清理挂 `syncDataInternal` 末尾（`sync.go:157` 成功后），LDAP 数据已在闭包内收集（`sync.go:99-124`）零额外 LDAP 往返；读路径均已过滤 `deleted_at IS NULL`（`user.go:50/88`、`ou.go:43/78`、`computer.go:94/151`），软删即从 UI 消失；必须配套复活路径防唯一约束撞车（Pitfall 1） |
| SHADOW-02 | `sys_ad_group_member` 重放改为单事务（Delete+Create 包 Transaction），消除空成员可见窗口 | 目标函数 `syncGroupMembers`（`sync.go:616-642`）现状两步无事务；3 个调用方（`sync.go:374`、`group_sync_service.go:160/336`）；GORM `Transaction` 闭包模式即可；sqlite 测试可用 CHECK 约束注入 Create 失败证伪回滚（见 Validation Architecture） |
| SHADOW-03 | 删除清理挂 `syncDataInternal`（复用 singleflight），不与全量同步并发 | 挂载点即 `syncDataInternal`（`sync.go:80`）成功路径；singleflight 已存在（`sync.go:30/63-64`，key=`sync:%s:%s`(configID, syncType)）；⚠ key 含 syncType，「full」与「incremental」不同 key 不互斥（Pitfall 5）；scheduler 实际只发 "full"（`ad_sync_tasks.go:332`），手动 API 可传其它 syncType（`ad_domain_handler.go:250-284`） |

</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 影子表删除清理（SHADOW-01） | API/Backend（`internal/services/addomain/` SyncService） | Database/Storage（软删列已存在） | 清理是同步编排的一部分，依赖闭包内已收集的 LDAP entries；无前端、无新端点 |
| 组成员事务化（SHADOW-02） | API/Backend（SyncService.syncGroupMembers） | — | 纯 service 层数据一致性修复，GORM Transaction |
| singleflight 互斥（SHADOW-03） | API/Backend（SyncService） | — | 既有 `singleflight.Group` 字段复用，无新并发原语 |
| 清理可观测性 | API/Backend（applogger + SyncResult JSON） | — | 无 DDL 前提下用日志 + 响应结构体字段 |

## Standard Stack

### Core

**零新增依赖。** 本 phase 只使用仓库既有组件：

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| gorm.io/gorm | v1.30.5 | `Transaction` 闭包 / 软删作用域 / `Unscoped()` | 仓库唯一 ORM，`go.mod` 既有 [VERIFIED: CLAUDE.md Technology Stack] |
| golang.org/x/sync/singleflight | 既有 | 同步互斥（已接线，不改） | `sync.go:14` 已 import，SHADOW-03 只是复用 |
| github.com/glebarez/sqlite | 既有 | 测试内存库 | `sync_78_05_test.go:29` 既有测试驱动 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Go 层复活路径（Unscoped 恢复软删行） | 改唯一约束为 partial unique index（`WHERE deleted_at IS NULL`） | partial index 需 DROP 既有 `uk_*` 约束再建——正踩 `cleanupOldConstraints` 42704 雷区（`database.go:395-423` 有整节前车之鉴），且 PG constraint 不支持 partial（只能 INDEX），双方言成本高。选 Go 层。 |
| `db.Transaction` 闭包 | 手动 `Begin/Commit/Rollback` | 闭包自动 rollback on error，GORM 惯用法，无泄漏面 |
| 按类别独立清理 | 一把梭全表清理 | 单类别 LDAP 返回 0（如域里真没 computer）不应牵连其它类别；与 group 先例（只管组）同构 |

**Installation:** 无需安装。

**Package Legitimacy Audit:** 本 phase 不安装任何外部包 → 无需运行 legitimacy gate，无 [SLOP]/[SUS] 项。

## Architecture Patterns

### System Architecture Diagram

```
手动 API (POST /ad-domain/configs/:id/sync, syncType 默认 full)      scheduler cron (每 5min, "full")
        │ (ad_domain_handler.go:250)                                  (ad_sync_tasks.go:332)
        ▼                                                                    │
        └────────────────► SyncService.SyncData ─────────────────────────────┘
                                │  singleflight.Do("sync:<configID>:<syncType>")   [sync.go:63-64]
                                ▼
                        syncDataInternal [sync.go:80]
                                │
                    ① Create ADSyncLog (running)
                                │
                    ② FailoverClient.ExecuteWithFailover 闭包内一次性收集 4 组 entries
                       ous / groups / users / computers   [sync.go:99-124]
                                │
                    ③ syncOUs → syncGroups(→syncGroupMembers×N) → syncUsers → syncComputers
                       [sync.go:135/142/149/157]
                                │                       任何一步失败 → updateSyncLog(failed) + return
                                ▼
                    ④ ★NEW Phase 61 清理段（SHADOW-01/03 挂载点，全部成功后才执行）
                       ├─ cleanupVanishedOUs(ous)       ─┐
                       ├─ cleanupVanishedUsers(userDNs)  ├ 每类独立两道闸门：
                       ├─ cleanupVanishedComputers(...) ─┘   gate1 len(entries)==0 → skip
                       │                                     gate2 拟删*2 > 存量 → skip
                       │                               通过 → 按 DN 差集 Unscoped 软删
                       │                               ★ 复活前提：③ 的 upsert 已能恢复软删行
                                ▼
                    ⑤ update last_sync_at + updateSyncLog(success)

        syncGroupMembers（SHADOW-02，被 ③ 的 syncGroups / 组同步两条路径共用）
            旧: tx 外 Delete(Unscoped) → Create      ← 空窗 + 半途失败丢数据
            新: db.Transaction( Delete → Create )    ← 原子，外部只见旧或新
```

### Recommended Project Structure

新增代码全部在 `internal/services/addomain/` 包内，建议单文件承载清理逻辑：

```
internal/services/addomain/
├── sync.go                        # syncDataInternal 末尾挂清理调用（SHADOW-03）；syncGroupMembers 事务化（SHADOW-02）
├── shadow_cleanup_service.go      # ★NEW：三类清理 + 两道闸门（SHADOW-01），方法挂在 SyncService 上
├── shadow_cleanup_test.go         # ★NEW：清理用例（复用 setupSync78DB fixture）
├── group_sync_service.go          # 只读参考：handleDeletedGroups 闸门先例（不改动，或 Q3 决定补复活时才动）
└── sync_78_05_test.go             # 既有 7 表 fixture（复用，不复制）
```

### Pattern 1: 两道闸门（逐字复用先例）

**What:** LDAP 返回 0 条 → 拒绝清理；拟删数 > 存量 50% → 拒绝清理。两种拒绝都是 WARN + return nil（非阻断、非 error）。
**When to use:** SHADOW-01 三类清理各自独立应用。
**Example（先例原文，`group_sync_service.go:351-354` 与 `:378-384`）:**

```go
// Gate 1 — [VERIFIED: internal/services/addomain/group_sync_service.go:351-354]
if len(ldapEntries) == 0 {
    applogger.Warnf("[组同步] 跳过过期组清理: LDAP 返回 0 条记录,可能 AD 不可达或 BaseDN 配置错误")
    return nil
}
// Gate 2 — [VERIFIED: internal/services/addomain/group_sync_service.go:378-384]
if len(staleGroups) > 0 && len(deletedDNs)*2 > len(staleGroups) {
    applogger.Warnf(
        "[组同步] 跳过过期组清理: 拟删除 %d/%d 个组(>50%%),"+
            "可能是 LDAP 返回不完整。请人工核实后手动清理",
        len(deletedDNs), len(staleGroups))
    return nil
}
```

判式语义（测试锁定）：`deletedDNs*2 > staleGroups` 严格大于 → **恰好 50% 放行**（`group_sync_threshold_test.go:242-245` 注释明示）。

### Pattern 2: DN 差集软删（存量以 `deleted_at IS NULL` 计）

```go
// 伪码骨架（SHADOW-01，每类一份；以 user 为例）
// 来源集合 = 本轮闭包已收集的 entries 经与 sync 相同的过滤后的 DN 集
//   user:   仅 sAMAccountName != "" 的 entry（sync.go:482-484 同款过滤）
//   computer: 仅 cn != "" 的 entry（computer.go:421-423 同款过滤）
//   ou:     全部 entry（syncOUs 无过滤）
ldapDNs := make(map[string]bool, len(entries))
for _, e := range entries { ldapDNs[e.DN] = true }

var stale []models.ADUser
s.db.WithContext(ctx).
    Where("ad_config_id = ? AND deleted_at IS NULL", config.ID).
    Find(&stale)                       // 存量 = 活行（闸门 2 的分母）

var vanished []string
for _, u := range stale {
    if !ldapDNs[u.UserDN] { vanished = append(vanished, u.UserDN) }
}
// gate1 / gate2 …
// 软删（GORM 默认作用域 Delete = UPDATE deleted_at）
s.db.WithContext(ctx).
    Where("ad_config_id = ? AND user_dn IN ?", config.ID, vanished).
    Delete(&models.ADUser{})
```

### Pattern 3: 软删行复活（清理的必要配套，防 Pitfall 1）

**What:** upsert 前对同唯一键的软删行先恢复（清 `deleted_at`），再走正常 update；或 Create 冲突时回退 Unscoped 恢复。
**When to use:** SHADOW-01 落地后，`syncOUs`/`syncUsers`/`syncComputers` 三条 upsert 路径都必须能处理「软删行 + 同 DN 复现」。
**Example（推荐写法：在 getExisting 查询处用 Unscoped 纳入软删行并复活）:**

```go
// sync.go getExistingOUs 附近（示意）——把软删行一并查出，命中即复活
var existingOUs []models.ADOU
s.db.WithContext(ctx).Unscoped().
    Where("ad_config_id = ? AND ou_dn IN ?", config.ID, ouDNs).
    Find(&existingOUs)
// categorizeOUs 命中软删行时：existingOU.DeletedAt = gorm.DeletedAt{}（清零）
// 并保证 upsert 的 AssignmentColumns 含 deleted_at，或单独一句
//   tx.Model(&models.ADOU{}).Unscoped().Where("id = ?", id).
//       Update("deleted_at", nil)
```

> 同款问题在 `sys_ad_group` 上**今天已潜伏**：`handleDeletedGroups` 软删组（`group_sync_service.go:387-391`）后，同 DN 组在 AD 复现 → `createGroupsInBatches` 裸 `Create`（`sync.go:383-400`）撞 `UNIQUE(ad_config_id, group_dn)` → 同步失败。见 Open Questions Q3。

### Pattern 4: 事务化重放（SHADOW-02）

```go
// sync.go:616-642 改造示意 —— Delete+Create 原子化
func (s *SyncService) syncGroupMembers(ctx context.Context, config *models.ADConfig, groupDN string, memberDNs []string) error {
    return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        if err := tx.Unscoped().
            Where("ad_config_id = ? AND group_dn = ?", config.ID, groupDN).
            Delete(&models.ADGroupMember{}).Error; err != nil {
            return fmt.Errorf("清理旧成员关系失败: %w", err)
        }
        if len(memberDNs) == 0 {
            return nil
        }
        members := make([]models.ADGroupMember, 0, len(memberDNs))
        for _, userDN := range memberDNs {
            members = append(members, models.ADGroupMember{
                ADConfigID: config.ID, GroupDN: groupDN, UserDN: userDN,
            })
        }
        if err := tx.Create(&members).Error; err != nil {
            return fmt.Errorf("批量添加组成员失败: %w", err)
        }
        return nil
    })
}
```

- GORM `Transaction` 闭包返回 error 自动回滚；PG READ COMMITTED 下外部查询要么见旧成员要么见新成员，无空窗。
- 保持**每组一个事务**（3 个调用方逐组调用），不要把整轮所有组包成一个大事务——锁面更大且违背「重放以组为单位」语义。

### Anti-Patterns to Avoid

- **清理放独立 goroutine / 独立 sys_job**：违背 SHADOW-03（必须借 syncDataInternal 的 singleflight 串行化），且项目明确禁 sys_job 承载 AD 排水/同步类后台件（STATE.md 58-05 D-05 口径）。
- **清理用硬删除**：ROADMAP Success Criteria 明写「软删除」；且 `sys_ad_user` 是对账 MV（`migration_168/176` JOIN `sys_ad_user`）的数据源，硬删破坏历史可追溯。
- **闸门拒绝时返回 error**：先例是 WARN + nil（`handleDeletedGroups` 调用方 `group_sync_service.go:80-83` 还把错误降级为非致命）。清理拒绝不是同步失败——整轮同步仍应 success。
- **把清理挂在某一步失败之后**：必须四类同步全部成功后才清理，否则 LDAP 数据不可信。
- **user/computer 清理集合不做 sync 同款过滤**：`SearchUsers` 过滤器排除 computer 与 `*DUPLICATE-*`（`ldap_client.go:207`），`syncUsers` 再跳过无 `sAMAccountName` 的 entry——清理集合必须与「本轮真正被 upsert 管理的行」同口径，否则会误删特殊对象行（Pitfall 2）。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 并发互斥 | 新 mutex / 新锁表 | 既有 `singleflight.Group`（`sync.go:30`） | 已覆盖 scheduler+手动+漂移三触发源，有测试（`sync_singleflight_test.go:29-79`） |
| 0 条 / 50% 闸门 | 新阈值算法 | 逐字复用 `handleDeletedGroups` 判式 | 语义已被 4 个测试锁定，含恰好 50% 边界 |
| 事务原子性 | 手动 Begin/Commit/扫描中间态 | `gorm.DB.Transaction` 闭包 | 自动回滚，GORM 惯用法 |
| 软删过滤 | 手写 `deleted_at IS NULL` 之外的过滤引擎 | GORM 默认作用域（`gorm.DeletedAt`）+ 显式 `Unscoped()` 逃生门 | BaseModel 已内建（`models/base.go:15`） |

**Key insight:** 本 phase 是「把既有模式推广到剩余三张影子表」，不是新机制设计。先例（`handleDeletedGroups`）已经把闸门语义、拒绝分档、成员级联清理全部定型，规划的任务应该是「复制语义 + 补复活路径」，任何新发明的阈值/状态机都是过度设计（本里程碑三方审计已两次砍掉同类过度设计）。

## Runtime State Inventory

> 本 phase 非 rename/refactor，但属于「数据生命周期变更」——首轮清理对存量数据的影响必须盘点。

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | 生产 `sys_ad_user/ou/computer` 中 AD 端已消失的行（`deleted_at IS NULL` 滞留）。**当前无任何代码软删这三张表**（全仓 grep `Delete(&models.ADUser/ADOU/ADComputer)` 零命中）[VERIFIED: grep 扫描 internal/]——SHADOW-01 是首个 writer。首轮全量同步将一次性软删全部历史滞留行 | 无需数据迁移脚本；但**首轮清理是一次性批量事件**：若 AD 端真的大批量消失，50% 闸门会拒绝（数据继续滞留 = 现状），规划须在计划里写明该预期行为，避免被当 bug 上报 |
| Live service config | 无——清理无需新增 sys_config 旋钮（闸门阈值 50% 硬编码与组先例一致；不建议加旋钮） | none |
| OS-registered state | 无 | None — verified by 需求范围（纯进程内同步逻辑） |
| Secrets/env vars | 无新密钥 | none |
| Build artifacts | 无新二进制外制品；`go build ./...` 常规验证 | none |

## Common Pitfalls

### Pitfall 1: 软删复活撞唯一约束（本 phase 最高风险）
**What goes wrong:** SHADOW-01 软删一行后，同 DN/cn 的对象在 AD 端复现（误删后自愈、OU 搬回、AD 回收站恢复等）。下一轮同步：默认作用域查询查不到软删行 → 判定「不存在」→ Create → 撞**不含 deleted_at 的唯一约束**。三种后果：`batchCreateOUs` 裸 Create → **整轮同步失败**；`syncUsers`/`batchUpdate`（OnConflict DoUpdates）→ **更新打到软删行上，该行永久不可见且每轮重演**。
**Why it happens:** 唯一约束是 constraint 级 `(ad_config_id, user_dn/ou_dn/group_dn)`（legacy 016 DDL :73/:111/:162）或模型 tag `uni_sys_ad_computer_config_name`（`models/ad_computer.go:18-19`），均不含 `deleted_at`；GORM 默认查询作用域排除软删行，两条规则叠加。
**How to avoid:** upsert 路径复活化（Pattern 3）：existing 查询改 `Unscoped()` 纳入软删行，命中即清 `deleted_at` 再 update。禁走改约束路线（`cleanupOldConstraints` 42704 雷区，`database.go:395-423`）。
**Warning signs:** 上线后出现「同步失败: duplicate key / UNIQUE constraint failed」或「AD 用户列表少人但 LDAP 里在」。

### Pitfall 2: 清理集合与 upsert 管理集合不同口径
**What goes wrong:** 直接用 `users []*ldap.Entry` 原始 DN 集做差集，会把「本轮因无 `sAMAccountName` 被 syncUsers 跳过」或「cn 为空被 syncComputers 跳过」的行误判为已消失而软删。
**Why it happens:** `syncUsers` 有 `username == "" → continue`（`sync.go:482-484`），`syncComputers` 有 `computerName == "" → continue`（`computer.go:421-423`），`userDNs` 只收有用户名的 DN（`sync.go:444-449`）。
**How to avoid:** 清理 DN 集 = 与各 sync 函数**相同过滤后**的集合；且 `userDNs` 为空时 `syncUsers` 直接 return（`sync.go:451-453`）——清理必须复用同一 entry 判定逻辑，而不是重新过滤一遍。
**Warning signs:** 清理上线后 `sys_ad_user` 出现非预期批量软删；闸门 2 频繁触发。

### Pitfall 3: 每类闸门分母口径
**What goes wrong:** 50% 闸门分母用「全表行数（含已软删）」→ 分母虚大，闸门失效；或用 Unscoped 查询统计。
**Why it happens:** 先例分母是 `staleGroups`——`deleted_at IS NULL` 的活行（`group_sync_service.go:364-366`）。
**How to avoid:** 分母恒为「该 config 下 `deleted_at IS NULL` 的活行数」，查询用默认作用域（不要 `Unscoped()`）。

### Pitfall 4: `handleDeletedGroups` 非 fatal 语义别丢
**What goes wrong:** 清理拒绝/失败时把整轮同步标 failed。
**Why it happens:** 误读先例——`SyncGroupsByConfig` 对 `handleDeletedGroups` 的错误只 `Warnf` 后继续（`group_sync_service.go:80-83`）。
**How to avoid:** 清理段内部吞掉可恢复异常（WARN），只有闸门放行后的软删 SQL 失败才需要上抛；即便上抛也只影响 sync_log 状态，不应回滚已成功的四类 upsert（它们已各自提交，无法回滚）。

### Pitfall 5: singleflight key 含 syncType，「full」≠「incremental」互斥
**What goes wrong:** 以为清理挂进 `syncDataInternal` 就与「一切同步」互斥。实际 key = `sync:%s:%s`（configID+syncType，`sync.go:63`），手动 API 可传 `incremental`（`ad_domain_handler.go:259-262` 默认 full 但接受任意值）→ 与 full 同步**并发**。
**Why it happens:** singleflight 天然按 key 分组。
**How to avoid:** 接受现状（增量同步与全量并发是既有行为，需求只锁「清理与全量同步不并发」——清理只跑在 full 末尾即满足）；**但**计划里不要写「清理因此与任何同步互斥」这类过强声明；如需更强互斥属 scope 扩张，须用户拍板。同时注意：即使同 key，增量同步如果也跑到清理段（若实现成不区分 syncType 都清理），会与「全量末尾清理」语义冲突——**清理应只在 `syncType == full` 分支执行**（Success Criteria 原文「全量同步末尾」）。
**Warning signs:** UAT 里手动发起 incremental 与 cron full 重叠时出现双份清理日志。

### Pitfall 6: `sys_ad_group_member` 无软删，事务内必须保持 Unscoped 硬删
**What goes wrong:** 顺手把 Delete 的 `Unscoped()` 去掉「统一风格」→ GORM 发出软删 UPDATE，而该表行会被每轮重放 Create → 行数爆炸 + `GetGroupSyncStatus` 计数虚高（`group_sync_service.go:198-202` 不滤 deleted_at）。
**Why it happens:** 模型嵌 `BaseModel`（有 `DeletedAt` 字段），但该表语义是「派生数据，随 LDAP 全量重放」，先例一律硬删（`sync.go:618`、`group_sync_service.go:394-397`）。
**How to avoid:** 事务化只包 Transaction，不改 Delete 的 Unscoped 语义。

### Pitfall 7: sqlite fixture 缺表/缺列
**What goes wrong:** 新清理代码触达 `sys_ad_user/ou/computer` 时，测试 fixture（`setupSync78DB`，`sync_78_05_test.go:47-229`）已有这三张表，但**若复活改造改动 existing 查询为 Unscoped**，fixture 的 `UNIQUE` 约束已存在（:100/:124/:160）无需动；新增 fixture 只在补 `sys_ad_computer` 相关用例时才需要（`setupSync78DB` **没有** sys_ad_computer 表，computer 用例在 `computer_78_06_test.go` 自建）。
**Why it happens:** 记忆教训（`sync_78_05_test.go:44-46` 注释）：被测函数触达的任何表都必须在 fixture 创建，报 `no such table` 先补表。
**How to avoid:** 清理用例统一复用 `setupSync78DB`（7 表已含 user/ou/group/group_member/config/sync_log/service_accounts），computer 清理用例复用 `computer_78_06_test.go` 的建表 helper；禁止再造第二套副本表（Phase 57/58 已多次纠偏「fixture 零副本」）。

### Pitfall 8: 本环境 gopls 诊断假阳性
**What goes wrong:** 编辑后 IDE 报 `undefined` 等编译错。
**Why it happens:** 项目记忆 `stale-gopls-diagnostics-verify-with-go-build`：诊断常因索引缓存滞后误报。
**How to avoid:** 一律 `go build ./...` 复核再动手改；逐文件修复后重建（CLAUDE.md 编译验证约定）。

## Code Examples

### 清理段挂载点（SHADOW-03）

```go
// sync.go syncDataInternal 成功路径末尾（:157 syncComputers 之后、:162 last_sync_at 前后皆可；
// 推荐放在 last_sync_at 更新之前，失败不阻断 success 状态）
if syncType == string(models.ADSyncTypeFull) {
    s.cleanupVanishedShadowRows(ctx, config, ous, users, computers, result)
}
```

### 组成员事务化证伪测试（SHADOW-02，sqlite CHECK 注入失败）

```go
// fixture 给 sys_ad_group_member 加 CHECK：user_dn = 'FORCED_FAIL' 时 INSERT 违约
// CREATE TABLE sys_ad_group_member (..., CHECK (user_dn != 'FORCED_FAIL'))
//
// 用例：预置 2 条旧成员 → 重放 members 含 1 条正常 + 1 条 'FORCED_FAIL'
// → syncGroupMembers 返回 error
// → 断言 Unscoped 查询：2 条旧成员原样仍在（Delete 已回滚）——这就是「无空窗」的可证伪形态
// （进程内 sqlite 无法起真并发观察者；PG READ COMMITTED 的窗口消除由事务语义保证，
//  可列入 UAT-61 人工项：真域控下重放期间另一连接 SELECT COUNT 不掉零）
```

### 闸门单元测试形态（对照先例逐条对齐）

先例 4 用例（`group_sync_threshold_test.go`）映射到新清理函数：
`RejectsEmptyLDAP`（0 条）、`RejectsOverThreshold`（>50%）、`AllowsUnderThreshold`（<50% 正常删）、`AtExactThreshold`（恰好 50% 放行）。新用例建议同名前缀 `TestShadowCleanup_*` 保持可检索。

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 组表无删除清理 → 全表误删事故 | P1-C2 两道闸门（0 条 / >50%） | Phase 38 P1 fix（代码注释标注 P1 fix） | SHADOW-01 直接继承该语义 |
| `syncGroupMembers` 两步无事务 | （本 phase）Transaction 包裹 | Phase 61 | 3 个调用方同时消除空窗与半途丢数据 |
| 手动 API 直推 AD | outbox-lite 队列唯一通路 | Phase 58 (58-06) | 本 phase 不碰推送路径 |
| 同步互斥 | singleflight by (configID, syncType) | 既有 | SHADOW-03 复用；key 含 syncType 的局限见 Pitfall 5 |

**Deprecated/outdated:** 无。`member_sync_service_test.go` 已是占位空文件（旧 DeptGroupMapping 体系已移除），不要在那儿加测试。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | 生产 PG 中 `sys_ad_user/ou/group` 上确实存在 016 DDL 声明的 `uk_ad_*` 约束（不含 deleted_at）。依据是 legacy 016 SQL + `cleanupOldConstraints` 清理清单（`database.go:414-418`，但清单写的是复数旧表名 `sys_ad_users` 等，与现表名单复数不一致，提示约束可能有历史变迁） | Pitfall 1 | 若生产实际无唯一约束：复活问题不会以「约束冲突」形态爆发，而是 OU 产生重复行——清理+复活路径仍是正确设计，仅故障形态不同；计划应含一条「启动冒烟后人工 `\d sys_ad_user` 核对约束」检查点 |
| A2 | 「LDAP 返回 0 条拒绝」按**每个类别独立**解释（OU 集为空只挡 OU 清理） | Open Questions Q1 / Architecture | 若用户意图是「任一类别为 0 就全部拒绝」：实现成 per-category 会更激进——但 per-category 下其它类别仍有各自闸门兜底，风险可控；讨论/plan checkpoint 向用户确认一次即可 |
| A3 | `SyncResult` 增加 Deleted 计数字段无兼容性风险（纯 JSON 增量字段，前端不读则忽略） | Standard Stack / 可观测性 | 极低；若前端对未知字段严格校验（未发现此模式）才会出问题 |
| A4 | 清理只挂在 `syncType == "full"` 分支（Success Criteria 原文「全量同步末尾」） | Pitfall 5 | 若用户期望 incremental 也清理：语义偏差但行为更保守（少删），可在 UAT 提出 |

## Open Questions

1. **闸门粒度：per-category 还是 per-run？**
   - What we know: 组先例是 per-category（只管组）；四类搜索独立发生，任一搜索失败会中止整轮同步（到不了清理段），所以「0 条」只可能来自「该类别在 AD 里真的为空」或过滤器极端情况。
   - What's unclear: 用户对「LDAP 本轮返回 0 条」的意图范围。
   - Recommendation: per-category（与先例同构，且每类各自有 50% 闸门兜底）；在 plan 的 checkpoint 里向用户确认一句即可。

2. **清理可观测性载体**
   - What we know: `SyncResult`（`sync.go:40-45`）与 `sys_ad_sync_log`（`models/ad_domain.go:194-209`）都只有正计数，无 deleted 计数。
   - What's unclear: 是否值得为 deleted 计数加 `sys_ad_sync_log` 列（→ 需要 migration_219 + 双方言 + BootstrapMissingTables 三处注册，Phase 57 F-03 教训）。
   - Recommendation: `SyncResult` 加字段 + applogger INFO/WARN，零 DDL。首版拒绝 DB 列方案。

3. **复活路径覆盖面：是否连 `sys_ad_group` 的潜伏同款问题一起修**
   - What we know: `handleDeletedGroups` 今天就在软删组（`group_sync_service.go:387-391`），同 DN 组复现会让下一轮 `createGroupsInBatches` 裸 Create 撞 `UNIQUE(ad_config_id, group_dn)`——这是**独立于本 phase 也会发生**的存量缺陷。
   - What's unclear: 修它是否算 scope 扩张（SHADOW-01 字面只列 user/ou/computer）。
   - Recommendation: 一并修（同一 Pattern、同一批测试、增量成本低），但 plan 里单列任务并标注「SHADOW-01 配套修复，先例路径同款缺陷」；若用户裁剪则本 phase 至少要在代码注释/RESEARCH 留警示。

4. **migration 是否真的为零**
   - What we know: 软删列（`deleted_at`）在 016 DDL 与模型 `BaseModel` 均已存在；`sys_ad_group_member` 缺的 `deleted_at/created_by/updated_by/version` 列由 AutoMigrate 补齐（`database.go:745` 注册）。本 phase 推荐 no-DDL。
   - What's unclear: 若 Q2 最终选 DB 列方案则需要 migration_219。
   - Recommendation: 锁定 no-DDL；编号 219 预留给未来（记录于 plan 头部备注即可）。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | 全部 | ✓ | go1.24.5 windows/amd64 | — |
| glebarez/sqlite（测试进程内库） | SHADOW-01/02 测试 | ✓ | go.mod 既有 | — |
| PostgreSQL | 生产运行（本机无实例） | ✗ 本机 | — | sqlite 测试覆盖不到 uuid/varchar 与真并发读——沿用 Phase 59 口径：PG 分支行为列 UAT 人工项 |
| LDAP 真域控 | 端到端验证 | ✗ 本机 | — | `[]*ldap.Entry` 字面量驱动测试（78-05 既有模式），真域控走 UAT |

**Missing dependencies with no fallback:** 无（本机缺 PG/域控为里程碑既定现状，非本 phase 引入）。

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | `go test`（testify require/assert + glebarez/sqlite 内存库） |
| Config file | 无（build tag `!skip_db_tests` 约定，见 `sync_78_05_test.go:1`） |
| Quick run command | `go test ./internal/services/addomain/ -count=1`（基线实测 **ok 8.165s**） |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SHADOW-01 | AD 消失行被软删（OU/User/Computer 各一） | unit (sqlite) | `go test ./internal/services/addomain/ -run TestShadowCleanup -count=1` | ❌ Wave 0（`shadow_cleanup_test.go`） |
| SHADOW-01 | gate1：该类 LDAP 0 条 → 不删 | unit | 同上 `-run TestShadowCleanup_RejectsEmptyLDAP` | ❌ Wave 0 |
| SHADOW-01 | gate2：拟删 >50% 拒绝；恰 50% 放行 | unit | 同上 `-run TestShadowCleanup_Threshold` | ❌ Wave 0 |
| SHADOW-01 | 复活：软删行同 DN 复现 → 下一轮同步恢复可见且不撞约束 | unit | `go test ./internal/services/addomain/ -run TestSync.*Resurrect -count=1` | ❌ Wave 0 |
| SHADOW-02 | Create 失败 → Delete 回滚、旧成员原样（无空窗的可证伪代理） | unit | `go test ./internal/services/addomain/ -run TestSyncGroupMembers -count=1`（更新既有 Task 4 用例 `sync_78_05_test.go:1218` 起） | ✅ 既有需改造 + ❌ 新增失败注入用例 |
| SHADOW-02 | 三个调用方路径行为不回归 | unit | 同上（`SyncSingleGroup` / `syncGroupEntries` 路径各有既有用例） | ✅ 既有 |
| SHADOW-03 | 清理只在 full 分支执行 / 挂载位置正确 | unit | `go test ./internal/services/addomain/ -run TestSync78_SyncDataByID -count=1`（失败路径既有）+ 新增挂载断言 | ✅/❌ |
| 全部 | 全套件不回归 | suite | `go test ./...` | ✅ |

### Sampling Rate

- **Per task commit:** `go build ./...` + `go test ./internal/services/addomain/ -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** 全绿 + `/gsd-verify-work`；PG/真域控行为列 UAT-61 人工项（首轮批量清理规模、事务空窗真观察、闸门真触发）

### Wave 0 Gaps

- [ ] `internal/services/addomain/shadow_cleanup_test.go` — SHADOW-01 全部用例，复用 `setupSync78DB`（勿新建副本表）
- [ ] `sync_78_05_test.go` fixture 扩展：`sys_ad_group_member` 建表语句加 `CHECK (user_dn != 'FORCED_FAIL')`（或独立建表 helper），供 SHADOW-02 失败注入
- [ ] computer 清理用例的建表：复用 `computer_78_06_test.go` 既有 helper（`setupSync78DB` 无 sys_ad_computer 表）
- 框架安装：无需（既有）

## Security Domain

> `config.json` 无 `security_enforcement` 键 → 视为启用。本 phase 无新端点、无新凭据面，安全面极窄。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 不触碰登录/JWT |
| V3 Session Management | no | — |
| V4 Access Control | no（无新端点） | 若意外新增端点：沿用 `pkg/middleware/permission.go` + 菜单 perms 模式（Phase 59 migration_218 精准授权先例） |
| V5 Input Validation | yes（弱） | syncType 参数已有默认值归一（`ad_domain_handler.go:259-261`）；清理函数只接受 service 内部构造的集合，不直收用户输入 |
| V6 Cryptography | no | 不触碰 SM2/SM3/SM4 |
| V14 Data Protection | yes | 删除是软删可追溯；清理动作必须 applogger 留痕（审计约定），拒绝场景 WARN 含拟删数量不含 DN 明文（对齐 Phase 60 `SyncError` 不含 DN 的口径） |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| LDAP 返回不完整 → 批量误删（Erosion of "source of truth"） | Tampering/Elevation | 两道闸门（复用先例）+ 清理只在全量成功末尾执行 |
| 软删行复活导致数据静默不可见 | Repudiation（数据丢失不可追溯） | Pattern 3 复活路径 + 软删可 Unscoped 审计 |
| 清理与同步并发产生中间态 | Tampering | singleflight（SHADOW-03）+ 清理只在 full 分支 |

## Sources

### Primary (HIGH confidence)

- 本会话 Read：`internal/services/addomain/sync.go`（全文 731 行）— SyncData/singleflight、syncDataInternal 编排、syncOUs/syncGroups/syncUsers/syncGroupMembers/updateSyncLog
- 本会话 Read：`internal/services/addomain/group_sync_service.go`（全文 406 行）— handleDeletedGroups 两道闸门先例、syncGroupMembers 调用方
- 本会话 Read：`internal/services/addomain/computer.go`、`user.go`、`ou.go`、`group.go`（节选）— 读路径 deleted_at 过滤、upsert 唯一键
- 本会话 Read：`internal/models/ad_domain.go`、`models/base.go`、`models/ad_computer.go`（tag）— 影子表模型与唯一索引 tag
- 本会话 Read：`internal/core/db/migrations/archive/legacy-2026-06-15/016_create_ad_domain_tables.sql`（全文）— 生产 DDL 唯一约束事实
- 本会话 Read：`internal/core/db/database.go`（:395-503、:720-829）— AutoMigrate 注册、cleanupOldConstraints 雷区、MV 依赖
- 本会话 Read：`internal/services/addomain/sync_78_05_test.go`（:1-330）、`group_sync_threshold_test.go`（全文）、`sync_singleflight_test.go`（全文）— 测试基座与闸门测试形态
- 本会话 Bash 实测：`go test ./internal/services/addomain/ -count=1` → ok 8.165s；`go version` → go1.24.5；migrations 目录清单（最高 218）

### Secondary (MEDIUM confidence)

- `.planning/ROADMAP.md`（Phase 61 段 :217-229）、`.planning/REQUIREMENTS.md`、`.planning/STATE.md`（58-xx 执行期决策：F-07 Pluck 位置铁律、fixture 零副本、禁 git stash 等执行纪律）

### Tertiary (LOW confidence)

- 无（本研究无 web/外部来源；所有外部包版本引自 CLAUDE.md Technology Stack，未在本 phase 重新安装验证）

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — 零新依赖，全部组件既有且实测
- Architecture: HIGH — 三个需求全部有同仓先例（handleDeletedGroups / GORM Transaction / singleflight），逐行核实
- Pitfalls: HIGH（Pitfall 1/2/5/6/7/8 均有源码行级证据）；A1（生产约束现状）为 ASSUMED 需运行时核对

**Research date:** 2026-10-20
**Valid until:** 2026-11-19（addomain 包在 Phase 62 前无其它变更计划；若 Phase 60 UAT 触发修复需复核挂载点行号）
