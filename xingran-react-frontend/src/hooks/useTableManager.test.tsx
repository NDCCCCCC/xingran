/**
 * useTableManager 统一表格管理 Hook 测试
 *
 * 覆盖：初始状态 / loadData 参数组装 / filters 持久化与恢复 /
 * applyFilters+handleSearch / handleReset / handleRefresh /
 * 编辑弹窗动作 / handleTableChange 分页+服务端排序 / externalPagination 委托。
 *
 * 注：antd Form 实例必须挂载 <Form.Item> 字段后 getFieldsValue() 才返回值
 * （rc-field-form 无参 getFieldsValue 只读已注册字段），故用 Harness 组件挂载表单。
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { Form, Input } from "antd";
import { useTableManager, type TableManagerOptions } from "./useTableManager";

const FILTERS_KEY = "xingran_table_state_system_user_filters";

type Manager = ReturnType<typeof useTableManager<Record<string, unknown>>>;

function makeSorter(field: string | undefined, order: "ascend" | "descend" | null) {
  return { field, order, columnKey: field, column: null } as unknown as Parameters<
    Manager["handleTableChange"]
  >[2];
}

/** 挂载 hook + 搜索/编辑两个真实 Form 的宿主组件 */
function createHarness(
  loadFunction: (params: Record<string, unknown>) => Promise<{ list: unknown[]; total: number }>,
  options?: TableManagerOptions<Record<string, unknown>>
) {
  // 属性可变容器规避 react-hooks/globals 渲染期变量重赋值禁令
  const apiRef: { current: Manager | null } = { current: null };
  const Harness = () => {
    const mgr = useTableManager<Record<string, unknown>>(loadFunction, options);
    // eslint-disable-next-line react-hooks/immutability -- test harness capture
    apiRef.current = mgr;
    return (
      <>
        <Form form={mgr.searchForm}>
          <Form.Item name="username" noStyle>
            <Input />
          </Form.Item>
          <Form.Item name="nickname" noStyle>
            <Input />
          </Form.Item>
          <Form.Item name="deptId" noStyle>
            <Input />
          </Form.Item>
        </Form>
        <Form form={mgr.editForm}>
          <Form.Item name="id" noStyle>
            <Input />
          </Form.Item>
          <Form.Item name="name" noStyle>
            <Input />
          </Form.Item>
        </Form>
      </>
    );
  };
  const { rerender, unmount } = render(
    <MemoryRouter initialEntries={["/system/user"]}>
      <Harness />
    </MemoryRouter>
  );
  return {
    api: () => apiRef.current!,
    rerender: () =>
      rerender(
        <MemoryRouter initialEntries={["/system/user"]}>
          <Harness />
        </MemoryRouter>
      ),
    unmount,
  };
}

