package utils

import (
	"fmt"
	"gorm.io/gorm"
)

// CheckExists 检查记录是否存在
func CheckExists(db *gorm.DB, model interface{}, where string, args ...interface{}) (bool, error) {
	var count int64
	if err := db.Model(model).Where(where, args...).Count(&count).Error; err != nil {
		return false, fmt.Errorf("检查记录是否存在失败: %w", err)
	}
	return count > 0, nil
}

// CheckExistsExclude 检查记录是否存在（排除指定ID）
func CheckExistsExclude(db *gorm.DB, model interface{}, excludeID, field string, value interface{}) (bool, error) {
	var count int64
	query := db.Model(model).Where(field+" = ?", value)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("检查记录是否存在失败: %w", err)
	}
	return count > 0, nil
}

// GetByID 根据ID获取记录
func GetByID(db *gorm.DB, model interface{}, id string) error {
	if err := db.Where("id = ?", id).First(model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("记录不存在")
		}
		return fmt.Errorf("查询记录失败: %w", err)
	}
	return nil
}

// DeleteByID 根据ID删除记录
func DeleteByID(db *gorm.DB, model interface{}, id string) error {
	if err := db.Delete(model, "id = ?", id).Error; err != nil {
		return fmt.Errorf("删除记录失败: %w", err)
	}
	return nil
}

// SoftDeleteByID 软删除记录
func SoftDeleteByID(db *gorm.DB, model interface{}, id string, deletedField string) error {
	if err := db.Model(model).Where("id = ?", id).Update(deletedField, true).Error; err != nil {
		return fmt.Errorf("删除记录失败: %w", err)
	}
	return nil
}
