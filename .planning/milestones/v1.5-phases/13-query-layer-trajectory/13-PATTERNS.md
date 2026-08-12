# Phase 13: 查询层与轨迹 - Pattern Map

**Mapped:** 2026-06-13
**Files analyzed:** 16
**Analogs found:** 14 / 16
**Phase boundary:** 后端查询层(QUERY-02/03/04) + 前端 ECharts Gantt 可视化(UI-03)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/mac_history_query_service.go` (MODIFY) | service | request-response / query | `internal/services/mac_history_query_service.go` (self-extend) | exact |
| `internal/services/mac_vendor_service.go` (NEW) | service | request-response | `internal/services/mac_history_partition.go` | role-match |
| `internal/models/mac_oui_vendor.go` (NEW) | model | CRUD | `internal/models/device_mac_history.go` | exact |
| `internal/core/db/migrations/migration_mac_oui_vendor.go` (NEW) | config / DDL | batch | `internal/core/db/migrations/migration_mac_history.go` | exact |
| `configs/oui-vendors.json` (NEW) | config | file-I/O | `configs/config.yaml` | role-match |
| `internal/api/v1/network/mac_history_handler.go` (MODIFY) | handler | request-response | `internal/api/v1/network/mac_history_handler.go` (self-extend) | exact |
| `internal/api/v1/network/mac_history_router.go` (MODIFY) | route | request-response | `internal/api/v1/network/mac_history_router.go` (self-extend) | exact |
| `internal/core/core.go` (MODIFY) | config | boot | `internal/core/core.go` (self-extend: PartitionService) | exact |
| `internal/services/mac_history_query_service_test.go` (NEW) | test | test | `internal/services/mac_collection_service_test.go` | role-match |
| `xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx` (NEW) | component | rendering | `xingran-react-frontend/src/components/charts/EChartsWrapper.tsx` (wraps echarts-for-react) | role-match |
| `xingran-react-frontend/src/pages/network/mac/trajectory.tsx` (NEW) | component / page | request-response | `xingran-react-frontend/src/pages/network/mac/index.tsx` | role-match |
| `xingran-react-frontend/src/lib/macHistoryApi.ts` (NEW) | utility | request-response | `xingran-react-frontend/src/lib/opsApi.ts` (CRUD factory) | role-match |
| `xingran-react-frontend/src/router/index.tsx` (MODIFY) | route | request-response | (dynamic menu routes) | partial |
| `xingran-react-frontend/src/store/authStore.ts` (MODIFY) | store | request-response | (permission) | role-match |
| `xingran-react-frontend/src/utils/normalizeMACAddress.ts` (EXTEND/CREATE) | utility | transform | (Phase 12 `normalizeMACAddress` in `internal/services/mac_history_service.go`) | partial |
| `internal/services/mac_vendor_service_test.go` (NEW, Wave 0) | test | test | `internal/services/mac_collection_service_test.go` | role-match |

## Pattern Assignments

### `internal/services/mac_history_query_service.go` (MODIFY: extend interface + impl)

**Analog:** 自身文件 `internal/services/mac_history_query_service.go` (lines 55-69, 177-276) — 已有的接口契约和 `QueryPortHistory` / `QueryDeviceHistory` 实现是扩展 `QueryMACTrajectory` / `QueryConnectionStats` / `GetVendor` 的最直接参照。

**Interface pattern** (lines 55-69):
```go
type MACHistoryQueryService interface {
    QueryPortHistory(ctx context.Context, req *PortHistoryQuery) (*MACHistoryQueryResult, error)
    QueryDeviceHistory(ctx context.Context, req *DeviceHistoryQuery) (*MACHistoryQueryResult, error)
}
```

**Impl struct + constructor** (lines 61-69):
```go
type macHistoryQueryServiceImpl struct {
    db *gorm.DB
}

func NewMACHistoryQueryService(db *gorm.DB) MACHistoryQueryService {
    return &macHistoryQueryServiceImpl{db: db}
}
```

**Time-range DoS protection pattern** (lines 99-117) — 复用此 `maxQueryRange = 365 * 24 * time.Hour` 限制:
```go
// WR-02 fix: 限制查询时间范围最大为1年，防止DoS攻击
const maxQueryRange = 365 * 24 * time.Hour // 最多查询1年

if req.StartTime != "" && req.EndTime != "" {
    startTime, err := time.Parse(time.RFC3339, req.StartTime)
    if err != nil {
        return nil, fmt.Errorf("无效的开始时间格式: %w", err)
    }
    endTime, err := time.Parse(time.RFC3339, req.EndTime)
    if err != nil {
        return nil, fmt.Errorf("无效的结束时间格式: %w", err)
    }
    if endTime.Sub(startTime) > maxQueryRange {
        return nil, fmt.Errorf("查询时间跨度过大，最大允许 %d 天", int(maxQueryRange.Hours()/24))
    }
    query = query.Where("first_seen >= ?", startTime).
        Where("first_seen <= ?", endTime)
}
```

**UUID validation + ctx + GORM Raw** (lines 73-87, 142-149):
```go
// WR-01 fix: 验证设备ID格式
if _, err := uuid.Parse(req.DeviceID); err != nil {
    return nil, fmt.Errorf("无效的设备ID格式: %w", err)
}
// ... 默认值 ...
query := s.db.WithContext(ctx).Table("sys_device_mac_history")
// ...
var records []models.DeviceMACHistory
if err := query.
    Order("first_seen DESC").
    Limit(req.PageSize).
    Offset(offset).
    Find(&records).Error; err != nil {
    applogger.Errorf("[MAC历史查询] 查询记录失败: %v", err)
    return nil, fmt.Errorf("查询历史记录失败: %w", err)
}
```

**Context propagation / error wrapping** (与 `internal/services/mac_history_service.go` lines 242-247 一致):
```go
if err := s.db.WithContext(ctx).Create(&historyRecords).Error; err != nil {
    return fmt.Errorf("批量插入MAC历史记录失败: %w", err)
}
applogger.Infof("[MAC历史] %s: 记录 %d 条变更事件", device.DeviceName, len(historyRecords))
```

**新增方法实现路径** — Phase 13 在同接口加:
- `QueryMACTrajectory(ctx, *MACTrajectoryQuery) (*MACTrajectoryResult, error)` — `db.Raw()` 调 LAG 窗口函数 + 应用层 `AggregateTrajectory` 区间聚合(参考 `MergeFlappingRecords` lines 263-376 的复合键合并思想)。
- `QueryConnectionStats(ctx, *ConnectionStatsQuery) (*ConnectionStatsResponse, error)` — 单条 `Raw` 聚合 SQL + Top-N `Raw` 查询。
- `GetVendor(ctx, macAddress) (string, error)` — 委托 `macVendorService.GetVendorByOUI()`(单一职责,本文件只编排)。

---

### `internal/services/mac_vendor_service.go` (NEW: OUI 厂商服务)

**Analog:** `internal/services/mac_history_partition.go` (lines 15-38) — 接口 + 私有 impl + 构造函数标准模式;以及 `internal/services/mac_history_service.go` (lines 49-57) 同样的 impl 私有化模式。

**Service interface + impl pattern** (来自 `mac_history_partition.go` lines 15-38):
```go
type PartitionService interface {
    CreateMonthlyPartition(ctx context.Context, year int, month int) error
    EnsurePartitionsExist(ctx context.Context, monthsAhead int) error
    DropExpiredPartitions(ctx context.Context) error
    GetRetentionDays(ctx context.Context) int
}

