# ROADMAP.md 更新方案

## 目标

将 ROADMAP.md 中的 Phase 24 恢复为"UUID类型一致性优化"，并将"虚拟机数据范围权限配置"移至新的 Phase 25。

## 当前状态

- ROADMAP.md 中 Phase 24 是"虚拟机数据范围权限配置"（错误）
- `.planning/milestones/infra-phases/24-uuid/` 目录包含 UUID 优化的完整内容（正确的 Phase 24）
- 需要添加新的 Phase 25 为虚拟机权限配置

## 更新步骤

### 1. 替换 Phase 24 内容

将当前 ROADMAP.md 中的：
```markdown
### Phase 24: 虚拟机数据范围权限配置

**Goal:** 为虚拟机列表配置细粒度权限，包括启动、关机、重启、同步、删除等操作；修改绑定用户功能以支持数据范围权限识别

**Requirements**:
- 各种操作（启动、关机、重启、同步、删除）需要单独配置权限
- 绑定用户功能仅用于数据范围权限识别：
  - 无绑定用户：仅数据范围为"全部"的角色可见
  - 绑定用户张三：张三所在部门和张三本人可见

**Depends on:** Phase 22 (深信服桌面云集成)
**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 24 to break down)
```

替换为：
```markdown
### Phase 24: UUID类型一致性优化

**Goal:** 统一代码库中的 UUID 类型处理模式，确保新代码遵循一致的规范。本阶段不涉及大规模重构已有表，而是建立清晰的规范指导未来开发，并通过文档和测试确保规范得到遵守。

**Requirements**:
- 统一 UUID 生成策略为 Go 侧生成
- 保持 Go 层使用 string 类型（向后兼容）
- 明确外部系统 UUID 字段处理方式
- 建立迁移脚本的编写规范
- 补充单元测试确保规范执行
- 更新开发规范文档

**Depends on:** 无（独立规范整理工作）
**Plans:** 2 plans

**Wave 1 - 文档与规范**:
- [x] 24-01-PLAN.md — 开发规范文档更新
- [x] 24-02-PLAN.md — 单元测试覆盖

**Phase Highlights**:
- 📚 Complete UUID handling standards documentation
- 🧪 Test coverage for UUID generation and validation
- 📝 Clear migration script patterns
- 🚫 Anti-pattern warnings for developers
```

### 2. 添加新的 Phase 25

在 Phase 24 之后添加：
```markdown

### Phase 25: 虚拟机数据范围权限配置

**Goal:** 为虚拟机列表配置细粒度权限，包括启动、关机、重启、同步、删除等操作；修改绑定用户功能以支持数据范围权限识别

**Requirements**:
- 各种操作（启动、关机、重启、同步、删除）需要单独配置权限
- 绑定用户功能仅用于数据范围权限识别：
  - 无绑定用户：仅数据范围为"全部"的角色可见
  - 绑定用户张三：张三所在部门和张三本人可见

**Depends on:** Phase 22 (深信服桌面云集成)
**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 25 to break down)
```

### 3. 更新 Progress 表格

在 Progress 表格中添加：
```markdown
| 24. UUID类型一致性优化 | - | 2/2 | Complete | 2026-05-27 |
| 25. 虚拟机数据范围权限配置 | - | 0/0 | Planned | - |
```

### 4. 更新里程碑统计

在 Total 计算中：
- 将 "83 plans completed" 更新为 "85 plans completed"（加上 Phase 24 的 2 个计划）
- 将 "108 total plans" 保持不变（因为 Phase 25 计划数待定）

## 验证步骤

1. 检查 ROADMAP.md 中 Phase 24 内容是否正确
2. 检查 Phase 25 是否正确添加
3. 检查 Progress 表格是否包含新的 Phase 24 和 Phase 25
4. 运行 `gsd-sdk query roadmap.get-phase 24` 验证 Phase 24 信息
5. 运行 `gsd-sdk query roadmap.get-phase 25` 验证 Phase 25 信息

## 执行后操作

更新完成后，执行：
```bash
# 删除错误的 24-vm-data-scope-permissions 目录（如果需要）
rm -rf .planning/phases/24-vm-data-scope-permissions

# 为新的 Phase 25 创建目录
mkdir -p .planning/phases/25-vm-data-scope-permissions

# 运行 discuss-phase 为 Phase 25 创建上下文
/gsd-discuss-phase 25
```

## 文件状态

- ✅ Phase 24 UUID 优化内容：存在于 `.planning/milestones/infra-phases/24-uuid/`
- ⏳ ROADMAP.md：需要手动更新
- ⏳ Phase 25 目录：需要创建
