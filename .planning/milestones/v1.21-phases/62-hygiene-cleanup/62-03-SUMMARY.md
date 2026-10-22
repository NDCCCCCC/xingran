---
plan: 62-03
phase: 62
requirement: HYGIENE-05
status: complete
completed: 2026-10-22
---

# 62-03 SUMMARY — HYGIENE-05 OU DN 冲突显式化

## What Was Built

**问题**：BuildOUDN 对同名兄弟部门必然产出同一 OU DN；UpsertMapping 异部门冲突时 Warnf + 静默删旧建新，第一个部门映射无声丢失且 SuccessDepts 照常自增。

**方案**：
- internal/services/addomain/dept_ou_mapper.go：
  - ErrOUDNConflict 哨兵 + OUDNConflictError{OUDN, ADConfigID, ExistingDeptID/Name, IncomingDeptID/Name}（Unwrap→哨兵，Error() 含双方部门名）
  - UpsertMapping 异部门分支改返回冲突错误，不删任何行（冲突时反查 sys_dept 取双方部门名）；同部门幂等更新与新建路径不变
  - UpsertMappingForce：共享内部 upsert(ctx, mapping, allowRebind)，保留原 Warnf+删旧+建新 P1 单事务语义，仅供手工重绑 API
- internal/services/addomain/dept_sync_service.go：syncDeptTree 冲突 errors.As 分流 → FailedDepts++ + Errors 追加 + Errorf 双部门名；其余错误保持 Warnf 非致命；树不中断
- internal/api/v1/system/ou_mapping_handler.go：UpdateOUDeptMapping 用 GetMappingByOU 预检 + 异部门时 Warnf 留痕 + UpsertMappingForce；响应结构/operlog 不变

## 语义变化表（UpsertMapping 严格化影响面）

| 调用方 | 旧行为 | 新行为 |
|--------|--------|--------|
| dept_sync_service.syncDeptTree | 静默删旧建新 + SuccessDepts++ | 冲突错误 → FailedDepts++ + Errorf |
| ou_mapping_handler.UpdateOUDeptMapping | 静默删旧建新 | 预检 Warnf + Force 删旧建新（显式） |
| user_ou_service.go:286 | （62-01 已删） | — |

生产当前 (ad_config_id, ou_dn) 重复=0，无存量数据迁移需求；零 DDL。

## Verification

- 翻转用例：TestUpsertMapping_DifferentOUTriggersDeleteAndInsert → 冲突错误 + incumbent 原样 + incoming 不落行（errors.As/errors.Is 断言）
- TestUpsertMapping_ForceRebindReplacesOld：Force 删旧建新
- TestUpsertMapping_ConflictCarriesDeptNames：双方部门名进错误消息
- TestUpsertMapping_SameDeptUpdateStillWorks：同部门幂等护栏
- TestSyncDeptTree_OUConflictCountedAsFailed：syncDeptTree 冲突计失败、树继续、incumbent 保留
- TestUpsertMapping_AtomicDeleteInsert 改走 Force 驱动（P1 atomic rollback 语义由 Force 承接，仍锁定）
- go build ./... + go vet ./...（仅存量 ad_authenticator_test.go:159 unreachable，非本次影响面）
- go test ./internal/services/addomain/ ./internal/api/v1/... ./internal/scheduler/ -count=1 全绿

## Files Modified

- internal/services/addomain/dept_ou_mapper.go
- internal/services/addomain/dept_sync_service.go
- internal/api/v1/system/ou_mapping_handler.go
- internal/services/addomain/dept_ou_mapper_atomic_test.go
- internal/services/addomain/dept_sync_conflict_test.go（新建）
