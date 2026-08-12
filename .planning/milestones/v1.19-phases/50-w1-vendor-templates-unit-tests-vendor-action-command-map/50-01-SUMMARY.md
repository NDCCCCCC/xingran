---
phase: 50-w1-vendor-templates-unit-tests-vendor-action-command-map
plan: 01
subsystem: network-device-write
tags: [go, vendor-template, port-action, scrapli, dot1x, huawei-vrp, h3c-comware, ruijie-rgos, testify]

# Dependency graph
requires: []
provides:
  - "RenderCommand public API: vendor x action x params -> []string, error"
  - "PortAction named type + 5 const + String() method (audit-friendly snake_case values)"
  - "PortTemplateParams struct with InterfaceName + Description"
  - "5 sentinel errors: ErrUnsupportedVendor/ErrUnknownAction/ErrEmptyInterfaceName/ErrDescriptionEmpty/ErrDescriptionTooLong"
  - "15 hardcoded (vendor, action) command templates (3 vendors x 5 actions)"
  - "Table-driven test coverage locking vendor syntax drift (STATE.md Pitfall #3)"
affects:
  - "phase-51-portwrite-service"
  - "phase-52-portwrite-handler"
  - "phase-54-mock-ssh-e2e"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Internal package-level dispatch map (vendorPortTemplate) keyed by typed constants"
    - "Sentinel errors with portcollection: prefix for context-line consistency"
    - "Sprintf-based command rendering (no regex, no escaping — D-06 deferred)"
    - "Cisco-style two-command shape for Ruijie dot1x (interface view + dot1x port-control)"
    - "VRP heritage shape for Huawei/H3C (single-line dot1x enable/undo dot1x enable)"

key-files:
  created:
    - "internal/services/portcollection/vendor_port_template.go"
    - "internal/services/portcollection/vendor_port_template_test.go"
  modified: []

key-decisions:
  - "Same-package coexistence with template_cache.go (TextFSM) confirmed — different subpackage imports (internal/templates vs internal/models), no symbol collision"
  - "Huawei/H3C share VRP heritage commands per D-08; Ruijie diverges per D-07 with Cisco-style interface view"
  - "Description max 80 chars enforced to match UI subset of device_port_status.Description size:500"

patterns-established:
  - "Phase 50 template contract: RenderCommand(vendor models.DeviceVendor, action PortAction, params PortTemplateParams) ([]string, error)"
  - "Phase 51 service consumer pattern: for i, cmd := range cmds { SendConfig(cmd) }; index i = failure locator"

requirements-completed: [SSH-05, SSH-01]

# Metrics
duration: ~10min
completed: 2026-07-06
---

# Phase 50 Plan 01: Vendor Templates + Unit Tests Summary

**锁定 15 个 (vendor, action) 命令模板 + 表驱动单测，覆盖 Huawei VRP / H3C Comware / Ruijie RGOS × shutdown / undo shutdown / description / dot1x enable / dot1x disable，为 Phase 51 PortWriteService 提供稳定底座**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-07-06T10:40Z (approx)
- **Completed:** 2026-07-06T10:50Z
- **Tasks:** 3
- **Files modified:** 2

## Accomplishments
- 实现 `RenderCommand` 公共入口，签名 `(vendor models.DeviceVendor, action PortAction, params PortTemplateParams) ([]string, error)` 锁定
- 落地 3 厂商 × 5 操作 = 15 个硬编码命令模板（Huawei/H3C VRP 同源、Ruijie Cisco 风格）
- 5 个哨兵错误（`ErrUnsupportedVendor` / `ErrUnknownAction` / `ErrEmptyInterfaceName` / `ErrDescriptionEmpty` / `ErrDescriptionTooLong`）覆盖所有失败路径
- 15 矩阵 + 5 负向 = 20 个测试用例全部通过；`go build ./...` / `go vet ./...` / `go test ./internal/services/portcollection/...` 全绿
- 与现有 `template_cache.go`（TextFSM 缓存）同包共存，互不干扰（导入路径不同、符号无碰撞）

## Task Commits

每个 task 原子提交：

1. **Task 1: Create vendor_port_template.go — types, map, RenderCommand, renderers** - `3f59a91e` (feat)
2. **Task 2: Create vendor_port_template_test.go — table-driven matrix + negative cases** - `c86b3523` (test)
3. **Task 3: Build + run all portcollection tests — Phase 50 acceptance gate** - 无文件变更（验收门）

## Files Created/Modified
- `internal/services/portcollection/vendor_port_template.go` (137 行) — PortAction 类型 + 5 const + PortTemplateParams struct + 5 sentinel errors + vendorPortTemplate map + RenderCommand + 4 个 renderer
- `internal/services/portcollection/vendor_port_template_test.go` (173 行) — TestRenderCommand_VendorActionMatrix (15 子测试) + 5 个独立负向测试

## Decisions Made
- **D-08 同源化**：Huawei VRP 与 H3C Comware 在 map 中展开为相同字面闭包（VRP 血统），便于后续 grep 验证；同时保留 H3C map key 让 Phase 51 调用方按厂商显式分发
- **D-07 Ruijie dot1x 双命令**：锐捷 `dot1x port-control auto` / `no dot1x port-control` 必须先 `interface <name>` 进 view，对应 `[2]string` 长度
- **Description 80 字符硬截**：与 `device_port_status.Description size:500` UI 子集约定对齐；`ErrDescriptionTooLong` 错误信息携带实际长度便于审计定位
- **不引入 Description 转义**：scrapli 文本模式透传，注入风险由设备 CLI 解析器兜底（D-06 deferred to future phase）
- **PortAction 暴露 String()**：便于 Phase 52 operlog 直接采 `action: dot1x_enable` 类审计可读形式

## Deviations from Plan

None - 计划严格按 D-01..D-09 与 specifics 测试骨架执行，未触发任何 deviation rule。

## Issues Encountered

None - 编译/测试/构建均一次通过。

## User Setup Required

None - 零外部依赖，无 service / DB / SSH 资源需要配置。

## Next Phase Readiness

- **Phase 51 PortWriteService** 可直接调用 `portcollection.RenderCommand(vendor, action, params)` 取得 `[]string` 命令序列，通过 scrapli.SendConfigs 顺序下发
- **失败定位**：`SendConfigs` 返回 `[]*Response`，索引 `i` 即失败命令定位锚点（D-05 决策生效）
- **operlog 兼容**：PortAction 值（`shutdown` / `undo_shutdown` / `description` / `dot1x_enable` / `dot1x_disable`）可直接作为 Phase 52 operlog action 字段值
- **Maipu 占位**：`models.VendorMaipu` 未注册到 `vendorPortTemplate`，调用返回 `ErrUnsupportedVendor`（D-01 scope 锁定，后续 phase 扩展）

---
*Phase: 50-w1-vendor-templates-unit-tests-vendor-action-command-map*
*Completed: 2026-07-06*