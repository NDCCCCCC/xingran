---
phase: 39-workstation-dept-location-alias
plan: 05
subsystem: frontend
tags: [react, typescript, react-query, tanstack-query, workstation, ops-api]

# Dependency graph
requires:
  - phase: 39-03
    provides: "POST /ops/workstation/dept-options 后端 union 端点 + DeptOption JSON 契约 (deptId/deptName/isAlias)"
  - phase: 39-04
    provides: "POST /ops/location-alias/{list,,/update,/delete} CRUD 端点"
provides:
  - locationAliasApi (list/create/update/delete) — Plan 39-07 Drawer UI 消费
  - workstationApi.deptOptions(orgId) — Plan 39-06 EditModal 部门下拉 union 注入
  - DeptOption / LocationAlias TypeScript 接口
  - queryKeys.locationAlias 段 (all/byLocation/list)
  - useAliasByLocation(locationId) React Query hook
affects:
  - 39-06 (EditModal.subDeptTree 消费 useAliasByLocation + workstationApi.deptOptions)
  - 39-07 (Drawer 管理 UI 消费 locationAliasApi + invalidate locationAlias 缓存)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - React Query hook + queryKey 工厂 + opsApi 三件套模式 (queryKeys ↔ opsApi ↔ hooks 三文件解耦)
    - hook enabled 守卫: locationId 空时不发请求(与 useDeptTree/useTableQuery 一致)
    - queryKey 工厂末尾 append 段,不修改既有段 (零回归风险)
    - JSON tag 与后端 struct 1:1 对齐 (deptId/deptName/isAlias ↔ DeptID/DeptName/IsAlias)
    - scope 默认值前端兜底 ("workstation"),后端也会兜底 (双兜底防御)

key-files:
  created:
    - xingran-react-frontend/src/hooks/useAliasByLocation.ts
  modified:
    - xingran-react-frontend/src/lib/opsApi.ts
    - xingran-react-frontend/src/lib/queryKeys.ts

key-decisions:
  - "post<DeptOption[]> 而非 post<{data?: DeptOption[]}>: 后端 response.Success(c, result) 中 data 直接是 []DeptOption, 前端 res.data 即数组, 避免双重 .data.data 取值"
  - "locationAliasApi.list 用 PageResponse<LocationAlias> 而非内联 PageData 字面量: 复用 @/types 已有的分页响应类型, 与项目其他 list 端点风格一致"
  - "useAliasByLocation 接受 string | undefined | null 三态: 兼容 antd Form.Select onChange 可能传 null 的场景, enabled 守卫统一用 !!locationId"
  - "staleTime/gcTime 与 useDeptTree 对齐 (5min/30min): alias 数据变更频率低, 长缓存命中率高; 写操作后由调用方显式 invalidate"

patterns-established:
  - "下拉数据源 hook 模式: useXxxByYyy(id) → post<T[]>(endpoint, {id}) → res.data ?? [], enabled=!!id"
  - "alias 模块前端命名空间 locationAlias (与后端路由 /ops/location-alias 对齐, 不用 alias 短名避免歧义)"

requirements-completed: [REQ-39-05, REQ-39-07]

# Metrics
duration: 8min
completed: 2026-06-25
---

# Phase 39 Plan 05: 前端 opsApi + queryKeys + useAliasByLocation 三件套 Summary

**前端 alias 数据获取基础设施落地: locationAliasApi (CRUD 4 方法) + workstationApi.deptOptions (union 数据源) + queryKeys.locationAlias 段 + useAliasByLocation React Query hook, JSON tag 与后端 DeptOption/LocationAlias 1:1 对齐, 沿用 post helper 不引入新依赖**

## Performance

- **Duration:** 8 min
- **Tasks:** 2
- **Files:** 3 (1 created + 2 modified)

## Accomplishments

