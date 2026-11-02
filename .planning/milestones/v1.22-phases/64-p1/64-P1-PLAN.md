# Phase 64 实施计划 — P1 修复：反馈与可访问性（FIX-01..05）

> 里程碑：v1.22「前端公用组件整治」 · 目标代码：`xingran-react-frontend/`（React 19 + TS 5.9 + AntD 6.1 + Zustand）
> 契约：`64-UI-SPEC.md`（APPROVED，§1–§5 + Probe Coverage 29/29）· 需求：`.planning/REQUIREMENTS.md` FIX-01..05 · 路线图：`ROADMAP.md` Phase 64
> 行号基准：2026-10-24（UI-SPEC 抽样验证基线，Phase 63 落地后可能漂移）。执行前必须按各任务给出的「特征串 grep」重新定位，语义不变。
> 结构先例：`63-P0-PLAN.md`（全局约束 / SC 映射 / 逐任务打勾 / 收口总闸）；63 落地现状见 `63-P0-SUMMARY.md`。

## 0. 全局约束（每个任务都必须遵守）

1. **零新组件/公共件**：本 phase 新建组件数量为 0（UI-SPEC Registry Safety 明确禁止）。错误出口复用既有 `handleApiError`（`src/utils/errorHandler.ts:136`），排序复用既有 `useServerSort` + `createSorterMeta`；对既有共享组件只允许**增量化修改**（`ActionButtons` 增强 aria 输出、`BaseEditModal` 收紧默认值 + 新增可选 prop），不删除既有 prop。
2. **两段式提示（FIX-02）**：本 phase 只做存量修补——关死 `maskClosable` + `keyboard`，**不做** `isFieldsTouched()` 脏检查、不做关闭确认弹窗（下沉 Phase 66 FormModal）。FIX-01 loading 契约下沉 Phase 66 PageTable，本 phase 不提前引入封装层。
3. **排序字段名严格对齐后端白名单，不得自造**：cache 7 字段 `key/type/location/ttl/size/createdAt/updatedAt`（`internal/services/monitor/cache_service.go:237-246` `cacheAllowedSortFields`）；job 主表与日志表 4 字段 `jobName/jobGroup/status/createdAt`（`internal/services/scheduler/job_service.go:300` / `job_log_service.go:117`）。白名单外列一律移除 sorter，**禁止**给后端不支持的字段加白名单或保留前端内存排序。
4. **纯前端 + 范围纪律**：不改任何 Go 代码；不触碰 Out of Scope（后端改造、巨型文件拆分、业务组件重设计、i18n）；不修 FIX 之外的审查问题（执行中发现清单外问题→上报 Lead 裁决，不扩散）。FIX-04 只收敛 network+monitor 域 52 处，其余域 235 处归 Phase 67，不触碰。
5. **不新增静态 `message.*`**（63 硬约束延续）：成功提示沿用各页既有通道（组件内 `App.useApp()`）；错误提示只经 `handleApiError`。
6. **禁用骨架屏/整页 Spin**：加载反馈唯一口径是 Table 原生 `loading` 蒙层（UI-SPEC §1）。
7. **不 commit**：每任务完成仅运行验证命令并汇报，由 Lead 审核后提交。

## 0.1 成功标准映射（ROADMAP.md Phase 64 五条）

| # | 成功标准 | 对应任务 |
|---|---------|---------|
| SC1 (FIX-01) | network/monitor 域列表页数据加载期间表格显示 loading 而非空白 | T2（7 处内嵌表 + 复扫兜底） |
| SC2 (FIX-02) | 含未保存修改的表单弹窗误点遮罩/ESC 不关闭、内容不丢 | T1（BaseEditModal 默认值）+ T3（22 个业务 Modal） |
| SC3 (FIX-03) | cache/job 分页表按列排序得到全量正确结果（服务端排序） | T4（cache 7 列）+ T5（job 主表 4 列 + jobLog） |
| SC4 (FIX-04) | 请求失败有可见错误提示，`console.*` 静默吞错收敛到统一出口 | T6（network 27 处）+ T7（monitor 20 + components 5 处） |
| SC5 (FIX-05) | 读屏器用户能通过可访问名识别每个图标按钮 | T8（ActionButtons + 散置 icon-only 按钮） |

---

## T1 — tracer 切片：BaseEditModal 默认值收紧（FIX-02 核心，一处改、全片生效、grep 可验收）

