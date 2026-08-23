---
phase: 71
slug: governance-baseline-and-ci-gate
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-20
---

# Phase 71 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go 标准 `testing` (无额外 framework — Phase 71 不补业务测试) |
| **Config file** | 无 — `go test` 直接跑 |
| **Quick run command** | `go test -v -count=1 ./pkg/cache/...` (SC#4 验收, flaky 已修守护) |
| **Full suite command** | `go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...` (SC#5 验收 + 产 coverage.out) |
| **Estimated runtime** | ~180-300 秒 (CI ubuntu-latest 历史数据) |

**辅助脚本:**
- `bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold` — 加权阈值 gate, exit 0/1

---

## Sampling Rate

- **After every task commit:** `go test -v -count=1 ./pkg/cache/...` (SC#4 守护)
- **After every plan wave:** 全量 `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` (SC#5)
- **Before `/gsd:verify-work`:** CI 三绿 (lint + test + coverage gate) + artifact 出现 + coverage-baseline.md 含 Phase 71 后行
- **Max feedback latency:** ~5 分钟 (本地) / ~10 分钟 (CI ubuntu-latest)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 71-01-01 | 01 | 1 | GOV-02 | — | N/A (工程基建,非业务) | smoke | `cat .coverage-threshold` 显示 `12.8` | ❌ W0 | ⬜ pending |
| 71-01-02 | 01 | 1 | GOV-02 | — | N/A | smoke (bash 脚本 + awk 公式) | `bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold` exit 0 | ❌ W0 | ⬜ pending |
| 71-01-03 | 01 | 1 | GOV-02 | — | N/A | manual (故意 bump 阈值验 fail) | `sed -i 's/12.8/99.9/' .coverage-threshold && bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold` exit 1 | ❌ W0 | ⬜ pending |
| 71-01-04 | 01 | 1 | GOV-01 | — | N/A | manual (读 diff) | `git diff .github/workflows/ci.yml` 含 4 个新 step + `-coverprofile=coverage.out -covermode=atomic -count=1` | ❌ W0 | ⬜ pending |
| 71-01-05 | 01 | 1 | GOV-01 | — | N/A | manual (push + 盯 CI) | `git push && gh run watch <run-id>` 三绿 + `backend-coverage` artifact 可下载 | ❌ W0 | ⬜ pending |
| 71-01-06 | 01 | 1 | GOV-04 | — | N/A | smoke (本地跑一次得数字) | 本地 `go test ... -coverprofile=...` 后 `bash check-coverage.sh` 输出含 per-package 表 | ❌ W0 | ⬜ pending |
| 71-01-07 | 01 | 1 | GOV-04 | — | N/A | visual (cat 文件) | `cat .planning/coverage-baseline.md` 含起点行 + Phase 71 后行 (含 per-package 76 行表) | ❌ W0 | ⬜ pending |
| 71-01-08 | 01 | 1 | SC#4 | — | N/A | unit | `go test -v -count=1 ./pkg/cache/...` 全过 (15/15 — 5ead742 守护) | ✅ | ⬜ pending |
| 71-01-09 | 01 | 1 | SC#5 | — | N/A | unit | `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` 全包 exit 0 | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `.coverage-threshold` (NEW) — 仓库根, 纯数字 `12.8\n`
- [ ] `.github/scripts/check-coverage.sh` (NEW) — bash + awk, chmod +x, 复用 quick-260820-bcs 加权公式
- [ ] `.planning/coverage-baseline.md` (NEW) — 起点行 (引用 quick-260820-bcs SUMMARY.md 数字) + Phase 71 后行占位
- [ ] `.github/workflows/ci.yml` (extend) — Test step 加 `-coverprofile=coverage.out -covermode=atomic -count=1`; 新增 Coverage HTML / Coverage gate / Upload coverage artifact 三 step

*Phase 71 是工程基建, 不创建新 test framework / 不补业务测试. 现有 `go test` 基建已覆盖所有 SC.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Coverage HTML step fail-fast 行为 | GOV-01 | Coverage HTML step 在 coverage.out 缺失时是否阻断后续 — 由 CI 跑测验证, 本地无法完全模拟 | push 到 PR, `gh run watch` 看 step 状态; HTML step `if: always()` + `[ -f coverage.out ]` 检查 |
| 阈值 gate fail 时 Upload artifact 仍工作 | GOV-01 + GOV-02 | 故意 bump 到 99.9 后 push 验证; 但 CI 资源宝贵, 一次性手动验 | 在新分支 push 改 .coverage-threshold 到 99.9, 看 CI fail 时 artifact 是否仍可下载 |
| CI artifact 体积 30 天保留是否超额 | GOV-01 | 估算 1.2GB 接近 1G 限额, 需长期观察 | `gh api repos/:owner/:repo/actions/artifacts` 看 storage 占用; 超限时改 retention-days 14 (D-02 决策后评估) |
| deploy.yml 通过 workflow_run gate 自然阻断 | GOV-01 | 端到端验证需完整 push + 看 deploy 是否发起 | coverage gate fail 的 PR 上看 deploy workflow 是否触发 (应不触发) |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies (9/9 ✅)
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify (每 task 都有 unit 或 smoke 命令)
- [ ] Wave 0 covers all MISSING references (4 个文件骨架, 现有 go test 基建覆盖 SC#4 #5)
- [ ] No watch-mode flags (`-count=1` 防缓存, 无 `-watch`)
- [ ] Feedback latency < 600s (本地全量测试 ~5 min, CI ~10 min)
- [ ] `nyquist_compliant: true` set in frontmatter (待 Wave 0 完成后设置)

**Approval:** pending
