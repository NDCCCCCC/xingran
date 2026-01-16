/**
 * 动态颜色选择器组件
 * 实时调整整个主题色调和侧边栏颜色
 *
 * 功能：
 * - 主色调自定义（自动生成浅色变体）
 * - 侧边栏背景色自定义（自动搭配文字颜色）
 * - 实时预览颜色变化
 * - 支持预设颜色和自定义颜色选择
 * - 持久化用户偏好
 * - 自动对比度计算（深色背景配浅色字体，浅色背景配深色字体）
 */

import { useState, useLayoutEffect } from 'react';
import { Button, Tooltip, Dropdown, ColorPicker, Space, Typography } from 'antd';
import { BgColorsOutlined, MenuOutlined, CheckOutlined } from '@ant-design/icons';
import type { MenuProps, ColorPickerProps } from 'antd';

const { Text } = Typography;

/**
 * 颜色预设类型
 */
export interface ColorPreset {
  name: string;
  primary: string;
  category?: 'primary' | 'accent' | 'neutral' | 'custom';
}

/**
 * 计算颜色的亮度（用于判断是深色还是浅色）
 * 返回 0-1 之间的值，0 为最暗，1 为最亮
 */
function getLuminance(hexColor: string): number {
  const hex = hexColor.replace('#', '');
  const r = parseInt(hex.substring(0, 2), 16) / 255;
  const g = parseInt(hex.substring(2, 4), 16) / 255;
  const b = parseInt(hex.substring(4, 6), 16) / 255;

  // 使用 sRGB 亮度公式
  const toLinear = (c: number) => {
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
  };

  const R = toLinear(r);
  const G = toLinear(g);
  const B = toLinear(b);

  return 0.2126 * R + 0.7152 * G + 0.0722 * B;
}

/**
 * 根据背景色自动选择合适的文字颜色
 * 深色背景（亮度 < 0.5）返回浅色文字
 * 浅色背景（亮度 >= 0.5）返回深色文字
 */
function getContrastTextColor(backgroundColor: string): string {
  const luminance = getLuminance(backgroundColor);

  if (luminance < 0.5) {
    // 深色背景，返回浅色文字（白色到浅灰）
    return '#F8FAFC';
  } else {
    // 浅色背景，返回深色文字
    return '#1E293B';
  }
}

/**
 * 生成悬停背景色
 * 深色背景变亮，浅色背景变暗
 */
