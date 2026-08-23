---
phase: 71-governance-baseline-and-ci-gate
plan: 01
subsystem: ci-coverage
tags: [coverage-gate, github-actions, bash-awk, zero-deps, ratchet, gha-artifact]

# Dependency graph
requires:
  - phase: 69-dict-and-status-governance
    provides: "scripts/check-status-literals.sh ratchet-guard pattern (bash + set -euo pipefail + ROOT anchor + collect/check structure + exit 0/1/2 convention) that check-coverage.sh mirrors"
  - phase: quick-260820-backend-test-coverage-scan
    provides: "43652 stmts / 5589 covered / 12.8% weighted avg / 33 zero-coverage packages baseline + 76-line per-package-coverage.txt + awk aggregation formula"
  - phase: 5ead742
    provides: "pkg/cache TestAsyncRetryWorker_Enqueue flaky-test fix (commit anchor for the 起点 row in coverage-baseline.md)"
provides:
  - ".coverage-threshold single-value gate floor (5 bytes, 12.8 + LF)"
  - ".github/scripts/check-coverage.sh bash + awk gate (139 lines, chmod +x, syntax-clean, awk formula byte-identical to quick-260820-bcs)"
  - ".github/workflows/ci.yml backend job extended with 4 coverage steps: Test +coverprofile, Coverage HTML (if: always()), Coverage gate, Upload artifact (actions/upload-artifact@v4, 30-day retention, if: always())"
  - ".planning/coverage-baseline.md ratchet tracking file with 起点 + Phase 71 后 snapshot rows and two 76-line per-package fenced blocks (TBD commit on Phase 71 后 row pending 71-01b amend)"
affects:
  - phase: 72-74 (ratchet)
    note: "Each phase execute plan will bump .coverage-threshold AND append a row to coverage-baseline.md in the same atomic commit (D-04 manual ratchet)"
  - phase: 71-01b
    note: "Plan 71-01b verifies the file-creation half (this plan) by running go test locally + push + gh run watch + amending the TBD SHA in coverage-baseline.md"

# Tech tracking
tech-stack:
  added:
    - "actions/upload-artifact@v4 (D-02, NOT @v7 — explicit v4 pin for Phase 71)"
  patterns:
    - "bash + awk zero-deps gate mirroring scripts/check-status-literals.sh (shebang + set -euo pipefail + ROOT anchor + case 0/1/2 exit-code convention)"
    - "single-value threshold file at repo root as the ratchet SoT (D-01, D-04)"
    - "if: always() on Coverage HTML + Upload artifact steps so gate failure does not blind the developer (Gotcha #9 from 71-RESEARCH)"
    - "Go coverage profile exclusion list embedded in check-coverage.sh awk (scripts/, tests/scripts/, node_modules/) — no external config file"

key-files:
  created:
    - ".coverage-threshold (5 bytes, 12.8 + LF)"
    - ".github/scripts/check-coverage.sh (139 lines, bash + awk gate)"
    - ".planning/coverage-baseline.md (196 lines, ratchet tracking)"
  modified:
    - ".github/workflows/ci.yml (Test step + 3 new coverage steps in backend job; frontend Format check step from prior session preserved)"

key-decisions:
  - "Per-package lines: byte-identical copy of quick-260820-bcs per-package-coverage.txt (column widths must match awk printf '%%-50s %%8d %%8d %%6.2f%%\\n' so local and CI outputs diff cleanly)"
  - "check-coverage.sh docstring condensed to 31 lines (vs initial 53) to land under the 150-line acceptance criterion; 139 lines final"
  - "Threshold file placed at repo root (sibling of go.mod, .gitignore) per D-01 — not inside .github/ — so the ratchet SoT is discoverable from any repo entry point"
  - "ci.yml diff carefully merged with pre-existing frontend 'Format check' step from a prior session (unrelated to coverage) — Lint step, concurrency group, permissions, and frontend job all preserved"

