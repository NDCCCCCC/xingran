# 计划：AD 域控用户同步问题全修复 + 通用经理同步方案

日期：2026-10-22
范围：RuoYi-Next 后端（Go）+ 前端（React）
工作目录：D:\CODE\ClaudeCode\ruoyi-go-backend

---

## 一、背景与用户目标

用户目标（verbatim）：「全部修复。确保字段同步，特别是部门联系人或者岗位同步到域控的经理字段」
追加（verbatim）：「我需要一套通用的但是能够覆盖以下需求的经理同步方案：
域控的经理字段同步于：1.部门负责人，2指定岗位的人（比如总经理），3.现状是属于分公司本部的，经理应该是部门负责人，非分公司本部的（如武汉中支），经理是机构负责人。」

澄清（verbatim）：「岗位负责人不是新增字段，是已存在的岗位模块」
→ 岗位角色一律复用**已存在的岗位模块**（`sys_post` + `user_posts` 关联表），**禁止**另造岗位表/岗位负责人字段体系。

设计选型（用户已确认）：
- 经理规则落地：**新建规则表 + 后台配置页**
- 岗位多人歧义：按用户后续澄清，改为「按机构 + 角色分层」推导（见 §四），不再单点取人
- 本部/机构判定：**规则表配置「机构集合 + 本部机构集合」，用户所属机构 = 祖先链上最近的命中机构**

---

## 二、必须修复的问题清单（验收标准）

| 编号 | 问题 | 落点 |
|------|------|------|
| P0-1 | hybrid 认证语义倒置：先 local 成功即返回 → AD 端禁用/离职在 hybrid 下全部失效 | `internal/core/security/hybrid_authenticator.go` |
| P0-2 | AD 来源账号本地密码恒为 "123456"，且全仓无登录时强制改密逻辑 | `internal/services/system/user_sync_service.go:214/317/782` |
| P1-3 | AD 禁用/删除不联动 `sys_user.status` → local/hybrid 下禁用无效 | `internal/services/addomain/shadow_cleanup_service.go`、`internal/services/addomain/service.go` |
| P1-4 | 批量同步路径缺 GUARD-01（直接覆盖 dept_id），与登录路径语义冲突 | `internal/services/system/user_sync_service.go:715/730` |
| P1-5 | 分类索引漏分类（username≠ad_username 脏数据撞唯一索引）+ 多域同名串味（无 ad_config_id） | `internal/services/system/user_sync_service.go:152-190` |
| P1-6 | mobile/title 从不写入 sys_user（模型无列 → 死字段） | `internal/models/user.go` + `user_sync_service.go` 四个写入函数 |
| P1-7 | 字段策略割裂：email/phone 覆盖本地、nickname 仅当空、mobile/title 永不更新 | `internal/services/system/user_sync_service.go:241-253` |
| P2-8 | 批量同步数据源是影子表快照而非 LDAP 实时 | `internal/api/v1/system/ad_domain_user_sync_handler.go:44`（改为快照+差异提示，保留性能） |
| P2-9 | AD 新建用户必须先等全量同步才能被勾选批量同步 | 同上（补「未同步到影子表」的 DN 提示） |
| P2-10 | 提前触发导致实际周期短于配置（3600→约 3300s） | `internal/scheduler/ad_sync_tasks.go:317` |
| P2-11 | sys_config 读取未过滤 deleted_at | `ad_domain_user_sync_handler.go:78`、`user_sync_service.go:422` |
| P2-12 | 计数口径：dept 更新失败仍记 Success；Total 是影子表命中数而非请求 DN 数 | `user_sync_service.go:709-742` |
| P2-13 | 分类常量声明顺序与注释编号跳序 | `user_sync_service.go:120-125` |
| P2-14 | 同步失败不更新 last_sync_at → 每 5 分钟重试一轮 | `internal/scheduler/ad_sync_tasks.go` |
| NEW | 通用经理同步方案（部门/岗位/机构分层） | 新增 `sys_manager_org` + `sys_manager_rule` + 规则引擎 |

### 已做对、不得破坏
- `internal/api/v1/auth.go:211-216`（AD 成功但 User==nil → 明确报错，不降级本地）
- `internal/api/v1/auth.go:225-237`（发 Token 前兜底查 sys_user.status）
- `user_sync_service.go:272-297` checkSoftDeletedRestore（统一入口 + DN 归一化 + 拒绝只 WARN 不返回 error）
- 影子清理双闸门（LDAP 0 条 / 拟删数*2>存量）
- 排水器 sys_config 每 tick 热重载 + 单例 TryLock
- `models.ADPushQueueActivePredicate`（`internal/models/ad_push_queue.go:53`）单一事实源，不得改写、不得加表名限定

---

## 三、数据模型变更（T1）

