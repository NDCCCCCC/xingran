# Phase 82: 口径修正与治理基建 - Pattern Map

**Mapped:** 2026-08-23
**Files analyzed:** 7 (3 modify + 4 new)
**Analogs found:** 7 / 7 (全部有强 analog;唯一无仓库先例的子模式是 node 扁平化,见 No Analog Found)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `xingran-react-frontend/vitest.config.ts` (modify) | config | transform (coverage 口径定义) | 自身现状 coverage 块 (L16-37) | exact (原地改) |
| `xingran-react-frontend/package.json` (modify) | config | — | 自身 scripts 块 (L17-19) | exact (原地改) |
| `.coverage-fe-floors` (new) | config (阈值数据文件) | file-I/O (被 gate 脚本读取) | `.coverage-threshold` (单值 55.5) | role-match (单值 → 表扩展) |
| `.github/scripts/check-frontend-coverage.sh` (new) | utility (CI gate 脚本) | batch (聚合 + 阈值比较) | `.github/scripts/check-coverage.sh` | exact (同角色同流程,仅输入格式不同) |
| `.github/scripts/check-frontend-diff-coverage.sh` (new) | utility (CI gate 脚本) | transform (diff 行解析 + coverage join) | `.github/scripts/check-diff-coverage.sh` | exact |
| `.github/workflows/ci.yml` (modify) | config (CI 编排) | event-driven (PR/push 触发) | 同文件 backend job (L27-90) + coverage-diff job (L92-119) | exact (文件内 analog) |
| `.planning/frontend-coverage-baseline.md` (new) | doc (治理记录) | — | `.planning/coverage-baseline.md` | exact |

**范围外确认:** `.gitignore` L120 已含 `xingran-react-frontend/coverage/`,本 phase 无需改动;`.coverage-fe-floors` 必须入库(与 `.coverage-threshold` 同级、被 git 追踪),**不得**加进 .gitignore。

---

## Pattern Assignments

### `xingran-react-frontend/vitest.config.ts` (config, modify)

**Analog:** 自身现状 (44 行,原地修改 coverage 块)

**现状 coverage 块** (lines 16-37):
```typescript
    coverage: {
      provider: "v8",
      reporter: ["text", "json", "html"],
      exclude: [
        "node_modules/",
        "src/test/",
        "**/*.d.ts",
        "**/*.config.*",
        "**/mockData/**",
        "dist/",
      ],
      // Phase 63 基准阈值:... (历史注释)
      thresholds: {
        statements: 24,
        branches: 15,
        functions: 18,
        lines: 24,
      },
    },
```

**修改要点 (三处,其余一律不动):**

1. **加 `include`** (D-10/GOV-01 全量口径唯一开关,放在 `provider` 之后):
```typescript
      include: ["src/**/*.{ts,tsx}"],
```
2. **重写 `exclude`** (Pitfall 6:include 圈定后 root 级模式失效可删;`src/test/` 与 `**/*.d.ts` 必须保留;白名单两项新增):
```typescript
      exclude: [
        "src/test/",
        "**/*.d.ts",
        // QUAL-02 白名单 (D-10: 单一真相源在此, gate 脚本做漂移检测)
        "src/components/cad-editor/**",    // 804 stmts / 8 文件
        "src/components/cad-elements/**",  // 224 stmts / 5 文件
      ],
```
3. **整段删除 `thresholds` (L27-36 含其上的 Phase 63 注释)** — D-16:外部 bash gate 是唯一真相源;保留会让 `test:coverage` 在口径切换瞬间直接红 (3.85% < 24)。

**注释风格:** 沿用现有中文 phase 注释体例 (参考 L10-13 的 testTimeout 注释:phase 号 + 原因 + 影响),新 include/exclude 行配 D 编号注释。`provider`/`reporter`/`testTimeout`/`setupFiles`/`resolve.alias` 全部保留不动 (D-03 锁定三 reporter)。

---

### `xingran-react-frontend/package.json` (config, modify)

**Analog:** 自身 scripts 块 (lines 17-19):
```json
    "test": "vitest",
    "test:ui": "vitest --ui",
    "test:coverage": "vitest --coverage",
```

