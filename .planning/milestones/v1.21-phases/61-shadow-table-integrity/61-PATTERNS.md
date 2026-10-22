# Phase 61: 影子表完整性（删除清理 + 组成员事务化） - Pattern Map

**Mapped:** 2026-10-20
**Files analyzed:** 6（2 新建 + 4 修改；模型/测试基座为只读参考）
**Analogs found:** 6 / 6（全部同包或同仓 tracked 源码，零镜像路径）

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/addomain/shadow_cleanup_service.go` ★NEW | service | batch（LDAP DN 集差集 → 软删） | `internal/services/addomain/group_sync_service.go`（`handleDeletedGroups`） | exact |
| `internal/services/addomain/shadow_cleanup_test.go` ★NEW | test | batch（sqlite fixture 驱动） | `internal/services/addomain/group_sync_threshold_test.go` + `sync_78_05_test.go` | exact |
| `internal/services/addomain/sync.go`（SHADOW-02 事务化） | service | batch（Delete+Create 重放） | `internal/services/addomain/dept_ou_mapper.go:93-137` | exact |
| `internal/services/addomain/sync.go`（复活路径，SHADOW-01 配套） | service | transform（upsert 前置复活） | `internal/services/addomain/user_ou_service.go:97-106`（备选形态 `network_device_service.go:296-316`） | exact |
| `internal/services/addomain/sync.go`（SHADOW-03 清理挂载） | service | event-driven（同步编排末段） | `internal/services/addomain/sync.go` 自身 `syncDataInternal` 编排段（:132-175） | self（结构性延伸） |
| `internal/services/addomain/group_sync_service.go`（仅 Q3 选做复活） | service | transform | `internal/services/addomain/user_ou_service.go:97-106` | role-match |
| `internal/services/addomain/sync_78_05_test.go`（fixture 扩展 CHECK） | test（fixture） | — | 自身 `setupSync78DB`（:47-229） | self |

**Phase 62 边界警示：** `user_ou_service.go` 整文件是 Phase 62（HYGIENE-03）的删除对象。此处只把它当**复活模式的抄写来源**——把 `Update("deleted_at", nil)` 这一行手法复制进新代码，不要在 plan/代码注释里建立对该文件的长期引用；复活逻辑落地后它对 Phase 62 的删除无阻碍。

## Pattern Assignments

### `internal/services/addomain/shadow_cleanup_service.go` ★NEW（service, batch）

**Analog:** `internal/services/addomain/group_sync_service.go` — `handleDeletedGroups`（:346-405）

**方法挂载形态**（对照 :347 签名；新方法挂在 `SyncService` 上，同包 `s.db *gorm.DB` 直用）：

```go
// [VERIFIED: internal/services/addomain/group_sync_service.go:346-347]
// handleDeletedGroups detects groups that exist in DB but not in LDAP and soft-deletes them
func (s *GroupSyncService) handleDeletedGroups(ctx context.Context, config *models.ADConfig, ldapEntries []*ldap.Entry, result *GroupSyncResult) error {
```

新签名建议同构：`func (s *SyncService) cleanupVanishedShadowRows(ctx context.Context, config *models.ADConfig, ous, users, computers []*ldap.Entry, result *SyncResult)`，内部再拆 per-category 私有方法。

**Gate 1 — LDAP 0 条拒绝**（:348-354，逐字复用判式与文案风格）：

```go
// P1 fix: 安全闸门 — 如果 LDAP 返回为空,直接拒绝任何删除。
// 否则 AD 不可达/网络抖动/搜索 BaseDN 配错都会导致整个 DB 组全表软删除,
// 此时用户彻底失去所有 AD 组关联,恢复极其困难。
if len(ldapEntries) == 0 {
    applogger.Warnf("[组同步] 跳过过期组清理: LDAP 返回 0 条记录,可能 AD 不可达或 BaseDN 配置错误")
    return nil
}
```

**存量查询 + DN 差集**（:356-373，注意分母 = 默认作用域活行，禁 `Unscoped()`）：

```go
// Build a set of current LDAP group DNs
ldapGroupDNs := make(map[string]bool, len(ldapEntries))
for _, entry := range ldapEntries {
    ldapGroupDNs[entry.DN] = true
}

// Find groups in DB that are not in LDAP
var staleGroups []models.ADGroup
s.db.WithContext(ctx).
    Where("ad_config_id = ? AND deleted_at IS NULL", config.ID).
    Find(&staleGroups)

var deletedDNs []string
for _, group := range staleGroups {
    if !ldapGroupDNs[group.GroupDN] {
        deletedDNs = append(deletedDNs, group.GroupDN)
    }
}
```

**Gate 2 — 拟删 >50% 拒绝**（:376-384，严格大于，恰 50% 放行）：

```go
// P1 fix: 阈值保护 — 若拟删除超过现有 DB 组的 50%,拒绝并要求人工确认。
// 防止 LDAP 返回严重不完整(部分 OU 搜索失败/分页中断)时误删大量数据。
if len(staleGroups) > 0 && len(deletedDNs)*2 > len(staleGroups) {
    applogger.Warnf(
        "[组同步] 跳过过期组清理: 拟删除 %d/%d 个组(>50%%),"+
            "可能是 LDAP 返回不完整。请人工核实后手动清理",
        len(deletedDNs), len(staleGroups))
    return nil
}
```

**软删写法**（:387-391；影子表清理用 GORM 默认作用域 Delete = UPDATE deleted_at）：

```go
// Soft-delete stale groups
if err := s.db.WithContext(ctx).
    Where("ad_config_id = ? AND group_dn IN ?", config.ID, deletedDNs).
    Delete(&models.ADGroup{}).Error; err != nil {
    return fmt.Errorf("删除过期用户组失败: %w", err)
}
```

**成员级联硬删（仅 group 类别需要；`sys_ad_group_member` 是派生数据，保持 `Unscoped()`）**（:393-398）：

```go
// Also clean up member relations (hard delete since they are derived data)
if err := s.db.WithContext(ctx).Unscoped().
    Where("ad_config_id = ? AND group_dn IN ?", config.ID, deletedDNs).
    Delete(&models.ADGroupMember{}).Error; err != nil {
    applogger.Warnf("[组同步] 清理过期组成员关系失败: %v", err)
}
```

**Imports pattern**（同包文件标准导入块，`group_sync_service.go:1-14`）：

```go
import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/go-ldap/ldap/v3"
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
    applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)
