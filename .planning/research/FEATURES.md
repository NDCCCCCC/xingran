# Feature Landscape: MAC地址历史数据管理

**Domain:** 网络运维管理系统 - MAC地址历史追踪与审计
**Researched:** 2026-05-09
**Milestone:** v1.5 MAC地址历史数据管理

## Executive Summary

MAC地址历史数据管理是网络运维管理系统的关键审计能力，支持安全追溯、异常检测和合规性报告。基于现有MAC地址采集系统（Phase 11已完成），本期重点增加历史快照、变更追踪和分析查询功能。

**核心发现:**
- 历史追踪应仅记录MAC地址变更（增量记录，节省存储）
- 典型保留周期为30-90天（活跃监控），最长12个月（合规要求）
- 查询模式分为三类：端口历史、MAC轨迹、异常告警
- 数据库设计需平衡查询性能与存储成本

## Table Stakes

用户期望的基础功能。缺失这些会让产品感觉不完整。

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **MAC变更快照记录** | 网络审计要求可追溯性 | Medium | 仅在MAC地址变化时记录（首次出现/消失/迁移） |
| **端口历史查询** | 追踪特定端口上的MAC变化 | Low | 按设备+接口名查询，时间正序/倒序 |
| **MAC轨迹查询** | 追踪终端设备移动路径 | Medium | 按MAC地址查询跨端口/设备的迁移历史 |
| **数据导出** | 合规报告和离线分析需要 | Low | Excel/CSV导出，支持时间范围筛选 |
| **自动清理机制** | 防止数据库无限增长 | Low | 可配置保留天数，定时任务清理过期数据 |

## Differentiators

区分产品的增值功能。用户未明确期望，但高度赞赏。

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **MAC异常告警** | 主动发现安全问题（MAC flapping、异常迁移） | High | 需规则引擎 + 告警通道集成 |
| **连接时长统计** | 识别长期占用端口的设备 | Medium | 计算 `last_seen - first_seen`，按MAC/端口聚合 |
| **热力图可视化** | 快速识别端口使用密度 | High | 前端图表（ECharts），需聚合API |
| **合规性报告** | 自动生成审计报告（PCI DSS等） | Medium | 模板化报告，支持导出PDF |
| **MAC指纹识别** | 识别设备类型（OUI厂商识别） | Low | 解析MAC前3字节OUI，厂商映射表 |

## Anti-Features

明确不应构建的功能。

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **实时MAC流处理** | 超出运维系统职责，应由专业NMS处理 | 保留定时采集（5-15分钟间隔） |
| **MAC地址欺骗检测** | 需要深度包检测，性能开销大 | 依赖网络安全设备（防火墙/IDS） |
| **物理位置精确定位** | 需要额外基础设施（WiFi定位等） | 提供楼层/工位关联（已存在） |
| **全量MAC表每次快照** | 存储成本极高（每设备数千条MAC） | 仅记录变更事件 |
| **MAC地址冲突检测** | 网络设备自身会处理CAM表冲突 | 信任设备数据，记录异常事件 |

## Feature Dependencies

```
Phase 11 (MAC采集优化) → Phase 12 (MAC历史追踪)
  ├─ LLDP发现服务 (已完成) ──→ 用于识别上行端口（排除历史查询干扰）
  ├─ 过滤规则配置 (已完成) ──→ 历史数据继承过滤策略
  └─ DeviceMACAddress模型 ──→ 扩展历史表设计

MAC历史数据 ──→ 异常告警 (Phase 13可选)
MAC历史数据 ──→ 合规报告 (Phase 13可选)
```

**关键依赖:**
1. **现有MAC采集系统稳定运行** — Phase 11已完成，数据源可靠
2. **DeviceMACAddress表结构** — 现有字段：`device_id`, `mac_address`, `interface_name`, `vlan_id`, `mac_type`, `collected_at`
3. **LLDP邻居数据** — 用于识别上行端口（历史查询时排除）
4. **定时调度器** — `internal/scheduler/` 用于历史数据清理任务

## MVP Recommendation

**Phase 12 核心功能（最小可行产品）:**

### 优先级 P0（必须实现）
1. **MAC变更快照记录**
   - 模型设计：`sys_mac_address_history` 表
   - 触发时机：MAC首次出现、消失、迁移（端口变化）
   - 记录字段：事件类型、前后状态、时间戳

2. **端口历史查询**
   - API: `GET /network/mac-history/port/:deviceId/:interfaceName`
   - 参数：时间范围、分页
   - 响应：MAC列表 + 事件时间线

3. **MAC轨迹查询**
   - API: `GET /network/mac-history/trace/:macAddress`
   - 响应：迁移路径（设备A/端口1 → 设备B/端口2 → ...）

