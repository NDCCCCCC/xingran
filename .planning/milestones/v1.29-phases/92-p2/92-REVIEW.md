---
phase: 92-p2
reviewed: 2026-09-05T05:33:58Z
depth: standard
files_reviewed: 32
files_reviewed_list:
  - internal/api/v1/monitor/cache_router.go
  - internal/core/core.go
  - internal/services/base/cache_functions.go
  - internal/services/base/cache_provider.go
  - internal/services/base/cache_service_base.go
  - internal/services/base/cache_service_base_test.go
  - internal/services/cache_config_service.go
  - internal/services/data_cache_service.go
  - internal/services/duty/duty_cache_impl.go
  - internal/services/knowledge/knowledge_cache_impl.go
  - internal/services/monitor/cache_service.go
  - internal/services/monitor/cache_service_test.go
  - internal/services/network/cache_impl.go
  - internal/services/operations/cache_invalidator.go
  - internal/services/operations/floor_cache_impl.go
  - internal/services/system/cache_adapter.go
  - internal/services/system/cache_infra_test.go
  - internal/services/system/cache_invariants_92_test.go
  - internal/services/system/cache_provider.go
  - internal/services/system/cache_utils.go
  - internal/services/system/config_cache_impl.go
  - internal/services/system/department_cache_impl.go
  - internal/services/system/dict_cache_impl.go
  - internal/services/system/menu_cache_impl.go
  - internal/services/system/notice_cache_impl.go
  - internal/services/system/post_cache_impl.go
  - internal/services/system/role_cache_impl.go
  - internal/services/system/role_cache_impl_test.go
  - internal/services/system/settings_cache_impl.go
  - internal/services/system/user_cache_impl.go
  - internal/services/system/user_cache_impl_test.go
  - internal/services/workorder/workorder_cache_impl.go
findings:
  critical: 0
  warning: 6
  info: 5
  total: 11
status: issues_found
---

# Phase 92-p2: Code Review Report

**Reviewed:** 2026-09-05T05:33:58Z
**Depth:** standard
**Files Reviewed:** 32
**Status:** issues_found

## Summary

本阶段（Phase 92 CACHE-UNIFY 内部重构，零业务行为变更）整体质量高。逐文件对照 `5ae394c^` 基线 diff 验证结论：

- **行为零变更达成**：system 9 个 `*_cache_impl` + `floor_cache_impl` 全部 32 处 `GetOrSet` 样板迁至 `base.GetOrSetJSON[T]`，缓存键构造（含 `buildMyNoticesKey` 两个 Sprintf 分支）、TTL 常量与默认值、失效键/模式集合逐字节比对均与迁移前一致。`noticeListPage` 具名化保持 JSON 字段名/顺序，现存 Redis 键迁移后可继续命中。
- **依赖方向正确**：`internal/services/base` 非测试文件仅 import stdlib + `pkg/logger` + `gorm`，零依赖红线成立；`system/cache_provider.go` type alias 编译期断言 `var _ CacheProvider = (*NoOpCacheProvider)(nil)` 在位。
- **D-08 rename 干净**：`monitor.CacheProvider → CacheOperator` 仅 10 行签名/注释替换，无行为面改动；`core.go` 仅 1 处 `base.InvalidatePattern` 换名（上游有 `c.Cache != nil` 守卫）。
- **已接受的执行偏差**（`base.SetValue` 导出、mock role provider dest 回填、`(&base.CacheServiceBase{...})` 寻址、invariants 扫描器 FieldList 空翻译单元）复核后确认均未引入缺陷。
- **验证**：`go build ./...` 通过；`go vet` 相关 10 包零告警；`TestBase92_*`（MemoryCache + miniredis 双装配）、`TestNoInterfaceGetOrSetResidue`（硬档 0 残留，warning 档 duty/knowledge/network/workorder 合计 11 处按设计记日志）、system/monitor/operations 包测试全部 PASS。

**未发现本阶段引入的 Critical 缺陷。** 6 个 Warning 中 5 个（WR-01～WR-05）是迁移前即存在、被"零行为变更"约束原样保留的真实缺陷，**不得在本阶段内"顺手修复"**（改键/改失效语义即违反 v1.29 D-05 与字节级键一致性约束），应登记后续行为变更阶段处理；WR-06 是本阶段新增代码自身的问题，可在本阶段内以删除死代码方式消除。

