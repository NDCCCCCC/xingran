# Phase 12: 数据模型与采集集成 - Research

**Researched:** 2026-05-09
**Domain:** PostgreSQL Time-Series Data + Go Service Integration
**Confidence:** HIGH

## Summary

本阶段需要为 MAC 地址历史数据建立基础设施，核心挑战是在现有采集流程中**最小侵入**地集成变更检测和历史记录功能。基于 CONTEXT.md 中的决策，采用增量快照策略（仅记录变更事件）而非全量存储，可节省 99% 存储空间。

**技术栈验证：**
- **PostgreSQL 18** 原生支持分区表和 BRIN 索引，无需引入 TimescaleDB [VERIFIED: CLAUDE.md]
- **GORM** 支持分区表操作和事务处理 [VERIFIED: codebase analysis]
- **robfig/cron** 已在项目中用于定时任务 [VERIFIED: internal/scheduler/cron.go]
- **现有采集服务** (`mac_collection_service.go`) 使用"删除再插入"策略，需要改造 [VERIFIED: code analysis]

**Primary recommendation:** 在 `collectDeviceMAC` 方法的事务中插入变更检测逻辑，先查询旧状态→对比→写历史→删除旧→插入新，确保数据一致性且不破坏现有采集流程。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 变更检测逻辑 | API / Backend | — | 在 `mac_collection_service.go` 中执行，属于业务逻辑层 |
| 历史数据存储 | Database / Storage | — | PostgreSQL 分区表存储历史数据 |
| 分区管理 | API / Backend | Database | 应用层代码创建/删除分区，但依赖 PostgreSQL DDL |
| 定时清理任务 | API / Backend | — | 使用 `internal/scheduler` 注册 Cron 任务 |
| 配置管理 | API / Backend | Database | 配置存储在 `sys_config` 表，运行时读取 |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| **GORM** | 1.30.5 | ORM / 数据库操作 | 项目已使用，支持事务、分区表、批量操作 [VERIFIED: go.mod] |
| **PostgreSQL** | 18 | 分区表 + BRIN 索引 | 原生支持时间序列优化，无需额外扩展 [VERIFIED: CLAUDE.md] |
| **robfig/cron** | v3.0.1 | 定时任务调度 | 项目已有 scheduler 封装，直接复用 [VERIFIED: internal/scheduler] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **github.com/google/uuid** | 1.6.0 | UUID 主键生成 | 历史表主键使用 UUID（同现有模型）[VERIFIED: go.mod] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| **PostgreSQL 原生分区** | TimescaleDB | TimescaleDB 需要额外扩展，增加部署复杂度；原生分区已满足需求 [ASSUMED: based on REQUIREMENTS.md decision] |
| **增量快照** | 全量快照 | 全量存储成本高（每设备数千条 MAC），增量快照节省 99% 空间 [CITED: REQUIREMENTS.md DATA-01] |

**Installation:**
无需安装新依赖 — 所有所需库已在项目中。

