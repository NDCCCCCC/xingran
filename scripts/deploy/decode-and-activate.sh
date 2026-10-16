#!/usr/bin/env bash
# decode-and-activate.sh — server 端（通过 ssh 'bash -s' 触发）
# 接收 upload-binary.sh 上传的分片,base64 -d 拼回 + sha256 校验,
# 然后 exec 调 deploy-remote.sh 走原 activate 流程(备份/替换/重启/health/rollback)。
#
# 用法: deploy.yml 走 `ssh 'bash -s' -- $VERSION < decode-and-activate.sh`
#   $VERSION: ldflags 注入的 main.Version,用于 /health 校验

set -euo pipefail

UPLOAD=/opt/xingran/upload
APP=/opt/xingran
SHA_FILE="$UPLOAD/xingran-backend.new.sha256"
SIZE_FILE="$UPLOAD/xingran-backend.new.size"
VERSION_FILE="$UPLOAD/xingran-backend.new.version"
NEW="$APP/xingran-backend.new"
VERSION="${1:-}"

[ -f "$SHA_FILE" ] || { echo "!! missing $SHA_FILE" >&2; exit 1; }
[ -f "$SIZE_FILE" ] || { echo "!! missing $SIZE_FILE" >&2; exit 1; }
[ -n "$VERSION" ]  || { echo "!! usage: $0 <version>" >&2; exit 1; }

# 检查 chunk 文件齐
CHUNK_COUNT=$(ls -1 "$UPLOAD"/chunk.*.b64 2>/dev/null | wc -l | tr -d ' ')
if (( CHUNK_COUNT == 0 )); then
  echo "!! no chunk.*.b64 files in $UPLOAD" >&2
  exit 1
fi

# 1. 磁盘预检查
FREE_KB=$(df -k "$APP" | tail -1 | awk '{print $4}')
if (( FREE_KB < 204800 )); then
  echo "!! insufficient disk: ${FREE_KB}KB free (need >= 200MB) on $APP" >&2
  exit 1
fi

# 2. 校验 server 记录下来的 version 与 ssh 传入的 version 一致
UPLOADED_VERSION=$(cat "$VERSION_FILE")
if [[ "$UPLOADED_VERSION" != "$VERSION" ]]; then
  echo "!! version mismatch: upload=${UPLOADED_VERSION} ssh=${VERSION}" >&2
  exit 1
fi

# 3. 严格按数字顺序 base64 -d 拼回 (chunk.0001.b64, chunk.0002.b64, ...)
#    base64 -d 容忍任意空白/换行,ls -1 默认字典序 = 数字序 (左 0 填充)
: > "$NEW"
for f in $(ls -1 "$UPLOAD"/chunk.*.b64 | sort); do
  base64 -d "$f" >> "$NEW"
done

# 4. sha256 校验
EXPECTED_SHA=$(awk '{print $1}' "$SHA_FILE")
ACTUAL_SHA=$(sha256sum "$NEW" | awk '{print $1}')
if [[ "$EXPECTED_SHA" != "$ACTUAL_SHA" ]]; then
  echo "!! sha256 mismatch: expected=${EXPECTED_SHA} actual=${ACTUAL_SHA}" >&2
  rm -f "$NEW"
  exit 1
fi

# 5. size 校验
EXPECTED_SIZE=$(cat "$SIZE_FILE")
ACTUAL_SIZE=$(stat -c %s "$NEW")
if [[ "$EXPECTED_SIZE" != "$ACTUAL_SIZE" ]]; then
  echo "!! size mismatch: expected=${EXPECTED_SIZE} actual=${ACTUAL_SIZE}" >&2
  rm -f "$NEW"
  exit 1
fi

# 6. chmod +x
sudo chmod 0755 "$NEW"

# 7. 清理 upload/ 临时文件
rm -f "$UPLOAD"/chunk.*.b64 "$SHA_FILE" "$SIZE_FILE" "$VERSION_FILE"

echo "decode + verify OK: ${CHUNK_COUNT} chunks, size=${ACTUAL_SIZE} sha256=${ACTUAL_SHA}"

# 8. exec 触发原有 activate 流程(零改动复用)
exec sudo -n bash "$APP/deploy/deploy-remote.sh" "$VERSION"
