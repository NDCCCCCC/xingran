package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ==================== 字典类型管理 ====================

// DictTypeListRequest 字典类型列表请求
type DictTypeListRequest struct {
	DictName *string `json:"dictName,omitempty"`
	DictType *string `json:"dictType,omitempty"`
	Status   *int    `json:"status,omitempty"`
	Current  int     `json:"current" binding:"min=1"`
	PageSize int     `json:"pageSize" binding:"min=1,max=100"`
}

// DictTypeCreateRequest 创建字典类型请求
type DictTypeCreateRequest struct {
	DictName string  `json:"dictName" binding:"required,min=2,max=100"`
	DictType string  `json:"dictType" binding:"required,min=2,max=100"`
	Status   int     `json:"status" binding:"min=0,max=1,oneof=0 1"`
	Remark   *string `json:"remark,omitempty"`
}

// DictTypeUpdateRequest 更新字典类型请求
type DictTypeUpdateRequest struct {
	ID       string  `json:"id,omitempty"`
	DictName string  `json:"dictName" binding:"required,min=2,max=100"`
	Status   int     `json:"status" binding:"min=0,max=1,oneof=0 1"`
	Remark   *string `json:"remark,omitempty"`
}

// listDictTypes 查询字典类型列表
// @Summary 查询字典类型列表
// @Description 分页查询字典类型列表
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param request body DictTypeListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /dicts/types/list [post]
func listDictTypes(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			rawReq = make(map[string]interface{})
		}

		req := DictTypeListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		}

		if val, ok := rawReq["dictName"].(string); ok && val != "" {
			req.DictName = &val
		}
		if val, ok := rawReq["dictType"].(string); ok && val != "" {
			req.DictType = &val
		}
		if statusVal, ok := rawReq["status"]; ok && statusVal != nil {
			var status int
			switch v := statusVal.(type) {
			case string:
				if v == "0" || v == "1" {
					status = parseInt(v)
				}
			case float64:
				status = int(v)
			case int:
				status = v
			}
			req.Status = &status
		}

		db := core.DB.GetDB().Model(&models.DictType{})

		if req.DictName != nil && *req.DictName != "" {
			db = db.Where("dict_name LIKE ?", "%"+*req.DictName+"%")
		}
		if req.DictType != nil && *req.DictType != "" {
			db = db.Where("dict_type LIKE ?", "%"+*req.DictType+"%")
		}
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
		}

		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询字典类型总数失败")
			return
		}

		var dictTypes []models.DictType
		offset := (req.Current - 1) * req.PageSize
		if err := db.Offset(offset).Limit(req.PageSize).Find(&dictTypes).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询字典类型列表失败")
			return
		}

		pageResp := response.PageResponse{
			List:     dictTypes,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// createDictType 创建字典类型
// @Summary 创建字典类型
// @Description 创建新字典类型
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param request body DictTypeCreateRequest true "字典类型信息"
// @Success 200 {object} response.Response
// @Router /dicts/types [post]
func createDictType(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DictTypeCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查字典类型是否已存在
		var existDict models.DictType
		if err := core.DB.GetDB().Where("dict_type = ?", req.DictType).First(&existDict).Error; err == nil {
			response.Error(c, response.ErrBadRequest, "字典类型已存在")
			return
		}

		dictType := models.DictType{
			DictName: req.DictName,
			DictType: req.DictType,
			Status:   req.Status,
			Remark:   func() string { if req.Remark != nil { return *req.Remark }; return "" }(),
		}

		if err := core.DB.GetDB().Create(&dictType).Error; err != nil {
			response.Error(c, response.ErrServerError, "创建字典类型失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"字典管理",
			1, // 1=新增
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"id": dictType.ID})
	}
}

// getDictType 获取字典类型详情
// @Summary 获取字典类型详情
// @Description 根据ID获取字典类型详情
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param id path string true "字典类型ID"
// @Success 200 {object} response.Response{data=models.DictType}
// @Router /dicts/types/:id [post]
func getDictType(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var dictType models.DictType
		if err := core.DB.GetDB().Where("id = ?", id).First(&dictType).Error; err != nil {
			response.Error(c, response.ErrNotFound, "字典类型不存在")
			return
		}

		response.Success(c, dictType)
	}
}

// updateDictType 更新字典类型
// @Summary 更新字典类型
// @Description 更新字典类型信息
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param id path string true "字典类型ID"
// @Param request body DictTypeUpdateRequest true "字典类型信息"
// @Success 200 {object} response.Response
// @Router /dicts/types/:id/update [post]
func updateDictType(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req DictTypeUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		var dictType models.DictType
		if err := core.DB.GetDB().Where("id = ?", id).First(&dictType).Error; err != nil {
			response.Error(c, response.ErrNotFound, "字典类型不存在")
			return
		}

		updates := map[string]interface{}{
			"dict_name": req.DictName,
			"status":    req.Status,
		}
		if req.Remark != nil {
			updates["remark"] = req.Remark
		}

		if err := core.DB.GetDB().Model(&dictType).Updates(updates).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新字典类型失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"字典管理",
			2, // 2=修改
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, nil)
	}
}

