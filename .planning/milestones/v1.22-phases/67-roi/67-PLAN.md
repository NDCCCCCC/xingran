---
phase: "67"
slug: "67-roi"
type: plan
created: "2026-10-25"
planner: gsd-planner
---

# Phase 67 执行计划 — 存量迁移（按 ROI 顺序落地契约组件）

**目标**：四页存量（discoveries 405 → backups 1141 → devices 1282 → cache 974 行）按 ROI 顺序迁移到 66 契约组件体系（PageTable/SearchForm/FormModal，API 零修改），每页独立 PR 可 revert，行为不变式 INV-1~12 逐字保持，行数下降可量化。

**权威契约**：`.planning/phases/67-roi/67-UI-SPEC.md`（approved，含 Probe Coverage 17 resolved + 2 dismissed）。任务书与本 PLAN 冲突时以 UI-SPEC 为准。前置：65/66 契约与文案模板（`ConfirmDelete`/`confirmDelete`/`FormModal` 脏检查文案）逐字沿用，不重述。

**验证基线**（66-SUMMARY.md 收口实测，planner 指定口径）：
- `npm run type-check` → 0 错误
- `npm run test -- --run` → **3900 passed / 0 failed**（564 文件；任何 failed 均为新增回归）
- `npm run lint` → 全仓 **83 errors** 存量 / 1414 warnings；**禁跑全量 lint 判闸**，验证一律限定文件范围或数新增
- 65 豁免清单 142 文件（selector 类）；66 豁免清单 **45 文件**（hook imports 类）——本 phase 逐页收缩两清单

**硬边界**（locked，违反即任务失败）：
- 四组件 `PageTable.tsx` / `SearchForm.tsx` / `FormModal.tsx` / `RemoteSelect.tsx` 对外 API **零修改**；发现契约缺口 → 停下上报 planner，不在迁移 PR 内顺手扩 API
- 迁移零 UI 变化：凡用户可感知差异必须落在 UI-SPEC §1.2 B/C 类已裁决清单内，executor 不得顺手"改进"任何未列出的行为
- cache 页现存裸 hex（`#3f8600`/`#337ab0` 等）**原样保留**（归 P68 DEBT-01），迁移过程禁止新增任何裸 hex
- 拆分仅随迁移顺带（ROADMAP Notes locked）：只拆 UI-SPEC §3.3/§3.4 列出的子组件，不单独立项、不额外抽 hook
- 页外文件零触碰：除四页目录 + 两豁免清单（eslint.config.js + gen-p66-exemptions.mjs 复跑）+ SUMMARY 外，不得修改任何文件（含 hooks/、components/、App.tsx）
- 不做 `index-v2.tsx` 文件级并存；回滚走 git revert（UI-SPEC §4）
- 全程不 commit（PR 与 revert 演练由 lead 在验收后操作；executor 只保证每页改动自成一组、`git revert --no-commit` 可干净回退）

**行号锚点纪律**：本 PLAN 全部 `文件:行号` 锚点为 2026-10-25 快照（HEAD），漂移需重定位——以语义锚（函数名/文案字符串/规则名）重新 grep，不得按失效行号盲改。

**Windows 大小写纪律**：新建文件 import 路径一律带显式扩展名语义的 `.tsx`（`./DeviceStatsCards` 等按现有页面惯例可省，但组件文件名与 import 名大小写必须逐字一致；禁止出现仅大小写不同的孪生文件——65 教训 ConfirmDelete.tsx/confirmDelete.ts）。

## 事实核查注记（planner 实测，executor 须知）

