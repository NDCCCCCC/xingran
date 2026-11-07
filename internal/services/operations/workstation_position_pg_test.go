package operations

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// =====================================================================
// BatchUpdatePositions 的 PostgreSQL 端到端回归测试
//
// 背景（workstation-positions-500 会话）：
// `CASE id WHEN ? THEN ? END` 在 PostgreSQL 上报
//   ERROR: column "position_x" is of type integer but expression is of type text
//   (SQLSTATE 42804)
// 根因是 THEN 分支占位符在 Parse 阶段无法推断类型，PG 把整个 CASE 退化为 text。
// 该缺陷在 SQLite 上无法复现（动态类型），只有真实 PostgreSQL 才能暴露。
//
// 本测试在真实 PG 上、事务内执行（结束 ROLLBACK），不会留下任何数据变更。
//
// 连接信息来源（按优先级）：
//  1. TEST_PG_DSN 环境变量（CI 推荐；v1.52 P1-9 与 pg_contract_test.go 统一）
//  2. 项目根 .env 中的 DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME（本地开发）
//  3. 以上都没有 → 跳过（skip），保证无 PG 环境的 CI 依然可跑
// =====================================================================

// workstationPositionProjectRoot 定位项目根目录（用于读取 .env）。
func workstationPositionProjectRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

// workstationPositionLoadDotEnv 把项目根 .env 的键注入环境变量（只读入内存，不打印值）。
func workstationPositionLoadDotEnv(t *testing.T) {
	t.Helper()
	f, err := os.Open(filepath.Join(workstationPositionProjectRoot(), ".env"))
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if k == "" {
			continue
		}
		if _, exists := os.LookupEnv(k); !exists {
			_ = os.Setenv(k, v)
		}
	}
}

// workstationPositionDSN 解析测试用 PostgreSQL DSN；无连接信息时 skip。
// v1.52 P1-9：门控变量与 pg_contract_test.go 统一为 TEST_PG_DSN；
// 保留既有 .env / DB_* fallback 与 skip 语义（不改测试行为）。
func workstationPositionDSN(t *testing.T) string {
	t.Helper()
	if dsn := strings.TrimSpace(os.Getenv("TEST_PG_DSN")); dsn != "" {
		return dsn
	}
	workstationPositionLoadDotEnv(t)

	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	name := os.Getenv("DB_NAME")
	if host == "" || user == "" || name == "" {
		t.Skip("未提供 PostgreSQL 连接信息（TEST_PG_DSN 或 DB_* 环境变量），跳过 PG 集成测试")
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, os.Getenv("DB_PASSWORD"), name, sslMode)
}

// TestBatchUpdatePositions_Postgres 在真实 PostgreSQL 上验证批量位置更新不再报 42804。
func TestBatchUpdatePositions_Postgres(t *testing.T) {
	dsn := workstationPositionDSN(t)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "连接 PostgreSQL 失败")

	ctx := context.Background()

	tx := db.WithContext(ctx).Begin()
	require.NoError(t, tx.Error, "开启事务失败")
	defer func() { _ = tx.Rollback().Error }()

	// 取两条真实工位 id（软删除过滤与 GORM 语义一致）
	type idRow struct {
		ID string `gorm:"column:id"`
	}
	var rows []idRow
	require.NoError(t,
		tx.Raw(`SELECT id FROM sys_workstation WHERE deleted_at IS NULL LIMIT 2`).Scan(&rows).Error,
		"读取工位 id 失败")
	if len(rows) < 2 {
		t.Skip("sys_workstation 不足 2 行可用数据，跳过 PG 集成测试")
	}

	svc := NewWorkstationService(tx)

	// 场景 A：仅必填字段（position_x / position_y）
	itemsA := []PositionUpdateItem{
		{ID: rows[0].ID, PositionX: 100, PositionY: 200},
		{ID: rows[1].ID, PositionX: 300, PositionY: 400},
	}
	require.NoError(t, svc.BatchUpdatePositions(ctx, itemsA),
		"仅必填字段的批量更新在 PostgreSQL 上不应报 42804")

	// 场景 B：带可选字段（rotation / width / depth / deskType）
	rot, w, d, dt := 45, 1800, 800, 1
	itemsB := []PositionUpdateItem{
		{ID: rows[0].ID, PositionX: 1000, PositionY: 2000,
			Rotation: &rot, Width: &w, Depth: &d, DeskType: &dt},
		{ID: rows[1].ID, PositionX: 3000, PositionY: 4000,
			Rotation: &rot, Width: &w, Depth: &d, DeskType: &dt},
	}
	require.NoError(t, svc.BatchUpdatePositions(ctx, itemsB),
		"带可选字段的批量更新在 PostgreSQL 上不应报 42804")

	// 断言坐标确实落库
	var ws struct {
		PositionX int `gorm:"column:position_x"`
		PositionY int `gorm:"column:position_y"`
	}
	require.NoError(t, tx.Raw(`SELECT position_x, position_y FROM sys_workstation WHERE id = ?`, rows[0].ID).Scan(&ws).Error)
	require.Equal(t, 1000, ws.PositionX)
	require.Equal(t, 2000, ws.PositionY)
}
