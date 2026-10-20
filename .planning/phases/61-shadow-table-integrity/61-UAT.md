---
status: testing
phase: 61-影子表完整性（删除清理 + 组成员事务化）
source: [61-VERIFICATION.md]
started: 2026-10-21T08:20:00Z
updated: 2026-10-21T08:20:00Z
---

## Current Test

number: 1
name: 生产 PG 真并发空窗观察（组成员重放执行期间，第二会话查询 sys_ad_group_member）
expected: |
  外部会话要么看到旧成员集、要么看到新成员集，观察不到空成员窗口（PG READ COMMITTED + 单事务原子性）
awaiting: user response

## Tests

### 1. 生产 PG 真并发空窗观察
expected: 重放期间从第二会话查询 `sys_ad_group_member`：外部会话要么看到旧成员集、要么看到新成员集，观察不到空成员窗口（PG READ COMMITTED + 单事务原子性）。
why_human: sqlite 进程内测试无法建立真正的跨连接并发观察者；单测只覆盖了原子性的进程内可证伪代理（TestSyncGroupMembers_CreateFailRollsBackDelete）。
result: [pending]

### 2. 首轮批量清理规模真实观察
expected: 部署后核对 SyncResult 三计数（deletedUserCount/deletedOUCount/deletedComputerCount）与 sys_oper_log 清理日志：清理计数与 AD 端实际消失对象数一致；若拟删 >50% 被闸门拒绝属预期保护行为，运维不得误报为故障。
why_human: 清理规模依赖生产 PG 存量与真实 AD 数据对比，无法在本地 sqlite fixture 复现。
result: [pending]

### 3. 闸门真触发演练
expected: 测试域控删除 >50% 对象后跑全量同步：清理被 Gate 2 拒绝，日志含「拟删除 x/y 个...(>50%)」WARN（不含 DN 明文），整轮同步仍 success。
why_human: 需人为构造 AD 端大规模删除的真实域控环境。
result: [pending]

### 4. 生产 PG 唯一约束核对（FLAGGED ASSUMPTION A1）
expected: 启动冒烟后执行 `\d sys_ad_user / sys_ad_ou / sys_ad_computer / sys_ad_group`：各表存在 (ad_config_id, *_dn/name) 唯一约束；若缺失，软删行复现将表现为重复行而非约束冲突（复活路径设计仍正确）。
why_human: 本机无 PG 实例，sqlite 单测无法覆盖生产约束形态。
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps

部署整体性提醒（承 61-02/61-VERIFICATION）：61-01/02/03 三个 plan 的产物必须**整体部署**——清理 writer（61-01）与复活路径（61-02）分隔部署会使软删行复现撞唯一约束。
