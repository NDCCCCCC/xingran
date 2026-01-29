package operations

import (
	"bytes"
	"context"
	"fmt"
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

// getExcelService 获取Excel服务实例（公共函数，减少重复代码）
func getExcelService(core *core.Core) *opsServices.ExcelService {
	cacheProvider := system.NewCacheAdapter(core.Cache)
	geocodingService := opsServices.NewGeocodingService(core.Config.Baidu.MapAK)
	return opsServices.NewExcelService(core.DB.GetDB(), core.PwdManager, cacheProvider, geocodingService)
}

// generateExcelFilename 生成Excel文件名
func generateExcelFilename(entityType, fileSuffix string) string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s_%s_%s.xlsx", entityType, fileSuffix, timestamp)
}

// writeExcelResponse 写入Excel响应
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

		// 验证文件类型
		if !isValidExcelFile(file.Filename) {
			response.Error(c, response.ErrBadRequest, "只支持.xlsx格式的Excel文件")
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
			"success": result.Success,
			"failed":  result.Failed,
			"errors":  result.Errors,
		})
	}
}

// exportData 导出数据
func exportData(entityType string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ExportRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 构建查询参数
		params := make(map[string]interface{})
		if req.Name != "" {
			params["name"] = req.Name
		}
		if req.Code != "" {
			params["code"] = req.Code
		}
		if req.Status != nil {
			params["status"] = req.Status
		}

		// 调用服务导出数据
		excelService := getExcelService(core)
		file, err := excelService.ExportData(c.Request.Context(), entityType, params)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 使用辅助函数写入响应
		filename := generateExcelFilename(entityType, "export")
		if err := writeExcelResponse(c, file, filename); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
		}
	}
}

// ExportRequest 导出请求
type ExportRequest struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Status *int   `json:"status"`
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

		// 使用辅助函数写入响应
		filename := generateExcelFilename(entityType, "template")
		if err := writeExcelResponse(c, file, filename); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
		}
	}
}

// ImportWithService 使用特定服务导入数据（已废弃，使用通用的 importData）
func ImportWithService(entityType string, core *core.Core) gin.HandlerFunc {
	// 直接使用通用的导入处理
	return importData(entityType, core)
}

// ExportWithService 使用特定服务导出数据（已废弃，使用通用的 exportData）
func ExportWithService(entityType string, core *core.Core) gin.HandlerFunc {
	// 直接使用通用的导出处理
	return exportData(entityType, core)
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
				{"value": int(operations.RoomDeviceStatusNormal), "label": "正常"},
				{"value": int(operations.RoomDeviceStatusFault), "label": "故障"},
				{"value": int(operations.RoomDeviceStatusScrapped), "label": "报废"},
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
