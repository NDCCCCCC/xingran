---
last_updated: 2026-09-04
milestone: v1.29
update_trigger: v1.29 workstream ROADMAP synced from .planning/ROADMAP.md — v1.27 content archived at .planning/milestones/v1.27-ROADMAP.md; phases 89-95 now tracked here with Progress table (was: stale v1.27 roadmap made phase-complete report is_last_phase=true after Phase 90)
---

# Roadmap: XingRan-Next 运维管理系统 — v1.29 milestone workstream

> **v1.27 及更早的 milestone 历史已归档**: `.planning/milestones/`(v1.27-ROADMAP.md / v1.27-REQUIREMENTS.md / v1.27-phases/)。
> 本文件自 2026-09-04 起只追踪 **v1.29 技术债治理 (Tech Debt Governance)**。

## Current Milestone: v1.29 技术债治理 (Tech Debt Governance)

**Goal:** 按 2026-09-03 综合审计发现的优先级，逐批治理 7 项技术债行动。**核心交付**: 后端常量集中化、CRUD 复用泛型抽象、缓存层三处架构合并、配置备份闭环、前端 API 工厂化、v1.28 阶段性收口。每项行动原子 commit + 既有测试 0 回归。

**审计基线 (2026-09-03)**:

- 后端覆盖率 78.12% / 前端覆盖率 45.13%
- 1688 tests passing / 45/45 dirs Gate
- 18 项真实 TODO / ~20 处中高度硬编码 / 8 个 CRUD services 60-70% 重复 / 缓存层 3 处架构重复 / config_backup 3 TODO

**Source planning data:**

- `.planning/ROADMAP.md` (v1.29 主 ROADMAP，含完整 phase 详情)
- `.planning/REQUIREMENTS.md` (7 类别 / 41 requirements)
- `.planning/PROJECT.md` (Current Milestone v1.29 段, D-01..D-06 locked decisions)

**Milestone success criteria:**

- SC-a (常量集中化): 12+ 处分页硬编码 + 6 处超时硬编码 + URL 协议 + SNMP 端口 + 并发数全部抽到 `pkg/constants/` ✅ (Phase 89 + 90 done)
- SC-b (CRUD 复用): 8 个 CRUD services 复用 `base.Repository[T]`，LOC 减少 ≥2000 行
- SC-c (缓存层统一): 三处 `CacheServiceBase` 合并到单一基类
- SC-d (config_backup 闭环): 3 个 TODO 空函数全部实现 + 回归测试
- SC-e (前端 API 工厂化 + v1.28 SHIP + v1.29 closeout): ~15 个 `*Api.ts` 迁移工厂模式；v1.28 SHIPPED 段写入 MILESTONES；最终 gate 全绿

**Phase 编号:** 从 Phase 89 起（v1.28 用 82-88，v1.27 用 75-81）。

### Phase Dependency Graph

```
Phase 89 (PAGINATION 常量集中化) ✅ ─┐
Phase 90 (TIMEOUTS/PORT/PROTOCOL/CONCURRENCY) ✅ ─┤
                                   ├─→ Phase 91 (CRUD 复用 base.Repository[T])
                                   ├─→ Phase 92 (缓存层三处架构统一)
                                   ├─→ Phase 93 (config_backup 三处 TODO 闭环)
                                   ├─→ Phase 94 (前端 API 工厂化)
                                   └─→ Phase 95 (v1.28 SHIP 收口 + v1.29 closeout)
```

**并行机会**: Phase 91/93 互不依赖可并行；Phase 91/92 都改 internal/services/ 需顺序；Phase 94 前端独立；Phase 95 必须最后。

---

### Phase 89: PAGINATION 常量集中化 ✅ SHIPPED 2026-09-04

**Goal**: 抽取 `pkg/constants/pagination.go` 3 个分页常量（89-CONTEXT 深度讨论后简化），统一经 `pkg/query.NormalizePagination()` 纯函数替换 8 个文件 12+ 处硬编码；AST 锁值 + invariants 扫描；`go test ./...` 0 失败回归。

**Depends on**: Nothing (v1.29 first phase)

