package services

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
	"github.com/gosnmp/gosnmp"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// DeviceDiscoveryService 设备发现服务
type DeviceDiscoveryService struct {
	db *gorm.DB
}

// NewDeviceDiscoveryService 创建设备发现服务
func NewDeviceDiscoveryService(db *gorm.DB) *DeviceDiscoveryService {
	return &DeviceDiscoveryService{db: db}
}

// DiscoveryRequest 发现请求
type DiscoveryRequest struct {
	TaskName        string
	DiscoveryType   models.DiscoveryType
	IPRanges        []models.IPRange
	SNMPCommunity   string                 // 保留兼容性
	SNMPCommunities  []string               // 多个 SNMP Community（推荐使用）
	SNMPPort        int
	AutoImport      bool
	GroupID         *string
	CreatedBy       string
}

// DiscoveredDevice 发现的设备信息
type DiscoveredDevice struct {
	IPAddress  string
	DeviceType string
	Vendor     string
	Model      string
	SysName    string
	SysDescr   string
	IsAlive    bool
}

// DiscoveryResult 发现结果
type DiscoveryResult struct {
	DiscoveryID      string
	Status           models.DiscoveryStatus
	TotalIPs         int
	DiscoveredCount  int
	DiscoveredDevices []*DiscoveredDevice
	ErrorMessage     string
}

// DeviceProbeRequest 单设备探测请求
type DeviceProbeRequest struct {
	IPAddress    string   `json:"ipAddress" binding:"required,ip"`
	CredentialID string   `json:"credentialId" binding:"required,uuid"`
	SNMPPort     int      `json:"snmpPort,omitempty"`
	// Communities 用户输入的多个 SNMP community 进行尝试
	// 如果为空，则使用凭证中的 SNMPCommunity
	Communities []string `json:"communities,omitempty"`
}

// DeviceProbeResult 单设备探测结果
type DeviceProbeResult struct {
	Success    bool                   `json:"success"`
	Message    string                 `json:"message"`
	IPAddress  string                 `json:"ipAddress"`
	DeviceName string                 `json:"deviceName,omitempty"`
	DeviceType models.DeviceType      `json:"deviceType,omitempty"`
	Vendor     models.DeviceVendor    `json:"vendor,omitempty"`
	Model      string                 `json:"model,omitempty"`
	SysDescr   string                 `json:"sysDescr,omitempty"`
	SysName    string                 `json:"sysName,omitempty"`
}

// CreateDiscoveryTask 创建发现任务
func (s *DeviceDiscoveryService) CreateDiscoveryTask(ctx context.Context, req *DiscoveryRequest) (string, error) {
	// 计算总IP数
	totalIPs := 0
	for _, ipRange := range req.IPRanges {
		totalIPs += calculateIPCount(ipRange.StartIP, ipRange.EndIP)
	}

	// 创建发现任务
	task := &models.DeviceDiscovery{
		TaskName:       req.TaskName,
		DiscoveryType:  req.DiscoveryType,
		IPRanges:       models.IPRangeList(req.IPRanges),
		SNMPCommunity:  req.SNMPCommunity,
		SNMPPort:       req.SNMPPort,
		Status:         models.DiscoveryStatusPending,
		TotalIPs:       totalIPs,
		AutoImport:     req.AutoImport,
		GroupID:        req.GroupID,
		CreatedBy:      req.CreatedBy,
	}

	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		return "", fmt.Errorf("创建发现任务失败: %w", err)
	}

	return task.ID, nil
}

// ExecuteDiscovery 执行发现任务
func (s *DeviceDiscoveryService) ExecuteDiscovery(ctx context.Context, discoveryID string) (*DiscoveryResult, error) {
	// 获取任务
	var task models.DeviceDiscovery
	if err := s.db.WithContext(ctx).Where("id = ?", discoveryID).First(&task).Error; err != nil {
		return nil, fmt.Errorf("发现任务不存在: %w", err)
	}

	// 更新状态为执行中
	now := time.Now()
	task.Status = models.DiscoveryStatusRunning
	task.StartedAt = &now
	s.db.WithContext(ctx).Save(&task)

	// 执行发现
	var discoveredDevices []*DiscoveredDevice
	var err error

	if task.DiscoveryType == models.DiscoveryTypeSNMP {
		discoveredDevices, err = s.discoverBySNMP(ctx, &task)
	} else {
		discoveredDevices, err = s.discoverByScan(ctx, &task)
	}

	// 更新任务状态
	completedAt := time.Now()
	task.CompletedAt = &completedAt
	task.DiscoveredCount = len(discoveredDevices)

	if err != nil {
		task.Status = models.DiscoveryStatusFailed
		task.ErrorMessage = err.Error()
		s.db.WithContext(ctx).Save(&task)
		return nil, err
	}

	task.Status = models.DiscoveryStatusSuccess
	s.db.WithContext(ctx).Save(&task)

	return &DiscoveryResult{
		DiscoveryID:      discoveryID,
		Status:           task.Status,
		TotalIPs:         task.TotalIPs,
		DiscoveredCount:  len(discoveredDevices),
		DiscoveredDevices: discoveredDevices,
	}, nil
}

