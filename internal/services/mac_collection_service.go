package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// MACCollectionService MAC地址采集服务
type MACCollectionService struct {
	db      *gorm.DB
	manager *device.Manager
}

// NewMACCollectionService 创建MAC采集服务
func NewMACCollectionService(db *gorm.DB, manager *device.Manager) *MACCollectionService {
	return &MACCollectionService{
		db:      db,
		manager: manager,
	}
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
	semaphore := make(chan struct{}, 10) // 限制并发数为10

	for _, dev := range devices {
		wg.Add(1)
		go func(device models.NetworkDevice) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := s.collectDeviceMAC(ctx, &device)

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
	result := &MACCollectionResult{
		DeviceID:       device.ID,
		DeviceName:     device.DeviceName,
		CollectionTime: time.Now(),
	}

	// 获取凭证
	credential, err := s.getDeviceCredential(ctx, device)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}

	// 连接设备
	wrapper, err := s.manager.ConnectToDevice(ctx, device, credential)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}
	defer s.manager.DisconnectFromDevice(device.ID)

	// 获取MAC地址表命令
	command := s.getMACCommand(device.Vendor)

	// 发送命令
	response, err := wrapper.SendCommand(command, true)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}

	// 解析MAC地址表
	macAddresses, err := s.parseMACAddressTable(response.Result, device.Vendor)
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

		if err := s.db.WithContext(ctx).Create(macRecord).Error; err == nil {
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
