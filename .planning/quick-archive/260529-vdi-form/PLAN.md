# Phase Quick Task 260529-vdi-form: VDI虚拟机创建表单UI优化

## 目标
优化VDI虚拟机创建表单的用户体验，解决以下三个问题：

1. **模态框布局优化** - 当前表单过长（14个字段垂直排列），改为两列布局
2. **虚拟机命名优化** - 当前命名不够友好，需要更好的命名规则
3. **API性能优化** - VTP/资源/运行位置获取速度慢，需要缓存加速

## 实现计划

### Task 1: 前端 - 两列布局优化
**文件**: `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx`

将当前的单列布局改为两列布局：
- 左列：基础配置（VDI服务器、资源组、资源、VTP平台、运行位置、存储位置、网络接口）
- 右列：虚拟机配置（名称、CPU、内存、磁盘、创建数量、主机位置）

使用Ant Design的Row和Col组件实现响应式两列布局。

### Task 2: 前端 - 虚拟机命名优化
**文件**: `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx`

当前命名方式：`VM-{资源组名}-{资源名}-{后缀}`

优化为更友好的命名：
- `VDI-{资源组名}-{资源名}-{用户名}-{序号}`
- 示例：`VDI-研发部-资源池1-zhangsan-001`

### Task 3: 前端 - API性能优化
**文件**: `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx`

当前问题：每次打开模态框都重新调用VDI API获取数据（3-10秒）

解决方案：
1. 添加前端内存缓存（useState缓存 + 5分钟过期）
2. 显示加载状态（Spin组件）
3. 后端添加Redis缓存（可选，如果前端缓存不够）

## 验收标准
- [ ] 模态框高度显著降低（从当前~800px降到~500px）
- [ ] 两列布局在宽屏下正常显示，移动端自动单列
- [ ] 虚拟机名称更易读易记
- [ ] 第二次打开模态框时数据加载时间<1秒（缓存命中）

## 风险
- 两列布局在移动端可能显示问题 → 使用响应式Col（xs={24} md={12}）
- 缓存可能导致数据过期 → 添加刷新按钮和缓存过期时间
