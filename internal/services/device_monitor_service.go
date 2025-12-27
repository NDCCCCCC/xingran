package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// DeviceMonitorService 设备监控服务
// 用于定时检查设备状态、更新设备信息、采集端口状态
type DeviceMonitorService struct {
	db                   *gorm.DB
	deviceManager        *device.Manager
	portCollectionSvc    *PortCollectionService
	macCollectionSvc     *MACCollectionService
	configBackupSvc      *ConfigBackupService
	maxConcurrent        int
	timeout              time.Duration
}

// DeviceMonitorConfig 监控服务配置
type DeviceMonitorConfig struct {
	MaxConcurrent int           // 最大并发数
	Timeout       time.Duration // 单个设备超时时间
}

// DefaultDeviceMonitorConfig 默认配置
func DefaultDeviceMonitorConfig() *DeviceMonitorConfig {
	return &DeviceMonitorConfig{
		MaxConcurrent: 10,
		Timeout:       30 * time.Second,
	}
}

// NewDeviceMonitorService 创建设备监控服务
func NewDeviceMonitorService(db *gorm.DB, sm4Cipher *security.SM4Cipher, config *DeviceMonitorConfig) *DeviceMonitorService {
	if config == nil {
		config = DefaultDeviceMonitorConfig()
	}

	deviceManager := device.NewManager(db, sm4Cipher, device.DefaultManagerConfig())

	return &DeviceMonitorService{
		db:            db,
		deviceManager: deviceManager,
		// 初始化时同时创建子服务
		portCollectionSvc: NewPortCollectionService(db, deviceManager),
		macCollectionSvc:  NewMACCollectionService(db, deviceManager),
		configBackupSvc:   NewConfigBackupService(db, deviceManager),
		maxConcurrent:     config.MaxConcurrent,
		timeout:           config.Timeout,
	}
}

// SetPortCollectionService 设置端口采集服务
func (s *DeviceMonitorService) SetPortCollectionService(svc *PortCollectionService) {
	s.portCollectionSvc = svc
}

// SetMACCollectionService 设置MAC采集服务
func (s *DeviceMonitorService) SetMACCollectionService(svc *MACCollectionService) {
	s.macCollectionSvc = svc
}

// SetConfigBackupService 设置配置备份服务
func (s *DeviceMonitorService) SetConfigBackupService(svc *ConfigBackupService) {
	s.configBackupSvc = svc
}

// CheckAllDevicesStatus 检查所有设备状态（通过SNMP）
// 返回: 在线数量, 离线数量, 错误
func (s *DeviceMonitorService) CheckAllDevicesStatus(ctx context.Context) (int, int, error) {
	// 获取所有需要检查的设备
	var devices []models.NetworkDevice
	if err := s.db.WithContext(ctx).Find(&devices).Error; err != nil {
		return 0, 0, fmt.Errorf("查询设备列表失败: %w", err)
	}

	if len(devices) == 0 {
		return 0, 0, nil
	}

	// 并发检查设备状态
	sem := semaphore.NewWeighted(int64(s.maxConcurrent))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var onlineCount, offlineCount int

	for _, dev := range devices {
		wg.Add(1)
		go func(device models.NetworkDevice) {
			defer wg.Done()

			// 获取信号量
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Printf("获取信号量失败: %v", err)
				return
			}
			defer sem.Release(1)

			// 检查设备状态
			isOnline, err := s.CheckDeviceStatus(ctx, device.ID)
			if err != nil {
				log.Printf("检查设备状态失败 [%s]: %v", device.IPAddress, err)
				return
			}

			// 更新统计
			mu.Lock()
			if isOnline {
				onlineCount++
			} else {
				offlineCount++
			}
			mu.Unlock()
		}(dev)
	}

	wg.Wait()

	log.Printf("设备状态检查完成: 在线 %d, 离线 %d", onlineCount, offlineCount)
	return onlineCount, offlineCount, nil
}

