# API响应规范

## 概述

本文档定义了XingRan Next项目中API响应的标准格式、状态码规范以及前端调用最佳实践。所有API接口都应遵循此规范，确保前后端数据交互的一致性和可靠性。

## 响应格式总览

### 基础响应结构

所有API接口的响应都应遵循以下统一格式：

```json
{
    "code": 0,
    "message": "success",
    "data": {},
    "timestamp": 1766380800,
    "request_id": "1234567890"
}
```

### 字段说明

| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| code | number | 是 | 业务状态码，0表示成功，其他值表示错误 |
| message | string | 是 | 响应消息，用于描述操作结果 |
| data | any | 否 | 业务数据，成功时包含具体数据，错误时可为null |
| timestamp | number | 是 | Unix时间戳，响应生成时间 |
| request_id | string | 是 | 唯一请求标识符，用于日志追踪 |

## 成功响应标准

### 状态码
- 成功响应的业务状态码必须为 **`0`**

### 成功响应示例

#### 单条数据响应
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": "123e4567-e89b-12d3-a456-426614174000",
        "username": "admin",
        "email": "admin@example.com",
        "status": 1,
        "created_at": "2024-01-01T12:00:00Z"
    },
    "timestamp": 1766380800,
    "request_id": "req_1234567890"
}
```

#### 列表数据响应
```json
{
    "code": 0,
    "message": "success",
    "data": [
        {
            "id": "123e4567-e89b-12d3-a456-426614174000",
            "username": "admin",
            "status": 1
        },
        {
            "id": "123e4567-e89b-12d3-a456-426614174001",
            "username": "user1",
            "status": 1
        }
    ],
    "timestamp": 1766380800,
    "request_id": "req_1234567890"
}
```

#### 分页数据响应
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "list": [
            {
                "id": "123e4567-e89b-12d3-a456-426614174000",
                "username": "admin",
                "status": 1
            }
        ],
        "total": 100,
        "current": 1,
        "pageSize": 10
    },
    "timestamp": 1766380800,
    "request_id": "req_1234567890"
}
```

## 错误响应规范

### 错误状态码分类

#### HTTP标准错误码
- `400` - 请求参数错误
- `401` - 未授权访问
- `403` - 权限不足
- `404` - 资源不存在
- `405` - 请求方法不允许
- `500` - 服务器内部错误
- `501` - 服务未实现

#### 业务错误码
- `1001` - 参数错误
- `1002` - 用户不存在
- `1003` - 密码错误
- `1004` - 用户已禁用
- `1005` - 验证码错误
- `1006` - 令牌已过期
- `1007` - 令牌无效
- `1008` - 令牌尚未生效
- `1009` - 数据已存在
- `1010` - 数据不存在
- `1011` - 密码哈希格式不支持
- `1012` - 密码解析失败
- `1013` - 用户名或密码错误
- `1014` - 数据库操作失败

### 错误响应示例

#### 参数验证错误
```json
{
    "code": 1001,
    "message": "参数错误",
    "data": null,
    "timestamp": 1766380800,
    "request_id": "req_1234567890"
}
```

#### 业务逻辑错误
```json
{
    "code": 1002,
    "message": "用户名或密码错误",
    "data": null,
    "timestamp": 1766380800,
    "request_id": "req_1234567890"
}
```

#### 系统错误
```json
{
    "code": 500,
    "message": "服务器内部错误",
    "data": null,
    "timestamp": 1766380800,
    "request_id": "req_1234567890"
}
```

## 特殊场景响应

### 文件上传响应
```json
{
    "code": 0,
    "message": "上传成功",
    "data": {
        "file_id": "file_1234567890",
        "file_name": "document.pdf",
        "file_size": 1024000,
        "file_type": "application/pdf",
        "file_url": "/uploads/2024/01/01/document.pdf"
    },
    "timestamp": 1766380800,
    "request_id": "req_1234567890"
}
```

