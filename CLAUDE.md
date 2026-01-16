# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

RuoYi-Next is a modern enterprise permission management system based on the RuoYi framework. It uses a Go backend + React frontend architecture with national cryptography (SM2/SM3/SM4) for security.

**Tech Stack:**
- Backend: Go 1.24, Gin, GORM, PostgreSQL 18, Redis 7.4
- Frontend: React 19, TypeScript 5, Vite 7, Ant Design 6, Zustand 5
- Security: SM2/SM3/SM4 national cryptography algorithms

---

## Common Commands

### Backend (Go)

```bash
# Build for Windows
go build -o ruoyi-backend.exe ./cmd/main.go

# Build for Linux (production)
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -o ruoyi-backend-linux ./cmd/main.go

# Run tests
go test ./...
go test ./internal/api/v1/

# Run backend
.\ruoyi-backend.exe
```

### Frontend (React)

```bash
cd ruoyi-react-frontend

# Install dependencies
npm install

# Development server (runs on http://localhost:4000)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Lint code
npm run lint
```

### Configuration

- Backend config: `configs/config.yaml`, `configs/config.dev.yaml`, `configs/config.prod.yaml`
- Frontend env: `ruoyi-react-frontend/.env.development`
- Environment variables: `.env` (DB_HOST, DB_PASSWORD, REDIS_URL, etc.)

---

## Architecture

### Backend Layer Structure

```
internal/
├── api/v1/          # HTTP handlers (45+ handlers)
├── services/        # Business logic layer
├── models/          # Data models (GORM)
├── core/            # Core modules
├── config/          # Configuration management
├── device/          # Device management
├── collectors/      # Data collectors
├── scheduler/       # Cron job scheduler
├── templates/       # Template parsing
├── utils/           # Utilities
└── websocket/       # WebSocket service

pkg/                 # Public packages (reusable)
├── cache/           # Redis + Memory cache
├── crypto/          # SM2/SM4 encryption
├── middleware/      # Auth, CORS, logging
├── permission/      # RBAC permission control
├── query/           # Query builders
└── response/        # Response wrappers
```

**Key Architecture Patterns:**
- **Service Layer Pattern**: Business logic in `internal/services/`, accessed via handlers
- **Middleware Chain**: Auth → Permission → Encryption → Handler
- **Dual Token System**: Access Token (short-lived) + Refresh Token (long-lived)
- **Request Encryption**: SM2+SM4 hybrid encryption for sensitive endpoints

### Frontend Structure

```
src/
├── pages/           # Route pages (organized by module)
├── components/      # Reusable components
├── store/           # Zustand state stores (7 stores)
├── hooks/           # Custom React hooks
├── services/        # API service layer
├── utils/           # Utilities (sm2.ts, sm4.ts, etc.)
├── types/           # TypeScript definitions
└── design-system/   # Design tokens and theme
```

**State Management (Zustand):**
- `authStore` - Authentication state
- `layoutStore` - Layout preferences
- `menuStore` - Menu navigation
- `noticeStore` - Notifications
- `settingsStore` - User settings
- `tabsStore` - Tab navigation
- `themeStore` - Theme (light/dark)

---

## Critical Development Rules

### Status Value Convention (IMPORTANT)

**Universal Rule:** `0 = enabled/normal/visible, 1 = disabled/stopped/hidden`

**Exception - Menu Visibility:** `1 = visible, 0 = hidden` (boolean semantics)

| Module | Field | 0 Value | 1 Value | Default |
|--------|-------|---------|---------|---------|
| User | status | Enabled | Disabled | 0 |
| Role | status | Normal | Stopped | 0 |
| Menu | status | Normal | Stopped | 0 |
| Menu | visible | Hidden | Visible | 1 |
| Dept | status | Normal | Stopped | 0 |
| Post | status | Normal | Stopped | 0 |

### API Response Format

All API responses follow this structure:

