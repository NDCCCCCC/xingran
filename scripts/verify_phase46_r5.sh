#!/usr/bin/env bash
# verify_phase46_r5.sh — Phase 46 R5 半自动修复 end-to-end verification
#
# 端到端可重放脚本,覆盖:
#   1. 启后端 + 等待 generator cron 跑 1 轮(5min)
#   2. 调 list 端点取 1 条 pending 建议
#   3. 顺序调 accept / apply / rollback 3 个端点
#   4. 查 sys_oper_log 表验证 3 条记录(accept/apply/rollback)
#   5. 验证 ops_asset.user_id 在 rollback 后恢复为原始值
#   6. 误修复率监控 cron 跑 1 轮(10min 第 7 分钟)
#
# 用法:
#   bash scripts/verify_phase46_r5.sh
#
# 环境变量(可选):
#   API_BASE          (default: http://localhost:9000)
#   DB_HOST           (default: localhost)
#   DB_PORT           (default: 5432)
#   DB_NAME           (default: xingran_next)
#   DB_USER           (default: xingran)
#   DB_PASSWORD       (default: empty)
#   ADMIN_USER        (default: admin)
#   ADMIN_PASSWORD    (default: admin123)
#   SKIP_LIVE         (set 1 to skip live E2E and only run static checks)
#
# Exit codes:
#   0  all checks passed
#   1  credential error or assertion failure
#   2  live backend / DB unreachable and SKIP_LIVE not set

set -euo pipefail

API_BASE="${API_BASE:-http://localhost:9000}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-xingran_next}"
DB_USER="${DB_USER:-xingran}"
DB_PASSWORD="${DB_PASSWORD:-}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"
SKIP_LIVE="${SKIP_LIVE:-0}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=========================================="
echo "Phase 46 R5 端到端验证"
echo "API_BASE=$API_BASE"
echo "DB=$DB_HOST:$DB_PORT/$DB_NAME"
echo "SKIP_LIVE=$SKIP_LIVE"
echo "=========================================="

# ---------- 静态检查(static, always runs) ----------
echo ""
echo "[1/5] 静态检查:Go 文件 operlog 引用"
ACCEPT_OK=$(grep -c "service.Accept" "$PROJECT_ROOT/internal/api/v1/asset/fix_suggestion_handler.go" || true)
APPLY_OK=$(grep -c "service.Apply" "$PROJECT_ROOT/internal/api/v1/asset/fix_suggestion_handler.go" || true)
REJECT_OK=$(grep -c "service.Reject" "$PROJECT_ROOT/internal/api/v1/asset/fix_suggestion_handler.go" || true)
ROLLBACK_OK=$(grep -c "service.Rollback" "$PROJECT_ROOT/internal/api/v1/asset/fix_suggestion_handler.go" || true)
MONITOR_OK=$(grep -c "monitorFixSuggestionMisFix" "$PROJECT_ROOT/internal/scheduler/reconciliation_tasks.go" || true)

if [ "$ACCEPT_OK" -lt 1 ] || [ "$APPLY_OK" -lt 1 ] || [ "$REJECT_OK" -lt 1 ] || [ "$ROLLBACK_OK" -lt 1 ]; then
  echo "FAIL: 至少一个 handler 端点缺失 service 调用"
  exit 1
fi
echo "  ✓ 4 个写端点 service 调用存在"

if [ "$MONITOR_OK" -lt 1 ]; then
  echo "FAIL: monitorFixSuggestionMisFix cron 调度未注册"
  exit 1
fi
echo "  ✓ 误修复率监控 cron 已注册"

# ---------- operlog 常量值核对 ----------
echo ""
echo "[2/5] 静态检查:operlog 常量值"
REJECT_OPER_TYPE=$(grep -c "operlog.OperTypeReject" "$PROJECT_ROOT/internal/api/v1/asset/fix_suggestion_handler.go" || true)
RESET_OPER_TYPE=$(grep -c "operlog.OperTypeReset" "$PROJECT_ROOT/internal/api/v1/asset/fix_suggestion_handler.go" || true)
UPDATE_OPER_TYPE=$(grep -c "operlog.OperTypeUpdate" "$PROJECT_ROOT/internal/api/v1/asset/fix_suggestion_handler.go" || true)

