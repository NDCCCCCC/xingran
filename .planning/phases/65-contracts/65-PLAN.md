---
phase: "65"
slug: "65-contracts"
type: plan
created: "2026-10-25"
planner: gsd-planner
---

# Phase 65 执行计划 — 基础契约组件（默认值与危险操作唯一入口）

**目标**：交付 4 个新公共件（AppInput / AppSelect / ConfirmDelete / confirmDelete）+ 1 套 lint 拦截规则，建立新代码的输入默认值与危险删除唯一入口契约，为 Phase 66 PageTable/FormModal 提供直接前置。

**权威契约**：`.planning/phases/65-contracts/65-UI-SPEC.md`（aa30e13f）。任务书与本 PLAN 冲突时以 UI-SPEC 为准。

**验证基线**（64-P1-SUMMARY.md:45-47,81）：
- `npm run type-check` → 0 错误
- `npm run test -- --run` → 3843 passed / 0 failed（64 收口实测全绿；历史基线 8 failed 为 apiFactory 3 + user render 4 + dashboard flake 1，64 落地后已全绿）
- `npm run lint` → 全仓 125 errors 存量；**禁跑全量 lint 判闸**，验证一律限定文件范围或用 `--no-error-on-unmatched-pattern` 数新增

**硬边界**（locked，违反即任务失败）：
- 抽契约不抽业务：不抽资产选择器/部门树选择器等业务语义组件
- AppTextarea/AppPassword 不扩面；`Input.TextArea|Search|Password` 不在 lint 拦截面
- RemoteSelect 仅锁契约（UI-SPEC §2.3），不写实现，归 Phase 66
- 存量 ~45 处 Modal.confirm / 18 处 Popconfirm 不迁移（归 Phase 67）
- 不新建 useConfirmAction / withConfirmSubmitting 第三路径
- UI 文案/颜色全走既有 token 与 antd 语义，禁止裸 hex
- 全程不 commit

**行号锚点纪律**：本 PLAN 全部 `文件:行号` 锚点（eslint.config.js:93/:113-118/:119、user/index.tsx:547-560、buttonStyles.tsx:68/:77、ActionButtons.tsx:113、errorHandler.ts:136、useTemplateModals.ts:129 等）为 2026-10-25 快照，**漂移需重定位**——以文件内语义锚（规则名/函数名/文案字符串）为准重新 grep，不得按失效行号盲改。

## 修订记录

- **2026-10-25 gsd-plan-checker REVISED**：① 事实注记 2 的「22 个页面文件含 handleBatchDelete」实测修正为 29 个源文件（漏 menu 对照组）；② 补 UI-SPEC Component Inventory 要求的 buttonStyles 常量升级动作入 T5（PLAN 原遗漏）；③ T2 退化方案表述修正（flat config 同一 rules 对象内同名键互相覆盖，须拆独立 config 块）+ verification 移除 flat config 下无效的 `--rules-dir`、数 error 口径改为数 `✖`/error 计数行；④ T3 verification 的 `grep -v 免除名单文件` 占位伪命令改为可行写法；⑤ T8 SC3 判定口径澄清（grep 命中须对照 T2 豁免清单，非字面「仅 confirmDelete.ts」）。

## 事实核查注记（planner 实测，executor 须知）

