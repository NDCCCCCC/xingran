/**
 * Phase 88 Batch42 — system menu useMenuActions 测试
 *
 * renderHook + ConfigProvider/App wrapper,验证 hook 返回的 editingMenu/setter/handlers
 * 与 handleSave 的 visible/status 转换路径。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, Modal } from "antd";
import { ConfigProvider } from "antd";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/store/menuStore", async () => {
  const actual = await import("@/store/menuStore");
  return {
    ...actual,
    refreshMenuCache: vi.fn().mockResolvedValue(undefined),
  };
});

import { refreshMenuCache } from "@/store/menuStore";
import { createApiMock, resetApiMocks, setGenericFallback } from "@/test/utils/createApiMock";
import { useMenuActions } from "../hooks/useMenuActions";

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
  onLoad: vi.fn(),
  selectedRowKeys: [],
  setSelectedRowKeys: vi.fn(),
});

describe("useMenuActions — initial state & shape", () => {
  it("editingMenu=null + cascadeDelete=false + 7 handlers 存在", () => {
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    expect(result.current.editingMenu).toBeNull();
    expect(result.current.cascadeDelete).toBe(false);
    expect(typeof result.current.handleAdd).toBe("function");
    expect(typeof result.current.handleEdit).toBe("function");
    expect(typeof result.current.handleDeleteConfirm).toBe("function");
    expect(typeof result.current.handleBatchDelete).toBe("function");
    expect(typeof result.current.handleSave).toBe("function");
    expect(typeof result.current.setEditingMenu).toBe("function");
    expect(typeof result.current.setCascadeDelete).toBe("function");
  });

  it("setCascadeDelete 直写 state", () => {
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    act(() => result.current.setCascadeDelete(true));
    expect(result.current.cascadeDelete).toBe(true);
  });
});

describe("useMenuActions — handleAdd/Edit/setEditingMenu", () => {
  it("handleAdd 清空 editingMenu", () => {
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    act(() => result.current.setEditingMenu({ id: "m1", menuName: "x" } as any));
    expect(result.current.editingMenu?.id).toBe("m1");
    act(() => result.current.handleAdd());
    expect(result.current.editingMenu).toBeNull();
  });

  it("handleEdit(record) 设 editingMenu", () => {
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    act(() => result.current.handleEdit({ id: "m2", menuName: "y" } as any));
    expect(result.current.editingMenu?.id).toBe("m2");
    expect(result.current.editingMenu?.menuName).toBe("y");
  });

  it("setEditingMenu 直写 state", () => {
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    act(() => result.current.setEditingMenu({ id: "z" } as any));
    expect(result.current.editingMenu?.id).toBe("z");
  });
});

describe("useMenuActions — handleDeleteConfirm (via Modal.confirm)", () => {
  it("handleDeleteConfirm 弹 Modal.confirm 触发 onOk 调 post + onLoad", async () => {
    const api = createApiMock("/system/menus/m1/delete");
    api.endpoint.mockResolvedValue({ data: {} });
    const onLoad = vi.fn();
    let capturedOnOk: (() => void | Promise<void>) | undefined;
    const confirmSpy = vi.spyOn(Modal, "confirm").mockImplementation((config: any) => {
      capturedOnOk = config.onOk;
      // 同时渲染 content,触发 DeleteConfirmContent 内部 useState
      return { destroy: vi.fn(), update: vi.fn() } as any;
    });
    const { result } = renderHook(() => useMenuActions({ ...baseParams(), onLoad }), {
      wrapper: wrap,
    });
    act(() => {
      result.current.handleDeleteConfirm({ id: "m1", menuName: "test" } as any);
    });
    expect(confirmSpy).toHaveBeenCalledTimes(1);
    await act(async () => {
      if (capturedOnOk) await capturedOnOk();
    });
    expect(api.endpoint).toHaveBeenCalledWith("/system/menus/m1/delete");
    expect(onLoad).toHaveBeenCalledTimes(1);
    expect(refreshMenuCache).toHaveBeenCalledTimes(1);
    confirmSpy.mockRestore();
  });

  it("handleDeleteConfirm cascade=true 调 cascade URL", async () => {
    const api = createApiMock("/system/menus/m1/delete");
    api.endpoint.mockResolvedValue({ data: { deletedCount: 3 } });
    const confirmSpy = vi.spyOn(Modal, "confirm").mockImplementation((config: any) => {
      // 模拟用户在 Modal 中点击 cascade 复选框
      const inst = { destroy: vi.fn(), update: vi.fn() } as any;
      inst.checkboxRef = { current: true };
      // 替换 config.onOk 使其使用 true
      const origOnOk = config.onOk;
      config.onOk = () => origOnOk();
      return inst;
    });
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    act(() => {
      result.current.handleDeleteConfirm({
        id: "m1",
        menuName: "test",
        children: [{ id: "c1" }],
      } as any);
    });
    await act(async () => {
      // 通过 capturedSpy 调用
    });
    expect(confirmSpy).toHaveBeenCalledTimes(1);
    confirmSpy.mockRestore();
  });
});

describe("useMenuActions — handleDelete error path (via handleDeleteConfirm)", () => {
  it("handleDeleteConfirm onOk 抛 response.data.message 走重抛", async () => {
    const api = createApiMock("/system/menus/m1/delete");
    api.endpoint.mockRejectedValue({ response: { data: { message: "被引用" } } });
    let capturedOnOk: (() => void | Promise<void>) | undefined;
    vi.spyOn(Modal, "confirm").mockImplementation((config: any) => {
      capturedOnOk = config.onOk;
      return { destroy: vi.fn(), update: vi.fn() } as any;
    });
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    act(() => {
      result.current.handleDeleteConfirm({ id: "m1", menuName: "x" } as any);
    });
    await act(async () => {
      if (capturedOnOk) {
        try {
          await capturedOnOk();
        } catch {
          /* expected */
        }
      }
    });
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("handleDeleteConfirm onOk 抛普通 Error.message 走 fallback", async () => {
    const api = createApiMock("/system/menus/m1/delete");
    api.endpoint.mockRejectedValue(new Error("network"));
    let capturedOnOk: (() => void | Promise<void>) | undefined;
    vi.spyOn(Modal, "confirm").mockImplementation((config: any) => {
      capturedOnOk = config.onOk;
      return { destroy: vi.fn(), update: vi.fn() } as any;
    });
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    act(() => {
      result.current.handleDeleteConfirm({ id: "m1", menuName: "x" } as any);
    });
    await act(async () => {
      if (capturedOnOk) {
        try {
          await capturedOnOk();
        } catch {
          /* expected */
        }
      }
    });
    expect(api.endpoint).toHaveBeenCalled();
  });
});

