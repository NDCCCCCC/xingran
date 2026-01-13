package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// WorkstationListRequest 工位列表请求
type WorkstationListRequest struct {
	WorkstationCode *string                   `json:"workstationCode,omitempty"`
	WorkstationName *string                   `json:"workstationName,omitempty"`
	DeptID          *string                   `json:"deptId,omitempty"`
	WorkstationType *models.WorkstationType   `json:"workstationType,omitempty"`
	Status          *models.WorkstationStatus `json:"status,omitempty"`
	Current         int                       `json:"current" binding:"min=1"`
	PageSize        int                       `json:"pageSize" binding:"min=1,max=100"`
}

// WorkstationCreateRequest 创建工位请求
type WorkstationCreateRequest struct {
	WorkstationCode string                  `json:"workstationCode" binding:"required,min=2,max=50"`
	WorkstationName string                  `json:"workstationName" binding:"required,min=2,max=100"`
	DeptID          *string                 `json:"deptId,omitempty"`
	Location        *string                 `json:"location,omitempty"`
	Floor           *string                 `json:"floor,omitempty"`
	WorkstationType models.WorkstationType  `json:"workstationType" binding:"min=0,max=2"`
	Status          models.WorkstationStatus `json:"status" binding:"min=0,max=1,oneof=0 1"`
	Capacity        int                     `json:"capacity" binding:"min=1"`
	Description     *string                 `json:"description,omitempty"`
}

// WorkstationUpdateRequest 更新工位请求
type WorkstationUpdateRequest struct {
	ID              string                   `json:"id,omitempty"`
	WorkstationName string                   `json:"workstationName" binding:"required,min=2,max=100"`
	DeptID          *string                  `json:"deptId,omitempty"`
	Location        *string                  `json:"location,omitempty"`
	Floor           *string                  `json:"floor,omitempty"`
	WorkstationType models.WorkstationType   `json:"workstationType" binding:"min=0,max=2"`
	Status          models.WorkstationStatus `json:"status" binding:"min=0,max=1,oneof=0 1"`
	Capacity        int                      `json:"capacity" binding:"min=1"`
	Description     *string                  `json:"description,omitempty"`
	UserID          *string                  `json:"userId,omitempty"`
}

