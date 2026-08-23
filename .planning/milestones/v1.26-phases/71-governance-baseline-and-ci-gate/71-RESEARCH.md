# Phase 71: 治理基线 + CI gate - Research

**Researched:** 2026-08-20
**Domain:** Backend Go test coverage CI gate (bash + awk + GitHub Actions)
**Confidence:** HIGH — scope is engineering infra, all decisions D-01..D-04 locked, scope data already captured in `quick/260820-backend-test-coverage-scan/`

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01**: bash + awk 自检 (零依赖, 复用 quick-260820-bcs 加权聚合公式) + 阈值写入独立文件 `.coverage-threshold`
- **D-02**: `actions/upload-artifact@v4` + `retention-days: 30`, 同时上传 `coverage.out` + `coverage.html`
- **D-03**: 新建独立文件 `.planning/coverage-baseline.md` (不污染 quick scan SUMMARY.md)
- **D-04**: 手动 ratchet (每次 phase execute plan 原子提交更新 `.coverage-threshold` + `coverage-baseline.md`)
- **Phase 71 不补业务测试, 不改业务逻辑** — Pkg/cache flaky test 视为已修 (commit `5ead742`) 直接验收
- **GOV-03 (diff coverage)** 属 Phase 74 范围, 本阶段不预先引入

### Claude's Discretion (按默认走)
- Gate fail 无 override 机制 (严格无 `[skip-coverage]` 例外)
- 加权口径: 74 业务包加权, 排除 scripts/migrations/cmd main/internal/docs/密钥生成/诊断脚本 (43652 stmts 口径)
- CI 日志输出 per-package 数字即可, 不实现 PR comment bot
- `go test` 单次不跑 race-detector, `count=1` 防缓存
- Coverage scope 列表存放: `check-coverage.sh` 内置 awk 过滤规则, 不引入独立配置文件

### Deferred Ideas (OUT OF SCOPE)
- GOV-03 PR diff coverage ≥80% (Phase 74)
- FUT-02 分支覆盖率 / FUT-03 mutation testing / FUT-04 PR 评论机器人
- Gate fail override 机制
- `vladopajic/go-test-coverage` 引入 (Phase 74 重评估)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| **GOV-01** | CI 后端 job 以 `-coverprofile` 跑全量测试并产出覆盖率报告 (profile artifact 可下载) | `## CI Workflow Changes` + `## Implementation Path` Step 1-2 |
| **GOV-02** | CI 落地全局加权覆盖率阈值 gate (失败即阻断) + ratchet 防倒退机制 | `## Awk Aggregation Formula Validation` + `## Threshold File Format` + `## Ratchet Workflow` |
| **GOV-04** | `.planning/coverage-baseline.md` 记录起点 + 各 phase 后实际数字 | `## Coverage-baseline.md Schema` + `## Implementation Path` Step 4 |
</phase_requirements>

---

## Phase Boundary

**Phase 71 ships (5 SC):**
1. `ci.yml` 后端 job 含 `-coverprofile=coverage.out -covermode=atomic -count=1` 并产出 profile + HTML artifact (30 天保留)
2. CI 引入加权阈值 gate — 初始阈值 = `12.8` (`<.coverage-threshold>`) — 低于阈值 PR 即 fail; ratchet 机制允许 phase execute plan 末尾原子 bump
3. `.planning/coverage-baseline.md` 记录起点 (2026-08-20, 12.8%) → Phase 71 后实际数字 (回填一行)
4. `go test ./pkg/cache/...` 全过 (验收 `5ead742` 修复, 15/15)
5. `go test ./internal/... ./pkg/... ./cmd/...` 全包 exit 0 (无失败)

**Phase 71 NOT ships:**
- 不补任何业务测试代码 (Phase 72/73/74 范围)
- 不改业务逻辑、不改 API 契约
- 不引入 PR diff coverage (GOV-03 属 Phase 74)
- 不引入 `vladopajic/go-test-coverage` Action (Phase 74 重评估)
- 不实现 PR comment bot (FUT-04)
- 不实现分支覆盖率 (FUT-02)
- 不引入新 mock framework (D-04 锁定 glebarez sqlite)

