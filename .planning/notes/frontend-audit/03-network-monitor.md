# 03 · Network / Monitor 分片组件审查

> 审查人：teammate `audit-network`（只读审查，未修改任何源码）
> 审查对象：`xingran-react-frontend/src/{pages/network, pages/monitor, components/network, components/CronSelector}`
> **行号约定**：所有 `文件:行号` 均相对 `xingran-react-frontend/`，以 `src/` 开头，可直接 grep 定位。
> **本版口径**：按 Lead 定稿模板的 10 个维度重排问题清单，每条标注 `[维度名]`；只讲本模块事实，全局统计数字（Input 总数、全前端 allowClear 率等）见主报告。

---

## 1. 审查范围与方法

### 1.1 范围

| 目录 | .ts/.tsx | 测试 | 非测试源文件 | 非测试源码行数 | 最大文件 |
|---|---|---|---|---|---|
| `src/pages/network` | 109 | 47 | 62 | 约 11,600 | `src/pages/network/devices/index.tsx`（1243 行） |
| `src/pages/monitor` | 48 | 22 | 26 | 约 4,000 | `src/pages/monitor/cache/index.tsx`（918 行） |
| `src/components/network` | 18 | 9 | 9 | 约 1,700 | `src/components/network/port-write/BulkWriteDrawer.tsx`（532 行） |
| `src/components/CronSelector` | 20 | 10 | 10 | 约 1,100 | `src/components/CronSelector/utils.ts`（379 行） |
| **合计** | **195** | **88** | **107** | **18,321** | — |

### 1.2 方法

1. **全量静态扫描（107 文件）**：脚本解析 JSX 开标签（含 `<Table<T>>` 泛型、跨行属性、嵌套花括号），统计 `allowClear` / `showSearch` / `scroll` / `loading` / `rowKey` / `locale` / `destroyOnHidden` / `okButtonProps` / `maskClosable` / `onPressEnter` / 空 `onSearch` / `setTotal` / `showTotal` / `sorter` / `any` / hex 颜色 / `console.*` / `setInterval` 等 50+ 指标。第一版按行 grep 会把 `<Select.Option>` 计入 Select、把 `<Table<T>` 泛型 `>` 当标签结束，本版已修正解析器并重新核算（与首版有 ±10% 差异，以本版为准）。
2. **代表页逐行精读（8 个）**：`devices`(1243)、`ports`(835)、`monitor/logs`(557)、`monitor/job`(343)、`monitor/cache`(918)、`monitor/server`(495)、`backups` 与 `discoveries`（精读分页链路）。
3. **公共设施全文读（7 个）**：`src/hooks/usePagination.ts`、`src/hooks/useTableManager.ts`、`src/utils/tableHelpers.tsx`、`src/components/shared/ActionButtons.tsx`、`NetworkExport.tsx`、`EmptyStateWithAction.tsx`、`ErrorAlertWithRetry.tsx`。
4. **交叉验证**：对“分页 total 是否回填”“批量删除是否有确认”“排序是本地还是服务端”三类结论，逐个文件回溯调用链，未确认的标“疑似”并写明依据。

### 1.3 本模块事实基线

| 项 | 本模块事实 |
|---|---|
| 列表页 / `<Table>` | 26 个列表页 / **25 个 Table**（22 文件，含 Drawer 与 Modal 内嵌表） |
| `rowKey` | **25/25 = 100%**（正面结论，无一处缺失） |
| `loading` | 15/25 有，**10 个缺失** |
| `scroll` | 19/25 有，**6 个缺失** |
| 分页 | `usePagination` 12 文件 / 36 处；`useTableManager` 5 文件；**4 处手写 pagination** |
| `showTotal` | 默认由 `usePagination` 提供“共 N 页”（**不显示总条数**）；另有 4 处手写“共 N 条” |
| 服务端排序 | `useServerSort` 11 文件、`createSorterMeta` 52 处、`useTableManager` 自动注入 `orderByColumn/isAsc` |
| 本地排序误用 | `cache`（7 列）/ `job`（6 列）在分页表上用 `createSorter` 本地排序；`server` 本地+服务端双轨 |
| `<Input>` 类控件 | **53 个**（21 文件，含 `Input.Password`/`TextArea`），缺 `allowClear` 31 个，其中表单/搜索区可清除候选 **21 个** |
| `<Select>` 控件 | **82 个**（28 文件，已排除 `Select.Option`），缺 `allowClear` **51 个**、缺 `showSearch` **77 个**（可搜仅 5 个） |
| 空 `onSearch` | **72 处 / 24 文件**（其中 24 处在 CronSelector 纯数字选择器，48 处为业务下拉） |
| `<Modal>` | **22 个**（17 文件）：`okButtonProps` 5/22、`destroyOnHidden` 6/22、`maskClosable` **0/22**、关闭脏检查 **0/22** |
| 危险操作 | 单条删除有确认（`Modal.confirm` 14 处 + `Popconfirm` 1 处）；**批量删除确认 0/5** |
| 空态 / 错误态 | `locale.emptyText` **0/25**；`EmptyStateWithAction` 8 处/2 文件；`ErrorAlertWithRetry` 11 处/4 文件；`console.*` 52 处/24 文件 |
| 样式硬编码 | 6 位 hex **109 处 / 25 文件**（含 3 位简写共 125 行命中）；内联 `style` 74 处/19 文件 |
| 状态 / 时间格式化 | 状态 `0/1` 裸三元 13 处/6 文件；手写日期格式 12 处/8 文件 |
| 可访问性 / 国际化 | `aria-*` 0、`role=` 0、`tabIndex` 0、`onKeyDown` 0、`useTranslation` 0 |
| 类型 | `: any` / `as any` 27 处/16 文件 |

---

## 2. 问题总览

### 2.1 数量统计

| 维度 | 问题数 | P0 | P1 | P2 |
|---|---|---|---|---|
| 3.1 功能完整性 | 4 | 1 | 2 | 1 |
| 3.2 数据展示完整性 | 3 | 0 | 2 | 1 |
| 3.3 交互反馈 | 6 | 1 | 4 | 1 |
| 3.4 表单校验 | 4 | 0 | 2 | 2 |
| 3.5 状态与分页 | 7 | 2 | 3 | 2 |
| 3.6 错误与空态 | 3 | 0 | 2 | 1 |
| 3.7 样式与主题一致性 | 4 | 0 | 0 | 4 |
| 3.8 可访问性与国际化 | 4 | 0 | 1 | 3 |
| 3.9 性能与副作用 | 4 | 0 | 0 | 4 |
| 3.10 代码复用与可维护性 | 6 | 0 | 0 | 6 |
| **合计** | **45** | **4** | **16** | **25** |

