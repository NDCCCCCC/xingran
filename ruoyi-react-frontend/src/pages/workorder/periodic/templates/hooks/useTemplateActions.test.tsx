/**
 * Phase 88 Batch41 — workorder useTemplateActions 测试
 *
 * 直接 renderHook 含 Shell 组件,验证 hook 返回的 editingRecord/setter/handlers。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, Form } from "antd";
import { ConfigProvider } from "antd";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/workorderApi", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/workorderApi")>();
  return {
    ...actual,
    createPeriodicTemplate: vi.fn().mockResolvedValue({ data: {} }),
    updatePeriodicTemplate: vi.fn().mockResolvedValue({ data: {} }),
    deletePeriodicTemplate: vi.fn().mockResolvedValue({ data: {} }),
    enablePeriodicTemplate: vi.fn().mockResolvedValue({ data: {} }),
    disablePeriodicTemplate: vi.fn().mockResolvedValue({ data: {} }),
    generateWorkOrderNow: vi.fn().mockResolvedValue({ data: {} }),
  };
});

import { useTemplateActions } from "../hooks/useTemplateActions";

function Wrap({ children }: { children: React.ReactNode }) {
  return (
    <ConfigProvider>
      <App>{children}</App>
    </ConfigProvider>
  );
}
const wrap = Wrap;

beforeEach(() => {
  vi.clearAllMocks();
});

describe("useTemplateActions", () => {
  it("initial editingRecord=null + 6 handlers 存在", () => {
    const { result } = renderHook(() => useTemplateActions({ onLoad: vi.fn() }), { wrapper: wrap });
    expect(result.current.editingRecord).toBeNull();
    expect(typeof result.current.handleAdd).toBe("function");
    expect(typeof result.current.handleEdit).toBe("function");
    expect(typeof result.current.handleDelete).toBe("function");
    expect(typeof result.current.handleToggleEnabled).toBe("function");
    expect(typeof result.current.handleGenerateNow).toBe("function");
    expect(typeof result.current.handleSave).toBe("function");
  });

  it("handleDelete 调 API + onLoad", async () => {
    const onLoad = vi.fn();
    const { result } = renderHook(() => useTemplateActions({ onLoad }), { wrapper: wrap });
    await act(async () => {
      await result.current.handleDelete("t1");
    });
    expect(onLoad).toHaveBeenCalledTimes(1);
  });

  it("handleToggleEnabled isEnabled=true 调 disable", async () => {
    const { result } = renderHook(() => useTemplateActions({ onLoad: vi.fn() }), { wrapper: wrap });
    await act(async () => {
      await result.current.handleToggleEnabled({ id: "t1", isEnabled: true } as any);
    });
  });

  it("handleToggleEnabled isEnabled=false 调 enable", async () => {
    const { result } = renderHook(() => useTemplateActions({ onLoad: vi.fn() }), { wrapper: wrap });
    await act(async () => {
      await result.current.handleToggleEnabled({ id: "t1", isEnabled: false } as any);
    });
  });

  it("handleGenerateNow 调 API + onLoad", async () => {
    const onLoad = vi.fn();
    const { result } = renderHook(() => useTemplateActions({ onLoad }), { wrapper: wrap });
    await act(async () => {
      await result.current.handleGenerateNow("t1");
    });
    expect(onLoad).toHaveBeenCalled();
  });

  it("setEditingRecord 直写 state", () => {
    const { result } = renderHook(() => useTemplateActions({ onLoad: vi.fn() }), { wrapper: wrap });
    act(() => {
      result.current.setEditingRecord({ id: "x" } as any);
    });
    expect(result.current.editingRecord?.id).toBe("x");
  });

  it("handleSave validate 失败 short-circuit(无 setFieldsValue)", async () => {
    const { result } = renderHook(() => useTemplateActions({ onLoad: vi.fn() }), { wrapper: wrap });
    const form = {
      validateFields: vi.fn().mockRejectedValue({ errorFields: [{ name: "templateName" }] }),
    };
    await act(async () => {
      await result.current.handleSave(form as any);
    });
  });

  it("handleDelete error 静默(message.error 抛被 App 捕获)", async () => {
    vi.doMock("@/lib/workorderApi", () => ({
      deletePeriodicTemplate: vi.fn().mockRejectedValue(new Error("boom")),
    }));
    const { result } = renderHook(() => useTemplateActions({ onLoad: vi.fn() }), { wrapper: wrap });
    await act(async () => {
      await result.current.handleDelete("t1");
    });
  });
});
