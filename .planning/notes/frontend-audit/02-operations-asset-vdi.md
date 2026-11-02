# 前端组件审查报告（分片 02）：operations / asset / vdi

> 审查对象：`xingran-react-frontend/src` 下 `pages/operations`、`pages/asset`、`pages/vdi`、`components/operations`、`components/asset`、`components/reconciliation`。
> 审查方式：**只读**审查，未修改任何源码。
> 路径口径：行号均相对 `xingran-react-frontend/`，例 `src/pages/operations/buildings/index.tsx:610`。
> 结论：**42 项问题**（P0 10 / P1 20 / P2 12），**12 项抽取候选**（必抽 4 / 应统一 6 / 不建议 2），预计消除约 **1,900 行**重复代码、扣除新抽象后**净减约 900 行**（本分片生产代码 25,687 行的 3.5%），并把 6 个已存在但闲置的公共组件重新接入。行数不是主要收益——真正的收益是让"分页总条数"这类缺陷从改 11 个页面收敛为改 1 个 hook。

---

## 1. 审查范围与方法

### 1.1 范围与规模

| 目录 | 生产文件数 | 内容 |
| --- | --- | --- |
| `src/pages/operations/` | 75 | 楼宇 / 楼层 / 工位 / 机房 / 设备 / 专线 / 信息点 / RPA / 智能工作站 |
| `src/pages/asset/` | 7 | 资产列表 + 资产对账（dashboard / exceptions / exception-rules / fix-suggestion） |
| `src/pages/vdi/` | 4 | VDI 服务器配置、虚拟机列表 / 详情 |
| `src/components/operations/` | 5 | WorkstationDeviceTable、AIScriptEditor 等 |
| `src/components/asset/` | 2 | ExceptionRuleForm 等 |
| `src/components/reconciliation/` | 10 | HealthCard / HealthBadge / ReconciliationDrawer / Timeline / 数据 hooks |

- `ts/tsx` 共 193 个（含测试）；**生产文件 103 个、25,687 行**。
- 技术栈：React 19.2 / TS 5.9 / Vite 7.2 / Ant Design 6.1 / Zustand 5 / TanStack Query 5。

### 1.2 方法

1. **定量扫描**：Node 脚本对 103 个生产文件做标签级 / 属性级 / 调用级正则计数（`<Input`、`allowClear`、`showTotal`、`Modal.confirm`、`message.`、hex 色值、`onSearch`、`onPressEnter` 等），得到覆盖率基线。
2. **定点精读**：对 40+ 个关键文件按行取证（重点：20 个含 `<Table>` 的页面、23 个 Modal/Drawer、6 处手写分页、4 份重复的工具栏 JSX、10 个 reconciliation 组件），确保每条问题都能落到 `文件:行号`。
3. **范式归类**：按"数据获取 → 状态管理 → 渲染骨架 → 反馈"四层，把页面归入 3 套实现范式，识别差异是"能力差异"还是"习惯差异"——前者才需要抽组件，后者需要定标准。

### 1.3 公共设施基线（本分片采用情况）

| 已有公共设施 | 位置 | 本分片采用 |
| --- | --- | --- |
| `usePagination` / `useTableManager` | `src/hooks/usePagination.ts`（140 行）/ `useTableManager.ts`（410 行） | 均 11 个文件 |
| `useTableQuery` / `useColumnConfig` | `src/hooks/` | 1 / 1 |
| `BaseEditModal` | `src/components/modal/BaseEditModal.tsx`（65 行） | **1 文件 / 4 处** |
| `ActionButtons` / `DepartmentTreeSelect` | `components/shared/` | 9 / 4 |
| `BatchDeleteButton` / `BatchExportModal` / `ModernTag` / `GlobalSearch` | `components/shared/` | **0 / 0 / 0 / 0** |
| `EmptyStateWithAction` / `ErrorAlertWithRetry` | `components/shared/` | **0 / 0** |
| `ColumnConfigModal` / `ExcelImportLazy` / `ExcelExport` | `components/shared/` | 1 / 8 / 7 |
| `createStatusColumn` / `createDateTimeColumn` / `createActionColumn` | `src/utils/tableHelpers.tsx` | 4 / 9 / 0 |
| `handleApiError` / `handleSuccess` / `withErrorHandling` | `src/utils/errorHandler.ts`（337 行） | 16 / 10 / **0** 文件 |
| `getAppMessage` / `setAppMessageInstance` | `src/utils/antdMessage.ts`（53 行） | 几乎未消费 |

> 关键判断：**抽象层已经付过成本，但没有被消费**。所以"是否要抽公共组件"的答案不是"再新建一层"，而是"为什么已有抽象没人用、如何让它可用"。第四章按这个思路评估。

---

## 2. 问题总览

### 2.1 数量统计与严重度分布

| 级别 | 数量 | 占比 | 定义 | 代表问题 |
| --- | --- | --- | --- | --- |
| **P0** | 10 | 24% | 用户可见的功能缺失、数据/操作风险 | 分页不显示总条数、对账 Drawer 两 Tab 永久空态、编辑功能未实现 |
| **P1** | 20 | 48% | 一致性 / 可维护性缺陷，持续放大开发成本 | 列表页三套范式并存、工具栏 JSX 重复 6 份、状态枚举各写一遍 |
| **P2** | 12 | 28% | 细节与体验瑕疵 | `allowClear` 覆盖率仅 22%、无 `sizeOptions`、键盘不可达 |
| **合计** | **42** | 100% | | |

### 2.2 本分片量化指标（实测）

| 指标 | 现状 | 备注 |
| --- | --- | --- |
| `Input` 标签 | 69 个，仅 15 带 `allowClear`（缺 54） | 覆盖率 22% |
| `Select` 标签 | 94 个，仅 38 带 `allowClear`（缺 56）、19 带 `showSearch` | 覆盖率 40% |
| `InputNumber` / `Input.TextArea` | 19 / 5，**0** 带 `allowClear` | 全缺 |
| `<Table>` | 23 处 / 20 文件，10 处配 `scroll` | `virtual` 与 `scroll` 混用 |
| 分页 `showTotal` | 7 处 / 6 文件，仅 5 处显示"共 N 条" | 其余走 hook 默认"共 N 页" |
| Modal / Drawer | 23 个；`destroyOnHidden` 仅 6 处、已废弃 `destroyOnClose` 2 处、`confirmLoading` 仅 8 处 | 关闭残留 + 可连点 |
| `Modal.confirm` / `Popconfirm` | 16 处 / 10 文件、7 处 / 6 文件 | 两套删除确认 |
| `message.*` | **106 次 / 22 文件**，`App.useApp()` 仅 1 处 | 主题 context 告警 |
| `onSearch={() => {}}` | **30+ 处** | 下拉"看似能搜、实际不搜" |
| `onPressEnter` | **0 处** | 搜索框不能回车提交 |
| hex 颜色字面量 | **351 处 / 45 文件** | 与 `var(--theme-*)` 混用 |
| `console.error` / `catch` 块 | 40 / 135 | 三套错误处理 |
| `useState` / `useEffect` | 251（46 文件）/ 104（38 文件） | 单页最多 22 个 `useState` |

