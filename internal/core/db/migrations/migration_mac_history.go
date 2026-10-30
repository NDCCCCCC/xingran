package migrations

import (
	"fmt"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// isPostgreSQL 在 migration_helpers.go 中统一定义。

// CreateMACHistoryTable 创建MAC地址历史表（分区表 + BRIN索引）
// 按 CONTEXT.md D-04, D-05 决策：
// - 使用月度分区（按 first_seen 字段）
// - 使用 BRIN 索引优化时间序列查询
// - 设备名称快照（非外键，确保设备删除后历史可读）
func CreateMACHistoryTable(db *gorm.DB) error {
	// 仅在PostgreSQL中创建分区表
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] MAC历史表跳过创建（非PostgreSQL数据库）")
		return nil
	}

	applogger.Infof("[迁移] 开始创建MAC地址历史表（分区表）")

	// 1. 创建主表（声明为 PARTITION BY RANGE）
	// 注意：分区表的唯一约束必须包含所有分区列，因此不使用 PRIMARY KEY
	// 使用 NOT NULL 约束和唯一索引替代
	createMainTableSQL := `
	CREATE TABLE IF NOT EXISTS sys_device_mac_history (
		id UUID NOT NULL DEFAULT gen_random_uuid(),
		device_id UUID NOT NULL,
		device_name_snapshot VARCHAR(100),
		mac_address VARCHAR(30) NOT NULL,
		interface_name VARCHAR(100) NOT NULL,
		vlan_id INT,
		event_type VARCHAR(20) NOT NULL,
		first_seen TIMESTAMP NOT NULL,
		last_seen TIMESTAMP NOT NULL,
		collected_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	) PARTITION BY RANGE (first_seen);
	`

	if err := db.Exec(createMainTableSQL).Error; err != nil {
		return fmt.Errorf("创建MAC历史主表失败: %w", err)
	}
	applogger.Infof("[迁移] MAC历史主表创建成功")

	// 2. 创建当前月份和未来2个月的分区
	now := time.Now()
	monthsToCreate := []time.Time{
		now,                      // 当前月
		now.AddDate(0, 1, 0),      // 下个月
		now.AddDate(0, 2, 0),      // 再下个月
	}

	for _, month := range monthsToCreate {
		year := month.Year()
		monthNum := int(month.Month())
		partitionName := fmt.Sprintf("sys_device_mac_history_%d_%02d", year, monthNum)

		// 计算分区范围
		startDate := fmt.Sprintf("%d-%02d-01", year, monthNum)
		endMonth := monthNum + 1
		endYear := year
		if endMonth > 12 {
			endMonth = 1
			endYear = year + 1
		}
		endDate := fmt.Sprintf("%d-%02d-01", endYear, endMonth)

		createPartitionSQL := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s
			PARTITION OF sys_device_mac_history
			FOR VALUES FROM ('%s') TO ('%s');
		`, partitionName, startDate, endDate)

		if err := db.Exec(createPartitionSQL).Error; err != nil {
			return fmt.Errorf("创建分区 %s 失败: %w", partitionName, err)
		}
		applogger.Infof("[迁移] 分区 %s 创建成功", partitionName)
	}

	// 3. 创建BRIN索引（RESEARCH.md Pattern 4）
	// pages_per_range = 128 适合时间序列数据
	// 注意：分区表不支持 CONCURRENTLY 创建索引
	createBRINIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_mac_history_first_seen_brin
	ON sys_device_mac_history USING BRIN (first_seen)
	WITH (pages_per_range = 128);
	`

	if err := db.Exec(createBRINIndexSQL).Error; err != nil {
		return fmt.Errorf("创建BRIN索引失败: %w", err)
	}
	applogger.Infof("[迁移] BRIN索引创建成功")

	// 4. 创建复合索引（用于设备+MAC查询）
	// 注意：分区表不支持 CONCURRENTLY 创建索引
	createCompositeIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_mac_history_device_mac
	ON sys_device_mac_history (device_id, mac_address);
	`

	if err := db.Exec(createCompositeIndexSQL).Error; err != nil {
		return fmt.Errorf("创建复合索引失败: %w", err)
	}
	applogger.Infof("[迁移] 复合索引创建成功")

	applogger.Infof("[迁移] MAC地址历史表创建完成")
	return nil
}

// DropMACHistoryTable 删除MAC地址历史表（用于回滚）
func DropMACHistoryTable(db *gorm.DB) error {
	if !isPostgreSQL(db) {
		return nil
	}

	applogger.Warnf("[迁移] 开始删除MAC地址历史表")

	// 删除主表（级联删除所有分区）
	dropTableSQL := `DROP TABLE IF NOT EXISTS sys_device_mac_history CASCADE;`

	if err := db.Exec(dropTableSQL).Error; err != nil {
		return fmt.Errorf("删除MAC历史表失败: %w", err)
	}

	applogger.Warnf("[迁移] MAC地址历史表已删除")
	return nil
}

// CreateMonthlyPartition 创建单个月度分区（用于定时任务）
func CreateMonthlyPartition(db *gorm.DB, year int, month int) error {
	if !isPostgreSQL(db) {
		return nil
	}

	partitionName := fmt.Sprintf("sys_device_mac_history_%d_%02d", year, month)
	startDate := fmt.Sprintf("%d-%02d-01", year, month)

	// 计算结束日期
	endMonth := month + 1
	endYear := year
	if endMonth > 12 {
		endMonth = 1
		endYear = year + 1
	}
	endDate := fmt.Sprintf("%d-%02d-01", endYear, endMonth)

	createPartitionSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		PARTITION OF sys_device_mac_history
		FOR VALUES FROM ('%s') TO ('%s');
	`, partitionName, startDate, endDate)

	if err := db.Exec(createPartitionSQL).Error; err != nil {
		return fmt.Errorf("创建分区 %s 失败: %w", partitionName, err)
	}

	applogger.Infof("[迁移] 分区 %s 创建成功", partitionName)
	return nil
}

// DropOldPartition 删除过期的月度分区（用于定时清理）
func DropOldPartition(db *gorm.DB, year int, month int) error {
	if !isPostgreSQL(db) {
		return nil
	}

	partitionName := fmt.Sprintf("sys_device_mac_history_%d_%02d", year, month)

	dropPartitionSQL := fmt.Sprintf(`DROP TABLE IF EXISTS %s CASCADE;`, partitionName)

	if err := db.Exec(dropPartitionSQL).Error; err != nil {
		return fmt.Errorf("删除分区 %s 失败: %w", partitionName, err)
	}

	applogger.Infof("[迁移] 分区 %s 已删除", partitionName)
	return nil
}
