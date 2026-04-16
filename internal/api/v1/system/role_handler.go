package system

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// RoleHandler 角色处理器
type RoleHandler struct {
	service systemServices.RoleService
}

// NewRoleHandler 创建角色处理器实例
func NewRoleHandler(service systemServices.RoleService) *RoleHandler {
	return &RoleHandler{service: service}
}

// Create 创建角色
// @Summary 创建角色
// @Description 创建新的角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param request body requests.RoleCreateRequest true "角色信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var req requests.RoleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "创建成功"})
}

// List 查询角色列表
// @Summary 查询角色列表
// @Description 分页查询角色列表
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param request body requests.RoleListParams true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /system/roles/list [post]
func (h *RoleHandler) List(c *gin.Context) {
	var req requests.RoleListParams
	// 允许空的请求体，设置默认值
	if err := c.ShouldBindJSON(&req); err != nil {
		req = requests.DefaultRoleListParams()
	}

	result, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取角色详情
// @Summary 获取角色详情
// @Description 根据角色ID获取角色详细信息
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/roles/:id [post]
func (h *RoleHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("角色ID"))
		return
	}

	role, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, role)
}

// Update 更新角色
// @Summary 更新角色
// @Description 更新角色信息
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Param request body requests.RoleUpdateRequest true "角色信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/roles/:id/update [post]
func (h *RoleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("角色ID"))
		return
	}

	var req requests.RoleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	req.ID = id
	if err := h.service.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除角色
// @Summary 删除角色
// @Description 删除指定角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /system/roles/:id/delete [post]
func (h *RoleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("角色ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// BatchDelete 批量删除角色
// @Summary 批量删除角色
// @Description 批量删除多个角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "角色ID列表"
// @Success 200 {object} response.Response
// @Router /system/roles/batch [post]
func (h *RoleHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateStatus 更新角色状态
// @Summary 更新角色状态
// @Description 更新角色的启用/停用状态
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Param request body object{status=int} true "状态"
// @Success 200 {object} response.Response
// @Router /system/roles/:id/status [post]
func (h *RoleHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("角色ID"))
		return
	}

	var req struct {
		Status int `json:"status" binding:"min=0,max=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "状态更新成功"})
}

// GetAllEnabled 获取所有启用的角色
// @Summary 获取所有启用的角色
// @Description 获取所有启用状态的角色，用于下拉选择
// @Tags 角色管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /system/roles/all [post]
func (h *RoleHandler) GetAllEnabled(c *gin.Context) {
	roles, err := h.service.GetAllEnabled(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	// 只返回必要的字段
	var simplifiedRoles []map[string]interface{}
	for _, role := range roles {
		simplifiedRoles = append(simplifiedRoles, map[string]interface{}{
			"id":       role.ID,
			"roleName": role.RoleName,
			"roleKey":  role.RoleKey,
		})
	}

	response.Success(c, simplifiedRoles)
}

// ==================== 辅助函数 ====================
// parseInt 函数已在 user_handler.go 中定义
