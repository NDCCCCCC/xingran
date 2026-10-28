# Milestone v1.25 Requirements — 全站权限治理与样式收敛

> **Defined**: 2026-10-28
> **Core Value**: 把 v1.24 只在 `/system/users` 单域完成的「操作级 RBAC 前后端双端闭环」推广到全站 76 页面 / 134 处后端路由挂载，并把样式层从手写 `!important` 规则收敛到 antd 6 官方 variant 机制 —— 让「谁能做什么」在前后端同一套语义下成立，让「页面长什么样」由 antd 6 的 theme token 单一决定。
>
> **事实基线**: `.planning/milestones/v1.24-REQUIREMENTS.md`（Future Requirements 段五项候选原文）｜ `.planning/notes/v1.24-css-audit-register.md`（36 条 CSS 登记）｜ `.planning/notes/2026-10-28-pg-permission-baseline.md`（#16 定责 + 生产 PG 权限矩阵）｜ `.planning/notes/v1.24-visual-verification.md`（21 形态 × 明暗矩阵 + 17 张证据）｜ `.planning/research/{STACK,PITFALLS}.md`（② 定向研究：antd 6.6.4 源码实证 + 项目 CSS 冲突面盘点）
>
> **立项来源**: v1.24 收档 Known Gaps 五项，**用户裁定全部立项**（2026-10-28）；执行顺序 **①权限 → ②样式**（用户裁定，避免样式波动与权限改造混在同一验证周期）。

## 起点基线（本次实测复核，非转述）

| 维度 | 实测值 | 判定 |
|------|--------|------|
| 前端业务页面 | `src/pages/**/index.tsx` **76 个** | 其中仅 **14 个**接入 `useMenuStore`（v1.24 口径 12） |
| 前端样式规模 | `src/index.css` ≈ 5000+ 行，`!important` **1359 处**，antd 选择器 **219 行**，暗色重复分区 **≈60** | 手写覆盖压过 antd 6 契约 |
| 后端路由组 | `internal/api/router.go` **76 个 Group** | 绝大多数为 ANY-of 组级语义 |
| 后端权限挂载 | `RequirePermissions` **90** + `RequirePermissionsWithQuery` **8** + `RequireStrictPermissions` **1** + `RequireAllPermissions` **0** = **134 处** | 仅 `system/users` 写组用了严格模式 |
| 严格权限覆盖 | `RequireStrictPermissions` **实际使用 1 处** | `/system/users` 写子组 |
| 动作映射死表 | `pkg/permission/config.go` 的 `GetRoutePermissions()` / `GetPermissionByPath()` **零调用点** | 表已写好但从未接线 |
| data_scope 消费 | `ApplyDataScope` / `ApplyDataScopeFromContext` **零调用点** | 中间件把 scope 塞进 context，下游查询无一消费 |
| 巨型文件 | `src/pages/operations/room-devices/index.tsx` **815 行** | 比已拆分的 devices 页（714→306）更长 |

---

## RBAC — 操作级权限全站治理

> **病灶定位（v1.24 已实证，本里程碑推广）**：#16「只读角色仍见写操作入口」是**两个独立缺陷叠加**——① 前端 45 个含「操作」列的文件完全不引用 permissions（`ActionButton` 契约原本没有 permission 字段）；② 后端 39 处路由组全为 ANY-of + `permission.go` Step 3 模块级降级 + Step 2 的 F→C 继承放大。v1.24 在 `/system/users` 单域用 `RequireStrictPermissions` + 读/写兄弟组分离 + `useActionPermission` 三态门控治好了这一域，**其余全域仍是旧病灶**。

- [ ] **RBAC-01**: 只读角色在**任意业务模块**调用写接口一律被拒（HTTP 403），且不产生任何数据变更 —— 全站写操作路由不再使用 ANY-of 组级语义，逐动作绑定单一权限点
- [ ] **RBAC-02**: 前端 76 个业务页面的行操作 / 批量操作 / 工具栏写入口按**动作级权限码**渲染：无权限 → 不渲染；权限未就绪（加载中 / 接口失败）→ 渲染但禁用并给出可访问的提示
- [ ] **RBAC-03**: `pkg/permission/config.go` 的动作级映射表被真实接线并可校验：新增或改动的写路由若未登记到映射表，校验脚本失败（防止映射表再次退化为死表）
- [ ] **RBAC-04**: `system` 域其余 4 组（roles / menus / depts / posts）写操作从 ANY-of 拆为逐动作单一权限点，与 `/system/users` 的已验证范式一致
- [ ] **RBAC-05**: 动作级权限**可被真实授予**：动作级 F 型菜单播种后，管理员在角色页勾选动作权限 → 该角色用户在前端看到对应操作入口 → 后端同步放行（前端可见性与后端放行同源于一套权限码）
- [ ] **RBAC-06**: 权限码单一真相源（前端 `src/constants/permissionCodes.ts` ↔ 后端 `pkg/permission` 常量 ↔ 菜单 F 型种子）三方零漂移，校验脚本覆盖本轮新铺开的面并接入 CI 门禁

