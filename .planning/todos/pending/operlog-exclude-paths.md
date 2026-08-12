---
title: operlog.exclude_paths 配置驱动白名单（解决 RPA 心跳日志污染）
date: 2026-06-16
priority: high
status: pending
origin: explore-session-260616-rpa-heartbeat-log-flooding
related_note: .planning/notes/260616-rpa-heartbeat-log-flooding.md
related_phase: pending (路由至 /gsd:plan-phase)
---

# operlog.exclude_paths 配置驱动白名单

## 目标

引入配置驱动的操作日志排除白名单机制，解决 RPA Worker 心跳请求（30s/Worker）每分钟往 `sys_oper_log` 写入大量无意义审计行的问题。同时为未来其他需要排除的高频端点（健康检查、metrics 上报等）提供统一配置入口。

## 设计要点（来自 .planning/notes/260616-rpa-heartbeat-log-flooding.md）

- **不破坏回归测试**：`Record`/`RecordWithBody` 签名不变、25 个 OperType 常量不变、34 个敏感关键词不变
- **风格一致性**：复用 `security.request_encryption.exclude_paths` 的 `filepath.Match` + `/*` 通配语义
- **配置驱动**：所有排除端点集中在 `configs/config.yaml`，便于运维审计与热观察
- **保留显式语义**：handler 仍显式调 `Record`，跳过决策由配置中心化，但 record 调用本身保持显式

## 实施步骤

### Step 1: 配置项注入
- [ ] 在 `configs/config.yaml` 新增 `operlog.exclude_paths: []`
- [ ] 在 `configs/config.dev.yaml` 同步新增（默认空数组）
- [ ] 确认 config struct 在 `internal/config/` 中能正确反序列化（YAML → Go struct）

### Step 2: operlog 包内实现
- [ ] 在 `internal/utils/operlog/operlog.go` 顶部新增包级 `var ExcludedPaths []string`
- [ ] 新增 `func Configure(paths []string)`：替换包级变量（线程安全：单次启动期调用即可，无需 mutex）
- [ ] 新增 `func IsExcludedPath(path string) bool`：使用 `filepath.Match` + `/*` 后缀通配，与 `pkg/middleware/request_decryption.go:294-315` 的 `isExcludedPath` 风格一致
- [ ] 修改 `Record` 函数体首行：`if IsExcludedPath(c.Request.URL.Path) { return }`
- [ ] 修改 `RecordWithBody` 函数体首行：同上（**必须**在 `c.GetRawData()` 之前，否则 body 已被消费）

### Step 3: core.go 引导
- [ ] 在 `internal/core/core.go` 配置加载后、调起任何 handler 前调一次 `operlog.Configure(core.Config.OperLog.ExcludePaths)`
- [ ] 确认配置结构路径正确（`core.Config.OperLog.ExcludePaths` 或项目实际使用的命名约定）
- [ ] 加注释说明启动顺序约束（"必须在任何 Record 调用之前"）

### Step 4: 测试
- [ ] 在 `internal/utils/operlog/regression_test.go` 新增 `TestExcludedPathsEarlyReturn`：
  - mock Recorder 实例，断言路径命中时 `RecordAsync` 不被调用
  - 断言路径未命中时 `RecordAsync` 仍被调用
  - 断言 `RecordWithBody` 在排除路径上不消费 request body（`GetRawData` 不被调用）
- [ ] 在 `internal/utils/operlog/operlog_test.go` 新增 `TestIsExcludedPath`：
  - 字面量匹配（exact match）
  - `/*` 后缀通配
  - 无匹配（return false）
  - `/list` 误伤测试：确认 `isExcludedPath("/system/user/list")` 不被 heartbeat 通配符误匹配

### Step 5: 验证
- [ ] 运行 `go build ./...` 确认编译通过
- [ ] 运行 `go test ./internal/utils/operlog/...` 确认所有测试通过（包括原 14 个 FilterSensitiveParams 子用例）
- [ ] 运行 `go test ./...` 确认无回归
- [ ] 配置 `operlog.exclude_paths: ["/api/v1/rpa/workers/*/heartbeat"]` 后，启动后端 + 一个 rpa-worker Agent，运行 5 分钟
- [ ] 验证 `sys_oper_log` 中无 heartbeat 相关 audit 行（用 SQL：`SELECT COUNT(*) FROM sys_oper_log WHERE oper_url LIKE '%heartbeat%' AND oper_time > NOW() - INTERVAL '5 minutes'`）

### Step 6: 默认配置
- [ ] 在 `configs/config.yaml` 中加入示例：
  ```yaml
  operlog:
    exclude_paths:
      - "/api/v1/rpa/workers/*/heartbeat"
  ```
- [ ] `progress` 端点**不加入**默认排除（保留任务进度审计价值），但加注释说明如需排除可加一行

## 风险与回退

- **回退方案**：若上线后发现误排除，注释配置项重启即可，影响面 0
- **不可回退点**：无（新增功能，不修改现有行为）
- **数据迁移**：无
- **前端影响**：0（纯后端变更）

## 不在本期范围（deferred）

1. rpaApi.ts 路径错配（前端 `/${id}/progress` 与后端 `/workers/progress` 不一致）→ 列入独立 todo
2. `/rpa/workers/autoscale/config` (POST) 无持久化（handler 内 `// TODO`）→ 独立 todo
3. operlog 运行时热改 `Refresh()` API → 待运维需求出现

## 验收标准（Nyquist UAT）

- [ ] heartbeat 端点在 5 分钟连续运行下，sys_oper_log 中 0 条新增审计行
- [ ] register / scale-up / scale-down 等"真业务写"端点仍然记录
- [ ] regression_test.go 全部用例（含 25 OperType 常量、6 参数 Record、5 参数 RecordWithBody、34 敏感关键词）通过
- [ ] `go build ./...` 0 错误
- [ ] `go test ./internal/utils/operlog/...` 全部通过
- [ ] `go test ./...` 0 回归

## 关联文件

- 主笔记：`.planning/notes/260616-rpa-heartbeat-log-flooding.md`
- 目标文件：`internal/utils/operlog/operlog.go`
- 测试文件：`internal/utils/operlog/regression_test.go`、`internal/utils/operlog/operlog_test.go`
- 配置：`configs/config.yaml`、`configs/config.dev.yaml`
- 引导：`internal/core/core.go`
- 风格蓝本：`pkg/middleware/request_decryption.go:292-315`（`isExcludedPath` 函数）
