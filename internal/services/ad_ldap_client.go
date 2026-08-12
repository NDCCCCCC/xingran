package services

import (
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// adTLSSkipVerify 缓存 AD/LDAP TLS 证书校验开关,从 config.AD.TLSSkipVerify 读取。
// 默认 true(跳过校验),以兼容 AD 域控内网自签证书的部署。
// 生产环境应在 configs/config.yaml 中显式将 ad.tls_skip_verify 设为 false。
var (
	adTLSSkipVerify     bool
	adTLSSkipVerifyOnce sync.Once
)

// loadADTLSSkipVerify 懒加载 AD TLS 跳过校验开关
func loadADTLSSkipVerify() bool {
	adTLSSkipVerifyOnce.Do(func() {
		cfg := config.Load()
		adTLSSkipVerify = cfg.AD.TLSSkipVerify
	})
	return adTLSSkipVerify
}

// LDAPClient LDAP客户端封装
type LDAPClient struct {
	config *models.ADConfig
	conn   *ldap.Conn
}

// NewLDAPClient 创建LDAP客户端
func NewLDAPClient(config *models.ADConfig) *LDAPClient {
	return &LDAPClient{config: config}
}

// Connect 连接AD服务器
func (c *LDAPClient) Connect() error {
	var err error
	address := fmt.Sprintf("%s:%d", c.config.ServerAddress, c.config.ServerPort)

	if c.config.UseSSL {
		// LDAPS (端口636)
		c.conn, err = ldap.DialURL("ldaps://"+address, ldap.DialWithTLSConfig(&tls.Config{
			// F-08 fix: 不再硬编码,改为从 config.AD.TLSSkipVerify 读取
			// 默认 true 保持向后兼容(AD 域控自签证书),生产应在 yaml 中显式设 false
			InsecureSkipVerify: loadADTLSSkipVerify(),
			ServerName:         c.config.ServerAddress,
		}))
	} else {
		// LDAP (端口389)
		c.conn, err = ldap.DialURL("ldap://" + address)
	}

	if err != nil {
		return fmt.Errorf("连接AD服务器失败: %w", err)
	}

	// 设置超时
	c.conn.SetTimeout(time.Second * 30)

	// 启动TLS (如果配置了StartTLS)
	if c.config.UseTLS && !c.config.UseSSL {
		err = c.conn.StartTLS(&tls.Config{
			InsecureSkipVerify: loadADTLSSkipVerify(),
			ServerName:         c.config.ServerAddress,
		})
		if err != nil {
			c.conn.Close()
			return fmt.Errorf("启动TLS失败: %w", err)
		}
	}

	// 绑定管理员账户
	username := c.formatUsername(c.config.AdminUsername)
	err = c.conn.Bind(username, c.config.AdminPassword)
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("绑定管理员账户失败: %w", err)
	}

	return nil
}

// formatUsername 格式化用户名
// 支持两种格式: username@domain.com 或 DOMAIN\username
func (c *LDAPClient) formatUsername(username string) string {
	// 如果已经包含@，直接返回
	if strings.Contains(username, "@") {
		return username
	}
	// 如果已经包含\，直接返回
	if strings.Contains(username, "\\") {
		return username
	}
	// 默认使用 DOMAIN\username 格式
	domainParts := strings.Split(c.config.DomainName, ".")
	if len(domainParts) > 0 {
		return fmt.Sprintf("%s\\%s", strings.ToUpper(domainParts[0]), username)
	}
	return username
}

// Close 关闭连接
func (c *LDAPClient) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// IsConnected 检查连接状态
func (c *LDAPClient) IsConnected() bool {
	return c.conn != nil
}

// ==================== 搜索操作 ====================

// SearchOUs 搜索OU组织单位
func (c *LDAPClient) SearchOUs(baseDN string) ([]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,     // SizeLimit, 0表示无限制
		0,     // TimeLimit, 0表示无限制
		false, // TypesOnly
		"(objectClass=organizationalUnit)",
		[]string{"dn", "ou", "description", "whenCreated", "whenChanged"},
		nil,
	)

	sr, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("搜索OU失败: %w", err)
	}

	return sr.Entries, nil
}

// SearchGroups 搜索用户组
func (c *LDAPClient) SearchGroups(baseDN string) ([]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		"(objectClass=group)",
		[]string{
			"dn", "cn", "description", "groupType", "memberOf",
			"member", "whenCreated", "whenChanged", "distinguishedName",
		},
		nil,
	)

	sr, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("搜索用户组失败: %w", err)
	}

	return sr.Entries, nil
}

// SearchUsers 搜索用户
func (c *LDAPClient) SearchUsers(baseDN string) ([]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		"(&(objectClass=user)(!(objectClass=computer)))",
		[]string{
			"dn", "cn", "sAMAccountName", "displayName", "mail",
			"telephoneNumber", "mobile", "title", "department", "company",
			"userAccountControl", "lastLogon", "pwdLastSet", "accountExpires",
			"description", "memberOf", "distinguishedName", "whenCreated",
			"whenChanged", "userPrincipalName",
		},
		nil,
	)

	sr, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("搜索用户失败: %w", err)
	}

	return sr.Entries, nil
}

