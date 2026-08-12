# 设备数据源逻辑重构计划

## 任务描述

重构工位设备的数据源逻辑，将AD和资产来源的设备从"同步保存"改为"实时查询"，只保存手动添加的设备。

## 设计概述

### 数据存储策略
- **手动添加**: 持久化到数据库 (`device_source='manual'`)
- **AD同步**: 实时查询，不保存
- **资产同步**: 实时查询，不保存
- **主设备**: 用户设置主设备时，可将AD/资产设备保存到数据库

### 前端查询流程
```
用户点击工位行展开
  ↓
并行发起三个请求：
  1. 获取手动设备：GET /ops/workstation-device/{workstationId}?source=manual
  2. 获取AD设备：GET /ops/workstation-device/{workstationId}/ad
  3. 获取资产设备：GET /ops/workstation-device/{workstationId}/asset
  ↓
合并显示：
  - 手动设备：正常显示（可编辑、删除、设为主设备）
  - AD/资产设备：默认折叠（只读，可设为主设备）
```

### 主设备设置流程
```
用户点击"设为主设备"（AD或资产设备）
  ↓
弹窗确认："是否将此设备信息同步到数据库？"
  ↓
确认后：
  - 创建手动设备记录（device_source='manual'）
  - 标记为主设备
  - 原主设备标记取消
```

## 关键文件

### 后端文件
- `internal/services/operations/workstation_device_service.go` - 设备服务
- `internal/api/v1/operations/workstation_device_handler.go` - 设备处理器
- `internal/models/` - 设备模型

### 前端文件
- `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx` - 设备表格组件
- `xingran-react-frontend/src/lib/opsApi.ts` - API 客户端

## 实施步骤

### 1. 后端 - 服务层改造

**新增方法**:
```go
// GetADDevices 实时查询AD设备（不保存）
GetADDevices(ctx context.Context, workstationID string) ([]WorkstationDevice, error)

// GetAssetDevices 实时查询资产设备（不保存）
GetAssetDevices(ctx context.Context, workstationID string) ([]WorkstationDevice, error)

// SetPrimaryAndSave 设置主设备并保存到数据库
SetPrimaryAndSave(ctx context.Context, deviceID string) error
```

**修改方法**:
```go
// GetDevicesByWorkstation 添加来源过滤
GetDevicesByWorkstation(ctx context.Context, workstationID string, source string) ([]WorkstationDevice, error)
```

**保留方法**:
```go
// GetADDevicesByUser 保留（用于实时查询）
// GetAssetDevicesByUser 保留（用于实时查询）
```

**废弃方法**:
```go
// SyncFromAD 改为内部调用或标记为废弃
// SyncFromAsset 改为内部调用或标记为废弃
```

### 2. 后端 - API层改造

**新增接口**:
```
GET /ops/workstation-device/{id}?source=manual
→ 只返回手动添加的设备

GET /ops/workstation-device/{id}/ad
→ 实时查询AD域控设备

GET /ops/workstation-device/{id}/asset
→ 实时查询资产系统设备

POST /ops/workstation-device/{id}/set-primary-and-save
→ 设置主设备并保存到数据库
```

**修改接口**:
```
GET /ops/workstation-device/{id}
→ 默认只返回手动设备（兼容旧调用）
```

**移除接口**:
```
POST /ops/workstation-device/sync-ad
POST /ops/workstation-device/sync-asset
```

### 3. 前端 - API客户端改造

**修改 opsApi.ts**:
```typescript
export const workstationDeviceApi = {
  // 获取手动设备
  getManual: async (workstationId: string) => {
    return await post<WorkstationDevice[]>(`/ops/workstation-device/${workstationId}?source=manual`, {});
  },

  // 实时查询AD设备
  getAD: async (workstationId: string) => {
    return await post<WorkstationDevice[]>(`/ops/workstation-device/${workstationId}/ad`, {});
  },

  // 实时查询资产设备
  getAsset: async (workstationId: string) => {
    return await post<WorkstationDevice[]>(`/ops/workstation-device/${workstationId}/asset`, {});
  },

  // 设置主设备并保存
  setPrimaryAndSave: async (deviceId: string, data: SetPrimaryAndSaveRequest) => {
    return await post(`/ops/workstation-device/${deviceId}/set-primary-and-save`, data);
  },

  // 保留现有方法
  addManual: ...,
  update: ...,
  delete: ...,
  setPrimary: ...,
}
```

### 4. 前端 - 组件改造

**修改 WorkstationDeviceTable**:
```typescript
// 状态管理
const [manualDevices, setManualDevices] = useState<WorkstationDevice[]>([]);
const [adDevices, setADDevices] = useState<WorkstationDevice[]>([]);
const [assetDevices, setAssetDevices] = useState<WorkstationDevice[]>([]);
const [adExpanded, setADExpanded] = useState(false);
const [assetExpanded, setAssetExpanded] = useState(false);

// 并行查询三种来源
useEffect(() => {
  const fetchAllDevices = async () => {
    const [manual, ad, asset] = await Promise.all([
      workstationDeviceApi.getManual(workstationId),
      workstationDeviceApi.getAD(workstationId).catch(() => ({ data: [] })),
      workstationDeviceApi.getAsset(workstationId).catch(() => ({ data: [] })),
    ]);
    setManualDevices(manual.data || []);
    setADDevices(ad.data || []);
    setAssetDevices(asset.data || []);
  };
  fetchAllDevices();
}, [workstationId]);

// 渲染：手动设备正常显示，AD/资产设备折叠显示
```

### 5. 主设备设置流程

**新增确认对话框**:
```typescript
const handleSetPrimaryAndSave = async (device: WorkstationDevice) => {
  if (device.deviceSource !== 'manual') {
    Modal.confirm({
      title: '设置主设备',
      content: '是否将此设备信息同步到数据库？同步后可手动编辑。',
      onOk: async () => {
        await workstationDeviceApi.setPrimaryAndSave(device.id, {
          ...device,
          workstationId,
        });
        message.success('设置成功');
        fetchAllDevices();
      },
    });
  } else {
    // 手动设备直接设置主设备
    await workstationDeviceApi.setPrimary(device.id);
    message.success('设置成功');
    fetchAllDevices();
  }
};
```

## 兼容性考虑

### 数据库兼容
- 保留历史数据：不清除现有的AD/资产来源数据
- 查询过滤：在 `GetDevicesByWorkstation` 中添加 `source` 参数过滤
- 逐步迁移：先上线新逻辑，观察稳定后再清理历史数据

### API兼容
- 保留现有接口路径，添加可选的 `source` 参数
- 前端逐步迁移到新接口
- 旧接口标记为 `@Deprecated`（在注释中说明）

## 测试要点

1. **后端测试**
   - 测试手动设备CRUD操作
   - 测试AD设备实时查询
   - 测试资产设备实时查询
   - 测试主设备设置和保存

2. **前端测试**
   - 测试展开时并行查询
   - 测试折叠面板显示
   - 测试主设备设置流程
   - 测试错误处理（AD/资产查询失败）

## 执行顺序

1. ⏳ 后端服务层改造
2. ⏳ 后端API层改造
3. ⏳ 前端API客户端改造
4. ⏳ 前端组件改造
5. ⏳ 测试验证
6. ⏳ 提交代码
