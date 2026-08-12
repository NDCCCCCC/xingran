package services

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ConfigBackupService 配置备份服务
type ConfigBackupService struct {
	db            *gorm.DB
	executor      *device.DeviceExecutor
	maxConcurrent int
}

// 配置备份并发数的默认配置键
const backupConcurrentConfigKey = "network.config.backup.concurrent"

// defaultBackupThresholdKB 配置备份变更阈值 fallback（单位：KB）
const defaultBackupThresholdKB = 100

// NewConfigBackupService 创建配置备份服务
func NewConfigBackupService(db *gorm.DB, executor *device.DeviceExecutor) *ConfigBackupService {
	svc := &ConfigBackupService{
		db:            db,
		executor:      executor,
		maxConcurrent: 5, // 默认值（配置备份较耗时，建议较低并发）
	}
	// 从数据库加载配置
	svc.loadConfigFromDB()
	return svc
}

// BackupRequest 备份请求
type BackupRequest struct {
	DeviceID      string
	DeviceName    string
	BackupType    models.BackupType
	ChangeReason  string
	CreatedBy     string
	CompressLarge bool // 是否压缩大文件
}

// BackupResult 备份结果
type BackupResult struct {
	BackupID     string
	DeviceID     string
	DeviceName   string
	StorageType  models.StorageType
	FilePath     string
	ConfigSize   int
	IsCompressed bool
	Version      int
}

// getDefaultThreshold 获取默认阈值（从系统参数读取，单位：KB）
func (s *ConfigBackupService) getDefaultThreshold(ctx context.Context) int {
	// 从系统参数中读取阈值配置
	var config models.Config
	err := s.db.WithContext(ctx).Where("config_key = ?", "network.config.backup.threshold").First(&config).Error
	if err != nil {
		// 默认阈值：100KB
		return defaultBackupThresholdKB
	}

	threshold, err := strconv.Atoi(config.ConfigValue)
	if err != nil {
		return defaultBackupThresholdKB // 默认100KB
	}
	return threshold
}

// getBackupDir 获取备份目录
func (s *ConfigBackupService) getBackupDir(deviceID string) string {
	// 备份目录结构: data/config-backups/{device_id}/{YYYY}/{MM}/
	now := time.Now()
	return filepath.Join("data", "config-backups", deviceID, now.Format("2006"), now.Format("01"))
}

// CreateBackup 创建配置备份
func (s *ConfigBackupService) CreateBackup(ctx context.Context, req *BackupRequest) (*BackupResult, error) {
	// 查询设备
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", req.DeviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("设备不存在: %w", err)
	}

	// 使用 executor 获取配置
	config, err := s.executor.GetConfig(ctx, req.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("获取配置失败: %w", err)
	}

	// 计算配置大小
	configSize := len(config)

	// 获取阈值
	threshold := s.getDefaultThreshold(ctx) // KB
	thresholdBytes := threshold * 1024

	// 获取最新版本号
	var latestBackup models.ConfigBackup
	s.db.WithContext(ctx).Where("device_id = ?", req.DeviceID).Order("version DESC").First(&latestBackup)
	newVersion := 1
	if latestBackup.ID != "" {
		newVersion = latestBackup.Version + 1
	}

	// 创建备份记录
	backup := &models.ConfigBackup{
		DeviceID:     req.DeviceID,
		DeviceName:   req.DeviceName,
		BackupType:   req.BackupType,
		ConfigHash:   calculateHash(config),
		Version:      newVersion,
		ChangeReason: req.ChangeReason,
		BackupSize:   configSize,
		CreatedBy:    req.CreatedBy,
	}

	// 根据大小决定存储方式
	if configSize < thresholdBytes {
		// 小配置：存储在数据库
		backup.StorageType = models.StorageTypeDatabase
		backup.ConfigContent = config
		backup.Compressed = false
	} else {
		// 大配置：存储在文件系统
		backup.StorageType = models.StorageTypeFile

		// 确保目录存在
		backupDir := s.getBackupDir(req.DeviceID)
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return nil, fmt.Errorf("创建备份目录失败: %w", err)
		}

		// 生成文件名
		timestamp := time.Now().Format("20060102_150405")
		fileName := fmt.Sprintf("%s_v%d_%s.conf", req.DeviceName, newVersion, timestamp)
		filePath := filepath.Join(backupDir, fileName)

		// 写入文件
		content := config
		if req.CompressLarge {
			// TODO: 实现压缩
			backup.Compressed = false
		} else {
			backup.Compressed = false
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("写入备份文件失败: %w", err)
		}

		backup.FilePath = filePath
	}

	// 保存备份记录
	if err := s.db.WithContext(ctx).Create(backup).Error; err != nil {
		return nil, fmt.Errorf("保存备份记录失败: %w", err)
	}

	return &BackupResult{
		BackupID:     backup.ID,
		DeviceID:     backup.DeviceID,
		DeviceName:   backup.DeviceName,
		StorageType:  backup.StorageType,
		FilePath:     backup.FilePath,
		ConfigSize:   backup.BackupSize,
		IsCompressed: backup.Compressed,
		Version:      backup.Version,
	}, nil
}

