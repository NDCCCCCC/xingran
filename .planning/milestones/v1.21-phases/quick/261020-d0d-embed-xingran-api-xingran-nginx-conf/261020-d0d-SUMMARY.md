---
phase: quick-261020-d0d
plan: 01
subsystem: infra
tags: [gin, embed, subpath, nginx, deployment, go, routing]

requires:
  - phase: "quick-261019-mmc"
    provides: "项目基线（main @ f5b6a2b，含 embed 前端目录统一到 xingran-react-frontend）"
provides:
  - "internal/server/subpath.go：共享 frontendSubPath/stripSubPath + 白名单前缀剥离 + 根路径重定向判定（双构建共用，无 build tag）"
  - "StripSubPathHandler：进入 gin 路由匹配之前剥掉 /xingran 前缀，使 /xingran/api/v1/... 命中 engine.Group(\"/api/v1\")、/xingran/uploads/... 命中 engine.Static(\"/uploads\")"
  - "StripSubPathMiddleware：同一套判定规则的 gin 中间件，保证 handler 看到剥前缀后的路径"
  - "embed 变体 ServeRoot：无 X-Forwarded-Prefix → 302 /xingran/；带该头 → 直接返回 index.html（防 nginx 部署重定向死循环）"
  - "deploy/xingran.nginx.conf：可照抄的 nginx 站点配置（upstream / 根 302 / 补斜杠 301 / 剥前缀 proxy_pass / XFP 头 / WS 透传 / 超时）"
affects: [internal/server, cmd/main, deploy, 运维部署手册]

actuals:
  tokens: 5356
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "gin 子路径部署：路由前缀剥离必须在 gin 之外（http.Handler 包装器）—— gin 先查路由树再执行中间件"
    - "反代子路径部署契约：X-Forwarded-Prefix 作为『已在子路径下』的唯一豁免信号，避免 / 与 /xingran/ 重定向死循环"

key-files:
  created:
    - "internal/server/subpath.go"
    - "internal/server/subpath_test.go"
    - "internal/server/subpath_embed_test.go"
    - "deploy/xingran.nginx.conf"
  modified:
    - "internal/server/embed_frontend_prod.go"
    - "internal/server/embed_frontend.go"
    - "cmd/main.go"

key-decisions:
  - "前缀剥离生效点放在 HTTP 边界（StripSubPathHandler 包装 gin engine）而非只放中间件：gin v1.10 的 handleHTTPRequest 先用 c.Request.URL.Path 查路由树、再执行 handlers 链，中间件里的改写无法改变路由匹配结果（计划前提在此不成立）"
  - "保留计划指定的 engine.Use(server.StripSubPathMiddleware()) 调用点：与包装器共用同一套判定规则（stripBackendSubPath），保证任何直接使用 gin engine 的场景下 handler 看到的也是剥前缀后的路径"
  - "白名单只含 /api 与 /uploads（后端真实注册路由的前缀）；/assets 刻意不纳入（已由 ServeFrontend/ServeSPA 内部 stripSubPath 处理），SPA 路由 /xingran/dashboard 行为保持不变"
  - "根路径重定向的唯一豁免信号是请求头 X-Forwarded-Prefix：nginx 剥前缀部署时该头由 conf 强制设置，直连 :9000 时不存在该头才 302"

patterns-established:
  - "双构建共享代码不放 build tag 文件：frontendSubPath/stripSubPath 搬到无 tag 的 internal/server/subpath.go，避免 embed 变体重复定义"
  - "embed 变体新增行为必须在 embed + 默认双构建下各有一份用例（subpath_embed_test.go 带 //go:build embed，subpath_test.go 无 tag）"

requirements-completed: [EMBED-SUBPATH-01, EMBED-NGINX-01]

