package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	addomainServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ADGroupMappingHandler 部门组映射处理器
type ADGroupMappingHandler struct {
	mappingService addomainServices.DeptGroupMappingService
	core           *core.Core
}

// NewADGroupMappingHandler 创建处理器实例
func NewADGroupMappingHandler(mappingService addomainServices.DeptGroupMappingService, core *core.Core) *ADGroupMappingHandler {
	return &ADGroupMappingHandler{
		mappingService: mappingService,
		core:           core,
	}
}

// SetupADGroupMappingRouter 设置部门组映射路由
func SetupADGroupMappingRouter(r *gin.RouterGroup, core *core.Core) {
	mappingService := addomainServices.NewDeptGroupMappingService(core.GetDB())
	handler := NewADGroupMappingHandler(mappingService, core)

	// 部门组映射管理
	mappings := r.Group("/mappings")
	mappings.Use(middleware.RequirePermissions([]string{
		"ops:ad:group:mapping:view",
		"ops:ad:group:mapping:add",
		"ops:ad:group:mapping:edit",
		"ops:ad:group:mapping:delete",
		"ops:ad:group:mapping:automap",
		"ops:ad:group:mapping:sync",
	}, core))
	{
		mappings.POST("/list", handler.ListMappings)
		mappings.POST("", handler.CreateMapping)
		mappings.GET("/:id", handler.GetMapping)
		mappings.POST("/:id/update", handler.UpdateMapping)
		mappings.POST("/:id/delete", handler.DeleteMapping)
		mappings.POST("/auto-map", handler.AutoMapDepartment)
	}
}

// ListMappings 查询映射列表
func (h *ADGroupMappingHandler) ListMappings(c *gin.Context) {
	var req addomainServices.ListMappingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 设置默认分页
	if req.Current <= 0 {
		req.Current = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	result, err := h.mappingService.ListMappings(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// CreateMapping 创建部门组映射
func (h *ADGroupMappingHandler) CreateMapping(c *gin.Context) {
	var req addomainServices.CreateMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	mapping, err := h.mappingService.CreateMapping(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, mapping)
}

// GetMapping 获取单个映射
func (h *ADGroupMappingHandler) GetMapping(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "缺少映射ID")
		return
	}

	mapping, err := h.mappingService.GetMapping(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, mapping)
}

// UpdateMapping 更新部门组映射
func (h *ADGroupMappingHandler) UpdateMapping(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "缺少映射ID")
		return
	}

	var req addomainServices.UpdateMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if err := h.mappingService.UpdateMapping(c.Request.Context(), id, &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// DeleteMapping 删除部门组映射
func (h *ADGroupMappingHandler) DeleteMapping(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "缺少映射ID")
		return
	}

	if err := h.mappingService.DeleteMapping(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// AutoMapDepartment 自动映射单个部门
func (h *ADGroupMappingHandler) AutoMapDepartment(c *gin.Context) {
	var req struct {
		DeptID   string `json:"deptId" binding:"required"`
		ConfigID string `json:"configId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	mapping, err := h.mappingService.AutoMapDepartment(c.Request.Context(), req.DeptID, req.ConfigID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, mapping)
}
