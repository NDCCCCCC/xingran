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

### 构建脚本（顶层 + `build/` 子目录）

`bat`/`sh` 配对，覆盖 Windows / Linux（macOS 通常也能跑 `.sh`）。`scripts/build/` 下脚本用 `%~dp0..\..`、`scripts/` 顶层脚本用 `%~dp0..` / `dirname/..` 各自上溯到项目根，可从任意 cwd 调用（两者层级不同，不可一概而论）。**产物统一 `out/`**（见上文「构建产物路径」）。

```bash
# scripts/ 顶层（产物 out/）
./build-linux.bat [root]     # 内嵌前端 Linux（VITE_BASE 切换）
./build_linux.sh             # 纯后端 Linux
./build-embedded.bat         # 内嵌前端 Windows
./build-embedded.sh          # 内嵌前端 Linux

# scripts/build/ 子目录（产物 out/）
cd scripts/build
./build.bat                  # 纯后端 Linux
./build_and_deploy.bat       # 纯后端 Linux + 部署说明
./generate_swagger.bat       # Swagger 生成（Windows）
```

```bash
# scripts/ 顶层其余工具
./generate_swagger.sh        # Swagger 生成（Linux）
./check-bundle.sh            # 校验打包后的 vendor chunks（D-07）
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

### 端到端测试（已收敛到 `tests/e2e/`）

> **2026-11-09 quick 261109-wdu 收敛**：原 `scripts/e2e/` 与 `scripts/operlog_e2e_verify.sh`
> 两套重复副本已物理合并，唯一权威为 **`tests/e2e/operlog_e2e_verify.sh`**
> + `tests/e2e/operlog_e2e_verify_test.go`（AGENTS.md「后端 e2e 归 `tests/e2e/`」）。
> 本目录不再承载 e2e，`scripts/e2e/` 已移除。

```bash
go test ./tests/e2e/...
```

> 静态 3 项 operlog 覆盖断言不在 e2e 脚本内，见
> `internal/utils/operlog/handler_coverage_test.go`（由 `go test ./internal/...`
> 硬门禁执行）；`tests/e2e/operlog_e2e_verify_test.go` 只承载 LIVE e2e
> （需 running backend + DB + 凭据，缺凭据可观测 skip）。

### `tests/` — 测试脚本

跨语言测试脚本目录。注意：仓库根统一用 `tests/`（复数）存放 Go/前端测试，`test/`（单数）已删除；本目录只放 `scripts/` 自身的测试辅助脚本。

| 项 | 类型 | 说明 |
|----|------|------|
| `test_excel_import.go` | Go | Excel 导入功能测试（见 `docs/EXCEL_IMPORT_GUIDE.md`） |

> RPA Worker 测试脚本的权威副本在 `tests/scripts/rpa/`，不在本目录（重复副本已清理）。

### `ops/` — 一次性运维 / 诊断 CLI

从 `cmd/` 迁入的 13 个一次性 Go CLI（`package main`，仍被 `go build ./...` 覆盖）：`cleanup_port_status`、`diag_4f001_red`、`mac_*`（6 个）、`reset-ad-breaker`、`verify-*`（4 个）。用法：`go run ./scripts/ops/<name>/`。**新增一次性工具一律放这里，不要放 `cmd/`**（`cmd/` 只保留 `main.go` 与 `agent/`）。

### 顶层在用脚本（登记）

| 文件 | 用途 |
|------|------|
| `build-embedded.bat/.sh`、`build-linux.bat`、`build_linux.sh` | 本地构建（产物统一 `out/`） |
| `check-route-registry.mjs`、`scan-routes.cjs`、`route-registry.json` | 路由注册表门禁 |
| `check-frontend-api-urls.mjs`、`frontend-url-exemptions.json` | 前端 API URL ↔ 后端路由交叉校验（读 route-registry.json，消费方 scripts/frontend-url-exemptions.json 为豁免登记，reason 必填） |
| `check-status-literals.sh` | 状态字面量门禁 |
| `check-bundle.sh` | vendor chunks 校验（D-07） |
| `generate_swagger.sh` | Swagger 生成（Windows 版在 `build/`） |
| `migrate-sm4-key.go` | SM4 密钥迁移（见 `docs/deployment.md`） |
| `rpa_test_templates.json`、`test-sm2-parse.go` | 测试辅助 |

> **操作日志 LIVE e2e**：唯一权威在 `tests/e2e/operlog_e2e_verify.sh`
> + `tests/e2e/operlog_e2e_verify_test.go`（见上方「端到端测试」段）；
> 静态覆盖断言在 `internal/utils/operlog/handler_coverage_test.go`。

> 无现存引用的一次性历史脚本（历史 phase 验证 / VDI 诊断 / 菜单工具箱等）已归档到 `.planning/milestones/v1.53-archived-scripts/`，见该目录 README。
>
> **两份 `operlog_e2e_verify.sh` 重复副本处置 —— ✅ 已物理合并（2026-11-09 quick 261109-wdu）**：
> Phase 145 曾「保留双份并登记权威/从属，未来是否合并由 Lead 单独裁定」；本轮用户裁定按
> AGENTS.md「后端 e2e 归 `tests/e2e/`」收敛为**唯一权威** —— 删除 `scripts/operlog_e2e_verify.sh`
> 与 `scripts/e2e/operlog_e2e_verify_test.go`（`scripts/e2e/` 目录一并移除），保留
> `tests/e2e/operlog_e2e_verify.sh` + `tests/e2e/operlog_e2e_verify_test.go`（driver 仍指向同目录脚本，
> LIVE 行为与 cwd 锚定不变、缺凭据仍 `exit 1`/可观测 skip）。静态覆盖断言统一由
> `internal/utils/operlog/handler_coverage_test.go`（`go test ./internal/...` 硬门禁）承载，不在脚本内。
> 上文 `.archive-scripts-2026-08-12/` 提到的 `operlog_e2e_verify.sh` 是历史一次性归档件，与当前在用副本无关。
> 历史 phase 文档（145-PLAN/SUMMARY 等）为收档时点快照，保持只读、不回改。

### 构建产物路径（强制）

所有本地构建脚本与 CI 的产物统一输出到仓库根 **`out/`**（gitignored）：`out/xingran-backend*`、`out/agent/`。禁止在根目录落 `.exe` / `-linux` / `-embedded` / 打包文件。完整约束见 `AGENTS.md`「构建产物与目录约定」。

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
5. `build/` 下脚本与 `scripts/` 根下脚本层级不同，路径定位不可一概而论，详见第 6 条。
6. **Swagger 生成脚本（Phase 135 / FIX-02 已修复，替代此前自相矛盾的旧表述）**：
   - `scripts/build/generate_swagger.bat` 位于 `scripts/build/` 下，已把 `cd /d "%~dp0.."` 改为 `cd /d "%~dp0..\.."`（上溯两级到项目根），从 `scripts/build/` 执行可正常找到 `cmd/main.go`，此前的层级缺陷已修；
   - Linux 版 `scripts/generate_swagger.sh` 位于 `scripts/` 根（不存在 `scripts/build/generate_swagger.sh`），其 `dirname/..` 上溯一级到项目根，本就正确，与 .bat 层级不同、无同一 bug；
   - 两脚本的 `-o` 均已由 `docs` 改为 `internal/docs`，产物落 `internal/docs/`（与 `docs/archive/ARCHIVE-INDEX.md`「swaggo 产物位置调整」节一致）；`docs/` 下 5 份 Swagger 死文件已删除；
   - `cmd/main.go` 已补 `_ ".../internal/docs"` blank import，`internal/docs/docs.go` 的 `swag.Register` 生效，`/swagger/*any` 可加载到规范；
   - Windows 上从任意 cwd 直接运行 `generate_swagger.bat` 即可，无需再手动确认 `cmd/main.go` 路径。

## 版本注入构建与 Release 版本口径

### 后端：`-ldflags -X main.Version`

`cmd/main.go` 的 `var Version = "dev"` 保持可注入（缺省 `"dev"`，本地 `go run` / 未注入构建即此值）。发布构建用 `-ldflags -X` 把版本号写进二进制：

```bash
# 本地自检（产物落到系统临时目录，不落仓库；仓库内正式构建产物一律 out/）
go build -ldflags "-X main.Version=v1.56.0-rc" -o /tmp/xr-verify ./cmd/main.go

# Windows（MSYS bash，同样落仓库外）
go build -ldflags "-X main.Version=v1.56.0" -o /tmp/xr-backend.exe ./cmd/main.go

# 生产 Linux 构建（正式发布时产物统一落 out/，见上文「构建产物路径」）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags "-s -w -X main.Version=v1.56.0" -o out/xingran-backend-linux ./cmd/main.go

# 校验注入是否生效（启动后另开终端；也可看启动日志「版本: v1.56.0」）
curl -s http://localhost:9000/version    # => {"version":"v1.56.0"}
curl -s http://localhost:9000/health     # => 含 "version":"v1.56.0"
```

注意 `-X` 的目标必须是包级字符串变量 `main.Version`（`main` 包路径，不是模块路径）；`-ldflags` 含空格时整体加引号。

### 版本出口与权限说明

- `GET /health`：既有公开只读健康检查，返回 `status`、`version`、`time` 三个字段，无认证。
- `GET /version`：**本次新增的只读版本出口**，响应体**只有 `version` 一个字段**，不返回任何配置、密钥、环境变量或运行时信息。

两者都注册在 `cmd/main.go` 的 `setupRoutes` 中、挂在 Gin engine 根级（`/api/v1` 之外），与 `internal/api/router.go` 里带 `JWTAuth` 的 `monitor` 组、`rpaPublic` Worker 通道完全隔离：**不经过 JWT / RBAC 权限 / 请求响应加解密中间件**，即「公开、无认证、只读」。因为复用的是 `/health` 已有的根级公开只读暴露面，**未新增认证或权限面**，也未在 `internal/api/router.go` 增加任何路由。该出口与 `/health` 同权限，仅供部署流水线做发布版本校验；如需收紧，应在网关/反向代理层统一按路径限制，不要在业务代码里单独加认证（会破坏既有 health 探活链路）。

> 说明：经排查 `internal/api/router.go` 现有无认证组只有 `rpaPublic`（RPA Worker 业务通道），`monitor` 组挂了 `JWTAuth`，**没有可复用的无认证只读版本端点**；且该文件不在本次改动所有权内，故版本出口落在 `cmd/main.go` 与 `/health` 同级。

### 前端：`__APP_VERSION__` 编译期注入

`xingran-react-frontend/vite.config.ts` 用 `define` 把 `__APP_VERSION__` 替换为 JSON 字符串字面量，**不读取也不修改 `package.json`**（其 `version` 固定 `0.0.0`，不参与版本口径）。来源优先级：

```bash
# 1) CI / 命令行环境变量（process.env 优先，推荐）
APP_VERSION=v1.56.0 npm run build
# 或
VITE_APP_VERSION=v1.56.0 npm run build

# 2) .env.production / .env.<mode> 里写 VITE_APP_VERSION=v1.56.0
# 3) 都没有 → 回落 "dev"
```

`src/vite-env.d.ts` 声明了 `declare const __APP_VERSION__: string`；`src/components/AboutCard.tsx` 是「关于」展示组件，读取该常量显示版本号，可在个人中心或系统设置页面引入（见下文接入示例）。

### Release 版本口径（前后端同源）

一次 Release 只用一个版本号（如 `v1.56.0`，预发布用 `v1.56.0-rc` 这类 SemVer 字符串），**前后端必须由同一个 CI 变量同时注入**：

| 端 | 注入方式 | 载体 | 未注入时 |
|----|----------|------|----------|
| 后端 | `go build -ldflags "-X main.Version=$RELEASE_VERSION"` | 二进制内 `main.Version` | `dev` |
| 前端 | `APP_VERSION=$RELEASE_VERSION npm run build` | `__APP_VERSION__` 编译期常量 | `dev` |

口径要点：

1. **唯一来源**：CI 流水线定义 `RELEASE_VERSION`（取自 Git tag / 发布单），分别传给后端 `-ldflags` 与前端 `APP_VERSION`，禁止两处各写一份字面量。
2. **不改 `package.json`**：前端 `version` 字段保持 `0.0.0` 仅为 npm 占位，不是发布版本号，避免双源不一致。
3. **可校验**：发布后 `curl /version`（或 `/health`）得到的 `version` 应等于前端「关于」里显示的版本号，也等于 `RELEASE_VERSION`；三者不一致即视为发布未生效。
4. **本地开发**：未注入时前后端都显示 `dev`，属预期，不代表 Release。

### 「关于」组件接入示例（供页面维护者使用）

本次改动只在 `src/` 下**新建**组件与类型声明，未改动既有个人中心 / 系统设置页面（严格文件所有权）。页面维护者在 `src/pages/profile/index.tsx` 或 `src/pages/settings/` 中按需引入即可：

```tsx
import AboutCard from "@/components/AboutCard";

// 个人中心：放在「我的值班」Card 之后
<Row gutter={[16, 16]} className="mt-4">
  <Col span={24}>
    <AboutCard />
  </Col>
</Row>

// 或系统设置：作为新分类卡片的 content
// { key: "about", label: "关于", icon: <InfoCircleOutlined />, content: <AboutCard />, maxWidth: 760 }
```
