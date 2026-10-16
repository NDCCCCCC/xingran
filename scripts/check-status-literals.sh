#!/usr/bin/env bash
# check-status-literals.sh — ratchet guard against bare numeric status literals
# in backend service/handler source (Phase 69 DICT-01).
#
# Purpose: internal/models is the single source of truth for status semantics.
# As DICT-01 batches (69-01/69-03/69-04/69-05) replace bare `status = 0/1`
# literals with models constants, this script prevents regressions: any NEW
# hit outside the ALLOWED list — or a hit count ABOVE the registered baseline
# — fails the check. The ALLOWED list is a ratchet: each completed replacement
# batch deletes its entries (counts may only go DOWN); at the end only
# cluster-F permanent exemptions remain.
#
# Usage:
#   bash scripts/check-status-literals.sh              # check mode (exit 0/1)
#   bash scripts/check-status-literals.sh --baseline   # print path=count lines
#                                                      # for pasting into ALLOWED
#
# Hit patterns (four, per Phase 69 RESEARCH A1 — keep in sync):
#   (a) raw SQL string:   status = [0-9]        e.g. Where("status = 0")
#   (b) struct literal:   Status: *[0-9]        e.g. Status: 0,  (not line-anchored,
#                                                     so PublishStatus: 1 also hits)
#   (c) comparison:       Status [=!]= *[0-9]   e.g. Status != 0
#   (d) map/JSON literal: "status": <ws> digit  e.g. "status":  0,  (escapes a/b/c)
#
# Exclusions: *_test.go; lines whose first non-blank characters are `//`.
# Scope: internal/api/v1 + internal/services only.
#
# CI hookup (Phase 63 candidate, NOT wired by this change): add a CI step
#   `bash scripts/check-status-literals.sh` and fail the build on exit 1.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Combined ERE for the four hit patterns.
PATTERN='status = [0-9]|Status: *[0-9]|Status [=!]= *[0-9]|"status":[[:space:]]*[0-9]'

# ALLOWED — ratchet baseline captured 2026-08-19 (43 files, 149 hits);
# batch 1 (services/system: dict/post/role/user/widget_data_fetcher) replaced
# and removed 2026-08-19 — 38 files remained.
# batch 2 (services/operations 10 files + api/v1/operations/excel_handler,
# 58 hits) replaced and removed 2026-08-19 — 27 files remained (69-03).
# batch 4 (workorder/duty/discovery/execution/dispatch/monitor/asset, api/v1,
# and scattered service/handler hits) replaced and removed 2026-08-19 — 17 files
# remain (69-05). Delete entries as replacement batches land; counts may only decrease.
# geocoding_service.go is cluster-F: Baidu API response-code contract, never migrated.
declare -A ALLOWED=(
  ["internal/services/operations/geocoding_service.go"]=1 # F 簇：百度 API 返回码契约，不迁移
)

# hits_in FILE — count non-comment lines matching PATTERN in FILE.
hits_in() {
  local n
  n=$(grep -vE '^[[:space:]]*//' "$1" | grep -cE "$PATTERN" || true)
  echo "${n:-0}"
}

# collect — print "count relpath" for every in-scope file with >=1 hit, sorted.
collect() {
  local f n rel
  while IFS= read -r f; do
    case "$f" in
      *_test.go) continue ;;
    esac
    n=$(hits_in "$f")
    if [ "$n" -gt 0 ]; then
      rel="${f#"$ROOT"/}"
      echo "$n $rel"
    fi
  done < <(grep -rlE --include='*.go' "$PATTERN" "$ROOT/internal/api/v1" "$ROOT/internal/services" || true)
}

# show_hits RELPATH — print up to 5 offending lines for failure messages.
show_hits() {
  grep -nvE '^[[:space:]]*//' "$ROOT/$1" | grep -E "$PATTERN" | head -5 | sed 's/^/    /' || true
}

baseline() {
  collect | while read -r n rel; do echo "$rel=$n"; done
}

check() {
  local failed=0 line n rel allowed
  while read -r line; do
    [ -z "$line" ] && continue
    n="${line%% *}"
    rel="${line#* }"
    allowed="${ALLOWED[$rel]:-}"
    if [ -z "$allowed" ]; then
      echo "FAIL: $rel has $n status-literal hit(s) but is not in the ALLOWED ratchet list"
      show_hits "$rel"
      failed=1
    elif [ "$n" -gt "$allowed" ]; then
      echo "FAIL: $rel hit count $n exceeds ratchet baseline $allowed"
      show_hits "$rel"
      failed=1
    elif [ "$n" -lt "$allowed" ]; then
      echo "ratchet down: $rel now has $n hit(s), baseline says $allowed — tighten or delete the ALLOWED entry"
    fi
  done < <(collect)

  # Flag stale ALLOWED entries whose files no longer exist or hit zero —
  # informational only (keeps the ratchet honest without blocking).
  for rel in "${!ALLOWED[@]}"; do
    if [ ! -f "$ROOT/$rel" ]; then
      echo "note: ALLOWED entry $rel points to a missing file — remove it"
    fi
  done

  if [ "$failed" -ne 0 ]; then
    echo ""
    echo "status-literal ratchet check FAILED — replace bare literals with internal/models"
    echo "constants (see Phase 69 DICT-01), or register/tighten the ALLOWED baseline."
    exit 1
  fi
  echo "status-literal ratchet check passed"
}

case "${1:-}" in
  --baseline) baseline ;;
  ""|check) check ;;
  *)
    echo "usage: $0 [--baseline]" >&2
    exit 2
    ;;
esac