coverage:
  - id: D1
    description: "直连 :9000 时 GET / 返回 302 且 Location 为 /xingran/（浏览器落到 basename 能匹配的地址，Router 不再白屏）"
    requirement: EMBED-SUBPATH-01
    verification:
      - kind: unit
        ref: "internal/server/subpath_embed_test.go#TestSubPath_ServeRoot_RedirectsToSubPath"
        status: pass
      - kind: unit
        ref: "internal/server/subpath_test.go#TestSubPath_RootRedirectPath"
        status: pass
    human_judgment: false
  - id: D2
    description: "GET /xingran/api/v1/... 命中 engine.Group(\"/api/v1\")，GET /xingran/uploads/... 命中 engine.Static(\"/uploads\")，不再是路由 404"
    requirement: EMBED-SUBPATH-01
    verification:
      - kind: unit
        ref: "internal/server/subpath_test.go#TestSubPath_Handler_RewritesBackendPrefixes"
        status: pass
    human_judgment: false
  - id: D3
    description: "SPA 客户端路由与静态资源行为不变；无前缀请求与相似前缀 /xingranfoo 逐字节零影响"
    requirement: EMBED-SUBPATH-01
    verification:
      - kind: unit
        ref: "internal/server/subpath_test.go#TestSubPath_Middleware_LeavesOtherPathsUnchanged"
        status: pass
      - kind: unit
        ref: "internal/server/embed_frontend_prod_test.go#TestServeSPA_SPARouteReturnsIndex"
        status: pass
    human_judgment: false
  - id: D4
    description: "nginx 剥前缀部署下无 / 与 /xingran/ 互相跳转的重定向死循环"
    requirement: EMBED-NGINX-01
    verification:
      - kind: unit
        ref: "internal/server/subpath_embed_test.go#TestSubPath_ServeRoot_WithForwardedPrefixServesIndex"
        status: pass
      - kind: unit
        ref: "internal/server/subpath_embed_test.go#TestSubPath_NoRedirectLoop"
        status: pass
    human_judgment: false
  - id: D5
    description: "deploy/xingran.nginx.conf 提供可照抄的站点配置（upstream / 根 302 / 301 补斜杠 / 剥前缀 proxy_pass / XFP 头 / WS 透传 / 超时）"
    requirement: EMBED-NGINX-01
    verification:
      - kind: other
        ref: "awk 括号配平 + grep -c 'proxy_set_header X-Forwarded-Prefix /xingran;' = 1 + grep -c 'proxy_pass http://127.0.0.1:9000/;' = 1"
        status: pass
    human_judgment: false
  - id: D6
    description: "双构建（默认 + -tags embed）编译与 internal/server 包测试全绿，既有 embed_frontend_prod_test.go 零改动且 9 条用例全绿"
    requirement: EMBED-SUBPATH-01
    verification:
      - kind: other
        ref: "go build ./... && go build -tags embed ./... && go vet ./internal/server/ ./cmd/ && go test ./internal/server/... -count=1 && go test -tags embed ./internal/server/... -count=1"
        status: pass
    human_judgment: false

duration: 19min
completed: 2026-10-20
status: complete
---

# Quick Task 261020-d0d Summary

**后端 /xingran 子路径剥离兜底（API/uploads 前缀）+ 根路径 302 到 /xingran/ + 可照抄的 nginx 剥前缀站点配置**

## Performance

- **Duration:** 19 min
- **Started:** 2026-10-20T01:22:00Z
- **Completed:** 2026-10-20T01:41:26Z
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments

- 修复 embed 部署直连 :9000 时「前端 API 全 404 + Router basename 不匹配白屏」：`/xingran/api/v1/...` 与 `/xingran/uploads/...` 现在真正落到 `engine.Group("/api/v1")` 与 `engine.Static("/uploads")`。
- 根路径语义拆分：`GET /` 无 `X-Forwarded-Prefix` → 302 `Location: /xingran/`（浏览器落到 basename 匹配地址）；带该头（nginx 剥前缀场景）→ 直接返回 index.html，不产生重定向死循环。
- 新增 `deploy/xingran.nginx.conf`（全仓第一份 nginx 配置）：upstream、根 302、`/xingran` 补斜杠 301、剥前缀 `proxy_pass .../9000/`、强制 `X-Forwarded-Prefix /xingran`、WebSocket 透传与超时。
- 双构建（默认 + `-tags embed`）编译、`go vet`、`internal/server` 包测试全绿；既有 `internal/server/embed_frontend_prod_test.go` 零改动、9 条用例全绿；新增 5 条默认构建用例 + 3 条 embed 用例。

