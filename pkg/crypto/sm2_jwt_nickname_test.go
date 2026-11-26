package crypto

// ============================================================================
// W7 / UI-FIX-02 回归用例：SM2 token 必须携带 nickname
//
// 缺陷实证（2026-11-26 生产 10.62.10.39 + 真实浏览器）：
//   登录后 access token 的 payload 只有 user_id / username / roles，没有 nickname。
//   而 pkg/middleware/auth.go 的 setUserContext 把 claims.Nickname 写进 gin
//   context，liveview_handler.go 的 operatorName() 因此在 nickname 为空时回落
//   username。前端 myName 用的却是登录响应里的 user.nickname（中文昵称）——
//   两边口径不一致，接管判据 `takeoverBy === myName` 恒不成立 →
//   接管成功但遮罩不消失（页面显示「已接管 · ninedrunk」而遮罩仍在）。
//
// 这三个用例把「签发→校验→字段保留」全链路钉住，任何一侧删字段都会立刻红。
// ============================================================================

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// decodeJWTPayload 取出 token 中间段的 JSON 字段（用于断言"payload 里真有什么"）。
func decodeJWTPayload(t *testing.T, token string) map[string]interface{} {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3, "JWT 必须是三段式")
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(payload, &out))
	return out
}

// TestSM2JWT_NicknameRoundTripClaims SM2 token 的 nickname 必须原样往返。
func TestSM2JWT_NicknameRoundTripClaims(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	require.NoError(t, err)

	claims := &Claims{
		UserID:   "u-nick-001",
		Username: "ninedrunk",
		Nickname: "超级管理员",
		Roles:    []string{"role-a"},
		Issuer:   "XingRan-Next",
		Subject:  "u-nick-001",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token, err := GenerateTokenWithSM2(claims, priv)
	require.NoError(t, err)

	got, err := ValidateTokenWithSM2(token, pub)
	require.NoError(t, err)
	require.Equal(t, "超级管理员", got.Nickname,
		"SM2 token 必须携带 nickname：缺它会让后端 operatorName 回落 username，"+
			"前端却用 user.nickname 判接管者 —— 两边口径不一致，遮罩不消失")
	require.Equal(t, "ninedrunk", got.Username, "username 仍必须保留")
}

// TestSM2JWT_NicknameInRawPayload 直接解 payload，防止序列化层被 omitempty 吃掉。
func TestSM2JWT_NicknameInRawPayload(t *testing.T) {
	priv, _, err := GenerateKeyPair()
	require.NoError(t, err)

	token, err := GenerateTokenWithSM2(&Claims{
		UserID:   "u-nick-002",
		Username: "ninedrunk",
		Nickname: "陈超",
		Roles:    []string{"role-b"},
		Issuer:   "XingRan-Next",
		Subject:  "u-nick-002",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}, priv)
	require.NoError(t, err)

	raw := decodeJWTPayload(t, token)
	require.Equal(t, "陈超", raw["nickname"],
		"payload 里必须真实存在 nickname 字段（不是只存在于结构体）")
	require.Equal(t, "ninedrunk", raw["username"])
	require.Equal(t, "u-nick-002", raw["user_id"])
}

// TestSM2JWT_EmptyNicknameOmitted 空昵称走 omitempty —— 不破坏既有 token 体积约定，
// 且读取端必须把「字段缺失」等价于空串（ValidateToken 反序列化天然如此）。
func TestSM2JWT_EmptyNicknameOmitted(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	require.NoError(t, err)

	token, err := GenerateTokenWithSM2(&Claims{
		UserID:   "u-nick-003",
		Username: "nobody",
		Issuer:   "XingRan-Next",
		Subject:  "u-nick-003",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}, priv)
	require.NoError(t, err)

	raw := decodeJWTPayload(t, token)
	_, hasNickname := raw["nickname"]
	require.False(t, hasNickname, "空昵称应被 omitempty 省略")
	got, err := ValidateTokenWithSM2(token, pub)
	require.NoError(t, err)
	require.Empty(t, got.Nickname, "字段缺失必须反序列化为空串（调用方据空串回落 username）")
}
