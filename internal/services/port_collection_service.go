package services

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/templates"
)

// PortCollectionService 端口状态采集服务
type PortCollectionService struct {
	db            *gorm.DB
	executor      *device.DeviceExecutor
	maxConcurrent int
	templateCache *TemplateCache // 模板缓存
}

// 端口采集并发数的默认配置键
const portConcurrentConfigKey = "network.port.collection.concurrent"

// NewPortCollectionService 创建端口采集服务
func NewPortCollectionService(db *gorm.DB, executor *device.DeviceExecutor) *PortCollectionService {
	svc := &PortCollectionService{
		db:            db,
		executor:      executor,
		maxConcurrent: 10, // 默认值
		templateCache: NewTemplateCache(), // 初始化模板缓存
	}
	// 从数据库加载配置
	svc.loadConfigFromDB()
	return svc
}

// PortCollectionResult 端口采集结果
type PortCollectionResult struct {
	DeviceID       string
	DeviceName     string
	SuccessCount   int
	FailedCount    int
	ErrorMessage   string
	CollectionTime time.Time
}

// CollectAllDevices 采集所有设备的端口状态
func (s *PortCollectionService) CollectAllDevices(ctx context.Context) ([]*PortCollectionResult, error) {
	// 获取所有在线设备
	var devices []models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("status = ?", models.DeviceStatusOnline).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询设备列表失败: %w", err)
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("没有在线设备")
	}

	var results []*PortCollectionResult
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
			deviceCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()

			result := s.collectDevicePort(deviceCtx, &device)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(dev)
	}

	wg.Wait()

	return results, nil
}

// CollectDevice 采集单个设备的端口状态
func (s *PortCollectionService) CollectDevice(ctx context.Context, deviceID string) (*PortCollectionResult, error) {
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("设备不存在: %w", err)
	}

	return s.collectDevicePort(ctx, &device), nil
}

