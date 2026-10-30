//go:build !skip_db_tests
// +build !skip_db_tests

package operations

import (
	"context"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupValidatorTestDB(t *testing.T) *gorm.DB {
	// 使用显式配置确保使用 modernc.org/sqlite
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file::memory:?cache=shared",
	}, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect test database: %v", err)
	}

	// 自动迁移测试表
	if err := db.AutoMigrate(&models.Floor{}, &operationsmodels.Wall{}, &operationsmodels.Door{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func TestValidator_ValidateFloor(t *testing.T) {
	db := setupValidatorTestDB(t)
	validator := NewValidator(db)
	ctx := context.Background()

	// 创建测试楼层
	name := "Test Floor"
	floor := &models.Floor{Name: name, FloorNo: "1", BuildingID: "test-building-1"}
	if err := db.Create(floor).Error; err != nil {
		t.Fatalf("failed to create test floor: %v", err)
	}

	tests := []struct {
		name          string
		floorID       string
		wantErrorCode apperrors.ErrorCode
	}{
		{
			name:          "存在的楼层",
			floorID:       floor.ID,
			wantErrorCode: 0, // 无错误
		},
		{
			name:          "不存在的楼层",
			floorID:       "non-existent-floor",
			wantErrorCode: apperrors.CodeFloorNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFloor(ctx, tt.floorID)
			gotCode := apperrors.GetErrorCode(err)
			if gotCode != tt.wantErrorCode {
				t.Errorf("ValidateFloor() error code = %v, wantErrorCode %v", gotCode, tt.wantErrorCode)
			}
		})
	}
}

func TestValidator_ValidateWall(t *testing.T) {
	db := setupValidatorTestDB(t)
	validator := NewValidator(db)
	ctx := context.Background()

	// 创建测试楼层和墙体
	floor := &models.Floor{Name: "Test Floor", FloorNo: "1", BuildingID: "test-building-1"}
	if err := db.Create(floor).Error; err != nil {
		t.Fatalf("failed to create test floor: %v", err)
	}

	wallName := "Test Wall"
	wall := &operationsmodels.Wall{
		FloorID:   floor.ID,
		Type:      operationsmodels.WallTypeStraight,
		Points:    `[{"x":100,"y":100},{"x":200,"y":200}]`,
		Name:      &wallName,
		Thickness: 10,
	}
	if err := db.Create(wall).Error; err != nil {
		t.Fatalf("failed to create test wall: %v", err)
	}

	tests := []struct {
		name          string
		wallID        string
		wantErrorCode apperrors.ErrorCode
	}{
		{
			name:          "存在的墙体",
			wallID:        wall.ID,
			wantErrorCode: 0, // 无错误
		},
		{
			name:          "不存在的墙体",
			wallID:        "non-existent-wall",
			wantErrorCode: apperrors.CodeWallNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateWall(ctx, tt.wallID)
			gotCode := apperrors.GetErrorCode(err)
			if gotCode != tt.wantErrorCode {
				t.Errorf("ValidateWall() error code = %v, wantErrorCode %v", gotCode, tt.wantErrorCode)
			}
		})
	}
}

func TestValidator_ValidateDoor(t *testing.T) {
	db := setupValidatorTestDB(t)
	validator := NewValidator(db)
	ctx := context.Background()

	// 创建测试楼层和门
	floor := &models.Floor{Name: "Test Floor", FloorNo: "1", BuildingID: "test-building-1"}
	if err := db.Create(floor).Error; err != nil {
		t.Fatalf("failed to create test floor: %v", err)
	}

	doorName := "Test Door"
	door := &operationsmodels.Door{
		FloorID: floor.ID,
		Type:    operationsmodels.DoorTypeSingle,
		Name:    &doorName,
	}
	if err := db.Create(door).Error; err != nil {
		t.Fatalf("failed to create test door: %v", err)
	}

	tests := []struct {
		name          string
		doorID        string
		wantErrorCode apperrors.ErrorCode
	}{
		{
			name:          "存在的门",
			doorID:        door.ID,
			wantErrorCode: 0, // 无错误
		},
		{
			name:          "不存在的门",
			doorID:        "non-existent-door",
			wantErrorCode: apperrors.CodeDoorNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateDoor(ctx, tt.doorID)
			gotCode := apperrors.GetErrorCode(err)
			if gotCode != tt.wantErrorCode {
				t.Errorf("ValidateDoor() error code = %v, wantErrorCode %v", gotCode, tt.wantErrorCode)
			}
		})
	}
}
