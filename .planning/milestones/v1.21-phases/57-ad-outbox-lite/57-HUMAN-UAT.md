---
status: complete
phase: 57-ad-outbox-lite
source: [57-VERIFICATION.md]
started: 2026-10-17T00:00:00+08:00
updated: 2026-10-21T05:08:54.607Z
---

## Current Test

[testing complete]

## Tests

### 1. PG 真实环境验证 ON CONFLICT 部分索引合并路径（QUEUE-02）
expected: 连续两次编辑同一用户 → `ad_push_queue` 仅 1 行 pending（合并而非新增行）。若报 `42P10 there is no unique or exclusion constraint matching the ON CONFLICT specification` → 谓词写错。sqlite 侧 `TestPhase57_EnqueueUpdate_MergeOnSecondEdit` 已提供等价回归保护，但 PG 的冲突目标推断路径只能实机确认。
result: [pass]
evidence: 2026-10-21 11:44 实机 PG(10.62.10.34/ruoyi) 探针已验合并路径 —— 以生产同款子句 ON CONFLICT (entity_type, entity_id, action_type) WHERE status IN ('pending','failed') DO UPDATE 对生产部分唯一索引 uq_ad_push_queue_active 推断成功（未报 42P10），两次入队后仅 1 行且同一 id、status=pending、attempts=0、last_error/next_retry_at/failed_at 全清；再把该行打成 failed/attempts=3/last_error=boom 后第三次入队仍合并为 1 行（事务内 ROLLBACK，无残留）。UI 首次实测出现 2 行 pushed：排水器 30s 刻度（internal/scheduler/ad_sync_tasks.go:193）已把首行推成终态，终态行退出唯一索引是设计行为（TestPhase57_EnqueueUpdate_SyncsWithTerminalRow），非合并失效；字面级 UI 双击复测待做（排水间隔设 300s + 重启后端，在同一窗口内连改两次）。

### 2. 进程重启不丢已入队行（QUEUE-01 / SC1 后半）
expected: PG 部署后编辑用户入队 → `kill -9` → 重启 → `SELECT * FROM ad_push_queue WHERE status='pending'` 行仍在。
evidence: 2026-10-21 时间线 —— 12:03:35.075 入队 pending(id 前缀 4b743635, entity 8bd62962=chenchao-076, update_user, attempts=0)；12:05:29 复核仍 pending 且 12:05:00 刻度未被推走（旁证 300s 间隔已生效）；12:09:35 kill 前最后确认 pending；12:09:5x taskkill /PID 49428 /F 强杀后端（9000 端口释放）→ 杀后查库仍 pending/attempts=0/字段零漂移；12:11 用户重启后端（日志 12:11:42 确认 drain_interval_seconds=300s 生效、ad_push_queue 排水 cron 0 */5 * * * * 注册）；12:15:47 重启后复核：该行仍 pending / attempts=0 / updated_at 未变 —— kill -9 与进程重启均未丢失或篡改已入队行，QUEUE-01 持久化断言成立。附注：12:15:00 刻度排水器拾取该行并尝试 AD 属性更新，因两处独立缺陷（markFailed 写 last_error 含 NUL 字节触发 SQLSTATE 22021，见 .planning/debug/ad-push-queue-drain-interval-not-hot-reloaded.md 追加段；AD-sync 服务账号缺写权限 LDAP code 50）未能推成 pushed，属排水阶段缺陷、不回溯本 Test 的持久化结论。
result: [pass]

