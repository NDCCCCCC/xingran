/**
 * Phase 88 Batch418 — pages/system/user 渲染
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import UserPage from "../index";
import { createApiMock } from "@/test/utils/createApiMock";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// 页面 useEffect 会拉职位下拉（/system/posts/list），未登记时通用回退 spy
// 返回 undefined 导致 .then 抛 TypeError——此处显式登记安全空结构
beforeEach(() => {
  createApiMock("/system/posts/list").endpoint.mockResolvedValue({
    data: { list: [], total: 0, current: 1, pageSize: 200 },
  });
});

vi.mock("@/lib/userApi", () => ({
  getUserList: vi.fn(async () => ({ list: [], total: 0 })),
  createUser: vi.fn(async () => ({})),
  updateUser: vi.fn(async () => ({})),
  deleteUser: vi.fn(async () => ({})),
  resetPassword: vi.fn(async () => ({})),
}));

vi.mock("@/lib/roleApi", () => ({
  getRoleList: vi.fn(async () => ({ list: [], total: 0 })),
}));

vi.mock("@/lib/dutyApi", () => ({
  getDeptTree: vi.fn(async () => []),
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/system/user"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("user page render", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<UserPage />, { wrapper: Wrap })).not.toThrow();
  });
});
