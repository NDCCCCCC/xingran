# Project Research Summary

**Project:** RuoYi-Next 运维管理系统（ruoyi-go-backend）
**Milestone:** v1.25「全站权限治理与样式收敛」
**Domain:** ① 操作级 RBAC 全站治理 ② 样式层收敛到 antd 6 variant ③ 权限模型三缺口 ④ Phase 77 视觉新发现 ⑤ room-devices 815 行拆分
**Researched:** 2026-10-28
**Scope:** **仅 ② 做定向研究**（用户裁定）。①③⑤ 依据本仓事实与 v1.24 既有实证 note，不做域外研究。
**Confidence:** **HIGH**（antd 6.6.4 本机 `node_modules` 源码实证，附 `文件:行号`；官方文档仅作辅证）
**详细文档：** [STACK.md](STACK.md)（771 行 · antd 6 机制与 API 语义）· [PITFALLS.md](PITFALLS.md)（1000 行 · 项目 CSS 冲突面与四批清单）
**归档：** v1.24 研究已归档至 `.planning/milestones/v1.24-research/`

---

## Executive Summary

本次研究只深挖了 ②「样式层收敛到 antd 6 variant」，产出两份基于 **antd 6.6.4 本机源码**的实证文档。结论比预期重要：**v1.24 登记的两条核心机制描述都是错的**（症状对、机制错），而 v1.24 的 CSS 契约脚本正在**锁死一条死规则**、动态断言脚本有一条**永远 FAIL** 的 fixture。这意味着如果按 v1.24 的机制理解去执行收敛，会修错地方、并把已有的假绿固化进新代码。

三条被证伪/修正的机制：

| v1.24 登记 | 实证结论 | 依据 |
|---|---|---|
| 「`darkAlgorithm` 不改写显式 token → 暗色白底」 | **机制不成立**。Map Token 在算法**之后**展开覆盖显式值（`colorBgContainer`/`colorBorder` 显式设**本就无效**）；真凶是 ① **组件级 token 在算法之后 merge**（`getComponentToken.js:28`）② **seed 层 `colorBgBase = cream.canvas(#F0ECE3)` 浅奶油** | `antd/es/theme/themes/default/index.js:30-45`；`@ant-design/cssinjs-utils/es/util/getComponentToken.js:28` |
| 「antd 规则特异性 +1 类 → 写在后面就能覆盖不成立」 | **机制不成立**。antd 6 默认 `hashPriority:'low'` → 生成 `:where(.css-xxx)`，**hash 特异性为 0** → 同特异性下后写者胜。`!important` 真因是「`border` 简写压掉长属性」+「antd 复合选择器达 (0,3,0)」 | `@ant-design/cssinjs/es/StyleContext.js:47`；`util/index.js:145` |
| 「antd 6 弃用警告 `onSearch→showSearch`」 | **不成立**。`onSearch` 在 antd 6.6.4 **未弃用**（`showSearch` 是「开启可搜索」的开关，不是替代品）。全包 `grep "onSearch" \| grep -i deprecat` **零命中** | `antd/es/select/index.js:217-225` 的 `deprecatedProps`；`select/index.d.ts` 的 `@deprecated` 清单 |

研究同时**坐实并扩充**了 v1.24 的 CSS 登记册：可整块删除批 **17 条**（含 **8 个从未被 import 的死文件 / 470 行**）、可降级为 token 引用批 **14 条**、必须保留并登记批 **11 条**、需重写而非删除批 **11 条**；并给出 M1–M16 量化基线（`!important` 全 `.css` **1435 → ≤480**、`index.css` **5632 行 → ≤3200**、暗色分区 **64 → ≤12**、hex **465 → ≤120**）。这些数字让 ② 从「感觉有债」变成「可验收的收敛」。

对 ①③⑤ 的关键结论来自本仓实测而非研究：严格权限中间件**全仓仅用 1 处**、`pkg/permission/config.go` 的动作映射表**零调用点**、`ApplyDataScope`/`ApplyDataScopeFromContext` **零调用点**（data_scope 是**从未被消费**而非「未被列表消费」）。这三条把 ①③ 的工程量与改造路径都钉死了。

---

## Key Findings

### ② antd 6 variant 机制（STACK.md）

