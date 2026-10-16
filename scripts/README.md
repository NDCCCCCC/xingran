# Scripts 说明

本目录包含项目日常开发、调试、运维、构建所用的脚本与工具。

> **2026-08-12 整理记录**
> 顶层原本散落着 22 个未归类脚本，且已有若干子目录未被 README 收录。本次整理做了三件事：
> 1. **归档 17 个过时/一次性脚本** → `.archive-scripts-2026-08-12/`（不是删除，移入归档以保留可恢复性）
> 2. **归位 10 个常用脚本** → 按用途移入对应分类子目录（`build/`、`db/`、`sql/`、`tests/`），同时修正 `build/` 子目录下脚本的路径定位（`..` → `../..`）
> 3. **重写本 README** 反映完整结构

## 目录结构

```
scripts/
├── README.md                       # 本文档
│
├── build/                          # 构建脚本（bat/sh 配对）
├── agent/                          # Agent 部署/测试
├── vmp-api/                        # VMP API 逆向/分析工具集
│
├── e2e/                            # 端到端测试（Go test）
├── tests/                          # 测试脚本（Python / 一工具一目录）
├── verify/                         # 格式/校验工具
│
├── vdi/                            # VDI 对接工具
├── diag/                           # 诊断工具
├── port/                           # 端口采集工具
├── mac/                            # MAC 地址管理工具
│
├── crypto/                         # 国密（SM2/SM3/SM4）工具
├── db/                             # 数据库运维/导出工具
├── env/                            # 环境/配置检查工具
├── tools/                          # 菜单权限维护工具（一工具一目录）
│
├── sql/                            # 散落 SQL 修复/查询脚本
├── migrations/                     # 数据库结构迁移 SQL
├── migrate_cache_keys/             # 缓存键迁移（含 Makefile + 文档）
│
├── .archive-migrate-2026-06-15/    # 历史一次性 DB 迁移归档（2026-06-15）
└── .archive-scripts-2026-08-12/    # 本次整理的过时/一次性脚本归档（2026-08-12）
```

## 环境变量

所有直接连 DB 的脚本都通过环境变量获取数据库连接信息，运行前请设置：

```bash
# Windows
set DB_HOST=localhost
set DB_PORT=5432
set DB_USER=postgres
set DB_PASSWORD=your_password
set DB_NAME=xingran

# Linux/macOS
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=xingran
```

或使用 `.env` + direnv 等工具自动加载。

## 各子目录说明

### `build/` — 构建脚本

`bat`/`sh` 配对，覆盖 Windows / Linux（macOS 通常也能跑 `.sh`）。**所有脚本内已用 `%~dp0..\..` / `dirname/../..` 定位项目根，可从任意 cwd 调用**。

```bash
cd scripts/build

# 构建后端（Linux 目标）
./build.bat                    # 或 ./build_and_deploy.bat
./build-embedded.sh            # 内嵌前端的版本（Linux）
build-embedded.bat             # 内嵌前端的版本（Windows）
build-linux.bat                # 纯 Windows 构建

# Swagger 文档
./generate_swagger.bat
./generate_swagger.sh

# 校验打包后的 vendor chunks（D-07）
./check-bundle.sh
```

### `agent/` — Agent 部署/测试

跨平台 Agent 安装与冒烟测试。

| 文件 | 说明 |
|------|------|
| `install-windows.ps1` / `install-linux.sh` | Agent 安装 |
| `build.sh` | Agent 构建 |
| `test-agent.bat` / `test-agent.sh` | Agent 冒烟测试 |
| `test-config.yaml` | 测试用配置 |

详见 `agent/README.md`。

### `vmp-api/` — VMP API 逆向分析工具集

针对 VMP 平台登录接口的浏览器侧加密分析与重放工具（一次性调研产物）。

详见 `vmp-api/README.md`。

### `e2e/` — 端到端测试

Go 测试入口，与 Go test 框架配合运行。

```bash
go test ./scripts/e2e/...
```

### `tests/` — 测试脚本

跨语言测试脚本目录。

| 项 | 类型 | 说明 |
|----|------|------|
| `test_excel_import/main.go` | Go | Excel 导入功能测试 |
| `test_rpa_worker.py` | Python | RPA Worker 任务执行测试 |
| `rpa_test_templates.json` | 配置 | `test_rpa_worker.py` 配套模板 |

### `verify/` — 校验/格式化工具

| 项 | 说明 |
|----|------|
| `format_unify/main.go` | 格式统一校验 |

### `vdi/` — VDI 对接工具

| 项 | 说明 |
|----|------|
| `test_standalone/main.go` | VDI 独立测试工具 |

> 注：早期一次性 VDI 调研脚本（`get_vdi_data.*`、`vdi_diagnostic_tool.*` 等共 11 个）已归档至 `.archive-scripts-2026-08-12/vdi-research/`。

### `diag/` — 诊断工具

| 项 | 说明 |
|----|------|
| `red_4f001/main.go` | 4F001 异常诊断 |

### `port/` — 端口采集工具

| 项 | 说明 |
|----|------|
| `cleanup_status/main.go` | 端口采集状态清理 |

### `mac/` — MAC 地址管理工具

