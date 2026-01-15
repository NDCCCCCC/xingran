package operations

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// SetupExcelRouter 设置Excel导入导出路由
func SetupExcelRouter(r *gin.RouterGroup, entityType string, core *core.Core) {
	r.POST("/import", importData(entityType, core))
	r.POST("/export", exportData(entityType, core))
	r.GET("/template", downloadTemplate(entityType, core))
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
		excelService := opsServices.NewExcelService(core.DB.GetDB())
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
		// 解析导出参数
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
		excelService := opsServices.NewExcelService(core.DB.GetDB())
		file, err := excelService.ExportData(c.Request.Context(), entityType, params)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 生成文件名
		timestamp := time.Now().Format("20060102_150405")
		filename := entityType + "_" + timestamp + ".xlsx"

		// 将Excel写入buffer
		var buf bytes.Buffer
		if err := file.Write(&buf); err != nil {
			response.Error(c, response.ErrServerError, "生成文件失败")
			return
		}

		// 设置响应头
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
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
		// 生成模板
		excelService := opsServices.NewExcelService(core.DB.GetDB())
		file, err := excelService.GenerateTemplate(entityType)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 生成文件名
		timestamp := time.Now().Format("20060102")
		filename := entityType + "_template_" + timestamp + ".xlsx"

		// 将Excel写入buffer
		var buf bytes.Buffer
		if err := file.Write(&buf); err != nil {
			response.Error(c, response.ErrServerError, "生成文件失败")
			return
		}

		// 设置响应头
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
	}
}

// ImportWithService 使用特定服务导入数据
func ImportWithService(entityType string, core *core.Core) gin.HandlerFunc {
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

		// 根据实体类型调用相应的服务
		ctx := c.Request.Context()
		var result interface{}
		var importErr error

		switch entityType {
		case "building":
			service := opsServices.NewBuildingService(core.DB.GetDB())
			result, importErr = importBuildings(ctx, service, file, userID.(string))
		case "floor":
			service := opsServices.NewFloorService(core.DB.GetDB())
			result, importErr = importFloors(ctx, service, file, userID.(string))
		case "workstation":
			service := opsServices.NewWorkstationService(core.DB.GetDB())
			result, importErr = importWorkstations(ctx, service, file, userID.(string))
		case "serverRoom":
			service := opsServices.NewServerRoomService(core.DB.GetDB())
			result, importErr = importServerRooms(ctx, service, file, userID.(string))
		case "roomDevice":
			service := opsServices.NewRoomDeviceService(core.DB.GetDB())
			result, importErr = importRoomDevices(ctx, service, file, userID.(string))
		case "dedicatedLine":
			service := opsServices.NewDedicatedLineService(core.DB.GetDB())
			result, importErr = importDedicatedLines(ctx, service, file, userID.(string))
		case "infoPoint":
			service := opsServices.NewInfoPointService(core.DB.GetDB())
			result, importErr = importInfoPoints(ctx, service, file, userID.(string))
		default:
			response.Error(c, response.ErrBadRequest, "不支持的实体类型")
			return
		}

		if importErr != nil {
			response.Error(c, response.ErrServerError, importErr.Error())
			return
		}

		response.Success(c, result)
	}
}

// importBuildings 导入楼宇数据
func importBuildings(ctx context.Context, service opsServices.BuildingService, file *multipart.FileHeader, userID string) (interface{}, error) {
	// TODO: 实现具体的导入逻辑
	// 1. 解析Excel文件
	// 2. 验证数据
	// 3. 批量创建记录
	// 4. 返回导入结果

	return gin.H{
		"success": 0,
		"failed":  0,
		"message": "待实现",
	}, nil
}

// importFloors 导入楼层数据
func importFloors(ctx context.Context, service interface{}, file *multipart.FileHeader, userID string) (interface{}, error) {
	return gin.H{
		"success": 0,
		"failed":  0,
		"message": "待实现",
	}, nil
}

// importWorkstations 导入工位数据
func importWorkstations(ctx context.Context, service interface{}, file *multipart.FileHeader, userID string) (interface{}, error) {
	return gin.H{
		"success": 0,
		"failed":  0,
		"message": "待实现",
	}, nil
}

// importServerRooms 导入机房数据
func importServerRooms(ctx context.Context, service interface{}, file *multipart.FileHeader, userID string) (interface{}, error) {
	return gin.H{
		"success": 0,
		"failed":  0,
		"message": "待实现",
	}, nil
}

// importRoomDevices 导入机房设备数据
func importRoomDevices(ctx context.Context, service interface{}, file *multipart.FileHeader, userID string) (interface{}, error) {
	return gin.H{
		"success": 0,
		"failed":  0,
		"message": "待实现",
	}, nil
}

// importDedicatedLines 导入专线数据
func importDedicatedLines(ctx context.Context, service interface{}, file *multipart.FileHeader, userID string) (interface{}, error) {
	return gin.H{
		"success": 0,
		"failed":  0,
		"message": "待实现",
	}, nil
}

// importInfoPoints 导入信息点数据
func importInfoPoints(ctx context.Context, service interface{}, file *multipart.FileHeader, userID string) (interface{}, error) {
	return gin.H{
		"success": 0,
		"failed":  0,
		"message": "待实现",
	}, nil
}

// ExportWithService 使用特定服务导出数据
func ExportWithService(entityType string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 解析导出参数
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

		// 根据实体类型调用相应的服务
		ctx := c.Request.Context()
		var file interface{}
		var filename string
		var exportErr error

		switch entityType {
		case "building":
			service := opsServices.NewBuildingService(core.DB.GetDB())
			file, filename, exportErr = exportBuildings(ctx, service, params)
		case "floor":
			service := opsServices.NewFloorService(core.DB.GetDB())
			file, filename, exportErr = exportFloors(ctx, service, params)
		default:
			response.Error(c, response.ErrBadRequest, "不支持的实体类型")
			return
		}

		if exportErr != nil {
			response.Error(c, response.ErrServerError, exportErr.Error())
			return
		}

		// 设置响应头并返回文件
		switch f := file.(type) {
		case interface{ Write(io.Writer) error }:
			var buf bytes.Buffer
			if err := f.Write(&buf); err != nil {
				response.Error(c, response.ErrServerError, "生成文件失败")
				return
			}
			c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			c.Header("Content-Disposition", "attachment; filename="+filename)
			c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
		default:
			response.Error(c, response.ErrServerError, "不支持的文件类型")
		}
	}
}

// exportBuildings 导出楼宇数据
func exportBuildings(ctx context.Context, service opsServices.BuildingService, params map[string]interface{}) (interface{}, string, error) {
	// TODO: 实现具体的导出逻辑
	// 1. 查询数据
	// 2. 生成Excel文件
	// 3. 返回文件和文件名

	timestamp := time.Now().Format("20060102_150405")
	filename := "buildings_" + timestamp + ".xlsx"

	return nil, filename, nil
}

// exportFloors 导出楼层数据
func exportFloors(ctx context.Context, service interface{}, params map[string]interface{}) (interface{}, string, error) {
	timestamp := time.Now().Format("20060102_150405")
	filename := "floors_" + timestamp + ".xlsx"

	return nil, filename, nil
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