type partitionServiceImpl struct {
    db *gorm.DB
}

func NewPartitionService(db *gorm.DB) PartitionService {
    return &partitionServiceImpl{db: db}
}
```

**Phase 13 OUI 服务接口草案:**
```go
type MACVendorService interface {
    InitOUITable(ctx context.Context) error
    GetVendorByOUI(ctx context.Context, macAddress string) (string, error)
}

type macVendorServiceImpl struct {
    db    *gorm.DB
    cache cache.Cache // 注入 pkg/cache.Cache 以支持 Redis L1
}
```

**Cache wrapper pattern** — 参考 `internal/services/data_cache_service.go` 的 `CacheProvider` 思路;但更轻量:Phase 13 OUI 只需 `Cache.Get` / `Cache.Set` 直调,无需封装 `CacheProvider`。

**File read + JSON unmarshal pattern** (不存在直接 analog,套用 Go 标准库):
```go
data, err := os.ReadFile("configs/oui-vendors.json")
if err != nil {
    applogger.Warnf("[OUI] 读取 oui-vendors.json 失败: %v (降级:Unknown Vendor)", err)
    return nil // 不阻断启动 (CONTEXT.md § Claude's Discretion)
}
var entries []OUIVendorEntry
if err := json.Unmarshal(data, &entries); err != nil {
    return fmt.Errorf("解析 oui-vendors.json 失败: %w", err)
}
```

**Redis cache lookup pattern** (来自 `pkg/cache/redis.go` lines 67-78 — `prefix` 自动加,传 key 时不要加 `xingran:`):
```go
// Cache.Get: 错误 ErrNotFound 表示未命中,业务正常处理
func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
    result, err := r.client.Get(ctx, r.buildKey(key)).Result()
    if err == redis.Nil {
        return "", ErrNotFound
    }
    return result, err
}
// 调用方: if cached, err := s.cache.Get(ctx, "mac:vendor:lookup"); errors.Is(err, cache.ErrNotFound) { ... }
```

---

### `internal/models/mac_oui_vendor.go` (NEW: GORM 模型)

**Analog:** `internal/models/device_mac_history.go` (lines 1-46) — UUID 主键 + `gorm` tag 模式 + `TableName` + `BeforeCreate` 钩子。

**Standard model pattern** (lines 22-46):
```go
type DeviceMACHistory struct {
    ID                  string       `gorm:"type:uuid;primary_key" json:"id"`
    DeviceID            string       `gorm:"type:uuid;not null" json:"deviceId"`
    DeviceNameSnapshot  string       `gorm:"size:100" json:"deviceNameSnapshot"`
    MACAddress          string       `gorm:"size:30;not null" json:"macAddress"`
    // ...
    EventType           MACEventType `gorm:"size:20;not null" json:"eventType"`
    FirstSeen           time.Time    `gorm:"not null" json:"firstSeen"`
    CreatedAt           time.Time    `json:"createdAt"`
}

func (DeviceMACHistory) TableName() string { return "sys_device_mac_history" }

func (d *DeviceMACHistory) BeforeCreate(tx *gorm.DB) error {
    if d.ID == "" || d.ID == "00000000-0000-0000-0000-000000000000" {
        d.ID = uuid.New().String()
    }
    return nil
}
```

**Phase 13 OUI 模型草案(精简,字符串主键):**
```go
package models

import "time"

// MACOUIVendor MAC地址OUI厂商表
type MACOUIVendor struct {
    OUIPrefix  string    `gorm:"primary_key;size:6" json:"ouiPrefix"` // "AABBCC" 大写无分隔符
    VendorName string    `gorm:"size:255;not null" json:"vendorName"`
    UpdatedAt  time.Time `json:"updatedAt"`
}

func (MACOUIVendor) TableName() string { return "sys_mac_oui_vendor" }
```

注:OUI 是 6 字节十六进制字符串主键,无需 UUID — 简化 GORM tag。

---

### `internal/core/db/migrations/migration_mac_oui_vendor.go` (NEW: DDL)

**Analog:** `internal/core/db/migrations/migration_mac_history.go` (lines 11-117) — `isPostgreSQL` 检查 + `CREATE TABLE IF NOT EXISTS` + `Exec` 模式;以及 `migration_148_create_ops_asset_table.go` (lines 1-126) — 简单 GORM migration(检查 HasTable + 单条 SQL)。

**PostgreSQL-aware migration pattern** (来自 `migration_mac_history.go` lines 11-52):
```go
func isPostgreSQL(db *gorm.DB) bool {
    return db.Config.Dialector.Name() == "postgres"
}

func CreateMACHistoryTable(db *gorm.DB) error {
    if !isPostgreSQL(db) {
        applogger.Infof("[迁移] MAC历史表跳过创建（非PostgreSQL数据库）")
        return nil
    }
    // ...
    createMainTableSQL := `
    CREATE TABLE IF NOT EXISTS sys_device_mac_history (
        id UUID NOT NULL DEFAULT gen_random_uuid(),
        ...
    ) PARTITION BY RANGE (first_seen);
    `
    if err := db.Exec(createMainTableSQL).Error; err != nil {
        return fmt.Errorf("创建MAC历史主表失败: %w", err)
    }
}
```

**Simple GORM migration pattern** (来自 `migration_148_create_ops_asset_table.go` lines 10-50):
```go
func Migrate148CreateOpsAssetTable(db *gorm.DB) error {
    log.Println("Running migration 148: Create ops_asset table")
    if db.Migrator().HasTable(&OpsAsset{}) {
        log.Println("Table ops_asset already exists, skipping migration 148...")
        return nil
    }
    sql := `CREATE TABLE IF NOT EXISTS ops_asset (...);`
    if err := db.Exec(sql).Error; err != nil {
        return err
    }
    return nil
}
```

**Phase 13 OUI migration 草案(简单 GORM 表,不分区):**
```go
package migrations

