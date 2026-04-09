package requests

// ServerRoomListRequest 机房列表查询请求
type ServerRoomListRequest struct {
	PaginationParams        // 嵌入分页参数
	StatusRequest           // 嵌入状态筛选
	Name             string `json:"name"`       // 机房名称（模糊查询）
	BuildingID       string `json:"buildingId"` // 楼宇ID
	FloorID          string `json:"floorId"`    // 楼层ID
	OrgID            string `json:"orgId"`      // 所属部门ID（支持子部门）
}

// ServerRoomBatchOperationRequest 机房批量操作请求
type ServerRoomBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