1. **⚠️ UI-SPEC §1.2 B 类 discoveries 行的事实修正**：UI-SPEC 称「discoveries 删除现状无确认弹窗，直接 post delete」。**实测不符**：确认弹窗存在于 `src/pages/network/discoveries/columns.tsx:114-121`（删除 action 的 onClick 内包 `Modal.confirm`：`确认删除?` / `确定` / `取消` / danger；`index.tsx:146-154` 的 handleDelete 是 onOk 体内的直连 post，两者是入口/执行体关系而非"无确认"）。**裁决不变、口径修正**：迁移到 `createDeleteAction`→`ConfirmDelete` 后文案按 65 模板对齐（`确认删除` / `将删除{target}，删除后不可恢复，是否继续？` / `删除` / `取消`），PR 描述按「文案对齐（B 类）」记录，**不按「STOP 级补确认缺口」记录**。onOk Promise 语义天然获得提交中防重（63 STOP-04），替代现状无防重的裸 confirm。
2. **四组件签名实测**（迁移目标，零修改）：`PageTableProps`（`PageTable.tsx:42-78`，含 `children` render-prop ctx `:81-94`）、`SearchField` 4 型（`SearchForm.tsx:30-48`）、`FormModalProps`（`FormModal.tsx:33-49`，okText 默认「保存」）、`RemoteSelectProps`（`RemoteSelect.tsx:27-35`）。
3. **backups setTotal effect** 实测在 `backups/index.tsx:176-179`（63 T2 产物注释 `STOP-02` 字样）；`resolveSorter` 时序修复在 `:677-696`（v129-recheck WR-02 注释可 grep 定位）；两处还原确认 `:206-214`（`确认恢复配置`）/`:527-533` 附近（`确认恢复到此版本?`）。
4. **devices 双 hook 并存**：`usePagination`（`:295`）+ `useTableManager`（`:354-394`，本 phase「hook 降格」最直接兑现）；单删裸 confirm `:242-257`（含 confirmInstance.update 防抖，随迁移放弃）；批删 confirmDelete 合规 `:518-537`；virtual scroll `:909-910`；防误关 `:921-922`。
5. **cache 页非 Form 搜索区**：`cache/index.tsx:198` 受控 state `setSearchForm`、`:568-638` 7 按钮搜索区（操作缓存/搜索/重置/刷新/统计信息/导出/清空）；清空确认 `:311-330`（`确认清空` 原文保持）；`resolveSorter` 用法 `:417`；降级 Tag 在监控概览 `:489-564` 搬迁面内。
6. **discoveries 列定义在独立 `columns.tsx`**（`getDiscoveryColumns` 收 handleViewResult/handleDelete/getSortOrder 闭包，`sortOrder: getSortOrder?.(...)` 手工视觉 `:85/:93`）——迁移时该文件**保留原样**（66 范围纪律：列文件不动），PageTable 只接 `columns={getDiscoveryColumns({...})}` 时需裁掉 getSortOrder 闭包与列内删除 confirm：**最小侵入方案**是给 `getDiscoveryColumns` 传参收窄（`getSortOrder` 传 undefined、删除 action 改为直接 `onClick: () => handleDelete(record.id)` 交给 `rowActions` 的 createDeleteAction 渲染）——即把操作列从 columns.tsx 挪到页面 `rowActions`，columns.tsx 内删除块删除。这是迁移的必要伴随改动，记 PR。
7. **discoveries 数据流**：`useDiscoveryData` 同时供列表 + 统计 + 结果弹窗（耦合较深）——按 UI-SPEC §3.1 裁决执行：request 直连列表 API 优先；若统计/结果加载与列表流强耦合（实测 `loadDiscoveries` 被轮询/导入/删除多入口复用），退路是 `children` ctx.loadData 桥接。执行时按行为不变取舍并记 PR。
8. **retry 语义**（Probe Coverage M1-error 裁决）：discoveries/backups/cache 现状「retry = 列表 + 统计/监控都刷」必须保持——经 `children` 内 useEffect 联动或保留手工 ErrorAlertWithRetry 条实现，取舍记入各页 PR 描述。
9. **豁免清单收缩机制**：每页迁移后 grep 该页目录确认裸组件/裸 confirm/hook import 清零，随后复跑 `node scripts/gen-p65-exemptions.mjs` 与 `node scripts/gen-p66-exemptions.mjs`（幂等），确认对应页文件从两清单移除；T5 记录逐页清单数变化。

