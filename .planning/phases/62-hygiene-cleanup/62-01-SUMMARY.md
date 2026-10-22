---
plan: 62-01
phase: 62
requirement: HYGIENE-03
status: complete
completed: 2026-10-22
---

# 62-01 SUMMARY — HYGIENE-03 死代码删除

## What Was Built

F-22 旧路径死代码在 SAFE-01（Phase 60）落地后安全移除：

- **删除 internal/services/addomain/user_ou_service.go 整文件**（353 行，7 方法零生产调用：HandleUserLoginAD / updateUserDeptAndADInfo / restoreDeletedUserWithADInfo / GetUserDeptByADOU / createDeptFromOUDN / buildAncestors / generateUniqueDeptCode）
- **删除 internal/services/addomain/user_ou_service_test.go 整文件**
- **共享测试 const 迁移**：createDeptOUMappingTable / createUserTable / createDeptTable 逐字节迁入新建 test_fixtures_test.go（供 dept_ou_mapper_test.go / ad_push_queue_drain_test.go 等继续零副本复用）
- **删除 BatchMoveUsersToNewOU**（user_ad_sync_service.go，『为未来功能预留』零调用方）；moveSingleUserToOU / moveUserToNewOU 保留（drain 测试载体 + 活调用）
- **过期注释修订**：dept_ou_mapper.go:31（指向已删除的 user_ou_service.resolveDeptFromOU → 改指 system/user_sync_service.go 实况）、:73-74（Force 语境更新）、ldap_iface.go:14/:16-17/:37（调用方更名）、auth_handler_test.go 6 处注释去 UserOUService 字样

## Verification

- 删除前全仓 grep 预检：符号仅剩自身 + 注释提及
- go build ./... exit 0
- go test ./internal/services/addomain/ ./internal/api/v1/auth/ -count=1 → ok 8.1s / 0.4s
- 终检 grep：全仓零残留（仅 dept_ou_mapper.go:73 一处描述删除事实的历史注释，合法）

## Files Modified

- 删除 user_ou_service.go / user_ou_service_test.go
- 新建 internal/services/addomain/test_fixtures_test.go
- internal/services/addomain/user_ad_sync_service.go（删 BatchMoveUsersToNewOU）
- internal/services/addomain/dept_ou_mapper.go / ldap_iface.go（注释）
- internal/api/v1/auth/auth_handler_test.go（注释）
