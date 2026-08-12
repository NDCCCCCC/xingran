---
phase: 39-workstation-dept-location-alias
plan: 02
subsystem: api
tags: [go, gin, gorm, operlog, validation, sys_dept]

# Dependency graph
requires:
  - phase: 39-01
    provides: "SysDeptLocationAlias GORM 模型 + Migration 165 建表 + 4 条 ops:location:alias:* 按钮权限 seed"
provides:
  - LocationAliasService interface + 三级校验 (validateAlias) + CRUD 实现
  - LocationAliasHandler 4 个端点 (List/Create/Update/Delete) + operlog 集成
  - 10 个单元测试守护三级校验契约 + CRUD 链路
affects:
  - 39-04 (router 注册 + 缓存失效 + 最终贯通测试)
  - 39-03 (工位部门下拉 union 注入 — 复用 LocationAliasService 查询接口)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - 三级校验 (自映射/外部机构/后代) 通过 validateAlias 单函数集中实现,Create/Update 复用
    - ancestors 逗号包围字符串匹配 (",ancestors," + ",location_id,") 防止 UUID 前缀子串误判
    - Handler 链式 WithCore 注入 (与 BuildingHandler/WorkstationHandler 同款,不改构造器签名)
    - 校验错误中文透传前端 (HTTP 400 + 明确错误信息,D-02 决策)

key-files:
  created:
    - internal/services/operations/location_alias_service.go
    - internal/services/operations/location_alias_service_test.go
    - internal/api/v1/operations/location_alias_handler.go
  modified: []

key-decisions:
  - "validateAlias 单函数集中三级校验,Create/Update 复用,错误信息含中文关键字('自映射'/'外部机构'/'后代')供前端断言"
  - "scope 字段在 service 层兜底 'workstation'(D-04 决策,空值/缺省一律设置)"
  - "Update 路径只对 dept_id/location_id 实际变更时跑 validateAlias,纯 remark 修改不触发校验(性能 + 体验)"
  - "用裸 id = ? 而非 id::text = ? 让 GORM 在 PG(uuid→string 自动转换) 和 SQLite(直接 TEXT 比较) 双 DB 行为一致"
  - "operlog 模块名严格对齐 workstation_handler.go '工位管理'(CLAUDE.md 强制约定)"

patterns-established:
  - "Handler 链式注入模式: NewXxxHandler(service) 不带 core,后置 WithCore(core) 注入 — 与 router.go 调用点兼容,无需重写构造器"
  - "校验失败透传 service 层中文 error → HTTP 400 + err.Error(),不 swallow,前端拿到明确错误直接弹 toast"

requirements-completed: [REQ-39-02, REQ-39-03]

# Metrics
duration: 18min
completed: 2026-06-25
---

# Phase 39 Plan 02: LocationAliasService + Handler + 三级校验

**工位部门↔物理位置映射 alias CRUD service/handler 三级校验 (自映射/外部机构/后代) 全链路落地 + 10 个单元测试守护契约**

## Performance

- **Duration:** 18 min
- **Started:** 2026-06-25T11:42:00Z
- **Completed:** 2026-06-25T12:00:00Z
- **Tasks:** 2
- **Files modified:** 3 (3 created + 0 modified)

## Accomplishments

- `LocationAliasService` 接口 (List/GetByID/Create/Update/Delete) + 私有 impl + 构造器,List 含 sys_dept 双向 JOIN 取原组织/物理位置 dept_name
- `validateAlias` 三级校验: 自映射拦截 / location 必须 is_external_org=1 / dept.ancestors 逗号包围匹配 location_id (避免 UUID 前缀子串误判)
- `LocationAliasHandler` 4 个端点 (List/Create/Update/Delete) + WithCore 链式注入 + operlog.Record 在 3 个写方法末尾统一调用,模块名 "工位管理"
- 10 个单元测试 (`TestValidateAlias_*` 4 个 + `TestLocationAliasService_*` 6 个) 全部通过,SQLite + AutoMigrate + 共享内存 DB + setup 清空数据避免污染
- 不引入新三方依赖;OperType 仅用 OperTypeCreate/Update/Delete (1/2/3),严格在 regression_test.go 守护范围内
- service 层 0 处引用 `ops:location:alias:` 权限字符串 (router 层职责,职责边界清晰)

## Task Commits

Each task was committed atomically:

1. **Task 1: LocationAliasService 接口 + 实现 + 三级校验 + 单测** - `a7f1421` (feat)
2. **Task 2: LocationAliasHandler 4 个端点 + operlog 集成** - `bc2b2c3` (feat)

## Files Created/Modified

- `internal/services/operations/location_alias_service.go` - Service interface + impl + validateAlias 三级校验,272 行
- `internal/services/operations/location_alias_service_test.go` - 10 个单测,SQLite + AutoMigrate,316 行
- `internal/api/v1/operations/location_alias_handler.go` - Handler 4 端点 + WithCore 链式注入,121 行

## Decisions Made

