package requests

// WallListRequest 墙列表查询请求
type WallListRequest struct {
	PaginationParams        // 嵌入分页参数
	StatusRequest           // 嵌入状态筛选
	FloorID          string `json:"floorId"`  // 楼层ID
	WallType         string `json:"wallType"` // 墙类型
}

// WallBatchOperationRequest 墙批量操作请求
type WallBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
