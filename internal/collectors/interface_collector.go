package collectors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"gorm.io/gorm"
)

// InterfaceCollector 接口信息采集器
type InterfaceCollector struct {
	CollectorBase
}

// NewInterfaceCollector 创建接口信息采集器
func NewInterfaceCollector(db *gorm.DB, executor *device.DeviceExecutor) *InterfaceCollector {
	return &InterfaceCollector{
		CollectorBase: CollectorBase{
			DB:       db,
			Executor: executor,
		},
	}
}

// InterfaceInfo 接口信息
type InterfaceInfo struct {
	Name         string
	AdminStatus  string
	OperStatus   string
	Description  string
	MAC          string
	Mtu          int64
	Bandwidth    int64
	InputBytes   int64
	OutputBytes  int64
	InputErrors  int64
	OutputErrors int64
}

// InterfaceCollectionResult 接口采集结果
type InterfaceCollectionResult struct {
	DeviceID    string
	DeviceName  string
	IPAddress   string
	Success     bool
	Count       int
	Error       string
	CollectedAt time.Time
}

// CollectDevice 采集单个设备的接口信息
func (c *InterfaceCollector) CollectDevice(ctx context.Context, deviceID string) (*InterfaceCollectionResult, error) {
	var device models.NetworkDevice
	if err := c.DB.WithContext(ctx).Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	result := &InterfaceCollectionResult{
		DeviceID:    device.ID,
		DeviceName:  device.DeviceName,
		IPAddress:   device.IPAddress,
		CollectedAt: time.Now(),
	}

	// 获取接口信息命令
	cmd := c.getInterfaceCommand(device.Vendor)

	// 使用 executor 执行命令
	output, err := c.Executor.ExecuteOnDevice(ctx, device.ID, cmd, true)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, fmt.Errorf("采集接口信息失败: %w", err)
	}

	// 解析接口信息
	interfaces, err := c.parseInterfaceOutput(output, device.Vendor)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, fmt.Errorf("解析接口信息失败: %w", err)
	}

	// 保存接口信息到数据库
	for _, iface := range interfaces {
		interfaceRecord := &models.DeviceInterface{
			DeviceID:      device.ID,
			InterfaceName: iface.Name,
			AdminStatus:   iface.AdminStatus,
			OperStatus:    iface.OperStatus,
			Description:   iface.Description,
			MAC:           iface.MAC,
			Mtu:           iface.Mtu,
			Bandwidth:     iface.Bandwidth,
			InputBytes:    iface.InputBytes,
			OutputBytes:   iface.OutputBytes,
			InputErrors:   iface.InputErrors,
			OutputErrors:  iface.OutputErrors,
			CollectedAt:   time.Now(),
		}
		c.DB.WithContext(ctx).Create(interfaceRecord)
	}

	result.Success = true
	result.Count = len(interfaces)
	return result, nil
}

// CollectAllDevices 采集所有设备的接口信息
func (c *InterfaceCollector) CollectAllDevices(ctx context.Context) ([]*InterfaceCollectionResult, error) {
	return CollectAllDevices(ctx, c.DB, c.CollectDevice)
}

// getInterfaceCommand 获取接口信息命令
func (c *InterfaceCollector) getInterfaceCommand(vendor models.DeviceVendor) string {
	return vendorCommand(vendor, "display interface", "show interface", "display interface")
}

// parseInterfaceOutput 解析接口信息输出
func (c *InterfaceCollector) parseInterfaceOutput(output string, vendor models.DeviceVendor) ([]InterfaceInfo, error) {
	var interfaces []InterfaceInfo

	lines := strings.Split(output, "\n")
	var currentInterface *InterfaceInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 根据不同厂商解析接口信息
		switch vendor {
		case models.VendorHuawei, models.VendorH3C:
			c.parseHuaweiInterfaceLine(line, &currentInterface, &interfaces)
		case models.VendorRuijie, models.VendorMaipu:
			c.parseRuijieInterfaceLine(line, &currentInterface, &interfaces)
		}
	}

	// 添加最后一个接口
	if currentInterface != nil && currentInterface.Name != "" {
		interfaces = append(interfaces, *currentInterface)
	}

	return interfaces, nil
}

