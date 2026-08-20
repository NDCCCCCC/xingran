---
phase: 71-governance-baseline-and-ci-gate
plan: 01b
type: execute
wave: 2
depends_on:
  - 71-01
files_modified:
  - .planning/coverage-baseline.md
autonomous: true
requirements:
  - GOV-01
  - GOV-02
  - GOV-04

must_haves:
  truths:
    - "PR fails when weighted coverage < .coverage-threshold (12.8) — verified by local smoke test (Task 1) + CI run (Task 4)"
    - "ci.yml backend job produces coverage.out + coverage.html artifacts retained 30 days — verified by gh run watch artifact check (Task 4)"
    - "go test ./pkg/cache/... passes 15/15 including TestAsyncRetryWorker_Enqueue — verified by Task 2"
    - "go test ./internal/... ./pkg/... ./cmd/... exits 0 (no failures) — verified by Task 3"
    - ".planning/coverage-baseline.md Phase 71 后 row's commit column is the actual short SHA of HEAD (not TBD) — verified by Task 4 amend"
  artifacts:
    - path: ".coverage-threshold"
      provides: "single-value gate threshold (12.8) — unchanged from plan 71-01"
      contains: "12.8\\n"
    - path: ".github/scripts/check-coverage.sh"
      provides: "gate script — unchanged from plan 71-01; exercised by Task 1 smoke test"
      exports: ["exit 0 (pass)", "exit 1 (gate fail)"]
    - path: ".github/workflows/ci.yml"
      provides: "extended backend job — unchanged from plan 71-01; verified by Task 4 CI run"
      contains: "actions/upload-artifact@v4"
    - path: ".planning/coverage-baseline.md"
      provides: "snapshot file — Phase 71 后 row commit column updated TBD → short SHA via Task 4 amend"
      contains: "## Phase 71 后"
  key_links:
    - from: "Task 1 smoke test"
      to: ".github/scripts/check-coverage.sh"
      via: "bash .github/scripts/check-coverage.sh /tmp/cov-phase71.out .coverage-threshold"
      pattern: "exit 0 at 12.8, exit 1 at 99.9"
    - from: "Task 4 CI run"
      to: ".github/workflows/ci.yml"
      via: "gh run watch + artifact download check"
      pattern: "backend-coverage artifact"
    - from: "Task 4 amend"
      to: ".planning/coverage-baseline.md"
      via: "git commit --amend --no-edit (TBD → short SHA)"
      pattern: "git rev-parse --short HEAD"
---

