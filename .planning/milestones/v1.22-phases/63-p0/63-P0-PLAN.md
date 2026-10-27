# Phase 63 实施计划 — P0 止血：数据可达与安全（STOP-01..07）

> 里程碑：v1.22「前端公用组件整治」 · 目标代码：`ruoyi-react-frontend/`（React 19 + TS 5.9 + AntD 6.1 + Zustand）
> 契约：`63-UI-SPEC.md`（APPROVED）· 需求：`REQUIREMENTS.md` STOP-01..07 · 路线图：`ROADMAP.md` Phase 63
> 行号基准：2026-10-24 对源码逐一复核（F3 要求，非直接引用审计快照）。执行时若源码已漂移，以 grep 命令重新定位，语义不变。

## 0. 全局约束（每个任务都必须遵守）

1. **零新组件/封装**：本 phase 不新建任何组件、hook、wrapper（组件抽取归 Phase 65/66）。STOP-04 只能用「onOk 返回 Promise」或「confirmLoading 受控 + 守卫」两种写法。
2. **不新增裸 hex 颜色**；样式只用既有 `var(--theme-*)` / Tailwind 工具类。
3. **不新增静态 `message.*`**：一切提示走 `App.useApp()` 的 context 版 message（各页已有先例）。
4. **纯前端**：唯一允许触碰后端的是 T0 的只读冒烟验证，不改任何 Go 代码。
5. **文案契约**（UI-SPEC Copywriting）：分页总计 `共 {total} 条 / {pages} 页`；删除成功 `已删除`；禁止 `待实现` / `暂未开放` 类文案；错误出口走 `src/utils/errorHandler.ts` 的 `handleApiError`。
6. **不扩 scope**：手写 showTotal 存量页、无 scroll 的非 fixed 表、my-duty 双通道分页（04 分片 P0-04）、menu 静默 catch、network 假搜索等一律不动（归 Phase 64/67）。
7. **不 commit**。每任务完成后仅运行验证命令并汇报。

## 0.1 成功标准映射（ROADMAP.md Phase 63 五条）

| # | 成功标准 | 对应任务 |
|---|---------|---------|
| SC1 | 任意分页列表页显示「共 N 条 / M 页」，total=0 不显示「共 1 页」 | T1（51 页生效）；T2/T3 触碰页同步改用 paginationProps |
| SC2 | backups / discoveries 可翻任意页且每页数据与总数一致 | T2 |
| SC3 | OU 列表用户表第 3 页及以后全部可达，不再被 100 条截断 | T3 |
| SC4 | 快速双击任意业务弹窗确认按钮，后端只收到一次请求、无重复记录 | T5 |
| SC5 | 批删必见二次确认；3 处假入口消失；宽表滚动固定操作列贴右可见 | T4 / T6 / T7 |

---

## T0 — F2 前置冒烟：OU 用户列表后端分页只读验证（STOP-03 前置）

**背景**：STOP-03 假设 OU 页用户表后端支持 `current/pageSize` 服务端分页。规划期已静态预核（结论见下），执行器需复核静态证据并做一次运行时冒烟。

**已完成的静态预核（planner 2026-10-24）**：
- 前端：`ruoyi-react-frontend/src/lib/adDomainApi.ts:377-381` — `getADUserList` → `post("/ad-domain/users/list", withDefaultPagination(params))`。
- 后端 handler：`internal/api/v1/system/ad_domain_handler.go:417-430` `ListUsers` — 绑定 `ADUserListRequest`（定义于 `internal/services/addomain/service.go:55-61`，内嵌 `base.BaseListRequest` 含 current/pageSize）→ `h.setDefaultPagination(&req.Current, &req.PageSize)` → `response.Page(c, list, total, req.Current, req.PageSize)`。**服务端分页 + total 已支持**。
- 风险点：`internal/services/addomain/computer.go:83-87` 显示同类请求把 PageSize 截到 `constants.MaxListPageSize`（100）。**必须确认 users/list 是否同样有上限**：读 `GetADUserList` 实现与 `base.BaseListRequest` 的规范化逻辑，若上限=100 则前端 pageSize 选项不得超过 100。

**改动**：无代码改动（只读）。