**文件**：`xingran-react-frontend/src/components/modal/BaseEditModal.tsx`（:44 默认 `maskClosable = true`；:30/:58 为其余既有 `maskClosable` 声明）

**为什么 tracer-first**：一处默认值修改 + 一个新增可选 prop，22 个业务 Modal 消费者自动收紧；验收纯 grep，能最快打通「契约 → 改动 → 静态验收」全链路，验证后再扩展 T3-T8。

**改动**：
1. `:44` 默认值 `maskClosable = true` 改为 `maskClosable = false`。
2. 新增 `keyboard` prop（可选，类型 `boolean | undefined`，默认 `false`）并透传给内部 `<Modal keyboard={keyboard}>`——所有消费者自动获得「点遮罩/按 ESC 均不关闭」。
3. 对外 props 增量化：不删除/不改签名既有 prop；个别确需可关的只读详情类调用点可显式传 `maskClosable`/`keyboard` 为 `true` 并注释原因（执行时逐个消费者 grep 复核，默认不允许）。

**read_first**：`64-UI-SPEC.md` §2；`BaseEditModal.tsx` 全文；63-P0-SUMMARY「Verifier B1 修复」节（该组件已有 `cancelButtonDisabled` 增量 prop 先例）；`grep -rn "BaseEditModal" src --include=*.tsx -l` 消费者清单。

**验收（SC2 部分）**：BaseEditModal 消费者（含 post、workstations 等）打开表单弹窗后点击遮罩与按 ESC 均不关闭；既有测试全绿。

**验证**：
```bash
cd xingran-react-frontend
grep -n "maskClosable" src/components/modal/BaseEditModal.tsx   # 期望默认 false
grep -n "keyboard" src/components/modal/BaseEditModal.tsx       # 期望存在且默认 false 并透传
npm run type-check && npm run test -- --run
```

**fails_when**：默认值改为 false 后任一既有 BaseEditModal 消费者测试失败（说明有测试断言遮罩可关）→ 上报 Lead 裁决（预期无此情况）；出现删除既有 prop / 改签名 → 违反增量纪律，回退。

---

## T2 — FIX-01：7 处内嵌表补 Table loading（契约规则优先于清单）

**文件**（扫描时点 2026-10-24 的 7 处 Modal/Drawer 内嵌表，行号执行前 grep 重定位）：
- `src/pages/network/command/modals/DetailDrawer.tsx`
- `src/pages/network/command/modals/DispatchModal.tsx`
- `src/pages/network/discoveries/modals/ResultModal.tsx`
- `src/pages/network/executions/modals/DetailDrawer.tsx`
- `src/pages/network/executions/modals/ExecuteModal.tsx`
- `src/pages/network/templates/modals/VariablesModal.tsx`
- `src/components/network/port-write/BulkWriteDrawer.tsx`

**改动**：
1. 逐表接 `loading` prop（统一 boolean，禁骨架屏/Spin 包裹）。数据来源：接 `useTableManager` 的取 hook 的 `loading`（`useTableManager.ts:33/:47` 已暴露）；手写数据层的用既有 boolean state（无则就地加一个，请求 try/finally 维护）。
2. **复扫兜底（规则优先于清单）**：
```bash
cd xingran-react-frontend
grep -rl "<Table" src/pages/network src/pages/monitor src/components/network
# 对每个命中文件检查：含 <Table 且全文件无 loading= 与 spread 注入 → 一并纳入本任务修复
```
复扫发现清单外的缺失表 → 一并修（仍限 FIX-01 症状面），在执行记录中列全；发现超出 network/monitor/components 域的缺失 → 只上报不修。
3. 补 loading 时若发现某表 `fixed` 失效（无 scroll.x），按 63 §7 规则成对补 `scroll={{ x: "max-content" }}` 并记录（UI-SPEC UI Considerations overflow 条目授权）。

**read_first**：`64-UI-SPEC.md` §1；`src/hooks/useTableManager.ts`（loading 出处）；7 个目标文件。

**验收（SC1）**：DevTools 节流 Slow 3G 下打开任一 network/monitor 内嵌表，表格区域显示 antd 转圈蒙层而非空白行区；loading 期间 data=[] 不渲染空态插画。

