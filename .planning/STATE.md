---
gsd_state_version: "1.0"
milestone: v1.55
milestone_name: LOCKED 残余根治
status: v1.55 SHIPPED 2026-11-09 (Awaiting next milestone)
last_updated: "2026-11-09T10:30:00.000Z"
last_activity: 2026-11-09
progress:
  total_phases: 7
  completed_phases: 7
  total_plans: 8
  completed_plans: 8
  percent: 100
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.55 SHIPPED 2026-11-09（7/7 Phase：145/146/147/148/149/150/151 全部完成；24/24 REQ 交付，含 CRYPTO-04 按硬 gate 合规跳过；**① 的 A(150)+B(151) 均完成 → P1-5 侧信道根治达成**；6 项 LOCKED / 7 子项全部从根源清偿）
**Last activity**: 2026-11-09
**Prior activity**: 2026-11-09 - v1.55 SHIPPED：W1（145 AUDIT / 146 CRYPTO / 147 CACHE）+ W2（148 DIALECT / 149 SCOPE / 150 侧信道 A）并行 + W3（151 统一 ScopedDB，单独执行）；3 波共 7 个 gsd-planner + 2 个 plan-checker + 7 个 gsd-executor + Lead 集成；全量回归闸门全绿（go build / go test ./... / tsc / lint:ci ALL PASS / 全量 vitest / check-route-registry 713 零漂移）；79 条人工验证跟踪表保持 0 勾选、验证类残余不伪造闭环
**Prior activity**: 2026-11-09 - v1.55 INITIATED：6 项 LOCKED 残余从根源根治（深度根因报告 + 5 项决策拍板 + 24 REQ 草案）；跳过领域研究；FUTURE-01 随 ① 关闭；79 条人工验证跟踪表保持 0 勾选
**Prior activity**: 2026-11-09 - v1.54 SHIPPED：4 项真实代码缺口全部清偿（发现设备真实导入 / 批量物理参数 + 型号级能力矩阵 / 告警前端+WS+webhook 收口），W1 4 个并行 executor + Lead 集成；`go build ./...` / `go test ./...` / `tsc` / `lint:ci ALL PASS` / 全量 vitest 全绿；权限三方逐字一致、migration 238 两处注册；验证类残余不伪造闭环
**Prior activity**: 2026-11-08 - v1.53 SHIPPED：8 个 Phase 全部交付，全量回归闸门全绿（go build / go test 全仓 / tsc / lint:ci / vitest 592 文件 4128 用例），GSD `milestone complete` 完成归档与权威交回
**Prior activity**: 2026-11-08 - v1.52 SHIPPED 并完成 GSD 归档
**Prior activity**: 2026-11-06 - Completed quick task 261106-wso: 一栋楼跨机构办公 —— 楼层可覆盖所属机构（ops_floors.org_id + 全链路 COALESCE）
**Prior activity**: 2026-10-31 — v1.51 立项（范围 A+B+D，21 REQ / Phases 121-129）
**Prior activity**: 2026-10-30 — v1.25 SHIPPED & ARCHIVED 并完成 GitHub 同步（PR #36/#37 合并，main CI/deploy 全绿，技术债 D1-D10 清偿）
**Prior activity**: 2026-10-29 — Phase 86（W4+W5）与 Phase 90（W0~W3）全部收口并验收；下一步 Phase 87
**Prior activity (2)**: 2026-10-28 — v1.24 全量执行完成（Phase 77 视觉验证 + Phase 83 拆分收口 + VIS-03 死规则修复）
**Prior activity (2)**: 2026-10-28 — v1.24 执行期：78 权限双端闭环（后端严格中间件 + 前端三态门控）+ 79 PG 实机验收（三层证据）+ 82 lint 清零
**Prior activity (3)**: 2026-10-28 — v1.23 SHIPPED & ARCHIVED（Phases 69-75, 22/22 REQ satisfied）
**Prior activity (4)**: 2026-10-27 — v1.23 roadmap CREATED（Phases 69-75, 22/22 REQ mapped）

## Current Milestone: v1.53 开放项全量清偿 (Phases 134-141) — ✅ SHIPPED & ARCHIVED 2026-11-08

