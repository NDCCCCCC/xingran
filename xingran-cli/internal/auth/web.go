package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// DefaultWebLoginTimeout 浏览器交互登录默认超时。
const DefaultWebLoginTimeout = 5 * time.Minute

// WebLoginOptions InteractiveLogin 选项。
type WebLoginOptions struct {
	BaseURL     string        // 后端地址，如 http://localhost:9000（必填）
	Username    string        // 预填用户名（页面上仍可修改）
	Timeout     time.Duration // 整体超时，默认 5 分钟；到时服务关闭并返回错误
	OpenBrowser bool          // 自动打开系统浏览器；false 时仅打印 URL
	Encryptor   Encryptor     // 可选：cmd 层注入 crypto 实现；nil 时按 auto 模式明文
	Store       TokenStore    // 可选：token 存储；nil 用默认文件存储
	Requester   Requester     // 可选：测试注入；nil 内部构造 HTTPRequester
}

// InteractiveLogin 浏览器交互式登录（M5-web）。
//
// 流程：本机起一次性 HTTP 服务（仅绑 127.0.0.1，随机端口，URL 携带一次性
// session token 防其他本地进程伪造）→ 打开浏览器 → 用户在页面输入密码并完成
// 验证码（slider 拖动 / normal 字符输入）→ 服务端复用 FetchCaptcha/VerifySlider/
// Login 编排 → CLI 收到 TokenPair（Login 内部已持久化）→ 服务关闭。
// 密码只在浏览器与服务端内存间流转，不经过命令行参数 / shell 历史。
func InteractiveLogin(ctx context.Context, opts WebLoginOptions) (*TokenPair, error) {
	if opts.BaseURL == "" {
		return nil, errors.New("InteractiveLogin: BaseURL 不能为空")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultWebLoginTimeout
	}

	req := opts.Requester
	if req == nil {
		req = &HTTPRequester{BaseURL: opts.BaseURL, Encryptor: opts.Encryptor}
	}
	store := opts.Store
	if store == nil {
		store = NewFileTokenStore()
	}
	a := NewAuthenticator(req, store, opts.Encryptor, opts.BaseURL)
	cp, ok := a.(CaptchaProvider)
	if !ok {
		return nil, errors.New("InteractiveLogin: authenticator 未实现 CaptchaProvider")
	}

	w := &webLoginServer{
		authn: a,
		cp:    cp,
		token: randomToken(),
		page:  renderLoginPage(opts.Username),
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("本地服务监听失败: %w", err)
	}
	srv := &http.Server{Handler: w.routes(), ReadHeaderTimeout: 10 * time.Second}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	url := fmt.Sprintf("http://127.0.0.1:%d/?s=%s", ln.Addr().(*net.TCPAddr).Port, w.token)
	if opts.OpenBrowser {
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "无法自动打开浏览器(%v)，请手动访问:\n%s\n", err, url)
		} else {
			fmt.Fprintf(os.Stderr, "已打开浏览器: %s\n", url)
		}
	} else {
		fmt.Fprintf(os.Stderr, "请在浏览器打开以下地址完成登录:\n%s\n", url)
	}

	timer := time.NewTimer(opts.Timeout)
	defer timer.Stop()
	select {
	case pair := <-w.done:
		_ = srv.Shutdown(context.Background())
		return pair, nil
	case err := <-serveErr:
		return nil, fmt.Errorf("本地服务异常退出: %w", err)
	case <-ctx.Done():
		_ = srv.Shutdown(context.Background())
		return nil, fmt.Errorf("交互登录被取消: %w", ctx.Err())
	case <-timer.C:
		_ = srv.Shutdown(context.Background())
		return nil, errors.New("交互登录超时（服务已关闭），请重试")
	}
}

// webLoginServer 一次性本地登录服务。
type webLoginServer struct {
	authn Authenticator
	cp    CaptchaProvider
	token string
	page  string
	done  chan *TokenPair
}

func newWebLoginServer(a Authenticator, cp CaptchaProvider, page string) *webLoginServer {
	return &webLoginServer{authn: a, cp: cp, token: randomToken(), page: page, done: make(chan *TokenPair, 1)}
}

// randomToken 生成 256bit hex 一次性 session token。
func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand 不可用: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// checkSession 校验 URL 参数中的 session token。
func (w *webLoginServer) checkSession(r *http.Request) bool {
	return r.URL.Query().Get("s") == w.token
}

func (w *webLoginServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", w.handlePage)
	mux.HandleFunc("/captcha", w.handleCaptcha)
	mux.HandleFunc("/login", w.handleLogin)
	return mux
}

