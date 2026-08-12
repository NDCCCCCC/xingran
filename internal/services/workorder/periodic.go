package workorder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
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

// PeriodicTemplateStatistics 周期性工单模板统计结果。
type PeriodicTemplateStatistics struct {
	Total          int64 `json:"total"`
	Enabled        int64 `json:"enabled"`        // is_enabled = true
	Disabled       int64 `json:"disabled"`       // is_enabled = false
	TotalGenerated int64 `json:"totalGenerated"` // SUM(total_generated)
}

// GetStatistics 统计周期性工单模板总数/启停数及累计生成工单数。
// 用条件聚合(SUM CASE)避免「按当前页 list 计算统计」的错误——旧前端用当前页 list 算
// total/enabled/disabled/totalGenerated,多页时严重偏小。is_enabled 用裸布尔表达式
// (CASE WHEN is_enabled)同时兼容 PostgreSQL(boolean)与 SQLite(0/1)。
func (s *PeriodicService) GetStatistics(ctx context.Context) (*PeriodicTemplateStatistics, error) {
	var result PeriodicTemplateStatistics
	err := s.db.WithContext(ctx).
		Model(&models.PeriodicWorkOrderTemplate{}).
		Select(
			"COUNT(*) AS total",
			"SUM(CASE WHEN is_enabled THEN 1 ELSE 0 END) AS enabled",
			"SUM(CASE WHEN NOT is_enabled THEN 1 ELSE 0 END) AS disabled",
			"COALESCE(SUM(total_generated), 0) AS total_generated",
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计周期性工单模板失败: %w", err)
	}
	return &result, nil
}

// PeriodicTemplateListRequest 周期性工单模板列表请求
type PeriodicTemplateListRequest struct {
	base.BaseListRequest
	Title     string `json:"title"`
	IsEnabled *bool  `json:"isEnabled"`
}

