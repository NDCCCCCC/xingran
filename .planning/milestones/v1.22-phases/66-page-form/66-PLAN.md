---
phase: "66"
slug: "66-page-form"
type: plan
created: "2026-10-25"
planner: gsd-planner
---

# Phase 66 执行计划 — 高级契约组件（PageTable 与 FormModal）

**目标**：交付 3 个新契约组件（PageTable / FormModal / RemoteSelect）+ 1 套 lint 增量门 + system/post 双页试点迁移——把 Phase 63/64 修出的 P0/P1 行为（分页 showTotal、loading 蒙层、提交防重、脏检查）固化为组件默认契约，新列表页样板 ~450 行 → ~120 行量级。

**权威契约**：`.planning/phases/66-page-form/66-UI-SPEC.md`（approved，含 checker 修订与 Probe Coverage 节）。任务书与本 PLAN 冲突时以 UI-SPEC 为准；UI-SPEC 与 65 UI-SPEC §2.3 冲突时以 65 §2.3 锁定 API 为准（RemoteSelect 逐字兑现）。

**验证基线**（64 收口值，planner 指定口径）：
- `npm run type-check` → 0 错误
- `npm run test -- --run` → 3843+ passed / **0 failed**（64 收口实测全绿，64-P1-SUMMARY.md:47；历史 8 failed 为 63 口径已清偿——本 phase 任何 failed 都是新增回归）
- `npm run lint` → 全仓 125 errors 存量；**禁跑全量 lint 判闸**，验证一律限定文件范围或数新增

**硬边界**（locked，违反即任务失败）：
- 本 phase 只交付 3 组件 + system/post 1 个试点页（列表+表单同页闭环）；**存量 76 页迁移归 Phase 67，discoveries 不抢**（ROADMAP P67 C1 点名）
- useTableManager **不重写**、公开 API 不删除（25 存量消费页零破坏）；从「页面级 hook」降格为「PageTable 内部实现细节 + 存量兼容层」；本 phase **不加** no-restricted-imports 禁存量页（清单机制见 T7，只拦「新业务文件」）
- PageTable **不自建第二套删除确认**（行删经 createDeleteAction→ConfirmDelete、批删经 BatchDeleteButton→confirmDelete，65 §6 边界 ①）
- FormModal 是 BaseEditModal **之上新增非替换**：BaseEditModal 对外 API 零变化（存量 2 消费页 system/post、operations/workstations/modals/EditModal 不动其 BaseEditModal 用法之外的破坏性改动）
- RemoteSelect **不做跨实例/持久缓存**（UI-SPEC §4 决策）；不新建独立 useDebouncedSearch hook（locked）
- 零新增色值/字号/文案模板：全部沿用 63/64/65 契约与 UI-SPEC Copywriting 表，禁止裸 hex
- 不引入 Skeleton（UI-SPEC §6 裁决：Table 蒙层是唯一 loading 口径）
- 全程不 commit

**行号锚点纪律**：本 PLAN 全部 `文件:行号` 锚点为 2026-10-25 快照（HEAD），漂移需重定位——以语义锚（规则名/函数名/文案字符串）重新 grep，不得按失效行号盲改。

## 事实核查注记（planner 实测，executor 须知）