**Version verification:**
```bash
# 验证 GORM 版本（当前项目）
grep "gorm.io/gorm" go.mod
# 输出: gorm.io/gorm v1.30.5

# 验证 cron 版本（当前项目）
grep "robfig/cron" go.mod
# 输出: github.com/robfig/cron/v3 v3.0.1

# 验证 PostgreSQL 版本（参考 CLAUDE.md）
# PostgreSQL 18 (已在 CLAUDE.md 中确认)
```

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    定时调度触发                               │
│              internal/scheduler (Cron)                      │
│                     每 5-15 分钟                             │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              MAC 采集服务 (现有流程)                         │
│         mac_collection_service.CollectAllDevices()          │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  collectDeviceMAC() [需改造]                          │  │
│  │                                                       │  │
│  │  1. 发现 LLDP 邻居（现有）                             │  │
│  │  2. 执行命令获取 MAC 表（现有）                         │  │
│  │  3. 解析 MAC 地址（现有）                               │  │
│  │  4. 应用过滤规则（现有）                                 │  │
│  │                                                       │  │
│  │  【新增】变更检测逻辑                                   │  │
│  │  ├─ 5. 查询旧 MAC 状态 (DELETE 前)                     │  │
│  │  ├─ 6. 对比新旧状态，识别变更类型                       │  │
│  │  └─ 7. 写入历史表 (appeared/disappeared/...)          │  │
│  │                                                       │  │
│  │  8. 删除旧数据（现有）                                  │  │
│  │  9. 插入新数据（现有）                                  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  PostgreSQL 数据库                           │
│                                                              │
│  ┌────────────────────────┐  ┌──────────────────────────┐  │
│  │ sys_device_mac_address │  │ sys_device_mac_history   │  │
│  │   (当前状态表)           │  │   (历史分区表)             │  │
│  │                          │  │                          │  │
│  │  - device_id            │  │  - id (UUID)             │  │
│  │  - mac_address          │  │  - device_id             │  │
│  │  - interface_name       │  │  - device_name_snapshot  │  │
│  │  - vlan_id              │  │  - mac_address           │  │
│  │  - collected_at         │  │  - interface_name        │  │
│  │                          │  │  - vlan_id               │  │
│  └────────────────────────┘  │  - event_type (enum)      │  │
                               │  - first_seen              │  │
                               │  - last_seen               │  │
                               │  - collected_at            │  │
                               │  ─────────────────────────  │  │
                               │  分区: 按月 (first_seen)    │  │
                               │  索引: BRIN (first_seen)    │  │
                               │       B-tree (device_id,    │  │
                               │               mac_address)  │  │
                               └──────────────────────────┘  │
│                                                              │
│  ┌────────────────────────┐  ┌──────────────────────────┐  │
│  │   sys_config           │  │  分区管理任务             │  │
│  │                          │  │  (定时清理)                │  │
│  │  network.mac.history.   │  │                          │  │
│  │  retention_days: 120    │  │  每天凌晨 2 点             │  │
│  └────────────────────────┘  │  DROP 过期分区             │  │
                               └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure
```
internal/
├── models/
│   └── device_mac_history.go          # [NEW] 历史表模型
├── services/
│   ├── mac_collection_service.go      # [MODIFY] 添加变更检测
│   ├── mac_history_service.go         # [NEW] 历史数据查询服务
│   └── mac_history_partition.go       # [NEW] 分区管理服务
└── scheduler/
    └── mac_history_tasks.go           # [NEW] 定时清理任务
```

### Pattern 1: 变更检测逻辑（事务内执行）
**What:** 在删除旧数据前先查询旧状态，与新采集结果对比，识别变更类型并写入历史表。
**When to use:** MAC 地址采集流程中，在 `DELETE` 之前插入。
**Example:**
```go
// Source: CONTEXT.md D-01, D-02
func (s *MACCollectionService) collectDeviceMAC(ctx context.Context, device *models.NetworkDevice) *MACCollectionResult {
    // ... 现有采集逻辑 ...

    // Step 5: 查询旧状态（DELETE 前）
    var oldMACs []models.DeviceMACAddress
    s.db.WithContext(ctx).Where("device_id = ?", device.ID).Find(&oldMACs)

    // Step 6: 构建新旧状态映射
    oldState := buildMACStateMap(oldMACs)
    newState := buildMACStateMap(filteredMACAddresses)

    // Step 7: 识别变更并写历史（在事务中）
    if err := s.recordMACChanges(ctx, device, oldState, newState); err != nil {
        applogger.Errorf("[MAC采集] %s: 记录历史失败: %v", device.DeviceName, err)
        // 不阻断采集流程（CONTEXT.md D-07）
    }

    // Step 8: 删除旧数据（现有逻辑）
    s.db.WithContext(ctx).Where("device_id = ?", device.ID).Delete(&models.DeviceMACAddress{})

    // Step 9: 插入新数据（现有逻辑）
    // ...
}
```

### Pattern 2: 状态转换矩阵（复合键变更检测）
**What:** 使用 `device_id + mac_address + interface_name + vlan_id` 复合键识别 4 种变更类型。
**When to use:** 对比新旧状态时判断事件类型。
**Example:**
```go
// Source: CONTEXT.md D-02
type MACEvent struct {
    MACAddress    string
    InterfaceName string
    VLANID        *int
}

func detectChangeEvent(oldMACs, newMACs map[MACEvent]bool) []EventType {
    var changes []EventType

    // appeared: 新数据存在且旧数据不存在
    for mac := range newMACs {
        if !oldMACs[mac] {
            changes = append(changes, EventAppeared)
        }
    }

    // disappeared: 新数据不存在且旧数据存在
    for mac := range oldMACs {
        if !newMACs[mac] {
            changes = append(changes, EventDisappeared)
        }
    }

    // moved: MAC 相同但接口不同
    // vlan_changed: 接口相同但 VLAN 不同
    // ... 详细逻辑见实现

    return changes
}
```

