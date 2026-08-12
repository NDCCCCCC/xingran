# VMP服务器登录逻辑分析

## 登录流程概述

VMP服务器使用**两步认证流程**：
1. **登录获取Cookie**: POST `/vapi/extjs/access/ticket`
2. **使用Cookie调用API**: GET `/vapi/extjs/cluster/vms`

## 登录API详解

### 端点
```
POST https://10.62.0.72/vapi/extjs/access/ticket
```

### 请求头
```
Content-Type: application/x-www-form-urlencoded; charset=UTF-8
Origin: https://10.62.0.72
Referer: https://10.62.0.72/login.pl
X-Requested-With: XMLHttpRequest
Cookie: LoginAuthCookie=<旧cookie> (如果有)
```

### 请求体（表单数据）
```
username=admin
password=<加密后的密码>
privacy=1
```

### 响应
**成功时** (HTTP 200):
```
Set-Cookie: LoginAuthCookie=<新cookie>; path=/; secure; httponly;
Content-Type: application/json;charset=UTF-8
```

**失败时**:
- HTTP 4xx 或返回错误信息

## 密码加密分析

### 明文密码
```
sangfor@2020
```

### 加密后密码（示例）
```
295141767757762b08c2002a7a65fe61baf7ee99aa7b907d23684f4803404a0efef8a33b9cd42e5b6a874694b603b6b2b86ef87a2a166dca169e5ae24a65f48ec959d209b7d04ee1c214502f0514009e91ac0c1f508dc1412ff3a1ca374599a9c2142c96f19c5312d02f10bc5b5ab0d5444db322dc45ec17ddb5f222b35d51f8d661e83bc872687b911c6ddbf17858183d90250865b79dd1ab26324287c8f788688a5e2c3047c557c4bfd68270e60807a09f641ca0f36fd1975f652d2ead3fa748c114dc8116fd22f073a28ce68fdb7660b6ec2c2bc782c2fa2776ec9db169c14704bf62cb7fbce52d15dd6a0474dc6ae8ca08a56055026ce1b6fae71b0b5d
```

### 加密特征分析
观察加密后的密码：
- **长度**: 512字符
- **格式**: 十六进制字符串
- **结构**: 可能是加密算法的输出

#### 可能的加密方式

1. **RSA加密**
   - 密文长度512字符 ≈ 2048位 = RSA-2048
   - 使用公钥加密，私钥在服务器端解密

2. **混合加密**
   - AES-256加密密码 (32字节)
   - RSA-2048加密AES密钥
   - Base64编码拼接

3. **SM2国密加密** (深信服可能使用)
   - 国密SM2椭圆曲线公钥加密
   - 密文长度512字符符合SM2特征

### 前端加密逻辑（推测）

```javascript
// 伪代码，实际逻辑需要分析前端JS
function encryptPassword(plaintext, publicKey) {
    // 1. 从服务器获取公钥
    // 2. 使用RSA/SM2公钥加密明文密码
    // 3. 返回Base64编码的密文
    return encrypt(plaintext, publicKey);
}
```

## 完整认证流程

```
┌─────────────┐
│  用户输入    │
│  明文密码    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  前端JS加密  │
│  RSA/SM2    │
└──────┬──────┘
       │
       ▼ 密文(512字符)
┌─────────────┐
│ POST /ticket │
│  username   │──→ 服务器验证
│  password   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 新Cookie     │
│ LoginAuth    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 后续API调用  │
│ 携带Cookie   │
└─────────────┘
```

## 使用方式

### 方式1：使用现有Cookie（临时）
如果Cookie未过期，可以直接使用：
```bash
VMP_AUTH_COOKIE="<你的Cookie>" python vmp_login.py
```

### 方式2：自动登录（推荐）
```bash
# 设置登录凭据
VMP_USERNAME=admin
VMP_PASSWORD=sangfor@2020
VMP_ENDPOINT=https://10.62.0.72

# 运行脚本（会自动登录）
python vmp_login.py
```

## 安全注意事项

### 重要提醒
1. **不要在生产环境硬编码密码**
2. **定期更换登录密码**
3. **使用环境变量传递凭据**
4. **Cookie会过期，需要重新登录**

### Cookie有效期
从Cookie字段推测：
- `LastOpTime`: 最后操作时间戳
- `logoutTime=1440`: 可能表示24小时（1440分钟）
- **Cookie有效期可能为24小时**

## 实现集成时的注意事项

### 1. 密码加密
如果需要在项目中实现自动登录：
- 需要实现前端加密算法
- 或使用API Key代替密码
- 考虑使用OAuth等更安全的认证方式

### 2. Token管理
```go
// 建议的Token管理结构
type VMPAuth struct {
    ServerID   string
    AuthCookie string
    ExpiresAt  time.Time
    LastUsed   time.Time
}
```

### 3. 自动刷新
```go
// 检查Token是否即将过期
func (v *VMPAuth) ShouldRefresh() bool {
    return time.Until(v.ExpiresAt) < 30*time.Minute
}
```

## 相关API端点

| 端点 | 方法 | 用途 |
|------|------|------|
| `/vapi/extjs/access/ticket` | POST | 登录获取Cookie |
| `/vapi/extjs/cluster/vms` | GET | 获取分组虚拟机列表 |
| `/vapi/extjs/cluster/vms` | GET | 获取虚拟机详情 |

## 测试命令

```bash
# 快速测试登录和API调用
cd scripts/vmp-api

# 设置凭据
export VMP_USERNAME="admin"
export VMP_PASSWORD="sangfor@2020"
export VMP_ENDPOINT="https://10.62.0.72"

# 运行测试
python vmp_login.py
```

## 故障排查

### 登录失败
1. 检查用户名密码是否正确
2. 确认网络连接正常
3. 检查账户是否被锁定

### API调用失败
1. 确认Cookie是否有效
2. 检查Cookie是否过期
3. 验证网络连接

### SSL/TLS问题
脚本已配置跳过SSL验证，但如果仍有问题：
- 检查Python版本（建议3.6+）
- 更新requests库：`pip install -U requests urllib3`

## 下一步工作

如需在项目中集成：
1. 实现完整的加密/解密逻辑
2. 添加Token自动刷新机制
3. 集成到Phase 22 - VDI基础集成
4. 添加错误重试和降级处理
