# Phase 71: 治理基线 + CI gate - Pattern Map

**Mapped:** 2026-08-20
**Files analyzed:** 4 (3 NEW + 1 MODIFY)
**Analogs found:** 4 / 4

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `.coverage-threshold` (NEW) | config (single value) | none — read by `check-coverage.sh` | `.husky/pre-commit` style single-line bash config + Go `coverage.out` plain text format | role-match |
| `.github/scripts/check-coverage.sh` (NEW) | utility (bash + awk gate) | file-I/O → awk → exit 0/1 | `scripts/check-status-literals.sh` (Phase 69 ratchet guard) | exact (same role: shell ratchet) |
| `.planning/coverage-baseline.md` (NEW) | doc (Markdown tracking table) | none — read by humans | `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` frontmatter + tables | role-match |
| `.github/workflows/ci.yml` (MODIFY) | config (GitHub Actions) | CI pipeline steps | existing `.github/workflows/ci.yml` (Test step) + `.github/workflows/deploy.yml` (upload-artifact pattern) | exact (same file + sister workflow) |

## Pattern Assignments

### `.coverage-threshold` (config, single-value)

**Analog:** no exact analog in repo (no `.nvmrc` / `.node-version` / `.coverage*` exists); pattern inferred from Go coverage profile text format (`coverage.out` line `file:start.col,end.col num_stmts count`).

**Content (per RESEARCH §"Threshold File Format"):**
```
12.8
```

**Format constraints** (from RESEARCH.md lines 205-225):
- 纯数字, 单行, `\n` 终止 (`12.8\n`, **不要** 写 `12.8%`)
- 小数点精度 `12.8` 一位小数与 quick scan SUMMARY.md 对齐
- 不写注释 — bash `read threshold < .coverage-threshold` 直接读
- 不写 JSON / YAML — D-01 明确"零依赖"
- 仓库根放置 (与 `go.mod`, `.gitignore` 同级)

**Reader pattern** (will live in `check-coverage.sh`, see below):
```bash
THRESHOLD=$(cat .coverage-threshold)
# awk 内对比: if (total_pct + 0 < THRESHOLD + 0) exit 1
```

---

### `.github/scripts/check-coverage.sh` (utility, bash ratchet gate)

**Analog:** `D:\code\ClaudeCode\guoguo\scripts\check-status-literals.sh` (Phase 69 DICT-01 ratchet guard — same role: bash script + grep/awk counting + exit 1 on regression).

**Imports / shebang pattern** (lines 1, 30):
```bash
#!/usr/bin/env bash
# check-coverage.sh — [purpose docstring, Phase link, scope exclusions, usage]
set -euo pipefail
```

**ROOT + collect + show_hits + check structure** (lines 32-126 of check-status-literals.sh):
```bash
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# PATTERN — replace status-literal ERE with awk aggregator
# (awk reads coverage.out, outputs "count relpath" + final weighted avg)

collect() {
  local f n rel
  while IFS= read -r f; do
    n=$(...)   # delegate to inner counting function
    if [ "$n" -gt 0 ]; then
      rel="${f#"$ROOT"/}"
      echo "$n $rel"
    fi
  done < <(scan-command)
}

check() {
  local failed=0
  while read -r line; do
    [ -z "$line" ] && continue
    n="${line%% *}"
    rel="${line#* }"
    # ... compare against allowed baseline; failed=1 on regression
  done < <(collect)
  if [ "$failed" -ne 0 ]; then
    echo "...FAILED — ..."
    exit 1
  fi
  echo "...passed"
}

case "${1:-}" in
  "") check ;;
  *) echo "usage: $0 [--baseline]" >&2; exit 2 ;;
esac
```

**Inner awk aggregator** (from RESEARCH.md lines 97-117 — copy verbatim into `check-coverage.sh` heredoc):
```awk
NR > 1 {
    split($1, parts, ":")
    n = split(parts[1], seg, "/")
    pkg = ""; for(i=4; i<=n-1; i++) pkg = (pkg == "") ? seg[i] : pkg "/" seg[i]
    if (pkg ~ /^scripts\// || pkg ~ /^tests\/scripts\// || pkg ~ /^node_modules\//) next
    num_stmts = $2 + 0; hit_count = $3 + 0
    covered = (hit_count > 0) ? num_stmts : 0
    biz_stmts[pkg] += num_stmts; biz_covered[pkg] += covered
}
END {
    total_s = 0; total_c = 0
    for (k in biz_stmts) {
        s = biz_stmts[k]; c = biz_covered[k]
        pct = (s > 0) ? c * 100.0 / s : 0
        printf "%-50s %8d %8d %6.2f%%\n", k, s, c, pct
        total_s += s; total_c += c
    }
    printf "PACKAGE %8d %8d %6.2f%%\n", total_s, total_c, total_c * 100.0 / total_s
}
```

