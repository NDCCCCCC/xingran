package requests

// InfoPointListRequest 信息点列表查询请求
type InfoPointListRequest struct {
	PaginationParams  // 嵌入分页参数
	StatusRequest     // 嵌入状态筛选
	FloorID    string `json:"floorId"`    // 楼层ID
	WorkID     string `json:"workId"`     // 工位ID
	PointType  string `json:"pointType"`  // 点位类型
	Name       string `json:"name"`       // 点位名称（模糊查询）
}

// InfoPointBatchOperationRequest 信息点批量操作请求
type InfoPointBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
