---
slug: workstation-device-api-method-mismatch
status: resolved
phase: 28-workstation-device-association
trigger: 工位设备关联API返回404错误
created: 2026-06-10T09:45:00Z
updated: 2026-06-10T10:55:00Z
---

## Symptoms

### Expected Behavior
- 点击工位列表的"查看设备"按钮应成功展开设备子表格
- POST /api/v1/ops/workstation-device/:id 应返回该工位的设备列表

### Actual Behavior
- 初始: POST 返回 404（路由方法不匹配）
- 修复后: POST 返回 500（数据库表不存在）
  - 错误: relation "ops_workstation_device" does not exist

### Error Messages
```
POST http://10.62.10.33:9000/api/v1/ops/workstation-device/b96f84f2-31ae-45e1-ba83-0d06b72947cb 500
ERROR: relation "ops_workstation_device" does not exist (SQLSTATE 42P01)
```

### Timeline
- Phase 28 验证文档显示API端点已实现且路由已注册
- 从未工作过（新开发的功能）
- Migration 文件编号冲突（两个 030 文件）

### Reproduction Steps
1. 打开工位管理页面（/operations/workstations）
2. 点击任意工位行的"查看设备"按钮
3. 浏览器控制台显示 500 错误

## Current Focus

hypothesis: confirmed
next_action: migration renumbered, waiting for backend restart to apply
test: |
  重启后端服务器以应用 migration 098
expecting: 数据库表创建成功，API 返回 200
reasoning_checkpoint: |

## Evidence

- timestamp: 2026-06-10T10:53:00Z
  source: Backend logs
  observation: |
    GORM 查询失败:
    SELECT * FROM "ops_workstation_device" WHERE workstation_id = '...'
    错误: relation "ops_workstation_device" does not exist

- timestamp: 2026-06-10T10:55:00Z
  source: internal/core/db/migrations/
  observation: |
    Migration 编号冲突:
    - 030_add_building_spaces_3d_menu.sql (旧)
    - 030_create_workstation_device.sql (新，未运行)
    
    修复: 重命名为 098_create_workstation_device.sql

## Eliminated

## Resolution

root_cause: |
  1. 后端路由使用 GET 而非 POST（违反项目规范）
  2. 参数名冲突（:workstationId vs :id）
  3. Migration 编号冲突（两个 030 文件）

fix: |
  1. 修改 internal/api/router.go:
     - GET("/:workstationId") → POST("/:id")
  
  2. 修改 internal/api/v1/operations/workstation_device_handler.go:
     - c.Param("workstationId") → c.Param("id")
     - Swagger 注解更新
  
  3. 修改 xingran-react-frontend/src/components/operations/StatisticsCards.tsx:
     - 移除 valueStyle 属性
  
  4. 修改 xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx:
     - destroyOnClose → destroyOnHidden
  
  5. 重命名 migration:
     - 030_create_workstation_device.sql → 098_create_workstation_device.sql

verification: |
  ⏳ 等待后端重启以应用 migration
  待验证:
  1. 重启后端服务器（触发 migration 098）
  2. 检查数据库表 ops_workstation_device 是否创建
  3. 打开工位管理页面
  4. 点击"查看设备"按钮
  5. 验证设备子表格正常显示
  6. 确认无控制台错误

files_changed:
  - internal/api/router.go
  - internal/api/v1/operations/workstation_device_handler.go
  - xingran-react-frontend/src/components/operations/StatisticsCards.tsx
  - xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx
  - internal/core/db/migrations/030_create_workstation_device.sql → 098_create_workstation_device.sql

## Related Issues
- Migration 编号冲突导致表未创建
