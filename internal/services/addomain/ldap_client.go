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

	// 清理域名：移除可能的 @domain 后缀
	cleanDomain := strings.TrimSuffix(c.config.DomainName, fmt.Sprintf("@%s", c.config.DomainName))
	// 如果清理后为空，使用原始域名
	if cleanDomain == "" {
		cleanDomain = c.config.DomainName
	}

	// 尝试多种绑定方式
	var lastErr error

	// 1. UPN 格式: user@domain.com (推荐)
	bindUsername := fmt.Sprintf("%s@%s", c.config.AdminUsername, cleanDomain)
	if err := conn.Bind(bindUsername, c.config.AdminPassword); err == nil {
		c.conn = conn
		return nil
	}
	lastErr = err

	// 2. NetBIOS 格式: DOMAIN\user (需要从域名提取 NetBIOS 名称)
	netbiosName := extractNetBIOSName(cleanDomain)
	bindUsername = fmt.Sprintf("%s\\%s", netbiosName, c.config.AdminUsername)
	if err := conn.Bind(bindUsername, c.config.AdminPassword); err == nil {
		c.conn = conn
		return nil
	}
	lastErr = err

	// 3. 尝试直接使用用户名（某些AD配置支持）
	bindUsername = c.config.AdminUsername
	if err := conn.Bind(bindUsername, c.config.AdminPassword); err == nil {
		c.conn = conn
		return nil
	}
	lastErr = err

	conn.Close()
	return fmt.Errorf("绑定管理员账号失败: %w (尝试的格式: UPN=%s, NetBIOS=%s, 直连用户名=%s)",
		lastErr,
		fmt.Sprintf("%s@%s", c.config.AdminUsername, cleanDomain),
		fmt.Sprintf("%s\\%s", netbiosName, c.config.AdminUsername),
		c.config.AdminUsername,
	)
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
		1000, // 每页1000条记录
	)
}

// SearchGroups 搜索用户组（使用分页搜索）
func (c *LDAPClient) SearchGroups(baseDN string) ([]*ldap.Entry, error) {
	return c.searchWithPaging(
		baseDN,
		"(objectClass=group)",
		[]string{"dn", "cn", "description", "member"},
		1000, // 每页1000条记录
	)
}

// SearchUsers 搜索用户（使用分页搜索）
func (c *LDAPClient) SearchUsers(baseDN string) ([]*ldap.Entry, error) {
	return c.searchWithPaging(
		baseDN,
		"(objectClass=user)",
		[]string{
			"dn", "sAMAccountName", "displayName", "mail", "telephoneNumber",
			"mobile", "title", "department", "company", "description",
			"userAccountControl", "memberOf", "lastLogon", "pwdLastSet",
		},
		1000, // 每页1000条记录
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
		1000, // 每页1000条记录
	)
}

// searchWithPaging 使用 LDAP 分页控制进行搜索
// pageSize: 每页返回的记录数
func (c *LDAPClient) searchWithPaging(baseDN, filter string, attributes []string, pageSize uint32) ([]*ldap.Entry, error) {
	var allEntries []*ldap.Entry

	// 创建分页控制
	pagingControl := ldap.NewControlPaging(pageSize)

	pageCount := 0
	for {
		pageCount++

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
			// 检查是否是 Size Limit Exceeded 错误
			if ldap.IsErrorWithCode(err, ldap.LDAPResultSizeLimitExceeded) {
				// 尝试使用更小的页面大小重试
				if pageSize > 100 {
					return c.searchWithPaging(baseDN, filter, attributes, pageSize/2)
				}
				return nil, fmt.Errorf("搜索结果超过服务器限制，即使使用分页也无法获取所有结果")
			}
			return nil, fmt.Errorf("搜索失败: %w", err)
		}

		// 添加当前页的结果
		entriesInPage := len(sr.Entries)
		allEntries = append(allEntries, sr.Entries...)

		// 输出调试信息
		fmt.Printf("[LDAP分页] 第 %d 页: 获取到 %d 条记录 (累计: %d)\n",
			pageCount, entriesInPage, len(allEntries))

		// 从响应中提取分页控制
		var responsePagingControl *ldap.ControlPaging
		for _, ctrl := range sr.Controls {
			if pc, ok := ctrl.(*ldap.ControlPaging); ok {
				responsePagingControl = pc
				break
			}
		}

		// 如果没有分页控制或Cookie为空，说明已到最后一页
		if responsePagingControl == nil {
			fmt.Printf("[LDAP分页] 服务器未返回分页控制，停止分页\n")
			break
		}

		if len(responsePagingControl.Cookie) == 0 {
			fmt.Printf("[LDAP分页] Cookie为空，已获取所有数据\n")
			break
		}

		// 准备下一页：使用返回的Cookie创建新的分页控制
		pagingControl = ldap.NewControlPaging(pageSize)
		pagingControl.SetCookie(responsePagingControl.Cookie)
	}

	fmt.Printf("[LDAP分页] 总计: %d 条记录，分 %d 页获取\n", len(allEntries), pageCount)

	// 检查是否可能触发了MaxPageSize限制
	if len(allEntries) == int(pageSize) {
		fmt.Printf("[LDAP分页] ⚠ 警告: 恰好获取到 %d 条记录（等于pageSize），可能触发了AD服务器的MaxPageSize限制\n", pageSize)
		fmt.Printf("[LDAP分页] 建议检查: 1) AD服务器是否返回了空的分页Cookie 2) AD服务器的MaxPageSize设置\n")
	}

	return allEntries, nil
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

// extractNetBIOSName 从域名中提取 NetBIOS 名称
// 例如: example.com -> EXAMPLE, corp.example.com -> CORP
func extractNetBIOSName(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) > 0 {
		// 返回第一部分并转为大写
		return strings.ToUpper(parts[0])
	}
	// 如果没有点，直接返回大写的域名
	return strings.ToUpper(domain)
}