## T1 — C1 试点：discoveries 迁移 PageTable（含删除文案对齐 + 状态筛选接线）

**exact files**：
- 修改 `ruoyi-react-frontend/src/pages/network/discoveries/index.tsx`（405 行 → 目标 ~200-240）：
  - 删 `usePagination` + `setTotal(hookTotal)` effect（`:34,:206-215`）、`useServerSort` + `createSorterMeta ×5`（`:37-54`）、手工搜索卡（`:281-362`）、错误条（`:219-225`）、Table Card + onChange 拼接（`:364-382`）
  - 改 `<PageTable request columns={getDiscoveryColumns(...)} rowKey="id" scroll={{x:1500}} searchFields={[{type:"select",name:"status",label:"状态",options:STATUS_OPTIONS}]} rowActions batchDelete? 无（本页无批删） sorterMetas toolbar>`；统计卡 Row ×6（`:227-278`）迁 `children` render-prop（或拆 `StatisticsCards.tsx`，二选一记 PR）
  - 状态筛选**接线生效**（C 类 1 默认裁决）：`searchFields` schema 使 status 进入请求参数——记 PR 首条缺陷修复；request 内透传 filters.status
  - 新增「重置」按钮 + 查询→搜索文案（C 类 2，SearchForm 内建自动获得）；retry 联动统计（事实注记 8）
  - 删除走 `rowActions` + `createDeleteAction(id, handleDelete, { label: 任务名 })`——B 类文案对齐（事实注记 1 口径）；`handleDelete` 内 message.success("删除成功") 保留语义（65 helper 自带"已删除"，对齐后以 helper 文案为准，差异记 PR）
- 修改 `ruoyi-react-frontend/src/pages/network/discoveries/columns.tsx`（仅收窄：删操作列删除块 `:109-123` 与 `getSortOrder` 参数及 `sortOrder:` 手工视觉，Modal import 若无他用一并删；其余列逐字不动）
- 修改 `ruoyi-react-frontend/src/pages/network/discoveries/__tests__/`（既有 `index.render.test.tsx` 回归锚适配：mock 端点后走真实 PageTable 链路，断言保留：分页 showTotal、删除确认文案、统计卡渲染）
- 复跑 `ruoyi-react-frontend/scripts/gen-p65-exemptions.mjs`、`gen-p66-exemptions.mjs`（discoveries 移出后清单数变化记录）

**read_first**：UI-SPEC §3.1 映射表 + §1.2 B/C 类裁决；`pages/system/post/index.tsx`（66 试点样板，480→268 同型）；discoveries `./hooks` 全文（裁决数据流路线）；`ActionButtons.tsx:114` createDeleteAction 位置参数签名

**acceptance_criteria**：
1. 用户视角行为不回归：分页 `共 N 条 / M 页`（total=0 不渲染）、空态 antd 默认、排序首点即生效、轮询（useDiscoveryPolling）继续工作、CreateModal/ResultModal/NetworkExport/BatchExportModal 原样
2. 删除确认文案 = 65 模板逐字；提交中防重（onOk Promise）；失败 toast 且可重试
3. 状态筛选接线生效：选择状态后请求参数含 status（测试断言 mock 端点收到参数）
4. grep `usePagination|useServerSort|setTotal|Modal.confirm` 于页面目录 = 0 命中（columns.tsx 亦然）
5. 行数 405 → ~200-240（记实际值）；type-check 0 / 定向 vitest 全过 / lint 该目录 0 新增 error / 两豁免清单各 -1 文件

**fails_when**：INV-1~12 任一口径回归（尤其 INV-1 分页文案、INV-6 排序首点）；四组件任一 API 被改；CreateModal/ResultModal/轮询被顺手迁移或改动；页外文件（hooks/、components/）被触碰；豁免清单未收缩；出现 `index-v2.tsx` 并存