### 2.2 严重度分布

- **P0（4 条，阻断性 / 用户马上会踩）**：分页不显示总条数（`showTotal` 口径错）、2 个列表页 total 恒 0 翻不到第 2 页、5 处批量删除无二次确认、48 处业务下拉“假搜索”。
- **P1（16 条，功能残缺或误导）**：表格无 loading / 无 scroll、弹窗无提交 loading / 无脏检查 / 可点遮罩误关、分页表上做本地排序、错误只进 console、表单校验覆盖不足、图标按钮无可访问名。
- **P2（25 条，一致性 / 可维护性债务）**：样式硬编码、日期/状态格式化重复、死代码、`any`、`index` 作 key、数据获取四套并存等。

### 2.3 Top3 最严重问题

| 排名 | 问题 | 证据 | 用户可见影响 | 为什么排第一梯队 |
|---|---|---|---|---|
| 1 | **列表页翻不了页 / 看不到总条数** | `src/hooks/usePagination.ts:122` 的 `showTotal` 渲染“共 N 页”；`src/pages/network/backups/index.tsx:79`+`:659`、`src/pages/network/discoveries/index.tsx:34`+`:354` 的 `setTotal` 被解构为 `_setTotal` 后从未调用 | 19 个表格只显示“共 3 页”而无“共 128 条”；备份/发现页永远只有第 1 页数据，第 2 页点了是空的 | 分页是列表页最基础能力，一处 hook 口径错误 + 一处 total 未回填会同时影响全模块所有列表，且用户会误以为“数据就这么多” |
| 2 | **批量删除无二次确认** | `devices:839-847`、`credentials:462`、`mac:492`、`ports:697`、`templates:320` 直接调 API；对照单条删除有 `Modal.confirm`（如 `devices:243`） | 多选后点“批量删除”立即执行、无撤销，误点即批量丢数据 | 数据破坏不可逆，属事故级；而单条删除有确认反衬出这是遗漏而非有意设计 |
| 3 | **48 处业务下拉“假搜索”** | 72 处 `onSearch={() => {}}`，其中 48 处业务下拉未开 `showSearch`（`devices:747/762/783/937/1047/1062/1095/1110`、`credentials:426/542/588/596`、`backups:594/825/957/982` 等） | 下拉选项多时（设备、凭据、模板）只能滚轮翻找，敲键盘无任何过滤，误以为组件坏了 | 用户第一视角的“小功能缺失”，恰好是本次审查被点名的典型；且是纯机械修复、收益巨大 |

---

## 3. 十维度问题清单

格式：`文件:行号` —— 现象 —— 级别 —— 修法。共 45 条，分布与第 2.1 节维度行逐格一致。标注 `（基线）` 的条目，数量口径来自第 1.3 节全量扫描，代表行号为本次精读复核确认。

### 3.1 功能完整性

1. [功能完整性] 48 处业务下拉"假搜索"：`Select` 未开 `showSearch` 却传 `onSearch={() => {}}`，选项多时（设备 / 凭据 / 模板 / 部门）只能滚动翻找，敲键盘无任何过滤。已复核行号：`src/pages/network/devices/index.tsx:752`、`:767`、`:788`、`:937`、`:1047`、`:1062`、`:1095`、`:1110`，`src/pages/network/credentials/index.tsx:431`、`:542`、`:592`、`:596`，`src/pages/network/backups/index.tsx:599`、`:704`、`:836`、`:969`、`:994`，`src/pages/network/ports/index.tsx:586`、`:609`、`:621`、`:633`、`:645`，`src/pages/network/mac/index.tsx:415`、`:449`，`src/pages/network/mac/history/MACHistoryPage.tsx:645`、`:660`，`src/pages/network/templates/index.tsx:271`、`:286`，`src/pages/network/templates/modals/EditModal.tsx:58`、`:68`、`:77`，`src/pages/network/discoveries/index.tsx:288`，`src/pages/network/discoveries/modals/CreateModal.tsx:47`、`:87`，`src/pages/network/command/index.tsx:206`，`src/pages/network/executions/index.tsx:214`，`src/pages/network/executions/modals/ExecuteModal.tsx:71` —— **P0** —— 修法：本地选项集用 `showSearch + optionFilterProp="label"`；远程数据源统一 `RemoteSelect`（见第 4 章 C7）；lint 禁止空 `onSearch`。
2. [功能完整性] `src/pages/network/templates/hooks/useTemplateModals.ts:129-141` —— 批量删除被封装进 hook（`handleBatchDelete(selectedRowKeys, handleSuccess)`），hook 签名不含确认参数，页面在 `src/pages/network/templates/index.tsx:320` 直接调用即发请求：确认闸门不在调用链上，任何新调用点都可静默批量删数据 —— **P1** —— 修法：把确认交互下沉进 hook（内部 `Modal.confirm`），或将签名改为必须传入 `confirm` 回调，使"无确认调用"在类型上不可表达。
3. [功能完整性] 手写分页 4 处未走 `usePagination`（基线 1.3，`usePagination` 本片 12 文件 / `useTableManager` 5 文件之外游离 4 处）—— 这 4 页没有页码 / 页大小持久化（切标签、刷新后回到第 1 页）与用户默认页大小配置，能力与其余 12 页不一致 —— **P1** —— 修法：统一迁入 `usePagination`（见 3.5 第 23 条的手写分页条目），持久化能力随之免费获得。
4. [功能完整性] `src/components/CronSelector/index.tsx:204` 及 `src/components/CronSelector/fields/{Second,Minute,Hour,Day,Week,Month}Field.tsx` 各 4 处（共 25 处，已复核）—— CronSelector 的纯数字选择器为消除 antd 告警统一传 `onSearch={() => {}}`，但控件仍渲染可聚焦输入的搜索框样式，用户会尝试输入过滤而无效果；与第 1 条同根不同场景（此处选项 ≤60 个、危害低）—— **P2** —— 修法：为 CronSelector 封装统一的 `NumberSelect` 内部组件，一次性声明 `showSearch={false}`、去掉空 `onSearch`，与业务下拉（第 1 条）分开治理。

### 3.2 数据展示完整性

