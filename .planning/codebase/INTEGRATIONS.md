# External Integrations

## Databases

### PostgreSQL 18
- **Primary data store** for all application data
- Connection: configurable via `configs/config.yaml` or env vars (`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`)
- ORM: GORM with auto-migration (`internal/core/db/migrations/`)
- UUID primary keys with `gen_random_uuid()`
- Soft delete pattern with `deleted_at` column
- Table prefix: `sys_` for system tables

### SQLite
- **Alternate/development database** via `modernc.org/sqlite`
- Pure Go implementation (no CGO required)
- Config: `database.type: "sqlite"` in config.yaml

## Cache Layer

### Redis 7.4
- **L2 distributed cache** with `xingran:` key prefix
- Connection: configurable via config or env vars (`REDIS_URL`, `REDIS_PASSWORD`)
- Features:
  - Two-tier caching: L1 (in-memory) + L2 (Redis)
  - Async L2 writer pool with retry and exponential backoff
  - Cache warming on startup (configurable)
  - Dynamic TTL configuration via `CacheConfigService`
- Key patterns:
  - `dict:type:all`, `dict:data:{type}` — Dictionary cache
  - `dept:tree`, `dept:list` — Department cache
  - `role:all`, `role:detail:{id}` — Role cache
  - `menu:tree`, `menu:user:{userId}` — Menu cache
  - `user:detail:{id}` — User cache
  - `post:all`, `post:detail:{id}` — Post cache
  - `cache:config:*` — Cache TTL configuration

## Authentication & Directory Services

### Active Directory / LDAP
- **LDAP client** with connection pooling (`internal/services/addomain/`)
- Operations: user sync, group sync, OU sync
- Protocols: LDAP, LDAPS, StartTLS
- Warning: Currently uses `InsecureSkipVerify: true` for LDAPS/StartTLS
- Scheduled sync via cron jobs

### JWT Authentication
- **Dual-token system**: Access token (2h) + Refresh token (7d)
- SM2-signed tokens (configurable)
- Token refresh via `TokenManager` in frontend authStore

## Network Device Integration

### SSH/Telnet (Scrapli)
- **Network device command execution** via `scrapligo`
- Used for: switch/router configuration, port status collection
- Template-based command parsing with TextFSM

### SNMP (GoSNMP)
- **SNMP v2c/v3 operations** for device monitoring
- Port collection, interface status queries

## External APIs

### Baidu Maps API
- **Geocoding service** (`internal/services/operations/geocoding_service.go`)
- Converts addresses to GPS coordinates
- In-memory cache: 30 min TTL
- Concurrency limit: 5 parallel requests
- API key from config (`baidu.map_ak`) or env var (`BAIDU_MAP_AK`)
- Proxied through backend to avoid CORS: `POST /ops/building/geocode`

## Security Integration

### National Cryptography (SM2/SM3/SM4)
- **SM2** — Asymmetric encryption for key exchange and signatures
  - Fixed key pair in config (production: use env vars)
  - Public key endpoint: `GET /api/v1/system/auth/public-key`
- **SM3** — Hash algorithm for password storage
- **SM4-CBC** — Symmetric encryption for request bodies
  - Hybrid encryption: SM4 encrypts body, SM2 encrypts SM4 key
  - Anti-replay: timestamp (300s window) + nonce
  - Configurable exclude paths in `security.request_encryption.exclude_paths`

## Real-time Communication

### WebSocket
- **gorilla/websocket** for real-time features (`internal/websocket/`)
- Used for: server monitoring, notification push, device status updates

## Excel Import/Export

### Excelize v2
- **Batch import** with validation and geocoding
- **Template download** for building, floor, workstation, infopoint
- Configurable column mappings in `internal/services/operations/excel_config.go`

## Monitoring

### Server Monitoring
- System metrics collection (CPU, memory, disk, network)
- Cache monitoring endpoint for Redis key management

### Logging
- Logrus structured logging with Lumberjack rotation
- Operation logs, login logs stored in database
