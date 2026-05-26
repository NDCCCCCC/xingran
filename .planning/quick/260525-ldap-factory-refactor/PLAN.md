---
quick_id: "260525-lfw"
slug: "ldap-factory-refactor"
description: "LDAP客户端连接代码重构"
created: "2026-05-25T07:26:17.216Z"
status: "in-progress"
---

# LDAP客户端连接代码重构

## 问题

以下代码模式在11个地方重复：

```go
config.AdminPassword = decryptPassword(config.AdminPassword)
client := NewLDAPClient(config)
if err := client.Connect(); err != nil {
    // error handling
}
```

## 解决方案

在 `internal/services/addomain/ldap_client.go` 添加工厂方法 `NewConnectedClient`，自动处理密码解密和连接。

### 新增方法

```go
// NewConnectedClient 创建并连接LDAP客户端（自动处理密码解密）
func NewConnectedClient(config *models.ADConfig) (*LDAPClient, error) {
    config.AdminPassword = decryptPassword(config.AdminPassword)

    client := NewLDAPClient(config)
    if err := client.Connect(); err != nil {
        return nil, err
    }

    return client, nil
}
```

## 影响范围

需要修改的文件（11个）：

1. **internal/services/addomain/config.go**
   - `TestConnection` 方法

2. **internal/services/addomain/dept_sync_service.go**
   - `SyncDeptStructureToAD` 方法

3. **internal/services/addomain/group.go**
   - `AddMember` 方法
   - `RemoveMember` 方法
   - `Update` 方法

4. **internal/services/addomain/sync.go**
   - `SyncData` 方法

5. **internal/services/addomain/user.go**
   - `Update` 方法
   - `Enable` 方法
   - `Disable` 方法
   - `Move` 方法

6. **internal/services/addomain/user_ad_sync_service.go**
   - 可能有其他需要修改的方法

## 预期收益

- 减少约33行重复代码（每个方法减少3行 × 11个方法）
- 避免类似bug（如忘记解密密码）
- 提高代码可维护性
- 未来如需修改连接逻辑，只需修改一处

## 风险评估

- **风险等级**: 低
- **原因**: 只是提取公共模式到工厂方法，不改变业务逻辑

## 实施步骤

1. 读取 `ldap_client.go`，了解现有结构
2. 添加 `NewConnectedClient` 工厂方法
3. 逐个文件替换重复代码
4. 运行 `go build ./...` 验证编译
5. 运行相关测试确保功能正常
6. 提交代码
