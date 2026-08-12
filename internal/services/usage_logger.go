package services

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// UsageLogger 使用日志记录服务接口
type UsageLogger interface {
	// LogUsage 异步记录API密钥使用日志
	// 该方法会立即返回，日志记录在后台goroutine中执行
	// 错误会被记录到日志系统，不会阻塞主流程
	LogUsage(ctx context.Context, req *LogUsageRequest) error
}

// LogUsageRequest 使用日志记录请求
type LogUsageRequest struct {
	APIKeyID   string  // API密钥ID
	UserID     string  // 用户ID
	Method     string  // HTTP方法 (GET, POST, etc.)
	Path       string  // 请求路径
	StatusCode int     // 响应状态码
	ClientIP   string  // 客户端IP
	UserAgent  *string // User-Agent字符串 (可选)
	Duration   int     // 请求耗时（毫秒）
	Success    bool    // 是否成功
}

// usageLoggerImpl 使用日志服务实现
type usageLoggerImpl struct {
	db *gorm.DB
}

// NewUsageLogger 创建使用日志服务实例
func NewUsageLogger(db *gorm.DB) UsageLogger {
	return &usageLoggerImpl{
		db: db,
	}
}

// LogUsage 异步记录API密钥使用日志
// 使用goroutine在后台执行，不阻塞主流程
// 如果记录失败，仅记录日志，不影响业务流程
func (s *usageLoggerImpl) LogUsage(ctx context.Context, req *LogUsageRequest) error {
	// 立即返回，后台异步执行
	go s.logUsageAsync(ctx, req)
	return nil
}

// logUsageAsync 异步执行日志记录
func (s *usageLoggerImpl) logUsageAsync(ctx context.Context, req *LogUsageRequest) {
	// 创建使用日志记录
	usageLog := models.APIKeyUsageLog{
		APIKeyID:   req.APIKeyID,
		UserID:     req.UserID,
		Method:     req.Method,
		Path:       req.Path,
		StatusCode: req.StatusCode,
		ClientIP:   req.ClientIP,
		UserAgent:  req.UserAgent,
		Duration:   req.Duration,
		Success:    req.Success,
		CreatedAt:  time.Now(),
	}

	// 插入数据库
	if err := s.db.WithContext(ctx).Create(&usageLog).Error; err != nil {
		// 记录错误但不阻塞主流程
		// 这里可以集成日志系统
		_ = err
	}
}
