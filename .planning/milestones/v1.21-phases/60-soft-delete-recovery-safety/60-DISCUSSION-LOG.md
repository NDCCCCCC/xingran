# Phase 60: 软删恢复安全 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-20
**Phase:** 60-软删恢复安全
**Areas discussed:** 存量 ad_dn 为空时怎么办, 拒绝后登录如何呈现, 审计日志通道, 批量路径被拒项记账

---

## 存量 ad_dn 为空时怎么办

| Option | Description | Selected |
|--------|-------------|----------|
| 放行恢复（推荐） | 空 DN = 无身份冲突可比对，接管形态不成立；校验仅在存量 ad_dn 非空时生效；拒绝路径绝不清空存量 DN（防二次登录变空 DN 绕过） | ✓ |
| 拒绝恢复（最保守） | 无法验证一致性就不恢复；local 删除用户 + AD 同名账号场景永远无法登录，管理员须先清理软删行 | |
| 放行+额外审计 | 恢复照常，空 DN 恢复额外写 INFO/WARN 审计，非空 DN 才做强校验 | |

**User's choice:** 放行恢复（推荐）
**Notes:** 随选项生效的关键不变量：拒绝路径绝不清空存量 ad_dn，否则空 DN 放行规则使二次登录绕过校验。

---

## 拒绝后登录如何呈现

| Option | Description | Selected |
|--------|-------------|----------|
| 新增专用原因码（推荐） | 如 restore_refused；专用文案「AD 身份不一致已被安全拦截」，运维可区分安全拦截与基础设施故障；DN 细节不进用户文案 | ✓ |
| 复用 user_sync 文案 | 零改动，但安全拦截在用户侧与基础设施故障不可区分 | |
| 通用文案+响应标记 | 复用文案但响应体附非敏感标记字段供前端/运维识别 | |

**User's choice:** 新增专用原因码（推荐）
**Notes:** 复用 ad_authenticator.go SyncErrorReason → auth.go syncErrorReasonMessage() 既有映射模式。

---

## 审计日志通道

| Option | Description | Selected |
|--------|-------------|----------|
| 纯 applogger WARN（推荐） | 两路径统一、service 层自包含，含 username/userID/存量DN/当前DN/路径(login|batch)；GUARD-01 同风格 | ✓ |
| applogger + operlog | 拒绝事件进 sys_oper_log（UI 可见）；登录路径需跨层上报、批量路径需汇总，复杂度明显上升 | |
| 自建安全事件表 | 新建 sys_security_event + 查询端点，审计语义最正规但超出本 phase 体量 | |

**User's choice:** 纯 applogger WARN（推荐）
**Notes:** 无。

---

## 批量路径被拒项记账

| Option | Description | Selected |
|--------|-------------|----------|
| 计入 Skipped（推荐） | 计入 result.Skipped + Errors 明细；语义准确（策略拒绝≠失败）；不触发 runInBatchedTx 降级重试；Skipped 字段首次获得真实语义 | ✓ |
| 计入 Failed | 复用既有错误隔离管道；语义混淆且需确保拒绝不抛 error | |
| 仅日志不计数 | 仅写审计 WARN；结果面板可观测性差 | |

**User's choice:** 计入 Skipped（推荐）
**Notes:** 拒绝在校验点短路跳过，不抛 error——抛 error 会触发整批降级逐个重试，把策略拒绝伪装成 DB 故障。

---

## Claude's Discretion

- 校验函数签名/命名/位置；批量校验时机（推荐阶段 2 plan 时短路，跳过哈希）
- 拒绝内部信号形态（sentinel error / 分类标记 / 布尔）
- 专用原因码具体字符串值与中文文案措辞
- WARN 日志具体格式（GUARD-01 :435-439 风格参照）
- 测试基建复用；预计零迁移（如需则从 migration_219 起）

## Deferred Ideas

- 拒绝后的管理员处置端点（查看/手动恢复/清理软删行）— 独立 phase
- catLocalSameName → bindLocalUserToAD 绑定路径同类校验 — ROADMAP SC 未要求
- 对账视图增加「软删恢复被拒」条目 — GUARD-02 后续 phase
- auth.go AD 登录不查 user.Status — 57 Deferred 遗留独立 bug