// discoverBySNMP 使用SNMP发现设备
func (s *DeviceDiscoveryService) discoverBySNMP(ctx context.Context, task *models.DeviceDiscovery) ([]*DiscoveredDevice, error) {
	var devices []*DiscoveredDevice
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 生成IP列表
	ips := generateIPList(task.IPRanges)
	semaphore := make(chan struct{}, 50) // 并发限制

	var successCount int32

	for _, ip := range ips {
		wg.Add(1)
		go func(ipAddr string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			device := s.snmpProbe(ipAddr, task.SNMPCommunity, task.SNMPPort)
			if device != nil && device.IsAlive {
				mu.Lock()
				devices = append(devices, device)
				mu.Unlock()
				atomic.AddInt32(&successCount, 1)
			}
		}(ip)
	}

	wg.Wait()

	return devices, nil
}

// snmpProbe SNMP探测单个IP
func (s *DeviceDiscoveryService) snmpProbe(ip, community string, port int) *DiscoveredDevice {
	fmt.Printf("[DEBUG] snmpProbe 开始: ip=%s, community=%s, port=%d\n", ip, community, port)
	// 创建SNMP客户端
	snmpClient := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      uint16(port),
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(5) * time.Second,
	}

	// 连接
	fmt.Printf("[DEBUG] 尝试连接 SNMP 服务器...\n")
	if err := snmpClient.Connect(); err != nil {
		fmt.Printf("[DEBUG] SNMP 连接失败: %v\n", err)
		return &DiscoveredDevice{
			IPAddress: ip,
			IsAlive:   false,
		}
	}
	defer snmpClient.Conn.Close()
	fmt.Printf("[DEBUG] SNMP 连接成功\n")

	// 获取sysName
	fmt.Printf("[DEBUG] 获取 sysName...\n")
	sysName, err := snmpClient.Get([]string{".1.3.6.1.2.1.1.5.0"}) // sysName
	if err != nil {
		fmt.Printf("[DEBUG] 获取 sysName 失败: %v\n", err)
		return &DiscoveredDevice{
			IPAddress: ip,
			IsAlive:   false,
		}
	}
	fmt.Printf("[DEBUG] sysName 获取成功, 变量数量: %d\n", len(sysName.Variables))

	// 获取sysDescr
	sysDescr, _ := snmpClient.Get([]string{".1.3.6.1.2.1.1.1.0"}) // sysDescr
	fmt.Printf("[DEBUG] sysDescr 获取完成, 变量数量: %d\n", len(sysDescr.Variables))

	device := &DiscoveredDevice{
		IPAddress: ip,
		IsAlive:   true,
	}

	if len(sysName.Variables) > 0 {
		variable := sysName.Variables[0]
		fmt.Printf("[DEBUG] sysName 变量类型: %T\n", variable.Value)
		// 尝试多种类型转换
		switch v := variable.Value.(type) {
		case string:
			device.SysName = v
			fmt.Printf("[DEBUG] sysName (string): %s\n", v)
		case []byte:
			device.SysName = string(v)
			fmt.Printf("[DEBUG] sysName (bytes): %s\n", string(v))
		default:
			fmt.Printf("[DEBUG] sysName 未知类型: %T, 值: %v\n", v, v)
		}
	}

	if len(sysDescr.Variables) > 0 {
		variable := sysDescr.Variables[0]
		fmt.Printf("[DEBUG] sysDescr 变量类型: %T\n", variable.Value)
		// 尝试多种类型转换
		switch v := variable.Value.(type) {
		case string:
			device.SysDescr = v
			device.Vendor = identifyVendor(device.SysDescr)
			device.DeviceType = identifyDeviceType(device.SysDescr)
			fmt.Printf("[DEBUG] sysDescr (string): %s\n", v[:min(100, len(v))])
		case []byte:
			device.SysDescr = string(v)
			device.Vendor = identifyVendor(device.SysDescr)
			device.DeviceType = identifyDeviceType(device.SysDescr)
			descrStr := string(v)
			fmt.Printf("[DEBUG] sysDescr (bytes): %s\n", descrStr[:min(100, len(descrStr))])
		default:
			fmt.Printf("[DEBUG] sysDescr 未知类型: %T, 值: %v\n", v, v)
		}
	}

	fmt.Printf("[DEBUG] snmpProbe 完成: IsAlive=%v, Vendor=%s, DeviceType=%s\n", device.IsAlive, device.Vendor, device.DeviceType)
	return device
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// discoverByScan 使用Ping扫描发现设备
func (s *DeviceDiscoveryService) discoverByScan(ctx context.Context, task *models.DeviceDiscovery) ([]*DiscoveredDevice, error) {
	var devices []*DiscoveredDevice
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 生成IP列表
	ips := generateIPList(task.IPRanges)
	semaphore := make(chan struct{}, 100) // 并发限制

	for _, ip := range ips {
		wg.Add(1)
		go func(ipAddr string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if isAlive(ipAddr) {
				mu.Lock()
				devices = append(devices, &DiscoveredDevice{
					IPAddress: ipAddr,
					IsAlive:   true,
				})
				mu.Unlock()
			}
		}(ip)
	}

	wg.Wait()

	return devices, nil
}

