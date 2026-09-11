---
phase: 101-closeout-uat-audit
plan: 02
subsystem: uat-closeout
tags: [v1.30-closeout, uat62, runbook, honest-writeback, sqlite-smoke]
requires:
  - "62-HUMAN-UAT.md 3 场景 expected 定义（status: resolved 语义保持）"
  - "D-101-2 自动化前置资产化 / D-101-3 人工边界不谎报 passed"
provides:
  - "UAT-RUNBOOK.md 三场景可执行手册（pre-R5 构造 SQL / advisory lock 双路径 / 场景 3 复跑步骤）"
  - "UAT62-03 sqlite 口径全自动证据链（unit 4 测试 + 两次真实启动日志 + salt DB 断言）"
  - "UAT62-01/02 保持 pending + runbook 引用（诚实回写）"
affects:
  - "TRACEABILITY-FINAL.md UAT62-03 行同步刷新（101-01 deviation 3 预告的独立 docs commit）"
tech-stack:
  added: []
  patterns:
    - "XINGRAN_DATABASE_PATH（viper AutomaticEnv XINGRAN_<KEY> 兜底）做 throwaway sqlite 指针——plan 的 DB_NAME 前提实测不成立（勘误见 Deviations 1）"
key-files:
  created:
    - ".planning/phases/101-closeout-uat-audit/UAT-RUNBOOK.md"
    - ".planning/phases/101-closeout-uat-audit/101-02-SUMMARY.md"
  modified:
    - ".planning/workstreams/milestone/phases/62-ai-internal-core-db/62-HUMAN-UAT.md"
decisions:
  - "UAT62-03 改判 [passed 2026-09-07]（sqlite 空库口径）：证据链三步全绿，PG 变体步骤留 RUNBOOK §3 可选"
  - "UAT62-01/02 保持 [pending]（D-101-3）：环境探针 docker/psql/本地 PG 全缺失，runbook §1/§2 资产就绪待人工"
metrics:
  duration: ~15min（含两次 10s 级启动 + 4 个 unit 测试 2.5s）
  completed: 2026-09-07
---

# Phase 101 Plan 02: UAT-RUNBOOK + 场景 3 全自动执行 + 62-HUMAN-UAT 诚实回写 Summary

三场景自动化前置资产化落盘（UAT-RUNBOOK.md，§1/§2 显式 [HUMAN EXECUTES]）+ UAT62-03 sqlite 空库口径全自动证据链三步全绿（unit 4 测试 / 默认凭据 WARN 启动 / env 覆盖启动，附 salt='' DB 断言）+ 62-HUMAN-UAT.md 诚实回写（1 passed + 2 pending 全部带证据或 runbook 引用，零谎报）。

## What Was Done

### Task 1: UAT-RUNBOOK.md 三场景验证手册（269+ 行）

- §0 环境探针结论（2026-09-07 executor 复测）: docker UNAVAILABLE / psql MISSING / 无本地 PG / `configs/config.yaml` `database.type: "sqlite"` / PG 残留配置指向 Supabase 远端且 `internal/core/db/database.go:296-298,:626-627` 明载 pooler AutoMigrate 卡死风险 → 场景 1/2 人工边界（D-101-3），场景 3 sqlite 全自动
- §1（UAT62-01）: pre-R5 旧结构 MV 构造 SQL 完整可执行（R5 SELECT 剔除 4 标记列 asset_username/physical_user_id/last_resolved_at/mv_refreshed_at + 其专属 pc JOIN，MV 名保持 `reconciliation_normalized`；「一次性试验库」红线 + 复用资产注记）+ 构造后 information_schema 自查 + 启动步骤 + 逐字 grep 断言（:200 回退 WARN / :295 已重建 / :310 索引就位 / :340 验证通过）+ 升级后列集断言 + 反向对照（二次启动 :168 快路径无回退）+ sqlite 跳过警示
- §2（UAT62-02）: 路径 A 主推（psql 会话 A `pg_try_advisory_lock(hashtext('xingran-migrations'))` 持锁 → 实例 B `SERVER_PORT=9001`（config.go :352 显式绑定实测）→ :811 逐字 WARN 断言 → `pg_advisory_unlock` 释放）+ 路径 B 备选（双实例相隔 5s）+ `pg_locks` 残留检查 SQL
- §3（UAT62-03）: 完整复跑步骤（含下方 Deviations 1 勘误后的 XINGRAN_DATABASE_PATH 机制）+ salt 断言 sqlite/PG 双变体
- HUMAN EXECUTES 标记 4 处（≥2）✓；三条关键逐字日志断言在文档 ✓；≥80 行 ✓