- **variant 取值恰好 4 个**：`outlined` / `borderless` / `filled` / `underlined`（`config-provider/context.js:4`），默认 `outlined`（`form/hooks/useVariants.js:24`）；优先级 = prop > 旧 `bordered={false}` > `Form variant`(VariantContext) > 组件级配置 > 全局配置。`Input` / `TextArea` / `Search` / `Password` **全部支持** variant。
- **边框与焦点环的归属**：边框声明在「**持有 variant 类的那一个元素**」上 —— 有 affix 时是 `.ant-input-affix-wrapper`，无 affix 时是 `<input>` 自身；内层 input 被强制 `border:none` + `:focus{box-shadow:none!important}` → **焦点环画在 wrapper**（`:focus-within`）。真实 status 规则是 `.ant-input-outlined.ant-input-status-error:not(.ant-input-disabled)`（特异性 0,3,0），**不存在** `ant-input-affix-wrapper-status-error`（v1.24 教训正确）。
- **Select 6.6.4 已彻底重构**：**无 `.ant-select-selector`**（`@rc-component/select/es/` 全目录零命中；antd 自己的 `color-picker/style/input.js:42` 都是死规则）；边框在 `.ant-select` 根元素、**由 CSS 变量驱动**（`--ant-select-border-color` 等）；`.ant-select-selection-placeholder` 已改名 `.ant-select-placeholder`。
- **InputNumber 错误态「仍灰」不是 antd 缺陷**：antd 状态规则完整（status 类落 `.ant-input-number` 根）。是项目 `index.css:875` 的 `border: … !important` **简写**压掉 antd 的 `border-color` **长属性**，且项目只给 Input 域补了 status 规则（`:958-965`）**漏补 InputNumber**，另有暗色块 `:2297` 二次覆盖。

### ② 项目 CSS 冲突面（PITFALLS.md）

- **F-01 `--input-bg` 串色坐实并定位到根因行**：`index.css:4815` 的 `:root{--input-bg:#fafaf9}` 与 `:198` 暗色块的 `--input-bg:#0f2e1b` **同特异性 (0,1,0)**，后者在前 → 暗色下解析为浅色；连锁 `--input-bg-hover/focus/disabled` **只在暗色块定义** → 暗色下形成「底浅、悬停深」的**半明半暗混合态**；影响 **22 处** `var()` 引用。
- **F-02 8 个死文件**（`theme-styles.css` 97 行、5 个 widget css、1 个 `.less` 85 行、1 个 0 字节 `.less`）**从未被 import** → 风险最低、收益最高的一批。
- **F-04 暗色下 2 处 `var()` 引用「只在浅色定义」的变量** → 声明整条作废，暗色 placeholder 回落继承值；另有 `--theme-warning-text` / `--radius` **全局从未定义**。
- **F-05 hex 治理对 `.css` 近乎完全失明**：`eslint.config.js:132` 的 hex 禁令块 `files: ["src/**/*.{ts,tsx}"]` **不含 `.css`**；`index.css` 311 处 hex 只命中 denylist **2 处** → **99.4% 硬编码色不受任何门禁**；且 `scripts/check-hardcoded-colors.mjs` **当前 exit=1**（18 文件命中）且**未接入 `lint:ci`** → 现状是「有扫描器、无门禁」。
- **两个必须先修的契约陷阱**：
  - `scripts/css-contract-check.js` 的 **C3b 正在锁死 `.dept-tree-search` 这条死规则** → 删规则必须同步把 C3b 改为**反向断言**「该选择器不得存在」，同时保留 C3c（`.user-form-input…`）为正向断言
  - `scripts/css-contract-assertions.js` 的错误态 fixture 用了 antd 6 **不存在**的 class → **该断言永远 FAIL**（假红）
- **P-M12（对 phase 拆分是硬约束）**：把「删规则」与「改 token」拆到**相隔较远**的 phase 会制造**双份真相**（中间态既不靠旧规则也不靠新 token）。

### ①③⑤ 本仓实测（本次复核）

