---
status: diagnosed
trigger: go test ./internal/services/system/... panic + UNIQUE 约束失败，独立于 MAC 死代码清理
created: 2026-07-02
updated: 2026-07-02
---

# Debug Session: apikey-test-raw-id

## Symptoms

### Trigger (verbatim)
go test ./internal/services/system/... panic + UNIQUE 约束失败，独立于 MAC 死代码清理

### 失败测试
- `TestListAPIKeys/关键词搜索` — `no such function: LEFT` panic @ line 452
- `TestUpdateAPIKey/正常更新` — `unrecognized token: "<uuid-prefix>"` panic @ line 586
- `TestCreateAPIKey/无效作用域` — `"[1001] 用户不存在" does not contain "无效的作用域"` @ line 266

### 关键错误日志（实际验证）
```
=== TestListAPIKeys/关键词搜索
SQL: SELECT count(*) FROM `sys_api_keys` WHERE user_id = "..." AND 
     (name LIKE "%Active%" OR LEFT(key, 12) LIKE "%Active%") AND `sys_api_keys`.`deleted_at` IS NULL
ERROR: no such function: LEFT                              ← SQLite 不支持 LEFT()

=== TestUpdateAPIKey/正常更新
SQL: SELECT * FROM `sys_api_keys` WHERE bdfd820f-a7ac-4be3-afbd-d40023c0938a AND 
     `sys_api_keys`.`deleted_at` IS NULL ORDER BY `sys_api_keys`.`id` LIMIT 1
ERROR: unrecognized token: "4be3"                           ← UUID 被 raw 拼到 WHERE
```

## Current Focus

- hypothesis: ROOT CAUSE CONFIRMED — 三处独立 bug,均与 SQLite 兼容性 + GORM `db.First(ptr, string)` 调用方式相关
- next_action: return diagnosis
- test: ran actual tests, captured SQL logs
- expecting: 三处问题各有独立 fix 路径

## Evidence (CONFIRMED via live test runs)

### Evidence #1 — `db.First(&updated, apiKey.ID)` 把 UUID 当 raw SQL
**File:** `internal/services/system/apikey_service_test.go:584`
```go
db.First(&updated, apiKey.ID)   // ❌ apiKey.ID 是 string,GORM 当作 WHERE raw 表达式
```

**GORM 内部行为** (`gorm.io/gorm@v1.30.5/statement.go:293-318` `BuildCondition`):
```go
if s, ok := query.(string); ok {
    if _, err := strconv.Atoi(s); err != nil {
        // not a number
        if len(args) == 0 || strings.Contains(s, "?") {
            // looks like a where condition
            return []clause.Expression{clause.Expr{SQL: s, Vars: args}}  // ← raw 表达式
        }
        ...
    }
}
```

当 `args == []` 且 string 不含 `?`,GORM 直接把 string 当成 WHERE 表达式。生成 SQL:
```
WHERE bdfd820f-a7ac-4be3-afbd-d40023c0938a AND ...
```
SQLite 把 `bdfd820f` 解析为 token → "unrecognized token"。

**对比 — service 层的正确用法** (`apikey_service.go:282`):
```go
s.db.WithContext(ctx).Preload("User").First(&apiKey, "id = ?", id).Error
                                                    ^^^^^^^^  ^ explicit placeholder
```
带 `?` 占位符,GORM 用 `clause.Eq{Column: "id", Value: id}`,正确生成 `WHERE id = "..."`。

### Evidence #2 — `LEFT(key, 12)` 是 PostgreSQL 函数,SQLite 不支持
**File:** `internal/services/system/apikey_service.go:239`
```go
query = query.Where("name LIKE ? OR LEFT(key, 12) LIKE ?", keyword, keyword)
```

**实测错误:**
```
no such function: LEFT
SELECT count(*) FROM `sys_api_keys` WHERE user_id = "..." AND 
  (name LIKE "%Active%" OR LEFT(key, 12) LIKE "%Active%") AND ...
```

SQLite 没有内置 `LEFT()` 函数(也没有 `RIGHT()`, `SUBSTRING` 需要 `substr()`)。PostgreSQL 有 `LEFT(string, n)` 返回前 n 字符。

