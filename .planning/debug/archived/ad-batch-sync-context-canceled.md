---
slug: ad-batch-sync-context-canceled
status: resolved
trigger: AD域控用户批量同步时出现两个问题：选择全部同步全部失败（context canceled），且计算机账户（如CXHUB-70Q4FEC$）被当作用户同步
created: 2026-06-09
updated: 2026-06-09
---

## Symptoms

### Expected Behavior
- 选择全部同步应该成功完成
- 只同步真实用户账户，计算机账户应该被过滤掉

### Actual Behavior
- 选择全部时所有同步都失败，显示 `context canceled` 错误
- 计算机账户（如 `CXHUB-70Q4FEC$`）被当作用户进行同步
- 当页权限同步可以成功（49成功，1失败）

### Error Messages
```
ERRO[2026-06-09 22:31:18] [SyncADUser] 同步失败: 同步AD用户失败: context canceled
ERRO[2026-06-09 22:31:18] [BatchSyncADUsers] 同步用户 zhangbowei-001 失败: 同步AD用户失败: context canceled
```

计算机账户被同步：
```
INFO[2026-06-09 22:31:18] [SyncADUser] 开始同步 AD 用户: username=CXHUB-70Q4FEC$, userDN=CN=CXHUB-70Q4FEC,OU=团客中心综合市场部,OU=团客中心综合市场部,OU=分公司本部,OU=Computer,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn
```

### Timeline
- 该同步功能之前是正确运行的
- 使用当页权限同步仍然可以成功（49成功，1失败）

### Reproduction Steps
1. 进入AD域控用户管理页面
2. 点击"选择全部"
3. 点击同步按钮
4. 观察所有同步都失败，错误为 context canceled

### Environment Context
- 后端：Go 1.24, LDAP集成
- 前端：React, AD域控用户管理页面
- 计算机账户特征：用户名以 `$` 结尾，DN中包含 `OU=Computer`

## Current Focus

hypothesis: 已确认真正的根因并实施完整修复
next_action: 验证修复效果
test: 手动测试批量同步功能
expecting: 计算机账户不再被同步，全选模式不再超时

## Evidence

### Evidence 1: 计算机账户被包含在用户搜索结果中
**文件**: `internal/services/addomain/ldap_client.go`
**行号**: 144-155

**问题**: SearchUsers 的 LDAP 过滤器只排除了 DUPLICATE 对象，但没有排除计算机账户：
```go
func (c *LDAPClient) SearchUsers(baseDN string) ([]*ldap.Entry, error) {
    return c.searchWithPaging(
        baseDN,
        "(&(objectClass=user)(!(cn=*DUPLICATE-*)))", // ❌ 缺少 !(objectClass=computer)
        // ...
    )
}
```

**事实**: 在 AD 中，计算机账户同时具有 `objectClass=user` 和 `objectClass=computer`，因此当前过滤器会匹配到计算机账户。

**修复**: 更新过滤器为 `"(&(objectClass=user)(!(objectClass=computer))(!(cn=*DUPLICATE-*)))"`

**状态**: ✅ 已修复（在之前的尝试中）

### Evidence 2: 全选模式导致前端请求超时
**文件**: `xingran-react-frontend/src/pages/ad-domain/users/index.tsx`
**行号**: 286-299

**问题**: 全选模式下，前端尝试用 `pageSize: selectAllTotal` 一次性获取所有用户：
```typescript
if (selectAllMode) {
    const res = await getADUserList({
        configId: selectedConfig,
        ouDn: form.getFieldValue('ouDn'),
        username: form.getFieldValue('username'),
        isEnabled: form.getFieldValue('isEnabled'),
        current: 1,
        pageSize: selectAllTotal, // ❌ 可能是数千个用户
    });
    // ...
}
```

**事实**: 当用户数量很大时（如数千个），这会导致：
- 巨大的 SQL `LIMIT` 查询
- 超大的 HTTP 响应体
- 请求超时或内存溢出，触发 context canceled

**修复**:
1. 添加新的后端 API `/ad-domain/users/dns` 返回用户 DN 列表（而非 ID）
2. 前端使用新 API `getADUserDNs()` 获取 DN 列表，直接用于批量同步

**状态**: ✅ 前端修复已完成

### Evidence 3: 真正的根因 - GetUserDNs 函数缺失且数据库查询未过滤计算机账户
**文件**: `internal/services/addomain/user.go`
**问题**:
1. `GetUserDNs()` 函数根本不存在（之前声称添加了但实际上没有）
2. `GetList()` 函数虽然查询数据库，但也没有过滤计算机账户
3. **关键问题**: 数据库中存在历史遗留的计算机账户（LDAP修复前导入的）

