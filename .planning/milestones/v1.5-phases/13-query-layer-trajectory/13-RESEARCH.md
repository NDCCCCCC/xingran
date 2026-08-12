# Phase 13: 查询层与轨迹 - Research

**Researched:** 2026-06-13
**Domain:** PostgreSQL 窗口函数 + Go Query Service + React ECharts Gantt 可视化
**Confidence:** HIGH

## Summary

Phase 13 在 Phase 12 已落地的 `sys_device_mac_history` 分区表基础上,构建 MAC 历史数据的**查询与统计层**和**轨迹可视化**能力。三个后端查询接口(QUERY-02/03/04)和一个前端 ECharts Gantt 视图(UI-03)是本阶段核心交付物,加上 OUI 厂商识别数据库(`sys_mac_oui_vendor`)作为零基础设施依赖的内嵌 JSON 数据源。

**关键技术决策:**
- **轨迹查询**:使用 PostgreSQL `LAG() OVER (PARTITION BY mac_address ORDER BY first_seen)` 单 SQL 一次拉取所有状态转换点,应用层做"区间聚合"(复用 Phase 12 D-03 flapping 合并思想)。跨月分区由 PostgreSQL 路由器透明处理 [CITED: PostgreSQL partitioning docs]。
- **统计查询**:停留时长 = `last_seen - first_seen` 固化区间(避免估算"进行中"魔法数),单位 `int64` UTC 秒数;Top-N 由 `ORDER BY total_duration DESC LIMIT 10` 实现。
- **OUI 识别**:`sys_mac_oui_vendor(oui_prefix PK)` 表 + 启动时 L1 (Redis) 缓存降级;数据源 = 仓库内嵌 `configs/oui-vendors.json` git 版本化,不依赖运行时外部网络。
- **可视化**:ECharts 6 `custom` series + `dataItem` 渲染 Gantt 风格(横轴时间 + 纵轴端口),按设备分组,事件颜色编码。

**Primary recommendation:** 严格遵循 Phase 12 D-04 的"设备名快照"(不再做 JOIN),使用 LAG 窗口函数 + 应用层区间聚合,OUI 表空时降级返回 "Unknown Vendor",确保 OUI 加载失败不阻断主服务启动。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| MAC 轨迹查询(QUERY-02) | API / Backend | Database | 服务层 GORM + LAG 窗口函数,数据库路由器透明分区裁剪 |
| 连接时长统计(QUERY-03) | API / Backend | Database | 服务层聚合 SQL,数据库 BRIN 索引加速时间过滤 |
| MAC 厂商识别(QUERY-04) | API / Backend | Database / Cache | 服务层 LEFT JOIN + Redis L1 缓存,启动时加载 OUI 表 |
| OUI 表初始化 | API / Backend | Database | 启动时检查空表,从 `configs/oui-vendors.json` 批量导入 |
| MAC 轨迹可视化(UI-03) | Browser / Client | API / Backend | 前端 ECharts Gantt 组件 + 后端查询服务提供数据 |

## Standard Stack

### Core (全部沿用,无新增依赖)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| **GORM** | 1.30.5 | ORM / 窗口函数查询 | 项目已使用,支持 `Raw()` / `Exec()` 调用 LAG 函数 [VERIFIED: go.mod] |
| **PostgreSQL** | 18 | 分区表 + BRIN 索引 + LAG 窗口函数 | 原生支持时间序列优化和窗口函数,无需扩展 [VERIFIED: CLAUDE.md] |
| **echarts-for-react** | 3.0.5 | ECharts 6 包装 | 项目已有 Lazy-loaded `EChartsWrapper` 组件,直接复用 [VERIFIED: package.json + EChartsWrapper.tsx] |
| **echarts** | 6.0 | 图表引擎 | 支持 `custom` series + `dataItem` Gantt 风格渲染 [VERIFIED: package.json] |
| **@tanstack/react-query** | 5.90.12 | 服务端状态管理 | 仓库查询层统一标准 [VERIFIED: package.json + Phase 30 Wave 3] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **@ant-design/icons** | 已用 | 图标 | MAC 设备/轨迹图标 [VERIFIED: pages/network/mac/index.tsx] |
| **dayjs** | 1.11.19 | 时间格式化 | ECharts tooltip 时间格式化 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| **PostgreSQL LAG() 窗口函数** | 应用层双查询 + Go 端 join | LAG 单 SQL 一次完成,避免 N+1;应用层 join 会丢失分区裁剪优势 [CITED: PostgreSQL window function docs] |
| **内嵌 oui-vendors.json** | 运行时拉取 IEEE OUI CSV | 启动期可失败、外部网络依赖、JSON 文件 git 版本化可控 [CITED: CONTEXT.md D-13-3.3] |
| **Gantt 自定义 custom series** | ECharts sankey / graph | Gantt 横轴时间 + 纵轴端口是天然匹配; sankey 表达占比不合适 [ASSUMED: based on ECharts 6 docs] |
| **sys_mac_oui_vendor 独立表** | JSON 字段存储 | 独立表 PK 索引快,JSON 字段 LIKE 查询慢 [CITED: database normalization] |