4. **自动数据清理**
   - 配置：保留天数（默认90天）
   - 定时任务：每日凌晨清理过期数据

### 优先级 P1（高价值）
5. **数据导出功能**
   - 导出格式：Excel（复用现有导出框架）
   - 筛选条件：时间范围、设备、接口

6. **历史统计Dashboard**
   - Widget：MAC变更趋势图（日/周/月）
   - Widget：最活跃端口（Top 10）

### 延迟到 Phase 13
- MAC异常告警（规则引擎 + 告警集成）
- 连接时长统计（需窗口函数优化）
- 热力图可视化（前端图表组件）
- MAC厂商识别（OUI数据库）
- 合规性报告（PDF生成）

## 典型数据保留策略

基于行业最佳实践和合规要求：

| 保留周期 | 适用场景 | 推荐值 |
|---------|---------|--------|
| **活跃数据** | 实时查询、异常检测 | 7-30天 |
| **审计数据** | 安全追溯、事件分析 | 90天（行业标准） |
| **合规数据** | PCI DSS等合规要求 | 12个月 |

**实施建议:**
- 默认保留90天（平衡存储与需求）
- 提供 `sys_config` 配置项 `mac.history.retention_days`
- 定时任务每日清理，使用软删除（`deleted_at`）+ 硬删除

## 常见查询模式

### 1. 端口历史查询
**用例:** "查看 GigabitEthernet0/1/1 端口最近7天的MAC变化"

```sql
-- 查询模式
SELECT * FROM sys_mac_address_history
WHERE device_id = ? AND interface_name = ?
  AND event_time BETWEEN ? AND ?
ORDER BY event_time DESC
LIMIT 50 OFFSET 0;
```

### 2. MAC轨迹查询
**用例:** "追踪设备 aa:bb:cc:dd:ee:ff 的迁移路径"

```sql
-- 查询模式
WITH mac_moves AS (
  SELECT
    device_id, interface_name, event_time,
    LAG(interface_name) OVER (ORDER BY event_time) AS prev_interface,
    LAG(device_id) OVER (ORDER BY event_time) AS prev_device
  FROM sys_mac_address_history
  WHERE mac_address = 'aa:bb:cc:dd:ee:ff'
)
SELECT * FROM mac_moves
WHERE interface_name != prev_interface OR prev_interface IS NULL;
```

### 3. 异常检测查询
**用例:** "发现MAC flapping（1小时内迁移>3次）"

```sql
-- 查询模式
SELECT mac_address, COUNT(*) AS move_count
FROM sys_mac_address_history
WHERE event_time > NOW() - INTERVAL '1 hour'
  AND event_type = 'MOVED'
GROUP BY mac_address
HAVING COUNT(*) > 3;
```

### 4. 统计聚合查询
**用例:** "每日新增MAC数量趋势"

```sql
-- 查询模式
SELECT
  DATE(event_time) AS date,
  COUNT(DISTINCT mac_address) AS new_macs
FROM sys_mac_address_history
WHERE event_type = 'APPEARED'
  AND event_time > NOW() - INTERVAL '30 days'
GROUP BY DATE(event_time)
ORDER BY date;
```

## 数据库设计建议

### 新增表: `sys_mac_address_history`

| 字段 | 类型 | 说明 | 索引 |
|------|------|------|------|
| `id` | uuid | 主键 | PK |
| `device_id` | uuid | 设备ID | FK, Index |
| `mac_address` | varchar(30) | MAC地址 | Index |
| `interface_name` | varchar(100) | 接口名 | - |
| `vlan_id` | int | VLAN | - |
| `mac_type` | varchar(20) | 类型（dynamic/static） | - |
| `event_type` | varchar(20) | **事件类型**（关键） | Index |
| `event_time` | timestamp | 事件时间 | Index |
| `old_interface` | varchar(100) | 迁移前接口（MOVED事件） | - |
| `old_device_id` | uuid | 迁移前设备（MOVED事件） | - |
| `metadata` | jsonb | 扩展字段（厂商OUI等） | - |
| `created_at` | timestamp | 记录创建时间 | - |

**事件类型枚举:**
```go
const (
    MACEventAppeared = "APPEARED" // 首次出现
    MACEventDisappeared = "DISAPPEARED" // 消失
    MACEventMoved = "MOVED" // 迁移（端口变化）
    MACEventReappeared = "REAPPEARED" // 重新出现
)
```

**索引策略:**
```sql
-- 复合索引：端口历史查询
CREATE INDEX idx_mac_history_port ON sys_mac_address_history(device_id, interface_name, event_time DESC);

-- 复合索引：MAC轨迹查询
CREATE INDEX idx_mac_history_trace ON sys_mac_address_history(mac_address, event_time DESC);

-- 复合索引：异常检测（时间窗口）
CREATE INDEX idx_mac_history_anomaly ON sys_mac_address_history(event_time DESC, event_type);
```

