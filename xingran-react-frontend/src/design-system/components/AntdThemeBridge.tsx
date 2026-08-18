/**
 * AntD 主题桥接组件
 * AntD Theme Bridge
 *
 * v1.22 品牌化（Phase 64 · TOKEN-03 → Phase 65 · THEME-01 收敛）
 * - 读取 xingranBrand 常量（TS 侧唯一色值真相源）映射 AntD ThemeConfig
 * - v1.22 Phase 65：单一品牌主题，主色优先级链已移除 ——
 *   colorPrimary / colorInfo / colorLink 直接使用 xingranBrand.greenPrimary，
 *   不再订阅用户自定义颜色与主题 store（多主题能力随 D-01 一并删除）
 * - theme.token 全量覆盖 colorPrimary / colorInfo / colorLink / colorSuccess /
 *   colorWarning / colorError / colorTextBase / colorBgBase / colorBgContainer /
 *   colorBgElevated / colorBgLayout / colorBorder / colorBorderSecondary /
 *   borderRadius / borderRadiusLG / fontFamily
 * - theme.components 覆盖 Button / Table / Input / Select / Menu / Tabs / Tag /
 *   Card 八组件，全部从 xingranBrand 读取
 * - algorithm 切换（darkAlgorithm / compactAlgorithm）保留，与明暗/密度正交
 *
 * 作用：
 * - 读取 settingsStore 中的 mode (light/dark) 与 layout.density
 * - 将上述值映射到 AntD 的 ThemeConfig（v1.22 起全量接 xingranBrand 品牌令牌）
 *
 * 原理：
 * - AntD 组件内部通过 token 系统获取 colorPrimary；缺少 theme prop 时回退到 AntD 默认色
 * - index.css 仅更新 CSS 变量 (--theme-primary 等)，但 AntD 组件不消费 CSS 变量
 * - 本组件把 xingranBrand.greenPrimary 直接喂给 AntD 的 ThemeConfig.token.colorPrimary
 * - v1.22 起：token/components 全量映射到 xingranBrand，Button/Table/Input/
 *   Select/Menu/Tabs/Tag/Card 等内置组件自动品牌化，无需逐组件 CSS override
 */

import { useEffect, useMemo, type FC, type ReactNode } from "react";
import { App, ConfigProvider, theme as antdTheme } from "antd";
import type { ThemeConfig } from "antd";
import zhCN from "antd/locale/zh_CN";
import { useSettingsStore } from "@/store/settingsStore";
import { setAppMessageInstance } from "@/utils/antdMessage";
import { xingranBrand } from "@/design-system/tokens/colors";
import { fontFamily } from "@/design-system/tokens/typography";

interface AntdThemeBridgeProps {
  children: ReactNode;
}

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
  // 订阅 settingsStore 的明暗模式与布局密度（Phase 65 后仅剩两个主题输入）
  const themeMode = useSettingsStore((state) => state.preferences.theme.mode);
  const density = useSettingsStore((state) => state.preferences.layout.density);

  const antdThemeConfig = useMemo<ThemeConfig>(() => {
    // 模式：dark 模式使用 darkAlgorithm
    const algorithm = themeMode === "dark" ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm;

    // 密度：compact 模式叠加 compactAlgorithm
    const algorithms = density === "compact" ? [algorithm, antdTheme.compactAlgorithm] : algorithm;

    // v1.22 品牌化: theme.token + theme.components 全量映射 xingranBrand
    // D-03 按钮纪律: colorPrimary 永远 = xingranBrand.greenPrimary (#156031)
    // 铜金 #C09058 不做实心主按钮（QA-01 反向断言锁住）
    return {
      token: {
        colorPrimary: xingranBrand.greenPrimary,
        colorInfo: xingranBrand.greenPrimary,
        colorLink: xingranBrand.greenPrimary,
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
        // 主按钮 D-03: 永远绿底白字 (Phase 66 · G4/COMP-03 白字 7.64:1)
        Button: {
          colorPrimary: xingranBrand.greenPrimary,
          primaryColor: xingranBrand.onDark.white,
          colorPrimaryHover: xingranBrand.greenPrimaryHover,
          colorPrimaryActive: xingranBrand.greenPrimaryActive,
          defaultBorderColor: xingranBrand.cream.border,
          borderRadius: 8,
          controlHeight: 36,
          fontWeight: 500,
        },
        // 表格: 表头 #E9EFEB 绿灰淡彩 / 行 hover #F7F5EE 浅交互底
        // (Phase 66 · G1/COMP-02 排序/筛选/选中态令牌补齐)
        Table: {
          headerBg: xingranBrand.cream.headerBg,
          headerColor: xingranBrand.cream.fg,
          borderColor: xingranBrand.cream.border,
          rowHoverBg: xingranBrand.cream.zebraBg,
          headerSortActiveBg: xingranBrand.cream.zebraBg,
          headerSortHoverBg: xingranBrand.cream.zebraBg,
          headerFilterHoverBg: xingranBrand.cream.zebraBg,
          fixedHeaderSortActiveBg: xingranBrand.cream.zebraBg,
          rowSelectedBg: xingranBrand.cream.headerBg,
          rowSelectedHoverBg: xingranBrand.cream.zebraBg,
          borderRadius: 8,
        },
        // 输入框: focus 环用品牌绿 (Phase 66 · G2/COMP-04 焦点环 2px)
        Input: {
          colorBgContainer: xingranBrand.cream.surface,
          activeBorderColor: xingranBrand.greenPrimary,
          hoverBorderColor: xingranBrand.greenPrimary,
          activeShadow: "0 0 0 2px rgba(21, 96, 49, 0.15)",
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
        // (Phase 66 · QA-02 卫生 — darkItem* 改为 xingranBrand 引用)
        Menu: {
          itemBg: "transparent",
          itemSelectedBg: xingranBrand.cream.headerBg,
          itemSelectedColor: xingranBrand.greenPrimary,
          itemHoverBg: xingranBrand.cream.zebraBg,
          darkItemBg: xingranBrand.green[900],
          darkItemSelectedBg: xingranBrand.greenPrimary,
          darkItemSelectedColor: xingranBrand.onDark.lightYellow,
          darkItemHoverBg: xingranBrand.greenPrimaryLight,
        },
        // Tabs: 选中色 = 品牌绿
        Tabs: {
          itemSelectedColor: xingranBrand.greenPrimary,
          itemActiveColor: xingranBrand.greenPrimary,
          itemHoverColor: xingranBrand.greenPrimaryHover,
          inkBarColor: xingranBrand.greenPrimary,
        },
        // Tag: 默认底用淡黄 (Phase 66 · G3/COMP-04 SM2/SM3/SM4 品牌锚点配方)
        Tag: {
          defaultBg: xingranBrand.onDark.paleYellow,
          defaultColor: xingranBrand.copper[500],
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
  }, [themeMode, density]);

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
