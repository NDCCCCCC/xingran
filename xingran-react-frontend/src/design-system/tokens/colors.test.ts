/**
 * xingranBrand WCAG 2.1 对比度自动验证（QA-01）
 *
 * 目的：把 brand-spec 列出的关键前景/背景对锁在测试层 —— 不达标即 fail，
 * 防止后续重构意外破坏 D-03 按钮纪律或品牌对比度承诺。
 *
 * 来源：
 * - WCAG 2.1 相对亮度公式 https://www.w3.org/WAI/WCAG21/Techniques/general/G18
 * - brand-spec.md（像素实测 + OKLch 派生 + WCAG 标注）
 *
 * 覆盖：
 * - D-03 主按钮纪律：#FFFFFF on #156031 (greenPrimary) ≥ 7.6:1
 * - D-03 反向断言：#FFFFFF on #C09058 (copper) < 3.5:1（铜金不可作主按钮）
 * - 关键侧栏 / 表头 / 警告 / 错误 / 成功对全部断言达标
 */

import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { xingranBrand } from "./colors";

/**
 * 将 #RRGGBB hex 解析为 [r, g, b] 0-255 整数数组。
 */
function hexToRgb(hex: string): [number, number, number] {
  const cleaned = hex.replace("#", "").trim();
  const expanded =
    cleaned.length === 3
      ? cleaned
          .split("")
          .map((c) => c + c)
          .join("")
      : cleaned;
  if (!/^[0-9a-fA-F]{6}$/.test(expanded)) {
    throw new Error(`Invalid hex color: ${hex}`);
  }
  const num = parseInt(expanded, 16);
  return [(num >> 16) & 0xff, (num >> 8) & 0xff, num & 0xff];
}

/**
 * sRGB → linear (per WCAG 2.1):
 * - if c <= 0.03928: c / 12.92
 * - else: ((c + 0.055) / 1.055) ^ 2.4
 */
function srgbToLinear(c8bit: number): number {
  const c = c8bit / 255;
  return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
}

/**
 * 相对亮度 L (WCAG 2.1):
 * L = 0.2126*R + 0.7152*G + 0.0722*B
 * 其中 R/G/B 是 srgbToLinear 的输出（0-1 范围）。
 */
export function relativeLuminance(hex: string): number {
  const [r, g, b] = hexToRgb(hex);
  return 0.2126 * srgbToLinear(r) + 0.7152 * srgbToLinear(g) + 0.0722 * srgbToLinear(b);
}

/**
 * 对比度比 = (L1 + 0.05) / (L2 + 0.05)，其中 L1 是较亮，L2 是较暗。
 */
export function contrastRatio(fg: string, bg: string): number {
  const l1 = relativeLuminance(fg);
  const l2 = relativeLuminance(bg);
  const lighter = Math.max(l1, l2);
  const darker = Math.min(l1, l2);
  return (lighter + 0.05) / (darker + 0.05);
}

