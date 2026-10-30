package security

import (
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/config"
)

// newSM2TestJWTManager 构造与生产 configs/config.yaml 对齐的 SM2 模式 JWTManager。
// 关键点：issuer 为配置值（非硬编码），use_sm2=true 且密钥为空 → 动态生成密钥对。
func newSM2TestJWTManager(t *testing.T, issuer string) *JWTManager {
	t.Helper()
	mgr, err := NewJWTManager(&config.JWTConfig{
		SecretKey:        "",
		AccessKeyExpire:  7200,
		RefreshKeyExpire: 604800,
		Issuer:           issuer,
		UseSM2:           true,
	})
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}
	return mgr
}

// TestSM2RefreshTokenValidatesWithConfiguredIssuer 复现 token-lost-on-refresh bug：
// SM2 模式下 refresh token 由 GenerateRefreshTokenWithSM2 生成，曾硬编码
// Issuer "Xingran-Next"，与配置 issuer（如 "XingRan-Next-Dev"）不一致，
// 导致 JWTManager.ValidateToken 的签发者校验恒定返回 ErrTokenInvalid(401)，
// 页面刷新时的 POST /system/auth/refresh 100% 失败，前端被迫回到登录页。
// 该测试模拟 auth.go refreshToken handler 的确切调用路径。
func TestSM2RefreshTokenValidatesWithConfiguredIssuer(t *testing.T) {
	const issuer = "XingRan-Next-Dev" // 与 configs/config.yaml jwt.issuer 一致

	mgr := newSM2TestJWTManager(t, issuer)

	pair, err := mgr.GenerateTokenPair("user-1", "admin", "管理员", []string{"role-1"})
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	// 对照组：access token 应当始终通过（其 issuer 来自配置）
	if _, err := mgr.ValidateToken(pair.AccessToken); err != nil {
		t.Fatalf("ValidateToken(accessToken) 意外失败（对照组）: %v", err)
	}

	// 核心断言：同进程签发的 refresh token 必须能通过 ValidateToken。
	// ValidateToken 内部已对签发者做 claims.Issuer == j.issuer 校验，
	// 通过即证明 token 的 iss 与配置 issuer 一致，无需再单独断言 issuer 字段。
	claims, err := mgr.ValidateToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateToken(refreshToken) 失败: %v — refresh token 签发者与配置 issuer 不一致会导致页面刷新恒定 401", err)
	}

	if len(claims.Roles) != 1 || claims.Roles[0] != "refresh" {
		t.Fatalf("refresh token roles = %v, want [refresh]", claims.Roles)
	}
}

// TestSM2RefreshTokenHonorsConfiguredExpiry 验证 refresh token 有效期跟随
// jwt.refresh_key_expire 配置而非硬编码 7 天（当前配置恰为 7 天，但机制上必须走配置）。
func TestSM2RefreshTokenHonorsConfiguredExpiry(t *testing.T) {
	mgr := newSM2TestJWTManager(t, "XingRan-Next-Dev")

	// 覆盖为非默认时长，若实现仍硬编码 7 天则此断言失败
	mgr.refreshKeyExpire = 30 * 24 * time.Hour

	pair, err := mgr.GenerateTokenPair("user-1", "admin", "管理员", []string{"role-1"})
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	claims, err := mgr.ValidateToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateToken(refreshToken) error = %v", err)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("refresh token 缺少 ExpiresAt")
	}
	gotDays := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time).Hours() / 24
	if gotDays < 29 || gotDays > 31 {
		t.Fatalf("refresh token 有效期 = %.1f 天, want 30 天（应跟随 refresh_key_expire 配置）", gotDays)
	}
}