**验证**：
```bash
cd xingran-react-frontend
npm run type-check && npm run test -- --run
# 复扫：每个含 <Table 的目标域文件均能 grep 到 loading=
grep -rL "loading=" $(grep -rl "<Table" src/pages/network src/pages/monitor src/components/network --include=*.tsx) || echo "no missing"
```

**fails_when**：为省事用 `<Spin>` 包裹整表或引入 Skeleton → 违反契约，回退；某 Modal 数据非本组件发起且无法拿到 loading 态 → 上报 Lead，不得伪造恒 false 的 `loading={false}`。

---

## T3 — FIX-02 存量：network/monitor 22 个业务 Modal 显式补齐

**依赖**：T1（BaseEditModal 承载的弹窗走新默认值，不重复声明）。

**文件**：03 分片 1.3 口径 `<Modal>` 22 个 / 17 文件（`pages/network` + `pages/monitor`）。执行时以复扫为准：
```bash
cd xingran-react-frontend
grep -rl "<Modal" src/pages/network src/pages/monitor
```

**改动**：
1. 逐个非 BaseEditModal 的表单类 Modal 显式补 `maskClosable={false} keyboard={false}`。
2. BaseEditModal 消费者不重复声明（T1 默认值已覆盖），在执行记录中列明走默认值的文件清单。
3. 已合规先例不动：`components/captcha/CaptchaModal.tsx:116`、`components/network/port-write/BulkWriteDrawer.tsx:238`（`maskClosable={!isExecuting}` 条件写法合法，保留；但补 `keyboard={false}` 需核对该文件现状，若已声明则不动）。
4. 非表单类纯详情 Modal 不强制，但补齐不禁止；`Modal.confirm` 函数式不在此列。
5. **不做** `isFieldsTouched()` 脏检查、不加关闭确认弹窗（两段式边界，Phase 66 落地）。

**read_first**：`64-UI-SPEC.md` §2（决策三点理由）；T1 执行记录（BaseEditModal 覆盖清单）。

**验收（SC2）**：任一表单弹窗填入内容后点遮罩或按 ESC，弹窗保持打开、内容完好；只能经「取消/关闭」按钮关闭。

**验证**：
```bash
cd xingran-react-frontend
# 复扫去重后：非 BaseEditModal 的表单类 <Modal 声明处应全部命中 maskClosable={false}
grep -rn "maskClosable" src/pages/network src/pages/monitor | wc -l   # 期望 ≥ 复扫清单中非 BaseEditModal 表单 Modal 数
npm run type-check && npm run test -- --run
```

**fails_when**：出现 `Form.isFieldsTouched` / 关闭确认弹窗代码 → 越界到 Phase 66，回退；把 `Modal.confirm` 也加 maskClosable → 类别错误（函数式确认框语义就是可取消），回退。

---

## T4 — FIX-03a：cache 列表 7 列接服务端排序

**文件**：`xingran-react-frontend/src/pages/monitor/cache/index.tsx`（:76-132 七列 `createSorter` 内存排序；列表请求发起处）

**改动**：
1. 引入 `useServerSort`（`src/hooks/useServerSort.ts:89`）+ `createSorterMeta`（先例：`src/pages/monitor/logs/columns/loginColumns.tsx`、`src/pages/network/discoveries/index.tsx:37-54` 标准写法）。
2. 7 列 `key/type/location/ttl/size/createdAt/updatedAt` 全部改 `createSorterMeta`——**字段名逐一对齐后端 `cacheAllowedSortFields`（`internal/services/monitor/cache_service.go:237-246`），不得自造**。
3. 列表请求透传 `orderByColumn` / `isAsc`（后端 `internal/api/v1/monitor/cache_handler.go:144-145` 已透传，零后端改动）；URL 持久化由 `useServerSort` 内建，验证即可。
4. 排序请求复用表格 loading（T2 契约）；排序变更页码回 1（`useServerSort` 既有行为，验证确认，无则显式 setCurrent(1)）。

**read_first**：`64-UI-SPEC.md` §3（checker 复核：后端三处均已支持白名单排序）；`src/hooks/useServerSort.ts` 全文；`internal/services/monitor/cache_service.go:237-246` 白名单原文；先例 `pages/monitor/logs/columns/loginColumns.tsx`。

**验收（SC3 部分）**：cache 表点击任一列表头，请求带正确 `orderByColumn/isAsc`，跨页全量有序（后端 ORDER BY）。