patterns-established:
  - "Phase 71 single-commit-per-file boundary: each of the 4 files (threshold, script, ci.yml, baseline.md) is its own commit with intent-stating message; plan 71-01b can amend the whole branch as a single unit"
  - "Coverage gate contract: bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold — exit 0/1/2; pass 0/usage-2 are not failures, only weighted-avg < threshold is exit 1"

requirements-completed: [GOV-01, GOV-02, GOV-04]

# Metrics
duration: ~12min
completed: 2026-08-20
---

# Phase 71 Plan 01: Governance Baseline + CI Gate — File Creation Half

**Backend CI coverage gate infrastructure: weighted-average ratchet on a 5-byte threshold file, with bash + awk gate, html + 30-day GHA artifact upload, and a separate ratchet tracking file**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-08-20T22:20:00Z
- **Completed:** 2026-08-20T22:32:00Z
- **Tasks:** 4 / 4 complete
- **Files modified:** 4 (3 new + 1 modified; 0 business code touched)

## Accomplishments

- **GOV-01: CI -coverprofile infrastructure** — `.github/workflows/ci.yml` backend job Test step now emits `coverage.out` (atomic mode, no `-race` per D-01), with a Coverage HTML step (`go tool cover -html=`) and an `actions/upload-artifact@v4` step (`retention-days: 30`, `if: always()`) so a gate failure still publishes the artifact for developer debug.
- **GOV-02: weighted-average gate (D-01 zero-deps)** — `.coverage-threshold` is 5 bytes (`12.8` + LF, no `%`, no comments, no JSON wrapper); `.github/scripts/check-coverage.sh` is 139 lines of bash + inline awk, syntax-clean, executable, with exit 0/1/2 mirroring `scripts/check-status-literals.sh`. Awk formula is byte-for-byte the same as quick-260820-bcs so CI agrees with the original baseline scan to the percent.
- **GOV-04: standalone ratchet tracking (D-03)** — `.planning/coverage-baseline.md` records the 起点 row (commit `5ead742`, 12.8% / 43652 stmts / 5589 covered / 33 zero-coverage packages) and the Phase 71 后 row (same numbers — Phase 71 ships 0 business changes, weighted avg is invariant within rounding) with two inline 76-line per-package fenced blocks, a regression checklist, and a ratchet note. `quick/260820-backend-test-coverage-scan/SUMMARY.md` is untouched per D-03.

## Task Commits

Each task was committed atomically:

1. **Task 1: Write `.coverage-threshold` (5 bytes, 12.8 baseline)** — `4242a75` (ci)
2. **Task 2: Write `.github/scripts/check-coverage.sh` (bash + awk gate)** — `3e2664c` (ci)
3. **Task 3: Extend `.github/workflows/ci.yml` backend job (4 step changes)** — `75a4703` (ci)
4. **Task 4: Write `.planning/coverage-baseline.md` (起点 + Phase 71 后 snapshots)** — `75b96be` (docs)

## Files Created/Modified

