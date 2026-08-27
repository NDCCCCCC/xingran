---
phase: 87-p2-pages-r3-rest
plan: 00
type: execute
wave: 0
requirements: [PAGES-04, QUAL-01]
must_haves:
  truths:
    - "[PAGES-04] 剩余 11 个零覆盖页面目录(duty/monitor/vdi/workorder/asset/knowledge/dashboard-system/ad-domain/profile/my-notices/ad)覆盖率提升"
    - "[QUAL-01] 1128 存量不回归,gate 0 FAIL"
---

# Phase 87 执行计划: P2 R3 — 剩余全部页面

| Wave | 目录(stmts) |
|------|------------|
| 1 | duty 1190 + ad-domain 1082 |
| 2 | monitor 627 + vdi 567 |
| 3 | workorder 551 + asset 475 |
| 4 | knowledge 262 + dashboard-system 203 + my-notices 127 + profile 87 + ad 37 |

模式同 85/86: 纯函数直测 + hooks mock + 常量断言 + 模块导入断言。
