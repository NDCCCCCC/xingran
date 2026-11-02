# Fix UpdateRequest ID Field Validation

## Problem

Multiple `UpdateRequest` structures in `internal/models/system/requests/` have incorrectly marked the `ID` field with `binding:"required"` tag, even though the ID is always retrieved from URL path parameters (not from the request body).

This causes 400 "请求参数错误" errors when the frontend sends update requests without including the `id` field in the JSON body.

## Root Cause

The pattern is:
- Handler gets ID from URL: `id := c.Param("id")` then `req.ID = id`
- Frontend sends update data without `id` in request body
- But struct validation requires `id` in JSON body → binding fails

## Solution

Remove `binding:"required"` from the `ID` field in all affected `UpdateRequest` structures and add a clarifying comment.

## Scope

Check and fix all `UpdateRequest` structures in:
- `internal/models/system/requests/*.go`

Already fixed:
- ✅ `UserUpdateRequest` (user_requests.go)

Need to check:
- `RoleUpdateRequest` (role_requests.go)
- `MenuUpdateRequest` (menu_requests.go)
- `DepartmentUpdateRequest` (department_requests.go)
- `PostUpdateRequest` (post_requests.go)
- `DictTypeUpdateRequest` (dict_requests.go)
- `DictDataUpdateRequest` (dict_requests.go)

Correct pattern (from ConfigUpdateRequest):
```go
ID string `json:"id"` // ID 从 URL 参数获取，不在请求体验证
```

## Execution Plan

1. Read all request files to identify affected structures
2. Fix each affected structure (one file at a time)
3. Verify compilation after each fix
4. Test with frontend to confirm no more 400 errors

## Success Criteria

- All `UpdateRequest` structures follow the correct pattern
- `go build ./...` succeeds
- Frontend can update users without 400 errors
