# 资产对账（多源数据 Reconciliation）架构决策记录 v0.3

**记录日期**: 2026-06-27
**记录人**: gsd:explore session
**版本演进**: v0.1（基础架构） → v0.2（IP 段例外 + 工位整合） → v0.3（菜单归属资产管理）
**目标里程碑**: v1.17（待启动）
**关联上下文**: v1.16 技术债清理 100% 完成（11/11 plans，5/5 code gaps fixed）

---

## 1. 业务背景与动机

XingRan-Next 运维管理系统已具备多路数据来源对资产-人员进行描述，但缺少对账机制：

| 数据源 | 类型 | 权威性 | 已知问题 |
|--------|------|--------|----------|
| 网络设备端口 → MAC | 物理层（ground truth） | ⭐⭐⭐⭐⭐ | 端口采集覆盖不全、信息点断接未登记 |
| 系统责任人（sys_asset.responsible_user_id） | 业务层（声明） | ⭐⭐⭐ | 离职未移交、工单遗漏 |
| AD 域控 managed_by | 外部层 | ⭐⭐ | code 32 / admin lockout / FailoverClient MarkFailure 双计数（项目记忆） |

**核心矛盾**：三路数据不一致 → 脏数据累积 → 审计失败 → 资产账实不符。

**用户决策**：
- ✅ 采用 **Observe-only 策略**（仅观测不修改）
- ✅ 采用 **告警驱动人工修复** 闭环
- ✅ 例外机制：**IP 段级别** + **多 actions 组合**
- ✅ 前端整合入口：**工位详情页**（已有的"工位 → 子表"结构）
- ✅ 菜单归属：**资产管理 / 数据质量** 子菜单组

---

## 2. 核心架构：5 层数据流

```
Layer 1: Raw Data Sources（已有，零改动）
   ↓
Layer 2: Normalization（reconciliation_normalized 物化视图，5min 增量刷新）
   ↓
Layer 3: Conflict Detection（引擎分类 Type A~F）
   ↓
Layer 3.5: Exception Filter（IP 段 + 冲突类型 + 范围三维过滤）🆕
   ↓
Layer 4: Reconciliation Report（sys_data_reconciliation，审计可追溯）
   ↓
Layer 5: Alerting & Workorder Closure（受 exception 影响）
```

---

## 3. 三路对账信任评估

### 3.1 冲突类型矩阵

| 冲突类型 | 物理链路 | 责任人 | AD | 业务优先级 |
|---------|---------|-------|-----|----------|
| **A**：物理有 / 责任人有且一致 | ✅ | ✅ | ❓ | 健康，无需动作 |
| **B**：物理有 / 责任人无 | ✅ | ❌ | ❓ | ⚠️ 中危 |
| **C**：物理有 / 责任人不一致 | ✅ | ⚠️ | ❓ | 🔴 **高危** |
| **D**：物理无 / 责任人有 | ❌ | ✅ | ❓ | 🟡 中危 |
| **E**：三路均无 | ❌ | ❌ | ❌ | ⚪ 低危（疑似幽灵资产） |
| **F**：AD managed_by 单独不一致 | 任意 | 任意 | ⚠️ | 🟡 中危（AD 已知不可靠） |

### 3.2 业务优先级排序

`C > B > D > F > E > A`

---

## 4. IP 段级别例外规则体系（v0.2 新增）

### 4.1 设计动机

观察模式必须解决**告警疲劳**问题（参考项目记忆 `ad-operation-prefix-failover-source` AD 故障风暴教训），否则短期就会被噪声淹没。

典型降噪场景：
- 办公网段（192.168.0.0/16）：人员出差频繁，Type D 告警意义不大
- 测试网段（10.99.0.0/16）：设备生命周期短，Type E 幽灵资产告警无意义
- VPN 段（10.8.0.0/16）：远程用户无固定工位，Type C/D 几乎全是误报
- DMZ 段（172.16.0.0/16）：服务器无人员归属，Type B/D 全是误报

### 4.2 例外 Actions 语义

| Action | 语义 |
|--------|------|
| `no_alert` | 不产生告警（WebSocket / 邮件 / 钉钉全静默） |
| `no_notice` | 不写入 SysNotice（系统通知中心不显示，但仍记录 reconciliation 表） |
| `no_workorder` | 不自动开工单（reconciliation 状态保留为 open，不进 workorder 模块） |
| `skip_severity` | 跳过当前告警级别（仍记录但不升级） |
| `silence` | 全静默（最强，不记录、不告警、不开工单，仅审计可查） |

