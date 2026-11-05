# M8 Excel 导入导出 · 接口契约（Lead 协调文档）

## 后端事实（已核实）
- 8 类实体，路径前缀 `/api/v1/ops/<entity>`：
  `building` `floor` `workstation` `asset` `infoPoint` `serverRoom` `dedicatedLine` `roomDevice`
- 每类 3 个动作：
  | 动作 | 方法 | 请求 | 响应 |
  |---|---|---|---|
  | import | POST | multipart/form-data，字段名 `file`，值 .xlsx | JSON `{inserted,updated,failed,errors}` |
  | export | POST | JSON body（可空 `{}`） | xlsx 二进制（Content-Disposition: attachment） |
  | template | GET | 无 | xlsx 二进制 |
- 额外：`GET /api/v1/ops/workstation/dept-mapping-template` → xlsx
- 通用 handler 在 `internal/api/v1/operations/excel_handler.go`，被 8 种 entityType 复用
- 权限：import/export/template 分挂不同 RBAC 组，但 URL 同上
- 上传校验：扩展名 .xlsx + 大小上限 + OOXML magic `PK\x03\x04`
- 模板自带 1 行示例数据，直接上传会报「引用的 所属楼宇 不存在」——批量导入前需删除

## 团队分工（写范围互斥）
| 成员 | 任务 | 写范围 |
|---|---|---|
| crypto-dev | task-12：24+1 个 swag 文档占位注解 | `internal/api/v1/operations/` |
| client-dev | task-13：multipart 上传 / 二进制下载 + 命令树生成 | `xingran-cli/internal/client`、`cmd`、`tools/cmdgen` |
| auth-dev | task-14：单测 + 集成测试 + README | `xingran-cli/tests/`、`README.md` |

## CLI 接口约定（client-dev 实现，auth-dev 测试按此写）
```go
// internal/client
func (c *Client) DoUpload(ctx context.Context, path, fileField, filePath string,
    fields map[string]string) (*Response, error)

func (c *Client) DoDownload(ctx context.Context, path, jsonBody, destPath string,
    method string) (savedPath string, err error)
```

命令形态（最终目标）：
```
xingran ops workstation template --out 工位模板.xlsx
xingran ops workstation import   --file 工位.xlsx
xingran ops workstation export   --out 工位导出.xlsx
# 通用兜底
xingran upload   /ops/workstation/import --file x.xlsx
xingran download /ops/workstation/template --out t.xlsx
```

## 关键约束
1. **不降级安全**：上传下载同样走 auto 加密探测、Bearer token、401 自动 refresh 重试
2. **错误响应不落盘**：code!=0 的 JSON 必须作为错误返回，不能写成 .xlsx
3. **swag 注解紧贴 func**：`@Router` 行与 func 声明之间不能有空行（否则整块被丢弃）
4. **docs 重生成由 Lead 统一执行**（`swag init ... --exclude ./xingran-react-frontend`），成员不要跑
5. **命令树端点总数自检**：新增 25 条路径后，端点数 551 → 576，verifyLeafCount 必须通过
6. **CI**：CLI 子模块测试已接入 PR #70 的流程，新增单测须 `go test ./...` 全绿

## 里程碑
- M1（并行）：三项交付各自自测通过
- M2：Lead 跑 swag init → 重生成命令树 → 全量 build/test
- M3：真实后端 E2E（工位 template→import，账号从环境变量读）
- M4：提交 + PR + CI
