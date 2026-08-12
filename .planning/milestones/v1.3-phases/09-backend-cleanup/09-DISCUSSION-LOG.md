# Phase 9: 后端代码优化 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-27
**Phase:** 09-后端代码优化
**Areas discussed:** Phase 范围, 清理验证, 安全测试, 死代码范围, Core 优化

---

## Phase 范围

| Option | Description | Selected |
|--------|-------------|----------|
| 不包括迁移 | 只执行 Phase 1-3，保持服务文件在当前结构 | ✓ |
| 包括迁移 | 执行完整计划，迁移服务文件到子目录 | |
| 分阶段执行 | Phase 1-3 本阶段，迁移到未来阶段处理 | |

**User's choice:** 不包括迁移
**Notes:** ROADMAP 只定义了 3 个计划，迁移属于更大范围的重构，延后到未来阶段

---

## 清理验证

| Option | Description | Selected |
|--------|-------------|----------|
| 构建验证 | 删除后运行 go build，成功即可 | |
| 引用检查 | 运行 grep 确认无外部引用后再删除 | |
| 两者都要 | 构建 + grep 双重验证确保安全 | ✓ |

**User's choice:** 两者都要
**Notes:** 双重验证确保删除安全，第一关 grep 确认无外部引用，第二关 go build 验证

---

## 安全测试

| Option | Description | Selected |
|--------|-------------|----------|
| 单元测试 | 编写单元测试覆盖修复场景 | ✓ |
| 手动验证 | 手动测试修复效果，不写测试 | |
| 两者都不 | 优先手动验证，测试作为技术债 | |

**User's choice:** 单元测试
**Notes:** 需要编写单元测试验证 CheckOrigin、竞态条件、错误日志修复

---

## 死代码范围

| Option | Description | Selected |
|--------|-------------|----------|
| 仅已知文件 | 只删除已识别的 2 个文件 | |
| 全面扫描 | 系统性扫描其他可能的死代码 | ✓ |
| 扫描 services | 系统性扫描但限制在 services 目录 | |

**User's choice:** 全面扫描
**Notes:** 系统性扫描 services 目录寻找其他死代码文件，不限于已识别的 2 个

---

## Core 优化

| Option | Description | Selected |
|--------|-------------|----------|
| 一次性清理 | 全部 12 个字段一起删除 | |
| 渐进式 | 分多个小步骤，每步验证 | ✓ |
| 跳过迁移 | Core 字段全部清理，Phase 4 延后 | |

**User's choice:** 渐进式
**Notes:** 分多个小步骤，每步删除 2-3 个字段，每步验证后独立提交

---

## Claude's Discretion

- 死代码扫描的具体工具选择 (go list, grep, 或其他静态分析工具)
- Core 字段删除的具体分组方式 (按功能模块或依赖关系)
- 单元测试的覆盖率目标
- 错误日志的格式和级别

## Deferred Ideas

- 服务文件迁移到子目录 (Phase 4 of 原优化计划)
- Core 结构的更大规模重构
- 其他目录的死代码扫描 (非 services 目录)
