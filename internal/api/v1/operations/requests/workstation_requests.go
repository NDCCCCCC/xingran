package requests

// WorkstationListRequest 工位列表查询请求（D-05 typed request，Phase 91-02）。
//
// 加性 JSON 契约：FloorCode/Type/OrgID 为新增可选字段，前端不传即零值/nil，
// 既有请求体零影响。
type WorkstationListRequest struct {
	PaginationParams // 嵌入分页参数
	StatusRequest    // 嵌入状态筛选
	// BuildingID 保留声明；服务从不消费（typed 化后不得开始消费，否则行为变更）。
	// 注意：floor_service 的 List 消费 buildingId，workstation 从不消费——两者不同。
	BuildingID string `json:"buildingId"` // 楼宇ID
	FloorID    string `json:"floorId"`    // 楼层ID
	Name       string `json:"name"`       // 工位名称（模糊查询）
	// Code 保留声明；服务从不消费（typed 化后不得开始消费，否则行为变更）。
	Code      string `json:"code"`      // 工位编码
	FloorCode string `json:"floorCode"` // 楼层编号（EXISTS 子查询按 floor_no 匹配）
	// Type 工位类型；nil 或 -1 → 跳过过滤（P6 -1 跳过语义）。
	Type  *int   `json:"type"`
	OrgID string `json:"orgId"` // 部门 ID（含子部门 EXISTS 筛选）
}

// WorkstationBatchOperationRequest 工位批量操作请求
type WorkstationBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
