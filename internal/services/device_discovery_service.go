package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gosnmp/gosnmp"
	devicepkg "github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
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

// DiscoveredDevice 发现的设备信息。
// JSON tag 与前端「发现结果」表格列对齐（camelCase）。
type DiscoveredDevice struct {
	IPAddress   string `json:"ipAddress"`
	DeviceType  string `json:"deviceType"`
	Vendor      string `json:"vendor"`
	Model       string `json:"model"`
	SysName     string `json:"deviceName"`
	SysDescr    string `json:"sysDescr"`
	IsAlive     bool   `json:"isAlive"`
	SysObjectID string `json:"sysObjectId,omitempty"` // B-3 / v1.56：SNMP sysObjectID
}

// snmpPDUString 把 gosnmp PDU 值（string / []byte / 其它）统一收敛为字符串，
// 并规范化 OID（去首尾空白与前导点）。B-3 / v1.56：避免各处硬类型断言。
func snmpPDUString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(x), "."))
	case []byte:
		return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(x)), "."))
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
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
	Model       string              `json:"model,omitempty"`
	SysDescr    string              `json:"sysDescr,omitempty"`
	SysName     string              `json:"sysName,omitempty"`
	SysObjectID string              `json:"sysObjectId,omitempty"` // B-3 / v1.56
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
	if err := s.db.WithContext(ctx).Save(&task).Error; err != nil {
		return nil, fmt.Errorf("保存发现任务执行中状态失败: %w", err)
	}

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
		if saveErr := s.db.WithContext(ctx).Save(&task).Error; saveErr != nil {
			return nil, fmt.Errorf("保存发现任务失败状态失败: %v（发现错误: %w）", saveErr, err)
		}
		return nil, err
	}

	task.Status = models.DiscoveryStatusSuccess
	if err := s.db.WithContext(ctx).Save(&task).Error; err != nil {
		return nil, fmt.Errorf("保存发现任务结果失败: %w", err)
	}

	// Phase 137 / FIX-07: 把发现明细持久化，供结果查询端点返回真实数据。
	// 先清除该任务的历史结果（重跑发现时以最后一次结果为准），再批量写入。
	if err := s.persistDiscoveryResults(ctx, discoveryID, discoveredDevices); err != nil {
		return nil, err
	}

	return &DiscoveryResult{
		DiscoveryID:       discoveryID,
		Status:            task.Status,
		TotalIPs:          task.TotalIPs,
		DiscoveredCount:   len(discoveredDevices),
		DiscoveredDevices: discoveredDevices,
	}, nil
}

