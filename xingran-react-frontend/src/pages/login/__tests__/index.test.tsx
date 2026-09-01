/**
 * Phase 88 Batch411 — pages/login/index 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/authApi", () => ({
  login: vi.fn(async () => ({ token: "test", refreshToken: "refresh" })),
  getCaptcha: vi.fn(async () => ({ id: "cap1", image: "" })),
}));

vi.mock("@/store/authStore", () => ({
  useAuthStore: vi.fn(() => ({
    login: vi.fn(),
  })),
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter initialEntries={["/login"]}>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("pages/login", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../index");
    expect(typeof mod.default).toBe("function");
  });

  it("基础渲染不抛错", async () => {
    const { default: Comp } = await import("../index");
    expect(() => render(<Comp />, { wrapper })).not.toThrow();
  }, 15000);
});