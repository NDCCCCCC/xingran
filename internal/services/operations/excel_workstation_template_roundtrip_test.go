package operations

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xuri/excelize/v2"
)

// xlsxToFileHeader 把 excelize 生成的 workbook 包装成 multipart.FileHeader。
func xlsxToFileHeader(t *testing.T, f *excelize.File, filename string) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(buf.Bytes())
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, req.ParseMultipartForm(10<<20))
	return req.MultipartForm.File["file"][0]
}

// TestWorkstationImport_TemplateRoundTrip 复现用户场景：
// 用户下载工位模板 → 在其中填写数据（含主设备序列号）→ 上传，导入却报「所属楼宇不存在」。
//
// 该测试按真实模板逐场景验证：
//   场景 A：模板原样上传（示例行未删）
//   场景 B：示例行被用户数据覆盖（只填数据不删行）
//   场景 C：示例行保留 + 用户数据追加在下一行
//   场景 D：删除示例行，只保留用户数据（含主设备序列号）
func TestWorkstationImport_TemplateRoundTrip(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b1', '真实大楼', 'D1', 2)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, floor_no, building_id) VALUES ('f1', '1楼', '1', 'b1')`).Error)
	// 工位落库需要 sys_workstation 表 + 唯一索引（对齐 PostgreSQL 生产约束）
	require.NoError(t, db.AutoMigrate(&sysmodels.Workstation{}))
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_ws_floor_name ON sys_workstation(floor_id, workstation_name)`).Error)

	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	// 生成真实模板（与前端 GET /ops/workstation/template 走同一条代码路径）
	tmpl, err := svc.GenerateTemplate("workstation")
	require.NoError(t, err)

	// 打印模板内容，确认表头与示例行的位置
	rows, err := tmpl.GetRows("工位列表")
	require.NoError(t, err)
	for i, r := range rows {
		t.Logf("模板第 %d 行: %#v", i+1, r)
	}

	// 找到「主设备序列号」列在模板中的位置（用户填的那一列）
	serialCol := -1
	for ci, h := range rows[0] {
		if h == "主设备序列号" {
			serialCol = ci
		}
	}
	require.NotEqual(t, -1, serialCol, "模板应含主设备序列号列")

	// 用户填的真实数据行
	userRow := make([]string, len(rows[0]))
	copy(userRow, rows[0])
	userRow[0] = "W-001"          // 工位名称
	userRow[1] = "真实大楼"        // 所属楼宇
	userRow[2] = "1楼"            // 所属楼层名称
	userRow[serialCol] = "SN-ABC-001" // 主设备序列号

	t.Run("场景A_模板原样上传_示例行未删", func(t *testing.T) {
		f, err := svc.GenerateTemplate("workstation")
		require.NoError(t, err)
		res, err := svc.ImportData(ctx, "workstation", xlsxToFileHeader(t, f, "a.xlsx"), "u1")
		require.NoError(t, err)
		t.Logf("场景A 结果: inserted=%d updated=%d failed=%d errors=%+v",
			res.Inserted, res.Updated, res.Failed, res.Errors)
		for _, e := range res.Errors {
			t.Logf("  场景A 错误: row=%d field=%s value=%q err=%s", e.Row, e.Field, e.Value, e.Error)
		}
	})

	t.Run("场景B_示例行被用户数据覆盖", func(t *testing.T) {
		f, err := svc.GenerateTemplate("workstation")
		require.NoError(t, err)
		for ci, v := range userRow {
			cell, _ := excelize.CoordinatesToCellName(ci+1, 2)
			require.NoError(t, f.SetCellValue("工位列表", cell, v))
		}
		res, err := svc.ImportData(ctx, "workstation", xlsxToFileHeader(t, f, "b.xlsx"), "u1")
		require.NoError(t, err)
		t.Logf("场景B 结果: inserted=%d updated=%d failed=%d errors=%+v",
			res.Inserted, res.Updated, res.Failed, res.Errors)
		for _, e := range res.Errors {
			t.Logf("  场景B 错误: row=%d field=%s value=%q err=%s", e.Row, e.Field, e.Value, e.Error)
		}
	})

	t.Run("场景C_示例行保留_用户数据追加下一行", func(t *testing.T) {
		f, err := svc.GenerateTemplate("workstation")
		require.NoError(t, err)
		for ci, v := range userRow {
			cell, _ := excelize.CoordinatesToCellName(ci+1, 3)
			require.NoError(t, f.SetCellValue("工位列表", cell, v))
		}
		res, err := svc.ImportData(ctx, "workstation", xlsxToFileHeader(t, f, "c.xlsx"), "u1")
		require.NoError(t, err)
		t.Logf("场景C 结果: inserted=%d updated=%d failed=%d errors=%+v",
			res.Inserted, res.Updated, res.Failed, res.Errors)
		for _, e := range res.Errors {
			t.Logf("  场景C 错误: row=%d field=%s value=%q err=%s", e.Row, e.Field, e.Value, e.Error)
		}
	})

	t.Run("场景D_删除示例行_仅保留用户数据", func(t *testing.T) {
		f, err := svc.GenerateTemplate("workstation")
		require.NoError(t, err)
		// 删除示例行（第2行）
		require.NoError(t, f.RemoveRow("工位列表", 2))
		for ci, v := range userRow {
			cell, _ := excelize.CoordinatesToCellName(ci+1, 2)
			require.NoError(t, f.SetCellValue("工位列表", cell, v))
		}
		res, err := svc.ImportData(ctx, "workstation", xlsxToFileHeader(t, f, "d.xlsx"), "u1")
		require.NoError(t, err)
		t.Logf("场景D 结果: inserted=%d updated=%d failed=%d errors=%+v",
			res.Inserted, res.Updated, res.Failed, res.Errors)
		for _, e := range res.Errors {
			t.Logf("  场景D 错误: row=%d field=%s value=%q err=%s", e.Row, e.Field, e.Value, e.Error)
		}
		assert.Equal(t, 0, res.Failed, "删除示例行后不应失败")
	})
}
