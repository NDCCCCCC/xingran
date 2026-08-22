---
phase: 74-p2-finalize-and-diff-coverage
plan: 10
subsystem: ci-diff-coverage-gate
status: complete
date: 2026-08-22
---

# 74-10 SUMMARY: GOV-03 PR diff coverage ≥80% gate

## Result

- **GOV-03 落地**: ci.yml 新增 PR-only `coverage-diff` job,变更 .go 行覆盖率 <80% 即 fail(exit 1 阻断 merge)。
- **D-14 工具选型**: 走 ratchet 条款自实现(见下),零第三方 action 依赖。
- **向后兼容**: backend job 7 步原样保留(加权平均 gate 不动);新 job `needs: backend` 串行,复用 `backend-coverage` artifact,不重复跑测试。
- 原子提交 `719e04a`(2 files, +234)。

## D-14 工具选型记录

| 候选 | 结论 |
|------|------|
| `gocover-coverage` action(D-14 首选) | **无法验证存在** — marketplace/web 均无此 action 的可验证页面 |
| `ory/xcoverage-action`(D-14 备选) | **不存在** — web 搜索无任何结果 |
| `vladopajic/go-test-coverage` | 存在且活跃,但 Phase 71 RESEARCH 评估其 diff 支持不完善,且引入未验证第三方 action 做硬门禁有 supply-chain 面 |
| **自实现(D-14 ratchet 条款)** | **采纳** — `.github/scripts/check-diff-coverage.sh`,与 check-coverage.sh 同范式(D-01 bash+awk 零依赖),100% 满足 GOV-03 ≥80% threshold,完全可审计 |

D-14 原文允许:"如锁定的工具在落地中发现不适合,Phase 74 executor 可降级为自实现方案 (git diff + awk + coverage.out 解析),但必须 100% 满足 GOV-03 ≥80% threshold"。

## 实现语义

**diff coverage = PR 变更的可测 .go 行中被测试覆盖的比例**

- 变更行: `git diff --unified=0 <base>...HEAD`(三点 merge-base),仅 `*.go`,排除 `*_test.go`
- 可测行: 排除空行与 `//` 纯注释行
- measured 语义(对齐 diff-cover 惯例):
  - 文件在 coverage.out 中有 block → 只有落在某个 block 内的变更行才进分母
    (`package` / import / 声明行不可覆盖,不罚分母)
  - 文件不在 coverage.out 中(包从未被测试执行)→ 全部变更可测行计入分母且未覆盖
    (全新未测试文件不得免费通过)
- 覆盖判定: 变更行落在 hit count > 0 的 block 内(block 粒度,与 `go tool cover` 一致)
- 输出: 每行 UNCOVERED 清单 + 每文件 FILE 行 + DIFF 汇总 + PASS/FAIL

**Exit codes**(镜像 check-coverage.sh): 0=pass/skip,1=gate fail,2=usage/输入缺失。

## ci.yml 变更(diff 摘要)

```yaml
coverage-diff:                        # 新 job,插在 backend 与 frontend 之间
  runs-on: ubuntu-latest
  timeout-minutes: 10
  needs: backend                      # 复用其 coverage.out artifact
  if: github.event_name == 'pull_request'
  steps:
    - actions/checkout@v7 (fetch-depth: 0)        # merge-base 需要全量历史
    - actions/download-artifact@v4 (backend-coverage)
    - bash .github/scripts/check-diff-coverage.sh coverage.out origin/${{ github.base_ref }} 80
```

## 本地验证

| 场景 | 输入 | 结果 |
|------|------|------|
| skip 路径 | base=HEAD~1(仅 test/docs 变更) | `no testable .go lines changed — PASS`,exit 0 ✓ |
| FAIL 路径 | 真实业务 diff 区间 + c.out(duty-only profile) | 50 measured 行 0% → FILE 明细 + FAIL,exit 1 ✓ |
| PASS 路径 | 同上 + threshold=0 | PASS exit 0 ✓ |
| usage 错误 | 无参数 | usage 文案,exit 2 ✓ |
| hunk 解析 | 构造 diff(含 -U0 增删混合 hunk、空行、注释行) | 行号映射正确,空行/注释排除 ✓ |
| measured 语义 | migration_209(28 行变更) | package/import 行不入分母 ✓ |
| YAML 合法性 | python yaml.safe_load | 3 jobs: backend(7 步不变)/coverage-diff/frontend ✓ |

## 备注

- push 到 main 不触发此 job(无 diff base;main 由加权平均 gate 守护)。
- `needs: backend` → backend 失败时本 job 跳过,不会产生误导性结果。
- 首次真实 PR 触发时如有误报,调参点仅在脚本内(threshold 参数 / measured 语义),ci.yml 无需改动。
- 按 plan 要求:不主动 push,由用户在测试 PR 上验证后再推。
