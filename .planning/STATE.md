---
gsd_state_version: "1.0"
milestone: v1.52
milestone_name: 运维域数据权限统一收口（终态）
status: v1.52 SHIPPED（2026-11-07）—— operations 全模块声明式 data_scope 统一收口，Phases 130-133 全部完成（7/7 REQ）
stopped_at: v1.52 全量交付（PR #75 merged `b56f1286`，4 phase / 24 commits / 全量闸门全绿）；收档文档走 `feat/v1.52-data-scope-unification` 分支 PR
last_updated: "2026-11-07T13:35:00+08:00"
last_activity: 2026-11-07
last_activity_desc: v1.52 立项：运维域数据权限统一收口（终态），Phases 130-133（DS-01~07）
state_head: 9f511268a7d748b0707836d95963dbf79842b1f6
progress:
  total_phases: 4
  completed_phases: 4
  total_plans: 0
  completed_plans: 0
  percent: 100
current_phase: 133
current_phase_name: ScopeProvider 基建（可见部门集统一实现 + strict 组注入 data_scope/user_id）
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.52 milestone active（运维域数据权限统一收口·终态，Phases 130-133 未开始）
**Last activity**: 2026-11-07 - v1.52 立项：运维域数据权限统一收口（终态，DS-01~07 / Phases 130-133）；同日前置 quick 三件套（261106-wsd 导入反推 / 261106-wso 楼层机构覆盖 / 261106-wsb 回填按钮 + 独立权限项 + P2 修复）已合入 PR #75
**Prior activity**: 2026-11-06 - Completed quick task 261106-wso: 一栋楼跨机构办公 —— 楼层可覆盖所属机构（ops_floors.org_id + 全链路 COALESCE）
**Prior activity**: 2026-10-31 — v1.51 立项（范围 A+B+D，21 REQ / Phases 121-129）
**Prior activity**: 2026-10-30 — v1.25 SHIPPED & ARCHIVED 并完成 GitHub 同步（PR #36/#37 合并，main CI/deploy 全绿，技术债 D1-D10 清偿）
**Prior activity**: 2026-10-29 — Phase 86（W4+W5）与 Phase 90（W0~W3）全部收口并验收；下一步 Phase 87
**Prior activity (2)**: 2026-10-28 — v1.24 全量执行完成（Phase 77 视觉验证 + Phase 83 拆分收口 + VIS-03 死规则修复）
**Prior activity (2)**: 2026-10-28 — v1.24 执行期：78 权限双端闭环（后端严格中间件 + 前端三态门控）+ 79 PG 实机验收（三层证据）+ 82 lint 清零
**Prior activity (3)**: 2026-10-28 — v1.23 SHIPPED & ARCHIVED（Phases 69-75, 22/22 REQ satisfied）
**Prior activity (4)**: 2026-10-27 — v1.23 roadmap CREATED（Phases 69-75, 22/22 REQ mapped）

## Current Milestone: v1.52 运维域数据权限统一收口（终态）(Phases 130-133)

**Core Value**: 把数据权限从「用户/VDI 两个模块的局部实现」升级为运维域全模块的声明式统一收口 —— 读写两端按登录人数据范围自动过滤与校验，使「中支管理员只能管本机构的工位/楼层/资产」从口头约定变为架构保证。

**立项动因（2026-11-07）**: v1.51 期间实测确认 operations 全模块未接 data_scope —— 工位 List orgId 由前端传、后端不校验；strict 权限组不注入 data_scope；三份 data_scope 实现（user_service / vdi / pkg/middleware）并存，其中 pkg/middleware 为无生产调用方的死实现。quick-261106-wsb 审计裁定逐端点手工接线为补丁非终态，用户决策「直接建立新里程碑，做终态」。

**事实基线**:
- `.planning/REQUIREMENTS.md`（**7 REQ**：DS-01~07，含验收硬约束）
- 可复用地基（quick-261106-wsd/wso/wsb 产物）：`effectiveOrgExpr` 有效机构表达式单一真实源（dept_filter.go）、`queryUserDeptMap` 用户→部门映射共享实现（reference_resolver.go）、`BuildDeptRecursiveFilter` 子树展开、PG contract 类型契约测试门控、独立按钮权限码 seed 范式（migration 235）
- 待治理清单：工位 List orgId 后端不校验｜Create/Update 的 deptId/orgId 可传任意 UUID｜strict 组不注入 data_scope｜三份 data_scope 实现（两处生产调用方 user_service / vdi + 一处无生产调用方的死实现 pkg/middleware）范围展开逻辑重复
- **迁移编号**：全库最大 235（migration_235_workstation_backfill_perm.go），v1.52 新迁移从 236 起编（无冲突）。

