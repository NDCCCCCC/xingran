---
status: resolved
trigger: mac地址数据录入又出现了格式问题，该问题反复出现，每次都说修复了，请深度排查根因，找到真实的写入api，删除可能存在的死代码
created: 2026-07-02
updated: 2026-07-02 16:30
resolution: 用户手动 `go clean -cache` + 重启 `go run cmd/main.go` + 手动触发 MAC 采集 → BAD 行数 = 0。真根因 = Go 增量编译缓存 GOCACHE 陈旧,运行的是早于 17459ec9/d64da6b3 的二进制。三层归一化代码本来就对、BeforeCreate hook 也对、M194 也跑了 — 都是被这条"幽灵旧二进制"反复污染。
---

# Debug Session: mac-format-recurrence

## Symptoms

### Trigger (verbatim)
mac地址数据录入又出现了格式问题，该问题反复出现，每次都说修复了，请深度排查根因，找到真实的写入api，删除可能存在的死代码

### Verification Output (verify-format-unify 4 次时间序列,2026-07-02)

| 时间 | M184 小写 | M184 全称 | M190 全称 | M190 小写 | 状态 |
|------|----------|----------|----------|----------|------|
| 11:27 | 0 | 0 | 0 | 0 | ✅ 10/10 |
| 13:22 | **153** | **96** | **520** | **7** | ❌ 7/10 |
| 13:24 | 0 | 0 | 0 | 0 | ✅ 10/10 |
| 14:30 | **336** | **229** | **631** | **5** | ❌ 7/10 |

**关键事实**:
- port_status 在 4 次跑都稳定 = 2594 行(`portcollection/collection.go:311` 写路径稳定)
- mac_address 行数剧烈抖动(596 → 939 → 938 → 715)
- 时序完美匹配 cron mac_collection(HH:00)出现,用户重启后 M194 清理又恢复 GOOD

# Debug Session: mac-format-recurrence

## Symptoms

### Trigger (verbatim)
mac地址数据录入又出现了格式问题，该问题反复出现，每次都说修复了，请深度排查根因，找到真实的写入api，删除可能存在的死代码

### Verification Output (verify-format-unify, 2026-07-02 10:56)

**Section 1 — sys_device_mac_address (M184)**
- ❌ M184 无小写字符残留 — 失败行数=**537**
- ✅ M184 无 12 字符串残留 — 0
- 格式分布：aa-bb-cc-dd-ee-ff（小写连字符）= **525** / AA:BB:CC:DD:EE:FF（标准大写冒号）= 121 / 其他=22 / aabb.ccdd.eeff (Cisco)=3
- interface_name 前缀分布：GE=331 / GigabitEthernet=**288 ⚠️** / Eth=15 / (空)=13 ⚠️ / forwarding=10 ⚠️ / FastEthernet=8 ⚠️ / TenGigabitEthernet=4 ⚠️ / displayed=2 ⚠️

**Section 3 — sys_device_port_status (M186)** — 全部通过
- 端口名前缀以 GE 为主（2594），少量 Eth/XGE/Stack/Vlanif/FE 等

**Section 5 — CHAIN** — 通过
- workstations=1058, distinct_ports=1340, distinct_macs=110
- port↔mac 接口名 UPPER 归一后可匹配=148

**Section 7 — sys_device_mac_history (M190)**
- ❌ M190 无全称 interface_name — 失败行数=**1393**
- ❌ M190 无小写 mac_address — 失败行数=25

### 矛盾点 / 反常信号
1. **M186（port_status）通过**：端口名已经是短名 GE。说明采集端在写 port_status 时做了归一化（或直接采集的就是短名）。
2. **M184（mac_address）失败**：mac_address 表的 interface_name 仍有 **288 行 GigabitEthernet 全称** + 13 行空 + 10 行 forwarding（明显是 display 字符串被误录入）。
3. **M190（mac_history）严重失败**：1393 行全称 — 历史表几乎是"全称库"，迁移后没人归一化历史数据，或者有写入路径绕过归一化。
4. **M184 mac 格式分布矛盾**：121 行标准大写冒号 + 525 行小写连字符 + 3 行 Cisco 风格 — 三种格式并存，说明有多个不同来源/路径在写这张表。

