# Milestone v1.22 Requirements — 前端公用组件整治

> 事实基线：`docs/2026-10-23-前端公用组件审查与最佳实践方案.md` + `.planning/notes/frontend-audit/{01..04}*.md`（合计 201 条：P0 24 / P1 87 / P2 90，全部精确到 文件:行号）。行号均相对 `ruoyi-react-frontend/`。

## v1.22 Requirements

### P0 止血（STOP）— 数据可达与安全

- [ ] **STOP-01**: 用户在所有分页列表页同时看到总条数与页数（如 `共 128 条 / 13 页`；total=0 时不显示误导性的「共 1 页」）—— 根因 `src/hooks/usePagination.ts:122` showTotal 只输出页数
- [ ] **STOP-02**: 用户能翻到 backups / discoveries 列表的任意一页 —— 根因 `src/pages/network/backups/index.tsx:79`、`src/pages/network/discoveries/index.tsx:34` 的 `_setTotal` 从未调用，total 恒 0
- [ ] **STOP-03**: 用户能看到 AD 域 OU 列表第 3 页及以后的数据 —— 根因 `src/pages/ad-domain/ous/index.tsx:130` 固定 `pageSize:100` 一次拉取却在表格做前端分页
- [ ] **STOP-04**: 用户双击弹窗确认按钮不会重复提交 —— 27 个业务弹窗 `confirmLoading` 0 命中（值班池/排班/工单等写入入口）
- [ ] **STOP-05**: 用户触发批量删除前必见二次确认 —— 7 处批量删除无确认（system 6 处 + network 5 处含重叠）
- [ ] **STOP-06**: 用户不再看到点了没反应的假功能入口 —— `src/pages/operations/assets/index.tsx:203` / `:687-691` 空确认、`ReconciliationDrawer.tsx:67-70` 永久空 Tab
- [ ] **STOP-07**: 用户在宽表上滚动时固定操作列保持在右侧 —— 6 处 `fixed:"right"` 落在没有 `scroll.x` 的表上（user:272、config:291、dict:129/231、captcha-background:96 等）

### P1 修复（FIX）— 反馈与可访问性

- [ ] **FIX-01**: 用户在数据加载时看到表格 loading 状态（network/monitor 全片 0 命中）
- [ ] **FIX-02**: 用户误点遮罩不会丢失未保存的表单内容 —— Modal `maskClosable` 0/22 全部可点即关，改为受控 + 脏检查
- [ ] **FIX-03**: 用户在 cache/job 等分页表上排序得到全量正确结果 —— 本地排序误用（cache 7 列 / job 6 列）改服务端排序或明确降级说明
- [ ] **FIX-04**: 用户在请求失败时看到可见的错误提示而非静默 —— `console.*` 52 处收敛到统一错误出口
- [ ] **FIX-05**: 使用读屏器的用户能识别图标按钮 —— `aria-*` / `role` / `tabIndex` 全为 0，图标按钮补可访问名

### 契约组件（COMP）— 默认值与契约下沉

- [ ] **COMP-01**: 新页面使用 AppInput/AppSelect 零配置获得可清除/可搜索行为 —— `<Input>` 285 个仅 66 个 allowClear（23%）、`<Select>` 400 个仅 122 个（30%），默认值必须下沉封装层，禁止逐页补（横跨 73 文件）
- [ ] **COMP-02**: 所有危险删除操作统一经 ConfirmDelete 组件（唯一入口，强制二次确认）
- [ ] **COMP-03**: 新列表页使用 PageTable 一站式骨架（分页/loading/空态/错误态契约内建，模板页样板 ~450 行 → ~120 行）
- [ ] **COMP-04**: 新表单弹窗使用 FormModal（confirmLoading/脏检查/destroyOnHidden 契约内建）
- [ ] **COMP-05**: 存量页面按 ROI 顺序迁移（C1 试点 discoveries 最小页 → backups → devices/cache 巨型页随拆分迁移；每页一个 PR，旧实现保留一个迭代；SearchForm 并入 PageTable 体系）

### P2 债务（DEBT）

- [ ] **DEBT-01**: 颜色统一走 design token —— 6 位 hex 109 处/25 文件（含 3 位简写共 125 行命中）收敛
- [ ] **DEBT-02**: 日期/状态展示统一 helper —— 裸三元 13 处/6 文件、手写日期格式 12 处/8 文件
- [ ] **DEBT-03**: 类型收敛 —— `: any` / `as any` 27 处/16 文件（随 COMP-04 FormModal 迁移逐步消除）

## 抽取原则（locked）

- 抽契约不抽业务：资产选择器、部门树选择器等业务语义组件已够用，不抽
- `useDebouncedSearch` 并入 RemoteSelect，不单独抽
- 巨型文件拆分不作为抽取立项（随 COMP-05 迁移顺带）
- 默认值必须下沉封装层；ConfirmDelete 是危险操作唯一入口

## Future Requirements（deferred）

- 三态组件（Empty/Skeleton/ErrorAlert）在 system 域的推广（0 使用量）—— 随 COMP-03 PageTable 间接覆盖，单独推广延后
- `useTableManager` 使用率提升（26% → 目标随 PageTable 自然达成）
- CronSelector 假搜索（25 处纯数字选择器）—— 单独评估，暂不纳入

## Out of Scope

- 业务语义组件重设计（资产选择器、部门树选择器）
- 巨型文件主动拆分重构（仅随迁移顺带）
- 后端接口改造（本里程碑纯前端）
- i18n（useTranslation 0/全片，另行立项）

## Traceability

（由 roadmap 填充：REQ → Phase 映射）
