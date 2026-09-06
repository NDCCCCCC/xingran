# UAT-RUNBOOK — 62-HUMAN-UAT 3 场景验证手册（v1.30 / UAT62-01..03）

**生成:** 2026-09-07（Phase 101 / plan 101-02 Task 1）
**服务对象:** `.planning/workstreams/milestone/phases/62-ai-internal-core-db/62-HUMAN-UAT.md` 的 3 个 pending 场景
**逐字断言锚点:** `internal/core/db/migrations/migration_176_reconciliation_physical_mv.go`（:200/:295/:310/:340/:168/:81）、`internal/core/db/database.go`（:810-814/:950-960）、`internal/core/db/init_data.go`（:253/:271/:288/:294）、`internal/config/config.go`（:340 DB_NAME / :352 SERVER_PORT 显式绑定）
**日志采集建议:** 启动输出重定向到文件后 `grep -n` 断言逐字串，避免终端编码干扰（Windows 下建议 `go run ./cmd/main.go > uat_run1.log 2>&1`）。

---

## §0 环境探针结论（2026-09-07 实测）

| 探针项 | 结果 | 证据 |
|--------|------|------|
| docker | **UNAVAILABLE**（命令不存在） | `command -v docker` 空 |
| psql | **MISSING**（命令不存在） | `command -v psql` 空 |
| 本地 PG 服务 | 无 | 同上（无任何 PG 客户端/服务探针通过） |
| `configs/config.yaml` | `database.type: "sqlite"`（本地 dev SQLite，`data/xingran.db`） | config.yaml `database:` 块注释 2026-08-17 切换记录 |
| config.yaml PG 残留配置 | 指向 Supabase 远端 | 同上（「切回 Supabase 只需把 type 改回 postgres」） |
| Supabase pooler 风险 | `internal/core/db/database.go:296-298` 明载 pooler 上 GORM PrepareStmt 会导致 AutoMigrate(80+ DDL) 卡死；`:626-627` cleanupOldConstraints 同类卡死记录 | database.go 源码注释 |

**结论（D-101-2/D-101-3）:**
- **场景 1/2 需要 PostgreSQL** → 本机无自动化基建（无 docker/psql/本地 PG），且 Supabase 生产 pooler **禁止**用于 UAT（卡死风险 + 业务库红线）→ executor 不执行、不谎报 passed，保持 `[pending]`，人工按 §1/§2 执行。
- **场景 3 是双方言语义**（admin 种子 init_data.go 通用）→ sqlite 口径全自动，executor 直接执行（§3 记录证据链）。

**红线（T-101-04）:** 场景 1/2 一切 DDL/构造只允许作用在**一次性试验 PG 库**（docker run postgres:16 或内网临时实例）。**禁止业务库、禁止 Supabase 远端**。构造 SQL 仅触碰 `reconciliation_normalized` 这一个 MV 对象。

---

## §1 场景 1（UAT62-01）: Migrate176 R1/R2→R5 就地升级（schema 校验回退）

### [HUMAN EXECUTES]

人工需完成三件事（executor 无法自动化）：① 提供一次性试验 PG 连接串；② 执行下方构造 SQL 与启动命令；③ 采集启动日志回填 62-HUMAN-UAT.md。

### 【前置条件】

- 一次性试验 PG 实例（postgres:14+），连接串（host/port/user/password/dbname）在手。
- 后端源码树（本仓库）可编译：`go build ./...` 通过。
- 已设置环境变量 `DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME` 指向试验库（config.go `bindEnvVars` 显式绑定，:340 同款机制）。
- `configs/config.yaml` 的 `database.type` 临时改回 `"postgres"`（或用环境变量覆盖路径——type 本身不走 env 绑定，需改 config）。

### 【自动化资产 1: pre-R5 旧结构 MV 构造 SQL】

目的：在试验库造出「MV 存在但缺 4 个 R5 标记列」的旧 schema，触发 `information_schema` 校验失败 → 回退 DROP+CREATE 慢路径。
口径：从 migration_176 R5 SELECT（migration_176_reconciliation_physical_mv.go :237-300）**剔除 4 个 R5 标记列**（`asset_username` / `physical_user_id` / `last_resolved_at` / `mv_refreshed_at`）及其专属的 R5 `reconciliation_physical_chain` JOIN（该 JOIN 仅服务 physical_* 列），其余列集保持 R1/R2 形态。**MV 名必须仍是 `reconciliation_normalized`**（迁移的探测/重建都锚定此名，这是可复用资产的关键）。

