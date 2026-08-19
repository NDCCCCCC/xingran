/**
 * Menu Constants
 * 菜单管理页面常量定义
 */

import type { ReactElement } from "react";
import { FolderOutlined, FileOutlined, AppstoreOutlined, MenuOutlined } from "@ant-design/icons";
import { Tag } from "antd";
import { NORMAL_STOP_OPTIONS } from "@/constants/status";

/** 菜单类型选项 */
export const MENU_TYPE_OPTIONS = [
  { label: "目录", value: "M" },
  { label: "菜单", value: "C" },
  { label: "按钮", value: "F" },
] as const;

/**
 * 菜单状态选项（Phase 69 DICT-03: 语义对齐共享常量 models.MenuStatusNormal=0/MenuStatusStop=1）
 * 保留字符串 value 形态（"0"/"1"）——搜索表单既有契约，页面零改动。
 */
export const MENU_STATUS_OPTIONS = NORMAL_STOP_OPTIONS.map((opt) => ({
  label: opt.label,
  value: String(opt.value),
}));

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
  // VISIBLE 反转例外（严禁与 status 0/1 统一）:
  // 对齐 models.VisibleShow=1(显示) / VisibleHidden=0(隐藏), internal/models/base.go
  // —— 与通用启停「0=正常/1=停用」方向相反；表单内以 boolean 承载，提交侧转换。
  visible: true,
};
