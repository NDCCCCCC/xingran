---
last_updated: 2026-11-09
update_trigger: v1.55 milestone SHIPPED 2026-11-09（LOCKED 残余根治：7/7 Phase、23/23 REQ、6 项 LOCKED / 7 子项全部从根源清偿；① 的 A+B 均完成 → P1-5 侧信道根治达成；全量回归闸门全绿；验证类残余不伪造闭环）
previous_update: 2026-11-09 v1.55 milestone INITIATED（LOCKED 残余根治：6 项 / 7 子项，用户裁定推翻 ① 产品语义纳入 A+B 根治；取数自 261109-v154-locked-root-cause-analysis.md + 261109-v155-requirements-draft.md；Phase 从 145 续编）
---

## Current Milestone: v1.55 LOCKED 残余根治 (Locked Residual Root-Cause Fix) — ✅ SHIPPED 2026-11-09

**Goal:** 把 v1.52 收档时以「已接受 / 仅登记」名义挂账的 6 项 LOCKED 残余（7 个子项，③ 拆 A/B）从根源上修掉（而非打补丁 / 加告警 / 加方言分支），使约束从「靠约定」升级为「靠机制」；按用户 2026-11-09 裁定允许推翻 ① 的「范围快照回显」产品语义，① 纳入 A+B 组合根治。

**Status:** ✅ SHIPPED 2026-11-09（7/7 Phase：145 AUDIT / 146 CRYPTO / 147 CACHE / 148 DIALECT / 149 SCOPE / 150 侧信道 A / 151 统一 ScopedDB；23/23 REQ Satisfied；3 波共 7 planner + 2 checker + 7 executor + Lead 集成）| **① 的 A(150)+B(151) 均完成 → P1-5 侧信道根治达成；6 项 LOCKED / 7 子项全部清偿** | 全量回归闸门全绿：`go build ./...` / `go test ./...` / `tsc` / `lint:ci ALL PASS` / 全量 vitest / `check-route-registry` 713 零漂移 | 交付证据 7 份 phase `*-SUMMARY.md` + `149-UAT-RECHECK.md` + `150-UAT-BACKFILL.md` + `.planning/MILESTONES.md` v1.55 条目

**Target features:**
- **AUDIT**（④）— operlog e2e 静态段 Go 单测化 + cwd 锚定 + CI 硬门禁（当前静态检查从未真正跑过）
- **CRYPTO**（⑥）— `pbkdf2SM3` 非法输入守卫 + `len(hashBytes)==32` 校验 + 黄金向量（免生产盘点）
- **CACHE**（⑤）— 删死键/空函数 + `GetByWorkstation` 例外变更立即一致联动失效
- **DIALECT**（③）— 删 `RightCast` 方言机制 + PG 专属 CTE 改 Go 层合并 + forbidigo 防复发；CI PG 双跑登记前置依赖
- **SCOPE**（②）— 5 项豁免面零迁移收口（补挂 rooms/photos 中间件 + 保存侧裁剪）+ UAT 重跑
- **SIDECHANNEL**（①，A）— 响应体去规模化（`ScopeTotal`/`ScopeOutOfScope`/行号）+ 去掉全库 `countQ` + 差分攻击测试 + 132/133 UAT 重跑
- **ARCH**（① 的 B，必需）— 统一 ScopedDB（GORM `Scopes` 强制唯一查询起链点 + 资源归属注册表 + 响应契约层），必须最后执行

**关键决策 (v1.55, 用户 2026-11-09 裁定):**
- **范围 = 6 项 LOCKED / 7 子项全覆盖**（① ② ③-A ③-B ④ ⑤ ⑥）；**不允许**用告警通道 / 粗粒度桶 / 方言分支 / 「接 Redis 后补 Delete」/ 仅调门禁顺序等补丁方式代替根治
- **DECISION-①（决定性）：允许重定语义** —— 取消 `BackfillDeptResult` 规模回显、去掉全库 `countQ`、去导入 `Errors[].Row` 行号、作废 132-SUMMARY.md:74 前端四类「本次范围：共 N 条」文案；① 按 A（响应体去规模化）+ B（统一 ScopedDB 防回归）组合根治
- **FUTURE-01 随 ① 根治一并关闭**（✅ v1.55 收档已关闭，4 处引用已由 Lead 清理），不再单列挂账
- **CI-PG**：本期不建 `services: postgres`，`TEST_PG_DSN` 双跑登记为前置依赖，不伪造双跑
- **存量密码**：全库写入均经 `HashPassword`（恒 32 字节），免盘点直接上守卫
- **例外缓存陈旧窗口**：要求立即一致，`CACHE-02` 必做联动失效
- **跳过领域研究**：7 子项均带本仓 `文件:行号` 实证与现成改写范本，根因报告即研究输入（沿用 v1.53/v1.54 先例）
- **79 条人工验证跟踪表保持 0 勾选**；FIX-01/04/09/13 验证类残余、v1.54 新增 7 项 GAP 不纳入；不伪造闭环
- **零新依赖**；遵循 Handler-Service / `response.Success()/Error()` / operlog 写操作约定 / 0=正常 1=停用 / UUID 主键 / SQLite+PG 双方言 / 前端 wrapped API / 构建产物只落 `out/`
- **提交授权**：用户 2026-11-09 沿用 2026-11-08 口径，授权 GSD 自动提交 v1.55 立项/需求/路线图产物与实现代码

**事实基线:** `.planning/notes/261109-v154-locked-root-cause-analysis.md`｜`.planning/notes/261109-v155-requirements-draft.md`（决策裁定记录）｜`.planning/notes/261108-v153-locked-residuals.md`（6 项裁定原文）｜`.planning/STATE.md ## Deferred Items`

---

## Past Milestone: v1.54 真实代码缺口清偿 (Code Gap Clearance) — ✅ SHIPPED 2026-11-09

**Goal:** 清偿 `.planning/STATE.md ## Deferred Items` 中登记的 4 项真实代码缺口，使 v1.53 收档后显式披露的「代码可修」残余全部真正落地；不触碰 LOCKED 残余、不伪造人工验证闭环。

**Status:** ✅ SHIPPED 2026-11-09（3/3 Phase：142 发现设备真实导入 / 143 端口写物理参数收口 / 144 同步漂移告警通道收口；7/7 REQ Satisfied；144 拆后端+前端两个并行 executor，共 4 PLAN；W1 四 executor 并行 + Lead 集成）| 全量回归闸门全绿：`go build ./...` / `go test ./...` / `tsc` / `lint:ci ALL PASS`（含 check-permission-codes 双向零漂移、check-route-registry 713 路由）/ 全量 vitest | 交付证据 4 份 phase `*-SUMMARY.md` + `.planning/MILESTONES.md` v1.54 条目

**Target features:**
- **DISC** — `ImportDiscoveredDevices`（`internal/services/device_discovery_service.go:509-523`）真实写入 `NetworkDevice`：字段映射、去重语义、事务边界、operlog、缓存失效、错误返回
- **ALRT** — 同步漂移告警通道收口（FIX-11 自报边界）：后端查询/配置端点 + WS 广播 + 前端告警页 + 企微/钉钉 webhook（凭证脱敏）
- **PORT** — 端口写物理参数收口（FIX-04/FIX-05 自报边界）：批量物理参数路径接入（复用 v1.51 批量进度事件）+ 型号级 `vendorCapabilities` 规则（型号来源与默认值回退）

**关键决策 (v1.54, 用户 2026-11-08 裁定):**
- **范围 = 4 项纯代码缺口**：`ImportDiscoveredDevices` 真实导入 / 告警前端+WS+webhook / 批量物理参数 / 型号级能力矩阵；3 项验证类（FIX-01 实机 AD UAT、FIX-09 端到端 HTTP P99 压测、FIX-13 生产库 migration 236/237）与 FIX-04 真机逐型号取证**不纳入**，继续登记为待人工验证、不伪造闭环
- **跳过领域研究**：4 项均带本仓 `文件:行号` 实证、无新领域能力需调研（沿用 v1.53 先例）；改为派 3 个并行 subagent 做本仓代码面探索作为需求输入
- **LOCKED 残余不写修复代码**：`.planning/notes/261108-v153-locked-residuals.md` 6 项只维持结论
- **79 条人工验证跟踪表不改**：`.planning/notes/261108-human-verification-tracker.md` 保持 0 勾选，按 `.planning/notes/261108-uat-writeback-policy.md` 由人工执行后回写
- **遵循既有架构**：Handler-Service / `response.Success()/Error()` / operlog 写操作约定 / 0=正常 1=停用 / UUID 主键 / SQLite+PG 双方言 / 前端 wrapped API / 构建产物只落 `out/`
- **智能体团队**：3 个探索 subagent（发现导入 / 告警通道 / 端口写）→ gsd-roadmapper 生成路线图 → 按 phase 分波并行 executor subagent → Lead 集成复核 + 全量回归闸门
- **提交授权**：用户 2026-11-08 明确授权 GSD 自动提交规划产物与实现代码（覆盖 CLAUDE.md 默认确认要求，仅限本里程碑）

**事实基线:** `.planning/STATE.md ## Deferred Items`（v1.53 三行）｜`.planning/MILESTONES.md:19-24`（v1.53 Known Gaps）｜`.planning/notes/261108-v153-locked-residuals.md`（LOCKED 边界）

---

## Past Milestone: v1.53 开放项全量清偿 (Open Items Clearance) — ✅ SHIPPED & ARCHIVED 2026-11-08

**Goal:** 以 `.planning/REQUIREMENTS.md` 的 **24 REQ** 为基线（对应 2026-11-08 开放项盘点的 21 条历史计数），按「代码可修 / 待人工验证 / 已裁定接受」三类分别闭环，使收档时每条开放项都有可核对的结论与证据。

**Status:** ✅ SHIPPED & ARCHIVED 2026-11-08（8/8 Phase：134-141 全部完成；18 REQ 交付 + 6 项人工验证跟踪不伪造闭环；21/21 条开放项全覆盖）| 归档产物 `.planning/milestones/v1.53-ROADMAP.md` / `v1.53-REQUIREMENTS.md` / `v1.53-phases/`；全量回归闸门全绿（go build / go test 全仓 / tsc / lint:ci / vitest 592 文件 4128 用例）

**Target features:**
- **FIX（14 REQ）** — 代码可修的缺口与缺陷：A1/A6/A7 AD 同步域（含纳入 seed `SEED-ad-push-enable-direction`）、A3-OQ7、B1~B12 中可修项（Swagger 产物链路 / 用户导出 / 工单评价 / GetDiscoveryResults stub / rpa-worker 双副本 / scheduler Rotation / 端口写物理参数 / 3 处排序 / 对账 MV 超时配置化 / 登录加密 benchmark）
- **VAL（7 REQ）** — 待人工验证：重建 5 份磁盘缺失的原始 UAT 清单 + 7 组验证跟踪表 + 回写口径；受生产 PG 只读 / 写操作授权 / 现场真机 / DBA 回传约束，**不伪造闭环、不擅自写生产数据**
- **ACK（2 REQ）** — A8 四项 LOCKED 残余 + B13 仅登记项：只落「接受/已知约束」结论，**不写修复代码**
- **GOV（1 REQ）** — 临时清单归档 + 权威交回 + 一致性复核（结论写入 GSD 自动生成的产物）

**关键决策 (v1.53, 用户 2026-11-08 裁定，均已落地):**
- **21 条全覆盖**（A 节 8 + B 节 13）；C 节 13 类「明确不属于开放项」防误读清单不纳入
- **Phase 从 134 续编**（v1.52 结束于 133），8 phases / 24 REQ / 100% 映射无孤儿无重复
- **跳过领域研究**（21 条均带本仓 `文件:行号` 实证、无新领域能力需调研，沿用 v1.25「研究范围仅定向」先例）
- **纳入 seed** `seeds/SEED-ad-push-enable-direction.md` 为 FIX-01 输入上下文（另 2 个 seed 已标 SUPERSEDED）
- **人工验证类交付物 = 清单 + 跟踪表 + 回写口径**，实际执行由现场/PG 授权方完成；沿用 v1.25 Phase 90「不擅自切换生产认证模式」先例
- **零新依赖**；遵循 Handler-Service / response 包装 / operlog 约定 / 双方言（SQLite+PG）
- **文档提交前须用户确认**（CLAUDE.md「Before making git commits, ask for explicit user confirmation」）
- **权威交回 GSD（用户 2026-11-08 追加裁定，已落地）**：本次临时输入清单已归档进 `.planning/milestones/`（历史快照），全仓「权威 / 唯一入口」引用已清理，GSD `milestone complete v1.53` 已自动生成 / 更新 ROADMAP / REQUIREMENTS / STATE / MILESTONES 承载当前状态；不再维护手工 A/B/C 节清单。

