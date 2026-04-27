# Requirements: v1.3 技术债清理

## Overview

**Milestone**: v1.3 技术债清理
**Goal**: 修复 SNMP panic 问题、优化后端代码结构、集成网络设备导出功能
**Scope**: 三个技术债类别 — SNMP 稳定性、代码质量、导出功能

---

## SNMP-01: SNMP Panic 修复

**Category**: 系统稳定性
**Priority**: P0 - 阻塞性问题

### Requirements

- [ ] **SNMP-01a**: 实现 panic 恢复包装器
  - 在所有 scrapligo 操作外添加 defer recover()
  - panic 发生时记录日志并返回错误而非崩溃
  - 保持服务可用性

- [ ] **SNMP-01b**: 实现并发安全机制
  - 使用 RWMutex 保护连接状态
  - 避免读写竞态条件
  - 支持 20+ 并发连接

- [ ] **SNMP-01c**: 实现连接就绪性验证
  - 添加 WaitForReady() 方法
  - 验证传输层已完全初始化
  - 超时后返回错误

### Success Criteria

1. SNMP Ping 操作不再导致应用崩溃
2. 高并发场景下连接池稳定
3. Panic 事件被正确记录和恢复

---

## CODE-02: 后端代码优化

**Category**: 代码质量
**Priority**: P1 - 安全隐患

### Requirements

- [ ] **CODE-02a**: 删除死代码文件
  - 删除 `internal/services/dashboard_service.go`
  - 删除 `internal/services/settings_service.go`
  - 验证 `go build ./...` 通过

- [ ] **CODE-02b**: 清理 Core 结构
  - 移除 12 个死字段（DeviceManager、NetworkDeviceService 等）
  - 提取 RPAScalingService 为正确接口
  - 简化 Init() 方法

- [ ] **CODE-02c**: 修复安全问题
  - 修复 WebSocket CheckOrigin 验证
  - 修复 GlobalDeviceMonitorService 竞态条件
  - 添加错误日志记录

### Success Criteria

1. 所有死代码已删除，构建通过
2. Core 结构清晰，依赖明确
3. 安全漏洞已修复

---

## EXPORT-03: 网络设备导出

**Category**: 功能集成
**Priority**: P2 - 功能增强

### Requirements

- [ ] **EXPORT-03a**: 注册导出路由
  - 在 network_router.go 中注册导出端点
  - 支持 9 种导出类型（设备、凭证、模板等）

- [ ] **EXPORT-03b**: 前端集成
  - 在设备列表页面添加导出按钮
  - 显示导出进度
  - 支持批量导出

- [ ] **EXPORT-03c**: 厂商特定格式
  - 支持 Cisco .cfg 格式
  - 支持 Huawei .zip 格式
  - 支持 H3C 兼容格式

### Success Criteria

1. 用户可以从 UI 导出设备配置
2. 支持主流厂商格式
3. 大数据集导出稳定

---

## Out of Scope

- 升级 scrapligo 版本（破坏性变更风险）
- 实时配置流式传输（过度工程）
- 多厂商配置标准化（保真度损失）
- 完整配置分析（独立关注点）

---

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| SNMP-01a | - | pending |
| SNMP-01b | - | pending |
| SNMP-01c | - | pending |
| CODE-02a | - | pending |
| CODE-02b | - | pending |
| CODE-02c | - | pending |
| EXPORT-03a | - | pending |
| EXPORT-03b | - | pending |
| EXPORT-03c | - | pending |

---
*Created: 2026-04-27*
