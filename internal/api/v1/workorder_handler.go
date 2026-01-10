package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/scheduler"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ==================== 工单管理 Handler ====================

// listWorkOrders 查询工单列表
func listWorkOrders(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.WorkOrderListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		list, total, err := workorderSvc.GetWorkOrderList(c.Request.Context(), &req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		pageResp := response.PageResponse{
			List:     list,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}
		response.Success(c, pageResp)
	}
}

// getMyPendingWorkOrders 获取当前用户的待办工单
func getMyPendingWorkOrders(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.GetMyPendingWorkOrdersRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体，使用默认值
			req.Limit = 5
		}

		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "用户未登录")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		list, total, err := workorderSvc.GetMyPendingWorkOrders(c.Request.Context(), &req, userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"list":  list,
			"total": total,
		})
	}
}

// getWorkOrder 获取工单详情
func getWorkOrder(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		workOrder, err := workorderSvc.GetWorkOrderByID(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		response.Success(c, workOrder)
	}
}

// createWorkOrder 创建工单
func createWorkOrder(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.WorkOrderCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		workOrder, err := workorderSvc.CreateWorkOrder(c.Request.Context(), &req, userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单管理",
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

		response.Success(c, workOrder)
	}
}

// updateWorkOrder 更新工单
func updateWorkOrder(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		var req services.WorkOrderUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		req.ID = id
		userID, _ := c.Get("user_id")

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.UpdateWorkOrder(c.Request.Context(), &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单管理",
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

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteWorkOrder 删除工单
func deleteWorkOrder(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.DeleteWorkOrder(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单管理",
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

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// batchDeleteWorkOrders 批量删除工单
func batchDeleteWorkOrders(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			IDs []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.BatchDeleteWorkOrders(c.Request.Context(), req.IDs); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单管理",
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

		response.Success(c, gin.H{
			"message": "批量删除成功",
			"count":   len(req.IDs),
		})
	}
}

// assignWorkOrder 分配工单
func assignWorkOrder(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		var req services.AssignWorkOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.AssignWorkOrder(c.Request.Context(), id, &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单管理",
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

		response.Success(c, gin.H{"message": "分配成功"})
	}
}

// assignToTodayDuty 分配给当天值班人员
func assignToTodayDuty(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.AssignToTodayDuty(c.Request.Context(), id, userID.(string)); err != nil {
			// 当没有值班人员时，静默处理，不显示错误
			if err.Error() == "今天没有值班人员" {
				response.Success(c, gin.H{"message": "今天没有值班人员，工单未分配"})
				return
			}
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单管理",
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

		response.Success(c, gin.H{"message": "已分配给当天值班人员"})
	}
}

// updateWorkOrderStatus 更新工单状态
func updateWorkOrderStatus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		var req services.UpdateStatusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.UpdateWorkOrderStatus(c.Request.Context(), id, &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单管理",
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

		response.Success(c, gin.H{"message": "状态更新成功"})
	}
}

// ==================== 工单评论 Handler ====================

// getWorkOrderComments 获取工单评论列表
func getWorkOrderComments(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		comments, err := workorderSvc.GetWorkOrderComments(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, comments)
	}
}

// addWorkOrderComment 添加工单评论
func addWorkOrderComment(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		var req services.AddCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.AddWorkOrderComment(c.Request.Context(), id, &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单管理",
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

		response.Success(c, gin.H{"message": "评论添加成功"})
	}
}

// getWorkOrderHistory 获取工单操作历史
func getWorkOrderHistory(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		history, err := workorderSvc.GetWorkOrderHistory(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, history)
	}
}

// ==================== 工单统计 Handler ====================

// getWorkOrderStatistics 获取工单统计数据
func getWorkOrderStatistics(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		stats, err := workorderSvc.GetWorkOrderStatistics(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, stats)
	}
}

// ==================== 工单分类 Handler ====================

// listWorkOrderCategories 查询工单分类列表
func listWorkOrderCategories(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		categories, err := workorderSvc.GetWorkOrderCategoryList(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, categories)
	}
}

// getEnabledCategories 获取启用的工单分类（用于下拉选择）
func getEnabledCategories(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		categories, err := workorderSvc.GetEnabledCategories(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, categories)
	}
}

// getWorkOrderCategory 获取工单分类详情
func getWorkOrderCategory(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "分类ID不能为空")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		category, err := workorderSvc.GetWorkOrderCategoryByID(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		response.Success(c, category)
	}
}

// createWorkOrderCategory 创建工单分类
func createWorkOrderCategory(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.WorkOrderCategory
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.CreateWorkOrderCategory(c.Request.Context(), &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单分类",
			1, // 1=新增
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			nil,
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, req)
	}
}

