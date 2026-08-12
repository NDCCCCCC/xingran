package v1

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	gorilla_ws "github.com/xingran-next/xingran-go-backend/internal/websocket"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// newWebSocketUpgrader 创建 WebSocket 升级器
// allowedOrigins: 允许的来源列表
//   - 含 "*": 显式允许所有来源 (开发环境,会记录 Warn 日志)
//   - 空或具体来源: 严格模式,仅允许同源 + localhost + 显式列表
//
// F-07: 之前实现把 "空列表" 等同于 "allowAll=true",导致运维忘记配置
// 时所有来源都被放行 + 仍然接受 cookie/Authorization 凭据,
// 构成 CSRF 与 token 盗用风险。改为只有显式 "*" 才放行所有。
func newWebSocketUpgrader(allowedOrigins []string) websocket.Upgrader {
	allowAll := containsOrigin(allowedOrigins, "*")

	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if allowAll {
				// 生产环境警告：显式 "*" 允许所有来源是不安全的
				applogger.Warnf("WebSocket CheckOrigin 允许所有来源（显式 '*' 开发模式），建议生产环境配置具体白名单")
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

			// 记录拒绝的来源（安全审计）
			applogger.Warnf("WebSocket 连接被拒绝: origin=%s, client_ip=%s", origin, r.RemoteAddr)
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
