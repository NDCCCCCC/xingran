package services

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// MAC历史保留期配置常量 (F-BE-39)
const (
	// defaultMACHistoryRetentionDays MAC历史数据默认保留期（天）
	defaultMACHistoryRetentionDays = 120
	// minMACHistoryRetentionDays MAC历史数据最小保留期（天），低于该值视为误配置
	minMACHistoryRetentionDays = 30
)

// PartitionService MAC历史表分区管理服务接口
type PartitionService interface {
	// CreateMonthlyPartition 创建指定年月的月度分区
	CreateMonthlyPartition(ctx context.Context, year int, month int) error

	// EnsurePartitionsExist 确保未来N个月的分区存在
	EnsurePartitionsExist(ctx context.Context, monthsAhead int) error

	// DropExpiredPartitions 删除过期的分区
	DropExpiredPartitions(ctx context.Context) error

	// GetRetentionDays 获取配置的数据保留期（天）
	GetRetentionDays(ctx context.Context) int
}

// partitionServiceImpl 分区管理服务私有实现
type partitionServiceImpl struct {
	db *gorm.DB
}

// NewPartitionService 创建分区管理服务实例
func NewPartitionService(db *gorm.DB) PartitionService {
	return &partitionServiceImpl{db: db}
}

// CreateMonthlyPartition 创建指定年月的月度分区
// 分区名称格式: sys_device_mac_history_YYYY_MM (如 sys_device_mac_history_2025_01)
func (s *partitionServiceImpl) CreateMonthlyPartition(ctx context.Context, year int, month int) error {
	// 验证年份和月份范围
	if year < 2020 || year > 2100 {
		return fmt.Errorf("无效的年份: %d，有效范围: 2020-2100", year)
	}
	if month < 1 || month > 12 {
		return fmt.Errorf("无效的月份: %d，有效范围: 1-12", month)
	}

	// 生成分区名称
	partitionName := fmt.Sprintf("sys_device_mac_history_%d_%02d", year, month)

	// 验证分区名称格式，防止SQL注入（CR-01 fix）
	// WR-08 fix: 使用更严格的模式防止连续下划线
	validPartitionName := regexp.MustCompile(`^[a-z]+(?:_[a-z]+)*_[0-9]{4}_[0-9]{2}$`)
	if !validPartitionName.MatchString(partitionName) {
		return fmt.Errorf("非法的分区名称: %s", partitionName)
	}

	// 计算分区边界
	startDate := fmt.Sprintf("'%d-%02d-01'", year, month)
	endYear := year
	endMonth := month + 1
	if endMonth > 12 {
		endYear++
		endMonth = 1
	}
	endDate := fmt.Sprintf("'%d-%02d-01'", endYear, endMonth)

	// 生成 CREATE PARTITION SQL DDL
	sql := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		PARTITION OF sys_device_mac_history
		FOR VALUES FROM (%s) TO (%s)
	`, partitionName, startDate, endDate)

	// 执行 DDL
	if err := s.db.WithContext(ctx).Exec(sql).Error; err != nil {
		return fmt.Errorf("创建分区 %s 失败: %w", partitionName, err)
	}

	applogger.Infof("成功创建MAC历史表分区: %s (范围: %s ~ %s)", partitionName, startDate, endDate)
	return nil
}

// EnsurePartitionsExist 确保未来N个月的分区存在
// monthsAhead: 未来月数（默认2）
func (s *partitionServiceImpl) EnsurePartitionsExist(ctx context.Context, monthsAhead int) error {
	if monthsAhead <= 0 {
		monthsAhead = 2 // 默认创建未来2个月的分区
	}

	now := time.Now()
	applogger.Infof("开始检查MAC历史表分区，将确保未来 %d 个月的分区存在", monthsAhead)

	// 遍历未来N个月，确保每个分区都存在
	var creationErrors []error
	for i := 0; i <= monthsAhead; i++ {
		targetDate := now.AddDate(0, i, 0)
		year := targetDate.Year()
		month := int(targetDate.Month())

		// 尝试创建分区（IF NOT EXISTS 会忽略已存在的分区）
		if err := s.CreateMonthlyPartition(ctx, year, month); err != nil {
			applogger.Errorf("创建 %d年%02d月 分区失败: %v", year, month, err)
			creationErrors = append(creationErrors, err)
		}
	}

	// If all partitions failed to create, return error
	if len(creationErrors) > 0 && len(creationErrors) > monthsAhead {
		return fmt.Errorf("创建所有MAC历史分区失败，共 %d 个错误", len(creationErrors))
	}

	applogger.Infof("MAC历史表分区检查完成")
	return nil
}

// GetRetentionDays 获取配置的数据保留期（天）
// 配置键: network.mac.history.retention_days
// 默认值: 120 天
func (s *partitionServiceImpl) GetRetentionDays(ctx context.Context) int {
	// 从 sys_config 表读取配置
	var config models.Config
	err := s.db.WithContext(ctx).
		Where("config_key = ?", "network.mac.history.retention_days").
		First(&config).Error

	if err == nil && config.ConfigValue != "" {
		// 尝试解析为整数
		if days, parseErr := strconv.Atoi(config.ConfigValue); parseErr == nil && days > 0 {
			// 验证最小值为30天（防止误配置导致数据丢失）
			if days < minMACHistoryRetentionDays {
				applogger.Warnf("配置的保留期 %d 天小于最小值%d天，使用默认值%d天",
					days, minMACHistoryRetentionDays, defaultMACHistoryRetentionDays)
				return defaultMACHistoryRetentionDays
			}
			return days
		}
	}

	// 配置不存在或解析失败，返回默认值
	return defaultMACHistoryRetentionDays
}

// DropExpiredPartitions 删除过期的分区
// 根据保留期配置，删除过期的完整月度分区
func (s *partitionServiceImpl) DropExpiredPartitions(ctx context.Context) error {
	retentionDays := s.GetRetentionDays(ctx)
	applogger.Infof("开始清理过期的MAC历史表分区（保留期: %d 天）", retentionDays)

	// 计算截止日期
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	applogger.Infof("截止日期: %s（将删除此日期之前的完整月度分区）", cutoffDate.Format("2006-01-02"))

	// 查询所有现有分区
	// pg_inherits 表存储了分区表的继承关系
	partitionsQuery := `
		SELECT
			inhrelid::regclass AS partition_name
		FROM
			pg_catalog.pg_inherits
		WHERE
			inhparent = 'sys_device_mac_history'::regclass
		ORDER BY
			partition_name
	`

	type PartitionInfo struct {
		PartitionName string `gorm:"column:partition_name"`
	}

	var partitions []PartitionInfo
	if err := s.db.WithContext(ctx).Raw(partitionsQuery).Scan(&partitions).Error; err != nil {
		return fmt.Errorf("查询分区列表失败: %w", err)
	}

	if len(partitions) == 0 {
		applogger.Infof("没有找到任何MAC历史表分区")
		return nil
	}

	applogger.Infof("找到 %d 个MAC历史表分区", len(partitions))

	// 解析分区名称并删除过期分区
	// 分区名称格式: sys_device_mac_history_YYYY_MM
	partitionPattern := regexp.MustCompile(`^sys_device_mac_history_(\d{4})_(\d{2})$`)
	droppedCount := 0

	for _, partition := range partitions {
		matches := partitionPattern.FindStringSubmatch(partition.PartitionName)
		if matches == nil {
			applogger.Warnf("跳过格式不匹配的分区: %s", partition.PartitionName)
			continue
		}

		year, err := strconv.Atoi(matches[1])
		if err != nil {
			applogger.Warnf("跳过年份解析失败的分区: %s", partition.PartitionName)
			continue
		}

		month, err := strconv.Atoi(matches[2])
		if err != nil {
			applogger.Warnf("跳过月份解析失败的分区: %s", partition.PartitionName)
			continue
		}

		// 计算分区的结束日期（下个月1号）
		partitionEnd := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)

		// 如果分区结束日期早于截止日期，则删除该分区
		if partitionEnd.Before(cutoffDate) {
			dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", partition.PartitionName)
			if err := s.db.WithContext(ctx).Exec(dropSQL).Error; err != nil {
				applogger.Errorf("删除分区 %s 失败: %v", partition.PartitionName, err)
				continue
			}

			applogger.Infof("成功删除过期分区: %s (分区结束: %s)", partition.PartitionName, partitionEnd.Format("2006-01-02"))
			droppedCount++
		} else {
			applogger.Debugf("保留分区: %s (分区结束: %s)", partition.PartitionName, partitionEnd.Format("2006-01-02"))
		}
	}

	applogger.Infof("MAC历史表分区清理完成，删除了 %d 个过期分区", droppedCount)
	return nil
}
