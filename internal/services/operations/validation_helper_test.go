// +build !skip_db_tests

package operations

import (
	"context"
	"testing"

	operationsmodels "github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动，必须在使用前导入
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
		name      string
		floorID   string
		wantError error
	}{
		{
			name:      "存在的楼层",
			floorID:   floor.ID,
			wantError: nil,
		},
		{
			name:      "不存在的楼层",
			floorID:   "non-existent-floor",
			wantError: ErrFloorNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFloor(ctx, tt.floorID)
			if err != tt.wantError {
				t.Errorf("ValidateFloor() error = %v, wantError %v", err, tt.wantError)
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
		Type:     operationsmodels.WallTypeStraight,
		Points:    `[{"x":100,"y":100},{"x":200,"y":200}]`,
		Name:     &wallName,
		Thickness: 10,
	}
	if err := db.Create(wall).Error; err != nil {
		t.Fatalf("failed to create test wall: %v", err)
	}

	tests := []struct {
		name      string
		wallID    string
		wantError error
	}{
		{
			name:      "存在的墙体",
			wallID:    wall.ID,
			wantError: nil,
		},
		{
			name:      "不存在的墙体",
			wallID:    "non-existent-wall",
			wantError: ErrWallNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateWall(ctx, tt.wallID)
			if err != tt.wantError {
				t.Errorf("ValidateWall() error = %v, wantError %v", err, tt.wantError)
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
		name      string
		doorID    string
		wantError error
	}{
		{
			name:      "存在的门",
			doorID:    door.ID,
			wantError: nil,
		},
		{
			name:      "不存在的门",
			doorID:    "non-existent-door",
			wantError: ErrDoorNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateDoor(ctx, tt.doorID)
			if err != tt.wantError {
				t.Errorf("ValidateDoor() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