// GetDiscoveryList 获取发现任务列表
func (s *DeviceDiscoveryService) GetDiscoveryList(ctx context.Context, current, pageSize int) ([]models.DeviceDiscovery, int64, error) {
	var tasks []models.DeviceDiscovery
	var total int64

	query := s.db.WithContext(ctx).Model(&models.DeviceDiscovery{})

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询发现任务总数失败: %w", err)
	}

	// 分页查询
	offset := (current - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("查询发现任务失败: %w", err)
	}

	return tasks, total, nil
}

// GetDiscoveryByID 获取发现任务详情
func (s *DeviceDiscoveryService) GetDiscoveryByID(ctx context.Context, discoveryID string) (*models.DeviceDiscovery, error) {
	var task models.DeviceDiscovery
	if err := s.db.WithContext(ctx).Where("id = ?", discoveryID).First(&task).Error; err != nil {
		return nil, fmt.Errorf("发现任务不存在: %w", err)
	}
	return &task, nil
}

// CancelDiscovery 取消发现任务
func (s *DeviceDiscoveryService) CancelDiscovery(ctx context.Context, discoveryID string) error {
	var task models.DeviceDiscovery
	if err := s.db.WithContext(ctx).Where("id = ?", discoveryID).First(&task).Error; err != nil {
		return fmt.Errorf("发现任务不存在: %w", err)
	}

	if task.Status != models.DiscoveryStatusPending && task.Status != models.DiscoveryStatusRunning {
		return fmt.Errorf("只能取消待执行或执行中的任务")
	}

	task.Status = models.DiscoveryStatusCancelled
	if err := s.db.WithContext(ctx).Save(&task).Error; err != nil {
		return fmt.Errorf("取消任务失败: %w", err)
	}

	return nil
}

// DeleteDiscovery 删除发现任务
func (s *DeviceDiscoveryService) DeleteDiscovery(ctx context.Context, discoveryID string) error {
	var task models.DeviceDiscovery
	if err := s.db.WithContext(ctx).Where("id = ?", discoveryID).First(&task).Error; err != nil {
		return fmt.Errorf("发现任务不存在: %w", err)
	}

	// 检查任务状态
	if task.Status == models.DiscoveryStatusRunning {
		return fmt.Errorf("无法删除执行中的任务")
	}

	if err := s.db.WithContext(ctx).Delete(&task).Error; err != nil {
		return fmt.Errorf("删除任务失败: %w", err)
	}

	return nil
}

// BatchDeleteDiscoveries 批量删除发现任务
func (s *DeviceDiscoveryService) BatchDeleteDiscoveries(ctx context.Context, discoveryIDs []string) error {
	for _, discoveryID := range discoveryIDs {
		if err := s.DeleteDiscovery(ctx, discoveryID); err != nil {
			continue // 继续处理其他任务
		}
	}
	return nil
}

