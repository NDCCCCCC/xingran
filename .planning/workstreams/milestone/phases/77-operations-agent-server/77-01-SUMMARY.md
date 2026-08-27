---
plan: 77-01
phase: 77-operations-agent-server
executed: 2026-08-27
commits:
  - 8568f86 (test: sqlite 7-table fixture + query-chain tests)
  - 004e540 (fix(quirk-77-1): GetADDevices IPAddress 映射,AD>req merge priority)
  - 8d3675c (test: 续成分支覆盖 + WIP 现场修复)
---

# 77-01 Summary — workstation_device sqlite 直测

## 交付

- `internal/services/operations/workstation_device_77_01_test.go` (~1244 行):sqlite 内存库 7 表 fixture(setupWSD77DB) + **19 个 TestWSD77_* 测试函数**,覆盖查询链(GetADDevices/GetAssetDevices/List/统计)、写链(AddDeviceManual/Update/Delete/SetPrimary*)、SyncFromAD/SyncFromAsset 全分支、mergeBySerial 三态(ad-only/asset-only/双源合并优先级 :53-61)、SetPrimaryAndSave(BySerial) 参数分支(nil req/空 ws/空 serial/工位不存在)。

## Coverage checkpoint(计划要求落 SUMMARY)

- **基线 61.1% → 实测 69.8% (+8.7pp)**,超过 ≥4pp 门槛。距 BLOCK-01 的 70.0% 差 ~0.2pp,由 77-02(excel 导出链)/77-03(导入分支+卫星文件)按计划余量收口。
- `go build ./...` exit 0;生产 .go 改动仅 quirk-77-1 一行(004e540,D-01 登记)。

## Quirks 处置(D-01/D-03)

- **quirk-77-1(修复)**:`GetADDevices` 转换漏映射 IPAddress → AD>req merge priority 下 IP 永远丢失。服务端补一行映射(`004e540`),测试锚定 "10.1.1.1" 断言。

## Deviations / 现场抢救记录

本 plan 执行横跨会话上下文压缩断点。续作现场(工作树 +607 行未提交 WIP)存在三处缺陷,均修复向前并经 `-count=3` flake 筛查:

1. `:705` `_, err := svc.SyncFromAD(...)` — 服务签名单返回值(error),删多余空白变量。
2. `:768` 同款 `SyncFromAsset` 双值误配,同修。
3. `:1072` SetPrimaryAndSave 参数分支注释标「空 ws」但实参误写已存在的 `wsd77WS1`(拷贝笔误),导致合法请求返回 nil error;按分支语义改为 `WorkstationID: ""`。

## Acceptance criteria 对照

| 标准 | 结果 |
|---|---|
| TestWSD77_ 函数 ≥15 | ✅ 19 |
| mergeBySerial 双源优先级用例(mac/ip/model 来源断言) | ✅ SN-BOTH 合并用例 |
| coverage 提升 ≥4pp 且数字入 SUMMARY | ✅ +8.7pp → 69.8% |
| go build ./... == 0 | ✅ |
| 生产 .go 改动 0(除登记 quirk) | ✅ |