**verification**：
```bash
cd ruoyi-react-frontend
npm run type-check
npx vitest run src/pages/network/discoveries/
npm run test -- src/pages/network
npx eslint src/pages/network/discoveries/
grep -rnE "usePagination|useServerSort|setTotal\(|Modal\.confirm" src/pages/network/discoveries/   # 期望 0
node scripts/gen-p65-exemptions.mjs && node scripts/gen-p66-exemptions.mjs && git diff --stat eslint.config.js
wc -l src/pages/network/discoveries/index.tsx   # 记录量化值
# 人工冒烟：UI-SPEC §2.2 模板逐项（分页/空态/错误重试/排序/删除确认/轮询/创建弹窗）
```
预估：4h

## T2 — backups 迁移（SC2 验收点：setTotal 承接 + resolveSorter 时序保持）

**exact files**：
- 新建 `ruoyi-react-frontend/src/pages/network/backups/modals/RestoreModals.tsx`（纯 JSX 搬迁：还原版本列表 Table + 恢复进度 + diff + 内容查看 Modal 群，约 400 行，`:738-1141` 一带——零逻辑改动，props 显式传参）
- 修改 `ruoyi-react-frontend/src/pages/network/backups/index.tsx`（1141 → 目标 ~820-880）：
  - 删 setTotal effect（`:176-179`，**SC2 验收点**：分页行为由 PageTable 内建 total 等价承接，与 63 运行时验收 STOP-01/02 基线一致——「共 171 条 / 4 页」口径）
  - onChange 手工拼参数 + resolveSorter（`:677-696`）→ `sorterMetas` prop（INV-6 验收点：首点排序即生效不发空）
  - 搜索卡（`:597-648`）→ `searchFields` 2 字段（deviceName input + backupType select）；新增重置按钮 + 查询→搜索（C 类 2）
  - 统计卡 Row ×4（`:553-584`）→ children render-prop；NetworkExport filterEmpty（`:653-663`）原样保留于 toolbar
  - 创建备份 Modal（`:701-735`）→ FormModal：**显式 `okText="确定"`**（零 UI 变化优先于 66 默认「保存」，注释注明）；dirtyCheck 默认 true（INV-10 红利，记 PR）
  - 两处还原确认**保持现状原文不动**（B 类豁免：`确认恢复配置`/`确认恢复到此版本?`，lint 豁免清单承接）；RestoreModals 拆出后调用点接线
  - retry 联动统计（事实注记 8）
- 修改 `ruoyi-react-frontend/src/pages/network/backups/__tests__/`（既有测试适配；SC2 断言：total 回填链路 + 翻页数据可达不回归）
- 复跑两个豁免脚本（backups 相关文件移出）

**read_first**：UI-SPEC §3.2 全表；63-P0-SUMMARY.md 运行时验收表（STOP-01/02 基线）；backups index.tsx 全文（1141 行，分段读）

**acceptance_criteria**：
1. SC2：PageTable 契约下分页 total 正确回填、翻页数据可达（测试 + 人工冒烟双验）；grep `setTotal` 于 backups 目录 = 0
2. INV-6：点击列头排序图标首次点击即携带 orderByColumn/isAsc 发请求（无时序空发）
3. RestoreModals 拆出后全部弹窗行为与迁移前一致（还原确认/进度/diff/内容查看）；两处还原确认文案逐字未动
4. 创建备份弹窗：防重（INV-9）+ 脏检查（INV-10，记 PR 红利）+ okText「确定」
5. 行数 1141 → ~820-880（index 实测值 + RestoreModals 行数分开记录）；基线四项全绿

**fails_when**：SC2 分页行为与 63 基线不一致（total 恒 0 / 空态渲染「共 0 条」）；resolveSorter 语义丢失（排序首点空发）；还原确认文案被"顺手对齐"为 65 模板（明确保持现状）；RestoreModals 搬迁夹带逻辑改动；FormModal okText 用了默认「保存」

