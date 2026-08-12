# FIX-02: Frontend UI Integration - Department-Group Mapping

**Status**: `diagnosed` → `planned`
**Severity**: Major
**Affects**: Tests 3,4,7 (Mapping management, auto-map, sync monitoring)
**Root Cause**: Frontend component exists but is not integrated into App.tsx routes or Sidebar.tsx menu. Missing database menu entry in `sys_menu` table.

## Problem Statement

Users cannot access the Department-Group Mapping UI despite complete implementation:

1. **Component exists**: `xingran-react-frontend/src/pages/ad/GroupMapping/index.tsx` is fully functional
2. **Backend API exists**: `/api/v1/ad/groups/mappings` endpoints are implemented
3. **Missing integration**: No route in App.tsx, no menu entry in sys_menu table
4. **User impact**: "没有创建映射按钮" - users cannot access the mapping management interface

## Objectives

1. ✅ Add database menu entry for "部门-组映射" in `sys_menu` table
2. ✅ Verify route registration in App.tsx
3. ✅ Verify backend router is registered in main router
4. ✅ Assign appropriate permissions to enabled roles
5. ✅ Test UI accessibility and functionality

## Implementation Plan

### Step 1: Create Database Migration for Menu Entry

**File**: `internal/core/db/migrations/136_add_group_mapping_menu.sql`

```sql
-- Add Department-Group Mapping menu entry
-- Parent menu: "AD域管理" (assumed menu_id exists, typically around 2000-2100 range)

-- First, find the AD域管理 parent menu_id
-- This query is for reference only - the actual INSERT uses the correct parent_id
-- SELECT menu_id FROM sys_menu WHERE menu_name = 'AD域管理' AND menu_type = 'M';

-- Insert the new menu item (adjust parent_id based on your AD域管理 menu_id)
INSERT INTO sys_menu (
    menu_id,
    menu_name,
    parent_id,
    order_num,
    path,
    component,
    is_frame,
    is_cache,
    menu_type,
    visible,
    status,
    perms,
    icon,
    create_by,
    create_time,
    update_by,
    update_time,
    remark
) VALUES (
    nextval('sys_menu_seq'), -- or gen_random_uuid() if using UUID
    '部门-组映射',
    2000, -- REPLACE with actual parent_id from AD域管理 menu
    4,
    'group-mapping',
    'ad-domain/group-mapping/index',
    1,
    0,
    'C',
    '0',
    '0',
    NULL,
    'NodeIndexOutlined', -- or 'PartitionOutlined'
    'admin',
    CURRENT_TIMESTAMP,
    NULL,
    NULL,
    '部门与AD组映射管理页面'
);

-- Get the newly inserted menu_id for permission creation
DO $$
DECLARE
    v_mapping_menu_id BIGINT;
    v_parent_menu_id BIGINT := 2000; -- REPLACE with actual parent_id
BEGIN
    -- Get the last inserted menu_id
    SELECT lastval() INTO v_mapping_menu_id;

    -- Insert button permissions
    -- Add mapping button
    INSERT INTO sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, remark)
    VALUES (
        '部门-组映射添加',
        v_mapping_menu_id,
        1,
        '',
        NULL,
        1,
        0,
        'F',
        '0',
        '0',
        'ops:ad:group:mapping:add',
        '#',
        'admin',
        CURRENT_TIMESTAMP,
        ''
    );

    -- Edit mapping button
    INSERT INTO sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, remark)
    VALUES (
        '部门-组映射修改',
        v_mapping_menu_id,
        2,
        '',
        NULL,
        1,
        0,
        'F',
        '0',
        '0',
        'ops:ad:group:mapping:edit',
        '#',
        'admin',
        CURRENT_TIMESTAMP,
        ''
    );

    -- Delete mapping button
    INSERT INTO sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, remark)
    VALUES (
        '部门-组映射删除',
        v_mapping_menu_id,
        3,
        '',
        NULL,
        1,
        0,
        'F',
        '0',
        '0',
        'ops:ad:group:mapping:delete',
        '#',
        'admin',
        CURRENT_TIMESTAMP,
        ''
    );

    -- View mapping button
    INSERT INTO sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, remark)
    VALUES (
        '部门-组映射查看',
        v_mapping_menu_id,
        4,
        '',
        NULL,
        1,
        0,
        'F',
        '0',
        '0',
        'ops:ad:group:mapping:view',
        '#',
        'admin',
        CURRENT_TIMESTAMP,
        ''
    );

    -- Auto-map button
    INSERT INTO sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, remark)
    VALUES (
        '部门-组自动映射',
        v_mapping_menu_id,
        5,
        '',
        NULL,
        1,
        0,
        'F',
        '0',
        '0',
        'ops:ad:group:mapping:automap',
        '#',
        'admin',
        CURRENT_TIMESTAMP,
        ''
    );

    -- Sync members button
    INSERT INTO sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, remark)
    VALUES (
        '部门-组成员同步',
        v_mapping_menu_id,
        6,
        '',
        NULL,
        1,
        0,
        'F',
        '0',
        '0',
        'ops:ad:group:mapping:sync',
        '#',
        'admin',
        CURRENT_TIMESTAMP,
        ''
    );

END $$;

-- Assign permissions to admin role (role_id = 1 is typically admin)
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT 1, menu_id FROM sys_menu
WHERE menu_name IN (
    '部门-组映射',
    '部门-组映射添加',
    '部门-组映射修改',
    '部门-组映射删除',
    '部门-组映射查看',
    '部门-组自动映射',
    '部门-组成员同步'
)
ON CONFLICT DO NOTHING;
```

