package services

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/base"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/lldp"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/topology"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// MACCollectionService MAC地址采集服务
type MACCollectionService struct {
	db              *gorm.DB
	executor        *device.DeviceExecutor
	maxConcurrent   int
	lldpService     *lldp.LLDPService
	filterRuleService topology.FilterRuleService
	enableHistory   bool // 是否启用MAC历史记录
}

// MAC采集并发数的默认配置键
const macConcurrentConfigKey = "network.mac.collection.concurrent"

// macCollectionDeviceTimeout MAC 采集单设备超时
const macCollectionDeviceTimeout = 10 * time.Minute

// NewMACCollectionService 创建MAC采集服务
func NewMACCollectionService(db *gorm.DB, executor *device.DeviceExecutor, lldpSvc *lldp.LLDPService, filterRuleSvc topology.FilterRuleService) *MACCollectionService {
	svc := &MACCollectionService{
		db:                db,
		executor:          executor,
		maxConcurrent:     10, // 默认值
		lldpService:       lldpSvc,
		filterRuleService: filterRuleSvc,
		enableHistory:     true, // 默认启用历史记录
	}
	// 从数据库加载配置
	svc.loadConfigFromDB()
	return svc
}

// MACCollectionResult MAC采集结果
type MACCollectionResult struct {
	DeviceID       string
	DeviceName     string
	SuccessCount   int
	FailedCount    int
	ErrorMessage   string
	CollectionTime time.Time
}

// CollectAllDevices 采集所有设备的MAC地址表
func (s *MACCollectionService) CollectAllDevices(ctx context.Context) ([]*MACCollectionResult, error) {
	// 获取所有在线设备
	var devices []models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("status = ?", models.DeviceStatusOnline).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询设备列表失败: %w", err)
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("没有在线设备")
	}

	var results []*MACCollectionResult
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, s.maxConcurrent) // 使用动态并发数

	for _, dev := range devices {
		wg.Add(1)
		go func(device models.NetworkDevice) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 为每个设备创建独立的 context（带超时控制）
			// 这样单个设备的问题不会影响其他设备
			deviceCtx, cancel := context.WithTimeout(ctx, macCollectionDeviceTimeout)
			defer cancel()

			result := s.collectDeviceMAC(deviceCtx, &device)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(dev)
	}

	wg.Wait()

	return results, nil
}

// CollectDevice 采集单个设备的MAC地址表
func (s *MACCollectionService) CollectDevice(ctx context.Context, deviceID string) (*MACCollectionResult, error) {
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("设备不存在: %w", err)
	}

	return s.collectDeviceMAC(ctx, &device), nil
}

