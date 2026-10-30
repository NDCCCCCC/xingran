/**
 * Phase 88 · VISFIX-06 C15 —— AntdThemeBridge useMemo 依赖覆盖测试
 *
 * 意图：组件的 antdThemeConfig useMemo 读取的外部状态集合 { themeMode, density }
 * 必须全部出现在 deps 数组中。Phase 89 扩色板引入新输入源（色板 / 密度档位等）
 * 时，若 useMemo deps 未同步扩充，本测试的行为断言（token 引用必须变化）会失败
 * —— 即 P-C2 的 detection 安全网：逼着扩 deps，防止「改了 store 却不重算主题」。
 *
 * 双层断言：
 *   1. 行为断言（优先）：themeMode / density 变化后，ConfigProvider 下发的 antd
 *      theme token 必须是重新计算出的新对象引用（useMemo 重算证据）。
 *   2. 静态兜底：源码文本中 useMemo deps 数组必须包含 themeMode 与 density。
 *
 * 约束：不改 AntdThemeBridge.tsx；不改 src/components/ConfigProvider.tsx（P-C2 冻结）。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render } from "@testing-library/react";
import { readFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { theme as antdTheme } from "antd";

// settingsStore 可控 mock：rerender 时按 prefsState 当前值重新取数
const prefsState = vi.hoisted(() => ({
  mode: "light" as string,
  density: "normal" as string,
}));
vi.mock("@/store/settingsStore", () => ({
  useSettingsStore: vi.fn((selector: (s: unknown) => unknown) =>
    selector({
      preferences: {
        theme: { mode: prefsState.mode },
        layout: { density: prefsState.density },
      },
    }),
  ),
}));

import AntdThemeBridge from "../AntdThemeBridge";

// 探针：记录 ConfigProvider 下发的 theme token 引用（每次渲染追加一条）
const seenTokens: object[] = [];
const TokenProbe = () => {
  const { token } = antdTheme.useToken();
  seenTokens.push(token);
  return null;
};

const mountBridge = () =>
  render(
    <AntdThemeBridge>
      <TokenProbe />
    </AntdThemeBridge>,
  );

describe("design-system/components/AntdThemeBridge useMemo 依赖覆盖（C15）", () => {
  beforeEach(() => {
    prefsState.mode = "light";
    prefsState.density = "normal";
    seenTokens.length = 0;
  });

  it("themeMode 变化 → antd theme token 重新计算（引用变化）", () => {
    const { rerender } = mountBridge();
    const before = seenTokens[seenTokens.length - 1];
    expect(before).toBeTruthy();

    prefsState.mode = "dark";
    rerender(
      <AntdThemeBridge>
        <TokenProbe />
      </AntdThemeBridge>,
    );
    const after = seenTokens[seenTokens.length - 1];
    expect(seenTokens.length).toBeGreaterThan(1);
    // darkAlgorithm 切换必须触发 antdThemeConfig 重算 → token 新对象
    expect(after).not.toBe(before);
  });

  it("density 变化 → antd theme token 重新计算（引用变化）", () => {
    const { rerender } = mountBridge();
    const before = seenTokens[seenTokens.length - 1];

    prefsState.density = "compact";
    rerender(
      <AntdThemeBridge>
        <TokenProbe />
      </AntdThemeBridge>,
    );
    const after = seenTokens[seenTokens.length - 1];
    // compactAlgorithm 叠加必须触发 antdThemeConfig 重算 → token 新对象
    expect(after).not.toBe(before);
  });

  it("静态兜底：useMemo deps 数组覆盖全部外部状态 { themeMode, density }", () => {
    const srcPath = resolve(
      dirname(fileURLToPath(import.meta.url)),
      "..",
      "AntdThemeBridge.tsx",
    );
    const src = readFileSync(srcPath, "utf-8");
    const m = src.match(/antdThemeConfig\s*=\s*useMemo[\s\S]*?\},\s*\[([^\]]*)\]\)/);
    expect(m, "未找到 antdThemeConfig 的 useMemo deps 数组").toBeTruthy();
    const deps = m![1].split(",").map((d) => d.trim()).filter(Boolean);
    expect(deps).toContain("themeMode");
    expect(deps).toContain("density");
    // deps 不允许出现 mock 外的未知输入源（防漂移提示，非硬性白名单）
    expect(deps.length).toBe(2);
  });
});

describe("Phase 89 · STYLE-01 明暗色板", () => {
  beforeEach(() => {
    prefsState.mode = "light";
    prefsState.density = "normal";
    seenTokens.length = 0;
  });

  type ComponentTokens = { colorBgContainer?: string };
  type ProbeToken = {
    colorBgContainer: string;
    colorBorder: string;
    colorBgBase: string;
    Select?: ComponentTokens;
    Input?: ComponentTokens;
    Card?: ComponentTokens;
  };
  const tokenOf = () => seenTokens[seenTokens.length - 1] as unknown as ProbeToken;
  const setMode = (mode: string) => {
    prefsState.mode = mode;
    return render(
      <AntdThemeBridge>
        <TokenProbe />
      </AntdThemeBridge>,
    );
  };

  it("dark 模式：全局 colorBgContainer 不再是 #FFFFFF（= #1a2e1f）", () => {
    const { rerender } = setMode("dark");
    rerender(
      <AntdThemeBridge>
        <TokenProbe />
      </AntdThemeBridge>,
    );
    expect(tokenOf().colorBgContainer).not.toBe("#FFFFFF");
    expect(tokenOf().colorBgContainer.toLowerCase()).toBe("#1a2e1f");
  });

  it("dark 模式：colorBorder 不再是浅色 #DBD7CE", () => {
    const { rerender } = setMode("dark");
    rerender(
      <AntdThemeBridge>
        <TokenProbe />
      </AntdThemeBridge>,
    );
    expect(tokenOf().colorBorder.toLowerCase()).not.toBe("#dbd7ce");
  });

  it("dark 模式：components.Select.colorBgContainer 不再为 #FFFFFF（VISFIX-02）", () => {
    const { rerender } = setMode("dark");
    rerender(
      <AntdThemeBridge>
        <TokenProbe />
      </AntdThemeBridge>,
    );
    // 组件级三项已删（X-1b），实际回落值可能为 undefined → not.toBe 口径
    expect(tokenOf().Select.colorBgContainer?.toLowerCase()).not.toBe("#ffffff");
  });

  it("dark 模式：components.Input / Card.colorBgContainer 不再为 #FFFFFF（X-1b 回归守护）", () => {
    const { rerender } = setMode("dark");
    rerender(
      <AntdThemeBridge>
        <TokenProbe />
      </AntdThemeBridge>,
    );
    const t = tokenOf();
    // 容忍 undefined（antd prepareComponentToken 回落行为），锁死「不得为白」
    expect(t.Input.colorBgContainer?.toLowerCase()).not.toBe("#ffffff");
    expect(t.Card.colorBgContainer?.toLowerCase()).not.toBe("#ffffff");
  });

  it("light 模式：色板不变（colorBgContainer #FFFFFF / colorBorder #DBD7CE），回归守护", () => {
    const { rerender } = setMode("light");
    rerender(
      <AntdThemeBridge>
        <TokenProbe />
      </AntdThemeBridge>,
    );
    expect(tokenOf().colorBgContainer.toLowerCase()).toBe("#ffffff");
    expect(tokenOf().colorBorder.toLowerCase()).toBe("#dbd7ce");
  });

  it("seed 层 colorBgBase 随模式切换（light #f0ece3 / dark #0f2e1b）", () => {
    const l = setMode("light");
    l.rerender(
      <AntdThemeBridge>
        <TokenProbe />
      </AntdThemeBridge>,
    );
    expect(tokenOf().colorBgBase.toLowerCase()).toBe("#f0ece3");

    prefsState.mode = "dark";
    const d = render(
      <AntdThemeBridge>
        <TokenProbe />
      </AntdThemeBridge>,
    );
    d.rerender(
      <AntdThemeBridge>
        <TokenProbe />
      </AntdThemeBridge>,
    );
    expect(tokenOf().colorBgBase.toLowerCase()).toBe("#0f2e1b");
  });
});
