---
phase: "84"
slug: "pmodel-gaps"
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-10-28"
---

# Phase 84 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> 依据：`84-RESEARCH.md` § Validation Architecture（研究员产出，Lead 抽查 5 条关键论断全部实证通过）。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test（sqlite 内存库夹具，仿 `internal/services/vdi/vm_service_test.go:498` 的 data_scope 样板） |
| **Config file** | 无新增——沿用各包既有 `_test.go` 基建 |
| **Quick run command** | `go test ./internal/services/system/ ./pkg/middleware/ ./internal/core/db/` |
| **Full suite command** | `go build ./... && go test ./...` |
| **Estimated runtime** | quick ~30s / full ~3min |

---

## Sampling Rate

- **After every task commit:** Run quick run command
- **After every plan wave:** Run full suite command
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30s（quick）/ 180s（full）

---

## Per-Task Verification Map

> Task ID 在 PLAN.md 定稿后由 planner/executor 回填映射；此处预置需求级的验证锚点。

| Requirement | Secure Behavior | Test Type | Automated Command（建议，planner 可细化） | Status |
|-------------|-----------------|-----------|-------------------------------------------|--------|
| PMODEL-01 | data_scope=3 用户查列表只见本部门；admin/data_scope=1 零变化 | unit（红绿验证：回退过滤即失败） | `go test -run TestDataScope ./internal/services/system/` | ⬜ pending |
| PMODEL-01 | 列表缓存 key 含 scope 维度（或带 scope 绕过缓存）——跨用户不串数据 | unit（双 scope 双用户互查缓存隔离） | `go test -run TestUserListCache.*Scope ./internal/services/system/` | ⬜ pending |
| PMODEL-02 | 已停用菜单（status=1）权限码不进前端权限集 | unit | `go test -run TestGetUserPermissions.*Status ./internal/services/system/` | ⬜ pending |
| PMODEL-03 | Step 2 语义命名/注释固化；语义描述与实现一致 | source assertion + 注释审查 | `grep -n "F 型" pkg/middleware/permission.go`（或新函数名） | ⬜ pending |
| PMODEL-04 | 迁移记录表存在且可对账「已应用 vs 仓库最新」 | unit + 脚本 | `go test -run "TestMigrate225|TestRunAndRecord|TestReconcile" ./internal/core/db/`（checker INFO-1 校正：命中 84-03 实际用例名） | ⬜ pending |
| 回归 | v1.24 #16 修复语义不破坏（严格权限中间件） | 回归 | `go test ./pkg/middleware/ ./internal/api/v1/system/` | ⬜ pending |

---

## Wave 0 Requirements

- [ ] `internal/services/system/user_data_scope_test.go`（或同构命名）— PMODEL-01 夹具：多部门用户 + 三种 scope 角色（3/1/admin）
- [ ] `internal/core/db/schema_migrations_test.go` — PMODEL-04 记录表 upsert + 对账用例

*其余沿用各包既有测试基建；无框架安装需求。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| PG 实机 data_scope=3 复测 total（对照 v1.24 实测基线 total=4231） | PMODEL-01 | 生产 PG 与 sqlite 行为差异（v1.24 #79 同款对照） | PG 建受限角色（data_scope=3）→ `POST /system/users/list` → total ≤ 本部门人数；admin 对照不变 |
| `sys_role_menu`/`sys_user_role` 生产库实际列集 vs 模型定义 | PMODEL-04 | 库表侧需 `\d` 实查 | psql `\d sys_role_menu` / `\d sys_user_role`，与模型对照，漂移项登记 |
| v1.24 #79 验收脚本是否断言过用户列表行数（scope 接入后行数会变） | 回归 | 历史脚本行为需人工核对 | 核对 `.planning/milestones/v1.24-phases/` 内 #79 验收记录与脚本 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
