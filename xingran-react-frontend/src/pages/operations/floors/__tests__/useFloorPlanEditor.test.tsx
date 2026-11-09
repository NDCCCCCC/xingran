/**
 * Phase 88 Batch90 — operations/floors/useFloorPlanEditor 测试(81 stmts, 16.0% → 高)
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  wallApi: {
    list: vi.fn(),
    save: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    batch: vi.fn(),
  },
  doorApi: {
    list: vi.fn(),
    save: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    batch: vi.fn(),
  },
  workstationApi: {
    list: vi.fn(),
    getFloorWorkstationsAll: vi.fn(() => Promise.resolve([])),
    updatePositions: vi.fn(),
  },
  floorPlanTextApi: {
    list: vi.fn(),
    save: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    batch: vi.fn(),
  },
}));

import { useFloorPlanEditor } from "../useFloorPlanEditor";

function wrapper({ children }: { children: React.ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("useFloorPlanEditor", () => {
  it("初始化默认值", () => {
    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    expect(result.current.floorPlanData).toBeNull();
    expect(result.current.floorPlanLoading).toBe(false);
    expect(result.current.isEditMode).toBe(false);
  });

  it("setEditMode → 切换状态", () => {
    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    act(() => result.current.setEditMode(true));
    expect(result.current.isEditMode).toBe(true);
  });

  it("loadFloorPlanData: 有 floorId → 加载", async () => {
    const { wallApi, doorApi, workstationApi, floorPlanTextApi } = await import("@/lib/opsApi");
    vi.mocked(wallApi.list).mockResolvedValueOnce({
      data: { list: [{ id: "w1", points: "[]" }] },
    } as unknown);
    vi.mocked(doorApi.list).mockResolvedValueOnce({ data: { list: [] } } as unknown);
    // V130R-09 D-03-6: 楼层全集改走 getFloorWorkstationsAll(直接返回 WorkstationOps[])
    vi.mocked(workstationApi.getFloorWorkstationsAll).mockResolvedValueOnce([]);
    vi.mocked(floorPlanTextApi.list).mockResolvedValueOnce({ data: { list: [] } } as unknown);

    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    await act(async () => {
      await result.current.loadFloorPlanData({ floorId: "f1" });
    });
    expect(wallApi.list).toHaveBeenCalled();
    expect(result.current.floorPlanData).toBeDefined();
  });

  it("loadFloorPlanData: 失败 → catch 路径", async () => {
    const { wallApi } = await import("@/lib/opsApi");
    vi.mocked(wallApi.list).mockRejectedValueOnce(new Error("net"));

    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    await act(async () => {
      await result.current.loadFloorPlanData({ floorId: "f1" });
    });
    expect(result.current.floorPlanLoading).toBe(false);
  });

  it("resetFloorPlan → 清空数据", () => {
    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    act(() => result.current.setEditMode(true));
    act(() => result.current.resetFloorPlan());
    expect(result.current.isEditMode).toBe(false);
    expect(result.current.floorPlanData).toBeNull();
  });

  it("saveFloorPlan: 无 data → 不发请求", async () => {
    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    await act(async () => {
      await result.current.saveFloorPlan();
    });
    expect(result.current.floorPlanLoading).toBe(false);
  });
});

/**
 * 回归测试：一次保存只弹一次「保存成功」toast。
 *
 * 缺陷现象：平面图编辑点一次「保存」弹出两个「保存成功」——
 * 一处来自 CADFloorPlanEditor.handleSave（未经 await，保存失败也会误报成功），
 * 一处来自 useFloorPlanEditor.saveFloorPlan（await 真实保存成功后）。
 * 修复后，成功提示只允许由 useFloorPlanEditor 在真实保存完成后发出。
 */