1. **`src/components/ui/` 目录与 `src/components/index.ts` 均不存在**——本 phase 全新建（与 UI-SPEC §1 行为契约 3 一致）；`@` alias 已在 `vite.config.ts:126` 配好。
2. **BatchDeleteButton 现网 0 业务消费点**（grep 全仓仅组件自身文件）——「11 处批删迁移到新入口」的 63 落地形态是**页面 handler 内包 `Modal.confirm`**（63 SUMMARY:16/:26 D3），不是 BatchDeleteButton 组件。批删迁移的落点是 `handleBatchDelete` 函数体，**实测 29 个源文件**含该函数（system user/config/post/dict/role/notice/menu + network devices/ports/mac/templates/credentials + operations buildings/floors/workstations/info-points/server-rooms/assets/room-devices/dedicated-lines + duty/schedules + workorder/orders，均在 `src/pages/` 下，已排除测试文件；其中 system/menu 为 63 T4 对照组、operations/assets 另有「确认批量删除」文案但不在 63 收敛清单——两者均归 Phase 67）（system user/config/post/dict/role/notice + network devices/ports/mac/templates + operations buildings/floors/workstations/info-points/server-rooms/assets/room-devices/dedicated-lines + duty/schedules + workorder/orders），其中 **63 T4 实际收敛的 11 处**是 UI-SPEC §5 表格列出的模块：system user/config/post/dict×2/role/notice + network devices/credentials/ports/mac/templates。executor 执行 T7 时以 UI-SPEC §5 列表为准（grep `确认批量删除` 定位），**不扩面到其余 handleBatchDelete 文件**（operations/duty/workorder 各页归 Phase 67）。
3. **createDeleteAction 现网 0 外部消费**（grep 全仓仅 ActionButtons.tsx 自身）——内部切换为渲染 ConfirmDelete 是零风险的内部等价替换；ActionButtons 消费页自动获得合规。
4. **eslint.config.js 结构**：`no-restricted-syntax` 在 `eslint.config.js:93-100`（现仅内网 IP selector）；测试 override 在 `eslint.config.js:113-118`，**关闭的是整条 no-restricted-syntax**——新 selector 追加后测试文件天然全豁免（符合预期）；`no-restricted-imports` 现网无既有配置，无冲突。规则结构为 flat config 数组，新增 override 追加在测试 override 之后、prettier（`:119`）之前。
5. **文案常量现状**：`buttonStyles.tsx:68` DELETE_CONFIRM / `:77` BATCH_DELETE_CONFIRM 均不含 N、无不可恢复警示。BatchDeleteButton（`BatchDeleteButton.tsx:44`）当前用 Popconfirm——lint Popconfirm selector 上线前**必须**先完成 T5 内部切换，否则封装层自身报 error（封装层目录 override 已兜底，但顺序上仍先切实现）。

---

## T1 — Tracer 切片：AppInput + AppSelect 封装 + 单测 + 现有页面单点接入

> **先行任务**：全链路（组件 → import → 页面渲染 → type-check → test）走通后才展开 T2-T7。若 T1 阻塞，后续任务不得开始。

**exact files**：
- 新建 `ruoyi-react-frontend/src/components/ui/AppInput.tsx`（按 UI-SPEC §1 API 表逐字实现：`allowClear = true` 唯一默认值 + rest 全透传）
- 新建 `ruoyi-react-frontend/src/components/ui/AppSelect.tsx`（按 UI-SPEC §2：`allowClear = true` + `showSearch ?? count >= 8` + `optionFilterProp ?? "label"`，spread 顺序保证显式 props 覆盖默认）
- 新建 `ruoyi-react-frontend/src/components/ui/index.ts`（导出 AppInput / AppSelect；含 `export { AppInput as Input }`、`export { AppSelect as Select }` 别名供 P67 降险，注释注明新代码必须具名）
- 新建 `ruoyi-react-frontend/src/components/ui/__tests__/AppInput.test.tsx`、`AppSelect.test.tsx`
- 修改 `ruoyi-react-frontend/src/pages/system/post/index.tsx`（tracer 接入点：选 1-2 个搜索区 Input/Select 单点替换为 AppInput/AppSelect——post 页搜索区小、有既有测试，回归面最小）

**read_first**：
- `.planning/phases/65-contracts/65-UI-SPEC.md` §1/§2/§2.3（API 逐字契约）
- `ruoyi-react-frontend/src/pages/system/post/index.tsx`（接入点现状）
- 任一既有组件测试（如 `src/components/shared/__tests__/`）对齐测试写法（render/renderHook 惯例）

**acceptance_criteria**：
1. AppInput：不传 props 渲染出的 input 具备可清除（allowClear）；显式 `allowClear={false}` 生效；`value/onChange/placeholder/ref` 透传行为与裸 Input 一致（fireEvent 断言 onChange 收到 e.target.value）
2. AppSelect：options.length < 8 时 showSearch=false，≥8 时自动 true；显式 `showSearch={false}` 覆盖自动值；默认 `optionFilterProp="label"`；显式传 `filterOption` 函数时透传生效
3. post 页搜索区替换后渲染正常（type-check 0 错误 + post 页既有测试通过）
4. 全链路验证命令全部通过

**fails_when**：封装吞掉/改名任何 rest prop；AppSelect spread 顺序使显式 props 无法覆盖默认值；别名导出缺 `Input`/`Select` 形态

