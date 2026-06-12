package operations

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"github.com/xuri/excelize/v2"
)

// SetupExcelRouter 设置Excel导入导出路由
func SetupExcelRouter(r *gin.RouterGroup, entityType string, core *core.Core) {
	r.POST("/import", importData(entityType, core))
	r.POST("/export", exportData(entityType, core))
	r.GET("/template", downloadTemplate(entityType, core))
}

func getExcelService(core *core.Core) *opsServices.ExcelService {
	cacheProvider := system.NewCacheAdapter(core.Cache)
	geocodingService := opsServices.NewGeocodingService(core.Config.Baidu.MapAK)
	return opsServices.NewExcelService(core.DB.GetDB(), core.PwdManager, cacheProvider, geocodingService)
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

		filename := generateExcelFilename(entityType, "template")
		if err := writeExcelResponse(c, file, filename); err != nil {
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
