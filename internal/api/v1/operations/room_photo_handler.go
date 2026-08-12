package operations

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// UploadPhotosRequest 上传照片请求
type UploadPhotosRequest struct {
	RoomID       string `form:"roomId" binding:"required"`
	PrimaryIndex int    `form:"primaryIndex"` // 设置为主图的索引，默认0
}

// UpdateDescriptionRequest 更新描述请求
type UpdateDescriptionRequest struct {
	Description string `json:"description"`
}

// UpdateSortRequest 更新排序请求
type UpdateSortRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1"`
}

// SetupRoomPhotoRouter 设置机房照片路由
func SetupRoomPhotoRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("/upload", uploadPhotos(core))
	r.GET("/list/:roomId", listPhotos(core))
	r.GET("/:id", getPhoto(core))
	r.PUT("/:id/primary", setPrimary(core))
	r.PUT("/:id/description", updateDescription(core))
	r.PUT("/sort", updateSort(core))
	r.DELETE("/:id", deletePhoto(core))
	r.POST("/batch-delete", batchDeletePhotos(core))
}

// uploadPhotos 上传机房照片
func uploadPhotos(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		// 获取参数
		var req UploadPhotosRequest
		if err := c.ShouldBind(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 获取上传的文件
		form, err := c.MultipartForm()
		if err != nil {
			response.Error(c, response.ErrBadRequest, "获取上传文件失败")
			return
		}

		files := form.File["files"]
		if len(files) == 0 {
			response.Error(c, response.ErrBadRequest, "请选择要上传的照片")
			return
		}

		// 限制最多上传20张
		if len(files) > 20 {
			response.Error(c, response.ErrBadRequest, "最多支持同时上传20张照片")
			return
		}

		// 调用服务上传
		service := opsServices.NewRoomPhotoService(core.DB.GetDB())
		photos, err := service.UploadPhotos(c.Request.Context(), req.RoomID, files, req.PrimaryIndex, userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		operlog.Record(c, core.OperLogService, core.GetDB(), "机房照片", operlog.OperTypeUpload,
			operlog.WithOperParam("roomId="+req.RoomID))
		response.Success(c, gin.H{
			"count":  len(photos),
			"photos": photos,
		})
	}
}

// listPhotos 获取机房照片列表
func listPhotos(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomID := c.Param("roomId")
		if roomID == "" {
			response.Error(c, response.ErrBadRequest, "机房ID不能为空")
			return
		}

		service := opsServices.NewRoomPhotoService(core.DB.GetDB())
		photos, err := service.ListByRoom(c.Request.Context(), roomID)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 构建返回数据
		result := make([]gin.H, 0, len(photos))
		for _, photo := range photos {
			result = append(result, gin.H{
				"id":          photo.ID,
				"roomId":      photo.RoomID,
				"fileId":      photo.FileID,
				"fileUrl":     photo.FileURL,
				"sortOrder":   photo.SortOrder,
				"isPrimary":   photo.IsPrimary,
				"description": photo.Description,
				"createdAt":   photo.CreatedAt,
			})
		}

		response.Success(c, result)
	}
}

// getPhoto 获取照片详情
func getPhoto(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		photoID := c.Param("id")
		if photoID == "" {
			response.Error(c, response.ErrBadRequest, "照片ID不能为空")
			return
		}

		service := opsServices.NewRoomPhotoService(core.DB.GetDB())
		photo, file, err := service.GetPhotoWithFile(c.Request.Context(), photoID)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"id":          photo.ID,
			"roomId":      photo.RoomID,
			"fileId":      photo.FileID,
			"fileName":    file.FileName,
			"fileUrl":     photo.FileURL,
			"sortOrder":   photo.SortOrder,
			"isPrimary":   photo.IsPrimary,
			"description": photo.Description,
			"createdAt":   photo.CreatedAt,
		})
	}
}

// setPrimary 设置主图
func setPrimary(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		photoID := c.Param("id")
		if photoID == "" {
			response.Error(c, response.ErrBadRequest, "照片ID不能为空")
			return
		}

		service := opsServices.NewRoomPhotoService(core.DB.GetDB())
		if err := service.SetPrimary(c.Request.Context(), photoID); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		operlog.Record(c, core.OperLogService, core.GetDB(), "机房照片", operlog.OperTypeUpdate,
			operlog.WithOperParam("photoId="+photoID))
		response.Success(c, nil)
	}
}

// updateDescription 更新照片描述
func updateDescription(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		photoID := c.Param("id")
		if photoID == "" {
			response.Error(c, response.ErrBadRequest, "照片ID不能为空")
			return
		}

		var req UpdateDescriptionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		service := opsServices.NewRoomPhotoService(core.DB.GetDB())
		if err := service.UpdateDescription(c.Request.Context(), photoID, req.Description); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		operlog.Record(c, core.OperLogService, core.GetDB(), "机房照片", operlog.OperTypeUpdate,
			operlog.WithOperParam("photoId="+photoID))
		response.Success(c, nil)
	}
}

// updateSort 更新照片排序
func updateSort(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UpdateSortRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		service := opsServices.NewRoomPhotoService(core.DB.GetDB())
		if err := service.UpdateSort(c.Request.Context(), req.PhotoIDs); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		operlog.Record(c, core.OperLogService, core.GetDB(), "机房照片", operlog.OperTypeUpdate)
		response.Success(c, nil)
	}
}

// deletePhoto 删除照片
func deletePhoto(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		photoID := c.Param("id")
		if photoID == "" {
			response.Error(c, response.ErrBadRequest, "照片ID不能为空")
			return
		}

		service := opsServices.NewRoomPhotoService(core.DB.GetDB())
		if err := service.DeletePhoto(c.Request.Context(), photoID); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		operlog.Record(c, core.OperLogService, core.GetDB(), "机房照片", operlog.OperTypeDelete,
			operlog.WithOperParam("photoId="+photoID))
		response.Success(c, nil)
	}
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// batchDeletePhotos 批量删除照片
func batchDeletePhotos(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BatchDeleteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		service := opsServices.NewRoomPhotoService(core.DB.GetDB())
		if err := service.BatchDeletePhotos(c.Request.Context(), req.IDs); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		operlog.Record(c, core.OperLogService, core.GetDB(), "机房照片", operlog.OperTypeDelete,
			operlog.WithOperParam(fmt.Sprintf("count=%d", len(req.IDs))))
		response.Success(c, gin.H{
			"count": len(req.IDs),
		})
	}
}
