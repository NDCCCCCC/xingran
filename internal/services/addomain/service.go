// Package addomain 提供AD域管理的模块化服务
// 将原本的 ad_domain_service.go (1,280行) 拆分为多个职责单一的模块
package addomain

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
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

	// db 直连句柄（P1-3）：AD → sys_user 状态联动 helper 需要它。
	// 各 sub-service 各自持有 DB，ADDomainService 此前不持有；此处仅供本文件
	// 底部的联动 helper 使用，不参与任何 sub-service 的构造。
	db *gorm.DB

	// enableUserFn / disableUserFn：AD 端"启用/禁用用户"写操作的注入点。
	// 生产路径恒为 nil → 委托 s.User.Enable / s.User.Disable（走 FailoverClient
	// 账号池写 LDAP + 更新 sys_ad_user.is_enabled）。
	// 仅测试注入假实现，用于在无真实域控的环境下驱动 P1-3 的联动分支
	// （AD 写成功 → 停用 sys_user；AD 写失败 → 不联动）。
	enableUserFn  func(ctx context.Context, config *models.ADConfig, userDN string) error
	disableUserFn func(ctx context.Context, config *models.ADConfig, userDN string) error
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
		db:       db,
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
//
// P1-3 决策（刻意不做 sys_user 联动）：AD 端启用**不**自动把 sys_user.status
// 改回 0（不"复活"系统账号）。理由：
//  1. sys_user.status 可能被管理员**人工**停用（离职、安全冻结、审计冻结），
//     AD 端的一次启用不应越过人工决策把本地账号解锁；
//  2. 反向对称性：停用是"AD → 系统"单向收敛的保守策略，恢复系统账号必须走
//     系统用户管理页显式操作（system.UserService.Update），形成可审计的操作留痕；
//  3. 避免 AD 端误操作（批量启用脚本）批量解锁本地被停用的账号。
// 恢复系统账号的正规路径：在「系统管理 → 用户管理」把该账号状态改回"正常"。
func (s *ADDomainService) EnableADUser(ctx context.Context, config *models.ADConfig, userDN string) error {
	if s.enableUserFn != nil {
		return s.enableUserFn(ctx, config, userDN)
	}
	return s.User.Enable(ctx, config, userDN)
}

// DisableADUser 禁用AD用户（兼容旧API）
//
// P1-3 修复：AD 端禁用成功后**联动停用 sys_user**（status 0→1），
// 否则 AD 账号被禁用/离职后，AD 登录仍能通过 auth.go 的本地状态兜底拿到 Token。
//
// 顺序：先写 AD（LDAP DisableUser），成功后再联动停用本地账号。
//   - AD 写失败 → 直接返回错误，**不动** sys_user（AD 侧没禁用就不该停本地，
//     否则 AD 抖动会误杀本地账号）；
//   - AD 写成功 → 联动停用；联动失败只记 ERROR 日志、不阻断（AD 侧禁用已生效，
//     对外语义仍是"禁用成功"，重复调用幂等）。
//
// ⚠ 联动走**绕过 service 层**的直接 DB 更新（见 disableSysUsersForADDNs 的注释），
// 目的正是避免"AD 禁用 → sys_user 停用 → sys_ad_push_queue 入队 disable →
// 排水器再推 AD"的反向同步循环。
func (s *ADDomainService) DisableADUser(ctx context.Context, config *models.ADConfig, userDN string) error {
	if s.disableUserFn != nil {
		if err := s.disableUserFn(ctx, config, userDN); err != nil {
			return err
		}
	} else if err := s.User.Disable(ctx, config, userDN); err != nil {
		return err
	}

	// P1-3：联动停用 sys_user（best-effort）。
	disabled, err := disableSysUsersForADDNs(ctx, s.db, config, []string{userDN})
	if err != nil {
		applogger.Warnf("[AD用户] AD 端禁用已生效，但联动停用 sys_user 失败 [dn=%s]: %v", userDN, err)
		return nil
	}
	if disabled > 0 {
		applogger.Infof("[AD用户] AD 端禁用已生效，联动停用 %d 个 sys_user 账号 [dn=%s]", disabled, userDN)
	}
	return nil
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

// ==================== P1-3：AD → sys_user 状态联动 ====================
//
// 背景（P1-3）：AD 侧禁用/删除此前**只写 AD**，从不联动 sys_user.status，
// 而登录鉴权（internal/api/v1/auth.go:229）在发 Token 前只查 sys_user.Status，
// 于是"AD 已禁用/已离职删除"的账号在系统里照样能登录。
//
// 本节提供两条链路的共用实现：
//   - DisableADUser（service.go）：AD 端手动禁用 → 停用本地账号
//   - cleanupVanishedUsers（shadow_cleanup_service.go）：AD 端对象消失 → 停用本地账号

// sysUserADRefRow sys_user 的 AD 关联投影（只取联动判定需要的列）。
//
// ⚠ 用独立结构体而非 models.User：models.User 虽在迁移 219 后含 AdConfigID 字段
// （internal/models/user.go:41，列名 ad_config_id），但**旧库/测试夹具的 sys_user
// 可能没有该列**（createUserTable 夹具即无），用 models.User 整行 SELECT 会直接
// SQL 报错。此处按"列存在与否"动态选择查询列。
type sysUserADRefRow struct {
	ID string
	// AdDn → ad_dn（sys_user 上的 AD 用户完整 DN）
	AdDn *string
	// AdConfigID → ad_config_id（迁移 219 起存在于 sys_user；列不存在时恒为 nil）
	AdConfigID *string
}

// sysUserHasADConfigColumn 探测 sys_user 是否存在 ad_config_id 列。
//
// 现状：迁移 219（internal/core/db/migrations/migration_219_user_ad_sync_fields.go）
// 已给 sys_user 补 ad_config_id 列并建索引，生产库恒为 true；
// 但旧库/测试夹具可能没有该列，故此处做运行时探测而不是写死。
func sysUserHasADConfigColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.User{}, "ad_config_id")
}

