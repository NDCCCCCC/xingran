/**
 * 设备搜索区 schema 构造器（v1.24 HYG-04 / D2）。
 *
 * 从 index.tsx 原样搬迁（零行为变化）：字段名、label、placeholder、options 来源逐字保留。
 *
 * ⚠️ 为什么是「函数」而不是常量：
 *   `deviceType` 的 options 来自 `useDict("network_device_type")`（Phase 69 DICT-03），
 *   是**动态**数据（字典空态/接口异常时回退静态 DEVICE_TYPE_OPTIONS）。
 *   因此本文件不能是纯常量表，也不放进 constants.ts —— 那会破坏 constants.ts
 *   「只放静态选项/常量」的语义（见 .planning/research/ARCHITECTURE.md §5.1）。
 *
 * ⚠️ 调用点纪律：本构造器**每次渲染调用一次**（与搬迁前 searchFields 每次渲染
 *   新建数组的语义一致）—— 刻意不做 useMemo 包装，避免改变 PageTable
 *   searchFields 的引用稳定性语义（v1.23 FAIL-1：引用不稳 → 页码被打回第 1 页）。
 */

import type { SearchField } from "@/components/shared";
import { STATUS_OPTIONS, VENDOR_OPTIONS } from "./constants";

/**
 * 构造设备搜索区 schema。
 *
 * @param deviceTypeOptions 设备类型下拉项（字典优先，回退静态 DEVICE_TYPE_OPTIONS）
 */
export function buildDeviceSearchFields(
  deviceTypeOptions: Array<{ label: string; value: string | number }>
): SearchField[] {
  return [
    {
      name: "deviceName",
      label: "设备名称",
      type: "input",
      placeholder: "请输入设备名称",
    },
    {
      name: "ipAddress",
      label: "IP地址",
      type: "input",
      placeholder: "请输入IP地址",
    },
    {
      name: "vendor",
      label: "厂商",
      type: "select",
      placeholder: "请选择厂商",
      options: VENDOR_OPTIONS.map((opt) => ({ label: opt.label, value: opt.value })),
    },
    {
      name: "deviceType",
      label: "设备类型",
      type: "select",
      placeholder: "请选择设备类型",
      options: deviceTypeOptions,
    },
    {
      name: "status",
      label: "状态",
      type: "select",
      placeholder: "请选择状态",
      options: STATUS_OPTIONS.map((opt) => ({ label: opt.label, value: opt.value })),
    },
  ];
}