**verification**：
```bash
cd ruoyi-react-frontend && npm run type-check
npx vitest run src/components/ui/__tests__/ src/pages/system/post/
npm run test -- --run   # 3843+ passed / 0 failed
```
预估：3h

## T2 — lint 拦截规则落地 + 存量豁免清单（§3 方案 a）

**exact files**：
- 修改 `ruoyi-react-frontend/eslint.config.js`：
  - `:93` 既有 `no-restricted-syntax` 数组追加 UI-SPEC §3 草案的 4 个 selector（Input / Select / Modal.confirm / Popconfirm，message 逐字用草案文案）
  - 新增 `no-restricted-imports`（patterns：`antd/Input`、`antd/Input/*`、`antd/Select`、`antd/Select/*`）
  - 新增封装层目录 override：`files: ["src/components/ui/**", "src/components/shared/**"]`，关闭上述新增 selector 对应条目（**只关新增的 4 selector 对应的 no-restricted-syntax 条目语义**——因规则是单条数组，实际做法是把 4 selector 放进独立的 `no-restricted-syntax` 命名组不可行（flat config 同名覆盖），故采用：**封装层 override 内把 `no-restricted-syntax` 重新声明为仅含既有内网 IP selector 的数组**，实现「关闭新增条目、保留既有条目」；内网 IP selector 的字面量同步复制，避免漂移则加注释双向指向）
  - 新增存量豁免 override：`files: [<存量命中文件清单>]` + 首行注释 `// P67 收敛后移除`，同样重声明数组只含内网 IP selector。清单**脚本生成**：跑 `npx eslint --no-error-on-unmatched-pattern 'src/**/*.{ts,tsx}'` 数出新增 4 selector 的命中文件全集，逐个列入
- 新建 `ruoyi-react-frontend/scripts/gen-p65-exemptions.mjs`（清单生成脚本，T2 与 T7 收口复跑）

**read_first**：
- `ruoyi-react-frontend/eslint.config.js` 全文（127 行，注意 flat config 结构与 prettier 必须最后）
- UI-SPEC §3 草案与边界说明 1-5

**acceptance_criteria**：
1. 新写一个含裸 `<Input>` 的业务测试文件外 tsx → lint 报 error 且 message 指向 AppInput
2. `src/components/ui/**`、`src/components/shared/**` 内的裸 `<Input>`/`<Select>`/`<Popconfirm>`/`Modal.confirm` 不报错（封装层豁免生效）
3. 测试文件（`__tests__`/`*.test.*`）内裸组件不报错（既有测试 override 天然覆盖新 selector）
4. 全量定向 lint：**新增 errors = 0**（存量 125 不动），存量命中文件全部被豁免清单覆盖
5. `no-restricted-imports` 拦截 `import { Input } from "antd/Input"` 类侧门

**fails_when**：封装层 override 把既有内网 IP 规则一并关闭（重声明数组必须保留内网 IP selector）；豁免清单遗漏文件导致定向 lint 有新增 error；override 位置放在 prettier 之后

**verification**：
```bash
cd ruoyi-react-frontend
# 全仓定向数新增（统计 error 总数与 125 基线对比；flat config 下不加 --rules-dir，eslint-rules/** 已在 ignores）
npx eslint src 2>&1 | tail -3   # 总 error 数 = 125 存量 + 0 新增
# 生成/校验豁免清单后复跑（对比豁免前后 error 数不变，即存量命中全部被豁免）
node scripts/gen-p65-exemptions.mjs && npx eslint src 2>&1 | tail -3
# 单文件试拦（临时 fixture，验证后删除）
npx eslint src/components/__lint_probe__.tsx
```
预估：4h

**实现注记**：若 flat config 同名规则覆盖语义导致「目录 override 重声明数组」方案在 executor 实测中过重（内网 IP 字面量复制易漂移），可退化为等价方案：**把 4 个新 selector 放进一个独立的 config 对象块**（`{ rules: { "no-restricted-syntax": ["error", …4 selectors] } }`；注意不能与既有块共用同一 rules 对象——对象字面量同名键后者覆盖前者），封装层 override 用 `"no-restricted-syntax": "off"` 整条关闭后，再追加一个更靠后的 override 单独重启用内网 IP——两方案验收标准相同（AC2/AC4），以 AC 为准不锁实现。

