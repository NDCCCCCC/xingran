---
status: investigating
trigger: "请确认为什么mac历史物化视图定时任务每次都执行两次？"
created: 2026-07-03T00:00:00Z
updated: 2026-07-03T00:00:00Z
goal: find_root_cause_only
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

(等待 gsd-debugger 填充)