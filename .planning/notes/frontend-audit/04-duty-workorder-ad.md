# 值班 / 工单 / 知识库 / AD 域 / 仪表盘 / CAD / 验证码 前端组件审查报告

> 审查人：audit-misc（Team Lead 分片 04）
> 审查对象：ruoyi-react-frontend/src 下 pages/duty、pages/workorder、pages/knowledge、pages/dashboard-system、pages/ad-domain、pages/ad、components/dashboard、components/cad-editor、components/cad-elements、components/captcha
> 审查方式：只读静态审查（逐文件精读 + 正则块级统计 + 22 个 Table 页面能力矩阵），未运行浏览器
> 结论一句话：本片 22 个列表页有 10 个用公共 usePagination，却因 usePagination.ts:120 只显示"共 N 页"，用户看不到总条数；另有 10 个页面手写分页（其中 5 个正确显示总条数），公共骨架未下沉，导致同类页面能力参差。

## 1. 审查范围与方法

### 1.1 文件规模（本片非测试文件 125 个 / 26,557 行）

| 目录 | 文件数 | 行数 | 最大单文件 | 说明 |
| --- | --- | --- | --- | --- |
| pages/duty | 66 | 8,767 | schedules/index.tsx 792 | 排班主列表 + 值班池 + 我的值班 + 管理子组件 |
| pages/workorder | 23 | 4,130 | orders/index.tsx 1009 | 本片最大列表页 |
| pages/knowledge | 14 | 1,927 | view/index.tsx 373 | 知识库浏览/分类树 |
| pages/dashboard-system | 15 | 1,526 | components/DashboardList.tsx 349 | 仪表盘 CRUD |
| pages/ad-domain | 24 | 6,022 | users/index.tsx 758 | AD 用户/OU/计算机/组/配置/日志 |
| pages/ad | 1 | 181 | SyncMonitor/index.tsx | AD 同步监控 |
| components/dashboard | 59 | 8,443 | settings/DisplayConfigForm.tsx 419 | 仪表盘设计器与 12 类 widget |
| components/cad-editor | 11 | 3,049 | CADFloorPlanEditor.tsx 1537 | 本片最大单文件（20 个 useState） |
| components/cad-elements | 6 | 1,116 | WorkstationElement.tsx 549 | CAD 图元 |
| components/captcha | 7 | 840 | SliderCaptcha.tsx 267 | 滑块/文本验证码 |

### 1.2 精读取证范围

逐行精读 27 个文件（本片行数前 15 + 全部含 Modal/Table/分页的页面），另对全部 125 个文件做 12 类模式的正则块级统计（分页、showTotal、Input/Select 属性、rules、Modal 属性、message 通道、hex 色值、状态三元、日期格式串、空态/错误态、useEffect 依赖、死代码引用）。22 个 Table 页面逐项打了能力矩阵（行选择/批量/空态/错误态/列配置/序号/showTotal/回车搜索/二次确认/排序/固定列）。

### 1.3 方法与局限

