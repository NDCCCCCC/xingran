# Phase 16: API 密钥管理功能 - Pattern Map

**Mapped:** 2026-05-18
**Files analyzed:** 12 (9 new files, 3 modified files)
**Analogs found:** 12 / 12

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/models/api_key.go` | model | CRUD | `internal/models/user.go` | exact |
| `internal/models/api_key_usage_log.go` | model | event-driven | `internal/models/user.go` | role-match |
| `internal/services/system/apikey_service.go` | service | CRUD | `internal/services/system/user_service.go` | exact |
| `internal/api/v1/system/apikey_handler.go` | handler | request-response | `internal/api/v1/system/user_handler.go` | exact |
| `internal/api/v1/system/apikey_router.go` | router | request-response | `internal/api/v1/system/user_router.go` | exact |
| `internal/middleware/apikey.go` | middleware | request-response | `pkg/middleware/auth.go` | role-match |
| `internal/services/rate_limiter.go` | service | event-driven | No direct analog | new-pattern |
| `xingran-react-frontend/src/types/apikey.ts` | types | type-definitions | `xingran-react-frontend/src/types/system.ts` | exact |
| `xingran-react-frontend/src/api/apikey.ts` | api-client | request-response | `xingran-react-frontend/src/lib/api.ts` | exact |
| `xingran-react-frontend/src/pages/system/apikeys/index.tsx` | component | request-response | `xingran-react-frontend/src/pages/system/user/index.tsx` | exact |
| `internal/api/router.go` | router-modification | routing | `internal/api/router.go` | self-reference |
| `pkg/middleware/permission.go` | middleware-modification | request-response | `pkg/middleware/permission.go` | self-reference |

## Pattern Assignments

### `internal/models/api_key.go` (model, CRUD)

**Analog:** `internal/models/user.go`

**Imports pattern** (lines 1-6):
```go
package models

import (
	"time"
)
```

**Base model embedding** (lines 9-12):
```go
type User struct {
	BaseModel
	Username      string     `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password      string     `gorm:"size:128;not null" json:"-"`
	// ... other fields
}
```

**Field naming conventions** (lines 10-27):
```go
// Use snake_case for JSON tags
// Use pointer types for optional fields (*string, *time.Time)
// Use enum types for status fields (UserStatus)
// GORM tags: size, not null, uniqueIndex, default
// JSON tags: -, omitempty for sensitive/optional fields
```

**Table naming** (lines 50-52):
```go
func (User) TableName() string {
	return "sys_user"
}
```

---

### `internal/models/api_key_usage_log.go` (model, event-driven)

**Analog:** `internal/models/user.go`

**Minimal model pattern** (lines 36-47):
```go
type UserRole struct {
	UserID    string    `gorm:"type:uuid;not null" json:"userId"`
	RoleID    string    `gorm:"type:uuid;not null" json:"roleId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (UserRole) TableName() string {
	return "sys_user_role"
}
```

**Adaptation for API key usage log:**
- Add foreign key fields: `api_key_id`, `user_id`
- Add request metadata: `method`, `path`, `status_code`, `client_ip`
- Add performance tracking: `duration`, `success`
- Use automatic timestamps: `created_at`

---

### `internal/services/system/apikey_service.go` (service, CRUD)

**Analog:** `internal/services/system/user_service.go`

**Service interface pattern** (lines 20-30):
```go
type UserService interface {
	Create(ctx context.Context, user *requests.UserCreateRequest) error
	Update(ctx context.Context, user *requests.UserUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	List(ctx context.Context, params requests.UserListParams) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
	UpdateStatus(ctx context.Context, id string, status int) error
	ResetPassword(ctx context.Context, id string, newPassword string) error
}
```

**Private implementation struct** (lines 33-36):
```go
type userService struct {
	db         *gorm.DB
	pwdManager PasswordManager
}
```

**Constructor with dependency injection** (lines 38-44):
```go
func NewUserService(db *gorm.DB, pwdManager PasswordManager) UserService {
	return &userService{
		db:         db,
		pwdManager: pwdManager,
	}
}
```

**Context propagation** (line 57):
```go
func (s *userService) Create(ctx context.Context, req *requests.UserCreateRequest) error {
	// Always use ctx for database operations
	s.db.WithContext(ctx)
}
```

**Error handling with apperrors** (lines 60-65):
```go
if count > 0 {
	return apperrors.UserExistsWithUsername(req.Username)
}
// ... later ...
if err != nil {
	return apperrors.DatabaseError(err)
}
```

**Transaction pattern** (lines 76-113):
```go
err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
	// Create main record
	if err := tx.Create(&user).Error; err != nil {
		return apperrors.DatabaseError(err)
	}
	
	// Batch insert associations to avoid N+1
	if len(req.RoleIds) > 0 {
		userRoles := make([]models.UserRole, len(req.RoleIds))
		for i, roleID := range req.RoleIds {
			userRoles[i] = models.UserRole{
				UserID: user.ID,
				RoleID: roleID,
			}
		}
		if err := tx.Create(&userRoles).Error; err != nil {
			return apperrors.DatabaseError(err)
		}
	}
	return nil
})
```

**Pagination result pattern** (lines 46-52):
```go
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Current  int         `json:"current"`
	PageSize int         `json:"pageSize"`
}
```

**List query building pattern** (lines 253-298):
```go
query := s.db.WithContext(ctx).Model(&models.User{})