- **validateAlias 单函数集中三级校验**: Create 和 Update(dept_id/location_id 变更时) 共享同一函数,避免逻辑分散;所有错误信息含中文关键字供前端断言
- **scope 字段 service 层兜底 "workstation"**: D-04 决策,前端不传 scope 时 service 默认设值,与 migration 165 模型 default 一致
- **Update 路径按需触发 validateAlias**: 仅当 dept_id 或 location_id 实际变更才跑校验,纯 remark 改动不触发(性能 + 用户体验,避免无效查询)
- **id = ? 而非 id::text = ?**: 让 GORM 按 driver 自动选择比较方式;PG 端 uuid 列会自动转 text 比较,SQLite 端 TEXT 直接比对,与 `building_service.go validateOrg` 同款
- **逗号包围匹配 ancestors**: `","+ancestors+","+","+locationID+","` 模式杜绝 UUID 前缀子串误判 (例: ancestors="uuid-abc," locationID="uuid-ab" 不会命中)
- **Handler 链式 WithCore 注入**: 与 BuildingHandler/WorkstationHandler 同款,不改 NewXxxHandler 签名,router.go 调用点兼容
- **校验失败透传 service error**: D-02 决策,HTTP 400 + err.Error() 中文直白展示,不 swallow,前端拿到明确错误直接弹 toast

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] 移除 PG 专属 `id::text = ?` 语法以兼容 SQLite 测试**

- **Found during:** Task 1 测试运行
- **Issue:** 原计划使用 `id::text = ?` 强制 PG uuid→text 转换,但 SQLite 不识别 `::` 语法,测试全部失败报 "unrecognized token"
- **Fix:** 改为裸 `id = ?`,让 GORM 按 driver 自动适配 — PG 端自动转换(uuid→string),SQLite 端直接 TEXT 比对;与 `building_service.go validateOrg` 同款
- **Files modified:** `internal/services/operations/location_alias_service.go`
- **Verification:** 10 个测试全部通过;PG 生产 DB 行为对齐 `building_service.go`(同项目已验证)
- **Committed in:** `a7f1421` (Task 1 commit)

**2. [Rule 1 - Bug] setup 函数清空共享内存 DB 避免测试间污染**

- **Found during:** Task 1 测试运行 (`TestLocationAliasService_List` total 期望 3 实际 4)
- **Issue:** `file::memory:?cache=shared` 让所有 gorm.Open 共享同一份 SQLite 内存 DB,前序测试插入的数据残留导致 count 偏差
- **Fix:** setup 函数末尾 TRUNCATE `sys_dept_location_alias` (软删除) + DELETE FROM `sys_dept`,确保每个测试从干净状态开始
- **Files modified:** `internal/services/operations/location_alias_service_test.go`
- **Verification:** 所有 10 个测试通过;前后两次运行无顺序依赖
- **Committed in:** `a7f1421` (Task 1 commit)

**3. [Rule 3 - Blocking] 移除自定义 scanInt 工具函数**

- **Found during:** Task 2 编写 handler 时
- **Issue:** 最初为简化 pageNum 解析写了 `scanInt` + `aliasIntError` 自定义类型,但项目已用 `strconv.Atoi` 标准库,引入自定义反而冗余
- **Fix:** 改用 `strconv.Atoi(v)` 标准库,删除自定义函数和 error 类型,代码更简洁且零自定义类型泄露
- **Files modified:** `internal/api/v1/operations/location_alias_handler.go`
- **Verification:** `go build ./...` 0 退出码;`go vet ./internal/api/v1/operations/...` 0 告警
- **Committed in:** `bc2b2c3` (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (2 bugs + 1 blocking)
**Impact on plan:** 全部是实现细节微调,无功能性偏差;PG/SQLite 双 DB 兼容性更稳健,测试隔离更可靠

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Service/Handler 层就绪,Plan 39-04 (router 注册 + 缓存失效 + operlog 端到端贯通) 可直接接入
- 4 个新端点待注册:`POST /ops/location-alias/list`、`POST /ops/location-alias`、`POST /ops/location-alias/:id/update`、`POST /ops/location-alias/:id/delete`
- 4 条按钮权限 `ops:location:alias:{list,add,edit,delete}`(Plan 39-01 seed) 待与路由绑定
- Plan 39-03 (工位部门下拉 union) 可在工位导入/查询 service 中 LEFT JOIN `sys_dept_location_alias` 表,通过 `LocationAliasService.List()` 拿到映射列表后 union 到部门下拉
- 校验契约有 10 个单测守护,后续修改 validateAlias 必须同步维护测试

---
*Phase: 39-workstation-dept-location-alias*
*Completed: 2026-06-25*

## Self-Check: PASSED

- Files created: `internal/services/operations/location_alias_service.go`, `internal/services/operations/location_alias_service_test.go`, `internal/api/v1/operations/location_alias_handler.go` (all exist)
- Commits verified: `a7f1421` (Task 1), `bc2b2c3` (Task 2) — both present in `git log --oneline`
- `go build ./...` exit 0
- `go test ./internal/services/operations/ -run "TestValidateAlias|TestLocationAliasService" -v -count=1` 10/10 通过
- operlog 模块名 "工位管理" 出现在 3 处(Comment + RecordCreate + RecordUpdate + RecordDelete)
- 0 处引用 `ops:location:alias:` 权限字符串 (grep verified)
- OperType 仅用 Create/Update/Delete (1/2/3),均在 regression_test.go 守护范围内
