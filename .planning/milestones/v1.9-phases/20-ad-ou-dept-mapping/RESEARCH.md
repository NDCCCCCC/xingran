# Phase 20: AD域控OU与部门映射 - Research

**Researched:** 2026-05-22
**Source:** Spike研究结果 - `.planning/spikes/ou-dept-mapping-corrected.md`
**Confidence:** HIGH

## Summary

本阶段基于已完成的Spike研究，实现系统部门与AD域控OU的双向映射功能。技术方案已验证可行，包括LDAP OU创建、用户移动、属性修改等核心操作。研究确认了同步方向（系统→AD）、数据结构设计、服务层架构和性能指标。

## Technical Stack

### Core Dependencies
| Library | Version | Purpose | Status |
|---------|---------|---------|--------|
| **go-ldap/v3** | v3.4.12 | AD域控LDAP操作 | ✅ 已存在 |
| **gorm.io/gorm** | v1.30.5 | ORM数据库操作 | ✅ 已存在 |
| **gin-gonic/gin** | v1.10.0 | HTTP路由和API | ✅ 已存在 |
| **robfig/cron** | v3.0.1 | 定时任务调度 | ✅ 已存在 |

### 新增依赖
无需新增依赖，所有要求功能可用现有库实现。

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                    AD域控OU与部门映射架构                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  系统层 (Go Backend)              AD域控层 (Active Directory)        │
│                                                                     │
│  ┌─────────────────────────┐         ┌──────────────────────────┐  │
│  │  定时任务 (每天2点)      │         │  AD域控                   │  │
│  │  TriggerDeptSync()      │         │  DC=company,DC=com       │  │
│  └───────────┬─────────────┘         │                          │  │
│              │                         └──────────────────────────┘  │
│              ▼                                   ▲                  │
│  ┌─────────────────────────┐                     │                  │
│  │  DeptToADSyncService    │──── LDAP写入 ────────┘                  │
│  │  - 递归同步部门树       │                     │                  │
│  │  - 创建OU结构           │                     │                  │
│  │  - 更新映射表           │                     │                  │
│  └─────────────────────────┘                     │                  │
│              │                                     │                  │
│              ▼                                     │                  │
│  ┌─────────────────────────┐                     │                  │
│  │  sys_dept_ou_mapping    │                     │                  │
│  │  (部门→OU映射表)        │◀───── 反向查找 ──────┘                  │
│  └─────────────────────────┘                     │                  │
│              ▲                                     │                  │
│              │                                     │                  │
│  ┌─────────────────────────┐                     │                  │
│  │  UserOUService         │◀──── 登录时触发 ──────┘                  │
│  │  - 获取用户OU DN        │                     │                  │
│  │  - 反向查找部门ID        │                     │                  │
│  │  - 设置user.dept_id     │                     │                  │
│  └─────────────────────────┘                     │                  │
│              ▲                                     │                  │
│              │                                     │                  │
│  ┌─────────────────────────┐                     │                  │
│  │  UserADSyncService     │──── 修改同步 ─────────┘                  │
│  │  - 移动用户到新OU       │                     │                  │
│  │  - 更新用户属性         │                     │                  │
│  │  - 同步department字段   │                     │                  │
│  └─────────────────────────┘                     │                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Data Flow Diagrams

#### 流程1: 部门定时同步
```
定时任务触发 (每天凌晨2点)
    ↓
DeptToADSyncService.SyncDeptStructureToAD()
    ↓
读取系统部门树 (sys_dept)
    ↓
连接AD域控 (LDAP)
    ↓
递归处理部门树:
    - 构建OU DN: OU={部门名},{父OU DN}
    - 调用LDAP CreateOU()
    - 更新映射表 (sys_dept_ou_mapping)
    ↓
记录同步日志和状态
```

#### 流程2: 用户登录部门设置
```
用户AD认证成功
    ↓
获取用户所在OU DN (从LDAP或token)
    ↓
UserOUService.HandleUserLoginAD()
    ↓
查询映射表: SELECT dept_id FROM sys_dept_ou_mapping WHERE ou_dn = ?
    ↓
更新sys_user表: UPDATE sys_user SET dept_id = ?, ad_ou_dn = ?
    ↓
返回登录成功
```