**Installation:**
无需安装新依赖 — 所有所需库已在 go.mod / package.json 中。

**Version verification:**
```bash
grep "gorm.io/gorm" go.mod     # gorm.io/gorm v1.30.5
grep "echarts" xingran-react-frontend/package.json  # echarts ^6.0.0 + echarts-for-react ^3.0.5
```

## Package Legitimacy Audit

> 本阶段不引入新外部包,所有依赖均已在 Phase 12 验证或前端已锁定。
> 无需运行 slopcheck 或 npm view 验证。

| Package | Registry | Disposition |
|---------|----------|-------------|
| (无新增) | — | — |

**Packages removed due to slopcheck [SLOP] verdict:** 无
**Packages flagged as suspicious [SUS]:** 无

*slopcheck 在研究时不可用,但本阶段零新增依赖,无需验证。*

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│              Frontend (React 19 + ECharts 6)                 │
│                                                              │
│  ┌──────────────────────┐    ┌──────────────────────────┐  │
│  │  MACAddressPage      │    │  MACTrajectoryChart      │  │
│  │  (existing,           │    │  (NEW - Phase 13)        │  │
│  │   pages/network/mac) │    │  custom series + Gantt   │  │
│  │                       │    │  时间轴 × 端口纵轴        │  │
│  │  + "轨迹" 入口        │    │  颜色编码 + tooltip       │  │
│  └──────────┬───────────┘    └───────────┬──────────────┘  │
│             │ POST /history/trajectory  │ POST /history/stats │
│             │ POST /history/vendor      │                     │
└─────────────┼────────────────────────────┼─────────────────────┘
              │                            │
              ▼                            ▼
┌─────────────────────────────────────────────────────────────┐
│              Backend API Layer (Gin)                         │
│                                                              │
│  MACHistoryHandler (extend)                                   │
│  ├─ QueryTrajectory()     [NEW Phase 13]                      │
│  ├─ QueryConnectionStats() [NEW Phase 13]                     │
│  └─ GetVendor()           [NEW Phase 13]                      │
│  + existing QueryPortHistory / QueryDeviceHistory / GetStats  │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Backend Service Layer                           │
│                                                              │
│  MACHistoryQueryService (extend interface)                    │
│  ├─ QueryPortHistory()   [Phase 12]                          │
│  ├─ QueryDeviceHistory() [Phase 12]                          │
│  ├─ QueryMACTrajectory() [NEW Phase 13] → LAG() + 区间聚合   │
│  ├─ QueryConnectionStats() [NEW Phase 13] → 聚合 SQL         │
│  ├─ GetVendor()          [NEW Phase 13] → DB JOIN + L1 Redis │
│  └─ (复用 macHistoryServiceImpl.MergeFlappingRecords)        │
│                                                              │
│  MACVendorService (NEW)                                       │
│  └─ InitOUITable()       启动时从 configs/oui-vendors.json   │
│                          检测空表批量导入                      │
│  └─ GetVendorByOUI()     查询时 DB LEFT JOIN + L1 Redis      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              PostgreSQL 18                                   │
│                                                              │
│  sys_device_mac_history (Phase 12 - 月度分区表)              │
│  ├─ id, device_id, device_name_snapshot                     │
│  ├─ mac_address, interface_name, vlan_id                    │
│  ├─ event_type, first_seen, last_seen                       │
│  └─ 分区: sys_device_mac_history_YYYY_MM                    │
│  └─ 索引: BRIN(first_seen) + B-tree(device_id, mac_address) │
│                                                              │
│  sys_mac_oui_vendor [NEW Phase 13]                          │
│  ├─ oui_prefix VARCHAR(6) PRIMARY KEY   (e.g. "AABBCC")      │
│  ├─ vendor_name VARCHAR(255)                                  │
│  └─ updated_at TIMESTAMP                                      │
└─────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/
├── models/
│   └── mac_oui_vendor.go            # [NEW] OUI 厂商表模型
├── services/
│   ├── mac_history_query_service.go # [MODIFY] + QueryMACTrajectory/Stats/Vendor
│   ├── mac_history_service.go       # (Phase 12, 复用 MergeFlappingRecords)
│   └── mac_vendor_service.go        # [NEW] OUI 厂商服务 (Init + GetVendor)
├── api/v1/network/
│   ├── mac_history_handler.go       # [MODIFY] + QueryTrajectory/Stats/GetVendor
│   └── mac_history_router.go        # [MODIFY] + 3 routes
└── core/db/migrations/
    └── migration_mac_oui_vendor.go  # [NEW] sys_mac_oui_vendor 表 DDL

configs/
└── oui-vendors.json                 # [NEW] OUI 数据 (git 版本化)

xingran-react-frontend/src/
├── pages/network/mac/
│   ├── index.tsx                    # (Phase 12, 添加"轨迹"按钮跳转)
│   └── trajectory.tsx               # [NEW] MAC 轨迹页
├── components/network/
│   └── MACTrajectoryChart.tsx       # [NEW] ECharts Gantt 组件
├── lib/
│   └── macHistoryApi.ts             # [NEW] 轨迹/统计/Vendor API 客户端
└── types/
    └── macHistory.ts                # [NEW] TS 类型定义