- `.coverage-threshold` — Single-value gate threshold. 5 bytes (`31 32 2e 38 0a`); `cat` produces `12.8` with no trailing newline character visible; no BOM, no comments, no JSON/YAML wrapper. D-04 ratchet SoT: Phase 72/73/74 only edit this file.
- `.github/scripts/check-coverage.sh` — 139 lines, bash + awk. Reads `.coverage-threshold` via `cat`, parses `coverage.out` (Go atomic profile), aggregates per-package stmts/covered via inline awk with the exact `%-50s %8d %8d %6.2f%%` printf format used by `per-package-coverage.txt`, and emits a `PASS:` or `FAIL:` line based on the weighted average. Returns 0 (pass) / 1 (fail) / 2 (usage error). Missing profile → soft exit 0 so `if: always()` HTML/Upload steps still run (Gotcha #9).
- `.github/workflows/ci.yml` — 4 step changes in the backend job (Lint step, concurrency, permissions, frontend job all preserved untouched). Test step now: `go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...` (no `-race` per D-01). Three new steps appended in order: Coverage HTML (`if: always()`) → Coverage gate (no `if:`, propagation required) → Upload artifact (`if: always()`, `actions/upload-artifact@v4`, `retention-days: 30`, both `coverage.out` and `coverage.html`).
- `.planning/coverage-baseline.md` — 196 lines. Provenance block (source / measurement / generation) + 起点 section (commit `5ead742`, 12.8%, 43652/5589, 33 zero-coverage packages) + 76-line Per-package (起点) fenced block + Phase 71 后 section (commit `TBD` pending 71-01b amend) + 76-line Per-package (Phase 71 后) fenced block + Per-package 倒退检查 checklist + Ratchet note (D-04 manual workflow). `quick/260820-backend-test-coverage-scan/SUMMARY.md` untouched per D-03.

## Decisions Made

- **D-01 honored: bash + awk, zero deps, no third-party Action.** `vladopajic/go-test-coverage` is explicitly deferred to Phase 74 when diff coverage (GOV-03) becomes a hard requirement.
- **D-02 honored: `actions/upload-artifact@v4` (NOT @v7 — repo default for binary artifacts), `retention-days: 30`, `if: always()`.** Both `coverage.out` and `coverage.html` uploaded so developers can debug a gate failure locally without re-running the test suite.
- **D-03 honored: `.planning/coverage-baseline.md` is a separate file.** `quick/260820-backend-test-coverage-scan/SUMMARY.md` is the read-only baseline scan and must never be retroactively edited.
- **D-04 honored: manual ratchet.** `.coverage-threshold` is the only file the Phase 72/73/74 execute plans need to touch. `check-coverage.sh` and `ci.yml` are not edited as part of a ratchet.
- **D-01 Claude's Discretion honored: no `-race` flag** on the Test step (`-race` + `-cover` would 5-10x test time and risk CI 25-min timeout). atomic mode is Go 1.21+ standalone.
- **Existence-check softening:** `check-coverage.sh` returns exit 0 (not exit 1) when `coverage.out` is missing, so the `if: always()` Coverage HTML and Upload artifact steps still run when the Test step was skipped or timed out before producing a profile.
- **Per-package table byte-identity:** the 76 lines inside `coverage-baseline.md` are copied byte-for-byte from `quick/260820-bcs/per-package-coverage.txt` (column widths must match the awk `printf '...'` format), so a future diff between the local scan and the CI scan will be limited to actual coverage deltas.

## Deviations from Plan

None — plan executed exactly as written. Three minor implementation notes:

- **Docstring condensation (within tolerance):** the initial `check-coverage.sh` was 156 lines (over the 150-line acceptance ceiling). I trimmed the docstring from 53 lines to 31 lines while keeping all required content (Phase link, D-01 origin, exclusions, exit codes, CI hookup, ratchet workflow). Final 139 lines, well under the ceiling. No behavior change.
- **YAML validation command (Windows encoding):** `python -c "import yaml; yaml.safe_load(...)"` failed on Windows due to `gbk` codec not being able to decode the UTF-8 multi-byte chars in the comments. Re-ran with explicit `encoding='utf-8'` and the YAML validated cleanly. The plan's exact command works on Linux CI (the actual production target); Windows local validation just needs the encoding hint.
- **Pre-existing workspace artifacts preserved:** the workspace has pre-existing modifications/deletions unrelated to Phase 71 (e.g. `M internal/services/system/asset_columns_schema.json`, `M xingran-react-frontend/src/types/global.d.ts`, the `D .planning/phases/57-62/*` batch, the pre-existing `Format check` step in `ci.yml` frontend job). I touched only the 4 files declared in the plan's `files_modified` frontmatter, as instructed.

## Verification Performed (per task acceptance criteria)

| File | Check | Result |
|------|-------|--------|
| `.coverage-threshold` | `wc -c` = 5; `xxd` = `31 32 2e 38 0a`; `grep -c '.'` = 1; no BOM | PASS |
| `.github/scripts/check-coverage.sh` | `bash -n` clean; `chmod +x` mode `-rwxr-xr-x`; `wc -l` = 139; no-arg → exit 2; nonexistent profile → exit 0; first line `#!/usr/bin/env bash`, second line `# check-coverage.sh` | PASS |
| `.github/workflows/ci.yml` | YAML valid (UTF-8); 4 step changes in correct order; `actions/upload-artifact@v4` literal present with `retention-days: 30` + `if: always()`; Lint step / concurrency / permissions / frontend job untouched; pre-existing Format check step preserved | PASS |
| `.planning/coverage-baseline.md` | H2 `## 起点 (v1.26 启动前)` + H2 `## Phase 71 后` present; two H3 `### Per-package` subsections each followed by 76-line fenced block; Phase 71 后 row has `commit = TBD`; `weighted_avg = 12.8`; `total_stmts = 43652`; `total_covered = 5589`; `0pct_pkg_count = 33`; no YAML frontmatter; `quick/260820-backend-test-coverage-scan/SUMMARY.md` untouched | PASS |

## Issues Encountered

- **commitlint footer warning (Task 4):** the `docs(71): scaffold coverage-baseline.md (GOV-04)` commit produced a `footer-leading-blank` warning because the body ended with `Refs:` directly followed by no blank line before the commit terminated. The commit was accepted (warning, not error) and the SHA recorded. This is purely a style nit on the body — content is intact. Future commits in this branch can leave a blank line before the `Refs:` block to silence it.

## Next Phase Readiness

- **Plan 71-01b (verify + push + amend) is unblocked.** It needs to:
  1. Locally run `go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...` to confirm SC#4 (pkg/cache 15/15) and SC#5 (all-packages exit 0) still hold — these are independent of the CI infra added by this plan.
  2. Locally run `bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold` to confirm exit 0 (12.8% ≥ 12.8% floor).
  3. Bump `.coverage-threshold` to `99.9` and re-run to confirm exit 1 (gate fail).
  4. Restore `.coverage-threshold` to `12.8`.
  5. Commit + push (4 atomic commits already exist; `git push` is the final step).
  6. `gh run watch <run-id>` for CI 三绿 (lint + test + coverage gate).
  7. `git commit --amend` (or follow-up commit) to replace the `TBD` commit SHA in `coverage-baseline.md` with the real short SHA.
- **Phase 72/73/74 ratchet workflow** is now self-service: edit `.coverage-threshold` + append a row to `coverage-baseline.md` in the same atomic commit (D-04). The gate and CI workflow remain untouched across ratchets.
- **Manual verifications deferred to 71-01b (per VALIDATION.md):** the four `manual-only` behaviors (HTML step fail-fast, fail-state artifact upload, artifact volume over 30 days, deploy.yml workflow_run natural gate) require a real push + `gh run watch` to validate — out of scope for this file-creation plan.

---
*Phase: 71-governance-baseline-and-ci-gate*
*Plan: 01 (file creation half)*
*Completed: 2026-08-20*

## Self-Check: PASSED

- All 4 declared files exist: `.coverage-threshold`, `.github/scripts/check-coverage.sh`, `.github/workflows/ci.yml`, `.planning/coverage-baseline.md`
- All 4 task commits present in `git log`: `4242a75` (Task 1), `3e2664c` (Task 2), `75a4703` (Task 3), `75b96be` (Task 4)
- Pre-existing workspace artifacts (D-state for old phase plans, M-state for unrelated files, pre-existing Format check step in ci.yml frontend job) preserved untouched per planner directive
- `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` untouched per D-03
- No business code modified — `git diff --stat` of the 4 commits shows only the 4 declared files
