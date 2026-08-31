/**
 * Phase 88 Batch239 — pages/network/command/types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type { CommandStatistics, CommandModalState } from "../types";

describe("network/command/types", () => {
  it("CommandStatistics shape", () => {
    const s: CommandStatistics = {
      total: 100,
      pending: 10,
      running: 5,
      success: 80,
      failed: 5,
    };
    expect(s.total).toBe(100);
  });

  it("CommandModalState shape", () => {
    const s: CommandModalState = {
      dispatchModalVisible: true,
      detailDrawerVisible: false,
      selectedRowKeys: ["a", "b"],
      currentExecution: null,
      executionDetails: [],
    };
    expect(s.dispatchModalVisible).toBe(true);
    expect(s.selectedRowKeys.length).toBe(2);
  });
});