## Task Commits

Each task was committed atomically:

1. **Task 1: 后端子路径剥离中间件 + 根路径重定向（含双构建测试）** — `f194d156` (test, RED) + `b66ffdd9` (feat, GREEN)
2. **Task 2: 新增 nginx 站点配置示例（剥前缀 + 转发前缀头）** — `eea54134` (feat)
3. **Task 3: 双构建回归 + 变更面核对** — 纯验证，无代码变更，无提交

### TDD 记录

- **RED**（`f194d156`）：先落 `internal/server/subpath_test.go`（5 例）+ `internal/server/subpath_embed_test.go`（3 例）+ 判定骨架。实测红：`TestSubPath_Handler_RewritesBackendPrefixes`（404≠200）、`TestSubPath_Middleware_RewritesBackendPrefixes`（路径未改写）、`TestSubPath_RootRedirectPath`（返回空串）、embed 侧 `TestSubPath_ServeRoot_RedirectsToSubPath`（200≠302）—— 全部是**目标断言失败**，无 build failure；既有 9 条 embed 用例在 RED 阶段仍为绿。
- **GREEN**（`b66ffdd9`）：实现 `stripBackendSubPath` 白名单判定、两个剥离入口、`rootRedirectPath`、embed 侧 `ServeRoot`；全部转绿。
- **REFACTOR**：无额外重构提交；`gofmt` 对新文件全部干净。

## Files Created/Modified

- `internal/server/subpath.go`（新）— 共享 `frontendSubPath` / `stripSubPath`（从 embed prod 文件搬移，无 build tag，双构建共用）+ `subPathBackendPrefixes` 白名单 + `stripBackendSubPath` 判定 + `StripSubPathHandler`（HTTP 边界包装器）+ `StripSubPathMiddleware`（gin 中间件）+ `rootRedirectPath`。
- `internal/server/subpath_test.go`（新）— 默认构建即可跑的 5 条用例：路由层前缀重写、中间件路径改写、不可重写路径清单、根路径重定向判定、`stripSubPath` 表驱动。
- `internal/server/subpath_embed_test.go`（新）— `//go:build embed`：根路径 302、带 `X-Forwarded-Prefix` 不重定向、跟随重定向后无二次重定向。
- `internal/server/embed_frontend_prod.go` — 删除搬走的 `frontendSubPath` / `stripSubPath`；新增 `ServeRoot`（302 或 index.html）。
- `internal/server/embed_frontend.go` — 新增 `ServeRoot`（开发态 503，文案与 `ServeFrontend` 逐字一致，不引入新行为）。
- `cmd/main.go` — `setupGinEngine` 注册 `engine.Use(server.StripSubPathMiddleware())`（Recovery 之后、Cors 之前）；`engine.GET("/", server.ServeRoot)`；`startServer` 的 `Handler` 改为 `server.StripSubPathHandler(engine)`。`/assets`、`/index.html`、`NoRoute` 三行逐字未动。
- `deploy/xingran.nginx.conf`（新）— nginx 站点配置示例。

## Decisions Made