**Goal:** 以 `.planning/REQUIREMENTS.md` 的 **24 REQ** 为基线（对应 2026-11-08 开放项盘点的 21 条历史计数），按「代码可修 / 待人工验证 / 已裁定接受」三类分别闭环，使收档时每条开放项都有可核对的结论与证据。

**立项动因（2026-11-08）**: v1.52 SHIPPED & 归档后 `Awaiting next milestone`；用户指示「查看 open-item 文档，建立新里程碑处理所有事项」。2026-11-08 开放项盘点明确「凡未列入本盘点清单的历史 TODO/FUTURE/待办/deferred/Known Gaps 字样一律视为文档债或历史归档」，故本里程碑严格以该盘点 21 条（A 节 8 + B 节 13）为范围，C 节 13 类防误读清单不纳入。

**范围裁定（用户 2026-11-08）**:
- **21 条全覆盖**，按三类分别处置：代码可修（~16 条）→ FIX REQ 写代码；待人工验证（~6 条）→ VAL REQ 交付清单重建 + 跟踪表 + 回写口径，**不伪造闭环**（受生产 PG 只读 / 写操作授权 / 现场真机 / DBA 回传约束，AI 不擅自写生产数据，沿用 v1.25 Phase 90 先例）；已裁定接受 / 仅登记（2 条）→ ACK REQ **只落结论、不写修复代码**
- **Phase 从 134 续编**（v1.52 结束于 133），8 phases 134-141 / 24 REQ（FIX 14 / VAL 7 / ACK 2 / GOV 1），100% 映射无孤儿无重复
- **跳过领域研究**（21 条均带本仓 `文件:行号` 实证、无新领域能力需调研，沿用 v1.25「研究范围仅定向」先例）
- **纳入 seed** `seeds/SEED-ad-push-enable-direction.md` 作为 FIX-01 输入上下文（另 2 个 seed 已标 SUPERSEDED，跳过）
- **权威回收（用户 2026-11-08 追加裁定，覆盖本文件所有「临时清单 = 权威 / 唯一入口」表述）**：临时输入清单仅用于本次里程碑取数、非长期权威；21 条全部处置干净后须归档（移入 `.planning/milestones/` 或删除）、不再维护 A/B/C 节，通过 GSD SDK `milestone complete` / `audit-open` 完成归档并清理引用，**权威交回 GSD 自动生成的 ROADMAP / REQUIREMENTS / STATE / MILESTONES**。GOV-01 / Phase 141 改为「临时清单归档 + 权威交回 + 一致性复核」，不再执行「清偿项移入 C 节」；VAL-* 结果写入 GSD 产物，不回写临时清单。

**事实基线**:
- `.planning/REQUIREMENTS.md`（**24 REQ**）｜`.planning/ROADMAP.md`（Phases 134-141）
- 状态权威 = STATE.md / ROADMAP.md / MILESTONES.md（GSD 自动生成体系）
- 人工验证类 5 份原始 UAT 清单磁盘缺失（`48/54/56/57-HUMAN-UAT.md` + 133-UAT 跟踪），Phase 140 须重建

**硬约束（v1.53）**:
1. 遵循 Handler-Service / `response.Success()/Error()` / operlog 约定 / 0=正常 1=停用 / SQLite+PG 双方言（uuid 双侧 CAST）
2. 回归基线：eslint 0 problems / vitest 全绿 / `go build ./...` + `go test ./...` 全绿
3. 人工验证类不写生产数据、不擅自切换生产认证模式；LOCKED 项不写修复代码
4. 结果写入 GSD 自动生成的产物（v1.53-REQUIREMENTS.md traceability / STATE.md Deferred Items）；GOV-01 统一收口 21/21 条终态结论
5. `phases/...` 证据路径按 `.planning/ARCHIVE-INDEX.md` 换算到 `.planning/milestones/v1.52-phases/`

---

## 历史基线：v1.52 运维域数据权限统一收口（终态）(Phases 130-133) — ✅ SHIPPED & ARCHIVED 2026-11-08（以下为非当前状态，仅供追溯）

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

Phase: Not started (roadmap created)
Plan: —
Status: Defining requirements
Last activity: 2026-11-09 — v1.55 roadmap created

## 历史归档（v1.25 及更早，已 SHIPPED / ARCHIVED；以下为非当前状态，仅供追溯）

