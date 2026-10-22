---
status: complete
phase: 61-影子表完整性（删除清理 + 组成员事务化）
source: [61-VERIFICATION.md]
started: 2026-10-21T08:20:00Z
updated: 2026-10-22T01:25:00Z
---

## Current Test

[testing complete]

## Tests

### 1. 生产 PG 真并发空窗观察
expected: 重放期间从第二会话查询 `sys_ad_group_member`：外部会话要么看到旧成员集、要么看到新成员集，观察不到空成员窗口（PG READ COMMITTED + 单事务原子性）。
why_human: sqlite 进程内测试无法建立真正的跨连接并发观察者；单测只覆盖了原子性的进程内可证伪代理（TestSyncGroupMembers_CreateFailRollsBackDelete）。
evidence: |
  【2026-10-21 三轮观测，20:05 轮捕获到真反例】
  - 16:41-16:42(+08) 第一轮布防：4Hz 采样 65 行（tmp-uat61-t1-samples.csv），member_count=2815 恒定
    （未在重放窗口内，仅基线）。
  - 17:05+08 轮：采样器因 WSL 会话退场提前停止，未捕获进行中样本；17:07 复核 2815 行 updated_at
    100% 落在 09:05:04.759-05.160（0.4s）窗口内——数据侧与单事务原子性一致。
  - 20:05+08 轮（12:05 UTC，双实例并发：10.62.10.39 00.004-06.778 success / 10.62.10.33
    00.010-21.046 failed=账号池拨号全败未达重放）：4Hz 采样 1134 行（tmp-uat61-t1-samples-v2.csv）。
    20:05:02.620 样本 = count 2815→2105（缺 710 行 = CN=CXHUB-VPN-NORMAL 组整组行数，该组恰 710 行）
    + min_updated_at 仍为 09:05 旧值 + max 已为 12:05 新值（组间混合态）；
    20:05:02.905 → 2808（过渡），03.188 起恢复 2815。V 形空窗持续 ~0.57s。
  【判定 2026-10-21 21:15 根因闭合】外部会话观测到了 VPN-NORMAL 组「Delete 已生效、Create 不可见」的瞬态——
  违反 expected「观察不到空成员窗口」。但最小复现实验（21:10，真 PG 10.62.10.34）证明单事务实现无罪：
  以 61-03 精确形态（BEGIN; DELETE 710 行 → INSERT 710 行 → pg_sleep(15) 未提交）+ 并发读会话
  0.25s 采样 12 次，count/DISTINCT 全程恒定（3525|60 无任何抖动），READ COMMITTED MVCC 下
  中间态完全不可见 → cb1382c3 单事务实现在真 PG 上无可观测空窗。
  根因改判：20:05 轮唯一 success 实例为 10.62.10.39（10.62.10.33 拨号失败未达重放），其生产
  二进制未包含 61-03 修复（cb1382c3 于 2026-10-21 01:47 提交；/health version=dev 无构建信息、
  前端 hash 与本机 embedded dist 一致仅证明前端资源，不证明 Go 二进制版本）——V 形与
  cb1382c3 注释中描述的旧实现（两步独立提交）逐特征自洽：02.620 = VPN 组 DELETE 已提交、
  INSERT 在途；02.905 = 全部组 DELETE 已提交、尾部 7 行 INSERT 未提交。
  【结论】代码层修复（61-03）经验证有效；生产观测到的空窗源于部署滞后（39 跑旧码），非代码缺陷。
  行动项：10.62.10.39 重新发版（build-linux.bat，需含 cb1382c3/af162852/75a3c015/6a08933d +
  426903fd 热重载/ff699b9b 登录状态检查），发版后在 03:05 轮复测本项。
  判 fail 维持（判据是生产形态，生产当前跑旧实现），但修复路径明确=发版，非代码返工。
  【2026-10-22 08:05 轮复测 — PASS】10.62.10.39 已重新发版（含 cb1382c3 单事务实现）。08:05(+08)
  轮 sync_log 双实例 success（08:05:00.011 dur=9s / 08:05:00.013 dur=6s，67 组，user_count=4177），
  数据侧 2815 行 updated_at 全部落在 08:05:04.882-05.769（0.887s 重放窗口）。采样器 v5
  （scripts/uat61-t1-poller-v5.ps1，0.2s 目标间隔实测 2.8 样本/秒）自 07:24:31 至 08:17:59 连续采样
  3025 点，**member_count 全程 2815、DISTINCT group_dn 全程 59，零偏差**；重放窗口内 26 个样本
  （08:05:00.141 / 04.670 / 05.011 / 05.386 / 05.740 / 06.126 … 08:05:09.822）逐条 2815|59。
  与 20:05 失败轮同等采样密度（0.57s 空窗被 3 个样本捕获）对比：本次 0.887s 窗口被 5 个样本覆盖
  且无任何 V 形 → 单事务实现的生产行为符合 expected，空窗不可观测。
  结论：原 fail 判定源于部署滞后（39 跑旧两步实现），发版后已消除。**result 改判 pass。**
result: [pass]