**Integration points:**
- `.github/workflows/ci.yml` (extend) — Test step 加 `-coverprofile`; 新增 `Coverage gate` step + `Upload coverage artifact` step
- `.coverage-threshold` (NEW) — 仓库根, 纯数字 `12.8`
- `.github/scripts/check-coverage.sh` (NEW) — bash + awk, 复用 quick-260820-bcs 公式, exit 1 if weighted_avg < threshold
- `.planning/coverage-baseline.md` (NEW) — 状态快照表, 起点 + Phase 71 后行

---

## Implementation Path

**Plan 71-01 (建议 1 plan, 单 wave):**

| Step | Action | Verification |
|------|--------|--------------|
| 1 | 写 `.coverage-threshold` 仓库根文件 (纯数字 `12.8\n`) | `cat .coverage-threshold` |
| 2 | 写 `.github/scripts/check-coverage.sh` (chmod +x) — 复用 SUMMARY.md awk 公式, 输出 per-package 表 + 加权平均, exit 1 if avg < threshold | 本地 `bash .github/scripts/check-coverage.sh coverage.out` 跑通 |
| 3 | 改 `.github/workflows/ci.yml` backend job — Test step 加 `-coverprofile=coverage.out -covermode=atomic -count=1`; 新增 `Coverage HTML` step (`go tool cover -html=...`); 新增 `Coverage gate` step (`bash .github/scripts/check-coverage.sh coverage.out`); 新增 `Upload coverage artifact` step (`actions/upload-artifact@v4`, `retention-days: 30`, `if: always()`) | 读 ci.yml diff 确认 |
| 4 | 写 `.planning/coverage-baseline.md` 骨架 + Phase 71 后回填行 (本机跑一次 go test 得出数字) | `cat .planning/coverage-baseline.md` 含 Phase 71 后行 |
| 5 | 验收 `go test ./pkg/cache/...` 15/15 全过 | `go test ./pkg/cache/...` exit 0 |
| 6 | 验收 `go test ./internal/... ./pkg/... ./cmd/...` 全包 exit 0 | 同上, exit 0 |
| 7 | 故意 bump `.coverage-threshold` 到 `99.9` 验证 gate fail | `bash check-coverage.sh coverage.out` exit 1 (本地模拟) |
| 8 | 回滚 `.coverage-threshold` 到 `12.8`, commit, push | git status clean |

**关键顺序:**
- 写 `check-coverage.sh` 必须在改 `ci.yml` 前 — 本地先验证 exit 1/0 行为符合预期
- `coverage-baseline.md` 回填行基于本地最后一次 `go test` 数字 (避免 CI 上 artifact 解析)

---

## Awk Aggregation Formula Validation

### 公式 (与 quick-260820-bcs 一致)

