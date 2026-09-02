/**
 * Phase 88 Batch419 — pages/system/notice 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import NoticePage from "../index";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/noticeApi", () => ({
  getNoticeList: vi.fn(async () => ({ list: [], total: 0 })),
  createNotice: vi.fn(async () => ({})),
  updateNotice: vi.fn(async () => ({})),
  deleteNotice: vi.fn(async () => ({})),
  publishNotice: vi.fn(async () => ({})),
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/system/notice"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("notice page render", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<NoticePage />, { wrapper: Wrap })).not.toThrow();
  });
});