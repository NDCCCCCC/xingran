---
phase: quick-261015-ops-fk-drift-guard
plan: 01
status: complete
subsystem: operations
tags: [gorm, postgres, type-contract, regression-guard, ops]
requires:
  - ops-list-join-uuid-500 (debug session, resolved 2026-10-15)
provides:
  - Workstation.BuildingID model/DB 类型对齐(gorm type:uuid)
  - PG-gated 类型契约集成测试 TestPGContract_OpsListJoins
affects:
  - internal/models/workstation.go
  - internal/services/operations/pg_contract_test.go
tech-stack:
  added: []
  patterns:
    - PG-gated integration test (TEST_PG_DSN + t.Skip gating)
    - model 对齐 DB 的 type:uuid 契约声明(照抄 OpsFloor.PlanImageID)
key-files:
  created:
    - internal/services/operations/pg_contract_test.go
  modified:
    - internal/models/workstation.go
decisions:
  - 漂移方向 = model 对齐 DB(不动 schema/数据/零 migration),保住 9c0e3e5c 的 uuid=uuid JOIN
  - 回归守护 = 真实 PG 集成测试(TEST_PG_DSN),否决 DryRun(SQLite 动态类型测不出 text=uuid)
  - GORM v1.30.5 无 gorm.OpenLazy → 用 &gorm.Config{DisableAutomaticPing: true} 等价实现
metrics:
  duration: ~5 min
  completed: 2026-10-15
---

# Quick Task 261015-ops-fk-drift-guard Summary

Workstation.BuildingID gorm tag 从 size:64 对齐为 type:uuid(与 DB 列 uuid 一致),并新增 TEST_PG_DSN 门控的三服务 List+Statistics PG 类型契约测试,守护 ops List JOIN 的 uuid/varchar 比较方向。

## 完成任务

| Task | 内容 | Commit | 文件 |
| ---- | ---- | ------ | ---- |
| 1 | Workstation.BuildingID tag 对齐 DB uuid 列 | 38b9987b | internal/models/workstation.go |
| 2 | PG-gated 类型契约集成测试 | 753e0b89 | internal/services/operations/pg_contract_test.go |

## 验证结果

- `go build ./...` 零错误(两次任务后各跑一次)
- `go test ./internal/services/operations/` 全绿(6.2s);`go test ./internal/models/...` 全绿
- `go vet ./internal/services/operations/` 通过
- 无 TEST_PG_DSN 环境:`go test -v -run TestPGContract` 输出 `--- SKIP`(非 FAIL),包整体 `ok`
- `git log --oneline -2`:恰好两个新 commit;`git show --stat` 各自只含本任务文件;零 `.planning/` 文件入库;零文件删除
- 审计复核:`grep -rn "sys_workstation\.building_id" --include="*.go" internal/` 全部命中与规划期定性表一致(infopoint_service.go:159 uuid=uuid / excel_query_builder.go:234 `::uuid` no-op / floor_service.go:581 参数化写入 / migration 192/193 uuid-aware),**无 varchar 方向比较残留,无修复需要**

## Deviations from Plan

### 装配偏差(约束第 9 条,以实际代码为准)

**1. `gorm.OpenLazy` 不存在,改用 `DisableAutomaticPing`**
- **发现于:** Task 2
- **问题:** 计划中的 `gorm.Open(postgres.Open(dsn), gorm.OpenLazy)` 在 GORM v1.30.5 编译失败(`undefined: gorm.OpenLazy`——GORM v2 无该功能选项,已核验模块源码 options/gorm.go)
- **修复:** 改为 `gorm.Open(postgres.Open(dsn), &gorm.Config{DisableAutomaticPing: true})`,等价实现 lazy 意图:Open 阶段不探活,由显式 `PingContext(5s)` 统一探活并控制超时。已在代码注释与 commit message 中注明
- **文件:** internal/services/operations/pg_contract_test.go
- **Commit:** 753e0b89

其余装配(New*Service 三构造器签名、List/Statistics 共 6 项调用签名、requests 类型)与计划 interfaces 逐项一致,无出入。

## 端到端验证(需 PG,由编排者/用户执行)

```bash
TEST_PG_DSN="postgres://user:pass@10.62.10.34/ruoyi?sslmode=disable" \
  go test -v -run TestPGContract ./internal/services/operations/
```

预期:三段 t.Run(floor/workstation/infopoint)全 PASS。

## Known Stubs

无。

## Self-Check: PASSED

- internal/models/workstation.go — FOUND(38b9987b,`type:uuid` 命中 BuildingID 行,size:64 无残留)
- internal/services/operations/pg_contract_test.go — FOUND(753e0b89,121 行,含 6 项调用 + 双 skip 分支 + 只读保证注释)
- commit 38b9987b — FOUND;commit 753e0b89 — FOUND
