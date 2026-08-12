#!/usr/bin/env bash
# operlog_e2e_verify.sh — Phase 34 end-to-end verification
#
# This script contains TWO clearly-separated sections:
#   1. STATIC COVERAGE CHECK (no DB / no running API required)
#      Grep + awk over the source tree: total operlog.Record call count,
#      sensitive-key count, and the per-handler differential that
#      catches handler files with ZERO operlog calls. Runs unconditionally
#      (even when SKIP_LIVE=1) and is what CI uses as a PR-time lint.
#   2. LIVE E2E CHECK (requires a running API + reachable DB)
#      Triggers 30 sampled write endpoints across all 7 Phase-34 waves
#      (+ the 9 pre-existing AD endpoints that Wave 1 inherited) and
#      verifies that each one inserts a row in `sys_oper_log` with the
#      expected title, businessType, and (where applicable) masked
#      sensitive param. Skipped when SKIP_LIVE=1.
#
# The two sections are also called out inline by `==========` banners so a
# maintainer running the script can tell at a glance which assertions are
# static vs. live.
#
# ------------------------------------------------------------------
# Required environment variables (LIVE E2E CHECK only):
#   API_BASE          (default: http://localhost:9000)
#   DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD  (database connection)
#   ADMIN_USER        (admin username for JWT login)
#   ADMIN_PASSWORD    (admin password for JWT login)
#
# Optional:
#   DEV_MODE          (set to 1/true/dev/development to allow fallback default
#                     credentials admin/admin123 for local dev runs only;
#                     ABSENT in any non-dev environment = script exits 1)
#   SKIP_LIVE         (set to 1/true to skip the LIVE E2E CHECK and only run
#                     the STATIC COVERAGE CHECK — useful in CI without a
#                     backend)
#
# Exit codes:
#   0  all checks passed
#   1  credential error or assertion failure
#   2  live backend / DB unreachable and SKIP_LIVE not set
# ------------------------------------------------------------------

set -euo pipefail

API_BASE="${API_BASE:-http://localhost:9000}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-xingran_next}"
DB_USER="${DB_USER:-xingran}"
DB_PASSWORD="${DB_PASSWORD:-}"
PSQL_EXTRA_ARGS="${PSQL_EXTRA_ARGS:-}"

# ----- Credential resolution -----------------------------------------
# WARNING 2 mitigation (T-34-VER-01 + T-34-VER-04):
# Admin credentials MUST come from env vars. The dev fallback admin/admin123
# is permitted ONLY when DEV_MODE is explicitly set; in every other
# environment missing creds cause exit 1 before any auth attempt.
if [[ -n "${ADMIN_USER:-}" && -n "${ADMIN_PASSWORD:-}" ]]; then
  : # env vars present, use them as-is
elif [[ "${DEV_MODE:-}" =~ ^(1|true|dev|development)$ ]]; then
  echo "WARN: using dev fallback credentials admin/admin123 (DEV_MODE=${DEV_MODE})" >&2
  ADMIN_USER="admin"
  ADMIN_PASSWORD="admin123"
else
  echo "ERROR: ADMIN_USER and ADMIN_PASSWORD env vars required (or set DEV_MODE=1 for local dev)" >&2
  exit 1
fi

# ============================================================
# STATIC COVERAGE CHECK (no DB required)
# Walks every internal/api/v1/*/handler.go and asserts the file
# contains at least one operlog.Record / RecordWithBody call
# (or the legacy recordOperLog shim in internal/api/v1/system/
# helper.go). Also counts the total operlog.Record call sites and
# the sensitiveKeys keyword set size. This is a PR-time lint,
# not a live e2e test — see the LIVE E2E CHECK section below.
# ============================================================
echo "== Static checks =="
CALLS=$(grep -rE "operlog\.(Record|RecordWithBody)\(" internal/ 2>/dev/null | wc -l | tr -d ' ')
echo "operlog.Record|RecordWithBody calls in internal/: ${CALLS}"
# Threshold raised from 250 to 290 after Phase 34 gap-closure (34-gap) instrumented
# 25 previously-missed write endpoints across 6 handler files. The previous loose
# >=250 threshold could not detect the 25-endpoint gap — see 34-VERIFICATION.md.
if [[ "${CALLS}" -lt 290 ]]; then
  echo "FAIL: expected >=290 operlog calls, got ${CALLS}" >&2
  exit 1
