# API响应规范

## 概述

本文档定义了RuoYi Next项目中API响应的标准格式、状态码规范以及前端调用最佳实践。所有API接口都应遵循此规范，确保前后端数据交互的一致性和可靠性。

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
- `1002` - 用户名或密码错误
- `1003` - 账户被禁用
- `1004` - 权限不足
- `1005` - 资源不存在
- `1006` - 操作失败
- `1007` - 令牌无效
- `1008` - 令牌已过期
- `1009` - 数据已存在
- `1010` - 数据不存在
- `1011` - 操作被拒绝
- `1012` - 文件上传失败
- `1013` - 文件类型不支持
- `1014` - 文件大小超限

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

### 后端字段命名
- 使用下划线命名：`snake_case`
- 示例：`request_id`, `created_at`, `user_name`

### 前端字段命名
- 使用驼峰命名：`camelCase`
- 示例：`requestId`, `createdAt`, `userName`

### 数据转换
```typescript
// 后端响应
interface BackendResponse {
    request_id: string;
    created_at: string;
    user_name: string;
}

// 前端接口（使用camelCase）
interface FrontendData {
    requestId: string;
    createdAt: string;
    userName: string;
}
```

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

本规范定义了RuoYi Next项目中API响应的统一标准，所有开发人员都应严格遵守。通过统一的响应格式、状态码规范和调用方式，可以显著提升开发效率和代码质量，减少前后端集成中的问题。

如有疑问或需要补充，请联系技术负责人。