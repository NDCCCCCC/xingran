// Package portcollection 提供端口状态采集的模块化服务
// 将原本的 port_collection_service.go (1,174行) 拆分为多个职责单一的模块
package portcollection

import (
	"github.com/xingran-next/xingran-go-backend/internal/device"
	"gorm.io/gorm"
)

// PortCollectionService 端口采集服务（主服务）
type PortCollectionService struct {
	Collection *CollectionService
	Query      *QueryService
}

// NewPortCollectionService 创建端口采集服务
func NewPortCollectionService(db *gorm.DB, executor *device.DeviceExecutor) *PortCollectionService {
	return &PortCollectionService{
		Collection: NewCollectionService(db, executor),
		Query:      NewQueryService(db),
	}
}
