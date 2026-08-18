#!/usr/bin/env bash
# fetch-release-and-activate.sh — server 端（通过 ssh 'bash -s' 触发）
#
# 从 GitHub Release 经 gh-proxy.com 镜像拉取部署 tar.gz（解决 GitHub
# runner ↔ 腾讯云国内机房双向链路被限速 ~10KB/s 的问题；实测服务器经
# gh-proxy.com 拉 GitHub Release 达 9.7MB/s）。
#
# 流程：
#   1. df 磁盘预检
#   2. 调 GitHub API（经 gh-proxy）解析 release asset 的 API URL
#   3. curl 带 Bearer token 经 gh-proxy 下载 asset（API URL 返回二进制流）
#   4. sha256 校验（runner 端 build 时生成并传入）
#   5. 解压出 xingran-backend + xingran.service，就位 .new 文件
#   6. exec deploy-remote.sh（备份/原子替换/重启/health/回滚，零改动）
#
# 用法: ssh ... 'bash -s' -- <VERSION> <RELEASE_TAG> <GH_TOKEN> <SHA256> < fetch-release-and-activate.sh
#   GH_TOKEN 仅存在于进程命令行/环境，不落盘。

set -euo pipefail

APP=/opt/xingran
UPLOAD=$APP/upload
NEW=$APP/xingran-backend.new
UNIT_NEW=$APP/deploy/xingran.service.new
REPO="NDCCCCCC/xingran"
PROXY="https://gh-proxy.com"

VERSION="${1:-}"
RELEASE_TAG="${2:-}"
GH_TOKEN="${3:-}"
EXPECTED_SHA="${4:-}"

fail() { echo "!! $*" >&2; exit 1; }

[ -n "$VERSION" ]     || fail "usage: $0 <version> <release_tag> <gh_token> <sha256>"
[ -n "$RELEASE_TAG" ] || fail "missing release tag"
[ -n "$GH_TOKEN" ]    || fail "missing gh token"
[ -n "$EXPECTED_SHA" ] || fail "missing sha256"

# 1. 磁盘预检（tar.gz ~22MB + 解压后 63MB + 备份 63MB）
FREE_KB=$(df -k "$APP" | tail -1 | awk '{print $4}')
[ "$FREE_KB" -ge 204800 ] || fail "insufficient disk: ${FREE_KB}KB free (need >= 200MB)"

sudo install -d -m 0755 "$UPLOAD"
TARBALL="$UPLOAD/xingran-backend.tar.gz"
rm -f "$TARBALL"

# 2. 解析 release asset 的 API URL
echo ">> resolving asset url for release $RELEASE_TAG"
RELEASE_JSON=$(curl -fsSL --max-time 30 \
  -H "Authorization: Bearer $GH_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  "$PROXY/https://api.github.com/repos/$REPO/releases/tags/$RELEASE_TAG") \
  || fail "release lookup failed"

ASSET_API_URL=$(echo "$RELEASE_JSON" \
  | grep -o '"url": *"[^"]*releases/assets/[^"]*"' \
  | head -1 | sed 's/.*"\(https[^"]*\)".*/\1/')
[ -n "$ASSET_API_URL" ] || fail "asset api url not found in release $RELEASE_TAG"

# 3. 下载 asset（Accept: application/octet-stream 让 API 直接返回二进制）
echo ">> downloading asset via gh-proxy: $ASSET_API_URL"
curl -fsSL --max-time 300 \
  -H "Authorization: Bearer $GH_TOKEN" \
  -H "Accept: application/octet-stream" \
  -o "$TARBALL" \
  "$PROXY/$ASSET_API_URL" \
  || fail "asset download failed"

SIZE=$(stat -c %s "$TARBALL")
[ "$SIZE" -gt 10000000 ] || fail "tarball too small ($SIZE bytes), download likely corrupted"
echo ">> downloaded $SIZE bytes"

# 4. sha256 校验
ACTUAL_SHA=$(sha256sum "$TARBALL" | awk '{print $1}')
[ "$EXPECTED_SHA" = "$ACTUAL_SHA" ] || { rm -f "$TARBALL"; fail "sha256 mismatch: expected=$EXPECTED_SHA actual=$ACTUAL_SHA"; }
echo ">> sha256 verified"

# 5. 解压就位 .new 文件
tar -xzf "$TARBALL" -C "$UPLOAD"
[ -f "$UPLOAD/xingran-backend" ] || fail "xingran-backend not in tarball"
[ -f "$UPLOAD/xingran.service" ] || fail "xingran.service not in tarball"

sudo install -m 0755 "$UPLOAD/xingran-backend" "$NEW"
install -m 0644 "$UPLOAD/xingran.service" "$UNIT_NEW"

rm -f "$TARBALL" "$UPLOAD/xingran-backend" "$UPLOAD/xingran.service"

echo ">> files in place, activating version $VERSION"
# 6. exec 原 activate 流程（84 行零改动）
exec sudo -n bash "$APP/deploy/deploy-remote.sh" "$VERSION"