// collectDevicePort 采集设备端口状态（内部方法）
func (s *PortCollectionService) collectDevicePort(ctx context.Context, device *models.NetworkDevice) *PortCollectionResult {
	// 添加 panic recovery，防止 scrapligo 库的 panic 导致程序崩溃
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[端口采集] 设备 %s 发生 panic: %v", device.DeviceName, r)
		}
	}()

	result := &PortCollectionResult{
		DeviceID:       device.ID,
		DeviceName:     device.DeviceName,
		CollectionTime: time.Now(),
	}

	// 从 executor 获取连接池
	pool := s.executor.GetScheduler().GetConnectionPool()

	// 获取设备连接
	conn, err := pool.GetConnection(ctx, device.ID)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}

	// 增加引用计数
	if !conn.Acquire() {
		result.ErrorMessage = "无法获取连接锁"
		return result
	}
	defer conn.Release()

	// 获取 wrapper
	wrapper := conn.GetWrapper()
	if wrapper == nil {
		result.ErrorMessage = "wrapper 为空"
		return result
	}

	// 获取接口列表命令
	interfaceCmd := s.getInterfaceCommand(device.Vendor)

	// 发送命令获取接口信息
	interfaceResp, err := wrapper.SendCommand(interfaceCmd, true)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}

	// 解析接口列表（物理属性）
	interfaces, err := s.parseInterfaceList(interfaceResp.Result, device.Vendor)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}

	// 获取接口状态和描述信息（仅锐捷/迈普设备）
	descriptionMap := make(map[string]InterfaceDescription)
	if device.Vendor == models.VendorRuijie || device.Vendor == models.VendorMaipu {
		descs, err := s.parseInterfaceDescriptions(wrapper, device.Vendor)
		if err == nil {
			descriptionMap = descs
		}
	}

	// 批量获取所有端口的802.1X状态（从96次命令降到1次）
	dot1xMap, err := s.getAllDot1xStatus(wrapper, device.Vendor)
	if err != nil {
		log.Printf("[警告] 获取802.1X状态失败: %v", err)
		dot1xMap = make(map[string]Dot1xInfo)
	}

	// 批量获取所有端口安全配置（从96次命令降到1次）
	securityMap, err := s.getAllPortSecurity(wrapper, device.Vendor)
	if err != nil {
		log.Printf("[警告] 获取端口安全配置失败: %v", err)
		securityMap = make(map[string]PortSecurityInfo)
	}

	// 获取802.1X和端口安全信息
	collectionTime := time.Now()

	// 收集所有端口状态数据
	var portStatuses []*models.DevicePortStatus

	if len(interfaces) == 0 && len(descriptionMap) > 0 {
		// 物理属性解析失败，但状态/描述解析成功
		// 这种情况通常发生在路由器上（不支持 show interfaces status 命令）
		// 使用降级方案：仅基于状态/描述数据创建端口记录（不包含 VLAN/Duplex/Speed/Type）
		for ifaceName, desc := range descriptionMap {
			portStatus := &models.DevicePortStatus{
				DeviceID:      device.ID,
				InterfaceName: ifaceName,
				AdminStatus:   desc.AdminStatus,
				OperStatus:    desc.OperStatus,
				Description:   desc.Description,
				// VLAN, Duplex, Speed, PortType 保持空值（路由器通常不需要这些字段）
				CollectedAt: collectionTime,
			}

			// 从批量获取的map中获取802.1X状态
			if dot1x, ok := dot1xMap[ifaceName]; ok {
				portStatus.Dot1xEnabled = dot1x.Enabled
				switch strings.ToUpper(dot1x.PortStatus) {
				case "AUTHORIZED":
					portStatus.Dot1xPortStatus = models.Dot1xStatusAuthorized
				case "UNAUTHORIZED":
					portStatus.Dot1xPortStatus = models.Dot1xStatusUnauthorized
				default:
					portStatus.Dot1xPortStatus = models.Dot1xStatusUnknown
				}
			}

			// 从批量获取的map中获取端口安全配置
			if security, ok := securityMap[ifaceName]; ok {
				portStatus.PortSecurityEnabled = security.Enabled

				var securityMode models.PortSecurityMode
				switch strings.ToUpper(security.SecurityMode) {
				case "RESTRICT":
					securityMode = models.PortSecurityModeRestrict
				case "PROTECT":
					securityMode = models.PortSecurityModeProtect
				case "SHUTDOWN":
					securityMode = models.PortSecurityModeShutdown
				default:
					securityMode = models.PortSecurityModeNone
				}
				portStatus.PortSecurityMode = securityMode

				if security.MaxMAC > 0 {
					portStatus.MaxMACCount = &security.MaxMAC
				}
				if security.CurrentMAC > 0 {
					portStatus.CurrentMACCount = &security.CurrentMAC
				}
			}

			portStatuses = append(portStatuses, portStatus)
		}
	} else {
		// 正常流程：基于物理属性数据创建端口记录
		for _, iface := range interfaces {
			// 标准化接口名称
			normalizedName := normalizeInterfaceName(iface.Name)

			// 从状态和描述映射中获取状态信息（使用标准化名称查找）
			var adminStatus, operStatus, description string
			if desc, ok := descriptionMap[normalizedName]; ok {
				adminStatus = desc.AdminStatus
				operStatus = desc.OperStatus
				description = desc.Description
			}

			portStatus := &models.DevicePortStatus{
				DeviceID:      device.ID,
				InterfaceName: normalizedName, // 使用标准化后的名称
				AdminStatus:   adminStatus,
				OperStatus:    operStatus,
				Description:   description,
				VLAN:          iface.VLAN,
				Duplex:        iface.Duplex,
				Speed:         iface.Speed,
				PortType:      iface.PortType,
				CollectedAt:   collectionTime,
			}

			// 从批量获取的map中获取802.1X状态（而不是发送命令）
			if dot1x, ok := dot1xMap[normalizedName]; ok {
				portStatus.Dot1xEnabled = dot1x.Enabled
				switch strings.ToUpper(dot1x.PortStatus) {
				case "AUTHORIZED":
					portStatus.Dot1xPortStatus = models.Dot1xStatusAuthorized
				case "UNAUTHORIZED":
					portStatus.Dot1xPortStatus = models.Dot1xStatusUnauthorized
				default:
					portStatus.Dot1xPortStatus = models.Dot1xStatusUnknown
				}
			}

			// 从批量获取的map中获取端口安全配置（而不是发送命令）
			if security, ok := securityMap[normalizedName]; ok {
				portStatus.PortSecurityEnabled = security.Enabled

				var securityMode models.PortSecurityMode
				switch strings.ToUpper(security.SecurityMode) {
				case "RESTRICT":
					securityMode = models.PortSecurityModeRestrict
				case "PROTECT":
					securityMode = models.PortSecurityModeProtect
				case "SHUTDOWN":
					securityMode = models.PortSecurityModeShutdown
				default:
					securityMode = models.PortSecurityModeNone
				}
				portStatus.PortSecurityMode = securityMode

				if security.MaxMAC > 0 {
					portStatus.MaxMACCount = &security.MaxMAC
				}
				if security.CurrentMAC > 0 {
					portStatus.CurrentMACCount = &security.CurrentMAC
				}
			}

			portStatuses = append(portStatuses, portStatus)
		}
	}

	// 批量保存到数据库（使用 ON CONFLICT 实现 upsert）
	err = s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}, {Name: "interface_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"admin_status", "oper_status", "description",
			"vlan", "duplex", "speed", "port_type",
			"dot1x_enabled", "dot1x_port_status",
			"port_security_enabled", "port_security_mode",
			"max_mac_count", "current_mac_count",
			"collected_at", "created_at",
		}),
	}).Create(&portStatuses).Error

	successCount := len(portStatuses)
	failedCount := 0
	if err != nil {
		log.Printf("[端口采集] %s (%s): 批量保存失败: %v",
			device.DeviceName, device.IPAddress, err)
		successCount = 0
		failedCount = len(portStatuses)
	} else {
		log.Printf("[端口采集] %s (%s): 成功=%d",
			device.DeviceName, device.IPAddress, len(portStatuses))
	}

	result.SuccessCount = successCount
	result.FailedCount = failedCount
	return result
}

