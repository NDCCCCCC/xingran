package addomain

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// LogService 同步日志服务
type LogService struct {
	db interface {
		WithContext(ctx context.Context) *gorm.DB
	}
}

// NewLogService 创建日志服务
func NewLogService(db interface {
	WithContext(ctx context.Context) *gorm.DB
}) *LogService {
	return &LogService{db: db}
}

// GetList 获取同步日志列表
// orderByColumn/isAsc 为服务端排序参数(可选,透传给 base.ApplySort 白名单)。
func (s *LogService) GetList(ctx context.Context, configID string, current, pageSize int, orderByColumn string, isAsc *bool) ([]models.ADSyncLog, int64, error) {
	var logs []models.ADSyncLog
	var total int64

	// 这里使用类型断言来获取 *gorm.DB
	var db *gorm.DB
	switch v := s.db.(type) {
	case *gorm.DB:
		db = v
	default:
		return nil, 0, fmt.Errorf("unsupported db type")
	}

	query := db.WithContext(ctx).Model(&models.ADSyncLog{})

	// 只在指定了configID时才添加过滤条件
	if configID != "" {
		query = query.Where("ad_config_id = ?", configID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询同步日志总数失败: %w", err)
	}

	offset := (current - 1) * pageSize
	// 用户排序(白名单)优先,无 OrderByColumn 时保留 start_time DESC 默认
	sortReq := base.BaseListRequest{
		Current:       current,
		PageSize:      pageSize,
		OrderByColumn: orderByColumn,
		IsAsc:         isAsc,
	}
	query = base.ApplySort(query, sortReq, adSyncLogAllowedSortFields)
	if orderByColumn == "" {
		query = query.Order("start_time DESC")
	}
	err := query.Offset(offset).Limit(pageSize).Find(&logs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询同步日志失败: %w", err)
	}

	return logs, total, nil
}

// adSyncLogAllowedSortFields AD同步日志可排序字段白名单(对应 sys_ad_sync_log 表列名)。
var adSyncLogAllowedSortFields = map[string]string{
	"adConfigId": "ad_config_id",
	"syncType":   "sync_type",
	"status":     "status",
	"startTime":  "start_time",
	"endTime":    "end_time",
	"createdAt":  "created_at",
}
