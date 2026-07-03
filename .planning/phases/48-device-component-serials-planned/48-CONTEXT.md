# Phase 48: 网络设备组件序列号采集 - Context

**Gathered:** 2026-07-04
**Status:** Ready for planning
**Source:** /gsd-explore session (6 轮讨论, 2026-07-03) + 36 个真机样本

<domain>
## Phase Boundary

将网络设备（交换机/路由器）的板卡/引擎卡、电源、风扇、光模块各自的序列号**作为资产设备纳入 `ops_asset`**，保存组件对交换机/路由的**从属关系**，前端展示"从属组件清单"，对接 v1.17 资产对账。

**In scope**:
- `ops_asset` 模型扩展（4 新列）
- 采集器双路径：SNMP ENTITY-MIB（主）+ CLI（补）
- 匹配不上生成对账异常
- 前端"从属组件清单"视图

**Out of scope**:
- H3C/迈普（无设备）
- 锐捷光模块 DDM TextFSM（本设备无光模块，需另环境）
- 自动化创建 ops_asset 行（采集只匹配不新建）

</domain>

<decisions>
## Implementation Decisions

### 数据建模

### D-01: 组件作为 `ops_asset` 行（不是独立子表）
- 每个有 SN 的组件（板卡/引擎/电源/风扇/光模块）→ `ops_asset` 一条记录
- `DeviceSN` = 组件自己的序列号（`ops_asset` 主键已存在）
- **不新建** `device_component` / `device_inventory` 之类独立子表
- 背景：资产系统统一为"序列号 = 资产"，贴合外部资产 Excel 导入习惯
- 源码：Note §2 锁定的设计决策

### D-02: 采集只匹配不新建（UPDATE-only on ops_asset）
- SSH/SNMP 采集发现的组件 SN 必须在 `ops_asset` **查找已有行**
- 匹配到 → UPDATE `parent_asset_id` + `source_device_id` + `component_type/slot`
- 匹配不上 → 生成对账异常（见 D-06）
- **不**为采集发现的组件自动 INSERT `ops_asset` 行（保住实物/声明分层）
- 原因：`ops_asset` 至今只有 Excel 导入一条路径，引入自动写入会污染 v1.17 分层

### D-03: 双向桥接（parent_asset_id + source_device_id）
- `parent_asset_id`：自引用 → `ops_asset.id`（指向交换机本体在 ops_asset 的行）
- `source_device_id`：→ `sys_network_device.id`（SSH/SNMP 采集来源）
- 两边都写，资产系统内（parent）+ 采集来源（source）双锚

### D-04: 父交换机不在 ops_asset 时降级处理
- 若采集时父交换机还不是 `ops_asset` 行 → `parent_asset_id` **留空**
- `source_device_id` 仍写（采集来源总是可定位）
- **不**为此额外报异常（交换机整机缺失是另一层问题，不混入组件对账）

### D-05: component_type / component_slot 新增专用列
- `component_type`：`chassis | card | engine | power | fan | transceiver`
- `component_slot`：槽位/接口位置（如 `Slot 1` / `GE1/0/24`）
- 新增专用列，**不**复用现有 `AttributeValue` JSON（因为要对账分类筛选）
- DB schema：扩展 `ops_asset` 表

### 对账集成

### D-06: 组件对账异常新增专属 category（sibling 列方案）
- 复用 v1.17 `sys_data_reconciliation` 表，**新增 sibling 列 `recon_category varchar(32)`** + 新字典 `asset_reconciliation_recon_category`
- `conflict_type` 仍用 **F（数据缺失）**，承载"实物/账本存在性差异"
- `recon_category` 承载"业务类型分类"（`component_serial` / 后续可扩展）
- 解锁 UI 单独筛出组件异常，不影响 A-E 现有语义
- 必走 migration_NNN：ADD COLUMN + DROP old partial unique `uniq_recon_asset_type_open` + CREATE `uniq_recon_asset_type_cat_open (asset_id, conflict_type, recon_category) WHERE open`
- 异常类型：
  - 实物有账无（盘盈/未登记）：conflict_type=F, recon_category='component_serial'
  - 账有实物无（盘亏/已拆）：conflict_type=F, recon_category='component_serial'

### D-07: 资产列表/统计默认排除组件行
- `asset_service.go` `List()` / `Statistics()` 默认加 `component_type IS NULL` 过滤
- 避免 "1 台交换机 + 6 块板卡" 被数成 7 台设备
- 组件另开独立视图（按 `parent_asset_id` 聚合）

### 采集技术路线（双路径）

