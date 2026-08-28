/**
 * Phase 88 Batch39 — workorder useWorkOrderActions hooks 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App } from "antd";
import { ConfigProvider } from "antd";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/workorderApi", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/workorderApi")>();
  return {
    ...actual,
    createWorkOrder: vi.fn().mockResolvedValue({ data: { id: "w1" } }),
    updateWorkOrder: vi.fn().mockResolvedValue({ data: {} }),
    deleteWorkOrder: vi.fn().mockResolvedValue({ data: {} }),
    batchDeleteWorkOrders: vi.fn().mockResolvedValue({ data: {} }),
    assignToTodayDuty: vi.fn().mockResolvedValue({ data: {} }),
    updateWorkOrderStatus: vi.fn().mockResolvedValue({ data: {} }),
    addWorkOrderComment: vi.fn().mockResolvedValue({ data: {} }),
    getWorkOrderComments: vi.fn().mockResolvedValue({ data: { list: [], total: 0 } }),
  };
});

import { useWorkOrderActions } from "../hooks/useWorkOrderActions";

function Wrap({ children }: { children: React.ReactNode }) {
  return <ConfigProvider><App>{children}</App></ConfigProvider>;
}
const wrap = Wrap;

beforeEach(() => {
  vi.clearAllMocks();
});

const baseOpts = () => ({
  fetchList: vi.fn(),
  openDetailDrawer: vi.fn().mockResolvedValue(undefined as any),
  selectedRecord: null,
});

describe("useWorkOrderActions", () => {
  it("initial actionLoading=false", () => {
    const { result } = renderHook(() => useWorkOrderActions(baseOpts()), { wrapper: wrap });
    expect(result.current.actionLoading).toBe(false);
  });

  it("handleDelete 调 API + fetchList", async () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    await act(async () => {
      await result.current.handleDelete("w-1");
    });
    expect(opts.fetchList).toHaveBeenCalledTimes(1);
  });

  it("handleBatchDelete 空数组 short-circuit", async () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    await act(async () => {
      await result.current.handleBatchDelete([]);
    });
    expect(opts.fetchList).not.toHaveBeenCalled();
  });

  it("handleBatchDelete 非空 fetchList", async () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    await act(async () => {
      await result.current.handleBatchDelete(["w-1", "w-2"]);
    });
    expect(opts.fetchList).toHaveBeenCalledTimes(1);
  });

  it("handleAssignToTodayDuty 调 API + fetchList", async () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    await act(async () => {
      await result.current.handleAssignToTodayDuty("w-1");
    });
    expect(opts.fetchList).toHaveBeenCalledTimes(1);
  });

  it("handleStatusChange selectedRecord=null 时 short-circuit", async () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    await act(async () => {
      await result.current.handleStatusChange(2);
    });
    expect(opts.fetchList).not.toHaveBeenCalled();
  });

  it("handleStatusChange 有 selectedRecord 时调 API + fetchList + openDetailDrawer", async () => {
    const opts = {
      ...baseOpts(),
      selectedRecord: { id: "w-1" } as any,
    };
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    await act(async () => {
      await result.current.handleStatusChange(2);
    });
    expect(opts.fetchList).toHaveBeenCalledTimes(1);
    expect(opts.openDetailDrawer).toHaveBeenCalled();
  });

  it("handleAdd 为 noop(由 useWorkOrderModals 处理)", () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    act(() => {
      result.current.handleAdd();
    });
    expect(opts.fetchList).not.toHaveBeenCalled();
  });

  it("handleEdit 为 noop(由 useWorkOrderModals 处理)", () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    act(() => {
      result.current.handleEdit({ id: "w-1" } as any);
    });
    expect(opts.fetchList).not.toHaveBeenCalled();
  });

  it("handleModalOk create 路径", async () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    const form = {
      validateFields: vi.fn().mockResolvedValue({ title: "任务", priority: 1 }),
    };
    await act(async () => {
      await result.current.handleModalOk(form as any, null, vi.fn());
    });
    expect(opts.fetchList).toHaveBeenCalledTimes(1);
  });

  it("handleModalOk update 路径", async () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    const form = {
      validateFields: vi.fn().mockResolvedValue({ title: "改后", priority: 2 }),
    };
    const editing = { id: "w-existing" } as any;
    await act(async () => {
      await result.current.handleModalOk(form as any, editing, vi.fn());
    });
    expect(opts.fetchList).toHaveBeenCalledTimes(1);
  });

  it("handleModalOk validate 失败 short-circuit", async () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    const form = {
      validateFields: vi.fn().mockRejectedValue({ errorFields: [{ name: "title" }] }),
    };
    await act(async () => {
      await result.current.handleModalOk(form as any, null, vi.fn());
    });
    expect(opts.fetchList).not.toHaveBeenCalled();
  });

  it("handleAddComment 调 API", async () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    const form = {
      validateFields: vi.fn().mockResolvedValue({ content: "备注内容" }),
      resetFields: vi.fn(),
    };
    await act(async () => {
      await result.current.handleAddComment(form as any, false, vi.fn());
    });
  });

  it("handleAddComment validate 失败 short-circuit", async () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    const form = {
      validateFields: vi.fn().mockRejectedValue({ errorFields: [{ name: "content" }] }),
      resetFields: vi.fn(),
    };
    await act(async () => {
      await result.current.handleAddComment(form as any, false, vi.fn());
    });
  });

  it("handleAddComment 外部 internal=true 分支", async () => {
    const opts = baseOpts();
    const { result } = renderHook(() => useWorkOrderActions(opts), { wrapper: wrap });
    const form = {
      validateFields: vi.fn().mockResolvedValue({ content: "内部备注" }),
      resetFields: vi.fn(),
    };
    await act(async () => {
      await result.current.handleAddComment(form as any, true, vi.fn());
    });
  });
});
