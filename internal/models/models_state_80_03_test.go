package models

// =====================================================================
// Phase 80-03 Task 5: models 状态机/DTO 纯表驱动。
//
// 纪律:status/UAC 位一律引用 models.* 常量;ToDTO 字段映射必逐字段断言;
// 状态机覆盖 status 0/1/2 + 默认值兜底;UAC 位覆盖单开/多开/未设。
// =====================================================================

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// =====================================================================
// ADServiceAccount 状态机(ad_service_account.go:54-82)
// =====================================================================

// TestMst8003_ADServiceAccount_StateMachine 表驱动:三状态 × 四断言。
func TestMst8003_ADServiceAccount_StateMachine(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	past := time.Now().Add(-1 * time.Hour)

	tests := []struct {
		name             string
		status           int
		breakerUntil     *time.Time
		wantAvailable    bool
		wantCircuitBreak bool
		wantDisabled     bool
		wantText         string
	}{
		{
			name:          "可用_状态0_无breaker",
			status:        ADAccountStatusAvailable,
			wantAvailable: true, wantCircuitBreak: false, wantDisabled: false,
			wantText: "可用",
		},
		{
			name:          "可用_状态0_breaker已过期",
			status:        ADAccountStatusAvailable,
			breakerUntil:  &past,
			wantAvailable: true, wantCircuitBreak: false, wantDisabled: false,
			wantText: "可用",
		},
		{
			name:          "停用_状态1",
			status:        ADAccountStatusDisabled,
			wantAvailable: false, wantCircuitBreak: false, wantDisabled: true,
			wantText: "已停用",
		},
		{
			name:          "熔断中_状态2_未到期",
			status:        ADAccountStatusBreaker,
			breakerUntil:  &future,
			wantAvailable: false, wantCircuitBreak: true, wantDisabled: false,
			wantText: "熔断中",
		},
		{
			name:          "熔断中_状态2_nil截止",
			status:        ADAccountStatusBreaker,
			breakerUntil:  nil,
			wantAvailable: false, wantCircuitBreak: false, wantDisabled: false,
			wantText: "熔断中",
		},
		{
			name:          "熔断中_状态2_过期截止",
			status:        ADAccountStatusBreaker,
			breakerUntil:  &past,
			wantAvailable: true, wantCircuitBreak: false, wantDisabled: false,
			wantText: "熔断中",
		},
		{
			name:          "未知状态_9",
			status:        9,
			wantAvailable: true, wantCircuitBreak: false, wantDisabled: false,
			wantText: "未知",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ADServiceAccount{
				Status:              tt.status,
				CircuitBreakerUntil: tt.breakerUntil,
			}
			assert.Equal(t, tt.wantAvailable, a.IsAvailable())
			assert.Equal(t, tt.wantCircuitBreak, a.IsCircuitBroken())
			assert.Equal(t, tt.wantDisabled, a.IsDisabled())
			assert.Equal(t, tt.wantText, a.StatusText())
		})
	}
}

// =====================================================================
// ADUser UAC 位三连(ad_domain.go:250-264)
// =====================================================================

// TestMst8003_ADUser_UAC 三个位运算断言真值表 + 多位组合。
func TestMst8003_ADUser_UAC(t *testing.T) {
	tests := []struct {
		name           string
		uac            int
		wantDisabled   bool
		wantLocked     bool
		wantPwdExpired bool
	}{
		{"全清", 0, false, false, false},
		{"禁用位单独开", ADAccountDisable, true, false, false},
		{"锁定位单独开", ADLockout, false, true, false},
		{"密码过期位单独开", ADPasswordExpired, false, false, true},
		{"禁用+锁定", ADAccountDisable | ADLockout, true, true, false},
		{"禁用+密码过期", ADAccountDisable | ADPasswordExpired, true, false, true},
		{"三位全开", ADAccountDisable | ADLockout | ADPasswordExpired, true, true, true},
		{"无关位开_NormalAccount", ADNormalAccount, false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &ADUser{UserAccountControl: tt.uac}
			assert.Equal(t, tt.wantDisabled, u.IsDisabledByUAC(), "IsDisabledByUAC")
			assert.Equal(t, tt.wantLocked, u.IsLockedByUAC(), "IsLockedByUAC")
			assert.Equal(t, tt.wantPwdExpired, u.IsPasswordExpiredByUAC(), "IsPasswordExpiredByUAC")
		})
	}
}

// =====================================================================
// ConfigBackup 存储位断言(config_backup.go:81+)
// =====================================================================

func TestMst8003_ConfigBackup_StoredIn(t *testing.T) {
	tests := []struct {
		name     string
		storage  StorageType
		wantDB   bool
		wantFile bool
	}{
		{"数据库存储", StorageTypeDatabase, true, false},
		{"文件存储", StorageTypeFile, false, true},
		{"空字符串_两端均假", StorageType(""), false, false},
		{"未知存储_两端均假", StorageType("bogus"), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := &ConfigBackup{StorageType: tt.storage}
			assert.Equal(t, tt.wantDB, cb.IsStoredInDatabase())
			assert.Equal(t, tt.wantFile, cb.IsStoredInFile())
		})
	}
}

// =====================================================================
// CaptchaBackground 方法(captcha_background.go)
// =====================================================================