// listWorkstations 查询工位列表
// @Summary 查询工位列表
// @Description 分页查询工位列表
// @Tags 工位管理
// @Accept json
// @Produce json
// @Param request body WorkstationListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /workstations/list [post]
func listWorkstations(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			rawReq = make(map[string]interface{})
		}

		req := WorkstationListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		}

		if val, ok := rawReq["workstationCode"].(string); ok && val != "" {
			req.WorkstationCode = &val
		}
		if val, ok := rawReq["workstationName"].(string); ok && val != "" {
			req.WorkstationName = &val
		}
		if val, ok := rawReq["deptId"].(string); ok && val != "" {
			req.DeptID = &val
		}
		if val, ok := rawReq["workstationType"]; ok && val != nil {
			switch v := val.(type) {
			case float64:
				wt := models.WorkstationType(int(v))
				req.WorkstationType = &wt
			case int:
				wt := models.WorkstationType(v)
				req.WorkstationType = &wt
			}
		}
		if val, ok := rawReq["status"]; ok && val != nil {
			switch v := val.(type) {
			case float64:
				s := models.WorkstationStatus(int(v))
				req.Status = &s
			case int:
				s := models.WorkstationStatus(v)
				req.Status = &s
			}
		}

		db := core.DB.GetDB().Model(&models.Workstation{})

		if req.WorkstationCode != nil && *req.WorkstationCode != "" {
			db = db.Where("workstation_code LIKE ?", "%"+*req.WorkstationCode+"%")
		}
		if req.WorkstationName != nil && *req.WorkstationName != "" {
			db = db.Where("workstation_name LIKE ?", "%"+*req.WorkstationName+"%")
		}
		if req.DeptID != nil && *req.DeptID != "" {
			db = db.Where("dept_id = ?", *req.DeptID)
		}
		if req.WorkstationType != nil {
			db = db.Where("workstation_type = ?", *req.WorkstationType)
		}
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
		}

		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询工位总数失败")
			return
		}

		var workstations []models.Workstation
		offset := (req.Current - 1) * req.PageSize
		if err := db.Offset(offset).Limit(req.PageSize).Find(&workstations).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询工位列表失败")
			return
		}

		// 加载部门名称
		var deptIDs []string
		for _, ws := range workstations {
			if ws.DeptID != nil && *ws.DeptID != "" {
				deptIDs = append(deptIDs, *ws.DeptID)
			}
		}
		var depts []models.Department
		if len(deptIDs) > 0 {
			core.DB.GetDB().Where("id IN ?", deptIDs).Find(&depts)
		}
		deptMap := make(map[string]string)
		for _, dept := range depts {
			deptMap[dept.ID] = dept.DeptName
		}
		for i := range workstations {
			if workstations[i].DeptID != nil {
				if deptName, ok := deptMap[*workstations[i].DeptID]; ok {
					workstations[i].DeptName = &deptName
				}
			}
		}

		pageResp := response.PageResponse{
			List:     workstations,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// createWorkstation 创建工位
// @Summary 创建工位
// @Description 创建新工位
// @Tags 工位管理
// @Accept json
// @Produce json
// @Param request body WorkstationCreateRequest true "工位信息"
// @Success 200 {object} response.Response
// @Router /workstations [post]
func createWorkstation(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req WorkstationCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查工位编码是否已存在
		var existWS models.Workstation
		if err := core.DB.GetDB().Where("workstation_code = ?", req.WorkstationCode).First(&existWS).Error; err == nil {
			response.Error(c, response.ErrBadRequest, "工位编码已存在")
			return
		}

		// 验证部门ID是否存在
		var deptName *string
		if req.DeptID != nil && *req.DeptID != "" {
			var dept models.Department
			if err := core.DB.GetDB().Where("id = ?", *req.DeptID).First(&dept).Error; err != nil {
				response.Error(c, response.ErrBadRequest, "部门不存在")
				return
			}
			deptName = &dept.DeptName
		}

		workstation := models.Workstation{
			WorkstationCode: req.WorkstationCode,
			WorkstationName: req.WorkstationName,
			DeptID:          req.DeptID,
			DeptName:        deptName,
			Location:        req.Location,
			Floor:           req.Floor,
			WorkstationType: req.WorkstationType,
			Status:          req.Status,
			Capacity:        req.Capacity,
			Description:     req.Description,
		}

		if err := core.DB.GetDB().Create(&workstation).Error; err != nil {
			response.Error(c, response.ErrServerError, "创建工位失败")
			return
		}

		response.Success(c, gin.H{"id": workstation.ID})
	}
}

// getWorkstation 获取工位详情
// @Summary 获取工位详情
// @Description 根据ID获取工位详情
// @Tags 工位管理
// @Accept json
// @Produce json
// @Param id path string true "工位ID"
// @Success 200 {object} response.Response{data=models.Workstation}
// @Router /workstations/:id [post]
func getWorkstation(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var workstation models.Workstation
		if err := core.DB.GetDB().Where("id = ?", id).First(&workstation).Error; err != nil {
			response.Error(c, response.ErrNotFound, "工位不存在")
			return
		}

		response.Success(c, workstation)
	}
}

// updateWorkstation 更新工位
// @Summary 更新工位
// @Description 更新工位信息
// @Tags 工位管理
// @Accept json
// @Produce json
// @Param id path string true "工位ID"
// @Param request body WorkstationUpdateRequest true "工位信息"
// @Success 200 {object} response.Response
// @Router /workstations/:id/update [post]
func updateWorkstation(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req WorkstationUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		var workstation models.Workstation
		if err := core.DB.GetDB().Where("id = ?", id).First(&workstation).Error; err != nil {
			response.Error(c, response.ErrNotFound, "工位不存在")
			return
		}

		// 验证部门ID是否存在
		var deptName *string
		if req.DeptID != nil && *req.DeptID != "" {
			var dept models.Department
			if err := core.DB.GetDB().Where("id = ?", *req.DeptID).First(&dept).Error; err != nil {
				response.Error(c, response.ErrBadRequest, "部门不存在")
				return
			}
			deptName = &dept.DeptName
		}

		updates := map[string]interface{}{
			"workstation_name": req.WorkstationName,
			"workstation_type": req.WorkstationType,
			"status":           req.Status,
			"capacity":         req.Capacity,
		}

		if req.DeptID != nil {
			updates["dept_id"] = req.DeptID
			updates["dept_name"] = deptName
		}
		if req.Location != nil {
			updates["location"] = req.Location
		}
		if req.Floor != nil {
			updates["floor"] = req.Floor
		}
		if req.Description != nil {
			updates["description"] = req.Description
		}
		if req.UserID != nil {
			updates["user_id"] = req.UserID
		}

		if err := core.DB.GetDB().Model(&workstation).Updates(updates).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新工位失败")
			return
		}

		response.Success(c, nil)
	}
}

