# v1.30 Requirements 追溯终表（TRACEABILITY-FINAL）

**生成:** 2026-09-07（Phase 101 / plan 101-01 Task 3）
**用途:** D-101-5 audit 输入——本表是 `gsd-audit-milestone` 产出 `.planning/v1.30-MILESTONE-AUDIT.md` 的输入材料。**本 phase 不写 v1.30-MILESTONE-AUDIT.md**（milestone lifecycle 专属产物，D-101-5 边界保持）。
**核验方式:** Phase 96-100 逐项 Glob 确认 plan SUMMARY 存在于 `.planning/phases/<phase-dir>/`，无独立 SUMMARY 的 plan 以 `git log` 修复 commit sha 为证据（2026-09-07 实测）；Phase 101 为本 phase 自产证据。
**Status 口径:** `done` = 代码/删除处置已入库且有 commit sha 或 SUMMARY 证据；`待人工 UAT` = 需真实 PG 环境人工执行（D-101-3 唯一合法 human-gated 项），不写未经核验的 passed。

## 22/22 追溯终表

| Requirement | Phase | Status | 证据指针 |
|-------------|-------|--------|----------|
| CACHEDEF-01 | Phase 96 | done | commit `d723570`（feat(96-01): department GetSelectDataWithCache 写键统一）+ `96-01-SUMMARY.md` |
| CACHEDEF-02 | Phase 96 | done | commit `3684924`（feat(96-01): config InvalidateConfigCache 补 id 键）+ `96-01-SUMMARY.md` |
| CACHEDEF-03 | Phase 96 | done | commit `da1fcc3`（feat(96-01): duty parseInt len>=4 前置修复）+ `96-01-SUMMARY.md` |
| CACHEDEF-04 | Phase 96 | done | commit `e679650`（feat(96-01): workorder 待办缓存键补 Limit 维度）+ `96-01-SUMMARY.md` |
| CACHEDEF-05 | Phase 96 | done | commit `e15f09a`（fix(96-02): normalizeCacheKeyForService HasPrefix+TrimPrefix）+ `acf962a`（quirk lock test）+ `96-02-SUMMARY.md` |
| JOBSTAT-01 | Phase 96 | done（重定性删除处置） | commit `683ad35`（refactor(96-03): 删除 GetJobStatistics + FormatDuration 死代码）+ `96-03-SUMMARY.md` |
| V130R-01 | Phase 97 | done | commit `2dd46a3`（fix(97): V130R-01/02/03 全实现）+ `97-01-SUMMARY.md`（超时互斥原子化 + TestV130R01_*） |
| V130R-02 | Phase 97 | done | commit `2dd46a3` + `97-02-SUMMARY.md`（多实例归属过滤 grace period + TestV130R02_*） |
| V130R-03 | Phase 97 | done | commit `2dd46a3` + `97-03-SUMMARY.md`（业务错误码 409/400 语义化 + TestV130R03_*） |
| V130R-04 | Phase 98 | done | commit `5fd062e`（fix(98-01): buildListCacheKey 冒号转义防碰撞）+ `98-01-SUMMARY.md` |
| V130R-05 | Phase 98 | done | commit `9a40341`（refactor(98-02): 四包迁 base.GetOrSetJSON[T]）+ `2b15574`（fix(98): EscapeCacheKeyValue + mock 修复）+ `98-02-SUMMARY.md` |
| V130R-06 | Phase 99 | done | commit `cf4f61f`（fix(99-01): Model() not Table() for Count 软删过滤）+ `99-VERIFICATION.md`（99-01 无独立 SUMMARY 文件，以 commit 为证据） |
| V130R-07 | Phase 99 | done | commit `9b861c8`（floor_service.go 乐观锁 WHERE building_id + floor_move_building_test.go 4 测试；代码随 99-03 docs commit 入库，`99-02-SUMMARY.md` 记载归属） |
| V130R-08 | Phase 99 | done | commit `f57f886`（refactor(99-03): 抽共享 BuildDeptRecursiveFilter + 中段漏匹配修复）+ `c1b35be`（refactor(99-06): 9 生产站点收敛，verifier 缺口闭合）+ `99-03-SUMMARY.md` + `99-06-SUMMARY.md` |
| V130R-09 | Phase 99 | done | commits `831047d`/`dec4843`/`8dc01e1`/`4602f8c`（99-04 clamp 收敛）+ `852263e`/`13891d5`/`0de7df3`（99-05 workstations-all 端点）+ `3cd5b25`/`83782a5`（99-06 internal/constants 并入 pkg/constants + base 注释处置）+ `99-06-SUMMARY.md`（99-04/99-05 无独立 SUMMARY，以 commit 链为证据） |
| V130R-10 | Phase 100 | done | commit `8de04d6`（refactor(100-01): rpaApi 裁至 17 存活方法）+ `5392f78`（refactor(100-02): vdiApi 端态 + accounts 删除）+ `100-frontend-contract-fixes/RECONCILIATION.md`（116 方法六列对账台账）+ apiFactory invariants keys 基线守卫 |
| V130R-11 | Phase 100 | done | commit `5e4b4cc`（feat(100-02): POST /vdi/vms/operate 补注册 vdi:vm:edit）+ `5392f78` + `RECONCILIATION.md`（rpa 97 dead 删除 + vdi 6 ghost/6 mismatch 处置全表） |
| V130R-12 | Phase 100 | done | commits `e8e98e1`（TDD RED）/`e7fad9a`（GREEN: JSON 错误体检测）/`85f6a1a`（networkApi 下载链收敛 download.ts）+ `100-frontend-contract-fixes/100-VERIFICATION.md` |
| TESTFILE-01 | Phase 101 | done | commit `4222dfa`（test(101): 4 个未跟踪测试文件入库纳入 go test gate，D-101-1）+ `101-01-SUMMARY.md` 七 gate 实测表 |
| UAT62-01 | Phase 101 | 待人工 UAT | `62-HUMAN-UAT.md` 场景 1 result [pending]（2026-09-07 环境探针：本机无 docker/psql/本地 PG，D-101-3）→ 自动化前置已就绪：`UAT-RUNBOOK.md` §1（pre-R5 旧结构 MV 构造 SQL + 启动断言清单已备，待人工提供一次性试验 PG 执行） |
| UAT62-02 | Phase 101 | 待人工 UAT | `62-HUMAN-UAT.md` 场景 2 result [pending]（同上环境约束）→ 自动化前置已就绪：`UAT-RUNBOOK.md` §2（advisory lock 持锁/双实例两路径 + pg_locks 残留检查已备） |
| UAT62-03 | Phase 101 | 待人工 UAT（暂记） | `62-HUMAN-UAT.md` 场景 3——本表落盘时点 101-02 尚未执行完毕，按 D-101-3 如实暂记 pending；unit 级前置（`internal/core/db/init_data_test.go` TestCreateDefaultUser 四语义）已确认存在。101-02 场景 3 sqlite 全自动执行结果出来后本行将同步刷新（与 62-HUMAN-UAT.md 回写严格一致） |

