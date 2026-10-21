---
phase: quick-261021-iqq
plan: 01
subsystem: scheduler
tags: [ad, ad_push_queue, drain, hot-reload, cron, sys_config, sanitize]

requires:
  - phase: "quick-261020-d0d"
    provides: "项目基线（main 未含排水热重载，Phase 58 D-05 单旋钮启动读一次）"
provides:
  - "ad_push_queue.drain_interval_seconds 热重载：改 sys_config 无需重启，≤30s 生效（方案A：固定 30s tick + drainGate 门控）"
  - "drainGate.shouldDrain(now, interval)：lastDrain/lastInterval 门控纯逻辑，零值首排 / 到期放行 / 间隔变化重置并打日志"
  - "DefaultDrainIntervalSeconds 导出常量：scheduler 与 addomain 共用默认 30s 的单一事实源"
  - "缺陷X 修复：markFailed/markDead 入口统一 sanitizeFailureReason，last_error 恒为合法 UTF-8（0x00 → PG SQLSTATE 22021 → UPDATE 整体失败 → 每 tick 无限重试，已被回归用例锁死）"
affects: [internal/scheduler, internal/services/addomain, internal/core/db/migrations]

actuals:
  tokens: 41000
  tasks: 4
  commits: 0  # 提交待用户确认（repo 规则：提交前必须问用户）

tech-stack:
  added: []
  patterns:
    - "robfig/cron 不支持运行时改 spec → 固定最小 tick + 应用层门控（每 tick 读配置 + 距上次执行不足 interval 则跳过），代价是 >tick 粒度的间隔有 ≤30s 抖动"
    - "错误报文入 DB 前必须过 sanitizeFailureReason（ToValidUTF8 + 去 0x00 + 截断 1000）：LDAP err.Error() 实测含 NUL"

key-files:
  created:
    - "internal/scheduler/ad_sync_tasks_drain_gate_test.go"
  modified:
    - "internal/scheduler/ad_sync_tasks.go"
    - "internal/services/addomain/ad_push_queue_drain_service.go"
    - "internal/services/addomain/ad_push_queue_drain_test.go"
    - "internal/core/db/migrations/migration_216_adpush_drain_sysconfig.go"

key-decisions:
  - "方案A（固定 */30 tick + drainGate 门控）而非 robfig/cron Reschedule：cron v3 无运行时改 spec 的安全原语；门控纯逻辑可零依赖单测，多实例部署各实例独立读 DB 自动收敛（Test 3 实锤 10.62.10.39/33 双实例排水）"
  - "LoadDrainIntervalSeconds 由启动读一次改为每 tick（30s）调用且成功路径不打日志：30s 一条会刷屏，值变化日志上移到 scheduler drainGate.shouldDrain（只在配置变化时打一条 Infof）"
  - "删除 SecondsToCronSpec 映射层及其 2 个测试：固定 tick 后无调用方；F-E 的 6 段 spec 步进陷阱知识保留在 git 历史与 PLAN 中"
  - "缺陷X 修复点选在 markFailed/markDead 入口而非 3 处调用点：入口单点覆盖所有 err.Error() 直拼路径，与账号池 MarkFailure 同款 sanitizeFailureReason 同包直调"

patterns-established:
  - "间隔类 sys_config 旋钮的热重载形态：固定最小 tick + shouldDrain 门控（drainGate 可复用于后续同类旋钮）"
  - "守护测试放独立文件（ad_sync_tasks_drain_gate_test.go）并用最小 fixture（NewScheduler(nil)）锁注册表，防重构误删"

requirements-completed: [DRAIN-HOTRELOAD-01, DRAIN-GUARD-01, DEFECT-X-SANITIZE-01]

debt:
  - "migration 216 seedRemark 只对新环境生效：存量环境 sys_config 里的旧 remark（含『修改后需重启后端生效』）不会自动更新，如需对齐可手工 UPDATE（非阻断）"
---

# 261021-iqq: ad_push_queue 排水间隔热重载 + 缺陷X sanitize

## What Was Built

Phase 58 D-05 收尾（quick 261021-iqq，源自 verify-work 57 UAT 两个缺陷档案）：

1. **DRAIN-HOTRELOAD-01 排水间隔热重载**（方案A，2026-10-21 用户裁决）：
   - `internal/scheduler/ad_sync_tasks.go`：删启动时 `LoadDrainIntervalSeconds → SecondsToCronSpec` 映射；cron 固定 `*/30 * * * * *` tick；新增 `drainGate{lastDrain, lastInterval}` + `shouldDrain(now, interval)`——每 tick 读 `sys_config`（s.db 为 nil 时兜底默认 30s），距上次实际排水 ≥ interval 才 `DrainOnce`；间隔变化打一条 Infof（热重载生效日志）；零值语义 = 启动后首个 tick 立即排水。
   - `internal/services/addomain/ad_push_queue_drain_service.go`：`DefaultDrainIntervalSeconds = 30` 导出（scheduler 引用同一默认值）；`LoadDrainIntervalSeconds` 注释改为"每 tick 调用、无需重启"，删成功路径 Infof；删 `SecondsToCronSpec`。
2. **DRAIN-GUARD-01 taskRegistry 防删守护**：`internal/scheduler/ad_sync_tasks_drain_gate_test.go`——`drainGate` 门控四用例（零值首排 / interval=300 内 9 个 tick 全跳过 / 间隔变化重置 / interval<=0 兜底）+ `ad_data_sync` / `ad_account_pool_recover_breakers` 注册守护（InvokeTarget 白名单 F-08 依赖）。
3. **DEFECT-X-SANITIZE-01 缺陷X 修复**：`markFailed`/`markDead` 入口统一 `msg = sanitizeFailureReason(msg)`（account_pool.go:359 同款：ToValidUTF8 + 去 0x00 + 截断 1000），覆盖 :581/:298/:390 三处 err.Error() 直拼调用点；新增 `TestPhase58_MarkFailed_SanitizesNulInLastError` 回归用例（markFailed + markDead 各一例，构造实测 LDAP 50 报文含 \x00 与非法 UTF-8，断言 last_error 不含 0x00 且 utf8.ValidString）。
4. **migration 216 remark 对齐**：seedRemark 去掉『修改后需重启后端生效』改为『修改后约30秒内生效(无需重启)』（幂等 seed 只对新环境生效，存量环境需手工 UPDATE——记入 debt）。

## Verification

- `go build ./...` ✅
- `go test ./internal/scheduler/` ✅（含新 drain_gate_test 4+1 用例）
- `go test ./internal/services/addomain/` 全包 ✅（含新 NUL 回归用例 + 保留的 LoadDrainIntervalSeconds_FallsBackOnGarbage 五子用例）
- 删除 `TestPhase58_SecondsToCronSpec_Valid/_InvalidFallsBack`（被删函数，随函数一起移除；cron import 一并清理）

## Commits

- 待用户确认后提交（repo 规则：提交前必须问用户）。
