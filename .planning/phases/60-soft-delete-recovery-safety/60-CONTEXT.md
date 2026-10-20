# Phase 60: 软删恢复安全 - Context

**Gathered:** 2026-10-20
**Status:** Ready for planning

<domain>
## Phase Boundary

`catSoftDeleted` 恢复前校验 AD 当前 DN 与存量 `sys_user.ad_dn` 一致（NormalizeDN 归一化比较），一致才恢复；不一致拒绝恢复、用户保持软删状态、写安全审计日志（SAFE-01）。登录路径（`SyncADUser → SyncUserFromAD`）与批量同步路径（`BatchSyncADUsers`）共用同一份软删分类与校验逻辑，同一场景两路径行为一致（SAFE-02）。

**不含：** HYGIENE-03 死代码删除（Phase 62 依赖本 phase，本 phase **不得动** `user_ou_service.go` / `GetUserDeptByADOU` / `BatchMoveUsersToNewOU`——它们是 F-22 对照语义的最后载体）、`catLocalSameName → bindLocalUserToAD` 绑定路径的同类校验（ROADMAP SC 只覆盖软删恢复）、`catActiveAD` 更新路径、拒绝后的管理员手动恢复/清理端点、`sys_user.ad_config_id` 用户级归属（58 Deferred）、UpdateStatus 双向推送（57 Deferred）、auth.go AD 登录不查 user.Status（57 Deferred 独立 bug）。

</domain>

<decisions>
## Implementation Decisions

### 校验判定（SAFE-01 核心）

- **D-01: 校验仅在存量 `ad_dn` 非空时生效；空 DN 放行恢复。** 软删行 `ad_dn` 为空/NULL = 从未绑定 AD 的 local 用户，不存在「陈旧 DN 指向他人」的接管形态；这类用户恢复行为与今天完全一致（`applyRestoreSoftDeleted` 强制转 AD）。
  — **Reversibility:** reversible — 收紧只需在校验入口加一个空值分支。

- **D-02: DN 一致性判定用 `NormalizeDN` 归一化比较**（`internal/services/addomain/dn_util.go`，单一真实源），与 Phase 58 D-02 / Phase 59 D-10 既定口径一致；不做裸字符串相等。

- **D-03: 拒绝时行保持软删状态、绝不清空存量 `ad_dn`。** 这是**关键安全不变量**：D-01 规定空 DN 放行，若拒绝路径顺手清了存量 DN，则该用户二次登录即变「空 DN → 放行」绕过校验——整道闸门失效。存量 DN 是审计证据与后续人工处置的依据。

### 登录路径呈现

- **D-04: 新增专用 `SyncErrorReason` 原因码**（命名如 `restore_refused` / `soft_delete_mismatch`，planner 定）+ `auth.go syncErrorReasonMessage()` 专用文案（如「该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员」）。`ad_authenticator.go:132` 既有「同步失败不崩认证、按原因如实反馈」管道自然承载。用户可见文案**不含 DN 内部细节**（防信息泄露），完整细节进审计 WARN。
  — 价值：运维看登录报错即可区分「安全拦截」与「AD 基础设施故障」（admin_bind/user_search 等），不必翻日志。

### 审计通道

- **D-05: 纯 applogger 结构化 WARN**，字段含 username / userID / 存量 ad_dn / AD 当前 DN / 路径（login|batch）。与 Phase 59 GUARD-01 同风格；service 层自包含，不引入 service→handler 上报通道、不进 sys_oper_log、不建新表。两路径共用同一个日志产出点。

### 批量路径记账

- **D-06: 被拒项计入 `result.Skipped` + `Errors` 明细**（error 文本标明「AD 身份不一致拒绝恢复」）。`BatchSyncResult.Skipped` 字段已存在但从未被递增过（`user_sync_service.go:696` 只打印）——本 phase 首次赋予真实语义。拒绝在校验点**短路跳过，不抛 error**：抛 error 会触发 `runInBatchedTx` 整批降级逐个重试，把策略拒绝伪装成 DB 故障。

