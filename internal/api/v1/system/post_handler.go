package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// PostHandler 岗位处理器
type PostHandler struct {
	service systemServices.PostService
	core    *core.Core
}

// NewPostHandler 创建岗位处理器实例
func NewPostHandler(service systemServices.PostService) *PostHandler {
	return &PostHandler{service: service}
}

// WithCore 注入 core 依赖（Phase 34 操作日志记录所需）。
func (h *PostHandler) WithCore(core *core.Core) *PostHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Statistics 岗位统计(读操作,不记操作日志)
func (h *PostHandler) Statistics(c *gin.Context) {
	result, err := h.service.Statistics(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Create 创建岗位
// @Summary 创建岗位
// @Description 创建新的岗位
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Param request body object{postCode=string,postName=string,postSort=int,status=int,remark=string} true "岗位信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/posts [post]
func (h *PostHandler) Create(c *gin.Context) {
	var req requests.PostCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "岗位管理", operlog.OperTypeCreate)
	response.Success(c, gin.H{"message": "创建成功"})
}

// List 查询岗位列表
// @Summary 查询岗位列表
// @Description 查询岗位列表，支持多条件筛选和分页
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Param request body object{postCode=string,postName=string,status=int,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/posts/list [post]
func (h *PostHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	params := requests.DefaultPostListParams()

	// 处理分页参数
	if val, ok := rawReq["current"]; ok {
		switch v := val.(type) {
		case float64:
			params.Current = int(v)
		case int:
			params.Current = v
		}
	}
	if val, ok := rawReq["pageSize"]; ok {
		switch v := val.(type) {
		case float64:
			params.PageSize = int(v)
		case int:
			params.PageSize = v
		}
	}

	// 处理字符串字段
	if val, ok := rawReq["postCode"].(string); ok && val != "" {
		params.PostCode = &val
	}
	if val, ok := rawReq["postName"].(string); ok && val != "" {
		params.PostName = &val
	}
	if val, ok := rawReq["status"]; ok && val != nil {
		switch v := val.(type) {
		case string:
			if v == "0" || v == "1" {
				status := parseInt(v)
				params.Status = &status
			}
		case float64:
			status := int(v)
			params.Status = &status
		case int:
			params.Status = &v
		}
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取岗位详情
// @Summary 获取岗位详情
// @Description 根据岗位ID获取详细信息
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Param id path string true "岗位ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/posts/:id [post]
func (h *PostHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("岗位ID"))
		return
	}

	post, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, post)
}

// Update 更新岗位
// @Summary 更新岗位
// @Description 更新岗位信息
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Param id path string true "岗位ID"
// @Param request body object{postCode=string,postName=string,postSort=int,status=int,remark=string} true "岗位信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/posts/:id/update [post]
func (h *PostHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("岗位ID"))
		return
	}

	var req requests.PostUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	req.ID = id
	if err := h.service.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "岗位管理", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除岗位
// @Summary 删除岗位
// @Description 删除指定的岗位
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Param id path string true "岗位ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/posts/:id/delete [post]
func (h *PostHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("岗位ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "岗位管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// BatchDelete 批量删除岗位
// @Summary 批量删除岗位
// @Description 批量删除多个岗位
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "岗位ID列表"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/posts/batch [post]
func (h *PostHandler) BatchDelete(c *gin.Context) {
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "岗位管理", operlog.OperTypeBatch)
	response.Success(c, nil)
}

// GetAll 获取所有岗位（使用缓存）
// @Summary 获取所有岗位
// @Description 获取所有岗位列表（使用缓存）
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/posts/all [post]
func (h *PostHandler) GetAll(c *gin.Context) {
	posts, err := h.service.GetAllWithCache(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, posts)
}

// GetAllEnabled 获取启用的岗位（使用缓存）
// @Summary 获取启用的岗位
// @Description 获取所有启用状态的岗位列表（使用缓存）
// @Tags 岗位管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/posts/enabled [post]
func (h *PostHandler) GetAllEnabled(c *gin.Context) {
	posts, err := h.service.GetEnabledWithCache(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, posts)
}