**Requirements**: PAGINATION-01..11 (11 项)

**Success Criteria**: 3 常量 + AST 锁值 / 业务代码 100% 引用常量 / go build 0 错误 / go test 0 失败 — 全部达成

**Plans**: 3(3/3 完成)

- [x] 89-01-PLAN.md — leaf const pkg + Pilot knowledge_service.go 迁移 + AST 锁值测试
- [x] 89-02-PLAN.md — handler 层 5 文件迁移(rpa/system/monitor/workorder) + binding max=100 删除(D-15)
- [x] 89-03-PLAN.md — service 层 3 文件迁移(asset/knowledge/account_pool) + CLAUDE.md Pagination Constants Convention 段

**Notes**: 决策 D-01..D-19 见 `.planning/workstreams/milestone/phases/89-pagination-constants/89-CONTEXT.md`；业务行为变更 6 处已用户接受(knowledge 100→10/500→200, account_pool 20→10, ad_domain ==0→<=0 bug 修复等)

---

### Phase 90: TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化 ✅ SHIPPED 2026-09-04

**Goal**: 抽取 4 个 leaf const pkg(timeouts/ports/protocol/concurrency) 共 10 个常量，覆盖 6 处业务超时 + 1 URL 协议 + 1 SNMP 端口 + 1 并发数 + 3 处 scheduler/cron 扩展审计(D-07)；AST 锁值 + 全量回归 0 失败；零业务行为变更(D-09)。

**Depends on**: Phase 89（leaf const + AST 锁值模式先例；内容上无硬依赖）

**Requirements**: TIMEOUTS-01..08 (8 项)

**Success Criteria**: 4 常量文件 + AST 锁值(10 tests) / 8 调用点迁移 / 零字面量残留 / go build+test 0 失败 — 全部达成

**Plans**: 4(4/4 完成)

- [x] 90-01-PLAN.md — 4 个 leaf const pkg + 4 个 AST 锁值测试(timeouts 6 Duration + SNMPPort + HTTPProto/HTTPSProto + CommandConcurrency)
- [x] 90-02-PLAN.md — network handlers 迁移(command_handler 58/61/98 + execution_handler 99/104 + discovery_handler 126)
- [x] 90-03-PLAN.md — scheduler/LDAP/WS 迁移(ad_ldap_client:74 + ws_notice_handler:47 + ad_sync_tasks 重命名 ADSyncTimeout/ADSyncTaskTimeout + cron.go SchedulerShutdownTimeout)
- [x] 90-04-PLAN.md — CLAUDE.md Timeout/Port/Protocol Constants Convention 段 + 全量回归

**Notes**: 决策 D-01..D-11 见 `.planning/workstreams/milestone/phases/90-timeouts-port-protocol-concurrency/90-CONTEXT.md`；REQUIREMENTS.md TIMEOUTS-01 命名已按 D-05/D-06 同步(无 Default 前缀 + 共用 CommandExecTimeout)

---

### Phase 91: CRUD 复用 base.Repository[T] (🔥 高优 P1)

**Goal**: 让 `internal/services/operations/` 下 8 个 CRUD services 复用 `base.Repository[T]` 抽象，LOC 减少 ≥2000 行；首个 pilot (workstation_service) 验证模式可复制。

**Depends on**: Phase 89 + Phase 90（常量基线稳定）

**Requirements**: CRUD-REUSE-01..08 (8 项)

**Success Criteria**:

1. `base.Repository[T]` 抽象完整（Create/Update/Delete/GetByID/List/Statistics/SearchOptions/BatchDelete）
2. 8 个 operations services 全部迁移完成
3. LOC 净减少 ≥2000 行
4. `go test ./internal/services/operations/...` 0 失败
5. handler 端到端 smoke 测试通过（workstation/building/floor CRUD 仍工作）

**Plans:** 4/4 plans complete

Plans:
**Wave 1**

