# Milestone v1.24 Requirements — 视觉回归修复与遗留债务清偿

> **Defined**: 2026-10-28
> **Core Value**: 修复 v1.23 引入的输入框「双重外框」视觉回归（用户主诉），把 #16 行操作权限从「UI 缺陷」升级为「前后端双端安全闭环」，并清偿 v1.23 收档结转的 4 项遗留债务与 2 项延后技术债。
>
> **事实基线**：`.planning/research/{STACK,ARCHITECTURE,PITFALLS}.md`（3 维研究完成，FEATURES 进行中）+ `.planning/notes/2026-10-28-double-border-rootcause.md`（双框根因实证）+ `.planning/notes/2026-10-28-pg-permission-baseline.md`（#16 定责 + 生产 PG 权限矩阵）+ `.planning/notes/2026-10-28-workstation-binding-facts.md`（工位绑定现状实测）

## 用户主诉与根因

> 「文本输入框的一键删除已经有了，但是有的输入框出现这种双重外框的情况」

**根因（已实证，非推测）**：

| 层 | 事实 |
|---|---|
| antd 6.6.4 结构事实 | **`.ant-input` 本身没有边框**（`genBasicInputStyle` 无 border 声明）——边框只在 `-outlined/-filled/-underlined` **variant 类**上；affix-wrapper 的内层 input 另有 `border:none`（`antd/es/input/style/index.js:382-395`） |
| 项目覆盖 | `src/index.css:871-880` 沿用 antd 5 心智模型写 `.ant-input { border: 1px solid var(--input-border) !important }` → **同时**命中顶层 input 与 wrapper 内层 input → 双框 |
| 覆盖机制 | antd 规则无 `!important`、项目有 → **important 优先于特异性**（0,1,0 压过 0,3,1）。且 antd cssinjs 的 hash 前缀使 antd 规则特异性 +1 类——「写在后面就能覆盖」在 antd 6 下不成立，项目 1346 处 `!important` 是被逼出来的 |
| 为何普遍 | v1.23 把 `allowClear` 默认化为 `true`（`hasPrefixSuffix = !!(prefix\|\|suffix\|\|allowClear\|\|showCount)`，单一 prop 即进 wrapper）→ 全站 Input 都走 affix-wrapper → 双框影响面 = 100% |
| 为何「有的」 | 项目内已有 3 处**局部补丁**（`login.css:351-358`、`index.css:2056-2060`、`index.css:2077-2081`）——那 3 处无双框，其余全双框 |
| TextArea | `AppTextArea` 默认 allowClear=true → 内层是 `textarea.ant-input` 同样被覆盖 → **全站 TextArea 100% 双框**（无任何局部补丁） |
| 免疫组件 | InputNumber（内层类名不匹配 `.ant-input`）、DatePicker（内层 input 无 ant-input class）、Select（边框在 `.ant-select` 上）——均无双框 |

## v1.24 Requirements（16 条）

### VIS — 视觉回归修复（P0，用户主诉）

