---
phase: 16-api-key-mgt
plan: 05b
type: execute
wave: 5
depends_on: [16-05a]
files_modified:
  - xingran-react-frontend/src/pages/system/apikeys/index.tsx
  - xingran-react-frontend/src/pages/system/apikeys/LogsModal.tsx
autonomous: true
requirements: ["INDEPENDENT"]
must_haves:
  truths:
    - 管理页面实现所有 CRUD 功能
    - 密钥脱敏显示（列表仅显示前12位）
    - 创建密钥时完整显示并支持复制
    - 使用日志和统计页面功能完整
    - 表单验证正确
    - 页面样式和交互符合项目规范
  artifacts:
    - path: xingran-react-frontend/src/pages/system/apikeys/index.tsx
      provides: 密钥管理主页面
      min_lines: 400
      contains:
        - export default function APIKeyManagement
        - Table.*columns.*dataSource
        - Modal.*visible.*form
        - const.*fetchData
    - path: xingran-react-frontend/src/pages/system/apikeys/LogsModal.tsx
      provides: 使用日志和统计 Modal
      min_lines: 200
      contains:
        - export.*function.*LogsModal
        - const.*summary
        - const.*logs
        - Modal.*visible.*onClose
  key_links:
    - from: xingran-react-frontend/src/pages/system/apikeys/index.tsx
      to: xingran-react-frontend/src/api/apikey.ts
      via: API 调用
      pattern: import.*apikey.*from.*@/api/apikey
    - from: xingran-react-frontend/src/pages/system/apikeys/LogsModal.tsx
      to: xingran-react-frontend/src/api/apikey.ts
      via: API 调用
      pattern: listUsageLogs|getUsageSummary
---

<objective>
创建 API 密钥管理的前端页面组件

目的：实现 API 密钥管理的前端功能，包括列表、创建、编辑、删除、使用日志和统计
输出：密钥管理主页面，使用日志和统计 Modal 组件

**说明：** 这是独立功能模块，不依赖 REQUIREMENTS.md 中的具体需求 ID。本计划专注于页面组件，依赖 16-05a 的类型定义和 API 客户端。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/16-api-key-mgt/16-CONTEXT.md
@.planning/phases/16-api-key-mgt/16-PATTERNS.md
@xingran-react-frontend/src/pages/system/user/index.tsx
@xingran-react-frontend/src/api/apikey.ts
@xingran-react-frontend/src/types/apikey.ts
</context>

<tasks>

<task type="auto" tdd="false">
  <name>Task 1: 创建密钥管理主页面</name>
  <files>xingran-react-frontend/src/pages/system/apikeys/index.tsx</files>
  <read_first>
    - xingran-react-frontend/src/pages/system/user/index.tsx
    - xingran-react-frontend/src/api/apikey.ts
  </read_first>
  <action>
创建 xingran-react-frontend/src/pages/system/apikeys/index.tsx 文件，实现管理页面：

1. 导入必要的模块：
   - React hooks: useState, useEffect, useMemo
   - Ant Design 组件: Table, Button, Modal, Form, Input, Select, Switch, message, Tag, Space, Tooltip
   - API 函数: listAPIKeys, createAPIKey, updateAPIKey, deleteAPIKey, toggleAPIKeyStatus
   - 类型: APIKey, APIKeyListParams

2. 定义组件状态：
   - dataSource: APIKey[]
   - loading: boolean
   - pagination: { current: number; pageSize: number; total: number }
   - modalVisible: boolean
   - modalType: 'create' | 'edit'
   - editingRecord: APIKey | null
   - createdKey: string | null
   - form: FormInstance

3. 实现 fetchData 函数：
   - 调用 listAPIKeys
   - 更新 dataSource 和 pagination
   - 错误处理：message.error

4. 实现 useEffect 初始化：
   - 初始加载数据
   - 使用 useMemo 稳定参数对象

5. 实现表格列定义：
   - 名称（name）
   - 密钥（key）：脱敏显示（maskKey 函数）
   - 作用域（scopes）：Tag 渲染
   - 继承权限（inherit_perms）：是/否
   - 状态（is_active）：Switch + Tag
   - 过期时间（expires_at）：格式化显示
   - 最后使用（last_used_at）：格式化显示
   - 操作列：
     - 查看详情
     - 编辑
     - 启用/禁用
     - 使用日志
     - 删除

6. 实现创建按钮：
   - 打开 Modal
   - 重置表单
   - 设置 modalType = 'create'

7. 实现编辑按钮：
   - 打开 Modal
   - 设置 modalType = 'edit'
   - 填充表单数据

8. 实现 Modal 表单：
   - 名称：Input，必填
   - 描述：Input.TextArea
   - 作用域：Select(mode="multiple")，选项：read, write, admin
   - 继承权限：Switch
   - IP 白名单：Input（支持 CIDR）
   - 过期时间：DatePicker（可选）

9. 实现表单提交：
   - 验证表单
   - 创建：调用 createAPIKey，显示完整密钥
   - 编辑：调用 updateAPIKey
   - 刷新列表
   - 关闭 Modal

10. 实现删除确认：
    - Modal.confirm
    - 调用 deleteAPIKey
    - 刷新列表

11. 实现启用/禁用切换：
    - 调用 toggleAPIKeyStatus
    - 刷新列表

12. 实现复制密钥功能：
    - navigator.clipboard.writeText
    - message.success('已复制到剪贴板')