// collectDeviceMAC 采集设备MAC地址表（内部方法）
func (s *MACCollectionService) collectDeviceMAC(ctx context.Context, device *models.NetworkDevice) *MACCollectionResult {
	// 添加 panic recovery，防止 scrapligo 库的 panic 导致程序崩溃
	defer func() {
		if r := recover(); r != nil {
			applogger.Infof("[MAC采集] 设备 %s 发生 panic: %v", device.DeviceName, r)
		}
	}()

	result := &MACCollectionResult{
		DeviceID:       device.ID,
		DeviceName:     device.DeviceName,
		CollectionTime: time.Now(),
	}

	// Step 1: 发现LLDP邻居（best-effort，失败不影响MAC采集）
	// 注意：LLDP服务返回的map键是已经规范化的接口名（来自Plan 01 Task 5）
	lldpNeighbors := make(map[string]*models.LLDPNeighborInfo)
	if s.lldpService != nil {
		neighbors, err := s.lldpService.DiscoverNeighbors(ctx, device)
		if err != nil {
			applogger.Warnf("[MAC采集] %s: LLDP发现失败 (仅使用MAC数过滤): %v", device.DeviceName, err)
		} else {
			lldpNeighbors = neighbors
			applogger.Infof("[MAC采集] %s: 发现 %d 个LLDP邻居", device.DeviceName, len(neighbors))
		}
	}

	// 获取MAC地址表命令
	command := s.getMACCommand(device.Vendor)

	// 使用 executor 发送命令
	response, err := s.executor.ExecuteOnDevice(ctx, device.ID, command, true)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}

	// 解析MAC地址表
	macAddresses, err := s.parseMACAddressTable(response, device.Vendor)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}

	// Step 2: 构建每个接口的MAC地址数量统计
	macCountByInterface := make(map[string]int)
	for _, mac := range macAddresses {
		normalized := lldp.NormalizeInterfaceName(mac.InterfaceName)
		macCountByInterface[normalized]++
	}

	// Step 3: 获取设备类型对应的MAC数量阈值
	threshold := s.getMACThreshold(device)
	applogger.Infof("[MAC采集] %s: MAC数阈值=%d (设备类型=%s)", device.DeviceName, threshold, device.DeviceType)

	// Step 4: 应用过滤规则
	filteredCount := 0
	var filteredMACAddresses []MACAddressEntry

	for _, mac := range macAddresses {
		normalizedIface := lldp.NormalizeInterfaceName(mac.InterfaceName)

		// 过滤规则1: LLDP邻居端口
		// 注意：LLDP map的键已经是规范化的（来自Plan 01），可以直接查找
		if neighbor, exists := lldpNeighbors[normalizedIface]; exists {
			applogger.Debugf("[MAC采集] %s: 过滤LLDP邻居端口 %s (邻居: %s)",
				device.DeviceName, mac.InterfaceName, neighbor.NeighborName)
			filteredCount++
			continue
		}

		// 过滤规则2: MAC数量超过阈值
		macCount := macCountByInterface[normalizedIface]
		if macCount > threshold {
			applogger.Debugf("[MAC采集] %s: 过滤高MAC数端口 %s (MAC数=%d, 阈值=%d)",
				device.DeviceName, mac.InterfaceName, macCount, threshold)
			filteredCount++
			continue
		}

		filteredMACAddresses = append(filteredMACAddresses, mac)
	}

	applogger.Infof("[MAC采集] %s: 总MAC=%d, 过滤=%d, 保留=%d",
		device.DeviceName, len(macAddresses), filteredCount, len(filteredMACAddresses))

	// 批量保存到数据库
	// 策略：先删除该设备的旧MAC地址，然后批量插入新数据
	collectionTime := time.Now()
	successCount := 0

	// 【新增】Step 5: 查询旧MAC地址状态（DELETE 前）- CONTEXT.md D-01
	var oldMACs []models.DeviceMACAddress
	if err := s.db.WithContext(ctx).Where("device_id = ?", device.ID).Find(&oldMACs).Error; err != nil {
		applogger.Warnf("[MAC采集] %s: 查询旧MAC状态失败: %v", device.DeviceName, err)
		// 继续执行，不阻断采集
	}

	// Step 6: 删除旧数据（原有逻辑）
	if err := s.db.WithContext(ctx).
		Where("device_id = ?", device.ID).
		Delete(&models.DeviceMACAddress{}).Error; err != nil {
		result.ErrorMessage = fmt.Sprintf("删除旧MAC地址失败: %v", err)
		return result
	}

	// Step 7: 构建新MAC地址记录列表
	var macRecords []*models.DeviceMACAddress
	for _, macAddr := range filteredMACAddresses {
		var vlanIDPtr *int
		if macAddr.VLANID > 0 {
			vlanIDPtr = &macAddr.VLANID
		}

		macType := models.MACTypeDynamic
		switch macAddr.MACType {
		case "STATIC":
			macType = models.MACTypeStatic
		case "SECURE":
			macType = models.MACTypeSecure
		}

		macRecord := &models.DeviceMACAddress{
			DeviceID:      device.ID,
			MACAddress:    macAddr.MACAddress,
			InterfaceName: macAddr.InterfaceName,
			VLANID:        vlanIDPtr,
			MACType:       macType,
			CollectedAt:   collectionTime,
		}
		macRecords = append(macRecords, macRecord)
	}

	// Step 8: 批量插入新的MAC地址记录
	if len(macRecords) > 0 {
		if err := s.db.WithContext(ctx).
			Create(&macRecords).Error; err != nil {
			result.ErrorMessage = fmt.Sprintf("批量插入MAC地址失败: %v", err)
			return result
		}
		successCount = len(macRecords)
	}

	// 【新增】Step 9: 记录MAC变更历史（DELETE 后）- CONTEXT.md D-06, D-07
	// 注意：使用历史服务记录变更（如果启用）
	if s.enableHistory && len(oldMACs) > 0 {
		// 创建历史服务实例
		historySvc := NewMACHistoryService(s.db)

		// 构建旧状态映射（从已查询的 oldMACs）
		oldState := historySvc.BuildMACStateMap(oldMACs)

		// 构建新状态映射（从已插入的 macRecords）
		var macRecordsSlice []models.DeviceMACAddress
		for _, record := range macRecords {
			macRecordsSlice = append(macRecordsSlice, *record)
		}
		newState := historySvc.BuildMACStateMap(macRecordsSlice)

		// 记录变更（非阻塞）
		if err := historySvc.RecordMACChange(ctx, device, oldState, newState); err != nil {
			applogger.Errorf("[MAC采集] %s: 记录历史失败: %v", device.DeviceName, err)
			// 不阻断采集流程（CONTEXT.md D-07）
		} else {
			applogger.Infof("[MAC采集] %s: MAC历史记录成功", device.DeviceName)
		}
	}

	result.SuccessCount = successCount
	return result
}

