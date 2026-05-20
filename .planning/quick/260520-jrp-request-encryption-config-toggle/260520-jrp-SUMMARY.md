# Quick Task 260520-jrp: 在参数管理页面添加请求加密开关配置参数

## Task Description
在参数管理页面添加请求加密开关配置参数，允许管理员在前端界面上动态控制请求加密功能的启停

## Status
✅ **COMPLETE**

## Implementation Summary

### Backend Changes

#### 1. Database Migration (Commit: fb5cb76)
- **File**: `internal/core/db/migrations/migration_086_request_encryption_toggle.go`
- Created `sys_config` entry for request encryption toggle
- Config key: `sys.request.encryption.enabled`
- Default value: `true` (enabled)
- System parameter (cannot be deleted)
- Idempotent migration (checks existence before insert)

#### 2. Dynamic Config Reading (Commit: d4f7071)
- **Modified Files**:
  - `pkg/middleware/request_decryption.go` - Added database config reading
  - `internal/services/system/config_service.go` - Added encryption config validation
  - `internal/api/router.go` - Pass database instance to middleware

- **Key Features**:
  - 30-second TTL cache to minimize DB queries
  - Thread-safe config cache with read-write locks
  - Fallback to static config if DB fails (graceful degradation)
  - Automatic config value parsing ("true"/"false"/"1"/"0")
  - `RefreshEncryptionConfigCache()` for manual cache invalidation
  - Validation in config update (T-QUICK-01, T-QUICK-05 mitigation)

### Frontend Changes

#### 3. UI Implementation (Commit: a0cddc9)
- **File**: `ruoyi-react-frontend/src/pages/system/config/index.tsx`
- Added encryption status card at top of page
  - Visual indicators (green=enabled, red=disabled)
  - Security impact messages
  - Quick toggle button with confirmation modal
- Added special badge in table for encryption config
- Implemented dynamic config loading and cache refresh

## Testing Checklist

- [x] Database migration runs successfully
- [x] Middleware reads config from database
- [x] Cache TTL works (30 seconds)
- [x] Frontend displays encryption status correctly
- [x] Toggle button changes config value
- [x] Cache is refreshed after config update
- [x] TypeScript compilation passes
- [x] All changes committed atomically

## Usage Example

1. Navigate to: System > Config Management
2. View encryption status card at top of page
3. Click "Enable/Disable Encryption" button
4. Confirm in modal dialog
5. Config changes take effect within 30 seconds

## Security Considerations

- System parameter cannot be deleted (isSystem=1)
- Config changes logged in audit trail
- Confirmation modal prevents accidental changes
- Clear security warnings when encryption is disabled
- Graceful fallback if database query fails

## Commits

| Commit | Message | Files Changed |
|--------|---------|---------------|
| fb5cb76 | feat(260520-jrp): add database migration for request encryption toggle config | 2 files |
| d4f7071 | feat(260520-jrp): implement dynamic config reading in request decryption middleware | 5 files |
| a0cddc9 | feat(quick-260520-jrp): add UI for request encryption toggle in config management | 1 file |

## Notes

- Total implementation time: ~15 minutes
- All atomic commits completed successfully
- No breaking changes to existing functionality
- Backward compatible with existing deployments
