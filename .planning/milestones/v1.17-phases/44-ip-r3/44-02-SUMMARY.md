---
phase: 44-ip-r3
plan: 02
subsystem: asset-reconciliation
tags: [r3, exception-rule, cron, baseline, excel, dashboard, operlog, react, tdd]
requires:
  - Phase 44 Plan 01 (CRUD service / matchException / module 常量 / assetApi baseline wire / queryKeys baselineCompare key)
  - Phase 42 R1 reconciliation tables (migration_168)
  - Phase 43 R2 转单 cron (createWorkorderCritical/High 已 shipped)
provides:
  - cleanupExpiredExceptionsDirect cron helper (软停用 is_active=1, 幂等)
  - createWorkorderBySeverity SQL 加 no_workorder 过滤 (IS NULL 兜底, BLOCKER-4)
  - ReconciliationBaselineService (Snapshot/Compare, 独立 COUNT, sys_config JSON)
  - SnapshotBaseline + CompareBaseline handler (44-01 未实现, 本 plan 新增 BLOCKER-2)
  - ImportRules + ExportRules + DownloadTemplate handler + 3 路由 (方案 B)
  - reconciliationExceptionRule ExcelConfig 条目 (9 列顺序严格)
  - dashboard 降噪效果卡片 (3 Statistic + 无 baseline Alert)
  - ResolveReconScopeID + ParseCSVToTextArray helper
  - ReadRawRowsByName operations helper (二次读 Excel 供后处理)
affects:
  - internal/scheduler/reconciliation_tasks.go (cleanupExpiredExceptions case + 转单 SQL)
  - internal/services/asset/reconciliation_exception.go (ImportFromExcel 接口扩展)
  - internal/api/v1/asset/reconciliation_exception_handler.go (struct 加 baselineSvc + 5 新 handler)
  - internal/api/v1/asset/reconciliation_exception_router.go (5 新路由)
  - internal/services/operations/excel_config.go (加 reconciliationExceptionRule 条目)
  - xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx (降噪卡片)
tech-stack:
  added: []
  patterns:
    - PG ANY(NULL) 三值逻辑兜底 (applied_actions IS NULL OR ... != ANY)
    - 软停用不软删除 (UPDATE is_active=1, deleted_at 保持 NULL, 审计链不断)
    - 独立 COUNT 规避 MaxPageSize=100 钳制 (stat-cards-from-list-length-capped-at-100)
    - sys_config JSON 存基线快照 (GetByKey 存在则 Update 否则 Create, 幂等覆盖)
    - Excel 方案 B 专用 ImportRules handler + 后处理 UPDATE scope_id + TEXT[] 转换
    - operlog 强制约定 (SnapshotBaseline 写 / CompareBaseline 读 / ImportRules 写)
key-files:
  created:
    - internal/services/asset/reconciliation_baseline.go
    - internal/services/asset/reconciliation_baseline_test.go
    - internal/services/asset/reconciliation_exception_excel_test.go
    - internal/services/operations/excel_reconciliation_test.go
    - internal/services/operations/excel_raw_rows.go
    - internal/scheduler/reconciliation_tasks_test.go
    - .planning/phases/44-ip-r3/deferred-items.md
  modified:
    - internal/scheduler/reconciliation_tasks.go
    - internal/services/asset/reconciliation_exception.go
    - internal/api/v1/asset/reconciliation_exception_handler.go
    - internal/api/v1/asset/reconciliation_exception_router.go
    - internal/api/v1/asset/reconciliation_exception_handler_test.go
    - internal/services/operations/excel_config.go
    - xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx
decisions:
  - D-R3-A4-03 软停用不软删除 (deleted_at NULL, 审计链 T-44-07)
  - BLOCKER-4 IS NULL 兜底防 PG 三值逻辑漏转 applied_actions=NULL critical/high 异常
  - WARN-8 baseline COUNT 含 silence (与 ListExceptions UI 隐藏 silence 解耦)
  - WARN-7 Excel 方案 B 专用 ImportRules handler + 后处理 UPDATE scope_id + TEXT[] 转换
  - BLOCKER-3 baseline service 文件头含运维文档化注释 (R3 部署前记录 R2 末期基线)
  - D-R3-A1-02 转单 SQL 单一真相源 (Layer 3.5 写 applied_actions, cron 仅读)
