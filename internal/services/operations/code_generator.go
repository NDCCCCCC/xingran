package operations

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CodeType 编号类型
type CodeType string

const (
	CodeTypeBuilding    CodeType = "BLD" // 楼宇
	CodeTypeFloor       CodeType = "FLR" // 楼层
	CodeTypeWorkstation CodeType = "WRK" // 工位
	CodeTypeServerRoom  CodeType = "ROM" // 机房
	CodeTypeRoomDevice  CodeType = "DEV" // 机房设备
)

// CodeGenerator 编号生成器
type CodeGenerator struct {
	db *gorm.DB
}

// NewCodeGenerator 创建编号生成器
func NewCodeGenerator(db *gorm.DB) *CodeGenerator {
	return &CodeGenerator{db: db}
}

// GenerateCode 生成编号
// 格式: {类型前缀}-{年月}-{序号}
// 例如: BLD-202501-001
func (g *CodeGenerator) GenerateCode(ctx context.Context, codeType CodeType, tableName string, columnName string) (string, error) {
	// 获取当前年月
	yearMonth := time.Now().Format("200601")

	// 构建编号前缀
	prefix := fmt.Sprintf("%s-%s-", codeType, yearMonth)

	// 查询当前月份最大的序号
	var maxCode string
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s LIKE $1 ORDER BY %s DESC LIMIT 1", columnName, tableName, columnName, columnName)
	err := g.db.WithContext(ctx).Raw(query, prefix+"%").Scan(&maxCode).Error

	var nextSerial int
	if err != nil {
		// 如果没有找到记录，从1开始
		if err == gorm.ErrRecordNotFound {
			nextSerial = 1
		} else {
			return "", fmt.Errorf("查询最大编号失败: %w", err)
		}
	} else {
		// 解析现有编号获取序号
		// 编号格式: BLD-202501-001
		if len(maxCode) >= len(prefix)+3 {
			serialStr := maxCode[len(prefix):]
			var serial int
			if _, err := fmt.Sscanf(serialStr, "%d", &serial); err == nil {
				nextSerial = serial + 1
			} else {
				nextSerial = 1
			}
		} else {
			nextSerial = 1
		}
	}

	// 生成新编号: 序号部分固定3位，不足补0
	code := fmt.Sprintf("%s%03d", prefix, nextSerial)

	return code, nil
}

// GenerateCodeWithCustomPrefix 使用自定义前缀生成编号
// 用于特殊场景，如按楼宇生成楼层编号
func (g *CodeGenerator) GenerateCodeWithCustomPrefix(ctx context.Context, prefix string, tableName string, columnName string) (string, error) {
	// 查询当前前缀下最大的序号
	var maxCode string
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s LIKE $1 ORDER BY %s DESC LIMIT 1", columnName, tableName, columnName, columnName)
	err := g.db.WithContext(ctx).Raw(query, prefix+"-%").Scan(&maxCode).Error

	var nextSerial int
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			nextSerial = 1
		} else {
			return "", fmt.Errorf("查询最大编号失败: %w", err)
		}
	} else {
		// 编号格式: {前缀}-{序号}
		parts := fmt.Sprintf("%s-", prefix)
		if len(maxCode) > len(parts) {
			serialStr := maxCode[len(parts):]
			var serial int
			if _, err := fmt.Sscanf(serialStr, "%d", &serial); err == nil {
				nextSerial = serial + 1
			} else {
				nextSerial = 1
			}
		} else {
			nextSerial = 1
		}
	}

	code := fmt.Sprintf("%s-%03d", prefix, nextSerial)
	return code, nil
}