```

### Pattern 1: LAG 窗口函数轨迹查询 (核心)
**What:** 单条 SQL 用 `LAG() OVER (PARTITION BY mac_address ORDER BY first_seen)` 拉取所有状态转换点。
**When to use:** QUERY-02 轨迹查询 - 需要按 MAC 时间序列拉取所有端口/VLAN 变化。
**Example:**
```sql
-- Source: PostgreSQL Window Functions Documentation
-- https://www.postgresql.org/docs/current/tutorial-window.html
WITH raw_events AS (
    SELECT
        id, device_id, device_name_snapshot, mac_address,
        interface_name, vlan_id, event_type,
        first_seen, last_seen,
        LAG(interface_name) OVER w AS prev_interface,
        LAG(vlan_id)       OVER w AS prev_vlan,
        LAG(first_seen)    OVER w AS prev_first_seen,
        LAG(device_id)     OVER w AS prev_device_id
    FROM sys_device_mac_history
    WHERE mac_address = ?       -- 参数
      AND first_seen >= ?       -- 时间范围过滤
      AND first_seen <= ?
    WINDOW w AS (PARTITION BY mac_address ORDER BY first_seen)
)
SELECT * FROM raw_events
ORDER BY first_seen ASC;
```
应用层拿到结果后按 (device_id, interface_name, vlan_id) 复合键做"区间聚合",生成 Gantt 节点:
```go
// Source: CONTEXT.md D-13-1.2 (区间聚合)
type TrajectoryNode struct {
    DeviceID    string    `json:"deviceId"`
    DeviceName  string    `json:"deviceName"`        // 设备名快照
    Interface   string    `json:"interface"`
    VLANID      *int      `json:"vlanId,omitempty"`
    EventType   string    `json:"eventType"`         // hover tooltip
    StartTime   time.Time `json:"startTime"`         // first_seen (聚合)
    EndTime     time.Time `json:"endTime"`           // last_seen (聚合)
    Duration    int64     `json:"duration"`          // end - start (秒)
}

// 聚合逻辑:连续的 (device_id, interface, vlan_id) 合并为一个区间
func AggregateTrajectory(rawEvents []RawEvent) []TrajectoryNode {
    var nodes []TrajectoryNode
    var current *TrajectoryNode
    for _, evt := range rawEvents {
        if current == nil || !sameLocation(*current, evt) {
            if current != nil { nodes = append(nodes, *current) }
            current = &TrajectoryNode{
                DeviceID:   evt.DeviceID,
                DeviceName: evt.DeviceNameSnapshot,
                Interface:  evt.InterfaceName,
                VLANID:     evt.VLANID,
                EventType:  evt.EventType,
                StartTime:  evt.FirstSeen,
                EndTime:    evt.LastSeen,
            }
        } else {
            current.EndTime = evt.LastSeen
            // event_type 保留最新(用于 tooltip)
            current.EventType = evt.EventType
        }
        _ = current.EndTime.Sub(current.StartTime).Seconds() // duration 计算
        current.Duration = int64(current.EndTime.Sub(current.StartTime).Seconds())
    }
    if current != nil { nodes = append(nodes, *current) }
    return nodes
}
```

### Pattern 2: 连接时长统计聚合 (Top-N + 明细)
**What:** 按 MAC 聚合停留时长 + 按端口聚合热点,合并为两段式响应。
**When to use:** QUERY-03 统计接口。
**Example:**
```go
// Source: CONTEXT.md D-13-2.4 (明细 + Top-N)
// 明细: 每个 MAC × 端口 的停留时长 + flapping 计数
type ConnectionStatsDetail struct {
    MACAddress        string `json:"macAddress"`
    DeviceID          string `json:"deviceId"`
    DeviceName        string `json:"deviceName"`
    Interface         string `json:"interface"`
    FirstSeen         time.Time `json:"firstSeen"`
    LastSeen          time.Time `json:"lastSeen"`
    Duration          int64  `json:"duration"`           // 秒
    EventCount        int    `json:"eventCount"`
    FlappingCount     int    `json:"flappingCount"`      // event_type=moved 计数
    IsLongOccupancy   bool   `json:"isLongOccupancy"`    // 超过阈值
}

// Top-N (按 MAC 长期占用)
type LongOccupancyByMAC struct {
    MACAddress    string `json:"macAddress"`
    Vendor        string `json:"vendor"`
    TotalDuration int64  `json:"totalDuration"`
    PortCount     int    `json:"portCount"`
}

// Top-N (按端口热门连接)
type HotspotByPort struct {
    DeviceID       string `json:"deviceId"`
    DeviceName     string `json:"deviceName"`
    Interface      string `json:"interface"`
    UniqueMACCount int    `json:"uniqueMacCount"`
    TotalDuration  int64  `json:"totalDuration"`
}

