---
slug: dot1x-interface-name-format-mism
status: resolved
trigger: dot1x字段没有显示。TextFSM日志显示已捕获接口名 GigabitEthernet0/0/1，但数据库没有保存。问题：数据库存储的是简写格式 GE0/0/1，而TextFSM捕获的是完整格式 GigabitEthernet0/0/1，两者不匹配。需要添加格式转换逻辑。
created: 2026-05-12
updated: 2026-05-12
---

# Debug: Dot1x Interface Name Format Mismatch

## Symptoms

- **Expected**: dot1x 字段应该显示在数据库中
- **Actual**: dot1x 字段没有显示，只有部分端口被保存
- **Error**: 部分接口（如45端口）没有被处理和保存
- **Timeline**: TextFSM 修复后，捕获所有接口但只有部分保存到数据库
- **Reproduction**: 对启用 dot1x 的华为设备执行端口采集

## Current Focus

- **hypothesis**: 接口名格式转换缺失 — TextFSM 捕获完整接口名但数据库使用简写格式
- **next_action**: resolved

## Evidence

- 2026-05-12: 接口名标准化已存在于 `normalizeInterfaceName` 函数中
- 2026-05-12: `getAllDot1xStatus` 函数正确调用 `normalizeInterfaceName`
- 2026-05-12: 发现第一个问题：TextFSM 模板没有捕获 `DOT1X_ENABLED`
- 2026-05-12: 修复模板后，所有接口的 dot1x 数据被正确捕获
- 2026-05-12: 用户报告：只有46端口保存，45端口没有
- 2026-05-12: 根因：华为设备只使用 `descriptionMap` 作为数据源，某些接口可能没有描述信息

## Eliminated

- 接口名格式不匹配 — 排除，代码已正确处理标准化
- normalizeInterfaceName 函数问题 — 排除
- TextFSM 模板捕获问题 — 已修复，现在正确捕获所有接口

## Resolution

- **root_cause**: 华为/H3C 设备的端口采集逻辑只使用 `descriptionMap`（来自 `display interface description`），但某些接口可能没有描述信息，因此不会出现在 `descriptionMap` 中，导致这些接口即使有 dot1x 数据也不会被处理和保存
- **fix**: 修改华为/H3C 设备的处理逻辑，改用 `parseInterfaceList` 获取完整接口列表作为基准，然后补充 `descriptionMap` 中的信息。这样确保所有接口（包括只有 dot1x 数据的接口）都会被处理
- **files_changed**:
  - `templates/huawei_vrp_display_dot1x.textfsm` -- 修复 DOT1X_ENABLED 捕获规则
  - `internal/services/portcollection/collection.go` -- 华为/H3C 改用接口列表作为基准