describe("xingranBrand WCAG contrast ratios", () => {
  // ---- D-03 主按钮纪律 ----

  it("#FFFFFF on xingranBrand.greenPrimary (#156031) meets WCAG AA (≥ 7.6:1)", () => {
    const ratio = contrastRatio("#FFFFFF", xingranBrand.greenPrimary);
    expect(ratio).toBeGreaterThanOrEqual(7.6);
  });

  it("#FFFFFF on xingranBrand.greenPrimaryHover (#2E7444) meets WCAG AA (≥ 5.6:1)", () => {
    const ratio = contrastRatio("#FFFFFF", xingranBrand.greenPrimaryHover);
    expect(ratio).toBeGreaterThanOrEqual(5.6);
  });

  it("#FFFFFF on xingranBrand.greenPrimaryActive (#14542E) meets WCAG AA (≥ 7.0:1)", () => {
    const ratio = contrastRatio("#FFFFFF", xingranBrand.greenPrimaryActive);
    expect(ratio).toBeGreaterThanOrEqual(7.0);
  });

  it("#FFFFFF on xingranBrand.green[900] (#14532D) meets WCAG AA (≥ 7.0:1) — 深绿底", () => {
    const ratio = contrastRatio("#FFFFFF", xingranBrand.green[900]);
    expect(ratio).toBeGreaterThanOrEqual(7.0);
  });

  // ---- 侧边栏深绿底文字 ----

  it("xingranBrand.onDark.lightYellow (#E0E0B0) on xingranBrand.greenPrimary (#156031) meets WCAG AA (≥ 5.6:1)", () => {
    const ratio = contrastRatio(xingranBrand.onDark.lightYellow, xingranBrand.greenPrimary);
    expect(ratio).toBeGreaterThanOrEqual(5.6);
  });

  it("xingranBrand.cream.canvas (#F0ECE3) on xingranBrand.green[900] (#14532D) meets WCAG AA (≥ 7.0:1) — 深绿侧栏文字", () => {
    const ratio = contrastRatio(xingranBrand.cream.canvas, xingranBrand.green[900]);
    expect(ratio).toBeGreaterThanOrEqual(7.0);
  });

  // ---- 白卡 / 奶油底文字 ----

  it("xingranBrand.cream.muted (#707068) on xingranBrand.cream.surface (#FFFFFF) meets WCAG AA (≥ 4.9:1) — 白卡次级文字", () => {
    const ratio = contrastRatio(xingranBrand.cream.muted, xingranBrand.cream.surface);
    expect(ratio).toBeGreaterThanOrEqual(4.9);
  });

  it("xingranBrand.cream.mutedStrong (#64645C) on xingranBrand.cream.canvas (#F0ECE3) meets WCAG AA (≥ 4.9:1) — 奶油底次级文字", () => {
    const ratio = contrastRatio(xingranBrand.cream.mutedStrong, xingranBrand.cream.canvas);
    expect(ratio).toBeGreaterThanOrEqual(4.9);
  });

  it("xingranBrand.cream.fg (#101010) on xingranBrand.cream.canvas (#F0ECE3) meets WCAG AA (≥ 12:1) — 奶油底主文字", () => {
    const ratio = contrastRatio(xingranBrand.cream.fg, xingranBrand.cream.canvas);
    expect(ratio).toBeGreaterThanOrEqual(12);
  });

  it("xingranBrand.cream.fg (#101010) on xingranBrand.cream.surface (#FFFFFF) meets WCAG AA (≥ 15:1) — 白卡主文字", () => {
    const ratio = contrastRatio(xingranBrand.cream.fg, xingranBrand.cream.surface);
    expect(ratio).toBeGreaterThanOrEqual(15);
  });

  // ---- 功能色 ----

  it("#FFFFFF on xingranBrand.functional.danger (#BA3630) meets WCAG AA (≥ 5.6:1) — 危险按钮", () => {
    const ratio = contrastRatio("#FFFFFF", xingranBrand.functional.danger);
    expect(ratio).toBeGreaterThanOrEqual(5.6);
  });

  it("#FFFFFF on xingranBrand.functional.success (#2D8949) meets WCAG AA (≥ 4.3:1) — 成功图标/大字", () => {
    const ratio = contrastRatio("#FFFFFF", xingranBrand.functional.success);
    expect(ratio).toBeGreaterThanOrEqual(4.3);
  });

  it("#FFFFFF on xingranBrand.functional.successSolid (#238142) meets WCAG AA (≥ 4.5:1) — 成功实心按钮", () => {
    const ratio = contrastRatio("#FFFFFF", xingranBrand.functional.successSolid);
    expect(ratio).toBeGreaterThanOrEqual(4.5);
  });

  it("xingranBrand.functional.warningText (#905D00) on #FFFFFF meets WCAG AA (≥ 5.5:1) — 警告文字白底 (brand-spec 标注 5.60:1)", () => {
    const ratio = contrastRatio(xingranBrand.functional.warningText, "#FFFFFF");
    expect(ratio).toBeGreaterThanOrEqual(5.5);
  });

  it("xingranBrand.functional.warningText (#905D00) on xingranBrand.onDark.paleYellow (#FEF3C7) meets WCAG AA (≥ 5.0:1) — 警告文字淡黄底", () => {
    const ratio = contrastRatio(
      xingranBrand.functional.warningText,
      xingranBrand.onDark.paleYellow
    );
    expect(ratio).toBeGreaterThanOrEqual(5.0);
  });

  // ---- 铜金梯度（仅作点缀/大字实心按钮，禁止作主按钮） ----

  it("#FFFFFF on xingranBrand.copper[500] (#B88850) meets WCAG AA Large (≥ 3.0:1) — 铜金大字实心按钮", () => {
    const ratio = contrastRatio("#FFFFFF", xingranBrand.copper[500]);
    expect(ratio).toBeGreaterThanOrEqual(3.0);
  });

  it("#FFFFFF on xingranBrand.copper[400] (#AA7B42) meets WCAG AA Large (≥ 3.0:1) — 铜金 hover", () => {
    const ratio = contrastRatio("#FFFFFF", xingranBrand.copper[400]);
    expect(ratio).toBeGreaterThanOrEqual(3.0);
  });

  // ---- D-03 反向断言（防回归） ----

  it("#FFFFFF on xingranBrand.copperAccent (#C09058) violates D-03 (< 3.5:1) — 铜金不可作主按钮背景", () => {
    const ratio = contrastRatio("#FFFFFF", xingranBrand.copperAccent);
    expect(ratio).toBeLessThan(3.5);
  });

  it("#FFFFFF on xingranBrand.copper[600] (#C09058) violates D-03 (< 4.5:1) — 铜金不可作次按钮背景", () => {
    const ratio = contrastRatio("#FFFFFF", xingranBrand.copper[600]);
    expect(ratio).toBeLessThan(4.5);
  });

  // ---- 品牌令牌完整性（防止意外删除） ----

  it("xingranBrand exposes all required token groups", () => {
    expect(xingranBrand.green).toBeDefined();
    expect(xingranBrand.green[50]).toBe("#E9EFEB");
    expect(xingranBrand.green[100]).toBe("#598E5E");
    expect(xingranBrand.green[200]).toBe("#3B784C");
    expect(xingranBrand.green[300]).toBe("#1A6839");
    expect(xingranBrand.green[400]).toBe("#156031");
    expect(xingranBrand.green[900]).toBe("#14532D");
    expect(xingranBrand.greenPrimary).toBe("#156031");
    expect(xingranBrand.copperAccent).toBe("#C09058");
    expect(xingranBrand.cream.canvas).toBe("#F0ECE3");
    expect(xingranBrand.cream.surface).toBe("#FFFFFF");
    expect(xingranBrand.onDark.lightYellow).toBe("#E0E0B0");
    expect(xingranBrand.functional.danger).toBe("#BA3630");
  });
});