### D-08: 主路径 = SNMP ENTITY-MIB 单 GET
- 走现有 `internal/device/snmp_client.go` + `gosnmp v1.35.0`
- **走单 GET**（不是 BulkWalk）—— 实测华为 S8700 / 锐捷 RS8607E 都拒绝 GETBULK
- 拉 OID：`entPhysicalSerialNum(.11)` / `entPhysicalModelName(.13)` / `entPhysicalClass(.5)` / `entPhysicalContainedIn(.4)` / `entPhysicalName(.7)`
- community 从 `sys_auth_credential.snmp_communities` 读，**别用默认 `public`**
- 实测覆盖：
  - 华为 S8700 V600R024C00：chassis / module / fan / 业务板 + 引擎 ✅
  - 锐捷 RS8607E：chassis / fan / 电源 / M1-M2 主控 ✅
  - 业务板 slot 1-5 在锐捷仅占位（SN 在 CLI 拿）
  - 华为电源 SN 在 ENTITY-MIB 返空（HW 漏洞）

### D-09: CLI 补充路径
- 锐捷 `show version`：拿业务板 slot 1-5 的 SN（一段搞定整机+所有模块）
- 华为 `display interface transceiver`：光模块 SN + DDM 唯一来源
- 华为 `display device esn`：整机 SN 互验（与 SNMP chassis SN 交叉验证）
- 华为 `display device` / `display device board`：硬件清单骨架

### D-10: 光模块只采在用/up 接口
- `show interfaces status` / `display interface status` 拿到 up 状态接口列表
- 锐捷光模块无插 → 跳过；华为 S8700 上 10GE5/0/4 有模块可采
- 平衡覆盖（避免扫大量空口）与采集耗时

### D-11: 锐捷私有 typo 噪声过滤
- 锐捷 ENTITY-MIB 中 `powerSupply(8)` 类的 `temprature*`（typo for temperature）节点是私有扩展伪实体
- 全表 352 个噪声，必须按 Name 前缀 `temprature*` 剔除
- 过滤规则写在 SNMP 采集器内（"丢 Class=8/powerSupply 且 Name 以 temprature 开头的节点"）

### D-12: 堆叠设备序列号归属
- 华为 iStack / H3C IRF / 锐捷 VSU 等堆叠场景
- 采集用 `entPhysicalContainedIn` 重建归属
- 板卡归属到本机 chassis，引擎卡归属到本机（Master/Slave 标记从 M1/M2 推断）
- 本环境无堆叠设备，v1 实现需保留接口但 UAT 难

### D-13: 操作日志
- module 名称：「资产管理」
- OperType：14 (OperTypeSync) — 同步采集
- `parent_asset_id` / `source_device_id` 更新时记
- 敏感字段（无 — 不涉及密码/密钥）走普通 `operlog.Record()`

### D-14: 依赖 v1.17 对账底座
- Phase 48 = 对账引擎新增"组件序列号"维度
- 是 v1.17 资产对账的 R2 物理层对账的具体形态
- 依赖 Phase 42-46（v1.17 已 shipped 2026-07-03）

### Claude's Discretion

- 前端"从属组件清单"具体 UI（表格 vs 卡片 vs 树形） — 实施时由前端 designer 决定
- TextFSM 模板具体行号 / 字段位置 — 已在真机样本中固化
- 采集调度（cron 间隔 / 触发器）— 复用现有 `DeviceInfoCollectionService` cron

</decisions>

<canonical_refs>
## Canonical References

**下游 agent (planner/checker) 必须先读这些。**

### 网络设备模型与采集
- `internal/models/network_device.go` — `NetworkDevice` 字段定义
- `internal/models/asset.go` — `ops_asset` 模型（要扩展）
- `internal/device/scrapli_wrapper.go` — Scrapli SSH 连接 + `SendCommand`
- `internal/device/snmp_client.go` — `SNMPClient` + `Walk` + `Get`
- `internal/device/connection_pool.go` — `GetConnection` + `Execute` 模式
- `internal/device/executor.go` — `DeviceExecutor` + `ExecuteOnDevice`
- `internal/services/portcollection/collection.go` — 参考架构（vendor 命令分发 + 批量 upsert）
- `internal/services/portcollection/parser.go` — vendor 命令分发模式（`getInterfaceCommand` 等）
- `internal/services/device_info_collection_service.go` — 现有异步采集 service（按 vendor 分发命令）

### 对账与资产系统
- `internal/services/operations/excel_config.go` — `ops_asset` Excel 导入（设备序列号 `devicesn` 作为 UpsertKey）
- `internal/core/db/migrations/148_create_ops_asset_table.go` — `uni_ops_asset_devicesn` UNIQUE 约束
- `internal/models/reconciliation.go` — `SysDataReconciliation` 模型（`asset_id` 外键 → `ops_asset.id`）
- `internal/services/reconciliation/reconciliation_detection.go` — 检测引擎（决定是否可扩展 category）
- `internal/services/reconciliation/reconciliation_service.go` — 检测 service

### 凭据与认证
- `internal/models/auth_credential.go` — `snmp_communities` 数组字段存 SNMP community
- `internal/core/db/database.go:786-808` — `snmp_community` → `snmp_communities` 数组转换
- `pkg/crypto/` — SM4 cipher 路径（连接池解密用）

