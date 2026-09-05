---
last_updated: 2026-09-04
milestone: v1.29
update_trigger: v1.29 技术债治理 — 7 项审计行动转 REQ-ID
status: executing
---

# Milestone v1.29 Requirements (技术债治理)

## Goal

按 2026-09-03 综合审计发现的优先级，逐批治理 7 项技术债行动：分页/超时常量统一 → CRUD 服务复用泛型抽象 → 缓存层三处架构合并 → 配置备份闭环 → 前端 API 工厂化 → v1.28 阶段性收口。每项行动原子 commit + 既有测试 0 回归。

**审计基线 (2026-09-03)**:
- 后端覆盖率 78.12% (v1.27 ✅)
- 前端覆盖率 45.13% (v1.28 ⏸️ → 本期 SHIP)
- 1688 tests passing / 45/45 dirs Gate
- 真实 TODO 18 项 (后端 13 + 前端 7)
- 中高度硬编码 ~20 处 (分页 12+ / 超时 6 / URL 协议 1 / SNMP 端口 1 / 并发数 1)
- CRUD 重复：8 services × 60-70% 重复；base.Repository[T] 已存在未用
- 缓存层重复：3 路径同一 CacheServiceBase 模式
- config_backup 3 个 TODO 空函数

---

## PAGINATION (分页常量集中化) — 🔥 立即 P0

> **审计源**: 12+ 处 `current=1/pageSize=10` 散布于 8 个文件

- [x] **PAGINATION-01**: 新建 `pkg/constants/pagination.go`，定义 3 个常量（89-CONTEXT 深度讨论后简化：`DefaultCurrent=1`、`DefaultPageSize=10`、`MaxPageSize=200`；原规格的 `MaxPageSize=10000` 判定为无 DoS 防护价值，`KnowledgePageSizeLarge/Huge` + `AccountPoolPageSize` 按 D-08/D-09/D-16 判定为"拍脑袋"错误抽象，全部删除）
- [x] **PAGINATION-02**: `internal/api/v1/rpa/handler_helpers.go` setPaginationDefaults 内部委托 `query.NormalizePagination`（D-06）
- [x] **PAGINATION-03**: `internal/api/v1/system/ad_domain_handler.go` 内联守卫替换（`== 0` → `<= 0` bug 修复，D-06）
- [x] **PAGINATION-04**: `internal/api/v1/system/notice_user_handler.go` 已迁移
- [x] **PAGINATION-05**: `internal/api/v1/monitor/cache_handler.go` setPaginationDefaults 内部委托（D-06）
- [x] **PAGINATION-06**: `internal/services/workorder/base.go` + `periodic.go` 已迁移
- [x] **PAGINATION-07**: `internal/services/asset/reconciliation_service.go` + `fix_suggestion_service.go` 已迁移
- [x] **PAGINATION-08**: `internal/services/knowledge_service.go` 两个代码路径统一走 `NormalizePagination`（default 100→10, max 500→200，用户已接受）
- [x] **PAGINATION-09**: `internal/services/addomain/account_pool.go` 反常回退改为标准 clamp（default 20→10，用户已接受）
- [x] **PAGINATION-10**: `pkg/query/pagination_test.go` 单测覆盖（DefaultCurrent 缺省 / MaxPageSize 截断 / Offset 计算）；回归 `go test ./...` 0 失败
- [x] **PAGINATION-11**: `pkg/constants/pagination_test.go` AST 锁值（TestPaginationConstantStability + Count 双锁）

## TIMEOUTS (超时/URL/SNMP 端口/并发数常量集中化) — 🔥 立即 P0

> **审计源**: 6 处业务超时 + 1 处 URL 协议 + 1 处 SNMP 端口 + 1 处并发数硬编码

