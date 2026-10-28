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

- [x] **VIS-01**: 输入框「双重外框」消除——新增 wrapper 上下文复位规则（**方案 1，推荐写法**）：`.ant-input-affix-wrapper > input.ant-input, .ant-input-affix-wrapper > textarea.ant-input { border:none !important; border-radius:0 !important; background:transparent !important; color:inherit !important; font-size:inherit !important; padding:0 !important }`——组合子在 `:not()` 外（CSS3 合法），对 variant 类与暗色标签选择器免疫；**禁止**使用 `.ant-input:not(.ant-input-affix-wrapper > .ant-input)` 形态（CSS3 下整条规则被**静默丢弃**）；**修法必须同步 4 个块**：`:871-905` 浅色主块、`:2252-2284` 暗色块（含全局 `input, textarea` **元素选择器**，特异性 0,1,1 会绕过类选择器收窄）、`:3717-3757` 暗色二块（可抄 `index.css:2056` 既有正确范式）；影响面精确 = **309 个输入框（235 AppInput + 74 AppTextArea，89 文件）全部双框**（✅ 2026-10-28 Phase 76 完成（commit 51b545dd）：wrapper 上下文复位规则 6 项（border/radius/background/color/font-size/padding），**无 box-shadow**；实测基线「双框元素数 = 3」→「0」；契约静态检查 scripts/css-contract-check.js ALL PASS）
- [x] **VIS-02**: 焦点环单圈化验证——焦点环画在 wrapper（antd `:focus-within`）；**复位规则里不写 box-shadow**（antd 自身的 `box-shadow:none!important` 在内层、特异性 0,4,1 已足够）；已知可接受的微小变化：焦点环**外扩 1px**；验收 = focus 特写截图确认单圈、无跳动；**不得删除** `index.css:892` / `:2275-2284` 的 focus 规则与 `:910` / `:916-923` 的 wrapper 边框/焦点环规则（✅ 2026-10-28 Phase 76 完成（commit 51b545dd）：焦点环单圈化——复位规则不写 box-shadow，焦点环由 wrapper 独家承担（实测 wrapperFocused ring #e9efeb 2px 单圈保留、内层 shadow=none 含暗色）；回归陷阱 4 项守护进静态检查脚本）
- [x] **VIS-03**: 错误态边框色恢复——Form `status="error"` / `"warning"` 的边框色不再被项目 `border: ... !important` 简写压掉（F1；验收：`borderTopColor` = antd `colorError`，非 `--input-border`）；实现前先实测 antd 6 status class 落点（root 还是 wrapper，决定修复点）（✅ 2026-10-28 Phase 76 完成（commit 51b545dd）：status 边框色还原块（error→colorError / warning→colorWarning，含暗色与 :hover 变体）；实测错误态 borderTopColor 由 rgb(219,215,206)=--input-border 变为 rgb(255,77,79) ✅；仅 Input 域，Select/DatePicker 不扩面）
- [x] **VIS-04**: CSS 一致性审计登记册——全局 1346 处 `!important` / 219 行 antd 选择器 / 96 个分区 / 60 个暗色重复分区 / `--input-bg` 就地重定义 3 次的冲突面全量盘点并登记（26 条输入类 + 8 条非输入类 + 12 条 Select 死规则——antd 6 已无 `.ant-select-selector`），产出可追溯清单 + 后续排期建议（**登记为主，不在本里程碑全量修**）（✅ 2026-10-28 Phase 76 完成（commit 51b545dd）：登记册 .planning/notes/v1.24-css-audit-register.md —— 34+2 条（P0 已修 8 / P1 待修 7 / P2 登记 19 / 保留 1 / 样板 1）+ Select 死规则族单列 4 条；!important 1346 → 1359（+13 = 复位 6 + status 7）；暗色 574 处/≈60 重复分区登记为 v1.25 方案 C 收敛对象）
- [ ] **VIS-05**: 回归验证落地——computed-style 断言脚本（双框元素数=0 / 内层 box-shadow=none / wrapper 焦点环非 none / 错误态色）+ 截图矩阵（5 类输入源 × 默认/hover/focus/readonly/disabled/error × 亮/暗；**必测带字数统计的 TextArea 与无字数统计的纯 textarea 两条 DOM 路径**；DatePicker 只做回归确认，**不得顺手统一其边框色**——既存欠账不扩面）；前置基线：修复前先跑一次断言取「双框元素数 > 0」的反证