5. [数据展示完整性] 6 个 `<Table>` 缺 `scroll`（基线 1.3：19/25 有、6 缺），列数多的表（设备、端口、备份）窄屏下横向挤压换行、列错位；正面参照已复核：`src/pages/network/backups/index.tsx:658`（`scroll={{ x: 1200 }}`）、`src/pages/network/discoveries/index.tsx:353`（`x: 1500`）—— 缺失清单以 1.3 扫描为准 —— **P1** —— 修法：列数 ≥6 的表统一 `scroll={{ x: <列宽合计> }}`；长期由 `useListPage.tableProps` 内建（第 4 章 C1）。
6. [数据展示完整性] `src/pages/monitor/logs/utils.tsx:13` —— `new Date(time).toLocaleString()` 手写时间格式化，与本片其余 12 处 / 8 文件 dayjs 手写格式（基线 1.3）双轨：同一字段在不同页出现不同精度与时区表现 —— **P1** —— 修法：抽 `src/utils/formatters.ts`（`formatDateTime` / `formatDate`），禁页面内 `toLocaleString`（见第 5 章）。
7. [数据展示完整性] `src/pages/monitor/cache/index.tsx:340` —— `(result.data?.size || 0).toLocaleString()` 手写字节格式化，无 KB/MB 换算、与缓存详情 Drawer 及 JSON 导出里的原始值口径不一致 —— **P2** —— 修法：并入 `formatters.ts` 的 `formatFileSize`，三处（列表 / 详情 / 导出）同源。

### 3.3 交互反馈

8. [交互反馈] **5 处批量删除无二次确认**（全部已复核）：`src/pages/network/devices/index.tsx:510`（`handleBatchDelete`）+`:839-847`（按钮直接 `onClick={handleBatchDelete}`）、`src/pages/network/credentials/index.tsx:188`+`:462`、`src/pages/network/ports/index.tsx:322`+`:697`、`src/pages/network/mac/index.tsx:226`+`:492`、`src/pages/network/templates/index.tsx:320`（转调 `useTemplateModals.ts:129`）—— 多选后单击立即批量删数据、不可撤销；对照单条删除有确认（`src/pages/network/devices/index.tsx:243` `Modal.confirm`），证明是遗漏而非设计 —— **P0** —— 修法：5 处统一接 `useDeleteConfirm`（第 4 章 C4）：危险按钮 + 数量文案 + loading 防连点 + 失败 `handleApiError` + 成功刷新。
9. [交互反馈] 22 个 `<Modal>` 均未显式关闭 `maskClosable`（基线 1.3 `maskClosable 0/22`；本片 grep `maskClosable` 0 命中，已复核）—— 默认可点遮罩关闭，表单填一半误点遮罩全部丢失 —— **P1** —— 修法：表单类 Modal 统一 `maskClosable={false}`，由 `SafeModal` 内建（第 4 章 C6）。
10. [交互反馈] 关闭脏检查 0/22（基线 1.3）—— 所有编辑弹窗关闭前不检查 `Form.isFieldsTouched()`，误关即丢稿，与第 9 条叠加放大 —— **P1** —— 修法：`SafeModal` 关闭前脏值确认（"更改尚未保存，确定关闭？"），页面无感获得。
11. [交互反馈] 提交 loading 缺失：`src/pages/monitor` 全部写操作 Modal 均无 `confirmLoading`（grep 0 命中，已复核）；network 侧 `okButtonProps` 仅 5/22（基线 1.3）—— 提交期间可反复点"确定"，产生重复创建 / 重复执行 —— **P1** —— 修法：`useMutationAction`（第 4 章 C5）产出 `isPending`，`SafeModal` 强制 `confirmLoading` 必填。
12. [交互反馈] 端口写入 4 个弹窗校验意外错误只进 console、用户点"确定"无任何反馈：`src/components/network/port-write/PortWriteModal.tsx:131`、`PortBindingModal.tsx:88`、`SetAccessVlanModal.tsx:86`、`BulkWriteDrawer.tsx:182`（均已复核）—— **P1** —— 修法：意外分支兜底 `message.error("操作失败，请重试")` 并上报，禁止 console 作为唯一出口。
13. [交互反馈] 删除确认两套交互并存：`Modal.confirm` 14 处 + `Popconfirm` 1 处（基线 1.3；`Modal.confirm` 例已复核：`src/pages/network/devices/index.tsx:243`）—— 同类"删除"在不同页是弹窗 vs 气泡，且 `Modal.confirm` 版本各自复制 `onOk: async () => { await api; message.success; refresh }` 样板 —— **P2** —— 修法：统一 `useDeleteConfirm`；`Popconfirm` 仅保留给行内轻量删除并带 `okButtonProps.loading`。

### 3.4 表单校验

14. [表单校验] Modal 表单普遍仅 `required`、缺长度 / 格式规则，提交错误依赖后端返回（基线 §2.2"表单校验覆盖不足"；已复核代表：`src/pages/network/templates/modals/EditModal.tsx:58-77` 表单区、`src/pages/network/credentials/index.tsx:542-596` 协议 / 端口表单区）—— 名称超长、端口越界等要等一次网络往返才报错 —— **P1** —— 修法：建 `src/utils/rules.ts`（`nameRule(max)`、`portRule`、`ipRule`、`cronRule`），`Form.Item` 直接引用；CronSelector 已有 `utils.ts` 解析器可复用为 `cronRule` 校验器。
15. [表单校验] 校验失败的错误分流缺失：`validateFields()` reject 后未区分"校验错误"与"意外错误"，后者仅 console（第 12 条 4 处已复核行号），前者也走同一 catch —— 校验报错与系统故障用户视角无法区分 —— **P1** —— 修法：统一用 `isFormValidationError`（`src/utils/errorHandler.ts`，`src/pages/network/discoveries/index.tsx:26` 已有正确 import 先例）分流：校验错误静默（Form 自带提示）、意外错误兜底 toast。
16. [表单校验] `catch (error: any)` 8 处（全部已复核）：`src/pages/network/command/index.tsx:135`、`credentials/index.tsx:214`、`devices/index.tsx:535`、`backups/index.tsx:262`、`mac/index.tsx:251`、`executions/index.tsx:129`、`templates/index.tsx:189`、`discoveries/index.tsx:183` —— 错误类型不收敛，无法做统一分流（第 15 条的前置障碍）—— **P2** —— 修法：随 C5 `useMutationAction` 迁移一并消除，`catch` 里不再出现裸 `any`。
17. [表单校验] `src/pages/monitor/logs/types.ts:67`（`timeRange: any`）与 `src/pages/monitor/logs/utils.tsx:49`（`processTimeRangeParams(timeRange: any, ...)`，均已复核）—— 时间区间参数类型失守，`[start, end]` 传反 / 传 null 编译期无感 —— **P2** —— 修法：定义 `type TimeRange = [Dayjs, Dayjs] | null`，`processTimeRangeParams` 收窄签名。

