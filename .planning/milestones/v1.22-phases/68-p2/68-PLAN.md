---
phase: "68"
slug: "68-p2-consistency-debt-color-date-type"
type: plan
created: "2026-10-25"
planner: gsd-planner
---

# Phase 68 执行计划 — P2 一致性债务（颜色/日期/类型收敛）

**目标**：三项既有代码收敛到既有真相源——① 硬编码 hex 收敛到 design token（残留入豁免清单）+ eslint 裸 hex 机器守卫；② 状态标签统一走 `statusHelpers`（唯一落点 `src/components/shared/statusHelpers.tsx`）、日期展示统一走既有 `src/utils/datetime.ts` 三档（含 2 处 `YYYY-MM-DDTHH:mm:ss` bug 修复）；③ any B 类独立修 + A/C/D 分类登记；④ build 守卫 vendor-react gzip ≤ 776 kB 零回归。

**权威契约**：`.planning/phases/68-p2/68-UI-SPEC.md`（c2c4e9e3，checker approved）。任务书与本 PLAN 冲突时以 UI-SPEC 为准。

**并行关系（67）**：`67-roi/67-PLAN.md` 截至本 PLAN 撰写（2026-10-25）**尚不存在**。依 UI-SPEC 口径声明：所有依赖「67 收口后现状」的量（any-A 类清单、与迁移页重叠的 hex/日期收敛面）一律标 **「以 67 收口实测为准」**——executor 开工时须用本 PLAN / UI-SPEC §6 给出的亲测命令重跑一遍作为真实基线，不按本 PLAN 记载数字盲改。若 67 已收口且迁移页与 68 收敛面重叠，PR 顺序 67 先行（UI-SPEC §7.7）。

**验证基线**（开工重跑覆盖，本 PLAN 记载为 2026-10-25 快照）：
- `npm run test -- --run` → **3900 passed / 0 failed** 起点（不新增失败）
- `npm run lint` → 全仓 **83 errors** 存量；验证一律对照基线**数新增**，存量不动
- `npm run type-check` → 0 错误
- `npm run build && gzip -c dist/assets/vendor-react-*.js | wc -c` → **≤ 794,624 字节**（776 × 1024；Windows bash gzip 已验证可用）
- hex 基线（pages tsx 362 处/77 文件、全 src tsx 504 处/106 文件）、状态三元 76 处、日期 format 7 种（2 基准/5 非基准）、any 宽口径 53 处/36 文件——全部为 SPEC 快照，开工重跑为准

**硬边界**（locked，违反即任务失败）：
- **不新建日期 helper**：`src/utils/datetime.ts` 是唯一真相源，只收敛调用（含导出格式常量挂在该文件），不建第二份时间工具
- **statusHelpers 唯一落点 `src/components/shared/statusHelpers.tsx`**：不放 `src/utils/`——utils 内裸 `<Tag>` 会命中 Phase 65 裸组件拦截（shared/ui 豁免块不含 utils，checker 修订 ③）
- 禁止新增 `var(--theme-x, #hex)` 双写兜底（Phase 63 既定纪律）；本 phase 不清偿 theme-styles.css 存量双写（E5 dismissed，CSS 静态降级留待后续）
- 3D/Canvas/echarts/主题定义层 hex 走豁免登记，**不收敛**（UI-SPEC §1.2）；`IconSelect` 色板为数据型，不强制
- any 消除**只动 B 类**（独立修）；A 类不动（待 67 迁移顺带，仅登记）；C 类审慎修可选、每处须全量 build+type-check 回归；D 类登记豁免；不强推 `type-check:strict`
- 操作语境三元（下拉菜单「停用/启用」动作项）不属展示 helper 范畴，仅统一文案动词（「停用」非「禁用」），不改交互
- 本 phase **不新增任何 npm 依赖**（lint 用 eslint 内置规则）
- 全程**不 commit**；豁免清单（`hex-exemptions.md` / `any-exemptions.md`）放 `.planning/phases/68-p2/`，不进 src

