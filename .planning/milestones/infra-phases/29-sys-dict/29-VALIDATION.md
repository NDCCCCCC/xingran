---
phase: 29
slug: sys-dict
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-10
---

# Phase 29 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (backend) + Vitest (frontend) |
| **Config file** | xingran-react-frontend/vitest.config.ts |
| **Quick run command** | `go test ./internal/services/operations/ -v -run TestWorkstationStatus` |
| **Full suite command** | `go test ./... && cd xingran-react-frontend && npm run test` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services/operations/ -v`
- **After every plan wave:** Run `go test ./... && cd xingran-react-frontend && npm run test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 29-01-01 | 01 | 0 | 字典初始化 | — | 数据完整性验证 | integration | `go test -run TestDictInitialization` | ❌ W0 | ⬜ pending |
| 29-02-01 | 02 | 1 | 数据迁移 | T-29-01 | 无数据丢失 | integration | `go test -run TestMigrationDataIntegrity` | ❌ W0 | ⬜ pending |
| 29-03-01 | 03 | 2 | 枚举移除 | — | 类型安全 | unit | `go test -run TestWorkstationModel` | ❌ W0 | ⬜ pending |
| 29-03-02 | 03 | 2 | status_text 生成 | T-29-02 | 无注入攻击 | unit | `go test -run TestStatusTextGeneration` | ❌ W0 | ⬜ pending |
| 29-04-01 | 04 | 3 | 前端字典加载 | T-29-03 | XSS 防护 | integration | `npm run test -- src/pages/operations/workstations/index.test.tsx` | ❌ W0 | ⬜ pending |
| 29-04-02 | 04 | 3 | 常量移除 | — | 无硬编码 | lint | `grep -r "STATUS_COLOR_MAP" src/pages/operations/workstations/` | ❌ W0 | ⬜ pending |
| 29-05-01 | 05 | 4 | 端到端验证 | — | 完整流程 | e2e | `npm run test -- src/e2e/workstation-status.spec.ts` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/operations/workstation_status_test.go` — 字典初始化和数据迁移测试
- [ ] `internal/models/workstation_test.go` — 工位模型类型安全测试
- [ ] `xingran-react-frontend/src/pages/operations/workstations/index.test.tsx` — 前端字典加载和渲染测试
- [ ] `xingran-react-frontend/src/e2e/workstation-status.spec.ts` — 端到端流程测试
- [ ] `internal/services/operations/migration_test.go` — 数据迁移完整性测试

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| UI 颜色一致性 | D-07 | css_class 视觉效果需人工确认 | 1. 打开工位列表页面 2. 检查状态标签颜色（绿色=空闲，红色=占用，橙色=维护） |
| 数据库字段类型 | D-02 | PostgreSQL 类型系统需直接验证 | 1. 连接数据库 2. 执行 `SELECT status FROM ops_workstation LIMIT 1` 3. 确认返回类型为 varchar |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
