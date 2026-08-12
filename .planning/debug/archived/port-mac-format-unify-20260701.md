---
slug: port-mac-format-unify-20260701
status: plan_approved
trigger: "请检查所有网络设备端口相关的逻辑，包括所有定时任务，rest api等等，要求统一端口名称格式和mac物理地址格式"
created: 2026-07-01
updated: 2026-07-01
session_type: refactor
goal: find_and_fix
---

## Current Focus

hypothesis: 7 处格式不一致已识别(MAC 大小写分裂 + 3 份 normalizeInterfaceName 副本 + 3 个 collector 写入路径绕过归一化 + 1 处 regex 漏前导空格 + 1 处 LIKE 不归一化);用户已批准 P0+P1 全套 + MAC 大写 + Port 短名
test: 待执行
expecting: go build + go test 全绿 + 数据 migration 跑通 + 回归测试锁住归一化契约
next_action: dispatch to gsd-debugger subagent with full fix plan

## Symptoms

### Port Name Issues
- **P1**: 3 份并行 `normalizeInterfaceName` 函数 (portcollection/utils.go:29 + lldp/lldp_service.go:79 + lldp/port_classifier.go:84),2026-06-29 物理链路修复只更新了第一份
- **P2**: `internal/collectors/port_collector.go:189-343` parseHuaweiInterfaceLine/Dot1x/PortSecurity 直接存 `matches[1]` 不归一化
- **P3**: `internal/collectors/interface_collector.go:181,232` 存 parts[0] 原始名;`mac_collector.go:160,209` 存 matches[4] 原始名
- **P4**: `port_collector.go:191` regex `^(\S+...)` 不容忍前导空格(虽然 line 170 有 TrimSpace)

### MAC Address Issues
- **M1**: 两套正常化函数大小写相反
  - `mac_history_service.go:90-125` NormalizeMACAddress → 大写
  - `mac_collector.go:280-300` normalizeMACAddress → 小写
  - 写入分裂:`sys_device_mac_address` 全部小写,`sys_device_mac_history` 全部大写,跨表 JOIN 失败
- **M2**: `arp_collector.go:333-334` 只换分隔符不动大小写,3 种格式混存
- **M3**: `mac_collection_service.go:605,636` GetMACAddressList 用 `macAddress LIKE %x%` 不归一化,用户输入大写查不到小写数据

## Eliminated

- hypothesis: 修复需要重写所有采集器 — evidence: 修复点集中在 normalize 调用 + 数据 migration,业务逻辑不变 → ELIMINATED
  timestamp: 2026-07-01
- hypothesis: 端口全称 GigabitEthernet0/0/1 是 XingRan 印刷惯例 — evidence: portcollection 已用短名,SQL JOIN 用短名(P1 P2 P3 一致推荐) → ELIMINATED
  timestamp: 2026-07-01
- hypothesis: 历史数据不能改 — evidence: 用户决策 P0+P1 全套含数据 migration;1247/1483 port_status + 全部 sys_device_mac_address + 全部 sys_device_arp_entries 走 migration → ELIMINATED
  timestamp: 2026-07-01

## Evidence

- timestamp: 2026-07-01
  checked: portcollection/utils.go:29-111 normalizeInterfaceName
  found: 完整双向归一化(全称↔短名),含 25 个前缀
  implication: 当前规范实现,作为单一真实源
- timestamp: 2026-07-01
  checked: lldp_service.go:79-97 normalizeInterfaceName
  found: 副本,只单向,缺 twe/tw/hge/fge/xe/ge 6 项
  implication: 2026-06-29 物理链路修复未同步
- timestamp: 2026-07-01
  checked: port_classifier.go:84-97 NormalizeInterfaceName
  found: 副本,只单向,同样缺前缀;注释自承认"复制自"
  implication: 应删,统一用 portcollection
- timestamp: 2026-07-01
  checked: mac_collector.go:99 db.Create(&mac)
  found: 不调任何 normalizer,直接 INSERT MACAddress
  implication: sys_device_mac_address 列存小写
- timestamp: 2026-07-01
  checked: mac_history_service.go:90-125 NormalizeMACAddress
  found: ToUpper + 重新插冒号
  implication: sys_device_mac_history 列存大写
- timestamp: 2026-07-01
  checked: mac_history_query_service.go:846-850
  found: 注释承认"两者字符串不匹配导致"
  implication: 跨表 JOIN bug 已自我承认

## Resolution

root_cause: 多个正常化函数分散在不同文件 + 采集器写入路径绕过归一化 + 大小写规范分裂(大写 vs 小写)

fix:
1. **MAC 统一大写** (M1)
   - `mac_collector.go:280-300` `normalizeMACAddress` 改为 `strings.ToUpper`(对齐 mac_history_service)
   - 新建 `internal/services/mac_normalize.go` 作为唯一真实源,导出 `NormalizeMACAddress`(大写)
   - `mac_collector.go:280-300` 删本地副本,改调共享函数
   - 新建 `migration_184_normalize_mac_address_to_uppercase.go`:`UPDATE sys_device_mac_address SET mac_address = UPPER(mac_address) WHERE mac_address ~ '[a-f]'`(只更新小写行,避免触发器)
   - 新建 `migration_185_normalize_arp_entries_to_uppercase.go`:同 ARP 表
