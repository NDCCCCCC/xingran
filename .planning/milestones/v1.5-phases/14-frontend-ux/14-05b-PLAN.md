<!-- Revised 2026-06-14 — plan-checker iteration 1: 修复 BLOCKER 1+2 + WARNING 1-4 -->
---
phase: 14-frontend-ux
plan: 05b
type: execute
wave: 3
depends_on:
  - 14-01
  - 14-04
  - 14-05a
files_modified:
  - xingran-react-frontend/src/pages/network/mac/history.tsx
  - xingran-react-frontend/src/pages/network/mac/trajectory.tsx
  - xingran-react-frontend/src/pages/network/devices/index.tsx
autonomous: true
requirements:
  - UI-01
  - UI-02
  - UI-04

must_haves:
  truths:
    - "桌面端 (>= 576px) 列表使用 AntD Table + virtual 滚动;移动端 (< 576px) 切换 AntD List 卡片视图,字段顺序:时间 / MAC / 设备 / 端口"
    - "空数据态:使用 EmptyStateWithAction 组件,文案 '该范围内未采集到 MAC 记录',提供 '前往设备管理' 链接跳 /network/devices"
    - "加载态:首次加载显示 AntD Skeleton (3-5 行表格骨架);后续分页/筛选使用 useQuery.isFetching 触发表头 Spin"
    - "错误态:使用 ErrorAlertWithRetry 组件,错误码 1006 显示 '该设备不存在或已被删除';1007 跳登录;500 显示 '服务暂不可用';其他 '查询失败'"
    - "网络设备列表页行内新增 '查看 MAC 历史' 按钮 (icon: HistoryOutlined),点击跳 /network/mac/history?deviceId=...&portName=..."
    - "14-04 注入的 '导出当前查询' / '导出全量' 两个按钮在 history.tsx 改造后仍存在且 prop 名称/位置不变"
  artifacts:
    - path: "xingran-react-frontend/src/pages/network/mac/history.tsx"
      provides: "改造 14-01 列表页,使用三态组件 + 移动端 List 卡片,保留 14-04 导出按钮"
      contains: "EmptyStateWithAction|ErrorAlertWithRetry"
    - path: "xingran-react-frontend/src/pages/network/mac/trajectory.tsx"
      provides: "改造 14-02 轨迹页,替换 Alert 为 ErrorAlertWithRetry + 空数据 EmptyStateWithAction"
      contains: "ErrorAlertWithRetry"
    - path: "xingran-react-frontend/src/pages/network/devices/index.tsx"
      provides: "网络设备列表页行内 '查看 MAC 历史' 按钮跳 /network/mac/history?deviceId=..."
      contains: "mac/history?deviceId"
  key_links:
    - from: "xingran-react-frontend/src/pages/network/mac/history.tsx"
      to: "xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx"
      via: "total === 0 时渲染"
      pattern: "EmptyStateWithAction"
    - from: "xingran-react-frontend/src/pages/network/mac/history.tsx"
      to: "xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx"
      via: "useQuery.error 非空时渲染"
      pattern: "ErrorAlertWithRetry"
    - from: "xingran-react-frontend/src/pages/network/devices/index.tsx"
      to: "/network/mac/history?deviceId=..."
      via: "useNavigate 跳路由"
      pattern: "navigate.*mac/history"
    - from: "xingran-react-frontend/src/pages/network/mac/history.tsx"
      to: "14-04 注入的 exportMACHistory 按钮"
      via: "保留 prop 名称 exportScope='current'|'all' 与工具栏位置"
      pattern: "导出当前查询|导出全量"
---

<objective>
将 14-01 / 14-02 创建的 history 与 trajectory 页面改造为使用 14-05a 提供的 `EmptyStateWithAction` 与 `ErrorAlertWithRetry` 共享组件,落实 Phase 14 的三态打磨(D-18 / D-20)。同时:
1. 移动端响应式:列表页用 Grid.useBreakpoint() 判定 xs,自动切到 AntD List 卡片视图
2. 网络设备列表页联动入口:行操作区新增 查看 MAC 历史 按钮,跳 /network/mac/history?deviceId=...&portName=...(D-16 锁定)
3. **保留 14-04 注入的导出按钮**:executor 在 history.tsx 改造时必须保持 "导出当前查询" / "导出全量" 两个按钮的 prop 名称(`exportScope='current'|'all'`)与 React 节点位置(工具栏查询/重置按钮之后)不变;按钮文字、权限控制、网络下载逻辑等 14-04 既有代码不删除