// persistDiscoveryResults 持久化设备发现明细（Phase 137 / FIX-07）。
// 同一任务重跑时覆盖旧结果；无设备时也清空旧结果，保证查询结果与最后一次执行一致。
//
// v1.55 收档登记（v1.54 顺带隐患）已于 quick 261109-wdu 清偿：Delete 与 Create
// 原先非原子，中途失败会留下「旧结果已删、新结果未写」的空档。现包进单个事务，
// 事务内全部走 tx（sqlite 单连接死锁红线，沿用 Phase 142 ImportDiscoveredDevices）。
func (s *DeviceDiscoveryService) persistDiscoveryResults(ctx context.Context, discoveryID string, devices []*DiscoveredDevice) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("discovery_id = ?", discoveryID).Delete(&models.DeviceDiscoveryResult{}).Error; err != nil {
			return fmt.Errorf("清理历史发现结果失败: %w", err)
		}
		if len(devices) == 0 {
			return nil
		}

		rows := make([]models.DeviceDiscoveryResult, 0, len(devices))
		seq := 0
		for _, dev := range devices {
			if dev == nil {
				continue
			}
			seq++
			rows = append(rows, models.DeviceDiscoveryResult{
				ID:          uuid.NewString(),
				DiscoveryID: discoveryID,
				Seq:         seq,
				IPAddress:   dev.IPAddress,
				DeviceType:  dev.DeviceType,
				Vendor:      dev.Vendor,
				Model:       dev.Model,
				SysName:     dev.SysName,
				SysDescr:    dev.SysDescr,
				IsAlive:     dev.IsAlive,
				SysObjectID: dev.SysObjectID, // B-3 / v1.56 中转列
			})
		}
		if len(rows) == 0 {
			return nil
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("保存发现结果失败: %w", err)
		}
		return nil
	})
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

	// B-3 / v1.56：同连接再取 sysObjectID（1.3.6.1.2.1.1.2.0）。
	// 失败不致命（部分设备/代理不返回），留空即可，匹配时自动回退 Model。
	if sysObjectID, oidErr := snmpClient.Get([]string{".1.3.6.1.2.1.1.2.0"}); oidErr == nil && len(sysObjectID.Variables) > 0 {
		device.SysObjectID = snmpPDUString(sysObjectID.Variables[0].Value)
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
	// 同步清理发现结果明细，避免任务软删后留下孤儿行。
	if err := s.db.WithContext(ctx).
		Where("discovery_id = ?", discoveryID).
		Delete(&models.DeviceDiscoveryResult{}).Error; err != nil {
		return fmt.Errorf("删除发现结果失败: %w", err)
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

// errDiscoveryTaskNotFound 是事务内「任务不存在」的哨兵；外层据此还原为
// 既有错误文案「发现任务不存在: <err>」，其余错误统一包成「导入发现设备失败」。
var errDiscoveryTaskNotFound = errors.New("发现任务不存在")

// discoveryImportMaxBatch 批量写入的单批上限，避免 PostgreSQL 65535 参数上限。
const discoveryImportMaxBatch = 1000

// ImportDiscoveredDevices 把一次发现任务的全部结果明细真实导入 sys_network_device。
//
// Phase 142 / DISC-01：忽略 deviceIDs 入参、恒全量导入该任务全部结果行（保留旧签名兼容）。
// 去重按 ip_address trim + net.ParseIP 校验：批内保留最后一条；库内未软删命中跳过不覆盖；
// 软删命中恢复（清 deleted_at + 覆盖字段）。单 GORM 事务全原子，事务内只用 tx。
func (s *DeviceDiscoveryService) ImportDiscoveredDevices(ctx context.Context, discoveryID string, deviceIDs []string, createdBy string) (int, error) {
	_ = deviceIDs // 恒全量导入；保留参数仅为兼容既有调用点

	count := 0
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1) 事务内用 tx 重新读取任务（校验存在 + 取 GroupID / SNMPPort）。
		var task models.DeviceDiscovery
		if err := tx.Where("id = ?", discoveryID).First(&task).Error; err != nil {
			return fmt.Errorf("%w: %w", errDiscoveryTaskNotFound, err)
		}

		// 2) 读取结果明细（含 ID/Seq 的原始行）。
		rows, err := loadDiscoveryResultRows(ctx, tx, discoveryID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		// 3) 按 IP 过滤 + 批内去重（保留最后一条），记录首次出现顺序。
		// 直接缓存构建好的待写入对象，避免后续重复映射。
		byIP := make(map[string]models.NetworkDevice, len(rows))
		ips := make([]string, 0, len(rows))
		for _, row := range rows {
			dev, ok := buildNetworkDevice(row, &task, createdBy)
			if !ok {
				continue
			}
			ip := dev.IPAddress
			if _, seen := byIP[ip]; !seen {
				ips = append(ips, ip)
			}
			byIP[ip] = dev
		}
		if len(ips) == 0 {
			return nil
		}

		// 4) 一次 IN 查询预加载库内已存在（含软删）行，避免 N+1。
		var existing []models.NetworkDevice
		if err := tx.Unscoped().Where("ip_address IN ?", ips).Find(&existing).Error; err != nil {
			return fmt.Errorf("查询已存在网络设备失败: %w", err)
		}
		existingByIP := make(map[string]models.NetworkDevice, len(existing))
		for _, dev := range existing {
			existingByIP[dev.IPAddress] = dev
		}

		newRows := make([]models.NetworkDevice, 0, len(ips))
		restored := make([]models.NetworkDevice, 0, len(ips))
		for _, ip := range ips {
			fresh := byIP[ip]
			if cur, hit := existingByIP[ip]; hit {
				if !cur.DeletedAt.Valid {
					// 库内未软删命中 → 跳过、不覆盖（保护人工维护字段）。
					continue
				}
				// 软删命中 → 恢复：清 deleted_at + 覆盖发现字段 + tx.Save。
				cur.DeletedAt = gorm.DeletedAt{}
				cur.DeviceName = fresh.DeviceName
				cur.DeviceType = fresh.DeviceType
				cur.Vendor = fresh.Vendor
				cur.Model = fresh.Model
				cur.IPAddress = fresh.IPAddress
				cur.Port = fresh.Port
				cur.SNMPPort = fresh.SNMPPort
				cur.CredentialID = nil
				cur.DeptID = fresh.DeptID
				cur.Location = ""
				cur.Status = fresh.Status
				cur.Description = fresh.Description
				cur.SerialNumber = ""
				cur.SoftwareVersion = ""
				cur.Uptime = ""
			cur.SysObjectID = fresh.SysObjectID // B-3 / v1.56：软删恢复同步刷新 OID
				cur.CreatedBy = fresh.CreatedBy
				cur.UpdatedBy = fresh.UpdatedBy
				restored = append(restored, cur)
				continue
			}
			newRows = append(newRows, fresh)
		}

		// 5) 分批 tx.Create；恢复行逐条 tx.Unscoped().Save（GORM 软删 UPDATE 会
		//    附加 deleted_at IS NULL，不加 Unscoped 会静默更新 0 行）。
		for i := 0; i < len(newRows); i += discoveryImportMaxBatch {
			end := i + discoveryImportMaxBatch
			if end > len(newRows) {
				end = len(newRows)
			}
			batch := newRows[i:end]
			if err := tx.Create(&batch).Error; err != nil {
				return fmt.Errorf("批量写入网络设备失败: %w", err)
			}
		}
		for i := range restored {
			if err := tx.Unscoped().Save(&restored[i]).Error; err != nil {
				return fmt.Errorf("恢复网络设备失败: %w", err)
			}
		}

		// 6) 显式写回 status=0（在线）。
		//    sys_network_device.status 的列定义带 `default:2`（未知），GORM 对带
		//    DefaultValueInterface 的字段会把零值无条件替换成默认值：即使结构体里
		//    赋 DeviceStatusOnline(0)，tx.Create / tx.Save 落库仍是 2，且 Select("*")
		//    也不能绕过。用一次单列 Update 显式写 0 才是双方言都可靠的做法；
		//    writtenIPs 只含新建 + 恢复行，不会碰到「跳过不覆盖」的人工维护行。
		writtenIPs := make([]string, 0, len(newRows)+len(restored))
		for i := range newRows {
			writtenIPs = append(writtenIPs, newRows[i].IPAddress)
		}
		for i := range restored {
			writtenIPs = append(writtenIPs, restored[i].IPAddress)
		}
		if len(writtenIPs) > 0 {
			if err := tx.Model(&models.NetworkDevice{}).
				Where("ip_address IN ?", writtenIPs).
				Update("status", models.DeviceStatusOnline).Error; err != nil {
				return fmt.Errorf("回写设备状态失败: %w", err)
			}
		}

		count = len(newRows) + len(restored)
		return nil
	})

	if txErr != nil {
		if errors.Is(txErr, errDiscoveryTaskNotFound) {
			// 保持既有文案「发现任务不存在: <err>」。
			return 0, txErr
		}
		return 0, fmt.Errorf("导入发现设备失败: %w", txErr)
	}
	return count, nil
}

