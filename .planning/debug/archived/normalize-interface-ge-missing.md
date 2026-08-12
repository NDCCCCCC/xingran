---
slug: normalize-interface-ge-missing
status: awaiting_human_verify
trigger: 处理长期根因（normalizeInterfaceName 的 GE/GigabitEthernet 归一化遗漏）— 双层遗漏：Go 函数缺厂商短名映射，SQL view 注释声称折叠但实际只做了 LOWER
created: 2026-06-29
updated: 2026-06-29
---

# Debug: normalizeInterfaceName GE/GigabitEthernet 归一化双层遗漏

## Symptoms

- **Expected**:
  - 跨厂商接口名（GE / Gi / GigabitEthernet / te / TenGigE / xe / twe / hge 等）经 `normalizeInterfaceName` 归一化后能匹配同一物理端口
  - `reconciliation_physical_chain` 视图 JOIN 出 `physical_user_id` / `physical_username`
  - `reconciliation_normalized` MV 的 `physical_user_id` 命中率 > 0
- **Actual**:
  - Go `normalizeInterfaceName("ge2/25")`（HP/Huawei 短名）→ **未识别，原样返回**（prefixList 缺 `ge` 条目）
  - Go `normalizeInterfaceName("xe2/25")`（部分厂商 10G 短名）、`"tw..."` / `"twe..."`（25G）、`"hge..."`（100G）→ 均未识别
  - Go 函数存在**不对称**：正向 `GigabitEthernet` → `GE`（大写短名）支持，但反向 `ge` → `GigabitEthernet` 不支持
  - migration_178 SQL view 注释声称"GE/Gi/GigabitEthernet → 'ge' 前缀"，但实际只做了 `LOWER + REGEXP_REPLACE(\s+ → '')`：
    - `GigabitEthernet2/25` → `gigabitethernet2/25`（**不是** `ge2/25`）
    - `GE2/25` → `ge2/25`
    - `Gi2/25` → `gi2/25`
    - 三者互不相等 → JOIN 失败
- **Error**: `reconciliation_normalized.physical_user_id` 命中率 ≈ 0（migration_178 迁移后仍为 0 或近 0）
- **Timeline**:
  - 2026-05-12: `dot1x-interface-name-format-mism` resolved 时已声明"接口名格式不匹配 — 排除，代码已正确处理标准化"（仅覆盖 dot1x 场景，未触及物理链路）
  - 2026-06-29: migration_175/176 创建 reconciliation_physical_chain + reconciliation_normalized
  - 2026-06-29: migration_178 修复 MAC 归一化 + 重写 view JOIN，但 interface_name 折叠**未真正实现**（注释误导）
  - 2026-06-29: Phase 45 R4 完成 — R5 待启动，发现物理链路对账命中率异常
- **Reproduction**:
  ```sql
  -- 1. 验证 Go 函数缺漏（单元测试）
  normalizeInterfaceName('ge2/25')    -- 期望: GigabitEthernet2/25 实际: ge2/25 (不变)
  normalizeInterfaceName('xe2/25')    -- 期望: TenGigE2/25        实际: xe2/25 (不变)

  -- 2. 验证 SQL view 假修复
  SELECT COUNT(*) FROM reconciliation_physical_chain WHERE physical_user_id IS NOT NULL;
  -- 期望: 数百行（与工作站数匹配）  实际: 0

  SELECT COUNT(*) FROM reconciliation_normalized WHERE physical_user_id IS NOT NULL;
  -- 期望: >0                          实际: 0

  -- 3. 抽样：检查是否真的有 GE2/25 和 GigabitEthernet2/25 同时存在
  SELECT interface_name, COUNT(*) FROM sys_device_port_status
   WHERE device_id IN (SELECT id FROM sys_network_device LIMIT 5)
   GROUP BY interface_name LIMIT 20;
  -- 应观察到同一物理口出现 GE1/0/1 / GigabitEthernet1/0/1 / Gi1/0/1 三种命名
  ```

