# Phase 88-04 执行日志（VISFIX-04：antd 6 弃用清理 + C14 断言收口）

执行时间：2026-10-28 17:30–18:35｜环境：dev server :4000 + `$TEMP/p75-uat` sqlite 后端 :9000（admin/admin123）
未执行 git add / git commit（按约束）。

## 0. 落地顺序（PLAN 显式时序）

1. **清理先行**：Task 4.1 / 4.2 全部迁移落地 → grep 归零。
2. **C14 后落地**：断言写进 `scripts/css-contract-check.js`（已在 `lint:ci` 链尾）→ 落地即天然绿（PASS）。
3. **红绿验证**：人为还原 5 处旧 prop → C14 FAIL / exit 1 → 复原 → PASS / exit 0。

## 1. 各类弃用 prop 迁移前后计数对照（88-RESEARCH §1.5 实测口径）

| 弃用类 | before | after | 迁移规则 | 证据 |
|---|---|---|---|---|
| `maskClosable`（**antd 组件上**） | **22** | **0** | → `mask={{ closable: … }}` | 18×`<Modal maskClosable={false}>` + BaseEditModal 透传 + 1×`<Drawer maskClosable={!isExecuting}>` + 1×`<Drawer maskClosable>`（裸） + 1×`Modal.confirm` 对象字面量 |
| `maskClosable`（**我方冻结契约位**） | 7 | **7（保留）** | props 声明 2 + 解构默认 2 + FormModal→BaseEditModal 透传 1 + FormModal 调用方 2 | v1.22 契约：对外 prop 名不改 |
| `maskClosable`（注释） | 5 | 5（措辞同步） | — | `confirmDelete.ts:14` / `FormModal.ts:141` / `cache/index.tsx:9,418` / `backups/index.tsx:595` |
| `maskClosable`（测试） | 12 | 12（**输入侧保留 11 + 内部形态断言更新 1**） | 输入侧=对外契约；`confirmDelete.test.tsx:91` 改断言新内部形态 | 见 §3 |
| `Card bordered={false}` | **6** | **0** | → `variant="borderless"` | `components/network/MACEventsTimeline.tsx`×4 + `pages/network/mac/history/MACHistoryPage.tsx`×2 |
| `Table bordered={false}`（**不动**） | 7 | 7（保留） | Table 未弃用 | `components/operations/WorkstationDeviceTable/index.tsx` |
| `Descriptions bordered` / 业务字段 `bordered`（**不动**） | ≈16 / 2 | 不变 | 未弃用 / 非 antd prop | PLAN 防误伤条款 |
| `bodyStyle` | **4** | **0** | → `styles={{ body: {…} }}` | 全部是 Card：`ad-domain/ous/index_with_dept.tsx`×2 + `workstations/views/FloorPlanView.tsx`×2 |
| `destroyOnClose` | **6** | **0** | → `destroyOnHidden` | Modal×5 + Drawer×1（`FixSuggestionDetailDrawer.tsx:85`） |
| `size="middle"` | **17** | **0** | → `size="medium"` | Space×12 / Table×4 / Button×1（逐处确认枚举，见 §4） |
| `size="default"` | **1** | **1（豁免登记）** | Drawer 不支持 medium | `system/notice/components/StatisticsDrawer.tsx:22` |
| Select 六类（dropdownRender / dropdownClassName / popupClassName / onDropdownVisibleChange / dropdownMatchSelectWidth / showArrow） | 0 | 0 | **不立项** | 88-RESEARCH 实测 0 命中 |
| `onSearch` / `showSearch` | 271 / 340 | **不动** | antd 6.6.4 未弃用（关键决策） | — |

### grep 零命中证据（迁移后实测）

```
destroyOnClose=>0   bodyStyle=>0   headStyle=>0   maskStyle=>0   footerStyle=>0
contentWrapperStyle=>0   size="middle"=>0
dropdownRender=>0   dropdownClassName=>0   popupClassName=>0
onDropdownVisibleChange=>0   dropdownMatchSelectWidth=>0   showArrow=>0
size="default"=>1（StatisticsDrawer.tsx:22 —— Drawer 豁免）
bordered={false}=>7（全部 WorkstationDeviceTable 的 Table，逐行核对无 Card 命中）
```

