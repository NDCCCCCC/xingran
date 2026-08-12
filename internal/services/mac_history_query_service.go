package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// PortHistoryQuery 端口历史查询请求
type PortHistoryQuery struct {
	DeviceID      string `json:"deviceId" binding:"required"`
	InterfaceName string `json:"interfaceName,omitempty"`
	StartTime     string `json:"startTime,omitempty"` // RFC3339 格式
	EndTime       string `json:"endTime,omitempty"`   // RFC3339 格式
	Current       int    `json:"current" binding:"min=1"`
	PageSize      int    `json:"pageSize" binding:"min=1,max=100"`
}

// DeviceHistoryQuery 设备历史查询请求
type DeviceHistoryQuery struct {
	DeviceID  string `json:"deviceId" binding:"required"`
	StartTime string `json:"startTime,omitempty"` // RFC3339 格式
	EndTime   string `json:"endTime,omitempty"`   // RFC3339 格式
	Current   int    `json:"current" binding:"min=1"`
	PageSize  int    `json:"pageSize" binding:"min=1,max=100"`
}

// MACHistoryRecord MAC历史记录响应
type MACHistoryRecord struct {
	ID                 string      `json:"id"`
	DeviceID           string      `json:"deviceId"`
	DeviceNameSnapshot string      `json:"deviceNameSnapshot"`
	MACAddress         string      `json:"macAddress"`
	InterfaceName      string      `json:"interfaceName"`
	VLANID             *int        `json:"vlanId,omitempty"`
	EventType          string      `json:"eventType"`
	FirstSeen          time.Time   `json:"firstSeen"`
	LastSeen           time.Time   `json:"lastSeen"`
	CollectedAt        time.Time   `json:"collectedAt"`
}

// MACHistoryListQuery 通用MAC历史列表查询请求（支持过滤+分页）
type MACHistoryListQuery struct {
	Current       int    `json:"current" form:"current" binding:"min=1"`
	PageSize      int    `json:"pageSize" form:"pageSize" binding:"min=1,max=100"`
	MAC           string `json:"mac,omitempty" form:"mac"`
	DeviceID      string `json:"deviceId,omitempty" form:"deviceId"`
	InterfaceName string `json:"interfaceName,omitempty" form:"interfaceName"`
	VLANID        *int   `json:"vlanId,omitempty" form:"vlanId"`
	EventType     string `json:"eventType,omitempty" form:"eventType"`
	Status        *int   `json:"status,omitempty" form:"status"`
	StartTime     string `json:"startTime,omitempty" form:"startTime"`
	EndTime       string `json:"endTime,omitempty" form:"endTime"`
	ExportScope   string `json:"exportScope,omitempty" form:"exportScope"` // "current" or "all"，仅 ExportHistory 使用
}

// MACHistoryQueryResult MAC历史查询结果
type MACHistoryQueryResult struct {
	List     []MACHistoryRecord `json:"list"`
	Total    int64              `json:"total"`
	Current  int                `json:"current"`
	PageSize int                `json:"pageSize"`
}

// ConnectionStatsQuery 连接时长统计查询请求
type ConnectionStatsQuery struct {
	MACAddress string `json:"macAddress,omitempty"` // 可选，留空统计所有MAC
	StartTime  string `json:"startTime" binding:"required"` // RFC3339，必填
	EndTime    string `json:"endTime" binding:"required"`   // RFC3339，必填
	TopN       int    `json:"topN,omitempty"`        // 默认10
}

// ConnectionStatsDetail 连接时长统计明细
type ConnectionStatsDetail struct {
	MACAddress      string    `json:"macAddress"`
	DeviceID        string    `json:"deviceId"`
	DeviceName      string    `json:"deviceName"`
	Interface       string    `json:"interface"`
	FirstSeen       time.Time `json:"firstSeen"`
	LastSeen        time.Time `json:"lastSeen"`
	Duration        int64     `json:"duration"`       // 秒
	EventCount      int       `json:"eventCount"`
	FlappingCount   int       `json:"flappingCount"`  // event_type='moved'计数
	IsLongOccupancy bool      `json:"isLongOccupancy"` // duration > threshold
}

// LongOccupancyByMAC 长期占用TOP（按MAC聚合）
type LongOccupancyByMAC struct {
	MACAddress    string `json:"macAddress"`
	Vendor        string `json:"vendor"`        // Phase 13留空字符串，由前端OUI调用补齐
	TotalDuration int64  `json:"totalDuration"` // 秒
	PortCount     int    `json:"portCount"`
}