// InterfaceInfo 接口信息
type InterfaceInfo struct {
	Name        string // 接口名称
	AdminStatus string // 管理状态
	OperStatus  string // 操作状态
	Description string // 端口描述
	VLAN        *int   // VLAN ID
	Duplex      string // 双工模式
	Speed       string // 速率
	PortType    string // 端口类型
}

// getInterfaceCommand 根据厂商获取接口命令
func (s *PortCollectionService) getInterfaceCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display interface brief", // 华为使用 brief 命令
		models.VendorH3C:     "display interface",
		models.VendorRuijie:  "show interfaces status",
		models.VendorMaipu:   "show interface",
	}

	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "show interface" // 默认命令
}

// getInterfaceTemplate 根据厂商获取接口解析模板
func (s *PortCollectionService) getInterfaceTemplate(vendor models.DeviceVendor) string {
	templates := map[models.DeviceVendor]string{
		models.VendorHuawei: "templates/huawei_vrp_display_interface_brief.textfsm", // 华为使用 brief 模板
		models.VendorH3C:     "templates/huawei_vrp_display_interface.textfsm",
		models.VendorRuijie:  "templates/ruijie_os_show_interfaces_status.textfsm",
		models.VendorMaipu:   "templates/ruijie_os_show_interfaces_status.textfsm",
	}

	if tmpl, ok := templates[vendor]; ok {
		return tmpl
	}
	return "templates/huawei_vrp_display_interface.textfsm" // 默认模板
}

// parseInterfaceList 解析接口列表（使用 textfsm 模板）
// 此方法只解析物理属性：端口名称、VLAN、Duplex、Speed、Type
// 状态和描述由 parseInterfaceDescriptions 解析
func (s *PortCollectionService) parseInterfaceList(output string, vendor models.DeviceVendor) ([]InterfaceInfo, error) {
	// 获取模板文件路径
	templatePath := s.getInterfaceTemplate(vendor)

	// 解析模板
	fsm, err := templates.ParseTemplate(templatePath)
	if err != nil {
		return nil, err
	}

	// 使用模板解析输出
	records, err := fsm.ParseText(output)
	if err != nil {
		return nil, err
	}

	// 转换为 InterfaceInfo（只包含物理属性）
	var interfaces []InterfaceInfo
	for _, record := range records {
		info := InterfaceInfo{}

		// 尝试多种可能的字段名（textfsm 可能使用不同的大小写）
		var name string
		if n, ok := record["INTERFACE"]; ok && n != "" {
			name = n
		} else if n, ok := record["Interface"]; ok && n != "" {
			name = n
		} else if n, ok := record["interface"]; ok && n != "" {
			name = n
		}

		if name == "" {
			continue
		}

		// 提取并标准化接口名称
		originalName := strings.ReplaceAll(name, " ", "")
		info.Name = normalizeInterfaceName(originalName)

		// 提取 VLAN
		if vlanStr, ok := record["VLAN"]; ok && vlanStr != "--" && vlanStr != "" {
			if vlan, err := strconv.Atoi(vlanStr); err == nil {
				info.VLAN = &vlan
			}
		}

		// 提取 Duplex
		if duplex, ok := record["DUPLEX"]; ok && duplex != "--" && duplex != "" {
			info.Duplex = strings.ToLower(duplex)
		}

		// 提取 Speed
		if speed, ok := record["SPEED"]; ok && speed != "--" && speed != "" {
			info.Speed = speed
		}

		// 提取 Type
		if portType, ok := record["TYPE"]; ok && portType != "--" && portType != "" {
			info.PortType = strings.ToLower(portType)
		}

		interfaces = append(interfaces, info)
	}

	return interfaces, nil
}

// getAvailableKeys 获取 map 中所有可用的 key（用于调试）
func getAvailableKeys(record map[string]string) []string {
	keys := make([]string, 0, len(record))
	for k := range record {
		keys = append(keys, k)
	}
	return keys
}

