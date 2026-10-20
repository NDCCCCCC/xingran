---
last_updated: 2026-10-16
update_trigger: v1.21 roadmap CREATED (Phases 57-62, 19/19 REQ coverage, 三方审计依赖约束全部落图)
previous_update: 2026-07-08 after Phase 54 (W5) shipped
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
- 🚧 **v1.21 AD 双向同步 bug 修复 (AD Bidirectional Sync Bug Fixes)** — Phases 57-62 (initiated 2026-10-16) — see [REQUIREMENTS.md](REQUIREMENTS.md)

---

## Phases (v1.21)

**Milestone goal:** 修复 AD↔系统双向同步的 2 个 P0 bug（部门回退乒乓、软删用户接管风险）、3 个 P1 数据一致性缺口和 P2 卫生项，以三方审计修订后的 outbox-lite 架构落地，消除"改了部门又弹回"与静默数据丢失。

- [x] **Phase 57: AD 推送队列基建** — outbox-lite 队列表 + 同事务入队 + 按实体合并 + 删除入队 disable (QUEUE-01/02/06, HYGIENE-06) — completed 2026-10-16 (3/3 plans, code review F-01~F-06 修复 + verifier GAP-01/02/03 闭环, [VERIFICATION](phases/57-ad-outbox-lite/57-VERIFICATION.md) human_needed: 3 项真环境人工验证在 [57-HUMAN-UAT.md](phases/57-ad-outbox-lite/57-HUMAN-UAT.md))
- [x] **Phase 58: 排水器与 compare-and-move 推送** — in-process cron 排水 + 退避重试 + 幂等 OU 移动 + 配置路由无歧义 (HYGIENE-01/02, QUEUE-03/04/05) — 7/7 plans，completed 2026-10-19（SC1~6 全部 VERIFIED，见 [58-VERIFICATION.md](phases/58-compare-and-move/58-VERIFICATION.md)）
- [x] **Phase 59: 登录回写守卫与部门对账视图** — AD 登录不再回滚本地编辑 + dept↔OU 不一致可观测 (GUARD-01/02) — 4/4 plans，completed 2026-10-19（两条 Manual-Only 视觉断言 + PG 真实启动冒烟归 [59-VALIDATION.md](phases/59-ad-login-guard-reconciliation/59-VALIDATION.md)）
- [ ] **Phase 60: 软删恢复安全** — ad_dn 校验后恢复 + 登录/批量同步共用分类逻辑 (SAFE-01/02)
- [ ] **Phase 61: 影子表完整性** — 删除清理两道闸门 + 组成员事务化 + singleflight 挂载 (SHADOW-01/02/03)
- [ ] **Phase 62: 卫生收尾与死代码清理** — F-22 旧路径删除 + 嵌套组标记 + 同名部门 OU DN 冲突告警 (HYGIENE-03/04/05)

### Phase 57: AD 推送队列基建（outbox-lite 同事务入队）

**Goal**: 管理员编辑用户产生的 AD 推送在业务事务内落队列、进程重启不丢失；同用户多次编辑按实体合并仅保留最新待推送状态；sys 删除用户入队 disable 动作闭环"复活"链路。
**Depends on**: 无（v1.21 首个 phase；前置条件已满足——1641 文件 5 组提交合入 main 并推送, 2026-10-16）
**Requirements**: QUEUE-01, QUEUE-02, QUEUE-06, HYGIENE-06
**Notes**: ⚠️ 编号修正（2026-10-16 plan-phase）：ROADMAP 原写的 migration_200 已失效——214 已被 `migration_214_create_ad_login_history.go` 占用，Phase 57 实际落地 **migration_215**（见 57-CONTEXT.md D 段与 phase_specific_constraints）。本里程碑 DB migration 编号从 **215** 起，后续 phase 需要迁移时顺延编号。队列基建（表 + 同事务入队）是 Phase 59 GUARD-01 信号守卫的前置（守卫查队列行）。排水器不注册 sys_job（PROJECT.md 关键决策）。
**Success Criteria** (what must be TRUE):

  1. 管理员编辑用户（部门/昵称/邮箱/手机）后，`ad_push_queue` 在同一业务事务内出现待推送行——业务保存回滚时队列行不存在，进程重启后已入队行仍在（QUEUE-01）
  2. 同一用户短时间内多次编辑后，队列中该用户仅保留最新一条待推送行，内容为最后一次编辑结果，后一次编辑不被丢弃（QUEUE-02）
  3. sys 侧删除用户后，队列出现 disable AD 账号动作行（QUEUE-06）
  4. SQLite 测试环境补建队列表后 `go test ./...` 全绿（HYGIENE-06）

