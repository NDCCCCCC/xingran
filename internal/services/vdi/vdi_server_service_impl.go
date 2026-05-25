package vdi

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// vdiServerServiceImpl VDI服务器服务实现
type vdiServerServiceImpl struct {
	db *gorm.DB
}

// NewVDIServerService 创建VDI服务器服务
func NewVDIServerService(db *gorm.DB) VDIServerService {
	return &vdiServerServiceImpl{
		db: db,
	}
}

// CreateServer 创建VDI服务器
func (s *vdiServerServiceImpl) CreateServer(ctx context.Context, req *CreateVDIServerRequest) (*VDIServerDTO, error) {
	// 1. 加密密码
	encryptedPwd := encryptVDIPassword(req.Password)

	// 2. 创建服务器记录
	server := &models.VDIServer{
		Name:              req.Name,
		Endpoint:          req.Endpoint,
		Username:          req.Username,
		PasswordEncrypted: encryptedPwd,
		TenantID:          req.TenantID,
		Status:            req.Status,
	}

	if err := s.db.WithContext(ctx).Create(server).Error; err != nil {
		return nil, fmt.Errorf("failed to create VDI server: %w", err)
	}

	return s.toDTO(server), nil
}

// GetServer 获取VDI服务器详情
func (s *vdiServerServiceImpl) GetServer(ctx context.Context, id string) (*VDIServerDTO, error) {
	var server models.VDIServer
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&server).Error; err != nil {
		return nil, fmt.Errorf("VDI server not found: %w", err)
	}

	return s.toDTO(&server), nil
}

// ListServers 获取VDI服务器列表
func (s *vdiServerServiceImpl) ListServers(ctx context.Context, page, pageSize int) (*VDIServerPageResult, error) {
	// 设置默认分页参数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	// 构建查询
	query := s.db.WithContext(ctx).Model(&models.VDIServer{})

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count VDI servers: %w", err)
	}

	// 分页查询
	var servers []models.VDIServer
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("failed to list VDI servers: %w", err)
	}

	// 转换为DTO
	dtos := make([]VDIServerDTO, len(servers))
	for i, server := range servers {
		dtos[i] = *s.toDTO(&server)
	}

	return &VDIServerPageResult{
		List:     dtos,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// UpdateServer 更新VDI服务器
func (s *vdiServerServiceImpl) UpdateServer(ctx context.Context, id string, req *UpdateVDIServerRequest) error {
	var server models.VDIServer
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&server).Error; err != nil {
		return fmt.Errorf("VDI server not found: %w", err)
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Endpoint != nil {
		updates["endpoint"] = *req.Endpoint
	}
	if req.Username != nil {
		updates["username"] = *req.Username
	}
	if req.Password != nil {
		// 加密新密码
		encryptedPwd := encryptVDIPassword(*req.Password)
		updates["password_encrypted"] = encryptedPwd
		// 清除token，因为密码已更改
		updates["auth_token"] = ""
		updates["token_expiry"] = nil
	}
	if req.TenantID != nil {
		updates["tenant_id"] = *req.TenantID
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if err := s.db.WithContext(ctx).Model(&server).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update VDI server: %w", err)
	}

	return nil
}

// DeleteServer 删除VDI服务器
func (s *vdiServerServiceImpl) DeleteServer(ctx context.Context, id string) error {
	// 1. 检查是否有关联的虚拟机
	var vmCount int64
	if err := s.db.WithContext(ctx).Model(&models.VDIVirtualMachine{}).Where("vdi_server_id = ?", id).Count(&vmCount).Error; err != nil {
		return fmt.Errorf("failed to check VMs: %w", err)
	}

	if vmCount > 0 {
		return fmt.Errorf("cannot delete VDI server with %d associated VMs", vmCount)
	}

	// 2. 删除服务器记录（软删除）
	if err := s.db.WithContext(ctx).Where("id = ?", id).Delete(&models.VDIServer{}).Error; err != nil {
		return fmt.Errorf("failed to delete VDI server: %w", err)
	}

	return nil
}

// TestConnection 测试VDI服务器连接
func (s *vdiServerServiceImpl) TestConnection(ctx context.Context, id string) error {
	// 1. 查询服务器配置
	var server models.VDIServer
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&server).Error; err != nil {
		return fmt.Errorf("VDI server not found: %w", err)
	}

	// 2. 创建VDI客户端配置
	cfg := config.VDIServerConfig{
		Endpoint: server.Endpoint,
		Username: server.Username,
		TenantID: server.TenantID,
	}

	// 3. 创建VDI客户端并测试认证
	client := NewVDIClientExtended(s.db, id, cfg)
	token, err := client.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if token == "" {
		return fmt.Errorf("received empty token")
	}

	// 4. 更新服务器token信息
	now := time.Now()
	server.AuthToken = token
	server.TokenExpiry = &now
	if err := s.db.WithContext(ctx).Save(&server).Error; err != nil {
		return fmt.Errorf("failed to update token: %w", err)
	}

	return nil
}

// toDTO 转换为DTO
func (s *vdiServerServiceImpl) toDTO(server *models.VDIServer) *VDIServerDTO {
	return &VDIServerDTO{
		ID:          server.ID,
		Name:        server.Name,
		Endpoint:    server.Endpoint,
		Username:    server.Username,
		TenantID:    server.TenantID,
		Status:      server.Status,
		TokenExpiry: server.TokenExpiry,
		CreatedAt:   server.CreatedAt,
		UpdatedAt:   server.UpdatedAt,
	}
}
