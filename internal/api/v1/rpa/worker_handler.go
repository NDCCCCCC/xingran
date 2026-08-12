package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
)

// WorkerHandler Worker 处理器
type WorkerHandler struct {
	workerService rpa.WorkerService
	redisClient   *redis.Client
	core          *core.Core
}

// NewWorkerHandler 创建 Worker 处理器
func NewWorkerHandler(workerService rpa.WorkerService, core *core.Core) *WorkerHandler {
	// Get Redis client from cache
	h := &WorkerHandler{
		workerService: workerService,
		core:          core,
	}

	// Try to get Redis client from cache
	if cache := core.Cache; cache != nil {
		// The cache implementation has a client field, but we need to access it
		// For now, create a new Redis client using config
		cfg := core.Config
		h.redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Cache.Host, cfg.Cache.Port),
			Password: cfg.Cache.Password,
			DB:       cfg.Cache.DB,
			PoolSize: 10,
		})
	}

	return h
}

// WithCore 覆盖 core 依赖（链式调用，供 router 在构造后再次注入用于操作日志埋点）
func (h *WorkerHandler) WithCore(core *core.Core) *WorkerHandler {
	h.core = core
	return h
}

// List Worker 列表
func (h *WorkerHandler) List(c *gin.Context) {
	var params rpa.WorkerListParams
	if !bindAndValidate(c, &params) {
		return
	}

	setPaginationDefaults(&params.Current, &params.PageSize)

	result, err := h.workerService.List(c.Request.Context(), &params)
	if handleError(c, err, http.StatusInternalServerError, "查询失败") {
		return
	}

	success(c, result)
}

// Statistics Worker 统计(读操作,不记操作日志)
func (h *WorkerHandler) Statistics(c *gin.Context) {
	result, err := h.workerService.Statistics(c.Request.Context())
	if handleError(c, err, http.StatusInternalServerError, "统计失败") {
		return
	}

	success(c, result)
}

// Register 注册 Worker
func (h *WorkerHandler) Register(c *gin.Context) {
	var req rpa.WorkerRegisterRequest
	if !bindAndValidate(c, &req) {
		return
	}

	worker, err := h.workerService.Register(c.Request.Context(), &req)
	if handleError(c, err, http.StatusInternalServerError, "注册失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA工作节点", operlog.OperTypeRegister)

	success(c, worker)
}

// Heartbeat 心跳上报
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

// GetAutoScaleConfig 获取自动扩缩容配置
func (h *WorkerHandler) GetAutoScaleConfig(c *gin.Context) {
	// TODO: 从数据库或配置获取自动扩缩容配置
	config := AutoScaleConfig{
		Enabled:            false,
		ScaleUpThreshold:   10,
		ScaleDownThreshold: 5,
		MinConcurrency:     1,
		MaxConcurrency:     10,
		CheckInterval:      30,
	}
	success(c, config)
}

// UpdateAutoScaleConfig 更新自动扩缩容配置
func (h *WorkerHandler) UpdateAutoScaleConfig(c *gin.Context) {
	var config AutoScaleConfig
	if !bindAndValidate(c, &config) {
		return
	}

	// TODO: 保存配置到数据库
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA工作节点", operlog.OperTypeUpdate)

	successMsg(c, "自动扩缩容配置已更新")
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
func (h *WorkerHandler) publishScaleCommand(ctx context.Context, cmd *ScaleCommand) error {
	channel := "worker:scale:all"
	if cmd.WorkerID != "" {
		channel = fmt.Sprintf("worker:scale:%s", cmd.WorkerID)
	}

	// Use JSON marshal
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("marshal scale command failed: %w", err)
	}

	return h.redisClient.Publish(ctx, channel, data).Err()
}