### 操作日志
- `internal/utils/operlog/` — operlog helper
- `internal/api/v1/system/ad_domain_handler.go` — 参考实现（Phase 34 标准）

### 现有真机样本与文档
- `.planning/notes/260703-network-device-component-serials.md` — 完整设计决策
- `.planning/research/questions.md` — RQ-001 真机采样数据 + 厂商命令表
- `.planning/seeds/snmp-entity-mib-acceleration.md` — 标记 SUPERSEDED（决策已并入 v1）
- `templates/samples/huawei_10_62_25_253_*` — 36 个真机原始回显样本
- `templates/samples/ruijie_10_62_63_21_*` — 36 个真机原始回显样本

</canonical_refs>

<specifics>
## Specific Ideas

### 数据模型扩展骨架

```go
// internal/models/asset.go 扩展 Asset（ops_asset）
type Asset struct {
    // ... 现有 60+ 字段 ...
    ParentAssetID  *string `gorm:"size:64;index;column:parent_asset_id" json:"parentAssetId,omitempty"`   // 自引用 → ops_asset.id
    SourceDeviceID *string `gorm:"size:64;index;column:source_device_id" json:"sourceDeviceId,omitempty"` // → sys_network_device.id
    ComponentType  *string `gorm:"size:32;column:component_type" json:"componentType,omitempty"`         // chassis/card/engine/power/fan/transceiver
    ComponentSlot  *string `gorm:"size:64;column:component_slot" json:"componentSlot,omitempty"`         // 槽位/接口位置
}
```

### 采集侧抽象

```go
// internal/services/device_component_collector/service.go（新）
type DeviceComponentCollector interface {
    // 单设备采集:返回 1 个 chassis + N 个 component
    CollectDeviceComponents(ctx, deviceID) (*ComponentSet, error)
}

type ComponentSet struct {
    Chassis     *Component  // 整机 1 个
    Components  []Component // 板卡/电源/风扇/光模块
}

type Component struct {
    ComponentType string  // chassis/card/engine/power/fan/transceiver
    Slot          string  // Slot 1 / PWR1 / GE1/0/24
    SerialNumber  string  // 主键(对账比对键)
    Model         string  // 部件型号
    Source        string  // "snmp" | "cli-huawei" | "cli-ruijie"
    Raw           string  // 原始证据
}
```

### 真机数据样本（已在 36 个文件中固化）

**锐捷 RS8607E `show version`**（一段拿全部）：
```
System serial number    : G1HLC0R000096
Module information:
  Slot M1 : M8600E-CM
    Serial number       : G1HLC0R000096
  Slot 1 : M8600E-24GT20SFP4XS-ED
    Serial number       : G1P7286000129
  ...
```

**华为 S8700 ENTITY-MIB**：
```
chassis  CloudEngine S8700-6           102599861597
fan      LSG7G48VX1E0 1                 102599806093
fan      LSG7SRUEX1C0 5 (Master)        102599806030
fan      LSG7SRUEX1C0 6 (Slave)         102599806031
stack    FAN 1                          5B2599001652
```

**华为 S8700 光模块**：
```
10GE5/0/4 transceiver information:
   Vendor Name : HUAWEI
   Manu. Serial Number : 8000012000082
   Manufacturing Date  : 2025-2-20
```

### 锐捷+华为 ENTITY-MIB 实测差异

| 组件 | 华为 S8700 (SNMP) | 锐捷 RS8607E (SNMP) |
|------|-------------------|---------------------|
| chassis | ✅ `102599861597` | ✅ `G1J40D100022A` |
| 业务板/单板 | ✅ via class=fan | ⚠️ 仅占位（SN 在 CLI） |
| 引擎/主控 | ✅ via class=fan | ✅ via class=fan |
| 风扇 | ✅ via class=stack | ✅ via class=stack |
| 电源 | ❌ **空（HW 漏洞）** | ✅ RG-PA600I 三个 SN |

</specifics>

<deferred>
## Deferred Ideas

- **H3C / 迈普 模板**：本环境无设备，模板必须留待其他环境
- **锐捷光模块 DDM TextFSM**：~~本设备 RS8607E 所有光口都没插模块，需另一台有 SFP 的设备采样~~ — **2026-07-04 升级为 in-scope**：researcher 复查发现样本里 10GE 1/47 + 1/48 有 2 个 SFP 已插（含 DDM），现成可写
- **SNMPv3 适配**：当前社区用 v2c 即可；v3 (auth/priv) 待需要时再说
- **采集调度优化**：当前用现成 `DeviceInfoCollectionService` cron，不做增量采集
- **板卡迁移事件追踪**：板卡从一台设备搬到另一台的事件流不在 v1 范围

---

*Phase: 48-device-component-serials*
*Context gathered: 2026-07-04 via /gsd-explore (6 轮) + 真机采样*
