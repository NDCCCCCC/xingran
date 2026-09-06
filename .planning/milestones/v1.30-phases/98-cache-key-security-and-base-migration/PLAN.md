---
phase: 98
phase_name: 缓存键安全与 base 迁移收尾
phase_slug: 98-cache-key-security-and-base-migration
milestone: v1.30
status: planned
planned_at: 2026-09-06
requirements:
  - V130R-04
  - V130R-05
plans:
  - plan_id: 98-01
    title: V130R-04 列表缓存键防碰撞修复
    type: fix
    gap_closure: false
  - plan_id: 98-02
    title: V130R-05 四包 interface{} GetOrSet 迁 base.GetOrSetJSON[T]
    type: refactor
    gap_closure: false
success_criteria:
  - V130R-04: buildListCacheKey 不再碰撞，Username="bob:status:1" 与 Username="bob"+Status=1 生成不同键
  - V130R-05: duty/knowledge/network/workorder cache_impl 全部 11 处 interface{} GetOrSet 收敛 base.GetOrSetJSON[T]，cache_invariants_92_test warning 档清零
  - go build ./... && go test ./... 0 failure
  - 七 gate 不倒退
---

# Phase 98 Plan: 缓存键安全与 base 迁移收尾

## Context

Phase 96 刚修复 duty/workorder cache_impl 缺陷（CACHEDEF-03/04），Phase 98 依赖 Phase 96 成果继续——同一文件族，先修缺陷再迁移，避免同文件冲突。

### V130R-04 根因（缓存键碰撞）

`user_cache_impl.go:172-216` 与 `role_cache_impl.go:51-78` 的 `buildListCacheKey` 用 `:` 分隔参数名和参数值：
```
key = "user:list:username:bob:status:1:page:1:size:10"
```
若 `Username = "bob:status:1"`，则：
- `Username="bob:status:1"` → `"user:list:username:bob:status:1:..."`
- `Username="bob"` + `Status=1` → `"user:list:username:bob:status:1:..."`

**两者完全相同，互相污染。**

### V130R-05 根因（interface{} 残留）

Phase 92 将 system/operations 的 cache_impl 收敛到 `base.GetOrSetJSON[T]`，但 duty/knowledge/network/workorder 四包（A5 范围外）未处理。残留 11 处 `interface{}` 闭包式 `GetOrSet`：

| 文件 | 方法 | 行 |
|------|------|-----|
| duty/duty_cache_impl.go | GetTodayDuty | ~142 |
| duty/duty_cache_impl.go | GetMonthlyDutySchedule | ~159 |
| duty/duty_cache_impl.go | GetHolidayList | ~239 |
| knowledge/knowledge_cache_impl.go | GetKnowledgeArticle | ~96 |
| knowledge/knowledge_cache_impl.go | GetKnowledgeCategoryList | ~151 |
| knowledge/knowledge_cache_impl.go | GetAllTags | ~204 |
| network/cache_impl.go | GetDeviceStatistics | ~273 |
| network/cache_impl.go | GetDevicesByDept | ~290 |
| network/cache_impl.go | GetDevicesByCredential | ~307 |
| workorder/workorder_cache_impl.go | GetMyPending | ~223 |
| workorder/workorder_cache_impl.go | GetStatistics | ~252 |

**共计 11 处。**

---

## Plan 98-01: V130R-04 列表缓存键防碰撞修复

### Steps

**Step 1: 确认碰撞边界（5 min）**
- 审查 `user_cache_impl.go:buildListCacheKey` 与 `role_cache_impl.go:buildListCacheKey`
- 确认所有使用 `:` 拼接参数值的调用点
- 边界：`Username`/`RoleName`/`RoleKey` 可能含 `:`，其他字段（status/page/size）通常不含

**Step 2: 选择方案并实施（20 min）**

方案 A（推荐）：URL-escape 参数值中的 `:` → `%3A`
- `strings.ReplaceAll(value, ":", "%3A")`
- 读取时无需解码（键构造和解构对称）
- 优点：实现简单，零运行时开销
- 缺点：键中含 `%` 字符（但不会二次碰撞）

方案 B：参数集哈希（`sha256(sortedParams)`）
- 优点：绝对不碰撞
- 缺点：键不可读，调试困难，diff 可见性差

**采用方案 A**（URL-escape `:`）。

**Step 3: 写回归测试（15 min）**

在 `user_cache_impl_test.go` 或新建 `user_cache_key_collision_test.go`：
- 测试 `Username="bob:status:1"` 与 `Username="bob"`+`Status=1` 生成不同键
- 测试含 `:` 的其他字段（deptId 等）

**Step 4: 同步修复 role_cache_impl.go（10 min）**
- 同理修复 `role_cache_impl.go:buildListCacheKey`

### Files to Modify