// loadDiscoveryResultRows 在事务内读取发现结果明细，保留原始 ID/Seq 顺序。
func loadDiscoveryResultRows(ctx context.Context, tx *gorm.DB, discoveryID string) ([]models.DeviceDiscoveryResult, error) {
	var rows []models.DeviceDiscoveryResult
	if err := tx.WithContext(ctx).
		Where("discovery_id = ?", discoveryID).
		Order("seq ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询发现结果失败: %w", err)
	}
	return rows, nil
}

// buildNetworkDevice 把一条发现结果映射为待写入的 NetworkDevice；ok=false 表示该行跳过。
// 字段兜底按 Phase 142 裁决表：SysName/DeviceType/Vendor/DeptID/SNMPPort 均有回落，
// Status 按 models.DeviceStatus（0 在线）赋值，不套用通用「0 正常/1 停用」。
func buildNetworkDevice(row models.DeviceDiscoveryResult, task *models.DeviceDiscovery, createdBy string) (models.NetworkDevice, bool) {
	ip := strings.TrimSpace(row.IPAddress)
	if !row.IsAlive || ip == "" || net.ParseIP(ip) == nil {
		return models.NetworkDevice{}, false
	}

	deviceName := row.SysName
	if deviceName == "" {
		// Device-<IP 末段>，与 network_device_service.go 的 getLastIPOctet 同义。
		last := ip
		if i := strings.LastIndex(ip, "."); i >= 0 {
			last = ip[i+1:]
		}
		deviceName = fmt.Sprintf("Device-%s", last)
	}

	deviceType := models.DeviceType(row.DeviceType)
	switch deviceType {
	case models.DeviceTypeRouter, models.DeviceTypeSwitch, models.DeviceTypeFirewall,
		models.DeviceTypeAP, models.DeviceTypeLoadBalancer:
		// 合法枚举，保留
	default:
		deviceType = models.DeviceTypeSwitch
	}

	var deptID *string
	if task != nil && task.GroupID != nil {
		group := strings.TrimSpace(*task.GroupID)
		// org_id 必须是合法 UUID（CLAUDE.md 硬约定）：非法值不写脏数据，留空即可。
		if constants.UUIDPattern.MatchString(group) {
			deptID = &group
		}
	}

	snmpPort := 161
	if task != nil && task.SNMPPort > 0 {
		snmpPort = task.SNMPPort
	}

	return models.NetworkDevice{
		BaseModel: models.BaseModel{
			CreatedBy: createdBy,
			UpdatedBy: createdBy,
		},
		DeviceName:  deviceName,
		DeviceType:  deviceType,
		Vendor:      models.DeviceVendor(row.Vendor), // 空 → ""（列允许空串）
		Model:       row.Model,
		IPAddress:   ip,
		Port:        22,
		SNMPPort:    snmpPort,
		DeptID:      deptID,
		Status:      models.DeviceStatusOnline, // 存活设备导入即在线
		Description: row.SysDescr,
		SysObjectID: row.SysObjectID, // B-3 / v1.56：OID 从发现结果带入网络设备
	}, true
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
	// B-3 / v1.56：同连接取 sysObjectID，失败不致命（留空回退 Model）。
	sysObjectID, _ := snmpClient.Get([]string{".1.3.6.1.2.1.1.2.0"})

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
	if sysObjectID != nil && len(sysObjectID.Variables) > 0 {
		discoveredDevice.SysObjectID = snmpPDUString(sysObjectID.Variables[0].Value)
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
			SysName:     discoveredDevice.SysName,
			Vendor:      vendor,
			Model:       model,
			SysObjectID: discoveredDevice.SysObjectID, // B-3 / v1.56
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

// GetDiscoveryResults 获取发现结果（Phase 137 / FIX-07）。
// 返回 ExecuteDiscovery 已持久化的设备明细；任务存在但未执行过时返回空列表。
func (s *DeviceDiscoveryService) GetDiscoveryResults(ctx context.Context, discoveryID string) ([]*DiscoveredDevice, error) {
	// 校验发现任务存在（不存在时返回错误，与 handler 的 1500 分支一致）
	var task models.DeviceDiscovery
	if err := s.db.WithContext(ctx).Where("id = ?", discoveryID).First(&task).Error; err != nil {
		return nil, fmt.Errorf("发现任务不存在: %w", err)
	}

	var rows []models.DeviceDiscoveryResult
	if err := s.db.WithContext(ctx).
		Where("discovery_id = ?", discoveryID).
		Order("seq ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询发现结果失败: %w", err)
	}

	devices := make([]*DiscoveredDevice, 0, len(rows))
	for _, row := range rows {
		devices = append(devices, &DiscoveredDevice{
			IPAddress:  row.IPAddress,
			DeviceType: row.DeviceType,
			Vendor:     row.Vendor,
			Model:      row.Model,
			SysName:     row.SysName,
			SysDescr:    row.SysDescr,
			IsAlive:     row.IsAlive,
			SysObjectID: row.SysObjectID, // B-3 / v1.56
		})
	}
	return devices, nil
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
