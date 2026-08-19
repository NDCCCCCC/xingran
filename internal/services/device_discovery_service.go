package services

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gosnmp/gosnmp"
	devicepkg "github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// DeviceDiscoveryService 设备发现服务
type DeviceDiscoveryService struct {
	db *gorm.DB
}

// NewDeviceDiscoveryService 创建设备发现服务
func NewDeviceDiscoveryService(db *gorm.DB) *DeviceDiscoveryService {
	return &DeviceDiscoveryService{db: db}
}

// DiscoveryStatistics 设备发现统计结果。
// 状态机由 models.DiscoveryStatus 定义。
type DiscoveryStatistics struct {
	Total        int64 `json:"total"`
	Pending      int64 `json:"pending"`      // DiscoveryStatusPending
	Running      int64 `json:"running"`      // DiscoveryStatusRunning
	Completed    int64 `json:"completed"`    // DiscoveryStatusSuccess
	Failed       int64 `json:"failed"`       // DiscoveryStatusFailed
	TotalDevices int64 `json:"totalDevices"` // SUM(discovered_count)
}

// GetStatistics 统计设备发现任务总数/各状态计数及累计发现设备数。
// 用条件聚合避免加载全量行进内存; totalDevices 用 COALESCE(SUM(discovered_count),0)。
func (s *DeviceDiscoveryService) GetStatistics(ctx context.Context) (*DiscoveryStatistics, error) {
	var result DiscoveryStatistics
	err := s.db.WithContext(ctx).Model(&models.DeviceDiscovery{}).
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS pending", int(models.DiscoveryStatusPending)),
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS running", int(models.DiscoveryStatusRunning)),
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS completed", int(models.DiscoveryStatusSuccess)),
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS failed", int(models.DiscoveryStatusFailed)),
			"COALESCE(SUM(discovered_count), 0) AS total_devices",
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计设备发现失败: %w", err)
	}
	return &result, nil
}

// DiscoveryRequest 发现请求
type DiscoveryRequest struct {
	TaskName        string
	DiscoveryType   models.DiscoveryType
	IPRanges        []models.IPRange
	SNMPCommunity   string   // 保留兼容性
	SNMPCommunities []string // 多个 SNMP Community（推荐使用）
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
	DiscoveryID       string
	Status            models.DiscoveryStatus
	TotalIPs          int
	DiscoveredCount   int
	DiscoveredDevices []*DiscoveredDevice
	ErrorMessage      string
}

// DeviceProbeRequest 单设备探测请求
type DeviceProbeRequest struct {
	IPAddress    string `json:"ipAddress" binding:"required,ip"`
	CredentialID string `json:"credentialId" binding:"required,uuid"`
	SNMPPort     int    `json:"snmpPort,omitempty"`
	// Communities 用户输入的多个 SNMP community 进行尝试
	// 如果为空，则使用凭证中的 SNMPCommunity
	Communities []string `json:"communities,omitempty"`
}

// DeviceProbeResult 单设备探测结果
type DeviceProbeResult struct {
	Success    bool                `json:"success"`
	Message    string              `json:"message"`
	IPAddress  string              `json:"ipAddress"`
	DeviceName string              `json:"deviceName,omitempty"`
	DeviceType models.DeviceType   `json:"deviceType,omitempty"`
	Vendor     models.DeviceVendor `json:"vendor,omitempty"`
	Model      string              `json:"model,omitempty"`
	SysDescr   string              `json:"sysDescr,omitempty"`
	SysName    string              `json:"sysName,omitempty"`
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
		TaskName:      req.TaskName,
		DiscoveryType: req.DiscoveryType,
		IPRanges:      models.IPRangeList(req.IPRanges),
		SNMPCommunity: req.SNMPCommunity,
		SNMPPort:      req.SNMPPort,
		Status:        models.DiscoveryStatusPending,
		TotalIPs:      totalIPs,
		AutoImport:    req.AutoImport,
		GroupID:       req.GroupID,
		CreatedBy:     req.CreatedBy,
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
		DiscoveryID:       discoveryID,
		Status:            task.Status,
		TotalIPs:          task.TotalIPs,
		DiscoveredCount:   len(discoveredDevices),
		DiscoveredDevices: discoveredDevices,
	}, nil
}