**Per-package output format** (matches existing `D:\code\ClaudeCode\guoguo\.planning\quick\260820-backend-test-coverage-scan\per-package-coverage.txt` exactly — column widths):
```
pkg/normalize                                                             45         44      97.8%
internal/config                                                          147        137      93.2%
...
PACKAGE                                                                 STMT    COVERED        PCT
-------                                                                 ----    -------        ---
```

**Threshold read + exit logic** (RESEARCH §"Threshold File Format" lines 222-225, transformed):
```bash
THRESHOLD=$(cat .coverage-threshold)
# after awk END prints "PACKAGE <stmts> <covered> <pct>%" line, capture pct:
#   if (pct + 0 < THRESHOLD + 0) { print "FAIL: ..." > "/dev/stderr"; exit 1 }
# (awk `+ 0` forces float comparison; `THRESHOLD + 0` mirrors awk side)
```

**Exit codes** (mirror check-status-literals.sh lines 118-125):
- `0` — weighted avg ≥ threshold
- `1` — weighted avg < threshold (ratchet fail) OR collect error
- `2` — usage error (bad CLI arg)

**Usage conventions to copy:**
- Docstring header explains purpose, Phase link, exclusions, CI hookup (check-status-literals.sh lines 1-30 is the model)
- `set -euo pipefail` strict mode (line 30)
- `ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"` to anchor paths from script location (line 32)
- `show_hits` style for failure context (lines 73-75)
- Final echo + exit 1 pattern (lines 109-115)

---

### `.github/workflows/ci.yml` (config, GitHub Actions — MODIFY)

**Analog:** itself (`D:\code\ClaudeCode\guoguo\.github\workflows\ci.yml` lines 50-57 — Test step to be extended) + sister workflow `D:\code\ClaudeCode\guoguo\.github\workflows\deploy.yml` (upload-artifact pattern lines 138-152).

**Existing Test step to extend** (ci.yml lines 50-57 — verbatim):
```yaml
      - name: Test
        # Explicit package list excludes the root tests/ dir (integration
        # tests that need a running service) and rpa-worker/ (separate Go
        # module). Most unit tests use glebarez sqlite (:memory:, pure Go).
        # If a package starts timing out, fall back to a whitelist:
        #   go test ./internal/config/... ./pkg/...
        # and re-admit packages one by one.
        run: go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...
```

**After (Phase 71 changes — from RESEARCH.md lines 159-189):**
```yaml
      - name: Test
        # Phase 71 GOV-01: -coverprofile + -covermode=atomic + -count=1
        run: |
          go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic \
            ./internal/... ./pkg/... ./cmd/...

      - name: Coverage HTML report
        if: always()
        run: go tool cover -html=coverage.out -o coverage.html

      - name: Coverage gate
        # GOV-02: bash + awk 检查加权平均 vs .coverage-threshold
        run: bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold

      - name: Upload coverage artifact
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: backend-coverage
          path: |
            coverage.out
            coverage.html
          retention-days: 30
```