新形态落点：`mask={{ closable … }}` **21 处 JSX**（+ 1 处注释示例）+ `mask: { closable: false }` **1 处**（`confirmDelete.ts:66`）= 22 处 antd 侧转场。

## 2. FormModal / BaseEditModal 对外签名冻结证据

| 组件 | 对外 props（**零变化**） | 内部转发（唯一 antd 面向点） |
|---|---|---|
| `src/components/shared/FormModal.tsx` | `maskClosable?: boolean;`（:51）+ 解构默认 `maskClosable = false`（:78） | `maskClosable={maskClosable}` → `<BaseEditModal>`（:146，**我方组件间转发，仍用冻结名**） |
| `src/components/modal/BaseEditModal.tsx` | `maskClosable?: boolean;`（:30）+ 解构默认 `maskClosable = false`（:46） | **`mask={{ closable: maskClosable }}`**（:63）→ `<Modal>` ← **全仓唯一 antd 面向转发点** |

- 调用方**零改动**：`monitor/cache/index.tsx:426`、`network/backups/index.tsx:604` 两处 `<FormModal maskClosable={false}>` 原样保留。
- 全仓 `maskClosable` 残留 26 行中，**触达 antd Modal/Drawer 的 = 0**；`maskClosable={` / `maskClosable:` 形态的 5 处命中全部是我方组件或测试 Harness。
- 行为等价实测（浏览器）：`/system/user` 新增弹窗 → 遮罩存在（`mask.visible=true`）、点遮罩**不关闭**（`mask.closable=false` 生效）。
- 冻结契约由 **C14b** 断言守护（对外 props 声明 + 内部 `mask={{ closable: maskClosable }}` 转发形态缺一即 FAIL）。

## 3. 测试断言同步（12 处 grep 命中逐行甄别）

| 文件:行 | 性质 | 处置 |
|---|---|---|
| `BaseEditModal.test.tsx:97` | 测试名 | 保留（描述对外 prop） |
| `BaseEditModal.test.tsx:99` | 对外输入 | **保留** |
| `confirmDelete.test.tsx:87` | 测试名 | 更新为新形态描述（`mask.closable=false`） |
| `confirmDelete.test.tsx:91` | **内部形态断言** | **更新**：`expect(opts.maskClosable).toBe(false)` → `expect(opts.mask).toEqual({ closable: false })` |
| `FormModal.test.tsx:29,39,55,200` | Harness 声明 / 对外输入 | **保留** |
| `FormModal.test.tsx:166,170,189,198` | 注释 / describe 名 / 测试名 | 保留（描述对外 prop，语义仍准确） |

**唯一需改的内部形态断言 = 1 处**（PLAN 预估 12 处全改 → 实际 1 处；其余 11 处是「输入侧」，按冻结契约必须保留，否则才是破坏对外签名）。

## 4. 逐处枚举确认（`size="middle"` → `"medium"`）

| 组件 | 处数 | antd 6.6.4 源码依据 | 视觉等价 |
|---|---|---|---|
| `Space` | 12 | `SpaceSize = SizeType \| number`；`_util/gapSize.js:4` `isPresetSize` 含 `middle`/`medium`；`space/style/index.js:59` `'&-gap-col-medium, &-gap-col-middle'` **同一条 CSS 规则** | ✅ 完全等价 |
| `Table` | 4 | `table/InternalTable.js:137` `customizeSize === 'middle' ? 'medium' : …` **运行时归一** | ✅ 完全等价 |
| `Button` | 1（`FloorPlanView.tsx:80`） | `button/Button.js:224` 仅对 `large`/`small` 加尺寸 class → `middle` 与 `medium` **同为默认尺寸** | ✅ 完全等价 |

`Card bordered={false}` → `variant="borderless"`：`form/hooks/useVariants.js:15-18` —— `legacyBordered === false` → `mergedVariant='borderless'`，与显式 `variant="borderless"` **同一分支、同一产物** ✅

