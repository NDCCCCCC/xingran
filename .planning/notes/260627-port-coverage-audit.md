# 端口采集覆盖率审计（R1 前置条件）

**审计日期**: 2026-06-27
**目的**: 验证对账引擎的"物理链路"输入是否足够支撑 R1 启动
**R1 启动门槛**: 端口采集覆盖率 ≥ 80%
**审计类型**: 离线 SQL 统计（无需应用部署）

---

## 1. 总体目标

| 指标 | 期望值 | 测量方式 |
|------|--------|---------|
| 端口采集覆盖率 | ≥ 80% | `有效 MAC 端口数 / 信息点数` |
| 工位绑定率 | ≥ 90% | `有 user_id 的工位数 / 总工位数` |
| 资产 MAC 录入率 | ≥ 70% | `有 MAC1/MAC2 的资产数 / 总资产数` |
| 信息点接线率 | ≥ 60% | `有 port_id 的信息点数 / 信息点总数` |

**判定逻辑**：
- 4 项均达标 → R1 启动
- 1-2 项未达标 → 延迟 R1，先做采集治理
- 3+ 项未达标 → 重新评估 R1 范围（可能缩到仅 asset 内部数据）

---

## 2. SQL 采集模板

### 2.1 端口采集覆盖率（T5 - 信息点绑定率）

```sql
-- 计算有效 MAC 端口数（最近 24h 有更新的端口）
SELECT
    COUNT(DISTINCT port_id) AS active_ports_with_mac
FROM sys_port_mac
WHERE deleted_at IS NULL
  AND observed_at > NOW() - INTERVAL '24 hours';

-- 计算总信息点数
SELECT COUNT(*) AS total_info_points FROM sys_info_point WHERE deleted_at IS NULL;

-- 计算有 port_id 的信息点数
SELECT COUNT(*) AS info_points_with_port 
FROM sys_info_point 
WHERE deleted_at IS NULL AND port_id IS NOT NULL AND port_id != '';

-- 覆盖率
-- active_ports_with_mac / NULLIF(total_info_points, 0)
```

### 2.2 工位绑定率

```sql
-- 总工位数
SELECT COUNT(*) AS total_workstations 
FROM sys_workstation 
WHERE deleted_at IS NULL;

-- 有 user_id 的工位数
SELECT COUNT(*) AS bound_workstations 
FROM sys_workstation 
WHERE deleted_at IS NULL 
  AND user_id IS NOT NULL 
  AND user_id != '';

-- 绑定率
-- bound_workstations / NULLIF(total_workstations, 0)
```

### 2.3 资产 MAC 录入率（T6 - MAC 数据质量）

```sql
-- 总资产数
SELECT COUNT(*) AS total_assets 
FROM ops_asset 
WHERE deleted_at IS NULL AND status = 0;  -- status=0 启用

-- 有 MAC1 或 MAC2 的资产数
SELECT COUNT(*) AS assets_with_mac 
FROM ops_asset 
WHERE deleted_at IS NULL 
  AND status = 0 
  AND (mac1 IS NOT NULL AND mac1 != '' OR mac2 IS NOT NULL AND mac2 != '');

-- MAC 录入率
-- assets_with_mac / NULLIF(total_assets, 0)

-- MAC 格式有效率（额外检查）
SELECT COUNT(*) AS assets_with_valid_mac
FROM ops_asset 
WHERE deleted_at IS NULL 
  AND status = 0
  AND (
    mac1 ~ '^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$'
    OR mac2 ~ '^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$'
  );

-- MAC 漂移率（同 MAC 关联到不同资产）
SELECT mac, COUNT(DISTINCT id) AS asset_count
FROM ops_asset
WHERE deleted_at IS NULL AND mac1 IS NOT NULL
GROUP BY mac1
HAVING COUNT(DISTINCT id) > 1;
```

### 2.4 信息点接线率

```sql
-- 已在 2.1 中包含（info_points_with_port / total_info_points）
```

---

## 3. 数据采集报告模板

请运维团队在数据库上执行上述 SQL，填入下表：

### 3.1 基础指标

| 指标 | 数值 | 是否达标 | 备注 |
|------|------|---------|------|
| active_ports_with_mac | _____ | — | 24h 内有效 MAC 端口数 |
| total_info_points | _____ | — | 信息点总数 |
| info_points_with_port | _____ | — | 接线信息点数 |
| **端口采集覆盖率** | _____% | ✅ ≥ 80% / ❌ < 80% | active_ports_with_mac / total_info_points |
| total_workstations | _____ | — | 工位总数 |
| bound_workstations | _____ | — | 已绑定 user_id 工位数 |
| **工位绑定率** | _____% | ✅ ≥ 90% / ❌ < 90% | bound_workstations / total_workstations |
| total_assets | _____ | — | 启用资产总数 |
| assets_with_mac | _____ | — | 有 MAC 的资产数 |
| **资产 MAC 录入率** | _____% | ✅ ≥ 70% / ❌ < 70% | assets_with_mac / total_assets |
| assets_with_valid_mac | _____ | — | MAC 格式有效数 |
| **MAC 格式有效率** | _____% | ✅ ≥ 95% / ❌ < 95% | assets_with_valid_mac / assets_with_mac |