```

**清理集合口径（Pitfall 2 防误删）——必须复用 sync 函数的既有过滤，不要重新发明：**

- User：仅 `sAMAccountName != ""` 的 entry（`sync.go:443-449` 构造 `userDNs` 与 `sync.go:482-484` 的 `username == "" → continue` 同口径）
- Computer：仅 `cn != ""` 的 entry（`computer.go:419-423`）
- OU：全部 entry（`syncOUs` 无过滤，`sync.go:179-194`）

**per-category 独立应用**：gate1/gate2 在每个类别内部各自判断（OU 为空只挡 OU 清理），与先例「只管组」同构。

---

### `internal/services/addomain/sync.go`（SHADOW-02 事务化，batch）

**Analog:** `internal/services/addomain/dept_ou_mapper.go` — `UpsertMapping`（:89-137），同包最近的「delete→create 包单事务」先例，含 P1 fix 理由注释范式：

```go
// [VERIFIED: internal/services/addomain/dept_ou_mapper.go:93-102]
func (m *DeptOUmapper) UpsertMapping(ctx context.Context, mapping *models.DeptOUMapping) error {
    // P1 fix: 把"先 delete 旧映射 → 再 Create 新映射"包在单事务中。
    // 原实现中两步在独立事务,delete 成功后 crash/网络抖动会让旧映射消失
    // 而新映射未建,等同于丢失整条映射关系。
    return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
```

**改造目标原文**（`sync.go:615-642` — 现状两步无事务，`Unscoped()` 语义保留，Pitfall 6）：

```go
// syncGroupMembers 同步用户组成员关系
func (s *SyncService) syncGroupMembers(ctx context.Context, config *models.ADConfig, groupDN string, memberDNs []string) error {
	// 先删除旧的成员关系
	if err := s.db.WithContext(ctx).Unscoped().Where("ad_config_id = ? AND group_dn = ?", config.ID, groupDN).
		Delete(&models.ADGroupMember{}).Error; err != nil {
		return fmt.Errorf("清理旧成员关系失败: %w", err)
	}
	if len(memberDNs) == 0 {
		return nil
	}
	// 准备批量数据
	members := make([]models.ADGroupMember, 0, len(memberDNs))
	for _, userDN := range memberDNs {
		members = append(members, models.ADGroupMember{
			ADConfigID: config.ID,
			GroupDN:    groupDN,
			UserDN:     userDN,
		})
	}
	if err := s.db.WithContext(ctx).Create(&members).Error; err != nil {
		return fmt.Errorf("批量添加组成员失败: %w", err)
	}
	return nil
}
```

改造 = 把 Delete+Create 移入 `s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {...})` 闭包，体内 `s.db` 全换 `tx`；`Unscoped()` 不动；空 `memberDNs` 提前 `return nil` 仍在事务内（Delete 已提交）。**保持每组一个事务**（调用方逐组调用），不要把整轮所有组合成一个大事务。

**三个调用方（零改动，事务化后同时受益）：**
- `sync.go:374`（`syncGroups` 内逐组）
- `group_sync_service.go:160`（`SyncSingleGroup`）
- `group_sync_service.go:336`（`syncGroupEntries` 增量，失败仅 `Warnf` 继续）

---

### `internal/services/addomain/sync.go`（复活路径，SHADOW-01 配套，transform）

**Analog A（首选形态，同包）：** `internal/services/addomain/user_ou_service.go:97-106` — `Update("deleted_at", nil)` 复活软删行：

```go
// restoreDeletedUserWithADInfo 恢复被软删除的用户并补充AD信息
// 用于处理用户被软删除后再次从AD登录的场景
func (s *UserOUService) restoreDeletedUserWithADInfo(ctx context.Context, username, userDN, ouDN, deptID string) error {
	// 1. 恢复软删除用户（设置 deleted_at 为 NULL）
	if err := s.db.WithContext(ctx).
		Model(&models.User{}).
		Where("username = ?", username).
		Update("deleted_at", nil).Error; err != nil {
		return fmt.Errorf("恢复用户失败: %w", err)
	}
```

**Analog B（备选形态，跨模块）：** `internal/services/network_device_service.go:296-316` — Unscoped 查出软删行 → 清零 `gorm.DeletedAt` → `Save`：

```go
// 如果设备存在但已删除，恢复并更新
if err == nil && existingDevice.DeletedAt.Valid {
    // 恢复已删除的设备（清空 deleted_at）
    existingDevice.DeletedAt = gorm.DeletedAt{}
    ...
    if err := s.db.WithContext(ctx).Save(&existingDevice).Error; err != nil {
        return nil, fmt.Errorf("恢复设备失败: %w", err)
    }
}
```

**改造点（在 `sync.go` 既有查询函数处把软删行纳入并复活）：**

| 函数 | 现状 | 复活改造 |
|------|------|---------|
| `getExistingOUs`（:221-225） | 默认作用域 `Where("ad_config_id = ? AND ou_dn IN ?")` | 改 `Unscoped()` 查出软删行；`categorizeOUs`（:228-265）命中软删行时清 `DeletedAt` 后走 update 分支，且 upsert 的 `AssignmentColumns`（:298-303）需补 `deleted_at`，或对命中行单独一句 `Update("deleted_at", nil)` |
| `syncUsers` existing 批查（:456-467） | 默认作用域 | 同上；`usersToUpdate` 的 OnConflict `AssignmentColumns`（:594-603）补 `deleted_at` 或单句复活 |
| `syncGroups` existing 查询（:318-320） | 默认作用域 | 同上；`updateGroupsInBatches` OnConflict（:420-428）补 `deleted_at` |
| `computer.go` upsert（:662-673） | OnConflict `(ad_config_id, computer_name)` | existing 查询 Unscoped 化 + 复活；否则 OnConflict 把更新打到软删行（永久不可见） |
| `batchCreateOUs`（:268-276）/ `createGroupsInBatches`（:383-400） | **裸 `Create`，无 OnConflict** | 若复活在 existing 查询层做足，软删行进入 update 分支，裸 Create 不再撞 `UNIQUE(ad_config_id, ou_dn/group_dn)`；这是 OU 类别唯一的防炸点 |

**禁走路线：** 不改唯一约束为 partial unique index（`cleanupOldConstraints` 42704 雷区，`internal/core/db/database.go:395-423` 有整节前车之鉴）；禁 PG TRIGGER（用户硬约束，走 Go service 层 + GORM hook）。

---

### `internal/services/addomain/sync.go`（SHADOW-03 清理挂载，event-driven）

**Analog:** 自身编排段（:132-175）——四类同步成功路径 + 失败即 `updateSyncLog(failed)` 早退。挂载位置：四类全部成功之后、`last_sync_at` 更新（:162-169）之前，且**只在 `syncType == full` 分支**（`models.ADSyncTypeFull`，`internal/models/ad_domain.go:180`）：

```go
// sync.go:171 现状（挂载点在它之前）
s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusSuccess, result.OUCount, result.GroupCount, result.UserCount, result.ComputerCount, "")
```

**singleflight 复用（不新增并发原语）**（:30 + :63-66）：

```go
syncGroup singleflight.Group
...
key := fmt.Sprintf("sync:%s:%s", config.ID, syncType)
v, err, shared := s.syncGroup.Do(key, func() (interface{}, error) {
    return s.syncDataInternal(ctx, config, syncType)
})
```

⚠ key 含 syncType：full 与 incremental 不互斥（既有行为，不扩大声明）；清理只跑 full 分支即满足 SHADOW-03。

**拒绝语义（Pitfall 4）**：清理拒绝/内部失败 = WARN + nil，**整轮同步仍 success**。先例调用方就是非致命降级（`group_sync_service.go:80-83`）：

```go
if err := s.handleDeletedGroups(ctx, &config, entries, result); err != nil {
    applogger.Warnf("[组同步] 处理已删除组失败: %v", err)
    // Non-fatal: continue
}
```

**可观测性**：`SyncResult`（`sync.go:40-45`）加 Deleted 计数 JSON 字段 + `applogger` INFO/WARN；不迁 DB 列。拒绝日志含拟删数量、不含 DN 明文。

---

### `internal/services/addomain/shadow_cleanup_test.go` ★NEW（test）

**Analog 1（闸门四用例，逐条对齐命名）：** `internal/services/addomain/group_sync_threshold_test.go`

- 文件头 build tag（:1-2）：`//go:build !skip_db_tests` + `// +build !skip_db_tests`
- 空 LDAP 拒绝：`TestHandleDeletedGroups_RejectsEmptyLDAP`（:111-140）
- 超阈值拒绝：`TestHandleDeletedGroups_RejectsOverThreshold`（:145-180）
- 阈值内放行：`TestHandleDeletedGroups_AllowsUnderThreshold`（:185-215）
- 恰 50% 放行（严格大于边界）：`TestHandleDeletedGroups_AtExactThreshold`（:219-245），判式注释原文：

