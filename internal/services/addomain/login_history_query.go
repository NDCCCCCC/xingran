package addomain

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
	"gorm.io/gorm"
)

// ADLoginHistoryQuery AD 设备登录历史查询请求(quick 261016-ad-login-history)
type ADLoginHistoryQuery struct {
	ComputerID string `json:"computerId" binding:"required"`
	Current    int    `json:"current"`
	PageSize   int    `json:"pageSize"`
	StartTime  string `json:"startTime,omitempty"` // 可选,RFC3339
	EndTime    string `json:"endTime,omitempty"`   // 可选,RFC3339
}

// ADLoginHistoryService AD 设备登录历史查询服务接口
type ADLoginHistoryService interface {
	// ListComputerLoginHistory 按设备查询登录历史(固定 login_time DESC,先 Count 后分页)
	ListComputerLoginHistory(ctx context.Context, req *ADLoginHistoryQuery) ([]models.ADLoginHistory, int64, error)
}

// adLoginHistoryServiceImpl 私有实现
type adLoginHistoryServiceImpl struct {
	db *gorm.DB
}

// NewADLoginHistoryService 创建登录历史查询服务
func NewADLoginHistoryService(db *gorm.DB) ADLoginHistoryService {
	return &adLoginHistoryServiceImpl{db: db}
}

// isPostgres 方言守卫:sys_ad_login_history 为 PG 分区表,
// sqlite(内存单测/dev)下不建表,查询直接返回空。
func (s *adLoginHistoryServiceImpl) isPostgres() bool {
	return s.db != nil && s.db.Dialector.Name() == "postgres"
}

// ListComputerLoginHistory 查询设备登录历史
// 排序由后端固定 login_time DESC(PLAN 风险 13:前端不做 server sorter)
func (s *adLoginHistoryServiceImpl) ListComputerLoginHistory(ctx context.Context, req *ADLoginHistoryQuery) ([]models.ADLoginHistory, int64, error) {
	// 分页钳制(仿 ComputerService.normalizePagination)
	if req.Current <= 0 {
		req.Current = constants.DefaultCurrent
	}
	if req.PageSize <= 0 {
		req.PageSize = constants.DefaultPageSize
	}
	if req.PageSize > constants.MaxListPageSize {
		req.PageSize = constants.MaxListPageSize
	}

	// 非 PG:表不存在,直接返回空列表 + 0(PLAN 风险 8)
	if !s.isPostgres() {
		return []models.ADLoginHistory{}, 0, nil
	}

	query := s.db.WithContext(ctx).Model(&models.ADLoginHistory{}).
		Where("computer_id = ?", req.ComputerID)

	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			query = query.Where("login_time >= ?", t)
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			query = query.Where("login_time <= ?", t)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计登录历史总数失败: %w", err)
	}

	list := make([]models.ADLoginHistory, 0)
	offset := (req.Current - 1) * req.PageSize
	if err := query.Order("login_time DESC").
		Offset(offset).Limit(req.PageSize).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询登录历史失败: %w", err)
	}

	return list, total, nil
}
