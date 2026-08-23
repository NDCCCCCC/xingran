# Phase 61: 资源权限矩阵与限流生产调优 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-13
**Phase:** 61-resource-permission-matrix-and-rate-limit-tuning
**Areas discussed:** 资源→权限映射形态, InheritPerms 资源校验语义, 多 scope 选择策略, 限流阈值配置化与调优

---

## Area 1: 资源→权限映射形态 (AUTH-04)

| Option | Description | Selected |
|--------|-------------|----------|
| 静态 map（编译期常量） | 新增 resource_action_map.go,预定义 {resource, action} → "system:resource:action" 映射表 | ✓ |
| 动态拼接（resource=模块名,action=操作） | 拼接 "system:" + resource + ":" + action,无需预定义 | |
| DB 驱动（sys_api_key_resource_permissions 表） | 新建表存 (api_key_id, resource, action, allowed) | |

**User's choice:** 静态 map（编译期常量）
**Notes:** 用户偏好「编译期检查、零运行时开销、文档化」，与 Phase 60 D-05 SEC-01「不写 Go migration」取向一致。

| Option | Description | Selected |
|--------|-------------|----------|
| 仅 system:* 模块 + 资源不存在时拒绝 | Map 仅涵盖 system:* 资源,resource 不在 map → 中间件拒绝 | ✓ |
| 仅 system:* + 资源不存在时返回 200（默认 pass） | 同上,但 resource 不在 map → 中间件跳过 | |
| 全模块可纳入 + 资源不存在时拒绝 | Map 涵盖所有 module:resource:action | |

