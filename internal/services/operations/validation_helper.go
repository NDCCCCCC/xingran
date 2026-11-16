package operations

import (
	"context"

	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// Validator 验证器接口
type Validator interface {
	ValidateFloor(ctx context.Context, floorID string) error
	ValidateWall(ctx context.Context, wallID string) error
	ValidateDoor(ctx context.Context, doorID string) error
}

type validator struct {
	db *gorm.DB
}

// NewValidator 创建验证器
func NewValidator(db *gorm.DB) Validator {
	return &validator{db: db}
}

func (v *validator) ValidateFloor(ctx context.Context, floorID string) error {
	// 2026-11-16：根包 models.Floor 已随 operations.go 收敛删除（handoff R6），
	// ops_floors 表的唯一 GORM 映射为子包 OpsFloor。
	return v.validateExists(ctx, &operationsmodels.OpsFloor{}, floorID, apperrors.FloorNotFound())
}

func (v *validator) ValidateWall(ctx context.Context, wallID string) error {
	return v.validateExists(ctx, &operationsmodels.Wall{}, wallID, apperrors.WallNotFound())
}

func (v *validator) ValidateDoor(ctx context.Context, doorID string) error {
	return v.validateExists(ctx, &operationsmodels.Door{}, doorID, apperrors.DoorNotFound())
}

// validateExists 验证记录是否存在
func (v *validator) validateExists(ctx context.Context, model interface{}, id string, notFoundError *apperrors.AppError) error {
	var count int64
	err := v.db.WithContext(ctx).
		Model(model).
		Where("id = ?", id).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return notFoundError
	}
	return nil
}