**多条规则取并集**：如规则 1 `no_alert` + 规则 2 `no_notice` → 全静默。

### 4.3 关键原则

- 即使匹配例外，**仍写入 `sys_data_reconciliation`**，并标记 `exception_rule_id` + `applied_actions` → 审计要求
- 例外**不影响 confidence_score**（评分按真实数据计算）
- 例外**不影响 raw_snapshot 完整记录**（事后溯源无歧义）

### 4.4 IP 归属解析顺序

```
asset.ip (自身) → workstation.ip → network_device.ip (via port) → unknown
```

**注意点**：若 `sys_asset` 无 `ip` 列，需加字段（migration）。

---

## 5. 数据库 Schema 增量

### 5.1 主表：`sys_data_reconciliation`

```sql
CREATE TABLE sys_data_reconciliation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL,
    conflict_type VARCHAR(2) NOT NULL,            -- A~F
    severity VARCHAR(16) NOT NULL,                 -- low/medium/high/critical
    physical_value JSONB,
    declared_value JSONB,
    ad_value JSONB,
    confidence_score DECIMAL(3,2),
    raw_snapshot JSONB NOT NULL,                  -- 完整冻结快照
    asset_ip INET,                                 -- 命中解析时的 IP
    exception_rule_id UUID REFERENCES sys_reconciliation_exception(id),  -- 🆕
    applied_actions TEXT[],                        -- 🆕 例外应用的 actions
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by BIGINT,
    resolution_note TEXT,
    workorder_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX uniq_recon_asset_type_open
    ON sys_data_reconciliation(asset_id, conflict_type)
    WHERE resolved_at IS NULL AND deleted_at IS NULL;
-- ↑ 唯一索引：同一资产同一冲突类型只能有一条未解决记录（防告警风暴）
```

### 5.2 例外表：`sys_reconciliation_exception` 🆕

```sql
CREATE TABLE sys_reconciliation_exception (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(128) NOT NULL,
    ip_range CIDR NOT NULL,
    conflict_types TEXT[] NOT NULL DEFAULT '{}',
    exception_actions TEXT[] NOT NULL,
    severity_override VARCHAR(16),
    scope_type VARCHAR(16) NOT NULL DEFAULT 'global',  -- 'global' / 'building' / 'floor'
    scope_id UUID,
    reason TEXT NOT NULL,                           -- 审计要求, ≥10 字符
    is_active BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMPTZ,                          -- 可选, 自动过期
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_actions CHECK (
        exception_actions <@ ARRAY['no_alert','no_notice','no_workorder','skip_severity','silence']
    ),
    CONSTRAINT chk_severity_override CHECK (
        severity_override IS NULL OR severity_override IN ('low','medium','high')
    )
);

CREATE INDEX idx_exc_active_range
    ON sys_reconciliation_exception USING gist (ip_range)
    WHERE is_active = true AND deleted_at IS NULL;
-- ↑ GiST 索引加速 PostgreSQL 内置 CIDR 包含查询
```

### 5.3 物化视图：`reconciliation_normalized`

5 分钟增量刷新；监听 `sys_port_mac` 变更触发。

```sql
CREATE MATERIALIZED VIEW reconciliation_normalized AS
SELECT
    a.id AS asset_id,
    a.code AS asset_code,
    a.mac AS asset_mac,
    a.ip AS asset_ip,
    a.responsible_user_id,
    u.username AS responsible_username,
    pm.port_id, pm.mac AS observed_mac, pm.observed_at,
    ip.id AS info_point_id, w.id AS workstation_id,
    w.user_id AS physical_user_id, wu.username AS physical_username,
    ad.managed_by_dn, ad.resolved_user_id AS ad_user_id
FROM sys_asset a
LEFT JOIN sys_user u ON u.id = a.responsible_user_id
LEFT JOIN sys_port_mac pm ON pm.mac = a.mac AND pm.deleted_at IS NULL
LEFT JOIN sys_info_point ip ON ip.port_id = pm.port_id AND ip.deleted_at IS NULL
LEFT JOIN sys_workstation_info_point wip ON wip.info_point_id = ip.id
LEFT JOIN sys_workstation w ON w.id = wip.workstation_id AND w.deleted_at IS NULL
LEFT JOIN sys_user wu ON wu.id = w.user_id
LEFT JOIN sys_user_ad_attrs ad ON ad.user_id = COALESCE(w.user_id, a.responsible_user_id)
WHERE a.deleted_at IS NULL;

CREATE UNIQUE INDEX idx_recon_norm_asset ON reconciliation_normalized(asset_id);
```

