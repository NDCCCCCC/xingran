---
plan: 62-02
phase: 62
requirement: HYGIENE-04
status: complete
completed: 2026-10-22
---

# 62-02 SUMMARY — HYGIENE-04 嵌套组成员 DN 过滤（零 DDL）

## What Was Built

**问题**：生产 sys_ad_group_member 2815 行中 26 行 user_dn 不在 sys_ad_user，其中 8 行实为嵌套组 DN（如 CN=CXHUBinternet-VIP、CN=CXHUB-财务部(FGS)）。sync.go syncGroupMembers 对 LDAP member 属性无任何区分，组套组关系混入用户成员关系表，GetMembers total 虚高。

**方案（零 DDL 过滤）**：
- 新建 internal/services/addomain/group_member_filter.go：
  - filterNestedGroupMemberDNs(memberDNs, knownGroupDNs) (kept, filtered) — 纯函数，strings.EqualFold 大小写不敏感比较，nil 集合=防御语义全保留
  - SyncService.loadKnownGroupDNSet(ctx, configID) — 每轮一次 Pluck sys_ad_group.group_dn（软删排除，小写归一）
- syncGroupMembers 签名追加第 5 参 groupDNSet map[string]struct{}；nil 时 Warnf 防御；过滤命中仅记数量不落 DN（Infof）
- 三个生产调用点全部显式加载集合：
  - sync.go syncGroups（建组完成后、成员循环前加载，上抛错误）
  - group_sync_service.go SyncSingleGroup（本组 Upsert 后加载，错误上抛）
  - group_sync_service.go syncGroupEntries（批量建组后加载，失败降级 Warnf+不过滤，沿该路径非致命语义）
- 既有 6 处直调用例（sync_78_05_test.go）补第 5 参 nil（语义=不过滤，与原行为一致）

**自愈**：每组 delete+insert 重放语义天然清除存量污染行，无需数据修复脚本。

## Verification

- go build ./... exit 0
- 专项：TestFilterNestedGroupMemberDNs_{Basic,CaseInsensitive,NilSetKeepsAll,EmptyMember}、TestSyncGroupMembers_{NestedGroupMemberNotPersisted,ReplayPurgesLegacyNestedRow,NilSetKeepsAllWarns}、TestLoadKnownGroupDNSet_ExcludesSoftDeleted 全绿
- 既有零漂移：TestSync78_SyncGroupMembers_FullDiff / TestSyncGroupMembers_CreateFailRollsBackDelete / TestSyncGroupMembers_ReplayIdempotent / TestSync78_SyncGroups_* 全绿
- 全量回归：go test ./internal/services/addomain/ ./internal/scheduler/ -count=1 → ok 8.4s / 10.2s

## Must-Haves 核对

- ✅ 嵌套组 DN 传入后不写入（NestedGroupMemberNotPersisted 用例锁死）
- ✅ 集合=每轮一次查询 + strings.EqualFold（CaseInsensitive 用例锁死）
- ✅ nil 集合不过滤；生产三路径全部显式加载
- ✅ 既有三条用例语义不变
- ✅ MemberCount 保持 LDAP 原值（未动）
- ✅ 零 DDL（无 migration 文件）

## 生产自愈验证指引（UAT 用）

部署后触发一次全量组同步（或等 3 小时周期轮），然后：

```sql
SELECT count(*) FROM sys_ad_group_member m
WHERE EXISTS (SELECT 1 FROM sys_ad_group g
  WHERE g.ad_config_id = m.ad_config_id
  AND lower(g.group_dn) = lower(m.user_dn));
```

预期从 8 归零（另 18 行已离开 sys_ad_user 的普通账号 DN 不在本需求范围，见 PLAN flagged_assumptions）。

## Files Modified

- internal/services/addomain/group_member_filter.go（新建）
- internal/services/addomain/group_member_filter_test.go（新建）
- internal/services/addomain/sync.go（签名 +5 参、过滤、syncGroups 加载集合）
- internal/services/addomain/group_sync_service.go（两个调用点）
- internal/services/addomain/sync_78_05_test.go（6 处补 nil 参）