```go
// The production code uses len(deletedDNs)*2 > len(staleGroups), which
// means 5*2=10 > 10 is FALSE → deletion proceeds at exactly 50%.
```

新用例统一前缀 `TestShadowCleanup_*`（`RejectsEmptyLDAP` / `RejectsOverThreshold` / `AllowsUnderThreshold` / `AtExactThreshold` + 各表 Resurrect 用例）。

**Analog 2（fixture 与驱动 helper，复用禁复制）：** `internal/services/addomain/sync_78_05_test.go:39-276`

- `setupSync78DB(t)`：7 表 sqlite 内存库，**已含** `sys_ad_user`（:130-162，含 `UNIQUE(ad_config_id, user_dn)`）、`sys_ad_ou`（:82-102，`UNIQUE(ad_config_id, ou_dn)`）、`sys_ad_group`（:106-126）、`sys_ad_group_member`（:166-179）、`sys_ad_config`、`sys_ad_sync_log`、`sys_ad_service_accounts`。**无 `sys_ad_computer` 表**
- `entry78(dn, kv)`（:232-238）：`[]*ldap.Entry` 字面量构造，零 LDAP 网络
- `newSyncSvc78(t)`（:242-247）：`&SyncService{db: db, pool: NewAccountPool(db, nil)}` 白盒直调私有方法
- `insertConfig78(t, db, id)`（:250-266）/ `closeDB(t, db)`（:269-276）/ `itoaForTest(i)`（:85-105，定义在 threshold_test.go）

