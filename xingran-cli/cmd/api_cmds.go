package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/xingran-next/ruoyi-go-backend/xingran-cli/internal/client"
	"github.com/xingran-next/ruoyi-go-backend/xingran-cli/internal/output"
)

func newAPICmd() *cobra.Command {
	var flagMethod string
	c := &cobra.Command{
		Use:   "api <path> [json-body]",
		Short: "泛化 API 调用：任意端点 + 可选 JSON body（万能兜底）",
		Long: `泛化 API 调用命令——三级命令树（xingran <module> <resource> <action>）
覆盖不到的任意端点都可用本命令直达，两者共享同一 HTTP 核心与输出约定。

示例：
  xingran api /system/users/list '{"current":1,"pageSize":10}'
  xingran api /system/auth/public-key
  xingran api /system/user/123 --dry-run

path 为后端 API 路径（可带或不带前导 /，自动补 /api/v1 前缀）。
body 为合法 JSON 文本。非 TTY 输出完整响应包装 {code,message,data,timestamp,request_id}。`,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) < 1 {
				return argError("用法: xingran api <path> [json-body]")
			}
			if len(args) > 2 {
				return argError("最多接受 2 个参数: <path> [json-body]")
			}
			if len(args) == 2 {
				if !json.Valid([]byte(args[1])) {
					return argError("json-body 不是合法 JSON")
				}
			}
			return nil
		},
		RunE: runE(func(ctx context.Context, _ *cobra.Command, args []string) runResult {
			c := newClient(ctx)
			path := args[0]
			var body any
			if len(args) == 2 {
				body = json.RawMessage(args[1])
			}

			if flagDryRun {
				dr, err := c.DryRun(ctx, flagMethod, path, body)
				if err != nil {
					return runResult{err: err}
				}
				return runResult{result: map[string]any{"dry_run": true, "request": dr}}
			}

			resp, err := c.Do(ctx, flagMethod, path, body)
			if err != nil {
				return runResult{err: err}
			}
			return runResult{result: resp}
		}),
	}
	c.Flags().StringVar(&flagMethod, "method", "POST", "HTTP 方法（默认 POST，后端以 POST 风格为主）")
	return c
}

func newSchemaCmd() *cobra.Command {
	var flagDocs string
	var flagFormat string
	c := &cobra.Command{
		Use:   "schema [path-prefix]",
		Short: "列出 API 端点元数据（Swagger 数据源，供模型自发现）",
		Long: `从仓库 docs/docs.go（swaggo 生成的 Swagger spec）解析端点列表，
输出 path/method/summary/description/tags/参数说明，按 path-prefix 过滤。

示例：
  xingran schema                     # 全部端点
  xingran schema /system/user        # 前缀过滤
  xingran schema /ops/building --docs D:/path/to/docs/docs.go`,
		Args: cobra.MaximumNArgs(1),
		RunE: runE(func(_ context.Context, _ *cobra.Command, args []string) runResult {
			docsPath := flagDocs
			if docsPath == "" {
				docsPath = client.DefaultDocsPath()
			}
			eps, basePath, err := client.LoadSchema(docsPath)
			if err != nil {
				return runResult{err: output.NewExitError(output.ExitOther, err.Error(), err)}
			}
			prefix := ""
			if len(args) == 1 {
				prefix = args[0]
			}
			eps = client.FilterEndpoints(eps, prefix)
			if flagFormat == "text" {
				var b strings.Builder
				fmt.Fprintf(&b, "# %d endpoints (basePath=%s, docs=%s)\n", len(eps), basePath, docsPath)
				for _, e := range eps {
					fmt.Fprintf(&b, "%-6s %-45s %s\n", e.Method, e.Path, e.Summary)
				}
				return runResult{result: b.String(), raw: true}
			}
			return runResult{result: map[string]any{
				"base_path": basePath,
				"docs":      docsPath,
				"total":     len(eps),
				"endpoints": eps,
			}}
		}),
	}
	c.Flags().StringVar(&flagDocs, "docs", "", "docs.go 路径（默认自动向上查找 docs/docs.go）")
	c.Flags().StringVar(&flagFormat, "format", "json", "输出格式: json|text")
	return c
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示 CLI 版本信息",
		RunE: runE(func(_ context.Context, _ *cobra.Command, _ []string) runResult {
			return runResult{result: map[string]any{
				"version": version,
				"commit":  commit,
			}}
		}),
	}
}
