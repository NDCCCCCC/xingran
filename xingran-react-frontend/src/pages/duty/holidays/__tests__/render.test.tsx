/**
 * Phase 88 Batch420 — pages/duty/holidays 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import HolidaysPage from "../index";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/dutyApi", () => ({
  getHolidayList: vi.fn(async () => ({ list: [], total: 0 })),
  createHoliday: vi.fn(async () => ({})),
  updateHoliday: vi.fn(async () => ({})),
  deleteHoliday: vi.fn(async () => ({})),
  batchImportHolidays: vi.fn(async () => ({})),
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/duty/holidays"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("duty holidays page render", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<HolidaysPage />, { wrapper: Wrap })).not.toThrow();
  });
});