// Add filters
if params.Username != nil && *params.Username != "" {
	query = query.Where("username LIKE ?", "%"+*params.Username+"%")
}
if params.Status != nil {
	query = query.Where("status = ?", *params.Status)
}

// Count total
if err := query.Count(&total).Error; err != nil {
	return nil, apperrors.DatabaseError(err)
}

// Paginate
offset := (params.Current - 1) * params.PageSize
if err := query.
	Preload("Dept").
	Order("created_at DESC").
	Offset(offset).Limit(params.PageSize).
	Find(&list).Error; err != nil {
	return nil, apperrors.DatabaseError(err)
}
```

**Service methods for API keys** (adaptation):
- `CreateAPIKey` - Generate random key, validate scopes
- `ValidateAPIKey` - Check format, expiration, status, IP whitelist
- `ListUsageLogs` - Query usage logs with pagination
- `GetUsageLogSummary` - Aggregate statistics

---

### `internal/api/v1/system/apikey_handler.go` (handler, request-response)

**Analog:** `internal/api/v1/system/user_handler.go`

**Handler struct pattern** (lines 11-18):
```go
type UserHandler struct {
	service systemServices.UserService
}

func NewUserHandler(service systemServices.UserService) *UserHandler {
	return &UserHandler{service: service}
}
```

**Request binding pattern** (lines 33-37):
```go
func (h *UserHandler) Create(c *gin.Context) {
	var req requests.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}
