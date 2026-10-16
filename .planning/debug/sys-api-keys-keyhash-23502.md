---
slug: sys-api-keys-keyhash-23502
name: sys-api-keys-keyhash-23502
status: resolved
trigger: keepalive 修复后启动继续失败: AutoMigrate ALTER TABLE sys_api_keys ADD key_hash varchar(64) NOT NULL 报 SQLSTATE 23502 (column contains null values)
created: 2026-10-15
updated: 2026-10-15 13:42
---

## 症状 (Symptoms)

**期望行为:**
- 应用启动完成 AutoMigrate 并继续启动流程

**实际行为:**
- AutoMigrate 阶段失败退出（exit status 1）——这是 pg-keepalive-param-42704 修复后暴露的启动链第二环

**错误信息:**
```
INFO[2026-10-15 13:16:12] [dropDependent] 当前架构下 noop(reconciliation_normalized MV 保留,普通 VIEW 由 175 CREATE OR REPLACE 重建)。
ERRO[2026-10-15 13:16:15] [GORM错误] ALTER TABLE "sys_api_keys" ADD "key_hash" varchar(64) NOT NULL | 错误: ERROR: column "key_hash" of relation "sys_api_keys" contains null values (SQLSTATE 23502)
exit status 1
```

**时间线:**
- 未提交的 Phase 60 / SEC-01 改动（git diff internal/models/api_key.go）：明文 `Key` 列 → `KeyHash/Salt/KeyPrefix` 三列（均 not null）

**重现步骤:**
1. `go run ./cmd/main.go`（keepalive 修复后）
2. 100% 复现

---

## 当前关注点 (Current Focus)

**假设 (Hypothesis):** [已确认 - 2026-10-15 13:20]
`sys_api_keys` 表非空（实测 1 行，key 形态 `rec_`+64hex=68 chars），模型新增的 `KeyHash varchar(64) NOT NULL` 无 DEFAULT → AutoMigrate 的 ADD COLUMN 对已有行填 NULL → 违反 NOT NULL → 23502。

**修复方案 (Fix):** 用户已选定 B1 — database.go 中 AutoMigrate 之前的存量回填步骤：
- 检测旧 key 列存在 → ADD COLUMN salt/key_hash/key_prefix (nullable) → 逐行回填（salt=16 随机字节 hex, key_hash=hex(SM3(key+salt)), key_prefix=key[:12]）→ SET NOT NULL → DROP COLUMN key
- 旧列不存在且新列已存在 → no-op（幂等）
- 算法必须与 apikey_service.go 的 generateSalt/hashAPIKey/generateKey 完全一致

**下一步行动 (Next Action):**
(已完成 — 2026-10-15 13:42) B1 修复已实现并验证通过：pre-AutoMigrate 回填步骤落地 dev DB（10.62.10.34），真实 AutoMigrate 路径无 23502，ValidateAPIKey 接受迁移后存量密钥，二次启动幂等 no-op。临时 probe 已删除，未做任何 git 提交。

---

## 证据 (Evidence)

- timestamp: 2026-10-15 13:18:00
  source: model_diff
  detail: |
    git diff internal/models/api_key.go（未提交）:
    - 移除: `Key string` (size:100;uniqueIndex;not null) 明文 `rec_`+64hex
    - 新增: `KeyHash string` (size:64;uniqueIndex;not null) + `Salt string` (size:32;not null) + `KeyPrefix string` (size:12;index;not null)
    - 注释标注 Phase 60 / SEC-01 (D-06)

- timestamp: 2026-10-15 13:19:00
  source: direct_experiment
  detail: |
    临时 pgx 程序连 .env 库 (10.62.10.34):
    - sys_api_keys 行数 = 1
    - 现有列: id,name,key(NOT NULL),user_id,expires_at,last_used_at,is_active,scopes,ip_whitelist,description,inherit_perms + BaseModel 列；key_hash/salt/key_prefix 不存在
    - key 采样: prefix=rec_, len=68 → 与旧模型 `rec_`+64hex 完全吻合，可无损迁移

- timestamp: 2026-10-15 13:21:00
  source: code_review
  detail: |
    internal/services/system/apikey_service.go 迁移必须对齐的算法:
    - hashAPIKey(key, salt) = hex(SM3(key ‖ salt))  (137-142行, 先 key 后 salt)
    - generateSalt() = crypto/rand 16 bytes → hex (32 chars)  (145-151行, per-key 独立)
    - generateKey() = "rec_" + 64hex → KeyPrefix = key[:12]
    - ValidateAPIKey: 按 KeyPrefix 缩窄候选 → SM3(key+salt) + subtle.ConstantTimeCompare

- timestamp: 2026-10-15 13:22:00
  source: code_review
  detail: |
    internal/core/db/database.go 执行顺序 (AutoMigrate():623): SKIP_AUTOMIGRATE 旁路 → cleanupOldConstraints → dropDependentMaterializedViews() → GORM AutoMigrate → migrations。
    关键: AutoMigrate 先于 migrations 执行 → 普通 migration_NNN 无法拦截本错误，必须在 AutoMigrate 之前做存量迁移（先例: cleanupOldConstraints 处理 sys_api_keys.key_hash 约束命名冲突 database.go:385，说明 SEC-01 配套已在落地，唯独缺存量行回填）。

