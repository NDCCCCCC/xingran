# Phase 14: 前端与用户体验 - Context

**Gathered:** 2026-06-14
**Status:** Ready for planning
**Source:** `gsd:discuss-phase 14` (用户主动澄清 5 个 gray area, GA-06 归 deferred)

<domain>
## Phase Boundary

基于 Phase 12 (数据模型与采集集成) 与 Phase 13 (查询层与轨迹) 的后端能力,补齐 MAC 地址历史数据管理的完整前端 UX — 包括查询列表页、轨迹可视化页 UX 增强、Excel 导出、菜单与权限注册、与网络设备模块的联动入口,以及移动端响应式适配与状态兜底。

**Requirements (锁定, 来自 `.planning/REQUIREMENTS.md` v1.5)**:
- **UI-01**: 实现 MAC 历史查询页面(时间筛选、MAC/设备/端口筛选、分页、操作列)
- **UI-02**: 实现数据导出功能(Excel)
- **UI-04**: 实现历史事件时间线组件

**Out of scope (Phase 13 已交付, Phase 14 集成不重写)**:
- **UI-03**: MAC 轨迹可视化 ECharts Gantt — Phase 13-04 已交付 `MACTrajectoryChart.tsx`, Phase 14 复用

**5 个预期子计划 (ROADMAP.md 锁定,本 phase 不再切分)**:
- 14-01: MAC 历史查询主列表页(虚拟滚动 + 快捷时间预设)
- 14-02: 轨迹可视化页 UX 增强(ECharts Gantt 交互、停留时长热力、时间范围预设)
- 14-03: 菜单/权限/路由注册(`network:mac:list/query/export`, 与网络设备并列)
- 14-04: Excel 导出与批量操作(后端 `format=xlsx` 返回 + 前端触发)
- 14-05: 移动端响应式 + 空/加载/错误三态打磨

**Phase Highlights (来自 ROADMAP.md)**:
- 复用 Phase 13 的 `MACTrajectoryChart.tsx`,不重写可视化核心
- 时间范围筛选复用 `useColumnConfig` 与 `useTableManager` hooks
- 列表页与轨迹页共享 MAC 输入,实现"查询→可视化"单步跳转
- 菜单注册与网络设备模块并列,权限点 `network:mac:list/query/export`
- 移动端使用 AntD 6 Grid + 卡片视图,继承 `HybridLayout` 侧边栏折叠行为
- 严格遵守 React Query 缓存键规范,避免重复请求(参考 Phase 30/33 经验)

</domain>

<decisions>
## Implementation Decisions

### 来自前序 Phase 继承的锁定决策 (D-01..D-05)

#### D-01: Phase 14 = 纯前端 phase
后端 API 全部就位 (Phase 12 + 13),本阶段**不动后端代码**,除 14-04 复用现有 `/network/history/list` 接口,通过 `format=xlsx` query 参数返回二进制流(后端需在现有 handler 增加 format 分支,属于"现有 endpoint 增强",不改 API 数量)。

#### D-02: 复用 Phase 13 已交付资产
- `xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx` — 不重写
- `xingran-react-frontend/src/pages/network/mac/trajectory.tsx` — 集成增强而非新建

#### D-03: 复用既有前端架构模式 (来自 Phase 30/33 + CLAUDE.md)
- 状态管理: Zustand (authStore, layoutStore 等)
- 数据获取: React Query 5(Phase 30/33 推广后标准)
- UI 库: Ant Design 6
- 列表管理: `useTableManager`, `useColumnConfig` hooks(虚拟滚动用 `placeholderData` 保持滚动位置)
- API 调用: `lib/api.ts` 包装函数(非 raw axios);Excel 下载走 `api.get(url, { responseType: 'blob' })`(Phase 33 M2 经验)
- 实时更新: `useRealtimeUpdates`, `useWidgetPolling` hooks (按需)
- 性能: 路由 `React.lazy` + Suspense 包裹(Phase 30 D-08),vendor chunk 策略保持

