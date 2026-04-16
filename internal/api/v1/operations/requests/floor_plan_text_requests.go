package requests

// FloorPlanTextListRequest 楼层平面图文字列表查询请求
type FloorPlanTextListRequest struct {
	PaginationParams        // 嵌入分页参数
	StatusRequest           // 嵌入状态筛选
	FloorID          string `json:"floorId"` // 楼层ID
	Content          string `json:"content"` // 文字内容（模糊查询）
}

// FloorPlanTextBatchOperationRequest 楼层平面图文字批量操作请求
type FloorPlanTextBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