```

**Service call pattern** (lines 39-42):
```go
user, err := h.service.Create(c.Request.Context(), &req)
if err != nil {
	response.Error(c, err)
	return
}
```

**Response pattern** (line 44):
```go
response.Success(c, gin.H{"message": "创建成功"})
```

**Parameter extraction pattern** (lines 90-94):
```go
func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}
```

**Context propagation** (line 96):
```go
user, err := h.service.GetByID(c.Request.Context(), id)
```

**Pagination handler pattern** (lines 56-76):
```go
func (h *UserHandler) List(c *gin.Context) {
	var params requests.UserListParams
	if err := c.ShouldBindJSON(&params); err != nil {
		params = requests.DefaultUserListParams()
	}

	current, pageSize := params.GetPagination()
	params.Current = current
	params.PageSize = pageSize

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}
```

**Handler methods for API keys** (adaptation):
- `Create` - Return full key only on creation
- `List` - Mask keys (show first 12 chars only)
- `GetByID` - Return masked key
- `Update` - Update name, scopes, IP whitelist, status
- `Delete` - Soft delete
- `ToggleStatus` - Enable/disable
- `ListUsageLogs` - Pagination
- `GetUsageSummary` - Aggregated stats

---

### `internal/api/v1/system/apikey_router.go` (router, request-response)

**Analog:** `internal/api/v1/system/user_router.go`

**Router setup pattern** (lines 9-42):
```go
func SetupUserRouter(r *gin.RouterGroup, core *core.Core) {
	// Create cache provider adapter
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// Create service with optional cache
	var userService systemServices.UserService
	if core.DataCacheService != nil {
		userService = systemServices.NewUserServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
			systemServices.NewPasswordManagerAdapter(core.PwdManager),
		)
	} else {
		userService = systemServices.NewUserService(
			core.DB.GetDB(),
			systemServices.NewPasswordManagerAdapter(core.PwdManager),
		)
	}

	// Create handler
	handler := NewUserHandler(userService)

	// Register routes
	r.POST("", handler.Create)
	r.POST("/list", handler.List)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)
	r.POST("/:id/status", handler.UpdateStatus)
	r.POST("/:id/reset-password", handler.ResetPassword)
}
```

**Route naming convention**:
- `POST /list` - List with pagination
- `POST /` - Create
- `POST /:id` - Get by ID
- `POST /:id/update` - Update
- `POST /:id/delete` - Delete
- `POST /:id/toggle` - Toggle status
- `POST /:id/logs` - Usage logs
- `GET /:id/summary` - Statistics

---

### `internal/middleware/apikey.go` (middleware, request-response)

**Analog:** `pkg/middleware/auth.go`

**Middleware function signature** (lines 17-36):
```go
func JWTAuth(jwtManager *security.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			response.Error(c, response.ErrUnauthorized, "缺少认证令牌")
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}

		setUserContext(c, claims)
		c.Next()
	}
}
```

**Header extraction pattern** (lines 75-92):
```go
func extractToken(c *gin.Context) string {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		return extractBearerToken(authHeader)
	}

	// WebSocket might use query param
	return c.Query("token")
}

func extractBearerToken(authHeader string) string {
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return ""
	}
	return authHeader[len(bearerPrefix):]
}
```

**Context setting pattern** (lines 95-99):
```go
func setUserContext(c *gin.Context, claims *security.CustomClaims) {
	c.Set("user_id", claims.UserID)
	c.Set("username", claims.Username)
	c.Set("roles", claims.Roles)
}
```

**API key middleware adaptation**:
- Extract from `X-API-Key` header
- Validate format: `rec_` + 64 hex chars
- Check expiration, status, IP whitelist
- Set user context from API key's user_id
- Async log usage (goroutine)

---

### `internal/services/rate_limiter.go` (service, event-driven)

**No direct analog** - New pattern for rate limiting

**Sliding window data structure** (from CONTEXT.md):
```go
type rateLimitWindow struct {
    minute []time.Time
    hour   []time.Time
    day    []time.Time
    mu     sync.Mutex
}

type RateLimiter struct {
    windows sync.Map // key -> *rateLimitWindow
    limits  map[string]RateLimit
}

