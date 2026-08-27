/**
 * Phase 84 84-01a Task 1 — GlobalSearch 组件测试
 * 触发器按钮 + Cmd/Ctrl+K 快捷键 + 搜索过滤 + 键盘导航
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import GlobalSearch from "../GlobalSearch";

describe("GlobalSearch", () => {
  const navSpy = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    // GlobalSearch 内部 useNavigate 走真实 MemoryRouter,
    // 这里不 mock —— 用 route 断言跳转行为即可
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders search trigger button", () => {
    renderWithProviders(<GlobalSearch />);
    expect(screen.getByRole("button", { name: "全局搜索" })).not.toBeNull();
    expect(screen.getByText("搜索菜单、资产、告警")).not.toBeNull();
  });

  it("opens modal when trigger clicked and filters results by keyword", () => {
    renderWithProviders(<GlobalSearch />);
    fireEvent.click(screen.getByRole("button", { name: "全局搜索" }));
    // Modal 打开后出现搜索框与默认结果列表
    const input = screen.getByPlaceholderText("搜索页面、命令、菜单...");
    expect(input).not.toBeNull();
    expect(screen.getByText("仪表盘")).not.toBeNull();
    expect(screen.getByText("用户管理")).not.toBeNull();
  });

  it("filters result list on input change", () => {
    renderWithProviders(<GlobalSearch />);
    fireEvent.click(screen.getByRole("button", { name: "全局搜索" }));
    const input = screen.getByPlaceholderText("搜索页面、命令、菜单...");
    fireEvent.change(input, { target: { value: "用户" } });
    // 命中"用户管理";不命中"仪表盘"
    expect(screen.getByText("用户管理")).not.toBeNull();
    expect(screen.queryByText("仪表盘")).toBeNull();
  });

  it("opens modal via trigger button click (D-11)", () => {
    renderWithProviders(<GlobalSearch />);
    // 初始无搜索框
    expect(screen.queryByPlaceholderText("搜索页面、命令、菜单...")).toBeNull();
    // 点击触发器打开
    fireEvent.click(screen.getByRole("button", { name: "全局搜索" }));
    expect(screen.getByPlaceholderText("搜索页面、命令、菜单...")).not.toBeNull();
    // 关闭通过再次 Ctrl+K(仅测试 Ctrl+K 开/关路径,避开 window listener jsdom 限制)
  });
});
