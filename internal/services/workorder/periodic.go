package workorder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// PeriodicService 周期性工单服务
type PeriodicService struct {
	db *gorm.DB
}

// NewPeriodicService 创建周期性工单服务
func NewPeriodicService(db *gorm.DB) *PeriodicService {
	return &PeriodicService{db: db}
}

// PeriodicTemplateListRequest 周期性工单模板列表请求
type PeriodicTemplateListRequest struct {
	Current   int    `json:"current"`
	PageSize  int    `json:"pageSize"`
	Title     string `json:"title"`
	IsEnabled *bool  `json:"isEnabled"`
}

// GetTemplateList 获取周期性工单模板列表
func (s *PeriodicService) GetTemplateList(ctx context.Context, req *PeriodicTemplateListRequest) ([]models.PeriodicWorkOrderTemplate, int64, error) {
	var list []models.PeriodicWorkOrderTemplate
	var total int64

	query := s.db.WithContext(ctx).Model(&models.PeriodicWorkOrderTemplate{})

	if req.Title != "" {
		query = query.Where("template_name LIKE ?", "%"+req.Title+"%")
	}
	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询周期性工单模板总数失败: %w", err)
	}

	current := req.Current
	if current <= 0 {
		current = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (current - 1) * pageSize

	if err := query.
		Preload("Category").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询周期性工单模板列表失败: %w", err)
	}

	return list, total, nil
}

// GetTemplate 获取周期性工单模板详情
func (s *PeriodicService) GetTemplate(ctx context.Context, id string) (*models.PeriodicWorkOrderTemplate, error) {
	var template models.PeriodicWorkOrderTemplate

	if err := s.db.WithContext(ctx).
		Preload("Category").
		Preload("ExecutionLogs", func(db *gorm.DB) *gorm.DB {
			return db.Order("executed_at DESC").Limit(10)
		}).
		Where("id = ?", id).
		First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("周期性工单模板不存在")
		}
		return nil, fmt.Errorf("查询周期性工单模板详情失败: %w", err)
	}

	return &template, nil
}

// CreateTemplateRequest 创建周期性工单模板请求
type CreateTemplateRequest struct {
	TemplateName   string                             `json:"templateName" binding:"required,max=100"`
	WorkOrderTitle string                             `json:"workOrderTitle" binding:"required,max=200"`
	Description    string                             `json:"description"`
	CategoryID     string                             `json:"categoryId" binding:"required,uuid"`
	Type           models.WorkOrderType               `json:"type" binding:"required"`
	Priority       models.WorkOrderPriority           `json:"priority"`
	CronExpression string                             `json:"cronExpression" binding:"required"`
	AssignType     models.PeriodicWorkOrderAssignType `json:"assignType"`
	AssignTargetID *string                            `json:"assignTargetId"`
	NotifyAssignee bool                               `json:"notifyAssignee"`
}

// CreateTemplate 创建周期性工单模板
func (s *PeriodicService) CreateTemplate(ctx context.Context, req *CreateTemplateRequest, creatorID string) (*models.PeriodicWorkOrderTemplate, error) {
	// 验证Cron表达式
	if _, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(req.CronExpression); err != nil {
		return nil, fmt.Errorf("无效的Cron表达式: %w", err)
	}

	template := &models.PeriodicWorkOrderTemplate{
		BaseModel: models.BaseModel{
			CreatedBy: creatorID,
			UpdatedBy: creatorID,
		},
		TemplateName:   req.TemplateName,
		WorkOrderTitle: req.WorkOrderTitle,
		Description:    req.Description,
		CategoryID:     req.CategoryID,
		Type:           req.Type,
		Priority:       req.Priority,
		CronExpression: req.CronExpression,
		AssignType:     req.AssignType,
		AssignTargetID: req.AssignTargetID,
		IsEnabled:      false,
		NotifyAssignee: req.NotifyAssignee,
		TotalGenerated: 0,
	}

	if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
		return nil, fmt.Errorf("创建周期性工单模板失败: %w", err)
	}

	return template, nil
}

