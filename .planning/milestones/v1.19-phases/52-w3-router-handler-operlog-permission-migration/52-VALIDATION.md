---
phase: 52
slug: w3-router-handler-operlog-permission-migration
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-07
---

# Phase 52 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: `52-RESEARCH.md` §4 (Validation Architecture / Nyquist Dimension 8).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go 内置 `testing` + `testify` (assert/mock) + Phase 51 已用的 sqlite in-memory |
| **Config file** | 无独立 config（Go 标准 `*_test.go` 同包） |
| **Quick run command** | `go test ./internal/api/v1/network/... -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **operlog 回归锁** | `go test ./internal/utils/operlog/... -count=1 -v` |
| **Phase 51 service 回归** | `go test ./internal/services/portwrite/... -count=1 -v` |
| **Estimated runtime** | ~15-25 秒（全量 `go test ./...`） |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...` + `go test ./internal/utils/operlog/... -count=1`（保 Phase 34 回归锁 intact）
- **After every plan wave:** Run `go test ./... -count=1`（全量，含 Phase 51 portwrite 回归 + 新 handler/router/migration 测试）
- **Before `/gsd:verify-work`:** Full suite must be green + `go vet ./...` exit 0
- **Max feedback latency:** ~25 秒

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 52-01-t1 | 01 | 1 | PERM-01 | T-52-01 | `NetworkPortWrite` 常量定义且 handler 引用非硬编码 | unit (编译时) | `go build ./pkg/permission/...` | ❌ W0 | ⬜ pending |
| 52-01-t2 | 01 | 1 | (路径 X) | — | `PortWriteAudit` model 定义供 Wave 1 handler 引用 | unit (编译时) | `go build ./internal/models/...` | ❌ W0 | ⬜ pending |
| 52-01-t3 | 01 | 1 | INFRA-03 | — | cache_keys.go 仅定义 2 常量（不写缓存） | unit (编译时) | `go build ./internal/services/portcollection/...` | ❌ W0 | ⬜ pending |
| 52-01-t4 | 01 | 1 | PORT-01..05 / BATCH-01 / AUDIT-01/02/03 / CONV-01..04 | T-52-02 | handler 调对应 service 方法 + operlog.Record 在 response.Success 前 + audit 行写入 + sentinel→HTTP 翻译 | handler unit test (mock service + sqlite in-memory) | `go test ./internal/api/v1/network/... -run TestPortWrite -v` | ❌ W0 | ⬜ pending |
| 52-01-t5 | 01 | 1 | PERM-02 / INFRA-02 | T-52-03 | 组级 `RequirePermissions([network:port:write], core)` + 6 kebab 端点可解析 | source-grep + gin.Engine 路由树断言 | `go test ./internal/api/v1/network/... -run TestSetupPortWriteRouter -v` | ❌ W0 | ⬜ pending |
| 52-01-t6 | 01 | 1 | INFRA-02 | — | `SetupPortWriteRouter(ports, core)` 在 network_router.go 注册 | unit (gin 路由树) | `go test ./internal/api/v1/network/... -count=1` | ❌ W0 | ⬜ pending |
| 52-02-t1 | 02 | 2 | INFRA-01 | — | `sys_port_write_audit` 表 + `(device_id,port_id,created_at)` 复合索引 + `(created_at)` 单列索引 | schema introspection | `db.Migrator().HasTable("sys_port_write_audit")` + 列检查 | ❌ W0 | ⬜ pending |
| 52-02-t2 | 02 | 2 | INFRA-01 | — | PortWriteAudit 加入 database.go AutoMigrate 列表 + 显式调 `Migrate202PortWriteAudit` | migration unit test (sqlite in-memory) | `go test ./internal/core/db/migrations/... -run TestMigrate202 -v` | ❌ W0 | ⬜ pending |
| 52-02-t3 | 02 | 2 | PERM-03 | — | `GrantNewMenuToRolesHavingParent` 幂等 helper | helper unit test (幂等性 + 只波及父关联角色) | `go test ./internal/core/db/migrations/... -run TestGrantNewMenu -v` | ❌ W0 | ⬜ pending |
| 52-02-t4 | 02 | 2 | PERM-03 | T-52-04 | `sys_menu` seed 行 (menu_type='F', perms='network:port:write') + `sys_role_menu` grant 行 | migration unit test | 跑 migration_202 后 `SELECT COUNT(*) FROM sys_menu WHERE menu_name='端口配置'` ≥ 1 + `sys_role_menu` 关联计数 | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

测试文件桩（planner 必须在每个对应任务里创建）：

- [ ] `internal/api/v1/network/port_write_router_test.go` — 覆盖 PERM-02 / INFRA-02（组级鉴权 + 6 端点路由可解析）
- [ ] `internal/api/v1/network/port_write_handler_test.go` — 覆盖 PORT-01..05 / BATCH-01 / AUDIT-01/02/03（handler 调 service + operlog 时序 + audit 行写入三态）
- [ ] `internal/core/db/migrations/migration_202_port_write_audit_test.go` — 覆盖 INFRA-01 / PERM-03（表/索引 + 菜单 seed + 角色授权）
- [ ] `internal/core/db/migrations/menu_grant_helpers_test.go` — 覆盖 D-08 helper 幂等性

*Framework 已就位（testify/mock + sqlite in-memory，Phase 51 已用），无 install gap。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| PG 生产库 `sys_port_write_audit` 表 + 索引 + 菜单 seed + 角色授权真实落库 | INFRA-01 / PERM-03 | 需要 PG 连接（CI 用 sqlite in-memory） | 启动 backend → `\d sys_port_write_audit` 验表 + 索引；`SELECT * FROM sys_menu WHERE menu_name='端口配置'`；`SELECT * FROM sys_role_menu WHERE menu_id=<新菜单>`。Phase 54 UAT 覆盖 |
| 真机 SSH 写命令端到端（6 端点 × 3 厂商） | PORT-01..05 / BATCH-01 | 需现场设备 | 见 `50-HUMAN-UAT.md`，v1.19 推迟到下次现场访问 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 25s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
