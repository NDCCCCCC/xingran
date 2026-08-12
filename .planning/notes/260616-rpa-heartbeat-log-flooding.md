# RPA 心跳请求操作日志污染 — 调研与设计决策

**调研日期**: 2026-06-16
**触发问题**: 用户报告 RPA 工作节点的定时心跳请求每分钟发送一次，导致 `sys_oper_log` 表被大量无意义审计行填充
**调研手段**: 双 gsd-phase-researcher 并行调研（handler 定位 + operlog helper 能力边界）

---

## 1. 症状校准

| 用户描述 | 调研校正 |
|---|---|
| "每分钟发送一次" | **实际是每 30 秒/Worker**（`rpa-worker/configs/config.yaml:33` `heartbeat_interval: 30s`），比用户感知更频繁 |
| "系统定时发送" | 调用方是**外部 rpa-worker Agent 进程**（`rpa-worker/internal/communication/api_client.go:61-62` 的 `heartbeatLoop`），不是后端 cron |
| "导致操作日志被大量填充" | 准确。N 个 Worker 每小时产 ≈120N 条 `business_type=0`（OperTypeOther）的审计行，且无 user 归属（公开端点，`GetUsernamePtr` 返回 nil） |

## 2. 调用 operlog.Record 的 RPA handler 全清单

| 端点 | 文件:行 | 鉴权 | OperType | 实际写入动作 | 审计必要性评估 |
|---|---|---|---|---|---|
| `POST /rpa/workers/register` | `internal/api/v1/rpa/worker_handler.go:73-87` | 公开 | `OperTypeRegister` | INSERT/UPSERT `sys_rpa_workers` | **高** — Worker 上下线生命周期事件 |
| `POST /rpa/workers/:id/heartbeat` | `internal/api/v1/rpa/worker_handler.go:89-110` | 公开 | `OperTypeOther` | UPDATE `sys_rpa_workers` (last_heartbeat/current_tasks/status) | **低** — 纯状态续期，无业务字段变更 |
| `POST /rpa/workers/progress` | `internal/api/v1/rpa/worker_handler.go:112-126` | 公开 | `OperTypeOther` | UPDATE `sys_rpa_executions` + 截图文件 + WebSocket 推送 | **中高** — 任务进度审计轨迹（screenshots、错误信息） |
| `POST /rpa/workers/:id/scale-up` | `worker_handler.go:128-164` | 鉴权 | `OperTypeStatus` | Redis Pub/Sub | 中（手动触发） |
| `POST /rpa/workers/:id/scale-down` | `worker_handler.go:166-202` | 鉴权 | `OperTypeStatus` | Redis Pub/Sub | 中（手动触发） |
| `POST /rpa/workers/scale-all` | `worker_handler.go:204-236` | 鉴权 | `OperTypeStatus` | Redis Pub/Sub | 中（手动触发） |
| `POST /rpa/workers/autoscale/config` | `worker_handler.go:262-273` | 鉴权 | `OperTypeUpdate` | **无实际持久化**（handler 内 `// TODO` 注释） | 低（仅记日志） |

**结论**：**heartbeat 端点是核心污染源**，应优先排除；progress 建议保留（审计轨迹有价值）；register 必保留（生命周期）。

## 3. operlog helper 现状能力边界

### 3.1 `Record` 签名锁定（Phase 34 显式设计）

```go
func Record(c *gin.Context, operLogSvc Recorder, db *gorm.DB, module string, operType int, opts ...RecordOption)
func RecordWithBody(c *gin.Context, operLogSvc Recorder, db *gorm.DB, module string, operType int)
```

- **无条件写入**：`Record` 函数体（`operlog.go:172-212`）无 path/method/user-agent 过滤
- **设计哲学**（`34-01-PLAN.md:53-55`、`34-REVIEW.md:280-287`）："日志覆盖度是审计要求，路径过滤应在 handler 层做，helper 不做隐式跳过"
- **回归守护**：`regression_test.go` 锁定 Record 6 参数、RecordWithBody 5 参数、25 OperType 常量、34 敏感关键词

### 3.2 旧中间件 `pkg/middleware/oper_log.go` — 已死代码

- 含 `ExcludePaths`（`oper_log.go:51-58`）和 `shouldLogOperation` 过滤函数
- **但 `SetOperLogInfo` 全代码库 0 调用**（`260615-oper-log-coverage-audit.md:108-117` 审计结果）
- Phase 34 完成后所有 handler 显式调 `operlog.Record`，绕过该中间件
- 模糊匹配（`strings.Contains` 匹配 `/list`）会误伤真实写操作

### 3.3 配置参考：`security.request_encryption.exclude_paths`

`configs/config.yaml:86-95` 已存在的 exclude 配置项（用于 SM2+SM4 加解密），含通配符风格：

```yaml
request_encryption:
  enabled: true
  exclude_paths:
    - "/api/v1/rpa/workers/register"
    - "/api/v1/rpa/workers/*/heartbeat"
    - "/api/v1/rpa/workers/progress"
```

匹配实现：`pkg/middleware/request_decryption.go:294-315` 的 `isExcludedPath`，使用 `filepath.Match` + `/*` 后缀通配。**这是 operlog exclude 的直接风格蓝本**。

## 4. 四种切入点对比