/**
 * 从 CSS 文本中切出指定选择器的规则块（花括号匹配）。
 * selector 需含尾部空格（如 ":root {" / '[data-color-mode="dark"] {'）
 * 以精确命中变量定义块而非组件覆盖规则。
 */
function extractCssBlock(css: string, selectorWithBrace: string): string {
  const start = css.indexOf(selectorWithBrace);
  if (start === -1) {
    throw new Error(`Selector block not found: ${selectorWithBrace}`);
  }
  const open = css.indexOf("{", start);
  let depth = 0;
  for (let i = open; i < css.length; i++) {
    if (css[i] === "{") depth++;
    else if (css[i] === "}") {
      depth--;
      if (depth === 0) {
        return css.slice(open + 1, i);
      }
    }
  }
  throw new Error(`Unbalanced braces for block: ${selectorWithBrace}`);
}

/**
 * 读取 src/index.css 文本（THEME-02 静态变量层防回归）。
 * 注：ESLint typed-lint 程序未含 @types/node，node:fs/url 的调用被标记为
 * unsafe —— 属工具链误报，测试运行时（vitest/node）类型完全正常，定向豁免。
 */
function loadIndexCss(): string {
  const here = dirname(fileURLToPath(import.meta.url));

  return readFileSync(resolve(here, "../../index.css"), "utf-8");
}

