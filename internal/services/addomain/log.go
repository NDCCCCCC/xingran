package addomain

import (
	"context"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
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
func (s *LogService) GetList(ctx context.Context, configID string, current, pageSize int) ([]models.ADSyncLog, int64, error) {
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
	err := query.Offset(offset).Limit(pageSize).Order("start_time DESC").Find(&logs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询同步日志失败: %w", err)
	}

	return logs, total, nil
}
