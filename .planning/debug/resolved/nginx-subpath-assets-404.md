---
status: resolved
trigger: "腾讯云 nginx 将 127.0.0.1:9000 代理到 127.0.0.1:8000/xingran/，前端静态资源全部 404（index-*.js / vendor-*.js / *.css）"
created: 2026-08-19
updated: 2026-08-19
resolved: 2026-08-19
---

# Debug Session: nginx-subpath-assets-404

## Symptoms

**Expected behavior:**
通过 nginx 子路径代理（http://server:8000/xingran/）访问前端应用，页面及全部 JS/CSS 静态资源正常加载。

**Actual behavior:**
HTML 页面本身能返回，但所有构建产物静态资源 404：
- index-DOdldQWy.js
- vendor-markdown-CTNRp5o7.js
- vendor-echarts-D4lsRrLc.js
- vendor-react-Cth8yn6f.js
- vendor-three-FNS0ngf0.js
- vendor-react-DONvmT-2.css
- index-otKD5oqr.css

**Error messages:**
浏览器控制台 `Failed to load resource: the server responded with a status of 404 ()` ×9（上述资源）。

**关键观察（用户提供）:**
404 资源的完整请求 URL 形态是 `/assets/xxx` —— **不带 `/xingran/` 前缀**，指向站点根路径。

**Timeline:**
CI/CD 流水线全部跑通后，首次在腾讯云服务器上以 nginx 子路径方式部署时发现。

**Reproduction:**
- 腾讯云服务器：nginx 监听 8000，将 `location /xingran/` 代理到 `127.0.0.1:9000`（Go 后端，服务前端构建产物）
- 访问 `http://server:8000/xingran/` → HTML 返回，静态资源请求发往 `/assets/...` → 404

**nginx 配置（用户提供）:**

```nginx
# ProMgr location
# upstream: 127.0.0.1:28080
location /xingran/ {
    proxy_pass http://127.0.0.1:9000/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_buffering off;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
}
```

**nginx 语义分析:**
- `proxy_pass http://127.0.0.1:9000/;` 带 URI（尾斜杠）→ nginx 将 `/xingran/xxx` 重写为 `/xxx` 转发后端。即后端收到的路径**不含** `/xingran/` 前缀。此写法与"后端无感知子路径"部署模型一致，配置本身合理。
- 注释暗示同 server 块默认 upstream 可能是 127.0.0.1:28080（ProMgr）。不匹配 `/xingran/` 的请求（如 `/assets/...`）会落到 server 级默认 location → 404。与用户观察吻合。
- 推论：**修复的关键是让浏览器发出的资源/路由 URL 都带 `/xingran/` 前缀**（前端 base），而不是改 proxy_pass。

**遗留风险（调查时须一并核实）:**
- 前端 API 请求路径（lib/api.ts baseURL，后端路由前缀 /api/v1）同样不带 /xingran/ 前缀 → 若无对应 nginx location，静态资源修好后 API 仍会 404。
- React Router（App.tsx）的 basename 是否需要配 /xingran/（SPA 内部导航与 history 刷新）。

## Architecture context (for investigator)

- 前端: `xingran-react-frontend/` — Vite 7.2 构建，产物带 hash 文件名（index-*.js / vendor-*.js）
- 后端: Go :9000 静态服务前端构建产物（具体挂载方式待查）
- 最近 deploy 相关提交（同一部署工作流）: 50ac7d0 (release job -R repo flag), a967035 (bundle deploy-remote.sh), 4a937ac (anonymous public-asset pull via gh-proxy), 9ea181c (resolve asset URL runner-side)
- 调查起点: `xingran-react-frontend/vite.config.ts` 的 `base` 配置、`.env.production`、CI 构建是否传 `--base=/xingran/`、Go 后端静态文件路由挂载路径、部署脚本（scripts/、.github/workflows/）

**Primary hypothesis (待验证):**
Vite 构建时 `base` 为默认 `/`，HTML 中资源引用为绝对路径 `/assets/...`，浏览器请求落在本站点根（:8000/assets/...），不匹配 nginx `location /xingran/`，落到默认 location → 404。与 nginx proxy_pass 尾斜杠写法是否叠加影响待 nginx 配置确认。

## Current Focus