### PERM — 操作级权限（#16 前后端双端闭环）

> **定责结论**：#16 是**两个独立缺陷叠加**——
> ① 前端：52 个含「操作」列的文件中 **45 个完全不引用 permissions** → 编辑/删除按钮**恒显示**（根本不判定，而非加载中默认显示）；结构性成因 = `ActionButton` 契约（`ActionButtons.tsx:15-24`）**没有 permission 字段**；全仓仅 12 个文件用 `useMenuStore.permissions`（**逐页 opt-in**）；用户假设的 `canWrite` 派生层**不存在**（`menuStore.ts:21` 只有 `permissions: string[]`，`canWrite` 是 `network/ports/index.tsx:132` 的局部 const）
> ② 后端：**不止 system 用户组**——全仓 **39 处路由组全部 ANY-of**、单权限严格语义 `Permission()` **用了 0 次**：`internal/api/router.go:157-162`（system users）+ `:541-546`（ops building）等；只持 `list` 的用户调 `delete` **返回 200 并真的删除**（安全缺陷，非 UX 缺陷）；叠加 `permission.go:200-218` 的**模块级降级**与 `:159-196` 的 F→C 继承放大 → 只读角色仍放行；`RequirePermissionsWithQuery` doc 注释「写操作保持严格权限」**与实现矛盾**；前端权限端点实为 `POST /system/my-menus/permissions`（`menuApi.ts:23-26`，无权限门）——v1.23 UAT 引用的 `/system/menus/permissions` 是管理员端点，**引用有误**

- [ ] **PERM-01**: `ActionButton` 契约与 `createDeleteAction` options 新增**可选** `permission?: string` 字段（加法扩字段，存量调用点不传即保持原行为，不破坏 v1.22 冻结契约）
- [ ] **PERM-02**: 新增**单一判定 helper**（建议 `src/components/shared/useActionPermission.ts`），**三态门控**：权限未就绪（显式 loading/error 位）→ 渲染但 disabled + title/aria 提示；就绪且含权限码 → 可用；就绪且不含 → **不渲染**。**禁用** `permissions.length === 0` 判「未就绪」（空权限集与未加载在数据上不可区分——接口失败时管理员也看不到按钮且无提示）
- [ ] **PERM-03**: 前端系统用户页接入三态门控——新增 `system:user:add`、编辑 `system:user:edit`、删除/批量删除 `system:user:remove`、重置密码 `system:user:resetPwd`、导入 `system:user:import`、导出 `system:user:export`
- [ ] **PERM-04**: 后端新增**不做模块级降级**的严格权限中间件（跳过 `permission.go` Step 3；Step 2 的 F→C 继承对写操作同样须跳过）+ `/system/users` 写操作从 ANY-of 组拆出为独立子组，逐动作绑定 `pkg/permission/config.go` 已定义但从未被引用的 `UserRemove` 等常量；参照 network 域范式 `port_write_router.go:51`（同仓 `network_router.go:183-192` 注释已明写 ANY-of 陷阱与修法）
- [ ] **PERM-05**: 受限角色验收——PG 环境走 **UI 正常路径**建角色（避开 sqlite ON CONFLICT 500 与 `menu:user:perms:{id}` 30 分钟缓存不失效；**绝不能挂 admin 角色**——isSuperAdmin 旁路）；构造「有 `system:user` 页面权限、无 F 型按钮权限」：按钮不可见、curl 直调写接口 **403**（T4 判据：200 ⇒ 后端 ANY-of 缺陷坐实）；admin 与 xtgl 行为零变化；改权限后等 30s（后端 TTL）再验证
- [ ] **PERM-06**: 「模块 × 操作 → 权限码」映射表**单一真相源** + 一致性校验脚本（对照后端 `init_data.go` / `pkg/permission/config.go`；migration_159 已证明权限码命名会漂移）——防止 45 个文件的后续铺开各自为政

### DIAG — 工位物理链路空因展示（3f130）