- [x] 91-01-PLAN.md — base.GORMRepository[T] scope 化改造（D-01/D-02/D-03：List scope 函数式 + interface/DSL 删除 + BatchDelete 空 ids 语义反转 + SortScope 双型 helper）+ base/service_test.go 泛型契约锁值

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 91-02-PLAN.md — Pilot: workstation_service 迁移（D-05 typed request 全套接线 + 6 表 JOIN scope 化）+ 分页语义收紧 checkpoint（A2 auto-approved；commits edc51fd/50577a5）

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 91-03-PLAN.md — building + floor + asset 迁移（map 签名不变 + floor 装饰器签名锁定 P7 + floor 行为基线测试先行）+ F3 软删 Total 修复 checkpoint

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 91-04-PLAN.md — 收尾 7 服务（door/wall/server_room/dedicated_line/floor_plan_text/room_device/infopoint，D-06 扩容）+ typesafe 死文件清理 + LOC 审计 ≥800（D-07）+ SUMMARY

**Notes**: SC-1 中 Statistics/SearchOptions 不进 Repository（D-04）、SC-3 LOC 标准为 ≥800（D-07）、服务数为 11 个非 9 个（D-06）——以 91-CONTEXT.md 为准；map 参数服务实际 4 个（F1）、workstation typed 化含 3 字段扩展（F2）等前提修正见 91-RESEARCH.md。

---

### Phase 92: 缓存层三处架构统一 (🟡 中优 P2)

