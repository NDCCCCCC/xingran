---
gsd_state_version: "1.0"
milestone: v1.25
milestone_name: 全站权限治理与样式收敛
current_phase: 84
current_phase_name: 权限模型三缺口
status: planning
stopped_at: v1.25 roadmap CREATED（Phases 84-93，22/22 REQ mapped，Traceability 已回填 `.planning/REQUIREMENTS.md`）
last_updated: "2026-10-28T06:48:20.510Z"
last_activity: 2026-10-28
last_activity_desc: v1.25 roadmap CREATED（Phases 84-93）
state_head: bdaca98816d430be9f6f04461da09258c914b652
progress:
  total_phases: 10
  completed_phases: 0
  total_plans: 4
  completed_plans: 0
  percent: 0
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.25 milestone planning（路线图已创建，待执行）
**Last activity**: 2026-10-28 — v1.25 roadmap CREATED（Phases 84-93，22/22 REQ mapped）
**Prior activity**: 2026-10-28 — v1.24 全量执行完成（Phase 77 视觉验证 + Phase 83 拆分收口 + VIS-03 死规则修复）
**Prior activity (2)**: 2026-10-28 — v1.24 执行期：78 权限双端闭环（后端严格中间件 + 前端三态门控）+ 79 PG 实机验收（三层证据）+ 82 lint 清零
**Prior activity (3)**: 2026-10-28 — v1.23 SHIPPED & ARCHIVED（Phases 69-75, 22/22 REQ satisfied）
**Prior activity (4)**: 2026-10-27 — v1.23 roadmap CREATED（Phases 69-75, 22/22 REQ mapped）

## Current Milestone: v1.25 全站权限治理与样式收敛 (Site-wide Permission Governance & antd 6 Style Convergence)

**Core Value**: 把 v1.24 只在 `/system/users` 单域完成的「操作级 RBAC 前后端双端闭环」推广到**全站 76 页面 / 134 处后端路由挂载**，并把样式层从手写 `!important` 规则收敛到 **antd 6 官方 variant 机制** —— 让「谁能做什么」在前后端同一套语义下成立，让「页面长什么样」由 antd 6 的 theme token 单一决定。

**事实基线**:
- `.planning/REQUIREMENTS.md`（**22 REQ**：RBAC×6 / STYLE×5 / PMODEL×4 / VISFIX×6 / SPLIT×1；每条含实测依据与量化验收指标）
- `.planning/milestones/v1.24-REQUIREMENTS.md`（v1.24 收档 Known Gaps 五项候选原文 → 本里程碑立项来源）
- `.planning/research/{STACK,PITFALLS,SUMMARY}.md`（② 定向研究：antd 6.6.4 源码实证 + 项目 CSS 冲突面盘点 + 四批规则清单 + M1-M16 基线 + 两份研究分歧裁定）
- `.planning/notes/v1.24-css-audit-register.md`（36 条 CSS 登记）｜`.planning/notes/2026-10-28-pg-permission-baseline.md`（#16 定责）｜`.planning/notes/v1.24-visual-verification.md`（21 形态 × 明暗矩阵 + 17 张证据）

**起点基线（Lead 实跑复核，非转述）**: 前端业务页面 76 个（仅 14 个接入 `useMenuStore`）｜后端路由组 76 个 / `RequirePermissions` 90 + `RequirePermissionsWithQuery` **8** + `RequireStrictPermissions` **1** + `RequireAllPermissions` **0** = **134** 处挂载｜`pkg/permission/config.go` 动作映射表**零调用点**｜`ApplyDataScope` **零调用点**（data_scope 从未被消费）｜CSS `!important` 全 `.css` 1435 / `index.css` 1359｜`room-devices/index.tsx` **815 行**｜`index.css` **5632 行**

> ⚠️ **数字冲突裁定（2026-10-28 roadmap 阶段，Lead 实跑复核）**：roadmapper 曾上报「`RequirePermissionsWithQuery` 7 处 / `room-devices` 791 行 / `index.css` 5569 行」三项偏差，经 `wc -l` 与 `grep -c` **逐项实跑复核，三项均为误测**——正确值是 **8 / 815 / 5632**（与 `.planning/REQUIREMENTS.md` 起点基线表、研究文档 PITFALLS.md 一致），已据此更正 STATE.md 与 ROADMAP.md 的全部引用。**需求条目数 22 条则确认正确**（6+5+4+6+1=22；原写「26 total」为需求定稿阶段的算术笔误，磁盘上不存在第 23-26 条）。

