# Phase 8: SNMP Panic 修复 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-27
**Phase:** 08-snmp-panic
**Areas discussed:** SNMP 客户端保护, 日志记录增强, 测试验证

---

## SNMP 客户端保护

| Option | Description | Selected |
|--------|-------------|----------|
| 是，添加保护 | 在所有 SNMP 操作方法中添加 defer recover，保持与 scrapli_wrapper.go 一致的模式 | ✓ |
| 否，不需要 | gosnmp 库足够稳定，不会 panic；让 panic 直接暴露更便于发现问题 | |
| 仅关键方法 | 只在常用的 Get() 和 Walk() 中添加保护，其他方法保持不变 | |

**User's choice:** 你来决定 (Claude 决定)

**Notes:** Claude 选择添加保护，理由：
1. 与 `scrapli_wrapper.go` 保持一致的错误处理模式
2. 网络操作在高并发场景下应该有保护
3. 服务稳定性优于快速失败

---

## 日志记录增强

| Option | Description | Selected |
|--------|-------------|----------|
| 完整诊断 | 堆栈跟踪 + 设备信息 + 操作上下文 + 时间戳（推荐，便于问题排查） | ✓ |
| 基础日志 | 仅设备信息和 panic 值（减少日志量，但问题排查困难） | |
| 可配置级别 | 通过配置决定日志详细程度（debug 模式记录完整信息） | |

**User's choice:** 完整诊断

**Notes:** 包含信息：
- 堆栈跟踪 (使用 `debug.Stack()`)
- 设备标识 (IP 地址、设备名称、设备 ID)
- 操作上下文 (方法名、命令/参数)
- 时间戳

---

## 测试验证

| Option | Description | Selected |
|--------|-------------|----------|
| 单元测试 | 编写测试用例模拟 panic 场景，验证恢复逻辑（快速、可重复） | ✓ |
| 并发压力测试 | 模拟 20+ 并发连接，验证连接池稳定性（更接近真实场景） | |
| 两者结合 | 单元测试验证逻辑 + 并发测试验证稳定性（推荐） | |

**User's choice:** 单元测试

**Notes:** 理由：
- 快速、可重复
- 可以模拟各种 panic 场景
- 不依赖实际设备

---

## Claude's Discretion

以下区域用户让 Claude 决定：
- SNMP 客户端是否需要 panic 保护 → 添加保护
- 单元测试的具体实现方式 → Claude discretion

---

## Deferred Ideas

- 升级 scrapligo 版本（破坏性变更风险，超出本期范围）
- 实时配置流式传输（过度工程）
- 并发压力测试（可选择添加，非必需）
