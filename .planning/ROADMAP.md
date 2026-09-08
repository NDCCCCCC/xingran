---
last_updated: 2026-09-08
milestone: v1.32
update_trigger: v1.31 SHIPPED — archive + v1.32 next milestone
---

# Roadmap: XingRan-Next 运维管理系统 — v1.32

> **v1.31 milestone 历史已归档**: `.planning/milestones/v1.31-ROADMAP.md` / `.planning/milestones/v1.31-REQUIREMENTS.md`
> 本文件自 2026-09-08 起追踪 **v1.32**。

## Current Milestone: v1.32 (TBD)

**Goal**: TBD — next milestone not yet defined.

**Source planning data**:

- `.planning/REQUIREMENTS.md`（pending v1.32 scope definition）
- `.planning/PROJECT.md`（Current Milestone 段待更新）

---

## v1.31 Milestone Summary

<details>
<summary>v1.31 V131 技术债清偿 — ✅ SHIPPED 2026-09-08（展开）</summary>

**Timeline**: 2026-09-07 → 2026-09-08（2 天）
**Phases**: 7 (Phase 102-108) | **Plans**: 25 | **Requirements**: 29（12 类别）
**Status**: ✅ SHIPPED

### Requirements Delivered

| Category | Req | Description | Status |
|----------|-----|-------------|--------|
| CACHE | CACHE-01 | captcha 12 处缓存键具名常量 | ✅ |
| CACHE | CACHE-02 | 10 模块 47 处缓存键收敛 | ✅ |
| STATUS | STATUS-01 | 12 处 status 字面量清零 | ✅ |
| PAGI | PAGI-01 | 分页口径归一 NormalizePagination | ✅ |
| CONV | CONV-01 | mac_history 迁 base.GetOrSetJSON | ✅ |
| CONV | CONV-02 | reconciliation 迁 base.GetOrSetJSON | ✅ |
| CONV | CONV-03 | rpa selector 迁 base.GetOrSetJSON | ✅ |
| CONV | CONV-04 | invariants 扫描扩口 | ✅ |
| WIRE | WIRE-01 | wire 契约统一 | ✅ |
| HANDLER | HANDLER-01 | operations 14 handler 收敛 | ✅ |
| HANDLER | HANDLER-02 | monitor 双 handler 去重 | ✅ |
| FEAPI | FEAPI-01..04 | 前端 CRUD 4 文件迁移 apiFactory | ✅ |
| FEMAP | FEMAP-01..03 | 前端映射统一 + 类型卫生 | ✅ |
| TODO | TODO-01..06 | 22 处 TODO 逐项实现或删除 | ✅ |
| SKIP | SKIP-01..02 | skip 测试恢复 | ✅ |
| TS | TS-01..02 | as any / eslint-disable 清零 | ✅ |
| NIL | NIL-01 | rpa/data_mapper nilness 闭环 | ✅ |

### Key Decisions

- D-01: 台账 12 组全做，不留兼容壳
- D-02: 七 gate 不倒退
- D-03: WIRE-01 / FEMAP-03 设计决策 phase 内敲定
- D-04: captcha-background 1=启用语义锁定
- D-05: Phase 编号从 102 续编

### Phase Dependency Graph

```
Phase 102 (CACHE/STATUS/PAGI 机械常量化)
   └─→ Phase 103 (CONV 缓存闭包收敛)
          └─→ Phase 104 (WIRE 契约 + HANDLER 切换)
                 └─→ Phase 107 (TODO 逐项决策)
Phase 105 (FEAPI lib 层 CRUD 收敛)
   └─→ Phase 106 (FEMAP+TS pages 层映射/类型)
Phase 108 (SKIP 测试恢复；独立)
```

### Archived Artifacts

- Full archive: `.planning/milestones/v1.31-ROADMAP.md`
- Requirements: `.planning/milestones/v1.31-REQUIREMENTS.md`
- Phase 102 COMPLETION: `.planning/phases/102-mechanical-constants/COMPLETION.md`
- Phase 103 COMPLETION: `.planning/phases/103-cache-closure-base-authority/COMPLETION.md`

</details>

---

## Phase Details

*(empty — no active phases)*

---

## Progress

| Phase | Status | Plans | Requirements | Started | Completed |
|-------|--------|-------|--------------|---------|-----------|
| Phase 102 机械常量化（缓存键/状态/分页） | ✅ SHIPPED 2026-09-07 | 5/5 | CACHE-01..02 + STATUS-01 + PAGI-01 | 2026-09-07 | 2026-09-07 |
| Phase 103 缓存闭包收敛 base 单一权威 | ✅ SHIPPED 2026-09-08 | 4/4 | CONV-01..04 | 2026-09-08 | 2026-09-08 |
| Phase 104 handler 层架构收敛（wire+handler） | ✅ SHIPPED 2026-09-08 | 4/4 | WIRE-01 + HANDLER-01..02 | 2026-09-08 | 2026-09-08 |
| Phase 105 前端 CRUD 收敛 apiFactory | ✅ SHIPPED 2026-09-08 | 4/4 | FEAPI-01..04 | 2026-09-08 | 2026-09-08 |
| Phase 106 前端映射统一与类型卫生 | ✅ SHIPPED 2026-09-08 | 4/4 | FEMAP-01..03 + TS-01..02 | 2026-09-08 | 2026-09-08 |
| Phase 107 TODO 清零 + nilness 排查 | ✅ SHIPPED 2026-09-08 | 4/4 | TODO-01..06 + NIL-01 | 2026-09-08 | 2026-09-08 |
| Phase 108 skip 测试恢复 | ✅ SHIPPED 2026-09-08 | 4/4 | SKIP-01..02 | 2026-09-08 | 2026-09-08 |

**Total:** 7 phases / 25 plans / 29 requirements — all delivered.

---

## Out of Scope (locked)

- **captcha-background 1=启用语义** — QUIRK-80-03-D 就地锁定，非 bug
- **operlog exclude_paths / LDAP InsecureSkipVerify / agent 裸 c.JSON / 三层 adapter 物理合并 / 协议默认值常量化** — Future Requirements

---

*Last updated: 2026-09-08 — v1.31 SHIPPED; v1.32 scope TBD. Previous milestone: v1.31 V131 技术债清偿 SHIPPED 2026-09-08（7 phases / 25 plans / 29 requirements）.*
