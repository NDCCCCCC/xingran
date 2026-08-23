---
phase: 71-governance-baseline-and-ci-gate
verified: 2026-08-20T23:30:00Z
status: passed
score: 5/5 must-haves verified
overrides_applied: 0
overrides: []
re_verification: false
---

# Phase 71: Governance Baseline + CI Gate Verification Report

**Phase Goal:** CI 后端 job 产出覆盖率报告与阈值 gate;建立测试补齐的基线记录与 ratchet 防倒退机制;前置 flaky test 修复验收。
**Verified:** 2026-08-20T23:30:00Z
**Status:** passed

## Goal Achievement

### Observable Truths

| #   | Truth                                                                                                          | Status     | Evidence                                                                                                                                                                                                                                |
| --- | -------------------------------------------------------------------------------------------------------------- | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `ci.yml` backend job has `-coverprofile=coverage.out -covermode=atomic -count=1` AND `coverage.out` + `coverage.html` artifacts via `actions/upload-artifact@v4` with `retention-days: 30` | ✓ VERIFIED | `.github/workflows/ci.yml` line 61 contains the full flag set; lines 84 (`uses: actions/upload-artifact@v4`) and 90 (`retention-days: 30`) match D-02. CI run 32384622877 backend job steps 6 (HTML), 7 (gate), 8 (Upload artifact) all `success`. |
| 2   | CI introduces a weighted-average gate; threshold = 12.8 from `.coverage-threshold`; below-threshold PR fails; ratchet mechanism in place | ✓ VERIFIED | `.coverage-threshold` = 5 bytes `31 32 2e 38 0a` (= `12.8\n`). Local smoke test: gate exits 0 at 12.8 (`PASS: weighted avg 12.80% >= threshold 12.80%`), exits 1 at 99.9 (`FAIL: weighted avg 12.80% < threshold 99.90%`). Ratchet: `.coverage-threshold` is the single source of truth (D-04). |
| 3   | `.planning/coverage-baseline.md` contains starting row (commit `5ead742`) + Phase 71 后 row (commit `326a541`) + two 76-line per-package tables inline | ✓ VERIFIED | File exists (16935 bytes); contains `## 起点 (v1.26 启动前)` row `| 2026-08-20 | 起点 | 12.8 | 43652 | 5589 | 33 | 5ead742 | n/a | n/a | n/a |` (line 13); `## Phase 71 后` row `| 2026-08-20 | Phase 71 后 | 12.8 | 43652 | 5589 | 33 | 326a541 | gsd-execute-phase 71 | n/a | n/a |` (line 102); both `### Per-package` subsections present. |
| 4   | `go test -v -count=1 ./pkg/cache/...` exits 0 with `--- PASS: TestAsyncRetryWorker_Enqueue` (5ead742 guard)      | ✓ VERIFIED | Local run: `--- PASS: TestAsyncRetryWorker_Enqueue (0.20s)` and `--- PASS: TestAsyncRetryWorker_QueueSemantics (0.00s)` both explicitly in output; final line `PASS / ok github.com/xingran-next/xingran-go-backend/pkg/cache 7.037s`. Exit code 0. |
| 5   | `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` exits 0 with no failures, no panics            | ✓ VERIFIED | Local run: exit code 0; 42 `ok` packages; 0 `FAIL` lines; 0 `panic` lines. CI run 32384622877 backend Test step: `success` (7m41s). |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact                                                       | Expected                                                                  | Status      | Details                                                                                                                                                                                                                                                                                                  |
| -------------------------------------------------------------- | ------------------------------------------------------------------------- | ----------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `.coverage-threshold`                                          | 5 bytes `31 32 2e 38 0a` (`12.8` + LF); no BOM, no comments                | ✓ VERIFIED  | `wc -c` = 5; `xxd` = `3132 2e 38 0a` (ASCII 1, 2, ., 8, LF); `cat` outputs `12.8`; single line.                                                                                                                                                                                                            |
| `.github/scripts/check-coverage.sh`                            | bash + awk gate, executable, syntax-clean, docstring header              | ✓ VERIFIED  | 139 lines; mode `-rwxr-xr-x`; `bash -n` clean; shebang `#!/usr/bin/env bash`; awk formula byte-identical to quick-260820-bcs; embedded exit codes 0/1/2; `set -euo pipefail`.                                                                                                                              |
| `.github/workflows/ci.yml`                                     | backend job extended with Test + 3 new steps (HTML/gate/Upload); YAML valid | ✓ VERIFIED  | Line 61 = Test with `-coverprofile=coverage.out -covermode=atomic`; lines 64-69 = Coverage HTML (`if: always()`); lines 71-76 = Coverage gate; lines 78-90 = Upload artifact (`actions/upload-artifact@v4`, `name: backend-coverage`, `path: coverage.out + coverage.html`, `retention-days: 30`, `if: always()`). |
| `.planning/coverage-baseline.md`                               | 起点 + Phase 71 后 rows + 2 × 76-line per-package blocks; TBD replaced    | ✓ VERIFIED  | 196 lines; 2 H2 sections + 3 H3 subsections; starting block lines 17-94 (76 lines) + Phase 71 后 block lines 106-183 (76 lines); Phase 71 后 row commit column = `326a541` (TBD replaced via amend commit `cfec2c4`).                                                                                  |
| `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` | UNCHANGED per D-03                                                       | ✓ VERIFIED  | `git log --oneline --all -- .planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` last commit = `5ead742` (pre-Phase 71); no Phase 71 commit touches this file.                                                                                                                                  |