13. 实现密钥脱敏显示：
    - maskKey 函数：key.slice(0, 12) + '...'

14. 实现搜索和筛选：
    - 关键词搜索框
    - 状态筛选（启用/禁用）
    - 作用域筛选

15. 实现分页：
    - onChange 处理
    - 显示总数

参考用户管理页面的布局和交互
使用 Ant Design 组件保持一致性
  </action>
  <verify>
    <automated>grep -E "export default function|const.*fetchData|const.*columns|Modal.*visible.*onOk|useEffect.*mem" xingran-react-frontend/src/pages/system/apikeys/index.tsx</automated>
  </verify>
  <done>
    - 页面组件实现完整
    - 表格列定义正确
    - CRUD 操作功能正常
    - 密钥脱敏显示正确
    - 表单验证正确
    - 错误处理完善
    - useEffect 依赖使用 useMemo 稳定
    - 样式和交互符合项目规范
  </done>
</task>

<task type="auto" tdd="false">
  <name>Task 2: 创建使用日志和统计 Modal</name>
  <files>xingran-react-frontend/src/pages/system/apikeys/LogsModal.tsx</files>
  <read_first>
    - xingran-react-frontend/src/pages/system/apikeys/index.tsx
    - xingran-react-frontend/src/api/apikey.ts
  </read_first>
  <action>
创建 xingran-react-frontend/src/pages/system/apikeys/LogsModal.tsx 文件，实现日志和统计查看：

1. 导入必要的模块：
   - React hooks: useState, useEffect
   - Ant Design 组件: Modal, Table, Statistic, Card, Row, Col, Descriptions, Tag
   - API 函数: listUsageLogs, getUsageSummary
   - 类型: APIKeyUsageLog, UsageSummary

2. 定义组件属性：
   - visible: boolean
   - apiKeyId: string
   - onClose: () => void

3. 定义组件状态：
   - logs: APIKeyUsageLog[]
   - summary: UsageSummary | null
   - loading: boolean
   - pagination: { current: number; pageSize: number; total: number }

4. 实现 useEffect 加载数据：
   - 调用 listUsageLogs 和 getUsageSummary
   - 更新状态
   - 使用 useMemo 稳定参数

5. 实现统计数据展示：
   - 总请求数（Statistic）
   - 成功率（Statistic，带百分比）
   - 平均耗时（Statistic，单位 ms）
   - 按方法统计（Card + Tag）
   - 按路径统计（Card + List）
   - 错误统计（Card + Tag）

6. 实现日志表格列定义：
   - 时间（created_at）：格式化显示
   - 方法（method）：Tag 渲染
   - 路径（path）
   - 状态码（status_code）：Tag 渲染（2xx绿色，4xx橙色，5xx红色）
   - 客户端IP（client_ip）
   - 耗时（duration）：ms
   - 成功（success）：是/否

7. 实现分页：
   - onChange 处理
   - 重新加载数据

8. 实现 Modal：
   - title: 使用日志
   - width: 1200
   - footer: null
   - 内容：统计数据 + 日志表格

9. 在主页面中集成：
   - 导入 LogsModal
   - 添加状态：logsModalVisible, selectedKeyId
   - 在操作列添加"查看日志"按钮
   - 点击时打开 Modal

使用 Card 和 Statistic 组件展示统计数据
使用 Tag 组件区分不同状态
使用 useMemo 稳定 useEffect 依赖
  </action>
  <verify>
    <automated>grep -E "export.*function.*LogsModal|const.*summary|const.*logs|Modal.*visible.*onClose|useEffect.*mem" xingran-react-frontend/src/pages/system/apikeys/LogsModal.tsx</automated>
  </verify>
  <done>
    - 日志 Modal 组件实现完整
    - 统计数据展示正确
    - 日志表格功能正常
    - 分页功能正确
    - 与主页面集成成功
    - useEffect 依赖使用 useMemo 稳定
    - 样式和交互符合规范
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| 前端状态 | 组件状态需要正确管理，防止状态泄露 |
| API 调用 | API 请求需要正确的错误处理和超时控制 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-16-24 | Information Disclosure | 密钥显示 | mitigate | 列表和详情仅显示前12位，完整密钥仅创建时显示一次 |
| T-16-25 | Tampering | 表单验证 | mitigate | 使用 Form.Item rules 验证输入，防止非法数据提交 |
| T-16-26 | Elevation of Privilege | 权限检查 | mitigate | 后端验证权限，前端仅做 UI 控制 |
| T-16-27 | Denial of Service | 请求限制 | mitigate | 使用分页和防抖，避免大量请求 |
</threat_model>

<verification>
1. 检查所有 TypeScript 文件是否存在且语法正确
2. 运行 npm run type-check 验证类型
3. 验证页面组件渲染正常
4. 验证 CRUD 操作功能完整
5. 验证密钥脱敏显示正确
6. 验证 useEffect 依赖稳定性
</verification>

<success_criteria>
1. 管理页面实现所有 CRUD 功能
2. 密钥脱敏显示正确
3. 创建密钥时完整显示并支持复制
4. 使用日志和统计页面功能完整
5. 表单验证正确
6. 页面样式和交互符合项目规范
7. useEffect 依赖使用 useMemo 稳定，避免无限循环
8. 类型检查通过
</success_criteria>

<output>
执行完成后，创建 .planning/phases/16-api-key-mgt/16-05b-SUMMARY.md
</output>
