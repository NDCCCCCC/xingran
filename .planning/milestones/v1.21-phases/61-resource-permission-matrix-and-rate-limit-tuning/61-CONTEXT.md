# Phase 61: 资源权限矩阵与限流生产调优 - Context

**Gathered:** 2026-08-13
**Status:** Ready for planning

<domain>
## Phase Boundary

让 `RequireAPIKeyResourcePermission(resource, action)` 的 `resource` 参数真实生效（资源→权限映射接入 `system:*` 模块），并把 `RateLimitByScope` 的多 scope 选择逻辑从「任意取 scopes[0]」改造为「按 action 推 required scope、scopes 不含则拒绝」的严格语义；同时让限流阈值走 `sys_config` 参数表实现运维可调。

**条件已满足**:Phase 60 AUTH-03=启用（`/system/apikeys/*` 已挂载 MultiAuth），Phase 61 启动。

**Requirements:** AUTH-04（`RequireAPIKeyResourcePermission.resource` 参数真实生效）、QUAL-03（`RateLimitByScope` 生产接入 + 多 scope 选择逻辑正确）。

**本 phase 不做的事**（Phase 60 D-13 已划出 scope）:
- 密钥轮换/吊销机制、配额告警（FUTURE-APIKEY-03/04 仍 v2 Future）
- 完整 RBAC 资源实例级权限（行级数据权限）—— 本 phase 只做模块级 `system:resource:action` 校验，不做行级
- per-API-Key 限流 override（统一阈值，不做 DB 表 per-key 配置）

</domain>

<decisions>
## Implementation Decisions

### 资源→权限映射形态（AUTH-04）

