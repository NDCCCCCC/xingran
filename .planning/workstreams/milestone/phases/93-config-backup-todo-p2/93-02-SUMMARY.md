---
phase: 93
plan: 02
subsystem: device/restore
tags: [scrapli, restore, fail-fast, e2e]
key-files:
  created:
    - internal/device/restore_config.go
    - internal/device/restore_config_test.go
    - internal/device/testdata/restore_huawei_success.fixture
    - internal/device/testdata/restore_huawei_rejected.fixture
  modified:
    - pkg/constants/timeouts.go
    - pkg/constants/timeouts_test.go
    - internal/device/scrapli_wrapper.go
metrics:
  tests_added: 4
  commits: 2
---

# Plan 93-02 Summary: DeviceExecutor.RestoreConfig 下发内核

## What Was Built

1. **RestoreConfigTimeout = 10min**（pkg/constants/timeouts.go，D-05）+ AST 锁值双锁同步（Stability map + Count 6→7）
2. **DeviceExecutor.RestoreConfig**（internal/device/restore_config.go，D-02）：与 GetConfig 对称（pool.GetDevice → vendor → ExecuteCustom）；**一次 AcquirePriv 批量下发**（scrapligo 原生 StopOnFailed fail-fast，D-12）；基础清洗 cleanConfigLines（D-03）；vendor 退出命令 exitConfigCommand（D-06：huawei/h3c→quit、其余→exit，与 VendorExitViewCmd 实证对齐，无 end）；结构化 RestoreResult（D-07）
3. **ScrapliWrapper.SendConfigsStopOnFailed**：新增批量方法（opoptions.WithStopOnFailed），中断时返回含失败行的部分 responses
4. **e2e ×2**（FileTransport 零 SSH，mustDebugLogger 使 fixture 错位可见而非静默挂起）+ 纯函数 ×2

## Key Discovery（执行中的重要发现）

`wrapper.SendConfigs` 是**逐行 driver.SendConfig 循环**——每行一次完整 AcquirePriv 舞步（GetPrompt 的 write\n+读 prompt），千行配置既低效又使 FileTransport fixture 消费翻倍。RestoreConfig 改用新增的 `SendConfigsStopOnFailed`（一次 AcquirePriv + scrapligo 原生 StopOnFailed），效率与 fail-fast 语义双优。fixture 需为 pool.GetConnection 的 `IsReady()` GetPrompt 探测留 spare prompt 行（79_06 同款纪律）。

## Commits

| Commit | Description |
|--------|-------------|
| 203feee | feat(93): add RestoreConfigTimeout constant with AST lock update (D-05) |
| a0a3454 | feat(93): add restore-config executor with vendor map and fail-fast (D-02/03/06/07/12) |

## Deviations

- fail-fast 用例从"fixture 截断 EOF"改为"设备拒绝 marker（% Error → failed_when_contains）"——FileTransport EOF 是无限阻塞而非错误返回，error-marker 才是可自动化的 fail-fast 验证路径。
- RestoreConfig 放独立文件 restore_config.go（plan 原案），executor.go 零改动。

## Self-Check: PASSED

- `go build ./...` + `go vet ./internal/device/` exit 0
- `go test ./internal/device/ -count=1` → 全绿（18.8s，含新 4 用例与存量零回归）
- `go test ./pkg/constants/ -count=1` → 全绿（双锁含 RestoreConfigTimeout）
