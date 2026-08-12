---
status: complete
phase: 14-frontend-ux
source:
  - 14-01-SUMMARY.md
  - 14-02-SUMMARY.md
  - 14-03-SUMMARY.md
  - 14-05a-SUMMARY.md
  - xingran-react-frontend/14-04-SUMMARY.md
  - xingran-react-frontend/14-05b-SUMMARY.md
started: 2026-06-15T00:00:00Z
updated: 2026-07-08T00:00:00Z
closed_by_audit: .planning/reports/uat-audit-2026-07-08.md
---

## Current Test

[testing complete]

## Tests

### 1. MAC 历史查询主列表页加载
expected: 浏览器访问 /network/mac/history,页面正常加载,顶部有查询表单(包含 6 个时间预设按钮 + RangePicker + 5 个筛选字段),下方为 8 列全列表格(time/mac/device/port/eventType/vlan/status/action)。
result: pass

### 2. 时间预设按钮组(14-01)
expected: 查询表单上方有 6 个时间预设按钮(近 1h / 近 24h / 近 7d / 近 30d / 近 90d / 自定义),点击"近 24h"后时间范围自动填充为过去 24 小时。
result: skipped
reason: 用户跳过全部测试

### 3. 操作列两个跳转按钮
expected: 表格操作列有两个按钮"查看轨迹"和"查看事件",点击"查看轨迹"会跳转到 /network/mac/trajectory 页(URL 含 mac、startTime、endTime 参数)。
result: waived
reason: UAT 审计 2026-07-08 判定 needs_update。MACHistoryPage.tsx:284-307 操作列仅实现"查看事件"(行展开内嵌 MACEventsTimeline),"查看轨迹"按钮未实现且 trajectory 页面从未构建(见 T5)。v1.19 归档前主动放弃该功能,后续如需轨迹页另立 phase。

### 4. URL 参数预填 (14-01 history.tsx)
expected: 浏览器直接访问 /network/mac/history?mac=AA:BB:CC:DD:EE:FF&deviceId=xxx&portName=Eth0/1,进入页面后表单字段会被 URL 参数自动填充(MAC、设备、端口)。
result: skipped
reason: 用户跳过全部测试

### 5. MAC 轨迹页加载 (14-02)
expected: 浏览器访问 /network/mac/trajectory,页面正常加载,顶部有查询表单(包含 6 个时间预设 + MAC 输入框),中部为 ECharts Gantt 图,右侧默认展开"事件时间线"侧边栏。
result: waived
reason: UAT 审计 2026-07-08 判定 stale。`/network/mac/trajectory` 页面从未实现,src 中 grep "trajectory" 零命中;实际交付为 `/network/mac/heatmap` (Phase 15 PERF-04)。trajectory Gantt 形态不再立项,该 UAT 项作废。

### 6. 轨迹页 URL 自动查询 (14-02)
expected: 浏览器访问 /network/mac/trajectory?mac=AA:BB:CC:DD:EE:FF&startTime=2026-06-14T00:00:00Z&endTime=2026-06-15T00:00:00Z,进入页面后自动填表并自动触发查询(无需点击"查询"按钮),Gantt 图渲染数据。
result: waived
reason: UAT 审计 2026-07-08 判定 stale。trajectory 页面从未实现(见 T5),URL 自动查询随之作废。

### 7. 轨迹页 dataZoom 默认范围
expected: 打开轨迹页后,ECharts Gantt 图的时间轴默认聚焦最近 1/3 时间区间(dataZoomStart=66, dataZoomEnd=100)。
result: waived
reason: UAT 审计 2026-07-08 判定 stale。trajectory 页面从未实现,dataZoom 配置随之作废。

### 8. 停留时长热力 tooltip
expected: 在 Gantt 图节点上 hover 鼠标,tooltip 顶部出现停留时长热力色块(< 1h 灰色 / < 24h 蓝色 / < 7d 橙色 / >= 7d 红色),色块下方显示停留时长文字。
result: waived
reason: UAT 审计 2026-07-08 判定 stale。trajectory 页面从未实现,4 档时长热力 tooltip 随之作废。`formatDurationSeconds()` 工具函数在 utils/duration.ts 中保留但未启用。

