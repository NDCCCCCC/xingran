package v1

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	gorilla_ws "github.com/ruoyi-next/ruoyi-go-backend/internal/websocket"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// newWebSocketUpgrader 创建 WebSocket 升级器
// allowedOrigins: 允许的来源列表，空或含 "*" 表示允许所有（开发环境）
func newWebSocketUpgrader(allowedOrigins []string) websocket.Upgrader {
	allowAll := len(allowedOrigins) == 0 || containsOrigin(allowedOrigins, "*")

	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if allowAll {
				return true
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // 非浏览器客户端（如Postman）无Origin头
			}

			// 允许同源请求
			host := r.Header.Get("Host")
			if host == "" {
				host = r.Host
			}
			if strings.HasPrefix(origin, "http://"+host) || strings.HasPrefix(origin, "https://"+host) {
				return true
			}

			// 允许 localhost（开发环境）和配置的域名
			if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
				return true
			}
			for _, allowed := range allowedOrigins {
				if origin == allowed || strings.HasPrefix(origin, allowed) {
					return true
				}
			}

			return false
		},
	}
}

// containsOrigin 检查来源列表是否包含指定值
func containsOrigin(origins []string, target string) bool {
	for _, o := range origins {
		if o == target {
			return true
		}
	}
	return false
}

// SetupNoticeWebSocketRouter 设置通知WebSocket路由
// allowedOrigins: 允许的来源列表，空或含 "*" 表示允许所有（开发环境）
func SetupNoticeWebSocketRouter(r *gin.RouterGroup, hub *gorilla_ws.NoticeHub, core *core.Core, allowedOrigins []string) {
	upgrader := newWebSocketUpgrader(allowedOrigins)

	r.GET("/notices", func(c *gin.Context) {
		// 从query或header获取token
		token := c.Query("token")
		if token == "" {
			token = c.GetHeader("Authorization")
			// 移除 "Bearer " 前缀
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
		}

		// 验证token并获取user_id
		claims, err := core.JWTManager.ValidateToken(token)
		if err != nil {
			response.Error(c, apperrors.Unauthorized())
			return
		}

		userID := claims.UserID

		// 升级连接
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			applogger.Warnf("WebSocket升级失败: %v", err)
			return
		}

		// 注册客户端
		client := hub.RegisterClient(userID, conn)

		// 读取客户端消息（用于保持连接）
		go func() {
			defer hub.UnregisterClient(client)
			for {
				messageType, message, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						applogger.Warnf("WebSocket错误: %v", err)
					}
					break
				}

				// 处理客户端消息（如果需要）
				if messageType == websocket.TextMessage {
					// 心跳响应
					if string(message) == "ping" {
						_ = conn.WriteMessage(websocket.TextMessage, []byte("pong"))
					}
				}
			}
		}()

		applogger.Warnf("用户 %s WebSocket连接已建立", userID)
	})
}
