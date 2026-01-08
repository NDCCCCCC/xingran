package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// DictCacheService 字典缓存服务
type DictCacheService struct {
	db        *gorm.DB
	dataCache *DataCacheService
}

// NewDictCacheService 创建字典缓存服务
func NewDictCacheService(db *gorm.DB, dataCache *DataCacheService) *DictCacheService {
	return &DictCacheService{
		db:        db,
		dataCache: dataCache,
	}
}

// GetDictDataByTypeWithCache 根据字典类型获取字典数据（带缓存）
func (s *DictCacheService) GetDictDataByTypeWithCache(ctx context.Context, dictType string) ([]*models.DictData, error) {
	if dictType == "" {
		return nil, fmt.Errorf("字典类型不能为空")
	}

	cacheKey := GetDictDataByTypeKey(dictType)
	var dictData []*models.DictData

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &dictData, 30*time.Minute, func() (interface{}, error) {
		return s.queryDictDataByType(dictType)
	})

	if err != nil {
		return nil, fmt.Errorf("获取字典数据失败: %w", err)
	}

	return dictData, nil
}

// queryDictDataByType 从数据库查询字典数据
func (s *DictCacheService) queryDictDataByType(dictType string) ([]*models.DictData, error) {
	var dictData []models.DictData

	if err := s.db.Where("dict_type = ? AND status = ?", dictType, 0).
		Order("dict_sort ASC, dict_value ASC").
		Find(&dictData).Error; err != nil {
		return nil, fmt.Errorf("查询字典数据失败: %w", err)
	}

	// 转换为指针数组
	dictDataPtrs := make([]*models.DictData, len(dictData))
	for i := range dictData {
		dictDataPtrs[i] = &dictData[i]
	}

	return dictDataPtrs, nil
}

// GetDictTypesWithCache 获取所有字典类型（带缓存）
func (s *DictCacheService) GetDictTypesWithCache(ctx context.Context) ([]*models.DictType, error) {
	cacheKey := CacheKeyDictType
	var dictTypes []*models.DictType

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &dictTypes, 30*time.Minute, func() (interface{}, error) {
		return s.queryDictTypes()
	})

	if err != nil {
		return nil, fmt.Errorf("获取字典类型失败: %w", err)
	}

	return dictTypes, nil
}

// queryDictTypes 从数据库查询字典类型
func (s *DictCacheService) queryDictTypes() ([]*models.DictType, error) {
	var dictTypes []models.DictType

	if err := s.db.Where("status = ?", 0).
		Order("dict_sort ASC").
		Find(&dictTypes).Error; err != nil {
		return nil, fmt.Errorf("查询字典类型失败: %w", err)
	}

	// 转换为指针数组
	dictTypePtrs := make([]*models.DictType, len(dictTypes))
	for i := range dictTypes {
		dictTypePtrs[i] = &dictTypes[i]
	}

	return dictTypePtrs, nil
}

// InvalidateDictCache 清除字典相关缓存
func (s *DictCacheService) InvalidateDictCache(ctx context.Context) error {
	patterns := []string{
		CacheKeyDictType + "*",
		CacheKeyDictDataByType + "*",
	}

	for _, pattern := range patterns {
		if err := s.dataCache.DeleteByPattern(ctx, pattern); err != nil {
			return fmt.Errorf("清除缓存失败 %s: %w", pattern, err)
		}
	}

	return nil
}

// InvalidateDictDataByType 清除指定类型的字典数据缓存
func (s *DictCacheService) InvalidateDictDataByType(ctx context.Context, dictType string) error {
	cacheKey := GetDictDataByTypeKey(dictType)
	if err := s.dataCache.Delete(ctx, cacheKey); err != nil {
		return fmt.Errorf("清除缓存失败 %s: %w", cacheKey, err)
	}
	return nil
}

// CreateDictType 创建字典类型
func (s *DictCacheService) CreateDictType(ctx context.Context, dictType *models.DictType) error {
	if err := s.db.Create(dictType).Error; err != nil {
		return fmt.Errorf("创建字典类型失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateDictCache(ctx); err != nil {
		logger.Warnf("[DICT] 清除缓存失败: %v", err)
	}

	return nil
}

// UpdateDictType 更新字典类型
func (s *DictCacheService) UpdateDictType(ctx context.Context, dictType *models.DictType) error {
	if err := s.db.Save(dictType).Error; err != nil {
		return fmt.Errorf("更新字典类型失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateDictCache(ctx); err != nil {
		logger.Warnf("[DICT] 清除缓存失败: %v", err)
	}

	return nil
}

// DeleteDictType 删除字典类型
func (s *DictCacheService) DeleteDictType(ctx context.Context, id string) error {
	if err := s.db.Delete(&models.DictType{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("删除字典类型失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateDictCache(ctx); err != nil {
		logger.Warnf("[DICT] 清除缓存失败: %v", err)
	}

	return nil
}

// CreateDictData 创建字典数据
func (s *DictCacheService) CreateDictData(ctx context.Context, dictData *models.DictData) error {
	if err := s.db.Create(dictData).Error; err != nil {
		return fmt.Errorf("创建字典数据失败: %w", err)
	}

	// 清除该类型的缓存
	if err := s.InvalidateDictDataByType(ctx, dictData.DictType); err != nil {
		logger.Warnf("[DICT] 清除缓存失败: %v", err)
	}

	return nil
}

// UpdateDictData 更新字典数据
func (s *DictCacheService) UpdateDictData(ctx context.Context, dictData *models.DictData) error {
	if err := s.db.Save(dictData).Error; err != nil {
		return fmt.Errorf("更新字典数据失败: %w", err)
	}

	// 清除该类型的缓存
	if err := s.InvalidateDictDataByType(ctx, dictData.DictType); err != nil {
		logger.Warnf("[DICT] 清除缓存失败: %v", err)
	}

	return nil
}

// DeleteDictData 删除字典数据
func (s *DictCacheService) DeleteDictData(ctx context.Context, id string, dictType string) error {
	if err := s.db.Delete(&models.DictData{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("删除字典数据失败: %w", err)
	}

	// 清除该类型的缓存
	if err := s.InvalidateDictDataByType(ctx, dictType); err != nil {
		logger.Warnf("[DICT] 清除缓存失败: %v", err)
	}

	return nil
}

// GetDictDataByCode 根据字典类型和字典值获取字典标签
func (s *DictCacheService) GetDictDataByCode(ctx context.Context, dictType, dictValue string) (string, error) {
	dictDataList, err := s.GetDictDataByTypeWithCache(ctx, dictType)
	if err != nil {
		return "", err
	}

	for _, data := range dictDataList {
		if data.DictValue == dictValue {
			return data.DictLabel, nil
		}
	}

	return "", fmt.Errorf("未找到字典数据: type=%s, value=%s", dictType, dictValue)
}

// GetDictDataMap 获取字典数据映射（value -> label）
func (s *DictCacheService) GetDictDataMap(ctx context.Context, dictType string) (map[string]string, error) {
	dictDataList, err := s.GetDictDataByTypeWithCache(ctx, dictType)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, data := range dictDataList {
		result[data.DictValue] = data.DictLabel
	}

	return result, nil
}