**分区策略（可选，数据量>100万条时考虑）:**
```sql
-- 按月分区（PostgreSQL）
CREATE TABLE sys_mac_address_history (
    -- ...
) PARTITION BY RANGE (event_time);

-- 自动创建未来分区
CREATE TABLE sys_mac_address_history_2026_05 PARTITION OF sys_mac_address_history
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
```

## 技术实现要点

### 1. 变更检测算法

在 `MACCollectionService.CollectDeviceMAC()` 中增加历史记录逻辑：

```go
// 伪代码
func (s *MACCollectionService) collectDeviceMAC(...) {
    // ... 现有采集逻辑 ...

    // Step 5: 对比新旧MAC地址，记录变更
    oldMACs := s.getExistingMACs(ctx, device.ID)
    newMACs := filteredMACAddresses

    for _, newMAC := range newMACs {
        if oldMAC, exists := oldMACs[newMAC.MACAddress]; !exists {
            // 新出现
            s.recordMACEvent(ctx, device.ID, MACEventAppeared, newMAC)
        } else if oldMAC.InterfaceName != newMAC.InterfaceName {
            // 端口迁移
            s.recordMACEvent(ctx, device.ID, MACEventMoved, newMAC, oldMAC)
        }
    }

    for _, oldMAC := range oldMACs {
        if _, exists := newMACs[oldMAC.MACAddress]; !exists {
            // 消失
            s.recordMACEvent(ctx, device.ID, MACEventDisappeared, oldMAC)
        }
    }
}
```

### 2. 批量插入优化

使用 GORM 的 `CreateInBatches` 避免N+1问题：

```go
func (s *MACCollectionService) recordMACEventBatch(ctx context.Context, events []MACHistoryEvent) error {
    if len(events) == 0 {
        return nil
    }

    records := make([]*models.MACAddressHistory, len(events))
    for i, event := range events {
        records[i] = &models.MACAddressHistory{
            DeviceID:      event.DeviceID,
            MACAddress:    event.MACAddress,
            InterfaceName: event.InterfaceName,
            EventType:     event.EventType,
            EventTime:     time.Now(),
            // ...
        }
    }

    return s.db.WithContext(ctx).CreateInBatches(records, 100).Error
}
```

### 3. 查询性能优化

**分页优化:** 使用 `cursor-based` 分页替代 `OFFSET`（大数据量场景）

```go
// 传统分页（慢）
OFFSET 10000 LIMIT 20

// 游标分页（快）
WHERE event_time < ? ORDER BY event_time DESC LIMIT 20
```

**缓存策略:** 热点查询结果缓存5分钟（Redis）

```go
cacheKey := fmt.Sprintf("mac:history:port:%s:%s:%d", deviceID, interfaceName, timeWindow)
cache.GetOrSet(ctx, cacheKey, &result, 5*time.Minute, func() (interface{}, error) {
    return s.queryPortHistoryFromDB(ctx, deviceID, interfaceName, timeWindow)
})
```

## 存储成本估算

假设中等规模网络环境：
- 设备数：100台交换机
- 每设备活跃MAC数：500个（过滤后）
- 采集频率：15分钟/次
- 每日MAC变更率：5%（25个MAC/设备/天）

**每日新增记录数:**
```
100 devices × 25 MAC changes/day = 2,500 records/day
```

**年增长估算:**
```
2,500 records/day × 365 days = 912,500 records/year
```

**存储空间（估算）:**
- 单条记录：~200 bytes（含索引）
- 90天存储：`2,500 × 90 × 200 = 45 MB`
- 12个月存储：`912,500 × 200 = 182.5 MB`

**结论:** 存储成本可控，无需特殊优化策略。

## 合规性考虑

### PCI DSS 要求
- **Requirement 10.3:** 至少保留12个月审计日志
- **Requirement 10.5:** 保护审计日志免遭篡改

**实施建议:**
- 默认保留90天（平衡成本）
- 提供"长期归档"选项（12个月，需额外存储）
- 历史表启用 `INSERT ONLY`（禁止UPDATE/DELETE物理记录）
- 清理操作使用软删除 + 定期VACUUM

### GDPR 考虑
- MAC地址可能关联个人设备（欧盟）
- 建议：提供数据匿名化选项（哈希MAC地址后缀）

## 集成点

### 复用现有组件

