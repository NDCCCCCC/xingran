package monitor

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// LoginLogService 登录日志服务接口
type LoginLogService interface {
	// List 查询登录日志列表
	List(ctx context.Context, params LoginLogListParams) (*PageResult, error)
	// GetByID 获取登录日志详情
	GetByID(ctx context.Context, id string) (*models.LoginLog, error)
	// Delete 删除登录日志
	Delete(ctx context.Context, id string) error
	// BatchDelete 批量删除登录日志
	BatchDelete(ctx context.Context, ids []string) error
	// Clean 清空登录日志
	Clean(ctx context.Context) error
}

// loginLogService 登录日志服务实现
type loginLogService struct {
	db *gorm.DB
}

// NewLoginLogService 创建登录日志服务实例
func NewLoginLogService(db *gorm.DB) LoginLogService {
	return &loginLogService{db: db}
}

// loginLogAllowedSortFields 登录日志可排序字段白名单。
// 值对应 sys_logininfor 表列名(无 JOIN)。
// 注意:前端列 dataIndex/json tag 是驼峰 ipAddr(userName/loginTime/status 同),
// 此处 key 必须与前端 dataIndex 一致才能命中白名单;旧小写 ipaddr 仅作向后兼容兜底。
var loginLogAllowedSortFields = map[string]string{
	"userName":  "user_name",
	"nickname":  "nickname",
	"ipAddr":    "ipaddr",
	"ipaddr":    "ipaddr",
	"loginTime": "login_time",
	"status":    "status",
}

// LoginLogListParams 登录日志列表查询参数
type LoginLogListParams struct {
	base.BaseListRequest
	Username  *string
	IPAddr    *string
	Status    *int
	BeginTime *string
	EndTime   *string
}

// DefaultLoginLogListParams 默认列表参数
func DefaultLoginLogListParams() LoginLogListParams {
	return LoginLogListParams{
		BaseListRequest: base.BaseListRequest{
			Current:  1,
			PageSize: 10,
		},
	}
}

// List 查询登录日志列表
func (s *loginLogService) List(ctx context.Context, params LoginLogListParams) (*PageResult, error) {
	var total int64
	var list []models.LoginLog

	query := s.db.WithContext(ctx).Model(&models.LoginLog{})

	// 用户名模糊查询
	if params.Username != nil && *params.Username != "" {
		query = query.Where("user_name LIKE ?", "%"+*params.Username+"%")
	}

	// IP地址模糊查询
	if params.IPAddr != nil && *params.IPAddr != "" {
		query = query.Where("ipaddr LIKE ?", "%"+*params.IPAddr+"%")
	}

	// 状态查询
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	// 开始时间查询
	if params.BeginTime != nil && *params.BeginTime != "" {
		query = query.Where("login_time >= ?", *params.BeginTime)
	}

	// 结束时间查询
	if params.EndTime != nil && *params.EndTime != "" {
		query = query.Where("login_time <= ?", *params.EndTime)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询登录日志总数失败: %w", err)
	}

	// 查询分页数据 - 用户排序(白名单)优先,无 OrderByColumn 时保留 login_time DESC 默认
	offset := (params.Current - 1) * params.PageSize
	query = base.ApplySort(query, params.BaseListRequest, loginLogAllowedSortFields)
	if params.OrderByColumn == "" {
		query = query.Order("login_time DESC")
	}
	if err := query.Offset(offset).Limit(params.PageSize).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询登录日志列表失败: %w", err)
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// GetByID 获取登录日志详情
func (s *loginLogService) GetByID(ctx context.Context, id string) (*models.LoginLog, error) {
	var loginLog models.LoginLog
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&loginLog).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("登录日志不存在")
		}
		return nil, fmt.Errorf("查询登录日志失败: %w", err)
	}
	return &loginLog, nil
}

// Delete 删除登录日志
func (s *loginLogService) Delete(ctx context.Context, id string) error {
	// 检查登录日志是否存在
	var loginLog models.LoginLog
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&loginLog).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("登录日志不存在")
		}
		return fmt.Errorf("查询登录日志失败: %w", err)
	}

	// 删除登录日志
	if err := s.db.WithContext(ctx).Where("id = ?", id).Delete(&models.LoginLog{}).Error; err != nil {
		return fmt.Errorf("删除登录日志失败: %w", err)
	}

	return nil
}

// BatchDelete 批量删除登录日志
func (s *loginLogService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids不能为空")
	}

	// 批量删除登录日志
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.LoginLog{}).Error; err != nil {
		return fmt.Errorf("批量删除登录日志失败: %w", err)
	}

	return nil
}

// Clean 清空登录日志
func (s *loginLogService) Clean(ctx context.Context) error {
	if err := s.db.WithContext(ctx).Exec("DELETE FROM sys_login_log").Error; err != nil {
		return fmt.Errorf("清空登录日志失败: %w", err)
	}
	return nil
}
