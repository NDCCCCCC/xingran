package operations

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
)

// setupDeptMappingTestDB 建一张与 sys_dept 兼容的最简表供测试
func setupDeptMappingTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "open sqlite")

	require.NoError(t, gormDB.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_code TEXT,
			dept_name TEXT NOT NULL,
			parent_id TEXT,
			ancestors TEXT,
			status INTEGER DEFAULT 0,
			deleted_at DATETIME
		)
	`).Error)

	return gormDB
}

func newTestCoreWithDB(gormDB *gorm.DB) *core.Core {
	return &core.Core{
		CoreInfra: &core.CoreInfra{
			DB: &db.Database{DB: gormDB, Type: "sqlite"},
		},
	}
}

func TestDownloadDeptMappingTemplate_ParentPathAndCols(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB := setupDeptMappingTestDB(t)

	// 插入测试数据: 3 级部门与同名部门
	// 根: root -> 公司
	// 2级: sub1 -> 湖北省分公司, sub2 -> 湖南省分公司
	// 3级同名: sub1下的营销部 (code: HB_YXB), sub2下的营销部 (code: HN_YXB)
	// 停用部门: (code: STOPPED, status: 1)
	// 软删部门: (code: DELETED, deleted_at 非空)
	require.NoError(t, gormDB.Exec(`
		INSERT INTO sys_dept (id, dept_code, dept_name, parent_id, status)
		VALUES ('root', '001', '总公司', NULL, 0)
	`).Error)
	require.NoError(t, gormDB.Exec(`
		INSERT INTO sys_dept (id, dept_code, dept_name, parent_id, status)
		VALUES ('sub1', 'HB', '湖北分公司', 'root', 0)
	`).Error)
	require.NoError(t, gormDB.Exec(`
		INSERT INTO sys_dept (id, dept_code, dept_name, parent_id, status)
		VALUES ('sub2', 'HN', '湖南分公司', 'root', 0)
	`).Error)
	require.NoError(t, gormDB.Exec(`
		INSERT INTO sys_dept (id, dept_code, dept_name, parent_id, status)
		VALUES ('d1', 'HB_YXB', '个人营销部', 'sub1', 0)
	`).Error)
	require.NoError(t, gormDB.Exec(`
		INSERT INTO sys_dept (id, dept_code, dept_name, parent_id, status)
		VALUES ('d2', 'HN_YXB', '个人营销部', 'sub2', 0)
	`).Error)
	// 停用
	require.NoError(t, gormDB.Exec(`
		INSERT INTO sys_dept (id, dept_code, dept_name, parent_id, status)
		VALUES ('stopped', 'STOP', '停用部门', 'root', 1)
	`).Error)
	// 软删除
	require.NoError(t, gormDB.Exec(`
		INSERT INTO sys_dept (id, dept_code, dept_name, parent_id, status, deleted_at)
		VALUES ('deleted', 'DEL', '已删部门', 'root', 0, '2026-01-01 00:00:00')
	`).Error)

	testCore := newTestCore(t)
	testCore.DB = &db.Database{DB: gormDB, Type: "sqlite"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ops/workstation/dept-mapping-template", nil)

	handler := DownloadDeptMappingTemplate(testCore)
	handler(c)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment; filename=dept_mapping_")

	// 读取 Excel 内容并校验
	excelFile, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
	require.NoError(t, err, "解析导出的 excel 文件")
	defer func() { _ = excelFile.Close() }()

	sheetName := "部门映射"
	rows, err := excelFile.GetRows(sheetName)
	require.NoError(t, err)

	// 必须有表头行 + 5个正常启用的部门数据行 = 6行
	require.Len(t, rows, 6)

	// 校验表头
	assert.Equal(t, []string{"dept_name", "parent_path", "dept_code"}, rows[0])

	// 校验数据行按 dept_code ASC 排序
	// 数据排序:
	// 1: '001' -> 总公司, parent_path: 总公司
	// 2: 'HB' -> 湖北分公司, parent_path: 总公司 → 湖北分公司
	// 3: 'HB_YXB' -> 个人营销部, parent_path: 总公司 → 湖北分公司 → 个人营销部
	// 4: 'HN' -> 湖南分公司, parent_path: 总公司 → 湖南分公司
	// 5: 'HN_YXB' -> 个人营销部, parent_path: 总公司 → 湖南分公司 → 个人营销部

	assert.Equal(t, []string{"总公司", "总公司", "001"}, rows[1])
	assert.Equal(t, []string{"湖北分公司", "总公司 → 湖北分公司", "HB"}, rows[2])
	assert.Equal(t, []string{"个人营销部", "总公司 → 湖北分公司 → 个人营销部", "HB_YXB"}, rows[3])
	assert.Equal(t, []string{"湖南分公司", "总公司 → 湖南分公司", "HN"}, rows[4])
	assert.Equal(t, []string{"个人营销部", "总公司 → 湖南分公司 → 个人营销部", "HN_YXB"}, rows[5])
}
