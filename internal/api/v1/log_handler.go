package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ==================== 操作日志管理 ====================

// OperLogListRequest 操作日志列表请求
type OperLogListRequest struct {
	Title        *string `json:"title,omitempty"`
	BusinessType *int    `json:"businessType,omitempty"`
	Status       *int    `json:"status,omitempty"`
	OperName     *string `json:"operName,omitempty"`
	BeginTime    *string `json:"beginTime,omitempty"`
	EndTime      *string `json:"endTime,omitempty"`
	Current      int     `json:"current" binding:"min=1"`
	PageSize     int     `json:"pageSize" binding:"min=1,max=100"`
}

// listOperLogs 查询操作日志列表
// @Summary 查询操作日志列表
// @Description 分页查询操作日志列表
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param request body OperLogListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /oper-logs/list [post]
func listOperLogs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			rawReq = make(map[string]interface{})
		}

		req := OperLogListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		}

		if val, ok := rawReq["title"].(string); ok && val != "" {
			req.Title = &val
		}
		if val, ok := rawReq["operName"].(string); ok && val != "" {
			req.OperName = &val
		}
		if val, ok := rawReq["businessType"]; ok && val != nil {
			switch v := val.(type) {
			case float64:
				bt := int(v)
				req.BusinessType = &bt
			case int:
				req.BusinessType = &v
			}
		}
		if val, ok := rawReq["status"]; ok && val != nil {
			switch v := val.(type) {
			case float64:
				s := int(v)
				req.Status = &s
			case int:
				req.Status = &v
			}
		}
		if val, ok := rawReq["beginTime"].(string); ok && val != "" {
			req.BeginTime = &val
		}
		if val, ok := rawReq["endTime"].(string); ok && val != "" {
			req.EndTime = &val
		}

		db := core.DB.GetDB().Model(&models.OperLog{})

		if req.Title != nil && *req.Title != "" {
			db = db.Where("title LIKE ?", "%"+*req.Title+"%")
		}
		if req.BusinessType != nil {
			db = db.Where("business_type = ?", *req.BusinessType)
		}
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
		}
		if req.OperName != nil && *req.OperName != "" {
			db = db.Where("operator_name LIKE ?", "%"+*req.OperName+"%")
		}
		if req.BeginTime != nil && *req.BeginTime != "" {
			db = db.Where("oper_time >= ?", *req.BeginTime)
		}
		if req.EndTime != nil && *req.EndTime != "" {
			db = db.Where("oper_time <= ?", *req.EndTime)
		}

		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询操作日志总数失败")
			return
		}

		var operLogs []models.OperLog
		offset := (req.Current - 1) * req.PageSize
		if err := db.Order("oper_time DESC").Offset(offset).Limit(req.PageSize).Find(&operLogs).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询操作日志列表失败")
			return
		}

		pageResp := response.PageResponse{
			List:     operLogs,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// getOperLog 获取操作日志详情
// @Summary 获取操作日志详情
// @Description 根据ID获取操作日志详情
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param id path string true "操作日志ID"
// @Success 200 {object} response.Response{data=models.OperLog}
// @Router /oper-logs/:id [post]
func getOperLog(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var operLog models.OperLog
		if err := core.DB.GetDB().Where("id = ?", id).First(&operLog).Error; err != nil {
			response.Error(c, response.ErrNotFound, "操作日志不存在")
			return
		}

		response.Success(c, operLog)
	}
}

// deleteOperLog 删除操作日志
// @Summary 删除操作日志
// @Description 删除指定操作日志
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param id path string true "操作日志ID"
// @Success 200 {object} response.Response
// @Router /oper-logs/:id/delete [post]
func deleteOperLog(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		if err := core.DB.GetDB().Where("id = ?", id).Delete(&models.OperLog{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除操作日志失败")
			return
		}

		response.Success(c, nil)
	}
}

// batchDeleteOperLogs 批量删除操作日志
// @Summary 批量删除操作日志
// @Description 批量删除多个操作日志
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param request body map[string][]string true "ID列表"
// @Success 200 {object} response.Response
// @Router /oper-logs/batch-delete [post]
func batchDeleteOperLogs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Ids []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if err := core.DB.GetDB().Where("id IN ?", req.Ids).Delete(&models.OperLog{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "批量删除操作日志失败")
			return
		}

		response.Success(c, nil)
	}
}

// ==================== 登录日志管理 ====================

