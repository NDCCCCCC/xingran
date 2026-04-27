---
phase: 9
slug: backend-cleanup
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-27
updated: 2026-04-27
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go 1.24 内置) |
| **Config file** | none — 使用标准 testing 包 |
| **Quick run command** | `go test ./internal/core/... ./internal/api/v1/... ./internal/scheduler/...` |
| **Full suite command** | `go test ./... && go build ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...` (快速验证)
- **After every plan wave:** Run `go test ./... && go build ./...` (完整验证)
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 09-00-01 | 00 | 0 | CODE-02c | T-9-01 | WebSocket CheckOrigin 测试存根存在 | unit | `go test ./internal/api/v1/... -run TestWebSocketUpgrader_CheckOrigin` | ✅ W0 | ⬜ pending |
| 09-00-02 | 00 | 0 | CODE-02c | T-9-02 | GlobalDeviceMonitorService 并发测试存根存在 | unit | `go test ./internal/scheduler/... -run TestGlobalDeviceMonitorService_ConcurrentAccess` | ✅ W0 | ⬜ pending |
| 09-00-03 | 00 | 0 | CODE-02c | T-9-03 | 登录错误日志测试存根存在 | unit | `go test ./internal/api/v1/... -run TestRecordLoginLog_ErrorLogging` | ✅ W0 | ⬜ pending |
| 09-01A-01 | 01A | 1 | CODE-02a | — | 删除已知死代码后构建通过 | integration | `go build ./...` | ✅ W0 | ⬜ pending |
| 09-01A-02 | 01A | 1 | CODE-02a | — | grep 确认零外部引用 | static | `grep -r "dashboard_service\|settings_service" --include="*.go" \| grep -v "^internal/services/"` | ✅ W0 | ⬜ pending |
| 09-01B-01 | 01B | 2 | CODE-02a | — | 额外死代码删除后构建通过 | integration | `go build ./...` | ✅ W0 | ⬜ pending |
| 09-01B-02 | 01B | 2 | CODE-02a | — | services 目录扫描完成 | static | `find internal/services/ -maxdepth 1 -name "*_service.go" -type f -exec basename {} \;` | ✅ W0 | ⬜ pending |
| 09-02-01 | 02 | 1 | CODE-02b | — | Core 死字段删除后构建通过 | integration | `go build ./...` | ✅ W0 | ⬜ pending |
| 09-02-02 | 02 | 1 | CODE-02b | — | grep 确认字段无外部引用 | static | `grep -r "FieldName" --include="*.go" \| grep -v "^internal/core/core.go"` | ✅ W0 | ⬜ pending |
| 09-03-01 | 03 | 2 | CODE-02c | T-9-01 | WebSocket CheckOrigin 拒绝非法请求 | unit | `go test ./internal/api/v1/... -run TestWebSocketUpgrader_CheckOrigin` | ✅ W0 | ⬜ pending |
| 09-03-02 | 03 | 2 | CODE-02c | T-9-02 | GlobalDeviceMonitorService 并发安全 | unit | `go test ./internal/scheduler/... -run TestGlobalDeviceMonitorService_ConcurrentAccess` | ✅ W0 | ⬜ pending |
| 09-03-03 | 03 | 2 | CODE-02c | T-9-03 | 错误日志正确记录 | unit | `go test ./internal/api/v1/... -run TestRecordLoginLog_ErrorLogging` | ✅ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/api/v1/ws_notice_handler_test.go` — WebSocket CheckOrigin 测试存根 (T-9-01) - **Plan 00 创建**
- [x] `internal/scheduler/cron_test.go` — GlobalDeviceMonitorService 并发测试存根 (T-9-02) - **Plan 00 创建**
- [x] `internal/api/v1/auth_test.go` — 错误日志测试存根 (T-9-03) - **Plan 00 创建**

*Note: Wave 0 (Plan 00) 使用 TDD 方法创建测试存根，所有测试初始为 Skip 状态，等待 Plan 03 实现后转为 Green。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|---|
| WebSocket 跨域行为 | CODE-02c | 需要浏览器环境验证 Origin 验证 | 1. 启动后端服务 2. 从不同域名访问 WebSocket endpoint 3. 验证请求被拒绝 |
| 并发压力测试 | CODE-02c | 竞态条件在高并发下才可能出现 | 1. 运行 20+ 并发 SNMP ping 2. 验证无 panic 发生 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

---

*Phase: 09-backend-cleanup*
*Validation strategy created: 2026-04-27*
*Updated: 2026-04-27 - Added Plan 00 (Wave 0) test stubs, marked nyquist_compliant: true*
