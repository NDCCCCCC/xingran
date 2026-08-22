#!/usr/bin/env bash
# check-diff-coverage.sh — Phase 74 GOV-03 PR diff coverage gate (D-14 fallback).
#
# D-14 locked an external action (gocover-coverage / ORY xcoverage-action) as
# the preferred tool, with an explicit ratchet clause: if the locked action
# turns out unsuitable or unmaintained, the executor MAY fall back to a
# self-implemented solution (git diff + awk + coverage.out parsing) as long as
# the GOV-03 >=80% threshold is enforced 100%.
#
# Executor finding (2026-08-22): "gocover-coverage" has no verifiable
# marketplace presence and "ory/xcoverage-action" does not exist at all.
# Rather than gamble on an unvetted third-party action for a hard merge gate,
# this script implements the gate in-repo with the same zero-dep bash+awk
# paradigm as check-coverage.sh (D-01) — auditable, no supply-chain surface.
#
# Semantics:
#   diff coverage = changed executable .go lines covered by tests / all
#                   changed executable .go lines in the PR
#   - changed lines: `+` lines from `git diff --unified=0 <base>...HEAD`
#     (three-dot = merge-base), restricted to *.go, excluding *_test.go
#   - executable: blank lines and // comment-only lines excluded
#   - covered: the changed line falls inside a coverage.out block whose
#     hit count > 0 (block granularity — matches go tool cover semantics)
#
# Usage:
#   bash .github/scripts/check-diff-coverage.sh <coverage.out> <base-ref> [threshold]
#
# Exit codes (mirror check-coverage.sh):
#   0 — diff coverage >= threshold, OR no testable .go lines changed (skip)
#   1 — diff coverage < threshold (gate fails)
#   2 — usage error / missing inputs / diff failed
#
# CI hookup (ci.yml, separate PR-only job depending on backend artifact):
#   coverage-diff:
#     needs: backend
#     if: github.event_name == 'pull_request'
#     steps:
#       - uses: actions/checkout@v7
#         with: { fetch-depth: 0 }
#       - uses: actions/download-artifact@v4
#         with: { name: backend-coverage }
#       - run: bash .github/scripts/check-diff-coverage.sh coverage.out origin/${{ github.base_ref }} 80
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

PROFILE="${1:-}"
BASE_REF="${2:-}"
THRESHOLD="${3:-80}"

if [ -z "$PROFILE" ] || [ -z "$BASE_REF" ]; then
  echo "usage: $0 <coverage-profile> <base-ref> [threshold]" >&2
  echo "" >&2
  echo "  <coverage-profile>  path to coverage.out (Go atomic/count profile)" >&2
  echo "  <base-ref>          diff base (e.g. origin/main) — three-dot merge-base diff" >&2
  echo "  [threshold]         diff coverage floor in percent (default: 80)" >&2
  exit 2
fi

cd "$ROOT"

if [ ! -f "$PROFILE" ]; then
  echo "check-diff-coverage.sh: coverage profile $PROFILE missing" >&2
  exit 2
fi

if ! git rev-parse --verify --quiet "$BASE_REF" >/dev/null; then
  echo "check-diff-coverage.sh: base ref $BASE_REF not found (fetch-depth 0 required in CI)" >&2
  exit 2
fi

MODULE="$(awk '/^module /{print $2; exit}' go.mod)"
if [ -z "$MODULE" ]; then
  echo "check-diff-coverage.sh: cannot read module path from go.mod" >&2
  exit 2
fi

# --- 1. Changed executable lines ---------------------------------------------
# Parse `git diff --unified=0` hunks: with zero context every line in a hunk is
# either + (added) or - (removed). Added lines map to new-file line numbers
# starting at the +c hunk offset; removed lines do not advance it.
# Emits "path<TAB>lineno" for each added executable line.

CHANGED="$(mktemp)"
trap 'rm -f "$CHANGED"' EXIT

if ! git diff --unified=0 "${BASE_REF}...HEAD" -- '*.go' ':(exclude)*_test.go' | awk '
  /^\+\+\+ b\// { file = substr($0, 7); in_hunk = 0; next }
  /^@@ / {
    match($0, /\+[0-9]+/)
    lineno = substr($0, RSTART + 1, RLENGTH - 1) + 0
    in_hunk = 1
    next
  }
  in_hunk && /^\+/ {
    line = substr($0, 2)
    # blank or comment-only lines are not executable — exclude from the gate
    if (line !~ /^[[:space:]]*$/ && line !~ /^[[:space:]]*\/\//) {
      printf "%s\t%d\n", file, lineno
    }
    lineno++
    next
  }
  in_hunk && /^-/ { next }    # removed line: not counted, lineno stays
  in_hunk && /^\\/ { next }   # "\ No newline at end of file"
