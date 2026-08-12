---
status: investigated
trigger: "分公司本部工位 3f130，在其绑定的信息点-网络设备端口 CX-WH-WH-04F-FL-RS8607E-01 GE2/6 上显示有物理地址（MAC），但是工位管理上"物理链路设备"字段为空。请定位根因。"
created: 2026-07-21
updated: 2026-07-21
session_type: observation
goal: find_root_cause_only
related_sessions:
  - workstation-physical-link-zero.md (5F003 — 2026-06-30 已根因定位)
  - info-point-port-drift-recheck-20260701.md (1247/1483 历史漂移)
  - reconciliation-normalized-mv-workstation-id-missing.md (R5 MV 修复)
---

# Debug Session: 工位 3f130 物理链路设备为空

## Current Focus

hypothesis: 5F003 修复（commit 6575a1e7）已合入主干，移除 strict device_id JOIN 并改用 ip.device_id 作 MAC 锚点。3f130 仍报同样症状的最可能原因是 5F003 修复场景之外的**剩余 4 个断点**之一：

1. `ops_info_points.workstation_id` 指向的工位不是 3f130（绑定错位 / 同名 id 误用 / migration_181 未应用）
2. `ops_info_points.workstation_id` 在 sys_workstation 不存在 / 被软删除（sys_workstation.deleted_at IS NULL）
3. 工位 3f130 的 `user_id` 为空（line 339 早退分支 — 对齐 5F003 的 H1 已知可能性）
4. `sys_network_device` 表中 CX-WH-WH-04F-FL-RS8607E-01 不存在 / 被软删除（line 404 EXISTS 过滤）

next_action: 用户提供下列 SQL 查询结果后即可判定（无需进一步静态分析）

expecting: 4 个假设 H1-H4 互斥，验证 SQL 互相排斥；同时可排除 5F003 同源（device_id 漂移）— 因为 6575a1e7 已合并。

## Symptoms

expected: 工位 3f130（分公司本部）下"物理链路设备"应显示该工位用户实际接入的设备（CX-WH-WH-04F-FL-RS8607E-01 GE2/6 上的 MAC）
actual: 工位管理页"物理链路设备"面板显示"该工位暂无通过 MAC→port→infoPoint 反推到的设备"或 (0台)
errors: 前端 `/ops/workstation-device/{id}/physical` 返回 `data: []`
reproduction: 工位管理 → 找到工位 3f130 → 物理链路折叠面板
started: 2026-07-21 报告
codebase_evidence: 5F003 修复 (commit 6575a1e7) 已合入主干:`workstation_device_service.go:380-405` CTE 不再有 strict device_id JOIN

## Eliminated

- hypothesis: 5F003 同源 device_id 漂移 — evidence: commit `6575a1e7 fix(operations): 5F003 工位物理链路 0 台设备` 在 HEAD；`workstation_device_service.go:391-403` 已改为仅按 `port.id = ip.port_id` 锚定单一端口，`ip.device_id AS effective_device_id` 作 MAC JOIN 锚点，line 404 加 `EXISTS(sys_network_device)` 防 orphan。d73c0984 pause commit 未在本分支残留 → ELIMINATED
  timestamp: 2026-07-21
- hypothesis: ClassifySignals.hasPhysical 永远 false (R1/R2 硬编码 NULL) — evidence: `reconciliation_detection.go:105` 现读 `row.PhysicalUserID != nil && *row.PhysicalUserID != ""`；migration_175 + 176 已建真物理链路 LEFT JOIN；migration_181 MV 工作站 ID 修复已合并 → ELIMINATED
  timestamp: 2026-07-21
- hypothesis: 接口名归一化不匹配 — evidence: port_collector.BeforeCreate 与 mac_address.BeforeCreate 均调 normalize.InterfaceName/MACAddress（port-mac-format-unify 2026-07-01 已合并）；migration_186 归一化历史数据；CTE 两侧用相同的 `^(gigabitethernet|gigabitether|ge|gi)` regex 归一 → ELIMINATED
  timestamp: 2026-07-21
- hypothesis: 物化视图 reconciliation_normalized 数据陈旧 — evidence: GetPhysicalDevices 是实时 SQL CTE，不读 MV；只有 ops_asset_physical 表或对账 HealthBadge 才用 MV → ELIMINATED
  timestamp: 2026-07-21

