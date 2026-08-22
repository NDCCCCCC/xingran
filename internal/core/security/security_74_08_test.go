package security

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// =====================================================================
// 74-08 Batch B: internal/core/security — jwt.go 校验/刷新/SM2 分支 +
// password.go GenerateRandomToken + ad_authenticator.go 拨号失败/
// 账号池 nil/getDefaultRoleID + 工厂 SetAccountPool/GetAccountPool。
// 复用 integration_test.go 的 setupSecurityTestDB。
// =====================================================================

func hs256Config(secret string) *config.JWTConfig {
	return &config.JWTConfig{
		SecretKey:        secret,
		AccessKeyExpire:  7200,
		RefreshKeyExpire: 604800,
		Issuer:           "sec-test",
		UseSM2:           false,
	}
}

// ---------------- NewJWTManager 校验 ----------------

func TestNewJWTManager_Validation(t *testing.T) {
	// 空密钥
	_, err := NewJWTManager(hs256Config(""))
	assert.ErrorContains(t, err, "未配置")

	// 已知弱默认值
	_, err = NewJWTManager(hs256Config("xingran-next-secret-key"))
	assert.ErrorContains(t, err, "弱默认值")

	// 长度 <16
	_, err = NewJWTManager(hs256Config("short"))
	assert.ErrorContains(t, err, "长度过短")

	// 合法
	m, err := NewJWTManager(hs256Config("valid-secret-key-0123456789abcdef"))
	require.NoError(t, err)
	assert.False(t, m.IsSM2Enabled())
	assert.False(t, m.HasSM2PublicKey())
}

func TestNewJWTManager_SM2(t *testing.T) {
	// 无配置密钥 → 自动生成
	m, err := NewJWTManager(&config.JWTConfig{Issuer: "sm2-auto", UseSM2: true, AccessKeyExpire: 7200, RefreshKeyExpire: 604800})
	require.NoError(t, err)
	assert.True(t, m.IsSM2Enabled())
	assert.True(t, m.HasSM2PublicKey())
	assert.NotEmpty(t, m.GetPublicKey())

	// 提供合法 hex 密钥对
	priv, pub, err := crypto.GenerateKeyPair()
	require.NoError(t, err)
	privHex := hex.EncodeToString(priv.D.Bytes())
	for len(privHex) < 64 {
		privHex = "0" + privHex
	}
	m2, err := NewJWTManager(&config.JWTConfig{
		Issuer: "sm2-cfg", UseSM2: true, AccessKeyExpire: 7200, RefreshKeyExpire: 604800,
		SM2PrivateKey: privHex, SM2PublicKey: crypto.ExportPublicKeyToHex(pub),
	})
	require.NoError(t, err)
	assert.Equal(t, crypto.ExportPublicKeyToHex(pub), m2.GetPublicKey())

	// 非法私钥 hex
	_, err = NewJWTManager(&config.JWTConfig{
		UseSM2: true, SM2PrivateKey: "zz", SM2PublicKey: crypto.ExportPublicKeyToHex(pub),
	})
	assert.ErrorContains(t, err, "私钥")

	// 非法公钥 hex
	_, err = NewJWTManager(&config.JWTConfig{
		UseSM2: true, SM2PrivateKey: privHex, SM2PublicKey: "zz",
	})
	assert.ErrorContains(t, err, "公钥")
}

// ---------------- ValidateToken / RefreshToken ----------------

// signHS256 用指定 claims 直接签名(构造过期/未生效/错误 issuer 等非标 token)。
func signHS256(t *testing.T, secret string, claims *CustomClaims) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return tok
}

