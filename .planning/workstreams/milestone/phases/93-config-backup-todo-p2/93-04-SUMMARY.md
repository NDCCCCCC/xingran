---
phase: 93
plan: 04
subsystem: api/network-backup
tags: [handler, async-task, routes, operlog]
key-files:
  modified:
    - internal/api/v1/network/backup_handler.go
    - internal/api/v1/network/network_router.go
    - internal/api/v1/network/backup_handler_test.go
    - internal/services/config_backup_service.go
    - internal/services/config_backup_service_79_06_test.go
metrics:
  tests_added: 8 (restore_returns_task_id / backup_not_found / cross_device_rejected / mutual_exclusion_rejected / binding_requires_deviceId + detail_by_id / detail_not_found / list_all / list_filter_by_deviceId；删除失真 stub 锁用例 ×2)
  commits: 2
---

# Plan 93-04 Summary: Restore handler 异步语义 + 任务查询端点 + 装配接线

## What Was Built

1. **BackupHandler.restoreTaskSvc 依赖**：struct 新字段 + NewBackupHandler 三参（backupService, restoreTaskSvc, db）
2. **Restore 异步语义**（D-01/D-07/D-17/D-18）：handler 调 `StartRestore(c.Request.Context(), id, deviceId, userID)` 立即返回 `gin.H{taskId, status, message}`；operlog.Record（OperTypeUpdate + module"配置备份"）保持 success path 末尾原位，发起时记录一次（任务结果只在任务表，不双记）
3. **任务查询双端点**：`GetRestoreTask`（POST /restore-tasks/:id）+ `ListRestoreTasks`（POST /restore-tasks/list，rawReq + getIntField/getOrderByColumn/getIsAscPtr 同 List 模式 + PageResponse）；查询端点不记 operlog
4. **network_router.go 装配**：restoreTaskSvc 与 backupService 同点构造（`core.DeviceExecutor` 注入）→ 非阻断 `RecoverStaleRunningTasks(context.Background())` 启动收敛 → /restore-tasks 双路由位于 backups 组 `Use(RequirePermissions)` 之后（D-31 继承组级 4 权限点，零新权限点）
5. **RestoreBackup stub 原子删除**（BACKUP-CLOSED-03 收口）：`config_backup_service.go` stub 方法 + "配置恢复功能待实现" + TODO 注释删除，`go build` 充当 checklist 证明零残留调用方
6. **失真锁测试清理**：`backup_handler_test.go` restore_is_a_documented_stub 删除；`config_backup_service_79_06_test.go` TestCbk7906_RestoreBackup 删除（锁的即是被实现的 stub）

## Test Notes

- newBackupTestEnv 增加 ConfigRestoreTask 迁移；newBackupHandler 装配真实 ConfigRestoreTaskService（nil executor：StartRestore 在校验/互斥/建任务层返回不触达下发；异步 goroutine 落入 runRestore 的 nil-executor 防护置 failed，不影响 HTTP 断言）
- restore_returns_task_id 断言 taskId 非空 + status=="pending"（handler 序列化的是 StartRestore 返回的内存对象，不受 goroutine 竞态影响）+ 任务行落库

## Commits

| Commit | Description |
|--------|-------------|
| 36fbfa7 | feat(93): switch restore endpoint to async taskId semantics and remove stub |
| f81964c | test(93): rewrite restore handler tests for async taskId contract |

## Deviations

- `config_backup_service.go` 的 gofmt import 排序问题（93-01 遗留，stash 验证非本次引入）顺手修复，归入 feat commit。
- 79_06 的 stub 锁测试删除归入 feat commit（编译器 checklist 效应的一部分，避免中间 commit 测试编译失败）。

## Self-Check: PASSED

- `go build ./...` + `go vet ./internal/services/ ./internal/api/v1/network/` exit 0
- `go test ./internal/api/v1/network/ -count=1` → ok 0.71s 全绿
- `go test ./internal/services/ -run TestCbk7906 -count=1` → ok 全绿
- `go test ./internal/utils/operlog/ -count=1` → ok（D-13 锁值防线零回归）
