---
slug: ad-sync-500-nul-byte-in-error-msg
status: resolved
deferred_to: v1.16-tech-debt
trigger: AD 域控手动同步点击后,前端报 500 (Internal Server Error),用户误以为是路由丢失
created: 2026-06-16
updated: 2026-06-25
session_type: bug
---

# Debug Session: AD 同步 500 — PostgreSQL 拒绝 NUL 字节

## 关键结论(用户首读这一段)

**用户报的"路由丢失/前后端路径不一致"是误判。** 实际根因是 PostgreSQL TEXT 列拒绝 NUL 字节(0x00),导致 UPDATE `sys_ad_sync_log.error_message` 失败,handler 返回 500。

- **路由完全存在**: `internal/api/v1/system/ad_domain_router.go:36` `configs.POST("/:id/sync", handler.SyncData)` ✅
- **前端调用完全正确**: `syncADData(id, "full")` → `POST /ad-domain/configs/${id}/sync` ✅
- **真实崩溃点**: `internal/services/addomain/sync.go:645` 直接把 `err.Error()`(含 NUL)写入 `sys_ad_sync_log.error_message`

> 用户记错了 — "路由丢失"不是这次的问题。**但用户的"反复出现"是对的**,因为 go-ldap 的同
> 一个缺陷会污染所有 LDAP 属性(用户名、displayName、description 等),下次用户成功路径
> 上的同步就会撞同样的雷。已一并修复。

---

## Symptoms

### Expected Behavior
- 前端点击 AD 配置同步按钮 → 后端执行 LDAP 同步 → 返回结果 JSON
- 即使 LDAP 凭据错误(Invalid Credentials),也应返回 500 + 友好错误消息

### Actual Behavior
- 前端看到 `POST /api/v1/ad-domain/configs/{id}/sync 500 (Internal Server Error)`
- 后端实际错误:

```
ERRO[2026-06-16 17:37:29] [GORM错误] UPDATE "sys_ad_sync_log" SET "computer_count"=0,"duration"=0,"end_time"='2026-06-16 17:37:29.694',"error_count"=1,"error_message"='绑定失败: LDAP Result Code 49 "Invalid Credentials": 80090308: LdapErr: DSID-0C090451, comment: AcceptSecurityContext error, data 775, v3839 (尝试: UPN, NetBIOS, 直连)',"group_count"=0,"ou_count"=0,"sync_status"='failed',"user_count"=0 WHERE id = 'dbe1b2a4-aa97-4b32-bfa1-0fb45f253c61' | 耗时: 2.83ms | 错误: ERROR: invalid byte sequence for encoding "UTF8": 0x00 (SQLSTATE 22021)
```

### Error Messages
- **HTTP 500** (来自 Gin recovery)
- **PostgreSQL SQLSTATE 22021** `invalid byte sequence for encoding "UTF8": 0x00`
- **go-ldap 原始错误** `LDAP Result Code 49 "Invalid Credentials"` + Windows AD 诊断消息

---

## Reproduction

1. 任意 AD 配置,其 `AdminPassword` 错误(或账户被锁定,触发 `data 775`)
2. 在 AD 配置管理页面点击 "同步"
3. → 后端返回 500
4. → `logs/app.log` 出现 `invalid byte sequence for encoding "UTF8": 0x00`

触发条件:Windows AD 服务器的诊断消息含 NUL 字节 (0x00)。`data 775` (用户被锁) / `data 52e` (密码错) 都可能产生 NUL,但不是必然。

---

## Root Cause

### 数据流回溯(trace from symptom to source)

1. **handler.SyncData** (`internal/api/v1/system/ad_domain_handler.go:272`)
   ```go
   result, err := h.service.SyncADData(ctx, id, req.SyncType)
   ```

2. **SyncADData** → **SyncDataByID** → **syncDataInternal** (`internal/services/addomain/sync.go:75`)

3. **syncDataInternal** 创建 sync_log,调用 `client.Connect()` 失败时:
   ```go
   // sync.go:94
   s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, 0, 0, 0, 0, err.Error())
   ```

