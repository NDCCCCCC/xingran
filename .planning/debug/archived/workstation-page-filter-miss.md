---
gsd_state_version: 1.0
slug: workstation-page-filter-miss
status: diagnosed
trigger: "工位管理页面，所属工位下拉框有limit限制导致部分工位无法查找，请检查并提出最佳实践解决方案"
created: 2026-06-29
updated: 2026-06-29
goal: find_root_cause_only
---

## Evidence

- **2026-06-29 — 现场定位：「所属工位」下拉框位于「信息点管理」页面（`info-points/index.tsx`），而非工位管理页面**：
  `xingran-react-frontend/src/pages/operations/info-points/index.tsx:641` `Form.Item name="workstationId" label="所属工位"`。
  `xingran-react-frontend/src/pages/operations/workstations/index.tsx:678-697` 工位管理页面筛选区只有「所属楼层 / 工位名称 / 状态」三个字段，无「所属工位」下拉。用户描述的「工位管理页面」实际指代该下拉所在的信息点管理页面（或口语化把工位相关页面统称）。下拉绑定的 `workstationId` 即用于按工位筛选信息点。
- **2026-06-29 — 数据源：`workstationApi.list({ current: 1, pageSize: 1000 })`**，调用位置 `xingran-react-frontend/src/pages/operations/info-points/index.tsx:244-248`（`loadSearchWorkstationOptions` 内部）。
- **2026-06-29 — 后端无截断**：`internal/services/operations/workstation_service.go:234-303` 的 `List` 走 `internal/services/operations/pagination_helper.go:35-44` `extractPagination` → `clampPageSize`(`MaxPageSize=10000`, `pagination_helper.go:17`)；`pageSize:1000` 被原样接受。所以 **不是后端 limit 把工位截断**——工位表 >1000 行时，前端从后端拿到的就是前 1000 条（GORM `Offset(0).Limit(1000)`），后端无报错、无 `total` 校验。
- **2026-06-29 — 客户端过滤：`info-points/index.tsx:652-655` `filterOption` 在已加载的 1000 条本地数组里做 substring 匹配**，未命中即「无匹配」；Select 没有 `onSearch`，未走远程搜索路径。
- **2026-06-29 — 缓存失效陷阱：`info-points/index.tsx:238-241` `loadSearchWorkstationOptions` 头部有 `if (workstationOptions.length > 0) return;` 早返回**，首次打开下拉触发 `onOpenChange` 后（line 649）拉一次即被「永久」缓存；切换左侧 DeptSidebar 改 `selectedDeptId` 才会重新拉取（`useCallback` 依赖 `workstationOptions.length, selectedDeptId`，line 261），但同部门下 1000 条后续永远不变。这与 `useWorkstationData.ts:118-140` `loadUserOptions` 已实现的「按 total 分页循环 + MAX_PAGES 上限」模式形成强烈对比（user 模块走 system `MaxPageSize=100`，已踩过同款坑并修复）。
- **2026-06-29 — 顺带定位：另一处同款反模式 `loadFloorOptions`**，`xingran-react-frontend/src/pages/operations/workstations/hooks/useWorkstationData.ts:67-82`：`floorApi.list({ current: 1, pageSize: 1000 })`，同样无分页循环；超过 1000 楼层时同样会被截断。但目前产品规模楼层 < 1000，暂未触发。
- **2026-06-29 — 历史教训：`stat-cards-from-list-length-capped-at-100.md` 已经把 system 模块 `MaxPageSize=100` 这层钳制总结清楚，并推广到 9 类模块的统计卡片**。本次问题位于 operations 模块（`MaxPageSize=10000`），不在历史教训直接覆盖范围内——但「把分页 list 接口当全集拉」这个反模式是同类，且本次的危害面更广（影响用户筛选体验而非只是统计数字）。

## Eliminated

