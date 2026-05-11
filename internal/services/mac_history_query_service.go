package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
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

// MACHistoryQueryResult MAC历史查询结果
type MACHistoryQueryResult struct {
	List     []MACHistoryRecord `json:"list"`
	Total    int64              `json:"total"`
	Current  int                `json:"current"`
	PageSize int                `json:"pageSize"`
}

// MACHistoryQueryService MAC历史查询服务接口
type MACHistoryQueryService interface {
	QueryPortHistory(ctx context.Context, req *PortHistoryQuery) (*MACHistoryQueryResult, error)
	QueryDeviceHistory(ctx context.Context, req *DeviceHistoryQuery) (*MACHistoryQueryResult, error)
}

// macHistoryQueryServiceImpl MAC历史查询服务实现
type macHistoryQueryServiceImpl struct {
	db *gorm.DB
}

// NewMACHistoryQueryService 创建MAC历史查询服务
func NewMACHistoryQueryService(db *gorm.DB) MACHistoryQueryService {
	return &macHistoryQueryServiceImpl{db: db}
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
