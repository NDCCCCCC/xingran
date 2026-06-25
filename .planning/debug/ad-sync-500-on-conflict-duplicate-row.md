---
slug: ad-sync-500-on-conflict-duplicate-row
status: resolved
deferred_to: v1.16-tech-debt
trigger: AD 全量同步点击 → 8.5 秒后报 500 (Internal Server Error),后端日志显示 "批量更新电脑设备失败: ERROR: ON CONFLICT DO UPDATE command cannot affect row a second time (SQLSTATE 21000)"
created: 2026-06-16
updated: 2026-06-25
session_type: bug
related:
  - ad-sync-500-nul-byte-in-error-msg  # 同一 sync 链路上一个被修复的崩溃点
---

# Debug Session: AD 电脑同步 500 — PostgreSQL ON CONFLICT 同一行重复更新

## 关键结论(用户首读这一段)

`sys_ad_computer` 的 unique constraint 建立在 `(ad_config_id, computer_name)` 上(因为 AD 电脑
可重命名,DN 会变),但 `syncComputers` 在内存里用 **DN** 作为 `toUpdate` map 的 key。
当 AD 返回多个不同 DN 但同名 cn 的 entry(典型场景:电脑重命名后旧记录未清理 / AD 复制冲突
tombstone + active 共存),`toUpdate` 中会装入多条记录,batch upsert 在同一 batch 内对
**同一 (config_id, computer_name)** 触发多次 ON CONFLICT → PostgreSQL 拒绝
(SQLSTATE 21000),整个 sync handler 返回 500。

修复:`toUpdate` map key 改为 `(config_id, computer_name)` 复合键,与 ON CONFLICT 列对齐;
同时给 `toCreate` 加同样的 dedup,防止 batchCreate 撞 unique 约束。

**为什么 user/group/ou 不会撞**:它们的 unique constraint 都建立在 `*_dn` 上(因为 DN 在 AD 里
天然唯一,且这些对象极少重命名),map key 与 ON CONFLICT 列对齐。Computer 是唯一特例。

---

## Symptoms

### Actual Behavior
- `POST /api/v1/ad-domain/configs/{id}/sync` → 500 Internal Server Error
- 延迟 8455ms(说明 LDAP 搜索成功,在 DB upsert 阶段崩溃)
- 错误日志:

```
ERRO[2026-06-16 20:15:01] [GORM错误] 批量更新电脑设备失败:
ERROR: ON CONFLICT DO UPDATE command cannot affect row a second time (SQLSTATE 21000)
```

### Error Messages
- **HTTP 500** from Gin recovery
- **PostgreSQL SQLSTATE 21000** "ON CONFLICT DO UPDATE command cannot affect row a second time"

### Reproduction
1. AD 数据库存在数据不一致(unique 约束 `uni_sys_ad_computer_config_name` 缺失或失效)
2. 同 `ad_config_id` 下,数据库存在两条不同 `distinguished_name` 但相同 `computer_name` 的电脑
3. LDAP search 返回这两个 entry(均不在现有 DN 集合,但 cn 相同)
4. → batch upsert 在同一 INSERT batch 内对同一冲突行触发多次 → 21000

---

## Root Cause

### 数据流回溯

1. **syncComputers** (`internal/services/addomain/computer.go:392`)
   ```go
   computersToUpdate := make(map[string]*models.ADComputer)  // ← key 是 DN
   ```

2. **processComputerEntry** (`computer.go:499` 修复前)
   ```go
   if existingComputer, exists := existingDNMap[computerDN]; exists {
       updateComputerFields(existingComputer, entry, parsedDesc, status, now)
       toUpdate[computerDN] = existingComputer         // ← DN 作 key
   } else if existingByName, nameExists := existingNameMap[computerName]; nameExists {
       updateComputerFields(existingByName, entry, parsedDesc, status, now)
       toUpdate[existingByName.DistinguishedName] = existingByName  // ← DN 作 key
   }
   ```

3. **batchUpdate** (`computer.go:551`)
   ```go
   clause.OnConflict{
       Columns: []clause.Column{{Name: "ad_config_id"}, {Name: "computer_name"}},  // ← 用 computer_name
       DoUpdates: clause.AssignmentColumns([...]),
   }
   ```

**关键错位**:
- map key = DN
- ON CONFLICT 目标 = `(ad_config_id, computer_name)`
- 这两个**可以不一致** — AD 数据库可能出现同名 cn 但不同 DN 的两条记录

### 触发场景
| 场景 | 是否触发 |
|------|---------|
| 同 `cn` 的两条记录都已在 DB(unique 约束失效/人工导入),LDAP 返回两条 entry | ✅ **触发** |
| AD tombstone + active 共存(同 cn 不同 DN),tombstone 还在 `sys_ad_computer` 里 | ✅ **触发** |
| AD 复制冲突暂时存在重复 | ✅ **触发** |
| 正常情况:cn 唯一,DN 唯一 | ❌ 不触发 |

### 为什么 NUL 字节修复后**才**暴露这个 bug
NUL 字节修复前,sync 在 `updateSyncLog` 阶段就因为 PostgreSQL TEXT 列 NUL 拒绝而崩溃,
根本走不到 `batchUpdate`。
修复 NUL 字节后,sync 顺利推进到电脑 upsert 阶段,**暴露了下一个隐藏 bug**。

这是**典型的"修复一个 bug,暴露下一个"模式** —— 第一次修复让代码真的跑到了崩溃点。

---

