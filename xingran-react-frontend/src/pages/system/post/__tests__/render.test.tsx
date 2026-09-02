/**
 * Phase 88 Batch418 — pages/system/post 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import PostPage from "../index";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/postApi", () => ({
  getPostList: vi.fn(async () => ({ list: [], total: 0 })),
  createPost: vi.fn(async () => ({})),
  updatePost: vi.fn(async () => ({})),
  deletePost: vi.fn(async () => ({})),
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/system/post"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("post page render", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<PostPage />, { wrapper: Wrap })).not.toThrow();
  });
});
