package collectors

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// AssetCollector 资产信息采集器
type AssetCollector struct {
	db       *gorm.DB
	executor *device.DeviceExecutor
}

// NewAssetCollector 创建资产信息采集器
func NewAssetCollector(db *gorm.DB, executor *device.DeviceExecutor) *AssetCollector {
	return &AssetCollector{
		db:       db,
		executor: executor,
	}
}

// AssetInfo 资产信息
type AssetInfo struct {
	SerialNumber      string
	HardwareVersion   string
	FirmwareVersion   string
	SoftwareVersion   string
	BootROMVersion    string
	SystemDescription string
	ProductName       string
	DeviceModel       string
	Uptime            int64 // 秒
	TotalMemory       int64 // KB
	FreeMemory        int64 // KB
	CPUUsage          float64
}

// AssetCollectionResult 资产采集结果
type AssetCollectionResult struct {
	DeviceID    string
	DeviceName  string
	IPAddress   string
	Success     bool
	Asset       *AssetInfo
	Error       string
	CollectedAt time.Time
}

// CollectDevice 采集单个设备的资产信息
func (c *AssetCollector) CollectDevice(ctx context.Context, deviceID string) (*AssetCollectionResult, error) {
	var device models.NetworkDevice
	if err := c.db.WithContext(ctx).Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	result := &AssetCollectionResult{
		DeviceID:    device.ID,
		DeviceName:  device.DeviceName,
		IPAddress:   device.IPAddress,
		CollectedAt: time.Now(),
	}

	assetInfo := &AssetInfo{}
	vendor := device.Vendor

	// 直接使用连接池获取连接
	pool := c.executor.GetScheduler().GetConnectionPool()
	conn, err := pool.GetConnection(ctx, deviceID)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, fmt.Errorf("获取设备连接失败: %w", err)
	}

	// F-14 Phase 31: GetConnection 内部已 refCount +1,直接 defer ReleaseRef。
	defer conn.ReleaseRef()

	wrapper := conn.GetWrapper()

	// 获取版本信息（包含序列号、版本等）
	versionCmd := c.getVersionCommand(vendor)
	versionResp, err := wrapper.SendCommand(versionCmd, true)
	if err == nil {
		c.parseVersionInfo(versionResp.Result, vendor, assetInfo)
	}

	// 获取系统信息（内存、CPU等）
	deviceCmd := c.getDeviceCommand(vendor)
	deviceResp, err := wrapper.SendCommand(deviceCmd, true)
	if err == nil {
		c.parseDeviceInfo(deviceResp.Result, vendor, assetInfo)
	}

	// 保存资产信息到数据库
	assetRecord := &models.DeviceAsset{
		DeviceID:          device.ID,
		SerialNumber:      assetInfo.SerialNumber,
		HardwareVersion:   assetInfo.HardwareVersion,
		FirmwareVersion:   assetInfo.FirmwareVersion,
		SoftwareVersion:   assetInfo.SoftwareVersion,
		BootROMVersion:    assetInfo.BootROMVersion,
		SystemDescription: assetInfo.SystemDescription,
		ProductName:       assetInfo.ProductName,
		DeviceModel:       assetInfo.DeviceModel,
		Uptime:            assetInfo.Uptime,
		TotalMemory:       assetInfo.TotalMemory,
		FreeMemory:        assetInfo.FreeMemory,
		CPUUsage:          assetInfo.CPUUsage,
		CollectedAt:       time.Now(),
	}
	c.db.WithContext(ctx).Create(assetRecord)

	result.Success = true
	result.Asset = assetInfo
	return result, nil
}

// CollectAllDevices 采集所有设备的资产信息
func (c *AssetCollector) CollectAllDevices(ctx context.Context) ([]*AssetCollectionResult, error) {
	var devices []models.NetworkDevice
	c.db.WithContext(ctx).Where("status = ?", models.DeviceStatusOnline).Find(&devices)

	var results []*AssetCollectionResult

	for _, device := range devices {
		result, err := c.CollectDevice(ctx, device.ID)
		if err != nil {
			results = append(results, result)
		} else {
			results = append(results, result)
		}
	}

	return results, nil
}

// getVersionCommand 获取版本信息命令
func (c *AssetCollector) getVersionCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display version",
		models.VendorH3C:    "display version",
		models.VendorRuijie: "show version",
		models.VendorMaipu:  "show version",
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "display version"
}

// getDeviceCommand 获取设备信息命令
func (c *AssetCollector) getDeviceCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display device",
		models.VendorH3C:    "display device",
		models.VendorRuijie: "show system",
		models.VendorMaipu:  "show system",
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "display device"
}