## Critical Issues

无。

## Warnings

### WR-01: 部门下拉缓存键 `dept:tree` 不在任何失效模式覆盖范围内（pre-existing）

**File:** `internal/services/system/department_cache_impl.go:74-79`（写路径）、`internal/services/system/department_cache_impl.go:95-102`（失效路径）
**Issue:** `GetSelectDataWithCache` 以裸常量 `CacheKeyDeptTree`（= `"dept:tree"`，见 `cache_keys.go:110`）作为缓存键，而 `InvalidateDeptCache` 的三个失效模式均为 `BuildDeptCacheKey(...) + "*"` = `"cache:dept:tree*"` / `"cache:dept:list*"` / `"cache:dept:tree:select*"`（`BuildDeptCacheKey` 恒带 `cache:` 前缀，见 `cache_keys.go:179-182`）。模式 `"cache:dept:tree*"` 无法匹配键 `"dept:tree"`——部门 Create/Update/Delete/BatchDelete/UpdateStatus 后，下拉数据缓存最长 30min（TTL）返回陈旧部门树。键与模式的不匹配自迁移前即存在（diff 确认旧代码同样写 `cacheKey := CacheKeyDeptTree`），本次重构按行为一致要求原样保留。缓解因素：全仓 grep 未发现 `GetSelectDataWithCache` 的生产调用方（仅测试引用），当前为潜伏缺陷。
**Fix:** 后续行为变更阶段：将 `GetSelectDataWithCache` 的键改为 `BuildDeptCacheKey("tree:select")`（与失效模式 `"cache:dept:tree:select*"` 对齐），或在 `InvalidateDeptCache` 中补 `"dept:tree*"` 精确模式；需同步评估 Redis 现存 `"dept:tree"` 键的一次性清理。**本阶段禁止修改（D-05 零行为变更 + 字节级键一致性）。**

### WR-02: config Delete 路径失效遗漏 `config:id:<id>`（pre-existing）

**File:** `internal/services/system/config_cache_impl.go:71-78`（`InvalidateConfigCache`）、`internal/services/system/config_cache_impl.go:104-118`（`Delete`）、`internal/services/system/config_cache_impl.go:37-42`（`GetByID` 写键）
**Issue:** `GetByID` 以 `"config:id:%s"` 读穿透缓存，但 `Delete` 成功后只调 `InvalidateConfigCache`，其键列表仅含 `"config:all"` 与 `"config:key:%s"`，不含 `"config:id:<被删id>"`（`Update`/`BatchDelete` 走 `InvalidateAllConfigCache` → `"config:*"` 模式可覆盖，唯独单条 `Delete` 路径遗漏）。结果：删除配置后，配置详情接口（`config_handler.go:222` 经 `config_router.go:17` 装配的缓存版服务）最长 30min 仍能从缓存读回已删除的配置。pre-existing，diff 确认键集合迁移前后一致。
**Fix:** 后续行为变更阶段：`InvalidateConfigCache` 键列表追加 `fmt.Sprintf("config:id:%s", id)`（需把入参从 `configKey` 扩为同时携带 id，或 `Delete` 内直接追加该键）。**本阶段禁止修改。**

### WR-03: duty 月度排班失效恒计算 month=0，月度缓存失效无效（pre-existing）

**File:** `internal/services/duty/duty_cache_impl.go:122-127`（`GenerateSchedule`）、`internal/services/duty/duty_cache_impl.go:184-189`（`ManualDuty`）、`internal/services/duty/duty_cache_impl.go:333-344`（`parseInt`）
**Issue:** `parseInt` 要求 `len(s) >= 4` 才解析，而月份切片 `req.StartDate[5:7]` / `req.DutyDate[5:7]` 恒为 2 字符（如 `"07"`）→ `parseInt` 直接返回 0。于是 `InvalidateMonthlyScheduleCache(ctx, 2026, 0)` 删除键 `"duty:monthly:2026:0"`，而真实缓存键是 `"duty:monthly:2026:7"`（`GetMonthlyDutySchedule` 写入，`duty_handler.go:325` 生产可达）。排班生成/手动值班后，受影响月份的月度排班缓存最长 30min 陈旧。pre-existing（`parseInt` 及两个调用点本次 diff 未触碰）。
**Fix:** 后续行为变更阶段：`parseInt` 去掉 `len(s) >= 4` 前置条件（逐位跳过非数字已足够），或改用 `strconv.Atoi` + 容错。**本阶段禁止修改。**