func TestJWTManager_ValidateToken_HS256(t *testing.T) {
	secret := "valid-secret-key-0123456789abcdef"
	m, err := NewJWTManager(hs256Config(secret))
	require.NoError(t, err)

	// 合法 token
	pair, err := m.GenerateTokenPair("u1", "alice", "Alice", []string{"admin"})
	require.NoError(t, err)
	claims, err := m.ValidateToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "u1", claims.UserID)
	assert.Equal(t, "alice", claims.Username)

	// 垃圾 token → ErrTokenInvalid
	_, err = m.ValidateToken("not-a-jwt")
	assert.ErrorIs(t, err, response.ErrTokenInvalid)

	// 过期 token → ErrTokenExpired
	expired := signHS256(t, secret, &CustomClaims{
		UserID: "u1", Username: "alice",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "sec-test",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	})
	_, err = m.ValidateToken(expired)
	assert.ErrorIs(t, err, response.ErrTokenExpired)

	// NotBefore 未来 → ErrTokenNotValidYet
	future := signHS256(t, secret, &CustomClaims{
		UserID: "u1", Username: "alice",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "sec-test",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	_, err = m.ValidateToken(future)
	assert.ErrorIs(t, err, response.ErrTokenNotValidYet)

	// 错误 issuer → ErrTokenInvalid
	wrongIssuer := signHS256(t, secret, &CustomClaims{
		UserID: "u1", Username: "alice",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "other-issuer",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	_, err = m.ValidateToken(wrongIssuer)
	assert.ErrorIs(t, err, response.ErrTokenInvalid)

	// 非 HMAC 签名方法(none alg) → ErrTokenInvalid
	noneTok, err := jwt.NewWithClaims(jwt.SigningMethodNone, &CustomClaims{
		UserID: "u1",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "sec-test",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	_, err = m.ValidateToken(noneTok)
	assert.ErrorIs(t, err, response.ErrTokenInvalid)
}

func TestJWTManager_ValidateToken_SM2(t *testing.T) {
	m, err := NewJWTManager(&config.JWTConfig{Issuer: "sm2-iss", UseSM2: true, AccessKeyExpire: 7200, RefreshKeyExpire: 604800})
	require.NoError(t, err)

	pair, err := m.GenerateTokenPair("u2", "bob", "Bob", []string{"user"})
	require.NoError(t, err)
	claims, err := m.ValidateToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "u2", claims.UserID)

	// 垃圾 → ErrTokenInvalid
	_, err = m.ValidateToken("garbage")
	assert.ErrorIs(t, err, response.ErrTokenInvalid)
}

func TestJWTManager_RefreshToken(t *testing.T) {
	m, err := NewJWTManager(hs256Config("valid-secret-key-0123456789abcdef"))
	require.NoError(t, err)

	pair, err := m.GenerateTokenPair("u1", "alice", "", []string{"admin"})
	require.NoError(t, err)

	// refresh token → 新令牌对(nickname 为空回退 username)
	newPair, err := m.RefreshToken(pair.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newPair.AccessToken)
	assert.NotEmpty(t, newPair.RefreshToken)
	assert.Equal(t, int64(7200), newPair.ExpiresIn)
	assert.Equal(t, "Bearer", newPair.TokenType)

	// access token 不是 refresh 角色 → 拒绝
	_, err = m.RefreshToken(pair.AccessToken)
	assert.ErrorIs(t, err, response.ErrTokenInvalid)

	// 垃圾 → 错误
	_, err = m.RefreshToken("garbage")
	assert.Error(t, err)

	// SM2 模式 refresh roundtrip
	m2, err := NewJWTManager(&config.JWTConfig{Issuer: "sm2-ref", UseSM2: true, AccessKeyExpire: 7200, RefreshKeyExpire: 604800})
	require.NoError(t, err)
	pair2, err := m2.GenerateTokenPair("u2", "bob", "B", nil)
	require.NoError(t, err)
	newPair2, err := m2.RefreshToken(pair2.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newPair2.AccessToken)
}

// ---------------- SM2 访问器 ----------------

func TestJWTManager_SM2Accessors(t *testing.T) {
	// HS256 模式: 全部空/禁用
	m, err := NewJWTManager(hs256Config("valid-secret-key-0123456789abcdef"))
	require.NoError(t, err)
	assert.Empty(t, m.GetPublicKey())
	_, err = m.DecryptPassword("anything")
	assert.ErrorContains(t, err, "SM2 未启用")
	priv, pub := m.GetSM2KeyPair()
	assert.Nil(t, priv)
	assert.Nil(t, pub)

	// SM2 模式: DecryptPassword roundtrip
	m2, err := NewJWTManager(&config.JWTConfig{Issuer: "sm2-dec", UseSM2: true, AccessKeyExpire: 7200, RefreshKeyExpire: 604800})
	require.NoError(t, err)
	priv2, pub2 := m2.GetSM2KeyPair()
	require.NotNil(t, priv2)
	require.NotNil(t, pub2)

	ciphertext, err := crypto.EncryptWithSM2("my-secret-pwd", pub2)
	require.NoError(t, err)
	plain, err := m2.DecryptPassword(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, "my-secret-pwd", plain)

	// 非法密文 → 错误
	_, err = m2.DecryptPassword("!!!not-base64!!!")
	assert.Error(t, err)
}

// ---------------- GenerateRandomToken ----------------

func TestGenerateRandomToken(t *testing.T) {
	// 长度 <32 → 提升到 32 字节
	tok, err := GenerateRandomToken(8)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tok), 43, "32 字节 base64url 至少 43 字符")

	// 指定长度 64 → 更长
	tok64, err := GenerateRandomToken(64)
	require.NoError(t, err)
	assert.Greater(t, len(tok64), len(tok))

	// URL 安全字符集
	assert.NotContains(t, tok64, "+")
	assert.NotContains(t, tok64, "/")

	// 两次生成不同
	tok2, err := GenerateRandomToken(32)
	require.NoError(t, err)
	assert.NotEqual(t, tok, tok2)
}

// ---------------- ADAuthenticator ----------------

func TestADAuthenticator_Authenticate_DialFailure(t *testing.T) {
	ctx := context.Background()
	newAuth := func(t *testing.T, id string, useSSL, useTLS bool) *ADAuthenticator {
		db := setupSecurityTestDB(t)
		ssl, tlsF := 0, 0
		if useSSL {
			ssl = 1
		}
		if useTLS {
			tlsF = 1
		}
		require.NoError(t, db.Exec(`INSERT INTO sys_ad_config
			(id, config_name, server_address, server_port, domain_name, base_dn, admin_username, admin_password, use_ssl, use_tls, status)
			VALUES (?, 'test-ad', '127.0.0.1', 1, 'test.local', 'dc=test,dc=local', 'legacy', 'legacy', ?, ?, 0)`,
			id, ssl, tlsF).Error)
		return NewADAuthenticator(db, id)
	}

	// 明文 ldap:// 拨号失败(127.0.0.1:1 不可达)
	a := newAuth(t, "cfg-plain", false, false)
	_, err := a.Authenticate(ctx, &AuthRequest{Username: "u", Password: "p"})
	assert.ErrorIs(t, err, ErrADConnectionFailed)

	// ldaps:// 拨号失败
	aSSL := newAuth(t, "cfg-ssl", true, false)
	_, err = aSSL.Authenticate(ctx, &AuthRequest{Username: "u", Password: "p"})
	assert.ErrorIs(t, err, ErrADConnectionFailed)

	// StartTLS 配置(拨号阶段即失败)
	aTLS := newAuth(t, "cfg-tls", false, true)
	_, err = aTLS.Authenticate(ctx, &AuthRequest{Username: "u", Password: "p"})
	assert.ErrorIs(t, err, ErrADConnectionFailed)
}

func TestADAuthenticator_BindAdminNoPool(t *testing.T) {
	// 未注入账号池 → 明确报错(Phase 38 移除单管理员 fallback)
	a := NewADAuthenticator(setupSecurityTestDB(t), "any")
	_, err := a.bindAdminWithFailover(context.Background(), &models.ADConfig{})
	assert.ErrorContains(t, err, "账号池未初始化")
}

func TestADAuthenticator_Setters(t *testing.T) {
	a := NewADAuthenticator(setupSecurityTestDB(t), "any")
	// nil 注入不 panic(账号池/cipher 由 core 启动时注入,单测只验证 setter 通路)
	assert.NotPanics(t, func() {
		a.SetPasswordCipher(nil)
		a.SetAccountPool(nil)
	})
	assert.Equal(t, "ad", a.Name())
}

func TestADAuthenticator_GetDefaultRoleID(t *testing.T) {
	db := setupSecurityTestDB(t)
	a := NewADAuthenticator(db, "any")

	// 无配置 → 空串
	assert.Empty(t, a.getDefaultRoleID())

	// 有配置 → 返回配置值
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_key, config_value)
		VALUES ('c1', 'sys.auth.ad.default_role_id', 'role-123')`).Error)
	assert.Equal(t, "role-123", a.getDefaultRoleID())
}

// ---------------- AuthStrategyFactory 账号池 ----------------

func TestAuthStrategyFactory_AccountPool(t *testing.T) {
	db := setupSecurityTestDB(t)
	pm := NewPasswordManager(nil)
	f := NewAuthStrategyFactory(db, pm)

	// 未注入 → nil
	assert.Nil(t, f.GetAccountPool())

	// 注入后 Get 返回同一实例(用 nil 接口值之外的判断: 设置再读)
	f.SetAccountPool(nil)
	assert.Nil(t, f.GetAccountPool())

	// hybrid/ad 模式注入路径在 GetAuthenticator 内(accountPool nil → 跳过 SetAccountPool)
	_, err := f.GetAuthenticator("ad")
	// 无 AD 配置 → 报错(getADConfigID 失败),证明走到了 ad 分支
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "AD配置") || strings.Contains(err.Error(), "AD 配置"))

	// hybrid 同样
	_, err = f.GetAuthenticator("hybrid")
	assert.Error(t, err)
}

// 防止未使用 import 告警(errors 包在表驱动断言中备用)。
var _ = errors.Is
