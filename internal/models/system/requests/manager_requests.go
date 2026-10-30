package requests

// 经理同步（规则/机构）列表查询请求。
// 分页约定与 UserListParams 一致：current 从 1 开始。

// ManagerRuleListRequest 经理同步规则列表请求
type ManagerRuleListRequest struct {
	Current  int    `json:"current"`
	PageSize int    `json:"pageSize"`
	Keyword  string `json:"keyword"` // 模糊匹配规则名
	Enabled  *bool  `json:"enabled"` // nil = 不过滤
}

// DefaultManagerRuleListRequest 列表请求默认值（空/解析失败时回落）
func DefaultManagerRuleListRequest() ManagerRuleListRequest {
	return ManagerRuleListRequest{Current: 1, PageSize: 10}
}

// ManagerOrgListRequest 经理同步机构列表请求
type ManagerOrgListRequest struct {
	Current  int    `json:"current"`
	PageSize int    `json:"pageSize"`
	Keyword  string `json:"keyword"` // 模糊匹配机构名
	Enabled  *bool  `json:"enabled"` // nil = 不过滤
}

// DefaultManagerOrgListRequest 列表请求默认值（空/解析失败时回落）
func DefaultManagerOrgListRequest() ManagerOrgListRequest {
	return ManagerOrgListRequest{Current: 1, PageSize: 10}
}
