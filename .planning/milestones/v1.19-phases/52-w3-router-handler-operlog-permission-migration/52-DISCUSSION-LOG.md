# Phase 52: W3 — Router/Handler/Operlog/Permission/Migration - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-07
**Phase:** 52-w3-router-handler-operlog-permission-migration
**Areas discussed:** 审计表 schema + before_value 捕获, 菜单 seed + 父菜单名实为端口状态, 路由组结构 /network/ports/write/*, INFRA-03 缓存策略 + BATCH-05 进度, audit↔operlog 关联

---

## 审计表 schema + before_value 捕获

### before_value 捕获方式

| Option | Description | Selected |
|--------|-------------|----------|
| handler 预 SELECT 快照 (JSONB) | handler 在调 service 前 SELECT,存 before_value JSONB;service 契约不变(D-15);command_sent/device_response TEXT | ✓ |
| service 返回 BeforeSnapshot | 改 Phase 51 PortResult 加字段;封装好但动已锁契约+28 测试 | |
| 不存 before/after (简化) | 只存 command_sent/device_response/status;失去人工回退依据 | |

**User's choice:** handler 预 SELECT 快照 (JSONB)
**Notes:** 保持 Phase 51 零侵入,D-15 service 不拥有审计原则成立。

### after_value 来源

| Option | Description | Selected |
|--------|-------------|----------|
| 目标态快照 (推荐) | handler 同步写入时填目标态;Enqueue 采集不回填 audit | ✓ |
| 采集后回填 | after_value 留空,collector 回填;需改 collector 加回调,跨 phase 边界 | |
| 去掉 after_value 列 | 只存 before_value;省列但失去点态对比 | |

**User's choice:** 目标态快照
**Notes:** 设备已接受命令 = 目标达成;Enqueue 只刷新 sys_device_port_status 实时表。

### audit 表索引

| Option | Description | Selected |
|--------|-------------|----------|
| 两索引 (推荐) | (device_id,port_id,created_at) 复合 + (created_at) 单列 | ✓ |
| 加 status 部分索引 | 额外 (status) WHERE status IN ('failed') 部分索引 | |
| 只一个复合索引 | YAGNI,只 (device_id,port_id,created_at) | |

**User's choice:** 两索引

### 批量端点 audit 写几行

| Option | Description | Selected |
|--------|-------------|----------|
| 每端口 1 条 audit (推荐) | batch N 个端口各 1 行;operlog 1 条汇总 OperTypeBatch | ✓ |
| 1 条汇总 audit | audit 粗粒度,失败定位靠 oper_param JSON | |
| 成功汇总+失败明细 | succeeded/failed 各 1 条,skipped 不写 | |

**User's choice:** 每端口 1 条 audit
**Notes:** 与 Phase 51 D-15 "batch 每端口写 1 条 audit" 一致。

---

## 菜单 seed + 父菜单名实为端口状态

### "端口配置" 菜单类型

| Option | Description | Selected |
|--------|-------------|----------|
| 按钮权限 F 型 (推荐) | menu_type='F',parent=端口状态,perms=network:port:write,path='write';不生路由,按钮 gating | ✓ |
| 路由菜单 C 型 | path='write',component=network/ports/write/index;UI-01 无独立页→404 | |
| 目录 M 型 | path='write',下挂子菜单;MVP 空目录无意义 | |

**User's choice:** 按钮权限 F 型
**Notes:** UI-01 写操作走现有列表页按钮+Modal/Drawer,F 型按钮权限最契合。

### 菜单授权落地

| Option | Description | Selected |
|--------|-------------|----------|
| 创建 helper (推荐) | 新建 menu_grant_helpers.go::GrantNewMenuToRolesHavingParent,migration_202 调用 | ✓ |
| 内联 SQL | migration_202 内联同逻辑;与 migration_195 风格一致 | |
| 不授权(超管旁路) | admin 走超管;违反 PERM-03 + memory 根治建议 | |

**User's choice:** 创建 helper
**Notes:** 一次性投入根治 antd 父子联动陷阱,memory `migration-grant-new-menu-precision-helper` 推荐路径。

**关键纠正**:ROADMAP 写父菜单"端口管理",实际 DB 是"端口状态"(path=network/ports, component=network/ports/index,见 archive/053_fix_menu_paths_unified.sql:185)。

---

## 路由组结构 /network/ports/write/*

### 6 端点挂载方式

| Option | Description | Selected |
|--------|-------------|----------|
| 子组 /write/* + 组级鉴权 (推荐) | ports.Group("/write") + 组级 RequirePermissions;6 端点 kebab | ✓ |
| 扁平 + 逐端点鉴权 | 直接挂 /network/ports/ 下;鉴权重置 6 次 | |
| 子组 + 逐端点鉴权 | 子组但鉴权逐端点;MVP 同权限冗余 | |

**User's choice:** 子组 /write/* + 组级鉴权
**Notes:** 端点:/shutdown /undo-shutdown /description /dot1x-enable /dot1x-disable /batch(kebab,与现有 /list /collect /batch-delete 同风格)。

---

## INFRA-03 缓存策略 + BATCH-05 进度

### batch 同步/异步 + 进度

| Option | Description | Selected |
|--------|-------------|----------|
| 同步 + 常量-only (推荐) | batch 端点同步阻塞;cache_keys.go 仅定义常量;BATCH-05 推 v1.19.x | ✓ |
| WebSocket 逐端口推 | handler 拆开 BatchWritePorts 自己 loop + ws 推;BATCH-05 本 phase 满足但重复 batch 逻辑 | |
| 改 Phase 51 加进度回调 | BatchWritePorts 接受 batch_id+回调;动已锁契约+28 测试 | |

**User's choice:** 同步 + 常量-only
**Notes:** BATCH-05 进度反馈推到 v1.19.x(需 WebSocket/SSE 重构,动 Phase 51 契约)。

---

## audit↔operlog 关联(补聊)

### audit.oper_log_id FK

| Option | Description | Selected |
|--------|-------------|----------|
| 加 oper_log_id FK (推荐) | audit.oper_log_id→sys_oper_log.id;UI-04 精准跳 | ✓ |
| 无 FK 模糊匹配 | 靠 (operator, created_at±2s, module) 模糊关联;并发可能错配 | |
| audit 独立不跳 operlog | UI-04 改跳 audit 详情页(需 Phase 53 另建) | |

**User's choice:** 加 oper_log_id FK
**Notes:** 关键发现:operlog.Record 是 async(RecordAsync),不返回 oper_id;但 OperLog.ID 在 BeforeCreate 中若预设则保留(BaseTimeLine 钩子)。推荐机制:新增 operlog.WithOperID RecordOption,handler 预生成 uuid 同步写 audit.oper_log_id。兜底:audit 先写 + WithOperParam(audit_ids) 嵌 operlog。oper_log_id 列可空(防 operlog 失败时 audit 仍落库)。

---

## Claude's Discretion

- PortWriteAudit model GORM tag 细节(按 migration-sql-name-must-match-model memory 推导)
- handler 取 operator 用 utils.GetUsername(c)(CLAUDE.md 惯例)
- audit 表不加 device_id/port_id 之外 FK(软关联,query JOIN)
- migration_202 用手写 SQL(精确控 JSONB+索引)+ model 加 `gorm:"-:migration"` 防双重 ALTER
- 批量 N 条 audit 每条独立 INSERT(不用大事务)
- 单端口 request body struct 形状(PortID + Description + Reason)

## Deferred Ideas

- BATCH-05 批量进度反馈(WebSocket/SSE)— v1.19.x
- sys_port_write_audit 详情查看 UI — v1.19.x+
- audit 表 TTL/归档策略 — v1.19.x+
- 跨设备批量 — Phase 51 D-17 已锁 ErrMixedDevices 拒绝
- 写命令前设备可达性预检(FUTURE-07)— v1.19.x+
- operlog WithOperID 是否同步进 RecordBackground — 本 phase 仅 Record 需要