- timestamp: 2026-10-15 13:34:00
  source: implementation
  detail: |
    B1 修复落地 (internal/core/db/database.go):
    - 新增 backfillSysAPIKeysKeyHash() 前置步骤 + backfillAPIKeyHash/backfillGenerateSalt 算法副本（与 apikey_service.go hashAPIKey/generateSalt 逐字节一致，双向注释锚点防漂移）+ tableColumnExists 双方言列探测（不能用 Migrator().HasColumn——模型已无 Key 字段会恒 false）
    - 语义（幂等）: 表不存在 → no-op；无旧 key 列且有 key_hash → no-op；有旧 key 列 → 事务内 ADD COLUMN 三列 nullable（varchar 64/32/12 严格对齐 model tag，保证 AutoMigrate 视为已对齐不再 ALTER）→ 逐行回填（含软删行）→ PG SET NOT NULL → DROP COLUMN key；短 key(<12) fail-fast
    - 不能用 Migrator().AddColumn: FullDataTypeOf 对 not null 字段带 NOT NULL，对存量行同样 23502
    - 调用点: AutoMigrate() sanitize 块之后、cleanupOldConstraints 之前（双方言）；错误 fail-fast 上抛（带根因信息）
    - apikey_service.go 仅加一行漂移锚点注释（hashAPIKey）

- timestamp: 2026-10-15 13:34:00
  source: unit_test
  detail: |
    internal/core/db/backfill_sys_api_keys_test.go 5/5 PASS (sqlite/glebarez):
    legacy 回填+删列+prefix缩窄+SM3恒定时间比对 / 幂等二次 no-op / 已迁移态 no-op / 表不存在 no-op / 短 key fail-fast。
    go test ./internal/core/db/ 全包 PASS；go build ./... PASS。

- timestamp: 2026-10-15 13:36:00
  source: real_db_verification
  detail: |
    临时 probe (cmd/tmp-verify-keyhash, 验证后已删除) 对 dev DB (.env → 10.62.10.34) 执行真实 d.AutoMigrate() 启动路径:
    - [pre] sys_api_keys rows=1, key_len=68 prefix=rec_ is_active=true（legacy 态确认）
    - [apikey-backfill] 检测旧列 → 回填 1 行 → 完成，AutoMigrate 全程 42s 无 23502
    - [post] key 列已删；key_hash/salt/key_prefix 全部 is_nullable=NO；hash==SM3(key+salt) 逐字节一致；prefix==key[:12]
    - ★ 真实 apikey_service.ValidateAPIKey(旧明文 key) → ACCEPTED（端到端金标准）
    - idx_sys_api_keys_key_hash 存在且 indisunique=true（GORM uniqueIndex 实际命名 idx_* 而非 uni_*，与 debug 文件早前假设不同——cleanupOldConstraints 两条名单无害冗余）

- timestamp: 2026-10-15 13:38:00
  source: real_db_verification
  detail: |
    幂等回归: 迁移后再次/三次运行 d.AutoMigrate()（各 ~20s）→ sys_api_keys 零 DDL 零回填（已迁移态 no-op 分支生效），无任何错误。
    观察到一处与本 debug 无关的既有问题: Migrate210NormalizeMenuParentID 对 uuid 列执行 WHERE parent_id='0' 报 22P02（非阻断 ERRO 日志，先前即存在，超出本 scope 未处理）。

---

## 已排除 (Eliminated)

- hypothesis: migration_NNN 写存量迁移即可修复
  reason: AutoMigrate 在 migrations 之前执行，错误发生在 AutoMigrate 阶段，migration 跑不到（同 [gorm-automigrate-blocked-by-matview] 教训）。

---

## 解决方案 (Resolution)

root_cause: 模型未提交的 SEC-01 改动（Phase 60）把 sys_api_keys 明文 Key 列换成 KeyHash/Salt/KeyPrefix 三列（均 not null 无 DEFAULT），而 dev DB 存量表（1 行 legacy 明文 key）上 GORM AutoMigrate 先于 migrations 执行，ADD COLUMN key_hash varchar(64) NOT NULL 对存量行填 NULL → SQLSTATE 23502 启动 FATA；migration_NNN 拦不住（执行顺序），缺的是 AutoMigrate 之前的存量回填步骤。
fix: database.go 新增 AutoMigrate 前置步骤 backfillSysAPIKeysKeyHash()（幂等，双方言）：检测旧 key 列 → 事务内 ADD 三新列 nullable（类型严格对齐 model tag）→ 逐行回填 salt=hex(16 随机字节)/key_hash=hex(SM3(key+salt))/key_prefix=key[:12]（算法与 apikey_service.go generateSalt/hashAPIKey 逐字节一致，双向注释锚点）→ PG SET NOT NULL → DROP COLUMN key；存量密钥经 ValidateAPIKey 仍有效。
verification: sqlite 单测 5/5（回填正确性/幂等/各 no-op 分支/短 key fail-fast）；go build ./... 与 go vet PASS；真实 dev DB (10.62.10.34) 跑 d.AutoMigrate() 启动路径无 23502，1 行回填值 hash/prefix 与明文逐字节一致，真实 ValidateAPIKey ACCEPTED 迁移后密钥，唯一索引 idx_sys_api_keys_key_hash indisunique=true，迁移后二次启动幂等 no-op。
files_changed:
  - internal/core/db/database.go (backfillSysAPIKeysKeyHash + 算法副本 + tableColumnExists + AutoMigrate 调用点 + imports)
  - internal/services/system/apikey_service.go (仅 hashAPIKey 漂移锚点注释)
  - internal/core/db/backfill_sys_api_keys_test.go (新增, 5 个单测)
