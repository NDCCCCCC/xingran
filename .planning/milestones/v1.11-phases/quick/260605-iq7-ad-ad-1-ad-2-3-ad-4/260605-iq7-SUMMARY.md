---
title: AD域组成员同步数据源分析
status: complete
completed_at: 2026-06-05T05:29:00Z
---

# AD域组成员同步数据源分析 - 执行摘要

## 任务目标

检查AD域组成员同步的数据源是否可以从系统用户替换成AD域控用户。

## 分析结论

**✅ 技术上可行，建议采用渐进式方案**

## 关键发现

### 1. 当前实现

**文件位置：** `internal/scheduler/dept_sync_tasks.go` (第115-276行)

**数据源：** 系统用户表（`sys_user`）

```go
// 当前查询逻辑（第181-185行）
var users []models.User
db.Where("ad_ou_dn LIKE ? AND status = ?", "%"+mapping.OUDN, models.UserStatusEnabled).
    Find(&users)
```

**同步方向：** 系统 → AD（将系统用户添加到AD域组）

### 2. 替换可行性

**✅ 可行** - AD域控用户表（`sys_ad_user`）具备所需字段：

| 需求字段 | sys_user | sys_ad_user |
|---------|----------|-------------|
| OU匹配 | ad_ou_dn | ou_dn |
| 用户DN | ad_dn | user_dn |
| 启用状态 | status (0=启用) | is_enabled (true=启用) |

### 3. 修改方案

**修改点：** `dept_sync_tasks.go` 第181-243行

**核心变更：**
```go
// 查询改为使用AD域控用户表
var adUsers []models.ADUser
db.Where("ou_dn LIKE ? AND is_enabled = ?", "%"+mapping.OUDN, true).
    Find(&adUsers)

// LDAP操作改为使用 user_dn 字段
ldapClient.AddGroupMember(adGroup.GroupDN, adUser.UserDN)
```

## 依赖和限制

### 前置条件

1. **AD同步必须完成** - `sys_ad_user` 表需要有数据
2. **OU-组映射配置正确** - `ou_dn` 匹配逻辑
3. **用户DN有效** - 必须在AD域中存在

### 潜在风险

1. **数据一致性** - 两个数据源可能不一致
2. **同步时机** - 必须先执行AD同步，再执行组成员同步
3. **LDAP依赖** - 最终需要LDAP操作成功

## 推荐实施步骤

### 阶段一：添加配置选项（保持向后兼容）
- 在 `OUGroupMapping` 模型中添加 `DataSource` 字段
- 修改同步任务支持双数据源
- 默认值保持 `sys_user`

### 阶段二：数据验证
- 验证 `sys_ad_user` 表数据完整性
- 对比两个数据源的用户数量
- 确认 `user_dn` 有效性

### 阶段三：测试切换
- 在测试环境切换到 `ad_user` 数据源
- 验证同步结果正确性
- 监控LDAP操作成功率

### 阶段四：生产切换
- 选择特定映射进行切换
- 逐步扩大切换范围
- 持续监控数据一致性

## 相关文件

| 文件 | 说明 |
|------|------|
| `internal/scheduler/dept_sync_tasks.go` | **需修改** - 成员同步任务实现 |
| `internal/services/addomain/sync.go` | AD数据同步服务 |
| `internal/models/ad_domain.go` | AD域数据模型 |
| `internal/models/ou_group_mapping.go` | OU-组映射模型 |

## 最佳实践实施方案

完整的技术实施方案请参考：[260605-iq7-IMPLEMENTATION.md](./260605-iq7-IMPLEMENTATION.md)

**核心修改：**
- 文件：`internal/scheduler/dept_sync_tasks.go`
- 修改行数：约60行（第181-243行）
- 变更：`sys_user` → `sys_ad_user`

**关键代码变更：**
```go
// 原代码
var users []models.User
db.Where("ad_ou_dn LIKE ? AND status = ?", "%"+mapping.OUDN, models.UserStatusEnabled)

// 新代码  
var adUsers []models.ADUser
db.Where("ou_dn LIKE ? AND is_enabled = ?", "%"+mapping.OUDN, true)
```

---

**任务完成时间：** 2026-06-05  
**分析深度：** 代码级别完整分析  
**建议置信度：** 高（基于实际代码实现）
