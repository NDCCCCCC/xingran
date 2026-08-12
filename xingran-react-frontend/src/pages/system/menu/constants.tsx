/**
 * Menu Constants
 * 菜单管理页面常量定义
 */

import type { ReactElement } from "react";
import { FolderOutlined, FileOutlined, AppstoreOutlined, MenuOutlined } from "@ant-design/icons";
import { Tag } from "antd";

/** 菜单类型选项 */
export const MENU_TYPE_OPTIONS = [
  { label: "目录", value: "M" },
  { label: "菜单", value: "C" },
  { label: "按钮", value: "F" },
] as const;

/** 菜单状态选项 */
export const MENU_STATUS_OPTIONS = [
  { label: "正常", value: "0" },
  { label: "停用", value: "1" },
] as const;

/** 菜单类型 */
export type MenuType = "M" | "C" | "F";

/** 获取菜单类型图标 */
export function getMenuIcon(menuType: MenuType): ReactElement {
  switch (menuType) {
    case "M":
      return <FolderOutlined />;
    case "C":
      return <FileOutlined />;
    case "F":
      return <AppstoreOutlined />;
    default:
      return <MenuOutlined />;
  }
}

/** 获取菜单类型标签 */
export function getMenuTypeTag(menuType: MenuType): ReactElement {
  switch (menuType) {
    case "M":
      return <Tag color="blue">目录</Tag>;
    case "C":
      return <Tag color="green">菜单</Tag>;
    case "F":
      return <Tag color="orange">按钮</Tag>;
    default:
      return <Tag>未知</Tag>;
  }
}

/** 默认表单值 */
export const DEFAULT_FORM_VALUES = {
  parentId: "",
  menuType: "M" as MenuType,
  orderNum: 0,
  status: true,
  visible: true,
};
