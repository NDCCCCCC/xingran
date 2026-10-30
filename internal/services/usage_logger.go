package services

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
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
	// D-02 / OBSERV-03: 用独立 ctx 写 DB, 忽略调用方 ctx 的取消信号。
	// 调用方 ctx (c.Request.Context()) 仅用于传递请求范围值, 不用于取消控制。
	// 超时兜底防 DB 挂起泄漏 goroutine (单条 INSERT 量级 ~10s)。
	// 先例: pkg/cache/redis.go:601-605 (detached ctx)。
	detachedCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx // 显式标注: 调用方 ctx 不用于本次 DB 写入取消控制

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
	if err := s.db.WithContext(detachedCtx).Create(&usageLog).Error; err != nil {
		// D-04: 替换 _ = err 静默吞错; 用 applogger 暴露写入失败 (升级 severity 为 Errorf)。
		// 先例: internal/services/config_backup_service.go:247 (applogger 调用模式)。
		// 安全: 仅记录 apiKeyID (UUID) + Path + err, 无 key 明文/密码/token 字段。
		applogger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", req.APIKeyID, req.Path, err)
	}
}