- [x] **TIMEOUTS-01**: 新建 `pkg/constants/timeouts.go` + `pkg/constants/ports.go` + `pkg/constants/concurrency.go`，定义 `CommandExecTimeout=300s`（设备命令执行超时，command_handler + execution_handler 共用 D-06）、`CommandReadTimeout=60s`、`LDAPConnTimeout=30s`、`ADSyncTimeout=30m`、`SchedulerShutdownTimeout=5s`、`ADSyncTaskTimeout=1m`（后 3 个为 scheduler/cron 扩展审计 D-07）+ `SNMPPort=161` + `CommandConcurrency=10`（不加 `Default` 前缀 D-05）
- [x] **TIMEOUTS-02**: 新建 `pkg/constants/protocol.go`，定义 `HTTPProto="http"`、`HTTPSProto="https"` (WS origin 协议白名单，按 D-03 裸 const 拼接 `HTTPProto+"://"+host`，无 helper 函数)
- [x] **TIMEOUTS-03**: `internal/api/v1/network/command_handler.go` 替换 58/61/98 行（Concurrency + 2× Timeout，按 D-02 `int(constants.Xxx.Seconds())` 强类型转换模式）
- [x] **TIMEOUTS-04**: `internal/api/v1/network/execution_handler.go` 替换 99/104 行（Concurrency + Timeout；104 行 `Timeout=300` 共用 `CommandExecTimeout` D-06）
- [x] **TIMEOUTS-05**: `internal/services/ad_ldap_client.go` 替换 74 行 `time.Second*30` → `constants.LDAPConnTimeout`（time.Duration 强类型直接替换 D-02）
- [x] **TIMEOUTS-06**: `internal/api/v1/ws_notice_handler.go` 替换 47 行协议字符串拼接 → `constants.HTTPProto+"://"+host` / `constants.HTTPSProto+"://"+host`（裸 const 拼接 D-03）
- [x] **TIMEOUTS-07**: `internal/api/v1/network/discovery_handler.go` 替换 126 行 `SNMPPort=161` + scheduler/cron 审计扩展（`internal/scheduler/ad_sync_tasks.go` 重命名 `adSchedulerSyncTimeout` → `ADSyncTimeout` + `internal/scheduler/cron.go` 重命名 `defaultShutdownTimeout` → `SchedulerShutdownTimeout` + `internal/scheduler/ad_sync_tasks.go:164` 内联 `1*time.Minute` 抽 `ADSyncTaskTimeout`，均按 D-07/D-08 决策）
- [x] **TIMEOUTS-08**: `pkg/constants/{timeouts,ports,protocol,concurrency}_test.go` AST 锁值（共 10 个测试，含 Stability + Count 双锁模式参考 `internal/utils/operlog/regression_test.go` 模板）；回归测试 `go build ./...` 0 错误 + `go test ./...` 0 失败（既有 1688+ 测试不回归）

## CRUD-REUSE (CRUD 服务复用 base.Repository[T]) — 🔥 高优 P1

> **审计源**: operations/ 下 8 个 CRUD service 60-70% 重复；base.Repository[T] 已存在未用

- [x] **CRUD-REUSE-01**: 改造 `internal/services/base/service.go` 为 gorm scope 函数式 GORMRepository[T] 完整 CRUD 抽象（List scope 化 + 删除 Repository[T] interface 与 Query/WhereCondition DSL，BatchDelete 已存在仅反转空 ids 语义；Statistics/SearchOptions 为异构业务查询，按 D-04 留 service 层不进 Repository）
- [x] **CRUD-REUSE-02**: `internal/services/operations/workstation_service.go` 迁移到 `base.Repository[T]`（首个 pilot，验证模式可复制）
- [x] **CRUD-REUSE-03**: `internal/services/operations/building_service.go` 迁移到 `base.Repository[T]`
- [x] **CRUD-REUSE-04**: `internal/services/operations/floor_service.go` 迁移
- [x] **CRUD-REUSE-05**: `internal/services/operations/asset_service.go` 迁移
- [x] **CRUD-REUSE-06**: `internal/services/operations/server_room_service.go` + `infopoint_service.go` + `dedicated_line_service.go` + `room_device_service.go` + `door_service.go` + `wall_service.go` + `floor_plan_text_service.go` 批量迁移（D-06 扩容后剩 7 个）
- [x] **CRUD-REUSE-07**: 每个 service 迁移后跑 `go test ./internal/services/operations/...` 全过；新增 `base/service_test.go` 锁定 Repository[T] 泛型契约
- [x] **CRUD-REUSE-08**: LOC 净减审计（D-07 混合标准：11 services 全部复用 GORMRepository + 每服务 CRUD 模板清零 ✓ + LOC 净减量化锚点——**OVR-91-01 用户 override 后重校准**：生产代码口径净减 +408 行（全口径 -8，测试基线计划内 +678 抵消；numstat 证据见 91-04-SUMMARY））；既有 handler 端到端测试 0 回归 ✓

## CACHE-UNIFY (缓存层三处架构统一) — 🟡 中优 P2