Purpose: 关闭 Phase 14 UX 兜底缺口,完成 v1.5 MAC 历史数据管理的工程级前端体验。Wave 3 串行执行,确保 14-04 注入的代码不被 14-05b 改造覆盖。

Output: 14-01 / 14-02 列表与轨迹页三态改造 + 网络设备列表页新增按钮,导出按钮保留。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/14-frontend-ux/14-CONTEXT.md (D-16..D-20 全部锁定)
@.planning/REQUIREMENTS.md (UI-01/UI-02/UI-04)
@.planning/ROADMAP.md (14-05 标题与目标)
@.planning/CLAUDE.md (HybridLayout 侧边栏折叠行为,ErrorCode 1006/1007/500 来自 API 响应规范)
@.planning/phases/14-frontend-ux/14-01-PLAN.md
@.planning/phases/14-frontend-ux/14-02-PLAN.md
@.planning/phases/14-frontend-ux/14-04-PLAN.md (含 14-04 注入的导出按钮契约)
@.planning/phases/14-frontend-ux/14-05a-PLAN.md (本 plan 依赖其提供的 shared 组件)

# 参考实现
@xingran-react-frontend/src/pages/operations/buildings/index.tsx
@xingran-react-frontend/src/pages/network/devices/index.tsx
</context>

<tasks>

<task type="auto">
  <name>Task 1: 改造 history 与 trajectory 页使用三态组件 + 移动端 List 卡片(保留 14-04 导出按钮)</name>
  <files>xingran-react-frontend/src/pages/network/mac/history.tsx, xingran-react-frontend/src/pages/network/mac/trajectory.tsx</files>
  <read_first>
    - D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\pages\network\mac\history.tsx (14-01 + 14-04 改动后,本 plan 改造)
    - D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\pages\network\mac\trajectory.tsx (14-02 增强版)
    - D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\pages\operations\buildings\index.tsx (xs=24 sm=12 md=8 lg=6 List 卡片参考)
    - D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\components\shared\EmptyStateWithAction.tsx (14-05a 创建)
    - D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\components\shared\ErrorAlertWithRetry.tsx (14-05a 创建)
  </read_first>
  <action>
    1. history.tsx 改造:
       - import { Grid, List, Skeleton, Spin } from 'antd'; import { EmptyStateWithAction, ErrorAlertWithRetry } from '@/components/shared'
       - useBreakpoint() = Grid.useBreakpoint(),const isMobile = !!breakpoint.xs
       - 表格首次加载:isLoading 时渲染 Skeleton (3 行 paragraph={rows:3}) 占位,不渲染 Table
       - 错误:<ErrorAlertWithRetry error={error} onRetry={refetch} />
       - 空数据:<EmptyStateWithAction description=该范围内未采集到 MAC 记录,请检查设备是否启用了 MAC 采集/端口采集周期 actionLabel=前往设备管理 actionPath=/network/devices />
       - 移动端 isMobile === true 且 data.length > 0:用 List 渲染 (itemLayout=vertical),Card 内显示 firstSeen 时间 / MAC / 设备 / 端口 / 事件类型 Tag / VLAN,操作列同桌面
       - 桌面端 isMobile === false:继续渲染 14-01 的 AntD Table + virtual
       - isFetching 时(非首次),Table 组件加 Spin prop(spinning={isFetching})
       - **保留 14-04 注入的导出按钮**:executor 不得删除/重命名 "导出当前查询" / "导出全量" 两个按钮的 React 节点、prop 名称(`exportScope='current'|'all'`)、位置(工具栏查询/重置按钮之后);`handleExport` 函数与权限 `network:mac:export` 判断保留。
    2. trajectory.tsx 改造:
       - import { EmptyStateWithAction, ErrorAlertWithRetry } from '@/components/shared'
       - 替换原 trajectory.tsx:125-133 内联 Alert 为 <ErrorAlertWithRetry error={error} onRetry={refetch} />
       - 当 trajectoryData 为空数组:在 MACTrajectoryChart 上方渲染 <EmptyStateWithAction description=该 MAC 在此时间范围内无轨迹数据 actionLabel=查看事件时间线 actionPath=# (点击后 open Drawer,或保持空) />
  </action>
  <verify>
    <automated>cd "D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend" && npx tsc --noEmit -p . 2>&1 | head -30</automated>
  </verify>
  <acceptance_criteria>
    - history.tsx 含 EmptyStateWithAction 与 ErrorAlertWithRetry import 与渲染(grep 命中 ≥ 2 个名字)
    - history.tsx 含 Grid.useBreakpoint 与 isMobile 逻辑(grep 命中 ≥ 1 行)
    - history.tsx 含 Skeleton + isLoading 占位(grep 命中 ≥ 1 行)
    - history.tsx 移动端 List 渲染分支(grep `isMobile &&` 或 `breakpoint.xs` 命中 ≥ 1 行)
    - history.tsx **保留 14-04 注入的 "导出当前查询" 与 "导出全量" 两个按钮**(grep 命中 2 行)
    - history.tsx **保留 exportScope 变量引用**(grep `exportScope` 命中 ≥ 1 行)
    - trajectory.tsx 含 ErrorAlertWithRetry 替换原 Alert(grep 命中 ≥ 1 行)
    - npx tsc --noEmit 退出码 0
  </acceptance_criteria>
  <done>三态打磨完成,移动端卡片视图与桌面表格双形态自动切换,空/错误引导明确;14-04 注入的导出按钮 prop 名称与位置完整保留。</done>
