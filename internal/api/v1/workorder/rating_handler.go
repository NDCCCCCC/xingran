package workorder

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/workorder"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// RatingHandler 工单评价处理器（FIX-06 / Phase 136）。
//
// 依赖已实现的 RatingService（internal/services/workorder/rating.go），
// 遵循 Handler-Service 分层：handler 只做参数绑定 / 取当前操作者 / 调 service /
// response 包装 / operlog，业务规则留在 service。
type RatingHandler struct {
	service *workorder.RatingService
	core    *core.Core
}

// NewRatingHandler 创建工单评价处理器。
func NewRatingHandler(service *workorder.RatingService, core *core.Core) *RatingHandler {
	return &RatingHandler{service: service, core: core}
}

// CreateRating 创建工单评价（双向评价：user 完成度 / handler 配合度）。
//
// 评价人 raterID 取自登录态 user_id（不信任请求体），service 校验工单状态、
// 重复评价、评分范围。成功路径末尾记 operlog（OperTypeCreate）。
//
// @Summary 创建工单评价
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body workorder.RatingCreateRequest true "评价信息"
// @Success 200 {object} response.Response
// @Router /workorder/orders/ratings [post]
func (h *RatingHandler) CreateRating(c *gin.Context) {
	var req workorder.RatingCreateRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, response.ErrUnauthorized)
		return
	}
	raterID, ok := userID.(string)
	if !ok || raterID == "" {
		response.Error(c, response.ErrUnauthorized)
		return
	}

	if err := h.service.Create(c.Request.Context(), &req, raterID); err != nil {
		response.Error(c, apperrors.BadRequest(err.Error()))
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单管理", operlog.OperTypeCreate)
	response.Success(c, gin.H{"message": "评价成功"})
}

// ListRatings 查询指定工单的评价列表。
//
// workOrderID 走 URL 路径段（/ratings/list/:workOrderId）或请求体 workOrderId，
// 与 service.GetList(ctx, workOrderID) 对齐；读操作不记 operlog。
//
// @Summary 查询工单评价列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /workorder/orders/ratings/list [post]
func (h *RatingHandler) ListRatings(c *gin.Context) {
	var req struct {
		WorkOrderID string `json:"workOrderId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体：workOrderId 也可由 URL 段提供
		_ = err
	}
	workOrderID := c.Param("workOrderId")
	if workOrderID == "" {
		workOrderID = req.WorkOrderID
	}
	if workOrderID == "" {
		response.Error(c, apperrors.ParamMissing("工单ID"))
		return
	}

	list, err := h.service.GetList(c.Request.Context(), workOrderID)
	if !responseHelpers.HandleServiceError(c, err, "获取工单评价列表") {
		return
	}
	response.Success(c, list)
}

// GetRatingStatistics 查询指定工单的评分统计。
//
// @Summary 查询工单评分统计
// @Tags 工单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /workorder/orders/ratings/statistics [post]
func (h *RatingHandler) GetRatingStatistics(c *gin.Context) {
	var req struct {
		WorkOrderID string `json:"workOrderId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = err
	}
	workOrderID := c.Param("workOrderId")
	if workOrderID == "" {
		workOrderID = req.WorkOrderID
	}
	if workOrderID == "" {
		response.Error(c, apperrors.ParamMissing("工单ID"))
		return
	}

	stats, err := h.service.GetStatistics(c.Request.Context(), workOrderID)
	if !responseHelpers.HandleServiceError(c, err, "获取工单评分统计") {
		return
	}
	// service 返回 snake_case key（total_count / avg_completion_score ...），
	// 前端统一用驼峰（WorkOrderRatingStatistics），在此做一层纯展示映射，
	// 不改 service 既有契约（rating.go 被其它调用方/测试引用）。
	response.Success(c, gin.H{
		"totalCount":          stats["total_count"],
		"avgCompletionScore":  stats["avg_completion_score"],
		"avgCooperationScore": stats["avg_cooperation_score"],
		"userRating":          stats["user_rating"],
		"handlerRating":       stats["handler_rating"],
	})
}
