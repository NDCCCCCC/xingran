---
phase: 44
slug: ip-r3
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-28
---

# Phase 44 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: `44-RESEARCH.md` §Validation Architecture + §Security Domain。

> **Revision Note (INFO-12)：** frontmatter `nyquist_compliant` / `wave_0_complete` 回填为 `true`。理由：(1) 44-01/44-02 所有 task 含 `<automated>` verify，无 3 个连续 task 缺自动验证（Nyquist 满足）；(2) Wave 0 测试桩清单（下方 8 项）全部在 PLAN.md task 中列为首批 RED 测试目标（44-01 Task 2/3/4/5 + 44-02 Task 1/2/4）。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing（后端）+ Vitest（前端，复用 Phase 42 模式）|
| **Config file** | `go test`（无 ini）；`xingran-react-frontend/vitest.config.ts` |
| **Quick run command** | `go test ./internal/services/asset/... ./internal/api/v1/asset/... ./internal/scheduler/... -count=1 -short` |
| **Full suite command** | `go test ./... && cd xingran-react-frontend && npm run test && npm run build` |
| **Estimated runtime** | ~60-90 秒（quick）；~5-8 分钟（full suite + build）|

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services/asset/... ./internal/api/v1/asset/... ./internal/scheduler/... -count=1 -short`
- **After every plan wave:** Run `go test ./... && cd xingran-react-frontend && npm run test && npm run build`
- **Before `/gsd:verify-work`:** Full suite must be green + `go build ./...` + `npm run build`（CLAUDE.md 强制）
- **Max feedback latency:** 90 秒（quick）/ 8 分钟（wave）

---

## Per-Task Verification Map

> Task IDs 在 PLAN.md 生成后回填；Requirement→Test 映射来自 44-RESEARCH.md §Phase Requirements → Test Map。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD-01 | 44-01 | 1 | EXCEPTION-01 | T-CIDR-Inject | `net.ParseCIDR` 严格校验，非法值拒绝创建；DB `cidr` 列兜底 SQLSTATE 22P02 | unit（service ValidateCIDR）+ migration 集成 | `go test ./internal/services/asset/ -run TestValidateCIDR -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-02 | 44-01 | 1 | EXCEPTION-01 | — | IPv4/IPv6/边界匹配正确 | unit（纯函数 matchException） | `go test ./internal/services/asset/ -run TestMatchException -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-03 | 44-01 | 1 | EXCEPTION-01 | — | GiST inet_ops 索引存在，`>>` 查询返回正确规则 | integration（PG only） | `go test ./internal/services/asset/ -run TestMatchTest -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-04 | 44-01 | 1 | EXCEPTION-02 | T-CIDR-Inject | 5 actions 白名单 + DB CHECK 约束（chk_recon_exc_actions）拒绝非法值 | unit + integration | `go test ./internal/services/asset/ -run TestValidateActions -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-05 | 44-01 | 1 | EXCEPTION-02 | — | 多规则 actions 取并集 | unit（纯函数 mergeActions） | `go test ./internal/services/asset/ -run TestMergeActions -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-06 | 44-01 | 1 | EXCEPTION-02 | — | skip_severity 降级链 critical→high→medium→low | unit（applySkipSeverity） | `go test ./internal/services/asset/ -run TestApplySkipSeverity -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-07 | 44-01 | 1 | EXCEPTION-02 | — | severity_override 多规则取最低（最宽松） | unit（mergeActions override 分支） | `go test ./internal/services/asset/ -run TestMergeActionsSeverityOverride -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-08 | 44-02 | 2 | EXCEPTION-02 | T-44-08 | 转单 cron 命中 no_workorder 不转单 + applied_actions=NULL 仍转单（IS NULL 兜底，BLOCKER-4） | integration（mock workorder service） | `go test ./internal/scheduler/ -run TestCreateWorkorderNoWorkorderFilter -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-09 | 44-01 | 1 | EXCEPTION-04 | T-SQLInject | 命中测试端点用 GORM 占位符 `Where("ip_range >> ?::inet", ip)` 不拼接 | unit + integration | `go test ./internal/services/asset/ -run TestMatchTestResult -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-10 | 44-01 | 1 | EXCEPTION-04 | — | 未指定 user/dept 时 dept/user scope 标记 needsUserDept，仅 global 参与；停用规则不参与（WARN-6） | unit | `go test ./internal/services/asset/ -run TestMatchTestNeedsUserDept -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-11 | 44-01 | 1 | SC 7 | — | 异常列表默认过滤 silence 记录；ShowSilenced=true 可见 | integration | `go test ./internal/services/asset/ -run TestListExceptionsSilenceFilter -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-12 | 44-01 | 1 | SC 10 | — | Layer 3.5 命中例外仍写表（exception_rule_id + applied_actions 非空）；dept scope 端到端覆盖（WARN-5） | integration | `go test ./internal/services/asset/ -run TestDetectLayer3ExceptionHit -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-13 | 44-01 | 1 | AUDIT-01 | T-Repudiation | 例外规则 CRUD handler 调 operlog.Record；ModuleReconciliationExceptionRule 不破坏 25 常量 + 18 mandatorySensitiveKeywords（WARN-11） | integration + regression | `go test ./internal/api/v1/asset/ -run TestExceptionRuleCRUDOperlog -count=1`（+ 现有 regression_test.go） | ❌ W0 新建 / ✅ 已存在 | ⬜ pending |
| TBD-14 | 44-02 | 2 | EXCEPTION-03 | T-Repudiation | cleanupExpiredExceptions 软停用 is_active=1 不删记录（审计链不断） | integration | `go test ./internal/scheduler/ -run TestCleanupExpiredExceptions -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-15 | 44-02 | 2 | EXCEPTION-03 | — | cleanupExpiredExceptions 幂等（二次调用 rowsAffected=0） | integration | `go test ./internal/scheduler/ -run TestCleanupExpiredExceptionsIdempotent -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-16 | 44-02 | 2 | EXCEPTION-03 | — | 过期软停用后历史 exception_rule_id 仍指向有效记录 | integration | `go test ./internal/services/asset/ -run TestSoftDisablePreservesFK -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-17 | 44-02 | 2 | SC 8 | — | 降噪对比端点用独立 COUNT（不用 list.length，MaxPageSize=100 钳制）；COUNT 含 silence 记录（WARN-8） | unit + 静态检查（grep list.length） | `go test ./internal/services/asset/ -run TestBaselineCompare -count=1` | ❌ W0 新建 | ⬜ pending |
| TBD-18 | 44-02 | 2 | EXCEPTION-02 | — | Excel 导入 scope_name→UUID 解析（dept/user/global 三分支）+ TEXT[] 逗号分隔 | integration | `go test ./internal/services/operations/ -run TestReconciliationExceptionImport -count=1` | ❌ W0 新建 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

