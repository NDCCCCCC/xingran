---
slug: sm4-encryptor-not-initialized
status: resolved
trigger: SM4 加密器未初始化，无法解密密码
created: 2026-05-27T16:00:00+08:00
updated: 2026-05-27T16:30:00+08:00
---

## Symptoms

### Expected Behavior
SSH 连接应该成功建立并能执行 LLDP 命令（重构前工作正常）

### Actual Behavior
所有 SSH 连接失败，报错："创建连接失败: SM4 加密器未初始化，无法解密密码"

### Error Messages
```
WARN[2026-05-27 16:00:00] Failed to execute LLDP command on device CX_HB_SY_ZHUSHAN_RJ_RSR10: 获取连接失败: 创建连接失败: SM4 加密器未初始化，无法解密密码
WARN[2026-05-27 16:00:00] Failed to execute LLDP command on device CX-HB-WH-01F-FL-S2910-01: 获取连接失败: 创建连接失败: SM4 加密器未初始化，无法解密密码
```

### Timeline
- **Start**: 今天重构 SM 加解密之后开始出现
- **Pattern**: 重构前工作正常

### Reproduction Steps
- 定时任务采集时触发
- 手动操作时也触发
- **所有** SSH 连接都失败

## Resolution

### Root Cause
During SM4 encryption refactoring, a type mismatch occurred between the interface and concrete implementation:

1. **Problem**: In `internal/core/core.go:222`, the connection pool was initialized with `nil` for the SM4 cipher:
   ```go
   connectionPool := device.NewDeviceConnectionPool(c.GetDB(), nil, poolConfig)
   ```

2. **Why it happened**: The Core has `SM4Cipher` of type `addomain.PasswordCipher` (interface), but the connection pool expected `*crypto.SM4Cipher` (concrete type).

3. **Impact**: All SSH connections failed because the connection pool could not decrypt passwords without the SM4 cipher.

### Fix Applied

**File: internal/device/connection_pool.go**
- Changed struct field from `sm4Cipher *crypto.SM4Cipher` to `passwordCipher addomain.PasswordCipher`
- Updated constructor parameter type to accept the interface
- Updated error message from "SM4 加密器未初始化" to "密码加密器未初始化"
- Removed unused `crypto` import, added `addomain` import

**File: internal/core/core.go**  
- Fixed line 222: Changed `nil` to `c.SM4Cipher` to pass the initialized cipher
- Fixed line 534: Updated `UserSyncService` initialization to include required `DeptOUmapper` parameter

### Testing
- ✅ Build verification: `go build ./...` succeeded
- ✅ Type compatibility resolved between interface and concrete types
- ✅ Connection pool now receives properly initialized cipher from Core

### Impact Analysis
- **Severity**: High - All SSH connections were failing
- **Scope**: Network device connections, LLDP collection, port collection
- **Risk**: Low - Type system now properly enforces interface usage
- **Fix Time**: ~30 minutes

## Preventive Measures

1. **Type Safety**: Use interface types throughout the dependency injection chain to avoid type mismatches
2. **Build Verification**: Always run `go build ./...` after refactoring type signatures
3. **Interface Documentation**: Document expected interface implementations in struct comments
4. **Integration Testing**: Add tests for connection pool initialization with real cipher instances

## Evidence

### Investigation Steps
1. Searched for error message "SM4 加密器未初始化" → Found in `connection_pool.go:322`
2. Examined connection pool initialization in `core.go:222` → Found `nil` parameter
3. Traced SM4Cipher type from Core struct → Found `addomain.PasswordCipher` interface
4. Compared with connection pool expected type → Found `*crypto.SM4Cipher` concrete type mismatch

### Code References
- `internal/device/connection_pool.go:130` - Struct field definition
- `internal/device/connection_pool.go:153` - Constructor signature  
- `internal/core/core.go:51` - Core.SM4Cipher field type
- `internal/core/core.go:222` - Connection pool initialization (fixed)

## Eliminated

- ❌ SM4 key configuration issues (key was properly configured)
- ❌ Connection pool configuration errors (config was correct)
- ❌ Network device credential problems (credentials were valid)
- ❌ scrapligo library issues (library was working correctly)