**verification**：
```bash
cd ruoyi-react-frontend
npm run type-check
npx vitest run src/pages/network/backups/
npm run test -- src/pages/network
npx eslint src/pages/network/backups/
grep -rnE "setTotal\(|usePagination\(" src/pages/network/backups/   # 期望 0
grep -n "确认恢复配置\|确认恢复到此版本" src/pages/network/backups/modals/RestoreModals.tsx src/pages/network/backups/index.tsx   # 原文仍在
wc -l src/pages/network/backups/index.tsx src/pages/network/backups/modals/RestoreModals.tsx
```
预估：5h

## T3 — devices 拆分迁移（三子组件 + 部门树 extraFilters + INV-8 逐字）

**exact files**：
- 新建 `ruoyi-react-frontend/src/pages/network/devices/components/DeviceStatsCards.tsx`（统计卡 Row，`:686-727` 附近，纯 JSX 搬迁）
- 新建 `ruoyi-react-frontend/src/pages/network/devices/components/DeptTreePanel.tsx`（部门树 + 选中回调；选中变化 → 父组件 `extraFilters={{ deptId }}`——PageTable `:162-169` 既有语义等价现状回第 1 页；useMemo 包引用避免每渲染触发）
- 新建 `ruoyi-react-frontend/src/pages/network/devices/components/DeviceModals.tsx`（快速创建 + 编辑 Modal 群 `:919-1282`：两表单弹窗迁 FormModal（删手写 submitting state，INV-9 内建）；详情只读 Modal 原样保留）
- 修改 `ruoyi-react-frontend/src/pages/network/devices/index.tsx`（1282 → 目标 ~350-420）：
  - `useTableManager + usePagination` 双 hook（`:295,:354-394`）→ PageTable 内部单实例
  - 搜索区 5 字段（`:740-820`，deviceTypeDict 动态 options）→ `searchFields`（动态 options 用 `custom` 类型或页面 useMemo 传入 select options）
  - 批删（`:518-537` confirmDelete 合规）→ `batchDelete={{ onDelete, target: "设备" }}`——**INV-8 文案逐字不变**（`确认批量删除` / `将删除选中的 {N} 项，删除后不可恢复，是否继续？` / `删除` / `取消` / danger）；「已选择」Alert（`:876-895`）迁 children ctx.selectedRowKeys（post 试点同模式）
  - 单删裸 Modal.confirm（`:238-257`，含 confirmInstance.update 防抖）→ `createDeleteAction`（B 类文案对齐：`确认删除?`→65 模板；onOk Promise 天然覆盖防重，C 类 4 裁决：批删防抖增强随迁移放弃）
  - `virtual scroll={{x:1600,y:600}}`（`:909-910`）rest 透传，行为不变
- 修改 `ruoyi-react-frontend/src/pages/network/devices/__tests__/`（适配 + INV-8 文案逐字断言保留/补齐）
- 复跑两个豁免脚本（devices 移出）

**read_first**：UI-SPEC §3.3 全表 + §1.2 C-4；devices index.tsx 全文（分段）；post 页批删/Alert 接线（`:277-310,:381-400` 一带 66 落地形态）

**acceptance_criteria**：
1. INV-8 逐字：批删确认四段文案 + danger 与迁移前 grep 比对零差异（测试断言）
2. 部门树点选 → 列表回第 1 页且请求带部门过滤（与现状 handleDeptSelect 行为等价）
3. 三子组件纯搬迁面：详情 Modal / 统计卡 / 部门树渲染逐帧一致（定向 render 测试 + 人工比对）
4. 快速创建/编辑：防重 + 脏检查 + 回填重置（INV-9/10/11）自动获得
5. virtual 滚动透传生效；单删文案对齐 65 模板并记 PR
6. 行数 1282 → index ~350-420 + 3 子组件（各 ~100-250）分开记录；基线四项全绿

**fails_when**：INV-8 文案任何字符漂移；extraFilters 每渲染触发重载（引用未 memo）；子组件搬迁夹带逻辑改动；PageTable/useTableManager API 被改以"适配"本页；virtual 属性丢失

