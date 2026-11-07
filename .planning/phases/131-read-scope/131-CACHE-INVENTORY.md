# Phase 131 ops 缓存面清点表（PMODEL-01 防跨范围泄漏）

- 生成日期：2026-11-07
- 依据：`.planning/notes/261107-v152-self-scope-decision.md`（LOCKED §1 关键不变量 3：All 档短路路径同样要满足缓存 key 含 scope 维度）、`.planning/REQUIREMENTS.md` DS-06、ROADMAP Phase 131「验收」条目
- 单一真实源：`internal/services/operations/scope_filter.go` 的 `scopeCacheKey(ctx)` / `scopeCacheKeyOf(scope, ok)`

## 1. scope 维度派生规则

| ScopeResult 档位 | scope 段 | 说明 |
|---|---|---|
| 未注入 / `All=true` | `all` | 超管 / 0 值 / 未挂中间件短路，**key 也带段**（不跳过） |
| `FailClosed=true` | `closed` | 异常枚举 fail-closed |
| `ResourceHook=HookSelfUser` | `self` + sha1(UserID)[:8] | 不同用户不串 key |
| `DeptIDs` 非空 | `d` + sha1(排序后 DeptIDs 逗号拼接)[:8] | Custom/Dept/DeptChild 已展开并排序 |

`DeptIDs` 在 `datascope.Resolve` 侧已 `normalizeIDs`（去重+去空+排序），`scopeCacheKeyOf` 再排序一次保证任何构造来源下稳定。四档 `DeptIDs` 都可能为空，但分档字符串（`all`/`closed`/`self…`/`d…`）互不碰撞（审查 P1-3 已确认）。

## 2. 逐 key 清点（现状 → 目标 → 是否停用 → 失效策略）

| # | 缓存 key（文件） | 现状（改造前） | 目标（Phase 131） | 是否停用 | 失效策略 |
|---|---|---|---|---|---|
| K1 | `floor:tree`（`floor_cache_impl.go:38-45`） | 固定 key、30min、全局共享、**0 维度**；`GetTree` 按可见集过滤后若继续共享会让 A 用户看到 B 的树 | `floor:tree:s{scopeKey}`（如 `floor:tree:sall` / `floor:tree:sd<hash>`）；未注入 ctx 保持旧 `floor:tree`（存量零回归） | **改结构**（非停用） | `InvalidateFloorCache` 用 `floor:tree*` 前缀通配，一次清旧 key + 全部 scope 变体；`InvalidateAllFloorCache` 用 `floor:*` |
| K2 | `floor:building:%s`（`floor_cache_impl.go:48-63`） | 15min、无 scope 段 | `floor:building:s{scopeKey}:%s` | **改结构** | `InvalidateFloorCache` 用 `floor:building:*` 前缀全清；`floor:*` 兜底 |
| K3 | `dropdown:{entity}:all` / `dropdown:{entity}:h{sha1前8}`（`dropdown_helper.go:32-57`） | `BuildDropdownCacheKey` 只 hash filters、无 scope 维度 | 新增 `BuildScopedDropdownCacheKey(ctx, entity, filters)` → `dropdown:{entity}:s{scopeKey}:{all\|h<hash>}`；**保留旧 `BuildDropdownCacheKey` 签名与逐字行为**（`dropdown_helper_test.go` 既有断言不破） | 旧函数保留、调用点切 scoped | `floor:*` / 各实体 CachePatterns 前缀通配仍覆盖新 key（scope 段在 entity 之后） |
| K4 | `floor:*` pattern（`floor_cache_impl.go:78-81`、`excel_config.go:143`） | 通配前缀 | 不变 | 不变 | `InvalidateAllFloorCache` / Excel 导入 `floor:*` 一次清所有 scope 变体 |
| K5 | `dropdown:floor:*`（`geocoding_photo_floor_test.go:441/447` 既有断言） | 前缀断言返回 1 个 key | `dropdown:floor:sall:all` 仍以 `dropdown:floor:` 开头，前缀命中不变 | 不变 | 同上 |
| K6 | `workstation:*` / `workstations:*`（`excel_config.go:159`、`workstation_service.go:644` Backfill 后失效） | 通配前缀 | **不改动**（属 Phase 132 写路径，本 phase 禁区） | 不变 | 132 消费 |
| K7 | `dept:*` / `user:*` / `building:*` / `asset:*` / `infoPoint:*` / `reconciliation:*`（`excel_config.go` CachePatterns） | 各实体导入后通配失效 | 不变（非本 phase 收口缓存面） | 不变 | 既有 |

## 3. 端点 / 缓存命中关系

| 端点 | 是否走缓存 | 收口后 key |
|---|---|---|
| `POST /ops/floor/tree` | 是（`GetTree`） | K1，按 scope 派生 |
| `GET /ops/floor/...`（`GetFloorsByBuildingID`） | 是 | K2，按 scope 派生 |
| `POST /ops/floor/dropdown-options` | 是（仅 `name=""` 走 5min 缓存） | K3 scoped 版本；`name` 非空绕过缓存 |
| `POST /ops/building/dropdown-options` | 否（既有实现不缓存） | 无 key；但读路径已注入可见集 |
| `POST /ops/workstation/dropdown-options` | 否（高频变更，注释明确不缓存） | 无 key；读路径已注入可见集 |
| `POST /ops/infopoint/dropdown-options` | 否 | 无 key；读路径已注入可见集 |
| 四域 List / Statistics / GetByID / counts-by-floor / workstations-all | 否 | 无 key；读路径已注入可见集 |
| Excel 导出两条路径 | 否 | 无 key；**独立注入可见集条件，不依赖 params orgId** |

## 4. 跨用户对照实验（防 PMODEL-01 重演）

`scope_read_403_test.go::TestScopeRead_CrossUserCacheIsolation` 用 SQLite 夹具 + 内存 CacheProvider：
- 用户 A（可见集 deptA）先 `GetTree` → 缓存 `floor:tree:sd<hashA>`（只含楼宇 A）；
- 用户 B（可见集 deptB）再 `GetTree` → 因 key 不同回源，得到只含楼宇 B 的独立树，**断言不是 A 的数据**；
- dropdown 同款：A 拿到 `floorA`、B 拿到 `floorB`；
- `InvalidateAllFloorCache` 后 `floor:*` 全部清空。

`scope_cache_key_test.go` 另断言：不同 scope 产出不同 key、All 档含 `sall` 段、相同 scope 稳定、DeptIDs 顺序无关。

## 5. 审查 P1-1 补收口（非缓存面，端点清点登记）

| 端点 | 现状 | 目标 | 登记 |
|---|---|---|---|
| `GET /ops/workstation/dept-mapping-template`（`router.go:818` → `excel_handler.go:227-237`） | `DeptPathCTE + SELECT ... FROM sys_dept d WHERE d.deleted_at IS NULL AND d.status = ?` 返回**全库部门树**（部门名/全链路 parent_path/部门代码）零范围过滤 | All / 未注入短路 SQL 逐字不变；Custom/Dept/DeptChild 追加 `AND CAST(d.id AS TEXT) IN ?`；Self/FailClosed/异常枚举 → `AND 1=0` 空集；响应形状零变更 | 审查 P1-1 补收口（三份裁定「其余 ops 端点一律收口」） |