// MACAddressEntry MAC地址条目
type MACAddressEntry struct {
	MACAddress    string
	InterfaceName string
	VLANID        int
	MACType       string
}

// getMACCommand 根据厂商获取MAC地址表命令
func (s *MACCollectionService) getMACCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display mac-address",
		models.VendorH3C:    "display mac-address",
		models.VendorRuijie: "show mac-address-table",
		models.VendorMaipu:  "show mac-address-table",
	}

	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "show mac-address-table" // 默认命令
}

// parseMACAddressTable 解析MAC地址表
func (s *MACCollectionService) parseMACAddressTable(output string, vendor models.DeviceVendor) ([]MACAddressEntry, error) {
	var entries []MACAddressEntry
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 过滤空行和分隔符行
		if line == "" || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "===") {
			continue
		}

		// 过滤标题行 - 检查常见的标题关键词
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "mac address") ||
		   strings.Contains(lowerLine, "vlan") && strings.Contains(lowerLine, "mac") ||
		   strings.Contains(lowerLine, "type") && strings.Contains(lowerLine, "interface") ||
		   strings.Contains(lowerLine, "port") && strings.Contains(lowerLine, "mac") ||
		   strings.Contains(lowerLine, "learned") {
			continue
		}

		// 根据不同厂商的格式解析
		entry, ok := s.parseMACLine(line, vendor)
		if ok {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// parseMACLine 解析MAC地址行
func (s *MACCollectionService) parseMACLine(line string, vendor models.DeviceVendor) (MACAddressEntry, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return MACAddressEntry{}, false
	}

	var entry MACAddressEntry

	switch vendor {
	case models.VendorHuawei, models.VendorH3C:
		// Huawei/H3C 格式: MAC Address VLAN Interface... Type
		// 实际输出示例需要确认，可能格式为:
		// d89e.f327.2d19 100 GigabitEthernet0/1/1 Dynamic
		// 或者包含更多空格的变体
		if len(fields) >= 4 {
			entry.MACAddress = fields[0]
			// 解析VLAN ID (fields[1])
			if vlanID, err := strconv.Atoi(fields[1]); err == nil {
				entry.VLANID = vlanID
			}

			// MAC类型是最后一个字段
			entry.MACType = fields[len(fields)-1]

			// 智能识别接口名称：从fields[2]开始，到最后一个非类型字段
			// 常见的MAC类型：Dynamic, Static, Secure
			interfaceParts := []string{}
			for i := 2; i < len(fields)-1; i++ {
				// 跳过已知的MAC类型字段
				lowerField := strings.ToLower(fields[i])
				if lowerField == "dynamic" || lowerField == "static" || lowerField == "secure" {
					continue
				}
				interfaceParts = append(interfaceParts, fields[i])
			}

			if len(interfaceParts) > 0 {
				entry.InterfaceName = strings.Join(interfaceParts, " ")
			} else {
				// fallback: 使用fields[2]
				entry.InterfaceName = fields[2]
			}

			// 清理接口名称中的时间戳格式（如 "2026-5-9 0:51:07"）
			entry.InterfaceName = s.cleanTimestampFromInterface(entry.InterfaceName)
		}

	case models.VendorRuijie, models.VendorMaipu:
		// Ruijie/Maipu 格式: VLAN MAC Address Type Interface...
		if len(fields) >= 4 {
			// 解析VLAN ID (fields[0])
			if vlanID, err := strconv.Atoi(fields[0]); err == nil {
				entry.VLANID = vlanID
			}
			entry.MACAddress = fields[1]
			entry.MACType = fields[2]

			// 接口名称从fields[3]开始
			interfaceParts := []string{}
			for i := 3; i < len(fields); i++ {
				interfaceParts = append(interfaceParts, fields[i])
			}

			if len(interfaceParts) > 0 {
				entry.InterfaceName = strings.Join(interfaceParts, " ")
			} else {
				entry.InterfaceName = fields[3]
			}

			// 清理接口名称中的时间戳格式（如 "2026-5-9 0:51:07"）
			entry.InterfaceName = s.cleanTimestampFromInterface(entry.InterfaceName)
		}
	}

	if entry.MACAddress != "" {
		return entry, true
	}
	return MACAddressEntry{}, false
}