### v1.52 硬约束

1. **uuid/varchar 列比较双侧 CAST AS TEXT；SQLite/PG 双方言** —— 新收口查询必须有 PG contract 或等价双方言测试（门控：TEST_PG_DSN）；**导出/导入路径须纳入 PG contract 覆盖**（现状 `pg_contract_test.go:87-208` 仅覆盖 List/Statistics/SearchOptions/Backfill）；新增测试统一门控变量名 `TEST_PG_DSN` 或显式兼容 `XINGRAN_TEST_PG_DSN`，CI 中 skip 必须可观测
2. **「有效机构」表达式走 `effectiveOrgExpr` 单一真实源**，禁止内联复制（P2-6 教训：内联即漂移）
3. **data_scope fail-closed 对齐 PMODEL-01** —— **0/unset 映射 All（超管零回归），仅非 0 异常枚举 fail-closed**；多角色取 MAX（getUserMaxDataScope 语义不变，**MAX=数值最大=最严档 1=All…5=Self**，配多角色组合 / 无角色 COALESCE→Self(5) 边界用例）
4. **缓存 key 必含数据范围维度段**（PMODEL-01 跨范围泄漏教训）；All 档特判短路路径同样可断言；**Phase 131 必交 ops 缓存面清点表**（逐 key 现状→目标，`floor_cache_impl.go:39-45 floor:tree` 固定 key 因按可见集过滤必须改结构或停用）
5. **回归基线（每 phase 复核）** —— eslint 0 problems / vitest 全绿 / `go build` + `go test` 全绿（含 pkg/permission 路由双层守卫、internal/api strict 守卫）

### 范围裁定（已前置为 phase 输入，2026-11-07 审计修订 P1-4）

- **Self 档 ops 域语义** → 前置为 **Phase 130 的开工输入**（ScopeProvider 基建即需 Self/All 资源语义钩子定义；若把这项裁定拖到 Phase 133 才定则过晚，133 本身不承载任何裁定动作）
- **Dept/Self 批量 + Excel 导入豁免** → 前置为 **Phase 132 的开工输入**（backfill 批量收口在 132 涉及）
- **DS-05 三项裁定**（dedicatedLine/roomDevice 归属链、workstation-device 归属、location-alias 查询面）→ 前置为 **Phase 133 的开工输入**，须在 133 开工前产出书面结论并落 `.planning/notes/`；133 仅按已裁定结论落地 + UAT，**不承载裁定动作**
- Phase 133 改为「按已裁定结论落地 + UAT」，不再承载裁定本身

> ⚠️ 上述两项裁定（Self 档 ops 语义 → 130；Dept/Self 批量 + Excel 导入豁免 → 132）须在对应 phase 开工前产出书面结论并落 `.planning/notes/`；未产出前 `/gsd-plan-phase 130` 仅可起草不含 Self 定案的 ScopeProvider 骨架。

### 历史目录编号消歧（立项必读）

- `.planning/phases/130-device-evidence/`（含 `130-EVIDENCE-PLAN-NO-DEVICE.md` / `130-RUIIJIE-TRUNK-ALLOWED.md`）是 **v1.51 Phase 129 真机 UAT / 端口写设备取证的遗留归档**（裁定日期 2026-11-02），其 "130-" 前缀为当时的任务编号、**非 GSD phase 号**，与 v1.52 Phase 130（ScopeProvider 基建 / DS-01/02）**无任何语义关联**。
- 该历史目录**保留不改名**（避免破坏 v1.51 归档可追溯性），不作为 v1.52 任何 phase 的产物目录。
- **v1.52 Phase 130 的 GSD 产物统一落新目录 `.planning/phases/130-scope-provider/`**；执行 `/gsd-plan-phase 130` 以此为准，**不得写入 `130-device-evidence/`**。

**历史里程碑:** v1.51 端口写生产级操作体验（Phases 121-129，21 REQ 全交付，tag v1.51.1/v1.51.2，需求归档 [v1.51-REQUIREMENTS.md](milestones/v1.51-REQUIREMENTS.md)）；更早见 ROADMAP.md Milestones。


## Current Position

Phase: Milestone v1.52 **SHIPPED**（运维域数据权限统一收口·终态，Phases 130-133 全完成）
Plan: —
Status: v1.52 全量交付完成（2026-11-07）：4 phase / 7 REQ / 24 commits，PR #75 merged `b56f1286`；全量闸门全绿（go build / go test / eslint 0 / vitest 592 文件 4128 用例 / format / lint:ci / 生产构建 / diff coverage）
Last activity: 2026-11-07 — v1.52 收档（ROADMAP 4 phase 勾选 shipped、STATE 进度 100%、里程碑总结落盘）；收档文档走 `feat/v1.52-data-scope-unification` 分支 PR