### Pattern 3: 分区表创建（应用层自动管理）
**What:** 应用启动时检查未来 1-2 个月分区是否存在，不存在则创建。
**When to use:** 应用初始化时（`main.go` 或 `core.Init()`）。
**Example:**
```go
// Source: CONTEXT.md D-05
func CreateMonthlyPartition(db *gorm.DB, year int, month int) error {
    tableName := fmt.Sprintf("sys_device_mac_history_%d_%02d", year, month)

    sql := fmt.Sprintf(`
        CREATE TABLE IF NOT EXISTS %s
        PARTITION OF sys_device_mac_history
        FOR VALUES FROM ('%d-%02d-01') TO ('%d-%02d-01')
    `, tableName, year, month, year, month+1)

    return db.Exec(sql).Error
}
```

### Pattern 4: BRIN 索引创建（时间序列优化）
**What:** 为 `first_seen` 字段创建 BRIN 索引，优化时间范围查询。
**When to use:** 历史表初始迁移时创建。
**Example:**
```go
// Source: CONTEXT.md D-04, PostgreSQL docs
func CreateBRINIndex(db *gorm.DB) error {
    // pages_per_range 默认 128（适合时间序列数据）
    sql := `
        CREATE INDEX CONCURRENTLY IF NOT EXISTS
        idx_mac_history_first_seen_brin
        ON sys_device_mac_history USING BRIN (first_seen)
        WITH (pages_per_range = 128)
    `
    return db.Exec(sql).Error
}
```

### Anti-Patterns to Avoid
- **在事务外执行变更检测:** 可能导致旧状态已删除，无法对比
- **使用外键关联设备名称:** 设备删除后历史记录无法读取，应使用快照字段
- **每次采集写全量快照:** 存储成本极高，违背增量快照设计
- **同步删除分区可能导致查询失败:** 应使用 `DROP PARTITION` 而非 `DELETE`，且在业务低峰期执行

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 分区管理 | 手动执行 SQL DDL | 应用层代码自动创建 + 定时清理 | 确保分区始终存在，避免数据写入失败 |
| Cron 任务调度 | 自己实现定时逻辑 | `internal/scheduler` (已有) | 复用现有基础设施，支持任务日志 |
| 配置管理 | 硬编码保留期 | `sys_config` 表 + 运行时读取 | 支持运行时调整，无需重启 |
| UUID 生成 | 手动拼接字符串 | `google/uuid` 库 | 避免格式错误，保证唯一性 |

**Key insight:** PostgreSQL 的分区表是声明式的，应用层只需创建子表，查询时路由器自动根据分区键路由到对应分区。手动管理分区边界容易出错（如数据写入时分区不存在），应使用自动化脚本在应用启动时检查并创建缺失分区。

## Runtime State Inventory

> 本阶段为新建功能，不涉及重命名/重构，无需运行时状态清单。

## Common Pitfalls

### Pitfall 1: 采集流程破坏导致历史记录不完整
**What goes wrong:** 如果在删除旧数据**之后**才查询旧状态，无法获取历史对比数据。
**Why it happens:** "删除再插入"策略是原有流程，在 DELETE 之后插入变更检测会导致旧数据丢失。
**How to avoid:** **必须在 DELETE 之前**查询旧状态，整个流程（查询→对比→写历史→删除→插入）在一个事务中完成。
**Warning signs:** 历史表只有 `appeared` 事件，缺少 `disappeared`/`moved`/`vlan_changed` 事件。