import (
    applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
    "gorm.io/gorm"
)

func CreateMACOUIVendorTable(db *gorm.DB) error {
    applogger.Infof("[迁移] 开始创建 sys_mac_oui_vendor 表")
    sql := `
    CREATE TABLE IF NOT EXISTS sys_mac_oui_vendor (
        oui_prefix  VARCHAR(6) PRIMARY KEY,
        vendor_name VARCHAR(255) NOT NULL,
        updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    CREATE INDEX IF NOT EXISTS idx_oui_vendor_name ON sys_mac_oui_vendor(vendor_name);
    `
    if err := db.Exec(sql).Error; err != nil {
        return fmt.Errorf("创建 sys_mac_oui_vendor 表失败: %w", err)
    }
    applogger.Infof("[迁移] sys_mac_oui_vendor 表创建完成")
    return nil
}
```

注:OUI 表是静态小表(~500-3000 条),**不需要分区 / BRIN**,沿用简单 B-tree 索引。

---

### `configs/oui-vendors.json` (NEW: 500 OUI 条目)

**Analog:** `configs/config.yaml` (YAML 格式) — 仓库内嵌的静态配置数据,git 版本化。

**JSON 数组结构(参考 RESEARCH.md Pattern 3):**
```json
[
  { "oui_prefix": "AABBCC", "vendor_name": "Example Corp" },
  { "oui_prefix": "001A11", "vendor_name": "Google, Inc." },
  ...
]
```

**OUI 规范化要求**(来自 CONTEXT.md D-13-3.1 + RESEARCH.md Pitfall 4):
- `oui_prefix` 格式:6 字符大写无分隔符(如 `AABBCC` / `001A11`)
- 500 条精选厂商(Microsoft / Apple / Cisco / 华为 / H3C / 锐捷 / Intel / Samsung / Dell / HP 等)
- `vendor_name` 限制 255 字符(与模型 gorm tag 一致)

**File-read pattern in service:**
```go
data, err := os.ReadFile("configs/oui-vendors.json")
// 路径相对项目根目录(运行 `go run cmd/main.go` 时 cwd = 项目根)
```

---

### `internal/api/v1/network/mac_history_handler.go` (MODIFY: extend with new endpoints)

**Analog:** 自身文件 `internal/api/v1/network/mac_history_handler.go` (lines 12-86) — 已有 `QueryPortHistory` / `QueryDeviceHistory` / `GetStats` 三个 handler 模板,直接追加 `QueryTrajectory` / `QueryConnectionStats` / `GetVendor`。

**Handler struct + constructor** (lines 12-20):
```go
type MACHistoryHandler struct {
    historyQueryService mac_history_query_service.MACHistoryQueryService
}

func NewMACHistoryHandler(historySvc mac_history_query_service.MACHistoryQueryService) *MACHistoryHandler {
    return &MACHistoryHandler{historyQueryService: historySvc}
}
```

**Standard handler pattern** (lines 33-46, `QueryPortHistory`):
```go
func (h *MACHistoryHandler) QueryPortHistory(c *gin.Context) {
    var req mac_history_query_service.PortHistoryQuery
    if !responseHelpers.HandleJSONBinding(c, &req) {
        return
    }
    result, err := h.historyQueryService.QueryPortHistory(c.Request.Context(), &req)
    if !responseHelpers.HandleServiceError(c, err, "查询端口MAC历史记录") {
        return
    }
    response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}
```

**Swagger annotation pattern** (lines 23-32):
```go
// @Summary 查询端口MAC地址历史记录
// @Description 按设备ID和接口名查询MAC地址变化历史，支持时间范围过滤和分页
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body mac_history_query_service.PortHistoryQuery true "查询条件"
// @Success 200 {object} response.Response{data=mac_history_query_service.MACHistoryQueryResult}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/history/port [post]
```

**Phase 13 新增 handler 签名:**
- `QueryTrajectory(c *gin.Context)` — POST `/network/history/trajectory`,body: `MACTrajectoryQuery`
- `QueryConnectionStats(c *gin.Context)` — POST `/network/history/stats`(替换原 TODO `GetStats`)
- `GetVendor(c *gin.Context)` — POST `/network/mac/vendor`,body: `{ macAddress: string }`

**GetStats TODO 替换:** 原 `GetStats` (lines 83-86) 是 stub,Phase 13 实现 `QueryConnectionStats` 替换。
```go
// OLD (lines 83-86):
func (h *MACHistoryHandler) GetStats(c *gin.Context) {
    // TODO: 实现统计功能（Phase 13）
    response.Error(c, http.StatusNotImplemented, "统计功能尚未实现")
}
```

---

### `internal/api/v1/network/mac_history_router.go` (MODIFY: register 3 new routes)

**Analog:** 自身文件 `internal/api/v1/network/mac_history_router.go` (lines 10-24) — `SetupMACHistoryRouter` 模式直接追加路由。

**Router pattern** (lines 10-24):
```go
func SetupMACHistoryRouter(r *gin.RouterGroup, core *core.Core) {
    historyQueryService := mac_history_query_service.NewMACHistoryQueryService(core.GetDB())
    historyHandler := NewMACHistoryHandler(historyQueryService)

    r.POST("/history/port", historyHandler.QueryPortHistory)
    r.POST("/history/device", historyHandler.QueryDeviceHistory)
    r.POST("/history/stats", historyHandler.GetStats)

    applogger.Infof("[路由注册] MAC历史查询路由已注册: /history/port, /history/device, /history/stats")
}
```

**Phase 13 修改后:**
```go
func SetupMACHistoryRouter(r *gin.RouterGroup, core *core.Core) {
    // 历史查询服务
    historyQueryService := mac_history_query_service.NewMACHistoryQueryService(core.GetDB())
    // OUI 厂商服务(Phase 13 新增)
    vendorService := mac_history_query_service.NewMACVendorService(core.GetDB(), core.Cache)
    historyHandler := NewMACHistoryHandler(historyQueryService, vendorService)

    r.POST("/history/port", historyHandler.QueryPortHistory)
    r.POST("/history/device", historyHandler.QueryDeviceHistory)
    r.POST("/history/trajectory", historyHandler.QueryTrajectory)  // NEW
    r.POST("/history/stats", historyHandler.QueryConnectionStats)  // Phase 13 实现
    r.POST("/mac/vendor", historyHandler.GetVendor)                // NEW
    // ...
}
```

注:主路由 `internal/api/router.go:325-329` 已注册 `SetupMACHistoryRouter`,**Phase 13 无需修改 router.go**(已确认 Phase 12 UAT 阻塞项已修复)。

---

### `internal/core/core.go` (MODIFY: wire MACVendorService)

**Analog:** `internal/core/core.go` lines 78, 254-262 — 已有 `PartitionService` 注入和初始化模式。

**Partition service wiring pattern** (lines 78, 254-262):
```go
// Field declaration (line 78):
PartitionService services.PartitionService // MAC历史表分区管理服务

