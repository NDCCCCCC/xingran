---
phase: quick
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx
  - xingran-react-frontend/src/lib/vdiApi.ts
  - xingran-react-frontend/src/types/vdi.ts
autonomous: true
requirements:
  - VM bind user dropdown with system user search
  - Display nickname in dropdown, save username to database
must_haves:
  truths:
    - "Bind user modal shows a searchable dropdown of system users"
    - "Dropdown displays user nickname, selection sends username to backend"
    - "User can search by nickname or username to filter results"
  artifacts:
    - path: "xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx"
      provides: "Bind user modal with Select component using system user API"
      contains: "Select"
    - path: "xingran-react-frontend/src/lib/vdiApi.ts"
      provides: "BindUserRequest type updated to use username field"
      contains: "BindUserRequest"
  key_links:
    - from: "xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx"
      to: "POST /system/users/list"
      via: "fetch in bind user modal for user dropdown"
      pattern: "post.*system/users/list"
    - from: "xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx"
      to: "vmApi.bindUser"
      via: "form submit with username"
      pattern: "vmApi.bindUser"
---

<objective>
Replace the plain text input for "Bind User" in the VM list page with an Ant Design Select dropdown that supports fuzzy search. The dropdown loads system users, displays their nickname, and sends their username to the backend BindUser API.

Purpose: Users currently must type a raw user ID to bind a VM. This is error-prone and unfriendly. A searchable dropdown with human-readable names makes the operation intuitive.
Output: Updated bind user modal in VirtualMachineList component.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md

<interfaces>
<!-- Key types and contracts the executor needs. Extracted from codebase. -->

From src/types/vdi.ts:
```typescript
export interface BindUserRequest {
  user_id: string;
}
```

From src/lib/api.ts:
```typescript
export async function post<T>(url: string, data?: any): Promise<ApiResponse<T>>
```

System User model (from backend models/user.go):
```go
type User struct {
    BaseModel                    // includes id (UUID)
    Username string              `json:"username"`
    Nickname *string             `json:"nickname,omitempty"`
    Status   UserStatus          `json:"status"`   // 0=enabled
}
```

Backend User List API: POST /system/users/list
- Request body: `{ current: number, pageSize: number, username?: string, nickname?: string }`
- Response: `{ code: 0, data: { list: User[], total: number } }`

Backend BindUser API: POST /vdi/vms/:id/bind_user
- Current request: `{ user_id: string }` where user_id is sent to VDI API
- The `user_id` field in BindUserServiceRequest is passed directly to `client.BindUser(ctx, vm.VMID, req.UserID)` which sends it as `UserID` to the VDI API
- The backend then updates `bound_user_id` and `bound_user_name` in the local VM record

IMPORTANT: The current backend `BindUserServiceRequest.UserID` is passed to the VDI API as-is. The requirement is to change this to use `username` (the login account name) instead of the database UUID. The VDI API's `user_id` field receives the system user's `username` value.

From backend vm_service_impl.go BindUser:
```go
updates := map[string]interface{}{
    "bound_user_id":   &req.UserID,
    "bound_user_name": vdiVMDetail.BoundUserName,
}
```

From backend vdi_client_extended.go BindUser:
```go
req := struct {
    RcID   int    `json:"rcid"`
    VmID   int    `json:"vmid"`
    Type   string `json:"type"`
    UserID string `json:"user_id"`
}{
    Type:   "1",
    UserID: userID,    // This is what gets sent to VDI API
    VmID:   vmidInt,
}
```

The requirement: "前端显示nickname，数据库保存username" means:
- Frontend dropdown shows: user nickname
- Frontend sends to backend: username (not UUID, not nickname)
- Backend saves username in bound_user_id field
- Backend sends username to VDI API
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Replace bind user modal Input with searchable Select dropdown</name>
  <files>
    xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx,
    xingran-react-frontend/src/lib/vdiApi.ts
  </files>
  <action>
In `index.tsx`, modify the "Bind User" modal and related state/handlers:

1. Add new state variables after existing state declarations (around line 54):
   ```typescript
   const [systemUsers, setSystemUsers] = useState<Array<{id: string; username: string; nickname?: string}>>([]);
   const [userSearchLoading, setUserSearchLoading] = useState(false);
   ```

2. Create a `loadSystemUsers` function that calls `POST /system/users/list` with a search keyword. Use the existing `post` import from `@/lib/api`. The function should:
   - Accept an optional `search` string parameter
   - Set `userSearchLoading` to true before fetching
   - Call `post('/system/users/list', { current: 1, pageSize: 50, username: search, nickname: search })` (pass search to both username and nickname fields so user can search by either)
   - On success, set `systemUsers` to the response list, filtering out users with status !== 0
   - On error, set `systemUsers` to empty array
   - Always set `userSearchLoading` to false in finally block

3. Modify the `handleBindUser` function (around line 558):
   - Change the form field being read from `user_id` to `username`
   - Update the `vmApi.bindUser` call to pass `{ username: values.username }` instead of `{ user_id: values.user_id }`

4. Replace the bind user modal content (around lines 1196-1209). Replace the `<Form.Item>` with `name="user_id"` with a Select component:
   ```tsx
   <Form.Item label="用户" name="username" rules={[{ required: true, message: '请选择用户' }]}>
     <Select
       showSearch
       placeholder="请输入用户名或昵称搜索"
       filterOption={false}
       loading={userSearchLoading}
       onSearch={(value) => {
         loadSystemUsers(value);
       }}
       onFocus={() => {
         if (systemUsers.length === 0) {
           loadSystemUsers('');
         }
       }}
       notFoundContent={userSearchLoading ? '搜索中...' : '未找到用户'}
     >
       {systemUsers.map(user => (
         <Select.Option key={user.id} value={user.username}>
           {user.nickname || user.username}
         </Select.Option>
       ))}
     </Select>
   </Form.Item>
   ```

