package rpa

import (
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
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
}

// NewServiceGroup 创建RPA服务组
// Phase 103 CONV-03 (D-103-20): 新增第 6 参数 cacheProvider base.CacheProvider
// 供 NewAIService → NewSelectorLearner 走 base 抽象；cacheInstance 保留
// 供 CredentialService / TaskService 继续使用（不在本 plan 范围）。
func NewServiceGroup(db *gorm.DB, cfg *config.Config, noticeHub *websocket.NoticeHub, cacheInstance cache.Cache, passwordCipher addomain.PasswordCipher, cacheProvider base.CacheProvider) *ServiceGroup {
	executionService := NewExecutionService(db, noticeHub)
	workerService := NewWorkerService(db, executionService, cfg.RPA.Storage.ScreenshotsDir)
	credentialService := NewCredentialService(db, passwordCipher, cacheInstance)

	return &ServiceGroup{
		db:                db,
		TaskService:       NewTaskService(db, cacheInstance, credentialService),
		WorkerService:     workerService,
		ExecutionService:  executionService,
		AIService:         NewAIService(cfg, db, cacheProvider),
		CredentialService: credentialService,
	}
}

// DB 获取数据库连接
func (s *ServiceGroup) DB() *gorm.DB {
	return s.db
}