```go
func newSyncSvc78(t *testing.T) (*SyncService, *gorm.DB) {
	t.Helper()
	db := setupSync78DB(t)
	svc := &SyncService{db: db, pool: NewAccountPool(db, nil)}
	return svc, db
}
```

**Analog 3（computer 表 fixture，复用勿重建）：** `internal/services/addomain/computer_78_06_test.go:47-51` — `setupComp78DB` 包装 `setupSync78DB` 再补 `sys_ad_computer` 建表，computer 清理/复活用例照此组合：

```go
func setupComp78DB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupSync78DB(t)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_computer ( ... ).
```

**软删行预置手法（既有用例直接示范复活测试的另一半）：** `sync_78_05_test.go:314-334` `TestSync78_GetExistingOUs` —— `db.Create(ouB)` 后 `db.Delete(ouB)` 造软删行，再断言默认作用域查不到（这正是复活改造后要翻转的行为）：

```go
ouB := &models.ADOU{ADConfigID: cfg.ID, OUN: "OU=B,DC=example,DC=com", OUName: "B", LastSyncAt: &now}
ouB.ID = uuid.NewString()
require.NoError(t, db.Create(ouB).Error)
require.NoError(t, db.Delete(ouB).Error) // 软删
```

**SHADOW-02 失败注入形态：** `sync_78_05_test.go:166-179` 的 `sys_ad_group_member` 建表加 `CHECK (user_dn != 'FORCED_FAIL')` → 预置 2 条旧成员 → 重放含 1 正常 + 1 `FORCED_FAIL` → 断言返回 error 且 `Unscoped` 查询 2 条旧成员原样仍在（Delete 已回滚 = 无空窗可证伪代理）。