### Pitfall 2: 分区边界导致数据丢失
**What goes wrong:** 数据写入时目标分区不存在，SQL 报错 "no partition of relation found for row"。
**Why it happens:** 分区是按月创建的，如果应用在月底运行，下个月分区可能还未创建。
**How to avoid:** 应用启动时检查未来 1-2 个月的分区，不存在则创建；或使用 `default partition` 捕获未匹配数据。
**Warning signs:** 采集日志中出现 "partition not found" 错误。

### Pitfall 3: MAC 地址格式不一致导致对比失败
**What goes wrong:** 同一 MAC 地址在不同设备/采集时间显示格式不同（如 `aabb.ccdd.eeff` vs `aa:bb:cc:dd:ee:ff`），导致 `moved` 检测失败。
**Why it happens:** 不同厂商输出格式不同，解析逻辑未统一规范化。
**How to avoid:** 数据库层强制规范化（使用 `macaddr` 类型）或在应用层统一转换为标准格式（如 `AA:BB:CC:DD:EE:FF`）。
**Warning signs:** 查询结果显示同一 MAC 的 `moved` 事件异常频繁。

### Pitfall 4: BRIN 索引参数配置不当
**What goes wrong:** BRIN 索引未生效或查询性能未提升。
**Why it happens:** `pages_per_range` 默认值 128 适合顺序写入的时间序列，但如果数据分布不均匀可能需要调整。
**How to avoid:** 根据实际数据量和查询模式调整参数；监控索引大小（BRIN 索引应该很小，通常几 MB）。
**Warning signs:** 查询计划显示 `Seq Scan` 而非 `Bitmap Heap Scan`。

### Pitfall 5: 清理任务删除正在查询的分区
**What goes wrong:** 用户查询历史数据时，分区被定时任务删除，查询返回不完整结果。
**Why it happens:** 清理任务和查询任务并发执行，无锁保护。
**How to avoid:** 清理任务在业务低峰期执行（凌晨 2 点）；保留期设为 120 天而非 90 天，避免边界问题。
**Warning signs:** 用户报告"昨天还能查询的数据今天不见了"。

## Code Examples

Verified patterns from official sources:

### 分区表创建（PostgreSQL 原生语法）
```sql
-- Source: PostgreSQL Documentation https://www.postgresql.org/docs/current/ddl-partitioning.html
-- 创建主表（声明为 PARTITIONED BY）
CREATE TABLE sys_device_mac_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL,
    device_name_snapshot VARCHAR(100),
    mac_address VARCHAR(30) NOT NULL,
    interface_name VARCHAR(100) NOT NULL,
    vlan_id INT,
    event_type VARCHAR(20) NOT NULL,
    first_seen TIMESTAMP NOT NULL,
    last_seen TIMESTAMP NOT NULL,
    collected_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) PARTITION BY RANGE (first_seen);

-- 创建月度分区
CREATE TABLE sys_device_mac_history_2025_01
PARTITION OF sys_device_mac_history
FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE sys_device_mac_history_2025_02
PARTITION OF sys_device_mac_history
FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
```

### BRIN 索引创建（时间序列优化）
```sql
-- Source: PostgreSQL Documentation https://www.postgresql.org/docs/current/indexes-brin.html
-- 为时间序列字段创建 BRIN 索引（极小的索引尺寸）
CREATE INDEX CONCURRENTLY idx_mac_history_first_seen_brin
ON sys_device_mac_history USING BRIN (first_seen)
WITH (pages_per_range = 128);

-- 复合索引（用于设备 + MAC 查询）
CREATE INDEX CONCURRENTLY idx_mac_history_device_mac
ON sys_device_mac_history (device_id, mac_address);
```

### GORM 事务操作（确保一致性）
```go
// Source: GORM Documentation https://gorm.io/docs/transactions.html
err := db.Transaction(func(tx *gorm.DB) error {
    // 1. 查询旧状态
    var oldMACs []DeviceMACAddress
    if err := tx.Where("device_id = ?", deviceID).Find(&oldMACs).Error; err != nil {
        return err
    }

    // 2. 写入历史（变更检测）
    for _, change := range detectChanges(oldMACs, newMACs) {
        history := &DeviceMACHistory{
            DeviceID:          deviceID,
            DeviceNameSnapshot: deviceName,
            MACAddress:        change.MAC,
            EventType:         change.Type,
            FirstSeen:         change.FirstSeen,
            LastSeen:          change.LastSeen,
        }
        if err := tx.Create(history).Error; err != nil {
            return err
        }
    }

    // 3. 删除旧数据
    if err := tx.Where("device_id = ?", deviceID).Delete(&DeviceMACAddress{}).Error; err != nil {
        return err
    }

    // 4. 插入新数据
    if err := tx.Create(&newMACs).Error; err != nil {
        return err
    }

    // 返回 nil 提交事务
    return nil
})

// 如果任一步骤失败，事务自动回滚
```

