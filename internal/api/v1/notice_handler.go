package v1

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// getIntField 从map中获取int字段
func getIntFieldFromMap(m map[string]interface{}, key string, defaultVal int) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case int64:
			return int(v)
		}
	}
	return defaultVal
}

// NoticeType 通知类型枚举
type NoticeType string

const (
	NoticeTypeNotice NoticeType = "1" // 公告
	NoticeTypeAlert  NoticeType = "2" // 警告
)

// NoticeListRequest 通知公告列表请求
type NoticeListRequest struct {
	NoticeTitle *string     `json:"noticeTitle,omitempty"`
	NoticeType  *NoticeType `json:"noticeType,omitempty"`
	CreateTime  *string     `json:"createTime,omitempty"`
	Current     int         `json:"current" binding:"min=1"`
	PageSize    int         `json:"pageSize" binding:"min=1,max=100"`
}

// NoticeChannelRequest 通知渠道配置请求
type NoticeChannelRequest struct {
	ChannelType      models.NotificationChannelType `json:"channelType" binding:"required"`
	EmailConfigID    *string                        `json:"emailConfigId,omitempty"`
	APIConfigID      *string                        `json:"apiConfigId,omitempty"`
	CustomRecipients *[]string                      `json:"customRecipients,omitempty"` // 自定义收件人列表（邮件地址或企微用户代码）
}

// RecurrenceConfig 周期性执行配置
type RecurrenceConfig struct {
	CronExpression *string `json:"cronExpression" binding:"required"` // 自定义 Cron 表达式
	EndDate        *string `json:"endDate,omitempty"`                // 结束日期（可选），ISO格式字符串
}

// NoticeCreateRequest 创建通知公告请求（增强版）
type NoticeCreateRequest struct {
	NoticeTitle      string                 `json:"noticeTitle" binding:"required,min=2,max=100"`
	NoticeType       NoticeType             `json:"noticeType" binding:"required,oneof=1 2"`
	NoticeContent    string                 `json:"noticeContent" binding:"required"`
	Priority         models.NoticePriority  `json:"priority" binding:"min=0,max=2"`
	PublishTime      *time.Time             `json:"publishTime"`
	ExecutionType    *string                `json:"executionType,omitempty" binding:"omitempty,oneof=once recurring"` // "once"(一次性) 或 "recurring"(周期性)
	RecurrenceConfig *RecurrenceConfig      `json:"recurrenceConfig,omitempty"`                                       // 周期性配置
	CronExpression   *string                `json:"cronExpression,omitempty"`                                         // Cron表达式，用于周期性发送（已废弃，使用RecurrenceConfig）
	TargetType       models.TargetType      `json:"targetType" binding:"min=0,max=3"`
	TargetDepts      []string               `json:"targetDepts,omitempty"`
	TargetRoles      []string               `json:"targetRoles,omitempty"`
	TargetUsers      []string               `json:"targetUsers,omitempty"`
	IsMarkdown       bool                   `json:"isMarkdown"`
	Status           int                    `json:"status" binding:"min=0,max=1,oneof=0 1"`
	Channels         []NoticeChannelRequest `json:"channels,omitempty"`
}

// NoticeUpdateRequest 更新通知公告请求（增强版）
type NoticeUpdateRequest struct {
	NoticeTitle      *string                `json:"noticeTitle" binding:"omitempty,min=2,max=100"`
	NoticeType       *NoticeType            `json:"noticeType" binding:"omitempty,oneof=1 2"`
	NoticeContent    *string                `json:"noticeContent" binding:"omitempty"`
	Priority         *models.NoticePriority `json:"priority" binding:"omitempty,min=0,max=2"`
	Status           *int                   `json:"status" binding:"omitempty,min=0,max=1,oneof=0 1"`
	PublishTime      *time.Time             `json:"publishTime,omitempty"`
	ClearPublishTime bool                   `json:"clearPublishTime,omitempty"`
	Channels         []NoticeChannelRequest `json:"channels,omitempty"`
}

