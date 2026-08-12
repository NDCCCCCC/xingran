# Phase 28: 工位设备关联子表格 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-10
**Phase:** 28-workstation-device-association
**Areas discussed:** 子表格交互, 数据存储, 同步逻辑, 序列号匹配, 权限控制, 错误处理, 并发控制, 审计日志

---

## 子表格交互 (UI/UX)

| Option | Description | Selected |
|--------|-------------|----------|
| 点击行展开 | 点击行展开/收起，默认收起（Ant Design 标准模式） | ✓ |
| 独立按钮 | 每行有独立的展开/收起按钮（更明确） | |
| 混合模式 | 点击行展开，但有独立的收起按钮（混合模式） | |

**User's choice:** 点击行展开
**Notes:** 使用 Ant Design Table 的标准 expandable 功能

---

| Option | Description | Selected |
|--------|-------------|----------|
| 简化列表 | 显示设备名称、序列号、状态、来源四个核心字段（简洁） | |
| 详细信息 | 显示设备详细信息（型号、责任人、部门、时间等） | ✓ |
| 标签化展示 | 使用 Tag 标签区分来源，关键信息高亮 | |

**User's choice:** 详细信息
**Notes:** 提供全面的设备视图

---

| Option | Description | Selected |
|--------|-------------|----------|
| 颜色标签 | 用不同颜色的 Tag 标签区分（域控=蓝色，资产=绿色，实际=橙色） | ✓ |
| 图标前缀 | 在设备名称前加图标区分域控/资产/实际来源 | |
| 文本列 | 使用独立的'来源'列显示文本（域控设备/资产系统/实际结果） | |

**User's choice:** 颜色标签
**Notes:** 三个来源需要先详细说明区别再决定

---

| Option | Description | Selected |
|--------|-------------|----------|
| 行内操作 | 子表格内每行设备都有操作按钮（删除、同步） | ✓ |
| 批量操作 | 子表格顶部有批量操作按钮（批量删除、批量同步） | |
| 组合方式 | 行内操作 + 顶部批量操作，提供完整的操作能力 | |

**User's choice:** 行内操作
**Notes:** 每行设备独立操作

---

| Option | Description | Selected |
|--------|-------------|----------|
| 删除设备 | 从实际结果中删除该设备关联 | |
| 同步域控 | 同步按钮：同步域控设备 → 实际结果 | ✓ |
| 同步资产 | 同步按钮：同步资产系统 → 实际结果 | ✓ |
| 编辑备注 | 编辑设备信息（如备注、优先级等附加信息） | ✓ |

**User's choice:** 同步域控, 同步资产, 编辑备注
**Notes:** 不提供删除操作

---

| Option | Description | Selected |
|--------|-------------|----------|
| 支持分页 | 显示分页器（每页10条），适合设备数量多的工位 | |
| 滚动显示 | 滚动显示所有设备，适合设备数量少的场景 | ✓ |

**User's choice:** 滚动显示
**Notes:** 适合设备数量不多的场景

---

| Option | Description | Selected |
|--------|-------------|----------|
| 需要提示 | 显示空状态提示和图标，引导用户操作 | |
| 简单文本 | 仅显示'无设备'文本，简洁 | ✓ |

**User's choice:** 简单文本
**Notes:** 暂无设备

---

| Option | Description | Selected |
|--------|-------------|----------|
| 行内加载 | 设备行显示加载动画，同步完成后更新数据 | |
| 全局进度 | 页面顶部显示进度条，同步完成后统一通知 | |
| 静默同步 | 点击同步按钮后显示 Toast 提示，后台静默同步 | ✓ |

**User's choice:** 静默同步
**Notes:** 同步完成后通过 Toast 提示结果

---

## 数据存储 (Data Storage)

| Option | Description | Selected |
|--------|-------------|----------|
| 新表存储 | 创建独立的工位-设备关联表，支持多对多关系（推荐） | ✓ |
| 设备表字段 | 在设备表添加 workstation_id 字段（简单但有限制） | |
| JSON字段 | 使用 JSON 字段存储设备 ID 列表（灵活但查询复杂） | |

**User's choice:** 新表存储
**Notes:** 表名: ops_workstation_device

---

| Option | Description | Selected |
|--------|-------------|----------|
| 基础字段 | 关联表必须的基础字段（工位ID、设备ID、创建时间） | ✓ |
| 来源字段 | 区分数据来源（域控/资产/手动）的字段 | |
| 同步字段 | 存储同步时间和同步状态，支持增量更新 | |
| 扩展字段 | 支持设备优先级排序、备注等扩展信息 | |

**User's choice:** 基础字段
**Notes:** 仅存储实际结果

---

| Option | Description | Selected |
|--------|-------------|----------|
| 区分来源 | 通过来源字段区分，同时存储域控、资产、实际结果 | |
| 仅实际结果 | 只存储实际结果，域控/资产作为临时数据不同步到关联表 | ✓ |

