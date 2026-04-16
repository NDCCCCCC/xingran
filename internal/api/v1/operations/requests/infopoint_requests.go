package requests

// InfoPointListRequest 信息点列表查询请求
type InfoPointListRequest struct {
	PaginationParams        // 嵌入分页参数
	StatusRequest           // 嵌入状态筛选
	FloorID          string `json:"floorId"`       // 楼层ID
	WorkID           string `json:"workId"`        // 工位ID（旧字段，保留兼容）
	WorkstationID    string `json:"workstationId"` // 工位ID（新字段，与前端一致）
	PointType        string `json:"pointType"`     // 点位类型（旧字段，保留兼容）
	InfoPointType    string `json:"infoPointType"` // 信息点类型（新字段，与前端一致）
	Name             string `json:"name"`          // 点位名称（模糊查询）
	OrgID            string `json:"orgId"`         // 部门ID（通过工位→楼层→楼宇→部门链路筛选）
}

// InfoPointBatchOperationRequest 信息点批量操作请求
type InfoPointBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