**修改:** 仅 L19 一行 — `"test:coverage": "vitest run --coverage"` (加 `run` 消除本地 watch 挂住,Pitfall 5;CI 里 CI=true 本就单跑,行为不变)。`"test"` 与 `"test:ui"` 保持原样 (本地 dev watch 用途)。

---

### `.coverage-fe-floors` (config/data, new)

**Analog:** `.coverage-threshold` — 全文内容即一行 `55.5`,由 gate 脚本读取:

`.github/scripts/check-coverage.sh` lines 63-75 (锚定路径 + cat 读取模式):
```bash
case "$THRESHOLD_FILE" in
  /*) THRESHOLD_PATH="$THRESHOLD_FILE" ;;
  *)  THRESHOLD_PATH="$ROOT/$THRESHOLD_FILE" ;;
esac

if [ ! -f "$THRESHOLD_PATH" ]; then
  echo "check-coverage.sh: threshold file $THRESHOLD_PATH not found" >&2
  exit 2
fi

THRESHOLD="$(cat "$THRESHOLD_PATH")"
```

**前端版格式** (D-07:全局行 + per-dir floor 表同文件;ratchet bump = 纯数据变更):
```bash
# 全局阈值行 (statements, 一位小数) — D-14
GLOBAL 3.8
# per-dir floor 表: 目录 <TAB> floor(%) — 目标 70.0, 过渡期 = D-06 ratchet 初值 (实测 −0.5pp)
pages/operations	0.0
pages/login	61.7
components	4.9
...
```

**关键差异 vs analog:** 后端是纯单值 (`cat` 即得);前端是结构化文件,gate 脚本解析时须跳过 `#` 注释行、区分 `GLOBAL` 行与目录行 (建议 `grep -v '^#'` + `awk '$1=="GLOBAL"'` / `awk '$1!="GLOBAL"'` 两类提取)。目录键格式与 D-05 粒度一致 (`src/` 一级目录名 + `pages/` 二级),RESEARCH.md 实测 per-dir 全表 (L338-368, 26 行) 是 `--init` 初值数据源;`(src root)` 与 `api` 两个显式条目必须入表 (Pitfall 8:消除 21 stmts 无主面积)。

---

### `.github/scripts/check-frontend-coverage.sh` (utility/gate, new)

**Analog:** `.github/scripts/check-coverage.sh` (309 行,四段式) — 结构照抄,四处已知差异:

| 段 | 后端行号 | 前端版处置 |
|----|---------|-----------|
| 头注释 (purpose/usage/exit codes/CI hookup/ratchet workflow) | L1-45 | 照抄体例,内容换前端 (Phase 82 / GOV-03/05 / D-07) |
| `set -euo pipefail` + ROOT 锚定 | L46-48 | 逐字照抄 |
| 参数检查 exit 2 + floors 文件锚定 | L50-73 | 照抄,THRESHOLD_FILE → FLOORS_FILE |
| profile 缺失软跳过 exit 0 | L77-85 | 逐字照抄 (Pitfall 9:测试失败时配合上游 if: always()) |
| **node 扁平化 (前端新增段)** | — (无 analog) | 插在聚合之前,见 Shared Patterns P-node |
| awk 聚合表 + 内嵌全局阈值判定 | L92-131 | 结构照抄,输入从 coverage.out 换成扁平化 TSV |
| 防御性 no-PASS → exit 1 | L141-154 | 逐字照抄 |
| per-dir floor (对称 P1/P1 段) | L158-216, L218-308 | 结构照抄,但 floor 表从 `.coverage-fe-floors` 读取 (D-07),**不复制**后端硬编码 P1_PACKAGES/P2_RATCHET 变量 |
| 白名单漂移检测 (前端新增) | — | `grep -Ei 'cad-editor\|cad-elements'` 命中扁平化输出 → exit 6 (D-10) |

**头注释 + exit code 块** (lines 12-25, 前端版改写模板):
```bash
# Usage:
#   bash .github/scripts/check-frontend-coverage.sh <coverage-final.json> <floors-file>
#
# Exit codes (mirror check-coverage.sh / check-status-literals.sh):
#   0 — 全局 statements >= GLOBAL 行 AND 所有 per-dir >= floor AND 无白名单漂移
#   1 — 全局阈值未达 OR 解析失败 (no PASS line)
#   2 — usage error (missing args / unreadable files)
#   4 — per-dir floor 违例 (GOV-05)
#   6 — 白名单漂移: cad-editor/cad-elements 出现在 coverage json (D-10)
#
# CI hookup (ci.yml frontend job):
#   - name: Coverage gate
#     working-directory: .        # 覆盖 job 级 working-directory (Pitfall 4)
#     run: bash .github/scripts/check-frontend-coverage.sh \
#             xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors
```