**事实基线（归档后）:** `.planning/milestones/v1.53-REQUIREMENTS.md`（24 REQ）｜`.planning/milestones/v1.53-ROADMAP.md`（Phases 134-141 完整详情）｜`.planning/milestones/v1.53-phases/`（8 份 SUMMARY）｜v1.52 终态 `.planning/MILESTONE-v1.52-SUMMARY.md`

**Known Gaps（登记，非阻断）**：79 条人工验证跟踪项全部待执行（`.planning/notes/261108-human-verification-tracker.md`，**保持 0 勾选**），需生产 PG 只读 / 写授权 / 现场真机 / DBA 回传后按 `notes/261108-uat-writeback-policy.md` 回写 GSD 产物；LOCKED 残余结论见 `.planning/notes/261108-v153-locked-residuals.md`；验证类残余（FIX-01/04/09/13）不伪造闭环；v1.54 新增边界（标记已读 / 企微钉钉路线 B / `sys_object_id` 未落库 / 批量无自动回滚 / 发现导入顺带隐患 / e2e 归位 / `RequiresInterfaceView` 联动点）见 `.planning/MILESTONES.md` v1.54 条目与 `STATE.md ## Deferred Items` 的 v1.54 行。

**Phase 映射:** 134=AD 同步域(FIX-01/10/11) ∥ 135=Swagger 链路(FIX-02) ∥ 136=端点接线(FIX-03/06/13) ∥ 137=Stub/双副本(FIX-07/08) ∥ 138=功能分支(FIX-04/05/14) ∥ 139=低优先级技术债(FIX-09/12)（W1 六者按模块隔离并行）→ 140=人工验证债组织(VAL-01~07) → 141=登记接受结论 + 临时清单归档与权威分层复核(ACK-01/02 + GOV-01，**必须最后执行**)。

---

## Past Milestone: v1.52 运维域数据权限统一收口（终态）(Operations-Wide Data-Scope Unification) — ✅ SHIPPED 2026-11-07（历史归档）

**Goal:** 把数据权限（data_scope）从「用户/VDI 两个模块的局部实现」升级为运维域全模块的声明式统一收口——读写两端按登录人数据范围自动过滤与校验，使「中支管理员只能管本机构的工位/楼层/资产」从口头约定变为架构保证。

**Status:** ✅ SHIPPED 2026-11-07（4/4 Phase：130/131/132/133 全部完成；7/7 REQ：DS-01~07 全部交付；PR #75 merged → `b56f1286`）| v1.52 收档时点快照见 `.planning/MILESTONE-v1.52-SUMMARY.md`

**范围裁定（用户 2026-11-07）:** 终态方案 = operations 全模块声明式统一收口，不做逐端点手工接线（quick-261106-wsb 审计裁定补丁为非终态）。

**Phase 映射:** 130=DS-01/02（ScopeProvider 基建 + 注入覆盖全域，Self 档语义裁定为开工输入）→ 131=DS-03/06 部分（读收口，含 tree/GetByID/缓存 key 断言）∥ 132=DS-04（写收口，Dept/Self 批量 + Excel 导入豁免裁定为开工输入）→ 133=DS-05/06/07（扩展实体 + vdi 合并迁移 + 按已裁定结论落地 + UAT）。

---

## Past Milestone: v1.51 端口写生产级操作体验与平台质量收口 (Production-Grade Port-Write UX & Platform Quality Closeout) — ✅ SHIPPED 2026-11-02（历史归档）

**Goal:** 让端口写从「能写」进化为「敢写、可审计、可观测」——dry-run 预览、批量实时进度、快照回滚、并发互斥、VLAN/端口安全能力扩展、Cisco 第 4 厂商，同步清偿平台技术债（覆盖率回 77.5、审计治理、RPA Token、目录改名）与收尾杂项。

**Status:** INITIATED 2026-10-31 | **Phases:** 9 (121-129，编号延续 GitHub v1.33 线) | **Requirements:** 21（PORTW×6 / PWX×6 / DEBT×6 / TIDY×3）

**范围裁定（用户 2026-10-31）:** 执行候选 A（网络设备端口写深化）+ B（平台技术债清偿）+ D（收尾杂项）；C 组（RPA 拖拽编辑器/插件录制）defer 至 v1.52+；对账 MV 超时根治继续等 DBA 诊断。

**关键外部依赖:**
- Phase 123/124/125/126 的命令验证与 Phase 129 的 UAT 回补需要**真机**（华为 S8700 / 锐捷 RS8607E / H3C，+ Cisco 若到位）
- DEBT-02 审计索引操作千万行级表，需 DBA 窗口

**锁定决策:**
- D-01 phase 编号从 121 续编（延续 GitHub v1.33 线），本地 84-93 与远端 114-120 两线编号就此合流
- D-02 dry-run（121）先行，是回滚/互斥等安全机制的基座
- D-03 波次：W1 零依赖（121→122/128）→ W2 需设备（123→124/125）→ W3（126→127）→ W4 收尾（129）
- D-04 全程七 gate 不倒退（go build / go test / 后端 coverage ≥76.5→目标 77.5 / 前端 floors / lint / type-check / diff coverage）

---

## Past Milestone: v1.25 全站权限治理与样式收敛 (Site-wide Permission Governance & antd 6 Style Convergence) — ✅ SHIPPED 2026-10-30（历史归档）

**交付终态（2026-10-30 收档）:** 全部 10 个 Phase（84~93）全量执行完毕并通过全系统双端终验，22/22 需求 Satisfied，99 个提交原子入库。
- **线 A（权限线）**：`data_scope` 本部门隔离生效；133 处写路由 ANY-of 拆分，`RequireStrictPermissions` 严格单点放行；全站 76 页面操作级三态门控全覆盖（恒等式 25+51=76）；72 个 F 型按钮权限码全播种，白名单彻底归零，授权链路闭环。
- **线 B（样式收敛线）**：453 条暗色 mega-cluster 手写规则彻底清零，手写覆盖交还 antd token；**!important 净降 983 处（降至 452 处 ≤ 455 预算），Hex 降至 26 处 ≤ 60 预算，`index.css` 行数降至 3194 行 ≤ 3200 预算**；`room-devices/index.tsx` 单体大文件由 896 行成功解耦拆分为 342 行（≤350行，子文件均≤250行，零行为变化）；C1~C16c 样式静态契约 ALL PASS；全站 14 页核心业务界面视觉巡检零异常断层。
- **全仓质量**：前端 Vitest 579 文件 4025 用例全绿，`lint:ci` 0 problems，生产打包 `vendor-react` gzip 650.74 kB ≤ 776 kB；后端 `go build` 0 错误，测试全过。

**Goal:** 把 v1.24 只在 `/system/users` 单域完成的「操作级 RBAC 前后端双端闭环」推广到全站 76 页面 / 134 处后端路由挂载，并把样式层从手写 `!important` 规则收敛到 antd 6 官方 variant 机制，同步清偿权限模型三缺口与 Phase 77 视觉新发现。

**问题来源（v1.24 收档 Known Gaps 五项，用户裁定全部立项）:**
- **① 操作级 RBAC 全站治理**：v1.24 只修 `/system/users` 一个域；全站 134 处 `RequirePermissions` 挂载仍是 ANY-of 组级语义 + `permission.go` 模块级降级 → 只读角色在全域都可能被放行写操作；前端仅 14/76 页面接入 `useMenuStore.permissions`（其余恒显示操作入口）；`pkg/permission/config.go:223-289` 动作级映射表已定义但从未接线；`system` 域其余 4 组（roles/menus/depts/posts）仍是 ANY-of
- **② 样式层收敛到 antd 6 variant**（方案 C）：手写 `!important` 规则 1359 处 / 219 行 antd 选择器 / 60 个暗色重复分区；圆角 CSS 6px 与 token 8px 漂移；暗色炸白底的**真实成因是两层**（见 STYLE-01）；`!important` 的真因是「`border` 简写压掉长属性」+「antd 复合选择器达 (0,3,0)」，**不是** hash 加特异性
- **③ 权限模型三缺口**：`data_scope` 只塞 context 不被列表查询消费（实测 `data_scope=3` 仍返回跨部门 4231 条）；`GetUserPermissions` 不过滤 `menu.status`（已停用菜单的权限码仍进前端权限集）；Step 2「子菜单继承」语义易写反（真实是「用户持 F 型按钮 → 继承父页面权限」）＋附带缺口（生产库无迁移记录表 / GORM 模型与库表 schema 漂移）
- **④ Phase 77 视觉新发现**：暗色 `--input-bg` 第 3 处就地重定义（`index.css:4807`，实测输入框底 rgb(250,250,249) 压在深绿底 rgb(15,46,27) 上对比度极低）；暗色 Select 白底 + 浅灰边（两套边框体系并存）；`.dept-tree-search` 补丁是死规则（实测命中 0）；antd 6 真实弃用清单（`maskClosable→mask.closable`、`bordered→variant`、`dropdownRender→popupRender` 等 9 类，**`onSearch` 未弃用**——v1.24 口径有误，见下）；InputNumber 错误态仍灰（成因是项目 `border` 简写压掉 antd `border-color` 长属性 + 只给 Input 域补 status 规则漏补 InputNumber）
- **⑤ `room-devices/index.tsx` 815 行**：比已拆分的 devices 页（714→306）更长，v1.24 明确登记不扩面

**Target features:**
- **RBAC（核心）**：后端写操作路由组逐动作拆严格权限（`RequireStrictPermissions`）+ 前端 76 页面操作入口三态门控铺开（`useActionPermission`）+ `pkg/permission/config.go` 动作映射死表接线 + 动作级 F 型菜单播种
- **STYLE**：先重构 `AntdThemeBridge` 明暗色板（F3，解冻 v1.22 冻结契约）+ 删/降级 4 批手写规则（≈-120 行）+ 统一圆角 6→8 + hex 治理覆盖 `.css`（现 selector 只匹配 JS/TS 节点）
- **PMODEL**：`data_scope` 被列表查询消费 + `GetUserPermissions` 过滤 `menu.status` + Step 2 语义纠偏与文档化 + 迁移记录表补齐 / GORM schema 漂移登记
- **VISFIX**：④ 五项视觉新发现逐项清偿（暗色输入框底 / 暗色 Select 两套边框体系 / 死规则清理 / antd 6 弃用警告 / InputNumber 错误态）
- **SPLIT**：`room-devices/index.tsx` 815 行按 v1.24 devices 页范式拆分（列定义工厂 + 搜索区构造函数 + handlers/行操作 + 页面 JSX 外壳）

**关键决策 (v1.25 init):**
- **执行顺序 ①权限 → ②样式**（用户裁定）——避免样式波动与权限改造混在同一验证周期；权限先行可复用 v1.24 已建的三态门控基建
- **研究范围仅 ② 定向**（用户裁定）——①③⑤ 靠本仓事实与既有实证 note，仅对 antd 6 官方 variant API 与暗色 token 派生规则做定向核验
- **复用 v1.24 基建不重造**：`RequireStrictPermissions`（只做 Step1 精确匹配，跳过 F 型继承与模块级降级）/ `useActionPermission` 三态门控（ready 含→可用 / ready 不含→不渲染 / loading·error→disabled+提示）/ `src/constants/permissionCodes.ts` 单一真相源 + `scripts/check-permission-codes.mjs` 双向校验
- **②须显式解冻 v1.22 冻结契约**——`AntdThemeBridge` 色值真相源变动属冻结面突破，须单独登记与验收
- **零新依赖**——沿用 v1.23/v1.24 原则；不引 stylelint / Playwright / jscodeshift
- **回归基线**：eslint 0 problems / vitest 569 文件 3960 用例全绿 / vendor-react gzip 776 kB / `go build ./...` + `go test` 全绿

