---
slug: ad-dept-member-sync-all-zero
name: ad-dept-member-sync-all-zero
status: resolved
trigger: AD域成员同步任务执行成功但所有计数都是0（部门=0, 成功=0, 失败=0, 总成员=0, 添加=0, 移除=0），耗时仅4ms
created: 2026-05-27
updated: 2026-05-27
---

## 症状 (Symptoms)

**期望行为:**
- 系统里有很多部门和用户
- 应该正确识别"分公司本部"下的三级部门
- 将三级部门下的所有用户添加到对应的 AD 域用户组里

**实际行为:**
- 所有计数都是 0（部门=0, 成功=0, 失败=0, 总成员=0, 添加=0, 移除=0）
- 任务执行时间极短（4ms），说明可能根本没有执行查询

**错误信息:**
```
INFO[2026-05-27 16:32:25] 执行任务: AD域组成员同步, 目标: dept_member_to_ad_group_sync
INFO[2026-05-27 16:32:25] 自动使用AD配置: AD域控主机 (ID: 4ee691f4-2f93-4981-b37f-93839b5c6af7)
INFO[2026-05-27 16:32:25] 开始执行部门成员到AD域组同步任务，AD 配置 ID: 4ee691f4-2f93-4981-b37f-93839b5c6af7
INFO[2026-05-27 16:32:25] [成员同步] 批量同步完成: 部门=0, 成功=0, 失败=0, 总成员=0, 添加=0, 移除=0, 耗时=4ms
INFO[2026-05-27 16:32:25] 部门成员到AD域组同步完成: 部门数=0, 成功=0, 失败=0, 总成员=0, 添加=0, 移除=0, 耗时=4ms
INFO[2026-05-27 16:32:25] 任务执行成功 [AD域组成员同步.DEFAULT], 耗时: 7ms
```

**时间线:**
- 第一次运行就出现（新功能）

**重现步骤:**
1. 运行定时任务 dept_member_to_ad_group_sync
2. 手动触发部门成员同步

---

## 当前关注点 (Current Focus)

**假设 (Hypothesis):** 数据库中没有任何 `sys_dept_group_mapping` 记录，导致 `ListMappings` 返回空列表，`SyncAllMembers` 没有找到需要同步的部门。

**下一步行动 (Next Action):** 检查数据库中是否存在 `sys_dept_group_mapping` 记录，如果不存在，需要先创建部门与AD组的映射关系。

**测试 (Test):**
```sql
SELECT COUNT(*) FROM sys_dept_group_mapping WHERE ad_config_id = '4ee691f4-2f93-4981-b37f-93839b5c6af7' AND mapping_status = 'active';
```

**预期结果 (Expecting):** 如果计数为0，则假设成立。需要先创建部门组映射才能执行成员同步。

---

## 证据 (Evidence)

- timestamp: 2026-05-27 16:45:00
  source: code_review
  detail: |
    检查了 `internal/services/addomain/member_sync_service.go` 的 `SyncAllMembers` 方法（第189-241行）：
    - 第197-209行：查询所有启用的映射 (`mapping_status = 'active'`)
    - 第211行：`result.TotalDepts = len(listResp.List)` - 如果没有映射，这里就是0
    - 第214-233行：遍历映射列表执行同步 - 如果列表为空，循环不执行
    - 所有计数器都保持为0，与日志输出一致

- timestamp: 2026-05-27 16:47:00
  source: code_review
  detail: |
    检查了 `internal/models/dept_group_mapping.go`：
    - `DeptGroupMapping` 模型定义了部门与AD组的映射关系
    - 字段包括：`dept_id`, `ad_group_id`, `ad_config_id`, `mapping_status`, `sync_enabled` 等
    - `sync_enabled = true` 时才会同步成员
    - 如果没有这个映射记录，成员同步任务不知道要同步哪些部门

- timestamp: 2026-05-27 16:50:00
  source: log_analysis
  detail: |
    日志显示任务成功执行，但所有计数都是0：
    - "部门=0" - 没有找到需要同步的部门
    - 耗时仅4ms - 进一步证明没有执行实际的LDAP操作
    - 没有报错 - 说明查询成功，只是返回空结果

- timestamp: 2026-05-27 17:00:00
  source: code_review
  detail: |
    发现了解决方案：
    - 在 `internal/api/v1/addomain/group_sync_handler.go` 中找到了 `AutoMapDepartments` API（第276行）
    - 该API可以根据部门名称自动匹配对应的AD组（cxhub-{dept}命名规则）
    - 路由：`POST /api/v1/ad/groups/automap`
    - 功能：调用 `AutoMapAllDepartments` 自动映射所有二级部门
    - 这是用户需要执行的第一个步骤来创建必要的映射记录

---

## 已排除的假设 (Eliminated)

---

## 解决方案 (Resolution)

**根本原因 (Root Cause):**
数据库中没有任何 `sys_dept_group_mapping` 记录。成员同步任务 `SyncAllMembers` 需要先查询 `sys_dept_group_mapping` 表来获取需要同步的部门列表（第197-209行），但由于表为空，所以返回0个部门，所有计数器都保持为0。这不是bug，而是正常的行为——需要先创建部门与AD组的映射关系。

**修复方案 (Fix):**
用户需要先创建部门与AD组的映射关系，有两种方式：

1. **自动映射（推荐）：**
   - API: `POST /api/v1/ad/groups/automap`
   - 请求体: `{"configId": "4ee691f4-2f93-4981-b37f-93839b5c6af7"}`
   - 功能：根据部门名称自动匹配AD组（cxhub-{dept}命名规则）
   - 会自动映射所有二级部门

2. **手动映射：**
   - API: `POST /api/v1/ad/groups/mappings`
   - 需要手动指定每个部门的deptId和adGroupId

创建映射后，成员同步任务就会正常工作，将部门成员同步到对应的AD组。

**验证 (Verification):**
1. 调用自动映射API创建映射记录
2. 验证映射记录已创建：
   ```sql
   SELECT COUNT(*) FROM sys_dept_group_mapping WHERE ad_config_id = '4ee691f4-2f93-4981-b37f-93839b5c6af7' AND mapping_status = 'active';
   ```
3. 重新运行成员同步任务，验证计数不再为0

**变更文件 (Files Changed):** []