// parseVersionInfo 解析版本信息
func (c *AssetCollector) parseVersionInfo(output string, vendor models.DeviceVendor, asset *AssetInfo) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		switch vendor {
		case models.VendorHuawei, models.VendorH3C:
			c.parseHuaweiVersionLine(line, asset)
		case models.VendorRuijie, models.VendorMaipu:
			c.parseRuijieVersionLine(line, asset)
		}
	}
}

// parseHuaweiVersionLine 解析华为/H3C版本信息行
func (c *AssetCollector) parseHuaweiVersionLine(line string, asset *AssetInfo) {
	// 序列号: H3C Serial Number
	if strings.Contains(line, "Serial Number") || strings.Contains(line, "DEVICE_SERIAL_NUMBER") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			asset.SerialNumber = strings.TrimSpace(parts[1])
		}
	}

	// 硬件版本
	if strings.Contains(line, "Hardware Version") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			asset.HardwareVersion = strings.TrimSpace(parts[1])
		}
	}

	// 软件版本
	if strings.Contains(line, "Software Version") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			asset.SoftwareVersion = strings.TrimSpace(parts[1])
		}
	}

	// BootROM版本
	if strings.Contains(line, "BootROM Version") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			asset.BootROMVersion = strings.TrimSpace(parts[1])
		}
	}

	// 产品名称
	if strings.Contains(line, "H3C") || strings.Contains(line, "Huawei") {
		if strings.Contains(line, "S") && strings.Contains(line, "Switch") {
			asset.ProductName = line
		}
	}

	// 系统描述
	if strings.Contains(line, "System Description") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			asset.SystemDescription = strings.TrimSpace(parts[1])
		}
	}

	// 运行时间
	if strings.Contains(line, "uptime") || strings.Contains(line, "Uptime") {
		asset.Uptime = c.parseUptime(line)
	}
}

// parseRuijieVersionLine 解析锐捷/迈普版本信息行
func (c *AssetCollector) parseRuijieVersionLine(line string, asset *AssetInfo) {
	// 序列号
	if strings.Contains(line, "Serial Number") || strings.Contains(line, "System Serial") {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			asset.SerialNumber = strings.Join(parts[2:], " ")
		}
	}

	// 硬件版本
	if strings.Contains(line, "Hardware Version") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			asset.HardwareVersion = strings.TrimSpace(parts[1])
		}
	}

	// 软件版本
	if strings.Contains(line, "Software Version") || strings.Contains(line, "System Software") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			asset.SoftwareVersion = strings.TrimSpace(parts[1])
		}
	}

	// 系统描述
	if strings.Contains(line, "System Description") || strings.Contains(line, "System Type") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			asset.SystemDescription = strings.TrimSpace(parts[1])
		}
	}

	// 运行时间
	if strings.Contains(line, "uptime") || strings.Contains(line, "Uptime") {
		asset.Uptime = c.parseUptime(line)
	}
}

// parseDeviceInfo 解析设备信息（内存、CPU）
func (c *AssetCollector) parseDeviceInfo(output string, vendor models.DeviceVendor, asset *AssetInfo) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		switch vendor {
		case models.VendorHuawei, models.VendorH3C:
			c.parseHuaweiDeviceLine(line, asset)
		case models.VendorRuijie, models.VendorMaipu:
			c.parseRuijieDeviceLine(line, asset)
		}
	}
}

// parseHuaweiDeviceLine 解析华为/H3C设备信息行
func (c *AssetCollector) parseHuaweiDeviceLine(line string, asset *AssetInfo) {
	// 内存信息
	if strings.Contains(line, "Memory") || strings.Contains(line, "memory") {
		if strings.Contains(line, "Total") {
			re := regexp.MustCompile(`(\d+)\s*(KB|MB|GB)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 2 {
				if mem, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
					asset.TotalMemory = c.convertToKB(mem, matches[2])
				}
			}
		}
		if strings.Contains(line, "Free") {
			re := regexp.MustCompile(`(\d+)\s*(KB|MB|GB)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 2 {
				if mem, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
					asset.FreeMemory = c.convertToKB(mem, matches[2])
				}
			}
		}
	}

	// CPU使用率
	if strings.Contains(line, "CPU") || strings.Contains(line, "cpu") {
		if strings.Contains(line, "Usage") || strings.Contains(line, "utilization") {
			re := regexp.MustCompile(`(\d+)%`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 2 {
				if usage, err := strconv.ParseFloat(matches[1], 64); err == nil {
					asset.CPUUsage = usage
				}
			}
		}
	}
}

// parseRuijieDeviceLine 解析锐捷/迈普设备信息行
func (c *AssetCollector) parseRuijieDeviceLine(line string, asset *AssetInfo) {
	// 内存信息
	if strings.Contains(line, "Memory") || strings.Contains(line, "memory") {
		re := regexp.MustCompile(`(\d+)\s*(KB|MB|GB)`)
		matches := re.FindAllStringSubmatch(line, -1)
		if len(matches) >= 2 {
			if total, err := strconv.ParseInt(matches[0][1], 10, 64); err == nil {
				asset.TotalMemory = c.convertToKB(total, matches[0][2])
			}
			if free, err := strconv.ParseInt(matches[1][1], 10, 64); err == nil {
				asset.FreeMemory = c.convertToKB(free, matches[1][2])
			}
		}
	}

	// CPU使用率
	if strings.Contains(line, "CPU") || strings.Contains(line, "cpu") {
		re := regexp.MustCompile(`(\d+)%`)
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			if usage, err := strconv.ParseFloat(matches[1], 64); err == nil {
				asset.CPUUsage = usage
			}
		}
	}
}

