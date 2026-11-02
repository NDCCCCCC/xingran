#!/usr/bin/env bash
# verify-pure-commit.sh —— 在**纯净 worktree** 中验证某个 commit 的目标测试。
#
# ★ 为什么需要这个（2026-11-02 团队沉淀）：
#   共享工作区里「你看到的绿」可能来自**别人或你自己未提交的改动**。
#   本轮真实案例：`go test` 本地全绿 ⇒ CI 却红 ⇒ 原因是修复漏了 `git add`，
#   commit 里跑的还是旧版用例。共享工作区的未提交改动（别人的并行编辑 +
#   我自己目录级 gofmt 的副作用）同样会污染本地测试结果。
#   ⇒ 凡是**要进 PR 的提交**，推送前用本脚本在纯净环境跑一遍目标测试。
#
# 用法：
#   scripts/devtools/verify-pure-commit.sh                      # 验证 HEAD，默认三包
#   scripts/devtools/verify-pure-commit.sh <commit>             # 验证指定 commit
#   scripts/devtools/verify-pure-commit.sh <commit> ./pkg/...   # 指定包
#
# 行为：
#   1. 用 `git worktree add --detach` 把目标 commit 检出到临时目录（**不碰共享工作区**）
#   2. 在该目录跑 `go test <包...>`
#   3. 无论成败都清理 worktree（trap EXIT）—— 不给共享仓库留垃圾
#
# 退出码：go test 的退出码（0 = 全绿）。
#
# 注意：
#   - 临时 worktree 会 checkout 整个仓库（数千文件，耗时约 1 分钟），属预期开销；
#   - 脚本内**不使用** `git worktree remove --force`（共享仓库安全网会拦），
#     正常路径下 worktree 是干净的，无需 --force；
#   - 本脚本不改动任何被验证的代码，只做「检出 + 测试 + 清理」。

set -euo pipefail

COMMIT="${1:-HEAD}"
if [ $# -gt 0 ]; then
  shift
fi

PKGS=("$@")
if [ ${#PKGS[@]} -eq 0 ]; then
  PKGS=(
    ./internal/services/portcollection/...
    ./internal/services/portwrite/...
    ./scripts/devtools/riguat/
  )
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
TMP_ROOT="$(mktemp -d)"
WORKTREE="$TMP_ROOT/verify-pure"

cleanup() {
  cd "$REPO_ROOT" >/dev/null 2>&1 || true
  git worktree remove "$WORKTREE" >/dev/null 2>&1 || true
  git worktree prune >/dev/null 2>&1 || true
  rm -rf "$TMP_ROOT" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "==> 纯净验证 commit=$COMMIT  包: ${PKGS[*]}"
echo "==> 检出到临时 worktree: $WORKTREE"

git worktree add --detach "$WORKTREE" "$COMMIT" >/dev/null

cd "$WORKTREE"
go test "${PKGS[@]}"
rc=$?

echo "==> 纯净环境测试结束（exit=$rc）；worktree 将被自动清理"
exit $rc
