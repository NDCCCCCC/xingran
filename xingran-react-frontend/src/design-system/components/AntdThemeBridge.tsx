/**
 * AntD 主题桥接组件
 * AntD Theme Bridge
 *
 * 作用：
 * - 读取 settingsStore 中的 customColors.primary / customColors.sidebar
 * - 读取 settingsStore 中的 mode (light/dark) 与 layout.density
 * - 读取 themeStore 当前应用的主题（style + mode，与 applyToDOM 同源）
 * - 将上述值映射到 AntD 的 ThemeConfig
 *   - token.colorPrimary: 主色
 *   - token.colorInfo: 主色（保持一致）
 *   - token.colorLink: 主色（保持一致）
 *   - algorithm: 暗色模式切换为 darkAlgorithm；密度紧凑切换为 compactAlgorithm
 *
 * 主色优先级链：
 * 1. customColors.primary（用户覆盖，最高优先级）
 * 2. 主题声明的 antdPrimary（如墨绿琥珀浅色 #166534 / 深色 #d4a574）
 * 3. DEFAULT_ANTD_PRIMARY (#1677ff，未声明主题保持 AntD 默认蓝，行为不变)
 *
 * 解决"工位管理页面表格/卡片/平面图三选一按钮硬编码蓝色"等
 * 所有 AntD 组件不响应用户主题色的问题。
 *
 * 原理：
 * - 之前的 App.tsx 仅 <AntConfigProvider locale={zhCN}>，没有 theme prop
 * - AntD 组件内部通过 token 系统获取 colorPrimary；缺少 theme prop 时使用默认 #1677ff
 * - themeStore 仅更新 CSS 变量 (--theme-primary 等)，但 AntD 组件不消费 CSS 变量
 * - 本组件把 customColors.primary 直接喂给 AntD 的 ThemeConfig.token.colorPrimary
 * - 同时挂载到 :root --ant-color-primary CSS 变量，方便自定义 CSS 引用
 */

import { useEffect, useMemo, type FC, type ReactNode } from "react";
import { App, ConfigProvider, theme as antdTheme } from "antd";
import type { ThemeConfig } from "antd";
import zhCN from "antd/locale/zh_CN";
import { useSettingsStore } from "@/store/settingsStore";
import { useThemeStore } from "@/store/themeStore";
import { getTheme } from "@/design-system/themes";
import { setAppMessageInstance } from "@/utils/antdMessage";

interface AntdThemeBridgeProps {
  children: ReactNode;
}

/**
 * 默认 AntD 主色（与 AntD 内部 defaultSeedToken.colorPrimary 保持一致）
 * 当用户未配置 customColors.primary 时使用此值
 */
const DEFAULT_ANTD_PRIMARY = "#1677ff";

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
    // → DEFAULT_ANTD_PRIMARY（未声明主题保持 AntD 默认蓝）
    const themeAntdPrimary = getTheme(appliedThemeStyle, appliedThemeMode)?.antdPrimary;
    const primary =
      typeof customColors?.primary === "string" && customColors.primary
        ? customColors.primary
        : themeAntdPrimary || DEFAULT_ANTD_PRIMARY;

    // 模式：dark 模式使用 darkAlgorithm
    const algorithm = themeMode === "dark" ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm;

    // 密度：compact 模式叠加 compactAlgorithm
    const algorithms = density === "compact" ? [algorithm, antdTheme.compactAlgorithm] : algorithm;

    // 把主色同时写入 colorInfo 和 colorLink，保证 Info / Link 类组件也跟随主题
    return {
      token: {
        colorPrimary: primary,
        colorInfo: primary,
        colorLink: primary,
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
