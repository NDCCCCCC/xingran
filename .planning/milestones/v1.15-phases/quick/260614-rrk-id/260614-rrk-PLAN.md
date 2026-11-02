# 工位编辑模态框：所属楼层/所属用户 字段显示优化

**Mode**: quick
**Date**: 2026-06-14
**Slug**: 260614-rrk-id
**Scope**: 仅前端，工位管理页面的"编辑工位"模态框，不涉及后端

## 背景/问题

工位管理页面的"编辑工位"模态框 (`src/pages/operations/workstations/modals/EditModal.tsx`)
中有两个字段当前对用户不友好：

1. **所属楼层**（`floorId`）：后端存的是 UUID（楼层 ID），但前端 Cascader 应显示
   "楼栋名 → 楼层名"的级联名称（如"研发楼 / 3F"），提交时再回写为 `floorId` UUID。
   现状是 Cascader 选项 (`cascaderOptions`) 经常无法在编辑模态框打开时及时填好，
   导致 Cascader 回显为 UUID 字面量（甚至空白）。

2. **所属用户**（`userId`）：后端存的是 UUID（用户 ID），但前端下拉应显示
   `username(nickname)`（如 `zhangsan(张三)`），提交时存 `userId` UUID。
   现状 `useWorkstationData.loadUserOptions` 只把 `username` 写进 `name` 字段
   （`useWorkstationData.ts:124`），没有把 `nickname` 拼上去，且 Cascader 路径
   预加载依赖 `setTimeout(0)` 的竞态时机 (`index.tsx:363-369`)。

## 修复方案

### 1. 所属楼层 (Cascader)

- 现状：`floorId` 字段已经是 Cascader（楼栋→楼层），`handleWorkstationModalOk`
  已把 `floorId` 数组的最后一个元素（楼层 UUID）写回提交。
  (`index.tsx:208-210`)
- 问题：Cascader 选项 `cascaderOptions` 是父组件 `index.tsx` 的 state，编辑时
  父组件调用 `preloadCascaderPath` 后再 `setTimeout(0)` 写表单，但：
  - `preloadCascaderPath` (`index.tsx:319-331`) 只把匹配 `buildingId` 的楼宇填充
    `children`；其他楼宇 `children: undefined`，且 `loadData` 是按 `isLeaf: false`
    设计的，导致 Cascader 不会自动展开其他楼宇。
  - `setTimeout(0)` 在 React 18 自动批处理下不可靠。
  - 父组件 `setEditFormValues` (`index.tsx:335-380`) 通过 `floorApi.get` +
    `buildingApi.get` 串行获取两个 ID，再 setState + setTimeout，可读性差且脆弱。

**修复要点**：
- 编辑时改用一个**单次串行调用 `floorApi.get` → 拿到 `buildingId` →
  并行 `buildingApi.get` + `floorApi.list({buildingId})`** 的预加载流程。
- 把预加载得到的 `[buildingId, floorId]` 直接作为表单值，但**先**
  `setCascaderOptions(...)`，等 React 状态提交后**再** `form.setFieldsValue(...)`。
- 用一个 `useEffect` 监听 `cascaderOptions` 变化（搭配一个 `pendingFloorId` ref），
  当 Cascader 选项就绪后再写入表单，彻底去掉 `setTimeout(0)`。
- **不**触碰 Cascader 选项结构、loadData、`handleWorkstationModalOk` 的提交逻辑
  （已经正确地把数组最后一项作为 UUID 提交）。

### 2. 所属用户 (Select)

- 现状：`loadUserOptions` (`useWorkstationData.ts:116-130`) 调用
  `post('/system/users/list', params)`，把返回的 `list` 投影成
  `{ id, name: String(u.username) }`，`EditModal.tsx:137-140` 用 `<Option>{u.name}</Option>` 渲染。
- 后端 `User` 模型（`internal/models/user.go:8-43`）已经返回 `username` 和 `nickname`，
  前端只多读一个 `nickname` 即可。

**修复要点**：
- `loadUserOptions` 在映射时把 `nickname` 也带上，把 `UserOption` 扩展为
  `{ id, username, nickname? }`（`types.ts:27-30`）。
- `EditModal` 的 `<Select>` 渲染 `<Option label={`${username}${nickname ? `(${nickname})` : ''}`}>`
  （优先 username + nickname 的组合，缺 nickname 时只显示 username）。
- `value` 仍为 `id`（UUID），提交时 `userId` 字段保持原行为不变。
- 父组件 `index.tsx` 不用动（state 类型兼容即可）。

## 任务清单

### Task 1: 扩展 UserOption 类型 + loadUserOptions 携带 nickname

**Files**:
- `xingran-react-frontend/src/pages/operations/workstations/types.ts`
- `xingran-react-frontend/src/pages/operations/workstations/hooks/useWorkstationData.ts`

**Action**:
1. `types.ts:27-30` 把 `UserOption` 改为：
   ```ts
   export interface UserOption {
     id: string;
     username: string;
     nickname?: string;
   }
   ```
2. `useWorkstationData.ts:122-126` 的 `users.map` 改为：
   ```ts
   users.map((u: Record<string, unknown>) => ({
     id: String(u.id),
     username: String(u.username),
     nickname: u.nickname ? String(u.nickname) : undefined,
   }))
   ```
3. 保留 `name` 字段（如有外部依赖），或在确认 `EditModal.tsx` 已迁移到新结构后
   删除（推荐直接删除以避免类型漂移）。

**Verify**:
- `cd xingran-react-frontend && npx tsc --noEmit` 0 错误（type-check:strict）
- `npm run lint` 0 错误

