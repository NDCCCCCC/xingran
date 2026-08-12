---
phase: quick
plan: 260520-jrp
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/api/router.go
  - pkg/middleware/request_decryption.go
  - xingran-react-frontend/src/lib/api.ts
autonomous: true
requirements:
  - QUICK-001: Add request encryption toggle config parameter
user_setup: []
must_haves:
  truths:
    - "管理员可以在参数管理页面看到请求加密开关配置"
    - "管理员可以通过修改配置值来动态控制请求加密功能的启停"
    - "修改配置后，中间件会重新读取配置并立即生效"
    - "前端在页面加载时显示当前加密开关状态"
  artifacts:
    - path: "internal/models/config.go"
      provides: "Config model for storing request encryption toggle"
      contains: "config_key='sys.request.encryption.enabled'"
    - path: "pkg/middleware/request_decryption.go"
      provides: "Dynamic config reading for encryption toggle"
      exports: ["getConfigFromDB", "refreshConfig"]
    - path: "xingran-react-frontend/src/pages/system/config/index.tsx"
      provides: "UI for displaying and editing encryption toggle"
      contains: "RequestEncryptionToggle component"
  key_links:
    - from: "sys_config table"
      to: "request_decryption.go"
      via: "database query on each request (cached)"
      pattern: "SELECT config_value FROM sys_config WHERE config_key='sys.request.encryption.enabled'"
    - from: "frontend config page"
      to: "/system/configs/{id}/update"
      via: "POST request to update config"
      pattern: "post('/system/configs/${id}/update', { configValue })"
---

<objective>
Add a request encryption toggle configuration parameter to the system config management page, allowing administrators to dynamically control the request encryption feature through the frontend UI.

Purpose: Enable administrators to enable/disable request encryption without modifying configuration files or restarting the server, improving operational flexibility and debugging capabilities.

Output: A new sys_config entry for request encryption toggle with dynamic config reading in middleware and UI display in config management page.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@internal/config/config.go
@internal/models/config.go
@internal/api/v1/system/config_handler.go
@pkg/middleware/request_decryption.go
@xingran-react-frontend/src/pages/system/config/index.tsx

# Current Architecture

**Config Model:** `internal/models/config.go`
```go
type Config struct {
    BaseModel
    ConfigName  string         `gorm:"size:100;not null" json:"configName"`
    ConfigKey   string         `gorm:"uniqueIndex;size:100;not null" json:"configKey"`
    ConfigValue string         `gorm:"size:500" json:"configValue"`
    ConfigType  ConfigType     `gorm:"default:'Y';size:1" json:"configType"`
    IsSystem    ConfigIsSystem `gorm:"default:0" json:"isSystem"`
    Remark      string         `gorm:"size:500" json:"remark,omitempty"`
}
```

**Current Request Encryption Config:** `internal/config/config.go`
```go
type RequestEncryptionConfig struct {
    Enabled           bool     `mapstructure:"enabled"`
    ExcludePaths      []string `mapstructure:"exclude_paths"`
    RequireEncryption bool     `mapstructure:"require_encryption"`
}
```

**Middleware Reading:** Currently reads from static config file on startup
**Frontend Page:** `xingran-react-frontend/src/pages/system/config/index.tsx` has CRUD for sys_config
</context>

<tasks>

<task type="auto">
  <name>Task 1: Create database migration for request encryption toggle config parameter</name>
  <files>internal/core/db/migrations/migration_request_encryption_toggle.go</files>
  <action>
    Create a new migration file that inserts a new sys_config entry for the request encryption toggle:
    
    **Migration Details:**
    - config_name: "请求加密开关"
    - config_key: "sys.request.encryption.enabled"
    - config_value: "true" (default enabled, matching current config.yaml)
    - config_type: "Y" (system parameter)
    - is_system: 1 (system built-in, cannot be deleted)
    - remark: "控制请求体加密功能的启停（true=启用，false=停用），修改后立即生效"

    **Implementation:**
    1. Create migration file in `internal/core/db/migrations/`
    2. Use GORM to check if config_key exists before insert (idempotent)
    3. Insert the config record with proper defaults
    4. Log success/error messages

    **File naming:** Follow existing migration pattern, e.g., `migration_085_request_encryption_toggle.go`
  </action>
  <verify>
    <automated>Run backend and check database: SELECT * FROM sys_config WHERE config_key='sys.request.encryption.enabled'</automated>
  </verify>
  <done>Database contains the new config parameter with correct values</done>