5. In `vdiApi.ts`, update the `BindUserRequest` type:
   ```typescript
   export interface BindUserRequest {
     username: string;
   }
   ```
   And update the `bindUser` method call site to pass `{ username }` instead of `{ user_id }`.

6. In `index.tsx`, add `useRef` for debounce timer for user search:
   ```typescript
   const userSearchTimer = useRef<ReturnType<typeof setTimeout>>();
   ```
   In the Select's `onSearch`, debounce the API call with 300ms:
   ```typescript
   onSearch={(value) => {
     if (userSearchTimer.current) clearTimeout(userSearchTimer.current);
     userSearchTimer.current = setTimeout(() => loadSystemUsers(value), 300);
   }}
   ```
   This prevents excessive API calls during typing.

7. Ensure the form resets when modal closes. The existing `setBindUserModalVisible(false)` is followed by `form.resetFields()` -- verify this works since the same `form` instance is used for both create and bind modals. The bind modal should use its own form instance to avoid conflicts. Add:
   ```typescript
   const [bindUserForm] = Form.useForm();
   ```
   And use `bindUserForm` in the bind user modal instead of `form`. Update `handleBindUser` to use `bindUserForm.validateFields()` and `bindUserForm.resetFields()`.
  </action>
  <verify>
    <automated>cd xingran-react-frontend && npx tsc --noEmit --pretty 2>&1 | head -30</automated>
  </verify>
  <done>
    Bind user modal displays a Select dropdown with system users.
    Dropdown shows user nickname (falls back to username if no nickname).
    Selecting a user and clicking OK calls vmApi.bindUser with { username: selectedUsername }.
    User can type to search by username or nickname with 300ms debounce.
    TypeScript compiles without errors.
  </done>
</task>

<task type="auto">
  <name>Task 2: Update backend BindUser to accept username and resolve to user</name>
  <files>
    internal/services/vdi/vm_service.go,
    internal/services/vdi/vm_service_impl.go
  </files>
  <action>
Update the backend BindUser flow so it accepts a `username` (system login name) instead of a raw `user_id`:

1. In `internal/services/vdi/vm_service.go`, update `BindUserServiceRequest`:
   ```go
   type BindUserServiceRequest struct {
       Username string `json:"username" validate:"required"`
   }
   ```
   Change `UserID` field to `Username`.

2. In `internal/services/vdi/vm_service_impl.go`, update the `BindUser` method:

   Before calling the VDI API, resolve the username to look up the system user and build the display name. The current code:
   ```go
   vdiVMDetail, err := client.BindUser(ctx, vm.VMID, req.UserID)
   ```
   Change to:
   ```go
   // Look up system user to build display name
   var systemUser models.User
   if err := s.db.WithContext(ctx).Where("username = ? AND deleted_at IS NULL", req.Username).First(&systemUser).Error; err != nil {
       return fmt.Errorf("system user not found: %s: %w", req.Username, err)
   }

   // Build display name: nickname (username)
   displayName := req.Username
   if systemUser.Nickname != nil && *systemUser.Nickname != "" {
       displayName = fmt.Sprintf("%s (%s)", *systemUser.Nickname, req.Username)
   }

   // Call VDI API with username
   vdiVMDetail, err := client.BindUser(ctx, vm.VMID, req.Username)
   ```

   Then update the local record updates:
   ```go
   updates := map[string]interface{}{
       "bound_user_id":   req.Username,    // Store username, not UUID
       "bound_user_name": displayName,     // Store "nickname (username)" for display
   }
   ```
   Note: If `vdiVMDetail.BoundUserName` has a value from VDI API, prefer that for `bound_user_name`. Otherwise use our computed `displayName`:
   ```go
   boundUserName := displayName
   if vdiVMDetail.BoundUserName != "" {
       boundUserName = vdiVMDetail.BoundUserName
   }
   updates := map[string]interface{}{
       "bound_user_id":   req.Username,
       "bound_user_name": boundUserName,
   }
   ```
  </action>
  <verify>
    <automated>cd D:/code/ClaudeCode/xingran-go-backend && go build ./internal/services/vdi/... 2>&1</automated>
  </verify>
  <done>
    BindUserServiceRequest accepts `username` field instead of `user_id`.
    BindUser method looks up system user by username to build display name.
    Local VM record stores `bound_user_id` = username, `bound_user_name` = "nickname (username)" or VDI returned name.
    Go compiles without errors in the vdi service package.
  </done>
</task>

</tasks>

<verification>
1. Frontend TypeScript compiles: `cd xingran-react-frontend && npx tsc --noEmit`
2. Backend Go compiles: `go build ./...`
3. Bind user modal shows Select dropdown instead of plain Input
4. Select dropdown loads system users from /system/users/list
5. Selecting a user sends username to /vdi/vms/:id/bind_user
</verification>

<success_criteria>
- Bind user modal uses Ant Design Select with showSearch
- Dropdown displays user nickname (fallback to username)
- Searching triggers debounced API call to /system/users/list
- Selected user's username is sent to backend bind_user endpoint
- Backend resolves username, calls VDI API with username
- bound_user_id stores username, bound_user_name stores display name
- TypeScript and Go both compile cleanly
</success_criteria>

<output>
After completion, create `.planning/quick/260604-cqo-nickname-username/260604-cqo-SUMMARY.md`
</output>