metrics:
  duration: 25
  completed: 2026-06-28
  tasks: 4
  files: 14
  commits: 9
---

# Phase 44 Plan 02: R3 收尾 (cron 软停用 + 转单 no_workorder 过滤 + 降噪基线 + dashboard 卡片 + Excel 导入导出) Summary

落地 R3 闭环:cleanupExpiredExceptions cron 软停用过期规则(幂等 + 审计链不断)+ 转单 SQL 加 no_workorder 过滤(IS NULL 兜底防 PG 三值逻辑漏转)+ 降噪基线 Snapshot/Compare service(独立 COUNT 规避 MaxPageSize=100)+ SnapshotBaseline/CompareBaseline handler(44-01 未实现后端)+ dashboard 降噪效果卡片 + Excel 导入导出例外规则(方案 B 专用 ImportRules handler + 后处理 UPDATE scope_id + TEXT[] 转换)。

## What Was Built

### Task 1: cleanupExpiredExceptions cron + 转单 SQL no_workorder 过滤 (IS NULL 兜底)
- **cleanupExpiredExceptionsDirect(ctx, db, now)** 新内部 helper:WHERE `expires_at IS NOT NULL AND expires_at < ? AND is_active = 0 AND deleted_at IS NULL`,`Update("is_active", 1)` 软停用(0=启用→1=停用)。软停用**不软删除**(deleted_at 保持 NULL,审计链不断 D-R3-A4-03 + T-44-07)。WHERE 含 `is_active=0` 保证二次调用 rowsAffected=0(幂等)。`now time.Time` 参数让 SQLite 单测 deterministic(PG NOW() 不可移植)。
- **createWorkorderBySeverity SQL 强制追加** `AND (applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions))` (BLOCKER-4)。理由:`internal/models/reconciliation.go:50` `AppliedActions pq.StringArray` 无 default tag,PG INSERT 未指定值默认 NULL。PG 三值逻辑下 `'no_workorder' != ANY(NULL)` 返回 NULL → 整个 WHERE 该行被过滤 → applied_actions=NULL 的 critical/high 异常被漏转,反向破坏 R2 WORKORDER-01/02。**禁止**裸 `AND 'no_workorder' != ANY(applied_actions)`。
- **sys_job 已 seed**(不重做):`对账-例外规则清理` cron `0 0 3 * * *` 在 `reconciliation_tasks.go:127-131` 已注册,InvokeTarget 映射在 `:232-233`,R3 仅填 case 实现。
- placeholder 字符串 `R1 placeholder, R3 真实实现` 已消失。

### Task 2: 降噪基线 Snapshot/Compare service
- **`BaselineSnapshot` / `BaselineCompareResult`** struct:4 字段快照(snapshot_at + total/totalWorkorders/critical)+ 3 个下降%(exceptions/workorders/critical reduction pct)。
- **`reconciliationBaselineServiceImpl`**:不依赖 system.ConfigService(避免 import 循环 + ctx-agnostic),直接 GORM `Table("sys_config")` 读写。
- **Snapshot**:`countCurrent` 用 3 个独立 `.Count(&...)`(total/totalWorkorders/critical),WHERE 仅 `deleted_at IS NULL` **不加 silence 过滤**(WARN-8:silence 是降噪手段应计入基准,与 ListExceptions UI 默认隐藏 silence 的运维视图偏好解耦)。**严禁** ListExceptions(Pitfall 5: MaxPageSize=100 钳制)。写 sys_config(config_key=`asset.reconciliation.baseline`):GetByKey 存在则 Update config_value,不存在则 Create(`uuid.NewString()` 显式赋 id,SQLite/PG 一致)。**幂等覆盖**(key 唯一)。
- **Compare**:读 baseline JSON,不存在返回 `errors.New("未找到基线快照,请先调用 Snapshot 记录基线")`。算下降%:`pct := (b-c)/b*100`(b=0 返回 0)。
- **文件头运维文档化**(BLOCKER-3):明确告知运维 "R3 部署前 + R2 数据保留期内必须调用 Snapshot 记录 R2 末期基线,否则 SC 8 ≥60% 降噪不可量化验证"。

