/**
 * Phase 88 Batch43 — network executions useExecutionModals 测试
 *
 * renderHook + ConfigProvider/App wrapper,验证 hook 返回的 modalState/handlers
 * 与 handleExecuteByTemplate 的 post + resetFields 链路。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, Modal } from "antd";
import { ConfigProvider } from "antd";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { createApiMock, resetApiMocks, setGenericFallback } from "@/test/utils/createApiMock";
import { useExecutionModals } from "../hooks/useExecutionModals";

function Wrap({ children }: { children: React.ReactNode }) {
  return (
    <ConfigProvider>
      <App>{children}</App>
    </ConfigProvider>
  );
}
const wrap = Wrap;

beforeEach(() => {
  resetApiMocks();
  setGenericFallback({ data: {} });
  vi.clearAllMocks();
});

const baseParams = () => ({
  dataState: {
    executions: [],
    devices: [],
    templates: [{ id: "t1", variables: { host: "1.1.1.1" } }],
    selectedTemplate: null,
    devicesByTemplate: {},
    loading: false,
  },
  setDataState: vi.fn(),
  loadExecutions: vi.fn().mockResolvedValue(undefined),
});

describe("useExecutionModals — initial state & shape", () => {
  it("modalState 3 字段全 false + selectedRowKeys=[] + 9 handlers 存在", () => {
    const { result } = renderHook(() => useExecutionModals(baseParams()), { wrapper: wrap });
    expect(result.current.modalState).toEqual({
      executeModalVisible: false,
      variableModalVisible: false,
      detailDrawerVisible: false,
    });
    expect(result.current.selectedRowKeys).toEqual([]);
    expect(typeof result.current.openExecuteModal).toBe("function");
    expect(typeof result.current.handleTemplateChange).toBe("function");
    expect(typeof result.current.handleExecuteByTemplate).toBe("function");
    expect(typeof result.current.handleCancelExecution).toBe("function");
    expect(typeof result.current.handleViewDetail).toBe("function");
    expect(typeof result.current.handleViewOutput).toBe("function");
    expect(typeof result.current.closeDetailDrawer).toBe("function");
    expect(typeof result.current.closeVariableModal).toBe("function");
    expect(typeof result.current.closeExecuteModal).toBe("function");
  });
});

describe("useExecutionModals — openExecuteModal", () => {
  it("开执行 modal + 清空 selectedRowKeys + 清空 selectedTemplate", async () => {
    const setDataState = vi.fn();
    const params = { ...baseParams(), setDataState };
    const { result } = renderHook(() => useExecutionModals(params), { wrapper: wrap });
    await act(async () => {
      await result.current.openExecuteModal();
    });
    expect(result.current.modalState.executeModalVisible).toBe(true);
    expect(result.current.selectedRowKeys).toEqual([]);
    expect(setDataState).toHaveBeenCalled();
  });
});

describe("useExecutionModals — handleTemplateChange", () => {
  it("找到模板 + 有 variables → 设 selectedTemplate + 开 variableModal", () => {
    const setDataState = vi.fn();
    const params = { ...baseParams(), setDataState };
    const { result } = renderHook(() => useExecutionModals(params), { wrapper: wrap });
    act(() => result.current.handleTemplateChange("t1"));
    expect(setDataState).toHaveBeenCalled();
    expect(result.current.modalState.variableModalVisible).toBe(true);
  });

  it("找到模板 + 无 variables → 设 selectedTemplate + 不开 variableModal", () => {
    const params = {
      ...baseParams(),
      dataState: {
        ...baseParams().dataState,
        templates: [{ id: "t2" }],
      },
    };
    const { result } = renderHook(() => useExecutionModals(params), { wrapper: wrap });
    act(() => result.current.handleTemplateChange("t2"));
    expect(result.current.modalState.variableModalVisible).toBe(false);
  });

  it("未找到模板 → 不调 setDataState", () => {
    const setDataState = vi.fn();
    const params = { ...baseParams(), setDataState };
    const { result } = renderHook(() => useExecutionModals(params), { wrapper: wrap });
    act(() => result.current.handleTemplateChange("not-found"));
    expect(setDataState).not.toHaveBeenCalled();
  });
});

describe("useExecutionModals — handleExecuteByTemplate", () => {
  it("调 post + 关 modal + resetFields + loadExecutions", async () => {
    const api = createApiMock("/network/executions/template/execute");
    api.endpoint.mockResolvedValue({ data: {} });
    const loadExecutions = vi.fn().mockResolvedValue(undefined);
    const params = { ...baseParams(), loadExecutions };
    const { result } = renderHook(() => useExecutionModals(params), { wrapper: wrap });
    act(() => result.current.setSelectedRowKeys(["d1"]));
    const form = {
      validateFields: vi.fn().mockResolvedValue({
        executionName: "exec1",
        templateId: "t1",
        templateVariables: { host: "1.2.3.4" },
      }),
      resetFields: vi.fn(),
    };
    await act(async () => {
      await result.current.handleExecuteByTemplate(form as any);
    });
    expect(api.endpoint).toHaveBeenCalledWith("/network/executions/template/execute", {
      executionName: "exec1",
      templateId: "t1",
      deviceIds: ["d1"],
      templateVariables: { host: "1.2.3.4" },
    });
    expect(result.current.modalState.executeModalVisible).toBe(false);
    expect(form.resetFields).toHaveBeenCalled();
    expect(result.current.selectedRowKeys).toEqual([]);
    expect(loadExecutions).toHaveBeenCalled();
  });

  it("validate 失败(errorFields 存在) → short-circuit 不调 post", async () => {
    const api = createApiMock("/network/executions/template/execute");
    const form = {
      validateFields: vi.fn().mockRejectedValue({ errorFields: [{ name: "executionName" }] }),
      resetFields: vi.fn(),
    };
    const { result } = renderHook(() => useExecutionModals(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleExecuteByTemplate(form as any);
    });
    expect(api.endpoint).not.toHaveBeenCalled();
  });

  it("validate 无 errorFields(网络错误) → message.error", async () => {
    const api = createApiMock("/network/executions/template/execute");
    api.endpoint.mockRejectedValue(new Error("network"));
    const form = {
      validateFields: vi.fn().mockResolvedValue({ executionName: "x", templateId: "t1" }),
      resetFields: vi.fn(),
    };
    const { result } = renderHook(() => useExecutionModals(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleExecuteByTemplate(form as any);
    });
    expect(api.endpoint).toHaveBeenCalled();
  });
});

describe("useExecutionModals — handleCancelExecution", () => {
  it("调 cancel URL + message.success + loadExecutions", async () => {
    const api = createApiMock("/network/executions/e1/cancel");
    api.endpoint.mockResolvedValue({ data: {} });
    const loadExecutions = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() => useExecutionModals({ ...baseParams(), loadExecutions }), {
      wrapper: wrap,
    });
    await act(async () => {
      await result.current.handleCancelExecution("e1");
    });
    expect(api.endpoint).toHaveBeenCalledWith("/network/executions/e1/cancel", {});
    expect(loadExecutions).toHaveBeenCalled();
  });

  it("cancel 抛错 → message.error fallback", async () => {
    const api = createApiMock("/network/executions/e1/cancel");
    api.endpoint.mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useExecutionModals(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleCancelExecution("e1");
    });
    expect(api.endpoint).toHaveBeenCalled();
  });
});

describe("useExecutionModals — handleViewDetail / handleViewOutput", () => {
  it("handleViewDetail 开 detailDrawer", async () => {
    const { result } = renderHook(() => useExecutionModals(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleViewDetail({ id: "e1" });
    });
    expect(result.current.modalState.detailDrawerVisible).toBe(true);
  });

  it("handleViewOutput 调 Modal.info 带 <pre> output", () => {
    const infoSpy = vi.spyOn(Modal, "info").mockImplementation(() => ({ destroy: vi.fn() }) as any);
    const { result } = renderHook(() => useExecutionModals(baseParams()), { wrapper: wrap });
    act(() => result.current.handleViewOutput("hello world"));
    expect(infoSpy).toHaveBeenCalledTimes(1);
    const config = infoSpy.mock.calls[0][0];
    expect(config.title).toBe("配置输出");
    expect(config.width).toBe(800);
    infoSpy.mockRestore();
  });
});

describe("useExecutionModals — close 系列", () => {
  it("closeDetailDrawer 关 detailDrawer", async () => {
    const { result } = renderHook(() => useExecutionModals(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleViewDetail({ id: "e1" });
    });
    expect(result.current.modalState.detailDrawerVisible).toBe(true);
    act(() => result.current.closeDetailDrawer());
    expect(result.current.modalState.detailDrawerVisible).toBe(false);
  });

  it("closeVariableModal 关 variableModal", () => {
    const { result } = renderHook(() => useExecutionModals(baseParams()), { wrapper: wrap });
    act(() => result.current.handleTemplateChange("t1"));
    expect(result.current.modalState.variableModalVisible).toBe(true);
    act(() => result.current.closeVariableModal());
    expect(result.current.modalState.variableModalVisible).toBe(false);
  });

  it("closeExecuteModal 关 executeModal", async () => {
    const { result } = renderHook(() => useExecutionModals(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.openExecuteModal();
    });
    expect(result.current.modalState.executeModalVisible).toBe(true);
    act(() => result.current.closeExecuteModal());
    expect(result.current.modalState.executeModalVisible).toBe(false);
  });
});
