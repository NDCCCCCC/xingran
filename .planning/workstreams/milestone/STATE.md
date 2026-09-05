---
gsd_state_version: 1.0
milestone: v1.29
milestone_name: 技术债治理
status: executing
last_updated: "2026-09-05T14:19:48.240Z"
last_activity: 2026-09-05
progress:
  total_phases: 7
  completed_phases: 5
  total_plans: 21
  completed_plans: 21
  percent: 71
---

# Project State (v1.29 — milestone workstream)

> 本文件自 2026-09-04 起追踪 v1.29。v1.27 workstream 状态历史见 `.planning/milestones/v1.27-ROADMAP.md`。

## Project Reference

Config: "mode": "yolo"

**Core value:** 按 2026-09-03 综合审计报告发现的优先级，逐批治理 7 项技术债行动；后端常量/CRUD/缓存/配置备份 + 前端 API 工厂化 + Phase 88 收口，使代码质量基线从此不可无声倒退。

**Current focus:** Phase 93 — config_backup 三处 TODO 闭环 (P2)

## Current Position

Phase: 94
Plan: Not started
Status: Ready to execute
Last activity: 2026-09-05
Resume file: .planning/workstreams/milestone/phases/94-api-p2/94-CONTEXT.md
Next action: /gsd:verify-work Phase 92；随后启动 93 (config_backup) / 94 (前端 API 工厂化)，Phase 95 必须最后

## Completed Phases (v1.29)

### Phase 91: CRUD 复用 base.Repository[T] — SHIPPED 2026-09-04

- 4 plans (91-01/02/03/04)，commits 5d0008b..963defe 区间
- `base.GORMRepository[T]` scope 函数式六方法仓储（D-01/D-02）：List scope 化 + Repository[T] interface/Query DSL 删除 + BatchDelete 空 ids 语义反转 + SortScope/SortScopeWithTail 双型排序 helper
- 11/11 operations CRUD 服务 repo 化（workstation D-05 typed pilot / building/floor/asset map 签名不变 / door/wall A 型复合尾随 / server_room·dedicated_line·floor_plan_text B 型 / room_device·infopoint JOIN typed P8 顺序敏感）
- 每服务 CRUD 模板（countRecords/fetchRecords/buildListQueryFromRequest）清零；Statistics/SearchOptions 按 D-04 留 service
- latent bugfix 2 个：floor 换楼异步同步死分支复活（Rule 1）+ building/asset 软删 Total 修复（F3/A3 checkpoint auto-approved）
- typesafe 死代码三文件 -597 行 + calculateOffset 修剪；Wave 0 缺口清零（base 契约锁值 + floor 基线 6 锁 + fpt 冒烟）
- REQUIREMENTS § CRUD-REUSE 措辞与 D-04/D-06/D-07 对齐（REQ_SYNC_OK gate）
- LOC 审计：生产代码净减 +408 / 全口径 -8（测试基线 +678 行），D-07 ≥800 未达成诚实记录，F5 组合 2 备选留决策
- 决策 D-01..D-07 见 91-CONTEXT.md；F1..F5 前提修正见 91-RESEARCH.md

### Phase 89: PAGINATION 常量集中化 — SHIPPED 2026-09-04

- 3 plans (89-01/02/03)，commits 238283c..3559626
- `pkg/constants/pagination.go` 3 常量 + `pkg/query.NormalizePagination` 纯函数唯一入口
- 8 文件 12+ 处硬编码迁移；6 处业务行为变更用户已接受
- CLAUDE.md 新增 Pagination Constants Convention 段
- 决策 D-01..D-19 见 89-CONTEXT.md

### Phase 90: TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化 — SHIPPED 2026-09-04

- 4 plans (90-01/02/03/04)，commits b51f44c..3a2efe5
- 4 个 leaf const pkg（timeouts 6 Duration + ports + protocol + concurrency）共 10 常量
- 8 调用点迁移（network handlers + ad_ldap + ws_notice + scheduler/cron 扩展审计 D-07）
- AST 锁值 10 tests；零业务行为变更（D-09）
- CLAUDE.md 新增 Timeout/Port/Protocol Constants Convention 段
- 决策 D-01..D-11 见 90-CONTEXT.md；VERIFICATION gaps 唯一项（REQUIREMENTS 命名漂移）已修复 3a2efe5

## Accumulated Context (carried forward)

### Decisions to preserve

