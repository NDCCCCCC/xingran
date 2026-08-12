// Package addomain 提供AD域管理的模块化服务
// 将原本的 ad_domain_service.go (1,280行) 拆分为多个职责单一的模块
package addomain

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// ==================== 向后兼容类型 ====================
// 这些类型保持与旧服务的API兼容

// ADConfigListRequest AD配置列表请求
type ADConfigListRequest struct {
	base.BaseListRequest
	Status *int `json:"status,omitempty"`
}

// ADConfigCreateRequest 创建AD配置请求
type ADConfigCreateRequest struct {
	ConfigName    string `json:"configName" binding:"required,max=100"`
	ServerAddress string `json:"serverAddress" binding:"required,max=255"`
	ServerPort    int    `json:"serverPort" binding:"required,min=1,max=65535"`
	DomainName    string `json:"domainName" binding:"required,max=255"`
	BaseDN        string `json:"baseDn" binding:"required,max=500"`
	UseSSL        bool   `json:"useSsl"`
	UseTLS        bool   `json:"useTls"`
	SyncEnabled   bool   `json:"syncEnabled"`
	SyncInterval  int    `json:"syncInterval" binding:"min=60"`
	MemberOUDN    string `json:"memberOuDn,omitempty"` // 本部部门分组OU DN
}

// ADConfigUpdateRequest 更新AD配置请求
type ADConfigUpdateRequest struct {
	ID            string `json:"-"`
	ConfigName    string `json:"configName" binding:"required,max=100"`
	ServerAddress string `json:"serverAddress" binding:"required,max=255"`
	ServerPort    int    `json:"serverPort" binding:"required,min=1,max=65535"`
	DomainName    string `json:"domainName" binding:"required,max=255"`
	BaseDN        string `json:"baseDn" binding:"required,max=500"`
	UseSSL        bool   `json:"useSsl"`
	UseTLS        bool   `json:"useTls"`
	SyncEnabled   bool   `json:"syncEnabled"`
	SyncInterval  int    `json:"syncInterval" binding:"min=60"`
	MemberOUDN    string `json:"memberOuDn,omitempty"` // 本部部门分组OU DN
	Status        *int   `json:"status,omitempty"`
}

// ADUserListRequest AD用户列表请求
type ADUserListRequest struct {
	base.BaseListRequest
	ConfigID  string  `json:"configId" binding:"required"`
	OUDN      *string `json:"ouDn,omitempty"`
	Username  *string `json:"username,omitempty"`
	IsEnabled *bool   `json:"isEnabled,omitempty"`
}

// ADUserUpdateRequest AD用户更新请求
type ADUserUpdateRequest struct {
	DisplayName *string `json:"displayName,omitempty"`
	Email       *string `json:"email,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Mobile      *string `json:"mobile,omitempty"`
	Title       *string `json:"title,omitempty"`
	Department  *string `json:"department,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ADGroupListRequest AD用户组列表请求
type ADGroupListRequest struct {
	base.BaseListRequest
	ConfigID  string  `json:"configId" binding:"required"`
	OUDN      *string `json:"ouDn,omitempty"`
	GroupName *string `json:"groupName,omitempty"`
}

// ADSyncResult AD同步结果
type ADSyncResult = SyncResult

// ADOUNode AD OU节点
type ADOUNode struct {
	ID       string     `json:"id"`
	OUDN     string     `json:"ouDn"`
	OUName   string     `json:"ouName"`
	Children []ADOUNode `json:"children,omitempty"`
}

// ADSyncRequest AD同步请求
type ADSyncRequest struct {
	SyncType string `json:"syncType"` // full/incremental
}

// ADDomainService AD域管理服务（主服务）
// 提供统一的访问入口，内部委托给各个专门的服务
type ADDomainService struct {
	Config    *ConfigService
	Sync      *SyncService
	OU        *OUService
	User      *UserService
	Group     *GroupService
	GroupSync *GroupSyncService
	// OU-组映射替代了部门-组映射
	OUGroupMapping *OUGroupMappingService
	GroupMgmt      GroupManagementService
	// MemberSync 依赖部门-组映射，已移除
	Log      *LogService
	Computer *ComputerService
}

// NewADDomainService creates the AD domain service.
// cipher is optional - if provided, it sets the global password cipher for the addomain package.
//
// Phase 38 Wave 1: 新增 pool 参数（AccountPool 实例）。pool 会透传到所有需要走账号池
// FailoverClient 的 sub-service（Config/Sync/User/Group/GroupSync/GroupMgmt），
// 38-02 已在 sub-service 内基于 s.pool 改造为 FailoverClient 闭包（单管理员直连已下线）。
// 调用方必须保证传入同一 AccountPool 实例（Pitfall 4：避免重复 New 导致缓存不共享）。
func NewADDomainService(db *gorm.DB, pool AccountPool, cipher ...PasswordCipher) *ADDomainService {
	if len(cipher) > 0 && cipher[0] != nil {
		SetADSM4Cipher(cipher[0])
	}
	return &ADDomainService{
		Config:         NewConfigService(db, pool),
		Sync:           NewSyncService(db, pool),
		OU:             NewOUService(db),
		User:           NewUserService(db, pool),
		Group:          NewGroupService(db, pool),
		GroupSync:      NewGroupSyncService(db, pool),
		OUGroupMapping: NewOUGroupMappingService(db),
		GroupMgmt:      NewGroupManagementService(db, pool),
		// MemberSync 已移除 - 依赖部门-组映射
		Log:      NewLogService(db),
		Computer: NewComputerService(db),
	}
}

// ==================== 向后兼容方法 ====================
// 这些方法保持与旧服务的API兼容，内部委托给新的模块化服务

// GetADConfigList 获取AD配置列表（兼容旧API）
// orderByColumn/isAsc 为服务端排序参数(可选,透传给底层 ApplySort 白名单)。
func (s *ADDomainService) GetADConfigList(ctx context.Context, status *int, current, pageSize int, orderByColumn string, isAsc *bool) ([]models.ADConfig, int64, error) {
	return s.Config.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{
			Current:       current,
			PageSize:      pageSize,
			OrderByColumn: orderByColumn,
			IsAsc:         isAsc,
		},
		Status: status,
	})
}