function getHoverBackgroundColor(backgroundColor: string): string {
  const hex = backgroundColor.replace('#', '');
  let r = parseInt(hex.substring(0, 2), 16);
  let g = parseInt(hex.substring(2, 4), 16);
  let b = parseInt(hex.substring(4, 6), 16);

  const luminance = getLuminance(backgroundColor);

  if (luminance < 0.5) {
    // 深色背景，变亮
    r = Math.min(255, r + 20);
    g = Math.min(255, g + 20);
    b = Math.min(255, b + 20);
  } else {
    // 浅色背景，变暗
    r = Math.max(0, r - 20);
    g = Math.max(0, g - 20);
    b = Math.max(0, b - 20);
  }

  return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`;
}

/**
 * 从主色自动生成浅色变体
 * 使用 HSL 颜色空间进行亮度调整
 */
function generateColorVariants(hexColor: string) {
  // 将 HEX 转换为 HSL
  const hex = hexColor.replace('#', '');
  const r = parseInt(hex.substring(0, 2), 16) / 255;
  const g = parseInt(hex.substring(2, 4), 16) / 255;
  const b = parseInt(hex.substring(4, 6), 16) / 255;

  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  let h = 0;
  let s = 0;
  const l = (max + min) / 2;

  if (max !== min) {
    const d = max - min;
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min);

    switch (max) {
      case r:
        h = ((g - b) / d + (g < b ? 6 : 0)) / 6;
        break;
      case g:
        h = ((b - r) / d + 2) / 6;
        break;
      case b:
        h = ((r - g) / d + 4) / 6;
        break;
    }
  }

  // 生成浅色变体：保持色相和饱和度，增加亮度
  const hexFromHSL = (h: number, s: number, l: number) => {
    let r = 0;
    let g = 0;
    let b = 0;

    if (s === 0) {
      r = g = b = l;
    } else {
      const hue2rgb = (p: number, q: number, t: number) => {
        if (t < 0) t += 1;
        if (t > 1) t -= 1;
        if (t < 1/6) return p + (q - p) * 6 * t;
        if (t < 1/2) return q;
        if (t < 2/3) return p + (q - p) * (2/3 - t) * 6;
        return p;
      };

      const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
      const p = 2 * l - q;
      r = hue2rgb(p, q, h + 1/3);
      g = hue2rgb(p, q, h);
      b = hue2rgb(p, q, h - 1/3);
    }

    const toHex = (x: number) => {
      const hex = Math.round(x * 255).toString(16);
      return hex.length === 1 ? '0' + hex : hex;
    };

    return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
  };

  // hover 色：稍微降低亮度
  const hoverL = Math.max(0.1, l - 0.1);
  const primaryHover = hexFromHSL(h, s, hoverL);

  // light 色：大幅增加亮度（95%）
  const primaryLight = hexFromHSL(h, s * 0.3, 0.97);

  // lighter 色：中等亮度（90%）
  const primaryLighter = hexFromHSL(h, s * 0.5, 0.92);

  return {
    primary: hexColor,
    primaryHover,
    primaryLight,
    primaryLighter,
  };
}

/**
 * 应用主色调到 CSS 变量
 */
function applyPrimaryColor(hexColor: string) {
  const variants = generateColorVariants(hexColor);

  const root = document.documentElement;
  root.style.setProperty('--theme-primary', variants.primary);
  root.style.setProperty('--theme-primary-hover', variants.primaryHover);
  root.style.setProperty('--theme-primary-light', variants.primaryLight);
  root.style.setProperty('--theme-primary-lighter', variants.primaryLighter);

  // 同步更新侧边栏强调色
  root.style.setProperty('--sidebar-accent', variants.primary);
}

/**
 * 应用侧边栏背景色到 CSS 变量（自动计算文字颜色）
 */
function applySidebarBackgroundColor(hexColor: string) {
  const root = document.documentElement;

  // 设置侧边栏背景色
  root.style.setProperty('--sidebar-bg', hexColor);

  // 自动计算并设置文字颜色
  const textColor = getContrastTextColor(hexColor);
  const hoverTextColor = getLuminance(hexColor) < 0.5 ? '#FFFFFF' : '#0F172A';
  const hoverBackgroundColor = getHoverBackgroundColor(hexColor);

  root.style.setProperty('--sidebar-text', textColor);
  root.style.setProperty('--sidebar-text-hover', hoverTextColor);
  root.style.setProperty('--sidebar-text-active', '#FFFFFF');
  root.style.setProperty('--sidebar-bg-hover', hoverBackgroundColor);
  root.style.setProperty('--sidebar-bg-active', getHoverBackgroundColor(hoverBackgroundColor));

  // 根据背景深浅调整边框色
  const borderColor = getLuminance(hexColor) < 0.5
    ? hoverBackgroundColor
    : '#E2E8F0';
  root.style.setProperty('--sidebar-border', borderColor);
}

/**
 * 预设颜色方案（主色调）
 */
export const primaryColorPresets: ColorPreset[] = [
  // 主要色系
  { name: '蓝色', primary: '#2563eb', category: 'primary' },
  { name: '靛蓝', primary: '#4f46e5', category: 'primary' },
  { name: '紫色', primary: '#7c3aed', category: 'primary' },
  { name: '粉色', primary: '#db2777', category: 'accent' },

  // 功能色系
  { name: '红色', primary: '#dc2626', category: 'accent' },
  { name: '橙色', primary: '#ea580c', category: 'accent' },
  { name: '琥珀', primary: '#d97706', category: 'accent' },
  { name: '绿色', primary: '#16a34a', category: 'primary' },

  // 青色系
  { name: '青色', primary: '#0891b2', category: 'primary' },
  { name: '蓝绿', primary: '#0d9488', category: 'primary' },
];

/**
 * 预设侧边栏颜色方案
 */
export const sidebarColorPresets: ColorPreset[] = [
  // 深色系（配浅色文字）
  { name: '深蓝灰', primary: '#1E293B', category: 'neutral' },
  { name: '深灰', primary: '#374151', category: 'neutral' },
  { name: '墨黑', primary: '#0F172A', category: 'neutral' },
  { name: '靛蓝深', primary: '#1E1B4B', category: 'primary' },
  { name: '森林绿', primary: '#14532D', category: 'primary' },
  { name: '酒红', primary: '#450A0A', category: 'accent' },

  // 浅色系（配深色文字）
  { name: '纯白', primary: '#FFFFFF', category: 'neutral' },
  { name: '浅灰', primary: '#F8FAFC', category: 'neutral' },
  { name: '象牙白', primary: '#FFFBEB', category: 'neutral' },
  { name: '淡蓝', primary: '#EFF6FF', category: 'primary' },
  { name: '薄荷绿', primary: '#F0FDF4', category: 'primary' },
  { name: '樱花粉', primary: '#FFF1F2', category: 'accent' },
];

interface ColorSwitcherProps {
  size?: 'small' | 'middle' | 'large';
  type?: 'default' | 'picker' | 'dropdown';
  showPresetName?: boolean;
}

/**
 * 颜色选择器组件
 */
import type { FC } from 'react';

const ColorSwitcher: FC<ColorSwitcherProps> = ({
  size = 'middle',
  type = 'dropdown',
}) => {
  const [primaryColor, setPrimaryColor] = useState<string>('#4F46E5');
  const [sidebarColor, setSidebarColor] = useState<string>('#1E293B');
  const [isOpen, setIsOpen] = useState(false);

  // 初始化：从 localStorage 读取保存的颜色
  useLayoutEffect(() => {
    const savedPrimary = localStorage.getItem('theme-primary-color');
    const savedSidebar = localStorage.getItem('theme-sidebar-color');

    if (savedPrimary) {
      // 使用 setTimeout 避免同步 setState
      setTimeout(() => setPrimaryColor(savedPrimary), 0);
      applyPrimaryColor(savedPrimary);
    }
    if (savedSidebar) {
      setTimeout(() => setSidebarColor(savedSidebar), 0);
      applySidebarBackgroundColor(savedSidebar);
    }

  }, []);

  /**
   * 主色调变化处理
   */
  const handlePrimaryColorChange: ColorPickerProps['onChange'] = (color) => {
    const hexColor = typeof color === 'string' ? color : color.toHexString();
    setPrimaryColor(hexColor);
    localStorage.setItem('theme-primary-color', hexColor);
    applyPrimaryColor(hexColor);
  };

  /**
   * 侧边栏颜色变化处理
   */
  const handleSidebarColorChange: ColorPickerProps['onChange'] = (color) => {
    const hexColor = typeof color === 'string' ? color : color.toHexString();
    setSidebarColor(hexColor);
    localStorage.setItem('theme-sidebar-color', hexColor);
    applySidebarBackgroundColor(hexColor);
  };

  /**
   * 获取当前颜色的名称
   */
  const getCurrentColorName = () => {
    const preset = primaryColorPresets.find(p => p.primary.toLowerCase() === primaryColor.toLowerCase());
    if (preset) return preset.name;
    return '自定义';
  };

  /**
   * 获取侧边栏颜色名称
   */
  const getSidebarColorName = () => {
    const preset = sidebarColorPresets.find(p => p.primary.toLowerCase() === sidebarColor.toLowerCase());
    if (preset) return preset.name;
    return '自定义';
  };

  /**
   * 预设主色调选择
   */
  const handlePrimaryPresetSelect = (preset: ColorPreset) => {
    setPrimaryColor(preset.primary);
    localStorage.setItem('theme-primary-color', preset.primary);
    localStorage.setItem('theme-primary-name', preset.name);
    applyPrimaryColor(preset.primary);
  };

  /**
   * 预设侧边栏颜色选择
   */
  const handleSidebarPresetSelect = (preset: ColorPreset) => {
    setSidebarColor(preset.primary);
    localStorage.setItem('theme-sidebar-color', preset.primary);
    localStorage.setItem('theme-sidebar-name', preset.name);
    applySidebarBackgroundColor(preset.primary);
  };

  // ===== 渲染模式 =====

  /**
   * 模式1: 仅颜色选择器（主色调）
   */
  if (type === 'picker') {
    return (
      <Tooltip title={`主题色：${getCurrentColorName()}`}>
        <ColorPicker
          value={primaryColor}
          onChange={handlePrimaryColorChange}
          showText
          format="hex"
          size={size}
          style={{
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <Button
            type="text"
            size={size}
            icon={<BgColorsOutlined />}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
            }}
          >
            <div
              style={{
                width: '16px',
                height: '16px',
                borderRadius: '4px',
                background: primaryColor,
                border: '1px solid rgba(0,0,0,0.1)',
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
  const menuItems: MenuProps['items'] = [
    // 主色调选择
    {
      key: 'primary-header',
      label: (
        <div style={{ padding: '8px 0' }}>
          <Space>
            <BgColorsOutlined style={{ color: primaryColor }} />
            <Text strong>主题色</Text>
          </Space>
        </div>
      ),
      disabled: true,
    },
    {
      type: 'group',
      label: '预设主题色',
      children: primaryColorPresets.map((preset) => ({
        key: `primary-${preset.primary}`,
        label: (
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', justifyContent: 'space-between' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
              <div
                style={{
                  width: '20px',
                  height: '20px',
                  borderRadius: '4px',
                  background: preset.primary,
                  border: '1px solid rgba(0,0,0,0.1)',
                }}
              />
              <span>{preset.name}</span>
            </div>
            {primaryColor.toLowerCase() === preset.primary.toLowerCase() && (
              <CheckOutlined style={{ color: primaryColor, fontSize: '12px' }} />
            )}
          </div>
        ),
        onClick: () => handlePrimaryPresetSelect(preset),
      })),
    },
    {
      key: 'primary-custom',
      label: (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '4px 0' }}>
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
      type: 'divider',
    },
    // 侧边栏颜色选择
    {
      key: 'sidebar-header',
      label: (
        <div style={{ padding: '8px 0' }}>
          <Space>
            <MenuOutlined style={{ color: getContrastTextColor(sidebarColor) === '#F8FAFC' ? sidebarColor : '#8B5CF6' }} />
            <Text strong>侧边栏</Text>
          </Space>
        </div>
      ),
      disabled: true,
    },
    {
      type: 'group',
      label: '深色背景（配浅色文字）',
      children: sidebarColorPresets.filter(p => getLuminance(p.primary) < 0.5).map((preset) => ({
        key: `sidebar-${preset.primary}`,
        label: (
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', justifyContent: 'space-between' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
              <div
                style={{
                  width: '20px',
                  height: '20px',
                  borderRadius: '4px',
                  background: preset.primary,
                  border: preset.primary === '#FFFFFF' ? '1px solid #E2E8F0' : 'none',
                }}
              />
              <span>{preset.name}</span>
            </div>
            {sidebarColor.toLowerCase() === preset.primary.toLowerCase() && (
              <CheckOutlined style={{ color: primaryColor, fontSize: '12px' }} />
            )}
          </div>
        ),
        onClick: () => handleSidebarPresetSelect(preset),
      })),
    },
    {
      type: 'group',
      label: '浅色背景（配深色文字）',
      children: sidebarColorPresets.filter(p => getLuminance(p.primary) >= 0.5).map((preset) => ({
        key: `sidebar-${preset.primary}`,
        label: (
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', justifyContent: 'space-between' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
              <div
                style={{
                  width: '20px',
                  height: '20px',
                  borderRadius: '4px',
                  background: preset.primary,
                  border: preset.primary === '#FFFFFF' ? '1px solid #E2E8F0' : 'none',
                }}
              />
              <span>{preset.name}</span>
            </div>
            {sidebarColor.toLowerCase() === preset.primary.toLowerCase() && (
              <CheckOutlined style={{ color: primaryColor, fontSize: '12px' }} />
            )}
          </div>
        ),
        onClick: () => handleSidebarPresetSelect(preset),
      })),
    },
    {
      key: 'sidebar-custom',
      label: (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '4px 0' }}>
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
      trigger={['click']}
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
              <MenuOutlined style={{ color: getContrastTextColor(sidebarColor) === '#F8FAFC' ? sidebarColor : '#8B5CF6' }} />
              <div
                style={{
                  width: '12px',
                  height: '12px',
                  borderRadius: '3px',
                  background: primaryColor,
                  border: '1px solid rgba(0,0,0,0.15)',
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
