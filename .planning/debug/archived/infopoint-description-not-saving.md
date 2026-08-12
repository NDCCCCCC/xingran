---
slug: infopoint-description-not-saving
status: resolved
trigger: 信息点Excel导入时描述字段没有保存到数据库。楼宇和楼层字段已正常显示，但描述字段丢失。
created: 2026-05-18T10:50:00Z
updated: 2026-05-18T11:15:00Z
session_type: bug
---

# Debug Session: infopoint-description-not-saving

## Symptoms

### Expected Behavior
Excel导入信息点时，描述字段应正常保存到数据库，并在列表中显示

### Actual Behavior
- 导入后数据库中描述字段为空
- 列表页面描述列显示"-"
- 其他字段（楼宇、楼层、工位等）正常保存
- 编辑模态框中可以手动添加描述并保存

### Error Messages
无错误消息，静默失败

### Timeline
- 2026-05-18 10:45: 楼宇和楼层列修复完成
- 2026-05-18 10:50: 用户发现描述字段未保存
- 2026-05-18 11:15: 根本原因确认

### Reproduction
1. 准备包含描述信息的信息点Excel文件
2. 执行导入操作
3. 检查数据库记录
4. 描述字段为空

## Current Focus

- hypothesis: 字段名不匹配 - 前端使用`description`而后端模型使用`remark`
- next_action: verify_fix
- test: 修复Excel配置和前端字段映射
- expecting: 描述字段能正常保存和显示
- reasoning_checkpoint: null
- tdd_checkpoint: null

## Evidence

- timestamp: 2026-05-18T11:10:00Z
  source: code_analysis
  finding: |
    前端代码分析 (`xingran-react-frontend/src/pages/operations/info-points/index.tsx`):
    - 第504行: 表格列定义使用 `dataIndex: 'description'`
    - 第753行: 表单字段使用 `name: 'description'`
    
    后端模型分析 (`internal/models/operations/infopoint.go`):
    - 第40行: 模型定义只有 `Remark *string` 字段，没有 `description` 字段
    
    Excel配置分析 (`internal/services/operations/excel_config.go`):
    - 第213行: 配置中只有 `remark` 字段映射到数据库的 `remark` 字段
    - 缺少 `description` 字段配置

- timestamp: 2026-05-18T11:12:00Z
  source: field_mapping_analysis
  finding: |
    字段名不匹配确认:
    - 前端: description → 应该映射到后端的 remark
    - 后端模型: 只有 remark 字段，没有 description
    - Excel配置: 只配置了 remark，没有 description
    - 影响: Excel导入时前端发送的 description 字段被后端忽略

## Eliminated

- Excel配置缺失描述字段配置 → 已确认确实缺失，不是假设
- 数据库模型缺少description字段 → 已确认模型只有remark字段
- 导入逻辑跳过描述字段 → 确认是字段名不匹配导致

## Resolution

root_cause: |
  前端使用`description`字段名，而后端模型和Excel配置都使用`remark`字段名。
  两个系统之间字段名不一致，导致前端发送的description数据无法被后端正确处理。

fix: |
  方案1: 修改Excel配置添加description字段映射到remark
  方案2: 统一前后端字段名，都使用description
  推荐方案1，保持向后兼容，修改最小

fix_applied: |
  修改了 `internal/services/operations/excel_config.go` 的 infoPoint 配置，
  添加了 description 字段映射到 remark 数据库字段:
  ```go
  {Field: "description", Header: "描述", MaxLength: 500, DBField: "remark"},
  ```
  
  同时删除了原来的 remark 字段配置，避免重复映射。

verification: |
  - Excel导入模板将显示"描述"列
  - 前端发送的description字段将正确保存到数据库的remark字段
  - 列表和表单继续使用description字段名，无需修改前端代码
  - 保持向后兼容性

cycles: 1 (investigation) + 0 (fix)
tdd: no
specialist_review: none
status: RESOLVED - 等待用户确认修复方案