## T3 — confirmDelete helper（内部唯一实现）+ 单测

**exact files**：
- 新建 `ruoyi-react-frontend/src/components/shared/confirmDelete.ts`（UI-SPEC §4 命令式 API：`confirmDelete(opts: ConfirmDeleteOptions): void`；内部唯一 `Modal.confirm` 调用点；文案模板 `确认删除` / `将删除${target}，删除后不可恢复，是否继续？` / okText `删除` / cancelText `取消` / `okButtonProps: { danger: true }` / onOk 返回 Promise / 成功 `message.success("已删除")` / 失败 `handleApiError(error, "删除")` 且 Modal 保持打开）
- 新建 `ruoyi-react-frontend/src/components/shared/__tests__/confirmDelete.test.tsx`

**read_first**：
- UI-SPEC §4 全节（API 表 + 行为契约 + 文案模板逐字）
- `ruoyi-react-frontend/src/utils/errorHandler.ts:136`（handleApiError 签名：`(error, context, showMessage?)`）
- 63 已落地批删 handler 形态参考：`src/pages/system/user/index.tsx:547-560`（handleBatchDelete 内 Modal.confirm 样板）

**acceptance_criteria**：
1. vi.spyOn(Modal, "confirm") 断言：title/content/okText/cancelText/okButtonProps.danger 逐字匹配模板；content 含 target 代入
2. onOk reject 时 Modal 不关闭（antd 语义）且 handleApiError 被调用（mock 之）；onOk resolve 时 `已删除` message 触发
3. extra 节点透传进 content 组合；title 可覆盖默认 `确认删除`
4. 全仓 grep：`Modal.confirm(` 唯一调用点在 `confirmDelete.ts`（+存量豁免文件）

**fails_when**：失败路径静默 console 或吞错不提示；确认期间可重复触发 onOk；文案偏离模板逐字（批量 `确认批量删除` 标题可由 title 覆盖，63 兼容）

**verification**：
```bash
cd ruoyi-react-frontend && npx vitest run src/components/shared/__tests__/confirmDelete.test.tsx
# 期望：非测试文件中 Modal.confirm( 的命中全部落在 T2 豁免清单 + confirmDelete.ts（对照 T2 生成的清单 diff，不新增业务命中）
grep -rn "Modal.confirm(" src --include="*.ts*" | grep -v __tests__ | grep -v "components/shared/confirmDelete.ts"
```
预估：2.5h

## T4 — ConfirmDelete 声明式组件（复用 helper 内部实现）+ 单测

**exact files**：
- 新建 `ruoyi-react-frontend/src/components/shared/ConfirmDelete.tsx`（UI-SPEC §4 声明式 API：`ConfirmDeleteProps { target, onDelete, children?, title?, extra?, disabled? }`；内部调 `confirmDelete()`，不自建第二套 Modal 逻辑；默认渲染 danger 删除按钮 icon=DeleteOutlined、`action-btn-link-danger` 类；disabled 或 onDelete 缺失时按钮禁用）
- 修改 `ruoyi-react-frontend/src/components/shared/index.ts`（追加导出 ConfirmDelete / confirmDelete；若该文件已有导出结构则追加，不重构）
- 新建 `ruoyi-react-frontend/src/components/shared/__tests__/ConfirmDelete.test.tsx`

**read_first**：
- `ruoyi-react-frontend/src/components/shared/index.ts`（现有导出结构）
- UI-SPEC §4 + UI Considerations 表（E1/E6 探针裁决：确认中 loading 不可重复触发、失败不关闭可重试）
- `src/components/shared/ActionButtons.tsx` aria-label 惯例（UI-SPEC Dimension 7）

**acceptance_criteria**：
1. 渲染后点击按钮 → Modal.confirm 被 confirmDelete 路径触发一次，文案含 target 与不可恢复警示
2. disabled=true 或 onDelete 未传 → 按钮 disabled；自定义 children 替换默认按钮内容
3. 删除失败 → 错误提示可见且 Modal 保持打开；再次点击可重试
4. ConfirmDelete 与 confirmDelete 共用同一内部实现（grep 组件文件无第二处 Modal.confirm）

**fails_when**：组件内出现平行实现的 Modal.confirm（第二路径）；按钮视觉偏离 danger + action-btn-link-danger 惯例

