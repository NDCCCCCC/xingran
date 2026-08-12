package services

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/internal/services/lldp"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/internal/services/topology"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
)

// DeviceMonitorService 设备监控服务
// 用于定时检查设备状态、更新设备信息、采集端口状态
type DeviceMonitorService struct {
	db                *gorm.DB
	deviceManager     *device.Manager
	deviceExecutor    *device.DeviceExecutor
	portCollectionSvc *portcollection.PortCollectionService
	macCollectionSvc  *MACCollectionService
	configBackupSvc   *ConfigBackupService
	maxConcurrent     int
	timeout           time.Duration
	useDynamicConfig  bool // 是否使用动态配置（从数据库读取）
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
func NewDeviceMonitorService(db *gorm.DB, passwordCipher addomain.PasswordCipher, config *DeviceMonitorConfig) *DeviceMonitorService {
	if config == nil {
		config = DefaultDeviceMonitorConfig()
	}

	// 创建设备管理器（新架构：简化版，执行器由外部设置）
	deviceManager := device.NewManager(db)

	svc := &DeviceMonitorService{
		db:            db,
		deviceManager: deviceManager,
		// 子服务在 SetExecutor 之后创建
		maxConcurrent:    config.MaxConcurrent,
		timeout:          config.Timeout,
		useDynamicConfig: true, // 默认启用动态配置
	}

	// 尝试从数据库读取配置
	svc.loadConfigFromDB()

	return svc
}

// SetExecutor 设置执行器并初始化子服务
func (s *DeviceMonitorService) SetExecutor(executor *device.DeviceExecutor) {
	s.deviceExecutor = executor
	// 初始化子服务
	s.portCollectionSvc = portcollection.NewPortCollectionService(s.db, executor)
	lldpSvc := lldp.NewLLDPService(executor)
	filterRuleSvc := topology.NewFilterRuleService(s.db)
	s.macCollectionSvc = NewMACCollectionService(s.db, executor, lldpSvc, filterRuleSvc)
	s.configBackupSvc = NewConfigBackupService(s.db, executor)
}

// SetPortCollectionService 设置端口采集服务
func (s *DeviceMonitorService) SetPortCollectionService(svc *portcollection.PortCollectionService) {
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

// loadConfigFromDB 从数据库加载并发数和超时配置
func (s *DeviceMonitorService) loadConfigFromDB() {
	// 读取并发数配置
	var concurrentConfig models.Config
	err := s.db.Where("config_key = ?", "network.device.monitor.concurrent").First(&concurrentConfig).Error
	if err == nil && concurrentConfig.ConfigValue != "" {
		if concurrent, parseErr := strconv.Atoi(concurrentConfig.ConfigValue); parseErr == nil && concurrent > 0 {
			s.maxConcurrent = concurrent
			applogger.Infof("[设备监控] 从数据库读取并发数配置: %d", s.maxConcurrent)
		}
	}

	// 读取超时配置
	var timeoutConfig models.Config
	err = s.db.Where("config_key = ?", "network.device.timeout").First(&timeoutConfig).Error
	if err == nil && timeoutConfig.ConfigValue != "" {
		if seconds, parseErr := strconv.Atoi(timeoutConfig.ConfigValue); parseErr == nil && seconds > 0 {
			s.timeout = time.Duration(seconds) * time.Second
			applogger.Infof("[设备监控] 从数据库读取超时配置: %v", s.timeout)
		}
	}
}

// ReloadConfig 重新加载配置（从数据库）
func (s *DeviceMonitorService) ReloadConfig() {
	s.loadConfigFromDB()
	// 同时刷新子服务的配置
	if s.portCollectionSvc != nil && s.portCollectionSvc.Collection != nil {
		s.portCollectionSvc.Collection.ReloadConfig()
	}
	if s.macCollectionSvc != nil {
		s.macCollectionSvc.ReloadConfig()
	}
	if s.configBackupSvc != nil {
		s.configBackupSvc.ReloadConfig()
	}
	applogger.Infof("[设备监控] 配置已重新加载")
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
				applogger.Infof("获取信号量失败: %v", err)
				return
			}
			defer sem.Release(1)

			// 检查设备状态
			isOnline, err := s.CheckDeviceStatus(ctx, device.ID)
			if err != nil {
				applogger.Infof("检查设备状态失败 [%s]: %v", device.IPAddress, err)
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

	applogger.Infof("设备状态检查完成: 在线 %d, 离线 %d", onlineCount, offlineCount)
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

	applogger.Infof("[CheckDeviceStatus] 设备: %s (%s), CredentialID: %v", device.IPAddress, device.DeviceName, device.CredentialID)

	// 获取凭证信息
	credential, err := s.getCredentialForDevice(ctx, &device)
	if err != nil {
		applogger.Infof("[CheckDeviceStatus] 获取凭证失败，使用默认配置: %v", err)
		// 如果没有凭证，使用默认community尝试
		defaultCommunity := "public"
		defaultVersion := models.SNMPVersionV2c
		credential = &models.AuthCredential{
			SNMPCommunities: []string{defaultCommunity},
			SNMPVersion:     defaultVersion,
		}
	} else {
		if len(credential.SNMPCommunities) > 0 {
			// 隐藏 SNMP 团体名敏感信息，只显示长度
			maskedCommunity := maskSensitiveString(credential.SNMPCommunities[0])
			applogger.Infof("[CheckDeviceStatus] 使用凭证: Community=%s, Version=%s", maskedCommunity, credential.SNMPVersion)
		} else {
			applogger.Infof("[CheckDeviceStatus] 凭证没有配置 SNMP Community，跳过检查")
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

	// 更新 last_seen_at（无论状态是否改变，都要更新最后看到的时间）
	// 这样可以反映设备最后一次被检测到的时间
	updates := map[string]interface{}{
		"last_seen_at": &now,
	}

	// 如果状态改变了，也更新状态
	if device.Status != status {
		updates["status"] = status
		if err := s.db.WithContext(ctx).Model(&device).Updates(updates).Error; err != nil {
			return isOnline, fmt.Errorf("更新设备状态失败: %w", err)
		}
		applogger.Infof("设备 [%s] 状态更新: %d -> %d, 最后看到时间: %s", device.IPAddress, device.Status, status, now.Format("2006-01-02 15:04:05"))
	} else {
		// 状态没有改变，只更新最后看到时间
		if err := s.db.WithContext(ctx).Model(&device).Updates(updates).Error; err != nil {
			return isOnline, fmt.Errorf("更新设备最后看到时间失败: %w", err)
		}
		applogger.Infof("设备 [%s] 状态未变 (%d)，更新最后看到时间: %s", device.IPAddress, status, now.Format("2006-01-02 15:04:05"))
	}

	return isOnline, nil
}

// pingDeviceViaSNMP 通过SNMP ping设备
func (s *DeviceMonitorService) pingDeviceViaSNMP(target string, port uint16, community string, version device.SNMPVersion) bool {
	// 隐藏 SNMP 团体名敏感信息，只显示长度
	maskedCommunity := maskSensitiveString(community)
	applogger.Infof("[SNMP Ping] 开始检查设备: %s:%d, community=%s, version=%v", target, port, maskedCommunity, version)

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
	applogger.Infof("[SNMP Ping] 尝试获取系统描述 OID: 1.3.6.1.2.1.1.1.0")
	result, err := snmpClient.Get("1.3.6.1.2.1.1.1.0")
	if err == nil {
		applogger.Infof("[SNMP Ping] 获取系统描述成功: %v", result)
		return true
	}
	applogger.Infof("[SNMP Ping] 获取系统描述失败: %v", err)

	// 如果失败，尝试获取系统名称
	applogger.Infof("[SNMP Ping] 尝试获取系统名称 OID: 1.3.6.1.2.1.1.5.0")
	result, err = snmpClient.Get("1.3.6.1.2.1.1.5.0")
	if err == nil {
		applogger.Infof("[SNMP Ping] 获取系统名称成功: %v", result)
		return true
	}
	applogger.Infof("[SNMP Ping] 获取系统名称失败: %v", err)

	applogger.Infof("[SNMP Ping] 设备 %s SNMP检查失败", target)
	return false
}

// CollectAllPortStatus 采集所有设备的端口状态
func (s *DeviceMonitorService) CollectAllPortStatus(ctx context.Context) error {
	if s.portCollectionSvc == nil {
		return fmt.Errorf("端口采集服务未初始化")
	}

	results, err := s.portCollectionSvc.Collection.CollectAllDevices(ctx)
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

	applogger.Infof("端口状态采集完成: 成功设备=%d/%d, 总端口数=%d", successCount, len(results), totalPorts)
	return nil
}

// CollectPortStatus 采集单个设备的端口状态
func (s *DeviceMonitorService) CollectPortStatus(ctx context.Context, deviceID string) error {
	if s.portCollectionSvc == nil {
		return fmt.Errorf("端口采集服务未初始化")
	}

	_, err := s.portCollectionSvc.Collection.CollectDevice(ctx, deviceID)
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

	applogger.Infof("MAC地址采集完成: 成功设备=%d/%d, 总MAC数=%d", successCount, len(results), totalMACs)
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

	applogger.Infof("配置备份任务完成")
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

// maskSensitiveString 隐藏敏感信息，只显示长度
// 例如: "public" -> "*** (5 chars)"
func maskSensitiveString(s string) string {
	if s == "" {
		return "(empty)"
	}
	length := len(s)
	if length <= 4 {
		return "*** (" + strconv.Itoa(length) + " chars)"
	}
	// 显示前两个字符和后两个字符，中间用 * 隐藏
	return s[:2] + "***" + s[length-2:] + " (" + strconv.Itoa(length) + " chars)"
}
