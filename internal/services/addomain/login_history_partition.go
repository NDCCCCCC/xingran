package addomain

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

// AD 登录历史保留期配置常量(quick 261016-ad-login-history)
const (
	// defaultADLoginHistoryRetentionDays 默认保留期(天)
	defaultADLoginHistoryRetentionDays = 120
	// minADLoginHistoryRetentionDays 最小保留期(天),低于该值视为误配置
	minADLoginHistoryRetentionDays = 30
	// ADLoginHistoryRetentionConfigKey sys_config 配置键(web 可调保留期)
	ADLoginHistoryRetentionConfigKey = "ops.ad.login_history.retention_days"
)

// ADLoginHistoryPartitionService AD 登录历史表分区管理服务接口
// (整体仿 internal/services/mac_history_partition.go,单模块内聚于 addomain 包)
type ADLoginHistoryPartitionService interface {
	// CreateMonthlyPartition 创建指定年月的月度分区
	CreateMonthlyPartition(ctx context.Context, year int, month int) error

	// EnsurePartitionsExist 确保未来N个月的分区存在
	EnsurePartitionsExist(ctx context.Context, monthsAhead int) error

	// DropExpiredPartitions 删除过期的分区
	DropExpiredPartitions(ctx context.Context) error

	// GetRetentionDays 获取配置的数据保留期(天)
	GetRetentionDays(ctx context.Context) int
}

// adLoginHistoryPartitionServiceImpl 分区管理服务私有实现
type adLoginHistoryPartitionServiceImpl struct {
	db *gorm.DB
}

// NewADLoginHistoryPartitionService 创建分区管理服务实例
func NewADLoginHistoryPartitionService(db *gorm.DB) ADLoginHistoryPartitionService {
	return &adLoginHistoryPartitionServiceImpl{db: db}
}

// isPostgres 方言守卫:表分区(PARTITION OF / pg_inherits)是 PG 专属特性,
// sqlite 下分区管理整体跳过(quick 261016 风险 8)。
func (s *adLoginHistoryPartitionServiceImpl) isPostgres() bool {
	return s.db != nil && s.db.Dialector.Name() == "postgres"
}

