package addomain

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/go-ldap/ldap/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// F-01: 仅记录一次 TLS 跳过验证的安全告警,避免每次 LDAP 连接重复刷屏。
var (
	tlsInsecureWarnOnce sync.Once
)

func logTLSInsecureSkipOnce() {
	tlsInsecureWarnOnce.Do(func() {
		applogger.Warnf(
			"[SECURITY] LDAP TLS InsecureSkipVerify=true 已启用 (env LDAP_TLS_INSECURE_SKIP_VERIFY) — " +
				"流量可能被 MITM,仅供内网自签证书过渡期使用,生产建议导入 AD 服务器 CA 证书并取消该环境变量",
		)
	})
}

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
//
// Phase 36 变更：增加 account 字段（用于 Bind 时从账号池选定的具体账号）
// 向后兼容：传 nil 时 fallback 到 config.AdminUsername/AdminPassword
type LDAPClient struct {
	conn    *ldap.Conn
	config  *models.ADConfig
	account *models.ADServiceAccount // Phase 36: 可选，来自账号池
}

// NewLDAPClient 创建LDAP客户端（兼容旧调用，account 传 nil 走 config 字段）
func NewLDAPClient(config *models.ADConfig, account ...*models.ADServiceAccount) *LDAPClient {
	c := &LDAPClient{
		config: config,
	}
	if len(account) > 0 && account[0] != nil {
		c.account = account[0]
	}
	return c
}

// GetAccount 返回当前 client 绑定的服务账号（Phase 36，失败上报时用）
func (c *LDAPClient) GetAccount() *models.ADServiceAccount {
	return c.account
}