// matchSysUserIDsByADDN 按 AD 端 DN 集合找出对应的 sys_user 行 ID。
//
// 匹配规则（按优先级）：
//  1. **首选**：sys_user.ad_config_id = config.ID **且** ad_dn 归一化后 == DN 归一化
//  2. **兜底**：历史数据 ad_config_id 为空（NULL/空串）时，只按 ad_dn 归一化匹配
//  3. ad_config_id 非空但**不等于** config.ID → 不命中（避免跨配置误伤）
//
// ⚠ 当 sys_user 表**没有** ad_config_id 列时（当前 schema 即如此），
//   规则 1/3 无从执行，退化为"只按 ad_dn 归一化匹配"（即规则 2 的口径）。
//   这是安全的：AD 的完整 DN 自带域路径，同一个 DN 不可能同时归属两个配置，
//   因此纯 DN 精确匹配不会跨配置误伤。
//
// DN 比较一律走 addomain.NormalizeDN（dn_util.go）—— 唯一真实源，禁在此处
// 手写归一化（大小写/空格差异会让 AD 回写的 DN 与本地存量 ad_dn 比不相等）。
//
// 只返回**活行**（deleted_at IS NULL）：软删账号不在登录态考量范围内。
func matchSysUserIDsByADDN(ctx context.Context, db *gorm.DB, config *models.ADConfig, dns []string) ([]string, error) {
	if db == nil || config == nil || len(dns) == 0 {
		return nil, nil
	}

	targets := make(map[string]bool, len(dns))
	for _, dn := range dns {
		if norm := NormalizeDN(dn); norm != "" {
			targets[norm] = true
		}
	}
	if len(targets) == 0 {
		return nil, nil
	}

	hasConfigCol := sysUserHasADConfigColumn(db)
	selectCols := "id, ad_dn"
	if hasConfigCol {
		selectCols = "id, ad_dn, ad_config_id"
	}

	var rows []sysUserADRefRow
	if err := db.WithContext(ctx).
		Table("sys_user").
		Select(selectCols).
		Where("deleted_at IS NULL").
		Where("ad_dn IS NOT NULL AND ad_dn <> ''").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询 AD 关联的 sys_user 失败: %w", err)
	}

	var ids []string
	for i := range rows {
		row := &rows[i]
		if row.AdDn == nil || *row.AdDn == "" {
			continue
		}
		if !targets[NormalizeDN(*row.AdDn)] {
			continue
		}
		if hasConfigCol && row.AdConfigID != nil && *row.AdConfigID != "" && *row.AdConfigID != config.ID {
			// 规则 3：归属别的 AD 配置 → 不动（跨配置误伤防护）
			continue
		}
		ids = append(ids, row.ID)
	}
	return ids, nil
}

// disableSysUsersForADDNs 按 AD 端 DN 集合停用对应的 sys_user 账号，返回实际停用行数。
//
// ⚠⚠ 刻意**绕过 service 层**（不走 system.UserService.Update）：
//
//	system.UserService.Update 在 status 由 0 变 1 时会调用
//	ADPushEnqueuer.EnqueueDisableOnStatusTransition 入队 sys_ad_push_queue，
//	排水器随后会把 disable 动作再推回 AD 端 —— 形成
//	「AD 禁用 → sys_user 停用 → 入队 disable → 排水器再推 AD」的**反向同步循环**。
//	这里直接用 GORM 更新 status，**不入队**，从根上切断该循环：
//	AD 是状态的单向权威源，本地只做收敛。
//
// ⚠ 停用是**软性状态变更**：只改 status（0 正常 → 1 停用），
//   **不改 deleted_at、不删行**（账号行保留，管理员可随时人工恢复）。
//
// ⚠ 只更新当前仍为"启用"的行（status = 0）：已停用的行保持不动，重复调用幂等。
//
// ⚠ 缓存失效（已知残余风险，刻意不处理）：
//	addomain 包内没有 CacheProvider 注入路径（构造函数无 cache 参数），
//	且 internal/services/system 与 internal/core 都 import addomain，
//	addomain 反向 import 它们会形成**循环依赖** —— 故本包内无法安全失效
//	menu:user:* 菜单缓存，且不为本缺陷引入新的跨包依赖。
//	风险可控的依据：登录鉴权链路（internal/api/v1/auth.go:229）**直查 DB**
//	（不走缓存），故 DB 状态变更后账号立即无法登录；受影响的只是已登录
//	会话的菜单/权限缓存过期（一个 TTL 周期内自愈），不影响鉴权。
//	DB 是权威源。
func disableSysUsersForADDNs(ctx context.Context, db *gorm.DB, config *models.ADConfig, dns []string) (int64, error) {
	ids, err := matchSysUserIDsByADDN(ctx, db, config, dns)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}

	// 绕过 service 层的直接 DB 更新（不入 sys_ad_push_queue，见上方注释）
	res := db.WithContext(ctx).
		Model(&models.User{}).
		Where("id IN ? AND status = ?", ids, models.UserStatusEnabled).
		Update("status", models.UserStatusDisabled)
	if res.Error != nil {
		return 0, fmt.Errorf("停用 sys_user 账号失败: %w", res.Error)
	}
	return res.RowsAffected, nil
}