### 2.3 Top3 最严重问题

**Top1｜公共分页 hook 把"总条数"实现成了"总页数"，一处根因污染 11 个列表页。**

`src/hooks/usePagination.ts:122` 的 `showTotal` 是 `total => 共 ${Math.max(1, Math.ceil(total / pageSize))} 页`；11 个接入该 hook 的列表页（buildings / dedicated-lines / floors / info-points / room-devices / server-rooms / workstations / rpa×3）全部透传 `paginationProps`，于是用户看到的永远是"共 N 页"。反证：本分片 6 处手写分页（`assets/index.tsx:724-731`、`exceptions/index.tsx:506-513`、`exception-rules/index.tsx:440`、`fix-suggestion/index.tsx:534`、`VDIServerConfig/index.tsx:256`、`VirtualMachineList/index.tsx:957`）全部正确显示"共 N 条"——正确写法存在，只是没有沉淀回 hook。这正是用户投诉"分页组件不显示总数量"的直接根因。

**Top2｜名义存在但实际不可用的"小功能"，是用户信任崩塌点。**

`src/components/reconciliation/ReconciliationDrawer.tsx:67-70`：`const [timeline] = useState([])`、`const [rules] = useState([])`，只解构值、从不调用 setter（注释写"Plan 02 接入"），导致"历史变更""例外规则"两个 Tab 对用户永久显示空态。同类：`assets/index.tsx:203` 编辑按钮直接 `message.info("编辑功能待实现")`；`assets/index.tsx:687-691`"下载模板"与"导入"同指向 `setImportVisible(true)`；`fix-suggestion/index.tsx:501-503` 责任部门筛选用 `Input` 让用户手填 UUID；`ExceptionRuleForm.tsx:215` 的 `scopeId` 要求手填 UUID。

**Top3｜列表页三套并行范式 + 已有公共组件几乎无人使用。**

（1）`useTableManager` + `usePagination` 派：11 页；（2）TanStack Query + 手写分页派：`VDIServerConfig`（2 个 `useState`）、`VirtualMachineList`（**22 个 `useState`、1286 行**）；（3）纯 `useState` 手写派：`ComponentListTab`、`LocationAliasDrawer`、`WorkstationDeviceTable`（13 个 `useState`）、`MatchTestPanel`。而 `BaseEditModal` / `EmptyStateWithAction` / `ErrorAlertWithRetry` / `BatchDeleteButton` / `ModernTag` 在本分片的使用次数分别是 1 / 0 / 0 / 0 / 0。

---

## 3. 十维度问题清单

格式：`文件:行号` —— 现象 —— 级别 —— 修法。共 42 条。

### 3.1 列表页骨架

1. `src/hooks/usePagination.ts:122` —— `showTotal` 实现为 `总条数 / pageSize` 的向上取整，输出"共 N 页"，11 个列表页（`buildings/index.tsx:702-710`、`dedicated-lines/index.tsx:712-720`、`floors`、`info-points/index.tsx:1003-1013`、`room-devices`、`server-rooms/index.tsx:576-584`、`rpa/executions`、`rpa/tasks`、`rpa/workers`、`workstations/index.tsx:626-639`）全部继承此缺陷 —— **P0** —— 修法：`showTotal: (t, range) => 共 ${t} 条（第 ${range[0]}-${range[1]} 条）`，并在 `paginationProps` 里补 `showSizeChanger` 与 `sizeOptions: ["10","20","50","100"]`；改一处即可修复全部 11 页。
2. `src/pages/asset/reconciliation/exceptions/index.tsx:402-427` —— `onTableChange` 中同时调用 `handleTableChange`（useTableManager 内部已维护 current/pageSize）和 `setCurrent/setPageSize`，两套状态并存 —— **P0** —— 修法：分页状态只保留一个来源。若保留 `useTableManager`，则删除页面级 `current/pageSize` 与 `onTableChange` 里的手动 set，`pagination` 直接用 `paginationProps`。
3. `src/pages/vdi/VDIServerConfig/index.tsx:33-34`（2 个 `useState` 手写分页）、`src/pages/vdi/VirtualMachineList/index.tsx:54-70`（22 个 `useState`、1286 行）、`src/components/operations/WorkstationDeviceTable/index.tsx:52-70`（13 个 `useState`）、`src/pages/operations/assets/components/ComponentListTab.tsx`、`src/pages/operations/workstations/LocationAliasDrawer.tsx` —— 列表数据获取与分页存在三套并行范式（`useTableManager` 派 / TanStack Query 手写派 / 纯 useState 派）—— **P1** —— 修法：统一到 `useListPage`（见第四章 C1），TanStack Query 作为其内部数据层，页面不再直接持有 `current/pageSize/total/loading`。
4. 全片 0 处传入 `sizeOptions`（如 `src/pages/operations/buildings/index.tsx:702-710` 的 `pagination={paginationProps}` 由 hook 内部产出，hook 未设 `sizeOptions`）—— 页大小下拉是 antd 默认的 10/20/50/100 且未与业务对齐，也没有"每页条数"文案 —— **P2** —— 修法：在 `usePagination` 统一 `sizeOptions`，并允许页面按数据量覆盖（如日志类默认 50）。

### 3.2 搜索区

5. `src/pages/asset/reconciliation/fix-suggestion/index.tsx:501-503` —— 责任部门筛选用 `<Input placeholder="部门 ID">`，要求用户手工输入 UUID —— **P0** —— 修法：换成 `DepartmentTreeSelect`（`components/shared/DepartmentTreeSelect.tsx` 已存在，本分片已用于 4 处），值类型改为部门 id。
6. `src/pages/operations/buildings/index.tsx:610`、`dedicated-lines/index.tsx:565`、`info-points/index.tsx:899`、`room-devices/index.tsx:555`、`server-rooms/index.tsx:478`、`assets/index.tsx:632` 等 **30+ 处** —— `onSearch={() => {}}` 死代码：`Select` 开了 `showSearch` 却没有任何过滤/远程搜索实现，输入关键词后列表不变 —— **P1** —— 修法：二选一。本地选项集用 `showSearch + optionFilterProp="label"`；远程数据源用统一 `RemoteSelect`（第四章 C7）；禁止提交空 `onSearch`。同时全片 0 处 `onPressEnter`（`src/pages/operations/buildings/index.tsx:596-627` 的搜索 `Form` 无 `onPressEnter`），搜索框不支持回车提交 —— 修法：`Form` 上加 `onPressEnter` 或 `Input` 上绑定 `onPressEnter={handleSearch}`。
7. `src/pages/operations/info-points/index.tsx:320-321`（debounce 300ms）、`src/pages/vdi/VirtualMachineList/index.tsx:1261-1264`（`setTimeout` 300ms 手写）、`src/pages/operations/workstations/index.tsx:774`（`onSearch={(kw) => loadFloorOptions(selectedDeptId, kw)}`，**无防抖**）—— 远程搜索防抖重复实现 3 份，其中 1 份漏了防抖（每次击键发请求）—— **P1** —— 修法：抽 `useDebouncedSearch` 并封装进 `RemoteSelect`，页面不再持有 timer。
8. `src/pages/vdi/VDIServerConfig/index.tsx:180-261`、`src/pages/vdi/VirtualMachineList/index.tsx:939-963` —— 列表页完全没有搜索/筛选区，只能翻页查找 —— **P1** —— 修法：补 `FilterBar`（第四章 C3），至少提供名称/状态两个筛选项。
9. `src/pages/operations/buildings/index.tsx:597-602`、`server-rooms/index.tsx:482`、`485`、`488`、`dedicated-lines/index.tsx:556-569` 等 —— `Input` 缺 `allowClear`（本分片 69 个 `Input` 中 54 个缺），用户修改筛选条件必须全选删除 —— **P2** —— 修法：在 ESLint 层禁止无 `allowClear` 的 `Input`/`InputNumber`/`TextArea`（自定义规则或 `no-restricted-syntax`），或在 `FilterBar` 内部统一注入。
10. `src/pages/operations/server-rooms/index.tsx:474-479`、`info-points/index.tsx:894-919`、`assets/index.tsx:631-663` 等 —— `Select` 缺 `allowClear`（94 个中 56 个缺），且仅 19 个带 `showSearch`；状态类下拉普遍无法清空回"全部" —— **P2** —— 修法：同上，`Select` 默认带 `allowClear`，仅"必选枚举"显式关闭并注释原因。
11. `src/pages/operations/assets/index.tsx:618-679`、`room-devices/index.tsx` 搜索区 —— `InputNumber` 19 处、`Input.TextArea` 5 处全部无 `allowClear`（数值区间与备注筛选无法快速清空）—— **P2** —— 修法：同上统一注入。

