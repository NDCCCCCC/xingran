# Concerns

## Security

### Critical: LDAP InsecureSkipVerify
- **Location:** `internal/services/addomain/`
- **Issue:** `InsecureSkipVerify: true` for LDAPS/StartTLS connections
- **Impact:** MITM attacks on AD/LDAP connections possible
- **Recommendation:** Use proper TLS certificate validation in production

### Config Contains Secrets
- **Location:** `configs/config.yaml`
- **Issue:** Database passwords, Redis passwords, SM2 key pairs, SM4 keys in plaintext YAML
- **Impact:** Credentials exposed if config file is committed/leaked
- **Recommendation:** Use environment variables exclusively for secrets; remove from config files

### SM2 Fixed Key Pair
- **Location:** `configs/config.yaml` → `jwt.sm2_private_key`
- **Issue:** Fixed SM2 key pair in config file
- **Impact:** If config is leaked, all JWT tokens can be forged
- **Recommendation:** Rotate keys; use HSM or env-based key management

## Architecture

### Dual Cache Architecture Confusion
- **Location:** Root-level `*_cache_service.go` vs `internal/services/system/*_cache_impl.go`
- **Issue:** Two parallel cache implementations coexist
  - Legacy: `internal/services/dept_service.go`, `role_cache_service.go`, etc.
  - New: `internal/services/system/dept_cache_impl.go`, etc.
- **Impact:** Developers unsure which to use; potential inconsistency
- **Recommendation:** Complete migration to new pattern; remove legacy files

### Core Singleton Pattern
- **Location:** `internal/core/core.go`
- **Issue:** Large monolithic `Core` struct holds all dependencies; routers manually instantiate services
- **Impact:** Difficult to test; tight coupling
- **Recommendation:** Consider wire/dig DI framework for large-scale refactoring

### Scheduler Namespace Confusion
- **Issue:** `internal/scheduler/` (engine) vs `api/v1/scheduler/` (CRUD UI)
- **Impact:** Developers may confuse execution engine with job management API
- **Recommendation:** Rename one; e.g., `internal/cron/` for the engine

## Technical Debt

### Sparse Test Coverage
- **Fact:** Only 17 test files for 485 Go source files
- **Impact:** High regression risk on changes
- **Areas most at risk:** handlers, middleware, non-operations services
- **See:** TESTING.md for details

### Temporary Files Risk
- **Location:** Project root
- **Issue:** `temp_*.go`, `test_*.go` files in root cause `main redeclared` build errors
- **Impact:** Build failures when these files exist
- **Recommendation:** Add gitignore pattern; clean up regularly

### Frontend useEffect Stability
- **Location:** Various React components
- **Issue:** Objects/arrays in useEffect dependency arrays cause infinite re-render loops
- **Impact:** Infinite API request loops, degraded performance
- **Recommendation:** Audit all useEffect hooks; memoize dependencies

## Performance

### Excel Import
- **Location:** `internal/services/operations/excel_service.go`
- **Issue:** Large Excel imports processed synchronously
- **Impact:** Request timeouts on large imports; memory pressure
- **Recommendation:** Consider async processing with progress reporting

### Geocoding Rate Limiting
- **Location:** `internal/services/operations/geocoding_service.go`
- **Issue:** In-memory rate limiter (5 concurrent); Baidu Maps API limits may still be hit
- **Impact:** Import failures when geocoding many addresses
- **Recommendation:** Add queuing/retry for geocoding failures

### L2 Cache Writer
- **Location:** `pkg/cache/`
- **Issue:** Async L2 writes with retry; potential data inconsistency window
- **Impact:** Stale cache reads between DB write and Redis write
- **Mitigation:** Fallback sync write on queue full

## Code Quality

### Mixed Language Comments
- **Issue:** Comments and docs in both Chinese and English
- **Impact:** Readability for international teams
- **Recommendation:** Establish language standard for code comments

### No API Versioning Strategy
- **Issue:** All routes under `/api/v1/` with no plan for v2
- **Impact:** Breaking changes require careful coordination
- **Recommendation:** Plan backward-compatible API evolution strategy

### Error Message Consistency
- **Issue:** Mix of Chinese error messages ("请求参数错误") and English errors
- **Impact:** Inconsistent user experience
- **Recommendation:** Centralize error message constants with i18n support