**verification**：
```bash
cd ruoyi-react-frontend && npx vitest run src/components/shared/__tests__/
npm run type-check
```
预估：2.5h

## T5 — 既有封装内部等价切换（BatchDeleteButton / createDeleteAction）

**exact files**：
- 修改 `ruoyi-react-frontend/src/components/shared/BatchDeleteButton.tsx`（内部 Popconfirm → 调 `confirmDelete({ target: \`选中的 ${selectedCount} 项\`, onDelete: onConfirm, title: confirmTitle === BATCH_DELETE_CONFIRM.title ? "确认批量删除" : confirmTitle })`；props 零变化、按钮视觉/文案 `批量删除 ({selectedCount})` 与 selectedCount===0 返回 null 行为全保留；confirmTitle prop 保留，默认值随常量升级语义收敛为传给 helper 的 title）
- 修改 `ruoyi-react-frontend/src/constants/buttonStyles.tsx`（UI-SPEC Component Inventory 契约：`DELETE_CONFIRM.title` `确定要删除吗？`→`确认删除`、`okText` `确定`→`删除`；`BATCH_DELETE_CONFIRM.title` → `确认批量删除`——消费方收敛到封装层内部，与 63 的 11 处批删文案逐字兼容；升级后 grep 确认无业务文件直接 import 这两个常量）
- 修改 `ruoyi-react-frontend/src/components/shared/ActionButtons.tsx:113` `createDeleteAction`（render 内 Popconfirm → 渲染 `<ConfirmDelete target={...} onDelete={() => handleDelete(id)} />`；对外 ActionButton 接口零变化；title/label/disabled/ghost options 语义保留——ghost 视觉经 ConfirmDelete children 定制或注明降级；title 默认值改消费升级后的 DELETE_CONFIRM.title）

**read_first**：
- UI-SPEC §4 行为契约 1/2（内部切换、props 零破坏、不新建 hook）
- 两个文件全文 + 既有相关测试（`src/components/shared/__tests__/`、页面 hook 测试中 vi.spyOn(Modal,"confirm") 的既有写法——63 T4 已有先例）

**acceptance_criteria**：
1. 两文件内 grep `Popconfirm` = 0 命中（封装层内部不再使用裸 Popconfirm）
2. props 零破坏：BatchDeleteButton 既有调用签名不变（现网 0 消费点，主要靠类型层确认）；createDeleteAction 返回的 ActionButton 结构不变
3. 相关测试更新后通过：confirm 路径断言改 `vi.spyOn(Modal,"confirm")` 模式（63 T4 同款）
4. type-check 0 错误

**fails_when**：props interface 增删必填项；按钮视觉/文案偏离现状（danger ghost + `批量删除 (N)`）；出现第三条确认路径

**verification**：
```bash
cd ruoyi-react-frontend
grep -rn "Popconfirm" src/components/shared/BatchDeleteButton.tsx src/components/shared/ActionButtons.tsx  # 期望 0
npx vitest run src/components/shared/__tests__/ && npm run type-check
```
预估：2.5h

## T6 — lint 规则与封装切换的整合校验（轻量门）

> T2-T5 各自完成后，本任务做一次整合验证，防止规则与豁免清单在多任务并行修改下漂移。

**exact files**：无新建（只跑验证；若发现豁免清单缺漏则修改 `eslint.config.js` 豁免段与 `scripts/gen-p65-exemptions.mjs`）

**read_first**：T2 生成的豁免清单注释段

**acceptance_criteria**：
1. 定向 lint 新增 errors = 0（对比 125 存量基线）
2. T1-T5 的全部单测通过
3. lint 探针 fixture（含裸 Input/Select/Modal.confirm/Popconfirm 各一）在业务目录路径下 4 条全报 error、移入封装层目录后 0 报错

**fails_when**：豁免清单与实际命中集不一致

**verification**：同 T2/T4/T5 验证命令合集
预估：1h

## T7 — 63 已触达 11 处批删迁移到 confirmDelete（等价替换）

