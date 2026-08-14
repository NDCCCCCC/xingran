/**
 * 动态颜色选择器组件（重构版 - 集成 SettingsStore）
 * Real-time Color Switcher (Refactored - Integrated with SettingsStore)
 *
 * 功能：
 * - 主色调自定义（自动生成浅色变体）
 * - 侧边栏背景色自定义（自动搭配文字颜色）
 * - 实时预览颜色变化
 * - 支持预设颜色和自定义颜色选择
 * - 同步到后端保存用户偏好
 * - 自动对比度计算（深色背景配浅色字体，浅色背景配深色字体）
 */

import { useState, useEffect } from "react";
import { Button, Tooltip, Dropdown, ColorPicker, Space, Typography } from "antd";
import { BgColorsOutlined, MenuOutlined, CheckOutlined } from "@ant-design/icons";
import type { MenuProps, ColorPickerProps } from "antd";
import { useSettingsStore } from "@/store/settingsStore";
import { applyPrimaryColor, applySidebarBackgroundColor } from "@/design-system/themes";
import { getLuminance } from "@/design-system/utils/color";

const { Text } = Typography;

/**
 * 颜色预设类型
 */
export interface ColorPreset {
  name: string;
  primary: string;
  category?: "primary" | "accent" | "neutral" | "custom";
}

/**
 * 预设主色调方案
 */
export const primaryColorPresets: ColorPreset[] = [
  // 主要色系
  { name: "蓝色", primary: "#2563eb", category: "primary" },
  { name: "靛蓝", primary: "#4f46e5", category: "primary" },
  { name: "紫色", primary: "#7c3aed", category: "primary" },
  { name: "粉色", primary: "#db2777", category: "accent" },

  // 功能色系
  { name: "红色", primary: "#dc2626", category: "accent" },
  { name: "橙色", primary: "#ea580c", category: "accent" },
  { name: "琥珀", primary: "#d97706", category: "accent" },
  { name: "绿色", primary: "#16a34a", category: "primary" },

  // 青色系
  { name: "青色", primary: "#0891b2", category: "primary" },
  { name: "蓝绿", primary: "#0d9488", category: "primary" },
];

/**
 * 预设侧边栏颜色方案
 */
export const sidebarColorPresets: ColorPreset[] = [
  // 深色系（配浅色文字）
  { name: "深蓝灰", primary: "#1E293B", category: "neutral" },
  { name: "深灰", primary: "#374151", category: "neutral" },
  { name: "墨黑", primary: "#0F172A", category: "neutral" },
  { name: "靛蓝深", primary: "#1E1B4B", category: "primary" },
  { name: "森林绿", primary: "#14532D", category: "primary" },
  { name: "酒红", primary: "#450A0A", category: "accent" },

  // 浅色系（配深色文字）
  { name: "纯白", primary: "#FFFFFF", category: "neutral" },
  { name: "浅灰", primary: "#F8FAFC", category: "neutral" },
  { name: "象牙白", primary: "#FFFBEB", category: "neutral" },
  { name: "淡蓝", primary: "#EFF6FF", category: "primary" },
  { name: "薄荷绿", primary: "#F0FDF4", category: "primary" },
  { name: "樱花粉", primary: "#FFF1F2", category: "accent" },
];

interface ColorSwitcherProps {
  size?: "small" | "middle" | "large";
  type?: "default" | "picker" | "dropdown";
}

/**
 * 颜色选择器组件
 */
import type { FC } from "react";

