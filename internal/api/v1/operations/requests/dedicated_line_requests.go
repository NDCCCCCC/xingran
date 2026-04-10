package requests

// DedicatedLineListRequest 专线列表查询请求
type DedicatedLineListRequest struct {
	PaginationParams          // 嵌入分页参数
	StatusRequest             // 嵌入状态筛选
	Name               string `json:"name"`               // 专线名称
	LineType           string `json:"lineType"`           // 专线类型
	ISP                string `json:"isp"`                // 运营商
	SourceRoomId       string `json:"sourceRoomId"`       // 源机房ID
	SourceRoomName     string `json:"sourceRoomName"`     // 源机房名称
	DestRoomId         string `json:"destRoomId"`         // 目的机房ID
	DestRoomName       string `json:"destRoomName"`       // 目的机房名称
	SourceDeviceName   string `json:"sourceDeviceName"`   // 源设备名称
	DestDeviceName     string `json:"destDeviceName"`     // 目的设备名称
	CarrierContactName string `json:"carrierContactName"` // 运营商联系人
}

// DedicatedLineBatchOperationRequest 专线批量操作请求
type DedicatedLineBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