1. **`RemoteSelect.tsx` 不存在**（`src/components/ui/` 仅 AppInput/AppSelect/index.ts + 各自测试）——本 phase 全新建，65 只锁了契约（65-UI-SPEC.md:149-153：`RemoteSelectProps extends Omit<AppSelectProps, "options"|"onSearch"|"loading">` + `fetcher(keyword)=>Promise<Option[]>` + `debounceMs?=300` + `initialOptions?`）。
2. **`createDeleteAction` 现网签名是位置参数**：`ActionButtons.tsx:114` 为 `createDeleteAction<T>(id, handleDelete, options?)`——UI-SPEC §1 示例里的 `createDeleteAction({ id: r.id, label: r.postName })` 是**语义示意而非签名契约**。实现沿用现网签名，不改 ActionButtons 对外 API；PageTable JSDoc 示例按现网签名书写。
3. **试点页现状 `src/pages/system/post/index.tsx`（480 行）**：useTableManager + usePagination **双 hook 并存**（:33、:54、:90-107 externalPagination 接线）；裸 `Popconfirm` 行删（:251-265）；弹窗内裸 `Input`/`Select`（:445、:466，post 页在 65 豁免清单内）；手写 `submitting` state 防重（:51、:424-431）；统计卡片（:277-310）与「已选择」Alert（:381-400）是业务内容**不可吸收**。post 页有 3 个既有测试（`__tests__/index.test.tsx`、`page.test.tsx`、`render.test.tsx`）作回归锚。
4. **65 豁免清单含 post 页**：`eslint.config.js:313` 列有 `src/pages/system/post/index.tsx`——试点迁移把该页裸 Input/Select/Popconfirm 全部清掉后，**必须复跑 `scripts/gen-p65-exemptions.mjs` 把 post 页从清单收缩**（豁免清单是活文档），清单文件数变化在 T8 收口复扫中说明。
5. **useTableManager 依赖 `useLocation`**（useTableManager.ts:133）——PageTable 内部实例化后，组件单测须包 `MemoryRouter`；PageTable 也因此只能在 Router 上下文使用（JSDoc 注明）。
6. **分页 showTotal 现网唯一正确口径**在 `usePagination.ts:122`：`total===0 ? "" : 共 N 条 / M 页`——PageTable **不经过 usePagination**（UI-SPEC Component Inventory 决策），在内部按 63 §1 同口径实现，杜绝第二处实现漂移。
7. **useTableManager catch 兜底**（useTableManager.ts:260-267）目前只 toast+清空、无错误态 UI——PageTable 的错误态（ErrorAlertWithRetry + 重试）需在其上**升格**：loadData 失败时置错误标记并渲染错误态，重试=重新 loadData。实现走「PageTable 包装 request → 失败记 error state」即可，不改 useTableManager 内部（保持存量页行为不变）。
8. **`handleApiError` 签名**：`errorHandler.ts:136` `(error, context, showMessage?)`；`isFormValidationError` 同文件已有（post 页 :34/:142 在用）——FormModal 失败分流直接复用。

---

## T1 — Tracer 切片：PageTable 最小骨架（request+三态+分页）+ 单测

> **先行任务**：验证「组件内部实例化 useTableManager + 三态契约」全链路可实施后才展开 T2-T6。若 T1 阻塞，后续任务不得开始。

**exact files**：
- 新建 `xingran-react-frontend/src/components/shared/PageTable.tsx`（最小可运行骨架：`request`/`columns`/`rowKey` 三必填 + 分页（内部 useTableManager 实例化，showTotal 按 63 §1 口径：`total>0 → 共 {total} 条 / {pages} 页`，total=0 不渲染）+ 三态（loading=Table 蒙层；空态=antd 默认 `暂无数据`；错误态=ErrorAlertWithRetry，重试=重新 loadData）+ rest 全透传 antd Table；文件头 10 行内用法示例 + 逐 prop JSDoc——§7 自描述要求从此文件开始执行）
- 新建 `xingran-react-frontend/src/components/shared/__tests__/PageTable.test.tsx`（MemoryRouter 包裹；三态/分页口径/request 失败重试夹具）

**read_first**：
- `.planning/phases/66-page-form/66-UI-SPEC.md` §1（API 面 + 决策 1/2）
- `xingran-react-frontend/src/hooks/useTableManager.ts` 全文（loadFunction 签名 :122-125、loadData :242-271、handleTableChange :335-361）
- `xingran-react-frontend/src/hooks/usePagination.ts:122`（showTotal 口径参照，不 import）
- `xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx`（props：error/onRetry）
- `src/components/shared/__tests__/` 任一既有测试（写法惯例）

**acceptance_criteria**：
1. 传 `request`+`columns`+`rowKey` 即渲染出带分页的表格；mock total=128 → 分页区出现 `共 128 条 / 13 页`；total=0 → 无该文案且显示 antd 空态
2. request reject → `handleApiError` 被调（mock）+ ErrorAlertWithRetry 可见；点击重试重新发起 request
3. request pending → Table 蒙层 loading；无 Skeleton/整页 Spin
4. `scroll`/`rowSelection` 等 rest props 透传到 Table（fireEvent 或 DOM 断言）
5. 组件单测在 MemoryRouter 内通过；type-check 0 错误

