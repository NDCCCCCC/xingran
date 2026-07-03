# Phase 47: 修复資產對賬系統根因 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-03
**Phase:** 47-phase-46-infopoint-layer3-port-status-parser
**Areas discussed:** R2/R3 UPSERT 语义, R5 MAC 校验 + 数据清理, R1 infopoint 漂移 (已闭合), R4 port_status 漂移 (已闭合), R1/R4 复盖确认, partial uniqueIndex 选择, 清理载体

---

## Area 1: R2/R3 — Layer 3 检测改为 UPSERT 的语义

| Option | Description | Selected |
|--------|-------------|----------|
| UPSERT 升级语义 (D-12) | ON CONFLICT DO UPDATE SET 全字段 + detected_at=NOW();允许 severity/类型升级 | ✓ |
| 只升级 severity/signals，不重置 detected_at | severity/raw_snapshot/三路 evidence + created_at 保留；语义更像人工运维更新 | |
| 仅在 conflict_type 真改变时 UPDATE | 仅 type D→C 时 UPSERT；同 type 下仍 skip；改动最小 | |

**User's choice:** UPSERT 升级语义 (D-12)
**Notes:** R2/R3 spec 中已锁 raw_snapshot/signal 三路 evidence 都升级；resolved_at 不变（部分索引约束下不重写已解决行）。

---

## Area 2: R5 — parseRuijiePortSecurityLine 加 MAC 校验后清理

| Option | Description | Selected |
|--------|-------------|----------|
| 代码层校验 + 数据清理脚本 | isCanonicalMAC + 一次性 migration 删除脏行 + mac_history 保留 | ✓ |
| 只修代码不清理历史 | 加校验即可；已有脏数据保留为历史 | |
| 代码层 + DB TRIGGER | (被用户偏好拒绝, 见 `user-prefers-code-fixes-no-db-triggers`) | |

**User's choice:** 代码层校验 + 数据清理脚本
**Notes:** mac_history 链保留（AUDIT-02），仅 sys_mac_address 软删除。

---

## Area 3: R1 — infoPoint.port_id 漂移根因

| Option | Description | Selected |
|--------|-------------|----------|
| 物化视图 port_id 锚定 + 既存迁移 | MV 已 port_id JOIN；一次性数据修复已落地 | ✓ (已闭合) |
| 修源头: FK 约束 + UPSERT | FK 约束 port_id → sys_device_port_status(id)；根除未来脏数据 | |
| 只验证 4F001 单工位案例 | 最小化: 单工位 UPDATE;不动 schema | |

**User's choice:** 这个已经处理了
**Notes:** 验证发现 `migration_188_fix_info_points_port_id_drift.go` 已落地，本 phase 不复实施。

---

## Area 4: R4 — sys_device_port_status 行 id/device_id 漂移

| Option | Description | Selected |
|--------|-------------|----------|
| 清理脏行 + 复合 uniqueIndex | 一次性数据清理 + 复合 unique；已有 uniqueIndex 即覆盖 | ✓ (已闭合) |
| 加 PG FK 约束 | FK fk_port_status_device + ALTER TABLE ... VALIDATE | |
| 只对设备/接口组合校验 | 仅 SELECT 检查无重复行；零 migration 改动 | |

**User's choice:** 这个应该也处理过了
**Notes:** 验证发现 `migration_183_add_port_status_device_fk.go` 已落地（FOREIGN KEY fk_port_status_device 已加 + VALIDATE 异步验证）。

---

## Area 5: R1/R4 是否复盖复述

| Option | Description | Selected |
|--------|-------------|----------|
| 仅写 R2/R3 + R5，R1/R4 锁为"已闭合" | CONTEXT 中说明 R1/R4 落地即可；计划仅含 R2/R3 + R5 两项 | ✓ |
| 依然复盖全部 4 项 (重新研究) | 重新在计划中复述所有 4 项 | |

**User's choice:** 仅写 R2/R3 + R5，R1/R4 锁为"已闭合"
**Notes:** docs/state/roadmap 中记为"迁移补齐"；计划仅含 R2/R3 + R5 两项。

---

## Area 6: R2/R3 UPSERT WHERE 过滤形式

| Option | Description | Selected |
|--------|-------------|----------|
| 用 partial uniqueIndex WHERE 排除 resolved_at IS NULL | 复用 uniq_recon_asset_type_open 索引；已 resolved 不参与 conflict index 路径 | ✓ |
| 不用 partial index，加 WHERE resolved_at IS NULL 谓词 | ON CONFLICT WHERE; 实现复杂 | |

**User's choice:** 用 partial uniqueIndex WHERE 排除 resolved_at IS NULL
**Notes:** D-01 锁 GORM `clause.OnConflict{Target: "uniq_recon_asset_type_open"}`。

---

## Area 7: R5 数据清理脚本载体

| Option | Description | Selected |
|--------|-------------|----------|
| 走一次性 migration (与 R2/R3 UPSERT 同 batch) | migration_NNN_cleanup_dirty_mac_rows.go + UPSERT migration 同 batch；GO 透明可回滚（软删） | ✓ |
| 独立 cmd/cleanup_dirty_macs | cmd/clear_dirty_macs/ 手动 go run | |
| 不写迁移，迁后留下查询定位脚本 | internal/sql/queries/find_dirty_macs.sql 仅巡查 | |

**User's choice:** 走一次性 migration (与 R2/R3 UPSERT 同 batch)
**Notes:** 备份表 `sys_dirty_mac_rows_backup` + UPDATE deleted_at=NOW()；mac_history 不动；幂等保护。

---

## Claude's Discretion

下列细节由 plan-phase 自决：
- UPSERT migration 文件名（建议 `migration_194`）
- 数据清理 migration 是否拆为独立 `migration_195`（推荐拆 2）
- DetectLayer3 内部 `rec.ID = ""` 显式置空确保不带 PK UPDATE
- 备份表是否建索引
- `logrus.Infof` 清理统计
- TestParseRuijiePortSecurityLine 新增 4 个负向 sub-test（推荐）

---

## Deferred Ideas

- v1.18 RQ-002 路线 候选：transport 层校验（raw_text 入口），MAC parser fuzz test
- 已闭合项 R1/R4 不复实施（migration_188/_183 已落地）
- DB TRIGGER 路线被用户偏好拒绝（项目记忆）
- 不动 mac_history 清理（脏数据的历史链是审计价值）