// SearchUsersByOU 搜索指定OU下的用户
func (c *LDAPClient) SearchUsersByOU(baseDN, ouDN string) ([]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		ouDN,
		ldap.ScopeSingleLevel,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		"(&(objectClass=user)(!(objectClass=computer)))",
		[]string{
			"dn", "cn", "sAMAccountName", "displayName", "mail",
			"telephoneNumber", "mobile", "title", "department", "company",
			"userAccountControl", "lastLogon", "pwdLastSet", "accountExpires",
			"description", "memberOf", "distinguishedName",
		},
		nil,
	)

	sr, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("搜索OU用户失败: %w", err)
	}

	return sr.Entries, nil
}

// SearchGroupMembers 搜索用户组成员
func (c *LDAPClient) SearchGroupMembers(groupDN string) ([]*ldap.Entry, error) {
	// 先获取组的成员DN列表
	groupRequest := ldap.NewSearchRequest(
		groupDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		"(objectClass=group)",
		[]string{"member"},
		nil,
	)

	sr, err := c.conn.Search(groupRequest)
	if err != nil {
		return nil, fmt.Errorf("搜索组失败: %w", err)
	}

	if len(sr.Entries) == 0 {
		return []*ldap.Entry{}, nil
	}

	memberDNs := sr.Entries[0].GetAttributeValues("member")
	if len(memberDNs) == 0 {
		return []*ldap.Entry{}, nil
	}

	// 根据成员DN获取用户详情
	// 由于LDAP查询限制，可能需要分批查询
	var users []*ldap.Entry

	// 构建过滤条件
	filterParts := make([]string, len(memberDNs))
	for i, dn := range memberDNs {
		filterParts[i] = fmt.Sprintf("(distinguishedName=%s)", ldap.EscapeFilter(dn))
	}
	filter := fmt.Sprintf("(&(|%s)(objectClass=user))", strings.Join(filterParts, ""))

	// 在整个域中搜索这些用户
	searchRequest := ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{
			"dn", "cn", "sAMAccountName", "displayName", "mail",
			"telephoneNumber", "mobile", "title", "department",
			"userAccountControl", "description", "distinguishedName",
		},
		nil,
	)

	userSR, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("搜索组成员失败: %w", err)
	}

	users = userSR.Entries
	return users, nil
}

// ==================== 更新操作 ====================

// UpdateUserAttribute 更新用户属性
func (c *LDAPClient) UpdateUserAttribute(userDN string, attributes map[string]string) error {
	modifyRequest := ldap.NewModifyRequest(userDN, nil)

	for attr, value := range attributes {
		if value == "" {
			// 空值则删除属性
			modifyRequest.Delete(attr, []string{})
		} else {
			// 替换属性值
			modifyRequest.Replace(attr, []string{value})
		}
	}

	err := c.conn.Modify(modifyRequest)
	if err != nil {
		return fmt.Errorf("更新用户属性失败: %w", err)
	}

	return nil
}

// UpdateUserMultipleAttributes 更新用户的多个属性
func (c *LDAPClient) UpdateUserMultipleAttributes(userDN string, attributes map[string][]string) error {
	modifyRequest := ldap.NewModifyRequest(userDN, nil)

	for attr, values := range attributes {
		if len(values) == 0 {
			modifyRequest.Delete(attr, []string{})
		} else {
			modifyRequest.Replace(attr, values)
		}
	}

	err := c.conn.Modify(modifyRequest)
	if err != nil {
		return fmt.Errorf("更新用户属性失败: %w", err)
	}

	return nil
}

// UpdateGroupAttribute 更新用户组属性
func (c *LDAPClient) UpdateGroupAttribute(groupDN string, attributes map[string]string) error {
	modifyRequest := ldap.NewModifyRequest(groupDN, nil)

	for attr, value := range attributes {
		if value == "" {
			modifyRequest.Delete(attr, []string{})
		} else {
			modifyRequest.Replace(attr, []string{value})
		}
	}

	err := c.conn.Modify(modifyRequest)
	if err != nil {
		return fmt.Errorf("更新用户组属性失败: %w", err)
	}

	return nil
}

// MoveUser 移动用户到其他OU
func (c *LDAPClient) MoveUser(userDN, newParentDN string) error {
	// 提取RDN (Relative Distinguished Name)
	rdn := c.extractRDN(userDN)

	// 使用ModifyDN操作移动用户
	modifyDNRequest := ldap.NewModifyDNRequest(userDN, rdn, true, newParentDN)

	err := c.conn.ModifyDN(modifyDNRequest)
	if err != nil {
		return fmt.Errorf("移动用户失败: %w", err)
	}

	return nil
}

// ==================== 用户状态控制 ====================