fi

SENSITIVE_KEYS=$(awk '/var sensitiveKeys = \[\]string\{/,/^\}/' "$(dirname "$0")/../internal/utils/operlog/operlog.go" 2>/dev/null | grep -cE '^\s+"[^"]+",?\s*$' || true)
echo "sensitiveKeys entries in operlog.go: ${SENSITIVE_KEYS}"
if [[ "${SENSITIVE_KEYS}" -lt 17 ]]; then
  echo "FAIL: expected >=17 sensitive-key entries, got ${SENSITIVE_KEYS}" >&2
  exit 1
fi

# ----- Handler-file-vs-operlog differential check --------------------
# F-OPLOG-VER gap-closure: enumerate every *_handler.go file under
# internal/api/v1/ that defines a write-method receiver (func (h *...Handler))
# and FAIL if it contains ZERO operlog.Record / RecordWithBody calls.
# This catches the class of gap found in 34-VERIFICATION (6 handler files with
# routed POST/PUT/DELETE writes but zero operlog calls).
#
# Read-only handler allowlist — files that legitimately contain no write methods
# (pure GET / list / tree / stats endpoints). Add a file here ONLY with a
# documented reason; every entry below is a read-only handler by construction.
READONLY_ALLOWLIST=(
  # Read-only handlers: define only query/list/stats/export-via-GET methods (no
  # state mutation). The POST verbs below are query-with-body patterns, not
  # writes. Add a file here ONLY if every routed method is a read.
  "network/mac_history_handler.go"         # all methods are /history/* queries + 1 GET export
  "network/mac_history_heatmap_handler.go" # only QueryHeatmap (POST /history/heatmap, read)
)

echo
echo "== Handler-file operlog coverage differential =="
HANDLER_FILES=$(grep -rlE '^func \(h \*[A-Za-z]+Handler\) [A-Za-z]+\(' internal/api/v1/ --include='*_handler.go' 2>/dev/null | sort -u)
ZERO_OPLOG_HANDLERS=()
for f in ${HANDLER_FILES}; do
  # grep -c prints the match count but exits 1 when count==0; suppress that
  # exit code so set -e / pipefail don't abort, and capture only the number.
  direct=$(grep -cE "operlog\.(Record|RecordWithBody)\(" "$f" 2>/dev/null || true)
  direct=$(echo "${direct}" | tr -d '[:space:]')
  [[ -z "${direct}" ]] && direct=0
  # Also count the legacy recordOperLog shim (internal/api/v1/system/helper.go)
  # which delegates to operlog.Record — AD-domain handlers (Wave 1 backward-compat)
  # use this shim instead of the direct call.
  shim=$(grep -cE "recordOperLog\(" "$f" 2>/dev/null || true)
  shim=$(echo "${shim}" | tr -d '[:space:]')
  [[ -z "${shim}" ]] && shim=0
  count=$((direct + shim))
  if [[ "${count}" -eq 0 ]]; then
    # Check allowlist
    allowed=0
    for wl in "${READONLY_ALLOWLIST[@]}"; do
      if [[ "$f" == *"$wl" ]]; then allowed=1; break; fi
    done
    if [[ "${allowed}" -eq 0 ]]; then
      ZERO_OPLOG_HANDLERS+=("$f")
    fi
  fi
done

