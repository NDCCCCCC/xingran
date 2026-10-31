#!/usr/bin/env bash
# ============================================================================
# smoke-v1.51.sh — v1.51 集成验证窗口冒烟脚本（integration checklist §2/§3）
#
# 用法：
#   BASE=http://localhost:9000 bash scripts/integration/smoke-v1.51.sh worker-auth
#   BASE=http://localhost:9000 bash scripts/integration/smoke-v1.51.sh dry-run <TOKEN>
#   BASE=http://localhost:9000 bash scripts/integration/smoke-v1.51.sh login    # 仅取 token
#
# 依赖：bash + curl（无 jq 依赖，断言用 grep）。
# 说明：真设备端口写实发不在本脚本（Phase 129 真机 UAT）；本脚本只走
#       错误路径与契约校验，安全无副作用（preview/dry-run 均不落交换机）。
#
# ✔ 已修复（2026-11-01，task-15）：sqlite 模式下 Worker 注册曾报
#   "no such function: gen_random_uuid" —— 根因是 Register 原生 SQL 里的
#   PG 专属函数（gen_random_uuid/NOW/RETURNING），非 migration_205（后者
#   本就有 isPostgreSQL 守卫）。修复：worker_service.go Register 增加 sqlite
#   方言分支（GORM upsert：应用层 UUID/时间戳；capabilities 必须 []byte，
#   string 回读 Scan 到 json.RawMessage 会报 unsupported）。PG 路径零变化。
#   矩阵 4/4 PASS 已连续两轮 sqlite 实测。
# ============================================================================
set -u
BASE="${BASE:-http://localhost:9000}"
API="$BASE/api/v1"
PASS=0; FAIL=0

ok()   { PASS=$((PASS+1)); echo "  PASS: $1"; }
bad()  { FAIL=$((FAIL+1)); echo "  FAIL: $1"; }
assert_contains() { # $1=body $2=needle $3=label
  case "$1" in *"$2"*) ok "$3";; *) bad "$3（期望含 '$2'，实际: ${1:0:200}）";; esac
}

# ---------------------------------------------------------------------------
# login：取管理员 token（验证码启用时需人工介入，见 checklist §3 SKIP 条件）
# ---------------------------------------------------------------------------
do_login() {
  local user="${1:-admin}" pass="${2:-admin123}"
  local body
  body=$(curl -sS -X POST "$API/system/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"$user\",\"password\":\"$pass\"}")
  local token
  token=$(printf '%s' "$body" | sed -n 's/.*"accessToken":"\([^"]*\)".*/\1/p')
  if [ -z "$token" ]; then
    echo "登录失败（验证码启用？密码不对？）：${body:0:200}" >&2
    return 1
  fi
  printf '%s' "$token"
}

