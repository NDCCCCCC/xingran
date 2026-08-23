#!/usr/bin/env bash
# check-frontend-diff-coverage.sh — Phase 82 GOV-04 PR diff coverage gate
# (frontend twin of check-diff-coverage.sh, D-15 lines caliber).
#
# Supply-chain decision chain — quoted verbatim from the backend 74-10
# precedent (.github/scripts/check-diff-coverage.sh L4-14); the frontend gate
# adopts the same conclusion, zero third-party actions:
#
#   D-14 locked an external action (gocover-coverage / ORY xcoverage-action) as
#   the preferred tool, with an explicit ratchet clause: if the locked action
#   turns out unsuitable or unmaintained, the executor MAY fall back to a
#   self-implemented solution (git diff + awk + coverage.out parsing) as long as
#   the GOV-03 >=80% threshold is enforced 100%.
#
#   Executor finding (2026-08-22): "gocover-coverage" has no verifiable
#   marketplace presence and "ory/xcoverage-action" does not exist at all.
#   Rather than gambling on an unvetted third-party action for a hard merge gate,
#   this script implements the gate in-repo with the same zero-dep bash+awk
#   paradigm as check-coverage.sh (D-01) — auditable, no supply-chain surface.
#
# Semantics (D-15, lines caliber — symmetric to the backend block granularity):
#   diff coverage = changed executable lines covered by tests / all changed
#                   executable lines in the PR (xingran-react-frontend/src)
#   - changed lines: `+` lines from `git diff --unified=0 <base>...HEAD`
#     (three-dot = merge-base), restricted to src *.ts/*.tsx. The exclude
#     pathspec MIRRORS the vitest coverage.exclude array in
#     xingran-react-frontend/vitest.config.ts (single truth source, D-10):
#     *.test.*, __tests__/**, **/*.d.ts, src/test/** and the cad whitelist
#     dirs cad-editor/** / cad-elements/**. The two lists MUST be maintained
#     in sync, in the same commit — a file excluded from the coverage json
#     but admitted by the diff pathspec falls into the "absent = all changed
#     lines uncovered" branch below and fails every PR touching it (CR-01).
#   - executable: blank lines and TS comment-only lines excluded — // line
#     comments, /* block-comment openers, and * JSDoc continuation lines
#   - covered: a changed line falls inside an istanbul statementMap range with
#     hit=1 (start.line <= L <= end.line). Statement ranges are the v8
#     provider's source for the lines dimension, so joining on them is the
#     faithful lines-caliber implementation (columns are ignored; end.column
#     may be null) — symmetric to the backend "inside a block with count > 0".
#   - file ABSENT from the coverage json (defensive branch — rare under the
#     all-src caliber, e.g. a file deleted after the json was generated):
#     all its changed executable lines count as measured + uncovered —
#     brand-new untested files must NOT get a free pass (T-82-03-03).
#     A changed line that falls into NO statement range (pure type /
#     declaration / import-only lines are not coverable) stays out of the
#     denominator and does not penalize the gate.
#
# Usage:
#   bash .github/scripts/check-frontend-diff-coverage.sh <coverage-final.json> <base-ref> <threshold>
#
# Exit codes (mirror check-diff-coverage.sh):
#   0 — diff coverage >= threshold, OR no testable .ts/.tsx lines changed (skip)
#   1 — diff coverage < threshold (gate fails) OR parse failure (no PASS line)
#   2 — usage error / missing inputs / base ref not rev-parseable / diff failed
#
# CI hookup (ci.yml PR-only job, wired in Phase 82-04):
#   frontend-coverage-diff:
#     needs: frontend
#     if: github.event_name == 'pull_request'
#     steps:
#       - uses: actions/checkout@v7
#         with: { fetch-depth: 0 }
#       - uses: actions/download-artifact@v4
#         with: { name: frontend-coverage }
#       - run: bash .github/scripts/check-frontend-diff-coverage.sh \
#                xingran-react-frontend/coverage/coverage-final.json origin/${{ github.base_ref }} 80
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

PROFILE="${1:-}"
BASE_REF="${2:-}"
THRESHOLD="${3:-}"

if [ -z "$PROFILE" ] || [ -z "$BASE_REF" ] || [ -z "$THRESHOLD" ]; then
  echo "usage: $0 <coverage-final.json> <base-ref> <threshold>" >&2
  echo "" >&2
  echo "  <coverage-final.json>  path to vitest istanbul json (xingran-react-frontend/coverage/...)" >&2
  echo "  <base-ref>             diff base (e.g. origin/main) — three-dot merge-base diff" >&2
  echo "  <threshold>            diff coverage floor in percent (e.g. 80)" >&2
  exit 2