// parseInterfaceListSimple 简单解析（降级方案）
func (s *PortCollectionService) parseInterfaceListSimple(output string, vendor models.DeviceVendor) ([]InterfaceInfo, error) {
	var interfaces []InterfaceInfo

	// 简单的正则匹配接口名称
	var interfaceRegex *regexp.Regexp

	switch vendor {
	case models.VendorRuijie, models.VendorMaipu:
		// 锐捷格式: GigabitEthernet1/0/1    up    up    ...
		interfaceRegex = regexp.MustCompile(`^(\S+(?:Ethernet|FastEthernet|GigabitEthernet|TenGigE|FortyGigE|Vlanif|Vlan|Loopback)[0-9/\s]+)\s+(\S+)\s+(\S+)`)
	default:
		// 华为/H3C 格式: GigabitEthernet1/0/1 current state : UP
		interfaceRegex = regexp.MustCompile(`^(\S+(?:Ethernet|FastEthernet|GigabitEthernet|TenGigE|FortyGigE|Vlanif|Vlan|Loopback)[0-9/\s]+)\s+`)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if interfaceRegex != nil {
			matches := interfaceRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				info := InterfaceInfo{
					Name: strings.ReplaceAll(matches[1], " ", ""),
				}
				if len(matches) > 2 {
					info.AdminStatus = strings.ToLower(matches[2])
				}
				if len(matches) > 3 {
					info.OperStatus = strings.ToLower(matches[3])
				}

				if isValidInterfaceName(info.Name) {
					interfaces = append(interfaces, info)
				}
			}
		}
	}

	return interfaces, nil
}

// isValidInterfaceName 验证是否为有效的接口名称
func isValidInterfaceName(name string) bool {
	if name == "" {
		return false
	}

	// 有效的接口前缀列表
	validPrefixes := []string{
		"GigabitEthernet", "FastEthernet", "Ethernet",
		"TenGigE", "FortyGigE", "HundredGigE",
		"Vlan", "Vlanif", "Loopback", "NULL",
		"Serial", "POS", "ATM", "Dialer",
		"Tunnel", "Port-Channel", "Bridge-Aggregation",
		"Route-Aggregation", "InLoopBack", "Register-Tunnel",
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// getDot1xStatus 获取802.1X状态
func (s *PortCollectionService) getDot1xStatus(wrapper *device.ScrapliWrapper, interfaceName string, vendor models.DeviceVendor) (bool, string) {
	command := s.getDot1xCommand(vendor)
	if strings.Contains(command, "{interface}") {
		command = strings.Replace(command, "{interface}", interfaceName, 1)
	}

	response, err := wrapper.SendCommand(command, true)
	if err != nil {
		return false, ""
	}

	// 解析响应
	output := response.Result
	enabled := false
	portStatus := "unknown"

	if strings.Contains(output, "Enabled") || strings.Contains(output, "enabled") {
		enabled = true
	}

	if strings.Contains(output, "authorized") {
		portStatus = "authorized"
	} else if strings.Contains(output, "unauthorized") {
		portStatus = "unauthorized"
	}

	return enabled, portStatus
}

// getDot1xCommand 获取802.1X命令
func (s *PortCollectionService) getDot1xCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display dot1x interface {interface}",
		models.VendorH3C:     "display dot1x interface {interface}",
		models.VendorRuijie:  "show dot1x interface {interface}",
		models.VendorMaipu:   "show dot1x interface {interface}",
	}

	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "show dot1x interface {interface}"
}

// getPortSecurity 获取端口安全配置
func (s *PortCollectionService) getPortSecurity(wrapper *device.ScrapliWrapper, interfaceName string, vendor models.DeviceVendor) (bool, string, int, int) {
	command := s.getPortSecurityCommand(vendor)
	if strings.Contains(command, "{interface}") {
		command = strings.Replace(command, "{interface}", interfaceName, 1)
	}

	response, err := wrapper.SendCommand(command, true)
	if err != nil {
		return false, "", 0, 0
	}

	// 解析响应
	output := response.Result
	enabled := false
	securityMode := ""
	maxMAC := 0
	currentMAC := 0

	if strings.Contains(output, "Enabled") || strings.Contains(output, "enabled") {
		enabled = true
	}

	// 解析安全模式
	if strings.Contains(output, "restrict") {
		securityMode = "restrict"
	} else if strings.Contains(output, "protect") {
		securityMode = "protect"
	} else if strings.Contains(output, "shutdown") {
		securityMode = "shutdown"
	}

	// 解析最大MAC数
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "max") || strings.Contains(line, "maximum") {
			fields := strings.Fields(line)
			for i, f := range fields {
				if num, err := strconv.Atoi(f); err == nil {
					maxMAC = num
					// 尝试获取当前MAC数（通常是下一个字段）
					if i+1 < len(fields) {
						if curr, err := strconv.Atoi(fields[i+1]); err == nil {
							currentMAC = curr
						}
					}
					break
				}
			}
		}
	}

	return enabled, securityMode, maxMAC, currentMAC
}

// getPortSecurityCommand 获取端口安全命令
func (s *PortCollectionService) getPortSecurityCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display port-security interface {interface}",
		models.VendorH3C:     "display port-security interface {interface}",
		models.VendorRuijie:  "show port-security interface {interface}",
		models.VendorMaipu:   "show port-security interface {interface}",
	}

	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "show port-security interface {interface}"
}

