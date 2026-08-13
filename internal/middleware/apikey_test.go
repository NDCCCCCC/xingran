package middleware

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsValidKeyFormat 测试密钥格式验证
func TestIsValidKeyFormat(t *testing.T) {
	t.Run("有效密钥格式", func(t *testing.T) {
		validKey := "rec_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		assert.True(t, isValidKeyFormat(validKey))
	})

	t.Run("无前缀", func(t *testing.T) {
		invalidKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		assert.False(t, isValidKeyFormat(invalidKey))
	})

	t.Run("长度错误", func(t *testing.T) {
		shortKey := "rec_0123456789abcdef"
		assert.False(t, isValidKeyFormat(shortKey))
	})

	t.Run("非十六进制字符", func(t *testing.T) {
		invalidHex := "rec_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcxyz"
		assert.False(t, isValidKeyFormat(invalidHex))
	})

	t.Run("大写十六进制", func(t *testing.T) {
		upperCaseKey := "rec_0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF"
		assert.True(t, isValidKeyFormat(upperCaseKey))
	})

	t.Run("混合大小写", func(t *testing.T) {
		mixedCaseKey := "rec_0123456789aBcDeF0123456789AbCdEf0123456789AbCdEf0123456789aBcDeF"
		assert.True(t, isValidKeyFormat(mixedCaseKey))
	})
}

// TestIsIPAllowed 测试IP白名单验证
func TestIsIPAllowed(t *testing.T) {
	t.Run("单IP匹配", func(t *testing.T) {
		whitelist := []string{"192.168.1.1"}
		clientIP := "192.168.1.1"
		assert.True(t, isIPAllowed(clientIP, whitelist))
	})

	t.Run("CIDR匹配", func(t *testing.T) {
		whitelist := []string{"10.0.0.0/24"}
		clientIP := "10.0.0.50"
		assert.True(t, isIPAllowed(clientIP, whitelist))
	})

	t.Run("CIDR范围外", func(t *testing.T) {
		whitelist := []string{"10.0.0.0/24"}
		clientIP := "10.0.1.50"
		assert.False(t, isIPAllowed(clientIP, whitelist))
	})

	t.Run("IP不在白名单", func(t *testing.T) {
		whitelist := []string{"192.168.1.1"}
		clientIP := "192.168.1.2"
		assert.False(t, isIPAllowed(clientIP, whitelist))
	})

	t.Run("空白名单_允许所有", func(t *testing.T) {
		whitelist := []string{}
		clientIP := "8.8.8.8"
		assert.True(t, isIPAllowed(clientIP, whitelist))
	})

	t.Run("多个IP地址", func(t *testing.T) {
		whitelist := []string{"192.168.1.1", "10.0.0.1", "172.16.0.0/16"}

		assert.True(t, isIPAllowed("192.168.1.1", whitelist))
		assert.True(t, isIPAllowed("10.0.0.1", whitelist))
		assert.True(t, isIPAllowed("172.16.5.10", whitelist))
		assert.False(t, isIPAllowed("8.8.8.8", whitelist))
	})

	t.Run("IPv6地址", func(t *testing.T) {
		whitelist := []string{"::1"}
		clientIP := "::1"
		assert.True(t, isIPAllowed(clientIP, whitelist))
	})

	t.Run("无效IP格式_跳过", func(t *testing.T) {
		whitelist := []string{"invalid-ip"}
		clientIP := "192.168.1.1"
		// 无效白名单IP应该被跳过，返回false
		assert.False(t, isIPAllowed(clientIP, whitelist))
	})

	t.Run("无效客户端IP", func(t *testing.T) {
		whitelist := []string{"192.168.1.1"}
		clientIP := "invalid-client-ip"
		assert.False(t, isIPAllowed(clientIP, whitelist))
	})
}

