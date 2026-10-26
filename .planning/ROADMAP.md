---
last_updated: 2026-10-23
update_trigger: v1.22 前端公用组件整治 roadmap CREATED (Phases 63-68, 20/20 REQ coverage)
previous_update: 2026-10-22 v1.21 SHIPPED & ARCHIVED
---

# Roadmap: RuoYi-Next 运维管理系统

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)
- ✅ **v1.4 MAC地址采集优化** — Phase 11 (shipped 2026-05-09)
- ✅ **v1.5 MAC地址历史数据管理** — Phases 12-15 (shipped 2026-06-15; Phase 13 gap closure retroactive 2026-06-26)
- ✅ **v1.6 API密钥管理系统** — Phase 16 (shipped 2026-05-19)
- ✅ **v1.7 前后端加密配置同步** — Phase 17 (shipped 2026-05-20)
- ✅ **v1.8 登录端点加密增强** — Phase 18 (shipped 2026-05-21)
- ✅ **v1.9 AD域控集成扩展** — Phases 19-20 (shipped 2026-05-24)
- ✅ **v1.10 网络设备权限修复** — Phase 21 (shipped 2026-05-24)
- ✅ **v1.11 AD组自动同步系统** — Phase 23 (shipped 2026-05-26)
- ✅ **v1.13 资产管理模块** — Phase 26 (shipped 2026-06-08)
- ✅ **v1.14 全局列自定义** — Phase 27 (shipped 2026-06-09)
- ✅ **v1.15 工位设备关联** — Phase 28 (shipped 2026-06-10)
- ✅ **v1.12 深信服桌面云集成 (22A+22B)** — Phases 22A/22B (shipped 2026-06-02)
- ✅ **v1.15 部门物理位置映射** — Phase 39 (shipped 2026-06-25)
- ✅ **v1.16 技术债清理 (Tech-Debt Cleanup)** — Phases 40-41 (shipped 2026-06-26)
- ✅ **v1.17 资产对账 (Asset Reconciliation)** — Phases 42-46 (shipped 2026-07-03) — see [milestones/v1.17-ROADMAP.md](milestones/v1.17-ROADMAP.md)
- ✅ **v1.18 网络设备硬件清单 (Device Component Serials)** — Phase 48 (shipped 2026-07-04) — see [milestones/v1.18-ROADMAP.md](milestones/v1.18-ROADMAP.md)
- ✅ **v1.19 网络设备写命令 (Network Device Port Write Operations)** — Phases 50-55 (shipped 2026-07-08) — see [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md) | [audit](milestones/v1.19-MILESTONE-AUDIT.md)
- ✅ **v1.20 网络设备 VLAN + 端口绑定 (Network Device VLAN + Port Binding)** — Phase 56 (shipped 2026-07-10) — see [milestones/v1.20-ROADMAP.md](milestones/v1.20-ROADMAP.md)
- ✅ **v1.21 AD 双向同步 bug 修复 (AD Bidirectional Sync Bug Fixes)** — Phases 57-62 (shipped 2026-10-22) — see [milestones/v1.21-ROADMAP.md](milestones/v1.21-ROADMAP.md) | [audit](milestones/v1.21-MILESTONE-AUDIT.md) | [requirements](milestones/v1.21-REQUIREMENTS.md)
- 🚧 **v1.22 前端公用组件整治 (Frontend Shared Component Remediation)** — Phases 63-68 (planning)

---

## v1.22 前端公用组件整治 — 🚧 In Progress

**Milestone goal:** 消化 2026-10-23 全量前端组件审查的 201 条问题（P0 24 / P1 87 / P2 90，事实基线：`docs/2026-10-23-前端公用组件审查与最佳实践方案.md` + `.planning/notes/frontend-audit/{01..04}*.md`），把公共层从"只有零件没有契约"补成"有契约"：默认值下沉（allowClear/showSearch）、分页契约下沉（showTotal/setTotal）、危险操作唯一入口（ConfirmDelete），使抽出的组件真正被用起来。

**抽取原则（locked）:** 抽契约不抽业务（资产选择器/部门树选择器等业务语义组件不抽）；`useDebouncedSearch` 并入 RemoteSelect 不单独抽；巨型文件拆分随 COMP-05 迁移顺带、不单独立项；默认值必须下沉封装层；ConfirmDelete 是危险操作唯一入口。

## Phases

