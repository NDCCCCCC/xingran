---
gsd_state_version: 1.0
milestone: v1.21
milestone_name: Milestone History
status: milestone_complete
stopped_at: Milestone complete — Phases 57-61 all code-complete (Phase 58 SC#1-SC#4 E2E deferred: dev DB perf)
last_updated: 2026-08-13T14:30:00.000Z
last_activity: 2026-08-14 — quick-260814-gor: assignAllMenusToAdmin 改增量幂等差集补全(消除先删后插丢权限炸弹)
progress:
  total_phases: 5
  completed_phases: 5
  total_plans: 8
  completed_plans: 8
  percent: 100
---

# Project State

**Project**: XingRan-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.21 API Key 认证链修复 + 能力补全 milestone COMPLETE — Phases 57-61 全部代码完成。Phase 58 SC#1-SC#4 端到端验证因 dev DB(Supabase pooler)性能延期(见 58-01-SUMMARY §Deferred),代码契约修复已提交且自动化门全绿。
**Last activity**: 2026-08-13 — Phase 58 标记 code-complete + 58-01-SUMMARY;本次会话另修后端启动 hang(config.go keepalive `7c821d7`)、dev 环境 SM2 登录链路(config.yaml use_sm2)、MV/分区补建(dbprobe `26e93fd`)、logger 镜像修复(`7901248`)、WIP 清理(dev 工具 + rebrand)。2026-08-14 quick-260814-164: 修复 RPA Worker 注册主键 NULL + 菜单接口 N+1。同日 quick-260814-211: 修复 workstation/list uuid=text 类型错误 + dept.leader 非 uuid 查询防御。

## Project Reference

See: [.planning/PROJECT.md](PROJECT.md) (updated 2026-08-12)

**Core value**: 端到端运维可观测与审计能力——每个写操作产生可追溯记录(who/when/what/from-where/before-after-state),敏感字段自动脱敏。API Key 作为 JWT 之外的第二条认证通道,其认证链、作用域校验、使用日志必须真实生效且可观测;Phase 61 落地资源级权限矩阵与限流生产调优。

**Current focus**: v1.21 milestone COMPLETE(5/5 phases, 8/8 plans)。唯一遗留:Phase 58 SC#1-SC#4 端到端验证延期(待更快 dev DB)。

## Current Position

Phase: 61(milestone 末 phase)—— 全部 57-61 代码完成
Plan: 8/8 complete
Status: Milestone complete
Last activity: 2026-08-13

Progress: [██████████] 100%

## Accumulated Context

### v1.21 Milestone — Critical Decisions (locked at init)

- **Scope**: 全修复 + 就绪 + 能力补全 — 修复全部 P0/P1/P2 确定性缺陷,MultiAuth 代码修好并已挂载;Phase 61 落地 AUTH-04 资源级权限矩阵与 QUAL-03 限流生产调优
- **Regression nature**: 对 v1.6「API 密钥管理系统」(Phase 16 / 2026-05-19 / 10 plans) 的回归修复,非新功能
- **Research skipped**: 回归修复场景,代码与问题均已调查清楚(`.planning/research/` 不存在)
- **Phase numbering**: 从 v1.20 末尾 Phase 56 续编(57-61)
- **Granularity**: standard(项目配置),5 phases (57-61) 为本 milestone 自然交付边界;Phase 61 为 2026-08-12 重规划新增(能力补全,conditional)
- **Scope evolution (2026-08-12 re-plan)**: 原"全修复 + 就绪"扩展为"全修复 + 就绪 + 能力补全"——FUTURE-APIKEY-01/02 升级为 v1 AUTH-04/QUAL-03 归 Phase 61(资源级权限矩阵 + 限流生产调优),仅在 Phase 60 AUTH-03=启用 后执行

### Phase 61 Plan 01 Locked Decisions (AUTH-04)

- **D-02**: 资源权限矩阵仅覆盖 `system:*` 11 资源(user/role/menu/dept/post/workstation/dict/config/captchaBackground/notice/apikey),共 59 个 (resource, action) 组合;`monitor:*` / `network:*` / `tool:*` / `operations:*` 不纳入
- **D-03**: 资源/操作未命中 map → 403 fail-closed,新增 resource 必须显式补 entry
- **D-05**: `RequireAPIKeyResourcePermission` 本 phase 仍为公共 helper,不挂载到 `apikey_router.go`
- **D-06**: `InheritPerms=true` 实时加载 User 权限代码,与 API Key scopes 取并集写入 `c.scopes`
- **D-07**: 每请求一次 DB 查询加载 User 权限,不引入缓存
- **D-08**: `InheritPerms=false` 行为不变,仅校验 API Key 自带 scopes
- **D-09**: User 权限加载失败(DB error / UserID nil / service error) → 401 fail-closed
- **D-10**: `c.Set("username", apiKey.User.Username)` + `c.Set("nickname", apiKey.User.Nickname)`(ValidateAPIKey 已 `Preload("User")`)
- **D-20/D-21**: 三层测试(单元 + 中间件单元 + 集成),无 gomock,真实 `permission.Service` + sqlite in-memory

### Phase 61 Plan 02 Locked Decisions (QUAL-03)

- **D-11**: `RateLimitByScope(rateLimiter, action string)` 新增 action 参数,注册期 `requiredScope := getRequiredScope(action)` 闭包捕获;`getRequiredScope` 扩展 `list → read`
- **D-12**: 多 scope 选择 action-aware 严格语义 — 精确匹配 requiredScope → admin 覆盖 → 403 fail-closed(无 fallback);提取纯函数 `SelectScope(scopes, inheritPerms, action)` 直接可单测,不新增 context 中转键
- **D-13**: `InheritPerms=true` 短路走 default 限额(细粒度 permission code 不参与 action 匹配)
- **D-14**: `RequireScope` 硬鉴权 / `RateLimitByScope` 精细限流职责分工,两中间件保留
- **D-15/D-16/D-17**: 12 个 `rate_limit.{read|write|admin|default}.{per_minute|per_hour|per_day}` 配置键复用 CacheConfigService(独立 rateLimits map,次数语义),默认值与既有硬编码一致,Min/Max 范围校验
- **D-18**: `RateLimiter` 移除硬编码 limits map,改 `RateLimitProvider` 接口注入;`NewRateLimiter(nil)` 兜底 staticRateLimitProvider;router.go 用 `core.CacheConfigService` 字段(非 getter)
- **D-19**: reload 后新阈值仅对新请求生效,在途滑动窗口保留旧阈值

### v1.21 — 根因调查结论(ground-truth 已验证)

| 缺陷 | 文件:行 | 根因 |
|------|---------|------|
| P0-2 | `internal/middleware/apikey.go:146-179` | `setUserContextForAPIKey(apiKey interface{})` 接收 `*models.APIKey` 指针,但断言为局部值类型 `apiKeyType`,恒 false → context 从未写入 |
| P0-1 | `internal/api/router.go` | `MultiAuth` 及下游 `RequireScope` / `RequireAPIKeyResourcePermission` / `RateLimitByScope` 未挂载任何生产路由(死代码) |
| P1-1 | `internal/api/v1/system/apikey_router.go` vs `src/api/apikey.ts` | 前端 GET/PUT/DELETE 与后端 POST 路由方法不匹配 → 单条 get/update/delete 404 |
| P1-2 | `internal/middleware/apikey.go:60-75` | 使用日志 goroutine 在 `c.Next()` 前触发,LogUsageRequest 仅填 Method/Path/ClientIP,StatusCode/Duration/Success 全零 → successRate ≈ 0% |
| P2-a | `internal/middleware/apikey.go:274-275` | `string(rune(result.Limit))` 编码错误:Limit=100 → "d" 而非 "100" |
| P2-b | `internal/middleware/apikey.go:66` | 异步 goroutine 复用 `c.Request.Context()`,请求结束 → ctx.Canceled,error 在 `usage_logger.go:73` 被 `_ = err` 吞掉 |
| P2-c | `internal/services/system/apikey_service.go` + migration_085 | API Key `Key` 字段明文存储(`WHERE key = ?` 直查) |
| P3 | migration_085 | `idx_api_keys_key` 与 `uniqueIndex` 重复 |

### Phase Dependencies (v1.21)

```
Phase 57 (认证链核心修复 + 回归测试)
   │
   ├─→ Phase 58 (前后端路由契约对齐) [depends on 57]
   │
   ├─→ Phase 59 (可观测性 / 使用日志修复) [depends on 57]
   │      │
   │      └─→ Phase 60 (安全加固与启用决策) [depends on 57 + 59]
   │             │
   │             └─→ Phase 61 (资源级权限矩阵 + 限流生产调优) [depends on 60 AUTH-03=启用]
```

Phase 58 可与 Phase 59 并行(两者依赖仅 Phase 57);Phase 60 必须在 59 后(启用决策需要可观测基础);Phase 61 仅在 Phase 60 AUTH-03 决策=启用 时执行,否则 defer。

### Pending Decisions (defer to phase-internal discuss)

- **AUTH-03** (Phase 60): 是否在生产路由挂载 MultiAuth 使 X-API-Key 认证真正生效?含作用域继承、IP 白名单、与 JWT 优先级/回退关系
- **SEC-01** (Phase 60): API Key `Key` 字段是否迁移到 SM3 / argon2id 哈希存储?含平滑过渡 + 回滚方案
- **CONTRACT-01** (Phase 58): 前后端契约对齐方向——选项 A 改前端用 POST 对齐 v1.6 既有模式,选项 B 后端补 RESTful GET/PUT/DELETE

### Blockers/Concerns

None currently. Roadblock risk: Phase 60 AUTH-03 启用决策若选"启用",会触发 Phase 61(资源级权限矩阵 + 限流调优)执行——已通过 2026-08-12 重规划将该项独立为 Phase 61,避免污染 Phase 60 决策型 scope。

## Quick Tasks Completed

| Quick ID | Description | Date | Commit | Plan |
|----------|-------------|------|--------|------|
| 260812-wu5 | clean constants dead code and unify pagination constants | 2026-08-12 | 759a65a | [260812-wu5-clean-constants-dead-code-and-unify-pagi](./quick/260812-wu5-clean-constants-dead-code-and-unify-pagi/) |
| 260814-164 | 修复 RPA Worker 注册主键 NULL(23502) + 菜单接口 N+1(context canceled 500) | 2026-08-14 | f0d0a1f / 4c2a900 | [260814-164-fix-rpa-pk-menu-n1](./quick/260814-164-fix-rpa-pk-menu-n1/) |
| 260814-211 | 修复 workstation/list uuid=text 类型错误(42883) + dept.leader 非 uuid 查询防御(22P02) | 2026-08-14 | c9ab875 / 08d97ed | [260814-211-fix-workstation-list-uuid-text-cast-dept](./quick/260814-211-fix-workstation-list-uuid-text-cast-dept/) |
| 260814-ehg | 旧库菜单去重导入 dev 库(保持层级) + admin 全量授权(239 菜单/10 顶级目录) | 2026-08-14 | ef1ba87 / cb81443 / e02d837 | [260814-ehg-dedupe-and-import-legacy-menu-data-into-](./quick/260814-ehg-dedupe-and-import-legacy-menu-data-into-/) |
| 260814-gor | assignAllMenusToAdmin 先删后插→增量幂等差集补全(消除丢权限炸弹+降载) | 2026-08-14 | a0ea57b | [260814-gor-fix-assignallmenustoadmin-delete-then-re](./quick/260814-gor-fix-assignallmenustoadmin-delete-then-re/) |

## Deferred Items

Items acknowledged and carried forward from previous milestone closes (non-v1.21 work):

| Category | Item | Status | Source |
|----------|------|--------|--------|
| uat_gap | v1.20 VLAN + port_binding 12 项真机 UAT (Huawei S8700 × 6 + Ruijie RS8607E × 4 + H3C × 2 conditional) | deferred (site-visit) | v1.20 close 2026-07-10 |
| requirement | VLAN-04 / BIND-06 / UI-06 批量端口写 | deferred to FUTURE-BATCH-05 | v1.20 close |
| uat_gap | v1.19 7 项真机 SSH transport verification | deferred (site-visit) | v1.19 close 2026-07-08 |
| uat_gap | v1.18 3 项 site-visit UAT (S8700/RS8607E) | deferred (site-visit) | v1.18 close 2026-07-04 |
| tech-debt | WR-01 / WR-03 / WR-04 / WR-05 + 14 项 v1.19.x+ future work | backlog | v1.19 REQUIREMENTS §Future |
| tech-debt | ~88 audit-open historical items (19 debug_sessions / 60 quick_tasks / etc.) | backlog | pre-v1.20 |

Full deferred detail in [milestones/v1.20-ROADMAP.md](milestones/v1.20-ROADMAP.md) + [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md).

## Completed Milestones

- ✅ v1.0 工位导入部门/用户关联 (2026-04-16) — 2 phases, 7 plans
- ✅ v1.1 信息点导入设备端口关联 (2026-04-16) — 1 phase, 1 plan
- ✅ v1.2 可配置仪表盘生产级改造 (2026-04-21) — 4 phases, 11 plans
- ✅ v1.3 技术债清理 (2026-04-27) — 3 phases, 9 plans
- ✅ v1.4 MAC地址采集优化 (2026-05-09) — 1 phase, 4 plans
- ✅ v1.5 MAC地址历史数据管理 (2026-06-15) — 4 phases, 26 plans
- ✅ **v1.6 API密钥管理系统 (2026-05-19) — 1 phase, 10 plans** ← v1.21 回归对象
- ✅ v1.7 前后端加密配置同步 (2026-05-20) — 1 phase, 6 plans
- ✅ v1.8 登录端点加密增强 (2026-05-21) — 1 phase, 4 plans
- ✅ v1.9 AD域控集成扩展 (2026-05-24) — 2 phases, 11 plans
- ✅ v1.10 网络设备权限修复 (2026-05-24) — 1 phase, 1 plan
- ✅ v1.11 AD组自动同步系统 (2026-05-26) — 1 phase, 18 plans
- ✅ v1.12 深信服桌面云集成 22A/22B (2026-06-02) — 6 plans
- ✅ v1.13 资产管理模块 (2026-06-08) — 1 phase, 6 plans
- ✅ v1.14 全局列自定义 (2026-06-09) — 1 phase, 1 plan
- ✅ v1.15 工位设备关联 + 部门物理位置映射 (2026-06-10 / 06-25) — Phases 28 + 39
- ✅ v1.16 技术债清理 (2026-06-26) — Phases 40-41, 8 plans
- ✅ v1.17 资产对账 (2026-07-03) — Phases 42-47, 16 plans
- ✅ v1.18 网络设备硬件清单 (2026-07-04) — Phases 48-49, 5 plans
- ✅ v1.19 网络设备写命令 (2026-07-08) — Phases 50-55, 9 plans
- ✅ v1.20 网络设备 VLAN + 端口绑定 (2026-07-10) — Phase 56, 5 plans

## Session Continuity

Last session: 2026-08-13T09:45:30.356Z
Stopped at: Completed 61-01-PLAN.md
Resume file: None

**Milestone status:** v1.21 IN PROGRESS — **Phase 61 Plan 01 COMPLETE** (commits c55a3c5 + cba12ce + 1eae873 + 3a71c11): Task 1 创建 `pkg/permission/resource_action_map.go` 静态 map 覆盖 system:* 11 资源 59 组合 + 单元测试; Task 2 改造 `MultiAuth`/`setUserContextForAPIKey`/`RequireAPIKeyResourcePermission` + `internal/api/router.go` 调用形态,实现 D-06/D-07/D-09/D-10; Task 3 新增 8 个 `RequireAPIKeyResourcePermission` 中间件单测 + 5 个 `InheritPerms` sqlite 集成测。`go build ./...` exit 0,核心包测试 PASS,既有 Phase 57/59/60 回归锚 PASS。

## Operator Next Steps

- `/gsd:execute-phase 61` — continue with Phase 61 Plan 02 (QUAL-03: RateLimitByScope action-aware tuning + rate_limit.* config + RateLimiter config-driven)