**步骤**：
1. 复核上述 文件:行号 证据仍成立（grep `setDefaultPagination`、`response.Page` 于 `ad_domain_handler.go`）。
2. 读 `internal/services/addomain/` 下 `GetADUserList`（user 列表 service）确认无 PageSize 截断或确认上限值，记录到执行备注。
3. 运行时冒烟（dev 环境已起时）：用 Swagger（`http://localhost:9000/swagger/index.html`）或已登录前端 DevTools Network，对 `POST /system/ad-domain/users/list` 分别发 `{configId, current:1, pageSize:10}` 与 `{current:2, pageSize:10}`，确认返回 `data.total` 正确且两页数据不重叠。
4. **回退方案**（仅当冒烟证明后端不支持分页时触发）：STOP-03 降级为「pageSize 提升到实际上限 + 分页器显示真实 total」，并在 PLAN 执行记录中标注偏差、上报 planner。不要因此改后端代码。

**验收**：冒烟通过（或记录降级决策）；SC3 的实现前提成立。
**验证命令**：
```bash
grep -n "setDefaultPagination" internal/api/v1/system/ad_domain_handler.go
grep -rn "func.*GetADUserList" internal/services/addomain/
```

---

## T1 — STOP-01：usePagination showTotal 一行修复（51 页自动生效）

**文件**：`ruoyi-react-frontend/src/hooks/usePagination.ts`（:115-126 paginationProps；:122 为目标行）

**改动**：把 :122
```ts
showTotal: (t) => `共 ${Math.max(1, Math.ceil(t / pageSize))} 页`,
```
改为（唯一口径，见 UI-SPEC §1）：
```ts
showTotal: (t) => (t === 0 ? "" : `共 ${t} 条 / ${Math.ceil(t / pageSize)} 页`),
```
- 禁止 `Math.max(1, …)`：total=0 时返回空字符串（F4：验收为视觉项——空列表不渲染「共 1 页」）。
- 不改 hook 其他导出（`setTotal` 保留，T2 仍要用；其 API 重构归 Phase 65/66）。

**验收（SC1）**：任意走 `usePagination` 的列表页分页器显示「共 N 条 / M 页」；清空筛选得到 0 条结果时分页器无任何占位文案。

**验证**：
```bash
cd ruoyi-react-frontend
grep -n "Math.max" src/hooks/usePagination.ts   # 期望 0 命中
npm run type-check && npm run test
```
手工冒烟：打开任一 hook 消费页（如 系统管理→用户管理、网络→端口），核对文案；把某筛选条件设为无结果，确认无「共 1 页」。

---

## T2 — STOP-02：backups / discoveries total 回填 + 双写清理

**文件**：
- `ruoyi-react-frontend/src/pages/network/backups/index.tsx`（:79 解构、:659-667 onChange 双写）
- `ruoyi-react-frontend/src/pages/network/discoveries/index.tsx`（:34 解构、:354 pagination 透传）

**改动**：
1. **backups**：在列表数据返回处（`setDeviceGroups(...)` 同一成功分支）补 `setTotal(data.total ?? 0)`；:79 的 `setTotal: _setTotal` 改回 `setTotal`。清理 :660-663 中手动的 `setCurrent(pagination.current ?? 1)` / `setPageSize(pagination.pageSize ?? 10)` 两行（`usePagination` 已通过 `paginationProps.onChange` 内部管理，双写是幽灵状态源）；`onChange` 保留 `handleBackupSortChange` 与排序解析逻辑。
2. **discoveries**：同法在 `useDiscoveryData` 返回处调用 `setTotal(result.data?.total ?? 0)`（:66 现有 `total: _total` 弃用值一并清理）；:34 解构改名。
3. 两页因走 paginationProps，自动获得 T1 的新 showTotal（SC1 同步满足）。

**验收（SC2 + SC1）**：backups / discoveries 翻到第 2、3 页均有数据且分页器 total 与后端一致；双写清理后翻页行为无回归（排序点击仍生效）。

**验证**：
```bash
cd ruoyi-react-frontend
grep -n "_setTotal" src/pages/network -r        # 期望 0 命中
npm run type-check
npx vitest run src/pages/network                # 若有相关测试
```
手工冒烟：网络→备份管理 / 设备发现，翻页、改 pageSize、点排序，观察 Network 面板请求参数 current/pageSize 正确、响应 total 回填。

---

## T3 — STOP-03：OU 页用户表改服务端分页

**文件**：`ruoyi-react-frontend/src/pages/ad-domain/ous/index.tsx`（:124-135 `loadUsers` 固定 `current:1, pageSize:100`；:535-542 users 表 `pagination={{ pageSize: 50, size: "small" }}` 前端分页）