**软跳过模式** (lines 77-85, 逐字迁移):
```bash
if [ ! -f "$PROFILE" ]; then
  echo "check-coverage.sh: coverage profile $PROFILE missing — was Test step skipped?" >&2
  echo "check-coverage.sh: skipping gate (exit 0) so HTML/Upload steps can still run with if: always()" >&2
  exit 0
fi
```

**awk 聚合 + 数值比较核心** (lines 110-130 — 前端版换输入但保公式与输出格式):
```awk
END {
    total_s = 0
    total_c = 0
    for (k in biz_stmts) {
        s = biz_stmts[k]
        c = biz_covered[k]
        pct = (s > 0) ? c * 100.0 / s : 0
        printf "%-50s %8d %8d %6.2f%%\n", k, s, c, pct
        total_s += s
        total_c += c
    }
    pkg_pct = (total_s > 0) ? total_c * 100.0 / total_s : 0
    if (pkg_pct + 0 < threshold + 0) {
        printf "FAIL: weighted avg %.2f%% < threshold %.2f%%\n", pkg_pct, threshold + 0
        exit 1
    } else {
        printf "PASS: weighted avg %.2f%% >= threshold %.2f%%\n", pkg_pct, threshold + 0
        exit 0
    }
}
```
保留点:`a + 0 >= b + 0` 数值强制 (L123, 反字符串序陷阱)、`%-50s %8d %8d %6.2f%%` 输出格式 (L117, 本地/CI diff-friendly)、PASS:/FAIL: 行约定 (L124-128)。前端版 TSV 每行 `relpath\tstmts\tcovered\tranges`,per-dir 键 = `split($1, seg, "/")` 后按 D-05 粒度拼前缀 (`pages/` 取两段,其余取一段,根文件归 `(src root)`)。

**防御性收尾** (lines 149-154, 逐字迁移):
```bash
if ! echo "$AWK_TABLE" | grep -qE '^PASS:'; then
  echo "check-coverage.sh: profile parsed but no PASS line emitted — treating as failure" >&2
  exit 1
fi
```

**per-dir floor 循环模式** (lines 188-205 的 P1 循环是结构模板):
```bash
P1_FAILED=0
for pkg in $P1_PACKAGES; do
  line="$(printf '%s\n' "$P1_PKG_TABLE" | awk -v p="$pkg" '$1 == p { print $2, $3; exit }')"
  if [ -z "$line" ]; then
    echo "FAIL: P1 $pkg not found in profile — no statements measured for this package" >&2
    P1_FAILED=$((P1_FAILED + 1))
    continue
  fi
  ...
  if awk -v a="$pct" -v b="$P1_FLOOR" 'BEGIN { exit !(a + 0 >= b + 0) }'; then
    echo "PASS: P1 $pkg $pct% >= $P1_FLOOR% ($covered/$stmts stmts)"
  else
    echo "FAIL: P1 $pkg $pct% < $P1_FLOOR% ($covered/$stmts stmts)" >&2
    P1_FAILED=$((P1_FAILED + 1))
  fi
done
```
前端版:目录清单来自 `.coverage-fe-floors` (非硬编码);"not found in profile" 分支保留 (目录下所有文件 0 语句时 json 可能无条目);违例汇总后 `exit 4`。**不要复制** L227-254 的 P2_FLOOR/P2_RATCHET 硬编码 + `floor_of()` 间接层 — D-07 数据文件已取代它 (RESEARCH Anti-Patterns 明示"别走回头路"),但 L230-241 的 ratchet 方法论注释 (测量噪声来源:patch 版本漂移/环境分支差异/异步方差) 值得摘录进前端版头注释,解释 −0.5pp (D-06) 的来由。

**`--init` 模式 (D-08):** 后端无 analog。建议实现为 `--init` flag:跳过 gate 判定,直接从扁平化 TSV 聚合输出 `.coverage-fe-floors` 格式初值 (实测 % − 0.5,下限 0,GLOBAL 行取 min(实测, 3.8)),stdout 重定向即得数据文件。

