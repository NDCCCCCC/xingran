# Tech Stack

## Overview
XingRan-Next is a full-stack enterprise permission management system with Go backend and React frontend, featuring national cryptography (SM2/SM3/SM4) for security.

## Backend Stack

### Language & Runtime
- **Go 1.24** (toolchain go1.24.5)
- Module: `github.com/xingran-next/xingran-go-backend`
- Entry point: `cmd/main.go`

### Web Framework
- **Gin v1.10.0** — HTTP routing, middleware, JSON binding
- **gin-contrib/cors v1.7.0** — CORS middleware
- **gin-contrib/gzip v0.0.6** — Response compression
- **swaggo/gin-swagger v1.6.0** — Swagger UI integration

### Database
- **PostgreSQL** (primary) via `gorm.io/driver/postgres v1.5.9`
- **SQLite** (alternate) via `gorm.io/driver/sqlite v1.5.4` + `modernc.org/sqlite v1.40.1`
- **GORM v1.30.5** — ORM with auto-migration
- Connection pooling: configurable max_open_conns, max_idle_conns, max_lifetime

### Cache
- **Redis 7.4** via `redis/go-redis/v9 v9.7.0`
- Two-tier: L1 (in-memory) + L2 (Redis) with async writer pool
- Prefix: `xingran:` for all keys
- L2 writer pool: configurable workers, queue size, retry with exponential backoff

### Authentication & Security
- **JWT v5.2.1** — Dual token (access + refresh)
- **tjfoc/gmsm v1.4.1** — SM2/SM3/SM4 national cryptography
- **SM2** — Key exchange, digital signatures
- **SM3** — Password hashing
- **SM4-CBC** — Request body encryption
- **base64Captcha v1.3.6** — Login captcha

### Task Scheduling
- **robfig/cron/v3 v3.0.1** — Cron job engine
- Built-in scheduler in `internal/scheduler/`
- AD sync tasks, periodic work orders

### Network Device Management
- **scrapli/scrapligo v1.3.3** — Network device SSH/Telnet connections
- **gosnmp/gosnmp v1.35.0** — SNMP operations
- **sirikothe/gotextfsm v1.0.1** — TextFSM template parsing

### Excel Processing
- **xuri/excelize/v2 v2.10.0** — Excel import/export
- Supports building, floor, workstation, infopoint batch operations

### Real-time
- **gorilla/websocket v1.5.3** — WebSocket connections for real-time features

### Configuration
- **spf13/viper v1.19.0** — YAML config with env var overrides
- Config files: `configs/config.yaml`, `config.dev.yaml`, `config.prod.yaml`

### Logging
- **sirupsen/logrus v1.9.3** — Structured logging
- **natefinch/lumberjack v2.2.1** — Log rotation

### Other Libraries
- **google/uuid v1.6.0** — UUID generation (primary keys)
- **go-ldap/ldap/v3 v3.4.12** — AD/LDAP integration
- **golang.org/x/sync v0.19.0** — Concurrency primitives (errgroup, singleflight)
- **swaggo/swag v1.16.4** — Swagger doc generation

## Frontend Stack

### Core
- **React 19.2** with TypeScript 5.9
- **Vite 7.2** — Build tool and dev server
- **ESLint** — Code linting

### UI Framework
- **Ant Design 6.1** — Component library
- **Tailwind CSS 4.1.18** — Utility-first styling

### State Management
- **Zustand 5.0.9** — Lightweight state stores (7 stores)
- **TanStack React Query v5.90.12** — Server state management

### Routing
- **react-router-dom v7.10.1** — Client-side routing

### 3D Visualization
- **Three.js v0.182.0** — 3D rendering
- **@react-three/fiber v9.5.0** — React renderer for Three.js
- **@react-three/drei v10.7.7** — Three.js helpers
- Used for building/floor 3D visualization and CAD-style floor plan editor

### Charts & Data
- **ECharts 6.0** via `echarts-for-react v3.0.5`
- **xlsx v0.18.5** — Client-side Excel processing

### Maps
- **@uiw/react-baidu-map v2.7.5** — Baidu Maps integration
- **react-baidu-map v1.3.5** — Alternative map component

### Other Frontend
- **sm-crypto v0.3.13** — Client-side SM2/SM4 encryption
- **axios v1.13.2** — HTTP client
- **dayjs v1.11.19** — Date/time handling
- **react-markdown v10.1.0** — Markdown rendering
- **@uiw/react-md-editor v4.0.11** — Markdown editor
- **react-grid-layout v2.2.2** — Drag-and-drop grid
- **jsonata v2.1.0** — JSON query/transformation
- **cron-parser/cron-validate** — Cron expression handling
- **vitest** — Testing framework

### Code Statistics
- **485 Go files** (backend)
- **506 TS/TSX files** (frontend)