// handlePage 登录页（校验 session token）。
func (w *webLoginServer) handlePage(rw http.ResponseWriter, r *http.Request) {
	if !w.checkSession(r) {
		http.Error(rw, "无效的会话令牌，请从 CLI 输出的地址进入", http.StatusForbidden)
		return
	}
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = rw.Write([]byte(w.page))
}

// captchaPayload GET /captcha 响应。
type captchaPayload struct {
	Mode       string `json:"mode"` // disabled|normal|slider
	CaptchaID  string `json:"captchaId,omitempty"`
	Token      string `json:"token,omitempty"`
	CaptchaImg string `json:"captchaImg,omitempty"`
	SliderImg  string `json:"sliderImg,omitempty"`
	PieceImg   string `json:"pieceImg,omitempty"`
	YPos       int    `json:"yPos,omitempty"`
}

// handleCaptcha 拉取新验证码（页面加载/重试时调用）。
func (w *webLoginServer) handleCaptcha(rw http.ResponseWriter, r *http.Request) {
	if !w.checkSession(r) {
		http.Error(rw, "无效的会话令牌", http.StatusForbidden)
		return
	}
	info, err := w.cp.FetchCaptcha(r.Context())
	if err != nil {
		writeJSON(rw, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	if info == nil {
		writeJSON(rw, http.StatusOK, captchaPayload{Mode: CaptchaTypeNone})
		return
	}
	writeJSON(rw, http.StatusOK, captchaPayload{
		Mode:       info.CaptchaType,
		CaptchaID:  info.CaptchaID,
		Token:      info.Token,
		CaptchaImg: info.CaptchaImg,
		SliderImg:  info.SliderImg,
		PieceImg:   info.PieceImg,
		YPos:       info.YPos,
	})
}

// loginRequest POST /login 请求体（浏览器提交）。
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	SliderX  int    `json:"sliderX,omitempty"`
	Code     string `json:"code,omitempty"`
}

// handleLogin 登录编排：FetchCaptcha → (slider: VerifySlider) → Login。
// 失败返回 {error}，前端展示并可重试（重试会重新拉验证码）。
func (w *webLoginServer) handleLogin(rw http.ResponseWriter, r *http.Request) {
	if !w.checkSession(r) {
		http.Error(rw, "无效的会话令牌", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var lr loginRequest
	if err := json.NewDecoder(r.Body).Decode(&lr); err != nil {
		writeJSON(rw, http.StatusBadRequest, map[string]any{"error": "请求体解析失败"})
		return
	}
	if lr.Username == "" || lr.Password == "" {
		writeJSON(rw, http.StatusBadRequest, map[string]any{"error": "请输入用户名和密码"})
		return
	}

	// 每次登录尝试都取新验证码（失败重试天然刷新 captchaId）
	info, err := w.cp.FetchCaptcha(r.Context())
	if err != nil {
		writeJSON(rw, http.StatusBadGateway, map[string]any{"error": "获取验证码失败: " + err.Error()})
		return
	}

	input := CaptchaInput{Type: CaptchaTypeNone}
	if info != nil {
		switch info.CaptchaType {
		case CaptchaTypeSlider:
			if lr.SliderX <= 0 {
				writeJSON(rw, http.StatusBadRequest, map[string]any{"error": "请先拖动拼图到缺口位置"})
				return
			}
			input = CaptchaInput{
				Type:      CaptchaTypeSlider,
				SliderX:   lr.SliderX,
				CaptchaID: info.CaptchaID,
				Token:     info.Token,
			}
		case CaptchaTypeNormal:
			if lr.Code == "" {
				writeJSON(rw, http.StatusBadRequest, map[string]any{"error": "请输入验证码字符"})
				return
			}
			input = CaptchaInput{
				Type:      CaptchaTypeNormal,
				Code:      lr.Code,
				CaptchaID: info.CaptchaID,
			}
		}
	}

	pair, err := w.authn.Login(r.Context(), lr.Username, lr.Password, input)
	if err != nil {
		// 服务端错误 message 原样回显页面（含锁定/验证码提示），可重试
		writeJSON(rw, http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}

	select {
	case w.done <- pair:
	default:
	}
	writeJSON(rw, http.StatusOK, map[string]any{"ok": true})
}

func writeJSON(rw http.ResponseWriter, status int, v any) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(status)
	_ = json.NewEncoder(rw).Encode(v)
}

// openBrowser 跨平台打开系统浏览器。
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		// rundll32 优先（不闪 cmd 窗口），cmd /c start 兜底
		cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		if err := cmd.Start(); err == nil {
			return nil
		}
		return exec.Command("cmd", "/c", "start", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
