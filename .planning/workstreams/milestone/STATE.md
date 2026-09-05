---
gsd_state_version: 1.0
milestone: v1.29
milestone_name: 技术债治理
status: shipped
last_updated: "2026-09-05T23:50:00.000Z"
last_activity: 2026-09-06 -- 95-02 complete + v1.29 SHIPPED（type-check 真实化 + flaky 双修复 + 七 gate 全绿 + audit 报告 + 记账闭环，commits d7e82ff..9173cb9）
progress:
  total_phases: 7
  completed_phases: 7
  total_plans: 26
  completed_plans: 26
  percent: 100
---

# Project State (v1.29 — milestone workstream)

> 本文件自 2026-09-04 起追踪 v1.29。v1.27 workstream 状态历史见 `.planning/milestones/v1.27-ROADMAP.md`。

## Project Reference

Config: "mode": "yolo"

**Core value:** 按 2026-09-03 综合审计报告发现的优先级，逐批治理 7 项技术债行动；后端常量/CRUD/缓存/配置备份 + 前端 API 工厂化 + Phase 88 收口，使代码质量基线从此不可无声倒退。

**Current focus:** v1.29 SHIPPED 2026-09-06 — milestone 已收口，待 /gsd-complete-milestone（用户触发）+ push 决策

## Current Position

Phase: 95 (v1.28 SHIP 收口 + v1.29 closeout + audit) — COMPLETE（2/2 plans）
Plan: 2 of 2 complete（95-01 commit 1c70eeb；95-02 commits d7e82ff / e49916b / 1ef2224 / 9173cb9）
Status: v1.29 SHIPPED 2026-09-06（7 phases / 45 requirements 45/45 done）
Last activity: 2026-09-06 -- 95-02 complete + v1.29 SHIPPED
Resume file: None（milestone 收口；audit 报告 .planning/milestones/v1.29-MILESTONE-AUDIT.md）
Next action: /gsd-complete-milestone 完整 archive（用户触发，deferred）；push 决策留用户（73-05 先例，本地领先 origin/main 179 commits）；v1.30 规划输入 = REQUIREMENTS § V130-CANDIDATES（CACHEDEF-01..05 + JOBSTAT-01）

## Completed Phases (v1.29)

### Phase 95: v1.28 SHIP 收口 + v1.29 closeout + audit — COMPLETE + v1.29 SHIPPED 2026-09-06

- 2 plans (95-01/02)，commits 1c70eeb + d7e82ff..9173cb9
- 95-01: v1.28 收口核对确认零补漏 + 措辞校准 + 记账补漏（BACKUP-CLOSED 补勾 / Progress 表修正 / 41→45）
- 95-02: D-03 type-check gate 真实化（-p tsconfig.app.json 3701 文件真检查面 + useRestoreTask:47 ?? null，npm run build 的 tsc -b 链同根因救活）+ gate ② flaky 双修复（newNetworkTestEnv SetMaxOpenConns(1) 消除 glebarez :memory: 每连接独立空库并发窗口 + TestJbu8003 种子正午锚定绕开 sqlite DATE() UTC 取日缺陷，均 -count=10 复验）+ D-06 七 gate 本地全绿（go build 0 错误 / go test 双口径 0 失败 / type-check 14s 真检查 / lint 0 errors 1389 warnings 存量 / 554 文件 3800 tests / 后端 coverage 78.33% ≥ 77.5 / 前端 45/45 dirs）+ v1.29-MILESTONE-AUDIT.md（六段 + type-check 空转与 flaky 双新增章节，45/45 追溯，89/90 实名证据口径）+ D-09 SHIPPED 双标记 + 94-HUMAN-UAT 流转 resolved（D-03 修复线）+ JOBSTAT-01 登记 V130-CANDIDATES + CLOSEOUT-03 记账闭环（两 ROADMAP Phase 95 行 Complete 2/2 + Total 45/45）
- 生产缺陷 JOBSTAT-01（job_utils.go:57 时区日界看板少计）本相不修，登记 V130-CANDIDATES；job_utils.go 零改动
- 跑批锚定 e49916b + 工作树快照记录放行（4 个未跟踪测试文件随跑批全绿）；push 决策留用户