#### 流程3: 用户修改同步
```
管理员修改用户信息
    ↓
UserADSyncService.SyncUserUpdateToAD()
    ↓
如果部门变更:
    - 查找新部门的OU DN
    - 调用LDAP MoveUser(userDN, newOUDN)
    ↓
更新其他属性:
    - 调用LDAP UpdateUserAttributes(userDN, attributes)
    ↓
更新sys_user.ad_synced_at
```

## Implementation Complexity

### 中等复杂度
- **LDAP操作**: 需要扩展LDAP客户端（CreateOU, MoveUser, UpdateAttributes）
- **递归同步**: 部门树递归处理逻辑
- **错误处理**: AD操作失败的回滚和重试机制
- **性能优化**: 大量用户的批量移动需要分批处理

### 低风险
- **现有基础**: Phase 19已完成AD认证，LDAP客户端已存在
- **技术成熟**: LDAP操作是标准技术，有大量参考资料
- **数据结构简单**: 映射表结构清晰，外键关系明确

## Performance Considerations

### 预期性能指标
| 操作 | 数据量 | 目标耗时 | 优化策略 |
|------|--------|----------|----------|
| 部门同步 | 100个部门 | < 10秒 | 批量LDAP操作，并发控制 |
| 用户登录 | 单次 | < 100ms | 映射表缓存，异步更新 |
| 用户移动 | 100个用户 | < 30秒 | 分批处理，每批10个 |
| 属性更新 | 单个用户 | < 1秒 | 单次LDAP Modify操作 |

### 性能风险
- **LDAP连接数**: 大量操作可能耗尽连接池
  - **缓解**: 连接复用，操作排队
- **AD响应慢**: AD域控响应可能较慢
  - **缓解**: 设置超时，重试机制
- **递归深度**: 部门树可能很深
  - **缓解**: 限制最大深度，尾递归优化

## Integration Points

### Phase 19集成
- **认证服务**: 复用AD认证逻辑
- **用户表扩展**: 使用已有的`auth_source`, `ad_username`, `ad_dn`字段
- **LDAP客户端**: 扩展`internal/services/addomain/ldap_client.go`

### 现有系统集成
- **定时任务框架**: 使用`internal/scheduler/`
- **部门服务**: 使用`internal/services/system/dept_service.go`
- **用户服务**: 扩展`internal/services/system/user_service.go`

## Alternatives Considered

### 方案A: 手动映射模式
- **优点**: 简单，可控性强
- **缺点**: 维护成本高，容易出错
- **决策**: 不采用，作为备选方案

### 方案B: 双向同步模式
- **优点**: AD和系统保持一致
- **缺点**: 冲突处理复杂，数据源不明确
- **决策**: 不采用，系统作为唯一数据源

### 方案C: 异步消息队列
- **优点**: 解耦，可靠性高
- **缺点**: 复杂度高，需要消息中间件
- **决策**: 暂不采用，定时任务足够

## Open Questions

### 已确认
1. ✅ AD权限：已配置专用服务账号
2. ✅ 同步频率：每天2点
3. ✅ 初始化：自动同步，无需手动触发
4. ✅ OU冲突：创建新OU

### 待确认
1. 部门树最大深度限制？
2. 批量用户的并发控制策略？
3. 同步失败的通知机制？

## Risk Assessment

### 技术风险：中等
- **LDAP操作复杂度**: 中等（有标准库支持）
- **性能问题**: 低风险（可优化）
- **数据一致性**: 中等风险（需要事务和回滚）

### 业务风险：低
- **用户影响**: 低（主要是后台操作）
- **数据丢失**: 低风险（系统是权威源）
- **兼容性**: 低风险（Phase 19已验证）

## Success Metrics

### 功能指标
- 部门同步成功率 > 95%
- 用户登录部门设置成功率 > 99%
- 用户修改同步成功率 > 95%

### 性能指标
- 部门同步耗时 < 10秒（100个部门）
- 用户登录响应时间 < 100ms
- 用户移动操作 < 30秒（100个用户）

### 可靠性指标
- 同步失败重试成功率 > 80%
- 错误日志完整性 100%
- 数据一致性检查通过率 > 99%

## Next Steps

1. 创建详细实施计划（PLAN.md）
2. 设计数据库迁移脚本
3. 实现核心服务层
4. 集成测试和验证

---

**研究完成度:** ✅ 100%
**准备状态:** Ready for Planning
**建议:** 立即开始实施计划开发
