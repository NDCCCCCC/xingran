---
status: completed
created: 2026-04-16
completed: 2026-04-27
quick_id: "260416-wmv"
---

# Backend Code Optimization Plan

## Status: ✅ COMPLETED

Completed via Phase 9 (5/5 plans) on 2026-04-27

## Original Goal

Clean up dead code, fix security issues, simplify Core struct, and improve code organization.

## Completed Work

See `.planning/milestones/v1.3-phases/09-backend-cleanup/` for full details:
- 09-00: 创建测试存根 (TDD RED)
- 09-01A: 删除已知死代码并验证
- 09-01B: 系统性扫描并删除额外死代码
- 09-02: 清理 Core 结构死字段并处理 RPAScalingService
- 09-03: 实现安全修复并使测试通过 (TDD GREEN)

## Phase 1: Delete Dead Code ✅
- [x] Deleted `internal/services/dashboard_service.go` (zero external refs)
- [x] Deleted `internal/services/settings_service.go` (zero external refs)
- [x] Verified `go build ./...` passes

## Phase 2: Fix Security Issues ✅
- [x] Fixed WebSocket CheckOrigin in `ws_notice_handler.go`
- [x] Fixed GlobalDeviceMonitorService race condition in `cron.go`
- [x] Added error logging for critical silent errors in `auth.go`
- [x] Verified `go build ./...` passes

## Phase 3: Clean Core Dead Fields ✅
- [x] Removed 12 dead fields from Core struct
- [x] Extracted `RPAScalingService` from `interface{}`
- [x] Simplified `Init()` by extracting helper methods
- [x] Verified `go build ./...` passes

## Phase 4: Migrate Flat Service Files ✅
- [x] Moved `internal/services/duty_*.go` → `internal/services/duty/`
- [x] Moved `internal/services/notice_*.go` → `internal/services/notice/`
- [x] Moved `internal/services/device_*.go` → `internal/services/device/`
- [x] Moved `internal/services/knowledge_service.go` → `internal/services/knowledge/`
- [x] Updated all import paths
- [x] Verified `go build ./...` passes