### 3.3 表格

12. `src/pages/operations/buildings/index.tsx:612-613`、`dedicated-lines/index.tsx:625-627`、`server-rooms/index.tsx:489-490`、`info-points/index.tsx:916-918`、`assets/index.tsx:658-659` —— 状态枚举 `<Option value={0}>正常</Option>` 在 5+ 页各写一遍；工位三态（`workstations/columns.tsx:92-101` 注释说明 0 空闲/1 占用/2 维护）无法复用 `createStatusColumn`，于是又手写一份 —— **P1** —— 修法：建 `src/constants/statusDict.ts` 导出 `STATUS_OPTIONS` / `statusTag(status, dict)`，`createStatusColumn` 接受字典参数；三态域单独注册 `workstationStatusDict`。
13. `<Table>` 配置不一致：`assets/index.tsx:711-739`、`workstations/index.tsx:626-639`、`info-points/index.tsx:1003-1013` 有 `virtual` + `scroll={{y:600}}`；`buildings/index.tsx:702-710`、`server-rooms/index.tsx:576-584` 无 `scroll`；`exceptions/index.tsx:499-515` 有 `x:1380` 无 `virtual` —— 同类列表的横向滚动/虚拟滚动/固定表头行为不统一，窄屏下列错位 —— **P1** —— 修法：`useListPage` 返回统一 `tableProps`（`size`、`scroll`、`rowKey`、`virtual` 阈值），页面只传 `columns`。
14. `src/components/reconciliation/ReconciliationDrawer.tsx:22`、`:167`（注释声称"锁定 780px"，实际 `size="large"`）—— 注释与实现不符，阅读者按注释改代码必然踩坑 —— **P2** —— 修法：改注释或改实现为 `width={780}`，并在 CI 里对"注释中的数值"不做强制，仅要求 PR 自查。

### 3.4 表单 Modal

15. `src/pages/operations/assets/index.tsx:203` —— 编辑按钮直接 `message.info("编辑功能待实现")`，资产编辑入口是死路 —— **P0** —— 修法：接入 `SafeModal` + 表单（复用 C6），或暂时隐藏按钮并加 `TODO` 追踪，不允许"点了弹提示"的假入口。
16. `src/components/operations/WorkstationDeviceTable/index.tsx:615-682`（表单仅 `deviceSerial`/`description`/`ipAddress` 三项）与 `:654-662`（Alert 文案"仍可以添加但需要手动填写"）—— 设备名称/型号/类型/MAC/责任人只能靠序列号自动回填且不可手改，资产库不存在时用户无法录入，文案与实现矛盾 —— **P0** —— 修法：表单字段全部可编辑，`deviceSerial` 的模糊搜索作为可选的"自动填充"按钮（填到即回填、填不到仍可手填）。
17. `src/components/asset/reconciliation/ExceptionRuleForm.tsx:215` —— `scopeId` 让用户手填 `00000000-0000-0000-0000-000000000000` 形式的 UUID（部门/用户范围）—— **P0** —— 修法：`scopeType === "dept"` 用 `DepartmentTreeSelect`，`=== "user"` 用远程 `RemoteSelect`（用户分页搜索）；UUID 输入框删除。
18. `src/pages/operations/dedicated-lines/index.tsx:198-214`（`handleSave` 无 `submitting` 状态，`Modal` 也仅传 `open/width/onOk/onCancel`，见 `:725`）、`src/components/operations/WorkstationDeviceTable/index.tsx:615-682`（无 `confirmLoading`）—— 提交期间可反复点击"确定"，产生重复创建 —— **P0** —— 修法：统一 `useMutationAction`（C5）产出 `submitting`，`Modal` 强制 `confirmLoading={submitting}`；`Modal` 的 `onOk` 内部 `if (submitting) return`。全分片 23 个弹窗仅 8 个有 `confirmLoading`。
19. 弹窗宽度 9 档无规范：520（`components/reconciliation/ReconciliationDrawer.tsx:167` 默认）、600（`buildings/index.tsx:715`、`floors/FloorModal.tsx:80`、`server-rooms/index.tsx:589`）、680（`components/asset/reconciliation/ExceptionRuleForm.tsx:142`）、700（`dedicated-lines/index.tsx:725`、`room-devices/index.tsx:688`）、720（`fix-suggestion/FixSuggestionDetailDrawer.tsx:85`）、760（`info-points/index.tsx:1018`）、780（`exception-rules/index.tsx:457`）、800（`VirtualMachineList/index.tsx:967`）、900（`rpa/tasks/EditModal.tsx:263`）—— **P1** —— 修法：定 4 档 `sm=520 / md=680 / lg=800 / xl=960`，`SafeModal` 用 `size` 枚举替代裸 `width`。
20. 校验规则零散：`src/components/operations/WorkstationDeviceTable/index.tsx:673` 的 IP 正则只有 4 段数字、无 0-255 范围校验；`src/components/asset/reconciliation/ExceptionRuleForm.tsx:148` 仅 `max: 128`；全分片 `pattern` 仅 1 处、`max` 仅 1 处 —— 长度/格式约束依赖后端报错 —— **P1** —— 修法：建 `src/utils/rules.ts`（`ipRule`、`cidrRule`、`nameRule(max)`、`uuidRule`、`reasonRule(min)`），`Form.Item` 直接引用；`ipRule` 用合法 IPv4 校验函数。

### 3.5 三态（loading / empty / error）