// Init (lines 254-262):
// 9.5. 初始化MAC历史分区管理服务（必须在调度器初始化之前）
c.PartitionService = services.NewPartitionService(c.GetDB())
if err := c.PartitionService.EnsurePartitionsExist(context.Background(), 2); err != nil {
    applogger.Warnf("初始化MAC历史分区失败: %v", err)
} else {
    applogger.Infof("MAC历史分区管理服务初始化完成")
}
```

**Phase 13 OUI 服务 wiring 草案:**
```go
// Field declaration (在 line 78 之后):
MACVendorService services.MACVendorService

// Init (在 line 262 之后,即分区初始化后):
// 9.6. 初始化MAC OUI厂商服务(Phase 13) — 失败不阻断启动
c.MACVendorService = services.NewMACVendorService(c.GetDB(), c.Cache)
if err := c.MACVendorService.InitOUITable(context.Background()); err != nil {
    applogger.Warnf("[OUI] 初始化OUI表失败: %v (降级为 Unknown Vendor)", err)
} else {
    applogger.Infof("MAC OUI厂商服务初始化完成")
}
```

**Core DI 模式**(整个 Phase 13 的依赖):
- `core.GetDB()` → `*gorm.DB`(主数据库)
- `core.Cache` → `cache.Cache`(Redis 或内存)
- 这两个字段是 Phase 12 已建立的,Phase 13 OUI 服务和 query 服务的输入。

---

### `internal/services/mac_history_query_service_test.go` (NEW: integration test)

**Analog:** `internal/services/mac_collection_service_test.go` (lines 1-80) — testify 风格,表驱动测试。

**Standard test pattern** (lines 1-65):
```go
package services

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/xingran-next/xingran-go-backend/internal/models"
)

func TestGetMACThreshold(t *testing.T) {
    service := &MACCollectionService{}
    tests := []struct {
        name     string
        device   *models.NetworkDevice
        expected int
    }{
        {name: "Router threshold", device: &models.NetworkDevice{DeviceType: models.DeviceTypeRouter}, expected: 500},
        // ...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := service.getMACThreshold(tt.device)
            assert.Equal(t, tt.expected, result, "Threshold mismatch for %s", tt.name)
        })
    }
}
```

**Phase 13 测试覆盖**(从 13-VALIDATION.md):
- `TestQueryMACTrajectory` — LAG 窗口函数 + 区间聚合
- `TestTrajectoryCrossPartition` — 跨分区连续状态
- `TestQueryConnectionStats` — 明细 + Top-N 两段式
- `TestLongOccupancyFilter` — `long_occupancy_threshold_days` 阈值过滤
- `TestMACVendorService` / `TestInitOUITable` — OUI 启动 + 查询降级

注:`/test/fixtures/` 目录已有 fixtures,Phase 13 复用其初始化测试 DB。

---

### `xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx` (NEW: ECharts Gantt)

**Analog:** `xingran-react-frontend/src/components/charts/EChartsWrapper.tsx` (lines 18-47) — 已有 `lazy(() => import('echarts-for-react'))` + `Suspense` 包装。

**ECharts lazy wrapper pattern** (lines 18-47):
```typescript
import { lazy, Suspense, forwardRef, type Ref, type ComponentProps } from 'react';
import { Spin } from 'antd';

const ReactECharts = lazy(() => import('echarts-for-react'));

type ReactEChartsProps = ComponentProps<typeof ReactECharts>;

const Loading = () => (
  <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 24, minHeight: 120 }}>
    <Spin tip="加载图表..." />
  </div>
);

export const EChartsWrapper = forwardRef<unknown, ReactEChartsProps>((props, ref) => (
  <Suspense fallback={<Loading />}>
    <ReactECharts {...props} ref={ref as Ref<unknown>} />
  </Suspense>
));
```

**Phase 13 MACTrajectoryChart 使用模式:**
```typescript
import EChartsWrapper from '@/components/charts/EChartsWrapper';

const option = useMemo(() => buildTrajectoryOption(nodes), [nodes]);

return (
  <EChartsWrapper
    option={option}
    style={{ height: 600 }}
    notMerge={false}
    lazyUpdate={true}  // RESEARCH.md Pitfall 6
  />
);
```

**`option` memo 化要求**(来自 RESEARCH.md Pitfall 6 — React 19 严格模式双渲染):
- 必须 `useMemo(() => buildOption(nodes), [nodes])` 包裹
- `notMerge: false, lazyUpdate: true` 避免闪烁
- 颜色编码常量:`appeared=#52c41a` / `disappeared=#ff4d4f` / `moved=#faad14` / `vlan_changed=#1890ff`

---

### `xingran-react-frontend/src/pages/network/mac/trajectory.tsx` (NEW: 轨迹页)

**Analog:** `xingran-react-frontend/src/pages/network/mac/index.tsx` (lines 1-100) — Ant Design `Table` / `Form` / `Input` / `Button` / `Card` / `Empty` 组合 + `useTableManager` hook 风格。