### v1.25 执行期必须守住的硬约束

**三处不可逆顺序约束（写死，不可交换）**
1. **Phase 89（F3 色板）必须先于 90 / 91 / 92** —— `AntdThemeBridge.tsx:83,86,122,130,164` 的显式 `cream.*` 是**模式无关**常量，`darkAlgorithm` 改不动 → 先删覆盖 = 暗色变白块
2. **批次 C（拆 `border` 简写）必须先于批次 B（删 VIS-01 复位块）**，且二者**同 commit** —— `:875` 的 `border:…!important` 是 important 声明，**胜过一切非 important 规则且与特异性无关**；只要它还在，删 `:937-945` 复位块立即恢复 309 个输入框的双框
3. **变量层（89 内先做）必须先于变体交还（90 / 91）** —— 否则暗色下 `var()` 引用未定义变量继续作废（暗色 placeholder 整条声明失效）

**零新依赖** —— 沿用 v1.23 / v1.24 原则；不引 stylelint / Playwright / jscodeshift；视觉验证用现有 computed-style 断言 + agent-browser。

**回归基线（每 phase 结束必须复核）** —— eslint **0 problems** / vitest 569 文件 3960 用例全绿 / vendor-react gzip **≤ 776 kB** / `go build ./...` + `go test` 全绿。

**不破坏 v1.22 冻结契约与 v1.24 已交付修复** —— v1.22 公共组件契约（`AppInput` / `AppSelect` / `AppTextArea` 默认值、`ConfirmDelete` 唯一入口、`handleApiError`、`statusHelpers`、`PageTable` / `FormModal`）与 v1.24 的 **VIS-01**（双框消除）/ **VIS-02**（焦点环单圈）/ **VIS-03**（错误态边框色）均为已验收资产；STYLE-01 涉及 `AntdThemeBridge` 色值真相源（v1.22 冻结面），须**单独登记与验收**。

**机制更正（v1.25 研究期源码实证，勿按 v1.24 旧口径实施）** —— ① 「`darkAlgorithm` 不改写显式 token」机制**不成立**：Map Token 在算法**之后**展开覆盖显式值，真凶是**组件级 token 在算法后 merge** + **seed 层 `colorBgBase = cream.canvas(#F0ECE3)` 浅奶油**；② 「antd hash 前缀使特异性 +1 类」**不成立**：antd 6 默认 `hashPriority:'low'` → `:where(.css-xxx)`，**hash 特异性为 0** → 同特异性下**后写者胜**，`!important` 的真因是「`border` 简写压掉长属性」+「antd 复合选择器达 (0,3,0)」；③ 「`onSearch→showSearch` 弃用」**不成立** —— antd 6.6.4 的 `deprecatedProps` 与类型定义双向确认 `onSearch` **未弃用**，**不得迁移**。

**其他约束** —— 门控取三态而非纯 fail-closed（空权限集与未加载在数据上不可区分）；生产数据只读（验收夹具须单事务清理、既有行零改动）；Phase 91/92 允许的视觉差异必须**逐条登记**，零未登记项。

## Current Position

Phase: 84 (权限模型三缺口) — READY TO EXECUTE
Plan: —
Status: 路线图已创建待执行（ROADMAP Phases 84-93，22/22 REQ mapped，Traceability 已回填）
Last activity: 2026-10-28 — v1.25 roadmap CREATED（Phases 84-93）

### v1.25 Phase 依赖速查