### 3.1 sys_user 补列
`internal/models/user.go` 新增：
```go
Mobile    *string `gorm:"size:20" json:"mobile,omitempty"`
Title     *string `gorm:"size:100" json:"title,omitempty"`
AdConfigID *string `gorm:"type:uuid;index" json:"adConfigId,omitempty"`
```

### 3.2 sys_ad_user 补列
`internal/models/ad_domain.go` 的 `ADUser` 新增：
```go
Manager string `gorm:"size:500" json:"manager,omitempty"` // 域控 manager 属性（DN）
```

### 3.3 新表 sys_manager_org（机构定义）
| 列 | 类型 | 说明 |
|----|------|------|
| id | uuid pk | |
| org_name | varchar(100) not null | 机构名称 |
| dept_id | uuid not null unique index | 指向 sys_dept 的机构节点 |
| parent_org_id | uuid nullable index | 上级机构（空=顶级） |
| org_type | int default 1 | 0=分公司本部 / 1=分支机构 |
| enabled | bool default true | |
| created_at/updated_at/deleted_at | | 软删 |

### 3.4 新表 sys_manager_rule（经理推导规则）
| 列 | 类型 | 说明 |
|----|------|------|
| id | uuid pk | |
| rule_name | varchar(100) not null | 规则名 |
| priority | int default 0 | 小者优先 |
| enabled | bool default true | |
| subject_type | varchar(32) | 适用主体：employee / dept_leader / org_head / post_holder |
| subject_post_id | uuid nullable | subject_type=post_holder 时的岗位 |
| subject_org_id | uuid nullable | 限定主体所属机构（空=不限） |
| manager_type | varchar(32) | 经理来源：dept_leader / post_holder / org_leader / org_head |
| manager_post_id | uuid nullable | manager_type=post_holder/org_head 时的目标岗位（如总经理、分管副总） |
| manager_org_id | varchar(64) nullable | 目标机构：空=用户自身机构；`parent`=上级机构；UUID=指定机构 |
| scope_type | int default 0 | 0=全部 / 1=部门子树 / 2=机构子树 |
| scope_id | uuid nullable | scope_type!=0 时的根 ID |
| created_at/updated_at/deleted_at | | 软删 |

### 3.5 迁移
- `internal/core/db/migrations/migration_219_user_ad_sync_fields.go`：sys_user 补 mobile/title/ad_config_id；sys_ad_user 补 manager
- `internal/core/db/migrations/migration_220_manager_rule_tables.go`：sys_manager_org + sys_manager_rule
- 写法参照 `internal/core/db/migrations/migration_218_ad_reconciliation_permission.go`：函数名 `Migrate219...`/`Migrate220...`，`log.Println("Running migration NNN: ...")`，非阻断降级用 `log.Printf("Warning: ...")`，结尾 `log.Println("Migration NNN completed: ...")`
- 新迁移需在 `internal/core/db/database.go` 的迁移注册链中调用（参照 migration_216/217/218 的既有注册位置）

---

## 四、经理同步通用方案（核心新功能，T5）

### 4.1 概念
- **机构（org）**：`sys_manager_org` 定义；「用户所属机构」= 用户部门 ancestors 链上**最近的**、命中 sys_manager_org 的部门节点
- **一把手（org_head）**：在指定「一把手岗位」（如"总经理"）任职、且其所属机构 == 某机构的那名用户
- **部门负责人（dept_leader）**：某部门的 `sys_dept.leader` 指向的用户
- **岗位任职人（post_holder）**：`user_posts` 关联的 sys_user

### 4.2 规则语义
对每个用户 U：
1. 算出 U 的「所属机构」org(U)（无机构 → 无规则可命中 → 跳过）
2. 按 priority 升序遍历启用规则 R：
   - **主体匹配**：
     - `employee`：U 不是任何 dept_leader / org_head
     - `dept_leader`：U 是某部门 leader（该部门在 R.scope 范围内时按 scope 判定）
     - `org_head`：U 是 R.subject_org_id（或用户自身机构）的一把手岗位任职人
     - `post_holder`：U 是 R.subject_post_id 的任职人
   - **范围匹配**：R.scope_type=1 → U 部门在 scope_id 子树；R.scope_type=2 → org(U) 在 scope_id 机构子树
   - 主体匹配且范围匹配 → 用 R 解析经理
3. **经理取值**：
   - `dept_leader`：U 所在部门的 leader（可回溯 ancestors，取最近的非空）
   - `post_holder`：R.manager_post_id 的任职人，按 R.manager_org_id 过滤机构；`ad_dn` 非空优先；多人 → 取按 id 升序第一个 + WARN
   - `org_leader`：org(U) 节点的 `sys_dept.leader`
   - `org_head`：R.manager_post_id（一把手岗位）任职人，机构按 R.manager_org_id 过滤（空=org(U)，`parent`=org(U) 的上级机构，UUID=指定机构）
