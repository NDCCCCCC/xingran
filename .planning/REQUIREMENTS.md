# Requirements — v1.21 AD 双向同步 bug 修复

> 方案来源：三份独立 subagent 审计（架构设计 / 代码现实核对 / 过度设计风险）后的修订方案，用户已拍板。
> 审计关键否决：时间戳仲裁（GORM 自动刷 updated_at 恒失效）、完整 Outbox（过度设计）、FieldRule 表驱动引擎、MV 对账。

## v1.21 Requirements

### 队列与推送可靠性 (QUEUE)
- [x] **QUEUE-01**: 管理员编辑用户（部门/昵称/邮箱/手机）后，变更在**业务事务内**写入 `ad_push_queue`，进程重启不丢失
- [x] **QUEUE-02**: 同一用户短时间内多次编辑时，队列**按实体合并**仅保留最新待推送状态，不丢弃后一次编辑
- [ ] **QUEUE-03**: 后台排水器（in-process cron，**不注册 sys_job**）按退避重试失败行，达上限转终态且可查询
- [ ] **QUEUE-04**: 排水执行 OU 移动采用 **compare-and-move**（闭包内重读最新值 + NormalizeDN 归一化比较），仅在需要时 MoveUser，重复排水幂等
- [ ] **QUEUE-05**: LDAP code 32（目标 DN 不存在）时队列行转 dead 终态并清空 `sys_user.ad_dn`，不进重试循环
- [x] **QUEUE-06**: sys 删除用户时入队 **disable AD 账号**动作，闭环"已删用户经 AD 登录复活"链路

### 登录回写守卫 (GUARD)
- [ ] **GUARD-01**: AD 登录写 dept 前，若该用户存在 pending/failed 的 dept 队列行则**跳过覆盖 + WARN**（本地编辑不被 AD 旧值回滚）
- [ ] **GUARD-02**: `v_dept_ou_mismatch` 普通 SQL VIEW + 只读端点，展示 dept↔OU 不一致及"映射指向软删部门"条目

### 软删恢复安全 (SAFE)
- [ ] **SAFE-01**: `catSoftDeleted` 恢复前校验 AD 当前 DN 与存量 `ad_dn` 一致，不一致拒绝恢复 + 安全审计日志
- [ ] **SAFE-02**: 登录路径与 batch-sync 共用分类逻辑与校验，行为一致

### 影子表完整性 (SHADOW)
- [ ] **SHADOW-01**: 全量同步末尾对 AD 端已消失的 `sys_ad_user/ou/computer` 软删除，复用两道闸门（LDAP 0 条拒绝 / 拟删 >50% 拒绝）
- [ ] **SHADOW-02**: `sys_ad_group_member` 重放改为**事务包裹**，消除空成员可见窗口
- [ ] **SHADOW-03**: 删除清理挂 `syncDataInternal`（复用 singleflight），不与全量同步并发

### 卫生与边界 (HYGIENE)
- [ ] **HYGIENE-01**: 推送路径按用户 `ad_dn` 归属解析 AD 配置（消灭 `First()` 歧义）；`getDefaultADConfigID` 双份合并为公共 helper
- [ ] **HYGIENE-02**: OU DN 拼接引入转义 helper（部门名特殊字符安全）
- [ ] **HYGIENE-03**: 死代码清理：user_ou_service.go 整文件、GetUserDeptByADOU、BatchMoveUsersToNewOU、dept_ou_mapper.go:31 等过期注释及其测试
- [ ] **HYGIENE-04**: 嵌套组 member DN 标记/过滤，不混入用户成员关系
- [ ] **HYGIENE-05**: 同名兄弟部门生成相同 OU DN 时检测并告警，不静默互相覆盖映射
- [x] **HYGIENE-06**: sqlite 测试补队列建表，测试套件全绿

## Future Requirements (deferred)

- **FUTURE-ADSYNC-01**: uSNChanged 增量同步（性能优化非正确性；AD 负载出现真实压力时再启动）
- **FUTURE-ADSYNC-02**: `sys_dept_ou_mapping` 等映射表真 FK 约束（NOT VALID 两步迁移；软删语义下收益低，后置）
- **FUTURE-ADSYNC-03**: 管理员"一键按 AD 校正 dept"对账修复动作（当前 observe-only）

## Out of Scope (explicit exclusions)

| 排除项 | 理由 |
|--------|------|
| 时间戳仲裁（updated_at vs ad_synced_at） | 三方审计一致否决：GORM 自动刷新 updated_at 使仲裁恒失效，行为反转 |
| 完整 Outbox（状态机/事务性队列全套） | 过度设计：单一调用点，outbox-lite 已覆盖全部丢失模式 |
| FieldRule 表驱动同步引擎 | 8 条字段映射不值得千行级引擎；代码内映射 struct 同等清晰 |
| 系统侧 profile 字段只读（"去 AD 改"UX） | 与部署现实相反（IT 在 AD 管组织、系统管业务），UX 代价大 |
| 消息队列（MQ）替代队列表 | 单机部署，引 MQ 多一个运维件无收益 |
| DB TRIGGER | 同步是多表有序编排需可观测，trigger 是错误的层（域特性判断） |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| QUEUE-01 | Phase 57 | Done (2026-10-17, Phase 57 verified) |
| QUEUE-02 | Phase 57 | Done (2026-10-17, Phase 57 verified) |
| QUEUE-03 | Phase 58 | Pending |
| QUEUE-04 | Phase 58 | Pending |
| QUEUE-05 | Phase 58 | Pending |
| QUEUE-06 | Phase 57 | Done (2026-10-17, Phase 57 verified) |
| GUARD-01 | Phase 59 | Pending |
| GUARD-02 | Phase 59 | Pending |
| SAFE-01 | Phase 60 | Pending |
| SAFE-02 | Phase 60 | Pending |
| SHADOW-01 | Phase 61 | Pending |
| SHADOW-02 | Phase 61 | Pending |
| SHADOW-03 | Phase 61 | Pending |
| HYGIENE-01 | Phase 58 | Pending |
| HYGIENE-02 | Phase 58 | Pending |
| HYGIENE-03 | Phase 62 | Pending |
| HYGIENE-04 | Phase 62 | Pending |
| HYGIENE-05 | Phase 62 | Pending |
| HYGIENE-06 | Phase 57 | Done (2026-10-17, Phase 57 verified) |

**Coverage**: 19/19 v1.21 requirements mapped (roadmap created 2026-10-16, no orphans)
