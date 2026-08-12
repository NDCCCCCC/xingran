---
slug: rpa-page-black-bg-leak
status: resolved
trigger: RPA管理似乎和其他页面布局不一样，请检查 (用户URL: http://127.0.0.1:4000/vdi/rpa)
created: 2026-06-16T00:00:00+08:00
updated: 2026-06-16T00:00:00+08:00
session_type: bug
---

# Debug Session: RPA 页面黑色背景透出

## Symptoms

### Expected Behavior
RPA 管理页面布局应与 `operations/workstations`（工位管理）保持一致：L1 外层黑框 + L2 内层白色内容区。黑色背景只作为最外层框架存在，不应透出到卡片之间/之外的空白处。

### Actual Behavior
**RPA 子页面（tasks / executions / workers）**的外层 `<div>` 设置了 `background: '#000'`，但缺少 workstations 那样的 L2 白色内容包装层（`<Layout>` + `<Content style={{ background: '#fff' }}>`）。导致黑色直接透出，表现为：
- 卡片之间/之外的空白处是**黑色**而非白色
- 用户反馈："背景色是黑色，或者说背景色显示了出来，其他页面的黑色背景色没有显示出来"

### Affected Pages
- `xingran-react-frontend/src/pages/operations/rpa/tasks/index.tsx`（行 137）
- `xingran-react-frontend/src/pages/operations/rpa/executions/index.tsx`（行 90）
- `xingran-react-frontend/src/pages/operations/rpa/workers/index.tsx`（行 108）

注：用户报告的 URL `http://127.0.0.1:4000/vdi/rpa` 与文件实际位置 `pages/operations/rpa/` 不一致（前端路径由后端菜单 API 动态生成，路径与组件名解耦），症状仍然精确指向 3 个子页面。

### Error Messages
无控制台报错；属于纯视觉/样式不一致。

### Timeline
- 2026-06-16 15:32 commit `94d5654` — `style(operations,network): apply workstations final layout (black/white two-tone) to 10 sibling pages`：
  - 把 workstations 的 "L1 黑 + L2 白" 模式应用到了 10 个兄弟页面
  - A 类（6 个文件，使用 `antd Layout + Content`）：替换 L0 颜色
  - **B 类（4 个文件，使用单一 `<div>` 容器）**：`operations/dedicated-lines/index.tsx` + **3 个 RPA 子页面**（executions/tasks/workers）— 仅把 `<div>` 的 `bg='#f0f2f5'` 改成 `bg='#000'`，**没有补一个内层白色 L2 包装**，所以黑色直接漏出
- 用户随即报"RPA 页面布局不一样" → 即本次 debug

## Current Focus

- **hypothesis**: 94d5654 commit 对 B 类（4 个文件）只换了 L0 颜色，没补 L2 白色包装层；其中 dedicated-lines 单页面可能未触发用户视觉路径（用户截图视点停在 RPA），但 3 个 RPA 子页面因为被父级 `RpaManagement` 的 `<Card variant="borderless">` 包住，黑底作为可见的"内层条带"出现
- **next_action**: 验证 hypothesis 两种修复路径：
  1. 最小修复：把 3 个 RPA 子页面外层 `<div>` 的 `background: '#000'` 改为 `background: '#fff'`（与 dedicated-lines 当前的样式不符，但与 workstations 的 L2 等价）
  2. 结构修复：把外层 `<div>` 替换为 `<Layout style={{ background: '#000' }}>` + `<Content style={{ background: '#fff', padding: 16 }}>`，与 workstations / A 类 6 个文件完全一致
- **test**: 在 3 个 RPA 子页面文件中肉眼对比（卡片外应该白色、整页最外侧才是黑色）
- **expecting**: 选结构修复 (路径 2)，理由：与 94d5654 的设计意图（"replicate the workstations page final visual"）完全对齐；dedicated-lines 顺带归并

## Evidence

- timestamp: 2026-06-16
  source: git
  finding: |
    Commit 94d5654 message 显式说明 "B 类 (single <div> container, 4 files):" 包括
    - operations/dedicated-lines/index.tsx
    - operations/rpa/executions/index.tsx
    - operations/rpa/tasks/index.tsx
    - operations/rpa/workers/index.tsx
    改动是 `<div padding bg='#f0f2f5'>` → `padding bg='#000'`，只改了一行 style。
  files:
    - xingran-react-frontend/src/pages/operations/rpa/tasks/index.tsx
    - xingran-react-frontend/src/pages/operations/rpa/executions/index.tsx
    - xingran-react-frontend/src/pages/operations/rpa/workers/index.tsx

- timestamp: 2026-06-16
  source: code_read
  finding: |
    Workstations（参考标准）:
    ```tsx
    <Layout style={{ background: '#000', minHeight: 'calc(100vh - 64px)' }}>
      <DeptSidebar ... />
      <Content style={{ background: '#fff' }}>
        ... cards ...
      </Content>
    </Layout>
    ```
    L1 黑 (Layout) + L2 白 (Content) 嵌套，黑色被 L2 完全遮盖。

    RPA tasks 子页面（问题代码）:
    ```tsx
    <div style={{ padding: 16, background: '#000', minHeight: 'calc(100vh - 64px)' }}>
      <Card>...</Card>
      <Card>...</Card>
    </div>
    ```
    L1 黑 div 直接包白 Card，卡片间/外空白处露出黑色。
  files:
    - xingran-react-frontend/src/pages/operations/workstations/index.tsx
    - xingran-react-frontend/src/pages/operations/rpa/tasks/index.tsx

- timestamp: 2026-06-16
  source: code_read
  finding: |
    RPA 父级 index.tsx（不带黑底，因为它已经是 Card variant="borderless"）:
    ```tsx
    <div style={{ padding: 0, height: '100%' }}>
      <Card variant="borderless" style={{ height: '100%' }}>
        <Tabs items={[
          { children: <TaskManagement /> },
          { children: <ExecutionManagement /> },
          { children: <WorkerMonitor /> },
        ]} />
      </Card>
    </div>
    ```
    父级 Card 是白色，3 个 TaskManagement/ExecutionManagement/WorkerMonitor 在 Tab 内渲染时，
    它们的 `<div bg='#000'>` 在白 Card 内部形成"黑底白卡片"的视觉异常。
  files:
    - xingran-react-frontend/src/pages/operations/rpa/index.tsx

## Resolution

- **root_cause**: commit `94d5654` 在把 workstations 的 L1 黑 + L2 白双层布局推广到 10 个兄弟页时，对 4 个 B 类（单一 `<div>` 容器）文件**只把外层 `<div>` 的 `bg='#f0f2f5'` 改成了 `bg='#000'`**，没补 L2 白色 `<Content>` 包装层。结果黑色直接透出到卡片间/外侧的空白处，与 workstations 视觉不一致。其中 3 个 RPA 子页面（tasks / executions / workers）因为被父级 `<Card variant="borderless">` 包裹，黑底作为可见的"内层条带"出现，对比更刺眼。
- **fix**: 对 4 个 B 类文件统一补 L2 包装，结构与 A 类 6 文件 + workstations 完全一致：
  - 外层 `<div padding bg='#000'>` → `<Layout style={{ background: '#000', minHeight: 'calc(100vh - 64px)' }}>`
  - 新增内层 `<Content style={{ background: '#fff', padding: 16 }}>` 包住所有原 `<div>` 内的 children
  - antd import 列表追加 `Layout`；文件顶部追加 `const { Content } = Layout;`
- **verification**:
  - `npm run type-check` → 0 errors（与 commit 94d5654 当时验证标准一致）
  - `git diff --stat` → 4 files changed, 24 insertions(+), 12 deletions(-) — 每个文件 6+/3- 净增 3 行，符合预期
  - 结构 diff 抽样（`rpa/workers/index.tsx`）：`+ Layout` import, `+ const { Content } = Layout;`, 外层 div → Layout/Content 嵌套，闭合 `</div>` → `</Content></Layout>`
  - 视觉效果预期：L1 黑框（外侧 antd Layout） + L2 白内容区（Content 覆盖所有内部） + 内部白 Card（搜索栏、表格）— 与 workstations 完全一致
- **files_changed**:
  - `xingran-react-frontend/src/pages/operations/dedicated-lines/index.tsx` (9 行：+6 -3)
  - `xingran-react-frontend/src/pages/operations/rpa/tasks/index.tsx` (9 行：+6 -3)
  - `xingran-react-frontend/src/pages/operations/rpa/executions/index.tsx` (9 行：+6 -3)
  - `xingran-react-frontend/src/pages/operations/rpa/workers/index.tsx` (9 行：+6 -3)
