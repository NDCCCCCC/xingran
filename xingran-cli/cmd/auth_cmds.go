package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/auth"
	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/client"
	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/crypto"
)

// login flag 值（支持非交互回填，agent 友好）。
var (
	flagUsername     string
	flagPassword     string
	flagCaptchaCode  string // normal 模式验证码字符
	flagSliderX      int    // slider 模式滑块 X 坐标
	flagCaptchaType  string // normal|slider|none，默认按 code/slider-x 自动推断
	flagGetCaptcha   bool   // 第一步：拉取验证码图片与元数据（两步登录流程）
	flagCaptchaOut   string // --get-captcha 图片保存路径
	flagCaptchaID    string // 两步登录第二步：显式注入 captchaId（跨进程）
	flagCaptchaToken string // 两步登录第二步：显式注入 slider token（跨进程）
	flagWebLogin     bool   // --web：交互式浏览器登录（推荐，人类用户）
)

func newLoginCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "login",
		Short: "登录并保存双 token",
		Long: `登录 XingRan-Next 后端，获取 access/refresh 双 token 并本地保存。

【推荐】浏览器交互登录（人类用户首选，密码不经过命令行）：
  xingran login --web [-u admin]
  # 打开浏览器完成滑动验证 + 输入密码，成功后 token 自动保存
  # 配合 --dry-run 可预览将启动的本地回调服务而不真正启动
  # 注意：--web 与 --captcha-id/--slider-x/--captcha-code/--get-captcha/--password 互斥

两步登录（agent / 多模态场景，后端启用 slider/normal 验证码时）：
  # 第一步：拉取验证码（输出 captchaId/token/图片路径，供多模态 agent 分析）
  xingran login --get-captcha [--captcha-img-out captcha.png]
  # 第二步：显式注入第一步的 captchaId/token + 识别结果
  xingran login -u admin -p '***' --captcha-id <id> --captcha-token <token> --slider-x <缺口x>
  # normal 两步：--captcha-id <id> --captcha-code <识别字符>

单步登录（后端验证码 disabled 时）：
  xingran login -u admin -p '***' --captcha-type none

验证码类型自动推断：给了 --slider-x 用 slider；给了 --captcha-code 用 normal；
显式 --captcha-type 覆盖推断。--captcha-id/--captcha-token 显式注入优先，
缺省时由本进程 FetchCaptcha 兜底（单进程场景）。`,
		RunE: runE(runLogin),
	}
	c.Flags().BoolVar(&flagWebLogin, "web", false, "【推荐】交互式浏览器登录（滑动验证+浏览器内输入密码，密码不经命令行；与其他 captcha 参数互斥）")
	c.Flags().StringVarP(&flagUsername, "username", "u", os.Getenv("XINGRAN_USERNAME"), "用户名（env XINGRAN_USERNAME）")
	c.Flags().StringVarP(&flagPassword, "password", "p", os.Getenv("XINGRAN_PASSWORD"), "密码（env XINGRAN_PASSWORD）")
	c.Flags().StringVar(&flagCaptchaCode, "captcha-code", "", "图形验证码字符（normal 模式）")
	c.Flags().IntVar(&flagSliderX, "slider-x", -1, "滑块 X 坐标（slider 模式缺口位置）")
	c.Flags().StringVar(&flagCaptchaType, "captcha-type", "", "验证码类型: normal|slider|none（默认自动推断）")
	c.Flags().BoolVar(&flagGetCaptcha, "get-captcha", false, "只拉取验证码并输出元数据（两步登录第一步）")
	c.Flags().StringVar(&flagCaptchaOut, "captcha-img-out", "", "验证码图片保存路径（默认captcha-<id>.png）")
	c.Flags().StringVar(&flagCaptchaID, "captcha-id", "", "显式注入 captchaId（两步登录第二步，跨进程）")
	c.Flags().StringVar(&flagCaptchaToken, "captcha-token", "", "显式注入 slider token（两步登录第二步，仅 slider 需要）")
	return c
}

// wiringCaptchaProvider 返回验证码编排接口（与 wiringAuthenticator 同一实现）。
func wiringCaptchaProvider() auth.CaptchaProvider {
	return wiringAuthenticator(context.Background()).(auth.CaptchaProvider)
}

