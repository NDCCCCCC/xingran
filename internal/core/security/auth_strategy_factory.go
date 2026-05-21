package security

import (
	"context"
	"errors"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// AuthStrategyFactory 认证策略工厂
// 根据认证模式动态创建对应的认证器
type AuthStrategyFactory struct {
	db          *gorm.DB
	pwdManager  *PasswordManager
	userSyncer  UserSyncer // 用户同步服务（可选）
}

// NewAuthStrategyFactory 创建认证策略工厂
func NewAuthStrategyFactory(db *gorm.DB, pwdMgr *PasswordManager) *AuthStrategyFactory {
	return &AuthStrategyFactory{
		db:         db,
		pwdManager: pwdMgr,
	}
}

// SetUserSyncer 设置用户同步服务
func (f *AuthStrategyFactory) SetUserSyncer(syncer UserSyncer) {
	f.userSyncer = syncer
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
		return ad, nil

	case "hybrid":
		local := NewLocalAuthenticator(f.db, f.pwdManager)
		configID, err := f.getADConfigID()
		if err != nil {
			return nil, err
		}
		ad := NewADAuthenticator(f.db, configID)
		ad.SetUserSyncer(f.userSyncer)
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
		return config.ConfigValue, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	// 2. sys_config中没有配置，尝试使用第一个启用的AD配置
	var adConfig models.ADConfig
	err = f.db.Where("status = 0").First(&adConfig).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("没有找到可用的AD配置")
		}
		return "", err
	}

	return adConfig.ID, nil
}