### Phase 94: 前端 API 工厂化 — COMPLETE 2026-09-06

- 3 plans (94-01/02/03)，commits b3bc745..1771e1d 区间
- 94-01: createResourceApi<T> 8 方法共享工厂（opsApi 私有工厂提升，D-01/D-03）+ CreatePayload 双命名并集派生类型（D-02）+ download.ts blob 链 GET/POST 归一（D-04）+ 双契约测试（D-11）
- 94-02: opsApi/rpaApi/vdiApi 对象形态全接入（双份私有工厂清零 + scriptApi spread + 裸 fetch downloadReport 消灭白得 5min 超时），导出签名零变化
- 94-03: workorder/knowledge/duty/notice/adDomain 扁平 5 件 cluster 委托（签名零变化，100+ 消费文件零改动）；adDomain deleteMapping :501 潜伏 404 URL bug 独立 commit 95be269 登记修复（T-94-07）；D-12 双档 AST 扫描防线（红绿演练验证）+ D-13 CLAUDE.md Convention 段 + D-01 措辞校准 + 覆盖率 gate 45/45 dirs（lib 90.48% ≥ 87.2）
- 13 个 *Api.ts 全部按迁移矩阵处置（3 对象迁移 + 5 扁平委托 + 5 KEEP）；全量 554 文件 / 3800 测试绿；lint 0 errors（1389 warnings 存量基线）
- 决策 D-01..D-14 见 94-CONTEXT.md；executor 偏离裁定（delete 直调保 D-14、硬档 allowedResidues 等值锁）见 94-03-SUMMARY.md

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

v1.29 SHIPPED 2026-09-06 — milestone 全部收口（7 phases / 26 plans / 45 requirements 45/45）。

- /gsd-complete-milestone 完整 archive（.planning/milestones/v1.29-phases/ 迁移、workstream 归位）— 用户触发，本相按 D-09 只做文档 SHIPPED 标记
- push 决策留用户（73-05 先例）：本地领先 origin/main 179 commits，CI 末次绿跑 2026-09-03（Phase 88 内容），CI 未见证 v1.29 代码
- v1.30 输入：REQUIREMENTS § V130-CANDIDATES（CACHEDEF-01..05 + Phase 95 增补 JOBSTAT-01）+ type-check:strict 空转注记 + 4 个未跟踪测试文件入库决策 + operlog exclude_paths todo

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Phase 92 P01 | 31min | 3 tasks | 10 files |
| Phase 92 P02 | 21min | 3 tasks | 10 files |
| Phase 92 P03 | 20min | 3 tasks | 9 files |
| Phase 92 P04 | 24min | 4 tasks | 7 files |
| Phase 94 P03 | 69min | 3 tasks | 9 files |
| Phase 95 P01 | 4min | 2 tasks | 3 files |
| Phase 95 P02 | 68min | 5 tasks | 11 files（含 audit 报告；含七 gate 跑批 ~40min wall time） |

## Decisions