### 批量操作响应
```json
{
    "code": 0,
    "message": "批量操作完成",
    "data": {
        "success_count": 8,
        "failed_count": 2,
        "failed_items": [
            {
                "index": 3,
                "item": {"id": "item3"},
                "error": "数据不存在"
            },
            {
                "index": 7,
                "item": {"id": "item7"},
                "error": "权限不足"
            }
        ]
    },
    "timestamp": 1766380800,
    "request_id": "req_1234567890"
}
```

### 异步任务响应
```json
{
    "code": 0,
    "message": "任务提交成功",
    "data": {
        "task_id": "task_1234567890",
        "task_name": "数据导出",
        "task_status": "pending",
        "estimated_duration": 30
    },
    "timestamp": 1766380800,
    "request_id": "req_1234567890"
}
```

### 网络设备端口写操作

v1.19 网络设备写命令（Network Device Port Write Operations）通过 6 个 POST 端点下发，前缀 `/network/ports/write/`，要求 `network:port:write` 权限 + SM2+SM4 请求体加密（D-04 保持加密）。

| 路径 | 方法 | 操作 | OperType 常量 |
|------|------|------|---------------|
| `/network/ports/write/shutdown` | POST | 关闭端口 | `OperTypeStatus (10)` |
| `/network/ports/write/undo-shutdown` | POST | 取消关闭 | `OperTypeStatus (10)` |
| `/network/ports/write/description` | POST | 修改端口描述 | `OperTypeUpdate (2)` |
| `/network/ports/write/dot1x-enable` | POST | 启用 802.1X | `OperTypeStatus (10)` |
| `/network/ports/write/dot1x-disable` | POST | 停用 802.1X | `OperTypeStatus (10)` |
| `/network/ports/write/batch` | POST | 批量写 | `OperTypeBatch (16)` |

请求体（单端口）：

```go
type PortWriteRequest struct {
    PortID      string `json:"portId" binding:"required"`      // 必填，UUID 字符串
    Description string `json:"description,omitempty"`           // 可选，≤80 字符，仅 description action 使用
    Reason      string `json:"reason,omitempty"`                // 可选，UI-02 操作原因（审计字段）
}
```

响应（单端口）：

```go
type PortResult struct {
    PortID       string `json:"portId"`
    Action       string `json:"action"`                         // shutdown | undo_shutdown | description | dot1x_enable | dot1x_disable
    Status       string `json:"status"`                         // succeeded | failed | skipped
    NoOp         bool   `json:"noOp"`                           // pre-state 已匹配目标态（PORT-06）
    CurrentState string `json:"currentState,omitempty"`         // admin_up / admin_down / dot1x_enabled / dot1x_disabled
    Error        string `json:"error,omitempty"`                // 失败时填入
    CommandSent  string `json:"commandSent,omitempty"`          // 审计真相源，下发命令原文，不脱敏
}
```

注意：`commandSent` 字段在 operlog 中被 `OperlogRecord` 自动脱敏（password / secret 等关键词），但 HTTP 响应体的 `commandSent` 是**未脱敏**真相源 — 调用方可信赖。`sys_port_write_audit` 表保存完整命令原文以供合规审计。

批量端点请求：

```go
type BatchWriteRequest struct {
    DeviceID    string   `json:"deviceId"`                      // 必填，UUID
    Action      string   `json:"action"`                        // 同单端口 action 枚举
    PortIDs     []string `json:"portIds"`                       // 必填，1-50 个
    Description string   `json:"description,omitempty"`         // 仅 description action 使用
}
```

批量响应（三数组同时存在，即便为空也输出 `[]` 而非 `null`）：

```go
type BatchResult struct {
    Succeeded []PortResult `json:"succeeded"`                   // 完全成功的端口
    Failed    []PortResult `json:"failed"`                      // 任一错误类型失败的端口
    Skipped   []PortResult `json:"skipped"`                     // pre-state NoOp 跳过的端口
}
```

