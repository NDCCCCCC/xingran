---
phase: 74-p2-finalize-and-diff-coverage
plan: 11
subsystem: final-ratchet-milestone-audit
status: complete
date: 2026-08-22
---

# 74-11 SUMMARY: coverage ratchet 25.9% → 55.5% + v1.26 milestone closeout

## Result

Phase 74 收口:11 个计划(74-01..74-11)+ escalation gap-closure 74-12(ad-hoc,按
Task 1 escalation 条款)全部执行。原子 ratchet commit(7 文件 per D-07 + D-11)落地,
`.coverage-threshold` 25.9 → **55.5**,check-coverage.sh 扩展第 4 层 p2_package_check
(exit 5)。**v1.26 milestone SHIPPED** — SC-b/c/d/e 4/5 达成,SC-a shortfall 诚实
文档化(55.56% vs 70% 目标,结构性阻塞)。

## Coverage delta(ratchet chain)

| 阶段 | weighted avg | 0%-pkg | ratchet |
|------|-------------:|-------:|---------|
| 起点 (2026-08-20) | 12.8% | 33 | — |
| Phase 71 后 | 12.8% | 33 | (gate 建立) |
| Phase 72 后 | 21.5% | 31 | 12.8→21.5 |
| Phase 73 后 | 25.9% | 22 | 21.5→25.9 |
| **Phase 74 后** | **55.5%** (实际 55.56) | **5** | **25.9→55.5** |

- total_stmts 43652(口径不变)/ total_covered **24254**
- 0%-pkg 22 → 5(减少 17):P2 各包补齐 + 74-12 消除 retry/common/models×4/requests;
  剩余 5 个全部为入口/装配/生成代码(cmd, cmd/agent, internal/api, internal/docs,
  internal/server)

## 10 P2 packages per-package(D-15)

| Package | Coverage | Floor | Status |
|---------|---------:|------:|--------|
| internal/api/v1/operations | 72.30% | 70.0 | PASS |
| internal/api/v1/asset | 84.52% | 70.0 | PASS |
| internal/api/v1/network | 75.34% | 70.0 | PASS |
| internal/services/rpa | 86.11% | 70.0 | PASS |
| internal/services/vdi | 85.09% | 70.0 | PASS |
| internal/core | 38.33% | 38.33 ratcheted | BLOCKED — documented |
| internal/device | 39.07% | 39.07 ratcheted | BLOCKED — documented |
| internal/utils | 95.10% | 70.0 | PASS |
| internal/agent/server | 22.08% | 22.08 ratcheted | BLOCKED — documented |
| internal/services/scheduler | 89.82% | 70.0 | PASS |

7/10 达 D-15 70% floor。3 个结构阻塞包的 floor **ratcheted UP-ONLY** 至实际值,
写死在 check-coverage.sh section 4(gate 显式豁免而非降标准;解除条件:包跨过
70.0% 即删除对应 P2_RATCHET 行)。

## 8 P1 packages(零回归,D-04/D-10 strict)

duty 83.0 / knowledge 84.2 / rpa 79.2 / vdi 76.2 + services duty 95.6 /
knowledge 95.3 / network 92.1 / monitor 95.3 — 全部 ≥70%,gate exit 4 层守护。

## 4 层 enforcement(全部 CI 生效)

| 层 | 检查 | exit code | 落地 |
|----|------|----------:|------|
| 1-2 | weighted avg ≥ threshold (55.5) | 1 | Phase 71 |
| 3 | P1 8 包 per-package ≥70% | 4 | Phase 73 (73-05) |
| 4 | P2 10 包 per-package(70×7 + ratcheted×3) | 5 | **Phase 74 (本 plan)** |
| — | PR diff coverage ≥80% | 1 | Phase 74 (74-10, GOV-03/D-14) |

## Escalation 路径记录(Task 1 触发)

Task 1 测量发现 P2 4 包 <70% + 0%-pkg 12 > 5 + weighted 54.65% < 70%。按 plan
escalation 条款执行 ad-hoc **74-12**(测试-only):utils 54.99→95.10 / operlog
→94.4 / retry 0→93.9 / common 0→100 / models×4 0→47.8~90.9 / operations/requests
0→100。二轮测量:P2 7/10、0%-pkg 5、weighted 55.56%。剩余缺口为结构性阻塞
(scrapligo 具体 SSH driver / Core.Init 全链 / agent 子进程 server / addomain LDAP
池 / operations Excel+geocoding 外部 HTTP),单测环境不可构造 → ratchet-to-actual
+ shortfall 文档化(pre-compaction 确认的路径)。

## Milestone SC status(详见 74-MILESTONE-AUDIT.md)

- SC-a weighted ≥70%:**部分达成** 55.56%(差 14.44pp,阻塞面清单化)
- SC-b 0%-pkg ≤5:**达成**(5/5,全部入口/装配/生成代码)
- SC-c threshold gate active:**达成**(25.9→55.5,UP-only ratchet chain)
- SC-d full go test 零失败:**达成**(65 ok / 0 FAIL / 0 panic)
- SC-e PR diff coverage ≥80% gate:**达成**(74-10 D-14 自实现,PR-only job)

## Per-SCALE / per-requirement

| Req | 状态 | 证据 |
|-----|------|------|
| GOV-03 | ✓ | ci.yml coverage-diff job + check-diff-coverage.sh(74-10-SUMMARY 验证矩阵) |
| SCALE-01 | ✓ 7/10 + 3 documented | 本 SUMMARY P2 表 + 74-08-SUMMARY 阻塞分析 |
| SCALE-02 | ✓ | 74-09:response 96.1 / ldaputils 97.0 / captcha 87.5 / time 85.7 等 |
| SCALE-03 | 部分 | 25.9→55.5 ratchet 落地;70% 目标 shortfall 文档化 |

## 执行统计(Phase 74 全程)

- 计划:11 planned + 1 escalation = 12;全部 EXECUTED
- 测试文件:~40 个 `*_test.go`(74-01..07 见各自 SUMMARY;74-08 11 个;74-12 9 个)
- QUIRKS 锁定:15 项(D-12 不修复,测试注释 + SUMMARY 双记录)
- 业务代码变更:**0**(D-12 STRICT 全程,`git diff` 仅 *_test.go + 文档 + gate 脚本)
- Phase executor:gsd-execute-phase 74(inline sequential,subagent 通道故障降级)
- CI 验证:本地 4 层 gate 全 PASS(exit 0);push 延后给用户(per 73-05 deviation
  先例)— 首次 push 后 ci.yml backend job + coverage-diff(PR 时)生效

## Atomic ratchet commit

- 7 文件 per D-07 + D-11:`.coverage-threshold` + `.planning/coverage-baseline.md`
  + `.github/scripts/check-coverage.sh` + `.planning/STATE.md` + `.planning/ROADMAP.md`
  + 本 SUMMARY + `74-MILESTONE-AUDIT.md`
- SHA 见 `git log --oneline -1`(提交后本文件不再 amend,Phase 71 protected-branch
  先例:如需补 SHA 走 fast-forward commit)

## Deviations

1. Task 1 escalation 触发 → ad-hoc 74-12(plan 预留条款,非偏差,记录备查)
2. P2 floor 落地为 70×7 + ratcheted×3(而非 70×10)— gate 诚实性优先:
   显式 UP-ONLY 豁免优于静默降 floor
3. Task 4 CI push 延后(73-05 同款;用户 push 后可用 `gh run watch` 验证,
   memory `push-watch-ci.md`)