```sql
-- ⚠️ 仅在一次性试验库执行（T-101-04 红线）
DROP MATERIALIZED VIEW IF EXISTS reconciliation_normalized CASCADE;

CREATE MATERIALIZED VIEW reconciliation_normalized AS
SELECT DISTINCT ON (a.id)
    a.id                                              AS asset_id,
    a.devicesn                                        AS asset_code,
    a.machine_ip                                      AS asset_ip,
    a.mac1                                            AS mac1,
    a.mac2                                            AS mac2,
    COALESCE(NULLIF(a.mac1,''), NULLIF(a.mac2,''))    AS mac_join,
    COALESCE(
        NULLIF(a.user_id,''),
        (SELECT user_id_by_name_and_dept FROM reconciliation_user_lookup WHERE asset_id = a.id LIMIT 1),
        (SELECT user_id_by_name          FROM reconciliation_user_lookup WHERE asset_id = a.id LIMIT 1)
    )                                                 AS asset_user_id,
    a.deleted_at                                      AS asset_deleted_at,
    -- ↓↓↓ 以下为刻意保留的 R1/R2 形态列（ad JOIN 依赖 asset_username 的同名子查询，保留 JOIN 仅去输出列）
    ad.id                                             AS ad_id,
    ad.username                                       AS ad_username,
    ad.is_enabled                                     AS ad_is_enabled,
    last_resolved.resolved_by                         AS last_resolved_by,
    last_resolved.conflict_type                       AS last_conflict_type
    -- 刻意缺失（4 个 R5 标记列）: asset_username / physical_user_id / last_resolved_at / mv_refreshed_at
FROM ops_asset a
LEFT JOIN sys_ad_user ad
       ON ad.username = COALESCE(
              (SELECT username FROM sys_user WHERE id::text = NULLIF(a.user_id,'') LIMIT 1),
              a.nowuser_name
          )
      AND ad.deleted_at IS NULL
      AND ad.is_enabled = TRUE
LEFT JOIN LATERAL (
    SELECT resolved_at, resolved_by, conflict_type
    FROM sys_data_reconciliation r
    WHERE r.asset_id = a.id
      AND r.resolved_at IS NOT NULL
      AND r.deleted_at IS NULL
    ORDER BY r.resolved_at DESC
    LIMIT 1
) last_resolved ON true
WHERE a.deleted_at IS NULL
ORDER BY a.id, ad.id NULLS LAST;
```

> 前提：试验库已先由一次正常启动建全基线表（ops_asset / sys_ad_user / sys_data_reconciliation / reconciliation_user_lookup / reconciliation_physical_chain 等）。若库为全空，可先不构造、先启动一次让迁移建全基线 → 再执行本构造 → 再启动做正式 UAT。

### 【自动化资产 2: 构造后列集校验（启动前自查）】

```sql
SELECT column_name FROM information_schema.columns
WHERE table_name = 'reconciliation_normalized' AND table_schema = current_schema()
ORDER BY column_name;
-- 期望：结果集中【不存在】asset_username / physical_user_id / last_resolved_at / mv_refreshed_at
--（与 migration_176 :140-150 的 r5MarkerCols 检查集逐字对应）
```

### 【启动步骤】

```bash
go run ./cmd/main.go > uat62_01_run1.log 2>&1
# 等待 HTTP listen 日志后 Ctrl+C / kill 停止
```

### 【日志 grep 断言（逐字）】

```bash
# ① 回退原因 WARN（migration_176 :200，缺列清单动态拼入 <missing>）：
grep -n "检测到旧版本 MV schema" uat62_01_run1.log
# 期望命中：[迁移 176] 检测到旧版本 MV schema(缺 R5 标记列: [...]),走 DROP+CREATE 升级路径

# ② 重建成功（:295）：
grep -n "R5 reconciliation_normalized 已重建(双源 declared + 真物理链路)" uat62_01_run1.log

# ③ 索引就位（:310）：
grep -n "R5 索引已就位" uat62_01_run1.log

# ④ 验证通过（:340，MV/declared/physical 行数 ≥0 均合法，语句必须出现）：
grep -n "R5 reconciliation_normalized 验证通过: MV=" uat62_01_run1.log
```

