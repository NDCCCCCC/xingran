// 生成式三级命令树的执行入口：cmd 层注入 commands.RunFunc / EmitFunc。
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/commands"
	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/output"
)

// runEndpoint 三级命令的执行：位置参数还原路径 → body 组装 → client.Do / dry-run。
func runEndpoint(ctx context.Context, m *commands.EndpointMeta, flags map[string]any, pathVals []string) (any, error) {
	// json 类型 flag 先校验合法性，避免把坏 JSON 发给后端
	for _, f := range m.Flags {
		if f.Type != "json" {
			continue
		}
		if v, ok := flags[f.Name]; ok {
			s, _ := v.(string)
			if s != "" && !json.Valid([]byte(s)) {
				return nil, argError(fmt.Sprintf("--%s 不是合法 JSON: %s", f.Name, s))
			}
		}
	}

	path := commands.RenderPath(m.APIPath, m.PathArgs, pathVals)
	bodyMap := commands.BuildBody(flags, m.Flags)
	var finalBody any = bodyOrNil(bodyMap)

	// 如果没有在命令行显式提供 flag 参数，且标准输入存在管道或重定向输入，自动作为整体 body
	if len(bodyMap) == 0 && m.Kind != commands.KindUpload {
		if stdinData, ok := readStdinIfAvailable(); ok && len(stdinData) > 0 {
			if !json.Valid(stdinData) {
				return nil, argError("从标准输入读取的内容不是合法 JSON")
			}
			finalBody = json.RawMessage(stdinData)
		}
	}

	c := newClient(ctx)

	// 按端点类型分流（M8）：multipart 上传 / 二进制下载 / 普通 JSON
	switch m.Kind {
	case commands.KindUpload:
		if flagDryRun {
			return map[string]any{"dry_run": true, "command": endpointCommandName(m),
				"message": fmt.Sprintf("将 POST multipart 上传 %s → %s（字段 file）", flags["file"], path)}, nil
		}
		file, _ := flags["file"].(string)
		resp, err := c.DoUpload(ctx, path, "file", file, nil)
		if err != nil {
			return nil, err
		}
		return resp, nil
	case commands.KindDownload:
		if flagDryRun {
			return map[string]any{"dry_run": true, "command": endpointCommandName(m),
				"message": fmt.Sprintf("将 %s %s 并保存响应文件流（--out 指定输出路径）", m.Method, path)}, nil
		}
		out, _ := flags["out"].(string)
		jsonBody := "{}"
		if b, ok := flags["body"].(string); ok && b != "" {
			jsonBody = b
		} else if raw, ok := finalBody.(json.RawMessage); ok {
			jsonBody = string(raw)
		}
		saved, err := c.DoDownload(ctx, path, jsonBody, out, m.Method)
		if err != nil {
			return nil, err
		}
		return map[string]any{"status": "ok", "saved": saved}, nil
	}

	if flagDryRun {
		dr, err := c.DryRun(ctx, m.Method, path, finalBody)
		if err != nil {
			return nil, err
		}
		return map[string]any{"dry_run": true, "command": endpointCommandName(m), "request": dr}, nil
	}

	resp, err := c.Do(ctx, m.Method, path, finalBody)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// emitEndpoint 统一输出与退出码（复用 output.Emit 与 silentError 约定）。
func emitEndpoint(result any, err error) error {
	code := output.EmitWithFormat(result, err, os.Stdout, os.Stderr, isTTY(), flagFormat)
	if code != output.ExitOK {
		return &silentError{code: code}
	}
	return nil
}

func bodyOrNil(body map[string]any) any {
	if len(body) == 0 {
		return nil
	}
	return body
}

// endpointCommandName 供 dry-run 输出显示等价的三级命令。
func endpointCommandName(m *commands.EndpointMeta) string {
	name := "xingran " + m.Module
	for _, r := range m.ResourcePath {
		name += " " + r
	}
	name += " " + m.Action
	for _, p := range m.PathArgs {
		name += " <" + p + ">"
	}
	return name
}