`Card bodyStyle` → `styles.body`：`card/Card.js:93` 弃用映射 + `card/Card.d.ts:27-31` `styles?: { header?; body? }` ✅

`Drawer` 弃用同规则：`drawer/Drawer.d.ts:21`（destroyOnClose）、`:27`（maskClosable）均标 `@deprecated` ✅

## 5. C14 断言落地与红绿验证

### 实现（`scripts/css-contract-check.js`）

- 扫描范围：`src/**/*.{ts,tsx}`（**含测试文件**），先剥注释（保留行号/列号）再匹配。
- 弃用清单 14 项：`maskClosable` / `destroyOnClose` / `bodyStyle` / `headStyle` / `maskStyle` / `footerStyle` / `contentWrapperStyle` / `dropdownRender` / `dropdownClassName` / `popupClassName` / `onDropdownVisibleChange` / `dropdownMatchSelectWidth` / `showArrow` / `size="middle"` / `size="default"`。
  **`bordered` 不进正则**（Descriptions/Table 未弃用会大量误报）；**`onSearch` 绝不进清单**（关键决策）。
- `maskClosable` 走**上下文判定**（新增 JSX 标签扫描器 + 花括号深度分段）：
  - 作为属性名传给 antd `<Modal>` / `<Drawer>` → **FAIL**；
  - 写成对象字面量键（antd config）→ **FAIL**；
  - props 类型声明 / 解构默认值 / 我方组件与测试 Harness 的 JSX prop / 属性值里的 JS 引用 → 允许。
  （深度判定解决了 `mask={{ closable: maskClosable }}` 中 `maskClosable` 是**值引用**而非属性名的误报。）
- 豁免登记 4 项（`C14_EXEMPTIONS` 数组，含理由与证据行号）：
  1. `StatisticsDrawer.tsx` `size="default"` —— `drawer/Drawer.d.ts:6` `const _SizeTypes: readonly ["default","large"]`（**Drawer 无 medium 枚举**）+ `Drawer.js:90`（`"default"` → `DEFAULT_SIZE`）+ `Drawer.js:93-97`（其它字符串会变非法 width）。
  2. `FormModal.tsx` `maskClosable` —— v1.22 契约冻结。
  3. `BaseEditModal.tsx` `maskClosable` —— v1.22 契约冻结。
  4. `design-system/components/ColorSwitcher.tsx` `size="middle"` —— 本仓自研组件**自有 prop 枚举**（`ColorSwitcherProps.size?: "small"|"middle"|"large"` + 解构默认值），非 antd 弃用 prop：antd 6 `SizeType` 仍含 `"middle"`，Button/ColorPicker 无弃用告警，且 `middle` ≡ `medium` 渲染；改写=变更本组件对外契约（扩范围）。
- **C14b**（新增）：FormModal/BaseEditModal 冻结契约守护（对外 `maskClosable?: boolean` + 内部 `mask={{ closable: maskClosable }}` 转发形态，缺一即 FAIL）。
- 阻断形态：命中即计入 `failures` → exit 1（`lint:ci` 链尾自动带上）。

### 落地输出（绿）

```
PASS  C14  antd 6 弃用 prop 零命中（阻断）  → C14 PASS：弃用清单 0 命中，豁免登记 4 项
PASS  C14b FormModal/BaseEditModal 冻结契约签名保留（阻断）  → C14b PASS：对外签名与内部转发形态均未被改动
结果: ALL PASS     EXIT=0
```

### 红绿验证（人为还原 5 处旧 prop）

| 还原点 | C14 报告 | exit |
|---|---|---|
| `CaptchaModal.tsx:116` → `maskClosable={false}` | 弃用 prop「maskClosable」（仍传给 antd `<Modal>`） | 1 |
| `confirmDelete.ts:66` → `maskClosable: false` | 弃用 prop「maskClosable」（对象字面量 / antd config 仍用旧形态） | 1 |
| `index_with_dept.tsx:382` → `bodyStyle={{…}}` | 弃用 prop「bodyStyle」（→ styles.body） | 1 |
| `LocationAliasDrawer.tsx:223` → `destroyOnClose` | 弃用 prop「destroyOnClose」（→ destroyOnHidden） | 1 |
| `FloorPlanView.tsx:80` → `size="middle"` | 弃用 prop「size="middle"」（→ size="medium"） | 1 |

