# 服务端排序任务状态（截至 2026-06-19）

> 排序任务主体已完成。本文档记录最终状态 + 剩余低 ROI 项。
> 避坑指南：见 `memory/server-sort-loadfunc-param-drop.md`（loadXxx 透传 + React 18 setState 异步时序）。

## ✅ 已完成并 UAT 真验证（后端白名单+ApplySort + 前端透传 全链路）

| 模块 | 页面 | 验证证据 |
|------|------|---------|
| system | user / role / post / dict / config / notice | 各列点击列头请求带 orderByColumn + 响应正确升降序 |
| system | apikeys | 请求 {orderByColumn:name,isAsc:true} 链路通（演示无数据） |
| operations | building / floor / workstation / asset | 跨页排序一致（asset 6687 条 devicesn 升序验证） |
| network | devices | devicesn 升序完美 |
| monitor | server / cache | hostName / key 排序工作 |
| monitor | logs（操作日志 + 登录日志） | operName / ipAddr 升序（888/679 条跨页） |
| ad-domain | users（8655条）/ groups / configs | username / groupName / configName 升序验证 |

## ⚠️ 代码已完成但路由不可达（dead code，待菜单/权限注册后生效）

这些页面代码已按标准模式接入排序，但当前用户菜单未挂载（动态路由不生成），无法 UAT。
后端白名单 + ApplySort + 前端透传均已就绪，待 sys_menu 注册菜单后即生效。

| 页面 | 后端白名单 | 状态 |
|------|-----------|------|
| duty/pools | dutyPoolAllowedSortFields（poolName/deptId/status/createdAt） | 纯前端修复（createSorter→服务端），commit d2e28bd |
| duty/schedules | dutyScheduleAllowedSortFields（scheduleDate/dutyType/status） | 纯前端修复，commit d2e28bd |
| knowledge/articles | knowledgeArticleAllowedSortFields | 前端无路由定义，commit c59bb9d |
| network/templates | templateAllowedSortFields | 前端无路由定义，commit c59bb9d |

## ❌ 真未做（低 ROI，按需推进）

### duty/my-duty（统计页）
- 是值班统计页（GetMyDutyStats），大概率无列表排序需求。需先确认是否有列表 Table。

### duty/holidays（后端签名需改造 + 路由不可达）
- 后端 `DutyHolidayService.GetHolidayList(ctx, year int)` 只接 year（无 BaseListRequest/current/pageSize），
  前端 `pagination={false}`（无分页）。需先改后端签名为 `(ctx, req *HolidayListRequest)` 嵌入 BaseListRequest，
  参考 `notice_service.go` GetNoticeList 改法。且 duty 模块整体路由不可达。

### system/menu + system/dept（树形 Table）
- 均为树形 Table（renderTreeData + pagination={false}）。
- **UX 评估**：树形主要按 order_num（排序号）/层级展示，点击列头排序的意义存疑
  （树形排序应通过拖拽调整 order_num，而非列头点击）。
- 如要做：menu_service.go 已硬编码 `Order("order_num ASC")`，需评估是否值得加白名单。
- 建议：与产品确认树形排序需求后再定，当前保持 order_num 默认排序。

## 实施模板（已完成页面通用模式）

### 后端（Go）
```go
// 1. service 顶部加白名单
var xxxAllowedSortFields = map[string]string{
    "fieldCamel": "db_snake_col",
}
// 2. List 方法内（Count 之后、Find 之前）
query = base.ApplySort(query, req.BaseListRequest, xxxAllowedSortFields)
if req.OrderByColumn == "" {
    query = query.Order("created_at DESC") // 默认排序兜底
}
// 3. ListRequest 嵌入 base.BaseListRequest（或加 OrderByColumn/IsAsc 字段）
// 4. handler 用 ShouldBindJSON 直接绑定（嵌入字段自动透传）
```

### 前端（React）
```typescript
// 关键避坑（详见 memory server-sort-loadfunc-param-drop）：
// 1. handleTableChange 读 sorter 后，用 local const 持有新值传 fetchXxx
//    （不能依赖 state——React 18 setState 异步，同周期读 state 仍为旧值）
// 2. fetchXxx/loadXxx 重建 requestParams 时必须透传 orderByColumn/isAsc
//    （不能只挑 current/pageSize + 搜索值，会丢排序参数）
// 3. 列 sorter:true + 受控 sortOrder（getColumnSortOrder 只对当前列返回方向）
```

## 相关 commit（fix/login-wrong-pwd-inline-error 分支）

核心修复：f1a952f, 82571a9, 84416b8
收尾修复：38c7f68（dict缓存+config）, e7a9a8d（notice）
P1/P2 推广：3c6b0e7（monitor）, f0816da（ad-domain）, c59bb9d（knowledge+network）
质量修复：3c0ac3a, 233b0fa, 7ab1189（React 18 setState 异步时序）
扩展：28f9a86（apikeys）, d2e28bd（duty pools+schedules）, 06fa8ba（ad-domain configs）