2. **Port 收口** (P1)
   - `internal/services/portcollection/utils.go:29-111` 改为 `NormalizeInterfaceName`(导出,大写 N)
   - `lldp_service.go:79-97` 删本地副本,改 import 调用 `portcollection.NormalizeInterfaceName`
   - `lldp/port_classifier.go:84-97` 删本地副本,改 import 调用
3. **Collector 写入归一化** (P2/P3)
   - `port_collector.go:189-217` parseHuaweiInterfaceLine 加 `normalizeInterfaceName(matches[1])`
   - `port_collector.go:251-271` parseHuaweiDot1xLine 同
   - `port_collector.go:303-343` parseHuaweiPortSecurityLine 同
   - `interface_collector.go:181,232` 加 normalize
   - `mac_collector.go:160,209` 加 normalize(同时 mac_address 改用 NormalizeMACAddress)
4. **port_collector regex 修复** (P4)
   - `port_collector.go:170` 已 TrimSpace,但 regex 仍 `^` 锚定 → 保持现状(line 已 TrimSpace 后续 regex `^` 是匹配行首,TrimSpace 后等于匹配 TrimSpace 后行首,正确)。实际无 bug,标记为 P4 = 防御性,无修复。
5. **mac_collection_service LIKE 归一化** (M3)
   - `mac_collection_service.go:605,636` 在 LIKE 拼接前 `macAddress = NormalizeMACAddress(macAddress)`
6. **arp_collector 归一化** (M2)
   - `arp_collector.go:333-334` 改调 `NormalizeMACAddress`
7. **历史 port_status 数据归一化**
   - 新建 `migration_186_normalize_port_status_interface_name.go`:
     - 把 `GigabitEthernet\d+/...` → `GE\d+/...`
     - 把 `TenGigE\d+/...` → `XGE\d+/...`
     - 把 `TwentyFiveGigE\d+/...` → `TWE\d+/...`
     - 把 `HundredGigE\d+/...` → `HGE\d+/...`
     - 把 `FortyGigE\d+/...` → `FOE\d+/...`
     - 把 `FastEthernet\d+/...` → `FE\d+/...`
     - 短名大写化:`ge\d+ → GE\d+`,`gi\d+ → GE\d+`(已是大写形式不变),`te\d+ → TE\d+`
8. **回归测试**
   - `internal/services/portcollection/utils_test.go` 已有 `TestNormalizeInterfaceName`,加新场景:
     - 三份函数返回值完全一致(等价锁)
     - 大小写不敏感输入
     - 前导/尾随空格剥离
   - `internal/services/mac_normalize_test.go` 新建:
     - `TestNormalizeMACAddress` 锁住大写契约
     - 验证 `mac_history_service.NormalizeMACAddress` 和 `mac_collector.normalizeMACAddress`(若保留别名)行为一致

verification:
- `go build ./...` 退出码 0
- `go vet ./...` 退出码 0
- `go test ./internal/services/portcollection/ -run TestNormalize` PASS
- `go test ./internal/services/ -run TestNormalizeMAC` PASS
- `go test ./internal/collectors/ -run TestParse` PASS(若存在)
- 手动:跑一次 `bash` 启动后端,采集 1 台设备,确认 sys_device_port_status 全是 GE/TE/XGE 短名大写,确认 sys_device_mac_address 全是大写

files_changed:
- internal/services/mac_normalize.go (NEW, 单一源)
- internal/services/mac_normalize_test.go (NEW)
- internal/collectors/mac_collector.go (删本地 normalize,改调共享)
- internal/collectors/port_collector.go (加 normalize 调用)
- internal/collectors/interface_collector.go (加 normalize 调用)
- internal/services/lldp/lldp_service.go (删本地 normalize,改 import)
- internal/services/lldp/port_classifier.go (删本地 normalize,改 import)
- internal/services/portcollection/utils.go (导出 NormalizeInterfaceName)
- internal/services/mac_collection_service.go (M3 LIKE 归一化)
- internal/collectors/arp_collector.go (M2 归一化)
- internal/core/db/database.go (注册 3 个 migration)
- internal/core/db/migrations/migration_184_normalize_mac_address_to_uppercase.go (NEW)
- internal/core/db/migrations/migration_185_normalize_arp_entries_to_uppercase.go (NEW)
- internal/core/db/migrations/migration_186_normalize_port_status_interface_name.go (NEW)
- internal/services/portcollection/utils_test.go (扩充测试)
- internal/collectors/mac_collector_test.go (扩充,若存在)

status: resolved (4 任务全完成 + verify 8/8 全绿;migration 184 v2 + 187 待用户重启服务自动跑)

## Summary

**任务**: 7 处格式不一致统一化
**用户决策**: P0+P1 全套(包含数据 migration)+ MAC 大写 + Port 短名
**工作流**: dispatch to gsd-debugger for execution,目标 = go build/test 全绿 + 3 个 migration 部署成功
**预计工作量**: 4-6h(代码改 1-2h,migration 1h,测试 1h,部署验证 1-2h)