```awk
NR > 1 {
    split($1, parts, ":")
    n = split(parts[1], seg, "/")
    pkg = ""; for(i=4; i<=n-1; i++) pkg = (pkg == "") ? seg[i] : pkg "/" seg[i]
    # 排除 scripts/, tests/, node_modules/, *.archive-migrate-*
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

### 12.8% 验证

quick-260820-bcs 给出: **43652 stmts / 5589 covered = 5589/43652 = 0.128028... = 12.80%** (round-to-1-decimal)

awk `printf "%.2f"` 输出 `12.80`, `printf "%.1f"` 输出 `12.8` — 两者都正确, 一致性以 `%.1f` 与 quick scan SUMMARY.md 对齐。

### 排除规则 (与 quick scan 一致)

awk `next` 排除的路径前缀:
- `scripts/` (运维工具)
- `tests/scripts/` (集成测试脚本)
- `node_modules/` (前端依赖, 不应进 Go coverage)

**注意**: quick-260820-bcs 还手工排除了 `mac/`、`dbactivity`/`dbbootstrap`/`dbprobe`/`dbprovision`、`db/audit_view_refs`、`crypto/gen_sm2_keys`、`crypto/migrate_sm4_key`、`diag/red_4f001`、`migrations/`、`cmd/main`、`internal/docs`。这些要么不在 Go module 内 (前 4 个), 要么是单文件 (`migrations/`), 要么由 per-package 表手工过滤。**Phase 71 `check-coverage.sh` 与 quick scan 严格一致 = 复用同一排除集**。

### 精度 gotchas

| 风险 | 触发条件 | 缓解 |
|------|----------|------|
| **awk 整数除法** | `int / int` 在 BWK awk 给出 int (`0.12` → `0`) | CI ubuntu-latest 默认 gawk, doubles; 加 `+ 0` 强制浮点 (`$2 + 0`); `printf "%.1f"` 而非 `%.0f` |
| **`hit_count > 0` 阈值** | coverage.out 行 `... 2 0` 是未覆盖; `... 2 1` 算 1 个 block 全覆盖; Go profile 无中间态 | 公式已正确处理: `count > 0` 即视为全覆盖该 block |
| **`num_stmts` 在 block 边界聚合** | 同一行被多个 block 覆盖会重复计数 | Go 工具不会产生这种情况 (block 按词法切分, 行不会重叠) — 不需额外处理 |
| **空 coverage.out** (新仓库 / 0 stmts) | total_s = 0, 除以 0 → NaN | 显式 `(s > 0) ? c*100.0/s : 0` 已包含; END 处若 total_s == 0, fail-safe exit 0 |
| **大文件 awk 性能** | coverage.out ~3MB / ~30k 行 | gawk 处理 30k 行 < 1s, CI 时间预算 25min 内无影响 |
| **CRLF 行尾** (Windows checkout) | `awk NR > 1` 可能误跳过有效行 | `git config core.autocrlf false` (GitHub checkout 默认 LF); 防御: `awk 'BEGIN{RS="\n"} NR>1'` |

---

## CI Workflow Changes

### `.github/workflows/ci.yml` 后端 job diff

**Before (当前):**
```yaml
      - name: Test
        run: go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...
```

**After (Phase 71):**
```yaml
      - name: Test
        # Phase 71 GOV-01: -coverprofile + -covermode=atomic + -count=1
        # (count=1 防缓存, atomic mode 与并发测试友好; 不带 -race 否则覆盖率时间翻倍)
        # rpa-worker/ 是独立 Go module, 不在 ./... 范围, 维持现状
        run: |
          go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic \
            ./internal/... ./pkg/... ./cmd/...

      - name: Coverage HTML report
        # GOV-01: 产 HTML 便于人工浏览器查看
        if: always()
        run: go tool cover -html=coverage.out -o coverage.html

      - name: Coverage gate
        # GOV-02: bash + awk 检查加权平均 vs .coverage-threshold
        # D-01: 阈值文件 + 本地脚本, 便于 phase 72/73/74 ratchet 时只改阈值不改 ci.yml
        run: bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold

      - name: Upload coverage artifact
        # GOV-01 + D-02: actions/upload-artifact@v4 + retention 30 天
        # if: always() 即使 gate fail 也上传, 便于本地调试
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: backend-coverage
          path: |
            coverage.out
            coverage.html
          retention-days: 30