</task>

<task type="auto">
  <name>Task 2: Implement dynamic config reading in request decryption middleware</name>
  <files>pkg/middleware/request_decryption.go, internal/services/system/config_service.go</files>
  <action>
    Modify the request decryption middleware to read the encryption toggle from sys_config table instead of only static config file.

    **Implementation Steps:**

    1. **Add ConfigService dependency to middleware:**
       - Modify `RequestDecryption()` function signature to accept `*gorm.DB` or a config reader function
       - Create a cached config reader that queries sys_config table

    2. **Add config caching mechanism:**
       - Create a global variable to cache the config value (default: true from config.yaml)
       - Add a `getConfigFromDB()` function that:
         - Checks cache first (TTL: 30 seconds)
         - Queries `SELECT config_value FROM sys_config WHERE config_key='sys.request.encryption.enabled'`
         - Parses "true"/"false" string to bool
         - Returns config value with proper error handling (fallback to static config if DB fails)

    3. **Update middleware logic:**
       - Replace static `config.Enabled` check with `getConfigFromDB()`
       - Ensure fallback to static config.yaml value if DB query fails
       - Add logging when config changes (e.g., "Request encryption toggled to: true")

    4. **Add config refresh endpoint (optional but recommended):**
       - Add cache invalidation when config is updated via `/system/configs/:id/update`
       - This can be done by checking the config_key in the update handler

    **Key Changes:**
    - `pkg/middleware/request_decryption.go`: Add `getConfigFromDB()` with caching
    - `internal/api/v1/system/config_handler.go`: Call config refresh after update
    - Router: Pass DB instance to middleware if needed

    **Error Handling:**
    - If DB query fails, fall back to static config.yaml value (log warning)
    - If invalid value in DB (not "true"/"false"), log error and use default
  </action>
  <verify>
    <automated>
      # Test 1: Check config is read from DB
      curl -X POST http://localhost:9000/system/configs/key/sys.request.encryption.enabled
      
      # Test 2: Update config and verify middleware behavior changes
      curl -X POST http://localhost:9000/system/configs/{config_id}/update -d '{"configValue": "false"}'
      # Then send unencrypted request to protected endpoint - should succeed
      
      # Test 3: Set back to true and verify encryption is enforced
      curl -X POST http://localhost:9000/system/configs/{config_id}/update -d '{"configValue": "true"}'
      # Then send unencrypted request - should be rejected (if require_encryption=true)
    </automated>
  </verify>
  <done>Middleware reads encryption toggle from sys_config table, changes take effect within cache TTL</done>
</task>