- 只读审查，未修改任何源码；未启动 dev server，未做运行时验证，故"用户可见影响"均由代码路径推导（如"翻页后请求参数与展示不一致"来自两处 pageSize 取值不同）。
- 行号为审查时快照行号（文件在审查期间未被修改）。
- 公共能力基线：hooks/usePagination.ts、hooks/useTableManager.ts（411 行）、hooks/useColumnConfig.ts、components/shared/*（ActionButtons、EmptyStateWithAction、ErrorAlertWithRetry 等）、components/modal/BaseEditModal.tsx。审查重点之一就是这些能力在本片的实际使用率。

## 2. 问题总览

### 2.1 数量与严重度

| 级别 | 条数 | 含义 |
| --- | --- | --- |
| P0 | 4 | 数据不可达/翻页错乱/重复提交，用户直接踩到 |
| P1 | 22 | 功能缺失或体验明显受损，模块内普遍存在 |
| P2 | 20 | 一致性/可维护性/规范问题，不立即影响功能 |
| 合计 | 46 | 全部精确到 文件:行号 |

### 2.2 按维度分布

| 维度 | 条数 | 其中 P0 | 本片最突出问题 |
| --- | --- | --- | --- |
| 功能完整性 | 7 | 1 | 搜索不可用（只有空 onSearch，无 showSearch）、列配置 0/22 |
| 数据展示完整性 | 5 | 1 | 总条数不显示、序号列只有 5/22、分页持久化不一致 |
| 交互反馈 | 6 | 1 | 弹窗提交无 confirmLoading、二次确认三套并存、搜索无回车 |
| 表单校验 | 4 | 0 | pattern 校验 0 处、编辑弹窗无 rules |
| 状态与分页 | 5 | 1 | 手写分页 10 文件、分页参数硬覆盖 |
| 错误与空态 | 5 | 0 | 自定义空态 2/22、错误态 4/22、空 catch |
| 样式与主题一致性 | 4 | 0 | 硬编码 hex 129 处、危险色用 style 而非 danger |
| 可访问性与国际化 | 2 | 0 | 图标按钮无 aria-label、中文格式串硬编码 |
| 性能与副作用 | 3 | 0 | CAD 单文件 1537 行/20 useState、useEffect 关掉 exhaustive-deps |
| 代码复用与可维护性 | 5 | 0 | 死代码 551 行、公共骨架本片 0 使用、巨型文件 5 个 |

### 2.3 Top3 最严重问题

1. **分页永远看不到总条数（P0-01）**：hooks/usePagination.ts:120 的 showTotal 是"共 N 页"，本片 10 个使用该 hook 的列表页全部无法回答"我一共多少条数据"。用户原话抱怨的就是这一点。而 5 个手写分页的页面（ScheduleManagement.tsx:294、DashboardList.tsx:327、groups/index.tsx:548、logs/index.tsx:206、ReconciliationTab.tsx:306）却都正确写成"共 N 条"——公共 hook 反而比手写弱，这是典型的"公共组件不完善导致各写各的"。
2. **AD 用户列表超过 100 条后无法访问（P0-02）**：pages/ad-domain/ous/index.tsx:130-131 固定 pageSize:100 一次拉取，:539 的表格却用 pagination={{ pageSize: 50, size: "small" }} 在前端分页且没有 total/onChange。第 3 页起数据不存在于前端，用户看不到，且没有任何提示。
3. **弹窗提交无 loading，双击产生重复数据（P0-03）**：本片 27 个 Modal 中 confirmLoading 出现 0 次，okButtonProps 的 13 处只用于 danger/disabled。pools/index.tsx:471、users/index.tsx:647、orders/index.tsx:706、periodic/templates/index.tsx:422、categories/index.tsx:257、configs/index.tsx:376、AccountPoolTab.tsx:418 均可在请求返回前重复点击"确定"。值班池、排班、工单都是会重复写入主数据的场景。

## 3. 十维度问题清单

> 格式：编号 | 维度 | 文件:行号 —— 现象 —— 级别 —— 修法。行号均为审查快照行号。

### 3.1 [功能完整性]

**P0-02 | [功能完整性] pages/ad-domain/ous/index.tsx:130-131 + :539 —— 用户列表固定取 pageSize:100 且表格用前端分页（无 total/onChange），第 3 页起数据不可达 —— P0 —— 改为服务端分页：请求带 current/pageSize，表格接 paginationProps，total 用后端 total。**

**P1-01 | [功能完整性] 全片 112 处 Select 无 showSearch（如 duty/schedules/index.tsx:475/490/505/520、ScheduleManagement.tsx:191/200/209/216、workorder/orders/index.tsx:577/592/607/622、knowledge/articles/index.tsx:243/258/273）—— 值班池/人员/分类/状态等下拉在选项多时无法输入过滤 —— P1 —— 统一改用可搜索选择器（SearchSelect，见 4.4），或在原处补 showSearch + optionFilterProp="label"。**

**P1-02 | [功能完整性] 全片约 91 处 onSearch={() => {}}（如 schedules/index.tsx:480/495/510/525/607/630、users/index.tsx:551/588、orders/index.tsx:582/597/612/629）—— 该写法只为压掉 antd "onSearch should be used with showSearch" 警告，没有 showSearch 时用户仍然不能搜索，属于掩盖而非修复 —— P1 —— 删除空 onSearch，补齐 showSearch；只在真正远程搜索时保留 onSearch。**

**P1-03 | [功能完整性] 22 个 Table 页面列配置使用 0/22（全片 0 处 useColumnConfig）—— 工单（orders 1009 行、12 列+）、AD 用户（users 758 行、scroll x1900）列多且无法自定义显示 —— P1 —— 表格容器内置 ColumnConfigModal 入口，列配置持久化到 sessionStorage/用户偏好。**

**P1-04 | [功能完整性] my-duty/index.tsx:402-408 —— 下拉 onChange 直接触发 handleSearch 请求，无防抖，逐字/多次切换会连发请求 —— P1 —— 由 useTableManager 的 filterEmpty + 防抖统一处理，或加 300ms debounce。**

**P1-05 | [功能完整性] cad-editor/CADFloorPlanEditor.tsx:1223 —— 删除元素直接执行，无二次确认（同文件 :1196 才提示"已删除 N 个元素"），而其他页面删除普遍有确认 —— P1（危险操作误删）—— 走统一 ConfirmAction，危险操作必须有确认。**

**P2-01 | [功能完整性] workorder/statistics/index.tsx:274/287/303 —— 三个统计 Table 均为 pagination={false}，数据超一屏时只能靠页面滚动，无法分页 —— P2 —— 统计表加 scroll y 或分页。**

### 3.2 [数据展示完整性]

**P0-01 | [数据展示完整性] hooks/usePagination.ts:120 —— showTotal 只返回"共 N 页"，不返回总条数；本片 10 个使用该 hook 的页面（duty/schedules、pools、my-duty、workorder/orders、periodic/templates、knowledge/articles、ad-domain/computers、groups、users、HolidayManagement）全部看不到总条数 —— P0 —— 改为"共 N 条"，并补 showTotalFormatter 参数兼容页数展示；同时统一 showSizeChanger 文案。**

**P2-02 | [数据展示完整性] 序号列只有 5/22 个页面有（且都是手写 index+1：pools/index.tsx:245、TableWidget.tsx:98 等），orders/index.tsx:690-702、schedules/index.tsx:578-590、users/index.tsx:625-643 等大表均无序号 —— P2 —— 表格容器提供 rowIndex 渲染函数（服务端分页时按 (current-1)*pageSize 计算）。**

**P2-03 | [数据展示完整性] 分页持久化不一致：usePagination 会把 current/pageSize 写入 sessionStorage（按 pathname 隔离），而 10 个手写分页页面（ScheduleManagement.tsx:288、DashboardList.tsx:321、AccountPoolTab.tsx:405、ReconciliationTab.tsx:300、groups/index.tsx:544、logs/index.tsx:201、ous/index.tsx:539 等）切页后刷新即回第 1 页 —— P2 —— 全部改用 usePagination，或将持久化下沉到 DataTable。**

**P2-04 | [数据展示完整性] ad-domain/groups/index.tsx:544-551 与 :602 —— 同一页面出现两套分页（成员表手写 total=memberTotal + showTotal，第二处只有 pageSize:10），成员总数与组列表分页语义混用 —— P2 —— 拆分为两个独立的表格容器实例，各自持有分页状态。**

**P2-05 | [数据展示完整性] 8 个页面表格无 scroll.x 却列数 ≥8（如 knowledge/articles/index.tsx:305、categories/index.tsx、holidays/index.tsx:123 有 x1000 但 operations 列固定缺失）—— 窄屏下列挤压、无横向滚动 —— P2 —— 统一由 DataTable 按列宽计算 scroll.x。**

### 3.3 [交互反馈]

**P0-03 | [交互反馈] 27 个 Modal 中 confirmLoading 出现 0 次（pools/index.tsx:471-479、users/index.tsx:647-652、orders/index.tsx:706-712、periodic/templates/index.tsx:422、categories/index.tsx:257-263、configs/index.tsx:376-383、AccountPoolTab.tsx:418-441）—— 提交期间确定按钮不 loading、不 disabled，双击即两次写请求 —— P0 —— 抽 useSubmit（见 5.2），Modal 统一接 confirmLoading。**

**P1-06 | [交互反馈] 二次确认三套并存：Modal.confirm 13 文件 16 处（schedules/index.tsx:211/560、pools/index.tsx:348、categories/index.tsx:238、configs/index.tsx:319、users/index.tsx:292 等）、Popconfirm 3 文件 9 处（orders/index.tsx:202、categories?、AccountPoolTab.tsx:343、DashboardList.tsx:257）、CAD 直接删除（CADFloorPlanEditor.tsx:1223）—— 交互不一致；且 Modal.confirm 的 onOk 返回 Promise 时才 loading，本片多处 onOk 未返回 Promise —— P1 —— 统一为 ConfirmAction / ActionButtons.confirm，危险操作强制二次确认。**

**P1-07 | [交互反馈] 搜索无回车触发：22 个页面只有 3 处支持回车（DashboardList.tsx:304 onPressEnter、ReconciliationTab.tsx:262 onPressEnter、knowledge/view/index.tsx:183-198 antd Search enterButton），schedules/pools/orders/templates/articles 等大表都必须鼠标点"查询" —— P1 —— SearchForm 统一监听 Enter 提交 + 重置按钮，搜索区可折叠。**

**P1-08 | [交互反馈] 静态 import { message } from "antd" 4 处（ad-domain/configs/AccountPoolTab.tsx:2-16、ad-domain/configs/index.tsx:2-14、ad-domain/groups/index.tsx:2-16、ad-domain/users/index.tsx:2-15 的 import 块）—— antd 6 下静态 message 脱离 ConfigProvider 上下文，主题色/语言/层级不跟随，控制台有 warning —— P1 —— 统一 App.useApp() 或项目已有的 @/utils/antdMessage（AntdThemeBridge.tsx:49-56 已注入 context 实例）。**

**P2-06 | [交互反馈] duty/schedules/index.tsx:230 —— const { message: _message } = App.useApp() 但全文无 message 调用（成功提示在 hooks 里自行处理），订阅即死代码 —— P2 —— 删除该行；或确认删除/生成类操作成功后确实有统一提示。**

**P1-09 | [交互反馈] 长任务无进度反馈：CADFloorPlanEditor.tsx:433-437 handleSave 直接 message.success("保存成功")，无保存中状态与防连点 —— P1 —— 抽 useSubmit 后接 loading + 防重。**

### 3.4 [表单校验]

**P1-10 | [表单校验] ad-domain/users/index.tsx:647-681 —— 编辑用户 Modal 无 rules、无 maxLength、8 个 Input 全无 allowClear 与 placeholder（:656/659/662/665/668/672/675）—— 可提交空显示名、非法邮箱、超长字段，后端报错后才失败 —— P1 —— 用 FormModal + 校验规则模板（email/phone/maxLength），输入框补 allowClear。**

**P1-11 | [表单校验] 全片 pattern 校验 0 处 —— 邮箱/手机/域控地址/host 无格式校验：users/index.tsx:656-678、ad-domain/configs/index.tsx:400（"如 ad.example.com 或 192.168.1.100"仅 placeholder）、:424（DC=example,DC=com）—— 非法值直接入库，AD 同步时才失败 —— P1 —— 抽校验规则常量（email、ip、hostname、dn），Form.Item 复用。**

**P2-07 | [表单校验] TextArea 无 maxLength：schedules/index.tsx:706/783、categories/index.tsx:302、AccountPoolTab.tsx:438（这个是正例 maxLength 500）—— 备注/原因字段可无限输入 —— P2 —— 统一 maxLength + showCount。**

**P1-12 | [表单校验] Input 无 allowClear 45 处 / 20 文件（users/index.tsx 7 处、DisplayConfigForm.tsx 12 处、configs/index.tsx 4 处等）—— 输入错误后只能全选删除，用户明确抱怨的"文本输入框没有一键清除" —— P1 —— 见 4.5：优先在公共封装层默认 allowClear，避免逐处补。**

### 3.5 [状态与分页]

**P0-04 | [状态与分页] duty/my-duty/index.tsx:417-431 —— 表格 pagination 展开 paginationProps 后又硬覆盖 pageSize:10、showSizeChanger:false，而请求参数用的是 paginationProps.pageSize —— P0 —— 删掉覆盖，pageSize 只保留一个来源；否则用户切到 20/50 条后表格仍按 10 条显示，翻页跳数据。**

**P1-13 | [状态与分页] pages/ad-domain/users/index.tsx:170-178 —— handleTableChange(pagination, _filters, sorter) 完全不用 pagination 参数，翻页依赖 paginationProps 内部 onChange、排序走 sorter 分支，双通道并存 —— P1 —— 分页与排序统一在 handleTableChange 中处理（参考 useTableManager.handleTableChange）。**

**P1-14 | [状态与分页] 手写分页 10 文件（ScheduleManagement.tsx:288、my-duty:417、DashboardList.tsx:321、computers:576、AccountPoolTab.tsx:405、ReconciliationTab.tsx:300、groups:544、logs:201、ous:539、ous/index_with_dept:434）—— 与 usePagination 风格分叉：有的显示总条数、有的不显示；有的有 pageSizeOptions、有的没有；其中 4 个文件与 usePagination 混用 —— P1 —— 统一到 DataTable/usePagination，禁止页面直接写 pagination={{...}}。**

**P2-08 | [状态与分页] 状态 0/1 裸三元 7 文件 17 处（computers/index.tsx:324/325/327/452/453/455、AccountPoolTab.tsx:208/212/338/341、configs/index.tsx:265/495、categories/index.tsx:75/76、DashboardList.tsx:213、pools/index.tsx:307、WorkstationElement.tsx:495）—— 文案与颜色分散，新增状态需改多处且易出现"启用/正常/在线"不一致 —— P2 —— 抽 StateTag / DictTag（见 4.6）。**

**P2-09 | [状态与分页] 手写日期格式 9 文件 20 处（useHolidayModals.ts:101/195、holidays/utils.tsx:82、GenerateScheduleModal.tsx:33/34、management/index.tsx:122/160、WeeklyView.tsx:17/63/65、my-duty/index.tsx:121/122、useScheduleData.ts:82/83、useScheduleModals.ts:77/78/119、schedules/index.tsx:388/390、schedules/utils.tsx:24）—— 与公共 formatDate/formatDateTime（utils/datetime.ts:10/23）并存，"YYYY-MM-DD"字面量散落 —— P2 —— 提交参数统一走 formatDate，展示统一走 formatDateTime。**

### 3.6 [错误与空态]

**P1-15 | [错误与空态] 自定义空态 2/22：orders/index.tsx:690-702、periodic/templates/index.tsx:410-416、schedules/index.tsx:578-590、users/index.tsx:625-643、groups、logs、configs、statistics 等均无 locale.emptyText —— 筛选无结果时只有 antd 默认图形，无"去新建/清空筛选"引导 —— P1 —— 统一接入 EmptyStateWithAction（components/shared/EmptyStateWithAction.tsx），按 hasFilter 区分"暂无数据"与"无匹配结果"。**

**P1-16 | [错误与空态] 错误态 4/22：列表请求失败只 toast 一次，页面区域保持空/旧数据，无重试入口（ReconciliationTab.tsx:281-299 是唯一接入 ErrorAlertWithRetry 的页面）—— P1 —— DataTable 内置 error/onRetry，复用 ErrorAlertWithRetry。**

**P1-17 | [错误与空态] components/dashboard/utils/dataFetcher.ts:152 —— 空 catch {}，请求失败被吞，widget 永远停在 loading/空白且无提示 —— P1 —— 至少 console.error + 返回 error 态给 BaseWidget（BaseWidget.tsx:213 已有 error Result 分支，只是没人喂）。**

**P2-10 | [错误与空态] components/dashboard/widgets/base/BaseWidget.tsx:229-231 —— 三态齐全（Skeleton :206、Result :213、Empty :229、Spin :236）但空态用裸 Empty、错误态用裸 Result，未复用公共 EmptyStateWithAction/ErrorAlertWithRetry —— P2 —— 复用公共组件，保持文案/图标统一。**

**P2-11 | [错误与空态] captcha/SliderCaptcha.tsx:141/185 —— 空 catch 后仅提示"验证码加载失败"，无重试按钮；:52-53 与 TextCaptcha.tsx:32 各自写 message.error("加载验证码失败") —— P2 —— 抽出 useCaptcha 统一错误处理 + 重试。**

### 3.7 [样式与主题一致性]

**P1-18 | [样式与主题一致性] 硬编码 hex 129 处（含 var(--theme-*) 回退）—— 重点行：schedules/index.tsx:209/397/398/412/419/443/558、pools/index.tsx:346/375/382、my-duty/index.tsx:300/315/327/347、WeeklyView.tsx:73/74/84/93/121、orders/index.tsx:656（同时出现 var 与裸 #ba3630）、ous/index.tsx:511/570/576/586/624/675、AccountPoolTab.tsx:366/375、logs/index.tsx:168、CADFloorPlanEditor（#337ab0/#333333）、SyncMonitor/index.tsx:130/139 —— 暗色主题下 #999/#ccc/#666 对比度不足，品牌绿更换后这些颜色不跟随 —— P1 —— 全部改 design-system/tokens/colors.ts 或 var(--theme-*)，禁止业务页写字面量颜色。**

**P1-19 | [样式与主题一致性] 危险色用内联 style 而非语义属性：schedules/index.tsx:558（批量删除 style color #ba3630）、pools/index.tsx:346、orders/index.tsx:656（color 与 borderColor 双写 #ba3630）、my-duty 等 —— 按钮 hover/disabled 态不自洽，主题切换不跟随 —— P1 —— 用 Button danger 或 <ActionButtons danger>；批量删除复用 BatchDeleteButton。**

**P2-12 | [样式与主题一致性] 间距/宽度魔法数：内联 style width 约 101 处、marginBottom: 16 约 22 处（如 ScheduleManagement.tsx:191 "width: 140"、orders 搜索区、DisplayConfigForm 多处 width: 120/80）—— 响应式断裂、密度切换无效 —— P2 —— 用 design-system 间距 token / Flex gap。**

**P2-13 | [样式与主题一致性] Modal 宽度随意（20 文件 27 处 width，520/600/700 混用，users/index.tsx:647 width=600、pools/index.tsx:471 width=600）—— P2 —— FormModal 提供 sm/md/lg 三档。**

### 3.8 [可访问性与国际化]

**P2-14 | [可访问性与国际化] 图标按钮无 aria-label：cad-editor/CADToolbar.tsx（纯图标工具按钮）、CADFloorPlanEditor.tsx:1517-1525（Modal + TextArea 无 label）、TableWidget 操作列等 —— 读屏软件读不出按钮含义 —— P2 —— icon-only Button 必带 aria-label/title。**

**P2-15 | [可访问性与国际化] 中文格式串硬编码在 format 里：WeeklyView.tsx:17 "YYYY年MM月DD日"、schedules/utils.tsx:24 同款、my-duty 等 —— 无法国际化 —— P2 —— 文案走 i18n 或统一 formatToken 常量（本仓库 locale 已是 zhCN，见 AntdThemeBridge.tsx:178，问题只在这些字面量）。**

### 3.9 [性能与副作用]

**P1-20 | [性能与副作用] cad-editor/CADFloorPlanEditor.tsx:65-152 —— 单文件 1537 行、约 20 个 useState（mode/selectedId/selectedIds/hoveredId/scale/offset/history/historyIndex/isTextInputVisible/…/isDraggingPlanImage）—— 任一状态变更整页重渲染，无法单测，日志显示 message 散落在 :215/435/513/696/753/1196/1223 —— P1 —— 拆分为 useEditorState（useReducer）+ 子组件（Canvas/Toolbar/PropertyPanel），先抽 state 再拆渲染。**

**P2-16 | [性能与副作用] useEffect 关闭 exhaustive-deps：schedules/index.tsx:302-310（初始化一次性拉 4 个接口 + fetchWeeklyDuty）、ous/index.tsx:79-99（fetchOUTree）—— 依赖闭包旧值风险，新增依赖不会被感知 —— P2 —— 改用 useQuery/useEffectEvent 或把初始化拆成独立数据 hook。**

**P2-17 | [性能与副作用] TanStack Query 在本片仅 1 文件 7 次使用（pools/index.tsx 的成员候选 useQuery），其余全部手写 loading + useState（本片 useState 284 处）—— 缓存/去重/重试能力闲置 —— P2 —— 列表与候选数据逐步迁到 useQuery，或至少统一 loading 语义。**

### 3.10 [代码复用与可维护性]

**P1-21 | [代码复用与可维护性] pages/ad-domain/ous/index_with_dept.tsx（551 行）—— 全项目仅被 pages/__tests__/batch51.zeroFiles.render.test.tsx 引用，生产路由未挂载，与 ous/index.tsx 几乎同构（:389 空 onSearch、:377/432/460/496 loading）—— 修一处漏一处 —— P1 —— 删除文件并同步清理测试引用。**

**P1-22 | [代码复用与可维护性] 公共骨架在本片使用率：useTableManager 0、useColumnConfig 0、BaseEditModal 0、opsApi 0、TableQuery 0 —— 22 个列表页各写 handleTableChange/handleSearch/handleReset（如 schedules:312-334、orders:293-309、articles:134）—— P1 —— 先选 3 页试点接入（见 6.2），验证后再全量。**

**P2-18 | [代码复用与可维护性] 重复颜色/状态映射：get*Color 函数 9 文件 11 处（schedules/index.tsx:337-348 getDutyTypeBadgeColor 用 switch 手写）、Tag color 三元 14 文件 20 处 —— 同一业务状态在不同页面颜色/文案不同 —— P2 —— 收敛为枚举 + 映射常量（一状态一处定义）。**

**P2-19 | [代码复用与可维护性] 巨型文件：orders/index.tsx 1009、schedules/index.tsx 792、users/index.tsx 758、ous/index.tsx 699、CADFloorPlanEditor.tsx 1537 —— 单文件承载搜索/表格/3 个 Modal/表单，改动风险高 —— P2 —— 按"页面 = 容器 + 列表 + 弹窗"拆分，先拆弹窗。**

**P2-20 | [代码复用与可维护性] destroyOnClose（旧 API）与 destroyOnHidden 混用 3 处（computers/index.tsx、configs/AccountPoolTab.tsx 的 Modal 等，本片 destroyOnHidden 18 处）—— antd 6 下旧属性行为不一致 —— P2 —— 统一 destroyOnHidden（或由 FormModal 固定）。**

## 4. 公共组件抽取候选与必要性评估

> 估算方法：每处/每页可减少的行数按"手写等价代码行数 - 调用行数"计，再乘以本片出现次数；"本模块复用次数"按本片实际出现次数统计，不含本片以外。

| # | 候选组件 | 解决的问题 | 本模块复用次数 | 涉及文件数 | 可减代码量估算 | 结论 |
| --- | --- | --- | --- | --- | --- | --- |
| 4.1 | DataTable（统一表格容器：分页/序号/空态/错误态/列配置/刷新/批量/scroll） | 22 页各写 Table + pagination + 空/错态；能力 2/22～5/22 参差 | 22 页 | 22 | 每页约 30-40 行 × 22 页 ≈ 700-880 行 | 必抽 |
| 4.2 | SearchForm（查询区：Enter 提交/折叠/重置/间距/防抖） | 回车 3/22；搜索区手写 15-40 行 | 22 页 | 22 | 每页约 25 行 × 22 ≈ 550 行 | 必抽 |
| 4.3 | FormModal（BaseEditModal 增强：confirmLoading 自动/滚动重置/宽度档/校验模板） | 27 Modal 的 confirmLoading=0；destroyOnHidden 混用；宽度随意 | 27 处 | 20 | 每处约 12 行 × 27 ≈ 320 行，另消除重复提交缺陷 | 必抽（扩展现有 BaseEditModal） |
| 4.4 | SearchSelect（可搜索选择器：showSearch + allowClear + 远程/本地过滤） | 112 处无 showSearch、93 处无 allowClear、91 处空 onSearch | 112 处 | 12+ | 每处 1-2 行 × 112 ≈ 150-220 行，能力统一 | 必抽 |
| 4.5 | InputWithClear / 全局 Input 默认 allowClear | 45 处 Input 无 allowClear（用户原话"没有一键清除"） | 45 处 | 20 | 无需逐处改；在封装层统一 | 必抽（封装层默认，不改业务页） |
| 4.6 | StateTag / DictTag（0/1 与业务状态 → 文案+颜色） | 17 处裸三元、11 处 get*Color、20 处 Tag 三元 | 17 处 | 7 | 每处 2-3 行 × 17 ≈ 50 行，且消除文案不一致 | 应统一 |
| 4.7 | ConfirmAction / ActionButtons.confirm | Modal.confirm 16 处 + Popconfirm 9 处 + CAD 无确认 | 25 处 | 16 | 每处约 8 行 × 25 ≈ 200 行 | 必抽（扩展 ActionButtons） |
| 4.8 | useSubmit / useAsyncAction（提交态 + 防重 + 成功/失败提示） | 27 Modal 无 confirmLoading；28 文件 88 次 message.success 手写 | 40+ 处 | 25 | 每处约 6 行 × 40 ≈ 240 行 | 必抽 |
| 4.9 | TableEmpty / ErrorState 接入（已有组件，只差接入） | 空态 2/22、错误态 4/22 | 22 页 | 22 | 每页约 8-10 行 × 22 ≈ 200 行 | 应统一（现有组件零使用，属接入问题而非新造） |
| 4.10 | usePagination 的 showTotal 修复（非新组件） | 10 页看不到总条数（用户原话） | 10 页 | 10 | 改 1 行，收益 10 页 | 必抽（1 行修复，最高性价比） |
| 4.11 | 列配置（useColumnConfig 已有，0 使用） | 列配置 0/22 | 22 页 | 22 | 每页约 10 行接入 × 22 ≈ 220 行 | 应统一（推广已有能力） |
| 4.12 | formatter 常量（DATETIME/DATE） | 20 处手写 .format("YYYY-MM-DD") | 20 处 | 9 | 每处 1 行，收益为一致性 | 应统一 |
| 4.13 | 主题色 token（已有 --theme-* 与 tokens/colors.ts） | 129 处 hex | 129 处 | 20 | 不抽组件；改为引用 token | 不建议抽组件（应改用法） |
| 4.14 | 状态枚举常量（DutyType/WorkOrderStatus/ADStatus） | 状态映射分散 | 26 处 | 10 | 少量行数，主要收益是一致性 | 应统一 |

## 5. 最佳实践方案

> 只给本片最需要的 3 个骨架：分页修复（1 行）、提交态 hook、可搜索选择器。组件完整实现不在本报告范围。

### 5.1 分页修复（最高优先级，改 1 处收益 10 页）

    // hooks/usePagination.ts 约 120 行，把只显示页数改为显示总条数
    // 改前
    showTotal: (t) => "共 " + Math.max(1, Math.ceil(t / pageSize)) + " 页",
    // 改后（默认总条数；需要页数时显式传 formatter）
    showTotal: options.showTotalFormatter
      ?? ((total) => "共 " + total + " 条"),
    showSizeChanger: true,
    pageSizeOptions: ["10", "20", "50", "100"],

要点：
- total 为 0 时 antd 不渲染分页，无需额外兜底。
- 手写分页的 10 个文件统一改为 usePagination，删除各自的 showTotal/pageSizeOptions（ScheduleManagement.tsx:288-297 等）。
- 禁止页面在 pagination 上覆盖 pageSize（my-duty/index.tsx:417-431 的反例），pageSize 只由 hook 持有。

### 5.2 useSubmit：弹窗提交唯一入口（消除 P0-03）

    // hooks/useSubmit.ts
    export function useSubmit<TArgs extends unknown[]>(
      action: (...args: TArgs) => Promise<void>,
      options?: { successText?: string; onDone?: () => void }
    ) {
      const [submitting, setSubmitting] = useState(false);
      const { message } = App.useApp();
      const run = async (...args: TArgs) => {
        if (submitting) return;            // 防连点（第一道闸）
        setSubmitting(true);
        try {
          await action(...args);
          if (options?.successText) message.success(options.successText);
          options?.onDone?.();
        } catch (e) {
          handleApiError(e, options?.successText ? options.successText + "失败" : "操作失败");
        } finally {
          setSubmitting(false);
        }
      };
      return { submitting, run };
    }

    // 页面用法：Modal 必须接 confirmLoading
    const { submitting, run } = useSubmit(async () => {
      const values = await form.validateFields();
      await updateDutyPool(editingRecord.id, values);
    }, { successText: "更新成功", onDone: () => { setVisible(false); fetchList(); } });

    <Modal open={visible} onOk={run} confirmLoading={submitting} ... />

要点：Modal.confirm 的 onOk 也要返回该 Promise，这样确认框自带 loading；本片 16 处 Modal.confirm 一并改造。

### 5.3 SearchSelect：输入控件默认能力收口（消除 P1-01/P1-02/P1-12）

    // components/shared/SearchSelect.tsx（antd Select 薄封装）
    interface SearchSelectProps extends SelectProps {
      /** 本地过滤字段，默认 label */
      filterProp?: string;
    }
    export function SearchSelect({ filterProp = "label", ...rest }: SearchSelectProps) {
      return (
        <Select
          showSearch
          allowClear
          optionFilterProp={filterProp}
          filterOption={(input, option) =>
            String(option?.[filterProp] ?? "").toLowerCase().includes(input.toLowerCase())
          }
          {...rest}
        />
      );
    }

