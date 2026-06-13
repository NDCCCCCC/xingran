package collectors

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// PortCollector 端口状态采集器
type PortCollector struct {
	db       *gorm.DB
	executor *device.DeviceExecutor
}

// PortStatusEntry 端口状态条目
type PortStatusEntry struct {
	InterfaceName       string
	AdminStatus         string
	OperStatus          string
	Dot1xEnabled        bool
	Dot1xPortStatus     models.Dot1xPortStatus
	PortSecurityEnabled bool
	PortSecurityMode    models.PortSecurityMode
	MaxMACCount         *int
	CurrentMACCount     *int
}

// NewPortCollector 创建端口状态采集器
func NewPortCollector(db *gorm.DB, executor *device.DeviceExecutor) *PortCollector {
	return &PortCollector{
		db:       db,
		executor: executor,
	}
}

// CollectFromDevice 从单个设备采集端口状态
func (c *PortCollector) CollectFromDevice(ctx context.Context, deviceID string) ([]PortStatusEntry, error) {
	// 获取设备信息
	var device models.NetworkDevice
	if err := c.db.Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	// 直接使用连接池获取连接
	pool := c.executor.GetScheduler().GetConnectionPool()
	conn, err := pool.GetConnection(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("获取设备连接失败: %w", err)
	}

	// F-14 Phase 31: GetConnection 内部已 refCount +1,直接 defer Release。
	defer conn.Release()

	wrapper := conn.GetWrapper()

	// 采集接口状态
	interfaceStatus, err := c.collectInterfaceStatus(device, wrapper)
	if err != nil {
		return nil, fmt.Errorf("采集接口状态失败: %w", err)
	}

	// 采集802.1X状态
	dot1xStatus, err := c.collectDot1xStatus(device, wrapper)
	if err != nil {
		// 802.1X采集失败不影响整体采集，记录日志继续
		dot1xStatus = make(map[string]PortStatusEntry)
	}

	// 采集端口安全配置
	securityStatus, err := c.collectPortSecurity(device, wrapper)
	if err != nil {
		// 端口安全采集失败不影响整体采集，记录日志继续
		securityStatus = make(map[string]PortStatusEntry)
	}

	// 合并所有状态信息
	entries := c.mergePortStatus(interfaceStatus, dot1xStatus, securityStatus)

	return entries, nil
}

// collectInterfaceStatus 采集接口状态
func (c *PortCollector) collectInterfaceStatus(device models.NetworkDevice, wrapper *device.ScrapliWrapper) (map[string]PortStatusEntry, error) {
	command := c.getInterfaceCommand(device.Vendor)
	response, err := wrapper.SendCommand(command, true)
	if err != nil {
		return nil, err
	}

	return c.parseInterfaceStatus(device.Vendor, response.Result)
}

// collectDot1xStatus 采集802.1X状态
func (c *PortCollector) collectDot1xStatus(device models.NetworkDevice, wrapper *device.ScrapliWrapper) (map[string]PortStatusEntry, error) {
	command := c.getDot1xCommand(device.Vendor)
	response, err := wrapper.SendCommand(command, true)
	if err != nil {
		return nil, err
	}

	return c.parseDot1xStatus(device.Vendor, response.Result)
}

// collectPortSecurity 采集端口安全配置
func (c *PortCollector) collectPortSecurity(device models.NetworkDevice, wrapper *device.ScrapliWrapper) (map[string]PortStatusEntry, error) {
	command := c.getPortSecurityCommand(device.Vendor)
	response, err := wrapper.SendCommand(command, true)
	if err != nil {
		return nil, err
	}

	return c.parsePortSecurity(device.Vendor, response.Result)
}