> 本节及后续子标题（进度快照、Phase 依赖速查、执行期决策、立项范围、Accumulated Context 等）为 **v1.25 及更早里程碑**的归档记录，**均非当前状态**；当前状态一律以文档上半段 **v1.55 active** 与 `## Current Position` 为准（v1.54 终态见上方「历史基线」段与 `.planning/MILESTONES.md` v1.54 条目，v1.53 及更早见 `.planning/MILESTONES.md`）。

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

- `gsd_run` CLI 在 v1.21 期不可用（`$HOME/.claude/get-shit-done/bin/gsd-tools.cjs` 不接受 `query <verb>`）—— 那是当时状态；**2026-11-08 v1.53 立项实测已可用**（`C:\Users\CPIC\.agents\gsd-core\bin\gsd-tools.cjs` 的 `query init.new-milestone` / `state.milestone-switch` / `state.get` / `init.progress` 均正常，STATE.md frontmatter 由 SDK 切换）。但 `roadmap.analyze` 仍解析不出 `<details>` 内的 `- [ ] Phase N:` 列表（`phases: []`），与 v1.52 归档格式一致，属本仓既有惯例 → 除 frontmatter 用 SDK 外，ROADMAP/REQUIREMENTS/PROJECT 正文继续保持手工编辑（Phase 57-62 先例）。
- 提交落在 main 分支（`branching_strategy: none`）
- DB migration 编号已用到 **235**（migration_235_workstation_backfill_perm.go，v1.51 期）；**v1.52 新迁移从 236 起编**。

### Decisions

- [v1.21 遗留] 57-HUMAN-UAT.md 三项真环境验证：① kill -9 重启耐久已由 `.planning/debug/ad-push-queue-drain-interval-not-hot-reloaded.md:172-174` 独立证实，可核销；② PG 合并路径、③ AD UAC 533 未见闭合证据，仍挂账；原始 57-HUMAN-UAT.md 文件当前磁盘不存在，需重建清单。v1.18/1.19/1.20 的 site-visit UAT 债保留挂账

### Blockers

- 无

### Quick Tasks Completed

> ✅ 4 个 v1.51/v1.52 期 quick 任务目录已于 2026-11-08 由 `/gsd-complete-milestone --archive-quick` 归档至 `.planning/milestones/v1.52-quick/`，表内数据行按 GSD 归档语义清空；逐目录索引见归档目录 `README.md` 与 `.planning/ARCHIVE-INDEX.md`。

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 261109-wdu | 清偿 v1.55 收档挂账 A 类代码缺口：3 项低风险代码（trunk operlog / computeByWorkstation append / 发现导入事务与 Save Error，含 2 组红绿回归断言）+ A-2 两份 e2e 脚本物理合并到 `tests/e2e/` + A-3 就近单测豁免登记；B/C 保持挂账 | 2026-11-09 | `6c21bf47`（代码/规范）+ 本提交（规划） | `.planning/quick/261109-wdu-v1-55-3-trunk-operlog-computebyworkstati/` |

### Todos（v1.52 / Phases 130-133，历史归档）

- **已清偿**：以下为 v1.52 立项期计划，全部已于 2026-11-07 交付（Phases 130-133 四个 phase 完成、三项开工输入裁定已落盘 `.planning/notes/261107-v152-*-decision.md`，PG contract 变量名统一 `TEST_PG_DSN` 与新迁移 236 起编亦已完成），见 MILESTONE-v1.52-SUMMARY.md。**v1.53 当前待办**以 `.planning/REQUIREMENTS.md`（24 REQ）为准，见 `.planning/ROADMAP.md` Phases 134-141。

### Roadmap Evolution

