---
status: approved
created: 2026-04-16
quick_id: "260416-wmv"
---

# Backend Code Optimization Plan

## Goal
Clean up dead code, fix security issues, simplify Core struct, and improve code organization.

## Phase 1: Delete Dead Code (2 files)
- [ ] Delete `internal/services/dashboard_service.go` (zero external refs, router uses system/ version)
- [ ] Delete `internal/services/settings_service.go` (zero external refs, Core uses system/ version)
- [ ] Verify `go build ./...` passes

## Phase 2: Fix Security Issues
- [ ] Fix WebSocket CheckOrigin in `internal/api/v1/ws_notice_handler.go` - validate Origin header
- [ ] Fix GlobalDeviceMonitorService race condition in `internal/scheduler/cron.go` - add mutex protection
- [ ] Add error logging for critical silent errors in `internal/api/v1/auth.go` (login log creation)
- [ ] Verify `go build ./...` passes

## Phase 3: Clean Core Dead Fields
- [ ] Remove 12 dead fields from Core struct:
  - `DeviceManager` (only created/closed internally)
  - `NetworkDeviceService` (network router creates its own)
  - `DeviceMonitorService` (accessed via scheduler global)
  - `UserService` through `SettingsService` (7 fields, only for cache warm-up) → convert to local vars
  - `APIMetadata` (only used internally)
- [ ] Extract `RPAScalingService` from `interface{}` to proper interface
- [ ] Simplify `Init()` by extracting helper methods
- [ ] Verify `go build ./...` passes

## Phase 4: Migrate Flat Service Files to Subdirectories
- [ ] Move `internal/services/duty_*.go` (6 files) → `internal/services/duty/`
- [ ] Move `internal/services/notice_*.go` (5 files) → `internal/services/notice/`
- [ ] Move `internal/services/device_*.go` (4 files) → `internal/services/device/`
- [ ] Move `internal/services/knowledge_service.go` → `internal/services/knowledge/`
- [ ] Update all import paths across codebase
- [ ] Verify `go build ./...` passes

## Scope Constraints
- Do NOT change any external API behavior
- Each phase produces a separate atomic commit
- Build must pass after each phase
