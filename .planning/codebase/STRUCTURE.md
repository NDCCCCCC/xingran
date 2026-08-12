# Structure

## Root Directory Layout

```
xingran-go-backend/
├── cmd/                          # Application entry point
│   └── main.go                   # Server bootstrap
├── configs/                      # Configuration files
│   ├── config.yaml               # Default config
│   ├── config.dev.yaml           # Development overrides
│   └── config.prod.yaml          # Production overrides
├── docs/                         # Project documentation (Chinese)
├── internal/                     # Private application code
│   ├── api/                      # HTTP layer
│   │   ├── router.go             # Main router registration
│   │   └── v1/                   # API version 1 handlers
│   │       ├── system/           # User, Role, Dept, Menu, Dict, Post, Config, Notice
│   │       ├── operations/       # Building, Floor, Workstation, ServerRoom
│   │       ├── scheduler/        # Job management
│   │       ├── workorder/        # Work orders
│   │       ├── network/          # Network devices
│   │       ├── monitor/          # Server/cache monitoring
│   │       ├── duty/             # Duty roster
│   │       ├── knowledge/        # Knowledge base
│   │       └── rpa/              # RPA module
│   ├── services/                 # Business logic layer
│   │   ├── system/               # System services (user, role, dept, etc.)
│   │   ├── operations/           # Operations services (building, floor, etc.)
│   │   ├── scheduler/            # Job scheduling services
│   │   ├── workorder/            # Work order services
│   │   ├── network/              # Network device services
│   │   ├── monitor/              # Monitoring services
│   │   ├── duty/                 # Duty management services
│   │   ├── knowledge/            # Knowledge base services
│   │   ├── addomain/             # AD/LDAP domain services
│   │   ├── portcollection/       # Port collection services
│   │   ├── common/               # Shared service utilities
│   │   ├── base/                 # Base service patterns
│   │   ├── rpa/                  # RPA services
│   │   ├── data_cache_service.go # Generic cache service
│   │   ├── cache_config_service.go # Dynamic cache TTL
│   │   └── *_cache_service.go    # Legacy cache services (dept, role, dict, etc.)
│   ├── models/                   # GORM data models
│   ├── core/                     # Core infrastructure
│   │   ├── core.go               # DI container (DB, Cache, JWT, Scheduler)
│   │   ├── security/             # JWT, password management
│   │   └── db/                   # Database connection + migrations
│   ├── config/                   # Config structures
│   ├── constants/                # Application constants
│   ├── device/                   # Device management (Scrapli, TextFSM)
│   ├── collectors/               # Data collectors
│   ├── scheduler/                # Cron job scheduler engine
│   ├── templates/                # TextFSM templates
│   ├── utils/                    # Internal utilities
│   ├── websocket/                # WebSocket service
│   └── pkg/                      # Internal shared packages
├── pkg/                          # Public reusable packages
│   ├── cache/                    # Cache interface (Redis + Memory)
│   ├── crypto/                   # SM2/SM4 encryption, nonce storage
│   ├── middleware/                # Auth, CORS, logging, encryption middleware
│   ├── permission/               # RBAC permission definitions
│   ├── response/                 # Standard response wrappers
│   ├── query/                    # Query builders
│   ├── errors/                   # Error types
│   ├── logger/                   # Logging setup
│   ├── captcha/                  # Captcha generation
│   ├── gormutil/                 # GORM utilities
│   └── time/                     # Time utilities
├── scripts/                      # Build and utility scripts
├── xingran-react-frontend/         # React frontend application
│   ├── src/
│   │   ├── pages/                # Route pages by module
│   │   │   ├── system/           # System management pages
│   │   │   ├── operations/       # Operations pages
│   │   │   ├── monitor/          # Monitoring pages
│   │   │   ├── network/          # Network device pages
│   │   │   ├── workorder/        # Work order pages
│   │   │   ├── duty/             # Duty roster pages
│   │   │   ├── knowledge/        # Knowledge base pages
│   │   │   ├── login/            # Login page
│   │   │   ├── dashboard-system/ # Dashboard
│   │   │   ├── ad-domain/        # AD domain pages
│   │   │   ├── settings/         # Settings pages
│   │   │   ├── profile/          # User profile
│   │   │   └── my-notices/       # Notifications
│   │   ├── components/           # Reusable components
│   │   │   ├── layout/           # Layout (HybridLayout, Sidebar, Header)
│   │   │   ├── three/            # 3D visualization components
│   │   │   ├── cad-editor/       # CAD-style floor plan editor
│   │   │   ├── operations/       # Operations-specific components
│   │   │   ├── DeptTree/         # Department tree selector
│   │   │   ├── CronSelector/     # Cron expression builder
│   │   │   ├── dashboard/        # Dashboard widgets
│   │   │   └── shared/           # Shared components
│   │   ├── store/                # Zustand state stores
│   │   ├── hooks/                # Custom React hooks
│   │   ├── lib/                  # API clients
│   │   │   ├── api.ts            # Core API with encryption + token refresh
│   │   │   └── opsApi.ts         # Operations module CRUD factory
│   │   ├── services/             # Service layer (operations, cache)
│   │   ├── utils/                # Utilities (sm2, sm4, authHelpers, etc.)
│   │   ├── types/                # TypeScript type definitions
│   │   ├── design-system/        # Design tokens and themes
│   │   ├── constants/            # App constants
│   │   └── router/               # Route configuration
│   └── package.json
└── CLAUDE.md                     # Claude Code instructions
```

## Key File Locations

| Purpose | Path |
|---------|------|
| Server entry | `cmd/main.go` |
| Main router | `internal/api/router.go` |
| Core DI | `internal/core/core.go` |
| Config types | `internal/config/` |
| Database setup | `internal/core/db/` |
| Migrations | `internal/core/db/migrations/` |
| Models | `internal/models/` |
| Base model | `internal/models/base.go` |
| Cache interface | `pkg/cache/` |
| Auth middleware | `pkg/middleware/` |
| Permission defs | `pkg/permission/permissions.go` |
| Response helpers | `pkg/response/` |
| Frontend entry | `xingran-react-frontend/src/main.tsx` |
| Frontend router | `xingran-react-frontend/src/App.tsx` |
| API wrapper | `xingran-react-frontend/src/lib/api.ts` |
| Ops API factory | `xingran-react-frontend/src/lib/opsApi.ts` |
| State stores | `xingran-react-frontend/src/store/` |

## Naming Conventions

- **Go files**: `snake_case.go` (e.g., `user_handler.go`, `building_service.go`)
- **Go structs**: PascalCase (e.g., `UserHandler`, `BuildingService`)
- **Go interfaces**: PascalCase with `er` suffix or `Service` suffix (e.g., `UserService`)
- **Go tests**: `*_test.go` in same package
- **TypeScript files**: `camelCase.ts` or `PascalCase.tsx` for components
- **API routes**: `/api/v1/{module}/{resource}` pattern
- **Database tables**: `sys_` prefix, `snake_case`
- **Cache keys**: `module:subkey:id` pattern, prefix `xingran:` added automatically
- **Config keys**: `yaml_case` in config files