**行号锚点纪律**：本 PLAN 全部 `文件:行号` 锚点为 2026-10-25 快照（67 迁移与并行改动会漂移行号），以文件内语义锚（函数名/文案字符串/selector 名）重新 grep 定位，不得按失效行号盲改。

## 任务概览与依赖

| 任务 | 内容 | 依赖 | 估时 |
|------|------|------|------|
| T1 | statusHelpers + 锚点改造 + 单测 | 无（先行） | 4h |
| T2 | 日期收敛 + 常量 + bug 修复 | 无 | 3.5h |
| T3 | hex 收敛分批 + 豁免登记 | T1 完成后可并行展开 | 8h |
| T4 | lint 裸 hex 块 + 三层豁免 | T3 首批收敛后 | 3h |
| T5 | any B 类修 + 分类登记 | 无（可与 T2 并行） | 3.5h |
| T6 | build 守卫 + 复扫总闸 + SUMMARY | 全部 | 2.5h |
| 合计 | | | **24.5h**（对齐 66 全程口径） |

---

## T1 — statusHelpers（shared 落点）+ tableHelpers/ModernTag 锚点改造 + 单测

> **先行任务**：helper → 锚点改造 → 全链路（type-check / test）走通后才展开 T3 的状态类收敛。UI-SPEC §2 全节为逐字契约。

**exact files**：
- 新建 `ruoyi-react-frontend/src/components/shared/statusHelpers.tsx`：按 UI-SPEC §2.1 逐字实现——
  - `export type StatusDomain = "enable" | "online" | "result" | "health" | "workstation" | "visible"`
  - `export function statusLabel(status: number, domain: StatusDomain): string`
  - `export function statusColor(status: number, domain: StatusDomain): "success" | "error" | "default" | "processing"`
  - `export function StatusTag({ status, domain }: { status: number; domain: StatusDomain }): JSX.Element`（内部 `<Tag color={statusColor(...)}>{statusLabel(...)}</Tag>`，未知值 → `"未知"` + `"default"`）
  - 文案唯一口径：enable「启用/停用」、result「成功/失败」、health「正常/异常」、online「在线/离线」；正向 `success`、负向 `error`；workstation 三态（0=空/1=占/2=维）；**visible 反转例外**（1=显示/0=隐藏）；未知 status 不得 crash
- 修改 `ruoyi-react-frontend/src/components/shared/index.ts`（追加导出，不重构既有结构）
- 修改 `ruoyi-react-frontend/src/utils/tableHelpers.tsx`：`createStatusColumn`（现 :146 `status === 0 ? "success" : "error"` + 「正常/停用」）内部改调 `statusHelpers`，签名与返回类型不变，**调用方零改动**
- 修改 `ruoyi-react-frontend/src/components/shared/ModernTag.tsx`：`renderStatusTag`（现 :74-93）内部改调 `statusHelpers`，`normalText/stopText` 参数语义保留（域固定 enable；「正常」默认文案按 UI-SPEC 收敛为 helper 口径输出）
- 新建 `ruoyi-react-frontend/src/components/shared/__tests__/statusHelpers.test.tsx`：6 域 × 全 status 值的 label/color 断言 + 未知值 + workstation 三态 + visible 反转