func TestMst8003_CaptchaBackground_IsEnabled(t *testing.T) {
	assert.True(t, (&CaptchaBackground{Status: CaptchaBgEnabled}).IsEnabled())
	assert.False(t, (&CaptchaBackground{Status: CaptchaBgDisabled}).IsEnabled())
	assert.False(t, (&CaptchaBackground{Status: CaptchaBackgroundStatus(99)}).IsEnabled(),
		"未知 status 应视作非启用")
}

func TestMst8003_CaptchaBackground_GetFileURL(t *testing.T) {
	cb := &CaptchaBackground{FileName: "x.png"}
	assert.Equal(t, "/uploads/captcha/backgrounds/x.png", cb.GetFileURL())

	cb2 := &CaptchaBackground{FileName: ""}
	assert.Equal(t, "/uploads/captcha/backgrounds/", cb2.GetFileURL(),
		"空文件名仍返合法 URL 前缀")
}

func TestMst8003_CaptchaBackground_GetAllowedShapesList(t *testing.T) {
	t.Run("有allowed_shapes_返该列表", func(t *testing.T) {
		cb := &CaptchaBackground{AllowedShapes: StringArray{"circle", "star"}}
		assert.Equal(t, []string{"circle", "star"}, cb.GetAllowedShapesList())
	})

	t.Run("空allowed_shapes_返全部四形状(默认兜底)", func(t *testing.T) {
		cb := &CaptchaBackground{}
		assert.Equal(t, []string{"circle", "square", "star", "heart"}, cb.GetAllowedShapesList(),
			"空 AllowedShapes 兜底为全四形状常量(AllPieceShapes)")
	})

	t.Run("SetAllowedShapesList_写回", func(t *testing.T) {
		cb := &CaptchaBackground{}
		cb.SetAllowedShapesList([]string{"square", "heart"})
		assert.Equal(t, StringArray{"square", "heart"}, cb.AllowedShapes)
	})
}

// TestMst8003_CaptchaBackground_ToDTO 字段映射逐个断言(常见回归点)。
func TestMst8003_CaptchaBackground_ToDTO(t *testing.T) {
	now := time.Now()
	last := now.Add(-1 * time.Hour)
	bg := &CaptchaBackground{
		ID:              "bg-dto-1",
		FileName:        "img.png",
		FilePath:        "/tmp/img.png",
		FileSize:        1024,
		FileWidth:       200,
		FileHeight:      100,
		FileMD5:         "abcdef0123456789",
		PieceShape:      PieceShapeStar,
		DifficultyLevel: DifficultyMedium,
		AllowedShapes:   StringArray{"star"},
		UseCount:        7,
		LastUsedAt:      &last,
		SortOrder:       3,
		Status:          CaptchaBgEnabled,
		Remark:          "测试备注",
		CreatedAt:       now.Add(-2 * time.Hour),
		UpdatedAt:       now,
	}

	dto := bg.ToDTO()
	assert.Equal(t, "bg-dto-1", dto.ID)
	assert.Equal(t, "img.png", dto.FileName)
	assert.Equal(t, "/tmp/img.png", dto.FilePath)
	assert.Equal(t, int64(1024), dto.FileSize)
	assert.Equal(t, 200, dto.FileWidth)
	assert.Equal(t, 100, dto.FileHeight)
	assert.Equal(t, "star", dto.PieceShape, "PieceShape string 化")
	assert.Equal(t, int(DifficultyMedium), dto.DifficultyLevel, "DifficultyLevel int 化")
	assert.Equal(t, []string{"star"}, dto.AllowedShapes)
	assert.Equal(t, int64(7), dto.UseCount, "UseCount → DTO 升 int64")
	assert.Equal(t, &last, dto.LastUsedAt)
	assert.Equal(t, 3, dto.SortOrder)
	assert.Equal(t, int(CaptchaBgEnabled), dto.Status)
	assert.Equal(t, "测试备注", dto.Remark)
	assert.Equal(t, now.Add(-2*time.Hour), dto.CreatedAt)
	assert.Equal(t, now, dto.UpdatedAt)
	assert.Equal(t, "/uploads/captcha/backgrounds/img.png", dto.PreviewURL,
		"PreviewURL 由 GetFileURL 填充")
}

// =====================================================================
// models 全包 ToDTO 转换函数逐个盘点 + 字段映射断言
// =====================================================================

// TestMst8003_ToDTO_FieldMapping 包内 ToDTO 盘点 + 至少一个字段映射断言。
// (执行 grep "^func .*ToDTO" 盘点本包全部 ToDTO/DTO 转换函数;这里对存在的写断言)
func TestMst8003_ToDTO_FieldMapping(t *testing.T) {
	t.Run("CaptchaBackground.ToDTO_覆盖上方", func(t *testing.T) {
		// 字段映射已在上方 TestMst8003_CaptchaBackground_ToDTO 覆盖,这里留作占位说明
		// 包内 ToDTO 目前仅 CaptchaBackground;后续模块新增 ToDTO 在本测试补断言
		assert.NotNil(t, (&CaptchaBackground{}).ToDTO())
	})
}

// =====================================================================
// BackupType / ConfigType / IsSystem 等枚举字符串化烟雾
// =====================================================================

func TestMst8003_BackupTypeStringConstants(t *testing.T) {
	assert.Equal(t, BackupTypeAuto, BackupType("auto"))
	assert.Equal(t, BackupTypeManual, BackupType("manual"))
	assert.True(t, strings.HasPrefix(string(BackupTypeAuto), "auto"))
	assert.True(t, strings.HasPrefix(string(BackupTypeManual), "manual"))
}
