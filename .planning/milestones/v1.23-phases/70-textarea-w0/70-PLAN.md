---
phase: "70"
slug: "70-textarea-w0"
type: plan
created: "2026-10-27"
planner: gsd-planner
wave: 1
depends_on: []
requirements: [TEXT-01, TEXT-03, TEXT-04, HYG-05]
files_modified:
  - .planning/phases/70-textarea-w0/70-DEFAULTS-AUDIT.md
  - .planning/phases/70-textarea-w0/70-baseline.json
  - ruoyi-react-frontend/src/components/ui/AppTextArea.tsx
  - ruoyi-react-frontend/src/components/ui/__tests__/AppTextArea.test.tsx
  - ruoyi-react-frontend/src/components/ui/index.ts
  - ruoyi-react-frontend/src/components/shared/SearchForm.tsx
  - ruoyi-react-frontend/src/components/shared/__tests__/SearchForm.test.tsx
  - ruoyi-react-frontend/eslint.config.js
  - ruoyi-react-frontend/scripts/gen-p65-exemptions.mjs
autonomous: true
granularity: standard
response_language: Chinese
---

# Phase 70 执行计划 — TextArea 契约基建 (W0)

**目标**：交付 `AppTextArea` 独立具名契约组件（默认 `allowClear`）+ SearchForm `textarea` field type + eslint「成员表达式 + 解构」双 selector（同步进 P65 契约块与 P68 hex 豁免块两处数组）+ HYG-05 权威计数基线落盘，为 Phase 71/74 提供迁移压力信号与对账分母。

**需求**：TEXT-01（契约默认清空能力）、TEXT-03（搜索区多行字段类型）、TEXT-04（裸 TextArea 双形态 eslint 拦截）、HYG-05（权威计数基线落盘）。

**权威上下文**：
- ROADMAP Phase 70 段（Goal/Notes/5 条 Success Criteria）
- `.planning/research/ARCHITECTURE.md` §4.2 W0 节、`STACK.md` (a) 节 AppTextArea 骨架、`PITFALLS.md` 领域 1（1.1 selector 盲区 / 1.2 allowClear 语义耦合）
- 惯例先例：`.planning/milestones/v1.22-phases/65-contracts/65-PLAN.md`（T1/T2 结构、契约测试写法）、`66-page-form/66-PLAN.md`

**验证基线**（`.planning/notes/v1.23-lint-baseline.json`，2026-10-27 实测）：
- `npm run lint` 全仓 **83 errors / 1367 warnings**（selector 上线当天新增 errors 必须为 0）
- `npm run type-check` → 0 错误；`npm run test -- --run` → 全绿（P64 后 3900+ passed 基线）
- grep 粗估（2026-10-27 计划期实测，非权威数）：`src/pages` 等业务目录 `<Input.TextArea` 约 26 行、`<TextArea` 约 48 行（含 components 下非 ui/shared 封装层文件）；**权威数以 T5 的 eslint selector 实跑命中为准**，grep 数只作三数对照之一。

**硬边界（locked，违反即任务失败）**：
1. `AppTextArea` 独立具名导出；**不挂 `AppInput.TextArea` 静态别名**（STACK.md 曾建议挂载，已被 locked 决策否决——P67 的 `AppInput as Input` 别名迁移路径因此对 TextArea 形态失效，Phase 71 需逐处替换，本 phase 不做迁移）。
2. 成员表达式 selector + 解构 selector **两条都要**，且**追加进既有 no-restricted-syntax 数组**（P65 契约块），**禁另起 config 块、禁置 "off"**（P68 DEBT-01 教训：flat config 同名规则键整体覆盖）。
3. P68 hex 豁免块重声明的完整数组必须同步追加两条新 selector（漏同步 = 该块覆盖的 27 个文件对新 selector 静默失明）。
4. 默认值白名单（`showCount`/`maxLength`/`autoSize` 等是否下沉为默认）**必须先对 74 处存量做四象限审计再定稿**（PITFALLS 1.2）；`allowClear = true` 是唯一预设默认，其余默认值的取舍以 T1 审计结论为准。
5. 零新依赖；契约测试用 vitest（先例 `src/components/ui/__tests__/AppInput.test.tsx`）。
6. 存量命中文件**不迁移**（归 Phase 71），靠预登记豁免保证 selector 上线当天零新增 error。

