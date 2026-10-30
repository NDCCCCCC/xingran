package migrations

import (
	"fmt"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate214CreateADLoginHistory 创建 AD 域控设备登录历史表
// (quick 261016-ad-login-history: sys_ad_computer.last_logon 每次同步被
// upsert 覆盖导致登录历史永久丢失,本表按月分区记录 lastLogon 变化事件)。
//
// 设计要点:
//   - 仅 PG 分支执行(isPostgreSQL 守卫);sqlite 不建表、不注册,writer/query
//     侧由 addomain 包 isPostgres 守卫跳过。
//   - 分区表必须手写 SQL(GORM AutoMigrate 不支持 PARTITION BY);对应模型
//     models.ADLoginHistory 禁止加入 database.go AutoMigrate 列表。
//   - 分区表不建 PRIMARY KEY(分区表的唯一约束必须包含分区列,仿 mac_history 注释理由)。
//   - 时间列全部 TIMESTAMPTZ(新表从 schema 层根治时区,模型不写 AfterFind)。
//   - 全部幂等(IF NOT EXISTS)。
func Migrate214CreateADLoginHistory(db *gorm.DB) error {
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] AD登录历史表跳过创建(非PostgreSQL数据库)")
		return nil
	}

	applogger.Infof("[迁移] 开始创建AD登录历史表(月分区表)")

	// 1. 创建主表(声明为 PARTITION BY RANGE,不建 PRIMARY KEY)
	createMainTableSQL := `
	CREATE TABLE IF NOT EXISTS sys_ad_login_history (
		id UUID NOT NULL DEFAULT gen_random_uuid(),
		computer_id UUID NOT NULL,
		computer_name_snapshot VARCHAR(255) NOT NULL,
		ad_config_id UUID NOT NULL,
		login_time TIMESTAMPTZ NOT NULL,
		detected_at TIMESTAMPTZ NOT NULL,
		login_user VARCHAR(255),
		ip_address VARCHAR(50),
		event_type VARCHAR(20) NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	) PARTITION BY RANGE (login_time);
	`
	if err := db.Exec(createMainTableSQL).Error; err != nil {
		return fmt.Errorf("创建AD登录历史主表失败: %w", err)
	}
	applogger.Infof("[迁移] AD登录历史主表创建成功")

	// 2. 创建当前月 + 未来 2 个月的分区(边界计算仿 migration_mac_history.go)
	now := time.Now()
	monthsToCreate := []time.Time{
		now,                 // 当前月
		now.AddDate(0, 1, 0), // 下个月
		now.AddDate(0, 2, 0), // 再下个月
	}

	for _, month := range monthsToCreate {
		year := month.Year()
		monthNum := int(month.Month())
		partitionName := fmt.Sprintf("sys_ad_login_history_%d_%02d", year, monthNum)

		// 计算分区范围 [当月1号, 次月1号)
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
			PARTITION OF sys_ad_login_history
			FOR VALUES FROM ('%s') TO ('%s');
		`, partitionName, startDate, endDate)

		if err := db.Exec(createPartitionSQL).Error; err != nil {
			return fmt.Errorf("创建分区 %s 失败: %w", partitionName, err)
		}
		applogger.Infof("[迁移] 分区 %s 创建成功", partitionName)
	}

	// 3. BRIN 索引(时间序列数据,pages_per_range = 128,仿 mac_history)
	// 注意: 分区表不支持 CONCURRENTLY 创建索引
	createBRINIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_ad_login_history_login_time_brin
	ON sys_ad_login_history USING BRIN (login_time)
	WITH (pages_per_range = 128);
	`
	if err := db.Exec(createBRINIndexSQL).Error; err != nil {
		return fmt.Errorf("创建AD登录历史BRIN索引失败: %w", err)
	}
	applogger.Infof("[迁移] AD登录历史BRIN索引创建成功")

	// 4. 复合索引(按设备查询历史,查询端固定 ORDER BY login_time DESC)
	createCompositeIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_ad_login_history_computer_time
	ON sys_ad_login_history (computer_id, login_time);
	`
	if err := db.Exec(createCompositeIndexSQL).Error; err != nil {
		return fmt.Errorf("创建AD登录历史复合索引失败: %w", err)
	}
	applogger.Infof("[迁移] AD登录历史复合索引创建成功")

	applogger.Infof("[迁移] AD登录历史表创建完成")
	return nil
}