- [ ] **Phase 63: P0 止血 — 数据可达与安全** — 修 showTotal 分页根因 / total 恒 0 / OU 截断 / 双击重复提交 / 批量删除无确认 / 假功能入口 / fixed 列失效（STOP-01..07）
- [ ] **Phase 64: P1 修复 — 反馈与可访问性** — 表格 loading / Modal 遮罩脏检查 / 本地排序误用 / console 错误出口收敛 / aria 可访问名（FIX-01..05）
- [ ] **Phase 65: 基础契约组件 — 默认值与危险操作唯一入口** — AppInput/AppSelect 默认值下沉 + ConfirmDelete 统一危险删除入口（COMP-01, COMP-02）
- [ ] **Phase 66: 高级契约组件 — PageTable 与 FormModal** — 列表/表单弹窗契约内建骨架，依赖 Phase 65 组件（COMP-03, COMP-04）
- [ ] **Phase 67: 存量迁移 — 按 ROI 顺序落地契约组件** — C1 试点 discoveries → backups → devices/cache 巨型页，每页一个 PR（COMP-05）
- [ ] **Phase 68: P2 一致性债务 — 颜色/日期/类型收敛** — design token 收敛 hex 硬编码 + 日期/状态 helper + any 类型消除（DEBT-01..03）
- [ ] **Phase 68.1: 收尾清偿 — 危险操作收口 / 契约 minor 补课 / 豁免收缩 / 存量迁移收尾** — 清偿 v1.22 审计登记的四组技术债，使里程碑以无债状态收档（DEBT-04）

## Phase Details

### Phase 63: P0 止血 — 数据可达与安全

**Goal**: 用户在所有列表页能看到真实的数据总量并翻到任意一页；双击确认不会重复提交；批量删除必有二次确认；假功能入口不再出现；宽表滚动时固定操作列保持可见。本 phase 全部为小根因修复（每项 ~0.5-1 天），ROI 最高，独立于组件抽取先行落地。
**Depends on**: Nothing (v1.22 首个 phase)
**Requirements**: STOP-01, STOP-02, STOP-03, STOP-04, STOP-05, STOP-06, STOP-07
**Notes**: 事实基线 `.planning/notes/frontend-audit/`（201 条，精确到 文件:行号）。STOP-01 根因 `src/hooks/usePagination.ts:122` showTotal 只输出页数（51 页受害）；STOP-02 根因 `src/pages/network/backups/index.tsx:79`、`src/pages/network/discoveries/index.tsx:34` 的 `_setTotal` 从未调用；STOP-03 根因 `src/pages/ad-domain/ous/index.tsx:130` 固定 `pageSize:100` 前端分页截断。
**Success Criteria** (what must be TRUE):
  1. 用户在任意分页列表页同时看到总条数与页数（如"共 128 条 / 13 页"），total=0 时不显示"共 1 页"（STOP-01）
  2. 用户能翻到 backups / discoveries 列表的任意一页，每页数据与总数一致（STOP-02）
  3. 用户能浏览 AD 域 OU 列表第 3 页及以后的全部数据，不再被 100 条截断（STOP-03）
  4. 用户快速双击任意业务弹窗确认按钮，后端只收到一次请求、不产生重复记录（STOP-04）
  5. 用户触发批量删除前必见二次确认弹窗；点击 3 处假功能入口（assets 空确认 / ReconciliationDrawer 空 Tab）能看到真实功能或入口已移除；宽表横向滚动时固定操作列始终贴右可见（STOP-05/06/07）

**Plans**: TBD
**UI hint**: yes

### Phase 64: P1 修复 — 反馈与可访问性

**Goal**: 系统在加载、出错、误操作时给用户明确反馈而非静默——表格有 loading、误点遮罩不丢表单、排序结果正确、失败有可见错误提示、读屏器用户能识别图标按钮。
**Depends on**: Phase 63（同属存量页修补，收敛节奏一致；无硬数据依赖）
**Requirements**: FIX-01, FIX-02, FIX-03, FIX-04, FIX-05
**Notes**: FIX-04 的 `console.*` 52 处收敛应建立统一错误出口（toast/errorHandler），该出口在 Phase 65+ 被契约组件复用。FIX-02 Modal `maskClosable` 0/22 全部可点即关，改受控 + 脏检查——该契约随后下沉到 Phase 66 FormModal。
**Success Criteria** (what must be TRUE):
  1. 用户打开 network/monitor 域任意列表页时，数据加载期间表格显示 loading 状态而非空白（FIX-01）
  2. 用户在含未保存修改的表单弹窗中误点遮罩时，弹窗不关闭或先提示确认，未保存内容不丢失（FIX-02）
  3. 用户在 cache/job 等分页表上按列排序时得到全量正确结果（服务端排序或明确的降级说明）（FIX-03）
  4. 用户请求失败时看到可见的错误提示（toast/message），`console.*` 静默吞错收敛到统一错误出口（FIX-04）
  5. 使用读屏器的用户能通过可访问名识别每个图标按钮的功能（aria-label/role 补齐）（FIX-05）

