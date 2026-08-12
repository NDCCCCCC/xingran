---
slug: vdi-vm-list-nil-pointer-panic
status: resolved
trigger: VDI VM list endpoint panic with nil pointer dereference in callAPI
created: 2026-05-26
updated: 2026-05-26
---

## Symptoms

**Expected behavior:**
访问 `/api/v1/vdi/vm/list` 端点应该返回虚拟机列表数据

**Actual behavior:**
服务器 panic，前端收到空响应，控制台显示 "响应格式无效"

**Error messages:**
```
runtime error: invalid memory address or nil pointer dereference
goroutine 184 [running]:
bytes.(*Buffer).Len(...)
        C:/Program Files/Go/src/bytes/buffer.go:79
net/http.NewRequestWithContext({0x7ff6ba253550, 0xc001d22d20}, {0x7ff6b9a0cd89?, 0x2?}, {0xc002015050?, 0x2a?}, {0x7ff6ba243d60, 0x0})
        C:/Program Files/Go/src/net/http/request.go:926 +0x564
github.com/xingran-next/xingran-go-backend/internal/services/vdi.(*vdiClientExtendedImpl).callAPI(0xc002958fc0, {0x7ff6ba253550, 0xc001d22d20}, {0xc0001c2960, 0x1c6}, {0x7ff6b9a0cd89, 0x3}, {0x7ff6b9a3cd53, 0x13}, {0x0, ...}, ...)
        D:/CODE/ClaudeCode/xingran-go-backend/internal/services/vdi/vdi_client_extended.go:344 +0x22e
```

**Timeline:**
- 问题发生在 2026-05-26 10:58:34
- 似乎是同步 VDI 资源时触发

**Reproduction:**
1. 用户登录系统
2. 访问 VDI 虚拟机列表页面
3. 前端调用 `POST /api/v1/vdi/vm/list`
4. 后端尝试从 VDI 系统同步虚拟机数据
5. 在调用 `ListResourceGroups` 时 panic

**Stack trace key frames:**
- `VMHandler.List()` (vm_handler.go:71)
- `vmServiceImpl.ListVMs()` (vm_service_impl.go:253)
- `syncVMsFromVDI()` (vm_service_impl.go:65)
- `vdiClientExtendedImpl.ListResourceGroups()` (vdi_client_extended.go:293)
- `callAPI()` (vdi_client_extended.go:344) ← PANIC
- `net/http.NewRequestWithContext()` ← nil pointer dereference

## Current Focus

**hypothesis:** callAPI 方法在构建 HTTP 请求时传递了 nil 的 body buffer，导致 NewRequestWithContext 内部调用 buffer.Len() 时发生空指针解引用

**test:** 检查 vdi_client_extended.go:344 的 callAPI 方法中 body 参数的构造逻辑

**expecting:** 找到传递给 NewRequestWithContext 的 io.Reader 为 nil 的原因

**next_action:** 读取 vdi_client_extended.go 文件，分析 callAPI 方法的实现

**reasoning_checkpoint:**

**tdd_checkpoint:**

## Evidence

- **Root cause identified:** In `callAPI` method (line 335-342), when `body` parameter is `nil`, the code declares `var reqBody *bytes.Buffer` but leaves it as `nil`. This nil pointer is then passed to `http.NewRequestWithContext()`, which expects an `io.Reader`. The Go HTTP library attempts to call methods on the nil reader, causing the panic.
- **Pattern analysis:** All GET requests (ListResourceGroups, ListVMs, GetUserVMs, GetAvailableUsers, GetVM) pass `nil` as the body parameter, making them all susceptible to this panic.
- **Code path:** `VMHandler.List` → `vmServiceImpl.ListVMs` → `syncVMsFromVDI` → `client.ListResourceGroups` → `callAPI` → panic

## Eliminated

- ❌ Not a VDI server configuration issue (server lookup succeeds)
- ❌ Not an authentication issue (token is obtained successfully)
- ❌ Not a network connectivity issue (panic occurs before request is sent)
- ❌ Not a data parsing issue (panic occurs in request creation, not response handling)

## Resolution

**root_cause:** Type mismatch in `callAPI` method - declared `var reqBody *bytes.Buffer` but should use `io.Reader` interface. When `body` is `nil`, `reqBody` remains nil, and passing nil `*bytes.Buffer` to `http.NewRequestWithContext()` causes internal nil pointer dereference in Go's HTTP library.

**fix:** Changed `var reqBody *bytes.Buffer` to `var reqBody io.Reader` in `callAPI` method (line 335). This allows nil to be passed correctly to `NewRequestWithContext`, as `io.Reader` is the expected parameter type and nil is a valid value for interface types.

**verification:**
- Code compiles successfully: `go build ./internal/services/vdi/...`
- The fix ensures GET requests (which pass nil body) no longer panic
- POST/PUT requests still work correctly as non-nil bodies are converted to `bytes.Buffer` as before
- No behavioral change for non-nil body cases

**files_changed:**
- `internal/services/vdi/vdi_client_extended.go` (line 335: `*bytes.Buffer` → `io.Reader`)

**technical_note:** The Go `http.NewRequestWithContext` function signature expects `io.Reader` for the body parameter. While a `*bytes.Buffer` implements `io.Reader`, a nil `*bytes.Buffer` is not the same as a nil `io.Reader`. The former causes a nil pointer dereference when the HTTP library tries to call `Len()` method on it, while the latter is handled correctly as "no body".