→ 5 处**全部命中**，`结果: 1 项 FAIL`，`EXIT=1`；复原后 `EXIT=0`（git diff 逐行复核复原字节一致）。

## 6. 浏览器 console 弃用警告实测（A/B 同页对照）

环境：dev :4000 + sqlite UAT 后端 :9000，admin/admin123。

| 阶段 | 页面 | console 弃用警告 |
|---|---|---|
| **BEFORE**（临时还原旧 prop，强制刷新） | `/monitor/cache`（FormModal → BaseEditModal → antd Modal） | **1**：`[antd: Modal] \`maskClosable\` is deprecated. Please use \`mask.closable\` instead.` |
| **AFTER**（复原后同 URL 强制刷新） | `/monitor/cache` | **0** |
| AFTER | `/system/user`（强制刷新） | **0** |
| AFTER | `/network/credentials`（清空 console 后加载） | **0** |

**M16 达成：antd 弃用警告 2 类 → 0**（验收口径 = 「无 deprecated 类警告」，**非**「零 Warning」）。

剩余 console 提示均为**清理范围外**的既有问题：
- `onSearch should work with showSearch`（rc-select 提示，PLAN 明确不迁 `onSearch`）；
- `Instance created by useForm is not connected to any Form element`（React Form 提示）。

## 7. 第二轮矩阵视觉 diff

- 方法：复用 `88-visual-evidence/matrix-fixture.js`（21 形态，注入 `.ant-app.css-var-_r_0_` **antd cssVar scope**）+ `matrix-sample.js`；删除 88-03 的 `after-*.json` 后**复采**，与 88-03 的 `before-*.json` 逐格比对。
- 结果：**360 格（21 形态 × 明暗 × carrier/inner × 5 属性），差异 0**（`matrix-diff-result.txt`：`单元格 360，差异 0`）。
- 采信自检：after 明暗内部差异 43 格（基线 before 同为 43）→ 暗色捕获真实生效，非恒等空跑。
- 第二轮截图（12 张）：`88-visual-evidence/r2-system_user|monitor_cache|network_mac|operations_workstations|ad-domain_ous|my-notices-{light,dark}.png`。

## 8. 闸门结果（实跑）

| 命令 | 结果 |
|---|---|
| `node scripts/css-contract-check.js` | **exit 0**，ALL PASS（24 项，含 C14 / C14b） |
| `npm run lint:ci` | **exit 0**（eslint 0 problems + 预算 + 豁免理由 + C14 全链） |
| `npm run lint` | **exit 0**，0 problems |
| `npm run type-check` | **exit 0** |
| `npm run test -- --run` | **exit 0**，**570 文件 / 3963 用例全过**（与 88-03 基线同数，无回归） |
| `npm run build` | **exit 0**，✓ built in 34.72s |
| 浏览器 console | 三页 deprecated 警告 **0** |
| 第二轮矩阵 diff | **360 格 / 差异 0** |

改动面：**46 个文件**（44 源文件 + `scripts/css-contract-check.js` + 1 个构建副产物），`330 insertions / 62 deletions`。

## 9. 偏离 PLAN 之处（逐条）

1. **FormModal.tsx:143 透传位未改成 `mask={{ closable }}`**（PLAN 字面要求）。
   理由：:143 是 `<BaseEditModal maskClosable={maskClosable}>` —— **我方组件之间**的转发，BaseEditModalProps 无 `mask` 字段；按 PLAN 字面改会①类型报错（`npm run type-check` 必红）②必须给 BaseEditModalProps **新增 `mask` prop** = 扩契约（PLAN 自己明令「不得擅自扩契约」）。
   处置：对外冻结名保留，**antd 面向转发收敛在 BaseEditModal.tsx:63 单点**（与用户指令「内部转发到 antd 新形态 mask={{ closable }}」一致）。