**行号锚点纪律**：本 PLAN 的 `eslint.config.js:114/:118/:202` 等行号为 2026-10-27 快照，漂移需按语义锚（规则名 / 注释文案 / 锚点注释 `__P65_EXEMPTION_FILES__`）重新 grep，不得按失效行号盲改。

**记录在案的计划期偏离**：本 phase 检出 frontend 信号但未生成 UI-SPEC（ui_safety_gate 为 blocking）。理由：Phase 70 是契约基建，组件形态完全复制 AppInput 已验证先例（透传 + 单默认值），无新视觉面；已在返回消息中列为 Open Question，用户如需可补跑 `/gsd-ui-phase 70`。

---

## 修订记录

- **2026-10-27 gsd-plan-checker REVISED**（第 1 轮，3 warnings → 0）：① T4 首选成员表达式 selector 写法由独立 `JSXMemberExpression[object.name=…]` 节点形态修正为挂在 `JSXOpeningElement` 上的 `name.type/name.object.name/name.property.name` 形态（独立节点形态报错位置落在 name 上、且与 P65 既有 selector 的报告口径不一致），locked 决策的语义（覆盖成员表达式形态）不变，T4 验证含 fixture 双形态试拦；② T5 基线补充 P66 豁免清单条目数与 `suppressed` 字段口径说明；③ T3 补「textarea 字段不传 onPressEnter（多行回车必须换行）」为显式验收项；④ 补 threat_model 块。

---

## 事实核查注记（planner 实测，executor 须知）

1. **eslint.config.js 结构**（544 行，flat config 数组）：P65 契约块 `:115-164`（no-restricted-syntax 数组现含 5 个 selector：Input/Select/Modal.confirm/Popconfirm/hex + no-restricted-imports antd 子包拦截）；P68 hex 豁免块 `:171-231`（files 27 项，重声明**不含 hex** 的完整 5 selector 数组）；封装层豁免块 `:239-251`（`src/components/ui/** + src/components/shared/**`，数组仅含内网 IP selector → **对全部 P65 selector 天然豁免，新 selector 无需改动此块**）；P66 存量豁免块 `:313-368`（no-restricted-imports，与 TextArea 无关）；P65 存量豁免块 `:373-509`（锚点 `__P65_EXEMPTION_FILES__`，数组仅内网 IP）；测试文件豁免块 `:515-535`（`no-restricted-syntax: "off"` → **测试文件天然豁免新 selector**）；prettier 在 `:537` 必须保持最后。
2. **`:114` 注释**现为「Input.TextArea|Search|Password|OTP 为成员表达式 JSX 名，不在本 selector 命中面（有意不拦）」——T4 必须把 TextArea 移出该清单并注明 Phase 70 已拦截，Search|Password|OTP 维持不拦。
3. **gen-p65-exemptions.mjs**（115 行）：靠 `P65_MESSAGE_MARKS` 数组（message 前缀匹配）识别命中文件，经临时 scan config（锚点段换占位 glob）规避自清空悖论。**新 selector 的 message 必须以统一前缀开头，并在 marks 数组追加该前缀**，否则豁免清单收不进 TextArea 命中文件。
4. **gen-p66-exemptions.mjs**：拦的是 `useTableManager`/`usePagination` import，与 TextArea 无关，本 phase **不修改**，但 T5 需跑 `--dry` 确认幂等（其清单条目数进基线）。
5. **SearchForm.tsx**：`SearchField` union（`:30-48`）+ `renderField` switch（`:66-112`）。`input` 分支传了 `onPressEnter={onSearch}`；**textarea 分支不得传**（多行回车必须换行，见 PITFALLS 1.2 受控语义）。
6. **AppInput.tsx** 16 行透传模式：`ComponentProps<typeof Input>` + `allowClear = true` + rest 全透传 + default 导出。AppTextArea 照抄结构，类型换成 `ComponentProps<typeof Input.TextArea>`（TS 5.9 实测可推导 `TextAreaProps`，ref 类型 `TextAreaRef` 来自 `@rc-component/input`；antd 6.6.4 的 TextArea 原生支持 `allowClear`）。
7. **存量分布**（STACK.md (a) 节）：`<Input.TextArea` 18 处 + `<TextArea` 43 处 JSX 开标签（研究期口径），计划期 grep 行数 26/48——两口径差异来自同文件多命中与行/开标签计数差，**一律以 T5 eslint 实跑为准**。
8. **ROADMAP SC1/SC2 的人工验收**（一键清空交互、搜索区渲染）由 T2/T3 的 vitest 行为断言 + `ant-input-clear-icon` class 断言覆盖；浏览器实机验收归 Phase 75 UAT，不阻塞本 phase。