---

## 6. 工位详情页前端整合（v0.2 新增）

### 6.1 整合点：现有"工位 → 子表"结构

```
工位行 (sys_workstation)
├── 子表 1：域控设备（AD devices）→ 加"对账健康"列
└── 子表 2：资产设备（asset devices）→ 加"对账健康"列
```

### 6.2 顶部健康度卡片

```
┌──────────────────────────────────────────────────────────┐
│  📊 对账健康度 [本周]   得分: 78/100  [详情报告] [申请例外]│
│  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌────────────────────┐  │
│  │ 5正常│ │1漂移│ │0冲突│ │2无数据│ │⚙ 例外规则: 1 条命中│  │
│  └─────┘ └─────┘ └─────┘ └─────┘ └────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

### 6.3 行内徽标 + 抽屉详情

点击任意徽标 → 抽屉显示冲突摘要 / 历史变更 / 例外规则命中。

### 6.4 关键组件

```
src/components/reconciliation/
├── HealthCard.tsx              // 顶部健康度聚合卡片
├── HealthBadge.tsx             // 通用徽标
├── ReconciliationDrawer.tsx    // 抽屉详情
├── ReconciliationTimeline.tsx  // 历史变更时间线
├── ExceptionMatchList.tsx      // 例外规则命中列表
└── hooks/
    ├── useWorkstationHealth.ts
    ├── useAssetHealth.ts
    └── useExceptionMatch.ts
```

### 6.5 数据获取

- 后端聚合 API：`GET /asset/reconciliation/by-workstation/:ws_id`（一次查询返回工位下所有资产的对账状态）
- 缓存：工位详情页打开时拉一次，WS 推送增量更新
- N+1 风险规避：必须后端聚合，禁止前端循环单查

---

## 7. 菜单归属与权限（v0.3 调整）

### 7.1 菜单结构（资产管理下）

```
资产管理
├── 资产卡片 / 资产分类 / 资产入库 / ...（existing）
└── ─── 数据质量（分组）─── 🆕
    ├── 资产对账（父路由）     /asset/reconciliation
    │   ├── 对账看板           /asset/reconciliation/dashboard
    │   ├── 异常列表           /asset/reconciliation/exceptions
    │   └── 例外规则           /asset/reconciliation/exception-rules
