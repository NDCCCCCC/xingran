---
phase: 39-workstation-dept-location-alias
plan: 03
subsystem: api
tags: [go, gin, gorm, postgres, workstation, sys_dept]

# Dependency graph
requires:
  - phase: 39-01
    provides: "sys_dept_location_alias 表 + scope='workstation' 过滤维度"
provides:
  - WorkstationService.GetWorkstationDeptOptions(ctx, orgId) 单 query union 数据源
  - DeptOption struct (deptId/deptName/isAlias) 供前端 EditModal 渲染 [映射] 后缀
  - POST /ops/workstation/dept-options 端点 (沿用 ops:workstation:* 权限)
affects:
  - 39-06 (EditModal.subDeptTree 消费 DeptOption[] + 按 isAlias 追加后缀)
  - 39-04 (端到端贯通测试可覆盖 /dept-options 路由)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - 单 Raw query UNION ALL 完成 orgId 子孙 + alias 映射 union (D-06 决策)
    - ancestors 逗号包围字符串匹配 (",ancestors," 包夹 orgId) 防 UUID 前缀子串误判 (与 39-02 validateAlias 一致)
    - is_external_org=0 过滤排除外部机构本身出现在子树
    - handler /list 之后 /statistics 之前注册具体路径 /dept-options (具体路径优先匹配 Gin)

key-files:
  created: []
  modified:
    - internal/services/operations/workstation_service.go
    - internal/api/v1/operations/workstation_handler.go
    - internal/api/router.go

key-decisions:
  - "单 query union (D-06): 一条 Raw SQL UNION ALL 同时取子孙节点 + alias 节点,避免 service 层多次往返 + 内存拼接"
  - "UNION ALL 而非 UNION: alias 节点本就唯一(dept_id+location_id+scope 唯一索引),去重交给前端;保留顺序"
  - "is_alias 标记透传前端: DeptOption.IsAlias → 前端按 true 追加 [映射] 后缀"
  - "不修改 workstationJoinClause (D-06 锁定): 新方法独立 Raw query,与 List/Statistics 的 Joins 链隔离"
  - "沿用既有 ops:workstation:* 中间件,不引入新权限字符串: 端点是工位列表的下拉数据源"

patterns-established:
  - "下拉数据源端点模式: GET 语义用 POST + body {orgId},与 Statistics 同款,不引入新权限"
  - "service 单 Raw union query 模式: 单次 DB 往返取 union 数据,避免 N+1"

requirements-completed: [REQ-39-05]

# Metrics
duration: 10min
completed: 2026-06-25
---

# Phase 39 Plan 03: WorkstationService.GetWorkstationDeptOptions 单 query union

**工位"所属部门"下拉 union 数据源落地: WorkstationService 新增 GetWorkstationDeptOptions 单 Raw query (UNION ALL) 同时取 orgId 子孙节点 + alias 映射节点 (含 is_alias 标记) + handler 端点 + router 注册 1 行,沿用既有权限中间件,workstationJoinClause 零修改**

## Performance

- **Duration:** 10 min
- **Tasks:** 2
- **Files modified:** 3 (0 created + 3 modified)

## Accomplishments

- `DeptOption` struct (deptId/deptName/isAlias) — 工位编辑"所属部门"下拉选项契约,`isAlias` 标记供前端追加 `[映射]` 后缀
- `WorkstationService` 接口新增 `GetWorkstationDeptOptions(ctx, orgId)` 方法 + 私有 impl
- 单 Raw query UNION ALL 实现 D-06 决策: 第一段 `sys_dept` 子孙节点 (ancestors 逗号包围匹配 OR id=orgId,排除 is_external_org=1),第二段 `sys_dept_location_alias a JOIN sys_dept d` (scope='workstation' AND location_id=orgId)
- `WorkstationHandler.GetWorkstationDeptOptions` handler: 读 body.orgId → service → response.Success
- `POST /ops/workstation/dept-options` 路由注册 (router.go `/list` 之后 `/statistics` 之前,具体路径优先匹配)
- `workstationJoinClause` 常量零修改 (D-06 锁定,已 grep 验证)
- 沿用既有 `ops:workstation:list/add/edit/delete` 权限中间件 + `ops:building:spaces:list` 查询旁路,不引入新权限字符串
- `go build ./...` 退出码 0