// HotspotByPort 端口热点TOP（按端口聚合）
type HotspotByPort struct {
	DeviceID       string `json:"deviceId"`
	DeviceName     string `json:"deviceName"`
	Interface      string `json:"interface"`
	UniqueMACCount int    `json:"uniqueMacCount"`
	TotalDuration  int64  `json:"totalDuration"` // 秒
}

// ConnectionStatsResponse 连接时长统计响应
type ConnectionStatsResponse struct {
	Details           []ConnectionStatsDetail `json:"details"`
	TopByMAC          []LongOccupancyByMAC    `json:"topByMac"`
	TopByPort         []HotspotByPort         `json:"topByPort"`
	LongOccupancyDays int                     `json:"longOccupancyDays"` // 当前阈值
}

// MACHistoryQueryService MAC历史查询服务接口
type MACHistoryQueryService interface {
	QueryPortHistory(ctx context.Context, req *PortHistoryQuery) (*MACHistoryQueryResult, error)
	QueryDeviceHistory(ctx context.Context, req *DeviceHistoryQuery) (*MACHistoryQueryResult, error)
	QueryHistory(ctx context.Context, req *MACHistoryListQuery) (*MACHistoryQueryResult, error)
	ExportHistory(ctx context.Context, req *MACHistoryListQuery, w io.Writer) error
	QueryConnectionStats(ctx context.Context, req *ConnectionStatsQuery) (*ConnectionStatsResponse, error)
	ImportOUIData(ctx context.Context) error
	GetVendor(ctx context.Context, macAddress string) (string, error)
}

// macHistoryQueryServiceImpl MAC历史查询服务实现
type macHistoryQueryServiceImpl struct {
	db         *gorm.DB
	cache      cache.Cache
	dataCache  *DataCacheService
	perfConfig *CacheConfigService
}

// NewMACHistoryQueryService 创建MAC历史查询服务
func NewMACHistoryQueryService(db *gorm.DB) MACHistoryQueryService {
	return &macHistoryQueryServiceImpl{db: db, cache: nil}
}

// NewMACHistoryQueryServiceWithCache 创建带缓存的查询服务 (Phase 15 PERF-03)
func NewMACHistoryQueryServiceWithCache(db *gorm.DB, dataCache *DataCacheService, perfConfig *CacheConfigService) MACHistoryQueryService {
	return &macHistoryQueryServiceImpl{db: db, cache: nil, dataCache: dataCache, perfConfig: perfConfig}
}