### 定时任务注册（复用现有 scheduler）
```go
// Source: internal/scheduler/cron.go (项目已有模式)
func RegisterMACHistoryCleanup(scheduler *scheduler.Scheduler, db *gorm.DB) {
    job := &models.Job{
        JobName:        "MAC地址历史数据清理",
        JobGroup:       "DEFAULT",
        CronExpression: "0 0 2 * * ?", // 每天凌晨 2 点
        InvokeTarget:   "mac_history_cleanup",
        Status:         0, // 启用
    }

    // 注册任务处理器
    scheduler.RegisterTaskHandler("mac_history_cleanup", func(ctx context.Context, params string) error {
        return CleanupExpiredPartitions(ctx, db)
    })

    // 添加到调度器
    if err := scheduler.AddJob(job); err != nil {
        applogger.Errorf("注册 MAC 历史清理任务失败: %v", err)
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 全量快照（每次采集存储完整 MAC 表） | 增量快照（仅记录变更事件） | v1.5 设计决策 | 节省 99% 存储空间，但需要变更检测逻辑 |
| 单表存储所有历史数据 | PostgreSQL 月度分区 + BRIN 索引 | v1.5 设计决策 | 查询性能提升，过期分区瞬间删除 |
| 硬编码数据保留期 | 配置表存储 + 运行时读取 | v1.5 设计决策 | 运维可灵活调整，无需重启应用 |

**Deprecated/outdated:**
- 手动管理分区 SQL（应使用应用层自动化脚本）
- 使用 `DELETE FROM ... WHERE collected_at < ...` 清理旧数据（应使用 `DROP PARTITION`）

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | PostgreSQL 18 原生分区性能满足需求，无需 TimescaleDB | Standard Stack | 如果数据量超预期（>10 亿条），可能需要迁移到 TimescaleDB |
| A2 | BRIN 索引 `pages_per_range=128` 适合时间序列查询 | Architecture Patterns | 如果查询性能不达标，需调整参数或改用 B-tree 索引 |
| A3 | 现有采集服务可承载变更检测带来的额外查询开销 | Architecture Patterns | 如果查询导致采集超时，需优化查询或异步处理历史记录 |
| A4 | 设备名称快照字段（`device_name_snapshot`）足够满足历史查询需求 | Don't Hand-Roll | 如果用户需要完整设备信息（如 IP、厂商），可能需要冗余更多字段 |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed.

## Open Questions

1. **MAC 地址规范化策略**
   - What we know: 不同厂商输出格式不同（`aabb.ccdd.eeff` vs `aa:bb:cc:dd:ee:ff`）
   - What's unclear: 是否在数据库层使用 `macaddr` 类型强制规范化，还是应用层统一格式
   - Recommendation: 应用层统一转换为 `AA:BB:CC:DD:EE:FF` 格式，避免数据库类型兼容性问题

2. **分区创建时机**
   - What we know: CONTEXT.md 提到"应用启动时检查未来 1-2 个月"
   - What's unclear: 具体是启动时一次性创建，还是使用定时任务每月创建
   - Recommendation: 启动时检查并创建未来 2 个月分区，同时保留定时任务作为兜底

3. **历史记录失败后的降级策略**
   - What we know: CONTEXT.md D-07 规定"历史记录失败不阻断采集"
   - What's unclear: 是否需要告警机制通知运维历史记录异常
   - Recommendation: 初期仅记录日志，Phase 15（告警功能）再集成告警通道

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| PostgreSQL | 数据库分区表 | ✓ | 18 | — |
| GORM | ORM 操作 | ✓ | 1.30.5 | — |
| robfig/cron | 定时清理任务 | ✓ | v3.0.1 | — |
| google/uuid | UUID 生成 | ✓ | 1.6.0 | — |

**Missing dependencies with no fallback:**
- 无

**Missing dependencies with fallback:**
- 无

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + GORM mock (建议使用 `github.com/DATA-DOG/go-sqlmock`) |
| Config file | 标准的 `go test` 支持，无需额外配置 |
| Quick run command | `go test -v ./internal/services/ -run TestMACChangeDetection` |
| Full suite command | `go test ./internal/services/...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DATA-01 | MAC 变更快照记录（4 种事件类型） | integration | `go test -v -run TestMACChangeDetection` | ❌ Wave 0 |
| DATA-02 | 自动清理过期分区 | integration | `go test -v -run TestPartitionCleanup` | ❌ Wave 0 |
| QUERY-01 | 端口历史查询（分页 + 时间过滤） | unit | `go test -v -run TestQueryPortHistory` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -v ./internal/services/mac_collection_service.go`
- **Per wave merge:** `go test ./internal/services/...`
- **Phase gate:** 全部测试通过 + 手动验证分区创建和清理逻辑

### Wave 0 Gaps
- [ ] `internal/services/mac_collection_service_test.go` — 变更检测逻辑单元测试
- [ ] `internal/services/mac_history_partition_test.go` — 分区创建/删除集成测试
- [ ] `internal/services/mac_history_query_test.go` — 历史查询功能测试
- [ ] `Test main.go` 或测试初始化脚本 — 确保测试环境自动创建分区表结构

**Framework install:** 无需安装 — Go 标准库已包含 testing 包。

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes | GORM 参数化查询（防止 SQL 注入） |
| V6 Cryptography | no | 本阶段不涉及加密功能 |
| V2 Authentication | no | 历史查询需要认证，但在 Phase 14 实现 |

### Known Threat Patterns for Go + PostgreSQL

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| SQL 注入（分区名称拼接） | Tampering | 使用 GORM 参数化查询，避免字符串拼接分区名（验证年份/月份格式） |
| 时间序列攻击（通过查询历史推断网络拓扑） | Information Disclosure | Phase 15 实施访问控制，限制敏感设备的历史查询权限 |
| 拒绝服务（写入大量历史数据撑爆磁盘） | Denial of Service | 分区自动清理 + 磁盘空间监控告警 |

## Sources

### Primary (HIGH confidence)
- [PostgreSQL 18: Table Partitioning Documentation](https://www.postgresql.org/docs/current/ddl-partitioning.html) — 分区表创建语法
- [PostgreSQL 18: BRIN Indexes](https://www.postgresql.org/docs/current/indexes-brin.html) — BRIN 索引用法和参数配置
- [GORM Transaction Documentation](https://gorm.io/docs/transactions.html) — 事务操作模式
- [Context7: robfig/cron v3] — Cron 表达式格式和任务注册（如可用）
- [项目代码: internal/scheduler/cron.go] — 现有 scheduler 实现模式
- [项目代码: internal/services/mac_collection_service.go] — 现有 MAC 采集流程

### Secondary (MEDIUM confidence)
- [Postgres Partitioning Best Practices - PyCon Italia 2025](https://2025.pycon.it/en/event/postgres-partitioning-best-practices) — 分区设计最佳实践
- [How to Partition a Table in PostgreSQL - Simplified Guide](https://www.simplified.guide/postgresql/table-partition) — 分区实现步骤
- [Declarative Partitioning with pg_partman and pg_cron](https://medium.com/@golaneduard1/declarative-partitioning-in-postgresql-migrating-and-automating-with-pg-partman-and-pg-cron-b6d978abb507) — 分区自动化管理思路

### Tertiary (LOW confidence)
- [When to Consider Postgres Partitioning in 2026](https://medium.com/@fklezin/when-to-consider-postgres-partitioning-in-2026-71189ac88728) — 分区性能评估（标记验证）

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - 所有库已在项目中使用并验证
- Architecture: HIGH - 基于 CONTEXT.md 锁定决策和现有代码分析
- Pitfalls: MEDIUM - 部分风险（如 BRIN 参数调整）需实际运行验证

**Research date:** 2026-05-09
**Valid until:** 30 days (PostgreSQL 18 分区功能稳定，GORM API 稳定)