**Plans**: TBD
**UI hint**: yes

### Phase 65: 基础契约组件 — 默认值与危险操作唯一入口

**Goal**: 新页面使用 AppInput/AppSelect 零配置获得可清除/可搜索行为；所有危险删除操作统一经 ConfirmDelete 唯一入口强制二次确认——把"逐页补 allowClear 不可维护"的根因（285 Input 仅 23% allowClear、400 Select 仅 30%）通过默认值下沉封装层解决。
**Depends on**: Nothing 硬依赖（可与 Phase 63/64 并行评估；按 roadmap 顺序置于其后）
**Requirements**: COMP-01, COMP-02
**Notes**: 抽契约不抽业务（locked）。`useDebouncedSearch` 并入 RemoteSelect，不单独抽。本 phase 产物是 Phase 66 PageTable/FormModal 的直接前置。
**Success Criteria** (what must be TRUE):
  1. 新页面使用 AppInput/AppSelect 时无需逐个配置即默认带 allowClear（Select 另含 showSearch），行为与全站一致（COMP-01）
  2. 开发者在封装层外的业务代码中直接写裸 `<Input>`/`<Select>` 会被 lint/约定拦截或有明确豁免清单（COMP-01）
  3. 用户点击任意危险删除按钮，都经过同一个 ConfirmDelete 组件的二次确认，不存在绕过入口的裸 Popconfirm/直接 delete 调用（COMP-02）
  4. ConfirmDelete 的确认文案统一包含操作对象与不可恢复警示，删除失败时有可见错误反馈（COMP-02）

**Plans**: TBD
**UI hint**: yes

### Phase 66: 高级契约组件 — PageTable 与 FormModal

**Goal**: 新列表页使用 PageTable 一站式骨架（分页/loading/空态/错误态契约内建，模板页样板 ~450 行 → ~120 行）；新表单弹窗使用 FormModal（confirmLoading/脏检查/destroyOnHidden 契约内建）——把 Phase 63/64 修出的 P0/P1 行为固化为组件默认契约，新代码不再可能回退。
**Depends on**: Phase 65（PageTable 内建分页契约依赖 AppSelect 等基础组件；FormModal 脏检查契约承接 FIX-02）
**Requirements**: COMP-03, COMP-04
**Notes**: 三态组件（Empty/Skeleton/ErrorAlert）随 PageTable 间接覆盖（REQUIREMENTS deferred 项）。FIX-01 loading / STOP-04 confirmLoading / FIX-02 脏检查在本 phase 完成契约化。
**Success Criteria** (what must be TRUE):
  1. 新列表页接入 PageTable 后，分页（含 showTotal）/loading/空态/错误态无需逐页实现即全部具备（COMP-03）
  2. PageTable 模板页代码量显著下降（样板 ~450 行 → ~120 行量级）， SearchForm 模式并入 PageTable 体系（COMP-03）
  3. 新表单弹窗接入 FormModal 后，confirmLoading 防重复提交、脏检查拦误关、destroyOnHidden 重置三项默认生效（COMP-04）
  4. 使用者可在不读源码的情况下按组件 README/类型签名正确接入 PageTable 与 FormModal（契约自描述）（COMP-03/04）

**Plans**: TBD
**UI hint**: yes

### Phase 67: 存量迁移 — 按 ROI 顺序落地契约组件