### 【DB 状态断言 SQL（升级后）】

```sql
SELECT column_name FROM information_schema.columns
WHERE table_name = 'reconciliation_normalized' AND table_schema = current_schema();
-- 期望：4 个 R5 标记列全部就位
SELECT COUNT(*) FROM reconciliation_normalized;  -- MV 可读
```

### 【反向对照步骤（快路径无回退）】

同库**不做任何改动**直接二次启动：

```bash
go run ./cmd/main.go > uat62_01_run2.log 2>&1
# 断言：走快路径，且无 ②③④ 慢路径日志
grep -n "reconciliation_normalized 已存在且 R5 schema 完整,走 REFRESH CONCURRENTLY 快路径" uat62_01_run2.log  # :168 期望命中
grep -c "检测到旧版本 MV schema" uat62_01_run2.log   # 期望 0
```

### ⚠️ sqlite 跳过警示

sqlite 模式下 migration_176 :81 打印 `[迁移] R5 MV 重写跳过(非 PostgreSQL 数据库)` 直接跳过——**场景 1 无法用 sqlite 口径替代**，这也是本场景必须人工提供 PG 的根因。

---

## §2 场景 2（UAT62-02）: Advisory lock 双实例并发迁移保护

### [HUMAN EXECUTES]

人工需完成三件事：① 提供一次性试验 PG（可与 §1 同实例、**新库名**）；② 持锁/启动双实例；③ 采集第二实例日志与 pg_locks 快照。

### 【前置条件】

- 一次性试验 PG + 已建全基线表（同 §1 前提）；`database.type: "postgres"`。
- 第二实例端口错开：`internal/config/config.go:352` 显式绑定 `{"server.port", "SERVER_PORT"}` → 设 `SERVER_PORT=9001`（或复制 config 改 `server.port`）。

### 【路径 A（主推，确定性变体）: psql 会话持锁 → 启动实例 B】

**步骤 1** — psql 会话 A 持锁（与 database.go `acquireMigrationAdvisoryLock` 同款单参 hashtext 键）：

```sql
SELECT pg_try_advisory_lock(hashtext('xingran-migrations'));
-- 期望返回 t；保持该会话不关（会话级锁）
```

**步骤 2** — 同一 PG 上启动实例 B（端口错开）：

```bash
SERVER_PORT=9001 go run ./cmd/main.go > uat62_02_instB.log 2>&1
```

**步骤 3** — B 日志逐字断言（database.go :811）：

```bash
grep -n "另一实例正在执行启动迁移" uat62_02_instB.log
# 期望命中：[advisory-lock] 另一实例正在执行启动迁移,本实例跳过 175/176/202-205 迁移块
```

**步骤 4** — 确认 B 跳过迁移块后**照常完成启动**（HTTP listen 日志出现、无 panic/fatal 退出）。

**步骤 5** — 会话 A 释放锁 + 残留检查：

```sql
SELECT pg_advisory_unlock(hashtext('xingran-migrations'));  -- 期望 t
SELECT locktype, granted, pid FROM pg_locks WHERE locktype='advisory';
-- 期望：两实例全部停止后无残留行（迁移结束后锁被 :950-960 releaseMigrationAdvisoryLock 正确释放）
```

### 【路径 B（备选，双实例变体）】

两实例相隔约 5 秒先后启动（同库、端口 9000/9001），观察**后启动者**日志出现 :811 WARN。说明：非确定性——若首实例迁移块（175/176/202-205）已执行完毕则后启动者正常获锁无 WARN，需多试或改用路径 A。

### 【DB 状态断言 SQL】

```sql
-- 迁移结束后（两实例均停）无残留 advisory lock：
SELECT locktype, granted, pid FROM pg_locks WHERE locktype='advisory';
-- 期望 0 行
```

---

## §3 场景 3（UAT62-03）: 空库首启 admin 种子凭据告警（双方言，sqlite 全自动）

> 本场景 executor 已于 2026-09-07 自动执行（证据链见 `101-02-SUMMARY.md`），以下为完整复跑步骤。

### 【前置条件】

