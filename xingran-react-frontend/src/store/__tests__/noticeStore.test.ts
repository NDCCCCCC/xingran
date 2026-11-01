/**
 * v1.51 Phase 122 (PORTW-03) — noticeStore 批量写进度事件分流测试
 *
 * 覆盖：onPortWriteProgress 订阅/退订、port_write_progress/completed 分流、
 * Content 缺失与非法 JSON 的降级（不抛错、不触发回调）。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { useNoticeStore } from "../noticeStore";
import type { PortWriteProgressEvent, PortWriteCompletedEvent } from "@/types/notice";

function makeProgressEvent(): PortWriteProgressEvent {
  return {
    type: "port_write_progress",
    batchId: "batch-1",
    deviceId: "d1",
    action: "shutdown",
    portId: "p1",
    interfaceName: "Gi0/1",
    index: 1,
    total: 2,
    status: "succeeded",
  };
}

function makeCompletedEvent(): PortWriteCompletedEvent {
  return {
    type: "port_write_completed",
    batchId: "batch-1",
    deviceId: "d1",
    action: "shutdown",
    portId: null,
    interfaceName: null,
    index: 2,
    total: 2,
    status: "failed",
    summary: '{"succeeded":0,"failed":1,"skipped":0}',
    message: "批次中断（fail-fast）",
  };
}

describe("noticeStore — PORTW-03 批量写进度事件分流", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("port_write_progress 事件经 Content 解析后触发订阅回调", () => {
    const callback = vi.fn();
    const unsubscribe = useNoticeStore.getState().onPortWriteProgress(callback);

    useNoticeStore.getState().handleWsMessage({
      type: "port_write_progress",
      data: null,
      timestamp: Date.now(),
      content: JSON.stringify(makeProgressEvent()),
    } as never);

    expect(callback).toHaveBeenCalledTimes(1);
    expect(callback.mock.calls[0][0]).toMatchObject({
      type: "port_write_progress",
      batchId: "batch-1",
      portId: "p1",
      status: "succeeded",
    });
    unsubscribe();
  });

  it("port_write_completed 事件分流至同一订阅管线", () => {
    const callback = vi.fn();
    const unsubscribe = useNoticeStore.getState().onPortWriteProgress(callback);

    useNoticeStore.getState().handleWsMessage({
      type: "port_write_completed",
      data: null,
      timestamp: Date.now(),
      content: JSON.stringify(makeCompletedEvent()),
    } as never);

    expect(callback).toHaveBeenCalledTimes(1);
    expect(callback.mock.calls[0][0]).toMatchObject({
      type: "port_write_completed",
      message: "批次中断（fail-fast）",
    });
    unsubscribe();
  });

  it("Content 缺失 → 不触发回调且不抛错（降级）", () => {
    const callback = vi.fn();
    const unsubscribe = useNoticeStore.getState().onPortWriteProgress(callback);

    expect(() =>
      useNoticeStore.getState().handleWsMessage({
        type: "port_write_progress",
        data: null,
        timestamp: Date.now(),
      } as never)
    ).not.toThrow();
    expect(callback).not.toHaveBeenCalled();
    unsubscribe();
  });

  it("Content 非法 JSON → 不触发回调且不抛错（降级）", () => {
    const callback = vi.fn();
    const unsubscribe = useNoticeStore.getState().onPortWriteProgress(callback);

    expect(() =>
      useNoticeStore.getState().handleWsMessage({
        type: "port_write_completed",
        data: null,
        timestamp: Date.now(),
        content: "{not-valid-json",
      } as never)
    ).not.toThrow();
    expect(callback).not.toHaveBeenCalled();
    unsubscribe();
  });

  it("退订后回调不再触发", () => {
    const callback = vi.fn();
    const unsubscribe = useNoticeStore.getState().onPortWriteProgress(callback);
    unsubscribe();

    useNoticeStore.getState().handleWsMessage({
      type: "port_write_progress",
      data: null,
      timestamp: Date.now(),
      content: JSON.stringify(makeProgressEvent()),
    } as never);

    expect(callback).not.toHaveBeenCalled();
  });
});
