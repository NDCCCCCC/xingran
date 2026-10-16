---
phase: 22a-vdi-basic-integration
name: VDI 基础集成
milestone: v1.11
status: planned
created: 2026-05-25
depends_on: []
split_from: 22-sangfor-vdi-integration
waves: [22-01, 22-02, 22-03, 22-04, 22-05]
estimated_hours: 12-15
---

# Phase 22A: VDI 基础集成

## 目标

完成深信服 VDI 平台的基础集成，实现虚拟机的完整生命周期管理。

## 包含的 Waves

- **Wave 1 (22-01)**: 数据模型与配置基础
- **Wave 2 (22-02)**: VDI API 客户端与认证
- **Wave 3 (22-03)**: VDI 服务层实现
- **Wave 4 (22-04)**: VDI 后端 API 层
- **Wave 5 (22-05)**: VDI 前端 UI 实现

## 依赖关系

- 无外部依赖（独立功能）

## 交付物

### 数据层
- 4 个数据表（sys_vdi_vm, sys_vdi_server, sys_vdi_resource_group, sys_vdi_user_binding）
- 数据库迁移脚本
- 数据模型定义

### 后端层
- VDI API 客户端（认证 + 所有 VDI API 调用）
- 完整的服务层实现（VMService, ServerService, ResourceGroupService）
- 17 个后端 API 端点

### 前端层
- 虚拟机管理页面
- VDI 服务器配置页面
- API 客户端封装（vdiApi）

## 成功标准

当以下所有条件为 TRUE 时，Phase 22A 视为完成：

1. ✅ 可配置多个 VDI 服务器
2. ✅ 可从 VDI 同步虚拟机列表
3. ✅ 可对虚拟机进行开关机、重启、删除操作
4. ✅ 可配置虚拟机 IP 地址
5. ✅ 可绑定/解绑用户
6. ✅ 前端可管理所有 VDI 资源
7. ✅ 所有端点需要认证和权限验证
8. ✅ API 响应格式统一使用 response.Success/Error
9. ✅ 密码使用 AES-128-GCM 加密存储
10. ✅ 前端 API 调用使用封装的 vdiApi

## 执行命令

```bash
# 执行整个 Phase 22A
/gsd-execute-phase 22a-vdi-basic-integration

# 或按 Wave 顺序执行
/gsd-execute-phase 22a-vdi-basic-integration --wave 1
/gsd-execute-phase 22a-vdi-basic-integration --wave 2
# ... 依次执行到 Wave 5
```

## 预估工作量

12-15 小时（2 个工作日）

## 注意事项

- 这是 Phase 22 的第一阶段，完成后可以独立使用
- 为后续的 Agent 服务（Phase 22B）奠定基础
- 需要深信服 VDI 服务器的访问权限和 API 文档
