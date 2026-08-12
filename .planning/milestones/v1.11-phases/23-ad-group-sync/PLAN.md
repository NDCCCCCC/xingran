# Phase 23: AD组自动同步系统

## Goal
实现AD组自动同步功能，将系统部门与AD组建立映射关系，实现部门成员自动成为对应组的成员，并确保每个成员只能属于一个组。

## Context
本功能建立在Phase 20 (AD域控OU与部门映射) 基础之上，实现用户组的自动化管理。

**业务规则：**
- 每个二级部门（如"科技创新部"）对应一个AD组，命名规则：`cxhub-{部门名}`
- "本部部门分组OU"：指定OU路径，用于存储这些部门对应的组
- 部门成员自动成为该组的成员
- 每个成员只能属于一个组（人员部门变动时自动移出旧组）
- 定时任务自动同步人员变动

## Constraints
- 必须使用现有的LDAP客户端接口（`ldap_client.go`已实现组操作）
- 必须兼容现有的AD域控配置（`sys_ad_config`表）
- 组创建失败不应阻断同步流程
- 成员变动应该有日志记录

## Success Criteria
1. ✅ 可以通过配置指定"本部部门分组OU"
2. ✅ 系统能自动为二级部门创建对应的AD组
3. ✅ 部门成员自动成为对应组的成员
4. ✅ 人员部门变动时自动更新组关系（移出旧组、加入新组）
5. ✅ 提供API手动触发同步和查询同步状态
6. ✅ 定时任务自动运行同步逻辑

## Implementation Plans

### 23-01: 数据模型设计
**Description:** 创建部门-组映射表和同步日志表

**Tasks:**
- 设计数据库表结构（`sys_dept_group_mapping`）
  - `id`: UUID主键
  - `dept_id`: 部门ID（外键到sys_dept）
  - `ad_config_id`: AD配置ID（外键到sys_ad_config）
  - `group_dn`: AD组DN
  - `group_name`: 组名称（cxhub-{部门名}）
  - `member_ou_dn`: 成员OU DN（"本部部门分组OU"）
  - `sync_enabled`: 是否启用同步
  - `last_sync_at`: 最后同步时间
  - `sync_status`: 同步状态（pending/synced/failed）
- 设计同步日志表（`sys_ad_group_sync_log`）
  - `id`: UUID主键
  - `ad_config_id`: AD配置ID
  - `sync_type`: 同步类型（full/incremental）
  - `start_time`: 开始时间
  - `end_time`: 结束时间
  - `total_groups`: 总组数
  - `total_members`: 总成员数
  - `success_count`: 成功数
  - `failed_count`: 失败数
  - `error_message`: 错误信息
  - `created_at`: 创建时间

**Validation:** 检查外键约束、索引设计、是否需要迁移脚本

**Files:**
- `internal/models/dept_group_mapping.go`: 新建
- `internal/models/ad_group_sync_log.go`: 新建
- `internal/core/db/migrations/127_create_dept_group_mapping_table.sql`: 新建
- `internal/core/db/migrations/128_create_ad_group_sync_log_table.sql`: 新建

---

### 23-02: AD组管理服务
**Description:** 实现AD组的创建、成员管理功能

**Tasks:**
- 创建`GroupSyncService`服务
  - `CreateGroup(ctx, deptID, groupOU)`: 创建AD组
  - `DeleteGroup(ctx, groupDN)`: 删除AD组
  - `AddGroupMembers(ctx, groupDN, userDNs)`: 批量添加成员
  - `RemoveGroupMembers(ctx, groupDN, userDNs)`: 批量移除成员
  - `GetGroupMembers(ctx, groupDN)`: 查询组成员
- 实现组命名规则：`cxhub-{部门名}`
- 实现错误处理和重试逻辑

**Validation:** 单元测试组创建、成员添加/移除功能

**Files:**
- `internal/services/addomain/group_sync_service.go`: 新建
- `internal/services/addomain/group_sync_service_test.go`: 新建

---

### 23-03: 部门-组映射服务
**Description:** 管理部门与AD组的映射关系

**Tasks:**
- 创建`DeptGroupMappingService`服务
  - `CreateMapping(ctx, deptID, groupOU)`: 创建映射
  - `GetMappingByDept(ctx, deptID)`: 查询部门的映射
  - `ListMappings(ctx, adConfigID)`: 列出所有映射
  - `DeleteMapping(ctx, mappingID)`: 删除映射
- 实现自动映射逻辑：查询二级部门，自动生成组名
- 实现映射关系缓存优化

**Validation:** 集成测试，验证映射创建和查询

