---
status: testing
phase: 59-ad-login-guard-reconciliation
source: [59-VERIFICATION.md]
started: 2026-10-20T00:45:00Z
updated: 2026-10-20T00:45:00Z
---

## Current Test

number: 1
name: 真实 AD 登录守卫拦截（GUARD-01 端到端）
expected: |
  管理员编辑用户 dept → 等队列入队（pending/failed）→ 该用户 AD 登录 → 日志出现『部门回写被活跃队列守卫拦截』WARN（含 userID/username/本地 dept UUID/AD 解析 dept UUID/队列行 ID/action_type/attempts），且 sys_user.dept_id 未被 AD 旧值覆盖
awaiting: user response

## Tests

### 1. 真实 AD 登录守卫拦截（GUARD-01 端到端）
expected: 管理员编辑用户 dept → 等队列入队（pending/failed）→ 该用户 AD 登录 → 日志出现『部门回写被活跃队列守卫拦截』WARN（含 userID/username/本地 dept UUID/AD 解析 dept UUID/队列行 ID/action_type/attempts），且 sys_user.dept_id 未被 AD 旧值覆盖
result: [pending]

### 2. 真实 PG 启动冒烟：migration_217 视图 + migration_218 权限 seed + 二次重启幂等
expected: 目标 PG 环境启动服务：AutoMigrate 前 DROP 视图、Migrate217 无错、SELECT count(*) FROM v_dept_ou_mismatch 可执行；再次重启同库无 SQLSTATE 0A000『cannot alter type』且视图重建成功；启动日志无『对账查询按钮权限 seed 失败』
result: [pending]

### 3. 对账 Tab 浏览器权限渲染（授权与未授权两态）
expected: 管理员为角色授予 ops:ad:config:reconciliation:list → 该角色登录后对账 Tab 出数据；未授权角色登录后 Tab 内渲染『无对账查看权限』403 且不发对账数据请求；直接 POST /api/v1/ad-domain/reconciliation/mismatch 被后端 RequirePermissions 拒为 403
result: [pending]

### 4. 生产库父菜单『AD域控配置』（menu_type='C'）存在性确认
expected: 生产 sys_menu 存在该父菜单 → migration_218 自动建 F 行并精准授权；若缺失 → 管理员需手建按钮权限 ops:ad:config:reconciliation:list 并为相关角色授权（迁移已按 warning + nil 降级，不会启动失败）
result: [pending]

### 5. GUARD-01/GUARD-02 验收口径人工复核（FLAGGED ASSUMPTION 闭环）
expected: 确认 ROADMAP SC1~SC4 原文即权威验收口径（四个 PLAN frontmatter 的 edge probe 均报 unclassified/unresolved，未假设 backstop）
result: [pending]

### 6. 空态标题字重 600（computed style）
expected: 浏览器打开 AD 配置页 → 对账 Tab → 将筛选调成 0 结果 → DevTools 选中空态标题，确认 computed font-weight: 600、font-size: 16px
result: [pending]

### 7. Tab 栏不溢出视觉回归（backstop）
expected: 两个 Tab 标签不换行/不截断
result: [pending]

## Summary

total: 7
passed: 0
issues: 0
pending: 7
skipped: 0
blocked: 0

## Gaps
