# Phase 60: 软删恢复安全 - Pattern Map

**Mapped:** 2026-10-20
**Files analyzed:** 7（4 个生产文件 + 3 个测试文件）
**Analogs found:** 7 / 7（全部为同文件内既有模式或同包测试基座）

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/core/security/authenticator.go` | model/contract | request-response | 同文件 `ErrUserNotFound` sentinel 块（:84-91） | exact |
| `internal/core/security/ad_authenticator.go` | service（错误映射） | request-response | 同文件 `SyncErrorReason` 分支（:131-141） | exact |
| `internal/api/v1/auth.go` | controller（响应文案） | request-response | 同文件 `syncErrorReasonMessage()`（:653-668） | exact |
| `internal/services/system/user_sync_service.go` | service | CRUD / batch | 同文件 `findActiveUserUpdateQueue` + GUARD-01 WARN（:826-846 / :428-446） | exact |
| `internal/services/system/user_sync_service_phase60_test.go`（新建） | test | CRUD / batch | `user_sync_service_phase59_test.go` fixture + `user_sync_classify_test.go` 纯函数表 | exact |
| `internal/core/security/ad_authenticator_phase60_test.go`（新建） | test | transform | `internal/core/security/ad_authenticator_test.go` | role-match |
| `internal/api/v1/auth_80_03_test.go`（修改） | test | request-response | 同文件 `TestAuth8003_SyncErrorReasonMessage` 表驱动（:623-640） | exact |

> 所有 analog 路径均已用 `git ls-files -- <path>` 验证为 git-TRACKED 源文件，无 install/runtime mirror。

---

## Pattern Assignments

### `internal/core/security/authenticator.go`（contract/model，request-response）

**Analog:** 同文件 :84-91

**sentinel 错误声明模式**:
```go
// 认证标准错误定义
var (
	ErrUserNotFound       = errors.New("用户不存在")
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserDisabled       = errors.New("用户已被禁用")
	ErrADConfigNotFound   = errors.New("AD配置不存在")
	ErrADConnectionFailed = errors.New("AD连接失败")
)
```

**本 phase 新增（落点）**:
- 在同一个 `var (...)` 块追加 `ErrSoftDeleteIdentityMismatch = errors.New("软删用户恢复被拒绝: AD 身份不一致")`。
- 文件顶部已 import `errors`（:5-8），无需新增 import。
- 原因码常量建议与 `AuthResult.SyncErrorReason` 注释（:33-36）同文件定义：
```go
// SyncReasonSoftDeleteMismatch 是 AuthResult.SyncErrorReason 的专用原因码。
const SyncReasonSoftDeleteMismatch = "soft_delete_mismatch"
```
原因码必须用常量，禁止在 `ad_authenticator.go` 与 `auth.go` 各写字面量（Pitfall 7）。

---

### `internal/core/security/ad_authenticator.go`（service，request-response）

**Analog:** 同文件 :131-141

**现有错误分支模板**:
```go
syncedUser, err := a.userSyncer.SyncADUser(ctx, adUserInfo, defaultRoleID)
if err != nil {
	// 同步失败不影响认证成功，但标记需要同步
	applogger.Errorf("[ADAuth] 同步AD用户失败: username=%s, error=%v", adUserInfo.Username, err)
	return &AuthResult{
		User:            nil,
		AuthSource:      "ad",
		ADUserInfo:      adUserInfo,
		NeedsSync:       true,
		SyncErrorReason: "user_sync",
	}, nil
}
```

**改法（保留结构，仅替换原因码来源）**:
```go
SyncErrorReason: syncErrorReasonFor(err),
```
并新增包内纯函数（便于单测，无 DB/网络依赖）:
```go
func syncErrorReasonFor(err error) string {
	if errors.Is(err, ErrSoftDeleteIdentityMismatch) {
		return SyncReasonSoftDeleteMismatch
	}
	return "user_sync"
}
```
该文件已 import stdlib `errors`（`ad_authenticator.go:6`），**无需新增 import**（重复添加会编译失败）。可选：对策略拒绝把 Errorf 降为 Warnf（Research Open Question 4）。

---

### `internal/api/v1/auth.go`（controller，request-response）

**Analog:** 同文件 :653-668

**现有 switch 映射**:
```go
func syncErrorReasonMessage(reason string) string {
	switch reason {
	case "admin_dial", "admin_bind":
		return "AD 管理员账号配置异常（账号可能被锁定或密码错误），请联系系统管理员"
	case "user_search":
		return "AD 用户信息搜索失败，请联系系统管理员"
	case "user_sync":
		return "AD 用户同步到本地失败，请联系系统管理员"
	case "no_syncer":
		return "AD 用户同步服务未配置，请联系系统管理员"
	default:
		return "AD 用户信息同步失败，请联系系统管理员"
	}
}
```

**新增 case（引用常量，不写字面量）**:
```go
case security.SyncReasonSoftDeleteMismatch:
	// DN 细节绝不进用户文案（D-04）。
	return "该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员"