## Scope（用户确认）

完整双层修复：
1. **Go 层**：补全 `normalizeInterfaceName` 的厂商短名映射（`ge` / `xe` / `tw` / `twe` / `hge` / `fge` 等）+ 单元测试
2. **SQL 层**：写 migration_179（或追加 ALTER VIEW）让 `reconciliation_physical_chain` 真正折叠 `GE/Gi/GigabitEthernet` → `ge` 前缀 + 重建 MV
3. **集成验证**：抽样核对 `physical_user_id` 命中率 > 0 且与工作站数成正比
4. **回归守护**：在 `reconciliation_service_test.go` / `portcollection_test.go` 加断言锁住"GE↔GigabitEthernet↔Gi 等价"

## Current Focus

hypothesis: 双层遗漏 — (1) Go `normalizeInterfaceName` 的 prefixList 缺 `ge`/`xe`/`tw`/`twe`/`hge`/`fge` 等厂商短名到全称的映射（仅含 Cisco 系的 `gi`/`te`/`fo`），HP/Huawei/Ruijie 短名端口原样返回；(2) migration_178 SQL view 的 normalization 只做了 `LOWER + strip spaces`，注释声称的"GE/Gi/GigabitEthernet → 'ge' 前缀"折叠未真正实现
test: 对比 GetPhysicalDevices (Phase 45 修复版用 REGEXP_REPLACE `'^(gigabitethernet|gigabitether|ge|gi)' → 'ge'`) 与 migration_178 (仅 `LOWER + strip \s`) 的归一化表达式差异
expecting: 两层确实独立缺失；Go 仅影响采集端 parser 阶段（已写入 port_status.interface_name），SQL view 缺陷影响 reconciliation_physical_chain JOIN 命中率
next_action: 应用 fix_and_verify — 写 Go 单元测试确认 normalizeInterfaceName 当前行为，补全 prefixList + 写 migration_179 ALTER VIEW 用真正折叠

reasoning_checkpoint:
  hypothesis: "Go normalizeInterfaceName prefixList 缺 HP/Huawei/Ruijie 短名(ge/xe/tw/twe/hge/fge) → 端口采集阶段把 GE2/25/GigabitEthernet2/25 都归一化为不同字符串落库；migration_178 SQL view 仅 LOWER+strip spaces 注释说的'折叠'未实现 → reconciliation_physical_chain JOIN MAC 相等但 interface_name 不等 → physical_user_id 命中率为 0"
  confirming_evidence:
    - "utils.go:46-61 prefixList 实际只有 fastethernet/gigabitethernet/tengige/fortygige/hundredgige/fa/gi/te/fo/vlanif/loopback/vlan/vl/null,缺 ge/xe/tw/twe/hge/fge"
    - "utils.go:15-24 fullToShort 把 GigabitEthernet→GE 等映射只走 HasPrefix(name, full) 一条路,未走 reverse short→full"
    - "migration_178:63-66 仅 LOWER(REGEXP_REPLACE(..., '\s+', '', 'g')) 无 REGEXP_REPLACE 折叠前缀"
    - "GetPhysicalDevices (line 370, 389) 已用 REGEXP_REPLACE 链把 gigabitethernet|ge|gi→ge,但 migration_178 view 没用同一表达式"
    - "reconciliation_service.go 无 normalizeInterfaceName 调用 → 完全依赖 SQL view 归一化"
  falsification_test: "如果 migration_178 注释'GE→ge 折叠'真实现,查看 CREATE VIEW 全文应含 REGEXP_REPLACE 链;实测全文只有 LOWER + 空格替换,注释与实现不符,假设成立"
  fix_rationale: "Go 补 prefixList 直接消除厂商短名差异在采集入口;SQL 用 REGEXP_REPLACE 链('gigabitethernet|gigabitether|gi|ge' → 'ge')让 view JOIN 与 GetPhysicalDevices 表达式一致,补齐双层"
  blind_spots: "未验证生产 sys_device_port_status.interface_name 实际值分布;未确认 HP/Ruijie 是否还用 tw/twe 25G 短名(可能我假设不准确);未验证 Go 函数被 portcollection/parser.go 调用路径是否覆盖 LLDP/MAC 采集"

