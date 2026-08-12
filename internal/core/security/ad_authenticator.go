package security

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/pkg/ldaputils"
	"gorm.io/gorm"
)

// ADAuthenticator AD域控认证器
// 使用LDAP绑定验证实现AD域控账号认证
type ADAuthenticator struct {
	db          *gorm.DB
	dbName      string                       // 用于日志
	configID    string                       // 默认AD配置ID
	userSyncer  UserSyncer                   // 用户同步服务（可选，为nil时不同步）
	sm4Cipher   addomain.PasswordCipher      // SM4加密器（用于解密AD管理员密码）
	accountPool addomain.AccountPool         // Phase 36: 账号池（可选，注入后启用多账号故障切换）
}

// NewADAuthenticator 创建AD认证器
func NewADAuthenticator(db *gorm.DB, configID string) *ADAuthenticator {
	return &ADAuthenticator{
		db:       db,
		configID: configID,
	}
}

// SetUserSyncer 设置用户同步服务
func (a *ADAuthenticator) SetUserSyncer(syncer UserSyncer) {
	a.userSyncer = syncer
}

// SetPasswordCipher 设置SM4加密器（用于解密AD管理员密码）
// 使用 addomain.PasswordCipher 接口，与 addomain 包的加解密逻辑保持一致
func (a *ADAuthenticator) SetPasswordCipher(cipher addomain.PasswordCipher) {
	a.sm4Cipher = cipher
}

// SetAccountPool Phase 36: 注入账号池
// 注入后 Authenticate 将使用 FailoverClient 进行管理员绑定，单账号锁定不阻断登录
func (a *ADAuthenticator) SetAccountPool(pool addomain.AccountPool) {
	a.accountPool = pool
}

// Authenticate 实现AD域控认证
// 通过LDAP绑定验证用户凭证，成功后搜索用户详细信息用于同步
func (a *ADAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	// 1. 获取AD配置
	config, err := a.getADConfig(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 拨号LDAP连接（不绑定管理员，用于用户绑定验证）
	address := fmt.Sprintf("%s:%d", config.ServerAddress, config.ServerPort)
	conn, err := a.dialConnection(config, address)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrADConnectionFailed, err)
	}
	defer conn.Close()

	// 3. 使用UPN格式绑定验证用户凭证 (username@domain.com)
	userUPN := fmt.Sprintf("%s@%s", req.Username, config.DomainName)
	if err := conn.Bind(userUPN, req.Password); err != nil {
		// 绑定失败 = 凭证无效，不泄露具体原因
		return nil, ErrInvalidCredentials
	}

	// 4. 用户绑定成功，现在需要搜索用户详细信息
	// 需要使用管理员绑定来搜索（因为用户可能没有搜索权限）
	// 通过账号池 FailoverClient 取一个已 Bind 成功的管理员连接用于搜索。
	// 修复（pre-existing bug）：此前单独 dial 了未绑定的 adminConn，而 bindAdminWithFailover
	// 绑定的是它自己的内部连接并丢弃，导致 searchUserInfo 跑在未绑定连接上 → LDAP "bind must be completed"。
	// 现改为使用 bindAdminWithFailover 返回的已绑定 client 的连接搜索。
	adminClient, err := a.bindAdminWithFailover(ctx, config)
	if err != nil {
		// 管理员绑定失败（账号池未初始化 / 无可用账号 / 全部 bind 失败）
		applogger.Warnf("[ADAuth] 管理员绑定失败（账号池耗尽）: username=%s, error=%v", req.Username, err)
		return &AuthResult{
			User:            nil,
			AuthSource:      "ad",
			ADUserInfo:      &ADUserInfo{Username: req.Username},
			NeedsSync:       true,
			SyncErrorReason: "admin_bind",
		}, nil
	}
	defer adminClient.Close()

	// 5. 搜索用户信息（在已绑定的管理员连接上搜索）
	applogger.Infof("[ADAuth] 开始搜索AD用户信息: username=%s, BaseDN=%s", req.Username, config.BaseDN)
	adUserInfo, err := a.searchUserInfo(adminClient.Conn(), config, req.Username)
	if err != nil {
		// 认证成功但搜索失败，返回基本信息
		applogger.Errorf("[ADAuth] 搜索AD用户信息失败: username=%s, error=%v", req.Username, err)
		return &AuthResult{
			User:            nil,
			AuthSource:      "ad",
			ADUserInfo:      &ADUserInfo{Username: req.Username},
			NeedsSync:       true,
			SyncErrorReason: "user_search",
		}, nil
	}
	applogger.Infof("[ADAuth] 成功获取AD用户信息: username=%s, displayName=%s, email=%s, dn=%s", adUserInfo.Username, adUserInfo.DisplayName, adUserInfo.Email, adUserInfo.UserDN)

	// 6. 如果设置了用户同步服务，自动同步AD用户到sys_user表
	applogger.Infof("[ADAuth] 检查用户同步服务: userSyncer是否为nil = %v", a.userSyncer == nil)
	if a.userSyncer != nil {
		defaultRoleID := a.getDefaultRoleID()
		applogger.Infof("[ADAuth] 开始同步AD用户: username=%s, defaultRoleID=%s", adUserInfo.Username, defaultRoleID)

		syncedUser, err := a.userSyncer.SyncADUser(ctx, adUserInfo, defaultRoleID)
		if err != nil {
			// 同步失败不影响认证成功，但标记需要同步
			applogger.Errorf("[ADAuth] 同步AD用户失败: username=%s, error=%v", adUserInfo.Username, err)
			return &AuthResult{
				User:            nil,
				AuthSource:      "ad",
				ADUserInfo:      adUserInfo,
				NeedsSync:       true,
				SyncErrorReason: "user_sync",
			}, nil
		}

		applogger.Infof("[ADAuth] 同步AD用户成功: userID=%s, username=%s", syncedUser.ID, syncedUser.Username)
		// 同步成功，返回完整的用户信息
		return &AuthResult{
			User: &UserResult{
				ID:       syncedUser.ID,
				Username: syncedUser.Username,
				Nickname: syncedUser.Nickname,
				Email:    syncedUser.Email,
				Phone:    syncedUser.Phone,
				Status:   syncedUser.Status,
				DeptID:   syncedUser.DeptID,
				Roles:    syncedUser.Roles,
			},
			AuthSource: "ad",
			ADUserInfo: adUserInfo,
			NeedsSync:  false, // 已同步
		}, nil
	}

	// 没有同步服务，返回AD用户信息，标记需要同步
	applogger.Warnf("[ADAuth] 用户同步服务未设置，返回 NeedsSync=true")
	return &AuthResult{
		User:            nil,
		AuthSource:      "ad",
		ADUserInfo:      adUserInfo,
		NeedsSync:       true,
		SyncErrorReason: "no_syncer",
	}, nil
}