**改动**：
1. 引入 `usePagination`（本页独立实例，命名如 `userPagination`），请求参数改为 `{ ...filters, current, pageSize }`（来自 hook state），pageSize 默认沿用 50。
2. 数据返回处 `userSetTotal(res.data?.total ?? 0)`。
3. :539 改为 `pagination={userPaginationProps}`（获得 total/onChange/showTotal，自动带 SC1 文案）。
4. 切换 pageSize 时页码重置第 1 页（usePagination 内部行为，验证即可）；切换 OU / configId 时 `userSetCurrent(1)` 再拉取。
5. 前端 users state 不再做客户端切片（dataSource 直出接口 list）。

**验收（SC3 + SC1）**：选一个用户数 >100 的 OU，第 3 页及以后数据可达且与后端一致；分页器显示「共 N 条 / M 页」。

**验证**：
```bash
cd ruoyi-react-frontend
grep -n "pageSize: 100" src/pages/ad-domain/ous/index.tsx   # 期望 0 命中
npm run type-check
```
手工冒烟：AD 域管理→OU，选大 OU 查看用户，翻到第 3 页+；Network 面板确认请求带动态 current/pageSize。
**注意**：`ous/index_with_dept.tsx` 是死文件（仅测试引用），**不修改**。

---

## T4 — STOP-05：批量删除二次确认（11 处清单 + 复核上报）

**契约**：UI-SPEC §5。首选替换/接入既有 `src/components/shared/BatchDeleteButton.tsx`（内部 Popconfirm + `批量删除 (N)` + danger ghost，组件不改）；不便替换按钮的最小止血用 `Modal.confirm`：标题 `确认批量删除`、内容 ``将删除选中的 ${N} 项，删除后不可恢复，是否继续？``、`okText: "删除"`、`cancelText: "取消"`、`okButtonProps: { danger: true }`、确认中 loading（接 T5 同一防重规则）、成功 `已删除`。

**逐文件打勾清单（planner 已对照源码 re-verify，2026-10-24）**：

- [ ] system/user — `src/pages/system/user/index.tsx:544`（handleBatchDelete）+ `:745`（裸 onClick 按钮）
- [ ] system/config — `src/pages/system/config/index.tsx:189` + `:430`
- [ ] system/post — `src/pages/system/post/index.tsx:156` + `:367`
- [ ] system/dict 类型表 — `src/pages/system/dict/hooks/useDictActions.ts:114`（handleBatchDeleteType），调用点 `src/pages/system/dict/index.tsx:614`
- [ ] system/dict 数据表 — `useDictActions.ts:180`（handleBatchDeleteData），调用点 `dict/index.tsx:790`
- [ ] system/role — `src/pages/system/role/hooks/useRoleActions.ts:132`，调用点 `role/index.tsx:318`
- [ ] system/notice — `src/pages/system/notice/hooks/useNoticeData.ts:100`（handleBatchDelete），调用链 `notice/index.tsx:411` → `NoticeList.tsx:329`
- [ ] network/devices — `src/pages/network/devices/index.tsx:510` + `:843`（按钮另带内联 danger 样式 :842，接 BatchDeleteButton 后可去）
- [ ] network/credentials — `src/pages/network/credentials/index.tsx:188` + `:462`
- [ ] network/ports — `src/pages/network/ports/index.tsx:322` + `:697`
- [ ] network/mac — `src/pages/network/mac/index.tsx:226` + `:492`
- [ ] network/templates — 确认闸门下沉：`src/pages/network/templates/hooks/useTemplateModals.ts:129`（hook 内加 `Modal.confirm`，页面 `templates/index.tsx:320` 调用点不变）——此点确认逻辑必须落在 hook 内，保证"无确认调用"不可达

**已有确认、不要动**（对照组，防止重复包裹）：`duty/schedules/index.tsx:561`、`duty/management/components/ScheduleManagement.tsx:264`、`operations/assets/index.tsx:235`、`system/menu/hooks/useMenuActions.tsx:243`。

**复核上报项**：执行 T4 时全站跑
```bash
grep -rn "onClick={handleBatchDelete}" ruoyi-react-frontend/src/pages
grep -rn "批量删除" ruoyi-react-frontend/src/pages/operations --include=*.tsx -l
```
若发现契约清单之外仍有无确认批删（规划期已注意到 operations 侧 room-devices:642、dedicated-lines:679、info-points:970、buildings:669、workstations:855、server-rooms:543、floors 组件按钮等候选——UI-SPEC §5 清单未含），**先上报 planner 裁决是否纳入本 phase**，不得擅自扩 scope。

