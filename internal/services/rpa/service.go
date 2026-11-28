package rpa

import (
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ServiceGroup RPA服务组
type ServiceGroup struct {
	db                *gorm.DB
	TaskService       TaskService
	WorkerService     WorkerService
	ExecutionService  ExecutionService
	AIService         AIService
	CredentialService CredentialService
	// LiveViewService Phase 155 / LIVEVIEW-01~05：实时画面中枢（watcher 注册表 +
	// 三模式状态机 + 审计 + 帧中转）。单例：一个后端进程一个中枢，内含唯一
	// janitor goroutine（不随 watch 增减，LIVEVIEW-05 泄漏验收前提）。
	LiveViewService *LiveViewHub
	// RecordingService Phase W6 / REC-01：录制与拾取会话中枢（会话表 +
	// Redis 下行 + Worker 上行接收 + 闲置回收 janitor）。
	// 与 LiveViewService 分立：帧通道复用 liveview，会话生命周期独立。
	RecordingService *RecordingHub
}

// NewServiceGroup 创建RPA服务组
// Phase 103 CONV-03 (D-103-20): 新增第 6 参数 cacheProvider base.CacheProvider
// 供 NewAIService → NewSelectorLearner 走 base 抽象；cacheInstance 保留
// 供 CredentialService / TaskService 继续使用（不在本 plan 范围）。
func NewServiceGroup(db *gorm.DB, cfg *config.Config, noticeHub *websocket.NoticeHub, cacheInstance cache.Cache, passwordCipher addomain.PasswordCipher, cacheProvider base.CacheProvider) *ServiceGroup {
	executionService := NewExecutionService(db, noticeHub)
	workerService := NewWorkerService(db, executionService, cfg.RPA.Storage.ScreenshotsDir)
	credentialService := NewCredentialService(db, passwordCipher, cacheInstance)
	taskService := NewTaskService(db, cacheInstance, credentialService)

	// Phase 155：实时画面中枢。logger 用窄接口（printf 风格），避免中枢反向
	// 依赖 core；noticeHub 允许为 nil（单测路径），中枢内部判空。
	var logSink interface{ Info(string, ...interface{}) } = apploggerShim{}
	liveView := NewLiveViewHub(db, cfg, noticeHub, cacheInstance, logSink)

	// Phase W6 / REC-01：录制会话中枢。cache 用于下发 Redis 指令 + 暂存会话态；
	// noticeHub 用于把录制状态/产出/拾取结果推给前端（复用既有 WS 连接）。
	recording := NewRecordingHub(RecordingHubOption{
		Cache:       cacheInstance,
		NoticeHub:   noticeHub,
		Log:         logSink,
		DB:          db,
		TaskService: taskService,
		// Config：max_sessions / idle_timeout 的唯一来源（与 worker 侧同值）。
		Config: cfg,
	})

	return &ServiceGroup{
		db:                db,
		TaskService:       taskService,
		WorkerService:     workerService,
		ExecutionService:  executionService,
		AIService:         NewAIService(cfg, db, cacheProvider),
		CredentialService: credentialService,
		LiveViewService:   liveView,
		RecordingService:  recording,
	}
}

// apploggerShim 把 pkg/logger 的函数式接口适配成 Info(string, ...interface{})。
//
// 动机：LiveViewHub 只需要「能打一行日志」，不想 import 具体 logger 实现，
// 也不想让 services 层依赖 handler 层的日志约定。
type apploggerShim struct{}

func (apploggerShim) Info(msg string, args ...interface{}) {
	if len(args) == 0 {
		applogger.Infof("%s", msg)
		return
	}
	applogger.Infof("%s %v", msg, args)
}

// DB 获取数据库连接
func (s *ServiceGroup) DB() *gorm.DB {
	return s.db
}