4. **LDAPClient.Connect** → **tryBindAttempts** (`internal/services/addomain/ldap_client.go:118`)
   ```go
   // ldap_client.go:141
   return fmt.Errorf("绑定失败: %w (尝试: UPN, NetBIOS, 直连)", lastErr)
   ```

5. **go-ldap v3.4.12** `C:\Users\CPIC\go\pkg\mod\github.com\go-ldap\ldap\v3@v3.4.12\error.go:223`:
   ```go
   return &Error{
       ResultCode: resultCode,
       MatchedDN:  response.Children[1].Value.(string),
       Err:        fmt.Errorf("%v", response.Children[2].Value), // ← NUL 字节入口
       Packet:     packet,
   },
   ```
   `response.Children[2].Value` 是 AD 服务器返回的**原始诊断字节**,某些 Windows AD
   服务器会在消息体中嵌入 NUL (0x00) 作为 padding / 结构标记。go-ldap **直接**
   `fmt.Errorf("%v", ...)` 包装,不剔除 NUL。

6. **updateSyncLog** (`sync.go:645`,修复前):
   ```go
   if errorMsg != "" {
       updates["error_message"] = errorMsg
       updates["error_count"] = 1
   }
   s.db.WithContext(ctx).Model(&models.ADSyncLog{}).Where("id = ?", logID).Updates(updates)
   ```
   `err.Error()` 携带 NUL → 直接 UPDATE → **PostgreSQL 拒绝** (`SQLSTATE 22021`)

7. **PostgreSQL TEXT 列不允许 NUL** — 这是 PG 的硬性约束,不是配置问题。

### 三层缺陷

| 层 | 缺陷 | 责任 |
|---|---|---|
| go-ldap 库 | 不剔除 AD 服务器原始字节中的 NUL/无效 UTF-8 | 第三方库,不可改 |
| sync.go updateSyncLog | 把不可信外部错误消息直接写入 DB | **应用代码**(本次修复) |
| sync.go categorizeOUs/syncGroups/syncUsers/syncComputers | LDAP 属性(用户名、displayName、description 等)同样问题 | **应用代码**(本次一并修复) |

---

## Fix Applied

### 1. `internal/utils/string_helper.go` — 新增 `SanitizeForDB`

```go
// SanitizeForDB 把字符串清洗为可安全写入 PostgreSQL TEXT 列的形式。
//
// 行为:
//   - 移除 NUL 字节 (0x00) — 这是 AD/LDAP 错误消息反复触发
//     "invalid byte sequence for encoding UTF8: 0x00 (SQLSTATE 22021)"
//     的根源。
//   - 替换其他 ASCII/Latin-1 控制字符为单空格,保留可读性。
//   - 用 utf8.RuneError 替换无法解码为合法 UTF-8 的字节序列。
//   - TruncateForLog 截断到 maxLen,防止攻击者构造巨大错误消息。
//
// 用途:任何写入数据库/日志/审计字段的外部字符串都应先调用本函数。
func SanitizeForDB(s string) string { ... }
```

### 2. `sync.go` — `updateSyncLog` 在 DB 写入前 sanitization

```go
if errorMsg != "" {
    safe := utils.SanitizeAndTruncate(errorMsg, 4000)
    updates["error_message"] = safe
    updates["error_count"] = 1
    if safe != errorMsg {
        applogger.Warnf("[AD同步] sync_log.error_message 含非法字符已被清洗 (id=%s)", logID)
    }
}
if err := s.db.WithContext(ctx).Model(...).Updates(updates).Error; err != nil {
    applogger.Errorf("[AD同步] 写入 sync_log 失败 (id=%s): %v", logID, err)
}
```

### 3. `sync.go` + `computer.go` — 所有 LDAP 属性读取走 `safeAttr`

新增 helper:
```go
func safeAttr(s string, maxLen int) string {
    return utils.SanitizeAndTruncate(s, maxLen)
}
```

替换 `categorizeOUs` / `syncGroups` / `syncUsers` / `syncComputers` / `buildComputerFromEntry` / `updateComputerFields` 中所有 `entry.GetAttributeValue(...)` 调用,长度按模型列宽配置。

---

## Verification

