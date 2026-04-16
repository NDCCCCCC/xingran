# Research: Pitfalls

## 1. 用户名字段混淆

**问题**: `sys_user` 表有两个名称字段：
- `user_name` — 登录用户名（唯一，稳定）
- `nickname` — 昵称（可能重复，用户可修改）

**风险**: 如果用 `nickname` 匹配，可能出现重复匹配或匹配不到。

**预防**: 使用 `user_name` 做匹配（`Reference: "sys_user.user_name"`），与用户要求一致。

## 2. dept_name 可能不唯一

**问题**: `sys_dept.dept_name` 在不同层级可能有同名部门（如多个"技术部"）。

**风险**: ReferenceResolver 匹配到第一个同名部门，可能不是用户期望的。

**预防**: 当前 `batchQueryIDs` 返回第一个匹配，可接受。如果后续需要精确匹配，可考虑添加 `DependsOn` 级联（类似楼层依赖楼宇的模式），但用户未要求，本期不做。

## 3. 类型转换问题

**问题**: `workstation_service.go:16` 中 JOIN 使用 `sys_dept.id::text = sys_workstation.dept_id`，说明 `dept_id` 是 varchar 类型存 UUID。

**风险**: 如果 reference_resolver 返回的 UUID 格式不一致，JOIN 可能匹配不到。

**预防**: `reference_resolver` 直接查数据库返回 `id`，格式一定一致。无需额外处理。

## 4. 导出模板可能需要同步更新

**问题**: 如果前端有下载导入模板功能，需要确保新模板包含"所属部门"和"所属用户"列头。

**风险**: 下载旧模板缺少新列头，用户不知道可以填。

**预防**: 检查模板下载逻辑，确保动态生成或静态模板已更新。

## 5. 已有数据的兼容性

**问题**: 现有工位数据 `dept_id`/`user_id` 可能为空。

**风险**: 无风险。JOIN 使用 LEFT JOIN，空值不会影响查询结果。

**预防**: 无需额外处理。