批量语义与现有"批量操作响应"（`success_count`/`failed_count` 结构）**不同**——本端点采用"三数组"分类，明确区分 succeeded / failed / skipped 三种终态。`failed` 中端口的 `error` 字段区分 SSH transport 错误（连接失败/超时/EOF）与设备命令拒绝（语法错/参数非法），前者触发 fail-fast 立即停止剩余端口（BATCH-02），后者亦 fail-fast；任一 fail 后剩余端口**不进**任何数组。

fail-fast 行为：

- 任一端口 SSH transport 错误或设备命令拒绝 → 立即停止后续端口
- 剩余端口**不**出现在 `succeeded`/`failed`/`skipped` 任一数组中
- 前端响应需提示"已执行 X/Y 个端口"，剩余 Y-X 端口需用户主动发起二次请求

成功路径自动触发 `DeviceInfoCollectionService.Enqueue` 后台采集（fire-and-forget），1-2s 内回填端口最新状态至 `sys_device_port_status` 表。失败路径**不**触发采集（避免 Enqueue 风暴 + 无意义的后台轮询，参见 Pitfall #6）。

### v1.20.1 扩展：set_access_vlan + port_binding

v1.20.1 在 v1.19 的 6 个端口写端点基础上新增 2 个端点，沿用 `network:port:write` 权限 + SM2+SM4 请求体加密 + `sys_port_write_audit` JSONB 审计真相源。请求体不再用通用 `PortWriteRequest`，而是各自的专用 struct（PATTERNS.md Option A：handler 显式绑定 action-specific request，避免通用 struct 字段越权）。

**端点（2 个新增）：**

| 路径 | 方法 | 操作 | OperType 常量 |
|------|------|------|---------------|
| `/network/ports/write/set-access-vlan` | POST | 修改端口 access VLAN（PVID） | `OperTypeUpdate (2)` |
| `/network/ports/write/port-binding` | POST | 端口绑定 add/remove（IP + 可选 MAC） | `op=add` → `OperTypeCreate (1)`；`op=remove` → `OperTypeDelete (3)` |

**SetAccessVlanRequest 请求体：**

```go
type SetAccessVlanRequest struct {
    PortID string `json:"portId" binding:"required"`                    // 必填，目标端口 UUID
    VLANID int    `json:"vlanId" binding:"required,min=1,max=4094"`     // 必填，VLAN 1-4094（0/4095 保留，binding 双重防线；service 仍校验为真相源）
    Reason string `json:"reason,omitempty"`                             // 可选，UI-02 操作原因
}
```

**PortBindingRequest 请求体：**

```go
type PortBindingRequest struct {
    PortID     string `json:"portId" binding:"required"`                // 必填，目标端口 UUID
    Op         string `json:"op" binding:"required,oneof=add remove"`   // 必填，add 或 remove
    IPAddress  string `json:"ipAddress" binding:"required"`             // 必填，IPv4（service 严格 regex 校验）
    MACAddress string `json:"macAddress,omitempty"`                     // 可选，MAC（仅 op=add 时有意义；空=IP-only binding）
    Reason     string `json:"reason,omitempty"`                         // 可选，UI-02 操作原因
}
```

**错误码（sentinel → HTTP 400）：**

| 错误 sentinel | HTTP code | 用户提示 |
|--------------|-----------|----------|
| `portwrite: vlanId out of range 1-4094` | 400 | VLAN ID 必须在 1-4094 之间 |
| `portwrite: bind op must be add or remove` | 400 | 绑定操作必须是 add 或 remove |
| `portwrite: invalid ipv4 address` | 400 | IP 地址格式不合法 |
| `portwrite: invalid mac address` | 400 | MAC 地址格式不合法 |

**请求示例（set_access_vlan）：**

```json
POST /network/ports/write/set-access-vlan
{
  "portId": "f1e2d3c4-b5a6-9876-5432-10fedcba9876",
  "vlanId": 100,
  "reason": "业务变更需要"
}
```

**请求示例（port_binding add with MAC）：**