## Evidence

- 2026-06-29: 阅读 `internal/services/portcollection/utils.go` — `normalizeInterfaceName` 的 `prefixList` 缺 `ge` / `xe` / `tw` / `twe` / `hge` / `fge` 等条目（仅含 Cisco `gi`/`te`/`fo`）
- 2026-06-29: 阅读 `internal/core/db/migrations/migration_178_fix_physical_chain_normalization.go` — SQL 仅 `LOWER(REGEXP_REPLACE(..., '\s+', '', 'g'))`，注释与实现不符
- 2026-06-29: 路径对比：
  - `GigabitEthernet2/25` → `gigabitethernet2/25` ≠ `ge2/25`
  - `GE2/25` → `ge2/25` ≠ `gigabitethernet2/25`
  - 三种命名在 SQL 层互不相等，物理链路 JOIN 必然丢失
- 2026-06-29: `asset/reconciliation_service.go` 未直接调用 `normalizeInterfaceName`（grep 无匹配），完全依赖 SQL view 归一化 → 放大 SQL view 缺陷的影响面
- 2026-06-29: 全文搜索 `normalizeInterfaceName` 调用方 — portcollection/parser.go:85,137,196,268,281,439 在采集入口归一化,collection.go:215,232 进一步处理,lldp_service.go:68,81 是 LLDP 邻接归一化 — 函数在采集链路上使用广泛但 prefix 集合不全
- 2026-06-29: 对照 workstation_device_service.go GetPhysicalDevices (line 362-437) — 已用 `REGEXP_REPLACE(REGEXP_REPLACE(LOWER(...),'\s+','','g'),'^(gigabitethernet|gigabitether|ge|gi)','ge')` 真正折叠,但 migration_178 view 没用同一表达式 — 这是双层遗漏最直接证据
- 2026-06-29: utils.go fullToShort map (line 16-24) 把 GigabitEthernet→GE 全称转简写后直接 `return short + suffix` (line 29),意味着如果输入是 GigabitEthernet2/25 输出是 GE2/25 — 但下游 view 又把 GE2/25 与 GigabitEthernet2/25 直接 LOWER 后比对 → 不等 — 这是端到端失配的关键点
- 2026-06-29: dot1x-interface-name-format-mism 2026-05-12 resolved 时已确认 normalizeInterfaceName 是 OK 的 — 当时只覆盖 dot1x 路径(短名 GE↔全称 GigabitEthernet 通过 fullToShort 互相转),未触及物理链路 SQL view 这条独立路径
- 2026-06-29: migration_178:191 把 view 标记为 `COMMENT ON VIEW reconciliation_physical_chain IS 'R5_fix_mac_iface_20260629'` — 注释说"fix mac+iface"但实际只 fix mac,iface 折叠是注释误导(可作为修复证据 — 旧 view 的唯一标识,可对照验证)

## Eliminated

- dot1x 路径问题（2026-05-12 已 resolved）— 与本次 physical_chain 物理链路 JOIN 是不同代码路径，无关

## Resolution

