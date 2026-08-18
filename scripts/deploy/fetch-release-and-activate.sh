#!/usr/bin/env bash
# fetch-release-and-activate.sh — server 端（通过 ssh 'bash -s' 触发）
#
# 从 GitHub Release 经 gh-proxy.com 镜像匿名拉取部署 tar.gz。
# 网络矩阵实测(2026-08-18): runner→server SSH ~10KB/s(不可用),
# server→GitHub 直连 0-1.6KB/s(不可用), server→gh-proxy 公开资产 2.7-9.7MB/s。
# 仓库已设为 public(用户决策),release 资产匿名可达,无 token / 无 rate limit。
#
# 流程：
#   1. df 磁盘预检
#   2. curl 匿名经 gh-proxy 下载 browser_download_url
#   3. sha256 校验（runner 端 build 时生成并传入）
#   4. 解压出 xingran-backend + xingran.service，就位 .new 文件
#   5. exec deploy-remote.sh（备份/原子替换/重启/health/回滚，零改动）
#
# 用法: ssh ... 'bash -s' -- <VERSION> <BROWSER_DOWNLOAD_URL> <SHA256> < fetch-release-and-activate.sh

set -euo pipefail

APP=/opt/xingran
UPLOAD=$APP/upload
NEW=$APP/xingran-backend.new
UNIT_NEW=$APP/deploy/xingran.service.new
PROXY="https://gh-proxy.com"

VERSION="${1:-}"
ASSET_URL="${2:-}"   # runner 侧已解析的 browser_download_url
EXPECTED_SHA="${3:-}"

fail() { echo "!! $*" >&2; exit 1; }

[ -n "$VERSION" ]    || fail "usage: $0 <version> <browser_download_url> <sha256>"
[ -n "$ASSET_URL" ]  || fail "missing asset url"
[ -n "$EXPECTED_SHA" ] || fail "missing sha256"

# 1. 磁盘预检（tar.gz ~22MB + 解压后 63MB + 备份 63MB）
FREE_KB=$(df -k "$APP" | tail -1 | awk '{print $4}')
[ "$FREE_KB" -ge 204800 ] || fail "insufficient disk: ${FREE_KB}KB free (need >= 200MB)"

sudo install -d -m 0755 "$UPLOAD"
TARBALL="$UPLOAD/xingran-backend.tar.gz"
rm -f "$TARBALL"

# 2. 匿名下载（公开资产,CDN 直达）
echo ">> downloading asset via gh-proxy: $ASSET_URL"
curl -fsSL --max-time 300 \
  -o "$TARBALL" \
  "$PROXY/$ASSET_URL" \
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