**verification**：
```bash
cd ruoyi-react-frontend
npm run type-check
npx vitest run src/pages/network/devices/
npm run test -- src/pages/network
npx eslint src/pages/network/devices/
grep -rnE "useTableManager|usePagination\(|Modal\.confirm" src/pages/network/devices/   # 期望 0（confirmDelete/ConfirmDelete 契约内调用除外）
grep -n "确认批量删除" src/pages/network/devices/ -r   # INV-8 文案仍在（经 batchDelete/confirmDelete 模板则断言在测试内）
wc -l src/pages/network/devices/index.tsx src/pages/network/devices/components/*.tsx
```
预估：6h

## T4 — cache 拆分迁移（非 Form→Form 搜索区改造 + 清空确认保持原文）

**exact files**：
- 新建 `ruoyi-react-frontend/src/pages/monitor/cache/components/MonitorOverview.tsx`（L1/L2/命中率/状态 4 卡监控概览 `:489-564`，纯 JSX 搬迁；**硬编码 hex 原样保留**、降级 Tag 正常/异常切换逻辑原样搬——M5 裁决）
- 新建 `ruoyi-react-frontend/src/pages/monitor/cache/components/CacheStatsModal.tsx`（统计信息 Modal，`:834-974` 附近 ~140 行 Statistic 群，纯 JSX 搬迁）
- 修改 `ruoyi-react-frontend/src/pages/monitor/cache/index.tsx`（974 → 目标 ~420-500）：
  - 手工 fetch + setTotal（`:231,:248`）+ useServerSort（`:225,:417`）→ PageTable request + sorterMetas
  - 搜索区**非 Form → Form 改造**（C 类 3）：受控 state `setSearchForm`（`:198,:401,:568-638`）→ `searchFields` 3 字段（key/type/level）；行为等价（值进查询参数），视觉不变；**操作缓存/统计信息/导出/清空 4 个业务按钮进 `toolbar`**（搜索/重置/刷新由 SearchForm 内建接管）
  - 清空确认（`:311-330` `确认清空` 原文 + ExclamationCircleOutlined icon）**保持现状原文**，保留在 toolbar handler 内（B 类豁免；lint 豁免清单承接，不加 eslint-disable 散点）
  - 操作缓存/批量操作 Modal（`:782` 起）保留原样（若迁 FormModal 同 backups okText 口径——默认不迁，超范围不动）；columns 内 formatTTL/formatMemorySize/内嵌 Statistic（`:654-780`）原样保留
- 修改 `ruoyi-react-frontend/src/pages/monitor/cache/__tests__/`（适配 + 监控概览/统计 Modal render 测试）
- 复跑两个豁免脚本（cache 移出）

**read_first**：UI-SPEC §3.4 全表 + §1.2 C-3 + M5 裁决；cache index.tsx 全文（分段）；backups T2 的 FormModal okText 先例（若涉及）

**acceptance_criteria**：
1. 搜索区改造行为等价：key/type/level 值进入请求参数（测试断言 mock 端点收到）；清空/重置语义与现状一致
2. 清空确认弹窗文案 + icon + danger 语义与迁移前逐字一致（原文 grep 比对）
3. MonitorOverview/CacheStatsModal 搬迁零逻辑改动：定向 render 测试 + 人工比对；降级 Tag 切换逻辑原样
4. 4 个业务按钮在 toolbar 全部可用（操作缓存/统计信息/导出/清空）；导出仍携带当前搜索条件
5. 裸 hex 数量迁移前后不变（`grep -cE "#[0-9a-fA-F]{3,6}\b"` 前后相等——与 verification 命令同口径，含 3 位简写；只许搬不许增删）；行数 974 → index ~420-500 + 2 子组件分开记录；基线四项全绿

**fails_when**：清空确认文案被对齐成 65 模板（明确保持现状）；hex 被顺手收敛或新增；搜索区改造丢失任一字段/按钮；统计信息按钮丢失（C-3 校正后的「统计信息」文案）；cache 页目录外文件被触碰