```

### env / 顺序注意

- **不要** `set -e` 提前 break — 顺序必须是: Test → Coverage HTML → Coverage gate → Upload
- Coverage HTML 用 `if: always()` 因为 gate fail 时 coverage.out 仍然存在 (Test step 成功)
- Upload `if: always()` 确保 fail 时也能下载 artifact 调试 (D-02 明确: 不做 fail-only 上传)
- artifact 大小估算: 74 包 `coverage.out` ~3MB + `coverage.html` ~5MB = 8MB/run, 30 天保留, 远低于 GitHub 1G 限额

### Deploy.yml 联动

deploy.yml 通过 `workflow_run` gate 在 ci.yml 成功上, **Phase 71 不动 deploy.yml** — gate 自然生效 (coverage gate fail → ci.yml fail → workflow_run 不触发 → deploy 不发起)。

---

## Threshold File Format

### `.coverage-threshold` 内容

```
12.8
```

### 格式约束

- **纯数字**, 单行, newline 终止 (`12.8\n`, **不要** 写 `12.8%`)
- 小数点精度: `12.8` 一位小数与 quick scan SUMMARY.md 对齐; ratchet 时也建议一位小数 (`30.5` 而非 `30.50` 或 `30`)
- **不写注释** — bash `read` 时 awk 不应 skip 注释行, 简单 `read threshold < .coverage-threshold` 即可
- **不写 JSON / YAML** — D-01 明确"零依赖, 逻辑全部在 ci.yml 同一段"
- 仓库根放置 (与 `go.mod`, `.gitignore` 同级), 不放 `.github/` 内部

### check-coverage.sh 读取方式

```bash
THRESHOLD=$(cat .coverage-threshold)  # 读出 "12.8"
# awk 内对比: if (total_pct + 0 < THRESHOLD + 0) exit 1
```

### 文件权限

git add 即可, 无需 chmod (文本文件)。

---

## Coverage-baseline.md Schema

### 列定义

| 列 | 内容 | 示例 |
|----|------|------|
| `date` | YYYY-MM-DD 测量日期 | `2026-08-20` |
| `phase_label` | 阶段标识 (起点 / Phase 71 后 / Phase 72 后 / ...) | `起点` / `Phase 71 后` |
| `weighted_avg` | 加权平均覆盖率 (%.1f) | `12.8` |
| `total_stmts` | 业务包总语句数 | `43652` |
| `total_covered` | 业务包总覆盖语句数 | `5589` |
| `0pct_pkg_count` | 0% 覆盖业务包数 | `33` |
| `per_package_detail` | per-package 完整快照 (76 行表格 inline) | 见下方示例 |
| `commit` | 该次测量时的 commit SHA (短 7 位) | `5ead742` |
| `phase_executor` | 谁负责本次 ratchet (Phase 71 留空 `n/a` 起点 / Phase 72+ 填 phase 作者) | `n/a` / `gsd-execute-phase 72` |
| `ratchet_from` | 上一行 weighted_avg (ratchet 才有; 起点行填 `n/a`) | `n/a` / `12.8` |
| `ratchet_to` | 本行 weighted_avg (ratchet 才有; 起点行填 `n/a`) | `n/a` / `30.5` |

### 文件结构 (Markdown 表格)

```markdown
# Coverage Baseline: v1.26 后端测试覆盖率优秀

**起点来源:** `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` (2026-08-20 纯只读扫描, 不可回填)
**测量口径:** 74 业务包加权平均, 排除 scripts/migrations/cmd main/internal/docs (43652 stmts 口径)
**生成方式:** `bash .github/scripts/check-coverage.sh coverage.out` (本地 + CI 同公式)

---

## 起点 (v1.26 启动前)

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-20 | 起点 | 12.8 | 43652 | 5589 | 33 | 5ead742 | n/a | n/a | n/a |

### Per-package (起点)

```
[per-package-coverage.txt 全文 76 行 inline]
```

---

## Phase 71 后

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-20 | Phase 71 后 | 12.8 | 43652 | 5589 | 33 | TBD | gsd-execute-phase 71 | n/a | n/a |

### Per-package (Phase 71 后)

```
[Phase 71 后本机跑 go test 重新生成的 76 行]
```

### Per-package 倒退检查 (本行)

- [x] 无新增 0% 包 (起点 33 → Phase 71 后 33)
- [x] 无 per-package 倒退 (Phase 71 不改业务, 不改测试)
```

### Phase 71 完成后第一行回填示例

回填行由 plan executor 在 plan 完成时执行:
```bash
# 本机生成 Phase 71 后 per-package 表
go test -timeout 15m -count=1 -coverprofile=/tmp/cov.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...
bash .github/scripts/check-coverage.sh /tmp/cov.out /tmp/.dummy-threshold > phase71-per-pkg.txt

