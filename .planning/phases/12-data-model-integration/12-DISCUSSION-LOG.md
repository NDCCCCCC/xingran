# Phase 12: 数据模型与采集集成 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-09
**Phase:** 12-数据模型与采集集成
**Areas discussed:** 变更检测策略, 历史表架构, 集成方式, 清理机制

---

## 变更检测策略

### 旧状态获取方式

| Option | Description | Selected |
|--------|-------------|----------|
| 删除前先查询 | 在删除前先查询当前 MAC 地址状态，保存到内存中，然后与采集结果对比 | ✓ |
| 先插入后删除 | 先批量插入新数据，再查询差异数据，最后删除不匹配的旧数据 | |
| 使用临时表 | 创建临时表存储新采集数据，使用 SQL JOIN 对比找出变更 | |

**User's choice:** 删除前先查询

**Notes:** 用户希望实现简单，无需修改表结构。虽然需要 3 次数据库操作，但单次采集本身就是批量操作，这个开销可以接受。

### 变更类型识别方式

| Option | Description | Selected |
|--------|-------------|----------|
| 状态转换矩阵 | 定义新旧状态转换规则：appeared/disappeared/moved/vlan_changed | ✓ |
| 简化检测（3类型） | 只检测 appeared、disappeared、moved，不区分 vlan_changed | |
| 使用事件标记 | 在历史表中添加 `change_reason` 字段，由采集逻辑标记变更原因 | |

**User's choice:** 状态转换矩阵

**Notes:** 用户需要完整的 4 种变更类型，逻辑清晰，易于理解。

### MAC Flapping 防护

| Option | Description | Selected |
|--------|-------------|----------|
| 时间窗口去重 | 在 5-15 分钟内，同一 MAC 地址在同一端口的重复 moved 事件只记录一次 | |
| 记录所有事件 | 不进行去重，记录所有变更事件，让上层应用处理 flapping 过滤 | |
| 智能合并 | 在历史表中使用 `first_seen` 和 `last_seen` 时间戳，自动合并连续的相同状态记录 | ✓ |

**User's choice:** 智能合并

**Notes:** 用户希望减少记录数，同时保留有效时间范围。这需要在历史表中添加额外的时间戳字段。

---

## 历史表架构

### 表结构设计

| Option | Description | Selected |
|--------|-------------|----------|
| 复制+扩展 | 复制 `sys_device_mac_address` 的所有字段，添加 `event_type`、`device_name_snapshot`、`first_seen`、`last_seen` | |
| 事件表模式 | 设计为纯事件日志表，不存储 first_seen/last_seen，由查询层聚合计算 | |
| 混合模式 | 历史表存储状态快照，每次变更插入新记录，不更新旧记录。通过 `first_seen` 和 `last_seen` 标识有效期 | ✓ |

**User's choice:** 混合模式（由 Claude 推荐）

**Notes:** Claude 推荐混合模式，因为与智能合并的 flapping 防护一致，且能高效支持停留时长计算（Phase 13 需求）。

### 分区创建方式

**用户澄清问题:** 不理解为什么需要分区

**解释:** 
- 没有分区：所有数据混在一起，查询要扫描全部
- 有分区：数据按月份分开，查询只打开相关分区
- 类比：没分区=大衣柜翻遍找衬衫；有分区=分格子直接去对应格子拿

**User's choice:** 自动创建

**Notes:** 用户理解了分区的价值后，选择应用层自动创建，避免手动维护的麻烦。

---

## 集成方式

### 集成时机

| Option | Description | Selected |
|--------|-------------|----------|
| 删除前对比 | 在执行 DELETE 之前，先查询当前状态，与采集结果对比，将变更写入历史表，然后删除旧数据，最后插入新数据 | ✓ |
| 插入后对比 | 先删除旧数据并插入新数据，然后与上次采集结果（需要缓存）对比生成历史记录 | |
| 使用数据库触发器 | 在数据库层创建触发器，DELETE/INSERT 时自动写入历史表 | |

**User's choice:** 删除前对比（由 Claude 推荐）

**Notes:** Claude 推荐删除前对比，因为保证数据一致性，符合"最小侵入"原则。

### 失败处理策略

| Option | Description | Selected |
|--------|-------------|----------|
| 非阻塞模式 | 历史记录失败只记录错误日志，不影响当前 MAC 地址的采集和存储 | ✓ |
| 事务回滚 | 历史记录失败时，整个采集事务回滚，不更新当前 MAC 地址 | |
| 降级模式 | 历史记录失败时，写入失败队列或文件，后续异步重试 | |

**User's choice:** 非阻塞模式

**Notes:** 用户希望保证采集流程稳定性，历史问题不应影响当前状态。

---

## 清理机制

### 清理方式

| Option | Description | Selected |
|--------|-------------|----------|
| DROP PARTITION | 直接删除整个过期分区 | ✓ |
| DELETE 语句 | 使用 DELETE FROM ... WHERE event_time < cutoff | |
| 混合方式 | 对于完整过期的月份使用 DROP PARTITION，对于当前月份的旧数据使用 DELETE | |

**User's choice:** DROP PARTITION，但保留期增加 30 天

**Notes:** 用户意识到分区边界问题，如果今天是 10 月 15 日，DROP 6 月分区可能丢失 6 月 1 日-15 日的数据。因此保留期设为 120 天（90 + 30 缓冲）。

### 调度方式

| Option | Description | Selected |
|--------|-------------|----------|
| 定时任务（Cron） | 使用现有的 scheduler 创建定时任务，每天凌晨 2 点执行清理 | ✓ |
| 应用启动时 | 在应用启动时检查并清理过期分区 | |
| 手动触发+定时 | 提供管理员手动触发清理的 API，同时创建默认定时任务作为兜底 | |

**User's choice:** 定时任务（Cron）

**Notes:** 复用现有基础设施，集中管理。

### 配置方式

| Option | Description | Selected |
|--------|-------------|----------|
| 数据库配置表 | 存储在 sys_config 表，key 为 'network.mac.history.retention_days'，支持运行时修改 | ✓ |
| 配置文件 | 写在 configs/config.yaml 中 | |
| 环境变量 | 通过环境变量 MAC_HISTORY_RETENTION_DAYS 配置 | |

**User's choice:** 数据库配置表

**Notes:** 用户希望无需重启应用即可修改配置，管理员可通过 UI 调整。

---

## Claude's Discretion

以下决策由 Claude 根据项目背景和最佳实践推荐：

1. **历史表架构**: 混合模式（状态快照 + first_seen/last_seen）
   - 理由：与智能合并的 flapping 防护一致，支持高效停留时长计算

2. **集成时机**: 删除前对比
   - 理由：保证数据一致性，符合"最小侵入"原则

---

## Deferred Ideas

- 实时 MAC 流处理（超出运维系统职责）
- MAC 地址欺骗检测（需要深度包检测）
- 异常告警功能（后续里程碑实现）
- MAC 地址随机化识别（后续里程碑实现）
