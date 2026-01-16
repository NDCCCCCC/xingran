package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils"
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
	Category string `form:"category" binding:"required"`
}

// buildFileResponse 构建文件响应数据
func buildFileResponse(file services.SysFileInfo, fileURL string) gin.H {
	return gin.H{
		"id":        file.GetID(),
		"fileName":  file.GetFileName(),
		"fileSize":  file.GetFileSize(),
		"fileType":  file.GetFileType(),
		"extension": file.GetExtension(),
		"url":       fileURL,
		"width":     file.GetWidth(),
		"height":    file.GetHeight(),
		"metadata":  file.GetMetadata(),
		"createdAt": file.GetCreatedAt(),
	}
}

// uploadFile 上传文件
func uploadFile(core *core.Core) gin.HandlerFunc {
	fileService := services.NewFileService(core.DB.GetDB())

	return func(c *gin.Context) {
		userID := utils.GetUserID(c)
		if userID == "" {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		var req UploadFileRequest
		if err := c.ShouldBind(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请指定文件分类")
			return
		}

		file, err := c.FormFile("file")
		if err != nil {
			response.Error(c, response.ErrBadRequest, "未找到上传文件")
			return
		}

		validation := services.GetValidationByCategory(req.Category)
		sysFile, err := fileService.UploadFile(c.Request.Context(), file, req.Category, userID, validation)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		userName := utils.GetUsername(c)
		_ = fileService.LogAccess(c.Request.Context(), sysFile.ID, "upload", userID, userName, utils.GetClientIP(c))

		response.Success(c, buildFileResponse(sysFile, fileService.GetFileURL(sysFile)))
	}
}

// getFile 获取文件信息
func getFile(core *core.Core) gin.HandlerFunc {
	fileService := services.NewFileService(core.DB.GetDB())

	return func(c *gin.Context) {
		fileID := c.Param("id")
		if fileID == "" {
			response.Error(c, response.ErrBadRequest, "文件ID不能为空")
			return
		}

		file, err := fileService.GetFile(c.Request.Context(), fileID)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, buildFileResponse(file, fileService.GetFileURL(file)))
	}
}

// deleteFile 删除文件
func deleteFile(core *core.Core) gin.HandlerFunc {
	fileService := services.NewFileService(core.DB.GetDB())

	return func(c *gin.Context) {
		fileID := c.Param("id")
		if fileID == "" {
			response.Error(c, response.ErrBadRequest, "文件ID不能为空")
			return
		}

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
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
}

// listFiles 列出文件
func listFiles(core *core.Core) gin.HandlerFunc {
	fileService := services.NewFileService(core.DB.GetDB())

	return func(c *gin.Context) {
		var req ListFilesRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		pagination := utils.ParsePagination(req.Page, req.PageSize)

		files, total, err := fileService.ListFiles(c.Request.Context(), req.BusinessType, req.UserID, pagination.Offset(), pagination.Limit())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		result := make([]gin.H, 0, len(files))
		for _, file := range files {
			result = append(result, buildFileResponse(file, fileService.GetFileURL(file)))
		}

		response.Success(c, utils.BuildListResponse(result, total, pagination.Page, pagination.PageSize))
	}
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// batchDeleteFiles 批量删除文件
func batchDeleteFiles(core *core.Core) gin.HandlerFunc {
	fileService := services.NewFileService(core.DB.GetDB())

	return func(c *gin.Context) {
		var req BatchDeleteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if err := fileService.BatchDeleteFiles(c.Request.Context(), req.IDs); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, utils.BuildCountResponse(len(req.IDs)))
	}
}

