#!/usr/bin/env bash
# upload-binary.sh — runner 端（GitHub Actions）
# 把 out/xingran-backend 拆成 500KB 切片,每片 base64 后用一次独立 scp
# 传到服务器(每片自闭合的短 ssh 会话,~666KB 短流)。
#
# 背景:scp/rsync/sftp put 75MB 二进制在 GitHub runner -> 212.129.154.78
# 链路上一致 15 分钟静默卡死;即便用 ssh stdin pipe 拆 158 个 500KB chunk,
# 每次 ssh 会话底层仍是 666KB 连续 pipe 长流,被中间盒 RST 卡死(32118118412)。
# 改用 scp 短文件传输:scp 命令本身一开一关,每片 < 1s,与长流无关。
#
# 流程:
#   1. 写私钥 + known_hosts 到 ~/.ssh/(fingerprint self-check)
#   2. server 端准备 upload/ 目录
#   3. split -b 500000 -d -a 4 -> chunk.0000, chunk.0001, ...
#   4. 逐片 base64 -w 76 + scp 短传(每片独立 ssh 会话)
#   5. 传 sha256 + size + version(server 端 base64 -d + sha256 校验)
#   6. 清理临时文件
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

# 通用 scp/ssh 选项
SCP_OPTS=(-P "$PORT" -i ~/.ssh/id_ed25519 -o IdentitiesOnly=yes \
          -o StrictHostKeyChecking=yes -o BatchMode=yes \
          -o ServerAliveInterval=30 -o ServerAliveCountMax=6 \
          -o ConnectTimeout=15)
SSH_OPTS=(-p "$PORT" -i ~/.ssh/id_ed25519 -o IdentitiesOnly=yes \
          -o StrictHostKeyChecking=yes -o BatchMode=yes \
          -o ServerAliveInterval=30 -o ServerAliveCountMax=6 \
          -o ConnectTimeout=15)

# 2. 远端准备
ssh "${SSH_OPTS[@]}" "${SSH_USER}@${SSH_HOST}" \
  "sudo install -d -m 0755 /opt/xingran/upload && \
   sudo rm -f /opt/xingran/upload/chunk.*.b64"

# 3. split 临时目录
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

split -b "$CHUNK_BYTES" -d -a 4 "$BIN" "$TMP_DIR/chunk."
CHUNK_COUNT=$(ls -1 "$TMP_DIR"/chunk.* | wc -l | tr -d ' ')
echo "chunks: ${CHUNK_COUNT} of ${CHUNK_BYTES} bytes each"

# 4. 逐片 base64 + scp 短传
# 每片 ~666KB 短流,scp 自闭合连接(每片一开一关)。
# ls 默认字典序 = 数字序(左 0 填充保证 chunk.0009 < chunk.0010)
i=0
for f in $(ls -1 "$TMP_DIR"/chunk.* | sort); do
  i=$((i + 1))
  base64 -w 76 "$f" > "$f.b64"
  scp "${SCP_OPTS[@]}" \
    "$f.b64" \
    "${SSH_USER}@${SSH_HOST}:/opt/xingran/upload/chunk.$(printf '%04d' $i).b64"
  if (( i % 10 == 0 || i == CHUNK_COUNT )); then
    echo "progress: ${i}/${CHUNK_COUNT}"
  fi
done

# 5. 传 sha256 + size + version
ssh "${SSH_OPTS[@]}" "${SSH_USER}@${SSH_HOST}" \
  "printf '%s  xingran-backend\n' '${LOCAL_SHA}' \
     > /opt/xingran/upload/xingran-backend.new.sha256 && \
   printf '%s\n' ${LOCAL_SIZE} \
     > /opt/xingran/upload/xingran-backend.new.size && \
   printf '%s\n' '${VERSION}' \
     > /opt/xingran/upload/xingran-backend.new.version"

echo "upload complete: ${CHUNK_COUNT} chunks, ${LOCAL_SIZE} bytes, sha256=${LOCAL_SHA}"