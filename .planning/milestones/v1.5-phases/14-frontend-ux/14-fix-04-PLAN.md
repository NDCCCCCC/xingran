---
phase: 14-frontend-ux
plan: fix-04
type: execute
wave: 3
depends_on:
  - 14-fix-01
  - 14-fix-02
  - 14-fix-03
gap_closure: true
gap_refs:
  - W2
  - W5
  - W6
  - W7
files_modified:
  - xingran-react-frontend/src/components/network/macEventMeta.ts
  - xingran-react-frontend/src/components/network/MACEventsTimeline.tsx
  - xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx
  - xingran-react-frontend/src/components/network/index.ts
  - xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx
  - xingran-react-frontend/src/pages/network/mac/history/index.tsx
  - xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx
  - xingran-react-frontend/src/pages/network/mac/trajectory/index.tsx
  - xingran-react-frontend/src/pages/network/mac/history.tsx
  - xingran-react-frontend/src/pages/network/mac/trajectory.tsx
  - xingran-react-frontend/src/lib/api/networkApi.ts
autonomous: true
requirements:
  - UI-01
  - UI-04

must_haves:
  truths:
    - "macEventMeta.ts 是 EVENT_COLORS / EVENT_ICON / EVENT_LABEL 的单一事实源,被 MACTrajectoryChart / MACEventsTimeline / history(MACHistoryPage)/ trajectory(TrajectoryPage) 4 处共同 import,不再有重复字面量"
    - "MACEventsTimeline 错误态使用共享 ErrorAlertWithRetry 组件(refetch 触发重试)"
    - "getMACEvents 当 result.data.total > pageSize 时,console.warn 输出提示(后端日志可见),不静默丢弃"
    - "mac/history.tsx 与 mac/history/index.tsx 合并为 mac/history/ 目录(内含 MACHistoryPage.tsx + index.tsx);mac/trajectory 同理"
  artifacts:
    - path: "xingran-react-frontend/src/components/network/macEventMeta.ts"
      provides: "事件类型单一事实源"
      contains: "EVENT_COLORS"
    - path: "xingran-react-frontend/src/components/network/MACEventsTimeline.tsx"
      provides: "import 共用 macEventMeta + ErrorAlertWithRetry 错误态"
      contains: "macEventMeta|ErrorAlertWithRetry"
    - path: "xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx"
      provides: "删除本地 EVENT_COLORS,改 import macEventMeta"
      contains: "macEventMeta"
    - path: "xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx"
      provides: "原 history.tsx 内容搬入,删除本地 EVENT_TAG_COLORS / EVENT_TYPE_LABELS"
      contains: "macEventMeta"
    - path: "xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx"
      provides: "原 trajectory.tsx 内容搬入"
      contains: "TrajectoryPage"
    - path: "xingran-react-frontend/src/pages/network/mac/history/index.tsx"
      provides: "re-export from MACHistoryPage"
      contains: "MACHistoryPage"
    - path: "xingran-react-frontend/src/pages/network/mac/trajectory/index.tsx"
      provides: "re-export from TrajectoryPage"
      contains: "TrajectoryPage"
  key_links:
    - from: "xingran-react-frontend/src/components/network/MACEventsTimeline.tsx"
      to: "xingran-react-frontend/src/components/network/macEventMeta.ts"
      via: "import { EVENT_COLORS, EVENT_ICON, EVENT_LABEL } from './macEventMeta'"
      pattern: "from './macEventMeta'"
    - from: "xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx"
      to: "xingran-react-frontend/src/components/network/macEventMeta.ts"
      via: "同上"
      pattern: "from './macEventMeta'"
    - from: "xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx"
      to: "xingran-react-frontend/src/components/network/macEventMeta.ts"
      via: "同上(替换本地 EVENT_TAG_COLORS / EVENT_TYPE_LABELS)"
      pattern: "from '@/components/network/macEventMeta'"
    - from: "xingran-react-frontend/src/pages/network/mac/history/index.tsx"
      to: "xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx"
      via: "export { default } from './MACHistoryPage'"
      pattern: "MACHistoryPage"
    - from: "xingran-react-frontend/src/lib/api/networkApi.ts"
      to: "/network/history/list"
      via: "getMACEvents 用 total 判断 truncation"
      pattern: "getMACEvents"
---

<objective>
收口 4 个 WARNING 级别的前端质量问题,统一事件元数据 + 修复重复文件 + 修复分页 truncation 静默丢失 + 使用共享错误组件。

