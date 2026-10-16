/**
 * Phase 70-05 — 用户设置页 D-06 Wave 0 单测（行式控件 onChange → store action 接线锁定）
 *
 * 用例覆盖：
 *   1. 密度模式 Select 切换 → updateLayout({ density: "compact" })。
 *   2. 侧栏折叠 Switch → updateLayout 参数含完整 sidebar 对象（浅合并防护：
 *      既有 width/collapsedWidth 必须随展开保留，错写丢字段会被本用例捕获）。
 *   3. 默认分页大小 Select → updateDataPageSize(50)（数字参数）。
 *   4. 深色模式分段卡点击 → updateTheme({ mode: "dark" })；aria-checked 落在选中卡。
 *   5. 分类注册表完整性：3 项 / key 唯一 / 均限宽 760（D-05 同构约束）。
 *
 * mock 策略：
 *   - vi.hoisted 持有可控 preferences + vi.fn spies（factory 提升规避 TDZ，70-04 先例）。
 *   - vi.mock("antd") 仅替换 Grid.useBreakpoint 固定桌面（lg）分支（70-02 先例）。
 *   - Select 交互：antd v6 dropdown 渲染于 body portal，mouseDown .ant-select 打开
 *     后点击 .ant-select-item-option-content（BulkWriteDrawer.test 先例）。
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, act } from "@testing-library/react";
import { screen, fireEvent, waitFor } from "@testing-library/dom";
import { App } from "antd";
import { MemoryRouter } from "react-router-dom";

// ---- Polyfill: antd v6 ResizeObserver (jsdom 缺失) ----
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
if (typeof globalThis.ResizeObserver === "undefined") {
  (globalThis as unknown as { ResizeObserver: typeof ResizeObserverStub }).ResizeObserver =
    ResizeObserverStub;
}

// ---- Mock: antd Grid.useBreakpoint 固定桌面分支（SettingsShell 走 Sider 导航） ----
let mockBreakpoint: { lg?: boolean } = { lg: true };

vi.mock("antd", async (importOriginal) => {
  const actual = await importOriginal<typeof import("antd")>();
  return {
    ...actual,
    Grid: {
      ...actual.Grid,
      useBreakpoint: () => mockBreakpoint,
    },
  };
});

// ---- Mock: settingsStore 直通可控 preferences + action spies ----
const mocks = vi.hoisted(() => {
  return {
    preferences: {
      version: 2,
      theme: { mode: "light" as const },
      layout: {
        type: "classic" as const,
        sidebar: { collapsed: false, width: 280, collapsedWidth: 64 },
        density: "comfortable" as const,
      },
      data: { defaultPageSize: 10, pageSizeOptions: [10, 20, 50, 100] },
      language: "zh-CN" as const,
    },
    initialized: true,
    updateTheme: vi.fn().mockResolvedValue(undefined),
    updateLayout: vi.fn().mockResolvedValue(undefined),
    updateDataPageSize: vi.fn().mockResolvedValue(undefined),
    initialize: vi.fn().mockResolvedValue(undefined),
  };
});

vi.mock("@/store/settingsStore", () => {
  const fullState = {
    preferences: mocks.preferences,
    initialized: mocks.initialized,
    loading: false,
    error: null,
    version: 2,
    initialize: mocks.initialize,
    updateTheme: mocks.updateTheme,
    updateLayout: mocks.updateLayout,
    updateDataPageSize: mocks.updateDataPageSize,
  };
  return {
    useSettingsStore: (selector?: (s: typeof fullState) => unknown) =>
      selector ? selector(fullState) : fullState,
  };
});

import SettingsPage, { userSettingsCategories } from "@/pages/settings";

function renderPage(cat?: string): void {
  const initialUrl = cat ? `/user/settings?cat=${cat}` : "/user/settings";
  render(
    <MemoryRouter initialEntries={[initialUrl]}>
      <App>
        <SettingsPage />
      </App>
    </MemoryRouter>
  );
}

/** antd v6 Select 交互 helper：mouseDown 打开 dropdown → 点击目标 option 文本 */
async function selectAntdOption(optionText: string): Promise<void> {
  const selectEl = document.querySelector(".ant-select");
  if (!selectEl) throw new Error("ant-select not found");

  await act(async () => {
    fireEvent.mouseDown(selectEl as HTMLElement);
  });

  await waitFor(() => {
    const options = document.querySelectorAll(".ant-select-item-option-content");
    expect(Array.from(options).some((el) => el.textContent === optionText)).toBe(true);
  });

  const target = Array.from(document.querySelectorAll(".ant-select-item-option-content")).find(
    (el) => el.textContent === optionText
  );
  if (!target) throw new Error(`Option not found: ${optionText}`);

  await act(async () => {
    fireEvent.click(target as HTMLElement);
  });
}