**read_first**：
- UI-SPEC §2 全节（API 契约 + 2.2 不一致实例清单 + 2.3 落地范围）
- `ruoyi-react-frontend/src/utils/tableHelpers.tsx:136-150`、`src/components/shared/ModernTag.tsx:69-94`（锚点现状）
- `eslint.config.js` Phase 65 块（确认 shared/** 豁免生效，StatusTag 内裸 Tag 不报错）
- `internal/models/captcha_background.go:19-20`（captcha-background 取值反转证据——本任务不改该模块，T3 时按「映射到反转域」处理，勿「修正」）

**acceptance_criteria**：
1. 6 域 × 已知值输出与 §2.1 契约逐字一致；未知值（如 enable 域传 2）→「未知」+ default，不 crash 不误报
2. `tableHelpers.createStatusColumn` 与 `ModernTag.renderStatusTag` 对外行为签名不变，调用方文件 0 改动
3. 「停用」Tag 色统一 `error`；文案统一去「已」前缀、去「禁用」写法（helper 输出面）
4. 全链路：type-check 0、statusHelpers 单测通过、全量 test 3900+ passed / 0 failed

**fails_when**：helper 落在 utils/**（65 lint 拦截即证明落错）；文案偏离唯一口径（保留「已启用」「禁用」「正常/关闭」混写）；visible 域未反转（显示写成 error 色）；锚点改造动了调用方文件或改了导出签名

**verification**：
```bash
cd ruoyi-react-frontend
npm run type-check
npx vitest run src/components/shared/__tests__/statusHelpers.test.tsx
npm run test -- --run   # 3900+ passed / 0 failed
```
预估：4h

---

## T2 — 日期收敛：12 处手写格式 → datetime.ts 三档 + 常量 + 2 处 T 分隔符 bug 修复

**exact files**：
- 修改 `ruoyi-react-frontend/src/utils/datetime.ts`：追加导出格式常量——`FORMAT_DATETIME_MINUTE = "YYYY-MM-DD HH:mm"`、`FORMAT_DATE_SHORT = "MM-DD"`、`FORMAT_TIME_MINUTE = "HH:mm"`（**只加常量，不改既有三档函数与 Z 剥离/空值"-"口径**）
- 修改手写 `dayjs(...).format(...)` 命中文件（SPEC §3.2 清单，开工重跑为准）：
  - `"YYYY-MM-DD"` 24 处 / `"YYYY-MM-DD HH:mm:ss"` 9 处 → 改调 `formatDate()` / `formatDateTime()`
  - `"YYYY年MM月DD日"` 4 处 → `formatDate()`（确属展示性场景保留的须登记豁免+理由，进 hex-exemptions 同目录豁免约定）
  - `"YYYY-MM-DD HH:mm"` 3 处 → `formatDateTime(x, FORMAT_DATETIME_MINUTE)`
  - `"YYYY-MM-DDTHH:mm:ss"` 2 处 → `formatDateTime()`（**bug 级**：T 原样渲染给用户；T6 验收证据单独列出这两处）
  - `"MM-DD"` 2 处 → `formatDate` 系常量 `FORMAT_DATE_SHORT`；`"HH:mm"` 2 处 → `formatTime` 或 `FORMAT_TIME_MINUTE`
  - 命中集中文件（收敛顺序）：`pages/operations/assets/index.tsx`(7)、`pages/duty/management/components/WeeklyView.tsx`(4)、`pages/duty/schedules/**`(7)、`pages/asset/reconciliation/exceptions/index.tsx`(3)
- 若触碰文件恰在 67 迁移列：随迁移顺带、跳过并标注「以 67 收口实测为准」；67 未收口时正常改（git 面窄、冲突自担风险已由 SPEC §7.7 说明）

**read_first**：
- UI-SPEC §3 全节（真相源契约 + 3.2 映射表）
- `ruoyi-react-frontend/src/utils/datetime.ts` 全文（29 行）
- SPEC §3.2 命中清单 + 开工重跑命令（§6 SC2 第二条）

**acceptance_criteria**：
1. 重跑 `grep -rEo "\.format\(['\"][^'\"]*['\"]\)"` 非 test 命中：仅剩 `"YYYY-MM-DD HH:mm:ss"`（helper 默认值）、`"YYYY-MM-DD"`、`"HH:mm:ss"` 及登记豁免项——**非基准格式（年月日/HH:mm/MM-DD/T 分隔符）散落字面量清零**
2. 2 处 `YYYY-MM-DDTHH:mm:ss` 改为用户可见正确格式，页面渲染无 `T` 分隔符
3. 所有日期展示调用经 `formatDateTime/formatDate/formatTime`（杜绝绕过空值/剥 Z 的 dayjs 直调）
4. datetime.ts 原有三档函数签名与行为零变化（Z 剥离、空值 "-"）
5. type-check 0、全量 test 0 failed

**fails_when**：新建第二份日期工具或把常量散落到调用文件；改动 datetime.ts 既有函数行为/签名；把「确为展示性场景」直接砍掉而非登记豁免；触碰 67 迁移中页面造成语义冲突

**verification**：
```bash
cd ruoyi-react-frontend
grep -rEo "\.format\(['\"][^'\"]*['\"]\)" --include="*.tsx" --include="*.ts" src | grep -v "__tests__\|\.test\." | sort | uniq -c
npm run type-check && npm run test -- --run
# T 分隔符清零（非 test）：
grep -rn "YYYY-MM-DDTHH:mm:ss" --include="*.tsx" --include="*.ts" src | grep -v "__tests__\|\.test\." | wc -l   # 0
```
预估：3.5h

---

## T3 — hex 收敛分批（pages 362 处/77 文件按域分批）+ hex-exemptions.md 豁免登记

> 主体工作量最大，按目录域分批推进，每批收敛后跑定向 lint + type-check。所有非 DOM 场景走豁免登记而非强改。
>
> **分批方法（checker 修订，防 executor 自由发挥）**：每批文件清单**机械生成**——用 SPEC §6 SC1 grep 重跑输出，按目录域分组切批：① 豁免登记批（3D/CAD/echarts/IconSelect，只登记不改动）→ ② 普通 CSS 批（index.css/App.css/DashboardView.css）→ ③ `pages/**` 常规批 → ④ `components/**` 常规批；每批 ≤15 文件 / ≤1.5h，批清单（文件路径 + hex 命中数）作为附录追加进 `hex-exemptions.md`；67 重叠迁移页在重跑输出里剔除后标注「以 67 收口实测为准」。禁止跨批自由挑选文件。

**exact files**：
- 新建 `.planning/phases/68-p2/hex-exemptions.md`：表头 `| 文件路径:行号 | hex | 场景 | 豁免理由（1.2 判定 a/b/c） |`，逐条登记
- 修改收敛面（每批一批，按 UI-SPEC §1.2 目录分类默认处置）：
  - **豁免（结构性）**：`src/design-system/themes/**`、`src/design-system/tokens/**`（含 `echartsTheme.ts`）——只登记不改动
  - **豁免候选→登记**：`pages/operations/building-spaces-3d/**` 中 Three.js/WebGL/地图着色部分（HubeiMap/HubeiMapGL/BuildingView3D/FloorView3D/BuildingModel3D/utils）；`components/cad-elements/**`、`components/cad-editor/**`、`components/shared/FloorPlanEditor.tsx`、`pages/operations/workstations/views/FloorPlanView.tsx`（Canvas 2D）；echarts 配色数组（`components/network/MACHeatmapChart.tsx`、`pages/network/mac/heatmap.tsx`、dashboard 图表 widget——前提统一从 `ruoyiBrand`/`echartsTheme.ts` 读值）；`components/IconSelect/index.tsx`（数据型色板，1.1-2 不强制）
  - **例外必须收敛（豁免目录内）**：antd Tag/style 内 hex，如 `BuildingModel3D.tsx:170` `#ba3630`；HubeiMap 状态 Tag（T1 完成后直接换 `StatusTag`）
  - **收敛主体**：`src/index.css`、`App.css`、`components/dashboard/DashboardView.css` 等普通 CSS（随主题 → var，品牌定值 → 注释标明来源）；其余 `pages/**`、`components/**` 常规样式 hex（362 处/77 文件 pages 面，迁移页与 67 重叠部分「以 67 收口实测为准」跳过/顺带）
- 状态语境 hex（red/green 预设色名与 hex）优先改用 T1 `statusHelpers` / 语义 token（`success/error`），而非逐处换 var

**read_first**：
- UI-SPEC §1 全节（三问判定 + 1.2 豁免标准 + 目录分类表）
- `src/design-system/tokens/colors.ts`（ruoyiBrand 语义组，逐字对照换算目标 token）
- 开工重跑 SPEC §6 SC1 grep 建真实基线（对照 67 收口现状）

**acceptance_criteria**：
1. 重跑 `grep -rEn "#[0-9a-fA-F]{6}\b" pages/ components/`（非 test）：每条残留都能在 `hex-exemptions.md` 找到对应行——**豁免清单外命中 = 0**
2. 每条豁免满足 1.2 三条件（非 DOM 渲染输入 + 不随主题语义 + 收敛反破坏效果），逐行有理由
3. 收敛写法正确：随主题 → `var(--theme-*)`；品牌定值 → `ruoyiBrand` 常量；无新增双写兜底
4. 状态色语境不残留 `color="green"/"red"` 预设与 hex（对照 SPEC 2.2 清单）
5. 每批后：type-check 0、定向 lint 无新增 error、全量 test 0 failed

**fails_when**：3D/Canvas/echarts 豁免场景被强收敛导致视觉破坏；把「固定色」误改成「随主题色」（或反之）造成双主题回归（light/dark 抽查 user 列表、monitor/server、dashboard、building-spaces-3d）；豁免清单缺行或理由缺失；新增 `var(--theme-x, #hex)` 双写；触碰 captcha-background 时把 `status===1?"success"` 当 bug「修正」（checker 已核非 bug，`internal/models/captcha_background.go:19-20` 取值反转，映射反转域处理）

**verification**：
```bash
cd ruoyi-react-frontend/src
grep -rEn "#[0-9a-fA-F]{6}\b" --include="*.tsx" --include="*.ts" pages/ components/ | grep -v "__tests__\|\.test\." | wc -l
# 逐条对照 .planning/phases/68-p2/hex-exemptions.md；清单外 = 0
cd .. && npm run type-check && npm run test -- --run
```
预估：8h

---

## T4 — lint 裸 hex 拦截块 + 三层豁免（与 65 同构）

> T3 首批收敛落地后再上规则，避免规则先到、收敛未完导致海量新增 error。

**exact files**：
- 修改 `ruoyi-react-frontend/eslint.config.js`：在 Phase 65 块之后、prettier 之前**新增独立 config 块**（勿并入既有块——flat config 同名 rules 键互相覆盖，既有注释已明示）：
  ```js
  // ===== Phase 68 DEBT-01 裸 hex 拦截 =====
  {
    files: ["src/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-syntax": [
        "error",
        {
          selector: "Literal[value=/#[0-9a-fA-F]{3,8}\\b/]",
          message:
            "禁止裸 hex 颜色字面量，请使用 var(--theme-*) 或 @/design-system/tokens/colors 的 ruoyiBrand；豁免须 eslint-disable-next-line 并注明原因（Phase 68 DEBT-01）",
        },
      ],
    },
  }
  ```
  selector 已由 checker 经 node+Linter 实测可解析、命中 hex 字符串字面量、不误伤普通字符串与 `0x` 数值字面量（SPEC checker sign-off）。
- 三层豁免（与 65 机制同构）：
  1. **目录 override**：`design-system/**`、`**/__tests__/**`、`**/*.test.*` 块内关闭本条（测试 override 若为「关整条 no-restricted-syntax」形态则天然覆盖，需亲核 65 落地后实际结构再决定加不加）；
  2. **逐行豁免**：真实非 DOM 场景 `// eslint-disable-next-line no-restricted-syntax -- <1.2 判定理由>`，同步登记 hex-exemptions.md；
  3. **存量文件豁免 override**：收敛完成后仍有整文件级合理场景（如 cad-editor）→ 文件列表 override 收口，首行注释 `// P69+ 清偿后移除`
