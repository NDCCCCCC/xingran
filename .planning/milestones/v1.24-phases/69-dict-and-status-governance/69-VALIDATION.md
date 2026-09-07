---
phase: 69
slug: dict-and-status-governance
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-19
---

# Phase 69 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> 来源：69-RESEARCH.md `## Validation Architecture`（2026-08-19 实测审计）。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing（后端）/ vitest 4.x + @testing-library/react（前端） |
| **Config file** | `xingran-react-frontend/vitest.config.ts`（Go 用仓内惯例，无统一 ini） |
| **Quick run command** | `go build ./... && go test ./internal/models/ ./internal/core/db/migrations/` |
| **Full suite command** | `go test ./...` ＋ 前端 `npm run type-check && npm run lint && npm run test` |
| **Estimated runtime** | ~60–180 seconds |

---

## Sampling Rate

- **After every task commit:** 后端批任务 `go build ./...` + 受影响包 `go test ./<pkg>/`；前端批任务 `npm run type-check && npx vitest run <相关文件>`
- **After every plan wave:** `go test ./...` 全绿 + `npm run test` 全绿
- **Before `/gsd:verify-work`:** 全套 + 守护脚本零新增命中 + dev 库字典管理页/4 个既有 useDict 页目检
- **Max feedback latency:** 180 seconds

---

## Per-Task Verification Map

> Task ID 在 PLAN.md 生成后由执行器回填；下表为 requirement 级映射（来自 RESEARCH.md）。

| Req ID | Behavior | Test Type | Automated Command | File Exists | Status |
|---------|----------|-----------|-------------------|-------------|--------|
| DICT-01 | 常量值/数量不被静默修改（UserStatusEnabled=0 等） | unit(regression) | `go test ./internal/models/ -run TestStatusConstants` | ❌ W0 新建 `internal/models/status_constants_test.go` | ⬜ pending |
| DICT-01 | 替换后行为不回归（dict/role/dept service 过滤抽样） | unit | `go test ./internal/services/system/` | ✅ 既有 | ⬜ pending |
| DICT-01 | 防新增裸字面量回归 | 脚本守护 | `scripts/check-status-literals`（白名单式 grep） | ❌ W0 新建 | ⬜ pending |
| DICT-02 | seed 幂等 + 行数正确 + 双方言 | unit | `go test ./internal/core/db/migrations/ -run TestMigrate208` | ❌ W0 新建 `migration_208_dict_seed_test.go` | ⬜ pending |
| DICT-02 | dev 库字典可见 | manual/SQL | `sqlite3 data/xingran.db "SELECT COUNT(*) FROM sys_dict_type"` > 0 | — | ⬜ pending |
| DICT-03 | 静态 fallback 断言（字典空时下拉仍有选项） | unit | `npx vitest run src/constants/status.test.ts` + 批次页面测试 | ❌ W0 新建 | ⬜ pending |
| DICT-03 | 字典改值 → 下拉变化 | manual | 字典管理页改 label → 刷新消费页 | — | ⬜ pending |
| DICT-04 | CLAUDE.md 无独立值表 | 静态检查 | grep CLAUDE.md 无 6 行值表格 | — | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/models/status_constants_test.go` — DICT-01 常量锁值（operlog regression_test 模式）
- [ ] `internal/core/db/migrations/migration_208_dict_seed_test.go` — DICT-02 幂等（sqlite 内存库跑两遍断言行数不变）
- [ ] `scripts/check-status-literals`（bash/mjs/go 皆可）— 防回归守护
- [ ] `xingran-react-frontend/src/constants/status.ts` + `status.test.ts` — DICT-03 批 4 共享常量与 fallback 断言

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| dev 库 seed 后字典管理页可见新字典类型 | DICT-02 | 页面目检 | 启动后端 → 字典管理页确认新 dictType 出现且 data 行数正确 |
| 字典改值 → 消费页下拉变化 | DICT-03 | 跨页联动 | 字典管理页改某 label → 刷新消费页（或 useInvalidateDict 免刷新）验证下拉随之变化 |
| 4 个既有 useDict 页不回归 | DICT-03 | 既有页面在 dev 库字典为空的修复验证 | seed 后下拉从空变为有值 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 180s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
