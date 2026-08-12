---
quick_id: "260605-h3p"
slug: "ad-group-auto-sync"
description: "完成AD组自动化同步功能 - 更新定时任务执行函数实现基于OU-组映射的成员同步"
created: "2026-06-05T04:18:50.854Z"
status: "in-progress"
author: "Claude"
priority: "high"
tags: ["ad", "group-sync", "ou-mapping", "scheduler", "automation"]
estimated_duration: "45min"
---

# Quick Task: AD组自动化同步功能

## 背景

1. 已有AD组同步定时任务配置（Migration 137），但执行函数标记为废弃
2. AD域控-组织单元页面已有OU-组绑定功能（`OUGroupMapping`）
3. 需要实现将数据库中属于该OU的用户自动同步到关联的AD组

## 需求

更新 `dept_sync_tasks.go` 中的 `executeDeptMemberToADGroupSyncTask` 函数，实现基于OU-组映射的成员同步：

1. **查询活动映射**: 读取 `sys_ou_group_mapping` 表中启用的映射关系
2. **获取用户**: 根据OU DN查询数据库中 `ad_ou_dn` 匹配的用户
3. **LDAP操作**: 将用户添加到对应的AD组
4. **记录日志**: 同步结果记录到 `sys_ou_group_mapping_sync_log` 表

## 实现步骤

### Step 1: 实现OU组同步执行函数

更新 `internal/scheduler/dept_sync_tasks.go` 中的 `executeDeptMemberToADGroupSyncTask` 函数：

```go
func executeDeptMemberToADGroupSyncTask(ctx context.Context, params map[string]interface{}) error {
    // 1. 获取AD配置
    // 2. 查询启用的OU-组映射
    // 3. 对每个映射执行同步：
    //    - 查询属于该OU的用户（WHERE ad_ou_dn = ?）
    //    - 连接AD域控
    //    - 将用户添加到AD组
    //    - 记录同步日志
}
```

### Step 2: 实现LDAP组成员操作

在 `internal/services/addomain/ldap_client.go` 中确保有：
- `GetGroupMembers(groupDN string)` - 获取组成员
- `AddGroupMember(groupDN, userDN string)` - 添加成员
- `RemoveGroupMember(groupDN, userDN string)` - 移除成员

### Step 3: 测试验证

- 定时任务页面配置并手动触发
- 检查AD域控中组成员是否正确
- 验证同步日志记录

## 验证标准

- [ ] `executeDeptMemberToADGroupSyncTask` 不再返回废弃错误
- [ ] 能正确查询活动OU-组映射
- [ ] 能根据OU DN找到匹配的用户
- [ ] LDAP操作成功执行
- [ ] 同步日志正确记录
- [ ] 编译通过 (`go build ./internal/scheduler/`)

## 相关文件

- `internal/scheduler/dept_sync_tasks.go` - 主要修改文件
- `internal/services/addomain/ldap_client.go` - LDAP客户端操作
- `internal/models/ou_group_mapping.go` - OU-组映射模型
- `internal/models/user.go` - 用户模型（包含 `ad_ou_dn` 字段）

## 数据流

```
sys_ou_group_mapping (活动映射)
         ↓
    查询 ad_ou_dn 匹配的用户
         ↓
    LDAP 连接AD域控
         ↓
    用户 → AD组成员操作
         ↓
sys_ou_group_mapping_sync_log (记录结果)
```

## 注意事项

1. 复用现有LDAP客户端方法，避免重复代码
2. 处理LDAP连接错误和权限问题
3. 同步失败时记录详细错误信息
4. 支持增量同步（仅添加不存在的成员）
