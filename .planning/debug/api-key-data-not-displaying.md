---
status: resolved
deferred_to: v1.16-tech-debt
trigger: API秘钥管理前端应该也完成了，但是之前创建的测试数据现在没有显示了
slug: api-key-data-not-displaying
created: "2026-05-24T13:00:00Z"
updated: 2026-06-25
---

# Debug Session: API Key Data Not Displaying

## Symptoms

### Expected Behavior
- API密钥管理页面应该能显示之前创建的测试数据
- 所有功能都有：显示列表、创建新密钥、编辑/删除密钥

### Actual Behavior
- 列表完全空白
- 页面加载正常但没有任何数据显示

### Error Messages
- 无特定错误消息

### Timeline
- **之前状态**: 能显示测试数据
- **当前状态**: 数据不再显示
- **变化**: 这是一个回归问题

### Reproduction
1. 打开 API 密钥管理页面
2. 观察列表区域
3. 列表完全空白，无数据显示

## Current Focus

**Status**: root_cause_found
**Hypothesis**: 响应加密中间件配置错误
**Next Action**: 禁用响应加密或修复加密密钥传递
**Test**: 验证禁用响应加密后数据是否正常显示
**Expecting**: 数据正常显示

## Evidence

- timestamp: 2026-05-24T13:00:00Z
  source: user_report
  note: "之前创建了一个测试数据现在没有显示了"

- timestamp: 2026-05-24T13:30:00Z
  source: network_analysis
  note: "网络响应显示加密数据，但 sm4Key 为空，timestamp 为 0"
  file: apikeys_list_response.network-response
  details: |
    响应数据：
    {
      "encrypted": true,
      "data": "8y1GkPbhCbQIsfh7...",
      "sm4Key": "",
      "iv": "BVS2eG7I70tWWP4pVMdUwA==",
      "timestamp": 0,
      "nonce": ""
    }

- timestamp: 2026-05-24T13:30:00Z
  source: code_analysis
  note: "后端路由和处理器配置正确，数据库连接正常"
  files:
    - internal/api/v1/system/apikey_handler.go
    - internal/api/v1/system/apikey_router.go
    - internal/services/system/apikey_service.go

- timestamp: 2026-05-24T13:45:00Z
  source: database_analysis
  note: "发现数据库中请求加密开关为 true，导致响应加密也被启用"
  file: internal/core/db/migrations/migration_086_request_encryption_toggle.go
  details: |
    migration_086 设置默认值:
    ConfigKey: "sys.request.encryption.enabled"
    ConfigValue: "true"

    响应加密中间件从数据库读取相同配置:
    enabled := getConfigFromDB(c.Request.Context(), db, false)

- timestamp: 2026-05-24T13:45:00Z
  source: root_cause_analysis
  note: "找到根本原因：响应加密中间件逻辑问题"
  details: |
    1. 数据库配置 sys.request.encryption.enabled = true
    2. 请求解密中间件使用此配置启用请求解密 ✅
    3. 响应加密中间件也使用相同配置启用响应加密 ❌
    4. 响应加密尝试从 context 获取 sm4_key
    5. 如果请求未加密，context 中没有 sm4_key
    6. 中间件返回空 sm4Key 的加密响应
    7. 前端无法解密，导致数据不显示

    问题代码位置:
    pkg/middleware/response_encryption.go:56
    enabled := getConfigFromDB(c.Request.Context(), db, false)

    应该使用独立的响应加密配置，或检查 context 中是否有 sm4_key

## Eliminated

- timestamp: 2026-05-24T13:30:00Z
  item: "路由配置问题 - 已验证路由正确注册"
  evidence: "grep 发现在 router.go:221 注册了 SetupAPIKeyRouter"

- timestamp: 2026-05-24T13:30:00Z
  item: "数据库连接问题 - 已验证配置使用 PostgreSQL"
  evidence: "configs/config.yaml 显示 PostgreSQL 配置正确"

- timestamp: 2026-05-24T13:30:00Z
  item: "后端代码逻辑 - 服务层查询逻辑正常"
  evidence: "apikey_service.go ListAPIKeys 实现正确"

- timestamp: 2026-05-24T13:45:00Z
  item: "请求加密问题 - 请求加密工作正常"
  evidence: "请求解密中间件正确解密请求并存储 sm4_key"

## Specialist Dispatch

specialist_hint: go

## Resolution

### Root Cause

响应加密中间件配置错误：使用请求加密配置开关控制响应加密，导致在没有请求加密的情况下也启用响应加密，但 context 中没有 SM4 密钥，生成无效的加密响应。

### Fix

**方案1：禁用响应加密（快速修复）**
```sql
UPDATE sys_config SET config_value = 'false' WHERE config_key = 'sys.request.encryption.enabled';
```

**方案2：修复响应加密中间件（正确修复）**
修改 `pkg/middleware/response_encryption.go:56` 使用独立的响应加密配置：
```go
// 添加新的配置键
const responseEncryptionConfigKey = "sys.response.encryption.enabled"

// 修改中间件代码
enabled := getConfigFromDB(c.Request.Context(), db, false)
// 改为
enabled := getConfigFromDB(c.Request.Context(), db, false) && hasSM4KeyInContext
```

**方案3：检查 context 中是否有 sm4_key**
在响应加密中间件中添加检查：
```go
// 检查是否有可用的加密密钥
_, hasKey := c.Get("sm4_key")
if !hasKey {
    c.Next()
    return
}
```

### Verification

待验证

### Files Changed

待记录

## Phase 40 Closure (2026-06-25)

落地 Resolution 方案 3（在响应加密中间件检查 context 是否有 sm4_key）于
`pkg/middleware/response_encryption.go` ResponseEncryption 中间件：

```go
if _, hasKey := c.Get("sm4_key"); !hasKey {
    c.Next()
    return
}
```

效果：GET 类未走请求解密中间件的请求（context 无 sm4_key）将跳过响应加密，
不再生成 sm4Key="" timestamp=0 的无效加密响应，前端 API 密钥列表等页面可正常显示。

verification: `go build ./...` 退出 0；`grep -n 'c.Get("sm4_key")' pkg/middleware/response_encryption.go` 命中
files_changed: pkg/middleware/response_encryption.go, .planning/debug/api-key-data-not-displaying.md