| Phase | 主题 | 硬依赖 | 需求 | 关键产出 |
|-------|------|--------|------|---------|
| 84 | 权限模型三缺口 | — | PMODEL-01~04 | `data_scope` 被列表消费；`GetUserPermissions` 过滤 `menu.status`；Step 2 语义命名固化；迁移记录表 + GORM 漂移登记 |
| 85 | 后端动作级严格化 | 84 | RBAC-01/03/04 | 动作映射表接线（零调用点 → 判定链路）；133 处 ANY-of 写路由逐动作拆分；`system` 域 4 组（roles/menus/depts/posts）严格化 |
| 86 | 前端操作入口三态门控铺开 | 85 + 84 | RBAC-02 | 76 / 76 页面接入 `useActionPermission` 三态门控（起点 14 / 76） |
| 87 | 动作级权限授予闭环 | 85 + 86 | RBAC-05/06 | 动作级 F 型菜单播种；DB / HTTP / 前端三层证据 + 决定性对照；权限码校验接入 CI 门禁 |
| 88 | 验证底座与零风险清理 | — | VISFIX-03/04/06（承接 STYLE-02 第 1 批 D1-D17） | 假红 fixture 修好（`passed === true`）；C3b 反转；C6-C12/C14/C15 落地；死规则 + 死文件清零；**视觉 diff = 0**；antd 弃用清零 |
| 89 | 变量层与明暗色板重构 | 88 | STYLE-01/04 + VISFIX-01/02（承接 STYLE-02 第 2 批 T1-T14） | `--input-bg` `#fafaf9`→`#0f2e1b`；暗色 Select 不再白底；新增 `ruoyiBrand.dark`；`colorBgBase` 按模式派生；hex 门禁覆盖 `.css` 并接入 `lint:ci` |
| 90 | Input 域交还 variant/token + 圆角统一 | 89 | STYLE-03 + VISFIX-05（承接 STYLE-02 R1-R5 + 批次 B） | 批次 C 与 B 同 commit；InputNumber 错误态 `rgb(219,215,206)`→`rgb(186,54,48)`；圆角 computed-style = 8px；双框数 = 0 |
| 91 | Select / InputNumber / Form / Table / Modal 域交还 | 90 | 承接 STYLE-02 R8 + VISFIX-04 剩余（主责见 92 / 88） | Select 边框交还 `--ant-select-border-color`；`.module.css` 6 条 antd 选择器迁出；Select 域六类弃用 prop 清零 |
| 92 | 暗色 mega-cluster 合并 | 91 | STYLE-02（主责） | 431 条暗色规则整块清零 + 逐条等价性确认；M1 1435→≤480 / M4 →≤150 / M5 →≤12 / M6 → 0 / M9 →≤3200 |
| 93 | room-devices 拆分 + 视觉回归验收 | 92 | SPLIT-01 + STYLE-05 | 815 → ≤350 行、每新文件 ≤250；computed-style + 截图矩阵双测；VIS-01/02/03 三项断言全绿 |

**并行区间**：线 A 权限 `84 → 85 → 86 → 87` ∥ 线 B 样式 `88 → 89 → 90 → 91 → 92 → 93`（两条线可全程并行：后端 Go vs 前端 CSS，文件面小重叠；仅 `room-devices` 页在 86 / 93 有交叠）。**线 B 内部严格串行** —— 每步都改 `index.css`，冲突面大且存在三处不可逆顺序。收口 = 93。

### v1.24 Phase 依赖速查（历史）

| Phase | 主题 | 硬依赖 | 需求 | 关键产出 |
|-------|------|--------|------|---------|
| 76 | 双框 P0 修复 + CSS 审计登记册 | — | VIS-01~04 | 复位块 + status 还原；双框 3→0；登记册 36 条 |
| 77 | 视觉回归验证收口 | 76 | VIS-05 | 21 形态 × 明暗矩阵；双框 = 0；17 张证据图入库 |
| 78 | #16 权限双端闭环 | — | PERM-01~04, 06 | 严格权限中间件 + 读/写兄弟分离 + 三态门控 |
| 79 | PG 实机验收 | 78 | PERM-05 | HTTP 8/8 403 + 对照实验；DB 层前后对照 |
| 80 | 3f130 工位物理诊断展示 | — | DIAG-01 | Table+Alert 分级渲染 + 静默失败补提示 |
| 81 | F5 刷新落 /dashboard 修复 | — | NAV-01 | catch-all 语义区分 + LAST_PATH 防自毁（红绿验证） |
| 82 | lint 存量清零 | — | HYG-01~03, 05 | eslint **0 problems**；豁免表 159→38 接入 lint:ci |
| 83 | devices 页拆分 | 78 | HYG-04 | 714 → 306 行（-57%）+ 行操作再拆至 192 行 |

**并行区间**：76/80/81/82 零依赖并行（Wave 1）→ 78/79（Wave 2）→ 83（Wave 3）→ 77 收口。