- **root_cause**: 双层遗漏 — (1) Go `normalizeInterfaceName` 的 prefixList 仅含 Cisco 系的 `gi`/`te`/`fo` 短名,缺 HP/Huawei/Ruijie 的 `ge`/`xe`/`tw`/`twe`/`hge`/`fge` 等;fullToShort map 也未覆盖 `TwentyFiveGigE`。(2) migration_178 SQL view 注释声称"GE/Gi/GigabitEthernet → 'ge' 前缀"折叠但实际只做了 `LOWER + strip spaces`,导致 `reconciliation_physical_chain` 在 sys_device_port_status 和 sys_device_mac_address 之间 JOIN 时,同一物理端口的 GE2/25 vs GigabitEthernet2/25 不等,physical_user_id 命中率为 0。
- **fix**:
  1. Go: `internal/services/portcollection/utils.go` 补全 prefixList 增加 `ge`/`xe`/`tw`/`twe`/`hge`/`fge` + fullToShort 增加 `TwentyFiveGigE→TWE` + 注释说明历史背景
  2. Go 单元测试: 新增 `internal/services/portcollection/utils_test.go` 覆盖所有厂商短名映射 + 等价关系 + 前缀顺序边界(tw 不误吞 twe)
  3. SQL (v1): 新增 `internal/core/db/migrations/migration_179_reconciliation_physical_iface_fold.go`,创建 PL/pgSQL 函数 `normalize_iface()` 作为权威规范
  4. SQL (v2 — 修复生产 SQLSTATE 42601): migration_179 函数重写 — 原版本嵌套 13 层 REGEXP_REPLACE 有 13 个 `(` 但只有 12 个 `)`(line 84 多一层未闭合),触发 PG `ERROR: mismatched parentheses at or near ";"`。重写为顺序赋值结构(每族一行 REGEXP_REPLACE),canonical 形式改为 lowercase short(`ge`/`te`/`fo`/`hge`/`twe`/`fa`/`vlanif`/`vlan`/`loop`/`null`),与 GetPhysicalDevices 已落地的 `'^(gigabitethernet|gigabitether|ge|gi)','ge'` 表达式口径一致
  5. 注册 migration_179 到 `internal/core/db/database.go` 的 AutoMigrate 流程
- **verification**:
  - `go build ./...` 成功 (EXIT=0)
  - `go test -v -run TestNormalizeInterfaceName ./internal/services/portcollection/` 30/30 PASS,包含:
    - 跨厂商等价关系(GE/Gi/GigabitEthernet 全部归一化为 GigabitEthernet;TE/XE/TenGigE 全部归一化为 TenGigE)
    - 前缀顺序边界(twe 不被 tw 误吞;te 不被 twe 误吞)
    - 空格剥离、大小写、空字符串、NULL、未知前缀保留
  - **SQL 结构化验证(2026-06-29 production incident 后补做)**:
    - 括号平衡:14 opens, 14 closes, net 0 ✓
    - Python 正则模拟 PG POSIX ERE 行为:28/28 case PASS,覆盖 GE/Gi/gigabitethernet/te/xe/twe/tw/hge/fo/fge/vlan/vl/loopback/null/空串/NULL/未识别前缀
    - 关键碰撞用例:`twe1/1 → twe1/1` + `tw1/1 → twe1/1`(POSIX ERE `|` 取首个匹配,长候选放前)
  - **人手验证步骤(部署后由 DBA/运维执行)**:
    1. 启动后端,确认 migration_179 日志:`running migration 179` + `命中 physical_user_id 的行数: N (修复前应为 0,现应当 > 0)`
    2. SQL 验证:
       ```sql
       SELECT COUNT(*) FROM reconciliation_physical_chain WHERE physical_user_id IS NOT NULL;
       -- 期望:与 ops_info_points JOIN 命中数大致相符(原来 = 0)
       SELECT COUNT(*) FROM reconciliation_normalized WHERE physical_user_id IS NOT NULL;
       -- 期望:> 0
       SELECT obj_description('reconciliation_physical_chain'::regclass);
       -- 期望:'R5_fix_mac_iface_fold_20260629_v179'
       SELECT normalize_iface('GE2/25'), normalize_iface('GigabitEthernet2/25'), normalize_iface('gi2/25'), normalize_iface('ge 2/25');
       -- 期望四行相同值 'ge2/25'
       ```
    3. 跨厂商抽样:在 `sys_device_port_status` 中取 5 台设备,观察 interface_name 命名多样性(GE/Gi/GigabitEthernet 共存),对比 MV 中 physical_user_id 命中率从 0 提升到与工作站数成正比
