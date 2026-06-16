/**
 * NetworkDevice 常量定义
 */

export interface SelectOption {
  label: string;
  value: string | number;
}

// 设备厂商选项
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

// 设备状态选项
export const STATUS_OPTIONS: SelectOption[] = [
  { label: "在线", value: 0 },
  { label: "离线", value: 1 },
  { label: "未知", value: 2 },
];

// 状态颜色映射
export const STATUS_COLOR_MAP: Record<number, string> = {
  0: "success",
  1: "error",
  2: "default",
};

// 设备类型标签颜色
export const DEVICE_TYPE_TAG_COLOR = "blue";

// 厂商标签颜色
export const VENDOR_TAG_COLOR = "geekblue";