```

### 7.2 权限命名空间

| 资源 | 权限 |
|------|------|
| 资产对账-查看 | `asset:reconciliation:list` |
| 资产对账-导出 | `asset:reconciliation:export` |
| 资产对账-看板 | `asset:reconciliation:dashboard` |
| 资产对账-例外查看 | `asset:reconciliation:exception:list` |
| 资产对账-例外创建 | `asset:reconciliation:exception:create` |
| 资产对账-例外更新 | `asset:reconciliation:exception:update` |
| 资产对账-例外删除 | `asset:reconciliation:exception:delete` |
| 资产对账-例外测试 | `asset:reconciliation:exception:test` |

### 7.3 API 路由前缀

`/asset/reconciliation/*`（从 `/ops/reconciliation/*` 调整为 `/asset/reconciliation/*`）

### 7.4 跨模块依赖：工位详情页读取对账状态

```
工位详情页 (ops/workstation)
   ├─ 数据 1: ops 模块（工位、用户）→ /ops/workstation/getInfo
   └─ 数据 2: asset 模块（对账状态）→ /asset/reconciliation/by-workstation/:ws_id
        ↑ 跨模块 service 层直接调用（非 HTTP）
        ↑ 权限降级：无 `asset:reconciliation:list` 时健康度卡片隐藏
```

**避免 403 的关键**（参考项目记忆 `xingran-perm-namespace-split-readonly-page`）：
- 菜单 perms 与 CRUD perms 不重叠会 403
- 工位详情页是查询路径，用 `RequirePermissionsWithQuery` 对查询放宽读权限
- 显式权限边界：用户在 ops 页需具备 `asset:reconciliation:list` 才显示健康度

### 7.5 菜单 Seed 注意事项

⚠️ 参考项目记忆：
- `migration-sql-name-must-match-model`：菜单 SQL 必须匹配 model 字段名
- `xingran-migrations-no-sql-autoloader`：必须用 `migration_NNN_*.go` 函数显式调用 + 加入 `AutoMigrate()`
- `xingran-gorm-sql-constraint-naming-conflict`：GORM `uniqueIndex` 期望 `uni_*_*`，SQL inline UNIQUE 用 PG 自动名 `*_key`

---

## 8. 后端模块结构

```
internal/
├── api/v1/
│   ├── operations/
│   │   └── workstation_handler.go  ← 修改：注入 ReconciliationService
│   └── asset/                      ← 🆕 新增模块
│       ├── reconciliation_handler.go
│       ├── reconciliation_exception_handler.go
│       └── router.go
├── services/
│   └── asset/                      ← 🆕 新增模块
│       ├── reconciliation_service.go
│       ├── reconciliation_detection.go      (Layer 3 引擎)
│       ├── reconciliation_exception.go      (Layer 3.5 例外过滤)
│       └── reconciliation_snapshot.go       (物化视图 ETL)
└── models/
    └── reconciliation.go            ← 🆕 新增模型（3 张表）
```

---

## 9. 前端文件结构

```
src/
├── pages/
│   ├── operations/workstation/    ← 修改：注入对账徽标
│   └── asset/                     ← 🆕 新增模块
│       └── reconciliation/
│           ├── index.tsx                  (父路由 - 资产对账)
│           ├── dashboard/index.tsx        (对账看板)
│           ├── exceptions/index.tsx       (异常列表)
│           └── exception-rules/index.tsx  (例外规则 CRUD)
├── components/
│   └── reconciliation/            ← 🆕 共享组件
└── lib/
    └── assetApi.ts                ← 🆕 参考 opsApi.ts 模式
```

---

## 10. 路线图（v0.3 调整）

| 阶段 | 周期 | 交付物 | UAT 标准 |
|------|------|--------|---------|
| **R1：观测底座** | 2-3 周 | 物化视图 + reconciliation 表 + dashboard（不含例外） | 能看到所有异常类型分布，零误报 |
| **R2：告警 + 工单闭环** | 2-3 周 | critical/high 自动转 workorder + 通知 | critical 异常从检出到工单创建 ≤2min |
| **R3：置信度评分 + 降噪** | 1-2 周 | + IP 段例外引擎 + 例外管理页 + 命中测试 | 告警量比 R2 末期下降 ≥60% |
| **R4：工位详情整合** | 1-2 周 | + 工位页健康度卡片 + 行内徽标 + 抽屉 | 跨模块调用性能 ≤200ms |
| **R5（可选）：半自动修复** | 2-4 周 | 高置信度建议修复（仍需人工确认） | 误修复率 < 1%，可一键回滚 |

---

## 11. 关键风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| 端口采集覆盖不全 | 物理链路查不到被误判为 Type D | R1 阶段先跑覆盖率报告，< 80% 不上线 R2 |
| AD managed_by 历史脏数据 | 早期大量 Type F 告警 | R3 阶段加 AD 数据清洗 cron（参考 `ad-update-no-such-object-vs-lockout`） |
| 信息点未接线登记 | 物理链路断链 | R1 阶段联动信息点治理 |
| 工单生成过多淹没运维 | 告警疲劳 | R3 节流 + R4 半自动前置 |
| reconciliation_normalized 性能 | 大表 join 慢 | 物化视图 + 增量刷新 |
| 现有 Phase 13 技术债回归 | 新代码破坏已完成清理 | 引入 constraint 命名规范 + operlog 强制约束 |
| 菜单权限命名空间割裂 | 读写全 403（参考 `xingran-perm-namespace-split-readonly-page`） | 显式声明跨模块权限边界 + 用 `RequirePermissionsWithQuery` |
| 修复回写触发循环对账 | 异常立即重新生成 | 修复后 7d 静默期 + 异步通知引擎 |

---

## 12. 已知项目记忆应用清单

| 记忆 | 应用点 |
|------|--------|
| `workstation-ad-device-managedby-vs-description` | AD 反查不走 managed_by，用 MAC + 信息点 + 工位链路 |
| `ad-update-no-such-object-vs-lockout` | 例外引擎遇到 DN 不存在时显式标记，不能静默吃下 |
| `ad-operation-prefix-failover-source` | 例外规则设计借鉴 AD 故障风暴教训，必须节流 |
| `ad-modify-fail-double-counts-breaker` | 修复回写不能双计数熔断 |
| `xingran-perm-namespace-split-readonly-page` | 跨模块调用必须显式权限声明 |
| `ops 菜单 seed perms 与路由命名不一致` | 菜单 seed 用 `asset:reconciliation:exception:*` 单数连字符，路由对应 |
| `migration-sql-name-must-match-model` | menu/migration 字段名匹配 model |
| `xingran-migrations-no-sql-autoloader` | 必须 .go migration + AutoMigrate 显式调用 |
| `xingran-gorm-sql-constraint-naming-conflict` | uniqueIndex 命名 `uni_*_*` 避免冲突 |
| `Excel 导入路由冲突陷阱` | router.go 不能预注册 asset/reconciliation/* 路径（避免与 ops 冲突） |
| `stat-cards-from-list-length-capped-at-100` | dashboard 健康度数字必须用 COUNT 端点，不用 list.length |
| `GORM migration tag 不阻止 INSERT` | 新表字段必须有正确的 gorm tag |

---

## 13. 决策点追踪（18 项）

### v0.1 锁定（1 项）
1. ✅ Observe-only + 告警驱动人工修复

### v0.2 锁定（5 项）
2. ✅ IP 段匹配用 CIDR
3. ✅ 例外作用范围三维：IP 段 + 冲突类型 + 范围（global/building/floor）
4. ✅ 命中例外仍记录 reconciliation 表 + 标记 exception_rule_id
5. ✅ 多条规则 actions 取并集
6. ✅ 支持临时例外（expires_at）

### v0.3 锁定（5 项）
7. ✅ reconciliation 模块归属资产管理
8. ✅ 菜单层级：资产对账（一级）+ 3 个二级
9. ✅ 权限命名空间 `asset:reconciliation:*`
10. ✅ API 路由前缀 `/asset/reconciliation/*`
11. ✅ 跨模块调用走 service 层 + 权限降级

### 待 R1 启动时细化（7 项）
12. ⏳ 物化视图刷新频率（建议 5 分钟，参照 AD sync cron）
13. ⏳ 告警分发范围（先 WebSocket + SysNotice，钉钉/邮件下个 phase）
14. ⏳ R5 半自动修复阈值（建议 confidence ≥0.9）
15. ⏳ dashboard 是否需要独立页面（建议作为二级菜单）
16. ⏳ operlog 记录字段（已确认用 Record / RecordWithBody）
17. ⏳ 是否需要双签流程（建议不强制，操作日志足够）
18. ⏳ 临时例外默认有效期（建议 30 天，可改）

---

## 14. 启动建议

### 14.1 推荐流程

```
当前（本 Note 已落地）
   ↓
[可选] 启动 /gsd-discuss-phase 收集 R1 详细需求
   ↓
[可选] 启动 /gsd-plan-phase 拆分 R1 为 4-6 个 plan
   ↓
[可选] 启动 /gsd-execute-phase 执行 R1
```

### 14.2 R1 首个 plan 候选（最小可验证）

**目标**：物化视图 + 后端查询 API + admin 异常列表（只读）

**任务**：
1. 创建 `internal/services/asset/` 目录与 4 个 service 文件骨架
2. 创建 `sys_data_reconciliation` + `sys_reconciliation_exception` + `reconciliation_normalized` migration
3. 实现 `GET /asset/reconciliation/exceptions`（分页 + 筛选）
4. 前端 `src/pages/asset/reconciliation/exceptions/` 列表页（只读）
5. 跑覆盖率报告，验证物化视图能正确产出 Type A~F

**UAT 标准**：
- 列表能展示所有异常类型分布
- 分页正确（不在 MaxPageSize=100 钳制下）
- 权限生效（无权限 403）

---

## 15. 复用审计 (v0.4)

**审计日期**: 2026-06-27
**审计方法**: 对照代码实现 + 命名约定 + 项目记忆清单，逐项打勾
**完整审计报告**: 详见 `.planning/notes/260627-reconciliation-reuse-audit.md`

### 15.1 审计总览

| 类别 | 状态 | 严重度 |
|------|------|--------|
| ✅ 已正确复用 | 13 项 | — |
| ⚠️ 部分复用 / 需补充 | 4 项 | 中 |
| ❌ 未复用 / 需新增 | 7 项 | 高 |
| **复用完整度** | **65%** | — |

### 15.2 必须补齐的 7 项 (R1 启动前)

| # | 项 | 优先级 | 触发阶段 |
|---|----|-------|---------|
| F1 | **Statistics 专用 COUNT 端点** | 🔴 高 | R1 schema 设计时同步定义 |
| F2 | **路由注册位置 + 冲突规避** | 🔴 高 | R1 router.go 设计时规划 |
| F3 | **数据字典 seed (4 个字典)** | 🔴 高 | R1 migration 定义 |
| F4 | **参数管理 seed (8 个 config 项)** | 🟡 中 | R1 启动时定义 |
| F5 | **Cache Key helper 函数** | 🟡 中 | R1 service 层第一行 |
| F6 | **前端 queryKeys 注册** | 🟡 中 | 前端第一行代码 |
| F7 | **operlog module 常量 + Cron 注册** | 🟢 低 | R1/R2 实施时 |

### 15.3 已复用的 13 项 (✅)

1. ✅ **数据字典后端基础设施** — `internal/models/dict.go` + `dict_cache_impl.go`，命名 `<module>_<resource>_<field>`（如 `ops_dedicated_line_type`）
2. ✅ **参数管理基础设施** — `sys_config` + `ConfigService.GetByKey()` 模式
3. ✅ **操作日志体系** — Phase 34 全模块覆盖，`operlog.Record` / `RecordWithBody` 强制约定
4. ✅ **通用 CRUD 模式** — `SetupXxxRouter` + Handler-Service + Cache 分支
5. ✅ **双缓存架构** — `CacheProvider` 接口 + `NewCacheAdapter` 模式
6. ✅ **响应式分页** — Phase A 基建 (`BaseListRequest` + `ApplySort` 白名单)
7. ✅ **Excel 导入导出** — `SetupExcelRouter` + `ExcelConfig` map + `excelEntityModuleNames` 映射
8. ✅ **UUID + 软删除 + BaseModel** — `internal/models/base.go` 自动生成 UUID
9. ✅ **Status Value Convention** — 0=启用 1=停用（无单独 status 字段，用 `resolved_at IS NULL`）
10. ✅ **前端 useDict hook** — `useDict(dictType)` + `useInvalidateDict()` + `queryKeys.dict`
11. ✅ **前端 opsApi.ts 工厂模式** — 已有 buildingApi/floorApi/.../excelApi 模式可复用
12. ✅ **ECharts 6** — 按项目记忆 `echarts6-customchart-tree-shaking-noop` 不追求按需引入
13. ✅ **响应式 UI 库** — Ant Design 6.1 + Tailwind CSS 4.1（不引入新库）

### 15.4 部分复用的 4 项 (⚠️ 需补充)

| # | 项 | 补充内容 | 触发阶段 |
|---|----|---------|---------|
| P1 | **数据字典枚举值定义** | 4 个字典完整 seed（见 §15.5） | R1 migration |
| P2 | **参数管理配置项** | 8 个 config_key seed（见 §15.6） | R1 启动 |
| P3 | **Excel 导入配置** | `ExcelConfigs` map 新增 2 个 entityType | R3 实施 |
| P4 | **operlog 中文模块名** | 4 个 module 常量映射 | R1 实施 |

### 15.5 数据字典 seed (P1 补充)

#### `asset_reconciliation_conflict_type`

| dict_label | dict_value | list_class | remark |
|------------|------------|------------|--------|
| 物理有/责任人有且一致 | A | success | 健康，无需动作 |
| 物理有/责任人无 | B | warning | 中危：资产无主 |
| 物理有/责任人不一致 | C | error | **高危**：实际 ≠ 记录 |
| 物理无/责任人有 | D | warning | 中危：物理未上线 |
| 三路均无 | E | default | 低危：幽灵资产 |
| AD 单独不一致 | F | processing | 中危：AD 不可靠 |

#### `asset_reconciliation_severity`

| dict_label | dict_value | list_class |
|------------|------------|------------|
| 低 | low | default |
| 中 | medium | warning |
| 高 | high | error |
| 严重 | critical | error |

#### `asset_reconciliation_exception_action`

| dict_label | dict_value | list_class |
|------------|------------|------------|
| 不告警 | no_alert | default |
| 不通知 | no_notice | default |
| 不开工单 | no_workorder | default |
| 降低严重级 | skip_severity | warning |
| 全静默 | silence | default |

#### `asset_reconciliation_status`

| dict_label | dict_value | list_class |
|------------|------------|------------|
| 未解决 | open | warning |
| 已解决 | resolved | success |

**migration 文件位置**：`internal/core/db/migrations/migration_NNN_reconciliation_dicts.go`
**关键约束**：必须走 `.go` migration + AutoMigrate 显式调用（参考 `xingran-migrations-no-sql-autoloader`）

### 15.6 参数管理 seed (P2 补充)

| config_key | config_value | config_type | remark |
|------------|--------------|-------------|--------|
| `asset.reconciliation.view.refresh_interval` | `5m` | Y | 物化视图刷新间隔 |
| `asset.reconciliation.score.physical` | `0.5` | Y | 物理链路命中加分 |
| `asset.reconciliation.score.declared` | `0.3` | Y | 系统责任人命中加分 |
| `asset.reconciliation.score.ad` | `0.2` | Y | AD 命中加分 |
| `asset.reconciliation.exception.default_expiry_days` | `30` | Y | 临时例外默认有效期（天） |
| `asset.reconciliation.alert.critical_threshold` | `5` | Y | critical 异常触发即时通知阈值 |
| `asset.reconciliation.alert.silence_after_resolved_hours` | `168` | Y | 修复后静默期（7d = 168h） |
| `asset.reconciliation.health.score_weights` | `{normal:1.0, drift:0.5, conflict:0.0, nodata:0.7}` | Y | 健康度评分权重 |

**关键原则**：
- ✅ `config_type='Y'` 表示用户可改，运行时可热更新
- ✅ `is_system=1` 的不可删（系统内置）
- ✅ 通过 `system.ConfigService.GetByKey(ctx, key)` 查询

### 15.7 Cache Key helper (F5 补充)

**新建文件**：`internal/services/asset/cache_keys.go`

```go
package asset

import "fmt"

const (
    CacheKeyReconciliationDashboard         = "reconciliation:dashboard:%s"
    CacheKeyReconciliationExceptionList     = "reconciliation:exception:list:%s"
    CacheKeyReconciliationExceptionByID     = "reconciliation:exception:byID:%s"
    CacheKeyReconciliationExceptionRuleList = "reconciliation:exceptionRule:list"
    CacheKeyReconciliationExceptionRuleByID = "reconciliation:exceptionRule:byID:%s"
    CacheKeyReconciliationViewLastRefresh   = "reconciliation:view:lastRefresh"
    CacheKeyReconciliationHealthByWorkstation = "reconciliation:health:workstation:%s"
    CacheKeyReconciliationHealthByAsset     = "reconciliation:health:asset:%s"
)

func GetReconciliationDashboardKey(scope string) string {
    return fmt.Sprintf(CacheKeyReconciliationDashboard, scope)
}
// ... 完整 helper 函数
```

**复用参考**：`internal/services/cache_keys.go` 现有 `GetDictDataByTypeKey(dictType)` 模式。

### 15.8 前端 queryKeys 注册 (F6 补充)

**修改文件**：`src/lib/queryKeys.ts`

```typescript
export const queryKeys = {
  // ... 已有
  reconciliation: {
    all: ['reconciliation'] as const,
    dashboard: (filters: DashboardFilters) => 
      [...queryKeys.reconciliation.all, 'dashboard', filters] as const,
    exceptionList: (params: ExceptionListParams) => 
      [...queryKeys.reconciliation.all, 'exceptions', params] as const,
    exceptionDetail: (id: string) => 
      [...queryKeys.reconciliation.all, 'exception', id] as const,
    ruleList: () => 
      [...queryKeys.reconciliation.all, 'rules'] as const,
    ruleDetail: (id: string) => 
      [...queryKeys.reconciliation.all, 'rule', id] as const,
    workstationHealth: (workstationId: string) => 
      [...queryKeys.reconciliation.all, 'workstationHealth', workstationId] as const,
    assetHealth: (assetId: string) => 
      [...queryKeys.reconciliation.all, 'assetHealth', assetId] as const,
    matchTest: (ip: string) => 
      [...queryKeys.reconciliation.all, 'matchTest', ip] as const,
  },
};
```

### 15.9 Statistics 专用端点 (F1 补充)

**新建文件**：`internal/services/asset/reconciliation_statistics.go`

| 端点 | 用途 |
|------|------|
| `POST /asset/reconciliation/statistics/summary` | 总览 KPI（5 张卡片） |
| `POST /asset/reconciliation/statistics/by-conflict-type` | 冲突类型分布（饼图） |
| `POST /asset/reconciliation/statistics/by-severity` | 严重级别分布（柱状图） |
| `POST /asset/reconciliation/statistics/health-trend` | 健康度趋势（折线图） |
| `POST /asset/reconciliation/statistics/top-unresolved` | Top 10 长期未解决 |
| `POST /asset/reconciliation/statistics/exception-rule-stats` | 例外规则生效统计 |

```go
type ReconciliationStatistics interface {
    Summary(ctx context.Context, filters StatsFilter) (*SummaryResult, error)
    ByConflictType(ctx context.Context, filters StatsFilter) (map[string]int64, error)
    BySeverity(ctx context.Context, filters StatsFilter) (map[string]int64, error)
    HealthTrend(ctx context.Context, filters StatsFilter) ([]TrendPoint, error)
    TopUnresolved(ctx context.Context, limit int) ([]ExceptionSummary, error)
    ExceptionRuleStats(ctx context.Context) ([]RuleStats, error)
}
```

**关键约束**：**严禁** 用 `list.length`，必须用 `SELECT COUNT(*)` 或聚合查询（参考 `stat-cards-from-list-length-capped-at-100`）。

### 15.10 路由注册 (F2 补充)

**修改文件**：`internal/api/router.go`

```go
// 在 asset 模块注册分组下添加：
assetGroup := r.Group("/asset")
asset.SetupAssetRouter(assetGroup, core)
asset.SetupReconciliationRouter(assetGroup, core)            // 🆕
asset.SetupReconciliationExceptionRouter(assetGroup, core)  // 🆕
```

**关键约束**：
- ⚠️ 参考 `xingran-excel-import-route-conflict`：router.go 不能预注册 `/asset/reconciliation/*` 通用路由（避免与具体 handler 冲突）
- ⚠️ `excel_handler.SetupExcelRouter` 不能预注册 `reconciliationException` entityType（避免冲突）

### 15.11 operlog module 常量 (F7 补充)

**新建常量**：`internal/api/v1/asset/reconciliation_handler.go`

```go
const (
    ModuleReconciliationException     = "资产对账"
    ModuleReconciliationExceptionRule = "资产对账-例外规则"
    ModuleReconciliationAutoWorkorder = "资产对账-自动转工单"
    ModuleReconciliationReportExport  = "资产对账-报告导出"
)
```

### 15.12 Cron 注册 (F7 补充)

**新建文件**：`internal/scheduler/reconciliation_tasks.go`

| Cron 表达式 | 用途 |
|------------|------|
| `@every 5m` | 物化视图刷新 |
| `@every 10m` | 异常检测 |
| `0 2 * * *` | 静默期到期重检测（每日 02:00） |
| `0 3 * * *` | 临时例外规则清理（每日 03:00） |

**复用现有模式**：参考 `internal/scheduler/ad_sync_tasks.go` 的 cron 注册 + 错误日志模式。

### 15.13 健康度评分函数 (F7 补充)

**新建文件**：`internal/services/asset/reconciliation_health.go`

```go
type HealthScoreCalculator interface {
    Score(ctx context.Context, workstationID string) (*HealthScore, error)
    BatchScore(ctx context.Context, workstationIDs []string) (map[string]*HealthScore, error)
}

type HealthScore struct {
    Total        int          `json:"total"`
    Normal       int          `json:"normal"`
    Drift        int          `json:"drift"`
    Conflict     int          `json:"conflict"`
    NoData       int          `json:"noData"`
    ExceptionHit int          `json:"exceptionHit"`
    Score        float64      `json:"score"`
    Trend        []TrendPoint `json:"trend"`
}
```

---

## 16. 变更日志

| 版本 | 日期 | 变更 |
|------|------|------|
| v0.1 | 2026-06-27 | 基础架构（5 层数据流 + 6 类冲突 + R1-R4 路线） |
| v0.2 | 2026-06-27 | IP 段例外规则 + 工位详情前端整合 |
| v0.3 | 2026-06-27 | 菜单统一归属资产管理 + 权限命名 + 跨模块架构 |
| v0.4 | 2026-06-27 | 复用审计：13 已复用 + 4 部分复用 + 7 必须补齐；新增 §15 完整审计章节 |