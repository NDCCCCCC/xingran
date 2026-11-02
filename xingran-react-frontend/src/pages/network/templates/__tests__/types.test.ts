/**
 * Phase 88 Batch241 — pages/network/templates/types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type { TemplateStatistics, TemplateModalState, SelectOption } from "../types";

describe("network/templates/types", () => {
  it("TemplateStatistics shape", () => {
    const s: TemplateStatistics = {
      total: 50,
      system: 20,
      custom: 25,
      init: 5,
    };
    expect(s.total).toBe(50);
  });

  it("TemplateModalState shape", () => {
    const s: TemplateModalState = {
      editModalVisible: true,
      previewVisible: false,
      variablesModalVisible: false,
      editingTemplate: null,
      selectedRowKeys: ["a", "b"],
      previewContent: "content",
      templateVariables: { var1: "x" },
    };
    expect(s.previewContent).toBe("content");
  });

  it("SelectOption shape", () => {
    const o: SelectOption = { label: "L", value: "v" };
    expect(o.label).toBe("L");
  });
});