### 3.5 状态与分页

18. [状态与分页] `src/hooks/usePagination.ts:122` —— `showTotal: (t) => \`共 ${Math.max(1, Math.ceil(t / pageSize))} 页\``，输出"共 N 页"而非"共 N 条"（已复核原文）—— 本片 12 个接入文件（含 `backups`、`discoveries` 等）全部继承：用户看得到页数却看不到数据总量，正是主报告点名投诉的直接根因 —— **P0** —— 修法：一行改为 `showTotal: (t, range) => \`共 ${t} 条（第 ${range[0]}-${range[1]} 条）\``，12 页同时修复；`usePagination` 归 `useListPage`（C1）内建后页面无法再写错。
19. [状态与分页] **2 个列表页 total 恒 0、翻不到第 2 页**（全部已复核）：`src/pages/network/backups/index.tsx:79`（`setTotal` 解构为 `_setTotal` 后从未调用）+`:659`（`pagination={paginationProps}` 透传，`total` 停留初始 0）、`src/pages/network/discoveries/index.tsx:34`（同写法 `_setTotal`）+`:354`（同透传）；对照 `useDiscoveryData` 返回值里 `total: _total`（`discoveries/index.tsx:66`）也被弃用 —— 接口明明返回了 total，hook 的 `total` 永远是 0，`showQuickJumper` 打开也是死跳 —— **P0** —— 修法：两页在数据返回处调用 `setTotal(data.total)`（或 C1 的 `useQuery` 方案直接以 `result.data?.total` 驱动，无需 setter）。
20. [状态与分页] 10 个 `<Table>` 无 `loading`（基线 1.3：15/25 有）—— 请求期间表格静默无反馈，慢网下像"页面坏了"；代表缺失面以 1.3 扫描清单为准，正面参照 `src/pages/network/backups/index.tsx:656`（`loading={loading}`，已复核）—— **P1** —— 修法：C1 `tableProps` 内建 `loading`，页面不再手动接。
21. [状态与分页] 4 处手写 `pagination` 对象（基线 1.3）绕开 `usePagination`：无持久化、无统一 `showTotal`、无 `showSizeChanger` 口径，与本片 12 个 hook 接入页形成两套分页行为 —— **P1** —— 修法：迁入 C1；迁移前至少补齐 `showTotal` 与 `showSizeChanger` 保持观感一致。
22. [状态与分页] 分页表上用本地排序（已复核）：`src/pages/monitor/cache/index.tsx:76-132`（7 列 `createSorter` 内存排序）、`src/pages/monitor/job/columns/jobColumns.tsx:37-103` 与 `jobLogColumns.tsx:19-53`（6+5 列）、`src/pages/monitor/server/index.tsx:199-298`（8 列，本地 + 服务端双轨并存）；正确范式对照 `src/pages/monitor/logs/index.tsx:104-119`（`createSorterMeta` 走服务端）—— 分页表点表头只排"当前页"，用户以为全量有序、跨页数据实际乱序，是静默的数据误导 —— **P1** —— 修法：分页表一律 `createSorterMeta` + `useServerSort`（`discoveries/index.tsx:37-54` 为本片标准写法）；仅单页小表（无分页）保留 `createSorter`。
23. [状态与分页] `src/pages/network/backups/index.tsx:79` + `:660-663` —— `usePagination` 已把 current/pageSize 持久化，页面又在 `Table.onChange` 里手动 `setCurrent`/`setPageSize` 同步一遍：双写状态源，一旦 hook 内部行为调整（如 onChangeRef 化），页面侧手动 set 即成幽灵更新 —— **P2** —— 修法：删除页面级手动 set，翻页回调只保留 `handleBackupSortChange` 与数据加载。
24. [状态与分页] 同模块两种分页文案：hook 页显示"共 N 页"（`usePagination.ts:122`），另有 4 处手写分页显示"共 N 条"（基线 1.3）—— 同一用户在相邻两个页面看到两种口径，观感即"做得不认真" —— **P2** —— 修法：随第 18 / 21 条统一，全片仅存"共 N 条"。

### 3.6 错误与空态

25. [错误与空态] 错误只进 console、无用户提示：`src/pages/monitor` 侧 20 处 `console.error` 作为 catch 唯一出口（已复核全部行号）：`server/index.tsx:92`、`:113`，`cache/index.tsx:214`、`:229`、`:251`、`:269`、`:291`、`:308`、`:354`，`logs/index.tsx:190`，`logs/hooks/useLogActions.tsx:63`，`logs/hooks/useLogData.ts:66`、`:100`，`job/hooks/useJobActions.ts:101`、`:115`、`:132`、`:149`，`job/hooks/useJobData.ts:76`、`:98`、`:117` —— 操作失败界面毫无反应 —— **P1** —— 修法：改走 `handleApiError`（本片已有 11 处 `ErrorAlertWithRetry` 先例，说明设施存在、习惯未跟上）；确需保留日志的走 DEV-only logger。
26. [错误与空态] `locale.emptyText` 0/25（基线 1.3）—— 25 个表格空态全部走 antd 默认，无"无匹配数据，试试调整筛选条件 / 新增一条"引导；`EmptyStateWithAction` 仅 8 处 / 2 文件、`ErrorAlertWithRetry` 11 处 / 4 文件（基线），覆盖高度集中 —— **P1** —— 修法：C1 `tableProps` 内建 `locale.emptyText` 指向 `PageState`（第 4 章 C10），筛选空态与无数据空态区分文案。
27. [错误与空态] 错误态"可重试"覆盖不全：`ErrorAlertWithRetry` 已存在但只在 4 个文件使用（基线 1.3），其余页面 `isError` 分支多为渲染一句固定文案或直接吞掉（第 25 条 console 现象的另一面）—— **P2** —— 修法：C10 `PageState` 的 `error` 态强制携带 `onRetry`，数据层用 TanStack Query 的 `refetch` 免费获得。

### 3.7 样式与主题一致性

