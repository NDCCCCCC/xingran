---
quick_id: 261110-tbq
status: in_progress
created: 2026-11-10
updated: 2026-11-11 Wave 1 完成（契约/模型/迁移240/权限/配置/依赖/前端权限码/router.go+core.go 接线），Wave 2 四代理并行中
scope: 数据库全量备份导出/导入恢复（本地/OSS/FTP/SMB + cron 定期 + 系统管理独立菜单）
commit: false  # 按 CLAUDE.md 提交前确认纪律，提交前先给用户看 diff
decision_source: 用户评估页确认「系统管理下独立菜单」「复用 internal/scheduler」；本轮追加确认「完整生产级」「原生 pg_dump/VACUUM INTO」「SMB 用 OS 挂载」
---

# Quick 261110-tbq · 数据库全量备份与恢复

## 0. 锁定决策

| 项 | 结论 | 来源 |
|---|---|---|
| 菜单 | 系统管理 → 「数据备份与恢复」，`path='backup'`，`component='system/backup/index'` | 评估页 + agent-4 纠正（path 是 URL 段，不能写 `system/backup/index`） |
| 权限 | C 本体 `system:backup`；F `system:backup:list/:export/:restore` | 评估页 |
| 定期任务 | 复用 `internal/scheduler` 内置 cron，`sys_job` 只读可见，不新增 job 类型 | 评估页 |
| 实现深度 | 完整生产级 | 用户本轮确认 |
| 导出引擎 | PG `pg_dump/pg_restore`；SQLite `VACUUM INTO` 原子替换 | 用户本轮确认 |
| SMB | OS 挂载共享当本地路径，Go 端不实现 SMB 协议 | 用户本轮确认 |

## 1. 实证约束（4 份只读调研汇总）

- 全局前缀 `/api/v1`（`cmd/main.go:253`）；`exclude_paths` 必须写完整路径。
- 加密中间件：请求解密与响应加密**共用** `security.request_encryption.exclude_paths`（`router.go:141,147`）；`security.response_encryption.exclude_paths` 未被读取。`operlog.exclude_paths` 绝不能加（会丢审计）。
- `MigrateModelList()` 在 `internal/core/db/database.go:663-780`；`registry.go` 的 `ActiveMigrations` 只对账、不控制执行。执行由 `database.go` PG 分支 `:1223` / sqlite 分支 `:1369` 的 `runAndRecord` 决定。下一个迁移号 **240**。
- 菜单 seed：`migration_207` 有「系统管理根已存在就整体跳过」快速路径（`migration_207:54-63`）→ **存量库必须新建 `.go` 迁移**，seed.sql 只对全新库生效。两者都要改。
- 权限码三处单一真相源：前端 `constants/permissionCodes.ts`、后端 `pkg/permission/config.go`、migrations 的 `sys_menu.perms`；`scripts/check-permission-codes.mjs` 双向校验，C 本体码不进 PERMISSION_CODES 表。
- cron 是 **6 字段含秒**（`cron.New(cron.WithSeconds())`，`cron.go:186`）；时区 Asia/Shanghai。`config_backup` 这个 InvokeTarget 已被占用 → 用 `db_backup`。
- 内置任务要在定时任务页可见，必须写一行 `models.Job` + `scheduler.AddJob`（参考 `mac_history_tasks.go:62-137`）。
- 多实例部署**无跨实例互斥**（`ad_sync_tasks.go:195-196` 实锤）→ 备份必须自己做互斥（DB 行锁 / 备份任务表唯一活跃索引）。
- `sys_config` 键需 `ConfigTypeNo + IsSystemNo` 才可在参数管理页编辑；seed 幂等不覆盖管理员值（模板 `migration_216`）。热重载抄 Phase 58「固定最小 tick + 每 tick 读配置门控」（`ad_sync_tasks.go:193-220`）。
- HTTP `WriteTimeout/ReadTimeout = 75s`（`cmd/main.go:279-280`）→ **绝不同步流式导出**，必须异步任务表 + 轮询 + 独立短下载端点。
- `data/` 是运行时目录且被 gitignore（`.gitignore:68`），`data/backups/` 合规；与「Excel 模板禁止落盘」不冲突。
- `minio-go/v7` 必须钉 **v7.0.98**（≥v7.0.99 要求 Go 1.25，与 `go 1.24.0` 冲突）；`jlaffaye/ftp v0.2.4`；全部纯 Go 守住 `CGO_ENABLED=0`。
- `RecordWithBody` 有 8192 字节截断（`operlog.go:71,393-402`）→ 存储配置端点显式 `WithOperParam` 传已过滤小 JSON；multipart 上传用 `WithOperParam` 手工摘要。
- 恢复前自动全量备份 + 失败即中止，照抄 `config_restore_task_service.go:171-197`；detached goroutine 用 `context.Background()` + `defer recover` + 启动收敛残留任务（`:371-400`）。
- 全仓无系统级维护模式 → 本期实现轻量版（原子标志 + 中间件拒绝写请求），不重构 auth 链路。
- 表清单按**物理库枚举**（PG `information_schema`/`pg_catalog`；SQLite `sqlite_master`），不用 `MigrateModelList`（会漏视图/MV/归档表）。导入排除 `sys_schema_migrations`、`sys_config_restore_task`、视图/MV、幻影 join 表。