// perfCacheTTL 读取 MAC 性能缓存 TTL (5 分钟兜底)
func (s *macHistoryQueryServiceImpl) perfCacheTTL() time.Duration {
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

// extractOUIPrefix 提取MAC地址的OUI前缀（前3字节）
// 移除分隔符 (.:-) + ToUpper + 取前6位
func extractOUIPrefix(mac string) string {
	// 移除所有分隔符
	normalized := strings.ReplaceAll(mac, ".", "")
	normalized = strings.ReplaceAll(normalized, ":", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ToUpper(normalized)

	// 取前6位（OUI前缀）
	if len(normalized) >= 6 {
		return normalized[:6]
	}
	return normalized
}

// validateMACAddress 验证MAC地址格式
// 正则校验：12位十六进制字符
func validateMACAddress(mac string) error {
	// 先尝试标准化处理
	normalized := strings.ReplaceAll(mac, ".", "")
	normalized = strings.ReplaceAll(normalized, ":", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ToUpper(normalized)

	// 验证格式：应该是12个十六进制字符
	macPattern := regexp.MustCompile(`^[0-9A-F]{12}$`)
	if !macPattern.MatchString(normalized) {
		return fmt.Errorf("无效的MAC地址格式")
	}

	return nil
}

// ImportOUIData 启动时从configs/oui-vendors.json导入OUI数据
func (s *macHistoryQueryServiceImpl) ImportOUIData(ctx context.Context) error {
	// 检查表是否已有数据
	var count int64
	if err := s.db.WithContext(ctx).Table("sys_mac_oui_vendor").Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check OUI table: %w", err)
	}
	if count > 0 {
		applogger.Infof("OUI table already has %d records, skipping import", count)
		return nil
	}

	// 读取JSON文件
	data, err := os.ReadFile("configs/oui-vendors.json")
	if err != nil {
		return fmt.Errorf("failed to read OUI JSON: %w", err)
	}

	var vendors []models.MACOUIVendor
	if err := json.Unmarshal(data, &vendors); err != nil {
		return fmt.Errorf("failed to parse OUI JSON: %w", err)
	}

	// 批量插入（每批100条）
	batchSize := 100
	for i := 0; i < len(vendors); i += batchSize {
		end := i + batchSize
		if end > len(vendors) {
			end = len(vendors)
		}
		batch := vendors[i:end]
		if err := s.db.WithContext(ctx).Create(&batch).Error; err != nil {
			return fmt.Errorf("failed to insert OUI batch %d: %w", i/batchSize, err)
		}
	}

	applogger.Infof("Imported %d OUI vendors", len(vendors))
	return nil
}

// GetVendor 查询MAC地址的厂商信息（基于OUI前6位）
func (s *macHistoryQueryServiceImpl) GetVendor(ctx context.Context, macAddress string) (string, error) {
	// 规范化MAC地址并提取OUI前缀
	normalized := strings.ReplaceAll(macAddress, ".", "")
	normalized = strings.ReplaceAll(normalized, ":", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ToUpper(normalized)

	if len(normalized) < 6 {
		return "Unknown Vendor", nil
	}
	oui := normalized[:6] // AABBCC格式

	// Redis缓存键
	cacheKey := fmt.Sprintf("mac:vendor:%s", oui)

	// 尝试从缓存获取（如果cache可用）
	if s.cache != nil {
		vendorName, err := s.cache.Get(ctx, cacheKey)
		if err == nil && vendorName != "" {
			return vendorName, nil
		}
		// 缓存未命中或出错，降级到DB查询
	}

	// DB查询
	var vendor models.MACOUIVendor
	if err := s.db.WithContext(ctx).Where("oui_prefix = ?", oui).First(&vendor).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 未知OUI，缓存"Unknown Vendor"避免重复查询
			if s.cache != nil {
				_ = s.cache.Set(ctx, cacheKey, "Unknown Vendor", 24*time.Hour)
			}
			return "Unknown Vendor", nil
		}
		return "", fmt.Errorf("failed to query OUI vendor: %w", err)
	}

	// 缓存结果（24小时）
	if s.cache != nil {
		_ = s.cache.Set(ctx, cacheKey, vendor.VendorName, 24*time.Hour)
	}

	return vendor.VendorName, nil
}

// QueryPortHistory 查询指定端口的历史记录
func (s *macHistoryQueryServiceImpl) QueryPortHistory(ctx context.Context, req *PortHistoryQuery) (*MACHistoryQueryResult, error) {
	// WR-01 fix: 验证设备ID格式
	if _, err := uuid.Parse(req.DeviceID); err != nil {
		return nil, fmt.Errorf("无效的设备ID格式: %w", err)
	}

	// 设置默认值
	if req.Current < 1 {
		req.Current = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	// Phase 15 PERF-03: cache-aside 装饰 (D-12 锁定)
	if s.dataCache != nil {
		var cached MACHistoryQueryResult
		cacheKey, keyErr := BuildMACQueryCacheKey("port-history", req)
		if keyErr == nil {
			err := s.dataCache.GetOrSet(ctx, cacheKey, &cached, s.perfCacheTTL(), func() (interface{}, error) {
				return s.queryPortHistoryFromDB(ctx, req)
			})
			if err == nil {
				return &cached, nil
			}
			applogger.Warnf("[MAC缓存] port-history 走直查: %v", err)
		}
	}
	return s.queryPortHistoryFromDB(ctx, req)
}

// queryPortHistoryFromDB 实际执行 SQL 查询 (无缓存)
func (s *macHistoryQueryServiceImpl) queryPortHistoryFromDB(ctx context.Context, req *PortHistoryQuery) (*MACHistoryQueryResult, error) {
	// 构建查询
	query := s.db.WithContext(ctx).Table("sys_device_mac_history")

	// 添加过滤条件
	query = query.Where("device_id = ?", req.DeviceID)

	// 可选：接口名过滤
	if req.InterfaceName != "" {
		query = query.Where("interface_name = ?", req.InterfaceName)
	}

	// 可选：时间范围过滤
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

		// 验证时间跨度
		if endTime.Sub(startTime) > maxQueryRange {
			return nil, fmt.Errorf("查询时间跨度过大，最大允许 %d 天", int(maxQueryRange.Hours()/24))
		}

		query = query.Where("first_seen >= ?", startTime).
			Where("first_seen <= ?", endTime)
	} else if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return nil, fmt.Errorf("无效的开始时间格式: %w", err)
		}
		query = query.Where("first_seen >= ?", startTime)
	} else if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return nil, fmt.Errorf("无效的结束时间格式: %w", err)
		}
		query = query.Where("first_seen <= ?", endTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		applogger.Errorf("[MAC历史查询] 查询总数失败: %v", err)
		return nil, fmt.Errorf("查询历史记录总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Current - 1) * req.PageSize
	var records []models.DeviceMACHistory
	if err := query.
		Order("first_seen DESC").
		Limit(req.PageSize).
		Offset(offset).
		Find(&records).Error; err != nil {
		applogger.Errorf("[MAC历史查询] 查询记录失败: %v", err)
		return nil, fmt.Errorf("查询历史记录失败: %w", err)
	}

	// 映射结果
	result := &MACHistoryQueryResult{
		List:     make([]MACHistoryRecord, 0, len(records)),
		Total:    total,
		Current:  req.Current,
		PageSize: req.PageSize,
	}

	for _, record := range records {
		result.List = append(result.List, MACHistoryRecord{
			ID:                 record.ID,
			DeviceID:           record.DeviceID,
			DeviceNameSnapshot: record.DeviceNameSnapshot,
			MACAddress:         record.MACAddress,
			InterfaceName:      record.InterfaceName,
			VLANID:             record.VLANID,
			EventType:          string(record.EventType),
			FirstSeen:          record.FirstSeen,
			LastSeen:           record.LastSeen,
			CollectedAt:        record.CollectedAt,
		})
	}

	return result, nil
}

// QueryDeviceHistory 查询指定设备的历史记录
func (s *macHistoryQueryServiceImpl) QueryDeviceHistory(ctx context.Context, req *DeviceHistoryQuery) (*MACHistoryQueryResult, error) {
	// WR-01 fix: 验证设备ID格式
	if _, err := uuid.Parse(req.DeviceID); err != nil {
		return nil, fmt.Errorf("无效的设备ID格式: %w", err)
	}

	// 设置默认值
	if req.Current < 1 {
		req.Current = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	// Phase 15 PERF-03: cache-aside 装饰 (D-12 锁定)
	if s.dataCache != nil {
		var cached MACHistoryQueryResult
		cacheKey, keyErr := BuildMACQueryCacheKey("device-history", req)
		if keyErr == nil {
			err := s.dataCache.GetOrSet(ctx, cacheKey, &cached, s.perfCacheTTL(), func() (interface{}, error) {
				return s.queryDeviceHistoryFromDB(ctx, req)
			})
			if err == nil {
				return &cached, nil
			}
			applogger.Warnf("[MAC缓存] device-history 走直查: %v", err)
		}
	}
	return s.queryDeviceHistoryFromDB(ctx, req)
}

// queryDeviceHistoryFromDB 实际执行 SQL 查询 (无缓存)
func (s *macHistoryQueryServiceImpl) queryDeviceHistoryFromDB(ctx context.Context, req *DeviceHistoryQuery) (*MACHistoryQueryResult, error) {
	// 构建查询
	query := s.db.WithContext(ctx).Table("sys_device_mac_history")

	// 添加过滤条件
	query = query.Where("device_id = ?", req.DeviceID)

	// 可选：时间范围过滤
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

		// 验证时间跨度
		if endTime.Sub(startTime) > maxQueryRange {
			return nil, fmt.Errorf("查询时间跨度过大，最大允许 %d 天", int(maxQueryRange.Hours()/24))
		}

		query = query.Where("first_seen >= ?", startTime).
			Where("first_seen <= ?", endTime)
	} else if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return nil, fmt.Errorf("无效的开始时间格式: %w", err)
		}
		query = query.Where("first_seen >= ?", startTime)
	} else if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return nil, fmt.Errorf("无效的结束时间格式: %w", err)
		}
		query = query.Where("first_seen <= ?", endTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		applogger.Errorf("[MAC历史查询] 查询总数失败: %v", err)
		return nil, fmt.Errorf("查询历史记录总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Current - 1) * req.PageSize
	var records []models.DeviceMACHistory
	if err := query.
		Order("first_seen DESC").
		Limit(req.PageSize).
		Offset(offset).
		Find(&records).Error; err != nil {
		applogger.Errorf("[MAC历史查询] 查询记录失败: %v", err)
		return nil, fmt.Errorf("查询历史记录失败: %w", err)
	}

	// 映射结果
	result := &MACHistoryQueryResult{
		List:     make([]MACHistoryRecord, 0, len(records)),
		Total:    total,
		Current:  req.Current,
		PageSize: req.PageSize,
	}

	for _, record := range records {
		result.List = append(result.List, MACHistoryRecord{
			ID:                 record.ID,
			DeviceID:           record.DeviceID,
			DeviceNameSnapshot: record.DeviceNameSnapshot,
			MACAddress:         record.MACAddress,
			InterfaceName:      record.InterfaceName,
			VLANID:             record.VLANID,
			EventType:          string(record.EventType),
			FirstSeen:          record.FirstSeen,
			LastSeen:           record.LastSeen,
			CollectedAt:        record.CollectedAt,
		})
	}

	return result, nil
}

// QueryHistory 通用MAC历史查询（支持任意字段过滤 + 分页）
// 用于 Phase 14 前端列表页 POST /network/history/list
func (s *macHistoryQueryServiceImpl) QueryHistory(ctx context.Context, req *MACHistoryListQuery) (*MACHistoryQueryResult, error) {
	// MAC 地址校验（可选）
	if req.MAC != "" {
		if err := validateMACAddress(req.MAC); err != nil {
			return nil, fmt.Errorf("MAC地址验证失败: %w", err)
		}
	}

	// DeviceID UUID 校验（可选）
	if req.DeviceID != "" {
		if _, err := uuid.Parse(req.DeviceID); err != nil {
			return nil, fmt.Errorf("无效的设备ID格式: %w", err)
		}
	}

	// 默认值
	if req.Current < 1 {
		req.Current = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	// 构建 GORM 查询链
	query := s.db.WithContext(ctx).Table("sys_device_mac_history")

	if req.DeviceID != "" {
		query = query.Where("device_id = ?", req.DeviceID)
	}
	if req.InterfaceName != "" {
		query = query.Where("interface_name = ?", req.InterfaceName)
	}
	if req.MAC != "" {
		query = query.Where("mac_address = ?", NormalizeMACAddress(req.MAC))
	}
	if req.VLANID != nil {
		query = query.Where("vlan_id = ?", *req.VLANID)
	}
	if req.EventType != "" {
		query = query.Where("event_type = ?", req.EventType)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	// 时间范围过滤（可选）
	const maxQueryRange = 365 * 24 * time.Hour

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
		// 把 RFC3339 UTC 瞬时值转本地 loc(Asia/Shanghai,见 cmd/main.go setTimeZone),
		// 让 pgx 发北京墙钟匹配 timestamp without time zone 列存储,避免 8h 错位
		// (根因同 device_mac_history.go 时区注释)。
		startTime = startTime.Local()
		endTime = endTime.Local()
		query = query.Where("first_seen >= ?", startTime).
			Where("first_seen <= ?", endTime)
	} else if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return nil, fmt.Errorf("无效的开始时间格式: %w", err)
		}
		startTime = startTime.Local()
		query = query.Where("first_seen >= ?", startTime)
	} else if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return nil, fmt.Errorf("无效的结束时间格式: %w", err)
		}
		endTime = endTime.Local()
		query = query.Where("first_seen <= ?", endTime)
	}

	// 总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		applogger.Errorf("[MAC历史查询] 查询总数失败: %v", err)
		return nil, fmt.Errorf("查询历史记录总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Current - 1) * req.PageSize
	var records []models.DeviceMACHistory
	if err := query.
		Order("first_seen DESC").
		Limit(req.PageSize).
		Offset(offset).
		Find(&records).Error; err != nil {
		applogger.Errorf("[MAC历史查询] 查询记录失败: %v", err)
		return nil, fmt.Errorf("查询历史记录失败: %w", err)
	}

	result := &MACHistoryQueryResult{
		List:     make([]MACHistoryRecord, 0, len(records)),
		Total:    total,
		Current:  req.Current,
		PageSize: req.PageSize,
	}

	for _, record := range records {
		result.List = append(result.List, MACHistoryRecord{
			ID:                 record.ID,
			DeviceID:           record.DeviceID,
			DeviceNameSnapshot: record.DeviceNameSnapshot,
			MACAddress:         record.MACAddress,
			InterfaceName:      record.InterfaceName,
			VLANID:             record.VLANID,
			EventType:          string(record.EventType),
			FirstSeen:          record.FirstSeen,
			LastSeen:           record.LastSeen,
			CollectedAt:        record.CollectedAt,
		})
	}

	return result, nil
}

