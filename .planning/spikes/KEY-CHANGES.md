# 方案更正关键变更说明

## 🔄 方案方向变更

### 原方案（错误理解）
**同步方向:** AD域控 → 系统部门
**核心理念:** 从AD同步组织结构到系统，自动创建部门

### 更正方案（实际需求）
**同步方向:** 系统部门 → AD域控
**核心理念:** 系统是唯一数据源，单向同步到AD

---

## 📋 核心业务逻辑变更

### 变更1: 部门同步方式

**原方案:**
```
AD域控OU → 自动创建 → 系统部门
（从AD读取结构，在系统中镜像）
```

**更正方案:**
```
系统部门 → 定时同步 → AD域控OU
（系统是权威源，推送到AD）
```

**示例映射:**
```
系统部门树:                    AD OU树:
湖北分公司 (根)         →     OU=湖北分公司,DC=company,DC=com
└── 分公司本部           →     └── OU=分公司本部,OU=湖北分公司,DC=company,DC=com
    └── 科技创新部       →         └── OU=科技创新部,OU=分公司本部,OU=湖北分公司,DC=company,DC=com
```

### 变更2: 用户登录处理

**原方案:** 无明确方案

**更正方案:**
```
用户AD登录成功
    ↓
获取用户所在OU DN
    ↓
反向查找映射表 → 系统部门ID
    ↓
设置 user.dept_id
    ↓
记录 ad_user_dn, ad_ou_dn
```

**关键:** OU → 部门（反向查找）

### 变更3: 用户信息同步

**原方案:** 单向（AD → 系统）

**更正方案:** 双向
- **登录时:** AD属性 → 系统用户信息（包括部门）
- **修改时:** 系统用户信息 → AD域控（包括OU移动）

**示例:**
```
管理员在系统中将"张三"从"科技创新部"调到"财务部"
    ↓
查找"财务部"对应的OU DN
    ↓
调用LDAP: MoveUser(张三DN, 财务部OU_DN)
    ↓
更新AD属性: department=财务部
    ↓
完成同步
```

---

## 🗃️ 数据结构变更

### 新增映射表

**原方案:** `sys_ad_ou_dept_mapping` (OU → 部门)

**更正方案:** `sys_dept_ou_mapping` (部门 → OU)
```sql
CREATE TABLE sys_dept_ou_mapping (
    id UUID PRIMARY KEY,
    dept_id UUID NOT NULL,              -- 系统部门ID（主键）
    ad_config_id UUID NOT NULL,
    ou_dn VARCHAR(500) NOT NULL,        -- AD OU的DN
    ou_name VARCHAR(255) NOT NULL,      -- OU名称
    parent_ou_dn VARCHAR(500),          -- 父OU DN
    sync_enabled BOOLEAN DEFAULT true,
    sync_status VARCHAR(20)             -- pending/synced/failed
);
```

### 用户表扩展

**新增字段:**
```sql
ALTER TABLE sys_user ADD COLUMN ad_user_dn VARCHAR(500);   -- AD用户完整DN
ALTER TABLE sys_user ADD COLUMN ad_ou_dn VARCHAR(500);      -- 用户所在OU DN
ALTER TABLE sys_user ADD COLUMN ad_synced_at TIMESTAMP;     -- 最后AD同步时间
```

---

## 🔧 服务层变更

### 新增服务

| 服务 | 职责 | 方向 |
|------|------|------|
| `DeptToADSyncService` | 部门结构同步到AD OU | 系统 → AD |
| `UserOUService` | 用户登录时处理部门设置 | AD → 系统 |
| `UserADSyncService` | 用户信息修改同步到AD | 系统 → AD |
| `DeptOUmapper` | 部门-OU映射关系管理 | 双向查询 |

### LDAP客户端扩展

**新增操作:**
```go
// 创建OU
CreateOU(ouDN, ouName string) error

// 移动用户到新OU
MoveUser(userDN, newParentOUDN string) error

// 更新用户属性
UpdateUserAttributes(userDN string, attributes map[string]string) error

// 检查OU是否存在
OUExists(ouDN string) (bool, error)
```

---

## 📊 数据流变更

### 原方案数据流
```
AD域控 (数据源)
    ↓ LDAP读取
同步服务
    ↓
系统部门表 (镜像)
    ↓
用户 (自动关联)
```

### 更正方案数据流
```
定时任务 (触发)
    ↓
系统部门表 (数据源)
    ↓ LDAP写入
AD域控 (目标)
    ↑
    │ 用户登录时反向设置
    │
系统用户表
```

---

## ⏰ 时间线变更

### 原方案时间线
```
实施阶段:
1. AD OU同步 → 系统部门
2. 用户自动关联部门
```

### 更正方案时间线
```
实施阶段:
1. 定时任务: 系统部门 → AD OU (每天凌晨2点)
2. 用户登录: AD OU → 系统部门 (实时)
3. 用户修改: 系统 → AD (实时)
```

---

## 🎯 实施优先级变更

### P0 (必须实现)
1. ✅ 系统部门 → AD OU定时同步
2. ✅ 用户登录时部门设置
3. ✅ 用户修改时AD同步（含OU移动）

### P1 (重要)
1. 映射关系管理界面
2. 同步状态监控
3. 错误处理和重试

### P2 (可选)
1. 部门别名映射（处理命名不一致）
2. 批量用户操作
3. 同步日志分析

---

## ⚠️ 风险变更

### 新增风险

**1. AD权限要求提高**
- 原方案: 只需读取权限
- 更正方案: 需要创建/修改OU、移动用户的权限
- 缓解: 使用专用服务账号，详细权限配置

**2. 同步失败影响更大**
- 原方案: 同步失败只影响系统部门
- 更正方案: 同步失败可能导致AD与系统不一致
- 缓解: 详细日志、回滚机制、人工干预接口

**3. 性能要求提高**
- 原方案: 读操作，性能影响小
- 更正方案: 写操作（创建OU、移动用户），需要优化
- 缓解: 分批处理、异步任务、缓存

---

## 📝 文档变更

### 保留文档
- `ou-dept-mapping.md` - 原始方案（标记为Superseded）
- `ou-dept-mapping-experiments.md` - 技术实验（DN解析等仍然有效）

### 新增文档
- `ou-dept-mapping-corrected.md` - 更正后的完整方案
- `KEY-CHANGES.md` - 本变更说明文档

---

## ✅ 验收标准变更

### 原方案验收
- [ ] 可以手动创建OU-部门映射
- [ ] 支持按名称自动匹配
- [ ] 用户同步时自动关联部门

### 更正方案验收
- [ ] 定时任务正确同步部门树到AD OU
- [ ] 用户首次登录自动设置正确部门
- [ ] 修改用户部门时同步移动到新OU
- [ ] 修改用户属性时同步到AD
- [ ] 提供同步状态查询

---

**总结:** 这是一个**完全不同**的方案，从"AD主导"变为"系统主导"，从"单向读取"变为"双向同步"。技术复杂度提高，但更符合实际业务需求。

**建议:** 立即开始实施更正方案，预计7天完成。
