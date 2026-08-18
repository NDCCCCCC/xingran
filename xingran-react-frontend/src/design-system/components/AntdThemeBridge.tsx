/**
 * AntD 主题桥接组件
 * AntD Theme Bridge
 *
 * v1.22 品牌化（Phase 64 · TOKEN-03）
 * - 读取 xingranBrand 常量（TS 侧唯一色值真相源）映射 AntD ThemeConfig
 * - theme.token 全量覆盖 colorPrimary / colorInfo / colorLink / colorSuccess /
 *   colorWarning / colorError / colorTextBase / colorBgBase / colorBgContainer /
 *   colorBgElevated / colorBgLayout / colorBorder / colorBorderSecondary /
 *   borderRadius / borderRadiusLG / fontFamily
 * - theme.components 覆盖 Button / Table / Input / Select / Menu / Tabs / Tag /
 *   Card 八组件，全部从 xingranBrand 读取
 *
 * 兼容 D-01/D-03：
 * - 主色优先级链 customColors.primary → theme.antdPrimary → DEFAULT_ANTD_PRIMARY
 *   保留至 Phase 65；本 phase 仅把 DEFAULT_ANTD_PRIMARY 常量值改为
 *   xingranBrand.greenPrimary（不再回退到 AntD 默认蓝）
 * - algorithm 切换（darkAlgorithm / compactAlgorithm）保留，与主题收敛正交
 *
 * 作用：
 * - 读取 settingsStore 中的 customColors.primary / customColors.sidebar
 * - 读取 settingsStore 中的 mode (light/dark) 与 layout.density
 * - 读取 themeStore 当前应用的主题（style + mode，与 applyToDOM 同源）
 * - 将上述值映射到 AntD 的 ThemeConfig（v1.22 起全量接 xingranBrand 品牌令牌）
 *
 * 主色优先级链：
 * 1. customColors.primary（用户覆盖，最高优先级）
 * 2. 主题声明的 antdPrimary（如墨绿琥珀浅色 / 深色）
 * 3. DEFAULT_ANTD_PRIMARY (xingranBrand.greenPrimary，v1.22 替换原 AntD 默认蓝)
 *
 * 原理：
 * - 之前的 App.tsx 仅 <AntConfigProvider locale={zhCN}>，没有 theme prop
 * - AntD 组件内部通过 token 系统获取 colorPrimary；缺少 theme prop 时回退到 AntD 默认色
 * - themeStore 仅更新 CSS 变量 (--theme-primary 等)，但 AntD 组件不消费 CSS 变量
 * - 本组件把 customColors.primary 直接喂给 AntD 的 ThemeConfig.token.colorPrimary
 * - 同时挂载到 :root --ant-color-primary CSS 变量，方便自定义 CSS 引用
 * - v1.22 起：token/components 全量映射到 xingranBrand，Button/Table/Input/
 *   Select/Menu/Tabs/Tag/Card 等内置组件自动品牌化，无需逐组件 CSS override
 */

import { useEffect, useMemo, type FC, type ReactNode } from "react";
import { App, ConfigProvider, theme as antdTheme } from "antd";
import type { ThemeConfig } from "antd";
import zhCN from "antd/locale/zh_CN";
import { useSettingsStore } from "@/store/settingsStore";
import { useThemeStore } from "@/store/themeStore";
import { getTheme } from "@/design-system/themes";
import { setAppMessageInstance } from "@/utils/antdMessage";
import { xingranBrand } from "@/design-system/tokens/colors";
import { fontFamily } from "@/design-system/tokens/typography";

interface AntdThemeBridgeProps {
  children: ReactNode;
}

/**
 * 默认 AntD 主色（v1.22 品牌化）
 * 从 AntD 默认蓝改为 xingranBrand.greenPrimary（#156031），
 * 避免未声明主题时残留 indigo；Phase 65 多主题移除后会进一步清理。
 */
const DEFAULT_ANTD_PRIMARY = xingranBrand.greenPrimary;

/**
 * message context 注入器
 * 运行在 antd <App> 内部（由 AntdThemeBridge 渲染），把 context-aware 的 message
 * 实例注入 module-level ref，供 axios 拦截器、静态错误处理器等无法调用 hook 的
 * 模块作用域代码共享同一 context-aware 实例（见 @/utils/antdMessage）。
 */
const MessageContextInjector: FC<{ children: ReactNode }> = ({ children }) => {
  const { message } = App.useApp();
  useEffect(() => {
    setAppMessageInstance(message);
    return () => setAppMessageInstance(null);
  }, [message]);
  return <>{children}</>;
};