<objective>
Plan 71-01b is the verification + push + amend half of Phase 71: it exercises the infrastructure created by plan 71-01 (gate pass/fail smoke test, SC#4 pkg/cache regression guard, SC#5 full-suite exit 0), commits + pushes all four files, watches the resulting CI run to three-green, and amends the commit so the Phase 71 后 snapshot row's `commit` column reflects the actual SHA of the commit that produced the snapshot.

Purpose: this plan stops at "CI green on `origin/main`, artifact downloadable, baseline.md points at the real SHA". Verification happens BEFORE push (Tasks 1-3) so a local failure aborts before polluting CI history. The amend pattern (Task 4) resolves the chicken-and-egg of "the snapshot row's `commit` column needs the SHA of the commit that creates the snapshot".

D-01..D-04 honored: bash + awk gate (D-01), `actions/upload-artifact@v4` + 30-day retention (D-02), separate `coverage-baseline.md` (D-03), manual ratchet (D-04). Zero business changes.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/phases/71-governance-baseline-and-ci-gate/71-CONTEXT.md
@.planning/phases/71-governance-baseline-and-ci-gate/71-RESEARCH.md
@.planning/phases/71-governance-baseline-and-ci-gate/71-VALIDATION.md
@.planning/phases/71-governance-baseline-and-ci-gate/71-PATTERNS.md
@.planning/phases/71-governance-baseline-and-ci-gate/71-01-PLAN.md (upstream plan — files this plan exercises + commits)
@.github/workflows/ci.yml (file this plan pushes; verify diff is as expected before commit)
@.planning/coverage-baseline.md (file this plan amends in Task 4)
@.planning/quick/260820-backend-test-coverage-scan/per-package-coverage.txt
@.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md
</context>

<interfaces>
<!-- Key contracts/values the executor must use verbatim. -->

From `.planning/phases/71-governance-baseline-and-ci-gate/71-RESEARCH.md` (Gotchas & Risks, lines 365-432):

- `-count=1` mandatory (prevents test-result caching); `-covermode=atomic` mandatory (correct for concurrent tests per Go docs); `-race` must NOT be present (would 5-10x test runtime)
- Gate stays pass when no profile exists (script exits 0 if `coverage.out` missing — `if: always()` on downstream steps)
- `actions/upload-artifact@v4` (NOT @v7 — D-02 explicit)

From `.planning/phases/71-governance-baseline-and-ci-gate/71-RESEARCH.md` (CI Workflow Changes, lines 159-189):

- Step order invariant: Test → Coverage HTML (`if: always()`) → Coverage gate (no `if:`) → Upload artifact (`if: always()`)
- Coverage HTML step `if: always()` + `[ -f coverage.out ]` check so it skips gracefully when no profile exists

From memory `push-watch-ci.md`:

After push, use `gh run watch --exit-status` in the background to monitor CI; capture run ID + exit status; the gate is what Phase 71 specifically tests (lint + test failures are out of Phase 71 scope, but Coverage gate MUST be green).
</interfaces>

<tasks>

<task type="auto">
  <name>Task 1: Local smoke test — generate coverage.out, verify gate exit 0/1 behaviour</name>
  <files>.coverage-threshold, .github/scripts/check-coverage.sh, /tmp/cov-phase71.out</files>
  <read_first>
    - .coverage-threshold (Task 1 from plan 71-01 output — current threshold value 12.8)
    - .github/scripts/check-coverage.sh (Task 2 from plan 71-01 output — script under test)
    - .planning/quick/260820-backend-test-coverage-scan/per-package-coverage.txt (reference 76-row table — smoke test result must match this within rounding tolerance)
    - .planning/quick/260820-backend-test-coverage-scan/SUMMARY.md (line 18 — 12.8% reference number)
    - .planning/phases/71-governance-baseline-and-ci-gate/71-RESEARCH.md (Gotchas & Risks section lines 365-432 — -count=1 must be present; -covermode=atomic required for concurrent tests; -race must NOT be present)
  </read_first>
  <action>
    Run three local validations on the executor's machine (Bash + Git Bash on Windows is acceptable; awk must be gawk or BWK with float semantics):

    1. Generate a fresh coverage profile — invoke `go test -timeout 15m -count=1 -coverprofile=/tmp/cov-phase71.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...` from the repo root. The `-count=1` flag is mandatory (prevents test-result caching); `-covermode=atomic` is mandatory (correct for concurrent tests per Go docs); `-race` must NOT be present (would 5-10x test runtime per RESEARCH.md line 380-384). Capture exit code — must be 0 (no failures).

    2. Validate gate passes at the current threshold — invoke `bash .github/scripts/check-coverage.sh /tmp/cov-phase71.out .coverage-threshold`. Capture stdout (must contain per-package rows + final `PACKAGE <stmts> <covered> <pct>%` summary line) and exit code (must be 0). Verify the printed `PACKAGE` line's `<stmts>` is ~43652, `<covered>` is ~5589, `<pct>` is `12.80` or `12.81` (matches quick-260820-bcs baseline within rounding). If the numbers diverge from 43652 / 5589 / 12.80, log the divergence in the task summary but proceed (a small drift is acceptable — the exact reproduction is only meaningful if no business code changed since the 2026-08-20 scan).

    3. Validate gate fails when threshold is artificially raised — `cp .coverage-threshold .coverage-threshold.bak` then `printf '99.9\n' > .coverage-threshold`. Re-invoke `bash .github/scripts/check-coverage.sh /tmp/cov-phase71.out .coverage-threshold`. Exit code MUST be 1 (not 0). Stderr MUST contain a `FAIL:` message naming both the actual weighted avg and the threshold (`99.9`). Then restore: `mv .coverage-threshold.bak .coverage-threshold` and re-invoke once more to confirm exit 0 again. If the file restoration fails for any reason, leave the threshold at 12.8 (the correct value) and abort with a loud error message.

    These three local validations gate the commit + push in Task 4. If any fails, do NOT proceed to Task 2 — fix the script first.
  </action>
  <acceptance_criteria>
    - `go test -timeout 15m -count=1 -coverprofile=/tmp/cov-phase71.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...` exits 0
    - `bash .github/scripts/check-coverage.sh /tmp/cov-phase71.out .coverage-threshold` exits 0, prints per-package rows matching `per-package-coverage.txt` column widths, final `PACKAGE` line shows ~43652 stmts and ~5589 covered
    - With `.coverage-threshold` temporarily set to `99.9`, the script exits 1 and stderr contains both the actual weighted avg and `99.9`
    - After restoring `.coverage-threshold` to `12.8`, re-invoking the script again exits 0 (proves restoration succeeded)
    - `/tmp/cov-phase71.out` exists and is non-empty (`ls -la /tmp/cov-phase71.out` shows size > 1MB; Go atomic coverage profile of 74 packages is ~3MB)
    - `.coverage-threshold` is exactly `12.8\n` after the test completes (verify with `cat` + `xxd`)
  </acceptance_criteria>
  <verify>
    <automated>go test -timeout 15m -count=1 -coverprofile=/tmp/cov-phase71.out -covermode=atomic ./internal/... ./pkg/... ./cmd/... && echo "--- gate pass test ---" && bash .github/scripts/check-coverage.sh /tmp/cov-phase71.out .coverage-threshold | tail -3 && echo "--- gate fail test ---" && cp .coverage-threshold .coverage-threshold.bak && printf '99.9\n' > .coverage-threshold && (bash .github/scripts/check-coverage.sh /tmp/cov-phase71.out .coverage-threshold; echo "exit=$?"); mv .coverage-threshold.bak .coverage-threshold && echo "--- restored ---" && cat .coverage-threshold && bash .github/scripts/check-coverage.sh /tmp/cov-phase71.out .coverage-threshold > /dev/null && echo "restored gate exit 0 OK"</automated>
  </verify>
  <done>
    Local smoke test green: gate passes at 12.8% with the existing weighted average, gate fails at 99.9%, and restoration leaves `.coverage-threshold` at exactly `12.8` with the gate still passing. The local coverage profile `/tmp/cov-phase71.out` exists for use as the basis for the Phase 71 后 snapshot in Task 4's amend.
  </done>
</task>

<task type="auto">
  <name>Task 2: SC#4 verification — pkg/cache TestAsyncRetryWorker_Enqueue passes (5ead742 regression guard)</name>
  <files></files>
  <read_first>
    - pkg/cache/retry_test.go (the test file containing TestAsyncRetryWorker_Enqueue — verify it still asserts the post-fix behavior, not the original racy assertion)
    - .planning/STATE.md (line 144 — quick-260820-far entry: "竞态断言改非负 + 新增 TestAsyncRetryWorker_QueueSemantics"; commit `5ead742` is the flaky fix)
    - .planning/quick/260820-fix-async-retry-worker-flaky-test/SUMMARY.md (if exists — documents the exact fix)
  </read_first>
  <action>
    Run the SC#4 verification per Phase 71 SUCCESS CRITERIA #4 and the 71-VALIDATION.md Per-Task Verification Map row 71-01-08.

    Invoke `go test -v -count=1 ./pkg/cache/...` from the repo root. The `-v` flag surfaces individual test names so the executor can confirm `TestAsyncRetryWorker_Enqueue` runs and passes (not just "all 15 passed" — must explicitly see `--- PASS: TestAsyncRetryWorker_Enqueue` in the output). The `-count=1` flag prevents Go's test-result cache from masking a real failure if the test source hasn't changed since the last run.

    Capture the full output. Expected: 15 tests pass (including `TestAsyncRetryWorker_Enqueue` and `TestAsyncRetryWorker_QueueSemantics` per quick-260820-far). Exit code must be 0. If `TestAsyncRetryWorker_Enqueue` is missing or fails, abort with a loud error and reference the `5ead742` fix — this means the flaky test has regressed and must be re-fixed before Phase 71 can ship.

    Do NOT modify `retry_test.go` in this task. This is pure verification.
  </action>
  <acceptance_criteria>
    - `go test -v -count=1 ./pkg/cache/...` exits 0
    - Output contains `--- PASS: TestAsyncRetryWorker_Enqueue` (explicit, not just "ok" line)
    - Output contains `--- PASS: TestAsyncRetryWorker_QueueSemantics` (the post-fix non-racy test added in 5ead742)
    - Total PASS count is 15 (or matches the current file's test count if upstream added more)
    - Output contains `FAIL` zero times
    - `pkg/cache` line in output ends with `ok` (not `FAIL`)
  </acceptance_criteria>
  <verify>
    <automated>go test -v -count=1 ./pkg/cache/... 2>&1 | tail -25 && echo "--- AsyncRetryWorker test count ---" && go test -v -count=1 ./pkg/cache/... 2>&1 | grep -E "PASS:|FAIL:" | grep -c AsyncRetryWorker && echo "async tests listed"</automated>
  </verify>
  <done>
    SC#4 verified: `pkg/cache` test suite (15 tests including `TestAsyncRetryWorker_Enqueue` and `TestAsyncRetryWorker_QueueSemantics`) passes cleanly. The `5ead742` flaky test fix is intact. Phase 71 SC#4 is satisfied.
  </done>
</task>

<task type="auto">
  <name>Task 3: SC#5 verification — full-package test suite exits 0 (no failures)</name>
  <files></files>
  <read_first>
    - .github/workflows/ci.yml (line 57 — canonical Test step command; matches RESEARCH.md's "Full suite command")
    - .planning/quick/260820-backend-test-coverage-scan/SUMMARY.md (lines 32, 49 — confirms this is the standard full-package command; known pre-scan test failure was pkg/cache.TestAsyncRetryWorker_Enqueue already fixed in 5ead742)
  </read_first>
  <action>
    Run the SC#5 verification per Phase 71 SUCCESS CRITERIA #5 and the 71-VALIDATION.md Per-Task Verification Map row 71-01-09.

    Invoke `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` from the repo root. The command matches the canonical ci.yml Test step (line 57) and the 71-RESEARCH.md Full suite command exactly. The `-timeout 15m` flag guards against the network/ports 3 timeout advisory from Phase 63 advisory backlog (real failures within timeout budget indicate actual regressions; timeouts indicate environment issues and should be flagged separately). The `-count=1` flag forces fresh runs (no cache).

    Capture exit code (must be 0) and the trailing output summary. If exit is non-zero, parse the output for `FAIL` markers and abort with the failure list — do not silently proceed. Known pre-Phase-71 failures per quick-260820-bcs scan were `pkg/cache.TestAsyncRetryWorker_Enqueue` (fixed in 5ead742, verified in Task 2) and potentially `login_encryption_test.go:235` + `pkg/errors/errors_test.go` + operations subtree (pre-existing from Phase 51-54 window per PROJECT.md line 171 — Phase 71 does not fix these; if they surface here, that is a regression and must be investigated before commit).

    Do NOT modify any test file in this task. This is pure verification.
  </action>
  <acceptance_criteria>
    - `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` exits 0
    - Output contains `FAIL` zero times
    - All packages end with `ok` lines (no `--- FAIL` anywhere)
    - Total packages reported matches the package count from Phase 63 ci.yml baseline (74 business packages + helper sub-packages; count may have grown slightly since Phase 63)
    - No `panic:` lines in output
    - Wall time < 15 minutes (the -timeout 15m per-package ceiling — total wall time should be ~5-10 minutes based on Phase 63 historical data)
  </acceptance_criteria>
  <verify>
    <automated>go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/... 2>&1 | tee /tmp/full-test-phase71.log | tail -10 && echo "--- fail grep ---" && grep -c "^FAIL" /tmp/full-test-phase71.log && echo "--- panic grep ---" && grep -c "^panic" /tmp/full-test-phase71.log</automated>
  </verify>
  <done>
    SC#5 verified: full backend test suite (`./internal/... ./pkg/... ./cmd/...`) exits 0 with no failures, no panics, and within the 15-minute wall-clock budget. Phase 71 SC#5 is satisfied. Combined with Task 2 (SC#4), both flaky-test-related SC are green.
  </done>
</task>

<task type="auto">
  <name>Task 4: Commit + push + gh run watch (CI three green) + amend baseline.md commit SHA</name>
  <files>.coverage-threshold, .github/scripts/check-coverage.sh, .github/workflows/ci.yml, .planning/coverage-baseline.md</files>
  <read_first>
    - .planning/STATE.md (line 143 — commit `fdefdefd` is the latest; verify the working tree is otherwise clean before adding Phase 71 files)
    - .planning/notes/ (if any pre-existing notes — verify no conflicts with current changes)
    - .planning/phases/71-governance-baseline-and-ci-gate/71-CONTEXT.md (D-04 — manual ratchet, no CI auto-bump; commit message must indicate Phase 71 origin)
    - .github/workflows/ci.yml (plan 71-01 output — the file being pushed; verify diff is as expected before commit)
    - .planning/coverage-baseline.md (plan 71-01 Task 4 output — the file being pushed)
    - Memory note `push-watch-ci.md` (per user memory — use `gh run watch` to monitor CI in background)
  </read_first>
  <action>
    Five steps in order:

    1. Stage the 4 files — `git add .coverage-threshold .github/scripts/check-coverage.sh .github/workflows/ci.yml .planning/coverage-baseline.md`. Run `git status` to verify exactly these 4 files are staged (no stray workspace artifacts like `temp_*.go` or `test_*.go` per CLAUDE.md "Temporary Files Cleanup" — abort if found). If `M internal/services/system/asset_columns_schema.json` (the pre-existing Phase 68 leftover per STATE.md line 200) is dirty, do NOT touch it; just confirm it's not staged.

    2. Commit — `git commit -m "ci(71): backend coverage gate + baseline snapshot

    GOV-01: -coverprofile=coverage.out -covermode=atomic -count=1 in Test step
    GOV-02: bash + awk weighted-average gate reading .coverage-threshold (12.8)
    GOV-04: .planning/coverage-baseline.md with starting + Phase 71 后 snapshot

    Phase 71 ships 0 business changes. Gate fails PRs whose weighted avg
    drops below 12.8%; ratchet is manual via .coverage-threshold edits
    in Phase 72/73/74 execute plans (per D-04).

    Coverage artifact uploaded 30 days via actions/upload-artifact@v4
    for local debugging on gate failure."`. Verify the commit with `git log -1 --stat` showing all 4 files (3 NEW + 1 modified for ci.yml).

    3. Push — `git push origin main`. Capture the push output for the upstream branch ref. If push is rejected (e.g., branch protection), do NOT force-push; report the rejection.

    4. Watch CI — immediately after push, invoke `gh run watch --exit-status` in the background (per memory `push-watch-ci.md`). Capture the run ID and exit status. The CI must show three green jobs:
       - `backend` (golangci-lint pass + go test pass + Coverage gate pass)
       - `frontend` (lint + type-check + vitest + build pass)
       - the artifact `backend-coverage` is downloadable from the PR page

       If backend fails on the Coverage gate (which would be unexpected since 12.8 >= 12.8), abort and investigate. If frontend fails, that's pre-existing and out of Phase 71 scope — note but do not block Phase 71 ship. If backend fails on lint or test, those are pre-existing issues — note but the Coverage gate is what Phase 71 specifically tests.

    5. Amend — after CI is green, run `git rev-parse --short HEAD` and Edit `.planning/coverage-baseline.md` Phase 71 后 row to replace `TBD` with the short SHA. Then `git add .planning/coverage-baseline.md && git commit --amend --no-edit` (amend the same commit so the SHA on the snapshot row matches the SHA that produced the snapshot). Then `git push --force-with-lease origin main` (safer than `--force`) to push the amended commit.

    Do NOT mark Phase 71 complete until all three CI jobs are green AND the amended commit is pushed.
  </action>
  <acceptance_criteria>
    - `git status` after the final amend shows clean working tree
    - `git log -1 --stat` shows all 4 files in the latest commit: `.coverage-threshold` (new), `.github/scripts/check-coverage.sh` (new), `.github/workflows/ci.yml` (modified), `.planning/coverage-baseline.md` (new)
    - The commit message contains the strings `GOV-01`, `GOV-02`, `GOV-04`, `Phase 71`, `coverage gate`, `12.8` (verifying the intent statement)
    - `.planning/coverage-baseline.md` Phase 71 后 row's `commit` column is the actual short SHA of HEAD (not `TBD`), matching `git rev-parse --short HEAD`
    - `git push --force-with-lease origin main` succeeded (pushed commit is reachable from `origin/main`)
    - `gh run watch` for the run triggered by the push exited 0 (or the equivalent exit-status check passed)
    - `gh api repos/:owner/:repo/actions/runs/<run-id>/artifacts` lists `backend-coverage` artifact (or the artifact appears in the GitHub UI PR page)
    - The three GitHub Actions jobs all reported success: backend (lint + test + gate), frontend (lint + type-check + test + build)
  </acceptance_criteria>
  <verify>
    <automated>git status && echo "--- log ---" && git log -1 --stat && echo "--- coverage-baseline commit field ---" && grep -E "^\| 2026-08-20 \| Phase 71" .planning/coverage-baseline.md && echo "--- pushed ---" && git log --oneline -3 origin/main && echo "--- gh runs (latest 3) ---" && gh run list --limit 3 --json status,conclusion,name,headBranch 2>&1 | head -30</automated>
  </verify>
  <done>
    Phase 71 is shipped: 4 files committed (3 NEW + 1 modified), pushed to `main`, CI shows three green jobs (backend: lint + test + Coverage gate pass; frontend: lint + type-check + test + build pass), `backend-coverage` artifact is downloadable from the PR page, and `.planning/coverage-baseline.md` references the actual commit SHA in its Phase 71 后 row. The coverage gate is now live for every PR — backsliding below 12.8% is impossible without overriding the threshold file (which is itself a reviewable git diff per D-04).
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| PR author → CI runner | Untrusted PR code runs `go test` and produces `coverage.out`; the gate decides pass/fail |
| Local executor → `.coverage-threshold` | The threshold file is edited by humans (D-04); integrity enforced by git history + PR review |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-71-01 | Tampering | `.coverage-threshold` | mitigate | Single source of truth at repo root, all edits visible as PR diffs (D-04); manual ratchet prevents silent threshold changes; commit history makes every bump auditable |
| T-71-02 | Tampering | `check-coverage.sh` | mitigate | Script lives under `.github/scripts/` (only maintainers edit); awk formula is identical to quick-260820-bcs (verified 12.8%); Task 1 local smoke test confirms exit 0/1 behaviour before push |
| T-71-03 | Information Disclosure | `backend-coverage` artifact (coverage.out + coverage.html) | accept | Coverage profile is non-sensitive (test instrumentation, no business secrets); 30-day retention is a CI cost decision not a privacy one |
| T-71-04 | Elevation of Privilege | Coverage gate fail → `if: always()` artifact upload | accept | Fail-state artifact upload is intentional for developer debugging; gate fail cannot bypass security checks (lint + test still gate deploy via workflow_run) |
| T-71-05 | Denial of Service | CI resource exhaustion via large coverage artifacts | mitigate | `actions/upload-artifact@v4` has a 1GB per-repo cap; estimated 150 artifacts × 8MB = 1.2GB approaching cap; RESEARCH §Gotchas #11 flags this for Phase 74+ re-evaluation |
| T-71-06 | Repudiation | Phase 71 threshold changes | mitigate | Every `.coverage-threshold` edit is a separate git commit; git log + commit message includes ratchet before/after numbers (per D-04 + 71-RESEARCH §Ratchet Workflow) |
| T-71-SC | Tampering | npm/pip/cargo installs | n/a | Phase 71 is engineering infrastructure, not package installation — no new dependencies introduced (D-01 zero-dep mandate) |

## Notes

- This threat model is intentionally thin because Phase 71 is engineering infrastructure (CI shell + workflow YAML), not business code. The "security boundary" is PR-author → CI-runner, and the existing `permissions: contents: read` block + step-level `actions/upload-artifact@v4` are the primary controls.
- No new network exposure introduced (gate runs entirely in-process on the CI runner).
- No new data exposure — coverage profile is generated from existing test code and business code that's already in the repo.
</threat_model>

<verification>
[Plan 71-01b checks — verification + push + amend]

- [ ] Local smoke test (Task 1) green: gate passes at 12.8%, fails at 99.9%, restores cleanly
- [ ] `go test ./pkg/cache/...` passes 15/15 including `TestAsyncRetryWorker_Enqueue` (SC#4)
- [ ] `go test ./internal/... ./pkg/... ./cmd/...` exits 0 (SC#5)
- [ ] 4 files committed in a single commit with intent-stating message
- [ ] Pushed to `origin/main`
- [ ] CI shows three green jobs (backend + frontend + artifact present)
- [ ] `.planning/coverage-baseline.md` Phase 71 后 row's `commit` field updated TBD → short SHA via amend
- [ ] Amend pushed via `--force-with-lease`
- [ ] No business code modified — `git diff <previous-HEAD>..HEAD -- '*.go'` returns only ci.yml + the 3 new files, no business logic touched
</verification>

<success_criteria>
[Plan 71-01b measurable completion — finishes the Phase 71 goal]

- [ ] **SC#1**: `ci.yml` backend job contains `-coverprofile=coverage.out -covermode=atomic -count=1` and produces `coverage.out` + `coverage.html` artifacts uploaded via `actions/upload-artifact@v4` with `retention-days: 30`. Verified by `git diff` + `gh run watch`.
- [ ] **SC#2**: CI introduces a weighted-average gate. Threshold = 12.8 (from `.coverage-threshold`). Below-threshold PR fails. Ratchet mechanism: future phases (72/73/74) edit `.coverage-threshold` + append a `.planning/coverage-baseline.md` row in the same atomic commit. Verified by local smoke test (Task 1: bump to 99.9 → exit 1) + CI run.
- [ ] **SC#3**: `.planning/coverage-baseline.md` contains both a starting row (from quick-260820-bcs, commit `5ead742`) and a Phase 71 后 row (commit = amended SHA), each followed by a 76-line per-package table inline. Verified by `cat` + grep.
- [ ] **SC#4**: `go test -v -count=1 ./pkg/cache/...` exits 0 with `--- PASS: TestAsyncRetryWorker_Enqueue` explicitly in output (5ead742 guard). Verified in Task 2.
- [ ] **SC#5**: `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` exits 0 with no failures, no panics. Verified in Task 3.

**Requirements coverage:**
- GOV-01 (CI -coverprofile + profile artifact): Task 4 CI verification
- GOV-02 (CI threshold gate + ratchet): Tasks 1 + 4 (local smoke + CI run)
- GOV-04 (Baseline snapshot with amended SHA): Task 4 amend

**Decisions honored:**
- D-01 (bash + awk + `.coverage-threshold`): verified by Task 1 smoke test
- D-02 (actions/upload-artifact@v4 + 30 days): verified by Task 4 artifact check
- D-03 (`.coverage-baseline.md` separate from quick scan): verified by `git diff` on quick scan SUMMARY.md
- D-04 (manual ratchet): plan-level (Phase 72/73/74 execute plans will bump)

All 5 SC, 3 requirements, 4 D-locked decisions covered. Phase 71 ships when all 4 tasks complete + plan 71-01 ships.
</success_criteria>

<output>
Create `.planning/phases/71-governance-baseline-and-ci-gate/71-01b-SUMMARY.md` when done
</output>
