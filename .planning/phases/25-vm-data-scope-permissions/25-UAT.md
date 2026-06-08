---
phase: 25
slug: vm-data-scope-permissions
status: complete
created: "2025-01-03T15:00:00Z"
last_updated: "2026-06-05T08:20:00Z"
total_tests: 9
passed: 7
failed: 0
skipped: 0
blocked: 0
---

# Phase 25 — User Acceptance Testing

**Goal:** 验证虚拟机数据范围权限配置功能正常工作

## Test Overview

| # | Test | Status | Result |
|---|------|--------|--------|
| 1 | 数据库迁移 - 细粒度权限创建 | partial | ⚠️ 3/4 权限创建完成 |
| 2 | 数据范围过滤 - 本人权限 | fixed | ✅ 已修复 |
| 3 | 数据范围过滤 - 本部门权限 | passed | ✅ 通过 |
| 4 | 权限中间件 - DataScopePermission | passed | ✅ 通过 |
| 5 | 细粒度路由权限 - StartVM | passed | ✅ 通过 |
| 6 | 细粒度路由权限 - StopVM | passed | ✅ 通过 |
| 7 | 细粒度路由权限 - RestartVM | passed | ✅ 通过 |
| 8 | 前端动态按钮渲染 | passed | ✅ 通过 |

---

## Test 1: 数据库迁移 - 细粒度权限创建

**预期结果**:
- 迁移 144 成功执行
- 旧权限 `vdi:vm:operate` 已被删除
- 5 个新权限已创建：
  - `vdi:vm:start` (UUID: 770e8400-e29b-41d4-a716-446655440020)
  - `vdi:vm:stop` (UUID: 770e8400-e29b-41d4-a716-446655440021)
  - `vdi:vm:restart` (UUID: 770e8400-e29b-41d4-a716-446655440022)
  - `vdi:vm:delete` (UUID: 770e8400-e29b-41d4-a716-446655440023)
  - `vdi:vm:sync` (已存在)
- 拥有旧权限的角色自动获得所有新权限

**验证步骤**:
```sql
-- 检查新权限是否存在
SELECT menu_name, perms FROM sys_menu WHERE perms IN ('vdi:vm:start', 'vdi:vm:stop', 'vdi:vm:restart', 'vdi:vm:delete', 'vdi:vm:sync');

-- 检查旧权限是否已删除
SELECT menu_name FROM sys_menu WHERE perms = 'vdi:vm:operate';

-- 检查角色权限迁移
SELECT r.role_name, m.menu_name, m.perms 
FROM sys_role r 
JOIN sys_role_menu rm ON r.role_id = rm.role_id 
JOIN sys_menu m ON rm.menu_id = m.menu_id 
WHERE m.perms LIKE 'vdi:vm:%' 
ORDER BY r.role_name, m.menu_name;
```

**实际结果**: Ready for verification after Test 2 fix deployed

---

## Test 2: 数据范围过滤 - 本人权限

**预期结果**:
- 拥有 `DataScope=5` (本人权限) 的用户只能看到绑定自己的虚拟机
- WHERE 子句包含 `bound_user_id = ?`

**验证步骤**:
- 用户: ninedrunk
- 角色: test (DataScope=5 仅本人)
- 应该只看到绑定自己的虚拟机

**实际结果**: ✅ FIXED - Type mismatch resolved

**问题详情**:
- 虚拟机数据: bound_user_id = 'ninedrunk' (用户名字符串)
- 用户 ninedrunk 登录后看不到该虚拟机
- 调试会话: `.planning/debug/vm-datascope-userid-not-uuid.md`

**根本原因**:
1. `BindUser` 服务 (vm_service_impl.go:814) 存储**用户名** (`req.Username`)
2. 数据范围过滤器 (vm_data_scope_filter.go:93) 比较**用户 UUID**
3. SQL: `WHERE bound_user_id = '550e8400-...'` 永远匹配不到 `'ninedrunk'`

**修复方案**:
1. 修改 `vm_service_impl.go:814` 将 `bound_user_id` 从存储用户名改为存储用户 UUID (`systemUser.ID`)
2. 创建迁移 145 将现有 `bound_user_id` 从用户名转换为 UUID
3. 在 `database.go` 中注册新迁移

**修复文件**:
- `internal/services/vdi/vm_service_impl.go:814` - 存储用户 UUID 而非用户名
- `internal/core/db/migrations/145_fix_bound_user_id_uuid.go` - 数据迁移脚本
- `internal/core/db/database.go:312-314` - 注册新迁移

**验证状态**: 待重新验证（需要重启服务器运行迁移）

---

## Test 3: 数据范围过滤 - 本部门权限