| 组件 | 用途 | 位置 |
|------|------|------|
| **定时调度器** | 历史数据清理任务 | `internal/scheduler/` |
| **Excel导出框架** | MAC历史数据导出 | `internal/services/operations/excel_service.go` |
| **LLDP服务** | 排除上行端口查询 | `internal/services/lldp/` |
| **过滤规则配置** | 历史数据继承过滤 | `internal/models/mac_filter_rule.go` |
| **Widget系统** | 历史统计Dashboard | `internal/services/dashboard/` |
| **审计日志** | 用户操作审计 | `internal/models/rpa/audit_log.go`（参考模式） |

### 新增服务

| 服务 | 职责 | 位置 |
|------|------|------|
| **MACHistoryService** | 变更记录、查询、统计 | `internal/services/mac_history_service.go` |
| **MACAnomalyService** | 异常检测、告警规则 | `internal/services/mac_anomaly_service.go` (Phase 13) |

## 用户界面建议

### 前端页面结构

```
/network/mac-history                    # MAC历史首页
├── /port/:deviceId/:interfaceName      # 端口历史详情
├── /trace/:macAddress                  # MAC轨迹追踪
├── /anomalies                          # 异常事件列表
└── /statistics                         # 统计Dashboard
```

### 关键交互

1. **端口历史时间线**
   - 视图：垂直时间轴，展示MAC出现/消失事件
   - 操作：点击事件查看详情（VLAN、类型、OUI厂商）

2. **MAC轨迹地图**
   - 视图：拓扑图 + 迁移路径箭头
   - 操作：时间滑块回放历史迁移

3. **异常告警面板**
   - 视图：表格 + 严重程度标签
   - 操作：批量确认、导出报告

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| **数据量爆炸** | 数据库性能下降 | 1) 仅记录变更（非全量快照）<br>2) 自动清理策略<br>3) 分区表（数据量>100万） |
| **查询性能** | 历史查询超时 | 1) 复合索引优化<br>2) 时间范围限制（最大90天）<br>3) Redis缓存热点数据 |
| **时区问题** | 事件时间混乱 | 统一使用UTC存储，前端展示时转换 |
| **MAC随机化** | WiFi设备频繁变化 | 1) 识别随机化MAC（OUI前缀）<br>2) 提供过滤选项 |

## 后续增强方向

**Phase 13 可选功能:**
1. **机器学习异常检测** — 基于历史模式识别异常行为
2. **预测性维护** — 分析端口老化趋势（MAC表错误率）
3. **自动化根因分析** — 结合LLDP拓扑分析MAC flapping根因
4. **多租户隔离** — 为不同部门提供独立视图

## Sources

- [MAC Addresses: Functions, Risks, and Management | ACTE (2026)](https://www.acte.in/mac-addresses-functions-security-risks-and-management-in-networking)
- [OpUtils MAC address tracker - ManageEngine](https://www.manageengine.com/products/oputils/blog/oputils-mac-address-tracker-we-have-got-your-networks-back.html)
- [2026 Network Monitoring Trends - AKIPS](https://www.akips.com/blog/2026-network-monitoring-trends/)
- [Network Management System Best Practices - Cisco](https://www.cisco.com/c/en/us/support/docs/availability/high-availability/15114-NMS-bestpractice.html)
- [Security Log Retention Best Practices - Optro](https://optro.ai/blog/security-log-retention-best-practices-guide)
- [Log Retention Guide - Huntress](https://www.huntress.com/cybersecurity-101/topic/log-retention)
- [MAC Address Tracking - ExtremeXOS User Guide](https://documentation.extremenetworks.com/exos_32.2/GUID-E57BDDE5-01B8-4ED8-92C1-32799854B1C4.shtml)
- [Randomized MAC Addresses Guide - Addigy](https://addigy.com/blog/randomized-mac-address-guide/)
- [Randomize MAC Address Analytics - Purple](https://www.purple.ai/en-gb/blogs/randomize-mac-address)

## Research Confidence

| Area | Confidence | Notes |
|------|------------|-------|
| Core Features | HIGH | 基于行业标准和Phase 11架构 |
| Database Design | HIGH | GORM + PostgreSQL模式已验证 |
| Query Patterns | MEDIUM | 需实际数据量验证性能 |
| Compliance Requirements | MEDIUM | 基于公开文档，需法务确认 |
| Storage Estimates | LOW | 假设中等规模，实际场景差异大 |

## Gaps & Phase-Specific Research

**需要Phase 12详细研究的问题:**
1. 变更检测算法性能影响（现有采集流程会增加多少延迟？）
2. 大数据量查询优化（PostgreSQL分区表 vs 时间序列数据库）
3. MAC随机化处理策略（识别和过滤机制）
4. 前端时间线组件选型（Ant Design Timeline vs 自定义）

**可以延后到Phase 13:**
- 异常检测规则引擎设计
- 机器学习模型集成可行性
- 合规报告模板设计