#### D-04: 菜单与权限规范
- 权限点: `network:mac:list`, `network:mac:query`, `network:mac:export`
- 父菜单: `network` (与"网络设备"模块并列展示)
- 复用 Phase 13-06 已交付的 `13-06-menu-registration-v4.sql` 模式(若已含 MAC 菜单则跳过,否则新增)

#### D-05: 移动端策略
- 桌面端: 表格视图(虚拟滚动)
- 移动端: 卡片视图(参考 `operations/buildings` 既有模式)
- 响应式断点: Ant Design 6 默认 (xs/sm/md/lg/xl)
- 侧边栏折叠: 继承 `HybridLayout` 行为

### GA-01 列表页查询模式(用户澄清)

#### D-06: 主列表分页 = 虚拟滚动
- 实现: 复用 `useTableManager` + AntD Table `virtual` 属性 + `useQuery` 的 `placeholderData: keepPreviousData`(React Query 5)
- 理由: 与 Phase 30 资产列表(43 列)策略一致,大数据量下用户体验更连贯
- 一次拉取上限: 后端分页 size=100,前端在表内做虚拟渲染;用户滚动到底部触发"加载更多"

#### D-07: 时间范围筛选 = 快捷预设 + RangePicker
- 预设按钮(单击切换): `近 1h` / `近 24h` / `近 7d` / `近 30d` / `近 90d` / `自定义`
- 自定义模式: AntD `RangePicker` (showTime 开启)
- 互斥: 选预设后 RangePicker 同步显示;选 RangePicker 后预设按钮清空(用户自定义)
- 默认: 进入页面时默认 `近 7d`(与 Phase 13 轨迹页一致)

#### D-08: 列结构 = 全列展示
- 全部 8 列: 时间 / MAC / 设备 / 端口 / 事件类型 / VLAN / 状态 / 操作
- 使用 `useColumnConfig`(Phase 27 资产列表同模式)
- 不分组(主+详情),用户通过 `useColumnConfig` 自行隐藏

### GA-02 时间线组件形态(用户澄清)

#### D-09: 独立跨页复用组件
- 文件: `xingran-react-frontend/src/components/network/MACEventsTimeline.tsx`
- 可被消费方: 查询页"展开行" / 轨迹页"右侧侧边栏" / 设备详情页"该设备 MAC 事件"
- 行为: 接收 `mac` + `timeRange` props,内部用 React Query 拉取 `/network/history/list?mac=...&startTime=...&endTime=...`

#### D-10: 垂直时间线 (AntD Timeline 风格)
- 视觉: 垂直从上到下,事件从新到旧
- 事件类型颜色与图标(与 MACTrajectoryChart 颜色体系一致):
  - `appeared` → 绿色 #52c41a + `PlusCircleOutlined`
  - `moved` → 黄色 #faad14 + `SwapOutlined`
  - `disappeared` → 红色 #ff4d4f + `MinusCircleOutlined`
  - `vlan_changed` → 蓝色 #1890ff + `TagOutlined`
- 每行内容: 时间 | 事件类型标签 | 设备 + 端口 | VLAN

#### D-11: 事件可点击跳轨迹页
- 点击时间线中的事件项 → 跳 `/network/mac/trajectory?mac=...&startTime=...&endTime=...&deviceId=...`
- 轨迹页打开后自动用这些参数填充查询条件(轨迹页 Phase 13-04 已有 queryParams 注入逻辑,扩展支持)
- 不在同页用 Tabs 切换(避免与轨迹页 Gantt 主视图争抢布局)

### GA-03 导出策略(用户澄清)

#### D-12: 格式 = 仅 Excel (xlsx)
- 后端复用 `xuri/excelize/v2`(已就位)
- 与所有其他模块(资产/工位/设备)导出保持一致格式

#### D-13: 范围 = 仅当前查询条件 + 全量导出
- 工具栏按钮(互斥):
  - **导出当前查询**: 按当前过滤条件(时间/MAC/设备/端口)导出当前页+后续分页(后端流式合并为单个 .xlsx)
  - **导出全量**: 无过滤条件导出全部历史(异步任务或后端流式响应,量大时显示进度)
- 不在操作列加"导出该 MAC"按钮(避免 UI 重复,导出当前查询可覆盖该需求)