// GetADConfigByID 根据ID获取AD配置（兼容旧API）
func (s *ADDomainService) GetADConfigByID(ctx context.Context, id string) (*models.ADConfig, error) {
	return s.Config.GetByID(ctx, id)
}

// CreateADConfig 创建AD配置（兼容旧API）
func (s *ADDomainService) CreateADConfig(ctx context.Context, req *ADConfigCreateRequest, creatorID string) (*models.ADConfig, error) {
	internalReq := &CreateRequest{
		ConfigName:    req.ConfigName,
		ServerAddress: req.ServerAddress,
		ServerPort:    req.ServerPort,
		DomainName:    req.DomainName,
		BaseDN:        req.BaseDN,
		UseSSL:        req.UseSSL,
		UseTLS:        req.UseTLS,
		SyncEnabled:   req.SyncEnabled,
		SyncInterval:  req.SyncInterval,
		MemberOUDN:    req.MemberOUDN,
	}
	return s.Config.Create(ctx, internalReq, creatorID)
}

// UpdateADConfig 更新AD配置（兼容旧API）
func (s *ADDomainService) UpdateADConfig(ctx context.Context, req *ADConfigUpdateRequest, updaterID string) error {
	internalReq := &UpdateRequest{
		ID:            req.ID,
		ConfigName:    req.ConfigName,
		ServerAddress: req.ServerAddress,
		ServerPort:    req.ServerPort,
		DomainName:    req.DomainName,
		BaseDN:        req.BaseDN,
		UseSSL:        req.UseSSL,
		UseTLS:        req.UseTLS,
		SyncEnabled:   req.SyncEnabled,
		SyncInterval:  req.SyncInterval,
		MemberOUDN:    req.MemberOUDN,
		Status:        req.Status,
	}
	return s.Config.Update(ctx, internalReq, updaterID)
}

// DeleteADConfig 删除AD配置（兼容旧API）
func (s *ADDomainService) DeleteADConfig(ctx context.Context, id string) error {
	return s.Config.Delete(ctx, id)
}

// TestADConnection 测试AD连接（兼容旧API）
func (s *ADDomainService) TestADConnection(ctx context.Context, configID string) error {
	config, err := s.Config.GetByID(ctx, configID)
	if err != nil {
		return err
	}
	return s.Config.TestConnection(ctx, config)
}

// SyncADData 同步AD数据（兼容旧API）
func (s *ADDomainService) SyncADData(ctx context.Context, configID string, syncType string) (*SyncResult, error) {
	return s.Sync.SyncDataByID(ctx, configID, syncType)
}

// GetOUTree 获取OU树（兼容旧API）
func (s *ADDomainService) GetOUTree(ctx context.Context, configID string) ([]OUNode, error) {
	return s.OU.GetTree(ctx, configID)
}