1. **前缀剥离的生效点放在 gin 之外**（`StripSubPathHandler` 包装 `*gin.Engine`，挂在 `http.Server.Handler`）。理由：gin v1.10 `handleHTTPRequest` 先用 `c.Request.URL.Path` 查路由树、再执行 handlers 链，中间件里的改写**不会**改变路由匹配结果 —— 计划的前提在此不成立（见 Deviation 1）。
2. **保留计划指定的 `engine.Use(server.StripSubPathMiddleware())` 调用点**，与包装器共用同一个 `stripBackendSubPath` 判定：包装器负责「路由可见」，中间件负责「handler 可见」，两处规则同源，不会漂移。
3. **白名单保持 `/api` + `/uploads`**：`/assets` 刻意不纳入（已由 `ServeFrontend`/`ServeSPA` 内部的 `stripSubPath` 处理），SPA 路由 `/xingran/dashboard` 行为逐字不变，相似前缀 `/xingranfoo` 不被误剥。
4. **根路径重定向以 `X-Forwarded-Prefix` 为唯一豁免信号**，并由 nginx conf 强制设置该头（`proxy_set_header X-Forwarded-Prefix /xingran;`），从配置侧闭环「不产生重定向死循环」。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] 计划依赖的「gin 中间件改写路径即可改变路由匹配」前提在 gin v1.10 不成立**
- **Found during:** Task 1（RED 阶段实测：中间件改写后请求仍 404）
- **Issue:** gin v1.10 `Engine.handleHTTPRequest`（`gin@v1.10.0/gin.go:605-671`）先用 `c.Request.URL.Path` 查路由树（:626），命中后才 `c.Next()` 执行 handlers 链。全局中间件（`engine.Use`）属于**已命中路由**的 handler 链，因此中间件里改写 `c.Request.URL.Path` 不会让 `/xingran/api/v1/...` 命中 `engine.Group("/api/v1")` —— 请求会先落到 NoRoute。这同时使 `engine.Static("/uploads")` 取不到文件（gin static handler 用请求路径做 `http.StripPrefix` 裁剪）。
- **Fix:** 在 `internal/server/subpath.go` 增加 `StripSubPathHandler(next http.Handler) http.Handler`，在进入 gin 之前改写 `URL.Path`/`RawPath`；`cmd/main.go` 的 `startServer` 把 `Handler` 由 `engine` 改为 `server.StripSubPathHandler(engine)`。判定规则与计划一致（白名单 + 尾斜杠 + 相似前缀不误剥），并与 `StripSubPathMiddleware` 共用 `stripBackendSubPath`。计划的中间件按原样保留（`engine.Use(server.StripSubPathMiddleware())` 调用点逐字保留，计划的静态核对项仍为 1 命中）。
- **Files modified:** `internal/server/subpath.go`, `cmd/main.go`
- **Verification:** `TestSubPath_Handler_RewritesBackendPrefixes`（`/xingran/api/v1/probe` → 200 body `/api/v1/probe`；`/xingran/uploads/a.png` → 200 body `/uploads/a.png`）；冒烟（按 `cmd/main.go` 装配重建引擎）`POST /xingran/api/v1/system/auth/encryption-config` → 200（路由命中，非 404）。
- **Committed in:** `b66ffdd9`

**2. [Rule 1 - Bug] 计划把「路由层前缀重写」写成中间件用例，实际需拆成两层**
- **Found during:** Task 1（测试设计）
- **Issue:** 计划的 `TestSubPath_Middleware_RewritesBackendPrefixes` 期望「注册 `/api/v1/probe` 路由 → 请求 `/xingran/api/v1/probe` 命中它」，这要求中间件本身能改变路由匹配；修正后该能力属于 HTTP 边界包装器。
- **Fix:** 拆成两条用例：`TestSubPath_Handler_RewritesBackendPrefixes`（包装器 → 路由命中）与 `TestSubPath_Middleware_RewritesBackendPrefixes`（中间件 → handler 可见路径被改写、`RequestURI` 保持原值）；计划要求的「不可重写路径」清单与「根路径重定向判定」两条用例原样保留。
- **Files modified:** `internal/server/subpath_test.go`
- **Verification:** 默认构建 5/5 通过；embed 构建 8/8（含既有 3 条子路径用例）通过。
- **Committed in:** `f194d156` / `b66ffdd9`

**3. [Rule 3 - Blocking] 派发的工作树基线与声明的期望基线不一致**
- **Found during:** 开工前的 worktree 分支检查
- **Issue:** 工作树 HEAD 为 `31c7eee`（`xingran-merge-20261016` 的分支尖端），比声明的期望基线 `f5b6a2b`（main）落后 106 个提交；计划上下文里的 `internal/server/xingran-react-frontend/dist` 与 `frontendSubPath` 在该基线上尚不存在。
- **Fix:** 执行 `git merge --ff-only main` —— 快进到声明的期望基线（`31c7eee` 是 `f5b6a2b` 的严格祖先，无提交丢失、无历史改写，`31c7eee` 仍由 `backup-pre-xingran-merge-20260924-120705` / `xingran-merge-20261016` 两个分支保留）。未使用 hard-reset / update-ref 等状态改写原语。
- **Files modified:** 无（仅工作树基线）
- **Verification:** `git rev-parse HEAD` == `f5b6a2b5642135fa23b1377301096eb6c131d6b2`（与派发的 EXPECTED_BASE 一致）
- **Committed in:** 不适用（基线对齐，非内容变更）

