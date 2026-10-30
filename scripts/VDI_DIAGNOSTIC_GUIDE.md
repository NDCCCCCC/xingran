# VDI 系统诊断工具使用指南

## 概述

本目录包含用于诊断 VDI 系统集成问题的工具和脚本。

## 工具列表

### 1. vdi_diagnostic_tool.sh (Linux/Mac)

直接调用 VDI API 并显示原始数据。

**使用方法：**
```bash
# 设置环境变量
export VDI_PASSWORD=your_password

# 运行诊断脚本
chmod +x scripts/vdi_diagnostic_tool.sh
./scripts/vdi_diagnostic_tool.sh
```

**可选参数：**
```bash
# 自定义 VDI 服务器地址
export VDI_URL=https://your-vdi-server:6060

# 自定义用户名
export VDI_USERNAME=your_username
```

### 2. vdi_diagnostic_tool.bat (Windows)

Windows 版本的诊断工具。

**使用方法：**
```cmd
REM 设置环境变量
set VDI_PASSWORD=your_password

REM 运行诊断脚本
scripts\vdi_diagnostic_tool.bat
```

### 3. check_vdi_local_data.sql

检查本地数据库中的 VDI 数据。

**使用方法：**

如果是 PostgreSQL：
```bash
psql -h localhost -U xingran -d xingran_next -f scripts/check_vdi_local_data.sql
```

如果是通过数据库管理工具：
1. 打开您的数据库管理工具（如 pgAdmin、DBeaver 等）
2. 连接到 xingran_next 数据库
3. 执行 `scripts/check_vdi_local_data.sql` 中的查询

### 4. check_vdi_data.go

Go 语言编写的完整诊断工具。

**使用方法：**
```bash
# 设置密码
export VDI_PASSWORD=your_password

# 运行
cd scripts
go run check_vdi_data.go
```

## 诊断流程

### 第一步：检查 VDI 系统

运行诊断脚本确认 VDI 系统本身有数据：

```bash
./scripts/vdi_diagnostic_tool.sh
```

**预期输出：**
- ✅ 认证成功
- ✅ 找到 X 个资源组
- ✅ 资源组下有虚拟机数据

如果 VDI 系统返回空数据，问题在 VDI 系统配置，不是代码问题。

### 第二步：检查本地数据库

执行 SQL 查询检查本地数据库：

```sql
-- 检查是否有虚拟机数据
SELECT COUNT(*) FROM sys_vdi_virtual_machine WHERE deleted_at IS NULL;
```

**可能的结果：**

1. **计数为 0**：说明数据没有同步到本地数据库
   - 检查 `syncVMsFromVDI` 函数是否正确执行
   - 查看日志中是否有 "saveOrUpdateVM" 相关的日志

2. **计数 > 0**：说明数据已同步
   - 检查前端查询条件
   - 检查分页参数

### 第三步：检查 API 响应

查看后端日志中的响应数据：

```bash
# 查看最新的日志
tail -f logs/app.log | grep "VDI"
```

**关键日志：**
- `[VDI AUTH DEBUG]`: 认证相关
- `[VDI API DEBUG]`: API 调用相关
- 查询结果统计信息

## 常见问题诊断

### 问题 1: API 返回 200 但前端显示空列表

**可能原因：**
1. 本地数据库没有数据
2. 前端查询条件过滤掉了所有数据
3. 分页参数不正确

**诊断步骤：**
1. 执行 SQL 查询确认本地有数据
2. 检查前端网络请求的响应体
3. 查看后端日志中的查询条件

### 问题 2: AUTH_TOKEN_INVALID

**可能原因：**
1. VDI 系统用户名/密码不正确
2. Token 缓存过期

**诊断步骤：**
1. 使用诊断脚本验证凭据
2. 清除数据库中的 token 缓存：
   ```sql
   UPDATE sys_vdi_server SET auth_token = NULL, token_expiry = NULL;
   ```
3. 重启后端服务

### 问题 3: 资源组为空

**可能原因：**
1. VDI 系统中没有配置资源组
2. 资源组未启用

**诊断步骤：**
1. 运行诊断脚本查看 VDI 系统中的资源组
2. 检查资源组的 `enable` 字段是否为 "1"

## 日志位置

- **后端日志**: `logs/app.log` 或控制台输出
- **数据库**: PostgreSQL/SQLite 数据文件

## 联系支持

如果以上工具无法解决问题，请收集以下信息：
1. 诊断脚本的完整输出
2. 后端日志（包含 VDI 相关的日志）
3. 数据库查询结果
4. 前端网络请求详情
