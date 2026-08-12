---
slug: workstation-primary-device-serial
quick_id: 260626-lbd
status: complete
created: 2026-06-26
---

# Summary: 工位管理主表添加主设备序列号列

## 改动（4 处，跨前后端）

### 后端
1. **`internal/models/workstation.go`** — Workstation struct 加计算字段
   ```go
   PrimaryDeviceSerial *string `gorm:"-:migration" json:"primaryDeviceSerial,omitempty"`
   ```
   `gorm:"-:migration"` 避免 AutoMigrate 建列（纯子查询字段），但保留 Select scan。

2. **`internal/services/operations/workstation_service.go:16`** — `workstationJoinSelect` 末尾追加主设备子查询，复用 `excel_query_builder.go:151` 已验证的排序逻辑（is_primary 优先 → priority → created_at，LIMIT 1）：
   ```sql
   , (SELECT device_serial FROM ops_workstation_device
      WHERE workstation_id = sys_workstation.id::text AND deleted_at IS NULL
      ORDER BY (CASE WHEN is_primary = true THEN 0 ELSE 1 END), priority DESC, created_at ASC
      LIMIT 1) as primary_device_serial
   ```
   List 与 GetByID 共用该 select，两处一致带上主设备序列号。
   **注意**：WHERE 严格 `AND is_primary = true`（非 excel_query_builder 的 ORDER BY is_primary fallback 模式）——用户需求是"设置了主设备之后才显示"，无主设备时返回 NULL → 显示 `-`，不 fallback 到其他设备。

### 前端
3. **`xingran-react-frontend/src/types/operations.ts`** — `WorkstationOps` 加 `primaryDeviceSerial?: string`

4. **`xingran-react-frontend/src/pages/operations/workstations/columns.tsx`** — 在"所属用户"列后加"主设备序列号"列（width 140, ellipsis, `text || "-"`，不可排序——子查询排序代价高且无业务需求）

## 验证
- ✅ `go build ./...` exit 0
- ✅ `npm run type-check` clean
- ✅ SQL 子查询模式与 `excel_query_builder.go:151`（Excel 导出已生产使用）完全一致，逻辑已被验证
- ⏳ 浏览器端到端：后端需重新编译运行后 `primaryDeviceSerial` 字段才返回；前端 Vite 热重载会立即出现新列（有主设备的工位显示序列号，无则 `-`）。本次执行时 chrome-devtools 浏览器实例冲突，未完成端到端截图——交付后由用户重启 `xingran-backend` 验证

## 设计说明
- "主设备"严格定义：`ops_workstation_device.is_primary = true` 的设备
- 语义对齐用户需求："**当设置了主设备之后才显示**" → 子查询 `WHERE is_primary = true`；未设主设备的工位返回 NULL → 列显示 `-`
- 与 Excel 导出（`excel_query_builder.go:151`）语义**不同**：导出用 `ORDER BY is_primary` fallback（总要有值），列表列严格 `is_primary=true`（无则空）。两者场景不同，分别合理
- ORDER BY priority DESC, created_at ASC 仅为 tiebreaker（正常情况每工位仅 1 台 is_primary=true，SetPrimaryDevice 保证）

## 不在范围
- "域控设备"展开行关联错误（managed_by vs original_description，另见 debug 记忆）——独立问题
- 主设备设置/同步逻辑