4. 经理 = 目标用户的 `ad_dn`；目标用户无 ad_dn → 跳过 + WARN
5. 自指（经理 == U）→ 跳过 + WARN

### 4.3 覆盖用户描述的四种情形（配置示例，非硬编码）
设机构：O0=总公司（org_type=0 本部），O1=武汉中支（org_type=1，parent=O0）；岗位：P_gm=总经理（一把手岗），P_vp=分管副总

| 场景 | 规则 |
|------|------|
| 分公司本部普通员工 → 部门负责人 | subject=employee, manager=dept_leader, scope_type=2(scope=O0) |
| 分公司部门经理 → 分管总 | subject=dept_leader, manager=post_holder(P_vp), manager_org=自身机构 |
| 非分公司普通员工 → 机构一把手 | subject=employee, manager=org_head(P_gm), manager_org=自身机构 |
| 非分公司一把手 → 分公司总经理 | subject=org_head, manager=org_head(P_gm), manager_org=parent |
| 兜底 | subject=employee, manager=dept_leader（全部） |

### 4.4 同步执行与触发
- 复用现有 `internal/services/addomain/user_ad_sync_service.go` 的 LDAP 写入路径（FailoverClient + `UpdateUserAttribute(userDN, map[string]string{"manager": dn})` + errgroup 并发 + `constants.MaxConcurrentADSync`）
- 替换 `SyncManagersToAD` 的部门 leader 逻辑为规则引擎；保留 `POST /system/users/sync-managers` 手动全量入口
- 成功写入后回写影子表 `sys_ad_user.manager`
- 增量触发点：部门 leader 变更（`internal/services/system/department_service.go:109` Update 分支）、用户岗位变更（`internal/services/system/user_service.go:132/246`）、用户部门变更 → 入队 `sys_ad_push_queue` 新动作类型 `update_manager`；排水器 `drainRow` 增加该动作分派
- **不做**独立后台执行体（沿用 SHADOW-03 约束：清理/推送均挂在既有唯一执行路径）

---

## 五、任务分解与写作用域（Agent Team 并行）

| 任务 | 负责 | 写作用域（不重叠） |
|------|------|------------------|
| T1 | models + migration | `internal/models/user.go`、`internal/models/ad_domain.go`、`internal/models/manager*.go`(新)、`internal/core/db/migrations/migration_219*.go`、`migration_220*.go`、`internal/core/db/database.go` |
| T2 | AD→系统字段同步 | `internal/services/system/user_sync_service.go`（+ 其测试文件） |
| T3 | 认证语义 | `internal/core/security/hybrid_authenticator.go`、`local_authenticator.go`、`auth_strategy_factory.go`、`internal/api/v1/auth.go` |
| T4 | AD 禁用/删除联动 | `internal/services/addomain/shadow_cleanup_service.go`、`internal/services/addomain/service.go`、`internal/api/v1/system/ad_domain_handler.go` |
| T5 | 经理规则引擎 | `internal/services/addomain/manager_rule_service.go`(新)、`user_ad_sync_service.go`、`internal/api/v1/system/manager_rule_handler.go`(新)、`manager_rule_router.go`(新)、`internal/api/v1/system/user_router.go`(挂载 sync-managers) |
| T6 | 前端 | `xingran-react-frontend/src/**` |

依赖：T1 → T2/T5。T2/T3/T4/T5 可并行。T6 在 T2/T5 的 API 契约确定后进行。

---

## 六、全局约束

- Go：Handler-Service 模式；`response.Success/Error`；禁 `c.JSON`；context 透传；错误用 `fmt.Errorf("...: %w", err)`
- 状态值：0=正常/启用，1=停用（菜单可见性例外）
- operlog：所有业务写操作在 `response.Success` 前调用 `operlog.Record(...)`；敏感字段用 `operlog.RecordWithBody`
- 迁移：编号接续 218 → 219/220；幂等；非阻断降级用 log.Printf("Warning: ...")
- 改 Go 代码后必须 `go build ./...` 通过；改前端后 `npm run type-check` 通过
- 禁止 `as any`（如确需则加注释说明理由）
- Redis key 记得 `ruoyi:` 前缀
- 中文注释，标识符/路径/报错原文保留英文

## 七、验收
1. `go build ./...` 通过
2. `go test ./internal/services/system/ ./internal/services/addomain/ ./internal/core/security/ ./internal/api/v1/system/ ./internal/core/db/migrations/` 通过
3. `cd xingran-react-frontend && npm run type-check` 通过
4. 上述 P0/P1/P2 全部有对应代码改动 + 回归测试
5. 经理规则引擎有单元测试覆盖 §4.2 的主体匹配 / 范围匹配 / 经理取值 / 自指跳过 / 无 ad_dn 跳过
