# Phase 32 Wave 6 SUMMARY

**Phase**: 32 — v1.14 P1 重构与 P2 架构优化
**Wave**: 6 of 7 — Migrations & Subprocess (P2-A4/A7/A8)
**Status**: ✅ COMPLETE
**Date**: 2026-06-13
**Commits**: 3 commits

---

## Wave Objectives

完成 P2 架构债的迁移与子进程管理任务（HIGH refactor risk）：
- **P2-A4**: 迁移文件编号冲突修复（10 个冲突文件）
- **P2-A7**: 子进程 Setpgid + reaper（account_manager.go 僵尸进程防护）
- **P2-A8**: Excel 导入事务包裹（ImportData 原子性）

---

## Task Completion Summary

### P2-A4 ✅ Migration File Conflict Documentation (COMPLETE)

**Objective**: 消除 10 个迁移文件的编号冲突歧义（027/029/030/031 重号）

**Investigation**: 
通过检查 migration runner 发现：SQL 文件**不是通过文件名排序自动加载**，而是通过 Go 代码中的迁移函数手动调用（如 `migrations.Migrate143VDIRemoveVMStatus(d.DB)`）。文件编号仅用于组织命名，不影响执行顺序。

**Conclusion**: 编号冲突**无害**（适用安全路径）。

**Solution**: 为 11 个文件添加 source-tracking 头部注释（保留文件名不变）
- 文档化每个文件的原始 commit hash 和创建日期
- 说明冲突原因及为何无需重命名
- 不修改任何 SQL DDL 内容

**Files Modified**: 11 migration files (027/028/029/030/031/036 prefixes)

**Commit**: `6230412` - docs(migrations): P2-A4 add source-tracking headers to conflicting files

**Verification**:
- ✅ `grep -l "P2-A4" internal/core/db/migrations/*.sql | wc -l` = 11
- ✅ `go build ./...` passed (SQL files not compiled)
- ✅ git history preserved (no renames)

---

### P2-A7 ✅ Subprocess Setpgid + Reaper (COMPLETE)

**Objective**: 为 account_manager.go 的 15+ exec.Command 调用添加进程组隔离，防止僵尸进程 FD 泄漏

**Approach**: 跨平台进程组隔离 + reaper goroutine

**Changes Made**:

**Step 1**: 创建跨平台进程组隔离辅助函数
- `internal/agent/server/subprocess.go`: `runCommand`, `runCommandOutput`, `newCommand`, `setProcessGroup` helpers
- `internal/agent/server/sysproc_linux.go`: Linux 用 `Setpgid: true`（//go:build linux || darwin）
- `internal/agent/server/sysproc_windows.go`: Windows 用 `CREATE_NEW_PROCESS_GROUP`（//go:build windows）

**Step 2**: 重构 account_manager.go
- 所有 15 个 `exec.Command` 调用改用 helper 函数
- Windows 策略：6 个 powershell 调用
- Linux 策略：9 个 useradd/userdel/usermod/chpasswd/getent/tee/chmod 调用
- 移除直接 `os/exec` 引用

**Step 3**: 创建 subprocess reaper（跨平台）
- `internal/core/subprocess_reaper_linux.go`: Linux 用 `syscall.Wait4` 循环（30s 间隔）清理僵尸进程
- `internal/core/subprocess_reaper_windows.go`: Windows no-op（OS 自动清理）
- 集成到 Core 生命周期：`Init()` 启动，`Close()` 停止（`reaperCtx`/`reaperCancel`）

**Step 4**: 测试验证
- `subprocess_pgroup_test.go`: 5 个测试用例
  - TestSubprocess_SetProcessGroup
  - TestRunCommand_ExecutesSuccessfully
  - TestRunCommandOutput_ExecutesSuccessfully
  - TestRunCommand_ContextCancellation
  - TestSetProcessGroup_Idempotent

**Files Created**:
- `internal/agent/server/subprocess.go` (47 lines)
- `internal/agent/server/subprocess_pgroup_test.go` (74 lines)
- `internal/agent/server/sysproc_linux.go` (12 lines)
- `internal/agent/server/sysproc_windows.go` (12 lines)
- `internal/core/subprocess_reaper_linux.go` (33 lines)
- `internal/core/subprocess_reaper_windows.go` (12 lines)

**Files Modified**:
- `internal/agent/server/account_manager.go` (重构为使用 helper)
- `internal/core/core.go` (Init/Close 集成 reaper)
- `internal/core/core_infra.go` (添加 reaperCtx/reaperCancel 字段)

**Commit**: `27662de` - fix(agent): P2-A7 subprocess Setpgid + reaper goroutine

**Verification**:
- ✅ `go build ./...` passed
- ✅ `go vet ./internal/agent/server/ ./internal/core/` clean
- ✅ 5 tests passing
- ✅ Every exec.Command now has process group isolation

---

### P2-A8 ✅ Excel Import Transaction Wrapper (COMPLETE)