## Artifacts this phase produces

| 符号/文件 | 类型 | 说明 |
|---|---|---|
| `AppTextArea` | 函数组件（具名导出 + default） | `src/components/ui/AppTextArea.tsx`；`allowClear = true` 唯一预设默认，其余透传 `Input.TextArea` |
| `AppTextAreaProps` | 导出类型 | `ComponentProps<typeof Input.TextArea>` |
| `SearchField` 的 `{ type: "textarea"; … }` union 成员 | 类型扩展 | `src/components/shared/SearchForm.tsx` |
| `renderField` 的 `case "textarea"` | 代码路径 | 渲染 `AppTextArea`，不传 onPressEnter |
| eslint selector ×2 | 规则条目 | `JSXOpeningElement[name.type='JSXMemberExpression'][name.object.name='Input'][name.property.name='TextArea']` 与 `JSXOpeningElement[name.name='TextArea']`，进 P65 契约块 + P68 hex 豁免块两处数组 |
| `P65_MESSAGE_MARKS` +1 前缀 | 脚本常量 | `scripts/gen-p65-exemptions.mjs` |
| `70-DEFAULTS-AUDIT.md` | 计划产物 | 四象限审计 + 默认值定稿记录 |
| `70-baseline.json` | 计划产物 | HYG-05 权威计数基线（三数对照） |

<threat_model>
**ASVS L1 / block on high。** 本 phase 纯前端契约基建：不新增 HTTP 端点、不触数据库、不加存储、不加加密路径、不改鉴权。`AppTextArea` 是 antd `Input.TextArea` 的 props 透传封装，无自定义 DOM 注入面（React 默认转义不变）；eslint selector 与豁免清单是构建期 lint 配置，无运行时面。结论：无 L1 威胁需要缓解，无 high 及以上项；Phase 75 UAT 的既有 XSS 巡检口径继续覆盖多行文本渲染。
</threat_model>

---

## T1 — 存量 74 处 TextArea 四象限审计 + 默认值定稿

> **先行任务**：T2 的默认值白名单、是否默认 `autoSize`、SearchForm textarea 字段的可选 props 全部由本任务定稿。T1 未完成不得开始 T2。

- **type**: analysis
- **wave**: 1
- **depends_on**: []
- **exact files**：
  - 新建 `.planning/phases/70-textarea-w0/70-DEFAULTS-AUDIT.md`

**read_first**：
- `ruoyi-react-frontend/src/components/ui/AppInput.tsx`、`AppSelect.tsx`（契约默认值先例）
- `.planning/research/PITFALLS.md` Pitfall 1.2（showCount/maxLength/受控/autoSize 四重耦合）
- `node_modules/antd/es/input/TextArea.d.ts`（可用 props 面）

**action**：
1. 用 grep 枚举 `ruoyi-react-frontend/src`（排除 `__tests__`/`*.test.*` 与 `src/components/ui/**`、`src/components/shared/**`）中全部 `<Input.TextArea` 与 `<TextArea` JSX 开标签出现点，逐处记录四象限：① 是否带 `showCount`；② 是否带 `maxLength`；③ 是否受控（`value=`）；④ 是否带 `onChange`；附加标注：是否带 `autoSize`/`rows`、所在文件是否已在 P65 豁免清单。
2. 在审计文档中给出定稿结论，每项默认值必须给出「存量命中统计 + 结论 + 理由」三段：`allowClear = true`（locked 预设，确认无高危耦合例外并记录例外清单）；`showCount`/`maxLength`（预期结论：**不下沉默认**——maxLength 截断语义与存量耦合，PITFALLS 1.2）；`autoSize`（预期结论倾向**不下沉默认**、由调用方显式传，若审计发现 ≥70% 存量显式 autoSize 也不改变结论——下沉默认会改变未显式声明处的高度行为）；受控 + onChange 清除路径（`onChange('')` 语义记录，归一化决策留 SearchForm 层：T3 中 textarea 字段空串直通 filters，与 input 分支现状一致）。
3. 定稿 T3 的 `SearchField` textarea 成员可选 props 清单（`placeholder` 必有；`rows`/`autoSize` 是否进 union 由审计结论决定）。