```json
POST /network/ports/write/port-binding
{
  "portId": "f1e2d3c4-b5a6-9876-5432-10fedcba9876",
  "op": "add",
  "ipAddress": "10.62.25.5",
  "macAddress": "AA:BB:CC:DD:EE:FF",
  "reason": "安全合规要求"
}
```

**响应（与 v1.19 端点一致，复用 `PortResult` schema，新增 `extra` 字段携带 audit 上下文）：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "portId": "f1e2d3c4-b5a6-9876-5432-10fedcba9876",
    "deviceId": "a1b2c3d4-...",
    "action": "set_access_vlan",
    "status": "succeeded",
    "noOp": false,
    "commandSent": "interface GE0/0/1 | port link-type access | port default vlan 100 | quit",
    "extra": {
      "vlanId": 100
    }
  },
  "timestamp": 1766380800,
  "request_id": "uuid-string"
}
```

`extra` 字段（v1.20.1 新增）携带 audit 上下文供 `sys_port_write_audit.after_value` JSON 写入（INFRA-01）：

- `set_access_vlan`：`{"vlanId": <int>}`
- `port_binding`：`{"bindOp": "add"|"remove", "ipAddress": "<ipv4>", "macAddress": "<original-input>"}`（`macAddress` 仅在非空时出现）

**审计**：复用 `sys_port_write_audit` JSONB 字段；`commandSent` 不脱敏（审计真相源，与 v1.19 一致）。OperType 按 op 分流（add=Create / remove=Delete / set_access_vlan=Update），便于 `sys_oper_log` 按业务类型过滤。

**厂商命令模板（3 厂商 × 2 actions × variants）：**

| 厂商 | set_access_vlan | port_binding add (IP-only) | port_binding add (with MAC) | port_binding remove |
|------|----------------|----------------------------|----------------------------|---------------------|
| Huawei VRP | `port link-type access` + `port default vlan <N>` | `user-bind static ip-address <IP>` | `user-bind static ip-address <IP> mac-address <AA-BB-CC-DD-EE-FF>` | `undo user-bind static ip-address <IP> [mac-address ...]` |
| H3C Comware | `port link-type access` + `port access vlan <N>` | `user-bind ip-address <IP>` | `user-bind ip-address <IP> mac-address <AA-BB-CC-DD-EE-FF>` | `undo user-bind ip-address <IP> [mac-address ...]` |
| Ruijie RGOS | `switchport mode access` + `switchport access vlan <N>` | `switchport port-security binding <IP>` | `switchport port-security binding <aabb.ccdd.eeff> <IP>` | `no switchport port-security binding [<mac>] <IP>` |

**MAC 格式**（per vendor CLI 规范）：Huawei/H3C = `AA-BB-CC-DD-EE-FF`（per-byte hyphenated）；Ruijie = `aabb.ccdd.eeff`（Cisco H.H.H 3-pair dotted lowercase）。service 层 `NormalizeMACAddress` 接受冒号/连字符/无分隔/cisco 点多种输入格式，归一后按厂商转换。

## 前端调用规范

### API调用函数使用

项目中已封装统一的API调用函数，应优先使用：

```typescript
import { post, get, put, del } from '@/lib/api';

// ✅ 正确写法 - 使用封装函数
const result = await post('/system/users/list', {
    current: 1,
    pageSize: 10
});
// 直接使用 result.data，无需手动检查 code

// ❌ 错误写法 - 直接使用 api 实例
const response = await api.post('/system/users/list', {
    current: 1,
    pageSize: 10
});
if (response.data.code === 0) {
    // 需要手动检查 code
}
```

### 响应数据处理

```typescript
// 获取数据
const result = await post('/system/users/list', params);
const users = result.data.list;  // 直接使用数据

