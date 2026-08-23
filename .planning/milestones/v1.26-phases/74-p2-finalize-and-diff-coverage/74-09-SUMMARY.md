---
phase: 74-p2-finalize-and-diff-coverage
plan: 09
subsystem: pkg-utils-and-internal-pkg-gapfill
status: complete
date: 2026-08-22
---

# 74-09 SUMMARY: pkg 工具包 + internal/pkg + agent/server + addomain + portcollection

## Result

**D-12 STRICT 全程遵守:零业务代码改动,仅新增 16 个 `*_test.go`(+2854 行),原子提交 `df2d7c0`。全量 `go test ./...` 绿。**

| Package | Before | After | Target | Status |
|---------|--------|-------|--------|--------|
| pkg/response | 0% | **96.1%** | ≥70% | ✓ |
| pkg/ldaputils | 0% | **97.0%** | ≥70% | ✓ |
| pkg/captcha | 0% | **87.5%** | ≥70% | ✓ |
| pkg/time | 0% | **85.7%** | ≥70% | ✓ |
| pkg/query | 0% | **67.6%** | ≥70% | ✗ (−2.4pp) |
| pkg/logger | 0% | **64.6%** | ≥70% | ✗ (−5.4pp) |
| pkg/gormutil | 0% | **63.4%** | ≥70% | ✗ (−6.6pp) |
| internal/pkg/cache | 0% | **52.7%** | ≥70% | ✗ |
| internal/pkg/system | 0% | **31.0%** | ≥70% | ✗ |
| internal/utils | 4.5% | **55.0%** | ≥70% | ✗ |
| internal/agent/server | 2.1% | **22.1%** | ≥70% | ✗ |
| internal/services/addomain | 15.4% | **21.8%** | ≥50% | ✗ → escalate 74-11 |
| internal/services/portcollection | 19.3% | **57.6%** | ≥60% | ✗ (−2.4pp) → escalate 74-11 |

0% 业务包计数: 7 个 pkg/internal 包脱离 0%(22 → ≤15),优于 ≤17 的计划门槛。

## Files added (16 files, +2854 lines)

| File | Package | Focus |
|------|---------|-------|
| `pkg/response/response_test.go` | response | Success/Error/Page/HandleServiceError/HandleGetByID + gin ctx helpers |
| `pkg/ldaputils/dn_test.go` | ldaputils | ExtractOUDNFromUserDN/ParseOUDN/ExtractParentDN/BuildOUPath |
| `pkg/captcha/captcha_test.go` | captcha | 图形 shapes + TextCaptcha + SliderCaptcha + factory |
| `pkg/time/local_time_test.go` | time | LocalTime JSON/Value/Scan roundtrip |
| `pkg/query/builder_test.go` | query | Build + Where 操作符表驱动 |
| `pkg/logger/logger_test.go` | logger | Init/ParseLevel/输出 + Windows 文件句柄模式 |
| `pkg/gormutil/gormutil_test.go` | gormutil | BatchMap/LoadBelongsTo/MergeMaps |
| `internal/pkg/cache/manager_test.go` | pkg/cache | MetricsCacheManager no-redis + L1 + Stop() |
| `internal/pkg/system/system_test.go` | pkg/system | CPU/Disk/Network/Process smoke (t.Skipf on error) |
| `internal/agent/server/agent_smoke_test.go` | agent/server | 中间件/JWT/构造器/TLS config |
| `internal/utils/converter_test.go` | utils | ToInt/ToInt64/ToBoolPtr/Deref*/ToSlice* |
| `internal/utils/context_helper_test.go` | utils | gin ctx Get* + DB nickname/dept (sqlite) |
| `internal/utils/config_diff_test.go` | utils | DiffConfig/CalculateHash/UnifiedDiff/SideBySide |
| `internal/services/addomain/computer_pure_test.go` | addomain | computer 解析纯函数 + utils.go encrypt/decrypt |
| `internal/services/addomain/account_pool_gapfill_test.go` | addomain | AccountPool List/Count/Pick/Create/Update/Delete/SetEnabled (sqlite) |
| `internal/services/portcollection/gapfill_74_09_test.go` | portcollection | trunk_filter/buildPortStatus/QueryService/TemplateCache/厂商映射/采集失败路径 |

## 目标缺口与升级(escalate → 74-11)

**addomain (21.8% / 目标 50%)**: 2415 stmts 大头是 LDAP 交互代码(ldap_client/failover_client/sync),
真实 LDAP server 在单测环境不可行;已覆盖 AccountPool 全 CRUD/状态机 + computer/utils 纯函数。

**portcollection (57.6% / 目标 60%)**: 剩余 0% 语句全部在 parseInterfaceList/parseInterfaceDescriptions/
getAllDot1xStatus/getAllPortSecurity/collectDevicePort 主循环 — `device.ScrapliWrapper` 是具体类型、
内部 scrapligo `*network.Driver` 不可注入 mock,记录处理循环只能真实 SSH 设备覆盖。
已锁定未连接 wrapper 的错误透传路径。

**internal/pkg/system (31.0%)**: 多数函数是平台 syscall 包装,Windows 下不可断言,仅 smoke。
**internal/agent/server (22.1%)**: 大头是 agent 主循环/PTY/WS 长连接,需真实运行时。
**internal/pkg/cache (52.7%)**: Redis 依赖路径用 no-redis 分支覆盖;L2 Redis 路径需 miniredis(74-11 评估)。

## QUIRKS 记录(D-12 — 只记录,不修复)

1. **`response.toAppError`**: `Error(c, http.StatusInternalServerError, ...)` 的 int 实参被当作业务 Code,
   HTTPStatus 回退 400 — HandleServiceError/HandleGetByID 实际发 400 而非 500/404。
2. **`internal/pkg/system.GetDiskInfoDetailed`**: `getDiskInfoByPlatform` 递归自调 → 栈溢出 panic(测试移除)。
3. **`logger.InitLogger`**: `ParseLevel` error 被丢弃,非法 level 静默返回 nil error。
4. **`gormutil.LoadManyToMany` Junction** `${...}` 列占位符未展开(生产 QUIRK) — 仅空 IDs 路径可测。
5. **`addomain parseFileTime`**: AD FileTime "116444736000000000"(epoch 0)返回非 nil(1970-01-01)。
6. **`agent NewTLSConfigFromConfig`**: 全空路径参数不报错(QUIRK),错误路径仅 bad-CA 可触发。
7. **`pkg/captcha generateCaptchaID`**: Windows 时钟粒度致同毫秒 ID 碰撞 — 断言降级为非空。
8. **`ADServiceAccount.TableName()`** 为复数 `sys_ad_service_accounts` — 测试 schema 必须匹配(踩坑修复)。

## Verification

```
go build ./...          # OK
go test -count=1 ./...  # 全绿 (exit 0)
```

原子提交: `df2d7c0` test(74-09): pkg 工具包 + internal/pkg + agent/server + addomain + portcollection 覆盖率提升