// CheckDeviceStatus 检查单个设备状态
// 通过SNMP Ping检测设备是否在线
func (s *DeviceMonitorService) CheckDeviceStatus(ctx context.Context, deviceID string) (bool, error) {
	// 获取设备信息
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", deviceID).First(&device).Error; err != nil {
		return false, fmt.Errorf("查询设备失败: %w", err)
	}

	log.Printf("[CheckDeviceStatus] 设备: %s (%s), CredentialID: %v", device.IPAddress, device.DeviceName, device.CredentialID)

	// 获取凭证信息
	credential, err := s.getCredentialForDevice(ctx, &device)
	if err != nil {
		log.Printf("[CheckDeviceStatus] 获取凭证失败，使用默认配置: %v", err)
		// 如果没有凭证，使用默认community尝试
		defaultCommunity := "public"
		defaultVersion := models.SNMPVersionV2c
		credential = &models.AuthCredential{
			SNMPCommunities: []string{defaultCommunity},
			SNMPVersion:     defaultVersion,
		}
	} else {
		if len(credential.SNMPCommunities) > 0 {
			log.Printf("[CheckDeviceStatus] 使用凭证: Community=%s, Version=%s", credential.SNMPCommunities[0], credential.SNMPVersion)
		} else {
			log.Printf("[CheckDeviceStatus] 凭证没有配置 SNMP Community，跳过检查")
			return false, nil
		}
	}

	// 尝试SNMP连接
	// 转换 SNMP 版本
	snmpVersion := s.convertSNMPVersion(credential.SNMPVersion)

	// 通过 SNMP ping 设备 - 使用第一个 SNMP community
	community := ""
	if len(credential.SNMPCommunities) > 0 {
		community = credential.SNMPCommunities[0]
	}
	isOnline := s.pingDeviceViaSNMP(device.IPAddress, uint16(device.SNMPPort), community, snmpVersion)

	// 更新设备状态
	now := time.Now()
	status := models.DeviceStatusOffline
	if isOnline {
		status = models.DeviceStatusOnline
	}

	updates := map[string]interface{}{
		"status":      status,
		"last_seen_at": &now,
	}

	// 如果设备从离线变为在线，或状态改变，更新数据库
	if device.Status != status {
		if err := s.db.WithContext(ctx).Model(&device).Updates(updates).Error; err != nil {
			return isOnline, fmt.Errorf("更新设备状态失败: %w", err)
		}
		log.Printf("设备 [%s] 状态更新: %d -> %d", device.IPAddress, device.Status, status)
	}

	return isOnline, nil
}

// pingDeviceViaSNMP 通过SNMP ping设备
func (s *DeviceMonitorService) pingDeviceViaSNMP(target string, port uint16, community string, version device.SNMPVersion) bool {
	log.Printf("[SNMP Ping] 开始检查设备: %s:%d, community=%s, version=%v", target, port, community, version)

	// 创建完整的 SNMP 客户端配置
	config := &device.SNMPClientConfig{
		Target:    target,
		Port:      port,
		Community: community,
		Version:   version,
		Timeout:   5 * time.Second,
		Retries:   2,
	}
	snmpClient := device.NewSNMPClient(config)
	defer snmpClient.Close()

	// 尝试获取系统描述
	log.Printf("[SNMP Ping] 尝试获取系统描述 OID: 1.3.6.1.2.1.1.1.0")
	result, err := snmpClient.Get("1.3.6.1.2.1.1.1.0")
	if err == nil {
		log.Printf("[SNMP Ping] 获取系统描述成功: %v", result)
		return true
	}
	log.Printf("[SNMP Ping] 获取系统描述失败: %v", err)

	// 如果失败，尝试获取系统名称
	log.Printf("[SNMP Ping] 尝试获取系统名称 OID: 1.3.6.1.2.1.1.5.0")
	result, err = snmpClient.Get("1.3.6.1.2.1.1.5.0")
	if err == nil {
		log.Printf("[SNMP Ping] 获取系统名称成功: %v", result)
		return true
	}
	log.Printf("[SNMP Ping] 获取系统名称失败: %v", err)

	log.Printf("[SNMP Ping] 设备 %s SNMP检查失败", target)
	return false
}

// UpdateAllDeviceInfo 更新所有设备的详细信息（通过SSH）
// 采集设备型号、版本、序列号等详细信息
func (s *DeviceMonitorService) UpdateAllDeviceInfo(ctx context.Context) error {
	// 只更新在线的设备
	var devices []models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("status = ?", models.DeviceStatusOnline).Find(&devices).Error; err != nil {
		return fmt.Errorf("查询在线设备失败: %w", err)
	}

	if len(devices) == 0 {
		log.Println("没有在线设备需要更新")
		return nil
	}

	log.Printf("开始更新 %d 个在线设备的详细信息", len(devices))

	// 并发更新设备信息
	sem := semaphore.NewWeighted(int64(s.maxConcurrent))
	var wg sync.WaitGroup

	for _, dev := range devices {
		// 跳过没有凭证的设备
		if dev.CredentialID == nil || *dev.CredentialID == "" {
			continue
		}

		wg.Add(1)
		go func(deviceID string) {
			defer wg.Done()

			// 获取信号量
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Printf("获取信号量失败: %v", err)
				return
			}
			defer sem.Release(1)

			// 更新设备信息
			if err := s.UpdateDeviceInfo(ctx, deviceID); err != nil {
				log.Printf("更新设备信息失败 [%s]: %v", deviceID, err)
			}
		}(dev.ID)
	}

	wg.Wait()
	log.Println("设备详细信息更新完成")
	return nil
}

