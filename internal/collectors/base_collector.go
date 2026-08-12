package collectors

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// CollectorBase 采集器公共基础设施
//
// 封装所有采集器共享的字段与通用方法，通过结构体嵌入复用。
// 典型用法:
//
//	type ARPCollector struct {
//	    CollectorBase  // 嵌入，自动获得 DB / Executor 字段和 CleanOldRecords 方法
//	}
type CollectorBase struct {
	DB       *gorm.DB
	Executor *device.DeviceExecutor
}

// CleanOldRecords 删除指定天数之前的历史采集记录。
//
// 各采集器的 CleanOldRecords 方法应调用此实现，避免重复的删除逻辑。
// model 必须是 GORM 模型（如 &models.DeviceARPEntry{}），用于指定删除目标表。
//
// 	// 子结构体中的调用示例:
// 	func (c *ARPCollector) CleanOldRecords(ctx context.Context, days int) (int64, error) {
// 	    return c.CollectorBase.CleanOldRecords(ctx, &models.DeviceARPEntry{}, days)
// 	}
func (b *CollectorBase) CleanOldRecords(ctx context.Context, model interface{}, days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result := b.DB.WithContext(ctx).Where("collected_at < ?", cutoff).Delete(model)
	return result.RowsAffected, result.Error
}

// GetOnlineDevices 返回所有在线的网络设备列表。
//
// 供 CollectAllDevices 等方法复用的基础查询。
func (b *CollectorBase) GetOnlineDevices(ctx context.Context) ([]models.NetworkDevice, error) {
	var devices []models.NetworkDevice
	err := b.DB.WithContext(ctx).
		Where("status = ?", models.DeviceStatusOnline).
		Find(&devices).Error
	return devices, err
}

// CollectAllDevices 通用的"遍历所有在线设备 → 逐个采集"逻辑。
//
// 由于 Go 不允许在非泛型类型的方法上使用类型参数，此函数为包级泛型函数。
// T 是采集结果类型（如 *ARPCollectionResult）。
// collectOne 是单设备采集函数，由具体采集器提供。
//
// 	// 调用示例:
// 	func (c *ARPCollector) CollectAllDevices(ctx context.Context) ([]*ARPCollectionResult, error) {
// 	    return CollectAllDevices(ctx, c.DB, c.CollectDevice)
// 	}
func CollectAllDevices[T any](
	ctx context.Context,
	db *gorm.DB,
	collectOne func(ctx context.Context, deviceID string) (*T, error),
) ([]*T, error) {
	var devices []models.NetworkDevice
	if err := db.WithContext(ctx).
		Where("status = ?", models.DeviceStatusOnline).
		Find(&devices).Error; err != nil {
		return nil, err
	}

	results := make([]*T, 0, len(devices))
	for _, d := range devices {
		r, err := collectOne(ctx, d.ID)
		if err != nil && r == nil {
			continue
		}
		results = append(results, r)
	}
	return results, nil
}