---

## 历史归档（v1.25 及更早，已 SHIPPED / ARCHIVED；以下为非当前状态，仅供追溯）

> 本节及后续子标题（进度快照、Phase 依赖速查、执行期决策、立项范围、Accumulated Context 等）为 v1.25 及更早里程碑的归档记录，**均非当前状态**；当前状态一律以文档上半段 v1.52 active 与 `## Current Position` 为准。

### v1.25 真实进度快照（历史，2026-10-29 晚，Lead 实跑复核）

| Phase | 主题 | Plans | 已执行 | 验证状态 | 状态 |
|---|---|---|---|---|---|
| 84 | 权限模型三缺口 | 4 | 4/4 | human_needed | ✅ 执行完成（2 项 PG 实机验证待用户执行，方法见 `.planning/notes/2026-10-29-pg-data-scope-verification.md`） |
| 85 | 后端动作级严格化 | 4 | 4/4 | passed | ✅ 完成 |
| 86 | 前端操作入口三态门控铺开 | 6 | 6/6 | passed | ✅ 完成（`check:gating` 恒等式 `25 已门控 + 51 豁免 = 76` 被脚本硬断言守护 + 红绿自检；已挂 `lint:ci`） |
| 87 | 动作级权限授予闭环 | 0 | — | — | ○ 未开始（历史状态；依赖 85 + 86 → 当时两前置均已就绪、曾可立即开工） |
| 88 | 验证底座与零风险清理 | 4 | 4/4 | passed | ✅ 完成 |
| 89 | 变量层与明暗色板重构 | 4 | 4/4 | passed | ✅ 完成 |
| 90 | Input 域交还 + 圆角统一 | 4 | 4/4 | human_needed | ✅ 执行完成（矩阵终跑 PASS / 权威断言 passed / 11 项闸门全绿；40 个巡检组合待人工） |
| 91 | Select/Form/Table/Modal 域交还 | 0 | — | — | ○ 未开始（历史状态；依赖 90 → 当时硬前置已交付） |
| 92 | 暗色 mega-cluster 合并 | 0 | — | — | ○ 未开始（依赖 91） |
| 93 | room-devices 拆分 + 视觉回归验收 | 0 | — | — | ○ 未开始（依赖 92） |

> ⚠️ **plan 级计数口径失真（已知先例，非缺陷）**：本 milestone 起沿用 v1.24「subagent 直执行 + 按刀原子提交」先例 —— 每个 plan 由 subagent 直接执行并原子提交，**不产出 per-plan `SUMMARY.md`**，phase 级上下文改由 `NN-VERIFICATION.md`（REQ 对照表 + 闸门输出 + commit 链）承载。因此 GSD CLI 的 `completed_plans` / `percent` 恒为 0（它以 SUMMARY.md 存在性判定 plan 完成）→ **真实 plan 级完成度 = 24/28（86%）**，以本表为准。

> ⚠️ **VERIFICATION 状态值契约（2026-10-29 修复）**：84/85/88/89 四份 VERIFICATION 原用 `status: validated`（该值属补丁子系统枚举，非验证报告枚举），导致 GSD CLI 的 `init.progress` / `roadmap.analyze` / `verification.status` **全部硬失败**。已按闭合集合 `passed | gaps_found | human_needed` 规范化：85/88/89 → `passed`；84 → `human_needed`（2 项 PG 实机验证待用户执行，方法已登记在 `84-VALIDATION.md`）。

> ⚠️ **两项人工验证债（历史：当时唯一阻塞 Phase 87 全量推进的事项，v1.25 已 SHIPPED 归档）**：
> ① **Phase 84 的 PG 实机验证**（data_scope=3 复测 total + `sys_role_menu`/`sys_user_role` 列集实查）—— 本机无 PG/psql，且远程 PG 需密码（安全网拦截）⇒ 方法已整理为可直接执行的清单 `.planning/notes/2026-10-29-pg-data-scope-verification.md`（含 A~F 六步 + 三个已知坑：多角色 MAX 取值污染 / 30s TTL 假失败 / 403≠过滤失败）。
> ② **Phase 90 的 40 个巡检组合**（13 页 × 明暗）—— 本波已实现并跑通**滑块验证码自动识别 + 拖动 + 服务端校验**（`p90-captcha-solve.cjs` / `p90-login.cjs`，实测 verify/slider 返回 200），但登录被 AD 认证拒（生产库 `sys.auth.default.mode = ad`）⇒ 切换属生产数据写入，不擅自执行；待人工清单见 `90-visual-evidence/inspection-manual-todo-90.json`。

