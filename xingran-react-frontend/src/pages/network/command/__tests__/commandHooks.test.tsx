/**
 * Phase 88 Batch29 — network/command 钩子 + modals + columns 测试(原 0%)
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, Form } from "antd";
import { render } from "@testing-library/react";
import { ConfigProvider } from "antd";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useCommandData } from "../hooks/useCommandData";
import { useCommandModals } from "../hooks/useCommandModals";
import { CommandDetailDrawer } from "../modals/DetailDrawer";
import { CommandDispatchModal } from "../modals/DispatchModal";
import * as apiModule from "@/lib/api";

const postSpy = vi.fn();

function Wrap({ children }: { children: React.ReactNode }) {
  return (
    <ConfigProvider>
      <App>{children}</App>
    </ConfigProvider>
  );
}

beforeEach(() => {
  postSpy.mockReset();
  vi.spyOn(apiModule, "post" as any).mockImplementation(postSpy as any);
});

describe("useCommandData", () => {
  it("loadExecutions 成功写 state", async () => {
    postSpy.mockResolvedValue({
      data: { list: [{ id: "e1" }], total: 1 },
    });
    const setExecLoading = vi.fn();
    const { result } = renderHook(() => useCommandData(setExecLoading, { current: 1, pageSize: 10 }), {
      wrapper: Wrap,
    });

    await act(async () => {
      await result.current.loadExecutions();
    });

    expect(postSpy).toHaveBeenCalledWith("/network/command/list", { current: 1, pageSize: 10 });
    expect(result.current.executions).toEqual([{ id: "e1" }]);
    expect(result.current.execTotal).toBe(1);
    expect(setExecLoading).toHaveBeenCalledWith(true);
    expect(setExecLoading).toHaveBeenCalledWith(false);
  });

  it("loadExecutions error 静默", async () => {
    postSpy.mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useCommandData(vi.fn(), { current: 1, pageSize: 10 }), {
      wrapper: Wrap,
    });
    await act(async () => {
      await result.current.loadExecutions();
    });
    expect(result.current.executions).toEqual([]);
  });

  it("loadStatistics 写 5 字段", async () => {
    postSpy.mockResolvedValue({
      data: { total: 10, pending: 1, running: 2, success: 6, failed: 1 },
    });
    const { result } = renderHook(() => useCommandData(vi.fn(), { current: 1, pageSize: 10 }), {
      wrapper: Wrap,
    });
    await act(async () => {
      await result.current.loadStatistics();
    });
    expect(result.current.statistics).toEqual({
      total: 10,
      pending: 1,
      running: 2,
      success: 6,
      failed: 1,
    });
  });

  it("loadDevices 返回 list", async () => {
    postSpy.mockResolvedValue({ data: { list: [{ id: "d1" }], total: 1 } });
    const { result } = renderHook(() => useCommandData(vi.fn(), { current: 1, pageSize: 10 }), {
      wrapper: Wrap,
    });
    let devices: unknown;
    await act(async () => {
      devices = await result.current.loadDevices();
    });
    expect(devices).toEqual([{ id: "d1" }]);
  });

  it("loadDevices error 返回 []", async () => {
    postSpy.mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useCommandData(vi.fn(), { current: 1, pageSize: 10 }), {
      wrapper: Wrap,
    });
    let devices: unknown;
    await act(async () => {
      devices = await result.current.loadDevices();
    });
    expect(devices).toEqual([]);
  });

  it("loadExecutionDetails 返回 execution+details", async () => {
    postSpy.mockResolvedValue({
      data: { id: "e1", details: [{ id: "det1" }] },
    });
    const { result } = renderHook(() => useCommandData(vi.fn(), { current: 1, pageSize: 10 }), {
      wrapper: Wrap,
    });
    let ret: any;
    await act(async () => {
      ret = await result.current.loadExecutionDetails("e1");
    });
    expect(ret.execution.id).toBe("e1");
    expect(ret.details).toEqual([{ id: "det1" }]);
  });

  it("loadExecutionDetails error rethrow", async () => {
    postSpy.mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useCommandData(vi.fn(), { current: 1, pageSize: 10 }), {
      wrapper: Wrap,
    });
    await expect(act(async () => {
      await result.current.loadExecutionDetails("e1");
    })).rejects.toThrow("boom");
  });
});

describe("useCommandModals", () => {
  it("handleQuickCommand 成功路径", async () => {
    postSpy.mockResolvedValue({});
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useCommandModals(), { wrapper: Wrap });
    const form = {
      validateFields: vi.fn().mockResolvedValue({ commandContent: "display version" }),
      resetFields: vi.fn(),
    };

    await act(async () => {
      await result.current.handleQuickCommand(["d1"], form as any, onSuccess);
    });

    expect(postSpy).toHaveBeenCalledWith("/network/command/quick", {
      commandContent: "display version",
      deviceIds: ["d1"],
    });
    expect(form.resetFields).toHaveBeenCalled();
    expect(onSuccess).toHaveBeenCalled();
  });

  it("handleQuickCommand 表单校验失败短路", async () => {
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useCommandModals(), { wrapper: Wrap });
    const form = {
      validateFields: vi.fn().mockRejectedValue({ errorFields: [{ name: "cmd" }] }),
      resetFields: vi.fn(),
    };

    await act(async () => {
      await result.current.handleQuickCommand(["d1"], form as any, onSuccess);
    });

    expect(postSpy).not.toHaveBeenCalled();
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("handleCancelExecution 成功路径", async () => {
    postSpy.mockResolvedValue({});
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useCommandModals(), { wrapper: Wrap });

    await act(async () => {
      await result.current.handleCancelExecution("e1", onSuccess);
    });

    expect(postSpy).toHaveBeenCalledWith("/network/executions/e1/cancel", {});
    expect(onSuccess).toHaveBeenCalled();
  });

  it("handleCancelExecution 失败静默", async () => {
    postSpy.mockRejectedValue(new Error("boom"));
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useCommandModals(), { wrapper: Wrap });

    await act(async () => {
      await result.current.handleCancelExecution("e1", onSuccess);
    });
    expect(onSuccess).not.toHaveBeenCalled();
  });
});

describe("CommandDetailDrawer", () => {
  it("execution null 返回 null", () => {
    const { container } = render(
      <CommandDetailDrawer open={false} execution={null} details={[]} onClose={vi.fn()} />
    );
    expect(container.firstChild).toBeNull();
  });

  it("渲染 steps+progress+table", async () => {
    const exec: any = {
      executionName: "批量巡检",
      status: "running",
      totalDevices: 10,
      successCount: 6,
      failureCount: 2,
    };
    const { baseElement, findByText } = render(
      <CommandDetailDrawer open execution={exec} details={[{ id: "det1" } as any]} onClose={vi.fn()} />
    );
    expect(await findByText(/批量巡检/)).toBeDefined();
    expect(await findByText("执行中")).toBeDefined();
    // percent = round((6+2)/10*100) = 80
    expect(baseElement.innerHTML).toContain("80");
  });
});

describe("CommandDispatchModal", () => {
  it("open 渲染设备表+命令框", async () => {
    const { findByText, baseElement } = render(
      <CommandDispatchModal
        open
        devices={[{ id: "d1", deviceName: "核心交换机" } as any]}
        selectedRowKeys={["d1"]}
        onOk={vi.fn()}
        onCancel={vi.fn()}
        onSelectionChange={vi.fn()}
      />
    );
    expect(await findByText("快速命令分发")).toBeDefined();
    expect(await findByText("已选择 1 台设备")).toBeDefined();
    expect(await findByText("命令内容")).toBeDefined();
    expect(baseElement.querySelector(".ant-table")).not.toBeNull();
  });

  it("selectedRowKeys 空时 OK 禁用", async () => {
    const { baseElement, findByText } = render(
      <CommandDispatchModal
        open
        devices={[]}
        selectedRowKeys={[]}
        onOk={vi.fn()}
        onCancel={vi.fn()}
        onSelectionChange={vi.fn()}
      />
    );
    await findByText("快速命令分发");
    const okBtn = baseElement.querySelector(".ant-btn-primary");
    expect(okBtn?.hasAttribute("disabled")).toBe(true);
  });
});
