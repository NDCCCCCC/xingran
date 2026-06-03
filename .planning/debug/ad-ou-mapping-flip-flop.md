---
slug: ad-ou-mapping-flip-flop
status: resolved
trigger: 测试时出现警告：OU DN在不同部门之间反复重新映射。例如：OU=业务科,OU=浠水支公司...先映射到部门08f4f2e1-ddb3-4e12-9bae-1bcb64603b73，然后映射到4e55f785-6acf-4f7f-9bbb-09ad93dc7590，接着又映射回08f4f2e1-ddb3-4e12-9bae-1bcb64603b73
created: 2026-05-28
updated: 2026-05-28
---

## Symptoms

**Expected behavior:**
- OU DN应该稳定映射到一个部门
- 如果OU已映射到某部门，后续同步应该跳过或保持该映射

**Actual behavior:**
- 同一个OU DN在两个部门之间反复切换
- 警告信息显示：OU=业务科,OU=浠水支公司...映射到部门A，然后映射到部门B，然后又映射回部门A
- 这种flip-flop现象发生在多个OU上

**Error messages:**
```
WARN: OU DN OU=业务科,OU=浠水支公司,OU=黄冈中心支公司,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn 已映射到部门 08f4f2e1-ddb3-4e12-9bae-1bcb64603b73，现在重新映射到部门 4e55f785-6acf-4f7f-9bbb-09ad93dc7590
WARN: OU DN OU=业务科,OU=浠水支公司,OU=黄冈中心支公司,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn 已映射到部门 4e55f785-6acf-4f7f-9bbb-09ad93dc7590，现在重新映射到部门 08f4f2e1-ddb3-4e12-9bae-1bcb64603b73
```

**Timeline:**
- 发生在刚才的AD同步测试期间（2026-05-28 12:47:40-12:47:45）
- 刚修复了dept_sync_service.go跳过顶层部门的逻辑

**Reproduction:**
- 执行AD同步操作
- 观察日志中的WARN级别消息

**Additional context:**
- 受影响的OU包括：业务科、经理室、综合管理科等
- 受影响的支公司包括：浠水支公司、英山支公司、嘉鱼支公司等
- 看起来像是部门名称匹配逻辑有问题，可能找到了多个同名部门

## Current Focus

**hypothesis:** 已确认 - user_ou_service.go的createDeptFromOUDN方法仅按部门名称查找部门(不含parent_id约束)，导致同名部门匹配错误

**next_action:** fix applied

**test:** go test ./internal/services/addomain/... - all pass

**expecting:** stable OU-to-dept mappings

**reasoning_checkpoint:** fix applied

**tdd_checkpoint:** n/a

## Evidence

- `user_ou_service.go:131-133` (original) - `Where("dept_name = ? AND status = 0", deptName)` queries dept by name only, no parent scope
- `models/dept.go` - No unique index on `(dept_name, parent_id)`, only `DeptCode` has unique index
- `dept_ou_mapper.go:83` - The "remapping" warning is the flip-flop indicator
- `user_ou_service.go:162-178` (original) - Creates mapping with potentially wrong dept ID from the unscoped name query
- Multiple branches (浠水支公司, 英山支公司, 嘉鱼支公司) all have sub-departments with same names (业务科, 经理室, etc.)
- `createDeptFromOUDN` processes dept names from OU DN bottom-up, each level matched by name only
- When processing `OU=业务科,OU=浠水支公司,...`, the name query for "业务科" can match ANY "业务科" across ALL branches

## Eliminated

- dept_sync_service.go syncDeptTree - uses dept.ID directly from tree traversal, correctly scoped
- sync.go SyncService - only syncs AD data to ad_ou/ad_group/ad_user tables, does not create DeptOUMapping

## Resolution

**root_cause:** `user_ou_service.go` 的 `createDeptFromOUDN` 方法在查找部门时仅按 `dept_name` 匹配，没有限定 `parent_id` 范围。当多个不同支公司下存在同名部门（如"业务科"在浠水支公司和英山支公司下都有）时，查询会随机匹配到错误分支的部门，导致OU映射在两个部门之间反复切换。

**fix:** 修改 `createDeptFromOUDN` 方法，在查找已存在部门时增加 `parent_id` 约束，确保每一层部门都匹配到正确的父部门下的子部门。顶层部门使用 `parent_id IS NULL`，子部门使用 `parent_id = ?`。构建通过，所有 addomain 测试通过。

**verification:** `go build ./...` pass, `go test ./internal/services/addomain/...` all pass

**files_changed:** ["internal/services/addomain/user_ou_service.go"]
