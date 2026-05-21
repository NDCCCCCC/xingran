package security

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	"gorm.io/gorm"
)

// ADAuthenticator AD域控认证器
// 使用LDAP绑定验证实现AD域控账号认证
type ADAuthenticator struct {
	db      *gorm.DB
	dbName  string // 用于日志
	configID string  // 默认AD配置ID
}

// NewADAuthenticator 创建AD认证器
func NewADAuthenticator(db *gorm.DB, configID string) *ADAuthenticator {
	return &ADAuthenticator{
		db:       db,
		configID: configID,
	}
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
	// 先解密管理员密码
	config.AdminPassword = addomain.DecryptPassword(config.AdminPassword)

	adminConn, err := a.dialConnection(config, address)
	if err != nil {
		// 用户认证成功但无法搜索用户信息
		// 仍然返回成功，但没有AD用户详情
		return &AuthResult{
			User:       nil,
			AuthSource: "ad",
			ADUserInfo: &ADUserInfo{
				Username: req.Username,
			},
			NeedsSync: true,
		}, nil
	}
	defer adminConn.Close()

	// 管理员绑定
	if err := a.bindAdmin(adminConn, config); err != nil {
		// 认证成功但搜索失败，返回基本信息
		return &AuthResult{
			User:       nil,
			AuthSource: "ad",
			ADUserInfo: &ADUserInfo{
				Username: req.Username,
			},
			NeedsSync: true,
		}, nil
	}

	// 5. 搜索用户信息
	adUserInfo, err := a.searchUserInfo(adminConn, config, req.Username)
	if err != nil {
		// 认证成功但搜索失败，返回基本信息
		return &AuthResult{
			User:       nil,
			AuthSource: "ad",
			ADUserInfo: &ADUserInfo{
				Username: req.Username,
			},
			NeedsSync: true,
		}, nil
	}

	return &AuthResult{
		User:       nil, // AD认证不一定有sys_user记录
		AuthSource: "ad",
		ADUserInfo: adUserInfo,
		NeedsSync:  true,
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

// bindAdmin 使用管理员凭证绑定
func (a *ADAuthenticator) bindAdmin(conn *ldap.Conn, config *models.ADConfig) error {
	// 尝试UPN格式绑定
	adminUPN := fmt.Sprintf("%s@%s", config.AdminUsername, config.DomainName)
	if err := conn.Bind(adminUPN, config.AdminPassword); err == nil {
		return nil
	}

	// 尝试直接绑定
	return conn.Bind(config.AdminUsername, config.AdminPassword)
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

	return &ADUserInfo{
		UserDN:      entry.DN,
		Username:    username,
		DisplayName: entry.GetAttributeValue("displayName"),
		Email:       entry.GetAttributeValue("mail"),
		Phone:       entry.GetAttributeValue("telephoneNumber"),
		Mobile:      entry.GetAttributeValue("mobile"),
		Title:       entry.GetAttributeValue("title"),
		Department:  entry.GetAttributeValue("department"),
	}, nil
}