// ExportHistory 导出MAC历史为 xlsx，写入 io.Writer
// 用于 Phase 14 前端 GET /network/history/list?format=xlsx
// 强制 30 天时间上限（UI-02 锁定）；最多 100000 行（DoS 保护）
func (s *macHistoryQueryServiceImpl) ExportHistory(ctx context.Context, req *MACHistoryListQuery, w io.Writer) error {
	// MAC 地址校验（可选）
	if req.MAC != "" {
		if err := validateMACAddress(req.MAC); err != nil {
			return fmt.Errorf("MAC地址验证失败: %w", err)
		}
	}

	// DeviceID UUID 校验（可选）
	if req.DeviceID != "" {
		if _, err := uuid.Parse(req.DeviceID); err != nil {
			return fmt.Errorf("无效的设备ID格式: %w", err)
		}
	}

	// 强制 30 天上限（导出）
	const maxExportRange = 30 * 24 * time.Hour

	var startTime, endTime time.Time
	if req.StartTime != "" && req.EndTime != "" {
		var err error
		startTime, err = time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return fmt.Errorf("无效的开始时间格式: %w", err)
		}
		endTime, err = time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return fmt.Errorf("无效的结束时间格式: %w", err)
		}
		if endTime.Sub(startTime) > maxExportRange {
			return fmt.Errorf("导出范围最大 30 天,请缩小查询条件")
		}
	} else {
		// 无时间范围时使用最近 30 天
		endTime = time.Now()
		startTime = endTime.Add(-30 * 24 * time.Hour)
	}

	// 构建 GORM 查询链（不带分页，但加 LIMIT 100000）
	query := s.db.WithContext(ctx).Table("sys_device_mac_history").
		Where("first_seen >= ?", startTime).
		Where("first_seen <= ?", endTime)

	if req.DeviceID != "" {
		query = query.Where("device_id = ?", req.DeviceID)
	}
	if req.InterfaceName != "" {
		query = query.Where("interface_name = ?", req.InterfaceName)
	}
	if req.MAC != "" {
		query = query.Where("mac_address = ?", NormalizeMACAddress(req.MAC))
	}
	if req.VLANID != nil {
		query = query.Where("vlan_id = ?", *req.VLANID)
	}
	if req.EventType != "" {
		query = query.Where("event_type = ?", req.EventType)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	var records []models.DeviceMACHistory
	if err := query.
		Order("first_seen DESC").
		Limit(100000).
		Find(&records).Error; err != nil {
		applogger.Errorf("[MAC历史导出] 查询失败: %v", err)
		return fmt.Errorf("查询导出记录失败: %w", err)
	}

	// 构建 xlsx 文件
	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	sheetName := "MAC 历史"
	if _, err := f.NewSheet(sheetName); err != nil {
		return fmt.Errorf("创建工作表失败: %w", err)
	}
	// 删除默认 Sheet1
	if err := f.DeleteSheet("Sheet1"); err != nil {
		applogger.Warnf("[MAC历史导出] 删除默认 Sheet1 失败: %v", err)
	}

	// 表头（9 列）
	headers := []string{"时间", "MAC", "设备", "端口", "VLAN", "事件类型", "首次出现", "最后出现", "采集时间"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheetName, cell, h); err != nil {
			return fmt.Errorf("写表头失败: %w", err)
		}
	}

	// 数据行
	const timeFormat = "2006-01-02 15:04:05"
	for rowIdx, record := range records {
		row := rowIdx + 2 // 第 1 行是表头
		rowData := []interface{}{
			record.FirstSeen.Format(timeFormat),
			record.MACAddress,
			record.DeviceNameSnapshot,
			record.InterfaceName,
			"", // VLAN 默认空
			record.EventType,
			record.FirstSeen.Format(timeFormat),
			record.LastSeen.Format(timeFormat),
			record.CollectedAt.Format(timeFormat),
		}
		if record.VLANID != nil {
			rowData[4] = *record.VLANID
		}
		for colIdx, val := range rowData {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			if err := f.SetCellValue(sheetName, cell, val); err != nil {
				return fmt.Errorf("写数据行失败: %w", err)
			}
		}
	}

	// 写入 io.Writer
	if err := f.Write(w); err != nil {
		applogger.Errorf("[MAC历史导出] 写入 xlsx 失败: %v", err)
		return fmt.Errorf("写入 xlsx 失败: %w", err)
	}

	return nil
}