28. [样式与主题一致性] 6 位 hex 颜色 109 处 / 25 文件（基线 1.3，含 3 位简写共 125 行命中），与 `var(--theme-*)` 混用；已复核代表：`src/pages/network/backups/index.tsx:279-304`（`#ffced0`/`#d4edda`/`#c92a2a`/`#2b7a41` 等 10 处）、`:551`、`:561`、`:790`、`:1010-1104`（9 处），`src/pages/network/command/columns.tsx:158`、`:183`，`src/pages/network/command/index.tsx:166`、`:176` —— 深色模式下备份状态色、卡片背景不随主题反转 —— **P2** —— 修法：收敛到 design token（`--color-success/warning/error`），CI 加 no-hex lint（允许 `var(--x, #fallback)` 过渡）。
29. [样式与主题一致性] 内联 `style` 74 处 / 19 文件（基线 1.3）；已复核代表：`src/pages/network/devices/index.tsx:842`（`style={{ color: "var(--theme-error, #ba3630)" }}`，批量删除按钮靠内联样式标红）—— 布局样式散落标签内，主题调整需逐处抓 —— **P2** —— 修法：颜色 / 间距走 token 与 Tailwind 工具类；`BatchDeleteButton`（C4 附带）内建 danger 样式后该内联消失。
30. [样式与主题一致性] 状态 `0/1` 裸三元渲染 13 处 / 6 文件（基线 1.3）—— `<Tag color={s===0?"green":"red"}>{s===0?"正常":"停用"}</Tag>` 式样各页手写，颜色与文案无单一来源 —— **P2** —— 修法：`StatusTag` + `statusDict`（第 4 章 C8），网络设备 / 凭据 / 任务各注册字典。
31. [样式与主题一致性] 手写日期格式 12 处 / 8 文件（基线 1.3；代表 `src/pages/monitor/logs/utils.tsx:13` 已复核）—— `YYYY-MM-DD`、`YYYY-MM-DD HH:mm:ss`、`toLocaleString` 三种并存 —— **P2** —— 修法：`formatters.ts` 统一（同第 6 条）。

### 3.8 可访问性与国际化

32. [可访问性与国际化] 图标按钮无可访问名：本片 `aria-*` / `role=` / `aria-label` 0 命中（已复核 grep = 0；基线 1.3 同）—— 表格行操作的纯图标按钮（删除 / 详情 / 执行）对读屏用户是无名按钮 —— **P1** —— 修法：`ActionButtons`（`src/components/shared/ActionButtons.tsx`，本片已接入）统一输出 `aria-label={label}`，图标 Button 强制 `title` / `aria-label` 必填，一处修全片。
33. [可访问性与国际化] `useTranslation` 0 / `onKeyDown` 0 / `tabIndex` 0（基线 1.3，已复核 aria 维度 grep = 0）—— 全部文案硬编码中文、行点击交互键盘不可达 —— **P2** —— 修法：国际化暂缓（单语言产品现实），但行点击类交互补 `tabIndex={0}` + `onKeyDown`（Enter 触发）或改显式"详情"按钮；文案集中到常量层为将来 i18n 留口。
34. [可访问性与国际化] 空回调假可点：第 1 / 4 条的 72 处空 `onSearch` 使输入框呈现"可搜"假象（视觉可达、行为不可达），是可访问性视角下最广泛的误导 —— **P2** —— 修法：同第 1 条，`RemoteSelect` / `NumberSelect` 收口后该假象从源头消失。
35. [可访问性与国际化] 弹窗焦点管理缺失：22 个 Modal 无一处处理打开时焦点落入 / 关闭后焦点归还（基线 `aria-*` 0 的伴生事实；`destroyOnHidden` 仅 6/22，基线 1.3）—— 键盘用户关闭弹窗后焦点回到 `<body>`，操作流断裂 —— **P2** —— 修法：antd Modal 自带焦点管理，主要修复点是 `destroyOnHidden` 全量化（C6 内建），并移除手写 `visible` 残留。

### 3.9 性能与副作用

36. [性能与副作用] `src/pages/monitor/cache/index.tsx:413-417` —— `setInterval` 自动刷新（清理函数正确，已复核）但无页面可见性判断：后台标签页仍按周期打监控接口，多开几个监控页即产生持续无效负载 —— **P2** —— 修法：`document.visibilityState === "hidden"` 时暂停（或改用 TanStack Query 的 `refetchIntervalInBackground: false` 默认行为）。
37. [性能与副作用] `src/pages/network/discoveries/hooks/useDiscoveryPolling.ts`（文件级，配套测试 `__tests__/useDiscoveryPolling.test.tsx` 已复核存在）—— 发现任务轮询同样无可见性 / 空闲判断，且与 `cache` 页的 `setInterval` 是两套轮询实现 —— **P2** —— 修法：抽 `usePolling(fn, { interval, pauseWhenHidden })` 统一，二处收敛为一处。
38. [性能与副作用] 第 22 条的本地排序在数据量大时是全量内存排序 + 全量重渲染（`cache/index.tsx:76-132`、`server/index.tsx:199-298` 已复核），且排序结果随翻页丢失，用户反复触发 —— **P2** —— 修法：随第 22 条一并改服务端排序后自动消除。
39. [性能与副作用] 巨型单文件组件未拆分 / 未 memo 化：`src/pages/network/devices/index.tsx`（1243 行）、`src/pages/monitor/cache/index.tsx`（918 行）、`src/components/network/port-write/BulkWriteDrawer.tsx`（532 行）、`src/components/CronSelector/utils.ts`（379 行）（行数见 1.1 基线）—— 单文件内任一 state 变更即整树重渲染，columns / 数据转换未 `useMemo` 的部分每帧重算 —— **P2** —— 修法：columns 与派生数据 `useMemo` 化（devices 已部分采用 `useMemo`，见 `:37-46` 同类写法），巨型文件按"搜索区 / 工具栏 / 表格 / 弹窗"物理拆分（与 C1/C2/C3 落地同步完成）。

### 3.10 代码复用与可维护性