- 若需脚本化生成存量豁免清单：新建 `ruoyi-react-frontend/scripts/gen-p68-exemptions.mjs`（65 有 `gen-p65-exemptions.mjs` 同类先例；可选，手列亦可）

**read_first**：
- `ruoyi-react-frontend/eslint.config.js` 全文（496 行；65 块结构 :108 起，测试 override 与 prettier 位置亲核）
- `.planning/phases/65-contracts/65-PLAN.md` T2（同构实现注记：独立块 vs 重声明数组两方案，以 AC 为准不锁实现）

**acceptance_criteria**：
1. 新写含裸 hex 的业务测试文件外 tsx → lint 报 error 且 message 指向 var/ruoyiBrand（临时 fixture 验证后删除）
2. `design-system/**`、测试文件内裸 hex 不报错（豁免生效）
3. 全量 lint：**errors 总数 = 83 存量 + 0 新增**（对照 T3 收敛后基线）
4. 含 eslint-disable 行的场景与 hex-exemptions.md 一一对应
5. 既有 65 块、内网 IP 块行为不变（65 定向验证不回归）

**fails_when**：新块并入既有 config 块导致规则互相覆盖；目录 override 连带关闭 65 契约块或内网 IP selector；豁免文件清单遗漏导致 lint 有新增 error；override 位于 prettier 之后

