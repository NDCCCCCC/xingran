---
status: resolved
trigger: 刚才删除虚拟机列表的名称字段引起了故障，我不需要的是在系统中输入名称，但是还是需要从vdi服务器获取虚拟机名称！请修复！
created: 2026-06-02T00:00:00Z
updated: 2026-06-02T00:00:00Z
slug: vdi-vm-name-display-empty
---

# Symptom Report

## Expected Behavior
名称为只读显示（从 VDI 服务器获取）

## Actual Behavior
名称显示为空

## Error Messages
没有明显错误

## Timeline
刚才/最近

## Reproduction Steps
1. 删除虚拟机列表的名称字段
2. 查看虚拟机列表页面
3. 虚拟机名称显示为空

# Current Focus
hypothesis: confirmed
next_action: fix applied
test: go build + go test
expecting: pass

# Evidence
- timestamp: 2026-06-02T00:00:00Z
  source: user_report
  observation: 名称显示为空
  data:
    - 实际行为: 名称显示为空
    - 预期行为: 名称为只读显示（从 VDI 服务器获取）
    - 时间线: 刚才/最近
    - 错误消息: 没有明显错误

- timestamp: 2026-06-02T00:00:01Z
  source: code_analysis
  observation: >
    两个 bug:
    1. SyncVMFromVDI 函数在同步单个虚拟机时没有更新 Name 字段
    2. saveOrUpdateVM 中缩进错误导致智能 IP 更新逻辑被嵌套在 ApplyUser 的 if 块内
    3. CreateVM 仍使用 req.Name 设置初始名称，但用户已删除名称输入字段
  data:
    - 文件: internal/services/vdi/vm_service_impl.go
    - SyncVMFromVDI 第 867-883 行: 缺少 vm.Name = targetVM.VMName
    - saveOrUpdateVM 第 344-365 行: if ApplyUser 块未正确关闭
    - CreateVM 第 528 行: Name: req.Name 应改为空字符串

# Eliminated
- 前端列定义正常: dataIndex: 'name' 存在于 VirtualMachineList/index.tsx 第 744 行
- VDI API 类型定义正常: VDIVMResource.VMName 字段存在于 vdi_types.go 第 166 行
- 数据库模型正常: VDIVirtualMachine.Name 字段存在于 models/vdi.go 第 16 行

# Investigation History
1. 检查 git log 发现提交 9b754b6 "移除虚拟机重命名功能和名称自动生成逻辑" 是最近的修改
2. 分析 saveOrUpdateVM 函数: 名称在完整同步时正确设置为 resource.VMName
3. 分析 SyncVMFromVDI 函数: 发现缺少名称同步
4. 发现 saveOrUpdateVM 缩进错误导致 IP 地址更新逻辑被嵌套在错误的 if 块内
5. 发现 CreateVM 仍使用用户输入的名称

# Resolution
root_cause: >
  SyncVMFromVDI（单虚拟机同步）函数在更新本地记录时没有同步 Name 字段，
  仅同步了 PowerState、IPAddress、MACAddress 等。
  此外 saveOrUpdateVM 中存在缩进错误，将智能 IP 更新逻辑错误地嵌套在
  if resource.ApplyUser != "" 块内，导致无绑定用户的虚拟机在同步时无法正确更新 IP 地址。
fix: >
  1. 在 SyncVMFromVDI 中添加 vm.Name = targetVM.VMName（第 866 行）
  2. 修复 saveOrUpdateVM 中 if ApplyUser 块的关闭大括号位置（第 344 行），
     将智能 IP 更新逻辑移出该 if 块
  3. CreateVM 中将初始名称设为空字符串，名称完全从 VDI 同步获取
files:
  - internal/services/vdi/vm_service_impl.go
