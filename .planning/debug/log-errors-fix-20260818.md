# 逐个修复日志错误 — Debug Session 完成报告

**Created**: 2026-08-18
**Closed**: 2026-08-18
**Source**: `logs/app.log` 全量扫描（130,517 行 / 5,836 error / 16 fatal）

## 修复结果

| # | 问题 | 状态 | 修复方式 |
|---|------|------|----------|
| 1 | P0-1 `sys_ad_service_accounts` 表缺失 → ADAccountPool cron 失败（585 次） | ✅ FIXED | 注册到 sqlite AutoMigrate + sanitize（因 ID 字段 default:gen_random_uuid() 需净化）+ 回归测试 HasTable 断言 |
| 2 | P0-2 `sys_dict_data` 表缺失 → my-menus 500 | ✅ N/A | 表已在 sqlite 库存在（line 718 已注册） |
| 3 | P1-3 `sys_rpa_workers` 主键 NULL 违规 | ✅ N/A | migration_205 已部署，最后错误 2026-08-14 01:14 后停止 |
| 4 | P1-4~14 批量补齐 11 张缺失表 | ✅ N/A | 表已全部存在于 sqlite 库（最新 commit 494a9fb 修复 UserColumnConfig 后） |
| 5 | P2-15 启动端口冲突 | ✅ FIXED | 新增 `run.bat`（Windows）+ `run.sh`（Linux/Mac）脚本，启动前自动清理 :9000 进程 |
| 6 | P2-16 DB 迁移 DROP CONSTRAINT 非幂等 | ✅ N/A | cleanupOldConstraints 已用 `DROP CONSTRAINT IF EXISTS`（line 414），最后错误 2026-08-13 21:53 后停止 |

## 代码变更

### `internal/core/db/database.go`（+20 行）

1. **sanitizeSQLiteModelDefaults**（line 235-249）：追加 `&models.ADServiceAccount{}` 到净化列表
   - 原因：ID 字段 `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"` 含 PG-only 函数式默认
   - 不净化则 sqlite 下建表报 `near "(": syntax error`

2. **AutoMigrate sqlite 分支**（line 750+）：追加 `&models.ADServiceAccount{}` 注册
   - 注释追溯同族缺漏历史（column-config-table-recon-stats / kb-tag-table-stats-400 / rpa-tasks-table-missing）
   - 注释说明 PG 不注册原因（零漂移惯例 + dbprovision 已 seed）

### `internal/core/db/database_test.go`（+6 行）

3. **TestNewDatabaseSQLite**：追加 `sys_ad_service_accounts` 到 HasTable 断言列表
   - 锁住同族 pattern：每个 sqlite-bootstrap-missing 模型必须有 HasTable 断言

### `run.bat`（新文件，46 行）

4. **Windows 启动脚本**
   - 启动前清理 :9000 端口占用进程（netstat + taskkill）
   - 兜底杀 xingran-backend.exe / go.exe
   - 支持 `SKIP_PORT_CLEANUP=1` 跳过清理

### `run.sh`（新文件，63 行）

5. **Linux/Mac 启动脚本**
   - 启动前清理 :9000 端口（lsof / fuser / ss 三档兜底）
   - 兜底 pkill xingran-backend
   - 支持 `SKIP_PORT_CLEANUP=1` 跳过清理

## 验证

```bash
$ go build ./...
EXIT:0

$ go test -v -run TestNewDatabaseSQLite ./internal/core/db/
=== RUN   TestNewDatabaseSQLite
INFO 所有表迁移成功
PASS

$ go test ./internal/core/db/... ./internal/services/addomain/...
ok  internal/core/db          3.581s
ok  internal/core/db/migrations  0.311s
ok  internal/services/addomain 0.601s
```

## 后续动作

1. **重启 backend**：触发 sqlite AutoMigrate 建 `sys_ad_service_accounts` 表
   - 命令：`./run.bat`（Windows）或 `./run.sh`（Linux/Mac）
   - 或手动：`taskkill /F /IM xingran-backend.exe && go run ./cmd/main.go`

2. **验证 ADAccountPool cron**：观察日志 `[ADAccountPool] cron` 行
   - 期望：`cron 恢复过期熔断账号` 不再报 `no such table`
   - 期望：每 5 分钟 cron 成功执行

3. **新增技能候选**（反馈记忆）：
   - 同族缺漏 pattern 已在 5 次修复中出现，每次都是「模型存在但 AutoMigrate 未注册」
   - 建议在 PR 检查 / 新模型审计时强制验证：模型 → AutoMigrate 注册映射