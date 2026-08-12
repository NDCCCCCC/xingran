---
slug: login-404-blank-after-auth
status: resolved
trigger: "启动构建好的exe程序（包含前后端），访问login报错，访问根目录/会正常跳转到login,但是登录后页面为空白，无法打开任何页面"
created: 2026-05-22
updated: 2026-05-22
---

# Debug Session: login-404-blank-after-auth

## Symptoms

- **Expected behavior:** 构建的exe包含前后端，访问 /login 应正常显示登录页，登录后应正常展示各页面
- **Actual behavior:**
  1. 直接访问 /login 返回 404 Not Found
  2. 访问根目录 / 能正常跳转到 /login（登录页正常显示）
  3. 登录成功后页面为空白，无法打开任何页面
- **Error messages:** 404 Not Found（直接访问 /login 时）
- **Timeline:** 一直都有这个问题，从未成功运行过构建后的版本。开发模式（npm run dev）可能正常
- **Reproduction:** 启动构建的exe → 浏览器访问

## Current Focus

- hypothesis: confirmed - two root causes
- test: code analysis of router.go, main.go, embed_frontend_prod.go, .env.production
- expecting: fix SPA routing and API base URL
- next_action: complete

## Evidence

- timestamp: 2026-05-22
  type: code_analysis
  file: cmd/main.go:188-190
  finding: "Only 3 routes mapped to ServeFrontend: GET /, GET /index.html, GET /assets/*filepath. No catch-all for SPA client routes like /login, /dashboard"

- timestamp: 2026-05-22
  type: code_analysis
  file: internal/server/embed_frontend_prod.go
  finding: "ServeFrontend already has SPA fallback logic (returns index.html for extensionless paths), but it is only invoked from 3 explicit routes, not from a catch-all"

- timestamp: 2026-05-22
  type: code_analysis
  file: xingran-react-frontend/.env.production
  finding: "VITE_API_BASE_URL was commented out, falling back to hardcoded http://10.62.10.33:9000/api/v1 in api.ts. .env.local overrides with same hardcoded URL"

- timestamp: 2026-05-22
  type: code_analysis
  file: xingran-react-frontend/src/App.tsx
  finding: "Uses BrowserRouter (HTML5 history API). Requires server-side SPA fallback to work"

- timestamp: 2026-05-22
  type: code_analysis
  file: xingran-react-frontend/src/lib/api.ts:42,160
  finding: "baseURL: import.meta.env.VITE_API_BASE_URL || 'http://10.62.10.33:9000/api/v1' - fallback to hardcoded IP"

## Eliminated

- React Router misconfiguration (BrowserRouter is correct, just needs server fallback)
- Frontend build output issues (dist/ contains correct files)
- Embed directive issues (embed_frontend_prod.go is correct)

## Resolution

- root_cause: "Two issues: (1) No SPA catch-all route in Go server — only /, /index.html, and /assets/*filepath were mapped to ServeFrontend, so /login, /dashboard etc. returned 404. (2) Production .env.production had VITE_API_BASE_URL commented out, causing frontend API calls to use a hardcoded IP address (10.62.10.33:9000) instead of relative /api/v1 path, so API calls failed when the exe runs on a different machine."
- fix: "(1) Added engine.NoRoute(server.ServeSPA) in cmd/main.go to serve index.html for all unmatched routes, with ServeSPA function that skips /api/, /uploads/, /swagger/, /debug/ paths. Added ServeSPA stub in dev-mode embed_frontend.go for compilation. (2) Set VITE_API_BASE_URL=/api/v1 in .env.production to use relative paths so the embedded Go server handles API requests."
- verification: "Go compilation verified for both embed and non-embed builds. Frontend needs rebuild (cd xingran-react-frontend && npm run build) then rebuild exe with scripts/build-embedded.bat"
- files_changed:
  - cmd/main.go (added NoRoute handler, reordered static routes after API routes)
  - internal/server/embed_frontend_prod.go (added ServeSPA function)
  - internal/server/embed_frontend.go (added ServeSPA stub for dev mode)
  - xingran-react-frontend/.env.production (uncommented VITE_API_BASE_URL=/api/v1)