### Task 3: SnapshotBaseline/CompareBaseline handler + dashboard 降噪卡片
- **ReconciliationExceptionHandler struct 加 `baselineSvc asset.ReconciliationBaselineService` 字段** + `WithBaselineService(svc)` 链式注入(与现有 `WithCore` 模式一致)。
- **SnapshotBaseline(c)**:调 `baselineSvc.Snapshot`,success path 末尾 `operlog.Record(..., OperTypeUpdate)`(写 sys_config 是更新语义)。**写操作必须调 operlog**(CLAUDE.md 强制约定)。
- **CompareBaseline(c)**:调 `baselineSvc.Compare`,失败(含 "未找到基线")返回 **400**(前端依赖此状态码显示引导 Alert,BLOCKER-3 可观察条件)。**读操作不调 operlog**(参考 TestRule/ListRules 只读路径)。
- **router 加** `/baseline/snapshot` + `/baseline/compare` 路由 + router 内构造 `baselineSvc` 注入 handler。
- **dashboard 加降噪效果卡片**:用 `useQuery({queryKey: queryKeys.reconciliation.baselineCompare(), queryFn: reconciliationApi.baseline.compare, retry: false})`。渲染 3 个 Statistic(异常/工单/Critical 下降%):≥60% 绿色达标 + Tag"达标",<60% 橙色。compare 返回 400(无 baseline)时显示 Alert "请先到例外规则管理页记录基线" + 引导链接。queryKey 用 queryKeys 工厂稳定引用(CLAUDE.md useEffect 稳定性)。
- **exception-rules admin 页 "记录当前为基线" 按钮**(44-01 Task 6b 已 wire):grep 验证 `baseline.snapshot` + `invalidateQueries({queryKey: baselineCompare})` + `message.success("基线已记录")` 已存在,本 plan 不重构。

### Task 4: Excel 导入导出 reconciliationExceptionRule (方案 B)
- **excel_config.go 加 `reconciliationExceptionRule` 条目**:9 列顺序严格 = name/ip_range/conflict_types/exception_actions/severity_override/scope_type/scope_name/expires_at/reason(项目记忆 xingran-excel-import-column-position-matching)。name 列 UpsertKey=true + DBField="name"(项目记忆 xingran-excel-import-upsertkey-needs-dbfield)。scopeName 是临时字段**无 DBField**(方案 B 后处理 UPDATE scope_id)。CachePatterns=`reconciliation:*`,PartialUpdate=true。
- **reconciliation_exception.go service 加 `ImportFromExcel(ctx, file)`**(方案 B,WARN-7 锁定):调 `operations.ExcelService.ImportData("reconciliationExceptionRule", file, "")` 写基础字段 + AffectedKeys 返回 name 列表 → `postProcessImportedRules` 二次读 Excel raw 行(operations.ReadRawRowsByName 新 helper)→ 对每条规则按 scope_type 用 `ResolveReconScopeID` 解析 scope_id + `ParseCSVToTextArray` 转换 conflict_types/exception_actions CSV→pq.StringArray → UPDATE sys_reconciliation_exception。GORM 占位符防 SQL 注入(T-44-10)。单条后处理失败不阻断主流程(基础字段已写库,运维可在 admin 页手动补)。
- **ResolveReconScopeID(ctx, db, scopeType, scopeName)** helper:dept→sys_dept.dept_name / user→sys_user.username / global→"" (scope_id NULL, D-R3-A4-02)。软删除兜底 `deleted_at IS NULL`。占位符防注入。
- **ParseCSVToTextArray(csv)** helper:"B,C,D"→["B","C","D"],自动 trim 空格 + 过滤空段。空串返回 nil。
- **ReadRawRowsByName(file, sheetName)** operations helper:打开 Excel 按 header 解析,返回 `map[name]map[headerField]value`。供 ImportFromExcel 后处理读 raw scopeName/conflictTypes/exceptionActions。
- **handler 加 ImportRules + ExportRules + DownloadTemplate**:ImportRules 调 service.ImportFromExcel + operlog.Record(OperTypeImport);ExportRules 调 excel_service.ExportData + operlog.Record(OperTypeExport);DownloadTemplate 调 excel_service.GenerateTemplate + operlog.Record(OperTypeDownload)。
- **router 加** `/exception-rule/import` (RequirePermissions asset:reconciliation:exception:create) + `/export` + `/template`(放宽,audit-only)3 路由。**不在 router.go 预注册**(项目记忆 xingran-excel-import-route-conflict)。

