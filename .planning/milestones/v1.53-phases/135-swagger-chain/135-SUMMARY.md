---
one-liner: 补齐 main.go docs blank import、两生成脚本改产物到 internal/docs 并修 bat 层级、删除 docs/ 下 5 份 Swagger 死文件
---

# Phase 135 — Swagger / 文档产物链路（v1.53 / FIX-02）

**一行摘要**：补齐 `cmd/main.go` 的 docs blank import、修正两生成脚本产物目录与 bat 层级 bug、删除 `docs/` 下 5 份 Swagger 死文件并同步全部引用。

## 交付清单

| 项 | 文件 | 内容 |
|---|---|---|
| ① | `cmd/main.go:3-30` | 按标准库/第三方/内部包三组重排 import，加 `_ ".../internal/docs"` blank import，使 `swag.Register` 生效 |
| ② | `scripts/build/generate_swagger.bat:47`、`scripts/generate_swagger.sh:46` | `-o docs` → `-o internal/docs`，提示文字同步 |
| ③ | `scripts/build/generate_swagger.bat:18-19`、`scripts/README.md:69,256-262` | `cd /d "%~dp0.."` → `"%~dp0..\.."`（上溯两级到项目根）；矛盾表述改如实反映修复后状态 |
| ④ | 删除 `docs/docs.go` / `swagger.json` / `swagger.yaml` / `swagger_templates.go` / `models.go` | 5 份死产物，删除前逐一校验 `package docs`、旧哈希、无 Go 源文件 import、`internal/docs/` 权威产物完好 |
| 引用 | `docs/archive/ARCHIVE-INDEX.md:58-74,82`、两份 `Swagger文档实施总结.md`、`CLAUDE.md:86` | 死链/矛盾/可执行命令 `-o` 同步修正；清理 OPEN-ITEMS 登记说明 6 处 |

## 验证
- `go build ./...` exit 0（连跑两次）
- `gofmt -l cmd/main.go` 无输出
- 静态链路闭合：`internal/docs/docs.go` 的 `init()→swag.Register`、`cmd/main.go:246` 的 `/swagger/*any`、模块名三者一致
- 未启动完整服务做实机 HTTP 验证（按要求仅静态验证）

## 边界
- 两份 Swagger 实施总结为历史归档文档，正文保留历史路径但逐处加「已删除 / 见顶部注记」标注。
