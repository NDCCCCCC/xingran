package requests

// RoomDeviceListRequest 机房设备列表查询请求
type RoomDeviceListRequest struct {
	PaginationParams        // 嵌入分页参数
	StatusRequest           // 嵌入状态筛选
	OrgID            string `json:"orgId"`      // 部门ID（通过机房关联）
	RoomID           string `json:"roomId"`     // 机房ID
	Name             string `json:"name"`       // 设备名称（模糊查询）
	DeviceType       string `json:"deviceType"` // 设备类型
	IPAddress        string `json:"ipAddress"`  // IP地址
}

// RoomDeviceBatchOperationRequest 机房设备批量操作请求
type RoomDeviceBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