要点：
- 业务页把 Select 换成 SearchSelect，删除空 onSearch；需要远程搜索时传 onSearch（保留一个真实实现，不是空函数）。
- 输入类控件同理：在 project 级封装 Input.WithClear（或统一在 Form.Item 层默认 allowClear），不逐处改 45 个 Input。
- 配套 ESLint 规则：no-restricted-syntax 禁止 onSearch={() => {}} 与 JSX 中的裸 hex 颜色字面量——从机制上防止回归。

## 6. 落地路线图

### 6.1 批次 A（P0，预计 0.5-1 人日）——先堵数据正确性

| 顺序 | 动作 | 涉及文件 | 验收口径 |
| --- | --- | --- | --- |
| A1 | 修 usePagination.ts:120 showTotal 为"共 N 条" | hooks/usePagination.ts | 任取 duty/schedules、orders、users 三页，分页右侧出现"共 N 条"，且 N 与后端 total 一致 |
| A2 | my-duty 删除 pageSize:10 硬覆盖 | pages/duty/my-duty/index.tsx:417-431 | 切 10/20/50 条后请求参数、表格行数、页码三者一致；刷新后保持（sessionStorage） |
| A3 | ous 用户列表改服务端分页 | pages/ad-domain/ous/index.tsx:130-131、:539 | 构造 >100 用户数据域，可翻到最后一页且总数正确 |
| A4 | 27 个 Modal 接 useSubmit + confirmLoading | pools/users/orders/templates/categories/configs/AccountPoolTab 等 | 提交中确定按钮 loading 且不可再点；网络节流下只产生 1 次写请求 |