- [Phase 95]: 95-02: D-03 修复线成立（实测全仓 1 处错误 < 5 降级线）——type-check script 显式 -p tsconfig.app.json（探针选型，「根 config 补 references」证伪）+ useRestoreTask:47 ?? null，npm run build 的 tsc -b 链同根因救活（pre-existing exit 2 → 0）；type-check:strict 不修仅注记（CI 未引用）；gate ② 双 flaky 全 test-infra 修复（SetMaxOpenConns(1) 全仓新 pattern + 种子正午锚定），备选 shared-cache DSN 未启用；生产看板缺陷 JOBSTAT-01 登记 V130 本相不修（job_utils.go:57 零改动守 D-04/D-05 红线）；gate ② 双口径（CI 三包主记录 540s + 字面 ./... 补充 476s）均 0 失败；跑批裁定记录放行（Pitfall 4 选项 c，锚 e49916b + 快照前后一致）；audit commit A（1ef2224）/ SHIPPED+UAT+记账 commit B（9173cb9）分线（D-11）
- [Phase 95]: 95-01: MILESTONES v1.28 段四件套核对通过零补漏（45.13%/24.87pp/Phase 88 备选/归档位置）+ PROJECT.md :35 SHIPPED+ARCHIVED 确认零改动 + frontend-coverage workstream 五项齐备零移动（D-01/D-02）；CLOSEOUT-01/02 措辞校准「核对确认（已存在）」并勾选 + 两份 ROADMAP Phase 95 SC-1/SC-2 与 milestone SC-e 同步校准（D-10 同 commit 1c70eeb）；记账补漏：BACKUP-CLOSED-01/02 补勾带证据锚（gzipCompress :603/gzipDecompress :617）+ Progress 表 Phase 91-94 stale 修正（Complete 4/4·4/4·6/6·3/3）+ 41→45 requirements 三文件五处校准（11+8+8+5+5+5+3）；两 ROADMAP 95-01 plan 复选框勾选（完成事实登记，Phase 95 行保持 Pending 由 95-02 收口）
- [Phase 94]: 94-03: 扁平 5 件 cluster 委托完成（workorder/knowledge/duty/notice/adDomain），导出签名零变化 + 5 个 .test.ts 零改动全绿；delete 形状函数统一保持单参 post 直调（plan DELEGATE 清单与 D-14 零改动红线冲突，工厂 delete 传 {} 属 wire 级 body 变更；deleteNotice/deleteOUGroupMapping/deleteMapping 例外——既有测试本就断言 {} body）；D-12 硬档按 allowedResidues 等值锁登记 KEEP 基线（opsApi 4 / vdiApi 1，plan「期望 0」被矩阵 KEEP 判定证伪）；adDomain deleteMapping :501 潜伏 404 独立 commit 登记（v1.29 D-05 例外条款纪律）；CLAUDE.md 新增 Frontend API Factory Convention（D-13）+ REQUIREMENTS/ROADMAP D-01 措辞校准；覆盖率 gate 45/45 dirs exit 0（lib 90.48% ≥ 87.2%）
- [Phase 92]: 92-01: base 缓存抽象全套落地（TTLResolver+CacheProvider 全家+泛型函数族），system 经 5 alias 翻转零改动；私有 setValue 因同包 CacheAdapter 消费导出为 base.SetValue（唯一迁移偏差） — D-01/D-02/D-03 锁定；SetValue 导出为 Rule 3 编译必需最小修复，语义零变更
- [Phase 92]: 92-02: system 9 文件 29 处 GetOrSet 样板全部收敛 base.GetOrSetJSON 单 return + 21 处失效调用改写 base 底层 + notice 逃兵归队（删私有 getExpiration）；user/role List 站点 T=*PageResult 为 Pitfall 5 单向改善；cache mock 须回填 dest（JSON 往返）为 Rule 1 测试契约修复；键构造 diff 级零变更，生产行为零变更 — InvalidateCache* 函数本体留 92-03 与外围 19 处改写同 commit 删除
- [Phase 92]: 92-03: D-04 完全达成——外围 19 处失效调用改写与 system.InvalidateCacheByPattern/ByKey 删除同 commit（编译器当 checklist 证明零遗漏），42 处调用全部收敛 base 唯一失效底层；CacheInvalidator 保留分发器底层委托；floor 3 处样板收敛（:168 注释假阳性未动）；DataCacheService 原地定性（GetExpiration 委托 base + D-07 定位注释不标 @Deprecated，装配链零改动）——plan 委托片段缺取地址为 Rule 3 编译必需修复（组合字面量不可寻址）；A5 纪律：duty/knowledge/network/workorder 12 处 GetOrSet 样板留 v1.30+
- [Phase 92]: 92-04: monitor CacheProvider rename CacheOperator 消歧（D-08 含测试断言面，裸引用清零，base.CacheProvider 仓内唯一权威）；TestNoInterfaceGetOrSetResidue invariants 锁进 CI（D-10② 硬档 0 残留 + 白名单 + 外围 warning 11 处，红→绿演练通过）；CLAUDE.md 五处缓存段修订 + Cache Service Convention 新段 + REQUIREMENTS/ROADMAP 措辞对齐（D-10①/D-06）；LOC 双口径诚实审计：口径 A 毛减 187<200 未达（剔除 D-07 注释投资后 207 达成）+ 口径 B 全口径净增 529（测试投资计划内）；扫描器空接口判定踩 go/parser 空 FieldList 坑为 Rule 1 修复