**exact files**（以 UI-SPEC §5 列表为准，grep `确认批量删除` 核对落点后逐个替换 handler 内 Modal.confirm 块为 `confirmDelete(...)` 单行调用）：
- `ruoyi-react-frontend/src/pages/system/user/index.tsx`（:547-560 一带）
- `ruoyi-react-frontend/src/pages/system/config/index.tsx`
- `ruoyi-react-frontend/src/pages/system/post/index.tsx`
- `ruoyi-react-frontend/src/pages/system/dict/hooks/useDictActions.ts`（dict×2 落点）
- `ruoyi-react-frontend/src/pages/system/role/hooks/useRoleActions.ts`
- `ruoyi-react-frontend/src/pages/system/notice/hooks/useNoticeData.ts`（notice 落点）
- `ruoyi-react-frontend/src/pages/network/devices/index.tsx`
- `ruoyi-react-frontend/src/pages/network/ports/index.tsx`
- `ruoyi-react-frontend/src/pages/network/mac/index.tsx`
- `ruoyi-react-frontend/src/pages/network/templates/hooks/useTemplateModals.ts`（:129 闸门下沉模式天然适配）
- （credentials 域文件路径含敏感词被工具拦截，executor 直接按 UI-SPEC §5 列表定位 `src/pages/network/credentials/index.tsx` 处理，与其他文件同款操作）

**read_first**：
- 每个落点文件内 `确认批量删除` 所在 handler 全文（替换前逐个读）
- UI-SPEC §5 范围表（**严格锁定 11 处，不扩面到 operations/duty/workorder 域**）

**acceptance_criteria**：
1. 11 处落点内 grep `Modal.confirm(` = 0（全部走 confirmDelete）；替换后确认文案与迁移前逐字等价（`确认批量删除` 标题 + `将删除选中的 {N} 项，删除后不可恢复，是否继续？` 内容由模板 + target 代入天然满足）
2. 各落点既有测试改 `vi.spyOn(Modal,"confirm")` 或直接 mock confirmDelete 后全通过（63 T4 已有 role/dict 先例）
3. operations/duty/workorder 域 handler 零改动（diff 核对）
4. type-check 0 / 全量 vitest 无新增失败

**fails_when**：迁移改变了确认行为语义（如丢掉成功刷新调用）；扩面到 §5 清单外文件；替换时丢失 onDelete 内的列表刷新逻辑

**verification**：
```bash
cd ruoyi-react-frontend
grep -rn "Modal.confirm(" src/pages/system src/pages/network --include="*.ts*" | grep -v __tests__  # 期望 0
npm run type-check && npm run test -- --run   # 3843+ / 0 failed
```
预估：4h

## T8 — 收口：SC1-4 逐条验证 + 基线保护 + SUMMARY

**exact files**：
- 新建 `.planning/phases/65-contracts/65-P1-SUMMARY.md`（不 commit）

**read_first**：
- `.planning/phases/64-p1/64-P1-SUMMARY.md`（结构模板）
- 本 PLAN 全部任务的 verification 输出

**acceptance_criteria（SC 逐条）**：
- **SC1**：AppInput/AppSelect 单测证明零配置获得 allowClear（Select 另含 showSearch≥8 自动策略）→ T1 测试输出
- **SC2**：定向 lint 证明封装层外裸组件被拦截（error）+ 存量豁免清单存在且带 `// P67 收敛后移除` 注释 → T2/T6 输出
- **SC3**：`grep -rn "Modal.confirm(" src --include="*.ts*" | grep -v __tests__` 的命中 = `confirmDelete.ts` + T2 豁免清单文件（逐个对照清单 diff，清单外不得有业务命中）；BatchDeleteButton/createDeleteAction 内 Popconfirm=0；11 处批删落点 0 裸 confirm → T3/T5/T7 输出
- **SC4**：ConfirmDelete 文案模板含 target + 不可恢复警示（单测逐字断言）；失败路径 handleApiError 可见反馈 + Modal 保持打开（单测）→ T3/T4 输出
- **基线保护**：type-check 0 错误；`npm run test -- --run` 3843+ passed / 0 新增失败；定向 lint 新增 errors = 0（存量 125 不动）
- SUMMARY 含：任务完成表（T1-T8 + commit 前置说明——本 phase 不 commit）、验证命令与输出、63→65 契约衔接说明（helper 即 63 SUMMARY:89 上报的兑现）、P67 待办清单（存量豁免清偿 + operations/duty/workorder 域迁移 + RemoteSelect 已归 66）

**fails_when**：任一 SC 无可执行验证输出支撑；基线任一项回退；SUMMARY 声称与实际验证输出不符

**verification**：SUMMARY 内嵌全部命令输出
预估：1.5h

