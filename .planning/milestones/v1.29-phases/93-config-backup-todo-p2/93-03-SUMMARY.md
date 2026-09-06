---
phase: 93
plan: 03
subsystem: services/restore-tasks
tags: [async-task, restore, state-machine, migration]
key-files:
  created:
    - internal/models/config_restore_task.go
    - internal/core/db/migrations/migration_211_create_config_restore_task.go
    - internal/services/config_restore_task_service.go
  modified:
    - internal/core/db/database.go
metrics:
  tests_added: 0 (handler/e2e 测试落在 93-04/93-06；本 plan 全量回归由 services 包 446s 套件守护)
  commits: 2
---

# Plan 93-03 Summary: ConfigRestoreTask 模型 + 异步恢复任务服务

## What Was Built

1. **models.ConfigRestoreTask**（sys_config_restore_task，D-15）：任务运行时状态与备份版本链职责分离；string 四态枚举 `RestoreTaskStatus*`（pending/running/success/failed，D-34，不支持取消）；BeforeCreate UUID hook（23502 防御惯例）
2. **Migrate211 双注册**（D-15）：MigrateModelList AutoMigrate 行（sqlite 路径）+ PG 分支 `CREATE TABLE IF NOT EXISTS` + device_id/status 双索引（互斥查询 D-08 与状态机流转查询）
3. **ConfigRestoreTaskService**（93-03-02，D-01/04/08/09/10/11/14/17/20/21/22/23/34）：
   - `StartRestore`：同设备校验（D-04）→ 同设备互斥（D-08，`status IN (pending,running)`，Enqueue 去重先例）→ 建 pending 任务 → **detached context** goroutine 执行（`WithTimeout(context.Background(), constants.RestoreConfigTimeout)`，P1：HTTP ctx 随响应取消）
   - `runRestore`：defer recover 防 panic 悬挂（T-93-08）→ nil-executor 安全降级 → 恢复前自动备份（D-10：CreateBackup 回读设备当前配置落库，失败即中止 D-14）→ GetBackupContent（93-01 解压双检查路径 D-23）→ RestoreConfig 下发（D-02/12）→ 回读 hash **仅警告不算失败**（D-11）→ 版本链恢复记录（D-09：BackupTypeManual + ChangeReason "恢复自版本 %d"，统计链路零改动）→ success + result JSON（D-07 载荷）
   - `GetRestoreTask` / `ListRestoreTasks`（NormalizePagination + base.ApplySort 白名单）/ `RecoverStaleRunningTasks`（启动收敛 A5：running → failed "服务重启，任务中断"，防互斥锁死）

## Key Discoveries

- **CreateBackup 的 DeviceName 语义**：CreateBackup 回读设备当前配置作为备份内容，`req.DeviceName` 直接用于记录与 sanitize 后的文件名——执行中修正了初稿把 `task.CreatedBy`（用户名）与硬编码占位串传入 DeviceName 的缺陷，统一以源备份的 `backup.DeviceName` 为准。
- **回读失败兜底**：版本链记录内容初稿在 GetConfig 失败时落空串+空 hash（污染版本链）；改为兜底用已下发的备份原文（hash 与备份一致），`hashMatched=false` 忠实表达"未验证"。
- StartRestore 互斥校验与任务创建之间存在理论竞窗（两请求同时通过检查）——与 Enqueue 去重先例同款接受（D-08 锁定方案即查询式互斥）。

## Commits

| Commit | Description |
|--------|-------------|
| 54ec868 | feat(93): add ConfigRestoreTask model with Migrate211 dual registration (D-15/D-34) |
| 1240105 | feat(93): implement ConfigRestoreTaskService async restore orchestration |

## Deviations

- 93-03-PLAN 的任务级测试合并到 93-04（handler 层 taskId 契约 + D-04/D-08 错误透传，直接驱动 StartRestore）与 93-06（e2e 断言链）——service 层无纯函数可单测（全部依赖 DB/executor），避免重复脚手架。
- 版本链 nextVersion 计算内联在 runRestore（与 CreateBackup 内部逻辑同款），未抽公共 helper——两处语义不同（恢复记录 vs 常规备份），强行抽象反而耦合。

## Self-Check: PASSED

- `go build ./...` + `go vet ./internal/services/` exit 0
- `gofmt -l` 干净
- `go test ./internal/services/ -count=1` → **ok 446.78s，exit 0**（全包零回归）
