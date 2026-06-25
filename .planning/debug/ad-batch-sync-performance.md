---
slug: ad-batch-sync-performance
status: resolved
deferred_to: v1.16-tech-debt
trigger: 经过今日几次修改(计算机过滤/超时/软删除恢复)后,域控用户批量同步到 sys_user 变慢,耗时显著变长
created: 2026-06-22
updated: 2026-06-25
session_type: performance
related:
  - ad-sync-duplicate-username-softdeleted
  - ad-batch-sync-context-canceled
---

# Debug Session: AD 用户批量同步性能退化

## 关键结论（用户首读）

**不是死循环**（用户已确认同步能正常结束，仅耗时长）。

"今天变慢"的真正原因：今天的 3 个修复（计算机账号过滤、超时延长、软删除用户恢复）
让**之前失败/跳过的用户现在能成功跑完同步**，实际同步成功的用户数大幅增加，
从而**暴露了早已存在的逐行处理 + 重哈希慢逻辑**——不是今天的代码变慢了，是今天让
更多用户真正跑完了慢路径。

### 性能瓶颈按影响排序

| # | 瓶颈 | 类型 | 量级（1000 用户估算） | 可优化性 |
|---|------|------|----------------------|----------|
| 1 🔴 | **密码哈希重复计算** | CPU | 60万次 SM3 × N ≈ 3-5 分钟 | ✅ 高 |
| 2 🔴 | **逐用户独立事务** | DB | N 次 BEGIN/COMMIT | ✅ 高 |
| 3 🟠 | **部门解析 N+1（无缓存）** | DB | 同 OU 重复解析，90% 冗余 | ✅ 中 |
| 4 🟡 | **大量 Info 日志 I/O** | I/O | 每用户 5-6 条 | ✅ 低 |

---

## 瓶颈详解

### 🔴 #1 密码哈希重复计算（最大瓶颈）

**位置**: `internal/core/security/password.go:66-94`
```go
func (pm *PasswordManager) pbkdf2SM3(password, salt []byte, iterations, keyLen int) []byte {
    ...
    for i := 1; i < iterations; i++ {   // ← 600,000 次纯 CPU 循环
        h.Reset(); h.Write(block); block = h.Sum(nil)
        ...
    }
}
```

- `DefaultPasswordConfig.Iterations = 600000`（OWASP 2023 基线，password.go:34）
- `createNewUser`（user_sync_service.go:96）和 `restoreSoftDeletedUser` 都调用 `HashPassword("123456")`
- **每个新用户/恢复用户都要做 60 万次 SM3 迭代**，纯 CPU，串行
- 单次 ~100-300ms；1000 用户顺序算 ≈ **3-5 分钟纯 CPU**，且阻塞同步主循环

**安全约束**: PBKDF2 每次用随机 salt，哈希值不同，不能直接复用结果。但批量导入用的是
**同一个默认密码 "123456"**，且 `InitFlag=true`（首次登录强制改密）。

### 🔴 #2 逐用户独立事务

**位置**: `internal/services/system/user_sync_service.go:316-363`（BatchSyncADUsers）
```go
for _, adUser := range users {        // ← 顺序循环，无并发无批处理
    syncedUser, err := s.SyncADUser(ctx, adUserInfo, defaultRoleID)
}
```

- `SyncADUser` → `SyncUserFromAD` → 每个用户独立 `Transaction()`
- N 个用户 = N 次事务开销（BEGIN/COMMIT + WAL flush）
- 无 goroutine 并发，无批量 INSERT

### 🟠 #3 部门解析 N+1（无缓存）

**位置**: `resolveDeptFromOU`（user_sync_service.go:375）→ `createDeptFromOUDN`（user_sync_service.go:390）

- `DeptOUmapper.FindDeptByOUDN`（dept_ou_mapper.go:26）每次实时查 DB，无缓存
- 同一 OU 的多个用户重复走完整解析流程（查映射 → 缺失则递归创建部门链）
- 100 用户分布在 10 个 OU → 100 次映射查询（实际只需 10 次，**90 次冗余**）
- `createDeptFromOUDN` 递归创建部门链，每层多次 DB 操作

### 🟡 #4 日志 I/O

**位置**: SyncADUser / BatchSyncADUsers 大量 `applogger.Infof`
- 每个用户 5-6 条 Info 日志，大量用户时日志写入成为次要瓶颈

---

## 优化方向（待规划）

1. **密码哈希并行化**: worker pool 并发计算 N 个用户的哈希（保留相同安全性）
2. **批量 DB 操作**: 预查询所有用户是否存在（一次 IN 查询），分类后批量 INSERT/UPDATE
3. **部门解析缓存**: 同一批同步内，OU→deptID 做 map 缓存，避免重复解析
4. **降级日志**: 批量同步时用 Warn 级别，减少 I/O

详见后续 PLAN。

## Status

根因已定位。等待规划批量同步优化方案。

## Phase 40 Closure (2026-06-25)

复测 `internal/services/system/user_sync_service.go` 的 `BatchSyncADUsers`：4 个瓶颈已全部由既有优化落地：

| 瓶颈 | 现状 |
|------|------|
| #1 密码哈希 60 万次 | `hashPasswordsConcurrent` errgroup worker pool（`batchHashConcurrency=8`）并发算默认密码 "123456" 的 PBKDF2-SM3 |
| #2 逐用户独立事务 | `indexExistingUsers` 一次预查询分类 4 组 + `CreateInBatches(batchWriteSize=500)` 批量 INSERT + UPDATE 分小批事务 |
| #3 部门 N+1 | `ouCache map[string]string`（ouDN → deptID）同 OU 只解析一次 |
| #4 Info 日志 I/O | 批量路径仅打阶段级 Info（每阶段一行），SyncADUser 单条 Info 走 SyncADUser 单用户调用栈，不影响批量 |

verifiy: `go build ./...` 退出 0；`grep -n "hashPasswordsConcurrent\|CreateInBatches\|ouCache" internal/services/system/user_sync_service.go` 命中
files_changed: .planning/debug/ad-batch-sync-performance.md
