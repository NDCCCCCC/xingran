package system

import (
	"context"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	systemRequests "github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	"gorm.io/gorm"
)

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
		// 删除旧配置
		if err := tx.Where("user_id = ? AND page_key = ?", userID, req.PageKey).
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

// defaultAssetColumns 资产列表默认配置（43 列）
func defaultAssetColumns() []ColumnConfigItem {
	return []ColumnConfigItem{
		{Key: "devicesn", Visible: true, Width: 150},                    // 设备序列号
		{Key: "sequenceNo", Visible: false, Width: 120},                // 序列号
		{Key: "fixAssetNo", Visible: true, Width: 120},                // 固定资产编号
		{Key: "deviceModelName", Visible: true, Width: 120},           // 设备型号
		{Key: "deviceTypeName", Visible: true, Width: 100},            // 设备类型
		{Key: "deviceCategorySecondName", Visible: false, Width: 120}, // 设备中类
		{Key: "deviceBasicTypeName", Visible: true, Width: 100},        // 是否固定资产
		{Key: "deviceStatusName", Visible: true, Width: 100},           // 设备状态
		{Key: "deptName", Visible: true, Width: 120},                   // 所属部门
		{Key: "recipientName", Visible: true, Width: 100},               // 领取人
		{Key: "recipientPhone", Visible: false, Width: 120},             // 领取人电话
		{Key: "recipientEmail", Visible: false, Width: 150},             // 领取人邮箱
		{Key: "assetIp", Visible: true, Width: 120},                    // 资产IP地址
		{Key: "assetMac", Visible: true, Width: 140},                   // 资产MAC地址
		{Key: "assetHostname", Visible: false, Width: 120},              // 主机名
		{Key: "osName", Visible: false, Width: 100},                     // 操作系统
		{Key: "osVersion", Visible: false, Width: 100},                   // 系统版本
		{Key: "cpuModel", Visible: false, Width: 120},                   // CPU型号
		{Key: "cpuCores", Visible: false, Width: 80},                     // CPU核心数
		{Key: "memoryCapacity", Visible: false, Width: 100},            // 内存容量
		{Key: "diskCapacity", Visible: false, Width: 100},              // 硬盘容量
		{Key: "graphicsCard", Visible: false, Width: 120},               // 显卡型号
		{Key: "purchaseDate", Visible: false, Width: 120},               // 购置日期
		{Key: "warrantyExpireDate", Visible: false, Width: 120},         // 保修到期日
		{Key: "supplierName", Visible: false, Width: 120},               // 供应商
		{Key: "manufacturerName", Visible: false, Width: 120},             // 制造商
		{Key: "assetLocation", Visible: true, Width: 120},                // 资产位置
		{Key: "buildingName", Visible: true, Width: 100},                // 所在大楼
		{Key: "floorName", Visible: true, Width: 80},                    // 所在楼层
		{Key: "workstationName", Visible: false, Width: 120},            // 所在工位
		{Key: "serverRoomName", Visible: false, Width: 120},             // 所在机房
		{Key: "networkSwitchName", Visible: false, Width: 120},          // 网络交换机
		{Key: "switchPort", Visible: false, Width: 120},                 // 交换机端口
		{Key: "voltage", Visible: false, Width: 80},                     // 电压
		{Key: "useStatusName", Visible: true, Width: 100},               // 使用状态
		{Key: "financeCode", Visible: false, Width: 100},                // 财务编码
		{Key: "projectCode", Visible: false, Width: 100},                // 项目编码
		{Key: "assetCost", Visible: false, Width: 100},                  // 资产成本
		{Key: "depreciationPeriod", Visible: false, Width: 100},        // 折旧年限
		{Key: "netBookValue", Visible: false, Width: 100},              // 账面净值
		{Key: "depreciationRate", Visible: false, Width: 100},          // 折旧率
		{Key: "accumulatorDepreciation", Visible: false, Width: 120},  // 累计折旧
		{Key: "assetValue", Visible: false, Width: 100},                 // 资产价值
		{Key: "assetYear", Visible: false, Width: 80},                   // 资产年份
		{Key: "remark", Visible: true, Width: 200},                      // 备注
	}
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
