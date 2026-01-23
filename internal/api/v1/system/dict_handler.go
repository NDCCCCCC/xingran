package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// DictTypeHandler 字典类型处理器
type DictTypeHandler struct {
	service systemServices.DictTypeService
}

// NewDictTypeHandler 创建字典类型处理器实例
func NewDictTypeHandler(service systemServices.DictTypeService) *DictTypeHandler {
	return &DictTypeHandler{service: service}
}

// Create 创建字典类型
func (h *DictTypeHandler) Create(c *gin.Context) {
	var req systemServices.DictTypeCreateRequest
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

// List 查询字典类型列表
func (h *DictTypeHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	params := systemServices.DefaultDictTypeListParams()

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

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取字典类型详情
func (h *DictTypeHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "字典类型ID不能为空")
		return
	}

	dictType, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, dictType)
}

// Update 更新字典类型
func (h *DictTypeHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "字典类型ID不能为空")
		return
	}

	var req systemServices.DictTypeUpdateRequest
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

// Delete 删除字典类型
func (h *DictTypeHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "字典类型ID不能为空")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetAll 获取所有字典类型（使用缓存）
func (h *DictTypeHandler) GetAll(c *gin.Context) {
	dictTypes, err := h.service.GetAllWithCache(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, dictTypes)
}

// DictDataHandler 字典数据处理器
type DictDataHandler struct {
	service systemServices.DictDataService
}

// NewDictDataHandler 创建字典数据处理器实例
func NewDictDataHandler(service systemServices.DictDataService) *DictDataHandler {
	return &DictDataHandler{service: service}
}

// Create 创建字典数据
func (h *DictDataHandler) Create(c *gin.Context) {
	var req systemServices.DictDataCreateRequest
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

// List 查询字典数据列表
func (h *DictDataHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	req := systemServices.DictDataListParams{
		Current:  getIntField(rawReq, "current", 1),
		PageSize: getIntField(rawReq, "pageSize", 10),
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

	result, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取字典数据详情
func (h *DictDataHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "字典数据ID不能为空")
		return
	}

	dictData, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, dictData)
}

// Update 更新字典数据
func (h *DictDataHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "字典数据ID不能为空")
		return
	}

	var req systemServices.DictDataUpdateRequest
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

// Delete 删除字典数据
func (h *DictDataHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "字典数据ID不能为空")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

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