**Objective**: 将 Excel ImportData 方法包装在单一 GORM Transaction 中，确保部分失败时回滚

**Problem**: 
原代码中 `processThreeLevelDepartments`（创建部门）和 `Upsert`（保存记录）是两个独立的 DB 操作序列，任一失败会导致部分数据写入（孤儿部门或孤儿记录）。

**Solution**: 三步重构

**Step 1**: ImportData 事务包裹
- 将 `processThreeLevelDepartments` + `Upsert` 包装在 `s.db.WithContext(ctx).Transaction()` 中
- 部分失败时整个导入回滚

**Step 2**: 线程化事务句柄
- 为 4 个函数添加 `db *gorm.DB` 参数：
  - `processThreeLevelDepartments(ctx, db, ...)`
  - `findDeptIDByCode(ctx, db, ...)`
  - `ensureDeptGroupExists(ctx, db, ...)`
  - `ensureDeptExists(ctx, db, ...)`
- 替换这些函数内所有 `s.db` 为 `db`
- BatchUpsert 用 `tx` 构造（`NewBatchUpsert(tx, config)`）

**Step 3**: 缓存失效移到事务后
- 缓存写入不是事务性的，必须在事务提交后运行
- 避免回滚时缓存已更新但数据库未变更的不一致状态

**Key Insight**: 
`processThreeLevelDepartments` 及其 3 个 helper 仅在 excel_service.go 内部调用，重构范围可控且安全。

**Files Modified**:
- `internal/services/operations/excel_service.go` (ImportData 事务包裹 + 4 函数签名)

**Files Created**:
- `internal/services/operations/excel_transaction_test.go` (5 contract/signature tests)

**Commit**: `608f3f9` - fix(excel): P2-A8 wrap ImportData in single transaction

**Verification**:
- ✅ `go build ./...` passed
- ✅ `go vet ./internal/services/operations/` clean
- ✅ 5 transaction tests passing (signature + contract guards)
- ✅ ImportData signature unchanged
- ✅ BatchUpsert existing tests still passing

---

## Build & Test Results

### Build Verification
```bash
$ go build ./...
BUILD SUCCESS ✅
```

### Test Coverage
| Test File | Tests | Status |
|-----------|-------|--------|
| subprocess_pgroup_test.go | 5 | ✅ All PASS |
| excel_transaction_test.go | 5 | ✅ All PASS |
| batch_upserter_test.go | (existing) | ✅ Still PASS |

### Pre-existing Failures (NOT caused by Wave 6)
- `validation_helper_test.go`: TestValidator_ValidateFloor/Wall/Door — 验证逻辑错误码 1500，与事务包裹无关

---

## Threat Model Mitigation

| Threat ID | Category | Mitigation | Status |
|-----------|----------|------------|--------|
| T-32-19 | Tampering (order ambiguity) | P2-A4: source-tracking comments | ✅ Resolved |
| T-32-20 | DoS (zombie FD) | P2-A7: Setpgid + reaper goroutine | ✅ Resolved |
| T-32-21 | Tampering (partial state) | P2-A8: Transaction wrapper | ✅ Resolved |

---

## Cross-Platform Design Decisions

### 1. Process Group Isolation
项目同时构建 Windows 和 Linux（per CLAUDE.md），进程组隔离需要平台特定实现：
- **Linux/Darwin**: `SysProcAttr{Setpgid: true}` — 创建新进程组
- **Windows**: `SysProcAttr{CreationFlags: CREATE_NEW_PROCESS_GROUP}` — 等效隔离

使用 build tags 分离平台代码，避免运行时判断开销。

### 2. Zombie Process Reaper
- **Linux**: 真实 reaper（`syscall.Wait4` 循环）
- **Windows**: no-op（Windows OS 自动清理子进程，无僵尸问题）

这避免了在 Windows 上引用不存在的 `Wait4` 系统调用。

---

## Remaining Work (Wave 7)

**Wave 7** (32-07):
- P2-A6: AD 模块测试覆盖（LDAPClientIface, mock tests）
- 补全/删除空壳测试文件（stripBaseDN_test.go, dept_ou_mapper_test.go）

---

## Commit History (Wave 6)

1. `6230412` - docs(migrations): P2-A4 add source-tracking headers to conflicting files
2. `27662de` - fix(agent): P2-A7 subprocess Setpgid + reaper goroutine
3. `608f3f9` - fix(excel): P2-A8 wrap ImportData in single transaction

**Total**: 3 commits, 16 files changed, +461 / -57 lines

---

## Next Actions

1. ✅ Create this SUMMARY document (done)
2. ⏳ Update `.planning/STATE.md` to mark Wave 6 complete
3. ⏳ Begin Wave 7 execution (P2-A6 AD module test coverage)

---

**Sign-off**: Wave 6 successfully completed. All three P2 items (A4/A7/A8) resolved. Build green, cross-platform subprocess isolation in place, Excel imports now atomic. Ready for Wave 7 (final wave).
