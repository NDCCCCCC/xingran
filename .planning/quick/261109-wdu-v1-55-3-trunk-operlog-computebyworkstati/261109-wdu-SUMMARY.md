---
quick_id: 261109-wdu
status: complete
completed: 2026-11-09
scope: v1.55 收档挂账 A 类代码缺口全量（3 项低风险代码 + A-2 脚本合并 + A-3 豁免登记）
commit: 6c21bf47  # 2026-11-10 用户授权提交；规划文档另 commit
---

# Quick 261109-wdu · SUMMARY

## 结论

清偿 v1.55 SHIPPED 收档挂账中 **A 类代码缺口全量**：
- 第一轮：3 项边界清晰的低风险代码缺口（A-1 / A-4 / A-5 的真实子集），含 2 组红绿回归断言；
- 第二轮（用户裁定后）：**A-2** 两份 e2e 脚本物理合并到 `tests/e2e/` 唯一权威；**A-3** `physical_e2e_test.go` 归位按 AGENTS.md 登记「就近单测合规豁免、不迁移」。

B 类需产品决策、C 类验证/人工项保持挂账、不伪造闭环。`go build ./...` + CI 口径全量
`go test ./internal/... ./pkg/... ./cmd/...` 全绿；另补 `tests/e2e/` 两种 e2e 口径验证。

## 改动清单（第一轮 7 个文件；第二轮 A-2/A-3 另见专节）

| 文件 | 改动 |
|---|---|
| `internal/api/v1/network/port_write_trunk_handler.go` | `TrunkWrite` 成功路径 `response.Success` 前补 `operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModulePortWrite, operlog.OperTypeUpdate)`；import 加 `internal/utils/operlog` |
| `internal/utils/operlog/handler_coverage_test.go` | 从 `knownGapHandlers` 摘除 `network/port_write_trunk_handler.go`（该测试有反向断言：补了 operlog 后必须摘除，否则 CI 红）；清单保留为空并更新注释，反向漂移机制继续生效 |
| `internal/services/asset/reconciliation_service.go` | `computeByWorkstation` 循环内 `assetItems = append(...)` 从 matcher 覆盖之前移到之后（纯顺序修正） |
| `internal/services/asset/reconciliation_append_order_test.go`（新） | 红绿回归断言：matcher 覆盖的 `rule-old` / `no_alert` 必须进入 `resp.Assets`（缺陷形态下残留 MV 的 `rule-1` / 空 actions → 红） |
| `internal/services/asset/reconciliation_cache_contract_test.go` | 更正立项期「生产既有缺陷 / 禁区禁止改」的过期注释为「已清偿」 |
| `internal/services/device_discovery_service.go` | `ExecuteDiscovery` 3 处 `Save` 全部检查 `.Error`（执行中 / failed / success）；`persistDiscoveryResults` 改 `Transaction`，Delete+Create 全走 `tx` |
| `internal/services/device_discovery_service_atomic_test.go`（新） | 红绿回归断言：注入 `sys_device_discovery_result` Create 失败后，旧结果 2 条必须因事务回滚完好（非事务形态下变 0 条 → 红） |

## 逐项验证

| 项 | 验证 | 结果 |
|---|---|---|
| A-1 trunk operlog | `go test -count=1 ./internal/utils/operlog/ ./internal/api/v1/network/` | ✅ 两包全绿；`HandlerDifferential` 不再产生 `KNOWN_GAP_*` Skip 子测试 |
| A-4 append 顺序 | 新测试 `TestComputeByWorkstation_MatcherOverrideReachesResponse` + `go test ./internal/services/asset/` | ✅ 绿；红绿自检：临时挪回 append → 精确变红（`rule-1` / 空 actions），恢复后复绿 |
| A-5 发现导入 | 新测试 `TestPersistDiscoveryResults_AtomicRollback` + `go test ./internal/services/ ./internal/api/v1/network/` | ✅ 绿；红绿自检：临时去掉事务 → 旧结果变 0 条（红），恢复后复绿 |
| 全量闸门 | `go build ./...` + `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` | ✅ **BUILD_OK** + 全部包 ok（含 `internal/core` 224s / `internal/services` 129s / `portwrite` 129s 等） |
| A-2 收敛（第二轮） | `SKIP_E2E=1` / `SKIP_LIVE=1` 两种口径跑 `./tests/e2e/` + `bash -n` + `go build ./...` | ✅ PASS / ok / 语法 OK / BUILD_OK，全仓无断链 |
| A-3 豁免（第二轮） | `AGENTS.md` 文档登记（零代码改动） | ✅ 合规例外已记明，`go build ./...` 不受影响 |