// getDeviceCredential 获取设备凭证
func (s *PortCollectionService) getDeviceCredential(ctx context.Context, device *models.NetworkDevice) (*models.AuthCredential, error) {
	if device.CredentialID != nil && *device.CredentialID != "" {
		var credential models.AuthCredential
		if err := s.db.WithContext(ctx).Where("id = ?", *device.CredentialID).First(&credential).Error; err != nil {
			return nil, fmt.Errorf("查询凭证失败: %w", err)
		}
		return &credential, nil
	}

	var credential models.AuthCredential
	if err := s.db.WithContext(ctx).Where("is_default = ?", true).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("未找到可用凭证")
	}
	return &credential, nil
}

// GetPortStatusList 获取端口状态列表
func (s *PortCollectionService) GetPortStatusList(ctx context.Context, current, pageSize int, deviceID, interfaceName, adminStatus, operStatus string, dot1xEnabled, portSecurityEnabled *bool) ([]models.DevicePortStatus, int64, error) {
	var portStatuses []models.DevicePortStatus
	var total int64

	query := s.db.WithContext(ctx).Model(&models.DevicePortStatus{})

	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if interfaceName != "" {
		query = query.Where("interface_name LIKE ?", "%"+interfaceName+"%")
	}
	if adminStatus != "" {
		query = query.Where("admin_status = ?", adminStatus)
	}
	if operStatus != "" {
		query = query.Where("oper_status = ?", operStatus)
	}
	if dot1xEnabled != nil {
		query = query.Where("dot1x_enabled = ?", *dot1xEnabled)
	}
	if portSecurityEnabled != nil {
		query = query.Where("port_security_enabled = ?", *portSecurityEnabled)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询端口状态总数失败: %w", err)
	}

	// 分页查询
	offset := (current - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("collected_at DESC").Find(&portStatuses).Error; err != nil {
		return nil, 0, fmt.Errorf("查询端口状态列表失败: %w", err)
	}

	return portStatuses, total, nil
}

// DiagnoseDevicePortData 诊断指定设备的端口数据
func (s *PortCollectionService) DiagnoseDevicePortData(ctx context.Context, deviceIP string) {
	log.Printf("[诊断] 开始检查设备 %s 的端口数据", deviceIP)

	// 先找到设备ID
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("ip_address = ?", deviceIP).First(&device).Error; err != nil {
		log.Printf("[诊断] 设备不存在: %v", err)
		return
	}

	// 检查该设备的端口数据
	var portData []models.DevicePortStatus
	if err := s.db.WithContext(ctx).Where("device_id = ?", device.ID).Find(&portData).Error; err != nil {
		log.Printf("[诊断] 查询端口数据失败: %v", err)
		return
	}

	log.Printf("[诊断] 设备 %s (%s): 找到 %d 条端口数据", device.DeviceName, deviceIP, len(portData))
}

// GetPortStatusStats 获取端口状态统计信息
func (s *PortCollectionService) GetPortStatusStats(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalRecords      int64
		UniqueDevices     int64
		Dot1xEnabledCount int64
		SecurityEnabledCount int64
		UpPortsCount      int64
		DownPortsCount    int64
		LatestCollection  time.Time
	}

	s.db.WithContext(ctx).Model(&models.DevicePortStatus{}).Count(&stats.TotalRecords)

	s.db.WithContext(ctx).Model(&models.DevicePortStatus{}).
		Select("COUNT(DISTINCT device_id)").Scan(&stats.UniqueDevices)

	s.db.WithContext(ctx).Model(&models.DevicePortStatus{}).
		Where("dot1x_enabled = ?", true).Count(&stats.Dot1xEnabledCount)

	s.db.WithContext(ctx).Model(&models.DevicePortStatus{}).
		Where("port_security_enabled = ?", true).Count(&stats.SecurityEnabledCount)

	s.db.WithContext(ctx).Model(&models.DevicePortStatus{}).
		Where("oper_status = ?", "up").Count(&stats.UpPortsCount)

	s.db.WithContext(ctx).Model(&models.DevicePortStatus{}).
		Where("oper_status = ?", "down").Count(&stats.DownPortsCount)

	var latest models.DevicePortStatus
	s.db.WithContext(ctx).Order("collected_at DESC").First(&latest)
	if latest.ID != "" {
		stats.LatestCollection = latest.CollectedAt
	}

	return map[string]interface{}{
		"totalRecords":        stats.TotalRecords,
		"uniqueDevices":       stats.UniqueDevices,
		"dot1xEnabledCount":   stats.Dot1xEnabledCount,
		"securityEnabledCount": stats.SecurityEnabledCount,
		"upPortsCount":        stats.UpPortsCount,
		"downPortsCount":      stats.DownPortsCount,
		"latestCollection":    stats.LatestCollection,
	}, nil
}