// periodicTemplateAllowedSortFields 周期性工单模板可排序字段白名单。
var periodicTemplateAllowedSortFields = map[string]string{
	"title":       "template_name",
	"isEnabled":   "is_enabled",
	"cronExpr":    "cron_expression",
	"createdAt":   "created_at",
	"updatedAt":   "updated_at",
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

	// 用户排序(白名单)优先,无 OrderByColumn 时保留 created_at DESC 默认
	query = base.ApplySort(query, req.BaseListRequest, periodicTemplateAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.
		Preload("Category").
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
	// F-12: rotationCounted 标记 Rotation 路径是否已通过 UPDATE...RETURNING 原子 +1,
	// 避免下方 line 447 重复 +1 让 total_generated 翻倍
	var rotationCounted bool
	var dutyPoolID *string
	var dutyType string

	// 根据 AssignType 分配处理人
	switch template.AssignType {
	case models.PeriodicAssignTypeManual:
		// 手动指定：直接使用 AssignTargetID 作为用户 ID
		assigneeID = template.AssignTargetID
		isAutoAssigned = false

	case models.PeriodicAssignTypeDutyPool:
		// 值班池分配：查询今天的排班记录
		isAutoAssigned = true
		if template.AssignTargetID != nil {
			dutyPoolID = template.AssignTargetID

			// 获取今天的日期（只取日期部分，不含时间）
			today := time.Now().Truncate(24 * time.Hour).Format("2006-01-02")
			todayTime, _ := time.Parse("2006-01-02", today)

			// 查询今天的排班记录
			var schedules []models.DutySchedule
			if err := s.db.WithContext(ctx).
				Where("pool_id = ? AND schedule_date = ? AND status = ?", *dutyPoolID, todayTime, models.DutyStatusNormal).
				Find(&schedules).Error; err != nil {
				// 查询失败，记录日志但不中断流程
				applogger.Warnf("[workorder] 查询值班池 %s 的排班记录失败: %v", *dutyPoolID, err)
			} else if len(schedules) > 0 {
				// 获取第一个值班人员的 ID
				assigneeID = &schedules[0].UserID
				dutyType = string(schedules[0].DutyType)
			} else {
				// 没有找到今天的排班记录，尝试从值班池成员中选择第一个
				var members []models.DutyPoolMember
				if err := s.db.WithContext(ctx).
					Where("pool_id = ?", *dutyPoolID).
					Order("member_order ASC").
					Find(&members).Error; err == nil && len(members) > 0 {
					assigneeID = &members[0].UserID
					applogger.Debugf("[workorder] 未找到排班记录，从值班池 %s 中选择成员 %s", *dutyPoolID, *assigneeID)
				} else {
					applogger.Warnf("[workorder] 值班池 %s 中没有找到有效成员", *dutyPoolID)
				}
			}
		}

	case models.PeriodicAssignTypeRotation:
		// 轮询分配：从值班池中按顺序选择
		isAutoAssigned = true
		if template.AssignTargetID != nil {
			dutyPoolID = template.AssignTargetID

			// 查询值班池成员
			var members []models.DutyPoolMember
			if err := s.db.WithContext(ctx).
				Where("pool_id = ?", *dutyPoolID).
				Order("member_order ASC").
				Find(&members).Error; err == nil && len(members) > 0 {
				// F-12 fix: 用 UPDATE...RETURNING 原子化 total_generated +1,
				// 杜绝并发场景下两个 goroutine 读取相同 generatedCount 后选中
				// 同一成员的 lost update bug。
				// RETURNING 返回 +1 后的新值,(newCount-1) 即"本次分配前"的序号,
				// 取模选择成员。下方 line 447 的 +1 因已在此处完成需跳过。
				var newCount int64
				if err := s.db.WithContext(ctx).
					Raw("UPDATE sys_periodic_workorder_template SET total_generated = total_generated + 1 WHERE id = ? RETURNING total_generated", template.ID).
					Scan(&newCount).Error; err == nil {
					selectedIndex := int(newCount-1) % len(members)
					assigneeID = &members[selectedIndex].UserID
					applogger.Debugf("[workorder] [F-12 原子分配] 值班池 %s 选中成员 %s (索引 %d/%d, total_generated → %d)",
						*dutyPoolID, *assigneeID, selectedIndex, len(members), newCount)
					rotationCounted = true
				} else {
					// 表名/RETURNING 不可用时降级为原逻辑(SQLite 测试或未 migrate)
					generatedCount := template.TotalGenerated
					selectedIndex := generatedCount % len(members)
					assigneeID = &members[selectedIndex].UserID
					applogger.Warnf("[workorder] [F-12 降级] RETURNING 不可用,使用 generatedCount=%d 选择成员: %v", generatedCount, err)
				}
			} else {
				applogger.Warnf("[workorder] 值班池 %s 中没有找到有效成员", *dutyPoolID)
			}
		}

	default:
		// 未知类型，默认为手动指定
		assigneeID = template.AssignTargetID
		isAutoAssigned = false
	}

	// 验证 assigneeID 是否有效（如果设置了的话）
	if assigneeID != nil && *assigneeID != "" {
		var user models.User
		if err := s.db.WithContext(ctx).Where("id = ?", *assigneeID).First(&user).Error; err != nil {
			applogger.Warnf("[workorder] 警告：分配的用户 ID %s 不存在，将设置为未分配", *assigneeID)
			assigneeID = nil
		}
	}

	// 获取提交者（F-BE-35: 通过 sys_config 读取默认提交人，避免硬编码 'admin'）
	submitterID, err := s.resolveDefaultSubmitter(ctx, template.CreatedBy)
	if err != nil {
		return nil, err
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
	// F-12: 若 Rotation 路径已经原子 +1,跳过本次更新避免计数翻倍
	if !rotationCounted {
		s.db.WithContext(ctx).Model(&template).UpdateColumn("total_generated", gorm.Expr("total_generated + 1"))
	}

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

// defaultSubmitterConfigKey 是 sys_config 表中用于读取「周期性工单默认提交人用户名」的配置项键名。
// 管理员可在「参数管理」中维护该键，未配置时回退到 hardcoded "admin"（与 F-SEC-02 模式一致：
// 优先从 sys_config 读业务配置，缺失时硬编码兜底以保证向后兼容）。
const defaultSubmitterConfigKey = "sys.workorder.default_submitter_username"

// defaultSubmitterFallback 是 sys_config 缺失或查询失败时的硬编码兜底用户名。
// 仅作保险使用，避免静默使用弱默认值——最终若该用户也不存在则返回错误。
const defaultSubmitterFallback = "admin"

// resolveDefaultSubmitter 决定本次周期性工单要使用的提交人用户 ID。
// 解析策略：
//  1. 若模板创建者（templateCreatedBy）非空，先按 ID 查询该用户是否存在；
//  2. 不存在则读取 sys_config 中 sys.workorder.default_submitter_username 配置；
//  3. sys_config 也缺失/为空/查询出错时，回退到 hardcoded "admin" 用户；
//  4. 三层兜底全部失败时返回错误，调用方应中止工单创建而非继续。
func (s *PeriodicService) resolveDefaultSubmitter(ctx context.Context, templateCreatedBy string) (string, error) {
	// 1. 优先尝试模板创建者
	if templateCreatedBy != "" {
		var submitter models.User
		if err := s.db.WithContext(ctx).Where("id = ?", templateCreatedBy).First(&submitter).Error; err == nil {
			return submitter.ID, nil
		} else if err != gorm.ErrRecordNotFound {
			// 真正的查询错误（非"记录不存在"）——为避免静默绑定 admin，将错误冒泡。
			return "", fmt.Errorf("查询模板创建者 %s 失败: %w", templateCreatedBy, err)
		}
		applogger.Warnf("[workorder] 警告：模板创建者 ID %s 不存在，回退到 sys_config 默认提交人", templateCreatedBy)
	}

	// 2. 读取 sys_config 中的默认提交人用户名
	var cfg models.Config
	err := s.db.WithContext(ctx).Where("config_key = ?", defaultSubmitterConfigKey).First(&cfg).Error
	username := defaultSubmitterFallback
	switch {
	case err == nil:
		trimmed := strings.TrimSpace(cfg.ConfigValue)
		if trimmed != "" {
			username = trimmed
			applogger.Debugf("[workorder] 使用 sys_config 配置的默认提交人: %s", username)
		} else {
			applogger.Warnf("[workorder] sys_config %s 值为空，使用 hardcoded fallback: %s",
				defaultSubmitterConfigKey, defaultSubmitterFallback)
		}
	case err == gorm.ErrRecordNotFound:
		applogger.Warnf("[workorder] sys_config 未配置 %s，使用 hardcoded fallback: %s",
			defaultSubmitterConfigKey, defaultSubmitterFallback)
	default:
		// 真正的查询错误——为避免静默绑定 admin，将错误冒泡。
		return "", fmt.Errorf("读取默认提交人配置 %s 失败: %w", defaultSubmitterConfigKey, err)
	}

	// 3. 按用户名查询提交人
	var fallbackUser models.User
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&fallbackUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("无法确定工单提交者：默认提交人用户名 %q 不存在（请在 sys_config 中配置 %s 或确保该用户已创建）",
				username, defaultSubmitterConfigKey)
		}
		return "", fmt.Errorf("查询默认提交人 %s 失败: %w", username, err)
	}
	return fallbackUser.ID, nil
}
