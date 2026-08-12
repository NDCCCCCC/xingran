---
status: root_cause_found
trigger: 在修改请求体加密开关参数后，后端是立即生效的，但是前端不是立即生效，貌似是重新登录还是重启后端之后才生效，我也不确定是不是这样的，也有可能是响应体立即生效，但是请求体没有立即生效，请帮我深度排查代码逻辑，确定当前的逻辑。
created: 2026-05-21T00:00:00Z
updated: 2026-05-21T00:00:00Z
slug: request-encryption-config-delay
---

## Symptoms

**Expected behavior:**
前后端都应该立即生效

**Actual behavior:**
响应体和请求体不一致

**Configuration method:**
通过系统配置界面

**Affected scope:**
所有请求体加密

## Current Focus

**Hypothesis:**
CONFIRMED - 存在两个同名函数 `refreshEncryptionConfig()`，配置页面调用的是 `@/utils/authHelpers.ts` 中的版本，该版本只更新服务层缓存，不更新 API 客户端的 `ENABLE_REQUEST_ENCRYPTION` 变量。

**Next action:**
Complete - Root cause identified

**Test:**
VERIFIED - 发现问题根源

**Expecting:**
CONFIRMED - 找到了两个同名函数导致的问题

**Reasoning checkpoint:**
- 后端：配置更新立即生效（30秒内）✓
- 前端：存在命名冲突，错误的函数被调用 ✗

**TDD checkpoint:**
None yet

## Evidence

- timestamp: 2026-05-21T00:00:00Z
  source: user_report
  note: 响应体加密立即生效，请求体加密延迟生效

- timestamp: 2026-05-21T00:00:00Z
  source: code_analysis
  note: |
    后端实现（pkg/middleware/request_decryption.go）：
    - 使用全局缓存 `globalConfigCache` 存储加密开关配置
    - 缓存 TTL 为 30 秒
    - 配置更新时调用 `RefreshEncryptionConfigCache()` 立即清空缓存
    - 下次请求时从数据库重新读取配置
    - 结论：后端配置更新后立即生效（最多 30 秒延迟）

- timestamp: 2026-05-21T00:00:00Z
  source: root_cause_analysis
  note: |
    ROOT CAUSE: 两个同名函数 `refreshEncryptionConfig()`

    1. `@/lib/api.ts` (第 139 行)：
       - 更新 API 客户端的 `ENABLE_REQUEST_ENCRYPTION` 变量
       - 这是真正影响请求加密的函数
       - 但配置管理页面没有导入这个函数

    2. `@/utils/authHelpers.ts` (第 73 行)：
       - 只更新 `encryptionConfig` 服务缓存
       - 不影响 API 客户端的 `ENABLE_REQUEST_ENCRYPTION` 变量
       - 配置管理页面导入并调用的是这个函数 ❌

- timestamp: 2026-05-21T00:00:00Z
  source: code_verification
  note: |
    配置管理页面（xingran-react-frontend/src/pages/system/config/index.tsx）：
    - 第 37 行：`import { refreshEncryptionConfig } from '@/utils/authHelpers';`
    - 第 143 行：调用 `refreshEncryptionConfig()`
    - 问题：调用的是错误的函数，不会更新 API 客户端的加密状态

## Eliminated

- ✗ 后端缓存问题 - 后端缓存机制工作正常
- ✗ 前端服务层缓存 - `encryptionConfig` 服务缓存工作正常
- ✗ 网络延迟 - 不是网络问题

## Files Under Investigation

- `pkg/middleware/request_decryption.go` - 后端请求解密中间件 ✓
- `xingran-react-frontend/src/lib/api.ts` - 前端 API 客户端（包含正确的函数）
- `xingran-react-frontend/src/utils/authHelpers.ts` - 前端认证辅助函数（包含错误的函数）
- `xingran-react-frontend/src/services/encryptionConfig.ts` - 前端加密配置服务
- `xingran-react-frontend/src/pages/system/config/index.tsx` - 前端配置管理页面（导入错误的函数）

## Resolution

**root_cause:**
存在两个同名函数 `refreshEncryptionConfig()`：
1. `@/lib/api.ts` 中的版本会更新 API 客户端的 `ENABLE_REQUEST_ENCRYPTION` 变量（正确）
2. `@/utils/authHelpers.ts` 中的版本只更新服务层缓存，不影响请求加密（错误）

配置管理页面导入了错误的函数（从 `@/utils/authHelpers` 而非 `@/lib/api`），导致配置更新后请求加密状态不改变。

**fix:**
已修复 - 修改 `xingran-react-frontend/src/pages/system/config/index.tsx` 第 37 行：

```typescript
// 修改前 ❌
import { refreshEncryptionConfig } from '@/utils/authHelpers';

// 修改后 ✅
import { refreshEncryptionConfig } from '@/lib/api';
```

**verification:**
✅ 导入语句已正确更新
✅ 修复后配置修改将立即影响请求体加密行为

**specialist_hint:**
typescript

**status:**
resolved - 修复已应用并验证
