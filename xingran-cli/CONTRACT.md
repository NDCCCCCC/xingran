# XingRan CLI 开发契约（Team Lead 协调文档）

目标：面向 AI agent / 人类双场景的后端 CLI 工具。安全要求：**不降级**，Web 端二次验证（验证码）CLI 必须完整支持。

## 技术选型
- Go 1.24，复用后端仓库（本仓库）`pkg/crypto`（tjfoc/gmsm SM2/SM3/SM4）
- 命令框架：cobra；输出：结构化 JSON（默认非 TTY）/ 人类可读（TTY）
- 模块路径：`github.com/xingran-next/ruoyi-go-backend/xingran-cli/...`（以 go.mod 实际为准）

## 模块划分与写权限（互斥）
| 目录 | 负责人 | 内容 |
|---|---|---|
| `internal/crypto/` | crypto-dev | SM2+SM4 混合加密 client：拉公钥(`GET /api/v1/system/auth/public-key`)、SM4-CBC 加密 body、SM2 加密 SM4 key、timestamp(±60s)+nonce 防重放 |
| `internal/auth/` | auth-dev | 登录链路：captcha 三模式(normal/slider/disabled) 获取+验证渲染回填、`POST /system/auth/login|logout|refresh`、JWT 双 token 自动刷新、token 本地存储(0600 权限，可选加密落盘)、登录失败锁定处理 |
| `internal/client/` | client-dev | HTTP 核心：请求加密接入、code!=0 错误解析、401 自动 refresh 重试一次、泛化命令 `api <path> [json]`、Swagger 元数据 `schema` 命令数据源(docs/docs.go) |
| `internal/output/` | client-dev | JSON 输出、语义化退出码(0 成功/1 网络错误/2 认证失败/3 参数错误/4 权限不足/5 其他)、非 TTY 不交互不染色 |
| `cmd/` | client-dev | cobra 命令树：`login/logout/refresh/api/schema/version`，`--captcha-code/--slider-x/--dry-run/--wait` |
| `tests/` | auth-dev | 端到端集成测试（需要后端运行时才跑） |

## 接口契约（各模块之间只通过这些接口耦合，可并行开发）
```go
// internal/crypto: 提供
type Encryptor interface {
    FetchPublicKey(ctx context.Context, baseURL string) error
    EncryptRequest(ctx context.Context, body []byte) (payload []byte, err error) // SM4 加密 body + SM2 加密 key + ts/nonce
}
// internal/auth: 提供
type Authenticator interface {
    Login(ctx context.Context, username, password string, captcha CaptchaInput) (*TokenPair, error)
    Refresh(ctx context.Context) (*TokenPair, error)
    GetAccessToken(ctx context.Context) (string, error) // 过期自动 refresh
    Logout(ctx context.Context) error
}
type CaptchaInput struct { Type string /* normal|slider|none */; Code string; SliderX int }
// internal/client: 提供
type Client interface {
    Do(ctx context.Context, method, path string, body any) (*Response, error) // Response = {code,message,data,timestamp,request_id}
}
```

## 后端事实（已调研核实）
- 端点 ~380+，全部 POST 风格为主：`/list`、``、`/:id`、`/:id/update`、`/:id/delete`
- 响应包装：`{code:0成功, message, data, timestamp, request_id}`；401/1007=token 无效
- captcha API：`GET /system/auth/captcha`（获取）+ `POST /system/auth/captcha/verify/slider`（滑动）；登录时 captchaId+captcha(或 slider token) 随请求提交
- request_encryption.enabled=true 但 require_encryption=false；登录已纳入加密
- 分页：`current` 从 1 开始；响应 `list/total/current/pageSize`

## 里程碑
- M1（并行）：crypto、auth 各自单元测试通过（crypto 可先行，auth 用 mock Encryptor）
- M2：client/output/cmd 串起来，`login → api /system/user/list` 冒烟通过
- M3：集成测试 + README 使用说明（含 agent 非交互用法示例）
