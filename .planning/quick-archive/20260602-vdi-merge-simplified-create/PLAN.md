---
description: 整合 VDI 虚拟机简化创建逻辑到主文件
created: 2026-06-02
status: in-progress
---

# 目标

将 `index.tsx.bk` 中的简化创建虚拟机功能整合到主文件 `index.tsx` 中。

# 背景

- `index.tsx.bk` 包含简化版创建功能：硬编码默认值，自动加载配置，简化表单
- `index.tsx` 是完整功能页面，但创建时需要手动选择所有配置
- 用户希望快速创建虚拟机而不需要手动选择所有配置项

# 任务分解

1. **分析两个文件差异**
   - 识别 `.bk` 文件中的简化创建逻辑核心部分
   - 确定需要迁移的功能点

2. **设计整合方案**
   - 选项 A: 添加"简化创建"按钮，打开简化模态框
   - 选项 B: 在现有创建模态框中添加"使用默认配置"开关
   - **采用方案 A**: 更清晰，不影响现有功能

3. **实现简化创建功能**
   - 添加 `simplifiedCreateModalVisible` 状态
   - 添加 `simplifiedConfig` 状态存储自动加载的配置
   - 添加 `loadSimplifiedDefaults` 函数
   - 添加 `handleSimplifiedCreate` 函数
   - 添加简化创建模态框 UI

4. **测试验证**
   - 运行前端开发服务器
   - 测试简化创建功能是否正常工作

# 实现细节

## 需要从 .bk 文件迁移的核心功能

1. **默认值常量**:
   ```typescript
   const DEFAULT_VTP_ID = 1;
   const DEFAULT_RESOURCE_GROUP_ID = '0';
   const DEFAULT_RESOURCE_NAME = '数据';
   const DEFAULT_POSITION_NAME = '研发';
   ```

2. **配置状态**:
   ```typescript
   const [simplifiedConfig, setSimplifiedConfig] = useState<{
     vdiServerId: string;
     resourceId: string;
     runPositionId: string;
     diskId: string;
     storageId: string;
     networkId: string;
     hostId: string;
   } | null>(null);
   ```

3. **loadSimplifiedDefaults 函数**: 自动加载默认配置
   - 获取 VDI 服务器
   - 获取资源组
   - 获取资源（匹配名称）
   - 获取 VTP 平台
   - 获取运行位置（匹配名称）
   - 获取存储和网络（选择第一个）

4. **handleSimplifiedCreate 函数**: 处理简化创建
   - 处理 run_position_id 逻辑（id == father_id 时为空）
   - 调用 vmApi.create

5. **简化创建模态框 UI**:
   - 显示已加载的默认配置信息
   - 简化的表单（名称、CPU、内存、磁盘、创建数量）

# 文件修改

- `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx`
  - 添加简化创建相关状态和函数
  - 添加"简化创建"按钮
  - 添加简化创建模态框

# 验收标准

- [ ] 简化创建按钮存在并可点击
- [ ] 打开简化创建模态框时自动加载默认配置
- [] 默认配置显示正确（VDI服务器、资源组、资源、VTP平台、运行位置、存储、网络）
- [] 填写简化表单后能成功创建虚拟机
- [] 不影响现有的完整创建功能