// discoverBySNMP 使用SNMP发现设备
func (s *DeviceDiscoveryService) discoverBySNMP(_ context.Context, task *models.DeviceDiscovery) ([]*DiscoveredDevice, error) {
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
// 完全按照测试脚本的逻辑：一个连接上执行多次 Get
func (s *DeviceDiscoveryService) snmpProbe(ip, community string, port int) *DiscoveredDevice {
	// 隐藏 SNMP 团体名敏感信息，只显示长度
	maskedCommunity := maskSensitiveStringDiscovery(community)
	logger.Debugf("[snmpProbe] 开始: ip=%s, community=%s, port=%d", ip, maskedCommunity, port)

	// 创建SNMP客户端（配置与测试脚本完全一致）
	snmpClient := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      uint16(port),
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(5) * time.Second,
		MaxOids:   1,
	}

	logger.Debugf("[snmpProbe] SNMP配置: Version=%d, Timeout=%v, MaxOids=%d",
		snmpClient.Version, snmpClient.Timeout, snmpClient.MaxOids)

	// 连接
	logger.Debug("[snmpProbe] 尝试连接 SNMP 服务器...")
	if err := snmpClient.Connect(); err != nil {
		logger.Debugf("[snmpProbe] SNMP 连接失败: %v", err)
		return &DiscoveredDevice{
			IPAddress: ip,
			IsAlive:   false,
		}
	}
	// 使用 defer 确保连接在函数结束时关闭（与测试脚本一致）
	defer snmpClient.Conn.Close()
	logger.Debug("[snmpProbe] SNMP 连接成功")

	// 在同一个连接上获取 sysName（与测试脚本一致）
	logger.Debug("[snmpProbe] 获取 sysName (OID: .1.3.6.1.2.1.1.5.0)...")
	startTime := time.Now()
	sysName, err := snmpClient.Get([]string{".1.3.6.1.2.1.1.5.0"})
	elapsed := time.Since(startTime)

	if err != nil {
		logger.Debugf("[snmpProbe] 获取 sysName 失败: %v (耗时: %v)", err, elapsed)
		return &DiscoveredDevice{
			IPAddress: ip,
			IsAlive:   false,
		}
	}
	logger.Debugf("[snmpProbe] sysName 获取成功, 耗时: %v, 变量数量: %d", elapsed, len(sysName.Variables))

	// 在同一个连接上获取 sysDescr（与测试脚本一致）
	logger.Debug("[snmpProbe] 获取 sysDescr (OID: .1.3.6.1.2.1.1.1.0)...")
	startTime = time.Now()
	sysDescr, err := snmpClient.Get([]string{".1.3.6.1.2.1.1.1.0"})
	elapsed = time.Since(startTime)

	if err != nil {
		logger.Debugf("[snmpProbe] sysDescr 获取失败: %v (耗时: %v)", err, elapsed)
		// sysDescr 失败不算致命错误，继续处理 sysName
	} else {
		logger.Debugf("[snmpProbe] sysDescr 获取成功, 耗时: %v, 变量数量: %d", elapsed, len(sysDescr.Variables))
	}

	// 解析结果
	device := &DiscoveredDevice{
		IPAddress: ip,
		IsAlive:   true,
	}

	if len(sysName.Variables) > 0 {
		variable := sysName.Variables[0]
		switch v := variable.Value.(type) {
		case string:
			device.SysName = v
		case []byte:
			device.SysName = string(v)
		}
		logger.Debugf("[snmpProbe] sysName=%s", device.SysName)
	}

	if len(sysDescr.Variables) > 0 {
		variable := sysDescr.Variables[0]
		switch v := variable.Value.(type) {
		case string:
			device.SysDescr = v
		case []byte:
			device.SysDescr = string(v)
		}
	}

	// 识别厂商和类型
	device.Vendor = string(devicepkg.IdentifyVendor(device.SysDescr))
	device.DeviceType = string(devicepkg.IdentifyDeviceType(device.SysDescr))

	logger.Debugf("[snmpProbe] 完成: IsAlive=%v, Vendor=%s, DeviceType=%s", device.IsAlive, device.Vendor, device.DeviceType)
	return device
}

// discoverByScan 使用Ping扫描发现设备
func (s *DeviceDiscoveryService) discoverByScan(_ context.Context, task *models.DeviceDiscovery) ([]*DiscoveredDevice, error) {
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
func (s *DeviceDiscoveryService) GetDiscoveryList(ctx context.Context, current, pageSize int, orderByColumn string, isAsc *bool) ([]models.DeviceDiscovery, int64, error) {
	var tasks []models.DeviceDiscovery
	var total int64

	query := s.db.WithContext(ctx).Model(&models.DeviceDiscovery{})

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询发现任务总数失败: %w", err)
	}

	// 分页查询 - 用户排序(白名单)优先,无 OrderByColumn 时保留 created_at DESC 默认
	offset := (current - 1) * pageSize
	sortReq := base.BaseListRequest{
		Current:       current,
		PageSize:      pageSize,
		OrderByColumn: orderByColumn,
		IsAsc:         isAsc,
	}
	query = base.ApplySort(query, sortReq, deviceDiscoveryAllowedSortFields)
	if orderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("查询发现任务失败: %w", err)
	}

	return tasks, total, nil
}