- [ ] **VIS-01**: 输入框「双重外框」消除——新增 wrapper 上下文复位规则（**方案 1，推荐写法**）：`.ant-input-affix-wrapper > input.ant-input, .ant-input-affix-wrapper > textarea.ant-input { border:none !important; border-radius:0 !important; background:transparent !important; color:inherit !important; font-size:inherit !important; padding:0 !important }`——组合子在 `:not()` 外（CSS3 合法），对 variant 类与暗色标签选择器免疫；**禁止**使用 `.ant-input:not(.ant-input-affix-wrapper > .ant-input)` 形态（CSS3 下整条规则被**静默丢弃**）
- [ ] **VIS-02**: 焦点环单圈化验证——焦点环画在 wrapper（antd `:focus-within`）；**复位规则里不写 box-shadow**（antd 自身的 `box-shadow:none!important` 在内层、特异性 0,4,1 已足够）；已知可接受的微小变化：焦点环**外扩 1px**；验收 = focus 特写截图确认单圈、无跳动；**不得删除** `index.css:892` / `:2275-2284` 的 focus 规则与 `:910` / `:916-923` 的 wrapper 边框/焦点环规则
- [ ] **VIS-03**: 错误态边框色恢复——Form `status="error"` / `"warning"` 的边框色不再被项目 `border: ... !important` 简写压掉（F1；验收：`borderTopColor` = antd `colorError`，非 `--input-border`）；实现前先实测 antd 6 status class 落点（root 还是 wrapper，决定修复点）
- [ ] **VIS-04**: CSS 一致性审计登记册——全局 1346 处 `!important` / 219 行 antd 选择器 / 96 个分区 / 60 个暗色重复分区 / `--input-bg` 就地重定义 3 次的冲突面全量盘点并登记（26 条输入类 + 8 条非输入类 + 12 条 Select 死规则——antd 6 已无 `.ant-select-selector`），产出可追溯清单 + 后续排期建议（**登记为主，不在本里程碑全量修**）
- [ ] **VIS-05**: 回归验证落地——computed-style 断言脚本（双框元素数=0 / 内层 box-shadow=none / wrapper 焦点环非 none / 错误态色）+ 截图矩阵（5 类输入源 × 默认/hover/focus/readonly/disabled/error × 亮/暗；**必测带字数统计的 TextArea 与无字数统计的纯 textarea 两条 DOM 路径**；DatePicker 只做回归确认，**不得顺手统一其边框色**——既存欠账不扩面）；前置基线：修复前先跑一次断言取「双框元素数 > 0」的反证

### PERM — 操作级权限（#16 前后端双端闭环）

> **定责结论**：#16 是**两个独立缺陷叠加**——
> ① 前端：52 个含「操作」列的文件中 **45 个完全不引用 permissions** → 编辑/删除按钮**恒显示**（根本不判定，而非加载中默认显示）；结构性成因 = `ActionButton` 契约（`ActionButtons.tsx:15-24`）**没有 permission 字段**
> ② 后端：`internal/api/router.go:157-162` 给 `/system/users` 挂 `[UserList, UserAdd, UserEdit, UserView]` 的 **ANY-of** 组 → 只持 `system:user:list` 的用户调 `delete` **返回 200 并真的删除**（安全缺陷，非 UX 缺陷）；叠加 `permission.go:200-218` 的**模块级降级**（`system:user:remove` → 降级匹配 `system:user`）→ 只读角色仍放行；`permission.go:305-315` 的 doc 注释「写操作保持严格权限」**与实现矛盾**

- [ ] **PERM-01**: `ActionButton` 契约与 `createDeleteAction` options 新增**可选** `permission?: string` 字段（加法扩字段，存量调用点不传即保持原行为，不破坏 v1.22 冻结契约）
- [ ] **PERM-02**: 新增**单一判定 helper**（建议 `src/components/shared/useActionPermission.ts`），**三态门控**：权限未就绪（显式 loading/error 位）→ 渲染但 disabled + title/aria 提示；就绪且含权限码 → 可用；就绪且不含 → **不渲染**。**禁用** `permissions.length === 0` 判「未就绪」（空权限集与未加载在数据上不可区分——接口失败时管理员也看不到按钮且无提示）
- [ ] **PERM-03**: 前端系统用户页接入三态门控——新增 `system:user:add`、编辑 `system:user:edit`、删除/批量删除 `system:user:remove`、重置密码 `system:user:resetPwd`、导入 `system:user:import`、导出 `system:user:export`
- [ ] **PERM-04**: 后端新增**不做模块级降级**的严格权限中间件（跳过 `permission.go` Step 3；Step 2 的 F→C 继承对写操作同样须跳过）+ `/system/users` 写操作从 ANY-of 组拆出为独立子组，逐动作绑定 `pkg/permission/config.go` 已定义但从未被引用的 `UserRemove` 等常量；参照 network 域范式 `port_write_router.go:51`（同仓 `network_router.go:183-192` 注释已明写 ANY-of 陷阱与修法）
- [ ] **PERM-05**: 受限角色验收——PG 环境走 **UI 正常路径**建角色（避开 sqlite ON CONFLICT 500 与 `menu:user:perms:{id}` 30 分钟缓存不失效；**绝不能挂 admin 角色**——isSuperAdmin 旁路）；构造「有 `system:user` 页面权限、无 F 型按钮权限」：按钮不可见、curl 直调写接口 **403**（T4 判据：200 ⇒ 后端 ANY-of 缺陷坐实）；admin 与 xtgl 行为零变化；改权限后等 30s（后端 TTL）再验证
- [ ] **PERM-06**: 「模块 × 操作 → 权限码」映射表**单一真相源** + 一致性校验脚本（对照后端 `init_data.go` / `pkg/permission/config.go`；migration_159 已证明权限码命名会漂移）——防止 45 个文件的后续铺开各自为政

