package operations

import (
	"context"
	"regexp"

	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// AssetService 资产服务接口
type AssetService interface {
	Create(ctx context.Context, asset *models.Asset) error
	Update(ctx context.Context, asset *models.Asset) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Asset, error)
	GetByDeviceSN(ctx context.Context, deviceSN string) (*models.Asset, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
	GetDeviceTypes(ctx context.Context) ([]DeviceTypeCount, error)
	GetDeviceCategories(ctx context.Context) ([]DeviceTypeCount, error)
	GetStatusValues(ctx context.Context) ([]DeviceTypeCount, error)
	// Statistics 资产统计(按 status/nbf_status 聚合,专用 COUNT 端点,不依赖分页列表)。
	Statistics(ctx context.Context) (*AssetStatisticsResult, error)
}

// AssetStatisticsResult 资产统计结果。
// status: 0=正常 1=停用; nbf_status: 0=否 1=拟报废(独立维度,nbf 计数与 normal/stopped 不互补)。
type AssetStatisticsResult struct {
	Total   int64 `json:"total"`
	Normal  int64 `json:"normal"`  // status = 0
	Stopped int64 `json:"stopped"` // status = 1
	NBF     int64 `json:"nbf"`     // nbf_status = 1
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
		uuidValidator: constants.UUIDPattern,
	}
}

// Statistics 统计资产(按 status + nbf_status 聚合,排除软删除)。
// 替代前端「total*0.8/0.15/0.05」的伪造比例占位实现。
//
// Phase 48 D-07:默认排除 component_type IS NOT NULL 的组件行(与 List() 一致),
// 组件行不参与设备总数/正常/停用/拟报废的统计(避免 1 台交换机 + N 板卡被重复计数)。
func (s *assetService) Statistics(ctx context.Context) (*AssetStatisticsResult, error) {
	var result AssetStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&models.Asset{}).
		Where("component_type IS NULL").
		Select(
			"COUNT(*) AS total",
			"COALESCE(SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END), 0) AS normal",
			"COALESCE(SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END), 0) AS stopped",
			"COALESCE(SUM(CASE WHEN nbf_status = 1 THEN 1 ELSE 0 END), 0) AS nbf",
		).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// assetAllowedSortFields 资产可排序字段白名单(对应 ops_asset 表列名)。
// 注意:key 必须与前端列 dataIndex/json tag 一致(驼峰)才能命中白名单;
// deviceSN 为旧版别名,保留作向后兼容。
var assetAllowedSortFields = map[string]string{
	"deviceSN":           "devicesn",
	"devicesn":           "devicesn",
	"name":               "name",
	"type":               "type",
	"status":             "status",
	"deptId":             "dept_id",
	"userId":             "user_id",
	"deviceTypeName":     "device_type_name",
	"lastInventoryDate":  "last_inventory_date",
	"createdAt":          "created_at",
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

// GetByDeviceSN 根据设备序列号获取资产
func (s *assetService) GetByDeviceSN(ctx context.Context, deviceSN string) (*models.Asset, error) {
	if deviceSN == "" {
		return nil, apperrors.ParamMissing("设备序列号")
	}

	var asset models.Asset
	err := s.db.WithContext(ctx).
		Where("devicesn = ? AND deleted_at IS NULL", deviceSN).
		First(&asset).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 未找到返回 nil，不是错误
		}
		return nil, err
	}

	return &asset, nil
}

// List 查询资产列表
func (s *assetService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	query := s.db.WithContext(ctx).Table("ops_asset")

	// Phase 48 D-07 默认过滤:组件行(component_type IS NOT NULL)不出现在常规列表,
	// 避免 "1 台交换机 + 6 块板卡" 被数成 7 台设备。组件另开独立视图(按 parent_asset_id 聚合)。
	// 硬编码在 service 层(非 controller),保证所有调用入口(含 Excel 导入 GetByDeviceSN 之外的列表
	// 统计路径)默认排除。GetByDeviceSN 不受影响,对账仍能查到组件 SN。
	query = query.Where("component_type IS NULL")

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

	// 用户排序(白名单);无 OrderByColumn 时保留 created_at DESC 默认
	sortReq := extractSortRequest(params)
	query = base.ApplySort(query, sortReq, assetAllowedSortFields)
	if sortReq.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}

	var list []models.Asset
	if err := query.Offset(offset).Limit(pagination.PageSize).Find(&list).Error; err != nil {
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

	// 设备中类筛选
	if deviceCategory, ok := params["deviceCategorySecondName"].(string); ok && deviceCategory != "" {
		query = query.Where("device_category_second_name = ?", deviceCategory)
	}

	// 状态筛选（useStatusLabel）
	if useStatusLabel, ok := params["useStatusLabel"].(string); ok && useStatusLabel != "" {
		query = query.Where("usestatus_label = ?", useStatusLabel)
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

// GetDeviceCategories 获取设备种类列表及数量统计
func (s *assetService) GetDeviceCategories(ctx context.Context) ([]DeviceTypeCount, error) {
	var results []DeviceTypeCount

	err := s.db.WithContext(ctx).Table("ops_asset").
		Select("device_category_second_name as value, COUNT(*) as count").
		Where("device_category_second_name IS NOT NULL AND device_category_second_name != ''").
		Group("device_category_second_name").
		Order("count DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

// GetStatusValues 获取状态列表及数量统计
func (s *assetService) GetStatusValues(ctx context.Context) ([]DeviceTypeCount, error) {
	var results []DeviceTypeCount

	err := s.db.WithContext(ctx).Table("ops_asset").
		Select("usestatus_label as value, COUNT(*) as count").
		Where("usestatus_label IS NOT NULL AND usestatus_label != ''").
		Group("usestatus_label").
		Order("count DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}
