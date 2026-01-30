package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// FileHandler 文件处理器
type FileHandler struct {
	service *systemServices.FileService
}

// NewFileHandler 创建文件处理器实例
func NewFileHandler(service *systemServices.FileService) *FileHandler {
	return &FileHandler{service: service}
}

// Upload 上传文件
func (h *FileHandler) Upload(c *gin.Context) {
	userID := utils.GetUserID(c)
	if userID == "" {
		response.Error(c, response.ErrUnauthorized)
		return
	}

	var req systemServices.UploadFileRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请指定文件分类")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, response.ErrBadRequest, "未找到上传文件")
		return
	}

	validation := systemServices.GetValidationByCategory(req.Category)
	sysFile, err := h.service.UploadFile(c.Request.Context(), file, req.Category, userID, validation)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	userName := utils.GetUsername(c)
	_ = h.service.LogAccess(c.Request.Context(), sysFile.ID, "upload", userID, userName, utils.GetClientIP(c))

	response.Success(c, buildFileResponse(sysFile, h.service.GetFileURL(sysFile)))
}

// GetByID 获取文件信息
func (h *FileHandler) GetByID(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		response.Error(c, response.ErrBadRequest, "文件ID不能为空")
		return
	}

	file, err := h.service.GetFile(c.Request.Context(), fileID)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	response.Success(c, buildFileResponse(file, h.service.GetFileURL(file)))
}

// Delete 删除文件
func (h *FileHandler) Delete(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		response.Error(c, response.ErrBadRequest, "文件ID不能为空")
		return
	}

	if err := h.service.DeleteFile(c.Request.Context(), fileID); err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// List 列出文件
func (h *FileHandler) List(c *gin.Context) {
	var req systemServices.ListFilesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	pagination := utils.ParsePagination(req.Page, req.PageSize)

	files, total, err := h.service.ListFiles(c.Request.Context(), req.BusinessType, req.UserID, pagination.Offset(), pagination.Limit())
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	result := make([]gin.H, 0, len(files))
	for _, file := range files {
		result = append(result, buildFileResponse(file, h.service.GetFileURL(file)))
	}

	response.Success(c, utils.BuildListResponse(result, total, pagination.Page, pagination.PageSize))
}

// BatchDelete 批量删除文件
func (h *FileHandler) BatchDelete(c *gin.Context) {
	var req systemServices.BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	if err := h.service.BatchDeleteFiles(c.Request.Context(), req.IDs); err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	response.Success(c, utils.BuildCountResponse(len(req.IDs)))
}

// buildFileResponse 构建文件响应数据
func buildFileResponse(file systemServices.SysFileInfo, fileURL string) gin.H {
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
