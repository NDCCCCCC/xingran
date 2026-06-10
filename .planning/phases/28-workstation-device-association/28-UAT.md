---
status: testing
phase: 28-workstation-device-association
source: [28-01-SUMMARY.md, 28-EXECUTE-SUMMARY.md]
started: 2026-06-10T10:00:00Z
updated: 2026-06-10T10:00:00Z
---

## Current Test

[testing complete - blocked by API compilation error]

## Test Results Summary

✅ **Passed**: 2/11
- Test 1: 工位列表页面加载 (blocked - 编译错误)
- Test 2: 展开设备子表格 (pass)
- Test 3: 手动添加设备模态框 (pass)
- Tests 4-11: Skipped (依赖 API)

**Blocking Issue**: internal/agent/server 编译错误阻止后端正确编译

## Tests

### 1. 工位列表页面加载
expected: |
  打开浏览器，访问工位管理页面（通常在 /operations/workstations 或类似路径）。
  
  预期看到：
  - 工位列表正常加载，显示工位数据（如果没有数据，页面应该显示空状态）
  - 每行工位数据前面有一个"查看设备"或类似的展开按钮
  - 页面没有 JavaScript 错误（按 F12 打开开发者工具查看 Console）
result: blocked
blocked_by: server
reason: internal/agent/server 包有编译错误（WithRequestID redeclared），阻止了后端正确编译，导致 API 端点未注册。用户选择跳过，等待编译错误修复后再测试 API 功能。

### 2. 展开设备子表格
expected: |
  点击任意工位行的"查看设备"按钮。
  
  预期看到：
  - 该行下方展开一个子表格区域
  - 子表格显示"关联设备"相关的表头（设备名称、序列号、来源、型号等）
  - 展开按钮文字变为"收起设备"
  - 展开区域有适当的背景色和间距
result: pass

### 3. 手动添加设备
expected: |
  在展开的设备子表格中，点击"手动添加"或"添加设备"按钮。
  
  预期看到：
  - 弹出一个模态框
  - 模态框包含表单字段：序列号（必填）、设备名称、型号、MAC地址、责任人、备注
  - 序列号字段有红色星号标记表示必填
  - 点击"取消"按钮可以关闭模态框
result: pass

### 4. 序列号匹配资产信息
expected: |
  在添加设备模态框中，输入一个已存在于资产系统的设备序列号，填写其他字段后点击"确定"。
  
  预期看到：
  - 设备添加成功
  - 如果该序列号在资产系统中存在，设备型号、类型等信息会自动填充
  - 子表格中显示新添加的设备
  - 显示成功提示消息
result: pending

### 5. 同步资产设备
expected: |
  在展开的设备子表格中，点击"同步资产设备"按钮。
  
  预期看到：
  - 系统调用 API 查询该工位绑定用户（责任人）名下的资产设备
  - 删除现有的"asset"来源设备
  - 添加用户负责的所有资产设备
  - 显示成功提示消息
  - 子表格刷新显示同步后的设备列表
  
  注意：这需要工位已绑定用户，且该用户在资产系统中有负责的设备
result: pending

### 6. 同步域控设备
expected: |
  在展开的设备子表格中，点击"同步AD设备"或"同步域控设备"按钮。
  
  预期看到：
  - 系统调用 API 查询域控设备
  - 删除现有的"ad"来源设备
  - 添加域控同步的设备
  - 显示成功提示消息
  - 子表格刷新显示同步后的设备列表
  
  注意：这可能需要工位绑定用户在域控中有设备记录
result: pending

### 7. 编辑设备
expected: |
  在设备子表格中，找到"来源"为"手动"或"资产"的设备，点击"编辑"按钮。
  
  预期看到：
  - 弹出编辑模态框，预填充该设备的现有信息
  - 可以修改设备名称、型号、MAC地址等字段
  - 保存后设备信息更新
  - 显示成功提示消息
  
  注意：域控来源（AD）的设备应该禁用编辑按钮
result: pending

### 8. 删除设备
expected: |
  在设备子表格中，找到"来源"为"手动"或"资产"的设备，点击"删除"按钮。
  
  预期看到：
  - 显示确认对话框（如"确定要删除这个设备吗？"）
  - 点击"确定"后，设备从列表中移除
  - 显示成功提示消息
  
  注意：域控来源（AD）的设备应该禁用删除按钮
result: pending

### 9. 设置主设备
expected: |
  在设备子表格中，找到一个没有主设备星标的设备，点击"设为主设备"按钮。
  
  预期看到：
  - 该设备旁边出现星标图标（★）
  - 之前的主设备（如果有）的星标消失
  - 只有一个设备显示主设备星标
  - 显示成功提示消息
result: pending

### 10. 设备来源标签显示
expected: |
  查看设备子表格中的"来源"列。
  
  预期看到：
  - 域控来源显示蓝色"域控"标签
  - 资产来源显示绿色"资产"标签
  - 手动添加显示橙色"手动"标签
  - 标签颜色区分明显，易于识别
result: pending

### 11. 收起设备子表格
expected: |
  点击"收起设备"按钮。
  
  预期看到：
  - 设备子表格区域收起隐藏
  - 按钮文字变回"查看设备"
  - 工位列表恢复正常显示
result: pending

## Summary

total: 11
passed: 2
issues: 0
pending: 7
skipped: 0
blocked: 1
note: 剩余测试跳过（依赖API）

## Gaps

- truth: 工位设备关联 API 端点应该可以正常访问
  status: failed
  reason: User reported: POST http://10.62.10.33:9000/api/v1/ops/workstation-device/b96f84f2-31ae-45e1-ba83-0d06b72947cb 404 (Not Found)
  severity: blocker
  test: 1
  artifacts: []
  missing: []

- truth: WorkstationDeviceTable 不应使用已弃用的 Ant Design API
  status: failed
  reason: User reported: `destroyOnClose` is deprecated, use `destroyOnHidden` instead
  severity: cosmetic
  test: 1
  artifacts: []
  missing: []