**acceptance_criteria**：
1. `70-DEFAULTS-AUDIT.md` 存在，含四象限逐处清单（条数 ≥ 计划期粗估 74 上下，允许±，以实测为准）与每个候选默认值的「统计/结论/理由」。
2. `allowClear`/`showCount`/`maxLength`/`autoSize` 四项各有明确定稿结论，无 TBD。
3. 明确列出「显式传 `allowClear={false}` 的存量例外清单」（若有），供 T2 测试覆盖与 Phase 71 迁移参考。

<verify>
  <automated>Select-String -Path ".planning/phases/70-textarea-w0/70-DEFAULTS-AUDIT.md" -Pattern "TBD|待定"</automated>
  <fails_when>文档不存在导致命令报错，或存在任意 TBD/待定 匹配行</fails_when>
  <automated>Select-String -Path ".planning/phases/70-textarea-w0/70-DEFAULTS-AUDIT.md" -Pattern "定稿结论"</automated>
  <fails_when>无匹配行（allowClear/showCount/maxLength/autoSize 至少各出现一次定稿结论，缺任一项即失败）</fails_when>
</verify>

预估：1.5h

---

## T2 — Tracer 切片：AppTextArea 组件 + 单测 + 出口导出

- **type**: tracer
- **wave**: 2
- **depends_on**: [T1]

> 全链路（组件 → 出口 → 单测 → type-check）走通后才展开 T3-T5。

**exact files**：
- 新建 `ruoyi-react-frontend/src/components/ui/AppTextArea.tsx`
- 新建 `ruoyi-react-frontend/src/components/ui/__tests__/AppTextArea.test.tsx`
- 修改 `ruoyi-react-frontend/src/components/ui/index.ts`

**read_first**：
- `ruoyi-react-frontend/src/components/ui/AppInput.tsx`（16 行透传模式，逐行照抄结构）
- `ruoyi-react-frontend/src/components/ui/__tests__/AppInput.test.tsx`（测试写法先例：render + fireEvent + `.ant-input-clear-icon` class 断言）
- `.planning/phases/70-textarea-w0/70-DEFAULTS-AUDIT.md`（T1 定稿的默认值白名单）
- `.planning/research/STACK.md` (a) 节骨架（注意：其中「挂载 `AppInput.TextArea = AppTextArea`」一行已被 locked 决策否决，**不实现**）

**action**：
1. `AppTextArea.tsx`：文件头注释注明「Phase 70 契约组件 — AppTextArea，契约来源 .planning/phases/70-textarea-w0/70-PLAN.md」；`import { Input } from "antd"`；`export type AppTextAreaProps = ComponentProps<typeof Input.TextArea>;`；具名函数 `AppTextArea({ allowClear = true, ...rest }: AppTextAreaProps)` 返回 `<Input.TextArea allowClear={allowClear} {...rest} />`；default 导出。默认值白名单严格按 T1 定稿：T1 结论为「不下沉」的 prop 一律不写默认（留在 rest 透传）。
2. `index.ts`：追加 `export { AppTextArea } from "./AppTextArea";` 与 `export type { AppTextAreaProps } from "./AppTextArea";`。**禁止**追加 `AppTextArea as TextArea` 或 `AppInput.TextArea = AppTextArea` 静态挂载（locked 决策；P67 别名段注释区不加 TextArea 条目）。
3. `__tests__/AppTextArea.test.tsx`（vitest + @testing-library/react，外层 `<App>` wrapper 同 AppInput.test.tsx）：
   - 零配置渲染 textarea（`container.querySelector("textarea")`）、输入值后出现 `.ant-input-clear-icon`（SC1 契约默认值生效的行为断言）；
   - 显式 `allowClear={false}` 逃逸口 → clear icon 为 null；
   - 受控 `value`/`onChange` 透传：fireEvent.change 后 onChange 收到事件的 `e.target.value`；
   - `placeholder`/`rows` rest 透传到 DOM；
   - `autoSize` 显式传入生效（若 T1 定稿不下沉，断言默认渲染无 autoSize 行为、显式传后生效）。

**acceptance_criteria**：
1. `npm run type-check` 0 错误。
2. `npx vitest run src/components/ui/__tests__/AppTextArea.test.tsx` 全绿，含 clear-icon 默认出现与 `allowClear={false}` 逃逸两条关键断言。
3. `index.ts` 导出 `AppTextArea` 具名 + 类型，且**不含** `TextArea` 别名导出（grep `as TextArea` 于 `src/components/ui/index.ts` 无匹配）。
4. `npm run test -- --run` 全量基线不回归。