### Task 2: 场景 3（UAT62-03）sqlite 空库全自动执行 —— 证据链三步全绿

**第一步 unit 级（exit 0）:**

```
=== RUN   TestCreateDefaultUser_EnvOverride        --- PASS (0.69s)
=== RUN   TestCreateDefaultUser_FallbackDefault    --- PASS (0.65s)
=== RUN   TestCreateDefaultUser_NoDefaultLiteralSalt  --- PASS (0.66s)（env_path + fallback_path 子测试）
=== RUN   TestCreateDefaultUser_Idempotent         --- PASS (0.34s)
PASS    ok  github.com/xingran-next/xingran-go-backend/internal/core/db  2.473s
```

**第二步真实启动 run 1**（全新空库 `uat_empty_a.db`，未设 SYS_ADMIN_BOOTSTRAP_PASSWORD，SERVER_PORT=9410，日志 `服务器启动在端口: 9410` 确认 listen，~10s）:

```
WARN[2026-09-07 03:45:42] =========================================================
WARN[2026-09-07 03:45:42] [安全告警] 管理员账户已使用出厂默认密码 admin123
WARN[2026-09-07 03:45:42]   1) 请立即登录并修改管理员密码
WARN[2026-09-07 03:45:42]   2) 或重建实例前设置环境变量 SYS_ADMIN_BOOTSTRAP_PASSWORD=<强密码>
WARN[2026-09-07 03:45:42]   3) 完整方案(首登强制改密)已 deferred,需登录链路改动
WARN[2026-09-07 03:45:42] =========================================================
INFO[2026-09-07 03:45:42] 创建默认管理员用户成功
```

- 断言: :288 逐字 WARN 命中 ✓；`环境变量读取` 行数 = 0 ✓；DB 断言 `sqlite3 → SELECT username, quote(salt) ...` = **`admin|''`**（salt 空串 ≠ "default"，init_data.go :271 语义）✓

**第三步真实启动 run 2**（第二个全新空库 `uat_empty_b.db` + `SYS_ADMIN_BOOTSTRAP_PASSWORD=<random hex 强密码,值不入档>`，~10s listen）:

```
INFO[2026-09-07 03:46:16] 默认管理员密码已从 SYS_ADMIN_BOOTSTRAP_PASSWORD 环境变量读取
INFO[2026-09-07 03:46:16] 创建默认管理员用户成功
INFO[2026-09-07 03:46:22] 服务器启动在端口: 9410
```

- 断言: :294 逐字 Infof 命中 ✓；`出厂默认密码 admin123` 行数 = 0 ✓；DB 断言 salt = **`admin|''`** ✓

**安全与清理（T-101-03 履行）:**

- `data/xingran.db` stat（size+mtime）跑批前后逐字一致 —— 开发者现有库零触碰 ✓
- 两进程 kill 后 tasklist 确认无残留；临时库文件（含 -wal/-shm）+ smoke exe 已删；shell env（SYS_ADMIN_BOOTSTRAP_PASSWORD / DB_NAME / XINGRAN_DATABASE_PATH / SERVER_PORT）复查为空 ✓

### Task 3: 62-HUMAN-UAT.md 诚实回写

- 场景 1/2（UAT62-01/02）: result 保持 `[pending]`，各追加自动化前置行 → `UAT-RUNBOOK.md` §1/§2（探针结论引用）；**未标 passed**
- 场景 3（UAT62-03）: `result: [pending]` → **`[passed 2026-09-07]`**，附两条日志摘录引用 + TestCreateDefaultUser 4 测试结论 + 口径注记 `（sqlite 空库口径; PG 变体步骤见 RUNBOOK §3）`
- Summary 段数字重算: total 3 / passed 1 / pending 2（与三个 result 行严格一致）
- frontmatter `status: resolved` 保持不变（per plan）

## Deviations from Plan

