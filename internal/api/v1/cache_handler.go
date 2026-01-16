package v1

// ==================== 请求/响应类型 ====================

// CacheListRequest 缓存列表请求
type CacheListRequest struct {
	Key      string `json:"key,omitempty"`
	Type     string `json:"type,omitempty"`
	Level    string `json:"level,omitempty"` // 新增: "l1", "l2", "all"
	Current  int    `json:"current"`
	PageSize int    `json:"pageSize"`
}

// CacheStatsRequest 缓存统计请求
type CacheStatsRequest struct {
	CacheType  string  `json:"cacheType,omitempty"`
	StartTime  *string `json:"startTime,omitempty"`
	EndTime    *string `json:"endTime,omitempty"`
	IsRealtime bool    `json:"isRealtime,omitempty"` // true=实时统计, false=历史统计
	Current    int     `json:"current"`
	PageSize   int     `json:"pageSize"`
}

// CacheOperateRequest 缓存操作请求
type CacheOperateRequest struct {
	Key       string `json:"key" binding:"required"`
	Value     string `json:"value,omitempty"`
	TTL       int64  `json:"ttl,omitempty"`
	Operation string `json:"operation" binding:"required,oneof=get set del exists expire ttl"`
}

// CacheBatchRequest 批量缓存操作请求
type CacheBatchRequest struct {
	Keys      []string `json:"keys" binding:"required"`
	Operation string   `json:"operation" binding:"required,oneof=get del"`
}
