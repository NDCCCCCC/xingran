/**
 * Phase 88 Batch32 — MACEventsTimeline 组件测试(原 2/47)
 *
 * 走 renderPageWithEndpoints 模式注册 /network/history/list 端点。
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import { screen } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import MACEventsTimeline from "../MACEventsTimeline";

const eventsPayload = {
  data: {
    list: [
      {
        id: "ev1",
        eventType: "appeared",
        macAddress: "AA:BB:CC:DD:EE:FF",
        deviceId: "d1",
        deviceNameSnapshot: "核心交换机",
        interfaceName: "GE0/0/1",
        vlanId: 10,
        firstSeen: "2026-08-28T10:00:00Z",
        lastSeen: "2026-08-28T10:05:00Z",
        seenCount: 5,
      },
      {
        id: "ev2",
        eventType: "moved",
        macAddress: "AA:BB:CC:DD:EE:FF",
        deviceId: "d2",
        deviceNameSnapshot: "接入交换机",
        interfaceName: "GE0/0/5",
        vlanId: 20,
        firstSeen: "2026-08-28T10:05:00Z",
        lastSeen: "2026-08-28T10:10:00Z",
        seenCount: 3,
      },
    ],
    total: 2,
  },
};

describe("MACEventsTimeline 渲染", () => {
  it("挂载渲染事件条目 + 中文 tag (出现/迁移)", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: 0, gcTime: 0 } },
    });
    renderPageWithEndpoints(
      <MACEventsTimeline
        mac="AA:BB:CC:DD:EE:FF"
        startTime="2026-08-01T00:00:00Z"
        endTime="2026-08-28T23:59:59Z"
      />,
      { endpoints: { "/network/history/list": eventsPayload }, queryClient }
    );

    expect(await screen.findByText("核心交换机")).toBeDefined();
    expect(await screen.findByText("接入交换机")).toBeDefined();
    expect(await screen.findByText("GE0/0/1")).toBeDefined();
    expect(await screen.findByText("GE0/0/5")).toBeDefined();
    expect((await screen.findAllByText("出现")).length).toBeGreaterThanOrEqual(1);
    expect((await screen.findAllByText("迁移")).length).toBeGreaterThanOrEqual(1);
  }, 20000);
});
