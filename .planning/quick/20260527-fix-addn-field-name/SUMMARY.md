# 修复完成：ADDN 字段名错误

## 状态
completed

## 问题
多个 AD 域服务文件使用了错误的字段名 `user.ADDN`，但 `models.User` 的正确字段是 `AdDn`。

## 字段映射关系
| 层级 | 名称 |
|------|------|
| 数据库列 | `ad_dn` (snake_case) |
| Go 结构体字段 | `AdDn` (camelCase) |
| JSON 字段 | `adDn` |

## 修复的文件
1. `internal/services/addomain/group_management_service.go` - 4处
2. `internal/services/addomain/member_sync_service.go` - 6处（包括匿名结构体）
3. `internal/services/addomain/user_ad_sync_service.go` - 6处
4. `internal/services/system/user_sync_service.go` - 1处

## 验证结果
```bash
go build ./... 2>&1 | grep -i "addn"
# 无输出 = 无 ADDN 相关错误
```

## 注意
当前仍有其他编译错误（SM4Cipher 类型断言问题），与本次修复无关。
