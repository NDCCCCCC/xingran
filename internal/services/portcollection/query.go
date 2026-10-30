package portcollection

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
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
	// 用户排序(白名单)优先,无 OrderByColumn 时保留 collected_at DESC 默认。
	// 特殊: interfaceName 走"板卡号/子卡号/接口号"数值排序 —— 字符串字典序会让
	// GE0/10 排在 GE0/2 之前(因为 '1' < '2'),业务上错误。仅 interfaceName 绕过
	// ApplySort,其余白名单字段(deviceId/adminStatus/operStatus/vlan/collectedAt)
	// 仍走字符串排序,行为不变。
	if req.OrderByColumn == "interfaceName" {
		if _, direction, ok := base.ResolveSort(req.BaseListRequest, portStatusAllowedSortFields); ok {
			query = query.Order(interfaceNameSortExpr(s.db.Dialector.Name(), direction))
		}
	} else {
		query = base.ApplySort(query, req.BaseListRequest, portStatusAllowedSortFields)
		if req.OrderByColumn == "" {
			query = query.Order("collected_at DESC")
		}
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

// interfaceNameSortExpr 返回按"速率前缀 + 板卡号/子卡号/接口号"数值排序的 ORDER BY 表达式。
//
// 背景: 接口名归一化后形如 GE0/1 / GE0/0/1 / XGE1/0/24,字符串字典序会让
// GE0/10 < GE0/2(因为 '1' < '2'),业务上错误,必须按数字段数值排序。
//
// 实现按 DB dialect 分支(遵循 internal/services/asset/reconciliation_service.go 惯例):
//   - postgres: 三个排序键 ——
//     1. 字母前缀(regexp_replace 取首个数字及之后部分为空): GE/XGE/HGE/FOE/FE 等,
//        把不同速率分开,避免 FE0/2 夹在 GE0/1 与 GE0/3 之间交替。
//     2. 数字段 int 数组(regexp_matches(interface_name,'([0-9]+)','g')): PG 数组逐元素
//        比较等价 (板卡号,子卡号,接口号) 元组比较,两段(GE0/1)和三段(GE0/0/1)都正确。
//        COALESCE 兜底无数字接口名(Vlan/NULL 等)为空数组,排末尾。
//     3. interface_name: 稳定 tiebreaker。
//   - 其他(sqlite 等开发/测试 dialect): 降级为字符串排序。端口采集主库为 PG,
//     sqlite 仅测试用,降级可接受。
//
// direction 由 base.ResolveSort 保证只取 "ASC"/"DESC" 字面量,dialectName 来自 gorm
// Dialector.Name(),均非用户输入 → 无 SQL 注入风险。
func interfaceNameSortExpr(dialectName, direction string) string {
	if dialectName == "postgres" {
		return fmt.Sprintf(
			`regexp_replace(interface_name, '[0-9].*$', '') %s, (SELECT COALESCE(array_agg(m[1]::int), ARRAY[]::int[]) FROM regexp_matches(interface_name, '([0-9]+)', 'g') AS m) %s, interface_name %s`,
			direction, direction, direction,
		)
	}
	return fmt.Sprintf(`interface_name %s`, direction)
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
