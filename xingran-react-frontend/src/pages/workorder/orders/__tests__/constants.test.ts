/**
 * Phase 88 Batch238 — pages/workorder/orders/constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { WorkOrderStatus, WorkOrderPriority, WorkOrderType } from "@/lib/workorderApi";
import { STATUS_CONFIG, PRIORITY_CONFIG, TYPE_CONFIG } from "../constants";

describe("workorder/orders/constants", () => {
  it("STATUS_CONFIG 5 状态", () => {
    expect(STATUS_CONFIG[WorkOrderStatus.Pending].text).toBe("待处理");
    expect(STATUS_CONFIG[WorkOrderStatus.Processing].text).toBe("处理中");
    expect(STATUS_CONFIG[WorkOrderStatus.Completed].text).toBe("已完成");
    expect(STATUS_CONFIG[WorkOrderStatus.Closed].text).toBe("已关闭");
    expect(STATUS_CONFIG[WorkOrderStatus.Rejected].text).toBe("已拒绝");
  });

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
});