describe("saveFloorPlan 的保存提示（一次保存只弹一次 toast）", () => {
  const messageMock = {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
    loading: vi.fn(),
  };

  const floor = { id: "f1", name: "一层", floorNo: 1 };

  let useAppSpy: ReturnType<typeof vi.spyOn> | undefined;

  beforeEach(() => {
    messageMock.success.mockClear();
    messageMock.error.mockClear();
    messageMock.warning.mockClear();
    messageMock.info.mockClear();
    messageMock.loading.mockClear();
    useAppSpy = vi.spyOn(AntdApp, "useApp").mockReturnValue({
      message: messageMock,
    } as unknown as ReturnType<typeof AntdApp.useApp>);
  });

  afterEach(() => {
    useAppSpy?.mockRestore();
  });

  it("保存成功 → 恰好弹一次「保存成功」，且不弹「保存失败」", async () => {
    const { wallApi, doorApi, workstationApi, floorPlanTextApi } = await import("@/lib/opsApi");
    vi.mocked(wallApi.list).mockResolvedValue({ data: { list: [] } } as unknown);
    vi.mocked(doorApi.list).mockResolvedValue({ data: { list: [] } } as unknown);
    vi.mocked(workstationApi.getFloorWorkstationsAll).mockResolvedValue([]);
    vi.mocked(floorPlanTextApi.list).mockResolvedValue({ data: { list: [] } } as unknown);
    vi.mocked(workstationApi.updatePositions).mockResolvedValue(undefined as never);

    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    await act(async () => {
      await result.current.saveFloorPlan(
        { walls: [], doors: [], texts: [], workstations: [] } as never,
        floor as never
      );
    });

    const successCalls = messageMock.success.mock.calls.filter((call) => call[0] === "保存成功");
    expect(successCalls).toHaveLength(1);
    expect(messageMock.error).not.toHaveBeenCalledWith("保存失败");
  });

  it("保存失败 → 弹「保存失败」，且不再误报「保存成功」", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.updatePositions).mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    await act(async () => {
      await result.current.saveFloorPlan(
        { walls: [], doors: [], texts: [], workstations: [{ id: "ws1", x: 1, y: 2 }] } as never,
        floor as never
      );
    });

    const errorCalls = messageMock.error.mock.calls.filter((call) => call[0] === "保存失败");
    expect(errorCalls).toHaveLength(1);
    expect(messageMock.success).not.toHaveBeenCalledWith("保存成功");
  });
});

/**
 * Phase 149 / SCOPE-03：保存侧增量 diff（防非超管整体覆写误删越界几何）。
 *
 * 场景：非超管本次加载只拿到「可见基线 3 面墙」（后端已按可见集裁剪），
 * 但编辑态 data 里混入 2 面越界墙（模拟后端未返回 / 编辑器残留注入）。
 * 断言：① 越界 ID 不出现在任何 create/update/delete 请求；② 可见元素
 * 的 update/create 仍全部提交；③ 保存前后几何行数断言（越界元素不因
 * 整体覆写被误删）；④ Promise.allSettled 失败提示语义不变。
 */