// GetBackupContent 获取备份内容
func (s *ConfigBackupService) GetBackupContent(ctx context.Context, backupID string) (string, error) {
	var backup models.ConfigBackup
	if err := s.db.WithContext(ctx).Where("id = ?", backupID).First(&backup).Error; err != nil {
		return "", fmt.Errorf("备份记录不存在: %w", err)
	}

	if backup.IsStoredInDatabase() {
		return backup.ConfigContent, nil
	}

	// 从文件读取
	content, err := os.ReadFile(backup.FilePath)
	if err != nil {
		return "", fmt.Errorf("读取备份文件失败: %w", err)
	}

	if backup.Compressed {
		// TODO: 解压
		return string(content), nil
	}

	return string(content), nil
}

// BatchBackupDevices 批量备份设备（并发执行）
func (s *ConfigBackupService) BatchBackupDevices(ctx context.Context, deviceIDs []string, backupType models.BackupType, createdBy string) ([]*BackupResult, error) {
	var results []*BackupResult
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, s.maxConcurrent) // 使用动态并发数

	applogger.Infof("[配置备份] 开始批量备份 %d 个设备", len(deviceIDs))

	for _, deviceID := range deviceIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var device models.NetworkDevice
			if err := s.db.WithContext(ctx).Where("id = ?", id).First(&device).Error; err != nil {
				applogger.Infof("[配置备份] 设备不存在，跳过: %s", id)
				return // 跳过不存在的设备
			}

			req := &BackupRequest{
				DeviceID:      id,
				DeviceName:    device.DeviceName,
				BackupType:    backupType,
				ChangeReason:  "",
				CreatedBy:     createdBy,
				CompressLarge: true,
			}

			applogger.Infof("[配置备份] 开始备份设备: %s (%s)", device.DeviceName, device.IPAddress)
			result, err := s.CreateBackup(ctx, req)
			if err != nil {
				applogger.Infof("[配置备份] 备份失败 [%s]: %v", device.DeviceName, err)
				return
			}

			applogger.Infof("[配置备份] 备份成功 [%s] 版本: %d, 大小: %d 字节", device.DeviceName, result.Version, result.ConfigSize)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(deviceID)
	}

	wg.Wait()
	applogger.Infof("[配置备份] 批量备份完成，成功: %d/%d", len(results), len(deviceIDs))
	return results, nil
}

// AutoBackupAllDevices 自动备份所有设备（智能备份：配置未改变则仅更新时间）
func (s *ConfigBackupService) AutoBackupAllDevices(ctx context.Context) error {
	var devices []models.NetworkDevice
	if err := s.db.WithContext(ctx).Find(&devices).Error; err != nil {
		return fmt.Errorf("查询设备列表失败: %w", err)
	}

	if len(devices) == 0 {
		return nil
	}

	applogger.Infof("[配置备份] 开始智能备份 %d 个设备", len(devices))

	var successCount, skippedCount int
	for _, device := range devices {
		skipped, err := s.AutoBackupSingleDevice(ctx, &device)
		if err != nil {
			applogger.Infof("[配置备份] 备份失败 [%s]: %v", device.DeviceName, err)
			continue
		}
		if skipped {
			skippedCount++
		} else {
			successCount++
		}
	}

	applogger.Infof("[配置备份] 智能备份完成: 新备份=%d, 配置未变=%d, 总计=%d", successCount, skippedCount, len(devices))
	return nil
}

