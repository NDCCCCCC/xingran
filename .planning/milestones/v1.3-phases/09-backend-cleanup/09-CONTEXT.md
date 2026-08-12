# Phase 9: 后端代码优化 - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

清理后端死代码、修复已知安全问题、简化 Core 结构。核心目标：删除未使用的代码、修复安全漏洞、使 Core 结构更清晰。不影响外部 API 行为，不包含服务文件迁移（迁移延后到未来阶段）。

</domain>

<decisions>
## Implementation Decisions

### 阶段范围
- **D-01**: 本阶段不包括 Phase 4 的服务文件迁移
  - 只执行 Phase 1-3：死代码删除、安全修复、Core 清理
  - 服务文件迁移 (duty_, notice_, device_, knowledge) 延后到未来阶段
  - 原因：ROADMAP 只定义了 3 个计划，迁移属于更大范围的重构

### 死代码发现策略
- **D-02**: 系统性扫描 services 目录寻找其他死代码
  - 不限于已识别的 2 个文件 (dashboard_service.go, settings_service.go)
  - 扫描 `internal/services/` 目录，检查零外部引用的文件
  - 使用 `grep -r` 或 Go 工具分析引用关系
  - 发现的额外死代码文件记录到规划中

### Core 结构清理方式
- **D-03**: 渐进式删除 Core 死字段
  - 分多个小步骤，每步删除 2-3 个字段
  - 每步后运行 `go build ./...` 和 grep 验证
  - 每个步骤独立提交，便于回滚
  - 12 个死字段分类：DeviceManager, NetworkDeviceService, DeviceMonitorService, UserService-SettingsService 组 (7 字段), APIMetadata

### 清理验证方法
- **D-04**: 双重验证确保删除安全
  - 第一关：运行 `grep -r "FieldName" --include="*.go" | grep -v "^internal/core/core.go"` 确认无外部引用
  - 第二关：运行 `go build ./...` 验证构建通过
  - 两关都通过才删除，任一失败则调查原因

### 安全测试要求
- **D-05**: 编写单元测试验证安全修复
  - WebSocket CheckOrigin 验证：测试 Origin header 验证逻辑
  - GlobalDeviceMonitorService 竞态：测试并发访问场景
  - 错误日志：测试错误日志记录功能
  - 测试文件命名：`ws_notice_handler_test.go`, `cron_test.go`, `auth_test.go`

### Claude's Discretion
- 死代码扫描的具体工具选择 (go list, grep, 或其他静态分析工具)
- Core 字段删除的具体分组方式 (按功能模块或依赖关系)
- 单元测试的覆盖率目标
- 错误日志的格式和级别

</decisions>

<specifics>
## Specific Ideas

- 死代码扫描优先检查 `*_service.go` 文件的外部引用
- Core 字段删除顺序建议：先删除明显未使用的 (DeviceManager, NetworkDeviceService)，再删除有依赖的 (DeviceMonitorService)，最后删除缓存预热相关的 (UserService-SettingsService 组)
- WebSocket CheckOrigin 应验证 Origin 是否来自允许的域名列表，或使用更安全的验证机制
- GlobalDeviceMonitorService 竞态条件修复参考 Phase 8 的 RWMutex 模式

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 项目规范
- `.planning/codebase/CONVENTIONS.md` — Go 代码风格、错误处理规范
- `.planning/codebase/ARCHITECTURE.md` — 分层架构、Handler-Service 模式
- `.planning/codebase/CONCERNS.md` — 技术债、已知问题

### 需求文档
- `.planning/REQUIREMENTS.md` — CODE-02a, CODE-02b, CODE-02c 需求定义
- `.planning/ROADMAP.md` — Phase 9 成功标准
- `.planning/quick/20260416-backend-optimize/PLAN.md` — 详细优化计划 (Phase 1-3)

### 现有代码
- `internal/core/core.go` — Core 结构定义，需要清理的目标
- `internal/api/v1/ws_notice_handler.go` — WebSocket CheckOrigin 修复位置
- `internal/scheduler/cron.go` — GlobalDeviceMonitorService 竞态条件修复位置
- `internal/api/v1/auth.go` — 错误日志添加位置

### 参考模式
- `.planning/phases/08-snmp-panic/08-CONTEXT.md` — 并发安全模式 (RWMutex) 参考

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **并发安全模式**: Phase 8 实现的 RWMutex 模式可用于 GlobalDeviceMonitorService 竞态修复
- **测试框架**: 项目已配置 `go test`，可直接使用 testing 包编写单元测试
- **错误日志模式**: 使用 `applogger.Errorf()` 统一日志记录

### Established Patterns
- **Handler-Service 模式**: 所有服务遵循此模式，Core 字段删除需考虑此依赖关系
- **错误包装**: 使用 `fmt.Errorf("context: %w", err)` 保持错误链
- **测试文件命名**: `xxx_test.go` 与源文件同目录

### Integration Points
- Core 结构被 `internal/api/router.go` 和所有 handler 使用
- WebSocket handler 在 `internal/api/v1/ws_notice_handler.go`
- 定时任务调度器在 `internal/scheduler/cron.go`

### Known Issues
- `internal/services/dashboard_service.go` — 死代码，零外部引用
- `internal/services/settings_service.go` — 死代码，零外部引用
- `internal/api/v1/ws_notice_handler.go:50` — CheckOrigin 缺失验证
- `internal/scheduler/cron.go` — GlobalDeviceMonitorService 竞态条件
- `internal/api/v1/auth.go:120` — 登录日志创建缺少错误日志

</code_context>

<deferred>
## Deferred Ideas

- 服务文件迁移到子目录 (Phase 4 of 原优化计划)
- Core 结构的更大规模重构
- 其他目录的死代码扫描 (非 services 目录)

</deferred>

---

*Phase: 09-backend-cleanup*
*Context gathered: 2026-04-27*
