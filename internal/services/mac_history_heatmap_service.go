package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// HeatmapCell 热力图单元格
type HeatmapCell struct {
	DeviceID           string `json:"deviceId"`
	DeviceNameSnapshot string `json:"deviceNameSnapshot"`
	InterfaceName      string `json:"interfaceName"`
	Date               string `json:"date"`
	ChangeCount        int    `json:"changeCount"`
}

// HeatmapResult 热力图查询结果
type HeatmapResult struct {
	Cells    []HeatmapCell `json:"cells"`
	TopN     int           `json:"topN"`
	Start    string        `json:"start"`
	End      string        `json:"end"`
	Total    int           `json:"total"`
	Snapshot string        `json:"snapshot"`
}

// HeatmapQuery 热力图查询请求
type HeatmapQuery struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	TopN      int    `json:"topN"`
}

// MACHistoryHeatmapService 热力图 service
//
// Phase 15 PERF-04 (D-16/D-17 锁定):
//   - 数据源严格走 MV-04 (mv_mac_port_daily_count)
//   - 不走原表 sys_device_mac_history
//   - 走 cache-aside (复用 15-03 装饰器)
type MACHistoryHeatmapService interface {
	QueryHeatmap(ctx context.Context, req *HeatmapQuery) (*HeatmapResult, error)
}

type macHistoryHeatmapServiceImpl struct {
	db         *gorm.DB
	dataCache  *DataCacheService
	perfConfig *CacheConfigService
}

// NewMACHistoryHeatmapService 构造函数
func NewMACHistoryHeatmapService(db *gorm.DB, dataCache *DataCacheService, perfConfig *CacheConfigService) MACHistoryHeatmapService {
	return &macHistoryHeatmapServiceImpl{db: db, dataCache: dataCache, perfConfig: perfConfig}
}

// perfCacheTTL 读取 MAC 性能缓存 TTL (复用 15-03 模式)
func (s *macHistoryHeatmapServiceImpl) perfCacheTTL() time.Duration {
	const fallback = 5 * time.Minute
	if s.perfConfig == nil {
		return fallback
	}
	d := s.perfConfig.GetDuration(MACPerfConfigCacheTTLSeconds)
	if d <= 0 {
		return fallback
	}
	return d
}

// perfTopN 读取热力图 TopN 配置 (默认 100, CacheConfigService 不缓存纯 int, 走 DB 读取)
func (s *macHistoryHeatmapServiceImpl) perfTopN(ctx context.Context) int {
	const fallback = 100
	if s.db == nil {
		return fallback
	}
	var cfg models.Config
	if err := s.db.WithContext(ctx).Where("config_key = ?", MACPerfConfigHeatmapTopN).First(&cfg).Error; err != nil {
		return fallback
	}
	n, err := strconv.Atoi(cfg.ConfigValue)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func (s *macHistoryHeatmapServiceImpl) QueryHeatmap(ctx context.Context, req *HeatmapQuery) (*HeatmapResult, error) {
	// 默认 7 天
	if req.StartTime == "" || req.EndTime == "" {
		now := time.Now()
		req.EndTime = now.Format(time.RFC3339)
		req.StartTime = now.AddDate(0, 0, -7).Format(time.RFC3339)
	}
	// 默认 topN
	if req.TopN <= 0 {
		req.TopN = s.perfTopN(ctx)
	}

	// 解析时间
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("无效的开始时间格式: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("无效的结束时间格式: %w", err)
	}

	// Phase 15 PERF-03: cache-aside 装饰
	if s.dataCache != nil {
		var cached HeatmapResult
		cacheKey, keyErr := BuildMACQueryCacheKey("heatmap", req)
		if keyErr == nil {
			err := s.dataCache.GetOrSet(ctx, cacheKey, &cached, s.perfCacheTTL(), func() (interface{}, error) {
				return s.queryHeatmapFromMV(ctx, startTime, endTime, req.TopN)
			})
			if err == nil {
				return &cached, nil
			}
			applogger.Warnf("[MAC热力图] 走直查: %v", err)
		}
	}
	return s.queryHeatmapFromMV(ctx, startTime, endTime, req.TopN)
}

// queryHeatmapFromMV 从 MV-04 物化视图查询热力图数据
func (s *macHistoryHeatmapServiceImpl) queryHeatmapFromMV(ctx context.Context, start, end time.Time, topN int) (*HeatmapResult, error) {
	if !s.isPostgreSQL() {
		return &HeatmapResult{
			Cells:    []HeatmapCell{},
			TopN:     topN,
			Start:    start.Format(time.RFC3339),
			End:      end.Format(time.RFC3339),
			Snapshot: time.Now().Format(time.RFC3339),
		}, nil
	}

	// 取 TopN (按 change_count 降序)
	sql := `
SELECT device_id, device_name_snapshot, interface_name, date, change_count
FROM mv_mac_port_daily_count
WHERE date >= ? AND date <= ?
ORDER BY change_count DESC
LIMIT ?
`

	type row struct {
		DeviceID           string
		DeviceNameSnapshot string
		InterfaceName      string
		Date               time.Time
		ChangeCount        int
	}

	var rows []row
	if err := s.db.WithContext(ctx).Raw(sql, start, end, topN).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询物化视图 mv_mac_port_daily_count 失败: %w", err)
	}

	cells := make([]HeatmapCell, 0, len(rows))
	for _, r := range rows {
		cells = append(cells, HeatmapCell{
			DeviceID:           r.DeviceID,
			DeviceNameSnapshot: r.DeviceNameSnapshot,
			InterfaceName:      r.InterfaceName,
			Date:               r.Date.Format("2006-01-02"),
			ChangeCount:        r.ChangeCount,
		})
	}

	return &HeatmapResult{
		Cells:    cells,
		TopN:     topN,
		Start:    start.Format(time.RFC3339),
		End:      end.Format(time.RFC3339),
		Total:    len(cells),
		Snapshot: time.Now().Format(time.RFC3339),
	}, nil
}

func (s *macHistoryHeatmapServiceImpl) isPostgreSQL() bool {
	return s.db.Config.Dialector.Name() == "postgres"
}

// 防止未使用 models 包导致编译错误
var _ = models.Config{}