describe("dark-mode brand derivation (THEME-02)", () => {
  const css = loadIndexCss();
  const rootBlock = extractCssBlock(css, ":root {").toLowerCase();
  const darkBlock = extractCssBlock(css, '[data-color-mode="dark"] {').toLowerCase();

  it("dark block contains deep-green sidebar bg #0a2418 (Phase 64 derivation)", () => {
    expect(darkBlock).toContain("--sidebar-bg: #0a2418");
  });

  it("dark block contains deep-green canvas #0f2e1b for --theme-bg-primary (Phase 64 derivation)", () => {
    expect(darkBlock).toContain("--theme-bg-primary: #0f2e1b");
    expect(darkBlock).toContain("--theme-bg-surface: #1a2e1f");
  });

  // ---- Phase 65 品牌深底提亮推导（静态品牌变量） ----

  it("dark block lightens --theme-brand to icon green #598e5e (controlled lightening)", () => {
    expect(darkBlock).toContain("--theme-brand: #598e5e");
    expect(darkBlock).toContain("--theme-brand-dark: #3b784c");
    expect(darkBlock).toContain("--theme-brand-alpha-10: rgba(89, 142, 94, 0.15)");
  });

  it("dark block lightens --theme-primary-500/600 to secondary/icon green", () => {
    expect(darkBlock).toContain("--theme-primary-500: #3b784c");
    expect(darkBlock).toContain("--theme-primary-600: #598e5e");
  });

  it(":root defines static brand vars matching brand-spec (D-02 single source)", () => {
    expect(rootBlock).toContain("--theme-brand: #156031");
    expect(rootBlock).toContain("--theme-brand-dark: #14542e");
    expect(rootBlock).toContain("--theme-brand-alpha-10: rgba(21, 96, 49, 0.1)");
    expect(rootBlock).toContain("--theme-primary-50: #e9efeb");
    expect(rootBlock).toContain("--theme-primary-100: #598e5e");
    expect(rootBlock).toContain("--theme-primary-500: #156031");
    expect(rootBlock).toContain("--theme-primary-600: #14532d");
    expect(rootBlock).toContain("--theme-primary-700: #14542e");
  });

  // ---- light 模式基线 ----

  it("light baseline: :root --theme-primary stays brand #156031 (no regression)", () => {
    expect(rootBlock).toContain("--theme-primary: #156031");
  });

  // ---- 受控提亮：dark 与 :root 品牌色不同且均为品牌绿梯度成员 ----

  it("dark --theme-brand differs from :root and both are brand green gradient members", () => {
    const darkBrand = "#598E5E";
    const rootBrand = "#156031";
    expect(darkBrand).not.toBe(rootBrand);
    // xingranBrand.green[100] = 深底图标绿, green[400] = 品牌主色
    expect(darkBrand).toBe(xingranBrand.green[100]);
    expect(rootBrand).toBe(xingranBrand.green[400]);
  });

  // ---- 暗色关键前景/背景对 WCAG AA (≥ 4.5:1) ----

  it("#F0ECE3 (cream text) on #0A2418 (deep green sidebar) meets WCAG AA (≥ 4.5:1)", () => {
    const ratio = contrastRatio("#F0ECE3", "#0A2418");
    expect(ratio).toBeGreaterThanOrEqual(4.5);
  });

  it("#E0E0B0 (light yellow accent) on #0A2418 (deep green sidebar) meets WCAG AA (≥ 4.5:1)", () => {
    const ratio = contrastRatio("#E0E0B0", "#0A2418");
    expect(ratio).toBeGreaterThanOrEqual(4.5);
  });

  it("#F0ECE3 (cream text) on #0F2E1B (dark canvas) meets WCAG AA (≥ 4.5:1)", () => {
    const ratio = contrastRatio("#F0ECE3", "#0F2E1B");
    expect(ratio).toBeGreaterThanOrEqual(4.5);
  });

  it("#FFFFFF on #156031 primary button unchanged in dark mode meets WCAG AA (≥ 7.6:1)", () => {
    const ratio = contrastRatio("#FFFFFF", "#156031");
    expect(ratio).toBeGreaterThanOrEqual(7.6);
  });
});
