/**
 * Phase 88 Batch130 — components/shared/ExcelImportLazy 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("../ExcelImport", () => ({
  default: ({ entityType, entityName }: any) => (
    <div data-testid="excel-import">
      <span>{entityType}</span>
      <span>{entityName}</span>
    </div>
  ),
}));

import ExcelImportLazy from "../ExcelImportLazy";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("ExcelImportLazy", () => {
  it("props 透传给懒加载组件", async () => {
    const { baseElement } = render(
      <ExcelImportLazy
        entityType="building"
        entityName="楼宇"
        visible
        onClose={vi.fn()}
        onImportSuccess={vi.fn()}
      />,
      { wrapper }
    );
    await waitFor(() => {
      expect(baseElement.querySelector('[data-testid="excel-import"]')).toBeTruthy();
    });
    expect(baseElement.textContent).toContain("building");
    expect(baseElement.textContent).toContain("楼宇");
  });
});
