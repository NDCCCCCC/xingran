package workorder

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// RatingService 评价服务
type RatingService struct {
	db *gorm.DB
}

// NewRatingService 创建评价服务
func NewRatingService(db *gorm.DB) *RatingService {
	return &RatingService{db: db}
}

// RatingCreateRequest 创建评价请求
type RatingCreateRequest struct {
	WorkOrderID      string `json:"workOrderId" binding:"required,uuid"`
	RatingType       string `json:"ratingType" binding:"required"` // user 或 handler
	CompletionScore  int    `json:"completionScore"`               // 完成度评分 (1-5)
	CooperationScore int    `json:"cooperationScore"`              // 配合度评分 (1-5)
	Comment          string `json:"comment"`
}

// Create 创建评价
func (s *RatingService) Create(ctx context.Context, req *RatingCreateRequest, raterID string) error {
	// 检查工单是否存在且状态是否允许评价
	var workOrder models.WorkOrder
	if err := s.db.WithContext(ctx).Where("id = ?", req.WorkOrderID).First(&workOrder).Error; err != nil {
		return fmt.Errorf("查询工单失败: %w", err)
	}

	// 只有已完成或已关闭的工单才能评价
	if workOrder.Status != models.WorkOrderStatusCompleted && workOrder.Status != models.WorkOrderStatusClosed {
		return fmt.Errorf("只有已完成或已关闭的工单才能评价")
	}

	// 检查是否已经评价过
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.WorkOrderRating{}).
		Where("work_order_id = ? AND rating_type = ? AND rater_id = ?", req.WorkOrderID, req.RatingType, raterID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("检查评价记录失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("您已经评价过该工单")
	}

	// 验证评价类型
	if req.RatingType != "user" && req.RatingType != "handler" {
		return fmt.Errorf("无效的评价类型")
	}

	// 验证评分
	if req.RatingType == "user" && (req.CompletionScore < 1 || req.CompletionScore > 5) {
		return fmt.Errorf("完成度评分必须在1-5之间")
	}
	if req.RatingType == "handler" && (req.CooperationScore < 1 || req.CooperationScore > 5) {
		return fmt.Errorf("配合度评分必须在1-5之间")
	}

	rating := &models.WorkOrderRating{
		ID:               uuid.New().String(),
		WorkOrderID:      req.WorkOrderID,
		RatingType:       req.RatingType,
		CompletionScore:  req.CompletionScore,
		CooperationScore: req.CooperationScore,
		Comment:          req.Comment,
		RaterID:          raterID,
	}

	if err := s.db.WithContext(ctx).Create(rating).Error; err != nil {
		return fmt.Errorf("创建评价失败: %w", err)
	}

	return nil
}

// GetList 获取工单评价列表
func (s *RatingService) GetList(ctx context.Context, workOrderID string) ([]models.WorkOrderRating, error) {
	var ratings []models.WorkOrderRating

	if err := s.db.WithContext(ctx).
		Where("work_order_id = ?", workOrderID).
		Preload("Rater").
		Order("created_at DESC").
		Find(&ratings).Error; err != nil {
		return nil, fmt.Errorf("查询评价列表失败: %w", err)
	}

	return ratings, nil
}

// GetStatistics 获取评分统计
func (s *RatingService) GetStatistics(ctx context.Context, workOrderID string) (map[string]interface{}, error) {
	var ratings []models.WorkOrderRating

	if err := s.db.WithContext(ctx).
		Where("work_order_id = ?", workOrderID).
		Find(&ratings).Error; err != nil {
		return nil, fmt.Errorf("查询评价列表失败: %w", err)
	}

	stats := map[string]interface{}{
		"total_count":    len(ratings),
		"user_rating":    nil,
		"handler_rating": nil,
	}

	var userRating, handlerRating *models.WorkOrderRating
	var completionSum, cooperationSum int
	var completionCount, cooperationCount int

	for _, r := range ratings {
		if r.RatingType == "user" && userRating == nil {
			userRating = &r
			completionSum += r.CompletionScore
			completionCount++
		}
		if r.RatingType == "handler" && handlerRating == nil {
			handlerRating = &r
			cooperationSum += r.CooperationScore
			cooperationCount++
		}
	}

	stats["user_rating"] = userRating
	stats["handler_rating"] = handlerRating

	// 计算平均分
	if completionCount > 0 {
		stats["avg_completion_score"] = float64(completionSum) / float64(completionCount)
	}
	if cooperationCount > 0 {
		stats["avg_cooperation_score"] = float64(cooperationSum) / float64(cooperationCount)
	}

	return stats, nil
}
