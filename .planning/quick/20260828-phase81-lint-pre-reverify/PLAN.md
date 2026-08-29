---
plan: quick-phase81-lint-pre-reverify
status: complete
archived: 2026-08-29
fix_commits:
  - 074ecad
  - 173169b
  - 71d9256
  - 10af153
---

# Phase 81 lint pre-reverify: 清 13 SA* warning

## 问题

CI run `33176387515` backend job 因 13 个 Phase 79/80 测试文件静态分析告警而 FAIL，导致 Coverage gate SKIPPED。本地无可测 CI Coverage gate green。Phase 81 81-02 完成的 push 不能算 SC-3 收口。

## 13 个 lint 告警所在文件 (Phase 79/80 测试文件)

主要为:
- `interface{}` 应替换为 `any`(最多)
- 部分 `for i := 0; i < N; i++` 循环可现代化
- `Ineffective break statement`
- `use of `nil` (SA1012)`

均集中在 `*_78_0X_test.go`、`*_79_0X_test.go`、`*_80_0X_test.go` 文件。生产代码不动。

## 修复

对每个文件用 `gofmt -r` / `sed` 批改 `interface{}` → `any`,然后逐文件 `go vet` 验证。零生产改动。

## 验证

1. `go vet ./...` exit 0
2. `go test -run <pattern> <pkg>` 不回归
3. `git push` 触发 CI backend job 重跑,期望 Coverage gate 解锁(不再 SKIPPED)且 green
4. `bash .github/scripts/check-coverage.sh` 本地仍然 exit 0(仅静态检查,不受影响)