**fails_when**：封装吞掉/改名任何 rest prop；出现 `AppInput.TextArea` 静态挂载或 `as TextArea` 别名导出；默认值白名单与 T1 定稿不一致（如下沉了结论为不下沉的 prop）。

<verify>
  <automated>cd ruoyi-react-frontend; npm run type-check</automated>
  <fails_when>非零退出码，或输出中出现任意 error TS 行</fails_when>
  <automated>cd ruoyi-react-frontend; npx vitest run src/components/ui/__tests__/AppTextArea.test.tsx</automated>
  <fails_when>非零退出码，或摘要行出现 failed 大于 0</fails_when>
  <automated>cd ruoyi-react-frontend; Select-String -Path src/components/ui/index.ts -Pattern "as TextArea|TextArea ="</automated>
  <fails_when>存在任意匹配行（应零匹配）</fails_when>
</verify>
（`npm run test -- --run` 全量回归并入 T6 收口闸门，避免逐任务全量重跑。）

预估：2h

---

## T3 — SearchForm `textarea` field type（TEXT-03，SC2）

- **type**: feature
- **wave**: 3
- **depends_on**: [T2]

**exact files**：
- 修改 `ruoyi-react-frontend/src/components/shared/SearchForm.tsx`
- 新建 `ruoyi-react-frontend/src/components/shared/__tests__/SearchForm.test.tsx`（若已存在则追加 describe 块，先 `ls` 确认）

**read_first**：
- `ruoyi-react-frontend/src/components/shared/SearchForm.tsx` 全文（union 与 renderField 结构）
- `ruoyi-react-frontend/src/components/ui/__tests__/AppTextArea.test.tsx`（T2 产出，AppTextArea 行为面）
- `.planning/phases/70-textarea-w0/70-DEFAULTS-AUDIT.md`（textarea 字段可选 props 定稿清单）

**action**：
1. `SearchField` union 追加成员：`{ type: "textarea"; name: string; label: string; placeholder?: string }` + T1 定稿的可选 props（若定稿进 union，则含 `rows?: number` / `autoSize?: boolean | { minRows: number; maxRows: number }`）。
2. `renderField` switch 追加 `case "textarea"`：`<Form.Item key={field.name} name={field.name} label={field.label}><AppTextArea placeholder={field.placeholder} {...(定稿可选 props 透传)} /></Form.Item>`；**不传 `onPressEnter`**（多行回车必须换行，区别于 input 分支）；不额外传 `style`，布局与 AppSelect 分支一致即可。
3. 更新 `renderField` 上方注释（`:65` 渲染分发说明）与文件头用法注释，加入 textarea 示例行。
4. 测试：渲染含 `{ type: "textarea", name: "remark", label: "备注" }` 的 SearchForm → 出现 `textarea` 元素且带 clear icon（输入值后）；显式 `allowClear={false}` 字段无 clear icon；断言 textarea 上**无** `onPressEnter` 语义（fireEvent.keyDown Enter 不触发 onSearch mock）；受控 value/onChange 接入 form（fireEvent.change 后 `form.getFieldValue("remark")` 收到值）。

**acceptance_criteria**：
1. `type: "textarea"` 字段渲染为 AppTextArea 且具备清空能力（SC2），业务侧无需 `custom.render` 逃生口。
2. textarea 分支不触发「回车即搜索」；input 分支行为不变（既有测试不回归）。
3. `npm run test -- --run` 全量不回归；type-check 0 错误。

**fails_when**：textarea 分支带 onPressEnter；SearchField union 改动破坏既有 `input/select/remoteSelect/custom` 字段的类型或渲染；搜索区仍需 custom.render 才能渲染多行字段。

<verify>
  <automated>cd ruoyi-react-frontend; npx vitest run src/components/shared/__tests__/SearchForm.test.tsx</automated>
  <fails_when>非零退出码，或摘要行出现 failed 大于 0</fails_when>
  <automated>cd ruoyi-react-frontend; npm run type-check</automated>
  <fails_when>非零退出码</fails_when>
</verify>
（全量 `npm run test -- --run` 并入 T6 收口闸门。）

预估：2h

---

## T4 — eslint 双 selector + gen 脚本 marks + 豁免预登记（TEXT-04，SC3/SC4）

- **type**: config
- **wave**: 3
- **depends_on**: [T1]
- **可与 T3 并行**（文件不相交；T3 触碰 SearchForm，T4 触碰 eslint.config.js 与 gen 脚本）

