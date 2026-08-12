# Phase 29: sys-dict（系统字典） - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-10
**Phase:** 29-sys-dict
**Areas discussed:** 7 gray areas

---

## [Area 1] 字典值设计

| Option | Description | Selected |
|--------|-------------|----------|
| 数字字符串 | dict_value 使用 '0', '1', '2' 等数字字符串 | |
| 语义字符串 | dict_value 使用 'available', 'occupied', 'maintenance' | |
| 混合方案 | dict_value 使用语义字符串，通过 dict_sort 映射到数字 | |

**User's choice:** 复用其他字典设计（语义字符串）

**Notes:** 
- 参考现有 sys_dict 模式（ops_info_point_type, ops_dedicated_line_type）
- 使用语义字符串提供更好的可读性
- 与数据库字段类型改为 string 配合

---

## [Area 2] 迁移策略

| Option | Description | Selected |
|--------|-------------|----------|
| 全量迁移 | 一次性迁移所有记录，更改字段类型 | ✓ |
| 双字段并行 | 添加新字段，逐步迁移 | |
| 仅迁移新数据 | 保持现有，仅新记录使用新逻辑 | |

**User's choice:** 全量迁移

**Notes:**
- 将 status 字段从 int 改为 string
- 创建迁移脚本：0→'available', 1→'occupied', 2→'maintenance'
- 在低峰期执行，可能需要短暂锁定表

---

## [Area 3] 前端缓存策略

| Option | Description | Selected |
|--------|-------------|----------|
| 组件级缓存 | useState 在组件内缓存 | ✓ |
| 全局缓存 | Zustand store 全局缓存 | |
| 无缓存 | 每次调用 API | |

**User's choice:** 检查当前项目实际做法 → 组件级缓存

**Notes:**
- 沿用项目现有模式（参考 info-points 和 dedicated-lines 页面）
- 使用 useState<DictData[]>([])
- 页面加载时通过 useEffect 获取字典数据

---

## [Area 4] 向后兼容性

| Option | Description | Selected |
|--------|-------------|----------|
| 完全替换 | 移除 WorkstationStatus 枚举 | ✓ |
| 类型定义保留 | 保留枚举作为类型定义 | |
| 动态映射 | 从字典动态获取映射 | |

**User's choice:** 完全替换

**Notes:**
- 移除 type WorkstationStatus int 和相关常量
- Status 字段类型从 WorkstationStatus 改为 string
- 全面搜索替换所有使用该枚举的地方

---

## [Area 5] 其他枚举类型

| Option | Description | Selected |
|--------|-------------|----------|
| 仅工位状态 | 只重构 WorkstationStatus | ✓ |
| 工位状态+类型 | 同时重构 WorkstationType | |
| 全部工位枚举 | 重构所有相关枚举 | |

**User's choice:** 仅工位状态

**Notes:**
- 本期范围明确：只重构 WorkstationStatus
- WorkstationType 和 DeskType 保持不变
- 减少风险和工作量

---

## [Area 6] UI 样式映射

| Option | Description | Selected |
|--------|-------------|----------|
| 使用 css_class | 使用 dict.css_class（success/error/warning） | ✓ |
| 保留映射表 | 保留 STATUS_COLOR_MAP | |
| 混合方案 | css_class + 映射表组合 | |

**User's choice:** 使用 css_class

**Notes:**
- 删除前端 STATUS_COLOR_MAP
- 在 sys_dict_data 中设置 css_class
- Tag 组件直接使用 dict.css_class

---

## [Area 7] API 响应格式

| Option | Description | Selected |
|--------|-------------|----------|
| 返回字符串 | 返回 dict_value | |
| 标签-值分离 | 显示用 dict_label，提交用 dict_value | |
| 双字段响应 | status + status_text | ✓ |

**User's choice:** 后端直接保存中文

**Notes:**
- API 返回包含 status（dict_value）和 status_text（dict_label）
- 前端直接使用 status_text 显示
- 提交时发送 status 字段

---

## Claude's Discretion

无 - 用户对每个灰色区域都做出了明确选择

---

## Deferred Ideas

- WorkstationType 字典化
- DeskType 字典化  
- 其他模块字典化
- 字典管理 UI
- 多语言支持
- 字典缓存优化

---

*Phase: 29-sys-dict*
*Discussion log: 2026-06-10*