// deleteWorkstation 删除工位
// @Summary 删除工位
// @Description 删除指定工位
// @Tags 工位管理
// @Accept json
// @Produce json
// @Param id path string true "工位ID"
// @Success 200 {object} response.Response
// @Router /workstations/:id/delete [post]
func deleteWorkstation(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var workstation models.Workstation
		if err := core.DB.GetDB().Where("id = ?", id).First(&workstation).Error; err != nil {
			response.Error(c, response.ErrNotFound, "工位不存在")
			return
		}

		if err := core.DB.GetDB().Delete(&workstation).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除工位失败")
			return
		}

		response.Success(c, nil)
	}
}

// batchDeleteWorkstations 批量删除工位
// @Summary 批量删除工位
// @Description 批量删除多个工位
// @Tags 工位管理
// @Accept json
// @Produce json
// @Param request body map[string][]string true "ID列表"
// @Success 200 {object} response.Response
// @Router /workstations/batch-delete [post]
func batchDeleteWorkstations(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Ids []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if err := core.DB.GetDB().Where("id IN ?", req.Ids).Delete(&models.Workstation{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "批量删除工位失败")
			return
		}

		response.Success(c, nil)
	}
}

// getAvailableWorkstations 获取可用工位列表
// @Summary 获取可用工位列表
// @Description 获取当前可用的工位列表（未分配用户的）
// @Tags 工位管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Workstation}
// @Router /workstations/available [get]
func getAvailableWorkstations(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var workstations []models.Workstation
		if err := core.DB.GetDB().Where("status = ? AND (user_id IS NULL OR user_id = '')", models.WorkstationStatusEnabled).Find(&workstations).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询可用工位失败")
			return
		}

		response.Success(c, workstations)
	}
}

// assignUser 分配用户到工位
// @Summary 分配用户到工位
// @Description 将用户分配到指定工位
// @Tags 工位管理
// @Accept json
// @Produce json
// @Param id path string true "工位ID"
// @Param request body map[string]string true "用户ID"
// @Success 200 {object} response.Response
// @Router /workstations/:id/assign [post]
func assignUser(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工位ID不能为空")
			return
		}

		var req struct {
			UserID string `json:"userId" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		var workstation models.Workstation
		if err := core.DB.GetDB().Where("id = ?", id).First(&workstation).Error; err != nil {
			response.Error(c, response.ErrNotFound, "工位不存在")
			return
		}

		// 获取用户信息
		var user models.User
		if err := core.DB.GetDB().Where("id = ?", req.UserID).First(&user).Error; err != nil {
			response.Error(c, response.ErrBadRequest, "用户不存在")
			return
		}

		// 检查工位是否已分配
		if workstation.UserID != nil && *workstation.UserID != "" {
			response.Error(c, response.ErrBadRequest, "工位已被占用")
			return
		}

		userID := req.UserID
		userName := user.Username

		if err := core.DB.GetDB().Model(&workstation).Updates(map[string]interface{}{
			"user_id":   userID,
			"user_name": userName,
		}).Error; err != nil {
			response.Error(c, response.ErrServerError, "分配工位失败")
			return
		}

		response.Success(c, nil)
	}
}

// unassignUser 取消工位用户分配
// @Summary 取消工位用户分配
// @Description 取消工位的用户分配
// @Tags 工位管理
// @Accept json
// @Produce json
// @Param id path string true "工位ID"
// @Success 200 {object} response.Response
// @Router /workstations/:id/unassign [post]
func unassignUser(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工位ID不能为空")
			return
		}

		var workstation models.Workstation
		if err := core.DB.GetDB().Where("id = ?", id).First(&workstation).Error; err != nil {
			response.Error(c, response.ErrNotFound, "工位不存在")
			return
		}

		if err := core.DB.GetDB().Model(&workstation).Updates(map[string]interface{}{
			"user_id":   nil,
			"user_name": nil,
		}).Error; err != nil {
			response.Error(c, response.ErrServerError, "取消分配失败")
			return
		}

		response.Success(c, nil)
	}
}