- `configs/config.yaml` 保持 `database.type: "sqlite"`；**不设** `SYS_ADMIN_BOOTSTRAP_PASSWORD`。
- sqlite 库文件路径经 `XINGRAN_DATABASE_PATH` 指向**全新空库文件**。
  > **plan 前提勘误（2026-09-07 实测）:** 101-02-PLAN 原写「DB_NAME 即 sqlite 库文件路径」——实测不成立：`database.dbname ← DB_NAME` 绑定（config.go :340）仅供 PG `GetDSN()`（config.go :543）消费；sqlite 文件路径取自 `database.path`（config.go :72，database.go sqlite 分支 `cfg.Path`），且无显式 env 绑定。可用覆盖机制是 viper AutomaticEnv（config.go :299-301，`XINGRAN_<KEY>` 兜底，:254 注释即以此为例）：`XINGRAN_DATABASE_PATH=<绝对路径>`。两次真实启动已实证该机制生效且 `data/xingran.db` 零触碰（stat 前后一致）。
  > **禁止指向开发者现有 `data/xingran.db`**（T-101-03）。

### 【Run 1: 默认凭据告警】

```bash
export XINGRAN_DATABASE_PATH="${TMPDIR:-/tmp}/uat_empty_a.db"   # 全新空文件(绝对路径)
unset SYS_ADMIN_BOOTSTRAP_PASSWORD
go run ./cmd/main.go > uat62_03_run1.log 2>&1
# 等待日志「服务器启动在端口: N」后停止
grep -n "管理员账户已使用出厂默认密码 admin123" uat62_03_run1.log
# 期望命中（init_data.go :288）：[安全告警] 管理员账户已使用出厂默认密码 admin123
# 附带告警横幅（:290 提示设置 SYS_ADMIN_BOOTSTRAP_PASSWORD 环境变量）
```

### 【Run 2: env 覆盖（必须换第二个全新空库）】

> admin 已存在时种子幂等跳过，:294 不会出现——**必须用全新空库**。

```bash
export XINGRAN_DATABASE_PATH="${TMPDIR:-/tmp}/uat_empty_b.db"   # 第二个全新空文件(绝对路径)
export SYS_ADMIN_BOOTSTRAP_PASSWORD="<random-strong-password>"
go run ./cmd/main.go > uat62_03_run2.log 2>&1
grep -n "默认管理员密码已从 SYS_ADMIN_BOOTSTRAP_PASSWORD 环境变量读取" uat62_03_run2.log  # init_data.go :294 期望命中
grep -c "出厂默认密码 admin123" uat62_03_run2.log   # 期望 0
```

### 【DB 断言（salt 非默认）】

- sqlite 变体：`sqlite3 "$XINGRAN_DATABASE_PATH" "SELECT username, quote(salt) FROM sys_user WHERE username='admin';"` → salt 非字符串 `default`（init_data.go :271 Salt 置空串，真实盐嵌入 `$sm3$iterations$salt$hash` 哈希串；executor 实测两次启动均得 `admin|''`）。
- PG 变体（可选，若已具备试验 PG）：`SELECT username, salt FROM sys_user WHERE username='admin';` → `salt IS DISTINCT FROM 'default'`。

### 【unit 级前置（已绿）】

```bash
go test ./internal/core/db/ -run TestCreateDefaultUser -v
# TestCreateDefaultUser_EnvOverride / _FallbackDefault / _NoDefaultLiteralSalt / _Idempotent 四语义全过
```

### 【复跑后清理】

删除两个临时库文件（含 -wal/-shm 兄弟文件）与日志中的敏感段（若密码串入日志）；`unset XINGRAN_DATABASE_PATH SYS_ADMIN_BOOTSTRAP_PASSWORD` 确认 shell 无残留变量。

---

## 附: 场景↔需求↔回写对照

| 场景 | Requirement | 执行边界 | 回写位置 |
|------|-------------|----------|----------|
| §1 | UAT62-01 | [HUMAN EXECUTES]（需试验 PG） | 62-HUMAN-UAT.md 场景 1 result |
| §2 | UAT62-02 | [HUMAN EXECUTES]（需试验 PG） | 62-HUMAN-UAT.md 场景 2 result |
| §3 | UAT62-03 | executor 全自动（sqlite 口径） | 62-HUMAN-UAT.md 场景 3 result + 101-02-SUMMARY.md |
