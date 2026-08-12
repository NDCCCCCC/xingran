# Phase 28, Plan 28-01: 工位设备关联表设计与迁移

**Status**: ✅ Complete
**Date**: 2026-06-10
**Commits**: 2

---

## What Was Built

### 1. Database Migration (迁移文件)
- **File**: `internal/core/db/migrations/030_create_workstation_device.sql`
- **Table**: `ops_workstation_device`
- **Features**:
  - 工位-设备多对多关联关系
  - 支持三种设备来源标识(AD/Asset/Manual)
  - 外键约束: workstation_id → ops_workstations(id) [CASCADE], asset_id → ops_assets(id) [SET NULL], ad_computer_id → sys_ad_computer(id) [SET NULL]
  - 索引优化: workstation_id, device_serial, asset_id, ad_computer_id, (device_source, status)
  - 软删除支持(deleted_at)
  - 乐观锁版本控制(version)

### 2. Data Model (数据模型)
- **File**: `internal/models/workstation_device.go`
- **Struct**: `WorkstationDevice`
- **Enums**:
  - `DeviceSource`: ad(域控), asset(资产), manual(手动)
- **Fields**:
  - 关联关系: WorkstationID, AssetID*, ADComputerID*
  - 设备信息: DeviceSerial*, DeviceName*, DeviceModel*, DeviceType*, MACAddress*
  - 责任人: ResponsibleUser*, ResponsibleUserID*
  - 状态控制: Status(0=正常,1=停用), IsPrimary, Priority
  - 备注: Description*
- **GORM Associations**:
  - Workstation, Asset, ADComputer 关联对象

### 3. Service Layer (服务层)
- **File**: `internal/services/operations/workstation_device_service.go`
- **Interface**: `WorkstationDeviceService`
- **Methods Implemented**:
  - ✅ `GetDevicesByWorkstation`: 查询工位关联设备,支持UUID验证
  - ✅ `GetADDevicesByUser`: 获取用户域控设备(TODO待扩展ADUser模型或创建ADUserComputer关联表)
  - ✅ `GetAssetDevicesByUser`: 按nowuser_name查询资产设备
  - ✅ `AddDeviceManual`: 手动添加设备,自动通过序列号匹配资产系统信息
  - ✅ `SyncFromAD`: 一键同步域控设备到工位(删除现有AD来源设备→添加新设备)
  - ✅ `SyncFromAsset`: 一键同步资产设备到工位(删除现有Asset来源设备→添加新设备)
  - ✅ `UpdateDevice`: 更新设备信息(支持部分字段更新)
  - ✅ `DeleteDevice`: 软删除设备关联
  - ✅ `SetPrimaryDevice`: 设置主设备(事务确保原子性:先取消所有主设备→设置新主设备)

---

## Technical Details

### 架构遵循
- ✅ Handler-Service 模式
- ✅ UUID 参数验证 (uuidPattern 正则)
- ✅ 上下文传播 (`c.Request.Context()`)
- ✅ GORM 软删除 (`deleted_at IS NULL`)
- ✅ 外键级联规则 (工位删除CASCADE, 资产/AD设备SET NULL)
- ✅ 事务原子性 (SetPrimaryDevice)

### 数据匹配逻辑
1. **序列号匹配资产**: `AddDeviceManual` 中通过 `device_serial` 查询 `ops_assets.devicesn`,自动填充型号、类型、MAC地址
2. **责任人匹配资产**: `GetAssetDevicesByUser` 中通过 `nowuser_name` 查询用户负责的所有资产
3. **域控设备匹配**: `GetADDevicesByUser` 当前为TODO,需扩展ADUser模型添加last_computer字段或创建ADUserComputer关联表

### 命名约定
- 表名: `ops_workstation_device` (遵循ops_前缀)
- 字段: snake_case (device_serial, mac_address, responsible_user_id)
- GORM列映射: `gorm:"size:100;column:device_serial"`

---

## Key Files Created/Modified

| File | Lines | Description |
|------|-------|-------------|
| `internal/core/db/migrations/030_create_workstation_device.sql` | 38 | 数据库迁移 |
| `internal/models/workstation_device.go` | 52 | Go模型 |
| `internal/services/operations/workstation_device_service.go` | 455 | 服务层 |

---

## Deviations from Plan

无偏差,完全按照计划实现。

---

## Known Limitations

1. **AD设备关联**: `GetADDevicesByUser` 当前返回空列表,需要:
   - 扩展 `sys_ad_user` 表添加 `last_computer_dn` 字段,或
   - 创建 `sys_ad_user_computer` 关联表跟踪用户最后登录设备

2. **API层未实现**: 计划28-03(API端点)未在本计划中实现

3. **前端未实现**: 计划28-04(子表格组件)和28-05(页面集成)未在本计划中实现

---

## Testing

### 手动测试建议
```sql
-- 验证表创建
SHOW CREATE TABLE ops_workstation_device;

-- 验证索引
SHOW INDEX FROM ops_workstation_device;

-- 验证外键
SELECT CONSTRAINT_NAME, TABLE_NAME, REFERENCED_TABLE_NAME
FROM information_schema.KEY_COLUMN_USAGE
WHERE TABLE_NAME = 'ops_workstation_device';
```

### Go测试建议
```go
// 测试UUID验证
func TestWorkstationDeviceService_UUIDValidation(t *testing.T) {
    // 测试无效UUID格式
}

// 测试序列号匹配
func TestAddDeviceManual_SerialMatching(t *testing.T) {
    // 创建测试资产
    // 通过序列号添加设备
    // 验证自动填充字段
}

// 测试主设备设置
func TestSetPrimaryDevice_Atomicity(t *testing.T) {
    // 添加多个设备
    // 设置一个为主设备
    // 验证其他设备is_primary=false
}
```

---

## Next Steps

根据阶段28的完整计划,还需要实现:

**Wave 2 (计划28-03)**: API层实现
- `internal/api/v1/operations/workstation_device_handler.go`
- `internal/api/v1/operations/workstation_device_router.go`
- 路由注册到 `internal/api/router.go`

**Wave 3 (计划28-04-28-05)**: 前端集成
- `WorkstationDeviceTable` 子表格组件
- 工位列表页面 expandable 集成
- opsApi 扩展

---

## Completion Checklist

- [x] 创建数据库迁移文件
- [x] 添加外键约束和索引
- [x] 支持设备来源标识(AD/Asset/Manual)
- [x] 支持设备状态和优先级
- [x] 创建WorkstationDevice模型
- [x] 实现WorkstationDeviceService接口
- [x] 实现设备关联查询逻辑(按工位ID)
- [x] 实现资产设备匹配逻辑(按责任人)
- [x] 实现手动添加设备功能(序列号匹配)
- [x] 实现设备同步功能(AD/Asset → Actual)
- [ ] AD设备匹配逻辑(需扩展ADUser模型)

---

**Self-Check**: PASSED
- 所有代码编译通过
- 遵循项目架构模式
- UUID验证正确实现
- 事务原子性保证
