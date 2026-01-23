package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// DepartmentHandler 部门处理器
type DepartmentHandler struct {
	service systemServices.DepartmentService
}

// NewDepartmentHandler 创建部门处理器实例
func NewDepartmentHandler(service systemServices.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{service: service}
}

// GetTree 获取部门树
// @Summary 获取部门树
// @Description 获取部门树形结构
// @Tags 部门管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Department}
// @Router /system/departments/tree [post]
func (h *DepartmentHandler) GetTree(c *gin.Context) {
	tree, err := h.service.GetTree(c.Request.Context(), true)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, tree)
}

// List 查询部门列表
// @Summary 查询部门列表
// @Description 查询部门列表
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param request body systemServices.DepartmentListParams true "查询条件"
// @Success 200 {object} response.Response{data=[]models.Department}
// @Router /system/departments/list [post]
func (h *DepartmentHandler) List(c *gin.Context) {
	var req systemServices.DepartmentListParams
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求
		req = systemServices.DepartmentListParams{}
	}

	result, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// GetByID 获取部门详情
// @Summary 获取部门详情
// @Description 根据部门ID获取部门详细信息
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path string true "部门ID"
// @Success 200 {object} response.Response{data=models.Department}
// @Router /system/departments/:id [post]
func (h *DepartmentHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "部门ID不能为空")
		return
	}

	dept, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, dept)
}

// Create 创建部门
// @Summary 创建部门
// @Description 创建新的部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param request body systemServices.DepartmentCreateRequest true "部门信息"
// @Success 200 {object} response.Response{data=models.Department}
// @Router /system/departments [post]
func (h *DepartmentHandler) Create(c *gin.Context) {
	var req systemServices.DepartmentCreateRequest
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

// Update 更新部门
// @Summary 更新部门
// @Description 更新部门信息
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path string true "部门ID"
// @Param request body systemServices.DepartmentUpdateRequest true "部门信息"
// @Success 200 {object} response.Response
// @Router /system/departments/:id/update [post]
func (h *DepartmentHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "部门ID不能为空")
		return
	}

	var req systemServices.DepartmentUpdateRequest
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

// Delete 删除部门
// @Summary 删除部门
// @Description 删除指定部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path string true "部门ID"
// @Success 200 {object} response.Response
// @Router /system/departments/:id/delete [post]
func (h *DepartmentHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "部门ID不能为空")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// BatchDelete 批量删除部门
// @Summary 批量删除部门
// @Description 批量删除多个部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "部门ID列表"
// @Success 200 {object} response.Response
// @Router /system/departments/batch [post]
func (h *DepartmentHandler) BatchDelete(c *gin.Context) {
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

// UpdateStatus 更新部门状态
// @Summary 更新部门状态
// @Description 更新部门的启用/停用状态
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path string true "部门ID"
// @Param request body object{status=int} true "状态"
// @Success 200 {object} response.Response
// @Router /system/departments/:id/status [post]
func (h *DepartmentHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "部门ID不能为空")
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

// RoleDeptTreeSelect 获取角色部门树选择器
// @Summary 获取角色部门树选择器
// @Description 获取指定角色的部门树和已选中的部门ID，用于角色部门权限分配
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param roleId path string true "角色ID"
// @Success 200 {object} response.Response{data=object{checkedKeys=[]string}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/departments/role-dept-tree-select/{roleId} [post]
func (h *DepartmentHandler) RoleDeptTreeSelect(c *gin.Context) {
	roleID := c.Param("roleId")
	if roleID == "" {
		response.Error(c, http.StatusBadRequest, "角色ID不能为空")
		return
	}

	// 查询角色已关联的部门ID
	var deptIDs []string
	if err := h.service.GetRoleDeptIDs(c.Request.Context(), roleID, &deptIDs); err != nil {
		response.Error(c, http.StatusInternalServerError, "查询角色部门关联失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"checkedKeys": deptIDs,
	})
}
