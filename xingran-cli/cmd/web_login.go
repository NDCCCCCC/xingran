package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/auth"
	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/crypto"
)

// WebLoginOptions 交互式浏览器登录选项（镜像 auth.WebLoginOptions 约定签名，
// auth-dev task-6 合流后由 wireInteractiveLogin 转换调用）。
type WebLoginOptions struct {
	BaseURL     string
	Username    string
	Timeout     time.Duration
	OpenBrowser bool
}

// wireInteractiveLogin 绑定 auth.InteractiveLogin（task-6 已合流）：
// 注入 crypto 加密器与文件 token 存储，与 CLI 其他登录路径行为一致。
func wireInteractiveLogin() func(ctx context.Context, opts WebLoginOptions) (*auth.TokenPair, error) {
	return func(ctx context.Context, opts WebLoginOptions) (*auth.TokenPair, error) {
		enc := crypto.NewEncryptor()
		return auth.InteractiveLogin(ctx, auth.WebLoginOptions{
			BaseURL:     opts.BaseURL,
			Username:    opts.Username,
			Timeout:     opts.Timeout,
			OpenBrowser: opts.OpenBrowser,
			Encryptor:   enc,
			Store:       auth.NewFileTokenStore(),
		})
	}
}

// webLoginConflicts 列出与 --web 互斥的 flag 名称（用于冲突报错提示）。
var webLoginConflicts = []string{"--captcha-id", "--slider-x", "--captcha-code", "--get-captcha"}

// hasWebLoginConflict 判断用户是否同时给了与 --web 互斥的参数。
func hasWebLoginConflict() bool {
	return flagCaptchaID != "" || flagCaptchaToken != "" || flagSliderX >= 0 ||
		flagCaptchaCode != "" || flagGetCaptcha
}

// runWebLogin --web 交互式浏览器登录。
//
// 冲突策略：与 --captcha-id/--slider-x/--captcha-code/--get-captcha 同给时直接报错
// （exit 3），--help 中已写明 --web 为推荐方式且互斥。
// --dry-run 组合：不启动浏览器，只打印将启动的本地服务地址并退出。
func runWebLogin(ctx context.Context) runResult {
	if hasWebLoginConflict() {
		return runResult{err: argError(fmt.Sprintf(
			"--web 与 %s 互斥，请只使用其一（--web 为推荐方式：浏览器内滑动验证+输入密码，密码不经命令行）",
			joinFlags(webLoginConflicts)))}
	}
	if flagPassword != "" {
		return runResult{err: argError("--web 模式下请在浏览器中输入密码（--password 无效且会泄漏到 shell 历史，已拒绝）")}
	}

	// dry-run：只描述将要发生的行为，不启动浏览器/本地服务。
	if flagDryRun {
		return runResult{result: map[string]any{
			"dry_run": true,
			"message": fmt.Sprintf("将启动本地回调服务并打开浏览器登录页（http://127.0.0.1:<port>，端口由 auth 模块回调服务决定），超时预算 %s", flagWait),
		}}
	}

	fn := wireInteractiveLogin()
	pair, err := fn(ctx, WebLoginOptions{
		BaseURL:     flagBaseURL,
		Username:    flagUsername,
		Timeout:     flagWait,
		OpenBrowser: true,
	})
	if err != nil {
		return runResult{err: err}
	}
	return runResult{result: map[string]any{
		"status":        "ok",
		"access_token":  mask(pair.AccessToken),
		"refresh_token": mask(pair.RefreshToken),
	}}
}

func joinFlags(fs []string) string {
	out := ""
	for i, f := range fs {
		if i > 0 {
			out += " "
		}
		out += f
	}
	return out
}
