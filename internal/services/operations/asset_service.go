package operations

import (
	"context"
	"regexp"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// AssetService 资产服务接口
type AssetService interface {
	Create(ctx context.Context, asset *models.Asset) error
	Update(ctx context.Context, asset *models.Asset) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Asset, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
	GetDeviceTypes(ctx context.Context) ([]DeviceTypeCount, error)
}

// DeviceTypeCount 设备类型统计
type DeviceTypeCount struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

type assetService struct {
	db            *gorm.DB
	uuidValidator *regexp.Regexp
}

// NewAssetService 创建资产服务实例
func NewAssetService(db *gorm.DB) AssetService {
	return &assetService{
		db:            db,
		uuidValidator: regexp.MustCompile(uuidPattern),
	}
}

// Create 创建资产
func (s *assetService) Create(ctx context.Context, asset *models.Asset) error {
	// 验证部门存在性（如果提供了 dept_id）
	if asset.DeptID != nil && *asset.DeptID != "" {
		if err := s.validateDept(ctx, *asset.DeptID); err != nil {
			return err
		}
	}

	// 验证用户存在性（如果提供了 user_id）
	if asset.UserID != nil && *asset.UserID != "" {
		if err := s.validateUser(ctx, *asset.UserID); err != nil {
			return err
		}
	}

	// 验证设备序列号唯一性
	if err := s.validateDeviceSNUnique(ctx, asset.DeviceSN, ""); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Create(asset).Error
}

// Update 更新资产
func (s *assetService) Update(ctx context.Context, asset *models.Asset) error {
	// 验证部门存在性
	if asset.DeptID != nil && *asset.DeptID != "" {
		if err := s.validateDept(ctx, *asset.DeptID); err != nil {
			return err
		}
	}

	// 验证用户存在性
	if asset.UserID != nil && *asset.UserID != "" {
		if err := s.validateUser(ctx, *asset.UserID); err != nil {
			return err
		}
	}

	// 验证设备序列号唯一性（排除自身）
	if err := s.validateDeviceSNUnique(ctx, asset.DeviceSN, asset.ID); err != nil {
		return err
	}

	// 使用 Omit 排除 CreatedAt 字段
	return s.db.WithContext(ctx).Omit("CreatedAt").Save(asset).Error
}

// Delete 删除资产
func (s *assetService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&models.Asset{}, "id = ?", id).Error
}

// GetByID 根据ID获取资产
func (s *assetService) GetByID(ctx context.Context, id string) (*models.Asset, error) {
	var asset models.Asset
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&asset).Error
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

// List 查询资产列表
func (s *assetService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	query := s.db.WithContext(ctx).Table("ops_asset")

	// 应用筛选条件
	query = s.applyFilters(query, params)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 应用分页
	pagination := extractPagination(params)
	offset := calculateOffset(pagination)

	var list []models.Asset
	if err := query.Offset(offset).Limit(pagination.PageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  pagination.Current,
		PageSize: pagination.PageSize,
	}, nil
}

// BatchDelete 批量删除资产
func (s *assetService) BatchDelete(ctx context.Context, ids []string) error {
	return s.db.WithContext(ctx).Delete(&models.Asset{}, "id IN ?", ids).Error
}

// applyFilters 应用查询筛选条件
func (s *assetService) applyFilters(query *gorm.DB, params map[string]interface{}) *gorm.DB {
	// 设备序列号筛选
	if deviceSN, ok := params["devicesn"].(string); ok && deviceSN != "" {
		query = query.Where("devicesn LIKE ?", "%"+deviceSN+"%")
	}

	// 设备型号筛选
	if deviceModel, ok := params["deviceModelName"].(string); ok && deviceModel != "" {
		query = query.Where("device_model_name LIKE ?", "%"+deviceModel+"%")
	}

	// 部门筛选（包含子部门）
	if deptId, ok := params["deptId"].(string); ok && deptId != "" {
		query = s.applyDeptFilter(query, deptId)
	}

	// 用户筛选
	if userId, ok := params["userId"].(string); ok && userId != "" {
		query = query.Where("user_id = ?", userId)
	}

	// 状态筛选
	if status := extractIntParam(params, "status", -1); status >= 0 {
		query = query.Where("status = ?", status)
	}

	// 是否拟报废筛选
	if nbfStatus := extractIntParam(params, "nbfStatus", -1); nbfStatus >= 0 {
		query = query.Where("nbf_status = ?", nbfStatus)
	}

	// 设备类型筛选（模糊匹配，兼容空格和大小写差异）
	if deviceType, ok := params["deviceTypeName"].(string); ok && deviceType != "" {
		query = query.Where("device_type_name LIKE ?", "%"+deviceType+"%")
	}

	return query
}

// applyDeptFilter 应用部门筛选（包含子部门）
func (s *assetService) applyDeptFilter(query *gorm.DB, deptId string) *gorm.DB {
	var deptIDs []string

	// 查询该部门及其所有子部门的ID
	err := s.db.Table("sys_dept").
		Where("id = ? OR ancestors LIKE ? OR ancestors LIKE ? OR ancestors = ?",
			deptId, "%,"+deptId+",%", "%,"+deptId, deptId).
		Pluck("id", &deptIDs).Error

	if err != nil || len(deptIDs) == 0 {
		return query.Where("1 = 0")
	}

	return query.Where("dept_id IN ?", deptIDs)
}

// validateDept 验证部门存在性（复用 building_service.go 的 validateOrg 模式）
func (s *assetService) validateDept(ctx context.Context, deptID string) error {
	// 验证UUID格式
	if !s.uuidValidator.MatchString(deptID) {
		return apperrors.ParamInvalid("所属部门ID格式无效：必须是有效的UUID格式")
	}

	// 验证部门是否存在
	var count int64
	if err := s.db.WithContext(ctx).Table("sys_dept").Where("id = ?", deptID).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return apperrors.ParamInvalid("所属部门不存在")
	}

	return nil
}

// validateUser 验证用户存在性
func (s *assetService) validateUser(ctx context.Context, userID string) error {
	// 验证UUID格式
	if !s.uuidValidator.MatchString(userID) {
		return apperrors.ParamInvalid("所属用户ID格式无效：必须是有效的UUID格式")
	}

	// 验证用户是否存在
	var count int64
	if err := s.db.WithContext(ctx).Table("sys_user").Where("id = ?", userID).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return apperrors.ParamInvalid("所属用户不存在")
	}

	return nil
}

// validateDeviceSNUnique 验证设备序列号唯一性
func (s *assetService) validateDeviceSNUnique(ctx context.Context, deviceSN, excludeID string) error {
	var count int64
	query := s.db.WithContext(ctx).Table("ops_asset").Where("devicesn = ?", deviceSN)

	// 更新时排除当前记录
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return apperrors.ParamInvalid("设备序列号已存在")
	}

	return nil
}

// GetDeviceTypes 获取设备类型列表及数量统计
func (s *assetService) GetDeviceTypes(ctx context.Context) ([]DeviceTypeCount, error) {
	var results []DeviceTypeCount

	err := s.db.WithContext(ctx).Table("ops_asset").
		Select("device_type_name as value, COUNT(*) as count").
		Where("device_type_name IS NOT NULL AND device_type_name != ''").
		Group("device_type_name").
		Order("count DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}
