package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"gorm.io/gorm"
)

const (
	// RedisTaskStream Redis任务流名称
	RedisTaskStream = "rpa:tasks"
)

// TaskService 任务服务接口
type TaskService interface {
	Create(ctx context.Context, req *CreateTaskRequest, userID string) (*rpamodels.Task, error)
	Update(ctx context.Context, req *UpdateTaskRequest, userID string) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*rpamodels.Task, error)
	List(ctx context.Context, params *TaskListParams) (*PageResult, error)
	Execute(ctx context.Context, req *ExecuteTaskRequest, userID string) (*rpamodels.Execution, error)
}

// taskServiceImpl 任务服务实现
type taskServiceImpl struct {
	db                *gorm.DB
	cache             cache.Cache
	credentialService CredentialService
}

// NewTaskService 创建任务服务
func NewTaskService(db *gorm.DB, cacheInstance cache.Cache, credService CredentialService) TaskService {
	return &taskServiceImpl{
		db:                db,
		cache:             cacheInstance,
		credentialService: credService,
	}
}

// WorkerAction Worker 期望的动作格式（与后端 ScriptAction 兼容）
type WorkerAction struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Selector    string                 `json:"selector,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"`
	Retry       int                    `json:"retry,omitempty"`
	Value       string                 `json:"value,omitempty"`
	AIAssisted  bool                   `json:"aiAssisted,omitempty"`
}

// TaskMessage 发布到Redis的任务消息格式
type TaskMessage struct {
	ExecutionID string                 `json:"executionId"`
	TaskID      string                 `json:"taskId"`
	TaskName    string                 `json:"taskName"`
	TargetURL   string                 `json:"targetUrl"`
	Script      []WorkerAction         `json:"script"` // 使用 WorkerAction 格式
	Timeout     time.Duration          `json:"timeout"`
	MaxRetry    int                    `json:"maxRetry"`
	Variables   map[string]interface{} `json:"inputParams"`
	TriggeredBy string                 `json:"triggeredBy"`
	TriggerType string                 `json:"triggerType"`
	CreatedAt   time.Time              `json:"createdAt"`

	// 自动登录相关
	CredentialID string                 `json:"credentialId,omitempty"`
	SessionID    string                 `json:"sessionId,omitempty"`
	SessionData  map[string]interface{} `json:"sessionData,omitempty"`
}

// Create 创建任务
func (s *taskServiceImpl) Create(ctx context.Context, req *CreateTaskRequest, userID string) (*rpamodels.Task, error) {
	// 序列化脚本
	scriptJSON, err := json.Marshal(req.Script)
	if err != nil {
		return nil, fmt.Errorf("脚本序列化失败: %w", err)
	}

	task := &rpamodels.Task{
		TaskName:    req.Name,
		Description: req.Description,
		Script:      scriptJSON,
		Timeout:     req.Timeout,
		RetryCount:  req.MaxRetry,
		Priority:    rpamodels.TaskPriority(req.Priority),
		Status:      rpamodels.TaskStatus(req.Status),
		// CONTRACT-03（D5）: tags 是 req 已有字段但旧实现从未赋值 —— 标签死字段。
		Tags: req.Tags,
	}

	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, err
	}

	// CONTRACT-03: Task.Priority 的 model tag 是 `default:1`（语义已裁定放宽为
	// 0-100，0 是合法值）。GORM 的 Create 会把「零值 + HasDefaultValue」的字段
	// 替换成 DB 默认值，于是显式传 priority:0 会被写成 1 —— 与本 REQ 要求的
	// 「0 可写回」冲突。这里在创建后按请求值补偿一次（仅当落库值与请求值不同，
	// 避免无谓 UPDATE）。Update 路径走 map，不受 default tag 影响。
	if int(task.Priority) != req.Priority {
		if err := s.db.WithContext(ctx).Model(&rpamodels.Task{}).
			Where("id = ?", task.ID).
			Update("priority", req.Priority).Error; err != nil {
			return nil, fmt.Errorf("回写任务优先级失败: %w", err)
		}
		task.Priority = rpamodels.TaskPriority(req.Priority)
	}

	return task, nil
}