// CleanOldRecords 清理旧的端口状态记录（保留最近N天）
func (s *PortCollectionService) CleanOldRecords(ctx context.Context, days int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -days)

	result := s.db.WithContext(ctx).
		Where("collected_at < ?", cutoffTime).
		Delete(&models.DevicePortStatus{})

	if result.Error != nil {
		return 0, fmt.Errorf("清理旧记录失败: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// InterfaceDescription 接口描述信息
type InterfaceDescription struct {
	InterfaceName string // 标准化后的接口名
	OriginalName  string // 原始接口名
	AdminStatus   string // 管理状态
	OperStatus    string // 操作状态
	Description   string // 描述
}

// normalizeInterfaceName 标准化接口名称
// 将 Fa0/1, FastEthernet0/1, fastethernet0/1 等统一为 FastEthernet0/1
func normalizeInterfaceName(name string) string {
	// 移除所有空格
	name = strings.ReplaceAll(name, " ", "")

	// 如果已经是标准格式（首字母大写），直接返回
	// 检查是否以标准前缀开头
	validPrefixes := []string{"FastEthernet", "GigabitEthernet", "TenGigE", "FortyGigE", "HundredGigE", "Vlan", "Vlanif", "Loopback", "NULL"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(name, prefix) {
			return name // 已经是标准格式，直接返回
		}
	}

	// 转换为小写进行匹配
	lowerName := strings.ToLower(name)

	// 常见接口前缀映射（按长度降序排列，确保最长匹配）
	type prefixMapping struct {
		short string
		full  string
	}
	prefixList := []prefixMapping{
		{"fastethernet", "FastEthernet"},
		{"gigabitethernet", "GigabitEthernet"},
		{"tengige", "TenGigE"},
		{"fortygige", "FortyGigE"},
		{"hundredgige", "HundredGigE"},
		{"vlanif", "Vlanif"},
		{"loopback", "Loopback"},
		{"fa", "FastEthernet"},
		{"gi", "GigabitEthernet"},
		{"te", "TenGigE"},
		{"fo", "FortyGigE"},
		{"vlan", "Vlan"},
		{"vl", "Vlan"},
		{"null", "NULL"},
	}

	// 查找匹配的最长前缀
	for _, mapping := range prefixList {
		if strings.HasPrefix(lowerName, mapping.short) {
			// 提取前缀后面的部分
			suffix := lowerName[len(mapping.short):]
			// 检查后面是否紧跟数字
			if len(suffix) > 0 && (suffix[0] >= '0' && suffix[0] <= '9') {
				return mapping.full + suffix
			}
			// 如果前缀后面没有数字，可能是 VLAN 或 Loopback 等
			if len(suffix) > 0 {
				return mapping.full + suffix
			}
		}
	}

	return name
}

// parseInterfaceDescriptions 解析接口状态和描述信息
// 此方法解析：端口名称、操作状态(STATUS)、管理状态(ADMINISTRATIVE)、描述(DESCRIPTION)
func (s *PortCollectionService) parseInterfaceDescriptions(wrapper *device.ScrapliWrapper, vendor models.DeviceVendor) (map[string]InterfaceDescription, error) {
	command := s.getInterfaceDescriptionCommand(vendor)

	resp, err := wrapper.SendCommand(command, true)
	if err != nil {
		return nil, fmt.Errorf("执行接口描述命令失败: %w", err)
	}

	templatePath := s.getInterfaceDescriptionTemplate(vendor)

	// 解析模板
	fsm, err := templates.ParseTemplate(templatePath)
	if err != nil {
		return nil, err
	}

	// 使用模板解析输出
	records, err := fsm.ParseText(resp.Result)
	if err != nil {
		return nil, err
	}

	// 转换为 map，key 为标准化接口名
	descriptions := make(map[string]InterfaceDescription)
	for _, record := range records {
		desc := InterfaceDescription{}

		// 提取接口名称并标准化
		if name, ok := record["INTERFACE"]; ok {
			// 去除首尾空格（因为模板使用固定宽度匹配）
			trimmedName := strings.TrimSpace(name)
			// 标准化后作为 map 的 key
			desc.InterfaceName = normalizeInterfaceName(trimmedName)
			desc.OriginalName = trimmedName
		}

		// 提取操作状态 (Status列)
		if status, ok := record["STATUS"]; ok {
			desc.OperStatus = strings.ToLower(status)
		}

		// 提取管理状态 (Administrative列)
		if admin, ok := record["ADMINISTRATIVE"]; ok {
			desc.AdminStatus = strings.ToLower(admin)
		}

		// 提取描述
		if description, ok := record["DESCRIPTION"]; ok {
			desc.Description = description
		}

		// 验证接口名称
		if desc.InterfaceName != "" {
			descriptions[desc.InterfaceName] = desc
		}
	}

	return descriptions, nil
}

// getInterfaceDescriptionCommand 根据厂商获取接口描述命令
func (s *PortCollectionService) getInterfaceDescriptionCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display interface description", // 华为使用 display interface description
		models.VendorH3C:     "display interface description", // H3C 使用 display interface description
		models.VendorRuijie:  "show int des",
		models.VendorMaipu:   "show int des",
	}

	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "show int des" // 默认命令
}