#### D-14: 后端接口 = 复用 `/network/history/list` + `format=xlsx` query 参数
- 不新增 API 端点(避免接口膨胀)
- 后端在现有 handler 增加分支: 收到 `format=xlsx` 时,直接生成 Excel 流并返回 `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- "导出全量" 使用同一端点 + 显式 `format=xlsx&exportScope=all` 参数;后端流式响应(Content-Disposition: attachment)

#### D-15: 前端下载走 axios blob 模式
- 使用 `api.get(url, { params, responseType: 'blob' })`(Phase 33 M2 已修正为走 axios 拦截器)
- 触发 a 标签 `URL.createObjectURL` + `download` 属性下载
- 错误处理: 当 blob 大小异常(< 1KB)时,尝试解析为 JSON 错误体并抛出业务错误

### GA-04 跨模块联动入口(用户澄清)

#### D-16: 联动入口 = 仅网络设备详情页
- 位置: 网络设备详情页头部操作区,新增按钮 `查看 MAC 历史` (icon: `HistoryOutlined`)
- 跳转: `/network/mac/history?deviceId=...&portName=...`(可选)
- 行为: 查询页读取 query 参数,自动填入"设备"和"端口"过滤项,用户可继续调整时间范围
- 不在工位详情页加入口(工位 → MAC 历史需要新增后端"按工位/用户查 MAC"接口,属未来能力)
- 不在轨迹页 Gantt 节点加点击入口(节点点击属增强 UX,归 deferred)

#### D-17: URL 参数 = query 串
- 格式: `?deviceId=xxx&portName=yyy&startTime=...&endTime=...`
- 查询页在 `useEffect` 读取 URLSearchParams(参考 Phase 13 轨迹页同模式)
- 字段全部可选,缺失时退化为"全部"

### GA-05 状态兜底(用户澄清)

#### D-18: 空数据 = 引导去设备采集
- 触发条件: 查询返回 `total === 0` 时
- 文案: `该范围内未采集到 MAC 记录,请检查设备是否启用了 MAC 采集/端口采集周期`
- 操作: 附 `前往设备管理` 链接(跳转 `/network/devices`)
- 实现: 抽 `EmptyStateWithAction` 组件,接受 `actionLabel` + `actionPath` props

#### D-19: 加载状态 = Skeleton 占位
- 表格首次加载: AntD `Skeleton` 表格骨架(3-5 行)
- 时间线首次加载: AntD `Skeleton` 输入/段落骨架
- 后续分页/筛选: 不显示骨架(避免闪烁),使用 `useQuery` 的 `isFetching` 触发表头 `Spin`

#### D-20: 错误状态 = 内联 Alert + 重试
- 错误展示: AntD `Alert type="error"` + 错误信息 + `重试` 按钮(回调 `query.refetch()`)
- 区分错误码:
  - `1006` 设备未找到 → 提示"该设备不存在或已被删除"
  - `1007` token 失效 → 跳登录
  - `500` 服务器错误 → 提示"服务暂不可用,请稍后重试"
  - 其他 → 通用"查询失败"
- 抽 `ErrorAlertWithRetry` 组件,接受 `error` + `onRetry` props

### Claude's Discretion
- 时间线与列表页"展开行"的交互细节(展开行容器高度、虚拟滚动中的行高稳定性)
- 移动端卡片视图的字段展示顺序与折叠策略(默认时间/MAC/设备/端口在最前,其他可点击展开)
- 快捷预设按钮在窄屏(< sm)的折叠策略(可显示为"更多"下拉)
- 14-04 后端流式响应的具体实现(同步流式 vs 异步任务;planner 阶段决定)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Roadmap
- `.planning/REQUIREMENTS.md` §UI-01/UI-02/UI-04 — 锁定需求 (UI-03 归 Phase 13)
- `.planning/ROADMAP.md` §Phase 14 — Goal、Depends on、5 个预期子计划、Phase Highlights
- `.planning/STATE.md` — Phase 13 已 shipped, Phase 14 Planned (0/5 plans)

### 直接前置 Phase 交付
- `.planning/phases/13-query-layer-trajectory/13-04-SUMMARY.md` — `MACTrajectoryChart.tsx` ECharts 组件、轨迹页 `trajectory.tsx`、颜色体系(appeared/moved/disappeared/vlan_changed)
- `.planning/phases/13-query-layer-trajectory/13-06-menu-registration-v4.sql` — Phase 13 已注册的菜单模式(本 phase 14-03 复用其结构)
- `.planning/phases/13-query-layer-trajectory/13-CONTEXT.md` — Phase 13 决策记录(API 路径 `/network/history/list`、`/network/history/trajectory` 等)

### 性能/前端架构参考
- `.planning/phases/30-js/30-CONTEXT.md` §D-06..D-09 — 路由懒加载、vendor chunk、按需加载(本 phase 新页面须遵守)
- `.planning/phases/33-vercel-react-best-practices-20260613-26/CONTEXT.md` §Wave 1+2 — C1-C7 性能/重渲染修复经验(M2 ExcelImport 走 axios blob 即用即用), `useTableManager` deps 稳定化(M1), 内联 style 提取(C7)

### 既有可复用组件/hooks
- `xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx` — 复用,不重写
- `xingran-react-frontend/src/components/shared/ExcelImport.tsx` — 参考 Excel 上传/下载模式(Phase 33 M2 已修复为 axios blob)
- `xingran-react-frontend/src/hooks/useTableManager.ts` — 列表管理 hook(虚拟滚动需扩 `placeholderData` 支持)
- `xingran-react-frontend/src/hooks/useColumnConfig.ts` — 列配置 hook(Phase 27 全局列自定义)
- `xingran-react-frontend/src/lib/api.ts` — `post`/`get`/`api.get(url, { responseType: 'blob' })` 包装
- `xingran-react-frontend/src/lib/api/networkApi.ts` — Phase 13 新增的 `queryMACTrajectory`,本 phase 新增 `queryMACHistory` 走同文件
- `xingran-react-frontend/src/pages/operations/buildings/index.tsx` — 桌面/移动 表格↔卡片 切换参考

### 规范/规范类文档
- `docs/开发规范.md` — 前端开发规范(由 CLAUDE.md 引用)
- `docs/项目概述和架构设计.md` — 总体架构
- `docs/安全和认证设计（国密）.md` — `network:mac:export` 权限点是否需加密,Phase 13 13-06 menu SQL 已处理

</canonical_refs>

<codebase_context>
## Existing Code Insights

### Reusable Assets
- `MACTrajectoryChart.tsx`: 已实现的 ECharts Gantt 组件,接受 trajectory 数据,自带 Empty/Loading/Error 状态、颜色编码 (appeared/moved/disappeared/vlan_changed),DataZoom 交互 → **本 phase 时间线组件复用同色系图标**(D-10)
- `useTableManager`: 列表分页/筛选/排序 hook → 14-01 列表页直接复用,需要小扩展(虚拟滚动场景加 `placeholderData: keepPreviousData`)
- `useColumnConfig`: 列显示/隐藏/排序 hook(Phase 27) → 14-01 列表页列配置用此 hook
- `api.get(url, { responseType: 'blob' })`: Excel 下载接口(Phase 33 M2 修复) → 14-04 导出按钮直接调用
- `lib/api/networkApi.ts`: Phase 13 新建,放 MAC 相关 API → 14-01/14-04 新增的 `queryMACHistory` / `exportMACHistory` 走此文件
- Phase 13 `trajectory.tsx` 已有 `useEffect` 读取 URL query 参数 → 14-01 列表页用同模式读取 `deviceId/portName`

### Established Patterns
- **React Query 缓存键规范** (Phase 30/33 经验): 列表查询用 `['macHistory', params]`,轨迹用 `['macTrajectory', params]`(Phase 13 已用),时间线用 `['macEvents', mac, range]`,**避免重复请求**
- **路由懒加载** (Phase 30 D-08): 新增 `/network/mac/history` 路由用 `React.lazy` + Suspense 包裹
- **错误码映射**: 后端定义 `1006 设备未找到`/`1007 token 失效`/`500 服务器错误`(CLAUDE.md API 响应规范) → 14-05 错误状态用此映射
- **菜单 SQL 注册**: Phase 13-06 `13-06-menu-registration-v4.sql` 模式 → 14-03 沿用(`network` 父菜单下新增 `mac-history` 子菜单)
- **状态值约定**: `0=正常 1=停用`(CLAUDE.md) → 列表页"状态"列展示时按此约定文案映射

### Integration Points
- **网络设备详情页**: 需要在该页头部操作区新增 `查看 MAC 历史` 按钮,跳 `/network/mac/history?deviceId=...` → 14-01 需读取 URL 参数
- **`/network/history/list`**: 现有后端接口(Phase 13),14-01 拉列表/14-04 加 `format=xlsx` 分支共用
- **`/network/mac/trajectory`**: 现有后端接口(Phase 13),14-02 增强轨迹页交互(从历史事件跳来时自动填参数)
- **`HybridLayout` + AntD Grid**: 14-05 移动端响应式走现有布局系统
- **`sys_menu` 表**: 14-03 需新增菜单 SQL,参考 `13-06-menu-registration-v4.sql`

</codebase_context>

<specifics>
## Specific Ideas

- **时间线颜色与图标与 Gantt 一致**: 用户在讨论中明确要求"按事件类型颜色编码,与 Phase 13 MACTrajectoryChart 一致"。颜色 `#52c41a`/`#faad14`/`#ff4d4f`/`#1890ff` 来自 13-04 SUMMARY。
- **导出走现有接口 + format 参数**: 用户优先选择"复用查询接口 + 特殊 query 参数",避免新增 API 端点。这与项目一贯的"少增端点,多用 query/header 分支"风格一致。
- **联动入口仅设备详情页**: 用户选择"先做网络设备详情页,工位/轨迹节点点击归 deferred",体现"分阶段交付、避免范围爆炸"。