// Name 返回认证器名称
func (a *ADAuthenticator) Name() string {
	return "ad"
}

// getADConfig 获取AD配置
func (a *ADAuthenticator) getADConfig(ctx context.Context) (*models.ADConfig, error) {
	var config models.ADConfig
	if err := a.db.WithContext(ctx).Where("id = ? AND status = 0", a.configID).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrADConfigNotFound
		}
		return nil, fmt.Errorf("获取AD配置失败: %w", err)
	}
	return &config, nil
}

// dialConnection 根据配置建立LDAP连接
func (a *ADAuthenticator) dialConnection(config *models.ADConfig, address string) (*ldap.Conn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // TODO: 生产环境应配置证书
	}

	switch {
	case config.UseSSL:
		return ldap.DialURL("ldaps://"+address, ldap.DialWithTLSConfig(tlsConfig))
	case config.UseTLS:
		conn, err := ldap.DialURL("ldap://" + address)
		if err != nil {
			return nil, err
		}
		if err := conn.StartTLS(tlsConfig); err != nil {
			conn.Close()
			return nil, err
		}
		return conn, nil
	default:
		return ldap.DialURL("ldap://" + address)
	}
}

// bindAdminWithFailover 用账号池多账号故障切换绑定管理员，返回已 Bind 成功的 *LDAPClient，
// 供调用方（Authenticate 的用户搜索）在同一已绑定连接上执行后续操作。
// 调用方负责 defer client.Close()。
//
// Phase 38: 单管理员 fallback 分支已移除（D-03 不保留双轨），accountPool 必须由
// core 在启动时注入（core.go initAuthFactory）。单管理员账号被 AD 锁定（data 775）
// 不再阻断用户登录——FailoverClient 自动切换池中其他可用账号。
func (a *ADAuthenticator) bindAdminWithFailover(ctx context.Context, config *models.ADConfig) (*addomain.LDAPClient, error) {
	if a.accountPool == nil {
		// Phase 38: 单管理员 fallback 已移除；accountPool 必须由 core 启动时注入
		return nil, fmt.Errorf("AD 账号池未初始化，请联系管理员检查 AD 配置（账号池 Tab）")
	}
	fc := addomain.NewFailoverClient(a.accountPool, config)
	client, _, err := fc.PickFirstConnect(ctx)
	if err != nil {
		return nil, fmt.Errorf("账号池无可用账号: %w", err)
	}
	// PickFirstConnect 内部已 Bind 成功（Connect 调用 tryBindAttempts）；返回供调用方搜索使用
	return client, nil
}

// searchUserInfo 搜索AD用户信息
func (a *ADAuthenticator) searchUserInfo(conn *ldap.Conn, config *models.ADConfig, username string) (*ADUserInfo, error) {
	searchRequest := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		fmt.Sprintf("(sAMAccountName=%s)", ldap.EscapeFilter(username)),
		[]string{"dn", "cn", "displayName", "mail", "telephoneNumber", "mobile", "title", "department"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("查询AD用户失败: %w", err)
	}

	if len(sr.Entries) == 0 {
		return nil, ErrUserNotFound
	}

	entry := sr.Entries[0]

	// 提取用户OU DN
	ouDN := ldaputils.ExtractOUDNFromUserDN(entry.DN)

	return &ADUserInfo{
		UserDN:      entry.DN,
		OUDN:        ouDN,
		Username:    username,
		DisplayName: entry.GetAttributeValue("displayName"),
		Email:       entry.GetAttributeValue("mail"),
		Phone:       entry.GetAttributeValue("telephoneNumber"),
		Mobile:      entry.GetAttributeValue("mobile"),
		Title:       entry.GetAttributeValue("title"),
		Department:  entry.GetAttributeValue("department"),
	}, nil
}

// getDefaultRoleID 获取默认角色ID（从sys_config读取）
func (a *ADAuthenticator) getDefaultRoleID() string {
	var config models.Config
	err := a.db.Where("config_key = ?", "sys.auth.ad.default_role_id").First(&config).Error
	if err != nil {
		return "" // 没有配置默认角色
	}
	return config.ConfigValue
}