**1. [Rule 3] plan 前提勘误: DB_NAME ≠ sqlite 库文件路径 → 改用 XINGRAN_DATABASE_PATH**
- **Found during:** Task 2 启动前安全核查
- **Issue:** plan 与 T-101-03 缓解措施假设「config.go :340 DB_NAME 绑定 → sqlite 模式下即库文件路径」。实测: `database.dbname ← DB_NAME` 仅被 PG `GetDSN()`（config.go :543）消费；sqlite 文件路径取自 `database.path`（config.go :72「sqlite 文件路径,仅 type=sqlite 时生效」；database.go sqlite 分支 `cfg.Path`，:186 默认 data/xingran.db），且 bindEnvVars 显式表无 `database.path` 条目。**按 plan 原样执行会静默打开开发者现有 `data/xingran.db`**
- **Fix:** 改用 viper AutomaticEnv 兜底机制 `XINGRAN_DATABASE_PATH=<绝对路径>`（config.go :299-301 SetEnvPrefix+Replacer+AutomaticEnv；:254 注释即以 XINGRAN_DATABASE_HOST 为同类示例），并实证: 两次启动均落在临时新文件 + `data/xingran.db` stat 前后一致；runbook §3 同步勘误
- **Files modified:** UAT-RUNBOOK.md（§3 前置/run1/run2/清理段）

**2. [流程适配] `go run` → 预编译 exe 直启 + PID kill**
- **Found during:** Task 2 执行设计
- **Issue:** Windows 下 `go run` 的子进程（真 exe）不随 go run 进程 kill 而退出，孤儿进程会占住 9410 端口与 sqlite 文件锁
- **Fix:** `go build -o $TEMP/uat_backend_10102.exe ./cmd/main.go`（3s）后直启 exe，bash `kill $PID` + tasklist 复核零残留；语义等价（同一 main），临时 exe 已删

**3. [端口适配] SERVER_PORT=9410（非 plan 示例 9001）**
- **Found during:** Task 2 执行
- **Issue:** 场景 3 与场景 2 示例端口无关；9410 任取避免与潜在在跑实例冲突，机制同为 config.go :352 `SERVER_PORT` 显式绑定
- **Fix:** env 注入；runbook §2 保留 9001 示例不变（场景 2 才需要双实例错开）

**4. [跨 plan 一致性] TRACEABILITY-FINAL.md UAT62-03 行刷新（独立 docs commit）**
- **Found during:** Task 3 完成后
- **Issue:** 101-01 落盘时 UAT62-03 暂记 pending（当时 101-02 未执行）；本 plan 完成后该行 stale
- **Fix:** 101-01-SUMMARY deviation 3 已预告——本 commit 之后单独 docs commit 刷新该行为 passed + 证据指针，保持与 62-HUMAN-UAT.md 严格一致

## Verification Results

- **T1**: runbook 269+ 行 ✓ / HUMAN EXECUTES ≥2 实测 4 ✓ / 三条逐字断言命中实测 8 ✓ / pg_locks 残留检查与 information_schema 四列检查 SQL 在档 ✓
- **T2**: `go test ./internal/core/db/ -run TestCreateDefaultUser -count=1` exit 0 ✓；SUMMARY 含 `安全告警`/`环境变量读取` 逐字摘录（grep 计数 ≥1）✓；run 1/2 日志摘录落盘 ✓
- **T3**: 62-HUMAN-UAT.md 场景 1/2 result 保持 [pending] 且带 runbook 引用；场景 3 [passed 2026-09-07] 带证据 + 口径注记；Summary total 3 / passed 1 / pending 2 与 result 行一致 ✓；commit 仅含 plan files_modified 的 3 文件 ✓

## Threat Model Compliance

- **T-101-03（DB_NAME 指错库）**: mitigated + 超越 plan 预期——发现 plan 前提缺陷（DB_NAME 不作用于 sqlite 路径）后改用 XINGRAN_DATABASE_PATH 并以 stat 前后一致实证开发者库零触碰；SKIP_AUTOMIGRATE 旁路未使用
- **T-101-04（runbook DDL 红线）**: mitigated——§1/§2 节首「一次性试验库」红线 + Supabase 生产库禁用明示；构造 SQL 仅触碰 reconciliation_normalized 单一 MV 对象
- **T-101-05（日志泄密）**: mitigated——摘录仅含日志语义行，零 DSN/password/token；run 2 密码值不入档（<random hex 强密码,值不入档> 口径）
- **T-101-SC（包安装）**: N/A — 零安装

## Known Stubs

无。本 plan 零生产代码变更（3 个 planning 文档 + 1 次只读 smoke 执行）。

## Self-Check: PASSED

- artifacts 存在: UAT-RUNBOOK.md / 101-02-SUMMARY.md（本文件）/ 62-HUMAN-UAT.md 回写
- commit: docs(101) 单 commit 含恰 3 文件（见 git log）
- 无临时文件/env 残留（git status 与 env 复查记录于 Task 2 安全段）