// listNotices 查询通知公告列表（增强版）
// @Summary 查询通知公告列表
// @Description 分页查询通知公告列表
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param request body NoticeListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /notices/list [post]
func listNotices(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			rawReq = make(map[string]interface{})
		}

		req := NoticeListRequest{
			Current:  getIntFieldFromMap(rawReq, "current", 1),
			PageSize: getIntFieldFromMap(rawReq, "pageSize", 10),
		}

		if val, ok := rawReq["noticeTitle"].(string); ok && val != "" {
			req.NoticeTitle = &val
		}
		if val, ok := rawReq["noticeType"].(string); ok && val != "" {
			noticeType := NoticeType(val)
			req.NoticeType = &noticeType
		}
		if val, ok := rawReq["createTime"].(string); ok && val != "" {
			req.CreateTime = &val
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())

		var noticeTypeStr *string
		if req.NoticeType != nil {
			s := string(*req.NoticeType)
			noticeTypeStr = &s
		}

		notices, total, err := noticeService.GetNoticeList(c.Request.Context(), req.Current, req.PageSize, req.NoticeTitle, noticeTypeStr)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		pageResp := response.PageResponse{
			List:     notices,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// createNotice 创建通知公告（增强版）
// @Summary 创建通知公告
// @Description 创建新通知公告（支持定向推送、定时发布、周期性发送、优先级、多渠道发送）
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param request body NoticeCreateRequest true "通知公告信息"
// @Success 200 {object} response.Response
// @Router /notices [post]
func createNotice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req NoticeCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 获取当前用户信息
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "未授权")
			return
		}

		// 获取用户名
		username, _ := c.Get("username")
		usernameStr, _ := username.(string)

		noticeService := services.NewNoticeService(core.DB.GetDB())

		createReq := &services.CreateNoticeRequest{
			NoticeTitle:   req.NoticeTitle,
			NoticeType:    string(req.NoticeType),
			NoticeContent: req.NoticeContent,
			Priority:      req.Priority,
			PublishTime:   req.PublishTime,
			TargetType:    req.TargetType,
			TargetDepts:   req.TargetDepts,
			TargetRoles:   req.TargetRoles,
			TargetUsers:   req.TargetUsers,
			IsMarkdown:    req.IsMarkdown,
		}

		notice, err := noticeService.CreateNoticeWithTargets(c.Request.Context(), createReq, userID.(string), usernameStr)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 判断执行类型，用于后续处理
		executionType := "once" // 默认一次性
		if req.ExecutionType != nil {
			executionType = *req.ExecutionType
		}

		// 标记是否需要更新发布状态为"定时发布中"
		shouldUpdateToScheduled := false

		// 设置发送渠道
		if len(req.Channels) > 0 {
			senderService := services.NewNotificationSenderService(core.DB.GetDB())
			channels := make([]models.NotificationChannel, len(req.Channels))
			for i, ch := range req.Channels {
				channels[i] = models.NotificationChannel{
					ChannelType:      ch.ChannelType,
					EmailConfigID:    ch.EmailConfigID,
					APIConfigID:      ch.APIConfigID,
					CustomRecipients: ch.CustomRecipients,
				}
			}
			if err := senderService.SetNotificationChannels(c.Request.Context(), notice.ID, channels); err != nil {
				response.Error(c, response.ErrServerError, "设置发送渠道失败: "+err.Error())
				return
			}
		}

		// 处理定时任务
		if core.Scheduler != nil {
			var cronExpr string
			var misfirePolicy models.MisfirePolicy

			// 调试日志
			logger.Debugf("[NOTICE] executionType=%s, RecurrenceConfig=%v", executionType, req.RecurrenceConfig != nil)

			if executionType == "recurring" && req.RecurrenceConfig != nil {
				// 周期性执行，直接使用 Cron 表达式
				if req.RecurrenceConfig.CronExpression == nil || *req.RecurrenceConfig.CronExpression == "" {
					response.Error(c, response.ErrBadRequest, "Cron表达式不能为空")
					return
				}
				cronExpr = *req.RecurrenceConfig.CronExpression
				misfirePolicy = models.MisfirePolicyDefault // 周期性任务使用默认策略，不自动删除
				shouldUpdateToScheduled = true              // 标记需要更新状态
			} else if req.PublishTime != nil && req.PublishTime.After(time.Now()) {
				// 一次性定时任务
				cronExpr = services.CalculateCronExpression(*req.PublishTime)
				misfirePolicy = models.MisfirePolicyExecuteOnce // 执行后删除
				shouldUpdateToScheduled = true                  // 标记需要更新状态
			} else {
				// 没有定时任务，直接返回
				response.Success(c, gin.H{"id": notice.ID, "message": "创建成功"})
				return
			}

			job := &models.Job{
				JobName:        services.GetNoticeJobName(notice.ID),
				JobGroup:       "notice",
				InvokeTarget:   "notice_publish:" + notice.ID,
				CronExpression: cronExpr,
				MisfirePolicy:  misfirePolicy,
				Concurrent:     false,
				Status:         0, // 正常状态
				Remark:         strPtr(fmt.Sprintf("通知定时发布: %s", req.NoticeTitle)),
			}

			if err := core.Scheduler.AddJob(job); err != nil {
				// 任务创建失败，返回错误
				response.Error(c, response.ErrServerError, "创建定时任务失败: "+err.Error())
				return
			}

			// 定时任务创建成功后，更新发布状态为"定时发布中"，并保存结束时间
			if shouldUpdateToScheduled {
				updates := map[string]interface{}{
					"publish_status": models.PublishStatusScheduled,
				}
				// 如果是周期性任务且有结束时间，保存结束时间
				if executionType == "recurring" && req.RecurrenceConfig != nil && req.RecurrenceConfig.EndDate != nil {
					// 解析 ISO 格式的时间字符串
					if endTime, err := time.Parse(time.RFC3339, *req.RecurrenceConfig.EndDate); err == nil {
						updates["end_date"] = endTime
					}
				}
				if err := core.DB.GetDB().Model(&models.Notice{}).
					Where("id = ?", notice.ID).
					Updates(updates).Error; err != nil {
					logger.Warnf("[NOTICE] 更新发布状态失败: %v", err)
					// 不中断流程，因为定时任务已创建成功
				}
			}
		} else if executionType == "recurring" || (req.PublishTime != nil && req.PublishTime.After(time.Now())) {
			// 需要定时任务但调度器不可用
			response.Error(c, response.ErrServerError, "调度器不可用，无法创建定时任务")
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "通知公告", OperTypeCreate)

		response.Success(c, gin.H{"id": notice.ID, "message": "创建成功"})
	}
}