- `RequireStrictPermissions` 全仓**仅 1 处**（`system/users` 写子组）；其余 `RequirePermissions` 90 + `RequirePermissionsWithQuery` 8 = 133 处仍是 ANY-of 组级语义 → **① 的工作量集中在后端 133 处挂载的逐动作拆分**，而非新增机制。
- `pkg/permission/config.go` 的 `GetRoutePermissions()` / `GetPermissionByPath()` **零调用点** → **死表确认**，接线即可激活 67 条动作映射。
- `ApplyDataScope` / `ApplyDataScopeFromContext` **零调用点** → data_scope **从未被消费**（比 v1.24 的「未被列表消费」更彻底）；`DataScopePermission` 挂在 5 处（usersRead/usersWrite/depts/vms）但下游无一消费。
- `scripts/check-permission-codes.mjs` 已支持按模块扩展（`PERMISSION_CODES` 按模块键组织）→ **①③ 的校验基建已就绪**，只需加模块条目。
- `room-devices/index.tsx` **815 行**；`devices` 页已拆到 306 行，范式可直接复用。

---

## 两份研究的分歧裁定（Lead 独立复核后裁决）

两份研究在两处口径冲突，均已由 Lead 用源码/实跑独立裁定：

| 分歧 | STACK.md（R1） | PITFALLS.md（R2） | **裁定** | 依据 |
|---|---|---|---|---|
| **VIS-01 复位块（`index.css:937-945`）能否先删** | 批次 B 建议删除；批次顺序 B 在 C 之前 | **反对**——`:875` 的 `border:…!important` 是 important 声明，**胜过一切非 important 规则且与特异性无关**，只要它还在，删复位块立即恢复 309 个输入框的双框 | **R2 正确**。**批次 C（拆 `border` 简写）必须先于批次 B**，B 与 C 须同 commit 或 C 先落地 | CSS 规范：important 声明胜过普通声明，无视特异性。STACK 自身顺序若照搬会踩坑 |
| **antd 6 弃用警告是否含 `onSearch`** | `onSearch` **未弃用**（源码 `deprecatedProps` + 类型定义双向确认） | M16 列「2 类（`maskClosable` / `onSearch`）」，沿用 v1.24 口径 | **R1 正确**。全包 `grep "onSearch" \| grep -i deprecat` **零命中**；六个组件的 `deprecatedProps`（AutoComplete/Cascader/DatePicker×2/Dropdown/Select/TreeSelect）均不含它。R2 该处沿用未核验旧口径 | `antd/es/select/index.js:217-225`；`select/index.d.ts` |
| `!important` 计数 | 1359（`index.css` 口径） | 1359（`index.css`）/ 1435（全 `.css`） | **一致**，但存在 6 种历史口径 → 已由 P-m7 固定为 M1–M16，phase 报告统一用 M1 口径 | 两份独立实跑 |

**裁决带来的结构性后果**：样式线不能按「先删死规则、再改 token」的直觉排。正确顺序是 **C（拆简写）→ B（可评估删复位块）**，且 R1–R8 的 variant 交还、暗色 mega-cluster 合并**全部硬前置 F3 色板重构**（`AntdThemeBridge.tsx:83,86,122,130,164` 的显式 `cream.*` 常量是**模式无关**的，`darkAlgorithm` 改不动 → 先删覆盖 = 暗色变白块）。

---

## Implications for Roadmap

建议 **9 个 phase（84–92）**，四条线：

**线 A · 权限（①③）— 核心，量最大**

1. **Phase 84 · 权限模型三缺口（PMODEL-01~04）** — 零 UI 依赖、后端为主，可与所有前端线并行
   - Addresses: `data_scope` 消费（复用已存在但零调用的 `ApplyDataScopeFromContext`）、`GetUserPermissions` 过滤 `menu.status`、Step 2 语义文档化、迁移记录表/GORM 漂移登记
   - Avoids: 把 `data_scope` 消费塞进 ① 的后端改造 → 两条线改同一批 handler，冲突面翻倍
2. **Phase 85 · 后端动作级严格化（RBAC-01/03/04）** — 死表接线 + 133 处 ANY-of 逐动作拆分 + `system` 域 4 组
   - Avoids: 直接改 `router.go` 而不先接线映射表 → 无法验证「未登记路由 → 校验失败」
3. **Phase 86 · 前端操作入口三态门控铺开（RBAC-02）** — 76 页面，**工程量最大的单个 phase**
   - Avoids: 复用 v1.24 的 `useActionPermission` 而不扩 `ActionButton` 的 permission 字段 → 45 个文件各自造轮子