21. `src/components/reconciliation/ReconciliationDrawer.tsx:67-70` —— `timeline` / `rules` 两个 `useState` 只取值不赋值，Tab2"历史变更"、Tab3"例外规则"永久空态 —— **P0** —— 修法：接 `useReconciliationTimeline` / `useExceptionRuleList`（TanStack Query，参照同目录 `useWorkstationHealth.ts:14-34` 的写法），加载中用 `Skeleton`、失败用 `ErrorAlertWithRetry`；在数据接入前先隐藏这两个 Tab，不要展示永久空态。
22. 三态实现四种风格：`src/components/reconciliation/HealthCard.tsx:40-69`（Skeleton + Result + Empty，**样板**）、`src/pages/asset/reconciliation/exceptions/index.tsx:430-436`（`isError` 只渲染 `Empty description="异常列表加载失败"`，无重试）、`src/pages/vdi/VirtualMachineList/index.tsx:975-978`（整页 `Spin` 遮挡）、`src/pages/asset/reconciliation/dashboard/index.tsx:345-360`（`Alert` 引导）—— **P1** —— 修法：统一三态组件 `PageState`（C10）：`loading` → `Skeleton`（表格用 `Skeleton` 行骨架）、`error` → `ErrorAlertWithRetry`（带 `refetch`）、`empty` → `EmptyStateWithAction`（带主操作）。异常页的错误态必须可重试。
23. `src/components/operations/WorkstationDeviceTable/index.tsx:606-609` —— 物理链路空态用 `<div>暂无物理链路信息</div>` 而非 `<Empty>`，样式与其它空态不一致 —— **P2** —— 修法：换 `Empty` 或 `EmptyStateWithAction`（`size="small"`）。
24. `src/pages/vdi/VirtualMachineList/index.tsx:975-978`、`:1202-1205` —— Modal 内用 `<Spin spinning>` + 手写 `div`"加载 VDI 配置中..."，遮罩整块表单且无骨架 —— **P2** —— 修法：用 `Skeleton` 占位表单区域，保留标题与按钮可点（或明确禁用并说明原因）。

### 3.6 反馈与确认

25. 删除确认两套并存：`Modal.confirm` 16 处（`assets/index.tsx:209`、`buildings/index.tsx:444`、`dedicated-lines/index.tsx:396`、`server-rooms/index.tsx:357`、`columns` 内 `workstations/columns.tsx:132` 等）与 `Popconfirm` 7 处（`WorkstationDeviceTable/index.tsx:447-497`、`VDIServerConfig/index.tsx:215-224` 等）—— 同一操作在不同页用不同交互，`Modal.confirm` 版本还各自复制 `onOk: async () => { await api; message.success; refresh }` 结构 —— **P1** —— 修法：抽 `useDeleteConfirm`（C4）统一"确认文案 + 危险按钮 + loading + 成功刷新 + 失败提示"，`Popconfirm` 仅用于行内轻量删除并提供 `loading`。
26. `src/pages/operations/buildings/index.tsx:152`、`floors/index.tsx:238`、`workstations/index.tsx:309`、`assets/index.tsx:145` 等 **22 个文件 106 次** —— 直接 `import { message } from "antd"` 静态调用，而 `src/utils/antdMessage.ts:1-53`（`getAppMessage`/`setAppMessageInstance`）与 `App.useApp()`（全分片仅 `ExceptionRuleForm.tsx:79` 使用）是正确做法 —— 静态 message 无法消费主题 context，antd 6 下会输出告警且深色模式下样式错乱 —— **P1** —— 修法：全局替换为 `const { message } = App.useApp()`；对非组件模块（如 `lib/opsApi.ts` 的 catch）用 `getAppMessage()` 惰性取实例。
27. 错误处理三套并存：`handleApiError`（16 文件，`buildings/index.tsx:278`、`:294`、`:312` 等）、裸 `message.error`（`VDIServerConfig/index.tsx:79-119` 固定文案"保存失败"，**丢失后端具体错误信息**）、`console.error` + 无提示（`floors/index.tsx:238` 等 40 处）；`withErrorHandling`（`errorHandler.ts`）0 使用；校验错误判断 `(err as { errorFields?: unknown }).errorFields` 在 `ExceptionRuleForm.tsx:118`、`dedicated-lines/index.tsx:211`、`exceptions/index.tsx` 重复 3 份 —— **P1** —— 修法：统一 `useMutationAction`（C5）内部：表单校验错误静默返回、API 错误走 `handleApiError`、未知错误兜底 `message.error`；禁止在组件里写 `console.error` 作为唯一反馈。
28. `src/pages/asset/reconciliation/exceptions/index.tsx:421` —— 表格排序变化时 `message.success("已按 X 升序")` 弹提示，每次点表头都弹 —— **P2** —— 修法：删除该提示，排序状态由表头箭头表达即可。

### 3.7 手写重复逻辑

29. `src/pages/operations/assets/index.tsx:687-691` —— 工具栏"下载模板"与"导入"两个按钮都调用 `setImportVisible(true)`，模板下载功能实际不存在 —— **P0** —— 修法：`ExcelImportLazy` 内部已支持模板下载（`components/shared/ExcelImportLazy.tsx`），把"下载模板"改为 `excelApi.downloadTemplate("asset")`，或直接删除该按钮。
30. `src/pages/operations/buildings/index.tsx:628-677`（50 行）、`dedicated-lines/index.tsx:642-687`（46 行）、`server-rooms/index.tsx:505-551`（47 行）、`info-points/index.tsx:933-1010+`、`room-devices/index.tsx`、`assets/index.tsx:682-708` —— 工具栏 JSX 逐行同构：查看模式 `Radio`、导入、导出、批量删除、新增，其中"导出"还要在页面内内联 `Object.fromEntries(Object.entries(filters).filter(([,v]) => v !== undefined && v !== null && v !== ""))` 过滤空值（`buildings/index.tsx:646-661`、`dedicated-lines/index.tsx:658-674` 各一份）—— **P1** —— 修法：抽 `ListToolbar`（C2），props 化 `onImport/onExport/onCreate/viewModes/batch`，导出参数过滤下沉到 `ListToolbar` 或 `useListPage.buildExportParams()`。
31. `src/pages/operations/buildings/index.tsx:679-698`、`dedicated-lines/index.tsx:689-708`、`room-devices/index.tsx:652-671` —— 选中行后的 `Alert`（已选 N 项 + 取消选择 + 批量删除）三段同构代码，样式与按钮位置各写一遍 —— **P1** —— 修法：抽 `SelectionAlert`（并入 C2，或在 `useListPage` 暴露 `selectionAlert` 渲染函数）。
32. `src/components/shared/` 下 `BatchDeleteButton.tsx`、`BatchExportModal.tsx`、`EmptyStateWithAction.tsx`、`ErrorAlertWithRetry.tsx`、`ModernTag.tsx`、`GlobalSearch.tsx` 在本分片使用次数为 **0**；`BaseEditModal` 仅 `workstations/modals/EditModal.tsx:24` 一处、`ColumnConfigModal` 仅 `assets` 一处 —— 公共组件与页面实现"双轨"，维护者改组件不影响页面 —— **P1** —— 修法：不是继续新增组件，而是给这批组件补"使用示例 + 类型收紧"并纳入 code review 清单；90 天内无使用者的组件（`ModernTag`、`GlobalSearch`）下架或与现有实现合并。