Purpose:
- **W7 (IN-04)**: EVENT_COLORS / EVENT_ICON / EVENT_LABEL 散落在 MACTrajectoryChart / MACEventsTimeline / history.tsx / trajectory.tsx 4 处,D-10 锁定的颜色体系依赖"各文件字面量碰巧一致",一处修改会与其他处漂移。集中到 `macEventMeta.ts` 后,只要改一处即可。
- **W5 (WR-04)**: MACEventsTimeline 的错误态用 `<Empty description="加载失败" />` 偷懒,无重试 — 改用 ErrorAlertWithRetry(已实现,直接复用)。
- **W6 (WR-03)**: getMACEvents 静默 `pageSize: 100` 上限,>100 条事件时无任何用户提示 — 增加 total > pageSize 时 console.warn(后端日志可见)。
- **W2 (CR-05)**: mac/history.tsx + mac/history/index.tsx 双源,mac/trajectory.tsx + mac/trajectory/index.tsx 双源,合并为目录结构。

Output: 1 个新文件(macEventMeta.ts) + 1 个 MACHistoryPage.tsx 重构 + 1 个 TrajectoryPage.tsx 重构 + 2 个 index.tsx 更新 + 2 个原文件删除 + 3 个原文件内部去重。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/14-frontend-ux/14-CONTEXT.md (D-10 锁定颜色体系;D-09 锁定跨页复用组件)
@.planning/phases/14-frontend-ux/14-VERIFICATION.md (W2 + W5 + W6 + W7 段)
@.planning/phases/14-frontend-ux/14-REVIEW.md (CR-05 + WR-03 + WR-04 + IN-04 段)
@xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx (行 42-47 EVENT_COLORS 本地)
@xingran-react-frontend/src/components/network/MACEventsTimeline.tsx (行 29-48 EVENT_COLORS + EVENT_META 本地;行 119-125 错误态)
@xingran-react-frontend/src/pages/network/mac/history.tsx (行 71-82 EVENT_TAG_COLORS / EVENT_TYPE_LABELS 本地)
@xingran-react-frontend/src/pages/network/mac/trajectory.tsx (颜色引用如存在)
@xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx (复用模式参考)
@xingran-react-frontend/src/lib/api/networkApi.ts (getMACEvents 行 106-122)
@xingran-react-frontend/src/lib/hooks/useTableQuery.ts (placeholderData: keepPreviousData 模式参考)
</context>

<tasks>

