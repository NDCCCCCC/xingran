// Package v1 验证码背景图管理API处理器
package v1

import (
	"encoding/json"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	corepkg "github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// SetupCaptchaBackgroundRouter 设置验证码背景图管理路由
func SetupCaptchaBackgroundRouter(r *gin.RouterGroup, core *corepkg.Core) {
	// 静态路由必须在动态路由（如 :id）之前注册
	r.POST("/upload", uploadCaptchaBackground(core))
	r.POST("/list", listCaptchaBackgrounds(core))
	r.POST("/statistics", getStatistics(core))
	r.POST("/preload", preloadCache(core))
	// 动态路由放在最后
	r.POST("/:id", getCaptchaBackground(core))
	r.POST("/:id/update", updateCaptchaBackground(core))
	r.POST("/:id/delete", deleteCaptchaBackground(core))
	r.POST("/:id/toggle", toggleCaptchaBackgroundStatus(core))
}

// listCaptchaBackgrounds 获取背景图列表
func listCaptchaBackgrounds(core *corepkg.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CaptchaBackgroundListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		db := core.CaptchaBackgroundService.GetDB()

		// 构建查询
		query := db.Model(&models.CaptchaBackground{}).Where("deleted_at IS NULL")

		// 筛选条件
		if req.FileName != nil && *req.FileName != "" {
			query = query.Where("file_name LIKE ?", "%"+*req.FileName+"%")
		}
		if req.PieceShape != nil && *req.PieceShape != "" {
			query = query.Where("piece_shape = ?", *req.PieceShape)
		}
		if req.DifficultyLevel != nil {
			query = query.Where("difficulty_level = ?", *req.DifficultyLevel)
		}
		if req.Status != nil {
			query = query.Where("status = ?", *req.Status)
		}

		// 计算总数
		var total int64
		query.Count(&total)

		// 分页查询
		var backgrounds []*models.CaptchaBackground
		if err := query.Offset((req.Current - 1) * req.PageSize).
			Limit(req.PageSize).
			Order("sort_order ASC, created_at DESC").
			Find(&backgrounds).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询失败")
			return
		}

		// 转换为DTO
		items := make([]*models.CaptchaBackgroundDTO, len(backgrounds))
		for i, bg := range backgrounds {
			items[i] = bg.ToDTO()
		}

		response.Success(c, &models.CaptchaBackgroundListResponse{
			Total: total,
			Items: items,
		})
	}
}

// uploadCaptchaBackground 上传背景图
func uploadCaptchaBackground(core *corepkg.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			response.Error(c, response.ErrBadRequest, "请选择文件")
			return
		}

		// 读取文件数据
		src, err := file.Open()
		if err != nil {
			response.Error(c, response.ErrServerError, "读取文件失败")
			return
		}
		defer src.Close()

		fileData, err := io.ReadAll(src)
		if err != nil {
			response.Error(c, response.ErrServerError, "读取文件数据失败")
			return
		}

		// 解析难度级别
		difficultyLevel := 1
		if diffStr := c.PostForm("difficultyLevel"); diffStr != "" {
			if diff, err := strconv.Atoi(diffStr); err == nil {
				difficultyLevel = diff
			}
		}

		// 解析允许的形状
		var allowedShapes []string
		if shapesStr := c.PostForm("allowedShapes"); shapesStr != "" {
			_ = json.Unmarshal([]byte(shapesStr), &allowedShapes)
		}

		// 构建上传请求
		req := &corepkg.UploadRequest{
			FileName:        file.Filename,
			FileData:        fileData,
			FileSize:        file.Size,
			PieceShape:      models.PieceShape(c.PostForm("pieceShape")),
			DifficultyLevel: difficultyLevel,
			AllowedShapes:   allowedShapes,
			Remark:          c.PostForm("remark"),
		}

		// 上传
		bg, err := core.CaptchaBackgroundService.Upload(c.Request.Context(), req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		operlog.Record(c, core.OperLogService, core.GetDB(), "验证码背景", operlog.OperTypeUpload)
		response.Success(c, bg.ToDTO())
	}
}

// getCaptchaBackground 获取背景图详情
func getCaptchaBackground(core *corepkg.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var bg models.CaptchaBackground
		if err := core.CaptchaBackgroundService.GetDB().Where("id = ?", id).First(&bg).Error; err != nil {
			response.Error(c, response.ErrNotFound, "背景图不存在")
			return
		}

		response.Success(c, bg.ToDTO())
	}
}

