# XingRan CLI

面向 AI agent / 人类双场景的 XingRan-Next 后端命令行工具。安全不降级：完整支持 Web 端二次验证（验证码三模式）。

## 模块结构

| 目录 | 职责 |
|---|---|
| `internal/crypto/` | SM2+SM4 混合加密 client（拉公钥、SM4-CBC body + SM2 key + ts/nonce 防重放） |
| `internal/auth/` | 登录链路：验证码三模式、login/logout/refresh、JWT 双 token 自动刷新、token 本地存储（0600） |
| `internal/client/` | HTTP 核心：加密接入、错误解析、401/1007 自动 refresh 重试、泛化 `api` 命令 |
| `internal/output/` | JSON 输出、语义化退出码（0 成功 / 1 网络 / 2 认证 / 3 参数 / 4 权限 / 5 其他） |
| `cmd/` | cobra 命令树与模块装配（wiringEncryptor / wiringAuthenticator） |
| `tests/` | 端到端集成测试（需真实后端） |

## 快速开始

```bash
cd xingran-cli
go build -o xingran-cli.exe ./cmd/xingran
```

### 三级命令树（主推用法）

模块 → 资源 → 动作三级结构，每级 `--help` 均含说明与示例：

```bash
# 登录（后端验证码为 disabled 时直通；交互式推荐 --web，见下文）
./xingran-cli.exe login -u admin -p <password>

# 查询用户列表（模块 system → 资源 user → 动作 list）
./xingran-cli.exe system user list --current 1 --pageSize 10
./xingran-cli.exe system user create --username alice --password '***'

# 各模块速查（每级 --help 均含「对应 API」与示例）
./xingran-cli.exe system --help      # 系统管理：用户/角色/部门/菜单/字典/岗位/参数/通知/仪表盘/文件等
./xingran-cli.exe ops --help         # 运维管理：楼宇/楼层/工位/信息点/专线等
./xingran-cli.exe monitor --help     # 监控：缓存/服务器等
./xingran-cli.exe duty --help        # 值班管理
./xingran-cli.exe knowledge --help   # 知识库
./xingran-cli.exe network --help     # 网络设备
./xingran-cli.exe workorder --help   # 工单

# 端点探查（Swagger docs，按路径前缀过滤）
./xingran-cli.exe schema /system/user
```

### 万能兜底：`api` 命令

命令树未覆盖的端点、或需原样透传任意 JSON body 时使用：

```bash
./xingran-cli.exe api /system/users/list '{"current":1,"pageSize":10}'
```

### 登出（清除本地 token）

```bash
./xingran-cli.exe logout
```

token 存储：`%USERPROFILE%/.xingran-cli/tokens.json`（POSIX 0600 / 目录 0700；Windows 退化为只读位）。access token 剩余有效期 <30s 时自动 refresh；refresh 失败清除本地凭据并要求重新登录。

## 验证码三模式（agent 非交互指南）

后端模式由数据库配置键动态决定（`sys.account.captchaEnabled` = disabled / normal / slider）。
CLI 库级 API（`internal/auth`）：

```go
enc := crypto.NewEncryptor()
req := &auth.HTTPRequester{BaseURL: baseURL, Encryptor: enc} // 加密模式默认 auto（见下）
a := auth.NewAuthenticator(req, auth.NewFileTokenStore(), enc, baseURL)
cp := a.(auth.CaptchaProvider) // 验证码编排扩展接口
```

### 探测当前模式

`POST /system/auth/captcha/config` → `{"data":{"enabled":"normal|slider|disabled",...}}`。
`cp.FetchCaptcha(ctx)` 在 disabled 模式返回 `(nil, nil)`。

### normal 字符验证码

```go
info, _ := cp.FetchCaptcha(ctx)
// 三种回填方式（任选，均为多模态 agent 友好）：
auth.RenderTerminal(info.CaptchaImg, 80)          // 1. 终端 ASCII 字符画
auth.SaveImage(info.CaptchaImg, "captcha.png")    // 2. 保存 PNG 供多模态模型读取
// 3. info.CaptchaImg 本身就是 data:image/png;base64,... dataURL，可直接投喂多模态模型

a.Login(ctx, user, pass, auth.CaptchaInput{Type: "normal", Code: "Ab12"})
```

