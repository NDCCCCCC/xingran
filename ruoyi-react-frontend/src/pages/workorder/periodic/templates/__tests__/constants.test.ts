/**
 * Phase 88 Batch201 — pages/workorder/periodic/templates/constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { WorkOrderPriority, WorkOrderType, PeriodicAssignType } from "@/lib/workorderApi";
import {
  PRIORITY_CONFIG,
  TYPE_CONFIG,
  ASSIGN_TYPE_CONFIG,
  DEFAULT_FORM_VALUES,
} from "../constants";

describe("workorder/periodic/templates/constants", () => {
  it("PRIORITY_CONFIG 4 优先级", () => {
    expect(PRIORITY_CONFIG[WorkOrderPriority.Low].text).toBe("低");
    expect(PRIORITY_CONFIG[WorkOrderPriority.Medium].text).toBe("中");
    expect(PRIORITY_CONFIG[WorkOrderPriority.High].text).toBe("高");
    expect(PRIORITY_CONFIG[WorkOrderPriority.Urgent].text).toBe("紧急");
  });

  it("TYPE_CONFIG 5 工单类型", () => {
    expect(TYPE_CONFIG[WorkOrderType.Fault].text).toBe("故障");
    expect(TYPE_CONFIG[WorkOrderType.Request].text).toBe("请求");
    expect(TYPE_CONFIG[WorkOrderType.Change].text).toBe("变更");
    expect(TYPE_CONFIG[WorkOrderType.Incident].text).toBe("事件");
    expect(TYPE_CONFIG[WorkOrderType.Question].text).toBe("咨询");
  });

  it("ASSIGN_TYPE_CONFIG 3 分配类型", () => {
    expect(ASSIGN_TYPE_CONFIG[PeriodicAssignType.Manual].text).toBe("手动指定");
    expect(ASSIGN_TYPE_CONFIG[PeriodicAssignType.DutyPool].text).toBe("当天值班人员");
    expect(ASSIGN_TYPE_CONFIG[PeriodicAssignType.Rotation].text).toBe("轮询");
  });

  it("DEFAULT_FORM_VALUES", () => {
    expect(DEFAULT_FORM_VALUES.type).toBe(WorkOrderType.Fault);
    expect(DEFAULT_FORM_VALUES.priority).toBe(WorkOrderPriority.Medium);
    expect(DEFAULT_FORM_VALUES.assignType).toBe(PeriodicAssignType.DutyPool);
    expect(DEFAULT_FORM_VALUES.notifyAssignee).toBe(true);
  });
});