type RateLimit struct {
    PerMinute int
    PerHour   int
    PerDay    int
}
```

**Rate limiting logic**:
```go
func (rl *RateLimiter) Check(key string, scope string) (bool, *RateLimitResult) {
    window := rl.getOrCreateWindow(key)
    
    window.mu.Lock()
    defer window.mu.Unlock()
    
    now := time.Now()
    
    // Clean expired entries
    window.minute = cleanOlderThan(window.minute, now.Add(-time.Minute))
    window.hour = cleanOlderThan(window.hour, now.Add(-time.Hour))
    window.day = cleanOlderThan(window.day, now.Add(-24*time.Hour))
    
    // Get limits for scope
    limit := rl.limits[scope]
    
    // Check limits
    if len(window.minute) >= limit.PerMinute {
        return false, &RateLimitResult{/* ... */}
    }
    if len(window.hour) >= limit.PerHour {
        return false, &RateLimitResult{/* ... */}
    }
    if len(window.day) >= limit.PerDay {
        return false, &RateLimitResult{/* ... */}
    }
    
    // Add current request
    window.minute = append(window.minute, now)
    window.hour = append(window.hour, now)
    window.day = append(window.day, now)
    
    return true, &RateLimitResult{
        Remaining: calculateRemaining(limit, window),
        ResetAt:   calculateReset(window),
    }
}
```

**Scope-based limits** (from CONTEXT.md):
```go
var scopeLimits = map[string]RateLimit{
    "read":  {30, 500, 5000},
    "write": {100, 1500, 15000},
    "admin": {200, 5000, 50000},
}
```

---

### `xingran-react-frontend/src/types/apikey.ts` (types, type-definitions)

**Analog:** `xingran-react-frontend/src/types/system.ts`

**Type definition pattern** (lines 10-33):
```typescript
export interface User {
  id: string;
  username: string;
  nickname?: string;
  employeeNo?: string;
  email?: string;
  phone?: string;
  avatar?: string;
  gender: 0 | 1 | 2;
  status: Status;
  deptId?: string;
  deptName?: string;
  roles: string[];
  roleIds?: string[];
  permissions: string[];
  isAdmin?: boolean;
  dataScope?: string;
  loginIp?: string;
  loginTime?: string;
  createTime: string;
  updateTime: string;
}
```

**List params pattern** (lines 38-44):
```typescript
export interface UserListParams extends PageParams {
  username?: string;
  nickname?: string;
  deptId?: string;
  status?: number;
  dateRange?: [string, string];
}
```

**API key types** (adaptation):
```typescript
export interface APIKey {
  id: string;
  name: string;
  key: string; // Masked in list
  scopes: string[];
  ip_whitelist: string[];
  inherit_perms: boolean;
  expires_at?: string;
  last_used_at?: string;
  is_active: boolean;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAPIKeyRequest {
  name: string;
  description?: string;
  scopes: string[];
  inherit_perms: boolean;
  ip_whitelist?: string[];
  expires_at?: string;
}

export interface UpdateAPIKeyRequest {
  name?: string;
  description?: string;
  scopes?: string[];
  inherit_perms?: boolean;
  ip_whitelist?: string[];
  is_active?: boolean;
}

export interface APIKeyListParams extends PageParams {
  keyword?: string;
  status?: boolean;
  scope?: string;
}

export interface APIKeyUsageLog {
  id: string;
  api_key_id: string;
  user_id: string;
  method: string;
  path: string;
  status_code: number;
  client_ip: string;
  user_agent?: string;
  duration: number;
  success: boolean;
  created_at: string;
}

export interface UsageSummary {
  total_requests: number;
  success_rate: number;
  avg_duration: number;
  requests_by_method: Record<string, number>;
  requests_by_path: Record<string, number>;
  errors_by_status: Record<number, number>;
}
```

---

### `xingran-react-frontend/src/api/apikey.ts` (api-client, request-response)

**Analog:** `xingran-react-frontend/src/lib/api.ts`

**API function pattern** (lines 399-413):
```typescript
export function get<T = unknown>(url: string, params?: unknown): Promise<BaseResponse<T>> {
	return api.get(url, { params });
}

export function post<T = unknown>(url: string, data?: unknown): Promise<BaseResponse<T>> {
	return api.post(url, data);
}

export function put<T = unknown>(url: string, data?: unknown): Promise<BaseResponse<T>> {
	return api.put(url, data);
}

export function del<T = unknown>(url: string): Promise<BaseResponse<T>> {
	return api.delete(url);
}
```

**API key API functions** (adaptation):
```typescript
import { get, post, put, del } from '@/lib/api';
import type {
	APIKey,
	CreateAPIKeyRequest,
	UpdateAPIKeyRequest,
	APIKeyListParams,
	APIKeyUsageLog,
	UsageSummary,
	BaseResponse,
	PageData,
} from '@/types/apikey';

export function listAPIKeys(params?: APIKeyListParams): Promise<BaseResponse<PageData<APIKey>>> {
	return post('/system/apikeys/list', params);
}

export function createAPIKey(data: CreateAPIKeyRequest): Promise<BaseResponse<{ key: string }>> {
	return post('/system/apikeys', data);
}

export function getAPIKey(id: string): Promise<BaseResponse<APIKey>> {
	return get(`/system/apikeys/${id}`);
}

export function updateAPIKey(id: string, data: UpdateAPIKeyRequest): Promise<BaseResponse<void>> {
	return put(`/system/apikeys/${id}`, data);
}

export function deleteAPIKey(id: string): Promise<BaseResponse<void>> {
	return del(`/system/apikeys/${id}`);
}

export function toggleAPIKeyStatus(id: string): Promise<BaseResponse<void>> {
	return post(`/system/apikeys/${id}/toggle`);
}

export function listUsageLogs(
	keyID: string,
	params?: { current: number; pageSize: number }
): Promise<BaseResponse<PageData<APIKeyUsageLog>>> {
	return post(`/system/apikeys/${keyID}/logs`, params);
}

export function getUsageSummary(keyID: string): Promise<BaseResponse<UsageSummary>> {
	return get(`/system/apikeys/${keyID}/summary`);
}
```

---

### `xingran-react-frontend/src/pages/system/apikeys/index.tsx` (component, request-response)

**Analog:** `xingran-react-frontend/src/pages/system/user/index.tsx`

**Component structure pattern** (from user management):
```typescript
import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Form, Input, Select, message } from 'antd';
import { listAPIKeys, createAPIKey, updateAPIKey, deleteAPIKey, toggleAPIKeyStatus } from '@/api/apikey';
import type { APIKey, APIKeyListParams } from '@/types/apikey';

