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

type Validator interface {
	ValidateFloor(ctx context.Context, floorID string) error
	ValidateWall(ctx context.Context, wallID string) error
	ValidateDoor(ctx context.Context, doorID string) error
}

type validator struct {
	db *gorm.DB
}

func NewValidator(db *gorm.DB) Validator {
	return &validator{db: db}
}

func (v *validator) ValidateFloor(ctx context.Context, floorID string) error {
	var count int64
	if err := v.db.WithContext(ctx).
		Model(&models.Floor{}).
		Where("id = ?", floorID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrFloorNotFound
	}
	return nil
}

func (v *validator) ValidateWall(ctx context.Context, wallID string) error {
	var count int64
	if err := v.db.WithContext(ctx).
		Model(&operationsmodels.Wall{}).
		Where("id = ?", wallID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrWallNotFound
	}
	return nil
}

func (v *validator) ValidateDoor(ctx context.Context, doorID string) error {
	var count int64
	if err := v.db.WithContext(ctx).
		Model(&operationsmodels.Door{}).
		Where("id = ?", doorID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrDoorNotFound
	}
	return nil
}