**Plans**: 3 plans (3 waves, sequential)

Plans:
**Wave 1**

- [x] 57-01-PLAN.md — `ad_push_queue` schema 层：GORM 模型 + migration_215 双方言建表/部分唯一索引 + database.go 双分支注册 + Wave 0 测试 (QUEUE-02, HYGIENE-06)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 57-02-PLAN.md — 入队 service（addomain）：四字段 diff 纯函数 + D-02 事务内前置条件 + ON CONFLICT 合并 upsert（update_user / disable）(QUEUE-01/02/06)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 57-03-PLAN.md — user_service.go 三处事务内接线（Update/Delete/BatchDelete）+ sqlite 测试环境补表 + `go test ./...` 全绿 (QUEUE-01/06, HYGIENE-06)

**Code Review 修复**（2026-10-16，见 [57-REVIEW-FIX.md](phases/57-ad-outbox-lite/57-REVIEW-FIX.md)）：F-01 disable 压制同实体活跃 `update_user` 行 / F-02 编辑表单停用入队 disable（安全缺口）/ F-03 `SKIP_AUTOMIGRATE` 旁路补建队列表 / F-04 队列表索引失败阻断启动 / F-05 去重逻辑抽纯函数（替掉假断言）/ F-06 Update 写路径收窄为显式字段（与入队 diff 语义对齐）。F-07~F-16 判定为 Phase 58 输入或低危，未在本轮处理。

### Phase 58: 排水器与 compare-and-move 推送

**Goal**: 队列被可靠排水——按用户 ad_dn 路由正确 AD 配置、闭包内重读最新值幂等移动 OU、失败退避重试至终态可查、目标 DN 消失时转终态不死循环。
**Depends on**: Phase 57（队列行存在才能排水）；phase 内 wave 顺序约束：HYGIENE-01 配置解析 helper 先于 QUEUE-04 compare-and-move（三方审计依赖）
**Requirements**: HYGIENE-01, HYGIENE-02, QUEUE-03, QUEUE-04, QUEUE-05
**Notes**: compare-and-move 是结构性修 F1（部门回退乒乓）的推送侧；禁用新旧值比较（会激活未验证的 MoveUser 重试分支）。`ErrADTargetNotFound` → 终态。

**Phase 57 移交待决项**（2026-10-16 code review 发现，Phase 57 按用户决定只记账不修）：

  1. **遗留直推与队列排水双推冲突（两条路径，缺一不可）** — 排水器落地后，现有两条绕过队列直接推 AD 的路径会同时生效 → 重复推送且可能乱序（同实体 update_user 先到、disable 后到会被覆盖回去）：
     - `internal/api/v1/system/user_handler.go:194` — 单用户编辑后的直推
     - `internal/api/v1/system/user_import_handler.go:114` `triggerADSyncAfterImport` — 批量导入后的直推（走 `BatchSyncUsersToAD`，含无 `ad_dn` 跳过、并发上限、失败降级等导入专用语义）

     **决策点**：两条路径都删 / 两者共存并加互斥。越早定越便宜——删除会使 AD 侧更新延迟到排水时才可见，且导入路径的批量语义无法直接套用单条队列行的处理方式。
  2. **`ad_push_queue` 无 `ad_config_id` 列** — 队列行不记录入队时归属的 AD 配置，排水器只能在排水时刻按 `ad_dn` 解析配置。若配置在入队与排水之间变更，重试会打到错误的域控。**决策点**：加列（需 Phase 58 迁移 + 回填策略）还是接受排水时解析现状。SUCCESS CRITERION 3 假设的是后者。
  3. **F-07~F-16 未处理项** — 见 [57-REVIEW-FIX.md](phases/57-ad-outbox-lite/57-REVIEW-FIX.md) 的逐条判定，Phase 58 排水器落地时需一并复核。

