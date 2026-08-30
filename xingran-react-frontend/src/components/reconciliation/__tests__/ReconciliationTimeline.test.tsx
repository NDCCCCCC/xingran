/**
 * Phase 88 Batch137 — components/reconciliation/ReconciliationTimeline 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("../HealthBadge", () => ({
  HealthBadge: ({ conflictType }: any) => <span data-testid="health-badge">{conflictType}</span>,
}));

import { ReconciliationTimeline } from "../ReconciliationTimeline";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("ReconciliationTimeline", () => {
  it("loading=true → 渲染 Skeleton", () => {
    const { baseElement } = render(<ReconciliationTimeline records={[]} loading />, { wrapper });
    expect(baseElement.querySelector(".ant-skeleton")).toBeTruthy();
  });

  it("records=[] → 渲染 Empty", () => {
    const { baseElement } = render(<ReconciliationTimeline records={[]} loading={false} />, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("暂无已解决的冲突记录");
  });

  it("records 有数据 → 渲染 Timeline + 解决信息", () => {
    const records = [
      {
        id: "r1",
        conflictType: "B",
        detectedAt: "2026-01-01T00:00:00Z",
        resolvedAt: "2026-01-02T00:00:00Z",
        resolvedByUsername: "admin",
        resolutionNote: "fixed",
      },
    ];
    const { baseElement } = render(<ReconciliationTimeline records={records} loading={false} />, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("物理有/责任人无");
    expect(baseElement.textContent).toContain("admin");
    expect(baseElement.textContent).toContain("fixed");
    expect(baseElement.querySelector(".ant-timeline")).toBeTruthy();
  });

  it("resolutionNote 缺失 → 显示 (无)", () => {
    const records = [
      {
        id: "r1",
        conflictType: "A",
        detectedAt: "2026-01-01T00:00:00Z",
        resolvedAt: "2026-01-02T00:00:00Z",
        resolvedByUsername: "admin",
      },
    ];
    const { baseElement } = render(<ReconciliationTimeline records={records} loading={false} />, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("(无)");
  });

  it("未知 conflictType → 显示原始字符", () => {
    const records = [
      {
        id: "r1",
        conflictType: "Z",
        detectedAt: "2026-01-01T00:00:00Z",
        resolvedAt: "2026-01-02T00:00:00Z",
        resolvedByUsername: "admin",
      },
    ];
    const { baseElement } = render(<ReconciliationTimeline records={records} loading={false} />, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("Z");
  });
});
