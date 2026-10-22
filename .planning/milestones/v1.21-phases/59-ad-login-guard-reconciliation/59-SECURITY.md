---
phase: "59"
slug: "ad-login-guard-reconciliation"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-10-20"
---

# Phase 59 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| AD 域控 → 登录同步路径 | 外部 LDAP 返回的 OU/DN 驱动本地 `sys_user.dept_id` 写入；活跃队列守卫只读不写 | OU/DN（非凭据） |
| 业务事务 / 管理员编辑 → ad_push_queue | 队列行是本地编辑意图的持久化信号，守卫仅读活跃谓词 | 队列状态行 |
| 应用 DB 账号 → 视图 DDL | 启动期执行静态 DDL，不接受任何外部输入 | 无用户输入 |
| 视图 → 业务表（sys_user / sys_dept / sys_ad_config / sys_dept_ou_mapping） | 只读引用；视图存在会阻塞 GORM ALTER TYPE | 只读查询 |
| 浏览器/客户端 → 对账端点 | 未信任的 JSON 请求体（current/pageSize/mismatchType/deptId/keyword） | 查询参数 |
| 角色授权 → 权限缓存 | `sys_role_menu`/`sys_menu.perms` 变化经 30s 权限缓存生效 | 权限矩阵 |
| 前端权限数组 → UI 渲染 | `permissions` 来自 `sys_menu.perms`，可被用户侧篡改但不构成鉴权绕过 | 展示层数据 |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-59-01 | Tampering | `user_sync_service.go` 守卫判定 | medium | mitigate | 直接引用 `models.ADPushQueueActivePredicate` 单一事实源（ad_push_queue.go:53）；`user_sync_service_phase59_test.go` 6 用例锁定 pending/failed 拦截与 pushed/dead/disable 放行 | closed |
| T-59-02 | Information Disclosure | 守卫 WARN 日志 | low | accept | 日志仅含 userID/username/dept UUID/队列行元数据（app.log 实测样本核实），不含密码、token 或 DN 凭据 | closed |
| T-59-03 | Tampering | 视图 DDL 与 body | medium | mitigate | DDL 与 SELECT body 为编译期常量（migration_217 `deptOUMismatchViewName` 字面量），无用户输入插值 | closed |
| T-59-04 | Denial of Service | 启动期 AutoMigrate 被视图阻塞 | high | mitigate | AutoMigrate 前定向 DROP（`dropDependentMaterializedViews` database.go:471 / `dropSQLiteReconciliationViews` database.go:840），migration_217 迁移后重建；三路径注册防 SKIP_AUTOMIGRATE 旁路漏建 | closed |
| T-59-05 | Elevation of Privilege | `/ad-domain/reconciliation/mismatch` | high | mitigate | 后端 `RequirePermissions(ops:ad:config:reconciliation:list)`（ad_domain_router.go:149）独立鉴权；migration_218 提供可授予的 F 型权限载体 | closed |
| T-59-06 | Tampering | 服务层 keyword/deptId/mismatchType | high | mitigate | GORM 参数绑定、`validMismatchType` 四值白名单、`escapeLike` + `LIKE ? ESCAPE '\'`（dept_ou_mismatch_service.go:56,139）；禁止字符串拼接用户输入 | closed |
| T-59-07 | Information Disclosure | 对账结果中的用户名 / DN | medium | mitigate | 端点位于 JWT 组 + RequirePermissions；返回字段限于 D-11 白名单；无匿名或公共入口 | closed |
| T-59-08 | Denial of Service | 未分页/超大 pageSize 的视图扫描 | medium | mitigate | `query.NormalizePagination` 上限 200（ad_reconciliation_handler.go:62）；确定性 ORDER BY；前端页大小选项上限 100 | closed |
| T-59-09 | Elevation of Privilege | 前端 Tab 权限门 | high | mitigate | 前端 403 仅 UX（ReconciliationTab.tsx:7 无权限不发请求）；后端 T-59-05 已用 RequirePermissions 独立强制，篡改前端权限数组不构成绕过 | closed |
| T-59-10 | Information Disclosure | 对账表格展示用户名 / OU DN | low | accept | 页面位于受权限保护的 AD 配置页；字段限于 D-11，无凭据/密码 | closed |
| T-59-SC | Tampering | npm/pip/cargo 依赖安装 | high | mitigate | 本 phase 全部提交零依赖清单变更（go.mod / go.sum / package.json / package-lock.json 无 diff），不引入供应链面 | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-59-01 | T-59-02 | 守卫 WARN 日志含 userID/username/UUID 元数据，属受控运维日志，无凭据泄露面；WARN 级别为守卫可观测性必需 | PLAN 59-01 disposition（用户未否决） | 2026-10-20 |
| AR-59-02 | T-59-10 | 对账表格展示用户名/OU DN 属业务数据本体；端点已受 JWT + 独立权限点保护，页面入口受权限门控 | PLAN 59-04 disposition（用户未否决） | 2026-10-20 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-20 | 11 | 11 | 0 | Claude（orchestrator，L1 grep 验证，asvs_level 1 短路规则） |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-20
