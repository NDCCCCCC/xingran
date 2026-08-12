---
slug: agent-server-dupe-decls
status: awaiting_human_verify
trigger: 'Cluster 1/5: 修复 internal/agent/server/ 编译错误 - 重名声明 (WithRequestID, log)'
created: 2026-06-12
updated: 2026-06-12
---

# Cluster 1: internal/agent/server/ Duplicate Declarations

## Symptoms (gathered 2026-06-12)

**Compile error cluster** (用户提供的 system-reminder diagnostics):
```
internal/agent/server/logger.go:70:6: WithRequestID redeclared in this block
    internal/agent/server/handlers.go:14:6: other declaration of WithRequestID
internal/agent/server/handlers.go:11:5: log already declared through import of package log ("log")
    internal/agent/server/config.go:8:2: other declaration of log
internal/agent/server/handlers.go:11:5: log already declared through import of package log ("log")
    internal/agent/server/jwt_auth.go:10:2: other declaration of log
```

**实际代码片段** (handlers.go:1-16):
```go
package server
import (
    "net/http"
    "strings"
    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)
var log = logrus.StandardLogger()
func WithRequestID(requestID string) *logrus.Entry {
    return log.WithField("request_id", requestID)
}
```

## Initial Hypothesis

疑似有人新增了 `logger.go` 作为统一 logger，但未移除 handlers.go / config.go / jwt_auth.go 中的旧 `var log = ...` 和 `func WithRequestID(...)` 声明。属于**半完成的 refactor**。

## Current Focus

- **hypothesis:** `logger.go` 是新版集中 logger（应保留），handlers.go:11 / config.go:8 / jwt_auth.go:10 的 `var log` 与 handlers.go:14 的 `WithRequestID` 是**未清理的旧声明**。
- **next_action:** 应用修复：(1) handlers.go 删除 `var log` + `WithRequestID` 函数；(2) config.go 与 jwt_auth.go 删除 `"log"` import 并把 `log.Printf` 改为 `WithFields(...).Warn(...)` 风格。
- **test:** 修复后 `go build ./internal/agent/server/...` 必须退出码 0；同时 `go vet ./internal/agent/server/...` 无警告；之后 `go build ./...` 验证无连锁影响。
- **expecting:** 4 个编译错误全部消失；agentserver 包对外契约不变（`InitLogger/Info/Warn/Fatal/WithFields/WithRequestID` 都仍由 logger.go 提供）。
- **blind_spots:** ① 其它包 import `server.WithRequestID` / `server.log` ——已 grep 确认无；② cmd/agent/main.go 已确认使用 `server.InitLogger/Info/Fatal/WithFields`，与 logger.go 对齐。
- **reasoning_checkpoint:** 修复策略为**只在 agent/server 包内**移除重复声明（`var log` / 旧 `WithRequestID` / `"log"` import），不影响其它模块的 logger 模式（其它包各自有独立 logger）。`logger.go` 是 single source of truth。
- **tdd_checkpoint:** 暂未启用 TDD；agent/server 无测试文件，无需新增。

## Evidence

<!-- 时序追加新证据 -->
- timestamp: 2026-06-12
  checked: internal/agent/server/logger.go 全文
  found: 包级 `var logger *logrus.Logger`（私有）+ 导出符号 `InitLogger/WithContext/WithRequestID/Debug/Info/Warn/Error/Fatal/WithFields`。
  implication: logger.go 是 single source of truth，handlers.go 重复声明 `WithRequestID` 是多余。
- timestamp: 2026-06-12
  checked: handlers.go 全文件
  found: 11 行 `var log = logrus.StandardLogger()`，13-16 行 `func WithRequestID(...)`（与 logger.go:70 冲突）；无其它 `log.` 用法；保留 `logrus.Fields` 类型引用，所以 `"github.com/sirupsen/logrus"` import 需保留。
  implication: 删除 11-16 行即可解 2 个错误。
- timestamp: 2026-06-12
  checked: config.go 全文
  found: import `"log"`，使用 3 次 `log.Printf`：147 行（agent_id/vm_id 警告）、194 行（证书全局可读警告）、357 行（自动注册失败）。无 `var log` 声明。
  implication: 删除 `"log"` import 并把 3 处 `log.Printf` 改用 logger.go 的 `Warn` 风格。
- timestamp: 2026-06-12
  checked: jwt_auth.go 全文
  found: import `"log"`，仅 1 处 `log.Printf`（121 行证书验证禁用警告）。无 `var log` 声明。
  implication: 删除 `"log"` import 并把 1 处 `log.Printf` 改用 logger.go 的 `Warn` 风格。
- timestamp: 2026-06-12
  checked: 全项目 grep `agent/server.(WithRequestID|log)`
  found: 无任何匹配。
  implication: 删除这两个符号不会破坏包外消费者。
- timestamp: 2026-06-12
  checked: cmd/agent/main.go 使用的 server.* 符号
  found: LoadConfig, Config, RegisterToBackend, InitLogger, NewTLSConfigFromConfig, Fatal, NewJWTAuthenticator, NewAccountManager, NewAgentHandler, NewConnectionManager, ConnectionState, WithFields, Info, RecoveryMiddleware, LoggingMiddleware, CORSMiddleware。所有引用与 logger.go 提供的 API 对齐。
  implication: logger.go 是被期望的 SOT，旧 `var log` / `"log"` import 是该清理的 refactor 残留。
