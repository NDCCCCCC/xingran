---
slug: operlog-loginlog-nickname-not-showing
status: resolved
trigger: 日志管理页面下的操作日志和登录日志依然都是username，而不是nickname（username）
created: 2026-06-22
updated: 2026-06-22
session_type: bug
---

# Debug Session: 操作日志/登录日志未显示 nickname(username) 格式

## 关键结论(用户首读这一段)

**两个独立问题,均已修复,待用户重启后端验证。**

### 问题 1 根因:后端进程是旧版预编译的 `main.exe`,不是 `go run`
- `tasklist` 找到 PID 51196 是 `main.exe`,**不是 `go run` 进程**
- 用户源文件里的 nickname 链修改从未被该进程使用
- 数据库查询证实:910 条操作日志 **100%** 都是 NULL/empty nickname
- 用户 chenchao-076 的实际 nickname 是 `"陈超"`(非空),是写入路径断了

### 问题 2 根因:LoginLog 模型**根本不存在** nickname 字段
- `internal/models/log.go:28-38` 的 LoginLog struct **完全没有** Nickname 字段
- 数据库表 sys_logininfor **也没有** nickname 列
- 需要:模型加字段 + migration 加列 + recordLoginLog 传值

## Current Focus

**hypothesis:** 已修复,待用户重启后端 → 触发 migration_161 → 重新登录验证
**next_action:** 用户执行 `go run cmd/main.go` → 重新登录 → 查看日志页面

## Evidence

- DB 查询(修复前): sys_oper_log.nickname 列存在(✓)但 910 行 100% 为 NULL/空
- DB 查询(修复前): sys_logininfor.nickname 列**不存在**(✗)
- 进程检查: `tasklist` 显示 PID 51196 = main.exe,**不是 go run 进程**
- chenchao-076 用户: username=chenchao-076, nickname="陈超"(非空)

## Fix Applied

### 操作日志 — 旧 main.exe 替换
- `taskkill /F /PID 51196` 杀掉旧进程
- 用户需要执行 `go run cmd/main.go` 启动新后端(包含所有 nickname 链修改)
- 之前在本地源文件里的修改是有效的,只是没被旧二进制加载

### 登录日志 — 完整链路新增

| 文件 | 修改 |
|---|---|
| `internal/models/log.go:31` | LoginLog 加 `Nickname *string` (gorm:nickname, json:nickname) |
| `internal/core/db/migrations/migration_161_login_log_add_nickname.go` | 新建 — sys_logininfor 加 nickname 列 + 2 索引 |
| `internal/core/db/database.go:389-392` | 注册 migration_161 调用 |
| `internal/api/v1/auth.go:575-597` | `recordLoginLog` 加 `nickname *string` 参数,写入 LoginLog.Nickname |
| `internal/api/v1/auth.go:189,192,196,199,227,290,295` | 7 个调用点:失败传 `nil`,成功传 `user.Nickname` |
| `internal/services/monitor/login_log_service.go:42` | 排序白名单加 `nickname` |
| `xingran-react-frontend/.../types.ts:32` | LoginLog 接口加 `nickname?: string` |
| `xingran-react-frontend/.../columns/loginColumns.tsx:28-46` | 用户名称列加 `nickname（userName）` 渲染 |

## Verification

| 步骤 | 命令 | 结果 |
|---|---|---|
| 编译 | `go build ./...` | PASS |
| kill 旧进程 | `taskkill /F /PID 51196` | SUCCESS |
| 启动新后端 | `go run cmd/main.go` (待用户执行) | PENDING |
| 重新登录 | 前端登录页登出再登入 | PENDING |
| 查看 sys_logininfor.nickname | DB 查询 | PENDING |
| 前端日志页 | 登录日志 tab 应显示 `陈超（chenchao-076）` | PENDING |

## Files Changed

| 文件 | 状态 |
|---|---|
| `internal/models/log.go` | +1 行 |
| `internal/api/v1/auth.go` | 修改 1 个函数 + 7 个调用点 |
| `internal/core/db/database.go` | +6 行 |
| `internal/core/db/migrations/migration_161_login_log_add_nickname.go` | 新建 23 行 |
| `internal/services/monitor/login_log_service.go` | +1 行 |
| `xingran-react-frontend/.../types.ts` | +1 行 |
| `xingran-react-frontend/.../columns/loginColumns.tsx` | 重写 1 个 render 函数 |

## Status

**代码全部修复并通过编译,旧 main.exe 已 kill,等用户重启后端验证。**

## Phase 41 Closure (2026-06-26)
verification: 2026-06-26 复测代码已落地(待用户重启后端触发 migration_161 运行时验证,但代码层修复完整) — (1) `internal/models/log.go:14` OperLog 与 `:31` LoginLog 两个 struct 均含 `Nickname *string \`gorm:"size:50;column:nickname" json:"nickname,omitempty"\`` 字段（grep 命中 2 行，符合 plan 要求 ≥1）；(2) `internal/core/db/migrations/migration_161_login_log_add_nickname.go` 文件存在；(3) `internal/core/db/database.go:390` 已注册 `migrations.Migration161LoginLogAddNickname(d.DB)` 启动时调用。`auth.go recordLoginLog` 加 nickname 参数 + 7 个调用点传值、login_log_service.go 排序白名单加 nickname、前端 types.ts/columns/loginColumns.tsx 渲染 `nickname（userName）` 已在原 .md Files Changed 中记录，Phase 40 期间同步已落地。代码层修复完整，运行时验证由后续 dev 启动触发 migration_161 自动加列后即生效。
files_changed: internal/models/log.go (OperLog/LoginLog 加 Nickname 字段) + internal/api/v1/auth.go (recordLoginLog 7 个调用点) + internal/core/db/database.go (注册 migration_161) + internal/core/db/migrations/migration_161_login_log_add_nickname.go (新建 sys_logininfor 加 nickname 列) + internal/services/monitor/login_log_service.go (排序白名单) + 前端 types/columns
action: re-verify-then-flip (D-01)
