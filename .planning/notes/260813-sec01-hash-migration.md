# SEC-01: API Key SM3 单向哈希存储迁移决策记录

**Phase**: 60-security-hardening-and-enable-decision (Plan 02 / Task 1)
**Requirement**: SEC-01（P2-c「API Key 明文存储」根因缺陷收敛）
**Date**: 2026-08-13
**锁定决策来源**: `.planning/phases/60-security-hardening-and-enable-decision/60-CONTEXT.md` D-05 / D-06 / D-07 / D-08 / D-09
**前提**: 用户确认「当前没有使用中的 API key」→ 无需双读兼容期 / 回填脚本（D-08）。

---

## 1. 存储方案（D-05）

API Key 的 data-at-rest 保护方案选型。三种候选：

| 方案 | 可逆性 | 密钥管理依赖 | 离线爆破面 | 与国密栈一致 | 决策 |
|------|--------|--------------|------------|--------------|------|
| **SM3 单次哈希** (`hashAPIKey`) | 不可逆 | 无（无 SM4_KEY） | 极低（256-bit 熵 key 无字典攻击） | ✅ sm3 原语复用 | **✅ 选定** |
| SM4 对称加密 | 可逆 | 依赖 `SM4_KEY`（泄漏即全部明文暴露） | 取决于 SM4_KEY 强度 | ✅ | ❌ |
| PBKDF2-SM3（如密码哈希） | 不可逆 | 无 | 极低 | ✅ | ❌ 过度设计 |

**选 SM3 单次哈希的理由**：

1. **API Key 是 256-bit 高熵随机串**（`rec_` + 64 hex = 32 字节随机），不是低熵用户密码 → 无字典/彩虹表攻击面，PBKDF2 的迭代加固无收益，单次 SM3 足够。
2. **不可逆**：DB 文件泄漏后无法还原明文 key，直接消除 P2-c 根因。SM4 对称加密虽能「解密回明文」，但收益为零（管理员只需展示一次明文）却引入 `SM4_KEY` 单点依赖——key 一旦泄漏，全表明文同时暴露。
3. **与密码哈希国密栈一致**：复用 `github.com/tjfoc/gmsm/sm3` 原语（见 `internal/core/security/password.go:79-82`），不引入新依赖。
4. **格式不复用 HashPassword 的 `$sm3$iterations$salt$hash`**：避免与密码哈希格式混淆；API Key 用裸 hex 哈希，无前缀（Claude's Discretion，D-05 锁定）。

**实现位置**: `internal/services/system/apikey_service.go` `hashAPIKey(key, salt string) string`（`sm3.New()` + `h.Write(key)` + `h.Write(salt)` + `hex.EncodeToString(h.Sum(nil))`）。

---

## 2. Schema 变更（D-06）

`internal/models/api_key.go` `APIKey` struct 字段替换：

| 移除 | 新增 | gorm tag | json tag | 用途 |
|------|------|----------|----------|------|
| `Key string` (`size:100;uniqueIndex;not null;json:"key"`) | `KeyHash string` | `size:64;uniqueIndex;not null` | `json:"-"`（隐藏） | SM3(key+salt) hex 64 字符，DB 唯一索引 |
| — | `Salt string` | `size:32;not null` | `json:"-"`（隐藏） | 16 字节随机盐 hex 32 字符 |
| — | `KeyPrefix string` | `size:12;index;not null` | `json:"keyPrefix"`（暴露） | 明文前 12 字符，用于 List 搜索 + 展示 |

**关键设计**：
- `KeyHash` / `Salt` 的 `json:"-"` → handler 序列化层绝不暴露哈希与盐（`apikey_handler.go` `maskAPIKeys` 仅输出 `keyPrefix`）。
- `KeyPrefix` 普通索引（非 unique）：多 key 前 12 字符理论上可重复（虽概率极低），用普通 index 支持 LIKE 搜索。
- `TableName()` 不变（`sys_api_keys`）。
- GORM AutoMigrate 启动时自动建 `sys_api_keys_key_hash_key`（uniqueIndex），`Key` 列与其索引 `idx_api_keys_key` 退役（见 SEC-02 索引收敛）。