// UpdateDeviceInfo 更新单个设备的详细信息
func (s *DeviceMonitorService) UpdateDeviceInfo(ctx context.Context, deviceID string) error {
	// 获取设备信息
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", deviceID).First(&device).Error; err != nil {
		return fmt.Errorf("查询设备失败: %w", err)
	}

	// 检查是否有凭证
	if device.CredentialID == nil || *device.CredentialID == "" {
		return fmt.Errorf("设备没有关联凭证")
	}

	// 获取凭证信息
	var credential models.AuthCredential
	if err := s.db.WithContext(ctx).Where("id = ?", *device.CredentialID).First(&credential).Error; err != nil {
		return fmt.Errorf("查询凭证失败: %w", err)
	}

	// 连接设备
	wrapper, err := s.deviceManager.ConnectToDevice(ctx, &device, &credential)
	if err != nil {
		return fmt.Errorf("连接设备失败: %w", err)
	}
	defer s.deviceManager.DisconnectFromDevice(device.ID)

	// 根据厂商获取设备信息
	info, err := s.collectDeviceInfo(device.Vendor, wrapper)
	if err != nil {
		return fmt.Errorf("采集设备信息失败: %w", err)
	}

	// 更新数据库
	updates := map[string]interface{}{}
	if info.Model != "" {
		updates["model"] = info.Model
	}
	if info.SerialNumber != "" {
		updates["serial_number"] = info.SerialNumber
	}
	if info.SoftwareVersion != "" {
		updates["software_version"] = info.SoftwareVersion
	}
	if info.Uptime != "" {
		updates["uptime"] = info.Uptime
	}

	if len(updates) > 0 {
		now := time.Now()
		updates["updated_at"] = now
		if err := s.db.WithContext(ctx).Model(&device).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新设备信息失败: %w", err)
		}
		log.Printf("设备 [%s] 信息已更新", device.IPAddress)
	}

	return nil
}

// DeviceInfo 设备详细信息
type DeviceInfo struct {
	Model           string
	SerialNumber    string
	SoftwareVersion string
	Uptime          string
}

// collectDeviceInfo 采集设备详细信息
func (s *DeviceMonitorService) collectDeviceInfo(vendor models.DeviceVendor, wrapper *device.ScrapliWrapper) (*DeviceInfo, error) {
	info := &DeviceInfo{}

	var commands []string
	switch vendor {
	case models.VendorHuawei, models.VendorH3C:
		commands = []string{
			"display version",
			"display device",
		}
	case models.VendorRuijie, models.VendorMaipu:
		commands = []string{
			"show version",
		}
	default:
		commands = []string{
			"show version",
		}
	}

	for _, cmd := range commands {
		response, err := wrapper.SendCommand(cmd, true)
		if err != nil {
			continue
		}
		s.parseDeviceInfo(response.Result, vendor, info)
	}

	return info, nil
}

// parseDeviceInfo 解析设备信息
func (s *DeviceMonitorService) parseDeviceInfo(output string, vendor models.DeviceVendor, info *DeviceInfo) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		switch vendor {
		case models.VendorHuawei, models.VendorH3C:
			s.parseHuaweiDeviceInfo(line, info)
		case models.VendorRuijie, models.VendorMaipu:
			s.parseRuijieDeviceInfo(line, info)
		default:
			s.parseGenericDeviceInfo(line, info)
		}
	}
}