- `DeptOption` TypeScript 接口 (deptId/deptName/isAlias) — 与后端 Plan 39-03 `DeptOption` struct JSON tag 完全对齐, 供 isAlias=true 时追加 `[映射]` 后缀
- `LocationAlias` TypeScript 接口 (id/deptId/locationId/scope/remark/createdAt/updatedAt) — Plan 39-07 Drawer UI 的 CRUD 数据契约
- `workstationApi.deptOptions(orgId): Promise<DeptOption[]>` — 工位编辑"所属部门"下拉 union 数据源, 调用 Plan 39-03 的 `POST /ops/workstation/dept-options` 端点
- `locationAliasApi` 对象 (list/create/update/delete 4 方法) — 调用 Plan 39-04 的 `POST /ops/location-alias/*` 端点群, create 默认 `scope="workstation"` 前端兜底
- `queryKeys.locationAlias` 段 (all/byLocation/list 三工厂) — 末尾 append, 现有 dict/list/dept/duty/role 段零修改
- `useAliasByLocation(locationId)` React Query hook — locationId 空时 `enabled=false` 不发请求, staleTime/gcTime 与 useDeptTree 对齐 (5min/30min)
- `cd xingran-react-frontend && npx tsc --noEmit` 退出码 0
- 0 引入新三方依赖 (沿用现有 axios / @tanstack/react-query / post helper)

## Task Commits

Each task was committed atomically:

1. **Task 1: locationAliasApi 客户端 + workstationApi.deptOptions 扩展** — `262ea12` (feat)
2. **Task 2: queryKeys.locationAlias + useAliasByLocation hook** — `cea4d4a` (feat)

## Files Created/Modified

- `xingran-react-frontend/src/lib/opsApi.ts` (+53 行) — 新增 DeptOption 接口 + LocationAlias 接口 + workstationApi.deptOptions 方法 + locationAliasApi 对象
- `xingran-react-frontend/src/lib/queryKeys.ts` (+6 行) — queryKeys 对象末尾追加 locationAlias 段
- `xingran-react-frontend/src/hooks/useAliasByLocation.ts` (新建, +34 行) — useAliasByLocation React Query hook + 完整 JSDoc

## Decisions Made

- **post<DeptOption[]> 而非 post<{data?: DeptOption[]}>**: 后端 `response.Success(c, result)` 把 `[]DeptOption` 放进 response.data, 前端 `post<T>` 已是 `Promise<BaseResponse<T>>`, 所以 `res.data` 直接是数组. 计划文档原写法会变成 `res.data.data` 双重取值, 实际改为更简洁的 `post<DeptOption[]>` (Rule 3 阻塞修复: 计划字面值不可用).
- **locationAliasApi.list 用 `PageResponse<LocationAlias>`**: 复用 `@/types` 已有分页响应类型, 与 buildingApi/floorApi 等其他 list 端点风格一致, 避免重复定义 PageData 字面量.
- **useAliasByLocation 接受 `string | undefined | null` 三态**: antd Form.Select `onChange` 在清空时可能传 `null`, 三态签名直接兼容, `enabled` 守卫统一用 `!!locationId`.
- **staleTime/gcTime 与 useDeptTree 对齐**: alias 数据变更频率低 (Drawer CRUD 才触发), 5min/30min 长缓存命中率高; 写操作后由调用方 (Plan 39-07 Drawer) 显式 `invalidateQueries(queryKeys.locationAlias.all)` 兜底.
- **queryKeys 段末尾 append 而非中间插入**: 现有 dict/list/dept/duty/role 段零修改, 零回归风险; `as const` 元组类型自动延伸.
- **scope 前端兜底 = "workstation"**: 与后端 Plan 39-02 service 层 `defaultScope` 双兜底, 即使前端漏传也能落到正确场景.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 修正 deptOptions 返回类型双重 .data 取值**
- **Found during:** Task 1
- **Issue:** 计划原文写 `post<{ data?: DeptOption[] }>` 然后取 `res.data`, 但后端 `response.Success(c, result)` 已把 `[]DeptOption` 放进 response.data, 加上 `post<T>` 返回 `Promise<BaseResponse<T>>`, 实际取值路径应是 `res.data` (单层), 字面值类型会让 TypeScript 把 `DeptOption[]` 当作 `{ data?: DeptOption[] }` 的形状, 语义错误.
- **Fix:** 改为 `post<DeptOption[]>` + `res.data ?? []`, 类型参数直接是数组元素类型, 取值路径单层.
- **Files modified:** `xingran-react-frontend/src/lib/opsApi.ts`
- **Commit:** `262ea12`