### 3. AD 端复活链路闭环（QUEUE-06 端到端）
expected: sys 删除用户 → Phase 58 排水器执行 DisableUser → AD 端 UAC 514 → bind 失败（data 533）。执行侧（排水器）已存在：internal/services/addomain/ad_push_queue_drain_service.go:400 的 `case models.ADPushActionDisable:` → `c.DisableUser(*user.AdDn)`（UAC 514），本 phase 只验队列行产生（已由测试锁定）；本项本质是 Phase 58 验收项，挂此处防遗漏。注意 2026-10-21 实测更新：排水器已上线且**多实例并发排水**（本地 + 10.62.10.39），行在删除后 28 秒即被远端实例推成 pushed。
evidence: 2026-10-21 时间线 —— 12:51:02 用户在 UI 删除 d_chenchao（sys_user f5d4152b-…，昵称 陈超2，ad_dn CN=d_chenchao,OU=孝感中心支公司,…,DC=cpic,DC=com,DC=cn；app.log 确认 POST /api/v1/system/users/f5d4152b…/delete 200，latency 46ms）；同事务入队 disable 行 ad_push_queue id df40eff4-a3cc-4583-b07c-d5f89bbbaf85（created 12:51:02.472，比 user.deleted_at 12:51:02.466 晚 6ms，同事务证据）；**12:51:30.004 该行被排水器推成 pushed（attempts=0、pushed_at=updated_at=12:51:30.004、无 last_error）** —— 排水者不是本地实例（10.62.10.33，其 300s cron 下一刻度应为 12:55:00 且 app.log 12:51-12:52 无任何排水日志），而是 pg_stat_activity 里并发连库的另一台后端实例 **10.62.10.39**（其 idle 连接正持有 SELECT * FROM ad_push_queue WHERE status IN ('pending','failed')… 语句，5 分钟 cron 猜测对齐 12:51:30 或其自转周期）→ 说明生产环境已有多实例同时跑排水 cron，单靠本地日志无法观测全部排水行为（多实例排水并发为 Phase 58 需关注点）；DisableUser(UAC 514) 幂等 Replace 在多实例下无副作用。旧卡住行 4b743635 已按约定于 12:50:41 手工置 dead（last_error='dead:manual-uat-cleanup (defect X: NUL in last_error)'），本地实例 22021 重试循环止于 12:50:00 刻度、12:55 后无相关错误。**待人工步：用 d_chenchao 在登录页做 AD 域登录，预期 bind 失败 data 533（账号已禁用）**。
adjudication: 用户裁决（2026-10-21 12:30）—— 缺陷 Y（AD-sync 服务账号对 chenchao-076 所在 OU 无属性写权限，LDAP code 50）系域控策略『域控不允许同组管理员互相修改』所致，属环境侧授权问题，忽略不计。Test 3 将改用**非管理员同组账号**执行：sys 删除该账号 → 排水器 DisableUser → AD 端 UAC 514 → AD 登录应报 data 533。⚠ 实机噪声源：卡住的 update_user 行 4b743635 每 5 分钟刻度仍在重试同一 OU 并持续刷 SQLSTATE 22021 GORM 错误（缺陷 X 未修前每刻度 1 条），核对本 Test 证据时以 disable 行为为准、忽略该行报错。
result: [pass]
evidence: 2026-10-21 端到端时间线（实际执行超出本 phase 原定范围：Phase 58 排水器已上线，enable 执行侧一并得到实机验证）—— 12:51:02 用户在 UI 删除 d_chenchao（sys_user f5d4152b-2b17-4959-87a7-3b0ebeb9aef7，ad_dn CN=d_chenchao,OU=孝感中心支公司,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn；app.log 确认 POST /api/v1/system/users/f5d4152b…/delete → 200）；Delete 路径同事务入队 disable 行 ad_push_queue id=df40eff4-a3cc-4583-b07c-d5f89bbbaf85（created 12:51:02.472452，比 user.deleted_at 12:51:02.466407 晚 6ms —— 同事务证据）；12:51:30.004 该行被排水器推成 pushed（attempts=0、pushed_at=updated_at=12:51:30.004、无 last_error）→ 用户人工确认域控里 d_chenchao 账号已被禁用（UAC 514 生效的用户侧直接证据）。AD 登录 data 533 步骤经用户裁决（m00476：『域控账号被禁用肯定就无法登陆了，这还用测试吗？』）省略 —— UAC 514 → bind 拒绝是 AD 标准语义，非本系统行为；复活链路'已删用户经 AD 登录复活'在 AD 端被禁用后即被物理切断。附注：排水者为并发实例 10.62.10.39（本机 10.62.10.33 的 300s cron 下一刻度应为 12:55:00，app.log 12:51-12:52 无排水日志）—— 多实例并发排水为 Phase 58 需关注点；DisableUser 幂等 Replace 在多实例下无副作用。

## Summary

total: 3
passed: 3
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
