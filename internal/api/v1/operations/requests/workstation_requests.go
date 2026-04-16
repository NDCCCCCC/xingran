package requests

// WorkstationListRequest 工位列表查询请求
type WorkstationListRequest struct {
	PaginationParams        // 嵌入分页参数
	StatusRequest           // 嵌入状态筛选
	BuildingID       string `json:"buildingId"` // 楼宇ID
	FloorID          string `json:"floorId"`    // 楼层ID
	Name             string `json:"name"`       // 工位名称（模糊查询）
	Code             string `json:"code"`       // 工位编码
}

// WorkstationBatchOperationRequest 工位批量操作请求
type WorkstationBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