// getInterfaceDescriptionTemplate 根据厂商获取接口描述解析模板
func (s *PortCollectionService) getInterfaceDescriptionTemplate(vendor models.DeviceVendor) string {
	templates := map[models.DeviceVendor]string{
		models.VendorHuawei: "templates/huawei_vrp_display_interface_description.textfsm", // 华为使用 description 模板
		models.VendorH3C:     "templates/huawei_vrp_display_interface_description.textfsm", // H3C 使用 description 模板
		models.VendorRuijie:  "templates/ruijie_os_show_interface_description.textfsm",
		models.VendorMaipu:   "templates/ruijie_os_show_interface_description.textfsm",
	}

	if tmpl, ok := templates[vendor]; ok {
		return tmpl
	}
	return "templates/ruijie_os_show_interface_description.textfsm" // 默认模板
}

// loadConfigFromDB 从数据库加载并发数配置
func (s *PortCollectionService) loadConfigFromDB() {
	var config models.Config
	err := s.db.Where("config_key = ?", portConcurrentConfigKey).First(&config).Error
	if err == nil && config.ConfigValue != "" {
		if concurrent, parseErr := strconv.Atoi(config.ConfigValue); parseErr == nil && concurrent > 0 {
			s.maxConcurrent = concurrent
			log.Printf("[端口采集] 从数据库读取并发数配置: %d", s.maxConcurrent)
		}
	}
}

// Dot1xInfo 802.1X信息
type Dot1xInfo struct {
	Enabled     bool
	PortStatus  string
	MaxUser     string
	CurrentUser string
}

// PortSecurityInfo 端口安全信息
type PortSecurityInfo struct {
	Enabled      bool
	SecurityMode string
	MaxMAC       int
	CurrentMAC   int
}

// getAllDot1xStatus 批量获取所有端口的802.1X状态
func (s *PortCollectionService) getAllDot1xStatus(wrapper *device.ScrapliWrapper, vendor models.DeviceVendor) (map[string]Dot1xInfo, error) {
	// 获取批量命令
	command := s.getDot1xBatchCommand(vendor)

	// 发送命令
	resp, err := wrapper.SendCommand(command, true)
	if err != nil {
		return nil, fmt.Errorf("发送802.1X命令失败: %w", err)
	}

	// 获取并解析模板
	templatePath := s.getDot1xBatchTemplate(vendor)
	fsm, err := s.templateCache.Get(templatePath)
	if err != nil {
		return nil, fmt.Errorf("加载802.1X模板失败: %w", err)
	}

	// 解析输出
	records, err := fsm.ParseText(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("解析802.1X输出失败: %w", err)
	}

	// 构建结果map
	result := make(map[string]Dot1xInfo)
	for _, record := range records {
		ifaceName := normalizeInterfaceName(record["INTERFACE"])

		// 根据厂商解析不同的字段
		if vendor == models.VendorHuawei || vendor == models.VendorH3C {
			// 华为/H3C: ENABLED, CURRENT_USERS
			enabled := record["ENABLED"] == "Enabled"
			currentUsers := 0
			if record["CURRENT_USERS"] != "" {
				currentUsers, _ = strconv.Atoi(record["CURRENT_USERS"])
			}
			// 有用户在线表示Authorized，否则根据status判断
			portStatus := "Unknown"
			if currentUsers > 0 {
				portStatus = "Authorized"
			} else if record["STATUS"] == "UP" {
				portStatus = "Unauthorized" // UP但无用户
			} else {
				portStatus = "Unknown" // DOWN端口
			}

			result[ifaceName] = Dot1xInfo{
				Enabled:     enabled,
				PortStatus:  portStatus,
				CurrentUser: record["CURRENT_USERS"],
			}
		} else {
			// 锐捷/迈普: AUTHENED (yes/no), DYNAMIC_USER, STATIC_USER
			authened := record["AUTHENED"] == "yes"
			portStatus := "Unauthorized"
			if authened {
				portStatus = "Authorized"
			}

			result[ifaceName] = Dot1xInfo{
				Enabled:     true, // 出现在列表中的端口都是启用的
				PortStatus:  portStatus,
				CurrentUser: record["DYNAMIC_USER"], // 动态用户数
			}
		}
	}
	return result, nil
}

