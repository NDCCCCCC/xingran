package portcollection

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// QueryService 端口状态查询服务
type QueryService struct {
	db *gorm.DB
}

// NewQueryService 创建查询服务
func NewQueryService(db *gorm.DB) *QueryService {
	return &QueryService{db: db}
}

// GetList 获取端口状态列表
func (s *QueryService) GetList(ctx context.Context, req *ListRequest) ([]models.DevicePortStatus, int64, error) {
	var portStatuses []models.DevicePortStatus
	var total int64

	query := s.db.WithContext(ctx).Model(&models.DevicePortStatus{})

	if req.DeviceID != "" {
		query = query.Where("device_id = ?", req.DeviceID)
	}
	if req.InterfaceName != "" {
		query = query.Where("interface_name LIKE ?", "%"+req.InterfaceName+"%")
	}
	if req.AdminStatus != "" {
		query = query.Where("admin_status = ?", req.AdminStatus)
	}
	if req.OperStatus != "" {
		query = query.Where("oper_status = ?", req.OperStatus)
	}
	if req.Dot1xEnabled != nil {
		query = query.Where("dot1x_enabled = ?", *req.Dot1xEnabled)
	}
	if req.PortSecurityEnabled != nil {
		query = query.Where("port_security_enabled = ?", *req.PortSecurityEnabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询端口状态总数失败: %w", err)
	}

	offset := (req.Current - 1) * req.PageSize
	// 用户排序(白名单)优先,无 OrderByColumn 时保留 collected_at DESC 默认
	query = base.ApplySort(query, req.BaseListRequest, portStatusAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("collected_at DESC")
	}
	if err := query.Offset(offset).Limit(req.PageSize).Find(&portStatuses).Error; err != nil {
		return nil, 0, fmt.Errorf("查询端口状态列表失败: %w", err)
	}

	return portStatuses, total, nil
}

// GetStats 获取端口状态统计信息
func (s *QueryService) GetStats(ctx context.Context) (*StatsResult, error) {
	var stats StatsResult

	s.db.WithContext(ctx).Model(&models.DevicePortStatus{}).Count(&stats.TotalRecords)
	s.db.WithContext(ctx).Model(&models.DevicePortStatus{}).Select("COUNT(DISTINCT device_id)").Scan(&stats.UniqueDevices)
	s.db.WithContext(ctx).Model(&models.DevicePortStatus{}).Where("dot1x_enabled = ?", true).Count(&stats.Dot1xEnabledCount)
	s.db.WithContext(ctx).Model(&models.DevicePortStatus{}).Where("port_security_enabled = ?", true).Count(&stats.SecurityEnabledCount)
	s.db.WithContext(ctx).Model(&models.DevicePortStatus{}).Where("oper_status = ?", "up").Count(&stats.UpPortsCount)
	s.db.WithContext(ctx).Model(&models.DevicePortStatus{}).Where("oper_status = ?", "down").Count(&stats.DownPortsCount)

	var latest models.DevicePortStatus
	s.db.WithContext(ctx).Order("collected_at DESC").First(&latest)
	if latest.ID != "" {
		stats.LatestCollection = &latest.CollectedAt
	}

	return &stats, nil
}

// CleanOldRecords 清理旧的端口状态记录
func (s *QueryService) CleanOldRecords(ctx context.Context, days int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -days)

	result := s.db.WithContext(ctx).
		Where("collected_at < ?", cutoffTime).
		Delete(&models.DevicePortStatus{})

	if result.Error != nil {
		return 0, fmt.Errorf("清理旧记录失败: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// ListRequest 查询请求
type ListRequest struct {
	base.BaseListRequest
	DeviceID            string `json:"deviceId"`
	InterfaceName       string `json:"interfaceName"`
	AdminStatus         string `json:"adminStatus"`
	OperStatus          string `json:"operStatus"`
	Dot1xEnabled        *bool  `json:"dot1xEnabled"`
	PortSecurityEnabled *bool  `json:"portSecurityEnabled"`
}

// portStatusAllowedSortFields 端口状态可排序字段白名单(对应 sys_device_port_status 表列名)。
var portStatusAllowedSortFields = map[string]string{
	"deviceId":      "device_id",
	"interfaceName": "interface_name",
	"adminStatus":   "admin_status",
	"operStatus":    "oper_status",
	"vlan":          "vlan",
	"collectedAt":   "collected_at",
}

// StatsResult 统计结果
type StatsResult struct {
	TotalRecords         int64      `json:"totalRecords"`
	UniqueDevices        int64      `json:"uniqueDevices"`
	Dot1xEnabledCount    int64      `json:"dot1xEnabledCount"`
	SecurityEnabledCount int64      `json:"securityEnabledCount"`
	UpPortsCount         int64      `json:"upPortsCount"`
	DownPortsCount       int64      `json:"downPortsCount"`
	LatestCollection     *time.Time `json:"latestCollection"`
}