## Evidence

- timestamp: 2026-07-21
  checked: git log --all -- internal/services/operations/workstation_device_service.go
  found: HEAD 上最近相关 commit `6575a1e7 fix(operations): 5F003 工位物理链路 0 台设备 — MAC JOIN 锚点改用 ip.device_id` (2026-07-01) 已合入；`460d95cd fix(backend): R5 物理链路 — 历史 MAC fallback + MergeByTransitions 兜底` (2026-07-02) 紧随其后
  implication: 5F003 类症状的代码路径已修复；3f130 不是同源问题

- timestamp: 2026-07-21
  checked: workstation_device_service.go:320-404 GetPhysicalDevices CTE
  found: 
    - line 339: `if workstation.UserID == nil || *workstation.UserID == ""` 早退返回 `[]`
    - line 401-403: WHERE 子句 `ip.workstation_id = ? AND ip.deleted_at IS NULL AND ip.status = 0`
    - line 404: `AND EXISTS (SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)` 防 orphan
  implication: CTE 内有 3 道过滤可独立返回 0 行：(a) workstation.UserID 空 (b) workstation_id 不匹配 3f130 (c) ip.device_id 不在 sys_network_device

- timestamp: 2026-07-21
  checked: workstation_device_service.go:472-489 MAC/asset JOIN
  found: 
    - `LEFT JOIN latest_mac mac ON mac.norm_iface = wp.norm_iface AND mac.device_id::text = wp.effective_device_id::text`
    - `LEFT JOIN ops_asset a ON (a.mac1/mac2 normalized = COALESCE(mac.norm_mac, hist.norm_mac))`
  implication: 即使 workstation_ports 有行，MAC 没采集到 GE2/6 也返回 0 行；但用户已确认端口**有** MAC，所以此断点不太可能

- timestamp: 2026-07-21
  checked: info_point handler/service 校验
  found: infopoint_service.go Create/Update 不做 device_id ↔ port_id 一致性校验；infopoint_handler.go:69 直接绑定 JSON。6575a1e7 commit msg 声称"infopoint_handler Create/Update 加 device_id↔port_id 一致性校验"，但 main 分支 infopoint_handler.go 是否实际应用待确认
  implication: REST API 路径仍可写错 device_id；但即使写错，因 `EXISTS(sys_network_device)` 兜底，orphan 行会被过滤掉，不影响 MAC JOIN（MAC JOIN 用 `ip.device_id` 锚点而非 `port.device_id`）

- timestamp: 2026-07-21
  checked: ops_info_points.WorkstationID 类型
  found: `gorm:"size:64"` —— VARCHAR(64)，存 UUID 字符串
  implication: workstation_id 是字符串而非 UUID 类型；CTE `ip.workstation_id = ?` (param 是 workstationID 字符串) 直接文本比较；如 3f130 的 workstationID 是 UUID 字符串，两者必须精确相等

## Resolution

root_cause: 静态分析无法独立确认；需要用户提供 4 项查询结果。最大可能性 = 工位绑定错位或工位 user_id 为空（5F003 修复未覆盖的边界）。

fix: 不在本次范围内（仅诊断）

verification: 见下方"待用户提供的关键信息"

files_changed: (N/A)
---

# 根因报告（Root Cause Report）

## 1. 字段归属

- **UI 字段**："物理链路设备（{N}台）" — 前端折叠面板标题
- **数据来源**：`/ops/workstation-device/{id}/physical` HTTP 端点
  - Handler: `internal/api/v1/operations/workstation_device_handler.go:152` `GetPhysicalDevices`
  - Service: `internal/services/operations/workstation_device_service.go:320` `GetPhysicalDevices`
- **返回模型**：`[]*models.WorkstationDevice`，但**不持久化**（CTE 实时查询）
- **关联表**：`sys_workstation`、`ops_info_points`、`sys_device_port_status`、`sys_device_mac_address`、`sys_network_device`、`ops_asset`、`sys_ad_computer`

## 2. 物理链路解析链（实时 SQL CTE）