# ---------------------------------------------------------------------------
# worker-auth：DEBT-03 宽限期矩阵（auth_required=false，默认配置）
# 前置：backend 已用默认配置启动；rpa.worker.auth.provisioning_tokens 含 prov-ok-2026
# ---------------------------------------------------------------------------
worker_auth() {
  echo "== worker-auth 宽限期矩阵 =="
  local r

  # 1) 心跳无 token（宽限期）→ 200
  r=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$API/rpa/workers/smoke-w-1/heartbeat" \
       -H 'Content-Type: application/json' -d '{"workerId":"smoke-w-1","currentTasks":0}')
  [ "$r" = "200" ] && ok "宽限期无 token 心跳 200" || bad "宽限期无 token 心跳（got $r, 期望 200）"

  # 2) 注册带合法 provisioning token → 200 + data.token wk_ 前缀
  r=$(curl -sS -X POST "$API/rpa/workers/register" \
       -H 'Content-Type: application/json' -H 'X-Provisioning-Token: prov-ok-2026' \
       -d '{"workerId":"smoke-w-2","name":"冒烟节点"}')
  assert_contains "$r" '"token":"wk_' "注册换发 Worker Token（wk_ 前缀）"

  # 3) 注册带错误 provisioning token → 401
  r=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$API/rpa/workers/register" \
       -H 'Content-Type: application/json' -H 'X-Provisioning-Token: wrong-token' \
       -d '{"workerId":"smoke-w-3","name":"冒烟节点三"}')
  [ "$r" = "401" ] && ok "错误 provisioning token 401" || bad "错误 provisioning token（got $r, 期望 401）"

  # 4) 换发的 worker token 打心跳 → 200（归属一致）
  local wt
  # 重发一次注册拿 token（步骤 2 的响应即含 token，此处确保变量干净）
  r=$(curl -sS -X POST "$API/rpa/workers/register" \
       -H 'Content-Type: application/json' -H 'X-Provisioning-Token: prov-ok-2026' \
       -d '{"workerId":"smoke-w-2","name":"冒烟节点"}')
  wt=$(printf '%s' "$r" | sed -n 's/.*"token":"\(wk_[^"]*\)".*/\1/p')
  if [ -n "$wt" ]; then
    r=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$API/rpa/workers/smoke-w-2/heartbeat" \
         -H 'Content-Type: application/json' -H "X-Worker-Token: $wt" -d '{"workerId":"smoke-w-2"}')
    [ "$r" = "200" ] && ok "worker token 心跳 200" || bad "worker token 心跳（got $r, 期望 200）"
    # 4b) 归属不符：拿 w-2 的 token 打 w-1 → 403（auth_required=true 才生效；宽限期下 200）
    echo "  （归属校验 403 仅在 auth_required=true 下生效——强制模式为手工步骤，见 checklist §2）"
  else
    bad "未取得 worker token（后端未配置 provisioning_tokens？）"
  fi

  cat <<'EOF'
  --- 切强制模式（手工） ---
  1. configs: rpa.worker.auth.auth_required: true
  2. 重启 backend
  3. 重跑本命令，追加断言：无 token 心跳=401、归属不符=403
EOF
}

# ---------------------------------------------------------------------------
# dry-run：Phase 121 端口写 preview 错误路径契约（无需真设备）
# ---------------------------------------------------------------------------
dry_run() {
  local token="$1"
  echo "== dry-run preview 错误路径契约 =="
  local r code

  # 1) 设备不存在 → 404
  code=$(curl -sS -o /tmp/smoke_body.$$ -w '%{http_code}' -X POST "$API/network/ports/write/preview" \
    -H 'Content-Type: application/json' -H "Authorization: Bearer $token" \
    -d '{"deviceId":"de000000-0000-4000-8000-00000000dead","action":"shutdown","portIds":["po000000-0000-4000-8000-00000000dead"]}')
  r=$(cat /tmp/smoke_body.$$); rm -f /tmp/smoke_body.$$
  [ "$code" = "404" ] && ok "设备不存在 404" || bad "设备不存在（got $code, 期望 404）"
  assert_contains "$r" '"request_id"' "响应契约含 request_id"
  assert_contains "$r" '"timestamp"' "响应契约含 timestamp"

  # 2) VLAN 越界 → 400
  code=$(curl -sS -o /tmp/smoke_body.$$ -w '%{http_code}' -X POST "$API/network/ports/write/preview" \
    -H 'Content-Type: application/json' -H "Authorization: Bearer $token" \
    -d '{"deviceId":"de000000-0000-4000-8000-00000000dead","action":"access_vlan","portIds":["po1"],"vlanId":5000}')
  [ "$code" = "400" ] && ok "VLAN 越界 400" || bad "VLAN 越界（got $code, 期望 400）"
  r=$(cat /tmp/smoke_body.$$ 2>/dev/null); rm -f /tmp/smoke_body.$$

  # 3) 无认证 → 401
  code=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$API/network/ports/write/preview" \
    -H 'Content-Type: application/json' \
    -d '{"deviceId":"x","action":"shutdown","portIds":["p"]}')
  [ "$code" = "401" ] && ok "无认证 401" || bad "无认证（got $code, 期望 401）"

  echo "  （真实设备实发验证 SKIP —— Phase 129 真机 UAT 范围）"
}

# ---------------------------------------------------------------------------
case "${1:-help}" in
  login)       do_login "${2:-}" "${3:-}" && echo ;;
  worker-auth) worker_auth ;;
  dry-run)
    [ -n "${2:-}" ] || { echo "用法: smoke-v1.51.sh dry-run <TOKEN>"; exit 2; }
    dry_run "$2" ;;
  *) sed -n '2,12p' "$0"; exit 2 ;;
esac

echo "== 结果: PASS=$PASS FAIL=$FAIL =="
[ "$FAIL" -eq 0 ]