// mergePortStatus 合并端口状态信息
func (c *PortCollector) mergePortStatus(interfaceStatus, dot1xStatus, securityStatus map[string]PortStatusEntry) []PortStatusEntry {
	// 使用接口状态作为基础
	result := make(map[string]PortStatusEntry)

	for iface, status := range interfaceStatus {
		result[iface] = status
	}

	// 合并802.1X状态
	for iface, status := range dot1xStatus {
		if existing, ok := result[iface]; ok {
			existing.Dot1xEnabled = status.Dot1xEnabled
			existing.Dot1xPortStatus = status.Dot1xPortStatus
			result[iface] = existing
		} else {
			result[iface] = status
		}
	}

	// 合并端口安全状态
	for iface, status := range securityStatus {
		if existing, ok := result[iface]; ok {
			existing.PortSecurityEnabled = status.PortSecurityEnabled
			existing.PortSecurityMode = status.PortSecurityMode
			existing.MaxMACCount = status.MaxMACCount
			existing.CurrentMACCount = status.CurrentMACCount
			result[iface] = existing
		} else {
			result[iface] = status
		}
	}

	// 转换为切片
	var entries []PortStatusEntry
	for _, status := range result {
		entries = append(entries, status)
	}

	return entries
}

// parseInterfaceStatus 解析接口状态
func (c *PortCollector) parseInterfaceStatus(vendor models.DeviceVendor, output string) (map[string]PortStatusEntry, error) {
	result := make(map[string]PortStatusEntry)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 根据不同厂商解析接口状态
		switch vendor {
		case models.VendorHuawei, models.VendorH3C:
			c.parseHuaweiInterfaceLine(line, result)
		case models.VendorRuijie, models.VendorMaipu:
			c.parseCiscoInterfaceLine(line, result)
		}
	}

	return result, nil
}

// parseHuaweiInterfaceLine 解析华为/H3C接口状态行
// 示例: GigabitEthernet0/0/1                up            up
func (c *PortCollector) parseHuaweiInterfaceLine(line string, result map[string]PortStatusEntry) {
	// 使用正则匹配接口行: 接口名 管理状态 操作状态
	re := regexp.MustCompile(`^(\S+(?:\d+/\d+/\d+|\d+/\d+|\d+))\s+(\w+)\s+(\w+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 4 {
		return
	}

	interfaceName := matches[1]
	adminStatus := strings.ToLower(matches[2])
	operStatus := strings.ToLower(matches[3])

	// 转换状态
	adminStatusStr := "down"
	if adminStatus == "up" || adminStatus == "enabled" {
		adminStatusStr = "up"
	}

	operStatusStr := "down"
	if operStatus == "up" {
		operStatusStr = "up"
	}

	result[interfaceName] = PortStatusEntry{
		InterfaceName: interfaceName,
		AdminStatus:   adminStatusStr,
		OperStatus:    operStatusStr,
	}
}

// parseCiscoInterfaceLine 解析Cisco类接口状态行
// 示例: GigabitEthernet0/0/1   up   up
func (c *PortCollector) parseCiscoInterfaceLine(line string, result map[string]PortStatusEntry) {
	// 类似华为的解析方式
	c.parseHuaweiInterfaceLine(line, result)
}

// parseDot1xStatus 解析802.1X状态
func (c *PortCollector) parseDot1xStatus(vendor models.DeviceVendor, output string) (map[string]PortStatusEntry, error) {
	result := make(map[string]PortStatusEntry)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "802.1X") || strings.Contains(line, "---") {
			continue
		}

		// 根据厂商解析802.1X状态
		switch vendor {
		case models.VendorHuawei, models.VendorH3C:
			c.parseHuaweiDot1xLine(line, result)
		case models.VendorRuijie, models.VendorMaipu:
			c.parseCiscoDot1xLine(line, result)
		}
	}

	return result, nil
}

// parseHuaweiDot1xLine 解析华为/H3C 802.1X状态行
// 示例: GigabitEthernet0/0/1      enabled     authorized
func (c *PortCollector) parseHuaweiDot1xLine(line string, result map[string]PortStatusEntry) {
	re := regexp.MustCompile(`^(\S+(?:\d+/\d+/\d+|\d+/\d+))\s+(\w+)\s+(\w+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 4 {
		return
	}

	interfaceName := matches[1]
	enabled := strings.ToLower(matches[2]) == "enabled"

	portStatus := models.Dot1xStatusUnauthorized
	if strings.ToLower(matches[3]) == "authorized" {
		portStatus = models.Dot1xStatusAuthorized
	}

	result[interfaceName] = PortStatusEntry{
		InterfaceName:   interfaceName,
		Dot1xEnabled:    enabled,
		Dot1xPortStatus: portStatus,
	}
}

// parseCiscoDot1xLine 解析Cisco类802.1X状态行
func (c *PortCollector) parseCiscoDot1xLine(line string, result map[string]PortStatusEntry) {
	c.parseHuaweiDot1xLine(line, result)
}

// parsePortSecurity 解析端口安全配置
func (c *PortCollector) parsePortSecurity(vendor models.DeviceVendor, output string) (map[string]PortStatusEntry, error) {
	result := make(map[string]PortStatusEntry)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "Port Security") || strings.Contains(line, "---") {
			continue
		}

		// 根据厂商解析端口安全配置
		switch vendor {
		case models.VendorHuawei, models.VendorH3C:
			c.parseHuaweiPortSecurityLine(line, result)
		case models.VendorRuijie, models.VendorMaipu:
			c.parseCiscoPortSecurityLine(line, result)
		}
	}

	return result, nil
}