### 2. 首轮批量清理规模真实观察
expected: 部署后核对 SyncResult 三计数（deletedUserCount/deletedOUCount/deletedComputerCount）与 sys_oper_log 清理日志：清理计数与 AD 端实际消失对象数一致；若拟删 >50% 被闸门拒绝属预期保护行为，运维不得误报为故障。
why_human: 清理规模依赖生产 PG 存量与真实 AD 数据对比，无法在本地 sqlite fixture 复现。
evidence: |
  Phase 61 清理代码生产首跑 = 2026-10-21 14:05(+08) 全量同步轮（sync_log 06:05 UTC 双行 =
  双实例 10.62.10.39/33 均 success，duration 7.2s/10.7s，error_count=0）。本机 logs/app.log :706711-706713：
  - "[OU同步] 清理了 977 个已不存在的OU"（14:05:09）
  - "[用户同步] 跳过过期用户清理: 拟删除 4643/8820 个用户(>50%),可能是 LDAP 返回不完整。
    请人工核实后手动清理"（14:05:10，WARNING）
  - "[电脑同步] 清理了 13 台已不存在的电脑"（14:05:10）
  对账：sync_log ou_count=3590 与 OU 影子表清理后活行数 3590 精确一致（差集清零）；
  user_count=4177 = 8820 - 4643（AD 端实到用户数与拟删数自洽），闸门拦截后用户影子表
  零删除（16:44 基线 8820 → 17:07 复核 8822，仅自然新增）；WARN 文案与
  shadow_cleanup_service.go:92 模板逐字一致且不含 DN 明文；整轮 sync_status=success。
  结论：三类清理计数与 AD 端数据一致；闸门拒绝为预期保护行为，与 expected 完全吻合。
result: [pass]

### 3. 闸门真触发演练
expected: 测试域控删除 >50% 对象后跑全量同步：清理被 Gate 2 拒绝，日志含「拟删除 x/y 个...(>50%)」WARN（不含 DN 明文），整轮同步仍 success。
why_human: 需人为构造 AD 端大规模删除的真实域控环境。
evidence: |
  2026-10-21 14:05(+08) 生产轮闸门被真实触发（非演练）：LDAP 返回不完整导致用户差集虚大
  （拟删 4643/8820 = 52.6% > 50%）→ Gate 2 拒绝，日志含完整 WARN（拟删除 4643/8820 个用户(>50%)，
  不含 DN 明文，自带「请人工核实后手动清理」运维指引），用户影子表 8820 行零删除，整轮
  sync_status=success、error_count=0。代码路径与「人为删对象」演练完全相同
  （cleanupVanishedUsers 差集 >50% 分支，shadow_cleanup_service.go:90-95）。
  与原设计的差异仅在触发源：LDAP 返回不完整（差集虚大）而非域控端真实删除；
  三个验收点（拒绝 / WARN 形态 / 轮次 success）等价覆盖。测试域控到位后可另做真删演练（可选项，非阻塞）。
result: [pass]

### 4. 生产 PG 唯一约束核对（FLAGGED ASSUMPTION A1）
expected: 启动冒烟后执行 `\d sys_ad_user / sys_ad_ou / sys_ad_computer / sys_ad_group`：各表存在 (ad_config_id, *_dn/name) 唯一约束；若缺失，软删行复现将表现为重复行而非约束冲突（复活路径设计仍正确）。
why_human: 本机无 PG 实例，sqlite 单测无法覆盖生产约束形态。
evidence: |
  2026-10-21 16:33 (+08) WSL psql 直连生产 PG 10.62.10.34/ruoyi 实测：
  - pg_constraint contype='u'：sys_ad_ou uk_ad_ou_dn UNIQUE (ad_config_id, ou_dn)；
    sys_ad_group uk_ad_group_dn UNIQUE (ad_config_id, group_dn)；
    sys_ad_user uk_ad_user_dn UNIQUE (ad_config_id, user_dn) —— 3 张表为真 UNIQUE 约束。
  - sys_ad_computer 无 UNIQUE 约束行，但 pg_indexes 显示唯一索引
    uni_sys_ad_computer_config_name UNIQUE (ad_config_id, computer_name)，
    另有 idx_ad_computer_dn 普通 (ad_config_id, distinguished_name) 索引。
  - 代码侧 computer.go:676 ON CONFLICT (ad_config_id, computer_name) 与该唯一索引
    完全对齐（注释「使用数据库中实际的唯一约束 (ad_config_id, computer_name)」属实），
    复活路径（deleted_at 入 DoUpdates）依赖的冲突目标真实存在。
  结论：A1 假设成立——user/ou/group 以约束形态、computer 以唯一索引形态；
  冲突目标列与代码 ON CONFLICT 一致，无重复行风险。
result: [pass]

## Summary

total: 4
passed: 4
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

部署整体性提醒（承 61-02/61-VERIFICATION）：61-01/02/03 三个 plan 的产物必须**整体部署**——清理 writer（61-01）与复活路径（61-02）分隔部署会使软删行复现撞唯一约束。