## Open to Standard Approaches
- 时间线组件 API 设计(props 形态、内部状态归属)由 planner 决定
- 14-05 空/加载/错误三态的具体实现细节(component 抽取粒度、文件位置)由 planner 决定
- 14-02 轨迹页 UX 增强的具体交互(dataZoom 默认范围、tooltip 内容扩展)由 planner 决定

</specifics>

<deferred>
## Deferred Ideas

### GA-06 厂商识别 (OUI 库)
- **原因**: OUI 库(厂商 ↔ MAC 前缀)需要数据源与缓存策略,本 phase 14 聚焦"前端 UX",数据层属后端/数据治理
- **去向**: 可作为 Phase 15 性能优化 或独立"MAC 厂商识别"phase(后端建 OUI 表 + 前端展示)
- **当前状态**: MACTrajectoryChart 已用 `event_type` 颜色编码,厂商识别不影响核心 UX

### 工位详情页 → MAC 历史入口
- **原因**: 工位需要新增"按工位/用户查 MAC 历史"后端接口(本 phase 不动后端)
- **去向**: 未来 phase(需后端配合),或当 Phase 15 性能优化时一并评估

### 轨迹页 Gantt 节点点击 → 查询页
- **原因**: 节点点击属"轨迹→查询"反向跳转,与本 phase D-16 "查询→轨迹"主链路方向相反;增加交互复杂度
- **去向**: 增强 UX,作为未来 phase(可能 Phase 15 或独立 UX 增强 phase)

### 单 MAC 导出按钮 (操作列)
- **原因**: 与"导出当前查询"功能重叠,且增加操作列宽度,在移动端不友好
- **去向**: 当前 D-13 工具栏"导出当前查询"已覆盖该需求

### PDF 导出
- **原因**: 用户明确选择"仅 Excel",PDF 含时间线截图需额外后端渲染能力
- **去向**: 未来 phase(运维周报场景)

### 跨页 "事件中心" `/network/mac/events` 独立路由
- **原因**: GA-02 讨论时作为选项,被否;当前时间线作为可复用组件嵌入多页
- **去向**: 仅当未来有"事件流订阅/告警"等需求时再考虑独立路由

</deferred>

---

*Phase: 14-frontend-ux*
*Context gathered: 2026-06-14 via gsd:discuss-phase (5 gray areas clarified, 1 deferred)*
</content>
</invoke>