// Update 更新任务
func (s *taskServiceImpl) Update(ctx context.Context, req *UpdateTaskRequest, userID string) error {
	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Script != nil {
		scriptJSON, err := json.Marshal(req.Script)
		if err != nil {
			return fmt.Errorf("脚本序列化失败: %w", err)
		}
		updates["script"] = scriptJSON
	}
	if req.Timeout > 0 {
		updates["timeout_seconds"] = req.Timeout
	}
	// CONTRACT-03（D5）: MaxRetry 的旧判定 `>= 0` 恒真 —— 未传（零值 0）时会
	// 把既有 retry_count 清零。改为「显式传了才写」：req.MaxRetry > 0。
	// （maxRetry=0 的「不重试」语义仍可通过 Create 表达；Update 侧保守保留原值。）
	if req.MaxRetry > 0 {
		updates["retry_count"] = req.MaxRetry
	}
	// CONTRACT-03（D5）: 指针语义 —— nil=未传（保留原值），非 nil=写回（含 0）。
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	// CONTRACT-03（D5）: tags 旧实现从未落库（只有 req 字段没有赋值）。
	// 空串视为「未传」，保持既有「只改 name/desc 不误清空标签」的 update 语义。
	if req.Tags != "" {
		updates["tags"] = req.Tags
	}

	updates["updated_by"] = userID
	updates["version"] = gorm.Expr("version + 1")

	return s.db.WithContext(ctx).Model(&rpamodels.Task{}).Where("id = ?", req.ID).Updates(updates).Error
}

// Delete 删除任务（软删除）
func (s *taskServiceImpl) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&rpamodels.Task{}).Error
}

