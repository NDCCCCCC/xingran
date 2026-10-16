# VMP服务器API验证工具

## 功能说明

这个工具用于验证深信服VMP（Virtual Machine Management Platform）服务器的API功能，支持：
- **自动登录**获取认证Cookie
- **获取分组虚拟机列表**
- **实时运行状态监控**

## 认证流程

VMP使用**两步认证**：
```
┌──────────────┐
│ 1. 登录获取Cookie │
│ POST /ticket │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 2. 调用业务API  │
│ GET /cluster/vms │
│ 携带Cookie      │
└──────────────┘
```

## 快速开始

### 方式1：自动登录（推荐）

```bash
cd scripts/vmp-api

# 设置登录凭据
set VMP_USERNAME=admin
set VMP_PASSWORD=sangfor@2020
set VMP_ENDPOINT=https://10.62.0.72

# 运行脚本（自动登录+API调用）
python vmp_login.py
```

### 方式2：使用现有Cookie

```bash
# 如果Cookie未过期，直接使用
set VMP_AUTH_COOKIE=Login:YWRtaW4=:6A1E337A::Av42iXUmJJd3maHIRAmVl9SnWSnJCExW3LPvb9s0vsBJ5Rr8ywu9jTbsvU+Zy2dA9P3AX56I9fjoKkdVa5yQ6wTimVDnHM2WTr8F+z4iHqpmD0QLS+kbrqFavvq/Q3LCwm4OLgSsE0X7SChgsImQr4x/V83CoBnH8iSyalLwYeuDhEv61N9f9fFBoJQYOU8hQOTYQso55vymRwugXcRVzKEv6yYOk+ARiMj2yl4Eh46jj2uFtacCk3xuZWkdLnco1xPchVUPNdF/KBPKd1LsYC6mPI01nhir+CmiK6Px9bCsZna95z5y215VOfZzsPTU2Mn3R9KpWjRdtrgWyYP6DQ==

python vmp_login.py
```

## 前置要求

```bash
pip install requests
```

## 文件说明

| 文件 | 说明 |
|------|------|
| `vmp_login.py` | **主脚本** - 支持自动登录和API调用 |
| `verify_vmp_api.py` | 简化版 - 仅API调用（需手动设置Cookie） |
| `LOGIN_ANALYSIS.md` | 登录逻辑详细分析 |
| `.env.example` | 配置示例 |
| `run.bat` | Windows快速启动脚本 |

## API端点详解

### 1. 登录端点
```
POST /vapi/extjs/access/ticket

Content-Type: application/x-www-form-urlencoded

请求体:
- username: admin
- password: <加密后的密码>
- privacy: 1

响应:
Set-Cookie: LoginAuthCookie=<新cookie>; path=/; secure; httponly;
```

### 2. 虚拟机列表端点
```
GET /vapi/extjs/cluster/vms?group_type=group&sort_type=&desc=1

Headers:
- Cookie: LoginAuthCookie=<认证Cookie>
- X-Requested-With: XMLHttpRequest

响应:
{
    "success": 1,
    "data": [
        {
            "name": "分组名称",
            "id": "分组ID",
            "data": [虚拟机列表]
        }
    ]
}
```

## 密码加密说明

### 明文密码
```
sangfor@2020
```

### 加密特征
- **密文长度**: 512字符（十六进制）
- **推测算法**: RSA-2048 或 SM2国密加密
- **加密方式**: 公钥加密，私钥在服务器端

**注意**: 当前脚本使用简化版本（直接传递明文），生产环境需实现完整加密逻辑。

## 预期输出

```
============================================================
VMP Server Login & API Verification Tool
============================================================
Server: https://10.62.0.72
Username: admin
============================================================

[STEP 1] Login to VMP Server
------------------------------------------------------------
[LOGIN] Logging in to https://10.62.0.72/vapi/extjs/access/ticket
[LOGIN] Success! Got new Cookie (length: 512)

[STEP 2] Fetch VM List
------------------------------------------------------------
[API] Fetching VM list from https://10.62.0.72/vapi/extjs/cluster/vms

============================================================
[SUMMARY] API Response Summary
============================================================
Success: 1
Groups count: 11
Total VMs: 42
Running: 15
Stopped: 27
============================================================

[GROUPS] Virtual Machine Groups
============================================================
1. 模板 (ID: 00512bba38ff)
   Total: 6 | Running: 0 | Stopped: 6
2. 研发 (ID: d64c800964fd)
   Total: 9 | Running: 3 | Stopped: 6
...

============================================================
[SUCCESS] Verification completed!
============================================================
```

## Cookie有效期

根据响应头推测：
- `logoutTime=1440`: 可能表示24小时（1440分钟）
- `LastOpTime`: 最后操作时间戳
- **建议Cookie有效期: 24小时**

## 获取Cookie的方法

### 方法1：浏览器开发者工具
1. 登录VMP管理界面: https://10.62.0.72
2. 打开开发者工具（F12）
3. 进入Network标签
4. 刷新页面，找到任意请求
5. 复制Cookie中的 `LoginAuthCookie=xxx` 值

### 方法2：使用脚本自动登录
```bash
# 脚本会自动登录并获取新Cookie
python vmp_login.py
```

## 故障排查

### 登录失败
- 检查用户名密码是否正确
- 确认网络连接正常
- 检查账户是否被锁定

### SSL/TLS握手失败
脚本已配置跳过SSL验证，但如果仍有问题：
- 更新requests库: `pip install -U requests urllib3`
- 检查Python版本（建议3.6+）

### Cookie过期
Cookie有效期约24小时，过期后需要重新登录。

### API调用失败
- 确认Cookie是否有效
- 检查网络连接
- 验证服务器状态

## 安全提醒

**重要安全注意事项**：
1. 不要在生产环境硬编码密码
2. 定期更换登录密码
3. 使用环境变量传递凭据
4. Cookie会过期，需要定期更新

## 集成到项目

如需在项目中集成VMP API，参考：
- **位置**: `internal/services/vdi/`
- **Phase**: Phase 22A - VDI基础集成
- **模型**: `internal/models/vdi.go`

## 相关文档

- `LOGIN_ANALYSIS.md` - 详细的登录逻辑分析
- Phase 22规划文档: `.planning/milestones/v1.12-phases/22-sangfor-vdi-integration/`