**Page pattern** (lines 16-100):
```typescript
import type { FC } from 'react';
import { Table, Button, Space, Form, Input, Select, Card, Row, Col, Statistic, Tag, message } from 'antd';
import { post } from '@/lib/api';
import { useTableManager } from '@/hooks/useTableManager';
import { withErrorHandling } from '@/utils/errorHandler';
// ...

const MACAddressPage: FC = () => {
  const [devices, setDevices] = useState<NetworkDevice[]>([]);
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();
  const { loading, data: macAddresses, searchForm, loadData: loadMACAddresses, handleSearch, handleReset, handleRefresh } =
    useTableManager<DeviceMACAddress>(
      async (params) => {
        const result = await post('/network/mac/list', { current: 1, pageSize: 10, ... });
        return result.data;
      },
      { externalPagination: { current: 1, pageSize: 10, setCurrent, setPageSize, setTotal } }
    );
  // ...
};
```

**Phase 13 trajectory 页结构草案:**
```typescript
const MACTrajectoryPage: FC = () => {
  const [macInput, setMacInput] = useState('');
  const [trajectory, setTrajectory] = useState<TrajectoryNode[] | null>(null);
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['mac-trajectory', macInput],
    queryFn: () => macHistoryApi.getTrajectory({ macAddress: macInput }),
    enabled: !!macInput,
  });
  // ...
  return (
    <Card>
      <Input.Search
        placeholder="输入 MAC 地址(支持 AA:BB:CC / aabbccddeeff 格式)"
        onSearch={(v) => setMacInput(normalizeMACAddress(v))}
      />
      {isLoading && <Skeleton active />}
      {error && <Alert type="error" message={error.message} />}
      {!isLoading && (!data || data.nodes.length === 0) && <Empty description="该 MAC 无历史轨迹" />}
      {data && data.nodes.length > 0 && <MACTrajectoryChart nodes={data.nodes} />}
    </Card>
  );
};
```

注:Phase 13 推荐 `useQuery`(React Query)而非 `useTableManager` — 因为轨迹是非分页的单一对象查询,无 `useTableManager` 的列表/分页场景。

---

### `xingran-react-frontend/src/lib/macHistoryApi.ts` (NEW: API 客户端)

**Analog:** `xingran-react-frontend/src/lib/opsApi.ts` (lines 1-80) — CRUD factory + `post` 包装。

**API factory pattern** (lines 1-58):
```typescript
import { post, get, postFormData } from './api';
import { getAccessToken } from '@/utils/authHelpers';
import type { BaseResponse, ... } from '@/types';

interface CrudApiConfig {
  basePath: string;
}

function createCrudApi<T>(config: CrudApiConfig) {
  const { basePath } = config;
  return {
    list: async (params: PageParams & Record<string, unknown>) => post<PageResponse<T>>(`${basePath}/list`, params),
    get: async (id: string) => post<T>(`${basePath}/${id}`, {}),
    create: async (data: Partial<T>) => post(basePath, data),
    update: async (id: string, data: Partial<T>) => post(`${basePath}/${id}/update`, data),
    delete: async (id: string) => post(`${basePath}/${id}/delete`, {}),
  };
}

export const buildingApi = createCrudApi<Building>({ basePath: '/ops/building' });
```

**Phase 13 macHistoryApi 草案(非标准 CRUD,用显式命名方法):**
```typescript
import { post } from './api';
import type { BaseResponse, MACTrajectory, ConnectionStats, MACVendor } from '@/types/macHistory';

export const macHistoryApi = {
  getPortHistory: (params: { deviceId: string; interfaceName?: string; startTime?: string; endTime?: string; current: number; pageSize: number }) =>
    post<PageResponse<MACHistoryRecord>>('/network/history/port', params),

  getTrajectory: (params: { macAddress: string; startTime?: string; endTime?: string }) =>
    post<MACTrajectoryResult>('/network/history/trajectory', params),

  getConnectionStats: (params: { startTime: string; endTime: string; topN?: number }) =>
    post<ConnectionStatsResult>('/network/history/stats', params),

  getVendor: (params: { macAddress: string }) =>
    post<MACVendor>('/network/mac/vendor', params),
};
```

注:Phase 13 API 端点是 `POST`(遵循项目约定 `POST /list`、`POST /:id`),即使查询也用 POST。`response.Success()` 解包由 `post` 函数内部完成(`@/lib/api.ts` 封装)。

---

### `xingran-react-frontend/src/router/index.tsx` (MODIFY: register trajectory route)

**Analog:** `xingran-react-frontend/src/router/componentLoader.tsx` (lines 33-38) — 动态 glob + lazy + 白名单。

**Component lazy pattern** (lines 33-38, 175-200):
```typescript
// import.meta.glob('/src/pages/**/index.tsx', { eager: false })
// 注意: glob 默认只匹配 index.tsx,trajectory.tsx 必须用 index.tsx 或调整 glob

// 创建 lazy 组件:
export function createLazyComponent(componentPath: string): ComponentType {
  // 标准化: 去掉 'pages/' 前缀(如果存在),添加 'pages/' 前缀,加 '.tsx'
  let normalizedPath = componentPath;
  if (normalizedPath.startsWith('/')) normalizedPath = normalizedPath.slice(1);
  if (normalizedPath.startsWith('pages/')) normalizedPath = normalizedPath.slice(6);
  if (!normalizedPath.startsWith('pages/')) normalizedPath = `pages/${normalizedPath}`;
  if (!normalizedPath.endsWith('.tsx')) normalizedPath += '.tsx';
  const fullPath = `/src/${normalizedPath}`;
  const moduleLoader = ComponentLoader.componentModules[fullPath];
  if (!moduleLoader) {
    console.error(`[createLazyComponent] Module not found: ${fullPath}`);
    return () => <div>页面加载失败</div>;
  }
  return lazy(moduleLoader);
}
```

**Phase 13 trajectory 路由方案 A(标准):** 创建 `xingran-react-frontend/src/pages/network/mac/trajectory/index.tsx` + 注册菜单(动态菜单由后端 `sys_menu` 表 + 权限点 `network:history:trajectory` 触发,见 DynamicRoutes)。

**Phase 13 trajectory 路由方案 B(显式):** 修改 `xingran-react-frontend/src/router/DynamicRoutes.tsx` 加静态 fallback 路由(只在 dynamic menu 加载失败时使用)。