**Goal**: 存量页面按 ROI 顺序迁移到契约组件体系，使新契约真实吃掉存量样板而非只服务新代码——C1 试点 discoveries 最小页 → backups → devices/cache 巨型页（随拆分迁移），每页一个 PR，旧实现保留一个迭代。
**Depends on**: Phase 66（迁移目标 PageTable/FormModal 必须先就位）
**Requirements**: COMP-05
**Notes**: 巨型文件拆分随迁移顺带（locked，不单独立项）。SearchForm 并入 PageTable 体系。每页一个 PR 便于回滚与 review。
**Success Criteria** (what must be TRUE):
  1. discoveries 页作为 C1 试点完成迁移，用户视角行为不回归（分页/删除确认/表单弹窗均走新契约）（COMP-05）
  2. backups 页完成迁移，Phase 63 修的 total/setTotal 行为在 PageTable 契约下保持（COMP-05）
  3. devices/cache 巨型页随拆分完成迁移（或完成拆分前置并明确迁移计划），迁移过程零行为回归（COMP-05）
  4. 每个迁移页独立 PR 可回滚，旧实现保留一个迭代后删除，样板行数下降可量化（COMP-05）

**Plans**: TBD
**UI hint**: yes

### Phase 68: P2 一致性债务 — 颜色/日期/类型收敛

**Goal**: 视觉与数据展示全站一致：颜色统一走 design token（109 处 6 位 hex / 25 文件，含 3 位简写共 125 行命中收敛）；日期/状态展示统一 helper（裸三元 13 处/6 文件 + 手写日期格式 12 处/8 文件）；`: any` / `as any` 27 处/16 文件随 FormModal 迁移逐步消除。
**Depends on**: Phase 66/67（any 消除挂在 FormModal 迁移路径上；helper 收敛在迁移后的页面上做最省）
**Requirements**: DEBT-01, DEBT-02, DEBT-03
**Success Criteria** (what must be TRUE):
  1. 用户在 light/dark 主题下看到的页面颜色一致来自 design token，硬编码 hex 收敛到 token（残留有明确豁免清单）（DEBT-01）
  2. 全站列表中的状态标签与日期展示格式统一（同一 helper 渲染），不再出现同一状态两种文案/同一日期两种格式（DEBT-02）
  3. 前端 `: any` / `as any` 收敛至目标基线（27 处 → 随迁移消除或有记录的豁免），`npm run type-check` 通过（DEBT-03）
  4. `npm run build` 通过且 vendor-react gzip 包体积不高于迁移前基线（零回归守卫，沿用 v1.19 起 776 kB 基线惯例）

**Plans**: TBD
**UI hint**: yes

---

### Phase 68.1: 收尾清偿 — 危险操作收口 / 契约 minor 补课 / 豁免收缩 / 存量迁移收尾

**Goal**: 清偿 v1.22 里程碑审计（.planning/v1.22-MILESTONE-AUDIT.md）登记的四组技术债，让 v1.22 以无债状态收档：①45 处未迁移的批量删除统一到 ConfirmDelete 唯一入口（危险操作必二次确认）；②Phase 63-68 验证报告登记的 7 项契约 minor 补课；③P65（140 项）+ P66（41 项）存量豁免清单逐项清偿收缩；④剩余存量列表页（dedicated-lines 等）完成契约组件迁移。
**Depends on**: Phase 68（豁免清单与契约组件均已就位）
**Requirements**: DEBT-04
**Success Criteria** (what must be TRUE):
  1. 全站批量删除入口只剩 ConfirmDelete 一条路径，源码中不再存在裸 Modal.confirm 删除（豁免项须逐条说明原因）（DEBT-04）
  2. 7 项契约 minor（PageTable 分页口径双点同步、FormModal 不透传 maskClosable/keyboard 等）已修复或显式记录为设计决策（DEBT-04）
  3. P65/P66 豁免清单条目数显著收缩，剩余每项都有可核查的豁免理由（DEBT-04）
  4. dedicated-lines 等剩余存量列表页完成契约组件迁移，样板行数下降可量化（DEBT-04）
  5. type-check 0 / vitest 全绿 / lint errors 不高于基线，npm run build 通过（DEBT-04）

**Plans**: TBD
**UI hint**: yes

---

### Progress Table (v1.22)

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 63. P0 止血 — 数据可达与安全 | 1/1 | ✅ Complete | 2026-10-25 |
| 64. P1 修复 — 反馈与可访问性 | 1/1 | ✅ Complete | 2026-10-25 |
| 65. 基础契约组件 — 默认值与危险操作唯一入口 | 1/1 | ✅ Complete | 2026-10-25 |
| 66. 高级契约组件 — PageTable 与 FormModal | 1/1 | ✅ Complete | 2026-10-25 |
| 67. 存量迁移 — 按 ROI 顺序落地契约组件 | 1/1 | ✅ Complete | 2026-10-26 |
| 68. P2 一致性债务 — 颜色/日期/类型收敛 | 1/1 | ✅ Complete | 2026-10-26 |
| 68.1 收尾清偿 — 危险操作收口 / 契约 minor 补课 / 豁免收缩 / 存量迁移收尾 | 0/TBD | Not started | - |

