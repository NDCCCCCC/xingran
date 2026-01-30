package system

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// UserHandler 用户处理器
type UserHandler struct {
	service systemServices.UserService
}

// NewUserHandler 创建用户处理器实例
func NewUserHandler(service systemServices.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Create 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body systemServices.UserCreateRequest true "用户信息"
// @Success 200 {object} response.Response{data=models.User}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req systemServices.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "创建成功"})
}

// List 查询用户列表
// @Summary 查询用户列表
// @Description 分页查询用户列表，支持多条件过滤
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body systemServices.UserListParams true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /system/users/list [post]
func (h *UserHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	// 允许空的请求体，设置默认值
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		// 如果是空请求体，初始化默认值
		rawReq = make(map[string]interface{})
	}

	// 转换为结构体并处理类型转换
	params := systemServices.DefaultUserListParams()

	// 处理分页参数
	if val, ok := rawReq["current"]; ok {
		switch v := val.(type) {
		case float64:
			params.Current = int(v)
		case int:
			params.Current = v
		case string:
			params.Current = parseInt(v)
		}
	}
	if val, ok := rawReq["pageSize"]; ok {
		switch v := val.(type) {
		case float64:
			params.PageSize = int(v)
		case int:
			params.PageSize = v
		case string:
			params.PageSize = parseInt(v)
		}
	}

	// 处理字符串字段
	if val, ok := rawReq["username"].(string); ok && val != "" {
		params.Username = &val
	}
	if val, ok := rawReq["nickname"].(string); ok && val != "" {
		params.Nickname = &val
	}
	if val, ok := rawReq["phone"].(string); ok && val != "" {
		params.Phone = &val
	}
	if val, ok := rawReq["deptId"].(string); ok && val != "" {
		params.DeptID = &val
	}
	if val, ok := rawReq["beginTime"].(string); ok && val != "" {
		params.BeginTime = &val
	}
	if val, ok := rawReq["endTime"].(string); ok && val != "" {
		params.EndTime = &val
	}
	if val, ok := rawReq["status"]; ok {
		params.Status = new(int)
		switch v := val.(type) {
		case float64:
			*params.Status = int(v)
		case int:
			*params.Status = v
		case string:
			*params.Status = parseInt(v)
		}
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取用户详情
// @Summary 获取用户详情
// @Description 根据用户ID获取用户详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response{data=models.User}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/users/:id [post]
func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, user)
}

// Update 更新用户
// @Summary 更新用户
// @Description 更新用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body systemServices.UserUpdateRequest true "用户信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/users/:id/update [post]
func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	var req systemServices.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	req.ID = id
	if err := h.service.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除用户
// @Summary 删除用户
// @Description 删除指定用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /system/users/:id/delete [post]
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// BatchDelete 批量删除用户
// @Summary 批量删除用户
// @Description 批量删除多个用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "用户ID列表"
// @Success 200 {object} response.Response
// @Router /system/users/batch [post]
func (h *UserHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// UpdateStatus 更新用户状态
// @Summary 更新用户状态
// @Description 更新用户的启用/禁用状态
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body object{status=int} true "状态"
// @Success 200 {object} response.Response
// @Router /system/users/:id/status [post]
func (h *UserHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	var req struct {
		Status int `json:"status" binding:"min=0,max=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "状态更新成功"})
}

// ResetPassword 重置用户密码
// @Summary 重置用户密码
// @Description 管理员重置指定用户的密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Router /system/users/:id/reset-password [post]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	// 默认密码为123456
	defaultPassword := "123456"
	if err := h.service.ResetPassword(c.Request.Context(), id, defaultPassword); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "重置成功，新密码为：123456"})
}

// ==================== 辅助函数 ====================

// parseInt 安全地将字符串转换为整数
func parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}
