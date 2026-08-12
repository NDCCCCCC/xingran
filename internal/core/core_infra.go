// Package core 提供系统的核心功能模块
// 基础设施层：管理数据库、缓存、加密、调度等基础设施组件
package core

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/scheduler"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// CoreInfra 核心基础设施层
// 包含系统运行所需的基础设施组件：配置、数据库、缓存、JWT、密码、加密、调度器等。
// 这些组件通常是底层依赖，不直接处理业务逻辑。
//
// 通过 struct embedding 嵌入到 Core 中，保持向后兼容的字段访问语法（core.DB、core.Cache 等）。
type CoreInfra struct {
	Config     *config.Config            // 系统配置
	DB         *db.Database              // 数据库连接管理器
	Cache      cache.Cache               // 缓存管理器
	JWTManager *security.JWTManager      // JWT 令牌管理器
	PwdManager *security.PasswordManager // 密码管理器
	SM4Cipher  addomain.PasswordCipher   // SM4 加密器（用于 AD 域密码加密）

	// 调度器与执行器
	Scheduler *scheduler.Scheduler // 定时任务调度器

	// 验证码服务
	CaptchaService           *CaptchaService           // 验证码服务
	CaptchaBackgroundService *CaptchaBackgroundService // 验证码背景图服务

	// 指标缓存服务
	MetricsCacheService *MetricsCacheService // 系统指标缓存服务

	// 认证策略工厂
	AuthFactory *security.AuthStrategyFactory // 认证策略工厂

	// 子进程 reaper（P2-A7: 清理僵尸子进程，防止 FD 泄漏）
	reaperCtx    context.Context
	reaperCancel context.CancelFunc
}