40. [代码复用与可维护性] `catch (error: any)` 8 处复制粘贴（第 16 条已复核 8 个行号）—— 同一"调 API → 成功 message → 失败 handleApiError"样板每页手写 5-8 行 —— **P2** —— 修法：C5 `useMutationAction` 收口，8 处随迁移删除。
41. [代码复用与可维护性] monitor/logs 模块 `any` 簇（已复核）：`src/pages/monitor/logs/types.ts:67`、`logs/utils.tsx:49`、`logs/hooks/useLogActions.tsx:19`、`:22`、`:35`、`logs/hooks/useLogData.ts:31`、`:32`、`:46`、`:49`、`:77`、`:80`，及页面级 `logs/index.tsx:48-70`、`server/index.tsx:146`、`cache/index.tsx:384` —— 类型防线在 logs 模块整体失守（本片 27 处 `any` 的主要聚集地）—— **P2** —— 修法：`OperLog` / `LoginLog` 类型已存在于 `src/types`，补齐 hook 入参 / 出参签名即可，无需新类型。
42. [代码复用与可维护性] 数据获取范式三套并存：`usePagination` 12 文件 / 36 处（基线 1.3；network 侧接入 `ports:36`、`devices:51`、`credentials:32`、`mac:32` 已复核 import）+ `useTableManager` 5 文件 + 手写 4 处（第 21 条）—— 同一"列表页"在本片有三种骨架，新页面无标准可抄 —— **P2** —— 修法：C1 `useListPage` 统一，旧 hook 降级为内部实现。
43. [代码复用与可维护性] 巨型文件债务（第 39 条的维护性切面）：devices 1243 行 / cache 918 行 / BulkWriteDrawer 532 行 / CronSelector utils 379 行（1.1 基线）—— 任一改动 diff 难以 review，测试只能整页级 —— **P2** —— 修法：随 C1-C3 迁移物理拆分；CronSelector/utils.ts 独立性好、有测试，仅补注释无需搬移。
44. [代码复用与可维护性] 空态 / 错误态组件复用不足：`EmptyStateWithAction` 8 处 / 2 文件、`ErrorAlertWithRetry` 11 处 / 4 文件（基线 1.3），其余 20+ 表格自写或裸默认空态（第 26 条）—— 公共组件"存在但少有人用"，与分片 02 结论同构 —— **P2** —— 修法：C10 `PageState` 重设计签名后全片替换，旧组件保留导出一个迭代。
45. [代码复用与可维护性] 状态 / 时间格式化无公共层：裸三元 13 处（第 30 条）+ 手写日期 12 处（第 31 条）+ 手写字节格式化（第 7 条）散落 10+ 文件 —— 后端新增一个状态枚举或调整时间精度，要抓 10+ 个文件 —— **P2** —— 修法：C8 `statusDict` + `formatters.ts` 两个常量 / 工具层收口，枚举与格式只存在一份。

---

## 4. 公共组件抽取候选与必要性评估

估算口径与分片 02 一致：`可减行数 = 每处重复行数 × 复用处数 − 新抽象自身行数`；行数是次要指标，主要收益是**缺陷根因收敛**（第 18 / 19 条证明：一处 hook 口径错误污染 12 页）与**范式统一**。

### 4.1 候选总表

| # | 候选组件 / Hook | 本模块复用点 | 可减代码量估算 | 结论 |
| --- | --- | --- | --- | --- |
| C1 | `useListPage`（合并 `useTableManager` + `usePagination`，TanStack Query 为数据层） | 16 个列表页（12 hook 接入 + 4 手写） | 每页 ~20 行 × 16 ≈ 320 行，hook 增量 ~120 行 → **净减 ~200 行**；同时消灭 19 / 20 / 21 / 23 / 24 五条分页缺陷的再发可能 | **必抽** |
| C2 | `ListToolbar`（导入 / 导出 / 批量删除 / 新增，含 `NetworkExport` 组合） | 26 个列表页的工具栏 | 每页 ~25 行 × 10 个同构页 ≈ 250 行，组件 ~100 行 → **净减 ~150 行** | **必抽** |
| C3 | `FilterBar`（搜索表单 + 回车提交 + allowClear/showSearch 注入） | 约 15 个含搜索区页面 | 每页 ~25 行 × 15 ≈ 375 行，组件 ~140 行 → **净减 ~235 行**；`allowClear`/假搜索两类问题从逐处修 100+ 变为修 1 个渲染器 | **必抽** |
| C4 | `useDeleteConfirm` + `BatchDeleteButton` | 5 处批量删除（P0）+ 15 处单删确认 | 每处 ~8 行 × 20 ≈ 160 行，hook ~70 行 → **净减 ~90 行**；直接消灭 P0 第 8 条 | **必抽** |
| C5 | `useMutationAction`（提交 loading / 错误分流 / 刷新） | ~45 个写操作（22 Modal + 各页 actions hook） | 每处 ~6 行 × 45 ≈ 270 行，hook ~80 行 → **净减 ~190 行**；顺带消除 8 处 `catch (error: any)` | 应统一 |
| C6 | `SafeModal`（升级现有 `BaseEditModal`：`maskClosable=false` / 脏检查 / `confirmLoading` 必填 / `destroyOnHidden`） | 22 个 Modal | 每处 ~5 行 × 22 ≈ 110 行，组件 ~80 行 → **净减 ~30 行**（价值在属性齐全而非行数） | 应统一 |
| C7 | `RemoteSelect`（防抖远程搜索 + `allowClear` + 空 / 加载态） | 48 处业务下拉（P0 第 1 条） | 每处 ~4 行 × 48 ≈ 190 行，组件 ~110 行 → **净减 ~80 行** | 应统一 |
| C8 | `StatusTag` + `statusDict` | 13 处裸三元 + 状态列 6 文件 | 每处 ~3 行 × 19 ≈ 57 行，常量 + 组件 ~50 行 → **基本持平**，收益在枚举唯一来源 | 应统一 |
| C9 | `formatters`（`formatDateTime` / `formatDate` / `formatFileSize`） | 12 处日期 + 3 处字节 | ~30 行 vs 工具 ~50 行 → **净增 ~20 行**，换来全片时间 / 字节口径一致 | 应统一 |
| C10 | `PageState`（重设计现有 `EmptyStateWithAction` / `ErrorAlertWithRetry`） | 25 个表格空态 + 20 处 console 错误出口 | 每处 ~6 行 × 25 ≈ 150 行，组件 ~70 行 → **净减 ~80 行** | 应统一 |
| C11 | `usePolling`（可见性感知轮询） | 2 处（cache setInterval + discoveries 轮询） | ~30 行 | 应统一（小而值） |
| C12 | `NumberSelect`（CronSelector 内部组件） | 25 处空 `onSearch` | 每处 ~1 行 × 25，组件 ~30 行 → **净减 ~0 行**，收益是消假象 | 仅组件内部，**不外提** |
| C13 | 巨型文件物理拆分（devices / cache） | 2 个文件 | 不减行、只搬移 | **不作为"抽取"立项**，随 C1-C3 迁移顺带完成 |