## Task Commits

Each task was committed atomically:

1. **Task 1: WorkstationService.GetWorkstationDeptOptions 单 query union** - `a7b9e40` (feat)
2. **Task 2: handler 端点 + router 注册 /ops/workstation/dept-options** - `05ca92f` (feat)

## Files Created/Modified

- `internal/services/operations/workstation_service.go` - 接口新增方法 + DeptOption struct + 单 query union impl (+36 行)
- `internal/api/v1/operations/workstation_handler.go` - GetWorkstationDeptOptions handler 方法 (+26 行)
- `internal/api/router.go` - workstations 路由组插入 `/dept-options` 1 行

## Decisions Made

- **单 query union (D-06)**: 一条 Raw SQL UNION ALL 同时取子孙节点 + alias 节点,避免 service 层多次 DB 往返 + 内存拼接;`is_alias` 布尔列直接在 SQL 中产出 (false/true) 供前端区分
- **UNION ALL 而非 UNION**: alias 节点在表上由 (dept_id, location_id, scope) 唯一索引保证唯一,去重交给前端;UNION ALL 保留顺序且省去内部排序
- **ancestors 逗号包围匹配**: `(',' || ancestors || ',') LIKE ('%,' || ? || ',%')` 杜绝 UUID 前缀子串误判 (例: ancestors 含 `uuid-abc,` 不会误匹配 orgId=`uuid-ab`);与 39-02 `validateAlias` 同款逻辑
- **is_external_org=0 过滤**: 第一段 query 排除外部机构本身,避免 location 节点出现在其子树里 (location 是物理位置容器,不应进入部门下拉)
- **第二段 JOIN sys_dept**: `sys_dept_location_alias` 表只存 dept_id 不存 dept_name,JOIN 取 dept_name 供前端展示
- **handler orgId 容错**: `ShouldBindJSON` 失败时 `req.OrgID = ""` 兜底,service 层 `orgId == ""` 直接返回空数组,避免 panic
- **router 注册位置**: `/list` 之后 `/statistics` 之前插入 `/dept-options`,与现有"具体路径优先"排列风格一致;Gin 路由树无冲突 (静态段不与 `/:id` 重叠)

## Deviations from Plan

None - plan executed exactly as written.

- Task 1 的代码已在工作树由前序 executor 写好 (与计划 SQL 完全一致),本次只做 grep/build 验证 + commit,未重写
- Task 2 按计划插入 handler 方法和 router 1 行,0 偏差

## Issues Encountered

None

## User Setup Required

None - no external service configuration required. 新端点沿用既有权限,无需配置。

## Next Phase Readiness

- 后端 union 数据源就绪,Plan 39-06 (EditModal 前端) 可直接 `POST /ops/workstation/dept-options {orgId}` 拿 DeptOption[]
- 前端按 `isAlias=true` 追加 `[映射]` 后缀;按 `deptName` 用 `trimTitleToLastSegment` 进一步收窄
- Plan 39-04 (端到端贯通测试) 可覆盖 `/dept-options` 路由 + 权限校验
- `LocationAliasService` CRUD (Plan 39-02) 是 alias 表的写入路径,本 plan 只读不写,两者通过 `sys_dept_location_alias` 表解耦
- `workstationJoinClause` 零修改,既有 List/Statistics 行为不变 (build 通过 + grep 验证)

---
*Phase: 39-workstation-dept-location-alias*
*Completed: 2026-06-25*

## Self-Check: PASSED

- Files modified: `internal/services/operations/workstation_service.go`, `internal/api/v1/operations/workstation_handler.go`, `internal/api/router.go` (all exist)
- Commits verified: `a7b9e40` (Task 1), `05ca92f` (Task 2) — both present in `git log --oneline`
- `go build ./...` exit 0
- `grep -n "workstationJoinClause" internal/services/operations/workstation_service.go` 行 17 内容未变 (D-06 锁定)
- `grep -n "/dept-options" internal/api/router.go` 命中行 594
- 0 处引入新权限字符串 (沿用既有 ops:workstation:*)
