package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"gorm.io/gorm"
)

// WorkerHandler Worker 处理器
type WorkerHandler struct {
	workerService rpa.WorkerService
	core          *core.Core
	// tokenSvc DEBT-03：Worker Token 换发服务（nil = 认证未配置，register 不换发）。
	tokenSvc rpa.WorkerTokenService
}

// NewWorkerHandler 创建 Worker 处理器
//
// Phase 158 / CLEANUP-01：删除「每实例 redis.NewClient」。
// 此前本构造函数每次都新建一条 Redis 连接（PoolSize=10）且从不 Close，
// 路由每次装配都会泄漏一个连接池；Redis 发布一律改走 core.Cache
// （由 rpa.PublishScaleCommand 复用其底层客户端）。
func NewWorkerHandler(workerService rpa.WorkerService, core *core.Core) *WorkerHandler {
	return &WorkerHandler{
		workerService: workerService,
		core:          core,
	}
}

// WithCore 覆盖 core 依赖（链式调用，供 router 在构造后再次注入用于操作日志埋点）
func (h *WorkerHandler) WithCore(core *core.Core) *WorkerHandler {
	h.core = core
	return h
}

// WithTokenService 注入 Worker Token 服务（DEBT-03，链式调用；nil 表示认证未配置）
func (h *WorkerHandler) WithTokenService(svc rpa.WorkerTokenService) *WorkerHandler {
	h.tokenSvc = svc
	return h
}

// List Worker 列表
// @Summary 查询RPA Worker列表
// @Description 分页查询RPA执行器（Worker）列表
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "查询参数"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/workers/list [post]
func (h *WorkerHandler) List(c *gin.Context) {
	var params rpa.WorkerListParams
	if !bindAndValidate(c, &params) {
		return
	}
	if !validateListParams(c, params.Validate()) {
		return
	}
	params.Status = normalizeOptionalStringFilter(params.Status)

	setPaginationDefaults(&params.Current, &params.PageSize)

	result, err := h.workerService.List(c.Request.Context(), &params)
	if handleError(c, err, http.StatusInternalServerError, "查询失败") {
		return
	}

	success(c, result)
}

// Statistics Worker 统计(读操作,不记操作日志)
// @Summary RPA Worker统计
// @Description RPA Worker 专用聚合统计端点
// @Tags RPA管理
// @Produce json
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/workers/statistics [post]
func (h *WorkerHandler) Statistics(c *gin.Context) {
	result, err := h.workerService.Statistics(c.Request.Context())
	if handleError(c, err, http.StatusInternalServerError, "统计失败") {
		return
	}

	success(c, result)
}