### 共用逻辑形态（SAFE-02）

- **D-07: 校验做成两路径共用的单一入口**（分类命中 catSoftDeleted 之后、恢复写库之前调用），登录与批量路径**不得各自实现**比对逻辑。具体挂接点（`restoreSoftDeletedUser` 入口 vs `applyRestoreSoftDeleted` 入口 vs 批量 plan 阶段）归 planner，但两路径必须收敛到同一个函数。
  — **Reversibility:** costly — SAFE-02 的本体就是「行为一致」；一旦两路径各自实现，漂移即失守，后补统一需同时改两处调用点并重写全部行为用例。

### Claude's Discretion

- 校验函数的签名/命名/文件位置（`user_sync_service.go` 包内私有）
- 批量路径校验时机：阶段 2 plan 时短路（推荐——跳过无谓的密码哈希）vs 阶段 5c 写前
- 拒绝的内部信号形态（sentinel error / 分类标记 / 返回布尔）
- 专用原因码具体字符串值与中文文案措辞
- WARN 日志具体格式（GUARD-01 :435-439 的字段风格可参照）
- 测试基建复用（`user_sync_classify_test.go` / `user_sync_service_test.go` / `user_sync_service_phase59_test.go` 的 sqlite fixture）
- **预计零迁移**：本 phase 无 schema 变更、无新菜单/权限点；若研究发现确需迁移，编号从 migration_219 起（215=队列表、216=sys_config 旋钮、217/218=Phase 59 视图与权限载体，均已占用）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与路线图
- `.planning/ROADMAP.md` — Phase 60 条目：Goal、SC1~SC3、Notes（HYGIENE-03 依赖约束）、Depends on 无硬依赖（:197-209）
- `.planning/REQUIREMENTS.md:22-25` — SAFE-01 / SAFE-02 原文
- `.planning/PROJECT.md` — v1.21 关键决策段（软删恢复 = ad_dn 校验后恢复，用户选定；禁 delete AD 账号）

### 改造对象（两路径）
- `internal/services/system/user_sync_service.go` — `classifyUser` :129（分类已共用，SAFE-02 现状基础）、`indexExistingUsers` :152（预查询含软删行全字段含 AdDn）、`SyncUserFromAD` :79（登录路径事务闭包）、`restoreSoftDeletedUser` :272、`applyRestoreSoftDeleted` :296（两路径共用的恢复写库点）、`BatchSyncADUsers` :505（阶段2分类 :529-533 / 阶段3哈希 :539 / 阶段5c恢复 :656-674）、`runInBatchedTx` :732（降级重试机制——D-06 要避开它）、`BatchSyncResult` :473（Skipped 字段）
- `internal/services/system/ad_domain_user_sync_handler.go:101-136` — 批量同步 HTTP 端点（result 透传前端）
- `internal/api/v1/system/ad_domain_router.go:117` — `/users/batch-sync` 路由

### 登录失败呈现管道
- `internal/core/security/ad_authenticator.go:125-161` — `SyncErrorReason` 机制（同步失败不崩认证，:132-141 是错误分支模板）
- `internal/api/v1/auth.go:211-216` — `User==nil` 时按原因码如实反馈
- `internal/api/v1/auth.go:649-668` — `syncErrorReasonMessage()` 原因码→中文文案映射（新增原因码落点）

### DN 口径单一真实源
- `internal/services/addomain/dn_util.go` — `NormalizeDN`（attr 名与值双侧 ToLower + 去空格，仅用于比较）

### 先例与既定契约
- `.planning/phases/57-ad-outbox-lite/57-CONTEXT.md` D-04 — 禁 delete AD 账号（与 ad_dn 校验恢复不兼容）+ 复活链路代码证据（ad_authenticator.go:131 → restoreSoftDeletedUser → auth.go 签发 JWT）
- `.planning/phases/58-compare-and-move/58-CONTEXT.md` D-02 — NormalizeDN 归一化比较契约
- `.planning/phases/59-ad-login-guard-reconciliation/59-CONTEXT.md` D-05/D-07 — 结构化 WARN 字段风格 + 查询失败保守降级口径
- `.planning/phases/57-ad-outbox-lite/57-RESEARCH.md` — F-22 软删用户接管问题的原始审计描述