**验证**：
```bash
cd xingran-react-frontend
grep -n "createSorter\b" src/pages/monitor/cache/index.tsx    # 期望 0 命中（内存排序移除）
grep -n "createSorterMeta" src/pages/monitor/cache/index.tsx  # 期望 ≥7（7 列）
npm run type-check && npm run test -- --run
```
手工冒烟（可并入 T5）：Network 面板确认排序请求参数与返回顺序一致；翻页后顺序保持。

**fails_when**：出现白名单外字段名（如 `keyPrefix`）→ 立即对齐白名单；后端返回 400/静默忽略某列 → 核对字段拼写与 `cacheAllowedSortFields` 原文，仍不符则上报（不得改后端）。

---

## T5 — FIX-03b：job 主表 4 列服务端排序 + 白名单外移除 sorter；jobLog 同法

**依赖**：T4（确立 useServerSort 接入范式后扩展）。

**文件**：
- `xingran-react-frontend/src/pages/monitor/job/columns/jobColumns.tsx`（:37-103，8 处 sorter）
- `xingran-react-frontend/src/pages/monitor/job/columns/jobLogColumns.tsx`（:19-53，5 处 sorter）
- job 列表/日志请求发起处（job 页面及 hooks）

**改动**：
1. **job 主表**：仅 `jobName/jobGroup/status/createdAt` 4 列接 `createSorterMeta`（对齐 `jobAllowedSortFields`，`job_service.go:300/:302`）；白名单外 `invokeTarget/cronExpression/concurrent/nextRunTime` 4 列**删除 `createSorter` 调用**，表头不再渲染排序箭头。
2. **jobLog 表**：`status` 列接 `createSorterMeta`；`startTime` 列映射 `orderByColumn: "createdAt"` 接入（`job_log_service.go:118-119` 无入参时默认即 `created_at DESC`，二者同义）；白名单外 `duration/jobMessage/exceptionInfo` 3 列删除 sorter。
3. 请求透传 `orderByColumn/isAsc`；**不做降级说明 caption**（服务端排序后歧义不存在，UI-SPEC §3 规则 4）。
4. `pages/monitor/server/index.tsx` 双轨排序是 P2 另一面，**不在 FIX-03 点名范围，不动**。

**read_first**：`64-UI-SPEC.md` §3；T4 执行记录；`internal/services/scheduler/job_service.go:300-302`、`job_log_service.go:117-119` 白名单原文；`useJobData` / 相关 hooks。

**验收（SC3）**：job 主表 4 列、jobLog 2 列（status/startTime）排序跨页全量正确；7 个白名单外列不再显示排序箭头，点击无排序行为。

**验证**：
```bash
cd xingran-react-frontend
grep -rn "createSorter\b" src/pages/monitor/job/            # 期望 0 命中
grep -c "createSorterMeta" src/pages/monitor/job/columns/jobColumns.tsx      # 期望 4
grep -c "createSorterMeta" src/pages/monitor/job/columns/jobLogColumns.tsx   # 期望 2（status + startTime→createdAt）
npm run type-check && npm run test -- --run
```

**fails_when**：给白名单外字段加 sorter 或加后端白名单 → Out of Scope，回退；jobLog 用 `startTime` 作 `orderByColumn` 直传（后端白名单无此字段，会被忽略）→ 必须映射 `createdAt`。

---

## T6 — FIX-04a：network 域 catch 三分支收敛（pages/network 27 处）

**文件**：`src/pages/network/**` 全部含 `console.*` 的非测试文件（27 处；含 03 分片 3.3 第 12 条点名的 port-write 4 弹窗、3.4 第 15 条校验分流缺失）。

**改动**：逐处按 UI-SPEC §4 唯一合法写法改造 catch 三分支：
```ts
catch (error) {
  if (isFormValidationError(error)) return;   // ① 校验错误静默（Form 自带行内提示）
  handleApiError(error, "动作名");              // ② 统一出口（内建 `${context}失败: ${message}` toast + 内部日志）
  // ③ 成功提示留在 try 内既有通道，不动
}
```
细则：
- `isFormValidationError` 从 `@/utils/errorHandler` import（先例 `src/pages/network/discoveries/index.tsx:26`）。
- **列表加载类失败优先置 error state 渲染 `ErrorAlertWithRetry`**（既有 11 处先例），toast 仅用于操作类失败；维持该页既有错误态结构，只把 console 出口换成正确分支。
- 删除的 `console.*` 不需要补日志——`handleApiError` 内部已 `console.error`；确有非用户可行动的纯调试分支可用 `console.warn` 保留但不得作为错误唯一出口。
- context 用各页既有动作名（`删除设备`、`派发命令` 等），不新造句式。
- 禁止在 `lib/api.ts` 拦截器层叠加 toast（防双重提示先例 `lib/api/networkApi.ts:158`）；不新建任何 toast helper。