```json
{
    "code": 0,           // 0 = success, other = error
    "message": "success",
    "data": {},          // Business data
    "timestamp": 1766380800,
    "request_id": "uuid-string"
}
```

**Success Code:** `0`
**Common Error Codes:** `400` (param error), `401` (unauthorized), `403` (forbidden), `1001` (invalid params), `1007` (invalid token)

### Frontend API Calling

**Use wrapped API functions, NOT raw axios:**

```typescript
// ✅ CORRECT - use wrapped functions
import { post } from '@/lib/api';
const result = await post('/system/users/list', params);
const users = result.data.list; // Direct access

// ❌ WRONG - don't use api instance directly
const response = await api.post('/system/users/list', params);
if (response.data.code === 0) { } // Manual check unnecessary
```

### Pagination Convention

**Request:**
```typescript
interface PageParams {
    current: number;    // Page number, starts from 1
    pageSize: number;
}
```

**Response:**
```typescript
interface PageData<T> {
    list: T[];
    total: number;
    current: number;
    pageSize: number;
}
```

### Database Naming

- Tables: `sys_` prefix, lowercase with underscores
- Fields: lowercase with underscores
- Primary Keys: UUID with `gen_random_uuid()` default
- Timestamps: `created_at`, `updated_at`, `deleted_at` (soft delete)

### Go Code Patterns

```go
// Service interface pattern
type UserService interface {
    CreateUser(ctx context.Context, user *User) error
}

// Handler pattern
func (h *UserHandler) CreateUser(c *gin.Context) {
    // 1. Bind request
    // 2. Call service
    // 3. Return response
}
```

---

## Security Considerations

### Request Encryption (SM2+SM4)

- **Enabled for sensitive endpoints** (configurable via `security.request_encryption.exclude_paths`)
- **SM4-CBC** encrypts request body
- **SM2** encrypts SM4 key
- **Anti-replay:** timestamp (300s window) + nonce

### LDAP/AD Connections

**CRITICAL:** Current code uses `InsecureSkipVerify: true` for LDAPS/StartTLS. This should be addressed for production.

### Authentication

- JWT dual-token mechanism
- SM3 for password hashing
- RBAC permission model
- All API endpoints require auth (except configured exclusions)

---

## Module-Specific Notes

### Network Device Management

- **Scrapli** for device connections
- **TextFSM** templates for command parsing
- **Template cache** for performance
- **Port collection** with parser abstraction

### AD Domain Management

- **LDAP client** with connection pooling
- **Sync tasks** via scheduler
- **Modular services**: config, group, user, OU, sync

### Workorder Management

- **Assignment logic** with rotation
- **Periodic workorders** with Cron scheduling
- **Statistics** and rating system

---

## Key File Locations

**Backend:**
- Entry: `cmd/main.go`
- Router: `internal/api/router.go`
- Models: `internal/models/base.go`
- Encryption: `pkg/crypto/request_encryption.go`
- Cache: `pkg/cache/`

**Frontend:**
- Entry: `ruoyi-react-frontend/src/main.tsx`
- Router: `ruoyi-react-frontend/src/App.tsx`
- API wrapper: `ruoyi-react-frontend/src/lib/api.ts`
- Crypto utils: `ruoyi-react-frontend/src/utils/sm2.ts`, `sm4.ts`
- Stores: `ruoyi-react-frontend/src/store/`

**Documentation:**
- Architecture: `docs/项目概述和架构设计.md`
- Standards: `docs/开发规范.md`
- API: `docs/API响应规范.md`
- Security: `docs/安全和认证设计（国密）.md`

---

## Development Workflow

1. **Feature development:** Create branch from `dev` or `feature-dev`
2. **Backend:** Add handler in `internal/api/v1/`, service in `internal/services/`
3. **Frontend:** Add page in `src/pages/`, components in `src/components/`
4. **API:** Use wrapped functions from `@/lib/api`
5. **State:** Use Zustand stores for global state
6. **Testing:** Add tests in `*_test.go` files
7. **Build:** Run `npm run build` for frontend, `go build` for backend
