# 设备数据源逻辑重构 - 执行总结

## 状态: 已完成

## 实施概述

成功重构工位设备的数据源逻辑，将 AD 和资产来源的设备从"同步保存"改为"实时查询"，只保存手动添加的设备。

---

## ✅ 完成的功能

### 1. 后端服务层改造
**文件**: `internal/services/operations/workstation_device_service.go`

**新增方法**:
- `GetADDevices()` - 实时查询 AD 设备，不保存到数据库
- `GetAssetDevices()` - 实时查询资产设备，不保存到数据库
- `SetPrimaryAndSave()` - 将 AD/资产设备设为主设备并保存到数据库

**修改方法**:
- `GetDevicesByWorkstation()` - 添加可选的 `source` 参数过滤

### 2. 后端 API 层改造
**文件**: `internal/api/v1/operations/workstation_device_handler.go`

**新增接口**:
- `POST /ops/workstation-device/{id}/ad` - 实时查询 AD 设备
- `POST /ops/workstation-device/{id}/asset` - 实时查询资产设备
- `POST /ops/workstation-device/{id}/set-primary-and-save` - 设置主设备并保存

**路由注册**: `internal/api/router.go`

### 3. 前端 API 客户端改造
**文件**: `xingran-react-frontend/src/lib/opsApi.ts`

**修改方法**:
- `getManual()` - 获取手动添加的设备
- `getAD()` - 实时查询 AD 设备
- `getAsset()` - 实时查询资产设备
- `setPrimaryAndSave()` - 设置主设备并保存

**移除方法**:
- `syncAD()` - 不再需要同步保存接口
- `syncAsset()` - 不再需要同步保存接口

### 4. 前端组件改造
**文件**: `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx`

**实现内容**:
- 并行查询三种来源的设备（手动/AD/资产）
- 手动设备正常显示（可编辑、删除）
- AD/资产设备默认折叠在折叠面板中（只读，可设为主设备）
- 移除"同步AD/资产"按钮
- 点击"设为主设备"时弹窗确认是否保存到数据库

---

## 📋 数据源逻辑

### 存储策略

| 来源 | 存储方式 | 说明 |
|------|---------|------|
| **手动添加** | 持久化到数据库 | `device_source='manual'` |
| **AD同步** | 实时查询，不保存 | 每次展开时从 AD 域控查询 |
| **资产同步** | 实时查询，不保存 | 每次展开时从资产系统查询 |
| **主设备** | 可选持久化 | 用户确认后保存到数据库 |

### 查询流程

```
用户点击工位行展开
  ↓
并行发起三个请求：
  1. GET /ops/workstation-device/{id}?source=manual
  2. POST /ops/workstation-device/{id}/ad
  3. POST /ops/workstation-device/{id}/asset
  ↓
合并显示：
  - 手动设备：正常显示（可编辑、删除、设为主设备）
  - AD/资产设备：折叠显示（只读，可设为主设备）
```

### 主设备设置流程

```
用户点击"设为主设备"（手动设备）
  ↓
直接设置主设备 ✓

用户点击"设为主设备"（AD/资产设备）
  ↓
弹窗确认："是否将此设备信息同步到数据库？"
  ↓
确认后：
  - 创建手动设备记录（device_source='manual'）
  - 标记为主设备
  - 原主设备标记取消
```

---

## 🎨 用户界面变化

### 组件结构
```
┌─────────────────────────────────┐
│ 手动添加的设备（正常显示）      │
│ - 设备A [编辑] [删除] [☆]        │
│ - 设备B [编辑] [删除] [设主设备] │
├─────────────────────────────────┤
│ ▼ 域控设备（默认折叠，2台）     │
│   - AD设备1 [设主设备]          │
│   - AD设备2 [设主设备]          │
├─────────────────────────────────┤
│ ▼ 资产设备（默认折叠，1台）     │
│   - 资产设备1 [设主设备]        │
└─────────────────────────────────┘
```

### 移除的元素
- ❌ "同步AD设备" 按钮
- ❌ "同步资产设备" 按钮
- ❌ autoSync 属性

### 新增的元素
- ✅ 折叠面板显示 AD/资产设备
- ✅ 主设备设置确认对话框
- ✅ 实时查询逻辑

---

## 📝 代码变更文件

### 后端文件
1. `internal/services/operations/workstation_device_service.go` - 服务层改造
2. `internal/api/v1/operations/workstation_device_handler.go` - API 层新增接口
3. `internal/api/router.go` - 路由注册

### 前端文件
1. `xingran-react-frontend/src/lib/opsApi.ts` - API 客户端重构
2. `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx` - 组件重构

---

## ⚠️ 兼容性说明

### 数据库兼容
- 保留现有的 AD/资产来源数据（不清除）
- 新逻辑只查询 `device_source='manual'` 的设备
- 历史数据不影响新功能

### API 兼容
- 保留现有接口路径
- 新增接口不影响旧接口
- 前端逐步迁移到新接口

---

## ✅ 验证结果

### TypeScript 检查
✅ 通过 - 无类型错误

### 功能测试建议
1. 测试手动设备 CRUD 操作
2. 测试展开时并行查询三种来源
3. 测试折叠面板显示 AD/资产设备
4. 测试主设备设置流程（手动和 AD/资产设备）
5. 测试错误处理（AD/资产查询失败）

---

**执行日期**: 2026-06-10
**任务 ID**: 260610-device-source-refactor