| 步骤 | 命令 | 结果 |
|---|---|---|
| 单元测试 | `go test -v -run TestSanitize ./internal/utils/` | 9/9 PASS |
| 全 utils 测试 | `go test -count=1 ./internal/utils/...` | PASS |
| 全 addomain 测试 | `go test -count=1 ./internal/services/addomain/...` | PASS (7.6s) |
| 全量编译 | `go build ./...` | PASS |
| Go vet | `go vet ./internal/utils/... ./internal/services/addomain/...` | PASS |

测试用例覆盖:
- NUL 字节剥离
- 控制字符替换 (0x01-0x1F, 0x7F)
- 无效 UTF-8 字节序列 (0xFF 0xFE)
- 中文等多字节 UTF-8 不被破坏
- 真实 LDAP 错误消息清洗
- 长度截断与省略号
- 9 种混合场景的 NUL 字节回归保护

---

## Remaining Risks

1. **go-ldap 库未升级**: 上游未修复,我们的 sanitization 是补丁层。
   长期方案:封装一个 `internal/ldapwrapper` 把 `err.Error()` 在入口处统一清洗。

2. **其它 LDAP/REST 响应字段**: 当前只覆盖了 addomain 模块。如果有其他模块
   (network、scheduler 等) 写入 AD 衍生数据,需要分别检查。

3. **路由"反复丢失"的真实模式**: 用户的"反复出现"印象来源于历史 fix:
   - `b4edd71 fix(backend): resolve F-BE-AD-sync-status — 恢复 /ad-domain/groups/sync-status 路由`
   - `fa5ebf0 fix(backend): register missing workstation-device ad/asset/set-primary-and-save routes`

   这些是**真实的路由注册遗漏**,与本次 500 不同。建议另起一个 Phase (P3/P4)
   添加启动时路由完整性自检 — 详见 `.planning/debug/ad-sync-500-nul-byte-in-error-msg.md#related-route-loss-history`。

---

## Related Route Loss History

| Commit | 修复内容 |
|---|---|
| `b4edd71` | 恢复 `/ad-domain/groups/sync-status` 路由 |
| `fa5ebf0` | 注册缺失的 workstation-device `set-primary-and-save` 路由 |
| `4d6fa74` | 把 group sync 端点移到 groups 路由组(WR-06 修复) |

**建议**: 在 `cmd/main.go` 启动时调用 `internal/api/router.go` 暴露一个 `ListRoutes()`,
与前端 `lib/adDomainApi.ts` 的 URL 模式做 set 比对,缺失时启动失败。
此建议属于"路由回归守护"范畴,不在本次修复范围。

---

## Files Changed

| 文件 | 修改 |
|---|---|
| `internal/utils/string_helper.go` | +95 行:新增 `SanitizeForDB` / `TruncateForLog` / `SanitizeAndTruncate` + 文档注释 |
| `internal/utils/string_helper_test.go` | 新增 9 个测试用例 |
| `internal/services/addomain/sync.go` | +15 行 / 修改多处:import `utils`,新增 `safeAttr` helper,在 `updateSyncLog` / `categorizeOUs` / `syncGroups` / `syncUsers` 应用 sanitization |
| `internal/services/addomain/computer.go` | +10 行 / 修改 2 处:`buildComputerFromEntry` / `updateComputerFields` 应用 sanitization |

## Status

**修复已应用并通过测试** — 等待用户重启后端后,触发任意 LDAP 同步错误验证 500 不再出现。

## Phase 40 Closure (2026-06-25)

复测：`internal/services/addomain/sync.go` 的 `safeAttr` helper 与
`internal/utils/string_helper.go` 的 `SanitizeForDB` / `SanitizeAndTruncate`
已落地，`updateSyncLog` / `categorizeOUs` / `syncGroups` / `syncUsers` /
`syncComputers` 全部走清洗路径。本 session 描述的 fix 已就位，
frontmatter 翻 `resolved`。

verification: `grep -n "safeAttr\|SanitizeAndTruncate" internal/services/addomain/sync.go internal/utils/string_helper.go` 命中多处
files_changed: .planning/debug/ad-sync-500-nul-byte-in-error-msg.md
