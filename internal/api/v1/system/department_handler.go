package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

// DepartmentHandler 部门处理器
type DepartmentHandler struct {
	service systemServices.DepartmentService
	db      *gorm.DB
	core    *core.Core
}

func NewDepartmentHandler(service systemServices.DepartmentService) *DepartmentHandler {
	var db *gorm.DB
	if svc, ok := service.(interface{ GetDB() *gorm.DB }); ok {
		db = svc.GetDB()
	}
	return &DepartmentHandler{service: service, db: db}
}

// WithCore 注入 core 依赖（Phase 34 操作日志记录所需）。
func (h *DepartmentHandler) WithCore(core *core.Core) *DepartmentHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// GetDB 获取数据库连接
func (h *DepartmentHandler) GetDB() (*gorm.DB, error) {
	if h.db != nil {
		return h.db, nil
	}
	return nil, http.ErrNotSupported
}

// GetTree 获取部门树
// @Summary 获取部门树
// @Description 获取部门树形结构，支持按名称和状态筛选
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param request body requests.DepartmentListParams false "查询条件"
// @Success 200 {object} response.Response
// @Router /system/departments/tree [post]
func (h *DepartmentHandler) GetTree(c *gin.Context) {
	var req requests.DepartmentListParams
	if err := c.ShouldBindJSON(&req); err != nil {
		req = requests.DepartmentListParams{}
	}

	tree, err := h.service.GetTreeWithFilter(c.Request.Context(), true, req)
	if err != nil {
		response.Error(c, err)
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
// @Param request body requests.DepartmentListParams true "查询条件"
// @Success 200 {object} response.Response
// @Router /system/departments/list [post]
func (h *DepartmentHandler) List(c *gin.Context) {
	var req requests.DepartmentListParams
	if err := c.ShouldBindJSON(&req); err != nil {
		req = requests.DepartmentListParams{}
	}

	result, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
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
// @Success 200 {object} response.Response
// @Router /system/departments/:id [post]
func (h *DepartmentHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("部门ID"))
		return
	}

	dept, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
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
// @Param request body requests.DepartmentCreateRequest true "部门信息"
// @Success 200 {object} response.Response
// @Router /system/departments [post]
func (h *DepartmentHandler) Create(c *gin.Context) {
	var req requests.DepartmentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "部门管理", operlog.OperTypeCreate)
	response.Success(c, gin.H{"message": "创建成功"})
}

// Update 更新部门
// @Summary 更新部门
// @Description 更新部门信息
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path string true "部门ID"
// @Param request body requests.DepartmentUpdateRequest true "部门信息"
// @Success 200 {object} response.Response
// @Router /system/departments/:id/update [post]
func (h *DepartmentHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("部门ID"))
		return
	}

	var req requests.DepartmentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	req.ID = id
	if err := h.service.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "部门管理", operlog.OperTypeUpdate)
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
		response.Error(c, apperrors.ParamMissing("部门ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "部门管理", operlog.OperTypeDelete)
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
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "部门管理", operlog.OperTypeBatch)
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
		response.Error(c, apperrors.ParamMissing("部门ID"))
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "部门管理", operlog.OperTypeStatus)
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
		response.Error(c, apperrors.ParamMissing("角色ID"))
		return
	}

	var deptIDs []string
	if err := h.service.GetRoleDeptIDs(c.Request.Context(), roleID, &deptIDs); err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("查询角色部门关联失败"))
		return
	}

	response.Success(c, gin.H{
		"checkedKeys": deptIDs,
	})
}

// GetUsers 获取部门用户列表
// @Summary 获取部门用户列表
// @Description 获取指定部门的用户列表
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path string true "部门ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/departments/{id}/users [get]
func (h *DepartmentHandler) GetUsers(c *gin.Context) {
	deptID := c.Param("id")
	if deptID == "" {
		response.Error(c, apperrors.ParamMissing("部门ID"))
		return
	}

	db, err := h.GetDB()
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("获取数据库连接失败"))
		return
	}

	var users []struct {
		ID       string  `gorm:"column:id" json:"id"`
		Username string  `gorm:"column:username" json:"username"`
		Nickname *string `gorm:"column:nickname" json:"nickname,omitempty"`
		Phone    *string `gorm:"column:phone" json:"phone,omitempty"`
		Email    *string `gorm:"column:email" json:"email,omitempty"`
	}

	err = db.WithContext(c.Request.Context()).
		Table("sys_user").
		Select("id, username, nickname, phone, email").
		Where("dept_id = ? AND deleted_at IS NULL", deptID).
		Order("username ASC").
		Find(&users).Error

	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("查询部门用户失败"))
		return
	}

	response.Success(c, users)
}