**fails_when**：组件重写数据层而非内部实例化 useTableManager；showTotal 口径偏离 63 §1 逐字文案；错误态静默（无 handleApiError 或无重试入口）；rest 透传吞 prop/改名

**verification**：
```bash
cd xingran-react-frontend && npm run type-check
npx vitest run src/components/shared/__tests__/PageTable.test.tsx
```
预估：3h

## T2 — PageTable 完整契约：插槽 + 行操作/批删 + 排序/extraFilters

**exact files**：
- 修改 `xingran-react-frontend/src/components/shared/PageTable.tsx`（补齐 UI-SPEC §1 全 API：`searchFields`/`searchChildren` 挂载点、`toolbar` 右区内建刷新（icon 按钮 aria-label+title="刷新"，64 FIX-05）、`rowActions?: (record) => ActionButton[]`（经 ActionButtons 渲染；含 createDeleteAction 的删除自动走 ConfirmDelete）、`batchDelete?: { onDelete, target }`（渲染 BatchDeleteButton，65 confirmDelete 模板，不自建确认）、`selectedRowKeys` 内建管理 + rowSelection 接线、`pageSize?`/`sorterMetas?`/`defaultSort?` 透传 useTableManager options、`extraFilters?`（变化 → applyFilters(extra) 回第 1 页）、`emptyAction?` 转发 EmptyStateWithAction）
- 修改 `xingran-react-frontend/src/components/shared/index.ts`（追加导出 PageTable + 类型）
- 修改 `xingran-react-frontend/src/components/shared/__tests__/PageTable.test.tsx`（新增插槽/批删/排序/extraFilters 夹具）

**read_first**：
- UI-SPEC §1 决策 1/2 + 示例（目标形态 ~120 行）+ §5 试点理由
- `xingran-react-frontend/src/components/shared/ActionButtons.tsx`（ActionButton 接口 :15-24、createDeleteAction 签名 :114-148——位置参数，见事实注记 2）
- `xingran-react-frontend/src/components/shared/BatchDeleteButton.tsx`（props：selectedCount/onConfirm，selectedCount=0 返回 null :43-45）
- `xingran-react-frontend/src/hooks/useTableManager.ts:53-77`（applyFilters/handleReset/resetSelection 语义）

**acceptance_criteria**：
1. `toolbar` 渲染在工具栏左区；内建刷新按钮有 aria-label+title 且触发 loadData
2. `rowActions` 返回的 actions 经 ActionButtons 渲染（<3 个平铺）；删除项走 ConfirmDelete（vi.spyOn(Modal,"confirm") 断言文案含 target）
3. 传 `batchDelete` 且选中 >0 → BatchDeleteButton 可见，确认走 confirmDelete 模板（`确认批量删除`/`将删除选中的 N 项…`）；onDelete 成功后自动 resetSelection + 重新 loadData
4. `sorterMetas`+`defaultSort` 生效（列 sorter/请求参数含 orderByColumn/isAsc）；表头点击触发带排序的请求
5. `extraFilters` 引用变化 → 回第 1 页重载并合并进 filters（对齐 applyFilters 语义）；筛选在翻页间保持（filtersRef 既有行为）
6. `emptyAction` 传入时空态渲染 EmptyStateWithAction
7. type-check 0 / 全部单测通过

**fails_when**：出现第二套删除确认实现；批删确认后不清选中/不刷新；extraFilters 每次渲染都触发重载（须以引用/深比较控制）；内建刷新缺 aria-label

**verification**：
```bash
cd xingran-react-frontend && npx vitest run src/components/shared/__tests__/PageTable.test.tsx && npm run type-check
```
预估：4h

## T3 — SearchForm：schema 驱动 + children 逃生口 + custom render