// GetByID 获取任务详情
func (s *taskServiceImpl) GetByID(ctx context.Context, id string) (*rpamodels.Task, error) {
	var task rpamodels.Task
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// List 查询任务列表
func (s *taskServiceImpl) List(ctx context.Context, params *TaskListParams) (*PageResult, error) {
	var tasks []rpamodels.Task
	var total int64

	query := s.db.WithContext(ctx).Model(&rpamodels.Task{}).Where("deleted_at IS NULL")

	// 添加过滤条件
	// CONTRACT-03（D3）: 真实列名是 `name`（Task.TaskName 的 gorm column:name），
	// 旧实现写 task_name → sqlite/PG 均报 "no such column: task_name"。
	if params.Name != "" {
		query = query.Where("name LIKE ?", "%"+params.Name+"%")
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}
	if params.Priority != nil {
		query = query.Where("priority = ?", *params.Priority)
	}
	if params.Tags != "" {
		query = query.Where("tags LIKE ?", "%"+params.Tags+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询 - 用户排序(白名单)优先,无 OrderByColumn 时保留 created_at DESC 默认
	offset := (params.Current - 1) * params.PageSize
	query = base.ApplySort(query, params.BaseListRequest, taskAllowedSortFields)
	if params.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Offset(offset).Limit(params.PageSize).Find(&tasks).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     tasks,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// Execute 执行任务
func (s *taskServiceImpl) Execute(ctx context.Context, req *ExecuteTaskRequest, userID string) (*rpamodels.Execution, error) {
	// 获取任务
	task, err := s.GetByID(ctx, req.TaskID)
	if err != nil {
		return nil, fmt.Errorf("任务不存在: %w", err)
	}

	// 检查任务状态
	if !task.IsEnabled() {
		return nil, fmt.Errorf("任务已禁用")
	}

	// 解析脚本动作
	actions, err := task.GetActions()
	if err != nil {
		return nil, fmt.Errorf("解析脚本失败: %w", err)
	}

	// 创建执行记录
	// Phase 158 / CLEANUP-02：trigger_type 由请求决定（manual / schedule）。
	// 调度触发与手动触发共用本入口（同一份状态机、同一条 Redis Stream 通道），
	// 差别只在 trigger_type 与 triggered_by 两个审计字段。
	triggerType := req.TriggerType
	if triggerType == "" {
		triggerType = "manual"
	}

	execution := &rpamodels.Execution{
		TaskID:      req.TaskID,
		TaskName:    task.TaskName,
		Status:      string(rpamodels.RPAExecutionStatusPending),
		TriggeredBy: userID,
		TriggerType: triggerType,
	}

	if err := s.db.WithContext(ctx).Create(execution).Error; err != nil {
		return nil, err
	}

	// 将任务发送到 Redis Stream
	if err := s.publishTaskToRedis(ctx, execution.ID, task, actions, req.InputParams, userID, req.CredentialID, triggerType); err != nil {
		// 发布失败，删除执行记录
		s.db.WithContext(ctx).Delete(execution)
		return nil, fmt.Errorf("发布任务到Redis失败: %w", err)
	}

	return execution, nil
}

// publishTaskToRedis 将任务发布到Redis Stream
func (s *taskServiceImpl) publishTaskToRedis(
	ctx context.Context,
	executionID string,
	task *rpamodels.Task,
	actions []rpamodels.ScriptAction,
	inputParams map[string]interface{},
	userID string,
	credentialID string,
	triggerType string,
) error {
	// 从脚本中提取目标URL
	targetURL := extractURLFromScript(actions)

	// 转换 ScriptAction 为 WorkerAction 格式
	workerActions := make([]WorkerAction, len(actions))
	for i, action := range actions {
		workerActions[i] = convertToWorkerAction(action, i)
	}

	// 构建任务消息
	if triggerType == "" {
		triggerType = "manual"
	}
	message := TaskMessage{
		ExecutionID:  executionID,
		TaskID:       task.ID,
		TaskName:     task.TaskName,
		TargetURL:    targetURL,
		Script:       workerActions, // 使用转换后的格式
		Timeout:      time.Duration(task.Timeout) * time.Second,
		MaxRetry:     task.RetryCount,
		Variables:    inputParams,
		TriggeredBy:  userID,
		TriggerType:  triggerType,
		CreatedAt:    time.Now(),
		CredentialID: credentialID,
	}

	// Phase 153 / RUNTIME-02：会话数据迁移到 Variables["__sessionData"]。
	// worker 端 engine.go（VariableSessionData 常量）按该键读取并真实注入
	// cookie/localStorage/header。顶层 sessionData 字段保留一版（兼容旧 worker）。
	if message.SessionData != nil {
		if message.Variables == nil {
			message.Variables = map[string]interface{}{}
		}
		message.Variables["__sessionData"] = message.SessionData
	}

	// 如果提供了凭证ID，尝试获取有效会话或凭证
	if credentialID != "" && s.credentialService != nil {
		// RUNTIME-02：目标系统按 URL host 规则解析（erp./hr./crm. 等子域 →
		// 系统名），无法识别时才回退 "default"
		targetSystem := extractTargetSystem(targetURL)

		// 先尝试获取有效会话
		session, err := s.credentialService.GetValidSession(ctx, credentialID, targetSystem)
		if err == nil && session != nil {
			// 有有效会话，传递会话数据（顶层 + __sessionData 双写）
			message.SessionID = session.ID
			message.SessionData = map[string]interface{}{
				"accessToken":  session.AccessToken,
				"refreshToken": session.RefreshToken,
				"cookies":      session.Cookies,
				"sessionData":  session.SessionData,
			}
			if message.Variables == nil {
				message.Variables = map[string]interface{}{}
			}
			message.Variables["__sessionData"] = message.SessionData
		} else {
			// 没有有效会话，传递凭证信息用于自动登录
			// RUNTIME-02：deptID 从执行上下文取真实值（按 userID 查 sys_user），
			// 查询失败不阻断（部门关联缺失时按空串降级）
			deptID := s.resolveUserDeptID(ctx, userID)
			cred, err := s.credentialService.GetCredentialForExecution(ctx, targetSystem, userID, deptID)
			if err == nil && cred != nil {
				// 解密凭证数据
				credData, err := s.credentialService.DecryptCredential(ctx, cred)
				if err == nil {
					message.CredentialID = cred.ID
					if message.Variables == nil {
						message.Variables = map[string]interface{}{}
					}
					message.Variables["__credentials"] = map[string]interface{}{
						"username":  credData.Username,
						"password":  credData.Password,
						"extraData": credData.ExtraData,
					}
					// 更新最后使用时间（尽力而为）
					_ = s.credentialService.UpdateLastUsed(ctx, cred.ID)
				}
			}
		}
	}

	// 序列化消息
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化任务消息失败: %w", err)
	}

	// 获取Redis客户端并发布到Stream
	// 首先检查是否为直接的 RedisCache
	type redisCacheInterface interface {
		getClient() *redis.Client
	}

	if redisCache, ok := s.cache.(redisCacheInterface); ok {
		redisClient := redisCache.getClient()
		if redisClient == nil {
			return fmt.Errorf("Redis客户端未初始化")
		}

		result := redisClient.XAdd(ctx, &redis.XAddArgs{
			Stream: RedisTaskStream,
			Values: map[string]interface{}{"data": string(data)},
		})

		if err := result.Err(); err != nil {
			return fmt.Errorf("XADD失败: %w", err)
		}

		return nil
	}

	// 检查是否为 MultiLevelCache，使用 DirectRedisXAdd 方法
	type multiLevelCacheInterface interface {
		DirectRedisXAdd(ctx context.Context, stream string, values map[string]interface{}) error
	}

	if mlCache, ok := s.cache.(multiLevelCacheInterface); ok {
		err := mlCache.DirectRedisXAdd(ctx, RedisTaskStream, map[string]interface{}{"data": string(data)})
		if err != nil {
			return fmt.Errorf("XADD失败: %w", err)
		}
		return nil
	}

	return fmt.Errorf("缓存类型不支持Redis操作: cache类型=%T", s.cache)
}

// extractURLFromScript 从脚本中提取第一个导航动作的URL
func extractURLFromScript(actions []rpamodels.ScriptAction) string {
	for _, action := range actions {
		if action.Type == "navigate" && action.Value != "" {
			return action.Value
		}
	}
	return ""
}

// extractTargetSystem 从URL中提取目标系统标识
func extractTargetSystem(url string) string {
	// Phase 153 / RUNTIME-02：可配置映射 —— URL host 规则优先。
	// 规则：子域名首段即目标系统标识（erp.company.com → erp、hr.a.b → hr）；
	// 裸 IP / localhost / 单标签 host 无法给出语义 → 回退 "default"。
	if url == "" {
		return "unknown"
	}

	host := url
	// 剥离 scheme
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	// 剥离 path / query / port / userinfo
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host = host[:i]
	}
	if i := strings.LastIndex(host, "@"); i >= 0 {
		host = host[i+1:]
	}
	if i := strings.LastIndex(host, ":"); i >= 0 && !strings.Contains(host[i:], "]") {
		host = host[:i]
	}
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return "default"
	}
	// 裸 IP / localhost 无语义
	if net.ParseIP(host) != nil || strings.EqualFold(host, "localhost") {
		return "default"
	}

	labels := strings.Split(host, ".")
	if len(labels) >= 2 {
		first := strings.ToLower(labels[0])
		if first != "" && first != "www" {
			return first
		}
	}
	return "default"
}

// resolveUserDeptID RUNTIME-02：按 userID 查真实部门 ID。
// sys_user 表在测试/无用户场景可能缺失，查询失败降级为空串（不阻断发布）。
func (s *taskServiceImpl) resolveUserDeptID(ctx context.Context, userID string) string {
	if userID == "" {
		return ""
	}
	var deptID string
	if err := s.db.WithContext(ctx).
		Table("sys_user").
		Select("dept_id").
		Where("id = ?", userID).
		Scan(&deptID).Error; err != nil {
		return ""
	}
	return deptID
}

// convertToWorkerAction 将 ScriptAction 转换为 Worker 期望的 Action 格式
// 统一前后端和 Worker 之间的数据格式
// 使用值接收者，因为方法不修改接收者状态
func convertToWorkerAction(action rpamodels.ScriptAction, index int) WorkerAction {
	workerAction := WorkerAction{
		ID:       fmt.Sprintf("action_%d", index),
		Type:     action.Type,
		Selector: action.Selector,
		Timeout:  action.Timeout,
		Retry:    action.Retry,
		Value:    action.Value,
	}

	// 初始化 params
	workerAction.Params = make(map[string]interface{})

	// 从 attributes 中提取描述到顶层 description
	if action.Attributes != nil {
		if desc, ok := action.Attributes["description"].(string); ok {
			workerAction.Description = desc
		}
		// 将其他 attributes 复制到 params
		for k, v := range action.Attributes {
			if k != "description" {
				workerAction.Params[k] = v
			}
		}
	}

	// 对于 navigate 动作，将 value 设置到 params.url
	if action.Type == "navigate" && action.Value != "" {
		workerAction.Params["url"] = action.Value
	}

	// 对于 fill/select 动作，将 value 设置到 params.value
	if (action.Type == "fill" || action.Type == "select") && action.Value != "" {
		workerAction.Params["value"] = action.Value
	}

	// 对于 wait 动作，将 attributes.duration 设置到 params.duration
	if action.Type == "wait" {
		if duration, ok := action.Attributes["duration"].(float64); ok {
			workerAction.Params["duration"] = int(duration)
		} else if duration, ok := action.Attributes["duration"].(int); ok {
			workerAction.Params["duration"] = duration
		}
	}

	return workerAction
}
