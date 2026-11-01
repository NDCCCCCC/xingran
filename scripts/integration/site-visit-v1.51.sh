#!/usr/bin/env bash
# site-visit-v1.51.sh — Phase 129 真机 UAT 单动作执行器（Phase 122/129 端口写全管线）
#
# 流程：dry-run 预览（零设备连接）→ 人工确认 → [--apply] 实发 → 采集验证命令输出
#       → 记录结果。默认只 dry-run；实发必须显式 --apply。
#
# 用法：
#   site-visit-v1.51.sh --host 10.62.1.10 --port 22 --user admin --vendor huawei \
#       --port-name GE1/0/0 --action set_access_vlan --params '{"vlanId":100}' \
#       [--apply] [--base-url http://127.0.0.1:9000] [--out results.txt]
#
# 动作（action）与 params（JSON）：
#   shutdown | undo_shutdown | description | dot1x_enable | dot1x_disable |
#   set_access_vlan | port_binding | trunk_set_native | trunk_set_allowed |
#   trunk_set_hybrid | portsec_enable | portsec_disable | portsec_max_mac |
#   portsec_sticky | portsec_static_bind
#   params 示例：{"description":"x"} / {"vlanId":100} /
#               {"op":"add","ipAddress":"10.62.25.5","macAddress":"AA-BB-CC-DD-EE-FF"} /
#               {"nativeVlan":100} / {"allowedVlans":[100,200],"mode":"add"} /
#               {"hybridTagged":[30],"hybridUntagged":[10,20]} /
#               {"maxMacCount":5} / {"macAddress":"AA-BB-CC-DD-EE-FF","vlanId":100}
#
# 凭据：环境变量 SITE_USER / SITE_PASS（或 SSH_KEY 私钥路径），禁止写入本脚本。
#   后端 API 鉴权：SITE_API_TOKEN（JWT access token，向运行中后端认证）。
#
# 退出码：0=通过 / 2=参数错 / 3=连接失败 / 4=命令被设备拒 / 5=验证不符 / 6=后端不可达
set -uo pipefail

BASE_URL="http://127.0.0.1:9000"
APPLY=0
OUT=""
HOST="" ; PORT="" ; USER_NAME="" ; VENDOR="" ; PORT_NAME="" ; ACTION="" ; PARAMS="{}"

usage() { sed -n '2,30p' "$0"; exit 2; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host) HOST="$2"; shift 2 ;;
    --port) PORT="$2"; shift 2 ;;
    --user) USER_NAME="$2"; shift 2 ;;
    --vendor) VENDOR="$2"; shift 2 ;;
    --port-name) PORT_NAME="$2"; shift 2 ;;
    --action) ACTION="$2"; shift 2 ;;
    --params) PARAMS="$2"; shift 2 ;;
    --apply) APPLY=1; shift ;;
    --base-url) BASE_URL="$2"; shift 2 ;;
    --out) OUT="$2"; shift 2 ;;
    -h|--help) usage ;;
    *) echo "未知参数: $1" >&2; exit 2 ;;
  esac
done

[[ -z "$HOST" || -z "$USER_NAME" || -z "$VENDOR" || -z "$PORT_NAME" || -z "$ACTION" ]] && { echo "缺少必填参数"; usage; }
[[ -z "$PORT" ]] && PORT=22
if [[ -z "${SITE_API_TOKEN:-}" ]]; then echo "错误：SITE_API_TOKEN 未设置（后端 JWT）" >&2; exit 2; fi

log() { echo "[$(date '+%F %T')] $*" | tee -a "${OUT:-/dev/null}"; }

# ---------- 步骤 1：定位 portId（backend audit/端口查询，按接口名） ----------
log "== site-visit v1.51 action=$ACTION vendor=$VENDOR port=$PORT_NAME apply=$APPLY =="

# 端口 UUID 由任务下发方提供（UAT 清单执行时手工带入），或用环境变量 PORT_ID
PORT_ID="${PORT_ID:-}"
if [[ -z "$PORT_ID" ]]; then
  echo "错误：PORT_ID 环境变量未设置（目标端口 UUID，从设备详情页或 sys_device_port_status 取）" >&2
  exit 2
fi
DEVICE_ID="${DEVICE_ID:-}"
if [[ -z "$DEVICE_ID" ]]; then
  echo "错误：DEVICE_ID 环境变量未设置（目标设备 UUID）" >&2
  exit 2
fi

api() { # api <method> <path> <json-body|-> ; 输出 body；退出码 3/6
  local method="$1" path="$2" body="$3"
  local args=(-s -w '\n%{http_code}' -X "$method" -H "Authorization: Bearer ${SITE_API_TOKEN}" \
      -H 'Content-Type: application/json' --max-time 60 "${BASE_URL}${path}")
  [[ "$body" != "-" ]] && args+=(-d "$body")
  local resp
  resp=$(curl "${args[@]}") || { log "ERROR: 后端不可达 ${BASE_URL}"; exit 6; }
  local code="${resp##*$'\n'}"
  echo "${resp%$'\n'}"
  [[ "$code" == "200" ]] && return 0
  return 1
}

# ---------- 步骤 2：dry-run 预览（零设备连接，PORTW-01） ----------
preview_body=$(printf '{"deviceId":"%s","action":"%s","portIds":["%s"],"params":%s}' \
  "$DEVICE_ID" "$ACTION" "$PORT_ID" "$PARAMS")
