---
quick_id: 261109-wdu
status: in_progress
created: 2026-11-09
scope: v1.55 收档挂账 A 类纯代码缺口（低风险子集）
---

# Quick 261109-wdu · 清偿 3 项真实低风险代码缺口

## 背景

v1.55 SHIPPED 收档时 `STATE.md ## Deferred Items` / `MILESTONES.md` / `ROADMAP.md Known Gaps`
挂账了一批 gaps。用户裁定先清 **A 类纯代码缺口**（B 类需产品决策、C 类验证/人工项保持 0 勾选不伪造闭环）。

实证复核（本轮）后，A 类 5 项中只有下列 3 项属"边界清晰、低风险、可直接修"：

| 项 | 证据 | 处置 |
|---|---|---|
| A-1 trunk 写端点无 operlog | `internal/api/v1/network/port_write_trunk_handler.go:79` 成功路径只有 `response.Success`，全文件零 `operlog.Record` | 补 `operlog.Record`（`ModulePortWrite` / `OperTypeUpdate`），**并**从 `internal/utils/operlog/handler_coverage_test.go` 的 `knownGapHandlers` 摘除（该文件有反向断言：补了调用后必须从清单摘除，否则 CI 红） |
| A-4 `computeByWorkstation` append 早于 matcher | `internal/services/asset/reconciliation_service.go:1076` `assetItems = append(assetItems, item)` 在 :1087-1096 的 `MatchException` 修改 `item` 之前 → per-asset matcher 结果未进响应 | 把 `append` 移到 matcher 块之后（纯顺序修正，零语义变化） |
| A-5 发现导入链路隐患 | `device_discovery_service.go:163/183` 两处 `Save` 未检查 `.Error`；`persistDiscoveryResults`（:209-245）未开事务（Delete + Create 非原子） | 3 处 `Save` 补错误检查；`persistDiscoveryResults` 改 `Transaction` 且内部全走 `tx`（sqlite 单连接死锁红线） |

**A-5 实证纠偏**：收档登记提到的 `user_id.(string)` 硬断言在当前代码**已不存在**（全 `internal` 搜 `.(string)` 零命中），属过时登记，本轮不伪造"修了 3 处"，只修真实存在的 2 类共 4 个代码点。

## 不在本轮（如实登记，不伪造闭环）

- **A-2 两份 `operlog_e2e_verify.sh` 合并**：现有 `scripts/` 顶层 + `scripts/e2e/` driver 与 `tests/e2e/` 全套两套载体；README 标顶层为权威、AGENTS.md 又要求后端 e2e 归 `tests/e2e/`。删哪边需产品/维护者裁定权威位置，非低风险机械改动 → 保持挂账。
- **A-3 `physical_e2e_test.go` 归位**：该文件是 `package portwrite`，依赖 `newTestDB` / `seedPortAndDevice` / `fileTransportExecutor` / `newTestService` 等包内私有 fixture，直接挪到 `tests/e2e/` 会编译失败；需先抽公共 fixture 或保留在业务包并登记豁免 → 保持挂账。
- **B 类**（告警标记已读 / 企微钉钉路线 B / `sys_object_id` 落库 / 批量自动回滚 / `RequiresInterfaceView` 联动、forbidigo 能力边界、两 handler HTTP 透传）：需产品/设计决策 → 保持挂账。
- **C 类**（79 条跟踪表、FIX-01/04/09/13、`TEST_PG_DSN` 双跑、149/150 UAT 重跑窗口）：验证/人工项 → 保持 0 勾选，不伪造闭环。

## 硬约束

- 遵循 CLAUDE.md：Handler-Service、`response.Success()/Error()`、写操作 success path 在 `response.Success` **之前** `operlog.Record`。
- `0=正常 1=停用`；SQLite+PG 双方言；零新依赖、零新 migration、零裸 `c.JSON`。
- 构建产物只落 `out/`；测试就近 `*_test.go`。
- **不执行 `git add` / `git commit`** —— 提交前须向用户展示 diff 并取得显式确认（CLAUDE.md Git Workflow）。

## 任务

### Task 1 · A-1 trunk operlog + 摘除 knownGap
- `files`: `internal/api/v1/network/port_write_trunk_handler.go`、`internal/utils/operlog/handler_coverage_test.go`
- `action`: 在 `TrunkWrite` 成功路径 `response.Success` 前加 `operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModulePortWrite, operlog.OperTypeUpdate)`；import 加 `internal/utils/operlog`。从 `knownGapHandlers` 删除 `network/port_write_trunk_handler.go` 条目，并把对应注释更新为"已于 261109-wdu 清偿"。
- `verify`: `go test -count=1 ./internal/utils/operlog/ ./internal/api/v1/network/`（含 `TestHandlerDifferential` 反向断言不再 Skip）
- `done`: trunk 写端点有 operlog、静态门禁零 knownGap Skip、两包测试全绿。

### Task 2 · A-4 append 顺序
- `files`: `internal/services/asset/reconciliation_service.go`
- `action`: `computeByWorkstation` 循环内把 `assetItems = append(assetItems, item)` 从 :1076 移到 :1096 matcher 块之后。
- `verify`: `go test -count=1 ./internal/services/asset/`
- `done`: per-asset `MatchException` 的 `ExceptionRuleID` / `AppliedActions` 覆盖值进入 `resp.Assets`；包测试全绿。

### Task 3 · A-5 发现导入事务 + Save 错误检查
- `files`: `internal/services/device_discovery_service.go`
- `action`: `ExecuteDiscovery` 的 3 处 `Save` 全部检查 `.Error`（:163 执行中、:183 failed、:188 success 已有检查保持）；`persistDiscoveryResults` 改 `s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {...})`，Delete 与 Create 全走 `tx`，错误用 `fmt.Errorf` 包装。
- `verify`: `go test -count=1 ./internal/services/ ./internal/api/v1/network/`
- `done`: 4 点修复到位、既有 `TestDdv7906_ExecuteDiscovery_*` 与 `TestImportDiscoveredDevices*` 全绿。

### Task 4 · 全量回归闸门 + 收档
- `verify`: `go build ./...` + `go test ./...`（CI 口径含 `./internal/... ./pkg/... ./cmd/...`）
- `action`: 写 `261109-wdu-SUMMARY.md`，更新 `STATE.md ## Deferred Items` 中 v1.55 行的 A-1/A-4/A-5 为已清偿（保留 B/C 与 A-2/A-3 挂账）、`STATE.md ## Quick Tasks Completed` 表加本行。
- `done`: 全绿、改动的 3 项有证据、未修项如实登记。
