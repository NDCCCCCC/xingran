---
description: VDI配置迁移到参数管理
created: 2026-05-26T00:00:00Z
status: in-progress
---

# VDI配置迁移到参数管理

## 任务描述

将VDI服务器配置从静态配置文件（`configs/config.yaml`）迁移到参数管理页面（`sys_config`表），实现动态配置管理。

## 变更范围

### 后端变更
1. **配置文件修改**
   - 从 `configs/config.yaml` 删除 `vdi.servers` 配置段
   - 从 `internal/config/config.go` 删除 VDI 服务器配置结构

2. **数据库层**
   - 在 `sys_config` 表中添加VDI配置参数（如果不存在）
   - 参数格式：`vdi.server.{index}.{field}` (如 `vdi.server.0.endpoint`)

3. **服务层修改**
   - 修改 `internal/core/core.go` 中的VDI客户端初始化逻辑
   - 从数据库配置服务读取VDI配置，而非静态配置文件
   - 确保配置更新时能动态重新加载VDI客户端

4. **API层（可选）**
   - 考虑添加配置刷新API，触发VDI客户端重新初始化

### 前端变更（可选）
- 在参数管理页面添加VDI配置分组
- 提供VDI服务器配置的表单界面

## 执行计划

1. 读取当前配置文件结构和VDI配置定义
2. 修改配置结构体，移除VDI静态配置
3. 修改Core初始化逻辑，从数据库读取VDI配置
4. 删除配置文件中的VDI配置段
5. 测试配置读取和VDI客户端初始化
6. 原子提交所有变更

## 技术约束

- 保持向后兼容：如果数据库中没有VDI配置，系统应能正常启动（VDI功能不可用）
- 配置格式：使用现有的 `sys_config` 表结构，`config_key` 存储配置键，`config_value` 存储配置值
- 安全考虑：密码字段应加密存储

## 验证标准

- 配置文件中没有VDI配置
- VDI配置从数据库成功读取
- VDI客户端能正常初始化
- 虚拟机列表能正确显示（当VDI配置有效时）