// deleteDictType 删除字典类型
// @Summary 删除字典类型
// @Description 删除指定字典类型
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param id path string true "字典类型ID"
// @Success 200 {object} response.Response
// @Router /dicts/types/:id/delete [post]
func deleteDictType(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var dictType models.DictType
		if err := core.DB.GetDB().Where("id = ?", id).First(&dictType).Error; err != nil {
			response.Error(c, response.ErrNotFound, "字典类型不存在")
			return
		}

		// 检查是否有字典数据
		var count int64
		if err := core.DB.GetDB().Model(&models.DictData{}).Where("dict_type = ?", dictType.DictType).Count(&count).Error; err == nil && count > 0 {
			response.Error(c, response.ErrBadRequest, "该字典类型下存在字典数据，不能删除")
			return
		}

		if err := core.DB.GetDB().Delete(&dictType).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除字典类型失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"字典管理",
			3, // 3=删除
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, nil)
	}
}

// batchDeleteDictTypes 批量删除字典类型
// @Summary 批量删除字典类型
// @Description 批量删除多个字典类型
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param request body map[string][]string true "ID列表"
// @Success 200 {object} response.Response
// @Router /dicts/types/batch-delete [post]
func batchDeleteDictTypes(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Ids []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if err := core.DB.GetDB().Where("id IN ?", req.Ids).Delete(&models.DictType{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "批量删除字典类型失败")
			return
		}

		response.Success(c, nil)
	}
}

// refreshDictCache 刷新字典缓存
// @Summary 刷新字典缓存
// @Description 刷新系统字典缓存
// @Tags 字典管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /dicts/refresh-cache [post]
func refreshDictCache(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现缓存刷新逻辑
		response.Success(c, gin.H{"message": "刷新缓存成功"})
	}
}

// ==================== 字典数据管理 ====================

// DictDataListRequest 字典数据列表请求
type DictDataListRequest struct {
	DictType  string  `json:"dictType" binding:"required"`
	DictLabel *string `json:"dictLabel,omitempty"`
	Status    *int    `json:"status,omitempty"`
	Current   int     `json:"current" binding:"min=1"`
	PageSize  int     `json:"pageSize" binding:"min=1,max=100"`
}

// DictDataCreateRequest 创建字典数据请求
type DictDataCreateRequest struct {
	DictSort   int     `json:"dictSort" binding:"min=0"`
	DictLabel  string  `json:"dictLabel" binding:"required,min=1,max=100"`
	DictValue  string  `json:"dictValue" binding:"required,min=1,max=100"`
	DictType   string  `json:"dictType" binding:"required,min=1,max=100"`
	CssClass   *string `json:"cssClass,omitempty"`
	ListClass  *string `json:"listClass,omitempty"`
	IsDefault  bool    `json:"isDefault"`
	Status     int     `json:"status" binding:"min=0,max=1,oneof=0 1"`
	Remark     *string `json:"remark,omitempty"`
}

// DictDataUpdateRequest 更新字典数据请求
type DictDataUpdateRequest struct {
	ID         string  `json:"id,omitempty"`
	DictSort   int     `json:"dictSort" binding:"min=0"`
	DictLabel  string  `json:"dictLabel" binding:"required,min=1,max=100"`
	DictValue  string  `json:"dictValue" binding:"required,min=1,max=100"`
	CssClass   *string `json:"cssClass,omitempty"`
	ListClass  *string `json:"listClass,omitempty"`
	IsDefault  bool    `json:"isDefault"`
	Status     int     `json:"status" binding:"min=0,max=1,oneof=0 1"`
	Remark     *string `json:"remark,omitempty"`
}

// listDictData 查询字典数据列表
// @Summary 查询字典数据列表
// @Description 根据字典类型查询字典数据列表
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param request body DictDataListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /dicts/data/list [post]
func listDictData(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		req := DictDataListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		}

		if val, ok := rawReq["dictType"].(string); ok {
			req.DictType = val
		}
		if val, ok := rawReq["dictLabel"].(string); ok && val != "" {
			req.DictLabel = &val
		}
		if statusVal, ok := rawReq["status"]; ok && statusVal != nil {
			var status int
			switch v := statusVal.(type) {
			case string:
				if v == "0" || v == "1" {
					status = parseInt(v)
				}
			case float64:
				status = int(v)
			case int:
				status = v
			}
			req.Status = &status
		}

		db := core.DB.GetDB().Model(&models.DictData{}).Where("dict_type = ?", req.DictType)

		if req.DictLabel != nil && *req.DictLabel != "" {
			db = db.Where("dict_label LIKE ?", "%"+*req.DictLabel+"%")
		}
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
		}

		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询字典数据总数失败")
			return
		}

		var dictData []models.DictData
		offset := (req.Current - 1) * req.PageSize
		if err := db.Order("dict_sort ASC").Offset(offset).Limit(req.PageSize).Find(&dictData).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询字典数据列表失败")
			return
		}

		pageResp := response.PageResponse{
			List:     dictData,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// getDictDataByType 根据字典类型查询字典数据
// @Summary 根据字典类型查询字典数据
// @Description 获取指定类型的所有字典数据
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param dictType query string true "字典类型"
// @Success 200 {object} response.Response{data=[]models.DictData}
// @Router /dicts/data/type/:dictType [get]
func getDictDataByType(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		dictType := c.Param("dictType")
		if dictType == "" {
			response.Error(c, response.ErrBadRequest, "字典类型不能为空")
			return
		}

		var dictData []models.DictData
		if err := core.DB.GetDB().Where("dict_type = ? AND status = 0", dictType).Order("dict_sort ASC").Find(&dictData).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询字典数据失败")
			return
		}

		response.Success(c, dictData)
	}
}

