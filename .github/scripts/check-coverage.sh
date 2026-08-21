#!/usr/bin/env bash
# check-coverage.sh — Phase 71 backend coverage ratchet gate (GOV-02).
#
# Reads coverage.out (Go atomic profile) and .coverage-threshold (e.g. 12.8),
# aggregates per-package stmts/covered via inline awk, exits non-zero when
# the weighted average drops below the threshold. Origin: D-01 (bash + awk,
# zero deps) — formula is byte-for-byte the same as the quick-260820-bcs
# baseline scan so CI and the original snapshot agree to the percent.
#
# Exclusions (matching quick-260820-bcs): scripts/, tests/scripts/, node_modules/.
#
# Usage:
#   bash .github/scripts/check-coverage.sh <profile> [<threshold-file>]
#
# Exit codes (mirror scripts/check-status-literals.sh):
#   0 — weighted average >= threshold AND all 8 P1 packages >= 70% (gate passes)
#   1 — weighted average <  threshold (gate fails) OR awk parse error
#   2 — usage error (missing args / unreadable files)
#   4 — Phase 73 P1 per-package floor failure (one of the 8 P1 packages < 70%)
#
# CI hookup (ci.yml, step order invariant: Test -> Coverage HTML -> Coverage
# gate -> Upload artifact):
#   - name: Coverage gate
#     run: bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold
#
# Ratchet workflow (D-04 — manual): Phase 72/73/74 execute plans end with a
# deliberate commit bumping .coverage-threshold AND appending a row to
# .planning/coverage-baseline.md. Phase 73 (Plan 73-05) additionally extended
# this script with the P1 per-package floor (section 3 below); ci.yml still
# invokes the same command — no workflow edits needed per ratchet.
#
# P1 per-package floor (Phase 73, IMP-01..06 / D-04 strict / D-10): the 8 P1
# packages (internal/api/v1/{duty,knowledge,rpa,vdi} +
# internal/services/{duty,knowledge,network,monitor}) must EACH stay >= 70.0%.
# This is deliberately stricter than the weighted average — a P1 package
# cannot regress below the floor even if the overall average rises.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

PROFILE="${1:-}"
THRESHOLD_FILE="${2:-}"

if [ -z "$PROFILE" ] || [ -z "$THRESHOLD_FILE" ]; then
  echo "usage: $0 <coverage-profile> [<threshold-file>]" >&2
  echo "" >&2
  echo "  <profile>         path to coverage.out (e.g. ./coverage.out)" >&2
  echo "  <threshold-file>  path to threshold file (e.g. ./.coverage-threshold)" >&2
  echo "" >&2
  echo "Default threshold file if omitted: .coverage-threshold (at repo root)" >&2
  exit 2
fi

# Anchored threshold path: always read the file at the repo root regardless of
# the caller's cwd. ci.yml invokes this from the workspace root so both work.
case "$THRESHOLD_FILE" in
  /*) THRESHOLD_PATH="$THRESHOLD_FILE" ;;
  *)  THRESHOLD_PATH="$ROOT/$THRESHOLD_FILE" ;;
esac

if [ ! -f "$THRESHOLD_PATH" ]; then
  echo "check-coverage.sh: threshold file $THRESHOLD_PATH not found" >&2
  exit 2
fi

THRESHOLD="$(cat "$THRESHOLD_PATH")"

# Fail-safe: when no profile exists (Test step skipped or coverage not generated),
# don't fail the job — the upstream `if: always()` on Coverage HTML and Upload
# steps still want to run for debug. This mirrors CI semantics: missing profile
# is a soft skip, not a hard gate failure.
if [ ! -f "$PROFILE" ]; then
  echo "check-coverage.sh: coverage profile $PROFILE missing — was Test step skipped?" >&2
  echo "check-coverage.sh: skipping gate (exit 0) so HTML/Upload steps can still run with if: always()" >&2
  exit 0
fi

# --- 1. Per-package aggregation table (printed to stdout for CI logs) ---------
# Format must match .planning/quick/260820-backend-test-coverage-scan/per-package-coverage.txt
# exactly: "%-50s %8d %8d %6.2f%%" so the same awk output is diff-friendly across
# local scans and CI runs.

AWK_TABLE=$(awk -v threshold="$THRESHOLD" '
NR > 1 {
    split($1, parts, ":")
    n = split(parts[1], seg, "/")
    pkg = ""
    for (i = 4; i <= n - 1; i++) {
        pkg = (pkg == "") ? seg[i] : pkg "/" seg[i]
    }
    if (pkg ~ /^scripts\//) next
    if (pkg ~ /^tests\/scripts\//) next
    if (pkg ~ /^node_modules\//) next
    if (pkg == "") next
    num_stmts  = $2 + 0
    hit_count  = $3 + 0
    covered    = (hit_count > 0) ? num_stmts : 0
    biz_stmts[pkg]   += num_stmts
    biz_covered[pkg] += covered
}
END {
    total_s = 0
    total_c = 0
    for (k in biz_stmts) {
        s = biz_stmts[k]
        c = biz_covered[k]
        pct = (s > 0) ? c * 100.0 / s : 0
        printf "%-50s %8d %8d %6.2f%%\n", k, s, c, pct
        total_s += s
        total_c += c
    }
    pkg_pct = (total_s > 0) ? total_c * 100.0 / total_s : 0
    printf "PACKAGE %8d %8d %6.2f%%\n", total_s, total_c, pkg_pct
    if (pkg_pct + 0 < threshold + 0) {
        printf "FAIL: weighted avg %.2f%% < threshold %.2f%%\n", pkg_pct, threshold + 0
        exit 1
    } else {
        printf "PASS: weighted avg %.2f%% >= threshold %.2f%%\n", pkg_pct, threshold + 0
        exit 0
    }
}
' "$PROFILE" || true)

AWK_EXIT=$?
echo "$AWK_TABLE"

# --- 2. Gate result ----------------------------------------------------------
# awk above embeds the threshold check + exits 0/1. Surface a clean CI message
# and propagate the exit code. We re-check the table tail to print a FAIL block
# for the developer even when awk exited 0 (defensive — keeps the log readable).

if [ "$AWK_EXIT" -ne 0 ]; then
  echo "" >&2
  echo "coverage gate FAILED — weighted average below $THRESHOLD%" >&2
  echo "Bump .coverage-threshold only if this is an intentional ratchet (D-04)," >&2
  echo "otherwise add tests to lift the weighted average back above the floor." >&2
  exit 1
fi

# If awk succeeded but the trailing line is not the PASS line we expect, treat
# it as a parse error (defensive — e.g. an empty profile slipped through).
if ! echo "$AWK_TABLE" | grep -qE '^PASS:'; then
  echo "check-coverage.sh: profile parsed but no PASS line emitted — treating as failure" >&2
  exit 1
fi

echo "coverage gate passed (threshold=$THRESHOLD%)"

# --- 3. p1_package_check: Phase 73 P1 per-package floor (8 packages >= 70%) --
# awk parses coverage.out (atomic profile, format: import/path/file.go:loc,loc
# stmts count) and aggregates stmts/covered per package using the SAME split
# logic as section 1, so both blocks agree to the percent.

P1_FLOOR="70.0"
P1_PACKAGES="internal/api/v1/duty internal/api/v1/knowledge internal/api/v1/rpa internal/api/v1/vdi internal/services/duty internal/services/knowledge internal/services/network internal/services/monitor"

P1_PKG_TABLE=$(awk '
NR > 1 {
    split($1, parts, ":")
    n = split(parts[1], seg, "/")
    pkg = ""
    for (i = 4; i <= n - 1; i++) {
        pkg = (pkg == "") ? seg[i] : pkg "/" seg[i]
    }
    if (pkg == "") next
    num_stmts = $2 + 0
    hit_count = $3 + 0
    covered   = (hit_count > 0) ? num_stmts : 0
    p1_stmts[pkg]   += num_stmts
    p1_covered[pkg] += covered
}
END {
    for (k in p1_stmts) {
        printf "%s %d %d\n", k, p1_stmts[k], p1_covered[k]
    }
}
' "$PROFILE")

P1_FAILED=0
for pkg in $P1_PACKAGES; do
  line="$(printf '%s\n' "$P1_PKG_TABLE" | awk -v p="$pkg" '$1 == p { print $2, $3; exit }')"
  if [ -z "$line" ]; then
    echo "FAIL: P1 $pkg not found in profile — no statements measured for this package" >&2
    P1_FAILED=$((P1_FAILED + 1))
    continue
  fi
  stmts="${line%% *}"
  covered="${line##* }"
  pct="$(awk -v s="$stmts" -v c="$covered" 'BEGIN { printf "%.2f", (s + 0 > 0) ? c * 100.0 / s : 0 }')"
  if awk -v a="$pct" -v b="$P1_FLOOR" 'BEGIN { exit !(a + 0 >= b + 0) }'; then
    echo "PASS: P1 $pkg $pct% >= $P1_FLOOR% ($covered/$stmts stmts)"
  else
    echo "FAIL: P1 $pkg $pct% < $P1_FLOOR% ($covered/$stmts stmts)" >&2
    P1_FAILED=$((P1_FAILED + 1))
  fi
done

if [ "$P1_FAILED" -ne 0 ]; then
  echo "" >&2
  echo "coverage gate FAILED — $P1_FAILED P1 package(s) below the $P1_FLOOR% floor" >&2
  echo "P1 floor is from Phase 73 (IMP-01..06, D-04 strict + D-10): a P1 package" >&2
  echo "cannot regress below 70% even when the weighted average stays above the" >&2
  echo "threshold. Add tests for the failing package(s) to lift it back." >&2
  exit 4
fi

echo "P1 per-package floor passed ($P1_FLOOR% x 8 packages)"
exit 0