## STYLE — 样式层收敛到 antd 6 variant

> **前置障碍（F3）**：`AntdThemeBridge.tsx:83-84` 显式设 `colorBgContainer='#FFFFFF'` / `colorBorder='#dbd7ce'`；antd `darkAlgorithm` **不改写显式传入的 token** → 暗色下容器仍白底 + 浅灰边，同页并存两套边框体系。**必须先重构明暗色板，才能谈 variant 收敛**。

- [ ] **STYLE-01**: `AntdThemeBridge` 明暗色板重构——品牌 token 交由算法派生而非硬编码，暗色下 `colorBgContainer` / `colorBorder` 正确派生，**暗色 Select 等容器不再白底**，同页不再并存两套边框体系
- [ ] **STYLE-02**: 手写 antd 覆盖规则按四批清单收敛：**可整块删除**批清零（antd 6 已不存在的选择器 + 零命中死规则）、**可降级为 token 引用**批完成替换、**需重写为 variant 机制**批完成重写；**必须保留**批逐条登记理由（含 v1.24 已交付的 wrapper 复位规则与焦点环）
- [ ] **STYLE-03**: 输入类组件圆角统一由 6px 收敛到设计 token 的 8px
- [ ] **STYLE-04**: hex 治理覆盖 `.css` 文件（现 eslint selector 只匹配 JS/TS 节点，对样式文件失明），**收敛过程不得新增硬编码色**
- [ ] **STYLE-05**: 收敛后明暗双主题下输入类 / 选择类 / 表格类组件与 antd 官方渲染一致——computed-style 断言 + 截图矩阵双测通过，且**不破坏 v1.24 已交付的 VIS-01/02/03 修复**（双框消除 / 焦点环单圈 / 错误态边框色）

## PMODEL — 权限模型三缺口

- [ ] **PMODEL-01**: `data_scope` 被列表查询**真实消费**：`data_scope=3`（本部门）的用户查询用户列表只返回本部门数据，不再返回跨部门全量（v1.24 实测该角色仍拿到 `total=4231` 跨部门数据）
- [ ] **PMODEL-02**: `GetUserPermissions` 过滤 `menu.status`——已停用菜单的权限码不再进入前端权限集（菜单停用后其按钮不再显示）
- [ ] **PMODEL-03**: Step 2「子菜单继承」语义**文档化并以明确命名/注释固化**（真实语义是「用户持有 F 型按钮 → 继承父页面权限」，不是反向），杜绝复刻实现得出相反结论
- [ ] **PMODEL-04**: 生产库迁移版本可核对（补齐迁移记录表），且 GORM 模型与库表 schema 漂移项（`sys_role_menu` / `sys_user_role` 的 `id` / `created_at` 未映射，靠 DB default）被登记为显式债务

## VISFIX — Phase 77 视觉新发现

- [ ] **VISFIX-01**: 暗色模式下输入框底色不再被 `--input-bg` 第 3 处就地重定义（`index.css:4807`）覆写回浅色 —— 暗色输入框底与页面底对比度达标（v1.24 实测 `rgb(250,250,249)` 压在 `rgb(15,46,27)` 上）
- [ ] **VISFIX-02**: 暗色模式 Select 容器底色与边框随色板派生，不再白底 + 浅灰边（与 STYLE-01 联动，一次改动同时消两债）
- [ ] **VISFIX-03**: `.dept-tree-search` 死规则清理（v1.24 实测命中数 0 —— 后代选择器要求祖先持有该 class，实际 class 落在 AppInput 自身上）
- [ ] **VISFIX-04**: antd 6 弃用警告 2 类清零（`onSearch→showSearch`、`Modal maskClosable→mask.closable`）——「无新增 console error」的自动化判定不再被污染
- [ ] **VISFIX-05**: InputNumber 错误态边框色还原，与 Input / TextArea 错误态一致（v1.24 登记册 #16 欠账）

## SPLIT — 巨型文件拆分

- [ ] **SPLIT-01**: `src/pages/operations/room-devices/index.tsx` 815 行按 v1.24 devices 页范式拆分（列定义工厂 / 搜索区构造函数 / handlers+行操作 / 页面 JSX 外壳），主文件 ≤ 350 行，每个新文件 ≤ 250 行，且拆分**不改变任何行为**

---

## 关键决策 (v1.25 init, locked)