- [x] **DIAG-01**: 工位管理页接入已交付的诊断端点（前端目前 **0 处调用**，`opsApi.ts:617-620` 仍调裸 `/physical` + 硬编码静态文案 `WorkstationDeviceTable/index.tsx:604-609` 只覆盖 `info_point_misbound` 一种子情形、对其余空因是**误导性文案**）；实现口径：`getPhysical` 内部改调诊断端点但**对外仍返回数组**（调用方零改动）+ 另加 `getPhysicalWithDiagnosis`；**「有数据但 emptyReason=no_user_bound」是正常业务态**（`workstation_device_service.go:800-827` 明确「未绑用户压过有数据」，测试 :388 断言锁定；75.3% 工位属此常态）→ 渲染 **Table + warning Alert 双区**（不渲染 Empty、不用 Alert 替代表格）；0 台时 `<Alert type="info" showIcon description={emptyMessage}/>`（后端文案 60-100 字，Empty 排版拥挤）；`warnings` 任何组合下逐条可见；`diagnosis` 计数默认不展示、折叠进 Tooltip；**前置**：修正后端自相矛盾注释（`workstation_physical_diagnosis.go:63-65` 写「有数据时 EmptyReason 为空」与实现/测试相反，不修前端会照注释写错）+ 补「有数据 + 空因」不变量测试；顺带补 `WorkstationDeviceTable/index.tsx:127-129` 查询失败只 console.warn 的静默失败（加 message.warning）；只读诊断，**不写生产数据**（✅ 2026-10-28 Phase 80 完成（commits 93ce5245 + b474dba9）：前端接入诊断端点（getPhysical 内部改调但对外仍返回数组 + 新增 getPhysicalWithDiagnosis）；Table+Alert 双区分级渲染（no_user_bound→info 常态 / 其余→warning / warnings 逐条 / diagnosis 计数折叠 Tooltip）；查询失败由静默 console.warn 改为 message.warning；后端自相矛盾注释已修正（口径与实现对齐：未绑用户压过有数据））

### NAV — F5 刷新落地页

- [x] **NAV-01**: F5 刷新保持当前路由——根因时序链（FEATURES/ARCHITECTURE 双研究一致）：`DynamicRoutes.tsx:173` 同步读 sessionStorage 缓存 → `:179-186` **useEffect 里**水合 store（首帧 `allMenus` 仍为空）→ `:227-240` `routeElements=[]` → 首帧空 `<Routes>` 无路由可匹配 → `:315` 的 `path="*"` catch-all 抢先命中 → `<Navigate to="/dashboard" replace/>`（passive effect 子先父后，早于水合）→ `:243-247` `saveLastPath` 把 `/dashboard` **反写回 LAST_PATH（自毁）**→ `useTabSync.ts:86-88` 把 activeTab 抢成 dashboard。修法：**方案 B（推荐）**——`<Route path="*" element={menusReady ? <Navigate to="/dashboard"/> : <InitializingFallback/>} />`（`menusReady = allMenus.length > 0`，把「菜单未加载」与「用户无权限」区分）+ 配套小修：`saveLastPath` 跳过自动兜底跳转产生的 pathname；**单独 useLayoutEffect 水合无效**（`<Navigate>` 已挂载）；回归面：`:285` 根路径→lastPath、`:263-270` 未登录跳 /login、无权限路径仍跳 /dashboard、**与 v1.23 FAIL-1 分页持久化恢复强耦合（75-UAT.md:76）须一起验收**（✅ 2026-10-28 Phase 81 完成（commit 1ce64d85）：catch-all 区分「菜单未就绪」(InitializingFallback) 与「真无路由」(跳 dashboard) + saveLastPath 加 allMenus.length>0 守卫防自毁反写；LAST_PATH 工具抽出 lastPathStorage.ts（顺带清 3 处 react-refresh）；新增 3 条回归测试并做**红绿验证**（回退实现 2 条失败 / 修复后 3 条全绿））

### HYG — 规范与存量技术债

