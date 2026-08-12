package security

import (
	"context"
	"errors"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"gorm.io/gorm"
)

// AuthStrategyFactory 认证策略工厂
// 根据认证模式动态创建对应的认证器
type AuthStrategyFactory struct {
	db          *gorm.DB
	pwdManager  *PasswordManager
	userSyncer  UserSyncer              // 用户同步服务（可选）
	sm4Cipher   addomain.PasswordCipher // SM4加密器（用于解密AD管理员密码）
	accountPool addomain.AccountPool    // Phase 36: 账号池（可选）
}

// NewAuthStrategyFactory 创建认证策略工厂
func NewAuthStrategyFactory(db *gorm.DB, pwdMgr *PasswordManager, sm4Cipher ...addomain.PasswordCipher) *AuthStrategyFactory {
	f := &AuthStrategyFactory{
		db:         db,
		pwdManager: pwdMgr,
	}
	if len(sm4Cipher) > 0 && sm4Cipher[0] != nil {
		f.sm4Cipher = sm4Cipher[0]
	}
	return f
}

// SetUserSyncer 设置用户同步服务
func (f *AuthStrategyFactory) SetUserSyncer(syncer UserSyncer) {
	f.userSyncer = syncer
}

// SetAccountPool Phase 36: 注入账号池到工厂
// 注入后 AD/Hybrid 模式自动使用 FailoverClient 多账号故障切换
func (f *AuthStrategyFactory) SetAccountPool(pool addomain.AccountPool) {
	f.accountPool = pool
}

// GetAccountPool Phase 38 Wave 1: 返回工厂持有的账号池实例（供 router 层复用共享实例）
// 用于 user_router 等 caller 接入同一 AccountPool（Pitfall 4：避免重复 New 导致缓存不共享）
func (f *AuthStrategyFactory) GetAccountPool() addomain.AccountPool {
	return f.accountPool
}

// GetAuthenticator 根据认证模式获取认证器
// mode: "local" | "ad" | "hybrid"
func (f *AuthStrategyFactory) GetAuthenticator(mode string) (Authenticator, error) {
	switch mode {
	case "local":
		return NewLocalAuthenticator(f.db, f.pwdManager), nil

	case "ad":
		// 从配置读取默认AD配置ID
		configID, err := f.getADConfigID()
		if err != nil {
			return nil, err
		}
		ad := NewADAuthenticator(f.db, configID)
		ad.SetUserSyncer(f.userSyncer)
		if f.sm4Cipher != nil {
			ad.SetPasswordCipher(f.sm4Cipher)
		}
		// Phase 36: 注入账号池（可选，未注入时走旧 bindAdmin）
		if f.accountPool != nil {
			ad.SetAccountPool(f.accountPool)
		}
		return ad, nil

	case "hybrid":
		local := NewLocalAuthenticator(f.db, f.pwdManager)
		configID, err := f.getADConfigID()
		if err != nil {
			return nil, err
		}
		ad := NewADAuthenticator(f.db, configID)
		ad.SetUserSyncer(f.userSyncer)
		if f.sm4Cipher != nil {
			ad.SetPasswordCipher(f.sm4Cipher)
		}
		// Phase 36: 注入账号池
		if f.accountPool != nil {
			ad.SetAccountPool(f.accountPool)
		}
		return NewHybridAuthenticator(local, ad), nil

	default:
		return nil, errors.New("不支持的认证模式: " + mode)
	}
}

// GetDefaultAuthMode 获取默认认证模式
// 从sys_config表读取配置，不存在时返回"local"
func (f *AuthStrategyFactory) GetDefaultAuthMode(ctx context.Context) (string, error) {
	var config models.Config
	err := f.db.WithContext(ctx).
		Where("config_key = ?", "sys.auth.default.mode").
		First(&config).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 配置不存在，返回默认值
			return "local", nil
		}
		return "", err
	}

	// 验证配置值是否有效 (T-19-02: 验证mode参数)
	mode := config.ConfigValue
	if mode != "local" && mode != "ad" && mode != "hybrid" {
		return "local", nil // 无效值，返回默认
	}

	return mode, nil
}

// getADConfigID 获取AD配置ID
// 优先从sys_config读取，不存在时使用第一个启用的AD配置
func (f *AuthStrategyFactory) getADConfigID() (string, error) {
	// 1. 尝试从sys_config读取
	var config models.Config
	err := f.db.Where("config_key = ?", "sys.auth.ad.config_id").First(&config).Error

	if err == nil {
		// 配置存在，检查值是否为空
		if config.ConfigValue != "" {
			return config.ConfigValue, nil
		}
		// 配置值为空，继续尝试查找可用配置
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	// 2. sys_config中没有配置或配置值为空，尝试使用第一个启用的AD配置
	var adConfig models.ADConfig
	err = f.db.Where("status = 0").First(&adConfig).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("没有找到可用的AD配置，请在AD域管理中添加配置")
		}
		return "", err
	}

	return adConfig.ID, nil
}