| 决策 | 理由 |
|------|------|
| **五条候选全纳入 v1.25**（用户裁定） | ①②③④ 是技术债清偿、⑤ 是小体量拆分，同族可打包；拆成两个里程碑会让样式债与权限债互相等待 |
| **执行顺序 ①权限 → ②样式**（用户裁定） | 权限改造动 JSX、样式收敛动 CSS，文件面小重叠；权限先行可复用 v1.24 三态门控基建，避免样式波动与权限改造混在同一验证周期 |
| **研究范围仅 ② 定向**（用户裁定） | ①③⑤ 靠本仓事实与既有实证 note；仅 ② 的 antd 6 官方 API 语义需外部核验，避免为已知债务做无产出的域外研究 |
| **复用 v1.24 基建不重造** | `RequireStrictPermissions`（只做 Step1 精确匹配，跳过 F 型继承与模块降级）/ `useActionPermission` 三态门控 / `permissionCodes.ts` 单一真相源 + 双向校验脚本——均为 v1.24 已验证资产 |
| **STYLE-01 必须先于 STYLE-02** | 显式 token 击穿 `darkAlgorithm` 是 F3；在色板重构前做规则收敛，会把暗色炸白底固化进新规则 |
| **门控取三态而非纯 fail-closed** | 沿用 v1.24 决策：空权限集与未加载在数据上不可区分，纯 fail-closed 会在接口失败时把管理员也锁死且无提示 |
| **零新依赖** | 沿用 v1.23 / v1.24 原则；不引 stylelint / Playwright / jscodeshift；视觉验证用现有 computed-style 断言 + agent-browser |
| **不破坏 v1.22 / v1.24 冻结契约** | v1.22 公共组件契约与 v1.24 VIS-01/02/03 修复均为已验收资产；STYLE-01 涉及 `AntdThemeBridge` 色值真相源（v1.22 冻结面），须单独登记与验收 |
| **回归基线** | eslint **0 problems** / vitest 569 文件 3960 用例全绿 / vendor-react gzip 776 kB / `go build ./...` + `go test` 全绿 |

## Future Requirements（延后到 v1.26 或独立归档）

- **sqlite `ON CONFLICT` 方言泄漏**（`role_service.go:425-430` + `migration_206:29-33` 的 `isPostgreSQL` 守卫 + `models/role.go:24-27` 无唯一索引）——sqlite 下无法创建带菜单授权的角色，也是 #16 曾在 sqlite 无法复测的根因
- **权限缓存陈旧**——后端 30s TTL 且 `InvalidateMiddlewarePermCache` 未接线（`permission.go:68`）；前端双层缓存（menuStore 5min + sessionStorage 30min）；`sys_menu.perms` 整串取出不拆逗号（`menu_service.go:547-552`）
- **`ops_workstation_device.responsible_user_id` 恒空**（445/445，写入侧从未回填 uuid）
- **React Compiler 纯渲染约束适配**（eslint-plugin-react-hooks v7.0.1 新增规则族）
- **双 ConfigProvider 嵌套**（`src/App.tsx:9,48` 项目自有 Provider vs `AntdThemeBridge.tsx:178` 真 antd Provider）
- **3f130 绑定 user_id**（业务核对后由用户执行）
- **文档漂移**——CLAUDE.md 写 antd 6.1 / React 19.2，实际 `^6.6.4` / `^19.3.0`；`75-UAT.md:66` 记 `optionFilterProp="value"` 而代码实际 `"label"`
- **AD 方向补齐**（`ad_push_queue` 缺 enable 方向，见 `.planning/seeds/SEED-ad-push-enable-direction.md`）
- **残留现场 UAT 债**（v1.18 三项真机 SNMP/CLI + v1.19 六项 SSH + v1.20 十二项 VLAN/绑定 site-visit）

## Out of Scope

| Feature | Reason |
|---------|--------|
| 破坏 v1.22 冻结契约（AppInput/AppSelect/AppTextArea 默认值、ConfirmDelete、handleApiError、statusHelpers、PageTable/FormModal） | 锁定契约 |
| 破坏 v1.24 已交付的 VIS-01/02/03 修复 | 已验收资产 |
| 引入 stylelint / Playwright / jscodeshift / 任何新依赖 | 零新依赖 |
| 动生产数据（角色调整、用户数据写入、3f130 绑定） | 用户裁定：只做代码与只读诊断 |
| 业务功能新增（工单、报表、第三方集成等） | 本里程碑只清偿已知债务 |
| AD 方向补齐 / 同组管理员 OU 写权限 / 同步节流与可观测性 | v1.21 遗留，独立立项 |
| 现场 UAT（真机 SSH / SNMP / VLAN 绑定） | 需现场运维同事执行 |
| 全站 3D 可视化 / 移动端 / AI 能力 | 与本里程碑无关 |

## Traceability

> 由 roadmap 阶段填充（每条需求映射到唯一 phase）。

| Requirement | Phase | Status |
|-------------|-------|--------|

**Coverage:**
- v1.25 requirements: 25 total（RBAC×6 · STYLE×5 · PMODEL×4 · VISFIX×5 · SPLIT×1）
- Mapped to phases: 待 roadmap 填充
- Unmapped: 待 roadmap 填充

---
*Requirements defined: 2026-10-28*
*Last updated: 2026-10-28 初版（起点基线为本次实测复核：76 页面 / 76 路由组 / 134 处权限挂载 / 严格权限 1 处 / 死表零调用点 / data_scope 零消费点；STYLE 量化目标待 ② 定向研究回填后校订）*