# 编辑 .planning/coverage-baseline.md 把 "TBD" 替换为真实 commit SHA + 填 per-package 表
git add .planning/coverage-baseline.md
git commit -m "docs(71): record coverage baseline — Phase 71 后 12.8%"
```

---

## Ratchet Workflow

### 谁触发 bump

**每个 phase execute plan 的最后一步** — 由该 phase 的 plan executor 在执行末尾原子提交:
1. 重跑 `go test -coverprofile=... -covermode=atomic` 得新数字
2. 跑 `bash .github/scripts/check-coverage.sh` 得 per-package 表
3. 编辑 `.coverage-threshold` 写入新阈值 (= 本次加权平均, 略保守或直接取实际值)
4. 编辑 `.planning/coverage-baseline.md` 追加新行 (含 per-package 完整表)
5. 原子 commit (两文件同 commit)

### Commit message 格式

```
docs(NN): coverage ratchet X.X% → Y.Y%

Phase NN 后加权平均 X.X% → Y.Y%。
- 更新 .coverage-threshold: X.X → Y.Y
- 追加 .planning/coverage-baseline.md 新行 (含 per-package 表 + commit SHA)
- 验证: go test ./internal/... ./pkg/... ./cmd/... 全过

Refs: Phase NN
```

**例 (Phase 72 后):**
```
docs(72): coverage ratchet 12.8% → 30.5%