**验收（SC5a）**：清单内每个入口点击后必先出现确认（Popconfirm 或 Modal.confirm），文案含数量 N；取消不发请求；确认才调 batch API。

**验证**：
```bash
cd ruoyi-react-frontend
npm run type-check && npm run test
grep -rn "onClick={handleBatchDelete}" src/pages/network src/pages/system   # 期望 0 命中（templates 走 hook 除外）
```
手工冒烟：逐页多选 2 条 → 点批量删除 → 取消（无请求）→ 再点 → 确认（一次请求，列表刷新）。

---

## T5 — STOP-04：业务弹窗双击防重（逐文件打勾）

**契约**：UI-SPEC §4。每个含异步提交的 Modal（含 BaseEditModal 调用点）二选一：① `onOk` 返回 Promise（antd 自动 loading + 拦截重复点击）；② 受控 `confirmLoading={submitting}` + `onOk` 首行 `if (submitting) return`。**同时**：`cancelButtonProps={{ disabled: submitting }}`。禁止新建 hook。

**5.1 duty / workorder / ad-domain（对应 audit「27 个」，planner grep 复核为 25 个 `<Modal` 声明 + 封装 Modal 组件；执行器按此清单逐一打勾，行号为当前快照）**：

- [ ] `src/pages/ad-domain/ous/index.tsx:634`、`:660`（2 个）
- [ ] `src/pages/ad-domain/groups/index.tsx:558`、`:575`（2）
- [ ] `src/pages/ad-domain/computers/index.tsx:426`、`:518`（2）
- [ ] `src/pages/ad-domain/users/index.tsx:647`、`:684`、`:711`（3）
- [ ] `src/pages/ad-domain/configs/index.tsx:376`（1）
- [ ] `src/pages/ad-domain/configs/AccountPoolTab.tsx:418`、`:447`（2，含 Unlock）
- [ ] `src/pages/duty/schedules/index.tsx:594`、`:658`、`:719`（3）
- [ ] `src/pages/duty/pools/index.tsx:471`（1）
- [ ] `src/pages/duty/holidays/modals/EditModal.tsx:73`、`BatchModal.tsx:38`（2）
- [ ] `src/pages/duty/management/modals/ManualScheduleModal.tsx:34`、`BatchHolidayModal.tsx:45`、`GenerateScheduleModal.tsx:48`（3）
- [ ] `src/pages/workorder/orders/index.tsx:706`（1）
- [ ] `src/pages/workorder/periodic/templates/index.tsx:422`（1）
- [ ] `src/pages/workorder/categories/index.tsx:257`（1）

（合计 25 处声明；audit 快照 27 与当前源的差额来自多行 Modal 标签/重构，执行时以本清单为基线、遇到清单外的新写 Modal 一并按契约处理。）

**5.2 system（audit §3.3 15 处，均无 confirmLoading——已 grep 确认这些文件 0 命中）**：

- [ ] `src/pages/system/user/index.tsx:814`、`:922`（2）
- [ ] `src/pages/system/role/index.tsx:344`（1）
- [ ] `src/pages/system/menu/index.tsx:365`（1）
- [ ] `src/pages/system/config/index.tsx:485`（1）
- [ ] `src/pages/system/dict/index.tsx:849`、`:895`（2）
- [ ] `src/pages/system/notice/components/NoticeForm.tsx:122`（1）
- [ ] `src/pages/system/profile/index.tsx:321`、`:372`（2）
- [ ] 其余 system 清单执行时复核：apikeys `index.tsx:702`（LogsModal:225 只读可豁免）、post `index.tsx:416`（BaseEditModal 调用点核对 confirmLoading 透传）、dept `modals/EditModal.tsx:51`、captcha-background `modals/EditModal.tsx:24`（UploadModal:42 已有 confirmLoading 豁免）

**已达标勿动**（grep 确认已有 confirmLoading）：manager-orgs:342、manager-rules:407、email-config:431/515、api-config:414、settings/captcha-background.tsx:491、modals/UploadModal.tsx:42。