### slider 滑动验证码（两阶段工作流）

```go
info, _ := cp.FetchCaptcha(ctx)
// 阶段1：保存 info.SliderImg（底图）+ info.PieceImg（拼图块）+ info.YPos
//        多模态 agent 观察底图中缺口位置
// 阶段2：回填缺口 x 坐标（后端容差 ±8px）
a.Login(ctx, user, pass, auth.CaptchaInput{Type: "slider", SliderX: 227})
// Login 内部自动: POST /system/auth/captcha/verify/slider → 通过后以 captcha="verified" 登录
```

**跨进程两步登录**（CLI `login --get-captcha` 场景）：阶段1 与阶段2 是两个独立进程，
进程内 pending 缓存不可用，此时显式注入阶段1 落盘/输出的 `captchaId` 与 `token`：

```go
a.Login(ctx, user, pass, auth.CaptchaInput{
    Type: "slider", SliderX: 227,
    CaptchaID: info.CaptchaID, // 显式注入（优先于进程内缓存）
    Token:     info.Token,
})
// normal 模式同理：CaptchaInput{Type:"normal", Code:"Ab12", CaptchaID: info.CaptchaID}
```

凭据来源规则：显式注入（`CaptchaID`/`Token`）优先；任一缺省时回退本进程 `FetchCaptcha`
的 pending 缓存（单进程路径兼容）；两者皆无则报错。

**缺口 x 坐标识别要点**（实测后端 `pkg/captcha` 生成逻辑得出）：

- 后端校验 `abs(xPos - 存储XPos) <= 8px`。实测存储值更贴近缺口**左边缘**（阴影从
  `xPos` 起绘制 `PieceWidth` 宽的形状）；但在 ±8px 容差内，左边缘与暗区中心都可能命中，
  **优先取暗区左端**，命中概率更高
- **缺口外观（client-dev 实测）**：缺口是背景**变暗约 25% 的深色矩形 + 黑色边框**；
  注意底图左侧的**白色拼图形状是 piece 起始展示位，不是缺口**——这是最常见的误判来源，
  定位时直接忽略左侧白色拼图块
- 背景图本身可能含树影等大片暗区干扰，逐列亮度阈值法建议**结合黑边框检测**定位缺口
  （黑色矩形边框是比整体变暗更强的特征）；服务端容差固定 ±8px
- 像素级精确定位法：在缺口 Y 行带（`info.YPos` 起 48px）内逐列计算亮度，找连续暗列区段（< 全图均值×0.72），区段左端即 x。本项目集成测试已用此方法实测命中
- 同一 captchaId 最多验证 3 次（`maxAttempts`），失败后需重新 FetchCaptcha；验证失败**不消耗**登录失败计数，不会触发账号锁定

### disabled 直通

```go
a.Login(ctx, user, pass, auth.CaptchaInput{Type: "none"}) // 不带验证码字段
```

### 浏览器交互式登录（--web，推荐）

不想在终端看图数像素？用 `login --web`：CLI 启动一个仅绑定 127.0.0.1 的一次性本地服务
（随机端口 + URL 携带 256bit session token，防其他本地进程伪造），自动打开系统浏览器
（Windows rundll32 / macOS open / Linux xdg-open，失败时打印 URL 手动访问）：

```bash
./xingran-cli.exe login --web -u admin
```

页面内完成全部认证，**密码只在浏览器与服务端内存间流转，不经过命令行参数 / shell 历史**：

- 用户名/密码输入框（用户名预填 `-u` 值，可修改）
- slider 模式：底图 + 可拖动拼图块，直接拖到缺口处松开（无需计算坐标）
- normal 模式：显示字符验证码图（点击可刷新）+ 输入框
- disabled 模式：无验证码区
- 失败自动回显服务端错误（含锁定提示），验证码自动刷新可重试
- 成功后 token 自动落盘，页面提示可关闭，CLI 侧服务随即关闭并输出 TokenPair