**Done**:
- `UserOption` 新结构对所有调用方类型安全
- `loadUserOptions` 返回的数组元素带 `username` + `nickname`（后者可能为 `undefined`）

### Task 2: 所属用户 Select 渲染 `username(nickname)`（提交仍存 UUID）

**Files**:
- `xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx`

**Action**:
1. `EditModal.tsx:31` 把 `userOptions` 的 prop 类型从
   `{ id: string; name: string }[]` 改为 `UserOption[]`（从 `'../types'` 导入）。
2. `EditModal.tsx:135-141` 的 `<Select>` 改为：
   ```tsx
   <Select placeholder="请先选择部门" allowClear showSearch optionFilterProp="children">
     {userOptions.map((u) => (
       <Option key={u.id} value={u.id}>
         {u.nickname ? `${u.username} (${u.nickname})` : u.username}
       </Option>
     ))}
   </Select>
   ```
3. `value` 仍是 `u.id`（UUID），提交不变。

**Verify**:
- `npm run type-check:strict` 通过
- `npm run lint` 通过

**Done**:
- 编辑工位模态框的"所属用户"下拉每一项渲染为 `username (nickname)`，
  缺 nickname 时只显示 username；提交仍存 UUID。

### Task 3: 编辑回显时可靠预加载 Cascader 路径（替换 setTimeout 0 竞态）

**Files**:
- `xingran-react-frontend/src/pages/operations/workstations/index.tsx`

**Action**:
1. `index.tsx:319-331` 的 `preloadCascaderPath` 改造为接收
   `(orgId, buildingId)`，先把 buildings 列表注入 `cascaderOptions`（即使没有 children），
   然后**显式**为匹配楼宇附加 `children`（`loadFloorsForCascader` 结果）。
2. `index.tsx:335-380` 的 `setEditFormValues` 改造为：
   - 调用 `floorApi.get(record.floorId)` 拿 `buildingId`。
   - 调 `buildingApi.get(buildingId)` 拿 `orgId`（如需 orgId 联动）。
   - 调用 `preloadCascaderPath(orgId, buildingId)` 完成 cascaderOptions 设置。
   - 用一个 `useEffect`（依赖 `[cascaderOptions, record?.id]`）来写表单值：
     当 `record.id` 存在且 `cascaderOptions` 非空且包含 `buildingId` 选项时，
     调用 `workstationForm.setFieldsValue({ ...record, floorId: [buildingId, record.floorId] })`。
   - **删除** `setTimeout(() => ..., 0)` 写法。
3. 新增一个 `useRef<{recordId: string; buildingId: string; floorId: string} | null>(null)`
   暂存"等待 cascaderOptions 就绪后写入"的待办值，effect 消费后清空。

**Verify**:
- `npm run type-check:strict` 通过
- `npm run lint` 通过
- 打开任一工位 → 点"编辑" → "所属楼层"Cascader 应正确显示"楼栋名 / 楼层名"路径，
  提交后 `floorId` 仍为楼层 UUID。
- 打开任一工位 → 点"编辑" → "所属用户"Select 应正确显示 `username (nickname)`，
  提交后 `userId` 仍为用户 UUID。

**Done**:
- 编辑模式下打开模态框，"所属楼层"Cascader 不再出现 UUID 字面量或空白。
- 不依赖 `setTimeout(0)` 之类的脆弱时序；`useEffect` + ref 取代。

## 影响文件清单

- `xingran-react-frontend/src/pages/operations/workstations/types.ts` (UserOption 扩展)
- `xingran-react-frontend/src/pages/operations/workstations/hooks/useWorkstationData.ts`
  (loadUserOptions 投影 nickname)
- `xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx`
  (userOptions 类型 + Select 渲染 + 透传 cascader/floor 显示)
- `xingran-react-frontend/src/pages/operations/workstations/index.tsx`
  (setEditFormValues / preloadCascaderPath 重构 + useEffect 等待 cascaderOptions)

## 验证步骤

1. `cd xingran-react-frontend && npm run type-check:strict` — 0 错误
2. `cd xingran-react-frontend && npm run lint` — 0 错误
3. `cd xingran-react-frontend && npm run build` — 构建通过
4. 浏览器/视觉确认（dev server `npm run dev`）：
   - 新增工位：选 orgId → 选楼栋 → 选楼层 → 选部门 → 选用户（应显示 `账号(昵称)`）→ 保存。
   - 编辑工位：打开任一行 → "所属楼层" Cascader 应显示"楼栋名 / 楼层名"，
     "所属用户" Select 应显示 `账号(昵称)`，保存后字段值仍为 UUID。
   - 不传部门时编辑：用户下拉应仍为列表（loadUserOptions 默认无 deptId）。

## 不涉及

- 后端 Go 代码（`/system/users/list` 已返回 `nickname`，`/system/departments/tree`、
  `buildingApi.list`/`get`、`floorApi.list`/`get` 均不变）。
- Excel 导入流程（`ExcelImport.tsx`、`excel_service.go`）— 已是"按名称匹配→存 ID"模式。
- 工位列表/详情/卡片/平面图视图。
- 字段增删、状态字段语义（0=空闲 1=占用 2=维护）、搜索筛选区。
- Vite `manualChunks` 配置（已知：React + Ant Design 不能拆独立 chunk，会
  `createContext/useLayoutEffect undefined`）。
- `workstationApi.create` / `update` 的请求体契约（仍 `{ floorId, userId, ... }` UUID）。
- Excel 导入的合并（260614-dpz）逻辑。
