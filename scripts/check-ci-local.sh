#!/usr/bin/env bash
# check-ci-local.sh — 在本地(WSL/Git Bash/Linux)逐条复刻 ci.yml 的判定步骤。
#
# 目的: push 前跑一遍本脚本,CI 上的 Lint/Test/Coverage gate/frontend checks
# 应当全绿 — 消除"本地绿 CI 红"的环境矩阵分歧(PR #4 曾四轮收敛: 9 staticcheck
# + 5 环境差异 + 1 map 序,见 memory local-vs-ci-test-divergence)。
#
# 与 ci.yml 的对应关系(改 CI 时同步改这里):
#   backend  Lint   -> golangci-lint run --timeout=5m ./...      (action pin v2.12.2)
#   backend  Test   -> go test -timeout 15m -count=1 -coverprofile -covermode=atomic
#   backend  Gate   -> bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold
#   PR-only  Diff   -> bash .github/scripts/check-diff-coverage.sh coverage.out origin/main 80
#   frontend        -> npm ci --legacy-peer-deps + format:check + lint + type-check + test + build
#
# 用法:
#   bash scripts/check-ci-local.sh            # backend 全段(lint+test+gate)
#   bash scripts/check-ci-local.sh --diff     # 追加 PR diff coverage gate(需 origin/main)
#   bash scripts/check-ci-local.sh frontend   # 仅 frontend 段
#   bash scripts/check-ci-local.sh all        # backend + frontend
#   bash scripts/check-ci-local.sh all --no-npm-ci   # frontend 复用现有 node_modules(快)
#
# 版本守卫: 与 CI 不一致时 FAIL(而非警告) — 版本漂移正是分歧之源。
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
ROOT="$(pwd)"
SCOPE="${1:-backend}"
shift || true
NO_NPM_CI=0
RUN_DIFF=0
for arg in "$@"; do
  case "$arg" in
    --diff) RUN_DIFF=1 ;;
    --no-npm-ci) NO_NPM_CI=1 ;;
    *) echo "unknown flag: $arg" >&2; exit 2 ;;
  esac
done

FAILED=0
step()  { printf '\n\033[1;36m== %s ==\033[0m\n' "$1"; }
pass()  { printf '\033[1;32mPASS\033[0m %s\n' "$1"; }
fail()  { printf '\033[1;31mFAIL\033[0m %s\n' "$1"; FAILED=1; }

# ---------- 版本守卫 ----------
step "版本守卫 (对齐 ci.yml pins)"

WANT_GO="$(awk '/^go /{print $2}' go.mod | cut -d. -f1-2)"  # 主.次两段;patch 任意
GOT_GO="$(go env GOVERSION | sed 's/^go//; s|[a-z].*||')"
if [ "${GOT_GO#"$WANT_GO"}" != "$GOT_GO" ]; then
  pass "go $GOT_GO (go.mod 要求 $WANT_GO.x)"
else
  fail "go $GOT_GO 与 go.mod $WANT_GO 不同系 — CI(setup-go go-version-file)会装 $WANT_GO.x。WSL: go install golang.org/dl/go${WANT_GO}.5@latest && go${WANT_GO}.5 download"
fi

if command -v golangci-lint >/dev/null 2>&1; then
  LINT_VER="$(golangci-lint version 2>/dev/null | grep -oE 'version [0-9.]+' | head -1 | cut -d' ' -f2)"
  if [ "$LINT_VER" = "2.12.2" ]; then
    pass "golangci-lint $LINT_VER (CI action pin 同版本)"
  else
    fail "golangci-lint $LINT_VER ≠ CI pin 2.12.2 — 规则集可能不同。安装: 见脚本头注释"
  fi
else
  fail "golangci-lint 未安装 — CI Lint 步骤无法本地复刻"
fi

# ---------- backend 段 ----------
if [ "$SCOPE" = "backend" ] || [ "$SCOPE" = "all" ]; then

  step "backend Lint (同 CI: --timeout=5m ./...)"
  if golangci-lint run --timeout=5m ./...; then pass "lint 0 issues"; else fail "lint 有问题"; fi

  step "backend Test (同 CI 命令 + 包列表)"
  rm -f coverage.out
  if go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic \
      ./internal/... ./pkg/... ./cmd/...; then
    pass "go test 全绿"
  else
    fail "go test 有失败 — 上面输出即 CI 将看到的失败"
  fi

  step "backend Coverage gate (4 层)"
  if bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold; then
    pass "coverage gate PASS"
  else
    fail "coverage gate FAIL"
  fi

  if [ "$RUN_DIFF" = "1" ]; then
    step "PR diff coverage gate (模拟 PR: 对 origin/main)"
    git fetch origin main --quiet 2>/dev/null || true
    if bash .github/scripts/check-diff-coverage.sh coverage.out origin/main 80; then
      pass "diff coverage PASS"
    else
      fail "diff coverage FAIL"
    fi
  fi
fi

# ---------- frontend 段 ----------
if [ "$SCOPE" = "frontend" ] || [ "$SCOPE" = "all" ]; then
  cd "$ROOT/xingran-react-frontend"

  if command -v node >/dev/null 2>&1; then
    NODE_MAJOR="$(node --version | cut -d. -f1 | tr -d v)"
    if [ "$NODE_MAJOR" = "24" ]; then pass "node $(node --version) (CI pin 24)"; else fail "node $(node --version) 主版本 ≠ 24"; fi
  else
    fail "node 未安装(CI pin 24)"
  fi

  if [ "$NO_NPM_CI" = "0" ]; then
    step "frontend npm ci --legacy-peer-deps (同 CI)"
    npm ci --legacy-peer-deps || { fail "npm ci"; exit 1; }
  else
    step "frontend npm ci 跳过(--no-npm-ci,复用 node_modules)"
  fi

  for s in format:check lint type-check test build; do
    step "frontend npm run $s"
    if npm run "$s" --silent; then pass "$s"; else fail "$s — 输出即 CI 将看到的失败"; fi
  done
  cd "$ROOT"
fi

# ---------- 汇总 ----------
step "汇总"
if [ "$FAILED" = "0" ]; then
  printf '\033[1;32m全部通过 — push 后 CI(同判定面)预期全绿\033[0m\n'
  exit 0
else
  printf '\033[1;31m存在失败 — 以上输出即 CI 将复现的失败,push 前先修\033[0m\n'
  exit 1
fi
