# Phase 8: SNMP Panic 修复 - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

修复 scrapligo 传输层 panic 问题，确保 SNMP 操作在高并发场景下稳定。核心目标：SNMP Ping 操作不再导致应用崩溃，支持 20+ 并发连接。

</domain>

<decisions>
## Implementation Decisions

### SNMP 客户端保护
- **D-01**: 为 `snmp_client.go` 中的所有 SNMP 操作方法添加 panic 恢复包装器
  - 方法: `Get()`, `GetNext()`, `Walk()`, `GetBulk()`
  - 使用 defer recover 捕获 gosnmp 库可能的 panic
  - 保持与 `scrapli_wrapper.go` 一致的错误处理模式
  - panic 发生时返回错误而非崩溃

### 日志记录增强
- **D-02**: panic 日志必须包含完整诊断信息
  - 堆栈跟踪: 使用 `debug.Stack()` 捕获完整调用栈
  - 设备标识: IP 地址、设备名称、设备 ID（如果可用）
  - 操作上下文: 方法名、命令/参数
  - 时间戳: panic 发生的时间
  - 日志级别: ERROR

### 测试验证策略
- **D-03**: 使用单元测试验证 panic 修复的有效性
  - 测试场景: 模拟各种 panic 条件（nil pointer、网络异常等）
  - 验证内容: panic 被正确捕获、错误被正确返回、连接状态被正确设置
  - 不依赖实际设备: 使用 mock 或模拟场景

### Claude's Discretion
- 单元测试的具体实现方式（使用 testing 包还是第三方测试框架）
- 堆栈跟踪的格式化方式
- 日志输出目的地（使用项目现有的 applogger 还是标准库 log）

</decisions>

<specifics>
## Specific Ideas

- panic 恢复模式参考 `scrapli_wrapper.go:296-304` 的实现
- 日志格式应包含足够的上下文信息以便快速定位问题
- 单元测试应该覆盖常用的 SNMP 操作场景

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 项目规范
- `.planning/codebase/CONVENTIONS.md` — Go 代码风格、错误处理、响应格式规范
- `.planning/codebase/ARCHITECTURE.md` — 分层架构、Handler-Service 模式
- `.planning/codebase/CONCERNS.md` — 技术债、已知问题

### 需求文档
- `.planning/REQUIREMENTS.md` — SNMP-01a, SNMP-01b, SNMP-01c 需求定义
- `.planning/ROADMAP.md` — Phase 8 成功标准

### 现有代码
- `internal/device/scrapli_wrapper.go` — 参考 panic 恢复模式
- `internal/device/snmp_client.go` — 需要添加 panic 保护的目标文件
- `internal/device/connection_pool.go` — 并发安全机制参考

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Panic 恢复模式**: `scrapli_wrapper.go` 中已实现的 defer recover 模式可直接复用
- **日志工具**: `applogger` 包已在项目中使用，用于统一日志记录
- **并发安全**: `connection_pool.go` 中的 RWMutex 模式已验证有效

### Established Patterns
- **Handler-Service 模式**: 网络设备相关代码遵循此模式
- **错误包装**: 使用 `fmt.Errorf("context: %w", err)` 保持错误链
- **状态管理**: `ConnectionState` 枚举定义了连接生命周期

### Integration Points
- SNMP 客户端被 `device_monitor_service.go` 和 `device_discovery_service.go` 使用
- 连接池在 `network_device_service.go` 中被引用
- 日志系统使用 `github.com/xingran-next/xingran-go-backend/pkg/logger`

</code_context>

<deferred>
## Deferred Ideas

- 升级 scrapligo 版本（破坏性变更风险，超出本期范围）
- 实时配置流式传输（过度工程）
- 并发压力测试（可选择添加，非必需）

</deferred>

---

*Phase: 08-snmp-panic*
*Context gathered: 2026-04-27*