// getNotice 获取通知公告详情（增强版）
// @Summary 获取通知公告详情
// @Description 根据ID获取通知公告详情（包含接收范围）
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response{data=models.Notice}
// @Router /notices/:id [post]
func getNotice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())
		notice, err := noticeService.GetNoticeByID(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		response.Success(c, notice)
	}
}

// updateNotice 更新通知公告（增强版）
// @Summary 更新通知公告
// @Description 更新通知公告信息
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Param request body NoticeUpdateRequest true "通知公告信息"
// @Success 200 {object} response.Response
// @Router /notices/:id/update [post]
func updateNotice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req NoticeUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())

		updateReq := &services.UpdateNoticeRequest{
			NoticeTitle: req.NoticeTitle,
			NoticeType: func() *string {
				if req.NoticeType != nil {
					s := string(*req.NoticeType)
					return &s
				}
				return nil
			}(),
			NoticeContent:    req.NoticeContent,
			Priority:         req.Priority,
			Status:           req.Status,
			PublishTime:      req.PublishTime,
			ClearPublishTime: req.ClearPublishTime,
		}

		if err := noticeService.UpdateNotice(c.Request.Context(), id, updateReq); err != nil {
			if err.Error() == "通知不存在" {
				response.Error(c, response.ErrNotFound, err.Error())
			} else {
				response.Error(c, response.ErrServerError, err.Error())
			}
			return
		}

		// 设置发送渠道（如果提供了）
		if len(req.Channels) > 0 {
			senderService := services.NewNotificationSenderService(core.DB.GetDB())
			channels := make([]models.NotificationChannel, len(req.Channels))
			for i, ch := range req.Channels {
				channels[i] = models.NotificationChannel{
					ChannelType:      ch.ChannelType,
					EmailConfigID:    ch.EmailConfigID,
					APIConfigID:      ch.APIConfigID,
					CustomRecipients: ch.CustomRecipients,
				}
			}
			if err := senderService.SetNotificationChannels(c.Request.Context(), id, channels); err != nil {
				response.Error(c, response.ErrServerError, "设置发送渠道失败: "+err.Error())
				return
			}
		}

		// 处理定时任务更新
		if core.Scheduler != nil {
			jobName := services.GetNoticeJobName(id)

			// 先删除旧的定时任务
			var oldJob models.Job
			if err := core.DB.GetDB().Where("job_name = ?", jobName).First(&oldJob).Error; err == nil {
				core.Scheduler.RemoveJob(oldJob.ID)
			}

			// 如果设置了新的定时发布时间，创建新任务
			if req.PublishTime != nil && req.PublishTime.After(time.Now()) {
				cronExpr := services.CalculateCronExpression(*req.PublishTime)
				title := "通知定时发布"
				if req.NoticeTitle != nil {
					title = *req.NoticeTitle
				} else {
					// 如果标题未修改，从数据库获取
					var notice models.Notice
					if err := core.DB.GetDB().Select("notice_title").Where("id = ?", id).First(&notice).Error; err == nil {
						title = notice.NoticeTitle
					}
				}
				job := &models.Job{
					JobName:        jobName,
					JobGroup:       "notice",
					InvokeTarget:   "notice_publish:" + id,
					CronExpression: cronExpr,
					MisfirePolicy:  models.MisfirePolicyExecuteOnce,
					Concurrent:     false,
					Status:         0,
					Remark:         strPtr(fmt.Sprintf("通知定时发布: %s", title)),
				}
				if err := core.Scheduler.AddJob(job); err != nil {
					// 任务创建失败，记录警告但不中断流程
					response.Error(c, response.ErrServerError, "创建定时任务失败: "+err.Error())
					return
				}
			}
		}

		// 记录操作日志
		recordOperLog(c, core, "通知公告", OperTypeUpdate)

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteNotice 删除通知公告
// @Summary 删除通知公告
// @Description 删除指定通知公告
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response
// @Router /notices/:id/delete [post]
func deleteNotice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())
		if err := noticeService.DeleteNotice(c.Request.Context(), id); err != nil {
			if err.Error() == "通知不存在" {
				response.Error(c, response.ErrNotFound, err.Error())
			} else {
				response.Error(c, response.ErrServerError, err.Error())
			}
			return
		}

		// 删除关联的定时任务
		if core.Scheduler != nil {
			jobName := services.GetNoticeJobName(id)
			// 查找任务ID
			var job models.Job
			if err := core.DB.GetDB().Where("job_name = ? AND job_group = ?", jobName, "notice").First(&job).Error; err == nil {
				// 删除任务
				core.Scheduler.RemoveJob(job.ID)
				// 删除数据库记录
				core.DB.GetDB().Delete(&job)
			}
		}

		// 记录操作日志
		recordOperLog(c, core, "通知公告", OperTypeDelete)

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// batchDeleteNotices 批量删除通知公告
// @Summary 批量删除通知公告
// @Description 批量删除多个通知公告
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param request body map[string][]string true "ID列表"
// @Success 200 {object} response.Response
// @Router /notices/batch-delete [post]
func batchDeleteNotices(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Ids []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())
		if err := noticeService.BatchDeleteNotices(c.Request.Context(), req.Ids); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 删除关联的定时任务
		if core.Scheduler != nil {
			for _, id := range req.Ids {
				jobName := services.GetNoticeJobName(id)
				// 查找任务ID
				var job models.Job
				if err := core.DB.GetDB().Where("job_name = ? AND job_group = ?", jobName, "notice").First(&job).Error; err == nil {
					// 删除任务
					core.Scheduler.RemoveJob(job.ID)
					// 删除数据库记录
					core.DB.GetDB().Delete(&job)
				}
			}
		}

		// 记录操作日志
		recordOperLog(c, core, "通知公告", OperTypeDelete)

		response.Success(c, gin.H{"message": "批量删除成功"})
	}
}