# trunk/portsec 动作参数平铺到请求根（契约：preview 与写共用参数形状）
case "$ACTION" in
  trunk_set_native|trunk_set_allowed|trunk_set_hybrid|portsec_*)
    preview_body=$(printf '{"deviceId":"%s","action":"%s","portIds":["%s"],%s}' \
      "$DEVICE_ID" "$ACTION" "$PORT_ID" "${PARAMS#{}") ;;
esac

log "== 步骤 1/3 dry-run 预览（不下发）=="
preview_resp=$(api POST "/network/ports/write/preview" "$preview_body") || {
  log "ERROR: dry-run 被拒绝（参数/厂商支持性/NoOp 语义见响应）: $preview_resp"
  echo "$preview_resp" >> "${OUT:-/dev/null}"
  exit 4
}
log "$preview_resp" 
echo "$preview_resp" >> "${OUT:-/dev/null}"

# 预览里 skipped（NoOp）则无需实发
if echo "$preview_resp" | grep -q '"noOp":true'; then
  log "== NoOp：pre-state 已匹配目标状态，跳过实发 =="
  exit 0
fi

# ---------- 步骤 3：人工确认 + 实发 ----------
if [[ $APPLY -ne 1 ]]; then
  log "== dry-run 完成。实发请加 --apply（人工确认预览命令序列后）=="
  exit 0
fi

echo "--------------------------------------------------------------"
echo "即将对 ${VENDOR}:${PORT_NAME} 实发 action=${ACTION}，预览命令见上方。"
read -r -p "确认实发？输入 yes 继续: " confirm
[[ "$confirm" == "yes" ]] || { log "人工确认未通过，终止"; exit 0; }

log "== 步骤 2/3 实发 =="
case "$ACTION" in
  trunk_*)
    write_body=$(printf '{"portId":"%s","action":"%s",%s}' "$PORT_ID" "$ACTION" "${PARAMS#{}")
    api POST "/network/ports/write/trunk" "$write_body" >/dev/null || { log "ERROR: 实发失败"; exit 4; }
    ;;
  portsec_*)
    write_body=$(printf '{"portId":"%s","action":"%s",%s}' "$PORT_ID" "$ACTION" "${PARAMS#{}")
    api POST "/network/ports/write/portsec" "$write_body" >/dev/null || { log "ERROR: 实发失败"; exit 4; }
    ;;
  set_access_vlan)
    vlan=$(echo "$PARAMS" | sed -n 's/.*"vlanId":\([0-9]*\).*/\1/p')
    api POST "/network/ports/write/set-access-vlan" \
      "$(printf '{"portId":"%s","vlanId":%s}' "$PORT_ID" "$vlan")" >/dev/null || { log "ERROR: 实发失败"; exit 4; }
    ;;
  port_binding)
    op=$(echo "$PARAMS" | sed -n 's/.*"op":"\([a-z]*\)".*/\1/p')
    ip=$(echo "$PARAMS" | sed -n 's/.*"ipAddress":"\([^"]*\)".*/\1/p')
    mac=$(echo "$PARAMS" | sed -n 's/.*"macAddress":"\([^"]*\)".*/\1/p')
    api POST "/network/ports/write/port-binding" \
      "$(printf '{"portId":"%s","op":"%s","ipAddress":"%s","macAddress":"%s"}' "$PORT_ID" "$op" "$ip" "$mac")" >/dev/null || { log "ERROR: 实发失败"; exit 4; }
    ;;
  shutdown)
    api POST "/network/ports/write/shutdown" "$(printf '{"portId":"%s"}' "$PORT_ID")" >/dev/null || { log "ERROR: 实发失败"; exit 4; }
    ;;
  undo_shutdown)
    api POST "/network/ports/write/undo-shutdown" "$(printf '{"portId":"%s"}' "$PORT_ID")" >/dev/null || { log "ERROR: 实发失败"; exit 4; }
    ;;
  description)
    desc=$(echo "$PARAMS" | sed -n 's/.*"description":"\([^"]*\)".*/\1/p')
    api POST "/network/ports/write/description" "$(printf '{"portId":"%s","description":"%s"}' "$PORT_ID" "$desc")" >/dev/null || { log "ERROR: 实发失败"; exit 4; }
    ;;
  dot1x_enable)
    api POST "/network/ports/write/dot1x-enable" "$(printf '{"portId":"%s"}' "$PORT_ID")" >/dev/null || { log "ERROR: 实发失败"; exit 4; }
    ;;
  dot1x_disable)
    api POST "/network/ports/write/dot1x-disable" "$(printf '{"portId":"%s"}' "$PORT_ID")" >/dev/null || { log "ERROR: 实发失败"; exit 4; }
    ;;
  *) echo "不支持的动作: $ACTION" >&2; exit 2 ;;
esac
log "实发完成（响应已含 PortResult，回滚状态看 extra.rollback）"

# ---------- 步骤 4：验证命令输出采集（由操作者按 UAT 清单"验证命令"列手动执行） ----------
log "== 步骤 3/3 验证 =="
log "请按 129-HUMAN-UAT.md 对应条目的『验证命令』列在设备上采集输出，"
log "并将 实际输出/结论 抄录 129-RESULTS.md 对应行。脚本不代执行设备侧验证命令"
log "（验证命令依赖交互式 SSH 会话与人工判读）。"

log "== done（结果记录：${OUT:-stdout}）=="
exit 0
