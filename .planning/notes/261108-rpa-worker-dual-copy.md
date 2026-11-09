# 261108 rpa-worker 双副本漂移登记（Phase 137 / FIX-08）

> 结论：**不合并，保留两份，显式登记差异与职责边界**。
> 本文是 OPEN-ITEMS.md B11 的收敛结论（「合并 or 显式登记」二选一中的后者）。

## 一、两个子模块

| | 根副本 | 前端副本 |
|---|---|---|
| 路径 | `rpa-worker/` | `xingran-react-frontend/rpa-worker/` |
| go.mod | 独立 `module github.com/xingran-next/rpa-worker`（go 1.21） | 同名独立 module（go 1.21） |
| 可独立运行 | 是，有 `cmd/main.go`、`configs/`、`deployments/` | **否**，无 `cmd/main.go`、无 `configs/`、无 `deployments/` |
| 定位 | 生产可部署的 RPA Worker（完整实现） | 随前端仓库存放的早期骨架/参考副本（长期未跟进） |

两份 module path 完全相同但物理隔离，Go 工具链各自解析，互不感知；`go build ./...` 必须分别在两个目录下执行。

## 二、逐项差异对照（治理后）

### 2.1 文件级差异（治理后行数）

| 文件 | 根副本 | 前端副本 | 说明 |
|---|---|---|---|
| `go.mod` | 40 行 | 12 行（治理后补齐） | 前端副本治理前缺 go-rod / go.sum，基线即编译失败 |
| `internal/config/config.go` | 352 行 | 148 行 | 根副本 `BrowserConfig` 含 `DevTools/SlowMo/NavigationTimeout/UserAgent/ChromePath/ChromeFlags`；前端副本仅 5 字段（治理后补 3 个必需字段） |
| `internal/logger/logger.go` | 153 行 | 146 行 | `Logger` 接口不同：根副本无 `Fatal`，前端副本有 `Fatal`；`Err` 语义不同（见 2.3） |
| `internal/types/types.go` | ~221 行 | ~100 行 | **协议不兼容**：动作类型数量 21 vs 8；`ActionWaitFor` 值 `waitFor` vs `wait_for`；状态 `StatusCancelled` vs `StatusCanceled` |
| `internal/browser/pool.go` | 168 行 | ~160 行 | `Page` 接口 15 方法 vs 8 方法（前端副本缺 `GetText/GetValue/GetHTML/GetAttribute/ScrollDown/ScrollUp/ScrollToElement`） |
| `internal/browser/chrome_page.go` | 375 行 | 治理后新建 183 行 | 根副本用 go-rod `Must*` API；前端副本移植版用 error 版 API（不 panic） |
| `internal/browser/page_manager.go` | 208 行 | 115 行 | 根副本有 `SelectorEventCollector` 埋点（TIDY-03 契约）；前端副本无 |
| `internal/browser/selector_events.go` | 123 行 + 测试 | **缺失** | 选择器明细上报契约，根副本独有 |
| `internal/executor/engine.go` | ~752 行 | 273 行 | 根副本含 loop/pause/condition/autologin/变量提取/重试风暴治理等；前端副本仅 8 个基础动作 |
| `internal/executor/task_splitter.go` | 489 行 | **缺失** | 混合模式子任务拆分，根副本独有 |
| `internal/worker/worker.go` | 641 行 | 3 行（占位） | 前端副本治理前是 1 字节空文件（无 package 声明） |
| `internal/worker/health_server.go` | 150 行 | **缺失** | 健康检查服务，根副本独有 |
| `internal/communication/*` | 各 100+ 行 | 各 50~100 行 | 根副本含 DEBT-03 provisioning token / 进度节流等；前端副本为简化版 |
| `cmd/main.go` | 46 行 | **缺失** | 前端副本无入口，不能独立启动 |
| `configs/` `deployments/` `docs/` | 完整 | **缺失** | 部署资产只在根副本 |

### 2.2 行为漂移（治理前）

| 项 | 根副本 | 前端副本 | 治理动作 |
|---|---|---|---|
| `Pool.createPage` | `NewChromePage(ctx, cfg, log)` 真实 Chrome（pool.go:160-168） | `return &mockPage{}, nil` stub（pool.go:148-156） | ✅ 前端副本改用 `NewChromePage(ctx, cfg, log)`，删除 `mockPage` |
| `PageManager.WaitForURL` | 首次 tick 即 `return nil`（:118） | 首次 tick 即 `return nil`（:84） | ✅ 两份均改为真实轮询：取 `window.location.href` + `path.Match`/子串校验，超时/失败返回 error |
| `PageManager.WaitForText` | 首次 tick 即 `return nil`（:137） | 首次 tick 即 `return nil`（:103） | ✅ 两份均改为真实轮询：取元素 `textContent` 判含目标文本 |
| `go build ./...` | ✅ 通过 | ❌ 基线失败（空 `worker.go`、缺 `go.sum`、`logger.Err` 双参、go-redis API 过时、`time` 未用 import、`createErrorResult` 返回值不匹配、`pooledPage.page` 越权访问） | ✅ 全部修复并通过 |

