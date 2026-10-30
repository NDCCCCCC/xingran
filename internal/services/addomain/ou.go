package addomain

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// OUService OU管理服务
type OUService struct {
	db *gorm.DB
}

// NewOUService 创建OU服务
func NewOUService(db *gorm.DB) *OUService {
	return &OUService{db: db}
}

// OUNode OU树节点
type OUNode struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	DN       string   `json:"dn"`
	Path     string   `json:"path,omitempty"`
	Children []OUNode `json:"children,omitempty"`
}

// GetTree 获取OU树形结构
func (s *OUService) GetTree(ctx context.Context, configID string) ([]OUNode, error) {
	// 首先获取配置的 BaseDN
	var config models.ADConfig
	if err := s.db.WithContext(ctx).
		Select("base_dn").
		Where("id = ?", configID).
		First(&config).Error; err != nil {
		return nil, fmt.Errorf("查询AD配置失败: %w", err)
	}

	var ous []models.ADOU
	if err := s.db.WithContext(ctx).
		Where("ad_config_id = ? AND deleted_at IS NULL", configID).
		Find(&ous).Error; err != nil {
		return nil, fmt.Errorf("查询OU失败: %w", err)
	}

	// 使用 BaseDN 作为根节点的父级 DN
	// 因为 LDAP 中 OU 是 BaseDN 的直接子节点
	return s.buildTree(ous, config.BaseDN), nil
}

// buildTree 构建OU树
func (s *OUService) buildTree(ous []models.ADOU, parentDN string) []OUNode {
	var tree []OUNode

	// 构建树
	for _, ou := range ous {
		if ou.ParentDN == parentDN {
			node := OUNode{
				ID:   ou.ID,
				Name: ou.OUName,
				DN:   ou.OUN,
				Path: ou.OUPath,
			}
			node.Children = s.buildTree(ous, ou.OUN)
			tree = append(tree, node)
		}
	}

	return tree
}

// GetByDN 根据DN获取OU
func (s *OUService) GetByDN(ctx context.Context, configID, ouDN string) (*models.ADOU, error) {
	var ou models.ADOU
	err := s.db.WithContext(ctx).
		Where("ad_config_id = ? AND ou_dn = ? AND deleted_at IS NULL", configID, ouDN).
		First(&ou).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("OU不存在")
		}
		return nil, fmt.Errorf("查询OU失败: %w", err)
	}
	return &ou, nil
}