const AntdThemeBridge: FC<AntdThemeBridgeProps> = ({ children }) => {
  // 订阅 settingsStore 的主题与布局配置
  const customColors = useSettingsStore((state) => state.preferences.theme.customColors);
  const themeMode = useSettingsStore((state) => state.preferences.theme.mode);
  const density = useSettingsStore((state) => state.preferences.layout.density);

  // 订阅 themeStore 当前应用的主题（与 applyToDOM 同源：configuration.style/mode），
  // 用于读取主题声明的 antdPrimary；configuration 在 previewTheme/previewMode/
  // syncFromSettings 时都会更新，保证刷新加载与预览两条路径都拿到正确主题
  const appliedThemeStyle = useThemeStore((state) => state.configuration.style);
  const appliedThemeMode = useThemeStore((state) => state.configuration.mode);

  const antdThemeConfig = useMemo<ThemeConfig>(() => {
    // 主色优先级链：customColors.primary（用户覆盖）→ 主题 antdPrimary（可选声明）
    // → DEFAULT_ANTD_PRIMARY（v1.22 起为 xingranBrand.greenPrimary #156031）
    const themeAntdPrimary = getTheme(appliedThemeStyle, appliedThemeMode)?.antdPrimary;
    const primary =
      typeof customColors?.primary === "string" && customColors.primary
        ? customColors.primary
        : themeAntdPrimary || DEFAULT_ANTD_PRIMARY;

    // 模式：dark 模式使用 darkAlgorithm
    const algorithm = themeMode === "dark" ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm;

    // 密度：compact 模式叠加 compactAlgorithm
    const algorithms = density === "compact" ? [algorithm, antdTheme.compactAlgorithm] : algorithm;

    // v1.22 品牌化: theme.token + theme.components 全量映射 xingranBrand
    // D-03 按钮纪律: colorPrimary 永远 = xingranBrand.greenPrimary (#156031)
    // 铜金 #C09058 不做实心主按钮（QA-01 反向断言锁住）
    return {
      token: {
        colorPrimary: primary,
        colorInfo: primary,
        colorLink: primary,
        colorSuccess: xingranBrand.functional.success,
        colorWarning: xingranBrand.functional.warning,
        colorError: xingranBrand.functional.danger,
        colorTextBase: xingranBrand.cream.fg,
        colorBgBase: xingranBrand.cream.canvas,
        colorBgContainer: xingranBrand.cream.surface,
        colorBgElevated: xingranBrand.cream.surface,
        colorBgLayout: xingranBrand.cream.canvas,
        colorBorder: xingranBrand.cream.border,
        colorBorderSecondary: xingranBrand.cream.borderStrong,
        // 控件 8px 一档 / 大容器 12px / 字体栈（sans 含中文）
        borderRadius: 8,
        borderRadiusLG: 12,
        fontFamily: fontFamily.sans,
      },
      components: {
        // 主按钮 D-03: 永远绿底白字
        Button: {
          colorPrimary: xingranBrand.greenPrimary,
          colorPrimaryHover: xingranBrand.greenPrimaryHover,
          colorPrimaryActive: xingranBrand.greenPrimaryActive,
          defaultBorderColor: xingranBrand.cream.border,
          borderRadius: 8,
          controlHeight: 36,
          fontWeight: 500,
        },
        // 表格: 表头 #E9EFEB 绿灰淡彩 / 行 hover #F7F5EE 浅交互底
        Table: {
          headerBg: xingranBrand.cream.headerBg,
          headerColor: xingranBrand.cream.fg,
          borderColor: xingranBrand.cream.border,
          rowHoverBg: xingranBrand.cream.zebraBg,
          borderRadius: 8,
        },
        // 输入框: focus 环用品牌绿
        Input: {
          colorBgContainer: xingranBrand.cream.surface,
          activeBorderColor: xingranBrand.greenPrimary,
          hoverBorderColor: xingranBrand.greenPrimary,
          borderRadius: 8,
          controlHeight: 36,
        },
        Select: {
          colorBgContainer: xingranBrand.cream.surface,
          activeBorderColor: xingranBrand.greenPrimary,
          hoverBorderColor: xingranBrand.greenPrimary,
          borderRadius: 8,
          controlHeight: 36,
          optionSelectedBg: xingranBrand.cream.headerBg,
        },
        // 菜单: 浅色底用绿灰 / 深色侧栏用品牌深绿
        Menu: {
          itemBg: "transparent",
          itemSelectedBg: xingranBrand.cream.headerBg,
          itemSelectedColor: xingranBrand.greenPrimary,
          itemHoverBg: xingranBrand.cream.zebraBg,
          darkItemBg: "#14532D",
          darkItemSelectedBg: "#156031",
          darkItemSelectedColor: xingranBrand.onDark.lightYellow,
          darkItemHoverBg: "#1A6839",
        },
        // Tabs: 选中色 = 品牌绿
        Tabs: {
          itemSelectedColor: xingranBrand.greenPrimary,
          itemActiveColor: xingranBrand.greenPrimary,
          itemHoverColor: xingranBrand.greenPrimaryHover,
          inkBarColor: xingranBrand.greenPrimary,
        },
        // Tag: 默认底用绿灰淡彩
        Tag: {
          defaultBg: xingranBrand.cream.headerBg,
          defaultColor: xingranBrand.cream.fg,
          borderRadiusSM: 4,
        },
        // Card: 白卡浮在奶油底上, 12px 圆角
        Card: {
          colorBgContainer: xingranBrand.cream.surface,
          headerBg: "transparent",
          borderRadiusLG: 12,
        },
      },
      algorithm: algorithms,
      // 主题变更时改变 hashed 标识，强制 AntD 重新生成 CSS（防止主题切换残留）
      // 注: 双圈 Loading 问题由 src/index.css 的 .ant-btn-loading::before 与
      // antd v6 内置 Spin 重复渲染导致, 与 hashed 配置无关, 修复见 index.css.
      hashed: true,
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- appliedThemeStyle/Mode & customColors.sidebar are stable theme inputs
  }, [
    customColors?.primary,
    customColors?.sidebar,
    themeMode,
    density,
    appliedThemeStyle,
    appliedThemeMode,
  ]);

  return (
    <ConfigProvider locale={zhCN} theme={antdThemeConfig}>
      {/*
				antd v6 默认启用 cssVar(CSS 变量模式), <App> 需要一个真实 DOM 节点挂载 CSS
				变量作用域。component={false} 不渲染节点, 会触发 warning:
				"[antd: App] When using cssVar, ensure `component` is assigned a valid React component string."
				故移除 component={false}, 使用 antd <App> 默认的 div 包裹。
			*/}
      <App>
        <MessageContextInjector>{children}</MessageContextInjector>
      </App>
    </ConfigProvider>
  );
};

export default AntdThemeBridge;