// getNoticeStatistics 获取通知阅读统计
// @Summary 获取通知阅读统计
// @Description 获取指定通知的阅读统计数据
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response{data=services.NoticeStatistics}
// @Router /notices/:id/statistics [get]
func getNoticeStatistics(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())
		stats, err := noticeService.GetNoticeStatistics(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, stats)
	}
}

// publishNotice 发布通知
// @Summary 发布通知
// @Description 将草稿状态的通知发布并发送到各个渠道
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response
// @Router /notices/:id/publish [post]
func publishNotice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		senderService := services.NewNotificationSenderService(core.DB.GetDB())
		if err := senderService.PublishAndSendNotice(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "发布成功"})
	}
}

// withdrawNotice 撤回/取消发布通知
// @Summary 撤回通知
// @Description 将已发布的通知撤回，退回到草稿状态
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response
// @Router /notices/:id/withdraw [post]
func withdrawNotice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())
		if err := noticeService.WithdrawNotice(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "撤回成功"})
	}
}

// strPtr 返回字符串指针的辅助函数
func strPtr(s string) *string {
	return &s
}

// getNoticeChannels 获取通知的发送渠道配置
// @Summary 获取通知的发送渠道
// @Description 获取指定通知的发送渠道配置
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response{data=[]models.NotificationChannel}
// @Router /notices/:id/channels [get]
func getNoticeChannels(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		senderService := services.NewNotificationSenderService(core.DB.GetDB())
		channels, err := senderService.GetNotificationChannels(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, channels)
	}
}