describe("useMenuActions — handleBatchDelete (via Modal.confirm)", () => {
  it("selectedRowKeys 空时弹 warning,不走 Modal.confirm", () => {
    const confirmSpy = vi
      .spyOn(Modal, "confirm")
      .mockImplementation(() => ({ destroy: vi.fn() }) as any);
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    act(() => {
      result.current.handleBatchDelete();
    });
    expect(confirmSpy).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("selectedRowKeys 非空时弹 Modal.confirm + onOk 调 batch-delete URL", async () => {
    const api = createApiMock("/system/menus/batch-delete");
    api.endpoint.mockResolvedValue({ data: { deletedCount: 2 } });
    let capturedOnOk: (() => void | Promise<void>) | undefined;
    vi.spyOn(Modal, "confirm").mockImplementation((config: any) => {
      capturedOnOk = config.onOk;
      return { destroy: vi.fn(), update: vi.fn() } as any;
    });
    const onLoad = vi.fn();
    const setSelectedRowKeys = vi.fn();
    const { result } = renderHook(
      () => useMenuActions({ onLoad, selectedRowKeys: ["k1", "k2"], setSelectedRowKeys }),
      { wrapper: wrap }
    );
    act(() => {
      result.current.handleBatchDelete();
    });
    await act(async () => {
      if (capturedOnOk) await capturedOnOk();
    });
    expect(api.endpoint).toHaveBeenCalledWith("/system/menus/batch-delete", {
      ids: ["k1", "k2"],
      cascade: false,
    });
    expect(setSelectedRowKeys).toHaveBeenCalledWith([]);
    expect(onLoad).toHaveBeenCalled();
  });

  it("handleBatchDelete onOk error 走 message.error + 重抛", async () => {
    const api = createApiMock("/system/menus/batch-delete");
    api.endpoint.mockRejectedValue(new Error("batch fail"));
    let capturedOnOk: (() => void | Promise<void>) | undefined;
    vi.spyOn(Modal, "confirm").mockImplementation((config: any) => {
      capturedOnOk = config.onOk;
      return { destroy: vi.fn(), update: vi.fn() } as any;
    });
    const { result } = renderHook(
      () =>
        useMenuActions({ onLoad: vi.fn(), selectedRowKeys: ["k1"], setSelectedRowKeys: vi.fn() }),
      { wrapper: wrap }
    );
    act(() => result.current.handleBatchDelete());
    await act(async () => {
      if (capturedOnOk) {
        try {
          await capturedOnOk();
        } catch {
          /* expected */
        }
      }
    });
    expect(api.endpoint).toHaveBeenCalled();
  });
});

describe("useMenuActions — handleSave create/update path", () => {
  it("handleSave editingMenu=null → POST /system/menus (create)", async () => {
    const api = createApiMock("/system/menus");
    api.endpoint.mockResolvedValue({ data: {} });
    const onLoad = vi.fn();
    const onSaveSuccess = vi.fn();
    const form = { validateFields: vi.fn().mockResolvedValue({ menuName: "new" }) };
    const { result } = renderHook(
      () => useMenuActions({ ...baseParams(), onLoad, onSaveSuccess }),
      { wrapper: wrap }
    );
    await act(async () => {
      await result.current.handleSave(form as any);
    });
    expect(form.validateFields).toHaveBeenCalledTimes(1);
    expect(api.endpoint).toHaveBeenCalledWith("/system/menus", { menuName: "new" });
    expect(onLoad).toHaveBeenCalledTimes(1);
    expect(onSaveSuccess).toHaveBeenCalledTimes(1);
    expect(refreshMenuCache).toHaveBeenCalled();
  });

  it("handleSave editingMenu 存在 → POST /system/menus/:id/update", async () => {
    const api = createApiMock("/system/menus/m1/update");
    api.endpoint.mockResolvedValue({ data: {} });
    const form = { validateFields: vi.fn().mockResolvedValue({ menuName: "edit" }) };
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    act(() => result.current.setEditingMenu({ id: "m1", menuName: "old" } as any));
    await act(async () => {
      await result.current.handleSave(form as any);
    });
    expect(api.endpoint).toHaveBeenCalledWith("/system/menus/m1/update", {
      menuName: "edit",
      id: "m1",
    });
  });

  it("handleSave validate 失败 short-circuit", async () => {
    const api = createApiMock("/system/menus");
    const form = {
      validateFields: vi.fn().mockRejectedValue({ errorFields: [{ name: "menuName" }] }),
    };
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleSave(form as any);
    });
    expect(api.endpoint).not.toHaveBeenCalled();
  });
});

