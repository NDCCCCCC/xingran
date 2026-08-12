package rpa

import (
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
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
func NewServiceGroup(db *gorm.DB, cfg *config.Config, noticeHub *websocket.NoticeHub, cacheInstance cache.Cache, passwordCipher addomain.PasswordCipher) *ServiceGroup {
	executionService := NewExecutionService(db, noticeHub)
	workerService := NewWorkerService(db, executionService, cfg.RPA.Storage.ScreenshotsDir)
	credentialService := NewCredentialService(db, passwordCipher, cacheInstance)

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