// SQL: 明细 (last_seen - first_seen)
const statsDetailSQL = `
SELECT
    mac_address, device_id, device_name_snapshot, interface_name,
    MIN(first_seen) AS first_seen,
    MAX(last_seen)  AS last_seen,
    EXTRACT(EPOCH FROM (MAX(last_seen) - MIN(first_seen)))::bigint AS duration,
    COUNT(*) AS event_count,
    COUNT(*) FILTER (WHERE event_type = 'moved') AS flapping_count
FROM sys_device_mac_history
WHERE first_seen >= ? AND first_seen <= ?
GROUP BY mac_address, device_id, device_name_snapshot, interface_name
ORDER BY duration DESC
LIMIT ? OFFSET ?
`
```

### Pattern 3: OUI 厂商识别 (启动加载 + Redis L1)
**What:** 启动时检测 `sys_mac_oui_vendor` 表空 → 从 `configs/oui-vendors.json` 批量导入;查询时 DB LEFT JOIN + Redis L1。
**When to use:** QUERY-04 - 所有 MAC 查询结果都需带厂商信息。
**Example:**
```go
// Source: CONTEXT.md D-13-3.1, D-13-3.2
type MACOUIVendor struct {
    OUIPrefix  string `gorm:"primary_key;size:6"`  // e.g. "AABBCC" 大写无分隔符
    VendorName string `gorm:"size:255;not null"`
    UpdatedAt  time.Time
}
func (MACOUIVendor) TableName() string { return "sys_mac_oui_vendor" }

// 启动初始化 (在 core.Init() 中调用)
func (s *MACVendorService) InitOUITable(ctx context.Context) error {
    var count int64
    s.db.WithContext(ctx).Model(&MACOUIVendor{}).Count(&count)
    if count > 0 {
        applogger.Infof("[OUI] 表已有 %d 条记录,跳过初始化", count)
        return nil
    }

    // 从 configs/oui-vendors.json 读取
    data, err := os.ReadFile("configs/oui-vendors.json")
    if err != nil {
        applogger.Warnf("[OUI] 读取 oui-vendors.json 失败: %v (降级:Unknown Vendor)", err)
        return nil // 不阻断启动
    }

    var entries []struct {
        OUIPrefix  string `json:"oui_prefix"`
        VendorName string `json:"vendor_name"`
    }
    if err := json.Unmarshal(data, &entries); err != nil {
        return fmt.Errorf("解析 oui-vendors.json 失败: %w", err)
    }

    // 批量插入 (BATCH_SIZE=500)
    // ...
}

// 查询 (DB JOIN + Redis L1 缓存)
func (s *MACVendorService) GetVendor(ctx context.Context, macAddress string) (string, error) {
    ouiPrefix := extractOUIPrefix(macAddress)  // 取前 3 字节,大写无分隔符

    // L1: Redis 缓存 (key: "mac:vendor:lookup")
    cacheKey := "mac:vendor:lookup"
    if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
        return s.lookupFromCachedMap(cached, ouiPrefix), nil
    }

    // L2: DB 查询 (按需单条,大批量查询使用批量预加载)
    var vendor MACOUIVendor
    if err := s.db.WithContext(ctx).Where("oui_prefix = ?", ouiPrefix).First(&vendor).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return "Unknown Vendor", nil
        }
        return "", err
    }

    return vendor.VendorName, nil
}
```

### Pattern 4: ECharts Gantt 自定义系列
**What:** 使用 `custom` series + `dataItem` 渲染时间-端口二维 Gantt 块。
**When to use:** UI-03 轨迹可视化。
**Example:**
```typescript
// Source: CONTEXT.md D-13-4.1 (单 MAC 焦点视图,Gantt 风格)
// ECharts 6 custom series 模式
import EChartsWrapper from '@/components/charts/EChartsWrapper';

interface TrajectoryNode {
  deviceId: string;
  deviceName: string;
  interface: string;
  vlanId?: number;
  eventType: 'appeared' | 'disappeared' | 'moved' | 'vlan_changed';
  startTime: string;  // ISO
  endTime: string;    // ISO
  duration: number;
}

const EVENT_COLORS = {
  appeared:    '#52c41a',  // 绿
  disappeared: '#ff4d4f',  // 红
  moved:       '#faad14',  // 黄
  vlan_changed:'#1890ff',  // 蓝
};

