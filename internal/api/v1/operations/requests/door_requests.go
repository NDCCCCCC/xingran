package requests

// DoorListRequest 门列表查询请求
type DoorListRequest struct {
	PaginationParams        // 嵌入分页参数
	StatusRequest           // 嵌入状态筛选
	FloorID          string `json:"floorId"`  // 楼层ID
	DoorType         string `json:"doorType"` // 门类型
}

// DoorBatchOperationRequest 门批量操作请求
type DoorBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