// createDictData 创建字典数据
// @Summary 创建字典数据
// @Description 创建新字典数据
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param request body DictDataCreateRequest true "字典数据信息"
// @Success 200 {object} response.Response
// @Router /dicts/data [post]
func createDictData(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DictDataCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查字典类型是否存在
		var dictType models.DictType
		if err := core.DB.GetDB().Where("dict_type = ?", req.DictType).First(&dictType).Error; err != nil {
			response.Error(c, response.ErrBadRequest, "字典类型不存在")
			return
		}

		dictData := models.DictData{
			DictSort:   req.DictSort,
			DictLabel:  req.DictLabel,
			DictValue:  req.DictValue,
			DictType:   req.DictType,
			CssClass:   req.CssClass,
			ListClass:  req.ListClass,
			IsDefault:  req.IsDefault,
			Status:     req.Status,
			Remark:     func() string { if req.Remark != nil { return *req.Remark }; return "" }(),
		}

		if err := core.DB.GetDB().Create(&dictData).Error; err != nil {
			response.Error(c, response.ErrServerError, "创建字典数据失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"字典管理",
			1, // 1=新增
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"id": dictData.ID})
	}
}

// getDictData 获取字典数据详情
// @Summary 获取字典数据详情
// @Description 根据ID获取字典数据详情
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param id path string true "字典数据ID"
// @Success 200 {object} response.Response{data=models.DictData}
// @Router /dicts/data/:id [post]
func getDictData(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var dictData models.DictData
		if err := core.DB.GetDB().Where("id = ?", id).First(&dictData).Error; err != nil {
			response.Error(c, response.ErrNotFound, "字典数据不存在")
			return
		}

		response.Success(c, dictData)
	}
}

// updateDictData 更新字典数据
// @Summary 更新字典数据
// @Description 更新字典数据信息
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param id path string true "字典数据ID"
// @Param request body DictDataUpdateRequest true "字典数据信息"
// @Success 200 {object} response.Response
// @Router /dicts/data/:id/update [post]
func updateDictData(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req DictDataUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		var dictData models.DictData
		if err := core.DB.GetDB().Where("id = ?", id).First(&dictData).Error; err != nil {
			response.Error(c, response.ErrNotFound, "字典数据不存在")
			return
		}

		updates := map[string]interface{}{
			"dict_sort":  req.DictSort,
			"dict_label": req.DictLabel,
			"dict_value": req.DictValue,
			"is_default": req.IsDefault,
			"status":     req.Status,
		}
		if req.CssClass != nil {
			updates["css_class"] = req.CssClass
		}
		if req.ListClass != nil {
			updates["list_class"] = req.ListClass
		}
		if req.Remark != nil {
			updates["remark"] = req.Remark
		}

		if err := core.DB.GetDB().Model(&dictData).Updates(updates).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新字典数据失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"字典管理",
			2, // 2=修改
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, nil)
	}
}

// deleteDictData 删除字典数据
// @Summary 删除字典数据
// @Description 删除指定字典数据
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param id path string true "字典数据ID"
// @Success 200 {object} response.Response
// @Router /dicts/data/:id/delete [post]
func deleteDictData(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var dictData models.DictData
		if err := core.DB.GetDB().Where("id = ?", id).First(&dictData).Error; err != nil {
			response.Error(c, response.ErrNotFound, "字典数据不存在")
			return
		}

		if err := core.DB.GetDB().Delete(&dictData).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除字典数据失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"字典管理",
			3, // 3=删除
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, nil)
	}
}

// batchDeleteDictData 批量删除字典数据
// @Summary 批量删除字典数据
// @Description 批量删除多个字典数据
// @Tags 字典管理
// @Accept json
// @Produce json
// @Param request body map[string][]string true "ID列表"
// @Success 200 {object} response.Response
// @Router /dicts/data/batch-delete [post]
func batchDeleteDictData(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Ids []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if err := core.DB.GetDB().Where("id IN ?", req.Ids).Delete(&models.DictData{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "批量删除字典数据失败")
			return
		}

		response.Success(c, nil)
	}
}