reasoning_checkpoint:
  hypothesis: "Vite 构建使用默认 base '/'，dist/index.html 以根绝对路径 /assets/... 引用资源；浏览器在 http://server:8000/xingran/ 下向 :8000/assets/... 发请求，不匹配 nginx location /xingran/，落到 server 默认 location（ProMgr :28080 upstream）→ 404。同理 /api/v1 与 /uploads 请求也会 404（潜在同类问题）"
  confirming_evidence:
    - "vite.config.ts 无 base 选项（默认 '/'）；CI deploy.yml 与手动 scripts/build/build-*.bat 均跑裸 `npm run build`，无 --base 覆盖"
    - "实测本地 dist/index.html（含 staged internal/server/xingran-react-frontend/dist/）：src=\"/assets/index-4Q8dxL48.js\"、href=\"/vite.svg\" 全部根绝对路径；vendor hash（CTNRp5o7/D4lsRrLc/DONvmT-2）与用户上报 404 文件名完全吻合"
    - "cmd/main.go:233-240 挂载 /assets/*filepath、/、/index.html + NoRoute→ServeSPA，后端只在根路径服务嵌入产物，对 /xingran/ 无感知 — 与 nginx 尾斜杠 proxy_pass 剥前缀模型一致"
    - ".env.production VITE_API_BASE_URL=/api/v1（根绝对）→ API 请求在子路径部署下同样落空"
  falsification_test: "若 dist/index.html 实际引用带 /xingran/ 前缀，或 vite/CI 存在 base 覆盖，则假设被推翻（已查：均不存在）"
  fix_rationale: "让浏览器发出的所有 URL（资源/路由/API/uploads）统一带 /xingran/ 前缀：vite base + Router basename + VITE_API_BASE_URL 前缀 + /uploads 构造点前缀。nginx 剥前缀后后端按原样服务，无需改 nginx。后端 embed 服务加可选前缀剥离，保证直连 :9000 根路径访问（本地验证工作流）不回归。"
  blind_spots: "无法在本地复现远端 nginx 环境做端到端验证（nginx 在腾讯云服务器上）；仪表盘 WS（/ws/dashboard）走独立 VITE_WS_BASE_PATH，且 nginx 未配 Upgrade 头，WS 经代理的可用性需服务器侧确认"

- hypothesis: CONFIRMED（见 reasoning_checkpoint）
- test: n/a（证据直接观测）
- expecting: n/a
- next_action: 实施修复 — vite.config.ts loadEnv base / .env.production VITE_BASE+API前缀 / App.tsx basename / embed_frontend_prod.go 前缀剥离 / useImageUpload+ExecutionDetailModal uploads 前缀

## Evidence

- timestamp: 2026-08-19T Phase1
  checked: xingran-react-frontend/vite.config.ts
  found: 无 `base` 选项 → 构建默认 base '/'；dev proxy /api、/uploads → localhost:9000（dev 模式不受影响）
  implication: 构建产物资源引用必为根绝对路径 /assets/...

- timestamp: 2026-08-19T Phase1
  checked: .env.production / .env.development
  found: 两者的 VITE_API_BASE_URL 均为 /api/v1（根绝对）；无任何 VITE_BASE/VITE_WS_* 变量
  implication: 生产 API 请求在 nginx 子路径下同样不匹配 location /xingran/ → 潜在第二处 404

- timestamp: 2026-08-19T Phase1
  checked: .github/workflows/deploy.yml（L59-63）+ scripts/build/build-linux.bat + build-embedded.bat
  found: 全部执行裸 `npm run build`，无 --base / VITE_BASE 覆盖；CI 将 dist 拷入 internal/server/xingran-react-frontend/ 后 go:embed 进二进制
  implication: 手动 bat 构建与 CI 构建产物一致地缺前缀 → 修复必须放进 .env.production（所有 production 构建自动生效），不能只改 CI

- timestamp: 2026-08-19T Phase1
  checked: 本地 dist/index.html 与 staged internal/server/xingran-react-frontend/dist/index.html
  found: src="/assets/index-4Q8dxL48.js"、href="/assets/vendor-markdown-CTNRp5o7.js"、href="/vite.svg" 等，全部根绝对；vendor hash 与用户 404 清单吻合
  implication: 直接观测到缺陷产物 — 假设的核心证据（smoking gun）

- timestamp: 2026-08-19T Phase1
  checked: cmd/main.go L233-240 + internal/server/embed_frontend_prod.go
  found: engine.GET("/assets/*filepath"|"/"|"/index.html", ServeFrontend) + NoRoute(ServeSPA)；embed 读取 dist/<path>，SPA 无扩展名路径回 index.html
  implication: 后端只按根路径服务；nginx 剥前缀模型下后端无需感知 /xingran/，但若给 base 加前缀，直连 :9000 根路径会 404 → 需在 ServeFrontend/ServeSPA 加可选前缀剥离保持本地工作流

