//go:build archive_skip


package migrations

import (
	"fmt"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate152MACMatView 创建 4 个 MAC 历史物化视图及其 UNIQUE 索引
//
// Phase 15 PERF-02 (D-09/D-10 锁定):
//   - 4 个物化视图全部建在分区表 sys_device_mac_history 上
//   - 每个视图都建 UNIQUE 索引 (CONCURRENTLY 刷新前置条件)
//   - 仅在 PostgreSQL 中执行 (SQLite 跳过)
//
// 视图清单:
//   - MV-01 mv_mac_port_latest          : 每端口最新 MAC 状态
//   - MV-02 mv_mac_device_summary       : 设备 MAC 汇总
//   - MV-03 mv_mac_long_occupancy_top   : 长期占用 Top-50
//   - MV-04 mv_mac_port_daily_count     : 每日端口使用计次 (热力图数据源)
func Migrate152MACMatView(db *gorm.DB) error {
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 物化视图跳过创建（非PostgreSQL数据库）")
		return nil
	}

	applogger.Infof("[迁移] 开始创建 4 个 MAC 历史物化视图")

	type viewSpec struct {
		name        string
		matViewSQL  string
		uniqueIndex string
	}

	views := []viewSpec{
		{
			name: "mv_mac_port_latest",
			matViewSQL: `
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_mac_port_latest AS
SELECT
    h.device_id,
    h.interface_name,
    h.device_name_snapshot,
    (SELECT h2.mac_address
       FROM sys_device_mac_history h2
      WHERE h2.device_id = h.device_id
        AND h2.interface_name = h.interface_name
      ORDER BY h2.last_seen DESC
      LIMIT 1)                                              AS mac_address,
    MAX(h.last_seen)                                        AS last_seen,
    (SELECT h3.event_type
       FROM sys_device_mac_history h3
      WHERE h3.device_id = h.device_id
        AND h3.interface_name = h.interface_name
      ORDER BY h3.last_seen DESC
      LIMIT 1)                                              AS event_type
FROM sys_device_mac_history h
GROUP BY h.device_id, h.interface_name, h.device_name_snapshot;
`,
			uniqueIndex: `
CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_port_latest_pk
ON mv_mac_port_latest (device_id, interface_name);
`,
		},
		{
			name: "mv_mac_device_summary",
			matViewSQL: `
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_mac_device_summary AS
SELECT
    device_id,
    MAX(device_name_snapshot)                              AS device_name_snapshot,
    COUNT(DISTINCT mac_address)                            AS mac_count,
    COUNT(DISTINCT CASE WHEN event_type = 'appeared' THEN mac_address END) AS active_count,
    MAX(last_seen)                                         AS last_update
FROM sys_device_mac_history
GROUP BY device_id;
`,
			uniqueIndex: `
CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_device_summary_pk
ON mv_mac_device_summary (device_id);
`,
		},
		{
			name: "mv_mac_long_occupancy_top",
			matViewSQL: `
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_mac_long_occupancy_top AS
SELECT
    mac_address,
    SUM(EXTRACT(EPOCH FROM (last_seen - first_seen)))      AS total_duration,
    (SELECT h2.interface_name
       FROM sys_device_mac_history h2
      WHERE h2.mac_address = h.mac_address
      ORDER BY h2.last_seen DESC
      LIMIT 1)                                              AS last_port,
    MAX(last_seen)                                         AS snapshot_at
FROM sys_device_mac_history h
WHERE last_seen - first_seen >= INTERVAL '24 hours'
GROUP BY mac_address
ORDER BY total_duration DESC
LIMIT 50;
`,
			uniqueIndex: `
CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_long_occupancy_pk
ON mv_mac_long_occupancy_top (mac_address, last_port);
`,
		},
		{
			name: "mv_mac_port_daily_count",
			matViewSQL: `
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_mac_port_daily_count AS
SELECT
    device_id,
    MAX(device_name_snapshot)                              AS device_name_snapshot,
    interface_name,
    DATE(first_seen)                                       AS date,
    COUNT(*)                                               AS change_count
FROM sys_device_mac_history
GROUP BY device_id, interface_name, DATE(first_seen);
`,
			uniqueIndex: `
CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_port_daily_pk
ON mv_mac_port_daily_count (device_id, interface_name, date);
`,
		},
	}

	for _, v := range views {
		applogger.Infof("[迁移] 创建物化视图 %s", v.name)
		if err := db.Exec(v.matViewSQL).Error; err != nil {
			return fmt.Errorf("创建物化视图 %s 失败: %w", v.name, err)
		}
		if err := db.Exec(v.uniqueIndex).Error; err != nil {
			return fmt.Errorf("创建物化视图 %s UNIQUE 索引失败: %w", v.name, err)
		}
		applogger.Infof("[迁移] 物化视图 %s + UNIQUE 索引 已创建", v.name)
	}

	applogger.Infof("[迁移] 4 个 MAC 历史物化视图全部创建完成")
	return nil
}