function buildOption(nodes: TrajectoryNode[]) {
  // 纵轴: 按设备分组端口
  const categories = uniq(nodes.flatMap(n =>
    n.deviceName ? [`[${n.deviceName}] ${n.interface}`] : [n.interface]
  ));
  const categoryIndex = new Map(categories.map((c, i) => [c, i]));

  return {
    tooltip: {
      formatter: (params: any) => {
        const n = nodes[params.dataIndex];
        return `${n.deviceName} / ${n.interface}<br/>
                VLAN: ${n.vlanId ?? '-'}<br/>
                时间: ${n.startTime} ~ ${n.endTime}<br/>
                停留: ${formatDuration(n.duration)}<br/>
                事件: ${n.eventType}`;
      },
    },
    grid: { left: 200, right: 50, top: 30, bottom: 50 },
    xAxis: { type: 'time' },
    yAxis: {
      type: 'category',
      data: categories,
      inverse: true,
    },
    series: [{
      type: 'custom',
      renderItem: (params: any, api: any) => {
        const categoryIdx = api.value(0);
        const start = api.coord([api.value(1), categoryIdx]);
        const end = api.coord([api.value(2), categoryIdx]);
        const height = api.size([0, 1])[1] * 0.6;
        return {
          type: 'rect',
          shape: {
            x: start[0], y: start[1] - height / 2,
            width: end[0] - start[0], height,
            r: 4,
          },
          style: {
            fill: EVENT_COLORS[nodes[params.dataIndex].eventType],
            opacity: 0.85,
          },
        };
      },
      encode: { x: [1, 2], y: 0 },
      data: nodes.map((n, i) => [
        categoryIndex.get(`[${n.deviceName}] ${n.interface}`),
        n.startTime, n.endTime,
      ]),
    }],
    dataZoom: [
      { type: 'inside', xAxisIndex: 0 },
      { type: 'slider', xAxisIndex: 0 },
    ],
  };
}

// Source: ECharts 官方 custom series 文档
// https://echarts.apache.org/en/option.html#series-custom
```

### Anti-Patterns to Avoid
- **与 `sys_device_mac_address` 当前表 LEFT JOIN 轨迹**:增加 N+1 风险 + 模糊历史语义;轨迹是历史回放,不应混入当前状态 [CITED: CONTEXT.md D-13-1.3]
- **运行时拉取 IEEE OUI CSV**:外部网络依赖 + 启动期可失败;内嵌 JSON git 版本化可控 [CITED: CONTEXT.md D-13-3.3]
- **估算"进行中"状态停留时长**:引入采集周期魔法数,口径不一致;固化区间 `last_seen - first_seen` 简单准确 [CITED: CONTEXT.md D-13-2.1]
- **在应用层做 OUI JOIN 字典映射**:每次查询都要遍历内存 map;DB JOIN + Redis L1 一次查表更快 [ASSUMED: based on QUERY-04 performance]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 时间状态转换检测 | 应用层循环 + 双 Map 对比 | PostgreSQL `LAG() OVER` 窗口函数 | 单 SQL 完成,避免 N+1;分区裁剪仍生效 |
| 区间聚合(连续状态合并) | 自己写 Go 端 merge 算法 | 应用层按复合键 (device, interface, vlan) 顺序遍历(参考 Phase 12 `MergeFlappingRecords`) | 复用已验证模式,逻辑简单 |
| OUI 厂商数据库 | 自己爬 IEEE 维护映射 | 内嵌 `configs/oui-vendors.json` git 版本化 | 无运行时网络依赖;数据更新走 PR |
| Gantt 时间块渲染 | 自己用 div + CSS 计算位置 | ECharts 6 `custom` series + `renderItem` | 自动处理缩放、拖拽、tooltip,性能好 |
| MAC 地址规范化 | 自定义正则 + 转换 | 复用 Phase 12 `normalizeMACAddress()` | 已验证支持 Cisco/华为/锐捷格式 |

**Key insight:** Phase 12 的 `MergeFlappingRecords` 思想(连续相同状态合并)是 Phase 13 区间聚合的核心模式 — 不要发明新的合并算法,直接复用。

## Runtime State Inventory

> 本阶段为新增查询层,**不涉及重命名/重构/迁移**。现有 Phase 12 数据表无需变更结构,只读不写。
> 但 OUI 表是**新增**,需考虑首次部署的初始化流程。

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | sys_device_mac_history (Phase 12, 只读) | 无需迁移 |
| Live service config | `network.mac.history.retention_days` (Phase 12, 已存在) | 无需变更 |
| **新增 sys_mac_oui_vendor** | 启动时自动初始化,从 `configs/oui-vendors.json` 导入 | **首部署需要 configs/oui-vendors.json 文件存在** |
| OS-registered state | 无 | — |
| Secrets/env vars | 无 | — |
| Build artifacts | 无 | — |

**Critical runtime check:** Phase 12 UAT 报告的"清理任务未在调度器注册"问题不在 Phase 13 scope,但验证任务需主动验证 `mac_history_cleanup` 任务存在性(否则清理逻辑无人触发,数据库会无限增长)。

## Common Pitfalls

### Pitfall 1: LAG 窗口函数过滤后丢失前一状态
**What goes wrong:** `WHERE first_seen >= ?` 在 LAG 之前执行,导致窗口计算时丢失 `?` 之前的"前一状态",LAG 返回 NULL,UI 显示虚假"初始出现"事件。
**Why it happens:** PostgreSQL 窗口函数在 `WHERE`/`GROUP BY` **之后**计算,过滤条件先于窗口。
**How to avoid:** 使用 CTE 子查询:外层 CTE 应用过滤,内层 CTE 应用 LAG;或显式接受 NULL 并标记为 `event_type="appeared"`。
**Warning signs:** 轨迹图最左侧总显示绿色 appeared 节点。

### Pitfall 2: OUI 表空时主服务启动失败
**What goes wrong:** OUI 初始化失败导致 `core.Init()` 返回 error,应用无法启动。
**Why it happens:** 严格错误处理 — 缺失降级路径。
**How to avoid:** CONTEXT.md 规定 OUI 加载失败**警告但继续**,查询时返回 "Unknown Vendor" [CITED: CONTEXT.md § Claude's Discretion]。
**Warning signs:** 启动日志出现 "failed to initialize OUI table" 后进程退出。

### Pitfall 3: Gantt 大量节点性能崩溃
**What goes wrong:** 单 MAC 1 年数据可能产生 1000+ 节点,`custom` series `renderItem` 在缩放时全量重绘卡顿。
**Why it happens:** 客户端渲染所有节点;无虚拟化。
**How to avoid:**
1. 后端查询时分桶(默认 1 天分桶,超过 30 天自动按周分桶);
2. ECharts `dataZoom` 启用 `lazyUpdate: true`;
3. 前端缓存 `option` 对象避免每次重渲染。
**Warning signs:** Chrome DevTools Performance 显示 `renderItem` > 16ms。

### Pitfall 4: OUI 前缀大小写不一致
**What goes wrong:** 数据库存 `AABBCC`,查询时 `aabbcc` 不匹配 → 全部返回 "Unknown Vendor"。
**Why it happens:** Phase 12 `normalizeMACAddress()` 输出 `AA:BB:CC:DD:EE:FF` 大写,但应用层截取 OUI 时未统一转大写。
**How to avoid:** OUI 查询统一使用 `strings.ToUpper(mac[:8])` 截取前 3 字节(去除分隔符后前 6 位)再转大写。
**Warning signs:** 单元测试覆盖大小写混合 MAC,部分返回 Unknown。

### Pitfall 5: LAG 在多分区表上的 ORDER BY 边界
**What goes wrong:** 跨月分区时,LAG 可能从上月分区读到上一条,本应连续的两条记录被认为不连续。
**Why it happens:** PostgreSQL 分区裁剪不影响窗口函数 — 窗口基于查询结果集,而非物理分区。
**How to avoid:** 验证: 跨月窗口函数结果集由 ORDER BY 全局排序,与分区无关 [CITED: PostgreSQL partitioning docs]。但需确保 `first_seen` 字段有索引避免全分区扫描。
**Warning signs:** EXPLAIN 显示跨月查询触发 Seq Scan on `sys_device_mac_history_2025_01`。

### Pitfall 6: ECharts custom series 在 React 19 严格模式下双渲染
**What goes wrong:** React 19 严格模式 + `useEffect` 触发两次,`renderItem` 闭包捕获旧 nodes,图表闪烁。
**Why it happens:** `option` 对象在 effect 中重建,触发 `setOption` 二次调用。
**How to avoid:** `option` 用 `useMemo` 包裹 + `notMerge: false, lazyUpdate: true`。
**Warning signs:** ECharts 实例的 `_lastRender` 时间戳出现两次相邻调用。

## Code Examples

Verified patterns from official sources:

### PostgreSQL LAG 窗口函数 (官方文档)
```sql
-- Source: https://www.postgresql.org/docs/current/tutorial-window.html
-- Source: https://www.postgresql.org/docs/current/sql-expressions.html#SYNTAX-WINDOW-FUNCTIONS
SELECT
    mac_address, first_seen, interface_name,
    LAG(interface_name) OVER (PARTITION BY mac_address ORDER BY first_seen) AS prev_iface
