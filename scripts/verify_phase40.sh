#!/usr/bin/env bash
# scripts/verify_phase40.sh
# Phase 40 验收脚本（D-16）
# 两条独立标准：
#   1. scripts/validate_debug_frontmatter.sh 全量扫描通过率 = 100%
#   2. gsd-sdk query audit-open 输出 debug_sessions < 5
# 退出码：
#   0  两条都通过
#   1  任意一条失败
# 用法：bash scripts/verify_phase40.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo "=========================================="
echo "Phase 40 Verification (Tech-Debt Cleanup)"
echo "=========================================="
echo

# ----- 标准 1: validator 100% pass -----
echo "== Standard 1: frontmatter validator 100% pass =="
if bash scripts/validate_debug_frontmatter.sh 2>&1 | tee /tmp/phase40_validator.log; then
  pass_line=$(grep "pass rate:" /tmp/phase40_validator.log | head -1)
  if [[ "$pass_line" == *"100.0%"* ]]; then
    echo "[OK] validator 100% pass"
    STD1=0
  else
    echo "[FAIL] validator pass rate not 100%: $pass_line"
    STD1=1
  fi
else
  echo "[FAIL] validator exited non-zero"
  STD1=1
fi
echo

# ----- 标准 2: audit-open debug_sessions < 5 -----
echo "== Standard 2: audit-open debug_sessions < 5 =="
if ! command -v gsd-sdk >/dev/null 2>&1; then
  echo "[FAIL] gsd-sdk not in PATH"
  STD2=1
else
  audit_json=$(gsd-sdk query audit-open 2>&1)
  debug_count=$(echo "$audit_json" | grep -oE '"debug_sessions"[[:space:]]*:[[:space:]]*[0-9]+' | grep -oE '[0-9]+' | head -1)
  if [[ -z "$debug_count" ]]; then
    echo "[FAIL] cannot parse debug_sessions from audit-open output"
    STD2=1
  elif [[ "$debug_count" -lt 5 ]]; then
    echo "[OK] debug_sessions=$debug_count < 5"
    STD2=0
  else
    echo "[FAIL] debug_sessions=$debug_count >= 5"
    STD2=1
  fi
fi
echo

# ----- 总结 -----
echo "=========================================="
echo "Summary"
echo "=========================================="
if [[ $STD1 -eq 0 ]]; then
  echo "  Standard 1 (validator 100% pass): PASS"
else
  echo "  Standard 1 (validator 100% pass): FAIL"
fi
if [[ $STD2 -eq 0 ]]; then
  echo "  Standard 2 (audit-open < 5):        PASS"
else
  echo "  Standard 2 (audit-open < 5):        FAIL"
fi

if [[ $STD1 -eq 0 && $STD2 -eq 0 ]]; then
  echo
  echo "[ALL PASS] Phase 40 verification SUCCESS"
  exit 0
else
  echo
  echo "[FAILED] Phase 40 verification FAILED"
  exit 1
fi
