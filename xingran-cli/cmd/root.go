// Package cmd 组装 XingRan CLI 的 cobra 命令树。
//
// 组装点：Encryptor（internal/crypto）与 Authenticator（internal/auth）
// 由本层的 wiring 函数注入 Client；两个模块完成后替换 TODO 即可。
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/xingran-next/ruoyi-go-backend/xingran-cli/internal/client"
	"github.com/xingran-next/ruoyi-go-backend/xingran-cli/internal/output"
)

// 版本信息（构建时可通过 -ldflags 注入）。
var (
	version = "dev"
	commit  = "none"
)

// 全局持久 flag 值。
var (
	flagBaseURL string
	flagDryRun  bool
	flagWait    time.Duration // --wait：单请求超时预算（agent 友好语义，留接口）
	flagNoColor bool          // 预留：TTY 下禁用着色
	flagNoTTY   bool          // 强制 JSON 输出（等价非 TTY 行为）
)

// silentError 表示输出已由 output.Emit 完成，Execute 只需透传退出码。
type silentError struct{ code int }

func (e *silentError) Error() string { return "" }

// wiringEncryptor / wiringAuthenticator 见 auth_cmds.go（auth/crypto 合流后的装配点）。

// newClient 按全局 flag 组装 HTTP 核心 Client。
func newClient(ctx context.Context) client.Client {
	return client.New(client.Options{
		BaseURL:       flagBaseURL,
		Timeout:       flagWait,
		Encryptor:     wiringEncryptor(ctx),
		Authenticator: wiringAuthenticator(ctx),
	})
}

// isTTY 判断 stdout 是否终端；--no-tty 强制走 JSON。
// 用 x/term 精确检测：os.ModeCharDevice 会把 /dev/null（字符设备）误判为 TTY。
func isTTY() bool {
	if flagNoTTY {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// runResult 一次命令执行的产物。raw=true 时 result 为纯文本（如 --format text），
// 直接写 stdout，不经 JSON 编码。
type runResult struct {
	result any
	err    error
	raw    bool
}

// runE 包装 RunE：统一经 output.Emit 输出并映射退出码。
func runE(fn func(ctx context.Context, cmd *cobra.Command, args []string) runResult) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		r := fn(c.Context(), c, args)
		if r.err == nil && r.raw {
			if s, ok := r.result.(string); ok {
				fmt.Fprint(os.Stdout, s)
				return nil
			}
		}
		code := output.Emit(r.result, r.err, os.Stdout, os.Stderr, isTTY())
		if code != output.ExitOK {
			return &silentError{code: code}
		}
		return nil
	}
}

// argError 参数校验错误（exit 3）。
func argError(msg string) error {
	return output.NewExitError(output.ExitParam, msg, nil)
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "xingran",
		Short: "XingRan-Next 后端 CLI（面向 AI agent / 人类双场景）",
		Long: `XingRan CLI —— XingRan-Next 后端的命令行客户端。

两种使用方式：
  1. 三级命令树（推荐，人类与 agent 均易读）：
       xingran <module> <resource> <action>
       例：xingran system user list / xingran system user get <id>
       每级都有完整 help：xingran system --help / xingran system user --help
  2. 万能兜底（agent 场景，任意端点）：
       xingran api <path> [json-body]
       例：xingran api /system/users/list '{"current":1}'
     三级树覆盖不到的端点用 api 命令直达；两者共享同一 HTTP 核心与输出约定。

其他命令：
  login / logout / refresh —— 认证（login --web 浏览器交互登录为推荐方式）
  schema —— 列出全部 API 端点元数据（Swagger 数据源，供模型自发现）
  version —— 版本信息

特性：
  - 全部输出为结构化 JSON（非 TTY / agent 场景）或人类可读（TTY）
  - 语义化退出码：0 成功 / 1 网络 / 2 认证 / 3 参数 / 4 权限 / 5 其他
  - SM2+SM4 国密请求加密（由 crypto 模块自动接入）
  - 401/1007 自动 refresh token 并重试一次
  - --dry-run 打印将发送的请求而不发送`,
		SilenceUsage:  true, // 错误时不打印 usage，agent 友好
		SilenceErrors: true, // 错误统一走 output.Emit
	}
	pf := root.PersistentFlags()
	pf.StringVar(&flagBaseURL, "base-url", envOr("XINGRAN_BASE_URL", "http://localhost:9000"),
		"后端地址（env XINGRAN_BASE_URL）")
	pf.BoolVar(&flagDryRun, "dry-run", false, "打印将发送的请求而不发送")
	pf.DurationVar(&flagWait, "wait", 30*time.Second,
		"单请求超时预算，如 30s / 2m")
	pf.BoolVar(&flagNoColor, "no-color", false, "禁用着色输出")
	pf.BoolVar(&flagNoTTY, "no-tty", false, "强制 JSON 输出（等价非 TTY 行为）")

	root.AddCommand(
		newLoginCmd(),
		newLogoutCmd(),
		newRefreshCmd(),
		newAPICmd(),
		newSchemaCmd(),
		newVersionCmd(),
	)
	// 三级命令树（tools/cmdgen 从 Swagger 生成，按模块注册）：
	// xingran <module> <resource...> <action>；xingran api 保持万能兜底。
	root.AddCommand(newSystemModuleCmd(runEndpoint, emitEndpoint))
	root.AddCommand(newOpsModuleCmd(runEndpoint, emitEndpoint))
	root.AddCommand(newNetworkModuleCmd(runEndpoint, emitEndpoint))
	root.AddCommand(newMonitorModuleCmd(runEndpoint, emitEndpoint))
	root.AddCommand(newWorkorderModuleCmd(runEndpoint, emitEndpoint))
	root.AddCommand(newDutyModuleCmd(runEndpoint, emitEndpoint))
	root.AddCommand(newKnowledgeModuleCmd(runEndpoint, emitEndpoint))
	return root
}

// Execute CLI 入口，返回进程退出码。
func Execute(args []string) int {
	root := newRootCmd()
	root.SetArgs(args)
	// cobra 的参数个数校验错误归类为参数错误（exit 3），其余兜底 exit 5。
	err := root.Execute()
	if err == nil {
		return output.ExitOK
	}
	var se *silentError
	{
		s, ok := err.(*silentError)
		if ok {
			se = s
		}
	}
	if se != nil {
		return se.code
	}
	if isArgsError(err) {
		fmt.Fprintf(os.Stderr, "{\"code\":-1,\"message\":%q,\"exit_code\":%d}\n",
			err.Error(), output.ExitParam)
		return output.ExitParam
	}
	return output.Emit(nil, err, os.Stdout, os.Stderr, isTTY())
}

// isArgsError 识别 cobra 的用法/参数校验错误。
func isArgsError(err error) bool {
	// cobra: "unknown command", "unknown flag", "accepts N arg(s)", "flag needs an argument"
	msg := err.Error()
	for _, p := range []string{"unknown command", "unknown flag", "accepts ", "arg(s)", "flag needs", "invalid argument"} {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