**verification**：
```bash
cd ruoyi-react-frontend
npx eslint src 2>&1 | tail -3      # 总 error 数 = 83 存量 + 0 新增
# fixture 试拦（临时文件，验证后删除）：
npx eslint src/components/__lint_probe_hex__.tsx
npm run lint                        # 对照基线数新增
```
预估：3h

---

## T5 — any B 类独立修 + any-exemptions.md 分类登记（A/C/D）

**exact files**：
- 新建 `.planning/phases/68-p2/any-exemptions.md`：表头 `| 文件路径:行号 | any 形式 | 豁免理由 | 登记人/日期 |`
- **B 类直接修**（补类型/泛型收窄，SPEC §4.1 清单开工重跑为准）：
  - `pages/monitor/logs/**`（13 处，主战场；日志页不在 67 迁移列）
  - `pages/operations/rpa/executions/ExecutionDetailModal.tsx`(3)、`pages/operations/floors/utils.ts`、`pages/ad-domain/{users,groups}/index.tsx`、`pages/duty/schedules/index.tsx`
- **A 类只登记不动**：`pages/network/{devices,backups,discoveries,credentials,command,executions,templates,mac}/index.tsx`、`pages/monitor/{server,cache}/index.tsx` 等 FormModal 迁移路径 any——理由「待 67 迁移顺带消除」（以 67 收口实测为准重列清单）
- **C 类审慎修（可选）**：`lib/api.ts`(3)、`components/shared/{FileUpload,ExcelImport,ExcelExport}.tsx`、`utils/tableHelpers.tsx`——每处修后须全量 build + type-check 回归；`lib/api.ts` 优先 `unknown` + 收窄，不做激进重类型化
- **D 类登记豁免**：`types/externals.d.ts`(2，声明文件)、`pages/system/settings/default-theme.tsx`(2，已在 eslint 既有豁免列表)

