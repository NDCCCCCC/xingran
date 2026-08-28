/**
 * Phase 88 Batch15 — design-system 组件 ConfigProvider 链路测试
 * 目标: 53% → 70%+ (AntdThemeBridge/ThemeProvider/LayoutProvider/DensitySwitcher/LayoutSwitcher/PageTitle/SettingsShell)
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ConfigProvider } from "antd";
import { MemoryRouter } from "react-router-dom";
import AntdThemeBridge from "../AntdThemeBridge";
import { ThemeProvider } from "../ThemeProvider";
import LayoutProvider from "../LayoutProvider";
import DensitySwitcher from "../DensitySwitcher";
import LayoutSwitcher from "../LayoutSwitcher";
import PageTitle from "../PageTitle";
import { SettingsShell } from "../SettingsShell";

vi.mock("@/store/settingsStore", () => ({
  useSettingsStore: vi.fn((sel?: any) => {
    const state = {
      preferences: {
        theme: { mode: "light" },
        layout: { density: "comfortable" },
      },
    };
    return typeof sel === "function" ? sel(state) : state;
  }),
}));

vi.mock("@/store/layoutStore", () => ({
  useLayoutStore: () => ({
    currentLayout: "hybrid",
    sidebarCollapsed: false,
    density: "comfortable",
    layout: "hybrid",
    setDensity: vi.fn(),
    setLayout: vi.fn(),
  }),
  useLayout: () => ({
    density: "comfortable",
    setDensity: vi.fn(),
    layout: "hybrid",
    setLayout: vi.fn(),
  }),
}));

vi.mock("@/store/themeStore", () => ({
  useThemeStore: vi.fn((sel?: any) => {
    const state = { mode: "light" as const, setMode: vi.fn() };
    return typeof sel === "function" ? sel(state) : state;
  }),
}));

vi.mock("@/utils/antdMessage", () => ({
  setAppMessageInstance: vi.fn(),
}));

function wrap(ui: React.ReactElement, opts: { initialEntries?: string[] } = {}) {
  return render(
    <MemoryRouter initialEntries={opts.initialEntries ?? ["/"]}>
      <ThemeProvider>
        <ConfigProvider>
          <AntdThemeBridge>{ui}</AntdThemeBridge>
        </ConfigProvider>
      </ThemeProvider>
    </MemoryRouter>
  );
}

describe("design-system/components — themes", () => {
  it("PageTitle renders pre/post/sub/actions", () => {
    const { container } = wrap(
      <PageTitle pre="系统" post="用户" sub="账号管理" actions={<button>新增</button>} />
    );
    const html = container.innerHTML;
    expect(html).toContain("系统");
    expect(html).toContain("用户");
    expect(html).toContain("账号管理");
    expect(html).toContain("新增");
    expect(container.querySelector(".dot")).toBeTruthy();
  });

  it("PageTitle without post omits dot", () => {
    const { container } = wrap(<PageTitle pre="总览" />);
    expect(container.querySelector(".dot")).toBeNull();
  });

  it("DensitySwitcher renders 3 options via Segmented", () => {
    wrap(<DensitySwitcher />);
    expect(screen.getByText("紧凑")).toBeTruthy();
    expect(screen.getByText("舒适")).toBeTruthy();
    expect(screen.getByText("宽松")).toBeTruthy();
  });

  it("LayoutSwitcher renders with layout-switcher class", () => {
    const { container } = wrap(<LayoutSwitcher />);
    expect(container.querySelector(".layout-switcher")).toBeTruthy();
  });

  it("SettingsShell renders categories sidebar (desktop)", () => {
    const cats = [
      { key: "a", label: "分类A", icon: <span>A</span>, content: <div>内容A</div> },
      { key: "b", label: "分类B", icon: <span>B</span>, content: <div>内容B</div> },
    ];
    wrap(<SettingsShell categories={cats} defaultCat="a" />, {
      initialEntries: ["/?cat=a"],
    });
    expect(screen.getByText("分类A")).toBeTruthy();
    expect(screen.getByText("分类B")).toBeTruthy();
    expect(screen.getByText("内容A")).toBeTruthy();
  });

  it("SettingsShell falls back to defaultCat on invalid param", () => {
    const cats = [
      { key: "a", label: "分类A", icon: <span>A</span>, content: <div>内容A</div> },
      { key: "b", label: "分类B", icon: <span>B</span>, content: <div>内容B</div> },
    ];
    wrap(<SettingsShell categories={cats} defaultCat="b" />, {
      initialEntries: ["/?cat=invalid"],
    });
    expect(screen.getByText("内容B")).toBeTruthy();
  });

  it("AntdThemeBridge renders ConfigProvider + App wrapper", () => {
    const { container } = render(
      <MemoryRouter>
        <AntdThemeBridge>
          <span data-testid="bridge-child">子节点</span>
        </AntdThemeBridge>
      </MemoryRouter>
    );
    expect(screen.getByTestId("bridge-child")).toBeTruthy();
    expect(container.firstChild).toBeTruthy();
  });

  it("ThemeProvider sets data-color-mode on documentElement", () => {
    render(
      <MemoryRouter>
        <ThemeProvider>
          <span>x</span>
        </ThemeProvider>
      </MemoryRouter>
    );
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("light");
  });

  it("LayoutProvider sets data-layout/data-density on documentElement", () => {
    render(
      <MemoryRouter>
        <LayoutProvider>
          <div data-testid="lp-child">LP</div>
        </LayoutProvider>
      </MemoryRouter>
    );
    expect(screen.getByTestId("lp-child")).toBeTruthy();
    expect(document.documentElement.getAttribute("data-layout")).toBe("hybrid");
    expect(document.documentElement.getAttribute("data-density")).toBe("comfortable");
  });
});
