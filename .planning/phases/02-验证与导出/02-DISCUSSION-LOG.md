# Phase 2: 验证与导出 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-16
**Phase:** 02-验证与导出
**Areas discussed:** 导出列解析方式, 用户显示字段, 验证策略

---

## 导出列解析方式

| Option | Description | Selected |
|--------|-------------|----------|
| JOIN 查询 | 与 floorName/buildingName 一致，通过 JOIN 查询 sys_dept/sys_user 获取名称 | ✓ |
| 导入时写入 name | 在导入时同时写入 dept_name/user_name 到 workstation 表 | |

**User's choice:** JOIN 查询
**Notes:** 保持与 floorName/buildingName 的 JOIN 模式一致，导出数据实时准确，不依赖缓存字段

---

## 用户显示字段

| Option | Description | Selected |
|--------|-------------|----------|
| 昵称 (nickname) | 与列表 API 一致 | |
| 登录名 (username) | 便于 IT 系统对接 | |
| 两者都包含 | 导出同时包含昵称和登录名两列 | ✓ |

**User's choice:** 两者都包含
**Notes:** 昵称列标题"所属人员"（与列表一致），登录名列标题"所属人员账号"

---

## 验证策略

| Option | Description | Selected |
|--------|-------------|----------|
| 编译验证为主 | 修改 export config + go build 验证编译 | ✓ |
| 编译 + 单元测试 | 额外写 Go 单元测试验证 JOIN 解析 | |
| 端到端手动测试 | 手动运行后端通过 Swagger/curl 测试 | |

**User's choice:** 编译验证为主
**Notes:** 与 Phase 1 一致，改动范围小（仅 export config），风险低

---

## Claude's Discretion

- JOIN 配置的具体字段映射细节
- 移除旧配置后的列顺序调整
- 是否保留 deptId/userId（ID 列）在导出中的位置

## Deferred Ideas

(None)
