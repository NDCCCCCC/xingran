---
quick_id: 260812-wu5
status: partial
date: 2026-08-12
---

# Quick Task 260812-wu5: 清理 internal/constants 死代码 + 统一分页常量(方案 B)

## 已完成
- **死代码清理**:删除 10 个生产代码 0 引用的常量(cache.go 的 RedisKeyPrefix/LoginFailure/CaptchaStorage/CaptchaLock/CaptchaBgCache/CaptchaBgCounter/UserOnline 共 7 个;time.go 的 ADSyncTimeout;pagination.go 的 MinAllowedPageSize/LDAPDefaultPageSize/LDAPMinPageSize)。
- **风格统一**:`/** */`→`//`;新增 `internal/constants/doc.go`(package 注释);删 time.go 悬空注释;example_test 清理 status.go 历史噪音 + 修 ExampleOneDay 误导。
- **UUIDPattern 重命名**:`UuidPattern`→`UUIDPattern`(符合 Go 缩写规范),含定义 + 8 处引用 + test 注释。
- **分页统一(方案 B)**:拆 `MaxListPageSize=100`(表格防 DoS)/ `MaxOptionsPageSize=10000`(下拉全集),消除 utils/operations/asset/system 四处 `MaxPageSize` 同名分叉;所有引用统一到 `constants`。
- **过时测试**:删 `TestPageSizeConstants`;修正 `TestClampPageSizeMath` 与 `TestExtractPagination` 以反映 operations 上限 100→10000 的新语义。

## 唯一行为变化
- `utils.ParsePagination` 默认 pageSize 20→10(统一全项目默认值),仅 `file_handler` 文件列表受影响(用户不传 pageSize 时每页 10 条而非 20 条)。

## 未完成(follow-up,resume 处理)
- **task4 cache key 接入**:`cache.go` 已保留 `TokenBlacklistKeyFormat`/`LoginLockKeyFormat`/`CaptchaVerifiedKeyFormat`,但内联点尚未改用:
  - `internal/services/token_blacklist_service.go:71`(`"token:blacklist:"+token`)
  - `internal/api/v1/system/user_unlock_handler.go:43`(`"login:lock:"+req.Username`)
  - `internal/core/captcha.go:485,506`(`login:lock:%s`)、`:379,444`(`captcha:verified:%s`)
- `user_statistics_test.go:18` 注释残留旧名 `constants.MaxPageSize`(低优,注释不影响编译)。

## 验证
- `go build ./...` → exit 0
- `go test ./internal/constants/` → ok
- `go test ./internal/services/operations/ -run 'TestClampPageSizeMath|TestExtractPagination|TestCalculateOffset'` → ok

## 关键设计决策(供 follow-up 参考)
- operations 的 `MaxPageSize=10000` 服务的是"下拉拉全集",**不是**总页数(总页数靠后端独立 COUNT 的 total)。方案 B 保持运行时行为零变化,仅消除同名歧义。
- asset 的 `const MaxPageSize = constants.MaxOptionsPageSize`(保留本地名,引用 constants 值)——最小改动消除分叉。