- [x] **HYG-01**: `ColumnConfigModal.tsx:136` 的 `Modal.confirm` **属合法调用**（`src/components/shared/` 命中 `eslint.config.js:271-283` 目录级封装层豁免 → 该处是豁免盲区「无人区」，不在 `exemption-reasons.json`）→ 补行内理由注释文档化；**不要**把该目录摘出豁免（会爆几十条 error）；若登记 json 必须双向同步 eslint 锚点（✅ 2026-10-28 Phase 82 完成：ColumnConfigModal.tsx:136 的 Modal.confirm 属封装层豁免（src/components/shared/ 目录级豁免块）→ 补行内理由注释文档化；未把目录摘出豁免）
- [x] **HYG-02**: lint 存量 **17 errors + 34 warnings** 全清零（目标 errors=0 / warnings=0；实测 `npx eslint .` = 51 problems）——17 errors 构成：12 exhaustive-deps + `set-state-in-effect`（DashboardView.tsx:161）+ `immutability`（useRealtimeUpdates.ts:119）+ `globals`（ReconciliationTab.test.tsx:51）+ `preserve-manual-memoization`（useRPAProgress.ts:78）+ `local/no-large-dropdown-list`（**契约组件 AppSelect.tsx:16 自己违规**）；⚠️ **lint 预算双写死必须同步改 0**：`package.json` 的 `--max-warnings 34` + `assert-lint-budget.mjs:20-21` 的 17/34，否则门禁形同虚设；⚠️ **AppSelect 契约与 `local/no-large-dropdown-list` 正面冲突**（规则要求必带 onSearch 服务端搜索，契约是客户端过滤 `showSearch ?? count>=8`）→ **需决策**：推荐规则加 `excludedFiles: ["src/components/ui/**"]`，否则 errors 降不到 0；**同文件约束**：修 `useRealtimeUpdates.ts:130` 的 immutability 可能触发/加剧 compiler 错误 → 同文件一起改一起验；`useCaptchaModals.tsx` 5 处 `message` 依赖逐处判语义（机械加依赖会引出新 error）；**门禁有效性回归**：故意注入 1 条 error 时 `npm run lint:ci` 必须 exit 1（✅ 2026-10-28 Phase 82 完成：eslint **0 problems**（原 51 = 17 errors + 34 warnings）——exhaustive-deps 11 处语义清偿 + 其余 4 errors（set-state-in-effect/immutability/globals/preserve-manual-memoization）+ AppSelect 契约与 local/no-large-dropdown-list 冲突按既定决策 excludedFiles; 预算双写归零（package.json --max-warnings 0 + assert-lint-budget maxErrors/maxWarnings=0）+ 门禁有效性回归通过）
- [x] **HYG-03**: `react-refresh` **33 处**清零（与 barrel index.ts 无关，全是 .tsx）——**三类分流**（P2 修法经同版本插件沙箱实测 9 样例验证）：（✅ 2026-10-28 Phase 82 完成：react-refresh **33 处清零**——P2 规则误判 13 处（3 个伪 constants.tsx 改字面量拷贝，实测 0 警告）+ P1 真问题拆分（hook/context/presets/categories 独立文件）+ 冻结契约登记豁免（PortWriteModal/statusHelpers/ActionButtons）+ lazy() 盲区 2 处登记）
  - **P1 真问题 18 处/14 文件**（组件+非组件同文件）→ 拆分：工具/常量搬 `utils/` `constants/` `hooks/`；含 `ActionButtons.tsx:114/:167`、`FormModal.tsx:61`、`statusHelpers.tsx:63/:68`（冻结契约，登记优先）、`PortWriteModal.tsx:78`（v1.19 冻结面，**只登记不拆**）、`DynamicRoutes.tsx:85/:95/:108`（**与 NAV-01 同文件建议合并改动**）、`ColorSwitcher`/`LayoutProvider`/`ThemeProvider`/`ReconciliationTab` 等
  - **P2 规则误判 13 处/3 文件**（workstations/dept/dict 伪 constants.tsx，零组件导出；触发点 = `export const X = Y`（Identifier 初始化 + 大写名）被判为组件导出而连坐）→ **必须改代码、绝不用豁免**；实测 0 警告路径 = 改**字面量拷贝**（`export const X = [...Y]`）或改小写名或迁 `.ts`（规则只扫 .jsx/.tsx）；⚠️ **`export { Y as X }` 重导出无效（仍 3 警告）**——网上常见建议在此是错的；字面量拷贝前确认共享常量无就地 mutate（已 grep 未发现，但失去引用相等性）
  - **P3 `lazy()` 盲区 2 处**（BuildingModel3D:316 / FloorPlan3D:693）→ 文件级 disable + 登记（合理例外）
  - 通用约束：`export { X }` → `export type { X }` 是 `verbatimModuleSyntax:true` 强制写法、运行时无差别（逐个确认纯类型；测试文件不在 type-check 范围须额外跑 vitest）；`allowConstantExport:true` **不豁免**数组/对象字面量；**契约层豁免名单**（shared/ui）划登记；`export * from` 直接报 exportAll 需改具名；**eslint.config.js 锚点已漂移**（现 457 行，P66 :347、P65 :407）→ 拆分必须同批跑 `gen-p65-exemptions.mjs` + 同步 hex-exemptions；`.tsx → .ts` 降级前 grep 显式扩展名引用
