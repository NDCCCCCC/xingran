# Phase 57: AD 推送队列基建（outbox-lite 同事务入队） - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-16
**Phase:** 57-AD 推送队列基建（outbox-lite 同事务入队）
**Areas discussed:** 入队触发与降级语义, 旧推送路径切换时序, QUEUE-06 disable 边界, 队列表结构与合并语义
**Mode:** Advisor（4 个 gsd-advisor-researcher 并行研究 → 综合对比表 → 用户拍板）

---

## 入队触发与降级语义

| Option | Description | Selected |
|--------|-------------|----------|
| (b) diff 入队（推荐） | 对比事务前已加载旧值，四字段实际变更才入队；纯角色/备注编辑不入队 | ✓ |
| (a) 带字段即入队 | 请求携带任一 AD 字段就入队；前端全量提交致每次保存都入队 | |
| (c) 仅部门变更 | email/phone/displayName 推送通道缺位，静默漏推 | |

**User's choice:** diff 入队（推荐项）
**Notes:** 降级语义随推荐项一并生效（研究定论）：`sys_ad_config` 无启用配置 → 不入队 + WARN 放行（前置条件，防永 pending 行污染 Phase 59 守卫）；队列 INSERT 真失败 → 同事务回滚（"跳过+WARN"会架空 QUEUE-01）。

---

## 旧推送路径切换时序

| Option | Description | Selected |
|--------|-------------|----------|
| (b) 保留至 58（推荐） | 57 纯增量入队，间隙期行为=现状无推送真空；58 排水器随车删除旧路径 | ✓ |
| (a) 57 直接删 | 57→58 部署间隙 AD 推送静默中断，遇部署漂移不可控 | |
| (c) 即时排水钩子 | 把 58 核心拖进 57 或退化为旧路径变体，phase 边界溶解 | |

**User's choice:** 保留至 58（推荐项）
**Notes:** 研究发现 roadmap 当前无任何 phase 调度 handler 旧路径删除点（Phase 62 HYGIENE-03/F-22 是 user_ou_service.go 另一组死代码，无关）→ 修正项已写入 CONTEXT.md：删除点必须写进 Phase 58 范围。

---

## QUEUE-06 disable 边界

| Option | Description | Selected |
|--------|-------------|----------|
| (b) Delete+BatchDelete（推荐） | 两条软删路径都入队 disable，复活链路完整闭环；枚举预留扩展位；UpdateStatus 明确不在本期 | ✓ |
| (a) 仅 Delete | 严格按 REQUIREMENTS 字面，但 BatchDelete(:569) 批量删除复活链路残留 | |
| (c) B+UpdateStatus | 停用→disable/启用→enable 双向；属 scope 膨胀，停用可 AD 登录的洞更宜 auth 补 status 检查 | |

**User's choice:** Delete+BatchDelete（推荐项）
**Notes:** 复活链路已代码确认（restoreSoftDeletedUser 强制 enabled + 密码重置）；disable 复用 ldap_client.go:337 DisableUser（幂等）；禁 delete AD 账号（全库无先例且与 Phase 60 ad_dn 校验恢复不兼容）。

---

## 队列表结构与合并语义

| Option | Description | Selected |
|--------|-------------|----------|
| (1) 部分唯一索引（推荐） | UNIQUE(entity_type,entity_id,action_type) WHERE status IN(pending,failed)；终态行留表可查；migration_199/212 双先例 | ✓ |
| (2) 一生一行状态机 | UNIQUE(entity_type,entity_id)，行数有界合并最简；无行内历史靠 ADSyncLog 兜底 | |
| (3) JSON 快照 | 存 buildADSyncMap 四字段快照；快照过期=时间戳仲裁土壤复活，不建议 | |

**User's choice:** 部分唯一索引（推荐项）
**Notes:** 退避列（attempts/next_retry_at/last_error/failed_at）Phase 57 一次建齐；模型 tag 不声明唯一索引（W-1 教训）；队列表不软删（ConfigRestoreTask 先例）。**重要修正：migration_200 编号已失效（214 被 ad_login_history 占用），实际为 migration_215。**

---

## Claude's Discretion

- diff helper 签名/命名、队列 service 包位置（addomain vs system）
- WARN 日志格式、entity_type/action_type 枚举具体值
- sqlite 测试建表方式（直调迁移 vs 复制建表 SQL）

## Deferred Ideas

- UpdateStatus 双向推送（枚举已预留，后续 milestone 可低成本接入）
- auth.go AD 登录路径不查 user.Status（停用用户可 AD bind 拿 JWT）— 修法为 auth 补 status 检查，候选后续 phase
- BatchDelete 之外的批量路径排查（Excel 删除类入口若有）