// parseHuaweiDeviceInfo 解析华为/H3C设备信息
func (s *DeviceMonitorService) parseHuaweiDeviceInfo(line string, info *DeviceInfo) {
	// 解析型号
	// 示例: H3C S5120-SI52P-EI
	//       Huawei Versatile Routing Platform Software
	if strings.Contains(line, "H3C") || strings.Contains(line, "Huawei") {
		if strings.Contains(line, "S5") || strings.Contains(line, "S6") || strings.Contains(line, "S31") || strings.Contains(line, "S51") || strings.Contains(line, "S65") {
			fields := strings.Fields(line)
			for i, f := range fields {
				if strings.HasPrefix(f, "S") && len(f) > 3 {
					info.Model = strings.Join(fields[i:], " ")
					break
				}
			}
		}
	}

	// 解析版本号
	// 示例: Version 5.20, Release 2432P02
	if strings.HasPrefix(line, "Version") || strings.Contains(line, "Software Version") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "Version" && i+1 < len(fields) {
				info.SoftwareVersion = fields[i+1]
				break
			}
		}
	}

	// 解析序列号
	// 示例: DEVICE_SERIAL_NUMBER : xxxxx
	if strings.Contains(line, "Serial") || strings.Contains(line, "DEVICE_SERIAL_NUMBER") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if (f == "NUMBER" || f == "Serial") && i+1 < len(fields) {
				if fields[i+1] != ":" {
					info.SerialNumber = fields[i+1]
				} else if i+2 < len(fields) {
					info.SerialNumber = fields[i+2]
				}
				break
			}
		}
	}

	// 解析运行时间
	// 示例: Uptime is 10 days, 2 hours, 30 minutes
	if strings.Contains(line, "Uptime") || strings.Contains(line, "uptime") {
		if idx := strings.Index(line, "is"); idx > 0 {
			info.Uptime = strings.TrimSpace(line[idx+2:])
		}
	}
}

// parseRuijieDeviceInfo 解析锐捷/迈普设备信息
func (s *DeviceMonitorService) parseRuijieDeviceInfo(line string, info *DeviceInfo) {
	// 解析型号
	// 示例: Ruijie Networks Switch S5750
	if strings.Contains(line, "S5750") || strings.Contains(line, "S6220") || strings.Contains(line, "N180") {
		fields := strings.Fields(line)
		for _, f := range fields {
			if strings.HasPrefix(f, "S") || strings.HasPrefix(f, "N") {
				info.Model = f
				break
			}
		}
	}

	// 解析版本号
	// 示例: Software Version 10.4(3b5)
	if strings.Contains(line, "Software Version") || strings.Contains(line, "SW Version") {
		if idx := strings.Index(line, "Version"); idx > 0 {
			info.SoftwareVersion = strings.TrimSpace(line[idx+7:])
		}
	}

	// 解析序列号
	if strings.Contains(line, "System Serial") || strings.Contains(line, "Serial Number") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if (f == "Number" || f == "Serial") && i+1 < len(fields) {
				if fields[i+1] != ":" {
					info.SerialNumber = fields[i+1]
				} else if i+2 < len(fields) {
					info.SerialNumber = fields[i+2]
				}
				break
			}
		}
	}

	// 解析运行时间
	if strings.Contains(line, "uptime") || strings.Contains(line, "Uptime") {
		if idx := strings.Index(line, "is"); idx > 0 {
			info.Uptime = strings.TrimSpace(line[idx+2:])
		}
	}
}

// parseGenericDeviceInfo 解析通用设备信息
func (s *DeviceMonitorService) parseGenericDeviceInfo(line string, info *DeviceInfo) {
	// 通用解析逻辑
	if strings.Contains(line, "Version") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "Version" && i+1 < len(fields) {
				info.SoftwareVersion = fields[i+1]
				break
			}
		}
	}

	if strings.Contains(line, "Serial") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if (f == "Serial" || f == "Number") && i+1 < len(fields) {
				if fields[i+1] != ":" {
					info.SerialNumber = fields[i+1]
				} else if i+2 < len(fields) {
					info.SerialNumber = strings.Join(fields[i+2:], " ")
				}
				break
			}
		}
	}

	if strings.Contains(line, "Uptime") || strings.Contains(line, "uptime") {
		if idx := strings.Index(line, "is"); idx > 0 {
			info.Uptime = strings.TrimSpace(line[idx+2:])
		}
	}
}

// CollectAllPortStatus 采集所有设备的端口状态
func (s *DeviceMonitorService) CollectAllPortStatus(ctx context.Context) error {
	if s.portCollectionSvc == nil {
		return fmt.Errorf("端口采集服务未初始化")
	}

	results, err := s.portCollectionSvc.CollectAllDevices(ctx)
	if err != nil {
		return fmt.Errorf("端口状态采集失败: %w", err)
	}

	// 统计结果
	successCount := 0
	totalPorts := 0
	for _, result := range results {
		if result.ErrorMessage == "" {
			successCount++
			totalPorts += result.SuccessCount
		}
	}

	log.Printf("端口状态采集完成: 成功设备=%d/%d, 总端口数=%d", successCount, len(results), totalPorts)
	return nil
}