<task type="checkpoint:human-verify">
  <name>Task 3: Add UI display and editing capability in frontend config management page</name>
  <files>xingran-react-frontend/src/pages/system/config/index.tsx</files>
  <action>
    Enhance the frontend config management page to prominently display the request encryption toggle parameter.

    **Implementation Steps:**

    1. **Add visual indicator for request encryption toggle:**
       - Add a specialized card or alert component at the top of the config page
       - Display current status with clear visual indicator (green=enabled, red=disabled)
       - Show icon (lock/unlock) and text description

    2. **Add quick toggle button:**
       - Add a "Toggle Encryption" button next to the status display
       - Button should:
         - Show current state ("Enable Encryption" or "Disable Encryption")
         - Use appropriate color (green for enable, red for disable)
         - Include confirmation modal before toggling
         - Call the update API endpoint
         - Refresh the config list after successful update

    3. **Enhance config table display:**
       - Add a special icon or badge in the config table for this parameter
       - Make it easily identifiable (e.g., lock icon in configName column)
       - Add tooltip explaining its purpose

    4. **Add validation and user feedback:**
       - Show confirmation modal: "Are you sure you want to disable request encryption? This may affect security."
       - Display success/error messages with clear descriptions
       - Add info alert explaining the impact: "Changing this toggle will immediately affect all encrypted API endpoints"

    **File Changes:**
    - `xingran-react-frontend/src/pages/system/config/index.tsx`:
      - Add `RequestEncryptionStatus` component at top of page
      - Add quick toggle button with confirmation
      - Enhance table rendering with special handling for this config_key

    **No API changes needed** - existing `/system/configs` endpoints handle the parameter

    **Design Considerations:**
    - Use Ant Design's `<Alert>` component for status display
    - Use `<Popconfirm>` for toggle confirmation
    - Use `<Tag>` with color prop for status badge
    - Follow existing page layout and styling patterns
  </action>
  <verify>
    <automated>npm run build (ensure no TypeScript errors)</automated>
  </verify>
  <done>
    Frontend config page displays request encryption toggle prominently with:
    - Visual status indicator (color-coded badge/icon)
    - Quick toggle button with confirmation
    - Clear explanations of security impact
    - Special identification in config table
  </done>
  <what-built>
    Frontend UI enhancements for request encryption toggle management including status display, quick toggle button, and enhanced table presentation.
  </what-built>
  <how-to-verify>
    1. Start backend and frontend
    2. Navigate to System > Config Management page
    3. Verify you see the request encryption status card at the top
    4. Click the "Toggle Encryption" button
    5. Confirm the action in the modal
    6. Verify the status changes and success message appears
    7. Test the effect by sending an API request with/without encryption
  </how-to-verify>
  <resume-signal>Type "approved" to continue or describe issues</resume-signal>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Frontend → API | Untrusted user input crosses here when updating config |
| API → Database | Config value storage and retrieval |
| Middleware → Database | Config reading on each request |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-QUICK-01 | Tampering | Config update endpoint | mitigate | Validate config_value is only "true" or "false" before accepting update |
| T-QUICK-02 | Information Disclosure | Config query endpoint | accept | Config values are not sensitive (system settings), no mitigation needed |
| T-QUICK-03 | Denial of Service | Cache flooding | accept | Low impact - 30s TTL prevents excessive queries, no mitigation needed |
| T-QUICK-04 | Elevation of Privilege | Config update permissions | mitigate | Existing RBAC in config_handler.go ensures only authorized users can update |
| T-QUICK-05 | Spoofing | Config value injection | mitigate | Strict validation in config_service.go ensures only boolean string values accepted |

**Mitigation Implementation:**
- Add validation in `config_service.go` Update method: check if config_key is "sys.request.encryption.enabled" and validate config_value is "true" or "false"
- Return error if invalid value provided
- Log all config changes for audit trail
</threat_model>

<verification>
**Backend Verification:**
- Migration runs successfully and creates config entry
- Middleware reads config from DB with proper fallback
- Config changes take effect within cache TTL
- Error handling works correctly (DB failures fall back to static config)

**Frontend Verification:**
- Page loads without errors
- Status display shows correct current state
- Toggle button works with confirmation
- Success/error messages display properly
- TypeScript compilation succeeds

**Integration Verification:**
- Toggling config from false → true enables encryption
- Toggling config from true → false disables encryption
- API endpoint behavior changes immediately (within cache TTL)
- No server restart required

**Security Verification:**
- Only authorized users can modify config
- Invalid config values are rejected
- Audit log records all changes
</verification>

<success_criteria>
- ✅ Database migration creates config parameter with correct defaults
- ✅ Middleware reads encryption toggle from sys_config table
- ✅ Config changes take effect within 30 seconds without server restart
- ✅ Frontend displays encryption status prominently on config page
- ✅ Administrators can toggle encryption via UI with one click
- ✅ Proper validation prevents invalid config values
- ✅ Error handling ensures fallback to static config if DB fails
- ✅ No breaking changes to existing functionality
</success_criteria>

<output>
After completion, create `.planning/quick/260520-jrp-request-encryption-config-toggle/260520-jrp-SUMMARY.md`
</output>