**5.3 operations（audit「23 弹窗 8 达标 → 15 缺失」候选清单，执行器逐点核对 confirmLoading 是否缺失）**：

- [ ] `src/pages/operations/dedicated-lines/index.tsx:725`（Modal）+ `:198-214`（handleSave 无 submitting）
- [ ] `src/pages/operations/room-devices/index.tsx:688`
- [ ] `src/pages/operations/server-rooms/index.tsx:589`
- [ ] `src/pages/operations/info-points/index.tsx:1018`
- [ ] `src/pages/operations/floors/components/FloorModal.tsx:80`
- [ ] `src/pages/operations/building-spaces/components/BuildingModal.tsx:73`
- [ ] `src/pages/operations/rpa/tasks/modals/EditModal.tsx:263`
- [ ] `src/pages/operations/rpa/tasks/modals/AIScriptEditor.tsx:127`
- [ ] `src/pages/operations/rpa/executions/ExecutionDetailModal.tsx:305`（只读详情可豁免，核对后标注）
- [ ] `src/pages/operations/workstations/modals/EditModal.tsx:141`（BaseEditModal 调用点，核对 confirmLoading/onOk Promise）
- [ ] WorkstationDeviceTable 写入弹窗（02 分片 :615-682 无 confirmLoading，执行时定位文件行号）

已达标勿动：buildings:726（geocoding.loading）、fix-suggestion:556、exceptions:522、RollbackModal:63、ExceptionRuleForm:138。

**验收（SC4）**：清单内每个弹窗在请求未返回前 ok 按钮 loading、不可重复触发、取消按钮禁用；DevTools 节流下双击只发 1 次请求。

**验证**：
```bash
cd ruoyi-react-frontend
npm run type-check && npm run test
# 抽查（不要求全 0 命中，只验证清单文件全部出现 confirmLoading 或 onOk: async 返回 Promise）
grep -rn "confirmLoading" src/pages/ad-domain src/pages/duty src/pages/workorder | wc -l
```
手工冒烟（抽样 ≥6 个弹窗，覆盖三域各 2）：打开新增弹窗 → DevTools Network 节流 Slow 3G → 双击确定 → 断言仅 1 条请求、无重复记录；提交中点取消无效。

---

## T6 — STOP-06：假功能入口处置（3 处）

**文件与改动**：

1. `src/pages/operations/assets/index.tsx:202-205` — `handleEdit` 目前 `message.info("编辑功能待实现")`。处置：**隐藏编辑按钮**（columns 定义中操作列的编辑 ActionButton 条件移除或注释），保留 `handleEdit` 死代码可一并删除，处加 `// TODO(P67): 资产编辑表单接入`。禁止保留「待实现」文案（含 :204 的 eslint-disable 注释随删除清理）。
2. `src/pages/operations/assets/index.tsx:687-689` — 「下载模板」按钮现与「导入」同指 `setImportVisible(true)`。处置：改为真实下载 `excelApi.downloadTemplate("asset")`（`import { excelApi } from "@/lib/opsApi"`，包 loading 态，失败走 `handleApiError`）。
3. `src/components/reconciliation/ReconciliationDrawer.tsx:67-70` — `timeline` / `rules` 两个 `useState([])` 永久空态。处置：Tab items 改为条件渲染——无数据源的「历史变更」「例外规则」Tab 不输出；处加 `// TODO(P67/P68): 接入 useReconciliationTimeline / 例外规则接口`。注意 :74 注释表明原 Plan 02 接入计划，隐藏不删除数据结构。

**验收（SC5b）**：资产列表无编辑按钮或点击有真实行为；「下载模板」点击真实下载 xlsx；对账 Drawer 不再出现永久空 Tab。

**验证**：
```bash
cd ruoyi-react-frontend
grep -rn "待实现" src/pages/operations --include=*.tsx    # 期望 0 命中（本 phase 范围内）
npm run type-check
```
手工冒烟：资产管理页核对按钮与下载；打开任一资产的对账 Drawer 核对 Tab 列表。

---

## T7 — STOP-07：fixed 列与 scroll.x 成对（6 处）

**契约**：UI-SPEC §7。有列宽合计用像素，否则 `scroll={{ x: "max-content" }}`；不删 fixed 声明；不给 12 个无 scroll 的非 fixed 表全量补（归 Phase 64/67）。