## 2. 模块边界（新增，不碰既有）

| 层 | 路径 | 说明 |
|---|---|---|
| Model | `internal/models/db_backup.go` | `DBBackup`（备份记录）、`DBBackupTask`（异步任务），表 `sys_db_backup` / `sys_db_backup_task` |
| Service | `internal/services/dbbackup/` | `contract.go`（接口+DTO，先落地）、`service.go`、`dumper.go`、`restorer.go`、`manifest.go`、`storage.go`、`config.go` |
| API | `internal/api/v1/dbbackup/` | `router.go`、`handler.go` |
| Scheduler | `internal/scheduler/db_backup_tasks.go` | `RegisterDBBackupTasks` |
| Migration | `internal/core/db/migrations/migration_240_db_backup.go` | 菜单 C+3F、权限常量声明、`sys_config` seed、两表 DDL（PG）、幂等 |
| Frontend | `src/pages/system/backup/` | `index.tsx`（Tabs）、`types.ts`、子组件、`src/lib/dbBackupApi.ts` |

**接线文件（集中改，避免并行冲突）**：`internal/api/router.go`、`internal/core/core.go`、`internal/core/db/database.go`、`internal/core/db/migrations/registry.go`、`internal/core/db/migrations/menu_catalog_seed.sql`、`pkg/permission/config.go`、`configs/config*.yaml`、`go.mod`/`go.sum`、`src/constants/permissionCodes.ts`。

**明确不改**：`internal/services/config_backup_service.go`（网络设备配置备份）、`sys_config_backup`、`sys_config_restore_task`、`network:backup:*`。

## 3. 任务拆分（3 波）

### Wave 1 · 地基与契约（lead 自己做）
1. 写 `261110-tbq-CONTRACT.md`：精确 API 路径、请求/响应字段、service 接口签名、DTO、存储配置结构、sys_config 键名。
2. `internal/models/db_backup.go`：两个 model，注册进 `database.go:668-780` 的 `MigrateModelList()`。
3. `internal/services/dbbackup/contract.go`：`DBBackupService` 接口 + 全部 DTO + 常量（供并行代理对齐）。
4. `migration_240_db_backup.go` + `database.go` 两分支 `runAndRecord(240)` + `registry.go` + `menu_catalog_seed.sql` 4 行。
5. `pkg/permission/config.go` 常量 + 路由映射；`configs/config*.yaml` 的 `exclude_paths` 加 `/api/v1/system/db-backup/*`。
6. `go.mod` 加 `minio-go/v7 v7.0.98`、`jlaffaye/ftp v0.2.4`（`go mod tidy`）。
7. `src/constants/permissionCodes.ts` 增 `systemBackup` 3 码 + `SYSTEM_BACKUP_PERM`。

### Wave 2 · 并行实现（4 代理，文件作用域互不重叠）
- **B 服务层**：`internal/services/dbbackup/{service,dumper,restorer,manifest,storage,config}.go`（**不碰 contract.go**）。含 `DumpExecutor` 接口抽象，单测可注入 fake，不依赖真实 pg_dump。
- **C API 层**：`internal/api/v1/dbbackup/{router,handler}.go`。
- **D 调度层**：`internal/scheduler/db_backup_tasks.go`（+ 在 `core.go` 的注册由 lead 接线）。
- **E 前端**：`src/pages/system/backup/**` + `src/lib/dbBackupApi.ts`（+ `src/types/backup.ts`、`types/index.ts` 导出）。

### Wave 3 · 集成与验证（lead）
1. `internal/api/router.go` 注册路由组（读组 `RequirePermissions` + 写兄弟组 `strictUserRouteGroup`）。
2. `internal/core/core.go` `initSchedulerAndTasks` 注册 `RegisterDBBackupTasks`。
3. `go build ./...` → 逐包修编译错误；`go test ./internal/services/dbbackup/... ./internal/api/v1/dbbackup/... ./internal/scheduler/... ./internal/core/db/migrations/...`。
4. 前端 `npm run type-check` + `npm run lint` + 权限码校验脚本 `check-permission-codes.mjs` + 相关 vitest。
5. code review（子代理复核 diff）→ 写 SUMMARY.md → STATE.md 登记 → **给用户看 diff，等确认后再提交**。

## 4. 验证标准

- `go build ./...` 通过；新增包单测通过（不依赖真实 pg_dump/OSS/FTP）。
- `npm run type-check` 通过；`check-permission-codes.mjs` 无漂移。
- 新库与存量库均出现菜单（seed + migration 240 双路径）。
- 导出/导入/下载/上传端点在 `exclude_paths` 内，operlog 记录 Export/Import/Upload/Download。
- 未在仓库提交任何 `data/` 或 `.xlsx` 文件；未触碰工作区既有未提交改动。

## 5. 风险与降级

- 部署机缺 pg_dump → 服务返回明确错误并在页面提示；不静默降级为自研 JSON。
- 无 superuser 时 `pg_restore --disable-triggers` 不可用 → 退化为 `--clean --if-exists --single-transaction` 按依赖顺序。
- 维护模式本期为轻量版（拒绝新写请求），不 drain 连接池。
- 远程存储凭据只存 `sys_config`（脱敏展示），不进 manifest、不进 operlog。
