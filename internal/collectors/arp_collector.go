package collectors

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"gorm.io/gorm"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// ARPCollector ARP表采集器
type ARPCollector struct {
	db      *gorm.DB
	manager *device.Manager
}

// NewARPCollector 创建ARP表采集器
func NewARPCollector(db *gorm.DB, manager *device.Manager) *ARPCollector {
	return &ARPCollector{
		db:      db,
		manager: manager,
	}
}

// ARPEntry ARP条目
type ARPEntry struct {
	IPAddress  string
	MACAddress string
	Interface  string
	Type       string // dynamic/static
	VLAN       int
}

// ARPCollectionResult ARP采集结果
type ARPCollectionResult struct {
	DeviceID    string
	DeviceName  string
	IPAddress   string
	Success     bool
	Count       int
	Error       string
	CollectedAt time.Time
}

// CollectDevice 采集单个设备的ARP表
func (c *ARPCollector) CollectDevice(ctx context.Context, deviceID string) (*ARPCollectionResult, error) {
	var device models.NetworkDevice
	if err := c.db.WithContext(ctx).Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	result := &ARPCollectionResult{
		DeviceID:    device.ID,
		DeviceName:  device.DeviceName,
		IPAddress:   device.IPAddress,
		CollectedAt: time.Now(),
	}

	// 获取设备和凭证
	_, credential, err := c.manager.GetDeviceWithCredentials(ctx, device.ID)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, fmt.Errorf("获取设备凭证失败: %w", err)
	}

	// 获取设备连接
	wrapper, err := c.manager.ConnectToDevice(ctx, &device, credential)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, fmt.Errorf("获取设备连接失败: %w", err)
	}
	defer wrapper.Close()

	// 获取ARP表
	arpEntries, err := c.collectARPTable(wrapper, &device)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, fmt.Errorf("采集ARP表失败: %w", err)
	}

	// 保存ARP条目到数据库
	for _, entry := range arpEntries {
		arpRecord := &models.DeviceARPEntry{
			DeviceID:    device.ID,
			IPAddress:   entry.IPAddress,
			MACAddress:  entry.MACAddress,
			Interface:   entry.Interface,
			Type:        entry.Type,
			VLAN:        entry.VLAN,
			CollectedAt: time.Now(),
		}
		c.db.WithContext(ctx).Create(arpRecord)
	}

	result.Success = true
	result.Count = len(arpEntries)
	return result, nil
}

// CollectAllDevices 采集所有设备的ARP表
func (c *ARPCollector) CollectAllDevices(ctx context.Context) ([]*ARPCollectionResult, error) {
	var devices []models.NetworkDevice
	c.db.WithContext(ctx).Where("status = ?", models.DeviceStatusOnline).Find(&devices)

	var results []*ARPCollectionResult

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

// collectARPTable 采集ARP表
func (c *ARPCollector) collectARPTable(wrapper *device.ScrapliWrapper, device *models.NetworkDevice) ([]ARPEntry, error) {
	cmd := c.getARPCommand(device.Vendor)
	resp, err := wrapper.SendCommand(cmd, true)
	if err != nil {
		return nil, err
	}

	arpEntries, err := c.parseARPOutput(resp.Result, device.Vendor)
	if err != nil {
		return nil, err
	}

	return arpEntries, nil
}

// getARPCommand 获取ARP表命令
func (c *ARPCollector) getARPCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display arp",
		models.VendorH3C:     "display arp",
		models.VendorRuijie:  "show arp",
		models.VendorMaipu:   "show arp",
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "display arp"
}

// parseARPOutput 解析ARP表输出
func (c *ARPCollector) parseARPOutput(output string, vendor models.DeviceVendor) ([]ARPEntry, error) {
	var entries []ARPEntry

	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过表头和分隔行
		if c.isHeaderLine(line, vendor) {
			continue
		}

		entry, ok := c.parseARPLine(line, vendor)
		if ok {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// isHeaderLine 判断是否为表头行
func (c *ARPCollector) isHeaderLine(line string, vendor models.DeviceVendor) bool {
	headerKeywords := []string{
		"IP ADDRESS", "MAC ADDRESS", "VLAN", "INTERFACE",
		"IP Address", "Hardware", "Type",
		"---", "===", "IP地址",
	}

	for _, keyword := range headerKeywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}
	return false
}

// parseARPLine 解析ARP表行
func (c *ARPCollector) parseARPLine(line string, vendor models.DeviceVendor) (ARPEntry, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return ARPEntry{}, false
	}

	switch vendor {
	case models.VendorHuawei, models.VendorH3C:
		return c.parseHuaweiARPLine(fields)
	case models.VendorRuijie, models.VendorMaipu:
		return c.parseRuijieARPLine(fields)
	default:
		return c.parseHuaweiARPLine(fields)
	}
}

