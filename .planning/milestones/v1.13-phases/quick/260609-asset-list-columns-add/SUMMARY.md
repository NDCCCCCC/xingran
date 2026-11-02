---
status: complete
title: 添加资产列表显示字段
completed_at: "2026-06-09T00:45:00Z"
---

## 任务完成摘要

### 修改内容
在资产列表页面 (`xingran-react-frontend/src/pages/operations/assets/index.tsx`) 添加了5个新的显示字段：

| 字段 | 中文名 | 位置 | 格式 |
|------|--------|------|------|
| `signOrgnoName` | 归属机构 | 受益部门后 | 文本 |
| `nowUserDeptCode` | 部门编码 | 责任人后 | 文本 |
| `drawingDate` | 接收日期 | 领取人后 | YYYY-MM-DD |
| `machineUptime` | 最后上线 | 接收日期后 | YYYY-MM-DD HH:mm |
| `lastInventoryDate` | 盘点日期 | 最后上线后 | YYYY-MM-DD |

### 技术细节
- 添加了 `dayjs` 导入用于日期格式化
- 日期字段使用条件渲染：`date ? dayjs(date).format(...) : '-'`
- 所有字段设置了合理的列宽和 `ellipsis: true` 属性

### 验证
- TypeScript 类型检查通过
- Git 提交完成：`cd62637`

### 后续建议
用户可通过"列设置"按钮调整这些新字段的显示/隐藏状态和顺序。
