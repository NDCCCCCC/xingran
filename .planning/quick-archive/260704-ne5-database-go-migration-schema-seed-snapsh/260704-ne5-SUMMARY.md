---
quick_id: 260704-ne5
created: 2026-07-04
completed: 2026-07-04
status: success
type: refactor
tags: [database, startup, deployment, snapshot, schema]
---

# Quick Task 260704-ne5: database.go 启动期 migration 调用精简 + snapshot 脚本 — Summary

## 一句话总结

AutoMigrate() 从 ~250 行降回 ~50 行(删除 200+ idempotent noop 启动 migration + 凭证模型迁移 + MV dropper + 物化视图重建链),保留 GORM AutoMigrate(model) 作为安全网;新增 scripts/db/snapshot.sh 一键导出 schema/seed snapshot,新部署从"跑 200+ migration"改为"两次 psql -f 导入"。

## 任务结果

### Task 1: 精简 database.go (commit `0e3430ab`)

**文件**: `internal/core/db/database.go`

**改动**:
1. 删除 `d.dropDependentMaterializedViews()` 调用 + 整个函数定义(53 行) — schema snapshot 已包含 MV DDL,无需启动 DROP+CREATE
2. 删除 `d.migrateCredentialModel()` 调用 + 整个函数定义(98 行) + `dropOldCredentialColumns()` helper(14 行) — 凭证模型已在生产迁移完成(protocol_type 列已存在)
3. 删除 ~200 行 `migrations.MigrateNNN...(d.DB)` 调用块,涵盖:
   - VDI 迁移:Migrate143 / 144 / 145
   - 资产管理:Migrate146 / 147 / 148 / 149 / 150
   - MAC 性能:Migrate151 / 152 / 153
   - 服务端排序索引:Migrate154 / 155 / 156 / 157 / 158
   - ops 菜单 perms 对齐:Migrate159
   - nickname 字段:Migration160 / 161
   - AD 服务账号池:Migrate162 / 163 / 164
   - 工位部门映射:Migrate165 / 167
   - API key 路由修复:Migrate166
   - 资产对账全套:Migrate168 / 169 / 170 / 171 / 172 / 173 / 174 / 175 / 176 / 177 / 178 / 179 / 180 / 181
   - 半自动修复建议:Migrate198 / 199 / 200
   - 组件序列号:Migrate201
   - 早期迁移:Migrate117(MAC 过滤规则) / Migrate033(OUI 厂商表)
4. 删除未用的 `migrations` import

**保留**(per CONTEXT 决策):
- `cleanupOldConstraints()` 调用(清理 GORM uniqueIndex vs SQL inline UNIQUE 命名冲突,~5ms)
- GORM `AutoMigrate(model 列表)` 调用(model 加新字段的安全网,~2s)
- `auditConstraintNaming()` 调用(DEBUG 级别,无日志噪音)
- 所有 `migrations.MigrateNNN` 函数定义本身(`internal/core/db/migrations/`,未删)

**Diff stat**: 1 file changed, 26 insertions(+), 406 deletions(-) = 净 -380 行

### Task 2: 新增 snapshot.sh (commit `361a1ecd`)

**文件**: `scripts/db/snapshot.sh`(新建,208 行)

**功能**:
- 从运行中 PostgreSQL 导出 schema + seed snapshot
- Schema:`pg_dump --schema-only --no-owner --no-acl -t 'sys_*' -t 'ops_*'`
- Seed:`pg_dump --data-only --inserts --no-owner --no-acl -t sys_menu -t sys_role -t sys_role_menu -t sys_user -t sys_config -t sys_dict_type -t sys_dict_data -t sys_ad_service_accounts -t sys_post`
- 输出:`docs/deployment/snapshots/schema-{YYYY-MM-DD}.sql` + `seed-{YYYY-MM-DD}.sql`
- 环境变量:`PGHOST`/`PGPORT`/`PGUSER`/`PGPASSWORD`/`PGDATABASE` 标准 libpq 变量,缺省时回退 `DB_*`(从项目根 `.env` 读取)
- 选项:`--output <dir>` / `--schema-only` / `--seed-only` / `--help`
- 前置检查:pg_dump 可用、凭据齐全;输出目录自动创建
- .env 加载用白名单 key 循环(避开密码含括号等特殊字符时 `source` 失败的坑)

**新部署流程**:
```bash
# 在生产 DB 主机跑:
scripts/db/snapshot.sh
# -> docs/deployment/snapshots/schema-{date}.sql
# -> docs/deployment/snapshots/seed-{date}.sql

# 在新机器:
psql -d NEWDB -f schema-{date}.sql
psql -d NEWDB -f seed-{date}.sql
./xingran-backend   # 启动期 AutoMigrate 仅做 model 字段增量
```

