# v1.24 规划输入 — 3f130 / 工位绑定现状（生产 PG 只读实测）

> 实测：2026-10-28｜PostgreSQL `10.62.10.34:5432/ruoyi`（只读）
> 用途：DIAG-01（3f130 空因前端展示）的完成判据与 UI 分级依据

## 1. 表结构事实（纠正 v1.23 的一处记述）

| 表 | 关键列 | 类型 |
|---|---|---|
| `sys_workstation` | `id` | **uuid** |
| | `workstation_name` | character varying（如 `3F130`、`47F049`、`1F07`） |
| | `user_id` | **character varying**（存 uuid 字符串） |
| | `user_name` | character varying |
| | `dept_id` / `dept_name` / `floor_id` / `floor_name` / `building_id` / `building_name` / `position_x` / `position_y` / `rotation` / `width` / `depth` / `desk_type` | — |
| `ops_workstation_device` | `workstation_id` | **character varying**（存 uuid 字符串） |
| | `responsible_user` | character varying（**中文姓名**） |
| | `responsible_user_id` | character varying（**全表为空**，见 §3） |
| | `device_serial` / `device_name` / `device_model` / `device_type` / `mac_address` / `asset_id` / `ad_computer_id` / `ip_address` / `is_primary` / `priority` / `status` | — |

> ⚠️ **JOIN 类型坑**：`sys_workstation.id` 是 uuid，`ops_workstation_device.workstation_id` 是 varchar → 直接 `d.workstation_id = w.id` 报 `operator does not exist: character varying = uuid`。必须写 `d.workstation_id = w.id::text` 或 `d.workstation_id::uuid = w.id`。（v1.23 已记录过同类坑：`ops_info_points.workstation_id`）

## 2. 3F130 现状（H3 空因确认）

```
sys_workstation.workstation_name = '3F130'  →  user_id = NULL
```

**确认**：工位 3F130 **未绑定用户**，与 v1.23 §4.5 实测结论一致（`mac_match_count=1` 但前端按用户维度呈现仍空白）。

## 3. 全站分布（关键设计输入）

### 3.1 工位绑定用户

| 指标 | 数量 | 占比 |
|---|---|---|
| 工位总数 | 1643 | 100% |
| **未绑定用户**（`user_id` 为空） | **1238** | **75.3%** |
| 已绑定用户 | 405 | 24.7% |

### 3.2 工位-设备关联

| 指标 | 数量 |
|---|---|
| `ops_workstation_device` 有效记录（`deleted_at IS NULL`） | 445 |
| 其中 `responsible_user_id` 为空 | **445（100%）** |
| 其中 `responsible_user`（中文姓名）非空 | 259（58.2%） |

**结论**：设备责任人**只有中文姓名、没有 uuid**——`responsible_user_id` 字段在生产中从未被写入。因此：

- 诊断端点的 H3 判定应基于 **`sys_workstation.user_id`**，而非 `ops_workstation_device.responsible_user_id`（后者恒空，会把所有工位都判成 no_user_bound）
- 设备侧的 `responsible_user` 中文名**不能**作为「已绑定用户」的判据

## 4. 对 DIAG-01 的设计约束

### 4.1 空因分级（因 `no_user_bound` 占 75.3%，必须分级，否则 3/4 工位都是警告噪音）

| 级别 | 空因 | 依据 | UI 呈现 |
|---|---|---|---|
| **中性（信息）** | `no_user_bound`（工位未绑定用户） | 75.3% 工位常态 | 灰色 info 说明：「该工位未绑定用户，无法按用户维度呈现物理链路」——不是错误 |
| **提示（warning）** | 数据部分缺失（`responsible_user_id` 恒空、MAC 未采集、资产未关联） | 结构性缺陷 | 黄色 warning，提示可补录 |
| **错误（error）** | `mac_mismatch` / 链路断 / 查询失败 | 真正的异常 | 红色 error + 重试入口 |

### 4.2 完成判据（可机械验证）

1. 工位管理页打开 `3F130` 的物理链路设备面板 → **不再空白**，显示诊断原因文案
2. 该原因取自 `POST /ops/workstation-device/:id/physical-diagnosis` 的 `emptyReason` / `emptyMessage`
3. 分级符合 §4.1：`no_user_bound` 渲染为中性 info，非 error
4. `warnings` 数组（如 `responsible_user_id` 未回填）单独以 warning 呈现，不与主因混淆
5. **对照组**：绑定用户的工位（如 `47F049`，`user_id` 非空）→ 正常展示设备列表，**不出现**空因提示
6. 该页面为只读诊断，**不写生产数据**（用户裁定：本里程碑只做前端展示，数据侧绑定留用户动作）

## 5. 遗留（不在 v1.24 修）

- `responsible_user_id` 恒空 → 需写入侧补 uuid（涉及 `ops_workstation_device` 的 Excel 导入/手工录入链路），属独立技术债
- 3F130 绑定 user_id → 业务核对后由用户执行
