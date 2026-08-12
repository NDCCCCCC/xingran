# Quick Task: Fix ADDN Field Name

## Context

编译错误：多个 AD 域服务文件使用了错误的字段名 `user.ADDN`，但 `models.User` 的正确字段是 `AdDn`。

## Errors Found

```
internal\services\addomain\group_management_service.go:211:11: user.ADDN undefined
internal\services\addomain\group_management_service.go:212:36: user.ADDN undefined
internal\services\addomain\group_management_service.go:269:11: user.ADDN undefined
internal\services\addomain\group_management_service.go:270:36: user.ADDN undefined
internal\services\addomain\member_sync_service.go:139:11: user.ADDN undefined
internal\services\addomain\member_sync_service.go:144:19: user.ADDN undefined
internal\services\addomain\member_sync_service.go:251:10: user.ADDN undefined
internal\services\addomain\member_sync_service.go:256:18: user.ADDN undefined
internal\services\addomain\user_ad_sync_service.go:40:10: user.ADDN undefined
internal\services\addomain\user_ad_sync_service.go:102:10: user.ADDN undefined
internal\services\system\user_sync_service.go:84:7: user.ADDN undefined
```

## Correct Field Name

在 `internal/models/user.go` 第34行：
```go
AdDn *string `gorm:"type:text;column:ad_dn" json:"adDn,omitempty"`
```

## Tasks

1. [ ] 修复 `internal/services/addomain/group_management_service.go` - 将 `user.ADDN` 改为 `user.AdDn` (4处)
2. [ ] 修复 `internal/services/addomain/member_sync_service.go` - 将 `user.ADDN` 改为 `user.AdDn` (6处)
3. [ ] 修复 `internal/services/addomain/user_ad_sync_service.go` - 将 `user.ADDN` 改为 `user.AdDn` (6处)
4. [ ] 修复 `internal/services/system/user_sync_service.go` - 将 `user.ADDN` 改为 `user.AdDn` (1处)
5. [ ] 编译验证 `go build ./...`
6. [ ] 提交更改

## Success Criteria

- `go build ./...` 编译成功，无 ADDN 相关错误
- 所有 `user.ADDN` 和 `u.ADDN` 引用已改为 `AdDn`