// 错误处理通过响应拦截器统一处理
try {
    const result = await post('/system/users/create', userData);
    // 成功处理逻辑
} catch (error) {
    // 错误已在拦截器中处理，这里可以添加额外逻辑
    console.error('操作失败:', error);
}
```

## 分页参数标准

### 请求参数
```typescript
interface PageParams {
    current: number;    // 当前页码，从1开始
    pageSize: number;   // 每页大小
}
```

### 分页响应数据
```typescript
interface PageData<T> {
    list: T[];         // 数据列表
    total: number;      // 总记录数
    current: number;    // 当前页码
    pageSize: number;   // 每页大小
}
```

## 时间戳规范

### 后端时间戳
- 使用Unix时间戳（秒级）
- 类型：`number`
- 示例：`1766380800`

### 前端时间处理
```typescript
// Unix时间戳转本地时间
const timestamp = 1766380800;
const date = new Date(timestamp * 1000);
const localString = date.toLocaleString();

// 本地时间转Unix时间戳
const now = new Date();
const timestamp = Math.floor(now.getTime() / 1000);
```

## 字段命名规范

本项目 JSON 序列化统一使用 **camelCase**（后端 Go struct 通过 `json:"xxxYyy"` tag 显式声明）：

```go
// 后端 Go struct 定义示例
type User struct {
    UserID    string `json:"userId"`
    CreatedAt int64  `json:"createdAt"`
    UserName  string `json:"userName"`
}

// 前端 TypeScript 接口（无需转换）
interface User {
    userId: string;
    createdAt: number;
    userName: string;
}
```

**唯一例外**：响应体顶层的 `request_id` 沿用 snake_case（出于日志聚合与历史兼容性，跨服务追踪统一约定）。

> 数据库列名仍按 `snake_case`（如 `user_id`、`created_at`、`sys_user.user_name`），但仅在 GORM model → 数据库层使用，**不暴露到 JSON API**。前端不需做 `request_id → requestId` 这类转换。

## 版本兼容性

### API版本控制
- 通过请求头指定API版本：`API-Version: v1`
- 向后兼容：新版本保持对旧版本接口的兼容
- 版本弃用：提前通知版本弃用计划

### 响应格式变更
- 新增字段：向后兼容，不影响现有功能
- 字段删除：保持字段存在，值可为null
- 字段类型变更：保持字符串类型，必要时进行格式转换

## 最佳实践

### 响应设计原则
1. **一致性**：所有接口遵循统一响应格式
2. **简洁性**：响应消息简洁明了，避免冗余信息
3. **完整性**：包含必要的调试信息（request_id、timestamp）
4. **安全性**：不返回敏感信息，如密码、密钥等

### 错误处理原则
1. **明确性**：错误消息明确指出具体问题
2. **可操作性**：提供解决建议或操作指导
3. **分类性**：合理分类错误类型，便于前端处理
4. **日志记录**：记录详细错误信息，便于问题排查

### 性能优化
1. **数据压缩**：对大数据量响应启用压缩
2. **分页查询**：避免一次性返回大量数据
3. **字段筛选**：支持按需返回指定字段
4. **缓存策略**：对不经常变化的数据启用缓存

## 示例代码

### 完整的前端API调用示例
```typescript
import { post } from '@/lib/api';
import { message } from 'antd';

// 获取用户列表
export const fetchUserList = async (params: PageParams) => {
  try {
    const result = await post('/system/users/list', params);
    return {
        list: result.data.list,
        total: result.data.total,
        current: result.data.current,
        pageSize: result.data.pageSize
    };
  } catch (error) {
    console.error('获取用户列表失败:', error);
    throw error;
  }
};

// 创建用户
export const createUser = async (userData: CreateUserData) => {
  try {
    const result = await post('/system/users/create', userData);
    message.success('创建成功');
    return result.data;
  } catch (error) {
    message.error('创建失败');
    throw error;
  }
};
```

## 总结

本规范定义了XingRan Next项目中API响应的统一标准，所有开发人员都应严格遵守。通过统一的响应格式、状态码规范和调用方式，可以显著提升开发效率和代码质量，减少前后端集成中的问题。

如有疑问或需要补充，请联系技术负责人。
