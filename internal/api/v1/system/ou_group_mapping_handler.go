package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	addomainServices "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// OUGroupMappingHandler OU组映射处理器
type OUGroupMappingHandler struct {
	mappingService *addomainServices.OUGroupMappingService
	core           *core.Core
}

// NewOUGroupMappingHandler 创建处理器实例
func NewOUGroupMappingHandler(mappingService *addomainServices.OUGroupMappingService, core *core.Core) *OUGroupMappingHandler {
	return &OUGroupMappingHandler{
		mappingService: mappingService,
		core:           core,
	}
}

// SetupOUGroupMappingRouter 设置OU组映射路由
func SetupOUGroupMappingRouter(r *gin.RouterGroup, core *core.Core) {
	mappingService := addomainServices.NewOUGroupMappingService(core.GetDB())
	handler := NewOUGroupMappingHandler(mappingService, core)

	// OU组映射管理
	mappings := r.Group("/ou-group-mappings")
	{
		mappings.POST("/list", handler.ListMappings)
		mappings.POST("", handler.CreateMapping)
		mappings.GET("/:id", handler.GetMapping)
		mappings.PUT("/:id", handler.UpdateMapping)
		mappings.DELETE("/:id", handler.DeleteMapping)
		mappings.GET("/ou/:ouDn", handler.GetMappingsByOU)
	}
}

// ListMappings 查询映射列表
// @Summary 查询OU组映射列表
// @Description 查询OU与用户组的映射列表，支持按配置、OU、组名、状态筛选和分页
// @Tags OU组映射管理
// @Accept json
// @Produce json
// @Param request body addomainServices.ListMappingsRequest true "查询参数"
// @Success 200 {object} response.Response{data=addomainServices.ListMappingsResponse}
// @Router /ad-domain/ou-group-mappings/list [post]
func (h *OUGroupMappingHandler) ListMappings(c *gin.Context) {
	var req addomainServices.ListMappingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.mappingService.ListMappings(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// CreateMapping 创建OU组映射
// @Summary 创建OU组映射
// @Description 创建OU与用户组的映射关系
// @Tags OU组映射管理
// @Accept json
// @Produce json
// @Param request body addomainServices.CreateMappingRequest true "映射信息"
// @Success 200 {object} response.Response{data=models.OUGroupMapping}
// @Router /ad-domain/ou-group-mappings [post]
func (h *OUGroupMappingHandler) CreateMapping(c *gin.Context) {
	var req addomainServices.CreateMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 从JWT上下文中获取当前用户ID作为审计字段
	userID := c.GetString("user_id")
	req.CreatedBy = userID

	mapping, err := h.mappingService.CreateMapping(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	recordOperLog(c, h.core, "OU组映射", OperTypeCreate)

	response.Success(c, mapping)
}

// GetMapping 获取单个映射
// @Summary 获取OU组映射详情
// @Description 根据ID获取OU组映射的详细信息
// @Tags OU组映射管理
// @Accept json
// @Produce json
// @Param id path string true "映射ID"
// @Success 200 {object} response.Response{data=models.OUGroupMapping}
// @Router /ad-domain/ou-group-mappings/{id} [get]
func (h *OUGroupMappingHandler) GetMapping(c *gin.Context) {
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

// UpdateMapping 更新OU组映射
// @Summary 更新OU组映射
// @Description 更新OU组映射的配置信息
// @Tags OU组映射管理
// @Accept json
// @Produce json
// @Param id path string true "映射ID"
// @Param request body addomainServices.UpdateMappingRequest true "更新信息"
// @Success 200 {object} response.Response
// @Router /ad-domain/ou-group-mappings/{id} [put]
func (h *OUGroupMappingHandler) UpdateMapping(c *gin.Context) {
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

	// 从JWT上下文中获取当前用户ID作为审计字段
	req.UpdatedBy = c.GetString("user_id")

	if err := h.mappingService.UpdateMapping(c.Request.Context(), id, &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	recordOperLog(c, h.core, "OU组映射", OperTypeUpdate)

	response.Success(c, nil)
}

// DeleteMapping 删除OU组映射
// @Summary 删除OU组映射
// @Description 根据ID删除OU组映射
// @Tags OU组映射管理
// @Accept json
// @Produce json
// @Param id path string true "映射ID"
// @Success 200 {object} response.Response
// @Router /ad-domain/ou-group-mappings/{id} [delete]
func (h *OUGroupMappingHandler) DeleteMapping(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "缺少映射ID")
		return
	}

	if err := h.mappingService.DeleteMapping(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	recordOperLog(c, h.core, "OU组映射", OperTypeDelete)

	response.Success(c, nil)
}

// GetMappingsByOU 获取OU的所有关联组
// @Summary 获取OU的所有关联组
// @Description 根据OU的DN获取该OU关联的所有用户组
// @Tags OU组映射管理
// @Accept json
// @Produce json
// @Param ouDn path string true "OU的DN"
// @Success 200 {object} response.Response{data=[]models.OUGroupMapping}
// @Router /ad-domain/ou-group-mappings/ou/{ouDn} [get]
func (h *OUGroupMappingHandler) GetMappingsByOU(c *gin.Context) {
	ouDn := c.Param("ouDn")
	if ouDn == "" {
		response.Error(c, http.StatusBadRequest, "缺少OU DN")
		return
	}

	mappings, err := h.mappingService.GetMappingsByOU(c.Request.Context(), ouDn)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, mappings)
}