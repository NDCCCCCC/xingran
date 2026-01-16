package v1

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	gorilla_ws "github.com/ruoyi-next/ruoyi-go-backend/internal/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 生产环境需要验证来源
		return true
	},
}

// SetupNoticeWebSocketRouter 设置通知WebSocket路由
func SetupNoticeWebSocketRouter(r *gin.RouterGroup, hub *gorilla_ws.NoticeHub, core *core.Core) {
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			return
		}

		userID := claims.UserID

		// 升级连接
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket升级失败: %v", err)
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
						log.Printf("WebSocket错误: %v", err)
					}
					break
				}

				// 处理客户端消息（如果需要）
				if messageType == websocket.TextMessage {
					// 心跳响应
					if string(message) == "ping" {
						conn.WriteMessage(websocket.TextMessage, []byte("pong"))
					}
				}
			}
		}()

		log.Printf("用户 %s WebSocket连接已建立", userID)
	})
}