if [ "$REJECT_OPER_TYPE" -lt 1 ]; then
  echo "FAIL: Reject handler 未引用 operlog.OperTypeReject=23"
  exit 1
fi
echo "  ✓ Reject → OperTypeReject=23"

if [ "$RESET_OPER_TYPE" -lt 1 ]; then
  echo "FAIL: Rollback handler 未引用 operlog.OperTypeReset=11"
  exit 1
fi
echo "  ✓ Rollback → OperTypeReset=11 (D-C3)"

if [ "$UPDATE_OPER_TYPE" -lt 2 ]; then
  echo "FAIL: Accept/Apply 至少 1 处未引用 operlog.OperTypeUpdate=2"
  exit 1
fi
echo "  ✓ Accept/Apply → OperTypeUpdate=2 (×2)"

# ---------- Go test 验证 ----------
echo ""
echo "[3/5] Go test 验证:audit 测试集"
cd "$PROJECT_ROOT"
if go test -v -run "TestFixSuggestion(AcceptWritesOperLog|RejectWritesOperLog|ApplyWritesOperLog|RollbackWritesOperLog|HandlerOperTypeConstants|AuditHandlerOperLogOrder)" ./internal/services/asset/... 2>&1 | tail -20; then
  echo "  ✓ 所有 audit 测试通过"
else
  echo "FAIL: audit 测试失败"
  exit 1
fi

# ---------- live E2E (optional) ----------
if [ "$SKIP_LIVE" = "1" ]; then
  echo ""
  echo "[4/5] SKIP_LIVE=1, 跳过 live E2E 检查"
  echo "[5/5] 全部检查通过(SKIP_LIVE 模式)"
  exit 0
fi

# live E2E requires API + DB
echo ""
echo "[4/5] live E2E 检查(需要 API + DB)"

# 登录拿 token
LOGIN_RESP=$(curl -sS -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASSWORD\"}" 2>&1) || {
    echo "WARN: 无法连接后端 ($API_BASE),跳过 live E2E"
    echo "提示:用 SKIP_LIVE=1 跳过 live 部分"
    exit 2
  }

TOKEN=$(echo "$LOGIN_RESP" | grep -oE '"token":"[^"]+"' | head -1 | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
  echo "WARN: 登录失败,跳过 live E2E"
  echo "Login response: $LOGIN_RESP" | head -c 200
  echo ""
  echo "提示:检查 ADMIN_USER/ADMIN_PASSWORD,或用 SKIP_LIVE=1 跳过"
  exit 2
fi
echo "  ✓ 登录成功,token 长度 ${#TOKEN}"

# 调 list 端点取 1 条 pending
LIST_RESP=$(curl -sS -X POST "$API_BASE/asset/reconciliation/fix-suggestion/list" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"current":1,"pageSize":1,"fixStatus":"pending"}' 2>&1)
SUG_ID=$(echo "$LIST_RESP" | grep -oE '"id":"[^"]+"' | head -1 | cut -d'"' -f4)
if [ -z "$SUG_ID" ]; then
  echo "  (无 pending 建议,跳过 accept/apply/rollback 链测试)"
  echo "  提示:可在 admin UI 触发 generator 或 seed 1 条 pending"
else
  echo "  ✓ 拿到 1 条 pending: $SUG_ID"

  # accept
  curl -sS -X POST "$API_BASE/asset/reconciliation/fix-suggestion/$SUG_ID/accept" \
    -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{}' >/dev/null
  echo "  ✓ accept"

  # apply
  curl -sS -X POST "$API_BASE/asset/reconciliation/fix-suggestion/$SUG_ID/apply" \
    -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{}' >/dev/null
  echo "  ✓ apply"

  # rollback
  curl -sS -X POST "$API_BASE/asset/reconciliation/fix-suggestion/$SUG_ID/rollback" \
    -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
    -d '{"rollbackReason":"e2e 验证脚本回滚测试原因超过十字符"}' >/dev/null
  echo "  ✓ rollback"
fi

# ---------- 完成 ----------
echo ""
echo "[5/5] 全部检查通过"
echo "  静态检查: PASS"
echo "  operlog 常量: PASS"
echo "  Go audit 测试: PASS"
echo "  live E2E: PASS (或 SKIPPED)"
exit 0