**exact files**：
- 新建 `xingran-react-frontend/src/components/shared/SearchForm.tsx`（`SearchField[]` schema 渲染：`input→AppInput`（onPressEnter 触发搜索）、`select→AppSelect`、`remoteSelect→RemoteSelect`（T5 后接通，先用类型占位+单测跳过或紧随 T5）、`custom→render(value, onChange)` 受控接入 filters；渲染端点**禁止裸 Input/Select**（契约禁止，且 shared 目录在 lint 豁免层内更须自律）；`searchChildren?: ReactNode` 逃生口（自行渲染 Form.Item，配合 PageTable 暴露的 searchForm）；按钮 `搜索`/`重置`/`刷新`（63 惯例文案，刷新 icon 按钮 aria-label+title）；搜索→handleSearch（filterEmpty+回第 1 页）、重置→handleReset；布局 Card `layout="inline"` `marginBottom:16`）
- 修改 `xingran-react-frontend/src/components/shared/PageTable.tsx`（SearchForm 与 PageTable 接线：`searchFields` 与 `searchChildren` 二选一，同时传 schema 优先并 console.warn；searchForm 实例经 render-props 或受控 ref 暴露给 searchChildren）
- 新建 `xingran-react-frontend/src/components/shared/__tests__/SearchForm.test.tsx`

**read_first**：
- UI-SPEC §2 全节（含 checker 修订 ①：custom 签名 `(value, onChange) => ReactNode`）
- `xingran-react-frontend/src/hooks/useTableManager.ts:275-301`（applyFilters/handleReset）
- `xingran-react-frontend/src/components/ui/AppInput.tsx`、`AppSelect.tsx`（rest 透传纪律参照）

**acceptance_criteria**：
1. 3 种已交付 SearchField 类型各渲染正确端点（input→AppInput DOM；select→AppSelect options；custom→render 返回节点且 onChange 写入 filters 状态）；remoteSelect→仅断言占位渲染不抛错（端点接线断言归 T5，见 :172 接通）
2. 输入后 Enter 触发搜索；点击搜索 → 请求参数含 schema 字段值且空值被过滤（filterEmpty）；重置 → 清 filters 持久化+清排序+回第 1 页
3. 按钮文案逐字 `搜索`/`重置`；刷新按钮 aria-label+title 齐备
4. searchChildren 渲染且其 Form.Item 值进入请求参数（经暴露的 searchForm 接线）
5. searchFields 与 searchChildren 同时传 → schema 生效且出现一次性警告
6. type-check 0 / 单测通过

**fails_when**：SearchForm 内部出现裸 `<Input>`/`<Select>`；custom.render 收不到 onChange 或写入后不触发搜索链路；重置不清排序（对齐 useTableManager.handleReset :292-301 语义）

**verification**：
```bash
cd xingran-react-frontend && npx vitest run src/components/shared/__tests__/SearchForm.test.tsx src/components/shared/__tests__/PageTable.test.tsx
```
预估：3h

## T4 — FormModal：BaseEditModal 之上契约化升级 + useFormModalForm + 单测

**exact files**：
- 新建 `xingran-react-frontend/src/components/shared/FormModal.tsx`（UI-SPEC §3 逐字：props `{ open, title, onFinish(values)=>Promise<void>, onCancel, initialValues?, width?, okText?="保存", dirtyCheck?=true, children: ReactNode | ((form: FormInstance) => ReactNode) }`；内部渲染 BaseEditModal（底座职责不变）；导出 `useFormModalForm(): FormInstance` 供二选一路线 B；行为：① open=true 时 `resetFields()` + `setFieldsValue(initialValues)`（destroyOnHidden 语义补齐）② onOk = 组件内包装的 async 提交：提交中 `cancelButtonDisabled` + antd confirmLoading 防重（63 STOP-04 方案 1 组件化）③ `onCancel` 触发时若 `dirtyCheck && form.isFieldsTouched() && 非提交中` → `Modal.confirm`（标题 `更改尚未保存`/内容 `当前修改尚未保存，关闭后将丢失，是否确认关闭？`/okText `放弃更改`/cancelText `继续编辑`/ok 非 danger）确认后才调 onCancel，未触碰直接关；遮罩/ESC 不进脏检查路径（BaseEditModal 默认关死）④ onFinish 抛错 → `isFormValidationError` 前置分流（校验错静默）→ 其余 `handleApiError(error, "保存")`，弹窗保持打开；文件头用法示例 + 逐 prop JSDoc）
- 修改 `xingran-react-frontend/src/components/shared/index.ts`（追加导出 FormModal / useFormModalForm + 类型）
- 新建 `xingran-react-frontend/src/components/shared/__tests__/FormModal.test.tsx`

