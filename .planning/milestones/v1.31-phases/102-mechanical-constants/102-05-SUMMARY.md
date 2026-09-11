---
phase: "102"
plan: "05"
subsystem: pagination
tags: [pagination, PAGI-01, migration]
dependency_graph:
  requires: []
  provides: [PAGI-01]
  affects: [internal/api/v1/system/file_handler.go]
tech_stack:
  added: []
  patterns:
    - query.NormalizePaginationWithMax 唯一实现核心
    - constants.MaxListPageSize = 100（cap 严格锁定）
    - 继承锁定 Phase 99 D-03-3：零调用方即删除，不留兼容壳
key_files:
  created:
    - .planning/phases/102-mechanical-constants/102-PAGI-CALLER-AUDIT.md
  modified:
    - internal/api/v1/system/file_handler.go
  deleted:
    - internal/utils/pagination.go
    - internal/utils/utils_74_12_test.go（TestParsePaginationAndOffset 函数）
decisions:
  - "唯一生产调用方 file_handler.go:160 迁移 query.NormalizePaginationWithMax，cap=100 注释自证历史分叉（D-102-9）"
  - "utils import 保留（BuildListResponse/GetUserID/GetUsername/GetClientIP 仍在用）"
  - "BuildListResponse 保留不动（CONTEXT Deferred 分叉）"
  - "utils/pagination.go 整文件删除（继承锁定 Phase 99 D-03-3）"
  - "TestParsePaginationAndOffset 随文件删除，剩余测试函数保留"
metrics:
  duration: "~5 minutes"
  completed: "2026-09-07"
---

# Phase 102 Plan 05: PAGI-01 分页口径归一 Summary

**One-liner:** 分页归一收敛到 pkg/query.NormalizePaginationWithMax 单一口径，file_handler 迁移后 utils/pagination.go 整文件删除，零残留引用。

## Tasks Completed

| # | Task | Commit | Status |
|---|------|--------|--------|
| 1 | file_handler.go 迁移 NormalizePaginationWithMax + 调用方核对清单落盘 | `046feff` | DONE |
| 2 | 删除 internal/utils/pagination.go 与对应测试 | `418110e` | DONE |

## Task 1: file_handler.go 迁移

**变更内容：**
- `List()` handler（:153-174）从 `utils.ParsePagination` 改为 `query.NormalizePaginationWithMax(req.Page, req.PageSize, constants.MaxListPageSize)`
- 局部算式 `offset := (current - 1) * pageSize` 替代 `pagination.Offset()`
- `BuildListResponse` 实参从 `pagination.Page, pagination.PageSize` 改为 `current, pageSize`
- 新增 import: `pkg/query` + `pkg/constants`
- `internal/utils` import 保留（BuildListResponse / GetUserID 等仍在用）
- cap=100 历史分叉注释自证

**Commit:** `046feff`

## Task 2: 删除 utils/pagination.go

**变更内容：**
- `internal/utils/pagination.go` 整文件删除（ParsePagination / PaginationParams / Offset / Limit / BuildPaginationResponse）
- `internal/utils/utils_74_12_test.go` 删除 `TestParsePaginationAndOffset` 函数（:203-230）
- 编译期守护：全仓 grep 零 utils 版 ParsePagination / BuildPaginationResponse 残留

**Commit:** `418110e`

## Deviations from Plan

None — plan executed exactly as written.

## Verification Results

```
go build ./...                                    ✓ BUILD OK
go test ./internal/api/v1/system/ -run TestFile  ✓ OK (0.199s)
go test ./internal/utils/                        ✓ OK (0.214s)
go test ./pkg/query/                             ✓ OK (0.209s)
go test ./pkg/constants/                         ✓ OK (0.525s)
grep ParsePagination (non-test, non-pkg/query)   ✓ 0 matches
grep BuildPaginationResponse (non-test)          ✓ 0 matches
```

## Artifacts

- **102-PAGI-CALLER-AUDIT.md** — 逐调用方核对清单（SC-3），含三行核对表 + grep 证据命令 + 执行日期

## Self-Check

- [x] file_handler.go contains `query.NormalizePaginationWithMax`
- [x] file_handler.go contains `constants.MaxListPageSize`
- [x] file_handler.go contains cap=100 注释
- [x] file_handler.go 不再引用 `utils.ParsePagination`
- [x] 102-PAGI-CALLER-AUDIT.md 存在且含核对表
- [x] internal/utils/pagination.go 不存在
- [x] utils_74_12_test.go 不含 TestParsePaginationAndOffset
- [x] go build ./... 退出码 0
- [x] go test ./internal/api/v1/system/ -run TestFile 0 失败

## Self-Check: PASSED