// AutoBackupSingleDevice 自动备份单个设备（智能备份：配置未改变则仅更新时间）
// 返回值: skipped-是否跳过创建新备份（配置未改变）, error-错误信息
func (s *ConfigBackupService) AutoBackupSingleDevice(ctx context.Context, device *models.NetworkDevice) (bool, error) {
	// 使用 executor 获取配置
	config, err := s.executor.GetConfig(ctx, device.ID)
	if err != nil {
		return false, fmt.Errorf("获取配置失败: %w", err)
	}

	// 计算当前配置的哈希值
	currentHash := calculateHash(config)

	// 查询该设备最近的一次备份
	var latestBackup models.ConfigBackup
	err = s.db.WithContext(ctx).
		Where("device_id = ?", device.ID).
		Order("created_at DESC, version DESC").
		First(&latestBackup).Error

	if err != nil {
		// 没有找到历史备份，创建新备份
		applogger.Infof("[配置备份] 设备 %s 无历史备份，创建新备份", device.DeviceName)
		return s.createNewAutoBackup(ctx, device, config, currentHash)
	}

	// 比较配置哈希
	if latestBackup.ConfigHash == currentHash {
		// 配置未改变，仅更新 UpdatedAt
		applogger.Infof("[配置备份] 设备 %s 配置未改变，更新时间", device.DeviceName)
		if err := s.db.WithContext(ctx).
			Model(&latestBackup).
			Update("updated_at", time.Now()).Error; err != nil {
			return false, fmt.Errorf("更新备份时间失败: %w", err)
		}
		return true, nil // skipped = true
	}

	// 配置已改变，创建新备份
	applogger.Infof("[配置备份] 设备 %s 配置已改变，创建新备份 (版本: %d -> %d)",
		device.DeviceName, latestBackup.Version, latestBackup.Version+1)
	return s.createNewAutoBackup(ctx, device, config, currentHash)
}

// createNewAutoBackup 创建新的自动备份记录
// 返回值: skipped-始终为false（创建了新备份）, error-错误信息
func (s *ConfigBackupService) createNewAutoBackup(ctx context.Context, device *models.NetworkDevice, config, configHash string) (bool, error) {
	configSize := len(config)

	// 获取阈值
	threshold := s.getDefaultThreshold(ctx) // KB
	thresholdBytes := threshold * 1024

	// 获取最新版本号
	var latestBackup models.ConfigBackup
	s.db.WithContext(ctx).Where("device_id = ?", device.ID).Order("version DESC").First(&latestBackup)
	newVersion := 1
	if latestBackup.ID != "" {
		newVersion = latestBackup.Version + 1
	}

	// 创建备份记录
	backup := &models.ConfigBackup{
		DeviceID:     device.ID,
		DeviceName:   device.DeviceName,
		BackupType:   models.BackupTypeAuto,
		ConfigHash:   configHash,
		Version:      newVersion,
		ChangeReason: "自动备份",
		BackupSize:   configSize,
		CreatedBy:    "system",
	}

	// 根据大小决定存储方式
	if configSize < thresholdBytes {
		// 小配置：存储在数据库
		backup.StorageType = models.StorageTypeDatabase
		backup.ConfigContent = config
		backup.Compressed = false
	} else {
		// 大配置：存储在文件系统
		backup.StorageType = models.StorageTypeFile

		// 确保目录存在
		backupDir := s.getBackupDir(device.ID)
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return false, fmt.Errorf("创建备份目录失败: %w", err)
		}

		// 生成文件名
		timestamp := time.Now().Format("20060102_150405")
		fileName := fmt.Sprintf("%s_v%d_%s.conf", device.DeviceName, newVersion, timestamp)
		filePath := filepath.Join(backupDir, fileName)

		// 写入文件
		if err := os.WriteFile(filePath, []byte(config), 0644); err != nil {
			return false, fmt.Errorf("写入备份文件失败: %w", err)
		}

		backup.FilePath = filePath
		backup.Compressed = false
	}

	// 保存备份记录
	if err := s.db.WithContext(ctx).Create(backup).Error; err != nil {
		return false, fmt.Errorf("保存备份记录失败: %w", err)
	}

	applogger.Infof("[配置备份] 备份成功 [%s] 版本: %d, 大小: %d 字节", device.DeviceName, newVersion, configSize)
	return false, nil // 创建了新备份，skipped = false
}

// BackupListItem 备份列表项（包含设备信息）
type BackupListItem struct {
	models.ConfigBackup
	IPAddress string `json:"ipAddress"`
}

