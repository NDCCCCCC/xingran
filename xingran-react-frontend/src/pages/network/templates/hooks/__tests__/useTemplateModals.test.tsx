/**
 * Phase 88 Batch89 — network/templates/hooks/useTemplateModals 测试(78 stmts, 24.4% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp, Form } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useTemplateModals } from "../useTemplateModals";
import { createApiMock } from "@/test/utils/createApiMock";

function HookWrapper() {
  const { result } = useTemplateModalsTest();
  return (
    <span data-testid="state">
      {JSON.stringify({
        editing: result.editingTemplate,
        editVisible: result.editModalVisible,
        previewVisible: result.previewVisible,
        previewContent: result.previewContent,
      })}
    </span>
  );
}

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

interface UseTemplateModalsTestHook {
  result: ReturnType<typeof useTemplateModals>;
}

// Re-export hook for direct testing via wrapper that passes an App
function useTemplateModalsTest(): UseTemplateModalsTestHook {
  const r = useTemplateModals();
  return { result: r };
}

describe("useTemplateModals", () => {
  it("初始化默认值", () => {
    const { result } = renderHook(() => useTemplateModals(), { wrapper });
    expect(result.current.editModalVisible).toBe(false);
    expect(result.current.previewVisible).toBe(false);
    expect(result.current.variablesModalVisible).toBe(false);
    expect(result.current.editingTemplate).toBeNull();
    expect(result.current.selectedRowKeys).toEqual([]);
  });

  it("openModal 无 record → 新建模式", () => {
    const { result } = renderHook(() => useTemplateModals(), { wrapper });
    act(() => result.current.openModal());
    expect(result.current.editModalVisible).toBe(true);
    expect(result.current.editingTemplate).toBeNull();
  });

  it("openModal 有 record + editForm → 编辑模式", () => {
    const { result } = renderHook(() => useTemplateModals(), { wrapper });
    const record = { id: "t1", name: "模板1", content: "abc" } as any;
    const fakeForm = {
      setFieldsValue: vi.fn(),
      resetFields: vi.fn(),
    } as any;
    act(() => result.current.openModal(record, fakeForm));
    expect(result.current.editingTemplate).toEqual(record);
    expect(result.current.editModalVisible).toBe(true);
  });

  it("closeModal → editModalVisible=false", () => {
    const { result } = renderHook(() => useTemplateModals(), { wrapper });
    act(() => result.current.openModal());
    act(() => result.current.closeModal());
    expect(result.current.editModalVisible).toBe(false);
    expect(result.current.editingTemplate).toBeNull();
  });

  it("handleCreate: 编辑模式 → POST /templates/{id}/update", async () => {
    const api = createApiMock("/network/templates/t1/update");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    const editingTemplate = { id: "t1", name: "old", content: "old" } as any;
    const { result } = renderHook(() => useTemplateModals(), { wrapper });

    // Stub a FormInstance that returns valid form values
    const fakeForm = {
      validateFields: vi.fn().mockResolvedValue({ name: "new", content: "new" }),
      resetFields: vi.fn(),
    } as any;
    await act(async () => {
      await result.current.handleCreate(editingTemplate, fakeForm, vi.fn());
    });
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("handleCreate: 新建模式 → POST /templates", async () => {
    const api = createApiMock("/network/templates");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    const { result } = renderHook(() => useTemplateModals(), { wrapper });
    const fakeForm = {
      validateFields: vi.fn().mockResolvedValue({ name: "new", content: "c" }),
      resetFields: vi.fn(),
    } as any;
    await act(async () => {
      await result.current.handleCreate(null, fakeForm, vi.fn());
    });
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("handleCreate: 校验失败 (errorFields) → 不调 API", async () => {
    const api = createApiMock("/network/templates");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    const { result } = renderHook(() => useTemplateModals(), { wrapper });
    const fakeForm = {
      validateFields: vi
        .fn()
        .mockRejectedValue({ errorFields: [{ name: "name", errors: ["required"] }] }),
      resetFields: vi.fn(),
    } as any;
    await act(async () => {
      await result.current.handleCreate(null, fakeForm, vi.fn());
    });
    expect(api.endpoint).not.toHaveBeenCalled();
  });

  it("handlePreview: 成功 → previewVisible=true + 写入 content", async () => {
    const api = createApiMock("/network/templates/t1/preview");
    api.endpoint.mockResolvedValueOnce({ code: 0, data: { content: "rendered" } } as any);
    const { result } = renderHook(() => useTemplateModals(), { wrapper });
    await act(async () => {
      await result.current.handlePreview("t1");
    });
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("handlePreview: 失败 → 不修改 state", async () => {
    const api = createApiMock("/network/templates/t1/preview");
    api.endpoint.mockRejectedValueOnce(new Error("net"));
    const { result } = renderHook(() => useTemplateModals(), { wrapper });
    await act(async () => {
      await result.current.handlePreview("t1");
    });
    expect(result.current.previewContent).toBe("");
    expect(result.current.previewVisible).toBe(false);
  });

  it("handleGetVariables: 成功 → 打开变量 modal + 写入变量", async () => {
    const api = createApiMock("/network/templates/t1/variables");
    api.endpoint.mockResolvedValueOnce({ code: 0, data: { variables: { a: 1 } } } as any);
    const { result } = renderHook(() => useTemplateModals(), { wrapper });
    await act(async () => {
      await result.current.handleGetVariables("t1");
    });
    expect(api.endpoint).toHaveBeenCalled();
    expect(result.current.templateVariables).toEqual({ a: 1 });
    expect(result.current.variablesModalVisible).toBe(true);
  });

  it("setEditModalVisible / setPreviewVisible 直接写入", () => {
    const { result } = renderHook(() => useTemplateModals(), { wrapper });
    act(() => {
      result.current.setEditModalVisible(true);
      result.current.setPreviewVisible(true);
      result.current.setVariablesModalVisible(true);
      result.current.setSelectedRowKeys(["k1"]);
      result.current.setPreviewContent("preview");
      result.current.setTemplateVariables({ foo: "bar" });
    });
    expect(result.current.editModalVisible).toBe(true);
    expect(result.current.previewVisible).toBe(true);
    expect(result.current.variablesModalVisible).toBe(true);
    expect(result.current.selectedRowKeys).toEqual(["k1"]);
    expect(result.current.previewContent).toBe("preview");
    expect(result.current.templateVariables).toEqual({ foo: "bar" });
  });
});