**4. [Rule 3 - Blocking] 计划文件在 git 对象库中不存在（派发说明的物化路径失效）**
- **Found during:** 开工前的计划物化
- **Issue:** 派发说明要求从 `f5b6a2b:.planning/quick/261020-d0d-.../261020-d0d-PLAN.md` 物化，但该路径在 `f5b6a2b` 及全部 ref 中均不存在（`git rev-list --all --objects | grep 261020` 无命中）—— 计划只是主仓库工作区里一个未提交的文件。
- **Fix:** 从主仓库工作区同名文件复制到工作树 `.planning/quick/261020-d0d-embed-xingran-api-xingran-nginx-conf/261020-d0d-PLAN.md`（内容未改动，docs 产物不提交，由 orchestrator 统一提交）。
- **Files modified:** `.planning/quick/.../261020-d0d-PLAN.md`（仅工作区物化，未提交）
- **Verification:** 文件存在，25932 字节，与主仓库工作区副本一致
- **Committed in:** 不适用（docs 产物）

**5. [Rule 3 - Blocking] embed 构建所需的 `internal/server/xingran-react-frontend/dist` 在工作树缺失**
- **Found during:** Task 1 双构建准备
- **Issue:** `f5b6a2b` 已把误提交的 `internal/server/ruoyi-react-frontend/dist`（143 个构建产物）从 git 索引移除，工作树内没有 dist → `go build -tags embed ./...` 会因 `pattern all:xingran-react-frontend/dist: no matching files found` 失败。
- **Fix:** 从主仓库工作区复制已构建的 `internal/server/xingran-react-frontend/dist` 到工作树（该路径被 `.gitignore` 第 11 行 `internal/server/xingran-react-frontend/` 覆盖，不进入 git 状态，也不触发前端重建）。
- **Files modified:** 无（gitignored 构建产物）
- **Verification:** `go build -tags embed ./...` 通过；`go test -tags embed ./internal/server/...` 14/14 通过（含 3 条依赖 dist 内容的用例，未 skip）
- **Committed in:** 不适用（gitignored 产物）

### 环境与工具限制（非计划内改动，仅记录）

- **`gsd_run` CLI 在本环境不存在**：`gsd_run check tdd-red-evidence`（RED 证据机器校验）无法执行，RED 证据改为人工记录真实 `go test -v` 输出（目标测试名 + 断言失败信息 + 无 build failure）。同理，`state.*` / `roadmap.*` / `commit` 等 SDK 动词不可用，STATE.md 的表格行由本次手工追加（未提交）。
- **`cmd/main.go` 存在既有的 gofmt import 排序问题**（`internal/server` 与 `internal/models` 顺序颠倒），在基线 `f5b6a2b` 即存在（`git show f5b6a2b:cmd/main.go | gofmt -d` 同样输出该 diff），属既有代码风格问题、不在本次范围内，未改动以免制造无关 diff。

---

**Total deviations:** 5 auto-fixed（2 × Rule 1 bug、3 × Rule 3 blocking）
**Impact on plan:** 全部为达成计划既定目标（API 不再 404、根路径 302、nginx 无死循环）所必需；文件范围严格等于计划 frontmatter 的 7 个文件，`.planning/phases/59-*/` 与 `ruoyi-react-frontend/` 零改动。

## Issues Encountered

- **计划前提与 gin 实际行为不符**（中间件改写路径不影响路由匹配）：已按 Deviation 1 修正机制，并用测试把「路由层重写」与「handler 层重写」分开锁定。
- **工作树基线落后 106 个提交 + 计划文件未入库 + embed dist 未 staged**：三项均在开工前/构建前发现并按 Deviation 3/4/5 处理，未产生内容越界改动。
- **RED 证据机器校验器不可用**（`gsd_run` 缺失）：以真实测试输出人工留证，RED 仍满足「目标断言失败、无 build failure」的实质要求。

## Known Stubs