**事实基线:** `.planning/milestones/v1.24-REQUIREMENTS.md`（Future Requirements 段五项候选原文）+ `.planning/notes/v1.24-css-audit-register.md`（36 条 CSS 登记）+ `.planning/notes/2026-10-28-pg-permission-baseline.md`（#16 定责）+ `.planning/notes/v1.24-visual-verification.md`（21 形态 × 明暗矩阵）

## Past Milestone: v1.24 视觉回归修复与遗留债务清偿 (Visual Regression Fix & Legacy Debt Clearance) — ✅ SHIPPED 2026-10-28（历史归档）

**交付终态（2026-10-28 收档）:** 18/18 需求 Satisfied；用户主诉「输入框双重外框」根治（309 个输入框全站，双框元素数 3→0，错误态恢复品牌红）；#16 行操作权限**前后端双端闭环**（只读角色调写接口从 200 真删 → 403 拒绝，三层证据：DB 层前后对照 + HTTP 实测含决定性对照实验 + 前端组件测试）；lint 51 → **0 problems**；devices 页 714 → 306 行；生产库验收夹具已按用户指示清理、既有行零改动。Known Gaps 转 v1.25 候选（用户裁定暂不立项）：操作级 RBAC 全站治理 / 样式层收敛到 antd 6 variant / 权限模型三缺口 / 视觉新发现 / room-devices 拆分。

> ⚠️ **机制更正（2026-10-28 v1.25 定向研究期，antd 6.6.4 源码实证）**：本段及 v1.24 REQUIREMENTS 中的两条**机制归因不成立**（**症状与修法均正确，仅机制需更正**）——
> ① 「antd 6 的 cssinjs hash 前缀使 antd 规则特异性 +1 类 → 写在后面就能覆盖不成立」→ antd 6 默认 `hashPriority:'low'`（`@ant-design/cssinjs/es/StyleContext.js:47`）生成 `:where(.css-xxx)`，**hash 特异性为 0**（`util/index.js:145`），同特异性下**后写者胜**；`!important` 的真因是「`border` 简写压掉长属性」+「antd 复合选择器达 (0,3,0)」。
> ② 「`darkAlgorithm` 不改写显式 token」→ `genColorMapToken` 展开在 `...token` **之后**（`antd/es/theme/themes/default/index.js:30-45`），**Map Token 覆盖显式值**（`colorBgContainer`/`colorBorder` 显式设**本就无效**）；暗色白底的真凶是**组件级 token 在算法之后 merge**（`getComponentToken.js:28`）+ **seed 层 `colorBgBase` 为浅奶油**。真实文件路径为 `src/design-system/components/AntdThemeBridge.tsx`（不存在 `src/components/theme/`）。
> ③ v1.24 登记的「antd 6 弃用警告 `onSearch→showSearch`」**不成立**——antd 6.6.4 源码 `deprecatedProps` 与类型定义 `@deprecated` 清单双向确认 `onSearch` **未弃用**。
> 详见 `.planning/REQUIREMENTS.md`（v1.25）STYLE-01 前置障碍段与关键决策表。

**Goal:** 修复 v1.23 引入的输入框「双重外框」视觉回归并做 CSS 一致性审计，同时清偿 v1.23 收档结转的 4 项遗留债务（#16 行操作权限 / 3f130 空因展示 / F5 跳 /dashboard / ColumnConfigModal 豁免注释）与 2 项延后技术债（lint 存量 errors / react-refresh + devices 拆分）。

**问题来源（用户实机截图 + v1.23 收档 Known Gaps）:**
- **用户实机反馈（2026-10-28）**：「文本输入框的一键删除已经有了，但是有的输入框出现这种双重外框的情况」——外层浅灰圆角框内又套一层白色圆角输入框
- **根因（已实证）**：`ruoyi-react-frontend/src/index.css:871` 的 `.ant-input { border: 1px solid var(--input-border) !important }` 命中了 antd 6 affix-wrapper 的**内层** input，antd 原生要求内层 `border: none`（边框只画在 wrapper 上）→ **双框**。v1.23 把 `allowClear` 默认化后，几乎所有 Input 都渲染为 affix-wrapper → 双框从少数变普遍
- **#16 行操作权限**：v1.23 UAT 实测「受限用户仍见 9 行编辑/删除按钮」。**已定责**：`src/pages/system/user/index.tsx:279-301` 操作列**无任何权限判定**（对比 `network/ports:132`、`asset/reconciliation:86`、`ad-domain:77` 均有判定）；生产 PG 实证：能进该页的只有 admin 与 xtgl 且都持完整写权限，故生产不复现，但配置只读角色即越权显示入口
- **3f130 空因**：后端诊断端点 `POST /ops/workstation-device/:id/physical-diagnosis` 已交付（v1.23 commit `8b85fe6b`），**前端未接**——用户裁定本里程碑只做前端诊断展示，不写生产数据
- **F5 刷新落 /dashboard**：v1.23 UAT 顺带发现的独立债
- **ColumnConfigModal:136**：缺 eslint-disable 理由注释（契约登记完整性）

**Target features:**
- **VIS（核心）**：消除 affix-wrapper 内层边框（双框消失）+ 全局 CSS 与 antd 6 变体的一致性审计（同类隐患一次性收口）
- **PERM**：系统用户页按 `system:user:edit/remove/resetPwd/add/import/export` 条件渲染操作入口（fail-closed）+ 权限矩阵单测
- **DIAG**：工位管理页「物理链路设备」为空时展示可诊断原因（emptyReason/emptyMessage/warnings），把后端已交付的诊断能力暴露到 UI
- **NAV**：F5 刷新保留当前落地页（不回退 /dashboard），且不影响未登录跳登录、无权限跳 403、deep link 直达
- **HYG**：ColumnConfigModal:136 豁免注释补齐；lint 存量 17 errors + exhaustive-deps 12 处清零；react-refresh 33 处清零 + devices 页 692 行拆分

**关键决策 (v1.24 init):**
- **双框修根因而非遮盖**——审计全局 `!important` 规则与 antd 6 结构的冲突面，不做「一处覆盖了事」（用户裁定「修根因 + CSS 一致性审计」）
- **#16 修复取 fail-closed**——权限集合为空/未加载时默认隐藏写操作入口（与「操作入口不该存在」契约一致），代价是权限加载完成前按钮不出现，需在 UI 上可接受
- **3f130 只做前端展示**——不动生产数据（用户裁定），数据侧绑定留用户动作
- **零新依赖**——沿用 v1.23 原则，不引视觉回归自动化工具链

**研究基线:** `.planning/research/{STACK,FEATURES,ARCHITECTURE,PITFALLS}.md`（4 维并行研究，2026-10-28）｜ **权限基线:** `.planning/notes/2026-10-28-pg-permission-baseline.md`（生产 PG 只读实测 + #16 定责）

## Past Milestone: v1.23 前端契约全量收尾与遗留清偿 (Frontend Contract Full Rollout & Legacy Clearance) — ✅ SHIPPED 2026-10-28（历史归档）

**Goal:** 把 v1.22 建立但未落地的契约真正推到全量——修复用户主诉的「文本输入框无法一键删除」，并清偿 v1.22 收档结转的 debug / UAT / 流程积压，让里程碑以「无债、无盲区」收档。

**交付终态（2026-10-28 收档）:** 22/22 需求 Satisfied；121 文件迁移、P65 豁免 127→0、无豁免口径裸组件归零；7 项 debug 全终结（#5 双侧归一化 / #7 空因可诊断端点 / #2 Cisco 采集 / #3 Go 写防线，#1/#4 实证不复现，#6 关单）；lint errors 81→17、warnings 1366→34、any 958→0、unused 364→0、门禁上线；实机 UAT 18/19 + FAIL-1 修复复验 + §8 回写关债。Known Gaps 转 v1.24：#16 需 PG 复测定责、3f130 绑 user_id、F5 跳转 /dashboard。

**问题来源（用户实机反馈 + v1.22 收档结转）:**
- 用户主诉：「v1.22 完成后文本输入框无法一键删除等细节问题还是没有解决」——经代码核实，**根因是多行输入框（TextArea）74 处 0 覆盖 allowClear**：v1.22 的 `AppInput` 契约只包了单行 `Input`，`eslint.config.js:114` 注释又明确声明 `Input.TextArea` 成员表达式「有意不拦」，形成零约束盲区
- 存量迁移未完成：`AppInput`/`AppSelect` 在 `src/pages/` 仅 15/13 处使用，裸 `<Input>` ~188-220 行、裸 `<Select>` ~271 行仍在 P65/P66 豁免 override 中（COMP-05 只做了试点页）
- 7 项 debug 遗留经逐份核读，**≥4 项调查未完成**（acknowledge ≠ 已定位），第一波必须是重验 spike 而非修复 wave

**Target features:**
- **TEXT（核心）**：`AppTextArea` 独立契约组件（allowClear 默认开启）+ SearchForm `textarea` 字段类型 + eslint 成员表达式/解构形态双 selector 拦截 + 74 处存量迁移
- **MIG**：裸 Input/Select 全量迁移到契约组件，P65/P66 豁免清单清空、豁免块移除，契约规则对全 `src/` 生效
- **DBG**：7 项 debug 遗留清偿（#6 scrapligo 零代码关单 / #2 Cisco 采集 / #3 MAC 格式防线 / #1 Linux 日志 / #4 迁移双执行 / #5 历史查询 / #7 工位链路）
- **UAT**：68.1 UI-SPEC §8 全表 19 条浏览器实机冒烟（重点 #13 分页 / #15 固定列 / #16 行操作权限），结果回写关闭债务
- **HYG**：`.planning/quick/` 94 个历史目录归属归档 + lint 清偿（unused-vars/no-console 归零、any 降 50%、`--max-warnings` 门禁）+ 权威计数基线落盘

**关键决策 (v1.23 init):**
- **AppTextArea 独立具名导出**，不挂 `AppInput.TextArea` 静态别名——成员表达式 selector 无法区分合法契约用法与非法裸用法，挂别名会永久放弃成员表达式拦截面、74 处收口假绿
- 数量排期口径 = **eslint selector 实跑命中数 + 豁免清单条目数**（权威），grep 数仅供粗估（研究文档间口径 157-220 不一）
- **DBG-03 走 Go 侧防线**（写入方枚举 + 显式 normalize + 回归测试），**不使用 DB TRIGGER**——维持 v1.19「禁用 DB TRIGGER 路线」决策，规避 SQLite 双实现成本与数据流隐式化
- 排序约束：迁移 → 收口 → lint → UAT（UAT 必须测迁移后的页面）；后端 debug 线与前端线全程无共享文件，可并行
- 零新依赖（前端不引 jscodeshift/Playwright，UAT 用已有 agent-browser skill；后端 scrapligo v1.4.2 现状即可）

**研究基线:** `.planning/research/{STACK,FEATURES,ARCHITECTURE,PITFALLS,SUMMARY}.md`（4 维并行研究，overall confidence HIGH，commit `b5b3aeaf`）

## Past Milestone: v1.22 前端公用组件整治 (Frontend Shared Component Remediation) — ✅ SHIPPED 2026-10-27（历史归档）

**Goal:** 消化 2026-10-23 全量前端组件审查的 201 条问题（P0 24 / P1 87 / P2 90），核心是把公共层从"只有零件没有契约"补成"有契约"：默认值下沉（allowClear/showSearch）、分页契约下沉（showTotal/setTotal）、危险操作唯一入口（ConfirmDelete），使抽出的组件真正被用起来。

**问题来源（已定稿的审查报告，作为本里程碑的事实基线）:**
- 主方案：`docs/2026-10-23-前端公用组件审查与最佳实践方案.md`（677 行：量化诊断 / 根因 / ROI 矩阵 / 三层架构 / 15 条硬规则 / 四阶段路线图）
- 分片报告：`.planning/notes/frontend-audit/{01-system,02-operations-asset-vdi,03-network-monitor,04-duty-workorder-ad}.md`（合计 201 条，全部精确到 文件:行号）

