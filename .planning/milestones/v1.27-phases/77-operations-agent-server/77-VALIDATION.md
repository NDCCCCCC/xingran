---
phase: 77
slug: operations-agent-server
status: ready
nyquist_compliant: true
wave_0_complete: true
created: 2026-08-24
---

# Phase 77 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib + testify v1.11.1) |
| **Config file** | none — existing infrastructure (Phase 76 已交付全部 test doubles/注入缝) |
| **Quick run command** | `go test ./internal/services/operations/ ./internal/agent/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~120 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services/operations/ ./internal/agent/...`
- **After every plan wave:** Run `go test ./...` + `.github/scripts/check-coverage.sh`（gate 不倒退）
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 77-01-T1 | 77-01 | 1 | BLOCK-01 | T-77-01-01 | fixture 仅 sqlite :memory:,测试文件禁 postgres DSN | unit(sqlite) | `go test -count=1 -run "TestWSD77_" ./internal/services/operations/ && go build ./...` | workstation_device_77_01_test.go（新建） | ⬜ |
| 77-01-T2 | 77-01 | 1 | BLOCK-01 | T-77-01-02 | quirk 修复 D-03 有据判定 + 原子 commit + SUMMARY deviation；coverage checkpoint 落 SUMMARY | unit+coverage | `go test -count=1 ./internal/services/operations/ && go test -count=1 -cover ./internal/services/operations/ && go build ./...` | 同 77-01-T1 | ⬜ |
| 77-02-T1 | 77-02 | 2 | BLOCK-01 | T-77-02-01 | D-06 全内存 xlsx 生成，零 testdata 二进制进 git；D-07 结构断言 | unit(sqlite) | `go test -count=1 -run "TestExp77_" ./internal/services/operations/ && go build ./...` | excel_export_chain_77_02_test.go（新建） | ⬜ |
| 77-02-T2 | 77-02 | 2 | BLOCK-01 | T-77-02-01/02 | 物理链路 sheet sqlite 降级为预期行为（P-77-10）；coverage ≥68% 落 SUMMARY | unit+coverage | `go test -count=1 ./internal/services/operations/ && go test -count=1 -cover ./internal/services/operations/ && go build ./...` | 同 77-02-T1 | ⬜ |
| 77-03-T1 | 77-03 | 3 | BLOCK-01 | T-77-03-02/03 | Q-77-C doc-only 修复（git diff 仅注释行）；D-06 畸形输入手工字节构造 | unit(sqlite) | `go test -count=1 -run "TestImp77_" ./internal/services/operations/ && go build ./...` | excel_import_rest_77_03_test.go + excel_raw_rows_77_03_test.go（新建） | ⬜ |
| 77-03-T2 | 77-03 | 3 | BLOCK-01 | T-77-03-01 | 状态断言引用 models 具名常量，禁裸 0/1；不为覆盖率改 PG-only SQL | unit(sqlite) | `go test -count=1 -run "TestImp77_" ./internal/services/operations/ && go build ./...` | reference_resolver_77_03_test.go + workstation_floor_code_77_03_test.go（新建） | ⬜ |
| 77-03-T3 | 77-03 | 3 | BLOCK-01 | T-77-03-01 | floor NOW() 不改（P-77-2）；BLOCK-01 收口判据 ≥70.0% | coverage 收口 | `go test -count=1 -cover ./internal/services/operations/ && go build ./...` | 同 77-03-T2 | ⬜ |
| 77-04-T1 | 77-04 | 1 | BLOCK-02 | T-77-04-01/03 | 出站 HTTP 仅 srv.URL 本地回环；token/secret 只用 "test-secret" 字面量 | unit(httptest) | `go test -count=1 -run "TestJWT77_" ./internal/agent/server/ && go build ./...` | jwt_conn_77_04_test.go（新建） | ⬜ |
| 77-04-T2 | 77-04 | 1 | BLOCK-02 | T-77-04-01/02 | goroutine 收尾 ctx cancel + Disconnect；channel 同步断言禁裸 sleep（P-77-4） | unit+coverage | `go test -count=1 ./internal/agent/server/ && go test -count=1 -cover ./internal/agent/server/ && go build ./...` | 同 77-04-T1 | ⬜ |
| 77-05-T1 | 77-05 | 2 | BLOCK-02 | T-77-05-01 | Q-77-A crypto/rand + Q-77-B 长度守卫，TDD RED→GREEN 各自原子 commit；viper.Reset 防污染（P-77-3） | unit(tdd) | `go test -count=1 -run "TestCfg77_" ./internal/agent/server/ && go build ./...` | config_account_77_05_test.go（新建） | ⬜ |
| 77-05-T2 | 77-05 | 2 | BLOCK-02 | T-77-05-02/03/04 | seam 先 t.Cleanup 后覆盖（P-77-9，禁 t.Parallel）；TestHelperProcess guard 保持第一行；git diff 仅 account_manager.go | unit(re-exec) | `go test -count=1 -run "TestAcct77_" ./internal/agent/server/ && go build ./... && go test -count=1 ./internal/agent/...` | subprocess_stub_test.go（扩展 4 shape） | ⬜ |
| 77-05-T3 | 77-05 | 2 | BLOCK-02 | T-77-05-05 | sanitizeError 脱敏断言（敏感原文不出现）；BLOCK-02 收口 ≥70.0% + SC#3 人工对比说明落 SUMMARY | coverage 收口 | `go test -count=1 -cover ./internal/agent/server/ && go build ./...` | handlers_77_05_test.go（新建） | ⬜ |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.（sqlite/httptest/excelize/re-exec stub 全部就绪 — 见 76-VERIFICATION.md INFRA-01..05）

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Windows 本地 vs ubuntu CI 覆盖率差 <2pp（SC#3，D-04/D-05） | BLOCK-02 | CI 数字需 push 后观测，本地无法程序化获取 | 收口时本地 `go test -cover ./internal/services/operations/ ./internal/agent/...` 与 CI backend-coverage artifact 的 per-package 数字人工对比，差值记录进 77-VERIFICATION.md |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies（12/12 task 均有 `<automated>` 命令，零 MISSING）
- [x] Sampling continuity: no 3 consecutive tasks without automated verify（每个 task 均有 automated verify，连续性天然满足）
- [x] Wave 0 covers all MISSING references（无 MISSING——Phase 76 基建全覆盖，见 Wave 0 Requirements）
- [x] No watch-mode flags（全部命令 `-count=1`，无 watch 模式）
- [x] Feedback latency < 120s（quick run 单包/双包口径实测 ~120s 内，见 Test Infrastructure）
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** ready（2026-08-24 planner 回填 12 行 Per-Task Map 并核对 Sign-off；执行期按 Map 逐行翻绿）