<task type="auto">
  <name>Task 1: 集中 macEventMeta + 使用 ErrorAlertWithRetry + truncation 提示 + 目录合并(覆盖 4 个 W)</name>
  <files>xingran-react-frontend/src/components/network/macEventMeta.ts (NEW), xingran-react-frontend/src/components/network/MACEventsTimeline.tsx, xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx, xingran-react-frontend/src/components/network/index.ts, xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx (NEW), xingran-react-frontend/src/pages/network/mac/history/index.tsx (UPDATE), xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx (NEW), xingran-react-frontend/src/pages/network/mac/trajectory/index.tsx (UPDATE), xingran-react-frontend/src/pages/network/mac/history.tsx (DELETE), xingran-react-frontend/src/pages/network/mac/trajectory.tsx (DELETE), xingran-react-frontend/src/lib/api/networkApi.ts</files>
  <read_first>
    - xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx (本地 EVENT_COLORS 行 42-47)
    - xingran-react-frontend/src/components/network/MACEventsTimeline.tsx (本地 EVENT_COLORS + EVENT_META 行 29-48;错误态行 119-125)
    - xingran-react-frontend/src/pages/network/mac/history.tsx (本地 EVENT_TAG_COLORS / EVENT_TYPE_LABELS 行 71-82;完整 page 实现约 770 行)
    - xingran-react-frontend/src/pages/network/mac/trajectory.tsx (完整 page 实现约 340 行;颜色引用检查)
    - xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx (复用接口 props.error / props.onRetry)
    - xingran-react-frontend/src/lib/api/networkApi.ts (getMACEvents 行 106-122;MACHistoryPageResult type 行 79-84)
    - xingran-react-frontend/src/components/network/index.ts (barrel 当前导出 MACEventsTimeline + MACTrajectoryChart)
  </read_first>
  <action>
    1. **W7 — 创建 macEventMeta.ts**:
       新建 `xingran-react-frontend/src/components/network/macEventMeta.ts`:
       ```ts
       /**
        * MAC 事件类型元数据(D-10 锁定单一事实源)
        * - 颜色:appeared=green,disappeared=red,moved=gold,vlan_changed=blue
        * - 图标:PlusCircleOutlined / MinusCircleOutlined / SwapOutlined / TagOutlined
        * - 中文标签:出现 / 消失 / 迁移 / VLAN 变更
        *
        * 被以下文件 import:
        * - components/network/MACTrajectoryChart.tsx
        * - components/network/MACEventsTimeline.tsx
        * - pages/network/mac/history/MACHistoryPage.tsx
        * - pages/network/mac/trajectory/TrajectoryPage.tsx
        */
       import {
         PlusCircleOutlined,
         MinusCircleOutlined,
         SwapOutlined,
         TagOutlined,
       } from '@ant-design/icons';
       import type { ComponentType, CSSProperties } from 'react';

       export type MACEventType = 'appeared' | 'disappeared' | 'moved' | 'vlan_changed';

       export const EVENT_COLORS: Record<MACEventType, string> = {
         appeared: '#52c41a',
         disappeared: '#ff4d4f',
         moved: '#faad14',
         vlan_changed: '#1890ff',
       };

       export const EVENT_ICON: Record<MACEventType, ComponentType<{ style?: CSSProperties }>> = {
         appeared: PlusCircleOutlined,
         disappeared: MinusCircleOutlined,
         moved: SwapOutlined,
         vlan_changed: TagOutlined,
       };

       export const EVENT_LABEL: Record<MACEventType, string> = {
         appeared: '出现',
         disappeared: '消失',
         moved: '迁移',
         vlan_changed: 'VLAN 变更',
       };

       /** AntD Tag color (与 ECharts hex 兼容) */
       export const EVENT_TAG_COLOR: Record<MACEventType, string> = {
         appeared: 'green',
         disappeared: 'red',
         moved: 'gold',
         vlan_changed: 'blue',
       };
       ```
    2. **W7 — 更新 MACTrajectoryChart.tsx**:
       - 删除行 42-47 `EVENT_COLORS` 本地定义。
       - 顶部加 `import { EVENT_COLORS, type MACEventType } from './macEventMeta';`
       - 替换所有 `EVENT_COLORS.xxx` 引用为 import 后的版本(图表本身只用 appeared/moved/disappeared/vlan_changed 的颜色,故仅 EVENT_COLORS 一处)。
       - 注意:`TrajectoryNode.eventType` 字段若与 MACEventType 不一致,在导入后用 `as MACEventType` cast 收敛类型。
    3. **W7 + W5 — 更新 MACEventsTimeline.tsx**:
       - 删除行 29-48 的本地 `EVENT_COLORS` + `EVENT_META`。
       - 顶部加入:`import { EVENT_COLORS, EVENT_ICON, EVENT_LABEL } from './macEventMeta';` + `import { ErrorAlertWithRetry } from '@/components/shared';`(ErrorAlertWithRetry 已在 shared/index.ts 导出)。
       - 内部用 `const Icon = EVENT_ICON[event.eventType];` + `const label = EVENT_LABEL[event.eventType];` + `const color = EVENT_COLORS[event.eventType];` 替换原来 `EVENT_META[event.eventType]`。
       - Tag color 用 `EVENT_TAG_COLOR[event.eventType]`。
       - **W5 修复**:行 119-125 错误态分支替换为:
         ```tsx
         if (error) {
           return (
             <Card size="small" title={`MAC 事件时间线 — ${mac}`} bordered={false}>
               <ErrorAlertWithRetry error={error as Error} onRetry={() => { void refetch(); }} />
             </Card>
           );
         }
         ```
       - 顶部 useQuery 解构改为 `const { data: events, isLoading, error, refetch } = useQuery({...});`。
    4. **W2 — 把 history.tsx 内容搬到 MACHistoryPage.tsx**:
       - 用 Read 完整读取 `xingran-react-frontend/src/pages/network/mac/history.tsx`(约 770 行)。
       - 创建 `xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx`,内容除以下几点外与原文件相同:
         * 删除行 71-82 的本地 `EVENT_TAG_COLORS` + `EVENT_TYPE_LABELS`,改为 `import { EVENT_LABEL, EVENT_TAG_COLOR } from '@/components/network/macEventMeta';`。
         * 所有 `EVENT_TAG_COLORS[xxx]` 替换为 `EVENT_TAG_COLOR[xxx]`,所有 `EVENT_TYPE_LABELS[xxx]` 替换为 `EVENT_LABEL[xxx]`。
         * 组件名 `MACHistoryPage` 保持(已是该名字)。
       - 更新 `xingran-react-frontend/src/pages/network/mac/history/index.tsx` 为 `export { default } from './MACHistoryPage';`(原是 `export { default } from '../history';`)。
       - 删除原 `xingran-react-frontend/src/pages/network/mac/history.tsx` 文件(`rm` 或 Git 删除)。
    5. **W2 — 把 trajectory.tsx 内容搬到 TrajectoryPage.tsx**:
       - 用 Read 完整读取 `xingran-react-frontend/src/pages/network/mac/trajectory.tsx`。
       - 创建 `xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx`,内容与原文件相同(如有本地 EVENT_COLORS/EVENT_LABEL,改 import macEventMeta;否则原样搬迁)。
       - 更新 `xingran-react-frontend/src/pages/network/mac/trajectory/index.tsx` 为 `export { default } from './TrajectoryPage';`。
       - 删除原 `xingran-react-frontend/src/pages/network/mac/trajectory.tsx` 文件。
    6. **W7 — 更新 components/network/index.ts**:
       - 添加 `export { EVENT_COLORS, EVENT_ICON, EVENT_LABEL, EVENT_TAG_COLOR } from './macEventMeta';` + `export type { MACEventType } from './macEventMeta';`,方便外部 import。
    7. **W6 — getMACEvents truncation 提示**:
       - 在 `xingran-react-frontend/src/lib/api/networkApi.ts` 行 106-122 的 `getMACEvents` 函数中:
         ```ts
         export const getMACEvents = async (
           mac: string,
           startTime: string,
           endTime: string,
         ): Promise<MACHistoryRecord[]> => {
           const result = await post<MACHistoryPageResult>('/network/history/list', {
             mac,
             startTime,
             endTime,
             current: 1,
             pageSize: 100,
           });
           const list = result.data?.list ?? [];
           const total = result.data?.total ?? 0;
           if (total > list.length) {
             console.warn(`[getMACEvents] 事件被截断:total=${total}, returned=${list.length}。前端 pageSize=100 上限。考虑添加 sort 扩展或扩大分页。`);
           }
           return list.slice().sort(
             (a, b) =>
               new Date(b.firstSeen).getTime() - new Date(a.firstSeen).getTime()
           );
         };
         ```
       - 保留 `slice().sort()` 行为(虽然 backend 已经 ORDER BY DESC,但前端兜底仍无副作用)。
    8. 验证:`cd xingran-react-frontend && npx tsc --noEmit -p .` 退出码 0;若 React 路由解析报错,确认 Vite glob(`src/pages/**/*.tsx`)仍能 pick 到 `history/index.tsx` 的 re-export。
  </action>
  <verify>
    <automated>cd xingran-react-frontend && npx tsc --noEmit -p .  # 退出码 0</automated>
    <automated>test -f xingran-react-frontend/src/components/network/macEventMeta.ts  # 必须存在</automated>
    <automated>test -f xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx  # 必须存在</automated>
    <automated>test -f xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx  # 必须存在</automated>
    <automated>test ! -f xingran-react-frontend/src/pages/network/mac/history.tsx  # 必须不存在(已删除)</automated>
    <automated>test ! -f xingran-react-frontend/src/pages/network/mac/trajectory.tsx  # 必须不存在(已删除)</automated>
    <automated>grep -c "from './macEventMeta'\|from '@/components/network/macEventMeta'" xingran-react-frontend/src/components/network/MACEventsTimeline.tsx xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx  # 至少 3 个文件 import macEventMeta</automated>
    <automated>grep -c "EVENT_TAG_COLORS\|EVENT_TYPE_LABELS" xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx  # 必须为 0(已删除本地版本)</automated>
    <automated>grep -c "ErrorAlertWithRetry" xingran-react-frontend/src/components/network/MACEventsTimeline.tsx  # 至少 1</automated>
    <automated>grep -n "console.warn.*截断\|console.warn.*truncation\|console.warn.*getMACEvents" xingran-react-frontend/src/lib/api/networkApi.ts  # 至少 1(W6 提示)</automated>
    <automated>grep -c "EVENT_COLORS = {" xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx  # 必须为 0(已删除本地)</automated>
  </verify>
  <acceptance_criteria>
    - `npx tsc --noEmit -p .` 退出码 0
    - macEventMeta.ts 存在并导出 EVENT_COLORS / EVENT_ICON / EVENT_LABEL / EVENT_TAG_COLOR / MACEventType
    - mac/history/MACHistoryPage.tsx + mac/history/index.tsx 存在;原 history.tsx 已删除
    - mac/trajectory/TrajectoryPage.tsx + mac/trajectory/index.tsx 存在;原 trajectory.tsx 已删除
    - MACEventsTimeline.tsx 顶部 import macEventMeta + ErrorAlertWithRetry
    - MACEventsTimeline 错误态分支渲染 `<ErrorAlertWithRetry error={error as Error} onRetry={() => refetch()} />`
    - MACTrajectoryChart.tsx 删除本地 EVENT_COLORS 字面量
    - MACHistoryPage.tsx 删除本地 EVENT_TAG_COLORS / EVENT_TYPE_LABELS 字面量
    - getMACEvents 在 total > list.length 时输出 console.warn 含"截断"或"truncation"字样
    - 路由 /network/mac/history 与 /network/mac/trajectory 仍可访问(Vite glob pick up index.tsx)
    - 4 个文件 import macEventMeta(MACTrajectoryChart + MACEventsTimeline + MACHistoryPage + components/network/index.ts barrel)
    - git status 显示 mac/history.tsx 与 mac/trajectory.tsx 为 deleted,新增 MACHistoryPage.tsx / TrajectoryPage.tsx
  </acceptance_criteria>
  <done>4 个 WARNING 全部收口:macEventMeta 单一事实源就位 + ErrorAlertWithRetry 在 timeline 错误态复用 + getMACEvents truncation 控制台提示 + mac/history & mac/trajectory 合并为目录结构;TypeScript 编译 0 退出码;文件不存在/存在的反向 grep 全部通过。</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| component → shared meta | macEventMeta imported by 4 files — single source of truth for D-10 lock |