```
[sys_workstation] workstationID  (line 331)
  ↓ UserID 空 → line 339 早退返回 []                                    ★ H3 断点
  ↓
[ops_info_points] WHERE workstation_id = ? AND deleted_at IS NULL
                  AND status = 0
  ↓ workstation_id 不匹配 3f130 → 返回 0 行                            ★ H1 断点
  ↓
[sys_device_port_status] JOIN port.id::text = ip.port_id
  ↓ port_id 未存或 IP 错指 → 0 行                                       ★ H1b 断点
  ↓
[sys_network_device] EXISTS(id::text = ip.device_id)                   ★ H4 断点
  ↓ device_id orphan → 过滤
  ↓
[sys_device_mac_address] JOIN mac.norm_iface = wp.norm_iface
                          AND mac.device_id::text = wp.effective_device_id::text
  ↓ MAC 未采集/格式不一致 → 0 行                                        ★ H5 断点（用户已排除）
  ↓
[ops_asset] JOIN a.mac1/mac2 normalized = mac.norm_mac
  ↓ 资产无对应 MAC → 命中但无设备详情
```

**关键决策点**：5F003 修复 (commit `6575a1e7`) 已合并，**device_id 漂移不构成本次 3f130 的根因**。剩余 4 个独立断点需用 SQL 逐一验证。

## 3. 假设清单（按可能性排序）

### 假设 H1：工位绑定错位 / workstation_id 解析失败（可能性：高）
- **描述**：`ops_info_points.workstation_id` 指向的工位不是 3f130，或 3f130 的 UUID 与前端传入不一致
- **代码位置**：`workstation_device_service.go:401` CTE WHERE 子句
- **证据**：`OpsInfoPoint.WorkstationID gorm:"size:64"` 字符串列，前端传 UUID 字符串，必须精确匹配
- **成立条件**：信息点 3f130 在数据库中的 workstation_id 字段被误改、或前端 URL 中 ID 错误解析
- **验证 SQL**：见 §4.1
- **验证日志**：无（纯字符串比对）

### 假设 H2：ops_info_points 软删除 / 状态非 0（可能性：中）
- **描述**：`ops_info_points.deleted_at IS NOT NULL` 或 `ops_info_points.status != 0` 导致 CTE 过滤掉
- **代码位置**：`workstation_device_service.go:402-403`
- **证据**：状态约定 `0=正常, 1=故障, 2=停用`；软删除由 GORM `deleted_at` 管理
- **成立条件**：信息点被管理员停用（status=2）或误软删除
- **验证 SQL**：见 §4.2

### 假设 H3：工位 3f130 未绑定 user_id（可能性：中）
- **描述**：`workstation.UserID == nil || *workstation.UserID == ""` 触发 line 339 早退分支，返回 `[]`
- **代码位置**：`workstation_device_service.go:339-341`
- **证据**：5F003 修复 session (2026-06-30) 曾把此列为 H1 高概率假设；UI 上"占用"状态不一定等同于 user_id 绑定
- **成立条件**：sys_workstation.user_id 字段为 NULL / 空字符串
- **验证 SQL**：见 §4.3

### 假设 H4：CX-WH-WH-04F-FL-RS8607E-01 在 sys_network_device 中缺失（可能性：低-中）
- **描述**：line 404 `EXISTS(SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)` 过滤
- **代码位置**：`workstation_device_service.go:404`
- **证据**：6575a1e7 引入该 EXISTS 防 orphan，但若 info_point.device_id 引用的 UUID 在 sys_network_device 已软删除或从未存在 → 整行被丢
- **成立条件**：网络设备被软删除 / 重命名后 info_point.device_id 未同步 / UUID 拼写错
- **验证 SQL**：见 §4.4

### 假设 H5：GE2/6 上 MAC 未采集（可能性：用户已排除）
- **描述**：`sys_device_mac_address` 无 GE2/6 条目
- **用户已确认排除**：报告称"端口上有 MAC"

### 假设 H6：infoPoint 工单已生成但未回写（可能性：低 / 误诊）
- **不适用**：本场景是前端实时查询，不依赖工单回写

## 4. 验证 SQL 模板（4 个独立假设互斥）