- [x] **HYG-05**: **豁免理由表失同步修复**——`node scripts/check-exemption-reasons.mjs` 当前 **exit 1 / 123 项失配**（理由清单 159 条 vs 实际豁免 P65 1 + P66 37 = 38）→ 68.1 §8 #12/#18「豁免清单 100% 带理由」口径已不成立（v1.23 UAT 只验了 gen 脚本 diff，**没跑这个校验脚本**，且该脚本未接入任何门禁）→ 修复清单失配 + 把校验接入 `lint:ci` 防再漂移（✅ 2026-10-28 Phase 82 完成：豁免理由表全量同步 **159 → 38 条**（P65 1 + P66 37），`check-exemption-reasons.mjs` 校验 **exit 0 / 一一对应**，且已接入 `lint:ci` 防再漂移）
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
- **v1.25 候选③**：**权限模型的三处剩余缺口**（2026-10-28 Phase 79 PG 实测新发现，与 #16 同族但独立）——
  ① **数据范围（data_scope）未被列表查询消费**：`DataScopePermission` 中间件只把 scope 塞进 context，查询侧未使用 → 实测夹具角色 `data_scope=3`（本部门）时 `POST /system/users/list` 仍返回 `total=4231` 的**跨部门**用户（含 admin）
  ② **`GetUserPermissions` 不过滤 `menu.status`**（`internal/services/system/menu_service.go:547-550` 只过滤 perms 非空 + 软删）→ **已停用菜单的权限码仍会进前端权限集**，菜单停用后前端按钮可能仍显示
  ③ **Step 2「子菜单继承」语义易写反**：真实实现是「用户持有 F 型按钮 → 继承父页面权限」，**不是**反向；任何复刻/审计该逻辑的实现都可能得出相反结论（本轮曾因此得到错误中间态）
  ④ 附带发现（部署/运维缺口）：**生产库无迁移记录表** → 迁移版本无法核对「已应用 vs 仓库最新」；**GORM 模型与库表 schema 漂移**（`sys_role_menu` / `sys_user_role` 的 `id`/`created_at` 未映射，靠 DB default）
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
| VIS-01 | 76 | Satisfied (2026-10-28) |
| VIS-02 | 76 | Satisfied (2026-10-28) |
| VIS-03 | 76 | Satisfied (2026-10-28) |
| VIS-04 | 76 | Satisfied (2026-10-28) |
| VIS-05 | 77 | Pending |
| PERM-01 | 78 | Pending |
| PERM-02 | 78 | Pending |
| PERM-03 | 78 | Pending |
| PERM-04 | 78 | Pending |
| PERM-05 | 79 | Pending |
| PERM-06 | 78 | Pending |
| DIAG-01 | 80 | Satisfied (2026-10-28) |
| NAV-01 | 81 | Satisfied (2026-10-28) |
| HYG-01 | 82 | Satisfied (2026-10-28) |
| HYG-02 | 82 | Satisfied (2026-10-28) |
| HYG-03 | 82 | Satisfied (2026-10-28) |
| HYG-05 | 82 | Satisfied (2026-10-28) |
| HYG-04 | 83 | Pending |

**Coverage:**
- v1.24 requirements: 17 total
- Mapped to phases: 17（VIS×5 → 76/77 · PERM×6 → 78/79 · DIAG → 80 · NAV → 81 · HYG×5 → 82/83）
- Unmapped: 0 ✓

---
*Requirements defined: 2026-10-28*
*Last updated: 2026-10-28 初版（STACK + ARCHITECTURE + PITFALLS 三维研究完成并交叉验证；FEATURES 研究进行中，定稿时校订）*
