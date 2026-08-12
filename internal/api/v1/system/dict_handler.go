package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// DictTypeHandler 字典类型处理器
type DictTypeHandler struct {
	service systemServices.DictTypeService
	core    *core.Core
}

// NewDictTypeHandler 创建字典类型处理器实例
func NewDictTypeHandler(service systemServices.DictTypeService) *DictTypeHandler {
	return &DictTypeHandler{service: service}
}

// WithCore 注入 core 依赖（Phase 34 操作日志记录所需）。
func (h *DictTypeHandler) WithCore(core *core.Core) *DictTypeHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Statistics 字典类型统计(读操作,不记操作日志)
func (h *DictTypeHandler) Statistics(c *gin.Context) {
	result, err := h.service.Statistics(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Create 创建字典类型
// @Summary 创建字典类型
// @Description 创建新的字典类型
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param request body object{dictName=string,dictType=string,status=int,remark=string} true "字典类型信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/dict-types [post]
func (h *DictTypeHandler) Create(c *gin.Context) {
	var req requests.DictTypeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "字典类型", operlog.OperTypeCreate)
	response.Success(c, gin.H{"message": "创建成功"})
}

// List 查询字典类型列表
// @Summary 查询字典类型列表
// @Description 查询字典类型列表，支持多条件筛选和分页
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param request body object{dictName=string,dictType=string,status=int,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/dict-types/list [post]
func (h *DictTypeHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	params := requests.DefaultDictTypeListParams()

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
	if val, ok := rawReq["dictName"].(string); ok && val != "" {
		params.DictName = &val
	}
	if val, ok := rawReq["dictType"].(string); ok && val != "" {
		params.DictType = &val
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

	// 服务端排序参数（透传给 service.List → base.ApplySort 白名单）
	if val, ok := rawReq["orderByColumn"].(string); ok && val != "" {
		params.OrderByColumn = val
	}
	if val, ok := rawReq["isAsc"]; ok {
		if b, ok := val.(bool); ok {
			params.IsAsc = &b
		}
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取字典类型详情
// @Summary 获取字典类型详情
// @Description 根据字典类型ID获取详细信息
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param id path string true "字典类型ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/dict-types/:id [post]
func (h *DictTypeHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("字典类型ID"))
		return
	}

	dictType, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dictType)
}

// Update 更新字典类型
// @Summary 更新字典类型
// @Description 更新字典类型信息
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param id path string true "字典类型ID"
// @Param request body object{dictName=string,dictType=string,status=int,remark=string} true "字典类型信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/dict-types/:id/update [post]
func (h *DictTypeHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("字典类型ID"))
		return
	}

	var req requests.DictTypeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	req.ID = id
	if err := h.service.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "字典类型", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除字典类型
// @Summary 删除字典类型
// @Description 删除指定的字典类型
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param id path string true "字典类型ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/dict-types/:id/delete [post]
func (h *DictTypeHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("字典类型ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "字典类型", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// GetAll 获取所有字典类型（使用缓存）
// @Summary 获取所有字典类型
// @Description 获取所有字典类型列表（使用缓存）
// @Tags 字典管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/dict-types/all [post]
func (h *DictTypeHandler) GetAll(c *gin.Context) {
	dictTypes, err := h.service.GetAllWithCache(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dictTypes)
}

// DictDataHandler 字典数据处理器
type DictDataHandler struct {
	service systemServices.DictDataService
	core    *core.Core
}

// NewDictDataHandler 创建字典数据处理器实例
func NewDictDataHandler(service systemServices.DictDataService) *DictDataHandler {
	return &DictDataHandler{service: service}
}

// WithCore 注入 core 依赖（Phase 34 操作日志记录所需）。
func (h *DictDataHandler) WithCore(core *core.Core) *DictDataHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Statistics 字典数据统计(读操作,不记操作日志;支持按 dictType 过滤)
func (h *DictDataHandler) Statistics(c *gin.Context) {
	var req struct {
		DictType string `json:"dictType"`
	}
	_ = c.ShouldBindJSON(&req)
	result, err := h.service.Statistics(c.Request.Context(), req.DictType)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Create 创建字典数据
// @Summary 创建字典数据
// @Description 创建新的字典数据
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param request body object{dictType=string,dictLabel=string,dictValue=string,dictSort=int,status=int,remark=string} true "字典数据信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/dict-data [post]
func (h *DictDataHandler) Create(c *gin.Context) {
	var req requests.DictDataCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "字典数据", operlog.OperTypeCreate)
	response.Success(c, gin.H{"message": "创建成功"})
}

// List 查询字典数据列表
// @Summary 查询字典数据列表
// @Description 查询字典数据列表，支持多条件筛选和分页
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param request body object{dictType=string,dictLabel=string,status=int,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/dict-data/list [post]
func (h *DictDataHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		response.Error(c, apperrors.ParamError())
		return
	}

	req := requests.DictDataListParams{
		BaseListRequest: base.BaseListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		},
	}

	if val, ok := rawReq["dictType"].(string); ok {
		req.DictType = val
	}
	if val, ok := rawReq["dictLabel"].(string); ok && val != "" {
		req.DictLabel = &val
	}
	if val, ok := rawReq["status"]; ok && val != nil {
		switch v := val.(type) {
		case string:
			if v == "0" || v == "1" {
				status := parseInt(v)
				req.Status = &status
			}
		case float64:
			status := int(v)
			req.Status = &status
		case int:
			req.Status = &v
		}
	}

	// 服务端排序参数（透传给 service.List → base.ApplySort 白名单）
	if val, ok := rawReq["orderByColumn"].(string); ok && val != "" {
		req.OrderByColumn = val
	}
	if val, ok := rawReq["isAsc"]; ok {
		if b, ok := val.(bool); ok {
			req.IsAsc = &b
		}
	}

	result, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取字典数据详情
// @Summary 获取字典数据详情
// @Description 根据字典数据ID获取详细信息
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param id path string true "字典数据ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/dict-data/:id [post]
func (h *DictDataHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("字典数据ID"))
		return
	}

	dictData, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dictData)
}

// Update 更新字典数据
// @Summary 更新字典数据
// @Description 更新字典数据信息
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param id path string true "字典数据ID"
// @Param request body object{dictType=string,dictLabel=string,dictValue=string,dictSort=int,status=int,remark=string} true "字典数据信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/dict-data/:id/update [post]
func (h *DictDataHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("字典数据ID"))
		return
	}

	var req requests.DictDataUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	req.ID = id
	if err := h.service.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "字典数据", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除字典数据
// @Summary 删除字典数据
// @Description 删除指定的字典数据
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param id path string true "字典数据ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/dict-data/:id/delete [post]
func (h *DictDataHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("字典数据ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "字典数据", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// ==================== 辅助函数 ====================

// getIntField 从map中获取整数字段，提供默认值
func getIntField(m map[string]interface{}, key string, defaultValue int) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			return parseInt(v)
		}
	}
	return defaultValue
}