### 4.1 验证 H1：workstation_id 绑定一致性
```sql
-- Step 1: 找到 3f130 工位的真实 ID
SELECT id, code, name, location, user_id, deleted_at, status
  FROM sys_workstation
 WHERE code = '3f130' OR name = '3f130' OR location LIKE '%3f130%';

-- Step 2: 找出"声称"绑定到该工位的信息点
SELECT ip.id, ip.name, ip.workstation_id, ip.port_id, ip.device_id, ip.status, ip.deleted_at
  FROM ops_info_points ip
 WHERE ip.workstation_id = (SELECT id FROM sys_workstation WHERE code = '3f130' OR name = '3f130' LIMIT 1);

-- Step 3: 反向核对 —— 工位 3f130 的 ID 是否与 info_point.workstation_id 一致
-- 如果 Step 2 返回 0 行 → H1 成立
-- 如果 Step 2 返回多行 → 检查 status / deleted_at（H2）
```

### 4.2 验证 H2：信息点 status / soft delete
```sql
SELECT id, name, workstation_id, port_id, device_id, status, deleted_at,
       created_at, updated_at
  FROM ops_info_points
 WHERE workstation_id IN (SELECT id FROM sys_workstation WHERE code = '3f130' OR name = '3f130')
    OR device_id = (SELECT id FROM sys_network_device WHERE device_name = 'CX-WH-WH-04F-FL-RS8607E-01' LIMIT 1);

-- 期望 status=0 AND deleted_at IS NULL；
-- 若 status != 0 或 deleted_at 非空 → H2 成立
```

### 4.3 验证 H3：user_id 是否绑定
```sql
SELECT id, code, name, status, user_id, deleted_at
  FROM sys_workstation
 WHERE code = '3f130' OR name = '3f130';

-- 若 user_id IS NULL 或 '' → H3 成立，触发 line 339 早退
```

### 4.4 验证 H4：网络设备是否在 sys_network_device
```sql
-- Step 1: 找到 CX-WH-WH-04F-FL-RS8607E-01 的 ID
SELECT id, device_name, ip_address, status, deleted_at
  FROM sys_network_device
 WHERE device_name = 'CX-WH-WH-04F-FL-RS8607E-01'
    OR ip_address = (SELECT ip_address FROM sys_network_device WHERE device_name = 'CX-WH-WH-04F-FL-RS8607E-01');

-- Step 2: 找 info_point 是否引用这个 device_id
SELECT ip.id, ip.name, ip.workstation_id, ip.device_id, ip.deleted_at, ip.status
  FROM ops_info_points ip
 WHERE ip.device_id = (SELECT id FROM sys_network_device WHERE device_name = 'CX-WH-WH-04F-FL-RS8607E-01' LIMIT 1);

-- Step 3: 反向 —— 如果 info_point.device_id 不在 sys_network_device → H4 成立
SELECT ip.id, ip.device_id
  FROM ops_info_points ip
  LEFT JOIN sys_network_device nd ON nd.id::text = ip.device_id
 WHERE nd.id IS NULL
   AND ip.device_id IS NOT NULL
   AND ip.deleted_at IS NULL;
```

### 4.5 综合诊断（一条 SQL 覆盖 4 个假设）
```sql
WITH ws AS (
  SELECT id FROM sys_workstation WHERE code = '3f130' OR name = '3f130' LIMIT 1
),
nd AS (
  SELECT id FROM sys_network_device WHERE device_name = 'CX-WH-WH-04F-FL-RS8607E-01' LIMIT 1
)
SELECT
  (SELECT user_id FROM sys_workstation WHERE id = (SELECT id FROM ws)) AS ws_user_id,
  (SELECT COUNT(*) FROM ops_info_points
    WHERE workstation_id = (SELECT id FROM ws)
      AND deleted_at IS NULL
      AND status = 0) AS active_info_points,
  (SELECT COUNT(*) FROM ops_info_points ip
    JOIN sys_network_device nd ON nd.id::text = ip.device_id
    WHERE ip.workstation_id = (SELECT id FROM ws)
      AND ip.device_id = (SELECT id FROM nd)
      AND ip.deleted_at IS NULL
      AND ip.status = 0) AS ip_bound_to_correct_device,
  (SELECT COUNT(*) FROM sys_device_mac_address mac
    JOIN ops_info_points ip ON ip.device_id::text = mac.device_id::text
                            AND LOWER(REGEXP_REPLACE(ip.interface_name, '\s+', '', 'g'))
                                = LOWER(REGEXP_REPLACE(mac.interface_name, '\s+', '', 'g'))
    WHERE ip.workstation_id = (SELECT id FROM ws)
      AND ip.deleted_at IS NULL
      AND ip.status = 0) AS mac_match_count;
```