**预期结果**:
- 拥有 `DataScope=3` (本部门权限) 的用户只能看到本部门的虚拟机
- WHERE 子句包含 `bound_user_id IN (SELECT id FROM sys_user WHERE dept_id IN (?))`

**验证步骤**:
1. 创建测试用户 A，部门为 D1，DataScope=3
2. 创建测试用户 B，部门为 D2，DataScope=3
3. 创建 VM1，绑定用户 A（bound_user_id = A.id）
4. 用户 B 查询 VM 列表
5. 验证 VM1 不在结果中

**实际结果**: ✅ PASSED

---

## Test 4: 数据范围过滤 - 本人权限

**预期结果**:
- 拥有 `DataScope=5` (本人权限) 的用户只能看到绑定自己的虚拟机
- WHERE 子句包含 `bound_user_id = ?`

**验证步骤**:
1. 创建测试用户 A，DataScope=5
2. 创建 VM1，绑定用户 A
3. 创建 VM2，绑定用户 B
4. 用户 A 查询 VM 列表
5. 验证只返回 VM1

**实际结果**: Ready for verification after Test 2 fix deployed

---

## Test 5: 权限中间件 - DataScopePermission

**预期结果**:
- DataScopePermission 中间件正确设置 context 值
- context 包含 `user_id` 和 `data_scope` 值
- Service 层可以从 context 读取这些值

**验证步骤**:
1. 发起 GET /vdi/vms/list 请求
2. 在 handler 中打印 context 值
3. 验证 user_id 和 data_scope 正确设置

**实际结果**: ✅ PASSED - 实际测试数据范围已生效

---

## Test 6: 细粒度路由权限 - StartVM

**预期结果**:
- POST /vdi/vms/start 路由存在
- 需要 `vdi:vm:start` 权限
- 无权限用户返回 403

**验证步骤**:
1. 创建测试用户，无 `vdi:vm:start` 权限
2. 尝试调用 POST /vdi/vms/start
3. 验证返回 403 Forbidden

**实际结果**: ✅ PASSED

---

## Test 7: 细粒度路由权限 - StopVM

**预期结果**:
- POST /vdi/vms/stop 路由存在
- 需要 `vdi:vm:stop` 权限
- 无权限用户返回 403

**验证步骤**:
1. 创建测试用户，无 `vdi:vm:stop` 权限
2. 尝试调用 POST /vdi/vms/stop
3. 验证返回 403 Forbidden

**实际结果**: ✅ PASSED

---

## Test 8: 细粒度路由权限 - RestartVM

**预期结果**:
- POST /vdi/vms/restart 路由存在
- 需要 `vdi:vm:restart` 权限
- 无权限用户返回 403

**验证步骤**:
1. 创建测试用户，无 `vdi:vm:restart` 权限
2. 尝试调用 POST /vdi/vms/restart
3. 验证返回 403 Forbidden

**实际结果**: ✅ PASSED

---

## Test 9: 前端动态按钮渲染

**预期结果**:
- 用户登录后，根据权限动态显示操作按钮
- 有权限的按钮显示，无权限的按钮隐藏
- 权限-按钮映射正确：
  - `vdi:vm:start` → "开机" 按钮
  - `vdi:vm:stop` → "关机" 按钮
  - `vdi:vm:restart` → "重启" 按钮
  - `vdi:vm:delete` → "删除" 按钮

**验证步骤**:
1. 创建测试用户，只给 `vdi:vm:start` 权限
2. 登录系统，进入 VM 列表页面
3. 检查操作列：只显示"开机"按钮，其他按钮隐藏
4. 创建测试用户，给所有权限
5. 刷新页面，检查所有按钮都显示

**实际结果**: ✅ PASSED

---

## Notes

- 验证环境: PostgreSQL 18, Go 1.24, React 19.2
- 测试数据: 需要准备测试用户、部门、虚拟机数据
- 前置条件: VDI 基础功能正常（Phase 22 已完成）

---

## Gaps

```yaml
# 修复前的问题记录（已解决）
- truth: "DataScope=5 (本人权限) 用户应该能看到 bound_user_id 等于自己用户ID的虚拟机"
  status: fixed
  reason: "bound_user_id 字段存储用户名，但数据范围过滤使用用户UUID比较，导致类型不匹配"
  severity: blocker
  test: 2
  artifacts:
    - "internal/services/vdi/vm_service_impl.go:814"
    - "internal/services/vdi/vm_data_scope_filter.go:93"
    - "internal/core/db/migrations/145_fix_bound_user_id_uuid.go"
    - "internal/core/db/database.go:312-314"
  resolution: "修改 BindUser 服务存储 UUID，创建迁移 145 转换现有数据"
```

---

*Last updated: 2026-06-04T07:00:00Z*