fi

case "$THRESHOLD" in
  ''|*[!0-9.]*)
    echo "check-frontend-diff-coverage.sh: threshold '$THRESHOLD' is not a number" >&2
    exit 2
    ;;
esac

cd "$ROOT"

# The base ref comes from CI event context — verify it resolves to an existing
# object BEFORE it is used anywhere else (T-82-03-02 injection surface).
if ! git rev-parse --verify --quiet "$BASE_REF" >/dev/null; then
  echo "check-frontend-diff-coverage.sh: base ref $BASE_REF not found (fetch-depth 0 required in CI)" >&2
  exit 2
fi

# Fail-safe: when no profile exists (frontend job's Test step skipped or
# coverage not generated), don't fail this job — the upstream failure is the
# real blocker. Missing profile is a soft skip, not a hard gate failure (same
# semantics as check-frontend-coverage.sh, pairing with the PR job's needs:).
if [ ! -f "$PROFILE" ]; then
  echo "check-frontend-diff-coverage.sh: coverage profile $PROFILE missing — was the frontend job's Test step skipped?" >&2
  echo "check-frontend-diff-coverage.sh: skipping gate (exit 0); the upstream job failure is the real blocker" >&2
  exit 0
fi

if [ ! -r "$PROFILE" ]; then
  echo "check-frontend-diff-coverage.sh: coverage profile $PROFILE not readable" >&2
  exit 2
fi

CHANGED="$(mktemp)"
FLAT="$(mktemp)"
trap 'rm -f "$CHANGED" "$FLAT"' EXIT

# Three-dot (merge-base) is the PR semantic. When the base shares NO history
# with HEAD, git aborts with "no merge base" — fall back to a direct two-tree
# diff against the base commit itself. For an unrelated base that is exactly
# "everything is new", which only widens the diff (fails safe, never narrows
# it). Real PRs always share history, so CI stays on the merge-base path; the
# fallback exists for local deterministic baselines (Phase 82-03 Task 2's
# synthetic empty-tree commit).
MERGE_BASE="$(git merge-base "$BASE_REF" HEAD 2>/dev/null || true)"
if [ -n "$MERGE_BASE" ]; then
  DIFF_ARGS=("${BASE_REF}...HEAD")
else
  DIFF_ARGS=("$BASE_REF" "HEAD")
fi

# --- 1. Changed executable lines ---------------------------------------------
# Parse `git diff --unified=0` hunks: with zero context every line in a hunk is
# either + (added) or - (removed). Added lines map to new-file line numbers
# starting at the +c hunk offset; removed lines do not advance it.
# Emits "path<TAB>lineno" for each added executable line. TS comment exclusion
# is three-state: // line comments, /* block openers and * JSDoc continuations.

if ! git diff --unified=0 "${DIFF_ARGS[@]}" -- \
  'xingran-react-frontend/src/*.ts' 'xingran-react-frontend/src/*.tsx' \
  ':(exclude)xingran-react-frontend/src/*.test.*' \
  ':(exclude)xingran-react-frontend/src/**/__tests__/**' \
  ':(exclude)xingran-react-frontend/src/**/*.d.ts' \
  ':(exclude)xingran-react-frontend/src/test/**' \
  ':(exclude)xingran-react-frontend/src/components/cad-editor/**' \
  ':(exclude)xingran-react-frontend/src/components/cad-elements/**' | awk '
  /^\+\+\+ b\// { file = substr($0, 7); in_hunk = 0; next }
  /^@@ / {
    match($0, /\+[0-9]+/)
    lineno = substr($0, RSTART + 1, RLENGTH - 1) + 0
    in_hunk = 1
    next
  }
  in_hunk && /^\+/ {
    line = substr($0, 2)
    # blank or TS-comment-only lines are not executable — exclude from the gate
    if (line !~ /^[[:space:]]*$/ && line !~ /^[[:space:]]*\/\// \
        && line !~ /^[[:space:]]*\/\*/ && line !~ /^[[:space:]]*\*/) {
      printf "%s\t%d\n", file, lineno
    }
    lineno++
    next
  }
  in_hunk && /^-/ { next }    # removed line: not counted, lineno stays
  in_hunk && /^\\/ { next }   # "\ No newline at end of file"
' > "$CHANGED"; then
  echo "check-frontend-diff-coverage.sh: git diff against $BASE_REF failed" >&2
  exit 2
fi

if [ ! -s "$CHANGED" ]; then
  echo "diff-coverage: no testable .ts/.tsx lines changed vs $BASE_REF — PASS (nothing to gate)"
  exit 0