**read_first**：
- UI-SPEC §3 全节 + Copywriting 表（脏检查文案逐字）
- `xingran-react-frontend/src/components/modal/BaseEditModal.tsx` 全文（底座默认值 :41-48、destroyOnHidden :63、cancelButtonDisabled :57）
- `xingran-react-frontend/src/utils/errorHandler.ts`（handleApiError :136、isFormValidationError）
- post 页现有弹窗接线 `src/pages/system/post/index.tsx:419-475`（T6 迁移底稿）

**acceptance_criteria**：
1. onOk 快速双击/重复点击 → onFinish 只执行一次（提交中禁用+loading）；提交中点取消无效
2. 触碰字段后点取消 → Modal.confirm 弹出且文案四段逐字匹配；确认「放弃更改」→ onCancel 被调；「继续编辑」→ 弹窗保持；未触碰直接 onCancel
3. `dirtyCheck={false}` → 取消无确认直接关
4. open 从 false→true：上次输入残留被 resetFields 清除且 initialValues 回填成功
5. onFinish reject（非校验错）→ handleApiError 被调 + 弹窗保持打开可重试；校验错 → 静默（行内提示）
6. children 函数形态收到 form 实例；`useFormModalForm()` 路线下 children 传 ReactNode 也能接线（两条路线各 1 夹具）
7. FormModal 文件内 Modal.confirm 是**唯一新增合法调用点**（已在 65 封装层豁免目录内）；grep 组件文件无第二套确认实现
8. type-check 0 / 单测通过

**fails_when**：脏检查弹窗 ok 按钮带 danger（丢弃更改是普通确认，UI-SPEC 明确不套 ConfirmDelete 模板）；提交失败关闭弹窗；destroyOnHidden 语义缺失导致回显残留；okText 默认不是 `保存`；改动了 BaseEditModal 对外 API

**verification**：
```bash
cd xingran-react-frontend && npx vitest run src/components/shared/__tests__/FormModal.test.tsx && npm run type-check
grep -rn "Modal.confirm(" src/components/shared/FormModal.tsx  # 唯一调用点 = 脏检查
```
预估：4h

## T5 — RemoteSelect 实现（逐字兑现 65 §2.3）+ 单测

**exact files**：
- 新建 `xingran-react-frontend/src/components/ui/RemoteSelect.tsx`（65-UI-SPEC.md:149-153 契约逐字：`RemoteSelectProps extends Omit<AppSelectProps, "options"|"onSearch"|"loading">` + `fetcher(keyword)=>Promise<Option[]>` + `debounceMs?=300` + `initialOptions?`；行为：输入防抖调 fetcher；**竞态守卫**递增 requestId 仅采纳最后一次；请求中 Select loading；失败 catch → `handleApiError(error, "搜索选项")` + 保留上次成功选项；keyword 清空（allowClear/删空）→ 回退 initialOptions 且**不发请求**；首屏渲染 initialOptions 无则空态）
- 修改 `xingran-react-frontend/src/components/ui/index.ts`（追加导出 RemoteSelect + 类型）
- 新建 `xingran-react-frontend/src/components/ui/__tests__/RemoteSelect.test.tsx`
- 修改 `xingran-react-frontend/src/components/shared/SearchForm.tsx`（T3 的 remoteSelect 占位接通真实组件；initialOptions 由页面经 SearchField 透传）

**read_first**：
- 65-UI-SPEC.md:140-157（§2.3 锁定契约原文，API 不得另起）
- 66-UI-SPEC §4（竞态守卫/失败保留/无缓存决策逐条）
- `xingran-react-frontend/src/components/ui/AppSelect.tsx`（AppSelectProps 来源）

**acceptance_criteria**：
1. 输入经 300ms 防抖后 fetcher 恰好收到最终 keyword（fake timers 断言中间输入不触发）
2. 竞态：慢的前一次响应返回时不得覆盖新结果（requestId 比对，单测两个延迟不同的 fetcher 夹具）
3. fetcher reject → handleApiError 被调（mock）+ 下拉保留上次成功选项、不清空、无 console
4. 无输入时渲染 initialOptions；keyword 清空 → 回到 initialOptions 且 fetcher 调用数不增加
5. `debounceMs: 0`/自定义值生效；显式 `allowClear/showSearch/placeholder` 等 AppSelectProps 透传生效
6. type-check 0 / 单测通过

