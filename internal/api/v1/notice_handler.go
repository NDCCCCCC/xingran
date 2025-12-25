package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// NoticeType 通知类型枚举
type NoticeType string

const (
	NoticeTypeNotice NoticeType = "1" // 公告
	NoticeTypeAlert  NoticeType = "2" // 警告
)

// NoticeListRequest 通知公告列表请求
type NoticeListRequest struct {
	NoticeTitle *string     `json:"noticeTitle,omitempty"`
	NoticeType  *NoticeType `json:"noticeType,omitempty"`
	CreateTime  *string     `json:"createTime,omitempty"`
	Current     int         `json:"current" binding:"min=1"`
	PageSize    int         `json:"pageSize" binding:"min=1,max=100"`
}

// NoticeCreateRequest 创建通知公告请求
type NoticeCreateRequest struct {
	NoticeTitle   string     `json:"noticeTitle" binding:"required,min=2,max=50"`
	NoticeType    NoticeType `json:"noticeType" binding:"required,oneof=1 2"`
	NoticeContent *string    `json:"noticeContent,omitempty"`
	Status        int        `json:"status" binding:"min=0,max=1,oneof=0 1"`
}

// NoticeUpdateRequest 更新通知公告请求
type NoticeUpdateRequest struct {
	ID            string     `json:"id,omitempty"`
	NoticeTitle   string     `json:"noticeTitle" binding:"required,min=2,max=50"`
	NoticeType    NoticeType `json:"noticeType" binding:"required,oneof=1 2"`
	NoticeContent *string    `json:"noticeContent,omitempty"`
	Status        int        `json:"status" binding:"min=0,max=1,oneof=0 1"`
}

// listNotices 查询通知公告列表
// @Summary 查询通知公告列表
// @Description 分页查询通知公告列表
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param request body NoticeListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /notices/list [post]
func listNotices(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			rawReq = make(map[string]interface{})
		}

		req := NoticeListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		}

		if val, ok := rawReq["noticeTitle"].(string); ok && val != "" {
			req.NoticeTitle = &val
		}
		if val, ok := rawReq["noticeType"].(string); ok && val != "" {
			noticeType := NoticeType(val)
			req.NoticeType = &noticeType
		}
		if val, ok := rawReq["createTime"].(string); ok && val != "" {
			req.CreateTime = &val
		}

		db := core.DB.GetDB().Model(&models.Notice{})

		if req.NoticeTitle != nil && *req.NoticeTitle != "" {
			db = db.Where("notice_title LIKE ?", "%"+*req.NoticeTitle+"%")
		}
		if req.NoticeType != nil {
			db = db.Where("notice_type = ?", *req.NoticeType)
		}
		if req.CreateTime != nil && *req.CreateTime != "" {
			db = db.Where("DATE(created_at) = ?", *req.CreateTime)
		}

		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询通知公告总数失败")
			return
		}

		var notices []models.Notice
		offset := (req.Current - 1) * req.PageSize
		if err := db.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&notices).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询通知公告列表失败")
			return
		}

		pageResp := response.PageResponse{
			List:     notices,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// createNotice 创建通知公告
// @Summary 创建通知公告
// @Description 创建新通知公告
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param request body NoticeCreateRequest true "通知公告信息"
// @Success 200 {object} response.Response
// @Router /notices [post]
func createNotice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req NoticeCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		notice := models.Notice{
			NoticeTitle:   req.NoticeTitle,
			NoticeType:    string(req.NoticeType),
			NoticeContent: req.NoticeContent,
			Status:        req.Status,
		}

		if err := core.DB.GetDB().Create(&notice).Error; err != nil {
			response.Error(c, response.ErrServerError, "创建通知公告失败")
			return
		}

		response.Success(c, gin.H{"id": notice.ID})
	}
}

// getNotice 获取通知公告详情
// @Summary 获取通知公告详情
// @Description 根据ID获取通知公告详情
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response{data=models.Notice}
// @Router /notices/:id [post]
func getNotice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var notice models.Notice
		if err := core.DB.GetDB().Where("id = ?", id).First(&notice).Error; err != nil {
			response.Error(c, response.ErrNotFound, "通知公告不存在")
			return
		}

		response.Success(c, notice)
	}
}

// updateNotice 更新通知公告
// @Summary 更新通知公告
// @Description 更新通知公告信息
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Param request body NoticeUpdateRequest true "通知公告信息"
// @Success 200 {object} response.Response
// @Router /notices/:id/update [post]
func updateNotice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req NoticeUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		var notice models.Notice
		if err := core.DB.GetDB().Where("id = ?", id).First(&notice).Error; err != nil {
			response.Error(c, response.ErrNotFound, "通知公告不存在")
			return
		}

		updates := map[string]interface{}{
			"notice_title": req.NoticeTitle,
			"notice_type":  string(req.NoticeType),
			"status":       req.Status,
		}
		if req.NoticeContent != nil {
			updates["notice_content"] = req.NoticeContent
		}

		if err := core.DB.GetDB().Model(&notice).Updates(updates).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新通知公告失败")
			return
		}

		response.Success(c, nil)
	}
}

// deleteNotice 删除通知公告
// @Summary 删除通知公告
// @Description 删除指定通知公告
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response
// @Router /notices/:id/delete [post]
func deleteNotice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var notice models.Notice
		if err := core.DB.GetDB().Where("id = ?", id).First(&notice).Error; err != nil {
			response.Error(c, response.ErrNotFound, "通知公告不存在")
			return
		}

		if err := core.DB.GetDB().Delete(&notice).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除通知公告失败")
			return
		}

		response.Success(c, nil)
	}
}

// batchDeleteNotices 批量删除通知公告
// @Summary 批量删除通知公告
// @Description 批量删除多个通知公告
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param request body map[string][]string true "ID列表"
// @Success 200 {object} response.Response
// @Router /notices/batch-delete [post]
func batchDeleteNotices(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Ids []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if err := core.DB.GetDB().Where("id IN ?", req.Ids).Delete(&models.Notice{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "批量删除通知公告失败")
			return
		}

		response.Success(c, nil)
	}
}