## TDD Gate Compliance

每个 task 严格遵循 RED → GREEN 双 commit 模式:

| Task | RED commit | GREEN commit |
|------|-----------|--------------|
| 1 cron + 转单 SQL | `c37a544d` (6 case: 4 SQLite runtime + 2 static source scan) | `2cd70fde` |
| 2 baseline service | `3fffe0ee` (9 case) | `45eeeff9` |
| 3 baseline handler + dashboard | `6d3375f8` (7 backend static scan) | `a171c741` (含前端 dashboard 卡片 type-check + build) |
| 4 Excel 导入导出 | `3f2e205a` (operations 7 + asset 9 case) | `f2fbb382` |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `cleanupExpiredExceptionsDirect` 加 `now time.Time` 参数 (SQLite 测试可移植)**
- **Found during:** Task 1 GREEN
- **Issue:** 原计划 SQL 用 `expires_at < NOW()`(PG 函数),但 SQLite 测试 DB 无 `NOW()` 函数,4 个 SQLite runtime 测试失败。
- **Fix:** 把 `NOW()` 参数化为 `now time.Time` 入参,cron 调用方传 `time.Now()`(等价 PG NOW() 同一 cron 周期时刻),测试传固定时间。生产 SQL 行为不变。
- **Files modified:** `internal/scheduler/reconciliation_tasks.go`, `internal/scheduler/reconciliation_tasks_test.go`
- **Commit:** `2cd70fde`

**2. [Rule 1 - Bug] baseline Snapshot `Pluck("id", &string)` 在 NULL 行触发 Scan error**
- **Found during:** Task 2 GREEN
- **Issue:** GORM `Pluck("id", &existingID)` 在 0 行结果时返回 `sql: Scan error converting NULL to string`(SQLite + PG 都会触发)。
- **Fix:** 改用 `Pluck("id", &[]string)` + `Limit(1)`(空切片而非 NULL string Scan)。生产 SQL 行为不变。
- **Files modified:** `internal/services/asset/reconciliation_baseline.go`
- **Commit:** `45eeeff9`

**3. [Rule 1 - Bug] baseline Snapshot Create 未显式赋 id (SQLite PRIMARY KEY 接受 NULL 一次)**
- **Found during:** Task 2 GREEN
- **Issue:** `Table("sys_config").Create(&map)` 不触发 model default tag(PG `gen_random_uuid()` default 不生效),SQLite/PG 都会写 NULL id → 二次 Create 冲突。
- **Fix:** 显式 `uuid.NewString()` 赋 id(参照 excel_service.go 现有模式)。
- **Files modified:** `internal/services/asset/reconciliation_baseline.go`
- **Commit:** `45eeeff9`

**4. [Rule 3 - Blocking] ResolveReconScopeID / ParseCSVToTextArray 测试跨包分布**
- **Found during:** Task 4 RED
- **Issue:** 这两个 helper 是 asset 包方法,但原计划把所有 Task 4 测试集中放 operations 包,跨包 import private symbol 不可行。
- **Fix:** 把 ExcelConfig + handler/router 静态扫描测试放 `operations/excel_reconciliation_test.go`(用相对路径 `../../api/v1/asset/...` 读源码),把 ResolveScopeID + ParseCSVToTextArray 测试放 `asset/reconciliation_exception_excel_test.go`(同包)。
- **Files modified:** 测试文件分布调整
- **Commit:** `3f2e205a`, `f2fbb382`

