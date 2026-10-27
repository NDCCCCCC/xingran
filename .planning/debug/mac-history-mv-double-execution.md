---
status: resolved
trigger: "请确认为什么mac历史物化视图定时任务每次都执行两次？"
created: 2026-07-03T00:00:00Z
updated: 2026-10-27T16:10:00+08:00
goal: find_root_cause_only
audit_acknowledged:
  milestone: v1.23
  at: 2026-10-26T10:00:00+08:00
  status: resolved

---

## Current Focus

hypothesis: 同一 job 被注册两次(sys_job 表双行)或者 scheduler 启动阶段的双重注册
test: (1) 搜代码中所有注册 MAC 历史物化视图刷新 job 的位置 (2) 查询 sys_job 表是否有重复 (3) 检查 startup 阶段是否有独立的 refresh goroutine
expecting: 应能定位到双重注册的位置
next_action: 开始代码搜索,从 scheduler 注册和 startup 入手

## Symptoms

expected: 任务 `MAC历史物化视图刷新` 每 5 分钟执行 1 次
actual: 每 5 分钟执行 2 次 - 同一时间戳两条 sys_job_log 记录,时长略不同(1.90s+1.44s / 1.87s+1.51s / 1.78s+1.46s / 1.81s+1.38s / 1.91s+1.40s),均"成功"
errors: 无(两条都是成功);背景统计 失败 535/9322 = 5.74%(可能是早期问题或资源争抢,但与本次问题弱相关)
reproduction: 稳定可复现 - 任意调度时刻(09:05:00, 09:00:00, 08:55:00, 08:50:00, 08:45:00 ...) 都出现双行
started: 启动即存在(待 gsd-debugger 确认何时开始)

## Eliminated

(等待 gsd-debugger 填充)

## Evidence

(等待 gsd-debugger 填充)

## Resolution

root_cause: **双实例部署**（来源：用户裁定 2026-10-27，原文「双执行是因为在 10.62.10.39 上还有一套实例一直在跑，这个问题不用管」）。同一台机器上有第二套后端实例持续运行，两个进程各自启动 scheduler 同时消费同一 DB 的 job，产生每 tick 双条 sys_job_log。**非代码缺陷**，注册层无双重注册问题。
fix: 无代码修复。Phase 73 的 cron.go 注册层调查取消——注意：STATE.md / ROADMAP.md 中 Phase 73 原本包含 #4 调查，**Phase 73 的 #4 部分随之取消，Phase 73 只剩 #5/#7**（ROADMAP/STATE 由主会话统一处理，本档仅记录）。
verification: 用户裁定确认。
files_changed: []

## 遗留约束（部署）

- **调度器必须单实例**。若未来需要多实例高可用，需引入分布式锁或 leader 选举（列为 future enhancement，不排期）。
---