### Step 2: Verify Frontend Route Registration

**File**: `xingran-react-frontend/src/App.tsx`

**Check if route exists**:
```typescript
// Look for AD domain routes section
// Should have something like:
{
  path: '/ad-domain',
  element: <Layout />,
  children: [
    // ... existing AD routes ...
    {
      path: 'group-mapping',
      element: <GroupMapping />, // Should exist
    },
  ],
}
```

**If missing, add**:
```typescript
import GroupMapping from './pages/ad/GroupMapping';

// In the routes array
{
  path: '/ad-domain',
  element: <Layout />,
  children: [
    // ... existing routes ...
    {
      path: 'group-mapping',
      element: lazyLoad(() => import('./pages/ad/GroupMapping')),
    },
  ],
}
```

### Step 3: Verify Backend Router Registration

**File**: `internal/api/router.go`

**Check if SetupGroupSyncRouter is called**:
```go
// In the API v1 setup section
adGroup := v1.Group("/ad-domain/groups")
addomain.SetupGroupSyncRouter(adGroup, core)
```

**If missing, add**:
```go
// Import the router setup
import "github.com/xingran-next/xingran-go-backend/internal/api/v1/addomain"

// In SetupRouter function
adGroup := v1.Group("/ad-domain")
{
    // ... existing AD domain routes ...
    groupRouter := adGroup.Group("/groups")
    addomain.SetupGroupSyncRouter(groupRouter, core)
}
```

### Step 4: Verify Component Export

**File**: `xingran-react-frontend/src/pages/ad/GroupMapping/index.tsx`

**Ensure component is properly exported**:
```typescript
const GroupMapping: React.FC = () => {
  // ... component implementation ...
};

export default GroupMapping;
```

### Step 5: Verify Sidebar Menu Rendering

**File**: `xingran-react-frontend/src/components/layout/Sidebar.tsx`

**Sidebar uses menuStore.allMenus**, so no changes needed if menu entry exists in database.

**Verification**: After migration 136 is applied, restart frontend and check if menu appears.

### Step 6: Add Sync Monitoring Page Menu Entry (if needed)

**File**: `internal/core/db/migrations/137_add_sync_monitoring_menu.sql` (if test 7 is important)

