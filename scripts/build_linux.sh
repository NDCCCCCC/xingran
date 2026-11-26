#!/bin/bash
# 权威构建入口：前后端一体化 Linux 产物（与 .github/workflows/deploy.yml 同形态）。
#
# 行为：
#   1. 前端产物缺失时自动构建（SKIP_AUTO_BUMP 门控保证版本只递增一次）
#   2. **同步 dist 到 embed 目录**（go:embed 是相对包目录解析的，见下方说明）
#   3. go build -tags=embed 嵌入前端 + ldflags 注入版本号
#   4. 产物自检（embed sanity check，同 CI）：二进制必须含前端特征
#
# 环境变量：
#   SKIP_FRONTEND=1  跳过「dist 缺失自动构建」（dist 仍必须存在，否则报错退出）
#
# 前置依赖：go、node（版本号来自 scripts/build/bump-version.mjs；若 dist 缺失还需 npm）。
#   注意本脚本**新增了 node 依赖**（旧版只调 go build）；CI 走 deploy.yml 自己的版本
#   解析（git tag / commit SHA），不受影响。
#
# 产物：out/xingran-backend-linux（前端已嵌入，可直接替换服务器二进制）
set -e
OUT_DIR="out"
OUT_FILE="$OUT_DIR/xingran-backend-linux"
DIST="xingran-react-frontend/dist"
# ⚠ go:embed 的路径基准是**包目录**，不是仓库根：
#   internal/server/embed_frontend_prod.go 里写的是
#     //go:embed all:xingran-react-frontend/dist
#   实际解析为 internal/server/xingran-react-frontend/dist。
#   该目录被 .gitignore 忽略（是构建产物），所以干净 checkout 上它不存在 ——
#   不复制就直接 go build -tags=embed 会报 "pattern ...: no matching files found"。
#   （本机实测：移走该目录后 go build -tags=embed ./internal/server/ 立即失败。
#    build-linux.bat 一直有 xcopy 这一步，bash 版此前漏了。）
EMBED_DIR="internal/server/xingran-react-frontend"
mkdir -p "$OUT_DIR"

# Step 1: 前端产物检查（embed 的 go:embed 源）
if [ ! -f "$DIST/index.html" ]; then
    if [ "${SKIP_FRONTEND:-0}" = "1" ]; then
        echo "ERROR: 前端产物缺失 ($DIST/index.html)。先跑 'npm run build' 或去掉 SKIP_FRONTEND=1。" >&2
        exit 1
    fi
    echo "[1/3] 前端产物缺失，自动构建前端（SKIP_AUTO_BUMP=1，版本由本脚本末尾统一递增）..."
    (cd xingran-react-frontend && SKIP_AUTO_BUMP=1 npm run build)
fi

# Step 2: 同步 dist 到 embed 目录（每次都重建，避免上一次构建的残留 chunk 混进来）
echo "[2/3] 同步前端产物到 embed 目录 ($EMBED_DIR/dist)..."
rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR"
cp -R "$DIST" "$EMBED_DIR/dist"
if [ ! -f "$EMBED_DIR/dist/index.html" ]; then
    echo "ERROR: 复制后 $EMBED_DIR/dist/index.html 仍不存在，请检查 $DIST。" >&2
    exit 1
fi

# Step 3: 后端构建（embed 前端 + 版本注入）
echo "[3/3] Building Linux binary -> $OUT_FILE ..."
APP_VER="$(node scripts/build/bump-version.mjs --get)"
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
go build -tags=embed -ldflags="-s -w -X main.Version=v${APP_VER}" -o "$OUT_FILE" ./cmd/main.go

# Sanity check（与 CI deploy.yml:113 完全同款，别自创判据）：
#   go:embed 会把文件路径原样存进二进制，即使 -s -w 剥符号后 grep -a 仍能找到。
#   用 embed 的 FS 标记而不是某个 chunk 名（chunk hash 每次构建都变）。
if ! grep -aq 'xingran-react-frontend/dist/index.html' "$OUT_FILE"; then
    echo "ERROR: 产物未包含前端资源（embed FS marker 不存在），禁止部署！" >&2
    exit 1
fi
BIN_SIZE=$(wc -c < "$OUT_FILE")
if [ "$BIN_SIZE" -lt 40000000 ]; then
    echo "ERROR: 产物仅 ${BIN_SIZE} 字节（< 40MB），embed 很可能失效，禁止部署！" >&2
    exit 1
fi

echo "Build successful: $OUT_FILE (version v${APP_VER}, frontend embedded)"
node scripts/build/bump-version.mjs --bump
ls -lh "$OUT_FILE"
