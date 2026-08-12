# Phase 20: AD域控OU与部门映射 - Context

**Created:** 2026-05-22
**Dependencies:** Phase 19 (AD域控账号登录功能)
**Status:** Planning

## Phase Overview

本阶段实现系统部门与AD域控OU的双向映射功能，包括：
1. **部门定时同步到AD OU**（系统 → AD，保持层级结构）
2. **用户登录时自动设置部门**（AD OU → 系统，反向查找）
3. **用户信息修改同步到AD**（系统 → AD，包括OU移动）

## Business Context

**关键业务需求：**
- 系统是组织结构的唯一数据源
- 定时将系统部门树同步到AD域控OU结构
- 用户AD认证登录时，根据所在OU自动设置系统部门
- 管理员修改用户信息时，同步更新AD域控（包括部门变更时的OU移动）

**用户确认细节（来自DISCUSSION.md）：**
1. ✅ 有AD管理员权限且已配置
2. ✅ 每天2点同步合适
3. ✅ 不需要第一次手动触发
4. ✅ OU冲突处理：智能合并策略（路径匹配复用，否则创建新OU）
5. ✅ OU无映射处理：设置默认部门，确保用户有归属
6. ✅ 同步失败处理：仅记录日志，不阻断系统操作
7. ✅ 缓存策略：使用现有Redis缓存（5分钟TTL）

## Technical Context

**现有基础：**
- Phase 19已完成AD认证登录功能
- 用户表已有AD相关字段：`auth_source`, `ad_username`, `ad_dn`
- 已有完整AD域控管理模块：`internal/services/addomain/`
- 已有LDAP客户端封装：`internal/services/addomain/ldap_client.go`
- **已有完整Redis缓存系统**：`internal/pkg/cache/` + `internal/services/data_cache_service.go`
- **已有部门缓存键**：`dept:tree`, `dept:list`, `dept:id`, `dept:children`
- **已有缓存失效机制**：`DeleteByPattern()`, `InvalidateDeptCache()`

**数据模型：**
- 用户表：`sys_user` (已有`DeptID`, `AuthSource`, `ADUsername`, `ADDN`)
- 部门表：`sys_dept` (层级结构，支持`ParentID`, `Ancestors`)
- AD配置：`sys_ad_config` (已存在)

**需要新增：**
- 映射表：`sys_dept_ou_mapping` (部门ID → AD OU DN)
- LDAP扩展：CreateOU, MoveUser, UpdateUserAttributes, OUExists
- 同步服务：DeptToADSyncService, UserOUService, UserADSyncService
- **智能冲突解决器**：OUConflictResolver（路径匹配+冲突解决）
- **默认部门分配器**：DefaultDeptAssigner（处理无映射情况）
- **异步同步服务**：AsyncSyncService（重试队列+状态追踪）
- **OU-OU映射缓存**：复用现有DataCacheService，使用键模式 `dept_ou_mapping:{config_id}:{ou_dn}`
- **缓存集成**：部门同步完成时失效 `dept:*` 模式缓存

## Spike Research Results

**已完成Spike研究：** `.planning/spikes/ou-dept-mapping-corrected.md`

**关键发现：**
- DN解析和层级处理已验证可行
- LDAP OU创建、用户移动操作技术成熟
- 双向同步架构已设计完成
- **项目已有完善的Redis缓存系统，可直接复用**
- **预计工作量：8.5天**（增加智能合并、异步同步等组件）

## Success Criteria

1. ✅ 定时任务能正确同步部门树到AD OU（保持层级）
2. ✅ 用户首次登录时自动设置正确的部门（通过OU反向查找）
3. ✅ 修改用户部门时同步移动用户到新OU
4. ✅ 修改用户属性时同步到AD
5. ✅ 提供同步状态查询和手动触发接口
6. ✅ 完整的错误处理和日志记录

## Constraints

- 必须保持系统部门与AD OU的层级一致性
- 同步失败不能影响现有功能
- 需要处理大量用户的批量移动（分批处理）
- AD权限要求：创建OU、移动用户、修改属性

## Risk Mitigation

**技术风险：**
- AD权限问题 → 使用专用服务账号（✅已配置）
- OU命名冲突 → 智能合并策略（路径匹配+后缀创建）
- 批量操作性能 → 分批处理，控制并发
- **缓存依赖** → Redis不可用时降级到数据库查询
- **异步同步复杂度** → 重试队列+状态追踪机制

**业务风险：**
- 同步失败影响 → 详细日志，回滚机制
- 部门结构差异 → 手动映射覆盖功能
- 用户登录性能 → 使用缓存存储映射关系

## Out of Scope

- 用户多部门归属（P2功能）
- 部门别名映射（P2功能）
- 实时部门变更推送（定时任务即可）
- AD域控到系统的组织结构同步（系统是权威源）