describe("useTableManager", () => {
  let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
    consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleErrorSpy.mockRestore();
  });

  it("初始状态为默认分页 + 空数据", () => {
    const loadFunction = vi.fn();
    const h = createHarness(loadFunction);
    const mgr = h.api();

    expect(mgr.current).toBe(1);
    expect(mgr.pageSize).toBe(10);
    expect(mgr.data).toEqual([]);
    expect(mgr.total).toBe(0);
    expect(mgr.loading).toBe(false);
    expect(mgr.selectedRowKeys).toEqual([]);
    expect(mgr.editModalVisible).toBe(false);
    expect(mgr.editingItem).toBeNull();
    h.unmount();
  });

  it("loadData 组装 current/pageSize 并 setData/setTotal", async () => {
    const loadFunction = vi.fn().mockResolvedValue({ list: [{ id: "a" }, { id: "b" }], total: 42 });
    const h = createHarness(loadFunction);

    await act(async () => {
      await h.api().loadData();
    });

    expect(loadFunction).toHaveBeenCalledWith({ current: 1, pageSize: 10 });
    expect(h.api().data).toEqual([{ id: "a" }, { id: "b" }]);
    expect(h.api().total).toBe(42);
    expect(h.api().loading).toBe(false);
    h.unmount();
  });

  it("loadData 对 list/total 缺失回退空值,失败也复位 loading", async () => {
    const loadFunction = vi.fn().mockResolvedValue({});
    const h = createHarness(loadFunction);

    await act(async () => {
      await h.api().loadData();
    });
    expect(h.api().data).toEqual([]);
    expect(h.api().total).toBe(0);
    h.unmount();

    const failing = vi.fn().mockRejectedValue(new Error("boom"));
    const h2 = createHarness(failing);
    await act(async () => {
      await h2
        .api()
        .loadData()
        .catch(() => {});
    });
    expect(h2.api().loading).toBe(false);
    h2.unmount();
  });

  it("externalPagination 优先并接收 setTotal", async () => {
    const loadFunction = vi.fn().mockResolvedValue({ list: [], total: 7 });
    const externalPagination = {
      current: 3,
      pageSize: 20,
      setCurrent: vi.fn(),
      setPageSize: vi.fn(),
      setTotal: vi.fn(),
    };
    const h = createHarness(loadFunction, {
      externalPagination,
      pageSize: 30,
    });
    const mgr = h.api();

    expect(mgr.current).toBe(3);
    expect(mgr.pageSize).toBe(20);

    await act(async () => {
      await mgr.loadData();
    });
    expect(loadFunction).toHaveBeenCalledWith({ current: 3, pageSize: 20 });
    expect(externalPagination.setTotal).toHaveBeenCalledWith(7);
    h.unmount();
  });

  it("handleSearch 读取表单值过滤空串并持久化 filters,回到第 1 页", async () => {
    const loadFunction = vi.fn().mockResolvedValue({ list: [], total: 0 });
    const h = createHarness(loadFunction);

    act(() => {
      h.api().searchForm.setFieldsValue({ username: "alice", nickname: "", deptId: "d1" });
    });

    await act(async () => {
      h.api().handleSearch();
    });

    const call = loadFunction.mock.calls.at(-1)?.[0] as Record<string, unknown>;
    expect(call.username).toBe("alice");
    expect(call.deptId).toBe("d1");
    expect(call).not.toHaveProperty("nickname"); // 空串被 filterEmpty 剔除
    expect(call.current).toBe(1);

    const persisted = sessionStorage.getItem(FILTERS_KEY);
    expect(persisted).toBeTruthy();
    expect(JSON.parse(persisted!)).toEqual({ username: "alice", deptId: "d1" });
    h.unmount();
  });

  it("applyFilters 合并 extra 参数并覆盖表单同名项", async () => {
    const loadFunction = vi.fn().mockResolvedValue({ list: [], total: 0 });
    const h = createHarness(loadFunction);

    act(() => {
      h.api().searchForm.setFieldsValue({ username: "bob" });
    });

    await act(async () => {
      h.api().applyFilters({ deptId: "extra-dept", username: "override" });
    });

    const call = loadFunction.mock.calls.at(-1)?.[0] as Record<string, unknown>;
    expect(call.username).toBe("override");
    expect(call.deptId).toBe("extra-dept");
    h.unmount();
  });

  it("mount 时从 sessionStorage 恢复 filters 并回填表单", () => {
    sessionStorage.setItem(FILTERS_KEY, JSON.stringify({ username: "restored" }));
    const loadFunction = vi.fn();
    const h = createHarness(loadFunction);

    expect(h.api().searchForm.getFieldsValue()).toMatchObject({ username: "restored" });
    h.unmount();
  });

  it("损坏或非对象的 filters 存储被安全忽略", () => {
    sessionStorage.setItem(FILTERS_KEY, "not-valid-json{{{");
    const loadFunction = vi.fn();
    const h = createHarness(loadFunction);
    expect(h.api().searchForm.getFieldsValue()).toEqual({});
    h.unmount();

    sessionStorage.setItem(FILTERS_KEY, JSON.stringify([1, 2, 3]));
    const h2 = createHarness(loadFunction);
    expect(h2.api().searchForm.getFieldsValue()).toEqual({});
    h2.unmount();
  });

  it("handleReset 清空持久化 filters 与排序,回第 1 页", async () => {
    const loadFunction = vi.fn().mockResolvedValue({ list: [], total: 0 });
    const sorterMetas = [{ field: "username" }];
    const h = createHarness(loadFunction, { sorterMetas });

    act(() => {
      h.api().searchForm.setFieldsValue({ username: "x" });
    });
    await act(async () => {
      h.api().handleSearch();
    });
    expect(sessionStorage.getItem(FILTERS_KEY)).toBeTruthy();

    // 先设置排序
    act(() => {
      h.api().handleTableChange({ current: 1, pageSize: 10 }, {}, makeSorter("username", "ascend"));
    });

    await act(async () => {
      h.api().handleReset();
    });

    expect(sessionStorage.getItem(FILTERS_KEY)).toBeNull();
    expect(h.api().current).toBe(1);
    const call = loadFunction.mock.calls.at(-1)?.[0] as Record<string, unknown>;
    expect(call).not.toHaveProperty("orderByColumn");
    expect(call).not.toHaveProperty("username");
    h.unmount();
  });

  it("handleRefresh 重新加载并触发 onSuccess", async () => {
    const loadFunction = vi.fn().mockResolvedValue({ list: [], total: 0 });
    const onSuccess = vi.fn();
    const h = createHarness(loadFunction, { onSuccess });

    await act(async () => {
      h.api().handleRefresh();
    });
    expect(loadFunction).toHaveBeenCalledTimes(1);
    expect(onSuccess).toHaveBeenCalledTimes(1);
    h.unmount();
  });

  it("编辑弹窗动作:handleAdd/handleEdit/handleModalClose", () => {
    const loadFunction = vi.fn();
    const h = createHarness(loadFunction);

    act(() => h.api().handleAdd());
    expect(h.api().editModalVisible).toBe(true);
    expect(h.api().editingItem).toBeNull();

    act(() => h.api().handleModalClose());
    expect(h.api().editModalVisible).toBe(false);
    expect(h.api().editingItem).toBeNull();

    act(() => h.api().handleEdit({ id: "1", name: "n" }));
    expect(h.api().editModalVisible).toBe(true);
    expect(h.api().editingItem).toEqual({ id: "1", name: "n" });
    expect(h.api().editForm.getFieldsValue()).toEqual({ id: "1", name: "n" });
    h.unmount();
  });

  it("handleTableChange 分页变化触发 setCurrent/setPageSize 并带旧 filters 加载", async () => {
    const loadFunction = vi.fn().mockResolvedValue({ list: [], total: 0 });
    const h = createHarness(loadFunction);

    act(() => {
      h.api().searchForm.setFieldsValue({ username: "keep-me" });
    });
    await act(async () => {
      h.api().handleSearch();
    });

    await act(async () => {
      h.api().handleTableChange({ current: 4, pageSize: 50 }, {}, makeSorter(undefined, null));
    });

    expect(h.api().current).toBe(4);
    expect(h.api().pageSize).toBe(50);
    const call = loadFunction.mock.calls.at(-1)?.[0] as Record<string, unknown>;
    expect(call.current).toBe(4);
    expect(call.pageSize).toBe(50);
    expect(call.username).toBe("keep-me"); // 翻页不丢搜索条件
    h.unmount();
  });

  it("服务端排序:白名单列 ascend/descend/清空 + getColumnSortOrder", async () => {
    const loadFunction = vi.fn().mockResolvedValue({ list: [], total: 0 });
    const sorterMetas = [{ field: "username" }, { field: "createdAt" }];
    const h = createHarness(loadFunction, { sorterMetas });

    await act(async () => {
      h.api().handleTableChange({ current: 1, pageSize: 10 }, {}, makeSorter("username", "ascend"));
    });
    expect(h.api().orderByColumn).toBe("username");
    expect(h.api().isAsc).toBe(true);
    expect(h.api().getColumnSortOrder("username")).toBe("ascend");
    expect(h.api().getColumnSortOrder("createdAt")).toBeUndefined();
    let call = loadFunction.mock.calls.at(-1)?.[0] as Record<string, unknown>;
    expect(call.orderByColumn).toBe("username");
    expect(call.isAsc).toBe(true);

    await act(async () => {
      h.api().handleTableChange(
        { current: 1, pageSize: 10 },
        {},
        makeSorter("username", "descend")
      );
    });
    expect(h.api().orderByColumn).toBe("username");
    expect(h.api().isAsc).toBe(false);

    // 清空排序
    await act(async () => {
      h.api().handleTableChange({ current: 1, pageSize: 10 }, {}, makeSorter("username", null));
    });
    expect(h.api().orderByColumn).toBeUndefined();
    call = loadFunction.mock.calls.at(-1)?.[0] as Record<string, unknown>;
    expect(call).not.toHaveProperty("orderByColumn");

    // 非白名单列被忽略
    await act(async () => {
      h.api().handleTableChange(
        { current: 1, pageSize: 10 },
        {},
        makeSorter("not-registered", "ascend")
      );
    });
    expect(h.api().orderByColumn).toBeUndefined();
    h.unmount();
  });

  it("loadData params override 优先级最高", async () => {
    const loadFunction = vi.fn().mockResolvedValue({ list: [], total: 0 });
    const h = createHarness(loadFunction);

    await act(async () => {
      await h.api().loadData({ current: 9, pageSize: 99, custom: "x" });
    });
    const call = loadFunction.mock.calls.at(-1)?.[0] as Record<string, unknown>;
    expect(call.current).toBe(9);
    expect(call.pageSize).toBe(99);
    expect(call.custom).toBe("x");
    h.unmount();
  });

  it("setters 直接暴露(setData/setTotal/setLoading/setSelectedRowKeys/resetSelection)", () => {
    const loadFunction = vi.fn();
    const h = createHarness(loadFunction);

    act(() => h.api().setData([{ id: "z" }]));
    act(() => h.api().setTotal(5));
    act(() => h.api().setSelectedRowKeys(["1", "2"]));
    expect(h.api().data).toEqual([{ id: "z" }]);
    expect(h.api().total).toBe(5);
    expect(h.api().selectedRowKeys).toEqual(["1", "2"]);

    act(() => h.api().resetSelection());
    expect(h.api().selectedRowKeys).toEqual([]);
    h.unmount();
  });
});