export default function APIKeyManagement() {
	const [dataSource, setDataSource] = useState<APIKey[]>([]);
	const [loading, setLoading] = useState(false);
	const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });
	const [modalVisible, setModalVisible] = useState(false);
	const [editingRecord, setEditingRecord] = useState<APIKey | null>(null);
	const [form] = Form.useForm();

	const fetchData = async (params?: APIKeyListParams) => {
		setLoading(true);
		try {
			const result = await listAPIKeys({
				current: pagination.current,
				pageSize: pagination.pageSize,
				...params,
			});
			setDataSource(result.data.list);
			setPagination({
				...pagination,
				total: result.data.total,
			});
		} catch (error) {
			message.error('获取数据失败');
		} finally {
			setLoading(false);
		}
	};

	useEffect(() => {
		fetchData();
	}, [pagination.current, pagination.pageSize]);

	// ... handlers for create, update, delete, toggle

	const columns = [
		{ title: '名称', dataIndex: 'name', key: 'name' },
		{ title: '密钥', dataIndex: 'key', key: 'key', render: (key: string) => maskKey(key) },
		{ title: '作用域', dataIndex: 'scopes', key: 'scopes' },
		{ title: '状态', dataIndex: 'is_active', key: 'is_active' },
		{ title: '过期时间', dataIndex: 'expires_at', key: 'expires_at' },
		{ title: '操作', key: 'action', render: (_, record) => actionButtons(record) },
	];

	return (
		<div>
			<Table
				columns={columns}
				dataSource={dataSource}
				loading={loading}
				pagination={pagination}
				onChange={(p) => setPagination({ ...pagination, current: p.current, pageSize: p.pageSize })}
			/>
			<Modal visible={modalVisible} /* ... */ />
		</div>
	);
}
```

**Key features to implement**:
- Mask key display (first 12 chars only)
- Show full key only on creation (one-time display)
- Copy to clipboard button
- Scope selection (checkboxes: read, write, admin)
- IP whitelist input (comma-separated or CIDR)
- Expiration date picker
- Toggle status switch
- Usage logs modal with chart
- Statistics dashboard

---

### `internal/api/router.go` (router-modification, routing)

**Analog:** Self-reference (modification needed)

**Router registration pattern** (lines 132-146):
```go
// 用户管理
users := authorized.Group("/users")
users.Use(middleware.RequirePermissions([]string{
	string(permission.UserList),
	string(permission.UserAdd),
	string(permission.UserEdit),
	string(permission.UserView),
}, core))
// 添加数据权限中间件
users.Use(middleware.DataScopePermission(core))
{
	// 新架构：结构体Handler + Service层
	systemV1.SetupUserRouter(users, core)
	// Excel导入导出
	operations.SetupExcelRouter(users, "user", core)
}
```

**API key router registration** (adaptation):
```go
// API密钥管理
apikeys := authorized.Group("/apikeys")
apikeys.Use(middleware.RequirePermissions([]string{
	"system:apikey:list",
	"system:apikey:add",
	"system:apikey:edit",
	"system:apikey:delete",
}, core))
{
	// Setup API key router
	systemV1.SetupAPIKeyRouter(apikeys, core)
}
```

**Middleware integration**:
- Add `MultiAuth()` middleware for dual JWT + API key auth
- Add `RequireScope()` middleware for scope validation
- Add `RateLimitByScope()` middleware for rate limiting

---

### `pkg/middleware/permission.go` (middleware-modification, request-response)

**Analog:** Self-reference (modification needed)

**Permission checking pattern** (from existing code):
```go
func RequirePermissions(permissions []string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		// Check permissions...
		if !hasPermission {
			response.Error(c, response.ErrForbidden, "权限不足")
			c.Abort()
			return
		}
		c.Next()
	}
}
```

**Scope-based permission middleware** (new addition):
```go
// RequireScope checks if API key has required scope
func RequireScope(requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get scopes from context (set by API key middleware)
		scopes, exists := c.Get("scopes")
		if !exists {
			response.Error(c, response.ErrForbidden, "缺少权限作用域")
			c.Abort()
			return
		}

		scopeList, ok := scopes.([]string)
		if !ok {
			response.Error(c, response.ErrForbidden, "无效的权限作用域")
			c.Abort()
			return
		}

		// Check if required scope is in the list
		hasScope := false
		for _, scope := range scopeList {
			if scope == requiredScope || scope == "admin" {
				hasScope = true
				break
			}
		}

		if !hasScope {
			response.Error(c, response.ErrForbidden, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAPIKeyResourcePermission maps resource actions to scopes
func RequireAPIKeyResourcePermission(resource string, action string) gin.HandlerFunc {
	// Map actions to scopes
	scopeMap := map[string]map[string]string{
		"view":   {"read": "read", "write": "write", "admin": "admin"},
		"create": {"write": "write", "admin": "admin"},
		"edit":   {"write": "write", "admin": "admin"},
		"delete": {"write": "write", "admin": "admin"},
	}

	requiredScope, exists := scopeMap[action][resource]
	if !exists {
		requiredScope = "read" // Default to read scope
	}

	return RequireScope(requiredScope)
}
```

---

## Shared Patterns

### Authentication Context Setting

**Source:** `pkg/middleware/auth.go`
**Apply to:** All authentication middleware

```go
func setUserContext(c *gin.Context, userID string, username string) {
	c.Set("user_id", userID)
	c.Set("username", username)
	c.Set("authenticated", true)
	c.Set("auth_type", "api_key") // or "jwt"
}
```

---

### Response Wrapping

**Source:** `pkg/response/`
**Apply to:** All handler files

```go
// Success response
response.Success(c, data)

// Error response
response.Error(c, apperrors.ParamMissing("用户ID"))

// Paginated response
response.Page(c, result.List, result.Total, result.Current, result.PageSize)
```

---

### Error Handling

**Source:** `pkg/errors/`
**Apply to:** All service and handler files

```go
// Parameter error
return apperrors.ParamMissing("用户ID")

// Not found error
return apperrors.UserNotFoundWithID(id)

// Database error
return apperrors.DatabaseError(err)

// Wrapped error
return apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误")
```

---

### Transaction Pattern

**Source:** `internal/services/system/user_service.go`
**Apply to:** All service methods with multiple writes

```go
err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
	// Perform multiple operations
	if err := tx.Create(&record).Error; err != nil {
		return apperrors.DatabaseError(err)
	}
	// More operations...
	return nil
})
```

---

### Batch Insert Pattern

**Source:** `internal/services/system/user_service.go`
**Apply to:** All services creating associations

```go
// Avoid N+1 by batching
records := make([]Model, len(ids))
for i, id := range ids {
	records[i] = Model{
		Field1: value1,
		Field2: id,
	}
}
if err := tx.Create(&records).Error; err != nil {
	return apperrors.DatabaseError(err)
}
```

---

### Context Propagation

**Source:** All service methods
**Apply to:** All service and handler methods

```go
// Handler: pass context from request
h.service.Create(c.Request.Context(), &req)

// Service: use context in all DB operations
s.db.WithContext(ctx).Model(&Model{})
```

---

### Pagination Convention

**Source:** Frontend and backend
**Apply to:** All list endpoints

**Backend response:**
```go
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Current  int         `json:"current"`
	PageSize int         `json:"pageSize"`
}
```

**Frontend request:**
```typescript
interface PageParams {
	current: number; // Page number, starts from 1
	pageSize: number;
}
```

---

## No Analog Found

Files with no close match in the codebase (planner should use CONTEXT.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/services/rate_limiter.go` | service | event-driven | No rate limiting implementation exists yet |
| `internal/middleware/apikey.go` | middleware | request-response | No API key middleware exists yet (similar to JWT auth but different flow) |

