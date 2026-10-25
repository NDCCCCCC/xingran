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
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

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
