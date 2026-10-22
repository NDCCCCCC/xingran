---
phase: 59
review: 59-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: open
    title: "无权限时 ReconciliationTab 仍触发 getDeptTree 网络请求，违反「零请求」契约"
  - id: WR-02
    severity: warning
    disposition: open
    title: "loadList 无过期响应守卫，快速操作可出现旧响应覆盖新响应"
  - id: IN-01
    severity: info
    disposition: open
    title: "BatchSyncADUsers 的 dept_id 回写无 GUARD-01 守卫（与登录路径不对称）"
  - id: IN-02
    severity: info
    disposition: open
    title: "视图分支①存在「跨配置映射」盲区（紧邻已登记的 Known Gap）"
  - id: IN-03
    severity: info
    disposition: open
    title: "handler 重复实现四值白名单循环，未复用 service 侧判定"
  - id: IN-04
    severity: info
    disposition: open
    title: "dropDependentMaterializedViews 的「noop」收尾日志已与行为矛盾"
  - id: IN-05
    severity: info
    disposition: open
    title: "migration_218 写入 VisibleHidden 但实际落库 visible=1（GORM default tag 零值省略）"
  - id: IN-06
    severity: info
    disposition: open
    title: "phase 触碰文件遗留 gofmt 不洁与缩进劣化"
open: 8
total: 8
recorded: 2026-10-19T16:21:06.846Z
---

# Phase 59: Code Review Disposition,,| Finding | Severity | Disposition | Source |,|---------|----------|-------------|--------|,| WR-01 | warning | open | - |,| WR-02 | warning | open | - |,| IN-01 | info | open | - |,| IN-02 | info | open | - |,| IN-03 | info | open | - |,| IN-04 | info | open | - |,| IN-05 | info | open | - |,| IN-06 | info | open | - |,
Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved.