**verification**：
```bash
cd ruoyi-react-frontend
npm run type-check
npx vitest run src/pages/monitor/cache/
npm run test -- src/pages/monitor
npx eslint src/pages/monitor/cache/
grep -rnE "setTotal\(|useServerSort|setSearchForm" src/pages/monitor/cache/   # 期望 0（子组件内如无亦然）
grep -cE "#[0-9a-fA-F]{3,6}\b" src/pages/monitor/cache/index.tsx src/pages/monitor/cache/components/*.tsx   # 与迁移基线比对
grep -n "确认清空" src/pages/monitor/cache/index.tsx   # 原文仍在
wc -l src/pages/monitor/cache/index.tsx src/pages/monitor/cache/components/*.tsx
```
预估：5h

## T5 — lint 豁免收缩 + 复扫总闸 + 回滚演练

**exact files**：
- 修改 `ruoyi-react-frontend/eslint.config.js`（仅经脚本：p65/p66 两豁免锚点段由脚本重写，人工不手改清单）
- 复跑 `ruoyi-react-frontend/scripts/gen-p66-exemptions.mjs`（幂等验证：连跑两次输出一致）
- 新建 `.planning/phases/67-roi/67-exemptions-ledger.md`（可选，若 SUMMARY 空间不足：逐页清单数变化台账）

**read_first**：66-SUMMARY「豁免清单对账」节（基线 142/45）；两脚本源码（幂等机制）

**acceptance_criteria**：
1. `no-restricted-imports` 0 新增命中：全仓 `npx eslint src 2>&1 | tail -3` 总 error = 83 存量 + 0 新增
2. 豁免清单递减记录：65 清单 142 → N₁、66 清单 45 → N₂，每页移出的文件名逐条可对账（四页各自目录内文件）
3. gen-p66-exemptions.mjs 幂等（复跑 diff 为空）
4. **四组件 API 零修改 diff 验证**：`git diff HEAD -- src/components/shared/PageTable.tsx src/components/shared/SearchForm.tsx src/components/shared/FormModal.tsx src/components/ui/RemoteSelect.tsx src/hooks/useTableManager.ts` = 空（本 phase 全程复核一次）
5. 回滚演练：每页迁移改动在独立分支上 `git revert --no-commit` 干净（无后续 commit 依赖迁移产物；executor 以本地工作区 stash/分支模拟验证并记录，实际 PR/revert 由 lead 执行）
6. 基线总闸：type-check 0 / 全量 vitest **3900+ passed / 0 failed**（新增测试计入 passed 数）/ lint 新增 error = 0

**fails_when**：任一迁移页仍残留在豁免清单而其裸组件/裸 confirm/hook import 已清零（清单与实际命中集不一致）；四组件或 useTableManager 出现 diff；全量 vitest 出现任何 failed；清单收缩遗漏导致「已迁移文件仍被豁免」掩盖 lint 新增

**verification**：
```bash
cd ruoyi-react-frontend
node scripts/gen-p65-exemptions.mjs && node scripts/gen-p66-exemptions.mjs
node scripts/gen-p66-exemptions.mjs && git diff --stat eslint.config.js   # 二次复跑应为空 diff（幂等）
npx eslint src 2>&1 | tail -3        # 83 存量 + 0 新增
git diff HEAD --stat -- src/components/ src/hooks/   # 仅允许 0 变更（四组件+useTableManager）
npm run type-check && npm run test -- --run
```
预估：2h

## T6 — 收口 SUMMARY（行数三栏量化表 + SC1-4 verdict，不 commit）

**exact files**：
- 新建 `.planning/phases/67-roi/67-SUMMARY.md`（不 commit）

**read_first**：66-SUMMARY（结构模板）；本 PLAN 全部任务 verification 输出；UI-SPEC §5 行数基线表

