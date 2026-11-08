# v1.53 脚本与工具整理 — 归档索引

**归档日期**：2026-11-08
**触发**：用户指示「cmd 下一次性运维工具移到 scripts、scripts 下大量过时脚本需要清理、统一构建产物路径、规范 test/tests 命名」。
**处置口径（用户裁定）**：有史料价值的一次性脚本归档保留、纯重复副本直接删除；4 组过时脚本全部纳入清理。

---

## 1. cmd/ 一次性工具迁移（13 个，git 记 rename）

生产入口 `cmd/main.go` / `cmd/main_test.go` / `cmd/agent/`（独立 Agent 服务）**保留在 cmd/**；其余 13 个一次性诊断/清理/验证 CLI 统一迁入 **`scripts/ops/`**，仍由 `go build ./...` 覆盖（`go list ./...` 已复核 13 个包全部纳入）。

| 原路径 | 新路径 | 用途 |
|---|---|---|
| `cmd/cleanup_port_status/` | `scripts/ops/cleanup_port_status/` | 清理异常端口状态行 |
| `cmd/diag_4f001_red/` | `scripts/ops/diag_4f001_red/` | 4F001 红点诊断 |
| `cmd/mac_cleanup/` | `scripts/ops/mac_cleanup/` | MAC 历史表清理 |
| `cmd/mac_list_backups/` | `scripts/ops/mac_list_backups/` | 列出 MAC 备份表 |
| `cmd/mac_merge_real/` | `scripts/ops/mac_merge_real/` | MAC 历史按状态转换点合并 |
| `cmd/mac_purge_drop_backup/` | `scripts/ops/mac_purge_drop_backup/` | DROP purge 备份表 |
| `cmd/mac_purge_meaningless/` | `scripts/ops/mac_purge_meaningless/` | 清理无意义 MAC 记录 |
| `cmd/mac_purge_verify/` | `scripts/ops/mac_purge_verify/` | 清理后抽样核对 |
| `cmd/reset-ad-breaker/` | `scripts/ops/reset-ad-breaker/` | 重置 AD 账号池熔断 |
| `cmd/verify-adbind/` | `scripts/ops/verify-adbind/` | AD bind 路径复现 |
| `cmd/verify-adcred/` | `scripts/ops/verify-adcred/` | AD 凭据自洽核对 |
| `cmd/verify-floorplan/` | `scripts/ops/verify-floorplan/` | 楼层平面图数据链路诊断 |
| `cmd/verify-format-unify/` | `scripts/ops/verify-format-unify/` | MAC 格式统一验证 |

**历史引用换算**：`.planning/debug/reconciliation-health-misjudgment.md:147` 的 `cmd/diag_4f001_red/`、`.planning/debug/mac-format-recurrence-R2.md:154` 的 `cmd/verify-format-unify/main.go` 均为收档时点快照，按本表换算；历史正文不回改。

## 2. 直接删除（纯重复，无副本保留）

| 路径 | 依据 |
|---|---|
| `test/`（含 `fixtures/lldp_output.go`、`mac_collection.go`） | 与 `tests/fixtures/` 逐字节相同（SHA256 一致），且全仓无 `test/fixtures` 的 import；`tests/fixtures` 被 3 个 Go 测试引用，保留 |
| `scripts/test_rpa_worker.py` | 与 `tests/scripts/rpa/test_rpa_worker.py` 逐字节相同 |

## 3. 本目录归档内容（37 个）

### `legacy-scripts/`（24 个）
历史 phase 验证 / 一次性修复 / VDI 诊断 / 部门同步 / 环境检查工具，均无现存运行时引用：

- 历史 phase：`verify_phase40.sh`、`verify_phase46_r5.sh`
- 文档债修复：`fix_debug_frontmatter.py`、`fix_ip_update.py`、`validate_debug_frontmatter.sh`
- VDI 诊断组：`get_vdi_data.bat/.py/.sh`、`parse_vdi_api.py`、`vdi_diagnostic_tool.bat/.sh`、`VDI_DIAGNOSTIC_GUIDE.md`、`README_VDI_DATA.md`、`vdi_groups.json`、`vdi_groups_raw.json`、`check_vdi_local_data.sql`、`vdi_test_standalone.go`
- 部门/SQL：`manual_dept_sync.sql`、`trigger_dept_sync.sh`、`dept_parent_chain_export.py`
- 其它：`check-env.go`（对应 `scripts/env/check/main.go` 有 `//go:build ignore` 的同功能副本）、`check-ci-local.sh`、`stop-backend.ps1`、`gen-sm2-keys.go`（同功能副本见 `scripts/crypto/gen_sm2_keys/main.go`）

### `tools/`（13 个）
`scripts/tools/` 下 v1.x 期一次性菜单检查/修复 CLI，全部无引用：`check_all_including_buttons.go`、`check_and_fix_apikey_menu.go`、`check_user_menu.go`、`cleanup_menus.go`、`execute_cleanup.go`、`find_duplicate_names.go`、`find_real_duplicates.go`、`fix_user_settings_menu.go`、`full_check.go`、`generate_cleanup.go`、`list_all_menus.go`、`search_menus.go`、`show_duplicate_details.go`。空的 `scripts/tools/` 目录已删除。

## 4. 保留在 `scripts/` 顶层（15 个，在用）

| 文件 | 依据 |
|---|---|
| `build-embedded.bat/.sh`、`build-linux.bat`、`build_linux.sh` | 本地构建脚本（产物路径已统一到 `out/`，见 §5） |
| `check-route-registry.mjs`、`scan-routes.cjs`、`route-registry.json` | 路由注册表门禁（`route_registry_test.go` / `strict_mount_count_test.go` 引用） |
| `check-status-literals.sh` | 状态字面量门禁（coverage 脚本 / `geocoding_service.go` 引用） |
| `check-bundle.sh` | `scripts/README.md:85` 引用 |
| `generate_swagger.sh` | Swagger 生成（docs 引用） |
| `migrate-sm4-key.go` | `docs/deployment.md:472,512` 引用 |
| `operlog_e2e_verify.sh` | 操作日志 e2e（MILESTONE-v1.52-SUMMARY 引用） |
| `rpa_test_templates.json` | `scripts/README.md:123` 引用 |
| `test-sm2-parse.go` | `tests/scripts/crypto/test_sm2_parse/main.go` 用法注释引用 |
| `README.md` | 脚本目录说明 |

`scripts/` 其余在用子目录（`build/`、`crypto/`、`db/`、`devtools/`、`e2e/`、`deploy/`、`tests/`、`env/`、`mac/`、`migrations/`、`sql/`、`vmp-api/`、`agent/`、`.archive-migrate-*` 等）未改动。

## 5. 构建产物路径统一

所有本地构建脚本与 CI 统一输出到 **`out/`**，禁止在仓库根目录落二进制：

| 脚本 | 原产物（根目录） | 新产物 |
|---|---|---|
| `scripts/build-linux.bat` | `xingran-backend-embedded-linux` / `-root` | `out/xingran-backend-embedded-linux` / `out/…-root` |
| `scripts/build_linux.sh` | `xingran-backend-linux` | `out/xingran-backend-linux` |
| `scripts/build-embedded.bat` | `xingran-backend-embedded.exe` | `out/xingran-backend-embedded.exe` |
| `scripts/build-embedded.sh` | 嵌入式 Linux 产物 | `out/` 下同名 |
| `scripts/build/build.bat` | `xingran-backend` | `out/xingran-backend` |
| `scripts/build/build_and_deploy.bat` | `xingran-backend-linux` | `out/xingran-backend-linux`（scp 路径同步） |
| `scripts/agent/build.sh` | `scripts/agent/build/`（相对 `BUILD_DIR`） | `out/agent/` |
| `.github/workflows/deploy.yml` | 已是 `out/xingran-backend` | 不变（作为权威参照） |

`.gitignore` 已有 `out/` 规则；新增约束见 `AGENTS.md` / `CLAUDE.md` 的「构建产物与目录约定」段。

## 6. test / tests 命名约定

统一 **`tests/`**（复数），删除 `test/`（单数）重复目录。规范已写入 `AGENTS.md` / `CLAUDE.md`。

## 7. 复核

- `go build ./...` / `go list ./...`：13 个 `scripts/ops/*` 包全部纳入，零错误
- 全量回归见当次提交说明；本目录为 `.planning/milestones/` 下的历史归档区，不参与构建，不作为当前状态权威。
