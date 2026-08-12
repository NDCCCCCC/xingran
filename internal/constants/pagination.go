// 分页相关常量:统一管理分页参数,避免硬编码。

package constants

// 分页大小常量
const (
	// DefaultCurrent 默认页码
	DefaultCurrent = 1

	// DefaultPageSize 默认每页记录数
	DefaultPageSize = 10

	// MinPageSize 最小每页记录数
	MinPageSize = 10

	// MaxListPageSize 表格列表分页的单页上限(防 DoS、控制单次响应大小)。
	//
	// 用于 system 等模块的表格 list 端点。总页数应从后端独立 COUNT 返回的
	// total 字段计算(ceil(total/pageSize)),与该上限无关;切勿用被钳制
	// 后的 list.length 充当总数(参见 stat-cards-from-list-length-capped-at-100 教训)。
	MaxListPageSize = 100

	// MaxOptionsPageSize 下拉全集 / 批量场景的单页上限。
	//
	// 用于 operations 等模块前端 Select 一次性拉取全集 option 的场景
	// (如选工位、选部门成员)。这是对"把 list 当全集下拉"反式的兼容,
	// 新功能应优先使用专用 options 端点而非放大分页上限。
	MaxOptionsPageSize = 10000
)