// parseHuaweiInterfaceLine 解析华为/H3C接口行
func (c *InterfaceCollector) parseHuaweiInterfaceLine(line string, currentInterface **InterfaceInfo, interfaces *[]InterfaceInfo) {
	// 接口名称行: GigabitEthernet0/0/1
	if strings.Contains(line, "Line protocol") || strings.Contains(line, "current state") {
		parts := strings.Fields(line)
		if len(parts) > 0 {
			if *currentInterface != nil && (*currentInterface).Name != "" {
				*interfaces = append(*interfaces, **currentInterface)
			}
			// 2026-07-01 port-mac-format-unify: 写入前归一化
			name := portcollection.NormalizeInterfaceName(strings.TrimSuffix(parts[0], " "))
			*currentInterface = &InterfaceInfo{Name: name}
		}
	}

	if *currentInterface == nil {
		return
	}

	// 管理状态
	if strings.Contains(line, "Line protocol") {
		if strings.Contains(line, "up") {
			(*currentInterface).AdminStatus = "up"
		} else {
			(*currentInterface).AdminStatus = "down"
		}
	}

	// 操作状态
	if strings.Contains(line, "Physical mode") {
		if strings.Contains(line, "up") {
			(*currentInterface).OperStatus = "up"
		} else {
			(*currentInterface).OperStatus = "down"
		}
	}

	// 描述
	if strings.HasPrefix(line, "Description:") {
		(*currentInterface).Description = strings.TrimPrefix(line, "Description:")
	}

	// MAC地址
	if strings.Contains(line, "Hardware address") {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			// 2026-07-01 port-mac-format-unify: MAC 大写+冒号
			(*currentInterface).MAC = services.NormalizeMACAddress(parts[2])
		}
	}
}

// parseRuijieInterfaceLine 解析锐捷/迈普接口行
func (c *InterfaceCollector) parseRuijieInterfaceLine(line string, currentInterface **InterfaceInfo, interfaces *[]InterfaceInfo) {
	// 接口名称行: interface GigabitEthernet 0/1
	if strings.HasPrefix(line, "interface") {
		if *currentInterface != nil && (*currentInterface).Name != "" {
			*interfaces = append(*interfaces, **currentInterface)
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			// 2026-07-01 port-mac-format-unify: 写入前归一化
			name := portcollection.NormalizeInterfaceName(strings.Join(parts[1:], " "))
			*currentInterface = &InterfaceInfo{Name: name}
		}
	}

	if *currentInterface == nil {
		return
	}

	// 状态信息
	if strings.Contains(line, "Status:") {
		if strings.Contains(line, "up") {
			(*currentInterface).OperStatus = "up"
		} else {
			(*currentInterface).OperStatus = "down"
		}
	}

	// 描述
	if strings.HasPrefix(line, "Description:") {
		(*currentInterface).Description = strings.TrimPrefix(line, "Description:")
	}

	// MAC地址
	if strings.Contains(line, "Hardware address") || strings.Contains(line, "MAC") {
		parts := strings.Fields(line)
		for i, part := range parts {
			if strings.Contains(part, ":") && len(part) == 17 {
				// 2026-07-01 port-mac-format-unify: 大写+冒号
				(*currentInterface).MAC = services.NormalizeMACAddress(part)
				break
			}
			if i > 0 && strings.Contains(parts[i-1], "address") {
				// 2026-07-01 port-mac-format-unify: 大写+冒号
				(*currentInterface).MAC = services.NormalizeMACAddress(part)
			}
		}
	}
}

// GetInterfaceStats 获取接口统计信息
func (c *InterfaceCollector) GetInterfaceStats(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalInterfaces   int64
		UpInterfaces      int64
		DownInterfaces    int64
		ByDevice          map[string]int64
		CollectedRecently int64
	}

	c.DB.WithContext(ctx).Model(&models.DeviceInterface{}).Count(&stats.TotalInterfaces)
	c.DB.WithContext(ctx).Model(&models.DeviceInterface{}).Where("oper_status = ?", "up").Count(&stats.UpInterfaces)
	c.DB.WithContext(ctx).Model(&models.DeviceInterface{}).Where("oper_status = ?", "down").Count(&stats.DownInterfaces)

	// 最近24小时采集的接口数量
	since := time.Now().Add(-24 * time.Hour)
	c.DB.WithContext(ctx).Model(&models.DeviceInterface{}).Where("collected_at > ?", since).Count(&stats.CollectedRecently)

	// 按设备统计
	stats.ByDevice = make(map[string]int64)
	c.DB.WithContext(ctx).Model(&models.DeviceInterface{}).
		Select("device_id, COUNT(*) as count").
		Group("device_id").
		Scan(&stats.ByDevice)

	return map[string]interface{}{
		"totalInterfaces":   stats.TotalInterfaces,
		"upInterfaces":      stats.UpInterfaces,
		"downInterfaces":    stats.DownInterfaces,
		"byDevice":          stats.ByDevice,
		"collectedRecently": stats.CollectedRecently,
	}, nil
}

// CleanOldRecords 清理旧的接口记录
func (c *InterfaceCollector) CleanOldRecords(ctx context.Context, days int) (int64, error) {
	return c.CollectorBase.CleanOldRecords(ctx, &models.DeviceInterface{}, days)
}