**read_first**：`64-UI-SPEC.md` §4（契约细则表）；`src/utils/errorHandler.ts`（:15 isFormValidationError、:136 handleApiError）；`src/components/shared/ErrorAlertWithRetry.tsx`；03 分片 3.3 第 12 条、3.4 第 15 条、3.6 第 25 条。

**验收（SC4 部分）**：network 域任一操作失败出现「XX失败: 原因」toast；列表加载失败出现可重试错误条；表单校验失败不再弹 toast。

**验证**：
```bash
cd xingran-react-frontend
# FLAG（checker 指出 UI-SPEC 正则转义有误，此处为标准写法）：
grep -rEn "console\.(error|warn|log)" src/pages/network --include="*.ts" --include="*.tsx" \
  | grep -v "__tests__" | grep -v "\.test\."    # 期望 0 命中
grep -rn "isFormValidationError" src/pages/network | wc -l   # 期望 > 0（分流落地）
npm run type-check && npm run test -- --run
```

**fails_when**：出现第二个错误 toast helper 或拦截器层 toast → 违反契约，回退；为清 console 把校验错误也弹 toast（漏 isFormValidationError 前置）→ 修复；测试文件中的 console 不计入，不得改测试。

---

## T7 — FIX-04b：monitor 域 + components 收敛（pages/monitor 20 + components/network 与 CronSelector 5 处）

**依赖**：T6（三分支范式确立后同法扩展）。

**文件**：
- `src/pages/monitor/**` 非测试文件 20 处 `console.error`（03 分片 3.6 第 25 条已逐一复核：`server/index.tsx:92/:113`、`cache/index.tsx:214/:229/:251/:269/:291/:308/:354`、`logs/index.tsx:190`、`logs/hooks/useLogActions.tsx:63`、`logs/hooks/useLogData.ts:66/:100`、`job/hooks/useJobActions.ts:101/:115/:132/:149`、`job/hooks/useJobData.ts:76/:98/:117`——行号执行前重定位）。
- `src/components/network` + `src/components/CronSelector` 非测试文件 5 处。

**改动**：同 T6 三分支；monitor 侧多数为 hooks 内数据加载 catch → 优先置 error state（hook 已有 error 通道则复用，`useLogData.ts`/`useJobData.ts` 已返回数据的结构保留），操作类（useJobActions 等）走 `handleApiError`。**52 处之外的域（system/operations/duty/workorder/components 其他子目录等 235 处）一律不触碰，归 Phase 67。**

**read_first**：`64-UI-SPEC.md` §4；T6 执行记录（范式与边界先例）；03 分片 3.6 第 25 条 20 处行号清单。

**验收（SC4）**：monitor 任一操作失败（如缓存清空失败、job 删除失败）出现可见 toast；任一列表加载失败出现可重试错误条。

**验证**：
```bash
cd xingran-react-frontend
grep -rEn "console\.(error|warn|log)" src/pages/monitor src/components/network src/components/CronSelector \
  --include="*.ts" --include="*.tsx" | grep -v "__tests__" | grep -v "\.test\."   # 期望 0 命中
# 52 处总量核对（network+monitor 域口径，排除测试）：
grep -rEn "console\.(error|warn|log)" src/pages/network src/pages/monitor src/components/network src/components/CronSelector \
  --include="*.ts" --include="*.tsx" | grep -v "__tests__" | grep -v "\.test\." | wc -l   # 期望 0
npm run type-check && npm run test -- --run
```

**fails_when**：发现 grep 复扫命中数与 52/域内口径不符（63 落地后可能已有变化）→ 以复扫为准全清（仍限本域），执行记录列差额；顺手清理其余域 console → 越界，回退并上报。

---

## T8 — FIX-05：图标按钮可访问名（ActionButtons 一处修全片 + 散置逐个补）

