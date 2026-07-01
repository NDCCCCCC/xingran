package collectors

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/portcollection"
	"gorm.io/gorm"
)

// MACCollector MAC地址采集器
type MACCollector struct {
	db       *gorm.DB
	executor *device.DeviceExecutor
}

// MACAddressEntry MAC地址条目
type MACAddressEntry struct {
	MACAddress    string
	InterfaceName string
	VLANID        *int
	MACType       models.MACType
}

// NewMACCollector 创建MAC地址采集器
func NewMACCollector(db *gorm.DB, executor *device.DeviceExecutor) *MACCollector {
	return &MACCollector{
		db:       db,
		executor: executor,
	}
}

// CollectFromDevice 从单个设备采集MAC地址
func (c *MACCollector) CollectFromDevice(ctx context.Context, deviceID string) ([]MACAddressEntry, error) {
	// 获取设备信息
	var device models.NetworkDevice
	if err := c.db.Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	// 获取MAC地址表命令
	command := c.getMACCommand(device.Vendor)

	// 使用 executor 执行命令
	output, err := c.executor.ExecuteOnDevice(ctx, deviceID, command, true)
	if err != nil {
		return nil, fmt.Errorf("获取MAC地址表失败: %w", err)
	}

	// 解析MAC地址表
	entries, err := c.parseMACAddressTable(device.Vendor, output)
	if err != nil {
		return nil, fmt.Errorf("解析MAC地址表失败: %w", err)
	}

	return entries, nil
}

// CollectFromDevices 从多个设备采集MAC地址（并发）
func (c *MACCollector) CollectFromDevices(ctx context.Context, deviceIDs []string) map[string][]MACAddressEntry {
	results := make(map[string][]MACAddressEntry)

	for _, deviceID := range deviceIDs {
		entries, err := c.CollectFromDevice(ctx, deviceID)
		if err != nil {
			results[deviceID] = nil
			continue
		}
		results[deviceID] = entries
	}

	return results
}

// SaveToDatabase 保存MAC地址到数据库
func (c *MACCollector) SaveToDatabase(deviceID string, entries []MACAddressEntry) error {
	if len(entries) == 0 {
		return nil
	}

	collectedAt := time.Now()

	// 批量插入新数据
	for _, entry := range entries {
		mac := models.DeviceMACAddress{
			DeviceID:      deviceID,
			MACAddress:    entry.MACAddress,
			InterfaceName: entry.InterfaceName,
			VLANID:        entry.VLANID,
			MACType:       entry.MACType,
			CollectedAt:   collectedAt,
		}
		if err := c.db.Create(&mac).Error; err != nil {
			return fmt.Errorf("保存MAC地址失败: %w", err)
		}
	}

	return nil
}

// CollectAndSave 采集并保存MAC地址
func (c *MACCollector) CollectAndSave(ctx context.Context, deviceID string) error {
	entries, err := c.CollectFromDevice(ctx, deviceID)
	if err != nil {
		return err
	}

	return c.SaveToDatabase(deviceID, entries)
}

// parseMACAddressTable 解析MAC地址表
func (c *MACCollector) parseMACAddressTable(vendor models.DeviceVendor, output string) ([]MACAddressEntry, error) {
	switch vendor {
	case models.VendorHuawei, models.VendorH3C:
		return c.parseHuaweiMACAddress(output)
	case models.VendorRuijie, models.VendorMaipu:
		return c.parseCiscoLikeMACAddress(output)
	default:
		return c.parseGenericMACAddress(output)
	}
}

// parseHuaweiMACAddress 解析华为/H3C MAC地址表
// 示例输出:
// MAC address table of slot 0:
// --------------------------------------------------------------------------
// MAC Address    VLAN /       Learned/        QoS/   Matched   Port
//
//	Port        Aging time      Type           packet
//
// --------------------------------------------------------------------------
// 00e0-fc12-3456 100/-        Dynamic/No      No             GE0/0/1
func (c *MACCollector) parseHuaweiMACAddress(output string) ([]MACAddressEntry, error) {
	var entries []MACAddressEntry

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行和标题行
		if line == "" || strings.Contains(line, "---") || strings.Contains(line, "MAC Address") {
			continue
		}

		// 使用正则表达式解析MAC地址行
		// 格式: MAC VLAN/Port Type Port
		re := regexp.MustCompile(`([0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4})\s+(\d+/\S+)?\s*(\S+)?\s*(\S+)`)
		matches := re.FindStringSubmatch(line)
		if len(matches) < 5 {
			continue
		}

		macAddr := services.NormalizeMACAddress(matches[1])
		interfaceName := portcollection.NormalizeInterfaceName(matches[4])
		vlanID := c.parseVLANID(matches[2])

		// 解析MAC类型
		macType := models.MACTypeDynamic
		if strings.Contains(strings.ToLower(matches[3]), "static") {
			macType = models.MACTypeStatic
		} else if strings.Contains(strings.ToLower(matches[3]), "security") {
			macType = models.MACTypeSecure
		}

		entries = append(entries, MACAddressEntry{
			MACAddress:    macAddr,
			InterfaceName: interfaceName,
			VLANID:        vlanID,
			MACType:       macType,
		})
	}

	return entries, nil
}