**关键约束**(来自 `App.tsx` lines 1-55 + `componentLoader.tsx` line 33):
- 路由**由后端 `sys_menu` 表动态生成**;新页 = 新菜单项 + 新组件路径
- 菜单的 `component` 字段必须以 `network/mac/trajectory` 形式存储(去掉 `pages/` 前缀,自动添加)
- 文件必须命名为 `index.tsx`(glob 模式只匹配此);若想用 `trajectory.tsx`,需修改 `componentLoader.tsx` 的 glob 模式
- **推荐方案**:用 `pages/network/mac/trajectory/index.tsx` 文件名 + 在后端菜单加一项

**Phase 13 关联后端权限点**:
- `network:history:trajectory` — 轨迹查询权限(沿用 `network:mac:query` 组的 `RequirePermissions`)

---

### `xingran-react-frontend/src/store/authStore.ts` (MODIFY: 新增权限点)

**Analog:** 不需要单独修改 — 权限点由后端 `pkg/permission/permissions.go` + `sys_menu` 权限分配管理,前端只通过 `useAuthStore().permissions` 拿到权限列表。

**参考路径:**
- `pkg/permission/permissions.go` — 注册新权限点 `network:history:trajectory`(如不存在)
- 后端菜单 SQL migration — 新增 `sys_menu` 记录,`perms` 字段填 `network:history:trajectory`

**前端使用模式**(`pages/network/mac/trajectory/index.tsx`):
```typescript
import { useAuthStore } from '@/store/authStore';

const hasTrajectoryPermission = useAuthStore((s) => s.permissions.includes('network:history:trajectory'));
if (!hasTrajectoryPermission) return <Result status="403" title="403" subTitle="无轨迹查询权限" />;
```

---

### `xingran-react-frontend/src/utils/normalizeMACAddress.ts` (EXTEND/CREATE)

**Analog:** 后端 `internal/services/mac_history_service.go` (lines 59-103) `normalizeMACAddress` 函数 — Phase 12 私有函数,前端需要**复刻**为 TS 版本。

**Source pattern** (lines 68-103):
```go
func normalizeMACAddress(mac string) string {
    if mac == "" { return "" }
    mac = strings.TrimSpace(mac)
    if mac == "" { return "" }

    // 移除所有分隔符（.: -）
    normalized := strings.ReplaceAll(mac, ".", "")
    normalized = strings.ReplaceAll(normalized, ":", "")
    normalized = strings.ReplaceAll(normalized, "-", "")
    normalized = strings.ToUpper(strings.TrimSpace(normalized))

    // 验证格式：12个十六进制字符
    macPattern := regexp.MustCompile(`^[0-9A-F]{12}$`)
    if !macPattern.MatchString(normalized) {
        applogger.Warnf("[MAC历史] MAC地址格式无效: %s", mac)
        return mac
    }

    // 插入冒号分隔符：AA:BB:CC:DD:EE:FF
    var result strings.Builder
    for i := 0; i < 12; i += 2 {
        if i > 0 { result.WriteString(":") }
        result.WriteString(normalized[i : i + 2])
    }
    return result.String()
}
```

**Phase 13 TS 版本草案:**
```typescript
// src/utils/normalizeMACAddress.ts
export function normalizeMACAddress(mac: string): string {
  if (!mac) return '';
  const cleaned = mac.trim().replace(/[.:\-]/g, '').toUpperCase();
  if (!/^[0-9A-F]{12}$/.test(cleaned)) {
    console.warn('[MAC] Invalid MAC address format:', mac);
    return mac; // 格式不合法,返回原值
  }
  return cleaned.match(/.{2}/g)!.join(':'); // "AABBCCDDEEFF" -> "AA:BB:CC:DD:EE:FF"
}

// OUI 前缀提取(6 字符大写无分隔符)
export function extractOUIPrefix(mac: string): string {
  const normalized = normalizeMACAddress(mac);
  return normalized.replace(/[:]/g, '').substring(0, 6);
}
```

**支持输入格式(来自 CONTEXT.md § Specifics):**
- `AA:BB:CC:DD:EE:FF`
- `aabb.ccdd.eeff`(Cisco/Huawei)
- `aa-bb-cc-dd-ee-ff`
- `aabbccddeeff`(无分隔符)

---

## Shared Patterns

### Handler-Service 标准模式
**Source:** `internal/api/v1/network/mac_history_handler.go` lines 12-46 + `internal/services/mac_history_query_service.go` lines 55-69
**Apply to:** 所有新 handler 和 service 文件
```go
// Handler:
type MACHistoryHandler struct {
    historyQueryService mac_history_query_service.MACHistoryQueryService
    vendorService       mac_history_query_service.MACVendorService // Phase 13 新增
}
func NewMACHistoryHandler(historySvc, vendorSvc ...) *MACHistoryHandler { ... }

// Service:
type MACHistoryQueryService interface { /* 4-5 个方法 */ }
type macHistoryQueryServiceImpl struct { db *gorm.DB }
func NewMACHistoryQueryService(db *gorm.DB) MACHistoryQueryService { ... }
```

### Response 包装与错误处理
**Source:** `pkg/response/response.go` lines 74-100 + `pkg/response/handler_helpers.go` lines 11-27
**Apply to:** 所有 handler
```go
import (
    "github.com/xingran-next/xingran-go-backend/pkg/response"
    responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// 成功(普通):
response.Success(c, data)

// 成功(分页):
response.Page(c, result.List, result.Total, result.Current, result.PageSize)

// 错误:
response.Error(c, http.StatusBadRequest, "请求参数错误")
response.Error(c, http.StatusInternalServerError, "操作失败: " + err.Error())

// Handler 内部使用 helper:
if !responseHelpers.HandleJSONBinding(c, &req) { return }
if !responseHelpers.HandleServiceError(c, err, "操作描述") { return }
```

### Context 传播 + GORM Raw
**Source:** `internal/services/mac_history_query_service.go` lines 86-87, 142-149
**Apply to:** 所有 service 数据库操作
```go
s.db.WithContext(ctx)
s.db.WithContext(ctx).Raw(`SELECT ... LAG(...) OVER (...) ...`, params...).Scan(&dest)
```

### UUID 校验
**Source:** `internal/services/mac_history_query_service.go` lines 73-76
**Apply to:** 所有需要 UUID 输入的 service 方法
```go
import "github.com/google/uuid"
if _, err := uuid.Parse(req.DeviceID); err != nil {
    return nil, fmt.Errorf("无效的设备ID格式: %w", err)
}
```