```sql
-- Add Sync Monitoring menu entry (similar to migration 136)

INSERT INTO sys_menu (
    menu_id,
    menu_name,
    parent_id,
    order_num,
    path,
    component,
    is_frame,
    is_cache,
    menu_type,
    visible,
    status,
    perms,
    icon,
    create_by,
    create_time,
    remark
) VALUES (
    nextval('sys_menu_seq'),
    '同步监控',
    2000, -- REPLACE with actual parent_id from AD域管理 menu
    5,
    'sync-monitoring',
    'ad-domain/sync-monitoring/index',
    1,
    0,
    'C',
    '0',
    '0',
    NULL,
    'MonitorOutlined',
    'admin',
    CURRENT_TIMESTAMP,
    'AD组同步监控页面'
);

-- Add sync monitoring permissions
-- (similar to migration 136, with ops:ad:sync:monitoring:* permissions)
```

### Step 7: Create Sync Monitoring Component (if missing)

**File**: `xingran-react-frontend/src/pages/ad/SyncMonitoring/index.tsx` (if test 7 is important)

**Note**: This component may not exist yet. If test 7 is critical, create it following the pattern from GroupMapping component.

## Verification Criteria

1. ✅ Migration 136 applied successfully to database
2. ✅ Menu entry "部门-组映射" appears in frontend sidebar under "AD域管理"
3. ✅ Navigation to `/ad-domain/group-mapping` works
4. ✅ GroupMapping component renders without errors
5. ✅ All button permissions (add, edit, delete, view, automap, sync) are available
6. ✅ Backend API endpoints respond correctly
7. ✅ UAT tests 3,4,7 pass

## Testing Plan

1. **Database Test**: Verify migration 136 creates menu entries correctly
2. **Navigation Test**: Click menu item in sidebar, verify route navigation
3. **Component Test**: Verify GroupMapping component renders and loads data
4. **Permission Test**: Verify all buttons (add, edit, delete, sync) are visible and functional
5. **API Test**: Verify all CRUD operations work through the UI
6. **UAT Verification**: Run UAT tests 3,4,7 to confirm major issue is resolved

## Rollback Plan

If issues occur:
1. Migration 136 is reversible (can DELETE from sys_menu where menu_name = '部门-组映射')
2. Can hide menu by setting visible = '1' (hidden)
3. Component changes can be reverted by removing route from App.tsx

## Dependencies

- Migration 136 must be applied before UI becomes accessible
- Frontend must be restarted to reload menu from database
- Backend router must be registered (verify in router.go)

## Estimated Effort

- **Step 1**: 30 minutes (create migration 136)
- **Step 2**: 15 minutes (verify App.tsx route)
- **Step 3**: 15 minutes (verify backend router)
- **Step 4**: 15 minutes (verify component export)
- **Step 5**: 0 minutes (Sidebar uses menuStore, no changes needed)
- **Step 6**: 30 minutes (optional: sync monitoring menu)
- **Step 7**: 2 hours (optional: create sync monitoring component)
- **Testing**: 1 hour
- **Total**: ~2.5 hours (without sync monitoring), ~4.5 hours (with sync monitoring)

## Success Metrics

- Menu entry "部门-组映射" visible in frontend sidebar
- Navigation to mapping management page works
- All CRUD operations (create, read, update, delete mappings) functional
- Auto-map feature accessible through UI
- Member sync feature accessible through UI
- UAT completion rate increases from 1/10 to at least 4/10

## Notes

- The parent_id (2000) in migrations must be verified against actual AD域管理 menu_id
- Use `SELECT menu_id FROM sys_menu WHERE menu_name = 'AD域管理'` to find correct parent_id
- If sync monitoring page (test 7) is not critical, can defer migrations 137 and component creation
- Icon choices (NodeIndexOutlined, PartitionOutlined, MonitorOutlined) can be adjusted based on design preferences