// UpdateTemplateRequest 更新周期性工单模板请求
type UpdateTemplateRequest struct {
	TemplateName   *string                             `json:"templateName"`
	WorkOrderTitle *string                             `json:"workOrderTitle"`
	Description    *string                             `json:"description"`
	CategoryID     *string                             `json:"categoryId"`
	Type           *models.WorkOrderType               `json:"type"`
	Priority       *models.WorkOrderPriority           `json:"priority"`
	CronExpression *string                             `json:"cronExpression"`
	AssignType     *models.PeriodicWorkOrderAssignType `json:"assignType"`
	AssignTargetID *string                             `json:"assignTargetId"`
	IsEnabled      *bool                               `json:"isEnabled"`
	NotifyAssignee *bool                               `json:"notifyAssignee"`
}

// UpdateTemplate 更新周期性工单模板
func (s *PeriodicService) UpdateTemplate(ctx context.Context, id string, req *UpdateTemplateRequest, operatorID string) error {
	var template models.PeriodicWorkOrderTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("周期性工单模板不存在")
		}
		return fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	updates := map[string]interface{}{
		"updated_by": operatorID,
	}

	if req.TemplateName != nil {
		updates["template_name"] = *req.TemplateName
	}
	if req.WorkOrderTitle != nil {
		updates["work_order_title"] = *req.WorkOrderTitle
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.CronExpression != nil {
		if _, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(*req.CronExpression); err != nil {
			return fmt.Errorf("无效的Cron表达式: %w", err)
		}
		updates["cron_expression"] = *req.CronExpression
	}
	if req.AssignType != nil {
		updates["assign_type"] = *req.AssignType
	}
	if req.AssignTargetID != nil {
		updates["assign_target_id"] = *req.AssignTargetID
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.NotifyAssignee != nil {
		updates["notify_assignee"] = *req.NotifyAssignee
	}

	if err := s.db.WithContext(ctx).Model(&template).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新周期性工单模板失败: %w", err)
	}

	return nil
}

// DeleteTemplate 删除周期性工单模板
func (s *PeriodicService) DeleteTemplate(ctx context.Context, id string) error {
	var template models.PeriodicWorkOrderTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("周期性工单模板不存在")
		}
		return fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	// 检查是否启用
	if template.IsEnabled {
		return fmt.Errorf("请先禁用周期性工单模板")
	}

	// 删除执行记录
	if err := s.db.WithContext(ctx).Where("template_id = ?", id).Delete(&models.PeriodicWorkOrderLog{}).Error; err != nil {
		return fmt.Errorf("删除执行记录失败: %w", err)
	}

	// 删除模板
	if err := s.db.WithContext(ctx).Delete(&template).Error; err != nil {
		return fmt.Errorf("删除周期性工单模板失败: %w", err)
	}

	return nil
}

// EnableTemplate 启用周期性工单模板
func (s *PeriodicService) EnableTemplate(ctx context.Context, id string, operatorID string) error {
	var template models.PeriodicWorkOrderTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("周期性工单模板不存在")
		}
		return fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	// 计算下次执行时间
	c := cron.New(cron.WithSeconds(), cron.WithLocation(time.Local))
	entryID, err := c.AddFunc(template.CronExpression, func() {})
	if err != nil {
		return fmt.Errorf("解析Cron表达式失败: %w", err)
	}
	c.Start()
	defer c.Stop()
	nextRun := c.Entry(entryID).Next

	// 更新模板
	updates := map[string]interface{}{
		"is_enabled":  true,
		"next_run_at": nextRun,
		"updated_by":  operatorID,
	}

	if err := s.db.WithContext(ctx).Model(&template).Updates(updates).Error; err != nil {
		return fmt.Errorf("启用周期性工单模板失败: %w", err)
	}

	return nil
}

// DisableTemplate 禁用周期性工单模板
func (s *PeriodicService) DisableTemplate(ctx context.Context, id string, operatorID string) error {
	var template models.PeriodicWorkOrderTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("周期性工单模板不存在")
		}
		return fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	// 更新模板
	updates := map[string]interface{}{
		"is_enabled": false,
		"updated_by": operatorID,
	}

	if err := s.db.WithContext(ctx).Model(&template).Updates(updates).Error; err != nil {
		return fmt.Errorf("禁用周期性工单模板失败: %w", err)
	}

	return nil
}

