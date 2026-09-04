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

- [ ] **CRUD-REUSE-01**: 阅读 `internal/services/base/service.go` 现有 `Repository[T]` 接口，补全缺失方法（Statistics、SearchOptions、BatchDelete）到完整 CRUD 抽象
- [ ] **CRUD-REUSE-02**: `internal/services/operations/workstation_service.go` 迁移到 `base.Repository[T]`（首个 pilot，验证模式可复制）
- [ ] **CRUD-REUSE-03**: `internal/services/operations/building_service.go` 迁移到 `base.Repository[T]`
- [ ] **CRUD-REUSE-04**: `internal/services/operations/floor_service.go` 迁移
- [ ] **CRUD-REUSE-05**: `internal/services/operations/asset_service.go` 迁移
- [ ] **CRUD-REUSE-06**: `internal/services/operations/server_room_service.go` + `infopoint_service.go` + `dedicated_line_service.go` + `room_device_service.go` + `door_service.go` 批量迁移（剩 5 个）
- [ ] **CRUD-REUSE-07**: 每个 service 迁移后跑 `go test ./internal/services/operations/...` 全过；新增 `base/service_test.go` 锁定 Repository[T] 泛型契约
- [ ] **CRUD-REUSE-08**: 统计 LOC 减少（target ~2000 行）并写入 SUMMARY；既有 handler 端到端测试 0 回归

## CACHE-UNIFY (缓存层三处架构统一) — 🟡 中优 P2

> **审计源**: legacy root / system/* / operations/* 三处同一 CacheServiceBase 组合模式

- [ ] **CACHE-UNIFY-01**: 抽 `internal/services/base/cache_service_base.go` 为单一基类（提供 Get/Set/Delete/Invalidate/InvalidatePattern 模板方法）
- [ ] **CACHE-UNIFY-02**: `internal/services/system/*_cache_impl.go` (9 个文件) 全部继承新基类，删除重复的 5 行模板代码
- [ ] **CACHE-UNIFY-03**: `internal/services/operations/*_cache_impl.go` (含 floor_cache_impl 等) 全部继承新基类
- [ ] **CACHE-UNIFY-04**: legacy root `internal/services/*_cache_service.go` (data_cache_service 等) 标注 @Deprecated 并迁移到 system/ 下，core.Core 引用路径同步更新
- [ ] **CACHE-UNIFY-05**: `base/cache_service_base_test.go` + system/operations 缓存实现单测覆盖 Get/Set/Delete/InvalidatePattern 路径；`go test ./internal/services/...` 0 失败

## BACKUP-CLOSED (config_backup 三处 TODO 闭环) — 🟡 中优 P2

> **审计源**: `internal/services/config_backup_service.go:158, 206, 543` 三个 TODO 空函数

- [ ] **BACKUP-CLOSED-01**: 实现 `config_backup_service.go:158` 压缩逻辑（gzip 标准库，压缩备份内容到 .gz 文件）
- [ ] **BACKUP-CLOSED-02**: 实现 `config_backup_service.go:206` 解压逻辑（识别 .gz 后缀，调用 gzip 解压）
- [ ] **BACKUP-CLOSED-03**: 实现 `config_backup_service.go:543` 配置恢复逻辑（事务化：读备份 → 校验 schema → 批量 upsert → 失效相关缓存）
- [ ] **BACKUP-CLOSED-04**: 新增 `config_backup_service_78_NN_test.go`（命名遵循项目 _NN 后缀模式），覆盖 compress/decompress/restore 三路径 + 失败场景（磁盘满 / 校验失败 / 事务回滚）
- [ ] **BACKUP-CLOSED-05**: `go test ./internal/services/...` 0 回归；端到端：备份 → 恢复 → 配置一致性校验通过

## API-FACTORY (前端 API 工厂化) — 🟡 中优 P2

> **审计源**: `xingran-react-frontend/src/lib/` 下 ~15 个 `*Api.ts` 散落，结构雷同

- [ ] **API-FACTORY-01**: 设计 `createResourceApi<T>(basePath, resourceName)` 工厂函数（封装 list/getByID/create/update/delete/import/export 7 个 CRUD 方法）
- [ ] **API-FACTORY-02**: 新建 `src/lib/apiFactory.ts` + 类型定义 `src/types/apiFactory.ts`
- [ ] **API-FACTORY-03**: `src/lib/opsApi.ts` (buildingApi/floorApi/workstationApi/assetApi 等) 迁移到工厂模式（保留同名导出，向后兼容）
- [ ] **API-FACTORY-04**: `src/lib/` 下其他 ~10 个 `*Api.ts` 文件逐个迁移（按低风险优先：先迁无复杂业务逻辑的，再迁带 import/export 的）
- [ ] **API-FACTORY-05**: `npm run type-check` + `npm run lint` + `npm run test` 全过；前端覆盖率不下降（基线 45.13% 维持）

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