> 合计：消除约 1,600 行重复，扣新增抽象约 900 行，**净减约 700 行**（本片 18,321 行的 ~3.8%）。核心收益同分片 02：`showTotal` 口径、`setTotal` 回填、批量删除确认这类缺陷从"逐页修"收敛为"改一处"。

### 4.2 必抽项论证

**C1 `useListPage`（必抽，对位 P0 第 18 / 19 条）**

- 现状：`usePagination.ts`（140 行）+ `useTableManager.ts` 覆盖了 12 个文件，但 `backups` / `discoveries` 证明接入 ≠ 正确——`setTotal` 暴露给页面却不强制调用（`backups:79`、`discoveries:34` 均解构即弃），`showTotal` 在 hook 内部就实现错（`:122`）。
- 与分片 02 的差异：本片 hook 接入率高（12/16 页），问题出在 **hook 自身 API 允许页面写错**。所以本片必抽的重点不是"再推广 hook"，而是把 `total` 从"页面记得调 setter"改为数据层驱动：`useQuery` 返回什么，`paginationProps.total` 就是什么，`setTotal` 从公开 API 移除。
- 关键设计：`{ data, total, paginationProps, tableProps, search, reset, refresh, selection }`；`tableProps` 内建 `loading` / `scroll` / `locale.emptyText`，`paginationProps` 内建"共 N 条"。

**C2 `ListToolbar`（必抽）**：本片 26 个列表页工具栏结构同构（NetworkExport + 批量按钮 + 新增），`devices:839-847` 这类"批量删除按钮各自手写"正是 P0 第 8 条的温床——按钮声明式化（含 danger 样式、确认文案、权限）后，"无确认的批删按钮"在组件层面写不出来。

**C3 `FilterBar`（必抽）**：72 处空 `onSearch` + 31 处 Input 缺 `allowClear` + 51 处 Select 缺 `allowClear`（基线 1.3），三类覆盖问题的共同根因是"每个控件属性逐处手写"。`FilterBar` 按 items 声明渲染，内部统一注入 `allowClear` / `showSearch` / `optionFilterProp` / `onPressEnter`，机械问题一次性终结。

**C4 `useDeleteConfirm`（必抽，对位 P0 第 8 条）**：5 处批量删除无确认中，`templates` 的问题（确认不在调用链上）说明确认逻辑必须离 API 更近、离按钮更远。hook 返回 `confirm(records)`，内部完成危险确认 + loading + `handleApiError` + 刷新；`BatchDeleteButton`（`components/shared/` 已存在）接同一实现。

### 4.3 应统一项说明

- **C5 `useMutationAction`**：本片 22 个 Modal 无 `confirmLoading`（第 11 条）、20 处 console 错误出口（第 25 条）、8 处 `catch (error: any)`（第 16 条）是同一根因的三个切面：写操作的横切逻辑没有归属。统一后校验错误静默（`isFormValidationError`，`discoveries:26` 已有正确用法）、业务错误透传、未知错误兜底、成功自动刷新。
- **C6 `SafeModal`**：直接改造 `components/modal/BaseEditModal.tsx`（不新建第二个组件，同分片 02 结论）；本片新增强制点：`maskClosable={false}`、关闭脏检查、`confirmLoading` 必填、`destroyOnHidden` 默认开。
- **C7 `RemoteSelect`**：48 处假搜索的机械修复路径。本地选项下拉不需要它——直接 `showSearch + optionFilterProp="label"` 即可（lint 可静态检查）；只有数据源分页 / 远程的下拉（设备、凭据、部门）才接 `RemoteSelect`。
- **C8-C11**：均为小件，随 C1/C3 迁移顺带落地；`usePolling` 是本片特有（监控 / 发现轮询），分片 02 无此件，注意与全局 hooks 命名协调。

### 4.4 不建议抽取的部分

- **不把 CronSelector 的 24 处空 `onSearch` 纳入全局 `RemoteSelect` 治理**：那是纯数字选择器的告警规避写法（≤60 个固定选项），正确修法是组件内部的 `NumberSelect`（C12），外提到 `components/shared` 反而造成误用。
- **不新建"统一监控面板组件"**：`monitor/dashboard`、`monitor/server` 的卡片布局各不相同是业务差异（登录统计 vs 硬件指标），不是习惯差异；强行配置化会把差异推进 `if/else`。
- **不为"批量操作进度"新建组件**：`BulkWriteDrawer`（532 行）已承载该职责，本轮只做错误反馈修复（第 12 条）与拆分（第 43 条），不叠加新抽象。

---

## 5. 最佳实践方案

本片的三个收口点与分片 02 相同（列表页 / 提交弹窗 / 搜索区），此处只写**本片特有**的规范与骨架，通用部分不重复分片 02 第 5 章。

### 5.1 分页铁律（对位 P0 第 18 / 19 条）

1. `total` 只能由数据层驱动：`paginationProps.total = result.data?.total ?? 0`。**`usePagination.setTotal` 从公开 API 移除**（它已被两处证明"暴露即弃用"）。
2. `showTotal` 全站唯一实现，输出"共 N 条（第 X-Y 条）"；出现"共 N 页"即评审拒绝。
3. 分页表禁止本地排序：`sorter` 只允许来自 `createSorterMeta` + `useServerSort`（标准写法：`src/pages/network/discoveries/index.tsx:37-54`）；`createSorter` 仅限无分页的小表，且须注释"当前页内存排序"。

### 5.2 危险操作铁律（对位 P0 第 8 条）

```ts
// 批量删除唯一合法写法：
const { confirm } = useDeleteConfirm({
  onConfirm: (keys) => deviceApi.batchDelete(keys),
  refresh: page.refresh,
});
// 页面工具栏不出现裸 onClick={handleBatchDelete}（devices:843 反例）
```

评审检查项：任何触发删除的按钮，其 onClick 调用链上必须经过 `useDeleteConfirm.confirm`；hook 导出的删除函数不允许绕过确认直接暴露（`useTemplateModals.ts:129` 反例）。

### 5.3 搜索区与轮询规范（本片特有）

- `Select` 三选一：本地枚举 → `showSearch + optionFilterProp="label"`；远程数据 → `RemoteSelect`；纯数字选择器（CronSelector）→ 内部 `NumberSelect` 显式 `showSearch={false}`。禁止出现空 `onSearch`（lint: `no-restricted-syntax` 匹配 `onSearch={() => {}}`）。
- 轮询统一 `usePolling(fn, { interval, pauseWhenHidden: true })`：`cache/index.tsx:413` 的裸 `setInterval` 与 discoveries 轮询收编；组件卸载 / 页面隐藏 / 请求未返回三种情况下不打新请求。

