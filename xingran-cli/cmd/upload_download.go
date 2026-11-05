// 通用上传/下载命令（M8 保底能力，agent 场景可直接对任意端点使用）。
// Excel 导入导出的三级命令树由 cmdgen 生成（ops <entity> import/export/template）。
package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newUploadCmd() *cobra.Command {
	var flagFile string
	var flagField string
	c := &cobra.Command{
		Use:   "upload <path> --file <文件路径>",
		Short: "multipart 文件上传（任意端点，agent 兜底）",
		Long: `multipart/form-data 文件上传——Excel 导入（ops <entity> import）等
multipart 端点的通用兜底，三级命令树覆盖不到时使用。

示例：
  xingran upload /ops/workstation/import --file 工位.xlsx
  xingran upload /other/upload --file data.bin --field upload

上传走 Bearer token + auto 加密探测 + 401 自动 refresh；响应为
统一包装 {code,message,data,...}（import 端点 data 含 inserted/updated/failed/errors）。`,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return argError("用法: xingran upload <path> --file <文件路径>")
			}
			return nil
		},
		RunE: runE(func(ctx context.Context, _ *cobra.Command, args []string) runResult {
			if flagFile == "" {
				return runResult{err: argError("需要 --file 指定上传文件路径")}
			}
			c := newClient(ctx)
			if flagDryRun {
				return runResult{result: map[string]any{
					"dry_run": true,
					"message": fmt.Sprintf("将 POST multipart 上传 %s → %s（字段 %s）", flagFile, args[0], flagField),
				}}
			}
			resp, err := c.DoUpload(ctx, args[0], flagField, flagFile, nil)
			if err != nil {
				return runResult{err: err}
			}
			return runResult{result: resp}
		}),
	}
	c.Flags().StringVar(&flagFile, "file", "", "上传文件路径（.xlsx）")
	c.Flags().StringVar(&flagField, "field", "file", "multipart 表单字段名（默认 file）")
	return c
}

func newDownloadCmd() *cobra.Command {
	var flagOut string
	var flagMethod string
	var flagBody string
	c := &cobra.Command{
		Use:   "download <path> [--out 文件路径]",
		Short: "二进制流下载（任意端点，agent 兜底）",
		Long: `二进制流下载——Excel 导出（POST+JSON）与模板下载（GET）等
文件流端点的通用兜底，三级命令树覆盖不到时使用。

示例：
  xingran download /ops/workstation/template --out 模板.xlsx
  xingran download /ops/workstation/export --method POST --body '{}' --out 导出.xlsx

JSON 错误响应（code!=0）不会落盘成文件，而是作为错误输出（exit=2/3/5）。
下载走 Bearer token + auto 加密探测 + 401 自动 refresh。`,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return argError("用法: xingran download <path> [--out 文件路径]")
			}
			if flagBody != "" && !json.Valid([]byte(flagBody)) {
				return argError("--body 不是合法 JSON")
			}
			return nil
		},
		RunE: runE(func(ctx context.Context, _ *cobra.Command, args []string) runResult {
			c := newClient(ctx)
			if flagDryRun {
				return runResult{result: map[string]any{
					"dry_run": true,
					"message": fmt.Sprintf("将 %s %s 下载响应文件流（--out 指定输出路径）", flagMethod, args[0]),
				}}
			}
			saved, err := c.DoDownload(ctx, args[0], flagBody, flagOut, flagMethod)
			if err != nil {
				return runResult{err: err}
			}
			return runResult{result: map[string]any{"status": "ok", "saved": saved}}
		}),
	}
	c.Flags().StringVar(&flagOut, "out", "", "输出文件路径（默认从响应 Content-Disposition 解析文件名）")
	c.Flags().StringVar(&flagMethod, "method", "GET", "HTTP 方法（模板 GET / 导出 POST）")
	c.Flags().StringVar(&flagBody, "body", "", "POST 请求 JSON body（导出场景，如 '{}'）")
	return c
}