- `internal/services/system/user_cache_impl.go` — `buildListCacheKey`
- `internal/services/system/role_cache_impl.go` — `buildListCacheKey`
- `internal/services/system/user_cache_impl_test.go` 或新建测试文件

### Verification

```bash
# 碰撞测试
go test -v -run TestCacheKeyCollision ./internal/services/system/

# 全量构建
go build ./...
go test ./internal/services/system/... -count=1
```

---

## Plan 98-02: V130R-05 四包 interface{} GetOrSet 迁 base.GetOrSetJSON[T]

### Steps

**Step 1: 统计现状（5 min）**
- 运行 `cache_invariants_92_test.go` 确认 warning 档数量基线
- 确认 11 处具体位置

**Step 2: 迁移 duty/duty_cache_impl.go（15 min）**

3 处：
```go
// BEFORE（interface{} 闭包）
err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
    return s.base.GetTodayDuty(ctx)
})

// AFTER（base.GetOrSetJSON[T]）
return base.GetOrSetJSON(ctx, s.cache, cacheKey, expiration, s.base.GetTodayDuty)
```
注：`GetTodayDuty` 返回 `([]services.TodayDutyMember, error)` → `func() ([]services.TodayDutyMember, error)` 签名匹配。

**Step 3: 迁移 knowledge/knowledge_cache_impl.go（20 min）**

4 处，逐个替换。

**Step 4: 迁移 network/cache_impl.go（15 min）**

3 处。

**Step 5: 迁移 workorder/workorder_cache_impl.go（15 min）**

2 处（`GetMyPending` 返回结构体需要调整返回值类型）。

**Step 6: 收敛 4 个平行 getExpiration（10 min）**

四包的 `getExpiration` 方法：
```go
func (s *xxxCacheServiceImpl) getExpiration(configKey string, defaultVal time.Duration) time.Duration {
    if s.config != nil {
        return s.config.GetDurationWithDefault(configKey, defaultVal)
    }
    return defaultVal
}
```
→ 删除，改为在调用处直接引用 `base.GetExpiration` 或让各 service 实现 `base.TTLResolver`。

实际分析：四包的 `getExpiration` 是 `CacheServiceBase` 的方法，Phase 92 约定 `CacheServiceBase.GetExpiration` 是统一入口。但四包 cache_impl 各自实现了自己的 `getExpiration` 而没有 embed `CacheServiceBase`。

**决策**：让四包的 struct embed `base.CacheServiceBase`，删除各自重复的 `getExpiration`。

**Step 7: 运行 invariants 测试（5 min）**

```bash
go test -v -run TestNoInterfaceGetOrSetResidue ./internal/services/system/
```

预期：warning 档 duty/knowledge/network/workorder 全部为 0。

### Files to Modify

- `internal/services/duty/duty_cache_impl.go` — 3 处 interface{} GetOrSet + 删除 getExpiration + embed CacheServiceBase
- `internal/services/knowledge/knowledge_cache_impl.go` — 4 处 + embed + 删除 getExpiration
- `internal/services/network/cache_impl.go` — 3 处 + embed + 删除 getExpiration
- `internal/services/workorder/workorder_cache_impl.go` — 2 处 + embed + 删除 getExpiration
- 每个文件对应的新增测试文件（回归测试）

### Verification

```bash
# interface{} 残留扫描
go test -v -run TestNoInterfaceGetOrSetResidue ./internal/services/system/

# 全量构建和测试
go build ./...
go test ./internal/services/duty/... ./internal/services/knowledge/... ./internal/services/network/... ./internal/services/workorder/... -count=1
```

---

## Wave Analysis

**Wave 1（可并行）：**
- Plan 98-01（V130R-04）：独立文件 user_cache_impl.go / role_cache_impl.go
- Plan 98-02（V130R-05）：独立包 duty/knowledge/network/workorder

**无依赖冲突**：98-01 改 system 包，98-02 改其他四包，零文件重叠。

**建议并行执行**（两个 plan 完全独立）。

---

## Regression Guards

1. `cache_invariants_92_test.go` — warning 档清零是 V130R-05 的强制验证
2. V130R-04 回归测试 — 碰撞场景覆盖
3. `go build ./...` — 无编译错误
4. `go test ./...` — 0 failure
5. 七 gate 全程不倒退

---

## Notes

- Phase 96 刚修完 duty/workorder 的 CACHEDEF-03/04，Phase 98 的 V130R-05 迁移是同一文件族的后续操作（先修缺陷再迁移，避免冲突）
- V130R-04 的 URL-escape 方案简单但需确认 `%` 不会二次碰撞（应该不会，因为键格式本身不含 `%`）
- V130R-05 的 4 个 getExpiration 删除后，TTL 来源统一到 base.CacheServiceBase.GetExpiration
