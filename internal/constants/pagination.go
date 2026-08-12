/**
 * 分页相关常量
 *
 * 统一管理分页参数，避免硬编码
 */
package constants

// 分页大小常量
const (
	// 默认页码
	DefaultCurrent = 1

	// 默认每页记录数
	DefaultPageSize = 10

	// 最小每页记录数
	MinPageSize = 10

	// 最大每页记录数
	MaxPageSize = 100

	// LDAP 默认分页大小
	LDAPDefaultPageSize = 1000

	// LDAP 最小分页大小
	LDAPMinPageSize = 100
)

// 分页验证规则
const (
	// 最小允许的分页大小
	MinAllowedPageSize = 1
)