// Register 注册 Worker
// @Summary 注册RPA Worker
// @Description Worker 自动注册（无需 JWT，使用 Provisioning 认证）
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "注册信息"
// @Success 200 {object} object
// @Failure 400 {object} object
// @Failure 500 {object} object
// @Router /rpa/workers/register [post]
func (h *WorkerHandler) Register(c *gin.Context) {
	var req rpa.WorkerRegisterRequest
	if !bindAndValidate(c, &req) {
		return
	}

	worker, err := h.workerService.Register(c.Request.Context(), &req)
	if handleError(c, err, http.StatusInternalServerError, "注册失败") {
		return
	}

	// DEBT-03：请求携带合法 Provisioning Token 时换发 Worker Token（响应 data.token）。
	// 未携带/服务未配置 → 不换发（宽限期双读：老 Worker 继续无 Token 通信）。
	if h.tokenSvc != nil && h.tokenSvc.ValidateProvisioningToken(c.GetHeader(middleware.HeaderProvisioningToken)) {
		if pair, terr := h.tokenSvc.IssueWorkerToken(c.Request.Context(), worker.WorkerID); terr != nil {
			// 换发失败不阻断注册（宽限期语义：Worker 仍可无 Token 通信），仅告警
			applogger.Warnf("[RPA-WORKER-AUTH] Worker %s Token 换发失败（注册成功但无 Token）: %v", worker.WorkerID, terr)
		} else if pair != nil {
			worker.Token = pair.Token
			worker.TokenExpiresAt = &pair.ExpiresAt
		}
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA工作节点", operlog.OperTypeRegister)

	success(c, worker)
}

// Heartbeat 心跳上报
// @Summary RPA Worker心跳
// @Description Worker 定期心跳上报（Worker Token 认证）
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param id path string true "Worker ID"
// @Param request body object true "心跳数据"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/workers/{id}/heartbeat [post]
func (h *WorkerHandler) Heartbeat(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	var req rpa.WorkerHeartbeatRequest
	if c.ShouldBindJSON(&req) == nil {
		req.WorkerID = id
	} else {
		req = rpa.WorkerHeartbeatRequest{WorkerID: id}
	}

	if handleError(c, h.workerService.Heartbeat(c.Request.Context(), &req), http.StatusInternalServerError, "心跳更新失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA工作节点", operlog.OperTypeOther)

	successMsg(c, "心跳更新成功")
}

// Progress 进度上报
// @Summary RPA执行进度上报
// @Description Worker 上报任务执行进度（Worker Token 认证）
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "进度数据"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/workers/progress [post]
func (h *WorkerHandler) Progress(c *gin.Context) {
	var req rpa.WorkerProgressRequest
	if !bindAndValidate(c, &req) {
		return
	}

	if handleError(c, h.workerService.Progress(c.Request.Context(), &req), http.StatusInternalServerError, "进度更新失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA工作节点", operlog.OperTypeOther)

	successMsg(c, "进度更新成功")
}

// ScaleUp 扩容 Worker
// @Summary RPA Worker扩容
// @Description 触发指定 Worker 扩容
// @Tags RPA管理
// @Produce json
// @Param id path string true "Worker ID"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/workers/{id}/scale-up [post]
func (h *WorkerHandler) ScaleUp(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	var req struct {
		Concurrency int    `json:"concurrency" binding:"required,min=1,max=50"`
		Reason      string `json:"reason"`
	}
	if !bindAndValidate(c, &req) {
		return
	}

	if req.Reason == "" {
		req.Reason = "manual scale up"
	}

	cmd := ScaleCommand{
		CommandID:   uuid.New().String(),
		WorkerID:    id,
		Direction:   "up",
		Concurrency: req.Concurrency,
		Reason:      req.Reason,
		Timestamp:   time.Now().Unix(),
	}

	if err := h.publishScaleCommand(c.Request.Context(), &cmd); err != nil {
		handleError(c, err, http.StatusInternalServerError, "发送扩容指令失败")
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA工作节点", operlog.OperTypeStatus)

	successMsg(c, "扩容指令已发送")
}

// ScaleDown 缩容 Worker
// @Summary RPA Worker缩容
// @Description 触发指定 Worker 缩容
// @Tags RPA管理
// @Produce json
// @Param id path string true "Worker ID"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/workers/{id}/scale-down [post]
func (h *WorkerHandler) ScaleDown(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	var req struct {
		Concurrency int    `json:"concurrency" binding:"required,min=1,max=50"`
		Reason      string `json:"reason"`
	}
	if !bindAndValidate(c, &req) {
		return
	}

	if req.Reason == "" {
		req.Reason = "manual scale down"
	}

	cmd := ScaleCommand{
		CommandID:   uuid.New().String(),
		WorkerID:    id,
		Direction:   "down",
		Concurrency: req.Concurrency,
		Reason:      req.Reason,
		Timestamp:   time.Now().Unix(),
	}

	if err := h.publishScaleCommand(c.Request.Context(), &cmd); err != nil {
		handleError(c, err, http.StatusInternalServerError, "发送缩容指令失败")
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA工作节点", operlog.OperTypeStatus)

	successMsg(c, "缩容指令已发送")
}

// ScaleAll 批量扩缩容所有 Worker
// @Summary RPA Worker全量扩缩容
// @Description 对全部 Worker 执行统一扩缩容指令
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "扩缩容指令"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/workers/scale-all [post]
func (h *WorkerHandler) ScaleAll(c *gin.Context) {
	var req struct {
		Direction   string `json:"direction" binding:"required,oneof=up down"`
		Concurrency int    `json:"concurrency" binding:"required,min=1,max=50"`
		Reason      string `json:"reason"`
	}
	if !bindAndValidate(c, &req) {
		return
	}

	if req.Reason == "" {
		req.Reason = "batch scale operation"
	}

	cmd := ScaleCommand{
		CommandID:   uuid.New().String(),
		WorkerID:    "", // Empty means all workers
		Direction:   req.Direction,
		Concurrency: req.Concurrency,
		Reason:      req.Reason,
		Timestamp:   time.Now().Unix(),
	}

	if err := h.publishScaleCommand(c.Request.Context(), &cmd); err != nil {
		handleError(c, err, http.StatusInternalServerError, "发送批量扩缩容指令失败")
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA工作节点", operlog.OperTypeStatus)

	successMsg(c, "批量扩缩容指令已发送")
}

// AutoScaleConfig 自动扩缩容配置
type AutoScaleConfig struct {
	Enabled            bool `json:"enabled"`
	ScaleUpThreshold   int  `json:"scale_up_threshold"`   // 队列长度触发扩容
	ScaleDownThreshold int  `json:"scale_down_threshold"` // 空闲时长触发缩容
	MinConcurrency     int  `json:"min_concurrency"`
	MaxConcurrency     int  `json:"max_concurrency"`
	CheckInterval      int  `json:"check_interval"` // seconds
}

// autoScaleConfigKey 自动扩缩容配置的 sys_config 落库键（Phase 158 / CLEANUP-01）。
const autoScaleConfigKey = "sys.rpa.autoscale.config"

// defaultAutoScaleConfig 自动扩缩容配置默认值。
//
// 这是「首次使用尚未落库」的兜底值，不是假数据：一旦 POST 过配置，
// GET 即返回落库值（读-改-写路径见 saveAutoScaleConfig）。
func defaultAutoScaleConfig() *AutoScaleConfig {
	return &AutoScaleConfig{
		Enabled:            false,
		ScaleUpThreshold:   10,
		ScaleDownThreshold: 5,
		MinConcurrency:     1,
		MaxConcurrency:     10,
		CheckInterval:      30,
	}
}

// GetAutoScaleConfig 获取自动扩缩容配置
// @Summary 查询自动扩缩容配置
// @Description 查询 RPA 自动扩缩容配置（持久化于 sys_config，重启后仍生效）
// @Tags RPA管理
// @Produce json
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/workers/autoscale/config [get]
func (h *WorkerHandler) GetAutoScaleConfig(c *gin.Context) {
	config, err := h.loadAutoScaleConfig(c.Request.Context())
	if handleError(c, err, http.StatusInternalServerError, "查询自动扩缩容配置失败") {
		return
	}
	success(c, config)
}

// UpdateAutoScaleConfig 更新自动扩缩容配置
// @Summary 更新自动扩缩容配置
// @Description 更新 RPA 自动扩缩容配置（持久化于 sys_config）
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "扩缩容配置"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/workers/autoscale/config [post]
func (h *WorkerHandler) UpdateAutoScaleConfig(c *gin.Context) {
	var config AutoScaleConfig
	if !bindAndValidate(c, &config) {
		return
	}

	if err := h.saveAutoScaleConfig(c.Request.Context(), &config); err != nil {
		handleError(c, err, http.StatusInternalServerError, "保存自动扩缩容配置失败")
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA工作节点", operlog.OperTypeUpdate)

	successMsg(c, "自动扩缩容配置已更新")
}

// loadAutoScaleConfig 从 sys_config 读取自动扩缩容配置。
//
// Phase 158 / CLEANUP-01：此前直接返回硬编码默认值（GET 恒等于常量、POST 丢弃
// 请求体），是本 phase 的第二处假配置。落库键 = sys.rpa.autoscale.config。
// 键不存在时返回默认配置（首次使用的兜底；一旦写回，读写即一致）。
func (h *WorkerHandler) loadAutoScaleConfig(ctx context.Context) (*AutoScaleConfig, error) {
	config := defaultAutoScaleConfig()

	db := h.core.GetDB()
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	var row models.Config
	err := db.WithContext(ctx).
		Where("config_key = ?", autoScaleConfigKey).
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 尚未配置过 → 返回默认值（写回后即为真值）
			return config, nil
		}
		return nil, fmt.Errorf("查询自动扩缩容配置失败: %w", err)
	}

	if row.ConfigValue == "" {
		return config, nil
	}
	if jErr := json.Unmarshal([]byte(row.ConfigValue), config); jErr != nil {
		return nil, fmt.Errorf("解析自动扩缩容配置失败: %w", jErr)
	}
	return config, nil
}

// saveAutoScaleConfig 把自动扩缩容配置写入 sys_config（读-改-写，幂等）。
func (h *WorkerHandler) saveAutoScaleConfig(ctx context.Context, config *AutoScaleConfig) error {
	db := h.core.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	payload, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化自动扩缩容配置失败: %w", err)
	}
	value := string(payload)

	var row models.Config
	err = db.WithContext(ctx).
		Where("config_key = ?", autoScaleConfigKey).
		First(&row).Error
	switch {
	case err == nil:
		if uErr := db.WithContext(ctx).
			Model(&models.Config{}).
			Where("id = ?", row.ID).
			Update("config_value", value).Error; uErr != nil {
			return fmt.Errorf("更新自动扩缩容配置失败: %w", uErr)
		}
		return nil
	case err == gorm.ErrRecordNotFound:
		newRow := &models.Config{
			ConfigName:  "RPA 自动扩缩容配置",
			ConfigKey:   autoScaleConfigKey,
			ConfigValue: value,
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemNo,
			Remark:      "Phase 158 / CLEANUP-01：RPA Worker 自动扩缩容参数（GET/POST 真读真写）",
		}
		if cErr := db.WithContext(ctx).Create(newRow).Error; cErr != nil {
			return fmt.Errorf("创建自动扩缩容配置失败: %w", cErr)
		}
		return nil
	default:
		return fmt.Errorf("查询自动扩缩容配置失败: %w", err)
	}
}

// ScaleCommand 扩缩容指令
type ScaleCommand struct {
	CommandID   string `json:"commandId"`
	WorkerID    string `json:"workerId"`
	Direction   string `json:"direction"`
	Concurrency int    `json:"concurrency"`
	Reason      string `json:"reason"`
	Timestamp   int64  `json:"timestamp"`
}

// publishScaleCommand 发布扩缩容指令到 Redis
//
// Phase 158 / CLEANUP-01：改走 core.Cache 复用底层 Redis 客户端，
// 删除 handler 持有的独立 *redis.Client（每实例新建连接池 → 泄漏）。
func (h *WorkerHandler) publishScaleCommand(ctx context.Context, cmd *ScaleCommand) error {
	channel := "worker:scale:all"
	if cmd.WorkerID != "" {
		channel = fmt.Sprintf("worker:scale:%s", cmd.WorkerID)
	}

	if h.core == nil {
		return fmt.Errorf("core 未注入，无法发布扩缩容指令")
	}

	return rpa.PublishScaleCommand(ctx, h.core.Cache, channel, cmd)
}