**VerifyAPIKey 改造**（`apikey_service.go`）：
不再 `WHERE key = ?` 明文查询。改为 **KeyPrefix 缩窄候选 + SM3 恒定时间比对**：
```go
// 1. 前缀缩窄候选行（前缀碰撞罕见，候选集极小）
var candidates []models.APIKey
db.Where("key_prefix = ? AND is_active = ?", keyStr[:12], true).Find(&candidates)
// 2. 逐候选恒定时间比对（防侧信道时序）
for i := range candidates {
    if subtle.ConstantTimeCompare([]byte(hashAPIKey(keyStr, candidates[i].Salt)), []byte(candidates[i].KeyHash)) == 1 {
        return &candidates[i]  // 命中
    }
}
```

---

## 3. List 搜索（D-07）

`ListAPIKeys` 关键词搜索原依赖 dialect 分支（SQLite `substr(key,1,12)` / PostgreSQL `LEFT(key,12)`），现在 `key_prefix` 是真实列，LIKE 在两 dialect 上一致：

```diff
- isSQLite := s.db.Dialector.Name() == "sqlite"   // keyword 分支用
- if isSQLite { query = query.Where("name LIKE ? OR substr(key, 1, 12) LIKE ?", ...) }
- else        { query = query.Where("name LIKE ? OR LEFT(key, 12) LIKE ?", ...) }
+ query = query.Where("name LIKE ? OR key_prefix LIKE ?", keyword, keyword)
```

**为什么用 `KeyPrefix` 而非 `KeyHash` 做搜索**：前缀是明文片段，可读可搜；哈希不可逆，无法按关键词命中。`KeyPrefix` 让管理员仍能按「记得的前几位」检索 key。

**范围限定**：Scope 搜索的 dialect 分支（SQLite `json_each` / PG `scopes @>`）**保留**——本 phase 仅收敛 KeyPrefix 一项，scope 搜索属 Phase 61 范畴。`isSQLite` 局部变量因此保留（scope 分支仍需要）。

---

## 4. 迁移路径（D-08）

**前提**：用户确认「现在没有使用中的 API key」→ **直接切换，无迁移、无双读兼容期、无回填脚本**。

| 决策 | 理由 |
|------|------|
| 不写 Go migration（如 migration_086） | 无存量明文 key 需哈希化；新 schema 由 AutoMigrate 自动建表 |
| 不写回填脚本 | DB 中无活跃 key 行需转换 |
| 不双读 | 旧客户端持有的明文 key 不存在 → 无需「先查 KeyHash，再回退查 Key」兼容期 |

**若未来出现活跃 key 的扩展路径**（记录备用，本 phase 不实现）：
1. 临时双读：`ValidateAPIKey` 先走 KeyHash 比对，失败再 `WHERE key = ?`（保留旧 Key 列一个过渡期）。
2. 回填脚本：遍历存量明文 key → `generateSalt` + `hashAPIKey` → 写 KeyHash/Salt/KeyPrefix → 删 Key 列。
3. 当前选「直接切换」是因前提排除了这些成本。

---

## 5. 创建流程（D-09）

`CreateAPIKey` 一次性返回明文，DB 永不存储明文：

```
generateKey()         → 明文 key (rec_ + 64 hex = 68 字符)
generateSalt()        → 16 字节随机盐 (hex 32 字符)
hashAPIKey(key, salt) → SM3 hex 哈希 (64 字符)
存储: KeyHash + Salt + KeyPrefix(key[:12])  ← DB 只存这三列
返回: *string(明文 key)  ← 仅本次响应一次性返回给管理员
```

**为什么一次性返回明文**：管理员密钥管理 UX 标准模式（类比 AWS / GitHub PAT）——创建时展示一次，之后无法再查看，只能轮换。这与「不可逆哈希存储」天然契合：DB 无法还原，前端也无法二次查询。

**轮换 = 重新创建**：忘记明文 → 删旧 key → 创建新 key（新明文一次性返回）。无「重置明文」路径（明文本就不存）。

**handler 层**（`apikey_handler.go`）：`Create` 返回 `gin.H{"key": *key}`（明文，仅此一次）；`GetByID` / `maskAPIKeys` 仅暴露 `KeyPrefix`，`KeyHash`/`Salt` 由 `json:"-"` 隐藏。

---

## 关联

- **SEC-02**（`.planning/notes/260813-sec02-redundant-index-removal.md`）：Key 列退役后，`idx_api_keys_key` 冗余索引收敛。
- **Phase 57**：MultiAuth 中间件（`apikey.go`）已就绪，本 phase 不改动中间件，仅改存储层。
- **Phase 61 / AUTH-04**：资源级权限矩阵 + InheritPerms 真实生效；SEC-01 哈希存储为其铺平凭据层。
