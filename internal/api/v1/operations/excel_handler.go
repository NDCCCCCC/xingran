package operations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"github.com/xuri/excelize/v2"
)

// excelEntityModuleNames maps each entityType to its Chinese module name for operlog.
// Used so audit log rows for Excel import/export carry the same module name as the
// underlying entity's own CRUD handlers (e.g. /ops/buildings/excel/import → 楼宇管理).
var excelEntityModuleNames = map[string]string{
	"building":      "楼宇管理",
	"floor":         "楼层管理",
	"workstation":   "工位管理",
	"serverRoom":    "机房管理",
	"roomDevice":    "机房设备",
	"dedicatedLine": "专线管理",
	"infoPoint":     "信息点管理",
	"asset":         "资产管理",
}

// excelModuleName returns the Chinese module name for entityType, defaulting to
// "Excel导入导出" if the entity is not in the known map (defensive).
func excelModuleName(entityType string) string {
	if name, ok := excelEntityModuleNames[entityType]; ok {
		return name
	}
	return "Excel导入导出"
}

// SetupExcelRouter 设置Excel导入导出路由
func SetupExcelRouter(r *gin.RouterGroup, entityType string, core *core.Core) {
	r.POST("/import", importData(entityType, core))
	r.POST("/export", exportData(entityType, core))
	r.GET("/template", downloadTemplate(entityType, core))
}

func getExcelService(core *core.Core) *opsServices.ExcelService {
	cacheProvider := system.NewCacheAdapter(core.Cache)
	geocodingService := opsServices.NewGeocodingService(core.Config.Baidu.MapAK)
	// 工位导入 post-import hook 需要按 deviceSerial 调 SetPrimaryAndSaveBySerial
	// 同步主设备;构造 ExcelService 时一并注入(可选,缺少时跳过不阻断)。
	deviceService := opsServices.NewWorkstationDeviceService(core.DB.GetDB())
	return opsServices.NewExcelService(core.DB.GetDB(), core.PwdManager, cacheProvider, geocodingService).
		WithDeviceService(deviceService)
}

func generateExcelFilename(entityType, fileSuffix string) string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s_%s_%s.xlsx", entityType, fileSuffix, timestamp)
}

func writeExcelResponse(c *gin.Context, file *excelize.File, filename string) error {
	var buf bytes.Buffer
	if _, err := file.WriteTo(&buf); err != nil {
		return fmt.Errorf("生成文件失败: %w", err)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
	return nil
}

// importData 导入数据
func importData(entityType string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		// 获取上传的文件
		file, err := c.FormFile("file")
		if err != nil {
			response.Error(c, response.ErrBadRequest, "未找到上传文件")
			return
		}

		// 验证文件类型(扩展名 + 大小上限 + 内容魔数三重校验)
		if !isValidExcelFile(file.Filename) {
			response.Error(c, response.ErrBadRequest, "只支持 .xlsx 格式的Excel文件。如需导入 .xls 文件，请先用 Excel 或 WPS 另存为 .xlsx 格式")
			return
		}
		// P1 fix: 防止超大文件触发 OOM / 内存泄漏
		if file.Size > maxExcelUploadSize {
			response.Error(c, response.ErrBadRequest, fmt.Sprintf("文件过大(%d 字节,上限 %d 字节)", file.Size, maxExcelUploadSize))
			return
		}
		// P1 fix: 仅按扩展名校验可被改后缀绕过 — 读取前 4 字节验证 OOXML/ZIP 魔数 (PK\x03\x04)
		if err := verifyExcelMagicBytes(file); err != nil {
			response.Error(c, response.ErrBadRequest, "文件内容非有效的 .xlsx 格式")
			return
		}

		// 调用服务导入数据
		excelService := getExcelService(core)
		result, err := excelService.ImportData(c.Request.Context(), entityType, file, userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// Phase 34 操作日志：只记录 filename + 行数统计（inserted/updated/failed），
		// 绝不记录原始 Excel 行数据（T-34-W3-01 信息泄露缓解）。OperType=Import(6)。
		operlog.Record(c, core.OperLogService, core.GetDB(), excelModuleName(entityType), operlog.OperTypeImport,
			operlog.WithOperParam(fmt.Sprintf(`{"filename":%q,"size":%d,"inserted":%d,"updated":%d,"failed":%d}`,
				file.Filename, file.Size, result.Inserted, result.Updated, result.Failed)))
		response.Success(c, gin.H{
			"inserted": result.Inserted,
			"updated":  result.Updated,
			"failed":   result.Failed,
			"errors":   result.Errors,
		})
	}
}

// exportData 导出数据
func exportData(entityType string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 支持动态参数接收，向后兼容旧格式
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		excelService := getExcelService(core)
		file, err := excelService.ExportData(c.Request.Context(), entityType, req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// Phase 34 操作日志：OperType=Export(5)。审计"谁在何时导出过哪份数据"。
		// 不记录导出的具体行（可能很大），只记录 module + 请求参数摘要。
		if paramBytes, err := json.Marshal(req); err == nil {
			operlog.Record(c, core.OperLogService, core.GetDB(), excelModuleName(entityType), operlog.OperTypeExport,
				operlog.WithOperParam(operlog.FilterSensitiveParams(string(paramBytes))))
		} else {
			operlog.Record(c, core.OperLogService, core.GetDB(), excelModuleName(entityType), operlog.OperTypeExport)
		}
		filename := generateExcelFilename(entityType, "export")
		if err := writeExcelResponse(c, file, filename); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
		}
	}
}

