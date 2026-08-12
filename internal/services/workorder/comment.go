package workorder

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// CommentService 评论服务
type CommentService struct {
	db *gorm.DB
}

// NewCommentService 创建评论服务
func NewCommentService(db *gorm.DB) *CommentService {
	return &CommentService{db: db}
}

// AddRequest 添加评论请求
type AddRequest struct {
	Content    string `json:"content" binding:"required"`
	IsInternal bool   `json:"isInternal"`
}

// Add 添加评论
func (s *CommentService) Add(ctx context.Context, workOrderID string, req *AddRequest, userID string) error {
	// 检查工单是否存在
	var workOrder models.WorkOrder
	if err := s.db.WithContext(ctx).Where("id = ?", workOrderID).First(&workOrder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("工单不存在")
		}
		return fmt.Errorf("查询工单失败: %w", err)
	}

	comment := &models.WorkOrderComment{
		ID:          uuid.New().String(),
		WorkOrderID: workOrderID,
		UserID:      userID,
		Content:     req.Content,
		IsInternal:  req.IsInternal,
	}

	if err := s.db.WithContext(ctx).Create(comment).Error; err != nil {
		return fmt.Errorf("添加评论失败: %w", err)
	}

	return nil
}

// GetList 获取评论列表
func (s *CommentService) GetList(ctx context.Context, workOrderID string) ([]models.WorkOrderComment, error) {
	var comments []models.WorkOrderComment

	if err := s.db.WithContext(ctx).
		Where("work_order_id = ?", workOrderID).
		Preload("User").
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, fmt.Errorf("查询评论列表失败: %w", err)
	}

	return comments, nil
}

// Delete 删除评论
func (s *CommentService) Delete(ctx context.Context, commentID, userID string) error {
	var comment models.WorkOrderComment
	if err := s.db.WithContext(ctx).Where("id = ?", commentID).First(&comment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("评论不存在")
		}
		return fmt.Errorf("查询评论失败: %w", err)
	}

	// 只有评论作者可以删除
	if comment.UserID != userID {
		return fmt.Errorf("只有评论作者可以删除评论")
	}

	if err := s.db.WithContext(ctx).Delete(&comment).Error; err != nil {
		return fmt.Errorf("删除评论失败: %w", err)
	}

	return nil
}
