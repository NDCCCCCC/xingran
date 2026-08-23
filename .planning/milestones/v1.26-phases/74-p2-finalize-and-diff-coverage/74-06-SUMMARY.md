---
phase: 74
plan: 06
subsystem: vdi-services-coverage
tags: [coverage, service-tests, sqlite, httptest, p2-finalize]
dependency_graph:
  requires: [phase-73-service-patterns, phase-73-02-vdi-handler-tests]
  provides: [vdi-services-test-suite]
  affects: [internal/services/vdi]
tech_stack:
  added: []
  patterns:
    - httptest 假 VDI API（深信服 error_code/token 格式）驱动 vdiClientExtendedImpl + VDIAuthManager 真实 HTTP 路径
    - fakeVDIClientExtended 全方法假实现注入 VMService（按方法注入结果/错误）
    - ListVMs localCount==0 自动同步入口 + SyncVMsFromVDIByServer 强制同步入口覆盖全链路 sync
    - AUTH_TOKEN_INVALID(1101) 一次注入验证 callAPIWithRetry 清缓存重认证
    - VDIAPIMock（包内自带 mock server）驱动旧版 VDIClient /api/v1/* 端点
key-files:
  created:
    - internal/services/vdi/vdi_server_service_test.go
    - internal/services/vdi/vm_service_test.go
    - internal/services/vdi/vdi_client_extended_test.go
    - internal/services/vdi/audit_service_test.go
  modified: []
decisions:
  - id: D-12-STRICT
    summary: 仅 4 个 *_test.go 入库，go.mod/go.sum 零改动
  - id: D-15-P2-FLOOR
    summary: services/vdi 2.7% → 85.1%（≥70%）
  - id: PLAN-DEVIATION
    summary: 计划设想的 base/group/desktop 子服务文件不存在；实际按包内真实文件拆为 server/vm/client_extended/audit+legacy_client 四个测试文件
metrics:
  completed_date: 2026-08-22
  baseline_coverage: 2.7
  final_coverage: 85.1
  coverage_delta: 82.4
  test_files_added: 4
---

# Phase 74 Plan 06: VDI Services Tests Summary

**One-liner:** `internal/services/vdi`（1127 stmts，Phase 73 交接时 2.7%）经 4 个测试文件（1677 行）推至 **85.1%**，无函数低于 60%。

## Files Created（4 files）

| File | Covers |
|------|--------|
| `vdi_server_service_test.go` | 共享 sqlite 建表 helper（8 张表）+ fakeVDIExtServer 假 VDI API + VDIServerService CRUD/排序白名单/密码加密/TestConnection + ClientManager（含单例）+ AES 密码往返 |
| `vm_service_test.go` | fakeVMVDIClient 注入 + ListResourceGroups/ListResources/CreateVM/GetVM/UpdateVM/ListVMs（4 种过滤+排序+数据范围）/DeleteVM（3001 特例）/OperateVM（4 种电源态本地更新）/BindUnbindUser/SyncVMFromVDI/SyncAllVMs + mapper 纯函数 + ApplyVMDataScopeFilter 全 6 分支 |
| `vdi_client_extended_test.go` | VDIAuthManager（缓存/过期/解密失败/401/DB 写回/IsTokenExpired/RefreshToken/ClearTokenCache/callAPI）+ extended client 全端点（含 AUTH_TOKEN_INVALID 重试）+ GetVTPPlatforms 去重 + ListVMs 触发全量同步（资源组/VM/孤儿清理/assign_ip 优先） |
| `audit_service_test.go` | AuditService 记录+查询（过滤/时间范围/排序白名单/分页）+ 操作摘要 + 旧版 VDIClient × VDIAPIMock 全端点（含未认证/token 过期/各操作失败路径） |

## T-NORESP quirk（ensureVDIServer response.Error int→400）

该 quirk 属 **api 层** `internal/api/v1/vdi/base_handler.go:49`（ensureVDIServer），非 services 包代码。Phase 73-02 已在 `internal/api/v1/vdi/base_handler_test.go` 以行为断言锁定（handler 测试断言 400 状态码与错误响应体）。本 plan 按 D-12 不修，services 层测试不重复该断言（无对应服务方法），此处引用存档。

## Documented Quirks（D-12 — 不修业务码）

1. **AuditService.RecordOperation 在 sqlite 恒报错**：`rpa.AuditLog.OldValue/NewValue` 是 `map[string]interface{}`，glebarez sqlite 驱动无法绑定 map 参数（PG 下由 pgx 原生编码 jsonb）。测试断言其报错并以 raw SQL 播种审计行。
2. **SyncVMFromVDI 只更新内存不落库**：方法末尾给 `vm` 结构体赋值后直接 return，无 `db.Save`——调用方拿不到更新，本地 DB 不变。测试仅断言无错。
3. **CreateVM 无视注入的 VDIClient**：内部 `NewVDIClientFromDB` 自建真实 client，VMService 的 `vdiClient` 字段对 CreateVM 无效（测试以 httptest 假端点驱动）。
4. **ListVMs 自动同步只在本地 VM 表全空时触发**（localCount==0 全表计数，非按 server 计数）：任一 server 有残留行则永不自动同步；孤儿清理需走 SyncVMsFromVDIByServer。
5. **OperateVM 对未知 action 不更新本地电源态但也不报错**（powerState map miss 静默跳过）；fake client 下无校验，真实 VDI 端点会拒绝。
6. **sqlite 字符串 DATETIME 时区偏移**：raw SQL 以本地时间字符串写 `token_expiry` 后读回按 UTC 解析（+8h 偏移）；测试用 GORM `Update` 传 time.Time 保持与 cacheToken 一致的写格式。
7. **VDIClient（旧版）Timeout 单位是毫秒**（`timeoutMs`），ClientManager 创建时传 `30 * 1000`；`NewVDIClient(url, 5000)` 实为 5 秒。
8. **vdi 包内 vdi_utils.go 与 models/vdi.go 各有一份 encrypt/decryptVDIPassword 复制**：行为差异——models 版解密失败返回原串，vdi 版返回空串（auth manager 依赖空串判断"解密失败"）。

## Constraints Honored

- D-12 STRICT: 仅 `*_test.go`（staged diff 验证，go.mod/go.sum 零改动）
- D-15: 85.1% ≥ 70%
- T-NORESP quirk 引用存档（api 层已由 73-02 锁定，未修）
- No STATE.md/ROADMAP.md updates（orchestrator-owned）；no push