**文件**：
- `xingran-react-frontend/src/components/shared/ActionButtons.tsx`（:126 仅有 `title`，aria 现状 0）
- `src/pages/network`、`src/pages/monitor`、`src/components/network` 工具栏/卡片散置 icon-only `<Button>`（刷新、清空、导出等）

**改动**：
1. **ActionButtons 内部增强**（改一处、network 全片行内按钮生效）：每个 icon-only 输出的 `<Button>` 加 `aria-label={label}` + `title={label}`（同文案）；带文字按钮补 `title` 即可。属既有组件增量化修改，不改 props 签名、不删既有行为；Phase 67 其他域接入后自动受益。
2. 散置 icon-only Button 逐个补 `aria-label` + `title`，文案 = 既有动作名（`刷新`/`清空缓存`/`导出`/`执行`/`详情` 等）——**必须是人类可读中文动作名，禁止 `btn-x` 类标识符**。
3. 不写 `role`（`<button>` 天然继承）、不动 `tabIndex`；**否决 IconButton 新封装**（UI-SPEC §5，避免与 ActionButtons 形成第二套行内封装）。
4. 其余域（system 51 处等）不在本 phase，归 Phase 67。

**read_first**：`64-UI-SPEC.md` §5（范围收敛三层策略 + 验收检查项）；`src/components/shared/ActionButtons.tsx` 全文；先例 `src/pages/system/user/index.tsx:737`（全片唯一 aria-label 先例）。

**验收（SC5）**：读屏器（NVDA/VoiceOver）聚焦 network/monitor 任一行内删除/详情按钮朗读动作名而非「按钮」；工具栏图标按钮同理。

**验证**：
```bash
cd xingran-react-frontend
# ActionButtons 内每个 icon-only Button 均有 aria-label（人工核对 + grep 辅助）
grep -n "aria-label" src/components/shared/ActionButtons.tsx        # 期望命中
grep -rEn "aria-label" src/pages/network src/pages/monitor src/components/network | wc -l   # 期望 > 0
npm run type-check && npm run test -- --run
```

**fails_when**：出现 `role=` / `tabIndex` 手写 → 违反最小充分集，回退；aria-label 用英文标识符或空串 → 违反文案契约，修复；ActionButtons 既有调用点因签名变化报错 → 增量纪律被破坏，回退。

---

## T9 — 收口：Phase 级验证总闸 + 64-P1-SUMMARY.md + verifier 交接

**依赖**：T1–T8 全部完成。

**文件**：
- 新建 `.planning/phases/64-p1/64-P1-SUMMARY.md`
- 更新 `.planning/REQUIREMENTS.md` Traceability 表（FIX-01..05 → Phase 64，Status 由 Lead 提交时定）

**改动**：
1. 跑收口总闸（见下），逐项记录结果。
2. 生成 `64-P1-SUMMARY.md`，结构对齐 `63-P0-SUMMARY.md`：逐任务完成状态表（含打勾清单结果）、偏离记录（复扫差额、D 类决策）、Verification 结果表（对照既有质量基线）、上报裁决项（范围外发现）、手工冒烟建议（留给 Lead/QA，含 NVDA 读屏抽样）。
3. **verifier 交接说明**：在 SUMMARY 末尾写明——验证基线（type-check 0 错误；测试基线 = 63 收口口径：3835 passed / 8 failed，其中 apiFactory invariants 3 + user render 4 为存量失败、1 个 dashboard flake，本 phase 不得新增失败）、各 FIX 的 grep 复扫命令、两段式边界（FIX-02 脏检查 / FIX-01 三态组件化归 Phase 66，其余域 console 与 aria 归 Phase 67——验证时不得判为缺失）。
4. **不 commit**，等 Lead 审核。

**验证（Phase 收口总闸）**：
```bash
cd xingran-react-frontend
npm run type-check        # 必须通过（0 错误）
npm run lint              # 对照基线 125 errors 不新增
npm run test -- --run     # 对照基线 8 failed 不新增（存量 7 + flake 1）
npm run build             # 建议通过
# 五条 FIX 复扫：
grep -rEn "console\.(error|warn|log)" src/pages/network src/pages/monitor src/components/network src/components/CronSelector --include="*.ts" --include="*.tsx" | grep -v "__tests__" | grep -v "\.test\."   # FIX-04：0 命中
grep -rn "createSorter\b" src/pages/monitor/cache src/pages/monitor/job   # FIX-03：0 命中
grep -n "maskClosable" src/components/modal/BaseEditModal.tsx             # FIX-02：默认 false
grep -rEn "aria-label" src/pages/network src/pages/monitor src/components/network | wc -l   # FIX-05：> 0
grep -rL "loading=" $(grep -rl "<Table" src/pages/network src/pages/monitor src/components/network --include=*.tsx)   # FIX-01：期望无输出
```

