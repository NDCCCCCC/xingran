package operations

import (
	"context"
	"errors"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	operationsmodels "github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

var (
	ErrFloorNotFound = errors.New("楼层不存在")
	ErrWallNotFound  = errors.New("墙体不存在")
	ErrDoorNotFound  = errors.New("门不存在")
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
	return v.validateExists(ctx, &models.Floor{}, floorID, ErrFloorNotFound)
}

func (v *validator) ValidateWall(ctx context.Context, wallID string) error {
	return v.validateExists(ctx, &operationsmodels.Wall{}, wallID, ErrWallNotFound)
}

func (v *validator) ValidateDoor(ctx context.Context, doorID string) error {
	return v.validateExists(ctx, &operationsmodels.Door{}, doorID, ErrDoorNotFound)
}

// validateExists 验证记录是否存在
func (v *validator) validateExists(ctx context.Context, model interface{}, id string, notFoundError error) error {
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