- hypothesis: 后端 limit（如 100 / 1000）截断导致下拉选项不全
  - evidence: operations 模块 `MaxPageSize=10000`（`internal/services/operations/pagination_helper.go:17`），前端传 `pageSize:1000` 完全被接受；后端返回完整前 1000 条 + 正确 `total`。截断发生在「前端用 List 接口当下拉全集」这个架构决策，而不是后端校验。
  - timestamp: 2026-06-29
- hypothesis: system 模块 `MaxPageSize=100` 钳制（user/notice 统计卡片历史问题同款）
  - evidence: 工位 list 走 operations 模块 helper，不走 system 模块常量（`internal/constants/pagination.go:20` 的 100 上限对 operations 不生效）。
  - timestamp: 2026-06-29

## Resolution

- **root_cause**: 「所属工位」筛选下拉（位于信息点管理页面）用 `workstationApi.list(pageSize:1000)` 一次性拉取前 1000 条工位当 Select 选项，前端 `filterOption` 在这 1000 条本地数据上做客户端 substring 匹配，未命中即「无匹配」。后端无截断，但架构上把「分页 List 接口」当成「下拉全集源」使用——超过 1000 工位后（典型规模≥1栋楼全量楼层），第 1001 条起的工位永远无法通过下拉搜到。
- **fix**: 待定（仅诊断，本次不修改代码）
- **verification**: 待定
- **files_changed**: []

### 推荐方案（Option A — 独立轻量端点 + 远程搜索）

**后端**：新增 `POST /ops/workstation/dropdown-options` 专用端点，请求参数 `keyword?`，返回字段仅 `id/name/buildingName/floorName/status`（不返回工位地址、坐标、备注等冗余字段）。底层走 `WorkstationService` 的查询逻辑，但跳过 pagination + 排序 + 全表扫描，只走 `WHERE name ILIKE '%kw%' OR no ILIKE '%kw%' LIMIT 50`。

**前端**：`info-points/index.tsx` 的 Select 组件改为
- `showSearch` + `onSearch` 远程搜索，debounce 250ms
- 选项渲染用 `labelInValue` 模式，label = `${name} (${floorName})`
- 首次打开下拉时立刻触发 `onSearch('')` 拉一次默认值（最多 50 条）
- 输入关键字后远程查询；选中后写入表单 `workstationId`

**预估工作量**：后端 ~80 行（含路由、handler、service 方法、DTO），前端 ~40 行。合计 ~1 人日。

**优势**：
- 不依赖「前端拉全量」的隐式上限，可扩展到任意工位规模
- 带宽/内存最优（每次请求 ~50 条，仅返回必要字段）
- 与 `stat-cards-from-list-length-capped-at-100.md` 历史教训原则一致：专用端点 > 拉全量前端算
- 易于加 Redis 缓存层（5 分钟 TTL，keyword → options）

### 备选方案

**Option B — 前端分页循环拉全量 + 客户端过滤**（仿 `loadUserOptions`）
- 改动量：前端 ~30 行，0.5 人日
- 缺点：`MaxPageSize=10000` 仍是隐式上限；首屏慢；Select vDOM 节点过多卡顿；与 `user_options` 模块同款历史问题已暴露

**Option C — Hybrid：首屏全量 + 输入远程**
- 改动量：前端 ~80 行，0.7 人日
- 缺点：交互逻辑复杂（首屏/搜索两套路径）；本地上限问题未根除

### 横向治理建议

1. 同步修复 `workstations/hooks/useWorkstationData.ts:67-82` 的 `loadFloorOptions`（同款反模式潜伏项，目前楼层 < 1000 暂未触发）
2. 在 ESLint 规则或前端代码审查 checklist 中加入「禁止在 Select/AutoComplete 组件中调用 *.list() 接口当数据源」的 lint 规则
3. 后端 helper 层（如 `pagination_helper.go`）提供 `WithoutPagination()` 显式开关，与 pagination 走不同 SQL 路径，避免「拉全量」语义被 pageSize=10000 隐式表达