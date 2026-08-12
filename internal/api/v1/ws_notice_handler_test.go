package v1

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWebSocketUpgrader_CheckOrigin 测试 CheckOrigin 验证逻辑
func TestWebSocketUpgrader_CheckOrigin(t *testing.T) {
	t.Run("允许所有来源", func(t *testing.T) {
		upgrader := newWebSocketUpgrader([]string{"*"})
		req := &http.Request{Header: http.Header{}, Host: "localhost:9000"}
		req.Header.Set("Origin", "http://evil.com")

		// 允许所有来源时，任何 origin 都应该返回 true
		result := upgrader.CheckOrigin(req)
		assert.True(t, result, "allowAll 模式应该允许所有来源")
	})

	t.Run("同源请求允许", func(t *testing.T) {
		upgrader := newWebSocketUpgrader([]string{"http://example.com"})
		req := &http.Request{Header: http.Header{}, Host: "example.com:9000"}
		req.Header.Set("Origin", "http://example.com:9000")

		result := upgrader.CheckOrigin(req)
		assert.True(t, result, "同源请求应该被允许")
	})

	t.Run("localhost 允许", func(t *testing.T) {
		upgrader := newWebSocketUpgrader([]string{})
		req := &http.Request{Header: http.Header{}, Host: "localhost:9000"}
		req.Header.Set("Origin", "http://localhost:8080")

		result := upgrader.CheckOrigin(req)
		assert.True(t, result, "localhost 应该被允许")
	})

	t.Run("127.0.0.1 允许", func(t *testing.T) {
		upgrader := newWebSocketUpgrader([]string{})
		req := &http.Request{Header: http.Header{}, Host: "localhost:9000"}
		req.Header.Set("Origin", "http://127.0.0.1:8080")

		result := upgrader.CheckOrigin(req)
		assert.True(t, result, "127.0.0.1 应该被允许")
	})

	t.Run("白名单匹配", func(t *testing.T) {
		upgrader := newWebSocketUpgrader([]string{"https://example.com"})
		req := &http.Request{Header: http.Header{}, Host: "localhost:9000"}
		req.Header.Set("Origin", "https://example.com")

		result := upgrader.CheckOrigin(req)
		assert.True(t, result, "白名单匹配应该被允许")
	})

	t.Run("白名单不匹配拒绝", func(t *testing.T) {
		upgrader := newWebSocketUpgrader([]string{"https://example.com"})
		req := &http.Request{Header: http.Header{}, Host: "localhost:9000"}
		req.Header.Set("Origin", "http://evil.com")

		result := upgrader.CheckOrigin(req)
		assert.False(t, result, "白名单不匹配应该被拒绝")
	})

	t.Run("空 Origin 允许", func(t *testing.T) {
		upgrader := newWebSocketUpgrader([]string{})
		req := &http.Request{Header: http.Header{}, Host: "localhost:9000"}
		// 不设置 Origin 头

		result := upgrader.CheckOrigin(req)
		assert.True(t, result, "空 Origin 应该被允许（非浏览器客户端）")
	})
}