// getLongOccupancyThreshold 获取长期占用阈值（天数）
// 从sys_config表查询，不存在或解析失败时返回默认值30
func (s *macHistoryQueryServiceImpl) getLongOccupancyThreshold(ctx context.Context) (int, error) {
	const defaultDays = 30
	const configKey = "network.mac.history.long_occupancy_threshold_days"

	var configValue string
	err := s.db.WithContext(ctx).
		Table("sys_config").
		Where("config_key = ?", configKey).
		Select("config_value").
		Row().
		Scan(&configValue)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 配置不存在，返回默认值
			return defaultDays, nil
		}
		return defaultDays, fmt.Errorf("查询长期占用阈值配置失败: %w", err)
	}

	// 解析配置值
	var days int
	if _, err := fmt.Sscanf(configValue, "%d", &days); err != nil || days <= 0 {
		// 解析失败或值无效，返回默认值
		applogger.Warnf("[MAC历史统计] 长期占用阈值配置值无效: %s，使用默认值 %d 天", configValue, defaultDays)
		return defaultDays, nil
	}

	return days, nil
}

// QueryConnectionStats 查询连接时长统计
// 输出明细（每个MAC×端口的停留时长+flapping计数）+Top-N（按MAC长期占用Top+按端口热门连接Top）
func (s *macHistoryQueryServiceImpl) QueryConnectionStats(ctx context.Context, req *ConnectionStatsQuery) (*ConnectionStatsResponse, error) {
	// Phase 15 PERF-03: cache-aside 装饰 (D-12 锁定)
	if s.dataCache != nil {
		var cached ConnectionStatsResponse
		cacheKey, keyErr := BuildMACQueryCacheKey("stats", req)
		if keyErr == nil {
			err := s.dataCache.GetOrSet(ctx, cacheKey, &cached, s.perfCacheTTL(), func() (interface{}, error) {
				return s.queryConnectionStatsFromDB(ctx, req)
			})
			if err == nil {
				return &cached, nil
			}
			applogger.Warnf("[MAC缓存] stats 走直查: %v", err)
		}
	}
	return s.queryConnectionStatsFromDB(ctx, req)
}

