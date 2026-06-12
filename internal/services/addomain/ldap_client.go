package addomain

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

const (
	// 默认分页大小
	defaultPageSize = 1000

	// 最小分页大小（用于重试）
	minPageSize = 100

	// 用户账户控制值
	uacNormalAccount   = "512" // NORMAL_ACCOUNT
	uacDisabledAccount = "514" // NORMAL_ACCOUNT | ACCOUNTDISABLE
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
	address := fmt.Sprintf("%s:%d", c.config.ServerAddress, c.config.ServerPort)
	conn, err := c.dialConnection(address)
	if err != nil {
		return err
	}

	cleanDomain := c.cleanDomain()
	if err := c.tryBindAttempts(conn, cleanDomain); err != nil {
		conn.Close()
		return err
	}

	c.conn = conn
	return nil
}

// dialConnection 根据配置建立连接
func (c *LDAPClient) dialConnection(address string) (*ldap.Conn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	switch {
	case c.config.UseSSL:
		return ldap.DialURL("ldaps://"+address, ldap.DialWithTLSConfig(tlsConfig))
	case c.config.UseTLS:
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

// cleanDomain 清理域名
func (c *LDAPClient) cleanDomain() string {
	suffix := fmt.Sprintf("@%s", c.config.DomainName)
	cleanDomain := strings.TrimSuffix(c.config.DomainName, suffix)
	if cleanDomain == "" {
		return c.config.DomainName
	}
	return cleanDomain
}

// tryBindAttempts 尝试多种绑定方式
func (c *LDAPClient) tryBindAttempts(conn *ldap.Conn, domain string) error {
	password := c.config.AdminPassword
	username := c.config.AdminUsername
	netbiosName := extractNetBIOSName(domain)

	attempts := []struct {
		name     string
		username string
	}{
		{"UPN", fmt.Sprintf("%s@%s", username, domain)},
		{"NetBIOS", fmt.Sprintf("%s\\%s", netbiosName, username)},
		{"Direct", username},
	}

	var lastErr error
	for _, attempt := range attempts {
		err := conn.Bind(attempt.username, password)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("绑定失败: %w (尝试: UPN, NetBIOS, 直连)", lastErr)
}

// Close 关闭连接
func (c *LDAPClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// SearchOUs 搜索OU（使用分页搜索）
func (c *LDAPClient) SearchOUs(baseDN string) ([]*ldap.Entry, error) {
	return c.searchWithPaging(
		baseDN,
		"(objectClass=organizationalUnit)",
		[]string{"dn", "ou", "description"},
		defaultPageSize,
	)
}

// SearchGroups 搜索用户组（使用分页搜索）
func (c *LDAPClient) SearchGroups(baseDN string) ([]*ldap.Entry, error) {
	return c.searchWithPaging(
		baseDN,
		"(objectClass=group)",
		[]string{"dn", "cn", "description", "member", "groupType"},
		defaultPageSize,
	)
}

// SearchUsers 搜索用户（使用分页搜索）
func (c *LDAPClient) SearchUsers(baseDN string) ([]*ldap.Entry, error) {
	return c.searchWithPaging(
		baseDN,
		"(&(objectClass=user)(!(cn=*DUPLICATE-*)))", // 排除AD复制冲突对象
		[]string{
			"dn", "sAMAccountName", "displayName", "mail", "telephoneNumber",
			"mobile", "title", "department", "company", "description",
			"userAccountControl", "memberOf", "lastLogon", "pwdLastSet",
		},
		defaultPageSize,
	)
}

// SearchComputers 搜索计算机（使用分页搜索）
func (c *LDAPClient) SearchComputers(baseDN string) ([]*ldap.Entry, error) {
	return c.searchWithPaging(
		baseDN,
		"(objectClass=computer)",
		[]string{
			"dn", "cn", "description", "lastLogon", "pwdLastSet", "logonCount",
			"operatingSystem", "operatingSystemVersion", "managedBy",
			"dNSHostName", "netbootGUID",
		},
		defaultPageSize,
	)
}

// searchWithPaging 使用 LDAP 分页控制进行搜索
func (c *LDAPClient) searchWithPaging(baseDN, filter string, attributes []string, pageSize uint32) ([]*ldap.Entry, error) {
	return c.searchWithPagingDepth(baseDN, filter, attributes, pageSize, 0)
}

// P1 fix: maxSearchRetryDepth 限制 handleSearchError 的 SizeLimitExceeded
// 递归降级深度。原实现 minPageSize=100 时最多 ~log2(2^32/100) ≈ 25 层,
// 理论上不致栈溢出,但配合 attacker 控制的 LDAP 服务器可制造更深降级链。
// 4 层(pageSize 减半 4 次 = 1/16)足够覆盖正常分页限制场景。
const maxSearchRetryDepth = 4

// searchWithPagingDepth 带递归深度控制的内部实现
func (c *LDAPClient) searchWithPagingDepth(baseDN, filter string, attributes []string, pageSize uint32, depth int) ([]*ldap.Entry, error) {
	var allEntries []*ldap.Entry
	pagingControl := ldap.NewControlPaging(pageSize)

	for {
		searchRequest := ldap.NewSearchRequest(
			baseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0,     // SizeLimit: 0 表示无限制（由分页控制）
			0,     // TimeLimit: 0 表示无限制
			false, // TypesOnly
			filter,
			attributes,
			[]ldap.Control{pagingControl},
		)

		sr, err := c.conn.Search(searchRequest)
		if err != nil {
			return c.handleSearchError(err, baseDN, filter, attributes, pageSize, depth)
		}

		allEntries = append(allEntries, sr.Entries...)

		// 检查是否还有更多页
		responsePagingControl := c.extractPagingControl(sr.Controls)
		if responsePagingControl == nil || len(responsePagingControl.Cookie) == 0 {
			break
		}

		// 准备下一页
		pagingControl = ldap.NewControlPaging(pageSize)
		pagingControl.SetCookie(responsePagingControl.Cookie)
	}

	return allEntries, nil
}

// handleSearchError 处理搜索错误，必要时重试
func (c *LDAPClient) handleSearchError(err error, baseDN, filter string, attributes []string, pageSize uint32, depth int) ([]*ldap.Entry, error) {
	if ldap.IsErrorWithCode(err, ldap.LDAPResultSizeLimitExceeded) {
		if depth >= maxSearchRetryDepth {
			return nil, fmt.Errorf("搜索结果超过服务器限制,递归降级 %d 次后仍失败 (pageSize=%d)", depth, pageSize)
		}
		if pageSize > minPageSize {
			return c.searchWithPagingDepth(baseDN, filter, attributes, pageSize/2, depth+1)
		}
		return nil, fmt.Errorf("搜索结果超过服务器限制，即使使用分页也无法获取所有结果")
	}
	return nil, fmt.Errorf("搜索失败: %w", err)
}

// extractPagingControl 从响应控件中提取分页控制
func (c *LDAPClient) extractPagingControl(controls []ldap.Control) *ldap.ControlPaging {
	for _, ctrl := range controls {
		if pc, ok := ctrl.(*ldap.ControlPaging); ok {
			return pc
		}
	}
	return nil
}

// UpdateUserAttribute 更新用户属性
func (c *LDAPClient) UpdateUserAttribute(userDN string, attrs map[string]string) error {
	return c.updateAttributes(userDN, attrs)
}

// UpdateGroupAttribute 更新用户组属性
func (c *LDAPClient) UpdateGroupAttribute(groupDN string, attrs map[string]string) error {
	return c.updateAttributes(groupDN, attrs)
}

// updateAttributes 通用的属性更新方法
func (c *LDAPClient) updateAttributes(dn string, attrs map[string]string) error {
	modifyRequest := ldap.NewModifyRequest(dn, nil)
	for attr, value := range attrs {
		modifyRequest.Replace(attr, []string{value})
	}
	return c.conn.Modify(modifyRequest)
}

// EnableUser 启用用户
func (c *LDAPClient) EnableUser(userDN string) error {
	return c.setUserAccountControl(userDN, uacNormalAccount)
}

// DisableUser 禁用用户
func (c *LDAPClient) DisableUser(userDN string) error {
	return c.setUserAccountControl(userDN, uacDisabledAccount)
}

// setUserAccountControl 设置用户账户控制属性
func (c *LDAPClient) setUserAccountControl(userDN, uacValue string) error {
	modifyRequest := ldap.NewModifyRequest(userDN, nil)
	modifyRequest.Replace("userAccountControl", []string{uacValue})
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

// CreateGroup 创建AD用户组
func (c *LDAPClient) CreateGroup(groupDN, groupName, description string, groupType int) error {
	// 构造组条目
	entry := ldap.NewAddRequest(groupDN, nil)
	entry.Attribute("objectClass", []string{"top", "group"})
	entry.Attribute("sAMAccountName", []string{groupName})
	entry.Attribute("cn", []string{groupName})
	if description != "" {
		entry.Attribute("description", []string{description})
	}
	// groupType: -2147483646 = Global Security Group, -2147483644 = Domain Local Security Group
	// 默认创建全局安全组
	if groupType == 0 {
		groupType = -2147483646
	}
	entry.Attribute("groupType", []string{fmt.Sprintf("%d", groupType)})

	return c.conn.Add(entry)
}

// DeleteGroup 删除AD用户组
func (c *LDAPClient) DeleteGroup(groupDN string) error {
	delRequest := ldap.NewDelRequest(groupDN, nil)
	return c.conn.Del(delRequest)
}

// AddGroupMembers 批量添加组成员
func (c *LDAPClient) AddGroupMembers(groupDN string, userDNs []string) error {
	if len(userDNs) == 0 {
		return nil
	}

	modifyRequest := ldap.NewModifyRequest(groupDN, nil)
	modifyRequest.Add("member", userDNs)

	return c.conn.Modify(modifyRequest)
}

// RemoveGroupMembers 批量移除组成员
func (c *LDAPClient) RemoveGroupMembers(groupDN string, userDNs []string) error {
	if len(userDNs) == 0 {
		return nil
	}

	modifyRequest := ldap.NewModifyRequest(groupDN, nil)
	modifyRequest.Delete("member", userDNs)

	return c.conn.Modify(modifyRequest)
}

// extractRDNFromDN 从DN中提取RDN
func extractRDNFromDN(dn string) string {
	idx := strings.Index(dn, ",")
	if idx == -1 {
		return dn
	}
	return dn[:idx]
}

// extractNetBIOSName 从域名中提取NetBIOS名称
func extractNetBIOSName(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) > 0 {
		return strings.ToUpper(parts[0])
	}
	return strings.ToUpper(domain)
}

// CreateOU 在AD中创建OU（幂等操作）
func (c *LDAPClient) CreateOU(ouDN, ouName string) error {
	// 检查OU是否已存在
	exists, err := c.OUExists(ouDN)
	if err != nil {
		return fmt.Errorf("检查OU存在性失败: %w", err)
	}
	if exists {
		// 已存在，跳过创建
		return nil
	}

	// 创建新OU
	addRequest := ldap.NewAddRequest(ouDN, nil)
	addRequest.Attribute("objectClass", []string{"top", "organizationalUnit"})
	addRequest.Attribute("ou", []string{ouName})
	addRequest.Attribute("description", []string{fmt.Sprintf("同步自系统部门: %s", ouName)})

	if err := c.conn.Add(addRequest); err != nil {
		return fmt.Errorf("创建OU失败 %s: %w", ouDN, err)
	}

	return nil
}

// OUExists 检查OU是否存在
func (c *LDAPClient) OUExists(ouDN string) (bool, error) {
	searchRequest := ldap.NewSearchRequest(
		ouDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0, 0, false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)

	_, err := c.conn.Search(searchRequest)
	if err != nil {
		if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
			return false, nil
		}
		return false, fmt.Errorf("搜索OU失败: %w", err)
	}
	return true, nil
}
