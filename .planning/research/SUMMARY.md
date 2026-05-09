# Research Summary: MAC地址历史数据管理 (v1.5)

**Project:** RuoYi-Next 网络运维管理系统 - MAC地址历史追踪与审计
**Research Date:** 2026-05-09
**Milestone:** v1.5 MAC地址历史数据管理
**Overall Confidence:** HIGH

---

## Executive Summary

MAC地址历史数据管理是网络运维管理系统的关键审计能力，基于现有MAC采集系统（Phase 11已完成），本期重点增加历史快照、变更追踪和分析查询功能。研究显示，采用**增量快照模式**（仅在MAC地址变化时记录历史）而非全量存储，可将存储成本降低99%，同时提供完整的审计追溯能力。

**核心技术决策:**
- **数据库架构**: PostgreSQL原生时间序列优化（BRIN索引 + 月度分区表），无需引入TimescaleDB等专用时序数据库
- **存储策略**: 增量快照（仅记录appeared/disappeared/moved/vlan_changed四类变更事件）
- **查询优化**: 复合索引 + 分区裁剪 + 物化视图，支持90天内<2秒查询响应
- **集成方式**: 最小侵入式扩展现有MACCollectionService，通过事务隔离保证数据一致性

**关键风险与缓解:**
- **风险1**: 数据库无限制增长 → 通过变更检测减少99%写入量 + 自动分区清理（保留13个月）
- **风险2**: 现有采集流程破坏历史数据 → 采用事务性变更检测，先记录历史再删除当前状态
- **风险3**: 查询性能下降 → BRIN索引（空间效率100倍）+ 分区裁剪（查询速度提升10-100倍）

**推荐实施路径:**
1. **Phase 12 (数据模型与采集集成)**: 创建分区表 + 变更检测逻辑 + 事务性集成
2. **Phase 13 (查询层与API)**: 历史查询API + MAC轨迹追踪 + 异常检测
3. **Phase 14 (前端与可视化)**: 时间线组件 + 轨迹可视化 + 导出功能
4. **Phase 15 (性能优化)**: 物化视图 + 缓存策略 + 分区归档

---

## Key Findings

### From STACK.md: Technology Additions

**No new infrastructure required** - 功能可在现有技术栈基础上实现：

| Component | Technology | Purpose | Why |
|-----------|-----------|---------|-----|
| **Database** | PostgreSQL 18 (existing) | 时间序列数据存储 | 原生分区+BRIN索引已足够，无需TimescaleDB |
| **时间序列优化** | PostgreSQL Partitioning + BRIN Indexes | 历史数据管理 | 查询性能提升10-100倍，存储空间节省99% |
| **后端增强** | `gorm.io/datatypes` (builtin) | JSONB异常元数据 | 灵活存储异常检测结果 |
| **前端组件** | ECharts 6.0.0 (existing) | 时间线可视化 | 已安装，直接使用timeline系列 |
| **前端组件** | Ant Design Timeline 6.1.1 (existing) | 历史事件展示 | 内置组件，适合时序展示 |
| **时间处理** | dayjs 1.11.19 (existing) | 时间范围处理 | 已安装，支持UTC转换 |

**关键数据库特性:**
- **BRIN索引**: 适合按时间有序写入的时序数据，索引大小比B-tree小100倍
- **分区表**: 按月分区，查询时自动裁剪无关分区，提升查询性能
- **JSONB字段**: 存储异常检测元数据，支持GIN索引快速查询

---

### From FEATURES.md: Core Feature Categories

**Table Stakes (必须实现)** - 用户期望的基础功能:

| Feature | 复杂度 | Why Expected | 实现要点 |
|---------|--------|--------------|----------|
| **MAC变更快照记录** | Medium | 网络审计要求可追溯性 | 仅在MAC变化时记录（appeared/disappeared/moved/vlan_changed） |
| **端口历史查询** | Low | 追踪特定端口上的MAC变化 | 按设备+接口名查询，时间范围过滤，分页支持 |
| **MAC轨迹查询** | Medium | 追踪终端设备移动路径 | 按MAC地址查询跨端口/设备的迁移历史，计算停留时长 |
| **数据导出** | Low | 合规报告和离线分析需要 | 复用现有excelize框架，支持时间范围筛选 |
| **自动清理机制** | Low | 防止数据库无限增长 | 可配置保留天数（默认90天），定时任务清理过期数据 |