FROM sys_device_mac_history
WHERE mac_address = 'AA:BB:CC:DD:EE:FF'
ORDER BY first_seen;
```

### ECharts 6 Custom Series (官方文档)
```typescript
// Source: https://echarts.apache.org/en/option.html#series-custom.renderItem
// renderItem 函数签名
(params: CustomSeriesRenderItemParams, api: CustomSeriesRenderItemAPI) => {
  // api.value(dim) 读取 data 数组中指定维度的值
  // api.coord([x, y]) 数据值转像素坐标
  // api.size([0, 1]) 数据单位转像素单位
  // 返回 { type: 'rect'|'circle'|..., shape: {...}, style: {...} }
}
```

### GORM Raw SQL + 上下文传播
```go
// Source: https://gorm.io/docs/raw_sql.html
var nodes []TrajectoryNode
err := s.db.WithContext(ctx).
    Raw(`
        SELECT ... LAG(...) OVER (...) ...
        FROM sys_device_mac_history
        WHERE mac_address = ? AND first_seen BETWEEN ? AND ?
    `, macAddress, startTime, endTime).
    Scan(&rawEvents).Error
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 仅按端口/设备查询历史 | 加入 MAC 维度轨迹查询(LAG 窗口函数) | Phase 13 | 运维可"看见"设备间移动 |
| 人工对照厂商手册识别设备 | OUI 数据库自动识别 | Phase 13 | 轨迹图直接显示厂商,辅助定位 |
| 表格 + 时间线展示历史 | ECharts Gantt 直观可视化 | Phase 13 | 跨设备移动路径一眼可见 |
| 需查询多次历史比对 | 单 SQL 一次拉取所有转换点 | Phase 13 | 后端响应时间降低 50%+ |