// downloadTemplate 下载模板
func downloadTemplate(entityType string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		excelService := getExcelService(core)
		file, err := excelService.GenerateTemplate(entityType)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// Phase 34 操作日志：OperType=Download(18)。模板下载虽是只读，但记录可审计
		// "谁在何时拉取过导入模板"（合规场景下有意义）。
		operlog.Record(c, core.OperLogService, core.GetDB(), excelModuleName(entityType), operlog.OperTypeDownload)
		filename := generateExcelFilename(entityType, "template")
		if err := writeExcelResponse(c, file, filename); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
		}
	}
}

// DownloadDeptMappingTemplate 下载部门名称↔代码映射表 (xlsx)
//
// quick 260713-df0 工位导入的辅助端点: 用户在 Excel 中填写 deptName 容易拼错,
// 提供一份 sys_dept 全量映射表(dept_name | dept_code 两列), 方便按代码填写或校对。
//
// 数据范围: WHERE deleted_at IS NULL AND status = 0(只含启用部门)。
// 排序: dept_code ASC, 方便用户按代码区间检索。
//
// 端点路径: GET /api/v1/ops/workstation/dept-mapping-template
// 继承 workstations 路由组的 ops:workstation:* 权限, 与 /import /template 保持一致。
// operlog: 记 1 条 部门管理 / OperTypeDownload 日志(独立于工位导入的 Import 日志)。
func DownloadDeptMappingTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 查询 sys_dept 中所有启用部门, 按 dept_code 升序
		type deptRow struct {
			DeptName string `gorm:"column:dept_name"`
			DeptCode string `gorm:"column:dept_code"`
		}
		var rows []deptRow
		if err := core.DB.GetDB().WithContext(c.Request.Context()).
			Table("sys_dept").
			Select("dept_name, dept_code").
			Where("deleted_at IS NULL AND status = 0").
			Order("dept_code ASC").
			Scan(&rows).Error; err != nil {
			response.Error(c, response.ErrServerError, fmt.Sprintf("查询部门映射失败: %v", err))
			return
		}

		// 创建 xlsx, sheet 名 部门映射, 表头 dept_name | dept_code
		f := excelize.NewFile()
		sheetName := "部门映射"
		index, err := f.NewSheet(sheetName)
		if err != nil {
			response.Error(c, response.ErrServerError, fmt.Sprintf("创建工作表失败: %v", err))
			return
		}
		f.SetActiveSheet(index)
		_ = f.DeleteSheet("Sheet1")

		headerStyle, err := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Size: 11},
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
			Border: []excelize.Border{
				{Type: "left", Color: "#000000", Style: 1},
				{Type: "top", Color: "#000000", Style: 1},
				{Type: "bottom", Color: "#000000", Style: 1},
				{Type: "right", Color: "#000000", Style: 1},
			},
		})
		if err != nil {
			response.Error(c, response.ErrServerError, fmt.Sprintf("创建表头样式失败: %v", err))
			return
		}

		// 表头
		if err := f.SetCellValue(sheetName, "A1", "dept_name"); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}
		if err := f.SetCellValue(sheetName, "B1", "dept_code"); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}
		_ = f.SetCellStyle(sheetName, "A1", "B1", headerStyle)

		// 数据行(从第 2 行开始)
		for i, row := range rows {
			cellName, _ := excelize.CoordinatesToCellName(1, i+2)
			_ = f.SetCellValue(sheetName, cellName, row.DeptName)
			cellCode, _ := excelize.CoordinatesToCellName(2, i+2)
			_ = f.SetCellValue(sheetName, cellCode, row.DeptCode)
		}

		// 列宽
		_ = f.SetColWidth(sheetName, "A", "A", 30)
		_ = f.SetColWidth(sheetName, "B", "B", 20)

		// 冻结表头
		_ = f.SetPanes(sheetName, &excelize.Panes{
			Freeze: true,
			YSplit: 1,
		})

		// 记录 operlog (部门管理 / Download)
		operlog.Record(c, core.OperLogService, core.GetDB(), "部门管理", operlog.OperTypeDownload)

		// 文件名带时间戳,便于区分多次下载
		filename := fmt.Sprintf("dept_mapping_%s.xlsx", time.Now().Format("20060102_150405"))
		if err := writeExcelResponse(c, f, filename); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
		}
	}
}

// BatchOperationRequest 批量操作请求
type BatchOperationRequest struct {
	Action string                   `json:"action" binding:"required"` // create, update, delete
	IDs    []string                 `json:"ids"`
	Data   []map[string]interface{} `json:"data"`
}