- Phases 134-141 created for v1.53「开放项全量清偿」: 134 AD 同步域（FIX-01/10/11）∥ 135 Swagger 链路（FIX-02）∥ 136 端点接线（FIX-03/06/13）∥ 137 Stub/双副本（FIX-07/08）∥ 138 功能分支（FIX-04/05/14）∥ 139 低优先级技术债（FIX-09/12）→ 140 人工验证债组织（VAL-01~07）→ 141 登记接受结论 + 临时清单归档与权威分层复核（ACK-01/02 + GOV-01，**必须最后执行**）。**24/24 需求映射，无孤儿无重复；21/21 条真实开放项全覆盖**（A 节 8 + B 节 13；C 节防误读清单不纳入）；跳过领域研究；纳入 SEED-ad-push-enable-direction。W1 六 phase 按模块隔离并行（134 与 138 均触及 `internal/scheduler/`，plan-phase 须产出函数级归属表，无法保证无交集则降级串行 134→138）。 (CRITICAL)
- Phases 84-93 created for v1.25「全站权限治理与样式收敛」: 84 权限模型三缺口（PMODEL-01~04）→ 85 后端动作级严格化（RBAC-01/03/04）→ 86 前端操作入口三态门控铺开（RBAC-02）→ 87 动作级权限授予闭环（RBAC-05/06）→ 88 验证底座与零风险清理（VISFIX-03/04/06 + STYLE-02 第 1 批）→ 89 变量层与明暗色板重构（STYLE-01/04 + VISFIX-01/02）→ 90 Input 域交还 variant/token + 圆角统一（STYLE-03 + VISFIX-05）→ 91 Select/InputNumber/Form/Table/Modal 域交还（承接 STYLE-02 R8 + VISFIX-04 剩余）→ 92 暗色 mega-cluster 合并（STYLE-02 主责）→ 93 room-devices 拆分 + 视觉回归验收（SPLIT-01 + STYLE-05）。**22/22 需求映射，无孤儿无重复**；线 A 权限 84-87 ∥ 线 B 样式 88-93，线内串行；写死三处不可逆顺序约束（89→90/91/92、批次 C→批次 B 同 commit、变量层→变体交还）。 (CRITICAL)
- Phases 76-83 created for v1.24「视觉回归修复与遗留债务清偿」: 76 双框 P0 修复 + CSS 审计登记册（VIS-01~04）→ 77 视觉回归验证收口（VIS-05）→ 78 #16 前后端双端闭环（PERM-01~04/06）→ 79 PG 实机验收（PERM-05）→ 80 3f130 空因诊断前端接入（DIAG-01）→ 81 F5 刷新落 /dashboard 修复（NAV-01）→ 82 lint 清零 + 豁免理由表修复（HYG-01/02/03/05）→ 83 devices 页拆分（HYG-04）。18/18 需求映射，无孤儿无重复；**全部 SHIPPED 2026-10-28**（tag `v1.24`）。 (CRITICAL)
- Phases 69-75 created for v1.23「前端契约全量收尾与遗留清偿」: 69 后端 debug 重验 spike（DBG-01）→ 70 TextArea 契约基建 W0（TEXT-01/03/04 + HYG-05）→ 71 TextArea 存量 + COMP-05 分域迁移 W1-W6（TEXT-02 + MIG-01/02/04）→ 72 后端 debug 修复 wave（DBG-02/03/04，#3 先于 #2）→ 73 DB 实测日 + cron.go 双执行调查（DBG-05/06/07）→ 74 COMP-05 收口 + lint 清偿 + quick 目录归档（MIG-03 + HYG-01~04）→ 75 浏览器实机 UAT + 收档（UAT-01/02）。22/22 需求映射，无孤儿无重复。 (CRITICAL)
- Phase 68.1 inserted after Phase 68: v1.22 收尾清偿 phase：危险操作收口 + 契约 minor 补课 + 豁免收缩 + 存量迁移收尾（DEBT-04） (URGENT)

## Project Reference

See: `.planning/PROJECT.md`（updated 2026-11-08 after v1.53 SHIPPED & ARCHIVED）

**Core value:** 端到端的运维可观测与可审计——每个写操作都留下可追溯记录（谁、何时、改了什么、从哪来、改前改后），敏感字段自动脱敏；配合 Excel 批量管理与引用自动解析（部门名 → UUID），运维人员对整个 IT 资产域拥有唯一可信的系统记录。
**Current focus:** 无 · v1.53 SHIPPED & ARCHIVED；权威已交回 GSD 自动生成的 ROADMAP / REQUIREMENTS / STATE / MILESTONES

## Session

