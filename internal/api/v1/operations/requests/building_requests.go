package requests

// BuildingListRequest 楼宇列表查询请求
type BuildingListRequest struct {
	PaginationParams        // 嵌入分页参数
	StatusRequest           // 嵌入状态筛选
	Name             string `json:"name"`  // 楼宇名称（模糊查询）
	OrgID            string `json:"orgId"` // 所属机构ID
}

// BuildingBatchOperationRequest 楼宇批量操作请求
type BuildingBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