---

### `.github/scripts/check-frontend-diff-coverage.sh` (utility/gate, new)

**Analog:** `.github/scripts/check-diff-coverage.sh` (205 行,三段式)

**结构映射:**

| 段 | 后端行号 | 前端版处置 |
|----|---------|-----------|
| 头注释 (74-10 第三方 action 否决记录 + 语义定义) | L1-42 | 照抄体例;supply-chain 决策链注释 (L4-14) 逐字引用 — 前端沿用同一结论 |
| `set -euo pipefail` + ROOT + 参数检查 | L43-58 | 逐字照抄 |
| profile/base-ref 存在性检查 | L62-70 | 逐字照抄 |
| go.mod MODULE 读取 | L72-76 | **删除** — 前端路径规范化在 node 扁平化内完成 (`xingran-react-frontend/src/` 截取) |
| mktemp + trap | L84-85 | 逐字照抄 |
| 第 1 段: diff hunk 解析 awk | L87-106 | **近乎逐字复刻**,仅换 pathspec + 前端注释语义 |
| 空 diff 软通过 | L111-114 | 逐字照抄 |
| 第 2 段: coverage join | L116-186 | 结构照抄;coverage.out 块解析换 statementMap 行区间 (经扁平化 TSV) |
| 第 3 段: gate + 防御性 no-PASS | L190-204 | 逐字照抄 |

**第 1 段 diff 解析** (lines 87-106 后端原文;前端版改 pathspec 与注释正则):
```bash
git diff --unified=0 "${BASE_REF}...HEAD" -- '*.go' ':(exclude)*_test.go' | awk '
  /^\+\+\+ b\// { file = substr($0, 7); in_hunk = 0; next }
  /^@@ / {
    match($0, /\+[0-9]+/)
    lineno = substr($0, RSTART + 1, RLENGTH - 1) + 0
    in_hunk = 1
    next
  }
  in_hunk && /^\+/ {
    line = substr($0, 2)
    if (line !~ /^[[:space:]]*$/ && line !~ /^[[:space:]]*\/\//) {
      printf "%s\t%d\n", file, lineno
    }
    lineno++
    next
  }
  in_hunk && /^-/ { next }    # removed line: not counted, lineno stays
  in_hunk && /^\\/ { next }   # "\ No newline at end of file"
' > "$CHANGED"
```
前端版两处替换 (RESEARCH 已实测验证):
- pathspec 换 `'xingran-react-frontend/src/*.ts' 'xingran-react-frontend/src/*.tsx' ':(exclude)xingran-react-frontend/src/*.test.*' ':(exclude)xingran-react-frontend/src/**/__tests__/**'` (git 实测:单星跨目录成立,310 命中;exclude 生效)
- 注释排除正则扩为 `line !~ /^[[:space:]]*\/\// && line !~ /^[[:space:]]*\/\*/ && line !~ /^[[:space:]]*\*/` (TS 的 `//`、`/*`、JSDoc 延续 `*` 三态)

**join 语义注释** (lines 119-126, 前端版同样要写成头注释 — 新文件不给免费通行):
```bash
#   - file HAS blocks in the profile: only changed lines inside SOME block
#     (covered or not) enter the denominator — package/import/declaration
#     lines are not coverable and must not penalize the gate. A line is
#     covered when it falls inside a block with count > 0.
#   - file ABSENT from the profile (package never exercised by tests):
#     all its changed executable lines count as measured + uncovered —
#     brand-new untested files must NOT get a free pass.
```
前端版:块区间 = 扁平化 TSV 第 4 列 (`start-end:hit` 逗号列表,来自 statementMap);行 L covered ⟺ ∃区间 hit=1 且 start ≤ L ≤ end。全量口径下所有 src 文件都在 json 中,"file ABSENT" 分支几乎不触发但防御性保留 (Pitfall 10)。改动行不落入任何区间 (纯类型/声明/注释) 不计分母。

