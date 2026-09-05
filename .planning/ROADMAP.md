---
last_updated: 2026-09-04
milestone: v1.29
status: planning
---

# Milestone v1.29 ROADMAP (技术债治理)

## Goal

按 2026-09-03 综合审计优先级，逐批治理 7 项技术债行动。**核心交付**: 后端常量集中化、CRUD 复用泛型抽象、缓存层三处架构合并、配置备份闭环、前端 API 工厂化、v1.28 阶段性收口。每项行动原子 commit + 既有测试 0 回归。

**审计基线 (2026-09-03)**:
- 后端覆盖率 78.12% / 前端覆盖率 45.13%
- 1688 tests passing / 45/45 dirs Gate
- 18 项真实 TODO / ~20 处中高度硬编码 / 8 个 CRUD services 60-70% 重复 / 缓存层 3 处架构重复 / config_backup 3 TODO

## Source planning data
- `.planning/REQUIREMENTS.md` (7 类别 / 41 requirements)
- `.planning/PROJECT.md` (Current Milestone v1.29 段)
- 2026-09-03 综合审计报告（4 维度并行扫描：TODO/FIXME、硬编码值、重复实现、项目完成度）

## Milestone success criteria (SC-a..e)

- **SC-a (常量集中化)**: 12+ 处分页硬编码 + 6 处超时硬编码 + 1 处 URL 协议 + 1 处 SNMP 端口 + 1 处并发数全部抽到 `pkg/constants/`，业务代码 100% 引用常量 ✅
- **SC-b (CRUD 复用)**: `internal/services/operations/` 下 8 个 CRUD services 全部复用 `base.Repository[T]`，LOC 减少 ≥2000 行 ✅
- **SC-c (缓存层统一)**: legacy root + system/* + operations/* 三处 `CacheServiceBase` 合并到单一基类，新代码统一继承 ✅
- **SC-d (config_backup 闭环)**: 3 个 TODO 空函数全部实现 + 回归测试覆盖；端到端 备份 → 恢复 配置一致 ✅
- **SC-e (前端 API 工厂化 + v1.28 SHIP + v1.29 closeout)**: ~15 个 `*Api.ts` 迁移到工厂模式；v1.28 SHIPPED 段写入 MILESTONES；最终 gate 全绿 ✅

## Phase Dependency Graph
```
Phase 89 (PAGINATION 常量集中化) ─┐
Phase 90 (TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化) ─┤
                                   ├─→ Phase 91 (CRUD 复用 base.Repository[T])
                                   ├─→ Phase 92 (缓存层三处架构统一)
                                   ├─→ Phase 93 (config_backup 三处 TODO 闭环)
                                   ├─→ Phase 94 (前端 API 工厂化)
                                   └─→ Phase 95 (v1.28 SHIP 收口 + v1.29 closeout)
```

**并行机会**: Phase 89/90/93 互不依赖，可并行；Phase 91/92 都改 internal/services/ 但不同子目录，需顺序避免 git diff 冲突；Phase 94 前端独立；Phase 95 必须最后。

---

## Phase 89: PAGINATION 常量集中化 (🔥 立即 P0) ✅ SHIPPED 2026-09-04

**Goal**: 抽取 `pkg/constants/pagination.go` 并替换 12+ 处 `current=1/pageSize=10` 散布到 8 个文件（含 RPA/系统/工单/资产/知识/AD 域），回归测试 0 失败。

**Requirements**: PAGINATION-01..11 (11 项)

**Plans (3)**:
- 89-01 新建 `pkg/constants/pagination.go` + `pagination_test.go` AST 锁值（DefaultCurrent/DefaultPageSize/MaxPageSize/KnowledgePageSizeLarge/KnowledgePageSizeHuge/AccountPoolPageSize 6 个常量）
- 89-02 接入 handler 层（5 个文件）：rpa/handler_helpers, system/ad_domain_handler, system/notice_user_handler, monitor/cache_handler, workorder/{base,periodic}
- 89-03 接入 service 层（3 个文件）：asset/{reconciliation_service,fix_suggestion_service}, knowledge_service, addomain/account_pool；跑 `go test ./internal/... ./pkg/...` 全过

**Success Criteria**:
1. `pkg/constants/pagination.go` 6 个常量定义 + AST 测试锁定值
2. 业务代码 100% 引用常量（除测试 fixture）
3. `go build ./...` 0 错误
4. `go test ./internal/... ./pkg/...` 0 失败（既有 1688+ 测试不回归）
5. LOC 减少估算（重复代码消除）

## Phase 90: TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化 (🔥 立即 P0) ✅ SHIPPED 2026-09-04

**Goal**: 抽取 `pkg/constants/timeouts.go` + `pkg/constants/protocol.go`，替换 6 处业务超时 + 1 处 URL 协议 + 1 处 SNMP 端口 + 1 处并发数硬编码。

**Requirements**: TIMEOUTS-01..08 (8 项)

**Plans (2)**:
- 90-01 新建常量包：timeouts.go（CommandExecTimeout=300s/CommandReadTimeout=60s/ExecutionTimeout=300s/LDAPConnTimeout=30s/DefaultSNMPPort=161/DefaultCommandConcurrency=10）+ protocol.go（HTTPProto/HTTPSProto）+ AST 锁值测试
- 90-02 接入业务代码（5 个文件）：network/command_handler, network/execution_handler, services/ad_ldap_client, network/ws_notice_handler, network/discovery_handler；回归测试

**Success Criteria**:
1. `pkg/constants/timeouts.go` + `protocol.go` 共 7 个常量定义
2. AST 锁值测试通过
3. 业务代码 100% 引用常量
4. `go test ./...` 0 失败

## Phase 91: CRUD 复用 base.Repository[T] (🔥 高优 P1)

**Goal**: 让 `internal/services/operations/` 下 8 个 CRUD services 复用 `base.Repository[T]` 抽象，LOC 减少 ≥2000 行；首个 pilot (workstation_service) 验证模式可复制。

**Requirements**: CRUD-REUSE-01..08 (8 项)

**Plans (4)**:
- 91-01 阅读 `internal/services/base/service.go` 现有 `Repository[T]`，补全缺失方法（Statistics、SearchOptions、BatchDelete）到完整 CRUD 抽象；新增 `base/service_test.go` 锁定泛型契约
- 91-02 Pilot: workstation_service.go 迁移到 base.Repository[T]（最大最复杂 service，跑通模式）
- 91-03 复制模式：building_service + floor_service + asset_service（3 个高频 CRUD）
- 91-04 收尾：server_room + infopoint + dedicated_line + room_device + door（5 个小 service）+ LOC 统计 + SUMMARY

**Success Criteria**:
1. `base.Repository[T]` 抽象完整（Create/Update/Delete/GetByID/List/Statistics/SearchOptions/BatchDelete）
2. 8 个 operations services 全部迁移完成
3. LOC 净减少 ≥2000 行
4. `go test ./internal/services/operations/...` 0 失败
5. handler 端到端 smoke 测试通过（workstation/building/floor CRUD 仍工作）

## Phase 92: 缓存层三处架构统一 (🟡 中优 P2)

**Goal**: 合并 legacy root + system/* + operations/* 三处 `CacheServiceBase` 重复模式到单一基类，新代码统一继承。

**Requirements**: CACHE-UNIFY-01..05 (5 项)

**Plans (4)** *(2026-09-05 plan-phase 校准：ROADMAP 原 3-plan 估算基于审计基线，未计入 D-04 失效涟漪 42 处（19 外围 + 21 in-system + floor 2）、monitor 同名接口 rename、三文档措辞同步与 invariants 守护的工作量；以 92-CONTEXT.md D-01..D-10 为准)*:
- 92-01 抽 `internal/services/base` 缓存抽象包（TTLResolver + CacheProvider 全家 + 泛型函数族 GetOrSetJSON/SetJSON/Invalidate/InvalidatePattern，D-01/D-02/D-03/D-04）+ system type alias 翻转 + nil-receiver 防护（Pitfall 1）+ `cache_service_base_test.go` miniredis 双装配测试（D-09）
- 92-02 system/*_cache_impl.go (9 个文件) 29 处 GetOrSet 样板迁移 base.GetOrSetJSON + 21 处失效调用改写 base.Invalidate*（user pilot → 批量 → notice 逃兵归队）
- 92-03 operations floor 3 处迁移 + CacheInvalidator 底层委托（D-04）+ 外围 19 处失效调用改写 + 删除 system.InvalidateCache* + DataCacheService 原地定性（D-06/D-07）
- 92-04 monitor CacheOperator rename 消歧（D-08）+ invariants 扫描锁（D-10②）+ CLAUDE.md/REQUIREMENTS/ROADMAP 措辞同步（D-10①/D-06）+ LOC 双口径审计（D-05）

**Success Criteria**:
1. `base/cache_service_base.go` 提供完整模板方法
2. system/ + operations/ 下所有 `*_cache_impl.go` 继承新基类
3. legacy root `*_cache_service.go` 标注 @Deprecated 并迁移
4. `go test ./internal/services/...` 0 失败
5. Cache Monitor 端到端验证（Redis 缓存读写 + 失效生效）

*(注: SC-1/SC-3/SC-5 措辞按 D-03/D-06/D-09 校准修订在 92-04 执行——Phase 90 commit 3a2efe5 先例)*

## Phase 93: config_backup 三处 TODO 闭环 (🟡 中优 P2)

**Goal**: 实现 `config_backup_service.go:158, 206, 543` 三个 TODO 空函数（压缩/解压/恢复逻辑），新增回归测试覆盖；端到端 备份 → 恢复 配置一致。

**Requirements**: BACKUP-CLOSED-01..05 (5 项)

**Plans (3)**:
- 93-01 实现压缩 (line 158) + 解压 (line 206)：gzip 标准库，识别 .gz 后缀
- 93-02 实现恢复 (line 543)：事务化（读备份 → 校验 schema → 批量 upsert → 失效相关缓存）
- 93-03 回归测试 `config_backup_service_78_NN_test.go`：覆盖 compress/decompress/restore 三路径 + 失败场景（磁盘满/校验失败/事务回滚）；端到端验证

**Success Criteria**:
1. 3 个 TODO 空函数全部实现，删除 `// TODO:` 注释
2. 回归测试覆盖 happy path + 失败场景
3. 端到端：备份 → 修改 → 恢复 → 配置一致性校验通过
4. `go test ./internal/services/...` 0 回归

## Phase 94: 前端 API 工厂化 (🟡 中优 P2)

**Goal**: 设计 `createResourceApi<T>()` 工厂函数，迁移 `src/lib/` 下 ~15 个 `*Api.ts` 文件到工厂模式；保持向后兼容。

**Requirements**: API-FACTORY-01..05 (5 项)

**Plans (3)**:
- 94-01 设计 + 实现 `src/lib/apiFactory.ts` + `src/types/apiFactory.ts`：封装 list/getByID/create/update/delete/import/export 7 个 CRUD 方法 + 类型推导
- 94-02 迁移 `src/lib/opsApi.ts` (buildingApi/floorApi/workstationApi/assetApi 等)：保留同名导出，内部用 createResourceApi 工厂
- 94-03 迁移其余 ~10 个 `*Api.ts` 文件：按低风险优先（先迁无复杂业务逻辑的，再迁带 import/export 的）；type-check + lint + test 全过

**Success Criteria**:
1. `src/lib/apiFactory.ts` 工厂函数实现完整
2. ~15 个 `*Api.ts` 迁移完成，向后兼容
3. `npm run type-check` + `npm run lint` + `npm run test` 0 错误
4. 前端覆盖率 ≥45.13%（不下降）

## Phase 95: v1.28 SHIP 收口 + v1.29 closeout + audit (🟢 长期 P4)

**Goal**: v1.28 阶段性收口（45.13% 写入 MILESTONES）+ v1.29 closeout 验证（7 项行动全部完成确认）+ 生成 v1.29-MILESTONE-AUDIT.md 验证报告。

**Requirements**: CLOSEOUT-01..03 (3 项)

**Plans (2)**:
- 95-01 v1.28 SHIP 收口：更新 `.planning/MILESTONES.md` 添加 v1.28 SHIPPED 段（45.13% 阶段性收口理由 + 距离 70% 目标 24.87pp）；更新 `.planning/PROJECT.md` v1.28 段 SHIPPED + ARCHIVED；归档 `.planning/workstreams/frontend-coverage/` 目录
- 95-02 v1.29 closeout + audit：跑完整 gate（go build / go test / npm type-check / npm lint / npm test / 后端 CI gate / 前端 CI gate）；7 项行动全部完成确认；生成 v1.29-MILESTONE-AUDIT.md（v1.27 同款模板）

**Success Criteria**:
1. `.planning/MILESTONES.md` v1.28 SHIPPED 段写入
2. `.planning/PROJECT.md` v1.28 段标记 SHIPPED + ARCHIVED
3. 所有 gate 全绿（go / npm / CI）
4. v1.29-MILESTONE-AUDIT.md 验证报告生成
5. v1.29 milestone SHIPPED 状态设置

---

## Progress

| Phase | Status | Plans | Requirements | Started | Completed |
|-------|--------|-------|--------------|---------|-----------|
| Phase 89 PAGINATION 常量集中化 | ✅ SHIPPED | 3/3 | PAGINATION-01..11 | 2026-09-04 | 2026-09-04 |
| Phase 90 TIMEOUTS/PORT/PROTOCOL/CONCURRENCY | ✅ SHIPPED | 4/4 | TIMEOUTS-01..08 | 2026-09-04 | 2026-09-04 |
| Phase 91 CRUD 复用 base.Repository[T] | Pending | 0/4 | CRUD-REUSE-01..08 | — | — |
| Phase 92 缓存层三处架构统一 | Pending | 0/4 | CACHE-UNIFY-01..05 | — | — |
| Phase 93 config_backup 三处 TODO 闭环 | Pending | 0/3 | BACKUP-CLOSED-01..05 | — | — |
| Phase 94 前端 API 工厂化 | Pending | 0/3 | API-FACTORY-01..05 | — | — |
| Phase 95 v1.28 SHIP + v1.29 closeout | Pending | 0/2 | CLOSEOUT-01..03 | — | — |

**Total:** 7 phases / 41 requirements (19/41 done — Phase 89 + 90 SHIPPED；91-95 待推进)

## Execution Order (建议)

由于 GSD 工作流采用 wave-based 并行，**建议执行顺序**:
1. **Wave 1 (并行)**: Phase 89 (PAGINATION) + Phase 90 (TIMEOUTS) + Phase 93 (config_backup 闭环) — 三者互不依赖，可同时启动
2. **Wave 2**: Phase 91 (CRUD 复用) — 依赖 Wave 1 完成以确保常量基线稳定
3. **Wave 3 (并行)**: Phase 92 (缓存统一) + Phase 94 (前端 API 工厂) — 后端 + 前端独立推进
4. **Wave 4 (顺序)**: Phase 95 (closeout) — 必须最后

**预估时间**: 6 phases × ~2-4 hours = 12-24 小时集中推进；按 7 actions / 41 requirements / 18+ atomic commits 估算。

---

## Out of Scope (locked from v1.29 init)

- 新业务功能（v1.30+ 候选）
- operlog exclude_paths 白名单（独立 deferred）
- 18 项真实 TODO 中非 config_backup 的 17 项
- 前端覆盖率推到 70%
- Phase 88 batch 续推
- brand-spec 视觉深化（v1.22 遗留）
