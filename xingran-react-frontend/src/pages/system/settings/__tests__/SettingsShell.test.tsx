/**
 * SettingsShell 单测（v1.56 目录化改造后重写）
 *
 * 改造前锁定「一次只显示一个分类」；目录化后语义变为：
 *   左侧目录 = 文档目录，右侧**全部分类纵向平铺**，目录点击 = 锚点滚动 + URL 双写。
 *
 * 用例覆盖：
 *   1. 平铺：全部分类内容同时渲染（不再互相排斥）。
 *   2. ?cat= 定位：URL 参数决定目录激活项（aria-current 恰好一个）。
 *   3. 点击目录：URL 同步为 `?cat=key#key`（MemoryRouter 探针断言）+ 触发 scrollIntoView。
 *   4. 非法 ?cat= 回退 defaultCat，不抛错。
 *   5. 断点降级：桌面走 Sider 目录、窄屏走 Segmented 目录，两者内容同样平铺。
 *   6. 区块锚点：每分类一个 section，id = key，aria-labelledby 指向区块标题。
 *
 * mock 策略：vi.mock("antd") 仅替换 Grid.useBreakpoint，保留其他 antd 导出；
 * jsdom 无 scrollIntoView 实现，用 prototype 打桩捕捉调用。
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render } from "@testing-library/react";
import { screen, fireEvent } from "@testing-library/dom";
import { App } from "antd";
import { MemoryRouter, useLocation } from "react-router-dom";
import type { ReactNode } from "react";

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

// ---- Mock: antd Grid.useBreakpoint 控制桌面/窄屏分支 ----
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

import { SettingsShell } from "@/design-system/components/SettingsShell";
import type { SettingsCategory } from "@/design-system/components/SettingsShell";

// ---- 夹具：3 个测试用分类，每项 content 含可断言文本 ----
const testCategories: SettingsCategory[] = [
  {
    key: "email",
    label: "邮箱配置",
    icon: <span data-testid="icon-email" />,
    content: <div data-testid="content-email">content-email</div>,
  },
  {
    key: "api",
    label: "API配置",
    icon: <span data-testid="icon-api" />,
    content: <div data-testid="content-api">content-api</div>,
  },
  {
    key: "captcha",
    label: "验证码背景图",
    icon: <span data-testid="icon-captcha" />,
    content: <div data-testid="content-captcha">content-captcha</div>,
  },
];

/** URL 探针：断言目录点击后 search + hash 同步（MemoryRouter 不暴露 history 浏览器栈） */
function LocationProbe(): ReactNode {
  const location = useLocation();
  return <div data-testid="location-probe">{location.search + location.hash}</div>;
}

function Wrapper({
  children,
  url = "/system/settings-page",
}: {
  children?: ReactNode;
  url?: string;
}) {
  return (
    <MemoryRouter initialEntries={[url]}>
      <App>
        {children}
        <LocationProbe />
      </App>
    </MemoryRouter>
  );
}

function renderShell(url = "/system/settings-page"): void {
  render(
    <Wrapper url={url}>
      <SettingsShell categories={testCategories} defaultCat="email" />
    </Wrapper>
  );
}

