# Phase 12: 数据模型与采集集成 - Context

**Gathered:** 2026-05-09
**Status:** Ready for planning

<domain>
## Phase Boundary

建立 MAC 地址历史数据基础设施，最小侵入集成到现有采集流程。核心目标：MAC 地址变更自动记录到历史表（appeared/disappeared/moved/vlan_changed），支持按设备/端口查询历史变化，使用月度分区存储并自动清理过期数据。

</domain>

<decisions>
## Implementation Decisions

### 变更检测策略
- **D-01**: 使用"删除前先查询"方案获取旧状态
  - 在执行 DELETE 之前，先查询当前 MAC 地址状态到内存
  - 与新采集结果对比，识别变更类型
  - 然后执行删除→插入操作
  - 优点：保证数据一致性，不会遗漏变更
  - 缺点：增加一次数据库查询操作

- **D-02**: 使用状态转换矩阵精确识别 4 种变更类型
  - **appeared**: 新数据存在且旧数据不存在
  - **disappeared**: 新数据不存在且旧数据存在
  - **moved**: MAC 地址相同但接口不同
  - **vlan_changed**: 接口相同但 VLAN 不同
  - 复合键：device_id + mac_address + interface_name + vlan_id

- **D-03**: 使用智能合并防止 MAC flapping 误判
  - 历史表使用 `first_seen` 和 `last_seen` 时间戳
  - 自动合并连续的相同状态记录
  - 减少重复记录，保留有效时间范围

### 历史表架构
- **D-04**: 采用混合模式设计历史表 `sys_device_mac_history`
  - 存储状态快照，每次变更插入新记录（不更新旧记录）
  - 使用 `first_seen` 和 `last_seen` 标识记录有效期
  - 核心字段：
    - `id` (UUID, 主键)
    - `device_id` (UUID, 外键关联)
    - `device_name_snapshot` (string, 设备名快照，非外键)
    - `mac_address` (string, MAC 地址)
    - `interface_name` (string, 接口名)
    - `vlan_id` (int, 可空)
    - `event_type` (enum, appeared/disappeared/moved/vlan_changed)
    - `first_seen` (timestamp, 记录首次出现时间)
    - `last_seen` (timestamp, 记录最后看到时间)

- **D-05**: 使用月度分区优化查询和清理性能
  - 按 `first_seen` 字段按月分区
  - 应用层自动创建分区（启动时检查未来 1-2 个月）
  - 使用 BRIN 索引优化时间序列查询

### 集成方式
- **D-06**: 在事务内完成变更检测和历史记录
  - 执行顺序：查询旧状态 → 对比识别变更 → 写入历史表 → 删除旧数据 → 插入新数据
  - 整个流程在一个数据库事务中完成
  - 修改位置：`mac_collection_service.go` 的 `collectDeviceMAC` 方法

- **D-07**: 采用非阻塞模式处理历史记录失败
  - 历史记录写入失败时只记录错误日志
  - 不影响当前 MAC 地址的采集和存储
  - 保证采集流程稳定性，历史问题不阻断当前状态

### 清理机制
- **D-08**: 使用 DROP PARTITION 删除过期分区
  - 保留期设为 120 天（90天 + 30天缓冲）
  - 确保默认 90 天数据全部保留，避免分区边界问题
  - 直接删除整个过期分区，瞬间释放空间

- **D-09**: 使用定时任务（Cron）自动清理
  - 复用现有 scheduler（internal/scheduler）
  - 每天凌晨 2 点执行清理任务
  - 检查并删除超过 120 天的完整月度分区

- **D-10**: 使用数据库配置表存储保留期配置
  - 配置键：`network.mac.history.retention_days`
  - 默认值：120
  - 支持运行时修改，无需重启应用
  - 管理员可通过 UI 修改

### Claude's Discretion
- 分区创建的具体时机（应用启动 vs 定时任务）
- 清理任务的 Cron 表达式具体值
- 错误日志的详细程度和格式
- BRIN 索引的页面大小参数

</decisions>

<specifics>
## Specific Ideas

- 变更检测逻辑应该在 `collectDeviceMAC` 方法的"删除旧数据"步骤之前插入
- 历史表分区命名规范：`sys_device_mac_history_YYYY_MM`（如 `sys_device_mac_history_2024_01`）
- 设备名称快照使用文本字段存储，不使用外键，确保设备删除后历史数据仍然可读
- MAC flapping 智能合并通过更新 `last_seen` 实现，而不是插入新记录
- 清理任务应该记录删除的分区名称和释放的空间大小

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求文档
- `.planning/REQUIREMENTS.md` — DATA-01, DATA-02, QUERY-01 需求定义
- `.planning/ROADMAP.md` — Phase 12 成功标准

### 现有代码
- `internal/models/device_mac_address.go` — 当前 MAC 地址模型
- `internal/collectors/mac_collector.go` — MAC 采集器（参考用）
- `internal/services/mac_collection_service.go` — MAC 采集服务，需要修改的目标文件
- `internal/scheduler/` — 定时任务调度器

### PostgreSQL 文档
- PostgreSQL Partitioning — https://www.postgresql.org/docs/current/partitioning.html
- BRIN Indexes — https://www.postgresql.org/docs/current/indexes-brin.html

### 项目规范
- `.planning/codebase/CONVENTIONS.md` — Go 代码风格、错误处理规范
- `.planning/codebase/ARCHITECTURE.md` — 分层架构、Handler-Service 模式

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **MAC 采集流程**: `collectDeviceMAC` 方法已经实现了完整的采集→解析→过滤→存储流程
- **批量操作**: 使用 GORM 的 `Create` 批量插入，性能已优化
- **错误处理**: 已有 panic 恢复机制（defer recover），可以复用到历史记录逻辑
- **定时任务**: `internal/scheduler` 已支持 Cron 任务，可直接使用

### Established Patterns
- **Handler-Service 模式**: 采集服务遵循此模式
- **事务模式**: 使用 `db.WithContext(ctx)` 确保上下文传递
- **日志规范**: 使用 `applogger` 包统一日志记录
- **配置管理**: 使用 `sys_config` 表存储运行时配置

### Integration Points
- MAC 采集服务被调度器定时调用
- 清理任务需要注册到调度器
- 历史表查询将在 Phase 13 实现
- 前端查询将在 Phase 14 实现

### Known Constraints
- 采集流程使用"删除再插入"策略，必须改造才能支持变更检测
- 设备删除后，历史记录中的设备名称必须仍然可读（使用快照）
- 分区管理需要额外的数据库权限（CREATE/DROP PARTITION）

</code_context>

<deferred>
## Deferred Ideas

- 实时 MAC 流处理（超出运维系统职责）
- MAC 地址欺骗检测（需要深度包检测）
- 异常告警功能（后续里程碑实现）
- MAC 地址随机化识别（后续里程碑实现）

</deferred>

---

*Phase: 12-data-model-integration*
*Context gathered: 2026-05-09*