| 切入点 | 位置 | 改动面 | 优点 | 缺点 |
|---|---|---|---|---|
| **A** handler 层 | 在 heartbeat handler 移除 `operlog.Record` | 1 handler | 零框架改动、符合 Phase 34 哲学 | 无集中审计清单、漏改易发 |
| **B** operlog 包级变量 | 新增 `var ExcludedPaths []string` + `RecordIf` | 1 包 + bootstrap | 集中策略 | 引入可变包级状态、有测试污染风险 |
| **C** 配置驱动白名单（**推荐**） | 新增 `operlog.exclude_paths` 配置 + bootstrap 注入 | 5 文件 + 1 测试 | 集中可观测、风格与 SM2+SM4 对齐、可热观察 | 需把 operlog 包从叶子包升级为接受配置注入 |
| **D** Service 层 | `RecordAsync` 内过滤 | 1 服务 | 不影响 handler 调用 | **与 Phase 34 哲学严重冲突**，且 `RecordAsync` 不接收 gin.Context |

## 5. 设计决策

**采纳 Option C（配置驱动白名单）**，理由：

1. **集中可观测**：所有排除端点在一处可见，便于审计与运维
2. **风格一致性**：与 `security.request_encryption.exclude_paths` 完全对齐，新人无需学习第二种匹配语义
3. **不破坏回归测试**：`Record`/`RecordWithBody` 签名不变，OperType 常量不变，敏感关键词不变
4. **未来可扩展**：若后续有别的端点需要排除（如健康检查、metrics 上报），加配置即可
5. **保留 handler 决策权**：handler 仍显式调用 `Record`，跳过决策由配置中心化，但 record 行为的"显式"语义不丢

### 5.1 改动清单

| # | 文件 | 改动 |
|---|---|---|
| 1 | `configs/config.yaml` | 新增 `operlog.exclude_paths: []` |
| 2 | `configs/config.dev.yaml` | 同上 |
| 3 | `internal/utils/operlog/operlog.go` | 新增包级 `var ExcludedPaths []string` + `Configure(paths)` + `IsExcludedPath(path)`（用 `filepath.Match` + `/*` 通配） |
| 4 | `internal/utils/operlog/operlog.go` | `Record` 首行加 `if IsExcludedPath(c.Request.URL.Path) { return }` |
| 5 | `internal/utils/operlog/operlog.go` | `RecordWithBody` 同样处理（**必须**在 `c.GetRawData()` 之前） |
| 6 | `internal/core/core.go` | 配置加载后调 `operlog.Configure(core.Config.OperLog.ExcludePaths)` |
| 7 | `internal/utils/operlog/regression_test.go` | 新增 `TestExcludedPathsEarlyReturn`：mock Recorder，断言路径命中时 `RecordAsync` 不被调 |
| 8 | `internal/utils/operlog/operlog_test.go` | 新增 `TestIsExcludedPath`：字面量、`/*` 通配、无匹配、exact match |

### 5.2 风险点

1. **启动顺序**：`operlog.Configure(...)` 必须在任何 handler 调用 `Record` 之前。`core.go` 配置加载后立即调一次即可。
2. **路径匹配误伤**：`/*` 是 `filepath.Match` 单段通配，不会跨 `/`，与 SM2+SM4 行为一致。
3. **运行时热改**：当前不实现。如运维需要，新增 `Refresh()` 函数即可。
4. **`progress` 端点**：建议**不排除**（任务进度有审计价值）；若业务后续确认无价值，加一行配置即可。

### 5.3 改动量评估

- ~50 行新代码（operlog 包内）
- ~5 行配置
- ~80 行新测试
- 总改动 **3 文件 + 2 配置 + 1 测试文件**

## 6. 关联发现（顺手清理）

调研中发现的两个非主线问题，列入 deferred items：

1. **rpaApi.ts 路径错配**（`xingran-react-frontend/src/lib/rpaApi.ts:262`）：`progress` 端点前端写成 `/${id}/progress`，后端注册的是 `/workers/progress`（无 id 段）
2. **`/rpa/workers/autoscale/config` (POST) 无持久化**（`worker_handler.go:269` 注释 `// TODO: 保存配置到数据库`）：仅记审计日志却无业务写入

## 7. 后续路由

本设计明确后，路由至 `/gsd:plan-phase` 生成完整 PLAN.md（含 Nyquist validation、UAT criteria、verification plan），供 v1.16 milestone 引用。

---

**参考文件清单**:
- `internal/api/v1/rpa/worker_handler.go` (全部 RPA 公开端点)
- `internal/api/v1/rpa/rpa_router.go:10-78` (路由声明)
- `internal/api/router.go:841-857` (路由挂载，含公开组与鉴权组)
- `internal/utils/operlog/operlog.go` (核心 helper)
- `internal/utils/operlog/regression_test.go` (公共 API 表面锁定)
- `pkg/middleware/oper_log.go` (死代码参考)
- `pkg/middleware/request_decryption.go:292-315` (排除路径实现蓝本)
- `configs/config.yaml:86-95` (现有 exclude_paths 风格参考)
- `.planning/milestones/infra-phases/34-oper-log-full-coverage/` (Phase 34 设计哲学来源)
- `.planning/notes/260615-oper-log-coverage-audit.md` (前置审计)
- `rpa-worker/internal/communication/api_client.go:61-62` (心跳调用方)
- `rpa-worker/configs/config.yaml:33` (心跳频率配置)