**read_first**：
- UI-SPEC §4 全节（分类 + 豁免格式 + 4.3 验收等式）
- 开工重跑 `grep -rEn ":\s*any\b|as any\b" --include="*.tsx" --include="*.ts" src | grep -v "__tests__\|\.test\."` 建真实基线（对照 67 收口）

**acceptance_criteria**：
1. SPEC §4.3 验收等式成立：残留 any 总数 == 0 **∪**（残留数 == any-exemptions.md 登记行数 且每行有理由）
2. A 类 0 改动、仅登记且理由为「待 67 迁移顺带」；D 类登记豁免
3. B 类修法为补类型/泛型收窄/`unknown`，不引入新的类型断言绕过（`as unknown as X` 双跳视同未修）
4. `npm run type-check` 0 错误（不强推 type-check:strict）；全量 test 0 failed
5. （可选非验收项）`@typescript-eslint/no-explicit-any` 锁 warn 数不增长

**fails_when**：A 类被单独强修（与 67 迁移冲突面失控）；为清数字把 any 换成 `as never`/`as unknown as` 类假收敛；C 类修后未跑全量回归；type-check 引入新错误

**verification**：
```bash
cd ruoyi-react-frontend
npm run type-check
grep -rEn ":\s*any\b|as any\b" --include="*.tsx" --include="*.ts" src | grep -v "__tests__\|\.test\." | wc -l
# 对照 any-exemptions.md 登记行数；等式成立即 pass
npm run test -- --run
```
预估：3.5h

