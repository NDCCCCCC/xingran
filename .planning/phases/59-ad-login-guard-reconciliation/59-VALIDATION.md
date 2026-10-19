---
phase: "59"
slug: "ad-login-guard-reconciliation"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-10-19"
validated: "2026-10-20"
---

# Phase 59 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go 1.24.5, stdlib testing + sqlite in-memory) |
| **Config file** | none — existing `_test.go` files in package dirs |
| **Quick run command** | `go test ./internal/services/addomain/ ./internal/services/system/ -count=1` |
| **Full suite command** | `go build ./... && go vet ./... && go test ./...` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services/addomain/ ./internal/services/system/ -count=1`
- **After every plan wave:** Run `go build ./... && go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

Each row mirrors the plan task's full `<automated>` command verbatim (including `go build ./...` where the plan has it), so running this map alone verifies the same contracts as the plans.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 59-01-01 | 01 | 1 | GUARD-01 | T-59-01 | 守卫只读队列，不改写谓词 | unit | `go build ./internal/services/system/ && go test ./internal/services/system/ -run TestPhase59_SyncADUser_ActivePendingQueueSkipsDept -count=1` | ✅ | ✅ green |
| 59-01-02 | 01 | 1 | GUARD-01 | T-59-01 | 查询失败保守跳过，不上抛 | unit | `go test ./internal/services/system/ -run 'TestPhase59_SyncADUser_' -count=1` | ✅ | ✅ green（6 用例全过，2026-10-20 实测） |
| 59-02-01 | 02 | 1 | GUARD-02 | T-59-04 | 视图 DDL 静态、无输入插值；PG 方言 DDL 文本守卫（无 PG 实例可失败） | migration | `go build ./... && go test ./internal/core/db/migrations/ -run TestPhase59_Migration217 -count=1` | ✅ | ✅ green |
| 59-02-02 | 02 | 1 | GUARD-02 | T-59-03 | 四类口径 + 双方言 CAST/布尔/后缀 + 共用 body 形状守卫 | migration | `go test ./internal/core/db/migrations/ -run 'TestPhase59_Migration217' -count=1 -v` | ✅ | ✅ green |
| 59-03-01 | 03 | 2 | GUARD-02 | T-59-06 | 参数绑定 + LIKE 转义 | unit | `go test ./internal/services/addomain/ -run 'TestPhase59_DeptOUMismatch|TestPhase59_MatchConsistency' -count=1` | ✅ | ✅ green（8 用例，2026-10-20 实测） |
| 59-03-02 | 03 | 2 | GUARD-02 | T-59-05 | RequirePermissions 独立鉴权；新路由不破坏全量注册 | unit（httptest） | `go build ./... && go test ./internal/api/v1/system/ -run 'TestPhase59_ReconHandler' -count=1 && go test ./internal/api/v1/system/ -run 'TestSetupRouters_AllRegisterRoutes' -count=1` | ✅ | ✅ green |
| 59-03-03 | 03 | 2 | GUARD-02 | T-59-05 | F 型权限载体 + 精准授权 | migration | `go build ./... && go test ./internal/core/db/migrations/ -run TestPhase59_Migration218 -count=1` | ✅ | ✅ green |
| 59-04-01 | 04 | 3 | GUARD-02 | T-59-09 | 无权限零请求 + 后端鉴权 | component | `cd ruoyi-react-frontend && npx vitest run src/lib/adDomainApi.test.ts src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx` | ✅ | ✅ green |
| 59-04-02 | 04 | 3 | GUARD-02 | T-59-10 | 不展示 UUID、无写入口 | component | `cd ruoyi-react-frontend && npx vitest run src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx` | ✅ | ✅ green |
| 59-04-03 | 04 | 3 | GUARD-02 | — | 无 hex / 无原生 axios；Tabs 与反模式契约 | component + source | `cd ruoyi-react-frontend && npx vitest run src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx src/pages/ad-domain/configs/__tests__/index.render.test.tsx` | ✅ | ✅ green（3 文件 38 用例，2026-10-20 实测） |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/services/system/user_sync_service_phase59_test.go` — GUARD-01 守卫用例（pending/failed 拦截 + WARN / pushed/dead/disable 放行 / 无行照常更新 / 查询失败保守跳过）✅ 59-01 落地
- [x] `internal/core/db/migrations/migration_217_dept_ou_mismatch_view_test.go` — 视图建表、幂等、列顺序、四类口径 ✅ 59-02 落地
- [x] `internal/services/addomain/dept_ou_mismatch_service_test.go` — 分页/筛选 + Go/SQL 双份实现一致性 ✅ 59-03 落地
- [x] `internal/api/v1/system/ad_reconciliation_handler_test.go` — 绑定错误 / 白名单 / 默认分页 ✅ 59-03 落地
- [x] `internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go` — 权限载体幂等与精准授权 ✅ 59-03 落地
- [x] `ruoyi-react-frontend/src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx` — 权限门/空态/错误态/部分行 ✅ 59-04 落地

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 真实 AD 登录回写 dept 被 WARN 拦截 | GUARD-01 | 需真实 AD 环境与 pending 队列行 | 编辑用户 dept → 等队列入队 → 用该用户 AD 登录 → 查日志 WARN + sys_user.dept_id 未变 |
| **PG 方言分支真实执行**：migration_217 的 `CREATE OR REPLACE VIEW` / uuid↔varchar CAST / `NOT sync_enabled` 路径 | GUARD-02 | 本机无 PG 实例（`pg_isready` 不在 PATH），sqlite 单测跑不到 `isPostgreSQL` 分支；自动化只覆盖 DDL 文本形状 | 在目标 PG 环境启动服务 → 确认 AutoMigrate 前的 DROP、`Migrate217DeptOUMismatchView` 无错、随后执行 `SELECT count(*) FROM v_dept_ou_mismatch` 成功；**再次重启**同库，确认不报 `0A000 cannot alter type` 且视图重建成功（幂等） |
| 对账 Tab 权限授予后内容渲染 | GUARD-02 | 需浏览器 + 角色权限配置 | 管理员为角色授予 `ops:ad:config:reconciliation:list` → 登录该角色 → AD 配置页对账 Tab 出数据 |
| 空态标题字重为 600（16px/600） | GUARD-02 | jsdom 不计算层叠样式，`!important` 覆盖 inline style 无法在 vitest 中断言 | 浏览器打开 AD 配置页 → 对账 Tab → 将筛选调成 0 结果 → DevTools 选中空态标题，确认 computed `font-weight: 600`、`font-size: 16px` |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-10-20 — 10/10 Per-Task Map 行全绿（Go 守卫 6 用例 / migration_217 / addomain 8 用例 / handler+路由注册 / migration_218 / 前端 3 文件 38 用例）；4 项 Manual-Only 维持（真实 AD 环境 / 真实 PG 启动冒烟 / 浏览器授权渲染 / computed font-weight）。

## Validation Audit 2026-10-19

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
