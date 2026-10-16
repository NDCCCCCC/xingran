# VDI API集成快速参考指南

## 深信服VDI API端点映射表

### 认证相关

| 功能 | HTTP方法 | 端点 | 实现位置 | 状态 |
|------|----------|------|----------|------|
| 获取Token | POST | /v1/auth/tokens | `vdi_auth.go:Authenticate()` | ✅ |

### 虚拟机管理

| 功能 | HTTP方法 | 端点 | 实现位置 | 状态 |
|------|----------|------|----------|------|
| 创建虚拟机 | POST | /v1/vm | `vm_service_impl.go:CreateVM()` | ✅ |
| 删除虚拟机 | DELETE | /v1/vm | `vm_service_impl.go:DeleteVM()` | ✅ |
| 操作虚拟机 | POST | /v1/vm/operate | `vm_service_impl.go:OperateVM()` | ✅ |
| 配置IP | POST | /v1/vm/config_ip | `vm_service_impl.go:BatchConfigIP()` | ✅ |
| 重命名 | PUT | /v1/vm/:id/name | `vm_service_impl.go:RenameVM()` | ✅ |
| 绑定用户 | POST | /v1/vm/:id/bind_user | `vm_service_impl.go:BindUser()` | ✅ |
| 获取详情 | GET | /v1/vm/:id | `vdi_client_impl.go:GetVM()` | ✅ |
| 列出虚拟机 | GET | /v1/resource/:id/vms | `vdi_client_impl.go:ListVMs()` | ✅ |
| 获取可关联用户 | GET | /v1/vm/:id/available_users | `vdi_client_impl.go:GetAvailableUsers()` | ✅ |

## 后端实现层次结构

```
前端 (React)
  ↓ vdiApi.ts
后端API层 (Handler)
  ↓ vmHandler.Create/Operate/ConfigIP/Rename/BindUser
服务层 (Service)
  ↓ vmService.CreateVM/OperateVM/BatchConfigIP/RenameVM/BindUser
VDI客户端层 (Client)
  ↓ vdiClient.CreateVM/OperateVM/ConfigIP/RenameVM/BindUser
深信服VDI服务器
```

## 密码处理流程

```
1. 用户输入密码 → 前端表单
2. 前端发送 → 后端API
3. 后端加密 → operations.EncryptVDIPassword()
4. 存储到数据库 → sys_vdi_server.password_encrypted
5. 使用时解密 → operations.DecryptVDIPassword()
6. 调用VDI API → 使用解密后的密码
```

## Token管理流程

```
1. 首次认证 → 调用POST /v1/auth/tokens
2. 缓存Token → 存储到sys_vdi_server.auth_token
3. 设置过期时间 → sys_vdi_server.token_expiry (23小时后)
4. 后续请求 → 检查Token是否过期
5. Token过期 → 自动刷新
6. API调用 → 携带Auth-Token头
```

## 关键文件位置

### 后端文件

| 文件 | 位置 | 功能 |
|------|------|------|
| VDI数据模型 | `internal/models/vdi.go` | 数据模型定义、密码加密函数 |
| VDI客户端 | `internal/services/vdi/vdi_client_impl.go` | VDI API调用实现 |
| 认证管理 | `internal/services/vdi/vdi_auth.go` | Token管理、密码解密 |
| 虚拟机服务 | `internal/services/vdi/vm_service_impl.go` | 业务逻辑、VDI API调用 |
| API处理器 | `internal/api/v1/vdi/vm_handler.go` | HTTP请求处理 |
| 路由配置 | `internal/api/v1/vdi/vm_router.go` | 路由注册 |

### 前端文件

| 文件 | 位置 | 功能 |
|------|------|------|
| VDI类型 | `src/types/vdi.ts` | TypeScript类型定义 |
| VDI API客户端 | `src/lib/vdiApi.ts` | API调用封装 |
| 虚拟机列表 | `src/pages/vdi/VirtualMachineList/index.tsx` | 虚拟机管理页面 |
| VDI服务器配置 | `src/pages/vdi/VDIServerConfig/index.tsx` | 服务器配置页面 |

## VDI API调用示例

### 创建虚拟机

```typescript
// 前端调用
const result = await vmApi.create({
  name: '测试虚拟机',
  resource_id: 'res-001',
  vdi_server_id: 'server-001',
  cpu: 2,
  memory: 4096,
  disk: 60,
});

// 后端处理流程
// 1. vmHandler.Create() 接收请求
// 2. vmService.CreateVM() 验证并创建
// 3. vdiClient.CreateVM() 调用VDI API
// 4. POST /v1/vm 创建虚拟机
// 5. 返回VDI生成的vm_id
// 6. 保存到本地数据库
```