// EnableUser 启用用户账户
func (c *LDAPClient) EnableUser(userDN string) error {
	// 获取当前userAccountControl值
	user, err := c.getUserByDN(userDN)
	if err != nil {
		return err
	}

	currentUAC := user.GetAttributeValue("userAccountControl")
	uacInt := c.parseIntOrDefault(currentUAC, 512)

	// 移除ACCOUNTDISABLE标志 (0x0002 = 2)
	newUAC := uacInt &^ models.ADAccountDisable

	return c.UpdateUserAttribute(userDN, map[string]string{
		"userAccountControl": fmt.Sprintf("%d", newUAC),
	})
}

// DisableUser 禁用用户账户
func (c *LDAPClient) DisableUser(userDN string) error {
	// 获取当前userAccountControl值
	user, err := c.getUserByDN(userDN)
	if err != nil {
		return err
	}

	currentUAC := user.GetAttributeValue("userAccountControl")
	uacInt := c.parseIntOrDefault(currentUAC, 512)

	// 添加ACCOUNTDISABLE标志 (0x0002 = 2)
	newUAC := uacInt | models.ADAccountDisable

	return c.UpdateUserAttribute(userDN, map[string]string{
		"userAccountControl": fmt.Sprintf("%d", newUAC),
	})
}

// UnlockUser 解锁用户账户
func (c *LDAPClient) UnlockUser(userDN string) error {
	// 获取当前userAccountControl值
	user, err := c.getUserByDN(userDN)
	if err != nil {
		return err
	}

	currentUAC := user.GetAttributeValue("userAccountControl")
	uacInt := c.parseIntOrDefault(currentUAC, 512)

	// 移除LOCKOUT标志 (0x0010 = 16)
	newUAC := uacInt &^ models.ADLockout

	// 还需要重置lockoutTime属性为0
	return c.UpdateUserMultipleAttributes(userDN, map[string][]string{
		"userAccountControl": {fmt.Sprintf("%d", newUAC)},
		"lockoutTime":        {"0"},
	})
}

// ResetPassword 重置用户密码
func (c *LDAPClient) ResetPassword(userDN, newPassword string) error {
	// 密码修改需要特殊的编码
	encodedPassword := c.encodePassword(newPassword)

	modifyRequest := ldap.NewModifyRequest(userDN, nil)
	modifyRequest.Replace("unicodePwd", []string{encodedPassword})

	err := c.conn.Modify(modifyRequest)
	if err != nil {
		return fmt.Errorf("重置密码失败: %w", err)
	}

	return nil
}

// ==================== 用户组成员管理 ====================

// AddGroupMember 添加用户到用户组
func (c *LDAPClient) AddGroupMember(groupDN, userDN string) error {
	modifyRequest := ldap.NewModifyRequest(groupDN, nil)
	modifyRequest.Add("member", []string{userDN})

	err := c.conn.Modify(modifyRequest)
	if err != nil {
		return fmt.Errorf("添加组成员失败: %w", err)
	}

	return nil
}

// RemoveGroupMember 从用户组中移除用户
func (c *LDAPClient) RemoveGroupMember(groupDN, userDN string) error {
	modifyRequest := ldap.NewModifyRequest(groupDN, nil)
	modifyRequest.Delete("member", []string{userDN})

	err := c.conn.Modify(modifyRequest)
	if err != nil {
		return fmt.Errorf("移除组成员失败: %w", err)
	}

	return nil
}

// ==================== 辅助方法 ====================

// getUserByDN 根据DN获取用户条目
func (c *LDAPClient) getUserByDN(userDN string) (*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		userDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		"(objectClass=user)",
		[]string{"dn", "userAccountControl"},
		nil,
	)

	sr, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	if len(sr.Entries) == 0 {
		return nil, fmt.Errorf("用户不存在")
	}

	return sr.Entries[0], nil
}

// extractRDN 从DN中提取RDN
// 例如: "CN=user1,OU=Sales,DC=example,DC=com" -> "CN=user1"
func (c *LDAPClient) extractRDN(dn string) string {
	// 查找第一个逗号
	idx := strings.Index(dn, ",")
	if idx == -1 {
		return dn
	}
	return dn[:idx]
}

// parseIntOrDefault 解析整数，失败返回默认值
func (c *LDAPClient) parseIntOrDefault(s string, defaultValue int) int {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}

// encodePassword 编码密码为AD要求的格式
// AD要求密码用引号括起来，并转换为UTF-16LE
func (c *LDAPClient) encodePassword(password string) string {
	// 密码必须用双引号括起来
	quoted := fmt.Sprintf("\"%s\"", password)
	// 转换为UTF-16LE编码
	runes := []rune(quoted)
	result := make([]byte, 0, len(runes)*2)
	for _, r := range runes {
		result = append(result, byte(r))
		result = append(result, byte(r>>8))
	}
	return string(result)
}

// GetConnection 获取底层LDAP连接（用于高级操作）
func (c *LDAPClient) GetConnection() *ldap.Conn {
	return c.conn
}

// Ping 测试连接是否正常
func (c *LDAPClient) Ping() error {
	if c.conn == nil {
		return fmt.Errorf("连接未建立")
	}
	// 执行一个简单的搜索操作来测试连接
	_, err := c.conn.Search(ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1,
		0,
		false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	))
	return err
}
