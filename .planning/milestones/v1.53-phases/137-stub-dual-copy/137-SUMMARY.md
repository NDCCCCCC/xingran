---
one-liner: GetDiscoveryResults 补持久化查询链路、rpa-worker 双副本去 browser stub 并修复前端副本基线编译
---

# Phase 137 — Stub 与双副本漂移治理（v1.53 / FIX-07 + FIX-08）

**一行摘要**：把 `GetDiscoveryResults` 从恒空 stub 补成「持久化 + 查询」最小可用链路，治理 rpa-worker 双副本 browser stub 与 Wait* 假校验并修复前端副本基线编译。

## 交付清单

| REQ | 文件 | 内容 |
|---|---|---|
| FIX-07 | `internal/models/device_discovery_result.go`（新建） | `sys_device_discovery_result` 明细表模型（seq 保序 + discovery_id 索引，camelCase json tag 对齐前端） |
| FIX-07 | `internal/core/db/database.go:701` | 注册进 `MigrateModelList()` |
| FIX-07 | `internal/services/device_discovery_service.go` | `ExecuteDiscovery` 成功后 `persistDiscoveryResults`（重跑以最后一次为准）、`GetDiscoveryResults` 查表、`DeleteDiscovery` 级联清理 |
| FIX-07 | `discovery_handler.go`、`discovery_handler_test.go`、`device_discovery_service_79_06_test.go` | 删「恒空」断言，补 `empty_when_no_persisted_results` / `returns_persisted_devices` 真实结果断言 |
| FIX-08 | `rpa-worker/internal/browser/page_manager.go`（根）+ `page_manager_test.go`（新建 8 例） | `WaitForURL` / `WaitForText` 改真实轮询（含超时 / probe 失败 error 路径） |
| FIX-08 | `xingran-react-frontend/rpa-worker/internal/browser/chrome_page.go`（新建）、`pool.go:156`、`page_manager.go`、`page_manager_test.go`（7 例） | 前端副本 `createPage` 改用真实 `NewChromePage`（go-rod error 版 API），删 `mockPage`；Wait* 同根副本 |
| FIX-08 | 前端副本 `go.mod` / `go.sum`（新建）、`internal/{worker,logger,communication,executor,config}` | 治理前**本就 build 失败**的基线修复（空文件 / var 块 / go-redis v9 API / 未用 import / 返回值 / 配置字段） |
| 结论 | `.planning/notes/261108-rpa-worker-dual-copy.md`（新建） | 逐项差异对照 + 「不合并，显式登记」结论与职责边界 |

## 双副本结论：**不合并，显式登记**
1. 协议不兼容（阻断级）：`ActionWaitFor` 值 `waitFor` vs `wait_for`、动作类型 21 vs 8、`StatusCancelled` vs `StatusCanceled`。
2. 公共 API 语义分歧：`logger.Logger` / `logger.Field` / `Page` 接口 15 vs 8 方法。
3. 前端副本无 `cmd/main.go` / `configs/` / `deployments/`，不能独立启动；方向性上根副本才是生产实现。
4. 职责边界：根副本 = 唯一生产可部署 Worker；前端副本 = 可编译参考骨架，禁止再新增业务功能。

## 验证
- 主模块 `go build ./...` ✓；`go test ./internal/services/ -run Ddv7906` / `./internal/api/v1/network/` / `./internal/core/db/` / `./internal/models/` 全绿
- 根副本 `go build ./...` + `go test ./...` 全绿（含 8 个 Wait* 用例）
- 前端副本 `go build ./...` + `go test ./...` 全绿（含 7 个 Wait* 用例；治理前 build 失败）

## 边界
- `ImportDiscoveredDevices` 仍为 stub（B10 之外独立缺口，按范围约束未动，测试如实锁定）。
- 前端副本为骨架、不能独立运行；`go-rod v0.116.2` 来自本机缓存，`go mod tidy` 离线成功，零新外部依赖。
