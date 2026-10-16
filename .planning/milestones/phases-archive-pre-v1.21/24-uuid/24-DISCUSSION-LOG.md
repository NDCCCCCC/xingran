# Phase 24: UUID类型一致性优化 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-27
**Phase:** 24-uuid
**Areas discussed:** UUID生成策略, Go类型安全, 外部UUID字段, 迁移规范

---

## UUID生成策略

| 选项 | 描述 | 已选择 |
|------|------|--------|
| 数据库侧生成 | 使用 gen_random_uuid() 作为默认值。优点：保证直接 SQL 插入也能生成 UUID。缺点：依赖 PostgreSQL 特定功能。 | |
| Go侧生成 | 统一使用 BeforeCreate 钩子生成。优点：跨数据库兼容，代码控制。缺点：直接 SQL 插入需要手动生成 UUID。 | ✅ |
| 混合策略 | 主键使用数据库侧生成，关联 ID 使用 Go 侧生成。优点：灵活控制。缺点：增加复杂度。 | |

**用户选择:** Go侧生成

**理由:**
- 跨数据库兼容（不依赖 PostgreSQL 特定功能）
- 代码控制 UUID 生成逻辑，便于测试和调试
- 与现有大部分模型模式一致

---

## Go类型安全

| 选项 | 描述 | 已选择 |
|------|------|--------|
| 保持 string 类型 | 继续使用 string 存储 UUID。优点：无需修改 API、前端代码。缺点：编译时无法验证 UUID 格式。 | ✅ |
| 迁移到强类型 UUID | 引入自定义 UUID 类型或使用 uuid.UUID。优点：编译时类型安全。缺点：需要修改大量代码，API 变更。 | |
| 分阶段迁移 | 新模型使用强类型 UUID，旧模型保持 string。优点：渐进式改进。缺点：代码库存在两种模式。 | |

**用户选择:** 保持 string 类型

**理由:**
- 向后兼容，无需修改 API 和前端代码
- JSON 序列化/反序列化简单
- 避免引入自定义 UUID 类型的复杂性

---

## 外部UUID字段处理 (VDI)

| 选项 | 描述 | 已选择 |
|------|------|--------|
| 保持 varchar(100) | 外部 UUID 继续使用 varchar(100)。优点：灵活处理各种 ID 格式。缺点：失去数据库级别的 UUID 验证。 | ✅ |
| 改用原生 UUID | 外部 UUID 也使用 UUID 类型。优点：类型一致性更好。缺点：需要验证外部数据符合 UUID 格式。 | |
| 双字段方案 | 使用 UUID 类型 + 单独字段存储源系统信息。优点：兼顾类型安全和追溯性。缺点：增加字段数量。 | |

**用户选择:** 保持 varchar(100) + 注释说明

**理由:**
- 灵活处理各种 ID 格式（UUID、字符串、数字等）
- 避免因格式验证导致的数据同步失败
- 与内部 UUID 字段明确区分

---

## 迁移脚本规范

| 选项 | 描述 | 已选择 |
|------|------|--------|
| 移除所有 DB 默认值 | 新建表时不使用 DEFAULT gen_random_uuid()。优点：策略统一，数据库无关。缺点：直接 SQL 插入需要手动生成 UUID。 | ✅ |
| 保留 DB 默认值 | 保留 DEFAULT gen_random_uuid() 作为降级方案。优点：数据库层保证数据完整性。缺点：两种生成方式并存。 | |
| 按字段类型区分 | 主键强制 DB 默认值，外键允许 Go 生成。优点：区分对待。缺点：规则复杂。 | |

**用户选择:** 移除所有 DB 默认值，统一 Go 生成

**理由:**
- 策略统一，减少混淆
- 数据库无关，便于未来迁移
- BeforeCreate 钩子保证所有 UUID 生成一致

---

## Claude's Discretion

以下方面由实现者决定：
- 测试文件组织
- 文档格式
- 代码注释详细程度
- 可选外键的默认值处理

---

## Deferred Ideas

- 大规模重构现有表 — 工作量大，收益不明显
- 引入强类型 UUID — 向后兼容性问题
- 外部 UUID 类型统一 — 依赖外部系统格式，风险高

---

*Phase: 24-uuid*
*Discussion log: 2026-05-27*