// queryConnectionStatsFromDB 实际执行 SQL 查询 (无缓存)
func (s *macHistoryQueryServiceImpl) queryConnectionStatsFromDB(ctx context.Context, req *ConnectionStatsQuery) (*ConnectionStatsResponse, error) {
	// 1. 校验StartTime/EndTime必填
	if req.StartTime == "" || req.EndTime == "" {
		return nil, fmt.Errorf("开始时间和结束时间为必填项")
	}

	// 2. 解析时间范围
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("无效的开始时间格式: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("无效的结束时间格式: %w", err)
	}

	// 3. 验证时间跨度（复用maxQueryRange）
	const maxQueryRange = 365 * 24 * time.Hour
	if endTime.Sub(startTime) > maxQueryRange {
		return nil, fmt.Errorf("查询时间跨度过大，最大允许 %d 天", int(maxQueryRange.Hours()/24))
	}

	// 4. 调用getLongOccupancyThreshold拿到阈值
	thresholdDays, err := s.getLongOccupancyThreshold(ctx)
	if err != nil {
		applogger.Warnf("[MAC历史统计] 获取长期占用阈值失败，使用默认值30天: %v", err)
		thresholdDays = 30 // 降级使用默认值
	}
	thresholdSec := int64(thresholdDays * 86400)

	// 5. 设置TopN默认值
	topN := req.TopN
	if topN <= 0 {
		topN = 10
	}

	// 6. 查询明细SQL（按MAC×Device×Interface聚合）
	detailSQL := `
		SELECT
			mac_address, device_id, device_name_snapshot, interface_name,
			MIN(first_seen) AS first_seen,
			MAX(last_seen) AS last_seen,
			EXTRACT(EPOCH FROM (MAX(last_seen) - MIN(first_seen)))::bigint AS duration,
			COUNT(*) AS event_count,
			COUNT(*) FILTER (WHERE event_type = 'moved') AS flapping_count
		FROM sys_device_mac_history
		WHERE first_seen >= ? AND first_seen <= ?
	`
	// 可选：MAC地址过滤
	args := []interface{}{startTime, endTime}
	if req.MACAddress != "" {
		// 验证MAC格式
		if err := validateMACAddress(req.MACAddress); err != nil {
			return nil, fmt.Errorf("MAC地址验证失败: %w", err)
		}
		// 规范化MAC地址
		normalizedMAC := NormalizeMACAddress(req.MACAddress)
		detailSQL += " AND mac_address = ?"
		args = append(args, normalizedMAC)
	}

	detailSQL += `
		GROUP BY mac_address, device_id, device_name_snapshot, interface_name
		ORDER BY duration DESC
		LIMIT ? OFFSET ?
	`

	type rawDetail struct {
		MACAddress         string
		DeviceID           string
		DeviceNameSnapshot string
		InterfaceName      string
		FirstSeen          time.Time
		LastSeen           time.Time
		Duration           int64
		EventCount         int
		FlappingCount      int
	}

	var rawDetails []rawDetail
	if err := s.db.WithContext(ctx).Raw(detailSQL, append(args, 1000, 0)...).Scan(&rawDetails).Error; err != nil {
		applogger.Errorf("[MAC历史统计] 查询明细失败: %v", err)
		return nil, fmt.Errorf("查询连接时长明细失败: %w", err)
	}

	// 转换为响应格式，标记IsLongOccupancy
	details := make([]ConnectionStatsDetail, 0, len(rawDetails))
	for _, d := range rawDetails {
		details = append(details, ConnectionStatsDetail{
			MACAddress:      d.MACAddress,
			DeviceID:        d.DeviceID,
			DeviceName:      d.DeviceNameSnapshot,
			Interface:       d.InterfaceName,
			FirstSeen:       d.FirstSeen,
			LastSeen:        d.LastSeen,
			Duration:        d.Duration,
			EventCount:      d.EventCount,
			FlappingCount:   d.FlappingCount,
			IsLongOccupancy: d.Duration > thresholdSec,
		})
	}

	// 7. 查询TopByMAC SQL（按MAC聚合，过滤长期占用）
	topByMACSQL := `
		SELECT
			mac_address,
			SUM(EXTRACT(EPOCH FROM (MAX(last_seen) - MIN(first_seen))))::bigint AS total_duration,
			COUNT(DISTINCT device_id || ':' || interface_name) AS port_count
		FROM sys_device_mac_history
		WHERE first_seen >= ? AND first_seen <= ?
	`

	topByMACArgs := []interface{}{startTime, endTime}
	if req.MACAddress != "" {
		normalizedMAC := NormalizeMACAddress(req.MACAddress)
		topByMACSQL += " AND mac_address = ?"
		topByMACArgs = append(topByMACArgs, normalizedMAC)
	}

	topByMACSQL += `
		GROUP BY mac_address
		HAVING SUM(EXTRACT(EPOCH FROM (MAX(last_seen) - MIN(first_seen)))) > ?
		ORDER BY total_duration DESC
		LIMIT ?
	`

	type rawTopMAC struct {
		MACAddress    string
		TotalDuration int64
		PortCount     int
	}

	var rawTopMACs []rawTopMAC
	if err := s.db.WithContext(ctx).Raw(topByMACSQL, append(topByMACArgs, thresholdSec, topN)...).Scan(&rawTopMACs).Error; err != nil {
		applogger.Errorf("[MAC历史统计] 查询TopByMAC失败: %v", err)
		return nil, fmt.Errorf("查询长期占用TOP失败: %w", err)
	}

	// 转换为响应格式（Vendor字段留空，由前端OUI调用补齐）
	topByMAC := make([]LongOccupancyByMAC, 0, len(rawTopMACs))
	for _, m := range rawTopMACs {
		topByMAC = append(topByMAC, LongOccupancyByMAC{
			MACAddress:    m.MACAddress,
			Vendor:        "", // Phase 13留空，由前端补齐
			TotalDuration: m.TotalDuration,
			PortCount:     m.PortCount,
		})
	}

	// 8. 查询TopByPort SQL（按端口聚合）
	topByPortSQL := `
		SELECT
			device_id, device_name_snapshot, interface_name,
			COUNT(DISTINCT mac_address) AS unique_mac_count,
			SUM(EXTRACT(EPOCH FROM (MAX(last_seen) - MIN(first_seen))))::bigint AS total_duration
		FROM sys_device_mac_history
		WHERE first_seen >= ? AND first_seen <= ?
	`

	topByPortArgs := []interface{}{startTime, endTime}
	// 注意：TopByPort不过滤MAC，因为用户可能想看特定MAC的端口热点
	// 但如果请求指定了MAC，则只统计该MAC的端口连接
	if req.MACAddress != "" {
		normalizedMAC := NormalizeMACAddress(req.MACAddress)
		topByPortSQL += " AND mac_address = ?"
		topByPortArgs = append(topByPortArgs, normalizedMAC)
	}

	topByPortSQL += `
		GROUP BY device_id, device_name_snapshot, interface_name
		ORDER BY total_duration DESC
		LIMIT ?
	`

	type rawTopPort struct {
		DeviceID           string
		DeviceNameSnapshot string
		InterfaceName      string
		UniqueMACCount     int
		TotalDuration      int64
	}

	var rawTopPorts []rawTopPort
	if err := s.db.WithContext(ctx).Raw(topByPortSQL, append(topByPortArgs, topN)...).Scan(&rawTopPorts).Error; err != nil {
		applogger.Errorf("[MAC历史统计] 查询TopByPort失败: %v", err)
		return nil, fmt.Errorf("查询端口热点TOP失败: %w", err)
	}

	// 转换为响应格式
	topByPort := make([]HotspotByPort, 0, len(rawTopPorts))
	for _, p := range rawTopPorts {
		topByPort = append(topByPort, HotspotByPort{
			DeviceID:       p.DeviceID,
			DeviceName:     p.DeviceNameSnapshot,
			Interface:      p.InterfaceName,
			UniqueMACCount: p.UniqueMACCount,
			TotalDuration:  p.TotalDuration,
		})
	}

	// 9. 返回三段式响应
	return &ConnectionStatsResponse{
		Details:           details,
		TopByMAC:          topByMAC,
		TopByPort:         topByPort,
		LongOccupancyDays: thresholdDays,
	}, nil
}
