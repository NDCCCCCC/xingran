---
one-liner: AD push 队列 enable 方向 + SAFE-01 身份校验/双向压制、全量同步指数退避节流、常态化同步漂移告警服务
---

# Phase 134 — AD 同步域（v1.53 / FIX-01 + FIX-10 + FIX-11）

**一行摘要**：补齐 AD push 队列 enable 方向（含 SAFE-01 身份校验与双向压制）、落地全量同步指数退避节流、把一次性采样脚本升级为常态化同步漂移告警服务。

**执行**：2026-11-08，独立智能体 phase-134；与 135/136/137/139/140 并行（按模块隔离），与 138 因同触 `internal/scheduler/` 串行。

## 交付清单

| REQ | 文件 | 内容 |
|---|---|---|
| FIX-01 | `internal/models/ad_push_queue.go:28-33` | 新增 `ADPushActionEnable`（追加新值，3 个既有常量不变） |
| FIX-01 | `internal/services/addomain/ad_push_queue_service.go` | `suppressOppositeAction` 双向压制（最新意图胜出）、`CheckRestoreIdentityForEnable`（SAFE-01）、`EnqueueEnableOnStatusTransition` / `EnqueueEnableOnRestore` |
| FIX-01 | `internal/services/system/user_service.go`、`user_sync_service.go` | Update/UpdateStatus 1→0、登录恢复、批量恢复四处同事务入队 enable |
| FIX-01 | `internal/services/addomain/ad_push_queue_drain_service.go:420-431` | `drainRow` enable 分支 → `EnableUser`（UAC 544 幂等） |
| FIX-10 | `internal/scheduler/ad_sync_backoff.go`（新建） | sys_config 4 旋钮热重载 + 指数退避纯函数 + 并发安全状态机 |
| FIX-10 | `internal/scheduler/ad_sync_tasks.go` | 每 tick 门控 + 成功/失败 record |
| FIX-11 | `internal/services/addomain/ad_sync_drift_alert_service.go`（新建） | 漂移判定 + applogger / sys_oper_log / sys_notice 三通道 + 5 旋钮 + 节流 + cron 接入 |
| 测试 | `phase134_ad_push_enable_test.go`（14+9 例）、`ad_sync_backoff_test.go`（9 例）、`ad_sync_drift_alert_service_test.go`（11 例） | 43 个新用例，含「连续失败 36 tick 仅 6 次触发」 |
| 兼容 | `ad_push_queue_test.go`、`ldap_client_mock_test.go`、`user_sync_service_test.go` | 旧 v1.21「1→0 不入队」断言同步到新契约 |

## 验证
- `go build ./...` exit 0
- `go test -count=1 ./internal/services/addomain/ ./internal/services/system/ ./internal/scheduler/` 三包全绿
- `go test ./...` 全仓全绿，零 FAIL / 零 panic

## 边界（如实声明）
- 实机 AD/LDAP UAT（UAC 544 + AD 登录恢复）未执行，仅 sqlite 自动化用例，归入 VAL 跟踪表。
- FIX-11 未做前端告警页 / WS 广播 / 企业微信钉钉 webhook；sys_config 旋钮无 seed（缺省即默认）。
