---
phase: 13
slug: query-layer-trajectory
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-13
---

# Phase 13 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (go test) + GORM + vitest (前端) |
| **Config file** | `go.mod` + `package.json` |
| **Quick run command** | `go test -v -run TestMAC ./internal/services/...` |
| **Full suite command** | `go test ./... && (cd xingran-react-frontend && npm run test -- --run)` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -v -run TestMAC ./internal/services/...`
- **After every plan wave:** Run `go test ./internal/services/... && cd xingran-react-frontend && npm run test -- --run src/components/network/MACTrajectoryChart.test.tsx`
- **Before `/gsd:verify-work`:** Full suite must be green (`go test ./... && npm run test`)
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 13-01-01 | 01 | 1 | QUERY-02 | T-13-01 | LAG 窗口函数仅参数化查询, MAC 正则验证 | integration | `go test -v -run TestQueryMACTrajectory` | ❌ W0 | ⬜ pending |
| 13-01-02 | 01 | 1 | QUERY-02 | — | 跨分区连续状态区间聚合 | integration | `go test -v -run TestTrajectoryCrossPartition` | ❌ W0 | ⬜ pending |
| 13-02-01 | 02 | 1 | QUERY-03 | — | 明细 + Top-N 两段式输出 | unit | `go test -v -run TestQueryConnectionStats` | ❌ W0 | ⬜ pending |
| 13-02-02 | 02 | 1 | QUERY-03 | — | 长期占用阈值过滤 (long_occupancy_threshold_days) | unit | `go test -v -run TestLongOccupancyFilter` | ❌ W0 | ⬜ pending |
| 13-03-01 | 03 | 2 | QUERY-04 | T-13-02 | sys_mac_oui_vendor 表 DDL + GORM 迁移 | integration | `go test -v -run TestMigrationOUIVendor` | ❌ W0 | ⬜ pending |
| 13-03-02 | 03 | 2 | QUERY-04 | T-13-02 | OUI 启动初始化 (空表导入 JSON) — INSERT ON CONFLICT DO NOTHING | integration | `go test -v -run TestInitOUITable` | ❌ W0 | ⬜ pending |
| 13-03-03 | 03 | 2 | QUERY-04 | — | OUI 查询 + Unknown 降级 + Redis L1 缓存降级 SQL | unit | `go test -v -run TestMACVendorService` | ❌ W0 | ⬜ pending |
| 13-04-01 | 04 | 2 | UI-03 | T-13-03 | Gantt 组件渲染节点 + 颜色编码 (custom series renderItem) | unit (vitest) | `npm run test -- --run src/components/network/MACTrajectoryChart.test.tsx` | ❌ W0 | ⬜ pending |
| 13-04-02 | 04 | 2 | UI-03 | T-13-03 | 空状态 + 骨架屏 + Ant Design Empty | unit (vitest) | `npm run test -- --run src/pages/network/mac/trajectory.test.tsx` | ❌ W0 | ⬜ pending |
| 13-04-03 | 04 | 2 | UI-03 | — | MAC 多格式输入 (AA:BB:CC / aabb.ccdd.eeff / aabb-ccdd-eeff) | unit (vitest) | `npm run test -- --run src/utils/normalizeMACAddress.test.ts` | ❌ W0 | ⬜ pending |
| 13-05-01 | 05 | 3 | Phase 12 UAT | — | mac_history_cleanup 清理任务在调度器注册 | smoke | `grep "mac_history_cleanup" internal/scheduler/*.go` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/mac_history_query_service_test.go` — LAG 窗口函数 + 区间聚合单元测试 (QUERY-02)
- [ ] `internal/services/mac_vendor_service_test.go` — OUI 启动初始化 + 查询测试 (QUERY-04)
- [ ] `internal/services/mac_oui_vendor_test.go` — 模型 + DDL 测试 (QUERY-04)
- [ ] `internal/core/db/migrations/migration_mac_oui_vendor.go` — sys_mac_oui_vendor 表 DDL
- [ ] `configs/oui-vendors.json` — 500 条精选厂商数据(版本化, git 跟踪)
- [ ] `xingran-react-frontend/src/components/network/MACTrajectoryChart.test.tsx` — ECharts Gantt 渲染测试 (UI-03)
- [ ] `xingran-react-frontend/src/pages/network/mac/trajectory.test.tsx` — 轨迹页面集成测试 (UI-03)
- [ ] `xingran-react-frontend/src/lib/macHistoryApi.ts` — API 客户端封装 (UI-03)

*Framework install:* 无需安装 — Go 标准 testing + 现有 vitest 4.0.18 已覆盖。

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| ECharts Gantt 实际渲染效果 | UI-03 | 视觉效果需人眼确认 | 启动 dev server, 访问 `/network/mac/trajectory`, 输入 MAC 地址, 确认颜色编码正确 |
| 跨月分区轨迹合并 | QUERY-02 | 真实数据跨度需手动注入 | 注入跨分区测试数据, 验证 LAG 跨分区无丢失 |
| Phase 12 清理任务 UAT 重测 | Phase 12 UAT | 需手动确认 scheduler 注册 | 启动服务, 查询 sys_scheduler_task, 确认 mac_history_cleanup 已注册 |

*如果所有行为均有自动化验证: 删除此节*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending