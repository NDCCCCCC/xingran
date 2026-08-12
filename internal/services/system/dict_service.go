package system

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// DictTypeService 字典类型服务接口
type DictTypeService interface {
	Create(ctx context.Context, req *requests.DictTypeCreateRequest) error
	Update(ctx context.Context, req *requests.DictTypeUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.DictType, error)
	List(ctx context.Context, params requests.DictTypeListParams) (*PageResult, error)

	// 新增缓存方法
	GetAllWithCache(ctx context.Context) ([]*models.DictType, error)
	// Statistics 字典类型统计(专用 COUNT 聚合,不依赖分页列表,不受 MaxPageSize=100 钳制)。
	Statistics(ctx context.Context) (*DictTypeStatisticsResult, error)
}

// DictTypeStatisticsResult 字典类型统计结果(status: 0=正常 1=停用)。
type DictTypeStatisticsResult struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Inactive int64 `json:"inactive"`
}

// dictTypeService 字典类型服务实现
type dictTypeService struct {
	db *gorm.DB
}

// NewDictTypeService 创建字典类型服务实例
func NewDictTypeService(db *gorm.DB) DictTypeService {
	return &dictTypeService{db: db}
}

// Statistics 统计字典类型(按 status 聚合,排除软删除)。
// 不依赖分页列表,避免「用 pageSize:1000 拉全量再 .length」被 MaxPageSize=100 钳制。
func (s *dictTypeService) Statistics(ctx context.Context) (*DictTypeStatisticsResult, error) {
	var result DictTypeStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&models.DictType{}).
		Select(
			"COUNT(*) AS total",
			"COALESCE(SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END), 0) AS active",
			"COALESCE(SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END), 0) AS inactive",
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计字典类型失败: %w", err)
	}
	return &result, nil
}

// ==================== 服务方法实现 ====================

func (s *dictTypeService) Create(ctx context.Context, req *requests.DictTypeCreateRequest) error {
	// 检查字典类型是否已存在
	var existDict models.DictType
	if err := s.db.WithContext(ctx).Where("dict_type = ?", req.DictType).First(&existDict).Error; err == nil {
		return fmt.Errorf("字典类型已存在")
	}

	dictType := models.DictType{
		DictName: req.DictName,
		DictType: req.DictType,
		Status:   req.Status,
		Remark:   toStringPtr(req.Remark),
	}

	if err := s.db.WithContext(ctx).Create(&dictType).Error; err != nil {
		return fmt.Errorf("创建字典类型失败: %w", err)
	}

	return nil
}

func (s *dictTypeService) Update(ctx context.Context, req *requests.DictTypeUpdateRequest) error {
	var dictType models.DictType
	if err := s.db.WithContext(ctx).First(&dictType, "id = ?", req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("字典类型不存在")
		}
		return fmt.Errorf("查询字典类型失败: %w", err)
	}

	dictType.DictName = req.DictName
	dictType.Status = req.Status
	dictType.Remark = toStringPtr(req.Remark)

	if err := s.db.WithContext(ctx).Save(&dictType).Error; err != nil {
		return fmt.Errorf("更新字典类型失败: %w", err)
	}

	return nil
}

func (s *dictTypeService) Delete(ctx context.Context, id string) error {
	var dictType models.DictType
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&dictType).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("字典类型不存在")
		}
		return fmt.Errorf("查询字典类型失败: %w", err)
	}

	// 检查是否有字典数据
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.DictData{}).Where("dict_type = ?", dictType.DictType).Count(&count).Error; err == nil && count > 0 {
		return fmt.Errorf("该字典类型下存在字典数据，不能删除")
	}

	if err := s.db.WithContext(ctx).Delete(&dictType).Error; err != nil {
		return fmt.Errorf("删除字典类型失败: %w", err)
	}

	return nil
}

func (s *dictTypeService) GetByID(ctx context.Context, id string) (*models.DictType, error) {
	var dictType models.DictType
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&dictType).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("字典类型不存在")
		}
		return nil, fmt.Errorf("查询字典类型失败: %w", err)
	}
	return &dictType, nil
}

func (s *dictTypeService) List(ctx context.Context, params requests.DictTypeListParams) (*PageResult, error) {
	var total int64
	var list []models.DictType

	query := s.db.WithContext(ctx).Model(&models.DictType{})

	if params.DictName != nil && *params.DictName != "" {
		query = query.Where("dict_name LIKE ?", "%"+*params.DictName+"%")
	}
	if params.DictType != nil && *params.DictType != "" {
		query = query.Where("dict_type LIKE ?", "%"+*params.DictType+"%")
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计字典类型总数失败: %w", err)
	}

	offset := (params.Current - 1) * params.PageSize
	query = base.ApplySort(query, params.BaseListRequest, dictTypeAllowedSortFields)
	if params.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Offset(offset).Limit(params.PageSize).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询字典类型列表失败: %w", err)
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// GetAllWithCache 获取所有字典类型（无缓存版本，直接查询数据库）
func (s *dictTypeService) GetAllWithCache(ctx context.Context) ([]*models.DictType, error) {
	var types []models.DictType
	if err := s.db.WithContext(ctx).Where("status = ?", 0).
		Order("dict_sort ASC").Find(&types).Error; err != nil {
		return nil, fmt.Errorf("查询字典类型失败: %w", err)
	}
	result := make([]*models.DictType, len(types))
	for i := range types {
		result[i] = &types[i]
	}
	return result, nil
}

// DictDataService 字典数据服务接口
type DictDataService interface {
	Create(ctx context.Context, req *requests.DictDataCreateRequest) error
	Update(ctx context.Context, req *requests.DictDataUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.DictData, error)
	List(ctx context.Context, params requests.DictDataListParams) (*PageResult, error)

	// 新增缓存方法
	GetByTypeWithCache(ctx context.Context, dictType string) ([]*models.DictData, error)
	// Statistics 字典数据统计(专用 COUNT 聚合,可选按 dictType 过滤,不受 MaxPageSize=100 钳制)。
	Statistics(ctx context.Context, dictType string) (*DictDataStatisticsResult, error)
}

// DictDataStatisticsResult 字典数据统计结果(status: 0=正常 1=停用)。
type DictDataStatisticsResult struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Inactive int64 `json:"inactive"`
}

// dictDataService 字典数据服务实现
type dictDataService struct {
	db *gorm.DB
}

// NewDictDataService 创建字典数据服务实例
func NewDictDataService(db *gorm.DB) DictDataService {
	return &dictDataService{db: db}
}

// Statistics 统计字典数据(按 status 聚合,可选按 dictType 过滤,排除软删除)。
// dictType 为空时统计全部。不依赖分页列表,避免被 MaxPageSize=100 钳制。
func (s *dictDataService) Statistics(ctx context.Context, dictType string) (*DictDataStatisticsResult, error) {
	var result DictDataStatisticsResult
	q := s.db.WithContext(ctx).Model(&models.DictData{})
	if dictType != "" {
		q = q.Where("dict_type = ?", dictType)
	}
	err := q.Select(
		"COUNT(*) AS total",
		"COALESCE(SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END), 0) AS active",
		"COALESCE(SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END), 0) AS inactive",
	).Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计字典数据失败: %w", err)
	}
	return &result, nil
}

// dictTypeAllowedSortFields 字典类型可排序字段白名单。
var dictTypeAllowedSortFields = map[string]string{
	"dictName":  "dict_name",
	"dictType":  "dict_type",
	"status":    "status",
	"createdAt": "created_at",
}

// dictDataAllowedSortFields 字典数据可排序字段白名单。
var dictDataAllowedSortFields = map[string]string{
	"dictLabel": "dict_label",
	"dictValue": "dict_value",
	"dictSort":  "dict_sort",
	"status":    "status",
	"createdAt": "created_at",
}

func (s *dictDataService) Create(ctx context.Context, req *requests.DictDataCreateRequest) error {
	// 检查字典类型是否存在
	var dictType models.DictType
	if err := s.db.WithContext(ctx).Where("dict_type = ?", req.DictType).First(&dictType).Error; err != nil {
		return fmt.Errorf("字典类型不存在")
	}

	dictData := models.DictData{
		DictSort:  req.DictSort,
		DictLabel: req.DictLabel,
		DictValue: req.DictValue,
		DictType:  req.DictType,
		CssClass:  req.CssClass,
		ListClass: req.ListClass,
		IsDefault: req.IsDefault,
		Status:    req.Status,
		Remark:    toStringPtr(req.Remark),
	}

	if err := s.db.WithContext(ctx).Create(&dictData).Error; err != nil {
		return fmt.Errorf("创建字典数据失败: %w", err)
	}

	return nil
}

func (s *dictDataService) Update(ctx context.Context, req *requests.DictDataUpdateRequest) error {
	var dictData models.DictData
	if err := s.db.WithContext(ctx).First(&dictData, "id = ?", req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("字典数据不存在")
		}
		return fmt.Errorf("查询字典数据失败: %w", err)
	}

	dictData.DictSort = req.DictSort
	dictData.DictLabel = req.DictLabel
	dictData.DictValue = req.DictValue
	dictData.CssClass = req.CssClass
	dictData.ListClass = req.ListClass
	dictData.IsDefault = req.IsDefault
	dictData.Status = req.Status
	dictData.Remark = toStringPtr(req.Remark)

	if err := s.db.WithContext(ctx).Save(&dictData).Error; err != nil {
		return fmt.Errorf("更新字典数据失败: %w", err)
	}

	return nil
}

func (s *dictDataService) Delete(ctx context.Context, id string) error {
	var dictData models.DictData
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&dictData).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("字典数据不存在")
		}
		return fmt.Errorf("查询字典数据失败: %w", err)
	}

	if err := s.db.WithContext(ctx).Delete(&dictData).Error; err != nil {
		return fmt.Errorf("删除字典数据失败: %w", err)
	}

	return nil
}

func (s *dictDataService) GetByID(ctx context.Context, id string) (*models.DictData, error) {
	var dictData models.DictData
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&dictData).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("字典数据不存在")
		}
		return nil, fmt.Errorf("查询字典数据失败: %w", err)
	}
	return &dictData, nil
}

func (s *dictDataService) List(ctx context.Context, params requests.DictDataListParams) (*PageResult, error) {
	var total int64
	var list []models.DictData

	query := s.db.WithContext(ctx).Model(&models.DictData{}).Where("dict_type = ?", params.DictType)

	if params.DictLabel != nil && *params.DictLabel != "" {
		query = query.Where("dict_label LIKE ?", "%"+*params.DictLabel+"%")
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计字典数据总数失败: %w", err)
	}

	offset := (params.Current - 1) * params.PageSize
	query = base.ApplySort(query, params.BaseListRequest, dictDataAllowedSortFields)
	if params.OrderByColumn == "" {
		query = query.Order("dict_sort ASC")
	}
	if err := query.Offset(offset).Limit(params.PageSize).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询字典数据列表失败: %w", err)
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// GetByTypeWithCache 根据类型获取字典数据（无缓存版本，直接查询数据库）
func (s *dictDataService) GetByTypeWithCache(ctx context.Context, dictType string) ([]*models.DictData, error) {
	var data []models.DictData
	if err := s.db.WithContext(ctx).Where("dict_type = ? AND status = ?", dictType, 0).
		Order("dict_sort ASC").Find(&data).Error; err != nil {
		return nil, fmt.Errorf("查询字典数据失败: %w", err)
	}
	result := make([]*models.DictData, len(data))
	for i := range data {
		result[i] = &data[i]
	}
	return result, nil
}
