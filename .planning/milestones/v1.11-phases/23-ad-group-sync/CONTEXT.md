# Phase 23: AD组自动同步系统

## Context
本功能建立在Phase 20 (AD域控OU与部门映射) 基础之上，实现用户组的自动化管理。

## Business Requirements
- 用户组管理页面默认显示"本部部门分组OU"下的所有组
- 每个二级部门（如"科技创新部"）对应一个AD组（命名规则：`cxhub-{部门名}`）
- 部门成员自动成为该组的成员
- 每个成员只能属于一个组（人员变动时自动移出旧组）
- 定时任务自动同步人员变动

## Technical Context
- 使用现有的LDAP客户端接口（`ldap_client.go`已实现组操作）
- 兼容现有的AD域控配置（`sys_ad_config`表）
- 组创建失败不应阻断同步流程
- 成员变动应该有日志记录

## Existing Components
- `internal/services/addomain/group.go`: GroupService（查询功能）
- `internal/services/addomain/ldap_client.go`: LDAP客户端（已实现`AddGroupMember`、`RemoveGroupMember`）
- `internal/scheduler/`: 定时任务引擎
- `internal/models/`: 数据模型（需要扩展）