**Target features（四阶段路线图）:**
- **P0 止血**：`usePagination.ts:122` showTotal 把总条数算成页数（51 页受害）；backups:79 / discoveries:34 `_setTotal` 从未调用（total 恒 0）；ad-domain/ous:130 固定 pageSize:100 前端分页截断；27 个业务弹窗 confirmLoading 0 命中（双击重复提交）；7 处批量删除无二次确认；3 处假功能入口（message.info 空确认、ReconciliationDrawer 永久空 Tab）；6 处 fixed 列无 scroll.x 失效
- **P1 修复**：表格 loading / scroll / Modal maskClosable+脏检查 / 本地排序误用 / console 错误出口收敛 / aria 可访问名
- **公共层契约组件抽取**：AppInput / AppSelect（默认值下沉，修 285 Input 仅 23% allowClear、400 Select 仅 30% 的根因）、AppTable / AppPagination、PageTable、FormModal、SearchForm、ConfirmDelete
- **P2 一致性债务**：hex 硬编码 / 日期状态格式化 / any 收敛 / index key 等

**关键决策 (v1.22 init):**
- 抽取原则：抽契约不抽业务（资产选择器/部门树等业务语义组件不抽）；useDebouncedSearch 并入 RemoteSelect 不单独抽
- 默认值必须下沉封装层，禁止逐页补 allowClear（横跨 73 文件，不可维护）
- ConfirmDelete 成为危险操作唯一入口；危险操作必须二次确认
- 预期收益：四片外推全前端消除约 7,000~8,000 行样板

## Past Milestone: v1.21 AD 双向同步 bug 修复 (AD Bidirectional Sync Bug Fixes) — ✅ SHIPPED 2026-10-22（历史归档）

**Goal:** 修复 AD↔系统双向同步的 2 个 P0 bug（部门回退乒乓、软删用户接管风险）、3 个 P1 数据一致性缺口和 P2 卫生项，以三方审计修订后的 outbox-lite 架构落地（≈400 行核心层），消除"改了部门又弹回"与静默数据丢失。

**背景:** 三份独立 subagent 审计（架构/代码核对/魔鬼代言人）确认 11 项 bug 属实，并否决了时间戳仲裁（GORM 自动刷新 updated_at 使其恒失效）与完整 Outbox/FieldRule 引擎（过度设计），形成修订方案。

**Target features / 修复项:**
- **ad_push_queue (outbox-lite)**：`user_service.Update` 同事务入队 + in-process cron 排水（不进 sys_job）+ 按实体合并 dedupe —— 解决重启丢推送、dedupe 丢弃、为登录守卫提供信号源
- **排水 compare-and-move**：闭包内重读最新值 → 目标 OU 归一化比较（NormalizeDN）→ 不等才 MoveUser（结构性修 F1，幂等，规避激活未验证重试分支）；`ErrADTargetNotFound` → 终态
- **登录守卫**：登录写 dept 前查该用户 pending/failed 队列行 → 有则跳过覆盖 + WARN（替代被证死的时间戳仲裁）
- **F2 软删恢复安全**：ad_dn 校验后恢复（不一致拒绝+审计）；sys 删除用户 → 入队 disable AD 账号（闭环复活链路）
- **影子表删除清理**：挂 syncDataInternal（复用 singleflight）+ 两道闸门（0 条拒绝 / >50% 拒绝）+ 覆盖组成员关系
- **组成员同步事务化**（Delete+Create 包 Transaction）
- **对账视图**：普通 SQL VIEW `v_dept_ou_mismatch`（禁用 MV）+ 只读端点；含"映射指向软删部门"条目
- **死代码清理**：user_ou_service.go 整文件 + GetUserDeptByADOU + BatchMoveUsersToNewOU + 过期注释（dept_ou_mapper.go:31 等）
- **P2 卫生**：First() 确定性/按 ad_dn 路由配置、OU DN 转义 helper、嵌套组 member 标记、同名兄弟部门 OU DN 冲突、getDefaultADConfigID 去重、sqlite 测试补建表

**关键决策 (v1.21 init，三方审计裁决):**
- 归属契约：代码内映射 struct + 注释（不建 FieldRule 表驱动引擎）；email/phone/displayName = write-through 委托写
- 时间仲裁：弃用（GORM updated_at 自动刷新致恒失效）→ 信号守卫
- FK 约束：后置可选（软删语义收益低）→ 应用层清理映射替代
- uSNChanged 增量同步：挂起（性能优化非正确性）
- 对账视图用 VIEW 禁用 MV（避免 MV 刷新僵尸/AutoMigrate 阻塞前科）

**前置条件（已满足 2026-10-16）:** ✅ 1641 个未提交文件已分 5 组提交并合入 main（`ced2d75b..31c7eee8`），已推送 origin（gitcode.com/ninedrunk/go_react），工作区干净。

**Phase 57 已完成（2026-10-16/17，v1.21 首个 phase）:** ad_push_queue 基建落地——migration_215（部分唯一索引谓词单一事实源 `models.ADPushQueueActivePredicate`）+ **五条**写路径同事务入队（Update / UpdateStatus / Delete / BatchDelete / profile.UpdateUserInfo，后两条分别为 GAP-01/GAP-02 补接）+ disable 压制同实体 pending update_user（F-01）。code review F-01~F-06 全修（变异验证可红）+ verifier 独立复核 GAP-01/02/03 闭环。VERIFICATION 判定 `human_needed`：PG 实机合并路径 / kill -9 重启耐久 / AD UAC 533 闭环三项留档 `57-HUMAN-UAT.md`。Phase 58 移交待决项（两条遗留直推路径处置、ad_config_id 取舍、F-07~F-16）已落 ROADMAP。

**Archived:** `.planning/milestones/v1.21-ROADMAP.md` + `.planning/milestones/v1.21-REQUIREMENTS.md` + `.planning/milestones/v1.21-MILESTONE-AUDIT.md`（里程碑审计 passed：6 phase VERIFICATION 全 passed、19/19 REQ、8/8 集成点 VERIFIED、E2E 流 A-D 全通过）

## Historical State (v1.25, archived)

- **Shipped**: v1.0 ~ **v1.24** — v1.24 = Phases 76-83（8 phases），**18/18 requirements satisfied**，SHIPPED & ARCHIVED 2026-10-28。46 commits / 104 files / +5,419 −2,232，约 3.5 小时（09:19 需求定稿 → 12:59 收口）。归档：`milestones/v1.24-ROADMAP.md` + `milestones/v1.24-REQUIREMENTS.md` + `milestones/v1.24-phases/`。
- **历史里程碑 v1.25 全站权限治理与样式收敛**（SHIPPED 2026-10-30，历史归档、非当前状态）：INITIATED 2026-10-28，五项候选全部立项（①操作级 RBAC 全站治理 / ②antd 6 variant 收敛 / ③权限模型三缺口 / ④Phase 77 视觉新发现 / ⑤room-devices 815 行拆分），执行顺序 ①→②；当前里程碑为 v1.52（见文档顶部）。
- **v1.24 终态**：用户主诉「输入框双重外框」根治（309 个输入框，双框元素数 3→0）；#16 行操作权限越权闭环（只读角色写接口 200 真删 → 403 拒绝）；lint 存量 51 → 0 problems 且门禁归零；devices 页 714 → 306 行；生产库验收夹具已清理、既有行零改动。
- **v1.25 起点基线**：前端 76 个页面（index.tsx），仅 14 个接入 `useMenuStore.permissions`；后端 76 个路由组 / 134 处 `RequirePermissions` 挂载，`pkg/permission/config.go:223-289` 动作级映射表已定义未接线；`room-devices/index.tsx` 815 行；CSS `!important` 1359 处 / 219 行 antd 选择器。
- **v1.22 终态**：前端公共层从"只有零件没有契约"补成"有契约"——AppInput/AppSelect 默认值下沉（allowClear/showSearch）、ConfirmDelete 危险操作唯一入口（68 处收口）、PageTable/FormModal/RemoteSelect 契约内建、hex/日期/any 三类债务收敛、handleApiError 唯一错误出口。
- **v1.22 遗留（→ v1.23）**：TextArea 74 处零 allowClear 覆盖（用户主诉根因）+ 裸 Input/Select 存量迁移未完成（P65/P66 豁免 override 仍在）+ 7 项 debug acknowledge（≥4 项调查未完成）+ 浏览器实机 UAT 债 + `.planning/quick/` 94 个历史目录 + lint 1367 warnings / 83 errors。
- **构建状态**：`go build ./...` clean；`go test ./internal/services/addomain/ ./internal/api/v1/... ./internal/scheduler/` 全绿；`go vet` 仅存量 `internal/core/security/ad_authenticator_test.go:159 unreachable code`（非本次影响面）。前端基线：eslint 0 problems / vitest 569 文件 3960 用例全绿 / vendor-react gzip 776 kB。
- **v1.22 终态**：前端公共层从"只有零件没有契约"补成"有契约"——AppInput/AppSelect 默认值下沉（allowClear/showSearch）、ConfirmDelete 危险操作唯一入口（68 处收口）、PageTable/FormModal/RemoteSelect 契约内建、hex/日期/any 三类债务收敛、handleApiError 唯一错误出口。
- **v1.22 遗留（→ v1.23）**：TextArea 74 处零 allowClear 覆盖（用户主诉根因）+ 裸 Input/Select 存量迁移未完成（P65/P66 豁免 override 仍在）+ 7 项 debug acknowledge（≥4 项调查未完成）+ 浏览器实机 UAT 债 + `.planning/quick/` 94 个历史目录 + lint 1367 warnings / 83 errors。
- **构建状态**：`go build ./...` clean；`go test ./internal/services/addomain/ ./internal/api/v1/... ./internal/scheduler/` 全绿；`go vet` 仅存量 `internal/core/security/ad_authenticator_test.go:159 unreachable code`（非本次影响面）。
- **（v1.21 历史）生产实锤**：全量同步轮 user_count=4177、error_count=0、67 组；嵌套组成员污染 8→0（重放自愈）；AD 部门变更 ≤30s 推送并 compare-and-move 落位（NormalizeDN 幂等短路）；登录回写守卫双半段（拦截 / 正常覆盖）与软删 DN 漂移拦截均端到端验证。
- **（v1.21 历史）Phase 62 引入的生产回归已修复**：全嵌套组过滤后 kept 为空 → GORM ErrEmptySlice 中止整轮 → 补早返回 + 回归用例 + 二进制重建发版（14:22）→ 14:35 轮 full success。

## Next Milestone Goals（v1.26 候选）

**v1.24 结转的 5 笔候选债 → 已全部立项为 v1.25（2026-10-28 用户裁定，五条全纳入，执行顺序 ①→②）**：

1. **操作级 RBAC 全站治理** — 108 页面 + 340 路由；`roles/menus/depts/posts` 4 组 ANY-of 拆分（逐动作清单已备）+ `pkg/permission/config.go` 死表接线 + 动作级 F 型菜单播种（当前 devices 等模块因无菜单种子而无法授予动作级权限）→ **v1.25 RBAC 线**
2. **样式层收敛到 antd 6 variant**（方案 C）— 需先重构 AntdThemeBridge 明暗色板（F3：`colorBgContainer` 硬编码 `#FFFFFF`，darkAlgorithm 不改写显式 token）→ **v1.25 STYLE 线**
3. **权限模型三缺口** — `data_scope` 未被列表查询消费（实测 data_scope=3 仍返回 total=4231 跨部门）/ `GetUserPermissions` 不过滤 `menu.status`（停用菜单权限仍进前端权限集）/ Step 2「子菜单继承」语义易写反；附带生产库无迁移记录表 + GORM schema 漂移 → **v1.25 PMODEL 线**
4. **Phase 77 视觉新发现** — 暗色 `--input-bg` 第 3 处就地重定义（`index.css:4807`）/ 暗色 Select 白底导致同页两套边框体系 / `.dept-tree-search` 死规则 / antd 6 弃用警告 2 类 / InputNumber 错误态仍灰 → **v1.25 VISFIX 线**
5. **room-devices 页拆分** — 815 行，比已拆的 devices 更长 → **v1.25 SPLIT 线**

**更早的 deferred（继续挂账）**：

> 本节为历史挂账快照；当前状态见 STATE.md / ROADMAP.md。

以下候选源自 v1.21/v1.22 收口审计与历史 deferred，**v1.23/v1.24 均未纳入**：

