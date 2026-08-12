# Phase 10: 网络设备导出集成 - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

将现有导出功能集成到路由和前端，实现全类型批量导出功能。核心目标：验证前端集成状态，实现批量导出（单次请求打包为 .zip），支持所有实体类型。厂商特定格式继续使用 Excel（已满足需求）。
</domain>

<decisions>
## Implementation Decisions

### 前端集成状态
- **D-01**: NetworkExport 组件已在所有网络管理页面集成
  - 验证范围：devices, credentials, templates, command, executions, backups, discoveries, mac, ports
  - 组件位置：`src/components/shared/NetworkExport.tsx`
  - 支持三种导出模式：筛选导出、当前页导出、全部导出
  - **EXPORT-03b 实际上已完成**，无需额外工作

### 厂商特定格式
- **D-02**: 继续使用 Excel 格式（当前实现已满足需求）
  - vendor 和 model 列已包含在导出中
  - vendor 映射为中文（华为、H3C、锐捷、迈普）
  - device_type 映射为中文（路由器、交换机、防火墙、AP、负载均衡）
  - **EXPORT-03c 需求已通过现有 Excel 格式满足**

### 导出进度显示
- **D-03**: 不需要额外的进度显示功能
  - 当前实现使用 `loading` 状态和 `message.success/error` 反馈已足够
  - 单次请求模式，快速响应
  - 异步导出 + WebSocket 进度推送属于过度工程

### 批量导出支持
- **D-04**: 实现全类型批量导出功能
  - **范围**：所有 9 个实体类型都支持批量导出
  - **打包格式**：单个 .zip 包，包含所有导出文件
  - **触发方式**：在各列表页面添加"批量导出"按钮
  - **实现方式**：利用现有 `rowSelection`（Ant Design Table 行选择）
  - **导出内容**：基于用户选中的行或当前筛选条件
  - **文件命名**：每个文件按实体类型命名（如 `网络设备.xlsx`, `授权凭证.xlsx`）
  - **ZIP 命名**：`网络管理_批量导出_{timestamp}.zip`

### Claude's Discretion
- 批量导出的具体实现方式（同步打包 vs 异步任务 + 下载）
- ZIP 内部文件的组织结构（扁平结构 vs 按类型分目录）
- 批量导出的数据量限制（是否设置最大设备数/记录数）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求文档
- `.planning/REQUIREMENTS.md` — EXPORT-03a, EXPORT-03b, EXPORT-03c 需求定义
- `.planning/ROADMAP.md` — Phase 10 成功标准

### 现有代码
- `internal/api/v1/network/network_export_handler.go` — 9 个导出函数实现（已存在）
- `internal/api/v1/network/network_router.go` — 导出路由注册（已完成）
- `xingran-react-frontend/src/components/shared/NetworkExport.tsx` — 前端导出组件（已存在）

### 前端页面
- `xingran-react-frontend/src/pages/network/devices/index.tsx` — 设备管理页面（已集成 NetworkExport）
- `xingran-react-frontend/src/pages/network/credentials/index.tsx` — 凭证管理页面（已集成 NetworkExport）
- `xingran-react-frontend/src/pages/network/templates/index.tsx` — 模板管理页面（已集成 NetworkExport）
- 其他网络管理页面（command, executions, backups, discoveries, mac, ports）

### 项目规范
- `.planning/codebase/CONVENTIONS.md` — Go 代码风格、错误处理规范
- `.planning/codebase/ARCHITECTURE.md` — 分层架构、Handler-Service 模式

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **NetworkExport 组件**: 完整的导出组件，支持三种模式（筛选、当前页、全部）
- **excelize v2**: 后端 Excel 生成库，已在 `network_export_handler.go` 中使用
- **Ant Design Table rowSelection**: 前端表格行选择，可用于批量操作

### Established Patterns
- **导出 API 响应**: 使用 `Content-Disposition` header 设置文件名，直接返回二进制数据
- **前端下载方式**: 创建 blob URL，动态生成 `<a>` 标签触发下载
- **错误处理**: 使用 `message.success/error` 显示导出结果

### Integration Points
- 后端导出端点：`/api/v1/network/*/export`（9 个实体类型）
- 前端导出组件：`NetworkExport` 组件已在所有网络管理页面使用
- 批量导出需要：后端新增批量导出端点 + 前端新增批量导出按钮

### 已实现功能
- 9 个实体的单个导出（devices, credentials, templates, command, executions, backups, discoveries, mac, ports）
- 三种导出模式（筛选、当前页、全部）
- Excel 格式，包含厂商和型号信息

</code_context>

<specifics>
## Specific Ideas

### 批量导出实现建议
1. **后端实现**：
   - 新增 `POST /api/v1/network/batch-export` 端点
   - 请求体包含：`entityTypes`（要导出的实体类型列表）、`filters`（各类型的筛选条件）
   - 使用 `archive/zip` 包创建 ZIP 文件
   - 为每个实体类型调用现有导出函数，收集生成的 Excel 文件
   - 返回 ZIP 文件流

2. **前端实现**：
   - 在各列表页面工具栏添加"批量导出"按钮
   - 按钮仅在 `rowSelection.selectedRowKeys.length > 0` 时启用
   - 点击后弹出模态框，选择要导出的实体类型（默认全选）
   - 调用批量导出 API，下载 ZIP 文件

3. **文件组织**：
   - ZIP 内部使用扁平结构：`网络设备.xlsx`, `授权凭证.xlsx`, 等
   - 或按类型分目录：`devices/网络设备.xlsx`, `credentials/授权凭证.xlsx`

</specifics>

<deferred>
## Deferred Ideas

- 异步导出 + WebSocket 进度推送（过度工程，当前不需要）
- 厂商特定文本格式（.cfg/.txt 原始配置格式）
- 多厂商配置标准化（保真度损失，不在本期范围）
- 实时配置流式传输（过度工程）

</deferred>

---

*Phase: 10-network-export*
*Context gathered: 2026-04-27*