describe("Phase 149 保存侧可见基线增量 diff", () => {
  const messageMock = {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
    loading: vi.fn(),
  };

  const floor = { id: "f1", name: "一层", floorNo: 1 };

  let useAppSpy: ReturnType<typeof vi.spyOn> | undefined;

  beforeEach(async () => {
    vi.clearAllMocks();
    messageMock.success.mockClear();
    messageMock.error.mockClear();
    messageMock.warning.mockClear();
    messageMock.info.mockClear();
    messageMock.loading.mockClear();
    useAppSpy = vi.spyOn(AntdApp, "useApp").mockReturnValue({
      message: messageMock,
    } as unknown as ReturnType<typeof AntdApp.useApp>);

    const { wallApi, doorApi, workstationApi, floorPlanTextApi } = await import("@/lib/opsApi");
    // 可见基线：3 面墙（后端按可见集裁剪后返回）。
    vi.mocked(wallApi.list).mockResolvedValue({
      data: {
        list: [
          { id: "visible-w1", points: "[]", type: "straight" },
          { id: "visible-w2", points: "[]", type: "straight" },
          { id: "visible-w3", points: "[]", type: "straight" },
        ],
      },
    } as unknown);
    vi.mocked(doorApi.list).mockResolvedValue({ data: { list: [] } } as unknown);
    vi.mocked(workstationApi.getFloorWorkstationsAll).mockResolvedValue([]);
    vi.mocked(workstationApi.updatePositions).mockResolvedValue(undefined as never);
    vi.mocked(floorPlanTextApi.list).mockResolvedValue({ data: { list: [] } } as unknown);
    vi.mocked(wallApi.create).mockResolvedValue(undefined as never);
    vi.mocked(wallApi.update).mockResolvedValue(undefined as never);
    vi.mocked(wallApi.delete).mockResolvedValue(undefined as never);
    vi.mocked(wallApi.batch).mockResolvedValue(undefined as never);
    vi.mocked(doorApi.create).mockResolvedValue(undefined as never);
    vi.mocked(doorApi.update).mockResolvedValue(undefined as never);
    vi.mocked(doorApi.delete).mockResolvedValue(undefined as never);
    vi.mocked(doorApi.batch).mockResolvedValue(undefined as never);
    vi.mocked(floorPlanTextApi.create).mockResolvedValue(undefined as never);
    vi.mocked(floorPlanTextApi.update).mockResolvedValue(undefined as never);
    vi.mocked(floorPlanTextApi.delete).mockResolvedValue(undefined as never);
    vi.mocked(floorPlanTextApi.batch).mockResolvedValue(undefined as never);
  });

  afterEach(() => {
    useAppSpy?.mockRestore();
  });

  const makeWall = (id: string) =>
    ({
      id,
      floorId: "f1",
      type: "straight",
      points: [],
      thickness: 10,
      height: 3,
      color: "#fff",
    }) as never;

  it("越界几何 ID 不进任何请求集；可见元素仍全部 update；保存前后几何行数不变", async () => {
    const { wallApi } = await import("@/lib/opsApi");
    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });

    // 建立可见基线 3 面墙。
    await act(async () => {
      await result.current.loadFloorPlanData("f1");
    });
    const before = result.current.floorPlanData?.walls?.length ?? 0;
    expect(before).toBe(3);

    // 编辑态：3 面可见墙（其中 1 面被移除）+ 2 面越界墙 + 1 面新建占位墙。
    const editedWalls = [
      makeWall("visible-w1"),
      makeWall("visible-w2"),
      makeWall("out-of-scope-w1"),
      makeWall("out-of-scope-w2"),
      makeWall("wall_new-1"),
    ];

    await act(async () => {
      await result.current.saveFloorPlan(
        { walls: editedWalls, doors: [], texts: [], workstations: [] } as never,
        floor as never
      );
    });

    // ① 越界 ID 不出现在任何 create / update 请求体或 delete 集。
    const allCreateBodies = vi.mocked(wallApi.create).mock.calls.map((c) => JSON.stringify(c));
    const allUpdateBodies = vi.mocked(wallApi.update).mock.calls.map((c) => JSON.stringify(c));
    const deleteIds = vi.mocked(wallApi.batch).mock.calls.flatMap((c) => (c[1] as string[]) ?? []);
    const allRequests = [...allCreateBodies, ...allUpdateBodies, ...deleteIds].join("|");
    expect(allRequests).not.toContain("out-of-scope-w1");
    expect(allRequests).not.toContain("out-of-scope-w2");

    // ② 可见元素的 update 仍全部提交（visible-w1 / visible-w2 仍在编辑态）。
    const updatedIds = vi.mocked(wallApi.update).mock.calls.map((c) => c[0]);
    expect(updatedIds).toContain("visible-w1");
    expect(updatedIds).toContain("visible-w2");
    // 新建占位墙正常 create。
    expect(vi.mocked(wallApi.create)).toHaveBeenCalledTimes(1);

    // 删除集 = 基线可见 ID − 当前编辑态 ID = visible-w3（只删可见、绝不删越界）。
    expect(deleteIds).toEqual(expect.arrayContaining(["visible-w3"]));
    expect(deleteIds).not.toContain("out-of-scope-w1");
    expect(deleteIds).not.toContain("out-of-scope-w2");

    // ③ 保存后重新加载：几何行数回到可见基线 3（越界 2 面从未入库、也从未被删）。
    const after = result.current.floorPlanData?.walls?.length ?? 0;
    expect(after).toBe(3);
    // ④ 保存成功提示语义不变（一次）。
    expect(messageMock.success).toHaveBeenCalledWith("保存成功");
  });

  it("越界墙单独编辑（整体覆写）不触发任何越界写请求", async () => {
    const { wallApi } = await import("@/lib/opsApi");
    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    await act(async () => {
      await result.current.loadFloorPlanData("f1");
    });

    // 编辑态只含越界 ID（模拟「看不到但仍被整体覆写」）。
    await act(async () => {
      await result.current.saveFloorPlan(
        {
          walls: [makeWall("out-of-scope-w1"), makeWall("out-of-scope-w2")],
          doors: [],
          texts: [],
          workstations: [],
        } as never,
        floor as never
      );
    });

    const deleteIds = vi.mocked(wallApi.batch).mock.calls.flatMap((c) => (c[1] as string[]) ?? []);
    // 可见基线里没有任何墙留在编辑态 → 删除集应是全部可见基线（visible-w1/2/3），
    // 但绝不含越界 ID；越界 ID 既不 update 也不 create。
    expect(deleteIds).toEqual(expect.arrayContaining(["visible-w1", "visible-w2", "visible-w3"]));
    expect(deleteIds).not.toContain("out-of-scope-w1");
    expect(deleteIds).not.toContain("out-of-scope-w2");
    expect(vi.mocked(wallApi.update)).not.toHaveBeenCalled();
    expect(vi.mocked(wallApi.create)).not.toHaveBeenCalled();
    const allBodies = vi
      .mocked(wallApi.update)
      .mock.calls.map((c) => JSON.stringify(c))
      .join("|");
    expect(allBodies).not.toContain("out-of-scope");
  });
});