**手工冒烟总清单（留给 Lead/QA）**：
1. FIX-01：DevTools Slow 3G 打开 network/monitor 内嵌表与列表页 → 表格转圈蒙层。
2. FIX-02：表单弹窗填内容 → 点遮罩 / 按 ESC → 不关闭、内容在；取消按钮可关。
3. FIX-03：cache 点 7 列任一表头、job 点 4 列 → Network 面板 orderByColumn/isAsc 正确且跨页有序；job 白名单外 7 列无排序箭头。
4. FIX-04：断网或改错参数触发失败 → 「XX失败: 原因」toast；列表加载失败 → 可重试错误条；表单校验失败无 toast。
5. FIX-05：NVDA/VoiceOver 抽读行内与工具栏图标按钮 → 朗读中文动作名。

---

## 执行顺序与依赖

```
T1（tracer：BaseEditModal）──→ T3（22 个业务 Modal，依赖 T1 覆盖清单）
T2（FIX-01 loading，独立）
T4（cache 排序）──→ T5（job 排序，复用范式）
T6（network console）──→ T7（monitor+components console，复用范式）
T8（aria，独立）
T9（收口，依赖全部）
```
可并行组：{T2, T4, T6, T8} 相互独立（不同文件面），T1 完成后 {T3, T5, T7} 可与剩余独立任务并行。

## 估算

| 任务 | 估算 |
|------|------|
| T1 | 0.5h |
| T2 | 2h（7 处 + 复扫兜底） |
| T3 | 2h（~22 处逐个打勾） |
| T4 | 1.5h |
| T5 | 1.5h |
| T6 | 3h（27 处 + 错误态分支） |
| T7 | 2.5h（25 处） |
| T8 | 2h |
| T9 | 1h |
| **合计** | **约 16h（2 个工作日）** |

## 关键取舍（分解决策记录）

1. **tracer 选 FIX-02 BaseEditModal 默认值**：一处改 + 22 处自动受益 + 纯 grep 验收，最快打通全链路；先于FIX-01（需逐文件接数据层）与 FIX-04（需逐 catch 判断分流）。
2. **FIX-02 拆两任务**：T1（共享封装默认值）与 T3（业务 Modal 显式补齐）分离——T3 依赖 T1 的覆盖清单决定哪些不重复声明，且 BaseEditModal 消费者改动若混在一起难以归因回归。
3. **FIX-03 拆 cache / job 两任务**：cache 7 列全白名单（纯接入），job 需「接入 + 移除 sorter」双动作且 jobLog 有 startTime→createdAt 映射，风险形态不同；T5 复用 T4 范式降成本。
4. **FIX-04 按域拆两任务**：network（27，含 port-write 校验分流）与 monitor（25，多为 hooks 内数据加载，走 error state 分支）错误形态不同，且 monitor 侧行号清单已逐一复核，可独立打勾。
5. **验证命令全部修正为标准 grep 正则** `grep -rEn "console\.(error|warn|log)"`（checker FLAG：UI-SPEC §4 内的转义写法有误，以本计划为准）；测试/构建基线显式写入 T9（lint 125 errors、test 8 failed 存量口径），避免 verifier 误判存量失败为本 phase 回归。
6. **严格两段式**：计划在 T3/契约处显式禁止 isFieldsTouched 脏检查与关闭确认弹窗（fails_when 反向守护），防止 executor 顺手把 Phase 66 的活干了。
7. **排序字段以 grep 白名单原文为准**：T4/T5 read_first 均指向后端白名单源码行，验收用 `createSorterMeta` 计数（cache≥7、job=4、jobLog=2），不自造字段。
8. **复扫兜底条文化**：FIX-01/02/04 三个清单型任务全部写明「规则优先于清单 + 复扫命令 + 差额上报不扩散」，对冲 63 落地后的行号漂移。