1. **AD 方向补齐** — ad_push_queue 目前只有 update_user / disable 两个动作方向，缺 enable（软删用户恢复后重新启用 AD 账号），见 `.planning/seeds/SEED-ad-push-enable-direction.md`。（2026-11-08 开放项盘点 #A1）
2. **同组管理员 OU 写权限** — ✅ 已裁决关闭（域控侧 ACL 策略，非代码缺陷；2026-10-21 用户裁决不修，仅作已知环境限制记录）。
3. **残留现场 UAT 债** — v1.18（3 项真机 SNMP/CLI）+ v1.19（6 项 SSH）+ v1.20（12 项 VLAN/绑定 site-visit）仍为 deferred，需现场运维同事执行后回写 UAT。⚠️ 原始 `48-HUMAN-UAT.md` / `54-HUMAN-UAT.md` / `56-HUMAN-UAT.md` 三份清单当前磁盘不存在（见 notes/261102-planning-sync-strategy.md 归档策略），执行前须重建；v1.51 Phase 129 另有 H3C 2 条 deferred 与 129-RESULTS.md §五 5 条复测遗留（2026-11-08 开放项盘点 #A5）。
4. **全量同步节流** — 3 小时全量 cron（sync_interval=10800）+ 失败后每 5 分钟 tick 重试，重试风暴与节流策略可优化。（2026-11-08 开放项盘点 #A6）
5. **同步可观测性** — 把 v1.21 期间用于捕获重放窗口漂移的采样脚本（`scripts/uat61-t1-poller-v5.ps1`）产品化为常态化的同步漂移告警。（2026-11-08 开放项盘点 #A7）
6. **v1.23 明确延后项**（用户裁定 2026-10-27）— react-refresh 33 处清零 / devices 页 692 行拆分 / 三态组件（Empty/Skeleton/ErrorAlert）推广 — ✅ 已落地（v1.24 Phase 82/83 + PageTable 三态组件），不再挂账。

## Past Milestone: v1.17 资产对账 (Asset Reconciliation) — ✅ SHIPPED 2026-07-03（历史归档、非当前状态）

**Goal:** 建立"实物层 vs 声明层"对账引擎，使资产系统数据保持与实际情况相符。

**Target features / 工作池:**
- 多源数据冲突检测（物理链路 vs 系统责任人 vs AD managed_by）
- Observe-only 策略 + 告警驱动人工修复（不动业务表）
- IP 段级别例外规则（5 种 actions：no_alert/no_notice/no_workorder/skip_severity/silence）
- 工位详情页整合对账健康度（顶部卡片 + 行内徽标 + 抽屉）
- 自动转工单闭环（critical/high 异常）
- 半自动修复建议（高置信度，R5 可选）

**5 phases (42-46):**
- Phase 42 / R1: 观测底座（物化视图 + reconciliation 表 + dashboard + Statistics 端点）
- Phase 43 / R2: 告警 + 工单闭环
- Phase 44 / R3: 置信度评分 + IP 段例外规则 + 例外管理页 + 命中测试
- Phase 45 / R4: 工位详情整合 + 资产详情摘要块
- Phase 46 / R5: 半自动修复（可选）