**Rate limiter implementation notes:**
- Use sliding window algorithm (CONTEXT.md specification)
- In-memory storage with sync.Map
- Three-tier limits: minute/hour/day
- Scope-based configuration
- Auto-cleanup of expired windows

**API key middleware implementation notes:**
- Extract from `X-API-Key` header (not `Authorization`)
- Validate format: `rec_` prefix + 64 hex chars
- Check expiration, active status
- Validate IP whitelist (CIDR support)
- Async usage logging (goroutine)
- Set user context for downstream handlers

---

## Key Generation Pattern

**From CONTEXT.md specification:**

```go
import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 64 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("rec_%x", bytes), nil
}
```

---

## IP Whitelist Validation Pattern

**From CONTEXT.md specification:**

```go
import (
	"net"
)

func isIPAllowed(clientIP string, whitelist []string) bool {
	if len(whitelist) == 0 {
		return true // No whitelist means allow all
	}

	clientAddr := net.ParseIP(clientIP)
	if clientAddr == nil {
		return false
	}

	for _, allowed := range whitelist {
		// Check if it's a CIDR notation
		if strings.Contains(allowed, "/") {
			_, ipNet, err := net.ParseCIDR(allowed)
			if err != nil {
				continue
			}
			if ipNet.Contains(clientAddr) {
				return true
			}
		} else {
			// Exact IP match
			if ip := net.ParseIP(allowed); ip != nil && ip.Equal(clientAddr) {
				return true
			}
		}
	}

	return false
}
```