// GetBackupList 获取备份列表（包含设备IP地址）
// orderByColumn/isAsc 为服务端排序参数(可选,透传给 base.ApplySort 白名单)。
func (s *ConfigBackupService) GetBackupList(ctx context.Context, current, pageSize int, deviceID string, orderByColumn string, isAsc *bool) ([]BackupListItem, int64, error) {
	var backups []models.ConfigBackup
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ConfigBackup{})

	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询备份记录总数失败: %w", err)
	}

	// 分页查询 - 用户排序(白名单)优先,无 OrderByColumn 时保留 created_at DESC 默认
	offset := (current - 1) * pageSize
	sortReq := base.BaseListRequest{
		Current:       current,
		PageSize:      pageSize,
		OrderByColumn: orderByColumn,
		IsAsc:         isAsc,
	}
	query = base.ApplySort(query, sortReq, configBackupAllowedSortFields)
	if orderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Offset(offset).Limit(pageSize).Find(&backups).Error; err != nil {
		return nil, 0, fmt.Errorf("查询备份记录失败: %w", err)
	}

	// 查询设备信息并组装返回数据
	var result []BackupListItem
	for _, backup := range backups {
		var device models.NetworkDevice
		ipAddress := ""
		if err := s.db.WithContext(ctx).Where("id = ?", backup.DeviceID).First(&device).Error; err == nil {
			ipAddress = device.IPAddress
		}

		result = append(result, BackupListItem{
			ConfigBackup: backup,
			IPAddress:    ipAddress,
		})
	}

	return result, total, nil
}

// configBackupAllowedSortFields 配置备份可排序字段白名单(对应 sys_config_backup 表列名)。
var configBackupAllowedSortFields = map[string]string{
	"deviceId":  "device_id",
	"version":   "version",
	"status":    "status",
	"createdAt": "created_at",
}

// GetBackupByID 获取备份详情
func (s *ConfigBackupService) GetBackupByID(ctx context.Context, backupID string) (*models.ConfigBackup, error) {
	var backup models.ConfigBackup
	if err := s.db.WithContext(ctx).Where("id = ?", backupID).First(&backup).Error; err != nil {
		return nil, fmt.Errorf("备份记录不存在: %w", err)
	}
	return &backup, nil
}

// DeleteBackup 删除备份
func (s *ConfigBackupService) DeleteBackup(ctx context.Context, backupID string) error {
	var backup models.ConfigBackup
	if err := s.db.WithContext(ctx).Where("id = ?", backupID).First(&backup).Error; err != nil {
		return fmt.Errorf("备份记录不存在: %w", err)
	}

	// 如果是文件存储，删除文件
	if backup.IsStoredInFile() && backup.FilePath != "" {
		_ = os.Remove(backup.FilePath)
	}

	// 软删除
	if err := s.db.WithContext(ctx).Delete(&backup).Error; err != nil {
		return fmt.Errorf("删除备份记录失败: %w", err)
	}

	return nil
}

// BatchDeleteBackups 批量删除备份
func (s *ConfigBackupService) BatchDeleteBackups(ctx context.Context, backupIDs []string) error {
	for _, backupID := range backupIDs {
		if err := s.DeleteBackup(ctx, backupID); err != nil {
			continue // 继续处理其他备份
		}
	}
	return nil
}

// DiffBackups 对比两个备份的差异
func (s *ConfigBackupService) DiffBackups(ctx context.Context, backupID1, backupID2 string) (string, string, string, error) {
	content1, err := s.GetBackupContent(ctx, backupID1)
	if err != nil {
		return "", "", "", err
	}

	content2, err := s.GetBackupContent(ctx, backupID2)
	if err != nil {
		return "", "", "", err
	}

	// 获取备份信息
	var backup1, backup2 models.ConfigBackup
	s.db.WithContext(ctx).Where("id = ?", backupID1).First(&backup1)
	s.db.WithContext(ctx).Where("id = ?", backupID2).First(&backup2)

	// 简单的差异对比（可以使用更复杂的diff库）
	diffResult := generateDiff(content1, content2)

	return backup1.DeviceName + " (版本" + strconv.Itoa(backup1.Version) + ")",
		backup2.DeviceName + " (版本" + strconv.Itoa(backup2.Version) + ")",
		diffResult, nil
}

