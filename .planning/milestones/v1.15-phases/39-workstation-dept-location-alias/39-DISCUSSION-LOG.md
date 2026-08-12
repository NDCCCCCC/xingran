# Phase 39: 工位部门物理位置映射 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-25
**Phase:** 39 - workstation-dept-location-alias
**Areas discussed:** 4 (D-01 节点展示, D-02 表单 UX, D-03 缓存失效, D-04 scope 字段)

---

## D-01: 映射节点展示方式

| Option | Description | Selected |
|--------|-------------|----------|
| 原名 + [映射] 后缀 | subDeptTree 里原名 + 拼接 "[映射]" 后缀，保留 trimTitleToLastSegment 后短名 | ✓ |
| 包在 Ant Design Tag 组件里 | TreeSelect 节点上使用 Ant Design Tag 包装标签，橙色背景 | |
| 原名 + 悬浮提示完整路径 | 部门下拉中点击映射节点后，右侧悬浮提示 Card 展示完整逻辑路径 | |

**User's choice:** 原名 + `[映射]` 后缀
**Notes:** 用户选择最简洁方案，不引入新组件类型，避免 ReactNode title 兼容性陷阱。仍走原 title 字符串逻辑，仅追加标识后缀。

---

## D-02: alias 表单 UX

| Option | Description | Selected |
|--------|-------------|----------|
| 两个独立 TreeSelect | Drawer 内两个独立 TreeSelect：dept + location，三级校验实时提示 | ✓ |
| 两阶段选择 (部门 → 位置) | 选择 location 后只展示该 location 下的子部门作为可选 dept | |
| 三件联动 (机构→部门) | 使用现有 DepartmentTreeSelect，加上"创建别名"选项 | |

**User's choice:** 两个独立 TreeSelect
**Notes:** 三级校验在 onChange 实时提示 + onSubmit 最终校验。前端 + 后端双重校验，不在 DB CHECK 约束里实现（DB 难表达 ancestor 关系）。删除操作二次确认 Modal。

---

## D-03: 缓存失效粒度

| Option | Description | Selected |
|--------|-------------|----------|
| 调用现有 dept cache 失效接口 | alias CRUD 后调现有 dept cache invalidation 接口，一次性失效全量 | ✓ |
| 细粒度失效 (按 location_id) | 只失效特定 location_id 的 subDeptTree 缓存 | |
| 不缓存 dept 下拉 | alias 数据是"恒冷数据"，查询本身走 dept_tree 缓存 | |

**User's choice:** 调用现有 dept cache 失效接口
**Notes:** 复用现有 dept_cache_service invalidation。前端 useInvalidateDept() 同步失效 alias 数据（同一 React Query family）。

---

## D-04: scope 字段设计

| Option | Description | Selected |
|--------|-------------|----------|
| 预留多 scope (默认 'workstation') | 预留未来扩展（building/floor/infopoint 同类问题可复用） | ✓ |
| 不预留，只存 dept_id + location_id | 本期不使用 scope | |
| 预留 scope 但现在冗余 | scope 字段存在但所有 CRUD 都传 'workstation' | |

**User's choice:** 预留多 scope
**Notes:** UNIQUE(dept_id, scope)，未来 building/floor/infopoint 同类问题可设置 scope='building'/'floor'/'infopoint' 直接复用。同一 dept 可在不同 scope 下映射到不同 location。

---

## Claude's Discretion

以下方面由实现者决定：
- migration scope 字段长度（推荐 VARCHAR(32)）
- alias 列表分页（推荐 pageSize=10）
- Drawer 宽度（推荐 600px）
- operlog 是否记录 dept_name/location_name（推荐记录便于审计）
- 三级校验中"祖先关系"SQL 实现（PG `ancestors LIKE '%X%'` 或 `string_to_array @> ARRAY[?]`）
- alias 删除策略（推荐软删除）
- operlog 使用 Record vs RecordWithBody（推荐 Record）

---

## Deferred Ideas

以下想法在 discuss 阶段被识别但不属于本 phase：

- **跨模块扩展（building/floor/infopoint）**：scope 字段已预留支持，未来新建 phase 可直接复用 `sys_dept_location_alias` 表
- **alias 批量导入**：Excel 批量导入 alias 映射
- **alias 历史版本**：保留映射变更轨迹（operlog 已满足基础审计）
- **alias 自动推荐**：基于用户部门历史工位数据自动建议
- **workstation org_id 历史映射数据回填工具**：批量改 dept_id 工具
</content>
</invoke>