// GetADUserList 获取AD用户列表（兼容旧API）
func (s *ADDomainService) GetADUserList(ctx context.Context, req *ADUserListRequest) ([]models.ADUser, int64, error) {
	return s.User.GetList(ctx, &UserListRequest{
		BaseListRequest: req.BaseListRequest,
		ConfigID:        req.ConfigID,
		OUDN:            req.OUDN,
		Username:        req.Username,
		IsEnabled:       req.IsEnabled,
	})
}

// GetADUserByDN 根据DN获取用户（兼容旧API）
func (s *ADDomainService) GetADUserByDN(ctx context.Context, configID, userDN string) (*models.ADUser, error) {
	return s.User.GetByDN(ctx, configID, userDN)
}

// GetADUserByID 根据数据库ID获取用户（兼容旧API）
func (s *ADDomainService) GetADUserByID(ctx context.Context, userID string) (*models.ADUser, error) {
	return s.User.GetByID(ctx, userID)
}

// GetADGroupList 获取AD用户组列表（兼容旧API）
func (s *ADDomainService) GetADGroupList(ctx context.Context, req *ADGroupListRequest) ([]models.ADGroup, int64, error) {
	return s.Group.GetList(ctx, &GroupListRequest{
		BaseListRequest: req.BaseListRequest,
		ConfigID:        req.ConfigID,
		OUDN:            req.OUDN,
		GroupName:       req.GroupName,
	})
}

// GetADSyncLogList 获取同步日志列表（兼容旧API）
// orderByColumn/isAsc 为服务端排序参数(可选,透传给底层 ApplySort 白名单)。
func (s *ADDomainService) GetADSyncLogList(ctx context.Context, configID string, current, pageSize int, orderByColumn string, isAsc *bool) ([]models.ADSyncLog, int64, error) {
	return s.Log.GetList(ctx, configID, current, pageSize, orderByColumn, isAsc)
}

// GetADGroupMembers 获取AD用户组成员（兼容旧API）
func (s *ADDomainService) GetADGroupMembers(ctx context.Context, configID, groupDN string, current, pageSize int) ([]models.ADUser, int64, error) {
	return s.Group.GetMembers(ctx, configID, groupDN, current, pageSize)
}

// GetADComputerList 获取AD电脑设备列表（兼容旧API）
func (s *ADDomainService) GetADComputerList(ctx context.Context, req *ComputerListRequest) ([]ComputerDetail, int64, error) {
	return s.Computer.List(ctx, req)
}

// GetADComputerByDN 根据DN获取电脑设备（兼容旧API）
func (s *ADDomainService) GetADComputerByDN(ctx context.Context, configID, computerDN string) (*ComputerDetail, error) {
	return s.Computer.GetByDN(ctx, configID, computerDN)
}

// UpdateADUser 更新AD用户（兼容旧API）
func (s *ADDomainService) UpdateADUser(ctx context.Context, config *models.ADConfig, userDN string, req *ADUserUpdateRequest) error {
	return s.User.Update(ctx, config, userDN, &UserUpdateRequest{
		DisplayName: req.DisplayName,
		Email:       req.Email,
		Phone:       req.Phone,
		Mobile:      req.Mobile,
		Title:       req.Title,
		Department:  req.Department,
		Description: req.Description,
	})
}

// MoveADUser 移动AD用户（兼容旧API）
func (s *ADDomainService) MoveADUser(ctx context.Context, config *models.ADConfig, userDN, newOUDN string) error {
	return s.User.Move(ctx, config, userDN, newOUDN)
}

// EnableADUser 启用AD用户（兼容旧API）
func (s *ADDomainService) EnableADUser(ctx context.Context, config *models.ADConfig, userDN string) error {
	return s.User.Enable(ctx, config, userDN)
}

// DisableADUser 禁用AD用户（兼容旧API）
func (s *ADDomainService) DisableADUser(ctx context.Context, config *models.ADConfig, userDN string) error {
	return s.User.Disable(ctx, config, userDN)
}

// GetADUserIds 获取AD用户ID列表（用于全选功能）
func (s *ADDomainService) GetADUserIds(ctx context.Context, req *ADUserListRequest) ([]string, error) {
	return s.User.GetUserIds(ctx, &UserListRequest{
		BaseListRequest: req.BaseListRequest,
		ConfigID:        req.ConfigID,
		OUDN:            req.OUDN,
		Username:        req.Username,
		IsEnabled:       req.IsEnabled,
	})
}