**关键决策 (locked):**
- 策略：Observe-only（不修改业务表，仅记录 + 告警 + 人工修复）
- 菜单归属：资产管理 / 数据质量（资产对账 + 3 个二级菜单）
- 权限命名：asset:reconciliation:*
- API 路由：/asset/reconciliation/*
- 跨模块调用：ops/workstation → asset/reconciliation（service 层直接 + 权限降级）
- Owner 合并：运维 + 资产 + 权限 同一人（无需双签，单人签字）

**复用审计结论 (v0.4):**
- ✅ 已复用 13 项：字典/参数/operlog/Handler-Service/CacheProvider/Excel/UUID/Status/前端 hooks/opsApi/ECharts/UI库
- ⚠️ 部分复用 4 项：字典枚举值/参数 seed/Excel config/operlog module 常量
- ❌ 必须新增 7 项：cache_key helper/Statistics 端点/queryKeys/operlog module/HealthScore 函数/路由注册/Cron 注册

**R1 启动门槛 (✅ 已满足):**
- 端口采集覆盖率达标（运维 owner 确认）
- T1-T7 基础设施调查完成
- v0.3 架构 + v0.4 复用审计 + 跨模块权限声明已落盘

**决策记录 (.planning/notes/):**
- asset-reconciliation-strategy.md (v0.3 + v0.4)
- 260627-reconciliation-reuse-audit.md
- 260627-cross-module-permission.md
- 260627-port-coverage-audit.md

**Seed:** .planning/seeds/asset-reconciliation-v1.17.md

**Todo（历史归档）:** .planning/todos/done/v1.17-reconciliation-decisions.md（v1.17 已 SHIPPED 2026-07-03，T1-T30 为 2026-06-27 原始计划，勿再推进）

# RuoYi-Next 运维管理系统

## What This Is

RuoYi-Nect is a Go backend + React frontend enterprise IT operations
management system. It has evolved from a workstation-import
association feature (v1.0) into a multi-domain platform covering:

- **Operations domain**: buildings, floors, workstations, server rooms,
  info-points, dedicated lines, assets (CRUD + Excel import/export
  with geocoding).
- **Network domain**: device discovery, SNMP, Scrapli-based command
  execution, credential vault, topology, MAC history, port collection.
- **Identity domain**: AD/LDAP integration (login, OU-dept mapping,
  group auto-sync), user/role/dept/menu management, RBAC.
- **VDI domain** (v1.11): Sangfor VDI virtual desktop management with
  VM CRUD, power operations, user binding, server config.
- **Audit domain** (v1.15 / Phase 34): full-coverage operation logging
  for all 309 write endpoints with sensitive-keyword masking.

National cryptography (SM2/SM3/SM4) is mandatory for in-transit and
at-rest data on sensitive endpoints.

## Core Value

End-to-end operational observability and auditability: every write
action in the system produces a traceable record (who, when, what,
from-where, before/after-state) with sensitive fields auto-masked.
Combined with Excel-driven bulk management (import/export with
auto-resolution of references like dept-name → UUID), this gives
operators a single trustworthy system-of-record for the entire IT
estate.

## Requirements

### Validated

- ✓ 工位 CRUD（创建、读取、更新、删除）— 现有
- ✓ Excel 批量导入/导出工位 — 现有
- ✓ 工位关联 `org_id`（部门 UUID）— 现有字段，导入时通过部门名称匹配
- ✓ 前端工位列表/详情页面 — 现有
- ✓ 部门表 `sys_dept` 含 `dept_name` 字段 — 现有
- ✓ 用户表 `sys_user` 含 `user_name` 字段 — 现有
- ✓ Excel 导入模板添加"所属部门"和"所属用户"两个可选列 — v1.0
- ✓ 导入时通过部门名称/用户名匹配，写入对应 ID — v1.0
- ✓ 匹配失败留空不阻断导入 — v1.0
- ✓ 信息点导入配置添加"所属设备"列 (`deviceName` → `device_id`) — v1.1
- ✓ 信息点导入配置添加"所属端口"列 (`portName` → `port_id`) — v1.1
- ✓ 两个新列均为可选，空值不阻断导入 — v1.1
- ✓ 模板下载包含设备和端口示例值 — v1.1
- ✓ Widget 数据获取机制 (GetWidgetData/GetBatchWidgetData) — v1.2
- ✓ 前端交互完善 (Widget 选择器、仪表盘设置、模板预览) — v1.2
- ✓ 实时数据刷新 (WebSocket 连接、轮询、缓存) — v1.2
- ✓ 用户体验优化 (拖拽布局、加载状态、响应式) — v1.2
- ✓ SNMP panic 修复 (panic 恢复 + 并发安全 + 连接验证) — v1.3
- ✓ 后端代码优化 (Core 清理 + 安全修复) — v1.3
- ✓ 批量导出功能 (多实体 ZIP 打包下载) — v1.3
- ✓ **v1.16 技术债清理** (Tech-Debt Cleanup) — Phases 40/41 — 2026-06-26 (22+29 个 deferred debug session + 6 audit 文件规范化)
- ✓ **v1.17 资产对账** (Asset Reconciliation) — Phases 42-46 — 2026-07-03 (observe-only 对账引擎 + 工单闭环 + IP 段例外 + 工位详情整合 + 半自动修复建议 + UAT 9/10 PASS)
  - R1 观测底座：物化视图 + Statistics 端点 + operlog 全覆盖
  - R2 告警+工单：critical/high 自动转工单 + WebSocket + SysNotice + 7d 静默期
  - R3 IP 段例外：CIDR + 5 actions + 命中测试 + Excel 导入导出
  - R4 工位详情：健康度卡片 + ReconciliationDrawer 3 Tab
  - R5 半自动修复：6 状态机 + 7d 回滚 + 误修复率告警三通道
- ✓ **Phase 47 根因修复** (infoPoint drift / Layer3 UPSERT / port_status 漂移 / MAC parser 校验) — 2026-07-03
- ✓ **v1.24 视觉回归修复与遗留债务清偿** (Visual Regression Fix & Legacy Debt Clearance) — Phases 76-83 — 2026-10-28 (8 phases, 18/18 REQ satisfied, 46 commits / 104 files / +5,419 −2,232 in ~3.5h)
  - 双框根治：affix-wrapper 内层复位块（6 项、无 box-shadow，焦点环归 wrapper）+ status 还原；**双框元素数 3 → 0**（harness 42 行 + 15 真实页 + 登录页，明暗双测）；错误态恢复品牌红
  - CSS 一致性审计登记册 36 条（P0 已修 8 / P1 待修 7 / P2 登记 19 / 保留 1 / 样板 1）
  - #16 前后端双端闭环：严格权限中间件（跳过 F 型继承与模块降级）+ `/system/users` 读组/写组**兄弟分离** + 三态门控 + 权限码单一真相源；三层证据（DB 前后对照 / HTTP 8/8 403 含决定性对照 / 组件测试 7/7）
  - F5 刷新回退修复（红绿验证）+ 3f130 空因诊断展示 + lint 51 → 0 problems + devices 714 → 306 行
  - **Phase 77 反查出 Phase 76 自身的 P0 死规则**（antd 6 status class 名误用 → 错误态不变红），契约检查同步升级为双向断言
- ✓ **v1.18 网络设备硬件清单** (Device Component Serials) — Phase 48 — 2026-07-04 (1 phase / 3 plans / 3 waves, 14/14 D-id 覆盖 + 3 site-visit UAT 推迟)
  - Wave 1 Schema：migration_201 给 `ops_asset` 加 4 列 (parent_asset_id / source_device_id / component_type / component_slot) + `sys_data_reconciliation.recon_category`
  - Wave 2 Collectors：`internal/services/component_collector/` 17 文件包,SNMP ENTITY-MIB 单 GET + Ruijie `temprature*` 噪声过滤 + Huawei dual-class 去重 + OwnerResolver stack containtedIn tree + Huawei/Ruijie CLI 双厂商 + 6 TextFSM 模板
  - Wave 3 Pipeline+Audit+UI：OpsAssetWriter UPDATE-only + ReconciliationEmitter (component_serial recon_category) + DeviceInfoCollectionService cron hook + operlog.RecordBackground (cron-path) + GET /ops/asset/components + ComponentListTab expandable row
- ✓ **v1.19 网络设备写命令 (Network Device Port Write Operations)** — Phases 50-55 — 2026-07-08 (6 phases / 9 plans / 5 build waves + 1 cleanup phase, 37/37 MVP requirements, 108 files / 25,224 insertions / 3,152 deletions in 1.7 days)
  - W1 Vendor Templates: `internal/services/portcollection/vendor_port_template.go` 15 (vendor, action) templates (Huawei/H3C VRP heritage + Ruijie Cisco-style) + 20 table-driven tests
  - W2 PortWriteService + Batch: `internal/services/portwrite/` 5-file package — parseConfigError 5-step priority scan + checkPreState NoOp detector + BatchWritePorts detached 30min context + serial fail-fast; 28 mock unit tests
  - W3 Router/Handler/Operlog/Permission/Migration: 6 kebab POST 端点 `/network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable,batch}` + `NetworkPortWrite = "network:port:write"` 组级 2-arg RequirePermissions + `sys_port_write_audit` Path A (AutoMigrate) + `GrantNewMenuToRolesHavingParent("端口状态")` 精准授权 + Path C audit↔operlog 关联 (audit_ids 嵌 operlog.oper_param; audit.oper_log_id 保持 NULL)
  - W4 Frontend: PortWriteModal 5-action unified Modal + BulkWriteDrawer select→executing→result 状态机 (D-05 indeterminate Spin 不伪造 X/Y) + ports/logs 页 useMenuStore canWrite gating + audit-link Toast + monitor/logs URL query 预填 + vendor-react gzip 774.96 kB 零回归
  - W5 E2E + UAT Deferral + Docs: scrapligo FileTransport 10 TestE2E_* (闭环 Phase 51 mockDeviceExecutor 不调 fn 漏洞) + API响应规范 + 安全设计 + CHANGELOG/README/MILESTONES/STATE 文档同步 + 54-HUMAN-UAT.md 7 项 deferred (owner = 现场运维同事)
  - Phase 55 Tech-Debt Cleanup: WR-02 antd validator 3-param signature + IN-01 instanceof Error narrowing + IN-02 eslint-disable placement 修正 (Lesson: directive 紧贴 `}, []);` 而非 useEffect 开头) + HealthCard.test.tsx regex match + CR-02 后端 batch_orchestrator.go fallback port 归属跨层防御 (纵深双保险)

### 历史 REQ 清单归档（v1.53 已 SHIPPED，24 REQ 见 `.planning/milestones/v1.53-REQUIREMENTS.md`；以下均为非当前状态）

### Validated — v1.53 开放项全量清偿（2026-11-08 SHIPPED）

- ✓ **FIX-01~14**: 14 条代码可修开放项全部交付 — v1.53
- ✓ **VAL-07**: 人工验证执行框架（5 份重建清单 + 79 条跟踪表 + 回写口径）— v1.53
- ✓ **ACK-01~02**: LOCKED 残余 / 已知约束结论落盘 — v1.53
- ✓ **GOV-01**: 临时清单归档 + 权威交回 GSD + 一致性复核 — v1.53
- ○ **VAL-01~06**: 交付物已落盘，实际人工验证待执行（受生产 PG 只读 / 写授权 / 现场真机 / DBA 回传约束，跟踪表 79 条全部待执行、0 勾选）— 非阻断，见 `.planning/notes/261108-human-verification-tracker.md`
- 归档清单：`.planning/milestones/v1.53-REQUIREMENTS.md`（24 REQ，18 勾选 + 6 项待人工验证）


#### v1.25 全站权限治理与样式收敛（历史归档 / SHIPPED 2026-10-30，非当前里程碑）

> REQ-ID 已于需求定义阶段定稿；权威清单见 `.planning/REQUIREMENTS.md`（共 **22** 条）。
> ⚠️ **计数更正（2026-10-28 roadmap 阶段）**：本行原写「共 25 条」、REQUIREMENTS.md 原写「26 total」，二者均与该文档**实际的 22 条需求条目**（RBAC×6 + STYLE×5 + PMODEL×4 + VISFIX×6 + SPLIT×1 = 22，已全量 grep 复核）不自洽 —— 磁盘上不存在第 23-26 条需求条目。Roadmap 按 **22 条**映射（22/22 全覆盖）。

**RBAC — 操作级权限全站治理**
- [ ] **RBAC-01**: 只读角色在任意业务模块调写接口一律 403 且无数据变更（写路由逐动作单一权限点，去 ANY-of）
- [ ] **RBAC-02**: 前端 76 页面操作入口按动作级权限码三态渲染（无权限不渲染 / 未就绪禁用+提示）
- [ ] **RBAC-03**: `pkg/permission/config.go` 动作映射表真实接线并可校验（未登记的写路由 → 校验失败）
- [ ] **RBAC-04**: `system` 域其余 4 组（roles/menus/depts/posts）写操作 ANY-of → 逐动作单一权限点
- [ ] **RBAC-05**: 动作级 F 型菜单播种，权限可被真实授予（前端可见性与后端放行同源）
- [ ] **RBAC-06**: 权限码单一真相源三方零漂移，校验脚本覆盖新铺开面并接入 CI 门禁

**STYLE — 样式层收敛到 antd 6 variant**
- [ ] **STYLE-01**: `AntdThemeBridge` 明暗色板重构（品牌 token 交算法派生，暗色容器不再白底）
- [ ] **STYLE-02**: 手写 antd 覆盖规则按四批清单收敛（可删批清零 / 可降级批替换 / 需重写批重写 / 必须保留批登记）
- [ ] **STYLE-03**: 输入类组件圆角统一 6px → 8px 设计 token
- [ ] **STYLE-04**: hex 治理覆盖 `.css`（现 eslint selector 只匹配 JS/TS 节点），收敛期不新增硬编码色
- [ ] **STYLE-05**: 明暗双主题下与 antd 官方渲染一致（computed-style 断言 + 截图矩阵），且不破坏 VIS-01/02/03

**PMODEL — 权限模型三缺口**
- [ ] **PMODEL-01**: `data_scope` 被列表查询真实消费（data_scope=3 不再返回跨部门全量）
- [ ] **PMODEL-02**: `GetUserPermissions` 过滤 `menu.status`（停用菜单权限码不进前端权限集）
- [ ] **PMODEL-03**: Step 2「子菜单继承」语义文档化并以明确命名/注释固化
- [ ] **PMODEL-04**: 迁移记录表补齐（版本可核对）+ GORM schema 漂移项登记为显式债务

**VISFIX — Phase 77 视觉新发现**
- [ ] **VISFIX-01**: 暗色输入框底色不再被 `--input-bg` 第 3 处就地重定义覆写（对比度达标）
- [ ] **VISFIX-02**: 暗色 Select 容器底色与边框随色板派生（与 STYLE-01 联动）
- [ ] **VISFIX-03**: `.dept-tree-search` 死规则清理
- [ ] **VISFIX-04**: antd 6 弃用警告 2 类清零（`onSearch→showSearch` / `Modal maskClosable→mask.closable`）
- [ ] **VISFIX-05**: InputNumber 错误态边框色还原
- [ ] **VISFIX-06**: CSS 契约脚本自身修复（`css-contract-assertions.js` 错误态 fixture 用了 antd 6 不存在的 class → VIS-03 动态断言从未真正通过；`css-contract-check.js` C3b 反转 + 新增 C6-C15 断言）

**SPLIT — 巨型文件拆分**
- [ ] **SPLIT-01**: `room-devices/index.tsx` 815 行按 devices 页范式拆分，主文件 ≤ 350 行，每新文件 ≤ 250 行，行为零变化

#### v1.24 视觉回归修复与遗留债务清偿（已 SHIPPED 2026-10-28）

- [x] **VIS-01**: affix-wrapper 内层复位，消除全站 309 个输入框的双重外框
- [x] **VIS-02**: 焦点环单圈化（边框与焦点环唯一归 wrapper）
- [x] **VIS-03**: Form 错误态边框色还原（status → colorError）
- [x] **VIS-04**: 全局 CSS 与 antd 6 变体的一致性审计登记册（36 条）
- [x] **VIS-05**: 浏览器实机视觉回归验证（21 形态 × 明暗矩阵 + 15 页巡检 + 17 张证据）
- [x] **PERM-01**: ActionButton 契约扩可选 permission 字段（加法，v1.19 冻结面对外形状不变）
- [x] **PERM-02**: 三态权限门控 helper（ready/loading/error，禁用 permissions.length===0 判未就绪）
- [x] **PERM-03**: 系统用户页操作入口按权限码门控
- [x] **PERM-04**: 后端严格权限中间件（不做模块降级）+ /system/users 写操作拆独立子组
- [x] **PERM-05**: 受限角色 PG 实机验收（HTTP 8/8 写接口 403 + 对照实验）
- [x] **PERM-06**: 权限码映射表单一真相源 + 双向漂移校验脚本
- [x] **DIAG-01**: 3f130 工位物理链路空因诊断前端展示（Table+Alert 分级）
- [x] **NAV-01**: F5 刷新保留当前落地页（catch-all 语义区分 + LAST_PATH 防自毁反写）
- [x] **HYG-01**: ColumnConfigModal:136 豁免理由注释补齐
- [x] **HYG-02**: lint errors + warnings 清零（51 → 0 problems）
- [x] **HYG-03**: react-refresh 33 处清零
- [x] **HYG-04**: devices 页拆分（714 → 306 行）
- [x] **HYG-05**: 豁免理由表全量同步（159 → 38）并接入 lint:ci

#### v1.23 前端契约全量收尾与遗留清偿（已 SHIPPED 2026-10-28）

- [ ] **TEXT-01**: 用户在任意表单弹窗的多行输入框（备注/描述/原因）能一键清空内容，无需手动逐字删除
- [ ] **TEXT-02**: 存量 74 处多行输入框全部接入 AppTextArea 契约，行为一致
- [ ] **TEXT-03**: 搜索区（SearchForm/PageTable）支持多行输入字段类型
- [ ] **TEXT-04**: 裸多行输入框被 eslint 拦截（成员表达式 + 解构形态双覆盖）
- [ ] **MIG-01**: 存量裸 `<Input>` 全部改用 AppInput 契约
- [ ] **MIG-02**: 存量裸 `<Select>` 全部改用 AppSelect 契约
- [ ] **MIG-03**: P65/P66 豁免清单清空、豁免块移除，契约规则对全 `src/` 生效
- [ ] **MIG-04**: 迁移不破坏 v1.22 冻结契约（分页口径 / ConfirmDelete 唯一入口 / handleApiError / statusHelpers）
- [ ] **DBG-01**: scrapligo GetPrompt nil panic 经人工验证无复现后关单
- [ ] **DBG-02**: Cisco 设备 MAC 地址采集支持厂商模板（验收用例 `b022.7a2e.4a4f`）
- [ ] **DBG-03**: MAC 地址格式问题不再复发（写入方枚举 + 显式 normalize + 回归测试）
- [ ] **DBG-04**: Linux 环境应用日志正常落盘
- [ ] **DBG-05**: MAC 历史迁移任务不重复执行（cron 共享引擎改动须过全 scheduler 测试套件）
- [ ] **DBG-06**: MAC 历史查询返回正确结果
- [ ] **DBG-07**: 工位 3F130 物理链路为空可区分空因状态（可诊断）
- [ ] **UAT-01**: 68.1 UI-SPEC §8 回归清单 19 条浏览器实机全部通过
- [ ] **UAT-02**: UAT 结果回写 68.1 UI-SPEC §8 并带证据，债务不再无主结转
- [ ] **HYG-01**: `.planning/quick/` 94 个历史任务目录归属判定与归档
- [ ] **HYG-02**: lint `no-unused-vars` + `no-console` 归零
- [ ] **HYG-03**: lint `no-explicit-any` 降低 50%
- [ ] **HYG-04**: lint `--max-warnings` 门禁纳入 npm script 防回退
- [ ] **HYG-05**: 权威计数基线落盘（eslint 命中数 + 豁免清单条目数）

## Past Milestone: v1.19 网络设备写命令 (Network Device Port Write Operations) — SHIPPED 2026-07-08（历史归档）

**v1.19 已闭环** — 6 phases (50-54 build + 55 cleanup) / 9 plans / 5 build waves + 1 cleanup / 37 MVP requirements all addressed (35 fully satisfied + 2 PARTIAL BATCH-05/UI-03 + 1 adjusted PERM-03 D-07 父菜单名). 闭环分析:
- **Build (W1-W5)**: Phase 50 (vendor templates) → Phase 51 (service + batch + mock) → Phase 52 (router/handler/operlog/permission/migration) → Phase 53 (frontend drawer + API wrappers) → Phase 54 (e2e + UAT deferral + docs)
- **Cleanup**: Phase 55 (tech-debt cleanup of Phase 53 leftovers — WR-02 custom-reason validator + IN-01 instanceof Error + IN-02 eslint-disable placement + HealthCard.test.tsx + CR-02 后端 batch_orchestrator 跨层防御)
- **Tech Debt Zero-regression**: Phase 51 mock + Phase 54 e2e 全绿(28+35 tests), operlog 25 OperType + 11 敏感关键词 regression_test.go 不回归, vendor-react gzip 774.96 kB 与 Phase 53 baseline 零回归
- **Phase 51 mock 漏洞 闭环**: scrapligo `transport.NewFileTransport()` 真实 SendConfigs + parseConfigError 链路, 10 TestE2E_* 测试全绿(5 single happy + 1 batch happy + device_rejected + transport_error + batch_failfast + noop)
- **文档同步**: API响应规范.md (网络设备端口写操作小节) + 加密设计.md (SSH vs HTTP 两层加密正交) + 新建 CHANGELOG.md + README.md 核心特性扩展 + MILESTONES.md v1.19 条目
- **UAT 推迟**: 6 项 SSH verification (Huawei/H3C/Ruijie × shutdown/description/dot1x) → 54-HUMAN-UAT.md (推迟到下次现场访问, owner = 现场运维同事); 7 项全 `pending`, 全 `verifier_status: human_needed`
- **Pre-existing 测试失败** (login_encryption_test.go:235 / pkg/errors/errors_test.go / operations 子树) 均先于 Phase 51-54 6 周-5 个月, git diff empty for these files, 不算 v1.19 缺陷

**Phase 54 must-not-break 全部守住**: SC#6 三绿 (go test portwrite + npm build vendor-react gzip 774.96 kB 与 Phase 53 baseline 零回归 + npm type-check); SC#7 operlog regression_test.go (25 OperType + 11 keyword + Record 5 参签名); SC#3 加密配置实证 (config.yaml exclude_paths 不含 /network/ports/write/*, D-04 lock)。

**🔜 Next Milestone (v1.19.x+ follow-up candidates)**:

参见 `milestones/v1.19-ROADMAP.md` §Tech Debt Incurred + `milestones/v1.19-REQUIREMENTS.md` §Future Requirements 列出的 14 项 future work + 5 项 tech debt（WR-01 / WR-03 / WR-04 / WR-05 + BATCH-05 real-time progress）。优先 candidates:
- **BATCH-05 real-time progress via SSE/WS** (MVP 用 indeterminate Spin) — `internal/services/portwrite/` 进度通道 + `BulkWriteDrawer` 改造为真实 X/Y 进度
- **3-vendor e2e fixture 全覆盖** (Huawei done; H3C + Ruijie 待补)
- **6 site-visit SSH verification** (Huawei S5700/S5735/H3C/Ruijie × shutdown/description/dot1x) — owner = 现场运维同事
- **WR-01..05 + IN-XX** remaining tech-debt items

## Past Milestone: v1.18 — SHIPPED 2026-07-04（历史归档；原 "awaiting v1.19 planning" 为当时状态）

**v1.18 已闭环** — Phase 48, 3/3 plans, 0 critical regressions, 14/14 D-id 覆盖 + 3 项 site-visit UAT 推迟到下次现场访问(S8700/RS8607E 真机)。

**Phase 49 v1.18 gap-closure (2026-07-05)** — 用户报告 RS8607E-03 展开组件清单为空,诊断出 3 个连锁缺口(Gap 3 chassis SN 全空 → Gap 2 关联断裂 → Gap 1 collectComponentInfo dead code)。2 plans 修复:enrichChassisSerial 接入 + collectBoardsInto 接入(chassis 行丢弃)。生产 E2E 验证:14 台设备 serial_number 回填(部署前 0),RS8607E-03 展开 9 条组件(7 板卡 + 2 光模块),超过 ≥6 门槛。E2E 期间附带修复 ReconciliationEmitter 的 PG `IS ?` SQLSTATE 42601 跨方言 bug。verifier 7/7 passed。

## Past Milestone: v1.19 (历史 planning 候选，已归档 / 非当前状态)

候选方向(择 1-3 个,phase scope 决策)：

- **真机 site-visit UAT 闭环** — 把 48-HUMAN-UAT.md 的 3 项 deferred(S8700 / RS8607E SNMP + CLI + D-10 两步流水线)由现场运维同事跑通后回写 `48-HUMAN-UAT.md` 为 `passed`;同时确认 49-HUMAN-UAT Step 3 前端 Tab 渲染
- **Phase 49 deferred follow-ups** — Huawei S5735 `ESN of slot` parser 增强固定形态设备 chassis SN 提取;连接池 TOCTOU 竞态根治(`connection_pool.go:266-273`);`recoverPendingTasks` 同时清理过期 `running` 僵尸任务(06-13 crash 暴露)
- **残余 deferred debug session 清理** — 当前 audit-open 18 debug_sessions + 60 quick_tasks,虽然 Phase 41 已闭环 29 个,仍可继续推回到 debug_sessions < 5
- **VDI 22C / 22D** — 依赖 v1.11 生产稳定性观察数据(账号管理 + Web Console 子阶段)
- **新功能(mobile app / AI features)** — 待业务决策

**Dependency**: v1.19 候选项均与 v1.18 独立;真机 site-visit 是唯一依赖 Phase 48 部署环境的项。

See `.planning/ROADMAP.md` for current state + `.planning/MILESTONES.md` for shipped-milestone history.

### Completed

#### v1.3 技术债清理 — ✅ Completed 2026-04-27

**SNMP-01: SNMP Panic 修复**
- [x] SNMP-01a: 实现 panic 恢复包装器
- [x] SNMP-01b: 实现 RWMutex 并发安全机制
- [x] SNMP-01c: 实现连接就绪性验证

**CODE-02: 后端代码优化**
- [x] CODE-02a: 系统性扫描死代码（无死代码可删除）
- [x] CODE-02b: 清理 Core 结构（删除 2 字段，文档化 12 字段）
- [x] CODE-02c: 修复安全问题（WebSocket、并发安全、错误日志）

**EXPORT-03: 网络设备导出**
- [x] EXPORT-03a: 注册导出路由 ✅
- [x] EXPORT-03b: 前端集成 ✅
- [x] EXPORT-03c: 厂商特定格式 ✅
- [x] D-04: 实现批量导出功能 ✅

#### v1.2 可配置仪表盘生产级改造（仿 Zabbix）— ✅ Completed 2026-04-21

**WDG-01: Widget 数据获取机制**
- [x] WDG-01a: 实现 `GetWidgetData` 方法
- [x] WDG-01b: 实现 `GetBatchWidgetData` 批量获取
- [x] WDG-01c: 支持数据转换配置（transform）
- [x] WDG-01d: 支持三种数据源类型（API、WebSocket、Static）

**FE-02: 前端未完成功能**
- [x] FE-02a: 实现 Widget 选择器组件
- [x] FE-02b: 实现仪表盘设置面板
- [x] FE-02c: 实现模板预览功能
- [x] FE-02d: 完善 Widget 编辑/删除交互

**RT-03: 实时数据刷新**
- [x] RT-03a: 实现 WebSocket 连接管理
- [x] RT-03b: 实现基于刷新间隔的轮询机制
- [x] RT-03c: 实现缓存策略
- [x] RT-03d: 前端实时数据订阅和更新

**UX-04: 用户体验优化**
- [x] UX-04a: Widget 拖拽布局优化
- [x] UX-04b: 仪表盘加载状态和错误处理
- [x] UX-04c: 数据为空时的友好提示
- [x] UX-04d: 响应式布局适配

### Out of Scope

- 导入时自动创建不存在的部门或用户 — 超出导入功能职责范围
- 批量修改已有工位的部门/用户关联 — 本期仅关注导入场景
- 工位与用户的多对多关系 — 当前只需一对一关联
- 设备/端口多对多关系 — 当前只需一对一
- 导入时自动创建不存在的设备或端口 — 超出导入功能职责
- 端口与设备的级联验证（确认端口属于指定设备）— 复杂度高，用户未要求
- **仪表盘系统统一** — 保留两套系统：固定监控仪表盘（/monitor/dashboard）和可配置仪表盘（/dashboard-system）
- 复杂图表类型（3D 图表、地图）— 超出本期范围
- 告警规则配置 — 属于监控系统，非仪表盘核心
- **roles/menus/depts/posts 4 组 ANY-of 拆分** — 与 #16 同族但属「全站治理」范围，v1.24 只闭环 system/user 一个模块（写路由逐动作清单已登记待用）
- **3f130 绑定 user_id 数据修复** — 用户裁定本里程碑只做前端诊断展示，不写生产数据
- **room-devices 页拆分** — 本轮只拆 devices（714→306），room-devices 815 行登记为候选债
- Zabbix 告警集成 — 超出本期范围

## Context

### 技术环境
- **后端**: Go 1.24 + Gin + GORM + PostgreSQL
- **前端**: React 19 + TypeScript + Ant Design 6 + Zustand
- **Excel**: excelize v2 用于导入导出
- **现有导入流程**: `excel_service.go` → `excel_config.go`（列映射）→ `reference_resolver.go`（名称→UUID）→ `batch_upserter.go`（批量写入）
- **仪表盘**: 自定义 Widget 系统，支持 StatCard、Chart、Table、List、Progress 类型
- **两套仪表盘**:
  - `/monitor/dashboard` — 固定监控仪表盘（保留不变）
  - `/dashboard-system` — 可配置仪表盘（v1.2 改进目标）
- **批量导出**: ZIP 打包多个 Excel 文件，支持 9 个实体类型

### Current state (2026-06-16)

- **Shipped**: v1.0 through v1.15 (and Phase 22 VDI 22A/22B partial for v1.11) —
  18 phases, 133 plans completed, ≈85% of total plan count.
- **Phases 30 / 31 / 32 / 33 / 34**: unlabeled (no v1.x tag in ROADMAP
  milestones list) but fully shipped. Phase 34 included a full code
  review with 1 critical security finding (sensitiveKeys did not
  match camelCase field names) and 7 atomic fix commits.
- **Codebase size**: 485 Go files (backend), 506 TS/TSX files
  (frontend) per `CLAUDE.md`.
- **Test coverage**: regression_test.go for operlog locks 25 OperType
  constants, 18 mandatory sensitive keywords, Record signature, keyword
  stability. 14 subtests in TestFilterSensitiveParams + 5 edge cases.
- **Build status** (post-Phase 34 review closure): `go build ./...`
  clean, full operlog test suite PASS, system test suite PASS.

### Known technical debt (post-v1.15)

- VDI 22C (账号管理) and 22D (Web Console) — listed in ROADMAP as
  v1.12 sub-phases but not started. Blocked on 22A/22B production
  observability data.
- Phase 25 (VM 数据范围权限) — depends on Phase 22, also not started.
- `operlog_e2e_verify.sh` live-DB section gated on `SKIP_E2E=1` by
  default — should be promoted to a CI gate.
- `regression_test.go:146` — `for i := 0; i < 5; i++` can be
  modernized to `for i := range 5` (Go 1.22+). Cosmetic, single line.
- 103 open items in `gsd-sdk query audit-open` (most are stale
  `root_cause_found` debug sessions from May 2026).

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 用户关联字段: 在 workstation 表添加 `user_id` 列 | 与现有 `org_id` 模式一致，一对一关联 | ✓ Good |
| 用户匹配方式: 通过 `user_name` 精确匹配 | 用户名唯一且稳定 | ✓ Good |
| 部门匹配方式: 通过 `dept_name` 精确匹配 | 与现有 `reference_resolver` 模式一致 | ✓ Good |
| 匹配失败策略: 留空不阻断 | 用户明确要求可选关联 | ✓ Good |
| Device Reference: `sys_network_device.device_name` | 设备名是用户可见的标识字段 | ✓ Good |
| Port Reference: `sys_device_port_status.interface_name` | 接口名是网络设备端口的唯一标识 | ✓ Good |
| Port 无 DependsOn: 全局查找不限设备 | 用户未要求级联验证 | ✓ Good |
| 配置驱动模式: 只改 excel_config.go + excel_service.go | 最小改动，复用现有管道 | ✓ Good |
| 保留两套仪表盘: 固定 + 可配置 | 用户明确要求保留固定监控仪表盘 | ✓ Good |
| SNMP panic 修复: 三层防护（恢复+并发+验证） | 防御式编程，确保稳定性 | ✓ Good |
| Core 清理: 文档化而非删除 | 大多数字段有外部依赖 | ✓ Good |
| TDD 方法: RED→GREEN→REFACTOR | 确保测试覆盖和安全修复质量 | ✓ Good |
| 批量导出: ZIP 格式打包多个 Excel | 用户需求，简化下载流程 | ✓ Good |
| VDI 集成: 拆分 22A→22D 4 子阶段 (P1) | 渐进式集成降低风险 | ✓ Good (22A/22B done; 22C/22D pending) |
| operlog 共享包: 提取到 `internal/utils/operlog` (P34) | 避免 handler module 间的循环依赖 | ✓ Good |
| OperType 25 常量集 (P34) | 锁定审计语义，不再用 OperTypeOther 兜底 | ✓ Good |
| OperTypeUnlock = 24 新增 (P34 review) | 解锁是独立审计动作 | ✓ Good |
| sensitiveKeys 34 条 camelCase/snake_case 全覆盖 (P34 review) | 修复 substring-search 不匹配 camelCase 字段的 critical bug | ✓ Good (CR-001) |
| 工位设备关联子表格: Ant Design Table expandable (P28) | 复用现有 Table 组件 | ✓ Good |
| Vendor 命令模板硬编码 Go map (v1.19 W1) | 落地为先，后续 phase 抽象 DB | ✓ Good |
| Path C audit↔operlog 关联 (v1.19 W3) | handler 先 INSERT audit → operlog.Record(WithOperParam(audit_ids)); audit.oper_log_id 列保持 NULL；不入 operlog 包 | ✓ Good |
| detached 30min context (v1.19 W2) | batch 入口 `context.WithTimeout(context.Background(), 30*time.Minute)` 规避 `Core.Close()` 30s 截止 | ✓ Good |
| menu_grant_helpers.go 精准授权 (v1.19 W3) | `INSERT INTO sys_role_menu SELECT...WHERE m.menu_name=parent...ON CONFLICT DO NOTHING` 解决 antd 父子联动陷阱 | ✓ Good (memory `migration-grant-new-menu-precision-helper`) |
| BATCH-05 indeterminate Spin MVP (v1.19 W4 D-05) | 诚实进度，不伪造 X/Y；批量同步阻塞无后端推送 | ⚠ Revisit (FUTURE-11 SSE/WS) |
| execSinglePort DRY helper (v1.19 W3) | 5 单端口 handler 公共流程合并；operlog.Record 物理调用点 6→2，语义等价 | ✓ Good |
| eslint-disable placement 教训 (v1.19 W4 Phase 55) | directive 紧贴 `}, []);` 而非 `useEffect(` 开头 | — Pending (Pattern doc) |
| 双框修根因而非遮盖 (v1.24) | 用户裁定「修根因 + CSS 一致性审计」；审计全局 `!important` 与 antd 6 结构冲突面，不做一处覆盖了事 | ✓ Good (双框 3→0，登记册 36 条) |
| CSS 复位块不写 box-shadow (v1.24) | 焦点环归 wrapper（antd 内层 `box-shadow:none!important` 特异性 0,4,1 自足）；写 box-shadow 会与焦点环叠加出双圈 | ✓ Good (焦点环单圈实测确认) |
| `:not(组合子)` 禁用 (v1.24) | CSS3 下整条规则静默丢弃 → 必须用 `.ant-input-affix-wrapper > input.ant-input` | ✓ Good |
| #16 双病灶同时修 (v1.24) | 组级 ANY-of + 模块级降级叠加；只藏前端按钮 = 假修复（API 直调仍可写） | ✓ Good (DB 前后对照 8/8 翻转) |
| gin Group 继承父组中间件链 → 写组必须是兄弟组 (v1.24) | 写组若挂在读组之下，读组 ANY-of 仍放行写操作，病灶① 不会被修掉 | ✓ Good (实施方识破 Lead 规格中的此错) |
| 三态门控禁用 `permissions.length===0` 判未就绪 (v1.24) | 空权限集与未加载完在数据上不可区分 → 接口失败时管理员也看不到按钮且无提示 | ✓ Good (组件测试 A6-3 证明两者可区分) |
| hydrate 路径须同步置 permissionsLoaded (v1.24) | 直接 setState 绕过 setPermissions → 硬刷新后所有操作按钮在 API 返回前一直 disabled | ✓ Good (Phase 83 实测发现根因) |
| CSS status class 用 antd 6 真实名 (v1.24) | antd 把 status class 打在 wrapper 自身（`ant-input-status-error`）；`ant-input-affix-wrapper-status-error` 在 antd 6 不存在 = 永不命中的死规则 | ✓ Good (契约检查升级为双向断言) |
| 契约检查须校验「能命中」而非「规则存在」 (v1.24) | Phase 76 的 C5 校验的正是死规则 → 检查通过而规则从未生效 | ✓ Good (Lesson) |

## Constraints

- **Tech Stack**: 必须沿用现有架构模式（Handler-Service、opsApi、excel_config）
- **Backward Compatible**: 新字段可选，不影响现有数据和导入流程
- **UUID Foreign Keys**: 关联字段必须是有效 UUID
- **Status Convention**: 遵循 0=正常, 1=停用 的惯例
- **Response Format**: 使用 `response.Success()` / `response.Error()` 包装
- **Widget 数据获取**: 必须支持权限过滤，用户只能访问有权限的 API 端点
- **保留固定仪表盘**: `/monitor/dashboard` 保持不变，仅改进 `/dashboard-system`
- **SNMP 稳定性**: 必须保证高并发场景下不崩溃
- **测试覆盖**: 安全修复必须有单元测试验证

## Evolution

This document evolves at phase transitions and milestone boundaries.

## Validated — v1.0 through v1.15 (summary)

The full Validated section listing all shipped requirements is preserved
in `.planning/REQUIREMENTS.md` history. High-level milestones:

- ✓ v1.0 工位导入部门/用户关联 (2026-04-16) — 2 phases, 7 plans
- ✓ v1.1 信息点导入设备端口关联 (2026-04-16) — 1 phase, 1 plan
- ✓ v1.2 可配置仪表盘生产级改造 (2026-04-21) — 4 phases, 11 plans
- ✓ v1.3 技术债清理 (2026-04-27) — 3 phases, 9 plans
- ✓ v1.4 MAC地址采集优化 (2026-05-09) — 1 phase, 4 plans
- ✓ v1.5 MAC地址历史数据管理 (2026-06-15) — 4 phases, 26 plans (REQ archive: `.planning/milestones/v1.5-REQUIREMENTS.md`)
- ✓ v1.6 API密钥管理系统 (2026-05-19) — 1 phase, 10 plans
- ✓ v1.7 前后端加密配置同步 (2026-05-20) — 1 phase, 6 plans
- ✓ v1.8 登录端点加密增强 (2026-05-21) — 1 phase, 4 plans
- ✓ v1.9 AD域控集成扩展 (2026-05-24) — 2 phases, 11 plans
- ✓ v1.10 网络设备权限修复 (2026-05-24) — 1 phase, 1 plan
- ✓ v1.11 深信服桌面云集成 (VDI 22A+22B partial) (2026-06-02) — 1 phase (22), 6 plans
- ✓ v1.13 资产管理模块 (Phase 26) (2026-06-08) — 1 phase, 6 plans
- ✓ v1.14 全局列自定义 (Phase 27) (2026-06-09) — 1 phase, 1 plan
- ✓ v1.15 工位设备关联 (Phase 28) (2026-06-10) — 1 phase, 4 plans
- ✓ Phase 30 前端性能优化 (unlabeled, 2026-06-13) — 5 plans
- ✓ Phase 31 P0 收尾 (2026-06-13) — 5 plans
- ✓ Phase 32 v1.14 P1/P2 (unlabeled, 2026-06-13) — 7 waves, 20 commits
- ✓ Phase 33 Vercel React Best Practices (unlabeled, 2026-06-13) — 4 waves
- ✓ Phase 34 操作日志全模块集成 + review (2026-06-16) — 11 plans + 7 review-fix commits
- ✓ Phase 39 工位部门物理位置映射 (workstation dept-location alias) (2026-06-25) — 8 plans + UAT 修复 commits
- ✓ Phase 44 R3 IP 例外规则引擎 (v1.17 资产对账) (2026-06-28) — 2 plans, CIDR GiST 引擎 + Layer 3.5 拦截 + 例外 CRUD/命中测试 + cron 过期清理 + 转单 no_workorder 过滤 + 降噪基线 + Excel 导入导出; VERIFICATION 10/10 passed, 1 CR-03 gap-closure fix (baseline 路由权限), 4 硬化债 BLOCKER 跟踪

---
## Past Milestone: v1.20 网络设备 VLAN + 端口绑定 (Network Device VLAN + Port Binding) — ✅ SHIPPED 2026-07-10（历史归档）

**Goal:** 在 v1.19 端口写命令 MVP 基础上扩展 2 个新写命令：① 修改 access VLAN ② IP+MAC+Port 静态绑定。复用 v1.19 vendor 模板 / operlog / 权限 / 批量 / e2e 全套基建，最小化新基建引入。

**Status:** SHIPPED (Phase 56 / 5 plans / 5 waves). 45 commits / 51 files / +10,961/-104 LOC in 2 days.

**Delivered:**
- 3 vendors × 2 actions 全覆盖（Huawei VRP / H3C Comware V7 / Ruijie RGOS 11.4）
- 2 new kebab HTTP 端点: `POST /network/ports/write/set-access-vlan` + `/port-binding`
- 4 sentinel validators (VLAN 1-4094 range / bind op add|remove / IPv4 regex / null-MAC reject)
- PortResult.Extra map 携带 audit after_value
- 2 前端 Modal + 2 networkApi wrapper + types/network.ts 扩展
- 11 new TestE2E_* via scrapligo FileTransport + 10 new fixtures
- vendor-react bundle gzip 774.96 kB (≤776 kB baseline, zero regression)
- Code review clean after fixing 4 CRITICAL (CR-01/04 batch path validator 共享、CR-02 vlan field、CR-03 modal pre-fill)

**Locked decisions (validated):**
- 复用 v1.19 vendor template 模式 + 新增 2 actions (set_access_vlan + port_binding) ✓
- 权限：单一 `network:port:write` 覆盖新 2 actions ✓
- Audit：`sys_port_write_audit.after_value` JSONB 写 PortResult.Extra ✓
- operlog OperType: VLAN → Update (=2); binding add → Create (=1); binding remove → Delete (=3) ✓
- Pre-state check: set_access_vlan 读 PVID NoOp 短路; port_binding 不 NoOp（Pitfall 6） ✓
- MAC 格式锁定：Huawei/H3C = `AA-BB-CC-DD-EE-FF` (per-byte hyphenated); Ruijie = `aabb.ccdd.eeff` (Cisco 3-pair) ✓
- IPv4 regex 仅做 shell-injection 防御，不验证 segment range（RFC 0.0.0.0/255.255.255.255 合法） ✓

**Deferred (acknowledged at close):**
- **12 site-visit UAT items** (Huawei S8700 × 6 + Ruijie RS8607E × 4 + H3C × 2 conditional) — 详见 `.planning/phases/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-HUMAN-UAT.md`，沿用 v1.19/v1.18 deferral 模式
- **VLAN-04 / BIND-06 / UI-06** (批量端口写) — 显式 defer 到 FUTURE-BATCH-05；`BatchWriteRequest` 已预留 4 字段（VLANID/BindOp/IPAddress/MACAddress）

**Archived:** `.planning/milestones/v1.20-ROADMAP.md` + `.planning/milestones/v1.20-REQUIREMENTS.md`
- **FUTURE-03 剩余部分**: trunk VLAN / hybrid VLAN / 多 VLAN list
- **FUTURE-04~14**: dry-run / 操作历史查看 UI / 批量冲突自动解决 / 设备不可达预检 / Cisco IOS-XE/XR / 自动回滚 / 多用户并发互斥 / HTTP handler e2e / 3-vendor fixture 全覆盖 / 跨固件命令差异 / BATCH-05 实时进度

**设计文档:** `docs/plans/2026-07-09-v1.20.1-design.md` (commit `f799cecc`，412 行)

**用户已确认数据来源:**
- Huawei S8700 `dis port vlan ge 1/0/0` / `dis user-bind static all` (实采样例)
- Ruijie RS8607E `sh int status` / `sh port-security binding` (实采样例)
- H3C: 复用 Huawei 模板结构 (per user direction)

---

*Last updated: 2026-11-08 after v1.53 milestone SHIPPED & ARCHIVED（开放项全量清偿，8/8 Phase、18 REQ 交付 + 6 项人工验证跟踪、21 条开放项全覆盖；全量回归全绿；GSD `milestone complete` 完成归档与权威交回，产物见 `.planning/milestones/v1.53-*`；路径映射见 `.planning/ARCHIVE-INDEX.md`）*