// getAllPortSecurity 批量获取所有端口的安全配置
func (s *PortCollectionService) getAllPortSecurity(wrapper *device.ScrapliWrapper, vendor models.DeviceVendor) (map[string]PortSecurityInfo, error) {
	command := s.getPortSecurityBatchCommand(vendor)

	resp, err := wrapper.SendCommand(command, true)
	if err != nil {
		return nil, fmt.Errorf("发送端口安全命令失败: %w", err)
	}

	templatePath := s.getPortSecurityBatchTemplate(vendor)
	fsm, err := s.templateCache.Get(templatePath)
	if err != nil {
		return nil, fmt.Errorf("加载端口安全模板失败: %w", err)
	}

	records, err := fsm.ParseText(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("解析端口安全输出失败: %w", err)
	}

	// 如果返回0条记录，说明设备不支持或未配置端口安全
	if len(records) == 0 {
		log.Printf("[端口采集] 设备不支持端口安全或未配置（厂商:%s）", vendor)
		return make(map[string]PortSecurityInfo), nil
	}

	result := make(map[string]PortSecurityInfo)

	// 根据厂商不同，解析方式也不同
	if vendor == models.VendorRuijie || vendor == models.VendorMaipu {
		// 锐捷/迈普：输出是MAC地址表，需要统计每个端口的MAC数量
		portMACCount := make(map[string]int)
		for _, record := range records {
			portName := normalizeInterfaceName(record["PORT"])
			portMACCount[portName]++
		}

		// 构建结果（有MAC地址的端口表示启用了端口安全）
		for portName, count := range portMACCount {
			result[portName] = PortSecurityInfo{
				Enabled:    true,
				CurrentMAC: count,
				// MaxMAC需要从其他地方获取，这里先设为默认值
				MaxMAC: 0, // 需要配合端口配置信息
			}
		}
	} else {
		// 华为/H3C：直接输出端口配置（如果支持的话）
		for _, record := range records {
			ifaceName := normalizeInterfaceName(record["INTERFACE"])
			maxMAC, _ := strconv.Atoi(record["MAX_MAC"])
			currentMAC, _ := strconv.Atoi(record["CURRENT_MAC"])

			result[ifaceName] = PortSecurityInfo{
				Enabled:      record["ENABLED"] == "enabled",
				SecurityMode: record["SECURITY_MODE"],
				MaxMAC:       maxMAC,
				CurrentMAC:   currentMAC,
			}
		}
	}
	return result, nil
}

// getDot1xBatchCommand 获取批量802.1X命令（基于实际设备命令）
func (s *PortCollectionService) getDot1xBatchCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display dot1x",           // 华为实际命令
		models.VendorH3C:     "display dot1x",           // H3C实际命令
		models.VendorRuijie:  "show dot1x port-control", // 锐捷实际命令（输出所有端口）
		models.VendorMaipu:   "show dot1x port-control", // 迈普使用锐捷格式
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "display dot1x"
}

// getPortSecurityBatchCommand 获取批量端口安全命令
func (s *PortCollectionService) getPortSecurityBatchCommand(vendor models.DeviceVendor) string {
	// 注意：华为和锐捷设备都不支持或未配置端口安全
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display port-security",   // 华为可能不支持
		models.VendorH3C:     "display port-security",   // H3C可能不支持
		models.VendorRuijie:  "show port-security all",  // 锐捷命令（但输出为空）
		models.VendorMaipu:   "show port-security all",  // 迈普使用锐捷格式
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "display port-security"
}

// getDot1xBatchTemplate 获取批量802.1X模板路径
func (s *PortCollectionService) getDot1xBatchTemplate(vendor models.DeviceVendor) string {
	templates := map[models.DeviceVendor]string{
		models.VendorHuawei: "templates/huawei_vrp_display_dot1x.textfsm",
		models.VendorH3C:     "templates/huawei_vrp_display_dot1x.textfsm",
		models.VendorRuijie:  "templates/ruijie_os_show_dot1x.textfsm",
		models.VendorMaipu:   "templates/ruijie_os_show_dot1x.textfsm",
	}
	if tmpl, ok := templates[vendor]; ok {
		return tmpl
	}
	return "templates/huawei_vrp_display_dot1x.textfsm"
}

// getPortSecurityBatchTemplate 获取批量端口安全模板路径
func (s *PortCollectionService) getPortSecurityBatchTemplate(vendor models.DeviceVendor) string {
	templates := map[models.DeviceVendor]string{
		models.VendorHuawei: "templates/huawei_vrp_display_port-security.textfsm",
		models.VendorH3C:     "templates/huawei_vrp_display_port-security.textfsm",
		models.VendorRuijie:  "templates/ruijie_os_show_port-security.textfsm",
		models.VendorMaipu:   "templates/ruijie_os_show_port-security.textfsm",
	}
	if tmpl, ok := templates[vendor]; ok {
		return tmpl
	}
	return "templates/huawei_vrp_display_port-security.textfsm"
}

// ReloadConfig 重新加载配置（从数据库）
func (s *PortCollectionService) ReloadConfig() {
	s.loadConfigFromDB()
}
