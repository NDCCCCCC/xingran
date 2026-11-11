# 261110-tbq 决策上下文（锁定，不再重议）

用户已通过评估页确认的两项决策：

1. **菜单位置**：系统管理（id `d67f4240-f887-481a-b345-94fb36782500`）下新建独立 C 型菜单
   「数据备份与恢复」，`path=system/backup/index`，`component=system/backup/index`，
   页面 `perms=system:backup`，新增 F 型子权限 `system:backup:list` / `system:backup:export` / `system:backup:restore`。
2. **定期任务接入**：复用 `internal/scheduler` 注册内置 cron 任务；周期与开关存 `sys_config`，
   执行状态写新增备份记录表；`sys_job` 页面只读可见，**不新增 job 类型**。

用户原始需求：

- 将数据库所有数据导出，并支持导入恢复。
- 支持自动定期备份。
- 默认备份到当前程序部署的服务器本地磁盘，可选择 OSS / FTP / SMB 等远程存储。
- 评估页面归属（已完成，见 `C:/Users/CPIC/.answer-me-with-html/pages/db-backup-placement-20261110-191306.html`）。

落地约束（来自 AGENTS.md / CLAUDE.md，锁定）：

- 构建产物落 `out/`；一次性运维 CLI 放 `scripts/ops/`；测试统一 `tests/`。
- 运行时数据放 `data/`（已被 gitignore），不纳入仓库；文档用 `<sqlite-db-path>` 占位。
- Linux 生产构建 `CGO_ENABLED=0`，远程存储客户端必须纯 Go。
- 所有业务写操作 handler 在 `response.Success` 前调用 `operlog.Record`；
  敏感端点用 `RecordWithBody` 脱敏。导出用 `OperTypeExport`，导入用 `OperTypeImport`。
- 请求体加密中间件已启用，导出下载 / 导入上传端点须加入
  `security.request_encryption.exclude_paths`（或按项目既有流式端点方式处理）。
- 现有 `internal/services/config_backup_service.go` 是网络设备 running-config 备份，
  语义不同，**不改名、不合并**；全库备份独立建模块。
- 提交前必须给用户看 diff 并等待明确确认；不得自行 commit。
- 工作区存在大量非本任务的未提交改动，本任务只新增/修改自身相关文件，不触碰、不暂存它们。