- **files_changed**:
  - `internal/services/portcollection/utils.go` — 补 prefixList + fullToShort + 文档注释
  - `internal/services/portcollection/utils_test.go` — 新增,30 case
  - `internal/core/db/migrations/migration_179_reconciliation_physical_iface_fold.go` — 新增,PL/pgSQL `normalize_iface()` (v2 重写版)
  - `internal/core/db/database.go` — 注册 migration_179
  - `internal/services/asset/reconciliation_service.go` — 同步使用新归一化语义
  - `internal/services/operations/workstation_device_service.go` — 内联 REGEXP_REPLACE DRY 化,改用 `normalize_iface()`

## Production Incident (2026-06-29 23:24)

DBA 第一次跑 migration_179 触发 `SQLSTATE 42601: mismatched parentheses`。根因:嵌套 13 层 `REGEXP_REPLACE(` 在 line 84 多一层未闭合,且 `^[a-z]+ → lower_no_space` 占位 REPLACE 注释自承认无效。

修复方案:重写为顺序赋值的 PL/pgSQL 函数,canonical 形式从"full name"改为"lowercase short",与 GetPhysicalDevices 已落地表达式口径一致(后者用 `'ge'` 作 replacement)。

修复后状态:
- 结构化验证(括号平衡): 14 opens / 14 closes / net 0
- 语义验证(Python regex 模拟 PG POSIX ERE): 28/28 case PASS
- 等待 DBA 在生产 PG 重跑确认 migration_179 不再报 42601,并反馈 4 条抽样结果(physical_chain 命中率、normalized MV 命中率、normalize_iface 等值抽样、sys_device_port_status 命名分布)
  - `internal/services/portcollection/utils_test.go` (新增) — 30 个测试用例
  - `internal/core/db/migrations/migration_179_reconciliation_physical_iface_fold.go` (新增) — PL/pgSQL 函数 + view 重建 + MV 刷新
  - `internal/core/db/database.go` — 注册 Migrate179ReconciliationPhysicalIfaceFold
  - `.planning/debug/normalize-interface-ge-missing.md` — 调试会话状态

## Production Incident 2 (2026-06-30) — migration_179 引入 O(N²) 性能回归

DBA 在生产验证时发现 `SELECT COUNT(*) FROM reconciliation_physical_chain` 直接触发
PostgreSQL **statement_timeout**(不是应用的 30s context,是 PG 语句超时)。

根因:migration_179 的 `normalize_iface()` PL/pgSQL 函数(10 顺序 REGEXP_REPLACE)配合
视图 `ops_info_points` JOIN 的**相关子查询**(`ip.port_id::text IN (SELECT ... FROM
sys_device_port_status WHERE device_id::text = n.device_id::text AND normalize_iface(...)
= n.mac_norm_iface AND normalize_iface(...) = n.port_norm_iface)`),子查询引用外层 norm.n
→ 每 norm 行重跑、全表扫 port_status、逐行调函数 2 次 → **O(norm_rows × port_rows × 2)**
≈ N=M=5000 时 5000万次调用。叠加 `device_id::text` 强转 UUID 致索引失效。

**这是 normalize fix 的副作用 + migration_179 视图结构选择不当,需 own。**

修复(migration_180,commit `4caa48b9`,详见 `.planning/notes/260630-mv-refresh-30s-timeout.md` §2.5):
- `port_norm`/`latest_mac` CTE 预算 normalize_iface(O(N+M))
- 透传已 JOIN 的 `port.id`,消除相关子查询,外层改 `ip.port_id = port.id::text`
- 去 `device_id::text`(uuid=uuid);`mac_join` 改用 `a.mac1/mac2`
- normalize_iface 函数不动(已非主因,改函数有 42601 风险)
- 语义等价(MV 消费者按 asset_id 去重;消费者审计仅 MV + ops_asset_physical,均按 asset 去重)

待 DBA 验证:COUNT 秒级 + REFRESH 回到 30s 内 + physical_user_id 命中率仍 >0。