### 测试基建
- `internal/services/system/user_sync_classify_test.go` — 分类纯函数既有测试
- `internal/services/system/user_sync_service_test.go` — SyncUserFromAD 各分类用例（sqlite）
- `internal/services/system/user_sync_service_phase59_test.go` — GUARD-01 fixture（可复用的 sqlite 基座）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `classifyUser` + `indexExistingUsers` — 分类已两路径共用，软删行 `AdDn` 已在预查询结果中可直接取用（零额外查询）
- `NormalizeDN`（dn_util.go）— 归一化比较直接调用，无新实现
- `SyncErrorReason → syncErrorReasonMessage()` 映射 — 新原因码零架构成本接入
- `BatchSyncResult.Skipped` 字段 — 已存在、有 JSON tag、前端可见，启用即可
- `user_sync_service_phase59_test.go` sqlite fixture — 行为用例基座

### Established Patterns
- Handler-Service 分层：校验与拒绝逻辑全部在 service 层，handler 不碰
- GUARD-01 守卫风格：结构化 WARN + 全套诊断字段 + 保守策略
- 批量同步逐用户错误隔离（`runInBatchedTx` 降级管道）——拒绝项必须绕开它（D-06）
- 登录失败如实反馈（不降级、不模糊，auth.go :208 注释明文）
- 代码层修复、禁 DB TRIGGER（用户既定偏好）

### Integration Points
- `restoreSoftDeletedUser` / `applyRestoreSoftDeleted` — 校验入口（登录路径前者、批量路径后者，D-07 要求收敛）
- `BatchSyncADUsers` 阶段 2 / 阶段 5c — 批量拒绝短路点（推荐阶段 2，跳过哈希）
- `ad_authenticator.go:132` 错误分支 — 新原因码注入点
- `auth.go syncErrorReasonMessage()` — 新文案落点
- Phase 57 复活链路闭环（QUEUE-06 disable 入队）— 本 phase 是同一「复活」问题的第二道闸（disable 失效场景下的身份校验兜底），SC 叙述时可引用

</code_context>

<specifics>
## Specific Ideas

- 用户 4 区全选研究推荐项（delegator 风格延续 Phase 57/58），无个性化偏离
- **关键安全不变量（讨论中显式确认）**：拒绝路径绝不清空存量 `ad_dn`——D-01 空 DN 放行 + 清 DN = 二次登录绕过校验，整道闸门失效。此不变量必须进验收断言
- 拒绝是**策略行为不是故障**：批量不触发降级重试、登录走专用原因码与基础设施故障可区分
- 预计零迁移、零前端改动（`BatchSyncResult.Skipped` 前端已可见；登录文案走既有响应管道）

</specifics>

<deferred>
## Deferred Ideas

- **拒绝后的管理员处置端点**（查看被拒明细 / 手动恢复 / 清理软删行）— 新能力，独立 phase；当前运维处置靠审计 WARN + DBA 手工
- **`catLocalSameName → bindLocalUserToAD` 绑定路径的同类身份校验** — ROADMAP SC 未要求，风险形态不同（活跃 local 用户主动绑定）
- **对账视图增加「软删恢复被拒」条目** — `v_dept_ou_mismatch` 是 dept↔OU 专属口径，扩展属 GUARD-02 后续 phase
- **auth.go AD 登录不查 user.Status** — 57 Deferred 遗留独立 bug，与 SAFE 无关

### Reviewed Todos (not folded)

- `operlog-exclude-paths.md` — 「operlog.exclude_paths 配置驱动白名单」：匹配度 0.4，关键词误匹配（notes/phase），与软删恢复无关
- `v1.17-reconciliation-decisions.md` — 「v1.17 资产对账决策追踪」：匹配度 0.2，资产对账领域，无关

</deferred>

---

*Phase: 60-软删恢复安全*
*Context gathered: 2026-10-20*
