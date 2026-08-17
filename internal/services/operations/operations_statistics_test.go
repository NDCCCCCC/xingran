package operations

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// 以下 7 个测试验证 operations 模块统计端点:按 status 聚合 + 排除软删除,
// 专用 COUNT 端点替代前端 statisticsHelper「分页拉全量再 .length」。
// 各 model status 枚举与前端 utils/statisticsHelper.ts 核对一致。

func nowStr() string { return time.Now().Format("2006-01-02 15:04:05") }

// TestBuildingService_Statistics 楼宇(status 0=正常 1=停用)。
func TestBuildingService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE ops_buildings (id TEXT PRIMARY KEY, name TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME, created_at DATETIME)`).Error)
	now := nowStr()
	insert := func(p string, n, st int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, status, created_at) VALUES (?, ?, ?, ?)`, fmt.Sprintf("%s-%d", p, i), p, st, now).Error)
		}
	}
	insert("a", 60, 0)
	insert("i", 30, 1)
	for i := 0; i < 10; i++ {
		require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, status, created_at, deleted_at) VALUES (?, 'd', 0, ?, ?)`, fmt.Sprintf("d-%d", i), now, now).Error)
	}
	r, err := (&buildingService{db: db}).Statistics(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, int64(90), r.Total)
	require.Equal(t, int64(60), r.Active)
	require.Equal(t, int64(30), r.Inactive)
}

// TestFloorService_Statistics 楼层(status 0/1)。
func TestFloorService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE ops_floors (id TEXT PRIMARY KEY, name TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME, created_at DATETIME)`).Error)
	now := nowStr()
	insert := func(p string, n, st int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, status, created_at) VALUES (?, ?, ?, ?)`, fmt.Sprintf("%s-%d", p, i), p, st, now).Error)
		}
	}
	insert("a", 70, 0)
	insert("i", 20, 1)
	for i := 0; i < 5; i++ {
		require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, status, created_at, deleted_at) VALUES (?, 'd', 0, ?, ?)`, fmt.Sprintf("d-%d", i), now, now).Error)
	}
	r, err := (&floorService{db: db}).Statistics(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(90), r.Total)
	require.Equal(t, int64(70), r.Active)
	require.Equal(t, int64(20), r.Inactive)
}

// TestServerRoomService_Statistics 机房(status 0/1)。
func TestServerRoomService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE ops_server_rooms (id TEXT PRIMARY KEY, name TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME, created_at DATETIME)`).Error)
	now := nowStr()
	insert := func(p string, n, st int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(`INSERT INTO ops_server_rooms (id, name, status, created_at) VALUES (?, ?, ?, ?)`, fmt.Sprintf("%s-%d", p, i), p, st, now).Error)
		}
	}
	insert("a", 40, 0)
	insert("i", 25, 1)
	for i := 0; i < 8; i++ {
		require.NoError(t, db.Exec(`INSERT INTO ops_server_rooms (id, name, status, created_at, deleted_at) VALUES (?, 'd', 0, ?, ?)`, fmt.Sprintf("d-%d", i), now, now).Error)
	}
	r, err := (&serverRoomService{db: db}).Statistics(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(65), r.Total)
	require.Equal(t, int64(40), r.Active)
	require.Equal(t, int64(25), r.Inactive)
}

// TestInfoPointService_Statistics 信息点(status 0=正常 1=故障 2=停用)。
func TestInfoPointService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE ops_info_points (id TEXT PRIMARY KEY, name TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME, created_at DATETIME)`).Error)
	now := nowStr()
	insert := func(p string, n, st int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(`INSERT INTO ops_info_points (id, name, status, created_at) VALUES (?, ?, ?, ?)`, fmt.Sprintf("%s-%d", p, i), p, st, now).Error)
		}
	}
	insert("n", 50, 0)
	insert("f", 20, 1)
	insert("d", 15, 2)
	for i := 0; i < 7; i++ {
		require.NoError(t, db.Exec(`INSERT INTO ops_info_points (id, name, status, created_at, deleted_at) VALUES (?, 'dd', 0, ?, ?)`, fmt.Sprintf("dd-%d", i), now, now).Error)
	}
	r, err := (&infoPointService{db: db}).Statistics(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(85), r.Total)
	require.Equal(t, int64(50), r.Normal)
	require.Equal(t, int64(20), r.Fault)
	require.Equal(t, int64(15), r.Disabled)
}