describe("用户设置页 — Phase 70-05 D-06 行式设置项接线", () => {
  beforeEach(() => {
    mockBreakpoint = { lg: true };
    vi.clearAllMocks();
  });

  it("用例1: 密度模式 Select 切换紧凑 → updateLayout({ density: 'compact' })", async () => {
    renderPage("layout");

    // 受控初值来自 store preferences（comfortable）
    expect(screen.getByText("密度模式")).toBeInTheDocument();

    await selectAntdOption("紧凑");

    await waitFor(() => {
      expect(mocks.updateLayout).toHaveBeenCalledTimes(1);
    });
    expect(mocks.updateLayout).toHaveBeenCalledWith({ density: "compact" });
  });

  it("用例2: 侧栏折叠 Switch → updateLayout 参数含完整 sidebar 对象（浅合并防护）", async () => {
    renderPage("layout");

    const switchEl = screen.getByRole("switch");
    expect(switchEl).toHaveAttribute("aria-checked", "false");

    await act(async () => {
      fireEvent.click(switchEl);
    });

    await waitFor(() => {
      expect(mocks.updateLayout).toHaveBeenCalledTimes(1);
    });
    // 既有 width/collapsedWidth 必须随展开保留 + collapsed = 新值
    expect(mocks.updateLayout).toHaveBeenCalledWith({
      sidebar: { collapsed: true, width: 280, collapsedWidth: 64 },
    });
  });

  it("用例3: 分页大小 Select 选 50 → updateDataPageSize(50)（数字参数）", async () => {
    renderPage("data");

    await selectAntdOption("50 条/页");

    await waitFor(() => {
      expect(mocks.updateDataPageSize).toHaveBeenCalledTimes(1);
    });
    expect(mocks.updateDataPageSize).toHaveBeenCalledWith(50);
  });

  it("用例4: 点击深色模式分段卡 → updateTheme({ mode: 'dark' })，aria-checked 落在选中卡", async () => {
    renderPage(); // 默认分类 appearance

    const lightCard = screen.getByRole("radio", { name: /浅色模式/ });
    const darkCard = screen.getByRole("radio", { name: /深色模式/ });

    // 受控态：aria-check 与 store preferences.theme.mode（light）一致
    expect(lightCard).toHaveAttribute("aria-checked", "true");
    expect(darkCard).toHaveAttribute("aria-checked", "false");

    await act(async () => {
      fireEvent.click(darkCard);
    });

    await waitFor(() => {
      expect(mocks.updateTheme).toHaveBeenCalledTimes(1);
    });
    expect(mocks.updateTheme).toHaveBeenCalledWith({ mode: "dark" });
  });

  it("用例5: userSettingsCategories 注册表完整（3 分类 / key 唯一 / 均限宽 760）", () => {
    expect(userSettingsCategories).toHaveLength(3);
    expect(userSettingsCategories.map((c) => c.key)).toEqual(["appearance", "layout", "data"]);

    const keys = new Set(userSettingsCategories.map((c) => c.key));
    expect(keys.size).toBe(3);

    for (const category of userSettingsCategories) {
      expect(category.label).toBeTruthy();
      expect(category.icon).toBeTruthy();
      expect(category.content).toBeTruthy();
      expect(category.maxWidth).toBe(760);
    }
  });
});
