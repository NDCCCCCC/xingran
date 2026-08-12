# Phase 25: 虚拟机数据范围权限配置 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-02
**Phase:** 25-虚拟机数据范围权限配置
**Areas discussed:** 细粒度操作权限分解, 数据范围权限实现, 前端权限控制, API验证

---

## 细粒度操作权限分解

| Option | Description | Selected |
|---------|-------------|----------|
| 完全分解 | 将 vdi:vm:operate 分解为独立的细粒度权限，每个操作独立控制 | ✓ |
| 部分分解 | 保持 vdi:vm:operate 作为基础权限，仅添加 vdi:vm:delete 作为危险操作独立权限 | |
| 保持现状 | 保持 vdi:vm:operate 不变，通过配置或策略控制操作 | |

**User's choice:** 完全分解

**Follow-up - 原有权限处理：**
| Option | Description | Selected |
|---------|-------------|----------|
| 移除通用权限 | 移除原有的 vdi:vm:operate 权限，仅保留细粒度权限 | ✓ |
| 保留作为通配符 | 保留 vdi:vm:operate 作为「通配符」权限，拥有此权限的用户自动拥有所有细粒度权限 | |
| 你决定 | 根据最佳实践选择 | |

**User's choice:** 移除通用权限

**Follow-up - 权限列表：**
| Option | Description | Selected |
|---------|-------------|----------|
| 5个基础权限 | vdi:vm:start, vdi:vm:stop, vdi:vm:restart, vdi:vm:sync, vdi:vm:delete | |
| 增加终端和强制重启 | 在5个基础权限上增加 vdi:vm:console 和 vdi:vm:reboot | |
| 完整操作集 | 包含所有可能的操作权限 | |

**User's choice:** 5个基础权限的基础上添加绑定用户（仅用于数据范围权限，不操作vdi服务器）

**Notes:** 绑定用户不是操作权限，而是用于数据范围权限控制的标记。

---

## 数据范围权限实现

| Option | Description | Selected |
|---------|-------------|----------|
| 参考现有模式 | 在 VDI Service 层添加数据范围过滤逻辑，参考现有 system service 的实现模式 | ✓ |
| 创建新枚举 | 创建新的 VDI 特定数据范围枚举（如 VMPDataScope） | |
| 查询参数过滤 | 使用查询参数过滤（如 ?bound_user=true），前端传递参数控制 | |

**User's choice:** 参考现有模式

**Follow-up - 过滤规则（多选）：**
| Option | Description | Selected |
|---------|-------------|----------|
| 无绑定用户规则 | bound_user_id IS NULL 时，仅 DataScope=1 的角色可见 | ✓ |
| 本人可见规则 | bound_user_id = 当前用户ID 时，DataScope=5 可见 | ✓ |
| 部门可见规则 | bound_user_id 所在部门用户，DataScope=3 或 4 可见 | ✓ |
| 自定义部门规则 | DataScope=2 通过角色关联的部门表过滤 | ✓ |

**User's choice:** 全部选中

**Follow-up - 实现层面：**
| Option | Description | Selected |
|---------|-------------|----------|
| 修改现有服务 | 修改现有 VDI Service 查询方法，添加数据范围过滤逻辑 | |
| 创建辅助函数 | 创建新的数据范围过滤辅助函数，Service 调用此函数 | |
| 你决定 | 根据代码架构选择最佳方案 | ✓ |

**User's choice:** 你决定

**Decision:** 创建独立的数据范围过滤辅助函数 `ApplyDataScopeFilter()`

---

## 前端权限控制

| Option | Description | Selected |
|---------|-------------|----------|
| 现有Hook | 使用现有的 usePermission() Hook，检查每个按钮对应的权限 | ✓ |
| 专用Hook | 创建新的 VDI 专用权限 Hook（如 useVDIPermission()） | |
| 组件内检查 | 在虚拟机列表组件中直接检查权限，不使用 Hook | |

**User's choice:** 现有Hook

**Follow-up - 按钮控制：**
| Option | Description | Selected |
|---------|-------------|----------|
| 独立按钮检查 | 每个按钮独立检查权限 | |
| 动态生成按钮 | 根据用户拥有的权限集合动态生成按钮 | ✓ |
| 配置对象驱动 | 使用权限配置对象，统一管理按钮-权限映射 | |

**User's choice:** 动态生成按钮

**Follow-up - 权限数据来源：**
| Option | Description | Selected |
|---------|-------------|----------|
| 后端权限数组 | 使用后端返回的 permissions 数组 | |
| 前端映射表 | 前端维护权限-按钮映射表 | |
| 现有权限系统 | 使用现有的权限系统（authStore.permissions） | ✓ |

**User's choice:** 现有权限系统

---

## API 层面的数据范围权限验证

**用户要求：** 检查当前代码的实际情况

**代码分析结果：**
1. VM 路由 (`vm_router.go`) — 没有使用权限中间件，直接注册路由
2. VDI 服务器路由 (`router.go`) — 使用了 `RequirePermissions` 中间件
3. Handler 层 — 没有进行权限验证，直接调用 Service
4. 现有系统模块使用 `DataScopePermission` 中间件在路由组级别

| Option | Description | Selected |
|---------|-------------|----------|
| 路由权限+Service过滤 | 在路由层添加细粒度权限中间件，Service 层处理数据范围过滤 | ✓ |
| Service层统一处理 | 在 Service 层同时处理操作权限和数据范围权限 | |
| 中间件分离 | 创建新的数据范围权限中间件，与操作权限中间件配合使用 | |

**User's choice:** 你来决定，要求符合当前项目架构，使用最佳实践方案

**Decision:** 基于现有代码模式，采用完整的实现方案：
- 路由层中间件（DataScopePermission + RequirePermissions）
- Service 层过滤（ApplyDataScopeFilter 函数）
- 菜单迁移（新增细粒度权限）
- 前端权限（动态生成按钮）

---

## Claude's Discretion

以下方面由实现者决定：
1. 数据范围过滤辅助函数的具体位置
2. 权限-按钮映射表的存储方式
3. 错误消息的具体文案
4. 菜单迁移脚本的命名规范
5. 前端按钮组件的具体实现

---

## Deferred Ideas

无 — 讨论保持在 Phase 25 范围内，无超出范围的提议。

---

*Phase: 25-vm-data-scope-permissions*
*Discussion log generated: 2026-06-02*