无。`internal/server/subpath.go` 的全部导出/包内符号均已实现并被用例覆盖；`serveFile` / `serveContent` / `ServeFrontend` / `ServeSPA` 等既有实现未改动。

## Threat Surface

本次未新增网络端点、未新增鉴权路径、未改 `engine.Static` 的文件读取逻辑。新增的路径改写面（`/xingran/api`、`/xingran/uploads`）受白名单约束：

- `stripBackendSubPath` 只在 `rest == pre || strings.HasPrefix(rest, pre+"/")` 时命中，`/xingranfoo/...`、`/apixyz` 均不命中（`TestSubPath_Middleware_LeavesOtherPathsUnchanged` 逐条锁死）。
- 改写后的路径只会更短（去掉 `/xingran`），不会把请求映射到新的前缀面；鉴权中间件按 handler 判定，改写不改变其行为；`RequestURI` 保持原值供审计留痕。
- `X-Forwarded-Prefix` 只影响「是否重定向」这一个表现，不放宽任何鉴权、不新增路由可达面（可被客户端伪造，退化结果 = 改前行为）。

## User Setup Required

None - no external service configuration required.

（运维侧可选：`cp deploy/xingran.nginx.conf /etc/nginx/conf.d/xingran.conf && nginx -t && systemctl reload nginx`；不装 nginx 时直连 :9000 即可，根路径会 302 到 `/xingran/`。）

## Next Phase Readiness

- 后端子路径兜底已就位，前端零改动（dist 里 base 已是 `/xingran`，无需重建前端）。
- 未验证项（需真环境）：真实 nginx 剥前缀部署下的 `curl` 冒烟（计划标注为可选）—— 本次以「按 `cmd/main.go` 装配重建引擎」的等价冒烟替代通过，未启动真实后端进程（仓库无 `configs/config.yaml`，启动会连生产 PG/Redis，副作用不可接受）。
- 未提交项：`.planning/quick/261020-d0d-*/`（PLAN + SUMMARY）与 STATE.md 表格行，由 orchestrator 统一 docs 提交。

---
*Phase: quick-261020-d0d*
*Plan: 01*
*Commits: `f194d156` (test/RED) → `b66ffdd9` (feat/GREEN) → `eea54134` (feat/nginx)*
*plan_head_before: f5b6a2b5642135fa23b1377301096eb6c131d6b2*
*plan_head_after: eea5413448352b153e8a7a14b187bb1a6b392ae3*
*Completed: 2026-10-20*

## Self-Check: PASSED

- 计划 frontmatter 的 7 个文件全部存在且已提交：`internal/server/subpath.go`、`internal/server/subpath_test.go`、`internal/server/subpath_embed_test.go`、`internal/server/embed_frontend_prod.go`、`internal/server/embed_frontend.go`、`cmd/main.go`、`deploy/xingran.nginx.conf`。
- 3 个提交均在 HEAD 历史中：`f194d156`、`b66ffdd9`、`eea54134`（`git rev-list --count f5b6a2b..HEAD` = 3，与 frontmatter `actuals.commits: 3` 一致）。
- 变更面核对：`git diff --stat f5b6a2b..HEAD` 恰好列出上述 7 个文件；`.planning/phases/59-*/` 与 `ruoyi-react-frontend/` 零改动；`internal/server/embed_frontend_prod_test.go` 零改动。
- 静态核对：`grep -rn "func stripSubPath" internal/server/` = 1；`grep -rn "frontendSubPath = " internal/server/` = 1；`grep -c "engine.Use(server.StripSubPathMiddleware())" cmd/main.go` = 1；`grep -c 'engine.GET("/", server.ServeRoot)' cmd/main.go` = 1。
- 构建与测试：`go build ./...`、`go build -tags embed ./...`、`go vet ./internal/server/ ./cmd/` 全部通过；`go test ./internal/server/... -count=1` 与 `go test -tags embed ./internal/server/... -count=1` 全绿（embed 14/14，默认 5/5）；`go test ./cmd/... -count=1` 通过。
- 工作树残留：仅 `.planning/STATE.md`（已改）与 `.planning/quick/261020-d0d-*/`（未跟踪）两项 docs 产物，按约定不提交，由 orchestrator 统一提交。