- timestamp: 2026-08-19T Phase1
  checked: src/App.tsx（BrowserRouter）、src/lib/api.ts、src/lib/api/networkApi.ts、src/lib/opsApi.ts(blobAxios)、src/components/shared/FileUpload.tsx
  found: Router 无 basename；全部 axios/XHR/fetch 实例统一读 VITE_API_BASE_URL || "/api/v1"
  implication: Router 需加 basename=import.meta.env.BASE_URL；API 只需改 .env.production 一处

- timestamp: 2026-08-19T Phase1
  checked: src/lib/noticeApi.ts buildWebSocketUrl、src/hooks/useRealtimeUpdates.ts、src/components/dashboard/utils/dataFetcher.ts
  found: 通知 WS 从 VITE_API_BASE_URL 推导（去 /api/vN + http→ws）→ 前缀自动继承；仪表盘 WS 用独立 VITE_WS_BASE_PATH 默认 /ws（根绝对）
  implication: 通知 WS 随 API base 自动修复（nginx 需 Upgrade 头，写入用户指引）；仪表盘 WS 为遗留问题仅记录

- timestamp: 2026-08-19T Phase1
  checked: /uploads 构造点（useImageUpload.ts L58/60/96、ExecutionDetailModal.tsx L108-119、router/DynamicRoutes.tsx L168）
  found: 前两处构造根绝对 /uploads/... URL；DynamicRoutes 用 React Router location.pathname（basename 剥离后）判断 /uploads → 无需改
  implication: 需给构造点加 import.meta.env.BASE_URL 前缀（dev base '/' 时行为逐字节不变）；DynamicRoutes 不动

## Eliminated

- hypothesis: "只改 vite base 即可,无需改后端"
  evidence: "若不剥 /xingran/,开发者直接跑 :9000 exe 调试时,index.html 引用的 /xingran/assets/... 会落到 NoRoute→ServeSPA→静态资源分支→fs.ReadFile(dist/xingran/assets/...),然后 404。本地工作流(本地直连 :9000 验证嵌入产物)回归。"
  timestamp: 2026-08-19T Phase5
- hypothesis: "只在 CI 加 VITE_BASE,本地构建不受影响"
  evidence: "scripts/build/build-linux.bat / build-embedded.bat 与 CI 同样跑裸 `npm run build`,且本地手动部署场景仍存在;若 CI-only 设置,这些路径产生静默不带前缀的产物,在 nginx 下和 CI 一致地 404 — 必须放进 .env.production,所有 production 构建统一生效。"
  timestamp: 2026-08-19T Phase5
- hypothesis: "/uploads 构造点保持 /uploads/... 根绝对,nginx 仅需加一条 location"
  evidence: "nginx 在腾讯云服务器上,不在本仓库改;且 /uploads 是浏览器从产物中读取的 URL,根绝对同样落不到 location /xingran/。修复面必须含前端,纯服务端改动不可行。"
  timestamp: 2026-08-19T Phase5

## Resolution

root_cause: |
  Vite 构建默认 base='/',dist/index.html 中所有静态资源 URL 均为根绝对(/assets/...);
  浏览器在 nginx 子路径部署 http://server:8000/xingran/ 下加载页面后,将这些 URL 解析为
  http://server:8000/assets/...,不匹配 nginx `location /xingran/`,落到 server 默认
  location(ProMgr :28080 upstream 或 404)。同类问题潜在:API 调用(VITE_API_BASE_URL
  同样根绝对)、上传图片 URL(useImageUpload / ExecutionDetailModal 硬编码 /uploads/)。
fix: |
  1) vite.config.ts 改为函数式,loadEnv 读 VITE_BASE(默认 '/')作为 base。
  2) xingran-react-frontend/.env.production 新增 VITE_BASE=/xingran/,
     VITE_API_BASE_URL=/xingran/api/v1(两者保持同步)。
  3) App.tsx BrowserRouter 加 basename={import.meta.env.BASE_URL !== '/' ? import.meta.env.BASE_URL : undefined}。
  4) useImageUpload.ts 与 ExecutionDetailModal.tsx 中 /uploads/... 构造点改用
     `${import.meta.env.BASE_URL}uploads/`,内部上传结果 storagePath 剥离统一用
     .split('/uploads/').pop()(兼容旧/新两种 URL 形式)。
  5) internal/server/embed_frontend_prod.go 新增 frontendSubPath='/xingran' + stripSubPath
     helper,ServeFrontend 与 ServeSPA 入口处先剥前缀再处理。这样:
     - nginx 剥前缀后后端见到的是无前缀路径,行为与前一致(向后兼容);
     - 直连 :9000 调试时(开发者绕过 nginx),/xingran/assets/xxx 也能被剥成 /assets/xxx 服务,
       本地嵌入产物测试工作流不受回归。
  6) 新增 internal/server/embed_frontend_prod_test.go(build tag embed),
     覆盖 stripSubPath 边界 + 根/子路径资源服务 + SPA 路由 + API 路径剥离后正确 404,
     不硬编码 hash 文件名(从 staged dist/assets 取首个 .js)。