describe("SettingsShell — v1.56 目录化（左侧目录 + 右侧平铺）", () => {
  const scrollSpy = vi.fn();

  beforeEach(() => {
    mockBreakpoint = { lg: true };
    scrollSpy.mockClear();
    // jsdom 未实现 scrollIntoView
    HTMLElement.prototype.scrollIntoView = scrollSpy;
  });

  afterEach(() => {
    mockBreakpoint = { lg: true };
  });

  it("用例1: 全部分类内容同时平铺渲染（不再一次只显示一个）", () => {
    renderShell("/system/settings-page?cat=api");

    expect(screen.getByTestId("content-email")).toBeInTheDocument();
    expect(screen.getByTestId("content-api")).toBeInTheDocument();
    expect(screen.getByTestId("content-captcha")).toBeInTheDocument();
  });

  it("用例2: ?cat=api 决定目录激活项，aria-current 恰好一个", () => {
    renderShell("/system/settings-page?cat=api");

    const apiBtn = screen.getByRole("button", { name: /API配置/ });
    expect(apiBtn).toHaveAttribute("aria-current", "true");

    const emailBtn = screen.getByRole("button", { name: /邮箱配置/ });
    expect(emailBtn).not.toHaveAttribute("aria-current");

    const activeBtns = screen
      .getAllByRole("button")
      .filter((b) => b.getAttribute("aria-current") === "true");
    expect(activeBtns).toHaveLength(1);
  });

  it("用例3: 点击目录 → URL 同步为 ?cat=key#key 且触发 scrollIntoView", () => {
    renderShell("/system/settings-page");

    scrollSpy.mockClear();
    fireEvent.click(screen.getByRole("button", { name: /验证码背景图/ }));

    // URL 双写：search 参数兼容旧链接，hash 提供锚点可分享
    expect(screen.getByTestId("location-probe")).toHaveTextContent("?cat=captcha#captcha");
    // 点击后滚动到对应区块（smooth）
    expect(scrollSpy).toHaveBeenCalled();
    const lastCall = scrollSpy.mock.calls[scrollSpy.mock.calls.length - 1];
    expect(lastCall[0]).toMatchObject({ behavior: "smooth", block: "start" });
  });

  it("用例4: ?cat=invalid 回退 defaultCat，不抛错", () => {
    expect(() => renderShell("/system/settings-page?cat=invalid")).not.toThrow();

    const emailBtn = screen.getByRole("button", { name: /邮箱配置/ });
    expect(emailBtn).toHaveAttribute("aria-current", "true");
  });

  it("用例5a: 桌面端渲染 Sider 目录白卡，无 Segmented", () => {
    mockBreakpoint = { lg: true };
    renderShell();

    expect(document.querySelector(".xr-settings-nav")).not.toBeNull();
    expect(document.querySelector(".xr-settings-sider")).not.toBeNull();
    expect(document.querySelector(".ant-segmented")).toBeNull();
  });

  it("用例5b: 窄屏降级为 Segmented 目录，无 Sider，但内容同样全部平铺", () => {
    mockBreakpoint = { lg: false };
    renderShell();

    expect(document.querySelector(".ant-segmented")).not.toBeNull();
    expect(document.querySelector(".xr-settings-nav")).toBeNull();
    expect(document.querySelector(".xr-settings-sider")).toBeNull();

    expect(screen.getByTestId("content-email")).toBeInTheDocument();
    expect(screen.getByTestId("content-api")).toBeInTheDocument();
    expect(screen.getByTestId("content-captcha")).toBeInTheDocument();
  });

  it("用例6: 每分类一个锚点 section（id = key，aria-labelledby 指向区块标题）", () => {
    renderShell();

    for (const c of testCategories) {
      const section = document.querySelector(`section#${c.key}`);
      expect(section).not.toBeNull();
      expect(section?.getAttribute("data-settings-section")).toBe(c.key);

      const titleId = section?.getAttribute("aria-labelledby");
      expect(titleId).toBe(`settings-${c.key}-title`);
      expect(document.getElementById(titleId as string)?.textContent).toContain(c.label);
    }
  });

  it("用例7: 权限码过滤 —— 未就绪时不隐藏（避免闪烁）", () => {
    // 权限集未加载时 useActionPermission 返回 loading，decide().visible = true
    const gatedCategories: SettingsCategory[] = [
      ...testCategories,
      {
        key: "manager",
        label: "经理同步",
        icon: <span />,
        content: <div data-testid="content-manager">content-manager</div>,
        permission: "ops:manager:rule:list",
      },
    ];

    render(
      <Wrapper>
        <SettingsShell categories={gatedCategories} defaultCat="email" />
      </Wrapper>
    );

    // 未就绪 → 仍渲染（不闪烁）
    expect(screen.getByTestId("content-manager")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /经理同步/ })).toBeInTheDocument();
  });
});
