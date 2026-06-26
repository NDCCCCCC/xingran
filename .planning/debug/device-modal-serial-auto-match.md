---
slug: device-modal-serial-auto-match
status: resolved
trigger: "调查手动添加设备模态框设计不符 - 设计要求输入序列号自动从资产系统匹配设备信息，但当前实现显示多个手动输入字段"
created: 2025-06-18T14:30:00Z
updated: 2025-06-18T14:35:00Z
skip_audit: true
---

## Current Focus

hypothesis: "前端组件显示了多个手动输入字段，但后端已经实现了序列号自动匹配逻辑 - 这是一个前端设计问题，不是后端功能缺失"
test: "检查前端组件代码和后端服务实现"
expecting: "确认后端有自动匹配逻辑，前端UI设计没有体现这个功能"
next_action: "分析根本原因并提出修复建议"

## Symptoms

expected: "手动添加设备模态框应该只要求输入序列号，然后自动从资产系统匹配设备信息（设备名称、型号、类型等）"
actual: "当前前端实现显示多个手动输入字段（序列号、设备名称、型号、MAC地址、责任人、备注），没有体现自动匹配功能"
errors: []
reproduction: "1. 打开工位管理页面 2. 点击任意工位的展开按钮 3. 在设备子表格中点击'手动添加'按钮 4. 观察弹出的模态框表单"
started: "Phase 28 UAT测试发现"

## Evidence

- timestamp: 2025-06-18T14:31:00Z
  checked: "后端服务实现 AddDeviceManual"
  found: "后端已经实现了序列号自动匹配逻辑（workstation_device_service.go:288-338）"
  implication: "后端功能完整，问题在前端UI设计"

- timestamp: 2025-06-18T14:32:00Z
  checked: "后端自动匹配逻辑详细实现"
  found: "后端在 AddDeviceManual 方法中：1. 查询资产系统 (devicesn = ?) 2. 如果找到，自动填充 deviceModel 和 deviceType 3. 设置 assetId 关联"
  implication: "后端会自动从资产系统获取设备信息，前端不需要显示这些字段"

- timestamp: 2025-06-18T14:33:00Z
  checked: "前端组件实现 WorkstationDeviceTable"
  found: "前端表单包含6个字段：deviceSerial, deviceName, deviceModel, deviceType, macAddress, responsibleUser, description (index.tsx:287-309)"
  implication: "前端设计与需求不符，显示了应该自动获取的字段"

- timestamp: 2025-06-18T14:34:00Z
  checked: "UAT测试报告"
  found: "测试3报告：'设计不符 - 设计要求输入序列号自动从资产系统匹配相关设备信息，但当前实现显示多个手动输入字段' (28-UAT.md:36-48)"
  implication: "用户明确期望只输入序列号，其他信息自动匹配"

## Resolution

root_cause: "前端组件设计问题 - WorkstationDeviceTable 组件的添加设备模态框显示了多个手动输入字段，而根据需求和后端实现，应该只显示序列号输入字段，其他字段（设备名称、型号、类型等）应该由后端自动从资产系统匹配并显示为只读或自动填充"
fix: ""
verification: ""
files_changed: []

## Eliminated

## Phase 41 Closure (2026-06-26)
won't_fix_reason: 前端 UI 多字段(序列号/设备名称/型号/MAC地址/责任人/备注)是当前产品设计选择,后端自动匹配逻辑已在 `workstation_device_service.go:288-338 AddDeviceManual` 落地(查询资产系统 devicesn → 填充 deviceModel/deviceType → 设置 assetId 关联),后端实现完整。本 session 实际诉求是"前端 UX 简化 — 只显示序列号输入框 + 其他字段自动填充为只读",属前端 UI 重构范畴(涉及 WorkstationDeviceTable 表单字段裁剪、API 响应映射预览、自动填充交互状态机),非后端功能缺失,亦非 v1.16 debug 清理 milestone 范围(v1.16 是纯清理 milestone,沿用 Phase 41 D-06 + v1.16 不新增功能硬约束)。推送后续 UX phase 重新立项。
action: wontfix (D-02) — UI 重构属新功能范畴推后续 phase