| 项 | 说明 |
|----|------|
| `cleanup/main.go` | MAC 清理 |
| `list_backups/main.go` | 备份列表 |
| `merge_real/main.go` | 真实数据合并 |
| `purge_drop_backup/main.go` | 丢弃备份清理 |
| `purge_meaningless/main.go` | 无意义数据清理 |
| `purge_verify/main.go` | 清理结果校验 |

### `crypto/` — 国密工具

| 项 | 说明 |
|----|------|
| `gen_sm2_keys/main.go` | 生成 SM2 密钥对 |
| `migrate_sm4_key/main.go` | SM4 密钥迁移 |
| `test_sm2_parse/main.go` | SM2 解析测试 |

### `db/` — 数据库运维/导出工具

| 文件 | 说明 |
|------|------|
| `snapshot.sh` | 数据库快照 |
| `audit_view_refs/main.go` | 视图引用审计 |
| `dept_parent_chain_export.py` | 部门父级链路导出（Excel） |
| `trigger_dept_sync.sh` | 手动触发部门到 AD 同步（填充 `sys_dept_ou_mapping`） |

### `env/` — 环境/配置检查

| 项 | 说明 |
|----|------|
| `check/main.go` | 环境前置检查 |

### `tools/` — 菜单权限维护工具

13 个 Go 工具，每个独立目录，统一用法 `go run <sub>/main.go`：

```bash
cd scripts/tools

# 设置数据库环境变量（见上文）
set DB_HOST=10.62.10.34
set DB_PORT=5432
set DB_USER=postgres
set DB_PASSWORD=<your-db-password>
set DB_NAME=xingran

# 列出所有菜单
go run list_all_menus/main.go

# 检查重复菜单
go run find_real_duplicates/main.go
```

可用工具：`check_all_including_buttons`、`check_and_fix_apikey_menu`、`check_user_menu`、`cleanup_menus`、`execute_cleanup`、`find_duplicate_names`、`find_real_duplicates`、`fix_user_settings_menu`、`full_check`、`generate_cleanup`、`list_all_menus`、`search_menus`、`show_duplicate_details`。

### `sql/` — SQL 修复/查询脚本

```bash
cd scripts/sql
psql -h localhost -U postgres -d xingran -f check_duplicate_menus.sql
```

包含：`check_duplicate_menus.sql`、`cleanup_duplicate_menus.sql`、`manual_dept_sync.sql`、`create_comprehensive_rpa_tests.py`。

### `migrations/` — 数据库结构迁移 SQL

按版本顺序执行的 SQL 迁移文件（与 `internal/core/db/migrations/` 不同 —— 这里是补充性独立迁移）。

| 文件 | 说明 |
|------|------|
| `add_source_work_order_id.sql` | 工单表加 `source_work_order_id` 列 |
| `add_system_menus.sql` / `add_system_menus_fixed.sql` | 系统菜单补充 |
| `captcha_background.sql` | 验证码背景配置 |
| `create_workstation_table.sql` | 工位表创建 |
| `drop_periodic_template_fk_assignee.sql` | 移除定期模板外键约束 |

### `migrate_cache_keys/` — 缓存键迁移

将旧格式 Redis 键迁移到新 `xingran:` 前缀的工具集（含 Makefile）。详见 `migrate_cache_keys/README.md` + `QUICKSTART.md`。

## 归档目录

### `.archive-migrate-2026-06-15/`

2026-06-15 由原 `migrate/` 重命名而来。归档一次性 DB 迁移脚本（`//go:build ignore`，正常编译不包含），所有迁移已在生产 DB 执行过。配套 SQL 见 `internal/core/db/migrations/archive/legacy-2026-06-15/`。

### `.archive-scripts-2026-08-12/`

本次整理（2026-08-12）新增的归档。**不是删除，是移入封存**——非 git 仓库无历史可回溯，移入归档以保留可恢复性（`mv` 回原位即可）。

分 3 组共 17 个文件：
- `phase-verification/` — 2 个历史阶段验收（Phase 40 / Phase 46 R5）
- `oneoff-fixes/` — 4 个一次性修复/验证脚本（含 `fix_ip_update.py` 源码 patch、Phase 40 frontmatter 修复、`operlog_e2e_verify.sh`）
- `vdi-research/` — 11 个 VDI 内网调研一次性产物（`get_vdi_data.*`、`vdi_diagnostic_tool.*`、`vdi_groups*.json` 等）

详见 `.archive-scripts-2026-08-12/README.md`。

## 注意事项

1. **生产环境使用前请备份数据库**
2. 确保环境变量配置正确
3. 建议先在测试环境验证脚本
4. **`.archive-*` 目录的脚本通常不需要再次运行**——除非你明确知道它还在用
5. `build/` 下脚本的路径定位（`%~dp0..\..` / `dirname/../..`）已适配子目录层级，可从任意 cwd 调用
6. **已知遗留 bug（未修）**：`build/generate_swagger.bat` 第 19 行 `cd /d "%~dp0.."` 在 `build/` 子目录里实际只切到 `scripts/`，应改为 `%~dp0..\..`（Linux 版 `generate_swagger.sh` 已在 2026-08-12 整理时一并修正）。如需在 Windows 上生成 Swagger，请优先使用 `cd scripts/build && generate_swagger.bat` 并手动确认 `cmd/main.go` 路径，或先 `cd ..` 再调用