// cleanTimestampFromInterface 清理接口名称中的时间戳
// 示例: "GigabitEthernet 0/12 2026-5-9 0:51:07" -> "GigabitEthernet 0/12"
func (s *MACCollectionService) cleanTimestampFromInterface(interfaceName string) string {
	// 使用正则表达式匹配时间戳格式
	// 可能的格式: "2026-5-9 0:51:07", "2026-05-09 08:52:15" 等
	timestampPattern := regexp.MustCompile(`\d{4}-\d{1,2}-\d{1,2}.*\d{1,2}:\d{2}:\d{2}`)
	cleaned := timestampPattern.ReplaceAllString(interfaceName, "")
	cleaned = strings.TrimSpace(cleaned)

	// 如果清理后为空，返回原始接口名称
	if cleaned == "" {
		return interfaceName
	}
	return cleaned
}

// MACAddressResponse MAC地址响应DTO（包含设备名称）
type MACAddressResponse struct {
	ID            string    `json:"id"`
	DeviceID      string    `json:"deviceId"`
	DeviceName    string    `json:"deviceName"`
	MACAddress    string    `json:"macAddress"`
	InterfaceName string    `json:"interfaceName"`
	VLANID        *int      `json:"vlanId,omitempty"`
	MACType       string    `json:"macType,omitempty"`
	CollectedAt   time.Time `json:"collectedAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

// GetMACAddressList 获取MAC地址列表
// orderByColumn/isAsc 为服务端排序参数(可选,透传给 base.ApplySort 白名单)。
func (s *MACCollectionService) GetMACAddressList(ctx context.Context, current, pageSize int, deviceID, macAddress, interfaceName string, orderByColumn string, isAsc *bool) ([]MACAddressResponse, int64, error) {
	var total int64

	// 构建基础查询（用于计数）
	baseQuery := s.db.WithContext(ctx).Model(&models.DeviceMACAddress{})

	if deviceID != "" {
		baseQuery = baseQuery.Where("device_id = ?", deviceID)
	}
	if macAddress != "" {
		baseQuery = baseQuery.Where("mac_address LIKE ?", "%"+macAddress+"%")
	}
	if interfaceName != "" {
		baseQuery = baseQuery.Where("interface_name LIKE ?", "%"+interfaceName+"%")
	}

	// 获取总数
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询MAC地址总数失败: %w", err)
	}

	// 分页查询，JOIN获取设备名称
	offset := (current - 1) * pageSize
	type macWithDeviceName struct {
		models.DeviceMACAddress
		DeviceName string
	}

	// 构建JOIN查询
	joinQuery := s.db.WithContext(ctx).
		Select("sys_device_mac_address.*, sys_network_device.device_name").
		Joins("LEFT JOIN sys_network_device ON sys_network_device.id = sys_device_mac_address.device_id")

	// 应用筛选条件到JOIN查询
	if deviceID != "" {
		joinQuery = joinQuery.Where("sys_device_mac_address.device_id = ?", deviceID)
	}
	if macAddress != "" {
		joinQuery = joinQuery.Where("sys_device_mac_address.mac_address LIKE ?", "%"+macAddress+"%")
	}
	if interfaceName != "" {
		joinQuery = joinQuery.Where("sys_device_mac_address.interface_name LIKE ?", "%"+interfaceName+"%")
	}

	var results []macWithDeviceName
	// 用户排序(白名单,带表别名)优先,无 OrderByColumn 时保留原默认
	sortReq := base.BaseListRequest{
		Current:       current,
		PageSize:      pageSize,
		OrderByColumn: orderByColumn,
		IsAsc:         isAsc,
	}
	joinQuery = base.ApplySort(joinQuery, sortReq, macAddressAllowedSortFields)
	if orderByColumn == "" {
		joinQuery = joinQuery.Order("sys_device_mac_address.collected_at DESC")
	}
	if err := joinQuery.
		Offset(offset).
		Limit(pageSize).
		Find(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("查询MAC地址列表失败: %w", err)
	}

	// 转换为响应DTO
	response := make([]MACAddressResponse, len(results))
	for i, r := range results {
		response[i] = MACAddressResponse{
			ID:            r.ID,
			DeviceID:      r.DeviceID,
			DeviceName:    r.DeviceName,
			MACAddress:    r.MACAddress,
			InterfaceName: r.InterfaceName,
			VLANID:        r.VLANID,
			MACType:       string(r.MACType),
			CollectedAt:   r.CollectedAt,
			CreatedAt:     r.CreatedAt,
		}
	}

	return response, total, nil
}

// macAddressAllowedSortFields MAC地址可排序字段白名单(对应 sys_device_mac_address 表列名)。
var macAddressAllowedSortFields = map[string]string{
	"deviceId":      "sys_device_mac_address.device_id",
	"macAddress":    "sys_device_mac_address.mac_address",
	"interfaceName": "sys_device_mac_address.interface_name",
	"vlanId":        "sys_device_mac_address.vlan_id",
	"collectedAt":   "sys_device_mac_address.collected_at",
}

// GetMACAddressStats 获取MAC地址统计信息
func (s *MACCollectionService) GetMACAddressStats(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalRecords     int64
		UniqueDevices    int64
		UniqueMACs       int64
		LatestCollection time.Time
	}

	s.db.WithContext(ctx).Model(&models.DeviceMACAddress{}).Count(&stats.TotalRecords)

	s.db.WithContext(ctx).Model(&models.DeviceMACAddress{}).
		Select("COUNT(DISTINCT device_id)").Scan(&stats.UniqueDevices)

	s.db.WithContext(ctx).Model(&models.DeviceMACAddress{}).
		Select("COUNT(DISTINCT mac_address)").Scan(&stats.UniqueMACs)

	var latest models.DeviceMACAddress
	s.db.WithContext(ctx).Order("collected_at DESC").First(&latest)
	if latest.ID != "" {
		stats.LatestCollection = latest.CollectedAt
	}

	return map[string]interface{}{
		"totalRecords":     stats.TotalRecords,
		"uniqueDevices":    stats.UniqueDevices,
		"uniqueMACs":       stats.UniqueMACs,
		"latestCollection": stats.LatestCollection,
	}, nil
}

// CleanOldRecords 清理旧的MAC地址记录（保留最近N天）
func (s *MACCollectionService) CleanOldRecords(ctx context.Context, days int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -days)

	result := s.db.WithContext(ctx).
		Where("collected_at < ?", cutoffTime).
		Delete(&models.DeviceMACAddress{})

	if result.Error != nil {
		return 0, fmt.Errorf("清理旧记录失败: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// BatchDelete 批量删除MAC地址记录
func (s *MACCollectionService) BatchDelete(ctx context.Context, ids []string) (int64, error) {
	result := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.DeviceMACAddress{})
	if result.Error != nil {
		return 0, fmt.Errorf("批量删除MAC地址记录失败: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// loadConfigFromDB 从数据库加载并发数配置
func (s *MACCollectionService) loadConfigFromDB() {
	var config models.Config
	err := s.db.Where("config_key = ?", macConcurrentConfigKey).First(&config).Error
	if err == nil && config.ConfigValue != "" {
		if concurrent, parseErr := strconv.Atoi(config.ConfigValue); parseErr == nil && concurrent > 0 {
			s.maxConcurrent = concurrent
			applogger.Infof("[MAC采集] 从数据库读取并发数配置: %d", s.maxConcurrent)
		}
	}
}

// ReloadConfig 重新加载配置（从数据库）
func (s *MACCollectionService) ReloadConfig() {
	s.loadConfigFromDB()
}

// getMACThreshold 根据设备类型获取MAC数量阈值
// 用于过滤设备间互联端口（上行链路通常有更多MAC地址）
// 优先使用数据库配置的过滤规则，回退到硬编码默认值
func (s *MACCollectionService) getMACThreshold(device *models.NetworkDevice) int {
	// 尝试从数据库获取有效规则（优先使用配置的阈值）
	if s.filterRuleService != nil {
		rule, err := s.filterRuleService.GetEffectiveRule(context.Background(), device)
		if err == nil && rule != nil {
			applogger.Debugf("[MAC采集] %s: 使用数据库配置阈值=%d (规则=%s)",
				device.DeviceName, rule.MACThreshold, rule.RuleName)
			return rule.MACThreshold
		}
		// 规则查询失败时使用默认值，记录警告
		applogger.Warnf("[MAC采集] %s: 无法获取过滤规则，使用默认阈值: %v", device.DeviceName, err)
	}

	// 硬编码默认阈值（当数据库规则不可用时）
	thresholds := map[models.DeviceType]int{
		models.DeviceTypeRouter:       500, // 路由器：通常连接多个子网，MAC数量多
		models.DeviceTypeSwitch:       10,  // 交换机：接入端口MAC数少，上行端口MAC数多
		models.DeviceTypeFirewall:     100, // 防火墙：可能连接多个网段
		models.DeviceTypeLoadBalancer: 50,  // 负载均衡器：连接后端服务器
	}

	if threshold, ok := thresholds[device.DeviceType]; ok {
		return threshold
	}
	return 10 // 默认阈值（针对未知设备类型）
}