### 2.3 关键 API 语义分歧（决定不合并）

1. **`logger.Err`**：根副本 `func Err(err error) Field`（单参，自定义 `Field{Key,Value}`）；前端副本治理前误用 `zap.NamedError`（双参），全代码库按单参调用 → 前端副本曾整体无法编译。治理后前端副本改为 `func Err(err error) Field { return zap.Error(err) }`，**返回类型仍是 `zap.Field` 别名**，与根副本的自定义 `Field` 结构体不同。
2. **任务协议**：`ActionWaitFor` 字符串值不同、动作类型集合不同、状态枚举拼写不同 → 后端下发的任务在两份副本上解释不一致。
3. **Page 接口**：根副本 15 方法，前端副本 8 方法；`ChromePage` 实现的方法集不同。
4. **配置默认值**：根副本 `setDefaults(cfg)` 走「零值填默认 + 环境变量覆盖」；前端副本 `setDefaults(v)` 走 viper `SetDefault`，且 `SetEnvPrefix` 不同（`WORKER` vs 无前缀）。

## 三、为何不合并（风险裁定）

| 风险 | 说明 |
|---|---|
| **协议不兼容（阻断级）** | 合并必须统一 `ActionType` 取值与 `ExecutionStatus` 拼写；后端 `internal/services/rpa/` 与任务下发方按哪套？未经全链路回归不得改，属跨 Phase 范围。 |
| **入口/部署资产缺失** | 前端副本无 `cmd/main.go`/`configs/`/`deployments/`，无法独立启动；若「合并」为删前端副本，前端仓库里引用 `rpa-worker/` 的脚本/文档需逐一核对（本次未发现 `package.json` 直接引用，但存在 `docs/` 历史文档引用）。 |
| **公共 API 语义分歧** | `logger.Logger`/`logger.Field`/`Page` 接口在两份中签名不同，统一需改动所有 `internal/*` 调用点，回归面 ≈ 全模块。 |
| **根副本是生产实现** | 根副本 `engine.go`/`worker.go`/`task_splitter.go` 含混合模式、自动登录、选择器埋点等生产逻辑；把前端骨架「合并」进去等于用低版本覆盖高版本，方向错误。 |
| **本 Phase 范围** | Phase 137 只要求「收敛/登记漂移」，不含任务协议重构；采用显式登记是范围内最小、可逆的做法。 |

**裁定**：保留两份，前端副本降级为「随前端仓库存放的参考/骨架副本」，不作为独立可部署 Worker；后续新功能一律在根副本 `rpa-worker/` 演进。若未来确需合并，须另立 Phase：统一任务协议 → 抽公共 `internal/` 为共享 module → 删前端副本并迁移部署资产 → 全链路回归。

## 四、治理后职责边界（显式登记）

- **根副本 `rpa-worker/`**：唯一生产可部署 Worker。承接所有新功能、bug 修复、部署脚本、配置样例、契约文档。
- **前端副本 `xingran-react-frontend/rpa-worker/`**：仅保留为「可编译的参考骨架」，用于对照浏览器池/执行引擎的最简形态。**禁止**在此副本新增业务功能；本次仅做「去 stub + 可编译 + 与根副本行为对齐（createPage/Wait*）」的治理。其 `go.mod`/`go.sum` 独立维护，与根副本不共享依赖锁。
- 两份 `go build ./...` 与 `go test ./...` 各自独立执行、独立通过（见交付报告）。

## 五、治理产物清单（Phase 137 / FIX-08）

- 前端副本 `createPage` 改用真实 Chrome：新增 `internal/browser/chrome_page.go`（go-rod error 版 API），`internal/browser/pool.go` 删除 `mockPage`、补 `PooledPage.Page()`。
- 两份 `page_manager.go` 的 `WaitForURL`/`WaitForText` 改为真实轮询 + 超时/失败断言；各新增 `page_manager_test.go`（成功/轮询后命中/超时/probe 报错 8+ 用例）。
- 前端副本基线编译修复：`internal/worker/worker.go`（空文件补 package）、`internal/logger/logger.go`（var 块闭合 + `Err` 单参）、`internal/communication/redis_client.go`（v9 `XGroupCreateMkStream`/`XPending`）、`internal/communication/api_client.go`（未用 import）、`internal/executor/engine.go`（`createErrorResult` 双值返回 + `Page()` getter）、`internal/config/config.go`（补 BrowserConfig 字段）、`go.mod`/`go.sum`（go-rod 依赖）。
