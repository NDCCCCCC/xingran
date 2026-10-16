import { get, post, del } from "@/lib/api";

export interface ColumnConfigItem {
  columnKey: string;
  visible: boolean;
  width?: number;
}

export interface ColumnConfigData {
  pageKey: string;
  columnConfigs: ColumnConfigItem[];
}

export interface UserColumnConfig {
  id: string;
  userId: string;
  pageKey: string;
  columnKey: string;
  visible: boolean;
  displayOrder: number;
  width: number;
  createdAt: string;
  updatedAt: string;
}

export const columnConfigApi = {
  // 获取页面列配置
  getByPageKey: (pageKey: string) => get<UserColumnConfig[]>(`/system/column-config/${pageKey}`),

  // 保存列配置
  save: (data: ColumnConfigData) => post("/system/column-config", data),

  // 重置列配置
  reset: (pageKey: string) => del(`/system/column-config/${pageKey}`),
};