| component → ErrorAlertWithRetry | cross-component reuse of error UX with retry callback |
| page → router | mac/history/index.tsx + mac/trajectory/index.tsx — Vite glob picks up single source |
| API → console | getMACEvents logs truncation to console for ops visibility |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-14W2-01 | Tampering | duplicate history.tsx + history/index.tsx | mitigate | Deleted sibling .tsx; only MACHistoryPage.tsx in history/ directory remains |
| T-14W5-01 | Elevation of Privilege | timeline error UX | mitigate | ErrorAlertWithRetry provides refetch callback; previously silent Empty was unrecoverable |
| T-14W6-01 | Information Disclosure | getMACEvents truncation | accept | console.warn is informational; backend ORDER BY DESC means most-recent events preserved; user can extend time range to see more |
| T-14W7-01 | Repudiation | EVENT_COLORS drift across files | mitigate | Single macEventMeta.ts source; grep verifies no local EVENT_COLORS literal in MACTrajectoryChart / MACHistoryPage |
| T-14W2-SC | Tampering | npm/pip/cargo installs | mitigate | No new dependencies — file moves + import path consolidation |
</threat_model>

<verification>
- `cd D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend && npx tsc --noEmit -p .` exits 0
- `test -f xingran-react-frontend/src/components/network/macEventMeta.ts` succeeds
- `test -f xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx` succeeds
- `test -f xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx` succeeds
- `test ! -f xingran-react-frontend/src/pages/network/mac/history.tsx` succeeds (deleted)
- `test ! -f xingran-react-frontend/src/pages/network/mac/trajectory.tsx` succeeds (deleted)
- `grep -l "from './macEventMeta'\|from '@/components/network/macEventMeta'"` matches >= 3 files
- `grep -c "EVENT_COLORS = {" xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx` == 0
- `grep -c "ErrorAlertWithRetry" xingran-react-frontend/src/components/network/MACEventsTimeline.tsx` >= 1
- End-to-end smoke (out-of-band): start dev server, navigate to /network/mac/history, expand a row, verify timeline loads; trigger an error in DevTools network throttle, verify ErrorAlertWithRetry renders with retry button
</verification>

<success_criteria>
- macEventMeta.ts 是 4 个文件中所有事件元数据的唯一来源
- mac/history 目录结构 + mac/trajectory 目录结构,无平级 .tsx + index.tsx 双源
- MACEventsTimeline 错误态使用 ErrorAlertWithRetry + 提供 refetch 重试
- getMACEvents 在 total > pageSize 时输出 truncation 提示
- TypeScript 编译 0 退出码
- 路由 /network/mac/history 与 /network/mac/trajectory 仍可访问
- Vite 启动后所有组件正确渲染
</success_criteria>

<output>
Create `.planning/phases/14-frontend-ux/14-fix-04-SUMMARY.md` when done
</output>