2. **PLAN 的 verify grep（`maskClosable={` / `maskClosable:`）仍命中 3 行**：`FormModal.tsx:146`、`cache/index.tsx:426`、`backups/index.tsx:604`。
   理由：这 3 行是**我方组件**的对外冻结 prop 调用位（不是 antd prop）—— 「调用方零改动」与「对外签名零变化」是硬约束，迁移它们=破坏冻结契约。故按语义而非字面执行；C14 用上下文判定精确区分（antd 组件上 = 0 命中）。
3. **测试同步实为 1 处断言**（PLAN 预估 12 处）：其余 11 处是「输入侧/声明位」，按冻结契约必须保留。
4. **`MACEventsTimeline.tsx` 实际路径是 `src/components/network/`**（PLAN frontmatter 写 `src/components/ui/` —— 该路径不存在），以 prop 名为锚定位，未新建文件。
5. **新增 `ColorSwitcher.tsx` 豁免**：PLAN 的 17 处计数口径 `grep 'size="middle"'` **漏掉**了 `size = "middle"`（解构默认值，带空格）形态；该处是本仓自研组件自有 prop 枚举，antd 6 未弃用 `'middle'` → 登记豁免、不迁（迁则变更本组件对外契约）。
6. **新增 C14b 断言**（PLAN 只列 C14）：把「v1.22 冻结契约」从注释变成可执行守护，否则契约会被后续改动无声破坏。
7. **C14 的 `maskClosable` 判定改为上下文感知**（PLAN 字面是「文本匹配 + 豁免数组」）：纯文本匹配无法区分「属性名」与「属性值里的 JS 引用」，会在 `mask={{ closable: maskClosable }}` 上自造假阳性；实现上补了 JSX 标签扫描器 + 花括号深度分段。
8. **`size="default"` 豁免登记**（PLAN 预留分支）：实测 `drawer/Drawer.d.ts:6` 枚举为 `["default","large"]` → Drawer **无 medium**，保留并登记证据行号（PLAN 的分支 2）。

## 10. 遗留 / 待后续（本 plan 未动，登记备查）

1. **`Drawer width` → `size` 弃用（PLAN 清单遗漏的一类）**：antd `drawer/Drawer.js:155` 弃用表含 `['width','size']`、`['height','size']`。
   实测浏览器仍报：`[antd: Drawer] \`width\` is deprecated. Please use \`size\` instead.`（`/network/mac`、`/ad-domain/computers` 各 1）。
   全仓 **9 处 / 8 文件**：`DashboardSettings.tsx:114(480)`、`WidgetEditor.tsx:117(520)`、`BulkWriteDrawer.tsx:239(720)`、`ad-domain/computers/index.tsx:573(720)`、`asset/reconciliation/exception-rules/index.tsx:470(780)`、`FixSuggestionDetailDrawer.tsx:85(720)`、`network/mac/index.tsx:342/346(300)、527(480)`、`workorder/periodic/templates/index.tsx:603(500)`。
   迁移规则（源自 `Drawer.js:84-103`）：`width={N}`（数字）→ `size={N}` **完全等价**（`isNumber(size)` 直返）；但 `height` 仅在 placement≠left/right 时生效 → **需逐处按 placement 核对**（本 plan 未做，登记为后续项）。
   ⚠️ 本 plan **未迁**：不在 PLAN 锁定清单内，且 9 处需逐处视觉核对（扩范围）。**建议作为 VISFIX-05 立项**。
2. `antd/es/config-provider/index.js:431` 的 `WarningContext` 聚合机制（`strict:false` 时弃用告警会合并成一条 `console.warn`）——本仓未设 `warning` prop，故走逐条 `console.error` 口径，实测有效。
3. 构建副产物：`internal/services/system/asset_columns_schema.json` 仅 `__generated__` 时间戳变化（内容零变化，先例见 88-03 的 `7e40dd4f`）。