- D-PRINCIPLE (v1.29 全局): 行业最佳实践为唯一依据,允许任何形式重构;去除硬编码/处理 todo/消除重复/合理抽象
- leaf const pkg + AST 锁值(Stability+Count 双锁)模式：Phase 89/90 先例，后续 phase 沿用
- 常量命名不加 Default 前缀（D-05，89/90 一致）
- D-26-01..05 (v1.26 后端覆盖率 gate): 4 层 CI 防倒退继续生效
- D-27-01..04 (v1.27 测试基建): miniredis/httpmock/ScrapliWrapper/LDAPClientIface/TestHelperProcess/AST守护 沿用
- operlog regression_test.go: 11 强制敏感关键词、25 OperType 常量全程保持绿
- status_constants_test.go: 状态 0/1 命名常量全程 AST 锁值

### Workstream 同步修复 (2026-09-04, 本 commit)

- `.planning/workstreams/milestone/ROADMAP.md` 由 stale v1.27 内容重写为 v1.29 追踪格式
- 根因：workstream roadmap 未随 v1.29 启动同步，phase-complete 在 Phase 90 后误报 is_last_phase=true
- 91-95 现已在 Progress 表注册，next_phase 可正确解析为 91

### Blockers (active)

- 无

### Pending Todos (carry forward, not in v1.29 scope)

- `.planning/todos/pending/operlog-exclude-paths.md` — operlog 白名单配置驱动（RPA heartbeat 日志污染），独立 deferred

## Next Step

执行 92-03-PLAN.md — operations 3 处样板 + D-04 外围 19 处失效调用改写 + cache_utils.go 删除 + root 定性 + monitor rename

或并行推进: 93 (config_backup) / 94 (前端 API 工厂化，完全独立)；Phase 95 必须最后

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Phase 92 P01 | 31min | 3 tasks | 10 files |
| Phase 92 P02 | 21min | 3 tasks | 10 files |
| Phase 92 P03 | 20min | 3 tasks | 9 files |
| Phase 92 P04 | 24min | 4 tasks | 7 files |

## Decisions

- [Phase 92]: 92-01: base 缓存抽象全套落地（TTLResolver+CacheProvider 全家+泛型函数族），system 经 5 alias 翻转零改动；私有 setValue 因同包 CacheAdapter 消费导出为 base.SetValue（唯一迁移偏差） — D-01/D-02/D-03 锁定；SetValue 导出为 Rule 3 编译必需最小修复，语义零变更
- [Phase 92]: 92-02: system 9 文件 29 处 GetOrSet 样板全部收敛 base.GetOrSetJSON 单 return + 21 处失效调用改写 base 底层 + notice 逃兵归队（删私有 getExpiration）；user/role List 站点 T=*PageResult 为 Pitfall 5 单向改善；cache mock 须回填 dest（JSON 往返）为 Rule 1 测试契约修复；键构造 diff 级零变更，生产行为零变更 — InvalidateCache* 函数本体留 92-03 与外围 19 处改写同 commit 删除
- [Phase 92]: 92-03: D-04 完全达成——外围 19 处失效调用改写与 system.InvalidateCacheByPattern/ByKey 删除同 commit（编译器当 checklist 证明零遗漏），42 处调用全部收敛 base 唯一失效底层；CacheInvalidator 保留分发器底层委托；floor 3 处样板收敛（:168 注释假阳性未动）；DataCacheService 原地定性（GetExpiration 委托 base + D-07 定位注释不标 @Deprecated，装配链零改动）——plan 委托片段缺取地址为 Rule 3 编译必需修复（组合字面量不可寻址）；A5 纪律：duty/knowledge/network/workorder 12 处 GetOrSet 样板留 v1.30+
- [Phase 92]: 92-04: monitor CacheProvider rename CacheOperator 消歧（D-08 含测试断言面，裸引用清零，base.CacheProvider 仓内唯一权威）；TestNoInterfaceGetOrSetResidue invariants 锁进 CI（D-10② 硬档 0 残留 + 白名单 + 外围 warning 11 处，红→绿演练通过）；CLAUDE.md 五处缓存段修订 + Cache Service Convention 新段 + REQUIREMENTS/ROADMAP 措辞对齐（D-10①/D-06）；LOC 双口径诚实审计：口径 A 毛减 187<200 未达（剔除 D-07 注释投资后 207 达成）+ 口径 B 全口径净增 529（测试投资计划内）；扫描器空接口判定踩 go/parser 空 FieldList 坑为 Rule 1 修复
