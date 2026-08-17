package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserPreference 用户个人设置模型（扩展版）
//
// 历史: 该表由归档 SQL(archive/legacy-2026-06-15/004 建表 + 005 扩展主题/布局列 +
// 044 自定义颜色列)创建,从未进入 AutoMigrate — PG 库因历史遗留已存在而无感,
// 全新 SQLite 库缺表导致登录后首屏 GET /system/settings/preferences 500
// (quick-260817-hfl 冒烟发现)。
//
// 2026-08-17: 模型从 internal/services/system/settings_service.go 迁移至此
// (该包经 type alias 保持兼容,scripts/dbprovision 引用不受影响),
// 成为 schema 单一事实源;sqlite 分支经 AutoMigrate 注册建表(PG 存量表不注册,
// 避免 GORM 对老列发起 DROP NOT NULL / 默认值改写等漂移 ALTER)。
type UserPreference struct {
	ID     string `gorm:"primaryKey;type:uuid" json:"id"`
	UserID string `gorm:"type:uuid;not null;uniqueIndex" json:"userId"`

	// 主题配置
	Theme      string `gorm:"size:20;default:light" json:"theme"`
	ThemeStyle string `gorm:"size:20;default:minimal" json:"themeStyle"`

	// 布局配置
	LayoutType            string `gorm:"size:20;default:classic" json:"layoutType"`
	LayoutDensity         string `gorm:"size:20;default:comfortable" json:"layoutDensity"`
	SidebarWidth          int    `gorm:"default:280" json:"sidebarWidth"`
	SidebarCollapsedWidth int    `gorm:"default:64" json:"sidebarCollapsedWidth"`
	SidebarCollapsed      bool   `gorm:"default:false" json:"sidebarCollapsed"`

	// 数据配置
	PageSize int `gorm:"default:10" json:"pageSize"`

	// 自定义颜色（JSON 格式存储）
	CustomPrimaryColor string `gorm:"size:20" json:"customPrimaryColor,omitempty"`
	CustomSidebarColor string `gorm:"size:20" json:"customSidebarColor,omitempty"`

	// 语言
	Language string `gorm:"size:10;default:zh-CN" json:"language"`
}

// TableName 指定表名
func (UserPreference) TableName() string {
	return "sys_user_preference"
}

// BeforeCreate GORM hook - 创建前生成UUID
func (up *UserPreference) BeforeCreate(tx *gorm.DB) error {
	if up.ID == "" {
		up.ID = uuid.New().String()
	}
	return nil
}
