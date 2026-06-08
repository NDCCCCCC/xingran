# AD域组成员同步数据源重构方案

**任务ID:** 260605-iq7
**创建时间:** 2026-06-05
**方案类型:** 最佳实践重构（不考虑向后兼容）

---

## 一、问题分析

### 1.1 当前架构问题

**现状：** AD域控的OU关联AD域控的用户组，却使用本地服务器用户（`sys_user`）作为源

```go
// 当前实现（internal/scheduler/dept_sync_tasks.go:181-185）
var users []models.User
db.Where("ad_ou_dn LIKE ? AND status = ?", "%"+mapping.OUDN, models.UserStatusEnabled).
    Find(&users)
```

**问题本质：**
- **数据源不一致**：AD域控对象（OU、组）却依赖本地用户数据
- **外部依赖**：依赖 `sys_user.ad_ou_dn` 字段，该字段由用户登录时设置
- **数据完整性风险**：`sys_user` 可能没有完整的AD域用户数据
- **同步逻辑混乱**：AD→AD的操作却需要经过本地用户表中转

### 1.2 正确的架构

**目标：** AD域控OU → AD域控用户 → AD域控组（纯AD域控数据流）

```
AD域控OU (sys_ad_ou)
    ↓ 后缀匹配
AD域控用户 (sys_ad_user)
    ↓ LDAP操作
AD域控组 (sys_ad_group)
```

---

## 二、重构方案

### 2.1 总体策略

**原则：** 直接替换，不考虑向后兼容，彻底解决架构问题

**策略：**
1. 修改数据源查询逻辑
2. 移除对 `sys_user` 表的依赖
3. 使用 `sys_ad_user` 表作为唯一数据源
4. 简化同步逻辑，减少中间环节

### 2.2 修改范围

| 文件 | 修改内容 | 影响范围 |
|------|----------|----------|
| `internal/scheduler/dept_sync_tasks.go` | 修改用户查询逻辑（第181-243行） | 核心同步逻辑 |
| `internal/models/user.go` | 可选：移除 `ad_ou_dn` 字段（不再使用） | 数据模型 |

---

## 三、具体实现

### 3.1 核心代码修改

**文件：** `internal/scheduler/dept_sync_tasks.go`

**修改位置：** 第181-243行

#### 原代码（第181-243行）

```go
// 查询属于该OU及其子OU的用户（后缀匹配）
var users []models.User
if err := db.WithContext(ctx).
    Where("ad_ou_dn LIKE ? AND status = ?", "%"+mapping.OUDN, models.UserStatusEnabled).
    Find(&users).Error; err != nil {
    applogger.Errorf("查询OU %s 的用户失败: %v", mapping.OUName, err)
    totalFailed++
    continue
}

if len(users) == 0 {
    applogger.Infof("OU %s 没有启用的用户，跳过", mapping.OUName)
    continue
}

applogger.Infof("OU %s 找到 %d 个用户，开始添加到AD组", mapping.OUName, len(users))

// 查询AD组信息
var adGroup models.ADGroup
if err := db.WithContext(ctx).Where("id = ?", mapping.ADGroupID).First(&adGroup).Error; err != nil {
    applogger.Errorf("查询AD组失败: %v", err)
    totalFailed++
    continue
}

// 记录组DN用于诊断
applogger.Infof("目标AD组: 名称=%s, DN=%s", adGroup.GroupName, adGroup.GroupDN)

// 执行LDAP添加成员操作
addedCount := 0      // 成功添加
skippedCount := 0    // 已存在（正常）
notFoundCount := 0   // 用户在AD中不存在
otherFailedCount := 0 // 其他错误

for _, user := range users {
    if user.AdDn == nil || *user.AdDn == "" {
        applogger.Warnf("用户 %s 的AD DN为空，跳过", user.Username)
        otherFailedCount++
        continue
    }

    // 添加用户到AD组
    if err := ldapClient.AddGroupMember(adGroup.GroupDN, *user.AdDn); err != nil {
        errMsg := err.Error()
        // 检查错误类型
        if strings.Contains(errMsg, "Entry Already Exists") || strings.Contains(errMsg, "68") {
            // 用户已存在 - 正常状态
            applogger.Infof("用户 %s 已在组中（跳过）", user.Username)
            skippedCount++
        } else if strings.Contains(errMsg, "No Such Object") || strings.Contains(errMsg, "32") {
            // 用户在AD中不存在
            applogger.Warnf("用户 %s 在AD中不存在: %v", user.Username, err)
            notFoundCount++
        } else {
            // 其他错误
            applogger.Errorf("添加用户 %s 到组 %s 失败: %v",
                user.Username, adGroup.GroupName, err)
            otherFailedCount++
        }
    } else {
        addedCount++
    }
}
```