默认 5 分钟超时；成功/超时/取消均会关闭本地服务。

库级 API（`internal/auth`）：

```go
pair, err := auth.InteractiveLogin(ctx, auth.WebLoginOptions{
    BaseURL:     "http://localhost:9000",
    Username:    "admin",
    Timeout:     5 * time.Minute, // 缺省同值
    OpenBrowser: true,
    Encryptor:   enc, // 可选；nil 则 auto 模式明文
})
```

## 请求加密模式（auto 探测）

后端加密开关是数据库动态配置（`sys.request.encryption.enabled`），经公开端点
`GET /api/v1/system/auth/encryption-config` 暴露。`HTTPRequester.EncryptionMode`：

- `auto`（默认）：首次加密请求前探测一次并缓存。`enabled=false` 时明文直发（服务端解密中间件直通，加密 envelope 会被 handler 当普通 body 解析失败）；探测网络失败按加密发送（安全优先）
- `on`：强制加密（服务端开启加密的环境）
- `off`：强制明文

登录失败锁定、验证码错误、限流等服务端提示均以 `APIError.Message` 原样透传（如 `账号已锁定，请 25 分钟后再试`），CLI 不做任何改写。

## 集成测试

需要真实后端运行（仓库根目录 `go build -o xingran-backend.exe ./cmd/main.go && .\xingran-backend.exe`）。

```bash
cd xingran-cli
go test -tags=integration ./tests/ -v
```

**命令树验收闸门**（tests/commands_test.go，M6 模块树质量把关）：

- TestCLIHelpSectionsRequired：全模块叶子命令 help 必须含「对应 API」与示例段落
- TestCLIEndpointCoverage：`schema <module>` 端点必须被命令树叶子「对应 API」全覆盖，不一致即 fail（防端点丢失）
- TestCLIHelpNoEmptyUsage / TestCLIHelpReadability：参数说明不得为空或纯英文字段名
- 框架自动编译测试自持二进制（tests/_tmpcli/，经 XINGRAN_CLI_BIN 注入），与开发产物隔离；也可用 `XINGRAN_CLI_BIN` 指定现成二进制跳过编译

用例与前置条件：

| 用例 | 前置 | 说明 |
|---|---|---|
| TestCaptchaConfigProbe | 后端在线 | 探测验证码模式 |
| TestFetchCaptchaStructure | 后端在线 | 拉验证码，校验响应结构 + 渲染链路；slider 模式落盘拼图与状态文件到 `%TEMP%/xingran-captcha-manual/` |
| TestVerifySliderManual | `XINGRAN_SLIDER_X` | 阶段2：注入观察到的缺口 x，真实 verify（不碰登录，不锁号）。先跑 TestFetchCaptchaStructure，5 分钟内完成 |
| TestLoginE2E | `XINGRAN_TEST_USER/PASS` + 验证码答案 | 完整登录；normal 需 `XINGRAN_CAPTCHA_ANSWER`，slider 需 `XINGRAN_SLIDER_X` |
| TestLoginWrongPasswordOnce | disabled 模式 + 测试账号 | 失败路径只执行一次（防锁号） |
| TestRefreshFlow | disabled 模式 + 测试账号 | 真实 refresh 链路 |
| TestSmokeLoginAndUserList | disabled 模式 + 测试账号 | M2 冒烟：login → `/system/users/list` → logout |

环境变量清单：

```bash
XINGRAN_BASE_URL        # 默认 http://localhost:9000
XINGRAN_TEST_USER       # 测试账号（务必专用账号，避免锁号影响生产）
XINGRAN_TEST_PASS
XINGRAN_CAPTCHA_ANSWER  # normal 模式验证码字符（多模态识别结果注入）
XINGRAN_SLIDER_X        # slider 缺口 x 坐标
```

注意：Git Bash 下运行如遇路径转换问题加 `MSYS_NO_PATHCONV=1`。

## 单元测试

```bash
cd xingran-cli
go test ./...                      # 全模块（tests/ 需要 integration tag，不会误触发）
go test -v ./internal/auth/        # auth 模块 27 用例
```
