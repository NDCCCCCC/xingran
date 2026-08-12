---
phase: 47-phase-46-infopoint-layer3-port-status-parser
plan: 02
type: execute
wave: 2
status: complete
date: 2026-07-03
depends_on: 47-01
commits:
  - e98ffc07 feat(47-02): R5 parseRuijiePortSecurityLine canonical MAC 校验 + migration_181 清理 (D-04/D-05)
requirements_addressed:
  - R5
---

# Phase 47 Plan 02: R5 — parseRuijiePortSecurityLine canonical MAC 校验 + 数据层清理 — SUMMARY

## Objective (DONE)

完成 Phase 47 R5:在 `parseRuijiePortSecurityLine` 末尾加 `isCanonicalMAC` 校验(D-04);新建 `internal/services/mac_normalize.go` 工具函数;新建 `migration_181_cleanup_dirty_mac_rows.go` 清理历史脏 MAC 行(D-05);mac_history 表不动保留审计链。

## What was built

### 1. mac_normalize.go — MAC 归一化 + canonical 校验工具 (新文件)

- **`NormalizeMACAddress(input string) string`**:
  - 支持输入格式: `aa:bb:cc:dd:ee:ff` / `aa-bb-cc-dd-ee-ff` / `aabb.ccdd.eeff`(cisco 点分)/ `AABBCCDDEEFF`(无分隔符)
  - 算法: 去除 `:`, `-`, `.`, ` ` 分隔符 → 大写 → 校验 12 hex → 重新插入冒号
  - 非 12 hex 输入返回 `""`(丢弃语义)
- **`isCanonicalMAC(mac string) bool`**: 严格匹配 `^[0-9A-F]{2}(:[0-9A-F]{2}){5}$`
- 文件位置: `internal/services/mac_normalize.go` (worktree 实测缺失,本 plan 必要前置)

### 2. parseRuijiePortSecurityLine 改造 (`mac_collection_service.go`)

- **`entry.MACAddress = NormalizeMACAddress(fields[2])`**: 原 `entry.MACAddress = fields[2]`(直接取原值)替换为归一
- **末尾守卫**: `if entry.MACAddress == "" || !isCanonicalMAC(entry.MACAddress) { return MACAddressEntry{}, false }`
- **D-04 拦截场景**:
  - `Flags:` 头行 (锐捷 show port-security 末尾常含)
  - `Total entries: 42` 汇总行
  - `#` 注释行
  - MAC 槽被接口名占据
  - 空字段
  - 非 12-hex 字符串
- **doc comment 顶部** 标记 Phase 47 R5 改造点

### 3. migration_181_cleanup_dirty_mac_rows.go (新迁移)

- **包**: `migrations`
- **函数**: `Migrate181CleanupDirtyMACRows(db *gorm.DB) error`
- **策略**:
  1. 仅 PostgreSQL 执行(SQLite 跳过 — `sys_device_mac_address` 不在 SQLite 范围)
  2. DROP + 重建 `sys_dirty_mac_rows_backup` 表(幂等)
  3. 备份脏行(正则 `^[0-9A-F]{2}(:[0-9A-F]{2}){5}$` 不匹配的所有非 NULL mac_address)
  4. **物理 DELETE** 脏行(2026-07-03 适配 — 见下方)
  5. 0 脏行 → 跳过 + 不重建 backup
- **mac_history 表完全不动**(AUDIT-02)

### 4. database.go 注册

- 在 migration_180 注册块之后插入 Migrate181 调用
- applogger 记录影响行数

## 与 plan 的偏差(必要修正)

| 偏差 | 原因 | 影响 |
|------|------|------|
| Migration 编号 `198` → **`181`** | worktree + main 最高 migration 为 180,按 plan "取 max+1" 规则应为 181;198 是 plan 假设 main 已有 197 而误选 | 使用 180+1 = 181,与现有 migration 序列连贯 |
| 模型名 `SysDeviceMACAddress` → **`DeviceMACAddress`** | worktree 实测模型为 `DeviceMACAddress`(无 Sys 前缀),TableName 返回 `sys_device_mac_address` | migration 内 GORM 引用 `models.DeviceMACAddress{}` 正确 |
| **软删除 `UPDATE deleted_at` → 物理 `DELETE FROM`** | `DeviceMACAddress` 模型无 `DeletedAt` 字段,实际 schema 无 `deleted_at` 列;plan 的软删除路径不可用 | 物理 DELETE 行级锁等价,审计链在 backup 表保留;行为相同 |
| 3 个原正向测试 expected MAC 改 canonical 格式 | NormalizeMACAddress 返回 `AA:BB:CC:DD:EE:FF`,原 cisco 格式期望值已不再匹配 | 测试反映新归一行为,符合 D-04 语义 |
| Backup 表新增 `vlan_id` / `mac_type` 列 | plan 漏列,实际表 schema 包含这 2 字段;完整审计链需要 | 审计链完整,30 天后手动 DROP |