> 📌 **本波新增的自动化能力（历史，曾可复用）**：无登录态的 antd 组件取证链路（`p90-session.cjs` / `p90-matrix-run.cjs` / `p90-dark-authoritative.cjs` / `p90-contract-assert.cjs` / `p90-hover-focus-probe.cjs`）+ 滑块验证码缺口识别（`p90-captcha-solve.cjs`）⇒ 91/92 的视觉验收可**不再依赖登录态**（矩阵 456 格逐格 computed-style 比对比 14 页截图更强）。

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
- DB migration 编号已用到 **235**（migration_235_workstation_backfill_perm.go，v1.51 期）；**v1.52 新迁移从 236 起编**。

### Decisions

- [v1.21 遗留] 57-HUMAN-UAT.md 三项真环境验证（PG 合并路径 / kill -9 重启 / UAC 533）仍待人工执行；v1.18/1.19/1.20 的 site-visit UAT 债同样挂账

### Blockers

- 无

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 261102-m94 | 修复暗色模式保存失败：theme_style 违反 chk_theme_style 检查约束（省略即不修改 + 绑定与 DB 约束对齐 + 迁移 233） | 2026-11-02 | 9f511268 | [261102-m94-theme-style-chk-theme-style](./quick/261102-m94-theme-style-chk-theme-style/) |
| 261106-wsd | 工位导入部门填写不规范时按所属人员反推部门（仅兜底 + 反推计数 + 6 个回归测试） | 2026-11-06 | 7259cf90 | [261106-wsd-workstation-dept-infer-from-user](./quick/261106-wsd-workstation-dept-infer-from-user/) |
| 261106-wso | 一栋楼跨机构办公：楼层可覆盖所属机构（ops_floors.org_id + migration 234 + 工位/楼层/楼宇全链路 COALESCE + 楼层表单 + 10 个回归测试） | 2026-11-06 | 7259cf90 | [261106-wso-workstation-floor-org-override](./quick/261106-wso-workstation-floor-org-override/) |
| 261106-wsb | 工位管理页「按人员反推部门」按钮（预览+确认两步、默认仅兜底+可选纠正不一致、复用 edit 权限、严格写组+operlog+缓存失效+route-registry 登记） | 2026-11-06 | 7259cf90 | [261106-wsb-workstation-backfill-dept-button](./quick/261106-wsb-workstation-backfill-dept-button/) |

### Todos（v1.52 / Phases 130-133）

- **两项开工输入裁定**（结论落 `.planning/notes/`）：① Self 档 ops 语义 → **Phase 130 开工输入**；② Dept/Self 批量 + Excel 导入豁免 → **Phase 132 开工输入**；未产出前对应 phase 不得定案
- **DS-05 三项裁定结论**（dedicatedLine/roomDevice 归属链、workstation-device 归属、location-alias 查询面）→ **Phase 133 开工输入**，133 开工前产出，133 不承载裁定动作
- **Phase 130**：ScopeProvider 基建（可见部门集统一实现 + `effectiveOrgExpr` 单一真实源）+ strict 组注入 data_scope/user_id + 超管 / 0 值边界用例；产物落 `.planning/phases/130-scope-provider/`
- **Phase 131**：四域读收口 + **Excel 导出独立收口** + **缓存面清点表（必交产物）** + 跨用户对照实验 + PG contract（含导出/导入路径）
- **Phase 132**：四域写收口 + backfill 范围（依赖 132 开工输入裁定）
- **Phase 133**：按已裁定结论落地（DS-05 开工输入）+ 四实体收口 + vdi 迁移 + pkg/middleware 死实现 / 死测试删除 + 前端联动 + UAT
- **PG contract 变量名统一 `TEST_PG_DSN`**（131/132/133 一致）
- **新迁移从 236 起编**（全库最大 235，无冲突）

### Roadmap Evolution

