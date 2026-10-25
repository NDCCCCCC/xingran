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
import { App as AntdApp, Modal } from "antd";
import type { ReactElement, ReactNode } from "react";

import { createSorterMeta } from "@/utils/tableHelpers";
import { createDeleteAction } from "../ActionButtons";

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
  handleSuccess: vi.fn(),
  isFormValidationError: (e: unknown) =>
    Boolean(e && typeof e === "object" && "errorFields" in e),
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
});

afterEach(() => {
  // 清理 Modal/portal 残留
  document.body.innerHTML = "";
});

describe("components/shared/PageTable（T1 最小骨架）", () => {
  it("传 request+columns+rowKey 渲染表格；total=128 → 共 128 条 / 13 页", async () => {
    const request = vi.fn(async () => ({ list: makeRows(10), total: 128 }));
    const { container } = render(
      <PageTable request={request} columns={columns} rowKey="id" />,
      { wrapper: Wrap }
    );
    await waitFor(() => {
      expect(screen.getByText("行1")).toBeTruthy();
    });
    expect(container.textContent).toContain("共 128 条 / 13 页");
    // 请求参数带分页
    expect(request.mock.calls[0][0]).toMatchObject({ current: 1, pageSize: 10 });
  });

  it("total=0 → 不渲染分页总文案且显示 antd 空态（.ant-empty，locale 无关）", async () => {
    const request = vi.fn(async () => ({ list: [], total: 0 }));
    const { container } = render(
      <PageTable request={request} columns={columns} rowKey="id" />,
      { wrapper: Wrap }
    );
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
    const { container } = render(
      <PageTable request={request} columns={columns} rowKey="id" />,
      { wrapper: Wrap }
    );
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
    const { container } = render(
      <PageTable request={request} columns={columns} rowKey="id" />,
      { wrapper: Wrap }
    );
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
      <PageTable request={request} columns={columns2} rowKey="id" toolbar={<button>新增</button>} />,
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
          createDeleteAction(r.id, handleDelete, { label: `删${r.name}` }),
        ]}
      />,
      { wrapper: Wrap }
    );
    await waitFor(() => expect(screen.getAllByText("行1").length).toBeGreaterThan(0));
    expect(screen.getByText("操作")).toBeTruthy();
    fireEvent.click(screen.getByText("删行1"));
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

  it("sorterMetas+defaultSort：defaultSort 进首次请求参数；表头点击触发带 orderByColumn/isAsc 的请求", async () => {
    const request = vi
      .fn<(p: Record<string, unknown>) => Promise<{ list: { id: string; name: string }[]; total: number }>>()
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
      .fn<(p: Record<string, unknown>) => Promise<{ list: { id: string; name: string }[]; total: number }>>()
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
});
