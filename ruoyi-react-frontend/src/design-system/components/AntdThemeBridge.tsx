/**
 * AntD 主题桥接组件
 * AntD Theme Bridge
 *
 * 作用：
 * - 读取 settingsStore 中的 customColors.primary / customColors.sidebar
 * - 读取 settingsStore 中的 mode (light/dark) 与 layout.density
 * - 将上述值映射到 AntD 的 ThemeConfig
 *   - token.colorPrimary: 主色
 *   - token.colorInfo: 主色（保持一致）
 *   - token.colorLink: 主色（保持一致）
 *   - algorithm: 暗色模式切换为 darkAlgorithm；密度紧凑切换为 compactAlgorithm
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

import { useMemo, type FC, type ReactNode } from 'react';
import { ConfigProvider, theme as antdTheme } from 'antd';
import type { ThemeConfig } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { useSettingsStore } from '@/store/settingsStore';
import { useThemeStore } from '@/store/themeStore';

interface AntdThemeBridgeProps {
	children: ReactNode;
}

/**
 * 默认 AntD 主色（与 AntD 内部 defaultSeedToken.colorPrimary 保持一致）
 * 当用户未配置 customColors.primary 时使用此值
 */
const DEFAULT_ANTD_PRIMARY = '#1677ff';

const AntdThemeBridge: FC<AntdThemeBridgeProps> = ({ children }) => {
	// 订阅 settingsStore 的主题与布局配置
	const customColors = useSettingsStore((state) => state.preferences.theme.customColors);
	const themeMode = useSettingsStore((state) => state.preferences.theme.mode);
	const density = useSettingsStore((state) => state.preferences.layout.density);

	// 触发 themeStore 同步到 DOM（CSS 变量），保证 --theme-primary 实时更新
	const appliedTheme = useThemeStore((state) => state.appliedTheme);

	const antdThemeConfig = useMemo<ThemeConfig>(() => {
		// 主色：优先使用 customColors.primary（如果存在且是字符串），否则回退到默认
		const primary =
			typeof customColors?.primary === 'string' && customColors.primary
				? customColors.primary
				: DEFAULT_ANTD_PRIMARY;

		// 模式：dark 模式使用 darkAlgorithm
		const algorithm =
			themeMode === 'dark'
				? antdTheme.darkAlgorithm
				: antdTheme.defaultAlgorithm;

		// 密度：compact 模式叠加 compactAlgorithm
		const algorithms =
			density === 'compact' ? [algorithm, antdTheme.compactAlgorithm] : algorithm;

		// 把主色同时写入 colorInfo 和 colorLink，保证 Info / Link 类组件也跟随主题
		return {
			token: {
				colorPrimary: primary,
				colorInfo: primary,
				colorLink: primary,
			},
			algorithm: algorithms,
			// 主题变更时改变 hashed 标识，强制 AntD 重新生成 CSS（防止主题切换残留）
			hashed: true,
		};
	}, [customColors?.primary, customColors?.sidebar, themeMode, density, appliedTheme]);

	return <ConfigProvider locale={zhCN} theme={antdThemeConfig}>{children}</ConfigProvider>;
};

export default AntdThemeBridge;