**Last session:** 2026-11-08T17:55:00+08:00
**Stopped at:** v1.53 SHIPPED & ARCHIVED（8/8 Phase、21/21 条开放项全覆盖、全量回归全绿、GSD 归档与权威交回完成）；下一步 `/gsd-new-milestone`
**Resume file:** 无（归档产物见 `.planning/milestones/v1.53-*`，人工验证跟踪见 `.planning/notes/261108-human-verification-tracker.md`）

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first:

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| verification_gaps | 84/84-VERIFICATION.md — 真实挂账，详见 129-UAT-BACKFILL.md:91-92 | human_needed（真实挂账） | 2026-10-30 | v1.25 |
| verification_gaps | 87/87-VERIFICATION.md — 真实挂账，详见 129-UAT-BACKFILL.md:91-92；87-6 另需写操作授权 | human_needed（真实挂账） | 2026-10-30 | v1.25 |
| uat_gaps | 129-HUMAN-UAT.md — 收档审计 acknowledge（0 pending scenarios） | acknowledged（deferred） | 2026-11-08 | v1.52 |
| uat_gaps | 129-UAT-BACKFILL.md — 收档审计 acknowledge（0 pending scenarios）；84/87 人工验收挂账载体 | acknowledged（deferred） | 2026-11-08 | v1.52 |
| uat_gaps | 133-UAT.md — 收档审计 acknowledge（0 pending scenarios）；37 项人工 UAT 全未勾选 | acknowledged（deferred） | 2026-11-08 | v1.52 |
| uat_gaps | VAL-01~06 人工验证（生产 PG 只读 / 87-6 写授权 / 现场真机 / DBA 回传）— 交付物已落盘（5 份重建清单 + 79 条跟踪表 + 回写口径），实际验证待执行；见 `notes/261108-human-verification-tracker.md` | 待执行（不标已完成） | 2026-11-08 | v1.53 |
| code_gaps | ~~`ImportDiscoveredDevices` 简化实现~~ **v1.54 Phase 142 已清偿（2026-11-09）**：`internal/services/device_discovery_service.go` 改为读 `sys_device_discovery_result` → 按 `ip_address` 去重（批内/已存在跳过/软删恢复）→ 单事务批量写 `sys_network_device`；handler 补缓存失效；9 个新单测 + 2 处旧 stub 断言改写。证据 `.planning/phases/142-disc-01-02/142-SUMMARY.md` | ✅ 已清偿 | 2026-11-08 → 清偿 2026-11-09 | v1.53 → v1.54 |
| verification_gaps | ~~Phase 自报边界汇总~~ **v1.54 Phase 143/144 已清偿其中 4 项代码缺口（2026-11-09）**：FIX-11 前端告警页 / WS 广播 / 企微钉钉 webhook（路线 A 通用 Webhook）｜FIX-04 型号级 `vendorCapabilities` 规则框架｜FIX-05 批量物理参数路径接入｜`ImportDiscoveredDevices` 见上一行。**仍未清偿（验证类，不伪造闭环）**：FIX-01 实机 AD/LDAP UAT｜FIX-04 三厂商物理参数真机逐型号取证｜FIX-09 端到端 HTTP P99 压测｜FIX-13 生产库 migration 236/237｜FIX-11 企微/钉钉专用 markdown+加签（路线 B） | 代码项 ✅ 已清偿 / 验证类 待执行（不伪造完成） | 2026-11-08 → 部分清偿 2026-11-09 | v1.53 → v1.54 |
| code_gaps | **v1.54 新增登记**：① 前端「标记已读」未实现（告警 `logs` 来自 `sys_oper_log` 无已读字段、后端 5 端点无已读契约）② 企微/钉钉专用 markdown + 加签（路线 B）未做，当前只走路线 A 通用 Webhook ③ `sys_object_id` 已采集未落库，型号 key 仅 `NetworkDevice.Model` ④ 批量物理参数无自动回滚（预览/审计已显式警示）⑤ ~~发现导入链路顺带隐患（`user_id.(string)` 硬断言、`persistDiscoveryResults` 未开事务、`ExecuteDiscovery` 未检查 Save Error）~~ → **2026-11-09 quick 261109-wdu 部分清偿**：`persistDiscoveryResults` 事务化（Delete+Create 全走 tx）+ `ExecuteDiscovery` 两处 `Save` 补 `.Error` 检查 + `device_discovery_service_atomic_test.go` 红绿回归断言；**实证纠偏**：`user_id.(string)` 硬断言当前代码已不存在（全 `internal` 零命中），属过时登记、无代码可修，不伪造闭环 ⑥ ~~e2e 归位（`physical_e2e_test.go` 仍在业务包内）~~ → ✅ **2026-11-09 quick 261109-wdu 已裁定收口**：经实证该文件及 `port_write_e2e_test.go` 依赖同包私有 fixture 与未导出生产类型，硬搬 `tests/e2e/` 会编译失败；按「Go 单测就近放业务包内」在 **AGENTS.md 登记为合规豁免、不迁移**，消除「待归位」悬置表述（真迁移需抽共享 fixture 并单独立项） ⑦ ~~`RequiresInterfaceView` 与型号画像 `RequiresView` 联动点~~ → ✅ **2026-11-10 quick 261110-161 已裁定**：实证两者当前完全脱钩但**所有画像 `RequiresView=true`、三家默认/基线均未覆盖 → 差异不触发、无实际 bug**，属「框架预留、未联动」；按推荐**不改代码**，仅登记为「待真实 `RequiresView=false` 型号触发时再联动」，真统一需改 6 处 + 4 组测试并触碰跨 phase 文件面，单独立项评估 | ⑤⑥⑦ 已收口 / ① B-1 后端最小闭环已做（前端契约对齐另记 v1.56）/ ②③ 立 v1.56 里程碑草案 / ④ 已补诚实性短板（quick 261110-161） | 2026-11-09 → 部分清偿 2026-11-10 | v1.54 |
| code_gaps | **v1.55 W1 新增登记（门禁真跑后暴露）**：① `POST /network/ports/write/trunk`（`internal/api/v1/network/port_write_trunk_handler.go:43-80`）成功路径直接 `response.Success`、前面**无 `operlog.Record`**，且 `internal/services/portwrite/` 全包零 operlog → 违反 CLAUDE.md「所有业务写操作 success path 在 `response.Success` 前必须 `operlog.Record`」强制约定，属 v1.51 Phase 124 遗留真实缺口；Phase 145 按文件面/范围未修，已用 `knownGapHandlers` + 可观测 `t.Skip` 在静态门禁中**登记而非放行**（`internal/utils/operlog/handler_coverage_test.go`、`145-SUMMARY.md §Known Gaps`），**W2 148 未覆盖它（portwrite 不在 148 文件面）** → ✅ **2026-11-09 quick 261109-wdu 已清偿**：补 `operlog.Record(ModulePortWrite, OperTypeUpdate)` + 从 `knownGapHandlers` 摘除（反向断言同步），`HandlerDifferential` 不再 Skip ② ~~`reconciliation_service.go` `computeByWorkstation:1074` 既有 `append` 早于 exception 字段修改~~ → ✅ **同 quick 已清偿**：append 移到 matcher 覆盖之后 + `reconciliation_append_order_test.go` 红绿回归断言；`reconciliation_cache_contract_test.go` 立项期「禁区禁止改」注释同步更正 ③ ~~两份 `operlog_e2e_verify.sh` 重复副本未物理合并~~ → ✅ **2026-11-09 quick 261109-wdu（用户裁定按 AGENTS.md「后端 e2e 归 `tests/e2e/`」）已物理合并**：删 `scripts/operlog_e2e_verify.sh` + `scripts/e2e/operlog_e2e_verify_test.go`（`scripts/e2e/` 移除），唯一权威 = `tests/e2e/operlog_e2e_verify.sh` + driver；`scripts/README.md` / `AGENTS.md:37` / `handler_coverage_test.go` 注释同步；实测 `SKIP_E2E=1` 可观测 skip、`SKIP_LIVE=1` driver+脚本 PASS、`bash -n` OK、`go build ./...` OK | ①②③ ✅ 全部清偿（2026-11-09 quick 261109-wdu） | 2026-11-09 → 清偿 2026-11-09 | v1.55 W1 |
| verification_gaps | **v1.55 W2 新增登记**：① `.golangci.yml` forbidigo（148）实测 golangci-lint 2.12.2 **只匹配 AST 标识符/选择器调用名，不命中字符串字面量或注释里的 SQL 方言 token**；真正的「反向变异必红」由 148 新增的 Go 源码静态断言测试承担，规则本身作为标识符形态准入守卫（探针实测生效）。`148-SUMMARY.md §4` 已如实登记能力边界，不伪造「forbidigo 必红」 ② `TEST_PG_DSN` 双跑仍为前置依赖（用户裁定本期不建 `services: postgres`），148/149/150 的 PG contract 测试均为 `t.Skip` 可观测 ③ 149 的 D1/D2 UAT 作废登记落 `149-UAT-RECHECK.md`、150 的 backfill UAT 落 `150-UAT-BACKFILL.md`，**重跑窗口/验收人待 QA 裁定，未勾选完成** ④ FUTURE-01 随 ① 关闭的 4 处引用清理、132/133 UAT 重跑，均待 **B（Phase 151 统一 ScopedDB）落地后由 Lead 收档统一执行** ⑤ `room_photo` / `location_alias` 两 handler 现为固定状态码形态（service 层已统一 `ErrScopeDenied`、149 T4 矩阵 `errors.Is` 全绿），HTTP 层透传统一属后续候选 | 能力边界 / 前置依赖 / 待人工（登记，非阻断，不伪造闭环） | 2026-11-09 | v1.55 W2 |
| note | **v1.55 W2 契约基线更新（非缺陷）**：`internal/api/datascope_mount_test.go` 的 `TestDataScopeMount_OpsCoverage` / `TestDataScopeMount_NotOnOpsAncestor` 原按 v1.52「roomPhotos 是显式豁免组、挂载总数 31」硬编码；用户 2026-11-09 裁定推翻该豁免后，149 补挂 `roomPhotos.Use(middleware.DataScopePermission(core))`，Lead 集成期按新语义把断言更新为「总数 32、roomPhotos 恰好挂 1 处」，并在注释写明裁定依据（这不是放宽门禁，是把契约测试对齐已批准的新语义） | 基线同步（已落地） | 2026-11-09 | v1.55 W2 |
| code_gaps | ~~6 项 LOCKED 残余（7 子项）~~ **v1.55 全部清偿（2026-11-09）**：① P1-5 侧信道 = A(150 响应体去规模化，删 6 字段/全库 `countQ`/越界行号，差分攻击测试) + B(151 统一 ScopedDB：pkg/query ScopedDB + 资源归属注册表 + `GORMRepository` 读链 2 行 + 8 核心读端点 32 处起链，写链一字不动、All 兜底零回归) ② 5 项豁免面 = 149 方案 D 零迁移收口 ③-A `RightCast` = 148 删方言机制+单侧 CAST ③-B PG CTE = 148 唯一共享 `recallPhysicalRows` Go 层合并 ④ e2e 门禁 = 145 静态段 Go 单测化+cwd 锚定+CI 硬门禁 ⑤ `invalidateCache` = 147 删死键/空函数+广播失效立即一致 ⑥ `pbkdf2SM3` = 146 守卫+`len==32`+黄金向量（CRYPTO-04 按硬 gate 合规跳过，守卫已收口非遗留缺陷）。证据：7 份 phase `*-SUMMARY.md` + `.planning/notes/261109-v154-locked-root-cause-analysis.md`。**裁定原文 `261108-v153-locked-residuals.md` 保持只读，收档结论由本表与 MILESTONES.md v1.55 条目承载** | ✅ 已清偿 | 2026-11-08 裁定 → 清偿 2026-11-09 | v1.52 → v1.55 |
| verification_gaps | **v1.55 收档后仍待人工（不伪造闭环）**：① 79 条人工验证跟踪表保持 0 勾选 ② FIX-01/04/09/13 验证类残余 ③ `TEST_PG_DSN` 双跑前置依赖（本期不建 `services: postgres`）④ 149 的 `149-UAT-RECHECK.md`（D1/D2）与 150 的 `150-UAT-BACKFILL.md` 重跑窗口/验收人待 QA 裁定 ⑤ **W1/W2 新增登记（2026-11-09 quick 261109-wdu 部分清偿）**：~~trunk operlog 真实缺口（无 `operlog.Record`，待后续 quick）~~ ✅ 已补 `operlog.Record`+摘除 `knownGapHandlers`｜~~`computeByWorkstation:1074` append 缺陷~~ ✅ 已移到 matcher 之后 + 红绿回归断言｜**已由该 quick 清偿**：两份 `operlog_e2e_verify.sh` 重复副本（用户裁定收敛到 `tests/e2e/` 唯一权威、物理删除 scripts 侧）｜`physical_e2e_test.go` 归位（AGENTS.md 登记就近单测合规豁免、不迁移）｜发现导入链路 `persistDiscoveryResults` 未开事务 + 两处 `Save` 未检查 Error（`user_id.(string)` 硬断言经实证当前代码已不存在，属过时登记、无代码可修，不伪造闭环）｜**仍挂账**：forbidigo 能力边界、`room_photo`/`location_alias` 两 handler HTTP 透传口径 ⑥ **FUTURE-01 已随 ① 根治关闭**：4 处引用已由 Lead 收档清理（`REQUIREMENTS.md` / `ROADMAP.md:91` / `STATE.md` / `261108-v153-locked-residuals.md:63`），不再是未来候选 | 待人工 / 后续候选（登记，非阻断，不伪造闭环） | 2026-11-09 | v1.55 收档 |