// CreateMonthlyPartition 创建指定年月的月度分区
// 分区名称格式: sys_ad_login_history_YYYY_MM (如 sys_ad_login_history_2026_10)
func (s *adLoginHistoryPartitionServiceImpl) CreateMonthlyPartition(ctx context.Context, year int, month int) error {
	// SQLite 不支持表分区,直接跳过(由 EnsurePartitionsExist 的守卫兜底,此处双保险)
	if !s.isPostgres() {
		applogger.Debugf("非 PostgreSQL 数据库,跳过创建AD登录历史分区: %d-%02d", year, month)
		return nil
	}
	// 验证年份和月份范围
	if year < 2020 || year > 2100 {
		return fmt.Errorf("无效的年份: %d，有效范围: 2020-2100", year)
	}
	if month < 1 || month > 12 {
		return fmt.Errorf("无效的月份: %d，有效范围: 1-12", month)
	}

	// 生成分区名称
	partitionName := fmt.Sprintf("sys_ad_login_history_%d_%02d", year, month)

	// 验证分区名称格式,防止 SQL 注入(CR-01 同款)
	validPartitionName := regexp.MustCompile(`^sys_ad_login_history_(\d{4})_(\d{2})$`)
	if !validPartitionName.MatchString(partitionName) {
		return fmt.Errorf("非法的分区名称: %s", partitionName)
	}

	// 计算分区边界 [当月1号, 次月1号)
	startDate := fmt.Sprintf("'%d-%02d-01'", year, month)
	endYear := year
	endMonth := month + 1
	if endMonth > 12 {
		endYear++
		endMonth = 1
	}
	endDate := fmt.Sprintf("'%d-%02d-01'", endYear, endMonth)

	sql := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		PARTITION OF sys_ad_login_history
		FOR VALUES FROM (%s) TO (%s)
	`, partitionName, startDate, endDate)

	if err := s.db.WithContext(ctx).Exec(sql).Error; err != nil {
		return fmt.Errorf("创建分区 %s 失败: %w", partitionName, err)
	}

	applogger.Infof("成功创建AD登录历史表分区: %s (范围: %s ~ %s)", partitionName, startDate, endDate)
	return nil
}

// EnsurePartitionsExist 确保未来N个月的分区存在
func (s *adLoginHistoryPartitionServiceImpl) EnsurePartitionsExist(ctx context.Context, monthsAhead int) error {
	if !s.isPostgres() {
		applogger.Debugf("非 PostgreSQL 数据库,跳过AD登录历史分区检查")
		return nil
	}

	if monthsAhead <= 0 {
		monthsAhead = 2 // 默认创建未来2个月的分区
	}

	now := time.Now()
	applogger.Infof("开始检查AD登录历史表分区，将确保未来 %d 个月的分区存在", monthsAhead)

	var creationErrors []error
	for i := 0; i <= monthsAhead; i++ {
		targetDate := now.AddDate(0, i, 0)
		year := targetDate.Year()
		month := int(targetDate.Month())

		// IF NOT EXISTS 会忽略已存在的分区
		if err := s.CreateMonthlyPartition(ctx, year, month); err != nil {
			applogger.Errorf("创建 %d年%02d月 分区失败: %v", year, month, err)
			creationErrors = append(creationErrors, err)
		}
	}

	if len(creationErrors) > 0 && len(creationErrors) > monthsAhead {
		return fmt.Errorf("创建所有AD登录历史分区失败，共 %d 个错误", len(creationErrors))
	}

	applogger.Infof("AD登录历史表分区检查完成")
	return nil
}

// GetRetentionDays 获取配置的数据保留期(天)
// 配置键: ops.ad.login_history.retention_days
// 默认 120 天;配置值低于最小值 30 天视为误配置,回退默认值 + Warn
func (s *adLoginHistoryPartitionServiceImpl) GetRetentionDays(ctx context.Context) int {
	var config models.Config
	err := s.db.WithContext(ctx).
		Where("config_key = ?", ADLoginHistoryRetentionConfigKey).
		First(&config).Error

	if err == nil && config.ConfigValue != "" {
		if days, parseErr := strconv.Atoi(config.ConfigValue); parseErr == nil && days > 0 {
			if days < minADLoginHistoryRetentionDays {
				applogger.Warnf("配置的AD登录历史保留期 %d 天小于最小值%d天，使用默认值%d天",
					days, minADLoginHistoryRetentionDays, defaultADLoginHistoryRetentionDays)
				return defaultADLoginHistoryRetentionDays
			}
			return days
		}
	}

	return defaultADLoginHistoryRetentionDays
}

// DropExpiredPartitions 删除过期的分区(按保留期配置,删除过期的完整月度分区)
func (s *adLoginHistoryPartitionServiceImpl) DropExpiredPartitions(ctx context.Context) error {
	if !s.isPostgres() {
		applogger.Debugf("非 PostgreSQL 数据库,跳过AD登录历史过期分区清理")
		return nil
	}

	retentionDays := s.GetRetentionDays(ctx)
	applogger.Infof("开始清理过期的AD登录历史表分区（保留期: %d 天）", retentionDays)

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	applogger.Infof("截止日期: %s（将删除此日期之前的完整月度分区）", cutoffDate.Format("2006-01-02"))

	// 查询 sys_ad_login_history 的所有分区(pg_inherits 系统目录)
	partitionsQuery := `
		SELECT
			inhrelid::regclass AS partition_name
		FROM
			pg_catalog.pg_inherits
		WHERE
			inhparent = 'sys_ad_login_history'::regclass
		ORDER BY
			partition_name
	`

	type PartitionInfo struct {
		PartitionName string `gorm:"column:partition_name"`
	}

	var partitions []PartitionInfo
	if err := s.db.WithContext(ctx).Raw(partitionsQuery).Scan(&partitions).Error; err != nil {
		return fmt.Errorf("查询AD登录历史分区列表失败: %w", err)
	}

	if len(partitions) == 0 {
		applogger.Infof("没有找到任何AD登录历史表分区")
		return nil
	}

	applogger.Infof("找到 %d 个AD登录历史表分区", len(partitions))

	partitionPattern := regexp.MustCompile(`^sys_ad_login_history_(\d{4})_(\d{2})$`)
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

		// 分区结束日期(下个月1号)
		partitionEnd := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)

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

	applogger.Infof("AD登录历史表分区清理完成，删除了 %d 个过期分区", droppedCount)
	return nil
}