**2. [Rule 3 - Blocking] locationAliasApi.list 用 PageResponse<LocationAlias> 而非内联字面量**
- **Found during:** Task 1
- **Issue:** 计划原文写内联 `{ data: { list: LocationAlias[]; total; current; pageSize } }` 字面量类型, 与项目 `@/types` 的 `PageResponse<T>` 重复定义, 会造成类型漂移.
- **Fix:** 改用 `PageResponse<LocationAlias>` (已在文件顶部 import 自 `@/types`), 与 buildingApi/floorApi 等其他 list 端点风格一致.
- **Files modified:** `xingran-react-frontend/src/lib/opsApi.ts`
- **Commit:** `262ea12`

## Out-of-Scope Discoveries (Logged, NOT Fixed)

详见 `deferred-items.md`. 摘要:

- `opsApi.ts:281/298` ESLint warning `no-unsafe-assignment` — 在 pre-existing `extractFilenameFromBlobResponse` / `triggerBrowserDownload` 中
- `opsApi.ts:491` ESLint error `no-unused-vars` (`maxAge` in `geocodeWithCache`) — pre-existing

这些 issue 全部在 Plan 39-05 之前就存在, NOT introduced by this plan. 按执行器 scope boundary 规则不修复, 留待独立清理工单.

## Issues Encountered

None

## User Setup Required

None — 无外部服务配置. 本 plan 只触碰前端文件, 后端端点 (Plan 39-03/39-04) 已就绪.

## Next Phase Readiness

- **Plan 39-06 (EditModal.subDeptTree 注入)** 可直接 `import { useAliasByLocation } from "@/hooks/useAliasByLocation"`, 把 union 结果按 `isAlias=true` 追加 `[映射]` 后缀渲染
- **Plan 39-07 (Drawer 管理 UI)** 可直接 `import { locationAliasApi } from "@/lib/opsApi"`, 调用 list/create/update/delete 4 方法, 写操作成功后 `invalidateQueries(queryKeys.locationAlias.all)` + `queryKeys.dept.all` 双失效
- **端到端贯通测试 (39-04 已完成)** 可通过 `workstationApi.deptOptions("xxx-uuid")` 验证后端 `/ops/workstation/dept-options` 连通性
- **缓存策略一致性**: locationAlias 段与 dept 段命名风格一致 (all + 具体工厂), 未来扩展 (例如 byDept) 可顺势追加

## Self-Check: PASSED

- 文件创建: `xingran-react-frontend/src/hooks/useAliasByLocation.ts` — FOUND
- 文件修改: `xingran-react-frontend/src/lib/opsApi.ts`, `xingran-react-frontend/src/lib/queryKeys.ts` — both FOUND
- 提交验证: `262ea12` (Task 1) + `cea4d4a` (Task 2) — both present in `git log --oneline`
- `cd xingran-react-frontend && npx tsc --noEmit` 退出码 0
- `queryKeys.locationAlias` 段 grep 命中
- `workstationApi.deptOptions` 方法 grep 命中
- `locationAliasApi` 对象 grep 命中
- `useAliasByLocation` 导出 grep 命中
- 现有 queryKeys 段 (dict/list/dept/duty/role) 0 修改
- 现有 hooks (useDeptTree 等) 0 修改

---
*Phase: 39-workstation-dept-location-alias*
*Completed: 2026-06-25*