> **closeout_type: override_closeout**（v1.52 收档时 `audit-open` 检出 3 项 uat_gaps，已按 [A] Acknowledge 全部登记；另有 3 项历史 debug suppression 结转）。Known verification overrides: **3 newly acknowledged, 3 carried forward**（同步记录于 MILESTONES.md v1.52 条目）。以上均为 verdict-preserving 的 self-invalidating suppression。**v1.53 收档后补登记（2026-11-08 18:45）**：上表另含 v1.53 的 3 行 `uat_gaps / code_gaps / verification_gaps`（79 条人工验证跟踪 + `ImportDiscoveredDevices` 真实缺口 + Phase 自报边界汇总），属**显式披露**而非 `audit-open` suppression；GSD `audit-open` 仍报全清（这些是文档登记项，不是 GSD 可扫描的 artifact）。真实挂账以本表与 MILESTONES.md v1.53 条目为准。

## Performance Metrics

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| v1.53（8 phase，subagent 团队并行 + Lead 集成收档） | 单日（2026-11-08 14:43→17:55） | 8 phase / 24 REQ / 21 条开放项 | 全仓（见最终提交 diff） |
| v1.25（10 phase，planning 阶段） | 路线图 1 轮完成（2026-10-28） | 10 phase × 5 成功标准 | — |

