/**
 * WCAG 对比度计算工具
 * 用于验证主题配色是否符合无障碍标准
 */

/**
 * 将 HEX 颜色转换为 RGB
 */
function hexToRGB(hex: string): { r: number; g: number; b: number } {
	const hexColor = hex.replace("#", "");
	const r = parseInt(hexColor.substring(0, 2), 16);
	const g = parseInt(hexColor.substring(2, 4), 16);
	const b = parseInt(hexColor.substring(4, 6), 16);
	return { r, g, b };
}

/**
 * 计算颜色的相对亮度
 * 根据 WCAG 2.0 规范
 * @param hex HEX 格式的颜色值
 * @returns 0-1 之间的亮度值
 */
export function calculateLuminance(hex: string): number {
	const { r, g, b } = hexToRGB(hex);

	// 将 RGB 值转换为线性 RGB
	const toLinear = (c: number) => {
		const normalized = c / 255;
		return normalized <= 0.03928 ? normalized / 12.92 : Math.pow((normalized + 0.055) / 1.055, 2.4);
	};

	const R = toLinear(r);
	const G = toLinear(g);
	const B = toLinear(b);

	// 计算相对亮度
	return 0.2126 * R + 0.7152 * G + 0.0722 * B;
}

/**
 * 计算两个颜色之间的对比度
 * @param foreground 前景色（文字色）
 * @param background 背景色
 * @returns 对比值（1-21 之间）
 */
export function calculateContrast(foreground: string, background: string): number {
	const l1 = calculateLuminance(foreground);
	const l2 = calculateLuminance(background);

	// 确保较亮的颜色作为分子
	const lighter = Math.max(l1, l2);
	const darker = Math.min(l1, l2);

	return (lighter + 0.05) / (darker + 0.05);
}

/**
 * 检查是否符合 WCAG AA 标准
 * @param foreground 前景色
 * @param background 背景色
 * @param fontSize 字体大小（px），用于判断是正常文本还是大文本
 * @param fontWeight 字体粗细，粗体文本可以使用较低的对比度
 * @returns 是否符合 WCAG AA 标准
 */
export function isWCAGAACompliant(
	foreground: string,
	background: string,
	fontSize: number = 14,
	fontWeight: number = 400
): boolean {
	const contrast = calculateContrast(foreground, background);

	// 大文本：18pt 以上或 14pt 以上粗体，对比度要求 3:1
	const isLargeText = fontSize >= 18 || (fontSize >= 14 && fontWeight >= 700);
	const requiredContrast = isLargeText ? 3.0 : 4.5;

	return contrast >= requiredContrast;
}

/**
 * 检查是否符合 WCAG AAA 标准（更严格）
 * @param foreground 前景色
 * @param background 背景色
 * @param fontSize 字体大小（px）
 * @param fontWeight 字体粗细
 * @returns 是否符合 WCAG AAA 标准
 */
export function isWCAGAAACompliant(
	foreground: string,
	background: string,
	fontSize: number = 14,
	fontWeight: number = 400
): boolean {
	const contrast = calculateContrast(foreground, background);

	// AAA 标准：正常文本 7:1，大文本 4.5:1
	const isLargeText = fontSize >= 18 || (fontSize >= 14 && fontWeight >= 700);
	const requiredContrast = isLargeText ? 4.5 : 7.0;

	return contrast >= requiredContrast;
}

/**
 * 获取对比度等级描述
 * @param contrast 对比值
 * @returns 等级描述
 */
export function getContrastRating(contrast: number): string {
	if (contrast >= 7.0) return "AAA (优秀)";
	if (contrast >= 4.5) return "AA (良好)";
	if (contrast >= 3.0) return "AA 大文本 (及格)";
	return "不符合标准";
}

/**
 * 验证主题配色是否符合 WCAG 标准
 * @param colors 颜色配置
 * @returns 验证结果
 */
export function validateThemeColors(colors: {
	text: { primary: string; secondary: string; tertiary: string };
	background: { primary: string; secondary: string; tertiary: string };
}): {
	pass: boolean;
	results: Array<{
		foreground: string;
		background: string;
		contrast: number;
		rating: string;
		wcagAA: boolean;
	}>;
} {
	const results = [];

	// 验证主要文字在各个背景上的对比度
	const textColors = ["text.primary", "text.secondary", "text.tertiary"] as const;
	const bgColors = ["background.primary", "background.secondary", "background.tertiary"] as const;

	for (const textKey of textColors) {
		for (const bgKey of bgColors) {
			const textCategory = textKey.split(".")[0] as "text" | "background";
			const textColorKey = textKey.split(".")[1] as "primary" | "secondary" | "tertiary";
			const bgCategory = bgKey.split(".")[0] as "text" | "background";
			const bgColorKey = bgKey.split(".")[1] as "primary" | "secondary" | "tertiary";
			const fg = colors[textCategory][textColorKey];
			const bg = colors[bgCategory][bgColorKey];

			const contrast = calculateContrast(fg, bg);
			results.push({
				foreground: textKey,
				background: bgKey,
				contrast,
				rating: getContrastRating(contrast),
				wcagAA: isWCAGAACompliant(fg, bg),
			});
		}
	}

	// 检查是否所有组合都符合 WCAG AA
	const pass = results.every((r) => r.wcagAA);

	return { pass, results };
}
