---
slug: ad-batch-sync-dept-default
status: resolved
trigger: ad域控用户同步到系统用户，ad_ou_dn也成功同步，但是部门还是不正确，被同步到了默认部门，这里应该和AD域控登录逻辑一样，根据ou解析部门！
created: 2026-05-27
updated: 2026-05-27
---

## Symptoms

### Expected Behavior
部门应该根据 OU 解析（像 AD 登录一样） - 批量同步时应该根据 ad_ou_dn 字段自动匹配到正确的部门

### Actual Behavior
所有用户都被分配到默认部门 - 即使 ad_ou_dn 已正确同步，部门仍不正确

### Error Messages
None explicitly mentioned

### Timeline
- ad_ou_dn sync was recently implemented and is working correctly
- Department resolution issue discovered after implementing ad_ou_dn sync

### Reproduction
- Perform batch sync of AD users to system users
- ad_ou_dn field is correctly populated
- Department field shows default department instead of OU-matched department

### Investigation Notes
- User is unsure where AD login department resolution logic is located
- Need to search codebase for AD login department matching logic
- batch sync logic may be missing department resolution step that exists in login flow

## Current Focus

**Status:** ROOT CAUSE FOUND - Moving to resolution

## Resolution

### Root Cause
`UserSyncService.createNewUser()` only sets default department, never calls `DeptOUmapper.FindDeptByOUDN()` to resolve department from OU DN. The batch sync flow has the `OuDn` data available and stores it in `ad_ou_dn`, but never uses it to resolve the corresponding department ID.

### Fix Required
**File:** `../internal/services/system/user_sync_service.go`

**Changes needed:**
1. Add `DeptOUmapper` dependency to `UserSyncService` struct
2. Update `NewUserSyncService` constructor to inject `DeptOUmapper`
3. Modify `createNewUser()` method to resolve department from OU before falling back to default
4. Update `updateExistingUser()` to also resolve department when OU changes

**Implementation:**
```go
// 1. Add import
import "github.com/xingran-next/xingran-go-backend/internal/services/addomain"

// 2. Update struct
type UserSyncService struct {
	db         *gorm.DB
	pwdManager PasswordManager
	ouMapper   *addomain.DeptOUmapper  // NEW
}

// 3. Update constructor
func NewUserSyncService(db *gorm.DB, pwdManager PasswordManager, ouMapper *addomain.DeptOUmapper) *UserSyncService {
	return &UserSyncService{
		db:         db,
		pwdManager: pwdManager,
		ouMapper:   ouMapper,  // NEW
	}
}

// 4. Add department resolution method
func (s *UserSyncService) resolveDeptFromOU(ctx context.Context, ouDN string) string {
	if ouDN == "" {
		return ""
	}
	deptID, err := s.ouMapper.FindDeptByOUDN(ctx, ouDN)
	if err != nil {
		applogger.Warnf("[UserSyncService] Failed to resolve department from OU DN %s: %v", ouDN, err)
		return ""
	}
	return deptID
}

// 5. Update createNewUser method (line 99-102)
func (s *UserSyncService) createNewUser(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync, defaultRoleID string) error {
	// ... existing code ...

	// Try to resolve department from OU DN first
	deptID := s.resolveDeptFromOU(context.Background(), adUser.OuDn)
	if deptID == "" {
		// Fallback to default department
		deptID = s.getDefaultDeptID(tx)
	}
	if deptID != "" {
		user.DeptID = &deptID
	}

	// ... rest of existing code ...
}
```

### Testing Required
1. Test batch sync with users having valid OU DN mappings
2. Test batch sync with users having invalid/unmapped OU DN (should fall back to default)
3. Test batch sync with users having empty OU DN (should use default)
4. Verify existing user sync still works

## Evidence

### 2026-05-27T12:00:00Z - Root Cause Analysis

**Source:** Code analysis of AD sync services

**Findings:**
1. `UserSyncService.createNewUser()` (line 100-102) only sets default department:
   ```go
   if defaultDeptID := s.getDefaultDeptID(tx); defaultDeptID != "" {
       user.DeptID = &defaultDeptID
   }
   ```

2. `DeptOUmapper.FindDeptByOUDN()` exists in `../internal/services/addomain/dept_ou_mapper.go` (line 26-38) but is NEVER called:
   ```go
   func (m *DeptOUmapper) FindDeptByOUDN(ctx context.Context, ouDN string) (string, error) {
       var mapping models.DeptOUMapping
       err := m.db.WithContext(ctx).
           Where("ou_dn = ? AND sync_enabled = ?", ouDN, true).
           First(&mapping).Error
       if err != nil {
           if err == gorm.ErrRecordNotFound {
               return "", fmt.Errorf("未找到OU DN %s 对应的部门", ouDN)
           }
           return "", fmt.Errorf("查询OU DN映射失败: %w", err)
       }
       return mapping.DeptID, nil
   }
   ```

3. `ad_ou_dn` field is being populated correctly (line 88 in createNewUser):
   ```go
   user.AdOuDn = &adOuDn
   ```

4. **Key Finding:** Searched entire codebase - `FindDeptByOUDN` is defined but never called!

5. **Additional Finding:** AD login flow also does NOT resolve department from OU - this is a missing feature in both flows

**Root Cause:** The batch sync flow has the `OuDn` data available and stores it in `ad_ou_dn`, but never uses it to resolve the corresponding department ID. The `DeptOUmapper` service exists with the exact functionality needed, but it's not integrated into the user sync service.

**Impact:** All batch-synced users get the default department instead of their OU-matched department. Same issue affects AD login flow.

## Eliminated

### 2026-05-27T12:05:00Z - AD Login Flow Check
- **Eliminated:** AD login flow does NOT resolve department from OU either
- **Evidence:** Searched all authentication code - no OU-to-department resolution exists in login flow
- **Conclusion:** This is a missing feature in both batch sync AND AD login flows. Both use default department only.