**acceptance_criteria（SC 逐条，对齐 ROADMAP Phase 67 SC1-4）**：
- **SC1**（discoveries 试点零回归）：T1 输出——INV-1~12 冒烟逐项 + 状态筛选接线生效 + 删除文案对齐记录（含事实注记 1 的口径修正说明）
- **SC2**（backups setTotal 保持）：T2 输出——分页行为与 63 STOP-01/02 基线一致证据（测试断言 + 冒烟）
- **SC3**（devices/cache 拆分迁移零回归）：T3/T4 输出——INV-8 逐字、子组件搬迁比对、清空确认原文
- **SC4**（每页独立 PR 可回滚 + 行数可量化）：**行数三栏量化表**（必填）：

| 文件 | 迁移前 | 迁移后（实测） | 拆出子组件（行数） |
|------|-------|--------------|------------------|
| discoveries/index.tsx | 405 | （填实测） | — |
| backups/index.tsx | 1141 | （填实测） | modals/RestoreModals.tsx（填实测） |
| devices/index.tsx | 1282 | （填实测） | components/ 3 文件（填实测） |
| cache/index.tsx | 974 | （填实测） | components/ 2 文件（填实测） |

- **PR 承接节**：四页各自 PR 粒度与 B/C 类差异逐条清单（供 PR 描述复用）；revert 粒度说明（squash 单 commit / 多 commit 用 `git revert -m 1`，UI-SPEC §4）；旧实现保留一个迭代声明（迭代结束无回归后再开清理 PR 复扫豁免清单）
- **豁免清单对账节**：T5 的 N₁/N₂ 递减数据
- SUMMARY 含：任务完成表（T1-T6 + 不 commit 声明）、验证命令与输出、遗留上报（CreateModal/ResultModal FormModal 迁移、操作缓存 Modal、裸 hex 归 P68 等）

**fails_when**：任一 SC 无可执行验证输出支撑；量化表缺实测值或与 wc -l 不符；基线任一项回退；SUMMARY 声称与实际输出不符

**verification**：SUMMARY 内嵌全部命令输出
预估：1.5h

---

## 依赖顺序与 PR 粒度

```
T1（discoveries 试点，必须先行——C1 风险最低点校准后续三页）
 └→ T2（backups，T1 验收通过后开）→ T3（devices）→ T4（cache）→ T5（豁免收缩 + 总闸）→ T6（收口）
```
- **每页一个独立 PR**（UI-SPEC §4 契约）：T1/T2/T3/T4 各自改动自成一组（页面目录 + 测试 + 豁免清单复跑），前页验收通过再开下页；executor 按 UI-SPEC §2.2 模板逐页收口验证。
- 顺序 locked（ROI：最小页校准 → 大页递进）；T3/T4 理论可并行但共享 eslint.config.js 豁免段，**串行执行避免清单写冲突**。
- 可并行：无（本 phase 全串行，每页 PR 边界优先于工时并行）。

**总工时预估：23.5h**（T1 4h / T2 5h / T3 6h / T4 5h / T5 2h / T6 1.5h）

## 执行注意（executor 必读）

1. 全量 `npm run lint` 与 `npm run test -- --run` 耗时长——判闸一律用定向命令 + 基线对比（type-check 0 / vitest 3900+ 且 failed=0 / lint 新增 error=0 对比 83 存量）。
2. **每页迁移前先录行为基线**：UI-SPEC §1.1 INV-1~12 逐项（分页文案复制/删除确认文案 grep 存档/排序交互），迁移后逐项比对——这是「行为不回归」的唯一可执行判据。
3. 豁免清单是活文档：每页迁移后立即复跑两脚本（不等 T5），T5 只做总对账 + 幂等验证；清单收缩遗漏 = 已迁移文件的裸命中被豁免掩盖，属闸门失效。
4. 任何任务发现必须触碰 UI-SPEC 未覆盖的组件/行为（尤其四组件 API 缺口），停下上报 planner，不得自行扩面（硬边界 locked）。70+ 其余存量页、operations/duty/workorder 域一律不碰。
5. 事实注记 1 的口径修正（discoveries 删除现状已有 confirm）务必写进 T1 PR 描述，避免验收方按 UI-SPEC 原文「补缺口」误判为新增行为。