### 3.8 硬编码

33. `src/pages/vdi/VirtualMachineList/index.tsx:638-641`（`DEFAULT_VTP_ID = 1`、`DEFAULT_RESOURCE_GROUP_ID = "0"`、`DEFAULT_RESOURCE_NAME = "VDI模板"`、`DEFAULT_POSITION_NAME = "研发"`）与 `:1207-1229`（快速创建时按"研发"位置 + **第一个可用**存储/网络静默选择，UI 仅以"自动选择第一个可用"字样提示）—— 用户可能在不知情下把虚拟机建到"研发"OU 与随机存储上，属于数据风险 —— **P0** —— 修法：默认值改从配置/字典读取（`sys_config` 或 `vdi` 字典），快速创建必须显式展示将要使用的位置/存储/网络并要求确认；无配置时阻断创建并提示配置入口。
34. 冲突类型中文映射 4 份：`src/components/asset/reconciliation/ExceptionRuleForm.tsx:50-56`、`src/components/reconciliation/ReconciliationTimeline.tsx:36-43`、`src/pages/asset/reconciliation/dashboard/index.tsx:49`、`src/pages/asset/reconciliation/fix-suggestion/index.tsx:495-499`（`["A"..."F"].map(v => "Type " + v)`，与前三份中文标签不一致）—— 同一个 A-F 在不同页面显示不同文案 —— **P1** —— 修法：`src/constants/reconciliation.ts` 导出 `CONFLICT_TYPE_LABELS` / `SEVERITY_LABELS` / `CONFLICT_TYPE_OPTIONS`，全分片唯一来源。
35. 日期时间格式化三套：`dayjs(x).format("YYYY-MM-DD")` / `("YYYY-MM-DD HH:mm:ss")` 12 处（`assets/index.tsx:412`、`421`、`433`、`494`、`502`、`517`、`525`、`rpa/workers/columns.tsx:17`、`exceptions/index.tsx:226`、`ExceptionMatchList.tsx:130`、`ReconciliationTimeline.tsx:64`、`68`）、`new Date(x).toLocaleString("zh-CN")` 6 处（`exception-rules/index.tsx:277`、`VDIServerConfig/index.tsx:189`、`196`、`VirtualMachineDetail.tsx:82`、`VirtualMachineList.tsx:855`、`src/components/asset/reconciliation/MatchTestPanel.tsx:145`）、`createDateTimeColumn` 9 文件 —— 同一时间字段在不同页显示不同精度/时区格式 —— **P1** —— 修法：抽 `src/utils/formatters.ts`（`formatDateTime` / `formatDate` / `formatRelative`），`createDateTimeColumn` 复用之；禁止页面内直接 `toLocaleString`。
36. hex 颜色字面量 **351 处 / 45 文件**，与 `var(--theme-*, #fallback)` 混用：`HealthCard.tsx:24-28`（`#2d8949`/`#b07a20`/`#ba3630`）、`dashboard/index.tsx:309`（`#722ed1`）、`:369`、`:382`、`:395`（`#2d8949`/`#b07a20`）、`exception-rules/index.tsx:353`（`#2d8949`）、`WorkstationDeviceTable/index.tsx:503`（`#fafafa`）、`:644`（`#f5f5f5`）、`room-devices/index.tsx:641`（`var(--theme-error, #ba3630)`）、`assets/index.tsx:596-615`（tailwind `text-green-600`/`text-red-600`）—— 深色模式下列表、标签、统计卡颜色不反转 —— **P1** —— 修法：颜色收敛到 `src/design-system` token（`--color-success/warning/error` + tailwind 主题映射），CI 加 `no-hex-color` lint（允许 `var(--x, #fallback)` 形式并逐步清理 fallback）。
37. `src/pages/asset/reconciliation/exception-rules/index.tsx:470-485`（`actionTagColor` 用 switch 手写 5 个动作颜色）、`src/pages/operations/room-devices/index.tsx:641`、`src/pages/operations/assets/index.tsx:687` —— 业务动作/状态的文案与颜色散落在各页面，不走字典—— **P2** —— 修法：动作/状态文案统一走字典（`useDict` 已存在，本分片 7 处使用），颜色映射放 `constants/`。

### 3.9 交互细节

38. `src/pages/asset/reconciliation/fix-suggestion/index.tsx:525-528`（`onRow` 整行点击打开 Drawer，无 `tabIndex`/`onKeyDown`，键盘与读屏不可达）、`src/components/reconciliation/ReconciliationTimeline.tsx:59`（`dot={<HealthBadge ... onClick={() => {}} />}` 空回调，看起来可点实际无反应）—— **P2** —— 修法：行点击保留但补 `tabIndex={0}` + `onKeyDown`（Enter 触发），或在行尾提供显式"详情"按钮；空回调的组件去掉 hover/pointer 样式或补真实行为。
39. `src/components/reconciliation/ReconciliationDrawer.tsx:100` —— 跳转用 `window.open(url, "_blank")` 新开页签，破坏 SPA 会话与面包屑，且被浏览器拦截时无提示 —— **P2** —— 修法：改用 `useNavigate()` 或 `<Link>`，需要新窗口时用 `<a target="_blank" rel="noreferrer">`（可被浏览器识别为用户手势）。
40. `src/pages/operations/buildings/index.tsx:152`、`floors/index.tsx:238`、`253`、`285`、`304`、`329`、`350`、`371`、`443`、`workstations/index.tsx:309`、`385`、`402`、`447`、`471`、`537`、`assets/index.tsx:145`、`157`、`179`、`WorkstationDeviceTable/index.tsx:116-130` 等 **40 处** —— `console.error` 作为错误出口（生产环境刷屏、无上报）—— **P2** —— 修法：删除或改走 `handleApiError`；确需保留的用统一 `logger`（仅在 `import.meta.env.DEV` 输出）。

### 3.10 类型与 API 调用

41. `src/components/reconciliation/ReconciliationDrawer.tsx:102` —— `useCallback` 的依赖数组包含整个 `asset` 对象（每次父级刷新都变化），注释称"React compiler 自动推断"与项目 `useEffect` 稳定性规范冲突 —— **P1** —— 修法：依赖改成 `asset?.id` 等原始值，或改用 `useEventCallback`；同时把 `window.open` 的 URL 构造抽成纯函数。
42. `src/pages/vdi/VirtualMachineList/index.tsx:46` —— 自造 `hasPermission = (permissions, perm) => permissions.includes(perm)`，绕过 `usePermission` hook（项目已有），权限逻辑无法统一演进 —— **P1** —— 修法：改用 `usePermission()` / `hasPerm` 工具函数，禁止页面内自造权限判断。

---

## 4. 公共组件抽取候选与必要性评估

估算方法：`可减行数 = 每处重复行数 × 复用处数 - 新抽象自身行数`。行数只是次要指标，**主要收益是缺陷根因收敛**（一处修全部修）与**范式统一**（新人只需学一套）。

### 4.1 候选总表