零 `RED-GREEN PROBE` 残留（已 grep 确认）。

## 实证纠偏（不伪造闭环）

- 收档登记 v1.54 ⑤ 列的 `user_id.(string)` 硬断言：全 `internal` 搜 `.(string)` **零命中**，当前代码已不存在 → 属过时登记，本轮**无代码可修**，如实登记为「过时登记」，不假装修了 3 处。
- A-5 真实修的是：`persistDiscoveryResults` 未开事务（Delete+Create 非原子）+ `ExecuteDiscovery` 两处 `Save` 未检查 `.Error`（第三处 success 本就有检查，保持）。

## A-2 / A-3 第二轮处置（用户 2026-11-10 裁定）

### A-2 收敛到 `tests/e2e/` 唯一权威（已物理合并）
- **删**：`scripts/operlog_e2e_verify.sh`、`scripts/e2e/operlog_e2e_verify_test.go`（`scripts/e2e/` 空目录一并移除）。
- **保留**：`tests/e2e/operlog_e2e_verify.sh` + `tests/e2e/operlog_e2e_verify_test.go`（driver 本就指向同目录脚本，零改动）。
- **文档同步**：`scripts/README.md`（目录树删 `e2e/`、e2e 段改为指向 `tests/e2e/`、顶层登记表两行移除、重复副本处置段改为「已物理合并」）；`AGENTS.md:37` 从「顶层在用脚本」列表移除 `operlog_e2e_verify.sh`；`internal/utils/operlog/handler_coverage_test.go:5` 注释改为唯一权威路径。
- **历史 phase 文档**（145-PLAN/SUMMARY、MILESTONE-v1.52-SUMMARY 等）按「收档时点快照」保持只读，不回改；`MILESTONES.md` / `ROADMAP.md` / `STATE.md` 以「后续处置」注记追加结论。
- **验证**：`SKIP_E2E=1 go test -v ./tests/e2e/` → 可观测 SKIP + PASS；`SKIP_LIVE=1 go test -run TestE2EAllEndpointsLogged ./tests/e2e/` → ok；`bash -n tests/e2e/operlog_e2e_verify.sh` → OK；`go build ./...` → OK；`go list ./... | grep scripts/e2e` → 无该包。全仓无断链引用（仅 README 的「已删除」说明文字提及旧路径）。

### A-3 登记就近单测合规豁免、不迁移
- 实证：`physical_e2e_test.go` / `port_write_e2e_test.go` 依赖 9 个同包私有 fixture 符号 + 未导出生产类型（`portWriteServiceImpl` / `portWriteExecutor` 等），Go `_test.go` 符号无法被其它包 import → 硬搬 `tests/e2e/` 编译失败。
- 在 `AGENTS.md` 测试目录段新增「就近单测豁免」条目：按「Go 单测就近放业务包内」判定为合规例外，保留、不迁移；真迁移须先抽共享 fixture 并单独立项。消除「待归位」悬置表述。
- 零代码改动、零迁移、零编译风险。

## 保持挂账（本轮不动）

- **B 类**：告警「标记已读」/ 企微钉钉路线 B / `sys_object_id` 落库 / 批量自动回滚 / `RequiresInterfaceView` 联动、forbidigo 能力边界、两 handler HTTP 透传口径 → 需产品/设计决策，保持挂账。
- **C 类**：79 条人工验证跟踪表 0 勾选、FIX-01/04/09/13、`TEST_PG_DSN` 双跑前置、149/150 UAT 重跑窗口 → 验证/人工项，保持挂账。

## hard constraints 复核

- ✅ Handler-Service / `response.Success()/Error()` / 写操作 success path 在 `response.Success` 前 `operlog.Record`
- ✅ SQLite+PG 双方言（纯 GORM / `Transaction` 内走 `tx`，沿用 Phase 142 死锁红线）
- ✅ 零新依赖、零新 migration、零裸 `c.JSON`
- ✅ 测试就近 `*_test.go`；构建产物未落仓库根
- ✅ **2026-11-10 用户已授权提交**：代码/规范文档落 `6c21bf47`，规划产物另 commit；提交前已展示 diff 且全量闸门全绿
