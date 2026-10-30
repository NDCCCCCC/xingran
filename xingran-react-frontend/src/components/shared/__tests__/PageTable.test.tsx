/**
 * Phase 66 — PageTable 单测（UI-SPEC §1 行为契约）
 *
 * T1 范围：三必填最小骨架 + 三态（loading/空/错误）+ 分页 showTotal 口径 + rest 透传。
 * 组件内部实例化 useTableManager（依赖 useLocation）→ 必须 MemoryRouter 包裹。
 * antd v6：Table 蒙层类 `.ant-spin`；空态文案 `暂无数据`；分页总文案由 showTotal 注入。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntdApp, Form, Modal } from "antd";
import type { ReactElement, ReactNode } from "react";

import { createSorterMeta } from "@/utils/tableHelpers";
import { createDeleteAction } from "../ActionButtons";

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
  handleSuccess: vi.fn(),
  isFormValidationError: (e: unknown) => Boolean(e && typeof e === "object" && "errorFields" in e),
}));

import { handleApiError } from "@/utils/errorHandler";
import { PageTable } from "../PageTable";

function Wrap({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter initialEntries={["/test"]}>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

const columns = [{ title: "名称", dataIndex: "name", key: "name" }];

const makeRows = (n: number, prefix = "行") =>
  Array.from({ length: n }, (_, i) => ({ id: String(i + 1), name: `${prefix}${i + 1}` }));

beforeEach(() => {
  vi.clearAllMocks();
  sessionStorage.clear();
});

afterEach(() => {
  // 清理 Modal/portal 残留
  document.body.innerHTML = "";
});

describe("components/shared/PageTable（T1 最小骨架）", () => {
  it("传 request+columns+rowKey 渲染表格；total=128 → 共 128 条 / 13 页", async () => {
    const request = vi.fn(async () => ({ list: makeRows(10), total: 128 }));
    const { container } = render(<PageTable request={request} columns={columns} rowKey="id" />, {
      wrapper: Wrap,
    });
    await waitFor(() => {
      expect(screen.getByText("行1")).toBeTruthy();
    });
    expect(container.textContent).toContain("共 128 条 / 13 页");
    // 请求参数带分页
    expect(request.mock.calls[0][0]).toMatchObject({ current: 1, pageSize: 10 });
  });

  it("total=0 → 不渲染分页总文案且显示 antd 空态（.ant-empty，locale 无关）", async () => {
    const request = vi.fn(async () => ({ list: [], total: 0 }));
    const { container } = render(<PageTable request={request} columns={columns} rowKey="id" />, {
      wrapper: Wrap,
    });
    await waitFor(() => {
      expect(container.querySelector(".ant-empty")).not.toBeNull();
    });
    expect(container.textContent).not.toContain("共 0 条");
    expect(container.textContent).not.toContain("共 0 条 /");
  });

  it("request reject → handleApiError 被调 + ErrorAlertWithRetry 可见；重试重新发起请求", async () => {
    const request = vi
      .fn<() => Promise<{ list: { id: string; name: string }[]; total: number }>>()
      .mockRejectedValueOnce(new Error("网络炸了"))
      .mockResolvedValueOnce({ list: makeRows(2), total: 2 });
    const { container } = render(<PageTable request={request} columns={columns} rowKey="id" />, {
      wrapper: Wrap,
    });
    await waitFor(() => {
      expect(handleApiError).toHaveBeenCalled();
      expect((handleApiError as ReturnType<typeof vi.fn>).mock.calls[0][1]).toBe("加载列表数据");
    });
    // 错误态可见（ErrorAlertWithRetry 默认文案含 查询失败 + 重新加载按钮）
    expect(container.textContent).toContain("查询失败");
    expect(screen.getByRole("button", { name: /重新加载/ })).toBeTruthy();
    // 点击重试 → 重新 loadData（第 2 次调用成功）
    fireEvent.click(screen.getByRole("button", { name: /重新加载/ }));
    await waitFor(() => {
      expect(screen.getByText("行1")).toBeTruthy();
    });
    expect(request).toHaveBeenCalledTimes(2);
    // 成功后错误态消失
    expect(container.textContent).not.toContain("查询失败");
  });

  it("request pending → Table 蒙层 loading；无 Skeleton", async () => {
    let resolveFn: (v: { list: { id: string; name: string }[]; total: number }) => void;
    const request = vi.fn(
      () =>
        new Promise<{ list: { id: string; name: string }[]; total: number }>((resolve) => {
          resolveFn = resolve;
        })
    );
    const { container } = render(<PageTable request={request} columns={columns} rowKey="id" />, {
      wrapper: Wrap,
    });
    await waitFor(() => {
      expect(container.querySelector(".ant-spin")).not.toBeNull();
    });
    expect(container.querySelector(".ant-skeleton")).toBeNull();
    resolveFn!({ list: [], total: 0 });
  });

  it("rest props 透传 Table（scroll → .ant-table-body 存在）", async () => {
    const request = vi.fn(async () => ({ list: makeRows(3), total: 3 }));
    const { container } = render(
      <PageTable request={request} columns={columns} rowKey="id" scroll={{ x: 800, y: 200 }} />,
      { wrapper: Wrap }
    );
    await waitFor(() => {
      expect(screen.getByText("行1")).toBeTruthy();
    });
    expect(container.querySelector(".ant-table-body")).not.toBeNull();
  });

  it("自定义 pageSize 生效（showTotal 分母随 pageSize）", async () => {
    const request = vi.fn(async () => ({ list: makeRows(5), total: 45 }));
    const { container } = render(
      <PageTable request={request} columns={columns} rowKey="id" pageSize={15} />,
      { wrapper: Wrap }
    );
    await waitFor(() => {
      expect(screen.getByText("行1")).toBeTruthy();
    });
    expect(container.textContent).toContain("共 45 条 / 3 页");
  });
});

describe("components/shared/PageTable（T2 完整契约）", () => {
  const columns2: typeof columns = [...columns];

  it("toolbar 渲染在工具栏左区；内建刷新按钮有 aria-label+title 且触发 loadData", async () => {
    const request = vi
      .fn<() => Promise<{ list: { id: string; name: string }[]; total: number }>>()
      .mockResolvedValueOnce({ list: makeRows(2), total: 2 })
      .mockResolvedValueOnce({ list: makeRows(3), total: 3 });
    render(
      <PageTable
        request={request}
        columns={columns2}
        rowKey="id"
        toolbar={<button>新增</button>}
      />,
      { wrapper: Wrap }
    );
    await waitFor(() => expect(screen.getByText("行1")).toBeTruthy());
    expect(screen.getByText("新增")).toBeTruthy();
    const refresh = screen.getByRole("button", { name: "刷新" });
    expect(refresh.getAttribute("title")).toBe("刷新");
    fireEvent.click(refresh);
    await waitFor(() => {
      expect(request).toHaveBeenCalledTimes(2);
      expect(screen.getByText("行3")).toBeTruthy();
    });
  });

  it("rowActions 经 ActionButtons 渲染操作列；删除走 ConfirmDelete（vi.spyOn Modal.confirm 断言含 target）", async () => {
    const confirmSpy = vi.spyOn(Modal, "confirm");
    const handleDelete = vi.fn(async () => {});
    const request = vi.fn(async () => ({ list: makeRows(1), total: 1 }));
    render(
      <PageTable
        request={request}
        columns={columns2}
        rowKey="id"
        rowActions={(r) => [
          { key: "edit", label: "编辑", onClick: () => {} },
          // 68.1-T1 契约：对象名走 target（label 已 @deprecated，按钮文字固定「删除」，aria-label 为 删除${target}）
          createDeleteAction(r.id, handleDelete, { target: `删${r.name}` }),
        ]}
      />,
      { wrapper: Wrap }
    );
    await waitFor(() => expect(screen.getAllByText("行1").length).toBeGreaterThan(0));
    expect(screen.getByText("操作")).toBeTruthy();
    fireEvent.click(screen.getByLabelText("删除删行1"));
    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalled();
      const opts = confirmSpy.mock.calls[0][0] as { title?: string; content?: string };
      expect(`${opts.title ?? ""}${opts.content ?? ""}`).toContain("删行1");
    });
    confirmSpy.mockRestore();
  });

  it("batchDelete：选中 >0 出现 BatchDeleteButton；确认后 onDelete + resetSelection + 重新 loadData", async () => {
    const confirmSpy = vi.spyOn(Modal, "confirm");
    const onDelete = vi.fn(async () => {});
    const request = vi.fn(async () => ({ list: makeRows(3), total: 3 }));
    render(
      <PageTable
        request={request}
        columns={columns2}
        rowKey="id"
        batchDelete={{ onDelete, target: "行" }}
      />,
      { wrapper: Wrap }
    );
    await waitFor(() => expect(screen.getByText("行1")).toBeTruthy());
    // 选中 2 行
    const checkboxes = document.body.querySelectorAll<HTMLInputElement>(
      ".ant-table-thead .ant-checkbox-input, .ant-table-tbody .ant-checkbox-input"
    );
    fireEvent.click(checkboxes[1]);
    fireEvent.click(checkboxes[2]);
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /批量删除 \(2\)/ })).toBeTruthy();
    });
    // 确认批量删除 → confirmDelete 模板（标题含 确认批量删除）
    fireEvent.click(screen.getByRole("button", { name: /批量删除 \(2\)/ }));
    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalled();
      const opts = confirmSpy.mock.calls[0][0] as { title?: string; content?: string };
      expect(`${opts.title ?? ""}${opts.content ?? ""}`).toContain("选中的 2 项");
    });
    // onOk（第 4 个参数）调用 → onDelete 成功 → 选中清空 + 重载
    const okFn = (confirmSpy.mock.calls[0][0] as { onOk?: () => Promise<void> }).onOk!;
    await okFn();
    await waitFor(() => {
      expect(onDelete).toHaveBeenCalledWith(["1", "2"]);
      expect(request).toHaveBeenCalledTimes(2);
      // 选中清空 → BatchDeleteButton(selectedCount=0) 返回 null（按钮文本含计数括号）
      expect(screen.queryAllByText(/批量删除 \(\d+\)/)).toHaveLength(0);
    });
    confirmSpy.mockRestore();
  });

  it("batchDelete.target 透传（68.1 minor #2）：确认正文含「选中的 N 项资产」", async () => {
    const confirmSpy = vi
      .spyOn(Modal, "confirm")
      .mockReturnValue({ destroy: vi.fn(), update: vi.fn() } as never);
    const onDelete = vi.fn(async () => {});
    const request = vi.fn(async () => ({ list: makeRows(3), total: 3 }));
    render(
      <PageTable
        request={request}
        columns={columns2}
        rowKey="id"
        batchDelete={{ onDelete, target: "资产" }}
      />,
      { wrapper: Wrap }
    );
    await waitFor(() => expect(screen.getByText("行1")).toBeTruthy());
    const checkboxes = document.body.querySelectorAll<HTMLInputElement>(
      ".ant-table-thead .ant-checkbox-input, .ant-table-tbody .ant-checkbox-input"
    );
    fireEvent.click(checkboxes[1]);
    fireEvent.click(screen.getByRole("button", { name: /批量删除 \(1\)/ }));
    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalled();
      const opts = confirmSpy.mock.calls[0][0] as { content?: string };
      expect(opts.content).toContain("选中的 1 项资产");
    });
    confirmSpy.mockRestore();
  });

  it("sorterMetas+defaultSort：defaultSort 进首次请求参数；表头点击触发带 orderByColumn/isAsc 的请求", async () => {
    const request = vi
      .fn<
        (
          p: Record<string, unknown>
        ) => Promise<{ list: { id: string; name: string }[]; total: number }>
      >()
      .mockResolvedValue({ list: makeRows(3), total: 3 });
    const sorterMetas = [createSorterMeta<{ id: string; name: string }>("name")];
    render(
      <PageTable
        request={request}
        columns={[{ title: "名称", dataIndex: "name", sorter: true }]}
        rowKey="id"
        sorterMetas={sorterMetas}
        defaultSort={{ orderByColumn: "name", isAsc: true }}
      />,
      { wrapper: Wrap }
    );
    await waitFor(() => expect(screen.getByText("行1")).toBeTruthy());
    expect(request.mock.calls[0][0]).toMatchObject({ orderByColumn: "name", isAsc: true });
    // 表头点击排序
    fireEvent.click(screen.getByText("名称"));
    await waitFor(() => {
      expect(request).toHaveBeenCalledTimes(2);
      expect(request.mock.calls[1][0]).toMatchObject({ orderByColumn: "name" });
    });
  });

  it("extraFilters 引用变化 → 回第 1 页重载且并入请求参数；不变则不触发", async () => {
    const request = vi
      .fn<
        (
          p: Record<string, unknown>
        ) => Promise<{ list: { id: string; name: string }[]; total: number }>
      >()
      .mockResolvedValue({ list: makeRows(2), total: 2 });
    const renderNode = (extraFilters: Record<string, unknown> | undefined) => (
      <PageTable request={request} columns={columns2} rowKey="id" extraFilters={extraFilters} />
    );
    const sameRef = { deptId: "d1" };
    const { rerender } = render(renderNode(sameRef), { wrapper: Wrap });
    await waitFor(() => expect(screen.getAllByText("行1").length).toBeGreaterThan(0));
    expect(request).toHaveBeenCalledTimes(1);
    expect(request.mock.calls[0][0]).toMatchObject({ deptId: "d1", current: 1 });
    // 同引用重渲染：不触发新请求
    rerender(renderNode(sameRef));
    expect(request).toHaveBeenCalledTimes(1);
    // 引用变化：回第 1 页重载
    rerender(renderNode({ deptId: "d2" }));
    await waitFor(() => {
      expect(request).toHaveBeenCalledTimes(2);
      expect(request.mock.calls[1][0]).toMatchObject({ deptId: "d2", current: 1 });
    });
  });

  // UAT FAIL-1 回归（Phase 75）：tab 切换 remount 场景——mount 首载必须保留
  // sessionStorage 恢复的页码（keepPage），不得硬编码回第 1 页并覆写 storage。
  // 原缺陷：PageTable mount effect 无条件走 applyFilters（回第 1 页），jsdom 旧用例
  // 未覆盖「预置持久化页码 + 挂载」组合，故实机失败而单测通过。
  it("UAT FAIL-1：sessionStorage 有持久化页码时 mount 首载恢复该页码且不覆写 storage", async () => {
    sessionStorage.setItem("xingran_table_state_test_current", "2");
    const request = vi
      .fn<
        (
          p: Record<string, unknown>
        ) => Promise<{ list: { id: string; name: string }[]; total: number }>
      >()
      .mockResolvedValue({ list: makeRows(10), total: 25 });
    const { container, unmount } = render(
      <PageTable request={request} columns={columns} rowKey="id" />,
      { wrapper: Wrap }
    );
    await waitFor(() => expect(screen.getAllByText("行1").length).toBeGreaterThan(0));
    // 首载请求用恢复的页码 2
    expect(request.mock.calls[0][0]).toMatchObject({ current: 2, pageSize: 10 });
    // sessionStorage 未被 mount 覆写
    expect(sessionStorage.getItem("xingran_table_state_test_current")).toBe("2");
    // 分页 UI 停在第 2 页
    expect(container.querySelector(".ant-pagination-item-active")?.textContent).toBe("2");

    // 模拟「切 tab → 切回」：卸载 → 重挂载，页码仍恢复为 2
    unmount();
    const remount = render(<PageTable request={request} columns={columns} rowKey="id" />, {
      wrapper: Wrap,
    });
    await waitFor(() => expect(screen.getAllByText("行1").length).toBeGreaterThan(0));
    expect(remount.container.querySelector(".ant-pagination-item-active")?.textContent).toBe("2");
    expect(sessionStorage.getItem("xingran_table_state_test_current")).toBe("2");
    sessionStorage.removeItem("xingran_table_state_test_current");
  });

  it("emptyAction：total=0 时替换 antd 默认空态", async () => {
    const request = vi.fn(async () => ({ list: [], total: 0 }));
    const { container } = render(
      <PageTable
        request={request}
        columns={columns2}
        rowKey="id"
        emptyAction={<div data-testid="guide">去新建第一条</div>}
      />,
      { wrapper: Wrap }
    );
    await waitFor(() => {
      expect(screen.getByTestId("guide")).toBeTruthy();
    });
    expect(container.querySelector(".ant-empty")).toBeNull();
  });

  it("searchFields schema 渲染搜索卡；搜索/重置走 handleSearch/handleReset 语义", async () => {
    const request = vi
      .fn<
        (
          p: Record<string, unknown>
        ) => Promise<{ list: { id: string; name: string }[]; total: number }>
      >()
      .mockResolvedValue({ list: makeRows(2), total: 2 });
    const { container } = render(
      <PageTable
        request={request}
        columns={columns2}
        rowKey="id"
        searchFields={[
          { type: "input", name: "kw", label: "关键字" },
          { type: "select", name: "status", label: "状态", options: [{ label: "正常", value: 0 }] },
        ]}
      />,
      { wrapper: Wrap }
    );
    await waitFor(() => expect(screen.getAllByText("行1").length).toBeGreaterThan(0));
    expect(container.querySelector("input#kw")).not.toBeNull();
    // 输入 + 搜索 → 请求参数含字段值
    const input = container.querySelector<HTMLInputElement>("input#kw")!;
    fireEvent.input(input, { target: { value: "abc" } });
    fireEvent.click(screen.getByText((c) => c.replace(/\s/g, "") === "搜索"));
    await waitFor(() => {
      expect(request).toHaveBeenCalledTimes(2);
      expect(request.mock.calls[1][0]).toMatchObject({ kw: "abc", current: 1 });
    });
    // 重置 → 清 filters 回第 1 页
    fireEvent.click(screen.getByText((c) => c.replace(/\s/g, "") === "重置"));
    await waitFor(() => {
      expect(request).toHaveBeenCalledTimes(3);
      expect(request.mock.calls[2][0]).toMatchObject({ current: 1 });
      // filterEmpty 语义：清空后的字段不进请求参数
      expect("kw" in request.mock.calls[2][0]).toBe(false);
    });
  });

  it("searchChildren 与 searchFields 同时传 → schema 优先 + console.warn 一次", async () => {
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    const request = vi.fn(async () => ({ list: makeRows(1), total: 1 }));
    const node = (
      <PageTable
        request={request}
        columns={columns2}
        rowKey="id"
        searchFields={[{ type: "input", name: "kw", label: "关键字" }]}
        searchChildren={<div data-testid="child-search">复杂搜索</div>}
      />
    );
    const { rerender } = render(node, { wrapper: Wrap });
    await waitFor(() => expect(screen.getAllByText("行1").length).toBeGreaterThan(0));
    expect(screen.queryByTestId("child-search")).toBeNull();
    expect(document.querySelector("input#kw")).not.toBeNull();
    expect(warnSpy).toHaveBeenCalledTimes(1);
    // 同 props 重渲染：不重复警告
    rerender(node);
    expect(warnSpy).toHaveBeenCalledTimes(1);
    warnSpy.mockRestore();
  });

  it("searchChildren 函数形态收到 searchForm，其 Form.Item 值进入请求参数", async () => {
    const request = vi
      .fn<
        (
          p: Record<string, unknown>
        ) => Promise<{ list: { id: string; name: string }[]; total: number }>
      >()
      .mockResolvedValue({ list: makeRows(1), total: 1 });
    const { container } = render(
      <PageTable
        request={request}
        columns={columns2}
        rowKey="id"
        searchChildren={(_form) => (
          <Form.Item name="deptId" label="部门">
            <input data-testid="dept-input" />
          </Form.Item>
        )}
      />,
      { wrapper: Wrap }
    );
    await waitFor(() => expect(screen.getAllByText("行1").length).toBeGreaterThan(0));
    fireEvent.change(screen.getByTestId("dept-input"), { target: { value: "d-1" } });
    fireEvent.click(screen.getByText((c) => c.replace(/\s/g, "") === "搜索"));
    await waitFor(() => {
      expect(request).toHaveBeenCalledTimes(2);
      expect(request.mock.calls[1][0]).toMatchObject({ deptId: "d-1" });
    });
    void container;
  });
});