### 3.2 MAC 漂移分析

```sql
-- MAC 漂移数（同 MAC 关联到不同资产）
SELECT COUNT(*) FROM (
    SELECT mac1 FROM ops_asset 
    WHERE deleted_at IS NULL AND mac1 IS NOT NULL
    GROUP BY mac1 HAVING COUNT(DISTINCT id) > 1
) t;

-- 最大漂移数
SELECT mac1, COUNT(DISTINCT id) AS dup_count
FROM ops_asset 
WHERE deleted_at IS NULL AND mac1 IS NOT NULL
GROUP BY mac1 HAVING COUNT(DISTINCT id) > 1
ORDER BY dup_count DESC LIMIT 10;
```

填入：
- MAC 漂移数：_____
- 最大漂移数：_____
- Top 3 漂移 MAC + 资产数：_____

### 3.3 sys_port_mac 数据质量

```sql
-- sys_port_mac 总数
SELECT COUNT(*) FROM sys_port_mac WHERE deleted_at IS NULL;

-- 最近 24h 有更新的端口数
SELECT COUNT(DISTINCT port_id) FROM sys_port_mac 
WHERE deleted_at IS NULL AND observed_at > NOW() - INTERVAL '24 hours';

-- 端口 down 率（端口未在最近 MAC 表中出现）
SELECT 
    COUNT(DISTINCT d.id) AS total_network_devices,
    COUNT(DISTINCT pm.port_id) AS active_ports
FROM sys_network_device d
LEFT JOIN sys_port_mac pm ON pm.port_id LIKE '%' || d.id || '%' 
    AND pm.deleted_at IS NULL 
    AND pm.observed_at > NOW() - INTERVAL '24 hours'
WHERE d.deleted_at IS NULL;
```

填入：
- sys_port_mac 总记录数：_____
- 24h 活跃端口数：_____
- 网络设备总数：_____
- 有 MAC 数据的端口数：_____
- 端口 down 率（粗估）：_____%（1 - active_ports / total_network_devices）

---

## 4. 审计判定

### 4.1 自动判定

```
IF 端口采集覆盖率 ≥ 80% AND 工位绑定率 ≥ 90% AND 资产 MAC 录入率 ≥ 70% AND MAC 格式有效率 ≥ 95%:
    R1 启动 → 继续
ELIF 1-2 项未达标:
    延迟 R1，先做对应治理（如端口采集扩容 / MAC 录入治理）
ELSE:
    重新评估 R1 范围
```

### 4.2 MAC 漂移率额外要求

| MAC 漂移数 | 判定 |
|-----------|------|
| 0 | ✅ 完美 |
| 1-100 | 🟡 需清洗后再启动 R1 |
| > 100 | ❌ 必须先治理 |

### 4.3 端口 down 率额外要求

| down 率 | 判定 |
|--------|------|
| < 20% | ✅ 健康 |
| 20-50% | 🟡 接受范围 |
| > 50% | ❌ 端口采集基础设施问题，先修复 |

---

## 5. 数据收集方式

### 5.1 提交时机

- R1 plan-phase 启动前**至少 3 个工作日**提交
- 团队评审通过后才能进入 plan-phase

### 5.2 提交格式

- 上方 SQL 输出结果截图（pgAdmin / Navicat 等）
- 填入 §3 数据采集报告模板
- 写入 `.planning/notes/260627-port-coverage-audit-result.md` 提交

### 5.3 责任人

| 任务 | 责任人 |
|------|--------|
| 执行 SQL | DBA 或运维同学 |
| 填入报告 | 资产管理 owner |
| 评审判定 | 资产管理 owner + 运维 owner 双签 |

---

## 6. 关联项目记忆

- `workstation-ad-device-managedby-vs-description` — MAC 关联优先于 AD managed_by
- `xingran-migrations-no-sql-autoloader` — sys_port_mac 表结构定义
- `stat-cards-from-list-length-capped-at-100` — 用 COUNT 不用 list.length

---

## 7. 当前状态

- ⏳ **待运维团队采集数据**
- ⏳ **待资产管理 owner 评审**
- ⏳ **待运维 owner 双签**

R1 启动门槛：**未达成**，待数据补充。