### 操作虚拟机（开关机）

```typescript
// 前端调用
const result = await vmApi.operate({
  vm_ids: ['vm-001', 'vm-002'],
  action: 'start', // 或 'stop', 'restart', 'suspend'
});

// 后端处理流程
// 1. vmHandler.Operate() 接收请求
// 2. vmService.OperateVM() 查询本地vm_id
// 3. vdiClient.OperateVM() 调用VDI API
// 4. POST /v1/vm/operate 批量操作
// 5. 返回操作结果
```

### 配置IP地址

```typescript
// 前端调用
const result = await vmApi.configIP([{
  vm_id: 'vm-001',
  ip_address: '192.168.1.100',
  netmask: '255.255.255.0',
  gateway: '192.168.1.1',
}]);

// 后端处理流程
// 1. vmHandler.ConfigIP() 接收请求
// 2. vmService.BatchConfigIP() 验证虚拟机
// 3. vdiClient.ConfigIP() 调用VDI API
// 4. POST /v1/vm/config_ip 配置IP
// 5. 更新本地数据库记录
```

## 错误处理模式

### VDI API错误处理

```go
// 1. VDI客户端层
if resp.ErrorCode != 0 {
    return &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
}

// 2. 服务层
vdiVM, err := s.vdiClient.CreateVM(ctx, req)
if err != nil {
    return nil, fmt.Errorf("failed to create VM in VDI: %w", err)
}

// 3. Handler层
if !handleServiceError(c, err, "创建") {
    return
}
```

### 前端错误处理

```typescript
try {
    await vmApi.operate({ vm_ids: ['vm-001'], action: 'start' });
    message.success('开机操作已提交，VDI API调用成功');
} catch (error) {
    message.error('操作失败，VDI API调用失败');
}
```

## 常见问题排查

### 1. Token过期问题
**症状**: API调用返回401或403
**解决**: 
- 检查sys_vdi_server.token_expiry是否为空
- 检查系统时间是否正确
- 手动触发Token刷新

### 2. 密码解密失败
**症状**: 认证失败，提示密码错误
**解决**:
- 检查password_encrypted字段是否为空
- 确认密码已正确加密
- 检查encryptVDIPassword/decryptVDIPassword函数

### 3. VDI API调用超时
**症状**: 操作无响应或超时
**解决**:
- 检查网络连接
- 增加超时时间配置
- 检查VDI服务器状态

### 4. 虚拟机状态不同步
**症状**: 前端显示状态与VDI服务器不一致
**解决**:
- 点击"同步"按钮手动同步
- 检查last_sync_at字段
- 确认SyncVMFromVDI方法正常工作

## 性能优化建议

### 1. 批量操作优化
- 将批量操作改为异步执行
- 使用消息队列处理大批量操作
- 提供操作进度查询

### 2. 缓存优化
- 虚拟机列表缓存5分钟
- Token缓存23小时
- 资源组缓存10分钟

### 3. 网络优化
- 使用HTTP连接池
- 启用请求重试机制
- 设置合理的超时时间

## 安全检查清单

- [ ] VDI服务器密码已加密存储
- [ ] Token不在日志中打印
- [ ] API通信使用HTTPS
- [ ] 参数验证完整
- [ ] 权限检查正确
- [ ] 错误信息脱敏处理
- [ ] 密码定期轮换机制

## 测试检查清单

- [ ] 单元测试覆盖率>80%
- [ ] VDI API客户端测试
- [ ] 服务层集成测试
- [ ] 前端组件测试
- [ ] 端到端测试
- [ ] 性能测试
- [ ] 安全测试

## 部署检查清单

- [ ] 数据库迁移执行成功
- [ ] VDI配置文件正确
- [ ] 环境变量设置正确
- [ ] API路由注册成功
- [ ] 前端路由配置正确
- [ ] Swagger文档可访问
- [ ] 日志配置正确

## 监控指标建议

### VDI API调用监控
- API调用成功率
- API响应时间
- Token刷新频率
- 错误类型分布

### 虚拟机操作监控
- 创建/删除操作数量
- 开关机操作数量
- 操作成功率
- 操作失败原因

### 系统性能监控
- CPU使用率
- 内存使用率
- 网络连接数
- 缓存命中率