### Coverage Map (v1.22)

```
STOP-01..07 → Phase 63 (7)
FIX-01..05  → Phase 64 (5)
COMP-01, COMP-02 → Phase 65 (2)
COMP-03, COMP-04 → Phase 66 (2)
COMP-05     → Phase 67 (1)
DEBT-01..03 → Phase 68 (3)
DEBT-04     → Phase 68.1 (1)

Mapped: 20/20 ✓  No orphans, no duplicates
```

---

## Archive: Shipped Milestones

<details>
<summary>✅ v1.21 AD 双向同步 bug 修复 (Phases 57-62) — SHIPPED 2026-10-22</summary>

### Phases (v1.21) — 6 phases / 22 plans / 45 tasks（SHIPPED 2026-10-22）

**Milestone goal:** 修复 AD↔系统双向同步的 2 个 P0 bug（部门回退乒乓、软删用户接管风险）、3 个 P1 数据一致性缺口和 P2 卫生项，以三方审计修订后的 outbox-lite 架构落地，消除"改了部门又弹回"与静默数据丢失。

- [x] **Phase 57: AD 推送队列基建** — outbox-lite 队列表 + 同事务入队 + 按实体合并 + 删除入队 disable (QUEUE-01/02/06, HYGIENE-06) — completed 2026-10-16 (3/3 plans, code review F-01~F-06 修复 + verifier GAP-01/02/03 闭环, [VERIFICATION](milestones/v1.21-phases/57-ad-outbox-lite/57-VERIFICATION.md) human_needed: 3 项真环境人工验证在 [57-HUMAN-UAT.md](milestones/v1.21-phases/57-ad-outbox-lite/57-HUMAN-UAT.md))
- [x] **Phase 58: 排水器与 compare-and-move 推送** — in-process cron 排水 + 退避重试 + 幂等 OU 移动 + 配置路由无歧义 (HYGIENE-01/02, QUEUE-03/04/05) — 7/7 plans，completed 2026-10-19（SC1~6 全部 VERIFIED，见 [58-VERIFICATION.md](milestones/v1.21-phases/58-compare-and-move/58-VERIFICATION.md)）
- [x] **Phase 59: 登录回写守卫与部门对账视图** — AD 登录不再回滚本地编辑 + dept↔OU 不一致可观测 (GUARD-01/02) — 4/4 plans，completed 2026-10-19（两条 Manual-Only 视觉断言 + PG 真实启动冒烟归 [59-VALIDATION.md](milestones/v1.21-phases/59-ad-login-guard-reconciliation/59-VALIDATION.md)）
- [x] **Phase 60: 软删恢复安全** — ad_dn 校验后恢复 + 登录/批量同步共用分类逻辑 (SAFE-01/02) — 2/2 plans，completed 2026-10-20（12/12 must-haves VERIFIED，见 [60-VERIFICATION.md](milestones/v1.21-phases/60-soft-delete-recovery-safety/60-VERIFICATION.md) human_needed: 3 项真环境人工验证在 [60-UAT.md](milestones/v1.21-phases/60-soft-delete-recovery-safety/60-UAT.md)）
- [x] **Phase 61: 影子表完整性** — 删除清理两道闸门 + 组成员事务化 + singleflight 挂载 (SHADOW-01/02/03) — 3/3 plans，**Complete 2026-10-22**（8/8 must-haves VERIFIED，61-UAT 4/4 pass；真环境人工项见 [61-UAT.md](milestones/v1.21-phases/61-shadow-table-integrity/61-UAT.md) 与 [61-VERIFICATION.md](milestones/v1.21-phases/61-shadow-table-integrity/61-VERIFICATION.md) status=passed）
- [x] **Phase 62: 卫生收尾与死代码清理** — F-22 旧路径删除 + 嵌套组标记 + 同名部门 OU DN 冲突告警 (HYGIENE-03/04/05) — 3/3 plans，completed 2026-10-22（18/18 must-haves VERIFIED，HYGIENE-03 死代码 grep 残留=0 / HYGIENE-04 嵌套组过滤零 DDL + 生产 8 行污染自愈 / HYGIENE-05 OU DN 冲突 typed error 显式化）