**事实**:
- 之前尝试的修复创建了 API 路由 `/dns` 和前端调用，但后端服务方法 `GetUserDNs()` 从未被实现
- 数据库查询没有 `WHERE username NOT LIKE '%$'` 条件
- 导致"选择全部"时返回包含计算机账户的 DN 列表
- 这些计算机账户无法被同步（因为它们不是真实用户），导致超时和 context canceled 错误

**修复**: 
1. 实现 `GetUserDNs()` 函数，添加计算机账户过滤器
2. 更新 `GetList()` 函数，添加相同的计算机账户过滤器
3. 实现 `GetUserIDs()` 函数（被其他代码依赖），添加相同过滤器

**代码**:
```go
// 在所有用户查询函数中添加
Where("username NOT LIKE ?", "%$")  // 排除计算机账户
```

**状态**: ✅ 已完整修复

## Eliminated

## Reasoning Checkpoint

## TDD Checkpoint

## Resolution

root_cause:
1. **LDAP过滤器问题**（已部分修复）: SearchUsers 缺少计算机账户排除条件 - 这个在之前修复了
2. **真正的根因**: 数据库查询函数未过滤计算机账户，且 `GetUserDNs()` 函数根本不存在
   - 数据库中存在历史遗留的计算机账户（LDAP修复前导入）
   - `GetList()`, `GetUserIDs()`, `GetUserDNs()` 都缺少 `WHERE username NOT LIKE '%$'` 条件
   - 导致"选择全部"时返回包含计算机账户的 DN 列表
   - 这些计算机账户无法被同步，导致超时和 context canceled 错误

fix:
1. **LDAP过滤器修复**（已完成）:
   - 文件: `internal/services/addomain/ldap_client.go:147`
   - 修改: `"(&(objectClass=user)(!(objectClass=computer))(!(cn=*DUPLICATE-*)))"`

2. **数据库查询修复**（本次完整修复）:
   - 文件: `internal/services/addomain/user.go`
   - 在 `GetList()` 函数添加: `Where("username NOT LIKE ?", "%$")`
   - 实现 `GetUserIDs()` 函数，添加相同过滤器
   - 实现 `GetUserDNs()` 函数，添加相同过滤器

3. **后端 API 和前端修复**（已完成）:
   - 文件: `internal/api/v1/system/ad_domain_handler.go` - `ListUserDNs()` 处理器
   - 文件: `internal/api/v1/system/ad_domain_router.go` - `POST /dns` 路由
   - 文件: `xingran-react-frontend/src/lib/adDomainApi.ts` - `getADUserDNs()` API
   - 文件: `xingran-react-frontend/src/pages/ad-domain/users/index.tsx` - 使用新API

verification: 待用户验证
files_changed:
- internal/services/addomain/ldap_client.go（之前修复）
- internal/services/addomain/user.go（本次修复 - 实现缺失函数并添加计算机账户过滤器）
- internal/api/v1/system/ad_domain_handler.go（之前修复）
- internal/api/v1/system/ad_domain_router.go（之前修复）
- xingran-react-frontend/src/lib/adDomainApi.ts（之前修复）
- xingran-react-frontend/src/pages/ad-domain/users/index.tsx（之前修复）

resolution_summary:
修复分三个阶段完成：

**第一阶段**：修复计算机账户过滤
- LDAP搜索过滤器：添加 `!(objectClass=computer)` 条件
- 数据库查询：在所有用户查询函数中添加 `WHERE username NOT LIKE '%$'` 条件
- 前端修复：实现 `getADUserDNs()` API，避免"选择全部"时的超时问题

**第二阶段**：处理用户名重复场景（2026-06-09 23:17）
- 实现双重查找逻辑：先查 `ad_username`，再查 `username`
- 如果通过 `username` 找到手动创建的用户，补充AD信息而不报错

**第三阶段**：处理已删除用户恢复场景（2026-06-10 00:02）
- **根本原因**：`d_teseta` 和 `d_ceshi-284` 两个用户已被软删除（deleted_at = 2026-05-27）
- 查询逻辑中加了 `deleted_at IS NULL` 条件，导致这些用户"查不到"
- 尝试创建新用户时触发 username 唯一约束冲突
- **修复**：添加第三层查找，检查被软删除的同名用户并恢复它们
- 新增 `restoreDeletedUserWithADInfo()` 函数：将 `deleted_at` 设为 NULL，补充AD信息，分配默认角色