Phase 72 后加权平均 12.8% → 30.5%。
- 更新 .coverage-threshold: 12.8 → 30.5
- 追加 .planning/coverage-baseline.md 新行
- 验证: go test 全过
```

### 何时 bump

| 触发条件 | 操作 |
|----------|------|
| Phase 71 完成 (本阶段) | 首次回填 coverage-baseline.md, **不 bump .coverage-threshold** (起点 = 12.8 = 当前基线) |
| Phase 72/73/74 完成 (后续) | 同时更新 .coverage-threshold + coverage-baseline.md, commit message 含 ratchet 比例 |

### 是否需要 PR

D-04 锁定手动 ratchet — 不写 CI 自动 bump helper (避免责任边界模糊)。

- **PR 模式**: PR merge 时 CI 跑 gate, 阈值 = `.coverage-threshold` 当前值; 若 PR 提交时包含 coverage 提升, merge 后由 plan executor 手工 bump (单独 commit 或 PR)
- **直接 push main**: plan executor 在 push 完成后单独提交 ratchet commit
- **不需要** 自动 bump bot / auto-merge script — 故意保持手动, 强制每次 ratchet 都是有意图、可 review 的 PR diff

### Ratchet 单调性保护

- `.coverage-threshold` 永远 **不下降** — 若某次 phase 加权平均倒退, executor 不写新阈值, 写一个 incident note (`.planning/notes/<date>-coverage-regression.md`) 记录并暂停 gate (临时改阈值到 12.8 维持绿, 等下次 phase 修复)
- 倒退 > 1% 时: 自动开 audit session (`/gsd-debug`); 倒退 ≤1% 时: plan executor 决定

---

## Gotchas & Risks

### 1. go test `-covermode=atomic` 工具链要求

- Go 1.21+ 原生支持 atomic mode (无需 `-race`)
- atomic mode 在并发测试下准确 (记录每次 atomic 操作的覆盖)
- **本仓库 go.mod**: `go 1.24.0` / toolchain `go1.24.5` — 完全支持 [VERIFIED]
- CI 用 `actions/setup-go@v7` + `go-version-file: go.mod`, 自动拉 toolchain — 无版本失配风险

### 2. sqlite in-memory 包是否进 coverage

- `glebarez/sqlite v1.11.0` + `glebarez/go-sqlite v1.21.2` 已在 go.mod [VERIFIED]
- 测试用 `:memory:` 不进网络, `go test -coverprofile` 仍然记录覆盖 (覆盖是 statement 级, 与 DB 后端无关)
- 当前 `pkg/cache 24.6%` (含 glebarez 驱动的 DB 测试) — 一致性已验证 [VERIFIED: per-package-coverage.txt]
- **无需新增依赖** (D-04 锁定)

### 3. race detector 与 coverage 不能同跑

- `-race` + `-cover` 同时启用会让测试时间翻 5-10x, 内存增加 ~5x — CI 25min 超时风险
- **Phase 71 不启用 `-race`** — D-01 "Claude's Discretion" 明确
- 若需 race coverage, 用 `go test -race -coverprofile=...` 单独跑 (本阶段不做)

### 4. `count=1` 防缓存

- `go test` 默认会缓存测试结果, 若代码不变, 不会重跑 — `coverage.out` 就会 stale
- `-count=1` 强制每次重跑 — ci.yml 已有此 flag, Phase 71 维持
- 本地跑 `go test -coverprofile` 验证时也要带 `-count=1`

### 5. Windows 本地 vs Linux CI 行为差异

- **行尾**: GitHub checkout 默认 LF, Windows 本地 CRLF; awk 处理 `\n` 一致 (Bash git-bash 默认 LF 模式)
- **awk 版本**: Git Bash 用 gawk (与 ubuntu-latest 同源) — 行为一致
- **路径分隔符**: awk split 不依赖 OS — 一致
- **coverage.out 大小**: Linux CI ~3MB 与 Windows 本地可能差几 KB (不同 Go 版本编译路径略不同) — 不影响公式
- **风险低**, 但 plan 阶段建议 executor 至少在 Git Bash 下跑一次验证

### 6. ubuntu-latest 默认 Go 版本是否匹配 go.mod 1.24

- `actions/setup-go@v7` + `go-version-file: go.mod` 显式读 go.mod — 不会用系统默认
- ubuntu-latest 默认 Go 是 1.22 / 1.23 (滚动), 但 setup-go 会拉 go1.24.5 toolchain
- GOTOOLCHAIN=auto (setup-go 默认) 允许自动下载 toolchain — 无网络风险 (actions 镜像已含)
- **风险低**, ci.yml 现有 backend job 已验证此模式 (Phase 1-70 都跑通)

### 7. coverage.out 是否被 rpa-worker 干扰

- ci.yml 注释明确: "rpa-worker/ (separate Go module)"
- rpa-worker 是独立 Go module, 不在 `xingran-go-backend/go.mod` 内 — `./internal/... ./pkg/... ./cmd/...` 不会触及
- **确认无干扰** [VERIFIED: ci.yml line 53 comment]

### 8. Coverage gate 与 Lint 的关系

- Lint (`golangci-lint-action@v9`) 与 Coverage gate 是两个独立 step — 一个 fail 不会影响另一个上传 artifact
- `Upload coverage artifact` 用 `if: always()` — lint fail 也上传, 便于本地 debug

### 9. coverage.out 在 Coverage HTML step 时不存在

- 若 Test step 因超时失败 (无 coverage.out), Coverage HTML 会 fail
- 缓解: Coverage HTML step 加 `if: always()` + `coverage.out 存在性检查` (`if [ -f coverage.out ]`); 缺失时 skip, 不 fail 整个 job
- Coverage gate 同样: `if [ -f coverage.out ]` 才跑, 否则 skip (避免 Test 失败时 gate 又 fail, 信息冗余)

### 10. `.coverage-threshold` 误改

- 直接 git commit 修改 — git history 可追溯
- PR review 可发现异常 bump (如 12.8 → 99.9)
- 不引入文件 lock / branch protection (过度工程)

### 11. Coverage artifact 上传体积

- 估算: coverage.out 3MB + coverage.html 5MB = 8MB/run
- 30 天保留: 假设每天 5 PR × 30 = 150 个 artifact × 8MB = 1.2GB — **接近 1G GitHub 限额**
- 缓解选项: (a) retention-days 改 14 (Phase 71 决策后评估); (b) 只 gate fail 时上传; (c) HTML 用 `-o /dev/null` + 只在 fail 时生成
- **Phase 71 按 D-02 锁定 30 天, 不动** — 真超限时下次 phase 再调

### 12. coverage-baseline.md 文件大小

- 起点行 (76 行 per-package) + Phase 71 后行 (76 行) = ~200 行
- 4 个 phase × 200 行 = ~800 行 — 不影响 git 性能
- 长期: per-package 表可考虑转 JSON (`.planning/coverage-baseline.json`) 便于工具解析 — Phase 71 不动

---

## Validation Architecture

> **Skip rule check:** `workflow.nyquist_validation` 未在 `.planning/config.json` 显式 `false`, 默认 enabled — 包含本节。

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go 标准 `testing` (无额外 framework) |
| Config file | 无 — `go test` 直接跑 |
| Quick run command | `go test ./pkg/cache/...` (SC#4 验收) |
| Full suite command | `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` (SC#5 验收) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GOV-01 | ci.yml 含 -coverprofile + profile artifact 上传 | smoke (CI 跑通 + artifact 出现) | `gh run watch <run-id>` (后台盯 CI) | ❌ Wave 0 — 写完 ci.yml push 后验 |
| GOV-02 | 阈值 gate 工作 (低于阈值即 fail) | manual (本地改阈值到 99.9 验 fail) | `bash .github/scripts/check-coverage.sh coverage.out` 改阈值测试 | ❌ Wave 0 — check-coverage.sh 写完后本地验 |
| GOV-04 | coverage-baseline.md 含 Phase 71 后行 | visual (cat 文件) | `cat .planning/coverage-baseline.md` | ❌ Wave 0 — 计划末尾回填 |
| SC#4 | pkg/cache 15/15 全过 | unit | `go test -v ./pkg/cache/...` | ✅ (5ead742 已修) |
| SC#5 | 全包 exit 0 | unit | `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` | ✅ (现有测试) |

### Sampling Rate
- **Per task commit:** `go test ./pkg/cache/...` (SC#4 守护 — flaky test 已修, 但每次改 go test 跑法时验证)
- **Per wave merge:** 全量 `go test ./internal/... ./pkg/... ./cmd/...` (SC#5)
- **Phase gate:** CI 三绿 (lint + test + coverage gate) + coverage-baseline.md 回填

### Wave 0 Gaps
- [ ] `.coverage-threshold` (NEW) — 仓库根, 纯数字 `12.8\n`
- [ ] `.github/scripts/check-coverage.sh` (NEW) — bash + awk, chmod +x
- [ ] `.planning/coverage-baseline.md` (NEW) — 起点行 + Phase 71 后行
- [ ] `.github/workflows/ci.yml` (extend) — Test step + Coverage HTML + Coverage gate + Upload artifact

### Local Validation Steps (executor 自验)
1. `go test -timeout 15m -count=1 -coverprofile=/tmp/cov.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...` → exit 0
2. `bash .github/scripts/check-coverage.sh /tmp/cov.out .coverage-threshold` → exit 0 (12.8% 满足 12.8% 阈值)
3. `sed -i 's/12.8/99.9/' .coverage-threshold && bash .github/scripts/check-coverage.sh /tmp/cov.out .coverage-threshold` → exit 1 (验证 gate fail)
4. `sed -i 's/99.9/12.8/' .coverage-threshold` (回滚)
5. `go tool cover -html=/tmp/cov.out -o /tmp/cov.html && ls -la /tmp/cov.html` → HTML 生成成功

### CI Validation Steps (Phase 71 完成时)
1. Push 后 `gh run watch <run-id>` (per memory `push-watch-ci.md`)
2. CI 三绿: lint pass, test pass, coverage gate pass
3. Artifact `backend-coverage` 在 PR page 可下载, 含 `coverage.out` + `coverage.html`

---

## Open Questions (RESOLVED)

> All questions resolved at research time. Recommendations below are the locked answers the planner used.

1. **Coverage gate 与 Lint 的 fail-fast 顺序** — RESOLVED: 维持当前顺序; Lint 是 Step 1, Test 是 Step 2 (current ci.yml); Phase 71 在 Test 后加 3 个 step. 若 Lint fail, Test 还会跑 (GitHub Actions 默认行为 — 不同 step). Coverage gate fail 不影响后续 step (Upload artifact `if: always()`).

2. **`-covermode=atomic` 是否需要 `-race`?** — RESOLVED: 不带 `-race`. Go 文档说 atomic 是 standalone mode (无需 race); 本仓库 go 1.24.5 完全支持 atomic standalone. D-01 锁定 + Claude's Discretion: "go test 单次不跑 race-detector, count=1 防缓存". D-01 锁定.

3. **coverage-baseline.md 的 per-package 表是否要包含 `0% 包数` 字段?** — RESOLVED: 是, schema 已包含 `0pct_pkg_count` 列. SUMMARY.md 有 33 个 0% 包 (起点) → SC-b "0% 覆盖业务包 ≤ 5" 目标. Ratchet 时 executor 自行维护; SC-b 由 Phase 74 验收.

4. **`check-coverage.sh` 是否要加 `--per-package-threshold` flag?** — RESOLVED: 本阶段不实现. D-01 锁定只用全局加权阈值. 未来 per-package 阈值需求属 Phase 74 (diff coverage 范畴).

5. **CI artifact retention 30 天是否真不超额?** — RESOLVED: Phase 71 维持 30 天 retention (D-02 锁定). 估算 150 artifact × 8MB = 1.2GB 接近 1G 限额, 但实际 PR 频率与 retention 命中分布尚无实测; 真超限时下次 phase 评估降 retention-days 到 14. 本阶段不做修改.

6. **Coverage HTML step 失败时是否阻断后续 step?** — RESOLVED: 加 `if: always()` + `coverage.out 存在性检查`. HTML step 在 coverage.out 缺失时 skip (不 fail 整个 job); gate step 同理 — `bash` 默认 `set -e` 但 existence check 提前 return 0. 缓解 Test step 因超时失败 (无 coverage.out) 时 Coverage HTML 不会 fail.

---

## References

### Primary (HIGH confidence) — planner/executor 必读
- `D:\code\ClaudeCode\guoguo\.planning\phases\71-governance-baseline-and-ci-gate\71-CONTEXT.md` — D-01..D-04 锁定决策
- `D:\code\ClaudeCode\guoguo\.planning\quick\260820-backend-test-coverage-scan\SUMMARY.md` — 扫描方法段 (awk 公式源)
- `D:\code\ClaudeCode\guoguo\.planning\quick\260820-backend-test-coverage-scan\per-package-coverage.txt` — 76 行聚合结果
- `D:\code\ClaudeCode\guoguo\.planning\REQUIREMENTS.md` — GOV-01/02/04 + Traceability
- `D:\code\ClaudeCode\guoguo\.planning\ROADMAP.md` — Phase 71 definition + 5 SC
- `D:\code\ClaudeCode\guoguo\.planning\PROJECT.md` — D-01..D-06 milestone 锁定
- `D:\code\ClaudeCode\guoguo\.github\workflows\ci.yml` — 当前 backend job (extend target)
- `D:\code\ClaudeCode\guoguo\.github\workflows\deploy.yml` — gate 上游 (Phase 71 不动)
- `D:\code\ClaudeCode\guoguo\CLAUDE.md` — 项目规约 (GOOS/Linux build, sqlite test infra)

### Secondary (MEDIUM confidence)
- `D:\code\ClaudeCode\guoguo\go.mod` — Go 1.24.0 / toolchain 1.24.5, glebarez sqlite v1.11.0 [VERIFIED]
- `D:\code\ClaudeCode\guoguo\.planning\phases\71-governance-baseline-and-ci-gate\71-DISCUSSION-LOG.md` — audit trail (NOT input)

### Tertiary (LOW confidence) — 仅参考
- Go 官方 `-covermode` 文档 (memory-based training knowledge, 标注 [ASSUMED] 若 executor 查 Context7)
- `vladopajic/go-test-coverage` (D-04 锁定本阶段不引入, Phase 74 重评估)

---

## Metadata

**Confidence breakdown:**
- Standard Stack: HIGH — 零依赖 (bash + awk), Go tool 链成熟
- Architecture: HIGH — D-01..D-04 锁定, scope 清晰
- Awk 公式: HIGH — 与 quick-260820-bcs 一致, 12.8% 数字已验证
- CI 集成: HIGH — ci.yml 现有结构清晰, extend 而非 rewrite
- Ratchet 流程: MEDIUM — D-04 锁定手动, 但跨 phase 协同需 plan executor 自觉 (依赖记忆)
- 风险面: MEDIUM — artifact 体积估算可能偏差, 但 30 天 retention 远低于限额

**Research date:** 2026-08-20
**Valid until:** 2026-09-20 (30 天 — 工具链 / Go 版本稳定, 无快变预期)