// parseHuaweiPortSecurityLine 解析华为/H3C端口安全配置行
// 示例: GigabitEthernet0/0/1    enabled    restrict    10    2
func (c *PortCollector) parseHuaweiPortSecurityLine(line string, result map[string]PortStatusEntry) {
	re := regexp.MustCompile(`^(\S+(?:\d+/\d+/\d+|\d+/\d+))\s+(\w+)\s+(\w+)?\s*(\d+)?\s*(\d+)?`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 3 {
		return
	}

	interfaceName := matches[1]
	enabled := strings.ToLower(matches[2]) == "enabled"

	entry := PortStatusEntry{
		InterfaceName:       interfaceName,
		PortSecurityEnabled: enabled,
	}

	if len(matches) > 3 && matches[3] != "" {
		mode := strings.ToLower(matches[3])
		switch mode {
		case "protect":
			entry.PortSecurityMode = models.PortSecurityModeProtect
		case "restrict":
			entry.PortSecurityMode = models.PortSecurityModeRestrict
		case "shutdown", "error-disable":
			entry.PortSecurityMode = models.PortSecurityModeShutdown
		}
	}

	if len(matches) > 4 && matches[4] != "" {
		if maxCount, err := strconv.Atoi(matches[4]); err == nil {
			entry.MaxMACCount = &maxCount
		}
	}

	if len(matches) > 5 && matches[5] != "" {
		if currentCount, err := strconv.Atoi(matches[5]); err == nil {
			entry.CurrentMACCount = &currentCount
		}
	}

	result[interfaceName] = entry
}

// parseCiscoPortSecurityLine 解析Cisco类端口安全配置行
func (c *PortCollector) parseCiscoPortSecurityLine(line string, result map[string]PortStatusEntry) {
	c.parseHuaweiPortSecurityLine(line, result)
}

// SaveToDatabase 保存端口状态到数据库
//
// P1 fix: 改为 GORM CreateInBatches 批量插入,替代原"N 次 Create"循环。
// 每台设备数十到上百端口,N 台设备同时采集会打爆 DB 连接池
// (每个 Create 一次 round-trip)。批量插入将 N 次 INSERT 合并为
// 单次 INSERT 多行,DB 往返从 N 降到 ceil(N/batchSize) ≈ 1-2 次。
func (c *PortCollector) SaveToDatabase(deviceID string, entries []PortStatusEntry) error {
	if len(entries) == 0 {
		return nil
	}

	collectedAt := time.Now()

	// 预分配 slice,避免 append 扩容
	portStatuses := make([]models.DevicePortStatus, 0, len(entries))
	for _, entry := range entries {
		portStatuses = append(portStatuses, models.DevicePortStatus{
			DeviceID:            deviceID,
			InterfaceName:       entry.InterfaceName,
			AdminStatus:         entry.AdminStatus,
			OperStatus:          entry.OperStatus,
			Dot1xEnabled:        entry.Dot1xEnabled,
			Dot1xPortStatus:     entry.Dot1xPortStatus,
			PortSecurityEnabled: entry.PortSecurityEnabled,
			PortSecurityMode:    entry.PortSecurityMode,
			MaxMACCount:         entry.MaxMACCount,
			CurrentMACCount:     entry.CurrentMACCount,
			CollectedAt:         collectedAt,
		})
	}

	// batchSize=100 平衡内存(每行 ~200 B,100 行 ~20 KB)与单语句大小
	const batchSize = 100
	if err := c.db.CreateInBatches(portStatuses, batchSize).Error; err != nil {
		return fmt.Errorf("批量保存端口状态失败: %w", err)
	}

	return nil
}

// CollectAndSave 采集并保存端口状态
func (c *PortCollector) CollectAndSave(ctx context.Context, deviceID string) error {
	entries, err := c.CollectFromDevice(ctx, deviceID)
	if err != nil {
		return err
	}

	return c.SaveToDatabase(deviceID, entries)
}

// GetPortStatusByDevice 查询设备的端口状态
func (c *PortCollector) GetPortStatusByDevice(deviceID string, limit, offset int) ([]models.DevicePortStatus, int64, error) {
	var results []models.DevicePortStatus
	var total int64

	query := c.db.Model(&models.DevicePortStatus{}).Where("device_id = ?", deviceID)

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

// GetPortStatusByInterface 查询指定接口的状态
func (c *PortCollector) GetPortStatusByInterface(deviceID, interfaceName string) (*models.DevicePortStatus, error) {
	var result models.DevicePortStatus
	err := c.db.Where("device_id = ? AND interface_name = ?", deviceID, interfaceName).
		Order("collected_at DESC").
		First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CleanOldRecords 清理旧的端口状态记录
func (c *PortCollector) CleanOldRecords(retentionDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	return c.db.Where("collected_at < ?", cutoffDate).Delete(&models.DevicePortStatus{}).Error
}

// GetPortsWithSecurityEnabled 获取启用了端口安全的端口列表
func (c *PortCollector) GetPortsWithSecurityEnabled(deviceID string) ([]models.DevicePortStatus, error) {
	var results []models.DevicePortStatus
	err := c.db.Where("device_id = ? AND port_security_enabled = ?", deviceID, true).
		Order("interface_name ASC").
		Find(&results).Error
	return results, err
}

// GetPortsWithDot1xEnabled 获取启用了802.1X的端口列表
func (c *PortCollector) GetPortsWithDot1xEnabled(deviceID string) ([]models.DevicePortStatus, error) {
	var results []models.DevicePortStatus
	err := c.db.Where("device_id = ? AND dot1x_enabled = ?", deviceID, true).
		Order("interface_name ASC").
		Find(&results).Error
	return results, err
}

// getInterfaceCommand 根据厂商获取接口状态命令
func (c *PortCollector) getInterfaceCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display interface",
		models.VendorH3C:    "display interface",
		models.VendorRuijie: "show interface",
		models.VendorMaipu:  "show interface",
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "display interface"
}

// getDot1xCommand 根据厂商获取802.1X状态命令
func (c *PortCollector) getDot1xCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display dot1x",
		models.VendorH3C:    "display dot1x",
		models.VendorRuijie: "show dot1x",
		models.VendorMaipu:  "show dot1x",
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "display dot1x"
}

// getPortSecurityCommand 根据厂商获取端口安全配置命令
func (c *PortCollector) getPortSecurityCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display port-security",
		models.VendorH3C:    "display port-security",
		models.VendorRuijie: "show port-security",
		models.VendorMaipu:  "show port-security",
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "display port-security"
}