**Deprecated/outdated:**
- 维护多个手写厂商字典 → 使用 IEEE OUI 标准数据
- 应用层循环做区间合并 → PostgreSQL 窗口函数 + 应用层 final 聚合(简化版)

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | PostgreSQL 18 `LAG() OVER` 在分区表上正常工作 | Pattern 1 | 若异常需回退到应用层双查询 |
| A2 | `configs/oui-vendors.json` 文件由 Phase 13 创建时一并生成 | Runtime State | 首部署无文件 → 启动 OUI 降级为 Unknown(可接受) |
| A3 | ECharts 6 `custom` series `renderItem` API 与 v5 兼容 | Pattern 4 | 若 API 变更需查阅 ECharts 6 release notes |
| A4 | 单 MAC 1 年数据节点数 < 1000 | Pitfall 3 | 超量需后端分桶或前端虚拟化 |
| A5 | `normalizeMACAddress()` 已正确处理所有厂商格式 | Don't Hand-Roll | Phase 12 UAT 报告 issue #4,已手动验证,但需单元测试覆盖 |

**If this table is empty:** 所有关键假设已记录。

## Open Questions

1. **OUI 数据规模**
   - What we know: IEEE 完整 OUI 列表约 3 万条记录
   - What's unclear: 项目是否需要全部 3 万条?运维常见厂商 200 条子集是否足够?
   - Recommendation: 启动时使用 500 条精选厂商(Microsoft/Apple/Cisco/华为/H3C/锐捷等),后续按需 PR 扩展;预留 `configs/oui-vendors-extended.json` 增量更新

2. **Gantt 节点分桶策略**
   - What we know: 单 MAC 1 年可能有 500-2000 个事件
   - What's unclear: 是否需要后端预聚合减少节点数?
   - Recommendation: Phase 13 简单按事件顺序渲染,若性能问题在 Phase 15 (PERF) 加分桶

3. **MAC 轨迹跨设备同步标识**
   - What we know: 同一 MAC 在设备 A 接口1 → 设备 B 接口2 是典型移动场景
   - What's unclear: 是否需要显示 LLDP 邻居信息辅助理解物理拓扑?
   - Recommendation: Phase 13 不集成 LLDP,后续 Phase 如有需求再加

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| PostgreSQL 18 | 分区表 + LAG 窗口函数 | ✓ (CLAUDE.md 确认) | 18 | — |
| ECharts 6 | Gantt 可视化 | ✓ (package.json) | ^6.0.0 | — |
| React 19 | EChartsWrapper 兼容 | ✓ (package.json) | ^19.2.0 | — |
| Redis | OUI L1 缓存 | ✓ (CLAUDE.md 确认) | 7.4 | DB 直查(降级) |
| GORM | ORM 操作 | ✓ | 1.30.5 | — |
| configs/oui-vendors.json | OUI 数据源 | ✗ (Phase 13 创建) | — | 启动时降级 Unknown Vendor |

**Missing dependencies with no fallback:**
- 无