// RestoreBackup 恢复配置（预留接口，实际需要设备支持）
func (s *ConfigBackupService) RestoreBackup(ctx context.Context, backupID string, deviceID string) error {
	// 获取备份内容
	_, err := s.GetBackupContent(ctx, backupID)
	if err != nil {
		return err
	}

	// TODO: 实现配置恢复逻辑
	// 这需要根据设备厂商和型号使用不同的命令
	return fmt.Errorf("配置恢复功能待实现")
}

// GetBackupStatistics 获取备份统计信息
func (s *ConfigBackupService) GetBackupStatistics(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalBackups     int64
		AutoBackups      int64
		ManualBackups    int64
		DBStorageCount   int64
		FileStorageCount int64
		TotalSize        int64
	}

	s.db.WithContext(ctx).Model(&models.ConfigBackup{}).Count(&stats.TotalBackups)
	s.db.WithContext(ctx).Model(&models.ConfigBackup{}).Where("backup_type = ?", models.BackupTypeAuto).Count(&stats.AutoBackups)
	s.db.WithContext(ctx).Model(&models.ConfigBackup{}).Where("backup_type = ?", models.BackupTypeManual).Count(&stats.ManualBackups)
	s.db.WithContext(ctx).Model(&models.ConfigBackup{}).Where("storage_type = ?", models.StorageTypeDatabase).Count(&stats.DBStorageCount)
	s.db.WithContext(ctx).Model(&models.ConfigBackup{}).Where("storage_type = ?", models.StorageTypeFile).Count(&stats.FileStorageCount)

	// 计算总大小
	var result struct {
		TotalSize int64
	}
	s.db.WithContext(ctx).Model(&models.ConfigBackup{}).Select("COALESCE(SUM(backup_size), 0) as total_size").Scan(&result)
	stats.TotalSize = result.TotalSize

	// 唯一设备数(供统计卡片,替代旧前端用 Set(list).size 的反模式)
	var uniqueDevices int64
	s.db.WithContext(ctx).Model(&models.ConfigBackup{}).Distinct("device_id").Count(&uniqueDevices)

	return map[string]interface{}{
		"totalBackups":     stats.TotalBackups,
		"autoBackups":      stats.AutoBackups,
		"manualBackups":    stats.ManualBackups,
		"dbStorageCount":   stats.DBStorageCount,
		"fileStorageCount": stats.FileStorageCount,
		"totalSize":        stats.TotalSize,
		"totalSizeMB":      stats.TotalSize / 1024 / 1024,
		"uniqueDevices":    uniqueDevices,
	}, nil
}

// calculateHash 计算配置内容的哈希值
func calculateHash(content string) string {
	hash := md5.Sum([]byte(content))
	return hex.EncodeToString(hash[:])
}

// generateDiff 生成差异对比（简化版本）
func generateDiff(content1, content2 string) string {
	lines1 := strings.Split(content1, "\n")
	lines2 := strings.Split(content2, "\n")

	var diff strings.Builder
	maxLen := len(lines1)
	if len(lines2) > maxLen {
		maxLen = len(lines2)
	}

	for i := 0; i < maxLen; i++ {
		line1 := ""
		line2 := ""

		if i < len(lines1) {
			line1 = lines1[i]
		}
		if i < len(lines2) {
			line2 = lines2[i]
		}

		if line1 == line2 {
			diff.WriteString("  " + line1 + "\n")
		} else {
			if line1 != "" {
				diff.WriteString("- " + line1 + "\n")
			}
			if line2 != "" {
				diff.WriteString("+ " + line2 + "\n")
			}
		}
	}

	return diff.String()
}

// loadConfigFromDB 从数据库加载并发数配置
func (s *ConfigBackupService) loadConfigFromDB() {
	var config models.Config
	err := s.db.Where("config_key = ?", backupConcurrentConfigKey).First(&config).Error
	if err == nil && config.ConfigValue != "" {
		if concurrent, parseErr := strconv.Atoi(config.ConfigValue); parseErr == nil && concurrent > 0 {
			s.maxConcurrent = concurrent
			applogger.Infof("[配置备份] 从数据库读取并发数配置: %d", s.maxConcurrent)
		}
	}
}

// ReloadConfig 重新加载配置（从数据库）
func (s *ConfigBackupService) ReloadConfig() {
	s.loadConfigFromDB()
}