// LoginLogListRequest 登录日志列表请求
type LoginLogListRequest struct {
	Username  *string `json:"username,omitempty"`
	IPAddr    *string `json:"ipaddr,omitempty"`
	Status    *int    `json:"status,omitempty"`
	BeginTime *string `json:"beginTime,omitempty"`
	EndTime   *string `json:"endTime,omitempty"`
	Current   int     `json:"current" binding:"min=1"`
	PageSize  int     `json:"pageSize" binding:"min=1,max=100"`
}

// listLoginLogs 查询登录日志列表
// @Summary 查询登录日志列表
// @Description 分页查询登录日志列表
// @Tags 登录日志
// @Accept json
// @Produce json
// @Param request body LoginLogListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /login-logs/list [post]
func listLoginLogs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			rawReq = make(map[string]interface{})
		}

		req := LoginLogListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		}

		if val, ok := rawReq["username"].(string); ok && val != "" {
			req.Username = &val
		}
		if val, ok := rawReq["ipaddr"].(string); ok && val != "" {
			req.IPAddr = &val
		}
		if val, ok := rawReq["status"]; ok && val != nil {
			switch v := val.(type) {
			case float64:
				s := int(v)
				req.Status = &s
			case int:
				req.Status = &v
			}
		}
		if val, ok := rawReq["beginTime"].(string); ok && val != "" {
			req.BeginTime = &val
		}
		if val, ok := rawReq["endTime"].(string); ok && val != "" {
			req.EndTime = &val
		}

		db := core.DB.GetDB().Model(&models.LoginLog{})

		if req.Username != nil && *req.Username != "" {
			db = db.Where("user_name LIKE ?", "%"+*req.Username+"%")
		}
		if req.IPAddr != nil && *req.IPAddr != "" {
			db = db.Where("ipaddr LIKE ?", "%"+*req.IPAddr+"%")
		}
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
		}
		if req.BeginTime != nil && *req.BeginTime != "" {
			db = db.Where("login_time >= ?", *req.BeginTime)
		}
		if req.EndTime != nil && *req.EndTime != "" {
			db = db.Where("login_time <= ?", *req.EndTime)
		}

		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询登录日志总数失败")
			return
		}

		var loginLogs []models.LoginLog
		offset := (req.Current - 1) * req.PageSize
		if err := db.Order("login_time DESC").Offset(offset).Limit(req.PageSize).Find(&loginLogs).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询登录日志列表失败")
			return
		}

		pageResp := response.PageResponse{
			List:     loginLogs,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// getLoginLog 获取登录日志详情
// @Summary 获取登录日志详情
// @Description 根据ID获取登录日志详情
// @Tags 登录日志
// @Accept json
// @Produce json
// @Param id path string true "登录日志ID"
// @Success 200 {object} response.Response{data=models.LoginLog}
// @Router /login-logs/:id [post]
func getLoginLog(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var loginLog models.LoginLog
		if err := core.DB.GetDB().Where("id = ?", id).First(&loginLog).Error; err != nil {
			response.Error(c, response.ErrNotFound, "登录日志不存在")
			return
		}

		response.Success(c, loginLog)
	}
}

// deleteLoginLog 删除登录日志
// @Summary 删除登录日志
// @Description 删除指定登录日志
// @Tags 登录日志
// @Accept json
// @Produce json
// @Param id path string true "登录日志ID"
// @Success 200 {object} response.Response
// @Router /login-logs/:id/delete [post]
func deleteLoginLog(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		if err := core.DB.GetDB().Where("id = ?", id).Delete(&models.LoginLog{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除登录日志失败")
			return
		}

		response.Success(c, nil)
	}
}

// batchDeleteLoginLogs 批量删除登录日志
// @Summary 批量删除登录日志
// @Description 批量删除多个登录日志
// @Tags 登录日志
// @Accept json
// @Produce json
// @Param request body map[string][]string true "ID列表"
// @Success 200 {object} response.Response
// @Router /login-logs/batch-delete [post]
func batchDeleteLoginLogs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Ids []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if err := core.DB.GetDB().Where("id IN ?", req.Ids).Delete(&models.LoginLog{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "批量删除登录日志失败")
			return
		}

		response.Success(c, nil)
	}
}

// unlockUser 解锁用户
// @Summary 解锁用户
// @Description 解锁被锁定的用户
// @Tags 登录日志
// @Accept json
// @Produce json
// @Param username path string true "用户名"
// @Success 200 {object} response.Response
// @Router /login-logs/unlock/:username [post]
func unlockUser(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		if username == "" {
			response.Error(c, response.ErrBadRequest, "用户名不能为空")
			return
		}

		// TODO: 实现解锁用户逻辑（如从Redis中删除锁定状态）

		response.Success(c, gin.H{"message": "解锁成功"})
	}
}
