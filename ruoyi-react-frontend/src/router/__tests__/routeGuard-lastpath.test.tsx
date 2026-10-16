/**
 * Phase 88 Batch17c — RouteGuard 权限分支 + routeConfigManager lastPath 工具
 */
import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { RouteGuard } from "../RouteGuard";
import { useMenuStore } from "@/store/menuStore";
import { getLastPath, saveLastPath, clearLastPath } from "../DynamicRoutes";

describe("RouteGuard", () => {
  beforeEach(() => {
    useMenuStore.setState({ permissions: [] });
  });

  const wrap = (ui: React.ReactElement) => render(<MemoryRouter>{ui}</MemoryRouter>);

  it("empty permissions → children rendered", () => {
    wrap(
      <RouteGuard permissions={[]}>
        <div data-testid="ok">内容</div>
      </RouteGuard>
    );
    expect(screen.getByTestId("ok")).not.toBeNull();
  });

  it("has permission → children rendered", () => {
    useMenuStore.setState({ permissions: ["system:user:list"] });
    wrap(
      <RouteGuard permissions={["system:user:list"]}>
        <div data-testid="ok">内容</div>
      </RouteGuard>
    );
    expect(screen.getByTestId("ok")).not.toBeNull();
  });

  it("no permission → 403 Result (default fallbackElement)", () => {
    wrap(
      <RouteGuard permissions={["system:user:list"]}>
        <div data-testid="ok">内容</div>
      </RouteGuard>
    );
    expect(screen.getByText("抱歉，您无权访问此页面。")).not.toBeNull();
    expect(screen.queryByTestId("ok")).toBeNull();
  });

  it("no permission with fallback path → Navigate redirect (children hidden)", () => {
    wrap(
      <RouteGuard permissions={["x"]} fallback="/home">
        <div data-testid="ok">内容</div>
      </RouteGuard>
    );
    expect(screen.queryByTestId("ok")).toBeNull();
  });

  it("no permission with custom fallbackElement → renders it", () => {
    wrap(
      <RouteGuard permissions={["x"]} fallbackElement={<div data-testid="custom">自定义</div>}>
        <div data-testid="ok">内容</div>
      </RouteGuard>
    );
    expect(screen.getByTestId("custom")).not.toBeNull();
  });
});

describe("routeConfigManager lastPath utils", () => {
  beforeEach(() => {
    clearLastPath();
  });

  it("getLastPath returns null when nothing saved", () => {
    expect(getLastPath()).toBeNull();
  });

  it("saveLastPath + getLastPath roundtrip", () => {
    saveLastPath("/system/user");
    expect(getLastPath()).toBe("/system/user");
  });

  it("clearLastPath removes saved path", () => {
    saveLastPath("/ops/building");
    clearLastPath();
    expect(getLastPath()).toBeNull();
  });
});
