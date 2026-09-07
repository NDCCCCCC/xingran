# Phase 57: 认证链核心修复 + 回归测试 - Discussion Log

**Date:** 2026-08-13
**Mode:** discuss (default, interactive)
**Participant decisions:** all 4 gray areas discussed + milestone re-plan triggered

## Areas Discussed

### Area 1 — 测试替身策略 (QUAL-02)
- **Options:** 手写 fake + httptest / 引入 gomock / sqlite 内存库
- **Selected:** 手写 fake + httptest
- **Rationale:** 无 DB、无 mock 框架，符合 TESTING.md "无 gomock" 既有约定，真正端到端验证中间件链写入 gin context。

### Area 2 — 就绪可挂载证据 (AUTH-02 / SC#2)
- **Options:** 测试里实例化真构造函数 / 加 feature-flag 装配点 / 仅 grep+文档
- **Selected:** 测试里实例化真构造函数（NewUsageLogger/NewRateLimiter）
- **Rationale:** fake UsageLogger ≠ NewUsageLogger；测试即真实调用点，证明构造函数可装配，不引入生产死代码。生产装配推迟 Phase 60。

### Area 3 — AUTH-02 审查深度
- **Options:** 只修链路正确性 / 全部逻辑瑕疵一并修 / 仅记录不动代码
- **Selected (initial):** 全部逻辑瑕疵一并修 → **escalated to:** 含完整 FUTURE 功能 → **escalated to:** 重划路线图
- **Final resolution:** Phase 57 只做中间件**自洽**（修类型签名 + 内联 RequireScope()(c) 调用路径反模式）；resource 忽略 + first-scope-only 升级为 AUTH-04/QUAL-03 归**新增 Phase 61**。
- **Rationale:** 完整资源权限矩阵依赖 MultiAuth 实际挂载（Phase 60 AUTH-03），架构上不可能在 Phase 57。用户确认重规划，FUTURE-APIKEY-01/02 拉入 v1.21 为 Phase 61（commit `0d599e9`）。

### Area 4 — 上下文键与 username 语义
- **Options:** 保留原行为 / 修正 username 语义 / 只设 SC#1 的 4 键
- **Selected:** 保留原行为
- **Rationale:** 仅修类型断言（AUTH-01），零行为变更，避免破坏下游读 username 的 handler。username 语义修正归 Phase 61。

## Milestone Re-plan (triggered by Area 3 escalation)
- User chose to pull FUTURE-APIKEY-01/02 into v1.21 milestone.
- **Phase 61 added** (资源级权限矩阵 + 限流生产调优), depends on Phase 60 AUTH-03=启用.
- FUTURE-APIKEY-01 → AUTH-04; FUTURE-APIKEY-02 → QUAL-03 (both Phase 61).
- ROADMAP/REQUIREMENTS/STATE updated + committed (`0d599e9`). Milestone 4→5 phases, 11→13 requirements.
- **Phase 57 boundary unchanged** (still AUTH-01/02/QUAL-02).

## Scope Creep Redirected
- "完整资源级权限矩阵 + 限流调优" 在 Phase 57 内提出 → 重定向为独立 Phase 61（非静默吸收进 Phase 57）。

## Claude's Discretion Items
- 测试文件命名/组织、fake 字段值构造、内联 RequireScope 重构具体写法。

---
*Discussion logged: 2026-08-13*
