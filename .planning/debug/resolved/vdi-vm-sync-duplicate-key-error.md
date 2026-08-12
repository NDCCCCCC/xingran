---
slug: vdi-vm-sync-duplicate-key-error
name: vdi-vm-sync-duplicate-key-error
status: resolved
trigger: 调查并修复 VDI 虚拟机同步中的 duplicate key 错误
created: 2026-05-29T08:40:00+08:00
updated: 2026-05-29T08:45:00+08:00
resolution: fixed
---

## Symptoms

**Expected behavior:**
同步过程应使用 "插入或更新" (upsert) 逻辑。对于已存在的虚拟机记录，应更新其字段而非报错。同步完成后显示成功统计，而非部分失败。

**Actual behavior:**
VDI 虚拟机同步过程中出现多个 `duplicate key value violates unique constraint "idx_sys_vdi_vm_vm_id"` 错误。系统尝试 INSERT 操作，但由于数据库中已存在相同 vm_id 的记录而失败。

**Error messages:**
```
ERRO[2026-05-29 08:40:06] [GORM错误] INSERT INTO "sys_vdi_vm" (...) VALUES (...)
| 耗时: 2.1513ms | 错误: ERROR: duplicate key value violates unique constraint "idx_sys_vdi_vm_vm_id" (SQLSTATE 23505)
```

**Affected VMs:**
- 数据0001_chenchao-076 (vm_id: 92)
- YF-0007_shuju003 (vm_id: 82)
- YF-0005_shuju001 (vm_id: 81)
- YF-0006_shuju002 (vm_id: 80)
- YF-0008_chenchao-076 (vm_id: 79)
- YF-0004_wangwenye-001 (vm_id: 77)

**Timeline:**
发生在定期 VDI 同步任务执行期间 (任务: [VDI虚拟机数据同步.DEFAULT])

**Reproduction:**
触发 VDI 虚拟机同步任务即可重现问题。日志显示问题在处理"研发"(ID: 5)和"数据"(ID: 9)资源组时出现。

---

## Current Focus

**hypothesis:** 待生成 - VDI 同步逻辑使用纯 INSERT 而非 GORM 的 Save/Clones upsert 机制

**next_action:** gather initial evidence - 定位 VDI 同步代码文件

**test:** 待设计

**expecting:** 待确定

---

## Evidence

---

## Eliminated

---

## Specialist Hint

---

## Root Cause Analysis

**根本原因：**
`saveOrUpdateVM` 方法（第167-274行）在查询虚拟机是否存在时，使用 GORM 默认查询行为，自动过滤软删除记录（因为 `VDIVirtualMachine` 模型包含 `DeletedAt` 字段）。

**问题流程：**
1. 查询代码：`Where("vm_id = ?", resource.ID).First(&vm)`
2. 如果虚拟机曾被软删除，GORM 自动排除该记录
3. 查询返回"未找到"（`gorm.ErrRecordNotFound`）
4. 代码尝试 `Create(&vm)` 创建新记录
5. 数据库实际已存在该 `vm_id`（仅被软删除）
6. 唯一约束 `idx_sys_vdi_vm_vm_id` 被触发

## Fix Plan

**修复方案：** 参考 `network_device_service.go:225` 的正确模式

1. **查询时包含软删除记录**：在查询时使用 `Unscoped()` 方法
2. **处理软删除记录**：如果找到的记录已被软删除，先永久删除旧记录，再创建新记录

**修改位置：** `internal/services/vdi/vm_service_impl.go` 第170-273行

**修改内容：**
```go
// 查询时使用 Unscoped() 包含软删除记录
err := s.db.WithContext(ctx).
    Unscoped().
    Where("vm_id = ?", resource.ID).
    First(&vm).Error

if err == gorm.ErrRecordNotFound {
    // 创建新记录...
} else if err == nil {
    // 检查记录是否已被软删除
    if vm.DeletedAt.Valid {
        // 永久删除旧的软删除记录
        s.db.WithContext(ctx).Unscoped().Delete(&vm)
        // 创建新记录...
    } else {
        // 更新现有记录...
    }
}
```

## Verification

**验证步骤：**
1. 编译代码确保语法正确
2. 等待下一次 VDI 同步任务执行
3. 检查日志是否还有 duplicate key 错误
4. 确认同步统计显示成功=0, 失败=0

**预期结果：**
- 不再出现 `ERROR: duplicate key value violates unique constraint` 错误
- 同步成功统计正确显示

## Files Changed

- `internal/services/vdi/vm_service_impl.go` (第170-273行)