#### 修改后代码

```go
// 查询属于该OU及其子OU的AD域控用户（后缀匹配）
var adUsers []models.ADUser
if err := db.WithContext(ctx).
    Where("ou_dn LIKE ? AND is_enabled = ?", "%"+mapping.OUDN, true).
    Find(&adUsers).Error; err != nil {
    applogger.Errorf("查询OU %s 的AD用户失败: %v", mapping.OUName, err)
    totalFailed++
    continue
}

if len(adUsers) == 0 {
    applogger.Infof("OU %s 没有启用的AD用户，跳过", mapping.OUName)
    continue
}

applogger.Infof("OU %s 找到 %d 个AD用户，开始添加到AD组", mapping.OUName, len(adUsers))

// 查询AD组信息
var adGroup models.ADGroup
if err := db.WithContext(ctx).Where("id = ?", mapping.ADGroupID).First(&adGroup).Error; err != nil {
    applogger.Errorf("查询AD组失败: %v", err)
    totalFailed++
    continue
}

// 记录组DN用于诊断
applogger.Infof("目标AD组: 名称=%s, DN=%s", adGroup.GroupName, adGroup.GroupDN)

// 执行LDAP添加成员操作
addedCount := 0      // 成功添加
skippedCount := 0    // 已存在（正常）
notFoundCount := 0   // 用户在AD中不存在
otherFailedCount := 0 // 其他错误

for _, adUser := range adUsers {
    if adUser.UserDN == "" {
        applogger.Warnf("AD用户 %s 的DN为空，跳过", adUser.Username)
        otherFailedCount++
        continue
    }

    // 添加AD用户到AD组
    if err := ldapClient.AddGroupMember(adGroup.GroupDN, adUser.UserDN); err != nil {
        errMsg := err.Error()
        // 检查错误类型
        if strings.Contains(errMsg, "Entry Already Exists") || strings.Contains(errMsg, "68") {
            // 用户已存在 - 正常状态
            applogger.Infof("AD用户 %s 已在组中（跳过）", adUser.Username)
            skippedCount++
        } else if strings.Contains(errMsg, "No Such Object") || strings.Contains(errMsg, "32") {
            // 用户在AD中不存在
            applogger.Warnf("AD用户 %s 在AD中不存在: %v", adUser.Username, err)
            notFoundCount++
        } else {
            // 其他错误
            applogger.Errorf("添加AD用户 %s 到组 %s 失败: %v",
                adUser.Username, adGroup.GroupName, err)
            otherFailedCount++
        }
    } else {
        addedCount++
    }
}
```

### 3.2 修改要点对比

| 项目 | 原代码（sys_user） | 新代码（sys_ad_user） |
|------|-------------------|----------------------|
| 数据表 | `models.User` | `models.ADUser` |
| 变量名 | `users` | `adUsers` |
| OU字段 | `ad_ou_dn` | `ou_dn` |
| 状态字段 | `status = 0` | `is_enabled = true` |
| DN字段 | `AdDn` (指针) | `UserDN` (直接值) |
| DN检查 | `user.AdDn == nil \|\| *user.AdDn == ""` | `adUser.UserDN == ""` |
| 日志标识 | "用户" | "AD用户" |

---

## 四、数据依赖验证

### 4.1 前置条件检查

**必须满足以下条件才能执行重构：**

```sql
-- 1. 检查 sys_ad_user 表是否有数据
SELECT COUNT(*) FROM sys_ad_user WHERE deleted_at IS NULL;

-- 2. 检查 OU-组映射配置
SELECT 
    m.id, 
    m.ou_dn, 
    m.ou_name, 
    g.group_name,
    g.group_dn
FROM sys_ou_group_mapping m
JOIN sys_ad_group g ON m.ad_group_id = g.id
WHERE m.mapping_status = 0 AND m.sync_enabled = true;

-- 3. 检查每个OU下是否有AD用户
SELECT 
    ou.ou_dn,
    ou.ou_name,
    COUNT(u.id) as user_count
FROM sys_ad_ou ou
LEFT JOIN sys_ad_user u ON u.ou_dn LIKE '%' || ou.ou_dn AND u.is_enabled = true
GROUP BY ou.ou_dn, ou.ou_name
ORDER BY user_count DESC;
```

