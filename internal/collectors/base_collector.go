package collectors

import (
	"context"
	"log"
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
// 各采集器的同名 CleanOldRecords 方法必须显式通过 c.CollectorBase.CleanOldRecords
// 委托到此实现，原因不是简单的"复用删除逻辑"，而是：子结构体已通过嵌入获得
// CollectorBase 的同名方法，若直接调用 c.CleanOldRecords(...) 会触发自身递归
// （Go 嵌入遮蔽），必须用限定路径绕过。
// model 必须是 GORM 模型（如 &models.DeviceARPEntry{}），用于指定删除目标表。
//
// 	// 子结构体中的调用示例:
// 	func (c *ARPCollector) CleanOldRecords(ctx context.Context, days int) (int64, error) {
// 	    return c.CollectorBase.CleanOldRecords(ctx, &models.DeviceARPEntry{}, days)
// 	}
func (b *CollectorBase) CleanOldRecords(ctx context.Context, model any, days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result := b.DB.WithContext(ctx).Where("collected_at < ?", cutoff).Delete(model)
	return result.RowsAffected, result.Error
}

// vendorCommand 根据设备厂商返回对应的采集命令字串。
//
// 华为/H3C 使用 display 前缀命令，锐捷/迈普使用 show 前缀命令，
// 未知厂商使用 fallback。display / show / fallback 三者由调用方按业务传入。
//
// 	// 调用示例:
// 	cmd := vendorCommand(device.Vendor, "display arp", "show arp", "display arp")
func vendorCommand(vendor models.DeviceVendor, display, show, fallback string) string {
	switch vendor {
	case models.VendorHuawei, models.VendorH3C:
		return display
	case models.VendorRuijie, models.VendorMaipu:
		return show
	default:
		return fallback
	}
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
			log.Printf("[CollectAllDevices] 跳过设备: deviceID=%s err=%v", d.ID, err)
			continue
		}
		results = append(results, r)
	}
	return results, nil
}