// deviceDiscoveryAllowedSortFields 设备发现任务可排序字段白名单。
var deviceDiscoveryAllowedSortFields = map[string]string{
	"taskName":  "task_name",
	"status":    "status",
	"ipRange":   "ip_range",
	"createdAt": "created_at",
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
	var communities []string

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
		// 隐藏 SNMP communities 敏感信息，只显示数量和长度信息
		maskedCommunities := make([]string, len(communities))
		for i, c := range communities {
			maskedCommunities[i] = maskSensitiveStringDiscovery(c)
		}
		logger.Debugf("[ProbeSingleDevice] 从凭证 %s 获取到 %d 个 SNMP communities: %v", credential.CredentialName, len(communities), maskedCommunities)
	}

	// 2. 只使用第一个 community（避免触发设备的防暴力破解机制）
	// 注意：测试表明华为设备会检测短时间内多个不同community的请求，并屏蔽源IP
	maskedFirstCommunity := maskSensitiveStringDiscovery(communities[0])
	logger.Debugf("[ProbeSingleDevice] 使用第 1 个 community 进行探测: %s (共%d个)", maskedFirstCommunity, len(communities))
	if len(communities) > 1 {
		logger.Debug("[ProbeSingleDevice] 注意: 凭证中配置了多个community，但只会尝试第一个。如需尝试其他，请手动调整顺序。")
	}

	community := communities[0]

	snmpClient := &gosnmp.GoSNMP{
		Target:    req.IPAddress,
		Port:      uint16(snmpPort),
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(5) * time.Second,
		MaxOids:   1,
	}

	err := snmpClient.Connect()
	if err != nil {
		logger.Debugf("[ProbeSingleDevice] SNMP 连接失败: %v", err)
		return &DeviceProbeResult{
			Success:   false,
			Message:   fmt.Sprintf("SNMP连接失败: %v", err),
			IPAddress: req.IPAddress,
		}, nil
	}
	// 使用defer确保连接总是被关闭，避免资源泄漏
	defer snmpClient.Conn.Close()

	sysName, err := snmpClient.Get([]string{".1.3.6.1.2.1.1.5.0"})
	if err != nil {
		logger.Debugf("[ProbeSingleDevice] 获取 sysName 失败: %v", err)
		return &DeviceProbeResult{
			Success:   false,
			Message:   fmt.Sprintf("SNMP request failed: %v, current community: %s", err, community),
			IPAddress: req.IPAddress,
		}, nil
	}

	sysDescr, _ := snmpClient.Get([]string{".1.3.6.1.2.1.1.1.0"})

	discoveredDevice := &DiscoveredDevice{
		IPAddress: req.IPAddress,
		IsAlive:   true,
	}

	if len(sysName.Variables) > 0 {
		variable := sysName.Variables[0]
		switch v := variable.Value.(type) {
		case string:
			discoveredDevice.SysName = v
		case []byte:
			discoveredDevice.SysName = string(v)
		}
	}

	if len(sysDescr.Variables) > 0 {
		variable := sysDescr.Variables[0]
		switch v := variable.Value.(type) {
		case string:
			discoveredDevice.SysDescr = v
		case []byte:
			discoveredDevice.SysDescr = string(v)
		}
	}

	discoveredDevice.Vendor = string(devicepkg.IdentifyVendor(discoveredDevice.SysDescr))
	discoveredDevice.DeviceType = string(devicepkg.IdentifyDeviceType(discoveredDevice.SysDescr))

	logger.Debugf("[ProbeSingleDevice] 探测成功: SysName=%s, Vendor=%s", discoveredDevice.SysName, discoveredDevice.Vendor)

	// 3. 处理探测结果
	if discoveredDevice.IsAlive {
		// 探测成功，提取型号信息
		model := ""
		vendor := models.DeviceVendor(discoveredDevice.Vendor)

		// 使用新的型号提取器
		if discoveredDevice.SysDescr != "" {
			extractor := devicepkg.NewModelExtractor(discoveredDevice.SysDescr, vendor)
			model = extractor.Extract()
			if model == "" {
				// 如果新提取器失败，回退到旧方法
				model = devicepkg.ExtractModelFromSysDescr(discoveredDevice.SysDescr, vendor)
			}
		}

		// 构造返回结果
		probeResult := &DeviceProbeResult{
			Success:    true,
			Message:    "探测成功",
			IPAddress:  req.IPAddress,
			DeviceName: discoveredDevice.SysName,
			SysDescr:   discoveredDevice.SysDescr,
			SysName:    discoveredDevice.SysName,
			Vendor:     vendor,
			Model:      model,
		}

		// 转换设备类型
		switch discoveredDevice.DeviceType {
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

	// 不应该到达这里，因为前面成功时已经返回
	return &DeviceProbeResult{
		Success:   false,
		Message:   "设备不可达",
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
		address := net.JoinHostPort(ip, fmt.Sprintf("%d", port))
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

// maskSensitiveStringDiscovery 隐藏敏感信息，只显示长度
// 例如: "public" -> "pu**c (5 chars)"
func maskSensitiveStringDiscovery(s string) string {
	if s == "" {
		return "(empty)"
	}
	length := len(s)
	if length <= 4 {
		return "*** (" + fmt.Sprintf("%d", length) + " chars)"
	}
	// 显示前两个字符和后两个字符，中间用 * 隐藏
	return s[:2] + "***" + s[length-2:] + " (" + fmt.Sprintf("%d", length) + " chars)"
}