**既有 SHADOW-02 行为锚点（改造后此用例的 Drop Table 分支语义不变）：** `TestSync78_SyncGroupMembers_FullDiff`（:1226-1275）。

**禁地：** `member_sync_service_test.go` 是 7 行占位空文件（旧 DeptGroupMapping 体系已移除），不往里加测试。

---

### 模型参考（只读，零修改）

| 模型 | 文件:行 | 唯一键（不含 deleted_at → Pitfall 1 根因） |
|------|---------|------------------------------------------|
| `ADUser` | `internal/models/ad_domain.go:114-117` | `uniqueIndex:uni_sys_ad_user_config_dn (ad_config_id, user_dn)` |
| `ADGroup` | `internal/models/ad_domain.go:94-97` | `uniqueIndex:uni_sys_ad_group_config_dn (ad_config_id, group_dn)` |
| `ADGroupMember` | `internal/models/ad_domain.go:163-172` | 无唯一约束；`TableName() = "sys_ad_group_member"`；嵌 `BaseModel` 但语义为派生数据硬删重放 |
| `ADOU` | `internal/models/ad_domain.go:50-61` | fixture 层 `UNIQUE(ad_config_id, ou_dn)`；模型 tag 无 uniqueIndex（生产约束来自 016 DDL） |
| `ADComputer` | `internal/models/ad_computer.go:16-20` | `uniqueIndex:uni_sys_ad_computer_config_name (ad_config_id, computer_name)` |
| `BaseModel` | `internal/models/base.go:11-27` | `DeletedAt gorm.DeletedAt`（默认作用域过滤来源）+ `BeforeCreate` 自动填 UUID |
| `ADSyncType` | `internal/models/ad_domain.go:177-182` | `ADSyncTypeFull = "full"`（挂载分支判据，禁裸字符串） |