- Phases 84-93 created for v1.25「全站权限治理与样式收敛」: 84 权限模型三缺口（PMODEL-01~04）→ 85 后端动作级严格化（RBAC-01/03/04）→ 86 前端操作入口三态门控铺开（RBAC-02）→ 87 动作级权限授予闭环（RBAC-05/06）→ 88 验证底座与零风险清理（VISFIX-03/04/06 + STYLE-02 第 1 批）→ 89 变量层与明暗色板重构（STYLE-01/04 + VISFIX-01/02）→ 90 Input 域交还 variant/token + 圆角统一（STYLE-03 + VISFIX-05）→ 91 Select/InputNumber/Form/Table/Modal 域交还（承接 STYLE-02 R8 + VISFIX-04 剩余）→ 92 暗色 mega-cluster 合并（STYLE-02 主责）→ 93 room-devices 拆分 + 视觉回归验收（SPLIT-01 + STYLE-05）。**22/22 需求映射，无孤儿无重复**；线 A 权限 84-87 ∥ 线 B 样式 88-93，线内串行；写死三处不可逆顺序约束（89→90/91/92、批次 C→批次 B 同 commit、变量层→变体交还）。 (CRITICAL)
- Phases 76-83 created for v1.24「视觉回归修复与遗留债务清偿」: 76 双框 P0 修复 + CSS 审计登记册（VIS-01~04）→ 77 视觉回归验证收口（VIS-05）→ 78 #16 前后端双端闭环（PERM-01~04/06）→ 79 PG 实机验收（PERM-05）→ 80 3f130 空因诊断前端接入（DIAG-01）→ 81 F5 刷新落 /dashboard 修复（NAV-01）→ 82 lint 清零 + 豁免理由表修复（HYG-01/02/03/05）→ 83 devices 页拆分（HYG-04）。18/18 需求映射，无孤儿无重复；**全部 SHIPPED 2026-10-28**（tag `v1.24`）。 (CRITICAL)
- Phases 69-75 created for v1.23「前端契约全量收尾与遗留清偿」: 69 后端 debug 重验 spike（DBG-01）→ 70 TextArea 契约基建 W0（TEXT-01/03/04 + HYG-05）→ 71 TextArea 存量 + COMP-05 分域迁移 W1-W6（TEXT-02 + MIG-01/02/04）→ 72 后端 debug 修复 wave（DBG-02/03/04，#3 先于 #2）→ 73 DB 实测日 + cron.go 双执行调查（DBG-05/06/07）→ 74 COMP-05 收口 + lint 清偿 + quick 目录归档（MIG-03 + HYG-01~04）→ 75 浏览器实机 UAT + 收档（UAT-01/02）。22/22 需求映射，无孤儿无重复。 (CRITICAL)
- Phase 68.1 inserted after Phase 68: v1.22 收尾清偿 phase：危险操作收口 + 契约 minor 补课 + 豁免收缩 + 存量迁移收尾（DEBT-04） (URGENT)

## Project Reference

See: `.planning/PROJECT.md`（updated 2026-11-07 after v1.52 立项）

**Core value:** 端到端的运维可观测与可审计——每个写操作都留下可追溯记录（谁、何时、改了什么、从哪来、改前改后），敏感字段自动脱敏；配合 Excel 批量管理与引用自动解析（部门名 → UUID），运维人员对整个 IT 资产域拥有唯一可信的系统记录。
**Current focus:** v1.52「运维域数据权限统一收口（终态）」IN PROGRESS（Phases 130-133，7 REQ mapped）——下一步 /gsd-plan-phase 130（须先产出两项开工输入裁定，见下）。

## Session

**Last session:** 2026-11-07T13:35:00+08:00
**Stopped at:** v1.52 立项（Phases 130-133，7 REQ）；计划经 2 轮审计修订
**Resume file:** 无（下一步 /gsd-plan-phase 130）

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first:

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| verification_gaps | 84/84-VERIFICATION.md | human_needed | 2026-10-30 | v1.25 |
| verification_gaps | 87/87-VERIFICATION.md | human_needed | 2026-10-30 | v1.25 |
| verification_gaps | 90/90-VERIFICATION.md | human_needed | 2026-10-30 | v1.25 |

## Performance Metrics

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| v1.25（10 phase，planning 阶段） | 路线图 1 轮完成（2026-10-28） | 10 phase × 5 成功标准 | — |
| v1.24（8 phase，subagent 直接执行，按刀原子提交） | ~3.5 h（09:19→12:59，单日） | 8 phase × 9 项闸门 | 104（+5,419 −2,232，46 commits） |
| v1.23（7 phase） | 单日 | 121 文件迁移 | — |

## Operator Next Steps

- 产出两项开工输入裁定（Self 档 ops 语义 → 130；Dept/Self 批量 + Excel 导入豁免 → 132），结论落 `.planning/notes/`
- /gsd-plan-phase 130（ScopeProvider 基建）；其后 131 ∥ 132（函数级无交集约束）→ 133（须先产出 DS-05 三项裁定结论）
