package rpa

import (
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// RPACredential RPA 登录凭证
type RPACredential struct {
	models.BaseModel

	// 基本信息
	Name         string `json:"name" gorm:"size:100;not null"`
	TargetSystem string `json:"targetSystem" gorm:"size:100;not null;index:idx_rpa_cred_system,priority:2"`
	TargetURL    string `json:"targetUrl" gorm:"size:500"`

	// 加密的凭证信息（使用SM4加密，这里存储加密后的base64字符串）
	UsernameEncrypted  string `json:"-" gorm:"column:username_encrypted;not null;type:text"`
	PasswordEncrypted  string `json:"-" gorm:"column:password_encrypted;not null;type:text"`
	ExtraDataEncrypted string `json:"-" gorm:"column:extra_data_encrypted;type:text"`

	// 明文仅用于创建/更新响应（不存储）
	Username  string                 `json:"username" gorm:"-"`
	Password  string                 `json:"password" gorm:"-"`
	ExtraData map[string]interface{} `json:"extraData" gorm:"-"`

	// 归属和权限
	UserID   string `json:"userId" gorm:"type:uuid;not null;index:idx_rpa_cred_user,priority:2"`
	DeptID   string `json:"deptId" gorm:"type:uuid;index:idx_rpa_cred_shared,priority:2"`
	IsShared bool   `json:"isShared" gorm:"default:false;index:idx_rpa_cred_shared,priority:3"`

	// 状态和元数据
	Status            int        `json:"status" gorm:"default:0;check:status IN (0,1)"`
	LastUsedAt        *time.Time `json:"lastUsedAt"`
	LastLoginAt       *time.Time `json:"lastLoginAt"`
	LoginSuccessCount int        `json:"loginSuccessCount" gorm:"default:0"`
	LoginFailCount    int        `json:"loginFailCount" gorm:"default:0"`
}

func (RPACredential) TableName() string {
	return "sys_rpa_credentials"
}

// RPASession RPA 会话（存储token/cookie）
type RPASession struct {
	models.BaseModel

	// 关联信息
	CredentialID string `json:"credentialId" gorm:"type:uuid;not null;index:idx_rpa_session_cred,priority:2"`
	ExecutionID  string `json:"executionId" gorm:"type:uuid;index:idx_rpa_session_exec"`

	// 目标系统信息
	TargetSystem string `json:"targetSystem" gorm:"size:100;not null;index:idx_rpa_session_system,priority:2"`
	TargetURL    string `json:"targetUrl" gorm:"size:500"`

	// 会话数据（加密存储）
	AccessTokenEncrypted  string `json:"-" gorm:"column:access_token_encrypted;type:text"`
	RefreshTokenEncrypted string `json:"-" gorm:"column:refresh_token_encrypted;type:text"`
	CookiesEncrypted      string `json:"-" gorm:"column:cookies_encrypted;type:text"`
	SessionDataEncrypted  string `json:"-" gorm:"column:session_data_encrypted;type:text"`

	// 明文仅用于响应
	AccessToken  string                 `json:"accessToken" gorm:"-"`
	RefreshToken string                 `json:"refreshToken" gorm:"-"`
	Cookies      []Cookie               `json:"cookies" gorm:"-"`
	SessionData  map[string]interface{} `json:"sessionData" gorm:"-"`

	// 过期和状态
	ExpiresAt     *time.Time `json:"expiresAt"`
	IsValid       bool       `json:"isValid" gorm:"default:true;index:idx_rpa_session_cred,priority:3;index:idx_rpa_session_system,priority:3"`
	InvalidReason string     `json:"invalidReason" gorm:"size:200"`

	// 关联
	Credential *RPACredential `json:"credential,omitempty" gorm:"foreignKey:CredentialID"`
}

func (RPASession) TableName() string {
	return "sys_rpa_sessions"
}

// Cookie HTTP Cookie
type Cookie struct {
	Name     string     `json:"name"`
	Value    string     `json:"value"`
	Domain   string     `json:"domain"`
	Path     string     `json:"path"`
	Expires  *time.Time `json:"expires"`
	Secure   bool       `json:"secure"`
	HTTPOnly bool       `json:"httpOnly"`
}

// CredentialCreateRequest 创建凭证请求
type CredentialCreateRequest struct {
	Name         string                 `json:"name" binding:"required"`
	TargetSystem string                 `json:"targetSystem" binding:"required"`
	TargetURL    string                 `json:"targetUrl"`
	Username     string                 `json:"username" binding:"required"`
	Password     string                 `json:"password" binding:"required"`
	ExtraData    map[string]interface{} `json:"extraData"`
	IsShared     bool                   `json:"isShared"`
}

// CredentialUpdateRequest 更新凭证请求
type CredentialUpdateRequest struct {
	Name      string                 `json:"name"`
	Username  string                 `json:"username"`
	Password  string                 `json:"password"`
	ExtraData map[string]interface{} `json:"extraData"`
	Status    *int                   `json:"status"`
	IsShared  *bool                  `json:"isShared"`
}

// SessionCreateRequest 创建会话请求
type SessionCreateRequest struct {
	CredentialID string                 `json:"credentialId" binding:"required"`
	ExecutionID  string                 `json:"executionId"`
	TargetSystem string                 `json:"targetSystem" binding:"required"`
	TargetURL    string                 `json:"targetUrl"`
	AccessToken  string                 `json:"accessToken"`
	RefreshToken string                 `json:"refreshToken"`
	Cookies      []Cookie               `json:"cookies"`
	SessionData  map[string]interface{} `json:"sessionData"`
	ExpiresAt    *time.Time             `json:"expiresAt"`
}

// CredentialListParams 凭证列表查询参数
type CredentialListParams struct {
	Current      int    `json:"current" form:"current"`
	PageSize     int    `json:"pageSize" form:"pageSize"`
	TargetSystem string `json:"targetSystem,omitempty" form:"targetSystem"`
	Status       *int   `json:"status,omitempty" form:"status"`
	IsShared     *bool  `json:"isShared,omitempty" form:"isShared"`
	MyCredOnly   bool   `json:"myCredOnly,omitempty" form:"myCredOnly"` // 仅查看我的凭证
}

// SessionListParams 会话列表查询参数
type SessionListParams struct {
	Current        int    `json:"current" form:"current"`
	PageSize       int    `json:"pageSize" form:"pageSize"`
	CredentialID   string `json:"credentialId,omitempty" form:"credentialId"`
	TargetSystem   string `json:"targetSystem,omitempty" form:"targetSystem"`
	IsValid        *bool  `json:"isValid,omitempty" form:"isValid"`
	IncludeInvalid bool   `json:"includeInvalid,omitempty" form:"includeInvalid"`
}