> **审计源**: legacy root / system/* / operations/* 三处同一 CacheServiceBase 组合模式

- [x] **CACHE-UNIFY-01**: 抽 `internal/services/base/cache_service_base.go` 为缓存抽象单一基类——**92-01 交付**（实际形态按 D-03 锁定：Go method 不能有类型参数，模板方法落地为 **TTLResolver 薄基类 + 泛型包级函数族** `GetOrSetJSON[T]`/`Invalidate`/`InvalidatePattern`，SC-1 措辞随之修订；`SetJSON[T]` 经 review WR-06 用户判定于收口时删除——零调用方 + 先删后写并发丢写窗口）
- [x] **CACHE-UNIFY-02**: `internal/services/system/*_cache_impl.go`（9 个文件）全部继承新基类，删除重复的模板代码——**92-02 交付**（实测 **29 处** GetOrSet 样板含 notice 逃兵归队：嵌入源迁 base.CacheServiceBase + 方法体换泛型函数 base.GetOrSetJSON 单 return；21 处 in-system 失效调用改写 base 底层）
- [x] **CACHE-UNIFY-03**: `internal/services/operations/*_cache_impl.go` 全部继承新基类——**92-03 交付**（实测 **3 处**真实样板收敛 base.GetOrSetJSON——:168 为注释行 grep 假阳性未动；CacheInvalidator 按 D-04 **保留分发器、底层委托** base.InvalidatePattern，Excel 管道 entityType 分发语义零改动）
- [x] **CACHE-UNIFY-04**: legacy root `internal/services/data_cache_service.go` 原地定性为基础设施（root↔system import cycle 硬约束，字面迁移不可行，措辞按 D-06 修订）——**92-03 交付**（消除平行 GetExpiration：保留签名、内部委托 base TTL 逻辑 + D-07 双定位注释，不标 @Deprecated；12+ API 文件引用与 core.Core 装配链零改动）
- [x] **CACHE-UNIFY-05**: `base/cache_service_base_test.go` + system/operations 缓存实现单测覆盖 Get/Set/Delete/InvalidatePattern 路径；`go test ./internal/services/...` 0 失败——**92-01 交付 base 侧**（TestBase92 十用例 miniredis+MemoryCache 双装配，D-09 miniredis 自动化）；**92-04 补齐 invariants 扫描**（`cache_invariants_92_test.go` TestNoInterfaceGetOrSetResidue：system/operations 硬档 0 残留 + 外围 warning 档）；20 包全绿实测

## BACKUP-CLOSED (config_backup 三处 TODO 闭环) — 🟡 中优 P2

> **审计源**: `internal/services/config_backup_service.go:158, 206, 543` 三个 TODO 空函数

- [ ] **BACKUP-CLOSED-01**: 实现 `config_backup_service.go:158` 压缩逻辑（gzip 标准库，压缩备份内容到 .gz 文件）
- [ ] **BACKUP-CLOSED-02**: 实现 `config_backup_service.go:206` 解压逻辑（识别 .gz 后缀，调用 gzip 解压）
- [x] **BACKUP-CLOSED-03**: 配置恢复异步任务化设备下发（恢复=把备份配置下发到网络设备：同设备校验 → 恢复前自动备份 → 基础清洗 → RestoreConfig 下发（fail-fast）→ 回读 hash 校验 → 版本链恢复记录；Phase 93 D-01 校准）
- [x] **BACKUP-CLOSED-04**: 新增 `config_backup_service_93_NN_test.go`（命名遵循项目 _NN 后缀模式；原文误写为 78 前缀，已修正），覆盖 compress/decompress/restore 三路径 + 失败场景（写失败 / 损坏 gzip / 下发中断留痕 / DB 写入失败）
- [x] **BACKUP-CLOSED-05**: `go test ./internal/services/...` 0 回归；端到端：备份 → 恢复 → 配置一致性校验通过（验收 = FileTransport 自动化集成测试断言链，Phase 92 D-09 先例）

## API-FACTORY (前端 API 工厂化) — 🟡 中优 P2

> **审计源**: `xingran-react-frontend/src/lib/` 下 ~15 个 `*Api.ts` 散落，结构雷同

- [x] **API-FACTORY-01**: 提升 opsApi 既有 8 方法工厂为共享 `createResourceApi<T>(config)`（list/get/create/update/delete/batch/statistics/searchOptions；import/export 不进工厂核心、getByID 不采用，per D-01 行业对齐 react-admin/refine）
- [x] **API-FACTORY-02**: 新建 `src/lib/apiFactory.ts` + 类型定义 `src/types/apiFactory.ts`
- [x] **API-FACTORY-03**: `src/lib/opsApi.ts` (buildingApi/floorApi/workstationApi/assetApi 等) 迁移到工厂模式（保留同名导出，向后兼容）
- [x] **API-FACTORY-04**: `src/lib/` 下其余 12 个 `*Api.ts` 文件逐个按迁移矩阵处置（DELEGATE / SPREAD+OVERRIDE / KEEP 三态判定，低风险优先）
- [x] **API-FACTORY-05**: `npm run type-check` + `npm run lint` + `npm run test` 全过；前端覆盖率不下降（基线 45.13% 维持）

## V128-CLOSEOUT (v1.28 阶段性收口 + v1.29 closeout) — 🟢 长期 P4

- [ ] **CLOSEOUT-01**: 更新 `.planning/MILESTONES.md`，添加 v1.28 SHIPPED 段（45.13% 阶段性收口理由 + 距离 70% 目标 24.87pp + 后续可重启 Phase 88 备选）
- [ ] **CLOSEOUT-02**: 更新 `.planning/PROJECT.md`，v1.28 段标记 SHIPPED + ARCHIVED；`.planning/workstreams/frontend-coverage/` 目录归档到 `.archive/` 或保留作历史
- [ ] **CLOSEOUT-03**: v1.29 closeout — 跑完整 `go build ./...` + `go test ./...` + `npm run type-check` + `npm run lint` + `npm run test` + 后端 CI gate + 前端 CI gate；7 项行动全部完成确认；生成 v1.29-MILESTONE-AUDIT.md 验证报告

---

## Traceability (filled by ROADMAP)

| Phase | Goal | Requirements |
|-------|------|--------------|
| 89 | PAGINATION 常量集中化 | PAGINATION-01..11 |
| 90 | TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化 | TIMEOUTS-01..08 |
| 91 | CRUD 复用 base.Repository[T] | CRUD-REUSE-01..08 |
| 92 | 缓存层三处架构统一 | CACHE-UNIFY-01..05 |
| 93 | config_backup 三处 TODO 闭环 | BACKUP-CLOSED-01..05 |
| 94 | 前端 API 工厂化 | API-FACTORY-01..05 |
| 95 | v1.28 SHIP 收口 + v1.29 closeout | CLOSEOUT-01..03 |

**Total**: 7 phases, 41 requirements, 全部覆盖 ✓

---

## V130-CANDIDATES (v1.30+ 缺陷候选 — Phase 92 review 登记，2026-09-05 用户判定)

> 来源：`92-REVIEW.md` WR-01..05——五个迁移前即存在的缓存缺陷，被 v1.29 零行为变更约束有意原样保留。当前 milestone 93/94/95 均不覆盖。修复属**行为变更**，须附带回归测试（v1.29 D-05 例外条款同款纪律）。

- [ ] **CACHEDEF-01**: `system/department_cache_impl.go:74-102` — `GetSelectDataWithCache` 写键 `"dept:tree"`（裸常量）与 `InvalidateDeptCache` 的 `cache:` 前缀模式永不匹配，失效永不命中（潜伏：暂无生产调用方）
- [ ] **CACHEDEF-02**: `system/config_cache_impl.go:71-118` — 单条 Delete 只失效 `config:all` + `config:key:*`，遗漏 `config:id:<id>`，已删配置经详情接口最长 30min 可从缓存读回（`config_router.go:17` 生产可达）
- [ ] **CACHEDEF-03**: `duty/duty_cache_impl.go:333-344` — `parseInt` 的 `len(s) >= 4` 前置使 2 字符月份切片恒返回 0，`GenerateSchedule`/`ManualDuty` 后月度排班缓存失效无效（`duty_handler.go:325` 生产可达）
- [ ] **CACHEDEF-04**: `workorder/workorder_cache_impl.go:207-234` — 待办缓存键仅含 userID，忽略 `GetMyPendingRequest.Limit`，不同 limit 共享同一缓存
- [ ] **CACHEDEF-05**: `monitor/cache_service.go:766-771` — `key[:6] == "xingran:"`（6 字节切片比 8 字节字面量）恒 false，前缀剥离永不生效（Phase 73-04 quirk Q1 同源；CLAUDE.md 旧示例 `key[6:]` 同错，现已随 Cache Service Convention 修订移除）

---

## Out of Scope (explicit exclusions with reasoning)

| 排除项 | 理由 |
|--------|------|
| 新业务功能（v1.30+ 候选） | v1.29 锁定为纯技术债治理，不引入新功能 |
| operlog exclude_paths 白名单（pending todo） | 独立 deferred，是配置驱动改造，与 7 项审计行动不重叠 |
| 18 项真实 TODO 中非 config_backup 的 17 项 | 风险分散，作为后续 milestone 候选；本次只闭环 BACKUP-CLOSED |
| 前端覆盖率推到 70% | v1.28 已 SHIP 45.13%；本里程碑只做工厂化，不补覆盖率 |
| Phase 88 batch 续推 | 已 SHIP 收口，不重启 |
| brand-spec 视觉深化（PROTO-01..04 / VIS-01..03） | v1.22 候选遗留，独立 deferred |

---

## Requirement Quality Self-Check

- ✅ **Specific and testable**: 每个 REQ 含明确文件 + 行号 + 可观测行为
- ✅ **User-centric**: 用"替换 / 迁移 / 闭环 / 收口"等动作词描述
- ✅ **Atomic**: 每条 REQ 单一动作，避免"替换并测试"
- ✅ **Independent**: PAGINATION / TIMEOUTS / CRUD / CACHE / BACKUP / APIFACTORY / CLOSEOUT 7 类相互独立