// TestDedicatedLineService_Statistics 专线(status 0=正常 1=故障 2=停用)。
func TestDedicatedLineService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE ops_dedicated_lines (id TEXT PRIMARY KEY, name TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME, created_at DATETIME)`).Error)
	now := nowStr()
	insert := func(p string, n, st int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(`INSERT INTO ops_dedicated_lines (id, name, status, created_at) VALUES (?, ?, ?, ?)`, fmt.Sprintf("%s-%d", p, i), p, st, now).Error)
		}
	}
	insert("n", 45, 0)
	insert("f", 18, 1)
	insert("d", 12, 2)
	for i := 0; i < 6; i++ {
		require.NoError(t, db.Exec(`INSERT INTO ops_dedicated_lines (id, name, status, created_at, deleted_at) VALUES (?, 'dd', 0, ?, ?)`, fmt.Sprintf("dd-%d", i), now, now).Error)
	}
	r, err := (&dedicatedLineService{db: db}).Statistics(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(75), r.Total)
	require.Equal(t, int64(45), r.Normal)
	require.Equal(t, int64(18), r.Fault)
	require.Equal(t, int64(12), r.Disabled)
}

// TestRoomDeviceService_Statistics 机房设备(status 0=正常 1=故障 2=报废)。
func TestRoomDeviceService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE ops_room_devices (id TEXT PRIMARY KEY, name TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME, created_at DATETIME)`).Error)
	now := nowStr()
	insert := func(p string, n, st int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(`INSERT INTO ops_room_devices (id, name, status, created_at) VALUES (?, ?, ?, ?)`, fmt.Sprintf("%s-%d", p, i), p, st, now).Error)
		}
	}
	insert("n", 55, 0)
	insert("f", 22, 1)
	insert("s", 8, 2)
	for i := 0; i < 5; i++ {
		require.NoError(t, db.Exec(`INSERT INTO ops_room_devices (id, name, status, created_at, deleted_at) VALUES (?, 'dd', 0, ?, ?)`, fmt.Sprintf("dd-%d", i), now, now).Error)
	}
	r, err := (&roomDeviceService{db: db}).Statistics(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(85), r.Total)
	require.Equal(t, int64(55), r.Normal)
	require.Equal(t, int64(22), r.Fault)
	require.Equal(t, int64(8), r.Scrapped)
}

// TestWorkstationService_Statistics 工位(status 0=可用 1=占用 2=维护)。
func TestWorkstationService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE sys_workstation (id TEXT PRIMARY KEY, workstation_name TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME, created_at DATETIME)`).Error)
	now := nowStr()
	insert := func(p string, n, st int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(`INSERT INTO sys_workstation (id, workstation_name, status, created_at) VALUES (?, ?, ?, ?)`, fmt.Sprintf("%s-%d", p, i), p, st, now).Error)
		}
	}
	insert("a", 60, 0)
	insert("o", 25, 1)
	insert("m", 10, 2)
	for i := 0; i < 5; i++ {
		require.NoError(t, db.Exec(`INSERT INTO sys_workstation (id, workstation_name, status, created_at, deleted_at) VALUES (?, 'dd', 0, ?, ?)`, fmt.Sprintf("dd-%d", i), now, now).Error)
	}
	r, err := (&workstationService{db: db}).Statistics(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, int64(95), r.Total)
	require.Equal(t, int64(60), r.Available)
	require.Equal(t, int64(25), r.Occupied)
	require.Equal(t, int64(10), r.Maintain)
}