## Shared Patterns

### GORM Transaction 闭包（自动回滚）
**Source:** `internal/services/addomain/dept_ou_mapper.go:93-137`
**Apply to:** `syncGroupMembers`（SHADOW-02）
`s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })` — 闭包体内一律用 `tx`；返回 error 自动回滚；注释写明「delete 成功后 crash 会让旧数据消失而新数据未建」的动机（同包先例注释范式）。

### 两道闸门（0 条 / >50%，WARN + nil 非阻断）
**Source:** `internal/services/addomain/group_sync_service.go:348-384`
**Apply to:** `shadow_cleanup_service.go` 每个类别独立应用；分母恒为默认作用域活行数；判式 `len(deletedDNs)*2 > len(stale)` 严格大于逐字保留。

### 软删复活（`Update("deleted_at", nil)`，Go 层）
**Source:** `internal/services/addomain/user_ou_service.go:97-106`（首选）；`internal/services/network_device_service.go:296-316`（`gorm.DeletedAt{}` + Save 备选）
**Apply to:** `getExistingOUs` / `syncUsers` existing 批查 / `syncGroups` existing 查询 / `computer.go` upsert 查询，全部 `Unscoped()` 化 + 命中即复活；禁 partial unique index、禁 PG TRIGGER。

### 测试基座（build tag + sqlite 内存库 + 字面量驱动）
**Source:** `internal/services/addomain/sync_78_05_test.go:1-49, 232-276`；`group_sync_threshold_test.go:1-63`
**Apply to:** `shadow_cleanup_test.go` 全部用例；fixture 零副本（`setupSync78DB` / `setupComp78DB` 组合复用）；状态常量一律 `models.*` 禁裸 0/1。

### 日志 + 非致命降级
**Source:** `internal/services/addomain/group_sync_service.go:80-83`（调用方降级）、:351-354/:379-383（WARN 文案格式）
**Apply to:** 清理段所有拒绝/可恢复失败路径；`applogger.Warnf("[X同步] 跳过…清理: …")` 文案风格对齐；拒绝日志不含 DN 明文。

## No Analog Found

无。全部目标文件均有同包 exact/role-match 先例。

备注两处「self + 扩展」：SHADOW-03 挂载是 `syncDataInternal` 编排段的自然延伸（无独立先例文件）；`sync_78_05_test.go` fixture 扩展是自身 `setupSync78DB` 的一行式增量。

## Metadata

**Analog search scope:** `internal/services/addomain/`（全包）、`internal/models/`、`internal/services/`（Transaction/复活先例横向扫描 `internal/` 全量 grep）
**Files scanned:** 14（深读 9：sync.go 全文、group_sync_service.go 全文、group_sync_threshold_test.go 全文、sync_78_05_test.go 两段、dept_ou_mapper.go 一段、user_ou_service.go 一段、network_device_service.go 一段、ad_computer.go 全文、base.go 全文；定位扫描 5：ad_domain.go、computer.go、computer_78_06_test.go、sync_result.go、member_sync_service_test.go）
**Tracked-source gate:** 全部 analog 路径经 `git ls-files` 核实为 tracked 源码，无 gitignored 镜像路径
**Pattern extraction date:** 2026-10-20