**fails_when**：API 偏离 65 §2.3（如自行加 options prop、暴露 onSearch）；实现跨实例缓存（每次防抖必须真实请求，UI-SPEC §4 决策）；失败静默 console 或清空选项；清空 keyword 仍发请求

**verification**：
```bash
cd xingran-react-frontend && npx vitest run src/components/ui/__tests__/RemoteSelect.test.tsx && npm run type-check
```
预估：2.5h

## T6 — 试点迁移：system/post 双页接入 PageTable + FormModal（删旧双轨）

**exact files**：
- 修改 `xingran-react-frontend/src/pages/system/post/index.tsx`（目标形态对齐 UI-SPEC §1 示例：① 列表区改 `<PageTable request={...} columns={...} searchFields={[postCode/postName/status]} toolbar={新增按钮} rowActions={(r)=>[editAction, createDeleteAction(r.id, handleDelete, {label: r.postName})]} batchDelete={{ onDelete, target: "岗位" }} rowKey="id" sorterMetas={...} />`——**页面不再 import useTableManager/usePagination**（双轨删除，hook 由组件内部持有）；统计卡片与「已选择」Alert 属业务内容保留（放 toolbar 区或 PageTable 上方，行为不变）；② 新增/编辑弹窗改 `<FormModal>`（onFinish 内 update/create + handleSuccess + 列表刷新经 PageTable 暴露的刷新或回调；submitting state 手写防重删除）；③ 裸 Popconfirm/Input/Select 全部消失（搜索区走 schema、弹窗 Form.Item 内控件换 AppInput/AppSelect、行删走 createDeleteAction→ConfirmDelete）；④ okText 用默认 `保存`）
- 修改 `xingran-react-frontend/src/pages/system/post/__tests__/`（3 个既有测试适配新结构；断言保留：分页 showTotal、删除确认、表单提交路径——回归锚不得放松）
- 复跑 `xingran-react-frontend/scripts/gen-p65-exemptions.mjs` + 修改 `xingran-react-frontend/eslint.config.js` 存量豁免段（post 页裸组件清零后从清单收缩——见事实注记 4）

**read_first**：
- UI-SPEC §1 示例 + §5（试点=样板非迁移对象，旧实现不留双轨）
- post 页全文（迁移底稿，保留统计卡片等业务块）
- post 页 3 个既有测试全文（断言迁移映射表）
- `ActionButtons.tsx:114`（createDeleteAction 位置参数签名）

**acceptance_criteria**：
1. post 页 grep `useTableManager|usePagination|Popconfirm|<Input|<Select ` = 0 命中（双轨与裸组件清零；`<InputNumber`/`Input.TextArea` 不在拦截面，允许保留）
2. 用户视角行为不回归：搜索/重置/刷新、分页 `共 N 条 / M 页`、排序、行删确认、批删确认、新增/编辑/回填/防重/统计卡片全部保持
3. 页面行数显著下降（迁前 480 行 → 目标 ~120 行业务量级；统计卡片等业务块不计入吸收面，量化值写入 T8 SUMMARY）
4. 既有 3 个测试适配后全通过；post 页从 65 豁免清单移除后定向 lint 该文件 0 新增 error
5. type-check 0 / 全量 vitest 相对基线（3843+/0 failed）无任何新增失败

**fails_when**：统计卡片/已选择 Alert 等业务行为丢失；弹窗回填残留或防重失效；试点保留旧代码路径（双轨）；豁免清单未收缩导致 post 页裸组件继续被豁免

**verification**：
```bash
cd xingran-react-frontend
grep -nE "useTableManager|usePagination|Popconfirm" src/pages/system/post/index.tsx   # 期望 0
node scripts/gen-p65-exemptions.mjs && git diff --stat eslint.config.js              # post 页移出清单
npx eslint src/pages/system/post/index.tsx                                            # 0 error
npx vitest run src/pages/system/post/ && npm run type-check
```
预估：4h

## T7 — lint 增量门：新代码强制 PageTable（存量豁免清单沿用 65 机制）