fi

# --- 2a. Flatten the istanbul json to TSV ------------------------------------
# Same P-node as check-frontend-coverage.sh (each script embeds its own copy;
# no cross-script imports). One line per file:
#   relpath<TAB>stmts<TAB>covered<TAB>start-end:hit,...
# The json is a 5.6MB single-line document — plain awk cannot parse it; node is
# the frontend toolchain's standard runtime (zero third-party deps).
# Path normalization (Windows/CI dual compatible): backslashes -> forward
# slashes, anchor "xingran-react-frontend/src/" matched case-insensitively,
# relpath = anchor onward.

if ! node -e '
const fs = require("fs");
const d = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
const out = [];
for (const p of Object.keys(d)) {
  const f = d[p] || {};
  const s = f.s || {};
  const sm = f.statementMap || {};
  const n = String(p).replace(/\\/g, "/");
  const i = n.toLowerCase().indexOf("xingran-react-frontend/src/");
  if (i < 0) continue;
  const rel = n.slice(i);
  let tot = 0, cov = 0;
  const ranges = [];
  for (const k of Object.keys(s)) {
    const m = sm[k];
    if (!m) continue;
    tot++;
    const hit = s[k] > 0;
    if (hit) cov++;
    ranges.push(m.start.line + "-" + m.end.line + ":" + (hit ? 1 : 0));
  }
  out.push(rel + "\t" + tot + "\t" + cov + "\t" + ranges.join(","));
}
process.stdout.write(out.join("\n") + (out.length > 0 ? "\n" : ""));
' "$PROFILE" > "$FLAT"; then
  echo "check-frontend-diff-coverage.sh: node flattening of $PROFILE failed — malformed json?" >&2
  exit 1
fi

# --- 2b. Join changed lines with statementMap ranges -------------------------
# Measured/covered semantics (frontend twin of the backend block join):
#   - file present in the json: only changed lines inside SOME statement range
#     (hit or miss) enter the denominator — pure type/declaration/import lines
#     are not coverable and must not penalize the gate. A line is covered when
#     it falls inside a range with hit = 1.
#   - file ABSENT from the json: all its changed executable lines count as
#     measured + uncovered — brand-new untested files must NOT get a free pass.

RESULT="$(awk -F'\t' -v threshold="$THRESHOLD" '
  NR == FNR { changed[++nc] = $1 SUBSEP $2; files[$1] = 1; next }
  NF >= 3 && ($1 in files) {
    file = $1
    in_tsv[file] = 1
    if ($4 != "") {
      n = split($4, rl, ",")
      for (i = 1; i <= n; i++) {
        split(rl[i], hp, ":")
        split(hp[1], se, "-")
        na[file]++
        astart[file, na[file]] = se[1] + 0
        aend[file, na[file]]   = se[2] + 0
        if ((hp[2] + 0) > 0) {
          nrng[file]++
          rstart[file, nrng[file]] = se[1] + 0
          rend[file, nrng[file]]   = se[2] + 0
        }
      }
    }
    next
  }
  END {
    total = 0
    covered = 0
    for (i = 1; i <= nc; i++) {
      split(changed[i], kv, SUBSEP)
      f = kv[1]; ln = kv[2] + 0
      measured = 0
      if (f in in_tsv) {
        # file measured: line must intersect a statement range to count
        for (j = 1; j <= na[f]; j++) {
          if (ln >= astart[f, j] && ln <= aend[f, j]) { measured = 1; break }
        }
      } else {
        # file absent from the json: stays measured + uncovered (no free pass)
        measured = 1
      }
      if (!measured) continue
      total++
      per_file_total[f]++
      hit = 0
      for (j = 1; j <= nrng[f]; j++) {
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
' "$CHANGED" "$FLAT")" && GATE_EXIT=0 || GATE_EXIT=$?

echo "$RESULT"

# --- 3. Gate result ----------------------------------------------------------

if [ "$GATE_EXIT" -ne 0 ]; then
  echo "" >&2
  echo "frontend diff coverage gate FAILED — changed lines below $THRESHOLD% coverage (GOV-04)" >&2
  echo "Add tests covering the UNCOVERED lines above, or shrink the diff." >&2
  exit 1
fi

# Defensive tail (mirrors backend L199-201): if awk succeeded but no PASS line
# was emitted, treat it as a parse failure — the gate must never pass silently.
if ! echo "$RESULT" | grep -qE '^PASS:'; then
  echo "check-frontend-diff-coverage.sh: no PASS line emitted — treating as failure" >&2
  exit 1
fi

echo "frontend diff coverage gate passed (threshold=$THRESHOLD%)"
exit 0