// BatchOperation 批量操作
func BatchOperation(entityType string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BatchOperationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		ctx := c.Request.Context()
		var result interface{}
		var err error

		switch req.Action {
		case "delete":
			result, err = batchDelete(ctx, entityType, req.IDs, core)
		default:
			response.Error(c, response.ErrBadRequest, "不支持的操作")
			return
		}

		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, result)
	}
}

// batchDelete 批量删除
func batchDelete(ctx context.Context, entityType string, ids []string, core *core.Core) (interface{}, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var count int64
	var err error

	switch entityType {
	case "building":
		service := opsServices.NewBuildingService(core.DB.GetDB())
		err = service.BatchDelete(ctx, ids)
		count = int64(len(ids))
	case "floor":
		service := opsServices.NewFloorService(core.DB.GetDB())
		err = service.BatchDelete(ctx, ids)
		count = int64(len(ids))
	case "workstation":
		service := opsServices.NewWorkstationService(core.DB.GetDB())
		err = service.BatchDelete(ctx, ids)
		count = int64(len(ids))
	case "serverRoom":
		service := opsServices.NewServerRoomService(core.DB.GetDB())
		err = service.BatchDelete(ctx, ids)
		count = int64(len(ids))
	case "roomDevice":
		service := opsServices.NewRoomDeviceService(core.DB.GetDB())
		err = service.BatchDelete(ctx, ids)
		count = int64(len(ids))
	case "dedicatedLine":
		service := opsServices.NewDedicatedLineService(core.DB.GetDB())
		err = service.BatchDelete(ctx, ids)
		count = int64(len(ids))
	case "infoPoint":
		service := opsServices.NewInfoPointService(core.DB.GetDB())
		err = service.BatchDelete(ctx, ids)
		count = int64(len(ids))
	default:
		return nil, fmt.Errorf("不支持的实体类型: %s", entityType)
	}

	if err != nil {
		return nil, err
	}

	return gin.H{
		"count": count,
	}, nil
}

// GetEntityStatusOptions 获取实体状态选项
func GetEntityStatusOptions(entityType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var options []map[string]interface{}

		switch entityType {
		case "building":
			options = []map[string]interface{}{
				{"value": int(operations.BuildingStatusNormal), "label": "正常"},
				{"value": int(operations.BuildingStatusStopped), "label": "停用"},
			}
		case "floor":
			options = []map[string]interface{}{
				{"value": int(operations.FloorStatusNormal), "label": "正常"},
				{"value": int(operations.FloorStatusStopped), "label": "停用"},
			}
		case "serverRoom":
			options = []map[string]interface{}{
				{"value": int(operations.RoomStatusNormal), "label": "正常"},
				{"value": int(operations.RoomStatusStopped), "label": "停用"},
			}
		case "roomDevice":
			options = []map[string]interface{}{
				{"value": 0, "label": "正常"},
				{"value": 1, "label": "故障"},
				{"value": 2, "label": "报废"},
			}
		case "dedicatedLine":
			options = []map[string]interface{}{
				{"value": int(operations.LineStatusNormal), "label": "正常"},
				{"value": int(operations.LineStatusFault), "label": "故障"},
				{"value": int(operations.LineStatusDisabled), "label": "停用"},
			}
		case "infoPoint":
			options = []map[string]interface{}{
				{"value": int(operations.InfoPointStatusNormal), "label": "正常"},
				{"value": int(operations.InfoPointStatusFault), "label": "故障"},
				{"value": int(operations.InfoPointStatusDisabled), "label": "停用"},
			}
		case "workstation":
			options = []map[string]interface{}{
				{"value": 0, "label": "启用"},
				{"value": 1, "label": "禁用"},
			}
		default:
			options = []map[string]interface{}{}
		}

		response.Success(c, options)
	}
}

// isValidExcelFile 验证文件是否为Excel格式
func isValidExcelFile(filename string) bool {
	return len(filename) >= 5 && filename[len(filename)-5:] == ".xlsx"
}

// maxExcelUploadSize Excel 上传文件大小上限(50 MB),防止超大文件 OOM
const maxExcelUploadSize int64 = 50 * 1024 * 1024

// verifyExcelMagicBytes 读取文件前 4 字节,验证 OOXML/ZIP 魔数 (P, K, \x03, \x04)。
// 防止仅扩展名校验被改后缀绕过 (例如上传 .exe 改名为 .xlsx)。
// 调用后会重置文件读位置,后续 excelize.OpenReader 仍能读取全部内容。
func verifyExcelMagicBytes(fileHeader *multipart.FileHeader) error {
	f, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	head := make([]byte, 4)
	n, err := f.Read(head)
	if err != nil || n != 4 {
		return fmt.Errorf("读取文件头失败: %w", err)
	}
	// OOXML / ZIP magic bytes: PK\x03\x04
	if head[0] != 0x50 || head[1] != 0x4B || head[2] != 0x03 || head[3] != 0x04 {
		return fmt.Errorf("文件内容魔数错误: 期望 PK\\x03\\x04, 实际 % X", head)
	}
	return nil
}