// updateWorkOrderCategory 更新工单分类
func updateWorkOrderCategory(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "分类ID不能为空")
			return
		}

		var req models.WorkOrderCategory
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.UpdateWorkOrderCategory(c.Request.Context(), &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单分类",
			2, // 2=修改
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			nil,
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteWorkOrderCategory 删除工单分类
func deleteWorkOrderCategory(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "分类ID不能为空")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.DeleteWorkOrderCategory(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单分类",
			3, // 3=删除
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			nil,
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// ==================== 工单评价 Handler ====================

// createWorkOrderRating 创建工单评价
func createWorkOrderRating(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		var req struct {
			RatingType        string `json:"ratingType" binding:"required"` // user 或 handler
			CompletionScore   int    `json:"completionScore"`              // 用户评价：完成度评分
			CooperationScore  int    `json:"cooperationScore"`            // 处理人员评价：配合度评分
			Comment           string `json:"comment"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.CreateWorkOrderRating(c.Request.Context(), id, req.RatingType, req.CompletionScore, req.CooperationScore, req.Comment, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "评价成功"})
	}
}

// getWorkOrderRatings 获取工单评价列表
func getWorkOrderRatings(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		ratings, err := workorderSvc.GetWorkOrderRatings(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, ratings)
	}
}

// getRatingStatistics 获取评分统计
func getRatingStatistics(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		stats, err := workorderSvc.GetRatingStatistics(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, stats)
	}
}

// ==================== 周期性工单 Handler ====================

// listPeriodicTemplates 查询周期性工单模板列表
func listPeriodicTemplates(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.PeriodicTemplateListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		list, total, err := workorderSvc.GetPeriodicTemplateList(c.Request.Context(), &req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		pageResp := response.PageResponse{
			List:     list,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}
		response.Success(c, pageResp)
	}
}

// getPeriodicTemplate 获取周期性工单模板详情
func getPeriodicTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "模板ID不能为空")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		template, err := workorderSvc.GetPeriodicTemplate(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		response.Success(c, template)
	}
}

// createPeriodicTemplate 创建周期性工单模板
func createPeriodicTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.CreatePeriodicTemplateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		template, err := workorderSvc.CreatePeriodicTemplate(c.Request.Context(), &req, userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单模板",
			1, // 1=新增
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			nil,
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, template)
	}
}

// updatePeriodicTemplate 更新周期性工单模板
func updatePeriodicTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "模板ID不能为空")
			return
		}

		var req services.UpdatePeriodicTemplateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.UpdatePeriodicTemplate(c.Request.Context(), id, &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单模板",
			2, // 2=修改
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			nil,
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deletePeriodicTemplate 删除周期性工单模板
func deletePeriodicTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "模板ID不能为空")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.DeletePeriodicTemplate(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 删除关联的定时任务
		scheduler.DisablePeriodicWorkOrderJob(core.Scheduler, id)

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单模板",
			3, // 3=删除
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			nil,
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// enablePeriodicTemplate 启用周期性工单模板
func enablePeriodicTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "模板ID不能为空")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.EnablePeriodicTemplate(c.Request.Context(), id, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 创建/启用关联的定时任务
		if err := scheduler.EnablePeriodicWorkOrderJob(core.Scheduler, id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单模板",
			2, // 2=修改
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			nil,
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"message": "启用成功"})
	}
}

// disablePeriodicTemplate 禁用周期性工单模板
func disablePeriodicTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "模板ID不能为空")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.DisablePeriodicTemplate(c.Request.Context(), id, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 停用关联的定时任务
		scheduler.DisablePeriodicWorkOrderJob(core.Scheduler, id)

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单模板",
			2, // 2=修改
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			nil,
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"message": "禁用成功"})
	}
}

// generateWorkOrderNow 立即生成工单（测试功能）
func generateWorkOrderNow(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "模板ID不能为空")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		workOrder, err := workorderSvc.GenerateWorkOrderFromTemplate(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, workOrder)
	}
}

// getPeriodicLogs 获取周期性工单执行记录
func getPeriodicLogs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "模板ID不能为空")
			return
		}

		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		logs, err := workorderSvc.GetPeriodicLogs(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, logs)
	}
}

// ==================== 工单配置 Handler ====================

// getWorkOrderConfig 获取工单配置
func getWorkOrderConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		config, err := workorderSvc.GetWorkOrderConfig(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, config)
	}
}

// updateWorkOrderConfig 更新工单配置
func updateWorkOrderConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.WorkOrderConfig
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		workorderSvc := services.NewWorkOrderService(core.DB.GetDB())
		if err := workorderSvc.UpdateWorkOrderConfig(c.Request.Context(), &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"工单配置",
			2, // 2=修改
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			nil,
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"message": "配置更新成功"})
	}
}