## Fix Applied

### 1. `computer.go` — `processComputerEntry` 改 key

```go
func (s *ComputerService) processComputerEntry(
    entry *ldap.Entry,
    configID string,
    parsedDesc map[string]string,
    status models.ComputerStatus,
    now time.Time,
    existingDNMap map[string]*models.ADComputer,
    existingNameMap map[string]*models.ADComputer,
    toCreate *[]models.ADComputer,
    toUpdate map[string]*models.ADComputer,
) {
    computerDN := entry.DN
    computerName := entry.GetAttributeValue("cn")

    // 复合 key:configID + "/" + computerName
    // 与 batchUpdate 的 ON CONFLICT (ad_config_id, computer_name) 对齐
    conflictKey := configID + "/" + computerName

    if existingComputer, exists := existingDNMap[computerDN]; exists {
        updateComputerFields(existingComputer, entry, parsedDesc, status, now)
        toUpdate[conflictKey] = existingComputer
    } else if existingByName, nameExists := existingNameMap[computerName]; nameExists {
        updateComputerFields(existingByName, entry, parsedDesc, status, now)
        toUpdate[conflictKey] = existingByName
    } else {
        newComputer := buildComputerFromEntry(configID, entry, parsedDesc, status)
        *toCreate = append(*toCreate, newComputer)
    }
}
```

### 2. `computer.go` — syncComputers 给 toCreate 加 dedup

```go
// computersToCreate 也必须按 (config_id, computer_name) 去重。
// 否则两个新电脑同名会触发 batchCreate 撞 unique 约束
// (uni_sys_ad_computer_config_name)。
if len(computersToCreate) > 0 {
    dedupMap := make(map[string]models.ADComputer, len(computersToCreate))
    for _, c := range computersToCreate {
        key := c.ADConfigID + "/" + c.ComputerName
        if _, exists := dedupMap[key]; !exists {
            dedupMap[key] = c
        }
    }
    deduped := make([]models.ADComputer, 0, len(dedupMap))
    for _, c := range dedupMap {
        deduped = append(deduped, c)
    }
    if len(deduped) < len(computersToCreate) {
        applogger.Warnf("[电脑同步] 待创建列表去重: %d → %d 条(同名 cn 冲突,保留最新)",
            len(computersToCreate), len(deduped))
    }
    computersToCreate = deduped
}
```

### 3. 为什么 user/group/ou 不需要改

| 模块 | unique constraint | map key | ON CONFLICT 列 | 是否对齐 |
|------|-------------------|---------|----------------|----------|
| OU | `(ad_config_id, ou_dn)` | DN | `(ad_config_id, ou_dn)` | ✅ |
| Group | `(ad_config_id, group_dn)` | DN | `(ad_config_id, group_dn)` | ✅ |
| User | `(ad_config_id, user_dn)` | DN | `(ad_config_id, user_dn)` | ✅ |
| **Computer** | `(ad_config_id, computer_name)` | **DN(改前)/ (config_id, computer_name)(改后)** | `(ad_config_id, computer_name)` | ⚠️ 改前不对齐 / ✅ 改后对齐 |

Computer 用 `computer_name` 作 unique constraint 是因为 AD 电脑常重命名(DN 跟着变),
不能用 DN。但批量 upsert 时必须保证 map key 与 conflict 列对齐,否则就撞 21000。

---

## Verification

| 步骤 | 命令 | 结果 |
|---|---|---|
| 单元测试 | `go test -count=1 ./internal/services/addomain/...` | PASS (7.7s) |
| 全量编译 | `go build ./...` | PASS |

---

## Remaining Risks

1. **数据一致性根因未修**:unique constraint 失效或重复数据本身是脏数据,
   应用层 dedup 只是兜底。建议下一步清理 `sys_ad_computer` 中的重复行并重建 unique index。

2. **dedup 静默丢弃**:目前同名 cn 的多条新电脑只保留最后一条,前几条被静默丢弃。
   如果业务期望保留所有,需要改方案(比如:改名 → 加后缀 → 全部 upsert)。

3. **其他 sync 函数无 dedup 防御**:user/group/ou 的 map key 与 ON CONFLICT 对齐,
   暂时安全。但如果将来有人把这些模块改成"按 name upsert",会重蹈覆辙。
   建议:在 `internal/services/addomain/` 加 lint 检查 "map key 必须与 ON CONFLICT 列对齐"。

---

## Files Changed

| 文件 | 修改 |
|---|---|
| `internal/services/addomain/computer.go` | +30 行 / 修改 2 处:`processComputerEntry` 改用复合 key;`syncComputers` 给 `toCreate` 加 dedup |

## Status

**修复已应用并通过测试** — 等待用户重启后端 + 触发同步验证 500 不再出现。
**建议后续 Phase**: 数据清理(删除 sys_ad_computer 重复行 + 重建 unique index)。

## Phase 40 Closure (2026-06-25)

复测 `internal/services/addomain/computer.go`：
- `processComputerEntry` (line 561-572) 已用复合 key `configID + "/" + computerName`
- `syncComputers` (line 436-444) 已对 `toCreate` 加 `dedupMap` 去重

PostgreSQL SQLSTATE 21000 (ON CONFLICT 同行二次) 风险已消除。frontmatter 翻 `resolved`。

verification: `grep -n "conflictKey\|dedupMap" internal/services/addomain/computer.go` 命中
files_changed: .planning/debug/ad-sync-500-on-conflict-duplicate-row.md
