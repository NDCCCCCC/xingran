package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// SetupFileRouter 设置文件管理路由
func SetupFileRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("/upload", uploadFile(core))
	r.GET("/:id", getFile(core))
	r.DELETE("/:id", deleteFile(core))
	r.GET("", listFiles(core))
	r.POST("/batch-delete", batchDeleteFiles(core))
}

// UploadFileRequest 上传文件请求
type UploadFileRequest struct {
	Category string `form:"category" binding:"required"` // room-photo, document, etc.
}

// uploadFile 上传文件
func uploadFile(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		// 获取分类参数
		var req UploadFileRequest
		if err := c.ShouldBind(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请指定文件分类")
			return
		}

		// 获取上传的文件
		file, err := c.FormFile("file")
		if err != nil {
			response.Error(c, response.ErrBadRequest, "未找到上传文件")
			return
		}

		// 根据分类选择验证规则
		var validation *services.FileValidation
		switch req.Category {
		case "room-photo", "avatar", "image":
			validation = services.ImageValidation
		case "document", "import", "export":
			validation = services.DocumentValidation
		default:
			validation = services.ImageValidation // 默认使用图片验证
		}

		// 调用服务上传文件
		fileService := services.NewFileService(core.DB.GetDB())
		sysFile, err := fileService.UploadFile(c.Request.Context(), file, req.Category, userID.(string), validation)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录上传日志
		userName, _ := c.Get("username")
		ip := c.ClientIP()
		_ = fileService.LogAccess(c.Request.Context(), sysFile.ID, "upload", userID.(string), userName.(string), ip)

		response.Success(c, gin.H{
			"id":        sysFile.ID,
			"fileName":  sysFile.FileName,
			"fileSize":  sysFile.FileSize,
			"fileType":  sysFile.FileType,
			"extension": sysFile.Extension,
			"url":       fileService.GetFileURL(sysFile),
			"createdAt": sysFile.CreatedAt,
		})
	}
}

// getFile 获取文件信息
func getFile(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileID := c.Param("id")
		if fileID == "" {
			response.Error(c, response.ErrBadRequest, "文件ID不能为空")
			return
		}

		fileService := services.NewFileService(core.DB.GetDB())
		file, err := fileService.GetFile(c.Request.Context(), fileID)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"id":           file.ID,
			"fileName":     file.FileName,
			"fileSize":     file.FileSize,
			"fileType":     file.FileType,
			"extension":    file.Extension,
			"url":          fileService.GetFileURL(file),
			"businessType": file.BusinessType,
			"uploaderID":   file.UploaderID,
			"createdAt":    file.CreatedAt,
		})
	}
}

// deleteFile 删除文件
func deleteFile(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileID := c.Param("id")
		if fileID == "" {
			response.Error(c, response.ErrBadRequest, "文件ID不能为空")
			return
		}

		fileService := services.NewFileService(core.DB.GetDB())
		if err := fileService.DeleteFile(c.Request.Context(), fileID); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// ListFilesRequest 列出文件请求
type ListFilesRequest struct {
	BusinessType string `form:"businessType"`
	UserID       string `form:"userId"`
	Page         int    `form:"page" binding:"min=1"`
	PageSize     int    `form:"pageSize" binding:"min=1,max=100"`
}

// listFiles 列出文件
func listFiles(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ListFilesRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 默认分页参数
		if req.Page == 0 {
			req.Page = 1
		}
		if req.PageSize == 0 {
			req.PageSize = 20
		}

		offset := (req.Page - 1) * req.PageSize

		fileService := services.NewFileService(core.DB.GetDB())
		files, total, err := fileService.ListFiles(c.Request.Context(), req.BusinessType, req.UserID, offset, req.PageSize)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 构建返回数据
		result := make([]gin.H, 0, len(files))
		for _, file := range files {
			result = append(result, gin.H{
				"id":           file.ID,
				"fileName":     file.FileName,
				"fileSize":     file.FileSize,
				"fileType":     file.FileType,
				"extension":    file.Extension,
				"url":          fileService.GetFileURL(file),
				"businessType": file.BusinessType,
				"uploaderID":   file.UploaderID,
				"createdAt":    file.CreatedAt,
			})
		}

		response.Success(c, gin.H{
			"list":     result,
			"total":    total,
			"page":     req.Page,
			"pageSize": req.PageSize,
		})
	}
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// batchDeleteFiles 批量删除文件
func batchDeleteFiles(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BatchDeleteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		fileService := services.NewFileService(core.DB.GetDB())
		if err := fileService.BatchDeleteFiles(c.Request.Context(), req.IDs); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"count": len(req.IDs),
		})
	}
}

// GetAccessLogsRequest 获取访问日志请求
type GetAccessLogsRequest struct {
	Page     int `form:"page" binding:"min=1"`
	PageSize int `form:"pageSize" binding:"min=1,max=100"`
}

// getFileAccessLogs 获取文件访问日志
func getFileAccessLogs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileID := c.Param("id")
		if fileID == "" {
			response.Error(c, response.ErrBadRequest, "文件ID不能为空")
			return
		}

		var req GetAccessLogsRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 默认分页参数
		if req.Page == 0 {
			req.Page = 1
		}
		if req.PageSize == 0 {
			req.PageSize = 20
		}

		offset := (req.Page - 1) * req.PageSize

		fileService := services.NewFileService(core.DB.GetDB())
		logs, total, err := fileService.GetAccessLogs(c.Request.Context(), fileID, offset, req.PageSize)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"list":     logs,
			"total":    total,
			"page":     req.Page,
			"pageSize": req.PageSize,
		})
	}
}
