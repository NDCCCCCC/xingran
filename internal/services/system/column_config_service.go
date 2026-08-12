package system

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	systemRequests "github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"gorm.io/gorm"
)

//go:embed asset_columns_schema.json
var assetColumnsSchemaFS embed.FS

// ColumnConfigService 列配置服务接口
type ColumnConfigService interface {
	GetByPageKey(ctx context.Context, userID string, pageKey string) ([]*models.UserColumnConfig, error)
	Save(ctx context.Context, userID string, req *systemRequests.ColumnConfigSaveRequest) error
	Reset(ctx context.Context, userID string, pageKey string) error
	GetDefaultConfig(ctx context.Context, pageKey string) ([]*models.UserColumnConfig, error)
}

type columnConfigService struct {
	db *gorm.DB
}

func NewColumnConfigService(db *gorm.DB) ColumnConfigService {
	return &columnConfigService{db: db}
}

// GetByPageKey 获取用户列配置
func (s *columnConfigService) GetByPageKey(ctx context.Context, userID string, pageKey string) ([]*models.UserColumnConfig, error) {
	if userID == "" {
		return nil, fmt.Errorf("未登录用户")
	}

	var configs []*models.UserColumnConfig
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND page_key = ?", userID, pageKey).
		Order("display_order ASC").
		Find(&configs).Error

	if err != nil {
		return nil, fmt.Errorf("查询列配置失败: %w", err)
	}

	// 如果没有配置，返回默认配置
	if len(configs) == 0 {
		return s.GetDefaultConfig(ctx, pageKey)
	}

	return configs, nil
}

// Save 保存列配置
func (s *columnConfigService) Save(ctx context.Context, userID string, req *systemRequests.ColumnConfigSaveRequest) error {
	if userID == "" {
		return fmt.Errorf("未登录用户")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 永久删除旧配置（包括软删除的记录，避免唯一约束冲突）
		if err := tx.Where("user_id = ? AND page_key = ?", userID, req.PageKey).
			Unscoped().
			Delete(&models.UserColumnConfig{}).Error; err != nil {
			return fmt.Errorf("删除旧配置失败: %w", err)
		}

		// 批量插入新配置
		for order, col := range req.ColumnConfigs {
			config := models.UserColumnConfig{
				UserID:       userID,
				PageKey:      req.PageKey,
				ColumnKey:    col.ColumnKey,
				Visible:      col.Visible,
				DisplayOrder: order + 1,
				Width:        col.Width,
			}
			if err := tx.Create(&config).Error; err != nil {
				return fmt.Errorf("插入列配置失败: %w", err)
			}
		}

		return nil
	})
}

// Reset 重置列配置
func (s *columnConfigService) Reset(ctx context.Context, userID string, pageKey string) error {
	if userID == "" {
		return fmt.Errorf("未登录用户")
	}

	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND page_key = ?", userID, pageKey).
		Delete(&models.UserColumnConfig{}).Error; err != nil {
		return fmt.Errorf("重置列配置失败: %w", err)
	}

	return nil
}

// GetDefaultConfig 获取默认配置（资产列表 43 列）
func (s *columnConfigService) GetDefaultConfig(ctx context.Context, pageKey string) ([]*models.UserColumnConfig, error) {
	defaultColumns := getDefaultColumnsForPage(pageKey)
	configs := make([]*models.UserColumnConfig, len(defaultColumns))

	for i, col := range defaultColumns {
		configs[i] = &models.UserColumnConfig{
			ColumnKey:    col.Key,
			Visible:      col.Visible,
			DisplayOrder: i + 1,
			Width:        col.Width,
		}
	}

	return configs, nil
}

// getDefaultColumnsForPage 根据页面键返回默认列配置
func getDefaultColumnsForPage(pageKey string) []ColumnConfigItem {
	switch pageKey {
	case "asset.list":
		return defaultAssetColumns()
	case "user.list":
		return defaultUserColumns()
	case "role.list":
		return defaultRoleColumns()
	case "dept.list":
		return defaultDeptColumns()
	default:
		return []ColumnConfigItem{}
	}
}

// ColumnConfigItem 列配置项
type ColumnConfigItem struct {
	Key     string
	Visible bool
	Width   int
}

// defaultAssetColumns 资产列表默认配置 — 数据来源 asset_columns_schema.json (go:embed)
// 真实源: xingran-react-frontend/src/pages/operations/assets/columnsSchema.ts
// 同步机制: 前端 npm run sync-columns-schema (prebuild 钩子) 序列化 → JSON → 后端 embed
func defaultAssetColumns() []ColumnConfigItem {
	return loadAssetColumnsFromEmbed()
}

// loadAssetColumnsFromEmbed 从 embed JSON 读取资产列表默认列配置。
// JSON 由前端 sync-columns-schema.mjs 维护,确保前后端单一真理源。
// 注:此函数无 ctx 参数;embed 在进程启动时已加载到内存。
func loadAssetColumnsFromEmbed() []ColumnConfigItem {
	data, err := assetColumnsSchemaFS.ReadFile("asset_columns_schema.json")
	if err != nil {
		// embed 失败 = 构建/部署错误,panic 暴露问题优于静默回退
		panic(fmt.Sprintf("asset_columns_schema.json embed read failed: %v", err))
	}
	var schema struct {
		Columns []ColumnConfigItem `json:"columns"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		panic(fmt.Sprintf("asset_columns_schema.json parse failed: %v", err))
	}
	return schema.Columns
}

// defaultUserColumns 用户列表默认配置（12 列）
func defaultUserColumns() []ColumnConfigItem {
	return []ColumnConfigItem{
		{Key: "username", Visible: true, Width: 120},
		{Key: "nickname", Visible: true, Width: 120},
		{Key: "deptName", Visible: true, Width: 120},
		{Key: "email", Visible: true, Width: 150},
		{Key: "phone", Visible: true, Width: 120},
		{Key: "status", Visible: true, Width: 80},
		{Key: "createTime", Visible: true, Width: 150},
		{Key: "updateTime", Visible: false, Width: 150},
		{Key: "lastLoginTime", Visible: false, Width: 150},
		{Key: "loginCount", Visible: false, Width: 100},
		{Key: "roleNames", Visible: true, Width: 150},
		{Key: "remark", Visible: false, Width: 200},
	}
}

// defaultRoleColumns 角色列表默认配置（8 列）
func defaultRoleColumns() []ColumnConfigItem {
	return []ColumnConfigItem{
		{Key: "roleName", Visible: true, Width: 120},
		{Key: "roleKey", Visible: true, Width: 120},
		{Key: "roleSort", Visible: true, Width: 80},
		{Key: "status", Visible: true, Width: 80},
		{Key: "createTime", Visible: true, Width: 150},
		{Key: "updateTime", Visible: false, Width: 150},
		{Key: "remark", Visible: false, Width: 200},
		{Key: "permissionCount", Visible: false, Width: 100},
	}
}

// defaultDeptColumns 部门列表默认配置（10 列）
func defaultDeptColumns() []ColumnConfigItem {
	return []ColumnConfigItem{
		{Key: "deptName", Visible: true, Width: 120},
		{Key: "parentName", Visible: true, Width: 120},
		{Key: "leader", Visible: true, Width: 100},
		{Key: "phone", Visible: true, Width: 120},
		{Key: "email", Visible: false, Width: 150},
		{Key: "status", Visible: true, Width: 80},
		{Key: "sort", Visible: true, Width: 80},
		{Key: "createTime", Visible: true, Width: 150},
		{Key: "updateTime", Visible: false, Width: 150},
		{Key: "remark", Visible: false, Width: 200},
	}
}
