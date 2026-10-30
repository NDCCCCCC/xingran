package v1

import (
	"net/http"
	"net/url"
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

			// v129-recheck WR-05: HasPrefix 前缀匹配可被子域绕过
			// （http://example.com.attacker.com 前缀命中 example.com）——改 url.Parse
			// 后精确比较 host:port。Origin 头不含 path，精确 host 匹配即正确语义。
			originURL, perr := url.Parse(origin)
			if perr != nil || originURL.Host == "" {
				applogger.Warnf("WebSocket 连接被拒绝（Origin 解析失败）: origin=%s, client_ip=%s", origin, r.RemoteAddr)
				return false
			}
			originHost := strings.ToLower(originURL.Host)

			// 允许同源请求（Origin host 与请求 Host 精确一致）
			host := r.Header.Get("Host")
			if host == "" {
				host = r.Host
			}
			if host != "" && originHost == strings.ToLower(host) {
				return true
			}

			// 允许 localhost（开发环境，精确 host + 任意端口形态）
			if originHost == "localhost" || originHost == "127.0.0.1" ||
				strings.HasPrefix(originHost, "localhost:") || strings.HasPrefix(originHost, "127.0.0.1:") {
				return true
			}

			// 配置白名单：解析后精确比较 host（配置项也允许裸 host 形态）
			for _, allowed := range allowedOrigins {
				if strings.EqualFold(origin, allowed) {
					return true
				}
				allowedURL, aerr := url.Parse(allowed)
				if aerr == nil && allowedURL.Host != "" {
					if originHost == strings.ToLower(allowedURL.Host) {
						return true
					}
				} else if originHost == strings.ToLower(allowed) {
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
		hub.RegisterClient(userID, conn)

		// v129-recheck C-5: 此处不得再起读循环或直写 conn——RegisterClient 内部
		// 已启动 Client.readPump（唯一读者：ReadDeadline/Pong/断连注销）与
		// Client.writePump（唯一写者：54s 协议级 ping + 广播）。gorilla/websocket
		// 约束单读者/单写者，历史遗留的 handler 读循环构成第二读者，其文本
		// ping/pong 直写构成第二写者（并发写会 panic("concurrent write to
		// websocket connection") 且无 recover，进程崩溃）。keep-alive 全权由
		// hub 泵承担（协议级 ping/pong，浏览器自动应答，无需应用层文本心跳）。

		applogger.Warnf("用户 %s WebSocket连接已建立", userID)
	})
}