| # | 候选组件 / Hook | 本模块复用次数 | 涉及文件数 | 可减代码量估算 | 结论 |
| --- | --- | --- | --- | --- | --- |
| C1 | `useListPage`（合并 `useTableManager` + `usePagination`） | 16 个列表页 | 12 | 每页 ~25 行 × 16 ≈ 400 行，减去 hook 自身 ~250 行 → **净减 ~150 行**；同时消灭 3 套范式 | **必抽** |
| C2 | `ListToolbar`（导入/导出/批量删除/新增/列设置 + 导出参数过滤） | 6 个列表页 | 6 | 每页 ~45 行 × 6 ≈ 270 行，组件 ~120 行 → **净减 ~150 行** | **必抽** |
| C3 | `FilterBar`（搜索表单 + 回车提交 + 搜索/重置/刷新 + 折叠） | 10 个列表页 | 10 | 每页 ~30 行 × 10 ≈ 300 行，组件 ~150 行 → **净减 ~150 行** | **必抽** |
| C4 | `useDeleteConfirm` + `SelectionAlert` | 23 处确认 / 3 处选中条 | 16 | 每处 ~8 行 × 23 ≈ 184 行，hook ~60 行 → **净减 ~124 行** | **必抽** |
| C5 | `useMutationAction`（提交 loading / 防连点 / 错误分流 / 刷新） | ~40 个写操作 | 25 | 每处 ~7 行 × 40 ≈ 280 行，hook ~80 行 → **净减 ~200 行** | 应统一 |
| C6 | `SafeModal`（升级现有 `BaseEditModal`） | 23 个弹窗 | 17 | 每处 ~6 行 × 23 ≈ 138 行，组件 ~100 行 → **净减 ~38 行**（价值在属性齐全） | 应统一 |
| C7 | `RemoteSelect`（防抖 + 远程搜索 + `allowClear` + 空态/加载态） | ~20 处 | 12 | 每处 ~10 行 × 20 ≈ 200 行，组件 ~120 行 → **净减 ~80 行** | 应统一 |
| C8 | `StatusTag` + `statusDict`（状态/严重度/冲突类型字典） | 30+ 处 | 15 | 每处 ~4 行 × 30 ≈ 120 行，常量+组件 ~60 行 → **净减 ~60 行** | 应统一 |
| C9 | `formatters`（`formatDateTime` / `formatDate` / `formatFileSize` / `formatPercent`） | 18 处 | 12 | 每处 ~2 行 × 18 ≈ 36 行，工具 ~60 行 → **净增 24 行**，但换来全站时间格式一致 | 应统一 |
| C10 | `PageState`（loading/empty/error 三态，重设计现有两个闲置组件） | 10+ 页 | 10 | 每页 ~12 行 × 10 ≈ 120 行，组件 ~80 行 → **净减 ~40 行** | 应统一 |
| C11 | `useDebouncedSearch` | 3 处 | 3 | ~20 行 | **不建议独立**（并入 C7） |
| C12 | `usePermission` 统一权限判断 | 5 处 | 5 | ~10 行 | **不建议新建**（沿用现有 hook，仅清理自造实现） |

> 合计：33 处重复代码被消除（约 1,900 行），扣除新抽象约 1,000 行，**净减约 900 行**（本分片生产代码的 3.5%）。真正的收益不在这 3.5%，而在：分页总条数这类缺陷从"改 11 个页面"变为"改 1 个 hook"；新页面从"抄一页改改"变为"组合 4 个标准件"。

### 4.2 必抽项论证

**C1 `useListPage`（必抽，P0）**

- 现状：`useTableManager.ts`（410 行）+ `usePagination.ts`（140 行）已经是 80% 的答案，但只被 11 页使用；`VDIServerConfig`、`VirtualMachineList` 走了 TanStack Query 手写路线（`VirtualMachineList/index.tsx:54-70` 22 个 `useState`），`ComponentListTab`、`LocationAliasDrawer`、`WorkstationDeviceTable`、`MatchTestPanel` 完全手写。
- 之所以"必抽"而非"应统一"：分页总条数缺陷（`usePagination.ts:122`）证明**任何留在页面里的样板都会各自长歪**。`useListPage` 需要把 `showTotal 共 N 条`、`sizeOptions`、`rowKey`、`scroll/virtual` 阈值、`sortOrder`、`rowSelection`、`exportParams` 全部内建，使页面无法再写错。
- 关键设计：内部用 TanStack Query（`keepPreviousData` + `placeholderData`）承接数据层，对外暴露 `{ data, total, paginationProps, tableProps, search, reset, refresh, selection, buildExportParams }`；`useTableManager` 降级为其内部实现细节，而非并列的第二套 API。

**C2 `ListToolbar`（必抽，P1）**

- 现状：`buildings/index.tsx:628-677`、`dedicated-lines/index.tsx:642-687`、`server-rooms/index.tsx:505-551`、`info-points/index.tsx:933+`、`room-devices`、`assets/index.tsx:682-708` 六处逐行同构，连"导出前内联过滤空值"的写法都复制了两份（`buildings/index.tsx:646-661`、`dedicated-lines/index.tsx:658-674`）。
- 为什么不是"应统一"：工具栏是**每页必有**的结构件，且直接承载 P0 缺陷（`assets/index.tsx:687-691` 下载模板与导入同源）。抽成组件的同时把"按钮动作"变成声明式数组（权限、禁用原因、确认文案一并声明），才能防止下一个假按钮。

**C3 `FilterBar`（必抽，P1）**

- 现状：10 个列表页各自写 `<Form layout="inline">` + 搜索/重置/刷新三按钮（`buildings/index.tsx:596-627`），无一处 `onPressEnter`（全分片 0 处），`onSearch` 死代码 30+ 处，远程搜索防抖 3 份实现（其中 `workstations/index.tsx:774` 漏防抖）。
- 抽法要点：`FilterBar` 接收 `items` 声明（`{ name, label, type: "input" | "select" | "remote" | "dateRange" | "tree", ... }`），内部统一注入 `allowClear`、`showSearch/optionFilterProp`、`onPressEnter`、`loading`；这样 P2 类"缺 allowClear"问题从"逐处修 110 个"变成"修 1 个渲染器"。

**C4 `useDeleteConfirm`（必抽，P1）**

- 现状：`Modal.confirm` 16 处（`assets/index.tsx:209`、`buildings/index.tsx:444`、`dedicated-lines/index.tsx:396`、`server-rooms/index.tsx:357`、`workstations/columns.tsx:132` 等）与 `Popconfirm` 7 处并存，且 `Modal.confirm` 版本各自复制 `onOk: async () => { await x; message.success; refresh }`。
- 抽法要点：`useDeleteConfirm({ onConfirm, successText, refresh })` 返回 `confirm(record)`，内部自动：危险按钮、`loading` 防重复、`handleApiError` 失败提示、成功后 `refresh()`；批量删除复用同一实现（与 `BatchDeleteButton` 合并）。

### 4.3 应统一项说明

