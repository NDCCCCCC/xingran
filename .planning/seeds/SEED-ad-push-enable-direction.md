---
title: AD 推送队列 enable 双向推送（恢复用户 → 自动重新启用 AD 账号）
date: 2026-10-21
priority: medium
status: pending
origin: verify-work 57 Test 3 实测发现（UAT 会话，2026-10-21 13:03）
related_phase: v1.21 Phase 57/58/60 已交付基线
related_debug: .planning/debug/ad-push-queue-drain-interval-not-hot-reloaded.md
---

# AD 推送队列 enable 双向推送（恢复用户 → 自动重新启用 AD 账号）

## 背景与缺口（Phase 57 Test 3 实测确认）

v1.21 交付的 outbox-lite 只覆盖"sys → AD"的 **disable 方向**：sys 删除/停用用户 →
同事务入队 `action_type='disable'` → 排水器 `DisableUser`（UAC 514）→ AD 端禁用。
已实机验证（2026-10-21 12:51 d_chenchao：删除 → 28 秒内 AD 账号被禁用）。

**反向缺失**：v1.21 刻意裁掉了 enable 方向（`internal/models/ad_push_queue.go:29`
注释明示 `ADPushActionEnable 在 v1.21 不存在`；`internal/services/system/user_service.go:690`
明示状态 1→0 启用方向不入队）。实测后果：

- sys 侧恢复（清 deleted_at / UI 恢复）**不会**把 AD 端账号重新启用；
- 用户必须等 AD 登录自动恢复通道（`user_sync_service.go:309 restoreSoftDeletedUser`，
  SAFE-01 守卫放行后恢复），但该通道的前提是 **AD 端账号可用** —— 账号已被 UAC 514
  禁用，AD 登录 bind 即失败 data 533，通道走不通；
- 唯一恢复路径 = AD 管理员手工解禁（或调 `ldap_client.go:333 EnableUser`——函数已存在
  但 v1.21 无任何调用入口）。

## 建议方案

1. 新增 `models.ADPushActionEnable`（扩展位已在常量注释预留）；
2. `user_service.go` 恢复路径 / 状态 1→0 转换处同事务入队 enable（复用
   `EnqueueDisableOnStatusTransition` 的反向形态）；
3. 排水器 `drainRow` 增加 `case models.ADPushActionEnable: c.EnableUser(*user.AdDn)`
   （`ldap_client.go:333` 原语现成，UAC 544 Replace 幂等）；
4. 与 Phase 60 SAFE-01/SAFE-02 的交互需评审：enable 行入队前是否也要过
   `checkSoftDeletedRestore` 身份校验（防止"恢复了一个 ad_dn 已漂移的账号"把错误身份
   重新启用）；
5. 部分唯一索引与合并语义：enable 与 disable 同 (entity_type, entity_id, action_type)
   维度不冲突，但"先 disable 后 enable"的行序语义（最新意图胜出）需设计——可复用
   F-01 压制模式（enable 入队时压制同实体 pending disable 行，或反之）。

## 验收口径（草案）

- sys 恢复用户 → enable 行入队 → 排水器执行 → AD 端 UAC 544 → 该账号可 AD 登录；
- 与 SAFE-01 守卫集成：身份不一致的恢复不入队 enable 且记安全 WARN；
- sqlite 自动化 + 一次实机 UAT（与 2026-10-21 d_chenchao 场景对称）。

## 关联

- Phase 57 QUEUE-06（disable 方向已闭环）
- Phase 60 SAFE-01/SAFE-02（恢复守卫）
- 2026-10-21 verify-work 57 Test 3 实测记录：`.planning/phases/57-ad-outbox-lite/57-HUMAN-UAT.md`