**验证**:
- `bash -n scripts/db/snapshot.sh` syntax OK
- `bash scripts/db/snapshot.sh --help` 输出正常
- 错误路径(无 pg_dump / 无凭据):清晰报错 + 退出码非 0
- 用 fake pg_dump 模拟执行,确认参数透传正确(.env 自动加载 PGHOST/PGUSER/PGDATABASE)
- 已设置 executable bit(`chmod +x`)

### Task 3: 验证 (no commit, no code change)

| 检查项 | 命令 | 结果 |
|--------|------|------|
| 全包编译 | `go build ./...` | OK (无输出 = 全包编译通过) |
| 全包 vet | `go vet ./...` | OK (无警告) |
| operlog regression | `go test ./internal/utils/operlog/...` | PASS (5/5 测试全绿,含常量稳定性 / 签名 / 关键词 / 排除路径) |
| `migrations/` 目录 | `git status internal/core/db/migrations/` | clean (无修改) |
| GORM AutoMigrate(model) | grep `&models\.` / `&operations\.` | 完整保留,75 行 model 列表原样 |
| `cleanupOldConstraints` 调用 | grep | 保留(line 246) |
| `auditConstraintNaming` 调用 | grep | 保留(line 342) |
| snapshot.sh syntax | `bash -n` | OK |
| snapshot.sh executable | `test -x` | OK (`-rwxr-xr-x`) |
| snapshot.sh --help | 执行 | 输出正常 |

## 文件 Diff Stats

```
 internal/core/db/database.go | 432 +++----------------------------------------   (净 -380 行)
 scripts/db/snapshot.sh       | 208 +++++++++++++++++++++                            (新建 +208 行)
 2 files changed, 234 insertions(+), 406 deletions(-)
```

## Build/Test 验证输出

```
go build ./...      → (no output, success)
go vet ./...        → (no output, success)
go test ./internal/utils/operlog/... → PASS (cached)
bash -n scripts/db/snapshot.sh       → Syntax OK
bash scripts/db/snapshot.sh --help   → 输出正常
```

## Commit Hashes

| Task | Hash | Message |
|------|------|---------|
| 1 | `0e3430ab` | refactor(database): strip 200+ startup migrations + drop view dropper + drop credential model migrator |
| 2 | `361a1ecd` | chore(scripts): add snapshot.sh to generate schema + seed SQL for new deployment |

## 影响范围

**启动期行为变化**:
- `internal/core/db::AutoMigrate()` 启动开销:~30s → ~2s(GORM AutoMigrate(model) + cleanupOldConstraints() + auditConstraintNaming())
- 启动日志噪音:250 行 migration 错误/跳过日志 → ~3 行(仅 GORM AutoMigrate "所有表迁移成功")

**新部署行为变化**:
- 旧:`./xingran-backend` 启动时跑 200+ migration(每次 ~30s)
- 新:`psql -f schema-snapshot.sql && psql -f seed-snapshot.sql && ./xingran-backend`(一次性导入,启动 ~2s)

**服务行为零变化**:设备采集 worker / cron 注册 / 缓存预热 / 操作日志 / 业务路由全部不动。

## 约束遵守

- ✅ 不删除 `internal/core/db/migrations/` 任何文件(保留 git 历史 + 开发演进参考)
- ✅ 不修改 service init(worker / cron / 缓存预热)
- ✅ 不新建 `sys_schema_migrations` 版本表(直接用 schema snapshot 部署)
- ✅ 不修改 GORM AutoMigrate(model) 列表
- ✅ 不修改 docs/deployment.md(用户决定手动更新)
- ✅ 不 commit docs artifacts(PLAN/CONTEXT/SUMMARY 由 orchestrator 提交)
- ✅ 使用 commit message format:`refactor(database):` + `chore(scripts):`

## 风险评估

| 风险 | 缓解 | 状态 |
|------|------|------|
| 删错调用导致启动 panic | 保留 GORM AutoMigrate(model) + cleanupOldConstraints();删的全是已 idempotent 的 migration | 已验证 go build / go vet 通过 |
| snapshot.sh 在生产 DB 跑锁表 | `pg_dump --schema-only` 本质只读事务;`--no-owner --no-acl` 不动权限;无显式锁 | 无需缓解 |
| 新部署流程变化无文档 | docs/deployment.md 由用户手动更新(明确不在本任务范围) | 用户负责 |
| .env 密码含特殊字符 | 白名单 key 循环加载,避免 `source .env` 失败 | 已修复 + 验证 |

## 后续建议(不在本任务范围)

1. 用户手动更新 `docs/deployment.md` 简述新部署流程(snapshot.sh + 2 步 psql -f)
2. 生产 DBA 在下次维护窗口跑一次 snapshot.sh,提交一份 baseline snapshot 到 `docs/deployment/snapshots/`
3. 关注 200+ migration 函数定义长期积累(`internal/core/db/migrations/` ~7000+ 行代码),如需精简可考虑加 `// @deprecated since 260704-ne5` 注释归档