### 错误包装 + 日志
**Source:** `internal/services/mac_history_query_service.go` lines 134-137, 147-149
**Apply to:** 所有 service 错误处理
```go
applogger.Errorf("[MAC历史查询] 查询总数失败: %v", err)
return nil, fmt.Errorf("查询历史记录总数失败: %w", err)
```

### Redis 缓存(降级到 DB)
**Source:** `pkg/cache/redis.go` lines 67-78 + `internal/services/data_cache_service.go` 整体
**Apply to:** OUI L1 缓存
```go
// 调用: cache.Get 返回 ErrNotFound 表示未命中,正常处理
import "github.com/xingran-next/xingran-go-backend/pkg/cache"

if cached, err := s.cache.Get(ctx, "mac:vendor:lookup"); err == nil && cached != "" {
    return lookupFromCache(cached, ouiPrefix), nil
} else if !errors.Is(err, cache.ErrNotFound) {
    applogger.Warnf("[OUI] Redis Get 失败,降级 DB: %v", err)
    // 降级:不返回,继续走 DB
}
// 降级路径
```

### 前端 React Query 模式
**Source:** `xingran-react-frontend/src/App.tsx` lines 16-25 (QueryClient 配置)
**Apply to:** `pages/network/mac/trajectory/index.tsx`
```typescript
import { useQuery } from '@tanstack/react-query';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 5 * 60 * 1000,    // 5 分钟 stale
      gcTime: 30 * 60 * 1000,      // 30 分钟 gc
    },
  },
});

const { data, isLoading, error } = useQuery({
  queryKey: ['mac-trajectory', macInput],
  queryFn: () => macHistoryApi.getTrajectory({ macAddress: macInput }),
  enabled: !!macInput,
});
```

### 前端 ECharts Lazy + useMemo
**Source:** `xingran-react-frontend/src/components/charts/EChartsWrapper.tsx` lines 18-47 + RESEARCH.md Pitfall 6
**Apply to:** `MACTrajectoryChart.tsx`
```typescript
import EChartsWrapper from '@/components/charts/EChartsWrapper';
const option = useMemo(() => buildTrajectoryOption(nodes), [nodes]); // 必须 memo 化
return <EChartsWrapper option={option} notMerge={false} lazyUpdate={true} style={{ height: 600 }} />;
```

### 时间范围 DoS 保护
**Source:** `internal/services/mac_history_query_service.go` lines 99-117
**Apply to:** 所有新查询方法(`QueryMACTrajectory` / `QueryConnectionStats`)
```go
const maxQueryRange = 365 * 24 * time.Hour // 1 年
if endTime.Sub(startTime) > maxQueryRange {
    return nil, fmt.Errorf("查询时间跨度过大,最大允许 %d 天", int(maxQueryRange.Hours()/24))
}
```