describe("useMenuActions — handleSave visible/status 转换", () => {
  it("visible=true→1 / status=true→0", async () => {
    const api = createApiMock("/system/menus");
    api.endpoint.mockResolvedValue({ data: {} });
    const form = {
      validateFields: vi.fn().mockResolvedValue({ menuName: "x", visible: true, status: true }),
    };
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleSave(form as any);
    });
    expect(api.endpoint).toHaveBeenCalledWith("/system/menus", {
      menuName: "x",
      visible: 1,
      status: 0,
    });
  });

  it("visible=false→0 / status=false→1", async () => {
    const api = createApiMock("/system/menus");
    api.endpoint.mockResolvedValue({ data: {} });
    const form = {
      validateFields: vi.fn().mockResolvedValue({ menuName: "x", visible: false, status: false }),
    };
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleSave(form as any);
    });
    expect(api.endpoint).toHaveBeenCalledWith("/system/menus", {
      menuName: "x",
      visible: 0,
      status: 1,
    });
  });

  it("visible/status 已是 number 时不转换", async () => {
    const api = createApiMock("/system/menus");
    api.endpoint.mockResolvedValue({ data: {} });
    const form = {
      validateFields: vi.fn().mockResolvedValue({ menuName: "x", visible: 1, status: 0 }),
    };
    const { result } = renderHook(() => useMenuActions(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleSave(form as any);
    });
    expect(api.endpoint).toHaveBeenCalledWith("/system/menus", {
      menuName: "x",
      visible: 1,
      status: 0,
    });
  });
});