### WR-04: workorder 待办缓存键忽略 `req.Limit` 参数（pre-existing）

**File:** `internal/services/workorder/workorder_cache_impl.go:207-234`
**Issue:** `GetMyPending` 缓存键仅含 userID（`"workorder:my_pending:%s"`），但 `GetMyPendingRequest.Limit`（`base.go:159-161`，默认 5、上限 100）影响查询结果。同一用户先以 limit=5 触发缓存后，再以 limit=20 请求会在 2min TTL 内命中 limit=5 的缓存，返回截断的列表/错误的 Total。pre-existing（本文件 diff 仅替换失效底层调用）。
**Fix:** 后续行为变更阶段：键中加入 limit（如 `fmt.Sprintf("workorder:my_pending:%s:%d", userID, limit)`），或对 limit 做白名单归一（0/5 视为同一档）后再入键。**本阶段禁止修改。**

### WR-05: `normalizeCacheKeyForService` 为恒等函数，监控页前缀剥离永不生效（pre-existing，quirk 锁定）

**File:** `internal/services/monitor/cache_service.go:766-771`
**Issue:** `key[:6] == "xingran:"` 用 6 字节切片与 8 字节字面量比较恒为 false，函数实际是恒等函数——监控页传入带 `xingran:` 前缀的键时，`del`/`exists`/`expire` 操作（经底层再补前缀）永远打在不存在的键上，静默无效。与 `CLAUDE.md`"Cache prefix confusion"一节记录的预期行为（"always strip it first"）直接矛盾；且 `CLAUDE.md` 给出的 `key[6:]` 示例本身也是错的（`"xingran:"` 为 8 字节，应为 `key[8:]`），文档与实现双重失效。该 quirk 已被 Phase 73-04 明确认收并以测试锁定（`cache_service_test.go:12-13` Q1、`TestNormalizeCacheKeyForService`），属于已知偏差而非本次回归，故列 Warning 而非 Critical。
**Fix:** 后续行为变更阶段：`len(key) > 8 && key[:8] == "xingran:" → key[8:]`；同步更新 `cache_service_test.go` 的 Q1 锁定断言与 `CLAUDE.md` 的 `key[6:]` 示例。**本阶段禁止修改。**

### WR-06: `base.SetJSON` 零调用方（死代码）且"先删后写"组合存在并发丢写窗口（new code）

**File:** `internal/services/base/cache_functions.go:53-61`
**Issue:** 两点：(1) 全仓 grep 确认 `base.SetJSON` 无任何调用方（现存 `SetJSON` 调用均为 `pkg/cache.Cache` 接口同名方法），属于"函数族完备性"预置 API，当前为死代码；(2) 其实现为 `Delete(key)` 成功后 `GetOrSet(key, &dest, ttl, 恒返回 value 闭包)`——若并发读穿透在 Delete 与 GetOrSet 之间回源重建了该键，GetOrSet 会判定命中并**静默跳过写入**，调用方既没写入 value，dest 还被回填为并发方回源的值，覆盖语义无声丢失。注释中"删除与写入之间的中间窗口读者只会 miss 回源，无脏读"只论证了读者侧，未覆盖这个写者侧竞态。零调用方意味着当前无实际影响，但这是面向未来调用者导出的 API，语义陷阱会被继承。
**Fix:** 本阶段内可直接修复（零调用方，删除无行为面影响）：删除 `SetJSON`，待 `CacheProvider` 接口未来引入原生 `Set`（D-02 锁定的 9 方法之外）时再以真 Set 原语重实现；若决定保留，至少在注释中写明"并发读穿透可致本次写入静默丢失"的约束。

