// 生成式三级命令树的执行入口：cmd 层注入 commands.RunFunc / EmitFunc。
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/xingran-next/ruoyi-go-backend/xingran-cli/internal/commands"
	"github.com/xingran-next/ruoyi-go-backend/xingran-cli/internal/output"
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
	body := commands.BuildBody(flags, m.Flags)
	c := newClient(ctx)

	if flagDryRun {
		dr, err := c.DryRun(ctx, m.Method, path, bodyOrNil(body))
		if err != nil {
			return nil, err
		}
		return map[string]any{"dry_run": true, "command": endpointCommandName(m), "request": dr}, nil
	}

	resp, err := c.Do(ctx, m.Method, path, bodyOrNil(body))
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// emitEndpoint 统一输出与退出码（复用 output.Emit 与 silentError 约定）。
func emitEndpoint(result any, err error) error {
	code := output.Emit(result, err, os.Stdout, os.Stderr, isTTY())
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