## Operator Next Steps

- `/gsd-new-milestone` — 启动下一里程碑（v1.55 已 SHIPPED，7/7 Phase、24/24 REQ、6 项 LOCKED 残余全部从根源清偿；权威仍由 GSD 自动生成体系承载，`REQUIREMENTS.md` 已按工作流清空待重建）
- 人工验证跟踪表 `.planning/notes/261108-human-verification-tracker.md` 的 **79 条待执行项保持 0 勾选**，需生产 PG 只读 / 写授权 / 现场真机 / DBA 回传后按 `notes/261108-uat-writeback-policy.md` 回写 GSD 产物；FIX-01/04/09/13 验证类残余**不伪造闭环**
- LOCKED 残余裁定原文 `.planning/notes/261108-v153-locked-residuals.md` 保持只读（v1.55 收档已更正其 :61 承载声明并关闭 FUTURE-01）；清偿结论见本表 v1.55 行与 `MILESTONES.md` v1.55 条目
- **下一里程碑取数**：本表 `## Deferred Items` 的「v1.55 收档后仍待人工」行（trunk operlog 缺口 / `computeByWorkstation` append / 重复脚本合并 / forbidigo 能力边界 / 两 handler 透传 / PG 双跑前置 / UAT 重跑窗口）与 `MILESTONES.md` v1.55 Known Gaps
- v1.55 交付证据：7 份 `.planning/phases/{145..151}-*/*-SUMMARY.md` + `149-UAT-RECHECK.md` + `150-UAT-BACKFILL.md`；根因判定 `.planning/notes/261109-v154-locked-root-cause-analysis.md`；草案 `.planning/notes/261109-v155-requirements-draft.md`
- v1.54 交付证据：`.planning/phases/{142-disc-01-02/142-SUMMARY.md, 143-port-01-02/143-SUMMARY.md, 144-alrt-01-02-03/144-01-backend-SUMMARY.md, 144-alrt-01-02-03/144-02-frontend-SUMMARY.md}` + 架构裁定 `.planning/notes/261108-v154-architecture-decisions.md` + 集成清单 `.planning/notes/261108-v154-integration-checklist.md`