**Goal**: 合并 legacy root + system/* + operations/* 三处 `CacheServiceBase` 重复模式到单一基类，新代码统一继承。

**Depends on**: Phase 91（都改 internal/services/，顺序执行避免 git diff 冲突）

**Requirements**: CACHE-UNIFY-01..05 (5 项)

**Success Criteria**:

1. `base/cache_service_base.go` 提供完整模板方法（Get/Set/Delete/Invalidate/InvalidatePattern）
2. system/ + operations/ 下所有 `*_cache_impl.go` 继承新基类
3. legacy root `*_cache_service.go` 标注 @Deprecated 并迁移
4. `go test ./internal/services/...` 0 失败
5. Cache Monitor 端到端验证（Redis 缓存读写 + 失效生效）

**Plans (3 planned, 待 plan-phase 生成)**:

- 92-01 抽 base/cache_service_base.go 单一基类 + 测试
- 92-02 system/*_cache_impl.go (9 个文件) 继承新基类
- 92-03 operations/*_cache_impl.go + legacy root 迁移 + core.Core 引用同步

---

### Phase 93: config_backup 三处 TODO 闭环 (🟡 中优 P2)

**Goal**: 实现 `config_backup_service.go:158, 206, 543` 三个 TODO 空函数（压缩/解压/恢复逻辑），新增回归测试；端到端 备份 → 恢复 配置一致。

**Depends on**: Phase 89 + Phase 90（与 91/92 互不依赖，可并行）

**Requirements**: BACKUP-CLOSED-01..05 (5 项)

**Success Criteria**:

1. 3 个 TODO 空函数全部实现，删除 `// TODO:` 注释
2. 回归测试覆盖 happy path + 失败场景（磁盘满/校验失败/事务回滚）
3. 端到端：备份 → 修改 → 恢复 → 配置一致性校验通过
4. `go test ./internal/services/...` 0 回归

**Plans (3 planned, 待 plan-phase 生成)**:

- 93-01 实现压缩(line 158) + 解压(line 206)：gzip 标准库，识别 .gz 后缀
- 93-02 实现恢复(line 543)：事务化（读备份 → 校验 schema → 批量 upsert → 失效缓存）
- 93-03 回归测试三路径 + 失败场景 + 端到端验证

---

### Phase 94: 前端 API 工厂化 (🟡 中优 P2)

**Goal**: 设计 `createResourceApi<T>()` 工厂函数，迁移 `src/lib/` 下 ~15 个 `*Api.ts` 到工厂模式；保持向后兼容。

**Depends on**: Phase 89 + Phase 90（前端独立，与 91/92/93 并行）

**Requirements**: API-FACTORY-01..05 (5 项)

**Success Criteria**:

1. `src/lib/apiFactory.ts` 工厂函数实现完整（list/getByID/create/update/delete/import/export 7 方法 + 类型推导）
2. ~15 个 `*Api.ts` 迁移完成，向后兼容（保留同名导出）
3. `npm run type-check` + `npm run lint` + `npm run test` 0 错误
4. 前端覆盖率 ≥45.13%（不下降）

**Plans (3 planned, 待 plan-phase 生成)**:

- 94-01 设计 + 实现 apiFactory.ts + types/apiFactory.ts
- 94-02 迁移 opsApi.ts（building/floor/workstation/asset 等）
- 94-03 迁移其余 ~10 个 *Api.ts（低风险优先）

---

### Phase 95: v1.28 SHIP 收口 + v1.29 closeout + audit (🟢 长期 P4)

**Goal**: v1.28 阶段性收口（45.13% 写入 MILESTONES）+ v1.29 closeout 验证（7 项行动全部完成确认）+ 生成 v1.29-MILESTONE-AUDIT.md；同时收口 workstream 同步遗留（本文件即其修复产物）。

**Depends on**: Phase 91 + 92 + 93 + 94（必须最后）

**Requirements**: CLOSEOUT-01..03 (3 项)

**Success Criteria**:

1. `.planning/MILESTONES.md` v1.28 SHIPPED 段写入
2. `.planning/PROJECT.md` v1.28 段标记 SHIPPED + ARCHIVED；归档 frontend-coverage workstream
3. 所有 gate 全绿（go build/test / npm type-check/lint/test / 后端 CI gate / 前端 CI gate）
4. v1.29-MILESTONE-AUDIT.md 验证报告生成（v1.27 同款模板）
5. v1.29 milestone SHIPPED 状态设置

**Plans (2 planned, 待 plan-phase 生成)**:

- 95-01 v1.28 SHIP 收口：MILESTONES.md + PROJECT.md + frontend-coverage workstream 归档
- 95-02 v1.29 closeout + audit：完整 gate + 7 项行动确认 + audit 报告

---

## Progress

| Phase | Status | Plans | Requirements | Started | Completed |
|-------|--------|-------|--------------|---------|-----------|
| Phase 89 PAGINATION 常量集中化 | SHIPPED | 3/3 | PAGINATION-01..11 | 2026-09-04 | 2026-09-04 |
| Phase 90 TIMEOUTS/PORT/PROTOCOL/CONCURRENCY | SHIPPED | 4/4 | TIMEOUTS-01..08 | 2026-09-04 | 2026-09-04 |
| Phase 91 CRUD 复用 base.Repository[T] | SHIPPED | 4/4 | CRUD-REUSE-01..08 | 2026-09-04 | 2026-09-04 |
| Phase 92 缓存层三处架构统一 | Pending | 0/3 | CACHE-UNIFY-01..05 | — | — |
| Phase 93 config_backup 三处 TODO 闭环 | Pending | 0/3 | BACKUP-CLOSED-01..05 | — | — |
| Phase 94 前端 API 工厂化 | Pending | 0/3 | API-FACTORY-01..05 | — | — |
| Phase 95 v1.28 SHIP + v1.29 closeout | Pending | 0/2 | CLOSEOUT-01..03 | — | — |

**Total:** 7 phases / 41 requirements (27/41 done — 89+90+91 shipped；92-95 待推进)

---

## Out of Scope (locked from v1.29 init)

- 新业务功能（v1.30+ 候选）
- operlog exclude_paths 白名单（独立 deferred）
- 18 项真实 TODO 中非 config_backup 的 17 项
- 前端覆盖率推到 70%
- Phase 88 batch 续推
- brand-spec 视觉深化（v1.22 遗留）

---

*Last updated: 2026-09-04 — Phase 91 SHIPPED（4/4 plans：91-01 base 改造 + 91-02 workstation pilot + 91-03 building/floor/asset + 91-04 收尾 7 服务，11/11 服务 repo 化；commits 5d0008b..963defe 区间）。Phase 90 SHIPPED（4 plans，commits b51f44c..3a2efe5）。Phase 89 SHIPPED 同日（3 plans，commits 238283c..3559626）。92-95 详情见 `.planning/ROADMAP.md`。*
