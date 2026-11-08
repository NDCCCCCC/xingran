# v1.53 项目目录整理 — 归档索引

**归档日期**：2026-11-08
**触发**：用户指示「整理项目目录，临时脚本、中间生成物等等各种文件，能删就删不能删就归档」。
**原则**：纯中间产物直接删除；有史料价值、被历史记录引用或属一次性 UAT 取证的脚本移入本目录（不进业务目录），保留可追溯性。

---

## 1. 已直接删除（不保留副本）

| 类别 | 路径 | 依据 |
|---|---|---|
| 构建产物 | `ruoyi-backend.exe` / `ruoyi-backend-embedded-linux` / `xingran-backend.exe` / `xingran-backend-embedded-linux`（~256MB） | 已 ignored，可由 `go build` 重建 |
| 构建产物 | `out/xingran-backend`（60MB） | `out/` 目录与 `sha256.txt` 被 `.github/workflows/deploy.yml` 引用，故保留目录，仅删二进制 |
| 覆盖率中间物 | `coverage.out`（3.3MB）、`coverage_diffcov_run.log`（232KB） | 每次 CI 由 `check-coverage.sh` 重新生成；脚本本身保留 |
| 日志 | 全仓 36 个 `**/logs/app.log`（36.48MB）+ 39 个空 `logs/` 目录、`xingran-react-frontend/dev-uat.log`、`test-baseline-t2.log`、根 `logs/` | 运行时产物，已 ignored |
| 崩溃/临时 | `bash.exe.stackdump`、`.tmp-msg69.txt`、`login-page-after-fix.png`、`元数据.xlsx` | 一次性事故/临时文件，无引用 |
| 事故目录 | `%SystemDrive%/ProgramData/...`（Windows 路径拼接事故产生的缓存 db）、`t1-verify/`（空壳 + 失效符号链接）、`data/integration-v151.db`（ignored 测试库） | 无引用、非源码 |
| 一次性根目录文件（git 记 delete） | `.continue-here.md`（2026-06-30 会话 handoff）、`add_asset_columns.sql`、`check_asset_types.sql`、`check_vdi_menu_fix.sh`、`debug_asset_filter.sql`、`run.sh` | 已跟踪但无引用、内容过时；按用户裁定直接删除 |
| 一次性根目录文件（untracked） | `QUICK_FIX_PATCH.md`（AD 同步补丁说明）、`xingran_menus_dedup.sql`（140KB 菜单去重导入产物） | 前者仅被 `.planning/tmp` 的 docs 验证快照提及；后者仅被 `migration_207_menu_catalog_seed.go:29` 注释作为历史背景提及（该迁移已把权威目录固化进启动链路，文件已被取代） |

## 2. 明确保留（未动）

| 路径 | 原因 |
|---|---|
| `.coverage-fe-floors` / `.coverage-threshold` | 被 `.github/scripts/check-frontend-coverage.sh` 与 `check-coverage.sh` 读取（CI 门禁输入） |
| `run.bat` | 被 `scripts/vmp-api/README.md:66` 引用 |
| `人员详情.xlsx` | 被 `internal/services/operations/excel_config.go:104` 引用（导入模板/样例） |
| `out/sha256.txt` | `.github/workflows/deploy.yml` 引用 |
| `test/fixtures`、`uploads/` 下 2 张已跟踪 jpg、`xingran-cli/` | 项目源码 / 运行时资源 |
| `scripts/.archive-migrate-2026-06-15/`、其余 `scripts/**` | 历史迁移归档与在用脚本 |
| `.planning/**`（除 `tmp/`）、`.github/`、`configs/`、`AGENTS.md`、`CLAUDE.md`、`CHANGELOG.md`、`README.md`、`package-lock.json` | 规划/配置/文档主体 |

## 3. 本目录归档内容

### `scripts-uat/`（4 个）
v1.21 期 `uat61-t1` 一次性 UAT 采样脚本，`RETROSPECTIVE.md:374` 明确记为「一次性重放窗口验证，非告警通道」；FIX-11 已由 `internal/services/addomain/ad_sync_drift_alert_service.go` 产品化为常态化告警通道取代。v4 已被 v5 取代，一并归档保留演化链。

| 原路径 | 说明 |
|---|---|
| `scripts/uat61-t1-poller-v4.ps1` | v4 采样器 |
| `scripts/uat61-t1-v4-watcher.ps1` | v4 观察器 |
| `scripts/uat61-t1-poller-v5.ps1` | v5 采样器（3025 次采样史实载体） |
| `scripts/uat61-t1-v5-watcher.ps1` | v5 观察器 |

> 注：`scripts/uat61-t1-poller-v5.ps1` 原被 OPEN-ITEMS.md A7 引用；该临时清单已归档为 `v1.53-OPEN-ITEMS.md`（历史快照），本表为新的路径换算依据。

### `planning-tmp-verify/`（43 个）
`.planning/tmp/verify-*.json` 与 `docs-work-manifest.json`：2026-10-30 前后文档批量验证的中间产物（记录当时各文档的 claims 检查数量）。已跟踪但无现存运行时引用；含对已删除 `QUICK_FIX_PATCH.md` 的 2 处历史引用（`docs-work-manifest.json:21`、`verify-QUICK_FIX_PATCH.md.json:2`），属时点快照，保留史料价值。空的 `.planning/tmp/` 目录已删除。

---

## 4. 复核记录

- 删除前逐一核验：git 跟踪状态、被引用情况、绝对路径归属（防止路径误删）。
- 删除后：`go build ./...` / `go test` 关键包、`git status` 复核见当次提交说明。
- 本目录为 `.planning/milestones/` 下的历史归档区，不参与构建，不作为当前状态权威；当前状态以 `STATE.md` / `ROADMAP.md` / `MILESTONES.md`（GSD 自动生成体系）为准。