### DIAG — 工位物理链路空因展示（3f130）

- [ ] **DIAG-01**: 工位管理页接入已交付的诊断端点（前端目前 **0 处调用**，仍用旧 `/physical` + 硬编码静态文案 `WorkstationDeviceTable/index.tsx:604-609`）；物理链路为空时展示**分级**原因——`no_user_bound` 中性 info（**75.3% 工位属此常态**，1238/1643）、数据缺失 warning、真实异常 error；**前置**：先对齐后端契约——`workstation_device_service.go:820` 与 `:871-874` 注释**自相矛盾**，实际代码（`:822`）无条件先判 `!UserBound` → 有数据也返回 `no_user_bound`；需补「有数据 + 空因」不变量测试；前端按 `devices.length` 分流「空态」与「有数据但带告警」两套 UI；只读诊断，**不写生产数据**

### NAV — F5 刷新落地页

- [ ] **NAV-01**: F5 刷新保持当前路由——根因：深层 URL 落进 `DynamicRoutes.tsx:315` 的 `path="*"` 兜底（`routeElements` 首帧为空，:228-230）→ 跳 `/dashboard` → **`:243-247` 的 effect 再把 `/dashboard` 反写回 LAST_PATH（自毁循环）**→ `useTabSync.ts:86-88` 又把 activeTab 抢成 dashboard（二阶损伤）。修法（三件套）：① 菜单/路由未就绪时不渲染 `*` 兜底（或渲染 InitializingFallback）② 兜底跳转**不写回 LAST_PATH** ③ LAST_PATH 加版本号校验。**不改** `path="*"` 的 fallback 目标本体；不得影响未登录跳登录、无权限跳 403、deep link 直达

### HYG — 规范与存量技术债

- [ ] **HYG-01**: `ColumnConfigModal.tsx:136` 的 `Modal.confirm` **属合法调用**（`src/components/shared/` 命中 `eslint.config.js:271-283` 目录级封装层豁免 → 该处是豁免盲区「无人区」，不在 `exemption-reasons.json`）→ 补行内理由注释文档化；**不要**把该目录摘出豁免（会爆几十条 error）；若登记 json 必须双向同步 eslint 锚点
- [ ] **HYG-02**: lint 存量 **17 errors + exhaustive-deps 12 处**清零——17 errors 构成：12 exhaustive-deps + `set-state-in-effect`（DashboardView.tsx:161）+ `immutability`（useRealtimeUpdates.ts:119）+ `globals`（ReconciliationTab.test.tsx:51）+ `preserve-manual-memoization`（useRPAProgress.ts:78）+ `local/no-large-dropdown-list`（**契约组件 AppSelect.tsx:16 自己违规**）；**同文件约束**：修 `useRealtimeUpdates.ts:130` 的 immutability 可能触发/加剧 compiler 错误 → 同文件一起改一起验；`useCaptchaModals.tsx` 5 处 `message` 依赖逐处判语义（机械加依赖会引出新 error）
- [ ] **HYG-03**: `react-refresh` **33 处**清零（函数/hook 22 + 复合字面量常量 11；与 barrel index.ts 无关，全是 .tsx）——类 A 伪 constants 13 处/3 文件（workstations/dept/dict）+ 类 B 组件文件额外导出 20 处/14 文件；`export { X }` → `export type { X }` 既是清 warning 的必要手段也是 `verbatimModuleSyntax:true` 的强制写法、运行时无差别（但须逐个确认纯类型；测试文件不在 type-check 范围须额外跑 vitest）；`allowConstantExport:true` **不豁免**数组/对象字面量；**契约层豁免名单**（shared/ui 的 PageTable/FormModal/ActionButtons/statusHelpers/ModernTag）不宜拆散 → 划豁免登记；`export * from` 会被直接报 exportAll 需改具名；**eslint.config.js 锚点已漂移**（现 457 行，P66 :347、P65 :407）→ 拆分必须同批跑 `gen-p65-exemptions.mjs` + 同步 hex-exemptions；`.tsx → .ts` 降级前 grep 显式扩展名引用
- [ ] **HYG-04**: devices 页拆分（实测 **714 行**）——按序：`buildRowActions`(:451-513) 提取 + 同时加权限 gate（一次改动同解 #16 前端侧与拆分）→ 列定义工厂 `getDeviceTableColumns`(:81-169 纯函数零风险) → 采集端口(:391-418) → 快速创建(:285-341，新建 `hooks/useDeviceQuickCreate.ts`，**不塞进已 404 行的 DeviceModals**) → searchFields(:539-563，依赖 useDict 动态**不混进 constants.ts**)；`deptFilters`(:277) 与 `columns`(:524) 的 useMemo **留主组件**（引用不稳会重犯 v1.23 FAIL-1 页码回跳）；**禁改** `port-write/**` 与 `ActionButtons.tsx` 对外形状（v1.19 冻结交互）；收敛目标 ~250 行；**已知债务**：`room-devices/index.tsx` 815 行比 devices 更长，本里程碑不扩面、登记即可