// updateCaptchaBackground 更新背景图
func updateCaptchaBackground(core *corepkg.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req models.CaptchaBackgroundUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 构建更新数据
		updates := make(map[string]interface{})

		if req.PieceShape != nil {
			updates["piece_shape"] = *req.PieceShape
		}
		if req.DifficultyLevel != nil {
			updates["difficulty_level"] = *req.DifficultyLevel
		}
		if req.AllowedShapes != nil {
			// 必须转换为 StringArray 类型以正确处理 JSONB
			updates["allowed_shapes"] = models.StringArray(*req.AllowedShapes)
		}
		if req.Status != nil {
			updates["status"] = *req.Status
		}
		if req.SortOrder != nil {
			updates["sort_order"] = *req.SortOrder
		}
		if req.Remark != nil {
			updates["remark"] = *req.Remark
		}

		updates["updated_at"] = time.Now()

		if err := core.CaptchaBackgroundService.GetDB().
			Model(&models.CaptchaBackground{}).
			Where("id = ?", id).
			Updates(updates).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新失败")
			return
		}

		operlog.Record(c, core.OperLogService, core.GetDB(), "验证码背景", operlog.OperTypeUpdate,
			operlog.WithOperParam("id="+id))
		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteCaptchaBackground 删除背景图
func deleteCaptchaBackground(core *corepkg.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// 获取背景图信息
		var bg models.CaptchaBackground
		if err := core.CaptchaBackgroundService.GetDB().Where("id = ?", id).First(&bg).Error; err != nil {
			response.Error(c, response.ErrNotFound, "背景图不存在")
			return
		}

		// 删除文件
		if err := os.Remove(bg.FilePath); err != nil {
			// 静默处理文件删除失败
			_ = err
		}

		// 软删除数据库记录
		if err := core.CaptchaBackgroundService.GetDB().
			Delete(&bg).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除失败")
			return
		}

		operlog.Record(c, core.OperLogService, core.GetDB(), "验证码背景", operlog.OperTypeDelete,
			operlog.WithOperParam("id="+id))
		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// toggleCaptchaBackgroundStatus 切换背景图状态
func toggleCaptchaBackgroundStatus(core *corepkg.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var bg models.CaptchaBackground
		if err := core.CaptchaBackgroundService.GetDB().Where("id = ?", id).First(&bg).Error; err != nil {
			response.Error(c, response.ErrNotFound, "背景图不存在")
			return
		}

		// 切换状态
		newStatus := models.CaptchaBgDisabled
		if bg.Status == models.CaptchaBgDisabled {
			newStatus = models.CaptchaBgEnabled
		}

		if err := core.CaptchaBackgroundService.GetDB().
			Model(&bg).
			Update("status", newStatus).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新状态失败")
			return
		}

		operlog.Record(c, core.OperLogService, core.GetDB(), "验证码背景", operlog.OperTypeStatus,
			operlog.WithOperParam("id="+id))
		response.Success(c, gin.H{"message": "状态更新成功", "status": newStatus})
	}
}

// getStatistics 获取统计信息
func getStatistics(core *corepkg.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := core.CaptchaBackgroundService.GetDB()

		var stats models.StatisticsResponse

		// 总数和启用/禁用数量
		var totalCount, enabledCount, disabledCount int64
		db.Model(&models.CaptchaBackground{}).Where("deleted_at IS NULL").Count(&totalCount)
		db.Model(&models.CaptchaBackground{}).Where("status = ? AND deleted_at IS NULL", models.CaptchaBgEnabled).Count(&enabledCount)
		db.Model(&models.CaptchaBackground{}).Where("status = ? AND deleted_at IS NULL", models.CaptchaBgDisabled).Count(&disabledCount)

		stats.TotalCount = int(totalCount)
		stats.EnabledCount = int(enabledCount)
		stats.DisabledCount = int(disabledCount)

		// 形状分布
		shapeDist := make(map[string]int)
		db.Model(&models.CaptchaBackground{}).
			Where("deleted_at IS NULL").
			Select("piece_shape, count(*) as count").
			Group("piece_shape").
			Scan(&shapeDist)
		stats.ShapeDistribution = shapeDist

		// 难度分布
		diffDist := make(map[int]int)
		rows, _ := db.Model(&models.CaptchaBackground{}).
			Where("deleted_at IS NULL").
			Select("difficulty_level, count(*) as count").
			Group("difficulty_level").
			Rows()

		for rows.Next() {
			var level int
			var count int
			_ = rows.Scan(&level, &count)
			diffDist[level] = count
		}
		rows.Close()
		stats.DifficultyDist = diffDist

		// 总使用次数
		db.Model(&models.CaptchaBackground{}).
			Where("deleted_at IS NULL").
			Select("COALESCE(SUM(use_count), 0)").
			Scan(&stats.TotalUsage)

		response.Success(c, stats)
	}
}

// preloadCache 预加载缓存
func preloadCache(core *corepkg.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := core.CaptchaBackgroundService.PreGeneratePool(c.Request.Context()); err != nil {
			response.Error(c, response.ErrServerError, "预加载失败")
			return
		}

		response.Success(c, gin.H{"message": "预加载完成"})
	}
}

