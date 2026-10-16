/**
 * Phase 84 84-01a Task 2 — Excel 导入导出测试
 * ExcelImport / ExcelImportLazy / ExcelExport
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ExcelExport from "../ExcelExport";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});
vi.mock("@/lib/opsApi", () => ({
  excelApi: { export: vi.fn(() => Promise.resolve()) },
}));

describe("ExcelExport", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders export button when visible=true", () => {
    renderWithProviders(
      <ExcelExport entityType="building" entityName="楼宇" visible onClose={vi.fn()} />
    );
    expect(screen.getByRole("button", { name: /导出/i })).not.toBeNull();
  });

  it("accepts custom filters prop without error", () => {
    renderWithProviders(
      <ExcelExport
        entityType="workstation"
        entityName="工位"
        visible
        filters={{ buildingId: "b-1", floorId: "f-2" }}
        onClose={vi.fn()}
      />
    );
    expect(screen.getByRole("button", { name: /导出/i })).not.toBeNull();
  });

  it("renders with onClose callback", () => {
    const onClose = vi.fn();
    renderWithProviders(<ExcelExport entityType="device" visible onClose={onClose} />);
    expect(onClose).not.toHaveBeenCalled();
  });
});