- **C5 `useMutationAction`**：不是为了减行，而是为了消灭 `catch` 三套实现。`VDIServerConfig/index.tsx:79-119` 用固定文案吞掉后端错误，`dedicated-lines/index.tsx:198-214` 无 `submitting`，`ExceptionRuleForm.tsx:118` 又单独判断 `errorFields`。统一后：校验错误静默、业务错误透传后端 message、未知错误兜底、成功统一 `handleSuccess`、成功后自动 `refresh` 或 `invalidateQueries`。
- **C6 `SafeModal`**：`BaseEditModal.tsx`（65 行）方向正确但只被 `workstations/modals/EditModal.tsx:24` 使用。升级点：`size` 四档替代裸 `width`、强制 `confirmLoading`、内置 `destroyOnHidden`、关闭前脏值确认（`Form.isFieldsTouched`）、错误分流交给 C5。**不要**新建第二个 Modal 组件，直接改造现有组件并迁移。
- **C7 `RemoteSelect`**：合并 3 份防抖实现与 30+ 处空 `onSearch`；内建 `debounce=300`、`filterOption=false`、`notFoundContent`（空态"无匹配项"、加载态 `Spin`）、`onOpenChange` 首次加载、`allowClear`。`info-points/index.tsx:863-883` 的工位远程搜索是最好的参照实现。
- **C8 `StatusTag` + `statusDict`**：状态枚举在 5+ 页各写一遍 `<Option>`（`buildings/index.tsx:612-613` 等），冲突类型中文映射有 4 份且互相不一致（`fix-suggestion/index.tsx:495-499` 显示 `Type A`，其它显示中文）。字典集中后，后端新增枚举只需改一处。
- **C9 `formatters`**：`dayjs.format` 12 处、`toLocaleString` 6 处、`createDateTimeColumn` 9 文件三套并存（`ReconciliationTimeline.tsx:64`、`VDIServerConfig/index.tsx:189` 等）。附带收益：统一时区与秒级精度，避免"同一工单在两个页面时间不同"的错觉。
- **C10 `PageState`**：`EmptyStateWithAction.tsx`、`ErrorAlertWithRetry.tsx` 已存在但 0 使用——说明**签名与调用场景不匹配**。重设计为一个 `PageState`：`status: "loading" | "error" | "empty" | "ready"` + `onRetry` + `emptyAction`，默认渲染 `Skeleton`/`ErrorAlertWithRetry`/`EmptyStateWithAction`；错误态必须带重试（`exceptions/index.tsx:430-436` 恰恰缺）。

### 4.4 不建议抽取的部分

- **不同意把"页面级业务组合"继续抽象成配置化表格（如全功能 ProTable 克隆）**：本分片差异最大的不是表格，而是每页的业务动作（RPA 执行重跑、工单派发、VDI 绑定用户）。配置化会把差异推进 `if/else`，比手写更难维护。`useListPage` 只收敛"分页 / 排序 / 三态 / 选择"，业务动作仍由页面组合声明式工具栏。
- **不建议为 `useDebouncedSearch`、`usePermission` 单独建模块**（C11/C12）：一个并入 `RemoteSelect`，一个直接用现有 hook 并清理 `VirtualMachineList/index.tsx:46` 的自造实现。为 3-5 个调用点新建公共模块，是把"重复代码"换成"重复的抽象层"。

---

## 5. 最佳实践方案

本模块最缺的不是"更多组件"，而是三个**收口点**：列表页收口、提交/弹窗收口、搜索区收口。下面只给这三处的骨架，其余为项目级通用规范，不在此重复。

### 5.1 收口点一：`useListPage` + 标准列表页

目标：页面里不再出现 `current/pageSize/total/loading` 与 `showTotal` 的痕迹。

```ts
// src/hooks/useListPage.ts
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { usePagination } from "./usePagination";

export interface UseListPageOptions<T, Q extends object> {
  queryKey: readonly unknown[];
  fetcher: (params: Q & PageParams) => Promise<PageData<T>>;
  defaultQuery?: Q;
  rowKey?: keyof T & string;
  sizeOptions?: string[];
  scrollX?: number;
  virtualThreshold?: number;
}

export function useListPage<T, Q extends object>(opts: UseListPageOptions<T, Q>) {
  const { current, pageSize, setPage, paginationProps } = usePagination({
    sizeOptions: opts.sizeOptions ?? ["10", "20", "50", "100"],
  });
  const [query, setQuery] = useState<Q>(opts.defaultQuery ?? ({} as Q));
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const result = useQuery({
    queryKey: [...opts.queryKey, current, pageSize, query],
    queryFn: () => opts.fetcher({ ...query, current, pageSize }),
    placeholderData: keepPreviousData,
  });

  return {
    ...result,
    list: result.data?.list ?? [],
    total: result.data?.total ?? 0,
    query, search: (next: Q) => { setQuery(next); setPage(1); },
    reset: () => { setQuery(opts.defaultQuery ?? ({} as Q)); setPage(1); },
    selection: { selectedRowKeys, setSelectedRowKeys },
    paginationProps,
    tableProps: {
      rowKey: opts.rowKey ?? "id",
      loading: result.isLoading,
      dataSource: result.data?.list ?? [],
      pagination: paginationProps,
      scroll: { x: opts.scrollX, y: 600 },
      virtual: opts.virtualThreshold ? (total) => total > opts.virtualThreshold! : undefined,
    },
    buildExportParams: () => ({ ...query, ...pickNonEmpty(query) }),
  };
}
```

页面侧只剩组合：

```tsx
const page = useListPage<Building, BuildingQuery>({
  queryKey: queryKeys.building.list(),
  fetcher: buildingApi.list,
  defaultQuery: { name: undefined, status: undefined },
  scrollX: 1600,
  virtualThreshold: 100,
});

return (
  <Card>
    <FilterBar items={buildingFilters} onSearch={page.search} onReset={page.reset} />
    <ListToolbar
      viewModes={["table", "card"]}
      onImport={() => setImportVisible(true)}
      onExport={() => excelApi.export("building", page.buildExportParams())}
      onCreate={() => openModal(null)}
      batch={<BatchDeleteButton selection={page.selection} onSuccess={page.refetch} />}
    />
    <Table {...page.tableProps} columns={columns} />
    <SafeModal open={modal.open} confirmLoading={mutation.isPending} />
  </Card>
);
```

关键约束（写进 review checklist）：
1. 页面禁止直接 import `usePagination` / 自己声明 `current/pageSize`；
2. `paginationProps` 内建 `showTotal: (t, r) => 共 ${t} 条（第 ${r[0]}-${r[1]} 条）`、`showSizeChanger`、`showQuickJumper`、`sizeOptions`；
3. 导出参数一律 `buildExportParams()`，页面不写 `filter(v => v !== "")`；
4. `virtual` 阈值由页面声明，`useListPage` 应用，避免 `scroll` 与 `virtual` 混用。

### 5.2 收口点二：`useMutationAction` + `SafeModal` + `useDeleteConfirm`

```ts
// src/hooks/useMutationAction.ts
export function useMutationAction<TVars, TResult>(opts: {
  action: (vars: TVars) => Promise<TResult>;
  successText?: string | ((r: TResult) => string);
  onSuccess?: (r: TResult) => void;
}) {
  const { message } = App.useApp();
  const mutation = useMutation({
    mutationFn: opts.action,
    onSuccess: (r) => {
      const text = typeof opts.successText === "function" ? opts.successText(r) : opts.successText;
      if (text) message.success(text);
      opts.onSuccess?.(r);
    },
    onError: (err) => {
      if (isFormValidationError(err)) return; // 表单校验失败由 Form 自带提示，不弹 toast
      handleApiError(err, "操作失败");   // 统一：透传后端 message + 401/403 兜底
    },
  });
  return { run: mutation.mutateAsync, isPending: mutation.isPending };
}
```