### 5.4 错误出口规范（对位第 25 条）

monitor 侧 20 处 `console.error` 唯一出口是本片最大错误处理债务。规范：组件层 catch 三分支——`isFormValidationError` 静默、业务错误 `handleApiError`、未知错误 `message.error("操作失败")` + DEV-only logger；`console.*` 在 `src/pages` / `src/components` 生产代码中清零（测试文件除外）。

---

## 6. 落地路线图

### 6.1 批次划分

| 批次 | 周期 | 任务 | 验收口径（可脚本验证） |
| --- | --- | --- | --- |
| **P0 止血** | 第 1 周 | ① 修 `usePagination.ts:122` 的 `showTotal`（一行）；② `backups/index.tsx:79`、`discoveries/index.tsx:34` 回填 `setTotal`（各 1-2 行）；③ 5 处批量删除接确认（devices:510 / credentials:188 / ports:322 / mac:226 / useTemplateModals.ts:129）；④ 48 处业务下拉假搜索批量修复（本地枚举加 `showSearch + optionFilterProp`，远程的先加 `showSearch` 止血、`RemoteSelect` 二期跟进） | ① `rg "共 \$\{Math.max" src/hooks` = 0 且 `rg "共 N 条" 口径` 覆盖 12 个 hook 接入页；② backups / discoveries 翻页可见第 2 页数据（手测）；③ `rg "onClick=\{handleBatchDelete\}" src/pages/network` = 0；④ `rg "onSearch=\{\(\) => \{\}\}" src/pages/network src/pages/monitor` = 0 |
| **P1** | 第 2-4 周 | ① 落地 C1 `useListPage` 并迁移 4 个手写分页页 + 2 个 total 缺陷页（C7 `RemoteSelect` 一并替换远程下拉）；② C4 `useDeleteConfirm` 统一 15 处单删 + 5 处批删；③ C5 `useMutationAction`：22 个 Modal 补 `confirmLoading`、消除 8 处 `catch (error: any)` 与 20 处 console 错误出口；④ C6 `SafeModal`：`maskClosable={false}` + 脏检查 + `destroyOnHidden` 全量化；⑤ `cache` / `job` / `server` 本地排序改服务端；⑥ monitor/logs `any` 簇补类型 | ① 25 个 Table `loading` 覆盖 25/25、`scroll` ≥6 列表 100%；② `rg "catch \(error: any\)" src/pages/(network|monitor)` = 0；③ `rg "console\.error" src/pages/monitor` = 0；④ `confirmLoading` 覆盖 22/22；⑤ 分页表 `createSorter` = 0（仅小表豁免并注释） |
| **P2** | 第 5-7 周 | ① hex 109 处 / 内联 style 74 处收敛 design token（页面级 PR）；② C8 `statusDict` + C9 `formatters` 替换裸三元 13 处、手写日期 12 处、字节格式化；③ C10 `PageState` 接管 25 个表格空态；④ `usePolling` 收编 2 处轮询；⑤ 巨型文件拆分（devices 1243 / cache 918）随 C1-C3 迁移完成；⑥ `ActionButtons` 补 `aria-label`，行点击补键盘可达 | ① 6 位 hex 在本片 < 30 处（当前 109）；② `rg "toLocaleString" src/pages/(network|monitor)` = 0；③ `locale.emptyText` 覆盖 25/25；④ 轮询裸 `setInterval` = 0；⑤ aria 覆盖行操作按钮 100% |

### 6.2 迁移顺序与回滚

1. **先修根因、零风险先行**：`usePagination.ts:122` 与两处 `setTotal` 回填合计 ~5 行改动、无结构变更，第 1 周内独立 PR 立即发布——它们缓解的是用户已投诉的症状。
2. **批量删除确认单独成 PR**：不依赖任何新抽象，5 处可先临时各加 `Modal.confirm` 止血，C4 落地后再统一替换——事故级风险不等框架。
3. **C1 试点页**选 `discoveries`（386 行、最小列表页）验证 happy path，再迁 `backups`（1118 行、有双写状态要清理），最后 `devices` / `cache`（巨型页随拆分迁移）。每页一个 PR，旧实现保留一个迭代。
4. **回滚**：C1 / C4 / C6 均为页面级一行切回（保留 `usePagination` / `useTableManager` / `BaseEditModal` 导出）；假搜索修复可按文件粒度 revert。

### 6.3 度量

- 每迭代末复跑 1.2 节扫描脚本，跟踪五个数字的单调收敛：`showTotal` 正确率（0/12 → 12/12）、批量删除确认覆盖（0/5 → 5/5）、空 `onSearch`（72 → 0）、`confirmLoading`（5/22 → 22/22）、console 错误出口（52 → 0）。
- 不追求的指标：抽取的组件数、净减行数（~700 行只是副产品）。追求的指标：新列表页样板行数（目标 ~450 → ~120），以及"同类缺陷修复点数量"（`showTotal` 类缺陷从 12 处收敛到 1 处）。

---

> **核对记录**：本片问题总数 45 条，与第 2.1 节维度行逐格一致（3.1:4 / 3.2:3 / 3.3:6 / 3.4:4 / 3.5:7 / 3.6:3 / 3.7:4 / 3.8:4 / 3.9:4 / 3.10:6），严重度合计 **P0 4 / P1 16 / P2 25**（第 2.1 节合计行原印"15 / 26"系笔误，已由 Lead 校正为维度行之和 16 / 25）。P0 行号已逐一回源码复核：`src/hooks/usePagination.ts:122`、`src/pages/network/backups/index.tsx:79` / `:659`、`src/pages/network/discoveries/index.tsx:34` / `:354` / `:66`、`src/pages/network/devices/index.tsx:510` / `:839-847`、`src/pages/network/credentials/index.tsx:188` / `:462`、`src/pages/network/ports/index.tsx:322` / `:697`、`src/pages/network/mac/index.tsx:226` / `:492`、`src/pages/network/templates/index.tsx:320`、`src/pages/network/templates/hooks/useTemplateModals.ts:129`、`src/pages/network/devices/index.tsx:243`（单删确认对照）、假搜索代表行 `devices:752/767/788/937/1047/1062/1095/1110` 等（grep 全量复核）。