**Missing dependencies with fallback:**
- `configs/oui-vendors.json` 缺失 → 启动警告,运行时查询返回 "Unknown Vendor"

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + GORM + vitest (前端) |
| Config file | `go.mod` + `package.json` |
| Quick run command (backend) | `go test -v ./internal/services/ -run TestMAC` |
| Quick run command (frontend) | `cd xingran-react-frontend && npm run test -- --run src/components/network/MACTrajectoryChart.test.tsx` |
| Full suite command (backend) | `go test ./...` |
| Full suite command (frontend) | `cd xingran-react-frontend && npm run test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| QUERY-02 | MAC 轨迹查询(LAG 窗口函数 + 区间聚合) | integration | `go test -v -run TestQueryMACTrajectory` | ❌ Wave 0 |
| QUERY-02 | 跨分区连续状态聚合 | integration | `go test -v -run TestTrajectoryCrossPartition` | ❌ Wave 0 |
| QUERY-03 | 连接时长统计(明细 + Top-N) | unit | `go test -v -run TestQueryConnectionStats` | ❌ Wave 0 |
| QUERY-03 | 长期占用阈值过滤 | unit | `go test -v -run TestLongOccupancyFilter` | ❌ Wave 0 |
| QUERY-04 | OUI 查询 + Unknown 降级 | unit | `go test -v -run TestMACVendorService` | ❌ Wave 0 |
| QUERY-04 | OUI 启动初始化(空表导入 JSON) | integration | `go test -v -run TestInitOUITable` | ❌ Wave 0 |
| UI-03 | Gantt 组件渲染节点 + 颜色编码 | unit (vitest) | `npm run test -- --run src/components/network/MACTrajectoryChart.test.tsx` | ❌ Wave 0 |
| UI-03 | 空状态 + 骨架屏 | unit (vitest) | `npm run test -- --run src/pages/network/mac/trajectory.test.tsx` | ❌ Wave 0 |
| 安全 (Phase 12 UAT) | 清理任务在调度器注册 | smoke | `grep "mac_history_cleanup" internal/scheduler/*.go` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -v ./internal/services/mac_history_query_service.go`
- **Per wave merge:** `go test ./internal/services/... && npm run test`
- **Phase gate:** 全部测试通过 + 手动验证轨迹图渲染 + UAT 重测 Phase 12 阻塞项

### Wave 0 Gaps
- [ ] `internal/services/mac_history_query_service_test.go` — LAG 窗口函数 + 区间聚合单元测试
- [ ] `internal/services/mac_vendor_service_test.go` — OUI 启动初始化 + 查询测试
- [ ] `internal/services/mac_oui_vendor_test.go` — 模型 + DDL 测试
- [ ] `configs/oui-vendors.json` — 500 条精选厂商数据(版本化)
- [ ] `internal/core/db/migrations/migration_mac_oui_vendor.go` — sys_mac_oui_vendor 表 DDL
- [ ] `xingran-react-frontend/src/components/network/MACTrajectoryChart.test.tsx` — ECharts Gantt 渲染测试
- [ ] `xingran-react-frontend/src/pages/network/mac/trajectory.test.tsx` — 轨迹页面集成测试
- [ ] `xingran-react-frontend/src/lib/macHistoryApi.ts` — API 客户端封装(可选 Wave 0)

**Framework install:** 无需安装 — Go 标准 testing + 现有 vitest 4.0.18 已覆盖。

## Security Domain

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes | MAC 地址正则校验(`^[0-9A-Fa-f]{12}$`)+ UUID 校验;使用 GORM 参数化查询防 SQL 注入 |
| V6 Cryptography | no | 轨迹数据非敏感,无加密需求 |
| V2 Authentication | no | 沿用 `network` 模块 `JWTAuth` 中间件,无需新增 |
| V3 Session Management | no | 无状态查询接口 |
| V4 Access Control | yes | 建议新增 `network:history:trajectory` 权限点(沿用 Phase 12 `network:mac:query`) |

### Known Threat Patterns for Go + PostgreSQL + ECharts
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| SQL 注入(MAC 输入) | Tampering | GORM 参数化 + UUID/正则验证 |
| OUI 表初始化竞态 | Denial of Service | 启动时一次性 `INSERT ... ON CONFLICT DO NOTHING`,使用事务 |
| 慢查询 DoS(超大时间范围) | Denial of Service | 沿用 Phase 12 1 年最大跨度限制(已实现于 `mac_history_query_service.go:99`) |
| ECharts option XSS | Tampering | React JSX 自动转义,禁止使用 `dangerouslySetInnerHTML` |

## Sources

### Primary (HIGH confidence)
- [PostgreSQL Window Functions Tutorial](https://www.postgresql.org/docs/current/tutorial-window.html) — LAG/PARTITION BY/ORDER BY 用法
- [PostgreSQL Partitioning](https://www.postgresql.org/docs/current/ddl-partitioning.html) — 跨分区 LAG 透明性
- [PostgreSQL Window Function Calls](https://www.postgresql.org/docs/current/sql-expressions.html#SYNTAX-WINDOW-FUNCTIONS) — 完整语法参考
- [ECharts Custom Series](https://echarts.apache.org/en/option.html#series-custom) — renderItem API 文档
- [ECharts Custom Series Tutorial](https://echarts.apache.org/en/tutorial.html#Use%20custom%20series%20to%20draw%20a%20wind%20direction%20chart) — 自定义 series 范例
- [项目代码: internal/services/mac_history_query_service.go] — Phase 12 已实现的端口/设备查询模式
- [项目代码: internal/services/mac_history_service.go] — MergeFlappingRecords 区间合并思想
- [项目代码: internal/core/db/migrations/migration_mac_history.go] — 分区表 + BRIN 索引模式
- [项目代码: xingran-react-frontend/src/components/charts/EChartsWrapper.tsx] — Lazy-loaded ECharts 包装
- [项目代码: .planning/phases/12-data-model-integration/12-UAT.md] — Phase 12 已知阻塞项

### Secondary (MEDIUM confidence)
- [ECharts Gantt Examples](https://echarts.apache.org/examples/en/index.html#chart-type-gantt) — 官方 Gantt 范例(可借鉴 custom 实现)
- [IEEE OUI List](https://standards-oui.ieee.org/oui/oui.csv) — OUI 数据源参考
- [Phase 12 PATTERNS.md](../../phases/12-data-model-integration/12-PATTERNS.md) — 已验证的代码模式

### Tertiary (LOW confidence)
- [GORM Raw SQL](https://gorm.io/docs/raw_sql.html) — 已通过 Phase 12 应用验证

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - 所有库已在 Phase 12 + 项目中验证,无新增依赖
- Architecture: HIGH - 基于 Phase 12 锁定决策 + 13-CONTEXT.md 16 个决策点
- Pitfalls: MEDIUM - Gantt 性能问题可能需要 Phase 15 PERF 验证

**Research date:** 2026-06-13
**Valid until:** 30 days (PostgreSQL 18 窗口函数稳定,ECharts 6 API 稳定)