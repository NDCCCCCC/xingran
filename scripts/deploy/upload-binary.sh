#!/usr/bin/env bash
# upload-binary.sh — runner 端（GitHub Actions）
# 把 out/xingran-backend 拆成 500KB 的 base64 短流分片上传到服务器，
# 走 SSH ControlMaster 复用连接。
#
# 背景：scp/rsync/sftp put 75MB 二进制在 GitHub runner -> 212.129.154.78
# 链路上一致 15 分钟静默卡死（本地 1KB 短流 0.6s 成功）。拆成 158x500KB
# 短流 + ControlMaster 复用可彻底规避长流 TCP 状态机被中间盒 RST。
#
# 流程：
#   1. 写私钥 + known_hosts 到 ~/.ssh/(fingerprint self-check)
#   2. SSH ControlMaster 握手复用
#   3. split -b 500000 -d -a 4 out/xingran-backend -> upload/chunk.0000 ..
#   4. 逐片 base64 -w 76 | ssh ... "cat >> /opt/xingran/upload/xingran-backend.new.b64"
#   5. 传 sha256 + size (server 端 base64 -d + sha256 校验)
#   6. 关闭 ControlMaster
#
# 用法: upload-binary.sh <SSH_HOST> <SSH_USER> <SSH_PRIVATE_KEY> <SSH_KNOWN_HOSTS> <VERSION>
#   VERSION 传给 server 端 decode-and-activate.sh,最终给 deploy-remote.sh

set -euo pipefail

SSH_HOST="${1:-}"
SSH_USER="${2:-ubuntu}"
SSH_KEY="${3:-}"
SSH_KNOWN_HOSTS="${4:-}"
VERSION="${5:-}"

if [[ -z "$SSH_HOST" || -z "$SSH_KEY" || -z "$VERSION" ]]; then
  echo "::error::usage: $0 <SSH_HOST> <SSH_USER> <SSH_PRIVATE_KEY> <SSH_KNOWN_HOSTS> <VERSION>" >&2
  exit 1
fi

PORT="${SSH_PORT:-22}"
BIN="${BIN:-out/xingran-backend}"
CHUNK_BYTES="${CHUNK_BYTES:-500000}"
EXPECTED_FP="SHA256:9RrflpLM3hblmkElV/xRaMGqmGz5oikfVatiOl92PJ8"

# 一些 sanity check
test -f "$BIN" || { echo "::error::$BIN not found"; exit 1; }

LOCAL_SIZE=$(stat -c %s "$BIN")
LOCAL_SHA=$(sha256sum "$BIN" | awk '{print $1}')
echo "binary: $BIN  size=${LOCAL_SIZE}  sha256=${LOCAL_SHA}"

# 1. Setup SSH key + known_hosts
install -m 700 -d ~/.ssh
printf '%s\n' "$SSH_KEY" > ~/.ssh/id_ed25519
chmod 600 ~/.ssh/id_ed25519
printf '%s\n' "$SSH_KNOWN_HOSTS" >> ~/.ssh/known_hosts

GOT_FP=$(ssh-keygen -lf ~/.ssh/id_ed25519 | awk '{print $2}')
echo "deploy key fingerprint: ${GOT_FP}"
if [[ "${GOT_FP}" != "${EXPECTED_FP}" ]]; then
  echo "::error::deploy private key fingerprint mismatch — refusing to connect"
  echo "::error::expected ${EXPECTED_FP}"
  exit 1
fi

SSH_BASE="-p ${PORT} -i ~/.ssh/id_ed25519 -o IdentitiesOnly=yes \
          -o StrictHostKeyChecking=yes -o BatchMode=yes \
          -o ServerAliveInterval=30 -o ServerAliveCountMax=6 \
          -o ConnectTimeout=15"

# 2. ControlMaster socket (PID 防同名)
SOCK="/tmp/ssh-c-$$-${SSH_HOST}-${PORT}"
trap 'ssh -S "$SOCK" -O exit -o BatchMode=yes "${SSH_USER}@${SSH_HOST}" 2>/dev/null || true; rm -f "$SOCK"' EXIT

# 3. 建立主连接
ssh -M -S "$SOCK" -o "ControlPersist=600" $SSH_BASE "${SSH_USER}@${SSH_HOST}" true

# 4. 远端准备
ssh -S "$SOCK" $SSH_BASE "${SSH_USER}@${SSH_HOST}" \
  "sudo install -d -m 0755 /opt/xingran/upload && \
   : > /opt/xingran/upload/xingran-backend.new.b64 && \
   chmod 0644 /opt/xingran/upload/xingran-backend.new.b64"

# 5. split 临时目录
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"; ssh -S "$SOCK" -O exit -o BatchMode=yes "${SSH_USER}@${SSH_HOST}" 2>/dev/null || true; rm -f "$SOCK"' EXIT

split -b "$CHUNK_BYTES" -d -a 4 "$BIN" "$TMP_DIR/chunk."
CHUNK_COUNT=$(ls -1 "$TMP_DIR"/chunk.* | wc -l | tr -d ' ')
echo "chunks: ${CHUNK_COUNT} of ${CHUNK_BYTES} bytes each"

# 6. 逐片上传（严格排序！-d -a 4 + ls sort = 数字序）
i=0
for f in $(ls -1 "$TMP_DIR"/chunk.* | sort); do
  i=$((i + 1))
  base64 -w 76 "$f" | \
    ssh -S "$SOCK" $SSH_BASE "${SSH_USER}@${SSH_HOST}" \
      "cat >> /opt/xingran/upload/xingran-backend.new.b64"
  if (( i % 20 == 0 || i == CHUNK_COUNT )); then
    echo "progress: ${i}/${CHUNK_COUNT}"
  fi
done

# 7. 传 sha256 + size + version
ssh -S "$SOCK" $SSH_BASE "${SSH_USER}@${SSH_HOST}" \
  "printf '%s  xingran-backend\n' '${LOCAL_SHA}' > /opt/xingran/upload/xingran-backend.new.sha256 && \
   printf '%s\n' ${LOCAL_SIZE} > /opt/xingran/upload/xingran-backend.new.size && \
   printf '%s\n' '${VERSION}' > /opt/xingran/upload/xingran-backend.new.version"

# 8. 关闭主连接
ssh -S "$SOCK" -O exit $SSH_BASE "${SSH_USER}@${SSH_HOST}"

echo "upload complete: ${CHUNK_COUNT} chunks, ${LOCAL_SIZE} bytes, sha256=${LOCAL_SHA}"
