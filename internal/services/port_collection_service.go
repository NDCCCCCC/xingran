package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/templates"
)

// PortCollectionService 端口状态采集服务
type PortCollectionService struct {
	db            *gorm.DB
	executor      *device.DeviceExecutor
	maxConcurrent int
}

// 端口采集并发数的默认配置键
const portConcurrentConfigKey = "network.port.collection.concurrent"

// NewPortCollectionService 创建端口采集服务
func NewPortCollectionService(db *gorm.DB, executor *device.DeviceExecutor) *PortCollectionService {
	svc := &PortCollectionService{
		db:            db,
		executor:      executor,
		maxConcurrent: 10, // 默认值
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
	log.Printf("[端口采集] 开始采集设备 %s (%s)", device.DeviceName, device.IPAddress)

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
	log.Printf("[端口采集] 设备 %s: 获取连接...", device.DeviceName)
	conn, err := pool.GetConnection(ctx, device.ID)
	if err != nil {
		log.Printf("[端口采集] 设备 %s: 获取连接失败: %v", device.DeviceName, err)
		result.ErrorMessage = err.Error()
		return result
	}

	// 增加引用计数
	if !conn.Acquire() {
		log.Printf("[端口采集] 设备 %s: 无法获取连接锁", device.DeviceName)
		result.ErrorMessage = "无法获取连接锁"
		return result
	}
	defer conn.Release()

	// 获取 wrapper
	wrapper := conn.GetWrapper()
	if wrapper == nil {
		log.Printf("[端口采集] 设备 %s: wrapper 为空", device.DeviceName)
		result.ErrorMessage = "wrapper 为空"
		return result
	}

	log.Printf("[端口采集] 设备 %s: 连接成功", device.DeviceName)

	// 获取接口列表命令
	interfaceCmd := s.getInterfaceCommand(device.Vendor)
	log.Printf("[端口采集] 设备 %s: 使用接口命令: %s", device.DeviceName, interfaceCmd)

	// 发送命令获取接口信息
	log.Printf("[端口采集] 设备 %s: 发送命令获取接口信息...", device.DeviceName)
	interfaceResp, err := wrapper.SendCommand(interfaceCmd, true)
	if err != nil {
		log.Printf("[端口采集] 设备 %s: 发送命令失败: %v", device.DeviceName, err)
		result.ErrorMessage = err.Error()
		return result
	}
	log.Printf("[端口采集] 设备 %s: 命令响应长度: %d 字符", device.DeviceName, len(interfaceResp.Result))

	// 解析接口列表（物理属性）
	log.Printf("[端口采集] 设备 %s: 开始解析接口列表...", device.DeviceName)
	interfaces, err := s.parseInterfaceList(interfaceResp.Result, device.Vendor)
	if err != nil {
		log.Printf("[端口采集] 设备 %s: 解析接口列表失败: %v", device.DeviceName, err)
		result.ErrorMessage = err.Error()
		return result
	}
	log.Printf("[端口采集] 设备 %s: 解析到 %d 个接口", device.DeviceName, len(interfaces))

	// 获取接口状态和描述信息（仅锐捷/迈普设备）
	descriptionMap := make(map[string]InterfaceDescription)
	if device.Vendor == models.VendorRuijie || device.Vendor == models.VendorMaipu {
		descs, err := s.parseInterfaceDescriptions(wrapper, device.Vendor)
		if err == nil {
			descriptionMap = descs
			log.Printf("[端口采集] 获取到 %d 个接口状态和描述", len(descs))
		} else {
			log.Printf("[端口采集] 获取接口状态和描述失败: %v", err)
		}
	}

	// 获取802.1X和端口安全信息
	collectionTime := time.Now()

	// 收集所有端口状态数据
	var portStatuses []*models.DevicePortStatus

	log.Printf("[端口采集] 开始处理接口数据，物理属性解析到 %d 个接口", len(interfaces))
	if len(interfaces) == 0 && len(descriptionMap) > 0 {
		// 物理属性解析失败，但状态/描述解析成功
		// 这种情况通常发生在路由器上（不支持 show interfaces status 命令）
		// 使用降级方案：仅基于状态/描述数据创建端口记录（不包含 VLAN/Duplex/Speed/Type）
		log.Printf("[端口采集] 物理属性解析为空，但状态/描述有 %d 条记录，使用降级方案", len(descriptionMap))
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

			// 获取802.1X状态（路由器可能不支持，但不影响采集）
			dot1xEnabled, dot1xPortStatusStr := s.getDot1xStatus(wrapper, ifaceName, device.Vendor)
			portStatus.Dot1xEnabled = dot1xEnabled
			switch strings.ToUpper(dot1xPortStatusStr) {
			case "AUTHORIZED":
				portStatus.Dot1xPortStatus = models.Dot1xStatusAuthorized
			case "UNAUTHORIZED":
				portStatus.Dot1xPortStatus = models.Dot1xStatusUnauthorized
			default:
				portStatus.Dot1xPortStatus = models.Dot1xStatusUnknown
			}

			// 获取端口安全配置
			securityEnabled, securityModeStr, maxMAC, currentMAC := s.getPortSecurity(wrapper, ifaceName, device.Vendor)
			portStatus.PortSecurityEnabled = securityEnabled

			var securityMode models.PortSecurityMode
			switch strings.ToUpper(securityModeStr) {
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

			if maxMAC > 0 {
				portStatus.MaxMACCount = &maxMAC
			}
			if currentMAC > 0 {
				portStatus.CurrentMACCount = &currentMAC
			}

			portStatuses = append(portStatuses, portStatus)
		}
		log.Printf("[端口采集] 降级方案创建了 %d 个端口记录（基于状态/描述数据）", len(portStatuses))
	} else {
		// 正常流程：基于物理属性数据创建端口记录
		if len(interfaces) == 0 {
			log.Printf("[端口采集] 警告：interfaces 切片为空！descriptionMap 有 %d 条记录", len(descriptionMap))
			// 打印 descriptionMap 的所有 key 用于调试
			for k := range descriptionMap {
				log.Printf("[端口采集] descriptionMap key: %s", k)
			}
		}

		for i, iface := range interfaces {
			// 标准化接口名称
			normalizedName := normalizeInterfaceName(iface.Name)

			log.Printf("[端口采集] 处理接口 %d: 原始名=%s, 标准化名=%s", i, iface.Name, normalizedName)

			// 从状态和描述映射中获取状态信息（使用标准化名称查找）
			var adminStatus, operStatus, description string
			if desc, ok := descriptionMap[normalizedName]; ok {
				adminStatus = desc.AdminStatus
				operStatus = desc.OperStatus
				description = desc.Description
			} else {
				log.Printf("[端口采集] 警告：未找到接口 %s 的状态信息", normalizedName)
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

			// 获取802.1X状态（使用标准化名称）
			dot1xEnabled, dot1xPortStatusStr := s.getDot1xStatus(wrapper, normalizedName, device.Vendor)
			portStatus.Dot1xEnabled = dot1xEnabled
			switch strings.ToUpper(dot1xPortStatusStr) {
			case "AUTHORIZED":
				portStatus.Dot1xPortStatus = models.Dot1xStatusAuthorized
			case "UNAUTHORIZED":
				portStatus.Dot1xPortStatus = models.Dot1xStatusUnauthorized
			default:
				portStatus.Dot1xPortStatus = models.Dot1xStatusUnknown
			}

			// 获取端口安全配置（使用标准化名称）
			securityEnabled, securityModeStr, maxMAC, currentMAC := s.getPortSecurity(wrapper, normalizedName, device.Vendor)
			portStatus.PortSecurityEnabled = securityEnabled

			var securityMode models.PortSecurityMode
			switch strings.ToUpper(securityModeStr) {
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

			if maxMAC > 0 {
				portStatus.MaxMACCount = &maxMAC
			}
			if currentMAC > 0 {
				portStatus.CurrentMACCount = &currentMAC
			}

			portStatuses = append(portStatuses, portStatus)
		}
	} // end of else (正常流程)

	// 批量保存到数据库（使用 Save 实现 upsert）
	successCount := 0
	failedCount := 0

	log.Printf("[端口采集] 设备 %s 开始保存 %d 个端口数据", device.DeviceName, len(portStatuses))

	for i, portStatus := range portStatuses {
		// 检查 context 是否已取消，但不要立即退出
		// 使用一个新的 context 来避免 context 取消导致保存失败
		saveCtx := context.Background()
		if ctx.Err() != nil {
			log.Printf("[端口采集] 设备 %s 保存端口时 context 已取消，使用新 context 继续保存 (已保存 %d/%d)",
				device.DeviceName, successCount, len(portStatuses))
		}

		// 先查找是否存在
		var existing models.DevicePortStatus
		err := s.db.WithContext(saveCtx).
			Where("device_id = ? AND interface_name = ?", portStatus.DeviceID, portStatus.InterfaceName).
			First(&existing).Error

		if err == nil {
			// 记录存在，更新
			portStatus.ID = existing.ID
			portStatus.CreatedAt = existing.CreatedAt // 保留创建时间
			if err := s.db.WithContext(saveCtx).Save(portStatus).Error; err == nil {
				successCount++
				if i == 0 || i == len(portStatuses)-1 || i%50 == 49 {
					log.Printf("[端口采集] 设备 %s 更新端口进度: %d/%d",
						device.DeviceName, i+1, len(portStatuses))
				}
			} else {
				failedCount++
				log.Printf("[端口采集] 设备 %s 更新端口失败 %s: %v",
					device.DeviceName, portStatus.InterfaceName, err)
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			// 记录不存在，新增
			if err := s.db.WithContext(saveCtx).Create(portStatus).Error; err == nil {
				successCount++
				if i == 0 || i == len(portStatuses)-1 || i%50 == 49 {
					log.Printf("[端口采集] 设备 %s 新增端口进度: %d/%d",
						device.DeviceName, i+1, len(portStatuses))
				}
			} else {
				failedCount++
				log.Printf("[端口采集] 设备 %s 新增端口失败 %s: %v",
					device.DeviceName, portStatus.InterfaceName, err)
			}
		} else {
			failedCount++
			// 详细的错误日志
			if ctx.Err() != nil {
				log.Printf("[端口采集] 设备 %s 查询端口失败 %s: %v (context 已取消)",
					device.DeviceName, portStatus.InterfaceName, err)
			} else {
				log.Printf("[端口采集] 设备 %s 查询端口失败 %s: %v",
					device.DeviceName, portStatus.InterfaceName, err)
			}
		}
	}

	log.Printf("[端口采集] 设备 %s 保存完成: 成功=%d, 失败=%d, 总数=%d",
		device.DeviceName, successCount, failedCount, len(portStatuses))

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
	log.Printf("[端口采集] 开始解析物理属性，厂商: %v", vendor)

	// 获取模板文件路径
	templatePath := s.getInterfaceTemplate(vendor)
	log.Printf("[端口采集] 使用模板: %s", templatePath)

	// 解析模板
	fsm, err := templates.ParseTemplate(templatePath)
	if err != nil {
		log.Printf("[端口采集] 解析模板失败: %v", err)
		return nil, err
	}

	// 使用模板解析输出
	records, err := fsm.ParseText(output)
	if err != nil {
		log.Printf("[端口采集] 解析输出失败: %v", err)
		return nil, err
	}

	log.Printf("[端口采集] 物理属性解析成功，textfsm 提取到 %d 条记录", len(records))

	// 转换为 InterfaceInfo（只包含物理属性）
	var interfaces []InterfaceInfo
	for i, record := range records {
		info := InterfaceInfo{}

		// 打印所有字段用于调试（前3条记录）
		if i < 3 {
			log.Printf("[端口采集] Record %d 原始数据: %+v", i, record)
		}

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
			log.Printf("[端口采集] Record %d 没有接口名称字段，跳过。可用字段: %v", i, getAvailableKeys(record))
			continue
		}

		// 提取并标准化接口名称
		originalName := strings.ReplaceAll(name, " ", "")
		info.Name = normalizeInterfaceName(originalName)

		log.Printf("[端口采集] Record %d: 原始名=%s, 标准化名=%s", i, name, info.Name)

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

	log.Printf("[端口采集] 物理属性解析完成，共 %d 个有效接口", len(interfaces))
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
	log.Printf("[端口采集] 使用简单解析模式")

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
					log.Printf("[端口采集] 简单解析提取: %s", info.Name)
				}
			}
		}
	}

	log.Printf("[端口采集] 简单解析完成，提取 %d 个接口", len(interfaces))
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
		log.Printf("[端口查询] 筛选设备ID: %s", deviceID)
	}
	if interfaceName != "" {
		query = query.Where("interface_name LIKE ?", "%"+interfaceName+"%")
		log.Printf("[端口查询] 筛选接口名: %s", interfaceName)
	}
	if adminStatus != "" {
		query = query.Where("admin_status = ?", adminStatus)
		log.Printf("[端口查询] 筛选管理状态: %s", adminStatus)
	}
	if operStatus != "" {
		query = query.Where("oper_status = ?", operStatus)
		log.Printf("[端口查询] 筛选操作状态: %s", operStatus)
	}
	if dot1xEnabled != nil {
		query = query.Where("dot1x_enabled = ?", *dot1xEnabled)
		log.Printf("[端口查询] 筛选802.1X: %v", *dot1xEnabled)
	}
	if portSecurityEnabled != nil {
		query = query.Where("port_security_enabled = ?", *portSecurityEnabled)
		log.Printf("[端口查询] 筛选端口安全: %v", *portSecurityEnabled)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询端口状态总数失败: %w", err)
	}

	log.Printf("[端口查询] 查询结果: 总数=%d, 当前页=%d, 每页=%d", total, current, pageSize)

	// 分页查询
	offset := (current - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("collected_at DESC").Find(&portStatuses).Error; err != nil {
		return nil, 0, fmt.Errorf("查询端口状态列表失败: %w", err)
	}

	log.Printf("[端口查询] 返回 %d 条记录", len(portStatuses))
	if len(portStatuses) > 0 {
		log.Printf("[端口查询] 首条记录: 设备ID=%s, 接口名=%s, 采集时间=%s",
			portStatuses[0].DeviceID, portStatuses[0].InterfaceName, portStatuses[0].CollectedAt.Format("2006-01-02 15:04:05"))
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

	log.Printf("[诊断] 设备ID=%s, 设备名=%s", device.ID, device.DeviceName)

	// 检查该设备的端口数据
	var portData []models.DevicePortStatus
	if err := s.db.WithContext(ctx).Where("device_id = ?", device.ID).Find(&portData).Error; err != nil {
		log.Printf("[诊断] 查询端口数据失败: %v", err)
		return
	}

	log.Printf("[诊断] 找到 %d 条端口数据", len(portData))
	if len(portData) == 0 {
		// 检查是否有其他设备的端口名称相同的情况
		var allPorts []models.DevicePortStatus
		s.db.WithContext(ctx).Find(&allPorts)
		log.Printf("[诊断] 数据库中共有 %d 条端口记录", len(allPorts))
		for _, p := range allPorts {
			log.Printf("[诊断] 端口记录: 设备ID=%s, 接口名=%s, 采集时间=%s",
				p.DeviceID, p.InterfaceName, p.CollectedAt.Format("2006-01-02 15:04:05"))
		}
	} else {
		for i, p := range portData {
			if i < 5 || i >= len(portData)-2 { // 只打印前5条和最后2条
				log.Printf("[诊断] 端口%d: 接口名=%s, 操作状态=%s, 采集时间=%s",
					i+1, p.InterfaceName, p.OperStatus, p.CollectedAt.Format("2006-01-02 15:04:05"))
			}
		}
	}
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

	// 打印原始输出（前 500 字符用于调试）
	rawOutput := resp.Result
	if len(rawOutput) > 500 {
		log.Printf("[端口采集] sh int des 原始输出（前500字符）:\n%s", rawOutput[:500])
	} else {
		log.Printf("[端口采集] sh int des 原始输出:\n%s", rawOutput)
	}

	templatePath := s.getInterfaceDescriptionTemplate(vendor)
	log.Printf("[端口采集] 使用接口描述模板: %s", templatePath)

	// 解析模板
	fsm, err := templates.ParseTemplate(templatePath)
	if err != nil {
		log.Printf("[端口采集] 解析接口描述模板失败: %v", err)
		return nil, err
	}

	// 使用模板解析输出
	records, err := fsm.ParseText(resp.Result)
	if err != nil {
		log.Printf("[端口采集] 解析接口描述输出失败: %v", err)
		return nil, err
	}

	log.Printf("[端口采集] 状态和描述解析成功，提取到 %d 条记录", len(records))

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
			log.Printf("[端口采集] 状态描述: %s, Oper=%s, Admin=%s, Desc=%s",
				desc.InterfaceName, desc.OperStatus, desc.AdminStatus, desc.Description)
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

// ReloadConfig 重新加载配置（从数据库）
func (s *PortCollectionService) ReloadConfig() {
	s.loadConfigFromDB()
}