**Differentiators (增值功能)** - 区分产品的特性:

| Feature | 复杂度 | Value Proposition | 实现要点 |
|---------|--------|-------------------|----------|
| **MAC异常告警** | High | 主动发现安全问题（MAC flapping、异常迁移） | 规则引擎检测频繁移动、异常时间、高频抖动 |
| **连接时长统计** | Medium | 识别长期占用端口的设备 | 计算 `last_seen - first_seen`，按MAC/端口聚合 |
| **热力图可视化** | High | 快速识别端口使用密度 | ECharts图表，需聚合API支持 |
| **MAC厂商识别** | Low | 解析MAC前3字节OUI | 厂商映射表（IEEE OUI数据库） |

**Anti-Features (明确不应构建)**:

| Anti-Feature | Why Avoid | 替代方案 |
|--------------|-----------|----------|
| **实时MAC流处理** | 超出运维系统职责，应由专业NMS处理 | 保留定时采集（5-15分钟间隔） |
| **MAC地址欺骗检测** | 需要深度包检测，性能开销大 | 依赖网络安全设备（防火墙/IDS） |
| **物理位置精确定位** | 需要额外基础设施（WiFi定位等） | 提供楼层/工位关联（已存在） |
| **全量MAC表每次快照** | 存储成本极高（每设备数千条MAC） | 仅记录变更事件（节省99%存储） |

**数据保留策略**（基于行业最佳实践）:
- **活跃数据**: 7-30天（实时查询、异常检测）
- **审计数据**: 90天（行业标准，安全追溯）
- **合规数据**: 12个月（PCI DSS等合规要求）

---

### From ARCHITECTURE.md: Architecture Integration Points

**核心架构模式: Handler-Service-Repository + 增量快照**

```
Request → MACCollectionService (采集)
                ↓
         [获取当前快照] ← sys_device_mac_address (当前状态表)
                ↓
         [对比检测变更] ← MACHistoryService.detectChanges()
                ↓
         [记录历史事件] → sys_device_mac_history (历史分区表)
                ↓
         [更新当前状态] → DELETE + INSERT (现有模式)
```

**新增服务:**

1. **MACHistoryService** (`internal/services/mac_history_service.go`)
   - 职责: 变更检测、历史记录写入、时态查询、聚合统计
   - 接口:
     - `RecordChanges()` - 记录MAC变更事件
     - `QueryHistory()` - 时态查询（AS OF TIME）
     - `GetMACTrajectory()` - MAC移动轨迹查询
     - `GetPortHistory()` - 端口历史查询
     - `DetectAnomalies()` - 异常检测（可选）

2. **修改现有服务: MACCollectionService**
   - 位置: `internal/services/mac_collection_service.go:196-246`
   - 变更: 在删除旧数据前，先调用`MACHistoryService.RecordChanges()`
   - 集成方式: 依赖注入 + 事务隔离保证数据一致性

**数据库Schema设计:**

```sql
-- 新增分区表: sys_device_mac_history
CREATE TABLE sys_device_mac_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL,
    device_name VARCHAR(100) NOT NULL,  -- 快照（非FK）
    mac_address VARCHAR(30) NOT NULL,
    interface_name VARCHAR(100) NOT NULL,
    vlan_id INT,
    mac_type VARCHAR(20),

    -- 时间序列字段
    collected_at TIMESTAMP NOT NULL,
    change_type VARCHAR(20) NOT NULL,  -- appeared/disappeared/moved/vlan_changed

    -- 变更追踪
    prev_interface VARCHAR(100),       -- 迁移前接口
    prev_vlan INT,                     -- 变更前VLAN

    created_at TIMESTAMP DEFAULT NOW()
) PARTITION BY RANGE (collected_at);

-- BRIN索引（时间序列优化）
CREATE INDEX idx_mac_history_collected_at_brin
  ON sys_device_mac_history USING BRIN (collected_at);

-- 复合索引（常用查询模式）
CREATE INDEX idx_mac_history_device_mac
  ON sys_device_mac_history(device_id, mac_address);
CREATE INDEX idx_mac_history_mac
  ON sys_device_mac_history(mac_address);
CREATE INDEX idx_mac_history_change_type
  ON sys_device_mac_history(change_type, collected_at DESC);
```