---

## Frontend Key Masking Pattern

**From CONTEXT.md specification:**

```typescript
function maskKey(key: string): string {
	if (!key) return '';
	return key.slice(0, 12) + '...';
}

function copyToClipboard(text: string): void {
	navigator.clipboard.writeText(text).then(() => {
		message.success('已复制到剪贴板');
	}).catch(() => {
		message.error('复制失败');
	});
}
```

---

## Metadata

**Analog search scope:**
- `internal/models/` - Model definitions
- `internal/services/system/` - Service implementations
- `internal/api/v1/system/` - Handler implementations
- `pkg/middleware/` - Middleware implementations
- `xingran-react-frontend/src/types/` - TypeScript type definitions
- `xingran-react-frontend/src/api/` - API client implementations
- `xingran-react-frontend/src/pages/system/` - React component implementations

**Files scanned:** 18
**Pattern extraction date:** 2026-05-18

---

## Implementation Notes

### Database Migration

The API key feature requires creating two new tables:

```sql
-- API keys table
CREATE TABLE sys_api_keys (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name VARCHAR(100) NOT NULL,
	key VARCHAR(100) UNIQUE NOT NULL,
	user_id UUID REFERENCES sys_user(id),
	expires_at TIMESTAMP,
	last_used_at TIMESTAMP,
	is_active BOOLEAN DEFAULT true,
	scopes JSONB,
	ip_whitelist JSONB,
	description TEXT,
	inherit_perms BOOLEAN DEFAULT false,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	deleted_at TIMESTAMP
);

-- API key usage logs table
CREATE TABLE sys_api_key_usage_logs (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	api_key_id UUID REFERENCES sys_api_keys(id),
	user_id UUID REFERENCES sys_user(id),
	method VARCHAR(10),
	path VARCHAR(500),
	status_code INTEGER,
	client_ip VARCHAR(50),
	user_agent TEXT,
	duration INTEGER,
	success BOOLEAN,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_api_keys_user_id ON sys_api_keys(user_id);
CREATE INDEX idx_api_keys_key ON sys_api_keys(key);
CREATE INDEX idx_api_key_logs_api_key_id ON sys_api_key_usage_logs(api_key_id);
CREATE INDEX idx_api_key_logs_created_at ON sys_api_key_usage_logs(created_at);
```