**User's choice:** 仅实际结果
**Notes:** 域控/资产数据使用 Redis 缓存临时存储

---

| Option | Description | Selected |
|--------|-------------|----------|
| 实时查询 | 域控/资产设备数据不存储，同步时实时查询 | |
| Redis缓存 | 使用缓存（Redis）存储域控/资产数据，设置5-30分钟过期 | ✓ |
| 临时表 | 创建临时表存储域控/资产数据，定期清理 | |

**User's choice:** Redis缓存
**Notes:** 30分钟过期

---

| Option | Description | Selected |
|--------|-------------|----------|
| 5分钟 | 5分钟过期，数据较新鲜，适合频繁同步场景 | |
| 15分钟 | 15分钟过期，平衡数据新鲜度和缓存命中率 | |
| 30分钟 | 30分钟过期，缓存命中率最高，适合同步频率低的场景 | ✓ |

**User's choice:** 30分钟
**Notes:** 需要参数化配置

---

| Option | Description | Selected |
|--------|-------------|----------|
| sys_workstation_device | 遵循系统表命名规范，使用 sys_ 前缀 | |
| ops_workstation_device | 使用 ops_ 前缀，表示运维模块 | ✓ |
| workstation_device | 不使用前缀，直接命名 | |

**User's choice:** ops_workstation_device
**Notes:** 与现有 ops 表保持一致

---

**User's free input:** 所有设置要以参数的形式定义在参数管理里，支持前端修改立即生效

**Notes:** 重要的架构决策，使用 sys_config 表驱动配置

---

## 同步逻辑 (Sync Logic)

| Option | Description | Selected |
|--------|-------------|----------|
| 手动触发 | 用户手动点击同步按钮触发（灵活但需要用户操作） | ✓ |
| 定时同步 | 后台定时任务自动同步（如每小时一次，减少用户操作） | |
| 组合方式 | 手动触发 + 定时同步，既支持主动同步也保持数据新鲜 | |

**User's choice:** 手动触发
**Notes:** 用户完全控制

---

| Option | Description | Selected |
|--------|-------------|----------|
| 全部同步 | 同步按钮同步该工位的所有域控/资产设备 | |
| 单独同步 | 用户选择特定设备后单独同步该设备 | ✓ |
| 组合方式 | 提供全部同步和单独同步两种方式 | |

**User's choice:** 单独同步
**Notes:** 每个设备有独立的同步按钮

---

| Option | Description | Selected |
|--------|-------------|----------|
| 覆盖 | 直接覆盖现有设备信息 | ✓ |
| 跳过 | 保留现有设备，显示提示消息 | |
| 询问用户 | 询问用户如何处理（覆盖/跳过/合并） | |

**User's choice:** 覆盖
**Notes:** 简单直接

---

| Option | Description | Selected |
|--------|-------------|----------|
| 基础信息 | 只同步设备ID和来源，保持实际结果简洁 | |
| 完整信息 | 同步设备完整信息（型号、部门、时间等），便于展示 | |
| 序列号匹配 | 同步时序列号匹配资产系统，获取资产详细信息 | ✓ |

**User's choice:** 序列号匹配
**Notes:** ROADMAP 明确要求的功能

---

| Option | Description | Selected |
|--------|-------------|----------|
| 精确匹配 | 精确匹配序列号，大小写敏感 | ✓ |
| 模糊匹配 | 忽略大小写和空格差异，宽松匹配 | |
| 部分匹配 | 支持序列号部分匹配（如前后6位） | |

**User's choice:** 精确匹配
**Notes:** 确保匹配准确性

---

| Option | Description | Selected |
|--------|-------------|----------|
| 报错终止 | 同步失败，显示错误提示 | ✓ |
| 降级处理 | 仅同步基础信息，不匹配资产系统 | |
| 记录日志 | 记录失败日志，继续处理其他设备 | |

**User's choice:** 报错终止
**Notes:** 确保数据一致性

---

## 序列号匹配 (Serial Number Matching)

| Option | Description | Selected |
|--------|-------------|----------|
| 底部输入框 | 子表格底部添加输入框，输入序列号后点击'添加'按钮 | |
| 模态框输入 | 点击'添加设备'按钮弹出模态框，输入序列号 | ✓ |
| 工具栏输入 | 子表格顶部工具栏添加输入框，支持快速添加 | |

**User's choice:** 模态框输入
**Notes:** 更好的输入体验

---

| Option | Description | Selected |
|--------|-------------|----------|
| 序列号输入 | 序列号输入框（必填） | ✓ |
| 设备预览 | 显示匹配到的设备信息（型号、责任人、部门）供确认 | ✓ |
| 备注输入 | 备注输入框（可选），用于记录设备用途等信息 | |
| 来源选择 | 来源选择器（手动/域控/资产） | |

**User's choice:** 序列号输入, 设备预览
**Notes:** 核心功能

---

