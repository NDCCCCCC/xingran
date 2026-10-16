/**
 * Phase 88 Batch405 — pages/duty/my-duty 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/dutyApi", () => ({
  getMyDutyStats: vi.fn(async () => ({})),
  getDutyScheduleList: vi.fn(async () => ({ data: { list: [], total: 0 } })),
  getDutyPoolsList: vi.fn(async () => ({ data: { list: [], total: 0 } })),
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/duty/my-duty"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("pages/duty/my-duty", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../index");
    expect(typeof mod.default).toBe("function");
  });

  it("基础渲染不抛错", async () => {
    const { default: Comp } = await import("../index");
    expect(() => render(<Comp />, { wrapper })).not.toThrow();
  }, 15000);

  // WR-02 regression: handleSearch must pass explicit page=1 to loadSchedules,
  // not rely on stale captured paginationProps.current.
  it("handleSearch → loadSchedules called with page=1 (explicit arg)", async () => {
    const { default: Comp } = await import("../index");
    const { baseElement } = render(<Comp />, { wrapper });

    const { getDutyScheduleList } = await import("@/lib/dutyApi");
    vi.mocked(getDutyScheduleList).mockClear();

    // Wait for the page to finish initial load
    const { waitFor, fireEvent, act } = await import("@testing-library/react");
    await waitFor(() => {
      expect(baseElement.querySelector(".ant-table-wrapper, .ant-card")).toBeTruthy();
    });

    // Open the dutyType Select via mouseDown
    const selectTrigger = baseElement.querySelector(".ant-select");
    expect(selectTrigger).toBeTruthy();

    await act(async () => {
      fireEvent.mouseDown(selectTrigger as HTMLElement);
    });

    // Select "工作日" from dropdown
    await act(async () => {
      const item = document.querySelector(
        ".ant-select-dropdown .ant-select-item-option[title='工作日']"
      ) as HTMLElement;
      expect(item).toBeTruthy();
      fireEvent.click(item);
    });

    // loadSchedules must have been called with page=1
    const lastCall = vi.mocked(getDutyScheduleList).mock.calls.at(-1);
    expect(lastCall).toBeDefined();
    expect(lastCall![0]).toMatchObject({ current: 1 });
  });
});