**已定位 5 处（planner 复核行号）**：
- [ ] `src/pages/system/user/index.tsx:272`（`fixed: "right" as const`）所在表补 scroll.x
- [ ] `src/pages/system/config/index.tsx:291` 所在表补
- [ ] `src/pages/system/dict/index.tsx:129`（字典类型表）所在表补
- [ ] `src/pages/system/dict/index.tsx:231`（字典数据表）所在表补
- [ ] `src/pages/system/captcha-background/columns.tsx:96` 所在表（表声明在 `captcha-background/index.tsx`，补在 Table 处）

**第 6 处（network/monitor 侧同类命中，规划期未定位到唯一文件）**：执行时运行
```bash
cd ruoyi-react-frontend
grep -rn 'fixed:\s*"right"' src/pages/network src/pages/monitor src/pages/ad-domain src/pages/workorder src/pages/duty --include=*.tsx
# 对每个命中文件检查其 <Table 是否带 scroll={{ x: ...
```
找到第一个「fixed 无 scroll.x」的表即第 6 处；若发现多于 1 处，一并修（仍限 STOP-07 症状面：仅 fixed 表），并在执行记录中列全。

**验收（SC5c）**：6 个表在窄视口横向滚动时操作列始终贴右可见、可点击。

**验证**：
```bash
cd ruoyi-react-frontend
npm run type-check
```
手工冒烟：用户管理 / 参数配置 / 字典（两表）/ 验证码背景 + 第 6 处页面，浏览器窗口缩窄触发横向滚动，观察操作列。

---

## T8 — 文档收口：UI-SPEC Checker Sign-Off 更新

**文件**：`.planning/phases/63-p0/63-UI-SPEC.md`（:237-245）

**改动**：将 7 个旧维度名（Copywriting / Visuals / Color / Typography / Spacing / Registry Safety / Inventory Provenance）更新为标准 7 维并补 Approval 行：

```markdown
## Checker Sign-Off

- [x] Dimension 1 Layout & Structure: PASS
- [x] Dimension 2 Typography: PASS
- [x] Dimension 3 Color: PASS
- [x] Dimension 4 Spacing: PASS
- [x] Dimension 5 Copywriting: PASS
- [x] Dimension 6 Interaction States: PASS
- [x] Dimension 7 Accessibility: PASS

Approval: approved
```

**验收**：维度名与 7-dimension checker 标准一致；Approval 行存在。

---

## 执行顺序与依赖

```
T0（F2 冒烟，只读） ──→ T3（依赖 T0 结论）
T1 ──→ T2 / T3（共用新 showTotal 口径）
T4、T5 相互独立，可与 T1-T3 并行（不同文件面）
T6、T7、T8 独立收尾
```

建议提交粒度（不 commit，仅逻辑分组供最终 review）：①T1+T2+T3（分页可达）②T4+T5（安全防重）③T6+T7+T8（入口/滚动/文档）。

## Verification（Phase 收口总闸）

```bash
cd ruoyi-react-frontend
npm run type-check        # 必须通过
npm run lint              # 必须通过（含既有规则；本 phase 不新增 lint 规则）
npm run test              # 必须通过
npm run build             # 建议通过（防 vite/TS 边角）
grep -rn "Math.max" src/hooks/usePagination.ts                    # 0 命中
grep -rn "_setTotal" src/pages/network                            # 0 命中
grep -rn "pageSize: 100" src/pages/ad-domain/ous/index.tsx        # 0 命中
grep -rn "待实现" src/pages/operations --include=*.tsx            # 0 命中
```

**手工冒烟总清单（逐 STOP）**：
1. STOP-01：任一列表页显示「共 N 条 / M 页」；空结果无占位文案。
2. STOP-02：备份管理 / 设备发现翻页第 2、3 页有数据，total 正确；排序无回归。
3. STOP-03：OU 用户表翻过 100 条；请求动态 current/pageSize。
4. STOP-04：三域各抽 2 个弹窗，节流下双击仅 1 次请求；提交中取消禁用。
5. STOP-05：清单 11 处逐一点击：先确认、含数量、取消不发请求。
6. STOP-06：资产编辑入口不存在；下载模板真实下载；对账 Drawer 无空 Tab。
7. STOP-07：6 个表窄屏横滚操作列贴右可见。

**收口报告**：执行完成后在 `.planning/phases/63-p0/` 追加执行记录（各打勾清单勾选结果、T0 冒烟结论、T4 复核上报项的裁决、STOP-07 第 6 处定位结果），仍不 commit。