// parseHuaweiARPLine 解析华为/H3C ARP行
// 格式: IP ADDRESS      MAC ADDRESS     VLAN      INTERFACE
//       192.168.1.1     00e0-fc12-3456  10        GigabitEthernet0/0/1
func (c *ARPCollector) parseHuaweiARPLine(fields []string) (ARPEntry, bool) {
	if len(fields) < 4 {
		return ARPEntry{}, false
	}

	entry := ARPEntry{
		IPAddress: fields[0],
	}

	// MAC地址可能是第二个或第三个字段
	for i := 1; i < len(fields); i++ {
		if c.isValidMAC(fields[i]) {
			entry.MACAddress = fields[i]
			// VLAN通常在MAC后面
			if i+1 < len(fields) {
				if vlan, err := parseVLAN(fields[i+1]); err == nil {
					entry.VLAN = vlan
				}
			}
			// 接口名称通常是最后一个字段
			if i+2 < len(fields) {
				entry.Interface = strings.Join(fields[i+2:], " ")
			}
			break
		}
	}

	if entry.MACAddress == "" {
		return ARPEntry{}, false
	}

	entry.Type = "dynamic"
	return entry, true
}

// parseRuijieARPLine 解析锐捷/迈普ARP行
// 格式: Protocol  Address          Hardware Address
//       Internet  192.168.1.1      00e0.fc12.3456
func (c *ARPCollector) parseRuijieARPLine(fields []string) (ARPEntry, bool) {
	if len(fields) < 3 {
		return ARPEntry{}, false
	}

	entry := ARPEntry{
		IPAddress: fields[1],
	}

	// 查找MAC地址
	for i := 2; i < len(fields); i++ {
		if c.isValidMAC(fields[i]) {
			entry.MACAddress = fields[i]
			entry.Type = "dynamic"
			return entry, true
		}
	}

	return ARPEntry{}, false
}

// isValidMAC 验证MAC地址格式
func (c *ARPCollector) isValidMAC(mac string) bool {
	// 支持格式: 00e0-fc12-3456, 00e0.fc12.3456, 00:e0:fc:12:34:56
	mac = strings.ReplaceAll(mac, "-", ":")
	mac = strings.ReplaceAll(mac, ".", ":")

	if _, err := net.ParseMAC(mac); err == nil {
		return true
	}
	return false
}

// parseVLAN 解析VLAN ID
func parseVLAN(s string) (int, error) {
	// 尝试直接解析
	var vlan int
	if _, err := fmt.Sscanf(s, "%d", &vlan); err == nil {
		return vlan, nil
	}
	return 0, fmt.Errorf("invalid VLAN")
}

// GetARPStats 获取ARP统计信息
func (c *ARPCollector) GetARPStats(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalEntries      int64
		ByDevice          map[string]int64
		ByType            map[string]int64
		CollectedRecently int64
	}

	c.db.WithContext(ctx).Model(&models.DeviceARPEntry{}).Count(&stats.TotalEntries)

	// 按设备统计
	stats.ByDevice = make(map[string]int64)
	c.db.WithContext(ctx).Model(&models.DeviceARPEntry{}).
		Select("device_id, COUNT(*) as count").
		Group("device_id").
		Scan(&stats.ByDevice)

	// 按类型统计
	stats.ByType = make(map[string]int64)
	c.db.WithContext(ctx).Model(&models.DeviceARPEntry{}).
		Select("type, COUNT(*) as count").
		Group("type").
		Scan(&stats.ByType)

	// 最近24小时采集的条目数
	since := time.Now().Add(-24 * time.Hour)
	c.db.WithContext(ctx).Model(&models.DeviceARPEntry{}).Where("collected_at > ?", since).Count(&stats.CollectedRecently)

	return map[string]interface{}{
		"totalEntries":      stats.TotalEntries,
		"byDevice":          stats.ByDevice,
		"byType":            stats.ByType,
		"collectedRecently": stats.CollectedRecently,
	}, nil
}

// CleanOldRecords 清理旧的ARP记录
func (c *ARPCollector) CleanOldRecords(ctx context.Context, days int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -days)
	result := c.db.WithContext(ctx).Where("collected_at < ?", cutoffTime).Delete(&models.DeviceARPEntry{})
	return result.RowsAffected, result.Error
}

// SearchByIP 按IP地址搜索ARP条目
func (c *ARPCollector) SearchByIP(ctx context.Context, ipAddress string) ([]models.DeviceARPEntry, error) {
	var entries []models.DeviceARPEntry
	err := c.db.WithContext(ctx).
		Where("ip_address LIKE ?", "%"+ipAddress+"%").
		Order("collected_at DESC").
		Find(&entries).Error
	return entries, err
}

// SearchByMAC 按MAC地址搜索ARP条目
func (c *ARPCollector) SearchByMAC(ctx context.Context, macAddress string) ([]models.DeviceARPEntry, error) {
	// 标准化MAC地址格式
	macAddress = strings.ReplaceAll(macAddress, "-", ":")
	macAddress = strings.ReplaceAll(macAddress, ".", ":")

	var entries []models.DeviceARPEntry
	err := c.db.WithContext(ctx).
		Where("mac_address LIKE ?", "%"+macAddress+"%").
		Order("collected_at DESC").
		Find(&entries).Error
	return entries, err
}