> Planner 必须在 PLAN.md 把这些测试桩列为首批任务（depends_on 链头）。

- [x] `internal/services/asset/reconciliation_exception_matcher_test.go` — covers EXCEPTION-01/02（matchException + mergeActions + applySkipSeverity 纯函数）— 44-01 Task 2
- [x] `internal/services/asset/reconciliation_exception_test.go` 扩展 — covers EXCEPTION-01/02/04（Create/Update/Delete/MatchTest service + ValidateCIDR/ValidateActions + 停用规则排除 WARN-6）— 44-01 Task 3
- [x] `internal/services/asset/reconciliation_detection_test.go` 扩展 — covers SC 10（Layer 3.5 插入后命中例外仍写表 + dept scope 端到端 WARN-5）— 44-01 Task 4
- [x] `internal/services/asset/reconciliation_service_test.go` 扩展 — covers SC 7（silence 过滤 + ShowSilenced）— 44-01 Task 4
- [x] `internal/services/asset/reconciliation_baseline_test.go` 新建 — covers SC 8（降噪对比 + 基线快照读写 sys_config + COUNT 含 silence WARN-8）— 44-02 Task 2
- [x] `internal/scheduler/reconciliation_tasks_test.go` 新建/扩展 — covers EXCEPTION-03（cleanupExpiredExceptions 软停用 + 幂等）+ EXCEPTION-02（转单 SQL no_workorder 过滤 + IS NULL 兜底 BLOCKER-4）— 44-02 Task 1
- [x] `internal/api/v1/asset/reconciliation_exception_handler_test.go` 扩展 — covers AUDIT-01（CRUD operlog 接入）+ baseline handler（BLOCKER-2）— 44-01 Task 5 + 44-02 Task 3
- [x] migration_174 集成测试（PG dev DB）：`SELECT indexname FROM pg_indexes WHERE indexname='idx_recon_exc_active_range'` + `SELECT conname FROM pg_constraint WHERE conname LIKE 'chk_recon_exc_%'` — 44-01 Task 1

*若 dev DB 不可用，部分 integration 测试可降级为 sqlmock 验证 SQL 正确性，但 **GiST 索引存在性必须 PG 集成验证**（sqlmock 无法验证 inet_ops）。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 例外规则 CRUD admin 页表单交互（CIDR 输入 + 冲突类型多选 + actions 多选 + scope 三选 + 有效期 DatePicker） | EXCEPTION-01/02 | UI 交互无法单测 | admin 页新建/编辑/启用/停用/删除一条规则，校验表单校验提示 + 列表刷新 |
| 命中测试 UI（输入 IP + 可选 user/dept 下拉，结果合并卡片 + 命中规则列表） | EXCEPTION-04 | UI 渲染 + 合并卡片可视化 | 输入测试 IP，校验顶部合并卡片 actions 并集 + severity + 命中规则列表 |
| 降噪效果卡片（基线 vs 当前下降%）— SC 8 前置：运维记录 R2 末期基线（BLOCKER-3） | SC 8 | 需真实告警数据对比 + 基线记录时机约束 | R3 部署前记录基线 → 配置例外规则 → 触发对账 → 看卡片下降% 是否 ≥60%；无基线时看引导提示 |
| 跨模块权限边界（无 `asset:reconciliation:exception:list` 时 admin 页 403/隐藏） | V4 Access Control | 需角色切换 | 用无权限账号访问，校验正确拦截 |
| GiST inet_ops 索引在真实 PG 创建成功 | EXCEPTION-01 | 需 PG dev DB | `SELECT indexname FROM pg_indexes WHERE tablename='sys_reconciliation_exception'` 含 `idx_recon_exc_active_range` |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references（上面 8 个测试桩 — 全部映射到 PLAN task）
- [x] No watch-mode flags
- [x] Feedback latency < 90s（quick）
- [x] `nyquist_compliant: true` set in frontmatter（INFO-12 回填）

**Approval:** pending
