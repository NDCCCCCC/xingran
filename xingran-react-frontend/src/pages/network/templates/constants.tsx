/**
 * Network Template Constants
 * 网络模板常量定义
 */

import { Tag } from "antd";
import type { SelectOption } from "./types";

// 厂商选项
export const VENDOR_OPTIONS: SelectOption[] = [
  { label: "Huawei", value: "huawei" },
  { label: "H3C", value: "h3c" },
  { label: "Ruijie", value: "ruijie" },
  { label: "Maipu", value: "maipu" },
];

// 设备类型选项
export const DEVICE_TYPE_OPTIONS: SelectOption[] = [
  { label: "路由器", value: "router" },
  { label: "交换机", value: "switch" },
  { label: "防火墙", value: "firewall" },
  { label: "无线AP", value: "ap" },
  { label: "负载均衡器", value: "loadbalancer" },
];

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

// 渲染设备类型标签
export function renderDeviceTypeTag(deviceType: string) {
  if (!deviceType) return "-";
  const option = DEVICE_TYPE_OPTIONS.find((o) => o.value === deviceType);
  return <Tag color="purple">{option?.label}</Tag>;
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