**输出与 gate 判定** (lines 170-184, 格式逐字保留):
```awk
      if (hit) { covered++; per_file_cov[f]++ }
      else { printf "UNCOVERED %s:%d\n", f, ln }
    }
    for (f in per_file_total) {
      printf "FILE %-60s %6d/%6d lines %6.2f%%\n", f, c, t, (t > 0) ? c * 100.0 / t : 0
    }
    pct = (total > 0) ? covered * 100.0 / total : 100
    printf "DIFF %d %d %.2f\n", covered, total, pct
    if (pct + 0 < threshold + 0) {
      printf "FAIL: diff coverage %.2f%% < threshold %.2f%%\n", pct, threshold + 0
      exit 1
    }
    printf "PASS: diff coverage %.2f%% >= threshold %.2f%%\n", pct, threshold + 0
```

**Exit codes:** 0 过/无 diff / 1 阈值未达或解析失败 / 2 usage (镜像后端 L28-31,前端不加 4/6 — 漂移检测归 check-frontend-coverage.sh)。

---

### `.github/workflows/ci.yml` (config/CI, modify)

**Analog:** 同文件 backend job (L27-90) + coverage-diff job (L92-119)。**严禁触碰 backend 与 coverage-diff 两个 job** (两 workstream 共享编辑区,v1.27 并行改后端)。只改 frontend job (L121-171) 并在其后新增 frontend-coverage-diff job。

**backend job 的 gate 步骤序 (D-01 的对称模板)** — 步骤顺序不变式 (check-coverage.sh L22-23 头注释明示:Test → Coverage HTML → Coverage gate → Upload artifact):

backend Test 步骤 (lines 50-62):
```yaml
      - name: Test
        run: |
          go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic \
            ./internal/... ./pkg/... ./cmd/...
```

backend gate + upload 步骤 (lines 71-90):
```yaml
      - name: Coverage gate
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

**frontend job 现状** (lines 121-127 — working-directory 是 Pitfall 4 的根源):
```yaml
  frontend:
    runs-on: ubuntu-latest
    timeout-minutes: 15

    defaults:
      run:
        working-directory: xingran-react-frontend
```

frontend 现有 Test 步骤 (lines 165-167, 被替换):
```yaml
      - name: Test
        # vitest auto-runs once (no watch) when CI=true
        run: npm run test
```

**frontend job 目标态步骤序** (Test 替换 → gate 插入 → upload 插入 → Build 保留在最后):
```yaml
      - name: Test (coverage)
        run: npm run test:coverage     # package.json 同步改 vitest run --coverage

      - name: Coverage gate
        # 全局阈值 + per-dir floor + 白名单漂移 (exit 1/4/6 任一即红)
        working-directory: .           # 关键: 覆盖 job 级 working-directory (Pitfall 4)
        run: bash .github/scripts/check-frontend-coverage.sh \
                xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors

      - name: Upload coverage artifact
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: frontend-coverage       # 与 backend-coverage 命名对齐
          path: xingran-react-frontend/coverage/   # 相对 workspace root, 非 working-directory
          retention-days: 30

      - name: Build
        run: npm run build
```
注意:gate 步骤无 `if: always()` (对称 backend L71 — gate 只在 Test 成功产出 json 后跑;软跳过逻辑在脚本内);upload 步骤有 `if: always()` (对称 backend L83)。

**新增 frontend-coverage-diff job** (对称 coverage-diff job L92-119 的逐字段模板):
```yaml
  coverage-diff:                        # ← 后端原版 (只读参照, 不改)
    runs-on: ubuntu-latest
    timeout-minutes: 10
    needs: backend
    if: github.event_name == 'pull_request'
    steps:
      - uses: actions/checkout@v7
        with:
          # merge-base (three-dot) diff needs full history
          fetch-depth: 0

      - name: Download coverage profile
        # Reuses the backend job's coverage.out — no second test run.
        uses: actions/download-artifact@v4
        with:
          name: backend-coverage

      - name: Diff coverage gate (≥80%)
        run: bash .github/scripts/check-diff-coverage.sh coverage.out origin/${{ github.base_ref }} 80
