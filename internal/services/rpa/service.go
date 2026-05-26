package rpa

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/websocket"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/cache"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/crypto"
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
func NewServiceGroup(db *gorm.DB, cfg *config.Config, noticeHub *websocket.NoticeHub, cacheInstance cache.Cache, sm4 *crypto.SM4Cipher) *ServiceGroup {
	executionService := NewExecutionService(db, noticeHub)
	workerService := NewWorkerService(db, executionService, cfg.RPA.Storage.ScreenshotsDir)
	credentialService := NewCredentialService(db, sm4, cacheInstance)

	return &ServiceGroup{
		db:                db,
		TaskService:       NewTaskService(db, cacheInstance, credentialService),
		WorkerService:     workerService,
		ExecutionService:  executionService,
		AIService:         NewAIService(cfg, db, cacheInstance),
		CredentialService: credentialService,
	}
}

// DB 获取数据库连接
func (s *ServiceGroup) DB() *gorm.DB {
	return s.db
}
