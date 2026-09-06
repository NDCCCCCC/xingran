---
phase: 93
slug: config-backup-todo-p2
status: planned
nyquist_compliant: true
wave_0_complete: false
created: 2026-09-05
---

# Phase 93 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> 验证需求 V1..V9 定义见 `93-RESEARCH.md` § Validation Architecture。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go 1.24) + testify；前端 vitest（仅 V8 前端切片） |
| **Config file** | none — 沿用既有 go.mod / vitest 配置 |
| **Quick run command** | `go test ./internal/services/ -run "TestCbk93" -count=1` |
| **Full suite command** | `go test ./internal/services/... -count=1` |
| **Estimated runtime** | quick ~20s / full ~180s |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...` + quick run command
- **After every plan wave:** Run `go test ./internal/services/... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green（含 `go test ./...` 0 失败，v1.29 D-03）
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

> Task ID 为占位——planner 生成 PLAN 后以实际 task 编号回填映射（映射关系按 V1..V9 → 任务归属）。

> 2026-09-05 plan-phase 回填：实际拆分为 6 plans / 5 waves（见 ROADMAP § Phase 93），任务编号 = {plan}-{task}。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 93-01-01/02 | 01 | 1 | BACKUP-CLOSED-01 (V1) | T-93-02/T-93-03 | 压缩产物仅落备份目录 + 解压 64MB 上限 | unit/roundtrip | `go test ./internal/services/ -run TestCbk93 -count=1` | ❌ 计划内新建（93-01-02） | ⬜ pending |
| 93-01-02 | 01 | 1 | BACKUP-CLOSED-02 (V2) | T-93-04 | 损坏流拒绝解析 + 双检查 | unit | `go test ./internal/services/ -run "TestCbk93Decompress|TestCbk93GetBackupContentCompressed"` | ❌ 计划内新建 | ⬜ pending |
| 93-02-01..03 | 02 | 2 | BACKUP-CLOSED-03 下发内核 (V3 部分) | T-93-05/T-93-06 | RestoreConfig 唯一入口 + fail-fast + 超时 | unit+e2e (FileTransport) | `go test ./internal/device/ -run "TestRestoreConfig|TestCleanConfig|TestExitConfig"` | ❌ 计划内新建（93-02-03） | ⬜ pending |
| 93-03-01/02 | 03 | 3 | D-04/D-08/D-10/D-14/D-15/D-34 (V6/V7 service) | T-93-01/T-93-07/T-93-08 | 同设备校验 + 互斥 + 恢复前备份中止 + 状态机 | build+vet（e2e 在 93-06） | `go build ./... && go test ./internal/core/db/ ./internal/services/ -count=1` | ✅/❌ 计划内新建 | ⬜ pending |
| 93-04-01/02 | 04 | 4 | BACKUP-CLOSED-03 收口 + D-13/D-17/D-18/D-31 (V8 handler) | T-93-10/T-93-11 | taskId 契约 + 组权限继承 + operlog 发起记录 | unit+type-check(go) | `go test ./internal/api/v1/network/ ./internal/utils/operlog/ -count=1` | ✅（测试文件重写） | ⬜ pending |
| 93-05-01/02 | 05 | 4 | D-16/D-19 (V8 前端切片) | T-93-12 | 轮询终态停止 + deviceId 携带 | vitest+type-check | `npm run type-check && npm run test -- --run src/pages/network/backups` | ❌ 计划内新建 | ⬜ pending |
| 93-06-01..04 | 06 | 5 | BACKUP-CLOSED-04/05 (V3/V4/V5/V9) | T-93-14/T-93-15 | e2e 断言链 + 失败 4 场景 + 锁值防线不回归 + 全量 gate | e2e (FileTransport)+regression | `go test ./... -count=1` + `npm run type-check && npm run lint && npm run test` | ❌ 计划内新建（93-06-01） | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] ~~独立 Wave 0 stub 文件~~ — 6-plan 拆分后无独立 Wave 0：93-01-02（tdd 任务）在实现同 commit 内新建 93_01 测试文件，后续 V 命令引用随即有效
- [x] 79_06 helper 复用确认（newCbk7906 / cbk7906Chdir / writeFixture7906 / newExecutor7906 同包可见——同包 `services` 无需导出；device 包 93-02-03 按 79_06:142 模式自建装配）

*Existing infra covers the rest（sqlite in-memory + FileTransport fixture 基建已就绪）。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 真机恢复（华为/H3C/锐捷实机下发） | BACKUP-CLOSED-03 生产语义 | FileTransport 无法覆盖真实设备 CLI 差异 | deferred——仿 v1.18/19 site-visit UAT 模式另行登记（owner = 现场运维同事），非本期验收门槛（D-30） |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