// setNoticeChannels 设置通知的发送渠道配置
// @Summary 设置通知的发送渠道
// @Description 设置指定通知的发送渠道配置
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Param request body []NoticeChannelRequest true "发送渠道配置"
// @Success 200 {object} response.Response
// @Router /notices/:id/channels [post]
func setNoticeChannels(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req []NoticeChannelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		senderService := services.NewNotificationSenderService(core.DB.GetDB())
		channels := make([]models.NotificationChannel, len(req))
		for i, ch := range req {
			channels[i] = models.NotificationChannel{
				ChannelType:      ch.ChannelType,
				EmailConfigID:    ch.EmailConfigID,
				APIConfigID:      ch.APIConfigID,
				CustomRecipients: ch.CustomRecipients,
			}
		}

		if err := senderService.SetNotificationChannels(c.Request.Context(), id, channels); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "设置成功"})
	}
}

// getCronExpressions 获取常用Cron表达式列表
// @Summary 获取常用Cron表达式
// @Description 获取常用的Cron表达式列表，用于周期性任务设置
// @Tags 通知公告
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]services.CommonCronExpression}
// @Router /notices/cron-expressions [get]
func getCronExpressions(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		expressions := services.GetCommonCronExpressions()
		response.Success(c, expressions)
	}
}
