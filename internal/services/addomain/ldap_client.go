package addomain

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// LDAPClient LDAP客户端封装
type LDAPClient struct {
	conn   *ldap.Conn
	config *models.ADConfig
}

// NewLDAPClient 创建LDAP客户端
func NewLDAPClient(config *models.ADConfig) *LDAPClient {
	return &LDAPClient{
		config: config,
	}
}

// Connect 连接到AD服务器
func (c *LDAPClient) Connect() error {
	var conn *ldap.Conn
	var err error

	// 构建连接地址
	address := fmt.Sprintf("%s:%d", c.config.ServerAddress, c.config.ServerPort)

	if c.config.UseSSL {
		// 使用LDAPS
		conn, err = ldap.DialTLS("tcp", address, &tls.Config{
			InsecureSkipVerify: true, // 生产环境应该验证证书
		})
	} else if c.config.UseTLS {
		// 使用StartTLS
		conn, err = ldap.Dial("tcp", address)
		if err != nil {
			return err
		}
		err = conn.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
		})
	} else {
		// 普通连接
		conn, err = ldap.Dial("tcp", address)
	}

	if err != nil {
		return fmt.Errorf("连接AD服务器失败: %w", err)
	}

	// 绑定管理员账号
	bindDN := fmt.Sprintf("%s\\%s", c.config.DomainName, c.config.AdminUsername)
	err = conn.Bind(bindDN, c.config.AdminPassword)
	if err != nil {
		conn.Close()
		return fmt.Errorf("绑定管理员账号失败: %w", err)
	}

	c.conn = conn
	return nil
}

// Close 关闭连接
func (c *LDAPClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// SearchOUs 搜索OU
func (c *LDAPClient) SearchOUs(baseDN string) ([]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=organizationalUnit)",
		[]string{"dn", "ou", "description"},
		nil,
	)

	sr, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	return sr.Entries, nil
}

// SearchGroups 搜索用户组
func (c *LDAPClient) SearchGroups(baseDN string) ([]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=group)",
		[]string{"dn", "cn", "description", "member"},
		nil,
	)

	sr, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	return sr.Entries, nil
}

// SearchUsers 搜索用户
func (c *LDAPClient) SearchUsers(baseDN string) ([]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=user)",
		[]string{
			"dn", "sAMAccountName", "displayName", "mail", "telephoneNumber",
			"mobile", "title", "department", "company", "description",
			"userAccountControl", "memberOf", "lastLogon", "pwdLastSet",
		},
		nil,
	)

	sr, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	return sr.Entries, nil
}

// UpdateUserAttribute 更新用户属性
func (c *LDAPClient) UpdateUserAttribute(userDN string, attrs map[string]string) error {
	modifyRequest := ldap.NewModifyRequest(userDN, nil)
	for attr, value := range attrs {
		modifyRequest.Replace(attr, []string{value})
	}

	return c.conn.Modify(modifyRequest)
}

// UpdateGroupAttribute 更新用户组属性
func (c *LDAPClient) UpdateGroupAttribute(groupDN string, attrs map[string]string) error {
	modifyRequest := ldap.NewModifyRequest(groupDN, nil)
	for attr, value := range attrs {
		modifyRequest.Replace(attr, []string{value})
	}

	return c.conn.Modify(modifyRequest)
}

// EnableUser 启用用户
func (c *LDAPClient) EnableUser(userDN string) error {
	// 设置userAccountControl，移除ACCOUNTDISABLE (0x0002)
	modifyRequest := ldap.NewModifyRequest(userDN, nil)
	// 这里简化处理，实际需要读取当前UAC值并修改
	modifyRequest.Replace("userAccountControl", []string{"512"}) // NORMAL_ACCOUNT

	return c.conn.Modify(modifyRequest)
}

// DisableUser 禁用用户
func (c *LDAPClient) DisableUser(userDN string) error {
	// 设置userAccountControl，添加ACCOUNTDISABLE (0x0002)
	modifyRequest := ldap.NewModifyRequest(userDN, nil)
	// 这里简化处理，实际需要读取当前UAC值并修改
	modifyRequest.Replace("userAccountControl", []string{"514"}) // NORMAL_ACCOUNT | ACCOUNTDISABLE

	return c.conn.Modify(modifyRequest)
}

// MoveUser 移动用户到其他OU
func (c *LDAPClient) MoveUser(userDN, newOUDN string) error {
	// 使用 ModifyDN 操作移动用户
	// 提取当前DN的RDN (相对标识名)
	rdn := extractRDNFromDN(userDN)
	if rdn == "" {
		return fmt.Errorf("无法从用户DN提取RDN")
	}

	// NewModifyDNRequest(dn, newRelativeDN, deleteOldRDN, newParent)
	req := ldap.NewModifyDNRequest(userDN, rdn, true, newOUDN)
	return c.conn.ModifyDN(req)
}

// AddGroupMember 添加用户组成员
func (c *LDAPClient) AddGroupMember(groupDN, userDN string) error {
	modifyRequest := ldap.NewModifyRequest(groupDN, nil)
	modifyRequest.Add("member", []string{userDN})

	return c.conn.Modify(modifyRequest)
}

// RemoveGroupMember 移除用户组成员
func (c *LDAPClient) RemoveGroupMember(groupDN, userDN string) error {
	modifyRequest := ldap.NewModifyRequest(groupDN, nil)
	modifyRequest.Delete("member", []string{userDN})

	return c.conn.Modify(modifyRequest)
}

// extractRDNFromDN 从DN中提取RDN (相对标识名)
func extractRDNFromDN(dn string) string {
	// 查找第一个逗号
	idx := strings.Index(dn, ",")
	if idx == -1 {
		return dn
	}
	return dn[:idx]
}