### 6.2 批次 B（P1，预计 2-3 人日）——抽 3 个骨架，先试点后全量

| 顺序 | 动作 | 涉及文件 | 验收口径 |
| --- | --- | --- | --- |
| B1 | 抽 SearchForm + DataTable，试点 3 页 | duty/schedules、workorder/orders、ad-domain/users | 试点页行数下降 ≥30%；回车可查询；空态/错误态/列配置齐全 |
| B2 | 全量替换空 onSearch、补 showSearch | 12 文件 112 处 | 全局搜索 onSearch={() => {}} 结果为 0；每个 Select 可输入过滤 |
| B3 | 空态/错误态接入（复用现有组件） | 22 页 | 22 页均有空态；请求失败有重试按钮 |
| B4 | 4 个静态 message 文件迁移 | AccountPoolTab、configs/index、groups、users | 本片 import { message } from "antd" 为 0；无 antd 上下文 warning |
| B5 | 删除死代码 ous/index_with_dept.tsx | 该文件 + 测试引用 | 构建/lint 通过，测试不引用 |
| B6 | 统一二次确认 | 25 处 | 危险操作 100% 有确认且确认后按钮 loading |

### 6.3 批次 C（P2，预计 1 周）——一致性与可维护性

| 顺序 | 动作 | 验收口径 |
| --- | --- | --- |
| C1 | StateTag/DictTag 收敛 17 处状态三元 | 同一状态在本片文案/颜色唯一；新增状态只改 1 处 |
| C2 | hex → token（129 处） | 业务页面无裸 hex；暗色主题下无对比度告警 |
| C3 | 日期 formatter 统一（20 处） | 无 .format("YYYY-" 字面量；提交统一 formatDate |
| C4 | 巨型文件拆分（CAD 1537 行优先） | CADFloorPlanEditor ≤ 800 行，状态收敛到 useEditorState |
| C5 | 列配置/序号/scroll 全量接入 | 22 页均可用列配置与序号 |

### 6.4 回归守护（建议加入 CI）

- ESLint 自定义规则：JSX 内禁止空 onSearch、禁止裸 hex 颜色、Modal 必须传 confirmLoading（可通过自定义 rule 或 lint 插件实现）。
- 审查快照对比：分页、空态、错误态、allowClear/showSearch 的出现率作为月度指标（本报告数字可直接作为基线：本片 showTotal 正确 5/22、空态 2/22、错误态 4/22、回车 3/22、列配置 0/22）。

---

附：本片可复用的正例清单（改造时优先抄这些）
- 搜索与回车：pages/dashboard-system/components/DashboardList.tsx:298-304（Input allowClear + onPressEnter）、pages/ad-domain/configs/ReconciliationTab.tsx:247-266、pages/knowledge/view/index.tsx:183-198（antd Search）。
- 三态最佳实践：pages/ad-domain/configs/ReconciliationTab.tsx:281-299（ErrorAlertWithRetry + EmptyStateWithAction + Table loading）。
- 删除确认：pages/dashboard-system/components/DashboardList.tsx:257-266（Popconfirm）、AccountPoolTab.tsx:343-351。
- 操作列收口：pages/knowledge/articles/columns.tsx:137-188（ActionButtons）。
- 表格组合键 rowKey：ReconciliationTab.tsx:64（rowKeyOf）。