**exact files**：
- 修改 `ruoyi-react-frontend/eslint.config.js`
- 修改 `ruoyi-react-frontend/scripts/gen-p65-exemptions.mjs`

**read_first**：
- `ruoyi-react-frontend/eslint.config.js` `:104-231`（P65 契约块 + P68 hex 豁免块）与 `:232-251`（封装层豁免）、`:369-509`（P65 存量豁免 + 锚点）、`:510-535`（测试豁免）
- `ruoyi-react-frontend/scripts/gen-p65-exemptions.mjs` 全文（`P65_MESSAGE_MARKS` 机制与自清空悖论规避）
- `.planning/research/PITFALLS.md` Pitfall 1.1（selector 盲区与追加纪律）

**action**：
1. **P65 契约块**（`no-restricted-syntax` 数组，hex selector 之后）追加两条 selector，message 共用前缀 `禁止直接使用 antd 多行输入框`：
   - 成员表达式形态：`selector: "JSXOpeningElement[name.type='JSXMemberExpression'][name.object.name='Input'][name.property.name='TextArea']"`，message：`禁止直接使用 antd 多行输入框 <Input.TextArea>，请使用 @/components/ui 的 AppTextArea（默认 allowClear）`。
   - 解构/具名形态：`selector: "JSXOpeningElement[name.name='TextArea']"`，message：`禁止直接使用 antd 多行输入框 <TextArea>，请使用 @/components/ui 的 AppTextArea（默认 allowClear）`。
   > locked 决策表述的 `JSXMemberExpression[object.name='Input'][property.name='TextArea']` 是成员表达式**语义**；实现采用挂在 JSXOpeningElement 上的 `name.*` 属性路径，报错位置与 P65 既有 selector 口径一致。T4 验证含 fixture 双形态试拦，若实测该写法不命中，退回独立节点形态 `JSXMemberExpression[object.name='Input'][property.name='TextArea']` 并在 PR 描述记录差异——两种写法满足同一 locked 语义（成员表达式形态被拦截），验收以 fixture 试拦为准。
2. **P68 hex 豁免块**（`:202` 起重声明的数组）**同步追加完全相同的两条 selector + message**（该块重声明完整数组，漏同步 = 27 个文件对新 selector 静默失明）。
3. **`:114` 注释**改写：TextArea 移出「有意不拦」清单，改为「Input.TextArea 已由 Phase 70 双 selector 拦截（P65 契约块 + P68 hex 豁免块两处同步）；Search|Password|OTP 仍为成员表达式形态，不在命中面（有意不拦）」。
4. **封装层豁免块、测试豁免块、P65 存量豁免块的 rules 数组均不改**（封装层与测试对整条规则天然豁免/关闭；P65 存量豁免块数组仅含内网 IP，存量 TextArea 命中文件由本块的 files 清单覆盖）。
5. `gen-p65-exemptions.mjs` 的 `P65_MESSAGE_MARKS` 数组追加一条 `"禁止直接使用 antd 多行输入框"`（与两处 selector message 前缀逐字一致）。
6. **预登记顺序（同 commit）**：① 改 eslint.config.js + gen 脚本；② `node scripts/gen-p65-exemptions.mjs --dry` 打印命中文件全集（此时含 TextArea 命中）；③ 正式跑（无 --dry）写回锚点段；④ `node scripts/gen-p66-exemptions.mjs --dry` 确认 P66 清单幂等无 diff（只读验证，不写）。禁止手工编辑锚点数组。
7. fixture 试拦（验证后删除）：在 `src/components/__lint_probe__.tsx` 临时写入 `<Input.TextArea />`（import 自 `antd`）与解构形态各一处，定向 lint 必须各报一条 error；同文件放到 `src/components/ui/` 下试一份则不报（封装层豁免）。

**acceptance_criteria**：
1. fixture：成员表达式与解构形态各命中 1 条 error，message 指向 AppTextArea；封装层目录内不命中。
2. selector 上线当天全量 `npx eslint src` **新增 error = 0**（对照 83 errors 基线，总 error 数不变；warnings 数亦不变）。
3. `gen-p65-exemptions.mjs --dry` 命中文件集合与写回后的锚点段一致（重跑 --dry 输出稳定可 diff）；gen-p66 --dry 输出与现清单零 diff。
4. P68 hex 豁免块内文件（如 `src/pages/system/settings/default-theme.tsx`）中若存在裸 TextArea 也被拦截（抽 1 个 P68 块文件定向验证 selector 同步生效）。