## Info

### IN-01: duty/knowledge/network/workorder 仍保留 4 份平行 `getExpiration` TTL 逻辑（记录在案的范围决策）

**File:** `internal/services/duty/duty_cache_impl.go:80-85`、`internal/services/knowledge/knowledge_cache_impl.go:70-75`、`internal/services/network/cache_impl.go:66-71`、`internal/services/workorder/workorder_cache_impl.go:71-76`
**Issue:** 四个外围模块未嵌入 `base.CacheServiceBase`，各持私有 `getExpiration`（与 `base.GetExpiration` 同构）。`cache_invariants_92_test.go` 只锁定 GetOrSet 闭包残留（硬档 system/operations，warning 档仅计数），TTL 平行逻辑无回归护栏；后续若有人改动 `base.GetExpiration` 的 nil 语义，这 4 份拷贝不会同步。
**Fix:** 按 A5 计划在 v1.30+ 迁移这 4 个模块时收敛到 `CacheServiceBase`；可考虑届时给 invariants 测试加一条"私有 getExpiration 函数残留"扫描。

### IN-02: TTL 配置键使用字符串字面量而非既有常量

**File:** `internal/services/system/settings_cache_impl.go:42`、`internal/services/system/notice_cache_impl.go:219`、`internal/services/system/notice_cache_impl.go:241`、`internal/services/system/config_cache_impl.go:56`、`internal/services/operations/floor_cache_impl.go:41`、`internal/services/operations/floor_cache_impl.go:51`、`internal/services/operations/floor_cache_impl.go:165`
**Issue:** `services.CacheConfigSettingsUser` / `CacheConfigNoticeMyNotices` / `CacheConfigNoticeUnreadCount` / `CacheConfigConfigAll` / `CacheConfigFloorTree` / `CacheConfigFloorBuilding` 等常量已在 `cache_config_service.go` 定义，但调用点写的是等值字符串字面量（如 `"cache.settings.user"`），与项目"常量唯一真相源"约定不一致；字面量拼错不会有编译期报错。pre-existing，迁移逐字保留。
**Fix:** 后续清理阶段统一替换为对应 `services.CacheConfig*` 常量。

### IN-03: `system/cache_provider.go` 头注释与 D-02 最终决策矛盾

**File:** `internal/services/system/cache_provider.go:8`
**Issue:** 注释称"alias 后续可删（D-02 锁定，删除属 92-03 收尾范围）"，但本交付（含 92-03/92-04）的锁定决策是**保留** type alias 作为原位引用层（20+ 消费文件零改动依赖它），注释会误导后续维护者真的去删除。
**Fix:** 改为"alias 为 D-02 锁定的长期形态，删除需先迁移全部消费点"。

### IN-04: 精确键经模式失效通道传递（语义混用，当前无害）

**File:** `internal/services/system/notice_cache_impl.go:296`
**Issue:** `InvalidateUserNoticeCache` 把无通配符的精确键 `fmt.Sprintf("notice:unread_count:%s", userID)` 与通配模式混在同一 `InvalidatePattern` 列表。当前 `DeleteByPattern` 以 Keys 全匹配实现，行为正确；但若 userID 未来允许含 `*`/`?`/`[` 等字符，或底层实现换成原生 SCAN/UNLINK 语义，精确键会被当 glob 解释。
**Fix:** 迁移期原样保留即可；后续可将精确键拆到 `base.Invalidate` 列表、仅模式项走 `InvalidatePattern`。

### IN-05: 测试卫生两则

**File:** `internal/services/system/notice_cache_impl.go:60`、`internal/services/system/user_cache_impl_test.go:477-480`
**Issue:** (1) `noticeCacheService.db` 字段构造时赋值但全文件无读取（pre-existing 死字段）；(2) `TestUserCache_ApperrorsImport` 是仅做 `_ = context.TODO` 的占位测试，名为"防 unused import"实际无断言价值（apperrors import 在生产文件中，测试文件的 import 与其无关）。
**Fix:** 删除 `db` 字段（连同构造参数如无他用）；删除占位测试。

---

_Reviewed: 2026-09-05T05:33:58Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