### Security Considerations

1. **Key Storage:** Keys are stored in plaintext but only shown once during creation
2. **Key Format:** `rec_` prefix + 64 hex chars (32 bytes random)
3. **Key Validation:** Strict format checking in middleware
4. **IP Whitelist:** CIDR notation supported for network ranges
5. **Scope Validation:** Hierarchical scopes (admin > write > read)
6. **Rate Limiting:** Per-key rate limits to prevent abuse
7. **Audit Logging:** All API key usage logged asynchronously

### Performance Considerations

1. **Usage Logging:** Async (goroutine) to avoid blocking requests
2. **Rate Limiting:** In-memory storage with automatic cleanup
3. **Database Indexes:** Indexes on user_id, key, created_at for fast queries
4. **Pagination:** All list endpoints support pagination
5. **Context Caching:** No caching for security (always validate against DB)

---

## Dependency Graph

```
API Key Feature Dependencies:

internal/models/api_key.go
  ├─→ models.BaseModel (base model fields)
  └─→ models.User (foreign key)

internal/models/api_key_usage_log.go
  ├─→ models.BaseModel
  ├─→ models.APIKey (foreign key)
  └─→ models.User (foreign key)

internal/services/system/apikey_service.go
  ├─→ models.APIKey
  ├─→ models.APIKeyUsageLog
  ├─→ gorm.DB (database)
  └─→ services/rate_limiter.go

internal/api/v1/system/apikey_handler.go
  └─→ services/system/apikey_service.go

internal/api/v1/system/apikey_router.go
  ├─→ api/v1/system/apikey_handler.go
  └─→ services/system/apikey_service.go

internal/middleware/apikey.go
  ├─→ services/system/apikey_service.go (ValidateAPIKey)
  └─→ services/rate_limiter.go (Check)

internal/services/rate_limiter.go
  └─→ No dependencies (pure in-memory)

xingran-react-frontend/src/types/apikey.ts
  └─→ types/base.ts (BaseResponse, PageParams)

xingran-react-frontend/src/api/apikey.ts
  ├─→ lib/api.ts (get/post/put/del)
  └─→ types/apikey.ts

xingran-react-frontend/src/pages/system/apikeys/index.tsx
  ├─→ api/apikey.ts
  └─── types/apikey.ts
```

---

*Pattern mapping complete. Ready for planning phase.*
*Phase: 16-api-key-mgt*
*Generated: 2026-05-18*