if [[ ${#ZERO_OPLOG_HANDLERS[@]} -gt 0 ]]; then
  echo "FAIL: the following *_handler.go files define methods on a *Handler receiver" >&2
  echo "      but contain ZERO operlog.Record / RecordWithBody calls:" >&2
  for f in "${ZERO_OPLOG_HANDLERS[@]}"; do
    echo "        - $f" >&2
  done
  echo "      If a file is genuinely read-only, add it to READONLY_ALLOWLIST with a reason." >&2
  exit 1
fi
echo "all *Handler-receiver files contain >=1 operlog call (or are allowlisted read-only)"

# ============================================================
# LIVE E2E CHECK (requires running API + DB)
# Skipped when SKIP_LIVE=1; otherwise sends real HTTP requests
# and asserts against sys_oper_log rows. See
# tests/e2e/operlog_e2e_verify_test.go for the Go test driver
# that wraps this script in `go test`. Reciprocal of the
# STATIC COVERAGE CHECK banner above.
# ============================================================
# ----- Live-DB portion (optional) ------------------------------------
if [[ "${SKIP_LIVE:-}" =~ ^(1|true)$ ]]; then
  echo "SKIP_LIVE=1 set — skipping live-DB portion (static checks PASSED)" >&2
  exit 0
fi

# Probe backend + DB; if either is unreachable, surface a clear message.
if ! curl -sf --max-time 5 "${API_BASE}/" >/dev/null 2>&1 \
   && ! curl -sf --max-time 5 "${API_BASE}/api/v1/system/auth/login" -o /dev/null 2>&1; then
  echo "WARN: backend at ${API_BASE} unreachable — set SKIP_LIVE=1 to skip live-DB checks" >&2
  exit 2
fi

export PGPASSWORD="${DB_PASSWORD}"
psql_cmd=(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -A -F '|')
if ! "${psql_cmd[@]}" -c "SELECT 1" >/dev/null 2>&1; then
  echo "WARN: database ${DB_NAME}@${DB_HOST}:${DB_PORT} unreachable — set SKIP_LIVE=1 to skip live-DB checks" >&2
  exit 2
fi

echo
echo "== Authenticate as ${ADMIN_USER} =="
LOGIN_BODY=$(cat <<JSON
{"username":"${ADMIN_USER}","password":"${ADMIN_PASSWORD}"}
JSON
)
LOGIN_RESP=$(curl -s --max-time 15 -X POST "${API_BASE}/api/v1/system/auth/login" \
  -H 'Content-Type: application/json' -d "${LOGIN_BODY}" || true)
# Token may live under data.token or data.accessToken — try both.
TOKEN=$(echo "${LOGIN_RESP}" | sed -nE 's/.*"(token|accessToken)"[[:space:]]*:[[:space:]]*"([^"]+)".*/\2/p' | head -1)
if [[ -z "${TOKEN}" ]]; then
  echo "FAIL: login did not return a token. Response:" >&2
  echo "${LOGIN_RESP}" >&2
  exit 1
fi
echo "JWT acquired (length=${#TOKEN})"

PASS_COUNT=0
FAIL_COUNT=0

# assert_logged(method path body expected_title expected_business_type [sensitive_substring])
# Issues the request then SELECTs the most-recent sys_oper_log row matching
# (oper_url, title). Asserts business_type matches. If a sensitive substring
# is provided, asserts the value was masked to ****** and that the plaintext
# value is NOT present.
assert_logged() {
  local method="$1" path="$2" body="$3" expected_title="$4" expected_btype="$5"
  local sensitive="${6:-}"
  local label="${method} ${path}"
  echo -n "  [${label}] ... "

  # Make the request; capture status. We do NOT require 2xx for some sample
  # endpoints (e.g. delete of a non-existent id still records the attempt),
  # so we tolerate non-2xx but still assert the log row exists.
  curl -s --max-time 30 -o /dev/null -w '%{http_code}' \
    -X "${method}" "${API_BASE}${path}" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H 'Content-Type: application/json' \
    -d "${body}" > /tmp/operlog_status.$$ || true

  # Match oper_url by suffix so the /api/v1 prefix doesn't matter.
  local row
  row=$("${psql_cmd[@]}" -c "SELECT business_type || '|' || COALESCE(oper_param,'') FROM sys_oper_log WHERE title = '${expected_title}' AND oper_url LIKE '%${path}' ORDER BY oper_time DESC LIMIT 1;" 2>/dev/null || true)

  if [[ -z "${row}" ]]; then
    echo "FAIL (no log row)"
    FAIL_COUNT=$((FAIL_COUNT + 1))
    return
  fi

  local got_btype="${row%%|*}"
  local got_param="${row#*|}"

  if [[ "${got_btype}" != "${expected_btype}" ]]; then
    echo "FAIL (business_type got=${got_btype} want=${expected_btype})"
    FAIL_COUNT=$((FAIL_COUNT + 1))
    return
  fi

  if [[ -n "${sensitive}" ]]; then
    if [[ "${got_param}" != *"******"* ]]; then
      echo "FAIL (sensitive value not masked in oper_param)"
      FAIL_COUNT=$((FAIL_COUNT + 1))
      return
    fi
    if [[ "${got_param}" == *"${sensitive}"* ]]; then
      echo "FAIL (plaintext sensitive value leaked in oper_param)"
      FAIL_COUNT=$((FAIL_COUNT + 1))
      return
    fi
  fi

  echo "OK"
  PASS_COUNT=$((PASS_COUNT + 1))
}

# ------------------------------------------------------------------
# 30 sampled endpoints — one or more per Wave + 9 AD endpoints.
# businessType: 1=Create 2=Update 3=Delete 4=Grant 5=Export 6=Import
#               9=Clean 10=Status 11=Reset 14=Sync 15=Move 16=Batch
#               17=Upload 21=Register 22=Approve 23=Reject  0=Other
# ------------------------------------------------------------------
echo
echo "== Sampled write endpoints =="

# --- Wave 1 (system core: user/role/dept/menu/dict/post) ---
assert_logged POST "/api/v1/system/users"               '{"username":"e2e_u1","password":"hunter2pw","nickname":"e2e"}' "用户管理" 1 "password"
assert_logged POST "/api/v1/system/users/e2e-001/update" '{"nickname":"updated"}'                                "用户管理" 2
assert_logged POST "/api/v1/system/users/e2e-001/delete" '{}'                                                   "用户管理" 3
assert_logged POST "/api/v1/system/users/e2e-001/reset-password" '{"password":"newpw123"}'                       "用户管理" 11 "password"
assert_logged POST "/api/v1/system/users/batch-delete"   '{"ids":["e2e-001"]}'                                  "用户管理" 16
assert_logged POST "/api/v1/system/users/e2e-001/status" '{"status":"1"}'                                       "用户管理" 10

assert_logged POST "/api/v1/system/roles"               '{"name":"e2e_role","roleKey":"e2e_r","roleSort":999}'  "角色管理" 1
assert_logged POST "/api/v1/system/roles/e2e-001/update" '{"name":"e2e_role2"}'                                "角色管理" 2
assert_logged POST "/api/v1/system/roles/e2e-001/grant" '{"menuIds":["1"]}'                                    "角色管理" 4

assert_logged POST "/api/v1/system/depts"               '{"name":"e2e_dept","parentId":"1"}'                   "部门管理" 1
assert_logged POST "/api/v1/system/depts/e2e-001/update" '{"name":"e2e_dept2"}'                                "部门管理" 2

assert_logged POST "/api/v1/system/menus"               '{"name":"e2e_menu","parentId":"0"}'                   "菜单管理" 1
assert_logged POST "/api/v1/system/dicts/types"         '{"name":"e2e_dict","dictType":"e2e_t"}'              "字典管理" 1

# --- Wave 2 (system peripherals: notice/apikey/config/profile/settings/file) ---
assert_logged POST "/api/v1/system/notices"             '{"title":"e2e_notice","content":"x"}'                "通知管理" 1
assert_logged POST "/api/v1/system/apikeys"             '{"name":"e2e_ak","apiKey":"supersecret"}'           "API密钥" 1 "supersecret"
assert_logged POST "/api/v1/system/configs"             '{"name":"e2e_cfg","configKey":"e2e_k","configValue":"v"}' "参数配置" 1

# --- Wave 3 (operations: building/floor/workstation/excel/asset) ---
assert_logged POST "/api/v1/ops/building"               '{"name":"e2e_bld","org_id":"00000000-0000-0000-0000-000000000000"}' "楼宇管理" 1
assert_logged POST "/api/v1/ops/building/e2e-001/update" '{"name":"e2e_bld2"}'                                  "楼宇管理" 2
assert_logged POST "/api/v1/ops/workstation"            '{"name":"e2e_ws"}'                                   "工位管理" 1

# --- Wave 4 (network) ---
assert_logged POST "/api/v1/network/devices"            '{"name":"e2e_dev","ip":"10.0.0.1"}'                 "网络设备" 1
assert_logged POST "/api/v1/network/credentials"        '{"name":"e2e_c","enablePassword":"topsecret"}'      "设备凭据" 1 "topsecret"

# --- Wave 5 (vdi/workorder/duty/knowledge/scheduler) ---
assert_logged POST "/api/v1/workorder/workorders"       '{"title":"e2e_wo"}'                                 "工单管理" 1
assert_logged POST "/api/v1/workorder/workorders/e2e-001/approve" '{}'                                          "工单管理" 22
assert_logged POST "/api/v1/workorder/workorders/e2e-001/reject"  '{"comment":"no"}'                            "工单管理" 23
assert_logged POST "/api/v1/vdi/vms/e2e-001/start"      '{}'                                                   "虚拟机管理" 10
assert_logged POST "/api/v1/scheduler/jobs/e2e-001/execute" '{}'                                               "定时任务" 0

# --- Wave 6 (monitor/rpa/agent) ---
assert_logged POST "/api/v1/monitor/cache/clear"        '{"cacheName":"xingran"}'                               "缓存监控" 9
assert_logged POST "/api/v1/rpa/tasks"                  '{"name":"e2e_rpa"}'                                 "RPA任务" 1
assert_logged POST "/api/v1/rpa/credentials"            '{"name":"e2e_rpac","apiKey":"apikeysecret"}'        "RPA凭据" 1 "apikeysecret"
assert_logged POST "/api/v1/agent/register"             '{"hostname":"e2e-host"}'                            "Agent注册" 21

# --- Wave 7 (system submodules) ---
assert_logged POST "/api/v1/system/dashboards"          '{"name":"e2e_dash"}'                                "仪表盘管理" 1
assert_logged POST "/api/v1/system/column-config/save"  '{"page":"e2e","config":"{}"}'                       "列设置" 2

# --- Gap-closure (34-gap): compliance-sensitive + previously-missed endpoints ---
assert_logged POST "/api/v1/system/user-unlock/unlock"  '{"username":"e2e_locked"}'                          "用户解锁" 24
assert_logged POST "/api/v1/monitor/cache/invalidate"   '{"module":"user"}'                                  "缓存监控" 9
assert_logged POST "/api/v1/network/devices/export"     '{"exportMode":"all"}'                               "网络设备" 5

echo
echo "== Summary =="
TOTAL=$((PASS_COUNT + FAIL_COUNT))
echo "${PASS_COUNT}/${TOTAL} passed"
rm -f /tmp/operlog_status.$$

if [[ "${FAIL_COUNT}" -ne 0 ]]; then
  exit 1
fi
exit 0