### Out-of-Scope Discoveries (Logged, Not Fixed)

**Pre-existing test failures** — TestValidator_ValidateFloor / ValidateWall / ValidateDoor(3 个子测试),floor plan 3D 编辑器校验 helper,完全无关本 plan。`git stash` 验证在 `3f2e205a` HEAD 也失败。已记 `.planning/phases/44-ip-r3/deferred-items.md`,不在本 plan 修复(scope boundary)。

### Plan Honored Verbatim

- **BLOCKER-4 强制**:转单 SQL 含 `(applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions))`,**禁止**裸 `!= ANY`(grep 验证 IS NULL 兜底存在 + 反向 NotContains 裸 ANY)。
- **BLOCKER-3 运维文档化**:baseline service 文件头注释明确告知运维责任(R3 部署前 + R2 数据保留期内调用 Snapshot)。
- **BLOCKER-2 新增 handler**:44-01 未实现后端 baseline handler,本 plan Task 3 新增 SnapshotBaseline + CompareBaseline(struct 加 baselineSvc 字段)。
- **WARN-8 silence 语义**:baseline COUNT 仅 `WHERE deleted_at IS NULL`,**不加** silence 过滤(静态源码扫描 Snapshot 函数体不含 `silence` / `applied_actions`)。
- **WARN-7 Excel 方案 B**:专用 ImportRules handler + 后处理 UPDATE scope_id + TEXT[] 转换 + 3 项目记忆强约束(列顺序严格 + UpsertKey 需 DBField + 不预注册全局路由)。
- **operlog 强制约定**:SnapshotBaseline=OperTypeUpdate(写),CompareBaseline 不调 operlog(读),ImportRules=OperTypeImport,ExportRules=OperTypeExport,DownloadTemplate=OperTypeDownload。

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` 退出 0 | ✅ |
| `go test ./internal/services/asset/... ./internal/api/v1/asset/... ./internal/scheduler/... -count=1 -short` | ✅ all pass |
| `go test ./internal/services/operations/ -count=1 -short`(过滤预存 floor plan 失败) | ✅ Excel/导入相关测试全通过 |
| `go test ./internal/utils/operlog/ -count=1`(25 OperType + 18 keywords 回归) | ✅ pass |
| `cd xingran-react-frontend && npm run type-check` | ✅ exit 0 |
| `cd xingran-react-frontend && npm run build` | ✅ exit 0 (built in 34.99s) |
| reconciliation_tasks.go cleanupExpiredExceptions 含 `Update("is_active", 1)` + WHERE 含 is_active=0 + 不含 placeholder | ✅ |
| createWorkorderBySeverity SQL 含 `(applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions))` (BLOCKER-4) | ✅ |
| reconciliation_baseline.go 含 Snapshot / Compare + 3 个独立 Count + 不含 ListExceptions + COUNT 不加 silence 过滤(WARN-8) | ✅ |
| reconciliation_baseline.go 文件头含运维文档化注释(BLOCKER-3) | ✅ |
| handler 含 SnapshotBaseline(OperTypeUpdate) + CompareBaseline(400 + 不调 operlog) + ImportRules(OperTypeImport) | ✅ |
| ReconciliationExceptionHandler struct 含 baselineSvc 字段 | ✅ |
| router 含 /baseline/snapshot + /baseline/compare + /exception-rule/{import,export,template} 5 路由 | ✅ |
| excel_config.go 含 reconciliationExceptionRule (9 列顺序严格 + name UpsertKey+DBField) | ✅ |
| service 含 ImportFromExcel (方案 B, ResolveReconScopeID + scope_id 解析) | ✅ |
| dashboard 含降噪效果卡片 + 3 Statistic + 无 baseline Alert | ✅ |

### Manual-Only(execute-phase human verify)
- 跨夜 cron 自动停用过期规则(需等 03:00 或手动触发 CleanupNow)
- 降噪效果卡片渲染(需真实告警数据对比 + 运维先记录基线)
- Excel 导入交互(下载模板 → 填数据 → 上传 → 校验列表刷新 → 检查 scope_id 正确解析)

### Phase-Gate 合并验证(44-01 + 44-02)
- ✅ `go build ./...` 退出 0
- ✅ `npm run type-check && npm run build` 退出 0
- ✅ operlog 回归 25 OperType + 18 keywords 守护
- ⚠ `go test ./...` 全套:3 个预存失败(TestValidator_ValidateFloor/Wall/Door,floor plan 校验,与本 plan 无关,记 deferred-items.md)
- ⏳ SC 1-10 UAT 走查:需运维在 dev DB 验证(manual-only)

## Known Stubs

无。本 plan 全部 4 个 task 都落地了真实业务逻辑,没有遗留 stub。44-01 SUMMARY 中记录的 baseline UI wire stub(assetApi.baseline.snapshot/compare UI 调用、后端待实现)在本 plan Task 3 已接入真实后端 handler + service,stub 解决。

## Threat Flags

无新增威胁面。本 plan 严格按 `<threat_model>` 实现缓解:
- **T-44-07** cleanupExpiredExceptions 硬删除风险 → 软停用 `Update("is_active", 1)` + WHERE deleted_at IS NULL + 测试 `pastDeletedAt.Valid == false` 锁定。
- **T-44-08** 转单过滤数据依赖 + NULL 漏转 → BLOCKER-4 IS NULL 兜底强制 + TestCreateWorkorderNoWorkorderFilterNullActions 锁定(applied_actions=NULL 的 critical 必须被转单)。
- **T-44-09** Excel CIDR 注入 → 复用 excel_service Required 校验 + service 层 ValidateCIDR 兜底 + DB cidr 列 SQLSTATE 22P02。
- **T-44-10** Excel SQL 注入 → ResolveReconScopeID 用 GORM 占位符 `Where("dept_name = ?", scopeName)`,禁字符串拼接。
- **T-44-11** 基线覆盖 → SnapshotBaseline 调 operlog.Record(OperTypeUpdate) 留痕,admin 权限限定。
- **T-44-12** Compare 无 baseline 信息泄露 → 返回 400 + 错误信息"未找到基线"不泄露敏感数据。
- **T-44-SC** 依赖合法性 → 零新增依赖(`google/uuid` 已是项目 dep)。

## Self-Check: PASSED

文件存在性验证:
- ✅ internal/scheduler/reconciliation_tasks.go (修改)
- ✅ internal/scheduler/reconciliation_tasks_test.go (新建)
- ✅ internal/services/asset/reconciliation_baseline.go (新建)
- ✅ internal/services/asset/reconciliation_baseline_test.go (新建)
- ✅ internal/services/asset/reconciliation_exception_excel_test.go (新建)
- ✅ internal/services/asset/reconciliation_exception.go (扩展 ImportFromExcel + ResolveReconScopeID + ParseCSVToTextArray)
- ✅ internal/api/v1/asset/reconciliation_exception_handler.go (扩展 baselineSvc 字段 + 5 新 handler)
- ✅ internal/api/v1/asset/reconciliation_exception_router.go (5 新路由)
- ✅ internal/api/v1/asset/reconciliation_exception_handler_test.go (扩展 Task 3 RED 测试)
- ✅ internal/services/operations/excel_config.go (加 reconciliationExceptionRule 条目)
- ✅ internal/services/operations/excel_raw_rows.go (新建 ReadRawRowsByName helper)
- ✅ internal/services/operations/excel_reconciliation_test.go (新建)
- ✅ xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx (降噪效果卡片)
- ✅ .planning/phases/44-ip-r3/deferred-items.md (预存失败记录)

Commit 存在性验证:
- ✅ c37a544d (Task 1 RED)
- ✅ 2cd70fde (Task 1 GREEN)
- ✅ 3fffe0ee (Task 2 RED)
- ✅ 45eeeff9 (Task 2 GREEN)
- ✅ 6d3375f8 (Task 3 RED)
- ✅ a171c741 (Task 3 GREEN)
- ✅ 3f2e205a (Task 4 RED)
- ✅ f2fbb382 (Task 4 GREEN)