4. **Phase 87 · 动作级权限授予闭环（RBAC-05/06）** — F 型菜单播种 + 三层证据验收（DB 前后对照 / HTTP 403 / 前端组件测试），仿 v1.24 Phase 78-79 范式

**线 B · 样式（②④）— 八段式，顺序不可乱**

> R2 给出 Phase 0-7 的收敛顺序，Lead 采纳并与本里程碑 phase 编号对齐。每段自带**可验收态**，其中「视觉 diff = 0」是删除类改动的硬判据。

5. **Phase 88 · 工具与口径冻结 + antd 弃用清理（VISFIX-04 前半 + VISFIX-06）** — 口径固定为 M1-M16；单独清理 `maskClosable` 46 处等弃用（**独立进行**，避免 console 警告污染后续「无新增 console error」判定）；修 `css-contract-assertions.js` 的假红 fixture + 新增 C6-C15 断言
   - Avoids: 弃用警告与样式改动混在一批 → 无法区分「新引入的 console error」与「存量警告」
6. **Phase 89 · 零风险删死代码（STYLE-02 第 1 批 D1-D17 + VISFIX-03）** — 8 条死选择器规则 + 8 个死文件 555 行；**验收 = 视觉 diff 必须为 0**
   - 必做联动：`css-contract-check.js` C3b → 反向断言
7. **Phase 90 · 变量层与 token 真相源（VISFIX-01 + STYLE-04 前半）** — `--input-bg` 串色（第 3 处重定义）+ 暗色未定义变量（`--input-placeholder`）+ `--theme-warning-text`/`--radius` 全局缺失
8. **Phase 91 · 明暗色板重构（F3，STYLE-01 + VISFIX-02）** — **硬前置**：新增 `ruoyiBrand.dark` 常量组 + 去组件级写死 `colorBgContainer` + seed 层 `colorBgBase` 改按模式派生；**验收标准就是「视觉 diff = 0」**
   - Avoids: 在色板重构前删规则 → 把暗色白底固化进新规则（P-M12 双份真相）
9. **Phase 92 · Input 域交还 variant/token（STYLE-02 R1-R5 + STYLE-03 + VISFIX-05）** — **批次 C 与 B 同 commit**；`border` 简写拆长属性 → hover/focus 交还 `Input.hoverBorderColor`/`activeBorderColor`/`activeShadow`（`AntdThemeBridge.tsx:123-125` 已设）；圆角 6px→8px（**肉眼只差 2px，必须 computed-style 断言**）
10. **Phase 93 · Select / InputNumber / Form / Table / Modal 域交还（STYLE-02 R8 + VISFIX-04 剩余）** — Select 边框交还 CSS 变量机制（`--ant-select-border-color`）
11. **Phase 94 · 暗色 mega-cluster 一次性合并（STYLE-02 R6-R7）** — 431 条规则 / 503 `!important`，**最后做**，合并后暗色分区 64 → ≤12
12. **Phase 95 · room-devices 拆分 + 视觉回归验收（SPLIT-01 + STYLE-05）** — 815 → ≤350 行；明暗双主题 computed-style + 截图矩阵（复用 `.planning/notes/v1.24-visual-verification.md` 的 14 页巡检清单）

**并行区间**：88（工具口径）→ 89（删死代码）→ 90（变量层）→ 91（色板）→ 92（Input 域）→ 93（其它域）→ 94（暗色合并）→ 95（拆分+验收）**串行为主**（每步都改 CSS，天然冲突面大）；**权限线（84-87）与样式线（88-95）可并行**（后端 vs 前端 CSS，文件面小重叠）；90/91 可与 84-87 并行。

