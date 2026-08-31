/**
 * Phase 88 Batch257 — pages/network/executions/types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type {
  ExecutionStatus,
  ExecutionStatistics,
  ModalState,
  ExecutionDataState,
} from "../types";

describe("network/executions/types", () => {
  it("ExecutionStatus 4 值", () => {
    const s: ExecutionStatus[] = ["pending", "running", "success", "failed"];
    expect(s.length).toBe(4);
  });

  it("ExecutionStatistics shape", () => {
    const s: ExecutionStatistics = {
      total: 100,
      pending: 5,
      running: 2,
      success: 90,
      failed: 3,
    };
    expect(s.total).toBe(100);
  });

  it("ModalState shape", () => {
    const s: ModalState = {
      executeModalVisible: true,
      variableModalVisible: false,
      detailDrawerVisible: false,
    };
    expect(s.executeModalVisible).toBe(true);
  });

  it("ExecutionDataState shape", () => {
    const s: ExecutionDataState = {
      devices: [],
      templates: [],
      executions: [],
      executionDetails: [],
      currentExecution: null,
      selectedTemplate: null,
    };
    expect(s.devices.length).toBe(0);
  });
});
