---
gsd_state_version: 1.0
milestone: v1.21
milestone_name: API Key 认证链修复
status: planning
last_updated: "2026-08-12T16:00:00.000Z"
last_activity: 2026-08-12
progress:
  total_phases: 4
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

**Project**: XingRan-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.21 API Key 认证链修复 milestone INITIATED — ROADMAP drafted (Phases 57-60), ready for Phase 57 planning
**Last activity**: 2026-08-12 — ROADMAP.md / REQUIREMENTS.md traceability / STATE.md synchronized for v1.21

## Project Reference

See: [.planning/PROJECT.md](PROJECT.md) (updated 2026-08-12)

**Core value**: 端到端运维可观测与审计能力——每个写操作产生可追溯记录(who/when/what/from-where/before-after-state),敏感字段自动脱敏。API Key 作为 JWT 之外的第二条认证通道,其认证链、作用域校验、使用日志必须真实生效且可观测。

**Current focus**: v1.21 Phase 57 — 认证链核心修复 + 回归测试

## Current Position

Phase: 57 of 60 (认证链核心修复 + 回归测试) — ready to plan
Plan: — (TBD, plan-phase not yet run)
Status: ROADMAP ready, awaiting `/gsd:plan-phase 57`
Last activity: 2026-08-12 — v1.21 ROADMAP drafted (4 phases / 11 requirements / 100% coverage)

Progress: [░░░░░░░░░░] 0% (0/4 phases, 0 plans)

## Accumulated Context

### v1.21 Milestone — Critical Decisions (locked at init)

- **Scope**: 全修复 + 就绪 — 修复全部 P0/P1/P2 确定性缺陷,MultiAuth 代码修好可接入;「是否在生产路由挂载 MultiAuth」作为 Phase 60 discuss 决策点(含安全影响评估)
- **Regression nature**: 对 v1.6「API 密钥管理系统」(Phase 16 / 2026-05-19 / 10 plans) 的回归修复,非新功能
- **Research skipped**: 回归修复场景,代码与问题均已调查清楚(`.planning/research/` 不存在)
- **Phase numbering**: 从 v1.20 末尾 Phase 56 续编(57-60)
- **Granularity**: standard(项目配置),4 phases 为本 milestone 自然交付边界

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
```

Phase 58 可与 Phase 59 并行(两者依赖仅 Phase 57);Phase 60 必须最后(启用决策需要可观测基础)。

### Pending Decisions (defer to phase-internal discuss)

- **AUTH-03** (Phase 60): 是否在生产路由挂载 MultiAuth 使 X-API-Key 认证真正生效?含作用域继承、IP 白名单、与 JWT 优先级/回退关系
- **SEC-01** (Phase 60): API Key `Key` 字段是否迁移到 SM3 / argon2id 哈希存储?含平滑过渡 + 回滚方案
- **CONTRACT-01** (Phase 58): 前后端契约对齐方向——选项 A 改前端用 POST 对齐 v1.6 既有模式,选项 B 后端补 RESTful GET/PUT/DELETE

### Blockers/Concerns

None currently. Roadblock risk: Phase 60 AUTH-03 启用决策若选"启用"会扩大 phase scope(需在 router.go 实际挂载并补权限矩阵测试)——已通过 discuss-mode 控制范围。

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

Last session: 2026-08-12
Stopped at: v1.21 ROADMAP drafted (Phases 57-60), REQUIREMENTS.md traceability confirmed, STATE.md updated
Resume file: None

**Milestone status:** v1.21 INITIATED — ROADMAP ready, awaiting `/gsd:plan-phase 57` to begin Phase 57 (认证链核心修复 + 回归测试).

## Operator Next Steps

- `/gsd:plan-phase 57` — decompose Phase 57 into executable plans (AUTH-01 + AUTH-02 + QUAL-02)
- After Phase 57 plans complete: Phase 58 (CONTRACT-01) and Phase 59 (OBSERV-01/02/03) can be planned in parallel
- Phase 60 (AUTH-03 + SEC-01/02 + QUAL-01) must come last — discuss decisions need Phase 57 + 59 foundation