**fails_when**：全量 lint error 数 > 83（存量未被豁免清单全覆盖）；P68 块文件内裸 TextArea 不报错（数组漏同步）；出现新起 config 块或任何 `"no-restricted-syntax": "off"` 写法；gen 脚本 marks 未同步导致 --dry 命中数为 0。

<verify>
  <automated>cd ruoyi-react-frontend; npx eslint src/components/__lint_probe__.tsx</automated>
  <fails_when>报错少于 2 条（成员表达式 + 解构形态须各命中 1 条 no-restricted-syntax error，message 含「禁止直接使用 antd 多行输入框」）</fails_when>
  <automated>cd ruoyi-react-frontend; npx eslint src 2>&1 | Select-String "error"</automated>
  <fails_when>输出中的 error 总数大于 83（基线 83，新增必须为 0）</fails_when>
  <automated>cd ruoyi-react-frontend; node scripts/gen-p65-exemptions.mjs --dry</automated>
  <fails_when>命中文件数为 0，或连续两次运行输出不一致（不稳定可 diff）</fails_when>
  <automated>cd ruoyi-react-frontend; node scripts/gen-p66-exemptions.mjs --dry</automated>
  <fails_when>退出码非零，或打印清单与 eslint.config.js 现锚点段存在 diff</fails_when>
</verify>

预估：3h

---

## T5 — HYG-05 权威计数基线落盘（SC5）

- **type**: docs
- **wave**: 4
- **depends_on**: [T4]

**exact files**：
- 新建 `.planning/phases/70-textarea-w0/70-baseline.json`

**read_first**：
- `.planning/notes/v1.23-lint-baseline.json`（既有全量基线：83/1367/by_rule/文件级明细——直接引用，不重扫全量）
- `ruoyi-react-frontend/eslint.config.js` 锚点段（豁免清单条目数）
- `.planning/phases/70-textarea-w0/70-DEFAULTS-AUDIT.md`（T1 的 grep 粗估数）

**action**：
1. 实跑采集三组数：① **selector 实跑命中数**——用 T4 的 gen 脚本 `--dry` 输出（命中文件数）+ 定向统计命中 message 条数（`npx eslint src --format json` 后按 `禁止直接使用 antd 多行输入框` 前缀计数，按成员/解构两形态分列）；② **豁免清单条目数**——`__P65_EXEMPTION_FILES__` 锚点段文件数 + `__P66_EXEMPTION_FILES__` 锚点段文件数（分别计数）；③ **grep 粗估数**——`<Input.TextArea` 与 `<TextArea` 各自行数（排除 tests 与 ui/shared 封装层）。
2. `70-baseline.json` 结构：`captured_at`（当日）、`phase: "70"`、`source`、`textarea_selector_hits: { files, member_form_messages, simple_form_messages }`、`exemptions: { p65_files, p66_files }`、`grep_estimate: { member_lines, simple_lines }`、`ref_full_lint: { file: ".planning/notes/v1.23-lint-baseline.json", errors: 83, warnings: 1367, suppressed_note: "suppressed=299 为 eslint json 口径的行内禁用数，与豁免清单条目数是不同口径，勿混用" }`、`usage_note`（写明 Phase 71/74 的引用路径与对账口径：**对账分母 = textarea_selector_hits 与 exemptions.p65_files，grep_estimate 仅对照参考**）。
3. 三数对照结论写进 JSON 的 `triangulation` 字段：selector 命中文件数 ≤ grep 涉及文件数属正常（一文件多处命中），若 selector 命中文件数 > grep 文件数则说明 grep 漏了形态（如多行 JSX），须在文档中解释差异。

**acceptance_criteria**：
1. `70-baseline.json` 存在且为合法 JSON（`node -e "JSON.parse(...)"` 通过），含上述全部字段，无 null 占位。
2. selector 命中文件数与 T4 写回的 P65 锚点段新增条目数一致（TextArea 命中文件全部进豁免清单）。
3. `usage_note` 写明 Phase 71/74 的引用路径：`.planning/phases/70-textarea-w0/70-baseline.json`。

**fails_when**：JSON 解析失败、缺任一字段、或 selector 命中数与豁免清单条目数对不上（差额未解释）。