// GenerateWorkOrder 根据模板生成工单
func (s *PeriodicService) GenerateWorkOrder(ctx context.Context, templateID string) (*models.WorkOrder, error) {
	var template models.PeriodicWorkOrderTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", templateID).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("周期性工单模板不存在")
		}
		return nil, fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	// 替换标题中的变量
	title := s.replaceTemplateVariables(template.WorkOrderTitle)
	description := s.replaceTemplateVariables(template.Description)

	// 确定处理人
	var assigneeID *string
	var isAutoAssigned bool
	var dutyPoolID *string
	var dutyType string

	// 这里简化处理，实际应该根据 AssignType 分配
	assigneeID = template.AssignTargetID
	isAutoAssigned = false

	// 获取提交者
	submitterID := ""
	if template.CreatedBy != "" {
		var submitter models.User
		if err := s.db.WithContext(ctx).Where("username = ?", template.CreatedBy).First(&submitter).Error; err == nil {
			submitterID = submitter.ID
		}
	}

	if submitterID == "" {
		var adminUser models.User
		if err := s.db.WithContext(ctx).Where("username = ?", "admin").First(&adminUser).Error; err == nil {
			submitterID = adminUser.ID
		} else {
			return nil, fmt.Errorf("无法确定工单提交者：未找到模板创建者或系统管理员")
		}
	}

	// 创建工单
	workOrder := &models.WorkOrder{
		BaseModel: models.BaseModel{
			CreatedBy: template.BaseModel.CreatedBy,
			UpdatedBy: template.BaseModel.CreatedBy,
		},
		Title:          title,
		CategoryID:     template.CategoryID,
		Type:           template.Type,
		Priority:       template.Priority,
		Status:         models.WorkOrderStatusPending,
		Description:    description,
		SubmitterID:    submitterID,
		AssigneeID:     assigneeID,
		IsAutoAssigned: isAutoAssigned,
		DutyPoolID:     dutyPoolID,
		DutyType:       dutyType,
		AssignStrategy: string(template.AssignType),
	}

	if err := s.db.WithContext(ctx).Create(workOrder).Error; err != nil {
		return nil, fmt.Errorf("创建工单失败: %w", err)
	}

	// 更新模板的生成计数和下次执行时间
	s.db.WithContext(ctx).Model(&template).UpdateColumn("total_generated", gorm.Expr("total_generated + 1"))

	c := cron.New(cron.WithSeconds(), cron.WithLocation(time.Local))
	entryID, _ := c.AddFunc(template.CronExpression, func() {})
	c.Start()
	defer c.Stop()
	nextRun := c.Entry(entryID).Next
	s.db.WithContext(ctx).Model(&template).UpdateColumn("next_run_at", nextRun)

	// 记录执行日志
	execLog := &models.PeriodicWorkOrderLog{
		ID:          uuid.New().String(),
		TemplateID:  templateID,
		WorkOrderID: workOrder.ID,
		ExecutedAt:  time.Now(),
		Status:      "success",
		Result:      fmt.Sprintf("成功创建工单: %s", workOrder.WorkOrderNo),
	}
	s.db.WithContext(ctx).Create(execLog)

	return workOrder, nil
}

// replaceTemplateVariables 替换模板变量
func (s *PeriodicService) replaceTemplateVariables(text string) string {
	now := time.Now()

	variables := map[string]string{
		"{date}":    now.Format("2006-01-02"),
		"{year}":    now.Format("2006"),
		"{month}":   now.Format("01"),
		"{day}":     now.Format("02"),
		"{weekday}": getWeekdayName(now.Weekday()),
		"{hour}":    now.Format("15"),
		"{minute}":  now.Format("04"),
	}

	result := text
	for key, value := range variables {
		result = strings.ReplaceAll(result, key, value)
	}

	return result
}

// GetLogs 获取周期性工单执行记录
func (s *PeriodicService) GetLogs(ctx context.Context, templateID string) ([]models.PeriodicWorkOrderLog, error) {
	var logs []models.PeriodicWorkOrderLog

	err := s.db.WithContext(ctx).
		Where("template_id = ?", templateID).
		Order("executed_at DESC").
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("查询执行记录失败: %w", err)
	}

	return logs, nil
}

// getWeekdayName 获取星期名称
func getWeekdayName(wd time.Weekday) string {
	weekdays := map[time.Weekday]string{
		time.Sunday:    "周日",
		time.Monday:    "周一",
		time.Tuesday:   "周二",
		time.Wednesday: "周三",
		time.Thursday:  "周四",
		time.Friday:    "周五",
		time.Saturday:  "周六",
	}
	return weekdays[wd]
}
