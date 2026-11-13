/**
 * Network Template Constants
 * 网络模板常量定义
 */

import { Tag } from "antd";
import type { SelectOption } from "./types";
import type { DictItem } from "@/hooks/useDict";
import { buildDictLabelIndex, resolveDictLabel } from "@/components/reconciliation/dictLabels";

// 厂商选项
export const VENDOR_OPTIONS: SelectOption[] = [
  { label: "Huawei", value: "huawei" },
  { label: "H3C", value: "h3c" },
  { label: "Ruijie", value: "ruijie" },
  { label: "Maipu", value: "maipu" },
  { label: "Cisco", value: "cisco" },
];

// 设备类型选项
export const DEVICE_TYPE_OPTIONS: SelectOption[] = [
  { label: "路由器", value: "router" },
  { label: "交换机", value: "switch" },
  { label: "防火墙", value: "firewall" },
  { label: "无线AP", value: "ap" },
  { label: "负载均衡器", value: "loadbalancer" },
];

/**
 * 设备类型静态回退文案（dictValue → 中文文案）。
 * 从 DEVICE_TYPE_OPTIONS 派生，避免重复维护一份硬编码映射；
 * 仅在字典为空 / 未命中时使用，正常路径以字典 network_device_type 为准。
 */
export const DEVICE_TYPE_LABELS: Record<string, string> = DEVICE_TYPE_OPTIONS.reduce<
  Record<string, string>
>((labels, option) => {
  labels[option.value] = option.label;
  return labels;
}, {});

// 模板类型选项
export const TEMPLATE_TYPE_OPTIONS: SelectOption[] = [
  { label: "初始化配置", value: "init" },
  { label: "业务配置", value: "config" },
  { label: "备份配置", value: "backup" },
];

// 渲染厂商标签
export function renderVendorTag(vendor: string) {
  if (!vendor) return <Tag>通用</Tag>;
  const option = VENDOR_OPTIONS.find((o) => o.value === vendor);
  return <Tag color="geekblue">{option?.label}</Tag>;
}

/**
 * 三级回退解析设备类型文案：
 *   1) 字典命中（dictValue → dictLabel）优先 —— 管理员在字典管理页可维护；
 *   2) 静态 DEVICE_TYPE_OPTIONS 映射兜底 —— 字典空态 / 接口异常时保持原展示；
 *   3) 两者皆无显示原始值，空值统一 "-"（不吞成空串）。
 * 本文件保持纯常量 / 纯函数，字典由页面层 useDict 传入，常量文件内不调 hook。
 */
export function resolveDeviceTypeLabel(
  deviceType: string | null | undefined,
  dictItems?: DictItem[]
): string {
  const { labels } = buildDictLabelIndex(dictItems);
  // 字典项已命中但 dictLabel 为空时视为脏数据：剔除该项，让静态映射 fallback 生效，
  // 避免管理员误配空文案把新类型吞成原始值 / 空串（三级回退口径不越级）。
  const effectiveLabels = { ...labels };
  for (const key of Object.keys(effectiveLabels)) {
    if (!effectiveLabels[key]) delete effectiveLabels[key];
  }
  return resolveDictLabel(deviceType, effectiveLabels, DEVICE_TYPE_LABELS) || deviceType || "-";
}

// 渲染设备类型标签（dictItems 可选：不传则退化为静态映射，行为与迁移前一致）
export function renderDeviceTypeTag(deviceType: string, dictItems?: DictItem[]) {
  if (!deviceType) return "-";
  return <Tag color="purple">{resolveDeviceTypeLabel(deviceType, dictItems)}</Tag>;
}

// 渲染模板类型标签
export function renderTemplateTypeTag(templateType: string) {
  const option = TEMPLATE_TYPE_OPTIONS.find((o) => o.value === templateType);
  return <Tag color="blue">{option?.label}</Tag>;
}

// 渲染系统模板标签
export function renderSystemTemplateTag(isSystem: boolean) {
  return isSystem ? <Tag color="gold">系统</Tag> : <Tag>自定义</Tag>;
}