<verify>
  <automated>node -e "JSON.parse(require('fs').readFileSync('.planning/phases/70-textarea-w0/70-baseline.json','utf8')); console.log('ok')"</automated>
  <fails_when>报 SyntaxError 或输出非 ok</fails_when>
  <automated>node -e "const b=JSON.parse(require('fs').readFileSync('.planning/phases/70-textarea-w0/70-baseline.json','utf8')); if(!b.textarea_selector_hits||!b.exemptions||!b.grep_estimate||!b.usage_note) process.exit(1)"</automated>
  <fails_when>退出码 1（缺 textarea_selector_hits / exemptions / grep_estimate / usage_note 任一字段或为 null 占位）</fails_when>
</verify>

预估：1h

---

## T6 — 收口验证（全量闸门）

- **type**: verification
- **wave**: 4
- **depends_on**: [T2, T3, T4, T5]

**action**：依次跑全量闸门并对照基线记录结果；任何一项失败回对应任务修复后重跑。
1. `npm run type-check` → 0 错误。
2. `npm run test -- --run` → 全绿（P64 基线 3900+ passed，0 failed）。
3. `npx eslint src` → error 总数 = 83（零新增）；warnings = 1367（零新增）。
4. `node scripts/gen-p65-exemptions.mjs --dry` 重跑输出与已写回锚点段一致（幂等复跑稳定）。
5. `npm run build` → 成功（新组件进产物无编译错误）。
6. 对照 ROADMAP 5 条 Success Criteria 逐条标注达成方式（SC1/T2 测试断言、SC2/T3 测试断言、SC3/T4 fixture、SC4/T5 三数对照、SC5/T5 基线文件），写入收口说明（可放 T5 基线 JSON 的 `sc_mapping` 字段或 commit 描述）。

<verify>
  <automated>cd ruoyi-react-frontend; npm run type-check</automated>
  <fails_when>非零退出码</fails_when>
  <automated>cd ruoyi-react-frontend; npm run test -- --run</automated>
  <fails_when>摘要出现 failed 大于 0 或 passed 数低于 3900 基线</fails_when>
  <automated>cd ruoyi-react-frontend; npx eslint src 2>&1 | Select-String "error"</automated>
  <fails_when>error 总数偏离 83 或 warnings 偏离 1367</fails_when>
  <automated>cd ruoyi-react-frontend; npm run build</automated>
  <fails_when>非零退出码</fails_when>
</verify>

预估：1h

---

## Wave 依赖总览

| Wave | 任务 | 产出 | 依赖 |
|---|---|---|---|
| 1 | T1 | 四象限审计 + 默认值定稿 | — |
| 2 | T2 | AppTextArea 组件 + 单测 + 出口 | T1 |
| 3 | T3 ∥ T4 | SearchForm textarea 字段 ∥ eslint 双 selector + 豁免预登记 | T2 ∥ T1 |
| 4 | T5 ∥ T6 | HYG-05 基线 → 收口验证 | T4 / 全部 |

## Open Questions（需用户拍板，不阻塞执行）

1. **默认 `autoSize` 与否**：locked 决策要求审计后定稿。T1 倾向「不下沉、显式传」（存量高度行为零外溢）；若用户希望多行框默认自适应高度，请在执行 T1 前明示，T1 将按「下沉」方向重新审计视觉外溢面。
2. **UI-SPEC 缺失**：ui_safety_gate 检出 frontend 信号（blocking）。本计划按契约基建（无新视觉面，复制 AppInput 先例）跳过 UI-SPEC 继续；如需补设计契约，先跑 `/gsd-ui-phase 70` 再执行。
3. **搜索区清除语义**（PITFALLS 1.2 受控耦合）：textarea 搜索字段点清空后 `onChange('')` 直通 filters（空串会被 `useTableManager.handleSearch` 的 filterEmpty 过滤，预期无害）。若用户要求「清空后不提交该参数」的更强归一化，应在 T3 执行前提出，改动点在封装层一次做完。

## Verification（phase 级）

- SC1：AppTextArea 零配置渲染带 clear icon（T2 测试 `ant-input-clear-icon` 断言）+ `allowClear={false}` 逃逸生效。
- SC2：SearchForm `type:"textarea"` 字段渲染 AppTextArea 且具备清空（T3 测试），无 custom.render 逃生口。
- SC3：fixture 成员表达式 + 解构形态各报 1 error；封装层/测试文件不报（T4）。
- SC4：selector 上线当天 `npx eslint src` error 总数 = 83（零新增），且 T5 三数对照无未解释差异。
- SC5：`70-baseline.json` 落盘，写明 Phase 71/74 引用路径与对账口径（T5）。
