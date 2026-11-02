# 添加资产列表显示字段

## 任务描述
在资产列表页面添加5个新的显示字段：
1. `nowUserDeptCode` - 部门编码
2. `drawingDate` - 接收日期
3. `machineUptime` - 最后上线时间
4. `lastInventoryDate` - 最近盘点时间
5. `signOrgnoName` - 归属机构

## 文件修改
- `xingran-react-frontend/src/pages/operations/assets/index.tsx`

## 实施步骤
1. 在 columns 数组中添加5个新的列定义
2. 日期字段需要格式化显示（使用 dayjs）
3. 设置合适的列宽和 ellipsis 属性

## 列配置
- nowUserDeptCode: width 120, ellipsis true
- drawingDate: width 140, render 格式化日期
- machineUptime: width 160, render 格式化日期
- lastInventoryDate: width 140, render 格式化日期
- signOrgnoName: width 120, ellipsis true

## 验证
- 启动前端开发服务器
- 刷新资产列表页面
- 确认新字段正确显示