- timestamp: 2026-06-12
  checked: connection_manager.go 全文件
  found: 3 处 `log.WithFields(logrus.Fields{...})`（180/185/195 行），无 `var log` 声明也未 import `"log"`，依赖 handlers.go 的 `var log`。这是隐藏耦合，删除 handlers.go 的 `var log` 会破坏它。
  implication: 必须把 `log.WithFields(...)` 一并改成 `WithFields(...)`（logger.go 导出的包级函数）。
- timestamp: 2026-06-12
  checked: 修复后 `go build ./internal/agent/server/...` + `go build ./...`
  found: 全部 EXIT=0。
  implication: 本 cluster 4 个编译错误全部消失，无连锁影响。
- timestamp: 2026-06-12
  checked: 修复后 `go vet ./internal/agent/server/...`
  found: EXIT=0，无 warning。
  implication: vet 干净。
- timestamp: 2026-06-12
  checked: 全量 `go vet ./...`
  found: 仍报告其它 cluster 的 vet 错误（migration_140 struct tag、scripts/ redundant newline、addomain Infof 格式串、operations workstation_device_service、ad_dept_sync_handler_test 等）。
  implication: 与本 cluster 无关，按 scope constrainment 不处理，记录到 Side findings。

## Side findings (其它 cluster 的问题，**不**在本 session 处理)

- `internal/core/db/migrations/migration_140_vdi_last_sync_time.go:32` — struct tag 语法错误 (`type:timestamp` 多余)
- `scripts/vdi_test_standalone.go` 612/818/946 行 + `scripts/migrate_cache_keys/verify/verify_migration.go:43` — `fmt.Println` redundant newline
- `internal/core/security/authenticator_test.go:116` — `TestADUserInfo` redeclared
- `internal/services/addomain/user_ou_service.go:43` — `logger.Infof` 格式串 `%s` 传入 `gorm.DeletedAt` 类型不匹配
- `internal/services/operations/workstation_device_service.go:640` — `logger.Infof` `%s` 传入 `*string` 类型不匹配
- `internal/api/v1/system/ad_dept_sync_handler_test.go:19` — `NewADDeptSyncHandler` 调用参数不足
- `internal/services/rate_limiter_test.go:17` — `assert.NotNil` 拷贝 `sync.Map` 含 `sync.noCopy`
- `internal/services/system/apikey_service_test.go:129` — 类型不匹配

## Eliminated

- hypothesis: `logger.go` 是被错误新增，应删除以保留 handlers.go 的旧声明
  evidence: cmd/agent/main.go 与 logger.go 提供的 API 完全对齐（`InitLogger/Info/Warn/Fatal/WithFields`），而 handlers.go 的旧 `var log = logrus.StandardLogger()` 与 logger.go 的 `var logger *logrus.Logger` 走的是不同 logger 实例，结构化 JSON 输出能力丢失。
  timestamp: 2026-06-12

## Eliminated

<!-- 已被排除的假设 -->

## Resolution

- root_cause: `internal/agent/server/logger.go` 是该包的 SOT（导出 `InitLogger/WithRequestID/Info/Warn/WithFields` 等），但之前的 refactor 没把旧文件清理干净：
  1. `handlers.go` 仍声明 `var log = logrus.StandardLogger()` 和同名 `func WithRequestID(...)`，与 logger.go 冲突；
  2. `connection_manager.go` 隐式依赖 handlers.go 的 `var log`（3 处 `log.WithFields(...)`）；
  3. `config.go` 与 `jwt_auth.go` 用标准库 `log.Printf` —— 在有了 `var log` 时与 `"log"` import 名称冲突；删除 `var log` 后它们也必须改用 logger.go 提供的 API。
  本质是"半完成的 logger 集中化 refactor"。

- fix: 保留 logger.go；把 handlers.go 的 `var log` + 旧 `WithRequestID` 删除；把 connection_manager.go 的 3 处 `log.WithFields(...)` 改成包级 `WithFields(...)`；把 config.go 的 3 处 `log.Printf` 与 jwt_auth.go 的 1 处 `log.Printf` 全部改用 logger.go 的 `WithFields(...).Warn(...)` 或 `Warn(...)`；删除两个文件中的 `"log"` import；config.go 新增 `"github.com/sirupsen/logrus"` import（因使用 `logrus.Fields` 类型字面量）。

- verification:
  - `go build ./internal/agent/server/...` → EXIT=0
  - `go vet ./internal/agent/server/...` → EXIT=0
  - `go build ./cmd/...` → EXIT=0
  - `go build ./...` → EXIT=0
  - 全量 `go vet ./...` 仍报其它 cluster 的错（见 Side findings），与本 cluster 无关。

- files_changed:
  - internal/agent/server/handlers.go
  - internal/agent/server/config.go
  - internal/agent/server/jwt_auth.go
  - internal/agent/server/connection_manager.go
  - .planning/debug/agent-server-dupe-decls.md