**`if: always()` precedent** (deploy.yml uses for SSH setup step at line 198 area; existing ci.yml doesn't use it yet — Phase 71 introduces first usage).

**`actions/upload-artifact@v4` precedent** (deploy.yml lines 138-152 — uses `@v7`, but Phase 71 needs `@v4` per D-02 lock):
```yaml
      - name: Upload binary artifact
        uses: actions/upload-artifact@v7
        with:
          name: xingran-backend-linux-amd64
          path: out/xingran-backend
          retention-days: 3
          compression-level: 0   # already a stripped binary; gzip wastes CPU
```

**Comment style precedent** (ci.yml lines 4-9, deploy.yml lines 1-29 — every workflow file has a header explaining design rationale. Phase 71 step comments must follow same density.)

**DO NOT MODIFY** (RESEARCH.md lines 199-201):
- The existing concurrency group (`ci-${{ github.ref }}` with `cancel-in-progress: true`)
- The `permissions: contents: read` at job level
- The `golangci/golangci-lint-action@v9` step (Lint runs before Test, fail-fast preserved)

**Step order invariant** (RESEARCH.md lines 192-196):
- Order: Test → Coverage HTML → Coverage gate → Upload
- NO `set -e` on bash — gate fail must NOT prevent Upload artifact step (use `if: always()` on Upload)

---

### `.planning/coverage-baseline.md` (doc, Markdown tracking)

**Analog:** `D:\code\ClaudeCode\guoguo\.planning\quick\260820-backend-test-coverage-scan\SUMMARY.md` (frontmatter lines 1-9 + tables lines 13-66) — same domain (coverage tracking), same column types.

**Frontmatter pattern** (SUMMARY.md lines 1-9):
```markdown
---
type: quick
slug: backend-test-coverage-scan
created: 2026-08-20
completed: 2026-08-20
status: complete
description: 后端 Go 代码覆盖率扫描 + 缺失测试模块清单,纯只读不修改源代码
duration: ~14m (go test 跑 10m + 解析 4m)
---
```

**Header + provenance block** (mirrors SUMMARY.md lines 11-28 "TL;DR" pattern):
```markdown
# Coverage Baseline: v1.26 后端测试覆盖率优秀

**起点来源:** `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` (2026-08-20 纯只读扫描, 不可回填)
**测量口径:** 74 业务包加权平均, 排除 scripts/migrations/cmd main/internal/docs (43652 stmts 口径)
**生成方式:** `bash .github/scripts/check-coverage.sh coverage.out` (本地 + CI 同公式)
```

**Snapshot table pattern** (from RESEARCH.md lines 263-280 — column schema is locked):
```markdown
| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-20 | 起点 | 12.8 | 43652 | 5589 | 33 | 5ead742 | n/a | n/a | n/a |
```

**Per-package detail block** (from RESEARCH.md lines 269-273 — inline 76-line table inside fenced code block):
```markdown
### Per-package (起点)

\`\`\`
[per-package-coverage.txt 全文 76 行 inline — copy from
.planning/quick/260820-backend-test-coverage-scan/per-package-coverage.txt]
\`\`\`
```

**Regression checklist pattern** (from RESEARCH.md lines 287-291):
```markdown
### Per-package 倒退检查 (本行)

- [x] 无新增 0% 包 (起点 33 → Phase 71 后 33)
- [x] 无 per-package 倒退 (Phase 71 不改业务, 不改测试)
```

**Section divider pattern** (horizontal rule + `## 起点 (v1.26 启动前)` style — matches SUMMARY.md lines 14, 28, 53, 73, 116, 131, 151, 162, 205).

---

## Shared Patterns

### Coverage tooling chain (applies to ci.yml + check-coverage.sh)

**Source:** RESEARCH.md §"CI Workflow Changes" + §"Awk Aggregation Formula Validation"

```bash
# 1. produce coverage.out with atomic mode + count=1 (no cache)
go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic \
  ./internal/... ./pkg/... ./cmd/...

# 2. produce human-readable HTML
go tool cover -html=coverage.out -o coverage.html

# 3. gate with check-coverage.sh (reads .coverage-threshold)
bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold
```

### `if: always()` for debug-ability on failure

**Source:** RESEARCH.md lines 170, 181 (Coverage HTML step + Upload step both `if: always()`)
**Apply to:** Coverage HTML step + Upload coverage artifact step (in ci.yml).
**Rationale:** Gate fail must NOT prevent artifact upload — developer needs coverage.out to debug regression locally.

### Existing shell ratchet guard pattern (for check-coverage.sh structure)

**Source:** `D:\code\ClaudeCode\guoguo\scripts\check-status-literals.sh` lines 30-126
**Apply to:** `check-coverage.sh` — copy shebang + `set -euo pipefail` + ROOT anchor + collect/check/case structure. Replace grep-ERE pattern + allowed-baseline comparison with awk aggregator + threshold compare.

### No `.github/scripts/` directory currently exists

RESEARCH/CONTEXT call this out (path `.github/scripts/check-coverage.sh` is new). No prior pattern to mirror for directory creation. Convention inferred: `.github/workflows/` exists + chmod +x on .sh files (cf. `scripts/deploy/fetch-release-and-activate.sh` rwxr-xr-x).

---

## No Analog Found

| File | Role | Reason |
|------|------|--------|
| (none) | — | All 4 Phase 71 files have analogs above |

`.coverage-threshold` has no in-repo analog (single-value dotfile pattern not used in this codebase yet — pattern inferred from coverage.out plain-text format + bash `cat` simplicity). The other 3 files have direct analogs.

---

## Metadata

**Analog search scope:**
- `.github/workflows/*.yml` (3 files)
- `.github/scripts/` (does not exist yet — pattern inferred from `scripts/*.sh` conventions)
- `scripts/*.sh` + `scripts/deploy/*.sh` (8 files) — for bash ratchet + CI shell patterns
- `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` + `per-package-coverage.txt` — for awk output format + tracking tables
- `.planning/MILESTONES.md` + `.planning/ROADMAP.md` — for milestone tracking table conventions
- Repo root dotfiles (none for coverage-threshold analog found)

**Files scanned:** ~15 (workflows + shell scripts + planning docs)

**Pattern extraction date:** 2026-08-20