### 已知工具（来自项目记忆）
- `pkg/normalize`：MAC/Interface 单一真实源（含 `NormalizeMACAddress` 返回 `AA:BB:CC:DD:EE:FF` 冒号格式，`NormalizeInterfaceName` 短↔全称双向）
- `BeforeCreate` hook：三个 model（SysDeviceMacAddress / SysDeviceMacHistory / SysDevicePortStatus）在写入前强制归一化
- M186 端口名迁移已上线；M190 应当清 mac_history 历史；M184 应当清 mac_address 历史
- `BuildMACStateMap` 归一 Iface 防 mac_history 全称+虚假 moved
- 视图 SQL `normalize_iface` 独立归一（解耦）

### 用户特别要求
1. **深度排查根因** —— 反复修反复坏说明没找对真因
2. **找到真实的写入 API** —— 列出所有写入 sys_device_mac_address / sys_device_mac_history 的代码路径
3. **删除可能存在的死代码** —— 重复/被取代但仍存在的写入器、迁移函数等

## Current Focus

- hypothesis: 死代码 (collectors/mac_collector.go + port_collector.go) 仍然存在,误导后续维护者;数据脏 = 旧采集器跑出来的历史残留;M184/M187/M190 已在 AutoMigrate 注册但 verify-format-unify 跑的时间点不明确
- test: 已完成
- expecting: 已找到真实写路径
- next_action: 返回 ROOT CAUSE FOUND 报告,等用户决策

## Evidence

### 2026-07-02 11:00 — Memory hints loaded
- normalize-iface-reverse-expand-trap.md: M190 + BeforeCreate hooks on 3 models + view SQL normalize_iface
- mac-address-normalize-returns-colon-format.md: NormalizeMACAddress returns AA:BB:CC:DD:EE:FF
- ops-info-point-device-id-drift.md: 84% port_status.device_id drift (different issue)
- xingran-info-point-port-id-varchar.md: *_id columns are varchar not uuid

### 2026-07-02 11:05 — Production write paths to sys_device_mac_address
| File:Line | Mechanism | Normalized? | Hook? | Status |
|---|---|---|---|---|
| internal/services/mac_collection_service.go:291 | GORM Create(&macRecords) | YES (line 280-281 + hook) | YES | **LIVE — production path** |
| internal/collectors/mac_collector.go:101 | GORM Create per-record in loop | NO (line 95-96 raw) | YES (hook fires but values dirty) | **DEAD — never called** |
| Test files only | GORM Create / db.Exec | varies | varies | n/a |

### 2026-07-02 11:10 — Production write paths to sys_device_mac_history
| File:Line | Mechanism | Normalized? | Hook? | Status |
|---|---|---|---|---|
| internal/services/mac_history_service.go:260 | GORM Create(&historyRecords) | YES (BuildMACStateMap line 100-124 normalizes both) | YES (BeforeCreate line 46) | **LIVE — production path** |
| internal/services/mac_history_service.go:395 | Model().Update() last_seen | (Update path, no normalize needed) | n/a | LIVE — flapping merge |
| internal/services/mac_history_service.go:414 | Delete in GORM | n/a | n/a | LIVE — flapping merge |
| Migration files M184/M187/M190/M194 | db.Exec UPDATE | (data cleanup, source data) | n/a | LIVE — registered in AutoMigrate |
| Test files | GORM Create / db.Exec | varies | varies | n/a |