**User's choice:** 仅 system:* 模块 + 资源不存在时拒绝
**Notes:** 用户偏好「范围限定、零风险面」，与 Phase 60 D-02「仅 system/apikeys/* 挂载」取向一致。

---

## Area 2: InheritPerms 资源校验语义

| Option | Description | Selected |
|--------|-------------|----------|
| 实时加载+合并 User 权限到上下文（每次请求 DB 查询） | RequireScope/RequireAPIKeyResourcePermission 调用 GetUserPermissions 实时加载 | ✓ |
| MultiAuth 内加载+写 context,后续中间件读 context | MultiAuth 检测 InheritPerms=true 时一次性加载,合并入 gin context "permissions" 键 | |
| 缓存 User 权限（Redis 5分钟 TTL）+ context 携带 union | CacheProvider Redis 5min TTL,key=apikey_inherit:{userID} | |
| 完全取消 InheritPerms 特性 | InheritPerms 保持当前实际行为 | |

**User's choice:** 实时加载+合并 User 权限到上下文（每次请求 DB 查询）
**Notes:** API Key 认证是 admin CRUD 管理面,请求频率极低,DB 查询成本可接受。缓存层留给 FUTURE-APIKEY-04。

| Option | Description | Selected |
|--------|-------------|----------|
| 取并集：API Key scopes ∪ User 权限代码 | context.scopes = apiKey.scopes ∪ userPermissions | ✓ |
| 权限代码上下文,不动 scopes,RequireScope 读 permissions | 不修改 context.scopes,另存 context.user_permissions | |
| API Key scopes 作 scope,User 权限作 permission code,两路独立 | 保留 current scopes,新增 user_permissions | |

**User's choice:** 取并集：API Key scopes ∪ User 权限代码
**Notes:** 用户偏好「并集语义更直观」「API Key scopes 已在 Phase 60 既有契约,引入新字段风险更大」。

---

## Area 3: 多 scope 选择策略 (QUAL-03 核心)

| Option | Description | Selected |
|--------|-------------|----------|
| 最宽松优先 (admin>write>read) | 从 scopes 中选权限等级最高的 | |
| 按 action 选择 (action→scope后检匹配) | RateLimitByScope 接收 action 参数,从 action 推 required scope | ✓ |
| 聚合 (任一 scope 通过即放行) | RateLimitByScope 遍历 scopes,各 scope 独立检 | |
| 严格匹配 (action 推 scope 必须包含) | scopes 必须严格包含该 scope (or admin) | |

**User's choice:** 按 action 选择 (action→scope后检匹配)
**Notes:** 用户偏好「贴合操作语义,限流粒度准」,承认需全面改动调用点。

| Option | Description | Selected |
|--------|-------------|----------|
| 无 fallback: scopes 不含所需 scope → 拒绝 | action 推 required scope,scopes 不包含 → 拒绝(401/403) | ✓ |
| Fallback: 不含所需 scope 则降为读最宽松 | scopes 不包含时降为 scopes 中最高限额 | |
| Fallback: 不含所需 scope 则 default | scopes 不含所需 scope 且不含 admin → 走 default scope | |

**User's choice:** 无 fallback: scopes 不含所需 scope → 拒绝
**Notes:** 用户偏好「fail-closed」「限流拒绝比静默放行更安全,admin 可创建新 key 修复」。

---

## Area 4: 限流阈值配置化与调优

| Option | Description | Selected |
|--------|-------------|----------|
| 保持硬编码（不变） | rate_limiter.go 硬编码,零额外改动 | |
| 移到 config.yaml,默认值=当前硬编码值 | configs/config.yaml 新增 rate_limit: 节点 | |
| config.yaml + sys_api_key_rate_limits 表(per-key override) | config.yaml 默认 + DB 新表 per-key 覆盖 | |
| Redis-backed 运行时配置 (热加载) | NewRateLimiter 从 Redis 读 rate_limit:scope:* key | |

**User's choice:** config提供默认值，复用参数表实现热加载 (自定义回答,非标准选项)
**Notes:** 用户明确希望「config 提供默认值,复用参数表实现热加载」。意图:既走 config.yaml 提供默认值,又走 sys_config 表提供运行时覆盖。

| Option | Description | Selected |
|--------|-------------|----------|
| 复用 CacheConfigService 模式(同服务不同前缀) | 在同一 service 中新增 rate_limit.* 配置项 | ✓ |
| 新建 RateLimitConfigService(独立服务) | 镜像 CacheConfigService 结构 | |
| 每请求查 sys_config(无缓存) | RateLimiter.Check 直接查 sys_config 表 | |

**User's choice:** 复用 CacheConfigService 模式(同服务不同前缀)
**Notes:** 用户明确「参数表本身就有缓存吧？」,认同 CacheConfigService 既有内存缓存机制。

| Option | Description | Selected |
|--------|-------------|----------|
| 跟随现有手动 reload 模式 | 沿用 POST /monitor/cache/reload 手动刷新 | ✓ |
| 本 phase 新增 30s 定时刷新 goroutine | RateLimitConfig 独立新增后台 goroutine | |
| 手动 reload + Phase 62+ 增定时刷新 | 本 phase 保持手动 reload | |

**User's choice:** 跟随现有手动 reload 模式
**Notes:** 用户偏好「零新增机制、与现有 cache TTL 调整同架构」。CacheConfigService 既有手动 reload 机制经 /monitor/cache/reload 触发,本 phase 直接复用。

---

## Claude's Discretion

- `resource_action_map.go` 嵌套 map vs 扁平 key ——planner 按可读性选
- `RateLimitByScope` action 参数校验失败处理 ——planner 按 fail-closed 建议选
- `User` 关联 Preload 注入点 ——planner 在 ValidateAPIKey/GetAPIKey/MultiAuth 三处按可测性选
- `CacheConfigService` 改造 vs 新增 method ——planner 按代码组织偏好选

## Deferred Ideas

- **User 权限加载加 Redis 缓存（TTL ~5min）** —— 留 FUTURE-APIKEY-04（API Key 认证扩展到外部 API 面时引入）
- **per-API-Key 限流 override（DB 表）** —— FUTURE-APIKEY-04
- **`RequireAPIKeyResourcePermission` 挂载到生产路由** —— 留独立 phase（跨模块 scope 风险大）
- **行级数据权限（按 dept_id 过滤）** —— 不在 Phase 61 scope（仅做模块级校验）
- **限流定时刷新（后台 goroutine）** —— 留独立 phase（与 CacheConfigService 现有架构保持一致）
- **限流计数器持久化** —— 留 FUTURE-APIKEY-04（多实例部署）
- **密钥轮换/吊销、配额告警** —— FUTURE-APIKEY-03/04（仍 v2 Future）