### v1.24 执行期关键决策（locked）

- **双框根因**：`index.css:871` 的 `.ant-input{border:...!important}` 命中 antd 6 affix-wrapper 的**内层** `input.ant-input`（antd 原生契约要求内层 `border:none`）→ 双框。`:not(组合子)` 在 CSS3 下整条规则静默丢弃 ⇒ 必须用 `.ant-input-affix-wrapper > input.ant-input`
- **复位块不写 box-shadow**：焦点环归 wrapper（antd 内层 `box-shadow:none!important` 特异性 0,4,1 自足）
- **#16 双病灶必须同时修**：组级 ANY-of（`router.go`）+ 模块级降级（`permission.go` Step 3）；只藏前端按钮 = 假修复
- **gin `Group()` 继承父组中间件链** ⇒ 写组必须是 `authorized` 的**兄弟组**（写成读组的子组则读组 ANY-of 仍放行写操作）
- **三态门控禁用 `permissions.length===0` 判未就绪**：空权限集与未加载完在数据上不可区分
- **hydrate 路径必须同步置 `permissionsLoaded`**：直接 `setState` 绕过 `setPermissions` ⇒ 硬刷新后所有操作按钮灰一阵
- **CSS status class 名**：antd 6 把 status 打在 wrapper **自身**（`ant-input-status-error`），**不存在** `ant-input-affix-wrapper-status-error`（该写法 = 永不命中的死规则，契约检查必须双向断言）
- **权限判定有 30s TTL 缓存**且 `InvalidateMiddlewarePermCache` 未接线 ⇒ 改权限后须等 ≤30s 生效

### v1.25 立项范围（v1.24 结转五项候选 —— 用户裁定全部立项，执行顺序 ①→②）

| 候选 | 内容 | 归属 phase |
|------|------|-----------|
| ① 操作级 RBAC 全站治理 | 76 页面 + 134 路由挂载；`roles/menus/depts/posts` 4 组 ANY-of 拆分；`pkg/permission/config.go` 死表接线；动作级 F 型菜单播种 | 84 → 85 → 86 → 87 |
| ② 样式层收敛到 antd 6 variant | 手写 `!important` 四批清单收敛；明暗色板重构（F3）；圆角 6→8；hex 治理覆盖 `.css` | 88 → 89 → 90 → 91 → 92 → 93 |
| ③ 权限模型三缺口 | `data_scope` 未被消费 / `GetUserPermissions` 不过滤 `menu.status` / Step 2 语义易写反；附带生产库无迁移记录表 + GORM schema 漂移 | 84 |
| ④ Phase 77 视觉新发现 | 暗色 `--input-bg` 第 3 处就地重定义 / 暗色 Select 白底两套边框体系 / `.dept-tree-search` 死规则 / antd 6 弃用警告 / InputNumber 错误态仍灰 | 88 → 89 → 90 → 91 |
| ⑤ `room-devices/index.tsx` 拆分 | 815 行（Lead 实跑复核），比已拆的 devices 页更长 | 93（与 STYLE-05 视觉验收同批） |

**合批理由**：①②③④ 同族可打包，拆成两个里程碑会让样式债与权限债互相等待；⑤ 体量小、与 ④ 的视觉验收共用同一轮截图矩阵，合批不额外增加风险。

### Phase 依赖速查（v1.23）