## 口径注记

1. **表头边界:** 本文件是 audit 输入，不是 audit 本身；`.planning/v1.30-MILESTONE-AUDIT.md` 由 milestone lifecycle 产出。
2. **SUMMARY 缺口诚实记录:** Phase 99 的 99-01/99-04/99-05 三个 plan 无独立 SUMMARY 文件（目录实测），对应 requirement 以 git log commit sha 为证据；Phase 100 无 per-plan SUMMARY，以 `100-VERIFICATION.md` + `RECONCILIATION.md` + commit 链为证据。
3. **UAT62-01/02 是 milestone 全程唯一合法 human-gated 项**（D-101-3）：2026-09-07 环境探针（无 docker/psql/本地 PG；config.yaml PG 指向 Supabase 远端且 `internal/core/db/database.go` 记载 pooler AutoMigrate 卡死风险）→ 不具备自动执行条件，runbook 已备待人工执行。
4. **OVR 台账补记**（V130R-06/V130R-07 各一条，99-VERIFICATION deferred 项）: 归宿为 milestone audit（`v1.29-REQUIREMENTS.md:157-158` 定义为 milestone 级 manual-only 项），由 milestone lifecycle 落账，本表不重复承载。

## 证据文件存在性实测（2026-09-07）

- `.planning/phases/96-cache-kanban-defect-fixes/`: 96-01/96-02/96-03-SUMMARY.md ✓
- `.planning/phases/97-config-backup-restore-chain-hardening/`: 97-01/97-02/97-03-SUMMARY.md ✓
- `.planning/phases/98-cache-key-security-and-base-migration/`: 98-01/98-02-SUMMARY.md + SUMMARY.md ✓
- `.planning/phases/99-operations口径统一/`: 99-02/99-03/99-06-SUMMARY.md ✓（99-01/99-04/99-05 无 SUMMARY，commit 链补位）
- `.planning/phases/100-frontend-contract-fixes/`: 100-VERIFICATION.md + RECONCILIATION.md ✓（无 per-plan SUMMARY，commit 链补位）