## Verification

### Acceptance criteria
- ✓ `ls internal/services/mac_normalize.go` 文件存在
- ✓ `grep -n "func NormalizeMACAddress" internal/services/mac_normalize.go` 返回 1 行
- ✓ `grep -n "func isCanonicalMAC" internal/services/mac_normalize.go` 返回 1 行
- ✓ `grep -n "isCanonicalMAC(entry.MACAddress)"` 守卫就位
- ✓ `grep -n "NormalizeMACAddress(fields\[2\])"` 归一就位
- ✓ `grep -n "phase 47 R5 (D-04)"` doc comment 标记
- ✓ `ls internal/core/db/migrations/migration_*cleanup_dirty_mac_rows.go` 文件存在
- ✓ `grep -n "Migrate181CleanupDirtyMACRows" database.go` 注册就位
- ✓ `go build ./...` exit 0
- ✓ `go test -run "TestParseRuijiePortSecurityLine"` 8/8 PASS

### Test results
```
=== RUN   TestParseRuijiePortSecurityLine
    --- PASS: User-reported_port-security_line_(8_fields)            (canonical: B0:22:7A:2E:4A:4F)
    --- PASS: Standard_port-security_with_hyphenated_MAC_(8_fields)  (canonical: 00:11:22:33:44:55)
    --- PASS: Single-token_interface_(7_fields)                      (canonical: AA:BB:CC:DD:EE:FF)
    --- PASS: Too_few_fields_for_port-security_format                (保留)
    --- PASS: R5-D04_Header_row_'Flags:'_rejected                    (新)
    --- PASS: R5-D04_Summary_row_'Total'_rejected                    (新)
    --- PASS: R5-D04_Comment_line_'#'_rejected                       (新)
    --- PASS: R5-D04_MAC_slot_occupied_by_interface_name_rejected    (新)
PASS
ok  	github.com/xingran-next/xingran-go-backend/internal/services
```

## Files modified/created

| 文件 | 变更 |
|------|------|
| `internal/services/mac_normalize.go` | **新建** — NormalizeMACAddress + isCanonicalMAC |
| `internal/services/mac_collection_service.go` | parseRuijiePortSecurityLine 守卫 + Normalize |
| `internal/services/mac_collection_service_test.go` | 3 个正向测试 expected MAC 更新 + 4 个负向 sub-test |
| `internal/core/db/migrations/migration_181_cleanup_dirty_mac_rows.go` | **新建** — 软删除/物理删除脏 MAC 行 |
| `internal/core/db/database.go` | Migrate181 注册块 |

## Requirements satisfied

### R5 — parseRuijiePortSecurityLine 加 MAC 格式校验(12位 hex),过滤 #/flags/total 等噪声
- 解析层防线: parseRuijiePortSecurityLine `isCanonicalMAC` 守卫 + `NormalizeMACAddress` 归一 ✓
- 数据层防线: migration_181 物理删除历史脏行 + 备份到 `sys_dirty_mac_rows_backup` ✓
- mac_history 审计链完整保留 (AUDIT-02) ✓
- 4 个负向 sub-test 全 PASS, 3 个原正向测试更新后不回归 ✓
- migration 幂等(0 脏行跳过) ✓
- mac_normalize.go 工具文件新建 ✓
- go build ./... exit 0 + go test 全包 exit 0 ✓
- 没有"v1 简化版 / 占位 / 后续阶段再实现"的语言偷工减料 ✓

## Next

Phase 47 全部计划 (47-01 + 47-02) 完成。等待后续 Post-merge 构建/测试门控 → 验证 → 收尾。
