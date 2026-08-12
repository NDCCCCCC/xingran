# Route Registration for MAC Trajectory Page

## Frontend Implementation Status: ✅ Complete

The frontend trajectory page is fully implemented at:
- **Page**: `xingran-react-frontend/src/pages/network/mac/trajectory.tsx`
- **Component**: `xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx`
- **API**: `xingran-react-frontend/src/lib/api/networkApi.ts`

## Backend Menu Registration Required

To make the route accessible in the application, add a menu entry to `sys_menu` table:

```sql
INSERT INTO sys_menu (
  menu_name,
  menu_type,
  path,
  component,
  parent_id,
  order_num,
  visible,
  status,
  icon,
  create_by,
  update_by,
  created_at,
  updated_at
) VALUES (
  'MAC轨迹查询',
  'C',
  'network/mac/trajectory',
  'pages/network/mac/trajectory',
  (SELECT id FROM sys_menu WHERE path = 'network/mac' LIMIT 1),
  5,
  1,
  0,
  'line-chart',
  'admin',
  'admin',
  NOW(),
  NOW()
);
```

## Route Configuration

- **Path**: `/network/mac/trajectory`
- **Component**: `pages/network/mac/trajectory` (auto-resolved by routeGenerator)
- **Parent**: Network MAC Management (network/mac)
- **Icon**: line-chart

## Access Pattern

Once the menu is registered, the route will be:
1. Automatically discovered by DynamicRoutes.tsx
2. Accessible via sidebar navigation
3. Protected by authentication and permission middleware
4. Rendered with Layout wrapper

## Verification

After menu registration, verify:
1. Menu item appears in Network > MAC Management section
2. Clicking menu navigates to `/network/mac/trajectory`
3. Page loads without 404 errors
4. MAC address input and time range picker are functional
5. Query button triggers API call to `/network/history/trajectory`