**Success Criteria** (what must be TRUE):

  1. 排水器以 in-process cron 运行且未注册 sys_job；处理 dept 推送与 disable 账号两类动作行（闭环 Phase 57 QUEUE-06 执行侧）；失败行按退避重试，达上限转终态且可通过查询确认（QUEUE-03）
  2. OU 移动采用 compare-and-move：闭包内重读用户最新值 + NormalizeDN 归一化比较，目标一致时跳过 MoveUser；同一队列行重复排水幂等，不产生重复移动（QUEUE-04）
  3. 多 AD 配置环境下推送按用户 `ad_dn` 归属解析配置，不再依赖 `First()` 取首条；`getDefaultADConfigID` 双份实现合并为公共 helper（HYGIENE-01）
  4. 含特殊字符的部门名经 OU DN 转义 helper 拼接后推送不产生非法 DN（HYGIENE-02）
  5. LDAP code 32（目标 DN 不存在）时队列行转 dead 终态并清空 `sys_user.ad_dn`，后续排水不再重试（QUEUE-05）
  6. 两条遗留直推路径（`user_handler.go:194` 单用户编辑、`user_import_handler.go:114` 导入后批量）均已按移交待决项 1 处置：或删除、或与排水互斥。不接受"两者并存且都可能触发"的终态

**Plans**: 7 plans (7 waves, sequential — 本仓 parallelization: false，波次串行)

Plans:
**Wave 0**

- [x] 58-01-PLAN.md — Wave 0 测试基座：addomain sqlite 五表 fixture + mockLDAPClient 调用计数 + F-F/F-G/F-15/F-16 缺口修补 — 3/3 tasks，completed 2026-10-19（SUMMARY: [58-01-SUMMARY.md](phases/58-compare-and-move/58-01-SUMMARY.md)）

**Wave 1**

- [x] 58-02-PLAN.md — DN 单一真实源（NormalizeDN / BuildOUDN）+ D-02 最长后缀唯一匹配纯函数 + D-03 三处调用点收敛（HYGIENE-01, HYGIENE-02）— 3/3 tasks，completed 2026-10-19（SUMMARY: [58-02-SUMMARY.md](phases/58-compare-and-move/58-02-SUMMARY.md)）

**Wave 2**

- [x] 58-03-PLAN.md — 排水器骨架：拾取谓词 + 退避曲线 + 状态机 + disable/属性两条推送路径 + F-12 UTC / F-14 default tag（QUEUE-03, QUEUE-05）— 2/2 tasks，completed 2026-10-19（SUMMARY: [58-03-SUMMARY.md](phases/58-compare-and-move/58-03-SUMMARY.md)）

**Wave 3**

- [x] 58-04-PLAN.md — compare-and-move 执行面：幂等比较 + F-A ad_dn 回写 + F-B config 作用域 OU 映射 + OU 路径 code 32 终态（QUEUE-04, QUEUE-05）— 3/3 tasks，completed 2026-10-19（SUMMARY: [58-04-SUMMARY.md](phases/58-compare-and-move/58-04-SUMMARY.md)）

**Wave 4**

- [x] 58-05-PLAN.md — cron 接线（Start() 第三个 AddJob）+ migration_216 sys_config 单旋钮 + SecondsToCronSpec + 禁 sys_job 证伪（QUEUE-03）— 2/2 tasks，completed 2026-10-19（SUMMARY: [58-05-SUMMARY.md](phases/58-compare-and-move/58-05-SUMMARY.md)）

**Wave 5**

- [x] 58-06-PLAN.md — SC6：删除两条直推路径 + EnqueueUpdateBatch 批量入队 + F-07 入队侧 Pluck + F-10 两个回滚用例（QUEUE-03）— 3/3 tasks，completed 2026-10-19（SUMMARY: [58-06-SUMMARY.md](phases/58-compare-and-move/58-06-SUMMARY.md)）

**Wave 6**

- [x] 58-07-PLAN.md — 收口：端到端用例 + 全量回归 + SC1~6 逐条证据核对 + 人工 UAT 清单 + REQUIREMENTS 回填（全部 5 REQ）— 3/3 tasks，completed 2026-10-19（SUMMARY: [58-07-SUMMARY.md](phases/58-compare-and-move/58-07-SUMMARY.md)；证据: [58-VERIFICATION.md](phases/58-compare-and-move/58-VERIFICATION.md)）