verification: |
  - go build ./...                                          OK
  - go test -tags=embed -v ./internal/server/               9/9 PASS
    (TestServeFrontend_Root|SubPath|AssetRootAndSubPath|Missing404,
     TestServeSPA_AssetSubPath|SPARoute|APIPath|NonGET,
     TestStripSubPath_EdgeCases)
  - xingran-react-frontend: npx tsc -b                       OK(无类型错误)
  - npm run build                                           OK,1m 36s
    验证 dist/index.html 现在引用全部带 /xingran/ 前缀:
      href="/xingran/vite.svg"
      src="/xingran/assets/index-vAqeTFw1.js"
      href="/xingran/assets/vendor-markdown-CTNRp5o7.js"  ← 用户上报 404 的同名 hash
      href="/xingran/assets/vendor-echarts-D4lsRrLc.js"    ← 同上
      href="/xingran/assets/vendor-react-DONvmT-2.css"     ← 同上
      (vendor-echarts/markdown/react-css 三个 hash 与用户 404 清单逐字吻合,验证改的就是这处产物)
  - npm run lint                                            0 errors,1031 warnings(全部预存 any 噪音)
  - 端到端(nginx)需在腾讯云服务器上确认 — 属于人工验证步骤
files_changed:
  - xingran-react-frontend/vite.config.ts
  - xingran-react-frontend/.env.production
  - xingran-react-frontend/src/App.tsx
  - xingran-react-frontend/src/hooks/useImageUpload.ts
  - xingran-react-frontend/src/pages/operations/rpa/executions/ExecutionDetailModal.tsx
  - internal/server/embed_frontend_prod.go
  - internal/server/embed_frontend_prod_test.go

## Server-side guidance (for human verification, NOT a repo change)

部署服务器侧的 nginx 现在缺两条配置,需要由用户在腾讯云服务器上手工补齐或调整:

1. **/uploads 子路径 location(否则上传的图片/附件资源会 404)**:
   nginx 默认 server 上加一个 location,把 /uploads/ 也代理到后端:
   ```
   location /uploads/ {
       proxy_pass http://127.0.0.1:9000/;
       proxy_set_header Host $host;
       proxy_set_header X-Real-IP $remote_addr;
       proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
       proxy_set_header X-Forwarded-Proto $scheme;
   }
   ```
   (后端 internal/api/router.go 已挂载 /uploads/* 静态服务。)

2. **WebSocket 升级头(否则通知中心 WS 与仪表盘实时推送会在 nginx 之后断连)**:
   `location /xingran/` 内需要:
   ```
   proxy_http_version 1.1;
   proxy_set_header Upgrade $http_upgrade;
   proxy_set_header Connection $connection_upgrade;

   # 在 http {} 块里定义 map:
   map $http_upgrade $connection_upgrade {
       default upgrade;
       ''      close;
   }
   ```
   用户当前配置里 `proxy_set_header Connection "";` 会把 WS 升级请求的
   Connection 头清空,与后端 WS 路由握手失败 — 这是 nginx 侧独立问题,
   与本次前端 base 修复无关,但若 /xingran/ 下启用 WS 推送必须一起改。

3. **/xingran (无尾斜杠) 重定向(可选,改善直接访问的体验)**:
   ```
   location = /xingran { return 301 /xingran/; }
   ```

4. **/ws/dashboard 等仪表盘 WS 路径(若启用)**:前端 useRealtimeUpdates 默认
   VITE_WS_BASE_PATH=/ws,部署子路径下需要设 VITE_WS_BASE_PATH=/xingran/ws
   并在 nginx 增加对应 location 转发 WS(同上 Connection/Upgrade)。本次修复
   未改动仪表盘 WS,属遗留项,不影响本 issue 报告的 assets 404。