### 启动时降级加载(不阻断主服务)
**Source:** `internal/core/core.go` lines 254-262 (PartitionService 初始化)+ 178-183 (缓存初始化)
**Apply to:** OUI 服务初始化
```go
if err := c.MACVendorService.InitOUITable(ctx); err != nil {
    applogger.Warnf("[OUI] 初始化 OUI 表失败: %v (降级为 Unknown Vendor)", err)
    // 继续:不返回
} else {
    applogger.Infof("MAC OUI 厂商服务初始化完成")
}
```

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx` | component | rendering | 无现有 Gantt 组件,ECharts `custom` series `renderItem` 是新模式 — 使用 RESEARCH.md Pattern 4 |
| `configs/oui-vendors.json` | config | file-I/O | 无现有 OUI 数据,内容由 Phase 13 创建(500 条精选厂商,git 版本化) |
| `internal/services/mac_vendor_service.go` (完整版) | service | request-response | 无现有厂商/OUI 服务;接口+启动初始化+缓存降级模式是新的 — 参考 `mac_history_partition.go` 框架 |
| `xingran-react-frontend/src/lib/macHistoryApi.ts` | utility | request-response | 无现有 macHistory API 客户端;`opsApi.ts` 的 CRUD factory 不完全适用(POST + 非标准方法名) |

---

## Metadata

**Analog search scope:**
- `internal/services/mac_history_query_service.go` (QueryPortHistory / QueryDeviceHistory pattern)
- `internal/services/mac_history_service.go` (MergeFlappingRecords 区间聚合思想)
- `internal/services/mac_history_partition.go` (Service interface + private impl pattern)
- `internal/services/mac_collection_service_test.go` (testify 风格测试)
- `internal/core/core.go` (Core DI + 启动初始化 + 降级模式)
- `internal/core/db/migrations/migration_mac_history.go` (PostgreSQL DDL)
- `internal/core/db/migrations/migration_148_create_ops_asset_table.go` (简单 GORM migration)
- `internal/api/v1/network/mac_history_handler.go` (handler pattern)
- `internal/api/v1/network/mac_history_router.go` (router pattern)
- `internal/api/router.go:325-329` (主路由已注册,Phase 13 不动)
- `internal/models/device_mac_history.go` (GORM model + BeforeCreate)
- `internal/models/mac_filter_rule.go` (另一个 model 参考)
- `internal/models/config.go` (Config 表参考)
- `pkg/cache/cache.go` + `pkg/cache/redis.go` (Cache 接口 + Redis 实现)
- `pkg/response/response.go` + `pkg/response/handler_helpers.go` (响应包装)
- `xingran-react-frontend/src/components/charts/EChartsWrapper.tsx` (ECharts lazy wrapper)
- `xingran-react-frontend/src/lib/api.ts` (axios + 加密 + Token 包装)
- `xingran-react-frontend/src/lib/opsApi.ts` (CRUD factory)
- `xingran-react-frontend/src/pages/network/mac/index.tsx` (页面 pattern)
- `xingran-react-frontend/src/router/componentLoader.tsx` (动态 lazy 加载)
- `xingran-react-frontend/src/router/DynamicRoutes.tsx` (动态菜单路由)
- `xingran-react-frontend/src/App.tsx` (QueryClient Provider)

**Files scanned:** 22
**Pattern extraction date:** 2026-06-13

---

## Implementation Notes

### Critical Integration Points

1. **LAG 窗口函数查询**(CONTEXT.md D-13-1.1)
   - 单 SQL:`LAG() OVER (PARTITION BY mac_address ORDER BY first_seen)` 拉取所有转换点
   - 应用层 `AggregateTrajectory` 按复合键 `(device_id, interface_name, vlan_id)` 合并连续状态
   - 复用 `macHistoryServiceImpl.MergeFlappingRecords` (lines 263-376) 的区间合并思想

2. **OUI 表初始化**(CONTEXT.md D-13-3.1)
   - 启动时 `db.Model(&MACOUIVendor{}).Count(&count)` 检查空表
   - 空表 → `os.ReadFile("configs/oui-vendors.json")` → 批量 `CreateInBatches(entries, 500)`
   - **降级**:`ReadFile` 失败时 `applogger.Warnf` + `return nil`(不阻断启动)

3. **OUI 缓存**(CONTEXT.md D-13-3.2)
   - L1 缓存键 `mac:vendor:lookup` → 实际 Redis 键 `xingran:mac:vendor:lookup`
   - Redis 不可用 → 降级 SQL `db.Where("oui_prefix = ?", oui).First(&vendor)`
   - 查询未命中 → 返回 `"Unknown Vendor"`(不识别随机化 MAC)

4. **Wiring in core.go**(CONTEXT.md § Specifics)
   - 在 line 262 之后(line 264 `Scheduler.Start` 之前)注入 `MACVendorService`
   - 启动时 `InitOUITable` 必须**不阻断**主服务启动

5. **Frontend 路由**(CONTEXT.md § Specifics + `App.tsx`)
   - 路由由**后端 `sys_menu` 动态生成**(无需修改 `App.tsx` 或 `DynamicRoutes.tsx`)
   - 新增 `pages/network/mac/trajectory/index.tsx` 文件(glob 只匹配 `index.tsx`)
   - 后端 SQL migration 新增菜单项:`path=/network/mac/trajectory`,`component=network/mac/trajectory`,`perms=network:history:trajectory`

6. **ECharts 性能**(RESEARCH.md Pitfall 3, 6)
   - `option` 必须 `useMemo` 包裹(避免 React 19 严格模式双渲染)
   - `notMerge: false, lazyUpdate: true`
   - 大量节点(>500)需启用 `dataZoom` 懒更新

### Critical Pitfalls to Avoid

1. **LAG 过滤后丢失前一状态**(RESEARCH.md Pitfall 1)
   - PostgreSQL 窗口函数在 `WHERE` 之后计算
   - **解决**:用 CTE 子查询(外层过滤,内层 LAG)或显式接受 NULL

2. **OUI 表空时主服务启动失败**(RESEARCH.md Pitfall 2)
   - `InitOUITable` 必须 `applogger.Warnf` + `return nil`,不返回 error

3. **OUI 前缀大小写不一致**(RESEARCH.md Pitfall 4)
   - OUI 表存大写 `AABBCC`,查询时 `aabbcc` 不匹配
   - **解决**:`extractOUIPrefix` 统一转 `strings.ToUpper`(Go)/ `.toUpperCase()`(TS)

4. **ECharts custom series 在 React 19 严格模式下双渲染**(RESEARCH.md Pitfall 6)
   - `option` 重建触发 `setOption` 二次调用
   - **解决**:`useMemo(() => buildOption(nodes), [nodes])` + `lazyUpdate: true`

5. **跨分区 LAG 边界**(RESEARCH.md Pitfall 5)
   - 分区裁剪不影响窗口函数,基于查询结果集
   - **验证**:EXPLAIN 确认 `first_seen` 索引被使用,无 Seq Scan

6. **Gantt 大量节点性能崩溃**(RESEARCH.md Pitfall 3)
   - 单 MAC 1 年数据可能 1000+ 节点
   - **解决**:后端按需分桶(默认按事件顺序,性能问题后续 Phase 加分桶)

### Naming Conventions

- OUI 模型主键:字符串 `OUIPrefix VARCHAR(6) PRIMARY KEY`(不是 UUID)
- Cache 键:`mac:vendor:lookup`(不带 `xingran:` 前缀,`buildKey` 自动加)
- API 端点:`POST /network/history/trajectory` / `POST /network/history/stats` / `POST /network/mac/vendor`
- 权限点:`network:history:trajectory`

### Testing Strategy

- Backend: 沿用 `testify` 表驱动测试 + 现有 `test/fixtures/` 初始化测试 DB
- Frontend: `vitest` + `@testing-library/react`(项目已锁定)
- 集成测试:用 `xingran-react-frontend/src/lib/macHistoryApi.ts` 测试调用后端 stub

### Files to Create/Modify (Final List)

**Backend (新建/修改 9 个文件):**
1. `internal/services/mac_history_query_service.go` (MODIFY) — 扩展接口 + 3 个新方法
2. `internal/services/mac_vendor_service.go` (NEW) — OUI 启动 + 查询
3. `internal/models/mac_oui_vendor.go` (NEW) — GORM 模型
4. `internal/core/db/migrations/migration_mac_oui_vendor.go` (NEW) — DDL
5. `configs/oui-vendors.json` (NEW) — 500 条 OUI 数据
6. `internal/api/v1/network/mac_history_handler.go` (MODIFY) — 新增 3 个 handler
7. `internal/api/v1/network/mac_history_router.go` (MODIFY) — 注册 3 个路由
8. `internal/core/core.go` (MODIFY) — 注入 MACVendorService
9. `internal/services/mac_history_query_service_test.go` (NEW) — QueryMACTrajectory / ConnectionStats 测试
10. `internal/services/mac_vendor_service_test.go` (NEW) — OUI 测试(Wave 0)

**Frontend (新建/修改 5 个文件):**
11. `xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx` (NEW) — ECharts Gantt
12. `xingran-react-frontend/src/pages/network/mac/trajectory/index.tsx` (NEW) — 轨迹页
13. `xingran-react-frontend/src/lib/macHistoryApi.ts` (NEW) — API 客户端
14. `xingran-react-frontend/src/utils/normalizeMACAddress.ts` (NEW) — MAC 规范化 + OUI 提取
15. (无需修改) `xingran-react-frontend/src/router/index.tsx` — 动态菜单自动生成
16. (无需修改) `xingran-react-frontend/src/store/authStore.ts` — 权限从后端菜单读

**Tests (新建 3 个):**
17. `xingran-react-frontend/src/components/network/MACTrajectoryChart.test.tsx` (NEW)
18. `xingran-react-frontend/src/pages/network/mac/trajectory/index.test.tsx` (NEW)
19. `xingran-react-frontend/src/utils/normalizeMACAddress.test.ts` (NEW)
