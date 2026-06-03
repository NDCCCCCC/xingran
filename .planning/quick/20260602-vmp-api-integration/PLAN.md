# Quick Task: VMP服务器分组虚拟机API集成

## 目标
集成VMP服务器的分组虚拟机列表API，实现获取虚拟机分组信息和实时运行状态的功能。

## 背景
用户提供了一个VMP服务器API endpoint，返回按分组组织的虚拟机列表，包含详细的运行状态信息（CPU、内存、磁盘使用率等）。这个API用于虚拟机监控和状态同步。

## API分析

### 请求
- **URL**: `https://10.62.0.72/vapi/extjs/cluster/vms`
- **方法**: GET
- **查询参数**: `group_type=group&sort_type=&desc=1`
- **认证**: Cookie中的LoginAuthCookie

### 响应结构
```json
{
    "success": 1,
    "data": [
        {
            "name": "分组名称",
            "id": "分组ID",
            "data": [
                {
                    "vmid": 123456789,
                    "name": "VM名称",
                    "status": "running/stopped/clone",
                    "ip": "IP地址",
                    "hostname": "主机名",
                    "cores_number": CPU核心数,
                    "memory": 内存大小(MB),
                    "ostype": "操作系统类型,
                    "vmtype": "vm/derive/tpl",
                    "cpu_ratio": CPU使用率,
                    "mem_ratio": 内存使用率,
                    "io_ratio": IO使用率,
                    "disk_status": { "ratio": 磁盘使用率, "free": 剩余空间, "total": 总空间 },
                    "cpu_status": { "ratio": CPU使用率, "mhz": 频率, "cpus": 核心数 },
                    "mem_status": { "ratio": 内存使用率, "free": 剩余, "total": 总计 },
                    "groupname": "分组名称",
                    "vmgroup": "分组ID",
                    // ... 更多字段
                }
            ]
        }
    ]
}
```

## 实施计划

### 1. 创建数据模型 (internal/models/vdi.go)
添加以下结构体：
- `VMPVMGroup` - 虚拟机组
- `VMPVMInfo` - 虚拟机信息
- `VMPPasswordStatus` - 密码状态
- `VMPDiskStatus` - 磁盘状态
- `VMPCPUStatus` - CPU状态
- `VMPMemStatus` - 内存状态

### 2. 扩展VDI客户端 (internal/services/vdi/vdi_client_impl.go)
添加方法：
- `GetGroupedVMs(ctx context.Context, serverID string) ([]*VMPVMGroup, error)`

### 3. 创建服务层 (internal/services/vdi/vm_sync_service.go)
添加方法：
- `SyncGroupedVMs(ctx context.Context, serverID string) ([]*VMPVMGroup, error)`
- `GetVMStatus(ctx context.Context, serverID, vmID string) (*VMPVMInfo, error)`

### 4. 添加HTTP端点 (internal/api/v1/vdi/)
创建文件：
- `vm_sync_handler.go` - 处理同步请求
- `vm_sync_router.go` - 路由配置

端点：
- `POST /api/v1/vdi/sync/vms` - 同步虚拟机列表
- `GET /api/v1/vdi/groups/:server_id` - 获取分组列表

### 5. 更新现有代码
- 在 `vdi_client_impl.go` 中添加新方法
- 更新 `vm_service_impl.go` 以使用新数据

### 6. 可选：前端集成
如果需要，可以：
- 在虚拟机列表页面添加"刷新状态"按钮
- 显示实时CPU/内存/磁盘使用率
- 添加分组视图

## 依赖
- 现有的VDI客户端基础设施
- sys_vdi_server表中的认证信息
- 现有的虚拟机数据模型

## 验证
1. 能够成功调用API并获取分组数据
2. 数据正确解析为Go结构体
3. HTTP端点返回正确的JSON响应
4. 前端能够显示虚拟机状态

## 预期时间
30-45分钟