---

## 依赖顺序

```
T1（tracer，必须先行）
 └→ T2（lint） ─┐
 ├→ T3（helper） ─┼→ T5（封装切换，依赖 T3/T4）→ T6（整合门）→ T7（批删迁移，依赖 T3）→ T8（收口）
 └→ T4（组件，依赖 T3）┘
```
可并行：T2 与 T3/T4 线程独立；T5 必须在 T3/T4 后；T7 必须在 T3 后且建议在 T2 后（豁免清单生成需规则已上线，否则存量 Modal.confirm 命中会报 error）。

**总工时预估：21h**（T1 3h / T2 4h / T3 2.5h / T4 2.5h / T5 2.5h / T6 1h / T7 4h / T8 1.5h）

## 执行注意（executor 必读）

1. T2 豁免清单生成必须在 T7 迁移**之前**跑一次（存量 Modal.confirm 全命中），T7 完成后**复跑脚本**把已迁移文件从豁免清单移除——豁免清单是活文档，收口时清单应只剩未迁移的存量命中。
2. 全量 `npm run lint` 与全量 `npm run test` 耗时长且存量红——一切判闸用定向命令 + 基线对比。
3. 任何任务发现必须触碰 UI-SPEC 未覆盖的组件/行为，停下上报 planner，不得自行扩面（硬边界 locked）。

---

## Plan Checker Sign-Off（2026-10-25，gsd-plan-checker）

亲测抽查：`eslint.config.js:93-100` 内网 IP selector / `:113-118` 测试 override 关整条 no-restricted-syntax / `:119` prettier 位置、`no-restricted-imports` 无既有配置 ✓；BatchDeleteButton 现网 0 业务消费（仅 shared 自身+测试）、createDeleteAction 0 外部消费（仅 ActionButtons.tsx + 测试）✓；`src/components/ui/` 与 `src/components/index.ts` 不存在 ✓；64-P1-SUMMARY 验证表 type-check 0 / 3843 passed 0 failed / lint 125 存量 ✓；63-P0-SUMMARY:16 T4（11 处批删 handler 内 Modal.confirm、文案逐字）、:26 D3、:89 上报「统一抽 helper」✓；handleApiError 签名（errorHandler.ts:136）✓；post 页含 `__tests__/` ✓。

1. **REQ 覆盖**：PASS — T1→SC1、T2→SC2、T3/T4/T5/T7→SC3、T3/T4 文案模板→SC4、T8 逐条收口，映射完整。
2. **契约一致性**：PASS（修订后）— API/逃生口/≥8 阈值、双形态共用 confirmDelete 内部实现、lint 4+1 selector、豁免方案 a、RemoteSelect 仅契约均忠实 UI-SPEC；修订 ② 补齐 UI-SPEC Component Inventory 的 buttonStyles 常量升级动作（原 PLAN 遗漏）。
3. **可执行性**：PASS（修订后）— 全部任务有可运行 verification 命令；修订 ③④ 修正 flat config 下无效的 `--rules-dir` 与占位伪命令；修订新增全局「行号锚点漂移需重定位」纪律。
4. **依赖与顺序**：PASS — T1 先行、T2∥T3→T4→T5→T6→T7→T8 自洽；T2 豁免清单 T7 前生成、T7 后复跑由执行注意 1 + T8 收口覆盖。
5. **硬边界**：PASS — 禁抽业务/禁扩面/禁第三路径/存量 45 处归 67/不 commit 全部写死，T7 AC3 显式锁 operations/duty/workorder 零改动。
6. **验证基线**：PASS — type-check 0 / vitest 3843+0 / lint 125 存量与 64 SUMMARY 逐字吻合；禁全量 lint 判闸、定向数新增口径正确。
7. **planner 事实声称**：PASS（修订后）— 7 项亲测 5 项吻合；「22 文件 handleBatchDelete」实测修正为 29 个源文件（修订 ①）。

FLAG（不阻塞，executor 须知）：operations/assets/index.tsx 存在「确认批量删除」文案 + handleBatchDelete，但不在 63 的 11 处收敛清单（63 T4 对照/后续域）——T7 严禁触碰，已随修订 ① 记入事实注记；system/menu 为 63 对照组，同归 P67。

Approval: approved（REVISED，共 5 处修订，见「修订记录」）