### Progress Table (v1.21)

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 57. AD 推送队列基建 | 3/3 | Complete | 2026-10-16 |
| 58. 排水器与 compare-and-move 推送 | 7/7 | Complete（2026-10-22 全收口） | 2026-10-22 |
| 59. 登录回写守卫与部门对账视图 | 4/4 | Complete（verification passed） | 2026-10-22 |
| 60. 软删恢复安全 | 2/2 | Complete（verification passed） | 2026-10-22 |
| 61. 影子表完整性 | 3/3 | Complete（VERIFICATION status=passed） | 2026-10-22 |
| 62. 卫生收尾与死代码清理 | 3/3 | Complete（VERIFICATION status=passed） | 2026-10-22 |

</details>

<details>
<summary>✅ v1.20 网络设备 VLAN + 端口绑定 (Phase 56) — SHIPPED 2026-07-10</summary>


### Phase 56: 网络设备 VLAN + 端口绑定 (v1.20.1) — 5/5 plans, shipped 2026-07-10

**Goal:** 在 v1.19 端口写命令 MVP 基础上扩展 2 个新写命令：① 修改 access VLAN ② IP+MAC+Port 静态绑定。复用 v1.19 vendor 模板 / operlog / 权限 / 批量 / e2e 全套基建。

**Requirements**: 22 (VLAN-01~06, BIND-01~07, INFRA-01~04, UI-01~06, TEST-01~05)

**Plans**: 5 plans (5 waves)

- [x] 56-01-PLAN.md — vendor 模板新增 (set_access_vlan + port_binding × 3 vendors) + 12+ 单元测试
- [x] 56-02-PLAN.md — pre_state_check + port_write_service 新增 2 方法 + 4 校验器 + audit JSONB
- [x] 56-03-PLAN.md — 2 kebab 端点 + 2 handler + operlog OperType 分流 + `network:port:write` 权限复用
- [x] 56-04-PLAN.md — 2 types + 2 networkApi wrappers + SetAccessVlanModal + PortBindingModal + ports/index.tsx
- [x] 56-05-PLAN.md — 11 TestE2E_* + 10 fixtures + 56-HUMAN-UAT.md + 文档同步

45 commits / 51 files / +10,961/-104 LOC。详见 [milestones/v1.20-ROADMAP.md](milestones/v1.20-ROADMAP.md)。

</details>

<details>
<summary>✅ v1.19 网络设备写命令 (Phases 50-54) — SHIPPED 2026-07-08</summary>

- [x] Phase 50: W1 Vendor Templates + Unit Tests — 1/1 plan — completed 2026-07-06
- [x] Phase 51: W2 PortWriteService + Batch + Mock Tests — 1/1 plan — completed 2026-07-06
- [x] Phase 52: W3 Router/Handler/Operlog/Permission/Migration — 2/2 plans — completed 2026-07-07
- [x] Phase 53: W4 Frontend Drawer + Progress Dialog + API Wrappers — 2/2 plans — completed 2026-07-07
- [x] Phase 54: W5 E2E + Real-Device UAT + Documentation — 1/1 plan — completed 2026-07-07
- [x] Phase 55: Tech-Debt Cleanup (Phase 53 leftover sweep) — 2/2 plans — completed 2026-07-08

See [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md) for archived details (108 files / 25,224 insertions / 3,152 deletions / 1.7 days / 37/37 requirements).

</details>

<details>
<summary>✅ v1.18 网络设备硬件清单 (Phase 48) — SHIPPED 2026-07-04</summary>

### Phase 48: 网络设备组件序列号采集 (Device Component Serials) — 3/3 plans, shipped 2026-07-04

**Goal**: 支持"一机多序列号"——将网络设备的板卡/引擎卡、电源、风扇、光模块各自的序列号**作为资产设备纳入资产系统 `ops_asset`**，保存组件从属关系并在前端展示。

**Plans**: 3 plans

- [x] 48-01-PLAN.md — Schema 迁移(migration_201) + model 扩展 + component_type IS NULL 过滤
- [x] 48-02-PLAN.md — component_collector 包(SNMP ENTITY-MIB + CLI 双厂商 + 6 TextFSM 模板)
- [x] 48-03-PLAN.md — OpsAssetWriter + ReconciliationEmitter + Pipeline + 前端组件清单 Tab + UAT 推迟声明