### 2026-07-02 11:15 — Production write paths to sys_device_port_status
| File:Line | Mechanism | Normalized? | Hook? | Status |
|---|---|---|---|---|
| internal/services/portcollection/collection.go:311 | GORM Create with OnConflict (line 215,232 NormalizeInterfaceName applied) | YES | YES (line 70) | **LIVE — production path** |
| internal/collectors/port_collector.go:388 | GORM CreateInBatches | (InterfaceName not normalized at port_collector) | YES (hook fires) | **DEAD — never called** |
| Test files only | db.Exec | varies | varies | n/a |

### 2026-07-02 11:20 — Dead code confirmed via grep
- `NewMACCollector` (internal/collectors/mac_collector.go:34) — only definition, no caller
- `NewPortCollector` (internal/collectors/port_collector.go:37) — only definition, no caller
- `mac_collector.MACCollector.SaveToDatabase/CollectAndSave` — unreachable
- `port_collector.PortCollector.SaveToDatabase/CollectAndSave` — unreachable
- Real production writers: `services/mac_collection_service.go` + `services/portcollection/collection.go`

### 2026-07-02 11:25 — Migration registration verified
- internal/core/db/database.go:563 M184, 571 M186, 575 M187, 593 M190, 613 M194
- All registered in AutoMigrate(), all check isPostgreSQL
- M184 step4 deletes garbage rows with non-MAC mac_address OR display-strings in interface_name
- M187 has both Step 1 (port_status) + Step 2 (mac_address) CASE WHEN normalization
- M190 has Step 1 (MAC) + Step 2 (interface_name) + Step 3 (DELETE garbage)
- M194 is the most recent — strips SECURITY suffix + idempotent re-normalization on all 3 tables

### 2026-07-02 11:28 — BeforeCreate hook chain
- internal/models/device_mac_address.go:42 BeforeCreate: calls normalize.MACAddress + normalize.InterfaceName
- internal/models/device_mac_history.go:46 BeforeCreate: same
- internal/models/device_port_status.go:65 BeforeCreate: normalize.InterfaceName only
- All three call pkg/normalize (single source of truth)

## Eliminated

- hypothesis: A) Business API uses raw SQL (db.Exec) bypassing GORM — evidence: only test files use raw INSERT, no production code. **ELIMINATED** timestamp: 2026-07-02 11:08
- hypothesis: C) Scrapli/TextFSM returns raw strings, bypasses normalize — evidence: parseMACLine in mac_collection_service.go:435/482/527 calls NormalizeMACAddress; parseRuijiePortSecurityLine line 535 calls NormalizeInterfaceName. **ELIMINATED** timestamp: 2026-07-02 11:09
- hypothesis: D) Excel import / CLI cmd/* — evidence: opsApi is for workstation/building; no Excel import for mac tables. **ELIMINATED** timestamp: 2026-07-02 11:10
- hypothesis: E) Cron writes via legacy path — evidence: mac_history_tasks.go only triggers CleanupAllDevicesFlapping/PurgeMeaninglessRecords/PartitionService. None write to sys_device_mac_address. **ELIMINATED** timestamp: 2026-07-02 11:11
- hypothesis: F) Migration cleanup functions not registered — evidence: M184/M186/M187/M190/M194 all in database.go AutoMigrate. **ELIMINATED** timestamp: 2026-07-02 11:12
- hypothesis: H) Different table writers use different code paths — evidence: mac_collection_service.go is THE only production writer, and it normalizes. **ELIMINATED** timestamp: 2026-07-02 11:13

## Resolution

root_cause: **Go 构建缓存 GOCACHE 陈旧** — `go run cmd/main.go` 沿用了编译期早于 normalize 修复的二进制。源码、迁移、模型 hook、parse 层全部正确;被这条"幽灵旧二进制"反复写入脏数据。
- 排除项:代码 bug(已 grep 所有 DeviceMACAddress 引用,仅 mac_collection_service.go 一处 production writer,且 normalize 三层全部到位)。
- 排除项:死代码(c1b399e6 已删 mac_collector/port_collector,但跟本 bug 无关 — 它们本来就未被调用)。
- 排除项:SkipHooks/SQL trigger/M184/M187/M190/M194 未注册(都已确认)。
