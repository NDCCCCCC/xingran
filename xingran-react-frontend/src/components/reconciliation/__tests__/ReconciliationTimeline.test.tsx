/**
 * Phase 88 Batch137 — components/reconciliation/ReconciliationTimeline 测试
 *
 * 字典驱动:冲突类型文案口径为「字典 → 静态回退 → 原始值」三级,
 * 本文件同时覆盖字典命中(dictLabel)与字典未命中(静态回退)两条路径。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("../HealthBadge", () => ({
  HealthBadge: ({ conflictType }: unknown) => (
    <span data-testid="health-badge">{conflictType}</span>
  ),
}));

// 组件改走 useDict —— mock 掉字典 hook,测试不依赖 React Query Provider
vi.mock("@/hooks/useDict", () => ({
  useDict: vi.fn(() => ({ data: [] })),
}));

import { useDict } from "@/hooks/useDict";
import { ReconciliationTimeline } from "../ReconciliationTimeline";

const mockedUseDict = vi.mocked(useDict);

/** 按字典类型注入字典数据(其他类型返回空数组) */
function mockDict(dictType: string, items: Array<{ dictValue: string; dictLabel: string }>): void {
  mockedUseDict.mockImplementation(((type: string) => ({
    data: type === dictType ? items : [],
  })) as unknown as typeof useDict);
}

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const baseRecord = {
  id: "r1",
  conflictType: "B",
  detectedAt: "2026-01-01T00:00:00Z",
  resolvedAt: "2026-01-02T00:00:00Z",
  resolvedByUsername: "admin",
};

describe("ReconciliationTimeline", () => {
  beforeEach(() => {
    // 默认字典为空 → 走静态回退文案
    mockedUseDict.mockImplementation((() => ({ data: [] })) as unknown as typeof useDict);
  });

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
        ...baseRecord,
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
        ...baseRecord,
        conflictType: "A",
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
        ...baseRecord,
        conflictType: "Z",
      },
    ];
    const { baseElement } = render(<ReconciliationTimeline records={records} loading={false} />, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("Z");
  });

  it("字典命中 → 显示 dictLabel(而非静态回退文案)", () => {
    mockDict("asset_reconciliation_conflict_type", [
      { dictValue: "B", dictLabel: "B类-物理有/责任人无" },
    ]);
    const { baseElement } = render(
      <ReconciliationTimeline records={[baseRecord]} loading={false} />,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-tag")?.textContent).toBe("B类-物理有/责任人无");
  });
});
