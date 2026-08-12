---
slug: apikey-service-test-types
status: resolved
trigger: 'Cluster 5/5: apikey_service_test.go - Scopes/IPWhitelist 类型不匹配'
created: 2026-06-12
updated: 2026-06-12
---

# Cluster 5: apikey_service_test.go Scopes/IPWhitelist 类型不匹配

## Symptoms

`go test -c -o /dev/null ./internal/services/system/` 报告 5 个错误：
```
:129:16: cannot use string(scopesJSON) (value of type string) as []string
:130:16: cannot use string(ipWhitelistJSON) (value of type string) as []string
:215:15: cannot use string(scopesJSON) (value of type string) as []string
:394:16: cannot use string(scopesJSON1) (value of type string) as []string
:400:16: cannot use string(scopesJSON2) (value of type string) as []string
```

## Initial Hypothesis

测试代码用 `json.Marshal` 把 `[]string` 序列化为 string 后赋给 `Scopes` 字段。
但 `models.APIKey.Scopes` 和 `IPWhitelist` 现在的类型是 `[]string`（带 `gorm:"type:jsonb;serializer:json"` 标签，GORM 自动 JSON 序列化），不再需要手动 marshal。

## 实际模型定义 (internal/models/api_key.go:16-17)
```go
Scopes       []string   `gorm:"type:jsonb;serializer:json" json:"scopes"`
IPWhitelist  []string   `gorm:"type:jsonb;serializer:json" json:"ipWhitelist"`
```

## 修复策略

5 处都是简单机械替换：
- `Scopes: string(scopesJSON)` → `Scopes: []string{"read", "write"}`（按原值）
- `IPWhitelist: string(ipWhitelistJSON)` → `IPWhitelist: []string{}`（或 nil）
- `key1.Scopes = string(scopesJSON1)` → `key1.Scopes = []string{"read"}`
- `key2.Scopes = string(scopesJSON2)` → `key2.Scopes = []string{"write"}`

同时清理：
- 删 `scopesJSON, _ := json.Marshal(...)` 等不再使用的 marshal 调用（避免 unused 警告）
- 删 `"encoding/json"` import 如所有 marshal 都被移除

## Current Focus

- **hypothesis:** 直接传 `[]string` 字面量即可。GORM 用 `serializer:json` tag 自动序列化。
- **next_action:** 5 处替换 + 清理 dead 变量 + 检查 import 必要性。
- **test:** `go test -c -o /dev/null ./internal/services/system/` 退出码 0。
- **expecting:** 5 个错误全部消失；GORM 写库时自动把 `[]string` 转为 JSONB。
- **blind_spots:** ① user_sync_service_test.go 也有 7+ 个错误（同包，但不属本 cluster 范围）→ 记录 Side findings；② 测试可能还有未捕获的 json.Marshal 在其它地方 — grep 全文确认。
- **tdd_checkpoint:** 不动测试语义。

## Side findings (用户原 cluster 5 列表之外)
- `internal/services/system/user_sync_service_test.go` 同包 7+ 错误:
  - `NewUserSyncService` 缺参（需要 PasswordManager, *addomain.DeptOUmapper）
  - `undefined: addomain.ADUser`
  - `assignment mismatch: 1 variable but service.SyncUserFromAD returns 2 values`
- 不属本 cluster 范围，**不修**，待用户决定是否单独处理。

## Evidence

- 2026-06-12 grep `json\.` in apikey_service_test.go: 命中 5 行 (122, 123, 210, 392, 398)，全部为待修 marshal 调用。
- 2026-06-12 修复后 `go test -c -o /dev/null ./internal/services/system/`: 0 个 apikey_service_test.go 错误。
- 2026-06-12 同包 user_sync_service_test.go 仍报错（10+ errors, out of scope）。

## Eliminated

- (无 — 假设在第一次验证即成立)

## Resolution

- root_cause: 测试代码在 `models.APIKey.Scopes`/`IPWhitelist` 字段由 `string`（旧定义）改为 `[]string`（`gorm:"type:jsonb;serializer:json"`）后，未同步更新；仍用 `string(json.Marshal(...))` 错误赋值为 `[]string` 字段。
- fix: 删除全部 5 处 `json.Marshal` 调用及 `encoding/json` import，改为 `[]string` 字面量（GORM 的 `serializer:json` 自动序列化）。
- verification: `go test -c -o /dev/null ./internal/services/system/` 过滤后 apikey_service_test.go 0 错误；残留错误全部在 user_sync_service_test.go（out of scope）。
- files_changed:
  - `internal/services/system/apikey_service_test.go`
    - 删除 `"encoding/json"` import (line 5)
    - 删除 `scopesJSON, _ := json.Marshal(...)` (line 122)
    - 删除 `ipWhitelistJSON, _ := json.Marshal(...)` (line 123)
    - 替换 `Scopes: string(scopesJSON)` → `Scopes: []string{"read", "write"}` (line 129)
    - 替换 `IPWhitelist: string(ipWhitelistJSON)` → `IPWhitelist: []string{}` (line 130)
    - 删除循环内 `scopesJSON, _ := json.Marshal([]string{"read"})` (line 210)
    - 替换 `Scopes: string(scopesJSON)` → `Scopes: []string{"read"}` (line 215)
    - 删除 `scopesJSON1, _ := json.Marshal([]string{"read"})` (line 392)
    - 替换 `key1.Scopes = string(scopesJSON1)` → `key1.Scopes = []string{"read"}` (line 394)
    - 删除 `scopesJSON2, _ := json.Marshal([]string{"write"})` (line 398)
    - 替换 `key2.Scopes = string(scopesJSON2)` → `key2.Scopes = []string{"write"}` (line 400)
