---
phase: 11
slug: mac-address-filter
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-05-09
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing package (stdlib) + Testify assertions |
| **Config file** | none — 使用 go test 命令行参数 |
| **Quick run command** | `go test -v ./internal/services/mac_collection/ -run TestMACFiltering` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -v <modified_package> -run Test<SpecificFunction>`
- **After every plan wave:** Run `go test ./internal/services/lldp/ ./internal/services/mac_collection/ ./internal/services/topology/`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 11-01-01 | 01 | 1 | MAC-01 | — | LLDP discovery 使用安全的连接池访问 | unit | `go test -v ./internal/services/lldp/ -run TestDiscoverNeighbors` | ✅ Plan 04 | ⬜ pending |
| 11-02-01 | 02 | 2 | MAC-02 | T-11-01 | LLDP neighbor ports 被正确过滤 | unit | `go test -v ./internal/services/mac_collection/ -run TestFilterLLDPPorts` | ✅ Plan 04 | ⬜ pending |
| 11-02-02 | 02 | 2 | MAC-03 | — | MAC count threshold 被正确应用 | unit | `go test -v ./internal/services/mac_collection/ -run TestMACThreshold` | ✅ Plan 04 | ⬜ pending |
| 11-03-01 | 03 | 2 | MAC-04 | — | LLDP 失败时 MAC 采集继续 | integration | `go test -v ./internal/services/mac_collection/ -run TestLLDPFallback` | ✅ Plan 04 | ⬜ pending |
| 11-04-01 | 04 | 3 | MAC-05 | T-11-02 | 配置的过滤规则被正确应用 | unit | `go test -v ./internal/services/topology/ -run TestFilterRules` | ✅ Plan 04 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/services/lldp/lldp_service_test.go` — LLDP discovery tests stubs (MAC-01)
- [x] `internal/services/mac_collection/mac_filter_test.go` — MAC filtering tests stubs (MAC-02, MAC-03, MAC-04)
- [x] `internal/services/topology/filter_rule_test.go` — Filtering rule tests stubs (MAC-05)
- [x] TextFSM templates for LLDP parsing (Huawei/H3C/Ruijie/Maipu) — Created in Plan 01

**Note:** 复用 `internal/services/portcollection/` 和 `internal/services/mac_collection_service.go` 中的现有测试模式。

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 真实网络设备的 LLDP 发现 | MAC-01 | 需要真实网络设备（或模拟器）验证 LLDP 命令 | 1. 连接测试网络中的锐捷/华为设备<br>2. 运行 LLDP 发现<br>3. 验证返回的邻居信息正确 |
| MAC 过滤效果验证 | MAC-02, MAC-03 | 需要验证过滤后存储的数据符合预期 | 1. 在测试设备上运行 MAC 采集<br>2. 检查数据库中的 MAC 地址数量<br>3. 确认 uplink 端口的 MAC 被过滤 |
| 性能测试（100+ 设备） | All | 需要大规模测试验证性能 | 1. 模拟 100+ 设备环境<br>2. 运行 MAC 采集<br>3. 测量完成时间和资源使用 |

---

## Security Domain References

| Threat Ref | Pattern | Mitigation |
|------------|---------|-----------|
| T-11-01 | Command injection on LLDP queries | 使用参数化命令（Scrapligo 防止注入） |
| T-11-02 | Unauthorized filter rule modification | 现有 RBAC 系统保护过滤规则配置 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

---

*Phase: 11-mac-address-filter*
*Validation created: 2026-05-09*