**Phase 58 收口结论（2026-10-19）**：6 条 Success Criteria **全部 VERIFIED**，每条附「命令 + 实际输出」（见 58-VERIFICATION.md）；F-13 已核对归档（留档不修）；8 项人工验证留档（见 58-HUMAN-UAT.md，含 2 项强制告知：Q2「无 OU 映射批量转 dead」/ F-11「编辑即续命」）；REQ 的 QUEUE-03/04/05 + HYGIENE-01/02 五项全部 Complete。65 个 `TestPhase58_*` 用例全 PASS。

**Phase 58 复审与补修（2026-10-19，收口之后）**

收口后跑了三道额外闸：`gsd-code-reviewer` 独立复审（**0 BLOCKER / 5 HIGH / 6 MEDIUM / 7 LOW**）→ 用户裁定修复 → `gsd-verifier` 目标达成反向验证（**ACHIEVED WITH CONDITIONS**）。

| 编号 | 摘要 | 处置 |
|------|------|------|
| H-01 | 排水器在 cron 闭包内构造 → 每轮新 mutex，轮次互斥**生产中完全失效** | ✅ 已修（构造提升为单例；单机部署，未引入行级认领） |
| H-02 | 互斥用例锁定了一个生产不成立的性质，却是 SC1 证据 | ✅ 已修（改写为断言同一实例） |
| H-03 | 退避 16min/30min 两档不可达，实际重试窗 15min vs D-05 声明 ~61min | ⏸ DEFERRED（D-05 自身两条声明互相矛盾，待真实熔断频率数据） |
| H-04 | 账号全部拨号失败不带池侧哨兵 → 一次 ~15min AD 断连**永久杀死单轮 100 行** | ✅ 已修（新增独立哨兵 `ErrAllAccountsDialFailed`） |
| H-05 | `MoveUser` 成功但本地回写失败被写成 `pushed` → AD 绑定永久失联 | ⚠ 部分修复（失败改为可见；`MoveUser` 重放经核查**不安全**，仍需人工重建 → UAT-58-9） |
| MISSED-02 | 池侧故障仍在自增 `attempts` → 长时宕机后**第一次业务失败即终态** | ✅ 已修（H-04 与本项是一对，单独修 H-04 会得到半修状态） |
| MISSED-01 | H-03 不变量注释在 H-04 后已过期，**主动误导** | ✅ 已修 |
| MISSED-03 | 58-VERIFICATION.md 端到端幂等叙述与代码不符 | ✅ 已修 |
| **MISSED-04** | **`extractRDNFromDN` 遇 RFC4514 转义逗号截断** | ⏭ **转独立 phase，见下** |

> **三次审计否决项（禁时间戳仲裁 / 禁新旧值比较 / 禁 `sys_job` 注册）在复审与补修后仍全部未命中**，由 review 与 verifier 各自独立复核。

**⚠ 移交下一 phase 的高优先项 — MISSED-04（数据破坏级）**

`extractRDNFromDN`（`internal/services/addomain/ldap_client.go`）在**第一个逗号**处截断，未处理 RFC4514 的转义逗号：`CN=Zhang\,San,OU=IT,...` → RDN 变成 `CN=Zhang\`。该 RDN 被喂给 `MoveUser` 的 `ModifyDN(deleteOldRDN=true)`，会**把 AD 对象改名**。Phase 58 只在 OU 侧加了 RFC4514 转义（HYGIENE-02），**RDN 侧没有**，并因此引入了一处不对称。

该缺陷是既存的，但 Phase 58 把它从「人工触发」推上了**每 30 秒的自动路径**。用户裁定：独立 phase 尽快修，**优先于 Phase 59**。

**Phase 58 未满足的前置条件（部署前必查）**

SC6 的价值兑现有前置：**目标环境必须已跑过部门同步**，否则导入更新会以 `unrecoverable:no-ou-mapping` 批量转 dead。建议部署前置检查：`SELECT count(*) FROM sys_dept_ou_mapping WHERE sync_enabled`。

**Wave 顺序约束落地**：HYGIENE-01 配置解析 helper（Wave 1）严格先于 QUEUE-04 compare-and-move（Wave 3）—— ROADMAP 的三方审计依赖约束。

**Cross-cutting constraints**（出现在 ≥2 个 plan 的 must_haves.truths / verify）：
- Phase 57 sentinel 回归 `go test ./internal/services/system/ -run TestPhase57 -count=1`（58-01/03/06/07 —— D-01 删直推与 F-07 BatchDelete 触碰 57 已合入代码）
- 禁 sys_job 静态证伪 grep（58-03/05/07 —— SC1：排水器不注册 sys_job）
- `TestPhase58_*` 命名约定 + quick 命令 `go test ./internal/services/addomain/ -run TestPhase58 -count=1`（58-01~06 —— Nyquist 采样契约，见 58-VALIDATION.md）
- addomain 包 sqlite 测试基座 `ad_push_queue_drain_test.go`（全部 7 plan —— Wave 0 先行的单一 fixture 事实源）

### Phase 59: 登录回写守卫与部门对账视图

**Goal**: AD 登录回写 dept 不再用旧值回滚本地编辑（信号守卫替代被三方审计否决的时间戳仲裁）；dept↔OU 不一致（含映射指向软删部门）可观测。
**Depends on**: Phase 57（守卫查询 `ad_push_queue` 的 pending/failed 行）
**Requirements**: GUARD-01, GUARD-02
**Notes**: 对账视图用普通 SQL VIEW，禁用 MV（避免 MV 刷新僵尸 / GORM AutoMigrate 阻塞前科）。
**Success Criteria** (what must be TRUE):

  1. 存在 pending/failed dept 队列行的用户经 AD 登录时，本地 dept 不被 AD 旧值覆盖，日志记录 WARN（GUARD-01）
  2. 无队列行的用户经 AD 登录时 dept 回写行为与现状一致，守卫不误伤正常同步（GUARD-01）
  3. `v_dept_ou_mismatch` 普通 SQL VIEW 可查询 dept↔OU 不一致条目及"映射指向软删部门"条目（GUARD-02）
  4. 只读端点返回对账数据且需登录鉴权访问（GUARD-02）

**Plans**: 4 plans (3 waves)
**Wave 1**
- [x] 59-01-PLAN.md — GUARD-01 登录回写守卫：活跃 update_user 队列行拦截 + WARN + 查询失败保守降级（2026-10-19, `3ba3b124` + `7334e169`）
- [x] 59-02-PLAN.md — GUARD-02 数据底座：migration_217 双方言普通视图 v_dept_ou_mismatch + 三路径注册 + AutoMigrate 前置 DROP（2026-10-19, `b4b88666` + `bda1e002`）

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 59-03-PLAN.md — GUARD-02 只读端点：分页/筛选服务 + /ad-domain/reconciliation/mismatch + migration_218 权限载体（2026-10-19, `2f1c8190` + `0d7b782c` + `a640658d`）

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 59-04-PLAN.md — GUARD-02 前端：AD 配置页「对账」Tab（权限门/筛选/表格/空错态）（2026-10-19, `a5abca44` + `08484062` + `e9df2e0b`）

### Phase 60: 软删恢复安全

**Goal**: 软删用户恢复不再盲目接管 AD 身份——AD 当前 DN 与存量 ad_dn 校验一致才恢复；登录路径与 batch-sync 对同一恢复场景行为一致。
**Depends on**: 无硬依赖（三方审计确认 SAFE 与 QUEUE/SHADOW 相互独立，可并行排期；本 roadmap 顺序执行置于 Phase 58 之后）
**Requirements**: SAFE-01, SAFE-02
**Notes**: HYGIENE-03（Phase 62）依赖本 phase 先落地——SAFE-01 落地前 user_ou_service.go 死代码是 F-22 语义的唯一载体。
**Success Criteria** (what must be TRUE):

  1. `catSoftDeleted` 恢复前校验 AD 当前 DN 与存量 `ad_dn` 一致，一致时恢复成功（SAFE-01）
  2. DN 不一致时拒绝恢复、用户保持软删状态，并写入安全审计日志（SAFE-01）
  3. 登录路径与 batch-sync 共用软删分类与校验逻辑，同一场景两路径行为一致（SAFE-02）

**Plans**: 2 plans (2 waves, sequential — 两个 plan 共享 `user_sync_service.go` 与同一测试文件)

**Wave 1**
- [x] 60-01-PLAN.md — SAFE-01 登录路径端到端：共享校验入口 `checkSoftDeletedRestore` + sentinel/原因码/中文文案 + 判定表/恢复/拒绝/审计测试 (SAFE-01, SAFE-02)（2026-10-20, `466acf04` + `1d7b3e86`）

**Wave 2** *(blocked on Wave 1 completion)*
- [ ] 60-02-PLAN.md — SAFE-02 批量路径收敛：阶段 2 过滤计入 `Skipped`（绕开降级管道）+ 双路径一致性用例 + VALIDATION/REQUIREMENTS/ROADMAP 回填 (SAFE-01, SAFE-02)

### Phase 61: 影子表完整性（删除清理 + 组成员事务化）

**Goal**: AD 端已消失的对象不再永久滞留 `sys_ad_user/ou/computer` 影子表（静默数据丢失消除）；组成员重放消除空成员可见窗口；清理与全量同步不并发。
**Depends on**: 无硬依赖（与 QUEUE/SAFE 相互独立可并行；phase 内 SHADOW-03 依赖 SHADOW-01 先落地）
**Requirements**: SHADOW-01, SHADOW-02, SHADOW-03
**Success Criteria** (what must be TRUE):

  1. 全量同步末尾，AD 端已消失的 `sys_ad_user/ou/computer` 行被软删除（SHADOW-01）
  2. 两道闸门生效：LDAP 本轮返回 0 条时清理拒绝执行；拟删数量超过存量 50% 时拒绝执行（SHADOW-01）
  3. `sys_ad_group_member` 重放在单事务（Delete+Create 包 Transaction）内完成，重放过程中外部查询看不到空成员窗口（SHADOW-02）
  4. 删除清理挂 `syncDataInternal` 内执行（复用 singleflight），与全量同步互斥不并发（SHADOW-03）

**Plans**: TBD

### Phase 62: 卫生收尾与死代码清理

**Goal**: F-22 旧路径死代码在 SAFE-01 落地后安全移除；嵌套组不污染用户成员关系；同名兄弟部门 OU DN 冲突显式告警不静默覆盖。
**Depends on**: Phase 60（HYGIENE-03 死代码删除必须在 SAFE-01 之后——先删会丢 F-22 对照语义）
**Requirements**: HYGIENE-03, HYGIENE-04, HYGIENE-05
**Success Criteria** (what must be TRUE):

  1. `user_ou_service.go` 整文件、`GetUserDeptByADOU`、`BatchMoveUsersToNewOU`、`dept_ou_mapper.go:31` 等过期注释及其关联测试全部删除后 `go build ./...` 通过且测试套件全绿（HYGIENE-03）
  2. 嵌套组的 member DN 被标记/过滤，不混入用户成员关系数据（HYGIENE-04）
  3. 同名兄弟部门生成相同 OU DN 时检测并告警，映射不静默互相覆盖（HYGIENE-05）

**Plans**: TBD

---

### Progress Table (v1.21)

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 57. AD 推送队列基建 | 3/3 | Complete | 2026-10-16 |
| 58. 排水器与 compare-and-move 推送 | 6/7 | In progress (wave 5 done 2026-10-19, SC6 终态达成) | - |
| 59. 登录回写守卫与部门对账视图 | 4/4 | Complete (59-01~59-04 done 2026-10-19) | 2026-10-19 |
| 60. 软删恢复安全 | 0/2 | Planned (2 waves) | - |
| 61. 影子表完整性 | 0/TBD | Not started | - |
| 62. 卫生收尾与死代码清理 | 0/TBD | Not started | - |

---

## Archive: Shipped Milestones

<details>
<summary>✅ v1.20 网络设备 VLAN + 端口绑定 (Phase 56) — SHIPPED 2026-07-10</summary>

### Phase 56: 网络设备 VLAN + 端口绑定 (v1.20.1) — 5/5 plans, shipped 2026-07-10

**Goal:** 在 v1.19 端口写命令 MVP 基础上扩展 2 个新写命令：① 修改 access VLAN ② IP+MAC+Port 静态绑定。复用 v1.19 vendor 模板 / operlog / 权限 / 批量 / e2e 全套基建。

**Depends on:** **v1.19 端口写命令 MVP (Phases 50-55)** — 复用 vendorPortTemplate + pre_state_check + port_write_service + execSinglePort + PortWriteModal + BulkWriteDrawer 全套基建

**Requirements**: 22 (VLAN-01~06, BIND-01~07, INFRA-01~04, UI-01~06, TEST-01~05)

**Success Criteria**: 7 项全部通过 (vendor template / portwrite 单测+e2e / go build / tsc / vite build vendor-react 774.95 kB 零回归 / operlog regression / 56-HUMAN-UAT.md 12 项 site-visit 推迟)。

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

See [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md) for archived details (108 files / 25,224 insertions / 3,152 deletions / 1.7 days / 37/37 requirements). 7 site-visit SSH verifications + 6 v1.19.x+ follow-ups deferred per `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md` and `v1.19-REQUIREMENTS.md` §Future Requirements / §Tech Debt sections.

</details>

<details>
<summary>✅ v1.18 网络设备硬件清单 (Phase 48) — SHIPPED 2026-07-04</summary>

### Phase 48: 网络设备组件序列号采集 (Device Component Serials) — 3/3 plans, shipped 2026-07-04

**Goal**: 支持"一机多序列号"——将网络设备的板卡/引擎卡、电源、风扇、光模块各自的序列号**作为资产设备纳入资产系统 `ops_asset`**（每个组件一条记录，`DeviceSN`=组件序列号），保存组件对交换机/路由的从属关系并在前端展示。

**Depends on**: **v1.17 对账底座（Phase 42-46）** — Phase 48 是 R2 物理层对账的组件序列号形态，需对账引擎先就位

**Requirements**: 14 D-ids (D-01~D-14) 覆盖,源自 48-CONTEXT.md,无正式 REQ-ID

**Success Criteria**: 16 项全部通过自动化 UAT(schema introspection / go build / 21 collector tests / 5 D-10 流水线测试 / 3 handler tests / 5 operlog regression / tsc / vite build)。3 项 site-visit 真机 UAT(S8700/RS8607E)按 RESEARCH §Environment Availability 显式推迟到下次现场访问。

**Plans**: 3 plans

- [x] 48-01-PLAN.md — Schema 迁移(migration_201) + Asset/Reconciliation model 扩展 + asset_service List/Statistics 默认 component_type IS NULL 过滤 (D-01, D-05, D-06, D-07)
- [x] 48-02-PLAN.md — component_collector 包(SNMP ENTITY-MIB 单 GET 收集器 + 锐捷 temprature* 过滤 + 华为 dual-class 去重 + OwnerResolver stack 接口 + 4 TextFSM 模板 + CLI 收集器) (D-08, D-09, D-10, D-11, D-12)
- [x] 48-03-PLAN.md — OpsAssetWriter(UPDATE-only + D-04 父缺失降级) + ReconciliationEmitter(conflict_type=F + recon_category=component_serial) + Pipeline + DeviceInfoCollectionService cron hook + operlog + 后端 list 端点 + 前端组件清单 Tab + UAT 推迟声明 (D-02, D-03, D-04, D-06, D-07, D-13, D-14)

See [milestones/v1.18-ROADMAP.md](milestones/v1.18-ROADMAP.md) for archived details. Phase 49 gap closure (2/2 plans, 2026-07-05) — enrichChassisSerial 接入 + collectBoardsInto 接入, 14 台设备 serial_number 回填。

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

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 42. 资产对账观测底座 (R1) | 5/6 | Complete | 2026-06-27 |
| 43. 告警 + 工单闭环 (R2) | 3/3 | Complete | 2026-06-27 |
| 44. 置信度评分 + IP 段例外 (R3) | 2/2 | Complete | 2026-06-28 |
| 45. 工位详情整合 (R4) | 2/2 | Complete | 2026-06-28 |
| 46. 半自动修复 (R5, 可选) | 2/2 | Complete | 2026-07-03 |
| 47. 修復資產對賬系統根因 | 2/2 | Complete | 2026-07-03 |

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

**Total shipped**: 156 plans across 46 phases (Phases 1-56)

</details>

---

*Last updated: 2026-10-16 — v1.21 AD 双向同步 bug 修复 roadmap created (Phases 57-62, 19/19 REQ coverage, 依赖约束来自三方审计)。注: 里程碑实际 REQ 数 = 19 (QUEUE×6 + GUARD×2 + SAFE×2 + SHADOW×3 + HYGIENE×6)。上一里程碑 v1.20 shipped 2026-07-10 (Phase 56, 5 plans)。*