**Phase ordering rationale:**
- 84 先于 85/86：data_scope 消费改的是同一批 handler 查询，早做可避免二次冲突
- 85 先于 86/87：后端不放行时前端铺开无意义；87 的 F 型播种需要 85 的权限码先存在
- **样式线 88→95 严格串行**：每步都改 `index.css`，天然冲突面大；且存在三处不可逆序——① **F3 色板（91）必须先于 variant 交还（92/93）与暗色合并（94）**（显式 `cream.*` 是模式无关常量，先删覆盖 = 暗色变白块）② **批次 C 必须先于批次 B**（important 压过一切非 important 规则）③ **变量层（90）必须先于变体交还（92/93）**（否则 `var()` 未定义变量继续作废）
- 89（删死代码）紧邻 90/91 但独立成 phase：17 条删除**零行为变更**、验收判据是「视觉 diff = 0」，可独立验证与回滚
- 94（暗色 mega-cluster 431 条）**最后做**：它是体量最大、风险最高的一块，且必须等前面各域都交还 token 之后
- 95（拆分 + 视觉验收）最后：视觉验收须在样式收敛稳定后做，否则截图基线不可用
- **权限线 84-87 与样式线 88-95 可并行**（后端 vs 前端 CSS，文件面小重叠；仅 devices / room-devices 页面在 86/95 有交叠）

**Research flags for phases:**
- **Phase 89**：**需要**——每条删除前的「无害性证据」需浏览器命中数实测（静态 grep 只证伪存在性，不证命中数）；并须同步反转 C3b
- **Phase 91**：**需要**——暗色品牌中性色取值（`ruoyiBrand.dark` 现不存在）需设计决策；双 ConfigProvider 嵌套下「theme token 真实生效点」需实证（P-C2：项目 `ConfigProvider.tsx` 只做副作用同步、**不渲染 Provider**，真实 antd Provider 只在 `AntdThemeBridge.tsx:178`）
- **Phase 92**：**需要**——`borderless` 变体的 focus 行为（社区 issue #58232，MEDIUM）需实测；批次 B 可否删除须在批次 C 落地后**同 commit** 重估
- **Phase 93**：**需要**——Select 6.6.4 的 CSS 变量驱动机制与项目 `border` 覆盖的等价性需比对
- **Phase 94**：**需要**——431 条暗色 cluster 中是否有 antd token 无法提供的**唯一**样式，需逐条确认（否则会丢样式）
- **Phase 88**：**需要**——`antd migrate 5 6 --apply` 是否含自动修复项未验证（官方文档只给 3→4、4→5 示例）
- **Phase 84/85/86/87/95**：标准模式，likely 不需要额外研究

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| antd 6 variant 机制（STACK） | **HIGH** | antd 6.6.4 本机源码逐条 `文件:行号` 实证；官方文档仅辅证 |
| 项目 CSS 冲突面（PITFALLS） | **HIGH** | 全部计数来自实跑命令（`find/grep/wc`）；口径已在 P-m7 固定为 M1–M16 |
| ①③⑤ 本仓事实 | **HIGH** | 本次实跑复核（0 调用点 / 1 处严格权限 / 815 行 / 76 页面） |
| 浏览器实际渲染形态（`:where(hash)` / 命中数 / 暗色取值） | **MEDIUM** | 源码推导 HIGH，**未 DevTools 实测** → Phase 89/90 前置验证项 |
| 批次精确行数与 `!important` 收益 | **MEDIUM** | 估算，实施时逐条 diff 实测 |
| `Descriptions bordered` / `Table bordered` 弃用状态 | **MEDIUM** | 源码未 grep 到弃用告警，仍需 `antd info` 核对 |

---

## Gaps to Address

- **暗色品牌中性色取值未定**：`ruoyiBrand.dark` 常量组不存在，Phase 88 需先做设计决策（可用现有 `green[900]` / `onDark.*` 家族推导）
- **`!important` 计数有 6 种口径**（1435 / 1429 / 1359 …）→ 已由 P-m7 固定为 M1–M16，phase 报告统一用 M1 口径
- **v1.24 登记册与实测有 3 处行号/口径偏差**：`--input-bg` 实际在 `:4815`（记 4807）；antd 选择器「219 行」实为 798 行 / 748 条规则；分区「96」实为 97 → 已在本 SUMMARY 与 REQUIREMENTS 中更正
- **`antd migrate 5 6 --apply` 能力未验证**（Phase 91 前置验证，不阻塞规划）
- **未做实机 DevTools 验证**：`:where(hash)` 形态、13 处死选择器的浏览器命中数、暗色 `--input-bg` 解析值 → 均为 Phase 89/90 的**前置验证项**，不阻塞 phase 启动
