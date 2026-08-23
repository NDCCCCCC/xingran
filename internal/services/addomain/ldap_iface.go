package addomain

import (
	"github.com/go-ldap/ldap/v3"
)

// LDAPClientIface 抽象 LDAP 客户端的操作，使 AD 同步服务可在测试中被 mock。
//
// 实现方：
//   - *LDAPClient（生产代码，调用真实 go-ldap 库）
//   - mockLDAPClient（测试用，见 ldap_client_mock_test.go）
//
// 接口覆盖 Connect/Close 生命周期、只读搜索（OU/Group/User/Computer），
// 以及 group_sync_service 与 user_ou_service 实际调用的写操作（成员增删、组创建/删除）。
//
// P2-A6: 抽取该接口以便 AD 关键路径（LDAP 连接、组同步、用户 OU 同步）
// 可在不依赖真实 AD 服务器的条件下进行单元测试，覆盖率达 ≥70%。
type LDAPClientIface interface {
	// 连接与生命周期
	Connect() error
	Close()

	// 只读搜索（分页）
	SearchOUs(baseDN string) ([]*ldap.Entry, error)
	SearchGroups(baseDN string) ([]*ldap.Entry, error)
	SearchUsers(baseDN string) ([]*ldap.Entry, error)
	SearchComputers(baseDN string) ([]*ldap.Entry, error)

	// 组成员管理（group_sync_service 使用）
	AddGroupMember(groupDN, userDN string) error
	RemoveGroupMember(groupDN, userDN string) error
	AddGroupMembers(groupDN string, userDNs []string) error
	RemoveGroupMembers(groupDN string, userDNs []string) error
	CreateGroup(groupDN, groupName, description string, groupType int) error
	DeleteGroup(groupDN string) error

	// 用户属性管理（user_ou_service 使用）
	UpdateUserAttribute(userDN string, attrs map[string]string) error
	MoveUser(userDN, newOUDN string) error
	EnableUser(userDN string) error
	DisableUser(userDN string) error

	// 组属性管理（group.go failover 闭包使用）
	UpdateGroupAttribute(groupDN string, attrs map[string]string) error
	// OU 管理（dept_sync_service.go failover 闭包使用）
	CreateOU(ouDN, ouName string) error
	// DN 存在性预检（user_ad_sync_service.go failover 闭包使用）
	DNExists(dn string) (bool, error)
}

// 编译期断言：*LDAPClient 必须满足 LDAPClientIface。
// 若 LDAPClient 漏实现方法，编译会立即失败，避免运行时才发现接口不匹配。
var _ LDAPClientIface = (*LDAPClient)(nil)
