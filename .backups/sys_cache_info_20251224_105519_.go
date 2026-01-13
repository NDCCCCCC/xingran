package models

import (
	"time"
)


// CacheInfo CacheInfo模型
type CacheInfo struct {
	BaseModel
	Key interface{} `column:key;type:text;primaryKey;not null json:"key"`
	Value string `column:value;type:text json:"value,omitempty"`
	Ttl int64 `column:ttl;type:bigint json:"ttl,omitempty"`
	Size int64 `column:size;type:bigint json:"size,omitempty"`
	Type interface{} `column:type;type:text json:"type,omitempty"`
}

// TableName 设置表名
func (C) TableName() string {
	return "sys_cache_info"
}
