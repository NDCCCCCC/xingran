---
phase: 56
slug: vlan-v1-20-1-0-5-plans-initiated-2026-07-09
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-09
---

# Phase 56 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | `testing` + `testify/assert` + `testify/mock` |
| **Framework (Frontend)** | `vitest` (existing) — Phase 56 不新增 frontend unit tests (type-check 即可) |
| **Config file** | None new — reuses v1.19 layout |
| **Quick run command (Go)** | `go test ./internal/services/portcollection/ ./internal/services/portwrite/ -count=1` |
| **Full suite command (Go)** | `go test ./...` |
| **Quick run command (Frontend)** | `npm run type-check` |
| **Estimated runtime** | ~60 seconds (go test 全包) |

---

## Sampling Rate

- **After every task commit:** `go test ./internal/services/portcollection/ ./internal/services/portwrite/ -count=1` (变更包)
- **After every plan wave:** `go build ./... && go test ./... && npm run type-check && npm run build`
- **Before `/gsd:verify-work`:** Full suite green + operlog regression + vendor-react bundle ≤ 776 kB baseline
- **Max feedback latency:** 60s

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 56-01-01 | 01 | 1 | VLAN-02, BIND-03 | T-56-01 (H3C keyword divergence) | Huawei/H3C/Ruijie 模板分别返回正确 vendor 命令 | unit | `go test ./internal/services/portcollection/ -run TestRenderCommand_VendorActionMatrix -count=1` | ✅ extends | ⬜ pending |
| 56-01-02 | 01 | 1 | VLAN-02, BIND-03, TEST-01 | T-56-01 | 12+ 新 vendor template test cases (4 actions × 3 vendors, 加上 legacy/edge variants) | unit | same as 56-01-01 | ✅ extends | ⬜ pending |
| 56-02-01 | 02 | 2 | VLAN-01, VLAN-03, VLAN-05, VLAN-06 | T-56-02 (VLAN range injection) | Service 拒绝 VLAN ID < 1 或 > 4094,sentinel error → 400 | unit | `go test ./internal/services/portwrite/ -run TestSetAccessVlan_Validation -count=1` | ❌ W2 creates | ⬜ pending |
| 56-02-02 | 02 | 2 | BIND-01, BIND-05, BIND-07 | T-56-03 (IP/MAC injection) | Service 拒绝非法 IPv4/MAC,IP/MAC regex 严格 | unit | `go test ./internal/services/portwrite/ -run TestPortBinding_Validation -count=1` | ❌ W2 creates | ⬜ pending |
| 56-02-03 | 02 | 2 | INFRA-01, VLAN-03, BIND-05 | — | pre_state_check 3 vendors 各自 handler;PVID match / binding exists → NoOp | unit | `go test ./internal/services/portwrite/ -run TestCheckPreState_SetAccessVlan\|TestCheckPreState_PortBinding -count=1` | ❌ W2 creates | ⬜ pending |
| 56-02-04 | 02 | 2 | INFRA-01, VLAN-06, BIND-07 | — | audit row JSONB 写入 (before/after sys_port_write_audit) | unit | `go test ./internal/services/portwrite/ -run TestAuditRow_JsonbWrites -count=1` | ❌ W2 creates | ⬜ pending |
| 56-03-01 | 03 | 3 | VLAN-01, BIND-01, INFRA-02 | T-56-04 (unauth access) | POST /network/ports/write/set-access-vlan 需 `network:port:write` perm | unit | `go test ./internal/api/v1/network/ -run TestSetupPortWriteRouter_RequirePermissions2Arg -count=1` | ✅ v1.19 exists | ⬜ pending |
| 56-03-02 | 03 | 3 | VLAN-01, BIND-02, INFRA-04 | — | operlog OperType 映射 (set_access_vlan=Update, port_binding add=Create, remove=Delete) | regression | `go test ./internal/utils/operlog/ -run TestOperTypeConstantStability -count=1` | ✅ exists | ⬜ pending |
| 56-04-01 | 04 | 4 | UI-01, UI-02, UI-03, UI-04, VLAN-04, BIND-06 | — | SetAccessVlanModal + PortBindingModal 通过 antd InputNumber/Input 校验,reason 5-200 字符 | type-check | `npm run type-check` | ❌ W4 creates | ⬜ pending |
| 56-04-02 | 04 | 4 | UI-05, UI-06 | — | ports/index.tsx 下拉菜单接入 7 actions,BulkWriteDrawer 零改动 | build | `npm run build` (vendor-react gzip ≤ 776 kB) | ❌ W4 creates | ⬜ pending |
| 56-05-01 | 05 | 5 | TEST-03, TEST-04 | — | 6+ fixture 文件 + 10+ TestE2E_* 用 scrapligo FileTransport 通过 | e2e | `go test ./internal/services/portwrite/ -run TestE2E_SetAccessVlan\|TestE2E_PortBinding -count=1` | ❌ W5 creates | ⬜ pending |
| 56-05-02 | 05 | 5 | TEST-05 | — | 56-HUMAN-UAT.md 创建,6+ site-visit SSH verification 项目 (Huawei S8700 + Ruijie RS8607E + H3C) | doc | `test -f .planning/phases/56-*/56-HUMAN-UAT.md && grep -c "^- \[ \]" .planning/phases/56-*/56-HUMAN-UAT.md \| awk '$1 >= 6'` | ❌ W5 creates | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/portcollection/vendor_port_template.go` — 3 actions × 3 vendors 新增 (W1)
- [ ] `internal/services/portcollection/vendor_port_template_test.go` — extends with 12+ test cases (W1)
- [ ] `internal/services/portwrite/pre_state_check.go` — 2 action handler 新增 (W2)
- [ ] `internal/services/portwrite/port_write_service.go` — SetAccessVlan + PortBinding methods 新增 (W2)
- [ ] `internal/services/portwrite/port_write_service_test.go` — extends with 5+ unit tests (W2)
- [ ] `internal/api/v1/network/port_write_handler.go` — 2 handler 新增 (W3)
- [ ] `internal/api/v1/network/port_write_router.go` — 2 kebab 端点新增 (W3)
- [ ] `internal/services/portwrite/port_write_e2e_test.go` — extends with 10+ e2e tests (W5)
- [ ] `internal/services/portwrite/testdata/*.fixture` — 6+ fixtures (W5)
- [ ] `xingran-react-frontend/src/types/network.ts` — 2 type aliases 新增 (W4)
- [ ] `xingran-react-frontend/src/lib/api/networkApi.ts` — 2 wrapper 新增 (W4)
- [ ] `xingran-react-frontend/src/components/network/port-write/SetAccessVlanModal.tsx` — CREATE (W4)
- [ ] `xingran-react-frontend/src/components/network/port-write/PortBindingModal.tsx` — CREATE (W4)
- [ ] `xingran-react-frontend/src/pages/network/ports/index.tsx` — 2 menu items 接入 (W4)
- [ ] `docs/plans/2026-07-09-v1.20.1-design.md` — §3.2 修正 (H3C `user-bind` no `static`, Ruijie full MAC+VLAN+IP 语法) (W1 之前由 researcher 触发)
- [ ] `.planning/phases/56-*/56-HUMAN-UAT.md` — site-visit UAT 推迟清单 (W5)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Huawei S5700/S5735/S8700 `port link-type access` + `port default vlan X` 实际生效 | VLAN-02 | 需真机 SSH 访问 | 现场 UAT: SSH 设备,执行 `port link-type access` + `port default vlan 100`,然后 `display port vlan GE1/0/0` 验证 PVID=100 |
| Huawei S8700 `user-bind static ip X mac Y interface Z` 实际生效 | BIND-03 | 需真机 SSH 访问 | 现场 UAT: SSH 设备,执行绑定命令,然后 `display user-bind static all` 验证 (IP, MAC, Interface) 出现在列表 |
| H3C Comware V7 `user-bind` (no `static`) 关键字正确性 | BIND-03 | 需 H3C 设备现场 + 关键字分歧风险 | 现场 UAT: SSH H3C 设备,执行 `user-bind ip-address 10.x.x.x mac-address xxxx-xxxx-xxxx`,然后 `display user-bind` 验证 |
| Ruijie RS8607E `switchport port-security binding <mac> <vlan?> <ip>` 完整语法 | BIND-03 | 需 Ruijie 设备现场,用户实采仅 IP 列 | 现场 UAT: SSH 设备,执行 `switchport port-security binding AABB.CCDD.EEFF 10.62.2.111`,然后 `show port-security binding` 验证 MAC 出现 |
| 跨固件版本命令差异 (Huawei V200R005 vs V600R024C00) | VLAN-02 | 需多个固件版本设备 | 现场 UAT: 至少 2 个不同固件版本对比 |
| 批量性能 (50 端口顺序下发) | INFRA-01 | 需批量数据 | 现场 UAT: 50 端口绑定,测量总耗时 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