```
前端版逐字段替换:`needs: frontend` / artifact 名 `frontend-coverage` / 脚本 `check-frontend-diff-coverage.sh xingran-react-frontend/coverage/coverage-final.json origin/${{ github.base_ref }} 80`。PR-only (`if: github.event_name == 'pull_request'`) 与 `fetch-depth: 0` 语义逐字保留。注意 diff gate 的 run 步骤无 working-directory 覆盖问题 (新 job 无 defaults,从 workspace root 跑);下载的 artifact 会还原到 workspace root 下的原相对路径 `xingran-react-frontend/coverage/`。

**其余保持:** `concurrency` (L19-21)、`permissions: contents: read` (L23-24)、setup-node 配置 (L134-142) 一律不动;frontend `timeout-minutes: 15` 保持 (D-04)。

---

### `.planning/frontend-coverage-baseline.md` (doc, new)

**Analog:** `.planning/coverage-baseline.md` (505 行)

**文档头模式** (lines 1-5):
```markdown
# Coverage Baseline: v1.26 后端测试覆盖率优秀

**起点来源:** `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` (2026-08-20 纯只读扫描, 不可回填)
**测量口径:** 74 业务包加权平均, 排除 scripts/migrations/cmd main/internal/docs (43652 stmts 口径)
**生成方式:** `bash .github/scripts/check-coverage.sh coverage.out` (本地 + CI 同公式)
```
前端版对应:起点 3.67% / 22602 stmts (白名单前) → 白名单后 3.85% / 21574 stmts;生成方式 `bash .github/scripts/check-frontend-coverage.sh ...` + gate 脚本 `--init`。

**ratchet 记录表 schema** (lines 11-13, 列名逐字复用):
```markdown
| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-20 | 起点 | 12.8 | 43652 | 5589 | 33 | 5ead742 | n/a | n/a | n/a |
```
前端版起点行:`weighted_avg` 3.85 (或 CI 首跑校准值) / `total_stmts` 21574 / `total_covered` 830 / `0pct_pkg_count` 换 0% 目录数 / `ratchet_from`→`ratchet_to` n/a (Phase 82 只建基线不 bump)。

**per-dir 快照模式** (lines 15-94):fenced code block 内逐行 `目录 %-50s stmts covered pct%`,数据直接取 RESEARCH.md 实测全表 (L338-368 的 26 行,含 `(src root)` 与 `api` 条目)。commit 列先写 `TBD (atomic ratchet)`,执行末尾回填短 SHA。

**倒退检查清单模式** (lines 185-189):
```markdown
### Per-package 倒退检查 (本行)

- [x] 无新增 0% 包 (起点 33 → Phase 71 后 33)
- [x] 无 per-package 倒退 (Phase 71 不改业务, 不改测试)
```

**Ratchet note blockquote 模式** (lines 298-302):
```markdown
> **Ratchet note (D-04):** The `commit` column on the Phase 72 后 row reads
> `TBD (atomic ratchet)` until plan 72-13 Task 4 amends this file with the actual
> short SHA of the commit that ships the .coverage-threshold + this file.
```
前端版:同一 commit 纪律作用于 `.coverage-fe-floors` + 本文件 (后端 D-04 的前端对称)。

**前端版新增段 (后端没有的):白名单登记** — D-09/D-11/D-12 要求三项齐全:
```markdown
## 白名单登记 (QUAL-02)

| 目录 | 排除理由 | 面积 (stmts/文件) | 占总语句 | 复审条件 |
|------|---------|------------------|---------|---------|
| src/components/cad-editor | <执行阶段填写> | 804 / 8 | 3.56% | 自身 statements ≥70% 可启动移除 (D-11); milestone 收口强制重审 |
| src/components/cad-elements | <执行阶段填写> | 224 / 5 | 0.99% | 同上 |
| **合计** | | **1028 / 13** | **4.55% ≤ 5%** | |

- D-12 锁死:仅此两项;新增须 milestone 级显式决策 + 重新核算总面积 ≤5%。
- D-10 同源:排除真相源是 vitest.config.ts coverage.exclude;gate 脚本漂移检测 (exit 6) 守护。
```

---

## Shared Patterns

### P-gate-skeleton: gate 脚本骨架 (两脚本共用)

**Source:** `.github/scripts/check-coverage.sh` L46-61 + L77-85 + L149-154;`.github/scripts/check-diff-coverage.sh` L43-58 + L190-204
**Apply to:** `check-frontend-coverage.sh`、`check-frontend-diff-coverage.sh`

```bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# 1. 参数检查 → exit 2
# 2. 输入文件存在性检查 → exit 2 (diff 版另有 base-ref git rev-parse 校验)
# 3. profile 缺失 → 软跳过 exit 0 (stderr 提示, 配合上游 if: always())
# 4. 判定主体 (awk)
# 5. 防御性收尾: 输出无 ^PASS: 行 → 按失败 exit 1 (拒绝静默通过)
```

### P-numeric: awk 数值比较强制

**Source:** `check-coverage.sh` L123 (`pkg_pct + 0 < threshold + 0`)、L199/L290 (`exit !(a + 0 >= b + 0)`)、`check-diff-coverage.sh` L179
**Apply to:** 两个前端 gate 脚本的全部阈值比较 — 百分比字符串序比较 ("10.0" < "9.9") 是已知陷阱,一律 `+ 0`。

### P-node: coverage-final.json 扁平化 (前端特有,唯一无仓库 analog 的子模式)

**Source:** RESEARCH.md Pattern 3 (L214-234, 本地实测验证过的骨架);后端不需要 (coverage.out 本就是行式文本)
**Apply to:** 两个前端 gate 脚本共用 — bash 包装内联 `node -e`,输出 TSV `relpath\tstmts\tcovered\tstart-end:hit,...`:

```javascript
const n = p.replace(/\\/g, "/");
const i = n.toLowerCase().indexOf("xingran-react-frontend/src/");
if (i < 0) continue;
const rel = n.slice(i);
```
路径规范化三要件 (Pitfall 3):反斜杠→正斜杠、toLowerCase 匹配锚点、`xingran-react-frontend/src/` 截取 — Windows 大写盘符与 Linux 绝对路径双兼容。statementMap 的 `end.column` 可为 null,只用 line 不受影响。

### P-artifact: CI artifact 复用 (零重复测试)

**Source:** `ci.yml` L92-119 (coverage-diff job)
**Apply to:** 新增 frontend-coverage-diff job — 四件套逐字对称:`needs: frontend` + `if: github.event_name == 'pull_request'` + `fetch-depth: 0` (三点 merge-base diff 需全历史) + `download-artifact@v4`。

### P-ratchet: ratchet 纪律 (数据 bump + 文档追加同 commit)

**Source:** `check-coverage.sh` L27-31 头注释 (Ratchet workflow D-04 manual) + L230-241 (保守下界方法论注释) + `.planning/coverage-baseline.md` L298-302 ratchet note
**Apply to:** `.coverage-fe-floors` 的每次 bump 必须与 `.planning/frontend-coverage-baseline.md` 追加在同一 commit;floor 只升不降;初值 = 实测 −0.5pp (D-06),禁止照抄实测值。

### P-softskip: 失败时序语义

**Source:** `ci.yml` L68/L83 (`if: always()` 只在 HTML/Upload 步骤) + `check-coverage.sh` L77-85
**Apply to:** frontend job — Test 红 → gate 不跑 (非 always) → upload 跑但可能拿不到 json (Pitfall 9, 可接受退化);gate 步骤本身脚本内软跳过,不靠 `continue-on-error`。

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| (子模式) node 扁平化段 | utility 内嵌步骤 | transform (JSON→TSV) | 后端 coverage.out 为行式文本,仓库无 JSON 解析先例;按 RESEARCH.md Pattern 3 (已本地实测验证) 实现,见 Shared Patterns P-node |
| (子模式) `--init` ratchet 生成 | utility 子命令 | batch (聚合→生成数据文件) | 后端 `.coverage-threshold` 为手写单值,无生成器;实现为 gate 脚本的 flag 分支,复用同一聚合 awk |

无整文件级缺 analog 项 — 7/7 文件均有 exact 或 role-match analog。

## Metadata

**Analog search scope:** `.github/scripts/` (2 文件全读)、`.github/workflows/ci.yml` (171 行全读)、`.planning/coverage-baseline.md` (505 行全读)、`.coverage-threshold`、`xingran-react-frontend/vitest.config.ts` (44 行全读)、`xingran-react-frontend/package.json` (scripts 段)、`.gitignore` (coverage 相关行)
**Files scanned:** 8
**Pattern extraction date:** 2026-08-23
**约束提醒 (给 planner):** ci.yml 的 backend/coverage-diff job 是只读参照物,严禁修改;所有行号以当前 main (1f3a8f0) 为准。