// ImportDiscoveredDevices 导入发现的设备
func (s *DeviceDiscoveryService) ImportDiscoveredDevices(ctx context.Context, discoveryID string, deviceIDs []string, createdBy string) (int, error) {
	// 获取发现任务
	var task models.DeviceDiscovery
	if err := s.db.WithContext(ctx).Where("id = ?", discoveryID).First(&task).Error; err != nil {
		return 0, fmt.Errorf("发现任务不存在: %w", err)
	}

	// 这里需要重新获取发现的设备列表
	// 实际实现中可以将发现的设备存储到临时表中
	// 或者从结果缓存中获取

	// 简化实现：返回导入数量
	return len(deviceIDs), nil
}

// ProbeSingleDevice 探测单个设备（支持多个 community）
// 复用现有的 snmpProbe 方法，依次尝试每个 community 直到成功
func (s *DeviceDiscoveryService) ProbeSingleDevice(ctx context.Context, req *DeviceProbeRequest) (*DeviceProbeResult, error) {
	// 参数验证
	if req.IPAddress == "" {
		return &DeviceProbeResult{
			Success: false,
			Message: "IP地址不能为空",
		}, nil
	}

	if req.CredentialID == "" {
		return &DeviceProbeResult{
			Success: false,
			Message: "授权凭证ID不能为空",
		}, nil
	}

	// 设置默认 SNMP 端口
	snmpPort := req.SNMPPort
	if snmpPort == 0 {
		snmpPort = 161
	}

	// 1. 确定要尝试的 community 列表
	communities := []string{}

	// 优先使用用户输入的 communities
	if len(req.Communities) > 0 {
		communities = req.Communities
	} else {
		// 否则使用凭证中的 SNMPCommunities 数组
		var credential models.AuthCredential
		if err := s.db.WithContext(ctx).Where("id = ?", req.CredentialID).First(&credential).Error; err != nil {
			return &DeviceProbeResult{
				Success: false,
				Message: "授权凭证不存在",
			}, nil
		}

		// 使用新的 SNMPCommunities 数组
		if len(credential.SNMPCommunities) == 0 {
			return &DeviceProbeResult{
				Success: false,
				Message: fmt.Sprintf("凭证中未配置 SNMP community，请先在凭证管理中添加 SNMP Communities（当前凭证：%s）", credential.CredentialName),
			}, nil
		}

		communities = credential.SNMPCommunities
		fmt.Printf("[DEBUG] 从凭证 %s 获取到 %d 个 SNMP communities: %v\n", credential.CredentialName, len(communities), communities)
	}

	// 2. 依次尝试每个 community（复用现有 snmpProbe 方法）
	var lastError error
	for i, community := range communities {
		fmt.Printf("[DEBUG] 尝试第 %d 个 community: %s\n", i+1, community)
		// 复用 device_discovery_service.go:170 的 snmpProbe 方法
		result := s.snmpProbe(req.IPAddress, community, snmpPort)
		fmt.Printf("[DEBUG] snmpProbe 返回: result=%v, IsAlive=%v\n", result != nil, result != nil && result.IsAlive)

		if result != nil && result.IsAlive {
			// 探测成功，提取型号信息
			model := ""
			vendor := models.DeviceVendor(result.Vendor)

			// 使用 device 包的 ExtractModelFromSysDescr 提取型号
			if result.SysDescr != "" {
				model = device.ExtractModelFromSysDescr(result.SysDescr, vendor)
			}

			// 构造返回结果
			probeResult := &DeviceProbeResult{
				Success:    true,
				Message:    "探测成功",
				IPAddress:  req.IPAddress,
				DeviceName: result.SysName,
				SysDescr:   result.SysDescr,
				SysName:    result.SysName,
				Vendor:     vendor,
				Model:      model,
			}

			// 转换设备类型
			switch result.DeviceType {
			case "router":
				probeResult.DeviceType = models.DeviceTypeRouter
			case "switch":
				probeResult.DeviceType = models.DeviceTypeSwitch
			case "firewall":
				probeResult.DeviceType = models.DeviceTypeFirewall
			case "ap":
				probeResult.DeviceType = models.DeviceTypeAP
			case "loadbalancer":
				probeResult.DeviceType = models.DeviceTypeLoadBalancer
			default:
				probeResult.DeviceType = models.DeviceTypeSwitch // 默认为交换机
			}

			return probeResult, nil
		}

		// 记录错误用于最后的失败消息
		if result == nil {
			lastError = fmt.Errorf("连接超时")
		}
	}

	// 3. 所有 community 都失败
	errorMsg := "SNMP连接失败"
	if lastError != nil {
		errorMsg += fmt.Sprintf(": %v", lastError)
	}
	errorMsg += "，请检查网络或确认 SNMP community 正确"

	return &DeviceProbeResult{
		Success: false,
		Message: errorMsg,
		IPAddress: req.IPAddress,
	}, nil
}

