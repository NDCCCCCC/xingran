---
status: partial
phase: 48-device-component-serials-planned
source:
  - .planning/phases/48-device-component-serials-planned/48-VERIFICATION.md
started: 2026-07-04T10:30:00Z
updated: 2026-07-04T10:30:00Z
milestone: v1.18
verifier_status: human_needed
verifier_score: "14/14 D-id 覆盖 (代码 + 测试证据) + 3 项 site-visit UAT deferred (informational, 非失败)"
automated_gates:
  task_3_grep_gate: PASSED
  go_build: PASSED
  go_test_collector: PASSED (21 tests)
  go_test_operlog_regression: PASSED (5 tests, 25 OperType + 11 keyword 不回归)
  go_test_asset_filter: PASSED (4 tests, Wave 1 List/Statistics 默认过滤)
  go_test_collect_component_info: PASSED (5 D-10 流水线测试)
  go_test_list_components: PASSED (3 handler 测试)
  tsc_no_emit: PASSED
  vite_build: PASSED (1m 40s, vendor-react 775 kB gzip 基线)
  psql_schema_introspection: PASSED (ops_asset 4 新列 + recon_category + 索引切换 + 字典 seed)
---

# Phase 48 Human UAT — Site-Visit Items (Deferred)

**Phase:** 48 (网络设备硬件清单 / Device Component Serials)
**Milestone:** v1.18
**Status:** partial — 等待现场访问

## 自动化闸门（已 PASS，本环境 2026-07-04 跑过）

- `go build ./...` clean
- `go test ./internal/services/component_collector/` — 21 tests
- `go test ./internal/utils/operlog/` — 5 regression tests（25 OperType + 11 sensitive keyword + Record 5参签名不回归）
- `go test ./internal/services/operations/ -run TestAsset` — 4 List/Statistics 过滤测试
- `go test ./internal/services/ -run TestCollectComponentInfo` — 5 D-10 两步流水线测试
- `go test ./internal/api/v1/operations/ -run TestListComponents` — 3 handler 测试
- `cd xingran-react-frontend && tsc --noEmit` clean
- `cd xingran-react-frontend && npm run build` — 1m 40s, vendor-react 775 kB gzip 基线与 Phase 48 前一致
- psql schema introspection（远程 PG 10.62.10.34/xingran）：ops_asset 4 新列 + sys_data_reconciliation.recon_category + 索引 uniq_recon_asset_type_cat_open PRESENT（旧的 uniq_recon_asset_type_open MISSING）+ 字典 asset_reconciliation_recon_category rows=2 seed OK
- **Task 3 automated gate** `grep -q "Phase 48 真机 UAT" .planning/STATE.md` PASSED（已通过 STATE.md §Phase 48 真机 UAT deferred 声明正式记录）

## Tests (site-visit, 3 项 deferred)

### 1. 真机 Huawei S8700 SNMP ENTITY-MIB 单 GET 路径
**expected:** EntityCollector.Collect 在真实 S8700 上拿到 chassis/fan/engine 序列号（D-08: Huawei S8700 拒绝 GetBulk,改单 GET loop）
**result:** [pending] — 推迟到现场访问（`48-RESEARCH.md` §Environment Availability 显式声明本环境无 S8700）
**why_human:** 无 S8700 真机,只有 fixture-driven 单元测试
**addressed_in:** 下次现场访问（运维同事携带 S8700 接入）

### 2. 真机 Ruijie RS8607E SNMP ENTITY-MIB 627 行含 352 `temprature*` 噪声过滤
**expected:** D-11 filter（`Class==8 && strings.HasPrefix(Name, "temprature")`）把 Components 削减到 ≤20 真实组件,real PSU 保留
**result:** [pending] — 推迟到现场访问
**why_human:** 无 RS8607E 真机;synthetic-fixture test 已 assert filter 规则,不验证 352 实际数字
**addressed_in:** 下次现场访问

### 3. D-10 真机两步流水线（Huawei S8700 `10GE5/0/4` actually-up fiber interface）
**expected:** `display interface status` 返回 up;`display interface transceiver` 返回 Vendor SN;down ports 跳过（D-10 enforcement at `cmd_dispatcher` level + `runTwoStepTransceiverPipeline` 编排）
**result:** [pending] — 推迟到现场访问
**why_human:** 无 S8700 fiber port 可实测 CLI 两步流水线
**addressed_in:** 下次现场访问

## Summary

| Metric | Count |
|--------|-------|
| total | 3 |
| passed | 0 |
| issues | 0 |
| pending | 3 |
| skipped | 0 |
| blocked | 0 |

## Gaps

none

## Owner

现场访问时由运维同事携带 S8700 / RS8607E 接入,跑 `collectComponentInfo` cron + 前端 `ComponentListTab` 实测。Site visit 完成后回写本文件 (将 `[pending]` 改为 `pass` / `fail` + 实测详情),并通知 owner 关闭此 UAT。

## 关联声明

- `.planning/STATE.md` §Phase 48 真机 UAT deferred 声明 (2026-07-04)
- `.planning/phases/48-device-component-serials-planned/48-VERIFICATION.md` human_verification section
- `.planning/phases/48-device-component-serials-planned/48-RESEARCH.md` §Environment Availability
- `.planning/phases/48-device-component-serials-planned/48-03-PLAN.md` Task 3 action 5