' > "$CHANGED"; then
  echo "check-diff-coverage.sh: git diff against $BASE_REF failed" >&2
  exit 2
fi

if [ ! -s "$CHANGED" ]; then
  echo "diff-coverage: no testable .go lines changed vs $BASE_REF — PASS (nothing to gate)"
  exit 0
fi

# --- 2. Join changed lines with coverage blocks ------------------------------
# coverage.out block format: <module>/<pkg>/<file>.go:startLine.startCol,endLine.endCol <numStmt> <count>
#
# Measured/covered semantics (mirrors diff-cover conventions):
#   - file HAS blocks in the profile: only changed lines inside SOME block
#     (covered or not) enter the denominator — package/import/declaration
#     lines are not coverable and must not penalize the gate. A line is
#     covered when it falls inside a block with count > 0.
#   - file ABSENT from the profile (package never exercised by tests):
#     all its changed executable lines count as measured + uncovered —
#     brand-new untested files must NOT get a free pass.

RESULT="$(awk -v mod="$MODULE/" -v threshold="$THRESHOLD" '
  NR==FNR { changed[++nc] = $1 SUBSEP $2; files[$1] = 1; next }
  NF == 3 && $1 ~ /:/ {
    split($1, loc, ":")
    file = loc[1]
    if (index(file, mod) == 1) file = substr(file, length(mod) + 1)
    if (!(file in files)) next
    split(loc[2], rng, ",")
    na[file]++
    astart[file, na[file]] = rng[1] + 0
    aend[file, na[file]]   = rng[2] + 0
    cnt = $3 + 0
    if (cnt > 0) {
      nr[file]++
      rstart[file, nr[file]] = rng[1] + 0
      rend[file, nr[file]]   = rng[2] + 0
    }
    next
  }
  END {
    total = 0
    covered = 0
    for (i = 1; i <= nc; i++) {
      split(changed[i], kv, SUBSEP)
      f = kv[1]; ln = kv[2] + 0
      measured = 1
      if (na[f] > 0) {
        # file measured: line must intersect a block to enter denominator
        measured = 0
        for (j = 1; j <= na[f]; j++) {
          if (ln >= astart[f, j] && ln <= aend[f, j]) { measured = 1; break }
        }
      }
      # file absent from profile: stays measured + uncovered
      if (!measured) continue
      total++
      per_file_total[f]++
      hit = 0
      for (j = 1; j <= nr[f]; j++) {
        if (ln >= rstart[f, j] && ln <= rend[f, j]) { hit = 1; break }
      }
      if (hit) { covered++; per_file_cov[f]++ }
      else { printf "UNCOVERED %s:%d\n", f, ln }
    }
    for (f in per_file_total) {
      t = per_file_total[f]
      c = per_file_cov[f] + 0
      printf "FILE %-60s %6d/%6d lines %6.2f%%\n", f, c, t, (t > 0) ? c * 100.0 / t : 0
    }
    pct = (total > 0) ? covered * 100.0 / total : 100
    printf "DIFF %d %d %.2f\n", covered, total, pct
    if (pct + 0 < threshold + 0) {
      printf "FAIL: diff coverage %.2f%% < threshold %.2f%%\n", pct, threshold + 0
      exit 1
    }
    printf "PASS: diff coverage %.2f%% >= threshold %.2f%%\n", pct, threshold + 0
    exit 0
  }
' "$CHANGED" "$PROFILE")" && GATE_EXIT=0 || GATE_EXIT=$?

echo "$RESULT"

# --- 3. Gate result ----------------------------------------------------------

if [ "$GATE_EXIT" -ne 0 ]; then
  echo "" >&2
  echo "diff coverage gate FAILED — changed lines below $THRESHOLD% coverage (GOV-03)" >&2
  echo "Add tests covering the UNCOVERED lines above, or shrink the diff." >&2
  exit 1
fi

if ! echo "$RESULT" | grep -qE '^PASS:'; then
  echo "check-diff-coverage.sh: no PASS line emitted — treating as failure" >&2
  exit 1
fi

echo "diff coverage gate passed (threshold=$THRESHOLD%)"
exit 0