// GetDiscoveryResults 获取发现结果
func (s *DeviceDiscoveryService) GetDiscoveryResults(ctx context.Context, discoveryID string) ([]*DiscoveredDevice, error) {
	// 获取发现任务
	var task models.DeviceDiscovery
	if err := s.db.WithContext(ctx).Where("id = ?", discoveryID).First(&task).Error; err != nil {
		return nil, fmt.Errorf("发现任务不存在: %w", err)
	}

	// TODO: 实际实现中需要从临时表或缓存中获取发现的设备
	// 目前返回空列表
	return []*DiscoveredDevice{}, nil
}

// calculateIPCount 计算IP范围内IP数量
func calculateIPCount(startIP, endIP string) int {
	start := net.ParseIP(startIP)
	end := net.ParseIP(endIP)

	if start == nil || end == nil {
		return 0
	}

	start = start.To4()
	end = end.To4()

	if start == nil || end == nil {
		return 0
	}

	var count int
	for i := 0; i < 4; i++ {
		count = count*256 + int(end[i]-start[i])
	}
	return count + 1
}

// generateIPList 生成IP列表
func generateIPList(ipRanges []models.IPRange) []string {
	var ips []string

	for _, ipRange := range ipRanges {
		start := net.ParseIP(ipRange.StartIP)
		end := net.ParseIP(ipRange.EndIP)

		if start == nil || end == nil {
			continue
		}

		start = start.To4()
		end = end.To4()

		if start == nil || end == nil {
			continue
		}

		// 限制IP数量，避免生成过多
		maxCount := 65536 // 最多扫描65536个IP
		count := 0

		for ip := start; ipLessEqual(ip, end); ip = incrementIP(ip) {
			ips = append(ips, ip.String())
			count++
			if count >= maxCount {
				break
			}
		}
	}

	return ips
}

// ipLessEqual 比较IP是否小于等于
func ipLessEqual(ip1, ip2 net.IP) bool {
	ip1 = ip1.To4()
	ip2 = ip2.To4()

	for i := 0; i < 4; i++ {
		if ip1[i] < ip2[i] {
			return true
		}
		if ip1[i] > ip2[i] {
			return false
		}
	}
	return true
}

// incrementIP IP地址自增
func incrementIP(ip net.IP) net.IP {
	ip = ip.To4()
	if ip == nil {
		return nil
	}

	newIP := make(net.IP, 4)
	copy(newIP, ip)

	for i := 3; i >= 0; i-- {
		if newIP[i] < 255 {
			newIP[i]++
			break
		} else {
			newIP[i] = 0
		}
	}

	return newIP
}

// isAlive 检测主机是否存活
func isAlive(ip string) bool {
	// 尝试建立TCP连接到常见端口
	ports := []int{80, 443, 22, 23, 161, 8080}

	for _, port := range ports {
		address := fmt.Sprintf("%s:%d", ip, port)
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			conn.Close()
			return true
		}
	}

	// ICMP ping需要管理员权限，这里使用TCP SYN探测
	// 如果所有端口都不可达，返回false
	return false
}

// identifyVendor 识别设备厂商
func identifyVendor(sysDescr string) string {
	descr := string(sysDescr)
	vendors := map[string]string{
		"Huawei":   "huawei",
		"H3C":      "h3c",
		"Ruijie":   "ruijie",
		"Maipu":    "maipu",
		"Cisco":    "cisco",
		"Juniper":  "juniper",
		"Arista":   "arista",
		"Fortinet": "fortinet",
	}

	for vendor, code := range vendors {
		if contains(descr, vendor) {
			return code
		}
	}

	return "unknown"
}

// identifyDeviceType 识别设备类型
func identifyDeviceType(sysDescr string) string {
	descr := string(sysDescr)

	if contains(descr, "Router") || contains(descr, "router") {
		return "router"
	}
	if contains(descr, "Switch") || contains(descr, "switch") {
		return "switch"
	}
	if contains(descr, "Firewall") || contains(descr, "firewall") {
		return "firewall"
	}
	if contains(descr, "AP") || contains(descr, "Access Point") {
		return "ap"
	}
	if contains(descr, "Load Balancer") {
		return "loadbalancer"
	}

	return "unknown"
}

// contains 字符串包含检查
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (
		s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