```
`auth.go` 已 import `security`，无需新增 import。上游 `authResult.User == nil` 分支（:211-216）保持不变，继续走 `response.Error(c, response.ErrServerError, ...)`。

---

### `internal/services/system/user_sync_service.go`（service，CRUD/batch）

**Analog 1（校验/日志风格）:** 同文件 :428-446（GUARD-01 结构化 WARN）

```go
applogger.Warnf("部门回写守卫查询失败，跳过 dept_id 更新: userID=%s, username=%s, localDeptID=%s, adDeptID=%s, err=%v",
	user.ID, user.Username, deptValue(user.DeptID), deptID, qerr)
```

**Analog 2（包内私有辅助函数形态）:** 同文件 :826-837
```go
func (s *UserSyncService) findActiveUserUpdateQueue(ctx context.Context, userID string) (*models.ADPushQueue, error) {
	var row models.ADPushQueue
	err := s.db.WithContext(ctx).
		Where("entity_id = ? AND entity_type = ? AND action_type = ?", ...).
		Where(models.ADPushQueueActivePredicate).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}
```

**共享校验函数（新增，建议紧邻 `restoreSoftDeletedUser` 之前）**:
```go
// checkSoftDeletedRestore 软删用户恢复身份校验（SAFE-01 / SAFE-02 共用入口）。
// 只读：拒绝时绝不清空 user.AdDn（D-03）。
func (s *UserSyncService) checkSoftDeletedRestore(user *models.User, currentUserDN, path string) bool {
	storedDN := ""
	if user.AdDn != nil {
		storedDN = *user.AdDn
	}
	normalizedStored := addomain.NormalizeDN(storedDN)

	// D-01: 空/NULL/纯空白 → 放行
	if normalizedStored == "" {
		return true
	}
	// D-02: 只归一化一次，复用 normalizedCurrent（本函数 NormalizeDN 恰好 2 次调用）
	normalizedCurrent := addomain.NormalizeDN(currentUserDN)
	if normalizedStored == normalizedCurrent {
		return true
	}

	// D-05: 两路径唯一审计日志产出点；DN 只进 WARN
	applogger.Warnf(
		"[SECURITY] 软删用户恢复被拒绝: AD 身份不一致, path=%s, userID=%s, username=%s, storedAdDN=%q, currentAdDN=%q, normalizedStored=%q, normalizedCurrent=%q",
		path, user.ID, user.Username, storedDN, currentUserDN,
		normalizedStored, normalizedCurrent,
	)
	return false
}
```
`addomain`、`models`、`applogger`、`fmt` 均已 import（:11-16）；`security` 也已 import（用于 sentinel）。

**调用点 1 — 登录路径**（`:272` `restoreSoftDeletedUser` 最前面，哈希与写库之前）:
```go
func (s *UserSyncService) restoreSoftDeletedUser(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync, defaultRoleID string) error {
	if !s.checkSoftDeletedRestore(user, adUser.UserDN, "login") {
		return fmt.Errorf("%w: username=%s", security.ErrSoftDeleteIdentityMismatch, adUser.Username)
	}
	// ... 既有 HashPassword → applyRestoreSoftDeleted → assignRole 不变
}
```
`SyncUserFromAD` 已有的 `fmt.Errorf("同步AD用户失败: %w", err)`（:105）会保留 `errors.Is` 链，无需改动。

**调用点 2 — 批量路径**（:533 阶段 2 分类之后、:539 阶段 3 哈希之前）:
```go
// SAFE-01/SAFE-02: 被拒项在哈希与任何写库前过滤，绕开 runInBatchedTx。
if items := plansByCat[catSoftDeleted]; len(items) > 0 {
	allowed := make([]*batchPlanItem, 0, len(items))
	for _, it := range items {
		if !s.checkSoftDeletedRestore(it.user, it.adUser.UserDN, "batch") {
			result.Skipped++
			result.Errors = append(result.Errors, SyncError{
				Username: it.adUser.Username,
				Error:    "AD 身份不一致，拒绝恢复（需管理员核对 ad_dn）",
			})
			continue
		}
		allowed = append(allowed, it)
	}
	plansByCat[catSoftDeleted] = allowed
}
```
关键约束（来自 `:732-770` `runInBatchedTx` 实测）：拒绝不得抛 error，否则触发「批次事务失败，降级逐个处理」并把拒绝计入 `Failed`。`BatchSyncResult`（:473-480）字段已齐备，零结构改动。

---

### `internal/services/system/user_sync_service_phase60_test.go`（新建 test，CRUD/batch）

**Analog:** `user_sync_service_phase59_test.go` + `user_sync_classify_test.go`

**Fixture 复用模式**（`user_sync_service_phase59_test.go:48-105`）:
```go
func setupPhase59GuardFixture(t *testing.T) *phase59GuardFixture {
	db := setupTestDBForSync(t)
	require.NotNil(t, db, "setupTestDBForSync 必须返回可用 DB")
	...
	f.svc = NewUserSyncService(db, security.NewPasswordManager(nil), addomain.NewDeptOUmapper(db))
	return f
}
```
Phase 60 应新建 `setupPhase60SoftDeleteFixture(t)`：
- 调 `setupTestDBForSync(t)`（`user_sync_service_test.go:16-165`，`sys_user` 已含 `ad_dn` / `deleted_at` 列）；
- 用 `db.Exec` 插入软删行（`deleted_at` 非空、`ad_dn` 非空）与可选活跃 local 行；
- `svc := NewUserSyncService(db, security.NewPasswordManager(nil), ...)`。批量用例把 `OuDn` 置空以避免 nil mapper（Pitfall 8）。

**日志断言复用**（`user_sync_service_phase59_test.go:142-148`）:
```go
func captureLogs(t *testing.T) (*bytes.Buffer, func()) {
	buf := &bytes.Buffer{}
	restore := applogger.SetTestBuffer(buf)
	return buf, restore
}
```
不要重复定义 `captureLogs` / `stringPtr` / `strPtrSyncTest`（Pitfall 9）；`strPtrSyncTest` 已由 `user_sync_classify_test.go:13` 提供。

**纯函数表驱动模式**（`user_sync_classify_test.go:15+`）:
```go
func TestPhase60_CheckSoftDeletedRestore_Predicate(t *testing.T) {
	tests := []struct {
		name      string
		storedDN  *string
		currentDN string
		want      bool
	}{
		{"nil 存量 DN 放行", nil, "CN=u,DC=corp,DC=com", true},
		{"空串存量 DN 放行", strPtrSyncTest(""), "CN=u,DC=corp,DC=com", true},
		{"纯空白存量 DN 放行", strPtrSyncTest("   "), "CN=u,DC=corp,DC=com", true},
		{"完全一致允许", strPtrSyncTest("CN=u,DC=corp,DC=com"), "CN=u,DC=corp,DC=com", true},
		{"大小写不同允许", strPtrSyncTest("cn=U,dc=CORP,dc=COM"), "CN=u,DC=corp,DC=com", true},
		{"段内空格允许", strPtrSyncTest("CN=u, DC=corp, DC=com"), "CN=u,DC=corp,DC=com", true},
		{"不同 DN 拒绝", strPtrSyncTest("CN=other,DC=corp,DC=com"), "CN=u,DC=corp,DC=com", false},
		{"存量非空但当前为空拒绝", strPtrSyncTest("CN=u,DC=corp,DC=com"), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &UserSyncService{}
			got := svc.checkSoftDeletedRestore(&models.User{Username: "u", AdDn: tt.storedDN}, tt.currentDN, "login")
			assert.Equal(t, tt.want, got)
		})
	}
}
```

**状态不变量断言模板**（D-03）:
```go
var after models.User
require.NoError(t, db.Unscoped().First(&after, "id = ?", userID).Error)
assert.True(t, after.DeletedAt.Valid, "拒绝恢复时用户必须保持软删")
require.NotNil(t, after.AdDn)
assert.Equal(t, storedDN, *after.AdDn, "拒绝路径绝不清空/改写存量 ad_dn")
```

**批量守恒断言**:
```go
assert.Equal(t, 1, result.Skipped)
assert.Equal(t, 0, result.Failed)
require.Len(t, result.Errors, 1)
assert.NotContains(t, result.Errors[0].Error, "CN=")
assert.Equal(t, result.Total, result.Success+result.Failed+result.Skipped)
```

---

### `internal/core/security/ad_authenticator_phase60_test.go`（新建 test，transform）

**Analog:** `internal/core/security/ad_authenticator_test.go`（package `security`，stdlib `errors` 已用）

**测试骨架**:
```go
package security

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhase60_SyncErrorReasonFor(t *testing.T) {
	assert.Equal(t, "user_sync", syncErrorReasonFor(errors.New("db down")))
	assert.Equal(t, SyncReasonSoftDeleteMismatch,
		syncErrorReasonFor(fmt.Errorf("wrap: %w", ErrSoftDeleteIdentityMismatch)))
}
```

---

### `internal/api/v1/auth_80_03_test.go`（修改 test，request-response）

**Analog:** 同文件 `TestAuth8003_SyncErrorReasonMessage`（:623-640）

在既有表驱动中追加一行（引用常量）:
```go
{security.SyncReasonSoftDeleteMismatch, "该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员"},
```
该文件已 import `security`（既有 `security.SyncErrorReason*` 使用），无需改 import。

---

## Shared Patterns

### 安全审计日志（D-05）
**Source:** `internal/services/system/user_sync_service.go:428-446`（Phase 59 GUARD-01 风格）
**Apply to:** `checkSoftDeletedRestore`（唯一产出点，两路径共用）
```go
applogger.Warnf("[SECURITY] 软删用户恢复被拒绝: AD 身份不一致, path=%s, userID=%s, username=%s, storedAdDN=%q, currentAdDN=%q, ...",
	path, user.ID, user.Username, storedDN, currentUserDN, ...)