// CollectPortStatus 采集单个设备的端口状态
func (s *DeviceMonitorService) CollectPortStatus(ctx context.Context, deviceID string) error {
	if s.portCollectionSvc == nil {
		return fmt.Errorf("端口采集服务未初始化")
	}

	_, err := s.portCollectionSvc.CollectDevice(ctx, deviceID)
	return err
}

// CollectAllMACAddresses 采集所有设备的MAC地址表
func (s *DeviceMonitorService) CollectAllMACAddresses(ctx context.Context) error {
	if s.macCollectionSvc == nil {
		return fmt.Errorf("MAC采集服务未初始化")
	}

	results, err := s.macCollectionSvc.CollectAllDevices(ctx)
	if err != nil {
		return fmt.Errorf("MAC地址采集失败: %w", err)
	}

	// 统计结果
	successCount := 0
	totalMACs := 0
	for _, result := range results {
		if result.ErrorMessage == "" {
			successCount++
			totalMACs += result.SuccessCount
		}
	}

	log.Printf("MAC地址采集完成: 成功设备=%d/%d, 总MAC数=%d", successCount, len(results), totalMACs)
	return nil
}

// CollectMACAddresses 采集单个设备的MAC地址表
func (s *DeviceMonitorService) CollectMACAddresses(ctx context.Context, deviceID string) error {
	if s.macCollectionSvc == nil {
		return fmt.Errorf("MAC采集服务未初始化")
	}

	_, err := s.macCollectionSvc.CollectDevice(ctx, deviceID)
	return err
}

// BackupAllConfigurations 备份所有设备配置
func (s *DeviceMonitorService) BackupAllConfigurations(ctx context.Context) error {
	if s.configBackupSvc == nil {
		return fmt.Errorf("配置备份服务未初始化")
	}

	if err := s.configBackupSvc.AutoBackupAllDevices(ctx); err != nil {
		return fmt.Errorf("配置备份失败: %w", err)
	}

	log.Printf("配置备份任务完成")
	return nil
}

// BackupConfiguration 备份单个设备配置
func (s *DeviceMonitorService) BackupConfiguration(ctx context.Context, deviceID string, createdBy string) error {
	if s.configBackupSvc == nil {
		return fmt.Errorf("配置备份服务未初始化")
	}

	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", deviceID).First(&device).Error; err != nil {
		return fmt.Errorf("设备不存在: %w", err)
	}

	req := &BackupRequest{
		DeviceID:      deviceID,
		DeviceName:    device.DeviceName,
		BackupType:    models.BackupTypeManual,
		ChangeReason:  "手动备份",
		CreatedBy:     createdBy,
		CompressLarge: true,
	}

	_, err := s.configBackupSvc.CreateBackup(ctx, req)
	return err
}

// getCredentialForDevice 获取设备的凭证
func (s *DeviceMonitorService) getCredentialForDevice(ctx context.Context, device *models.NetworkDevice) (*models.AuthCredential, error) {
	// 如果设备关联了凭证，使用关联凭证
	if device.CredentialID != nil && *device.CredentialID != "" {
		var credential models.AuthCredential
		if err := s.db.WithContext(ctx).Where("id = ?", *device.CredentialID).First(&credential).Error; err != nil {
			return nil, fmt.Errorf("查询凭证失败: %w", err)
		}
		return &credential, nil
	}

	// 否则查找默认凭证
	var credential models.AuthCredential
	if err := s.db.WithContext(ctx).Where("is_default = ?", true).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("未找到默认凭证: %w", err)
	}

	return &credential, nil
}

// Close 关闭服务
func (s *DeviceMonitorService) Close() error {
	if s.deviceManager != nil {
		return s.deviceManager.Close()
	}
	return nil
}

// convertSNMPVersion 转换 SNMP 版本字符串到 device.SNMPVersion
func (s *DeviceMonitorService) convertSNMPVersion(version models.SNMPVersion) device.SNMPVersion {
	switch version {
	case models.SNMPVersionV1:
		return device.SNMPVersion1
	case models.SNMPVersionV2c:
		return device.SNMPVersion2c
	case models.SNMPVersionV3:
		return device.SNMPVersion3
	default:
		return device.SNMPVersion2c // 默认使用 v2c
	}
}