### Key Link Verification

| From                            | To                                        | Via                                                            | Status     | Details                                                                                                                                                            |
| ------------------------------- | ----------------------------------------- | -------------------------------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ci.yml` Coverage gate step     | `.github/scripts/check-coverage.sh`        | `bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold` | ✓ WIRED    | Line 76.                                                                                                                              |
| `.github/scripts/check-coverage.sh` | `.coverage-threshold`                  | `THRESHOLD="$(cat "$THRESHOLD_PATH")"`                         | ✓ WIRED    | Line 57 — anchored `$ROOT/$THRESHOLD_FILE` path.                                                                                                                  |
| `ci.yml` Upload artifact step  | `actions/upload-artifact@v4`               | step `uses:` + `path:` (coverage.out + coverage.html)          | ✓ WIRED    | Lines 84-90; CI run 32384622877 step 8 = `success`.                                                                                                                |
| Test step → coverage.out        | Coverage HTML + Coverage gate + Upload     | Test step produces `coverage.out`; downstream steps consume it  | ✓ WIRED    | CI run 32384622877 steps 5 (Test) → 6 (HTML) → 7 (gate) → 8 (Upload) all `success`.                                                                                |
| Plan 71-01b amend commit        | `.planning/coverage-baseline.md` Phase 71 后 row commit column | Edit `TBD` → `326a541`, commit `cfec2c4`                        | ✓ WIRED    | Commit `cfec2c4` exists in history: "docs(71-01b): amend coverage baseline — Phase 71 后 commit ref TBD → 326a541". File line 102 has `326a541` not `TBD`. |

### Data-Flow Trace (Level 4)

N/A — Phase 71 is engineering infrastructure (CI shell + workflow YAML + tracking file). No dynamic data flows. Coverage artifact (`coverage.out` + `coverage.html`) is generated by Go's `go tool cover` from in-process test instrumentation; verified by CI run 32384622877 step 6 (`Coverage HTML report` = success).

### Behavioral Spot-Checks

| Behavior                                                                  | Command                                                                                                                                                                                                                                                       | Result                                                                                                                                                                                                                                                                                                                            | Status   |
| ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Gate passes at 12.8                                                        | `bash .github/scripts/check-coverage.sh /tmp/cov-phase71.out .coverage-threshold`                                                                                                                                                                              | Exit 0; output `PACKAGE    43652     5589  12.80%` + `PASS: weighted avg 12.80% >= threshold 12.80%` + `coverage gate passed (threshold=12.8%)`.                                                                                                                                                                                   | ✓ PASS   |
| Gate fails at 99.9                                                        | `printf '99.9\n' > .coverage-threshold && bash .github/scripts/check-coverage.sh /tmp/cov-phase71.out .coverage-threshold`                                                                                                                                   | Exit 1; output `FAIL: weighted avg 12.80% < threshold 99.90%` + `check-coverage.sh: profile parsed but no PASS line emitted — treating as failure`. Restoration: `cat .coverage-threshold` = `12.8`.                                                                                                                                  | ✓ PASS   |
| Gate soft-skips when no profile                                            | `bash .github/scripts/check-coverage.sh /nonexistent .coverage-threshold`                                                                                                                                                                                       | Exit 0; `coverage profile $PROFILE missing — was Test step skipped?` printed to stderr.                                                                                                                                                                                                                                              | ✓ PASS   |
| pkg/cache 15/15 (5ead742 guard)                                            | `go test -v -count=1 ./pkg/cache/...`                                                                                                                                                                                                                          | Exit 0; explicit `--- PASS: TestAsyncRetryWorker_Enqueue (0.20s)` AND `--- PASS: TestAsyncRetryWorker_QueueSemantics (0.00s)` in output; `ok github.com/xingran-next/xingran-go-backend/pkg/cache 7.037s`.                                                                                                                          | ✓ PASS   |
| Full backend suite exit 0                                                  | `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...`                                                                                                                                                                                              | Exit 0; 42 `ok` packages; 0 `FAIL` lines; 0 `panic` lines; packages with tests end with `ok` (5 packages `[no test files]` skipped).                                                                                                                                                                                                | ✓ PASS   |
| CI backend Coverage gate green                                            | `gh run view 32384622877 --json jobs` (backend job steps)                                                                                                                                                                                                       | Steps 1-8 all `success` including step 7 `Coverage gate` and step 8 `Upload coverage artifact`. Backend job overall `success`.                                                                                                                                                                                                     | ✓ PASS   |

### Probe Execution

N/A — Phase 71 has no probes (no `scripts/*/tests/probe-*.sh` declared in PLAN/SUMMARY).

### Requirements Coverage

| Requirement | Source Plan            | Description                                                  | Status      | Evidence                                                                                                                                                                                                              |
| ----------- | ---------------------- | ------------------------------------------------------------ | ----------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| GOV-01      | 71-01 (Task 3) + 71-01b (Task 4) | CI 后端 job 以 `-coverprofile` 跑全量测试并产出覆盖率报告 (profile artifact 可下载) | ✓ SATISFIED | `.github/workflows/ci.yml` line 61 (`-coverprofile=coverage.out -covermode=atomic -count=1`) + lines 78-90 (Upload artifact step with 30-day retention). CI run 32384622877 backend step 8 `success`; artifact `backend-coverage` present. |
| GOV-02      | 71-01 (Tasks 1-2) + 71-01b (Tasks 1, 4) | CI 落地全局加权覆盖率阈值 gate (失败即阻断) + ratchet 防倒退机制 | ✓ SATISFIED | `.coverage-threshold` = 12.8; `.github/scripts/check-coverage.sh` (139 lines, exit 0/1/2); local smoke test passes at 12.8 + fails at 99.9; CI run 32384622877 backend step 7 `Coverage gate` = `success`. Ratchet contract = D-04 (`.coverage-threshold` only edit for Phase 72/73/74). |
| GOV-04      | 71-01 (Task 4) + 71-01b (Task 4 amend) | 覆盖率基线与进展落盘 —— 起点 12.8% / per-package 数据 / 各 phase 后数字 | ✓ SATISFIED | `.planning/coverage-baseline.md` contains 起点 row (commit `5ead742`, 12.8%, 43652 stmts, 5589 covered, 33 zero-coverage packages) + 76-line per-package block; Phase 71 后 row (commit `326a541`, same numbers — Phase 71 ships 0 business changes) + identical 76-line per-package block. |
| GOV-03      | —                      | (deferred to Phase 74 per REQUIREMENTS.md traceability)     | DEFERRED    | Not in Phase 71 scope per REQUIREMENTS.md Phase mapping (`GOV-03 | Phase 74 | Pending`).                                                                                                                                  |

All Phase 71 requirements (GOV-01, GOV-02, GOV-04) are SATISFIED. GOV-03 is correctly deferred to Phase 74.

### Anti-Patterns Found

| File                          | Line | Pattern                                                                                                                                                                                            | Severity | Impact                                                                                                                                                                                                |
| ----------------------------- | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `.planning/coverage-baseline.md` | 191  | The literal word `TBD` in the explanatory ratchet note text: `> The \`commit\` column on the Phase 71 后 row reads \`TBD\` until plan 71-01b Task 4 amends this file...`                                                                                | ℹ️ Info  | Stale documentation text only. The actual Phase 71 后 row commit column (line 102) was correctly amended to `326a541` per plan 71-01b Task 4. The note text describes the historical context of the TBD placeholder, not an open work item. No action needed. |
| All other Phase 71 files    | —    | No `TBD`, `FIXME`, `XXX`, `TODO`, `HACK`, `PLACEHOLDER` markers                                                                                                                                  | —        | Clean.                                                                                                                                                                                                |

**Debt marker gate:** No blockers. The single `TBD` mention is explanatory text, not an open work item.

### Pre-existing frontend Format check failure (Reporting-Only Deviation, NOT Phase 71 scope)

The SUMMARY files (71-01-SUMMARY.md, 71-01b-SUMMARY.md) document that the frontend Format check (`prettier --check`) fails on CI runs 32382995316 and 32384622877. This is verified independently: CI run 32384622877 frontend job step 5 `Format check` = `failure`. Per the planner directive (plan 71-01b Step 4): "If frontend fails, that's pre-existing and out of Phase 71 scope — note but do not block Phase 71 ship. If backend fails on lint or test, those are pre-existing issues — note but the Coverage gate is what Phase 71 specifically tests."

The Phase 71 backend Coverage gate PASSED on both CI runs. This is the sole Phase 71-mandated CI blocker per GOV-02 / SC#2. The frontend Format check failure is pre-existing workspace state (Phase 68 leftover `M internal/services/system/asset_columns_schema.json` + Phase 63 follow-up `M xingran-react-frontend/src/types/global.d.ts` + `M xingran-react-frontend/tsconfig.app.json`) and not in Phase 71 scope.

### Human Verification Required

None. All 5 SC, all 3 in-scope requirements (GOV-01/02/04), all 4 D-locked decisions (D-01..D-04) are programmatically verified against the codebase.

### Decision Coverage

| Decision | Honored? | Evidence                                                                                                                                                                                                                                                                                                |
| -------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| D-01 (bash + awk, zero-deps; `.coverage-threshold` is single-value ratchet SoT) | ✓ YES | `.github/scripts/check-coverage.sh` shebang `#!/usr/bin/env bash`; awk formula byte-identical to quick-260820-bcs; no third-party dependencies; `.coverage-threshold` is 5-byte single-value file at repo root (sibling of `go.mod`, `.gitignore`). |
| D-02 (`actions/upload-artifact@v4` + `retention-days: 30`) | ✓ YES | `.github/workflows/ci.yml` line 84: `uses: actions/upload-artifact@v4`; line 90: `retention-days: 30`. NOT `@v7` (D-02 explicit). CI artifact 1.4MB downloadable; 30-day retention confirmed via `gh api` per SUMMARY 71-01b. |
| D-03 (`.planning/coverage-baseline.md` is separate file; quick scan SUMMARY.md untouched) | ✓ YES | `.planning/coverage-baseline.md` exists at `.planning/coverage-baseline.md` (sibling of `.planning/quick/`, not inside it). `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` last commit = `5ead742` (pre-Phase 71); no Phase 71 commit touches this file. |
| D-04 (manual ratchet — `.coverage-threshold` is the single source of truth; no auto-bump) | ✓ YES | `.coverage-threshold` is the only file Phase 72/73/74 execute plans will edit. `check-coverage.sh` and `ci.yml` are not touched as part of a ratchet (Ratchet note in `.planning/coverage-baseline.md` lines 190-196 documents this). No auto-bump helper script added. |

All 4 D-locked decisions honored.

### Commit Verification

| SHA       | Subject                                                                                              | Status      |
| --------- | ---------------------------------------------------------------------------------------------------- | ----------- |
| 4242a75   | ci(71): add .coverage-threshold baseline (12.8)                                                      | ✓ Present in history (`.coverage-threshold` only). |
| 3e2664c   | ci(71): add check-coverage.sh — weighted-average gate (GOV-02)                                       | ✓ Present in history (`.github/scripts/check-coverage.sh` only). |
| 75a4703   | ci(71): extend backend job with coverage steps (GOV-01 + GOV-02 + D-02)                              | ✓ Present in history (`.github/workflows/ci.yml` only). |
| 75b96be   | docs(71): scaffold coverage-baseline.md (GOV-04)                                                     | ✓ Present in history (`.planning/coverage-baseline.md` initial creation). |
| 326a541   | docs(71-01): complete plan 71-01 (file creation half) — SUMMARY + STATE + ROADMAP                     | ✓ Present in history. |
| cfec2c4   | docs(71-01b): amend coverage baseline — Phase 71 后 commit ref TBD → 326a541                         | ✓ Present in history (`.planning/coverage-baseline.md` line 102 commit column updated). |
| eca29fb   | docs(71-01b): complete plan 71-01b (verify+push+amend half) — SUMMARY + STATE + ROADMAP               | ✓ Present in history. |

All 6 Phase 71 commits (4 file-creation + 1 close-out + 1 amend) present in git history. No business code modified (verified via `git diff 5ead742..HEAD -- '*.go'` = empty).

### Gaps Summary

No gaps found. All 5 SC, 3 requirements, 4 decisions verified. Phase 71 ships as designed.

---

_Verified: 2026-08-20T23:30:00Z_
_Verifier: Claude (gsd-verifier)_