```
规则：字段齐全（username / userID / 存量 DN / 当前 DN / path）；DN 细节只进 WARN，不进用户文案或 `SyncError.Error`。

### Sentinel 错误 + `errors.Is` 跨层传播
**Source:** `internal/core/security/authenticator.go:84-91` + `internal/services/system/user_sync_service.go:434`（既有 `errors.Is` 用法）
**Apply to:** `system` → `security` → `api/v1` 的错误链
- sentinel 声明在 `security`（避免 `security → system` 循环依赖）；
- service 用 `fmt.Errorf("%w: ...")` 包装；
- `ad_authenticator.syncErrorReasonFor` 用 `errors.Is` 判定；
- `auth.go` 只消费原因码常量。

### DN 归一化比较（D-02）
**Source:** `internal/services/addomain/dn_util.go:36-56`
**Apply to:** `checkSoftDeletedRestore`（本 phase 唯一调用点）
```go
normalized := addomain.NormalizeDN(dn) // 空串/纯空白 → ""
```
禁止裸字符串比较或自写 `strings.EqualFold`；判空必须用 `NormalizeDN(x) == ""`（不要仅判 `AdDn != nil`）。

### 批量拒绝记账（D-06）
**Source:** `internal/services/system/user_sync_service.go:473-480`（`BatchSyncResult`）+ `:732-770`（`runInBatchedTx`）
**Apply to:** `BatchSyncADUsers` 阶段 2 过滤
- 拒绝 = `Skipped++` + `Errors` 明细；
- 绝不以 error 进入 `runInBatchedTx`（否则降级重试 + 计入 `Failed`）；
- 从 `plansByCat[catSoftDeleted]` 移除被拒项，保证 `Total == Success+Failed+Skipped`。

### 测试基建复用
**Source:** `user_sync_service_test.go:16`（`setupTestDBForSync`）/ `user_sync_service_phase59_test.go:143`（`captureLogs`）/ `user_sync_classify_test.go:13`（`strPtrSyncTest`）
**Apply to:** 全部 Phase 60 测试
- 新 helper 统一加 `phase60` 前缀，避免 `redeclared in this block`；
- 批量用例将 `OuDn` 置空以跳过 `resolveDeptFromOU` 的 nil mapper 风险。

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| — | — | — | 无。所有目标文件均能在同文件或同包测试中找到直接可复制的先例。 |

## 明确不修改的文件（范围护栏）

| File | 原因 |
|------|------|
| `internal/services/addomain/user_ou_service.go` | Phase 62 HYGIENE-03 删除对象，F-22 对照语义载体（Pitfall 10） |
| `internal/services/system/user_sync_service.go` 中 `applyRestoreSoftDeleted` | 批量路径的校验**不得**下沉到这里（会钻进 `runInBatchedTx` 降级管道） |
| `ruoyi-react-frontend/**` | `skipped` 字段与登录响应管道已存在，零前端改动 |
| `internal/core/db/migrations/**` | 预计零迁移；如确需，编号从 `migration_219` 起 |

## Metadata

**Analog search scope:** `internal/core/security/`, `internal/api/v1/`, `internal/services/system/`, `internal/services/addomain/`, `internal/models/`
**Files scanned:** 12（含 3 个测试基座与 1 个前端类型引用）
**Tracked-source verification:** 全部 analog 通过 `git ls-files` 校验
**Pattern extraction date:** 2026-10-20