### 4.2 数据一致性验证

**执行重构前建议运行以下验证：**

```sql
-- 验证关键OU下的用户覆盖情况
-- 例如：检查某个特定OU
SELECT 
    'sys_user' as source,
    COUNT(*) as count
FROM sys_user 
WHERE ad_ou_dn LIKE '%OU=Sales,DC=example,DC=com' AND status = 0

UNION ALL

SELECT 
    'sys_ad_user' as source,
    COUNT(*) as count
FROM sys_ad_user 
WHERE ou_dn LIKE '%OU=Sales,DC=example,DC=com' AND is_enabled = true;
```

---

## 五、测试验证方案

### 5.1 单元测试修改

**测试文件：** `internal/scheduler/dept_sync_tasks_test.go`

```go
func TestExecuteDeptMemberToADGroupSyncTask_WithADUsers(t *testing.T) {
    // 准备测试数据
    adConfig := &models.ADConfig{
        ID:          "test-config-1",
        ConfigName:  "Test AD",
        ServerAddress: "ldap.example.com",
        BaseDN:      "DC=example,DC=com",
        Status:      models.ADConfigStatusEnabled,
    }
    
    ouMapping := &models.OUGroupMapping{
        ID:          "test-mapping-1",
        ADConfigID:  adConfig.ID,
        OUDN:        "OU=Sales,DC=example,DC=com",
        OUName:      "Sales",
        ADGroupID:   "test-group-1",
        SyncEnabled: true,
        MappingStatus: models.OUGroupMappingStatusActive,
    }
    
    // 创建AD用户（不是系统用户）
    adUsers := []models.ADUser{
        {
            ADConfigID: adConfig.ID,
            UserDN:     "CN=user1,OU=Sales,DC=example,DC=com",
            Username:   "user1",
            OUN:        "OU=Sales,DC=example,DC=com",
            IsEnabled:  true,
        },
        {
            ADConfigID: adConfig.ID,
            UserDN:     "CN=user2,OU=Sales,DC=example,DC=com",
            Username:   "user2",
            OUN:        "OU=Sales,DC=example,DC=com",
            IsEnabled:  true,
        },
    }
    
    // ... 执行测试
}
```

### 5.2 集成测试步骤

**1. 准备环境**
```bash
# 确保AD同步已完成
# 检查 sys_ad_user 表有完整数据
```

**2. 手动触发同步任务**
```bash
# 通过API或直接调用触发
curl -X POST http://localhost:9000/api/v1/scheduler/execute \
  -H "Authorization: Bearer <token>" \
  -d '{
    "taskName": "dept_member_to_ad_group_sync",
    "params": {}
  }'
```

**3. 验证结果**
```bash
# 检查日志输出
# 应该看到 "找到 X 个AD用户" 而不是 "找到 X 个用户"
```

**4. LDAP验证**
```bash
# 在AD域控服务器上验证组成员
# 确认用户被正确添加到组中
```

---

## 六、部署方案

### 6.1 部署步骤

**阶段一：代码准备**
1. 创建功能分支
   ```bash
   git checkout -b refactor/ad-member-sync-source
   ```

2. 应用代码修改
   - 修改 `dept_sync_tasks.go`
   - 更新单元测试

3. 代码审查
   ```bash
   # 确保所有修改正确
   go build ./...
   go test ./internal/scheduler/...
   ```

**阶段二：测试环境验证**
1. 部署到测试环境
2. 准备测试数据（确保AD同步已完成）
3. 手动触发同步任务
4. 验证结果正确性

**阶段三：生产部署**
1. 选择维护窗口
2. 执行部署
3. 监控首次同步执行
4. 验证生产数据正确性

### 6.2 回滚方案

**如果出现问题，回滚步骤：**

1. 还原代码到修改前版本
   ```bash
   git revert <commit-hash>
   ```

2. 重新部署

3. 验证回滚后系统正常

---

## 七、监控和日志

### 7.1 关键监控指标

**新增监控指标：**

| 指标 | 说明 | 告警阈值 |
|------|------|----------|
| `ad_member_sync_source_users` | 同步的AD用户数量 | 记录基线，偏差>20%告警 |
| `ad_member_sync_success_rate` | 同步成功率 | <95% 告警 |
| `ad_member_sync_ldap_error_rate` | LDAP操作错误率 | >5% 告警 |
| `ad_member_sync_empty_ou_count` | 无用户的OU数量 | >3个告警 |