// parseUptime 解析运行时间（秒）
func (c *AssetCollector) parseUptime(line string) int64 {
	// 尝试提取天数、小时、分钟、秒
	dayRe := regexp.MustCompile(`(\d+)\s*days?`)
	hourRe := regexp.MustCompile(`(\d+)\s*hours?`)
	minRe := regexp.MustCompile(`(\d+)\s*minutes?`)
	secRe := regexp.MustCompile(`(\d+)\s*seconds?`)

	var seconds int64

	if matches := dayRe.FindStringSubmatch(line); len(matches) >= 2 {
		if days, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			seconds += days * 86400
		}
	}

	if matches := hourRe.FindStringSubmatch(line); len(matches) >= 2 {
		if hours, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			seconds += hours * 3600
		}
	}

	if matches := minRe.FindStringSubmatch(line); len(matches) >= 2 {
		if mins, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			seconds += mins * 60
		}
	}

	if matches := secRe.FindStringSubmatch(line); len(matches) >= 2 {
		if secs, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			seconds += secs
		}
	}

	return seconds
}

// convertToKB 转换为KB
func (c *AssetCollector) convertToKB(value int64, unit string) int64 {
	switch strings.ToUpper(unit) {
	case "GB":
		return value * 1024 * 1024
	case "MB":
		return value * 1024
	case "KB":
		return value
	default:
		return value
	}
}

// GetAssetStats 获取资产统计信息
func (c *AssetCollector) GetAssetStats(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalDevices      int64
		BySoftwareVersion map[string]int64
		ByProduct         map[string]int64
		AvgCPUUsage       float64
		AvgMemoryUsage    float64
	}

	c.db.WithContext(ctx).Model(&models.DeviceAsset{}).Count(&stats.TotalDevices)

	// 按软件版本统计
	stats.BySoftwareVersion = make(map[string]int64)
	c.db.WithContext(ctx).Model(&models.DeviceAsset{}).
		Select("software_version, COUNT(*) as count").
		Group("software_version").
		Scan(&stats.BySoftwareVersion)

	// 按产品型号统计
	stats.ByProduct = make(map[string]int64)
	c.db.WithContext(ctx).Model(&models.DeviceAsset{}).
		Select("product_name, COUNT(*) as count").
		Group("product_name").
		Scan(&stats.ByProduct)

	// 平均CPU使用率
	var avgCPU, avgMem float64
	c.db.WithContext(ctx).Model(&models.DeviceAsset{}).
		Select("AVG(cpu_usage)").
		Scan(&avgCPU)
	stats.AvgCPUUsage = avgCPU

	// 平均内存使用率
	c.db.WithContext(ctx).Model(&models.DeviceAsset{}).
		Where("total_memory > 0").
		Select("AVG((total_memory - free_memory) * 100.0 / total_memory)").
		Scan(&avgMem)
	stats.AvgMemoryUsage = avgMem

	return map[string]interface{}{
		"totalDevices":      stats.TotalDevices,
		"bySoftwareVersion": stats.BySoftwareVersion,
		"byProduct":         stats.ByProduct,
		"avgCPUUsage":       stats.AvgCPUUsage,
		"avgMemoryUsage":    stats.AvgMemoryUsage,
	}, nil
}

// CleanOldRecords 清理旧的资产记录
func (c *AssetCollector) CleanOldRecords(ctx context.Context, days int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -days)
	result := c.db.WithContext(ctx).Where("collected_at < ?", cutoffTime).Delete(&models.DeviceAsset{})
	return result.RowsAffected, result.Error
}

// GetLatestAsset 获取设备最新资产信息
func (c *AssetCollector) GetLatestAsset(ctx context.Context, deviceID string) (*models.DeviceAsset, error) {
	var asset models.DeviceAsset
	err := c.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Order("collected_at DESC").
		First(&asset).Error
	return &asset, err
}
