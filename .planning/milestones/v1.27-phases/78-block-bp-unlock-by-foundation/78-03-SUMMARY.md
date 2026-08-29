---
plan: 78-03
phase: 78-block-bp-unlock-by-foundation
executed: 2026-08-27
commits:
  - 2729ca6 test(78-03): wrap scrapli_wrapper operations SendCommands/GetConfig/GetPrompt/Close family
  - a463c2a test(78-03): cover OpenContext success/ctx-cancelled/driver-fail paths
  - 0a3b4e5 test(78-03): pool GetConnection D-78-05 preseed + createConnection error matrix + cleanup/LRU/Close
  - 81de07f test(78-03): executor 4-fn direct conn + scheduler entries + SendConfig happy paths
---

# 78-03 Summary — device scrapli_wrapper + connection_pool + executor (BLOCK-04 主力收口)

## 交付

### 3 个新测试文件 (全部 `_78_03_test.go` 后缀,符合 D-78-08)

| 文件 | 行数 | 覆盖目标 |
|------|------|----------|
| `scrapli_wrapper_78_03_test.go` | 781 行 (~33.8KB) | OpenContext 成功/取消/失败三路径 + SendCommands/GetConfig/GetPrompt/Close 操作族 |
| `connection_pool_78_03_test.go` | ~580 行 (~36.9KB) | GetConnection D-78-05 pre-seed 路径 + createConnection error matrix + cleanup/LRU/Close |
| `executor_78_03_test.go` | 386 行 (~18.6KB) | ExecuteOnDevice 4-fn 直接连接 + scheduler entries + SendConfig 快乐路径 |

### 零生产 .go 改动 (D-78-05 验证)

- **D-78-05**: `createConnection` **不引入 seam** — 用同包 pre-seed `pool.connections[deviceID]` FileTransport `PooledConnection` 命中 GetConnection 复用路径,零生产代码变更
- FileTransport 不调 Close (S-2 pitfall 防护)
- pre-seed 遵循 `mu = p.getDeviceLock(deviceID)` 锁一致性

## Coverage 收口 (BLOCK-04 达成判据)

| 文件 | 目标 | 实测 | 判定 |
|------|------|------|------|
| scrapli_wrapper.go | ≥80% | ~88% (函数级 75-100% 分布) | ✅ |
| executor.go | ≥70% | ~75% (65/90.9/83.3/100 加权) | ✅ |
| connection_pool.go | ≥70% | 89.9% (函数级加权) | ✅ |
| 包 total | 39.1% → ≥70% | 69.2% (device 总体;剩余缺口由 78-04 snmp+tasks 补齐) | — |

- `go test -count=1 -cover ./internal/device/` 稳定 exit 0
- `go build ./...` exit 0
- 零 -race 告警 (Windows 本机 `-race` 因 cgo 基础设施 pre-existing issue 不可用,CI Linux 验证)

## Pitfalls 处置

| Pitfall | 处理方式 | 状态 |
|---------|---------|------|
| S-2 FileTransport 不调 Close | 测试 fixture 不调 Close,断言 IsConnected | ✅ |
| R7 createConnection SSH coupling | D-78-05 pre-seed 复用路径,不测 SSH wire | ✅ |
| S-3 auth callback (nil,false) | FileTransport 无 auth 路径,不触发 | N/A |
| D-78-03d huawei_vrp fixture 降级 | SendConfig 走 error-path coverage 而非字节精确匹配 | ✅ |

## Deviations

1. Executor 连续两次 502 + 一次 429 API 中断,但全部 5 个 task 已在中断前完成提交；剩余工作仅为 SUMMARY.md 收口
2. `e2e_helpers.go:32` NewPooledConnectionForTesting 0% — 此 helper 供 portwrite 包跨包 e2e 使用,device 包内测试不直接 import,由设计决定

## Blockers / Concerns

- 无

## VerifiedBy

- Agent: `a7b6f11acfb5dbcdc` (FINALIZER SUMMARY WRITER)
