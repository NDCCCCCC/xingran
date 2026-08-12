/**
 * 主题切换器组件
 * 提供用户界面来切换主题
 */

import { useState } from "react";
import type { FC } from "react";
import { Dropdown, Button, Space, Tooltip } from "antd";
import { useTheme } from "@/store/themeStore";
import { useSettingsStore } from "@/store/settingsStore";
import { themePresets } from "@/design-system/themes";
import type { MenuProps } from "antd";

const ThemeSwitcher: FC = () => {
  const { theme } = useTheme();
  const { updateTheme } = useSettingsStore();
  const [open, setOpen] = useState(false);

  // 构建菜单项
  const menuItems: MenuProps["items"] = themePresets.map((preset) => ({
    key: preset.id,
    label: (
      <div className="flex items-center justify-between gap-3">
        <Space size="small">
          <span className="text-lg">{preset.icon}</span>
          <span>{preset.name}</span>
        </Space>
        {theme === preset.id && (
          <span className="text-blue-500">✓</span>
        )}
      </div>
    ),
    onClick: () => {
      updateTheme({ style: preset.id });
      setOpen(false);
    },
  }));

  return (
    <Dropdown
      menu={{ items: menuItems }}
      trigger={["click"]}
      open={open}
      onOpenChange={setOpen}
      placement="bottomRight"
    >
      <Tooltip title="切换主题">
        <Button
          type="text"
          icon={<span style={{ fontSize: "16px" }}>{themePresets.find(p => p.id === theme)?.icon}</span>}
          className="flex items-center gap-2"
          style={{
            transition: "var(--theme-transition-base)",
          }}
        >
          <span className="hidden sm:inline">主题</span>
        </Button>
      </Tooltip>
    </Dropdown>
  );
};

export default ThemeSwitcher;