**判读**：
- `ws_user_id IS NULL` → H3
- `active_info_points = 0` → H2（status 非 0 / 已删除）
- `ip_bound_to_correct_device = 0` → H1（workstation_id 错位）
- `ip_bound_to_correct_device = 0` 但 `active_info_points > 0` → H4（device_id 漂移）
- `mac_match_count > 0` 但前端仍显示 0 → 第 6 步 ops_asset JOIN 失败（设备未被资产化）

## 5. 推荐排查路径

1. **先验证 H3**（一行 SQL 即可，耗时 <100ms）
2. **再验证 H1+H2**（workstation_id 是否绑定 + 信息点状态）
3. **再验证 H4**（device_id 是否指向真实网络设备）
4. **最后**看 mac_match_count 是否 > 0，若 > 0 仍有 0 台 → 查 ops_asset JOIN 链

## 6. 修复方向（仅诊断）

| 假设 | 修复方向 |
|------|----------|
| H1 成立 | 修正 `ops_info_points.workstation_id` 指向 3f130 真实 ID |
| H2 成立 | `UPDATE ops_info_points SET status = 0, deleted_at = NULL WHERE id = ?` |
| H3 成立 | `UPDATE sys_workstation SET user_id = ? WHERE id = ?`（需与人事系统核对） |
| H4 成立 | 修正 `ops_info_points.device_id` 指向 `sys_network_device` 中 CX-WH-WH-04F-FL-RS8607E-01 的 id |

## 7. 待用户提供的关键信息

- [ ] 数据库只读访问权限（执行 §4 SQL）
- [ ] 工位 3f130 在 `sys_workstation` 中的实际 `id`（UUID 字符串）
- [ ] 工位 3f130 的 `user_id` 字段值（是否非空）
- [ ] `ops_info_points` 中 "声称"绑定到 3f130 的记录（workstation_id / port_id / device_id / status / deleted_at）
- [ ] `sys_network_device` 中 CX-WH-WH-04F-FL-RS8607E-01 的实际 `id`
- [ ] 可选：sys_job_log 中最近 24h 内 `reconciliation:detectLayer3` 任务执行状态（确认 R5 对账未引入对 station 健康度的副作用）

## 8. 关键证据文件路径

- `D:\code\ClaudeCode\xingran-go-backend\internal\services\operations\workstation_device_service.go:320-575` — GetPhysicalDevices 主实现
- `D:\code\ClaudeCode\xingran-go-backend\internal\services\operations\workstation_device_service.go:339` — user_id 早退分支
- `D:\code\ClaudeCode\xingran-go-backend\internal\services\operations\workstation_device_service.go:391-404` — CTE workstation_ports
- `D:\code\ClaudeCode\xingran-go-backend\internal\api\v1\operations\workstation_device_handler.go:152-166` — Handler
- `D:\code\ClaudeCode\xingran-go-backend\internal\models\operations\infopoint.go` — InfoPoint 模型
- `D:\code\ClaudeCode\xingran-go-backend\internal\models\workstation.go` — Workstation 模型
- `D:\code\ClaudeCode\xingran-go-backend\internal\models\device_port_status.go` — PortStatus 模型
- `D:\code\ClaudeCode\xingran-go-backend\internal\models\device_mac_address.go` — MAC 模型
- `D:\code\ClaudeCode\xingran-go-backend\internal\models\network_device.go` — NetworkDevice 模型
- `D:\code\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\components\operations\WorkstationDeviceTable\index.tsx:64` — 前端调用点
- `D:\code\ClaudeCode\xingran-go-backend\.planning\debug\workstation-physical-link-zero.md` — 5F003 同源 session（已根治）
- `D:\code\ClaudeCode\xingran-go-backend\.planning\debug\info-point-port-drift-recheck-20260701.md` — 1247/1483 历史漂移统计