// parseCiscoLikeMACAddress 解析类Cisco设备(Ruijie/Maipu)MAC地址表
// 示例输出:
// Vlan    Mac Address       Type        Ports
// ----    -----------       --------    -----
//
//	100    00e0.fc12.3456     DYNAMIC     Gi0/0/1
func (c *MACCollector) parseCiscoLikeMACAddress(output string) ([]MACAddressEntry, error) {
	var entries []MACAddressEntry

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行和标题行
		if line == "" || strings.Contains(line, "---") || strings.Contains(line, "Vlan") {
			continue
		}

		// 使用正则表达式解析MAC地址行
		re := regexp.MustCompile(`(\d+)\s+([0-9a-fA-F]{4}\.[0-9a-fA-F]{4}\.[0-9a-fA-F]{4})\s+(\S+)\s+(.+)`)
		matches := re.FindStringSubmatch(line)
		if len(matches) < 5 {
			continue
		}

		vlanID, _ := strconv.Atoi(matches[1])
		macAddr := services.NormalizeMACAddress(matches[2])
		interfaceName := portcollection.NormalizeInterfaceName(strings.TrimSpace(matches[4]))

		// 解析MAC类型
		macType := models.MACTypeDynamic
		typeUpper := strings.ToUpper(matches[3])
		switch typeUpper {
		case "STATIC":
			macType = models.MACTypeStatic
		case "SECURE", "SECURITY":
			macType = models.MACTypeSecure
		}

		entries = append(entries, MACAddressEntry{
			MACAddress:    macAddr,
			InterfaceName: interfaceName,
			VLANID:        &vlanID,
			MACType:       macType,
		})
	}

	return entries, nil
}

// parseGenericMACAddress 通用MAC地址表解析
func (c *MACCollector) parseGenericMACAddress(output string) ([]MACAddressEntry, error) {
	var entries []MACAddressEntry

	// 尝试匹配各种MAC地址格式
	macRegex := regexp.MustCompile(`([0-9a-fA-F]{2}[:-][0-9a-fA-F]{2}[:-][0-9a-fA-F]{2}[:-][0-9a-fA-F]{2}[:-][0-9a-fA-F]{2}[:-][0-9a-fA-F]{2}|[0-9a-fA-F]{4}\.[0-9a-fA-F]{4}\.[0-9a-fA-F]{4}|[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4})`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := macRegex.FindAllString(line, -1)
		for _, macMatch := range matches {
			macAddr := services.NormalizeMACAddress(macMatch)
			if isValidMACAddress(macAddr) {
				entries = append(entries, MACAddressEntry{
					MACAddress:    macAddr,
					InterfaceName: "unknown",
					MACType:       models.MACTypeDynamic,
				})
			}
		}
	}

	return entries, nil
}

// parseVLANID 解析VLAN ID
func (c *MACCollector) parseVLANID(vlanStr string) *int {
	if vlanStr == "" || vlanStr == "-" {
		return nil
	}

	parts := strings.Split(vlanStr, "/")
	if len(parts) > 0 {
		vlanID, err := strconv.Atoi(parts[0])
		if err == nil && vlanID > 0 {
			return &vlanID
		}
	}

	return nil
}

// isValidMACAddress 验证MAC地址格式
func isValidMACAddress(mac string) bool {
	_, err := net.ParseMAC(mac)
	return err == nil
}

// getMACCommand 根据厂商获取MAC地址表命令
func (c *MACCollector) getMACCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display mac-address",
		models.VendorH3C:    "display mac-address",
		models.VendorRuijie: "show mac-address-table",
		models.VendorMaipu:  "show mac-address-table",
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "display mac-address"
}

// GetMACAddressesByDevice 查询设备的MAC地址
func (c *MACCollector) GetMACAddressesByDevice(deviceID string, limit, offset int) ([]models.DeviceMACAddress, int64, error) {
	var results []models.DeviceMACAddress
	var total int64

	query := c.db.Model(&models.DeviceMACAddress{}).Where("device_id = ?", deviceID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if err := query.Order("collected_at DESC").Limit(limit).Offset(offset).Find(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// SearchMACAddress 搜索MAC地址
func (c *MACCollector) SearchMACAddress(macAddress string, limit, offset int) ([]models.DeviceMACAddress, int64, error) {
	var results []models.DeviceMACAddress
	var total int64

	// 标准化MAC地址
	normalizedMAC := services.NormalizeMACAddress(macAddress)

	query := c.db.Model(&models.DeviceMACAddress{}).Where("mac_address LIKE ?", "%"+normalizedMAC+"%")

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if err := query.Preload("Device").Order("collected_at DESC").Limit(limit).Offset(offset).Find(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// CleanOldRecords 清理旧的MAC地址记录
func (c *MACCollector) CleanOldRecords(retentionDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	return c.db.Where("collected_at < ?", cutoffDate).Delete(&models.DeviceMACAddress{}).Error
}
