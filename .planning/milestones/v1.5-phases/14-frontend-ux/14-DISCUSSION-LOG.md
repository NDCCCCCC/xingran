# Phase 14: 前端与用户体验 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-14
**Phase:** 14-frontend-ux
**Areas discussed:** GA-01 列表页查询模式, GA-02 时间线组件形态, GA-03 导出策略, GA-04 跨模块联动入口, GA-05 状态兜底

---

## GA-01: 列表页查询模式

### GA-01.1 主列表分页策略

| Option | Description | Selected |
|--------|-------------|----------|
| 虚拟滚动(推荐) | 与 Phase 30 资产列表一致(43 列场景),上下滚动更连贯,代码复杂度中 | ✓ |
| 传统分页 (page/size) | 与 XingRan 默认分页一致,代码简单;数据量超过 1000+ 体验差 | |
| 游标分页 (cursor-based) | 按 time/id 游标加载更多,需后端提供 cursor 参数 | |

**User's choice:** 虚拟滚动
**Notes:** 用户倾向 Phase 30 已验证的方案,接受 placeholderData/keepPreviousData 复杂度。

### GA-01.2 时间范围筛选器形态

| Option | Description | Selected |
|--------|-------------|----------|
| 快捷预设 + RangePicker(推荐) | 预设备选按钮(近 1h/24h/7d/30d/90d/自定义),互斥同步 | ✓ |
| 纯 RangePicker | 只有 RangePicker,无预设入口 | |
| 双独立 DatePicker | 开始/结束各一个 DatePicker,占地方 | |

**User's choice:** 快捷预设 + RangePicker
**Notes:** 显式默认 `近 7d`(与 Phase 13 轨迹页一致)。

### GA-01.3 列结构

| Option | Description | Selected |
|--------|-------------|----------|
| 全列展示 | 0-9 列,使用 useColumnConfig | ✓ |
| 可分组(主+详情) | 主列+详情分组,与资产列表一致 | |

**User's choice:** 全列展示
**Notes:** 列数适中(8 列),不需要分组;仍走 useColumnConfig 让用户可隐藏。

---

## GA-02: 时间线组件形态

### GA-02.1 时间线定位

| Option | Description | Selected |
|--------|-------------|----------|
| 独立组件跨页复用(推荐) | `components/network/MACEventsTimeline.tsx`,多页消费 | ✓ |
| 查询页内嵌子组件 | 仅查询页"抽屉/右侧面板" | |
| 独立路由"事件中心" | `/network/mac/events` 专门看事件流 | |

**User's choice:** 独立组件跨页复用
**Notes:** 代码集中,后续设备详情页/工位详情页可复用。

### GA-02.2 时间线视觉布局

| Option | Description | Selected |
|--------|-------------|----------|
| 垂直时间线(推荐) | AntD Timeline 风格,按事件类型颜色/图标 | ✓ |
| 水平滑动时间线 | 与 Gantt 接近,需额外库 | |
| 事件流紧凑卡 | GitHub 提交历史风格,最省空间 | |

**User's choice:** 垂直时间线
**Notes:** 颜色与图标必须与 MACTrajectoryChart 一致(绿/黄/红/蓝 + PlusCircleOutlined / SwapOutlined / MinusCircleOutlined / TagOutlined)。

### GA-02.3 时间线与 Gantt 轨迹图的关系

| Option | Description | Selected |
|--------|-------------|----------|
| 事件可点击跳轨迹(推荐) | 点击事件项跳 `/network/mac/trajectory?mac=...&...` | ✓ |
| 仅展示不联动 | 只作"详情增强" | |
| 同页 Tabs 切换 | 表格/时间线/Gantt 三视图 | |

**User's choice:** 事件可点击跳轨迹
**Notes:** 沿查询→轨迹主链路,避免在同页与 Gantt 主视图争抢布局。

---

## GA-03: 导出策略

### GA-03.1 导出格式

| Option | Description | Selected |
|--------|-------------|----------|
| 仅 Excel(推荐) | 复用后端 xuri/excelize/v2 | ✓ |
| Excel + CSV | 多一套下载逻辑 | |
| Excel + PDF | 需额外后端渲染 | |

**User's choice:** 仅 Excel
**Notes:** 与所有其他模块(资产/工位/设备)保持一致格式。

### GA-03.2 导出范围与触发 (multiSelect)

| Option | Description | Selected |
|--------|-------------|----------|
| 仅当前查询条件(推荐) | 工具栏按钮 | ✓ |
| 全量导出 | 异步任务或流式 | ✓ |
| 单 MAC 导出 | 操作列"导出该 MAC" | |

**User's choice:** 仅当前查询条件 + 全量导出
**Notes:** 不在操作列重复按钮(避免 UI 重复);工具栏两个互斥按钮。

### GA-03.3 后端接口提供方式