</task>

<task type="auto">
  <name>Task 2: 网络设备列表页行内新增 查看 MAC 历史 联动按钮</name>
  <files>xingran-react-frontend/src/pages/network/devices/index.tsx</files>
  <read_first>
    - D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\pages\network\devices\index.tsx
  </read_first>
  <action>
    1. 在网络设备列表页 src/pages/network/devices/index.tsx 行操作列(action 列)添加 LinkButton:
       - 文字 查看 MAC 历史,icon = HistoryOutlined
       - onClick 调 useNavigate 跳 /network/mac/history?deviceId={record.id}&portName={record.selectedInterface} (portName 可选,设备列表行不携带具体端口,故只传 deviceId)
       - 权限控制 network:mac:list (与 history 页主权限点一致)
    2. 若设备详情页 detail.tsx 存在,同样在头部操作区添加该按钮 (使用 selectedInterface / portName 状态);若不存在,跳过。
    3. HistoryOutlined 已在 @ant-design/icons 中,无新增依赖。
  </action>
  <verify>
    <automated>cd "D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend" && npx tsc --noEmit -p . 2>&1 | head -30</automated>
  </verify>
  <acceptance_criteria>
    - devices/index.tsx 含 HistoryOutlined import(grep 命中 ≥ 1 行)
    - devices/index.tsx 含 查看 MAC 历史 文字(grep 命中 ≥ 1 行)
    - devices/index.tsx 含 navigate 跳 mac/history?deviceId=... 的逻辑(grep 命中 ≥ 1 行)
    - npx tsc --noEmit 退出码 0
  </acceptance_criteria>
  <done>网络设备模块与 MAC 历史模块建立入口联动,符合 D-16 锁定决策。</done>
</task>

</tasks>

<verification>
1. 用户访问 /network/mac/history:
   - 桌面端 表格 + virtual 渲染;移动端 (< 576px) 自动切 List 卡片
   - 首次加载显示 3 行 Skeleton 占位,之后分页切换不显示骨架,仅表头 Spin
   - total === 0 时显示 EmptyStateWithAction,带 前往设备管理 链接
   - 错误时显示 ErrorAlertWithRetry,错误码 1006/1007/500 文案分级
   - 工具栏右侧 "导出当前查询" / "导出全量" 两个按钮仍存在(14-04 注入代码未被覆盖)
2. 用户访问 /network/mac/trajectory:
   - 错误时 ErrorAlertWithRetry 替换内联 Alert
   - 数据为空时 EmptyStateWithAction 提示
3. 用户访问 /network/devices,行操作列含 查看 MAC 历史 按钮,点击跳 /network/mac/history?deviceId=xxx;14-01 列表页 useEffect 读取 URL 参数自动填入设备过滤项
4. npx tsc --noEmit 0 错误
</verification>

<success_criteria>
- [ ] UI-01 完整满足(含移动端响应式 + 三态)
- [ ] UI-02 完整满足(14-04 导出按钮在 14-05b 改造后保留,导出失败 ErrorAlertWithRetry 覆盖)
- [ ] UI-04 完整满足(时间线组件已在 14-01 落地,14-05b 仅三态打磨)
- [ ] D-16 网络设备联动入口生效
- [ ] D-18/19/20 三态打磨与 CONTEXT.md 锁定决策一致
- [ ] 桌面/移动 双形态自动切换
- [ ] 14-04 注入的导出按钮 prop 名称与位置在 14-05b 改造后完整保留
</success_criteria>

<output>
Create `.planning/phases/14-frontend-ux/14-05b-SUMMARY.md` when done
</output>
