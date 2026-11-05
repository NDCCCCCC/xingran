package client

// ParseSchema 的 M8 新增分支覆盖率：upload(formData file) / download({file} file)。

import (
	"strings"
	"testing"
)

const m8SwaggerJSON = `{
  "swagger": "2.0",
  "basePath": "",
  "paths": {
    "/ops/workstation/import": {
      "post": {
        "summary": "导入工位数据",
        "tags": ["工位管理"],
        "parameters": [
          {"name": "file", "in": "formData", "type": "file", "required": true},
          {"name": "note", "in": "query", "type": "string"}
        ],
        "responses": {"200": {"description": "ok", "schema": {"type": "object"}}}
      }
    },
    "/ops/workstation/template": {
      "get": {
        "summary": "下载模板",
        "parameters": [],
        "responses": {"200": {"description": "ok", "schema": {"type": "file"}}}
      }
    },
    "/ops/workstation/list": {
      "post": {
        "summary": "查询工位列表",
        "parameters": [
          {"name": "request", "in": "body", "schema": {"$ref": "#/definitions/Req"}}
        ],
        "responses": {"200": {"description": "ok", "schema": {"type": "object"}}}
      }
    }
  },
  "definitions": {
    "Req": {
      "type": "object",
      "properties": {
        "current": {"type": "integer"},
        "pageSize": {"type": "integer"}
      }
    }
  }
}`

func TestParseSchemaM8Kinds(t *testing.T) {
	// ParseSchema 接受 docs.go 的模板字符串，需包装成 const docTemplate = `...` 形式。
	wrapped := "const docTemplate = `" + m8SwaggerJSON + "`"
	eps, base, err := ParseSchema(wrapped)
	if err != nil {
		t.Fatalf("ParseSchema 失败: %v", err)
	}
	if base != "" {
		t.Fatalf("basePath 应为空: %q", base)
	}
	if len(eps) != 3 {
		t.Fatalf("应解析出 3 个端点，实际 %d", len(eps))
	}

	byPath := map[string]EndpointInfo{}
	for _, e := range eps {
		byPath[e.Path] = e
	}

	// upload：formData file 被识别，且不进普通参数
	up := byPath["/ops/workstation/import"]
	if up.Kind != "upload" {
		t.Fatalf("import 应识别为 upload，实际 %q", up.Kind)
	}
	if len(up.Params) != 1 || up.Params[0].Name != "note" {
		t.Fatalf("file 参数不应进普通参数列表: %+v", up.Params)
	}

	// download：200 schema type=file
	dl := byPath["/ops/workstation/template"]
	if dl.Kind != "download" {
		t.Fatalf("template 应识别为 download，实际 %q", dl.Kind)
	}

	// 普通 JSON 端点：body $ref 展开为字段
	js := byPath["/ops/workstation/list"]
	if js.Kind != "" {
		t.Fatalf("普通端点 Kind 应为空，实际 %q", js.Kind)
	}
	names := make([]string, 0, len(js.Params))
	for _, p := range js.Params {
		names = append(names, p.Name)
	}
	if !strings.Contains(strings.Join(names, ","), "current") {
		t.Fatalf("body $ref 应展开字段: %+v", js.Params)
	}
}