| Option | Description | Selected |
|--------|-------------|----------|
| 实时预览 | 输入序列号后实时查询并显示设备预览（体验好但请求频繁） | |
| 按钮触发 | 点击'查询'或'验证'按钮后显示设备预览 | ✓ |
| 失焦触发 | 输入序列号后失焦（blur）时查询设备预览 | |

**User's choice:** 按钮触发
**Notes:** 减少 API 调用

---

| Option | Description | Selected |
|--------|-------------|----------|
| 报错阻止 | 显示错误提示，阻止添加 | ✓ |
| 询问继续 | 提示设备不存在，询问是否继续添加（手动模式） | |
| 允许添加 | 允许添加，标记为'未识别'设备 | |

**User's choice:** 报错阻止
**Notes:** 确保设备有效性

---

| Option | Description | Selected |
|--------|-------------|----------|
| 报错阻止 | 显示错误提示，该设备已关联 | ✓ |
| 提示跳过 | 提示设备已存在，跳过添加 | |
| 询问更新 | 询问是否更新现有设备信息 | |

**User's choice:** 报错阻止
**Notes:** 避免重复添加

---

## 权限控制 (Permission Control)

| Option | Description | Selected |
|--------|-------------|----------|
| 所有用户 | 所有登录用户都可以查看工位设备列表 | |
| 部门限制 | 仅工位所属部门或楼宇的用户可以查看 | |
| 权限控制 | 需要特定权限（如ops:workstation:device:view） | |

**User's choice:** 和工位管理页面权限放一起
**Notes:** 复用现有权限

---

| Option | Description | Selected |
|--------|-------------|----------|
| 查看=编辑 | 可以查看就可以添加/删除设备（简化权限模型） | ✓ |
| 独立权限 | 需要额外的编辑权限（如ops:workstation:device:edit） | |
| 负责人权限 | 仅工位负责人或部门管理员可以编辑 | |

**User's choice:** 查看=编辑
**Notes:** 简化权限模型

---

## 错误处理 (Error Handling)

| Option | Description | Selected |
|--------|-------------|----------|
| 提示用户 | 显示 Toast 错误提示，用户可见但不阻塞操作 | ✓ |
| 静默处理 | 静默记录错误日志，用户无感知 | |
| 分类处理 | 根据错误类型区分处理（网络错误重试，业务错误提示） | |

**User's choice:** 提示用户
**Notes:** 用户可见错误

---

| Option | Description | Selected |
|--------|-------------|----------|
| 自动重试 | 网络错误自动重试3次，间隔递增 | ✓ |
| 手动重试 | 提供'重试'按钮，用户手动触发重试 | |
| 不重试 | 不重试，直接显示错误 | |

**User's choice:** 自动重试
**Notes:** 网络容错

---

## 并发控制 (Concurrency Control)

| Option | Description | Selected |
|--------|-------------|----------|
| 覆盖模式 | 后端不加锁，后提交的覆盖前面的（last write wins） | |
| 乐观锁 | 使用乐观锁，检测到冲突时提示用户刷新 | |
| 前端锁 | 前端加锁，一人编辑时他人只读 | ✓ |

**User's choice:** 前端锁
**Notes:** localStorage 实现

---

| Option | Description | Selected |
|--------|-------------|----------|
| 本地锁 | 使用 localStorage 实现简单的页面级锁 | ✓ |
| 服务端锁 | 后端API提供锁机制，编辑时请求锁 | |
| 实时锁 | 使用 WebSocket 实现实时锁同步 | |

**User's choice:** 本地锁
**Notes:** 适合单浏览器场景

---

## 审计日志 (Audit Log)

| Option | Description | Selected |
|--------|-------------|----------|
| 添加记录 | 记录谁在何时添加了设备 | ✓ |
| 删除记录 | 记录谁在何时删除了设备 | ✓ |
| 同步记录 | 记录谁在何时执行了同步操作 | ✓ |
| 修改记录 | 记录设备信息的修改历史 | ✓ |

**User's choice:** 添加记录, 删除记录, 同步记录, 修改记录
**Notes:** 完整审计

---

| Option | Description | Selected |
|--------|-------------|----------|
| 独立表 | 创建独立的审计日志表（sys_workstation_device_log） | |
| 复用表 | 使用现有的 sys_oper_log 表 | ✓ |
| 应用日志 | 不存储，仅记录到应用日志 | |

**User's choice:** 复用表
**Notes:** 复用现有基础设施

---

## Claude's Discretion

Areas where user deferred to Claude's judgment:
- 模态框尺寸和样式
- Tag 颜色具体值
- 同步按钮图标
- 空状态引导文案
- 错误提示具体措辞
- 本地锁超时时间
- 重试延迟具体值

---

## Deferred Ideas

Ideas mentioned during discussion that were noted for future phases:
- 多工位批量设备操作
- 设备分组管理
- 设备状态监控
- 自动同步策略
- 设备生命周期管理
- 设备调拨流程
- 设备借用管理
- 跨域设备查询

---

*Discussion Log generated: 2026-06-10*
*Total areas discussed: 8*
*Total decisions captured: 34*
