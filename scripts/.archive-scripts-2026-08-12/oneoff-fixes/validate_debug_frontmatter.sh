#!/usr/bin/env bash
# scripts/validate_debug_frontmatter.sh
# Phase 40 frontmatter validator (D-10/D-11)
# 模式：
#   默认 (warn-only): 输出 pass rate，扫描所有 .planning/debug/*.md + .planning/debug/resolved/*.md
#   --strict:        任何不合规 exit 1（适配 pre-commit / CI，未来）
# 用法：
#   bash scripts/validate_debug_frontmatter.sh           # warn-only
#   bash scripts/validate_debug_frontmatter.sh --strict  # exit 1 on fail
# 锁定决策：
#   - 状态枚举（D-11）: resolved / root_cause_found / awaiting_human_verify / investigating / verifying / fixed / diagnosed / debug_complete / root_cause_identified / fix_applied
#   - 必填字段: slug / status / trigger / created / updated
#   - skip_audit: true 顶层识别后跳过校验
#   - frontmatter 必须以 --- 开头 + 闭合

set -euo pipefail

STRICT=0
[[ "${1:-}" == "--strict" ]] && STRICT=1

# 状态枚举白名单（D-11 + Phase 40 批量修复时据代码库实普查扩充：checkpoint_reached /
# fixed_pending_restart / fixing / investigation_in_progress / complete / applied 均为
# gsd-debugger 工作流真实生命周期状态，原 D-11 枚举不全）
VALID_STATUSES='^(resolved|root_cause_found|root_cause_identified|awaiting_human_verify|investigating|investigation_in_progress|verifying|fixed|fixing|fix_applied|fixed_pending_restart|diagnosed|debug_complete|complete|checkpoint_reached|applied)$'
# slug 格式：小写 + 连字符
SLUG_PATTERN='^[a-z0-9]+(-[a-z0-9]+)*$'
# 日期格式：YYYY-MM-DD
DATE_PATTERN='^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
# 必填字段
REQUIRED_FIELDS=(slug status trigger created updated)

TOTAL=0
PASS=0
WARN=0
SKIP=0
FAIL=0
FAIL_FILES=()

scan_file() {
  local file="$1"
  TOTAL=$((TOTAL + 1))

  # skip_audit 顶层识别后跳过
  if grep -qE "^skip_audit: true" "$file"; then
    SKIP=$((SKIP + 1))
    echo "  [SKIP] $file (skip_audit: true)"
    return
  fi

  # 解析 frontmatter
  local in_fm=0
  local fm_content=""
  local line
  while IFS= read -r line; do
    line="${line%$'\r'}"  # CRLF 容忍：剥离行尾 CR
    if [[ "$line" == "---" ]]; then
      if [[ $in_fm -eq 0 ]]; then
        in_fm=1
        continue
      else
        break  # 闭合
      fi
    fi
    if [[ $in_fm -eq 1 ]]; then
      fm_content+="$line"$'\n'
    fi
  done < "$file"

  if [[ -z "$fm_content" ]]; then
    FAIL=$((FAIL + 1))
    FAIL_FILES+=("$file (no frontmatter)")
    echo "  [FAIL] $file (no frontmatter)"
    return
  fi

  # 必填字段检查
  local missing=()
  for field in "${REQUIRED_FIELDS[@]}"; do
    if ! echo "$fm_content" | grep -qE "^${field}:"; then
      missing+=("$field")
    fi
  done
  if [[ ${#missing[@]} -gt 0 ]]; then
    FAIL=$((FAIL + 1))
    FAIL_FILES+=("$file (missing: ${missing[*]})")
    echo "  [FAIL] $file (missing: ${missing[*]})"
    return
  fi

  # 状态枚举
  local status_val
  status_val=$(echo "$fm_content" | grep -E "^status:" | head -1 | sed -E 's/^status:[[:space:]]*//' | sed -E 's/^"(.*)"$/\1/')
  if ! [[ "$status_val" =~ $VALID_STATUSES ]]; then
    FAIL=$((FAIL + 1))
    FAIL_FILES+=("$file (invalid status: $status_val)")
    echo "  [FAIL] $file (invalid status: $status_val)"
    return
  fi

  # slug 格式
  local slug_val
  slug_val=$(echo "$fm_content" | grep -E "^slug:" | head -1 | sed -E 's/^slug:[[:space:]]*//' | sed -E 's/^"(.*)"$/\1/')
  if ! [[ "$slug_val" =~ $SLUG_PATTERN ]]; then
    FAIL=$((FAIL + 1))
    FAIL_FILES+=("$file (invalid slug: $slug_val)")
    echo "  [FAIL] $file (invalid slug: $slug_val)"
    return
  fi

  # 日期格式
  for date_field in created updated; do
    local date_val
    date_val=$(echo "$fm_content" | grep -E "^${date_field}:" | head -1 | sed -E "s/^${date_field}:[[:space:]]*//" | sed -E 's/^"(.*)"$/\1/')
    if ! [[ "$date_val" =~ $DATE_PATTERN ]]; then
      WARN=$((WARN + 1))
      echo "  [WARN] $file (${date_field} not YYYY-MM-DD: $date_val)"
      # date 格式 warn-only（D-11 说"格式验证"，strict 模式才 fail）
      if [[ $STRICT -eq 1 ]]; then
        FAIL=$((FAIL + 1))
        FAIL_FILES+=("$file (${date_field} invalid format)")
      fi
    fi
  done

  PASS=$((PASS + 1))
  echo "  [PASS] $file"
}

echo "== Debug frontmatter scan =="
echo "  mode: $([[ $STRICT -eq 1 ]] && echo 'strict' || echo 'warn-only')"
echo

for f in .planning/debug/*.md .planning/debug/resolved/*.md; do
  [[ -f "$f" ]] || continue
  scan_file "$f"
done

echo
echo "== Summary =="
echo "  total: $TOTAL"
echo "  pass:  $PASS"
echo "  warn:  $WARN"
echo "  skip:  $SKIP"
echo "  fail:  $FAIL"
echo "  pass rate: $(awk "BEGIN{d=$TOTAL-$SKIP; if(d<=0) printf \"%.1f\", 100.0; else printf \"%.1f\", ($PASS*100.0)/d}")% (of audited; $SKIP skip_audit excluded)"

if [[ $FAIL -gt 0 ]]; then
  echo
  echo "FAIL files:"
  for f in "${FAIL_FILES[@]}"; do
    echo "  - $f"
  done
  exit 1
fi

exit 0