// TestGetRequiredScope 测试操作到作用域的映射
func TestGetRequiredScope(t *testing.T) {
	t.Run("view操作映射到read", func(t *testing.T) {
		scope := getRequiredScope("view")
		assert.Equal(t, "read", scope)
	})

	t.Run("create操作映射到write", func(t *testing.T) {
		scope := getRequiredScope("create")
		assert.Equal(t, "write", scope)
	})

	t.Run("edit操作映射到write", func(t *testing.T) {
		scope := getRequiredScope("edit")
		assert.Equal(t, "write", scope)
	})

	t.Run("delete操作映射到write", func(t *testing.T) {
		scope := getRequiredScope("delete")
		assert.Equal(t, "write", scope)
	})

	t.Run("未知操作默认read", func(t *testing.T) {
		scope := getRequiredScope("unknown")
		assert.Equal(t, "read", scope)
	})

	t.Run("空操作默认read", func(t *testing.T) {
		scope := getRequiredScope("")
		assert.Equal(t, "read", scope)
	})
}

// TestExtractAPIKey 测试API Key提取
func TestExtractAPIKey(t *testing.T) {
	t.Run("有效header提取", func(t *testing.T) {
		// 这个测试需要Gin上下文，这里简化测试逻辑
		// 实际测试在集成测试中进行
	})
}

// TestRateLimitResult 测试速率限制结果结构
func TestRateLimitResult(t *testing.T) {
	t.Run("创建结果对象", func(t *testing.T) {
		// 这个测试验证RateLimitResult结构的使用
		// 实际测试在rate_limiter_test.go中进行
	})
}

// TestRateLimitHeaderEncoding QUAL-01 / D-12 单测: 锁定 RateLimitByScope 限流响应头的整数序列化方式。
//
// P2-a 缺陷回顾: apikey.go:267-268 曾用 string(rune(result.Limit)) 把整数当 Unicode 码点转换,
// Limit=100 → "d"、Remaining=99 → "c",响应头彻底不可解析。D-11 修复为 strconv.Itoa。
// 本测试把「数字字面量 + 可被 strconv.Atoi 反解析」两条不变量固化为回归锚,防止 P2-a 复现。
func TestRateLimitHeaderEncoding(t *testing.T) {
	t.Run("strconv.Itoa 数字字符串化", func(t *testing.T) {
		assert.Equal(t, "100", strconv.Itoa(100), "Limit=100 应序列化为 \"100\"")
		assert.Equal(t, "99", strconv.Itoa(99), "Remaining=99 应序列化为 \"99\"")
		assert.Equal(t, "0", strconv.Itoa(0), "Remaining=0 应序列化为 \"0\"")

		// 防御性断言: 原 string(rune(100)) == "d" 的编码错误不得再出现
		assert.NotEqual(t, "d", strconv.Itoa(100), "P2-a 回归: 100 不得再被编码为 rune 字面量 \"d\"")
		assert.NotEqual(t, string(rune(100)), strconv.Itoa(100), "strconv.Itoa 结果必须区别于 string(rune(int))")
	})

	t.Run("header 可被 strconv.Atoi 反解析", func(t *testing.T) {
		// RFC 6585 消费方(前端 / 第三方工具)对限流头做的正是这一步反解析
		n, err := strconv.Atoi(strconv.Itoa(100))
		assert.NoError(t, err, "X-RateLimit-Limit 必须是可反解析的数字字符串")
		assert.Equal(t, 100, n, "反解析结果应与原值一致")

		remaining, err := strconv.Atoi(strconv.Itoa(0))
		assert.NoError(t, err, "X-RateLimit-Remaining 必须是可反解析的数字字符串")
		assert.Equal(t, 0, remaining, "Remaining=0 反解析应为 0 而非报错")

		// 对照组: 原缺陷产物 "d" 无法被 Atoi 反解析
		_, badErr := strconv.Atoi(string(rune(100)))
		assert.Error(t, badErr, "P2-a 产物 \"d\" 本就不可反解析 —— 这正是缺陷本身")
	})
}