// runGetCaptcha 两步登录第一步：FetchCaptcha → 保存图片 → 输出元数据。
func runGetCaptcha(ctx context.Context) runResult {
	prov := wiringCaptchaProvider()
	info, err := prov.FetchCaptcha(ctx)
	if err != nil {
		return runResult{err: err}
	}
	if info == nil {
		return runResult{result: map[string]any{
			"captcha_enabled": false,
			"hint":            "后端验证码未启用，直接 login -u -p --captcha-type none",
		}}
	}
	out := map[string]any{
		"captcha_id":   info.CaptchaID,
		"captcha_type": info.CaptchaType,
		"token":        info.Token,
	}
	// 保存主图（slider 存底图；normal 存字符图）
	img := info.SliderImg
	if img == "" {
		img = info.CaptchaImg
	}
	if img != "" {
		path := flagCaptchaOut
		if path == "" {
			path = fmt.Sprintf("captcha-%s.png", info.CaptchaID)
		}
		if err := auth.SaveImage(img, path); err != nil {
			return runResult{err: err}
		}
		out["image_path"] = path
		out["hint"] = "分析缺口/字符后：login -u -p --captcha-id <captcha_id> --captcha-token <token> --slider-x <缺口x坐标>"
	}
	return runResult{result: out}
}

func runLogin(ctx context.Context, _ *cobra.Command, _ []string) runResult {
	if flagWebLogin {
		return runWebLogin(ctx) // --web 优先（推荐方式）
	}
	if flagGetCaptcha {
		return runGetCaptcha(ctx)
	}
	authImpl := wiringAuthenticator(ctx)
	if flagUsername == "" || flagPassword == "" {
		return runResult{err: argError("需要 --username 与 --password（或设置 XINGRAN_USERNAME/XINGRAN_PASSWORD）")}
	}

	captcha := inferCaptcha(flagCaptchaType, flagCaptchaCode, flagSliderX)
	// 显式注入两步登录凭据（跨进程），优先于进程内 FetchCaptcha 兜底。
	captcha.CaptchaID = flagCaptchaID
	captcha.Token = flagCaptchaToken
	pair, err := authImpl.Login(ctx, flagUsername, flagPassword, captcha)
	if err != nil {
		return runResult{err: err}
	}
	return runResult{result: map[string]any{
		"status":        "ok",
		"access_token":  mask(pair.AccessToken),
		"refresh_token": mask(pair.RefreshToken),
	}}
}

// inferCaptcha 由 flag 推断 CaptchaInput。
func inferCaptcha(captchaType, code string, sliderX int) auth.CaptchaInput {
	t := captchaType
	if t == "" {
		switch {
		case sliderX >= 0:
			t = "slider"
		case code != "":
			t = "normal"
		default:
			t = "none"
		}
	}
	in := auth.CaptchaInput{Type: t, Code: code}
	if sliderX >= 0 {
		in.SliderX = sliderX
	}
	return in
}

// mask 脱敏 token：只留前 4 后 4 位。
func mask(tok string) string {
	if len(tok) <= 8 {
		return "***"
	}
	return fmt.Sprintf("%s...%s", tok[:4], tok[len(tok)-4:])
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "登出并清除本地 token",
		RunE: runE(func(ctx context.Context, _ *cobra.Command, _ []string) runResult {
			if err := wiringAuthenticator(ctx).Logout(ctx); err != nil {
				return runResult{err: err}
			}
			return runResult{result: map[string]any{"status": "ok"}}
		}),
	}
}

func newRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "手动刷新 access token",
		RunE: runE(func(ctx context.Context, _ *cobra.Command, _ []string) runResult {
			pair, err := wiringAuthenticator(ctx).Refresh(ctx)
			if err != nil {
				return runResult{err: err}
			}
			return runResult{result: map[string]any{
				"status":        "ok",
				"access_token":  mask(pair.AccessToken),
				"refresh_token": mask(pair.RefreshToken),
			}}
		}),
	}
}

// wiringEncryptor 返回 Encryptor 实现（internal/crypto）。
// 公钥拉取由 client / auth 在首次加密请求前自动完成，这里只构造实例；
// 加密失败不降级（CONTRACT 安全要求）。
func wiringEncryptor(_ context.Context) client.Encryptor {
	return crypto.NewEncryptor()
}

// wiringAuthenticator 返回 Authenticator 实现（internal/auth）：
// HTTPRequester(共享 Encryptor) + 文件 TokenStore（0600，auth 负责）。
func wiringAuthenticator(_ context.Context) client.Authenticator {
	enc := crypto.NewEncryptor()
	req := &auth.HTTPRequester{
		BaseURL:   flagBaseURL,
		Encryptor: enc,
	}
	return auth.NewAuthenticator(req, auth.NewFileTokenStore(), enc, flagBaseURL)
}