// Conn 返回底层已 Bind 成功的 *ldap.Conn，供外部包（如 ad_authenticator 的用户搜索）
// 在同一已绑定连接上执行操作。
// 调用方不得自行 Close 返回的 conn——生命周期由 LDAPClient.Close 统一管理。
func (c *LDAPClient) Conn() *ldap.Conn {
	return c.conn
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
//
// F-01 fix: tlsConfig.InsecureSkipVerify 不再硬编码 true,而是读环境变量
// LDAP_TLS_INSECURE_SKIP_VERIFY (默认 false = 严格校验)。
// 生产部署默认拒绝自签证书,杜绝 MITM;
// 需要兼容自签证书的内网部署可显式 export LDAP_TLS_INSECURE_SKIP_VERIFY=true。
func (c *LDAPClient) dialConnection(address string) (*ldap.Conn, error) {
	insecureSkip := strings.EqualFold(os.Getenv("LDAP_TLS_INSECURE_SKIP_VERIFY"), "true")
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkip,
		MinVersion:         tls.VersionTLS12,
	}
	if insecureSkip {
		// 仅在启用时记录一次安全告警(避免每次连接都告警污染日志,但不能完全静默)
		logTLSInsecureSkipOnce()
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
//
// Phase 36 修复（C1）：优先使用 c.account 的凭证（账号池入参），fallback 到 c.config
// 否则 FailoverClient 遍历多账号时所有账号都用同一密码绑定 → 账号池失效
func (c *LDAPClient) tryBindAttempts(conn *ldap.Conn, domain string) error {
	password := c.config.AdminPassword
	username := c.config.AdminUsername
	if c.account != nil {
		// Phase 36: 账号池模式。
		// password_ciphertext 写入时由 core.SM4Cipher.Encrypt() 加密（ad_account_handler.go），
		// bind 前必须解密得到明文密码——LDAP bind 需要明文，传密文必然 LDAP error 49。
		// 与单管理员路径（config.AdminPassword 的 decryptPassword 处理）保持一致。
		// 历史契约"caller 负责解密"已废弃：FailoverClient 遍历多账号极易遗漏，改为内部解密更鲁棒。
		password = decryptPassword(c.account.PasswordCiphertext)
		username = c.account.Username
	}
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
		"(&(objectClass=user)(!(objectClass=computer))(!(cn=*DUPLICATE-*)))", // 排除计算机账号和AD复制冲突对象
		[]string{
			"dn", "sAMAccountName", "displayName", "mail", "telephoneNumber",
			"mobile", "title", "department", "company", "description",
			"userAccountControl", "memberOf", "lastLogon", "pwdLastSet",
			// manager：域控 manager 属性（DN 格式），影子表 sys_ad_user.manager 依赖它回显
			"manager",
		},
		defaultPageSize,
	)
}

// SearchWithRequest 执行调用方构造的原始 LDAP 搜索请求
// 76-03 接口化编译门强制补入：group_sync_service.go SyncSingleGroup 闭包
// 原直接访问 client.conn.Search，闭包参数接口化后不可达，经此委托方法保持
// 行为与错误语义完全不变（纯转发，零逻辑）。
func (c *LDAPClient) SearchWithRequest(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error) {
	return c.conn.Search(searchRequest)
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

// DeleteUserAttribute 删除单个用户属性（Modify delete 语义）。
// 用途：清空 AD manager —— Replace 空值在 AD 上会报错，必须走 delete。
func (c *LDAPClient) DeleteUserAttribute(userDN, attr string) error {
	modifyRequest := ldap.NewModifyRequest(userDN, nil)
	modifyRequest.Delete(attr, nil)
	return c.conn.Modify(modifyRequest)
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

// groupMemberRangePageSize 是 AD 组成员 Range 检索的页大小。
// AD 对单组 member 属性超过 ~1500 时启用 range 检索（member;range=0-1499），
// 必须按页循环读取；2807 个影子成员分布在 67 组（平均 ~42/组），此页大小足够。
const groupMemberRangePageSize = 1500

// GetGroupMembers 读取AD组当前成员DN列表（域控审查 P0-1）
// 通过 SearchWithRequest 直读 member 属性；大组自动走 AD Range 检索循环补页。
func (c *LDAPClient) GetGroupMembers(groupDN string) ([]string, error) {
	var members []string
	attr := "member"
	for {
		searchRequest := ldap.NewSearchRequest(
			groupDN,
			ldap.ScopeBaseObject,
			ldap.NeverDerefAliases,
			0,     // SizeLimit
			0,     // TimeLimit
			false, // TypesOnly
			"(objectClass=*)",
			[]string{attr},
			nil,
		)
		result, err := c.SearchWithRequest(searchRequest)
		if err != nil {
			return nil, fmt.Errorf("查询组 %s 成员失败: %w", groupDN, err)
		}
		if len(result.Entries) == 0 {
			return nil, fmt.Errorf("组 %s 在AD中不存在", groupDN)
		}
		values := result.Entries[0].GetAttributeValues(attr)
		members = append(members, values...)
		if len(values) < groupMemberRangePageSize {
			return members, nil
		}
		// 本页已满：下次请求下一 Range 段（AD 响应中的属性名形如 member;range=1500-）
		attr = fmt.Sprintf("member;range=%d-", len(members))
	}
}

// DeleteOU 删除AD端OU（域控审查 P0-2 扩展：部门删除联动）
// 调用方须先预检"部门无用户且 OU 下无 AD 用户"；OU 下仍有子对象时 AD 返回
// Not Allowed On Non-Leaf（错误码 66），原样上抛由调用方降级处理。
func (c *LDAPClient) DeleteOU(ouDN string) error {
	delRequest := ldap.NewDelRequest(ouDN, nil)
	return c.conn.Del(delRequest)
}

// RenameOU 重命名AD端OU（域控审查 P0-2 扩展：部门改名联动）
// ModifyDN 只改 RDN（OU 名），parent 不变，子树对象 DN 随动（LDAP 语义）；
// RDN 拼接规则与 dn_util.go BuildOUDN 保持一致（ldap.EscapeDN）。
func (c *LDAPClient) RenameOU(ouDN, newOUName string) error {
	rdn := "OU=" + ldap.EscapeDN(newOUName)
	parentDN := parentDNFromDN(ouDN)
	if parentDN == "" {
		return fmt.Errorf("无法从OU DN提取父DN: %s", ouDN)
	}
	req := ldap.NewModifyDNRequest(ouDN, rdn, true, parentDN)
	return c.conn.ModifyDN(req)
}

// extractRDNFromDN 从DN中提取RDN（第一个 RDN）
//
// 按 RFC4514，逗号只有在**未被转义**时才是 RDN 分隔符 —— 即其前面是偶数个连续反斜杠。
// 值内的转义逗号（CN=Zhang\,San）属于 RDN 本身，不是分隔符。
//
// 为什么不能直接 strings.Index(dn, ",")：本函数的产物会喂给 MoveUser 的
// ldap.NewModifyDNRequest(userDN, rdn, true, newOUDN)，deleteOldRDN=true 会让 AD
// 把对象改名成给定的 RDN 并删除旧 RDN。按字面逗号切会把 CN=Zhang\,San 截成
// `CN=Zhang\`，AD 对象的名字就此永久丢失（Phase 58 遗留 MISSED-04，数据破坏级）。
//
// Phase 58 只给 OU 侧加了 ldap.EscapeDN（见 dn_util.go 的 BuildOUDN），RDN 侧此前
// 仍是字面切分 —— 本函数补齐这半边。
//
// 实现要点：
//   - 单趟**字节**扫描。RFC4514 的转义字符与分隔符都是 ASCII，中文等多字节 UTF-8 值
//     的续字节不可能等于 ',' 或 '\'，按字节遍历天然安全；反之若按 rune 迭代
//     （for range）再拿 rune 当下标，会引入偏移错位。
//   - escaping 状态位天然实现"奇偶反斜杠"语义：两个连续反斜杠是「反斜杠 + 被转义的
//     反斜杠」，状态位回到 false，故其后的逗号判为未转义分隔符。无需另数反斜杠个数。
//   - 输入是**已在库里的 AD 侧 DN**，本就是合法转义形态；不要改用 NormalizeDN 的口径
//     （NormalizeDN 刻意不用 ldap.ParseDN，因为存量 sys_dept_ou_mapping.ou_dn 含未转义
//     特殊字符），二者期望语义本就不同，不要顺手"统一"。
func extractRDNFromDN(dn string) string {
	escaping := false
	for i := 0; i < len(dn); i++ {
		switch dn[i] {
		case '\\':
			escaping = !escaping
		case ',':
			if !escaping {
				return dn[:i]
			}
			escaping = false
		default:
			escaping = false
		}
	}
	return dn
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

// DNExists 预检目标 DN 是否存在（避免对不存在的对象执行 Modify/ModifyDN
// 返回 LDAP code 32 "No Such Object"）。
//
// 用途（debug session ad-update-attr-no-such-object）：
//   - user_ad_sync_service.syncUserAttributes 调用前先 DNExists，
//     避免 sys_user.ad_dn 残留过期 DN（如 AD 端对象已被删/移走）
//     触发 modify 失败 code 32 → handler 3 次重试 → 每次 MarkFailure
//     累加 → 应用层 breaker 熔断 30 分钟 → 全池 bind 失败 → 用户看到
//     "管理员账号被锁"。
//   - 上游根据返回值决定：false → 清空 ad_dn + INFO 跳过，让下次 login sync
//     重新拉取。
//
// 与 OUExists 区别：OUExists 只验证 OU 形态（objectClass=organizationalUnit），
// DNExists 对任意 DN 做 base scope (objectClass=*) 存在性预检，
// 用于保护 Modify/ModifyDN 操作。
func (c *LDAPClient) DNExists(dn string) (bool, error) {
	if dn == "" {
		return false, fmt.Errorf("DN 不能为空")
	}
	searchRequest := ldap.NewSearchRequest(
		dn,
		ldap.ScopeBaseObject,   // Scope
		ldap.NeverDerefAliases, // DerefAliases
		1,                      // SizeLimit=1: base scope 只命中该 DN 自身
		0,                      // TimeLimit
		false,                  // TypesOnly
		"(objectClass=*)",      // Filter
		[]string{"dn"},         // Attributes
		[]ldap.Control{},       // Controls(必须非 nil,空切片合法)
	)

	_, err := c.conn.Search(searchRequest)
	if err != nil {
		// code 32 = NoSuchObject，语义上等价于"不存在"，对外返回 (false, nil)
		// 避免上游把"对象不存在"误判为"网络/认证错误"导致不必要的重试
		if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
			return false, nil
		}
		return false, fmt.Errorf("DN 存在性预检失败: %w", err)
	}
	return true, nil
}