**Files:**
- `internal/services/addomain/dept_group_mapping_service.go`: 新建
- `internal/services/addomain/dept_group_mapping_service_test.go`: 新建

---

### 23-04: 成员同步逻辑
**Description:** 实现部门成员到AD组的同步功能

**Tasks:**
- 实现成员同步核心逻辑：
  1. 查询部门的直接成员（不包括子部门成员）
  2. 查询成员当前所属的组
  3. 比较部门与当前组的映射关系
  4. 如果不一致：移出旧组、加入新组
  5. 记录同步日志
- 创建`MemberSyncService`服务
  - `SyncDeptMembers(ctx, deptID, groupDN)`: 同步部门成员
  - `SyncAllMembers(ctx, adConfigID)`: 同步所有成员
  - `GetMemberChanges(ctx, deptID)`: 查询成员变动
- 实现排他性逻辑：每个成员只能属于一个组

**Validation:** 测试人员部门变动时的组同步

**Files:**
- `internal/services/addomain/member_sync_service.go`: 新建
- `internal/services/addomain/member_sync_service_test.go`: 新建

---

### 23-05: 定时同步任务
**Description:** 创建定时任务自动同步成员组信息

**Tasks:**
- 创建定时任务调度器
  - Cron表达式：每小时执行一次（可配置）
  - 任务名称：`ad_group_member_sync`
- 实现`GroupSyncScheduler`服务
  - `RegisterScheduleJob(ctx)`: 注册定时任务
  - `ExecuteSync(ctx)`: 执行同步逻辑
  - `HandleError(ctx, error)`: 错误处理
- 集成到现有的`scheduler`包
- 添加任务执行日志

**Validation:** 手动触发定时任务，验证同步逻辑

**Files:**
- `internal/scheduler/group_sync_scheduler.go`: 新建
- `internal/core/core.go`: 修改（注册调度器）

---

### 23-06: API接口设计
**Description:** 提供手动触发同步和查询状态的API

**Tasks:**
- 创建Handler和Router
  - `POST /api/v1/ad/groups/sync`: 手动触发同步
  - `GET /api/v1/ad/groups/sync/status`: 查询同步状态
  - `GET /api/v1/ad/groups/mappings`: 查询部门-组映射
  - `POST /api/v1/ad/groups/mappings`: 创建映射
  - `DELETE /api/v1/ad/groups/mappings/:id`: 删除映射
- 实现请求参数验证
- 实现响应格式统一

**Validation:** 使用Postman/curl测试API

**Files:**
- `internal/api/v1/addomain/group_sync_handler.go`: 新建
- `internal/api/v1/addomain/group_sync_router.go`: 新建
- `internal/api/router.go`: 修改（注册路由）

---

### 23-07: 配置管理
**Description:** 在系统参数配置中添加组同步相关配置

**Tasks:**
- 添加系统参数：
  - `sys.ad.group.sync.enabled`: 是否启用组同步（true/false）
  - `sys.ad.group.sync.cron`: 同步Cron表达式（默认：`0 0 * * * *`）
  - `sys.ad.group.member_ou`: "本部部门分组OU"路径
  - `sys.ad.group.auto_create`: 是否自动创建组（true/false）
- 读取配置参数的逻辑
- 提供配置修改API

**Validation:** 测试配置参数的读取和修改

**Files:**
- `internal/services/addomain/group_config_service.go`: 新建
- 前端配置页面：修改（添加组同步配置项）

---

## Integration Points

1. **Phase 20 (AD域控OU与部门映射)**: 使用已存在的`sys_dept_ou_mapping`表，获取部门-OU关系
2. **internal/scheduler**: 注册定时任务到调度器
3. **internal/services/addomain**: 复用LDAP客户端、AD配置等服务
4. **sys_config**: 存储组同步相关配置参数

## Estimated Effort
- 数据模型设计: 2-3h
- AD组管理服务: 3-4h
- 部门-组映射服务: 2-3h
- 成员同步逻辑: 4-5h
- 定时同步任务: 2-3h
- API接口设计: 2-3h
- 配置管理: 1-2h
- **Total: 16-23h**

## Dependencies
- ✅ Phase 20: AD域控OU与部门映射 (已完成)
- ✅ internal/scheduler: 定时任务引擎 (已存在)
- ✅ internal/services/addomain: AD域控服务 (已存在)

## Risk Mitigation
1. **LDAP权限问题**: 组创建/成员管理需要足够的LDAP权限
2. **性能问题**: 大量成员同步可能耗时较长，需要分批处理
3. **数据一致性**: 并发同步可能导致数据冲突，需要加锁或事务处理
