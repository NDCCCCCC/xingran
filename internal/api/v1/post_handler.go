package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ==================== 岗位管理 ====================

// PostListRequest 岗位列表请求
type PostListRequest struct {
	PostCode *string `json:"postCode,omitempty"`
	PostName *string `json:"postName,omitempty"`
	Status   *int    `json:"status,omitempty"`
	Current  int     `json:"current" binding:"min=1"`
	PageSize int     `json:"pageSize" binding:"min=1,max=100"`
}

// PostCreateRequest 创建岗位请求
type PostCreateRequest struct {
	PostCode string  `json:"postCode" binding:"required,min=2,max=64"`
	PostName string  `json:"postName" binding:"required,min=2,max=50"`
	PostSort int     `json:"postSort" binding:"min=0"`
	Status   int     `json:"status" binding:"min=0,max=1,oneof=0 1"`
	Remark   *string `json:"remark,omitempty"`
}

// PostUpdateRequest 更新岗位请求
type PostUpdateRequest struct {
	ID       string  `json:"id,omitempty"`
	PostName string  `json:"postName" binding:"required,min=2,max=50"`
	PostSort int     `json:"postSort" binding:"min=0"`
	Status   int     `json:"status" binding:"min=0,max=1,oneof=0 1"`
	Remark   *string `json:"remark,omitempty"`
}

// listPosts 查询岗位列表
// @Summary 查询岗位列表
// @Description 分页查询岗位列表
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Param request body PostListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /posts/list [post]
func listPosts(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			rawReq = make(map[string]interface{})
		}

		req := PostListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		}

		if val, ok := rawReq["postCode"].(string); ok && val != "" {
			req.PostCode = &val
		}
		if val, ok := rawReq["postName"].(string); ok && val != "" {
			req.PostName = &val
		}
		if statusVal, ok := rawReq["status"]; ok && statusVal != nil {
			var status int
			switch v := statusVal.(type) {
			case string:
				if v == "0" || v == "1" {
					status = parseInt(v)
				}
			case float64:
				status = int(v)
			case int:
				status = v
			}
			req.Status = &status
		}

		db := core.DB.GetDB().Model(&models.Post{})

		if req.PostCode != nil && *req.PostCode != "" {
			db = db.Where("post_code LIKE ?", "%"+*req.PostCode+"%")
		}
		if req.PostName != nil && *req.PostName != "" {
			db = db.Where("post_name LIKE ?", "%"+*req.PostName+"%")
		}
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
		}

		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询岗位总数失败")
			return
		}

		var posts []models.Post
		offset := (req.Current - 1) * req.PageSize
		if err := db.Order("post_sort ASC").Offset(offset).Limit(req.PageSize).Find(&posts).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询岗位列表失败")
			return
		}

		pageResp := response.PageResponse{
			List:     posts,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// createPost 创建岗位
// @Summary 创建岗位
// @Description 创建新岗位
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Param request body PostCreateRequest true "岗位信息"
// @Success 200 {object} response.Response
// @Router /posts [post]
func createPost(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req PostCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查岗位编码是否已存在
		var existPost models.Post
		if err := core.DB.GetDB().Where("post_code = ?", req.PostCode).First(&existPost).Error; err == nil {
			response.Error(c, response.ErrBadRequest, "岗位编码已存在")
			return
		}

		post := models.Post{
			PostCode: req.PostCode,
			PostName: req.PostName,
			PostSort: req.PostSort,
			Status:   models.PostStatus(req.Status),
			Remark:   func() string { if req.Remark != nil { return *req.Remark }; return "" }(),
		}

		if err := core.DB.GetDB().Create(&post).Error; err != nil {
			response.Error(c, response.ErrServerError, "创建岗位失败")
			return
		}

		response.Success(c, gin.H{"id": post.ID})
	}
}

// getPost 获取岗位详情
// @Summary 获取岗位详情
// @Description 根据ID获取岗位详情
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Param id path string true "岗位ID"
// @Success 200 {object} response.Response{data=models.Post}
// @Router /posts/:id [post]
func getPost(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var post models.Post
		if err := core.DB.GetDB().Where("id = ?", id).First(&post).Error; err != nil {
			response.Error(c, response.ErrNotFound, "岗位不存在")
			return
		}

		response.Success(c, post)
	}
}

// updatePost 更新岗位
// @Summary 更新岗位
// @Description 更新岗位信息
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Param id path string true "岗位ID"
// @Param request body PostUpdateRequest true "岗位信息"
// @Success 200 {object} response.Response
// @Router /posts/:id/update [post]
func updatePost(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req PostUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		var post models.Post
		if err := core.DB.GetDB().Where("id = ?", id).First(&post).Error; err != nil {
			response.Error(c, response.ErrNotFound, "岗位不存在")
			return
		}

		updates := map[string]interface{}{
			"post_name": req.PostName,
			"post_sort": req.PostSort,
			"status":    models.PostStatus(req.Status),
		}
		if req.Remark != nil {
			updates["remark"] = req.Remark
		}

		if err := core.DB.GetDB().Model(&post).Updates(updates).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新岗位失败")
			return
		}

		response.Success(c, nil)
	}
}

// deletePost 删除岗位
// @Summary 删除岗位
// @Description 删除指定岗位
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Param id path string true "岗位ID"
// @Success 200 {object} response.Response
// @Router /posts/:id/delete [post]
func deletePost(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var post models.Post
		if err := core.DB.GetDB().Where("id = ?", id).First(&post).Error; err != nil {
			response.Error(c, response.ErrNotFound, "岗位不存在")
			return
		}

		// TODO: 检查是否有关联用户

		if err := core.DB.GetDB().Delete(&post).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除岗位失败")
			return
		}

		response.Success(c, nil)
	}
}

// batchDeletePosts 批量删除岗位
// @Summary 批量删除岗位
// @Description 批量删除多个岗位
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Param request body map[string][]string true "ID列表"
// @Success 200 {object} response.Response
// @Router /posts/batch-delete [post]
func batchDeletePosts(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Ids []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// TODO: 检查是否有关联用户

		if err := core.DB.GetDB().Where("id IN ?", req.Ids).Delete(&models.Post{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "批量删除岗位失败")
			return
		}

		response.Success(c, nil)
	}
}

// SetupPostRouter 设置岗位路由
func SetupPostRouter(router *gin.RouterGroup, core *core.Core) {
	{
		router.POST("/list", listPosts(core))
		router.POST("", createPost(core))
		router.POST("/:id", getPost(core))
		router.POST("/:id/update", updatePost(core))
		router.POST("/:id/delete", deletePost(core))
		router.POST("/batch-delete", batchDeletePosts(core))
	}
}