Phase 49 gap closure (2/2 plans, 2026-07-05) — enrichChassisSerial 接入 + collectBoardsInto 接入, 14 台设备 serial_number 回填。

</details>

<details>
<summary>✅ v1.17 资产对账 (Phases 42-46) — SHIPPED 2026-07-03</summary>

- [x] Phase 42: 资产对账观测底座 (R1) — 5/6 plans (42-03 skeleton superseded by R3)
- [x] Phase 43: 告警 + 工单闭环 (R2) — 3/3 plans
- [x] Phase 44: 置信度评分 + IP 段例外 (R3) — 2/2 plans
- [x] Phase 45: 工位详情整合 (R4) — 2/2 plans
- [x] Phase 46: 半自动修复 (R5, 可选) — 2/2 plans (UAT 9/10 PASS)
- [x] Phase 47: 资产对账根因修复 — 2/2 plans (2026-07-03)

See [milestones/v1.17-ROADMAP.md](milestones/v1.17-ROADMAP.md) for details.

</details>

<details>
<summary>Pre-v1.17 Phase History (Phases 1-41)</summary>

### Phase Summary (cumulative)

- ✓ Phases 1-11 — v1.0–v1.4 基础功能 (27/27 plans)
- ✓ Phases 12-15 — v1.5 MAC地址历史数据管理 (26/26 plans)
- ✓ Phase 16 — v1.6 API密钥管理 (10/10 plans)
- ✓ Phase 17 — v1.7 加密配置同步 (6/6 plans)
- ✓ Phase 18 — v1.8 登录端加密 (4/4 plans)
- ✓ Phases 19-20 — v1.9 AD域控集成 (11/11 plans)
- ✓ Phase 21 — v1.10 网络设备权限 (1/1 plan)
- ✓ Phases 22A/22B — v1.12 深信服 VDI (6/6 plans)
- ✓ Phase 23 — v1.11 AD组自动同步 (6/7 plans, 23-13 superseded)
- ✓ Phase 24 — UUID一致性优化 (2/2 plans)
- ✓ Phase 25 — VM 数据范围权限 (5/5 plans)
- ✓ Phase 26 — v1.13 资产管理 (6/6 plans)
- ✓ Phase 27 — v1.14 全局列自定义 (1/1 plan)
- ✓ Phase 28 — v1.15 工位设备关联 (4/4 plans)
- ✓ Phase 30 — 前端性能优化 (5/5 plans, 25/25 must-haves)
- ✓ Phase 31 — P0 收尾 (5/5 plans, F-14 + F-17)
- ✓ Phase 32 — v1.14 P1 重构 + P2 架构优化 (7/7 waves, 15 P1 + 8 P2)
- ✓ Phase 33 — Vercel React Best Practices (4/4 waves, 26 defects)
- ✓ Phase 34 — 操作日志全模块集成 (11/10 plans, 100% coverage 298/298)
- ✓ Phase 37 — 前端部门选择组件统一收敛 (6/6 plans, 10 migration points)
- ✓ Phase 38 — AD 账号池统一 (4/4 plans)
- ✓ Phase 39 — v1.15 部门物理位置映射 (8/8 plans)
- ✓ Phases 40-41 — v1.16 技术债清理 (8 plans)
- ✓ Phases 42-46 — v1.17 资产对账 (14/16 plans, 42-03 superseded)
- ✓ Phase 47 — 资产对账根因修复 (2/2 plans)
- ✓ Phase 48 — v1.18 网络设备组件 (3/3 plans)
- ✓ Phase 49 — v1.18 gap closure (2/2 plans)
- ✓ Phases 50-55 — v1.19 网络设备写命令 (9/9 plans, 6 phases)
- ✓ Phase 56 — v1.20 VLAN + 端口绑定 (5/5 plans)

**Total shipped**: 162 plans across 56 phases (Phases 1-56) + v1.21 Phases 57-62 (22 plans)

</details>

---

*Last updated: 2026-10-23 — v1.22 前端公用组件整治 roadmap created (Phases 63-68, 20/20 REQ coverage: STOP×7 + FIX×5 + COMP×5 + DEBT×3)。上一里程碑 v1.21 shipped 2026-10-22 (Phases 57-62, 22 plans)。*
