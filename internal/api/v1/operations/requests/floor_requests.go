package requests

// FloorListRequest 楼层列表查询请求
type FloorListRequest struct {
	PaginationParams        // 嵌入分页参数
	StatusRequest           // 嵌入状态筛选
	BuildingID       string `json:"buildingId"` // 楼宇ID
	Name             string `json:"name"`       // 楼层名称（模糊查询）
	Code             string `json:"code"`       // 楼层编码
	FloorNo          string `json:"floorNo"`    // 楼层号
	OrgID            string `json:"orgId"`      // 所属机构ID（关联部门）
}

// FloorBatchOperationRequest 楼层批量操作请求
type FloorBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