**同一文件 line 245 也有 SQLite 不支持的语法:**
```go
query = query.Where("scopes @> ?", fmt.Sprintf("[\"%s\"]", *params.Scope))
```
`@>` 是 PostgreSQL JSONB 包含操作符,SQLite 通过 JSON1 extension 也不直接支持 `@>`。作用域筛选 subtest 会同样失败(虽未被 panic 路径覆盖,因为它排在关键词搜索之后)。

### Evidence #3 — TestCreateAPIKey/无效作用域 失败 = 测试代码 bug (非 service bug)
**File:** `internal/services/system/apikey_service_test.go:178, 226-255, 257-267`
```go
user := createTestUser(t, db)   // line 178 - parent scope user

t.Run("密钥数量限制", func(t *testing.T) {
    limitUser := createTestUser(t, db)   // local user
    ...
    cleanupTestData(t, db)               // ← unscoped delete,会把 parent user 也删!
})

t.Run("无效作用域", func(t *testing.T) {
    _, err := service.CreateAPIKey(ctx, user.ID, req)   // ← user.ID 已被删
    assert.Contains(t, ..., "无效的作用域")               // ← 实际错误是"用户不存在"
})
```

**实测错误:**
```
apikey_service.go:167 record not found
SELECT * FROM `sys_user` WHERE id = "af77fdc5-495d-4063-b66f-0c9870b5bbd2" ...
"[1001] 用户不存在" does not contain "无效的作用域"
```

`cleanupTestData` 用 `Unscoped().Where("1 = 1").Delete(&models.User{})` 真删整个 users 表,parent user 也被清掉。后续 subtest 拿失效 user.ID 调 service 立即被 user-existence check 拦截,永远走不到 scope 验证逻辑。

### 关于"UNIQUE constraint failed" (用户原始报告)
用户的原始报告提到 `line 145 → UNIQUE constraint failed: sys_api_keys.key`。
**实测复现路径**:在 `cache=shared` SQLite 模式下,如果 `TestListAPIKeys` 的 subtest 失败导致前面 `cleanupTestData` 没执行,数据残留,下一次测试运行就会 UNIQUE 冲突。这是个 **二级症状**,根因是 Evidence #2 触发的 panic 中断了 cleanup,而不是 `createTestAPIKey` 本身的 nanos 碰撞。

`createTestAPIKey` 用了 `time.Now().UnixNano()` 拼唯一 key,理论上不会碰撞,但如果一行 `cleanupTestData` 因 panic 跳过,后续测试的 key 不会冲突——但 subtest 之间的 key 是独立的。所以 UNIQUE 主要由 **shared cache + 残留数据** 引起,不是 nanos 重合。

## Eliminated

- hypothesis: MAC 死代码清理导致 panic
  evidence: 在 d8233b70 commit (清理前) 上同样 panic — 用户预验证已排除
  timestamp: 2026-07-02

- hypothesis: nanos 时间戳碰撞导致 UNIQUE
  evidence: 实际跑测试,UNIQUE 在某些运行顺序下出现但根因是 cleanup 因 panic 跳过,不是 nanos 相同;createTestAPIKey 用 %016x%048x 已经确保唯一性
  timestamp: 2026-07-02

- hypothesis: apikey_service.go:282 "record not found" log 是 panic 源头
  evidence: 282 行是 GetAPIKey 的正常 ErrRecordNotFound 分支,行为正确;TestGetAPIKey/密钥不存在 subtest 正常 pass。原始触发条件不存在。
  timestamp: 2026-07-02

## Resolution

root_cause: 三处独立 bug:
  1. apikey_service_test.go:584 `db.First(&updated, apiKey.ID)` 错误调用,GORM 当 raw WHERE 表达式,UUID 未加引号 → SQLite 报 "unrecognized token" → line 586 析构 nil pointer
  2. apikey_service.go:239 用 PostgreSQL 专属函数 `LEFT(key, 12)`,SQLite 不支持 → ListAPIKeys 返回 error → line 452 析构 nil pointer
  3. apikey_service_test.go:226-255 cleanupTestData 真删整个 user 表,影响后续 parent-scope user.ID subtest → TestCreateAPIKey/无效作用域 误报"用户不存在"

fix: 详见 structured diagnosis 报告
verification: 实测三个 fix 方案可消除对应 panic
files_changed:
  - internal/services/system/apikey_service_test.go (Evidence #1 + #3)
  - internal/services/system/apikey_service.go (Evidence #2)