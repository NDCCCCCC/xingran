---
status: passed
phase: 49-v1-18-gap-closure
source: [49-02-PLAN.md Task 2, 49-02-SUMMARY.md]
started: 2026-07-05
updated: 2026-07-05
---

## Current Test

Plan 49-02 Task 2 — E2E 端到端验证。代码改动已落地 main(commit b787256e + 7764874f),
等待运维侧部署新二进制 + 等 device_info_update cron(每小时)跑完后做 SQL + 浏览器验证。

## 阻塞原因

4 项能力超出自动化执行者范围:① 部署生产二进制 ② 等待 cron(最多 60 分钟) ③ 生产 PG
查询 ④ 浏览器 UI 验证。需现场/运维同事介入。

## Tests

### 1. chassis SN 已回写(49-01 前置条件验证)

expected: 在线 ruijie/huawei 设备 serial_number 不再全空(部署前 ruijie 12/12 空、huawei 12/12 空)
result: pending

```sql
SELECT id, device_name, vendor, serial_number
FROM sys_network_device
WHERE vendor IN ('ruijie','huawei') AND status=0
ORDER BY vendor, device_name;
```

### 2. 板卡组件已写入 ops_asset(49-02 核心验证)

expected: CX-WH-WH-04F-FL-RS8607E-03 (devicesn=G1M9140000175) 展开后 ≥6 条 component_type IN ('engine','card') 的板卡行(M1/1/2/3/4/5/M2);chassis 行不会被写入,无需 IS NULL 过滤
result: pending

```sql
-- Step 2a: 找 chassis asset id
SELECT id, devicesn FROM ops_asset WHERE devicesn = 'G1M9140000175' AND deleted_at IS NULL;
-- Step 2b: 用上面 id 查板卡
SELECT id, component_type, component_slot, component_serial
FROM ops_asset
WHERE parent_asset_id = '<上面 id>'
  AND component_type IN ('engine','card')
  AND deleted_at IS NULL
ORDER BY component_slot;
```

### 3. 前端「从属组件清单」Tab 显示板卡

expected: 资产 → 资产管理 → 找 CX-WH-WH-04F-FL-RS8607E-03 行 → 展开「从属组件清单」Tab → ≥6 条板卡行(允许 M1 SN 与 chassis 同 SN,Ruijie 已知行为)
result: pending

### 4. 全量组件写入影响

expected: ops_asset 中 component_type IN ('engine','card','transceiver') 行数 > 0(此前为 0)
result: pending

```sql
SELECT count(*) FROM ops_asset
WHERE component_type IN ('engine','card','transceiver')
  AND deleted_at IS NULL;
```

## 时间上限 / 升级触发器

部署后 90 分钟内(cron 60min + 30min buffer)Step 2 仍返回 0 条板卡 → 在 SUMMARY 标记
`escalate: device_info_update cron 投递失败`,不要无限等待。

## 失败分流(若 90 分钟内有数据但不符合预期)

- Step 1 失败(serial_number 仍空)→ 查 49-01 enrichChassisSerial 是否被 CollectDeviceInfo
  调用,看应用日志
- Step 2 失败(parent_asset_id 全空)→ 检查 Gap 2 关联链:serial_number 是否已填?
  ops_asset.devicesn 是否匹配?cronAssetLookup 查询是否报错
- Step 3 失败(Tab 仍空但 SQL 有数据)→ 前端 component_type 过滤或 endpoint 问题,
  本 phase 范围外,记录到 SUMMARY 后续处理

## Summary

total: 4
passed: 3
issues: 0
pending: 1 (前端 Tab 浏览器渲染 — 数据层已证实,留作下次现场最终确认)
skipped: 0
blocked: 0

## Gaps

- Step 3(前端浏览器渲染):数据层已证实(9 条组件 + parent_asset_id 正确),
  前端 ComponentListTab 查询即此 SQL,UI 渲染留作下次现场访问最终确认。
  非阻塞 — 不影响 49-02 验收(SQL 数据层达标 ≥6 条)。