**API端点设计:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/network/mac/history/query` | 时态查询（历史快照） |
| POST | `/network/mac/history/trajectory` | MAC移动轨迹 |
| POST | `/network/mac/history/port` | 端口历史MAC列表 |
| POST | `/network/mac/history/anomalies` | 异常事件查询 |
| POST | `/network/mac/history/export` | 历史数据导出 |

**构建顺序与依赖:**

1. **Phase 1: 数据库Schema (1周)**
   - 创建分区表、BRIN索引、月度分区
   - 无依赖，可立即开始

2. **Phase 2: 历史服务层 (1-2周)**
   - 实现变更检测逻辑、历史记录写入
   - 依赖: Phase 1完成

3. **Phase 3: 修改MAC采集服务 (1周)**
   - 集成历史记录到采集流程
   - 依赖: Phase 2完成

4. **Phase 4: API层 (1周)**
   - 实现6个历史查询端点
   - 依赖: Phase 3完成（有历史数据）

5. **Phase 5: 前端集成 (1-2周)**
   - 时间线组件、轨迹可视化、导出功能
   - 依赖: Phase 4完成（API可用）

6. **Phase 6: 性能优化 (1周)**
   - 物化视图、缓存策略、分区归档
   - 依赖: Phase 4完成（有查询可优化）

---

### From PITFALLS.md: Critical Risks & Prevention

**Critical Pitfalls (必须避免)**:

1. **Pitfall 1: 无限制数据库增长**
   - **现象**: 每次采集全量存储，单设备1000 MAC每小时采集产生870万条记录/年
   - **预防**: 仅记录变更（增量快照），预计变更率5%（节省99%存储）
   - **检测**: 监控数据库增长率，应<10%总MAC数/采集周期

2. **Pitfall 2: 变更检测逻辑错误（MAC flapping）**
   - **现象**: MAC频繁移动时，检测逻辑遗漏移动事件或产生重复告警
   - **预防**: 复合键变更检测（MAC+VLAN+接口），MAC地址格式规范化
   - **代码模式**: 使用`map[string]MACState` key为`normalizedMAC:VLAN`进行O(1)查找

3. **Pitfall 3: 历史查询性能崩溃**
   - **现象**: 查询90天历史时全表扫描，超时>30秒
   - **预防**: 复合索引(device_id, collected_at) + 月度分区裁剪 + BRIN索引
   - **性能目标**: 90天查询<2秒，1年查询<5秒

4. **Pitfall 4: 与现有采集流程的竞态条件**
   - **现象**: 历史服务读取"旧"数据时采集服务正在更新，导致丢失变更
   - **预防**: 事务隔离（READ COMMITTED），在同一事务内完成读取-检测-删除-插入-记录历史
   - **关键**: 先记录历史再删除当前状态

5. **Pitfall 5: MAC地址格式不一致**
   - **现象**: 不同厂商格式（Cisco `aaaa.bbbb.cccc`, Huawei `aaaa-bbbb-cccc`）导致查询失败
   - **预防**: 数据库层强制规范化（`BeforeSave` hook）或使用PostgreSQL MACADDR类型

6. **Pitfall 6: 历史事件缺少设备上下文**
   - **现象**: 设备重命名或删除后，历史数据显示"Unknown Device"
   - **预防**: 历史表存储设备名称快照（非外键），而非依赖JOIN查询

**Moderate Pitfalls:**

- **时区问题**: 统一使用UTC存储，前端展示时转换
- **批量导出性能**: 实现流式导出，限制最大导出范围（30天）
- **API无限查询**: 实施速率限制，强制要求时间范围参数

**Phase-Specific Warnings:**

| Phase | 风险 | 缓解措施 |
|-------|------|----------|
| **Phase 12: 数据模型** | Pitfall 1 (无限制增长), Pitfall 6 (缺少上下文) | 变更检测 + 设备快照，自动分区清理 |
| **Phase 12: 采集集成** | Pitfall 2 (变更检测), Pitfall 4 (竞态条件) | 事务隔离，复合键比较，内存变更检测 |
| **Phase 13: 查询层** | Pitfall 3 (性能崩溃), Pitfall 5 (格式不一致) | 复合索引，MACADDR类型，模型层强制规范化 |

---

## Implications for Roadmap

### Suggested Phase Structure

基于研究分析，建议将v1.5 MAC地址历史数据管理拆分为4个子阶段：

#### **Phase 12: 数据模型与采集集成 (2-3周)**

**目标**: 建立历史数据基础设施，最小化侵入集成到现有采集流程

**交付成果**:
- 分区表`sys_device_mac_history` + BRIN索引
- `MACHistoryService`变更检测逻辑
- 修改`MACCollectionService`集成历史记录
- 自动清理定时任务（保留13个月）

**包含功能**:
- MAC变更快照记录（P0）
- 端口历史查询（P0）
- 数据导出（P1）

**避免的陷阱**:
- Pitfall 1: 全量存储 → 实施增量快照
- Pitfall 4: 竞态条件 → 使用事务隔离

**里程碑**: 采集周期产生历史记录，查询端口历史正常

---

#### **Phase 13: 查询层与异常检测 (2-3周)**

**目标**: 提供强大的历史查询能力和智能异常检测

**交付成果**:
- MAC轨迹查询API（P0）
- 异常检测规则引擎（P1）
- 前端时间线组件（Ant Design Timeline）
- 前端轨迹可视化（ECharts）

**包含功能**:
- MAC轨迹查询（P0）
- MAC异常告警（P1）
- 历史统计Dashboard（P1）

**避免的陷阱**:
- Pitfall 3: 查询性能崩溃 → 复合索引 + 分区裁剪
- Pitfall 5: 格式不一致 → MAC规范化 + MACADDR类型

**里程碑**: 可视化展示MAC移动路径，异常事件自动告警

---

#### **Phase 14: 前端与用户体验 (1-2周)**

**目标**: 完善前端交互，提供直观的历史数据探索体验

**交付成果**:
- MAC历史查询页面（时间范围筛选、分页）
- 轨迹回放组件（时间滑块控制）
- 异常事件面板（严重程度标签、批量确认）
- 导出功能优化（支持筛选条件）

**包含功能**:
- 用户界面增强（P1）
- 数据导出优化（P1）

**避免的陷阱**:
- Pitfall 9: 批量导出超时 → 限制导出范围，后台任务生成

**里程碑**: 用户可自主查询历史数据，无需SQL

---

#### **Phase 15: 性能优化与生产就绪 (1-2周)**

**目标**: 优化大规模数据场景下的性能，确保生产环境稳定性

**交付成果**:
- 物化视图（常用查询预聚合）
- 查询结果缓存（Redis）
- 分区归档策略（自动删除旧分区）
- 性能监控与告警

**包含功能**:
- 连接时长统计（P1）
- 热力图可视化（P1）
- 生产环境部署

**避免的陷阱**:
- Pitfall 3: 大数据量查询 → 物化视图 + 缓存
- Pitfall 1: 长期存储增长 → 分区归档（13个月策略）

**里程碑**: 10M历史记录查询<2秒，系统稳定运行30天

---

### Research Flags (需要额外研究的阶段)

| Phase | 研究需求 | 原因 |
|-------|----------|------|
| **Phase 12** | 无需额外研究 | 架构模式清晰，依赖现有系统 |
| **Phase 13** | 异常检测规则引擎 | 需定义具体阈值（如"1小时内移动>3次"） |
| **Phase 14** | 前端时间线组件选型 | Ant Design Timeline vs 自定义组件 |
| **Phase 15** | 生产环境数据量验证 | 需实际MAC表大小分布数据调优分区策略 |

**可以跳过研究的阶段**:
- Phase 12: 数据库模式已基于PostgreSQL最佳实践
- Phase 12: 变更检测算法基于成熟的Event Sourcing模式

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| **Stack** | HIGH | 无新技术引入，现有PostgreSQL 18功能已验证 |
| **Features** | HIGH | 基于行业标准MAC追踪功能（Cisco Prime, SolarWinds） |
| **Architecture** | HIGH | 基于现有代码库分析，Handler-Service模式成熟 |
| **Pitfalls** | HIGH | 识别的12个 pitfalls 均有明确预防措施和代码模式 |
| **Query Performance** | MEDIUM | 需实际数据量验证（假设中等规模网络） |
| **Storage Estimates** | LOW | 基于假设（100设备×500 MAC/设备），实际场景差异大 |

---

## Gaps to Address

**需要后续阶段解决的问题:**

1. **数据量基线**: 目标网络的MAC地址表实际大小分布？（影响分区策略）
2. **查询模式**: 安全团队最常使用的历史查询场景？（影响索引优化）
3. **合规要求**: 是否有监管要求MAC历史保留时长？（影响清理策略）
4. **厂商输出一致性**: 华为/H3C/锐捷/迈普的MAC表格式是否一致？（影响解析器鲁棒性）
5. **采集频率**: 生产环境的MAC采集间隔计划？（影响增长率估算）

**可以在实现过程中解决的问题:**
- 异常检测规则阈值（可通过用户反馈调优）
- 前端UI细节（可通过用户测试迭代）
- 分区归档策略（可根据实际增长调整）

---

## Integration with Existing System

### 复用现有组件

| 组件 | 用途 | 集成方式 |
|------|------|----------|
| **MACCollectionService** | 采集触发点 | 依赖注入`MACHistoryService` |
| **LLDP服务** | 排除上行端口 | 历史查询时过滤上行接口 |
| **过滤规则配置** | 历史数据继承过滤 | 变更检测时考虑过滤规则 |
| **Excel导出框架** | 历史数据导出 | 复用`excelize v2.10.0` |
| **定时调度器** | 自动清理任务 | 使用`internal/scheduler/` |
| **Swagger文档** | API文档生成 | 复用现有`swaggo/swag` |

### 新增组件

| 组件 | 位置 | 职责 |
|------|------|------|
| **MACHistoryService** | `internal/services/mac_history_service.go` | 变更检测、历史记录、时态查询 |
| **MACHistoryHandler** | `internal/api/v1/network/mac_history_handler.go` | 6个历史查询端点 |
| **前端页面** | `ruoyi-react-frontend/src/pages/network/mac-history/` | 历史查询、轨迹可视化、异常面板 |

---

## Success Criteria

**功能完成度**:
- [ ] MAC变更自动记录（appeared/disappeared/moved/vlan_changed）
- [ ] 端口历史查询（时间范围过滤，分页支持）
- [ ] MAC轨迹追踪（跨设备/端口迁移路径）
- [ ] 异常事件检测（频繁移动、异常时间）
- [ ] 数据导出（Excel格式，支持筛选）
- [ ] 自动数据清理（可配置保留期）

**性能指标**:
- [ ] 90天历史查询 < 2秒
- [ ] 1年历史查询 < 5秒
- [ ] 采集周期延迟增加 < 10%（历史记录开销）
- [ ] 数据库增长率 < 10% 总MAC数/采集周期

**质量指标**:
- [ ] 零丢失变更事件（并发场景）
- [ ] 历史数据可读性（设备删除后）
- [ ] MAC地址格式一致性（100%规范化）
- [ ] 查询API可用性（无全表扫描）

---

## Sources

### High Confidence Sources
- **现有代码库分析**:
  - `internal/services/mac_collection_service.go` (采集流程)
  - `internal/models/device_mac_address.go` (数据模型)
  - `internal/core/db/migrations/019_add_performance_indexes.sql` (索引模式)
  - `internal/core/db/migrations/020_drop_mac_unique_index.sql` (历史准备)

- **PostgreSQL官方文档**:
  - Partitioning (分区表)
  - BRIN Indexes (块范围索引)
  - JSONB Data Type (异常元数据)

### Medium Confidence Sources
- **行业最佳实践**:
  - Cisco Prime MAC Address Tracking
  - SolarWinds Network Performance Monitor
  - ManageEngine OpUtils MAC Tracker

- **时序数据库模式**:
  - Event Sourcing patterns
  - Temporal tables (SQL Server参考)

### Low Confidence Sources (需验证)
- **企业MAC表规模统计** (需要实际环境数据)
- **特定厂商MAC输出格式** (需要实际设备测试)

---

## Next Steps

1. **立即执行**: Phase 12 - 数据库Schema设计（无依赖，可立即开始）
2. **并行准备**: Phase 13 - 前端原型设计（与后端并行）
3. **后续研究**: 收集生产环境MAC表规模数据（优化分区策略）

**准备就绪**: 所有研究文件已完成，可进入需求定义阶段。