- **D-01:** **静态 map（编译期常量）**。在 `pkg/permission/` 下新增 `resource_action_map.go`，预定义 `{resource, action}` → `PermissionCode` 的显式映射表。**不**采用动态拼接（避免拼错漏检）或 DB 表（避免新增 schema、违背 v1.21 SEC-01「不写 Go migration」的取向）。
- **D-02:** **覆盖范围仅 `system:*` 模块**。Map 涵盖 `system:user:list/view/add/edit/remove/export/import/resetPwd`、`system:role:list/view/add/edit/remove/export`、`system:menu:list/view/add/edit/remove`、`system:dept:list/view/add/edit/remove`、`system:post:list/view/add/edit/remove`、`system:workstation:list/view/add/edit/remove`、`system:dict:list/view/add/edit/remove`、`system:config:list/view/add/edit/remove`、`system:captchaBackground:list/view/add/edit/remove`、`system:notice:list/view/add/edit/remove`、`system:apikey:list/view/add/edit/remove`。**不**纳入 `monitor:*` / `network:*` / `tool:*` / `operations:*` 模块——Phase 61 仅修复资源权限矩阵的「接入」，不做跨模块推广。
- **D-03:** **资源/操作不在 map → 拒绝（403）**。`RequireAPIKeyResourcePermission` 查 map 命中 → 转换为 `PermissionCode` 走 scope check；未命中 → `response.Error(c, ErrForbidden, "资源权限未定义")` + `c.Abort()`。**不**默认放行（避免设计期未考虑的资源被漏检）；**不**默认 deny-all（避免后续添加新 resource 时静默拒绝合法请求——planner 在 add new resource 时必须显式补 map entry，编译期漏报错即回归到本 phase 讨论）。
- **D-04:** **action 词汇集对齐 `pkg/permission/config.go` 既有 PermissionCode 末段**：`list` / `view` / `add` / `edit` / `remove` / `export` / `import` / `query` / `write` / `execute` / `clean` / `start` / `stop` / `run` / `restore` / `diff` / `resetPwd` / `forceLogout`。`remove` 与 `delete` 同义（`remove` 是 system:* 用法、`delete` 是 network:* 用法），本 phase map 仅含 `remove`（system 模块约定）。
- **D-05:** **`RequireAPIKeyResourcePermission` 调用点不挂载本 phase**。Phase 60 D-01 已挂载的中间件链 `RequirePermissions([]string{"system:apikey:*"}) → MultiAuth → RateLimitByScope` 中，资源权限校验由 `RequirePermissions` 覆盖（`*` 通配全部子权限）。`RequireAPIKeyResourcePermission` 仍保留为公共 helper，**本 phase 仅做单元测试覆盖 + 文档化**，**不**修改 `apikey_router.go` 调用形态（最小爆炸半径，与 Phase 60 D-02「仅 system/apikeys/* 挂载」一致）。

### InheritPerms 资源校验语义（AUTH-04）

- **D-06:** **`InheritPerms=true` → 实时加载 User 权限代码列表 + 与 API Key scopes 取并集**。`MultiAuth`（`apikey.go:155` `setUserContextForAPIKey`）在 `InheritPerms=true && apiKey.User != nil` 时调用 `permission.Service.GetUserPermissions(db, userID)` 加载 `[]string` 权限代码（如 `["system:user:list", "system:role:list", ...]`），与 API Key 自带 `scopes`（如 `["read", "write"]`）取**并集**写入 `c.Set("scopes", mergedScopes)`。**注意**:scope 是粗粒度（read/write/admin），permission code 是细粒度（system:user:list），两者语义不同但都进入 `c.scopes` 同一集合，供 `RequireScope`/`getScopeFromContext` 检。
- **D-07:** **每请求一次 DB 查询**（不缓存）。Phase 61 不引入 User 权限缓存层，理由：① API Key 认证是 admin CRUD 管理面，请求频率极低（远低于 user login 热路径），DB 查询成本可接受；② 缓存层增加 invalidation 复杂度（角色菜单调整 → 缓存失效），与 Phase 61 聚焦「接入而非性能优化」的 scope 一致。**未来若 auth/login 也接 API Key 认证**，需独立 phase 加 Redis 缓存（属 FUTURE-APIKEY-04 范畴）。
- **D-08:** **`InheritPerms=false`（默认）行为不变**。`MultiAuth` 不加载 User 权限，`c.scopes` 仅含 API Key 自带 scopes。`RequireScope` / `RequireAPIKeyResourcePermission` 仅检 API Key 自带 scopes。
- **D-09:** **User 权限加载失败处理 = 拒绝（401）+ 不静默放行**。DB 查询失败 / `userID` 为空（`apiKey.UserID == nil`）/`GetUserPermissions` 返回 error → `response.Error(c, ErrUnauthorized, "用户权限加载失败")` + `c.Abort()`。理由：继承权限是 admin 显式开启的高级能力，加载失败意味着「无法判断是否真有权限」，fail-closed 是唯一安全选择。`InheritPerms=true && apiKey.UserID == nil` 视为配置错误（同 fail-closed）。
- **D-10:** **username 语义修正**（Phase 60 deferred → 本 phase 落地）。`c.Set("username", apiKey.Name)` 改为 `c.Set("username", apiKey.User.Username)`（关联加载 User）。`apiKey.User` 已在 `MultiAuth` 认证阶段 Eager-load（`apikey.go:171` 已有 `apiKey.User != nil` 检查）。**注意**:`User` 关联需 `apikey_service.go` `ValidateAPIKey`/`GetAPIKey` 用 `Preload("User")` 确保返回结构含 User 字段（planner 验证调用链）。`nickname` 同样取 `apiKey.User.Nickname`（Phase 57 D-04 留的 nickname="" 占位清理）。

### 多 scope 选择策略（QUAL-03 核心）

- **D-11:** **`RateLimitByScope` 接口新增 `action` 参数**。原签名 `RateLimitByScope(rateLimiter *services.RateLimiter) gin.HandlerFunc` 改为 `RateLimitByScope(rateLimiter, action string) gin.HandlerFunc`，注册期计算 `requiredScope := getRequiredScope(action)`（复用 `apikey.go:229` 既有 `getRequiredScope` 函数：`view → read`、`create/edit/delete → write`，本 phase 扩展 `list → read`），闭包捕获。
- **D-12:** **多 scope 选择逻辑改为 action-aware**。`getScopeFromContext` 改造为 `getScopeFromContext(c *gin.Context, action string) string`：
  - 从 `action` 推 `requiredScope`（与 D-11 一致：`view/list → read`、`create/edit/delete → write`）；
  - 检 `c.scopes` 是否含 `requiredScope`：
    - 含 → 用 `requiredScope` 做限流；
    - 含 `admin` → 用 `admin` 做限流（admin 最高限额）；
    - 都不含 → **拒绝（403 "权限作用域不足"）+ 不走限流**（与 Phase 60 D-13 留白的多 scope 选择逻辑解耦，承接落地）；
  - `inherit_perms=true` 行为保留：仍走 `default` 限额（不对 scopes 做 scope check，因 User 权限已在 MultiAuth 加载合并入 scopes）。**注意**:InheritPerms=true 时 scopes 已含 User 权限代码（细粒度 system:user:list），不包含粗粒度 read/write/admin，**与 action 推 requiredScope 的匹配会失败**。**解**:InheritPerms=true 时跳过 action-based check，固定走 `default` 限额（D-12 末尾分支保留既有 `inherit_perms` 短路）。
- **D-13:** **API Key scopes 仅含细粒度 permission code（InheritPerms=true）时**，`getScopeFromContext` 不应期望 scopes 含 read/write/admin——D-12 已通过 `inherit_perms` 短路处理。`InheritPerms=false` 时要求 API Key 自带 scopes 必含粗粒度 read/write/admin（创建校验已保证，见 `apikey_service.go:104-114` `validateScopes`）。
- **D-14:** **`RateLimitByScope` 与 `RequireScope` 的 scope 校验职责划分**：`RequireScope` 在前做硬性 scope check（必须有 read/write/admin），`RateLimitByScope` 在后做精细限流（action-aware 选择限额档位）。**两个中间件都保留**（不合并），各自职责清晰：`RequireScope` = 鉴权、`RateLimitByScope` = 限流。`RequireScope` 不感知 action，`RateLimitByScope` 感知 action。

### 限流阈值配置化（QUAL-03 调优）

- **D-15:** **复用 `CacheConfigService` 模式**：在同一 `cache_config_service.go` 中新增 `rate_limit.*` 前缀配置项（与 `cache.*` 并存），启动时一次性加载到内存 map，运行时通过既有 `POST /monitor/cache/reload` 手动触发 `ReloadConfig(ctx)` 刷新。**不**新增独立 `RateLimitConfigService`（避免服务碎片化），**不**新增定时刷新 goroutine（与 CacheConfigService 现有架构保持一致）。
- **D-16:** **配置键定义**（12 个键，每个 scope 3 个时间窗）：
  ```
  rate_limit.read.per_minute   (默认 30,  Min 1,   Max 10000)
  rate_limit.read.per_hour     (默认 500, Min 1,   Max 100000)
  rate_limit.read.per_day      (默认 5000,Min 1,   Max 1000000)
  rate_limit.write.per_minute  (默认 100, Min 1,   Max 10000)
  rate_limit.write.per_hour    (默认 1500,Min 1,   Max 100000)
  rate_limit.write.per_day     (默认 15000,Min 1,  Max 1000000)
  rate_limit.admin.per_minute  (默认 200, Min 1,   Max 10000)
  rate_limit.admin.per_hour    (默认 5000,Min 1,   Max 100000)
  rate_limit.admin.per_day     (默认 50000,Min 1,  Max 1000000)
  rate_limit.default.per_minute  (默认 120,Min 1,  Max 10000)  // InheritPerms=true / 无 scope 时
  rate_limit.default.per_hour    (默认 2000,Min 1, Max 100000)
  rate_limit.default.per_day     (默认 20000,Min 1, Max 1000000)
  ```
  **不**预留 `per_api_key` per-key override 配置（DB 表 per-key 限额超 scope，留 FUTURE-APIKEY-04）。
- **D-17:** **默认值与既有 `rate_limiter.go:48-50` 硬编码一致**（read=30/500/5000、write=100/1500/15000、admin=200/5000/50000）。`default` scope 默认值（120/2000/20000）与 `rate_limiter.go:64` 既有 fallback 一致。
- **D-18:** **`RateLimiter` 改造**：移除硬编码 `limits map[string]RateLimit`，改为构造时接收 `RateLimitConfigService`（或 `CacheConfigService` 引用），`Check(key, scope)` 内部从 service 读 `scope.per_minute/per_hour/per_day`，miss 时降级到 `default` 配置。`RateLimiter` 结构体新增 `config *CacheConfigService`（或单独的 `RateLimitProvider` interface，便于未来替换实现）。
- **D-19:** **配置刷新与运行中限流一致性**：reload 后新阈值**仅对 reload 之后的新请求生效**（在途请求的滑动窗口保留旧阈值记录）。这是合理的——阈值变更不影响既有窗口计数，仅影响后续请求的限额判断。

### 测试覆盖

- **D-20:** **三层测试策略**：
  - **单元测试**（`pkg/permission/resource_action_map_test.go`）：覆盖 map lookup、unmapped resource → error。
  - **中间件单元测试**（`internal/middleware/apikey_test.go` 扩展）：`RequireAPIKeyResourcePermission` 接受各种 resource/action 组合 → 走 RequireScope 路径。
  - **集成测试**（`internal/middleware/apikey_integration_test.go` 扩展，沿用 Phase 57 D-01 fake UsageLogger + Phase 59 D-03 sqlite in-memory 模式）：覆盖 ① `MultiAuth` + `InheritPerms=true` 加载 User 权限 ② `RateLimitByScope` action-aware 多 scope 选择 ③ reload 后阈值生效。
- **D-21:** **测试不引入 gomock**（与 `.planning/codebase/TESTING.md` 约定一致）。User permission 加载用真实 `permission.Service` + 真实 sqlite DB；`RateLimitByScope` 限流用真实 `RateLimiter` + 真实 CacheConfigService 实例化。

### Claude's Discretion

- **`resource_action_map.go` 命名风格**：用 `map[resource]map[action]PermissionCode` 嵌套 map 还是 `map[string]PermissionCode` 扁平 + key `"resource:action"` ——planner 按可读性选。
- **`RateLimitByScope` `action` 参数校验**：未识别 action（不在 `getRequiredScope` map）→ 走 default read 或 fail-closed 拒绝——planner 选（建议 fail-closed，与 D-12 一致）。
- **`getScopeFromContext` 私有/导出**：是否保留为私有函数（既有）——保留。
- **`User` 关联 Preload 注入点**：`ValidateAPIKey` / `GetAPIKey` / `MultiAuth` 三处调用方——planner 在哪处加 Preload("User") 由可测性选，但**至少**确保 `MultiAuth` 调用路径含 User。
- **`CacheConfigService` 改造 vs 新增 method**：在同一文件新增 `rate_limit.*` 配置项注册（不动 cache.* 既有逻辑），还是抽 `RateLimitProvider` interface ——planner 按代码组织偏好选。

### Folded Todos

（无——`cross_reference_todos` 未命中任何 todo）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与规划
- `.planning/ROADMAP.md` §Phase 61 — Goal / Depends on（Phase 60 AUTH-03=启用）/ Requirements（AUTH-04 + QUAL-03）/ Success Criteria
- `.planning/REQUIREMENTS.md` — AUTH-04（ex-FUTURE-APIKEY-01）+ QUAL-03（ex-FUTURE-APIKEY-02）定义
- `.planning/STATE.md` §v1.21 milestone 状态 — Phase 60 AUTH-03=启用已 locked，Phase 61 条件已满足

### 上游 phase 锁定决策（不可偏离）
- `.planning/phases/57-auth-chain-core-fix-regression-test/57-CONTEXT.md` — D-04 保留全部 7 context 键（user_id/username/nickname/api_key_id/scopes/auth_type/inherit_perms）
- `.planning/phases/58-route-contract-alignment/58-CONTEXT.md` — D-03 字段命名约定（camelCase 全项目约定）
- `.planning/phases/59-observability-usage-log-fix/59-CONTEXT.md` — D-02 detached context 契约 + D-03 sqlite 测试模式 + D-04 失败用 pkg/logger
- `.planning/phases/60-security-hardening-and-enable-decision/60-CONTEXT.md` — D-01 挂载链 + D-02 仅 system/apikeys/* + D-04 IP 白名单严格拒绝 + D-05 SM3 单向哈希 + D-13 getScopeFromContext 与 resource 接入留 Phase 61

### 待改造源码（核心）
- `internal/middleware/apikey.go`:
  - line 24 `MultiAuth`（D-06 InheritPerms 加载入口、D-09 失败处理、D-10 username 修正）
  - line 155-175 `setUserContextForAPIKey`（D-06 合并 scopes、D-10 取 User.Username）
  - line 177-215 `RequireScope`（D-14 保留职责）
  - line 217-225 `RequireAPIKeyResourcePermission`（D-01/D-05 静态 map helper，本 phase 不挂载到 apikey_router）
  - line 222-243 `getRequiredScope`（D-11 扩展：list→read）
  - line 245-285 `RateLimitByScope`（D-11 新增 action 参数 + D-18 配置化）
  - line 287-308 `getScopeFromContext`（D-12 改 action-aware 多 scope 选择 + inherit_perms 短路保留）
- `internal/services/rate_limiter.go`:
  - line 31-53 `RateLimiter` 结构 + `NewRateLimiter`（D-18 移除硬编码、改从 config service 读）
  - line 59-131 `Check`（D-18 内部从 config 读限额）
  - line 64 default fallback（D-17 默认值 120/2000/20000）
- `pkg/permission/config.go` — D-02 既有 `PermissionCode` 常量（system:user:list 等）作为 map 值的对齐目标
- `pkg/permission/service.go` — D-06 `Service.GetUserPermissions(db, userID)` 实时加载 User 权限代码
- `internal/services/cache_config_service.go` — D-15 复用既有模式，新增 rate_limit.* 配置项
- `internal/api/v1/system/apikey_router.go` — D-05 本 phase 不修改（路由调用形态不变）
- `internal/services/system/apikey_service.go`:
  - `ValidateAPIKey` / `GetAPIKey` 需确保 User 关联 Preload（D-10 Claude's Discretion）
  - line 104-114 `validateScopes`（既有 scope 校验，D-13 沿用）
  - line 103-114 scopes 词汇集（read/write/admin 三档，D-13 既有保证）

### 既有测试基建
- `.planning/codebase/TESTING.md` — 无 gomock、需 DB 用真实连接（D-21 依据）
- `internal/middleware/apikey_test.go` — 既有 3 个纯函数测试（isValidKeyFormat/isIPAllowed/getRequiredScope），D-20 扩展点
- `internal/middleware/apikey_integration_test.go` — Phase 57 集成测试基线（fake UsageLogger + httptest），D-20 复用
- Phase 59 sqlite in-memory 模式 — D-21 集成测试可复用
- `pkg/permission/config_v1201_test.go` — D-20 既有 permission code 测试

### 代码库先例
- `CacheConfigService` 模式（D-15 复用依据）:
  - line 92-104 `NewCacheConfigService` 启动加载
  - line 107-185 `LoadConfigs` 从 sys_config 读 + 默认值修复
  - line 285-289 `ReloadConfig` 手动刷新
  - line 304+ `ConfigInfo{Min, Max, Default}` 范围校验
- `pkg/permission/service.go:272-286` `GetUserPermissions`（D-06 复用实现，User 权限加载）
- `apikey.go:171` `apiKey.User != nil` 既有 Eager-load 守卫（D-10 复用基础）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/permission/config.go` `PermissionCode` 常量（D-02 对齐目标，无需新增权限代码常量）
- `pkg/permission/service.go` `Service.GetUserPermissions(db, userID)` （D-06 实时加载 User 权限代码，可直接调用）
- `internal/services/cache_config_service.go` 既有 `CacheConfigService` 模式（D-15 复用，新增 rate_limit.* 节点）
- `internal/middleware/apikey.go:229` `getRequiredScope(action)` 既有（D-11 扩展 list→read）
- `pkg/errors` `apperrors.CodeParamError` / `response.ErrForbidden` / `response.ErrUnauthorized`（D-03 拒绝响应复用）
- `pkg/logger`（Phase 59 D-04 失败日志惯例）
- sqlite GORM（Phase 59 D-03 测试模式，D-21 复用）

### Established Patterns
- `module:resource:action` 三段式权限代码（pkg/permission/config.go 既有约定，本 phase 严格沿用）
- `RequireScope` 层级 admin > write > read（apikey.go:198-205 admin 含所有权限）
- `CacheConfigService` 启动加载 + 内存 map + 手动 ReloadConfig 模式（D-15 复用）
- `config_backup_service.go:247` applogger.Errorf 失败日志惯例（Phase 59 D-04）
- v1.19 batch `context.WithTimeout(context.Background(), ...)` detached context（Phase 59 D-02）

### Integration Points
- `MultiAuth`（apikey.go:24）— D-06 InheritPerms 加载 User 权限入口 + D-10 username 修正点
- `setUserContextForAPIKey`（apikey.go:155-175）— D-06 scopes 合并写入 + D-10 User 关联读取
- `RateLimitByScope`（apikey.go:245）— D-11 接口签名变 + D-12 action-aware 选 scope
- `getScopeFromContext`（apikey.go:287）— D-12 多 scope 选择逻辑改造
- `RateLimiter.Check`（rate_limiter.go:59）— D-18 配置化限额读取
- `CacheConfigService.LoadConfigs`（cache_config_service.go:107）— D-15 新增 rate_limit.* 注册
- `PermissionCode` 常量表（pkg/permission/config.go）— D-02 静态 map 值的目标

</code_context>

<specifics>
## Specific Ideas

- **用户取向：API Key 系统要「真正可用 + 真正安全」（承自 Phase 60 specifics）**。本 phase 从「功能完整」推到「语义严谨」：限流不再是任意 scope、权限校验不再是死代码（resource 接入）、InheritPerms 不再是空 flag。
- **用户决策偏离推荐**：
  - D-06 选「实时加载 + scopes 与 permission code 取并集」（我推荐「独立 context 字段 user_permissions 隔离两种语义」）——理由是「API Key scopes 已在 Phase 60 既有契约，引入新字段风险更大」+「并集语义更直观」。
  - D-12 选「无 fallback：scopes 不含所需 scope 则拒绝」（我推荐「降级为 default 限额」）——理由是「限流拒绝比静默放行更安全，admin 可创建新 key 修复」。
  - D-15 选「复用 CacheConfigService 模式」（我推荐「新建 RateLimitConfigService」）——理由是「与服务碎片化相反，单一 config service 维护成本低」。
- **用户偏好最小爆炸半径**（Phase 60 D-02 一贯）：
  - D-05 不修改 `apikey_router.go` 调用形态（`RequireAPIKeyResourcePermission` 仅做单元测试 + 文档化）
  - D-07 不引入 Redis 缓存（每请求 DB 查询可接受）
  - D-15 不新增独立 service（复用 CacheConfigService）
- **用户取向：fail-closed 而非 fail-open**（D-09 拒绝、D-12 无 fallback）——「不确定就拒绝」是用户对 admin 系统的安全偏好。
- **`pkg/permission/config.go` PermissionCode 既有的 `remove` 与 `delete` 混用**（system:* 用 remove，network:* 用 delete）——本 phase map 仅用 remove（system 模块约定，planner 需对照既有 constant 决定）。

</specifics>

<deferred>
## Deferred Ideas

- **User 权限加载加 Redis 缓存（TTL ~5min）**：D-07 留白，理由：API Key 认证是 admin CRUD 管理面，请求频率低，DB 查询可接受。**未来若 API Key 认证扩展到外部 API 面**，需独立 phase 加 Redis 缓存层（属 FUTURE-APIKEY-04 范畴）。
- **per-API-Key 限流 override（DB 表）**：D-16 留白，统一阈值。FUTURE-APIKEY-04 范畴。
- **`RequireAPIKeyResourcePermission` 挂载到生产路由**：D-05 本 phase 不挂载（仅单元测试 + 文档化）。Phase 60 D-02 挂载形态已覆盖 `system:apikey:*` 全权限，**未来需细粒度权限校验时**独立 phase 推广（跨模块 scope 风险大，需独立评估）。
- **行级数据权限**（按 dept_id 过滤）：本 phase 仅做模块级 `system:resource:action` 校验，不做行级实例权限。`Role.DataScope` 既有字段（service.go:53）已有 `DataScopeAll` 等枚举，本 phase 不引入新逻辑。
- **限流定时刷新（后台 goroutine）**：D-15 留白，与 CacheConfigService 现有架构保持一致。定时刷新需求可独立 phase 加。
- **限流计数器持久化**：当前 `RateLimiter.windows`（rate_limiter.go:32）是 in-memory sync.Map，进程重启清零。多实例部署下每个实例独立计数——超出 Phase 61 scope，留 FUTURE-APIKEY-04。
- **密钥轮换/吊销、配额告警**：FUTURE-APIKEY-03/04（仍 v2 Future，未升级 v1）。

</deferred>

---

*Phase: 61-资源权限矩阵与限流生产调优*
*Context gathered: 2026-08-13*