| Option | Description | Selected |
|--------|-------------|----------|
| 复用查询接口 + 特殊 header(推荐) | `/network/history/list?format=xlsx`,不增 API | ✓ |
| 新增专用 /export 接口 | 接受复杂参数,与列表解耦 | |

**User's choice:** 复用查询接口 + 特殊 header/query
**Notes:** 与项目"少增端点,多用 query/header 分支"风格一致。

---

## GA-04: 跨模块联动入口

### GA-04.1 入口位置 (multiSelect)

| Option | Description | Selected |
|--------|-------------|----------|
| 网络设备详情页(推荐) | 头部"查看 MAC 历史"按钮,带 deviceId/portName | ✓ |
| 工位详情页 | 需后端"按工位/用户查 MAC"新接口 | |
| 主菜单独立入口 | 仅主菜单,联动效果有限 | |
| 轨迹页"设备/MAC"按钮 | 节点点击,反向跳转 | |

**User's choice:** 仅网络设备详情页
**Notes:** 工位/轨迹节点点击均归 deferred(需后端配合或属增强 UX)。

### GA-04.2 URL 参数传递格式

| Option | Description | Selected |
|--------|-------------|----------|
| query 串(推荐) | `?deviceId=xxx&portName=yyy&startTime=...&endTime=...` | ✓ |
| URL path | `/network/mac/history/{deviceId}/{portName}`,需动态路由 | |
| 携带 MAC | 只传 MAC,设备/端口默认不限 | |

**User's choice:** query 串
**Notes:** 与 Phase 13 轨迹页同模式(已用 `useEffect` 读 URLSearchParams);查询页可读同套。

---

## GA-05: 状态兜底 (用户同时选择讨论)

### GA-05.1 空数据状态

| Option | Description | Selected |
|--------|-------------|----------|
| 设备采集未启用(推荐) | 提示 + "前往设备管理"链接 | ✓ |
| 参数不正确 | 仅静态文案 | |
| 分页不一致 | 区分"尚未查询过"vs"无结果" | |

**User's choice:** 设备采集未启用
**Notes:** 抽 `EmptyStateWithAction` 组件,接受 `actionLabel` + `actionPath` props。

### GA-05.2 加载状态表现

| Option | Description | Selected |
|--------|-------------|----------|
| 骨架 + 内联 Alert(推荐) | AntD Skeleton + Alert + 重试按钮 | ✓ |
| Spin 居中加载 | 仅 Spin | |
| 什么都不做 | 仅 console.error | |

**User's choice:** 骨架 + 内联 Alert
**Notes:** 与 Phase 30/33 风格一致;后续分页/筛选不显示骨架(用 isFetching 触发表头 Spin)。

### GA-05.3 错误状态分级

| Option | Description | Selected |
|--------|-------------|----------|
| 区分错误码(推荐) | 1006/1007/500 等分类 | ✓ |
| 统一通用文案 | 仅"查询失败" | |

**User's choice:** 区分错误码
**Notes:** 1006 设备未找到 / 1007 token 失效 / 500 服务器错误 / 其他通用;抽 `ErrorAlertWithRetry` 组件。

---

## GA-06: 厂商识别 (用户选择归 deferred)

**User's choice:** 不深入讨论(归 deferred)
**Notes:** OUI 库属数据层能力,本 phase 14 聚焦"前端 UX";MACTrajectoryChart 已用 event_type 颜色编码,不影响核心 UX。归入 Phase 15 性能优化或独立"MAC 厂商识别"phase。

---

## Claude's Discretion

| Area | 决定 | 依据 |
|------|------|------|
| 时间线与列表页"展开行"交互细节 | planner 决定 | 属实现细节,不影响决策 |
| 移动端卡片视图字段顺序 | planner 决定(时间/MAC/设备/端口 最前) | 用户未指定 |
| 快捷预设按钮窄屏折叠 | planner 决定(可显示"更多"下拉) | UX 细节 |
| 14-04 后端流式响应实现 | planner 决定(同步流式 vs 异步任务) | 实现选型 |

## Deferred Ideas

| Idea | 原因 | 建议去向 |
|------|------|----------|
| OUI 厂商识别 (GA-06) | 需数据源与缓存策略,聚焦数据层 | Phase 15 / 独立 phase |
| 工位详情页 → MAC 历史入口 | 需后端"按工位/用户查 MAC"新接口 | 未来 phase |
| 轨迹页 Gantt 节点点击 → 查询页 | 反向跳转,增加交互复杂度 | 未来 UX 增强 |
| 单 MAC 导出按钮(操作列) | 与"导出当前查询"重叠 | 当前已覆盖 |
| PDF 导出 | 需后端渲染,用户未选 | 未来 phase(运维周报) |
| 跨页"事件中心"独立路由 | 讨论时作为选项,被否 | 仅当"事件流订阅/告警"出现时 |

---

*Discussion completed: 2026-06-14 (5 gray areas clarified, 1 deferred to future phases)*
</content>
</invoke>