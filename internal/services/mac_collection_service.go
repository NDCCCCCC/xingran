package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MACCollectionService MAC地址采集服务
type MACCollectionService struct {
	db            *gorm.DB
	executor      *device.DeviceExecutor
	maxConcurrent int
}

// MAC采集并发数的默认配置键
const macConcurrentConfigKey = "network.mac.collection.concurrent"

// NewMACCollectionService 创建MAC采集服务
func NewMACCollectionService(db *gorm.DB, executor *device.DeviceExecutor) *MACCollectionService {
	svc := &MACCollectionService{
		db:            db,
		executor:      executor,
		maxConcurrent: 10, // 默认值
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
			deviceCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
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
			log.Printf("[MAC采集] 设备 %s 发生 panic: %v", device.DeviceName, r)
		}
	}()

	result := &MACCollectionResult{
		DeviceID:       device.ID,
		DeviceName:     device.DeviceName,
		CollectionTime: time.Now(),
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

	// 保存到数据库
	collectionTime := time.Now()
	successCount := 0
	for _, macAddr := range macAddresses {
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

		// 使用 OnConflict 处理重复键：如果存在则更新，否则插入
		if err := s.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "device_id"}, {Name: "mac_address"}, {Name: "interface_name"}},
				DoUpdates: clause.AssignmentColumns([]string{"vlan_id", "mac_type", "collected_at"}),
			}).
			Create(macRecord).Error; err == nil {
			successCount++
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
		models.VendorH3C:     "display mac-address",
		models.VendorRuijie:  "show mac-address-table",
		models.VendorMaipu:   "show mac-address-table",
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
		if line == "" || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "MAC Address") {
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
		// Huawei/H3C 格式: MAC Address VLAN Learned-From Type
		if len(fields) >= 4 {
			entry.MACAddress = fields[0]
			entry.InterfaceName = fields[2]
			entry.MACType = fields[len(fields)-1]
		}

	case models.VendorRuijie, models.VendorMaipu:
		// Ruijie/Maipu 格式: VLAN MAC Address Type Port
		if len(fields) >= 4 {
			entry.MACAddress = fields[1]
			entry.InterfaceName = fields[3]
			entry.MACType = fields[2]
		}
	}

	if entry.MACAddress != "" {
		return entry, true
	}
	return MACAddressEntry{}, false
}

// getDeviceCredential 获取设备凭证
func (s *MACCollectionService) getDeviceCredential(ctx context.Context, device *models.NetworkDevice) (*models.AuthCredential, error) {
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

// GetMACAddressList 获取MAC地址列表
func (s *MACCollectionService) GetMACAddressList(ctx context.Context, current, pageSize int, deviceID, macAddress, interfaceName string) ([]models.DeviceMACAddress, int64, error) {
	var macAddrs []models.DeviceMACAddress
	var total int64

	query := s.db.WithContext(ctx).Model(&models.DeviceMACAddress{})

	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if macAddress != "" {
		query = query.Where("mac_address LIKE ?", "%"+macAddress+"%")
	}
	if interfaceName != "" {
		query = query.Where("interface_name LIKE ?", "%"+interfaceName+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询MAC地址总数失败: %w", err)
	}

	// 分页查询
	offset := (current - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("collected_at DESC").Find(&macAddrs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询MAC地址列表失败: %w", err)
	}

	return macAddrs, total, nil
}

// GetMACAddressStats 获取MAC地址统计信息
func (s *MACCollectionService) GetMACAddressStats(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalRecords    int64
		UniqueDevices   int64
		UniqueMACs      int64
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

// loadConfigFromDB 从数据库加载并发数配置
func (s *MACCollectionService) loadConfigFromDB() {
	var config models.Config
	err := s.db.Where("config_key = ?", macConcurrentConfigKey).First(&config).Error
	if err == nil && config.ConfigValue != "" {
		if concurrent, parseErr := strconv.Atoi(config.ConfigValue); parseErr == nil && concurrent > 0 {
			s.maxConcurrent = concurrent
			log.Printf("[MAC采集] 从数据库读取并发数配置: %d", s.maxConcurrent)
		}
	}
}

// ReloadConfig 重新加载配置（从数据库）
func (s *MACCollectionService) ReloadConfig() {
	s.loadConfigFromDB()
}