**exact files**：
- 修改 `xingran-react-frontend/eslint.config.js`（新增独立 config 块「Phase 66 契约强制」，放 65 契约块之后、存量豁免 override 之前；**只拦 import 侧门，不加 JSX selector**：`no-restricted-imports` patterns = **合并 65 契约块既有 antd 子包 patterns（eslint.config.js:142-153）+ 新增 `@/hooks/useTableManager` 与 `@/hooks/usePagination` 两条**，不得另起数组——flat config 后块同名规则整体覆盖前块，另起数组会让 65 的 antd 子包拦截在 src/** 静默失效；message 指向 PageTable（「新列表页请使用 @/components/shared 的 PageTable；存量页见豁免清单，P67 迁移后移除」）——不加会让 25+46 个存量页全部报错的 no-restricted-syntax 规则）
- 新增封装层 import 豁免小段：`files: ["src/components/shared/**", "src/components/ui/**"]` 重声明 `no-restricted-imports` = 仅保留 65 既有 antd 子包 patterns、去掉 hook patterns（封装层 override `eslint.config.js:163-175` 现只豁免 no-restricted-syntax，PageTable 自身 import useTableManager 会报错，必须显式豁免）
- 新增存量豁免 override：`files: [<存量命中清单>]` + 注释 `// P67 收敛后移除`，豁免内容 = **重声明完整 `no-restricted-imports` 数组（保留 antd 子包 patterns、仅省去两条 hook patterns）**，禁止直接置 `"off"`（会连带关闭 65 的 antd 子包拦截）；其余规则不动；清单**脚本生成**
- 新建 `xingran-react-frontend/scripts/gen-p66-exemptions.mjs`（沿用 gen-p65-exemptions.mjs 机制：跑 eslint 数出两个 hook 的存量 import 文件全集，替换锚点写入 files 数组；T7 生成、T8 收口复跑。实现前提：① 扫描用临时 config 必须基于完整 config 复制、只替换 p66 锚点段，保留 P65 段原样（p65 段只关 no-restricted-syntax，不影响 import 扫描）；② 命中识别按 no-restricted-imports 的 message 标记（类比 gen-p65-exemptions.mjs:31-36 的 P65_MESSAGE_MARKS 机制））
- 修改 `xingran-react-frontend/src/components/shared/PageTable.tsx`（JSDoc 注明「新列表页唯一合法列表数据层入口」）

**read_first**：
- `xingran-react-frontend/scripts/gen-p65-exemptions.mjs`（脚本机制模板）
- `xingran-react-frontend/eslint.config.js` 全文（flat config 块顺序纪律：prettier 必须 :350 最后；65 三个先例块 :115/:163/:180）
- 65-PLAN.md T2 修订 ③（flat config 同名键覆盖教训：独立 config 块）

**acceptance_criteria**：
1. 新建一个业务 tsx `import { useTableManager } from "@/hooks/useTableManager"` → lint error 且 message 指向 PageTable
2. 存量消费页（25+46 页全集）经豁免清单后 0 新增 error；全仓定向 lint 新增 errors = 0（125 存量不动）
3. `src/components/shared/**` 与 `src/components/ui/**` 内部（PageTable 自身 import useTableManager）经封装层 import 豁免小段 0 error，且 antd 子包拦截在该层仍生效（探针 1 error）
4. 测试文件天然豁免（既有测试 override :343-348 关整条 no-restricted-syntax，但 no-restricted-imports 不在该 override 内——需确认并按需扩展测试 override 或依赖封装层/存量清单覆盖，实测后择一写死）
5. 复跑 gen-p66-exemptions.mjs 幂等（清单与实际命中集一致）；新文件命名避免与同目录既有文件仅大小写不同（65 教训：ConfirmDelete.tsx/confirmDelete.ts）

**fails_when**：规则误伤 25+46 存量页任一文件（豁免清单遗漏）；豁免清单把 PageTable/FormModal/SearchForm 自身也列进去；新块插到 prettier 之后；拦截 message 未指向 PageTable

**verification**：
```bash
cd xingran-react-frontend
node scripts/gen-p66-exemptions.mjs
npx eslint src 2>&1 | tail -3        # 总 error 数 = 125 存量 + 0 新增
npx eslint src/components/__lint_probe__.tsx   # 临时 fixture 试拦后删除
```
预估：2.5h

## T8 — 收口：SC 逐条验证 + 基线保护 + 豁免复扫 + SUMMARY

**exact files**：
- 新建 `.planning/phases/66-page-form/66-SUMMARY.md`（不 commit）

**read_first**：
- `.planning/phases/65-contracts/65-PLAN.md` T8（SUMMARY 结构模板）与 65-P1-SUMMARY.md（若已存在）
- 本 PLAN 全部任务的 verification 输出

**acceptance_criteria（SC 逐条，对齐 ROADMAP Phase 66）**：
- **SC1**（分页/loading/空/错四态零配置）：T1/T2 单测输出——total=128 显示 `共 128 条 / 13 页`、total=0 不渲染、错误态 ErrorAlertWithRetry 可重试
- **SC2**（~450 → ~120 行 + SearchForm 并入）：T6 迁移前后行数量化（480 → N 行）+ T3 单测
- **SC3**（FormModal 三默认）：T4 单测——防重/脏检查两分支/回填重置/失败保持打开
- **SC4**（契约自描述）：三组件文件头示例 + 逐 prop JSDoc 齐备；「试点页 diff 即样板」对照说明
- **基线保护**：type-check 0 错误；vitest 3843+ passed / **0 failed**（64 收口全绿，任何 failed 均为新增回归）；定向 lint 新增 errors = 0（83 实测存量/125 历史口径，判闸只数新增）
- **基线不回退专项**：63/64 口径复扫——分页文案逐字、BaseEditModal maskClosable/keyboard 默认值未动（git diff 确认）、aria（刷新/操作按钮）
- **65 豁免清单复扫说明**（必填节）：复跑 `gen-p65-exemptions.mjs` 前后清单文件数对比（post 页移出）；`gen-p66-exemptions.mjs` 新清单文件数；P67 待办承接说明
- SUMMARY 含：任务完成表（T1-T8 + 不 commit 声明）、验证命令与输出、66→67 衔接（PageTable 是 P67 迁移唯一目标形态；RemoteSelect demo 用法说明——交付不强制接入试点，以 T5 单测 + 用法示例验收）

**fails_when**：任一 SC 无可执行验证输出支撑；基线任一项回退；豁免清单复扫缺数据；SUMMARY 声称与实际输出不符

**verification**：SUMMARY 内嵌全部命令输出
预估：1.5h

---

## 依赖顺序

```
T1（tracer，必须先行）
 └→ T2（PageTable 完整契约）→ T3（SearchForm，依赖 T2 接线）→ T6（试点迁移，依赖 T2/T3/T4）
 ├→ T4（FormModal，线程独立，T1 后即可开始）──────────────┘
 ├→ T5（RemoteSelect，线程独立；T3 的 remoteSelect 字段在其后接通）
 └→ T7（lint 增量门，依赖 T2 稳定 + T6 收缩 post 页后生成清单）→ T8（收口，最后）
```
可并行：T4、T5 与 T2/T3 线程独立；T6 必须在 T2/T3/T4 后（RemoteSelect 不阻塞 T6，post 页无远程字典字段则以单测+demo 验收，UI-SPEC §5）；T7 必须在 T6 后（豁免清单反映 post 页清零后的真实存量）；T8 收尾。

**总工时预估：24.5h**（T1 3h / T2 4h / T3 3h / T4 4h / T5 2.5h / T6 4h / T7 2.5h / T8 1.5h）

## 执行注意（executor 必读）

1. 全量 `npm run lint` 与 `npm run test -- --run` 耗时长——一切判闸用定向命令 + 基线对比（type-check 0 / vitest 3843+ 且 failed=0 / lint 新增 error=0）。
2. 豁免清单是活文档：T6 收缩 65 清单（post 页移出）、T7 生成 66 清单、T8 复跑两个脚本对账——收口时两清单与实际命中集必须一致。
3. 任何任务发现必须触碰 UI-SPEC 未覆盖的组件/行为，停下上报 planner，不得自行扩面（硬边界 locked）。存量 76 页迁移、discoveries、operations/duty/workorder 域一律不碰（归 Phase 67）。
4. PageTable/FormModal/RemoteSelect 均为纯新增文件（除 index.ts 导出与 eslint.config.js 外不动既有组件源码）；BaseEditModal/useTableManager/usePagination 对外 API 零变化。