### 7.2 日志改进

**修改日志输出：**

```go
// 明确标识使用AD域控用户
applogger.Infof("[AD域成员同步] 开始同步配置ID: %s", configID)
applogger.Infof("[AD域成员同步] 找到 %d 个活动的OU-组映射", len(mappings))
applogger.Infof("[AD域成员同步] OU=%s → 组=%s", mapping.OUName, adGroup.GroupName)
applogger.Infof("[AD域成员同步] OU %s 找到 %d 个AD用户", mapping.OUName, len(adUsers))
applogger.Infof("[AD域成员同步] 添加=%d, 已存在=%d, AD中不存在=%d, 其他错误=%d",
    addedCount, skippedCount, notFoundCount, otherFailedCount)
```

---

## 八、风险评估

### 8.1 潜在风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| AD同步未完成 | 无用户可同步 | 确保AD同步任务正常执行后再部署 |
| OU-组映射配置错误 | 同步错误用户 | 部署前验证所有映射配置 |
| LDAP权限不足 | 无法添加成员 | 确保LDAP账户有修改组成员权限 |
| 数据不一致 | 同步结果不符合预期 | 部署前后对比数据，添加监控 |

### 8.2 验证清单

**部署前检查：**
- [ ] AD同步任务正常执行，`sys_ad_user` 表有完整数据
- [ ] OU-组映射配置正确且启用
- [ ] LDAP账户有修改组成员权限
- [ ] 代码审查完成，测试通过
- [ ] 回滚方案准备就绪

**部署后验证：**
- [ ] 同步任务成功执行
- [ ] 日志输出正确（使用AD用户）
- [ ] LDAP组成员正确添加
- [ ] 无异常错误
- [ ] 监控指标正常

---

## 九、清理工作（可选）

### 9.1 移除遗留依赖

**如果确认不再需要，可以清理以下内容：**

1. **移除 `sys_user.ad_ou_dn` 字段**
   ```sql
   -- 谨慎操作：建议先备份数据
   ALTER TABLE sys_user DROP COLUMN IF EXISTS ad_ou_dn;
   ```

2. **移除相关代码**
   - 检查其他使用 `ad_ou_dn` 的代码
   - 移除用户登录时设置 `ad_ou_dn` 的逻辑

3. **清理数据库迁移**
   - 移除添加 `ad_ou_dn` 字段的迁移文件

**注意：** 清理工作应该在重构部署并稳定运行后进行。

---

## 十、总结

### 10.1 重构收益

1. **架构一致性**：AD域控对象使用AD域控数据源
2. **消除外部依赖**：不再依赖本地用户表
3. **简化数据流**：AD→AD 直接同步，无需中转
4. **提高可靠性**：使用单一数据源，减少数据不一致风险
5. **降低维护成本**：减少中间环节，简化同步逻辑

### 10.2 实施建议

**推荐实施顺序：**

1. **立即执行**：修改同步逻辑（核心修改）
2. **验证测试**：在测试环境充分验证
3. **生产部署**：选择合适时机部署
4. **监控观察**：部署后密切监控
5. **清理优化**：稳定后清理遗留代码

**预计工作量：**
- 代码修改：1-2小时
- 测试验证：2-4小时
- 部署监控：1-2小时
- **总计：半天工作量**

---

## 十一、代码修改清单

### 文件修改列表

| 文件路径 | 修改类型 | 行数估计 |
|----------|----------|----------|
| `internal/scheduler/dept_sync_tasks.go` | 修改用户查询和处理逻辑 | ~60行 |
| `internal/scheduler/dept_sync_tasks_test.go` | 更新单元测试 | ~40行 |
| （可选）`internal/models/user.go` | 移除 `ad_ou_dn` 字段 | ~3行 |

### 变更对比摘要

```
主要变更：
- 数据表：sys_user → sys_ad_user
- OU字段：ad_ou_dn → ou_dn  
- 状态字段：status → is_enabled
- DN字段：AdDn (指针) → UserDN (值)
- 变量命名：user → adUser
- 日志标识："用户" → "AD用户"
```

---

**方案完成时间：** 2026-06-05
**建议优先级：** 高（架构不合理，应尽快修复）
**实施复杂度：** 低（简单替换，逻辑清晰）
