package requests

// ColumnConfigSaveRequest 保存列配置请求
type ColumnConfigSaveRequest struct {
	PageKey       string             `json:"pageKey" binding:"required"`
	ColumnConfigs []ColumnConfigItem `json:"columnConfigs" binding:"required"`
}

// ColumnConfigItem 列配置项
type ColumnConfigItem struct {
	ColumnKey string `json:"columnKey" binding:"required"`
	Visible   bool   `json:"visible"`
	Width     int    `json:"width"`
}