| Phase | 主题 | 硬依赖 | 需求 |
|-------|------|--------|------|
| 69 | 后端 debug 重验 spike + 信息收集下发 | — | DBG-01 |
| 70 | TextArea 契约基建 (W0) | — | TEXT-01, TEXT-03, TEXT-04, HYG-05 |
| 71 | TextArea 存量迁移 + COMP-05 分域迁移 (W1-W6) | 70 | TEXT-02, MIG-01, MIG-02, MIG-04 |
| 72 | 后端 debug 修复 wave (#3→#2→#1-b) | 69 | DBG-02, DBG-03, DBG-04 |
| 73 | DB 实测日 + cron.go 双执行调查 | 72 + 69 | DBG-05, DBG-06, DBG-07 |
| 74 | COMP-05 收口 + lint 清偿第一波 + 流程积压归档 | 71 | MIG-03, HYG-01, HYG-02, HYG-03, HYG-04 |
| 75 | 浏览器实机 UAT + 收档 | 74 + 72/73 | UAT-01, UAT-02 |

**并行区间**：Phase 69/70/71（后端线 + 前端线无共享文件）∥ Phase 72/73；Phase 74/75 必须等前端线（71）完成。

## Accumulated Context

### v1.25 关键决策（init + roadmap 期, locked）

- **执行顺序 ①权限 → ②样式**（用户裁定）：避免样式波动与权限改造混在同一验证周期；权限先行可复用 v1.24 已建的三态门控基建
- **研究范围仅 ② 定向**（用户裁定）：①③⑤ 靠本仓事实与既有实证 note，仅对 antd 6 官方 variant API 与暗色 token 派生规则做定向核验
- **复用 v1.24 基建不重造**：`RequireStrictPermissions`（只做 Step1 精确匹配，跳过 F 型继承与模块级降级）/ `useActionPermission` 三态门控 / `src/constants/permissionCodes.ts` 单一真相源 + `scripts/check-permission-codes.mjs` 双向校验
- **②须显式解冻 v1.22 冻结契约**：`AntdThemeBridge` 色值真相源变动属冻结面突破，须单独登记与验收
- **零新依赖**：沿用 v1.23 / v1.24 原则；不引 stylelint / Playwright / jscodeshift
- **门控取三态而非纯 fail-closed**：空权限集与未加载在数据上不可区分，纯 fail-closed 会在接口失败时把管理员也锁死且无提示
- **三处不可逆顺序约束**（Phase 89 → 90/91/92；批次 C → 批次 B 同 commit；变量层 → 变体交还）已写进 ROADMAP 的 phase 描述与 STATE 速查表
- **需求→phase 映射采用「主责登记」制**：横跨多 phase 的需求（STYLE-02 四批 / VISFIX-04 两段）只在**终态验收所在 phase** 登记一次，其余 phase 以「承接」标注，避免 Traceability 重复计数
- **需求总数以磁盘条目为准 = 22**（REQUIREMENTS.md 的分解 6+5+4+6+1 与其自称的「26 total」、PROJECT.md 的「共 25 条」不自洽；磁盘上不存在第 23-26 条需求条目，已全量 grep 复核）

### v1.24 关键决策（init + roadmap 期, locked）+ 收档终态

- **双框根因**：`index.css:871` 的 `.ant-input{border:...!important}` 命中 antd 6 affix-wrapper 的**内层** `input.ant-input`（antd 原生契约要求内层 `border:none`）→ 双框。`:not(组合子)` 在 CSS3 下整条规则静默丢弃 ⇒ 必须用 `.ant-input-affix-wrapper > input.ant-input`
- **复位块不写 box-shadow**：焦点环归 wrapper（antd 内层 `box-shadow:none!important` 特异性 0,4,1 自足）
- **#16 双病灶必须同时修**：组级 ANY-of（`router.go`）+ 模块级降级（`permission.go` Step 3）；只藏前端按钮 = 假修复
- **gin `Group()` 继承父组中间件链** ⇒ 写组必须是 `authorized` 的**兄弟组**（写成读组的子组则读组 ANY-of 仍放行写操作）
- **三态门控禁用 `permissions.length===0` 判未就绪**：空权限集与未加载完在数据上不可区分
- **hydrate 路径必须同步置 `permissionsLoaded`**：直接 `setState` 绕过 `setPermissions` ⇒ 硬刷新后所有操作按钮灰一阵
- **CSS status class 名**：antd 6 把 status 打在 wrapper **自身**（`ant-input-status-error`），**不存在** `ant-input-affix-wrapper-status-error`（该写法 = 永不命中的死规则，契约检查必须双向断言）
- **权限判定有 30s TTL 缓存**且 `InvalidateMiddlewarePermCache` 未接线 ⇒ 改权限后须等 ≤30s 生效
- **收档终态（2026-10-28）**：18/18 REQ Satisfied；双框元素数 3→0；#16 前后端双端闭环（HTTP 8/8 403 + 决定性对照）；lint 51→0 problems；devices 714→306 行；生产库夹具已清理、既有行零改动。Known Gaps 五项 → v1.25 立项（见上方「v1.25 立项范围」表）
- **v1.25 研究期三条机制更正**：① 「`darkAlgorithm` 不改写显式 token」机制不成立（Map Token 后展开覆盖 + 组件 token 后 merge + 浅奶油 seed）；② 「antd hash 前缀 +1 类特异性」不成立（`hashPriority:'low'` → hash 特异性 0）；③ 「`onSearch` 已弃用」不成立（antd 6.6.4 未弃用，**不得迁移**）

### v1.23 关键决策（init + roadmap 期, locked）

- `AppTextArea` **独立具名导出**，不挂 `AppInput.TextArea` 静态别名——成员表达式 selector 无法区分合法别名用法与非法裸用法，挂别名会永久放弃成员表达式拦截面、74 处收口假绿
- 排期口径 = **eslint selector 实跑命中数 + 豁免清单条目数**（权威），grep 数仅供粗估；每个 phase 的成功标准以「权威计数落盘/对账」表述，不硬编码某个 grep 数
- DBG-03 走 **Go 侧防线**（写入方枚举 + 显式 normalize + 回归测试），**不用 DB TRIGGER**（维持 v1.19 决策）
- 零新依赖：不引 jscodeshift / Playwright；UAT 用环境已有的 agent-browser skill
- 排序硬约束：迁移 → 收口 → lint → UAT（UAT 必须测迁移后的页面；lint 排在迁移后因迁移波次自身产生新 warn 峰值）
- 后端 debug 线与前端线**全程无共享文件、可并行**；#3 先于 #2（同文件 mac_collection_service.go）；cron.go 改动独立隔离（回归闸门 = `go test ./internal/scheduler/` 全套）

### v1.23 Phase 71 执行期决策（W1-W2 实证, locked for W3-W6）

- **P67 别名降险法在本项目不可用**（实测推翻了规划假设）：P65/P70 的 selector 是 **JSX 名匹配**（`JSXOpeningElement[name.name='Input']`），不是 import 路径匹配。`import { AppInput as Input }` 后 `<Input/>` 仍报 `no-restricted-syntax`，文件无法脱离豁免清单 → 无法达成「grep 命中 = 豁免剩余」对账。**W3-W6 必须用具名迁移**（JSX 名一并改 `<AppInput/>`/`<AppSelect/>`/`<AppTextArea/>`），diff 仍可极小（多数文件仅 import 行 + 标签名）
- **文件移出 P65 豁免块后，P68 hex 与 P65 内网 IP 两条契约会突然对该文件生效** → 迁移前必须先跑定向 eslint 预清存量裸 hex / 内网 IP 字面量，否则「收缩即超红线」（W2 已暴露 manager-rules 的 `#888` 2 处新 error）
- **Select 形态分类器（W1/W2 实测 56 处）**：①options 形态直接迁；②③children/fieldNames/labelInValue **必须改写为 `options={[...]}`**（否则 `optionFilterProp="label"` 指向不存在字段，过滤会「输什么都过滤光」）；④明确禁搜显式 `showSearch={false}`；⑤children+复合子节点保留写法并显式 `optionFilterProp="label"`；⑦multiple/tags 显式 `showSearch`
- **只读展示型 TextArea 显式 `allowClear={false}`**（如 logs 页 operParam/errorMessage）——默认清空按钮在 readOnly 受控区是纯 UI 倒退
- **UUID 炸弹真实分布**：W1/W2 范围唯一 UUID 下拉（dept 负责人）原本 allowClear=true 行为未变；**W3 operations 需重点排查「迁移后凭空获得清空能力」的 org_id/device_id 字段**（那才需 `allowClear={false}`）
- **P66 豁免清单（hook import）与 P65 是两条独立线**：W1/W2 迁完 P65 127→97，P66 保持 37 不动（迁移文件仍在用 usePagination/useTableManager）

- DB 实测日集中 #4/#5/#7（一次环境准备换三个清单）

### v1.22 关键决策（init + roadmap 期, locked）

- 抽契约不抽业务：资产选择器、部门树选择器等业务语义组件已够用，不抽
- `useDebouncedSearch` 并入 RemoteSelect，不单独抽
- 巨型文件拆分不单独立项，随 COMP-05 迁移顺带
- 默认值必须下沉封装层，禁止逐页补 allowClear（横跨 73 文件，不可维护）
- ConfirmDelete 是危险操作唯一入口；危险操作必须二次确认
- COMP-01/02（AppInput/AppSelect/ConfirmDelete）先于 COMP-03/04（PageTable/FormModal）落地，后者依赖前者
- COMP-05 迁移顺序：C1 试点 discoveries 最小页 → backups → devices/cache 巨型页；每页一个 PR，旧实现保留一个迭代
- FIX-02 Modal 脏检查在 Phase 64 存量修补、Phase 66 契约化到 FormModal（两段式）
- 本里程碑纯前端，不动后端接口；预期收益：消除约 7,000~8,000 行样板

### 继承自 v1.21 的环境约束

- `gsd_run` CLI 在本环境不可用（`$HOME/.claude/get-shit-done/bin/gsd-tools.cjs` 不接受 `query <verb>`）——STATE/ROADMAP/REQUIREMENTS 用手工编辑（Phase 57-62 先例）
- 提交落在 main 分支（`branching_strategy: none`）
- DB migration 编号已用到 218（Phase 59）；v1.23 **可能**新增 migration（#4 若确认 `sys_job` 双行根因 → 唯一索引；#3 走 Go 侧防线**不需要** DB TRIGGER migration），落地时从 219 起编

### Decisions

- [v1.21 遗留] 57-HUMAN-UAT.md 三项真环境验证（PG 合并路径 / kill -9 重启 / UAC 533）仍待人工执行；v1.18/1.19/1.20 的 site-visit UAT 债同样挂账

### Blockers

- 无

### Todos

- **Phase 84 / 88 零外部依赖，可立即并行起草**：`/gsd-plan-phase 84`（权限模型三缺口，后端为主）+ `/gsd-plan-phase 88`（验证底座与零风险清理，前端 CSS）
- Phase 85 等 Phase 84（同一批 handler 查询，先做避免二次冲突）；Phase 89 等 Phase 88（口径冻结 + C6-C15 断言是验收底座）
- Phase 86 等 Phase 85（后端不放行时前端铺开无意义）；Phase 90 等 Phase 89（F3 硬前置 + 变量层先于变体交还）
- Phase 87 等 Phase 85 + 86；Phase 91 等 Phase 90；Phase 92 等 Phase 91（暗色手写副本须在 token 化完成后才可删）
- Phase 93 等 Phase 92（样式收敛稳定后截图基线才可用）
- vendor-react gzip 776 kB 基线沿用为 Phase 86 / 93 零回归守卫
- M1-M16 量化基线由 Phase 88 落盘（M1 口径 = `find src -name "*.css" -exec cat {} + | grep -o '!important' | wc -l`），Phase 89-93 以其为对账分母
- `scripts/check-permission-codes.mjs` 的模块条目扩展由 Phase 85 落地，Phase 87 接入 CI 门禁
- `scripts/check-hardcoded-colors.mjs` 当前 exit=1（18 文件命中）→ Phase 89 修复并接入 `lint:ci`

### Roadmap Evolution

- Phases 84-93 created for v1.25「全站权限治理与样式收敛」: 84 权限模型三缺口（PMODEL-01~04）→ 85 后端动作级严格化（RBAC-01/03/04）→ 86 前端操作入口三态门控铺开（RBAC-02）→ 87 动作级权限授予闭环（RBAC-05/06）→ 88 验证底座与零风险清理（VISFIX-03/04/06 + STYLE-02 第 1 批）→ 89 变量层与明暗色板重构（STYLE-01/04 + VISFIX-01/02）→ 90 Input 域交还 variant/token + 圆角统一（STYLE-03 + VISFIX-05）→ 91 Select/InputNumber/Form/Table/Modal 域交还（承接 STYLE-02 R8 + VISFIX-04 剩余）→ 92 暗色 mega-cluster 合并（STYLE-02 主责）→ 93 room-devices 拆分 + 视觉回归验收（SPLIT-01 + STYLE-05）。**22/22 需求映射，无孤儿无重复**；线 A 权限 84-87 ∥ 线 B 样式 88-93，线内串行；写死三处不可逆顺序约束（89→90/91/92、批次 C→批次 B 同 commit、变量层→变体交还）。 (CRITICAL)
- Phases 76-83 created for v1.24「视觉回归修复与遗留债务清偿」: 76 双框 P0 修复 + CSS 审计登记册（VIS-01~04）→ 77 视觉回归验证收口（VIS-05）→ 78 #16 前后端双端闭环（PERM-01~04/06）→ 79 PG 实机验收（PERM-05）→ 80 3f130 空因诊断前端接入（DIAG-01）→ 81 F5 刷新落 /dashboard 修复（NAV-01）→ 82 lint 清零 + 豁免理由表修复（HYG-01/02/03/05）→ 83 devices 页拆分（HYG-04）。18/18 需求映射，无孤儿无重复；**全部 SHIPPED 2026-10-28**（tag `v1.24`）。 (CRITICAL)
- Phases 69-75 created for v1.23「前端契约全量收尾与遗留清偿」: 69 后端 debug 重验 spike（DBG-01）→ 70 TextArea 契约基建 W0（TEXT-01/03/04 + HYG-05）→ 71 TextArea 存量 + COMP-05 分域迁移 W1-W6（TEXT-02 + MIG-01/02/04）→ 72 后端 debug 修复 wave（DBG-02/03/04，#3 先于 #2）→ 73 DB 实测日 + cron.go 双执行调查（DBG-05/06/07）→ 74 COMP-05 收口 + lint 清偿 + quick 目录归档（MIG-03 + HYG-01~04）→ 75 浏览器实机 UAT + 收档（UAT-01/02）。22/22 需求映射，无孤儿无重复。 (CRITICAL)
- Phase 68.1 inserted after Phase 68: v1.22 收尾清偿 phase：危险操作收口 + 契约 minor 补课 + 豁免收缩 + 存量迁移收尾（DEBT-04） (URGENT)

## Project Reference

See: `.planning/PROJECT.md`（updated 2026-10-28 after v1.24）

**Core value:** 端到端的运维可观测与可审计——每个写操作都留下可追溯记录（谁、何时、改了什么、从哪来、改前改后），敏感字段自动脱敏；配合 Excel 批量管理与引用自动解析（部门名 → UUID），运维人员对整个 IT 资产域拥有唯一可信的系统记录。
**Current focus:** v1.25「全站权限治理与样式收敛」IN PROGRESS（Phases 84-93，22/22 REQ mapped）——Phase 84 / 88 无外部依赖可并行起草。

## Session

**Last session:** 2026-10-28T13:31:58+08:00
**Stopped at:** v1.25 roadmap CREATED（Phases 84-93，22/22 REQ mapped，Traceability 已回填 `.planning/REQUIREMENTS.md`）
**Resume file:** 无（路线图已就绪，下一步 `/gsd-plan-phase 84` 与 `/gsd-plan-phase 88` 并行起草）

## Performance Metrics

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| v1.25（10 phase，planning 阶段） | 路线图 1 轮完成（2026-10-28） | 10 phase × 5 成功标准 | — |
| v1.24（8 phase，subagent 直接执行，按刀原子提交） | ~3.5 h（09:19→12:59，单日） | 8 phase × 9 项闸门 | 104（+5,419 −2,232，46 commits） |
| v1.23（7 phase） | 单日 | 121 文件迁移 | — |

## Operator Next Steps

- **v1.25 路线图已创建**（`.planning/ROADMAP.md` Phases 84-93；`.planning/REQUIREMENTS.md` Traceability 22/22 回填，Status 全 `Pending`）
- 立即可并行起草：**Phase 84**（权限模型三缺口，后端）与 **Phase 88**（验证底座与零风险清理，前端 CSS）——两条线零外部依赖、零共享文件
- 样式线**严格串行**：88 → 89 → 90 → 91 → 92 → 93；**三处不可逆顺序约束**已写进 ROADMAP 与 STATE 速查表，不得交换
- **不要 git commit** —— 提交由 orchestrator 负责
- **待用户决定是否 push tag**（remote `origin` = gitcode.com/ninedrunk/go_react，upstream `origin/main`）
- v1.26 候选债已登记在 `.planning/REQUIREMENTS.md` Future Requirements（sqlite `ON CONFLICT` 方言泄漏 / 权限缓存陈旧 / `responsible_user_id` 恒空 / React Compiler 约束适配 / 双 ConfigProvider 嵌套 / 文档漂移 / AD 方向补齐 / 残留现场 UAT 债）
