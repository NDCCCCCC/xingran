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
	"time"

	"gorm.io/gorm"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// ConfigBackupService 配置备份服务
type ConfigBackupService struct {
	db      *gorm.DB
	manager *device.Manager
}

// NewConfigBackupService 创建配置备份服务
func NewConfigBackupService(db *gorm.DB, manager *device.Manager) *ConfigBackupService {
	return &ConfigBackupService{
		db:      db,
		manager: manager,
	}
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
		return 100
	}

	threshold, err := strconv.Atoi(config.ConfigValue)
	if err != nil {
		return 100 // 默认100KB
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

	// 获取设备凭证
	credential, err := s.getDeviceCredential(ctx, &device)
	if err != nil {
		return nil, fmt.Errorf("获取设备凭证失败: %w", err)
	}

	// 连接设备
	wrapper, err := s.manager.ConnectToDevice(ctx, &device, credential)
	if err != nil {
		return nil, fmt.Errorf("连接设备失败: %w", err)
	}
	defer s.manager.DisconnectFromDevice(device.ID)

	// 获取配置
	config, err := wrapper.GetConfig()
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

// BatchBackupDevices 批量备份设备
func (s *ConfigBackupService) BatchBackupDevices(ctx context.Context, deviceIDs []string, backupType models.BackupType, createdBy string) ([]*BackupResult, error) {
	var results []*BackupResult

	for _, deviceID := range deviceIDs {
		var device models.NetworkDevice
		if err := s.db.WithContext(ctx).Where("id = ?", deviceID).First(&device).Error; err != nil {
			continue // 跳过不存在的设备
		}

		req := &BackupRequest{
			DeviceID:     deviceID,
			DeviceName:   device.DeviceName,
			BackupType:   backupType,
			ChangeReason: "",
			CreatedBy:    createdBy,
			CompressLarge: true,
		}

		result, err := s.CreateBackup(ctx, req)
		if err != nil {
			// 记录错误但继续处理其他设备
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

// AutoBackupAllDevices 自动备份所有设备
func (s *ConfigBackupService) AutoBackupAllDevices(ctx context.Context) error {
	var devices []models.NetworkDevice
	if err := s.db.WithContext(ctx).Find(&devices).Error; err != nil {
		return fmt.Errorf("查询设备列表失败: %w", err)
	}

	if len(devices) == 0 {
		return nil
	}

	var deviceIDs []string
	for _, device := range devices {
		deviceIDs = append(deviceIDs, device.ID)
	}

	_, err := s.BatchBackupDevices(ctx, deviceIDs, models.BackupTypeAuto, "system")
	return err
}

// GetBackupList 获取备份列表
func (s *ConfigBackupService) GetBackupList(ctx context.Context, current, pageSize int, deviceID string) ([]models.ConfigBackup, int64, error) {
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

	// 分页查询
	offset := (current - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&backups).Error; err != nil {
		return nil, 0, fmt.Errorf("查询备份记录失败: %w", err)
	}

	return backups, total, nil
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
		if err := os.Remove(backup.FilePath); err != nil {
			// 文件删除失败不影响数据库删除
		}
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
	config, err := s.GetBackupContent(ctx, backupID)
	if err != nil {
		return err
	}

	// 获取设备
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", deviceID).First(&device).Error; err != nil {
		return fmt.Errorf("设备不存在: %w", err)
	}

	// 获取凭证
	credential, err := s.getDeviceCredential(ctx, &device)
	if err != nil {
		return err
	}

	// 连接设备
	wrapper, err := s.manager.ConnectToDevice(ctx, &device, credential)
	if err != nil {
		return err
	}
	defer s.manager.DisconnectFromDevice(device.ID)

	// TODO: 实现配置恢复逻辑
	// 这需要根据设备厂商和型号使用不同的命令
	_ = wrapper
	_ = config

	return fmt.Errorf("配置恢复功能待实现")
}

// GetBackupStatistics 获取备份统计信息
func (s *ConfigBackupService) GetBackupStatistics(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalBackups    int64
		AutoBackups     int64
		ManualBackups   int64
		DBStorageCount  int64
		FileStorageCount int64
		TotalSize       int64
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

	return map[string]interface{}{
		"totalBackups":     stats.TotalBackups,
		"autoBackups":      stats.AutoBackups,
		"manualBackups":    stats.ManualBackups,
		"dbStorageCount":   stats.DBStorageCount,
		"fileStorageCount": stats.FileStorageCount,
		"totalSize":        stats.TotalSize,
		"totalSizeMB":      stats.TotalSize / 1024 / 1024,
	}, nil
}

// getDeviceCredential 获取设备凭证
func (s *ConfigBackupService) getDeviceCredential(ctx context.Context, device *models.NetworkDevice) (*models.AuthCredential, error) {
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