---

## T6 — build 守卫 + 复扫总闸 + SUMMARY（不 commit）

**exact files**：
- 新建 `.planning/phases/68-p2/68-SUMMARY.md`：四 SC 逐条验收证据（含重跑基线数字 vs 收口数字对照、T 分隔符 2 处 bug 修复前后证据、豁免清单行数、lint error 对比）
- 不修改任何源码（守卫任务；复扫不过则回对应任务返工）

**read_first**：
- UI-SPEC §5（build 守卫契约）+ §6（全部验证命令）
- `vite.config.ts:140` manualChunks「未知包默认归入 vendor-react」兜底规则（超限回查入口）

**acceptance_criteria（SC 逐条，全部以开工重跑基线为对照）**：
1. **SC1**：`grep -rEn "#[0-9a-fA-F]{6}\b" pages/ components/`（非 test）残留逐条对应 hex-exemptions.md，清单外 = 0；eslint hex 块 0 新增 error
2. **SC2**：状态三元 Tag 语境收敛完成（对照 SPEC 2.2 清单逐条核销）；非基准日期格式字面量清零（豁免项除外）；2 处 T 分隔符 bug 修复有前后证据
3. **SC3**：any 验收等式成立（残留 == 0 ∪ == 豁免行数）；`npm run type-check` 0
4. **SC4**：`npm run build` 通过且 `gzip -c dist/assets/vendor-react-*.js | wc -c` **≤ 794,624**；超限回查是否误引入依赖进 vendor-react 兜底
5. **回归底线**：`npm run test -- --run` 3900+ passed / 0 failed；`npm run lint` errors = 基线 + 0 新增；T 格式 0 残留（`grep -rn "YYYY-MM-DDTHH:mm:ss"` 非 test = 0）
6. 双主题人工抽查记录写入 SUMMARY：user 列表、monitor/server、dashboard、building-spaces-3d 在 light/dark 下无「暗色白块/亮色隐形描边」类回归
7. SUMMARY 明确「不 commit」；变更集待用户确认后另行提交

**fails_when**：任一 SC 复扫不过；vendor-react gzip 超 794,624 字节且回查不出归属；lint 新增 error 未归零即宣告完成；SUMMARY 伪造未实际跑过的验证结果

**verification**：
```bash
cd ruoyi-react-frontend
npm run build && gzip -c dist/assets/vendor-react-*.js | wc -c   # ≤ 794624
npm run type-check && npm run test -- --run && npm run lint
# 复扫三闸（SC1/SC2/SC3 命令见各任务 verification，全部重跑）
```
预估：2.5h

---

## 验收口径汇总

| SC | 判据 | 命令 |
|----|------|------|
| SC1 | 豁免清单外裸 hex = 0 + lint hex 块 0 新增 | SPEC §6 SC1 |
| SC2 | 状态/日期同一 helper；非基准格式清零；T bug 修复 | SPEC §6 SC2 |
| SC3 | any 等式成立 + type-check 0 | SPEC §6 SC3 |
| SC4 | build 通过 + vendor-react gzip ≤ 794,624 | SPEC §5 |
| 底线 | test 3900+/0、lint 基线+0、不 commit | 本 PLAN 各任务 verification |