const ColorSwitcher: FC<ColorSwitcherProps> = ({ size = "middle", type = "dropdown" }) => {
  const { preferences, updateTheme } = useSettingsStore();

  // 从 preferences 读取当前颜色（如果有自定义颜色）
  // 确保颜色值始终是字符串格式
  const primaryColor = (
    typeof preferences.theme.customColors?.primary === "string"
      ? preferences.theme.customColors.primary
      : "#4F46E5"
  ) as string;
  const sidebarColor = (
    typeof preferences.theme.customColors?.sidebar === "string"
      ? preferences.theme.customColors.sidebar
      : "#1E293B"
  ) as string;
  const [isOpen, setIsOpen] = useState(false);

  // 初始化：如果有自定义颜色，应用它们
  useEffect(() => {
    if (
      preferences.theme.customColors?.primary &&
      typeof preferences.theme.customColors.primary === "string"
    ) {
      applyPrimaryColor(preferences.theme.customColors.primary);
    }
    if (
      preferences.theme.customColors?.sidebar &&
      typeof preferences.theme.customColors.sidebar === "string"
    ) {
      applySidebarBackgroundColor(preferences.theme.customColors.sidebar);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional mount-only initialization
  }, []);

  /**
   * 主色调变化处理
   */
  const handlePrimaryColorChange: ColorPickerProps["onChange"] = (color) => {
    const hexColor = typeof color === "string" ? color : color.toHexString();

    // 立即预览
    applyPrimaryColor(hexColor);

    // 保存到后端
    updateTheme({
      customColors: {
        ...preferences.theme.customColors,
        primary: hexColor,
      },
    });
  };

  /**
   * 侧边栏颜色变化处理
   */
  const handleSidebarColorChange: ColorPickerProps["onChange"] = (color) => {
    const hexColor = typeof color === "string" ? color : color.toHexString();

    // 立即预览
    applySidebarBackgroundColor(hexColor);

    // 保存到后端
    updateTheme({
      customColors: {
        ...preferences.theme.customColors,
        sidebar: hexColor,
      },
    });
  };

  /**
   * 获取当前颜色的名称
   */
  const getCurrentColorName = () => {
    const preset = primaryColorPresets.find(
      (p) => p.primary.toLowerCase() === primaryColor.toLowerCase()
    );
    if (preset) return preset.name;
    return "自定义";
  };

  /**
   * 获取侧边栏颜色名称
   */
  const getSidebarColorName = () => {
    const preset = sidebarColorPresets.find(
      (p) => p.primary.toLowerCase() === sidebarColor.toLowerCase()
    );
    if (preset) return preset.name;
    return "自定义";
  };

  /**
   * 预设主色调选择
   */
  const handlePrimaryPresetSelect = (preset: ColorPreset) => {
    // 立即预览
    applyPrimaryColor(preset.primary);

    // 保存到后端
    updateTheme({
      customColors: {
        ...preferences.theme.customColors,
        primary: preset.primary,
      },
    });
  };

  /**
   * 预设侧边栏颜色选择
   */
  const handleSidebarPresetSelect = (preset: ColorPreset) => {
    // 立即预览
    applySidebarBackgroundColor(preset.primary);

    // 保存到后端
    updateTheme({
      customColors: {
        ...preferences.theme.customColors,
        sidebar: preset.primary,
      },
    });
  };

  // ===== 渲染模式 =====

  /**
   * 模式1: 仅颜色选择器（主色调）
   */
  if (type === "picker") {
    return (
      <Tooltip title={`主题色：${getCurrentColorName()}`}>
        <ColorPicker
          value={primaryColor}
          onChange={handlePrimaryColorChange}
          showText
          format="hex"
          size={size}
          style={{
            display: "flex",
            alignItems: "center",
          }}
        >
          <Button
            type="text"
            size={size}
            icon={<BgColorsOutlined />}
            style={{
              display: "flex",
              alignItems: "center",
              gap: "8px",
            }}
          >
            <div
              style={{
                width: "16px",
                height: "16px",
                borderRadius: "4px",
                background: primaryColor,
                border: "1px solid rgba(0,0,0,0.1)",
              }}
            />
          </Button>
        </ColorPicker>
      </Tooltip>
    );
  }

  /**
   * 模式2: 下拉菜单 + 主色调和侧边栏颜色选择
   */
  const menuItems: MenuProps["items"] = [
    // 主色调选择
    {
      key: "primary-header",
      label: (
        <div style={{ padding: "8px 0" }}>
          <Space>
            <BgColorsOutlined style={{ color: primaryColor }} />
            <Text strong>主题色</Text>
          </Space>
        </div>
      ),
      disabled: true,
    },
    {
      type: "group",
      label: "预设主题色",
      children: primaryColorPresets.map((preset) => ({
        key: `primary-${preset.primary}`,
        label: (
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "10px",
              justifyContent: "space-between",
            }}
          >
            <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
              <div
                style={{
                  width: "20px",
                  height: "20px",
                  borderRadius: "4px",
                  background: preset.primary,
                  border: "1px solid rgba(0,0,0,0.1)",
                }}
              />
              <span>{preset.name}</span>
            </div>
            {primaryColor.toLowerCase() === preset.primary.toLowerCase() && (
              <CheckOutlined style={{ color: primaryColor, fontSize: "12px" }} />
            )}
          </div>
        ),
        onClick: () => handlePrimaryPresetSelect(preset),
      })),
    },
    {
      key: "primary-custom",
      label: (
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "4px 0",
          }}
        >
          <Text type="secondary">自定义主题色</Text>
          <ColorPicker
            value={primaryColor}
            onChange={handlePrimaryColorChange}
            showText={false}
            format="hex"
            size="small"
            trigger="click"
          />
        </div>
      ),
    },
    {
      type: "divider",
    },
    // 侧边栏颜色选择
    {
      key: "sidebar-header",
      label: (
        <div style={{ padding: "8px 0" }}>
          <Space>
            <MenuOutlined
              style={{ color: getLuminance(sidebarColor) < 0.5 ? sidebarColor : "#8B5CF6" }}
            />
            <Text strong>侧边栏</Text>
          </Space>
        </div>
      ),
      disabled: true,
    },
    {
      type: "group",
      label: "深色背景（配浅色文字）",
      children: sidebarColorPresets
        .filter((p) => getLuminance(p.primary) < 0.5)
        .map((preset) => ({
          key: `sidebar-${preset.primary}`,
          label: (
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: "10px",
                justifyContent: "space-between",
              }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                <div
                  style={{
                    width: "20px",
                    height: "20px",
                    borderRadius: "4px",
                    background: preset.primary,
                    border: preset.primary === "#FFFFFF" ? "1px solid #E2E8F0" : "none",
                  }}
                />
                <span>{preset.name}</span>
              </div>
              {sidebarColor.toLowerCase() === preset.primary.toLowerCase() && (
                <CheckOutlined style={{ color: primaryColor, fontSize: "12px" }} />
              )}
            </div>
          ),
          onClick: () => handleSidebarPresetSelect(preset),
        })),
    },
    {
      type: "group",
      label: "浅色背景（配深色文字）",
      children: sidebarColorPresets
        .filter((p) => getLuminance(p.primary) >= 0.5)
        .map((preset) => ({
          key: `sidebar-${preset.primary}`,
          label: (
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: "10px",
                justifyContent: "space-between",
              }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                <div
                  style={{
                    width: "20px",
                    height: "20px",
                    borderRadius: "4px",
                    background: preset.primary,
                    border: preset.primary === "#FFFFFF" ? "1px solid #E2E8F0" : "none",
                  }}
                />
                <span>{preset.name}</span>
              </div>
              {sidebarColor.toLowerCase() === preset.primary.toLowerCase() && (
                <CheckOutlined style={{ color: primaryColor, fontSize: "12px" }} />
              )}
            </div>
          ),
          onClick: () => handleSidebarPresetSelect(preset),
        })),
    },
    {
      key: "sidebar-custom",
      label: (
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "4px 0",
          }}
        >
          <Text type="secondary">自定义侧边栏色</Text>
          <ColorPicker
            value={sidebarColor}
            onChange={handleSidebarColorChange}
            showText={false}
            format="hex"
            size="small"
            trigger="click"
          />
        </div>
      ),
    },
  ];

  return (
    <Dropdown
      menu={{ items: menuItems }}
      placement="bottomLeft"
      trigger={["click"]}
      open={isOpen}
      onOpenChange={setIsOpen}
    >
      <Tooltip title={`主题色：${getCurrentColorName()} | 侧边栏：${getSidebarColorName()}`}>
        <Button
          type="text"
          size={size}
          icon={
            <Space size={4}>
              <BgColorsOutlined style={{ color: primaryColor }} />
              <MenuOutlined
                style={{ color: getLuminance(sidebarColor) < 0.5 ? sidebarColor : "#8B5CF6" }}
              />
              <div
                style={{
                  width: "12px",
                  height: "12px",
                  borderRadius: "3px",
                  background: primaryColor,
                  border: "1px solid rgba(0,0,0,0.15)",
                }}
              />
            </Space>
          }
        />
      </Tooltip>
    </Dropdown>
  );
};

export default ColorSwitcher;