配套 `SafeModal`（改造现有 `BaseEditModal`，不新建第二个组件）：

```tsx
<SafeModal
  open={open}
  size="md"                    // sm 520 / md 680 / lg 800 / xl 960，禁止裸 width
  title={record ? "编辑工位" : "新增工位"}
  confirmLoading={mutation.isPending}   // 由 props 强制，调用方无法漏传
  destroyOnHidden
  onOk={async () => {
    const values = await form.validateFields();  // 校验失败自动被吞，不进 catch 弹窗
    await mutation.run(values);
    onClose();
  }}
  onCancel={onClose}
/>
```

`useDeleteConfirm`（替代 16 处 `Modal.confirm` 与 7 处 `Popconfirm`）：

```ts
const { confirm } = useDeleteConfirm({ onConfirm: (row) => buildingApi.delete(row.id), refresh: page.refetch });
<ActionButtons actions={[{ key: "delete", label: "删除", danger: true, onClick: () => confirm(record) }]} />
```

### 5.3 收口点三：`FilterBar` + `RemoteSelect`（搜索区一次写好）

```tsx
const buildingFilters: FilterItem[] = [
  { name: "name", label: "楼宇名称", type: "input", placeholder: "模糊搜索" },
  { name: "status", label: "状态", type: "select", options: STATUS_OPTIONS },
  { name: "orgId", label: "所属部门", type: "tree", tree: "dept" },
  { name: "floorId", label: "楼层", type: "remote", remote: floorRemoteSearch, debounce: 300 },
];

// FilterBar 内部统一保证（页面无法写错）：
// 1. 每个可清空控件自动 allowClear；
// 2. Select 自动 showSearch + optionFilterProp="label"；
// 3. Form onPressEnter 触发 onSearch（全片当前 0 处）；
// 4. remote 类型由 RemoteSelect 承载 debounce/filterOption=false/notFoundContent；
// 5. 搜索/重置/刷新三按钮位置与图标固定。
```

> 反例清单（评审时直接拒）：`onSearch={() => {}}`、`<Select showSearch>` 无过滤策略、`<Input placeholder="部门 ID">`、`Form.Item` 内混入"新建/导出"等操作按钮（`exception-rules/index.tsx:405-424`）。

---

## 6. 落地路线图

### 6.1 批次划分

| 批次 | 周期 | 任务 | 验收口径（可脚本验证） |
| --- | --- | --- | --- |
| **P0** | 第 1-2 周 | ① 修 `usePagination.ts:122` 的 `showTotal` 并补 `sizeOptions`；② 抽 `useListPage` 并在 buildings / server-rooms / workstations 三页试点；③ 修 `assets/index.tsx:203` 编辑假入口、`:687-691` 模板下载；④ `ReconciliationDrawer` 两个空 Tab 先隐藏、后接数据；⑤ 抽 `useMutationAction`，补齐 23 个弹窗的 `confirmLoading`；⑥ `VirtualMachineList` 静默默认值改为显式确认；⑦ `fix-suggestion` 责任部门、`ExceptionRuleForm` 的 `scopeId` 改选择器 | ① `rg "共 \$\{[^}]*\} 页" src` = **0**，6 处手写分页全部为"共 N 条"；② 23 个 Modal/Drawer `confirmLoading` 覆盖率 **100%**；③ `rg "编辑功能待实现"` = **0**；④ `rg "onSearch=\{\(\) =>"` = **0**；⑤ 三页 `useListPage` 试点页不再出现 `current/pageSize` 声明 |
| **P1** | 第 3-5 周 | ① 落地 `FilterBar` / `ListToolbar` / `RemoteSelect` / `StatusTag` + `statusDict` / `PageState`，迁移剩余 13 个列表页；② `message` 静态调用全部改 `App.useApp()`；③ 错误处理统一到 `handleApiError`，清理 40 处 `console.error`；④ 颜色收敛到 design token；⑤ `formatters` 替换 18 处日期格式化；⑥ Modal 宽度四档化 | ① `rg "import \{ message \} from \"antd\"" src/pages src/components` = **0**；② hex 六位色值在本分片 **< 50** 处（当前 351）；③ `rg "toLocaleString" src` = **0**；④ `rg "console.error" src/pages/(operations|asset|vdi) src/components/(operations|asset|reconciliation)` = **0**；⑤ 列表页 `showTotal` 100% 为"共 N 条" |
| **P2** | 第 6-8 周 | ① `allowClear` 全覆盖并 lint 化（禁止无 `allowClear` 的 `Input/Select/InputNumber/TextArea`）；② 键盘可达性（行点击补 `tabIndex`/`onKeyDown`，删除空回调）；③ 空格/加载态统一（Modal 内用 `Skeleton` 而非整页 `Spin`）；④ 下架 90 天无使用者的公共组件（`ModernTag`、`GlobalSearch`）或补齐文档；⑤ 产出"新增列表页 Checklist"文档并纳入 PR 模板 | ① 脚本统计 `allowClear` 覆盖率 **≥ 95%**（当前 22%）；② `rg "pagination=\{\{" src` = **0**（分页必须来自 `useListPage`）；③ 行点击组件均含 `tabIndex`；④ Checklist 文档合并进 `docs/开发规范.md` |

### 6.2 迁移顺序与回滚

1. **先修根因，再抽组件**：`usePagination.ts:122` 一行改动即可让 11 页同时正确，先做这个建立信心，再动结构性重构。
2. **每批最多迁移 3 个页面**，每页一个 PR，保留旧实现一个迭代；`useTableManager` 在并存期不改签名，只作为 `useListPage` 的内部实现，避免大面积回归。
3. **试点页选"最像标准页"的**：buildings（有搜索/工具栏/批量/导入导出/Modal）作为模板页，server-rooms（简单）验证最小配置，workstations（三态 + 平面图）验证复杂分支。
4. **回滚策略**：`useListPage` 与 `SafeModal` 均可在页面级一行切回旧实现（保留 `useTableManager` 与 `BaseEditModal` 导出）；颜色 token 化以"页面级 PR"推进，可单页回滚。

### 6.3 责任与度量

- **框架 owner**：`src/hooks/useListPage.ts`、`src/components/shared/` 由一人整体负责，避免多人并行改同一抽象层；页面迁移由模块负责人各自认领。
- **度量**：每迭代末跑一次统计脚本（`allowClear` 覆盖率、`showTotal` 正确率、`confirmLoading` 覆盖率、hex 色值数、`console.error` 数、`onSearch` 空函数数），进入迭代报告，只允许单调收敛。
- **不追求的指标**：组件数量、抽出的 hook 行数。**追求的指标**：新页面手写样板行数（目标从 ~450 行降到 ~120 行）、同类缺陷的修复点数量（从 11 处降到 1 处）。