## 关键决策 (v1.24 init + 研究期, locked)

| 决策 | 理由 |
|------|------|
| **双框修根因 + CSS 一致性审计**（用户裁定） | 局部遮盖留下同类隐患；审计产出登记册供后续排期 |
| 修复采用**方案 A**（wrapper 上下文复位，≈6 行）而非方案 C | 方案 C 需先重构 `AntdThemeBridge` 明暗色板（F3：`colorBgContainer` 硬编码 `#FFFFFF`，`darkAlgorithm` 不改写显式 token → 暗色炸白底），触碰 v1.22 冻结契约 → 推迟 v1.25 |
| 复位规则**不写 box-shadow**、**不成组删既有 focus/wrapper 规则**、**不删 3 处局部补丁** | A2/A8 + 回归陷阱清单 |
| **禁用** `:not(组合子)` 形态（CSS3 静默丢弃整条规则） | A1；用组合子在外部的 wrapper 上下文选择器替代 |
| **门控取三态而非纯 fail-closed**（用户裁定的「无权限不渲染」精神不变，「未就绪」态细化为禁用+提示） | B2：空权限集与未加载不可区分，纯 fail-closed 会在接口失败时把管理员也锁死且无提示 |
| #16 取**前后端双端闭环**（用户裁定），全站治理立 v1.25 | 108 页面 / 340 路由回归面过大 |
| 后端写操作用**不做模块降级的严格中间件** | Step 3 降级 + ANY-of 双病灶叠加 |
| 权限码映射表**单一真相源 + 校验脚本** | migration_159 证明权限码命名会漂移；45 文件铺开的前提设施 |
| 3f130 **只做前端展示，不写生产数据**（用户裁定）+ **先对齐后端契约再接前端** | B4：后端注释自相矛盾且「有数据也带 no_user_bound」 |
| F5 修**三件套**（兜底不首帧挂载 + 不反写 LAST_PATH + 版本号），**不改 fallback 目标** | B5：只改 fallback 会留下 LAST_PATH 自毁循环 |
| `ActionButton` 扩字段采用**加法** | 不破坏 v1.22 冻结契约 |
| 圆角漂移（CSS 6px vs token 8px）不在本里程碑统一 | 与方案 C 绑定，推 v1.25 |
| 零新依赖 | 沿用 v1.23 原则；不引 Playwright/stylelint；视觉验证用 computed-style 断言 + agent-browser |

## Future Requirements（延后）

