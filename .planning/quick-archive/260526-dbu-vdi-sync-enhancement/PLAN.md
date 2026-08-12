# VDI 同步功能完善

## 目标
完善 VDI 虚拟机同步功能，提升用户体验和可维护性。

## 实施步骤

### 1. 添加同步历史记录表
**文件**: `internal/models/vdi_sync_log.go`（新建）
- 创建 `VDISyncLog` 模型
- 字段包括：ID, ServerID, ServerName, StartTime, EndTime, Status, SuccessCount, FailCount, ErrorMessage
- 添加索引：server_id, status, start_time

### 2. 完善同步结果统计
**文件**: `internal/services/vdi/vm_service_impl.go`
- 修改 `SyncAllVMs` 方法，返回详细的同步结果
- 返回成功数量、失败数量、总数量等统计信息
- 记录同步日志到 `VDISyncLog` 表

**文件**: `internal/api/v1/vdi/vm_handler.go`
- 修改 `SyncAll` 方法，返回详细的同步结果
- 包括：total, success, failed, duration 等信息

### 3. 前端添加同步按钮
**文件**: `xingran-react-frontend/src/pages/vdi/VirtualMachine/index.tsx`
- 在工具栏添加"同步"按钮
- 添加同步状态显示
- 显示同步结果（成功/失败数量）
- 添加加载状态和进度提示

**文件**: `xingran-react-frontend/src/lib/vdiApi.ts`
- 添加 `syncAll()` 方法调用同步 API

## 验证标准
- [ ] 同步历史记录表已创建
- [ ] 同步 API 返回详细的统计信息
- [ ] 前端同步按钮可正常工作
- [ ] 同步历史可查询
- [ ] 代码编译通过

## 技术细节

### 同步结果格式
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 150,
    "success": 145,
    "failed": 5,
    "duration": "2.5s",
    "servers": [
      {
        "server_id": "uuid",
        "server_name": "生产环境",
        "success": 100,
        "failed": 0
      }
    ]
  }
}
```

### 前端UI设计
- 同步按钮图标：`SyncOutlined`
- 加载状态：显示 `loading` 属性
- 结果提示：使用 `message.success()` 或 `message.error()`