### 9. 右侧事件时间线 Drawer
expected: 轨迹页右侧 Drawer 默认打开,内嵌 MACEventsTimeline 组件,显示 4 种事件类型(appeared/moved/disappeared/vlan_changed)及对应颜色图标。Drawer 关闭后出现"显示时间线"按钮可重新打开。
result: skipped
reason: 用户跳过全部测试
note: 组件已实现但部署为 history 页的行展开(expandedRowRender),非 Drawer 形态;Drawer 形态仅在 mac list 页使用。

### 10. Excel 导出 - 工具栏按钮
expected: 在 history 页工具栏"查询"/"重置"按钮之后,出现两个导出按钮"导出当前查询"(蓝色 primary)和"导出全量"(灰色 default),均带 DownloadOutlined 图标。
result: skipped
reason: 用户跳过全部测试

### 11. Excel 导出 - 下载文件
expected: 点击"导出全量"按钮,浏览器下载一个 .xlsx 文件,文件名格式为 mac_history_all_YYYYMMDD_HHmmss.xlsx。
result: waived
reason: UAT 审计 2026-07-08 判定 needs_update。`networkApi.ts:172` 实际 fallback 文件名格式为 `mac_history_${scope}_${Date.now()}.xlsx`(Unix ms 非格式化日期);服务器 `Content-Disposition` 头另控文件名。该差异为非关键,接受现状不动;v1.19 归档前主动放弃 strict 文件名格式校验。

### 12. 导出按钮权限控制
expected: 当前用户没有 network:mac:export 权限时,工具栏上完全不显示"导出当前查询"和"导出全量"两个按钮(使用条件渲染,非 disabled)。
result: skipped
reason: 用户跳过全部测试

### 13. EmptyStateWithAction 空数据态
expected: 在 history 页查询一个无数据的条件(如未来时间范围),列表区域显示 EmptyStateWithAction 组件,文案为"该范围内未采集到 MAC 记录,请检查设备是否启用了 MAC 采集/端口采集周期",下方有"前往设备管理"按钮点击跳转 /network/devices。
result: skipped
reason: 用户跳过全部测试

### 14. ErrorAlertWithRetry 错误态
expected: 手动制造一个 API 错误(如断网),history 页列表区域显示 ErrorAlertWithRetry 组件,提供"重新加载"按钮点击触发 refetch。
result: skipped
reason: 用户跳过全部测试

### 15. 移动端卡片视图
expected: 使用浏览器 devtools 切换到 < 576px 视口宽度,history 页从 AntD Table 切换为 AntD List 卡片视图,卡片字段顺序:时间 / MAC / 设备 / 端口 / 事件类型 / VLAN。
result: skipped
reason: 用户跳过全部测试

### 16. 网络设备列表"查看 MAC 历史"按钮
expected: 在 /network/devices 列表页,每一行新增一个"查看 MAC 历史"按钮(HistoryOutlined 图标),点击跳转 /network/mac/history?deviceId=xxx&portName=xxx。
result: skipped
reason: 用户跳过全部测试

### 17. 轨迹页三态打磨 (14-05b)
expected: trajectory 页的错误态使用 ErrorAlertWithRetry 组件替换原内联 Alert,空数据态使用 EmptyStateWithAction。
result: waived
reason: UAT 审计 2026-07-08 判定 stale。trajectory 页面从未实现;实际三态打磨已落地于 `/network/mac/heatmap` 页(Phase 15)。trajectory 形态作废,该 UAT 项随之关闭。

## Summary

total: 17
passed: 1
issues: 0
pending: 0
skipped: 9
waived: 7
waived_at: 2026-07-08
waived_reason: trajectory 页面形态作废(5 项 stale)+ "查看轨迹"按钮未实现(1 项 needs_update)+ 文件名格式非关键差异(1 项 needs_update);详见 .planning/reports/uat-audit-2026-07-08.md

## Gaps

[none yet]