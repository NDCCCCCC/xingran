package operations

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestAssetService_Statistics 验证资产统计:
// 按 status(0=正常 1=停用) + nbf_status(0=否 1=拟报废,独立维度) 聚合 + 排除软删除。
// 替代前端「total*0.8/0.15/0.05」伪造比例占位实现。
func TestAssetService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE ops_asset (
			id TEXT PRIMARY KEY,
			devicesn TEXT,
			component_type TEXT,
			status INTEGER DEFAULT 0,
			nbf_status INTEGER DEFAULT 0,
			deleted_at DATETIME,
			created_at DATETIME
		)
	`).Error)

	now := time.Now().Format("2006-01-02 15:04:05")
	insert := func(prefix string, n, status, nbf int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(
				`INSERT INTO ops_asset (id, devicesn, status, nbf_status, created_at) VALUES (?, ?, ?, ?, ?)`,
				fmt.Sprintf("%s-%d", prefix, i), fmt.Sprintf("sn-%s-%d", prefix, i), status, nbf, now,
			).Error)
		}
	}
	// 70 正常 + 20 停用 + 10 正常但拟报废 + 5 停用且拟报废 = 105
	insert("n", 70, 0, 0)
	insert("s", 20, 1, 0)
	insert("nb1", 10, 0, 1)
	insert("nb2", 5, 1, 1)
	// 8 条已软删除(不计入)
	for i := 0; i < 8; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO ops_asset (id, devicesn, status, nbf_status, created_at, deleted_at) VALUES (?, ?, 0, 0, ?, ?)`,
			fmt.Sprintf("d-%d", i), fmt.Sprintf("sn-d-%d", i), now, now,
		).Error)
	}

	svc := &assetService{db: db}
	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(105), result.Total, "排除 8 条软删除")
	require.Equal(t, int64(80), result.Normal, "status=0: 70+10")
	require.Equal(t, int64(25), result.Stopped, "status=1: 20+5")
	require.Equal(t, int64(15), result.NBF, "nbf_status=1: 10+5(独立维度)")
}