- **v1.25 候选①**：**操作级 RBAC 全站治理**——前端 108/119 页面 + 后端 340/367 路由；system 域其余 4 组（roles/menus/depts/posts）ANY-of 拆分；`pkg/permission/config.go:223-289` 动作级映射死表接线
- **v1.25 候选②**：**样式层收敛到 antd 6 variant**（方案 C）——先重构 AntdThemeBridge 明暗色板（F3）、删/降级 4 批手写规则（≈-120 行）、统一圆角（6→8）、hex 治理对 .css 失明（`eslint.config.js:150` selector 只匹配 JS/TS 节点）
- **sqlite `ON CONFLICT` 方言泄漏**——`role_service.go:425-430` + `migration_206:29-33` 的 `isPostgreSQL` 守卫 + `models/role.go:24-27` 无唯一索引 → sqlite 下无法创建带菜单授权的角色（#16 无法在 sqlite 复测的根因）
- **权限缓存陈旧**——后端 30s TTL 且 `InvalidateMiddlewarePermCache` 未接线（`permission.go:68`）；前端双层缓存（menuStore 5min + sessionStorage 30min `DynamicRoutes.tsx:34`）；`sys_menu.perms` 整串取出不拆逗号（`menu_service.go:547-552`）
- **`ops_workstation_device.responsible_user_id` 恒空**（445/445）——写入侧从未回填 uuid
- **React Compiler 纯渲染约束适配**（eslint-plugin-react-hooks v7.0.1 新增规则族）
- **双 ConfigProvider 嵌套**（`App.tsx:9,48` 项目自有 Provider vs `AntdThemeBridge.tsx:178` 真 antd Provider）
- **3f130 绑定 user_id**（业务核对后由用户执行）
- **`room-devices/index.tsx` 815 行**拆分（比 devices 更长）
- **文档漂移**：CLAUDE.md 写 antd 6.1 / React 19.2，实际 ^6.6.4 / ^19.3.0；`75-UAT.md:66` 记 `optionFilterProp="value"` 而代码实际 `"label"`

## Out of Scope

| Feature | Reason |
|---------|--------|
| v1.22 冻结契约破坏性改名 | 锁定契约 |
| antd 6 variant 全局改造（方案 C） | 需先解 F3 + 解冻 ruoyiBrand 色值真相源；立 v1.25 |
| 全站 RBAC 治理 | 立 v1.25 |
| sqlite ON CONFLICT 方言修复 | 独立归档 |
| responsible_user_id 写入侧回填 | 独立归档 |
| DatePicker 边框色统一 | 既存欠账，B 项 A5 明确「不得顺手修」 |
| room-devices 拆分 | 登记，不扩面 |
| 引入 stylelint / Playwright / jscodeshift | 零新依赖 |
| 动生产数据（3f130 绑定 / 角色调整） | 用户裁定：只做只读诊断展示与代码修复 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| VIS-01 | 76 | Pending |
| VIS-02 | 76 | Pending |
| VIS-03 | 76 | Pending |
| VIS-04 | 76 | Pending |
| VIS-05 | 77 | Pending |
| PERM-01 | 78 | Pending |
| PERM-02 | 78 | Pending |
| PERM-03 | 78 | Pending |
| PERM-04 | 78 | Pending |
| PERM-05 | 79 | Pending |
| PERM-06 | 78 | Pending |
| DIAG-01 | 80 | Pending |
| NAV-01 | 81 | Pending |
| HYG-01 | 82 | Pending |
| HYG-02 | 82 | Pending |
| HYG-03 | 82 | Pending |
| HYG-04 | 83 | Pending |

**Coverage:**
- v1.24 requirements: 17 total
- Mapped to phases: 17（VIS×5 → 76/77 · PERM×6 → 78/79 · DIAG → 80 · NAV → 81 · HYG×4 → 82/83）
- Unmapped: 0 ✓

---
*Requirements defined: 2026-10-28*
*Last updated: 2026-10-28 初版（STACK + ARCHITECTURE + PITFALLS 三维研究完成并交叉验证；FEATURES 研究进行中，定稿时校订）*
